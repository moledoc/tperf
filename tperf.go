package tperf

import (
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

type helperConf struct {
	wg         sync.WaitGroup
	test       func(any) (any, error)
	step       float64
	startBound float64
	endBound   float64
	compare    func(float64, float64) bool
}

func (plan *Plan) helper(hc helperConf, iters int) {
	plan.T.Helper()
	var wg sync.WaitGroup
	for i := 0; i < iters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := plan.Setup()
			if err != nil {
				return
			}
			resp, err := hc.test(req)
			if err != nil {
				return
			}
			plan.Cleanup(resp)
		}()
	}
	wg.Wait()
}

func (plan *Plan) loop(conf helperConf) {
	plan.T.Helper()
	for i := float64(conf.startBound); conf.compare(i, conf.endBound); i += conf.step {
		go plan.helper(conf, int(i))
		<-time.After(1 * time.Second)
	}
}

func (plan *Plan) Execute() []Result {
	plan.T.Helper()

	load := max(int(plan.LoadFor.Seconds()), 1)

	collector := make(chan Result, load*plan.RequestPerSecond)
	var wg sync.WaitGroup

	rampMiddle := func(req any) (any, error) {
		return plan.Test(req)
	}
	rampConf := helperConf{
		wg:         wg,
		test:       rampMiddle,
		step:       float64(plan.RequestPerSecond) / plan.Rampup.Seconds(),
		startBound: 0,
		endBound:   float64(plan.RequestPerSecond),
		compare:    func(a float64, b float64) bool { return a < b },
	}

	testMiddle := func(req any) (any, error) {
		start := time.Now()
		resp, err := plan.Test(req)
		dur := time.Since(start)
		collector <- Result{Duration: dur, Error: err}
		return resp, err
	}
	testConf := helperConf{
		wg:         wg,
		test:       testMiddle,
		step:       1,
		startBound: 0,
		endBound:   float64(plan.RequestPerSecond),
		compare:    func(a float64, b float64) bool { return a < b },
	}

	// TODO: ramp-up
	plan.loop(rampConf)
	plan.T.Logf("Rampup done\n")

	// TODO: hit
	for i := 0; i < load; i++ {
		iterStart := time.Now()
		plan.loop(testConf)
		iterDuration := time.Since(iterStart)
		<-time.After((time.Duration(1) - iterDuration) * time.Second)
	}
	plan.T.Logf("Test done\n")

	// TODO: ramp-down
	rampConf.step *= -1
	newStart := rampConf.endBound
	newEnd := rampConf.startBound
	rampConf.startBound = newStart
	rampConf.endBound = newEnd
	rampConf.compare = func(a float64, b float64) bool { return a > b }
	plan.loop(rampConf)

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

func (plan *Plan) Execute2() []Result {
	plan.T.Helper()

	load := max(int(plan.LoadFor.Seconds()), 1)

	collector := make(chan Result, load*plan.RequestPerSecond)
	var wg sync.WaitGroup

	// TODO: ramp-up
	step := float64(plan.RequestPerSecond) / max(plan.Rampup.Seconds(), 1)
	for rps := float64(0); plan.Rampup.Seconds() > 0 && rps < float64(plan.RequestPerSecond); rps += step {
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
	plan.T.Logf("Rampup done\n")

	// TODO: hit
	for i := 0; i < load; i++ {
		iterStart := time.Now()
		for i := float64(0); i < float64(plan.RequestPerSecond); i++ {
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

	// TODO: ramp-down
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
