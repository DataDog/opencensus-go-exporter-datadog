// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/DataDog/datadog-go/statsd"
	"go.opencensus.io/stats/view"
)

const (
	// DefaultStatsAddrUDP specifies the default protocol (UDP) and address
	// for the DogStatsD service.
	DefaultStatsAddrUDP = "localhost:8125"

	// DefaultStatsAddrUDS specifies the default socket address for the
	// DogStatsD service over UDS. Only useful for platforms supporting unix
	// sockets.
	DefaultStatsAddrUDS = "unix:///var/run/datadog/dsd.socket"
)

// collector implements statsd.Client
type statsExporter struct {
	opts        Options
	client      *statsd.Client
	mu          sync.Mutex // mu guards viewData
	viewData    map[string]*view.Data
	percentiles map[string]float64
}

func newStatsExporter(o Options) (*statsExporter, error) {
	endpoint := o.StatsAddr
	if endpoint == "" {
		endpoint = DefaultStatsAddrUDP
	}

	client, err := statsd.New(endpoint)
	if err != nil {
		return nil, err
	}

	percentiles := make(map[string]float64, len(o.HistogramPercentiles))
	for _, percentileStr := range o.HistogramPercentiles {
		percentile, err := strconv.ParseFloat(percentileStr, 64)
		if err != nil {
			return nil, fmt.Errorf("'HistogramPercentiles' must be in float format, got %s", percentileStr)
		}
		if percentile < 0 || percentile > 1 {
			return nil, fmt.Errorf("'HistogramPercentiles' must be between 0 and 1, got %f", percentile)
		}
		percentiles[percentileName(percentile)] = percentile
	}

	return &statsExporter{
		opts:        o,
		viewData:    make(map[string]*view.Data),
		client:      client,
		percentiles: percentiles,
	}, nil
}

func (s *statsExporter) addViewData(vd *view.Data) {
	sig := viewSignature(s.opts.Namespace, vd.View)
	s.mu.Lock()
	s.viewData[sig] = vd
	s.mu.Unlock()

	var lastErr error
	for _, row := range vd.Rows {
		if err := s.submitMetric(vd.View, row, sig); err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		s.opts.onError(lastErr) // Only report last error.
	}
}

func (s *statsExporter) submitMetric(v *view.View, row *view.Row, metricName string) error {
	var err error
	const rate = float64(1)
	client := s.client
	opt := s.opts
	tags := []string{}

	switch data := row.Data.(type) {
	case *view.CountData:
		return client.Gauge(metricName, float64(data.Value), opt.tagMetrics(row.Tags, tags), rate)

	case *view.SumData:
		return client.Gauge(metricName, float64(data.Value), opt.tagMetrics(row.Tags, tags), rate)

	case *view.LastValueData:
		return client.Gauge(metricName, float64(data.Value), opt.tagMetrics(row.Tags, tags), rate)

	case *view.DistributionData:
		var metrics = map[string]float64{
			"min":             data.Min,
			"max":             data.Max,
			"count":           float64(data.Count),
			"avg":             data.Mean,
			"squared_dev_sum": data.SumOfSquaredDev,
		}
		if len(v.Aggregation.Buckets) > 0 {
			for name, percentile := range s.percentiles {
				metrics[name] = calculatePercentile(percentile, v.Aggregation.Buckets, data.CountPerBucket)
			}
		}

		for name, value := range metrics {
			err = client.Gauge(metricName+"."+name, value, opt.tagMetrics(row.Tags, tags), rate)
		}
		if !s.opts.DisableCountPerBuckets {
			for x := range data.CountPerBucket {
				addlTags := []string{"bucket_idx:" + fmt.Sprint(x)}
				err = client.Gauge(metricName+".count_per_bucket", float64(data.CountPerBucket[x]), opt.tagMetrics(row.Tags, addlTags), rate)
			}
		}
		return err
	default:
		return fmt.Errorf("aggregation %T is not supported", v.Aggregation)
	}
}
// calculatePercentile return value for a given percentile using distribution buckets
// percentile: target percentile we want the value to be returned
// buckets: bucket bounds from distribution data
// countPerBucket: number of data point for each bucket from distribution data
//
// Buckets and countPerBucket are used to compute the value for a specific percentile.
// Caveat: the precision of the value return depend on the the granularity of the buckets.
func calculatePercentile(percentile float64, buckets []float64, countPerBucket []int64) float64 {
	// create intermediate cumulative buckets
	cumulativePerBucket := make([]int64, len(countPerBucket))
	var sum int64
	for n, count := range countPerBucket {
		sum += count
		cumulativePerBucket[n] = sum
	}

	// search for the bucket corresponding the percentile
	atBin := int64(math.Floor(percentile * float64(sum)))

	var previousCount int64
	for n, count := range cumulativePerBucket {
		if atBin >= previousCount && atBin <= count && n < len(buckets) {
			return buckets[n]
		}
		previousCount = count
	}
	return buckets[len(buckets)-1]
}

func percentileName(percentile float64) string {
	multiplier := 100
	if percentile > 0.99 {
		multiplier = 1000
	}
	return strconv.Itoa(int(percentile * float64(multiplier))) + "percentile"
}
