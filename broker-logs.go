package middleware

import (
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/server"
)

type brokerLogWrapper struct {
	broker.Broker
}

func (w *brokerLogWrapper) Publish(t string, b *broker.Message, opts ...broker.PublishOption) error {
	log.Info("Published message")
	// do wrapper thing
	err := w.Broker.Publish(t, b, opts...)

	return err
}

func (w *brokerLogWrapper) Subscribe(t string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	// wrapped handler
	wh := func(p broker.Publication) error {
		log.WithFields(log.Fields{
			"topic": p.Topic(),
		}).Info("Received message")

		err := h(p)

		return err
	}

	return w.Broker.Subscribe(t, wh, opts...)
}

func LogBrokerWrapper(b broker.Broker) broker.Broker {
	return &brokerLogWrapper{b}
}

// LogSubscriberWrapper implements the server.HandlerWrapper interface
func LogSubscriberWrapper(fn server.SubscriberFunc) server.SubscriberFunc {
	return func(ctx context.Context, msg server.Publication) error {
		md, _ := metadata.FromContext(ctx)
		log.WithFields(log.Fields{
			"ctx":          md,
			"topic":        msg.Topic(),
			"content-type": msg.ContentType(),
			"event":        msg.Message(),
		}).Infof("Received message")

		err := fn(ctx, msg)

		return err
	}
}
