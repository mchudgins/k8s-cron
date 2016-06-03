package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/go-kit/kit/metrics"
	"github.com/mchudgins/cron"
)

// CronTab fires webEvents (by convention)
// -- see https://github.com/mchudgins/cron/blob/master/doc.go for
// an explanation of the format of the time interval.
func CronTab(c *cron.Cron) {
	c.AddFunc("@every 0h0m1s", func() { webEvent("EventName", "https://www.dstresearch.com") })
	c.AddFunc("@every 0h0m5s", func() { webEvent("Every 5s", "https://www.lunarlanding.com") })
	c.AddFunc("@every 0h0m2s", func() { webEvent("Every 2s", "https://www.yahoo.com") })
}

func webEvent(eventName string, targetURL string) {
	defer func(begin time.Time) {
		methodField := metrics.Field{Key: "method", Value: eventName}
		errorField := metrics.Field{Key: "error", Value: fmt.Sprintf("%v", nil)}
		RequestCount.With(methodField).With(errorField).Add(1)
		RequestLatency.With(methodField).With(errorField).Observe(time.Since(begin))
	}(time.Now())

	err := hystrix.Do(eventName, func() error {
		log.Printf("Event: %s, %s\n", eventName, targetURL)
		hostname, _ := os.Hostname()
		buf := fmt.Sprintf("{ 'event' : '%s', 'cron' : '%s' }", eventName, hostname)
		log.Printf("\tbuf: %s\n", buf)
		resp, err := http.Post(targetURL, "application/json", strings.NewReader(buf))
		if err != nil {
			return err
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("StatusCode: %d", resp.StatusCode)
		}
		return nil
	}, nil)
	if err != nil {
		log.Printf("\t%s occurred invoking event %s with target %s\n", err, eventName, targetURL)
	}
}
