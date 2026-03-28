package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/ir"
	"github.com/flowc-labs/flowc/internal/flowc/resource"
	"github.com/flowc-labs/flowc/internal/flowc/server/models"
	"github.com/flowc-labs/flowc/internal/flowc/xds/cache"
	listenerbuilder "github.com/flowc-labs/flowc/internal/flowc/xds/resources/listener"
	"github.com/flowc-labs/flowc/internal/flowc/xds/translator"
	"github.com/flowc-labs/flowc/pkg/types"
)

// translateDeployment translates a single deployment into xDS resources.
// It loads the API, finds the listener/environment, parses the spec to IR,
// converts to model types, resolves strategies, and runs the translator.
// Note: the translator produces clusters + routes only (no listeners).
// Listeners are generated at the reconciler level by buildListeners.
func (r *Reconciler) translateDeployment(
	ctx context.Context,
	gw *resource.GatewayResource,
	dep *resource.DeploymentResource,
	listeners []*resource.ListenerResource,
	envsByListener map[string][]*resource.EnvironmentResource,
) (*translator.XDSResources, error) {
	project := gw.Meta.Project
	nodeID := gw.Spec.NodeID

	// Load the referenced API
	api, err := r.typedStore.GetAPI(ctx, project, dep.Spec.APIRef)
	if err != nil {
		return nil, fmt.Errorf("API %q not found: %w", dep.Spec.APIRef, err)
	}

	// Find the listener for this deployment
	var listener *resource.ListenerResource
	for _, l := range listeners {
		if l.Meta.Name == dep.Spec.ListenerRef {
			listener = l
			break
		}
	}
	if listener == nil {
		return nil, fmt.Errorf("Listener %q not found", dep.Spec.ListenerRef)
	}

	// Find the environment for this deployment
	var env *resource.EnvironmentResource
	for _, e := range envsByListener[dep.Spec.ListenerRef] {
		if e.Meta.Name == dep.Spec.EnvironmentRef {
			env = e
			break
		}
	}
	if env == nil {
		return nil, fmt.Errorf("Environment %q not found", dep.Spec.EnvironmentRef)
	}

	// Parse the API spec to IR (transient)
	var irAPI *ir.API
	if api.Spec.SpecContent != "" {
		apiType := ir.APIType(api.Spec.APIType)
		if apiType == "" {
			apiType = ir.APITypeREST
		}
		parsed, err := r.parserRegistry.Parse(ctx, apiType, []byte(api.Spec.SpecContent))
		if err != nil {
			return nil, fmt.Errorf("failed to parse spec: %w", err)
		}
		irAPI = parsed
		irAPI.Metadata.BasePath = normalizeBasePath(api.Spec.Context)
	}

	// Build models for the existing translator
	modelDeployment := toModelDeployment(dep, api)
	modelGateway := toModelGateway(gw)
	modelListener := toModelListener(listener)
	modelEnv := toModelEnvironment(env)

	// Resolve strategies
	resolver := translator.NewConfigResolver(gw.Spec.Defaults, r.logger)
	resolvedConfig := resolver.Resolve(dep.Spec.Strategy)

	factory := translator.NewStrategyFactory(translator.DefaultTranslatorOptions(), r.logger)
	strategies, err := factory.CreateStrategySet(resolvedConfig, modelDeployment)
	if err != nil {
		return nil, fmt.Errorf("strategy creation failed: %w", err)
	}

	compositeTranslator, err := translator.NewCompositeTranslator(strategies, translator.DefaultTranslatorOptions(), r.logger)
	if err != nil {
		return nil, fmt.Errorf("translator creation failed: %w", err)
	}

	compositeTranslator.SetTranslationContext(&translator.TranslationContext{
		Gateway:     modelGateway,
		Listener:    modelListener,
		Environment: modelEnv,
	})

	xdsResources, err := compositeTranslator.Translate(ctx, modelDeployment, irAPI, nodeID)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	return xdsResources, nil
}

