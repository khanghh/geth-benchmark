package main

import (
	"context"
	"fmt"
	"geth-benchmark/internal/benchmark"
	"geth-benchmark/internal/benchmark/testcase"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jedib0t/go-pretty/table"

	"github.com/urfave/cli/v2"
)

const (
	defaultSeedPhrase = "test test test test test test test test test test test junk"
)

var (
	gitCommit = ""
	gitDate   = ""
	app       *cli.App
)

func init() {
	app = cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Version = fmt.Sprintf("%s - %s", gitCommit, gitDate)
	app.Usage = "Ethereum network benchmark tool"
	app.Flags = []cli.Flag{
		testcaseFlag,
		rpcUrlFlag,
		connectionsFlags,
		seedPhraseFlag,
		workersFlag,
		durationFlag,
		execRateFlag,
		erc20AddrFlag,
		txReceiptFlag,
		influxDBFlag,
		influxDBUrlFlag,
		influxDBTokenFlag,
		influxDBBucketFlag,
		influxDBOrgFlag,
	}
	app.Action = run
}

func mustLoadSeedPhrase(seedPhraseFile string) string {
	seedPhrase := defaultSeedPhrase
	if seedPhraseFile == "" {
		fmt.Println("No seed phrase file provided. Fall back to default seed phrase.")
	} else {
		buf, err := os.ReadFile(seedPhrase)
		if err != nil {
			log.Fatal(err)
		}
		seedPhrase = strings.TrimSpace(string(buf[:]))
	}
	return seedPhrase
}

func initInfluxDBReporter(ctx *cli.Context) *benchmark.InfluxDBReporter {
	influxDBUrl := ctx.String(influxDBUrlFlag.Name)
	influxDBToken := ctx.String(influxDBTokenFlag.Name)
	influxDBOrg := ctx.String(influxDBOrgFlag.Name)
	influxDBBucket := ctx.String(influxDBBucketFlag.Name)
	return benchmark.NewInfluxDBReporter(influxDBUrl, influxDBToken, influxDBOrg, influxDBBucket, map[string]string{})
}

func printBenchmarkResult(ret *benchmark.BenchmarkResult) {
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
			ret.Testcase,
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

func run(ctx *cli.Context) error {
	testcaseNum := ctx.Uint(testcaseFlag.Name)
	rpcUrl := ctx.String(rpcUrlFlag.Name)
	numClients := ctx.Uint(connectionsFlags.Name)
	seedPhrase := mustLoadSeedPhrase(ctx.String(seedPhraseFlag.Name))
	numWorkers := ctx.Uint(workersFlag.Name)
	durationStr := ctx.String(durationFlag.Name)
	execRate := ctx.Uint(execRateFlag.Name)
	influxDBEnabled := ctx.Bool(influxDBBucketFlag.Name)
	waitForReceipt := ctx.Bool(txReceiptFlag.Name)
	erc20Addr := common.HexToAddress(ctx.String(erc20AddrFlag.Name))

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Fatal("Invalid benchmark duration provided.")
	}

	engine := benchmark.NewBenchmarkEngine(benchmark.Options{
		RpcUrl:      rpcUrl,
		NumClients:  int(numClients),
		NumWorkers:  int(numWorkers),
		ExecuteRate: int(execRate),
		Duration:    duration,
		Timeout:     1 * time.Minute,
	})

	if influxDBEnabled {
		engine.SetReporter(initInfluxDBReporter(ctx))
	}

	var testToRun benchmark.BenchmarkTest
	if testcaseNum == 1 {
		testToRun = &testcase.TransferEth{
			SeedPhrase:     seedPhrase,
			WaitForReceipt: waitForReceipt,
		}
	} else if testcaseNum == 2 {
		testToRun = &testcase.QueryERC20Balance{
			SeedPhrase: seedPhrase,
			Erc20Addr:  erc20Addr,
		}
	} else {
		log.Fatal("Unknown benchmark testcase.")
	}

	fmt.Println("Starting benchmark test.")
	result := engine.Run(context.Background(), testToRun)
	printBenchmarkResult(result)
	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
