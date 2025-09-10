/* Copyright 2023-2025, Offchain Labs, Inc.
   For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// want package:"{package a .*structinit/a.* .*}"
*/

// The comment above ensures that during tests, the `structinit` analyzer
// will produce a `Fact` about the `structinit/a` package (with some prefix
// and suffix in its path). Since the fact will be of type `*accumulatedFieldCounts`,
// we just match arbitrary pattern (.*) - it will be just some address.
// For a reference, see: https://pkg.go.dev/golang.org/x/tools/go/analysis/analysistest#Run

package a

// lint:require-exhaustive-initialization
type InterestingStruct struct {
	X int
	B *BoringStruct
}

type BoringStruct struct {
	X, Y int
}

func init() {
	_ = &InterestingStruct{ // want `initialized with: 1 of total: 2 fields`
		X: 1,
	}
	_ = InterestingStruct{ // want `initialized with: 1 of total: 2 fields`
		B: nil,
	}
	_ = InterestingStruct{ // Not an error, all fields are initialized.
		X: 1,
		B: nil,
	}
	_ = &BoringStruct{ // Not an error since it's not annotated for the linter.
		X: 1,
	}
}
