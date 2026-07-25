package lexer

import (
	"reflect"
	"testing"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/tokens"
)

func TestScannerEmptySource(t *testing.T) {
	assertTokens(t, "", []expectedToken{{tokens.EOF, "", nil, 1}})
}

func TestScannerSingleCharacterTokens(t *testing.T) {
	assertTokens(t, "(){}.-+;*/", []expectedToken{
		{tokens.LEFT_PAREN, "(", nil, 1}, {tokens.RIGHT_PAREN, ")", nil, 1},
		{tokens.LEFT_BRACE, "{", nil, 1}, {tokens.RIGHT_BRACE, "}", nil, 1},
		{tokens.DOT, ".", nil, 1}, {tokens.MINUS, "-", nil, 1}, {tokens.PLUS, "+", nil, 1},
		{tokens.SEMICOLON, ";", nil, 1}, {tokens.STAR, "*", nil, 1}, {tokens.SLASH, "/", nil, 1},
		{tokens.EOF, "", nil, 1},
	})
}

func TestScannerOneAndTwoCharacterOperators(t *testing.T) {
	assertTokens(t, "! != = == > >= < <=", []expectedToken{
		{tokens.BANG, "!", nil, 1}, {tokens.BANG_EQUAL, "!=", nil, 1},
		{tokens.EQUAL, "=", nil, 1}, {tokens.EQUAL_EQUAL, "==", nil, 1},
		{tokens.GREATER, ">", nil, 1}, {tokens.GREATER_EQUAL, ">=", nil, 1},
		{tokens.LESS, "<", nil, 1}, {tokens.LESS_EQUAL, "<=", nil, 1}, {tokens.EOF, "", nil, 1},
	})
}

func TestScannerIdentifiers(t *testing.T) {
	// Keywords are intentionally empty for now, so all words are identifiers.
	assertTokens(t, "alpha _private value2 A_B_3", []expectedToken{
		{tokens.IDENTIFIER, "alpha", nil, 1}, {tokens.IDENTIFIER, "_private", nil, 1},
		{tokens.IDENTIFIER, "value2", nil, 1}, {tokens.IDENTIFIER, "A_B_3", nil, 1}, {tokens.EOF, "", nil, 1},
	})
}

func TestScannerNumbers(t *testing.T) {
	assertTokens(t, "0 42 3.14 0.5 123.", []expectedToken{
		{tokens.NUMBER_INT, "0", int64(0), 1},
		{tokens.NUMBER_INT, "42", int64(42), 1},
		{tokens.NUMBER_FLOAT, "3.14", float64(3.14), 1},
		{tokens.NUMBER_FLOAT, "0.5", float64(0.5), 1},
		{tokens.NUMBER_INT, "123", int64(123), 1}, {tokens.DOT, ".", nil, 1},
		{tokens.EOF, "", nil, 1},
	})
}

func TestScannerString(t *testing.T) {
	assertTokens(t, `"hello, world" ""`, []expectedToken{
		{tokens.STRING, `"hello, world"`, "hello, world", 1},
		{tokens.STRING, `""`, "", 1}, {tokens.EOF, "", nil, 1},
	})
}

func TestScannerMultilineStringUpdatesLine(t *testing.T) {
	assertTokens(t, "\"first\nsecond\" next", []expectedToken{
		{tokens.STRING, "\"first\nsecond\"", "first\nsecond", 2},
		{tokens.IDENTIFIER, "next", nil, 2}, {tokens.EOF, "", nil, 2},
	})
}

func TestScannerWhitespaceCommentsAndLineNumbers(t *testing.T) {
	assertTokens(t, "  // ignored\nfoo\t+\n  12 // also ignored\nbar", []expectedToken{
		{tokens.IDENTIFIER, "foo", nil, 2},
		{tokens.PLUS, "+", nil, 2},
		{tokens.NUMBER_INT, "12", int64(12), 3},
		{tokens.IDENTIFIER, "bar", nil, 4}, {tokens.EOF, "", nil, 4},
	})
}

func TestScannerMixedSource(t *testing.T) {
	assertTokens(t, `var_name = (12.5 >= 10) + "ok"; // comment`, []expectedToken{
		{tokens.IDENTIFIER, "var_name", nil, 1}, {tokens.EQUAL, "=", nil, 1},
		{tokens.LEFT_PAREN, "(", nil, 1},
		{tokens.NUMBER_FLOAT, "12.5", float64(12.5), 1},
		{tokens.GREATER_EQUAL, ">=", nil, 1},
		{tokens.NUMBER_INT, "10", int64(10), 1}, {tokens.RIGHT_PAREN, ")", nil, 1},
		{tokens.PLUS, "+", nil, 1}, {tokens.STRING, `"ok"`, "ok", 1},
		{tokens.SEMICOLON, ";", nil, 1}, {tokens.EOF, "", nil, 1},
	})
}

