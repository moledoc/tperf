package tperf

import (
	"cmp"
	"fmt"
	"math"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
)

type Plan struct {
	T                *testing.T
	Ramping          time.Duration
	rampups          []int
	rampdowns        []int
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
	Throughput   float64 // req/s
	Errors       int
	Ramping      time.Duration
	rampups      []int
	rampdowns    []int
}

func (r Report) String() string {
	fieldCount := reflect.TypeOf(Report{}).NumField()
	format := "\n----------------------------------------------\n"
	for i := 0; i < fieldCount; i++ {
		format += "%30s: %v\n"
	}
	format += "----------------------------------------------\n"
	return fmt.Sprintf(
		format,
		"Test name", r.TestName,
		"Full duration", r.FullDuration,
		"Request count", r.RequestCount,
		"P50", r.P50,
		"P90", r.P90,
		"P95", r.P95,
		"P99", r.P99,
		"Avg", r.Avg,
		"Std", r.Std,
		"Throughput (req/s)", r.Throughput,
		"Error count", r.Errors,
		"Ramping", r.Ramping,
		"Ramp-up req/s", r.rampups,
		"Ramp-down req/s", r.rampdowns,
	)
}

func (plan *Plan) Execute() []Result {
	plan.T.Helper()

	load := max(int(plan.LoadFor.Seconds()), 1)

	collector := make(chan Result, load*plan.RequestPerSecond)
	var wg sync.WaitGroup

	// ramp-up
	step := float64(plan.RequestPerSecond) / max(plan.Ramping.Seconds(), 1)
	var rampups []int
	for rps := float64(1); 0 < plan.Ramping.Seconds() && rps <= float64(plan.RequestPerSecond); rps += step {
		iterStart := time.Now()
		j := 0
		for i := float64(0); i < rps; i++ {
			j++
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
		rampups = append(rampups, j)
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
	var rampdowns []int
	for rps := float64(plan.RequestPerSecond); 0 < plan.Ramping.Seconds() && float64(1) <= rps; rps -= step {
		iterStart := time.Now()
		j := 0
		for i := float64(0); i < rps; i++ {
			j++
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
		rampdowns = append(rampdowns, j)
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
	plan.rampups = rampups
	plan.rampdowns = rampdowns

	return results
}

func (plan *Plan) Summary(results []Result) Report {
	slices.SortFunc(results, func(a Result, b Result) int {
		return cmp.Compare(a.Duration, b.Duration)
	})
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
		Throughput:   float64(len(results)) / dur.Seconds(),
		Errors:       errCount,
		Ramping:      plan.Ramping,
		rampups:      plan.rampups,
		rampdowns:    plan.rampdowns,
	}
	fmt.Printf("%v\n", report)
	return report
}
