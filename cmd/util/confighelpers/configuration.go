// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package confighelpers

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	koanfjson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/providers/s3"
	"github.com/mitchellh/mapstructure"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/cmd/genericconf"
)

var (
	version  = ""
	datetime = ""
	modified = ""
)

func ApplyOverrides(f *flag.FlagSet, k *koanf.Koanf) error {
	// Apply command line options and environment variables
	if err := applyOverrideOverrides(f, k); err != nil {
		return err
	}

	// Load configuration file from S3 if setup
	if len(k.String("conf.s3.secret-key")) != 0 {
		if err := loadS3Variables(k); err != nil {
			return fmt.Errorf("error loading S3 settings: %w", err)
		}

		if err := applyOverrideOverrides(f, k); err != nil {
			return err
		}
	}

	// Local config file overrides S3 config file
	configFiles := k.Strings("conf.file")
	for _, configFile := range configFiles {
		if len(configFile) > 0 {
			if err := k.Load(file.Provider(configFile), json.Parser()); err != nil {
				return fmt.Errorf("error loading local config file: %w", err)
			}

			if err := applyOverrideOverrides(f, k); err != nil {
				return err
			}
		}
	}

	return nil
}

// applyOverrideOverrides for configuration values that need to be re-applied for each configuration item applied
func applyOverrideOverrides(f *flag.FlagSet, k *koanf.Koanf) error {
	// Command line overrides config file or config string
	if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		return fmt.Errorf("error loading command line config: %w", err)
	}

	// Config string overrides any config file
	configString := k.String("conf.string")
	if len(configString) > 0 {
		if err := k.Load(rawbytes.Provider([]byte(configString)), json.Parser()); err != nil {
			return fmt.Errorf("error loading config string config: %w", err)
		}

		// Command line overrides config file or config string
		if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
			return fmt.Errorf("error loading command line config: %w", err)
		}
	}

	// Environment variables overrides config files or command line options
	if err := loadEnvironmentVariables(k); err != nil {
		return fmt.Errorf("error loading environment variables: %w", err)
	}

	return nil
}

var envvarsToSplitOnComma map[string]any = map[string]any{
	"auth.api":                              struct{}{},
	"auth.origins":                          struct{}{},
	"chain.info-files":                      struct{}{},
	"conf.file":                             struct{}{},
	"execution.secondary-forwarding-target": struct{}{},
	"execution.sequencer.sender-whitelist":  struct{}{},
	"graphql.corsdomain":                    struct{}{},
	"graphql.vhosts":                        struct{}{},
	"http.api":                              struct{}{},
	"http.corsdomain":                       struct{}{},
	"http.vhosts":                           struct{}{},
	"node.batch-poster.data-poster.blob-tx-replacement-times":        time.Duration(0),
	"node.batch-poster.data-poster.replacement-times":                time.Duration(0),
	"node.data-availability.rest-aggregator.urls":                    struct{}{},
	"node.feed.input.secondary-url":                                  struct{}{},
	"node.feed.input.url":                                            struct{}{},
	"node.feed.input.verify.allowed-addresses":                       struct{}{},
	"node.seq-coordinator.signer.ecdsa.allowed-addresses":            struct{}{},
	"node.staker.batch-poster.data-poster.blob-tx-replacement-times": time.Duration(0),
	"node.staker.batch-poster.data-poster.replacement-times":         time.Duration(0),
	"p2p.bootnodes":       struct{}{},
	"p2p.bootnodes-v5":    struct{}{},
	"validation.api-auth": struct{}{},
	"validation.arbitrator.redis-validation-server-config.module-roots": struct{}{},
	"validation.wasm.allowed-wasm-module-roots":                         struct{}{},
	"ws.api":     struct{}{},
	"ws.origins": struct{}{},
}

