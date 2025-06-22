package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// like k8s.io/cli-runtime/pkg/printers.printRows
// cli-runtime printers assume single kind for whole table, but ours may vary
func formatNameColumn(o metav1.Object, gvk schema.GroupVersionKind) string {
	return fmt.Sprintf(
		"%s/%s",
		strings.ToLower(gvk.GroupKind().String()),
		o.GetName(),
	)
}

func main() {
	var fs flag.FlagSet
	klog.InitFlags(&fs)
	pflag.CommandLine.AddGoFlagSet(&fs)
	cflags := genericclioptions.NewConfigFlags(true)
	cflags.AddFlags(pflag.CommandLine)
	pflag.Parse()

	dc, err := cflags.ToDiscoveryClient()
	if err != nil {
		klog.Fatalf("cannot get discovery client: %s", err)
	}

	w := printers.GetNewTabWriter(os.Stdout)
	defer w.Flush()
	fmt.Fprintln(w, "NAME\tMANAGERS")

	// TODO namespaced
	resources, err := dc.ServerPreferredResources()
	if err != nil {
		klog.Fatalf("cannot perform discovery: %s", err)
	}

	for _, rlist := range resources {
		gv, err := schema.ParseGroupVersion(rlist.GroupVersion)
		if err != nil {
			klog.Fatalf("cannot parse GroupVersion: %s", err)
		}
		for _, r := range rlist.APIResources {
			if !slices.Contains(r.Verbs, "list") {
				continue
			}

			klog.V(1).Infof("fetching %s %s", rlist.GroupVersion, r.Name)
			// XXX metrics.k8s.io discovery v2 seems to return wrong responseKind group version
			gvr := gv.WithResource(r.Name)
			gvk := gv.WithKind(r.Kind)

			// XXX QPS doesn't seem to work across builders?
			v := resource.NewBuilder(cflags).
				Unstructured().
				SelectAllParam(true).
				RequestChunksOf(512).
				// TODO don't on other output formats
				TransformRequests(func(req *rest.Request) {
					// XXX protobuf seems not to work with builder
					// TODO handle stuff without PartialObjectMetadataList support? (aggregated apis?)
					req.SetHeader("Accept", "application/json;as=PartialObjectMetadataList;g=meta.k8s.io;v=v1")
				}).
				// builder uses schema.Parse{Resource,Kind}Arg
				// resource.version.group: pod.v1. works but pod.v1 does not
				// not gvr.String()
				ResourceTypes(fmt.Sprintf("%s.%s.%s", gvr.Resource, gvr.Version, gvr.Group)).
				Do()
			err = v.Err()
			if err != nil {
				klog.Fatalf("cannot build visitor: %s", err)
			}
			err = v.Visit(func(i *resource.Info, e error) error {
				if e != nil {
					return e
				}

				l, ok := i.Object.(*unstructured.UnstructuredList)
				if !ok {
					return fmt.Errorf("unexpected type")
				}
				for _, i := range l.Items {
					managers := []string{}
					for _, mf := range i.GetManagedFields() {
						if strings.HasPrefix(mf.Manager, "kubectl") {
							managers = append(managers, mf.Manager)
							// TODO find a way to show fieldsV1?
							klog.V(2).Infof("%s %s %s managed by %s: %v", rlist.GroupVersion, gvr.Resource, i.GetName(), mf.Manager, mf.FieldsV1)
						}
					}
					if len(managers) > 0 {
						fmt.Fprintf(w, "%s\t%s\n", formatNameColumn(&i, gvk), strings.Join(managers, ","))
					}
				}
				return nil
			})
			if err != nil {
				klog.Warningf("cannot list %s %s: %s", rlist.GroupVersion, gvr.Resource, err)
			}
		}
	}
}
