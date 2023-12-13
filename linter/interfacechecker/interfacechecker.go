// Static analyzer to prevent conditional checks on select interfaces.
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/interfacechecker"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(interfacechecker.Analyzer)
}
