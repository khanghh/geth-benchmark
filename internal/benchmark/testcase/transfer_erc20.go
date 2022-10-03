package testcase

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"geth-benchmark/internal/benchmark"
	"geth-benchmark/internal/benchmark/erc20"
	"log"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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
	erc20Token     *erc20.ERC20
	waitForReceipt waitForReceiptFunc
}

func (w *TransferERC20Worker) eip1559TransferERC20(ctx context.Context, receiverAddr common.Address, amount *big.Int) (*types.Transaction, error) {
	opts := bind.TransactOpts{
		From:      w.account.Address,
		Nonce:     big.NewInt(int64(w.pendingNonce)),
		GasFeeCap: big.NewInt(10 * params.GWei),
		GasTipCap: big.NewInt(10 * params.GWei),
	}
	tx, err := w.erc20Token.Transfer(&opts, receiverAddr, amount)
	if err != nil {
		return nil, err
	}
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
	Erc20Addr      common.Address
	WaitForReceipt bool
	monitor        *benchmark.TxnsMonitor
	wallet         *TestWallet
	chainId        *big.Int
}

func (t *TransferERC20) Name() string {
	return reflect.TypeOf(*t).Name()
}

func (t *TransferERC20) Prepair(opts benchmark.Options) {
	log.Println("Prepairing testcase", t.Name())
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

	if t.Erc20Addr == nilAddress {
		log.Println("Deploying ERC20.", opts.RpcUrl)
		erc20Addr, _, err := erc20.DeployBenchmarkToken(context.Background(), rpcClient, wallet.PrivateKeys[0])
		if err != nil {
			log.Fatal("Failed to deploy ERC20 token", err)
		}
		t.Erc20Addr = erc20Addr
		log.Println("ERC20Token deployed at", t.Erc20Addr)
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
	client := ethclient.NewClient(rpcClient)
	erc20Token, err := erc20.NewERC20(t.Erc20Addr, client)
	if err != nil {
		panic(err)
	}
	worker := &TransferERC20Worker{
		client:       ethclient.NewClient(rpcClient),
		chainId:      t.chainId,
		account:      t.wallet.Accounts[workerIdx],
		privateKey:   t.wallet.PrivateKeys[workerIdx],
		pendingNonce: t.wallet.PendingNonces[workerIdx],
		erc20Token:   erc20Token,
	}
	if t.WaitForReceipt {
		worker.waitForReceipt = t.monitor.WaitForTxnReceipt
	}
	return worker
}

func (t *TransferERC20) OnFinish(result *benchmark.BenchmarkResult) {
}
