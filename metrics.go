package main

import (
	"net/http"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/go-kit/kit/metrics"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"

	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	fieldKeys    = []string{"method", "error"}
	RequestCount = kitprometheus.NewCounter(stdprometheus.CounterOpts{
		Namespace: "cron",
		Subsystem: "event_invocation",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	RequestLatency = metrics.NewTimeHistogram(time.Microsecond, kitprometheus.NewSummary(stdprometheus.SummaryOpts{
		Namespace: "cron",
		Subsystem: "event_invocation",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys))
	CountResult = kitprometheus.NewSummary(stdprometheus.SummaryOpts{
		Namespace: "cron",
		Subsystem: "event_invocation",
		Name:      "count_result",
		Help:      "The result of each count method.",
	}, []string{}) // no fields here

)

// SetupMetrics turns on Prometheus and Hystrix
func SetupMetrics() {

	// set up the http handlers and middleware

	http.Handle("/metrics", stdprometheus.Handler())

	// Enable hystrix dashboard metrics
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()

	http.Handle("/hystrix", hystrixStreamHandler)

}
