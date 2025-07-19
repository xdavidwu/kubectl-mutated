package printers

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
)

type Printer interface {
	ConfigureBuilder(r *resource.Builder) *resource.Builder
	PrintObject(r runtime.Object, gvk schema.GroupVersionKind) error
	Flush() error
}

var _ Printer = &TablePrinter{}
var _ Printer = &HighlightedYAMLPrinter{}
var _ Printer = &FilteredYAMLPrinter{}
var _ Printer = &FilteredJSONPrinter{}
