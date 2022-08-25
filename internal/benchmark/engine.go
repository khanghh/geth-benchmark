package benchmark

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/ratelimit"
)

const (
	updateInterval = 1 * time.Second
)

type WorkloadFunc func(workerIndex int) error
type OnRoundFinishedFunc func(roundIndex int, result *BenchmarkResult)

type workerResult struct {
	WorkerIndex int
	Elapsed     time.Duration
	Error       error
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

func (r *BenchmarkResult) collectResult(work *workerResult) {
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
	MaxThread   int
	ExecuteRate int
	NumWorkers  int
	NumRounds   int
	Timeout     time.Duration
}

type BenchmarkTest interface {
	Prepair()
	DoWork(ctx context.Context, workerIndex int) error
	OnFinish(roundIndex int, result *BenchmarkResult)
}

type BenchmarkEngine struct {
	BenchmarkOptions
	limiter   ratelimit.Limiter
	ticker    time.Ticker
	records   map[int]*workerResult
	testToRun BenchmarkTest
	result    *BenchmarkResult
	submitted uint64
	wg        *LimitWaitGroup
}

func (e *BenchmarkEngine) worker(wg *LimitWaitGroup, workerIndex int) {
	defer func() {
		wg.Done()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
	defer cancel()
	startTime := time.Now()
	errCh := make(chan error, 1)
	select {
	case errCh <- e.testToRun.DoWork(ctx, workerIndex):
		err := <-errCh
		if err != nil {
			fmt.Printf("Worker %d failed: %s\n", workerIndex, err)
		}
		e.result.collectResult(&workerResult{
			WorkerIndex: workerIndex,
			Elapsed:     time.Since(startTime),
			Error:       err,
		})
	case <-ctx.Done():
		fmt.Printf("Worker %d failed: %s\n", workerIndex, ctx.Err())
		e.result.collectResult(&workerResult{
			WorkerIndex: workerIndex,
			Elapsed:     time.Since(startTime),
			Error:       ctx.Err(),
		})
	}
}

func (e *BenchmarkEngine) generateResult(startTime time.Time) *BenchmarkResult {
	return &BenchmarkResult{
		StartTime: startTime,
	}
}

func (e *BenchmarkEngine) SetBenchmark(test BenchmarkTest) {
	e.testToRun = test
}

func (e *BenchmarkEngine) runRound(roundIdx int) {
	for i := 0; i < e.NumWorkers; i++ {

	}
}

func (e *BenchmarkEngine) printStatus() {
	for {
		time.Sleep(1 * time.Second)
		fmt.Println("Submmited:", e.submitted)
		fmt.Println("Succeeded:", e.result.Succeeded)
		fmt.Println("Failed:", e.result.Failed)
		fmt.Printf("MinLatency: %dms\n", e.result.MinLatency/time.Millisecond)
		fmt.Printf("AvgLatency: %dms\n", e.result.AvgLatency/time.Millisecond)
		fmt.Printf("MaxLatency: %dms\n", e.result.MaxLatency/time.Millisecond)
		fmt.Printf("ExecPerSec: %.2f\n", e.result.ExecPerSec)
		fmt.Printf("SubmitedPerSec: %.2f\n", float64(e.submitted)/float64(time.Since(e.result.StartTime)/time.Second))
		fmt.Println("Working:", e.wg.Size())
		fmt.Println()
	}
}

func (e *BenchmarkEngine) Run(ctx context.Context) {
	e.testToRun.Prepair()
	e.result = &BenchmarkResult{
		StartTime:  time.Now(),
		MinLatency: time.Duration(math.MaxInt64),
	}
	go e.printStatus()
	e.wg = NewLimitWaitGroup(e.MaxThread)
	for roundIdx := 0; roundIdx < e.NumRounds; roundIdx++ {
		for idx := 0; idx < e.NumWorkers; idx++ {
			e.limiter.Take()
			e.wg.Add()
			atomic.AddUint64(&e.submitted, 1)
			go e.worker(e.wg, idx)
		}
	}
	e.wg.Wait()
}

func NewBenchmarkEngine(opts BenchmarkOptions) *BenchmarkEngine {
	return &BenchmarkEngine{
		BenchmarkOptions: opts,
		records:          make(map[int]*workerResult),
		limiter:          ratelimit.New(opts.ExecuteRate, ratelimit.WithSlack(100)),
		ticker:           *time.NewTicker(updateInterval),
	}
}
