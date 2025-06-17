// +kubebuilder:object:generate=true
// +groupName=mesh.dubbo.apache.org
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	meshproto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
)


// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ConditionRouteResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   meshproto.ConditionRoute       `json:"spec,omitempty"`
	Status ConditionRouteStatus `json:"status,omitempty"`
}

type ConditionRouteStatus struct {

}

// +kubebuilder:object:root=true
type ConditionRouteResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConditionRouteResource `json:"items"`
}

func (c *ConditionRouteResource) GetTypeMeta() metav1.TypeMeta {
	return c.TypeMeta
}

func (c *ConditionRouteResource) GetObjectMeta() metav1.ObjectMeta {
	return c.ObjectMeta
}

func (c *ConditionRouteResource) GetSpec() interface{} {

	copySpec := c.Spec.DeepCopy()
	return copySpec
}

func (c *ConditionRouteResource) GetStatus() interface{} {
	return c.Status
}
