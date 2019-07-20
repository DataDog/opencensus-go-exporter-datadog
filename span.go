// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadog.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"

	"go.opencensus.io/trace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
)

// statusCodes maps (*trace.SpanData).Status.Code to their message and http code. See:
// https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto.
var statusCodes = map[int32]statusCode{
	trace.StatusCodeOK:                 {message: "ok", httpCode: http.StatusOK},
	trace.StatusCodeCancelled:          {message: "cancelled", httpCode: 499},
	trace.StatusCodeUnknown:            {message: "unknown", httpCode: http.StatusInternalServerError},
	trace.StatusCodeInvalidArgument:    {message: "invalid_argument", httpCode: http.StatusBadRequest},
	trace.StatusCodeDeadlineExceeded:   {message: "deadline_exceeded", httpCode: http.StatusGatewayTimeout},
	trace.StatusCodeNotFound:           {message: "not_found", httpCode: http.StatusNotFound},
	trace.StatusCodeAlreadyExists:      {message: "already_exists", httpCode: http.StatusConflict},
	trace.StatusCodePermissionDenied:   {message: "permission_denied", httpCode: http.StatusForbidden},
	trace.StatusCodeResourceExhausted:  {message: "resource_exhausted", httpCode: http.StatusTooManyRequests},
	trace.StatusCodeFailedPrecondition: {message: "failed_precondition", httpCode: http.StatusBadRequest},
	trace.StatusCodeAborted:            {message: "aborted", httpCode: http.StatusConflict},
	trace.StatusCodeOutOfRange:         {message: "out_of_range", httpCode: http.StatusBadRequest},
	trace.StatusCodeUnimplemented:      {message: "unimplemented", httpCode: http.StatusNotImplemented},
	trace.StatusCodeInternal:           {message: "internal", httpCode: http.StatusInternalServerError},
	trace.StatusCodeUnavailable:        {message: "unavailable", httpCode: http.StatusServiceUnavailable},
	trace.StatusCodeDataLoss:           {message: "data_loss", httpCode: http.StatusNotImplemented},
	trace.StatusCodeUnauthenticated:    {message: "unauthenticated", httpCode: http.StatusUnauthorized},
}

type statusCode struct {
	message  string
	httpCode int
}

func statusMessage(code int32) string {
	if sc, exists := statusCodes[code]; exists {
		return sc.message
	}
	return "error code " + strconv.FormatInt(int64(code), 10)
}

func httpCode(code int32) int {
	if sc, exists := statusCodes[code]; exists {
		return sc.httpCode
	}
	return 500
}

// convertSpan takes an OpenCensus span and returns a Datadog span.
func (e *traceExporter) convertSpan(s *trace.SpanData) *ddSpan {
	startNano := s.StartTime.UnixNano()
	span := &ddSpan{
		TraceID:  binary.BigEndian.Uint64(s.SpanContext.TraceID[8:]),
		SpanID:   binary.BigEndian.Uint64(s.SpanContext.SpanID[:]),
		Name:     "opencensus",
		Resource: s.Name,
		Service:  e.opts.Service,
		Start:    startNano,
		Duration: s.EndTime.UnixNano() - startNano,
		Metrics:  map[string]float64{},
		Meta:     map[string]string{},
	}
	if s.ParentSpanID != (trace.SpanID{}) {
		span.ParentID = binary.BigEndian.Uint64(s.ParentSpanID[:])
	}

	httpCode := httpCode(s.Status.Code)
	switch s.SpanKind {
	case trace.SpanKindClient:
		span.Type = "client"
		if httpCode >= 400 && httpCode < 500 {
			span.Error = 1
		}
	case trace.SpanKindServer:
		span.Type = "server"
		if httpCode >= 500 && httpCode < 600 {
			span.Error = 1
		}
	default:
		if httpCode >= 500 && httpCode < 600 {
			span.Error = 1
		}
	}

	if span.Error == 1 {
		span.Meta[ext.ErrorType] = statusMessage(s.Status.Code)
		if msg := s.Status.Message; msg != "" {
			span.Meta[ext.ErrorMsg] = msg
		}
	}

	span.Meta[keyStatusCode] = strconv.Itoa(int(s.Status.Code))
	span.Meta[keyStatus] = statusMessage(s.Status.Code)
	if msg := s.Status.Message; msg != "" {
		span.Meta[keyStatusDescription] = msg
	}

	for key, val := range e.opts.GlobalTags {
		setTag(span, key, val)
	}
	for key, val := range s.Attributes {
		setTag(span, key, val)
	}
	return span
}

const (
	keySamplingPriority     = "_sampling_priority_v1"
	keyStatusDescription    = "opencensus.status_description"
	keyStatusCode           = "opencensus.status_code"
	keyStatus               = "opencensus.status"
	keySpanName             = "span.name"
	keySamplingPriorityRate = "_sampling_priority_rate_v1"
)

func setTag(s *ddSpan, key string, val interface{}) {
	if key == ext.Error {
		setError(s, val)
		return
	}
	switch v := val.(type) {
	case string:
		setStringTag(s, key, v)
	case bool:
		if v {
			setStringTag(s, key, "true")
		} else {
			setStringTag(s, key, "false")
		}
	case float64:
		setMetric(s, key, v)
	case int64:
		setMetric(s, key, float64(v))
	default:
		// should never happen according to docs, nevertheless
		// we should account for this to avoid exceptions
		setStringTag(s, key, fmt.Sprintf("%v", v))
	}
}

func setMetric(s *ddSpan, key string, v float64) {
	switch key {
	case ext.SamplingPriority:
		s.Metrics[keySamplingPriority] = v
	default:
		s.Metrics[key] = v
	}
}

func setStringTag(s *ddSpan, key, v string) {
	switch key {
	case ext.ServiceName:
		s.Service = v
	case ext.ResourceName:
		s.Resource = v
	case ext.SpanType:
		s.Type = v
	case ext.AnalyticsEvent:
		if v != "false" {
			setMetric(s, ext.EventSampleRate, 1)
		} else {
			setMetric(s, ext.EventSampleRate, 0)
		}
	case keySpanName:
		s.Name = v
	default:
		s.Meta[key] = v
	}
}

func setError(s *ddSpan, val interface{}) {
	switch v := val.(type) {
	case string:
		s.Error = 1
		s.Meta[ext.ErrorMsg] = v
	case bool:
		if v {
			s.Error = 1
		} else {
			s.Error = 0
		}
	case int64:
		if v > 0 {
			s.Error = 1
		} else {
			s.Error = 0
		}
	case nil:
		s.Error = 0
	default:
		s.Error = 1
	}
}
