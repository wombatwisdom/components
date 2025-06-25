package core

import "context"

type System interface {
    Schema() string
    Connect(ctx context.Context) error
    Close(ctx context.Context) error
    Client() any
}
