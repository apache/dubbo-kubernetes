package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level represents the logging level
type Level int

const (
	NoneLevel Level = iota
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

var levelNames = map[Level]string{
	ErrorLevel: "error",
	WarnLevel:  "warn",
	InfoLevel:  "info",
	DebugLevel: "debug",
}

// Scope represents a logging scope with its own level and output
type Scope struct {
	name        string
	description string
	outputLevel Level
	mu          sync.RWMutex
	output      io.Writer
}

// Logger provides logging methods for a scope
type Logger struct {
	scope *Scope
}

// Scope returns the underlying scope
func (l *Logger) Scope() *Scope {
	return l.scope
}

var (
	scopes            = make(map[string]*Scope)
	scopesMu          sync.RWMutex
	defaultOut        io.Writer = os.Stderr
	globalMu          sync.Mutex
	usePrettyLog      bool = false
	prettyLogMu       sync.RWMutex
	defaultScope      = "default"
	defaultLogger     *Logger
	defaultLoggerOnce sync.Once
)

// RegisterScope creates and registers a new logging scope
func RegisterScope(name, description string) *Logger {
	scopesMu.Lock()
	defer scopesMu.Unlock()

	if scope, exists := scopes[name]; exists {
		return &Logger{scope: scope}
	}

	scope := &Scope{
		name:        name,
		description: description,
		outputLevel: InfoLevel,
		output:      defaultOut,
	}

	scopes[name] = scope
	return &Logger{scope: scope}
}

// SetOutputLevel sets the output level for the scope
func (s *Scope) SetOutputLevel(level Level) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outputLevel = level
}

// GetOutputLevel returns the current output level
func (s *Scope) GetOutputLevel() Level {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.outputLevel
}

// SetOutput sets the output writer for the scope
func (s *Scope) SetOutput(w io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.output = w
}

// GetOutput returns the current output writer
func (s *Scope) GetOutput() io.Writer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.output
}

// Name returns the scope name
func (s *Scope) Name() string {
	return s.name
}

// Description returns the scope description
func (s *Scope) Description() string {
	return s.description
}

// SetDefaultOutput sets the default output writer for all scopes
func SetDefaultOutput(w io.Writer) {
	globalMu.Lock()
	defer globalMu.Unlock()
	defaultOut = w
}

// SetScopeLevel sets the output level for a specific scope
func SetScopeLevel(name string, level Level) {
	scopesMu.RLock()
	scope, exists := scopes[name]
	scopesMu.RUnlock()

	if exists {
		scope.SetOutputLevel(level)
	}
}

// SetAllScopesLevel sets the output level for all scopes
func SetAllScopesLevel(level Level) {
	scopesMu.RLock()
	defer scopesMu.RUnlock()

	for _, scope := range scopes {
		scope.SetOutputLevel(level)
	}
}

// FindScope returns a scope by name
func FindScope(name string) *Scope {
	scopesMu.RLock()
	defer scopesMu.RUnlock()
	return scopes[name]
}

// AllScopes returns all registered scopes
func AllScopes() map[string]*Scope {
	scopesMu.RLock()
	defer scopesMu.RUnlock()

	result := make(map[string]*Scope)
	for k, v := range scopes {
		result[k] = v
	}
	return result
}

// log writes a log message if the level is enabled
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if l.scope.GetOutputLevel() < level {
		return
	}

	msg := formatMessage(format, args...)
	levelName := levelNames[level]
	scopeName := l.scope.Name()

	// Check deduplication
	dedup := GetDeduplicator()
	key := LogKey{
		Scope:   scopeName,
		Level:   levelName,
		Message: msg,
	}

	shouldLog, _, summary := dedup.ShouldLog(key)
	if !shouldLog {
		// Suppress duplicate log
		return
	}

	// Append summary if there were duplicates
	if summary != "" {
		msg = msg + " " + summary
		// Clean any newlines that might have been introduced
		msg = strings.TrimRight(msg, "\n\r")
		msg = strings.ReplaceAll(msg, "\n", " ")
		msg = strings.ReplaceAll(msg, "\r", " ")
	}

	output := l.scope.GetOutput()
	if output == nil {
		output = defaultOut
	}

	var logLine string
	prettyLogMu.RLock()
	usePretty := usePrettyLog
	prettyLogMu.RUnlock()

	if usePretty {
		prettyFormatter := GetPrettyFormatter()
		logLine = prettyFormatter.Format(time.Now().UTC(), levelName, scopeName, msg)
	} else {
		// Use standard format by default, or klog format if configured
		logFormatMu.RLock()
		format := logFormat
		logFormatMu.RUnlock()

		if format == FormatKlog {
			// Use klog format with caller information
			logLine = formatKlogLine(level, scopeName, msg, callerSkip)
		} else {
			// Use standard format (default) with fixed-width timestamp
			timestamp := formatFixedWidthTimestamp(time.Now().UTC())
			logLine = formatStandardLine(timestamp, levelName, scopeName, msg)
		}
	}

	// Ensure logLine has exactly one newline at the end, no more, no less
	// Remove ALL newlines and carriage returns, then add exactly one newline
	logLine = strings.TrimRight(logLine, "\n\r \t")
	if logLine != "" {
		logLine = logLine + "\n"
		output.Write([]byte(logLine))
	}
}

