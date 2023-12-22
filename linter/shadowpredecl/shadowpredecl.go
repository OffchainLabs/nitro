// Static analyzer which disallows declaring constructs that shadow predeclared
// Go identifiers by having the same name.
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/shadowpredecl"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(shadowpredecl.Analyzer)
}
