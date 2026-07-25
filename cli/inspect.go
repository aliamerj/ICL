package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/output"
	"github.com/aliamerj/icl/pipeline"
)

func runInspect(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  icl inspect [--json|-j] <file.ic>")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Options:")
		fs.PrintDefaults()
	}

	jsonLong := fs.Bool("json", false, "output as JSON instead of pretty text")
	jsonShort := fs.Bool("j", false, "output as JSON instead of pretty text")

	args = normalizeArgs(args)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	path := fs.Arg(0)

	if err := validateICFile(path); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot read %s: %v\n", path, err)
		return 1
	}
	strSource := string(source)

	configs, diags := pipeline.Run(strSource)
	if len(diags) > 0 {
		printDiagnostics(diags, strSource, path, stderr)
		return 1
	}

	// output provider
	if *jsonLong || *jsonShort {
		if err := output.FormatJSON(configs, stdout); err != nil {
			fmt.Fprintf(stderr, "error: failed to format output: %v\n", err)
			return 1
		}
	} else {
		output.FormatPretty(configs, stdout)
	}

	return 0
}

func printDiagnostics(diags []diagnostics.Diagnostic, source, filename string, w io.Writer) {
	formatter := diagnostics.NewFormatter(source)
	formatter.SetFilename(filename)
	_ = diagnostics.WriteAll(w, formatter, diags)
}
