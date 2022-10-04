package core

import (
	"context"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

var (
	rpcDialTimeout    = 5 * time.Second
	rpcDialRetryDelay = 1 * time.Second
)

func CreateRpcClients(rpcUrl string, numClient int) ([]*rpc.Client, error) {
	clients := make([]*rpc.Client, numClient)
	for idx := 0; idx < numClient; idx++ {
		log.Println("Dialing RPC node", rpcUrl)
		client, err := rpc.Dial(rpcUrl)
		if err != nil {
			return nil, nil
		}
		clients[idx] = client
	}
	return clients, nil
}

func DialRpc(rawUrl string) (*rpc.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcDialTimeout)
	defer cancel()
	return rpc.DialContext(ctx, rawUrl)
}

func TryConnect(ctx context.Context, rawUrl string) (client *rpc.Client, err error) {
	for {
		log.Printf("Dialing Ethereum RPC node %s\n", rawUrl)
		client, err = DialRpc(rawUrl)
		if err == nil {
			return client, nil
		}
		select {
		case <-time.After(rpcDialRetryDelay):
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
