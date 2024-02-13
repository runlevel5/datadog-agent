package parser

type TypeNode struct {
	Doc    string
	Name   string
	Fields []FieldNode
}

type FieldNode struct {
	Doc        string
	FilterTags []string
	IsEmbed    bool
	Name       string
	Type       string

	SeclMappings []SeclMapping
}

type SeclMapping struct {
	Name    string
	Options DefinitionOptions
}

type DefinitionOptions = map[string]string
