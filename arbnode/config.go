package arbnode

import (
	"math/big"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/arbitrum"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type ArbNodeConfig struct {
	ArbConfig              arbitrum.Config
	Sequencer              bool
	L1Reader               bool
	InboxReaderConfig      InboxReaderConfig
	DelayedSequencerConfig DelayedSequencerConfig
	BatchPoster            bool
	BatchPosterConfig      BatchPosterConfig
	ForwardingTarget       string // "" if not forwarding
	BlockValidator         bool
	BlockValidatorConfig   validator.BlockValidatorConfig
	Broadcaster            bool
	BroadcasterConfig      wsbroadcastserver.BroadcasterConfig
	BroadcastClient        bool
	BroadcastClientConfig  broadcastclient.BroadcastClientConfig
	L1Validator            bool
	L1ValidatorConfig      validator.L1ValidatorConfig
	SeqCoordinator         bool
	SeqCoordinatorConfig   SeqCoordinatorConfig
	DataAvailabilityMode   das.DataAvailabilityMode
	DataAvailabilityConfig das.DataAvailabilityConfig
}

func ArbNodeConfigAddOptions(prefix string, f *flag.FlagSet) {
	arbitrum.ConfigAddOptions(prefix+".arbconfig", f)
	f.Bool(prefix+".sequencer", ArbNodeConfigDefault.Sequencer, "enable sequencer")
	f.Bool(prefix+".l1-reader", ArbNodeConfigDefault.Sequencer, "enable l1 reader")
	InboxReaderConfigAddOptions(prefix+".inbox-reader", f)
	DelayedSequencerConfigAddOptions(prefix+".delayed-sequencer", f)
	f.Bool(prefix+".batch-poster", ArbNodeConfigDefault.Sequencer, "enable batch poster")
	BatchPosterConfigAddOptions(prefix+".batch_poster", f)
	f.String(prefix+".forwarding-target", ArbNodeConfigDefault.ForwardingTarget, "forwarding target")
	f.Bool(prefix+".block-validator", ArbNodeConfigDefault.BlockValidator, "enable block validator")
	validator.BlockValidatorConfigAddOptions(prefix+".block-validator", f)
	broadcastclient.FeedConfigAddOptions(prefix+".feed", f)
	// TODO
}

var ArbNodeConfigDefault = ArbNodeConfig{
	ArbConfig:              arbitrum.DefaultConfig,
	Sequencer:              false,
	L1Reader:               true,
	InboxReaderConfig:      DefaultInboxReaderConfig,
	DelayedSequencerConfig: DefaultDelayedSequencerConfig,
	BatchPoster:            true,
	BatchPosterConfig:      DefaultBatchPosterConfig,
	ForwardingTarget:       "",
	BlockValidator:         false,
	BlockValidatorConfig:   validator.DefaultBlockValidatorConfig,
	Broadcaster:            false,
	BroadcasterConfig:      wsbroadcastserver.DefaultBroadcasterConfig,
	BroadcastClient:        false,
	BroadcastClientConfig:  broadcastclient.DefaultBroadcastClientConfig,
	L1Validator:            false,
	L1ValidatorConfig:      validator.DefaultL1ValidatorConfig,
	SeqCoordinator:         false,
	SeqCoordinatorConfig:   DefaultSeqCoordinatorConfig,
	DataAvailabilityMode:   das.OnchainDataAvailability,
	DataAvailabilityConfig: das.DefaultDataAvailabilityConfig,
}

var NodeConfigL1Test = ArbNodeConfig{
	ArbConfig:              arbitrum.DefaultConfig,
	Sequencer:              true,
	L1Reader:               true,
	InboxReaderConfig:      TestInboxReaderConfig,
	DelayedSequencerConfig: TestDelayedSequencerConfig,
	BatchPoster:            true,
	BatchPosterConfig:      TestBatchPosterConfig,
	ForwardingTarget:       "",
	BlockValidator:         false,
	BlockValidatorConfig:   validator.DefaultBlockValidatorConfig,
	Broadcaster:            false,
	BroadcasterConfig:      wsbroadcastserver.DefaultBroadcasterConfig,
	BroadcastClient:        false,
	BroadcastClientConfig:  broadcastclient.DefaultBroadcastClientConfig,
	L1Validator:            false,
	L1ValidatorConfig:      validator.DefaultL1ValidatorConfig,
	SeqCoordinator:         false,
	SeqCoordinatorConfig:   DefaultSeqCoordinatorConfig,
	DataAvailabilityMode:   das.OnchainDataAvailability,
	DataAvailabilityConfig: das.DefaultDataAvailabilityConfig,
}

