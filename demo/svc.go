package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

type status int

var (
	statuses []status = []status{
		status(http.StatusOK),
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

		effort := time.Duration(300+rand.Intn(100)) * time.Millisecond
		<-time.After(effort)
		stat := statuses[rand.Intn(len(statuses))]
		w.WriteHeader(int(stat))
		fmt.Fprintf(w, "work took this long: %v\n", effort)
	}
	http.HandleFunc("/", workHandler)
	addr := ":3000"
	fmt.Printf("Serving on %s\n", addr)
	http.ListenAndServe(addr, nil)
}
