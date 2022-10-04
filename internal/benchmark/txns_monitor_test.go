package benchmark

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSplitTxHashes(t *testing.T) {
	txHashes := []common.Hash{}
	for i := 0; i < 10; i++ {
		hash := [common.HashLength]byte{}
		hash[len(hash)-1] = byte(i)
		txHashes = append(txHashes, hash)
	}
	ret := splitTxHashes(txHashes, 4)
	for _, part := range ret {
		fmt.Println(part)
	}
}
