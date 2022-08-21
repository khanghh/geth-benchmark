package benchmark

import (
	"context"
	"geth-benchmark/internal/core"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

const (
	rpcTimeOut       = 10 * time.Second
	walletDerivePath = "m/44'/60'/0'/0/%d"
)

type TxBenchmark struct {
	rpcUrl    string
	accNonces map[common.Address]uint64
	accounts  []accounts.Account
	client    *ethclient.Client
	wallet    *hdwallet.Wallet
}

func (b *TxBenchmark) TransferERC20(workerIndex int) error {
	return nil
}

func (b *TxBenchmark) fetchNonce(ctx context.Context, acc *accounts.Account) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeOut)
	defer cancel()
	return b.client.PendingNonceAt(ctx, acc.Address)
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
		acc := &b.accounts[i]
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

func (b *TxBenchmark) Prepair() error {
	log.Println("Dialing RPC node", b.rpcUrl)
	rpcClient, err := core.DialRpc(b.rpcUrl)
	if err != nil {
		return err
	}
	b.client = ethclient.NewClient(rpcClient)
	log.Println("Fetching accounts' nonces.")
	if err := b.fetchAllNonces(); err != nil {
		return err
	}
	return nil
}

func (b *TxBenchmark) DoWork(workerIndex int) error {
	return nil
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
