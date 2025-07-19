package metadata

import (
	"bytes"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
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
			klog.Warning("found invalid FieldsV1", "manager", e.Manager, "fieldsV1", string(e.FieldsV1.Raw))
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

// TODO perhaps a cached variant by metadata.uid?
func FindSoleManualManagers(es []metav1.ManagedFieldsEntry) []metav1.ManagedFieldsEntry {
	candidates := []metav1.ManagedFieldsEntry{}

	systemManagedSet := &fieldpath.Set{}
	for _, e := range es {
		if IsManualManager(e) {
			candidates = append(candidates, e)
		} else {
			s := fieldpath.Set{}
			err := s.FromJSON(bytes.NewBuffer(e.FieldsV1.Raw))
			if err != nil {
				klog.Warning("found invalid FieldsV1", "manager", e.Manager, "fieldsV1", string(e.FieldsV1.Raw))
				continue
			}

			systemManagedSet = systemManagedSet.Union(s.Leaves()).Leaves()
		}
	}

	res := []metav1.ManagedFieldsEntry{}
	for _, e := range candidates {
		s := fieldpath.Set{}
		err := s.FromJSON(bytes.NewBuffer(e.FieldsV1.Raw))
		if err != nil {
			klog.Warning("found invalid FieldsV1", "manager", e.Manager, "fieldsV1", string(e.FieldsV1.Raw))
			res = append(res, e)
			continue
		}

		if !s.Leaves().Difference(systemManagedSet).Empty() {
			res = append(res, e)
		}
	}
	return res
}

func SolelyManuallyManagedSet(mfs []metav1.ManagedFieldsEntry) (*fieldpath.Set, error) {
	s := &fieldpath.Set{}
	for _, mf := range FindSoleManualManagers(mfs) {
		ms := &fieldpath.Set{}
		err := ms.FromJSON(bytes.NewBuffer(mf.FieldsV1.Raw))
		if err != nil {
			return nil, err
		}
		s = s.Union(ms)
	}
	return s.Leaves(), nil
}
