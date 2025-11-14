package testflag

import (
	"flag"
	"log"
	"os"
)

var (
	fs                                            = flag.NewFlagSet("test", flag.ExitOnError)
	StateSchemeFlag                               = fs.String("test_state_scheme", "", "State scheme to use for tests")
	DatabaseEngineFlag                            = fs.String("test_database_engine", "", "Database engine to use for tests")
	RedisFlag                                     = fs.String("test_redis", "", "Redis URL for testing")
	RecordBlockInputsEnable                       = fs.Bool("recordBlockInputs.enable", false, "Whether to record block inputs as a json file")
	RecordBlockInputsWithSlug                     = fs.String("recordBlockInputs.WithSlug", "", "Slug directory for validationInputsWriter")
	RecordBlockInputsWithBaseDir                  = fs.String("recordBlockInputs.WithBaseDir", "", "Base directory for validationInputsWriter")
	RecordBlockInputsWithTimestampDirEnabled      = fs.Bool("recordBlockInputs.WithTimestampDirEnabled", true, "Whether to add timestamp directory while recording block inputs")
	RecordBlockInputsWithBlockIdInFileNameEnabled = fs.Bool("recordBlockInputs.WithBlockIdInFileNameEnabled", true, "Whether to record block inputs using test specific block_id")
	LogLevelFlag                                  = fs.String("test_loglevel", "", "Log level for tests")
	SeedFlag                                      = fs.String("seed", "", "Seed for random number generator")
	RunsFlag                                      = fs.String("runs", "", "Number of runs for test")
	LoggingFlag                                   = fs.String("logging", "", "Enable logging")
	CompileFlag                                   = fs.String("test_compile", "", "[STORE|LOAD] to allow store/load in compile test")
)

// This is a workaround for the fact that we can only pass flags to the package in which they are defined.
// So to avoid doing that we pass the flags after adding a delimiter "--" to the command line.
// We then parse the arguments only after the delimiter to the flagset.
func init() {
	var args []string
	foundDelimiter := false
	for _, arg := range os.Args {
		if foundDelimiter {
			args = append(args, arg)
		}
		if arg == "--" {
			foundDelimiter = true
		}
	}
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}
}
