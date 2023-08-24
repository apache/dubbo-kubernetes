/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/apache/dubbo-admin/pkg/core/gen/apis/dubbo.apache.org/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ConditionRouteLister helps list ConditionRoutes.
// All objects returned here must be treated as read-only.
type ConditionRouteLister interface {
	// List lists all ConditionRoutes in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.ConditionRoute, err error)
	// ConditionRoutes returns an object that can list and get ConditionRoutes.
	ConditionRoutes(namespace string) ConditionRouteNamespaceLister
	ConditionRouteListerExpansion
}

// conditionRouteLister implements the ConditionRouteLister interface.
type conditionRouteLister struct {
	indexer cache.Indexer
}

// NewConditionRouteLister returns a new ConditionRouteLister.
func NewConditionRouteLister(indexer cache.Indexer) ConditionRouteLister {
	return &conditionRouteLister{indexer: indexer}
}

// List lists all ConditionRoutes in the indexer.
func (s *conditionRouteLister) List(selector labels.Selector) (ret []*v1alpha1.ConditionRoute, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ConditionRoute))
	})
	return ret, err
}

// ConditionRoutes returns an object that can list and get ConditionRoutes.
func (s *conditionRouteLister) ConditionRoutes(namespace string) ConditionRouteNamespaceLister {
	return conditionRouteNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ConditionRouteNamespaceLister helps list and get ConditionRoutes.
// All objects returned here must be treated as read-only.
type ConditionRouteNamespaceLister interface {
	// List lists all ConditionRoutes in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.ConditionRoute, err error)
	// Get retrieves the ConditionRoute from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.ConditionRoute, error)
	ConditionRouteNamespaceListerExpansion
}

// conditionRouteNamespaceLister implements the ConditionRouteNamespaceLister
// interface.
type conditionRouteNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ConditionRoutes in the indexer for a given namespace.
func (s conditionRouteNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.ConditionRoute, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ConditionRoute))
	})
	return ret, err
}

// Get retrieves the ConditionRoute from the indexer for a given namespace and name.
func (s conditionRouteNamespaceLister) Get(name string) (*v1alpha1.ConditionRoute, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("conditionroute"), name)
	}
	return obj.(*v1alpha1.ConditionRoute), nil
}