// buildListeners generates xDS listeners at the gateway level.
// One xDS listener is created per physical listener resource, with a filter
// chain for each environment that has at least one successfully translated
// route configuration. This approach is correct because listeners are shared
// across deployments — multiple APIs on the same listener/environment share
// a single filter chain that references the same RDS route config name.
func (r *Reconciler) buildListeners(
	listeners []*resource.ListenerResource,
	envsByListener map[string][]*resource.EnvironmentResource,
	activeRoutes map[string]struct{}, // set of route config names that were successfully generated
) []*cache.ListenerWithName {
	var results []*cache.ListenerWithName

	for _, l := range listeners {
		envs := envsByListener[l.Meta.Name]
		if len(envs) == 0 {
			continue
		}

		var filterChains []*listenerbuilder.FilterChainConfig
		for _, env := range envs {
			routeName := fmt.Sprintf("route_%s_%s", l.Meta.Name, env.Meta.Name)
			if _, ok := activeRoutes[routeName]; !ok {
				continue // No successful deployment for this environment
			}
			filterChains = append(filterChains, &listenerbuilder.FilterChainConfig{
				Name:            env.Meta.Name,
				Hostname:        env.Spec.Hostname,
				HTTPFilters:     env.Spec.HTTPFilters,
				RouteConfigName: routeName,
			})
		}

		if len(filterChains) == 0 {
			continue
		}

		addr := l.Spec.Address
		if addr == "" {
			addr = "0.0.0.0"
		}

		config := &listenerbuilder.ListenerConfig{
			Name:         fmt.Sprintf("listener_%d", l.Spec.Port),
			Port:         l.Spec.Port,
			Address:      addr,
			FilterChains: filterChains,
			HTTP2:        l.Spec.HTTP2,
		}

		xdsListener, err := listenerbuilder.CreateListenerWithFilterChains(config)
		if err != nil {
			r.logger.WithFields(map[string]interface{}{
				"listener": l.Meta.Name,
				"error":    err.Error(),
			}).Error("Failed to create xDS listener")
			continue
		}

		results = append(results, &cache.ListenerWithName{Listener: xdsListener})
	}

	return results
}

// reconcileGatewayResource performs the full xDS reconciliation for a single gateway.
// It translates every deployment from scratch and replaces the entire xDS snapshot.
func (r *Reconciler) reconcileGatewayResource(ctx context.Context, gw *resource.GatewayResource) error {
	project := gw.Meta.Project
	nodeID := gw.Spec.NodeID

	r.logger.WithFields(map[string]interface{}{
		"gateway": gw.Meta.Name,
		"project": project,
		"nodeId":  nodeID,
	}).Info("Reconciling gateway")

	// Load all listeners referencing this gateway
	allListeners, err := r.typedStore.ListListeners(ctx, project)
	if err != nil {
		return fmt.Errorf("list listeners: %w", err)
	}
	var listeners []*resource.ListenerResource
	for _, l := range allListeners {
		if l.Spec.GatewayRef == gw.Meta.Name {
			listeners = append(listeners, l)
		}
	}

	// Load all environments referencing this gateway
	allEnvs, err := r.typedStore.ListEnvironments(ctx, project)
	if err != nil {
		return fmt.Errorf("list environments: %w", err)
	}
	envsByListener := make(map[string][]*resource.EnvironmentResource)
	for _, e := range allEnvs {
		if e.Spec.GatewayRef == gw.Meta.Name {
			envsByListener[e.Spec.ListenerRef] = append(envsByListener[e.Spec.ListenerRef], e)
		}
	}

	// Load all deployments referencing this gateway
	allDeployments, err := r.typedStore.ListDeployments(ctx, project)
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}
	var deployments []*resource.DeploymentResource
	for _, d := range allDeployments {
		if d.Spec.GatewayRef == gw.Meta.Name {
			deployments = append(deployments, d)
		}
	}

	// Single pass: translate each deployment, accumulate clusters + routes
	cacheDeployment := &cache.APIDeployment{}
	activeRoutes := make(map[string]struct{}) // route config names with successful translations

	for _, dep := range deployments {
		xds, err := r.translateDeployment(ctx, gw, dep, listeners, envsByListener)
		if err != nil {
			r.updateDeploymentStatus(ctx, dep, "Failed", err.Error())
			continue
		}

		cacheDeployment.Clusters = append(cacheDeployment.Clusters, xds.Clusters...)
		cacheDeployment.Endpoints = append(cacheDeployment.Endpoints, xds.Endpoints...)
		cacheDeployment.Routes = append(cacheDeployment.Routes, xds.Routes...)

		// Track which route config names were successfully generated
		for _, rc := range xds.Routes {
			activeRoutes[rc.Name] = struct{}{}
		}

		r.updateDeploymentStatus(ctx, dep, "Deployed", "")
	}

	// Generate listeners at the gateway level — one xDS listener per physical
	// listener, with filter chains only for environments that have route configs.
	for _, lw := range r.buildListeners(listeners, envsByListener, activeRoutes) {
		cacheDeployment.Listeners = append(cacheDeployment.Listeners, lw.Listener)
	}

	// Replace the entire snapshot — this is a from-scratch rebuild so we don't
	// want to merge with whatever was there before.
	if err := r.configManager.ReplaceSnapshot(nodeID, cacheDeployment); err != nil {
		return fmt.Errorf("replace xDS snapshot: %w", err)
	}

	// Update gateway status
	r.updateGatewayStatus(ctx, gw, "Ready")

	r.logger.WithFields(map[string]interface{}{
		"gateway":     gw.Meta.Name,
		"deployments": len(deployments),
		"clusters":    len(cacheDeployment.Clusters),
		"routes":      len(cacheDeployment.Routes),
		"listeners":   len(cacheDeployment.Listeners),
	}).Info("Gateway reconciliation complete")

	return nil
}

