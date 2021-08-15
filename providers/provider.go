package providers

import (
	"context"
)

type Provider interface {
	Associate(ctx context.Context, publicIP string, privateIP string) (sourceIP string, err error)
	Dissociate(ctx context.Context, sourceIP string) error
}
