package utils

import "code.byted.org/gopkg/logs"

// Logger .
type Logger interface {
	Debugf(format string, a ...interface{})
	Infof(format string, a ...interface{})
	Warnf(format string, a ...interface{})
	Errorf(format string, a ...interface{})
	Fatalf(format string, a ...interface{})
}

type defaultLogger struct {
	name string
}

func NewLogger(modelName string) Logger {
	return &defaultLogger{modelName}
}

func (l *defaultLogger) Debugf(format string, a ...interface{}) {
	logs.Debugf("["+l.name+"] "+format, a...)
}

func (l *defaultLogger) Infof(format string, a ...interface{}) {
	logs.Infof("["+l.name+"] "+format, a...)
}

func (l *defaultLogger) Warnf(format string, a ...interface{}) {
	logs.Warnf("["+l.name+"] "+format, a...)
}

func (l *defaultLogger) Errorf(format string, a ...interface{}) {
	logs.Errorf("["+l.name+"] "+format, a...)
}

func (l *defaultLogger) Fatalf(format string, a ...interface{}) {
	logs.Fatalf("["+l.name+"] "+format, a...)
}
