package printers

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/lexer"
	"github.com/mattn/go-isatty"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type FilteredYAMLPrinter struct {
	filteredPrinter
}

func (p *FilteredYAMLPrinter) PrintObject(r runtime.Object, gvk schema.GroupVersionKind) error {
	o, err := p.getFilteredObject(r, gvk)
	if err != nil {
		return fmt.Errorf("cannot get filtered object: %s", err)
	}

	b, err := yaml.Marshal(o.Object)
	if err != nil {
		return fmt.Errorf("cannot marshal YAML: %s", err)
	}

	// TODO wrap it with a list instead?
	fmt.Println("---")
	if isatty.IsTerminal(os.Stdout.Fd()) {
		tokens := lexer.Tokenize(string(b))
		fmt.Println(coloringYAMLPrinter.PrintTokens(tokens))
	} else {
		fmt.Print(string(b))
	}
	return nil
}

func (p *FilteredYAMLPrinter) Flush() error {
	return nil
}
