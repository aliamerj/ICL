package parser

import "github.com/aliamerj/icl/tokens"

func (p *parser) parseVarDecl() *VarDecl {
	kwTok := p.advance() // consume VAR
	nameTok, ok := p.expect(tokens.IDENTIFIER)
	if !ok {
		return nil
	}

	decl := &VarDecl{
		Name: &Identifier{
			Name: nameTok.Lexeme,
			Rng:  rangeOf(nameTok),
		},
	}

	if p.cur().Type == tokens.IS {
		p.advance()
		typeTok, ok := p.expect(tokens.IDENTIFIER)
		if !ok {
			return nil
		}

		decl.Type = &Identifier{
			Name: typeTok.Lexeme,
			Rng:  rangeOf(typeTok),
		}
	}

	switch p.cur().Type {
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
	case tokens.EQUAL:
		p.advance()
		def := p.parseExpression()
		if def == nil {
			return nil
		}
		decl.Default = def
		decl.Rng = rangePos{
			Start: rangeOf(kwTok).Start,
			End:   def.Range().End,
		}
	default:
		end := rangeOf(nameTok)
		if decl.Type != nil {
			end = decl.Type.Rng
		}
		decl.Rng = rangePos{
			Start: rangeOf(kwTok).Start,
			End:   end.End,
		}
	}

	return decl
}
