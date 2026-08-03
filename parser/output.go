package parser

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/tokens"
)

func (p *parser) parseOutputDecl() *OutputDecl {
	kwTok := p.advance() // consume OUTPUT

	nameTok, ok := p.expect(tokens.IDENTIFIER)
	if !ok {
		return nil
	}
	decl := &OutputDecl{
		Name: &Identifier{
			Name: nameTok.Lexeme,
			Rng:  rangeOf(nameTok),
		},
	}

	switch p.cur().Type {
	case tokens.EQUAL:
		p.advance()
		val := p.parseExpression()
		if val == nil {
			return nil
		}
		decl.Value = val
		decl.Rng = rangePos{
			Start: rangeOf(kwTok).Start,
			End:   val.Range().End,
		}
	case tokens.LEFT_BRACE:
		p.advance()
		body := p.parseBody()
		if body == nil {
			return nil
		}
		endTok, ok := p.expect(tokens.RIGHT_BRACE)
		if !ok {
			return nil
		}
		decl.Body = body
		decl.Rng = spanOf(kwTok, endTok)
	default:
		p.reporter.ErrorAtOffsetWithCode(p.cur().Offset, diagnostics.MISSING_OUTPUT_VALUE,
			fmt.Sprintf("output %q must have a value", nameTok.Lexeme),
			"e.g. `output "+nameTok.Lexeme+" = some_resource.id` or a body with `value = ...`")
		return nil
	}
	return decl
}
