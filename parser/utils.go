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
