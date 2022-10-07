package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	txsMonitor *TxsMonitor
)

type BlockTransactions struct {
	Number       *big.Int
	Transactions []common.Hash
}

type TxsMonitor struct {
	rpcClient  *rpc.Client
	newHeadCh  chan *types.Header
	newHeadSub ethereum.Subscription
	txSubs     map[common.Hash]chan int
	mtx        sync.Mutex
}

func (p *TxsMonitor) fetchBlockTxs(blockNum *big.Int) ([]common.Hash, error) {
	var resp struct {
		Transactions []common.Hash `json:"transactions"`
	}
	err := p.rpcClient.Call(&resp, "eth_getBlockByNumber", hexutil.EncodeBig(blockNum), false)
	if err != nil {
		return nil, err
	}
	return resp.Transactions, nil
}

func (m *TxsMonitor) dispatchTxConfirmed(txHash common.Hash) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if sub, ok := m.txSubs[txHash]; ok {
		sub <- 1
		close(sub)
	}
}

func (p *TxsMonitor) handleNewHead(head *types.Header) {
	txHashes, err := p.fetchBlockTxs(head.Number)
	if err != nil {
		log.Fatal(err)
	}
	for _, txHash := range txHashes {
		p.dispatchTxConfirmed(txHash)
	}
	fmt.Printf("=> New head #%d: %d/%d transactions confrimed\n", head.Number, len(txHashes), len(p.txSubs))
}

func (p *TxsMonitor) eventLoop() {
	for {
		select {
		case head := <-p.newHeadCh:
			p.handleNewHead(head)
		case <-p.newHeadSub.Err():
			log.Fatalln("TxPool exited.")
		}
	}
}

func (p *TxsMonitor) start() (err error) {
	ctx := context.Background()
	client := ethclient.NewClient(p.rpcClient)
	p.newHeadSub, err = client.SubscribeNewHead(ctx, p.newHeadCh)
	if err != nil {
		return err
	}
	go p.eventLoop()
	return nil
}

func (p *TxsMonitor) add(txHash common.Hash) chan int {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	subCh := make(chan int, 1)
	p.txSubs[txHash] = subCh
	return subCh
}

func (p *TxsMonitor) remove(txHash common.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	delete(p.txSubs, txHash)
}

func (p *TxsMonitor) WaitForTxConfirmed(ctx context.Context, txHash common.Hash) error {
	subCh := p.add(txHash)
	defer p.remove(txHash)
	select {
	case <-subCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func NewTxsMonitor(rpcUrl string) (*TxsMonitor, error) {
	rpcClient, err := DialRpc(rpcUrl)
	if err != nil {
		return nil, err
	}
	txpool := &TxsMonitor{
		rpcClient: rpcClient,
		newHeadCh: make(chan *types.Header),
		txSubs:    make(map[common.Hash]chan int),
	}
	if err := txpool.start(); err != nil {
		return nil, err
	}
	return txpool, nil
}

func InitTxsMonitor(rpcUrl string) (*TxsMonitor, error) {
	var err error = nil
	txsMonitor, err = NewTxsMonitor(rpcUrl)
	return txsMonitor, err
}

func WaitForTxConfirmed(ctx context.Context, txHash common.Hash) error {
	if txsMonitor != nil {
		return txsMonitor.WaitForTxConfirmed(ctx, txHash)
	}
	return errors.New("not init")
}
