package testcase

import (
	"geth-benchmark/internal/benchmark"

	"github.com/ethereum/go-ethereum/common"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

type TransferEthWorker struct {
}

func (w *TransferEthWorker) DoWork(workIdx int) error {
	return nil
}

type TransferEthBenchmark struct {
	Wallet    *hdwallet.Wallet
	RpcUrl    string
	Erc20Addr common.Address
}

func (b *TransferEthBenchmark) Prepair() {
}

func (b *TransferEthBenchmark) CreateWorker(workerIndex int) (benchmark.BenchmarkWorker, error) {
	return nil, nil
}

func (b *TransferEthBenchmark) OnFinish(result *benchmark.BenchmarkResult) {

}
