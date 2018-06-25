// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package main

import (
	"context"

	datadog "github.com/DataDog/opencensus-go-exporter-datadog"
	"go.opencensus.io/trace"
)

func main() {
	exporter := datadog.NewExporter(datadog.Options{Service: "my-app"})
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
}
