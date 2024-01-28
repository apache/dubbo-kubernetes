package tags

import (
	"fmt"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"strings"
)

func Serialize(tags mesh_proto.MultiValueTagSet) string {
	var pairs []string
	for _, key := range tags.Keys() {
		pairs = append(pairs, fmt.Sprintf("&%s=%s&", key, strings.Join(tags.Values(key), ",")))
	}
	return strings.Join(pairs, "")
}
