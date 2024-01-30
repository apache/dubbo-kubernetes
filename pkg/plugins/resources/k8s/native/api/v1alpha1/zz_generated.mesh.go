// Generated by tools/resource-gen
// Run "make generate" to update this file.

// nolint:whitespace
package v1alpha1

import (
	"fmt"
)

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/registry"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=dubbo,scope=Cluster
type Dataplane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Mesh is the name of the dubbo mesh this resource belongs to.
	// It may be omitted for cluster-scoped resources.
	//
	// +kubebuilder:validation:Optional
	Mesh string `json:"mesh,omitempty"`
	// Spec is the specification of the Dubbo Dataplane resource.
	// +kubebuilder:validation:Optional
	Spec *apiextensionsv1.JSON `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
type DataplaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dataplane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dataplane{}, &DataplaneList{})
}

func (cb *Dataplane) GetObjectMeta() *metav1.ObjectMeta {
	return &cb.ObjectMeta
}

func (cb *Dataplane) SetObjectMeta(m *metav1.ObjectMeta) {
	cb.ObjectMeta = *m
}

func (cb *Dataplane) GetMesh() string {
	return cb.Mesh
}

func (cb *Dataplane) SetMesh(mesh string) {
	cb.Mesh = mesh
}

func (cb *Dataplane) GetSpec() (core_model.ResourceSpec, error) {
	spec := cb.Spec
	m := mesh_proto.Dataplane{}

	if spec == nil || len(spec.Raw) == 0 {
		return &m, nil
	}

	err := util_proto.FromJSON(spec.Raw, &m)
	return &m, err
}

func (cb *Dataplane) SetSpec(spec core_model.ResourceSpec) {
	if spec == nil {
		cb.Spec = nil
		return
	}

	s, ok := spec.(*mesh_proto.Dataplane)
	if !ok {
		panic(fmt.Sprintf("unexpected protobuf message type %T", spec))
	}

	cb.Spec = &apiextensionsv1.JSON{Raw: util_proto.MustMarshalJSON(s)}
}

func (cb *Dataplane) Scope() model.Scope {
	return model.ScopeCluster
}

func (l *DataplaneList) GetItems() []model.KubernetesObject {
	result := make([]model.KubernetesObject, len(l.Items))
	for i := range l.Items {
		result[i] = &l.Items[i]
	}
	return result
}

func init() {
	registry.RegisterObjectType(&mesh_proto.Dataplane{}, &Dataplane{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "Dataplane",
		},
	})
	registry.RegisterListType(&mesh_proto.Dataplane{}, &DataplaneList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "DataplaneList",
		},
	})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=dubbo,scope=Namespaced
type DataplaneInsight struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Mesh is the name of the dubbo mesh this resource belongs to.
	// It may be omitted for cluster-scoped resources.
	//
	// +kubebuilder:validation:Optional
	Mesh string `json:"mesh,omitempty"`
	// Status is the status the dubbo resource.
	// +kubebuilder:validation:Optional
	Status *apiextensionsv1.JSON `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type DataplaneInsightList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataplaneInsight `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataplaneInsight{}, &DataplaneInsightList{})
}

func (cb *DataplaneInsight) GetObjectMeta() *metav1.ObjectMeta {
	return &cb.ObjectMeta
}

func (cb *DataplaneInsight) SetObjectMeta(m *metav1.ObjectMeta) {
	cb.ObjectMeta = *m
}

func (cb *DataplaneInsight) GetMesh() string {
	return cb.Mesh
}

func (cb *DataplaneInsight) SetMesh(mesh string) {
	cb.Mesh = mesh
}

func (cb *DataplaneInsight) GetSpec() (core_model.ResourceSpec, error) {
	spec := cb.Status
	m := mesh_proto.DataplaneInsight{}

	if spec == nil || len(spec.Raw) == 0 {
		return &m, nil
	}

	err := util_proto.FromJSON(spec.Raw, &m)
	return &m, err
}

func (cb *DataplaneInsight) SetSpec(spec core_model.ResourceSpec) {
	if spec == nil {
		cb.Status = nil
		return
	}

	s, ok := spec.(*mesh_proto.DataplaneInsight)
	if !ok {
		panic(fmt.Sprintf("unexpected protobuf message type %T", spec))
	}

	cb.Status = &apiextensionsv1.JSON{Raw: util_proto.MustMarshalJSON(s)}
}

func (cb *DataplaneInsight) Scope() model.Scope {
	return model.ScopeNamespace
}

func (l *DataplaneInsightList) GetItems() []model.KubernetesObject {
	result := make([]model.KubernetesObject, len(l.Items))
	for i := range l.Items {
		result[i] = &l.Items[i]
	}
	return result
}

func init() {
	registry.RegisterObjectType(&mesh_proto.DataplaneInsight{}, &DataplaneInsight{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "DataplaneInsight",
		},
	})
	registry.RegisterListType(&mesh_proto.DataplaneInsight{}, &DataplaneInsightList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "DataplaneInsightList",
		},
	})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=dubbo,scope=Cluster
type Mesh struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Mesh is the name of the dubbo mesh this resource belongs to.
	// It may be omitted for cluster-scoped resources.
	//
	// +kubebuilder:validation:Optional
	Mesh string `json:"mesh,omitempty"`
	// Spec is the specification of the Dubbo Mesh resource.
	// +kubebuilder:validation:Optional
	Spec *apiextensionsv1.JSON `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
