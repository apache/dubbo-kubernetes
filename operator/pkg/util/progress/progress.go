package progress

type InstallState int

const (
	StateInstalling InstallState = iota
	StatePruning
	StateComplete
	StateUninstallComplete
)

const inProgress = `{{ yellow (cycle . "-" "-" " ") }}`
