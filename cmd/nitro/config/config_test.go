// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/r3labs/diff/v3"
	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/daprovider/anytrust"
	"github.com/offchainlabs/nitro/nitroversion"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestEmptyCliConfig(t *testing.T) {
	f := pflag.NewFlagSet("", pflag.ContinueOnError)
	NodeConfigAddOptions(f)
	k, err := confighelpers.BeginCommonParse(f, []string{})
	Require(t, err)
	err = anytrust.FixKeysetCLIParsing("node.data-availability.rpc-aggregator.backends", k)
	Require(t, err)
	err = anytrust.FixKeysetCLIParsing("node.da.anytrust.rpc-aggregator.backends", k)
	Require(t, err)
	err = arbnode.FixCompressionLevelsCLIParsing("node.batch-poster.compression-levels", k)
	Require(t, err)
	var emptyCliNodeConfig NodeConfig
	err = confighelpers.EndCommonParse(k, &emptyCliNodeConfig)
	Require(t, err)
	if !reflect.DeepEqual(emptyCliNodeConfig, NodeConfigDefault) {
		changelog, err := diff.Diff(emptyCliNodeConfig, NodeConfigDefault)
		Require(t, err)
		Fail(t, "empty cli config differs from expected default", changelog)
	}
}

func TestSeqConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.transaction-streamer.track-block-metadata-from=10", " ")
	_, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestUnsafeStakerConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.staker.parent-chain-wallet.pathname /l1keystore --node.staker.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.staker.enable --node.staker.strategy MakeNodes --node.staker.staker-interval 10s --execution.forwarding-target null --node.staker.dangerous.without-block-validator", " ")
	_, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

const validatorArgs = "--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.staker.parent-chain-wallet.pathname /l1keystore --node.staker.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.staker.enable --node.staker.strategy MakeNodes --node.staker.staker-interval 10s --execution.forwarding-target null"

func TestValidatorConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.staker.parent-chain-wallet.pathname /l1keystore --node.staker.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.staker.enable --node.staker.strategy MakeNodes --node.staker.staker-interval 10s --execution.forwarding-target null", " ")
	_, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestInvalidCachingStateSchemeForValidator(t *testing.T) {
	validatorArgsWithPathScheme := fmt.Sprintf("%s --execution.caching.state-scheme path", validatorArgs)
	args := strings.Split(validatorArgsWithPathScheme, " ")
	_, _, err := ParseNode(context.Background(), args)
	if !strings.Contains(err.Error(), "path cannot be used as execution.caching.state-scheme when validator is required") {
		Fail(t, "failed to detect invalid state scheme for validator")
	}
}

// TestAggregatorConfig tests the deprecated --node.data-availability.* flags
// to ensure backward compatibility. These flags are deprecated in favor of
// --node.da.anytrust.* but must continue to work.
func TestAggregatorConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.data-availability.enable --node.data-availability.rpc-aggregator.backends [{\"url\":\"http://localhost:8547\",\"pubkey\":\"abc==\"}] --node.transaction-streamer.track-block-metadata-from=10", " ")
	nodeConfig, _, err := ParseNode(context.Background(), args)
	Require(t, err)
	// Verify migration copied config to new location
	if !nodeConfig.Node.DA.AnyTrust.Enable {
		Fail(t, "deprecated --node.data-availability.enable should migrate to Node.DA.AnyTrust.Enable")
	}
	if len(nodeConfig.Node.DA.AnyTrust.RPCAggregator.Backends) != 1 {
		Fail(t, "deprecated --node.data-availability.rpc-aggregator.backends should migrate to Node.DA.AnyTrust.RPCAggregator.Backends")
	}
}

