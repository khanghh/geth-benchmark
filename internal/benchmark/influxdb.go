package benchmark

import (
	"context"
	"fmt"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type InfluxDBReporter struct {
	Tags        map[string]string
	client      influxdb2.Client
	writeAPI    api.WriteAPIBlocking
	workBuckets map[uint64]uint64
}

func (r *InfluxDBReporter) collectResult(work *workResult) {
	bucketId := uint64(work.Elapsed / time.Millisecond / 100)
	if _, ok := r.workBuckets[bucketId]; !ok {
		r.workBuckets[bucketId] = 0
	}
	r.workBuckets[bucketId] += 1
}

func (r *InfluxDBReporter) publishMetrics(ctx context.Context, result *BenchmarkResult) {
	now := time.Now()
	pts := []*write.Point{}
	measurementName := "work/duration/histogram"
	for bucketId, count := range r.workBuckets {
		fields := map[string]interface{}{
			fmt.Sprintf("%d", bucketId*100): count,
		}
		pts = append(pts, influxdb2.NewPoint(measurementName, r.Tags, fields, now))
	}
	for _, pt := range pts {
		err := r.writeAPI.WritePoint(ctx, pt)
		if err != nil {
			log.Printf("Failed to publish metrics to influxdb, measurement: %s, error: %v", pt.Name(), err)
			return
		}
	}
}

func NewInfluxDBReporter(serverURL, authToken, org, bucket string, tags map[string]string) *InfluxDBReporter {
	client := influxdb2.NewClient(serverURL, authToken)
	writeAPI := client.WriteAPIBlocking(org, bucket)
	return &InfluxDBReporter{
		client:      client,
		workBuckets: map[uint64]uint64{},
		writeAPI:    writeAPI,
		Tags:        tags,
	}
}
