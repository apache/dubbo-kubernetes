package clog

import "io"

type ConsoleLogger struct {
	stdOut io.Writer
	stdErr io.Writer
	scope  *log.Scope
}

type Logger interface {
	LogAndPrint(v ...any)
	LogAndError(v ...any)
	LogAndFatal(v ...any)
	LogAndPrintf(format string, a ...any)
	LogAndErrorf(format string, a ...any)
	LogAndFatalf(format string, a ...any)
}

func NewDefaultLogger() *ConsoleLogger {
	return nil
}

func (l *ConsoleLogger) LogAndPrint(v ...any) {
	return
}
