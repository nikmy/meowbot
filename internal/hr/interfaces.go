package hr

import (
	"context"

	"github.com/valyala/fasthttp"
)

type Server interface {
	Serve(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type reqIdGetter interface {
	GetRequestId(r *fasthttp.Request) string
}

type authorizer interface {
	Authorize(r *fasthttp.Request) (bool, error)
}
