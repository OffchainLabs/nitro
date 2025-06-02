package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	lightclient "github.com/EspressoSystems/espresso-network/sdks/go/light-client"

	"github.com/ethereum/go-ethereum/common"
)

func createL1AndL2Node(
	ctx context.Context,
	t *testing.T,
	delayedSequencer bool,
	blobsEnabled bool,
) (*NodeBuilder, func()) {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l1StackConfig.HTTPPort = 8545
	builder.l1StackConfig.WSPort = 8546
	builder.l1StackConfig.HTTPHost = "0.0.0.0"
	builder.l1StackConfig.HTTPVirtualHosts = []string{"*"}
	builder.l1StackConfig.WSHost = "0.0.0.0"
	builder.l1StackConfig.DataDir = t.TempDir()
	builder.l1StackConfig.WSModules = append(builder.l1StackConfig.WSModules, "eth")
	builder.l2StackConfig.HTTPPort = 8945
	builder.l2StackConfig.HTTPHost = "0.0.0.0"
	builder.l2StackConfig.IPCPath = tmpPath(t, "test.ipc")
	builder.useL1StackConfig = true

	// poster config
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.BatchPoster.EspressoTxnsPollingInterval = 2 * time.Second
	builder.nodeConfig.BatchPoster.ErrorDelay = 5 * time.Second
	builder.nodeConfig.BatchPoster.MaxSize = 1000
	builder.nodeConfig.BatchPoster.PollInterval = 10 * time.Second
	builder.nodeConfig.BatchPoster.MaxDelay = -1000 * time.Hour
	builder.nodeConfig.BatchPoster.LightClientAddress = lightClientAddress
	builder.nodeConfig.BatchPoster.HotShotUrls = []string{hotShotUrl, hotShotUrl}
	builder.nodeConfig.BatchPoster.UseEscapeHatch = false
	// validator config
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.ValidationPoll = 2 * time.Second
	builder.nodeConfig.BlockValidator.ValidationServer.URL = fmt.Sprintf("ws://127.0.0.1:%d", arbValidationPort)
	builder.nodeConfig.DelayedSequencer.Enable = delayedSequencer
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1

	// sequencer config
	builder.nodeConfig.Sequencer = true
	builder.nodeConfig.ParentChainReader.Enable = true // This flag is necessary to enable sequencing transactions with espresso behavior
	builder.nodeConfig.ParentChainReader.UseFinalityData = true
	builder.nodeConfig.Dangerous.NoSequencerCoordinator = true
	builder.execConfig.Sequencer.Enable = true
	builder.execConfig.Caching.StateScheme = "hash"
	builder.execConfig.Caching.Archive = true

	if blobsEnabled {
		builder.nodeConfig.BatchPoster.Post4844Blobs = true
		builder.nodeConfig.BatchPoster.IgnoreBlobPrice = true
		builder.withL1 = true
		builder.deployBold = false
		// Enabling this to false because we dont have a blob reader in the tests
		// which is needed for staker
		builder.nodeConfig.BlockValidator.Enable = false
		builder.nodeConfig.Staker.Enable = false

	}

	cleanup := builder.Build(t)

	mnemonic := "indoor dish desk flag debris potato excuse depart ticket judge file exit"
	err := builder.L1Info.GenerateAccountWithMnemonic("CommitmentTask", mnemonic, 5)
	Require(t, err)
	builder.L1.TransferBalance(t, "Faucet", "CommitmentTask", new(big.Int).Mul(big.NewInt(9e18), big.NewInt(1000)), builder.L1Info)

	return builder, cleanup
}

func TestEspressoSovereignSequencer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	valNodeCleanup := createValidationNode(ctx, t, true)
	defer valNodeCleanup()

	builder, cleanup := createL1AndL2Node(ctx, t, true, false)
	defer cleanup()

	err := waitForL1Node(ctx)
	Require(t, err)

	cleanEspresso := runEspresso()
	defer cleanEspresso()

	// wait for the builder
	err = waitForEspressoNode(ctx)
	Require(t, err)

	// create light client reader
	lightClientReader, err := lightclient.NewLightClientReader(common.HexToAddress(lightClientAddress), builder.L1.Client)

	Require(t, err)

	// wait for hotshot liveness
	err = waitForHotShotLiveness(ctx, lightClientReader)
	Require(t, err)
	err = checkTransferTxOnL2(t, ctx, builder.L2, "User14", builder.L2Info)
	Require(t, err)

	msgCnt, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)

	err = waitForWith(ctx, 8*time.Minute, 60*time.Second, func() bool {
		validatedCnt := builder.L2.ConsensusNode.BlockValidator.Validated(t)
		return validatedCnt == msgCnt
	})
	Require(t, err)
}
