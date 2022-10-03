package benchmark

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type TxnsMonitor struct {
	client  *rpc.Client
	txnSubs map[common.Hash]chan *types.Receipt
	mtx     sync.Mutex
}

func (m *TxnsMonitor) checkForTxnReceipts() (int, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	txnsConfirmed := 0
	batchReq := []rpc.BatchElem{}
	for txHash := range m.txnSubs {
		batchElem := rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{txHash},
			Result: new(types.Receipt),
		}
		batchReq = append(batchReq, batchElem)
	}
	if err := m.client.BatchCall(batchReq); err != nil {
		return 0, err
	}
	for _, elem := range batchReq {
		receipt := elem.Result.(*types.Receipt)
		if sub, ok := m.txnSubs[receipt.TxHash]; ok {
			sub <- receipt
			close(sub)
			delete(m.txnSubs, receipt.TxHash)
			txnsConfirmed += 1
		}
	}
	return txnsConfirmed, nil
}

func (m *TxnsMonitor) mainLoop(headCh chan *types.Header) {
	for head := range headCh {
		startTime := time.Now()
		txnCount := len(m.txnSubs)
		txnsConfirmed, err := m.checkForTxnReceipts()
		if err != nil {
			log.Println("Could not fetch tx receipts", err)
			return
		}
		log.Printf("=====> New head %d: %d/%d txns confirmed. take %v\n", head.Number, txnsConfirmed, txnCount, time.Since(startTime))
	}
}

func (m *TxnsMonitor) start() error {
	client := ethclient.NewClient(m.client)
	headCh := make(chan *types.Header)
	subs, err := client.SubscribeNewHead(context.Background(), headCh)
	if err != nil {
		return err
	}
	go func() {
		err := <-subs.Err()
		log.Println("TxnsMonitor exited.", err)
		close(headCh)
	}()
	go m.mainLoop(headCh)
	return nil
}

func (m *TxnsMonitor) WaitForTxnReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	m.mtx.Lock()
	subCh := make(chan *types.Receipt, 1)
	m.txnSubs[txHash] = subCh
	m.mtx.Unlock()

	select {
	case receipt := <-subCh:
		return receipt, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func NewTxnsMonitor(client *rpc.Client) (*TxnsMonitor, error) {
	monitor := &TxnsMonitor{
		client:  client,
		txnSubs: make(map[common.Hash]chan *types.Receipt),
	}
	if err := monitor.start(); err != nil {
		return nil, err
	}
	return monitor, nil
}
