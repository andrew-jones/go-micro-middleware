package middleware

import (
	"time"

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
	begin := time.Now()
	logMsg := log.WithFields(log.Fields{
		"ctx":     md,
		"service": req.Service(),
		"method":  req.Method(),
	})

	logMsg.Info("Calling service")

	err := l.Client.Call(ctx, req, rsp)

	if err != nil {
		logMsg = logMsg.WithFields(log.Fields{
			"error": err,
		})
	}
	// Add the duration in ms to the log message, rounding to the nearest int64
	logMsg.WithFields(log.Fields{
		"duration": int64(float64(time.Since(begin))/float64(time.Millisecond) + 0.5),
	}).Info("Called service")

	return err
}

// LogClientWrapper implements client.Wrapper as logWrapper interface
func LogClientWrapper(c client.Client) client.Client {
	return &logWrapper{c}
}

// LogHandlerWrapper implements the server.HandlerWrapper interface
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
