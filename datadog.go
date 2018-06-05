// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

// Package datadog contains a Datadog exporter.
//
// This exporter is currently work in progress
package datadog

// import "go.opencensus.io/exporter/datadog"

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/DataDog/datadog-go/statsd"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Exporter exports stats to Datadog
type Exporter struct {
	opts      Options
	collector *collector
	client    *statsd.Client
}

// Options contains options for configuring the exporter
type Options struct {
	// Namespace to prepend to all metrics
	Namespace string

	// Host for DogStatsD connection
	Host string

	// Port for DogStatsD connection
	Port string

	// OnError is the hook to be called when there is
	// an error occurred when uploading the stats data.
	// If no custom hook is set, errors are logged.
	// Optional.
	OnError func(err error)

	// Tags are global tags added to each metric
	Tags []string
}

const (
	defaultHost   = "localhost"
	defaultPort   = "8125"
	opencensusTag = "source:Opencensus"
)

var (
	newExporterOnce      sync.Once
	errSingletonExporter = errors.New("expecting only one exporter per instance")
	err                  = errSingletonExporter
	exporter             *Exporter
	tags                 = []string{opencensusTag}
)

// NewExporter returns an exporter that exports stats to Datadog
func NewExporter(o Options) (*Exporter, error) {
	newExporterOnce.Do(func() {
		exporter, err = newExporter(o)
	})
	return exporter, err
}

func getEndpoint(o Options) string {
	var endpoint string
	host, port := o.Host, o.Port
	if host == "" || port == "" {
		host, port = defaultHost, defaultPort
	}
	endpoint = host + ":" + port
	log.Printf("Endpoint set at: %v", endpoint)
	return endpoint
}

func newExporter(o Options) (*Exporter, error) {

	endpoint := getEndpoint(o)

	client, err := statsd.New(endpoint)
	if err != nil {
		log.Fatal(err)
	}
	collector := newCollector(o)

	e := &Exporter{
		opts:      o,
		collector: collector,
		client:    client,
	}
	return e, nil
}

func newCollector(o Options) *collector {
	return &collector{
		opts:     o,
		viewData: make(map[string]*view.Data),
	}
}

// ExportView exports to Datadog if view data has one or more rows.
func (e *Exporter) ExportView(vd *view.Data) {
	if len(vd.Rows) == 0 {
		return
	}
	e.collector.addViewData(vd, e.client)
}

func viewName(namespace string, v *view.View) string {
	if namespace != "" {
		namespace = strings.Replace(namespace, " ", "", -1)
	}
	names := []string{namespace, v.Name}
	// Replace all non-alphanumerical characters to underscore
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	for x := range names {
		names[x] = reg.ReplaceAllString(names[x], "_")
	}
	return names[0] + "." + names[1]
}

func viewSignature(namespace string, v *view.View) string {
	var buf bytes.Buffer
	buf.WriteString(viewName(namespace, v))
	for _, k := range v.TagKeys {
		buf.WriteString("_" + k.Name())
	}
	return buf.String()
}

func (c *collector) addViewData(vd *view.Data, client *statsd.Client) {
	sig := viewSignature(c.opts.Namespace, vd.View)

	c.mu.Lock()
	c.viewData[sig] = vd
	fmt.Println(c.viewData[sig])
	c.mu.Unlock()

	for _, row := range vd.Rows {
		submitMetric(client, vd.View, row, sig)
	}
}

func submitMetric(client *statsd.Client, v *view.View, row *view.Row, metricName string) error {
	rate := 1
	var err error

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

func tagMetrics(t []tag.Tag, ct []string) []string {
	var names, finaltag []string
	for _, tag := range t {
		names = append(names, tag.Key.Name())
	}
	for _, ctag := range ct {
		finaltag = append(names, ctag)
	}
	return finaltag
}

func (o *Options) onError(err error) {
	if o.OnError != nil {
		o.OnError(err)
	} else {
		log.Printf("Failed to export to Datadog: %v\n", err)
	}
}
