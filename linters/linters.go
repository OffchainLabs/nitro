package main

import (
	"github.com/offchainlabs/nitro/linters/koanf"
	"github.com/offchainlabs/nitro/linters/pointercheck"
	"github.com/offchainlabs/nitro/linters/rightshift"
	"github.com/offchainlabs/nitro/linters/structinit"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		koanf.Analyzer,
		pointercheck.Analyzer,
		rightshift.Analyzer,
		structinit.Analyzer,
	)
}
