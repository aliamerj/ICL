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
		case tokens.PROVIDER:
			if block := p.parseBlock(tokens.PROVIDER); block != nil {
				prog.Statements = append(prog.Statements, block)
			} else {
				p.synchronize()
			}
		case tokens.RESOURCE:
			if block := p.parseBlock(tokens.RESOURCE); block != nil {
				prog.Statements = append(prog.Statements, block)
			} else {
				p.synchronize()
			}
		case tokens.LOOKUP:
			if block := p.parseBlock(tokens.LOOKUP); block != nil {
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

	prog.Rng = spanOf(startTok, p.cur())
	return prog
}

func (p *parser) synchronize() {
	p.advance()

	for p.cur().Type != tokens.EOF {
		if p.cur().Type == tokens.RIGHT_BRACE {
			p.advance()
			return
		}
		switch p.cur().Type {
		case tokens.PROVIDER, tokens.RESOURCE:
			return
		}
		p.advance()
	}
}
