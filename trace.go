package middleware

import (
	uuid "github.com/satori/go.uuid"

	"golang.org/x/net/context"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/metadata"
)

// trace wrapper attaches a unique trace ID - timestamp
type traceWrapper struct {
	client.Client
}

func (t *traceWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	// get metadata from context or create new
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = metadata.Metadata{}
	}
	// Set new trace id for this call
	md["X-Trace-Id"] = uuid.NewV4().String()
	// create new context with trace id
	ctx = metadata.NewContext(ctx, md)
	// continue call stack
	return t.Client.Call(ctx, req, rsp)
}

// Implements client.Wrapper as traceWrapper
func TraceWrap(c client.Client) client.Client {
	return &traceWrapper{c}
}
