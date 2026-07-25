package parser

import (
	"github.com/aliamerj/icl/tokens"
)

func (p *parser) parseProviderBlock() *Block {
	kwTok := p.advance()

	labelTok, ok := p.expect(tokens.IDENTIFIER)
	if !ok {
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
    Keyword: tokens.PROVIDER,
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
	keyTok, ok := p.expect(tokens.IDENTIFIER)
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
