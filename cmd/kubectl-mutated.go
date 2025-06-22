package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"

	"github.com/xdavidwu/kubectl-mutated/internal/metadata"
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

func must(op string, err error) {
	if err != nil {
		klog.Fatalf("cannot %s: %s", op, err)
	}
}

func main() {
	var fs flag.FlagSet
	klog.InitFlags(&fs)
	pflag.CommandLine.AddGoFlagSet(&fs)
	cflags := genericclioptions.NewConfigFlags(true)
	cflags.AddFlags(pflag.CommandLine)
	rflags := (&genericclioptions.ResourceBuilderFlags{}).
		WithAllNamespaces(false)
	rflags.AddFlags(pflag.CommandLine)
	pflag.Parse()

	dc, err := cflags.ToDiscoveryClient()
	must("get discovery client", err)

	// namespace may come from kubeconfig, not just cli flags
	// this is normally hidden under ResourceBuilderFlags.ToBuilder
	// but that prevents further builder config
	ns, _, err := cflags.ToRawKubeConfigLoader().Namespace()
	must("read config", err)

	w := printers.GetNewTabWriter(os.Stdout)
	defer w.Flush()
	if *rflags.AllNamespaces {
		fmt.Fprint(w, "NAMESPACE\t")
	}
	fmt.Fprintln(w, "NAME\tMANAGERS")

	var resources []*metav1.APIResourceList
	if *rflags.AllNamespaces {
		resources, err = dc.ServerPreferredResources()
	} else {
		resources, err = dc.ServerPreferredNamespacedResources()
	}
	must("perform discovery", err)

	scheme := runtime.NewScheme()
	must("build metav1 scheme", metav1.AddMetaToScheme(scheme))

	for _, rlist := range resources {
		gv, err := schema.ParseGroupVersion(rlist.GroupVersion)
		must("parse GroupVersion", err)
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
				WithScheme(scheme, metav1.SchemeGroupVersion).
				SelectAllParam(true).
				NamespaceParam(ns).
				DefaultNamespace().
				AllNamespaces(*rflags.AllNamespaces).
				RequestChunksOf(512).
				// TODO don't on other output formats
				// TODO handle stuff without PartialObjectMetadataList support? (aggregated apis?)
				TransformRequests(metadata.ToPartialObjectMetadataList).
				// builder uses schema.Parse{Resource,Kind}Arg
				// resource.version.group: pod.v1. works but pod.v1 does not
				// not gvr.String()
				ResourceTypes(fmt.Sprintf("%s.%s.%s", gvr.Resource, gvr.Version, gvr.Group)).
				Flatten().
				// for disabling mapper for Flatten(),
				// avoid attempt on PartialObjectMetadata,
				// still perform lists
				Local().
				Do()
			must("build visitor", err)
			err = v.Visit(func(i *resource.Info, e error) error {
				if e != nil {
					return e
				}

				o, ok := i.Object.(*metav1.PartialObjectMetadata)
				if !ok {
					return fmt.Errorf("unexpected type")
				}
				managers := []string{}
				for _, mf := range o.GetManagedFields() {
					if metadata.IsManualManager(mf.Manager) {
						managers = append(managers, mf.Manager)
						// TODO find a way to show fieldsV1?
						klog.V(2).Infof("%s %s %s managed by %s: %v", rlist.GroupVersion, gvr.Resource, o.GetName(), mf.Manager, mf.FieldsV1)
					}
				}
				if len(managers) > 0 {
					if *rflags.AllNamespaces {
						if r.Namespaced {
							fmt.Fprint(w, o.GetNamespace())
							fmt.Fprint(w, "\t")
						} else {
							fmt.Fprint(w, "<none>\t")
						}
					}
					fmt.Fprintf(w, "%s\t%s\n", formatNameColumn(o, gvk), strings.Join(managers, ","))
				}
				return nil
			})
			if err != nil {
				klog.Warningf("cannot list %s %s: %s", rlist.GroupVersion, gvr.Resource, err)
			}
		}
	}
}
