package benchmark

const (
	walletDerivePath = "m/44'/60'/0'/0/%d"
)

type TxBenchmark struct {
}

func (b *TxBenchmark) TransferERC20(workerIndex int) error {
	return nil
}

// func fetchNonces() map[common.Address]uint64 {
// accNonces := map[common.Address]uint64{}
// wg := sync.WaitGroup{}
// mtx := sync.Mutex{}
// for i := 0; i < numAccount; i++ {
// 	wg.Add(1)
// 	go func(acc *accounts.Account) {
// 		defer wg.Done()
// 		ctx, cancel := context.WithTimeout(context.Background(), rpcTimeOunt)
// 		defer cancel()
// 		nonce, err := ethClient.PendingNonceAt(ctx, acc.Address)
// 		if err != nil {
// 			log.Fatal("Could not get nonce for account "+acc.Address.Hex(), err)
// 			os.Exit(1)
// 		}
// 		mtx.Lock()
// 		accNonces[acc.Address] = nonce
// 		mtx.Unlock()
// 	}(testAccs[i])
// }
// wg.Wait()
// return accNonces
// }

func (b *TxBenchmark) Prepair() {
}

func (b *TxBenchmark) DoWork(workerIndex int) error {
	return nil
}

func (b *TxBenchmark) OnFinish(roundIndex int, result *BenchmarkResult) {

}

func NewTxBenchmark(rpcUrl string) *TxBenchmark {
	return &TxBenchmark{}
}
