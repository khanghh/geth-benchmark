package benchmark

import (
	"context"
	"fmt"
	"math/big"
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

func (m *TxnsMonitor) parseReceiptsBlock(blockNum *big.Int) ([]*types.Receipt, error) {
	block, err := ethclient.NewClient(m.client).BlockByNumber(context.Background(), blockNum)
	if err != nil {
		return nil, nil
	}
	batchReq := []rpc.BatchElem{}
	transactions := block.Transactions()
	for _, tx := range transactions {
		batchElem := rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{tx.Hash()},
			Result: new(types.Receipt),
		}
		batchReq = append(batchReq, batchElem)
	}
	if err := m.client.BatchCall(batchReq); err != nil {
		return nil, err
	}
	ret := make([]*types.Receipt, len(transactions))
	for idx, elem := range batchReq {
		receipt := elem.Result.(*types.Receipt)
		ret[idx] = receipt
	}
	return ret, nil
}

func (m *TxnsMonitor) dispatchReceipts(receipts []*types.Receipt) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	for _, receipt := range receipts {
		if sub, ok := m.txnSubs[receipt.TxHash]; ok {
			sub <- receipt
			close(sub)
		}
	}
}

func (m *TxnsMonitor) mainLoop(headCh chan *types.Header) {
	for head := range headCh {
		fmt.Println("onNewHead", head.Number)
		go func(blockNum *big.Int, startTime time.Time) {
			receipts, err := m.parseReceiptsBlock(blockNum)
			if err != nil {
				fmt.Println("Could not parse block", blockNum, err)
				return
			}
			fmt.Printf("=====> New head %d: %d txns. take %v\n", blockNum, len(receipts), time.Since(startTime))
			m.dispatchReceipts(receipts)
		}(head.Number, time.Now())
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
		fmt.Println("TxnsMonitor exited.", err)
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