func TestScannerUnexpectedCharacterReportsErrorAndContinues(t *testing.T) {
	scanner, diags := newScanner("@ok")
	assertTokenList(t, scanner.Tokens(), []expectedToken{{tokens.IDENTIFIER, "ok", nil, 1}, {tokens.EOF, "", nil, 1}})
	if !diags.HasErrors() {
		t.Fatal("expected an error for an unexpected character")
	}
}

func TestScannerUnterminatedStringReportsErrorAndOmitsToken(t *testing.T) {
	scanner, diags := newScanner(`"unfinished`)
	assertTokenList(t, scanner.Tokens(), []expectedToken{{tokens.EOF, "", nil, 1}})
	if !diags.HasErrors() {
		t.Fatal("expected an error for an unterminated string")
	}
}

type expectedToken struct {
	tokenType tokens.Type
	lexeme    string
	literal   any
	line      int
}

func TestScannerUnicodeIdentifiers(t *testing.T) {
	assertTokens(t, "\u03c0 caf\u00e9_\u53d8\u91cf2", []expectedToken{
		{tokens.IDENTIFIER, "\u03c0", nil, 1},
		{tokens.IDENTIFIER, "caf\u00e9_\u53d8\u91cf2", nil, 1},
		{tokens.EOF, "", nil, 1},
	})
}

func TestScannerStringEscapes(t *testing.T) {
	assertTokens(t, `"line\n\"quoted\"\\tab\t"`, []expectedToken{
		{tokens.STRING, `"line\n\"quoted\"\\tab\t"`, "line\n\"quoted\"\\tab\t", 1},
		{tokens.EOF, "", nil, 1},
	})
}

func TestScannerTokensReturnsACopy(t *testing.T) {
	scanner := New("name", diagnostics.New("name"))
	tks := scanner.Tokens()
	tks[0].Lexeme = "changed"
	tks[0].Type = tokens.NUMBER_INT

	original := scanner.Tokens()
	if original[0].Lexeme != "name" || original[0].Type != tokens.IDENTIFIER {
		t.Fatalf("Tokens returned mutable scanner state: got %#v", original[0])
	}
}

func TestScannerKeywordsFromTable(t *testing.T) {
	for keyword, wantType := range keywords {
		tokens := New(keyword, diagnostics.New(keyword)).Tokens()
		if len(tokens) != 2 {
			t.Fatalf("keyword %q: got %d tokens, want keyword and EOF", keyword, len(tokens))
		}
		if tokens[0].Type != wantType || tokens[0].Lexeme != keyword {
			t.Errorf("keyword %q: got %#v, want type=%v and lexeme=%q", keyword, tokens[0], wantType, keyword)
		}
	}
}

func TestScannerTokenOffsets(t *testing.T) {
	source := `provider aws {}`
	tokens := New(source, diagnostics.New(source)).Tokens()

	want := []int{0, 9, 13} // "provider", "aws", "{" - start offsets
	for i, w := range want {
		if tokens[i].Offset != w {
			t.Errorf("token %d (%q): Offset = %d, want %d", i, tokens[i].Lexeme, tokens[i].Offset, w)
		}
	}
	if tokens[len(tokens)-1].Offset != len(source) {
		t.Errorf("EOF Offset = %d, want %d", tokens[len(tokens)-1].Offset, len(source))
	}
}

func assertTokens(t *testing.T, source string, expected []expectedToken) {
	t.Helper()
	assertTokenList(t, New(source, diagnostics.New(source)).Tokens(), expected)
}

func assertTokenList(t *testing.T, got []Token, expected []expectedToken) {
	t.Helper()
	if len(got) != len(expected) {
		t.Fatalf("got %d tokens, want %d: %#v", len(got), len(expected), got)
	}
	for i, want := range expected {
		if got[i].Type != want.tokenType || got[i].Lexeme != want.lexeme ||
			got[i].Line != want.line || !reflect.DeepEqual(got[i].Literal, want.literal) {
			t.Errorf("token %d: got %#v, want type=%v lexeme=%q literal=%#v line=%d",
				i, got[i], want.tokenType, want.lexeme, want.literal, want.line)
		}
	}
}

func newScanner(source string) (*scanner, *diagnostics.Reporter) {
  diags := diagnostics.New(source)
	scan := New(source, diags)
	return scan, diags
}
