package middleware

import (
	"golang.org/x/net/context"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-os/metrics"
)

var (
	MetricPublish   = "service.publish"
	MetricSubscribe = "service.subscribe"
)

type brokerMetricWrapper struct {
	broker.Broker
	stats *stats
}

func (w *brokerMetricWrapper) Publish(t string, b *broker.Message, opts ...broker.PublishOption) error {
	// Begin the timer
	begin := time.Now()
	// do wrapper thing
	err := w.Broker.Publish(t, b, opts...)
	// status success or error
	tags := map[string]string{"status": "error"}
	if err == nil {
		tags["status"] = "success"
	}
	// Record
	w.stats.histogram(MetricPublish).WithFields(tags).Record(w.stats.durationToUnit(time.Since(begin)))

	return err
}

func (w *brokerMetricWrapper) Subscribe(t string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	// wrapped handler
	wh := func(p broker.Publication) error {
		// Begin the timer
		begin := time.Now()
		// do wrapper thing
		err := h(p)
		// status success or error
		// successful request, record and return
		tags := map[string]string{"status": "error"}
		if err == nil {
			tags["status"] = "success"
		}
		// Record
		w.stats.histogram(MetricSubscribe).WithFields(tags).Record(w.stats.durationToUnit(time.Since(begin)))

		return err
	}

	return w.Broker.Subscribe(t, wh, opts...)
}

func MetricBrokerWrapper(b broker.Broker, m metrics.Metrics, u time.Duration) broker.Broker {
	stats := newStats(m, u)

	return &brokerMetricWrapper{b, stats}
}

// MetricSubscriberWrapper implements the server.SubscriberWrapper interface
func MetricSubscriberWrapper(m metrics.Metrics, u time.Duration) server.SubscriberWrapper {
	return func(fn server.SubscriberFunc) server.SubscriberFunc {

		stats := newStats(m, u)

		return func(ctx context.Context, msg server.Publication) error {
			// Begin the timer
			begin := time.Now()
			// Run additional middleware + subscriber function
			err := fn(ctx, msg)
			// Record success or error
			tags := map[string]string{"status": "error"}
			if err == nil {
				tags["status"] = "success"
			}

			stats.histogram(MetricSubscribe).WithFields(tags).Record(stats.durationToUnit(time.Since(begin)))

			return err
		}
	}
}
