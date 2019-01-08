// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"fmt"
	"log"
	"regexp"
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

var beginsWithLetter, _ = regexp.Compile("^[^a-zA-Z]")

// collector implements statsd.Client
type statsExporter struct {
	opts     Options
	client   *statsd.Client
	mu       sync.Mutex // mu guards viewData
	viewData map[string]*view.Data
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

	return &statsExporter{
		opts:     o,
		viewData: make(map[string]*view.Data),
		client:   client,
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

// Metrics should adhere to Datadog rules, log a message if they aren't met.
// https://docs.datadoghq.com/developers/metrics/#naming-metrics
func verifyMetricName(metricName string) bool {

	// Must be less than 200 chars
	if len(metricName) > 200 {
		log.Printf("Metric %s is longer than 200 characters, it will be truncated within Datadog", metricName)
		return false
	}
	// Must start with a letter
	if beginsWithLetter.MatchString(metricName) {
		log.Printf("Metric %s begins with non alphabetic character, it will be converted within Datadog", metricName)
		return false
	}
	return true
}

func (s *statsExporter) submitMetric(v *view.View, row *view.Row, metricName string) error {
	var err error
	const rate = float64(1)
	verifyMetricName(metricName)
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

		for name, value := range metrics {
			err = client.Gauge(metricName+"."+name, value, opt.tagMetrics(row.Tags, tags), rate)
		}

		for x := range data.CountPerBucket {
			addlTags := []string{"bucket_idx:" + fmt.Sprint(x)}
			err = client.Gauge(metricName+".count_per_bucket", float64(data.CountPerBucket[x]), opt.tagMetrics(row.Tags, addlTags), rate)
		}
		return err
	default:
		return fmt.Errorf("aggregation %T is not supported", v.Aggregation)
	}
}
