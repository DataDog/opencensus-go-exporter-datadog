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
	//"github.com/DataDog/datadog-go/statsd"
)

func newView(measureName string, agg *view.Aggregation) *view.View {
	measure := stats.Int64(measureName, "bytes", stats.UnitBytes)
	return &view.View{
		Name:        "test_measure",
		Description: "foo",
		Measure:     measure,
		Aggregation: agg,
	}
}

func TestHistograms(t *testing.T) {
	exporter, err := newExporter(Options{})
	if err != nil {
		t.Fatalf("Failed to create datadog exporter: %v", err)
	}

	view.RegisterExporter(exporter)
	reportPeriod := time.Millisecond
	view.SetReportingPeriod(reportPeriod)

	m := stats.Float64("tests/histo", "histogram test", stats.UnitDimensionless)
	v := &view.View{
		Name:        "test_histo",
		Description: "testing histogram",
		Measure:     m,
		Aggregation: view.Distribution(1, 5, 5, 5, 5, 10, 20, 50, 100, 250),
	}

	if err := view.Register(v); err != nil {
		t.Fatalf("Register error occurred: %v", err)
	}
	defer view.Unregister(v)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	val := []float64{0.25, 245.67, 12, 1.45, 199.9, 7.69, 187.12}

	ctx := context.Background()
	ms := make([]stats.Measurement, len(val))
	for _, value := range val {
		m := m.M(value)
		ms = append(ms, m)
	}
	stats.Record(ctx, ms...)

	// Wait for recording to start
	<-time.After(10 * reportPeriod)

}
