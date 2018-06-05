// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"sync"

	"go.opencensus.io/stats/view"
)

// client implements datadog.Client
type collector struct {
	opts Options

	// mu guards all the fields.
	mu sync.Mutex

	skipErrors bool

	// viewData is accumulated and appended on every Export
	// invocation from stats.
	viewData map[string]*view.Data
}
