// Static analyzer for detecting unsafe uint to int casts.
// Use `lint:ignore uintcast` with proper justification to ignore this check.
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/uintcast"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(uintcast.Analyzer)
}
