# OpenCensus Go Datadog

[![CircleCI](https://circleci.com/gh/DataDog/opencensus-go-exporter-datadog.svg?style=svg)](https://circleci.com/gh/DataDog/opencensus-go-exporter-datadog) [![GoDoc][godoc-image]][godoc-url]

Provides OpenCensus stats and trace exporter support for Datadog Metrics and Datadog APM. The [examples folder](https://github.com/DataDog/opencensus-go-exporter-datadog/tree/master/examples)
provides some simple usage examples.

### Requirements:

- [Go 1.10+](https://golang.org/doc/install)
- [Datadog Agent 6](https://docs.datadoghq.com/agent/)

[godoc-image]: https://godoc.org/github.com/DataDog/opencensus-go-exporter-datadog?status.svg
[godoc-url]: https://godoc.org/github.com/DataDog/opencensus-go-exporter-datadog

### Disclaimer

For trace, this package is considered experiemental and comes with limitations. More specifically, due to the differences in operation between Datadog and OpenCensus, statistics (such as percentiles) seen in the Datadog application will be inaccurate and will be limited to only sampled traces. It is not advised to rely on these numbers to assert accurate system behaviour.  We are aware of the issue and the situation could change in the near future.
