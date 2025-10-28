package xds

import (
	"context"
)

func (s *DiscoveryServer) authenticate(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (s *DiscoveryServer) authorize(con *Connection, identities []string) error {
	if con == nil || con.proxy == nil {
		return nil
	}
	return nil
}
