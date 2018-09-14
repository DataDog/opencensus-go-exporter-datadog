// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package datadog

// DefaultStatsAddrUDS specifies the default socket address for the DogStatsD service over UDS.
const DefaultStatsAddrUDS = "unix:///var/run/datadog/dsd.socket"
