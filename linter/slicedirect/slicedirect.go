// Static analyzer to ensure that code does not contain applications of [:]
// on expressions which are already slices.
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/slicedirect"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(slicedirect.Analyzer)
}
