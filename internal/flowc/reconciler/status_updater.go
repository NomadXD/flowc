package reconciler

import (
	"context"
	"encoding/json"

	"github.com/flowc-labs/flowc/internal/flowc/resource"
	"github.com/flowc-labs/flowc/internal/flowc/resource/store"
)

// updateDeploymentStatus updates the status of a deployment resource.
func (r *Reconciler) updateDeploymentStatus(ctx context.Context, dep *resource.DeploymentResource, phase, message string) {
	dep.Status.Phase = phase

	reason := "Reconciled"
	condStatus := "True"
	if phase == "Failed" {
		reason = "ReconcileFailed"
		condStatus = "False"
	}

	dep.Status.Conditions = resource.SetCondition(dep.Status.Conditions, resource.Condition{
		Type:    "Ready",
		Status:  condStatus,
		Reason:  reason,
		Message: message,
	})

	statusJSON, err := json.Marshal(dep.Status)
	if err != nil {
		r.logger.WithError(err).Error("Failed to marshal deployment status")
		return
	}

	// Read current stored resource to update status only
	key := dep.Meta.Key()
	stored, err := r.store.Get(ctx, key)
	if err != nil {
		r.logger.WithFields(map[string]interface{}{
			"deployment": dep.Meta.Name,
			"error":      err.Error(),
		}).Error("Failed to get deployment for status update")
		return
	}

	stored.StatusJSON = statusJSON
	_, err = r.store.Put(ctx, stored, store.PutOptions{
		ExpectedRevision: stored.Meta.Revision,
	})
	if err != nil {
		r.logger.WithFields(map[string]interface{}{
			"deployment": dep.Meta.Name,
			"error":      err.Error(),
		}).Warn("Failed to update deployment status (may have been modified)")
	}
}

// updateGatewayStatus updates the status of a gateway resource.
func (r *Reconciler) updateGatewayStatus(ctx context.Context, gw *resource.GatewayResource, phase string) {
	gw.Status.Phase = phase
	gw.Status.Conditions = resource.SetCondition(gw.Status.Conditions, resource.Condition{
		Type:   "Ready",
		Status: "True",
		Reason: "Reconciled",
	})

	statusJSON, err := json.Marshal(gw.Status)
	if err != nil {
		r.logger.WithError(err).Error("Failed to marshal gateway status")
		return
	}

	key := gw.Meta.Key()
	stored, err := r.store.Get(ctx, key)
	if err != nil {
		return
	}

	stored.StatusJSON = statusJSON
	r.store.Put(ctx, stored, store.PutOptions{
		ExpectedRevision: stored.Meta.Revision,
	})
}
