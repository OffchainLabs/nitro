// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package config

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/signature"
)

type DataPosterConfig struct {
	RedisSigner            signature.SimpleHmacConfig `koanf:"redis-signer"`
	ReplacementTimes       []time.Duration            `koanf:"replacement-times"`
	BlobTxReplacementTimes []time.Duration            `koanf:"blob-tx-replacement-times"`
	// This is forcibly disabled if the parent chain is an Arbitrum chain,
	// so you should probably use DataPoster's waitForL1Finality method instead of reading this field directly.
	WaitForL1Finality      bool              `koanf:"wait-for-l1-finality" reload:"hot"`
	MaxMempoolTransactions uint64            `koanf:"max-mempool-transactions" reload:"hot"`
	MaxMempoolWeight       uint64            `koanf:"max-mempool-weight" reload:"hot"`
	MaxQueuedTransactions  int               `koanf:"max-queued-transactions" reload:"hot"`
	TargetPriceGwei        float64           `koanf:"target-price-gwei" reload:"hot"`
	UrgencyGwei            float64           `koanf:"urgency-gwei" reload:"hot"`
	MinTipCapGwei          float64           `koanf:"min-tip-cap-gwei" reload:"hot"`
	MinBlobTxTipCapGwei    float64           `koanf:"min-blob-tx-tip-cap-gwei" reload:"hot"`
	MaxTipCapGwei          float64           `koanf:"max-tip-cap-gwei" reload:"hot"`
	MaxBlobTxTipCapGwei    float64           `koanf:"max-blob-tx-tip-cap-gwei" reload:"hot"`
	MaxFeeBidMultipleBips  arbmath.UBips     `koanf:"max-fee-bid-multiple-bips" reload:"hot"`
	NonceRbfSoftConfs      uint64            `koanf:"nonce-rbf-soft-confs" reload:"hot"`
	Post4844Blobs          bool              `koanf:"post-4844-blobs" reload:"hot"`
	AllocateMempoolBalance bool              `koanf:"allocate-mempool-balance" reload:"hot"`
	UseDBStorage           bool              `koanf:"use-db-storage"`
	UseNoOpStorage         bool              `koanf:"use-noop-storage"`
	LegacyStorageEncoding  bool              `koanf:"legacy-storage-encoding" reload:"hot"`
	Dangerous              DangerousConfig   `koanf:"dangerous"`
	ExternalSigner         ExternalSignerCfg `koanf:"external-signer"`
	MaxFeeCapFormula       string            `koanf:"max-fee-cap-formula" reload:"hot"`
	ElapsedTimeBase        time.Duration     `koanf:"elapsed-time-base" reload:"hot"`
	ElapsedTimeImportance  float64           `koanf:"elapsed-time-importance" reload:"hot"`
	// When set, dataposter will not post new batches, but will keep running to
	// get existing batches confirmed.
	DisableNewTx bool `koanf:"disable-new-tx" reload:"hot"`
}

type ExternalSignerCfg struct {
	// URL of the external signer rpc server, if set this overrides transaction
	// options and uses external signer
	// for signing transactions.
	URL string `koanf:"url"`
	// Hex encoded ethereum address of the external signer.
	Address string `koanf:"address"`
	// API method name (e.g. eth_signTransaction).
	Method string `koanf:"method"`
	// (Optional) Path to the external signer root CA certificate.
	// This allows us to use self-signed certificates on the external signer.
	RootCA string `koanf:"root-ca"`
	// (Optional) Client certificate for mtls.
	ClientCert string `koanf:"client-cert"`
	// (Optional) Client certificate key for mtls.
	// This is required when client-cert is set.
	ClientPrivateKey string `koanf:"client-private-key"`
	// TLS config option, when enabled skips certificate verification of external signer.
	InsecureSkipVerify bool `koanf:"insecure-skip-verify"`
}

