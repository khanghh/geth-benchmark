package benchmark

import (
	"context"
	"geth-benchmark/internal/benchmark/erc20"
	"geth-benchmark/internal/core"
	"log"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

type CallBenchmark struct {
	rpcUrl     string
	erc20Addr  *common.Address
	erc20Token *erc20.ERC20
	client     *ethclient.Client
	wallet     *hdwallet.Wallet
	accounts   []accounts.Account
	mtx        sync.Mutex
}

func (b *CallBenchmark) getOrDeployERC20() (*common.Address, *erc20.ERC20, error) {
	if b.erc20Addr != nil {
		erc20Token, err := erc20.NewERC20(*b.erc20Addr, b.client)
		if err != nil {
			return nil, nil, err
		}
		return b.erc20Addr, erc20Token, nil
	}
	privateKey, err := b.wallet.PrivateKey(b.accounts[0])
	if err != nil {
		return nil, nil, err
	}
	deployment := NewContractDeployment(b.client, privateKey)
	return deployment.deployERC20(context.Background())
}

func (b *CallBenchmark) Prepair() {
	log.Println("Dialing RPC node", b.rpcUrl)
	rpcClient, err := core.DialRpc(b.rpcUrl)
	if err != nil {
		log.Fatal("Could not dial rpc node", err)
	}
	b.client = ethclient.NewClient(rpcClient)
	b.erc20Addr, b.erc20Token, err = b.getOrDeployERC20()
	if err != nil {
		log.Fatal("Could not deploy ERC20 token. ", err)
	} else {
		log.Println("ERC20 token deployed at:", b.erc20Addr)
	}
}

func (b *CallBenchmark) queryERC20Blance(ctx context.Context, acc accounts.Account) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeOut)
	defer cancel()
	opts := &bind.CallOpts{
		Context: ctx,
	}
	return b.erc20Token.BalanceOf(opts, acc.Address)
}

func (b *CallBenchmark) DoWork(ctx context.Context, workerIndex int) error {
	acc := b.accounts[workerIndex]
	_, err := b.queryERC20Blance(ctx, acc)
	return err
}

func (b *CallBenchmark) OnFinish(roundIndex int, result *BenchmarkResult) {

}

func NewCallBenchmark(rpcUrl string, wallet *hdwallet.Wallet, erc20Addr *common.Address) *CallBenchmark {
	return &CallBenchmark{
		rpcUrl:    rpcUrl,
		erc20Addr: erc20Addr,
		wallet:    wallet,
		accounts:  wallet.Accounts(),
	}
}
