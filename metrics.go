package middleware

import (
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
	// internal server errors, code 500+
	errors metrics.Histogram
}

func newEndpoint(m metrics.Metrics, s string) *endpoint {
	return &endpoint{
		success: m.Histogram(s + ".success"),
		bad:     m.Histogram(s + ".bad"),
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

func (s *stats) Record(req server.Request, d time.Duration, err error) {

	endpoint := s.endpoint(req.Method())

	d_unit := s.durationToUnit(d)

	// successful request, record and return
	if err == nil {
		endpoint.success.Record(d_unit)
		return
	}

	// parse the error
	perr, ok := err.(*errors.Error)
	if !ok {
		perr = &errors.Error{Id: req.Service(), Code: 500, Detail: err.Error()}
	}
	// filter error code into bad requests and errors
	switch {
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
			endpoint := stats.endpoint(msg.Topic())
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
