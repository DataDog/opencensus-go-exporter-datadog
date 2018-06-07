// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.
package datadog

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	measureCount = stats.Int64("fooCount", "testing count metrics", stats.UnitBytes)
	measureSum   = stats.Int64("fooSum", "testing sum metrics", stats.UnitBytes)
	measureLast  = stats.Int64("fooLast", "testing LastValueData metrics", stats.UnitBytes)
	measureDist  = stats.Int64("fooHisto", "testing histogram metrics", stats.UnitDimensionless)
	testTags     []tag.Tag
)

func newView(agg *view.Aggregation) *view.View {
	return &view.View{
		Name:        "fooCount",
		Description: "fooDesc",
		Measure:     measureCount,
		Aggregation: agg,
	}
}

func customNewView(measureName string, agg *view.Aggregation, measure *stats.Int64Measure) *view.View {
	return &view.View{
		Name:        measureName,
		Description: "fooDesc",
		Measure:     measureCount,
		Aggregation: agg,
	}
}
func TestExportView(t *testing.T) {
	exporter := newExporter(Options{})

	view.RegisterExporter(exporter)
	reportPeriod := time.Millisecond
	view.SetReportingPeriod(reportPeriod)

	vd := &view.Data{
		View: customNewView("fooCount", view.Count(), measureCount),
	}
	if err := view.Register(vd.View); err != nil {
		t.Fatalf("Register error occurred: %v\n", err)
	}
	defer view.Unregister(vd.View)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.collector.viewData["fooCount"].View
	if actual != vd.View {
		t.Errorf("Expected: %v, Got: %v\n", vd, actual)
	}
}

func TestSanitizeString(t *testing.T) {
	str1 := sanitizeString("data-234_123!doge")
	sanStr1 := "data_234_123_doge"
	if str1 != sanStr1 {
		t.Errorf("Expected: %v, Got: %v\n", str1, sanStr1)
	}

	str2 := sanitizeString("hello!good@morn#ing$test%")
	sanStr2 := "hello_good_morn_ing_test_"
	if str2 != sanStr2 {
		t.Errorf("Expected: %v, Got: %v\n", str1, sanStr1)
	}
}

func TestSanitizeMetricName(t *testing.T) {
	namespace1 := "opencensus"
	vd := &view.Data{
		View: customNewView("fooGauge", view.Count(), measureCount),
	}

	res := sanitizeMetricName(namespace1, vd.View)
	exp := "opencensus.fooGauge"
	if res != exp {
		t.Errorf("Expected: %v, Got: %v\n", exp, res)
	}

	namespace2 := "data!doge"
	vd2 := &view.Data{
		View: customNewView("bar-Sum", view.Sum(), measureSum),
	}
	exp2 := "data_doge.bar_Sum"
	res2 := sanitizeMetricName(namespace2, vd2.View)

	if res2 != exp2 {
		t.Errorf("Expected: %v, Got: %v\n", exp2, res2)
	}
}

func TestSignature(t *testing.T) {
	namespace := "opencensus"
	vd := &view.Data{
		View: customNewView("fooGauge", view.Count(), measureCount),
	}

	res := viewSignature(namespace, vd.View)
	exp := "opencensus.fooGauge"
	if res != exp {
		t.Errorf("Expected: %v, Got: %v\n", exp, res)
	}
}

func TestTagMetrics(t *testing.T) {
	key, _ := tag.NewKey("testTags")
	tags := append(testTags, tag.Tag{Key: key, Value: "Metrics"})
	customTag := []string{"program_name:main"}
	result := tagMetrics(tags, customTag)
	expected := []string{"program_name:main", "testTags:Metrics"}

	if !(expected == nil && result == nil) && len(expected) == len(result) {
		for i := range result {
			if result[i] != expected[i] {
				t.Errorf("Expected: %v, Got: %v\n", result, expected)
			}
		}
	}
}

func TestOnErrorNil(t *testing.T) {
	opt := &Options{}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	opt.onError(nil)
	result := buf.String()
	expected := "Failed to export to Datadog: <nil>"
	if !strings.Contains(result, expected) {
		t.Errorf("Expected: %v, Got: %v\n", result, expected)
	}
}

func TestOnError(t *testing.T) {
	expected := "Testing error"
	testError := errors.New(expected)
	opt := &Options{}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	opt.onError(testError)
	result := buf.String()
	if !strings.Contains(result, expected) {
		t.Errorf("Expected: %v, Got: %v\n", result, expected)
	}
}

func TestNewExporter(t *testing.T) {
	opt := Options{}
	exp := NewExporter(opt)
	actual := exp.collector.client
	if actual == nil {
		t.Errorf("Expected DogstatsD Client got: %v\n", nil)
	}
}

func TestCountData(t *testing.T) {
	exporter := newExporter(Options{})

	view.RegisterExporter(exporter)
	reportPeriod := time.Millisecond
	view.SetReportingPeriod(reportPeriod)

	vd := &view.Data{
		View: customNewView("fooCount", view.Count(), measureCount),
	}
	if err := view.Register(vd.View); err != nil {
		t.Fatalf("Register error occurred: %v\n", err)
	}
	defer view.Unregister(vd.View)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.collector.viewData["fooCount"].View
	if actual != vd.View {
		t.Errorf("Expected: %v, Got: %v\n", vd, actual)
	}
}

func TestSumData(t *testing.T) {
	exporter := newExporter(Options{})

	view.RegisterExporter(exporter)
	reportPeriod := time.Millisecond
	view.SetReportingPeriod(reportPeriod)

	vd := &view.Data{
		View: customNewView("fooSum", view.Sum(), measureSum),
	}
	if err := view.Register(vd.View); err != nil {
		t.Fatalf("Register error occurred: %v\n", err)
	}
	defer view.Unregister(vd.View)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.collector.viewData["fooSum"].View
	if actual != vd.View {
		t.Errorf("Expected: %v, Got: %v\n", vd, actual)
	}
}

func TestLastValueData(t *testing.T) {
	exporter := newExporter(Options{})

	view.RegisterExporter(exporter)
	reportPeriod := time.Millisecond
	view.SetReportingPeriod(reportPeriod)

	vd := &view.Data{
		View: customNewView("fooLast", view.LastValue(), measureLast),
	}
	if err := view.Register(vd.View); err != nil {
		t.Fatalf("Register error occurred: %v\n", err)
	}
	defer view.Unregister(vd.View)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.collector.viewData["fooLast"].View
	if actual != vd.View {
		t.Errorf("Expected: %v, Got: %v\n", vd, actual)
	}
}
func TestHistogram(t *testing.T) {
	exporter := newExporter(Options{})

	view.RegisterExporter(exporter)
	reportPeriod := time.Millisecond
	view.SetReportingPeriod(reportPeriod)

	vd := &view.Data{
		View: customNewView("fooHisto", view.Distribution(), measureDist),
	}
	if err := view.Register(vd.View); err != nil {
		t.Fatalf("Register error occurred: %v\n", err)
	}
	defer view.Unregister(vd.View)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.collector.viewData["fooHisto"].View
	if actual != vd.View {
		t.Errorf("Expected: %v, Got: %v\n", vd, actual)
	}
}
