package main

import "github.com/urfave/cli/v2"

var (
	testcaseFlag = &cli.UintFlag{
		Name:  "testcase",
		Usage: "Bechmark testcase 1-transaction processing, 2-query processing",
		Value: 1,
	}
	rpcUrlFlag = &cli.StringFlag{
		Name:  "rpc-url",
		Usage: "RPC url of geth node",
		Value: "ws://localhost:8546",
	}
	seedPhraseFlag = &cli.StringFlag{
		Name:  "seed",
		Usage: "Wallet seed phrase file. If not provided, default mnemonic is used",
	}
	workersFlag = &cli.UintFlag{
		Name:  "workers",
		Usage: "Number of workers to run the benchmark test",
		Value: 1000,
	}
	connectionsFlags = &cli.UintFlag{
		Name:  "connections",
		Usage: "Number of rpc connection to establish. Connections are shared between workers",
		Value: 1,
	}
	durationFlag = &cli.StringFlag{
		Name:  "duration",
		Usage: "Duration to run the benchmark test",
		Value: "10m",
	}
	execRateFlag = &cli.UintFlag{
		Name:  "exec-rate",
		Usage: "Benchmark workload execution rate",
		Value: 1000,
	}
	erc20AddrFlag = &cli.StringFlag{
		Name:  "erc20",
		Usage: "ERC20 token address",
	}
	txReceiptFlag = &cli.BoolFlag{
		Name:  "receipt",
		Usage: "Wait for transaction's receipt",
	}
	influxDBFlag = &cli.BoolFlag{
		Name:  "influxdb",
		Usage: "Enable influxdb",
	}
	influxDBUrlFlag = &cli.BoolFlag{
		Name:    "influxdb.url",
		Usage:   "InfluxDB url",
		EnvVars: []string{"INFLUXDB_URL"},
	}
	influxDBTokenFlag = &cli.StringFlag{
		Name:    "influxdb.token",
		Usage:   "InfluxDB token",
		EnvVars: []string{"INFLUXDB_TOKEN"},
	}
	influxDBBucketFlag = &cli.StringFlag{
		Name:    "influxdb.bucket",
		Usage:   "InfluxDB bucket",
		EnvVars: []string{"INFLUXDB_BUCKET"},
	}
	influxDBOrgFlag = &cli.StringFlag{
		Name:    "influxdb.org",
		Usage:   "InfluxDB organization",
		EnvVars: []string{"INFLUXDB_ORG"},
	}
)
