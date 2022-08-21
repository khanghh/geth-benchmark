package benchmark

import (
	"context"
	"errors"
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
	WorkPerSec float64
	StartTime  time.Time
	FinishTime time.Time
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
	results   []*BenchmarkResult
	records   map[int]*workerResult
	testToRun BenchmarkTest
	mtx       sync.Mutex
}

func (e *BenchmarkEngine) onWorkerFinished(result *workerResult) {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	e.records[result.WorkerIndex] = result
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
		e.onWorkerFinished(&workerResult{
			WorkerIndex: workerIndex,
			Elapsed:     time.Since(startTime),
			Error:       err,
		})
	case <-time.After(e.Timeout):
		e.onWorkerFinished(&workerResult{
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
	for i := 0; i < e.NumWorkers; i++ {
		e.limiter.Take()
		wg.Add()
		go e.worker(wg, i)
	}
	wg.Wait()
	return e.generateResult(startTime)
}

func (e *BenchmarkEngine) Run(ctx context.Context) {
	e.testToRun.Prepair()
	for roundIdx := 0; roundIdx < e.NumRounds; roundIdx++ {
		result := e.runRound(roundIdx)
		e.results = append(e.results, result)
		e.testToRun.OnFinish(roundIdx, result)
	}
}

func NewBenchmarkEngine(opts BenchmarkOptions) *BenchmarkEngine {
	return &BenchmarkEngine{
		BenchmarkOptions: opts,
		limiter:          ratelimit.New(opts.ExecuteRate),
		records:          make(map[int]*workerResult),
		ticker:           *time.NewTicker(updateInterval),
	}
}
