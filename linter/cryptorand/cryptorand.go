// Static Analyzer to ensure the crypto/rand package is used for randomness
// throughout the codebase.
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/cryptorand"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(cryptorand.Analyzer)
}
