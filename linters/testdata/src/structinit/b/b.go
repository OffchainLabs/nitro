// want package:"{package b .*structinit/b.* .*}"
// The comment above ensures that during tests, the `structinit` analyzer
// will produce a `Fact` about the `structinit/b` package (with some prefix
// and suffix in its path). Since the fact will be of type `*accumulatedFieldCounts`,
// we just match arbitrary pattern (.*) - it will be just some address.
// For a reference, see: https://pkg.go.dev/golang.org/x/tools/go/analysis/analysistest#Run

package b

import (
	"fmt"

	"github.com/offchainlabs/nitro/linters/testdata/src/structinit/a"
)

// lint:require-exhaustive-initialization
type AnotherStruct struct {
	X int
}

func init() {
	var silentlyInitialized = &a.InterestingStruct{} // Error: no field is initialized.
	fmt.Println(silentlyInitialized)
}
