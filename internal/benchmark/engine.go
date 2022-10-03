package benchmark

import (
	"context"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"go.uber.org/ratelimit"
)

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
	resultCollector
	limiter   ratelimit.Limiter
	testToRun BenchmarkTest
	clients   []*rpc.Client
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
		go e.onWorkFinish(&workResult{
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
		e.onWorkStart(workIdx)
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

func (e *BenchmarkEngine) SetReporter(reporter *InfluxDBReporter) {
	e.reporter = reporter
}

func (e *BenchmarkEngine) Run(ctx context.Context, testToRun BenchmarkTest) *BenchmarkResult {
	log.Println("Preparing connections.")
	if err := e.prepairClients(); err != nil {
		log.Fatal(err)
	}

	log.Println("Preparing testcase.")
	e.testToRun = testToRun
	e.testToRun.Prepair(e.Options)

	log.Printf("Running testcase with %d workres.", e.NumWorkers)
	wg := NewLimitWaitGroup(e.NumWorkers)
	workCh := make(chan int, 10*e.ExecuteRate)
	for workerIdx := 0; workerIdx < e.NumWorkers; workerIdx++ {
		wg.Add()
		go e.consumeWork(wg, workerIdx, workCh)
	}
	e.initBenchmarkResult(testToRun.Name())
	go e.monitorLoop(ctx)
	e.produceWork(workCh)

	log.Println("Waiting for all workers to finish.")
	wg.Wait()
	e.testToRun.OnFinish(e.result)
	return e.result
}

func NewBenchmarkEngine(opts Options) *BenchmarkEngine {
	limiter := ratelimit.New(opts.ExecuteRate, ratelimit.WithSlack(opts.ExecuteRate*10/100))
	return &BenchmarkEngine{
		Options: opts,
		limiter: limiter,
	}
}
