// Static Analyzer for detecting nested or recursive mutex read lock statements.
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/recursivelock"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(recursivelock.Analyzer)
}
