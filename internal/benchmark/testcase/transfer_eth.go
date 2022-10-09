package testcase

import (
	"context"
	"crypto/ecdsa"
	"geth-benchmark/internal/benchmark"
	"geth-benchmark/internal/txpool"
	"log"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

type TransferEthWorker struct {
	rpcClient    *rpc.Client
	chainId      *big.Int
	account      accounts.Account
	privateKey   *ecdsa.PrivateKey
	pendingNonce uint64
}

func (w *TransferEthWorker) DoWork(ctx context.Context, workIdx int) error {
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   w.chainId,
		Nonce:     w.pendingNonce,
		GasFeeCap: big.NewInt(101 * params.GWei),
		GasTipCap: big.NewInt(101 * params.GWei),
		Gas:       21000,
		To:        &w.account.Address,
		Value:     big.NewInt(0),
	})
	w.pendingNonce += 1
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(w.chainId), w.privateKey)
	if err != nil {
		return err
	}
	return txpool.NewTxSender(w.rpcClient).SendTransaction(ctx, signedTx)
}

type TransferEth struct {
	SeedPhrase     string
	WaitForReceipt bool
	wallet         *benchmark.TestWallet
	chainId        *big.Int
}

func (t *TransferEth) Name() string {
	return reflect.TypeOf(*t).Name()
}

func (t *TransferEth) Prepair(opts benchmark.Options) {
	rpcClient, err := rpc.Dial(opts.RpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer rpcClient.Close()
	client := ethclient.NewClient(rpcClient)
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	t.chainId = chainId

	log.Printf("Generating %d accounts\n", opts.NumWorkers)
	wallet, err := benchmark.NewTestWallet(t.SeedPhrase, opts.NumWorkers)
	if err != nil {
		log.Fatal(err)
	}
	t.wallet = wallet

	log.Println("Fetching accounts' nonces")
	_, err = t.wallet.FetchNonces(rpcClient)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting transaction monitor")
	if _, err := txpool.InitTxPool(opts.RpcUrl); err != nil {
		log.Fatal(err)
	}
}

func (t *TransferEth) CreateWorker(rpcClient *rpc.Client, workerIdx int) benchmark.BenchmarkWorker {
	worker := &TransferEthWorker{
		rpcClient:    rpcClient,
		chainId:      t.chainId,
		account:      t.wallet.Accounts[workerIdx],
		privateKey:   t.wallet.PrivateKeys[workerIdx],
		pendingNonce: t.wallet.PendingNonces[workerIdx],
	}
	return worker
}

func (t *TransferEth) OnFinish(result *benchmark.BenchmarkResult) {
}
