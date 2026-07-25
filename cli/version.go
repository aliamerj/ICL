package cli

import (
	"fmt"
	"io"
	"runtime"
)

var (
	Version = "0.1.0-dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func printVersion(w io.Writer) int {
	fmt.Fprintf(w, "Version:    %s\n", Version)
	fmt.Fprintf(w, "Commit:     %s\n", Commit)
	fmt.Fprintf(w, "Built:      %s\n", Date)
	fmt.Fprintf(w, "Go Version: %s\n", runtime.Version())
	fmt.Fprintf(w, "OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	return 0
}
