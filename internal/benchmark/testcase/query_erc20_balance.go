package testcase

import (
	"context"
	"fmt"
	"geth-benchmark/internal/benchmark"
	"geth-benchmark/internal/benchmark/erc20"
	"log"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	nilAddress = common.Address{}
)

type QueryERC20BalanceWorker struct {
	client     *ethclient.Client
	erc20Token *erc20.ERC20
	address    common.Address
}

func (w *QueryERC20BalanceWorker) DoWork(ctx context.Context, workIdx int) error {
	_, err := w.erc20Token.BalanceOf(&bind.CallOpts{}, w.address)
	return err
}

type QueryERC20Balance struct {
	SeedPhrase string
	Erc20Addr  common.Address
	wallet     *TestWallet
}

func (w *QueryERC20Balance) Name() string {
	return fmt.Sprintf("%v", QueryERC20Balance{})
}

func (b *QueryERC20Balance) Prepair(opts benchmark.Options) {
	fmt.Printf("Generating %d accounts\n", opts.NumWorkers)
	wallet, err := NewTestWallet(b.SeedPhrase, opts.NumWorkers)
	if err != nil {
		log.Fatal("Failed to generate test accounts", err)
	}
	b.wallet = wallet

	if b.Erc20Addr == nilAddress {
		fmt.Println("Deploying ERC20.", opts.RpcUrl)
		rpcClient, err := rpc.Dial(opts.RpcUrl)
		if err != nil {
			log.Fatal(err)
		}
		defer rpcClient.Close()
		erc20Addr, _, err := erc20.DeployBenchmarkToken(context.Background(), rpcClient, wallet.PrivateKeys[0])
		if err != nil {
			log.Fatal("Failed to deploy ERC20 token", err)
		}
		b.Erc20Addr = erc20Addr
		fmt.Println("ERC20Token deployed at", b.Erc20Addr)
	}
}

func (b *QueryERC20Balance) CreateWorker(rpcClient *rpc.Client, workerIdx int) benchmark.BenchmarkWorker {
	client := ethclient.NewClient(rpcClient)
	erc20Token, err := erc20.NewERC20(b.Erc20Addr, client)
	if err != nil {
		panic(err)
	}
	acc := b.wallet.Accounts[workerIdx]
	return &QueryERC20BalanceWorker{
		client:     client,
		erc20Token: erc20Token,
		address:    acc.Address,
	}
}

func (b *QueryERC20Balance) OnFinish(result *benchmark.BenchmarkResult) {
}
