package lexer

import "unicode/utf8"

func (s *scanner) match(expected rune) bool {
	if s.isAtEnd() || s.peek() != expected {
		return false
	}
	s.next()
	return true
}

func (s *scanner) next() rune {
	ch, size := utf8.DecodeRuneInString(s.source[s.current:])
	s.current += size
	return ch
}

func (s *scanner) peek() rune {
	if s.isAtEnd() {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(s.source[s.current:])
	return ch
}

func (s *scanner) peekNext() rune {
	if s.isAtEnd() {
		return 0
	}
	_, size := utf8.DecodeRuneInString(s.source[s.current:])
	if s.current+size >= len(s.source) {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(s.source[s.current+size:])
	return ch
}

func (s *scanner) isAtEnd() bool {
	return s.current >= len(s.source)
}
