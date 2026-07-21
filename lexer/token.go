package lexer

type Token struct {
	Type    TokenType
	Lexeme  string
	Literal any
	Line    int
}

func (s *Scanner) Tokens() []Token {
	return append([]Token(nil), s.tokens...)
}

func (s *Scanner) addConditionalToken(expected rune, yes, no TokenType) {
	if s.match(expected) {
		s.addToken(yes)
		return
	}
	s.addToken(no)
}

func (s *Scanner) addToken(tokenType TokenType) {
	s.addTokenLiteral(tokenType, nil)
}

func (s *Scanner) addTokenLiteral(tokenType TokenType, literal any) {
	s.tokens = append(s.tokens, Token{
		Type:    tokenType,
		Lexeme:  s.source[s.start:s.current],
		Literal: literal,
		Line:    s.line,
	})
}