func ExternalSignerTestCfg(addr common.Address, url string) (*ExternalSignerCfg, error) {
	cp, err := externalsignertest.CertPaths()
	if err != nil {
		return nil, fmt.Errorf("getting certificates path: %w", err)
	}
	return &ExternalSignerCfg{
		Address:          common.Bytes2Hex(addr.Bytes()),
		URL:              url,
		Method:           externalsignertest.SignerMethod,
		RootCA:           cp.ServerCert,
		ClientCert:       cp.ClientCert,
		ClientPrivateKey: cp.ClientKey,
	}, nil
}

type DangerousConfig struct {
	// This should be used with caution, only when dataposter somehow gets in a
	// bad state, and we require clearing it.
	ClearDBStorage bool `koanf:"clear-dbstorage"`
}

// Validate checks that the DataPosterConfig is valid.
func (c *DataPosterConfig) Validate() error {
	if len(c.ReplacementTimes) == 0 {
		return fmt.Errorf("replacement-times must have at least one value")
	}
	if c.Post4844Blobs && len(c.BlobTxReplacementTimes) == 0 {
		return fmt.Errorf("blob-tx-replacement-times must have at least one value when post-4844-blobs is enabled")
	}
	return nil
}

// ConfigFetcher function type is used instead of directly passing config so
// that flags can be reloaded dynamically.
type ConfigFetcher func() *DataPosterConfig

// DataPosterUsageContext indicates what component is using the DataPoster to determine
// which config options to expose.
type DataPosterUsageContext int

const (
	// DataPosterUsageStaker indicates the DataPoster is being used by the staker/validator.
	// Staker posts small (~250 byte) assertions, so blob options don't make sense.
	// Blob reading is also not supported with assertions.
	DataPosterUsageStaker DataPosterUsageContext = iota
	// DataPosterUsageBatchPoster indicates the DataPoster is being used by the batch poster.
	// Note: The enable flag (post-4844-blobs) is NOT exposed here because batch poster
	// controls that at its own configuration level.
	DataPosterUsageBatchPoster
)

