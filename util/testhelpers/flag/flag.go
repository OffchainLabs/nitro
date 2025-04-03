package testflag

import (
	"flag"
)

var (
	StateSchemeFlag                               = flag.String("test_state_scheme", "", "State scheme to use for tests")
	RedisFlag                                     = flag.String("test_redis", "", "Redis URL for testing")
	RecordBlockInputsEnable                       = flag.Bool("recordBlockInputs.enable", true, "Whether to record block inputs as a json file")
	RecordBlockInputsWithSlug                     = flag.String("recordBlockInputs.WithSlug", "", "Slug directory for validationInputsWriter")
	RecordBlockInputsWithBaseDir                  = flag.String("recordBlockInputs.WithBaseDir", "", "Base directory for validationInputsWriter")
	RecordBlockInputsWithTimestampDirEnabled      = flag.Bool("recordBlockInputs.WithTimestampDirEnabled", true, "Whether to add timestamp directory while recording block inputs")
	RecordBlockInputsWithBlockIdInFileNameEnabled = flag.Bool("recordBlockInputs.WithBlockIdInFileNameEnabled", true, "Whether to record block inputs using test specific block_id")
	LogLevelFlag                                  = flag.String("test_loglevel", "", "Log level for tests")
	SeedFlag                                      = flag.String("seed", "", "Seed for random number generator")
	RunsFlag                                      = flag.String("runs", "", "Number of runs for test")
	LoggingFlag                                   = flag.String("logging", "", "Enable logging")
	CompileFlag                                   = flag.String("test_compile", "", "[STORE|LOAD] to allow store/load in compile test")
)
