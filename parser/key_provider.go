package parser

import "github.com/aliamerj/icl/lexer"

func (p *Parser) parseProviderBlock() *Block {
	kwTok := p.advance()

	labelTok, ok := p.expect(lexer.IDENTIFIER)
	if !ok {
		return nil
	}

	if _, ok := p.expect(lexer.LEFT_BRACE); !ok {
		return nil
	}

	body := p.parseBody()
	if body == nil {
		return nil
	}

	endTok, ok := p.expect(lexer.RIGHT_BRACE)
	if !ok {
		return nil
	}

	return &Block{
		Keyword: "provider",
		Labels:  []*Identifier{{Name: labelTok.Lexeme, Rng: rangeOf(labelTok)}},
		Body:    body,
		Rng:     spanOf(kwTok, endTok),
	}
}

func (p *Parser) parseBody() *Body {
	body := &Body{}

	for p.cur().Type != lexer.RIGHT_BRACE && p.cur().Type != lexer.EOF {
		attr := p.parseAttribute()
		if attr == nil {
			return nil
		}
		body.Statements = append(body.Statements, attr)
	}
	return body
}

func (p *Parser) parseAttribute() *Attribute {
	keyTok, ok := p.expect(lexer.IDENTIFIER)
	if !ok {
		return nil
	}
	if _, ok := p.expect(lexer.EQUAL); !ok {
		return nil
	}

	value := p.parseExpression()
	if value == nil {
		return nil
	}

	return &Attribute{
		Name:  &Identifier{Name: keyTok.Lexeme, Rng: rangeOf(keyTok)},
		Value: value,
		Rng:   Range{Start: rangeOf(keyTok).Start, End: value.Range().End},
	}
}
