package middleware

import (
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-os/metrics"
)

type stats struct {
	// Metrics interface
	// see https://github.com/micro/go-plugins/tree/master/metrics
	metrics metrics.Metrics
	// unit of time to be recorded
	unit time.Duration
	// track each service function
	endpoints map[string]*endpoint
}

// endpoint metrics
type endpoint struct {
	// successful request
	success metrics.Histogram
	// bad request, code 400-499
	bad metrics.Histogram
	// dropped requests, code 408
	dropped metrics.Histogram
	// internal server errors, code 500+
	errors metrics.Histogram
}

func newEndpoint(m metrics.Metrics, s string) *endpoint {
	return &endpoint{
		success: m.Histogram(s + ".success"),
		bad:     m.Histogram(s + ".bad"),
		dropped: m.Histogram(s + ".dropped"),
		errors:  m.Histogram(s + ".errors"),
	}
}

func newStats(m metrics.Metrics, u time.Duration) *stats {
	return &stats{
		metrics:   m,
		unit:      u,
		endpoints: make(map[string]*endpoint),
	}
}

func (s *stats) endpoint(e string) *endpoint {
	endpoint, ok := s.endpoints[e]
	if !ok {
		endpoint = newEndpoint(s.metrics, e)
		s.endpoints[e] = endpoint
	}

	return endpoint
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
	// Get the endpoint stats
	endpoint := s.endpoint(req.Method())
	// convert the duration into the time unit
	d_unit := s.durationToUnit(d)
	// successful request, record and return
	if err == nil {
		endpoint.success.Record(d_unit)
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
		endpoint.dropped.Record(d_unit)
	case 400 <= perr.Code && perr.Code <= 499:
		endpoint.bad.Record(d_unit)
	default:
		endpoint.errors.Record(d_unit)
	}
}

// Implements the server.HandlerWrapper
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

// Implements the server.SubscriberWrapper
func MetricSubscriberWrapper(m metrics.Metrics, u time.Duration) server.SubscriberWrapper {
	return func(fn server.SubscriberFunc) server.SubscriberFunc {

		stats := newStats(m, u)

		return func(ctx context.Context, msg server.Publication) error {
			// Begin the timer
			begin := time.Now()
			// Find the endpoint for metrics
			endpoint := stats.endpoint("subscriber")
			// Run additional middleware + subscriber function
			err := fn(ctx, msg)
			// Record success or error
			if err == nil {
				endpoint.success.Record(stats.durationToUnit(time.Since(begin)))
			} else {
				endpoint.errors.Record(stats.durationToUnit(time.Since(begin)))
			}

			return err
		}
	}
}
