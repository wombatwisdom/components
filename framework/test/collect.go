package test

import (
	"context"
	"sync"

	"github.com/wombatwisdom/components/framework/spec"
)

func NewListCollector() *ListCollector {
	return &ListCollector{
		dataLock: make(chan struct{}),
	}
}

type ListCollector struct {
	lock sync.Mutex

	dataLock chan struct{}

	messages []spec.Message
}

func (l *ListCollector) Wait() {
	l.lock.Lock()
	hasMessages := len(l.messages) > 0
	l.lock.Unlock()
	
	if hasMessages {
		return
	}

	<-l.dataLock
}

func (l *ListCollector) Messages() []spec.Message {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.messages
}

func (l *ListCollector) Collect(message spec.Message) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.messages = append(l.messages, message)

	if l.dataLock != nil {
		close(l.dataLock)
	}
	return nil
}

func (l *ListCollector) Flush() (spec.Batch, error) {
	// For testing, we don't need to return an actual batch
	return nil, nil
}

func (l *ListCollector) Write(message spec.Message) error {
	return l.Collect(message)
}

func (l *ListCollector) Disconnect() error {
	return nil
}

// Legacy methods
func (l *ListCollector) WriteOld(ctx context.Context, message spec.Message) error {
	return l.Collect(message)
}

func (l *ListCollector) DisconnectOld(ctx context.Context) error {
	return nil
}
