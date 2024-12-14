package log

var defaultScope = registerDefaultScopes()

func registerDefaultScopes() (defaults *Scope) {
	return registerScope(DefaultScopeName, "Unscoped logging messages.", 1)
}
