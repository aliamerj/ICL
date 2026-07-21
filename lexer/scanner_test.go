package lexer

import (
	"io"
	"os"
	"reflect"
	"testing"
)

func TestScannerEmptySource(t *testing.T) {
	assertTokens(t, "", []expectedToken{{EOF, "", nil, 1}})
}

func TestScannerSingleCharacterTokens(t *testing.T) {
	assertTokens(t, "(){}.-+;*/", []expectedToken{
		{LEFT_PAREN, "(", nil, 1}, {RIGHT_PAREN, ")", nil, 1},
		{LEFT_BRACE, "{", nil, 1}, {RIGHT_BRACE, "}", nil, 1},
		{DOT, ".", nil, 1}, {MINUS, "-", nil, 1}, {PLUS, "+", nil, 1},
		{SEMICOLON, ";", nil, 1}, {STAR, "*", nil, 1}, {SLASH, "/", nil, 1},
		{EOF, "", nil, 1},
	})
}

func TestScannerOneAndTwoCharacterOperators(t *testing.T) {
	assertTokens(t, "! != = == > >= < <=", []expectedToken{
		{BANG, "!", nil, 1}, {BANG_EQUAL, "!=", nil, 1},
		{EQUAL, "=", nil, 1}, {EQUAL_EQUAL, "==", nil, 1},
		{GREATER, ">", nil, 1}, {GREATER_EQUAL, ">=", nil, 1},
		{LESS, "<", nil, 1}, {LESS_EQUAL, "<=", nil, 1}, {EOF, "", nil, 1},
	})
}

func TestScannerIdentifiers(t *testing.T) {
	// Keywords are intentionally empty for now, so all words are identifiers.
	assertTokens(t, "alpha _private value2 A_B_3", []expectedToken{
		{IDENTIFIER, "alpha", nil, 1}, {IDENTIFIER, "_private", nil, 1},
		{IDENTIFIER, "value2", nil, 1}, {IDENTIFIER, "A_B_3", nil, 1}, {EOF, "", nil, 1},
	})
}

func TestScannerNumbers(t *testing.T) {
	assertTokens(t, "0 42 3.14 0.5 123.", []expectedToken{
		{NUMBER, "0", float64(0), 1}, {NUMBER, "42", float64(42), 1},
		{NUMBER, "3.14", float64(3.14), 1}, {NUMBER, "0.5", float64(0.5), 1},
		{NUMBER, "123", float64(123), 1}, {DOT, ".", nil, 1}, {EOF, "", nil, 1},
	})
}

func TestScannerString(t *testing.T) {
	assertTokens(t, `"hello, world" ""`, []expectedToken{
		{STRING, `"hello, world"`, "hello, world", 1},
		{STRING, `""`, "", 1}, {EOF, "", nil, 1},
	})
}

func TestScannerMultilineStringUpdatesLine(t *testing.T) {
	assertTokens(t, "\"first\nsecond\" next", []expectedToken{
		{STRING, "\"first\nsecond\"", "first\nsecond", 2},
		{IDENTIFIER, "next", nil, 2}, {EOF, "", nil, 2},
	})
}

func TestScannerWhitespaceCommentsAndLineNumbers(t *testing.T) {
	assertTokens(t, "  // ignored\nfoo\t+\n  12 // also ignored\nbar", []expectedToken{
		{IDENTIFIER, "foo", nil, 2}, {PLUS, "+", nil, 2},
		{NUMBER, "12", float64(12), 3}, {IDENTIFIER, "bar", nil, 4}, {EOF, "", nil, 4},
	})
}

func TestScannerMixedSource(t *testing.T) {
	assertTokens(t, `var_name = (12.5 >= 10) + "ok"; // comment`, []expectedToken{
		{IDENTIFIER, "var_name", nil, 1}, {EQUAL, "=", nil, 1}, {LEFT_PAREN, "(", nil, 1},
		{NUMBER, "12.5", float64(12.5), 1}, {GREATER_EQUAL, ">=", nil, 1},
		{NUMBER, "10", float64(10), 1}, {RIGHT_PAREN, ")", nil, 1}, {PLUS, "+", nil, 1}, {STRING, `"ok"`, "ok", 1},
		{SEMICOLON, ";", nil, 1}, {EOF, "", nil, 1},
	})
}

func TestScannerUnexpectedCharacterReportsErrorAndContinues(t *testing.T) {
	output := captureStderr(func() {
		assertTokens(t, "@ok", []expectedToken{{IDENTIFIER, "ok", nil, 1}, {EOF, "", nil, 1}})
	})
	if output == "" {
		t.Fatal("expected an error for an unexpected character")
	}
}

func TestScannerUnterminatedStringReportsErrorAndOmitsToken(t *testing.T) {
	output := captureStderr(func() {
		assertTokens(t, `"unfinished`, []expectedToken{{EOF, "", nil, 1}})
	})
	if output == "" {
		t.Fatal("expected an error for an unterminated string")
	}
}

type expectedToken struct {
	tokenType TokenType
	lexeme    string
	literal   any
	line      int
}

func assertTokens(t *testing.T, source string, expected []expectedToken) {
	t.Helper()
	got := NewScanner(source).Tokens()
	if len(got) != len(expected) {
		t.Fatalf("source %q: got %d tokens, want %d: %#v", source, len(got), len(expected), got)
	}
	for i, want := range expected {
		if got[i].Type != want.tokenType || got[i].Lexeme != want.lexeme ||
			got[i].Line != want.line || !reflect.DeepEqual(got[i].Literal, want.literal) {
			t.Errorf("source %q token %d: got %#v, want type=%v lexeme=%q literal=%#v line=%d",
				source, i, got[i], want.tokenType, want.lexeme, want.literal, want.line)
		}
	}
}

func captureStderr(fn func()) string {
	original := os.Stderr
	read, write, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stderr = write
	fn()
	_ = write.Close()
	os.Stderr = original
	output, _ := io.ReadAll(read)
	_ = read.Close()
	return string(output)
}

func TestScannerUnicodeIdentifiers(t *testing.T) {
	assertTokens(t, "\u03c0 caf\u00e9_\u53d8\u91cf2", []expectedToken{
		{IDENTIFIER, "\u03c0", nil, 1},
		{IDENTIFIER, "caf\u00e9_\u53d8\u91cf2", nil, 1},
		{EOF, "", nil, 1},
	})
}

func TestScannerStringEscapes(t *testing.T) {
	assertTokens(t, `"line\n\"quoted\"\\tab\t"`, []expectedToken{
		{STRING, `"line\n\"quoted\"\\tab\t"`, "line\n\"quoted\"\\tab\t", 1},
		{EOF, "", nil, 1},
	})
}

func TestScannerTokensReturnsACopy(t *testing.T) {
	scanner := NewScanner("name")
	tokens := scanner.Tokens()
	tokens[0].Lexeme = "changed"
	tokens[0].Type = NUMBER

	original := scanner.Tokens()
	if original[0].Lexeme != "name" || original[0].Type != IDENTIFIER {
		t.Fatalf("Tokens returned mutable scanner state: got %#v", original[0])
	}
}

func TestScannerKeywordsFromTable(t *testing.T) {
	for keyword, wantType := range keywords {
		tokens := NewScanner(keyword).Tokens()
		if len(tokens) != 2 {
			t.Fatalf("keyword %q: got %d tokens, want keyword and EOF", keyword, len(tokens))
		}
		if tokens[0].Type != wantType || tokens[0].Lexeme != keyword {
			t.Errorf("keyword %q: got %#v, want type=%v and lexeme=%q", keyword, tokens[0], wantType, keyword)
		}
	}
}
