// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"fmt"
	"io"
	"math"
	"math/rand"
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

func listenUDP(addr string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", udpAddr)
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
	conn, err := listenUDP("localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	addr := conn.LocalAddr().String()

	client, err := statsd.NewBuffered(addr, 100)
	if err != nil {
		t.Fatal(err)
	}

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

	testCases := map[string]struct {
		Options         Options
		ExpectedResults []string
	}{
		"ok": {
			Options{
				StatsAddr: addr,
			},
			[]string{
				`fooCount.avg:3.000000|g`,
				`fooCount.count:15.000000|g`,
				`fooCount.count_per_bucket:0.000000|g|#bucket_idx:0`,
				`fooCount.count_per_bucket:2.000000|g|#bucket_idx:1`,
				`fooCount.count_per_bucket:3.000000|g|#bucket_idx:2`,
				`fooCount.max:5.000000|g`,
				`fooCount.min:1.000000|g`,
				`fooCount.squared_dev_sum:10.000000|g`,
			},
		},
		"disabled": {
			Options{
				StatsAddr:              addr,
				DisableCountPerBuckets: true,
			},
			[]string{
				`fooCount.avg:3.000000|g`,
				`fooCount.count:15.000000|g`,
				`fooCount.max:5.000000|g`,
				`fooCount.min:1.000000|g`,
				`fooCount.squared_dev_sum:10.000000|g`,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			exporter, err := testExporter(tc.Options)
			if err != nil {
				t.Fatal(err)
			}
			exporter.client = client
			exporter.statsExporter.addViewData(data)

			buffer := make([]byte, 4096)
			n, err := io.ReadAtLeast(conn, buffer, 1)
			if err != nil {
				t.Fatal(err)
			}
			result := string(buffer[:n])

			results := strings.Split(result, "\n")
			sort.Strings(results)
			for i, res := range results {
				if res != tc.ExpectedResults[i] {
					t.Errorf("Got `%s`, expected `%s`", res, tc.ExpectedResults[i])
				}
			}
		})
	}
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

func Test_calculatePercentile(t *testing.T) {
	var buckets []float64
	for i := 0.01; i < 100; i += 0.01 {
		buckets = append(buckets, i)
	}

	var countsPerBucket []int64
	for i := int64(0); i <= int64(len(buckets)); i += 1 {
		countsPerBucket = append(countsPerBucket, 1)
	}

	testCases := []struct {
		expected        float64
		percentile      float64
		buckets         []float64
		countsPerBucket []int64
	}{
		{
			0,
			0,
			buckets,
			countsPerBucket,
		},
		{
			10,
			0.1,
			buckets,
			countsPerBucket,
		},
		{
			50,
			0.5,
			buckets,
			countsPerBucket,
		},
		{
			70,
			0.70,
			buckets,
			countsPerBucket,
		},
		{
			81,
			0.81,
			buckets,
			countsPerBucket,
		},
		{
			100,
			1.0,
			buckets,
			countsPerBucket,
		},
		{
			99,
			0.99,
			buckets,
			countsPerBucket,
		},
		{
			99.9,
			0.999,
			buckets,
			countsPerBucket,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v", tc.percentile), func(t *testing.T) {
			got := calculatePercentile(tc.percentile, tc.buckets, tc.countsPerBucket)

			if math.Abs(tc.expected-got) > 0.01 {
				t.Errorf("Expected: %v to be within 0.1 of %v", tc.expected, got)
			}
		})

	}
}

func Test_calculatePercentileStandard(t *testing.T) {
	var buckets []float64
	for i := float64(-100); i < 100; i += 0.1 {
		buckets = append(buckets, i)
	}

	// Calculate a normal distribution with a standard deviation of 1.
	normalDistribution := calculateNormalDistribution(buckets, 0, 1)

	// The following tests can be confirmed using the Cumulative Standard Normal table (https://en.wikipedia.org/wiki/Standard_normal_table#Cumulative).
	tsts := []struct {
		expected        float64
		percentile      float64
		buckets         []float64
		countsPerBucket []int64
	}{
		{
			0,
			0.5,
			buckets,
			normalDistribution,
		},
		{
			0.69,
			0.75,
			buckets,
			normalDistribution,
		},
		{
			1.67,
			0.95,
			buckets,
			normalDistribution,
		},
		{
			2.33,
			0.99,
			buckets,
			normalDistribution,
		},
	}

	for _, tst := range tsts {
		t.Run(fmt.Sprintf("%v", tst.percentile), func(t *testing.T) {
			got := calculatePercentile(tst.percentile, tst.buckets, tst.countsPerBucket)

			if math.Abs(tst.expected-got) > 0.1 {
				t.Errorf("Expected: %v to be within 0.1 of %v", tst.expected, got)
			}
		})

	}
}

// Given a seed and a set of latency buckets, uses rand.NormFloat64 to generate a normal distribution
func calculateNormalDistribution(buckets []float64, seed int64, standardDeviation float64) []int64 {
	r := rand.New(rand.NewSource(seed))

	normalDistribution := make([]int64, len(buckets))
	for n := 0; n < 1e6; n++ {
		rnd := r.NormFloat64() * standardDeviation
		var previousBucket float64
		for bidx, bucket := range buckets {
			if rnd > previousBucket && rnd <= bucket {
				normalDistribution[bidx]++
				break
			}
			previousBucket = bucket
		}
	}
	return normalDistribution
}
