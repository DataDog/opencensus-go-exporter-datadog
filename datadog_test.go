// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.
package datadog

import (
	// "bytes"
	// "errors"
	//"fmt"

	// "sync"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	//"github.com/DataDog/datadog-go/statsd"
)

var measureCount = stats.Int64("fooCount", "bytes", stats.UnitBytes)

func newView(measureName string, agg *view.Aggregation) *view.View {
	return &view.View{
		Name:        "fooCount",
		Description: "fooDesc",
		Measure:     measureCount,
		Aggregation: agg,
	}
}
