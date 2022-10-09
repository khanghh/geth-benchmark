package testcase

import (
	"context"
	"crypto/ecdsa"
	"geth-benchmark/internal/benchmark"
	"geth-benchmark/internal/benchmark/erc20"
	"geth-benchmark/internal/txpool"
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
	rpcClient    *rpc.Client
	chainId      *big.Int
	account      accounts.Account
	privateKey   *ecdsa.PrivateKey
	pendingNonce uint64
	erc20Token   *erc20.ERC20
}

func (w *TransferERC20Worker) DoWork(ctx context.Context, workIdx int) error {
	opts := bind.TransactOpts{
		From:      w.account.Address,
		Nonce:     big.NewInt(int64(w.pendingNonce)),
		GasFeeCap: big.NewInt(101 * params.GWei),
		GasTipCap: big.NewInt(101 * params.GWei),
		GasLimit:  600000,
		NoSend:    true,
	}
	tx, err := w.erc20Token.Transfer(&opts, w.account.Address, big.NewInt(0))
	if err != nil {
		return err
	}
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(w.chainId), w.privateKey)
	if err != nil {
		return err
	}
	return txpool.NewTxSender(w.rpcClient).SendTransaction(ctx, signedTx)
}

type TransferERC20 struct {
	SeedPhrase     string
	Erc20Addr      common.Address
	WaitForReceipt bool
	wallet         *benchmark.TestWallet
	chainId        *big.Int
}

func (t *TransferERC20) Name() string {
	return reflect.TypeOf(*t).Name()
}

func (t *TransferERC20) Prepair(opts benchmark.Options) {
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

	if t.Erc20Addr == nilAddress {
		log.Println("Deploying ERC20 token")
		erc20Addr, _, err := erc20.DeployBenchmarkToken(context.Background(), rpcClient, wallet.PrivateKeys[0])
		if err != nil {
			log.Fatal("Failed to deploy ERC20 token: ", err)
		}
		t.Erc20Addr = erc20Addr
		log.Println("ERC20Token deployed at", t.Erc20Addr)
	}

	log.Println("Initialize transaction monitor")
	if _, err := txpool.InitTxPool(opts.RpcUrl); err != nil {
		log.Fatal(err)
	}
}

func (t *TransferERC20) CreateWorker(rpcClient *rpc.Client, workerIdx int) benchmark.BenchmarkWorker {
	erc20Token, err := erc20.NewERC20(t.Erc20Addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	worker := &TransferERC20Worker{
		rpcClient:    rpcClient,
		chainId:      t.chainId,
		account:      t.wallet.Accounts[workerIdx],
		privateKey:   t.wallet.PrivateKeys[workerIdx],
		pendingNonce: t.wallet.PendingNonces[workerIdx],
		erc20Token:   erc20Token,
	}
	return worker
}

func (t *TransferERC20) OnFinish(result *benchmark.BenchmarkResult) {
}