type MeshList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mesh `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Mesh{}, &MeshList{})
}

func (cb *Mesh) GetObjectMeta() *metav1.ObjectMeta {
	return &cb.ObjectMeta
}

func (cb *Mesh) SetObjectMeta(m *metav1.ObjectMeta) {
	cb.ObjectMeta = *m
}

func (cb *Mesh) GetMesh() string {
	return cb.Mesh
}

func (cb *Mesh) SetMesh(mesh string) {
	cb.Mesh = mesh
}

func (cb *Mesh) GetSpec() (core_model.ResourceSpec, error) {
	spec := cb.Spec
	m := mesh_proto.Mesh{}

	if spec == nil || len(spec.Raw) == 0 {
		return &m, nil
	}

	err := util_proto.FromJSON(spec.Raw, &m)
	return &m, err
}

func (cb *Mesh) SetSpec(spec core_model.ResourceSpec) {
	if spec == nil {
		cb.Spec = nil
		return
	}

	s, ok := spec.(*mesh_proto.Mesh)
	if !ok {
		panic(fmt.Sprintf("unexpected protobuf message type %T", spec))
	}

	cb.Spec = &apiextensionsv1.JSON{Raw: util_proto.MustMarshalJSON(s)}
}

func (cb *Mesh) Scope() model.Scope {
	return model.ScopeCluster
}

func (l *MeshList) GetItems() []model.KubernetesObject {
	result := make([]model.KubernetesObject, len(l.Items))
	for i := range l.Items {
		result[i] = &l.Items[i]
	}
	return result
}

func init() {
	registry.RegisterObjectType(&mesh_proto.Mesh{}, &Mesh{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "Mesh",
		},
	})
	registry.RegisterListType(&mesh_proto.Mesh{}, &MeshList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "MeshList",
		},
	})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=dubbo,scope=Cluster
type ZoneIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Mesh is the name of the dubbo mesh this resource belongs to.
	// It may be omitted for cluster-scoped resources.
	//
	// +kubebuilder:validation:Optional
	Mesh string `json:"mesh,omitempty"`
	// Spec is the specification of the Dubbo ZoneIngress resource.
	// +kubebuilder:validation:Optional
	Spec *apiextensionsv1.JSON `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
type ZoneIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZoneIngress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZoneIngress{}, &ZoneIngressList{})
}

func (cb *ZoneIngress) GetObjectMeta() *metav1.ObjectMeta {
	return &cb.ObjectMeta
}

func (cb *ZoneIngress) SetObjectMeta(m *metav1.ObjectMeta) {
	cb.ObjectMeta = *m
}

func (cb *ZoneIngress) GetMesh() string {
	return cb.Mesh
}

