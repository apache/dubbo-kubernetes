package log

import (
	"fmt"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Scope struct {
	name            string
	nameToEmit      string
	outputLevel     *atomic.Value
	stackTraceLevel *atomic.Value
	logCallers      *atomic.Value
	labels          map[string]any
	labelKeys       []string
	callerSkip      int
}

var (
	scopes = make(map[string]*Scope)
	lock   sync.RWMutex
)

func RegisterScope(name string) *Scope {
	return registerScope(name, 0)
}

func registerScope(name string, callerSkip int) *Scope {
	if strings.ContainsAny(name, ":,.") {
		panic(fmt.Sprintf("scope name %s is invalid, it cannot contain colons, commas, or periods", name))
	}
	lock.Lock()
	defer lock.Unlock()
	s, ok := scopes[name]
	if !ok {
		s = &Scope{
			name:            name,
			callerSkip:      callerSkip,
			outputLevel:     &atomic.Value{},
			stackTraceLevel: &atomic.Value{},
			logCallers:      &atomic.Value{},
		}
		s.SetOutputLevel(InfoLevel)
		s.SetStackTraceLevel(NoneLevel)
		s.SetLogCallers(false)
		if name != DefaultScopeName {
			s.nameToEmit = name
		}
		scopes[name] = s
	}
	s.labels = make(map[string]any)
	return s
}

func (s *Scope) SetOutputLevel(l Level) {
	s.outputLevel.Store(l)
}
func (s *Scope) SetStackTraceLevel(l Level) {
	s.stackTraceLevel.Store(l)
}
func (s *Scope) SetLogCallers(logCallers bool) {
	s.logCallers.Store(logCallers)
}

func (s *Scope) GetOutputLevel() Level {
	return s.outputLevel.Load().(Level)
}

func (s *Scope) GetStackTraceLevel() Level {
	return s.stackTraceLevel.Load().(Level)
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
	// TODO
}

func (s *Scope) GetLogCallers() bool {
	return s.logCallers.Load().(bool)
}
