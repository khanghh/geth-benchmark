package core

import (
	"context"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	EventNewHeads     = "newHeads"
	EventNewPendingTx = "newPendingTransactions"
)

var (
	listenerRestartDelay = 1 * time.Second
)

type NewHeadeHandleFunc func(*types.Header)
type NewPendingTxHandleFunc func(common.Hash)

type EthListener struct {
	rpcUrl         string
	client         *rpc.Client
	OnNewHead      NewHeadeHandleFunc
	OnNewPendingTx NewPendingTxHandleFunc
}

func (l *EthListener) listenEventsWorkerImpl(ctx context.Context) error {
	headCh := make(chan *types.Header)
	newHeadSub, err := l.client.EthSubscribe(ctx, headCh, EventNewHeads)
	if err != nil {
		return err
	}

	txHashCh := make(chan common.Hash)
	newTxSub, err := l.client.EthSubscribe(ctx, txHashCh, EventNewPendingTx)
	if err != nil {
		return err
	}

	for {
		select {
		case head := <-headCh:
			go l.OnNewHead(head)
		case txHash := <-txHashCh:
			l.OnNewPendingTx(txHash)
		case err := <-newHeadSub.Err():
			return err
		case err := <-newTxSub.Err():
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func (l *EthListener) listenEventsWorker(ctx context.Context) {
	for {
		l.client, _ = TryConnect(ctx, l.rpcUrl)
		errCh := make(chan error, 1)
		select {
		case errCh <- l.listenEventsWorkerImpl(ctx):
			log.Println("RPC connection closed.", <-errCh)
			log.Println("Restarting listener.")
			time.Sleep(listenerRestartDelay)
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (l *EthListener) Start() {
	l.listenEventsWorker(context.Background())
}

func NewNodeListener(rpcUrl string) *EthListener {
	return &EthListener{
		rpcUrl: rpcUrl,
	}
}
