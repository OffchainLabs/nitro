// Static analyzer that checks for high cognitive complexity and complains when
// it's too high.
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/gocognit"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(gocognit.Analyzer)
}
