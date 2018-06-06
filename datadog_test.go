// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.
package datadog

import (
	// "bytes"
	// "errors"
	//"fmt"

	// "sync"

	"context"
	"testing"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	//"github.com/DataDog/datadog-go/statsd"
)

var measureCount = stats.Int64("fooCount", "bytes", stats.UnitBytes)
var testTags []tag.Tag

func newView(agg *view.Aggregation) *view.View {
	return &view.View{
		Name:        "fooCount",
		Description: "fooDesc",
		Measure:     measureCount,
		Aggregation: agg,
	}
}

func customNewView(measureName string, tags []tag.Tag, agg *view.Aggregation) *view.View {
	return &view.View{
		Name:        measureName,
		Description: "fooDesc",
		Measure:     measureCount,
		Aggregation: agg,
	}
}
func TestExportView(t *testing.T) {
	key, _ := tag.NewKey("datadog")
	tags := append(testTags, tag.Tag{Key: key, Value: "opencensus"})

	exporter, err := newExporter(Options{})
	if err != nil {
		t.Fatalf("Failed to create datadog exporter: %v", err)
	}

	view.RegisterExporter(exporter)
	reportPeriod := time.Millisecond
	view.SetReportingPeriod(reportPeriod)

	vd := &view.Data{
		View: customNewView("fooCount", tags, view.Count()),
	}
	if err := view.Register(vd.View); err != nil {
		t.Fatalf("Register error occurred: %v", err)
	}
	defer view.Unregister(vd.View)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.collector.viewData["fooCount"].View
	if actual != vd.View {
		t.Errorf("Expected: %v, Got: %v", vd, actual)
	}
}

func TestSanitizeString(t *testing.T) {
	str1 := sanitizeString("data-234_123!doge")
	sanStr1 := "data_234_123_doge"
	if str1 != sanStr1 {
		t.Errorf("Expected: %v, Got: %v", str1, sanStr1)
	}

	str2 := sanitizeString("hello!good@morn#ing$test%")
	sanStr2 := "hello_good_morn_ing_test_"
	if str2 != sanStr2 {
		t.Errorf("Expected: %v, Got: %v", str1, sanStr1)
	}
}

func TestSanitizeMetricName(t *testing.T) {
	namespace1 := "opencensus"
	vd := &view.Data{
		View: customNewView("fooGauge", testTags, view.Count()),
	}

	res := sanitizeMetricName(namespace1, vd.View)
	exp := "opencensus.fooGauge"
	if res != exp {
		t.Errorf("Expected: %v, Got: %v", exp, res)
	}

	namespace2 := "data!doge"
	vd2 := &view.Data{
		View: customNewView("bar-Sum", testTags, view.Sum()),
	}
	exp2 := "data_doge.bar_Sum"
	res2 := sanitizeMetricName(namespace2, vd2.View)

	if res2 != exp2 {
		t.Errorf("Expected: %v, Got: %v", exp2, res2)
	}
}

func TestSignature(t *testing.T) {
	key, _ := tag.NewKey("datadog")
	tags := append(testTags, tag.Tag{Key: key, Value: "opencensus"})

	namespace3 := "opencensus"
	vd3 := &view.Data{
		View: customNewView("fooGauge", tags, view.Count()),
	}

	res3 := viewSignature(namespace3, vd3.View)
	exp3 := "opencensus.fooGauge"
	if res3 != exp3 {
		t.Errorf("Expected: %v, Got: %v", exp3, res3)
	}
}
