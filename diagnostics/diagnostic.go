package diagnostics

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Severity uint8

const (
	Error Severity = iota
	Warning
	Note
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	case Note:
		return "note"
	default:
		return "unknown"
	}
}

type Location struct {
	Line   int
	Column int
	Offset int
}

type Position = Location

type Span struct {
	Start Location
	End   Location
}

type Diagnostic struct {
	Severity Severity
	Code     errorCode
	Message  string
	Help     string
	Span     Span
}

// Reporter records diagnostics. It does not format or print them.
type Reporter struct {
	source      string
	diagnostics []Diagnostic
}

func New(source string) *Reporter {
	return &Reporter{source: source}
}

func (r *Reporter) Error(line, column int, message string) {
	r.Report(Error, line, column, message)
}

func (r *Reporter) Warning(line, column int, message string) {
	r.Report(Warning, line, column, message)
}

func (r *Reporter) Note(line, column int, message string) {
	r.Report(Note, line, column, message)
}

func (r *Reporter) ErrorWithHint(line, column int, message, help string) {
	r.ReportWithHint(Error, line, column, message, help)
}

func (r *Reporter) WarningWithHint(line, column int, message, help string) {
	r.ReportWithHint(Warning, line, column, message, help)
}

func (r *Reporter) NoteWithHint(line, column int, message, help string) {
	r.ReportWithHint(Note, line, column, message, help)
}

func (r *Reporter) ErrorWithCode(line, column int, code errorCode, message, help string) {
	r.ReportWithCode(Error, line, column, code, message, help)
}

func (r *Reporter) WarningWithCode(line, column int, code errorCode, message, help string) {
	r.ReportWithCode(Warning, line, column, code, message, help)
}

func (r *Reporter) NoteWithCode(line, column int, code errorCode, message, help string) {
	r.ReportWithCode(Note, line, column, code, message, help)
}

func (r *Reporter) ErrorAtOffset(offset int, message string) {
	r.ReportAtOffset(Error, offset, message)
}

func (r *Reporter) WarningAtOffset(offset int, message string) {
	r.ReportAtOffset(Warning, offset, message)
}

func (r *Reporter) NoteAtOffset(offset int, message string) {
	r.ReportAtOffset(Note, offset, message)
}

func (r *Reporter) ErrorAtOffsetWithHint(offset int, message, help string) {
	r.ReportAtOffsetWithHint(Error, offset, message, help)
}

func (r *Reporter) WarningAtOffsetWithHint(offset int, message, help string) {
	r.ReportAtOffsetWithHint(Warning, offset, message, help)
}

func (r *Reporter) NoteAtOffsetWithHint(offset int, message, help string) {
	r.ReportAtOffsetWithHint(Note, offset, message, help)
}

func (r *Reporter) ErrorAtOffsetWithCode(offset int, code errorCode, message, help string) {
	r.ReportAtOffsetWithCode(Error, offset, code, message, help)
}

func (r *Reporter) WarningAtOffsetWithCode(offset int, code errorCode, message, help string) {
	r.ReportAtOffsetWithCode(Warning, offset, code, message, help)
}

func (r *Reporter) NoteAtOffsetWithCode(offset int, code errorCode, message, help string) {
	r.ReportAtOffsetWithCode(Note, offset, code, message, help)
}

func (r *Reporter) Report(severity Severity, line, column int, message string) {
	r.ReportWithHint(severity, line, column, message, "")
}

func (r *Reporter) ReportWithHint(severity Severity, line, column int, message, help string) {
	r.ReportWithCode(severity, line, column, "", message, help)
}

func (r *Reporter) ReportWithCode(severity Severity, line, column int, code errorCode, message, help string) {
	start := normalizeLocation(Location{Line: line, Column: column, Offset: -1})
	r.report(Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  message,
		Help:     help,
		Span:     Span{Start: start, End: start},
	})
}

func (r *Reporter) ReportAtOffset(severity Severity, offset int, message string) {
	r.ReportAtOffsetWithHint(severity, offset, message, "")
}

func (r *Reporter) ReportAtOffsetWithHint(severity Severity, offset int, message, help string) {
	r.ReportAtOffsetWithCode(severity, offset, "", message, help)
}

func (r *Reporter) ReportAtOffsetWithCode(severity Severity, offset int, code errorCode, message, help string) {
	location := r.locationAtOffset(offset)
	r.report(Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  message,
		Help:     help,
		Span:     Span{Start: location, End: location},
	})
}

func (r *Reporter) ReportDiagnostic(d Diagnostic) {
	d.Span.Start = normalizeLocation(d.Span.Start)
	d.Span.End = normalizeLocation(d.Span.End)
	if d.Span.End.Line == 1 && d.Span.End.Column == 1 && d.Span.End.Offset == 0 {
		d.Span.End = d.Span.Start
	}
	r.report(d)
}

