package providers

// ID defines underlying platform supporting service registry
type ID string

const (
	// Kubernetes is a service registry backed by k8s API server
	Kubernetes ID = "Kubernetes"
)

func (id ID) String() string {
	return string(id)
}
