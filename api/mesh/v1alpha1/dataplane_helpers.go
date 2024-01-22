package v1alpha1

const (
	KubeNamespaceTag = "k8s.dubbo.io/namespace"
)

const (
	ServiceTag = "dubbo.io/service"
	// Locality related tags
	ZoneTag = "dubbo.io/zone"

	// ResourceOriginLabel is a standard label that has information about the origin of the resource.
	// It can be either "global" or "zone".
	ResourceOriginLabel  = "dubbo.io/origin"
	ResourceOriginGlobal = "global"
	ResourceOriginZone   = "zone"

	// DisplayName is a standard label that can be used to easier recognize policy name.
	// On Kubernetes, Dubbo resource name contains namespace. Display name is original name without namespace.
	// The name contains hash when the resource is synced from global to zone. In this case, display name is original name from originated CP.
	DisplayName = "dubbo.io/display-name"
)

type ProxyType string

const (
	DataplaneProxyType ProxyType = "dataplane"
	IngressProxyType   ProxyType = "ingress"
	EgressProxyType    ProxyType = "egress"
)
