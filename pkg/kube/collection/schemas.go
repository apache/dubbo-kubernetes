package collection

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"github.com/apache/dubbo-kubernetes/operator/pkg/schema"
)

type Schemas struct {
	byCollection map[config.GroupVersionKind]schema.Schema
	byAddOrder   []schema.Schema
}

type SchemasBuilder struct {
	schemas Schemas
}

func NewSchemasBuilder() *SchemasBuilder {
	s := Schemas{
		byCollection: make(map[config.GroupVersionKind]schema.Schema),
	}
	return &SchemasBuilder{schemas: s}
}

func (b *SchemasBuilder) Add(s schema.Schema) error {
	if _, found := b.schemas.byCollection[s.GroupVersionKind()]; found {
		return fmt.Errorf("collection already exists: %v", s.GroupVersionKind())
	}
	b.schemas.byCollection[s.GroupVersionKind()] = s
	b.schemas.byAddOrder = append(b.schemas.byAddOrder, s)
	return nil
}

func (b *SchemasBuilder) MustAdd(s schema.Schema) *SchemasBuilder {
	if err := b.Add(s); err != nil {
		panic(fmt.Sprintf("SchemasBuilder.MustAdd: %v", err))
	}
	return b
}
