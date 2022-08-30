package testcase

import (
	"context"
	"geth-benchmark/internal/benchmark"
	"geth-benchmark/internal/benchmark/erc20"
	"log"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

type QueryERC20BalanceWorker struct {
	client     *ethclient.Client
	erc20Token *erc20.ERC20
	accounts   []accounts.Account
}

func (w *QueryERC20BalanceWorker) DoWork(workIdx int) error {
	acc := w.accounts[workIdx%len(w.accounts)]
	_, err := w.erc20Token.BalanceOf(&bind.CallOpts{}, acc.Address)
	return err
}

type QueryERC20BalanceBenchmark struct {
	Wallet    *hdwallet.Wallet
	RpcUrl    string
	Erc20Addr common.Address
	accounts  []accounts.Account
}

func (b *QueryERC20BalanceBenchmark) Prepair() {
	b.accounts = b.Wallet.Accounts()
	if b.Erc20Addr == nilAddress {
		rpcClient, err := rpc.Dial(b.RpcUrl)
		if err != nil {
			log.Fatal(err)
		}
		privateKey, err := b.Wallet.PrivateKey(b.accounts[0])
		if err != nil {
			log.Fatal(err)
		}
		erc20Addr, _, err := deployERC20(context.Background(), rpcClient, privateKey)
		if err != nil {
			log.Fatal("Failed to deploy ERC20 token", err)
		}
		b.Erc20Addr = erc20Addr
	}
}

func (b *QueryERC20BalanceBenchmark) CreateWorker(workerIndex int) (benchmark.BenchmarkWorker, error) {
	rpcClient, err := rpc.Dial(b.RpcUrl)
	if err != nil {
		return nil, err
	}
	client := ethclient.NewClient(rpcClient)
	erc20Token, err := erc20.NewERC20(b.Erc20Addr, client)
	if err != nil {
		return nil, err
	}
	return &QueryERC20BalanceWorker{
		client:     client,
		erc20Token: erc20Token,
		accounts:   b.accounts,
	}, nil
}

func (b *QueryERC20BalanceBenchmark) OnFinish(result *benchmark.BenchmarkResult) {

}
