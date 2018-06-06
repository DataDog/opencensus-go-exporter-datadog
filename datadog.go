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

func (o *Options) getEndpoint() string {
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
	endpoint := o.getEndpoint()

	client, err := statsd.New(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	collector := &collector{
		opts:     o,
		viewData: make(map[string]*view.Data),
		client:   client,
	}

	e := &Exporter{
		opts:      o,
		collector: collector,
	}
	return e, nil
}

// ExportView exports to Datadog if view data has one or more rows.
func (e *Exporter) ExportView(vd *view.Data) {
	if len(vd.Rows) == 0 {
		return
	}
	e.collector.addViewData(vd)
}

func sanitizeString(str string) string {
	// Replace all non-alphanumerical characters to underscore
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	return reg.ReplaceAllString(str, "_")
}

func sanitizeMetricName(namespace string, v *view.View) string {
	if namespace != "" {
		namespace = strings.Replace(namespace, " ", "", -1)
		return sanitizeString(namespace) + "." + sanitizeString(v.Name)
	}
	return sanitizeString(v.Name)
}

func viewSignature(namespace string, v *view.View) string {
	var buf bytes.Buffer
	buf.WriteString(sanitizeMetricName(namespace, v))
	for _, k := range v.TagKeys {
		buf.WriteString("_" + k.Name())
	}
	return buf.String()
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
