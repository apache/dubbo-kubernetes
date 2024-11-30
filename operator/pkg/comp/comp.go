package comp

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/apis"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
)

type Name string

const (
	BaseComponentName Name = "Base"
)

type Comp struct {
	UserFacingName Name
	SpecName       string
	Default        bool
	HelmSubDir     string
	HelmTreeRoot   string
}

var AllComps = []Comp{
	{
		UserFacingName: BaseComponentName,
		SpecName:       "base",
		Default:        true,
		HelmSubDir:     "base",
		HelmTreeRoot:   "global",
	},
}

var (
	userFacingCompNames = map[Name]string{
		BaseComponentName: "Dubbo Core",
	}
	Icons = map[Name]string{
		BaseComponentName: "🚄",
	}
)

func UserFacingCompName(name Name) string {
	s, ok := userFacingCompNames[name]
	if !ok {
		return "Unknown"
	}
	return s
}

func (c Comp) Get(merged values.Map) ([]apis.MetadataCompSpec, error) {
	return nil, nil
}
