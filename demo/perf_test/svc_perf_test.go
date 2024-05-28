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
		return nil, nil
	}
	test := func(req any) (any, error) {
		return http.Get("http://localhost:3000")
	}
	cleanup := func(any) {
	}
	plan := tperf.Plan{
		T:                t,
		Rampup:           time.Duration(0 * time.Second),
		RequestPerSecond: 2,
		LoadFor:          time.Duration(3 * time.Second),
		Setup:            setup,
		Test:             test,
		Cleanup:          cleanup,
	}
	results := plan.Execute()
	fmt.Println(results)
}
