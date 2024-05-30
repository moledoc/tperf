package svc_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/moledoc/tperf"
)

func TestXxx(t *testing.T) {
	setup := func() (any, error) {
		route := "work"
		return route, nil
	}
	test := func(req any) (any, error) {
		route := req.(string)
		path := fmt.Sprintf("http://localhost:3000/%s", route)
		resp, respErr := http.Get(path)
		var err error
		if respErr != nil || resp == nil || resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("statuscode: %v err: %v", resp.StatusCode, err)
		}
		return resp, err
	}
	cleanup := func(any) (any, error) {
		return nil, nil
	}
	asserts := func(t *testing.T, report tperf.Report) (any, error) {
		KPI := 375 * time.Millisecond
		if report.P95 > time.Duration(KPI) {
			t.Logf("P95 greater than allowed, expected <%v, got %v\n", KPI, report.P95)
			t.FailNow()
		}
		return nil, nil
	}
	formalize := func(any) (any, error) {
		// EXAMPLE: uploading results
		fmt.Println("uploading results")
		return nil, nil
	}
	plan := tperf.Plan{
		T:                t,
		Ramping:          time.Duration(0 * time.Second),
		RequestPerSecond: 10,
		LoadFor:          time.Duration(1 * time.Second),
		Setup:            setup,
		Test:             test,
		Cleanup:          cleanup,
		Asserts:          asserts,
		Formalize:        formalize,
	}
	results := plan.Execute()
	// fmt.Println(results)
	report := plan.Summary(results)
	report.Print()
	plan.Asserts(t, report)
	plan.Formalize(nil)
}
