package rules

import (
	"encoding"
	"fmt"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

const RuleMatchesHashTag = "__rule-matches-hash__"

type InboundListener struct {
	Address string
	Port    uint32
}

// We need to implement TextMarshaler because InboundListener is used
// as a key for maps that are JSON encoded for logging.
var _ encoding.TextMarshaler = InboundListener{}

func (i InboundListener) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

func (i InboundListener) String() string {
	return fmt.Sprintf("%s:%d", i.Address, i.Port)
}

type FromRules struct {
	Rules map[InboundListener]Rules
}

type ToRules struct {
	Rules Rules
}

type GatewayRules struct {
	ToRules   map[InboundListener]Rules
	FromRules map[InboundListener]Rules
}

type SingleItemRules struct {
	Rules Rules
}

type PolicyItemWithMeta struct {
	core_model.PolicyItem
	core_model.ResourceMeta
}

// Tag is a key-value pair. If Not is true then Key != Value
type Tag struct {
	Key   string
	Value string
	Not   bool
}

// Subset represents a group of proxies
type Subset []Tag

// IsSubset returns true if 'other' is a subset of the current set.
// Empty set is a superset for all subsets.
func (ss Subset) IsSubset(other Subset) bool {
	if len(ss) == 0 {
		return true
	}
	otherByKeys := map[string][]Tag{}
	for _, t := range other {
		otherByKeys[t.Key] = append(otherByKeys[t.Key], t)
	}
	for _, tag := range ss {
		oTags, ok := otherByKeys[tag.Key]
		if !ok {
			return false
		}
		for _, otherTag := range oTags {
			if !isSubset(tag, otherTag) {
				return false
			}
		}
	}
	return true
}

func isSubset(t1, t2 Tag) bool {
	switch {
	// t2={y: b} can't be a subset of t1={x: a} because point {y: b, x: c} belongs to t2, but doesn't belong to t1
	case t1.Key != t2.Key:
		return false

	// t2={y: !a} is a subset of t1={y: !b} if and only if a == b
	case t1.Not == t2.Not:
		return t1.Value == t2.Value

	// t2={y: a} is a subset of t1={y: !b} if and only if a != b
	case t1.Not:
		return t1.Value != t2.Value

	// t2={y: !a} can't be a subset of t1={y: b} because point {y: c} belongs to t2, but doesn't belong to t1
	case t2.Not:
		return false

	default:
		panic("impossible")
	}
}

// Intersect returns true if there exists an element that belongs both to 'other' and current set.
// Empty set intersects with all sets.
func (ss Subset) Intersect(other Subset) bool {
	if len(ss) == 0 || len(other) == 0 {
		return true
	}
	otherByKeysOnlyPositive := map[string][]Tag{}
	for _, t := range other {
		if t.Not {
			continue
		}
		otherByKeysOnlyPositive[t.Key] = append(otherByKeysOnlyPositive[t.Key], t)
	}
	for _, tag := range ss {
		if tag.Not {
			continue
		}
		oTags, ok := otherByKeysOnlyPositive[tag.Key]
		if !ok {
			return true
		}
		for _, otherTag := range oTags {
			if otherTag != tag {
				return false
			}
		}
	}
	return true
}

func (ss Subset) WithTag(key, value string, not bool) Subset {
	return append(ss, Tag{Key: key, Value: value, Not: not})
}

func MeshSubset() Subset {
	return Subset{}
}

func MeshService(name string) Subset {
	return Subset{{
		Key: mesh_proto.ServiceTag, Value: name,
	}}
}

func SubsetFromTags(tags map[string]string) Subset {
	subset := Subset{}
	for k, v := range tags {
		subset = append(subset, Tag{Key: k, Value: v})
	}
	return subset
}

// NumPositive returns a number of tags without negation
func (ss Subset) NumPositive() int {
	pos := 0
	for _, t := range ss {
		if !t.Not {
			pos++
		}
	}
	return pos
}

func (ss Subset) IndexOfPositive() int {
	for i, t := range ss {
		if !t.Not {
			return i
		}
	}
	return -1
}

// Rule contains a configuration for the given Subset. When rule is an inbound rule (from),
// then Subset represents a group of clients. When rule is an outbound (to) then Subset
// represents destinations.
type Rule struct {
	Subset Subset
	Conf   interface{}
	Origin []core_model.ResourceMeta
}

type Rules []*Rule
