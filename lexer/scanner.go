package lexer

type Scanner struct {
	source string
	tokens []Token

	start   int
	current int
	line    int
}

func NewScanner(source string) *Scanner {
	t := &Scanner{
		source: source,
		tokens: []Token{},
		line:   1,
	}
	t.scanTokens()
	return t
}

func (s *Scanner) scanTokens() {
	for !s.isAtEnd() {
		s.start = s.current
		s.scanToken()
	}

	s.tokens = append(s.tokens, Token{
		Type: EOF,
		Line: s.line,
	})
}
