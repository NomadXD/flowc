package profile

import (
	"context"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
	"github.com/flowc-labs/flowc/internal/flowc/resource/store"
	"github.com/flowc-labs/flowc/pkg/logger"
	"github.com/flowc-labs/flowc/pkg/types"
)

const (
	// builtinManager is the ownership tag for built-in profiles.
	builtinManager = "flowc-builtin"
)

// BuiltinProfiles returns all built-in gateway profiles.
func BuiltinProfiles() []*resource.GatewayProfileResource {
	return []*resource.GatewayProfileResource{
		edgeProfile(),
		mediationProfile(),
		sidecarProfile(),
		egressProfile(),
		aiProfile(),
	}
}

// SeedBuiltinProfiles writes built-in profiles to the store if they don't
// already exist. Called once at startup.
func SeedBuiltinProfiles(ctx context.Context, ts *store.TypedStore, log *logger.EnvoyLogger) error {
	for _, p := range BuiltinProfiles() {
		// Check if it already exists — don't overwrite user customizations.
		_, err := ts.GetGatewayProfile(ctx, p.Meta.Name)
		if err == nil {
			continue // already exists
		}

		p.Meta.Kind = resource.KindGatewayProfile
		_, err = ts.PutGatewayProfile(ctx, p, store.PutOptions{ManagedBy: builtinManager})
		if err != nil {
			log.WithFields(map[string]interface{}{
				"profile": p.Meta.Name,
				"error":   err.Error(),
			}).Warn("Failed to seed built-in profile")
			continue
		}

		log.WithFields(map[string]interface{}{
			"profile": p.Meta.Name,
			"type":    p.Spec.ProfileType,
		}).Info("Seeded built-in gateway profile")
	}
	return nil
}

// GetProfileDefaults retrieves the strategy defaults for a named profile.
// Returns nil if the profile doesn't exist.
func GetProfileDefaults(ctx context.Context, ts *store.TypedStore, profileName string) *types.StrategyConfig {
	profile, err := ts.GetGatewayProfile(ctx, profileName)
	if err == nil {
		return profile.Spec.DefaultStrategy
	}
	return nil
}
