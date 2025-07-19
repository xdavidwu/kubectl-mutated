package printers

import (
	"bytes"
	"fmt"
	"iter"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/parser"
	yamlprinter "github.com/goccy/go-yaml/printer"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"

	"github.com/xdavidwu/kubectl-mutated/internal/metadata"
)

type HighlightedYAMLPrinter struct {
}

func (HighlightedYAMLPrinter) ConfigureBuilder(r *resource.Builder) *resource.Builder {
	// TODO use scheme if possible, to utilize protobuf
	return r.Unstructured()
}

type highlighter struct{}

func (v *highlighter) Visit(n ast.Node) ast.Visitor {
	if t := n.GetToken(); t != nil {
		if t.Value != "" {
			t.Origin = "\x1b[1;3m" + t.Origin + "\x1b[22;23m"
		}
	}
	return v
}

func highlight(n ast.Node) {
	ast.Walk(&highlighter{}, n)
}

type UnexpectedTypeError struct {
	Expected ast.NodeType
	Seen     ast.NodeType
}

func (e UnexpectedTypeError) Error() string {
	return fmt.Sprintf("unexpected type %s, expecting %s", e.Seen, e.Expected)
}

// TODO complete with path element other than f:
func iterate(
	n ast.Node,
	ps iter.Seq[fieldpath.PathElement],
	fnkv func(kv *ast.MappingValueNode, p fieldpath.PathElement) error,
) error {
	for p := range ps {
		switch {
		case p.FieldName != nil:
			if n.Type() != ast.MappingType {
				return UnexpectedTypeError{Expected: ast.MappingType, Seen: n.Type()}
			}
			m := n.(*ast.MappingNode)

			for _, kv := range m.Values {
				if kv.Key.Type() != ast.StringType {
					continue
				}
				sn := kv.Key.(*ast.StringNode)
				if sn.Value == *p.FieldName {
					if err := fnkv(kv, p); err != nil {
						return err
					}
					break
				}
			}
		}
	}
	return nil
}

func traverse(n ast.Node, s *fieldpath.Set) error {
	if err := iterate(
		n,
		s.Members.All(),
		func(kv *ast.MappingValueNode, _ fieldpath.PathElement) error {
			highlight(kv)
			return nil
		},
	); err != nil {
		return err
	}

	return iterate(
		n,
		s.Children.All(),
		func(kv *ast.MappingValueNode, p fieldpath.PathElement) error {
			return traverse(kv.Value, s.Children.Descend(p))
		},
	)
}

func (p *HighlightedYAMLPrinter) PrintObject(r runtime.Object, gvk schema.GroupVersionKind) error {
	o, ok := r.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("unexpected type")
	}

	c := o.DeepCopy()
	c.SetManagedFields(nil)

	// a whole round-trip make all tokens there, including spaces
	b, err := yaml.Marshal(c.Object)
	if err != nil {
		return err
	}
	tokens := lexer.Tokenize(string(b))
	f, err := parser.Parse(tokens, 0)
	if err != nil {
		return err
	}
	t := f.Docs[0].Body

	for _, mf := range metadata.FindSoleManualManagers(o.GetManagedFields()) {
		s := &fieldpath.Set{}
		err := s.FromJSON(bytes.NewBuffer(mf.FieldsV1.Raw))
		if err != nil {
			return err
		}

		err = traverse(t, s.Leaves())
		if err != nil {
			klog.Warning("err", err)
		}
	}

	// TODO wrap it with a list instead?
	fmt.Println("---")
	pr := yamlprinter.Printer{}
	// FIXME k of first kv in map is broken?
	//pr.PrintErrorToken(tokens[0], true) // hack to set default colors
	//pr.LineNumber = false // altered by PrintErrorToken
	if _, err := fmt.Println(pr.PrintTokens(tokens)); err != nil {
		return err
	}
	return nil
}

func (p *HighlightedYAMLPrinter) Flush() error {
	return nil
}
