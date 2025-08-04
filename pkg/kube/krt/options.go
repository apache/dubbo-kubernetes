package krt

type OptionsBuilder struct {
	namePrefix string
	stop       <-chan struct{}
}

type BuilderOption func(opt CollectionOption) OptionsBuilder

func NewOptionsBuilder(stop <-chan struct{}, namePrefix string) OptionsBuilder {
	return OptionsBuilder{
		namePrefix: namePrefix,
		stop:       stop,
	}
}

func (k OptionsBuilder) Stop() <-chan struct{} {
	return k.stop
}

func (k OptionsBuilder) WithName(n string) []CollectionOption {
	name := n
	if k.namePrefix != "" {
		name = k.namePrefix + "/" + name
	}
	return []CollectionOption{WithStop(k.stop), WithName(name)}
}

func WithStop(stop <-chan struct{}) CollectionOption {
	return func(c *collectionOptions) {
		c.stop = stop
	}
}

func WithName(name string) CollectionOption {
	return func(c *collectionOptions) {
		c.name = name
	}
}

func WithObjectAugmentation(fn func(o any) any) CollectionOption {
	return func(c *collectionOptions) {
		c.augmentation = fn
	}
}

func WithDebugging(handler *DebugHandler) CollectionOption {
	return func(c *collectionOptions) {
		c.debugger = handler
	}
}
