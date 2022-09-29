package benchmark

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
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
		fmt.Println(work.Error)
		r.Failed += 1
	} else {
		r.Succeeded += 1
	}
	execCount := r.Succeeded + r.Failed
	r.TimeTaken = time.Since(r.StartTime)
	r.ExecPerSec = float64(execCount*uint64(time.Second)) / float64(r.TimeTaken)
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

type Options struct {
	RpcUrl      string
	NumWorkers  int
	NumClients  int
	ExecuteRate int
	Duration    time.Duration
	Timeout     time.Duration
}

type BenchmarkWorker interface {
	DoWork(ctx context.Context, workerIndex int) error
}

type BenchmarkTest interface {
	Name() string
	Prepair(opts Options)
	CreateWorker(client *rpc.Client, workerIdx int) BenchmarkWorker
	OnFinish(result *BenchmarkResult)
}

type BenchmarkEngine struct {
	Options
	limiter   ratelimit.Limiter
	testToRun BenchmarkTest
	workers   []BenchmarkWorker
	clients   []*rpc.Client
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

func (e *BenchmarkEngine) consumeWork(wg *LimitWaitGroup, workerIdx int, workCh <-chan int) {
	defer wg.Done()
	client := e.clients[workerIdx%len(e.clients)]
	worker := e.testToRun.CreateWorker(client, workerIdx)
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

func (e *BenchmarkEngine) produceWork(workCh chan<- int) {
	defer close(workCh)
	deadline := time.Now().Add(e.Duration)
	for workIdx := 0; true; workIdx++ {
		e.limiter.Take()
		workCh <- workIdx
		atomic.AddUint64(&e.result.Total, 1)
		e.result.SubmitPerSec = float64(e.result.Total*uint64(time.Second)) / float64(time.Since(e.result.StartTime))
		if time.Now().After(deadline) {
			return
		}
	}
}

func (e *BenchmarkEngine) prepairClients() error {
	clients := make([]*rpc.Client, e.NumClients)
	for idx := 0; idx < e.NumClients; idx++ {
		log.Println("Dialing RPC node", e.RpcUrl)
		client, err := rpc.Dial(e.RpcUrl)
		if err != nil {
			return err
		}
		clients[idx] = client
	}
	e.clients = clients
	return nil
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
			fmt.Sprintf("%.2f", ret.SubmitPerSec),
			fmt.Sprintf("%.2f", ret.ExecPerSec),
			ret.TimeTaken,
		},
	})
	tw.Render()
}

func (e *BenchmarkEngine) Run(ctx context.Context) {
	log.Println("Preparing connections.")
	if err := e.prepairClients(); err != nil {
		log.Fatal(err)
	}

	log.Println("Preparing testcase.")
	e.testToRun.Prepair(e.Options)

	log.Printf("Running testcase with %d workres.", e.NumWorkers)
	wg := NewLimitWaitGroup(e.NumWorkers)
	workCh := make(chan int, 10*e.ExecuteRate)
	for workerIdx := 0; workerIdx < e.NumWorkers; workerIdx++ {
		wg.Add()
		go e.consumeWork(wg, workerIdx, workCh)
	}
	e.result = newBenchmarkResult()
	go printStatus(e.result, workCh)
	e.produceWork(workCh)

	log.Println("Waiting for all workers to finish.")
	wg.Wait()
	e.testToRun.OnFinish(e.result)
	e.printResult()
}

func NewBenchmarkEngine(opts Options) *BenchmarkEngine {
	limiter := ratelimit.New(opts.ExecuteRate, ratelimit.WithSlack(opts.ExecuteRate*10/100))
	return &BenchmarkEngine{
		Options: opts,
		limiter: limiter,
	}
}
