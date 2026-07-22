package main

import (
	"fmt"

	"github.com/aliamerj/icl/inspect"
)

func main() {
	report := inspect.Inspect(`
    provider aws {
      source  = "hashicorp/aws"
      version = "5.37.0"
  }
    `)
	fmt.Print(report)
}