// upsertDeploymentResources translates a single deployment and merges its
// resources into the existing gateway snapshot via DeployAPI (additive upsert
// with dedup). Used when only one deployment changed.
func (r *Reconciler) upsertDeploymentResources(ctx context.Context, gatewayName, depName, depProject string) error {
	// Load the gateway
	gw, err := r.typedStore.GetGateway(ctx, depProject, gatewayName)
	if err != nil {
		return fmt.Errorf("get gateway %q: %w", gatewayName, err)
	}

	// Load the deployment
	dep, err := r.typedStore.GetDeployment(ctx, depProject, depName)
	if err != nil {
		return fmt.Errorf("get deployment %q: %w", depName, err)
	}

	// Load listeners and environments for context
	allListeners, err := r.typedStore.ListListeners(ctx, depProject)
	if err != nil {
		return fmt.Errorf("list listeners: %w", err)
	}
	var listeners []*resource.ListenerResource
	for _, l := range allListeners {
		if l.Spec.GatewayRef == gw.Meta.Name {
			listeners = append(listeners, l)
		}
	}

	allEnvs, err := r.typedStore.ListEnvironments(ctx, depProject)
	if err != nil {
		return fmt.Errorf("list environments: %w", err)
	}
	envsByListener := make(map[string][]*resource.EnvironmentResource)
	for _, e := range allEnvs {
		if e.Spec.GatewayRef == gw.Meta.Name {
			envsByListener[e.Spec.ListenerRef] = append(envsByListener[e.Spec.ListenerRef], e)
		}
	}

	// Translate this single deployment
	xds, err := r.translateDeployment(ctx, gw, dep, listeners, envsByListener)
	if err != nil {
		r.updateDeploymentStatus(ctx, dep, "Failed", err.Error())
		return fmt.Errorf("translate deployment %q: %w", depName, err)
	}

	// Build the activeRoutes set. We need to include both the new deployment's
	// routes AND all existing deployments' routes for the affected listener, so
	// the rebuilt listener has filter chains for all active environments.
	activeRoutes := make(map[string]struct{})
	for _, rc := range xds.Routes {
		activeRoutes[rc.Name] = struct{}{}
	}

	// Find other deployments on the same listener to include their route names
	allDeployments, err := r.typedStore.ListDeployments(ctx, depProject)
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}
	for _, d := range allDeployments {
		if d.Spec.GatewayRef == gw.Meta.Name && d.Spec.ListenerRef == dep.Spec.ListenerRef {
			routeName := fmt.Sprintf("route_%s_%s", d.Spec.ListenerRef, d.Spec.EnvironmentRef)
			activeRoutes[routeName] = struct{}{}
		}
	}

	// Build the listener for the affected listener resource (with all its envs)
	listenerResults := r.buildListeners(listeners, envsByListener, activeRoutes)

	// Merge into existing snapshot via DeployAPI (dedup handles replacements)
	cacheDeployment := &cache.APIDeployment{
		Clusters:  xds.Clusters,
		Endpoints: xds.Endpoints,
		Routes:    xds.Routes,
	}
	for _, lw := range listenerResults {
		cacheDeployment.Listeners = append(cacheDeployment.Listeners, lw.Listener)
	}

	if err := r.configManager.DeployAPI(gw.Spec.NodeID, cacheDeployment); err != nil {
		r.updateDeploymentStatus(ctx, dep, "Failed", fmt.Sprintf("deploy to xDS cache: %v", err))
		return fmt.Errorf("deploy to xDS cache: %w", err)
	}

	r.updateDeploymentStatus(ctx, dep, "Deployed", "")

	r.logger.WithFields(map[string]interface{}{
		"gateway":    gatewayName,
		"deployment": depName,
		"clusters":   len(xds.Clusters),
		"routes":     len(xds.Routes),
		"listeners":  len(cacheDeployment.Listeners),
	}).Info("Single deployment upsert complete")

	return nil
}

