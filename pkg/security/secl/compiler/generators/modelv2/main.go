package main

import (
	"fmt"
	"os"

	"github.com/DataDog/datadog-agent/pkg/security/secl/compiler/generators/modelv2/parser"
)

func main() {
	content, err := os.ReadFile("./pkg/security/secl/compiler/generators/modelv2/example.prego")
	if err != nil {
		panic(err)
	}

	lexer := parser.NewTokenizer(string(content))

	for {
		tok, err := lexer.NextToken()
		if err != nil {
			panic(err)
		}
		fmt.Println(tok)

		if tok.Kind == parser.EOF {
			break
		}
	}
}
