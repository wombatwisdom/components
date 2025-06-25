package test

import (
	"context"
	"github.com/wombatwisdom/components/spec"
	"sync"
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
	if len(l.messages) > 0 {
		return
	}

	<-l.dataLock
}

func (l *ListCollector) Messages() []spec.Message {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.messages
}

func (l *ListCollector) Write(ctx context.Context, message spec.Message) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.messages = append(l.messages, message)

	if l.dataLock != nil {
		close(l.dataLock)
	}
	return nil
}

func (l *ListCollector) Disconnect(ctx context.Context) error {
	return nil
}
