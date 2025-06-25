package spec

import "context"

type System interface {
	Connect(ctx context.Context) error
	Close(ctx context.Context) error
	Client() any
}
