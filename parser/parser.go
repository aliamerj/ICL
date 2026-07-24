package parser

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
)

type Parser struct {
	tokens   []lexer.Token
	pos      int
	reporter *diagnostics.Reporter
}

func New(tokens []lexer.Token, reporter *diagnostics.Reporter) *Parser {
	return &Parser{tokens: tokens, reporter: reporter}
}

func (p *Parser) ParseProgram() *Program {
	prog := &Program{}
	startTok := p.cur()

	for p.cur().Type != lexer.EOF {
		switch p.cur().Type {
		case lexer.PROVIDER:
			if block := p.parseProviderBlock(); block != nil {
				prog.Statements = append(prog.Statements, block)
			} else {
				p.synchronize()
			}
		default:
			p.reporter.ErrorAtOffsetWithCode(
				p.cur().Offset,
				diagnostics.UNEXPECTED_TOKEN,
				fmt.Sprintf("unexpected token %q at top level", p.cur().Lexeme),
				"expected a block keyword like `provider`",
			)
			p.synchronize()
		}
	}

	prog.Rng = spanOf(startTok, p.cur()) // p.cur() is EOF here — gives you the full file span
	return prog
}

func (p *Parser) synchronize() {
	p.advance()

	for p.cur().Type != lexer.EOF {
		if p.cur().Type == lexer.RIGHT_BRACE {
			p.advance()
			return
		}
		switch p.cur().Type {
		case lexer.PROVIDER:
			return
		}
		p.advance()
	}
}

func (p *Parser) parseProviderBlock() *Block {
	kwTok := p.advance() // consume PROVIDER

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

// parseExpression parses a single value on the right-hand side of `=`.
// Grows over time: literals now, identifiers/references/arrays/objects later.
func (p *Parser) parseExpression() Expression {
	switch p.cur().Type {
	case lexer.STRING:
		tok := p.advance()
		return &StringLiteral{
			Value: tok.Literal.(string),
			Rng:   rangeOf(tok),
		}
	case lexer.NUMBER_INT:
		tok := p.advance()
		return &IntLiteral{
			Value: tok.Literal.(int64),
			Rng:   rangeOf(tok),
		}
	case lexer.NUMBER_FLOAT:
		tok := p.advance()
		return &FloatLiteral{
			Value: tok.Literal.(float64),
			Rng:   rangeOf(tok),
		}
	default:
		p.reporter.ErrorAtOffsetWithCode(
			p.cur().Offset,
			diagnostics.UNEXPECTED_TOKEN,
			fmt.Sprintf("expected a value, found %q", p.cur().Lexeme),
			"expected a string, number, or other expression here",
		)
		return nil
	}
}

func (p *Parser) expect(tt lexer.TokenType) (lexer.Token, bool) {
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

func (p *Parser) advance() lexer.Token {
	t := p.cur()
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return t
}

func (p *Parser) cur() lexer.Token {
	return p.tokens[p.pos]
}

func rangeOf(tok lexer.Token) Range {
	return Range{
		Start: Pos{Line: tok.Line, Offset: tok.Offset},
		End:   Pos{Line: tok.Line, Offset: tok.Offset + len(tok.Lexeme)},
	}
}

func spanOf(start, end lexer.Token) Range {
	return Range{
		Start: Pos{Line: start.Line, Offset: start.Offset},
		End:   Pos{Line: end.Line, Offset: end.Offset + len(end.Lexeme)},
	}
}
