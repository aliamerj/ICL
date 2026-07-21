package lexer

import (
	"fmt"
	"os"
	"unicode/utf8"
)

func (s *Scanner) match(expected rune) bool {
	if s.isAtEnd() || s.peek() != expected {
		return false
	}
	s.next()
	return true
}

func (s *Scanner) next() rune {
	ch, size := utf8.DecodeRuneInString(s.source[s.current:])
	s.current += size
	return ch
}

func (s *Scanner) peek() rune {
	if s.isAtEnd() {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(s.source[s.current:])
	return ch
}

func (s *Scanner) peekNext() rune {
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

func (s *Scanner) isAtEnd() bool {
	return s.current >= len(s.source)
}

// TODO: replace later with structured lexer errors.
func reportError(line, where int, message string) {
	fmt.Fprintf(os.Stderr, "[line %d] Error%d: %s\n", line, where, message)
}
