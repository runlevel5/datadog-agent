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

	fmt.Fprintf(os.Stdout, `// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build unix

//go:generate go run github.com/DataDog/datadog-agent/pkg/security/secl/compiler/generators/accessors -tags unix -types-file model.go -output accessors_unix.go -field-handlers field_handlers_unix.go -doc ../../../../docs/cloud-workload-security/secl.json -field-accessors-output field_accessors_unix.go

// Package model holds model related files
package model

import (
	"time"

	"github.com/DataDog/datadog-agent/pkg/security/secl/compiler/eval"
)
`)

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
					if k == "handler" || k == "opts" || k == "check" {
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
