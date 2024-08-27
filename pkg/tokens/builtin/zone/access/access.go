package access

import (
	"context"

	"github.com/apache/dubbo-kubernetes/pkg/core/user"
)

type ZoneTokenAccess interface {
	ValidateGenerateZoneToken(ctx context.Context, zone string, user user.User) error
}
