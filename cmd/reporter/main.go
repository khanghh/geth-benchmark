package main

import (
	"fmt"
	"geth-benchmark/internal/core"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gopkg.in/urfave/cli.v1"
)

var (
	gitCommit = ""
	gitDate   = ""
	app       = cli.NewApp()
)

const (
	blocksFileName = "blocks.txt"
	txsFileName    = "txs.txt"
)

func init() {
	app.Name = filepath.Base(os.Args[0])
	app.Version = fmt.Sprintf("%s - %s", gitCommit, gitDate)
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "ipc",
			Usage: "IPC path of geth node (defailt: geth.ipc)",
			Value: "geth.ipc",
		},
	}
	app.Action = run
}

func onNewHeads(header *types.Header) {
	blockTime := header.Time * 1000
	receiveTime := time.Now().UnixMilli()
	latency := receiveTime - int64(blockTime)
	fmt.Printf("New block %d, timestamp %d, receiveTime: %d, latency: %d \n", header.Number.Uint64(), blockTime, receiveTime, latency)
	file, err := os.OpenFile(blocksFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer file.Close()
	text := fmt.Sprintf("%d %d %d\n", header.Number.Uint64(), header.Time*1000, time.Now().UnixMilli())
	if _, err = file.WriteString(text); err != nil {
		panic(err)
	}
}

func onNewPendingTx(txHash common.Hash) {
	file, err := os.OpenFile(txsFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	text := fmt.Sprintf("%s %d\n", txHash, time.Now().UnixMilli())
	if _, err = file.WriteString(text); err != nil {
		panic(err)
	}
}

func run(ctx *cli.Context) {
	ipcPath := ctx.GlobalString("ipc")
	listener := core.NewNodeListener(ipcPath)
	listener.OnNewHead = onNewHeads
	listener.OnNewPendingTx = onNewPendingTx
	fmt.Println("reporter is running...")
	listener.Start()
}

func main() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
