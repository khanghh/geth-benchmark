package main

import "gopkg.in/urfave/cli.v1"

var (
	benchmarkTypeFlag = cli.UintFlag{
		Name:  "type",
		Usage: "Benchmark type 1-transaction processing, 2-query processing (default: 1)",
		Value: 1,
	}
	rpcUrlFlag = cli.StringFlag{
		Name:  "rpc-url",
		Usage: "RPC url of geth node (default: ws://localhost:8546)",
		Value: "ws://localhost:8546",
	}
	mnemonicFlag = cli.StringFlag{
		Name:  "mnemonic",
		Usage: "Wallet seed phrase file (default: mnemonic.txt)",
		Value: "mnemonic.txt",
	}
	accountsFlag = cli.UintFlag{
		Name:  "accounts",
		Usage: "Number of accounts to conduct the benchmark test (default: 10000)",
		Value: 10000,
	}
	roundsFlags = cli.UintFlag{
		Name:  "rounds",
		Usage: "Number of rounds to conduct the benchmark test (default: 1)",
		Value: 1,
	}
	execRateFlag = cli.UintFlag{
		Name:  "exec-rate",
		Usage: "Benchmark workload execution rate (default: 1000)",
		Value: 1000,
	}
	erc20AddrFlag = cli.StringFlag{
		Name:  "erc20",
		Usage: "ERC20 token address",
	}
)
