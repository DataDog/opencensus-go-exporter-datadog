// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/DataDog/datadog-go/statsd"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type testStatsExporter struct{ *Exporter }

func (e *testStatsExporter) view(name string) *view.View {
	e.statsExporter.mu.Lock()
	defer e.statsExporter.mu.Unlock()
	return e.Exporter.statsExporter.viewData[name].View
}

func testExporter(opts Options) (*testStatsExporter, error) {
	e, err := NewExporter(opts)
	if err != nil {
		return nil, err
	}
	view.RegisterExporter(e)
	view.SetReportingPeriod(time.Millisecond)
	return &testStatsExporter{e}, nil
}

func createServer(t *testing.T, addr string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	return server, nil
}

func TestAddViewData(t *testing.T) {
	exporter, err := testExporter(Options{Namespace: "hello", Tags: []string{"test:optionalTag"}})
	if err != nil {
		t.Error(err)
	}
	expected := &view.Data{
		View: newView(view.Count()),
	}
	exporter.statsExporter.addViewData(expected)
	actual := exporter.statsExporter.viewData["hello.fooCount"]
	if actual != expected {
		t.Errorf("Expected: %v, Got: %v", expected, actual)
	}
}

func TestUDPExportError(t *testing.T) {
	_, err := testExporter(Options{
		StatsAddr: "invalid_address",
	})
	if err == nil {
		t.Errorf("Expected an error")
	}
}

func TestUDSExportError(t *testing.T) {
	var expected error

	exporter, err := testExporter(Options{
		StatsAddr: "unix:///invalid.socket", // Ideally we wwouln't hit the filesystem.
		OnError: func(err error) {
			expected = err
		},
	})
	if err != nil {
		t.Error(err)
	}

	data := &view.Data{
		View: newView(view.Count()),
		Rows: []*view.Row{
			{
				Tags: []tag.Tag{},
				Data: &view.CountData{},
			},
		},
	}
	exporter.statsExporter.addViewData(data)

	if expected == nil {
		t.Errorf("Expected an error")
	}
}
func TestDistributionData(t *testing.T) {
	addr := "localhost:1201"
	server, _ := createServer(t, addr)

	defer server.Close()

	exporterCount, _ := testExporter(Options{
		StatsAddr: addr,
	})

	exporterNoCount, _ := testExporter(Options{
		StatsAddr:              addr,
		DisableCountPerBuckets: true,
	})

	client, _ := statsd.NewBuffered(addr, 100)

	exporterCount.client = client
	exporterNoCount.client = client

	data := &view.Data{
		View: newView(view.Count()),
		Rows: []*view.Row{
			{
				Tags: []tag.Tag{},
				Data: &view.DistributionData{
					CountPerBucket:  []int64{0, 2, 3},
					Min:             1,
					Max:             5,
					Mean:            3,
					SumOfSquaredDev: 10,
					Count:           15,
				},
			},
		},
	}

	t.Run("Test DistributionData", func(t *testing.T) {
		exporterCount.statsExporter.addViewData(data)

		buffer := make([]byte, 4096)
		n, _ := io.ReadAtLeast(server, buffer, 1)
		result := string(buffer[:n])

		expectedResult := []string{
			`fooCount.avg:3.000000|g`,
			`fooCount.count:15.000000|g`,
			`fooCount.count_per_bucket:0.000000|g|#bucket_idx:0`,
			`fooCount.count_per_bucket:2.000000|g|#bucket_idx:1`,
			`fooCount.count_per_bucket:3.000000|g|#bucket_idx:2`,
			`fooCount.max:5.000000|g`,
			`fooCount.min:1.000000|g`,
			`fooCount.squared_dev_sum:10.000000|g`,
		}

		results := strings.Split(result, "\n")
		sort.Strings(results)
		for i, res := range results {
			if res != expectedResult[i] {
				t.Errorf("Got `%s`, expected `%s`", res, expectedResult[i])
			}
		}
	})

	t.Run("Test DistributionData with DisableCountPerBuckets true", func(t *testing.T) {
		exporterNoCount.statsExporter.addViewData(data)

		buffer := make([]byte, 4096)
		n, _ := io.ReadAtLeast(server, buffer, 1)
		result := string(buffer[:n])

		expectedResult := []string{
			`fooCount.avg:3.000000|g`,
			`fooCount.count:15.000000|g`,
			`fooCount.max:5.000000|g`,
			`fooCount.min:1.000000|g`,
			`fooCount.squared_dev_sum:10.000000|g`,
		}

		results := strings.Split(result, "\n")
		sort.Strings(results)
		for i, res := range results {
			if res != expectedResult[i] {
				t.Errorf("Got `%s`, expected `%s`", res, expectedResult[i])
			}
		}
	})
}

func TestNilAggregation(t *testing.T) {
	exporter, err := testExporter(Options{})
	if err != nil {
		t.Error(err)
	}
	noneAgg := &view.Aggregation{
		Type:    view.AggTypeNone,
		Buckets: []float64{1},
	}
	v := newCustomView("fooNone", noneAgg, testTags, measureCount)
	row := &view.Row{
		Tags: []tag.Tag{},
	}
	actual := exporter.statsExporter.submitMetric(v, row, "fooNone")
	if actual == nil {
		t.Errorf("Expected: %v, Got: %v", fmt.Errorf("aggregation *view.Aggregation is not supported"), actual)
	}
}
