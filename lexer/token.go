package lexer

import "github.com/aliamerj/icl/tokens"

type Comment struct {
	Text string
	Line int
}

type Token struct {
	Type    tokens.Type
	Lexeme  string
	Literal any
	Line    int
	Offset  int
}

func (s *scanner) Tokens() []Token {
	return append([]Token(nil), s.tokens...)
}

func (s *scanner) Comments() []Comment {
	return append([]Comment(nil), s.comments...)
}

func (s *scanner) addConditionalToken(expected rune, yes, no tokens.Type) {
	if s.match(expected) {
		s.addToken(yes)
		return
	}
	s.addToken(no)
}

func (s *scanner) addToken(tokenType tokens.Type) {
	s.addTokenLiteral(tokenType, nil)
}

func (s *scanner) addTokenLiteral(tokenType tokens.Type, literal any) {
	s.tokens = append(s.tokens, Token{
		Type:    tokenType,
		Lexeme:  s.source[s.start:s.current],
		Literal: literal,
		Line:    s.line,
		Offset:  s.start,
	})
}
