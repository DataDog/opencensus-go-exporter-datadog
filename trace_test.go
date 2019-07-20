// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadog.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinylib/msgp/msgp"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
)

const (
	// testFlushInterval is the flush interval that will be used for the
	// duration of the tests.
	testFlushInterval = 24 * time.Hour

	// testFlushThreshold is the flush threshold that will be used for the
	// duration of the tests.
	testFlushThreshold = 1e3

	// testInChannelSize is the input channel's buffer size that will be used
	// for the duration of the tests.
	testInChannelSize = 1000
)

func TestMain(m *testing.M) {
	o1, o2, o3 := flushInterval, flushThreshold, inChannelSize
	flushInterval = testFlushInterval
	flushThreshold = testFlushThreshold
	inChannelSize = testInChannelSize

	defer func() {
		flushInterval, flushThreshold, inChannelSize = o1, o2, o3
	}()

	os.Exit(m.Run())
}

func TestTraceExporter(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	t.Run("service", func(t *testing.T) {
		me := newTraceExporter(Options{})
		defer me.stop()
		if me.opts.Service == "" {
			t.Fatal("service should never be empty")
		}
	})

	t.Run("threshold", func(t *testing.T) {
		me := newTestTraceExporter(t)
		defer me.stop()
		span := spanPairs["tags"].oc
		count := 5 // 5 spans should take us overboard
		for i := 0; i < count; i++ {
			me.exportSpan(span)
		}
		time.Sleep(time.Millisecond) // wait for recv
		me.wg.Wait()                 // wait for flush
		flushed := me.payloads()
		eq := equalFunc(t)
		eq(len(flushed), 1)
		eq(len(flushed[0][0]), count)
	})

	t.Run("stop", func(t *testing.T) {
		me := newTestTraceExporter(t)
		me.exportSpan(spanPairs["root"].oc)
		me.stop()

		if len(me.payloads()) != 1 {
			t.Fatalf("expected to flush 1, got %d", len(me.payloads()))
		}
	})

	t.Run("sampler", func(t *testing.T) {
		eq := equalFunc(t)
		me := newTestTraceExporter(t)
		me.exportSpan(spanPairs["server_error_5xx"].oc)
		me.stop()

		// sampler is updated after flush
		eq(me.sampler.rates["service:db.users,env:"], 0.9)
		eq(me.sampler.defaultRate, 0.8)

		// got the sent span
		payload := me.payloads()
		eq(len(payload), 1)
		eq(len(payload[0]), 1)
		eq(len(payload[0][0]), 1)

		// span has sampling priority and rate applied
		span1 := payload[0][0][0]
		p, ok := span1.Metrics[keySamplingPriority]
		if !ok || !(p == ext.PriorityAutoKeep || p == ext.PriorityAutoReject) {
			t.Fatal(p, ok)
		}
		if v := span1.Metrics[keySamplingPriorityRate]; v != 1 {
			t.Fatalf("got %f", v)
		}
	})
}

// testTraceExporter wraps a traceExporter, recording all flushed payloads.
type testTraceExporter struct {
	*traceExporter
	t *testing.T

	mu      sync.RWMutex
	flushed []ddPayload
}

func newTestTraceExporter(t *testing.T) *testTraceExporter {
	te := newTraceExporter(Options{Service: "mock.exporter"})
	me := &testTraceExporter{traceExporter: te, flushed: make([]ddPayload, 0)}
	me.traceExporter.uploadFn = me.uploadFn
	return me
}

// payloads returns all payloads that were uploaded by this exporter.
func (me *testTraceExporter) payloads() []ddPayload {
	me.mu.RLock()
	defer me.mu.RUnlock()
	return me.flushed
}

func (me *testTraceExporter) uploadFn(buf *bytes.Buffer, _ int) (io.ReadCloser, error) {
	var ddp ddPayload
	if err := msgp.Decode(buf, &ddp); err != nil {
		me.t.Fatal(err)
	}
	me.mu.Lock()
	me.flushed = append(me.flushed, ddp)
	me.mu.Unlock()
	return ioutil.NopCloser(strings.NewReader(`{"rate_by_service":{"service:,env:":0.8,"service:db.users,env:":0.9}}`)), nil
}
