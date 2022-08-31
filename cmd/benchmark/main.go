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

	"gopkg.in/urfave/cli.v1"
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
		seedPhraseFlag,
		accountsFlag,
		workersFlag,
		durationFlag,
		execRateFlag,
		erc20AddrFlag,
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

func run(ctx *cli.Context) {
	testcaseNum := ctx.GlobalUint(testcaseFlag.Name)
	rpcUrl := ctx.GlobalString(rpcUrlFlag.Name)
	seedPhrase := mustLoadSeedPhrase(ctx.GlobalString(seedPhraseFlag.Name))
	numWorkers := ctx.GlobalUint(workersFlag.Name)
	numAccs := ctx.GlobalUint(accountsFlag.Name)
	durationStr := ctx.GlobalString(durationFlag.Name)
	execRate := ctx.GlobalUint(execRateFlag.Name)
	erc20Addr := common.HexToAddress(ctx.GlobalString(erc20AddrFlag.Name))

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Fatal("Invalid benchmark duration provided.")
	}

	engine := benchmark.NewBenchmarkEngine(benchmark.BenchmarkOptions{
		ExecuteRate: int(execRate),
		NumWorkers:  int(numWorkers),
		Duration:    duration,
		Timeout:     5 * time.Minute,
	})

	var testToRun benchmark.BenchmarkTest
	if testcaseNum == 1 {
		testToRun = &testcase.TransferEthBenchmark{
			SeedPhrase:  seedPhrase,
			RpcUrl:      rpcUrl,
			NumAccounts: int(numAccs),
		}
	} else if testcaseNum == 2 {
		testToRun = &testcase.QueryERC20BalanceBenchmark{
			SeedPhrase:  seedPhrase,
			RpcUrl:      rpcUrl,
			Erc20Addr:   erc20Addr,
			NumClient:   4,
			NumAccounts: int(numAccs),
		}
	} else {
		log.Fatal("Unknown benchmark testcase.")
	}

	fmt.Println("Starting benchmark test...")
	engine.SetBenchmarkTest(testToRun)
	engine.Run(context.Background())
}

func main() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
