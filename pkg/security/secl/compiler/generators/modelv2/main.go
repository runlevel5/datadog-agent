package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/security/secl/compiler/generators/modelv2/parser"
)

//go:embed header.tpl
var header string

func main() {
	content, err := os.ReadFile("./pkg/security/secl/compiler/generators/modelv2/example.prego")
	if err != nil {
		panic(err)
	}

	lexer := parser.NewTokenizer(string(content))
	pars := parser.NewParser(lexer)

	out, err := os.Create("./pkg/security/secl/model/model_unix.go")
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(out, header)

	for !pars.IsAtEof() {
		tn, err := pars.ParseTypeNode()
		if err != nil {
			panic(err)
		}
		emitOldModel(out, tn)
	}

	if err := out.Close(); err != nil {
		panic(err)
	}

	cmd := exec.Command("gofmt", "-s", "-w", out.Name())
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func emitOldModel(w io.Writer, tn parser.TypeNode) {
	if tn.Doc != "" {
		fmt.Fprintf(w, "// %s\n", tn.Doc)
	}
	if tn.Name == "Event" {
		fmt.Fprintf(w, "// genaccessors\n")
	}
	fmt.Fprintf(w, "type %s struct {\n", tn.Name)

	for _, field := range tn.Fields {
		if field.IsEmbed {
			fmt.Fprintf(w, "\t%s", field.Type)
		} else {
			fmt.Fprintf(w, "\t%s %s", field.Name, field.Type)
		}

		if len(field.SeclMappings) == 0 {
			if !field.IsEmbed {
				fmt.Fprintf(w, " `field:\"-\"`\n")
			} else {
				fmt.Fprintf(w, "\n")
			}
		} else {
			var fieldContent []string
			var comment string
			if field.EventType != "" && field.Name != "Async" {
				comment = field.Doc
			}

			for _, mapping := range field.SeclMappings {
				fieldTag := mapping.Name
				for k, v := range mapping.Options {
					if k == "handler" || k == "opts" || k == "check" || k == "weight" {
						fieldTag += fmt.Sprintf(",%s:%s", k, v)
					}
				}
				fieldContent = append(fieldContent, fieldTag)

				doc := mapping.Options["doc"]
				if doc == "" {
					doc = field.Doc
				}

				if field.EventType == "" || field.Name == "Async" {
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
