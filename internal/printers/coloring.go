package printers

import (
	yamlprinter "github.com/goccy/go-yaml/printer"
	"github.com/goccy/go-yaml/token"
)

var (
	coloringYAMLPrinter = yamlprinter.Printer{}
)

func init() {
	tk := token.Space(&token.Position{})
	coloringYAMLPrinter.PrintErrorToken(tk, true) // hack to set default colors
	coloringYAMLPrinter.LineNumber = false        // altered by PrintErrorToken
}
