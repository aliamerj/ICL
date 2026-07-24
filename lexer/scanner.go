package lexer

import "github.com/aliamerj/icl/diagnostics"

type Scanner struct {
	source string
	tokens []Token

	reporter *diagnostics.Reporter

	start   int
	current int
	line    int
}

func New(source string) *Scanner {
	return NewWithoutReporter(source, diagnostics.New(source))
}

func NewWithoutReporter(source string,reporter *diagnostics.Reporter) *Scanner{
	t := &Scanner{
		source:   source,
		tokens:   make([]Token, 0),
		reporter: reporter,
		line:     1,
	}
	t.scanTokens()
  return t
}

func (s *Scanner) Diagnostics() []diagnostics.Diagnostic {
	return s.reporter.Diagnostics()
}

func (s *Scanner) HasErrors() bool {
	return s.reporter.HasErrors()
}

func (s *Scanner) scanTokens() {
	for !s.isAtEnd() {
		s.start = s.current
		s.scanToken()
	}

	s.tokens = append(s.tokens, Token{
			Type:   EOF,
		Line:   s.line,
		Offset: s.current,
	})
}
