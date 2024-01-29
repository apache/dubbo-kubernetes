// +kubebuilder:object:generate=true
package v1alpha1

// ServiceNameMapping
// +dubbo:policy:skip_registration=fase
type ServiceNameMapping struct {
	Namespace        string   `json:"namespace"`
	InterfaceName    string   `json:"interfaceName"`
	ApplicationNames []string `json:"applicationNames"`
}