// TestAggregatorConfigNewFlags tests the new --node.da.anytrust.* flags
func TestAggregatorConfigNewFlags(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.da.anytrust.enable --node.da.anytrust.rpc-aggregator.backends [{\"url\":\"http://localhost:8547\",\"pubkey\":\"abc==\"}] --node.transaction-streamer.track-block-metadata-from=10", " ")
	nodeConfig, _, err := ParseNode(context.Background(), args)
	Require(t, err)
	if !nodeConfig.Node.DA.AnyTrust.Enable {
		Fail(t, "--node.da.anytrust.enable should set Node.DA.AnyTrust.Enable")
	}
	if len(nodeConfig.Node.DA.AnyTrust.RPCAggregator.Backends) != 1 {
		Fail(t, "--node.da.anytrust.rpc-aggregator.backends should set Node.DA.AnyTrust.RPCAggregator.Backends")
	}
}

func TestExternalProviderSingularConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.da.external-provider.rpc.url http://localhost:8547 --node.da.external-provider.with-writer=true --node.transaction-streamer.track-block-metadata-from=10", " ")
	_, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestCompressionLevelsConfig(t *testing.T) {
	args := strings.Split(`--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.batch-poster.compression-levels [{"backlog":0,"level":11,"recompression-level":11},{"backlog":30,"level":6,"recompression-level":11}] --node.transaction-streamer.track-block-metadata-from=10`, " ")
	_, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestCompressionLevelsConflict(t *testing.T) {
	args := strings.Split(`--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.batch-poster.compression-level 8 --node.batch-poster.compression-levels [{"backlog":0,"level":11,"recompression-level":11}] --node.transaction-streamer.track-block-metadata-from=10`, " ")
	_, _, err := ParseNode(context.Background(), args)
	if err == nil {
		Fail(t, "expected error when both compression-level and compression-levels are specified")
	}
	if !strings.Contains(err.Error(), "cannot specify both compression-level (deprecated) and compression-levels") {
		Fail(t, "error message should mention conflict between deprecated and new config, got:", err.Error())
	}
}

func TestGenesisJsonFileDirectoryClearsDefaultEmptyInit(t *testing.T) {
	chainId := uint64(42170)
	tempDir := t.TempDir()
	genesisFile := filepath.Join(tempDir, fmt.Sprintf("%d.json", chainId))
	Require(t, os.WriteFile(genesisFile, []byte("{}"), 0600))

	args := strings.Split(fmt.Sprintf("--persistent.chain /tmp/data --chain.id %d --init.genesis-json-file-directory %s", chainId, tempDir), " ")
	nodeConfig, _, err := ParseNode(context.Background(), args)
	Require(t, err)

	if nodeConfig.Init.GenesisJsonFile != genesisFile {
		Fail(t, "expected genesis file from directory", genesisFile, "got", nodeConfig.Init.GenesisJsonFile)
	}
	if nodeConfig.Init.Empty {
		Fail(t, "expected genesis file from directory to disable empty init")
	}
}

func TestConfigVersionRange(t *testing.T) {
	type provenance struct {
		Tag    string
		Branch string
		Commit string
	}
	makeVersion := func(p provenance) nitroversion.Version {
		var timestamp string
		if p.Tag != "" {
			timestamp = "2026-01-02T03:04:05Z"
		}
		version, err := nitroversion.New(nitroversion.Provenance{
			Tag:       p.Tag,
			Branch:    p.Branch,
			Commit:    p.Commit,
			Timestamp: timestamp,
		})
		Require(t, err)
		return version
	}

	for _, tc := range []struct {
		name       string
		version    nitroversion.Version
		jsonConfig string
		// Empty error expectations mean the configuration is expected to parse.
		wantErr        error
		wantErrMessage string
	}{
		//////////////////////////////////////////////////////////////////////////////////////////////////////////
		// excluded from version checks - builds without a SemVer tag have no ordered version to enforce //
		//////////////////////////////////////////////////////////////////////////////////////////////////////////
		{
			name:       "excluded from version checks - local build without provenance",
			jsonConfig: `{"conf":{"min-version":"v3.9.9", "max-version":"v3.9.9"}}`,
		},
		{
			name:       "excluded from version checks - untagged ci build is not checked",
			version:    makeVersion(provenance{Branch: "dev"}),
			jsonConfig: `{"conf":{"min-version":"v3.9.9", "max-version":"v3.9.9"}}`,
		},
		{
			name:       "excluded from version checks - untagged branch build is not checked",
			version:    makeVersion(provenance{Branch: "branch.name"}),
			jsonConfig: `{"conf":{"min-version":"v3.9.9", "max-version":"v3.9.9"}}`,
		},
		{
			name:       "excluded from version checks - semver-looking branch remains untagged",
			version:    makeVersion(provenance{Branch: "v3.9.9"}),
			jsonConfig: `{"conf":{"max-version":"v3.9.8"}}`,
		},
		{
			name:       "excluded from version checks - consensus tag is not semver-tagged",
			version:    makeVersion(provenance{Tag: "consensus-v61"}),
			jsonConfig: `{"conf":{"min-version":"v99.0.0"}}`,
		},
		{
			name:       "excluded from version checks - non-semver private tag is not semver-tagged",
			version:    makeVersion(provenance{Tag: "v3.11.x-private-patches-1"}),
			jsonConfig: `{"conf":{"min-version":"v99.0.0"}}`,
		},
		//////////////////////////
		// version check passes //
		//////////////////////////
		{
			name:       "version check passes - no version fields present",
			version:    makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig: `{"conf":{}}`,
		},
		{
			name:           "version check passes - unknown key is reported",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"min-version":"v3.9.0", "totally-unknown-key":true}}`,
			wantErrMessage: "invalid keys",
		},
		{
			name:       "version check passes - commit is metadata and does not affect ordering",
			version:    makeVersion(provenance{Tag: "v3.9.9", Commit: "26b4b9b"}),
			jsonConfig: `{"conf":{"min-version":"v3.9.9","max-version":"v3.9.9"}}`,
		},
		{
			name:       "version check passes - simple",
			version:    makeVersion(provenance{Tag: "v3.9.3"}),
			jsonConfig: `{"conf":{"min-version":"v3.9.0","max-version":"v3.10.0"}}`,
		},
		{
			name:       "version check passes - simple min",
			version:    makeVersion(provenance{Tag: "v3.9.0"}),
			jsonConfig: `{"conf":{"min-version":"v3.9.0","max-version":"v3.10.0"}}`,
		},
		{
			name:       "version check passes - simple max",
			version:    makeVersion(provenance{Tag: "v3.10.0"}),
			jsonConfig: `{"conf":{"min-version":"v3.9.0","max-version":"v3.10.0"}}`,
		},
		{
			name:       "version check passes - single",
			version:    makeVersion(provenance{Tag: "v3.9.3"}),
			jsonConfig: `{"conf":{"min-version":"v3.9.3","max-version":"v3.9.3"}}`,
		},
		{
			name:       "version check passes - just max",
			version:    makeVersion(provenance{Tag: "v2.9.3"}),
			jsonConfig: `{"conf":{"max-version":"v3.9.3"}}`,
		},
		{
			name:       "version check passes - just min",
			version:    makeVersion(provenance{Tag: "v4.9.3"}),
			jsonConfig: `{"conf":{"min-version":"v3.9.3"}}`,
		},
		{
			name:       "version check passes - with prerelease",
			version:    makeVersion(provenance{Tag: "v3.8.1-rc.1"}),
			jsonConfig: `{"conf":{"min-version":"v3.8.0","max-version":"v3.10.0"}}`,
		},
		{
			name:       "version check passes - prerelease equals min-version",
			version:    makeVersion(provenance{Tag: "v3.8.1-rc.2"}),
			jsonConfig: `{"conf":{"min-version":"v3.8.1-rc.2"}}`,
		},
		{
			name:       "version check passes - prerelease later than min-version",
			version:    makeVersion(provenance{Tag: "v3.8.1-rc.3"}),
			jsonConfig: `{"conf":{"min-version":"v3.8.1-rc.2"}}`,
		},
		{
			name:       "version check passes - final release later than prerelease min-version",
			version:    makeVersion(provenance{Tag: "v3.8.1"}),
			jsonConfig: `{"conf":{"min-version":"v3.8.1-rc.2"}}`,
		},
		{
			name:       "version check passes - prerelease equals max-version",
			version:    makeVersion(provenance{Tag: "v3.8.1-rc.2"}),
			jsonConfig: `{"conf":{"max-version":"v3.8.1-rc.2"}}`,
		},
		{
			name:       "version check passes - prerelease earlier than max-version",
			version:    makeVersion(provenance{Tag: "v3.8.1-rc.1"}),
			jsonConfig: `{"conf":{"max-version":"v3.8.1-rc.2"}}`,
		},
		{
			name:       "version check passes - private derivative is between its base and next prerelease",
			version:    makeVersion(provenance{Tag: "v3.12.0-dev.1.private.2"}),
			jsonConfig: `{"conf":{"min-version":"v3.12.0-dev.1","max-version":"v3.12.0-dev.2"}}`,
		},
		/////////////////////////
		// version check fails //
		/////////////////////////
		{
			name:           "version check fails - tagged build is subject to bounds",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"max-version":"v3.9.8"}}`,
			wantErr:        nitroversion.ErrUnsupportedVersion,
			wantErrMessage: "conf.max-version",
		},
		{
			name:           "version check fails - even with unknown key present",
			version:        makeVersion(provenance{Tag: "v3.10.1"}),
			jsonConfig:     `{"conf":{"min-version":"v3.9.0","max-version":"v3.10.0","totally-unknown-key":true}}`,
			wantErr:        nitroversion.ErrUnsupportedVersion,
			wantErrMessage: "conf.max-version",
		},
		{
			name:           "version check fails - prerelease is before final min-version",
			version:        makeVersion(provenance{Tag: "v3.9.0-rc.2"}),
			jsonConfig:     `{"conf":{"min-version":"v3.9.0","max-version":"v3.10.0"}}`,
			wantErr:        nitroversion.ErrUnsupportedVersion,
			wantErrMessage: "conf.min-version",
		},
		{
			name:           "version check fails - prerelease is after max-version",
			version:        makeVersion(provenance{Tag: "v3.10.1-rc.2"}),
			jsonConfig:     `{"conf":{"min-version":"v3.9.0","max-version":"v3.10.0"}}`,
			wantErr:        nitroversion.ErrUnsupportedVersion,
			wantErrMessage: "conf.max-version",
		},
		{
			name:           "version check fails - prerelease earlier than min-version prerelease",
			version:        makeVersion(provenance{Tag: "v3.8.1-rc.1"}),
			jsonConfig:     `{"conf":{"min-version":"v3.8.1-rc.2"}}`,
			wantErr:        nitroversion.ErrUnsupportedVersion,
			wantErrMessage: "conf.min-version",
		},
		{
			name:           "version check fails - prerelease later than max-version prerelease",
			version:        makeVersion(provenance{Tag: "v3.8.1-rc.3"}),
			jsonConfig:     `{"conf":{"max-version":"v3.8.1-rc.2"}}`,
			wantErr:        nitroversion.ErrUnsupportedVersion,
			wantErrMessage: "conf.max-version",
		},
		{
			name:           "version check fails - final release later than max-version prerelease",
			version:        makeVersion(provenance{Tag: "v3.8.1"}),
			jsonConfig:     `{"conf":{"max-version":"v3.8.1-rc.2"}}`,
			wantErr:        nitroversion.ErrUnsupportedVersion,
			wantErrMessage: "conf.max-version",
		},
		{
			name:           "version check fails - newer than max-version",
			version:        makeVersion(provenance{Tag: "v3.9.1"}),
			jsonConfig:     `{"conf":{"max-version":"v3.9.0"}}`,
			wantErr:        nitroversion.ErrUnsupportedVersion,
			wantErrMessage: "conf.max-version",
		},
		/////////////////////////////
		// config versions invalid //
		/////////////////////////////
		{
			name:           "config versions invalid - invalid version bounds - min-version - short syntax not allowed",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"min-version":"v3.9"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.min-version",
		},
		{
			name:           "config versions invalid - invalid version bounds - max-version - short syntax not allowed",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"max-version":"v3.9"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.max-version",
		},
		{
			name:           "config versions invalid - min-version requires leading v",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"min-version":"3.9.0"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.min-version",
		},
		{
			name:           "config versions invalid - invalid version bounds - min-version",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"min-version":"abc"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.min-version",
		},
		{
			name:           "config versions invalid - invalid version bounds - max-version",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"max-version":"def"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.max-version",
		},
		{
			name:           "config versions invalid - build metadata not allowed in min-version",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"min-version":"v3.9.0+build"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.min-version",
		},
		{
			name:           "config versions invalid - build metadata not allowed in max-version",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"max-version":"v3.10.0+build"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.max-version",
		},
		{
			name:           "config versions invalid - unsupported alpha prerelease",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"min-version":"v3.10.0-alpha.1"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.min-version",
		},
		{
			name:           "config versions invalid - legacy commit suffix",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"max-version":"v3.10.0-rc.1-abcdef"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.max-version",
		},
		{
			name:           "config versions invalid - min-version is greater than max-version",
			version:        makeVersion(provenance{Tag: "v3.9.9"}),
			jsonConfig:     `{"conf":{"min-version":"v3.10.0","max-version":"v3.9.0"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "conf.min-version v3.10.0 is greater than conf.max-version v3.9.0",
		},
		{
			name:           "config versions invalid - reversed bounds are validated for untagged build",
			version:        makeVersion(provenance{Branch: "dev"}),
			jsonConfig:     `{"conf":{"min-version":"v3.10.0","max-version":"v3.9.0"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "conf.min-version v3.10.0 is greater than conf.max-version v3.9.0",
		},
		{
			name:           "config versions invalid - malformed bound is validated for untagged build",
			version:        makeVersion(provenance{Branch: "dev"}),
			jsonConfig:     `{"conf":{"min-version":"not-semver"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.min-version",
		},
		{
			name:           "config versions invalid - reversed bounds are validated for consensus tag",
			version:        makeVersion(provenance{Tag: "consensus-v61"}),
			jsonConfig:     `{"conf":{"min-version":"v3.10.0","max-version":"v3.9.0"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "conf.min-version v3.10.0 is greater than conf.max-version v3.9.0",
		},
		{
			name:           "config versions invalid - malformed bound is validated for consensus tag",
			version:        makeVersion(provenance{Tag: "consensus-v61"}),
			jsonConfig:     `{"conf":{"min-version":"not-semver"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.min-version",
		},
		{
			name:           "config versions invalid - malformed bound is validated for local build",
			jsonConfig:     `{"conf":{"min-version":"not-semver"}}`,
			wantErr:        nitroversion.ErrInvalidVersionRange,
			wantErrMessage: "invalid conf.min-version",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config.json")
			Require(t, WriteToConfigFile(configFile, tc.jsonConfig))

			args := []string{"--persistent.chain", "/tmp/data", "--chain.id", "421613", "--conf.file", configFile}
			_, _, err := ParseNodeWithVersion(context.Background(), args, tc.version)

			if tc.wantErr == nil && tc.wantErrMessage == "" {
				Require(t, err)
				return
			}
			if err == nil {
				Fail(t, "expected error", tc.wantErr, tc.wantErrMessage)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				Fail(t, "expected error", tc.wantErr, "got:", err)
			}
			if tc.wantErrMessage != "" && !strings.Contains(err.Error(), tc.wantErrMessage) {
				Fail(t, "expected error containing", tc.wantErrMessage, "got:", err.Error())
			}
		})
	}
}

func TestReloads(t *testing.T) {
	var check func(node reflect.Value, cold bool, path string)
	check = func(node reflect.Value, cold bool, path string) {
		if node.Kind() != reflect.Struct {
			return
		}

		for i := 0; i < node.NumField(); i++ {
			hot := node.Type().Field(i).Tag.Get("reload") == "hot"
			dot := path + "." + node.Type().Field(i).Name
			if hot && cold {
				t.Fatalf(
					"Option %v%v%v is reloadable but %v%v%v is not",
					colors.Red, dot, colors.Clear,
					colors.Red, path, colors.Clear,
				)
			}
			if hot {
				colors.PrintBlue(dot)
			}
			check(node.Field(i), !hot, dot)
		}
	}

	config := NodeConfigDefault
	update := NodeConfigDefault
	update.Node.BatchPoster.MaxCalldataBatchSize++

	check(reflect.ValueOf(config), false, "config")
	Require(t, config.CanReload(&config))
	Require(t, config.CanReload(&update))

	testUnsafe := func() {
		t.Helper()
		if config.CanReload(&update) == nil {
			Fail(t, "failed to detect unsafe reload")
		}
		update = NodeConfigDefault
	}

	// check that non-reloadable fields fail assignment
	update.Metrics = !update.Metrics
	testUnsafe()
	update.ParentChain.ID++
	testUnsafe()
	update.Node.Staker.Enable = !update.Node.Staker.Enable
	testUnsafe()
}

func TestLiveNodeConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a config file
	configFile := filepath.Join(t.TempDir(), "config.json")
	jsonConfig := "{\"chain\":{\"id\":421613}}"
	Require(t, WriteToConfigFile(configFile, jsonConfig))

	args := strings.Split("--file-logging.enable=false --persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.transaction-streamer.track-block-metadata-from=10", " ")
	args = append(args, []string{"--conf.file", configFile}...)
	config, _, err := ParseNode(context.Background(), args)
	Require(t, err)

	liveConfig := genericconf.NewLiveConfig[*NodeConfig](args, config, func(ctx context.Context, args []string) (*NodeConfig, error) {
		nodeConfig, _, err := ParseNode(ctx, args)
		return nodeConfig, err
	})

	// check updating the config
	update := config.ShallowClone()
	expected := config.ShallowClone()
	update.Node.BatchPoster.MaxCalldataBatchSize += 100
	expected.Node.BatchPoster.MaxCalldataBatchSize += 100
	Require(t, liveConfig.Set(update))
	if !reflect.DeepEqual(liveConfig.Get(), expected) {
		Fail(t, "failed to set config")
	}

	// check that an invalid reload gets rejected
	update = config.ShallowClone()
	update.ParentChain.ID++
	if liveConfig.Set(update) == nil {
		Fail(t, "failed to reject invalid update")
	}
	if !reflect.DeepEqual(liveConfig.Get(), expected) {
		Fail(t, "config should not change if its update fails")
	}

	// starting the LiveConfig after testing LiveConfig.set to avoid race condition in the test
	liveConfig.Start(ctx)

	// reload config
	expected = config.ShallowClone()
	Require(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))
	if !PollLiveConfigUntilEqual(liveConfig, expected) {
		Fail(t, "live config differs from expected")
	}

	// check that reloading the config again doesn't change anything
	Require(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))
	time.Sleep(80 * time.Millisecond)
	if !reflect.DeepEqual(liveConfig.Get(), expected) {
		Fail(t, "live config differs from expected")
	}

	// change the config file
	expected = config.ShallowClone()
	expected.Node.BatchPoster.MaxCalldataBatchSize += 100
	jsonConfig = fmt.Sprintf("{\"node\":{\"batch-poster\":{\"max-calldata-batch-size\":\"%d\"}}, \"chain\":{\"id\":421613}}", expected.Node.BatchPoster.MaxCalldataBatchSize)
	Require(t, WriteToConfigFile(configFile, jsonConfig))

	// trigger LiveConfig reload
	Require(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))

	if !PollLiveConfigUntilEqual(liveConfig, expected) {
		Fail(t, "failed to update config", config.Node.BatchPoster.MaxCalldataBatchSize, update.Node.BatchPoster.MaxCalldataBatchSize)
	}

	// change chain.id in the config file (currently non-reloadable)
	jsonConfig = fmt.Sprintf("{\"node\":{\"batch-poster\":{\"max-calldata-batch-size\":\"%d\"}}, \"chain\":{\"id\":421703}}", expected.Node.BatchPoster.MaxCalldataBatchSize)
	Require(t, WriteToConfigFile(configFile, jsonConfig))

	// trigger LiveConfig reload
	Require(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))

	if PollLiveConfigUntilNotEqual(liveConfig, expected) {
		Fail(t, "failed to reject invalid update")
	}
}

func TestPeriodicReloadOfLiveNodeConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create config file with ReloadInterval = 20 ms
	configFile := filepath.Join(t.TempDir(), "config.json")
	jsonConfig := "{\"conf\":{\"reload-interval\":\"20ms\"}}"
	Require(t, WriteToConfigFile(configFile, jsonConfig))

	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.transaction-streamer.track-block-metadata-from=10", " ")
	args = append(args, []string{"--conf.file", configFile}...)
	config, _, err := ParseNode(context.Background(), args)
	Require(t, err)

	liveConfig := genericconf.NewLiveConfig[*NodeConfig](args, config, func(ctx context.Context, args []string) (*NodeConfig, error) {
		nodeConfig, _, err := ParseNode(ctx, args)
		return nodeConfig, err
	})
	liveConfig.Start(ctx)

	// test if periodic reload works
	expected := config.ShallowClone()
	expected.Conf.ReloadInterval = 0
	jsonConfig = "{\"conf\":{\"reload-interval\":\"0\"}}"
	Require(t, WriteToConfigFile(configFile, jsonConfig))
	start := time.Now()
	if !PollLiveConfigUntilEqual(liveConfig, expected) {
		Fail(t, fmt.Sprintf("failed to update config after %d ms, while reload interval is %s", time.Since(start).Milliseconds(), config.Conf.ReloadInterval))
	}

	// test if previous config successfully disabled periodic reload
	expected = config.ShallowClone()
	expected.Conf.ReloadInterval = 10 * time.Millisecond
	jsonConfig = "{\"conf\":{\"reload-interval\":\"10ms\"}}"
	Require(t, WriteToConfigFile(configFile, jsonConfig))
	time.Sleep(80 * time.Millisecond)
	if reflect.DeepEqual(liveConfig.Get(), expected) {
		Fail(t, "failed to disable periodic reload")
	}
}

func WriteToConfigFile(path string, jsonConfig string) error {
	return os.WriteFile(path, []byte(jsonConfig), 0600)
}

func PollLiveConfigUntilEqual(liveConfig *genericconf.LiveConfig[*NodeConfig], expected *NodeConfig) bool {
	return PollLiveConfig(liveConfig, expected, true)
}
func PollLiveConfigUntilNotEqual(liveConfig *genericconf.LiveConfig[*NodeConfig], expected *NodeConfig) bool {
	return PollLiveConfig(liveConfig, expected, false)
}

func PollLiveConfig(liveConfig *genericconf.LiveConfig[*NodeConfig], expected *NodeConfig, equal bool) bool {
	for i := 0; i < 16; i++ {
		if reflect.DeepEqual(liveConfig.Get(), expected) == equal {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
