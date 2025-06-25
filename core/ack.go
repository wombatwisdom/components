package core

import "context"

// ProcessedCallback is a function signature for a callback to be called after a
// message or message batch has been processed. The provided error indicates whether the processing was successful.
// TODO: add extra information on what happens if an error is returned
type ProcessedCallback func(ctx context.Context, err error) error

func NoopCallback(ctx context.Context, err error) error {
	return err
}
