package lexer

func (s *Scanner) scanToken() {
	switch ch := s.next(); ch {
	case '(':
		s.addToken(LEFT_PAREN)
	case ')':
		s.addToken(RIGHT_PAREN)
	case '{':
		s.addToken(LEFT_BRACE)
	case '}':
		s.addToken(RIGHT_BRACE)
	case '.':
		s.addToken(DOT)
	case '-':
		s.addToken(MINUS)
	case '+':
		s.addToken(PLUS)
	case ';':
		s.addToken(SEMICOLON)
	case '*':
		s.addToken(STAR)
	case '!':
		s.addConditionalToken('=', BANG_EQUAL, BANG)
	case '=':
		s.addConditionalToken('=', EQUAL_EQUAL, EQUAL)
	case '<':
		s.addConditionalToken('=', LESS_EQUAL, LESS)
	case '>':
		s.addConditionalToken('=', GREATER_EQUAL, GREATER)
	case '/':
		if s.match('/') {
			for s.peek() != '\n' && !s.isAtEnd() {
				s.next()
			}
			return
		}
		s.addToken(SLASH)
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
			reportError(s.line, s.current, "unexpected character")
		}
	}
}
