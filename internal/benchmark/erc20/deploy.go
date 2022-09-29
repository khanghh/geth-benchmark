package erc20

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	erc20TokenName   = "BenchmarkTest"
	erc20TokenSymbol = "BENCH"
)

var (
	nilAddress = common.Address{}
)

func waitForTxConfirmed(ctx context.Context, rpcClient *rpc.Client, hash common.Hash) (*types.Receipt, error) {
	client := ethclient.NewClient(rpcClient)
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		select {
		case <-time.After(500 * time.Millisecond):
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func DeployBenchmarkToken(ctx context.Context, rpcClient *rpc.Client, privateKey *ecdsa.PrivateKey) (common.Address, *ERC20, error) {
	client := ethclient.NewClient(rpcClient)
	chainId, err := client.ChainID(ctx)
	if err != nil {
		return nilAddress, nil, err
	}
	opts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return nilAddress, nil, err
	}
	opts.Value = big.NewInt(0)
	opts.GasTipCap = big.NewInt(100 * params.GWei)
	opts.GasFeeCap = big.NewInt(101 * params.GWei)
	addr, tx, erc20Token, err := DeployERC20(opts, client, erc20TokenName, erc20TokenSymbol)
	if err != nil {
		return nilAddress, nil, err
	}
	if _, err = waitForTxConfirmed(ctx, rpcClient, tx.Hash()); err != nil {
		return nilAddress, nil, err
	}
	return addr, erc20Token, nil
}
