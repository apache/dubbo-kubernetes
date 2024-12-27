package log

const (
	DefaultScopeName       = "default"
	defaultStackTraceLevel = NoneLevel
	defaultOutputPath      = "stdout"
	defaultErrorOutputPath = "stderr"
)

type Level int

const (
	// NoneLevel disables logging
	NoneLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)
