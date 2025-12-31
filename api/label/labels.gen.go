
// GENERATED FILE -- DO NOT EDIT

package label

type FeatureStatus int

const (
	Alpha FeatureStatus = iota
	Beta
	Stable
)

func (s FeatureStatus) String() string {
	switch s {
	case Alpha:
		return "Alpha"
	case Beta:
		return "Beta"
	case Stable:
		return "Stable"
	}
	return "Unknown"
}

type ResourceTypes int

const (
	Unknown ResourceTypes = iota
    Any
    Deployment
    Gateway
    HorizontalPodAutoscaler
    Namespace
    Pod
    PodDisruptionBudget
    Service
    ServiceAccount
)

func (r ResourceTypes) String() string {
	switch r {
	case 1:
		return "Any"
	case 2:
		return "Deployment"
	case 3:
		return "Gateway"
	case 4:
		return "HorizontalPodAutoscaler"
	case 5:
		return "Namespace"
	case 6:
		return "Pod"
	case 7:
		return "PodDisruptionBudget"
	case 8:
		return "Service"
	case 9:
		return "ServiceAccount"
	}
	return "Unknown"
}

// Instance describes a single resource label
type Instance struct {
	// The name of the label.
	Name string

	// Description of the label.
	Description string

	// FeatureStatus of this label.
	FeatureStatus FeatureStatus

	// Hide the existence of this label when outputting usage information.
	Hidden bool

	// Mark this label as deprecated when generating usage information.
	Deprecated bool

	// The types of resources this label applies to.
	Resources []ResourceTypes
}

var (

	OrgApacheDubboRev = Instance {
		Name:          "dubbo.apache.org/rev",
		Description:   "Dubbo control plane revision or tag associated with the "+
                        "resource; e.g. `canary`",
		FeatureStatus: Beta,
		Hidden:        false,
		Deprecated:    false,
		Resources: []ResourceTypes{
			Namespace,
			Gateway,
			Pod,
		},
	}

	IoK8sNetworkingGatewayGatewayClassName = Instance {
		Name:          "gateway.networking.k8s.io/gateway-class-name",
		Description:   "Automatically added to all resources [automatically "+
                        "created](/docs/tasks/traffic-management/ingress/gateway-api/#automated-deployment) "+
                        "by Dubbo Gateway controller to indicate which "+
                        "`GatewayClass` resulted in the object creation. Users "+
                        "should not set this label themselves.",
		FeatureStatus: Stable,
		Hidden:        false,
		Deprecated:    false,
		Resources: []ResourceTypes{
			ServiceAccount,
			Deployment,
			Service,
			PodDisruptionBudget,
			HorizontalPodAutoscaler,
		},
	}

	IoK8sNetworkingGatewayGatewayName = Instance {
		Name:          "gateway.networking.k8s.io/gateway-name",
		Description:   "Automatically added to all resources [automatically "+
                        "created](/docs/tasks/traffic-management/ingress/gateway-api/#automated-deployment) "+
                        "by Dubbo Gateway controller to indicate which `Gateway` "+
                        "resulted in the object creation. Users should not set "+
                        "this label themselves.",
		FeatureStatus: Stable,
		Hidden:        false,
		Deprecated:    false,
		Resources: []ResourceTypes{
			ServiceAccount,
			Deployment,
			Service,
			PodDisruptionBudget,
			HorizontalPodAutoscaler,
		},
	}

	OrgApacheDubboOperatorComponent = Instance {
		Name:          "operator.dubbo.apache.org/component",
		Description:   "Dubbo operator component name of the resource, e.g. "+
                        "`Pilot`",
		FeatureStatus: Alpha,
		Hidden:        true,
		Deprecated:    false,
		Resources: []ResourceTypes{
			Any,
		},
	}

	OrgApacheDubboOperatorManaged = Instance {
		Name:          "operator.dubbo.apache.org/managed",
		Description:   "Set to `Reconcile` if the Dubbo operator will reconcile "+
                        "the resource.",
		FeatureStatus: Alpha,
		Hidden:        true,
		Deprecated:    false,
		Resources: []ResourceTypes{
			Any,
		},
	}

	OrgApacheDubboOperatorVersion = Instance {
		Name:          "operator.dubbo.apache.org/version",
		Description:   "The Dubbo operator version that installed the resource, "+
                        "e.g. `1.6.0`",
		FeatureStatus: Alpha,
		Hidden:        true,
		Deprecated:    false,
		Resources: []ResourceTypes{
			Any,
		},
	}

	OrgApacheDubboProxylessInject = Instance {
		Name:          "proxyless.dubbo.apache.org/inject",
		Description:   "Specifies whether or not an proxyless adapter should be "+
                        "automatically injected into the workload.",
		FeatureStatus: Alpha,
		Hidden:        false,
		Deprecated:    false,
		Resources: []ResourceTypes{
			Pod,
		},
	}

)

func AllResourceLabels() []*Instance {
	return []*Instance {
		&OrgApacheDubboRev,
		&IoK8sNetworkingGatewayGatewayClassName,
		&IoK8sNetworkingGatewayGatewayName,
		&OrgApacheDubboOperatorComponent,
		&OrgApacheDubboOperatorManaged,
		&OrgApacheDubboOperatorVersion,
		&OrgApacheDubboProxylessInject,
	}
}

func AllResourceTypes() []string {
	return []string {
		"Any",
		"Deployment",
		"Gateway",
		"HorizontalPodAutoscaler",
		"Namespace",
		"Pod",
		"PodDisruptionBudget",
		"Service",
		"ServiceAccount",
	}
}
