package nethexec

import (
	"github.com/offchainlabs/nitro/execution/gethexec"
)

// ExecutionMode controls how the wrapper uses internal vs external EL
type ExecutionMode uint8

const (
	ModeInternalOnly ExecutionMode = iota // default
	ModeDualCompare                       // call both, compare results
	ModeExternalOnly                      // return external only
)

// GetExecutionMode reads from config
func GetExecutionMode(config *gethexec.Config) ExecutionMode {
	switch config.ExecutionMode {
	case "internal", "":
		return ModeInternalOnly
	case "compare", "dual", "both":
		return ModeDualCompare
	case "external", "nethermind":
		return ModeExternalOnly
	default:
		return ModeInternalOnly
	}
}
