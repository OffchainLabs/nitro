// Static analyzer to ensure that errors are handled in go code.
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/errcheck"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(errcheck.Analyzer)
}