func loadEnvironmentVariables(k *koanf.Koanf) error {
	envPrefix := k.String("conf.env-prefix")
	if len(envPrefix) != 0 {
		return k.Load(env.ProviderWithValue(envPrefix+"_", ".", func(key string, v string) (string, interface{}) {
			// FOO__BAR -> foo-bar to handle dash in config names
			key = strings.ReplaceAll(strings.ToLower(
				strings.TrimPrefix(key, envPrefix+"_")), "__", "-")
			key = strings.ReplaceAll(key, "_", ".")

			if value, found := envvarsToSplitOnComma[key]; found {
				// If there are commas in the value, split the value into a slice.
				if _, ok := value.(time.Duration); ok {
					// Special case for time.Duration
					// v[1:len(v)-1] removes the '[' , ']' around the string
					durationStrings := strings.Split(v[1:len(v)-1], ",")
					var durations []time.Duration
					for _, durationString := range durationStrings {
						duration, err := time.ParseDuration(durationString)
						if err != nil {
							return key, nil
						}
						durations = append(durations, duration)
					}
					return key, durations
				}
				if strings.Contains(v, ",") {
					return key, strings.Split(v, ",")

				}
			}

			return key, v
		}), nil)
	}

	return nil
}

func loadS3Variables(k *koanf.Koanf) error {
	return k.Load(s3.Provider(s3.Config{
		AccessKey: k.String("conf.s3.access-key"),
		SecretKey: k.String("conf.s3.secret-key"),
		Region:    k.String("conf.s3.region"),
		Bucket:    k.String("conf.s3.bucket"),
		ObjectKey: k.String("conf.s3.object-key"),
	}), nil)
}

var ErrVersion = errors.New("configuration: version requested")

func GetVersion() (string, string, string) {
	return genericconf.GetVersion(version, datetime, modified)
}

func PrintErrorAndExit(err error, usage func(string)) {
	vcsRevision, _, vcsTime := GetVersion()
	fmt.Printf("Version: %v, time: %v\n", vcsRevision, vcsTime)
	if err != nil && errors.Is(err, ErrVersion) {
		// Already printed version, just exit
		os.Exit(0)
	}
	usage(os.Args[0])
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Printf("\nFatal configuration error: %s\n", err.Error())
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func devFlagArgs() []string {
	args := []string{
		"--init.dev-init",
		"--init.dev-init-address", "0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E",
		"--node.dangerous.no-l1-listener",
		"--node.parent-chain-reader.enable=false",
		"--parent-chain.id=1337",
		"--chain.id=412346",
		"--persistent.chain", "/tmp/dev-test",
		"--node.sequencer",
		"--execution.sequencer.enable",
		"--node.dangerous.no-sequencer-coordinator",
		"--node.staker.enable=false",
		"--init.empty=false",
		"--http.port", "8547",
		"--http.addr", "127.0.0.1",
	}
	return args
}

func BeginCommonParse(f *flag.FlagSet, args []string) (*koanf.Koanf, error) {
	for _, arg := range args {
		if arg == "--version" || arg == "-v" {
			return nil, ErrVersion
		} else if arg == "--dev" {
			args = devFlagArgs()
			break
		}
	}
	if err := f.Parse(args); err != nil {
		return nil, err
	}

	if f.NArg() != 0 {
		// Unexpected number of parameters
		return nil, fmt.Errorf("unexpected parameter: %s", f.Arg(0))
	}

	var k = koanf.New(".")

	// Initial application of command line parameters and environment variables so other methods can be applied
	if err := ApplyOverrides(f, k); err != nil {
		return nil, err
	}

	return k, nil
}

func EndCommonParse(k *koanf.Koanf, config interface{}) error {
	decoderConfig := mapstructure.DecoderConfig{
		ErrorUnused: true,

		// Default values
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc()),
		Metadata:         nil,
		Result:           config,
		WeaklyTypedInput: true,
	}
	err := k.UnmarshalWithConf("", config, koanf.UnmarshalConf{DecoderConfig: &decoderConfig})
	if err != nil {
		return err
	}

	return nil
}

func DumpConfig(k *koanf.Koanf, extraOverrideFields map[string]interface{}) error {
	overrideFields := map[string]interface{}{"conf.dump": false}

	// Don't keep printing configuration file
	for k, v := range extraOverrideFields {
		overrideFields[k] = v
	}

	err := k.Load(confmap.Provider(overrideFields, "."), nil)
	if err != nil {
		return fmt.Errorf("error removing extra parameters before dump: %w", err)
	}

	c, err := k.Marshal(koanfjson.Parser())
	if err != nil {
		return fmt.Errorf("unable to marshal config file to JSON: %w", err)
	}

	fmt.Println(string(c))
	os.Exit(0)
	return fmt.Errorf("Unreachable")
}
