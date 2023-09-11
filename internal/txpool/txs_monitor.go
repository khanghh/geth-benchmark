package txpool

import (
	"context"
	"fmt"
	"geth-benchmark/internal/core"
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
	pool *TxPool
)

type TxSender interface {
	SendTransaction(ctx context.Context, signedTx *types.Transaction) error
}

type txSenderImpl struct {
	rpcClient *rpc.Client
}

// SendTransaction send and wait until transaction minted
func (sender *txSenderImpl) SendTransaction(ctx context.Context, signedTx *types.Transaction) error {
	subCh := pool.subscribeTxConfirmed(signedTx.Hash())
	client := ethclient.NewClient(sender.rpcClient)
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return err
	}
	select {
	case <-subCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type TxPool struct {
	rpcClient  *rpc.Client
	newHeadCh  chan *types.Header
	newHeadSub ethereum.Subscription
	txSubs     map[common.Hash]chan int
	mtx        sync.Mutex
}

func (p *TxPool) fetchBlockTxs(blockNum *big.Int) ([]common.Hash, error) {
	var resp struct {
		Transactions []common.Hash `json:"transactions"`
	}
	err := p.rpcClient.Call(&resp, "eth_getBlockByNumber", hexutil.EncodeBig(blockNum), false)
	if err != nil {
		return nil, err
	}
	return resp.Transactions, nil
}

func (p *TxPool) handleNewHead(head *types.Header) {
	totalTxs := len(p.txSubs)
	txHashes, err := p.fetchBlockTxs(head.Number)
	if err != nil {
		log.Fatal(err)
	}
	p.dispatchTxsConfirmed(txHashes)
	fmt.Printf("=> New head #%d: %d/%d transactions confrimed\n", head.Number, len(txHashes), totalTxs)
}

func (p *TxPool) eventLoop() {
	for {
		select {
		case head := <-p.newHeadCh:
			p.handleNewHead(head)
		case <-p.newHeadSub.Err():
			log.Fatalln("TxPool exited.")
		}
	}
}

func (p *TxPool) start() (err error) {
	ctx := context.Background()
	client := ethclient.NewClient(p.rpcClient)
	p.newHeadSub, err = client.SubscribeNewHead(ctx, p.newHeadCh)
	if err != nil {
		return err
	}
	go p.eventLoop()
	return nil
}

func (p *TxPool) subscribeTxConfirmed(txHash common.Hash) chan int {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	subCh := make(chan int, 1)
	p.txSubs[txHash] = subCh
	return subCh
}

func (m *TxPool) dispatchTxsConfirmed(txHashes []common.Hash) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	for _, txHash := range txHashes {
		if sub, ok := m.txSubs[txHash]; ok {
			sub <- 1
			close(sub)
			delete(m.txSubs, txHash)
		}
	}
}

func NewTxPool(rpcUrl string) (*TxPool, error) {
	rpcClient, err := core.DialRpc(rpcUrl)
	if err != nil {
		return nil, err
	}
	txpool := &TxPool{
		rpcClient: rpcClient,
		newHeadCh: make(chan *types.Header),
		txSubs:    make(map[common.Hash]chan int),
	}
	if err := txpool.start(); err != nil {
		return nil, err
	}
	return txpool, nil
}

func InitTxPool(rpcUrl string) (*TxPool, error) {
	var err error = nil
	pool, err = NewTxPool(rpcUrl)
	return pool, err
}

func NewTxSender(rpcClient *rpc.Client) TxSender {
	return &txSenderImpl{rpcClient}
}
