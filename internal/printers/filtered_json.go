package printers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml/lexer"
	yamlprinter "github.com/goccy/go-yaml/printer"
	"github.com/goccy/go-yaml/token"
	"github.com/mattn/go-isatty"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	indent = "    "
)

var (
	prefix = strings.Repeat(indent, 2)
)

type FilteredJSONPrinter struct {
	filteredPrinter
	trailer string
	first   bool
}

func NewFilteredJSONPrinter() (*FilteredJSONPrinter, error) {
	wrapper := map[string]any{
		"apiVersion": "v1",
		"kind":       "List",
		"items":      []any{},
	}
	b, err := json.MarshalIndent(wrapper, "", indent)
	if err != nil {
		return nil, err
	}
	tokens := lexer.Tokenize(string(b))
	var endStr, end int
	for i, t := range tokens {
		if t.Type == token.SequenceStartType {
			endStr = t.Position.Offset
			end = i + 1
		}
	}

	var trailer string
	if isatty.IsTerminal(os.Stdout.Fd()) {
		pr := yamlprinter.Printer{}
		pr.PrintErrorToken(tokens[0], true) // hack to set default colors
		pr.LineNumber = false               // altered by PrintErrorToken
		fmt.Print(pr.PrintTokens(tokens[:end]))
		trailer = indent + pr.PrintTokens(tokens[end:])
	} else {
		fmt.Print(string(b)[:endStr])
		trailer = indent + string(b)[endStr:]
	}
	return &FilteredJSONPrinter{trailer: trailer, first: true}, nil
}

func (p *FilteredJSONPrinter) PrintObject(r runtime.Object, gvk schema.GroupVersionKind) error {
	o, err := p.getFilteredObject(r, gvk)
	if err != nil {
		return fmt.Errorf("cannot get filtered object: %s", err)
	}

	b, err := json.MarshalIndent(o.Object, prefix, indent)
	if err != nil {
		return fmt.Errorf("cannot marshal JSON: %s", err)
	}

	if !p.first {
		fmt.Print(",")
	}
	fmt.Print("\n" + prefix)
	// TODO wrap it with a list instead?
	if isatty.IsTerminal(os.Stdout.Fd()) {
		tokens := lexer.Tokenize(string(b))
		pr := yamlprinter.Printer{}
		pr.PrintErrorToken(tokens[0], true) // hack to set default colors
		pr.LineNumber = false               // altered by PrintErrorToken
		fmt.Print(pr.PrintTokens(tokens))
	} else {
		fmt.Print(string(b))
	}
	p.first = false
	return nil
}

func (p *FilteredJSONPrinter) Flush() error {
	fmt.Println()
	fmt.Println(p.trailer)
	return nil
}
