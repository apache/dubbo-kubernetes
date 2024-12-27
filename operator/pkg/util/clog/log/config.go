package log

import (
	"go.uber.org/atomic"
	"go.uber.org/zap/zapcore"
)

var (
	funcs   = &atomic.Value{}
	useJSON atomic.Value
)

type patchTable struct {
	write     func(ent zapcore.Entry, fields []zapcore.Field) error
	errorSink zapcore.WriteSyncer
}
