package main

// BenchmarkConfig represents the input YAML structure
type BenchmarkConfig struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"` // defaults to "app"
	Services  map[string]Service `yaml:"services"`
	Edges     []Edge            `yaml:"edges"`
	Roshanfer *RoshanferConfig  `yaml:"roshanfer,omitempty"`
	CPU       *CPUConfig        `yaml:"cpu,omitempty"`
}

type Service struct {
	Type            string `yaml:"type"` // "frontend" or "backend"
	PreRepeat       int    `yaml:"pre_repeat"`
	PostRepeat      int    `yaml:"post_repeat"`
	BackendRepeat   int    `yaml:"backend_repeat,omitempty"` // for backend services
	ResponseSize    int    `yaml:"response_size"`
	HTTPEndpoints   []string `yaml:"http_endpoints,omitempty"` // for frontend services
	CPUCores        int    `yaml:"cpu_cores"` // number of cores to pin
	SidecarCPUCores int    `yaml:"sidecar_cpu_cores,omitempty"` // cores for sidecar, defaults to 2
}

type Edge struct {
	From       string `yaml:"from"`   // frontend service name
	To         string `yaml:"to"`     // backend service name
	Method     string `yaml:"method"` // method name for this call
	PreRepeat  *int   `yaml:"pre_repeat,omitempty"`  // optional override for pre-processing
	PostRepeat *int   `yaml:"post_repeat,omitempty"` // optional override for post-processing
}

type RoshanferConfig struct {
	Limits       map[string]LimitConfig      `yaml:"limits"`
	SLOs         map[string]int             `yaml:"slos,omitempty"`         // endpoint -> SLO in ms
	PPM          map[string]int              `yaml:"ppm"`                    // service -> PPM limit
	SidecarConfig map[string]SidecarConfigOverride `yaml:"sidecar_config,omitempty"` // service -> sidecar config overrides
}

type SidecarConfigOverride struct {
	RingSize              *int `yaml:"ring_size,omitempty"`
	BufferCount           *int `yaml:"buffer_count,omitempty"`
	BufferSize            *int `yaml:"buffer_size,omitempty"`
	NumThreads            *int `yaml:"num_threads,omitempty"`
	IngressPoolConnections *int `yaml:"ingress_pool_connections,omitempty"`
	FrontendPoolConnections *int `yaml:"frontend_pool_connections,omitempty"`
}

type LimitConfig struct {
	ServiceLimits map[string]int `yaml:"service_limits,omitempty"` // service -> limit
	EndpointLimits map[string]int `yaml:"endpoint_limits,omitempty"` // endpoint -> limit (for ingress)
}

type CPUConfig struct {
	StartCore int `yaml:"start_core"` // starting core number
	TotalCores int `yaml:"total_cores"` // total cores available
}

// GeneratedConfig holds computed values during generation
type GeneratedConfig struct {
	BenchmarkConfig
	Ports        map[string]ServicePorts
	CPUCores     map[string]CPUCoreAssignment
	ProtoServices map[string][]string // service -> methods
}

type ServicePorts struct {
	HTTPPort        int // frontend HTTP port
	GRPCPort        int // backend gRPC port
	IngressPort     int // sidecar ingress listener port
	EgressPort      int // sidecar egress listener port
	UpstreamPort    int // port where service listens (for sidecar to connect)
}

type CPUCoreAssignment struct {
	ServiceCores []int // cores for the service
	SidecarCores []int // cores for the sidecar
}

