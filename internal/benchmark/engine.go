package benchmark

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jedib0t/go-pretty/table"
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
	Total         uint64
	Succeeded     uint64
	Failed        uint64
	MaxLatency    time.Duration
	MinLatency    time.Duration
	AvgLatency    time.Duration
	ExecPerSec    float64
	SubmitPerSec  float64
	StartTime     time.Time
	TimeTaken     time.Duration
	totalExecTime time.Duration
	mtx           sync.Mutex
}

func (r *BenchmarkResult) collectResult(work *workResult) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.totalExecTime += work.Elapsed
	if work.Error != nil {
		r.Failed += 1
	} else {
		r.Succeeded += 1
	}
	execCount := r.Succeeded + r.Failed
	r.TimeTaken = time.Since(r.StartTime)
	r.ExecPerSec = float64(execCount) / float64(r.TimeTaken/time.Second)
	r.SubmitPerSec = float64(r.Total) / float64(r.TimeTaken/time.Second)
	r.AvgLatency = time.Duration(uint64(r.totalExecTime) / execCount)
	if work.Elapsed > r.MaxLatency {
		r.MaxLatency = work.Elapsed
	}
	if r.MinLatency == 0 || work.Elapsed < r.MinLatency {
		r.MinLatency = work.Elapsed
	}
}

func newBenchmarkResult() *BenchmarkResult {
	return &BenchmarkResult{
		StartTime: time.Now(),
	}
}

type BenchmarkOptions struct {
	ExecuteRate int
	NumWorkers  int
	Duration    time.Duration
	Timeout     time.Duration
}

type BenchmarkWorker interface {
	DoWork(ctx context.Context, workerIndex int) error
}

type BenchmarkTest interface {
	Name() string
	Prepair()
	CreateWorker(workerIdx int) (BenchmarkWorker, error)
	OnFinish(result *BenchmarkResult)
}

type BenchmarkEngine struct {
	BenchmarkOptions
	limiter   ratelimit.Limiter
	testToRun BenchmarkTest
	workers   []BenchmarkWorker
	result    *BenchmarkResult
}

func (e *BenchmarkEngine) SetBenchmarkTest(test BenchmarkTest) {
	e.testToRun = test
}

func (e *BenchmarkEngine) doWork(worker BenchmarkWorker, workIdx int) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
	defer cancel()
	errCh := make(chan error, 1)
	select {
	case errCh <- worker.DoWork(ctx, workIdx):
		return <-errCh
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *BenchmarkEngine) consumeWork(wg *sync.WaitGroup, workerIdx int, workCh <-chan int) {
	defer wg.Done()
	worker := e.workers[workerIdx]
	for workIdx := range workCh {
		startTime := time.Now()
		err := e.doWork(worker, workIdx)
		e.result.collectResult(&workResult{
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
			atomic.AddUint64(&e.result.Total, 1)
		case <-ctx.Done():
			return
		}
	}
}

func (e *BenchmarkEngine) prepairWorkers() {
	wg := &sync.WaitGroup{}
	e.workers = make([]BenchmarkWorker, e.NumWorkers)
	for workerIdx := 0; workerIdx < len(e.workers); workerIdx++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			worker, err := e.testToRun.CreateWorker(idx)
			if err != nil {
				log.Fatal("could not create worker ", idx, err)
			}
			e.workers[idx] = worker
		}(workerIdx)
		time.Sleep(100 * time.Microsecond)
	}
	wg.Wait()
}

func printStatus(result *BenchmarkResult, workCh chan int) {
	for {
		time.Sleep(1 * time.Second)
		timeTaken := time.Since(result.StartTime)
		fmt.Println("Total:", result.Total)
		fmt.Println("Succeeded:", result.Succeeded)
		fmt.Println("Failed:", result.Failed)
		fmt.Println("MinLatency:", result.MinLatency)
		fmt.Println("AvgLatency:", result.AvgLatency)
		fmt.Println("MaxLatency:", result.MaxLatency)
		fmt.Printf("ExecPerSec: %.2f\n", result.ExecPerSec)
		fmt.Printf("SubmitedPerSec: %.2f\n", result.SubmitPerSec)
		fmt.Println("TimeTaken: ", timeTaken)
		fmt.Println("Pending:", len(workCh))
		fmt.Println()
	}
}

func (e *BenchmarkEngine) printResult() {
	ret := e.result
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.AppendHeader(table.Row{
		"TestCase",
		"Total",
		"Succeeded",
		"Failed",
		"MinLatency",
		"MaxLatency",
		"AvgLatency",
		"SubmitPerSec",
		"ExecPerSec",
		"TimeTaken",
	})
	tw.AppendRows([]table.Row{
		{
			e.testToRun.Name(),
			ret.Total,
			ret.Succeeded,
			ret.Failed,
			ret.MinLatency,
			ret.MaxLatency,
			ret.AvgLatency,
			ret.SubmitPerSec,
			ret.ExecPerSec,
			ret.TimeTaken,
		},
	})
	tw.Render()
}

func (e *BenchmarkEngine) Run(ctx context.Context) {
	fmt.Println("Preparing testcase...")
	e.testToRun.Prepair()
	e.prepairWorkers()

	e.result = newBenchmarkResult()
	wg := &sync.WaitGroup{}
	workCh := make(chan int, e.NumWorkers)
	for workerIdx := 0; workerIdx < e.NumWorkers; workerIdx++ {
		wg.Add(1)
		go e.consumeWork(wg, workerIdx, workCh)
	}
	go printStatus(e.result, workCh)
	e.produceWork(ctx, workCh)

	fmt.Println("Waiting for all workers to finish...")
	wg.Wait()
	e.result.TimeTaken = time.Since(e.result.StartTime)
	e.testToRun.OnFinish(e.result)
	e.printResult()
}

func NewBenchmarkEngine(opts BenchmarkOptions) *BenchmarkEngine {
	limiter := ratelimit.New(opts.ExecuteRate, ratelimit.WithSlack(opts.ExecuteRate*10/100))
	return &BenchmarkEngine{
		BenchmarkOptions: opts,
		limiter:          limiter,
	}
}
