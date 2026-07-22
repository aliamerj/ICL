package lexer

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/aliamerj/icl/diagnostics"
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
				s.reporter.ErrorAtOffsetWithCode(s.start, diagnostics.UNTERMINATED_STRING_LITERAL, "unterminated string literal", "add a closing quote before the end of the string")
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

	s.reporter.ErrorAtOffsetWithCode(s.start, diagnostics.UNTERMINATED_STRING_LITERAL, "unterminated string literal", "add a closing quote before the end of the string")
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
		s.reporter.ErrorAtOffsetWithCode(s.start, diagnostics.ERROR_NUMBER_LITERAL, err.Error(), "check that the number literal is valid")
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
