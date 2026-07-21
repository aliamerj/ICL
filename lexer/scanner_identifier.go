package lexer

import (
	"strconv"
	"strings"
	"unicode"
)

func (s *Scanner) identifier() {
	for isAlphaNumeric(s.peek()) {
		s.next()
	}

	text := s.source[s.start:s.current]
	tokenType, ok := keywords[text]
	if !ok {
		tokenType = IDENTIFIER
	}
	s.addToken(tokenType)
}

func (s *Scanner) string() {
	var value strings.Builder

	for !s.isAtEnd() {
		ch := s.next()
		switch ch {
		case '"':
			s.addTokenLiteral(STRING, value.String())
			return
		case '\\':
			if s.isAtEnd() {
				reportError(s.line, s.current, "unterminated string")
				return
			}
			switch escaped := s.next(); escaped {
			case 'n':
				value.WriteByte('\n')
			case 'r':
				value.WriteByte('\r')
			case 't':
				value.WriteByte('\t')
			case '\\', '"':
				value.WriteRune(escaped)
			default:
				// Preserve unknown escapes without silently dropping data.
				value.WriteRune('\\')
				value.WriteRune(escaped)
			}
		case '\n':
			s.line++
			value.WriteRune(ch)
		default:
			value.WriteRune(ch)
		}
	}

	reportError(s.line, s.current, "unterminated string")
}

func (s *Scanner) number() {
	for isDigit(s.peek()) {
		s.next()
	}

	if s.peek() == '.' && isDigit(s.peekNext()) {
		s.next()
		for isDigit(s.peek()) {
			s.next()
		}
	}

	value, err := strconv.ParseFloat(s.source[s.start:s.current], 64)
	if err != nil {
		reportError(s.line, s.current, err.Error())
		return
	}
	s.addTokenLiteral(NUMBER, value)
}

func isAlphaNumeric(ch rune) bool {
	return isAlpha(ch) || isDigit(ch)
}

func isAlpha(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
