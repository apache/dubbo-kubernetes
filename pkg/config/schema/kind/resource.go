package kind

const (
	Unknown Kind = iota
	AuthorizationPolicy
	CustomResourceDefinition
	DestinationRule
	MeshConfig
	MeshNetworks
	MutatingWebhookConfiguration
	Namespace
	PeerAuthentication
	Pod
	RequestAuthentication
	Secret
	Service
	ServiceAccount
	StatefulSet
	ValidatingWebhookConfiguration
	VirtualService
)

func (k Kind) String() string {
	switch k {
	case AuthorizationPolicy:
		return "AuthorizationPolicy"
	case CustomResourceDefinition:
		return "CustomResourceDefinition"
	case DestinationRule:
		return "DestinationRule"
	case MeshConfig:
		return "MeshConfig"
	case MeshNetworks:
		return "MeshNetworks"
	case MutatingWebhookConfiguration:
		return "MutatingWebhookConfiguration"
	case Namespace:
		return "Namespace"
	case PeerAuthentication:
		return "PeerAuthentication"
	case Pod:
		return "Pod"
	case RequestAuthentication:
		return "RequestAuthentication"
	case Secret:
		return "Secret"
	case Service:
		return "Service"
	case ServiceAccount:
		return "ServiceAccount"
	case StatefulSet:
		return "StatefulSet"
	case ValidatingWebhookConfiguration:
		return "ValidatingWebhookConfiguration"
	case VirtualService:
		return "VirtualService"
	default:
		return "Unknown"
	}
}
