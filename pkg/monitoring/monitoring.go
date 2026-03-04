//
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

package monitoring

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Unit encodes the standard name for describing the quantity
// measured by a Metric (if applicable).
type Unit string

// Predefined units for use with the monitoring package.
const (
	None         Unit = "1"
	Bytes        Unit = "By"
	Seconds      Unit = "s"
	Milliseconds Unit = "ms"
)

// A Metric collects numerical observations.
type Metric interface {
	// Increment records a value of 1 for the current measure.
	Increment()

	// Decrement records a value of -1 for the current measure.
	Decrement()

	// Name returns the name value of a Metric.
	Name() string

	// Record makes an observation of the provided value for the given measure.
	Record(value float64)

	// RecordInt makes an observation of the provided value for the measure.
	RecordInt(value int64)

	// With creates a new Metric, with the LabelValues provided.
	With(labelValues ...LabelValue) Metric
}

// A Label provides a named dimension for a Metric.
type Label struct {
	Name string
}

// A LabelValue represents a Label with a specific value.
type LabelValue struct {
	Name  string
	Value string
}

// Value creates a LabelValue for this Label.
func (l Label) Value(value string) LabelValue {
	return LabelValue{Name: l.Name, Value: value}
}

// CreateLabel creates a new Label.
func CreateLabel(name string) Label {
	return Label{Name: name}
}

// Options encode changes to the options passed to a Metric at creation time.
type Options func(*options)

type options struct {
	unit   Unit
	labels []string
}

// WithUnit provides configuration options for a new Metric.
func WithUnit(unit Unit) Options {
	return func(opts *options) {
		opts.unit = unit
	}
}

// WithLabels provides configuration options for a new Metric with predefined label names.
func WithLabels(labels ...string) Options {
	return func(opts *options) {
		opts.labels = labels
	}
}

// NewSum creates a new Metric with an aggregation type of Sum.
func NewSum(name, description string, opts ...Options) Metric {
	return newMetric(name, description, prometheus.CounterValue, opts...)
}

// NewGauge creates a new Metric with an aggregation type of Gauge.
func NewGauge(name, description string, opts ...Options) Metric {
	return newMetric(name, description, prometheus.GaugeValue, opts...)
}

// NewDistribution creates a new Metric with an aggregation type of Distribution.
func NewDistribution(name, description string, bounds []float64, opts ...Options) Metric {
	return newDistribution(name, description, bounds, opts...)
}

// DerivedGauge is a Metric that is computed from a function.
type DerivedGauge interface {
	// Name returns the name value of a DerivedGauge.
	Name() string

	// ValueFrom sets the function to be called to get the value.
	ValueFrom(valueFn func() float64) DerivedGauge
}

// NewDerivedGauge creates a new DerivedGauge.
func NewDerivedGauge(name, description string, opts ...Options) DerivedGauge {
	return newDerivedGauge(name, description, opts...)
}

var (
	registry     = prometheus.NewRegistry()
	registryLock sync.RWMutex
)

type metric struct {
	name        string
	description string
	valueType   prometheus.ValueType
	labelNames  []string
	vec         *prometheus.GaugeVec
	vecOnce     sync.Once
	counter     prometheus.Counter
	gauge       prometheus.Gauge
}

func newMetric(name, description string, valueType prometheus.ValueType, opts ...Options) Metric {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	m := &metric{
		name:        name,
		description: description,
		valueType:   valueType,
		labelNames:  o.labels,
	}

	// If labels are predefined, create the vector immediately
	if len(o.labels) > 0 {
		if valueType == prometheus.CounterValue {
			m.vec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: name,
				Help: description,
			}, o.labels)
		} else {
			m.vec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: name,
				Help: description,
			}, o.labels)
		}
		registryLock.Lock()
		registry.MustRegister(m.vec)
		registryLock.Unlock()
	} else {
		// No predefined labels, create scalar metric
		if valueType == prometheus.CounterValue {
			m.counter = prometheus.NewCounter(prometheus.CounterOpts{
				Name: name,
				Help: description,
			})
			registryLock.Lock()
			registry.MustRegister(m.counter)
			registryLock.Unlock()
		} else {
			m.gauge = prometheus.NewGauge(prometheus.GaugeOpts{
				Name: name,
				Help: description,
			})
			registryLock.Lock()
			registry.MustRegister(m.gauge)
			registryLock.Unlock()
		}
	}

	return m
}

func (m *metric) Increment() {
	m.Record(1)
}

func (m *metric) Decrement() {
	m.Record(-1)
}

func (m *metric) Name() string {
	return m.name
}

func (m *metric) Record(value float64) {
	if m.counter != nil {
		m.counter.Add(value)
	} else if m.gauge != nil {
		if value >= 0 {
			m.gauge.Set(value)
		} else {
			m.gauge.Add(value)
		}
	}
}

func (m *metric) RecordInt(value int64) {
	m.Record(float64(value))
}

