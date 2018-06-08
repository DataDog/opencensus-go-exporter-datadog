// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

// Package datadog contains a Datadog exporter.
package datadog

import (
	"log"
	"regexp"
	"strings"

	"github.com/DataDog/datadog-go/statsd"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const (
	defaultHost   = "localhost"
	defaultPort   = "8125"
	opencensusTag = "source:Opencensus"
)

var (
	exporter *Exporter
	tags     = []string{opencensusTag}
	reg      = regexp.MustCompile("[^a-zA-Z0-9]+")
)

// Exporter exports stats to Datadog.
type Exporter struct {
	opts      Options
	collector *collector
}

// ExportView exports to Datadog if view data has one or more rows.
func (e *Exporter) ExportView(vd *view.Data) {
	if len(vd.Rows) == 0 {
		return
	}
	e.collector.addViewData(vd)
}

// Options contains options for configuring the exporter.
type Options struct {
	// Namespace specifies the namespace to which metrics are appended.
	Namespace string

	// Host for DogStatsD connection
	Host string

	// Port for DogStatsD connection
	Port string

	// OnError will be called in the case of an error while uploading the stats.
	// If not set, errors are simply logged.
	OnError func(err error)

	// Tags specifies a set of global tags to attach to each metric.
	Tags []string
}

// getEndpoint determines the address for the statsD client
func (o *Options) getEndpoint() string {
	host, port := o.Host, o.Port
	if host == "" || port == "" {
		host, port = defaultHost, defaultPort
	}
	endpoint := host + ":" + port
	return endpoint
}

func (o *Options) onError(err error) {
	if o.OnError != nil {
		o.OnError(err)
	} else {
		log.Printf("Failed to export to Datadog: %v\n", err)
	}
}

// NewExporter returns an exporter that exports stats to Datadog
func NewExporter(o Options) *Exporter {
	exporter = newExporter(o)
	if exporter == nil {
		log.Fatalf("Exporter not initialized")
	}
	return exporter
}

func newExporter(o Options) *Exporter {
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
	return e
}

// sanitizeString replaces all non-alphanumerical characters to underscore
func sanitizeString(str string) string {
	return reg.ReplaceAllString(str, "_")
}

// sanitizeMetricName formats the custom namespace and view name to
// Datadog's metric naming convention
func sanitizeMetricName(namespace string, v *view.View) string {
	if namespace != "" {
		namespace = strings.Replace(namespace, " ", "", -1)
		return sanitizeString(namespace) + "." + sanitizeString(v.Name)
	}
	return sanitizeString(v.Name)
}

// viewSignature creates the view signature with custom namespace
func viewSignature(namespace string, v *view.View) string {
	var buf strings.Builder
	buf.WriteString(sanitizeMetricName(namespace, v))
	for _, k := range v.TagKeys {
		buf.WriteString("_" + k.Name())
	}
	return buf.String()
}

// tagMetrics concatenates user input custom tags with row tags
func tagMetrics(rowTags []tag.Tag, customTags []string) []string {
	var finaltag []string
	for key := range rowTags {
		finaltag = append(customTags,
			rowTags[key].Key.Name()+":"+rowTags[key].Value)
	}
	return finaltag
}
