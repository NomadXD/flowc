package resource

// ListenerResource represents a port binding within a gateway.
type ListenerResource struct {
	Meta   ResourceMeta   `json:"metadata" yaml:"metadata"`
	Spec   ListenerSpec   `json:"spec" yaml:"spec"`
	Status ListenerStatus `json:"status" yaml:"status"`
}

// ListenerSpec defines the desired state of a listener.
type ListenerSpec struct {
	// GatewayRef is the name of the parent Gateway resource.
	GatewayRef string `json:"gatewayRef" yaml:"gatewayRef"`

	// Port is the bind port; must be unique within the referenced gateway.
	Port uint32 `json:"port" yaml:"port"`

	// Address is the bind address (default "0.0.0.0").
	Address string `json:"address,omitempty" yaml:"address,omitempty"`

	// TLS contains optional TLS configuration.
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`

	// HTTP2 enables HTTP/2 on the listener.
	HTTP2 bool `json:"http2,omitempty" yaml:"http2,omitempty"`
}

// TLSConfig mirrors the existing models.TLSConfig for listener TLS settings.
type TLSConfig struct {
	CertPath          string   `json:"certPath" yaml:"certPath"`
	KeyPath           string   `json:"keyPath" yaml:"keyPath"`
	CAPath            string   `json:"caPath,omitempty" yaml:"caPath,omitempty"`
	RequireClientCert bool     `json:"requireClientCert,omitempty" yaml:"requireClientCert,omitempty"`
	MinVersion        string   `json:"minVersion,omitempty" yaml:"minVersion,omitempty"`
	CipherSuites      []string `json:"cipherSuites,omitempty" yaml:"cipherSuites,omitempty"`
}

// ListenerStatus is the observed state of a listener.
type ListenerStatus struct {
	Phase      string      `json:"phase" yaml:"phase"`
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

func (r *ListenerResource) GetMeta() *ResourceMeta { return &r.Meta }
