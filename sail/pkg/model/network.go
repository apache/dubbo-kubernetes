package model

type NetworkManager struct {
	env        *Environment
	xdsUpdater XDSUpdater
}

func NewNetworkManager(env *Environment, xdsUpdater XDSUpdater) (*NetworkManager, error) {
	mgr := &NetworkManager{
		env:        env,
		xdsUpdater: xdsUpdater,
	}
	return mgr, nil
}
