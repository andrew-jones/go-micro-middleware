package middleware

import (
	log "github.com/sirupsen/logrus"

	"golang.org/x/net/context"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/server"
)

type logWrapper struct {
	client.Client
}

func (l *logWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	md, _ := metadata.FromContext(ctx)

	log.WithFields(log.Fields{
		"ctx":     md,
		"service": req.Service(),
		"method":  req.Method(),
	}).Info("Calling service")

	return l.Client.Call(ctx, req, rsp)
}

// Implements client.Wrapper as logWrapper
func LogWrap(c client.Client) client.Client {
	return &logWrapper{c}
}

// Implements the server.HandlerWrapper
func LogHandlerWrapper(fn server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp interface{}) error {
		md, _ := metadata.FromContext(ctx)
		log.WithFields(log.Fields{
			"ctx":    md,
			"method": req.Method(),
		}).Infof("Serving request")

		err := fn(ctx, req, rsp)

		return err
	}
}

// Implements the server.HandlerWrapper
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
