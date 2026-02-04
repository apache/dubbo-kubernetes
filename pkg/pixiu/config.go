package pixiu

// PixiuBootstrap represents Pixiu Bootstrap configuration
type PixiuBootstrap struct {
	StaticResources StaticResources `yaml:"static_resources" json:"static_resources"`
}

// StaticResources contains static resources
type StaticResources struct {
	Listeners []*Listener `yaml:"listeners" json:"listeners"`
	Clusters  []*Cluster  `yaml:"clusters" json:"clusters"`
}

// Listener represents a Pixiu listener
type Listener struct {
	Name        string      `yaml:"name" json:"name"`
	Address     Address     `yaml:"address" json:"address"`
	ProtocolStr string      `yaml:"protocol_type" json:"protocol_type"`
	FilterChain FilterChain `yaml:"filter_chains" json:"filter_chains"`
}

// Address represents network address
type Address struct {
	SocketAddress SocketAddress `yaml:"socket_address" json:"socket_address"`
}

// SocketAddress represents socket address
type SocketAddress struct {
	Address   string `yaml:"address" json:"address"`
	PortValue int    `yaml:"port_value" json:"port_value"`
}

// FilterChain represents filter chain
type FilterChain struct {
	Filters []Filter `yaml:"filters" json:"filters"`
}

// Filter represents a filter
type Filter struct {
	Name   string                 `yaml:"name" json:"name"`
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// Cluster represents a Pixiu cluster
type Cluster struct {
	Name        string       `yaml:"name" json:"name"`
	Type        string       `yaml:"type" json:"type"`
	LbPolicy    string       `yaml:"lb_policy" json:"lb_policy"`
	Endpoints   []Endpoint   `yaml:"endpoints" json:"endpoints"`
	HealthCheck *HealthCheck `yaml:"health_check,omitempty" json:"health_check,omitempty"`
}

// Endpoint represents an endpoint
type Endpoint struct {
	Address Address `yaml:"address" json:"address"`
}

// HealthCheck represents health check configuration
type HealthCheck struct {
	Timeout            string `yaml:"timeout" json:"timeout"`
	Interval           string `yaml:"interval" json:"interval"`
	UnhealthyThreshold int    `yaml:"unhealthy_threshold" json:"unhealthy_threshold"`
	HealthyThreshold   int    `yaml:"healthy_threshold" json:"healthy_threshold"`
}
