package testcase

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"geth-benchmark/internal/benchmark"
	"log"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

type TransferERC20Worker struct {
	client         *ethclient.Client
	chainId        *big.Int
	account        accounts.Account
	privateKey     *ecdsa.PrivateKey
	pendingNonce   uint64
	waitForReceipt waitForReceiptFunc
}

func (w *TransferERC20Worker) eip1559TransferERC20(ctx context.Context, receiverAddr common.Address, value *big.Int) (*types.Transaction, error) {
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   w.chainId,
		Nonce:     w.pendingNonce,
		GasFeeCap: big.NewInt(10 * params.GWei),
		GasTipCap: big.NewInt(10 * params.GWei),
		Gas:       21000,
		To:        &receiverAddr,
		Value:     value,
	})
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(w.chainId), w.privateKey)
	if err != nil {
		return nil, err
	}

	if err := w.client.SendTransaction(ctx, signedTx); err != nil {
		return nil, err
	}
	w.pendingNonce += 1
	return signedTx, err
}

func (w *TransferERC20Worker) DoWork(ctx context.Context, workIdx int) error {
	tx, err := w.eip1559TransferERC20(context.Background(), w.account.Address, big.NewInt(0))
	if err != nil {
		return err
	}
	if w.waitForReceipt != nil {
		receipt, err := w.waitForReceipt(ctx, tx.Hash())
		if err != nil {
			return err
		}
		if receipt.Status == 0 {
			return errors.New("transaction reverted")
		}
	}
	return nil
}

type TransferERC20 struct {
	SeedPhrase     string
	WaitForReceipt bool
	monitor        *benchmark.TxnsMonitor
	wallet         *TestWallet
	chainId        *big.Int
}

func (t *TransferERC20) Name() string {
	return reflect.TypeOf(*t).Name()
}

func (t *TransferERC20) Prepair(opts benchmark.Options) {
	log.Println("Prepairing testcase ", t.Name())
	rpcClient, err := rpc.Dial(opts.RpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	client := ethclient.NewClient(rpcClient)
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	t.chainId = chainId

	log.Printf("Generating %d accounts\n", opts.NumWorkers)
	wallet, err := NewTestWallet(t.SeedPhrase, opts.NumWorkers)
	if err != nil {
		log.Fatal(err)
	}
	t.wallet = wallet

	log.Println("Fetching accounts' nonces.")
	_, err = t.wallet.FetchNonces(rpcClient)
	if err != nil {
		log.Fatal(err)
	}
	if t.WaitForReceipt {
		log.Println("Staring transactions monitor.")
		t.monitor, err = benchmark.NewTxnsMonitor(rpcClient)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		rpcClient.Close()
	}
}

func (t *TransferERC20) CreateWorker(rpcClient *rpc.Client, workerIdx int) benchmark.BenchmarkWorker {
	worker := &TransferEthWorker{
		client:       ethclient.NewClient(rpcClient),
		chainId:      t.chainId,
		account:      t.wallet.Accounts[workerIdx],
		privateKey:   t.wallet.PrivateKeys[workerIdx],
		pendingNonce: t.wallet.PendingNonces[workerIdx],
	}
	if t.WaitForReceipt {
		worker.waitForReceipt = t.monitor.WaitForTxnReceipt
	}
	return worker
}

func (t *TransferERC20) OnFinish(result *benchmark.BenchmarkResult) {
}
