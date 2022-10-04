package benchmark

import (
	"context"
	"geth-benchmark/internal/core"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/exp/maps"
)

type TxnsMonitor struct {
	clients []*rpc.Client
	txnSubs map[common.Hash]chan *types.Receipt
	mtx     sync.Mutex
}

func (m *TxnsMonitor) fetchTxReceipts(wg sync.WaitGroup, txHashes []common.Hash, retCh chan<- *types.Receipt) {
	defer wg.Done()
	batchReq := []rpc.BatchElem{}
	for txHash := range txHashes {
		batchElem := rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{txHash},
			Result: new(types.Receipt),
		}
		batchReq = append(batchReq, batchElem)
	}
	for _, elem := range batchReq {
		retCh <- elem.Result.(*types.Receipt)
	}
}

func splitTxHashes(txHashes []common.Hash, numPart int) [][]common.Hash {
	totalTxHashes := len(txHashes)
	partAmount := totalTxHashes / numPart
	ret := make([][]common.Hash, numPart)
	for idx := 0; idx < numPart; idx++ {
		part := append([]common.Hash{}, txHashes[idx*partAmount:idx*partAmount+partAmount]...)
		ret[idx] = part
	}
	txIdx := numPart * partAmount
	for idx := 0; idx < numPart; idx++ {
		if txIdx < totalTxHashes {
			ret[idx] = append(ret[idx], txHashes[txIdx])
		}
		txIdx++
	}
	return ret
}

func (m *TxnsMonitor) dispatchTxReceipt(receipt *types.Receipt) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if sub, ok := m.txnSubs[receipt.TxHash]; ok {
		sub <- receipt
		close(sub)
		delete(m.txnSubs, receipt.TxHash)
	}
}

func (m *TxnsMonitor) checkForTxnReceipts() (uint64, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	var txnsConfirmed uint64 = 0
	txHashesArr := splitTxHashes(maps.Keys(m.txnSubs), len(m.clients))
	receiptCh := make(chan *types.Receipt)
	wg := sync.WaitGroup{}
	for idx := 0; idx < len(m.clients); idx++ {
		wg.Add(1)
		go m.fetchTxReceipts(wg, txHashesArr[idx], receiptCh)
	}
	go func() {
		for receipt := range receiptCh {
			m.dispatchTxReceipt(receipt)
			atomic.AddUint64(&txnsConfirmed, 1)
		}
	}()
	wg.Wait()
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
	client := ethclient.NewClient(m.clients[0])
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

func NewTxnsMonitor(rpcUrl string, numClient int) (*TxnsMonitor, error) {
	clients, err := core.CreateRpcClients(rpcUrl, numClient)
	if err != nil {
		return nil, err
	}
	monitor := &TxnsMonitor{
		clients: clients,
		txnSubs: make(map[common.Hash]chan *types.Receipt),
	}
	if err := monitor.start(); err != nil {
		return nil, err
	}
	return monitor, nil
}
