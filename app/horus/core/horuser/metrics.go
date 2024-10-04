package horuser

import "github.com/prometheus/client_golang/prometheus"

func (h *Horuser) Collect(ch chan<- prometheus.Metric) {}

func (h *Horuser) Describe(ch chan<- *prometheus.Desc) {}
