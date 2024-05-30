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
	cleanup := func(any) {
	}
	plan := tperf.Plan{
		T:                t,
		Rampup:           time.Duration(6 * time.Second),
		RequestPerSecond: 4,
		LoadFor:          time.Duration(4 * time.Second),
		Setup:            setup,
		Test:             test,
		Cleanup:          cleanup,
	}
	results := plan.Execute()
	// fmt.Println(results)
	plan.Summary(results)
}
