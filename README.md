# OpenCensus Go Datadog Exporter

## Basic Usage

A basic usage example can be seen below. Visit the [godoc page](https://godoc.org/github.com/DataDog/opencensus-go-exporter-datadog) for a more in depth look at the
available options.

```go
opts := datadog.Options{Service: "my-app"}
exporter := datadog.NewExporter(opts)
defer exporter.Stop()

view.RegisterExporter(exporter)

// define the measure
videoCount = stats.Int64("my.org/measures/video_count", "number of processed videos", stats.UnitDimensionless)

// define the view
countView, _ := &view.View{
	Name:        "video_count",
	Description: "number of videos processed over time",
	Measure:     videoCount,
	Aggregation: view.Count(),
}

```

### Requirements:

- Go 1.10
- Datadog Agent 6

### Disclaimer

For trace, this package is considered experiemental and comes with limitations. More specifically, due to the differences in operation between Datadog and OpenCensus, statistics (such as percentiles) seen in the Datadog application will be inaccurate and will be limited to only sampled traces. It is not advised to rely on these numbers to assert accurate system behaviour.  We are aware of the issue and the situation could change in the near future.
