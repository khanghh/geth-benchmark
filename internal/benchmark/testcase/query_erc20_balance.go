package testcase

import (
	"context"
	"geth-benchmark/internal/benchmark"
	"geth-benchmark/internal/benchmark/erc20"
	"log"
	"reflect"

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
	wallet     *benchmark.TestWallet
}

func (t *QueryERC20Balance) Name() string {
	return reflect.TypeOf(*t).Name()
}

func (t *QueryERC20Balance) Prepair(opts benchmark.Options) {
	log.Println("Prepairing testcase", t.Name())
	log.Printf("Generating %d accounts\n", opts.NumWorkers)
	wallet, err := benchmark.NewTestWallet(t.SeedPhrase, opts.NumWorkers)
	if err != nil {
		log.Fatal("Failed to generate test accounts", err)
	}
	t.wallet = wallet

	if t.Erc20Addr == nilAddress {
		log.Println("Deploying ERC20.", opts.RpcUrl)
		rpcClient, err := rpc.Dial(opts.RpcUrl)
		if err != nil {
			log.Fatal(err)
		}
		defer rpcClient.Close()
		erc20Addr, _, err := erc20.DeployBenchmarkToken(context.Background(), rpcClient, wallet.PrivateKeys[0])
		if err != nil {
			log.Fatal("Failed to deploy ERC20 token", err)
		}
		t.Erc20Addr = erc20Addr
		log.Println("ERC20Token deployed at", t.Erc20Addr)
	}
}

func (t *QueryERC20Balance) CreateWorker(rpcClient *rpc.Client, workerIdx int) benchmark.BenchmarkWorker {
	client := ethclient.NewClient(rpcClient)
	erc20Token, err := erc20.NewERC20(t.Erc20Addr, client)
	if err != nil {
		panic(err)
	}
	acc := t.wallet.Accounts[workerIdx]
	return &QueryERC20BalanceWorker{
		client:     client,
		erc20Token: erc20Token,
		address:    acc.Address,
	}
}

func (t *QueryERC20Balance) OnFinish(result *benchmark.BenchmarkResult) {
}
