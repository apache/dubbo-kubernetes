package apis

import (
	"encoding/json"
	proto "github.com/gogo/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubetype-gen
// +kubetype-gen:groupVersion=install.dubbo.io/v1alpha1
// +k8s:deepcopy-gen=true
type DubboOperator struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec DubboOperatorSpec `json:"spec,omitempty"`
}

type DubboOperatorSpec struct {
	Profile    string                   `json:"profile,omitempty"`
	Control    string                   `json:"control,omitempty"`
	Dashboard  *DubboAdminDashboardSpec `json:"dashboard,omitempty"`
	Components *DubboComponentSpec      `json:"components,omitempty"`
	Values     json.RawMessage          `json:"values,omitempty"`
}

type DubboComponentSpec struct {
	Base     *BaseComponentSpec `json:"base,omitempty"`
	Register *RegisterSpec      `json:"register,omitempty"`
}

type BaseComponentSpec struct {
	Enabled *BoolValue `json:"enabled,omitempty"`
}

type DubboAdminDashboardSpec struct {
	Admin *ComponentSpec `json:"admin,omitempty"`
}

type RegisterSpec struct {
	Nacos     *RegisterComponentSpec `json:"nacos,omitempty"`
	Zookeeper *RegisterComponentSpec `json:"zookeeper,omitempty"`
}

type RegisterComponentSpec struct {
	Enabled *BoolValue `json:"enabled,omitempty"`
}

type ComponentSpec struct {
	Enabled   *BoolValue     `json:"enabled,omitempty"`
	Namespace string         `json:"namespace,omitempty"`
	Raw       map[string]any `json:"-"`
}

type MetadataCompSpec struct {
	ComponentSpec
}

type BoolValue struct {
	bool
}

func (d *DubboOperator) Reset() { *d = DubboOperator{} }

func (d *DubboOperator) String() string { return proto.CompactTextString(d) }

func (*DubboOperator) ProtoMessage() {}

func (b *BoolValue) MarshalJSON() ([]byte, error) { return json.Marshal(b.GetValueOrFalse()) }

func (b *BoolValue) UnmarshalJSON(bytes []byte) error {
	bb := false
	if err := json.Unmarshal(bytes, &bb); err != nil {
		return err
	}
	*b = BoolValue{bb}
	return nil
}

func (b *BoolValue) GetValueOrFalse() bool {
	if b == nil {
		return false
	}
	return b.bool
}

func (b *BoolValue) GetValueOrTrue() bool {
	if b == nil {
		return true
	}
	return b.bool
}

var (
	_ json.Unmarshaler = &BoolValue{}
	_ json.Marshaler   = &BoolValue{}
)
