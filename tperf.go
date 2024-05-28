package tperf

import (
	"testing"
	"time"
)

type Plan struct {
	T                *testing.T
	Rampup           time.Duration
	RequestPerSecond int
	LoadFor          time.Duration
	Setup            func() (any, error)
	Hit              func(any) (any, error)
	Cleanup          func(any)
}

type Result struct {
	Duration time.Duration
	Error    error
}

type rps int // request per second

type Report struct {
	TestName   string
	P50        time.Duration
	P90        time.Duration
	P99        time.Duration
	Avg        time.Duration
	Std        time.Duration
	Throughput rps
	Errors     int
}

func (plan *Plan) Execute() []Result {
	// TODO: ramp-up

	// TODO: hit

	// TODO: collect results

	// TODO: ramp-down
	return result
}
