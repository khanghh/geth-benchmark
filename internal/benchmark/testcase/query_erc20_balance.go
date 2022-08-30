package testcase

import (
	"context"
	"fmt"
	"geth-benchmark/internal/benchmark"
	"geth-benchmark/internal/benchmark/erc20"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/jedib0t/go-pretty/v6/table"
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
	SeedPhrase string
	RpcUrl     string
	Erc20Addr  common.Address
	NumClient  int
	accounts   []accounts.Account
	clients    []*rpc.Client
}

func (b *QueryERC20BalanceBenchmark) Prepair() {
	numAcc := 10000
	fmt.Printf("Generating %d accounts\n", numAcc)
	wallet, err := createHDWallet(b.SeedPhrase, numAcc)
	if err != nil {
		log.Fatal("Failed to create HDWallet ", err)
	}
	b.accounts = wallet.Accounts()

	if b.NumClient == 0 {
		b.NumClient = 1
	}
	for i := 0; i < b.NumClient; i++ {
		fmt.Println("Dialing RPC node", b.RpcUrl)
		rpcClient, err := rpc.Dial(b.RpcUrl)
		if err != nil {
			log.Fatal(err)
		}
		b.clients = append(b.clients, rpcClient)
	}

	if b.Erc20Addr == nilAddress {
		privateKey, err := wallet.PrivateKey(b.accounts[0])
		if err != nil {
			log.Fatal(err)
		}
		erc20Addr, _, err := deployERC20(context.Background(), b.clients[0], privateKey)
		if err != nil {
			log.Fatal("Failed to deploy ERC20 token", err)
		}
		b.Erc20Addr = erc20Addr
		fmt.Println("ERC20Token deployed at", b.Erc20Addr)
	}
}

func (b *QueryERC20BalanceBenchmark) CreateWorker(workerIndex int) (benchmark.BenchmarkWorker, error) {
	client := ethclient.NewClient(b.clients[workerIndex%len(b.clients)])
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
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"TestCase", "Total", "Succeeded", "Failed", "AvgLatency", "ExecPerSec", "TimeTaken"})
	t.AppendRows([]table.Row{
		{"Query ERC20 token balance", result.Total, result.Succeeded, result.Failed, result.AvgLatency, result.ExecPerSec, result.TimeTaken},
	})
	t.Render()
}
