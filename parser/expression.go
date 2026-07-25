package parser

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/tokens"
)

func (p *parser) parsePrimary() Expression {
	switch p.cur().Type {
	case tokens.STRING:
		tok := p.advance()
		return &StringLiteral{
			Value: tok.Literal.(string),
			Rng:   rangeOf(tok),
		}

	case tokens.NUMBER_INT:
		tok := p.advance()
		return &IntLiteral{
			Value: tok.Literal.(int64),
			Rng:   rangeOf(tok),
		}

	case tokens.NUMBER_FLOAT:
		tok := p.advance()
		return &FloatLiteral{
			Value: tok.Literal.(float64),
			Rng:   rangeOf(tok),
		}
	case tokens.TRUE:
		tok := p.advance()
		return &BoolLiteral{
			Value: true,
			Rng:   rangeOf(tok),
		}
	case tokens.FALSE:
		tok := p.advance()
		return &BoolLiteral{
			Value: false,
			Rng:   rangeOf(tok),
		}
	case tokens.IDENTIFIER:
		tok := p.advance()
		return &Identifier{
			Name: tok.Lexeme,
			Rng:  rangeOf(tok),
		}
	case tokens.LEFT_PAREN:
		p.advance()
		expr := p.parseExpression()
		if expr == nil {
			return nil
		}
		if _, ok := p.expect(tokens.RIGHT_PAREN); !ok {
			return nil
		}
		return expr
	default:
		p.reporter.ErrorAtOffsetWithCode(
			p.cur().Offset,
			diagnostics.UNEXPECTED_TOKEN,
			fmt.Sprintf("expected an expression, found %q", p.cur().Lexeme),
			"expected a string, number, identifier, or parenthesized expression here",
		)

		return nil
	}
}

func (p *parser) parseExpression() Expression {
	return p.parseComparison()
}

func (p *parser) parseComparison() Expression {
	expr := p.parseTerm()
	if expr == nil {
		return nil
	}

	for {
		switch p.cur().Type {
		case tokens.GREATER,
			tokens.GREATER_EQUAL,
			tokens.LESS,
			tokens.LESS_EQUAL,
			tokens.EQUAL_EQUAL,
			tokens.BANG_EQUAL:
			opTok := p.advance()
			right := p.parseTerm()
			if right == nil {
				return nil
			}
			expr = &BinaryExpr{
				Left:     expr,
				Operator: opTok.Lexeme,
				Right:    right,
				Rng:      rangePos{Start: expr.Range().Start, End: right.Range().End},
			}
		default:
			return expr
		}
	}
}

func (p *parser) parseTerm() Expression {
	expr := p.parseFactor()
	if expr == nil {
		return nil
	}

	for {
		switch p.cur().Type {
		case tokens.PLUS, tokens.MINUS:
			opTok := p.advance()
			right := p.parseFactor()
			if right == nil {
				return nil
			}
			expr = &BinaryExpr{
				Left:     expr,
				Operator: opTok.Lexeme,
				Right:    right,
				Rng:      rangePos{Start: expr.Range().Start, End: right.Range().End},
			}
		default:
			return expr
		}
	}
}

func (p *parser) parseFactor() Expression {
	expr := p.parsePrimary()
	if expr == nil {
		return nil
	}

	for {
		switch p.cur().Type {
		case tokens.STAR, tokens.SLASH:
			opTok := p.advance()
			right := p.parsePrimary()
			if right == nil {
				return nil
			}
			expr = &BinaryExpr{
				Left:     expr,
				Operator: opTok.Lexeme,
				Right:    right,
				Rng:      rangePos{Start: expr.Range().Start, End: right.Range().End},
			}
		default:
			return expr
		}
	}
}
