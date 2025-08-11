package main

import (
	"flag"
	"fmt"
	"maps"
	"strings"

	"os"
	"runtime/debug"
	"slices"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"

	"github.com/xdavidwu/kubectl-mutated/internal/completion"
	"github.com/xdavidwu/kubectl-mutated/internal/metadata"
	"github.com/xdavidwu/kubectl-mutated/internal/printers"
)

type printerOption struct {
	desc string
	get  func() (printers.Printer, error)
}

var (
	mutatedCmd = &cobra.Command{
		Use:  "kubectl-mutated",
		Long: "Show what resources have been mutated by a field manager that might be operated manually, like kubectl",
		Example: `  # List such resources under current namespace
  kubectl mutated

  # List such resources under namespace "my-space"
  kubectl mutated -n my-space

  # List such resources of all types under any namespaces, including cluster-scoped resources
  kubectl mutated --all-namespaces`,
		Annotations: map[string]string{
			cobra.CommandDisplayNameAnnotation: "kubectl mutated",
		},
		PreRunE: cobra.NoArgs,
		Run:     mutated,
	}

	cflags = genericclioptions.NewConfigFlags(true)
	rflags = (&genericclioptions.ResourceBuilderFlags{}).
		WithAllNamespaces(false).
		WithLabelSelector("")
	output *string

	printerOptions = map[string]printerOption{
		"hyaml": {
			"YAML stream with mutated fields highlighted",
			func() (printers.Printer, error) {
				return printers.NewHighlightedYAMLPrinter(*rflags.AllNamespaces)
			},
		},
		"fyaml": {
			"YAML stream filtered to mutated fields",
			func() (printers.Printer, error) {
				return printers.NewFilteredYAMLPrinter(*rflags.AllNamespaces)
			},
		},
		"fjson": {
			"JSON filtered to mutated fields",
			func() (printers.Printer, error) {
				return printers.NewFilteredJSONPrinter(*rflags.AllNamespaces)
			},
		},
		"": {
			"Table with manual managers and mutated fields count",
			func() (printers.Printer, error) {
				return printers.NewTablePrinter(os.Stdout, *rflags.AllNamespaces)
			},
		},
	}
)

func init() {
	pflag := mutatedCmd.Flags()

	var fs flag.FlagSet
	klog.InitFlags(&fs)
	pflag.AddGoFlagSet(&fs)
	cflags.AddFlags(pflag)
	rflags.AddFlags(pflag)

	popts := make([]string, 0, len(printerOptions))
	descs := make([]string, 0, len(printerOptions))
	for _, k := range slices.Sorted(maps.Keys(printerOptions)) {
		disp := k
		if k == "" {
			disp = "\"\""
		}
		popts = append(popts, disp)
		descs = append(descs, fmt.Sprintf("%s:\t%s", disp, printerOptions[k].desc))
	}
	output = pflag.StringP("output", "o", "",
		fmt.Sprintf(
			"Output format. One of: (%s)\n%s",
			strings.Join(popts, ", "),
			strings.Join(descs, "\n"),
		))
	pflag.SortFlags = false

	must(
		"register config flags completions",
		completion.RegisterConfigFlagsCompletion(mutatedCmd, cflags),
	)

	oc := make([]cobra.Completion, 0, len(printerOptions))
	for k, v := range printerOptions {
		oc = append(oc, cobra.CompletionWithDesc(k, v.desc))
	}
	must(
		"register output flag completion",
		mutatedCmd.RegisterFlagCompletionFunc(
			"output",
			cobra.FixedCompletions(oc, cobra.ShellCompDirectiveNoFileComp),
		),
	)

	b, ok := debug.ReadBuildInfo()
	if ok {
		mutatedCmd.Version = b.Main.Version
	}
}

func must(op string, err error) {
	if err != nil {
		klog.Fatalf("cannot %s: %s", op, err)
	}
}

func mutated(_ *cobra.Command, _ []string) {
	dc, err := cflags.ToDiscoveryClient()
	must("get discovery client", err)

	// namespace may come from kubeconfig, not just cli flags
	// this is normally hidden under ResourceBuilderFlags.ToBuilder
	// but that prevents further builder config
	ns, _, err := cflags.ToRawKubeConfigLoader().Namespace()
	must("read config", err)

	opt, ok := printerOptions[*output]
	if !ok {
		must("set up printer", fmt.Errorf("unrecognized printer: %s", *output))
	}
	p, err := opt.get()
	must("set up printer", err)
	defer p.Flush()

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

			all := true
			if *rflags.LabelSelector != "" {
				all = false
			}
			// XXX QPS doesn't seem to work across builders?
			v := p.ConfigureBuilder(resource.NewBuilder(cflags), gvk).
				SelectAllParam(all).
				NamespaceParam(ns).
				DefaultNamespace().
				AllNamespaces(*rflags.AllNamespaces).
				LabelSelectorParam(*rflags.LabelSelector).
				RequestChunksOf(512).
				// builder uses schema.Parse{Resource,Kind}Arg
				// resource.version.group: pod.v1. works but pod.v1 does not
				// not gvr.String()
				ResourceTypes(fmt.Sprintf("%s.%s.%s", gvr.Resource, gvr.Version, gvr.Group)).
				Flatten().
				Do()
			err = resource.NewFilteredVisitor(v, metadata.HasManuallyManagedFields).
				Visit(func(i *resource.Info, e error) error {
					if e != nil {
						return e
					}
					return p.PrintObject(i.Object, gvk)
				})
			if err != nil {
				klog.Warningf("cannot list %s %s: %s", rlist.GroupVersion, gvr.Resource, err)
			}
		}
	}
}
