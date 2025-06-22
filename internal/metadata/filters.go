package metadata

import (
	"fmt"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"
)

func HasManuallyManagedFields(i *resource.Info, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	o, ok := i.Object.(*metav1.PartialObjectMetadata)
	if !ok {
		return false, fmt.Errorf("unexpected type")
	}

	return slices.ContainsFunc(
		o.GetManagedFields(),
		func(mf metav1.ManagedFieldsEntry) bool {
			return IsManualManager(mf.Manager)
		},
	), nil
}
