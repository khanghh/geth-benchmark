package testcase

import (
	"crypto/ecdsa"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

type TestWallet struct {
	Accounts      []accounts.Account
	PrivateKeys   []*ecdsa.PrivateKey
	PendingNonces []uint64
	mtx           sync.Mutex
}

func (w *TestWallet) generateAccounts(seedPhrase string, numAcc int) error {
	wallet, err := hdwallet.NewFromMnemonic(seedPhrase)
	if err != nil {
		return err
	}
	accounts := []accounts.Account{}
	privateKeys := []*ecdsa.PrivateKey{}
	for i := 0; i < numAcc; i++ {
		walletDerivePath := fmt.Sprintf("m/44'/60'/0'/0/%d", i)
		derivationPath := hdwallet.MustParseDerivationPath(walletDerivePath)
		acc, err := wallet.Derive(derivationPath, true)
		if err != nil {
			return err
		}
		privateKey, err := wallet.PrivateKey(acc)
		if err != nil {
			return err
		}
		accounts = append(accounts, acc)
		privateKeys = append(privateKeys, privateKey)
	}
	w.Accounts, w.PrivateKeys = accounts, privateKeys
	return nil
}

func (w *TestWallet) FetchNonces(client *rpc.Client) ([]uint64, error) {
	batchReq := []rpc.BatchElem{}
	w.PendingNonces = make([]uint64, len(w.Accounts))
	for _, acc := range w.Accounts {
		batchElem := rpc.BatchElem{
			Method: "eth_getTransactionCount",
			Args:   []interface{}{acc.Address, "pending"},
			Result: new(hexutil.Uint64),
		}
		batchReq = append(batchReq, batchElem)
	}
	err := client.BatchCall(batchReq)
	if err != nil {
		return nil, nil
	}
	for idx, elem := range batchReq {
		w.PendingNonces[idx] = uint64(*elem.Result.(*hexutil.Uint64))
	}
	return w.PendingNonces, nil
}

func (b *TestWallet) GetPendingNonce(idx int) uint64 {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	nonce := b.PendingNonces[idx]
	b.PendingNonces[idx] += 1
	return nonce
}

func NewTestWallet(seedPhrase string, numAcc int) (*TestWallet, error) {
	wallet := &TestWallet{}
	if err := wallet.generateAccounts(seedPhrase, numAcc); err != nil {
		return nil, err
	}
	return wallet, nil
}
