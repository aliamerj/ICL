package lexer

import (
	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/tokens"
)

type scanner struct {
	source string
	tokens []Token

	reporter *diagnostics.Reporter

	start   int
	current int
	line    int
}

func New(source string, reporter *diagnostics.Reporter) *scanner {
	t := &scanner{
		source:   source,
		tokens:   make([]Token, 0, len(source)/2+1),
		reporter: reporter,
		line:     1,
	}
	t.scanTokens()
	return t
}

func (s *scanner) scanTokens() {
	for !s.isAtEnd() {
		s.start = s.current
		s.scanToken()
	}

	s.tokens = append(s.tokens, Token{
		Type:   tokens.EOF,
		Line:   s.line,
		Offset: s.current,
	})
}
