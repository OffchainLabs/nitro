// Static analyzer to prevent leaking globals in tests.
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/ineffassign"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(ineffassign.Analyzer)
}
