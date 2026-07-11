// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gateway

import (
	"fmt"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const (
	kubernetesSecretPrefix        = "kubernetes://"
	kubernetesGatewaySecretPrefix = "kubernetes-gateway://"
)

type referenceGrantStore struct {
	Collection krt.Collection[ReferenceGrant]
	Index      krt.Index[ReferencePair, ReferenceGrant]
}

type Reference struct {
	Kind      config.GroupVersionKind
	Namespace gatewayv1.Namespace
}

func (r Reference) String() string {
	return r.Kind.String() + "/" + string(r.Namespace)
}

type ReferencePair struct {
	To, From Reference
}

func (p ReferencePair) String() string {
	return fmt.Sprintf("%s->%s", p.From, p.To)
}

type ReferenceGrant struct {
	Source      types.NamespacedName
	From        Reference
	To          Reference
	AllowAll    bool
	AllowedName string
}

func (g ReferenceGrant) ResourceName() string {
	nameKey := "*"
	if !g.AllowAll {
		nameKey = g.AllowedName
	}
	return g.Source.String() + "/" + g.From.Kind.String() + "/" + string(g.From.Namespace) + "/" + g.To.Kind.String() + "/" + string(g.To.Namespace) + "/" + nameKey
}

func ReferenceGrantsCollection(
	referenceGrants krt.Collection[*gatewayv1beta1.ReferenceGrant],
	opts krt.OptionsBuilder,
) krt.Collection[ReferenceGrant] {
	return krt.NewManyCollection(referenceGrants, func(ctx krt.HandlerContext, obj *gatewayv1beta1.ReferenceGrant) []ReferenceGrant {
		result := make([]ReferenceGrant, 0, len(obj.Spec.From)*len(obj.Spec.To))
		for _, from := range obj.Spec.From {
			fromKind := normalizeReference(&from.Group, &from.Kind, config.GroupVersionKind{})
			if fromKind != gvk.KubernetesGateway && fromKind != gvk.HTTPRoute {
				continue
			}
			fromRef := Reference{
				Kind:      fromKind,
				Namespace: from.Namespace,
			}
			for _, to := range obj.Spec.To {
				toKind := normalizeReference(&to.Group, &to.Kind, config.GroupVersionKind{})
				if toKind != gvk.Secret {
					continue
				}
				grant := ReferenceGrant{
					Source: types.NamespacedName{
						Name:      obj.Name,
						Namespace: obj.Namespace,
					},
					From: fromRef,
					To: Reference{
						Kind:      toKind,
						Namespace: gatewayv1.Namespace(obj.Namespace),
					},
					AllowAll: to.Name == nil,
				}
				if to.Name != nil {
					grant.AllowedName = string(*to.Name)
				}
				result = append(result, grant)
			}
		}
		return result
	}, opts.WithName("ReferenceGrants")...)
}

func newReferenceGrantStore(collection krt.Collection[ReferenceGrant]) referenceGrantStore {
	return referenceGrantStore{
		Collection: collection,
		Index: krt.NewIndex(collection, "toFrom", func(o ReferenceGrant) []ReferencePair {
			return []ReferencePair{{To: o.To, From: o.From}}
		}),
	}
}

func (refs referenceGrantStore) SecretAllowed(ctx krt.HandlerContext, kind config.GroupVersionKind, resourceName string, namespace string) bool {
	if refs.Collection == nil {
		return false
	}
	name, targetNamespace, targetKind, ok := parseSecretResourceName(resourceName, namespace)
	if !ok {
		return false
	}
	pair := ReferencePair{
		From: Reference{Kind: kind, Namespace: gatewayv1.Namespace(namespace)},
		To:   Reference{Kind: targetKind, Namespace: gatewayv1.Namespace(targetNamespace)},
	}
	for _, grant := range krt.FetchOrList(ctx, refs.Collection, krt.FilterIndex(refs.Index, pair)) {
		if grant.AllowAll || grant.AllowedName == name {
			return true
		}
	}
	return false
}

func parseSecretResourceName(resourceName string, defaultNamespace string) (name string, namespace string, kind config.GroupVersionKind, ok bool) {
	kind = gvk.Secret
	switch {
	case strings.HasPrefix(resourceName, kubernetesGatewaySecretPrefix):
		parts := strings.Split(strings.TrimPrefix(resourceName, kubernetesGatewaySecretPrefix), "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", config.GroupVersionKind{}, false
		}
		return parts[1], parts[0], kind, true
	case strings.HasPrefix(resourceName, kubernetesSecretPrefix):
		parts := strings.Split(strings.TrimPrefix(resourceName, kubernetesSecretPrefix), "/")
		if len(parts) == 1 && parts[0] != "" {
			return parts[0], defaultNamespace, kind, true
		}
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[1], parts[0], kind, true
		}
		return "", "", config.GroupVersionKind{}, false
	default:
		parts := strings.Split(resourceName, "/")
		if len(parts) == 1 && parts[0] != "" {
			return parts[0], defaultNamespace, kind, true
		}
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[1], parts[0], kind, true
		}
		return "", "", config.GroupVersionKind{}, false
	}
}

func normalizeReference[G ~string, K ~string](group *G, kind *K, fallback config.GroupVersionKind) config.GroupVersionKind {
	out := fallback
	if group != nil {
		out.Group = string(*group)
	}
	if kind != nil {
		out.Kind = string(*kind)
	}
	if schema, found := collections.All.FindByGroupKind(out); found {
		return schema.GroupVersionKind()
	}
	return out
}
