package main 
// import "go.opencensus.io/examples/exporter"

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/DataDog/Opencensus-dd-exporter"
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

	exporter, err := datadog.NewExporter(datadog.Options{Tags: []string{"Christine"}})
	if err != nil {
		log.Fatal(err)
	}
	view.RegisterExporter(exporter)

	// Create view to see the number of processed videos cumulatively.
	// Create view to see the amount of video processed
	// Subscribe will allow view data to be exported.
	// Once no longer needed, you can unsubscribe from the view.
	if err = view.Register(
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
	); err != nil {
		log.Fatalf("Cannot subscribe to the view: %v\n", err)
	}

	// Set reporting period to report data at every second.
	view.SetReportingPeriod(1 * time.Second)

	// Record some data points...
		for true {
			log.Printf("recording...\n")
			stats.Record(ctx, videoCount.M(1), videoSize.M(rand.Int63()))
			<-time.After(time.Millisecond * time.Duration(1+rand.Intn(400)))
		}
	
}