// removeDeploymentResources handles a deployment deletion by falling back to
// a full gateway rebuild. The deleted deployment is already gone from the store
// so it simply won't appear in the new snapshot.
func (r *Reconciler) removeDeploymentResources(ctx context.Context, gatewayName string) error {
	return r.reconcileGateway(ctx, gatewayName)
}

// --- Conversion helpers: resource types → models types (for xDS translator) ---

func toModelDeployment(dep *resource.DeploymentResource, api *resource.APIResource) *models.APIDeployment {
	now := time.Now()
	return &models.APIDeployment{
		ID:      dep.Meta.Name,
		Name:    api.Meta.Name,
		Version: api.Spec.Version,
		Context: api.Spec.Context,
		Status:  dep.Status.Phase,
		Metadata: types.FlowCMetadata{
			Name:    api.Meta.Name,
			Version: api.Spec.Version,
			Context: api.Spec.Context,
			APIType: api.Spec.APIType,
			Upstream: types.UpstreamConfig{
				Host:    api.Spec.Upstream.Host,
				Port:    api.Spec.Upstream.Port,
				Scheme:  api.Spec.Upstream.Scheme,
				Timeout: api.Spec.Upstream.Timeout,
			},
			Gateway: types.GatewayConfig{
				NodeID:      "", // filled via translation context
				Environment: dep.Spec.EnvironmentRef,
			},
		},
		CreatedAt: dep.Meta.CreatedAt,
		UpdatedAt: now,
	}
}

func toModelGateway(gw *resource.GatewayResource) *models.Gateway {
	return &models.Gateway{
		ID:        gw.Meta.Name,
		NodeID:    gw.Spec.NodeID,
		Name:      gw.Meta.Name,
		Status:    models.GatewayStatusConnected,
		Defaults:  gw.Spec.Defaults,
		Labels:    gw.Meta.Labels,
		CreatedAt: gw.Meta.CreatedAt,
		UpdatedAt: gw.Meta.UpdatedAt,
	}
}

func toModelListener(l *resource.ListenerResource) *models.Listener {
	ml := &models.Listener{
		ID:        l.Meta.Name,
		GatewayID: l.Spec.GatewayRef,
		Port:      l.Spec.Port,
		Address:   l.Spec.Address,
		HTTP2:     l.Spec.HTTP2,
		CreatedAt: l.Meta.CreatedAt,
		UpdatedAt: l.Meta.UpdatedAt,
	}
	if ml.Address == "" {
		ml.Address = "0.0.0.0"
	}
	if l.Spec.TLS != nil {
		ml.TLS = &models.TLSConfig{
			CertPath:          l.Spec.TLS.CertPath,
			KeyPath:           l.Spec.TLS.KeyPath,
			CAPath:            l.Spec.TLS.CAPath,
			RequireClientCert: l.Spec.TLS.RequireClientCert,
			MinVersion:        l.Spec.TLS.MinVersion,
			CipherSuites:      l.Spec.TLS.CipherSuites,
		}
	}
	return ml
}

func toModelEnvironment(e *resource.EnvironmentResource) *models.GatewayEnvironment {
	return &models.GatewayEnvironment{
		ID:          e.Meta.Name,
		ListenerID:  e.Spec.ListenerRef,
		Name:        e.Meta.Name,
		Hostname:    e.Spec.Hostname,
		HTTPFilters: e.Spec.HTTPFilters,
		Labels:      e.Meta.Labels,
		CreatedAt:   e.Meta.CreatedAt,
		UpdatedAt:   e.Meta.UpdatedAt,
	}
}

func normalizeBasePath(path string) string {
	if path == "" {
		return "/"
	}
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	if path[0] != '/' {
		path = "/" + path
	}
	return path
}

func unmarshalJSON(data json.RawMessage, v interface{}) error {
	return json.Unmarshal(data, v)
}
