package parser

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/tokens"
)

func (p *parser) parsePrimary() Expression {
	switch p.cur().Type {
	case tokens.LEFT_BRACKET:
		return p.parseListLiteral()
	case tokens.LEFT_BRACE:
		return p.parseObjectLiteral()
	case tokens.STRING:
		tok := p.advance()
		value, ok := tokenLiteral[string](p.reporter, tok, "string")
		if !ok {
			return nil
		}
		return &StringLiteral{Value: value, Rng: rangeOf(tok)}

	case tokens.NUMBER_INT:
		tok := p.advance()
		value, ok := tokenLiteral[int64](p.reporter, tok, "integer")
		if !ok {
			return nil
		}
		return &IntLiteral{Value: value, Rng: rangeOf(tok)}

	case tokens.NUMBER_FLOAT:
		tok := p.advance()
		value, ok := tokenLiteral[float64](p.reporter, tok, "float")
		if !ok {
			return nil
		}
		return &FloatLiteral{Value: value, Rng: rangeOf(tok)}
	case tokens.TRUE:
		tok := p.advance()
		return &BoolLiteral{Value: true, Rng: rangeOf(tok)}
	case tokens.FALSE:
		tok := p.advance()
		return &BoolLiteral{Value: false, Rng: rangeOf(tok)}
	case tokens.IDENTIFIER:
		tok := p.advance()
		id := &Identifier{Name: tok.Lexeme, Rng: rangeOf(tok)}
		return p.parsePostfix(id)
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

func tokenLiteral[T any](reporter *diagnostics.Reporter, tok lexer.Token, kind string) (T, bool) {
	value, ok := tok.Literal.(T)
	if ok {
		return value, true
	}

	var zero T
	reporter.ErrorAtOffsetWithCode(
		tok.Offset,
		diagnostics.UNEXPECTED_TOKEN,
		fmt.Sprintf("malformed %s literal %q", kind, tok.Lexeme),
		"rebuild the scanner output or report this bug",
	)
	return zero, false
}

func (p *parser) parseListLiteral() Expression {

	startTok := p.advance() // consume '['
	var elements []Expression

	for p.cur().Type != tokens.RIGHT_BRACKET && p.cur().Type != tokens.EOF {
		elem := p.parseExpression()
		if elem == nil {
			return nil
		}
		elements = append(elements, elem)

		if p.cur().Type == tokens.COMMA {
			p.advance() // trailing comma before ']' is fine — loop condition catches it next iteration
			continue
		}
		break
	}

	endTok, ok := p.expect(tokens.RIGHT_BRACKET)
	if !ok {
		return nil
	}

	return &ListExpr{Elements: elements, Rng: spanOf(startTok, endTok)}
}

func (p *parser) parseObjectLiteral() Expression {
	startTok := p.advance() // consume '{'
	var fields []*Attribute

	for p.cur().Type != tokens.RIGHT_BRACE && p.cur().Type != tokens.EOF {
		attr := p.parseAttribute() // exact same function block bodies already use
		if attr == nil {
			return nil
		}
		fields = append(fields, attr)
	}

	endTok, ok := p.expect(tokens.RIGHT_BRACE)
	if !ok {
		return nil
	}

	return &ObjectExpr{Fields: fields, Rng: spanOf(startTok, endTok)}
}

func (p *parser) parsePostfix(base Expression) Expression {
	for p.cur().Type == tokens.DOT {
		p.advance()
		propTok, ok := p.expect(tokens.IDENTIFIER)
		if !ok {
			return nil
		}
		base = &MemberExpr{
			Object:   base,
			Property: propTok.Lexeme,
			Rng: rangePos{
				Start: base.Range().Start,
				End:   rangeOf(propTok).End,
			},
		}
	}
	return base
}
