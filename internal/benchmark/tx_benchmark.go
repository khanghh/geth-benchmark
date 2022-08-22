package benchmark

import (
	"context"
	"geth-benchmark/internal/benchmark/erc20"
	"geth-benchmark/internal/core"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

const (
	rpcTimeOut       = 10 * time.Second
	walletDerivePath = "m/44'/60'/0'/0/%d"
	erc20Name        = "BenchmarkERC20"
	erc20Symbol      = "BENCH"
)

type TxBenchmark struct {
	rpcUrl     string
	accNonces  map[common.Address]uint64
	accounts   []accounts.Account
	chainId    *big.Int
	client     *ethclient.Client
	erc20Token *erc20.ERC20
	wallet     *hdwallet.Wallet
	mtx        sync.Mutex
}

func (b *TxBenchmark) transferERC20(sender accounts.Account, receiver accounts.Account, value *big.Int) error {
	privateKey, err := b.wallet.PrivateKey(sender)
	if err != nil {
		return err
	}
	opts, err := bind.NewKeyedTransactorWithChainID(privateKey, b.chainId)
	if err != nil {
		return err
	}
	opts.Nonce = big.NewInt(int64(b.takeNonce(sender)))
	opts.Value = big.NewInt(0)
	opts.GasTipCap = big.NewInt(101 * params.GWei) // MaxPriorityFeePerGas
	opts.GasFeeCap = big.NewInt(101 * params.GWei) // MaxFeePerGas
	opts.GasLimit = 500000
	amount := big.NewInt(0)
	_, err = b.erc20Token.Transfer(opts, receiver.Address, amount)
	if err != nil {
		return err
	}
	return nil
}

func (b *TxBenchmark) fetchNonce(ctx context.Context, acc accounts.Account) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeOut)
	defer cancel()
	return b.client.PendingNonceAt(ctx, acc.Address)
}

func (b *TxBenchmark) fetchChainID(ctx context.Context) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeOut)
	defer cancel()
	return b.client.ChainID(ctx)
}

func (b *TxBenchmark) fetchAllNonces() error {
	accNonces := map[common.Address]uint64{}
	wg := NewLimitWaitGroup(len(b.accounts))
	mtx := sync.Mutex{}
	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < len(b.accounts); i++ {
		wg.Add()
		acc := b.accounts[i]
		go func() {
			defer wg.Done()
			nonce, err := b.fetchNonce(ctx, acc)
			if err != nil {
				errCh <- err
				return
			}
			mtx.Lock()
			accNonces[acc.Address] = nonce
			mtx.Unlock()
		}()
	}
	select {
	case err := <-errCh:
		return err
	case <-wg.Wait():
		b.accNonces = accNonces
		return nil
	}
}

func (b *TxBenchmark) takeNonce(acc accounts.Account) uint64 {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	nonce := b.accNonces[acc.Address]
	b.accNonces[acc.Address] += 1
	return nonce
}

func (b *TxBenchmark) deployERC20(acc accounts.Account) (common.Address, *erc20.ERC20, error) {
	privateKey, _ := b.wallet.PrivateKey(acc)
	opts, err := bind.NewKeyedTransactorWithChainID(privateKey, b.chainId)
	if err != nil {
		return common.Address{}, nil, err
	}
	opts.Nonce = big.NewInt(int64(b.takeNonce(acc)))
	opts.Value = big.NewInt(0)
	opts.GasTipCap = big.NewInt(100 * params.GWei)
	opts.GasFeeCap = big.NewInt(101 * params.GWei)

	addr, _, token, err := erc20.DeployERC20(opts, b.client, erc20Name, erc20Symbol)
	if err != nil {
		return addr, nil, err
	}
	return addr, token, nil
}

func (b *TxBenchmark) Prepair() {
	log.Println("Dialing RPC node", b.rpcUrl)
	rpcClient, err := core.DialRpc(b.rpcUrl)
	if err != nil {
		log.Fatal("Could not dial rpc node", err)
	}
	b.client = ethclient.NewClient(rpcClient)
	b.chainId, err = b.fetchChainID(context.Background())
	if err != nil {
		log.Fatal("Could not fetch chainID ", err)
	}
	log.Println("Fetching accounts' nonces.")
	if err := b.fetchAllNonces(); err != nil {
		log.Fatal("Failed to fetch accounts' nonces. ", err)
	}
	erc20Addr, erc20Token, err := b.deployERC20(b.accounts[0])
	if err != nil {
		log.Fatal("Could not deploy ERC20 token. ", err)
	} else {
		log.Println("ERC20 token deployed at:", erc20Addr)
		b.erc20Token = erc20Token
	}
}

func (b *TxBenchmark) DoWork(workerIndex int) error {
	acc := b.accounts[workerIndex]
	return b.transferERC20(acc, acc, big.NewInt(0))
}

func (b *TxBenchmark) OnFinish(roundIndex int, result *BenchmarkResult) {

}

func NewTxBenchmark(rpcUrl string, wallet *hdwallet.Wallet) *TxBenchmark {
	return &TxBenchmark{
		rpcUrl:   rpcUrl,
		wallet:   wallet,
		accounts: wallet.Accounts(),
	}
}
