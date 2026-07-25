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

	prog.Rng = spanOf(startTok, p.cur())
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
