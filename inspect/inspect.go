package inspect

// Testing only
import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/lexer"
)

type Options struct {
	Filename   string
	Color      bool
	IncludeEOF bool
	Reporter   *diagnostics.Reporter
}

type Inspection struct {
	Source           string
	Duration         time.Duration
	Bytes            int
	Tokens           []lexer.Token
	Diagnostics      []diagnostics.Diagnostic
	DiagnosticOutput string
}

func New(source string) Inspection {
	return NewWithOptions(source, Options{
		Filename: "<icl-file>",
		Color:    true,
	})
}

func NewWithOptions(source string, options Options) Inspection {
	filename := options.Filename
	if filename == "" {
		filename = "<icl>"
	}
	reporter := options.Reporter
	if reporter == nil {
		reporter = diagnostics.New(source)
	}

	start := time.Now()
	scanner := lexer.New(source)
	duration := time.Since(start)

	tokens := scanner.Tokens()
	if !options.IncludeEOF {
		tokens = withoutEOF(tokens)
	}

	diagnosticsList := scanner.Diagnostics()
	formatter := diagnostics.NewFormatter(source)
	formatter.SetFilename(filename)
	formatter.SetColor(options.Color)

	var diagnosticOutput bytes.Buffer
	_ = diagnostics.WriteAll(&diagnosticOutput, formatter, diagnosticsList)

	return Inspection{
		Source:           source,
		Duration:         duration,
		Bytes:            len(source),
		Tokens:           tokens,
		Diagnostics:      diagnosticsList,
		DiagnosticOutput: diagnosticOutput.String(),
	}
}

func (i Inspection) HasErrors() bool {
	for _, diagnostic := range i.Diagnostics {
		if diagnostic.Severity == diagnostics.Error {
			return true
		}
	}
	return false
}

func (i Inspection) String() string {
	var b strings.Builder
	fmt.Fprintln(&b, "ICL lexer inspection")
	fmt.Fprintf(&b, "source bytes: %d\n", i.Bytes)
	fmt.Fprintf(&b, "scan time: %s\n", i.Duration)
	fmt.Fprintf(&b, "tokens: %d\n", len(i.Tokens))
	for _, token := range i.Tokens {
		literal := ""
		if token.Literal != nil {
			literal = fmt.Sprintf(" literal=%v", token.Literal)
		}
		fmt.Fprintf(&b, "  line %-3d %-14s lexeme=%s%s\n", token.Line, token.Type, strconv.Quote(token.Lexeme), literal)
	}

	if len(i.Diagnostics) == 0 {
		fmt.Fprintln(&b, "diagnostics: none")
		return b.String()
	}

	fmt.Fprintf(&b, "diagnostics: %d\n", len(i.Diagnostics))
	b.WriteString(i.DiagnosticOutput)
	return b.String()
}

func withoutEOF(tokens []lexer.Token) []lexer.Token {
	result := make([]lexer.Token, 0, len(tokens))
	for _, token := range tokens {
		if token.Type != lexer.EOF {
			result = append(result, token)
		}
	}
	return result
}
