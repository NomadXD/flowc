package profile

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
)

//go:embed templates/bootstrap.yaml.tmpl
var bootstrapTemplate string

// BootstrapParams are the parameters used to render the Envoy bootstrap template.
type BootstrapParams struct {
	GatewayName      string
	NodeID           string
	ProfileType      string
	ClusterName      string
	XDSClusterName   string
	ControlPlaneHost string
	ControlPlanePort int
	AdminPort        int
}

// GenerateBootstrapYAML renders an Envoy bootstrap configuration for the given
// gateway and profile. controlPlaneHost and controlPlanePort specify where the
// Envoy should connect for xDS.
func GenerateBootstrapYAML(gw *resource.GatewayResource, prof *resource.GatewayProfileResource, controlPlaneHost string, controlPlanePort int) ([]byte, error) {
	params := BootstrapParams{
		GatewayName:      gw.Meta.Name,
		NodeID:           gw.Spec.NodeID,
		ClusterName:      gw.Meta.Name,
		XDSClusterName:   "flowc_xds_cluster",
		ControlPlaneHost: controlPlaneHost,
		ControlPlanePort: controlPlanePort,
		AdminPort:        9901,
	}

	if prof != nil {
		params.ProfileType = prof.Spec.ProfileType
		if prof.Spec.Bootstrap != nil {
			if prof.Spec.Bootstrap.AdminPort > 0 {
				params.AdminPort = int(prof.Spec.Bootstrap.AdminPort)
			}
			if prof.Spec.Bootstrap.XDSClusterName != "" {
				params.XDSClusterName = prof.Spec.Bootstrap.XDSClusterName
			}
		}
	} else {
		params.ProfileType = "default"
	}

	tmpl, err := template.New("bootstrap").Parse(bootstrapTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse bootstrap template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return nil, fmt.Errorf("execute bootstrap template: %w", err)
	}

	return buf.Bytes(), nil
}
