package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return printUsage(stderr)
	}
	switch args[0] {
	case "version":
		return printVersion(stdout)
	case "inspect":
		return runInspect(args[1:], stdout, stderr)
	case "build":
		return runBuild(args[1:], stdout, stderr)
	case "fmt":
		return runFmt(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "icl: unknown command %q\n\n", args[0])
		return printUsage(stderr)

	}
}

func printUsage(w io.Writer) int {
	fmt.Fprintln(w, `usage: icl <command> [arguments]

commands:
  inspect <file.ic>          Parse and evaluate a file, print the resolved config
  build <file.ic> [-o path]  Compile a file to Terraform JSON (main.tf.json)
  fmt  
  version                     Print the icl version`)
	return 2
}

func validateICFile(path string) error {
	if filepath.Ext(path) != ".ic" {
		return fmt.Errorf("%q is not an ICL source file (expected .ic)", path)
	}
	return nil
}

func normalizeArgs(args []string) []string {
	var flags []string
	var positional []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
		} else {
			positional = append(positional, arg)
		}
	}

	return append(flags, positional...)
}

func normalizeArgsWithValues(args []string, valueFlags map[string]bool) []string {
	var flags []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)

			if valueFlags[arg] && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}

		positional = append(positional, arg)
	}

	return append(flags, positional...)
}
