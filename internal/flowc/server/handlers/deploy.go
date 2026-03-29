package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
	"github.com/flowc-labs/flowc/internal/flowc/resource/store"
	"github.com/flowc-labs/flowc/pkg/logger"
)

// DeployHandler generates deployment instructions for gateways.
type DeployHandler struct {
	typedStore       *store.TypedStore
	logger           *logger.EnvoyLogger
	controlPlaneHost string
	controlPlanePort int
	apiPort          int
}

// NewDeployHandler creates a new deploy instructions handler.
func NewDeployHandler(s store.Store, controlPlaneHost string, controlPlanePort, apiPort int, log *logger.EnvoyLogger) *DeployHandler {
	return &DeployHandler{
		typedStore:       store.NewTypedStore(s),
		logger:           log,
		controlPlaneHost: controlPlaneHost,
		controlPlanePort: controlPlanePort,
		apiPort:          apiPort,
	}
}

// HandleDeploy generates deployment instructions for a gateway.
// GET /api/v1/gateways/{name}/deploy
func (h *DeployHandler) HandleDeploy(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	gw, err := h.typedStore.GetGateway(r.Context(), name)
	if err != nil {
		if err == resource.ErrNotFound {
			writeError(w, http.StatusNotFound, "gateway not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Load the referenced profile.
	var prof *resource.GatewayProfileResource
	if gw.Spec.ProfileRef != "" {
		prof, _ = h.typedStore.GetGatewayProfile(r.Context(), gw.Spec.ProfileRef)
	}

	// Load listeners for port mappings.
	allListeners, err := h.typedStore.ListListeners(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var listeners []*resource.ListenerResource
	for _, l := range allListeners {
		if l.Spec.GatewayRef == gw.Meta.Name {
			listeners = append(listeners, l)
		}
	}

	instructions := h.buildInstructions(gw, prof, listeners)
	writeJSON(w, http.StatusOK, instructions)
}

// DeployInstructions is the response body for the deploy endpoint.
type DeployInstructions struct {
	Gateway    GatewayInfo        `json:"gateway"`
	Docker     DockerInstructions `json:"docker"`
	Kubernetes K8sInstructions    `json:"kubernetes"`
}

// GatewayInfo summarizes the gateway for deploy instructions.
type GatewayInfo struct {
	Name       string `json:"name"`
	NodeID     string `json:"nodeId"`
	Profile    string `json:"profile,omitempty"`
	EnvoyImage string `json:"envoyImage"`
}

// DockerInstructions contains Docker deployment details.
type DockerInstructions struct {
	BootstrapURL   string `json:"bootstrapUrl"`
	RunCommand     string `json:"runCommand"`
	ComposeSnippet string `json:"composeSnippet"`
}

// K8sInstructions contains Kubernetes deployment details.
type K8sInstructions struct {
	Manifest     string `json:"manifest"`
	ApplyCommand string `json:"applyCommand"`
}

func (h *DeployHandler) buildInstructions(gw *resource.GatewayResource, prof *resource.GatewayProfileResource, listeners []*resource.ListenerResource) *DeployInstructions {
	envoyImage := "envoyproxy/envoy:v1.31-latest"
	adminPort := 9901
	profileName := ""

	if prof != nil {
		profileName = prof.Spec.ProfileType
		if prof.Spec.EnvoyImage != "" {
			envoyImage = prof.Spec.EnvoyImage
		}
		if prof.Spec.Bootstrap != nil && prof.Spec.Bootstrap.AdminPort > 0 {
			adminPort = int(prof.Spec.Bootstrap.AdminPort)
		}
	}

	bootstrapURL := fmt.Sprintf("http://%s:%d/api/v1/gateways/%s/bootstrap",
		h.controlPlaneHost, h.apiPort, gw.Meta.Name)

	// Build Docker port mappings from listeners.
	var portMappings []string
	portMappings = append(portMappings, fmt.Sprintf("-p %d:%d", adminPort, adminPort))
	for _, l := range listeners {
		portMappings = append(portMappings, fmt.Sprintf("-p %d:%d", l.Spec.Port, l.Spec.Port))
	}

	dockerRun := fmt.Sprintf(
		"docker run --rm --name %s \\\n  %s \\\n  -v $(pwd)/envoy-bootstrap.yaml:/etc/envoy/envoy.yaml \\\n  %s",
		gw.Meta.Name,
		strings.Join(portMappings, " \\\n  "),
		envoyImage,
	)

	composeSnippet := fmt.Sprintf(`  %s:
    image: %s
    volumes:
      - ./envoy-bootstrap.yaml:/etc/envoy/envoy.yaml
    ports:`, gw.Meta.Name, envoyImage)
	composeSnippet += fmt.Sprintf("\n      - \"%d:%d\"", adminPort, adminPort)
	for _, l := range listeners {
		composeSnippet += fmt.Sprintf("\n      - \"%d:%d\"", l.Spec.Port, l.Spec.Port)
	}
	composeSnippet += fmt.Sprintf("\n    network_mode: host")

	// K8s manifest: a Deployment + Service for the Envoy proxy.
	k8sManifest := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  labels:
    app: %s
    flowc.io/gateway: "%s"
    flowc.io/profile: "%s"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: envoy
        image: %s
        ports:
        - containerPort: %d
          name: admin`,
		gw.Meta.Name, gw.Meta.Name, gw.Meta.Name, profileName,
		gw.Meta.Name, gw.Meta.Name, envoyImage, adminPort)

	for _, l := range listeners {
		k8sManifest += fmt.Sprintf("\n        - containerPort: %d\n          name: listener-%d", l.Spec.Port, l.Spec.Port)
	}

	k8sManifest += fmt.Sprintf(`
        volumeMounts:
        - name: bootstrap
          mountPath: /etc/envoy/envoy.yaml
          subPath: envoy.yaml
      volumes:
      - name: bootstrap
        configMap:
          name: %s-bootstrap`, gw.Meta.Name)

	return &DeployInstructions{
		Gateway: GatewayInfo{
			Name:       gw.Meta.Name,
			NodeID:     gw.Spec.NodeID,
			Profile:    profileName,
			EnvoyImage: envoyImage,
		},
		Docker: DockerInstructions{
			BootstrapURL:   bootstrapURL,
			RunCommand:     dockerRun,
			ComposeSnippet: composeSnippet,
		},
		Kubernetes: K8sInstructions{
			Manifest:     k8sManifest,
			ApplyCommand: fmt.Sprintf("kubectl apply -f %s-deployment.yaml", gw.Meta.Name),
		},
	}
}
