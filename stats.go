// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"fmt"
	"log"
	"sync"

	"github.com/DataDog/datadog-go/statsd"
	"go.opencensus.io/stats/view"
)

// collector implements statsd.Client
type statsExporter struct {
	opts     Options
	mu       sync.Mutex // mu guards viewData
	client   *statsd.Client
	viewData map[string]*view.Data
}

func newStatsExporter(o Options) *statsExporter {
	endpoint := o.StatsAddr
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	client, err := statsd.New(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	return &statsExporter{
		opts:     o,
		viewData: make(map[string]*view.Data),
		client:   client,
	}
}

func (s *statsExporter) addViewData(vd *view.Data) {
	sig := viewSignature(s.opts.Namespace, vd.View)
	s.mu.Lock()
	s.viewData[sig] = vd
	s.mu.Unlock()

	for _, row := range vd.Rows {
		s.submitMetric(vd.View, row, sig)
	}
}

func (s *statsExporter) submitMetric(v *view.View, row *view.Row, metricName string) error {
	var err error
	const rate = float64(1)
	client := s.client
	customTags := s.opts.Tags

	switch data := row.Data.(type) {
	case *view.CountData:
		return client.Gauge(metricName, float64(data.Value), tagMetrics(row.Tags, customTags), rate)

	case *view.SumData:
		return client.Gauge(metricName, float64(data.Value), tagMetrics(row.Tags, customTags), rate)

	case *view.LastValueData:
		return client.Gauge(metricName, float64(data.Value), tagMetrics(row.Tags, customTags), rate)

	case *view.DistributionData:
		var metrics = map[string]float64{
			"min":             data.Min,
			"max":             data.Max,
			"count":           float64(data.Count),
			"avg":             data.Mean,
			"squared_dev_sum": data.SumOfSquaredDev,
		}

		for name, value := range metrics {
			err = client.Gauge(metricName+"."+name, value, tagMetrics(row.Tags, customTags), rate)
		}

		for x := range data.CountPerBucket {
			bucketTags := append(customTags, "bucket_idx"+fmt.Sprint(x))
			err = client.Gauge(metricName+".count_per_bucket", float64(data.CountPerBucket[x]), tagMetrics(row.Tags, bucketTags), rate)
		}
		return err
	default:
		return fmt.Errorf("aggregation %T is not supported", v.Aggregation)
	}
}
