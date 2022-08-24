package benchmark

import (
	"context"
	"crypto/ecdsa"
	"geth-benchmark/internal/benchmark/erc20"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

const (
	deployTimeout = 10 * time.Second
)

type ContractDeployment struct {
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
}

func (d *ContractDeployment) waitForTxConfirmed(ctx context.Context, hash common.Hash) (*types.Transaction, error) {
	for {
		tx, pending, _ := d.client.TransactionByHash(ctx, hash)
		if !pending {
			return tx, nil
		}
		select {
		case <-time.After(500 * time.Millisecond):
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (d *ContractDeployment) deployERC20(pCtx context.Context) (*common.Address, *erc20.ERC20, error) {
	ctx, cancel := context.WithTimeout(pCtx, deployTimeout)
	defer cancel()
	chainId, err := d.client.ChainID(ctx)
	if err != nil {
		return nil, nil, err
	}
	opts, err := bind.NewKeyedTransactorWithChainID(d.privateKey, chainId)
	if err != nil {
		return nil, nil, err
	}
	opts.Value = big.NewInt(0)
	opts.GasTipCap = big.NewInt(100 * params.GWei)
	opts.GasFeeCap = big.NewInt(101 * params.GWei)
	addr, tx, token, err := erc20.DeployERC20(opts, d.client, erc20Name, erc20Symbol)
	if err != nil {
		return nil, nil, err
	}
	if _, err = d.waitForTxConfirmed(ctx, tx.Hash()); err != nil {
		return nil, nil, err
	}
	return &addr, token, nil
}

func (d *ContractDeployment) getERC20TokenByAddress(erc20Addr *common.Address) (*common.Address, *erc20.ERC20, error) {
	erc20Token, err := erc20.NewERC20(*erc20Addr, d.client)
	if err != nil {
		return nil, nil, err
	}
	return erc20Addr, erc20Token, nil
}

func NewContractDeployment(client *ethclient.Client, privateKey *ecdsa.PrivateKey) *ContractDeployment {
	return &ContractDeployment{
		client:     client,
		privateKey: privateKey,
	}
}
