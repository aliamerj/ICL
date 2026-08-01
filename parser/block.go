package parser

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/tokens"
)

// parseBlock is the one grammar function for every "KEYWORD label [as name] { body }"
// construct in the language.
func (p *parser) parseBlock(keyword tokens.Type) *Block {
	kwTok := p.advance()

	labelTok, ok := p.expect(tokens.IDENTIFIER)
	if !ok {
		return nil
	}

	var name *Identifier
	if p.cur().Type == tokens.AS {
		p.advance()
		nameTok, ok := p.expect(tokens.IDENTIFIER)
		if !ok {
			return nil
		}
		name = &Identifier{Name: nameTok.Lexeme, Rng: rangeOf(nameTok)}
	} else if keyword == tokens.RESOURCE || keyword == tokens.LOOKUP {
		// controls whether a missing `as name`
		p.reporter.ErrorAtOffsetWithCode(
			p.cur().Offset,
			diagnostics.MISSING_REQUIRED_NAME,
			fmt.Sprintf("%s %q must have a name", keyword, labelTok.Lexeme),
			fmt.Sprintf("add `as <name>`, e.g. `%s %s as my_%s`", keyword, labelTok.Lexeme, keyword),
		)
		return nil
	}

	if _, ok := p.expect(tokens.LEFT_BRACE); !ok {
		return nil
	}

	body := p.parseBody()
	if body == nil {
		return nil
	}

	endTok, ok := p.expect(tokens.RIGHT_BRACE)
	if !ok {
		return nil
	}

	return &Block{
		Keyword: keyword,
		Name:    name,
		Labels:  []*Identifier{{Name: labelTok.Lexeme, Rng: rangeOf(labelTok)}},
		Body:    body,
		Rng:     spanOf(kwTok, endTok),
	}
}

func (p *parser) parseBody() *Body {
	body := &Body{}

	for p.cur().Type != tokens.RIGHT_BRACE && p.cur().Type != tokens.EOF {
		attr := p.parseAttribute()
		if attr == nil {
			return nil
		}
		body.Statements = append(body.Statements, attr)
	}
	return body
}

func (p *parser) parseAttribute() *Attribute {
	keyTok, ok := p.expectAttributeKey()
	if !ok {
		return nil
	}
	if _, ok := p.expect(tokens.EQUAL); !ok {
		return nil
	}

	value := p.parseExpression()
	if value == nil {
		return nil
	}

	return &Attribute{
		Name:  &Identifier{Name: keyTok.Lexeme, Rng: rangeOf(keyTok)},
		Value: value,
		Rng:   rangePos{Start: rangeOf(keyTok).Start, End: value.Range().End},
	}
}

func (p *parser) expectAttributeKey() (lexer.Token, bool) {
	tok := p.cur()
	if tok.Type == tokens.IDENTIFIER || tok.Type == tokens.PROVIDER || tok.Type == tokens.RESOURCE {
		return p.advance(), true
	}
	p.reporter.ErrorAtOffsetWithCode(
		tok.Offset,
		diagnostics.UNEXPECTED_TOKEN,
		fmt.Sprintf("expected an attribute name, found %q", tok.Lexeme),
		"",
	)
	return lexer.Token{}, false
}
