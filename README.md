# Datadog OpenCensus Metrics Exporter
Owned by Team Agent-Integrations (Tiramisu)

## Basic Usage

By default, the datadog exporter will connect to the agent running at `127.0.0.1:8125`.

```
exporter, err := datadog.NewExporter(datadog.Options{})
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