func DataPosterConfigAddOptions(prefix string, f *pflag.FlagSet, defaultDataPosterConfig DataPosterConfig, usageContext DataPosterUsageContext) {
	f.DurationSlice(prefix+".replacement-times", defaultDataPosterConfig.ReplacementTimes, "comma-separated list of durations since first posting to attempt a replace-by-fee")
	f.Bool(prefix+".wait-for-l1-finality", defaultDataPosterConfig.WaitForL1Finality, "only treat a transaction as confirmed after L1 finality has been achieved (recommended)")
	f.Uint64(prefix+".max-mempool-transactions", defaultDataPosterConfig.MaxMempoolTransactions, "the maximum number of transactions to have queued in the mempool at once (0 = unlimited)")
	f.Uint64(prefix+".max-mempool-weight", defaultDataPosterConfig.MaxMempoolWeight, "the maximum number of weight (weight = min(1, tx.blobs)) to have queued in the mempool at once (0 = unlimited)")
	f.Int(prefix+".max-queued-transactions", defaultDataPosterConfig.MaxQueuedTransactions, "the maximum number of unconfirmed transactions to track at once (0 = unlimited)")
	f.Float64(prefix+".target-price-gwei", defaultDataPosterConfig.TargetPriceGwei, "the target price to use for maximum fee cap calculation")
	f.Float64(prefix+".urgency-gwei", defaultDataPosterConfig.UrgencyGwei, "the urgency to use for maximum fee cap calculation")
	f.Float64(prefix+".min-tip-cap-gwei", defaultDataPosterConfig.MinTipCapGwei, "the minimum tip cap to post transactions at")
	f.Float64(prefix+".max-tip-cap-gwei", defaultDataPosterConfig.MaxTipCapGwei, "the maximum tip cap to post transactions at")
	f.Uint64(prefix+".max-fee-bid-multiple-bips", uint64(defaultDataPosterConfig.MaxFeeBidMultipleBips), "the maximum multiple of the current price to bid for a transaction's fees (may be exceeded due to min rbf increase, 0 = unlimited)")
	f.Uint64(prefix+".nonce-rbf-soft-confs", defaultDataPosterConfig.NonceRbfSoftConfs, "the maximum probable reorg depth, used to determine when a transaction will no longer likely need replaced-by-fee")
	f.Bool(prefix+".allocate-mempool-balance", defaultDataPosterConfig.AllocateMempoolBalance, "if true, don't put transactions in the mempool that spend a total greater than the batch poster's balance")
	f.Bool(prefix+".use-db-storage", defaultDataPosterConfig.UseDBStorage, "uses database storage when enabled")
	f.Bool(prefix+".use-noop-storage", defaultDataPosterConfig.UseNoOpStorage, "uses noop storage, it doesn't store anything")
	f.Bool(prefix+".legacy-storage-encoding", defaultDataPosterConfig.LegacyStorageEncoding, "encodes items in a legacy way (as it was before dropping generics)")
	f.String(prefix+".max-fee-cap-formula", defaultDataPosterConfig.MaxFeeCapFormula, "mathematical formula to calculate maximum fee cap gwei the result of which would be float64.\n"+
		"This expression is expected to be evaluated please refer https://github.com/Knetic/govaluate/blob/master/MANUAL.md to find all available mathematical operators.\n"+
		"Currently available variables to construct the formula are BacklogOfBatches, UrgencyGWei, ElapsedTime, ElapsedTimeBase, ElapsedTimeImportance, and TargetPriceGWei")
	f.Duration(prefix+".elapsed-time-base", defaultDataPosterConfig.ElapsedTimeBase, "unit to measure the time elapsed since creation of transaction used for maximum fee cap calculation")
	f.Float64(prefix+".elapsed-time-importance", defaultDataPosterConfig.ElapsedTimeImportance, "weight given to the units of time elapsed used for maximum fee cap calculation")

	signature.SimpleHmacConfigAddOptions(prefix+".redis-signer", f)
	addDangerousOptions(prefix+".dangerous", f)
	addExternalSignerOptions(prefix+".external-signer", f)
	f.Bool(prefix+".disable-new-tx", defaultDataPosterConfig.DisableNewTx, "disable posting new transactions, data poster will still keep confirming existing batches")

	includeBlobTuning := usageContext != DataPosterUsageStaker
	if includeBlobTuning {
		f.DurationSlice(prefix+".blob-tx-replacement-times", defaultDataPosterConfig.BlobTxReplacementTimes, "comma-separated list of durations since first posting a blob transaction to attempt a replace-by-fee")
		f.Float64(prefix+".min-blob-tx-tip-cap-gwei", defaultDataPosterConfig.MinBlobTxTipCapGwei, "the minimum tip cap to post EIP-4844 blob carrying transactions at")
		f.Float64(prefix+".max-blob-tx-tip-cap-gwei", defaultDataPosterConfig.MaxBlobTxTipCapGwei, "the maximum tip cap to post EIP-4844 blob carrying transactions at")
	}

	// We intentionally don't expose an option to configure Post4844Blobs.
	// Components using DataPoster should set it on DataPoster's config themselves.
}

func addDangerousOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".clear-dbstorage", DefaultDataPosterConfig.Dangerous.ClearDBStorage, "clear database storage")
}

func addExternalSignerOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".url", DefaultDataPosterConfig.ExternalSigner.URL, "external signer url")
	f.String(prefix+".address", DefaultDataPosterConfig.ExternalSigner.Address, "external signer address")
	f.String(prefix+".method", DefaultDataPosterConfig.ExternalSigner.Method, "external signer method")
	f.String(prefix+".root-ca", DefaultDataPosterConfig.ExternalSigner.RootCA, "external signer root CA")
	f.String(prefix+".client-cert", DefaultDataPosterConfig.ExternalSigner.ClientCert, "rpc client cert")
	f.String(prefix+".client-private-key", DefaultDataPosterConfig.ExternalSigner.ClientPrivateKey, "rpc client private key")
	f.Bool(prefix+".insecure-skip-verify", DefaultDataPosterConfig.ExternalSigner.InsecureSkipVerify, "skip TLS certificate verification")
}

