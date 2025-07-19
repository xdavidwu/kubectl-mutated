package printers

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/lexer"
	yamlprinter "github.com/goccy/go-yaml/printer"
	"github.com/mattn/go-isatty"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/xdavidwu/kubectl-mutated/internal/metadata"
)

type FilteredYAMLPrinter struct {
	unstructuredPrinter
}

func (p *FilteredYAMLPrinter) PrintObject(r runtime.Object, gvk schema.GroupVersionKind) error {
	o, err := p.toUnstructured(r)
	if err != nil {
		return fmt.Errorf("cannot convert to unstructured: %s", err)
	}

	c := o.DeepCopy()
	c.SetManagedFields(nil)

	s, err := metadata.SolelyManuallyManagedSet(o.GetManagedFields())
	if err != nil {
		return fmt.Errorf("cannot conclude field set: %s", err)
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
