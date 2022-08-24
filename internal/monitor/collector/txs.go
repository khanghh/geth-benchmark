package collector

import (
	"geth-benchmark/internal/monitor"

	"github.com/ethereum/go-ethereum/metrics"
)

type TPSGauge struct {
	tpsGauge metrics.Gauge
}

func (g TPSGauge) OnEthEvent(eventType monitor.EthEvent, data interface{}) {

}

func (g *TPSGauge) Setup(r metrics.Registry) {
	gau := metrics.NewRegisteredGauge("ethmetrics/tps/10", r)
	g.tpsGauge = gau
}

func NewTPSGauge() *TPSGauge {
	return &TPSGauge{}
}
