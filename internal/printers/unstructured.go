package printers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type unstructuredPrinter struct {
	withNamespace bool
}

func (unstructuredPrinter) ConfigureBuilder(r *resource.Builder, gvk schema.GroupVersionKind) *resource.Builder {
	// use scheme if possible, to utilize protobuf
	if scheme.Scheme.Recognizes(gvk) {
		return r.WithScheme(scheme.Scheme, gvk.GroupVersion()).
			TransformRequests(func(req *rest.Request) {
				req.SetHeader("Accept", "application/vnd.kubernetes.protobuf,application/json")
			})
	}
	return r.Unstructured()
}

func (p unstructuredPrinter) toUnstructured(o runtime.Object, gvk schema.GroupVersionKind) (*unstructured.Unstructured, error) {
	u, ok := o.(*unstructured.Unstructured)
	if !ok {
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
		if err != nil {
			return nil, err
		}
		u = &unstructured.Unstructured{Object: obj}
		u.SetGroupVersionKind(gvk)
	}

	if !p.withNamespace {
		delete(u.Object["metadata"].(map[string]any), "namespace")
	}
	return u, nil
}
