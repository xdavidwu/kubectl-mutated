package printers

import (
	"bytes"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/lexer"
	yamlprinter "github.com/goccy/go-yaml/printer"
	"github.com/mattn/go-isatty"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"

	"github.com/xdavidwu/kubectl-mutated/internal/metadata"
)

type FilteredYAMLPrinter struct {
}

func (FilteredYAMLPrinter) ConfigureBuilder(r *resource.Builder) *resource.Builder {
	// TODO use scheme if possible, to utilize protobuf
	return r.Unstructured()
}

func (p *FilteredYAMLPrinter) PrintObject(r runtime.Object, gvk schema.GroupVersionKind) error {
	o, ok := r.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("unexpected type")
	}

	c := o.DeepCopy()
	c.SetManagedFields(nil)

	s := &fieldpath.Set{}
	for _, mf := range metadata.FindSoleManualManagers(o.GetManagedFields()) {
		ms := &fieldpath.Set{}
		err := ms.FromJSON(bytes.NewBuffer(mf.FieldsV1.Raw))
		if err != nil {
			return err
		}
		s = s.Union(ms)
	}

	f, err := Filter(c, s)
	if err != nil {
		return fmt.Errorf("cannot filter resource: %s", err)
	}

	b, err := yaml.Marshal(f.Object)
	if err != nil {
		return fmt.Errorf("cannot marshal YAML: %s", err)
	}

	// TODO wrap it with a list instead?
	fmt.Println("---")
	if isatty.IsTerminal(os.Stdout.Fd()) {
		tokens := lexer.Tokenize(string(b))
		pr := yamlprinter.Printer{}
		pr.PrintErrorToken(tokens[0], true) // hack to set default colors
		pr.LineNumber = false               // altered by PrintErrorToken
		fmt.Println(pr.PrintTokens(tokens))
	} else {
		fmt.Print(string(b))
	}
	return nil
}

func (p *FilteredYAMLPrinter) Flush() error {
	return nil
}
