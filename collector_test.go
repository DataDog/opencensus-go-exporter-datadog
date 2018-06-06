// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.
package datadog

import (
	"testing"
	"time"

	"go.opencensus.io/stats/view"
)

func TestAddViewData(t *testing.T) {
	exporter, err := newExporter(Options{})
	if err != nil {
		t.Fatalf("Failed to create datadog exporter: %v", err)
	}

	view.RegisterExporter(exporter)
	reportPeriod := time.Millisecond
	view.SetReportingPeriod(reportPeriod)

	expected := &view.Data{
		View: newView("fooCount", view.Count()),
	}
	exporter.collector.addViewData(expected)
	actual := exporter.collector.viewData["fooCount"]

	if actual != expected {
		t.Errorf("Expected: %v, Got: %v", expected, actual)
	}
}

func TestMetricSubmitToStatsd(t *testing.T) {
	exporter, err := newExporter(Options{})
	if err != nil {
		t.Fatalf("Failed to create datadog exporter: %v", err)
	}

	view.RegisterExporter(exporter)
	reportPeriod := time.Millisecond
	view.SetReportingPeriod(reportPeriod)
	vd := &view.Data{
		View: newView("fooCount", view.Count()),
	}
	sig := viewSignature("", vd.View)

	for _, row := range vd.Rows {
		exporter.collector.submitMetric(vd.View, row, sig)
	}
	t.Logf("testtttttttt")
}