func (r *Reporter) report(d Diagnostic) {
	r.diagnostics = append(r.diagnostics, d)
}

func (r *Reporter) Diagnostics() []Diagnostic {
	return append([]Diagnostic(nil), r.diagnostics...)
}

func (r *Reporter) HasErrors() bool {
	for _, d := range r.diagnostics {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

func (r *Reporter) locationAtOffset(offset int) Location {
	line, column := lineAndColumn(r.source, offset)
	return Location{Line: line, Column: column, Offset: clampOffset(r.source, offset)}
}

type Formatter struct {
	source   string
	filename string
	color    bool
}

func NewFormatter(source string) *Formatter {
	return &Formatter{source: source, filename: "<source>"}
}

func (f *Formatter) SetFilename(filename string) {
	if filename != "" {
		f.filename = filename
	}
}

func (f *Formatter) SetColor(enabled bool) {
	f.color = enabled
}

func (f *Formatter) Format(d Diagnostic) string {
	location := normalizeLocation(d.Span.Start)
	lineText := sourceLine(f.source, location.Line)
	column := clampColumn(location.Column, lineText)
	lineNo := strconv.Itoa(location.Line)
  gutterWidth := max(len(lineNo), 2)

	severity := d.Severity.String()
	marker := markerForSpan(d.Span, column, lineText)
	helpLabel := "help"
	if f.color {
		severity = colorForSeverity(d.Severity) + severity + ansiReset
		marker = colorForSeverity(d.Severity) + marker + ansiReset
		helpLabel = ansiCyan + helpLabel + ansiReset
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s%s: %s\n", severity, codeSuffix(string(d.Code)), d.Message)
	fmt.Fprintf(&b, "%s--> %s:%d:%d\n", strings.Repeat(" ", gutterWidth+1), f.filename, location.Line, location.Column)
	fmt.Fprintf(&b, "%s |\n", strings.Repeat(" ", gutterWidth))
	fmt.Fprintf(&b, "%*d | %s\n", gutterWidth, location.Line, lineText)
	fmt.Fprintf(&b, "%s | %s%s\n", strings.Repeat(" ", gutterWidth), strings.Repeat(" ", column-1), marker)
	if d.Help != "" {
		fmt.Fprintf(&b, "%s = %s: %s\n", strings.Repeat(" ", gutterWidth), helpLabel, d.Help)
	}
	return b.String()
}


func WriteAll(w io.Writer, formatter *Formatter, diagnostics []Diagnostic) error {
	if w == nil || formatter == nil {
		return nil
	}
	for _, diagnostic := range diagnostics {
		if _, err := io.WriteString(w, formatter.Format(diagnostic)); err != nil {
			return err
		}
	}
	return nil
}

func codeSuffix(code string) string {
	if code == "" {
		return ""
	}
	return "[" + code + "]"
}

func markerForSpan(span Span, column int, line string) string {
	width := 1
	start := normalizeLocation(span.Start)
	end := normalizeLocation(span.End)
	if start.Line == end.Line && end.Column > start.Column {
		width = end.Column - start.Column
	}
	maxWidth := max(utf8.RuneCountInString(line) - column + 1, 1)

	if width > maxWidth {
		width = maxWidth
	}
	return strings.Repeat("^", width)
}

func sourceLine(source string, wanted int) string {
	lines := strings.Split(source, "\n")
	if wanted < 1 || wanted > len(lines) {
		return ""
	}
	return strings.TrimSuffix(lines[wanted-1], "\r")
}

func lineAndColumn(source string, offset int) (int, int) {
	offset = clampOffset(source, offset)
	line, column := 1, 1
	for _, ch := range source[:offset] {
		if ch == '\n' {
			line, column = line+1, 1
			continue
		}
		column++
	}
	return line, column
}

func clampOffset(source string, offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(source) {
		return len(source)
	}
	return offset
}

func normalizeLocation(location Location) Location {
	if location.Line < 1 {
		location.Line = 1
	}
	if location.Column < 1 {
		location.Column = 1
	}
	return location
}

func clampColumn(column int, line string) int {
	if column < 1 {
		return 1
	}
	lineLength := utf8.RuneCountInString(line)
	if column > lineLength+1 {
		return lineLength + 1
	}
	return column
}

func colorForSeverity(severity Severity) string {
	switch severity {
	case Error:
		return ansiRedBold
	case Warning:
		return ansiYellowBold
	case Note:
		return ansiBlueBold
	default:
		return ansiBold
	}
}

const (
	ansiReset      = "\x1b[0m"
	ansiBold       = "\x1b[1m"
	ansiRedBold    = "\x1b[1;31m"
	ansiYellowBold = "\x1b[1;33m"
	ansiBlueBold   = "\x1b[1;34m"
	ansiCyan       = "\x1b[36m"
)
