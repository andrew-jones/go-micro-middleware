package middleware

import (
	uuid "github.com/satori/go.uuid"

	"golang.org/x/net/context"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/server"
)

// trace wrapper attaches a unique trace ID - timestamp
type traceWrapper struct {
	client.Client
}

func (t *traceWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	// Add trace id to context if it doesn't exist
	ctx = addTraceId(ctx)
	// continue call stack
	return t.Client.Call(ctx, req, rsp)
}

// Implements the server.HandlerWrapper
func TraceHandlerWrapper(fn server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp interface{}) error {
		// Add trace id to context if it doesn't exist
		ctx = addTraceId(ctx)
		// continue call stack
		return fn(ctx, req, rsp)
	}
}

func addTraceId(ctx context.Context) context.Context {
	// get metadata from context or create new
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = metadata.Metadata{}
	}

	if _, ok := md["X-Trace-Id"]; !ok {
		// Set new trace id for this call
		md["X-Trace-Id"] = uuid.Must(uuid.NewV4()).String()
		// create new context with trace id
		ctx = metadata.NewContext(ctx, md)
	}

	return ctx
}

// Implements client.Wrapper as traceWrapper
func TraceClientWrapper(c client.Client) client.Client {
	return &traceWrapper{c}
}
