package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/security/secl/compiler/generators/modelv2/parser"
)

func main() {
	content, err := os.ReadFile("./pkg/security/secl/compiler/generators/modelv2/example.prego")
	if err != nil {
		panic(err)
	}

	lexer := parser.NewTokenizer(string(content))
	pars := parser.NewParser(lexer)

	for !pars.IsAtEof() {
		tn, err := pars.ParseTypeNode()
		if err != nil {
			panic(err)
		}
		emitOldModel(tn)
	}
}

func emitOldModel(tn parser.TypeNode) {
	w := os.Stdout

	if tn.Doc != "" {
		fmt.Fprintf(w, "// %s\n", tn.Doc)
	}
	fmt.Fprintf(w, "type %s struct {\n", tn.Name)

	for _, field := range tn.Fields {
		if field.IsEmbed {
			fmt.Fprintf(w, "\t%s", field.Type)
		} else {
			fmt.Fprintf(w, "\t%s %s", field.Name, field.Type)
		}

		if len(field.SeclMappings) == 0 {
			fmt.Fprintf(w, " `field:\"-\"`\n")
		} else {
			var fieldContent []string
			for _, mapping := range field.SeclMappings {
				field := mapping.Name
				for k, v := range mapping.Options {
					if k == "handler" || k == "opts" {
						field += fmt.Sprintf(",%s:%s", k, v)
					}
				}
				fieldContent = append(fieldContent, field)
			}

			fmt.Fprintf(w, " `field:\"%s\"`\n", strings.Join(fieldContent, ";"))
		}
	}

	fmt.Fprintf(w, "}\n\n")
}
