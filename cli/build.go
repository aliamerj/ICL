package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliamerj/icl/pipeline"
	"github.com/aliamerj/icl/tfjson"
)

func runBuild(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)

	outDirShort := fs.String("o", "", "output directory (defaults to the source file directory)")
	outDirLong := fs.String("output", "", "output directory (defaults to the source file directory)")

	args = normalizeArgsWithValues(args, map[string]bool{
		"-o":       true,
		"--output": true,
	})

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: icl build <file.ic> [--output|-o <directory>]")
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
	sourceText := string(source)

	configs, diags := pipeline.Run(sourceText)
	if len(diags) > 0 {
		printDiagnostics(diags, sourceText, path, stderr)
		return 1
	}

	out, err := tfjson.Marshal(configs)
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to build Terraform JSON: %v\n", err)
		return 1
	}

	outDir := *outDirLong
	if outDir == "" {
		outDir = *outDirShort
	}

	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	outputName := base + ".tf.json"

	var outputPath string
	if outDir == "" {
		outputPath = filepath.Join(filepath.Dir(path), outputName)
	} else {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			fmt.Fprintf(stderr, "error: cannot create output directory %s: %v\n", outDir, err)
			return 1
		}

		outputPath = filepath.Join(outDir, outputName)
	}

	if err := os.WriteFile(outputPath, out, 0o644); err != nil {
		fmt.Fprintf(stderr, "error: cannot write %s: %v\n", outputPath, err)
		return 1
	}

	fmt.Fprintf(stdout, "wrote %s\n", outputPath)
	return 0
}
