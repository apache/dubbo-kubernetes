package access

import (
	"context"

	"github.com/apache/dubbo-kubernetes/pkg/core/user"
)

type EnvoyAdminAccess interface {
	ValidateViewConfigDump(ctx context.Context, user user.User) error
	ValidateViewStats(ctx context.Context, user user.User) error
	ValidateViewClusters(ctx context.Context, user user.User) error
}
