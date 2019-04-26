// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package main

import (
	"context"
	"log"

	datadog "github.com/DataDog/opencensus-go-exporter-datadog"
	"go.opencensus.io/trace"
	ddtraceext "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
)

func main() {
	exporter, err := datadog.NewExporter(datadog.Options{Service: "my-app"})
	if err != nil {
		log.Fatal(err)
	}
	defer exporter.Stop()

	trace.RegisterExporter(exporter)

	// For demoing purposes, always sample.
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})

	ctx, span := trace.StartSpan(context.Background(), "/foo")
	bar(ctx)
	span.End()
}

func bar(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "/bar")
	defer span.End()

	// Do bar...

	// Set Datadog APM Trace Metadata
	span.AddAttributes(
		trace.StringAttribute(ddtraceext.ResourceName, "my-app-resource"),
		trace.StringAttribute(ddtraceext.SpanType, ddtraceext.SpanTypeWeb),
	)
}
