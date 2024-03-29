# OpenCensus Go Datadog

[![CircleCI](https://circleci.com/gh/DataDog/opencensus-go-exporter-datadog.svg?style=svg)](https://circleci.com/gh/DataDog/opencensus-go-exporter-datadog) [![GoDoc][godoc-image]][godoc-url]

Provides OpenCensus stats and trace exporter support for Datadog Metrics and Datadog APM. The [examples folder](https://github.com/DataDog/opencensus-go-exporter-datadog/tree/master/examples)
provides some simple usage examples.

**Note**: **This repository is deprecated and is in maintenance mode. Bugs are being addressed, but new features are not being implemented.** OpenCensus and OpenTracing have merged into OpenTelemetry. It is recommended that OpenTelemetry alongside the 
[Datadog OpenTelemetry Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/datadogexporter)
be used instead. 


### Requirements:

- [Go 1.10+](https://golang.org/doc/install)
- [Datadog Agent 6](https://docs.datadoghq.com/agent/)

[godoc-image]: https://godoc.org/github.com/DataDog/opencensus-go-exporter-datadog?status.svg
[godoc-url]: https://godoc.org/github.com/DataDog/opencensus-go-exporter-datadog

### Contributing

Please make sure to read through our [contribution guidelines](https://github.com/DataDog/opencensus-go-exporter-datadog/tree/master/CONTRIBUTING.md) before opening an issue or a Pull Request.

### Disclaimer

In order to get accurate Datadog APM statistics and full distributed tracing, trace sampling must be done by the Datadog stack. For this to be possible, OpenCensus must be notified to forward all traces to our exporter:

```go
trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
```

This change simply means that Datadog will handle sampling. It does not mean that all traces will be sampled.
