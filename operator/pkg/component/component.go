package component

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/apis"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
)

type Name string

const (
	BaseComponentName              Name = "Base"
	AdminComponentName             Name = "Admin"
	NacosRegisterComponentName     Name = "Nacos"
	ZookeeperRegisterComponentName Name = "Zookeeper"
)

type Component struct {
	UserFacingName     Name
	SpecName           string
	ResourceType       string
	ResourceName       string
	Default            bool
	HelmSubDir         string
	HelmValuesTreeRoot string
	FlattenValues      bool
}

var AllComponents = []Component{
	{
		UserFacingName:     BaseComponentName,
		SpecName:           "base",
		Default:            true,
		HelmSubDir:         "base",
		HelmValuesTreeRoot: "global",
	},
	{
		UserFacingName:     AdminComponentName,
		SpecName:           "admin",
		ResourceType:       "Deployment",
		Default:            true,
		HelmSubDir:         "admin",
		HelmValuesTreeRoot: "admin",
	},
	{
		UserFacingName:     NacosRegisterComponentName,
		SpecName:           "nacos",
		ResourceType:       "StatefulSet",
		ResourceName:       "register",
		Default:            true,
		HelmSubDir:         "dubbo-control/register-discovery/nacos",
		HelmValuesTreeRoot: "nacos",
	},
	{
		UserFacingName:     ZookeeperRegisterComponentName,
		SpecName:           "zookeeper",
		ResourceType:       "StatefulSet",
		ResourceName:       "register",
		Default:            true,
		HelmSubDir:         "dubbo-control/register-discovery/zookeeper",
		HelmValuesTreeRoot: "zookeeper",
	},
}

var (
	userFacingCompNames = map[Name]string{
		BaseComponentName:          "Dubbo Core",
		AdminComponentName:         "Dubbo Dashboard",
		NacosRegisterComponentName: "Dubbo Nacos Register Plane",
	}

	Icons = map[Name]string{
		BaseComponentName: "🛸",
		// TODO DubbodComponentName: "📡",
		NacosRegisterComponentName:     "🪝",
		ZookeeperRegisterComponentName: "⚓",
		AdminComponentName:             "🛰",
	}
)

func UserFacingCompName(name Name) string {
	s, ok := userFacingCompNames[name]
	if !ok {
		return "Unknown"
	}
	return s
}

func (c Component) Get(merged values.Map) ([]apis.MetadataCompSpec, error) {
	defaultNamespace := merged.GetPathString("metadata.namespace")
	var defaultResp []apis.MetadataCompSpec
	def := c.Default
	if def {
		defaultResp = []apis.MetadataCompSpec{{
			ComponentSpec: apis.ComponentSpec{
				Namespace: defaultNamespace,
			}},
		}
	}
	buildSpec := func(m values.Map) (apis.MetadataCompSpec, error) {
		spec, err := values.ConvertMap[apis.MetadataCompSpec](m)
		if err != nil {
			return apis.MetadataCompSpec{}, fmt.Errorf("fail to convert %v: %v", c.SpecName, err)
		}

		if spec.Namespace == "" {
			spec.Namespace = defaultNamespace
		}
		if spec.Namespace == "" {
			spec.Namespace = "dubbo-system"
		}

		spec.Raw = m
		return spec, nil
	}
	s, ok := merged.GetPathMap("spec.components." + c.SpecName)
	if !ok {
		return defaultResp, nil
	}
	spec, err := buildSpec(s)
	if err != nil {
		return nil, err
	}
	if !(spec.Enabled.GetValueOrTrue()) {
		return nil, nil
	}
	return []apis.MetadataCompSpec{spec}, nil
}
