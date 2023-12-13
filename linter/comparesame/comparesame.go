// Static Analyzer to ensure code does not contain comparisons of identical expressions.
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/comparesame"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(comparesame.Analyzer)
}
