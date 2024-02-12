package parser

type TypeNode struct {
	Name   string
	Fields []FieldNode
}

type FieldNode struct {
	FilterTags []string
	Name       string
	Type       string // TODO

	SECLName string
	Options  DefinitionOptions
}

type DefinitionOptions = map[string]string
