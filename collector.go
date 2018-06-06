// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"fmt"
	"sync"

	"github.com/DataDog/datadog-go/statsd"
	"go.opencensus.io/stats/view"
)

// client implements datadog.Client
type collector struct {
	opts Options

	// mu guards all the fields.
	mu sync.Mutex

	// viewData is accumulated and appended on every Export
	// invocation from stats.
	viewData map[string]*view.Data

	client *statsd.Client
}

func (c *collector) addViewData(vd *view.Data) {
	sig := viewSignature(c.opts.Namespace, vd.View)
	c.mu.Lock()
	c.viewData[sig] = vd
	c.mu.Unlock()

	for _, row := range vd.Rows {
		c.submitMetric(vd.View, row, sig)
	}
}

func (c *collector) submitMetric(v *view.View, row *view.Row, metricName string) error {
	rate := 1
	var err error
	client := c.client

	switch data := row.Data.(type) {
	case *view.CountData:
		return client.Gauge(metricName, float64(data.Value), tagMetrics(row.Tags, tags), float64(rate))

	case *view.SumData:
		return client.Gauge(metricName, float64(data.Value), tagMetrics(row.Tags, tags), float64(rate))

	case *view.LastValueData:
		return client.Gauge(metricName, float64(data.Value), tagMetrics(row.Tags, tags), float64(rate))

	case *view.DistributionData:
		var metrics = map[string]float64{
			"min":             data.Min,
			"max":             data.Max,
			"count":           float64(data.Count),
			"avg":             data.Mean,
			"squared_dev_sum": data.SumOfSquaredDev,
		}

		for name, value := range metrics {
			err = client.Gauge(metricName+"."+name, value, tagMetrics(row.Tags, tags), float64(rate))
		}

		for x := range data.CountPerBucket {
			bucketTags := append(tags, "bucket_idx"+fmt.Sprint(x))
			err = client.Gauge(metricName+".count_per_bucket", float64(data.CountPerBucket[x]), tagMetrics(row.Tags, bucketTags), float64(rate))
		}
		return err
	default:
		return fmt.Errorf("aggregation %T is not supported", v.Aggregation)
	}
}
