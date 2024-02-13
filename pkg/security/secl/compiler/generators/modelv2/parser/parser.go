package parser

import (
	"fmt"
	"strings"
)

type Parser struct {
	lexer         *Tokenizer
	peekCache     Token
	peekCacheFull bool
}

func NewParser(lexer *Tokenizer) *Parser {
	return &Parser{
		lexer: lexer,
	}
}

func (p *Parser) IsAtEof() bool {
	tok, err := p.peekToken()
	if err != nil {
		return false
	}
	return tok.Kind == EOF
}

func (p *Parser) nextToken() (Token, error) {
	if p.peekCacheFull {
		p.peekCacheFull = false
		return p.peekCache, nil
	}

	return p.lexer.NextToken()
}

func (p *Parser) peekToken() (Token, error) {
	if !p.peekCacheFull {
		tok, err := p.lexer.NextToken()
		if err != nil {
			return tok, err
		}
		p.peekCache = tok
		p.peekCacheFull = true
	}
	return p.peekCache, nil
}

func (p *Parser) isNextTokenA(against TokenKind) bool {
	tok, err := p.peekToken()
	if err != nil {
		return false
	}

	return tok.Kind == against
}

func unexpectedError(got TokenKind, expected ...TokenKind) error {
	return fmt.Errorf("unexpected %s, expected %s", got, expected)
}

func (p *Parser) acceptToken(expected ...TokenKind) (Token, error) {
	tok, err := p.nextToken()
	if err != nil {
		return Token{}, err
	}

	for _, e := range expected {
		if tok.Kind == e {
			return tok, nil
		}
	}
	return Token{}, unexpectedError(tok.Kind, expected...)
}

func (p *Parser) advanceIf(against ...TokenKind) (bool, TokenKind, error) {
	tok, err := p.peekToken()
	if err != nil {
		return false, Undefined, err
	}

	for _, a := range against {
		if tok.Kind == a {
			p.nextToken()
			return true, a, nil
		}
	}
	return false, Undefined, nil
}

func (p *Parser) ParseTypeNode() (TypeNode, error) {
	doc, err := p.parseDocComment()
	if err != nil {
		return TypeNode{}, err
	}

	_, err = p.acceptToken(TypeKeyword)
	if err != nil {
		return TypeNode{}, err
	}

	id, err := p.acceptToken(Identifier)
	if err != nil {
		return TypeNode{}, err
	}

	_, err = p.acceptToken(StructKeyword)
	if err != nil {
		return TypeNode{}, err
	}

	_, err = p.acceptToken(LeftCurlyBracket)
	if err != nil {
		return TypeNode{}, err
	}

	var fields []FieldNode

	for !p.isNextTokenA(RightCurlyBracket) {
		field, err := p.parseFieldNode()
		if err != nil {
			return TypeNode{}, err
		}
		fields = append(fields, field)
	}

	_, err = p.acceptToken(RightCurlyBracket)
	if err != nil {
		return TypeNode{}, err
	}

	return TypeNode{
		Doc:    doc,
		Name:   id.Content,
		Fields: fields,
	}, nil
}

func (p *Parser) parseFieldNode() (FieldNode, error) {
	doc, err := p.parseDocComment()
	if err != nil {
		return FieldNode{}, err
	}

	filterTags, err := p.parseFilterTags()
	if err != nil {
		return FieldNode{}, err
	}

	// handle embed struct
	isEmbed, _, err := p.advanceIf(EmbedKeyword)
	if err != nil {
		return FieldNode{}, err
	}

	if isEmbed {
		typeName, typeBase, err := p.parseGoType()
		if err != nil {
			return FieldNode{}, err
		}

		return FieldNode{
			Doc:        doc,
			FilterTags: filterTags,
			IsEmbed:    true,
			Name:       typeBase,
			Type:       typeName,
		}, nil
	}

	id, err := p.acceptToken(Identifier)
	if err != nil {
		return FieldNode{}, err
	}

	typeName, _, err := p.parseGoType()
	if err != nil {
		return FieldNode{}, err
	}

	var eventType string
	isDoubleArrow, _, err := p.advanceIf(DoubleArrow)
	if err != nil {
		return FieldNode{}, err
	}
	if isDoubleArrow {
		eventType, err = p.parseEventType()
		if err != nil {
			return FieldNode{}, err
		}
	}

	var mappings []SeclMapping
	for p.isNextTokenA(Arrow) {
		_, err := p.acceptToken(Arrow)
		if err != nil {
			return FieldNode{}, err
		}

		var mapping SeclMapping
		mapping.Name, err = p.parseSECLName()
		if err != nil {
			return FieldNode{}, err
		}

		if p.isNextTokenA(LeftCurlyBracket) {
			mapping.Options, err = p.parseOptions()
			if err != nil {
				return FieldNode{}, err
			}
		}

		mappings = append(mappings, mapping)
	}

	return FieldNode{
		Doc:        doc,
		FilterTags: filterTags,
		Name:       id.Content,
		Type:       typeName,

		EventType:    eventType,
		SeclMappings: mappings,
	}, nil
}

