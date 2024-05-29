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
		return http.Get(path)
	}
	cleanup := func(any) {
	}
	plan := tperf.Plan{
		T:                t,
		Rampup:           time.Duration(10 * time.Second),
		RequestPerSecond: 10,
		LoadFor:          time.Duration(2 * time.Second),
		Setup:            setup,
		Test:             test,
		Cleanup:          cleanup,
	}
	results := plan.Execute2()
	fmt.Println(results)
}
