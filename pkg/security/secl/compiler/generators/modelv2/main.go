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
			var comment string
			if field.EventType != "" {
				comment = field.Doc
			}

			for _, mapping := range field.SeclMappings {
				fieldTag := mapping.Name
				for k, v := range mapping.Options {
					if k == "handler" || k == "opts" {
						fieldTag += fmt.Sprintf(",%s:%s", k, v)
					}
				}
				fieldContent = append(fieldContent, fieldTag)

				doc := mapping.Options["doc"]
				if doc == "" {
					doc = field.Doc
				}

				if field.EventType == "" {
					comment += fmt.Sprintf("SECLDoc[%s] Definition:`%s`", mapping.Name, doc)
					if c := mapping.Options["constants"]; c != "" {
						comment += fmt.Sprintf(" Constants:`%s`", c)
					}
				}
			}

			fmt.Fprintf(w, " `field:\"%s\"", strings.Join(fieldContent, ";"))
			if field.EventType != "" {
				fmt.Fprintf(w, " event:\"%s\"", field.EventType)
			}
			if comment != "" {
				fmt.Fprintf(w, "` // %s\n", comment)
			} else {
				fmt.Fprintf(w, "`\n")
			}
		}
	}

	fmt.Fprintf(w, "}\n\n")
}
