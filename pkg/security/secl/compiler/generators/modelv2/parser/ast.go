package parser

type TypeNode struct {
	Doc    string
	Name   string
	Fields []FieldNode
}

type FieldNode struct {
	Doc        string
	FilterTags []string
	Name       string
	Type       string // TODO

	SeclMappings []SeclMapping
}

type SeclMapping struct {
	Name    string
	Options DefinitionOptions
}

type DefinitionOptions = map[string]string