func (m *metric) With(labelValues ...LabelValue) Metric {
	// If vector already exists (predefined labels), use it directly
	if m.vec != nil {
		labels := make(prometheus.Labels)
		for _, lv := range labelValues {
			labels[lv.Name] = lv.Value
		}
		return &labeledMetric{
			parent: m,
			gauge:  m.vec.With(labels),
			labels: labels,
		}
	}

	// Otherwise, create vector on first call (legacy behavior)
	m.vecOnce.Do(func() {
		labelNames := make([]string, len(labelValues))
		for i, lv := range labelValues {
			labelNames[i] = lv.Name
		}

		if m.valueType == prometheus.CounterValue {
			m.vec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: m.name,
				Help: m.description,
			}, labelNames)
		} else {
			m.vec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: m.name,
				Help: m.description,
			}, labelNames)
		}

		registryLock.Lock()
		registry.MustRegister(m.vec)
		registryLock.Unlock()
	})

	labels := make(prometheus.Labels)
	for _, lv := range labelValues {
		labels[lv.Name] = lv.Value
	}

	return &labeledMetric{
		parent: m,
		gauge:  m.vec.With(labels),
		labels: labels,
	}
}

type labeledMetric struct {
	parent *metric
	gauge  prometheus.Gauge
	labels prometheus.Labels
}

func (lm *labeledMetric) Increment() {
	lm.Record(1)
}

func (lm *labeledMetric) Decrement() {
	lm.Record(-1)
}

func (lm *labeledMetric) Name() string {
	return lm.parent.name
}

func (lm *labeledMetric) Record(value float64) {
	if lm.parent.valueType == prometheus.CounterValue {
		lm.gauge.Add(value)
	} else {
		if value >= 0 {
			lm.gauge.Set(value)
		} else {
			lm.gauge.Add(value)
		}
	}
}

func (lm *labeledMetric) RecordInt(value int64) {
	lm.Record(float64(value))
}

func (lm *labeledMetric) With(labelValues ...LabelValue) Metric {
	newLabels := make(prometheus.Labels)
	for k, v := range lm.labels {
		newLabels[k] = v
	}
	for _, lv := range labelValues {
		newLabels[lv.Name] = lv.Value
	}

	return &labeledMetric{
		parent: lm.parent,
		gauge:  lm.parent.vec.With(newLabels),
		labels: newLabels,
	}
}

type distribution struct {
	name        string
	description string
	histogram   *prometheus.HistogramVec
	histOnce    sync.Once
	bounds      []float64
}

func newDistribution(name, description string, bounds []float64, opts ...Options) Metric {
	d := &distribution{
		name:        name,
		description: description,
		bounds:      bounds,
	}

	d.histOnce.Do(func() {
		d.histogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    name,
			Help:    description,
			Buckets: bounds,
		}, []string{})

		registryLock.Lock()
		registry.MustRegister(d.histogram)
		registryLock.Unlock()
	})

	return &distributionMetric{
		parent:    d,
		histogram: d.histogram.WithLabelValues(),
	}
}

type distributionMetric struct {
	parent    *distribution
	histogram prometheus.Observer
	labels    prometheus.Labels
}

func (dm *distributionMetric) Increment() {
	dm.Record(1)
}

func (dm *distributionMetric) Decrement() {
	dm.Record(-1)
}

func (dm *distributionMetric) Name() string {
	return dm.parent.name
}

func (dm *distributionMetric) Record(value float64) {
	dm.histogram.Observe(value)
}

func (dm *distributionMetric) RecordInt(value int64) {
	dm.Record(float64(value))
}

func (dm *distributionMetric) With(labelValues ...LabelValue) Metric {
	if len(labelValues) == 0 {
		return dm
	}

	// Create new histogram vec with labels if needed
	labelNames := make([]string, len(labelValues))
	labels := make(prometheus.Labels)
	for i, lv := range labelValues {
		labelNames[i] = lv.Name
		labels[lv.Name] = lv.Value
	}

	// Check if we need to recreate the histogram with labels
	if dm.labels == nil {
		newHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    dm.parent.name,
			Help:    dm.parent.description,
			Buckets: dm.parent.bounds,
		}, labelNames)

		registryLock.Lock()
		registry.MustRegister(newHist)
		registryLock.Unlock()

		dm.parent.histogram = newHist
	}

	return &distributionMetric{
		parent:    dm.parent,
		histogram: dm.parent.histogram.With(labels),
		labels:    labels,
	}
}

type derivedGauge struct {
	name        string
	description string
	valueFn     func() float64
	collector   prometheus.Collector
}

func newDerivedGauge(name, description string, opts ...Options) DerivedGauge {
	dg := &derivedGauge{
		name:        name,
		description: description,
	}
	return dg
}

func (dg *derivedGauge) Name() string {
	return dg.name
}

func (dg *derivedGauge) ValueFrom(valueFn func() float64) DerivedGauge {
	dg.valueFn = valueFn

	// Create a custom collector
	dg.collector = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: dg.name,
		Help: dg.description,
	}, valueFn)

	registryLock.Lock()
	registry.MustRegister(dg.collector)
	registryLock.Unlock()

	return dg
}

// RegisterPrometheusExporter registers the Prometheus exporter.
func RegisterPrometheusExporter(ctx context.Context) error {
	// Already registered in the package-level registry
	return nil
}

// GetRegistry returns the Prometheus registry.
func GetRegistry() *prometheus.Registry {
	registryLock.RLock()
	defer registryLock.RUnlock()
	return registry
}
