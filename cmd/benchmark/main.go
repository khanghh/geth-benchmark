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
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"

	"gopkg.in/urfave/cli.v1"
)

const (
	defaultMnemonic = "test test test test test test test test test test test junk"
)

var (
	gitCommit = ""
	gitDate   = ""
	app       *cli.App
	wallet    *hdwallet.Wallet
)

func init() {
	app = cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Version = fmt.Sprintf("%s - %s", gitCommit, gitDate)
	app.Usage = "Ethereum network benchmark tool"
	app.Flags = []cli.Flag{
		testcaseFlag,
		rpcUrlFlag,
		mnemonicFlag,
		accountsFlag,
		durationFlag,
		execRateFlag,
		erc20AddrFlag,
	}
	app.Action = run
}

func mustCreateWallet(mnemonic string, numAcc uint) *hdwallet.Wallet {
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		log.Fatal("Could not create HD wallet. ", err)
	}
	for i := 0; i < int(numAcc); i++ {
		walletDerivePath := fmt.Sprintf("m/44'/60'/0'/0/%d", i)
		derivationPath := hdwallet.MustParseDerivationPath(walletDerivePath)
		_, err := wallet.Derive(derivationPath, true)
		if err != nil {
			log.Fatal("Could not generate test accounts.", err)
		}
	}
	return wallet
}

func run(ctx *cli.Context) {
	testcaseNum := ctx.GlobalUint(testcaseFlag.Name)
	rpcUrl := ctx.GlobalString(rpcUrlFlag.Name)
	mnemonicFile := ctx.GlobalString(mnemonicFlag.Name)
	numAccs := ctx.GlobalUint(accountsFlag.Name)
	durationStr := ctx.GlobalString(durationFlag.Name)
	execRate := ctx.GlobalUint(execRateFlag.Name)
	erc20Addr := common.HexToAddress(ctx.GlobalString(erc20AddrFlag.Name))

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Fatal("Invalid benchmakr duration provided.")
	}

	mnemonic := defaultMnemonic
	if mnemonicFile != "" {
		buf, err := os.ReadFile(mnemonicFile)
		if err != nil {
			log.Fatal(err)
		}
		mnemonic = strings.TrimSpace(string(buf[:]))
	}
	wallet = mustCreateWallet(mnemonic, numAccs)

	engine := benchmark.NewBenchmarkEngine(benchmark.BenchmarkOptions{
		ExecuteRate: int(execRate),
		NumWorkers:  len(wallet.Accounts()),
		Duration:    duration,
		Timeout:     20 * time.Second,
	})

	var testToRun benchmark.BenchmarkTest
	if testcaseNum == 1 {
		testToRun = &testcase.TransferEthBenchmark{
			RpcUrl:    rpcUrl,
			Erc20Addr: erc20Addr,
		}
	} else if testcaseNum == 2 {
		testToRun = &testcase.QueryERC20BalanceBenchmark{
			RpcUrl:    rpcUrl,
			Erc20Addr: erc20Addr,
			Wallet:    wallet,
		}
	} else {
		log.Fatal("Unknown benchmark type.")
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
