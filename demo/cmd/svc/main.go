package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type status int

var (
	errStatuses []status = []status{
		status(http.StatusBadRequest),
		status(http.StatusUnauthorized),
		status(http.StatusForbidden),
		status(http.StatusNotFound),
		status(http.StatusConflict),
		status(http.StatusInternalServerError),
		status(http.StatusNotImplemented),
	}
)

func main() {
	workHandler := func(w http.ResponseWriter, req *http.Request) {

		effort := time.Duration(100+rand.Intn(300)) * time.Millisecond
		<-time.After(effort)
		weight := rand.Intn(10)
		var stat status
		if weight < 8 {
			stat = status(http.StatusOK)
		} else {
			stat = errStatuses[rand.Intn(len(errStatuses))]
		}
		msg := fmt.Sprintf("work took this long: %v\n", effort)
		w.WriteHeader(int(stat))
		fmt.Fprintf(w, msg)
		fmt.Fprintf(os.Stderr, msg)
	}
	http.HandleFunc("/work", workHandler)
	addr := ":3000"
	fmt.Printf("Serving on %s\n", addr)
	http.ListenAndServe(addr, nil)
}
