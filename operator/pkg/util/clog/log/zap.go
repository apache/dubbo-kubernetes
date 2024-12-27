package log

import "go.uber.org/zap/zapcore"

var toLevel = map[zapcore.Level]Level{
	zapcore.FatalLevel: FatalLevel,
	zapcore.ErrorLevel: ErrorLevel,
	zapcore.WarnLevel:  WarnLevel,
	zapcore.InfoLevel:  InfoLevel,
	zapcore.DebugLevel: DebugLevel,
}

const callerSkipOffset = 3

func dumpStack(level zapcore.Level, scope *Scope) bool {
	thresh := toLevel[level]
	if scope != defaultScope {
		thresh = ErrorLevel
		switch level {
		case zapcore.FatalLevel:
			thresh = FatalLevel
		}
	}
	return scope.GetStackTraceLevel() >= thresh
}
