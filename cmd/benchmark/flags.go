package main

import "gopkg.in/urfave/cli.v1"

var (
	testcaseFlag = cli.UintFlag{
		Name:  "testcase",
		Usage: "Bechmark testcase 1-transaction processing, 2-query processing",
		Value: 1,
	}
	rpcUrlFlag = cli.StringFlag{
		Name:  "rpc-url",
		Usage: "RPC url of geth node",
		Value: "ws://localhost:8546",
	}
	seedPhraseFlag = cli.StringFlag{
		Name:  "seed",
		Usage: "Wallet seed phrase file. If not provided, default mnemonic is used",
	}
	accountsFlag = cli.UintFlag{
		Name:  "accounts",
		Usage: "Number of accounts to run the benchmark test",
		Value: 1000,
	}
	workersFlag = cli.UintFlag{
		Name:  "workers",
		Usage: "Number of workers to run the benchmark test",
		Value: 1000,
	}
	durationFlag = cli.StringFlag{
		Name:  "duration",
		Usage: "Duration to run the benchmark test",
		Value: "10m",
	}
	execRateFlag = cli.UintFlag{
		Name:  "exec-rate",
		Usage: "Benchmark workload execution rate",
		Value: 1000,
	}
	erc20AddrFlag = cli.StringFlag{
		Name:  "erc20",
		Usage: "ERC20 token address",
	}
	txReceiptFlag = cli.BoolFlag{
		Name:  "receipt",
		Usage: "Wait for transaction receipt",
	}
)
