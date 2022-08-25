package main

import (
	"context"
	"fmt"
	"geth-benchmark/internal/benchmark"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"

	"gopkg.in/urfave/cli.v1"
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
	app.Flags = []cli.Flag{
		benchmarkTypeFlag,
		rpcUrlFlag,
		mnemonicFlag,
		accountsFlag,
		roundsFlags,
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
	benchmarkType := ctx.GlobalUint(benchmarkTypeFlag.Name)
	rpcUrl := ctx.GlobalString(rpcUrlFlag.Name)
	mnemonicFile := ctx.GlobalString(mnemonicFlag.Name)
	numAccs := ctx.GlobalUint(accountsFlag.Name)
	numRounds := ctx.GlobalUint(roundsFlags.Name)
	execRate := ctx.GlobalUint(execRateFlag.Name)
	erc20AddrStr := ctx.GlobalString(erc20AddrFlag.Name)
	var erc20Addr *common.Address = nil
	if erc20AddrStr != "" {
		addr := common.HexToAddress(erc20AddrStr)
		erc20Addr = &addr
	}

	buf, err := ioutil.ReadFile(mnemonicFile)
	if err != nil {
		log.Fatal(err)
	}
	mnemonic := strings.TrimSpace(string(buf[:]))
	wallet = mustCreateWallet(mnemonic, numAccs)

	engine := benchmark.NewBenchmarkEngine(benchmark.BenchmarkOptions{
		MaxThread:   10000,
		ExecuteRate: int(execRate),
		NumWorkers:  len(wallet.Accounts()),
		NumRounds:   int(numRounds),
		Timeout:     20 * time.Second,
	})

	if benchmarkType == 1 {
		txBechmark := benchmark.NewTxBenchmark(rpcUrl, wallet, erc20Addr)
		engine.SetBenchmark(txBechmark)
	} else if benchmarkType == 2 {
		txBechmark := benchmark.NewCallBenchmark(rpcUrl, wallet, erc20Addr)
		engine.SetBenchmark(txBechmark)
	} else {
		log.Fatal("Unknown benchmark type.")
	}
	fmt.Println("Starting benchmark test...")
	engine.Run(context.Background())
}

func main() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
