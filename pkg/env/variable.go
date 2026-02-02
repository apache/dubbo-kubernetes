package env

import "os"

// Variable is a wrapper for an environment variable.
type Variable string

// Name of the environment variable.
func (e Variable) Name() string {
	return string(e)
}

// Value of the environment variable.
func (e Variable) Value() string {
	return os.Getenv(e.Name())
}

// ValueOrDefault returns the value of the environment variable if it is non-empty. Otherwise returns the value provided.
func (e Variable) ValueOrDefault(defaultValue string) string {
	return e.ValueOrDefaultFunc(func() string {
		return defaultValue
	})
}

// ValueOrDefaultFunc returns the value of the environment variable if it is non-empty. Otherwise returns the value function provided.
func (e Variable) ValueOrDefaultFunc(defaultValueFunc func() string) string {
	if value := e.Value(); value != "" {
		return value
	}
	return defaultValueFunc()
}
