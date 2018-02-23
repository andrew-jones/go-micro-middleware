package middleware

import (
	"golang.org/x/net/context"
	"strconv"
	"strings"
	"time"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-os/metrics"
)

var (
	MetricRequest = "service.request"
)

type stats struct {
	// Metrics interface
	// see https://github.com/micro/go-plugins/tree/master/metrics
	metrics metrics.Metrics
	// unit of time to be recorded
	unit time.Duration
	// track multiple histograms by name
	histograms map[string]metrics.Histogram
}

func newHistogram(m metrics.Metrics, s string) metrics.Histogram {
	return m.Histogram(s)
}

func newStats(m metrics.Metrics, u time.Duration) *stats {
	return &stats{
		metrics:    m,
		unit:       u,
		histograms: make(map[string]metrics.Histogram),
	}
}

func (s *stats) histogram(n string) metrics.Histogram {
	histogram, ok := s.histograms[n]
	if !ok {
		histogram = newHistogram(s.metrics, n)
		s.histograms[n] = histogram
	}

	return histogram
}

func (s *stats) durationToUnit(d time.Duration) int64 {
	return d.Nanoseconds() / int64(s.unit)
}

func codeFromString(s string) int64 {
	search := "\"code\":"
	i := strings.Index(s, search)
	if i == -1 {
		// search string not found in input string
		return 500
	}
	// parse the code
	code := s[i+len(search) : i+len(search)+3]
	// convert to int64
	i64, err := strconv.ParseInt(code, 10, 32)
	if err == nil {
		return i64
	}
	// default status code is 500
	return 500
}

func (s *stats) Record(req server.Request, d time.Duration, err error) {
	// Get the service stats
	// service := s.endpoint(DefaultSumOfAllRequestsName)
	// Get the endpoint stats
	// endpoint := s.endpoint(req.Method())
	// convert the duration into the time unit
	dUnit := s.durationToUnit(d)
	// successful request, record and return
	tags := map[string]string{"status": "error"}

	if err == nil {
		tags["status"] = "success"
		s.histogram(MetricRequest).WithFields(tags).Record(dUnit)
		return
	}

	// parse the error
	perr, ok := err.(*errors.Error)
	if !ok {
		// Check the error message for a response code, error can be client.serverError
		perr = &errors.Error{Id: req.Service(), Code: int32(codeFromString(err.Error())), Detail: err.Error()}
	}
	// filter error code into bad requests and errors
	switch {
	case perr.Code == 408:
		tags["status"] = "dropped"
	case 400 <= perr.Code && perr.Code <= 499:
		tags["status"] = "bad"
	}

	s.histogram(MetricRequest).WithFields(tags).Record(dUnit)
}

// MetricHandlerWrapper implements the server.HandlerWrapper interface
func MetricHandlerWrapper(m metrics.Metrics, u time.Duration) server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {

		stats := newStats(m, u)

		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			// Begin the timer
			begin := time.Now()
			// Run additional middleware + handler function
			err := fn(ctx, req, rsp)
			// Request is almost done, record metrics
			stats.Record(req, time.Since(begin), err)

			return err
		}
	}
}
