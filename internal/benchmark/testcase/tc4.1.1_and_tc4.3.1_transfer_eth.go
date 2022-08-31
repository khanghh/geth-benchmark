package testcase

import (
	"context"
	"errors"
	"fmt"
	"geth-benchmark/internal/benchmark"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

type TransferEthBenchmark struct {
	SeedPhrase     string
	RpcUrl         string
	TxTimeout      time.Duration
	NumAccounts    int
	WaitForReceipt bool
	monitor        *benchmark.TxnsMonitor
	wallet         *hdwallet.Wallet
	chainId        *big.Int
	accounts       []accounts.Account
	nonces         []int64
	client         *rpc.Client
	mtx            sync.Mutex
}

func (w *TransferEthBenchmark) Name() string {
	return "Transfer ETH"
}

func (b *TransferEthBenchmark) transferETH(ctx context.Context, nonce uint64, sender accounts.Account, receiver accounts.Account, value *big.Int) (*types.Transaction, error) {
	client := ethclient.NewClient(b.client)
	privateKey, err := b.wallet.PrivateKey(sender)
	if err != nil {
		return nil, err
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   b.chainId,
		Nonce:     nonce,
		GasFeeCap: big.NewInt(100000 * params.GWei),
		GasTipCap: big.NewInt(100000 * params.GWei),
		Gas:       21000,
		To:        &receiver.Address,
		Value:     value,
	})
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(b.chainId), privateKey)
	if err != nil {
		return nil, err
	}

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return nil, err
	}
	return signedTx, err
}

func (b *TransferEthBenchmark) fetchNonces() error {
	batchReq := []rpc.BatchElem{}
	b.nonces = make([]int64, len(b.accounts))
	for _, acc := range b.accounts {
		batchElem := rpc.BatchElem{
			Method: "eth_getTransactionCount",
			Args:   []interface{}{acc.Address, "pending"},
			Result: new(hexutil.Uint64),
		}
		batchReq = append(batchReq, batchElem)
	}
	err := b.client.BatchCall(batchReq)
	if err != nil {
		return nil
	}
	for idx, elem := range batchReq {
		b.nonces[idx] = int64(*elem.Result.(*hexutil.Uint64))
	}
	return nil
}

func (b *TransferEthBenchmark) Prepair() {
	fmt.Printf("Generating %d accounts\n", b.NumAccounts)
	wallet, err := createHDWallet(b.SeedPhrase, b.NumAccounts)
	if err != nil {
		log.Fatal("Failed to create HDWallet ", err)
	}
	b.wallet = wallet
	b.accounts = wallet.Accounts()

	fmt.Println("Dialing RPC node", b.RpcUrl)
	rpcClient, err := rpc.Dial(b.RpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	b.client = rpcClient

	client := ethclient.NewClient(rpcClient)
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		fmt.Println("Could not fetch chainId", err)
	}
	b.chainId = chainId

	fmt.Println("Fetching accounts' nonces.")
	if err := b.fetchNonces(); err != nil {
		fmt.Println("Failed to fetch accounts' nonces", err)
	}

	if b.WaitForReceipt {
		fmt.Println("Staring transactions monitor")
		fmt.Println("Dialing RPC node", b.RpcUrl)
		rpcClient, err = rpc.Dial(b.RpcUrl)
		if err != nil {
			log.Fatal(err)
		}
		b.monitor, err = benchmark.NewTxnsMonitor(rpcClient)
		if err != nil {
			fmt.Println("Could not create TxnsMonitor", err)
		}
	}
}

func (b *TransferEthBenchmark) takeNonce(accIdx int) int64 {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	nonce := b.nonces[accIdx]
	b.nonces[accIdx] += 1
	return nonce
}

func (w *TransferEthBenchmark) DoWork(ctx context.Context, workIdx int) error {
	accIdx := workIdx % len(w.accounts)
	acc := w.accounts[accIdx]
	nonce := w.takeNonce(accIdx)
	tx, err := w.transferETH(context.Background(), uint64(nonce), acc, acc, big.NewInt(0))
	if err != nil {
		return err
	}
	if w.WaitForReceipt {
		receipt, err := w.monitor.WaitForTxnReceipt(ctx, tx.Hash())
		if err != nil {
			return err
		}
		if receipt.Status == 0 {
			return errors.New("transaction failed")
		}
	}
	return nil
}

func (b *TransferEthBenchmark) CreateWorker(workerIndex int) (benchmark.BenchmarkWorker, error) {
	return b, nil
}

func (b *TransferEthBenchmark) OnFinish(result *benchmark.BenchmarkResult) {
}
