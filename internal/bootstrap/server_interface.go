package bootstrap

import (
	"context"
	"time"
)

type Server interface {
	Stop(ctx context.Context, timeout time.Duration) error
	Start(ctx context.Context) error
}
