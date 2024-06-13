package svc_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/moledoc/tsvc"
)

var (
	globalPlan = func(t *testing.T) tsvc.Plan {
		return tsvc.Plan{
			Setup: func() (any, error) {
				route := "work"
				return route, nil
			},
			Test: func(req any, _ error) (any, error) {
				route := req.(string)
				path := fmt.Sprintf("http://localhost:3000/%s", route)
				resp, respErr := http.Get(path)
				var err error
				if respErr != nil || resp == nil || resp.StatusCode != http.StatusOK {
					err = fmt.Errorf("statuscode: %v err: %v", resp.StatusCode, err)
				}
				return resp, err
			},
			Cleanup: func(any, error) (any, error) {
				return nil, nil
			},
			Assert: func(report *tsvc.Report) (any, error) {
				KPI := 375 * time.Millisecond
				if report.P95 > time.Duration(KPI) {
					t.Logf("P95 greater than allowed, expected <%v, got %v\n", KPI, report.P95)
					t.Fail()
				}
				return nil, nil
			},
			Formalize: func() (any, error) {
				// EXAMPLE: uploading results
				fmt.Println("uploading results")
				return nil, nil
			},
		}
	}
)

func TestXxx_functional(t *testing.T) {
	plan := globalPlan(t)
	plan.T = t
	plan.RequestPerSecond = 1
	plan.Ramping = time.Duration(0 * time.Second)
	plan.Duration = time.Duration(0 * time.Second)
	report := plan.Run()
	fmt.Println(report)
}

func TestXxx_performance(t *testing.T) {
	plan := globalPlan(t)
	plan.T = t
	plan.RequestPerSecond = 10
	plan.Ramping = time.Duration(10 * time.Second)
	plan.Duration = time.Duration(4600 * time.Millisecond)
	fmt.Println(plan.Run())
}

func TestXxx2(t *testing.T) {
	setup := func() (any, error) {
		route := "work"
		return route, nil
	}
	test := func(req any, _ error) (any, error) {
		route := req.(string)
		path := fmt.Sprintf("http://localhost:3000/%s", route)
		resp, respErr := http.Get(path)
		var err error
		if respErr != nil || resp == nil || resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("statuscode: %v err: %v", resp.StatusCode, err)
		}
		return resp, err
	}
	cleanup := func(any, error) (any, error) {
		return nil, nil
	}
	asserts := func(report *tsvc.Report) (any, error) {
		KPI := 375 * time.Millisecond
		if report.P95 > time.Duration(KPI) {
			t.Logf("P95 greater than allowed, expected <%v, got %v\n", KPI, report.P95)
			t.Fail()
		}
		return nil, nil
	}
	formalize := func() (any, error) {
		// EXAMPLE: uploading results
		fmt.Println("uploading results")
		return nil, nil
	}
	plan := tsvc.Plan{
		T:                t,
		Ramping:          time.Duration(2 * time.Second),
		RequestPerSecond: 5,
		Duration:         time.Duration(3 * time.Second),
		Setup:            setup,
		Test:             test,
		Cleanup:          cleanup,
		Assert:           asserts,
		Formalize:        formalize,
	}
	report := plan.Run()
	fmt.Println(report)
	fmt.Printf("%+v\n", report.Results)
	fmt.Println(report.Ramping)
}
