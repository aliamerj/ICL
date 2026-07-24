package main

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/inspect"
	"github.com/aliamerj/icl/parser"
)

func main() {
	source := `provider aws {
      source  = "hashicorp/aws"
      version = "5.37.0"
      someNumber = 5
      someFloat = 12.6
  }
`
	r := diagnostics.New(source)

	insp := inspect.NewWithOptions(source, inspect.Options{
		Reporter: r,
	})
	p := parser.New(insp.Tokens, r)
	prog := p.ParseProgram()
	for _, d := range r.Diagnostics() {
		f := diagnostics.NewFormatter(source)
		fmt.Print(f.Format(d))
	}
	fmt.Printf("%+v\n", prog)
}
