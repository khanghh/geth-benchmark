# Ethereum benchmark tool

Benchmark tool for ethereum's forks blockchain

## Usage

```bash
USAGE:
   benchmark [global options] command [command options] [arguments...]
   Example ./bin/benchmark --type=1 --rpc-url=ws://localhost:8546 --rounds=1000 --accounts=10000

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --type value      Benchmark type 1-transaction processing, 2-query processing (default: 1) (default: 1)
   --rpc-url value   RPC url of geth node (default: ws://localhost:8546) (default: "ws://localhost:8546")
   --mnemonic value  Wallet seed phrase file (default: mnemonic.txt) (default: "mnemonic.txt")
   --accounts value  Number of accounts to conduct the benchmark test (default: 10000) (default: 10000)
   --rounds value    Number of rounds to conduct the benchmark test (default: 1) (default: 1)
   --help, -h        show help
   --version, -v     print the version
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.