func (p *Parser) parseDocComment() (string, error) {
	if p.isNextTokenA(DocComment) {
		doc, err := p.acceptToken(DocComment)
		if err != nil {
			return "", err
		}
		return doc.Content, nil
	}
	return "", nil
}

func (p *Parser) parseFilterTags() ([]string, error) {
	var tags []string
	for {
		isTag, _, err := p.advanceIf(CommercialAt)
		if err != nil {
			return nil, err
		}

		if !isTag {
			break
		}

		tag, err := p.acceptToken(Identifier)
		if err != nil {
			return nil, err
		}

		tags = append(tags, tag.Content)
	}
	return tags, nil
}

func (p *Parser) parseSECLName() (string, error) {
	if p.isNextTokenA(StringLiteral) {
		lit, err := p.acceptToken(StringLiteral)
		return lit.Content, err
	}

	return p.parseDottedIdentifier()
}

func (p *Parser) parseDottedIdentifier() (string, error) {
	var parts []string

	for {
		tok, err := p.acceptToken(Identifier)
		if err != nil {
			return "", err
		}
		parts = append(parts, tok.Content)

		isDotNext, _, err := p.advanceIf(Dot)
		if err != nil {
			return "", err
		}

		if !isDotNext {
			break
		}
	}

	return strings.Join(parts, "."), nil
}

func (p *Parser) parseOptions() (map[string]string, error) {
	_, err := p.acceptToken(LeftCurlyBracket)
	if err != nil {
		return nil, err
	}

	options := make(map[string]string)

	for !p.isNextTokenA(RightCurlyBracket) {
		field, err := p.acceptToken(Identifier)
		if err != nil {
			return nil, err
		}

		_, err = p.acceptToken(Colon)
		if err != nil {
			return nil, err
		}

		value, err := p.parseOptionValue()
		if err != nil {
			return nil, err
		}

		if _, ok := options[field.Content]; ok {
			return nil, fmt.Errorf("option `%s` already specified", field.Content)
		}
		options[field.Content] = value

		isCommaNext, _, err := p.advanceIf(Comma)
		if err != nil {
			return nil, err
		}

		// no comma after option, then we must exit
		if !isCommaNext {
			break
		}
	}

	_, err = p.acceptToken(RightCurlyBracket)
	if err != nil {
		return nil, err
	}

	return options, nil
}

func (p *Parser) parseOptionValue() (string, error) {
	nextTok, err := p.acceptToken(Identifier, NumberLiteral, StringLiteral, TrueKeyword, FalseKeyword)
	if err != nil {
		return "", err
	}

	return nextTok.Content, nil
}

func (p *Parser) parseGoType() (string, string, error) {
	isSub, kind, err := p.advanceIf(Star, LeftSquareBracket)
	if err != nil {
		return "", "", err
	}

	if isSub {
		switch kind {
		case Star:
			sub, base, err := p.parseGoType()
			return "*" + sub, base, err
		case LeftSquareBracket:
			prefix := "["
			if p.isNextTokenA(NumberLiteral) || p.isNextTokenA(Identifier) {
				count, err := p.acceptToken(NumberLiteral, Identifier)
				if err != nil {
					return "", "", err
				}
				prefix += count.Content
			}
			_, err := p.acceptToken(RightSquareBracket)
			if err != nil {
				return "", "", err
			}
			prefix += "]"
			sub, base, err := p.parseGoType()
			return prefix + sub, base, err
		default:
			panic("unreachable")
		}
	}

	id, err := p.parseDottedIdentifier()
	if err != nil {
		return "", "", err
	}
	return id, id, nil
}

func (p *Parser) parseEventType() (string, error) {
	isStar, _, err := p.advanceIf(Star)
	if err != nil {
		return "", err
	}
	if isStar {
		return "*", nil
	}

	return p.parseDottedIdentifier()
}
