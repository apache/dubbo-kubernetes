package log

import (
	"fmt"
	"go.uber.org/atomic"
	"go.uber.org/zap/zapcore"
	"strings"
	"sync"
	"time"
)

type Scope struct {
	name        string
	nameToEmit  string
	outputLevel *atomic.Value
	logCallers  *atomic.Value
}

var (
	lock sync.RWMutex
)

func RegisterScope(name string, desc string) *Scope {
	return registerScope(name, desc)
}

func registerScope(name string, desc string) *Scope {
	if strings.ContainsAny(name, ":,.") {
		panic(fmt.Sprintf("scope name %s is invalid, it cannot contain colons, commas, or periods", name))
	}
	return nil
}

func (s *Scope) GetOutputLevel() Level {
	return s.outputLevel.Load().(Level)
}

func (s *Scope) Infof(format string, args ...any) {
	if s.GetOutputLevel() >= InfoLevel {
		msg := maybeSprintf(format, args)
		s.emit(zapcore.InfoLevel, msg)
	}
}

func maybeSprintf(format string, args []any) string {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	return msg
}

func (s *Scope) emit(level zapcore.Level, msg string) {
	s.emitWithTime(level, msg, time.Now())
}

func (s *Scope) emitWithTime(level zapcore.Level, msg string, t time.Time) {
	if t.IsZero() {
		t = time.Now()
	}

	e := zapcore.Entry{
		Message:    msg,
		Level:      level,
		Time:       t,
		LoggerName: s.nameToEmit,
	}

	if s.GetLogCallers() {
		e.Caller = zapcore.NewEntryCaller(runtime.Caller(s.callerSkip + callerSkipOffset))
	}

	if dumpStack(level, s) {
		e.Stack = zap.Stack("").String
	}

	var fields []zapcore.Field
	if useJSON.Load().(bool) {
		fields = make([]zapcore.Field, 0, len(s.labelKeys))
		for _, k := range s.labelKeys {
			v := s.labels[k]
			fields = append(fields, zap.Field{
				Key:       k,
				Interface: v,
				Type:      zapcore.ReflectType,
			})
		}
	} else if len(s.labelKeys) > 0 {
		sb := &strings.Builder{}
		// Assume roughly 15 chars per kv pair. Its fine to be off, this is just an optimization
		sb.Grow(len(msg) + 15*len(s.labelKeys))
		sb.WriteString(msg)
		sb.WriteString("\t")
		space := false
		for _, k := range s.labelKeys {
			if space {
				sb.WriteString(" ")
			}
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(fmt.Sprint(s.labels[k]))
			space = true
		}
		e.Message = sb.String()
	}

	pt := funcs.Load().(patchTable)
	if pt.write != nil {
		if err := pt.write(e, fields); err != nil {
			_, _ = fmt.Fprintf(pt.errorSink, "%v log write error: %v\n", time.Now(), err)
			_ = pt.errorSink.Sync()
		}
	}
}

func (s *Scope) GetLogCallers() bool {
	return s.logCallers.Load().(bool)
}