var DefaultDataPosterConfig = DataPosterConfig{
	ReplacementTimes:       []time.Duration{5 * time.Minute, 10 * time.Minute, 20 * time.Minute, 30 * time.Minute, time.Hour, 2 * time.Hour, 4 * time.Hour, 6 * time.Hour, 8 * time.Hour, 12 * time.Hour, 16 * time.Hour, 18 * time.Hour, 20 * time.Hour, 22 * time.Hour},
	BlobTxReplacementTimes: []time.Duration{5 * time.Minute, 10 * time.Minute, 30 * time.Minute, time.Hour, 4 * time.Hour, 8 * time.Hour, 16 * time.Hour, 22 * time.Hour},
	WaitForL1Finality:      true,
	TargetPriceGwei:        60.,
	UrgencyGwei:            2.,
	MaxMempoolTransactions: 18,
	MaxMempoolWeight:       18,
	MinTipCapGwei:          0.05,
	MinBlobTxTipCapGwei:    1, // default geth minimum, and relays aren't likely to accept lower values given propagation time
	MaxTipCapGwei:          1.2,
	MaxBlobTxTipCapGwei:    1, // lower than normal because 4844 rbf is a minimum of a 2x
	MaxFeeBidMultipleBips:  arbmath.OneInUBips * 10,
	NonceRbfSoftConfs:      1,
	Post4844Blobs:          false,
	AllocateMempoolBalance: true,
	UseDBStorage:           true,
	UseNoOpStorage:         false,
	LegacyStorageEncoding:  false,
	Dangerous:              DangerousConfig{ClearDBStorage: false},
	ExternalSigner:         ExternalSignerCfg{Method: "eth_signTransaction", InsecureSkipVerify: false},
	MaxFeeCapFormula:       "((BacklogOfBatches * UrgencyGWei) ** 2) + ((ElapsedTime/ElapsedTimeBase) ** 2) * ElapsedTimeImportance + TargetPriceGWei",
	ElapsedTimeBase:        10 * time.Minute,
	ElapsedTimeImportance:  10,
	DisableNewTx:           false,
}

var DefaultDataPosterConfigForValidator = func() DataPosterConfig {
	config := DefaultDataPosterConfig
	// the validator cannot queue transactions
	config.MaxMempoolTransactions = 1
	config.MaxMempoolWeight = 1
	// Clear blob-related fields since they're not applicable to validator
	config.BlobTxReplacementTimes = nil
	config.MinBlobTxTipCapGwei = 0
	config.MaxBlobTxTipCapGwei = 0
	return config
}()

var TestDataPosterConfig = DataPosterConfig{
	ReplacementTimes:       []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 20 * time.Second, 30 * time.Second, time.Minute, 5 * time.Minute},
	BlobTxReplacementTimes: []time.Duration{1 * time.Second, 10 * time.Second, 30 * time.Second, 5 * time.Minute},
	RedisSigner:            signature.TestSimpleHmacConfig,
	WaitForL1Finality:      false,
	TargetPriceGwei:        60.,
	UrgencyGwei:            2.,
	MaxMempoolTransactions: 18,
	MaxMempoolWeight:       18,
	MinTipCapGwei:          0.05,
	MinBlobTxTipCapGwei:    1,
	MaxTipCapGwei:          5,
	MaxBlobTxTipCapGwei:    1,
	MaxFeeBidMultipleBips:  arbmath.OneInUBips * 10,
	NonceRbfSoftConfs:      1,
	Post4844Blobs:          false,
	AllocateMempoolBalance: true,
	UseDBStorage:           false,
	UseNoOpStorage:         false,
	LegacyStorageEncoding:  false,
	ExternalSigner:         ExternalSignerCfg{Method: "eth_signTransaction", InsecureSkipVerify: true},
	MaxFeeCapFormula:       "((BacklogOfBatches * UrgencyGWei) ** 2) + ((ElapsedTime/ElapsedTimeBase) ** 2) * ElapsedTimeImportance + TargetPriceGWei",
	ElapsedTimeBase:        10 * time.Minute,
	ElapsedTimeImportance:  10,
	DisableNewTx:           false,
}

var TestDataPosterConfigForValidator = func() DataPosterConfig {
	config := TestDataPosterConfig
	// the validator cannot queue transactions
	config.MaxMempoolTransactions = 1
	config.MaxMempoolWeight = 1
	return config
}()
