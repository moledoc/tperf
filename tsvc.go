package tsvc

import (
	"cmp"
	"fmt"
	"io"
	"math"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
)

type Case struct {
	T                *testing.T
	Ramping          time.Duration
	rampups          []int
	rampdowns        []int
	RequestPerSecond int
	Duration         time.Duration
	Setup            func() (any, error)
	Test             func(any) (any, error)
	Cleanup          func(any) (any, error)
	Assert           func(Report) (any, error)
	Formalize        func() (any, error)
}

type result struct {
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
	ErrorRate    float64
	Ramping      time.Duration
	rampups      []int
	rampdowns    []int
	Results      []result
}

func (r Report) String() string {
	fieldCount := reflect.TypeOf(Report{}).NumField()
	format := fmt.Sprintf("\n%37s\n\n", "-- Summary --")
	for i := 0; i < fieldCount-1; i++ { // ignore: Results
		format += "%30s: %v\n"
	}
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
		"Error rate", r.ErrorRate,
		"Ramping", r.Ramping,
		"Ramp-up req/s", r.rampups,
		"Ramp-down req/s", r.rampdowns,
	)
}

func (r Report) Print() {
	fmt.Println(r)
}

func (r Report) Fprint(w io.Writer) {
	fmt.Fprintf(w, r.String())
}

func (kase *Case) Execute() Report {
	kase.T.Helper()

	kase.RequestPerSecond = max(kase.RequestPerSecond, 1)
	kase.Duration = max(kase.Duration, 1*time.Second)

	collector := make(chan result, int(math.Ceil(kase.Duration.Seconds()))*kase.RequestPerSecond)
	var wg sync.WaitGroup

	// ramp-up
	step := float64(kase.RequestPerSecond) / max(kase.Ramping.Seconds(), 1)
	var rampups []int
	for rps := float64(1); 0 < kase.Ramping.Seconds() && rps <= float64(kase.RequestPerSecond); rps += step {
		iterStart := time.Now()
		j := 0
		for i := float64(0); i < rps; i++ {
			j++
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, err := kase.Setup()
				if err != nil {
					return
				}
				resp, err := kase.Test(req)
				if err != nil {
					return
				}
				kase.Cleanup(resp)
			}()
		}
		rampups = append(rampups, j)
		iterDuration := time.Since(iterStart)
		<-time.After(max(1*time.Second-iterDuration, 0))
	}
	kase.T.Logf("Rampup done\n")

	// hit
	for i := float64(0); i < kase.Duration.Seconds(); i++ {
		iterStart := time.Now()
		for i := 0; i < kase.RequestPerSecond; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, err := kase.Setup()
				if err != nil {
					return
				}
				start := time.Now()
				resp, err := kase.Test(req)
				dur := time.Since(start)
				collector <- result{Duration: dur, Error: err}
				if err != nil {
					return
				}
				kase.Cleanup(resp)
			}()
		}
		iterDuration := time.Since(iterStart)
		<-time.After(max(time.Duration(1*time.Second)-iterDuration, 0))
	}
	kase.T.Logf("Test done\n")

	// ramp-down
	var rampdowns []int
	for rps := float64(kase.RequestPerSecond); 0 < kase.Ramping.Seconds() && float64(1) <= rps; rps -= step {
		iterStart := time.Now()
		j := 0
		for i := float64(0); i < rps; i++ {
			j++
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, err := kase.Setup()
				if err != nil {
					return
				}
				resp, err := kase.Test(req)
				if err != nil {
					return
				}
				kase.Cleanup(resp)
			}()
		}
		rampdowns = append(rampdowns, j)
		iterDuration := time.Since(iterStart)
		<-time.After(max(time.Duration(1*time.Second)-iterDuration, 0))
	}

	kase.T.Logf("Rampdown done\n")

	wg.Wait()
	close(collector)

	results := make([]result, len(collector))
	{
		i := 0
		for res := range collector {
			results[i] = res
			i++
		}
	}
	kase.rampups = rampups
	kase.rampdowns = rampdowns

	report := kase.Summary(results)
	kase.Assert(report)
	kase.Formalize()
	return report
}

func (kase *Case) Summary(results []result) Report {
	slices.SortFunc(results, func(a result, b result) int {
		return cmp.Compare(a.Duration, b.Duration)
	})
	mean := func(arr []result) time.Duration {
		var avg int64
		for _, res := range results {
			avg += res.Duration.Milliseconds()
		}
		return time.Duration(avg/int64(len(results))) * time.Millisecond
	}
	std := func(arr []result, avg int64) time.Duration {
		var std int64
		for _, res := range results {
			step := avg - res.Duration.Milliseconds()
			std += step * step
		}
		return time.Duration(math.Sqrt(float64(std/int64(max(len(results)-1,1))))) * time.Millisecond
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
	return Report{
		TestName:     kase.T.Name(),
		FullDuration: dur,
		RequestCount: len(results),
		P50:          results[len(results)*50/100].Duration,
		P90:          results[len(results)*90/100].Duration,
		P95:          results[len(results)*95/100].Duration,
		P99:          results[len(results)*99/100].Duration,
		Avg:          avg,
		Std:          std(results, avg.Milliseconds()),
		Throughput:   float64(len(results)) / dur.Seconds(),
		ErrorRate:    float64(errCount) / float64(len(results)),
		Ramping:      kase.Ramping,
		rampups:      kase.rampups,
		rampdowns:    kase.rampdowns,
		Results:      results,
	}
}