// Error logs an error message
func (l *Logger) Error(args ...interface{}) {
	l.log(ErrorLevel, "%v", args...)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(args ...interface{}) {
	l.log(WarnLevel, "%v", args...)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Info logs an info message
func (l *Logger) Info(args ...interface{}) {
	l.log(InfoLevel, "%v", args...)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(args ...interface{}) {
	l.log(DebugLevel, "%v", args...)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// formatMessage formats the message with arguments
// Removes any trailing newlines to prevent blank lines in output
func formatMessage(format string, args ...interface{}) string {
	var msg string
	if len(args) == 0 {
		msg = format
	} else {
		msg = fmt.Sprintf(format, args...)
	}
	// Remove any trailing newlines or carriage returns to prevent blank lines
	msg = strings.TrimRight(msg, "\n\r")
	// Replace any internal newlines with spaces to keep everything on one line
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, "\r", " ")
	return msg
}

// formatLogLine formats a log line in standard style (deprecated, use formatStandardLine)
// Kept for backward compatibility
func formatLogLine(timestamp, level, scope, message string) string {
	return formatStandardLine(timestamp, level, scope, message)
}

// EnablePrettyLogging enables pretty logging with colors and better formatting
func EnablePrettyLogging() {
	prettyLogMu.Lock()
	defer prettyLogMu.Unlock()
	usePrettyLog = true
	GetPrettyFormatter()
}

// DisablePrettyLogging disables pretty logging, reverts to standard Istio format
func DisablePrettyLogging() {
	prettyLogMu.Lock()
	defer prettyLogMu.Unlock()
	usePrettyLog = false
}

// IsPrettyLoggingEnabled returns whether pretty logging is enabled
func IsPrettyLoggingEnabled() bool {
	prettyLogMu.RLock()
	defer prettyLogMu.RUnlock()
	return usePrettyLog
}

// getDefaultLogger returns the default logger instance
func getDefaultLogger() *Logger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = RegisterScope(defaultScope, "Default logging scope")
	})
	return defaultLogger
}

// SetDefaultScope sets the default scope name for package-level logging functions
// This should be called before any package-level logging functions are used
func SetDefaultScope(name string) {
	globalMu.Lock()
	defer globalMu.Unlock()
	defaultScope = name
	// Reset the default logger to use the new scope
	defaultLogger = RegisterScope(name, "Default logging scope")
}

// Package-level logging functions using the default scope

// Info logs an info message using the default scope
func Info(args ...interface{}) {
	getDefaultLogger().Info(args...)
}

// Infof logs a formatted info message using the default scope
func Infof(format string, args ...interface{}) {
	getDefaultLogger().Infof(format, args...)
}

// Warn logs a warning message using the default scope
func Warn(args ...interface{}) {
	getDefaultLogger().Warn(args...)
}

// Warnf logs a formatted warning message using the default scope
func Warnf(format string, args ...interface{}) {
	getDefaultLogger().Warnf(format, args...)
}

// Debug logs a debug message using the default scope
// Debug level is the lowest log level (most verbose)
func Debug(args ...interface{}) {
	getDefaultLogger().Debug(args...)
}

// Debugf logs a formatted debug message using the default scope
// Debug level is the lowest log level (most verbose)
func Debugf(format string, args ...interface{}) {
	getDefaultLogger().Debugf(format, args...)
}

// Error logs an error message using the default scope
func Error(args ...interface{}) {
	getDefaultLogger().Error(args...)
}

// Errorf logs a formatted error message using the default scope
func Errorf(format string, args ...interface{}) {
	getDefaultLogger().Errorf(format, args...)
}
