package access

import "github.com/apache/dubbo-kubernetes/pkg/core/user"

type GenerateUserTokenAccess interface {
	ValidateGenerate(user user.User) error
}
