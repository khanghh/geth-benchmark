package testcase

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"geth-benchmark/internal/benchmark/erc20"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

const (
	erc20TokenName   = "BenchmarkTest"
	erc20TokenSymbol = "BENCH"
)

var (
	nilAddress = common.Address{}
)

func createHDWallet(mnemonic string, numAcc int) (*hdwallet.Wallet, error) {
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}
	for i := 0; i < numAcc; i++ {
		walletDerivePath := fmt.Sprintf("m/44'/60'/0'/0/%d", i)
		derivationPath := hdwallet.MustParseDerivationPath(walletDerivePath)
		_, err := wallet.Derive(derivationPath, true)
		if err != nil {
			return nil, err
		}
	}
	return nil, err
}

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

func deployERC20(ctx context.Context, rpcClient *rpc.Client, privateKey *ecdsa.PrivateKey) (common.Address, *erc20.ERC20, error) {
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
	addr, tx, erc20Token, err := erc20.DeployERC20(opts, client, erc20TokenName, erc20TokenSymbol)
	if err != nil {
		return nilAddress, nil, err
	}
	if _, err = waitForTxConfirmed(ctx, rpcClient, tx.Hash()); err != nil {
		return nilAddress, nil, err
	}
	return addr, erc20Token, nil
}