var NodeConfigL2Test = ArbNodeConfig{
	ArbConfig: arbitrum.DefaultConfig,
	Sequencer: true,
	L1Reader:  false,
}

type InboxReaderConfig struct {
	DelayBlocks int64
	CheckDelay  time.Duration
	HardReorg   bool // erase future transactions in addition to overwriting existing ones
}

func InboxReaderConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int64(prefix+".delay-blocks", DefaultInboxReaderConfig.DelayBlocks, "number of latest blocks to ignore to reduce reorgs")
	f.Duration(prefix+".check-delay", DefaultInboxReaderConfig.CheckDelay, "how long to wait between inbox checks")
	f.Bool(prefix+".hard-reorg", DefaultInboxReaderConfig.HardReorg, "erase future transactions in addition to overwriting existing ones on reorg")
}

var DefaultInboxReaderConfig = InboxReaderConfig{
	DelayBlocks: 4,
	CheckDelay:  2 * time.Second,
	HardReorg:   true,
}

var TestInboxReaderConfig = InboxReaderConfig{
	DelayBlocks: 0,
	CheckDelay:  time.Millisecond * 10,
	HardReorg:   true,
}

type DelayedSequencerConfig struct {
	FinalizeDistance *big.Int      `koanf:"finalize-distance"`
	BlocksAggregate  *big.Int      `koanf:"blocks-aggregate"`
	TimeAggregate    time.Duration `koanf:"time-aggregate"`
}

func DelayedSequencerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int64(prefix+".finalize-distance", DefaultDelayedSequencerConfig.FinalizeDistance.Int64(), "how many blocks in the past L1 block is considered final")
	f.Int64(prefix+".blocks-aggregate", DefaultDelayedSequencerConfig.BlocksAggregate.Int64(), "how many blocks we aggregate looking for delayedMessage")
	f.Duration(prefix+".time-aggregate", DefaultDelayedSequencerConfig.TimeAggregate, "how many blocks we aggregate looking for delayedMessages")
}

var DefaultDelayedSequencerConfig = DelayedSequencerConfig{
	FinalizeDistance: big.NewInt(12),
	BlocksAggregate:  big.NewInt(5),
	TimeAggregate:    time.Minute,
}

var TestDelayedSequencerConfig = DelayedSequencerConfig{
	FinalizeDistance: big.NewInt(12),
	BlocksAggregate:  big.NewInt(5),
	TimeAggregate:    time.Second,
}

type BatchPosterConfig struct {
	MaxBatchSize         int           `koanf:"max-batch-size"`
	MaxBatchPostInterval time.Duration `koanf:"max-batch-post-interval"`
	BatchPollDelay       time.Duration `koanf:"batch-poll-delay"`
	PostingErrorDelay    time.Duration `koanf:"posting-error-delay"`
	CompressionLevel     int           `koanf:"compression-level"`
}

func BatchPosterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".max-batch-size", DefaultBatchPosterConfig.MaxBatchSize, "maximum batch size")
	f.Duration(prefix+".max-batch-post-interval", DefaultBatchPosterConfig.MaxBatchPostInterval, "maximum batch posting interval")
	f.Duration(prefix+".batch-poll-delay", DefaultBatchPosterConfig.BatchPollDelay, "how long to delay after successfully posting batch")
	f.Duration(prefix+".posting-error-delay", DefaultBatchPosterConfig.PostingErrorDelay, "how long to delay after error posting batch")
	f.Int(prefix+".compression-level", DefaultBatchPosterConfig.CompressionLevel, "batch compression level")
}

var DefaultBatchPosterConfig = BatchPosterConfig{
	MaxBatchSize:         500,
	BatchPollDelay:       time.Second,
	PostingErrorDelay:    time.Second * 5,
	MaxBatchPostInterval: time.Minute,
	CompressionLevel:     brotli.DefaultCompression,
}

var TestBatchPosterConfig = BatchPosterConfig{
	MaxBatchSize:         10000,
	BatchPollDelay:       time.Millisecond * 10,
	PostingErrorDelay:    time.Millisecond * 10,
	MaxBatchPostInterval: 0,
	CompressionLevel:     2,
}
