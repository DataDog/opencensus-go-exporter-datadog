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

	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const (
	defaultEndpoint = "localhost:8125"
)

// Exporter exports stats to Datadog.
type Exporter struct {
	statsExporter *statsExporter
}

// ExportView exports to Datadog if view data has one or more rows.
func (e *Exporter) ExportView(vd *view.Data) {
	if len(vd.Rows) == 0 {
		return
	}
	e.statsExporter.addViewData(vd)
}

// Options contains options for configuring the exporter.
type Options struct {
	Namespace string          // Namespace specifies the namespace to which metrics are appended.
	StatsAddr string          // Endpoint for DogStatsD
	OnError   func(err error) // OnError will be called in the case of an error while uploading the stats.
	Tags      []string        // Tags specifies a set of global tags to attach to each metric.

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
	return &Exporter{
		statsExporter: newStatsExporter(o),
	}
}

// regex pattern
var reg = regexp.MustCompile("[^a-zA-Z0-9]+")

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
