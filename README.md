# Ethereum benchmark tool

Benchmark tool for ethereum's forks blockchain

## Usage

```bash
NAME:
   benchmark - Ethereum network benchmark tool

USAGE:
   benchmark [global options] command [command options] [arguments...]
   Example ./bin/benchmark --testcase=1 --rpc-url=ws://localhost:8546 --workers=10000 --exec-rate=100 --duration=1h

VERSION:
   a995c267a5085651a2ad2b2c4f01ba88b401b89a - Thu Sep 29 15:07:52 +07 2022

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --testcase value     Bechmark testcase 1-transaction processing, 2-query processing (default: 1)
   --rpc-url value      RPC url of geth node (default: "ws://localhost:8546")
   --connections value  Number of rpc connection to establish. Connections are shared between workers (default: 1)
   --seed value         Wallet seed phrase file. If not provided, default mnemonic is used
   --workers value      Number of workers to run the benchmark test (default: 1000)
   --duration value     Duration to run the benchmark test (default: "10m")
   --exec-rate value    Benchmark workload execution rate (default: 1000)
   --erc20 value        ERC20 token address
   --receipt            Wait for transaction's receipt
   --help, -h           show help
   --version, -v        print the version
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.
