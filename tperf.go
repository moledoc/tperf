package tperf

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"sync"
	"testing"
	"time"
)

type Plan struct {
	T                *testing.T
	Rampup           time.Duration
	RequestPerSecond int
	LoadFor          time.Duration
	Setup            func() (any, error)
	Test             func(any) (any, error)
	Cleanup          func(any)
}

type Result struct {
	Duration time.Duration
	Response any
	Error    error
}

type rps float64 // request per second

type Report struct {
	TestName     string
	FullDuration time.Duration
	RequestCount int
	P50          time.Duration
	P90          time.Duration
	P95          time.Duration
	P99          time.Duration
	Avg          time.Duration
	Std          time.Duration
	Throughput   rps
	Errors       int
}

func (r Report) String() string {
	return fmt.Sprintf("Test name: %s\nFull duration: %v\nRequest count: %v\nP50: %v\nP90: %v\nP95: %v\nP99: %v\nAvgerage: %v\nStandard deviation: %v\nThroughput: %v requests/second\nErrors count: %v", r.TestName, r.FullDuration, r.RequestCount, r.P50, r.P90, r.P95, r.P99, r.Avg, r.Std, r.Throughput, r.Errors)
}

func (plan *Plan) Execute() []Result {
	plan.T.Helper()

	load := max(int(plan.LoadFor.Seconds()), 1)

	collector := make(chan Result, load*plan.RequestPerSecond)
	var wg sync.WaitGroup

	// ramp-up
	step := float64(plan.RequestPerSecond) / max(plan.Rampup.Seconds(), 1)
	for rps := float64(0); plan.Rampup.Seconds() > 0 && rps <= float64(plan.RequestPerSecond); rps += step {
		iterStart := time.Now()
		for i := float64(0); i < rps; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, err := plan.Setup()
				if err != nil {
					return
				}
				resp, err := plan.Test(req)
				if err != nil {
					return
				}
				plan.Cleanup(resp)
			}()
		}
		iterDuration := time.Since(iterStart)
		<-time.After(max(1*time.Second-iterDuration, 0))
	}
	plan.T.Logf("Rampup done\n")

	// hit
	for i := 0; i < load; i++ {
		iterStart := time.Now()
		for i := 0; i < plan.RequestPerSecond; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, err := plan.Setup()
				if err != nil {
					return
				}
				start := time.Now()
				resp, err := plan.Test(req)
				dur := time.Since(start)
				collector <- Result{Duration: dur, Error: err}
				if err != nil {
					return
				}
				plan.Cleanup(resp)
			}()
		}
		iterDuration := time.Since(iterStart)
		<-time.After(max(time.Duration(1*time.Second)-iterDuration, 0))
	}
	plan.T.Logf("Test done\n")

	// ramp-down
	for rps := float64(plan.RequestPerSecond); plan.Rampup.Seconds() > 0 && rps > float64(0); rps -= step {
		iterStart := time.Now()
		for i := float64(0); i < rps; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, err := plan.Setup()
				if err != nil {
					return
				}
				resp, err := plan.Test(req)
				if err != nil {
					return
				}
				plan.Cleanup(resp)
			}()
		}
		iterDuration := time.Since(iterStart)
		<-time.After(max(time.Duration(1*time.Second)-iterDuration, 0))
	}

	plan.T.Logf("Rampdown done\n")

	wg.Wait()
	close(collector)

	results := make([]Result, len(collector))
	{
		i := 0
		for res := range collector {
			results[i] = res
			i++
		}
	}

	return results
}

func (plan *Plan) Summary(results []Result) Report {
	// TODO: sort results based on duration
	slices.SortFunc(results, func(a Result, b Result) int {
		return cmp.Compare(a.Duration, b.Duration)
	})
	plan.T.Logf("sorted: %v\n", results)
	mean := func(arr []Result) time.Duration {
		var avg int64
		for _, res := range results {
			avg += res.Duration.Milliseconds()
		}
		return time.Duration(avg/int64(len(results))) * time.Millisecond
	}
	std := func(arr []Result, avg int64) time.Duration {
		var std int64
		for _, res := range results {
			step := avg - res.Duration.Milliseconds()
			std += step * step
		}
		return time.Duration(math.Sqrt(float64(std/int64(len(results)-1)))) * time.Millisecond
	}
	avg := mean(results)
	errCount := 0
	var dur time.Duration
	for i := 0; i < len(results); i++ {
		if results[i].Error != nil {
			errCount++
		}
		dur += results[i].Duration
	}
	report := Report{
		TestName:     plan.T.Name(),
		FullDuration: dur,
		RequestCount: len(results),
		P50:          results[len(results)*50/100].Duration,
		P90:          results[len(results)*90/100].Duration,
		P95:          results[len(results)*95/100].Duration,
		P99:          results[len(results)*99/100].Duration,
		Avg:          avg,
		Std:          std(results, avg.Milliseconds()),
		Throughput:   rps(float64(len(results)) / dur.Seconds()),
		Errors:       errCount,
	}
	plan.T.Logf("%v\n", report)
	return report
}
