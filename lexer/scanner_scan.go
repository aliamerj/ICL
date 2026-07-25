package lexer

import (
	"strconv"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/tokens"
)

func (s *scanner) scanToken() {
	switch ch := s.next(); ch {
	case '(':
		s.addToken(tokens.LEFT_PAREN)
	case ')':
		s.addToken(tokens.RIGHT_PAREN)
	case '{':
		s.addToken(tokens.LEFT_BRACE)
	case '}':
		s.addToken(tokens.RIGHT_BRACE)
	case '.':
		s.addToken(tokens.DOT)
	case '-':
		s.addToken(tokens.MINUS)
	case '+':
		s.addToken(tokens.PLUS)
	case ';':
		s.addToken(tokens.SEMICOLON)
	case '*':
		s.addToken(tokens.STAR)
	case '!':
		s.addConditionalToken('=', tokens.BANG_EQUAL, tokens.BANG)
	case '=':
		s.addConditionalToken('=', tokens.EQUAL_EQUAL, tokens.EQUAL)
	case '<':
		s.addConditionalToken('=', tokens.LESS_EQUAL, tokens.LESS)
	case '>':
		s.addConditionalToken('=', tokens.GREATER_EQUAL, tokens.GREATER)
	case '/':
		if s.match('/') {
			for s.peek() != '\n' && !s.isAtEnd() {
				s.next()
			}
			return
		}
		s.addToken(tokens.SLASH)
	case ' ', '\r', '\t':
		return
	case '\n':
		s.line++
	case '"':
		s.string()
	default:
		switch {
		case isDigit(ch):
			s.number()
		case isAlpha(ch):
			s.identifier()
		default:
			s.reporter.ErrorAtOffsetWithCode(s.start, diagnostics.UNEXPECTED_CHAR, "unexpected character "+strconv.Quote(s.source[s.start:s.current]), "remove it or replace it with a valid token")
		}
	}
}