func (cb *ZoneIngress) SetMesh(mesh string) {
	cb.Mesh = mesh
}

func (cb *ZoneIngress) GetSpec() (core_model.ResourceSpec, error) {
	spec := cb.Spec
	m := mesh_proto.ZoneIngress{}

	if spec == nil || len(spec.Raw) == 0 {
		return &m, nil
	}

	err := util_proto.FromJSON(spec.Raw, &m)
	return &m, err
}

func (cb *ZoneIngress) SetSpec(spec core_model.ResourceSpec) {
	if spec == nil {
		cb.Spec = nil
		return
	}

	s, ok := spec.(*mesh_proto.ZoneIngress)
	if !ok {
		panic(fmt.Sprintf("unexpected protobuf message type %T", spec))
	}

	cb.Spec = &apiextensionsv1.JSON{Raw: util_proto.MustMarshalJSON(s)}
}

func (cb *ZoneIngress) Scope() model.Scope {
	return model.ScopeCluster
}

func (l *ZoneIngressList) GetItems() []model.KubernetesObject {
	result := make([]model.KubernetesObject, len(l.Items))
	for i := range l.Items {
		result[i] = &l.Items[i]
	}
	return result
}

func init() {
	registry.RegisterObjectType(&mesh_proto.ZoneIngress{}, &ZoneIngress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "ZoneIngress",
		},
	})
	registry.RegisterListType(&mesh_proto.ZoneIngress{}, &ZoneIngressList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "ZoneIngressList",
		},
	})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=dubbo,scope=Namespaced
type ZoneIngressInsight struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Mesh is the name of the dubbo mesh this resource belongs to.
	// It may be omitted for cluster-scoped resources.
	//
	// +kubebuilder:validation:Optional
	Mesh string `json:"mesh,omitempty"`
	// Spec is the specification of the Dubbo ZoneIngressInsight resource.
	// +kubebuilder:validation:Optional
	Spec *apiextensionsv1.JSON `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type ZoneIngressInsightList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZoneIngressInsight `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZoneIngressInsight{}, &ZoneIngressInsightList{})
}

func (cb *ZoneIngressInsight) GetObjectMeta() *metav1.ObjectMeta {
	return &cb.ObjectMeta
}

func (cb *ZoneIngressInsight) SetObjectMeta(m *metav1.ObjectMeta) {
	cb.ObjectMeta = *m
}

func (cb *ZoneIngressInsight) GetMesh() string {
	return cb.Mesh
}

func (cb *ZoneIngressInsight) SetMesh(mesh string) {
	cb.Mesh = mesh
}

func (cb *ZoneIngressInsight) GetSpec() (core_model.ResourceSpec, error) {
	spec := cb.Spec
	m := mesh_proto.ZoneIngressInsight{}

	if spec == nil || len(spec.Raw) == 0 {
		return &m, nil
	}

	err := util_proto.FromJSON(spec.Raw, &m)
	return &m, err
}

func (cb *ZoneIngressInsight) SetSpec(spec core_model.ResourceSpec) {
	if spec == nil {
		cb.Spec = nil
		return
	}

	s, ok := spec.(*mesh_proto.ZoneIngressInsight)
	if !ok {
		panic(fmt.Sprintf("unexpected protobuf message type %T", spec))
	}

	cb.Spec = &apiextensionsv1.JSON{Raw: util_proto.MustMarshalJSON(s)}
}

func (cb *ZoneIngressInsight) Scope() model.Scope {
	return model.ScopeNamespace
}

func (l *ZoneIngressInsightList) GetItems() []model.KubernetesObject {
	result := make([]model.KubernetesObject, len(l.Items))
	for i := range l.Items {
		result[i] = &l.Items[i]
	}
	return result
}

func init() {
	registry.RegisterObjectType(&mesh_proto.ZoneIngressInsight{}, &ZoneIngressInsight{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "ZoneIngressInsight",
		},
	})
	registry.RegisterListType(&mesh_proto.ZoneIngressInsight{}, &ZoneIngressInsightList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "ZoneIngressInsightList",
		},
	})
}
