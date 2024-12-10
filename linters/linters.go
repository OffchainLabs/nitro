package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/offchainlabs/nitro/linters/koanf"
	"github.com/offchainlabs/nitro/linters/pointercheck"
	"github.com/offchainlabs/nitro/linters/rightshift"
	"github.com/offchainlabs/nitro/linters/structinit"
)

func main() {
	multichecker.Main(
		koanf.Analyzer,
		pointercheck.Analyzer,
		rightshift.Analyzer,
		structinit.Analyzer,
	)
}
