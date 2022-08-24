package monitor

import (
	"context"
	"geth-benchmark/internal/core"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	EventNewHeads     = "newHeads"
	EventNewPendingTx = "newPendingTransactions"
)

var (
	engineRestartDelay             = 1 * time.Second
	enginePendingTxHashChannelSize = 1000
)

type Options struct {
	RpcClient    *rpc.Client
	Registry     metrics.Registry
	PollInterval time.Duration
}

type EthEvent struct {
	Client  *ethclient.Client
	Name    string
	Payload interface{}
}

type MetricsCollector interface {
	Setup(registry metrics.Registry)
	Update()
	OnEthEvent(event *EthEvent)
}

type MetricsOptions struct {
	RpcUrl         string
	UpdateInterval time.Duration
	Collectors     []MetricsCollector
}

type MetricsEngine struct {
	*MetricsOptions
	registry     metrics.Registry
	client       *rpc.Client
	updateTicker *time.Ticker
}

func (e *MetricsEngine) dispatchEvent(eventName string, data interface{}) {
	for _, collector := range e.Collectors {
		go collector.OnEthEvent(&EthEvent{
			Client:  ethclient.NewClient(e.client),
			Name:    eventName,
			Payload: data,
		})
	}
}

func (e *MetricsEngine) updateCollectors() {
	for _, collector := range e.Collectors {
		go collector.OnEthEvent(&EthEvent{})
	}
}

func (e *MetricsEngine) listenEventsWorkerImpl(ctx context.Context) error {
	headCh := make(chan *types.Header)
	newHeadSub, err := e.client.EthSubscribe(ctx, headCh, EventNewHeads)
	if err != nil {
		return err
	}

	txHashCh := make(chan common.Hash, enginePendingTxHashChannelSize)
	newTxnSub, err := e.client.EthSubscribe(ctx, txHashCh, EventNewPendingTx)
	if err != nil {
		return err
	}

	e.updateTicker = time.NewTicker(e.UpdateInterval)
	for {
		select {
		case head := <-headCh:
			e.dispatchEvent(EventNewHeads, head)
		case txHash := <-txHashCh:
			e.dispatchEvent(EventNewPendingTx, txHash)
		case err := <-newHeadSub.Err():
			return err
		case <-e.updateTicker.C:
			e.updateCollectors()
		case err := <-newTxnSub.Err():
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func (e *MetricsEngine) listenEventsWorker() {
	ctx := context.TODO()
	for {
		e.client, _ = core.TryConnect(ctx, e.RpcUrl)
		errCh := make(chan error, 1)
		select {
		case errCh <- e.listenEventsWorkerImpl(ctx):
			log.Println("RPC connection closed.", <-errCh)
			log.Println("Restarting metrics server.")
			time.Sleep(engineRestartDelay)
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (e *MetricsEngine) Registry() metrics.Registry {
	return e.registry
}

func NewMetricsEngine(opts MetricsOptions) *MetricsEngine {
	engine := &MetricsEngine{
		MetricsOptions: &opts,
	}
	go engine.listenEventsWorker()
	return engine
}
