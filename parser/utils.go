package parser

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/tokens"
)

func (p *parser) expect(tt tokens.Type) (lexer.Token, bool) {
	if p.cur().Type != tt {
		p.reporter.ErrorAtOffsetWithCode(
			p.cur().Offset,
			diagnostics.UNEXPECTED_TOKEN,
			fmt.Sprintf("expected %v, found %q", tt, p.cur().Lexeme),
			fmt.Sprintf("add a %v here", tt),
		)
		return lexer.Token{}, false
	}
	return p.advance(), true
}

func (p *parser) advance() lexer.Token {
	t := p.cur()
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return t
}

func (p *parser) cur() lexer.Token {
	return p.tokens[p.pos]
}

func rangeOf(tok lexer.Token) rangePos {
	return rangePos{
		Start: pos{Line: tok.Line, Offset: tok.Offset},
		End:   pos{Line: tok.Line, Offset: tok.Offset + len(tok.Lexeme)},
	}
}

func spanOf(start, end lexer.Token) rangePos {
	return rangePos{
		Start: pos{Line: start.Line, Offset: start.Offset},
		End:   pos{Line: end.Line, Offset: end.Offset + len(end.Lexeme)},
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
