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
)

type Config struct {
	RPC              arbitrum.Config                `koanf:"rpc"`
	Sequencer        SequencerConfig                `koanf:"sequencer"`
	EnableL1Reader   bool                           `koanf:"enable-l1-reader"`
	InboxReader      InboxReaderConfig              `koanf:"inbox-reader"`
	DelayedSequencer DelayedSequencerConfig         `koanf:"delayed-sequencer"`
	BatchPoster      BatchPosterConfig              `koanf:"batch-poster"`
	ForwardingTarget string                         `koanf:"forwarding-target"`
	BlockValidator   validator.BlockValidatorConfig `koanf:"block-validator"`
	Feed             broadcastclient.FeedConfig     `koanf:"feed"`
	Validator        validator.L1ValidatorConfig    `koanf:"validator"`
	SeqCoordinator   SeqCoordinatorConfig           `koanf:"seq-coordinator"`
	DataAvailability das.DataAvailabilityConfig     `koanf:"data-availability"`
}

func ConfigAddOptions(prefix string, f *flag.FlagSet, feedInputEnable bool, feedOutputEnable bool) {
	arbitrum.ConfigAddOptions(prefix+".rpc", f)
	SequencerConfigAddOptions("sequencer", f)
	f.Bool(prefix+".enable-l1-reader", ConfigDefault.EnableL1Reader, "enable l1 reader")
	InboxReaderConfigAddOptions(prefix+".inbox-reader", f)
	DelayedSequencerConfigAddOptions(prefix+".delayed-sequencer", f)
	BatchPosterConfigAddOptions(prefix+".batch-poster", f)
	f.String(prefix+".forwarding-target", ConfigDefault.ForwardingTarget, "transaction forwarding target URL, or \"null\" to disable forwarding (iff not sequencer)")
	validator.BlockValidatorConfigAddOptions(prefix+".block-validator", f)
	broadcastclient.FeedConfigAddOptions(prefix+".feed", f, feedInputEnable, feedOutputEnable)
	validator.L1ValidatorConfigAddOptions(prefix+".validator", f)
	SeqCoordinatorConfigAddOptions(prefix+".seq-coordinator", f)
	das.DataAvailabilityConfigAddOptions(prefix+".data-availability", f)
	// TODO
}

var ConfigDefault = Config{
	RPC:              arbitrum.DefaultConfig,
	Sequencer:        DefaultSequencerConfig,
	EnableL1Reader:   true,
	InboxReader:      DefaultInboxReaderConfig,
	DelayedSequencer: DefaultDelayedSequencerConfig,
	BatchPoster:      DefaultBatchPosterConfig,
	ForwardingTarget: "",
	BlockValidator:   validator.DefaultBlockValidatorConfig,
	Feed:             broadcastclient.FeedConfigDefault,
	Validator:        validator.DefaultL1ValidatorConfig,
	SeqCoordinator:   DefaultSeqCoordinatorConfig,
	DataAvailability: das.DefaultDataAvailabilityConfig,
}

var ConfigDefaultL1Test = Config{
	RPC:              arbitrum.DefaultConfig,
	Sequencer:        DefaultSequencerConfig,
	EnableL1Reader:   true,
	InboxReader:      TestInboxReaderConfig,
	DelayedSequencer: TestDelayedSequencerConfig,
	ForwardingTarget: "",
	BlockValidator:   validator.DefaultBlockValidatorConfig,
	Feed:             broadcastclient.FeedConfigDefault,
	Validator:        validator.DefaultL1ValidatorConfig,
	SeqCoordinator:   DefaultSeqCoordinatorConfig,
	DataAvailability: das.DefaultDataAvailabilityConfig,
}

var ConfigDefaultL2Test = Config{
	RPC:            arbitrum.DefaultConfig,
	Sequencer:      DefaultSequencerConfigL2Test,
	EnableL1Reader: false,
}

type InboxReaderConfig struct {
	DelayBlocks int64         `koanf:"delay-blocks"`
	CheckDelay  time.Duration `koanf:"check-delay"`
	HardReorg   bool          `koanf:"hard-reorg"`
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

type SequencerConfig struct {
	Enable bool `koanf:"enable"`
}

var DefaultSequencerConfig = SequencerConfig{
	Enable: false,
}

var DefaultSequencerConfigL2Test = SequencerConfig{
	Enable: true,
}

func SequencerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultSequencerConfig.Enable, "act and post to l1 as sequencer")
}

type BatchPosterConfig struct {
	Enable           bool          `koanf:"enable"`
	MaxSize          int           `koanf:"max-size"`
	MaxInterval      time.Duration `koanf:"max-interval"`
	PollDelay        time.Duration `koanf:"poll-delay"`
	ErrorDelay       time.Duration `koanf:"error-delay"`
	CompressionLevel int           `koanf:"compression-level"`
}

func BatchPosterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBatchPosterConfig.Enable, "enable posting batches to l1")
	f.Int(prefix+".max-size", DefaultBatchPosterConfig.MaxSize, "maximum batch size")
	f.Duration(prefix+".max-interval", DefaultBatchPosterConfig.MaxInterval, "maximum batch posting interval")
	f.Duration(prefix+".poll-delay", DefaultBatchPosterConfig.PollDelay, "how long to delay after successfully posting batch")
	f.Duration(prefix+".error-delay", DefaultBatchPosterConfig.ErrorDelay, "how long to delay after error posting batch")
	f.Int(prefix+".compression-level", DefaultBatchPosterConfig.CompressionLevel, "batch compression level")
}

var DefaultBatchPosterConfig = BatchPosterConfig{
	Enable:           false,
	MaxSize:          500,
	PollDelay:        time.Second,
	ErrorDelay:       time.Second * 5,
	MaxInterval:      time.Minute,
	CompressionLevel: brotli.DefaultCompression,
}

var TestBatchPosterConfig = BatchPosterConfig{
	MaxSize:          10000,
	PollDelay:        time.Millisecond * 10,
	ErrorDelay:       time.Millisecond * 10,
	MaxInterval:      0,
	CompressionLevel: 2,
}
