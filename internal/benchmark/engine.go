package benchmark

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/ratelimit"
)

const (
	updateInterval = 1 * time.Second
)

var errWorkerTimeout = errors.New("timed out")

type WorkloadFunc func(workerIndex int) error
type OnRoundFinishedFunc func(roundIndex int, result *BenchmarkResult)

type workerResult struct {
	WorkerIndex int
	Elapsed     time.Duration
	Error       error
}

type BenchmarkResult struct {
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
	DoWork(workerIndex int) error
	OnFinish(roundIndex int, result *BenchmarkResult)
}

type BenchmarkEngine struct {
	BenchmarkOptions
	limiter   ratelimit.Limiter
	ticker    time.Ticker
	records   map[int]*workerResult
	testToRun BenchmarkTest
	result    *BenchmarkResult
	wg        *LimitWaitGroup
}

func (e *BenchmarkEngine) worker(wg *LimitWaitGroup, workerIndex int) {
	defer func() {
		wg.Done()
	}()
	startTime := time.Now()
	errCh := make(chan error, 1)
	go func() {
		errCh <- e.testToRun.DoWork(workerIndex)
	}()
	select {
	case err := <-errCh:
		if err != nil {
			fmt.Printf("Worker %d failed: %s\n", workerIndex, err)
		}
		e.result.collectResult(&workerResult{
			WorkerIndex: workerIndex,
			Elapsed:     time.Since(startTime),
			Error:       err,
		})
	case <-time.After(e.Timeout):
		e.result.collectResult(&workerResult{
			WorkerIndex: workerIndex,
			Elapsed:     time.Since(startTime),
			Error:       errWorkerTimeout,
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

func (e *BenchmarkEngine) runRound(roundIdx int) *BenchmarkResult {
	startTime := time.Now()
	wg := NewLimitWaitGroup(e.MaxThread)
	e.wg = wg
	for i := 0; i < e.NumWorkers; i++ {
		e.limiter.Take()
		wg.Add()
		go e.worker(wg, i)
	}
	wg.Wait()
	return e.generateResult(startTime)
}

func (e *BenchmarkEngine) printStatus() {
	for {
		time.Sleep(1 * time.Second)
		fmt.Println("Succeeded: ", e.result.Succeeded)
		fmt.Println("Failed: ", e.result.Failed)
		fmt.Printf("MinLatency: %dms\n", e.result.MinLatency/time.Millisecond)
		fmt.Printf("MaxLatency: %dms\n", e.result.MaxLatency/time.Millisecond)
		fmt.Printf("ExecPerSec: %.2f\n", e.result.ExecPerSec)
		fmt.Println()
	}
}

func (e *BenchmarkEngine) Run(ctx context.Context) {
	e.result = &BenchmarkResult{
		StartTime:  time.Now(),
		MinLatency: time.Duration(math.MaxInt64),
	}
	e.testToRun.Prepair()
	go e.printStatus()
	for roundIdx := 0; roundIdx < e.NumRounds; roundIdx++ {
		result := e.runRound(roundIdx)
		e.testToRun.OnFinish(roundIdx, result)
	}
}

func NewBenchmarkEngine(opts BenchmarkOptions) *BenchmarkEngine {
	return &BenchmarkEngine{
		BenchmarkOptions: opts,
		records:          make(map[int]*workerResult),
		limiter:          ratelimit.New(opts.ExecuteRate),
		ticker:           *time.NewTicker(updateInterval),
	}
}
