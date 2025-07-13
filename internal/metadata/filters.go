package metadata

import (
	"fmt"

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

	return len(FindSoleManualManagers(o.GetManagedFields())) > 0, nil
}
