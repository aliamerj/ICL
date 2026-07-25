package parser

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
)

func (p *Parser) parsePrimary() Expression {
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
	case lexer.TRUE:
		tok := p.advance()
		return &BoolLiteral{
			Value: true,
			Rng:   rangeOf(tok),
		}
	case lexer.FALSE:
		tok := p.advance()
		return &BoolLiteral{
			Value: false,
			Rng:   rangeOf(tok),
		}
	case lexer.IDENTIFIER:
		tok := p.advance()
		return &Identifier{
			Name: tok.Lexeme,
			Rng:  rangeOf(tok),
		}
	case lexer.LEFT_PAREN:
		p.advance()
		expr := p.parseExpression()
		if expr == nil {
			return nil
		}
		if _, ok := p.expect(lexer.RIGHT_PAREN); !ok {
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

func (p *Parser) parseExpression() Expression {
	return p.parseComparison()
}

func (p *Parser) parseComparison() Expression {
	expr := p.parseTerm()
	if expr == nil {
		return nil
	}

	for {
		switch p.cur().Type {
		case lexer.GREATER, lexer.GREATER_EQUAL, lexer.LESS, lexer.LESS_EQUAL, lexer.EQUAL_EQUAL, lexer.BANG_EQUAL:
			opTok := p.advance()
			right := p.parseTerm()
			if right == nil {
				return nil
			}
			expr = &BinaryExpr{
				Left:     expr,
				Operator: opTok.Lexeme,
				Right:    right,
				Rng:      Range{Start: expr.Range().Start, End: right.Range().End},
			}
		default:
			return expr
		}
	}
}

func (p *Parser) parseTerm() Expression {
	expr := p.parseFactor()
	if expr == nil {
		return nil
	}

	for {
		switch p.cur().Type {
		case lexer.PLUS, lexer.MINUS:
			opTok := p.advance()
			right := p.parseFactor()
			if right == nil {
				return nil
			}
			expr = &BinaryExpr{
				Left:     expr,
				Operator: opTok.Lexeme,
				Right:    right,
				Rng:      Range{Start: expr.Range().Start, End: right.Range().End},
			}
		default:
			return expr
		}
	}
}

func (p *Parser) parseFactor() Expression {
	expr := p.parsePrimary()
	if expr == nil {
		return nil
	}

	for {
		switch p.cur().Type {
		case lexer.STAR, lexer.SLASH:
			opTok := p.advance()
			right := p.parsePrimary()
			if right == nil {
				return nil
			}
			expr = &BinaryExpr{
				Left:     expr,
				Operator: opTok.Lexeme,
				Right:    right,
				Rng:      Range{Start: expr.Range().Start, End: right.Range().End},
			}
		default:
			return expr
		}
	}
}
