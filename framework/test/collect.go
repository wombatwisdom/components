package test

import (
	"context"
	"sync"
	"time"

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
	locked   bool

	messages []spec.Message
}

func (l *ListCollector) Wait() {
	l.WaitWithTimeout(30 * time.Second)
}

func (l *ListCollector) WaitWithTimeout(timeout time.Duration) bool {
	l.lock.Lock()
	hasMessages := len(l.messages) > 0
	dataLock := l.dataLock
	locked := l.locked
	l.lock.Unlock()

	if hasMessages {
		return true
	}

	if locked {
		return true
	}

	select {
	case <-dataLock:
		return true
	case <-time.After(timeout):
		return false
	}
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

	if l.dataLock != nil && !l.locked {
		close(l.dataLock)
		l.locked = true
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

func (l *ListCollector) Reset() {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.messages = nil
	if l.locked {
		l.dataLock = make(chan struct{})
		l.locked = false
	}
}

// Legacy methods
func (l *ListCollector) WriteOld(ctx context.Context, message spec.Message) error {
	return l.Collect(message)
}

func (l *ListCollector) DisconnectOld(ctx context.Context) error {
	return nil
}
