package test

import (
	"fmt"
	"log/slog"
)

type logger struct {
	log *slog.Logger
}

func (l *logger) Debugf(format string, args ...interface{}) {
	l.log.Debug(fmt.Sprintf(format, args...))
}

func (l *logger) Infof(format string, args ...interface{}) {
	l.log.Info(fmt.Sprintf(format, args...))
}

func (l *logger) Warnf(format string, args ...interface{}) {
	l.log.Warn(fmt.Sprintf(format, args...))
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.log.Error(fmt.Sprintf(format, args...))
}
