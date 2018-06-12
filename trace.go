// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadog.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"bytes"
	"sync"
	"time"

	"go.opencensus.io/trace"
)

const (
	// payloadLimit specifies the maximum payload size that the Datadog
	// agent will accept. Request bodies larger than this will be rejected.
	payloadLimit = int(1e7) // 10M

	// defaultService specifies the default service name that will be used
	// with the registered traces. Users should normally specify a different
	// service name.
	defaultService = "opencensus-app"
)

// allows tests to override
var (
	// inChannelSize specifies the size of the buffered channel which
	// takes spans and adds them to the payload.
	inChannelSize = int(2e5) // 200K

	// flushThreshold specifies the payload's size threshold in bytes. If it
	// is exceed, a flush will be triggered. It is half of the maximum
	// payload size accepted by the Datadog agent.
	flushThreshold = payloadLimit / 2

	// flushInterval specifies the interval at which the payload will
	// automatically be flushed.
	flushInterval = 2 * time.Second
)

var _ trace.Exporter = (*traceExporter)(nil)

type traceExporter struct {
	opts    Options
	payload *payload
	errors  *errorAmortizer

	// uploadFn specifies the function used for uploading.
	// Defaults to (*transport).upload; replaced in tests.
	uploadFn func(pkg *bytes.Buffer, count int) error

	wg   sync.WaitGroup // counts active uploads
	in   chan *ddSpan
	exit chan struct{}
}

func newTraceExporter(o Options) *traceExporter {
	if o.Service == "" {
		o.Service = defaultService
	}
	e := &traceExporter{
		opts:     o,
		payload:  newPayload(),
		errors:   newErrorAmortizer(defaultErrorFreq, o.OnError),
		uploadFn: newTransport(o.TraceAddr).upload,
		in:       make(chan *ddSpan, inChannelSize),
		exit:     make(chan struct{}),
	}

	go e.loop()

	return e
}

func (e *traceExporter) loop() {
	defer close(e.exit)
	tick := time.NewTicker(flushInterval)
	defer tick.Stop()

	for {
		select {
		case span := <-e.in:
			if err := e.payload.add(span); err != nil {
				e.errors.log(errorTypeEncoding, err)
			}
			if e.payload.size() > flushThreshold {
				e.flush()
			}
		case <-tick.C:
			e.flush()
		case <-e.exit:
			e.flush()
			e.wg.Wait() // wait for uploads to finish
			return
		}
	}
}

// ExportSpan implements trace.Exporter.
func (e *traceExporter) ExportSpan(s *trace.SpanData) {
	select {
	case e.in <- e.convertSpan(s):
	default:
		e.errors.log(errorTypeOverflow, nil)
	}
}

func (e *traceExporter) flush() {
	n := len(e.payload.traces)
	if n == 0 {
		return
	}
	buf := e.payload.buffer()
	e.wg.Add(1)
	go func() {
		if err := e.uploadFn(buf, n); err != nil {
			e.errors.log(errorTypeTransport, err)
		}
		e.wg.Done()
	}()
	e.payload.reset()
}

// Stop cleanly stops the exporter, flushing any remaining spans to the transport.
// Only call Stop once per exporter. Repeated calls will cause panic.
func (e *traceExporter) Stop() {
	e.exit <- struct{}{}
	<-e.exit
}