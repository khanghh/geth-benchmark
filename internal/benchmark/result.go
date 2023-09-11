package benchmark

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type WorkResult struct {
	WorkIndex int
	Elapsed   time.Duration
	Error     error
}

type BenchmarkResult struct {
	Testcase     string
	Total        uint64
	Succeeded    uint64
	Failed       uint64
	MaxLatency   time.Duration
	MinLatency   time.Duration
	AvgLatency   time.Duration
	ExecPerSec   float64
	SubmitPerSec float64
	StartTime    time.Time
	TimeTaken    time.Duration
}

type resultCollector struct {
	mtx           sync.Mutex
	totalExecTime time.Duration
	result        *BenchmarkResult
	reporter      BenchmarkReporter
}

func (c *resultCollector) initBenchmarkResult(testcase string) {
	c.totalExecTime = 0
	c.result = &BenchmarkResult{
		Testcase:  testcase,
		StartTime: time.Now(),
	}
}

func (c *resultCollector) onWorkStart(workIdx int) {
	atomic.AddUint64(&c.result.Total, 1)
	c.result.SubmitPerSec = float64(c.result.Total*uint64(time.Second)) / float64(time.Since(c.result.StartTime))
}

func (c *resultCollector) onWorkFinish(work *WorkResult) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.totalExecTime += work.Elapsed
	if c.reporter != nil {
		c.reporter.CollectWorkResult(work)
	}
	result := c.result
	if work.Error != nil {
		log.Println(work.Error)
		result.Failed += 1
	} else {
		result.Succeeded += 1
	}
	execCount := result.Succeeded + result.Failed
	result.TimeTaken = time.Since(result.StartTime)
	result.ExecPerSec = float64(execCount*uint64(time.Second)) / float64(result.TimeTaken)
	result.AvgLatency = time.Duration(uint64(c.totalExecTime) / execCount)
	if work.Elapsed > result.MaxLatency {
		result.MaxLatency = work.Elapsed
	}
	if result.MinLatency == 0 || work.Elapsed < result.MinLatency {
		result.MinLatency = work.Elapsed
	}
}

func (e *resultCollector) SetReporter(reporter BenchmarkReporter) {
	e.reporter = reporter
}

func (c *resultCollector) printStatus() {
	result := c.result
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
	fmt.Println("Working:", result.Total-(result.Succeeded+result.Failed))
	fmt.Println()
}

func (c *resultCollector) monitorLoop(ctx context.Context) {
	for {
		select {
		case <-time.After(1 * time.Second):
			c.printStatus()
			if c.reporter != nil {
				c.reporter.PublishReport(ctx)
			}
		case <-ctx.Done():
			return
		}
	}
}
