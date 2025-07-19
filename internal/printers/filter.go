package printers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v6/value"

	"github.com/xdavidwu/kubectl-mutated/internal/metadata"
)

func filterMap(v map[string]any, s *fieldpath.Set) (map[string]any, error) {
	res := map[string]any{}

	for p := range s.Members.All() {
		if p.FieldName == nil {
			return nil, fmt.Errorf("path of unexpected type: %s", p)
		}

		c, ok := v[*p.FieldName]
		if !ok {
			return nil, fmt.Errorf("missing field: %s", p)
		}
		res[*p.FieldName] = c
	}

	for p := range s.Children.All() {
		if p.FieldName == nil {
			return nil, fmt.Errorf("path of unexpected type: %s", p)
		}

		c, ok := v[*p.FieldName]
		if !ok {
			return nil, fmt.Errorf("missing field: %s", p)
		}
		var err error
		res[*p.FieldName], err = doFilter(c, s.Children.Descend(p))
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func findIndex(v []any, p fieldpath.PathElement) (int, error) {
	i := -1
	switch {
	case p.Key != nil:
	MapLoop:
		for ii, vv := range v {
			vm, ok := vv.(map[string]any)
			if !ok {
				return 0, fmt.Errorf("unexpected type %T", vv)
			}

			for _, f := range *p.Key {
				c, ok := vm[f.Name]
				if !ok {
					continue MapLoop
				}
				if !value.Equals(value.NewValueInterface(c), f.Value) {
					continue MapLoop
				}
			}
			i = ii
			break
		}
	case p.Value != nil:
		for ii, vv := range v {
			if value.Equals(value.NewValueInterface(vv), *p.Value) {
				i = ii
				break
			}
		}
	case p.Index != nil:
		if *p.Index < len(v) {
			i = *p.Index
		}
	default:
		return 0, fmt.Errorf("path of unexpected type: %s", p)
	}
	if i == -1 {
		return 0, fmt.Errorf("no match for path: %s", p)
	}
	return i, nil
}

func filterSlice(v []any, s *fieldpath.Set) ([]any, error) {
	used := make([]bool, len(v))
	vals := make([]any, len(v))

	for p := range s.Members.All() {
		i, err := findIndex(v, p)
		if err != nil {
			return nil, err
		}

		used[i] = true
		vals[i] = v[i]
	}

	for p := range s.Children.All() {
		i, err := findIndex(v, p)
		if err != nil {
			return nil, err
		}

		used[i] = true
		vals[i], err = doFilter(v[i], s.Children.Descend(p))
		if err != nil {
			return nil, err
		}
	}

	res := []any{}
	for i, c := range vals {
		if used[i] {
			res = append(res, c)
		}
	}
	return res, nil
}

func doFilter(v any, s *fieldpath.Set) (any, error) {
	switch v := v.(type) {
	case map[string]any:
		return filterMap(v, s)
	case []any:
		return filterSlice(v, s)
	default:
		return v, nil
	}
}

func Filter(u *unstructured.Unstructured, s *fieldpath.Set) (*unstructured.Unstructured, error) {
	res, err := filterMap(u.Object, s)
	if err != nil {
		return nil, err
	}

	r := unstructured.Unstructured{Object: res}

	// ensure identifying fields are set
	r.SetAPIVersion(u.GetAPIVersion())
	r.SetKind(u.GetKind())
	r.SetName(u.GetName())
	ns := u.GetNamespace()
	if ns != "" {
		r.SetNamespace(ns)
	}
	return &r, nil
}

type filteredPrinter struct {
	unstructuredPrinter
}

func (p *filteredPrinter) getFilteredObject(r runtime.Object, gvk schema.GroupVersionKind) (*unstructured.Unstructured, error) {
	o, err := p.toUnstructured(r, gvk)
	if err != nil {
		return nil, fmt.Errorf("cannot convert to unstructured: %s", err)
	}

	c := o.DeepCopy()
	c.SetManagedFields(nil)

	s, err := metadata.SolelyManuallyManagedSet(o.GetManagedFields())
	if err != nil {
		return nil, fmt.Errorf("cannot conclude field set: %s", err)
	}

	f, err := Filter(c, s)
	if err != nil {
		return nil, fmt.Errorf("cannot filter resource: %s", err)
	}

	return f, nil
}
