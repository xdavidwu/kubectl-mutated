package printers

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/liggitt/tabwriter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	crprinters "k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"

	"github.com/xdavidwu/kubectl-mutated/internal/metadata"
)

var (
	metav1Scheme = runtime.NewScheme()
)

func init() {
	if err := metav1.AddMetaToScheme(metav1Scheme); err != nil {
		panic(fmt.Errorf("cannot build metav1 scheme: %s", err))
	}
}

// like k8s.io/cli-runtime/pkg/printers.printRows
// cli-runtime printers assume single kind for whole table, but ours may vary
func formatNameColumn(o metav1.Object, gvk schema.GroupVersionKind) string {
	return fmt.Sprintf(
		"%s/%s",
		strings.ToLower(gvk.GroupKind().String()),
		o.GetName(),
	)
}

type TablePrinter struct {
	w             *tabwriter.Writer
	withNamespace bool
}

func NewTablePrinter(o io.Writer, withNamespace bool) (*TablePrinter, error) {
	w := crprinters.GetNewTabWriter(o)

	if withNamespace {
		if _, err := fmt.Fprint(w, "NAMESPACE\t"); err != nil {
			return nil, err
		}
	}
	if _, err := fmt.Fprintln(w, "NAME\tMANAGERS\tCOUNT"); err != nil {
		return nil, err
	}

	return &TablePrinter{w: w, withNamespace: withNamespace}, nil
}

func (TablePrinter) ConfigureBuilder(r *resource.Builder, _ schema.GroupVersionKind) *resource.Builder {
	return r.WithScheme(metav1Scheme, metav1.SchemeGroupVersion).
		// TODO handle stuff without PartialObjectMetadataList support? (aggregated apis?)
		TransformRequests(metadata.ToPartialObjectMetadataList).
		// for disabling mapper for Flatten(),
		// avoid attempt on PartialObjectMetadata,
		// still perform lists
		Local()
}

func (t *TablePrinter) PrintObject(r runtime.Object, gvk schema.GroupVersionKind) error {
	o, ok := r.(metav1.Object)
	if !ok {
		return fmt.Errorf("unexpected type")
	}

	m := map[string]bool{}
	for _, mf := range metadata.FindSoleManualManagers(o.GetManagedFields()) {
		m[mf.Manager] = true
	}
	managers := slices.Collect(maps.Keys(m))
	slices.Sort(managers)

	s, err := metadata.SolelyManuallyManagedSet(o.GetManagedFields())
	if err != nil {
		return fmt.Errorf("cannot conclude field set: %s", err)
	}
	c := s.Size()

	// TODO find a way to show fieldsV1?
	if t.withNamespace {
		ns := o.GetNamespace()
		var err error
		if ns != "" {
			_, err = fmt.Fprint(t.w, ns, "\t")
		} else {
			_, err = fmt.Fprint(t.w, "<none>\t")
		}
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(
		t.w,
		"%s\t%s\t%d\n",
		formatNameColumn(o, gvk),
		strings.Join(managers, ","),
		c,
	)
	return err
}

func (t *TablePrinter) Flush() error {
	return t.w.Flush()
}
