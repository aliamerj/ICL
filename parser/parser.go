package parser

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
	"github.com/aliamerj/icl/tokens"
)

type parser struct {
	tokens   []lexer.Token
	pos      int
	reporter *diagnostics.Reporter
}

func New(tokens []lexer.Token, reporter *diagnostics.Reporter) *parser {
	return &parser{tokens: tokens, reporter: reporter}
}

func (p *parser) ParseProgram() *Program {
	prog := &Program{}
	startTok := p.cur()

	for p.cur().Type != tokens.EOF {
		switch p.cur().Type {
		case tokens.PROVIDER, tokens.RESOURCE, tokens.LOOKUP:
			if block := p.parseBlock(); block != nil {
				prog.Statements = append(prog.Statements, block)
			} else {
				p.synchronize()
			}
		case tokens.VAR:
			if varBlock := p.parseVarDecl(); varBlock != nil {
				prog.Statements = append(prog.Statements, varBlock)
			} else {
				p.synchronize()
			}
		case tokens.OUTPUT:
			if output := p.parseOutputDecl(); output != nil {
				prog.Statements = append(prog.Statements, output)
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

	prog.Rng = spanOf(startTok, p.cur())
	return prog
}

func (p *parser) synchronize() {
	for p.cur().Type != tokens.EOF {
		if p.cur().Type == tokens.RIGHT_BRACE {
			p.advance()
			return
		}
		switch p.cur().Type {
		case tokens.PROVIDER, tokens.RESOURCE, tokens.LOOKUP, tokens.VAR, tokens.OUTPUT:
			return
		}
		p.advance()
	}
}
