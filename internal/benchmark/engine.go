package benchmark

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"go.uber.org/ratelimit"
)

const (
	updateInterval = 1 * time.Second
)

type workResult struct {
	WorkIndex int
	Elapsed   time.Duration
	Error     error
}

type BenchmarkResult struct {
	Total      uint64
	Succeeded  uint64
	Failed     uint64
	MaxLatency time.Duration
	MinLatency time.Duration
	AvgLatency time.Duration
	ExecPerSec float64
	StartTime  time.Time
	TimeTaken  time.Duration
	mtx        sync.Mutex
}

func (r *BenchmarkResult) collectResult(work *workResult) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.TimeTaken += work.Elapsed
	if work.Error != nil {
		r.Failed += 1
	} else {
		r.Succeeded += 1
	}
	execCount := r.Succeeded + r.Failed
	r.ExecPerSec = float64(execCount) / float64(time.Since(r.StartTime)/time.Second)
	r.AvgLatency = time.Duration(uint64(r.TimeTaken) / execCount)
	if work.Elapsed > r.MaxLatency {
		r.MaxLatency = work.Elapsed
	}
	if work.Elapsed < r.MinLatency {
		r.MinLatency = work.Elapsed
	}
}

type BenchmarkOptions struct {
	ExecuteRate int
	NumWorkers  int
	Duration    time.Duration
	Timeout     time.Duration
}

type BenchmarkWorker interface {
	DoWork(workerIndex int) error
}

type BenchmarkTest interface {
	Prepair()
	CreateWorker(workerIdx int) (BenchmarkWorker, error)
	OnFinish(result *BenchmarkResult)
}

type BenchmarkEngine struct {
	BenchmarkOptions
	limiter   ratelimit.Limiter
	testToRun BenchmarkTest
	workers   []BenchmarkWorker
	workCh    chan int
	result    *BenchmarkResult
	submitted uint64
}

func (e *BenchmarkEngine) SetBenchmarkTest(test BenchmarkTest) {
	e.testToRun = test
}

func (e *BenchmarkEngine) printStatus() {
	for {
		time.Sleep(1 * time.Second)
		if e.result == nil {
			continue
		}
		fmt.Println("Submmited:", e.submitted)
		fmt.Println("Succeeded:", e.result.Succeeded)
		fmt.Println("Failed:", e.result.Failed)
		fmt.Printf("MinLatency: %dms\n", e.result.MinLatency/time.Millisecond)
		fmt.Printf("AvgLatency: %dms\n", e.result.AvgLatency/time.Millisecond)
		fmt.Printf("MaxLatency: %dms\n", e.result.MaxLatency/time.Millisecond)
		fmt.Printf("ExecPerSec: %.2f\n", e.result.ExecPerSec)
		fmt.Printf("SubmitedPerSec: %.2f\n", float64(e.submitted)/float64(time.Since(e.result.StartTime)/time.Second))
		fmt.Println("Pending:", len(e.workCh))
		fmt.Println()
	}
}

func (e *BenchmarkEngine) doWork(worker BenchmarkWorker, workIdx int) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
	defer cancel()
	errCh := make(chan error, 1)
	select {
	case errCh <- worker.DoWork(workIdx):
		return <-errCh
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *BenchmarkEngine) consumeWork(workerIdx int, workCh <-chan int) {
	worker := e.workers[workerIdx]
	for workIdx := range workCh {
		startTime := time.Now()
		err := e.doWork(worker, workIdx)
		go e.result.collectResult(&workResult{
			WorkIndex: workIdx,
			Elapsed:   time.Since(startTime),
			Error:     err,
		})
	}
}

func (e *BenchmarkEngine) produceWork(ctx context.Context, workCh chan<- int) {
	defer close(workCh)
	ctx, cancel := context.WithTimeout(ctx, e.Duration)
	defer cancel()
	for workIdx := 0; true; workIdx++ {
		e.limiter.Take()
		select {
		case workCh <- workIdx:
		case <-ctx.Done():
			return
		}
	}
}

func (e *BenchmarkEngine) prepairWorkers() []BenchmarkWorker {
	wg := &sync.WaitGroup{}
	workers := make([]BenchmarkWorker, e.NumWorkers)
	for workerIdx := 0; workerIdx < len(e.workers); workerIdx++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			worker, err := e.testToRun.CreateWorker(idx)
			if err != nil {
				log.Fatal("could not create worker ", idx, err)
			}
			workers[idx] = worker
		}(workerIdx)
	}
	wg.Wait()
	return workers
}

func (e *BenchmarkEngine) Run(ctx context.Context) {
	e.testToRun.Prepair()
	e.workers = e.prepairWorkers()
	e.result = &BenchmarkResult{
		StartTime:  time.Now(),
		MinLatency: time.Duration(math.MaxInt64),
	}
	go e.printStatus()
	workCh := make(chan int, len(e.workers))
	e.workCh = workCh
	for workerIdx := 0; workerIdx < e.NumWorkers; workerIdx++ {
		go e.consumeWork(workerIdx, workCh)
	}
	e.produceWork(ctx, workCh)
	e.testToRun.OnFinish(e.result)
}

func NewBenchmarkEngine(opts BenchmarkOptions) *BenchmarkEngine {
	return &BenchmarkEngine{
		BenchmarkOptions: opts,
		limiter:          ratelimit.New(opts.ExecuteRate, ratelimit.WithSlack(100)),
	}
}
