package kubetypes

type InformerOptions struct {
	LabelSelector   string
	FieldSelector   string
	Namespace       string
	ObjectTransform func(obj any) (any, error)
	InformerType    InformerType
}

type InformerType int

const (
	StandardInformer InformerType = iota
	DynamicInformer
	MetadataInformer
)
