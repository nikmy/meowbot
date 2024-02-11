package api

import "context"

type Server interface {
	Serve(ctx context.Context) error
	Shutdown(ctx context.Context) error
}
