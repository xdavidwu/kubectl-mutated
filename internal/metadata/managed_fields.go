package metadata

import (
	"bytes"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

var (
	fluxReconcileSet = fieldpath.NewSet(
		fieldpath.MakePathOrDie(
			"metadata",
			"annotations",
			"reconcile.fluxcd.io/requestedAt",
		),
		fieldpath.MakePathOrDie(
			"metadata",
			"annotations",
			"reconcile.fluxcd.io/forceAt",
		),
	)
)

// Returns if something useful is managed by a manual manager
//
// Manager is either explicitly specified, or from user-agent before '/'
// see k8s.io/apiserver/pkg/endpoints/handlers.managerOrUserAgent
func IsManualManager(e metav1.ManagedFieldsEntry) bool {
	if e.Manager == "flux" {
		s := fieldpath.Set{}
		err := s.FromJSON(bytes.NewBuffer(e.FieldsV1.Raw))
		if err != nil {
			return true
		}

		// flux reconcile
		if s.Leaves().Difference(fluxReconcileSet).Empty() {
			return false
		}
		return true
	}

	return (strings.HasPrefix(e.Manager, "kubectl") && e.Manager != "kubectl-rollout") ||
		// helm cli managed resources (via its generic client)
		// (flux helm-controller uses "helm-controller")
		e.Manager == "helm" ||
		// helm cli storage (secrets, configmaps) implicitly via user-agent Helm/<version>
		// (flux helm-controller users "helm-controller")
		e.Manager == "Helm" ||
		e.Manager == "Sparkles"
}
