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

func newStats(m metrics.Metrics) *stats {
	return &stats{
		metrics:   m,
		endpoints: make(map[string]*endpoint),
	}
}

func (s *stats) Record(req server.Request, d time.Duration, err error) {
	endpoint, ok := s.endpoints[req.Method()]
	if !ok {
		endpoint = newEndpoint(s.metrics, req.Method())
		s.endpoints[req.Method()] = endpoint
	}

	d_ms := d.Nanoseconds() / int64(time.Millisecond)

	// successful request, record and return
	if err == nil {
		endpoint.success.Record(d_ms)
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
		endpoint.bad.Record(d_ms)
	default:
		endpoint.errors.Record(d_ms)
	}
}

// Implements the server.HandlerWrapper
func MetricHandlerWrapper(m metrics.Metrics) server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {

		stats := newStats(m)

		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			begin := time.Now()

			err := fn(ctx, req, rsp)

			stats.Record(req, time.Since(begin), err)

			return err
		}
	}
}
