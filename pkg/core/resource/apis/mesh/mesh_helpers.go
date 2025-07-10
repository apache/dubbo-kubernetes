package mesh

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	meshproto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
)

func (m *MeshResource) HasPrometheusMetricsEnabled() bool {
	return m != nil && m.GetEnabledMetricsBackend().GetType() == meshproto.MetricsPrometheusType
}

func (m *MeshResource) GetEnabledMetricsBackend() *meshproto.MetricsBackend {
	// TODO: support this!
	return nil
}

func (m *MeshResource) GetMetricsBackend(name string) *meshproto.MetricsBackend {
	// TODO: support this!
	return nil
}

func (m *MeshResource) MTLSEnabled() bool {
	return m != nil && m.Spec.GetMtls().GetEnabledBackend() != ""
}

// ZoneEgress works only when mTLS is enabled.
// Configuration of mTLS is validated on Mesh configuration
// change and when zoneEgress is enabled.
func (m *MeshResource) ZoneEgressEnabled() bool {
	return m != nil && m.Spec.GetRouting().GetZoneEgress()
}

func (m *MeshResource) LocalityAwareLbEnabled() bool {
	return m != nil && m.Spec.GetRouting().GetLocalityAwareLoadBalancing()
}

func (m *MeshResource) GetLoggingBackend(name string) *meshproto.LoggingBackend {
	backends := map[string]*meshproto.LoggingBackend{}
	for _, backend := range m.Spec.GetLogging().GetBackends() {
		backends[backend.Name] = backend
	}
	if name == "" {
		return backends[m.Spec.GetLogging().GetDefaultBackend()]
	}
	return backends[name]
}

func (m *MeshResource) GetTracingBackend(name string) *meshproto.TracingBackend {
	backends := map[string]*meshproto.TracingBackend{}
	for _, backend := range m.Spec.GetTracing().GetBackends() {
		backends[backend.Name] = backend
	}
	if name == "" {
		return backends[m.Spec.GetTracing().GetDefaultBackend()]
	}
	return backends[name]
}

// GetLoggingBackends will return logging backends as comma separated strings
// if empty return empty string
func (m *MeshResource) GetLoggingBackends() string {
	var backends []string
	for _, backend := range m.Spec.GetLogging().GetBackends() {
		backend := fmt.Sprintf("%s/%s", backend.GetType(), backend.GetName())
		backends = append(backends, backend)
	}
	return strings.Join(backends, ", ")
}

// GetTracingBackends will return tracing backends as comma separated strings
// if empty return empty string
func (m *MeshResource) GetTracingBackends() string {
	var backends []string
	for _, backend := range m.Spec.GetTracing().GetBackends() {
		backend := fmt.Sprintf("%s/%s", backend.GetType(), backend.GetName())
		backends = append(backends, backend)
	}
	return strings.Join(backends, ", ")
}

func (m *MeshResource) GetEnabledCertificateAuthorityBackend() *meshproto.CertificateAuthorityBackend {
	return m.GetCertificateAuthorityBackend(m.Spec.GetMtls().GetEnabledBackend())
}

func (m *MeshResource) GetCertificateAuthorityBackend(name string) *meshproto.CertificateAuthorityBackend {
	for _, backend := range m.Spec.GetMtls().GetBackends() {
		if backend.Name == name {
			return backend
		}
	}
	return nil
}

var durationRE = regexp.MustCompile("^([0-9]+)(y|w|d|h|m|s|ms)$")

// ParseDuration parses a string into a time.Duration
func ParseDuration(durationStr string) (time.Duration, error) {
	// Allow 0 without a unit.
	if durationStr == "0" {
		return 0, nil
	}
	matches := durationRE.FindStringSubmatch(durationStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("not a valid duration string: %q", durationStr)
	}
	var (
		n, _ = strconv.Atoi(matches[1])
		dur  = time.Duration(n) * time.Millisecond
	)
	switch unit := matches[2]; unit {
	case "y":
		dur *= 1000 * 60 * 60 * 24 * 365
	case "w":
		dur *= 1000 * 60 * 60 * 24 * 7
	case "d":
		dur *= 1000 * 60 * 60 * 24
	case "h":
		dur *= 1000 * 60 * 60
	case "m":
		dur *= 1000 * 60
	case "s":
		dur *= 1000
	case "ms":
		// Value already correct
	default:
		return 0, fmt.Errorf("invalid time unit in duration string: %q", unit)
	}
	return dur, nil
}

func (ml *MeshResourceList) MarshalLog() interface{} {
	maskedList := make([]*MeshResource, 0, len(ml.Items))
	for _, mesh := range ml.Items {
		maskedList = append(maskedList, mesh.MarshalLog().(*MeshResource))
	}
	return MeshResourceList{
		Items:      maskedList,
		Pagination: ml.Pagination,
	}
}

func (m *MeshResource) MarshalLog() interface{} {
	// TODO: support this!
	return nil
}
