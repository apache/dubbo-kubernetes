package kind

const (
	Unknown Kind = iota
	Address
	CustomResourceDefinition
	MeshConfig
	MeshNetworks
	Namespace
	Pod
	Secret
	Service
	ServiceAccount
	StatefulSet
	ValidatingWebhookConfiguration
	MutatingWebhookConfiguration
	PeerAuthentication
	RequestAuthentication
	VirtualService
	DestinationRule
)

func (k Kind) String() string {
	switch k {
	case Address:
		return "Address"
	case CustomResourceDefinition:
		return "CustomResourceDefinition"
	case MeshConfig:
		return "MeshConfig"
	case MeshNetworks:
		return "MeshNetworks"
	case Namespace:
		return "Namespace"
	case Pod:
		return "Pod"
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
	case MutatingWebhookConfiguration:
		return "MutatingWebhookConfiguration"
	case PeerAuthentication:
		return "PeerAuthentication"
	case RequestAuthentication:
		return "RequestAuthentication"
	case VirtualService:
		return "VirtualService"
	case DestinationRule:
		return "DestinationRule"
	default:
		return "Unknown"
	}
}
