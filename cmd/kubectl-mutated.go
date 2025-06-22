package main

import (
	"context"
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
	"k8s.io/client-go/metadata"
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
	config, err := cflags.ToRESTConfig()
	if err != nil {
		klog.Fatalf("cannot get RESTConfig: %s", err)
	}
	// TODO switch to dynamic/unstructured on other output formats
	c, err := metadata.NewForConfig(config)
	if err != nil {
		klog.Fatalf("cannot get RESTClient: %s", err)
	}

	// TODO namespaced
	resources, err := dc.ServerPreferredResources()
	if err != nil {
		klog.Fatalf("cannot perform discovery: %s", err)
	}

	w := printers.GetNewTabWriter(os.Stdout)
	defer w.Flush()
	fmt.Fprintln(w, "NAME\tMANAGERS")

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
			rc := c.Resource(gv.WithResource(r.Name))
			gvk := schema.GroupVersionKind{Group: r.Group, Version: r.Version, Kind: r.Kind}
			err := resource.FollowContinue(
				&metav1.ListOptions{Limit: 512}, // TODO
				func(o metav1.ListOptions) (runtime.Object, error) {
					l, err := rc.List(context.TODO(), o)
					for _, i := range l.Items {
						managers := []string{}
						for _, mf := range i.GetManagedFields() {
							if strings.HasPrefix(mf.Manager, "kubectl") {
								managers = append(managers, mf.Manager)
								// TODO find a way to show fieldsV1?
								klog.V(2).Infof("%s %s %s managed by %s: %v", rlist.GroupVersion, r.Name, i.GetName(), mf.Manager, mf.FieldsV1)
							}
						}
						if len(managers) > 0 {
							fmt.Fprintf(w, "%s\t%s\n", formatNameColumn(&i, gvk), strings.Join(managers, ","))
						}
					}
					return l, err
				},
			)
			if err != nil {
				klog.Warningf("cannot list %s %s: %s", rlist.GroupVersion, r.Name, err)
			}
		}
	}
}
