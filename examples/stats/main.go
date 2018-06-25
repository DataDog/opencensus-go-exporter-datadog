// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	datadog "github.com/DataDog/opencensus-go-exporter-datadog"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

// Create measures. The program will record measures for the size of
// processed videos and the number of videos marked as spam.
var (
	videoCount = stats.Int64("my.org/measures/video_count", "number of processed videos", stats.UnitDimensionless)
	videoSize  = stats.Int64("my.org/measures/video_size", "size of processed video", stats.UnitBytes)
)

func main() {
	ctx := context.Background()

	exporter := datadog.NewExporter(datadog.Options{})
	defer exporter.Stop()

	view.RegisterExporter(exporter)

	// Create view to see the number of processed videos cumulatively.
	// Create view to see the amount of video processed
	view.Register(
		&view.View{
			Name:        "video_count",
			Description: "number of videos processed over time",
			Measure:     videoCount,
			Aggregation: view.Count(),
		},
		&view.View{
			Name:        "video_size",
			Description: "processed video size over time",
			Measure:     videoSize,
			Aggregation: view.Distribution(0, 1<<16, 1<<32),
		},
	)

	// Set reporting period to report data at every second.
	view.SetReportingPeriod(1 * time.Second)

	// Record some measures...
	for {
		log.Printf("recording...\n")
		stats.Record(ctx, videoCount.M(1), videoSize.M(rand.Int63()))
		<-time.After(time.Millisecond * time.Duration(1+rand.Intn(400)))
	}

}
