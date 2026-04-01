package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flowc-labs/flowc/internal/flowc/ir"
	"github.com/flowc-labs/flowc/internal/flowc/profile"
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
	vhostsByListener map[string][]*resource.VirtualHostResource,
) (*translator.XDSResources, error) {
	nodeID := gw.Spec.NodeID

	// Load the referenced API
	api, err := r.typedStore.GetAPI(ctx, dep.Spec.APIRef)
	if err != nil {
		return nil, fmt.Errorf("API %q not found: %w", dep.Spec.APIRef, err)
	}

	// Resolve listener ref (auto-select if omitted and unambiguous)
	listenerRef := dep.Spec.Gateway.Listener
	if listenerRef == "" {
		if len(listeners) == 0 {
			return nil, fmt.Errorf("no listeners found for gateway %q", gw.Meta.Name)
		}
		if len(listeners) > 1 {
			return nil, fmt.Errorf("multiple listeners found for gateway %q; spec.listenerRef is required", gw.Meta.Name)
		}
		listenerRef = listeners[0].Meta.Name
	}

	// Find the listener for this deployment
	var listener *resource.ListenerResource
	for _, l := range listeners {
		if l.Meta.Name == listenerRef {
			listener = l
			break
		}
	}
	if listener == nil {
		return nil, fmt.Errorf("Listener %q not found", listenerRef)
	}

	// Resolve virtual host ref (auto-select if omitted and unambiguous)
	vhostRef := dep.Spec.Gateway.VirtualHost
	if vhostRef == "" {
		vhosts := vhostsByListener[listenerRef]
		if len(vhosts) == 0 {
			return nil, fmt.Errorf("no virtual hosts found for listener %q", listenerRef)
		}
		if len(vhosts) > 1 {
			return nil, fmt.Errorf("multiple virtual hosts found for listener %q; spec.virtualHostRef is required", listenerRef)
		}
		vhostRef = vhosts[0].Meta.Name
	}

	// Find the virtual host for this deployment
	var vhost *resource.VirtualHostResource
	for _, v := range vhostsByListener[listenerRef] {
		if v.Meta.Name == vhostRef {
			vhost = v
			break
		}
	}
	if vhost == nil {
		return nil, fmt.Errorf("VirtualHost %q not found", vhostRef)
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
	modelVHost := toModelVirtualHost(vhost)

	// Load profile defaults if the gateway references a profile
	var profileDefaults *types.StrategyConfig
	if gw.Spec.ProfileRef != "" {
		profileDefaults = profile.GetProfileDefaults(ctx, r.typedStore, gw.Spec.ProfileRef)
	}

	// Resolve strategies (4-level precedence: API > Gateway > Profile > Builtin)
	resolver := translator.NewConfigResolver(profileDefaults, gw.Spec.Defaults, r.logger)
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
		VirtualHost: modelVHost,
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
	vhostsByListener map[string][]*resource.VirtualHostResource,
	activeRoutes map[string]struct{}, // set of route config names that were successfully generated
) []*cache.ListenerWithName {
	var results []*cache.ListenerWithName

	for _, l := range listeners {
		vhosts := vhostsByListener[l.Meta.Name]
		if len(vhosts) == 0 {
			continue
		}

		var filterChains []*listenerbuilder.FilterChainConfig
		for _, vh := range vhosts {
			routeName := fmt.Sprintf("route_%s_%s", l.Meta.Name, vh.Meta.Name)
			if _, ok := activeRoutes[routeName]; !ok {
				continue // No successful deployment for this virtual host
			}
			filterChains = append(filterChains, &listenerbuilder.FilterChainConfig{
				Name:            vh.Meta.Name,
				Hostname:        vh.Spec.Hostname,
				HTTPFilters:     vh.Spec.HTTPFilters,
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
	nodeID := gw.Spec.NodeID

	r.logger.WithFields(map[string]interface{}{
		"gateway": gw.Meta.Name,
		"nodeId":  nodeID,
	}).Info("Reconciling gateway")

	// Load all listeners referencing this gateway
	allListeners, err := r.typedStore.ListListeners(ctx)
	if err != nil {
		return fmt.Errorf("list listeners: %w", err)
	}
	var listeners []*resource.ListenerResource
	for _, l := range allListeners {
		if l.Spec.GatewayRef == gw.Meta.Name {
			listeners = append(listeners, l)
		}
	}

	// Load all virtual hosts referencing this gateway
	allVHosts, err := r.typedStore.ListVirtualHosts(ctx)
	if err != nil {
		return fmt.Errorf("list virtual hosts: %w", err)
	}
	vhostsByListener := make(map[string][]*resource.VirtualHostResource)
	for _, v := range allVHosts {
		if v.Spec.GatewayRef == gw.Meta.Name {
			vhostsByListener[v.Spec.ListenerRef] = append(vhostsByListener[v.Spec.ListenerRef], v)
		}
	}

	// Load all deployments referencing this gateway
	allDeployments, err := r.typedStore.ListDeployments(ctx)
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}
	var deployments []*resource.DeploymentResource
	for _, d := range allDeployments {
		if d.Spec.Gateway.Name == gw.Meta.Name {
			deployments = append(deployments, d)
		}
	}

	// Single pass: translate each deployment, accumulate clusters + routes
	cacheDeployment := &cache.APIDeployment{}
	activeRoutes := make(map[string]struct{}) // route config names with successful translations

	for _, dep := range deployments {
		xds, err := r.translateDeployment(ctx, gw, dep, listeners, vhostsByListener)
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
	// listener, with filter chains only for virtual hosts that have route configs.
	for _, lw := range r.buildListeners(listeners, vhostsByListener, activeRoutes) {
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
func (r *Reconciler) upsertDeploymentResources(ctx context.Context, gatewayName, depName string) error {
	// Load the gateway
	gw, err := r.typedStore.GetGateway(ctx, gatewayName)
	if err != nil {
		return fmt.Errorf("get gateway %q: %w", gatewayName, err)
	}

	// Load the deployment
	dep, err := r.typedStore.GetDeployment(ctx, depName)
	if err != nil {
		return fmt.Errorf("get deployment %q: %w", depName, err)
	}

	// Load listeners and environments for context
	allListeners, err := r.typedStore.ListListeners(ctx)
	if err != nil {
		return fmt.Errorf("list listeners: %w", err)
	}
	var listeners []*resource.ListenerResource
	for _, l := range allListeners {
		if l.Spec.GatewayRef == gw.Meta.Name {
			listeners = append(listeners, l)
		}
	}

	allVHosts, err := r.typedStore.ListVirtualHosts(ctx)
	if err != nil {
		return fmt.Errorf("list virtual hosts: %w", err)
	}
	vhostsByListener := make(map[string][]*resource.VirtualHostResource)
	for _, v := range allVHosts {
		if v.Spec.GatewayRef == gw.Meta.Name {
			vhostsByListener[v.Spec.ListenerRef] = append(vhostsByListener[v.Spec.ListenerRef], v)
		}
	}

	// Translate this single deployment
	xds, err := r.translateDeployment(ctx, gw, dep, listeners, vhostsByListener)
	if err != nil {
		r.updateDeploymentStatus(ctx, dep, "Failed", err.Error())
		return fmt.Errorf("translate deployment %q: %w", depName, err)
	}

	// Build the activeRoutes set. We need to include both the new deployment's
	// routes AND all existing deployments' routes for the affected listener, so
	// the rebuilt listener has filter chains for all active virtual hosts.
	activeRoutes := make(map[string]struct{})
	for _, rc := range xds.Routes {
		activeRoutes[rc.Name] = struct{}{}
	}

	// Find other deployments on the same listener to include their route names
	allDeployments, err := r.typedStore.ListDeployments(ctx)
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}
	for _, d := range allDeployments {
		if d.Spec.Gateway.Name == gw.Meta.Name && d.Spec.Gateway.Listener == dep.Spec.Gateway.Listener {
			routeName := fmt.Sprintf("route_%s_%s", d.Spec.Gateway.Listener, d.Spec.Gateway.VirtualHost)
			activeRoutes[routeName] = struct{}{}
		}
	}

	// Build the listener for the affected listener resource (with all its virtual hosts)
	listenerResults := r.buildListeners(listeners, vhostsByListener, activeRoutes)

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
				NodeID:         "", // filled via translation context
				VirtualHostRef: dep.Spec.Gateway.VirtualHost,
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

func toModelVirtualHost(v *resource.VirtualHostResource) *models.GatewayVirtualHost {
	return &models.GatewayVirtualHost{
		ID:          v.Meta.Name,
		ListenerID:  v.Spec.ListenerRef,
		Name:        v.Meta.Name,
		Hostname:    v.Spec.Hostname,
		HTTPFilters: v.Spec.HTTPFilters,
		Labels:      v.Meta.Labels,
		CreatedAt:   v.Meta.CreatedAt,
		UpdatedAt:   v.Meta.UpdatedAt,
	}
}

func normalizeBasePath(path string) string {
	if path == "" || path == "/" {
		return ""
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
