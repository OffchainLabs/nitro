package arbtest

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestTimeboostTransactionSyncActiveSequencer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir := t.TempDir()
	auctionContractAddr, aliceBidderClient, bobBidderClient, roundDuration, builderSeq, cleanupSeq, _, cleanupFeedListener, _ :=
		setupExpressLaneAuction(t, tmpDir, ctx, withFeedListener, 0)
	_, seqClient, seqInfo := builderSeq.L2.ConsensusNode, builderSeq.L2.Client, builderSeq.L2Info
	defer cleanupSeq()
	defer cleanupFeedListener()

	runTimeboostTransactionSyncScenario(
		t,
		ctx,
		auctionContractAddr,
		aliceBidderClient,
		bobBidderClient,
		roundDuration,
		seqClient,
		seqInfo,
		builderSeq.L2.ConsensusNode.Stack.HTTPEndpoint(),
	)
}

func TestTimeboostTransactionSyncNonActiveSequencer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir := t.TempDir()
	auctionContractAddr, aliceBidderClient, bobBidderClient, roundDuration, builderSeq, cleanupSeq, forwarder, cleanupFeedListener, _ :=
		setupExpressLaneAuction(t, tmpDir, ctx, withForwardingSeq, 0)
	_, seqClient, seqInfo := builderSeq.L2.ConsensusNode, builderSeq.L2.Client, builderSeq.L2Info
	defer cleanupSeq()
	defer cleanupFeedListener()

	runTimeboostTransactionSyncScenario(
		t,
		ctx,
		auctionContractAddr,
		aliceBidderClient,
		bobBidderClient,
		roundDuration,
		seqClient,
		seqInfo,
		forwarder.ConsensusNode.Stack.HTTPEndpoint(),
	)
}

func runTimeboostTransactionSyncScenario(
	t *testing.T,
	ctx context.Context,
	auctionContractAddr common.Address,
	aliceBidderClient, bobBidderClient *timeboost.BidderClient,
	roundDuration time.Duration,
	seqClient *ethclient.Client,
	seqInfo info,
	sequencerHTTPEndpoint string,
) {
	// Fund Charlie
	seqInfo.GenerateAccount("Charlie")
	TransferBalance(t, "Owner", "Charlie", arbmath.BigMulByUint(oneEth, 500), seqInfo, seqClient, ctx)

	// Auction + timing info
	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, seqClient)
	Require(t, err)

	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)

	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)

	// Decide winner and wait for next round
	placeBidsAndDecideWinner(t, ctx, seqClient, seqInfo, auctionContract, "Bob", "Alice", bobBidderClient, aliceBidderClient, roundDuration)
	time.Sleep(roundTimingInfo.TimeTilNextRound())

	// Prepare express lane client
	chainID, err := seqClient.ChainID(ctx)
	Require(t, err)

	bobPriv := seqInfo.Accounts["Bob"].PrivateKey

	seqDial, err := rpc.Dial(sequencerHTTPEndpoint)
	Require(t, err)

	expressLaneClient := newExpressLaneClient(
		bobPriv,
		chainID,
		*roundTimingInfo,
		auctionContractAddr,
		seqDial,
	)
	expressLaneClient.Start(ctx)

	const size = 80 * 1024 // 80 KB
	data := make([]byte, size)
	_, err = rand.Read(data)
	Require(t, err)

	tx := seqInfo.PrepareTx("Owner", "Bob", 700000000, big.NewInt(1e8), data)
	timeoutMs := hexutil.Uint64(10000)

	isTimeboosted, err := expressLaneClient.SendTransactionSync(ctx, tx, &timeoutMs)
	Require(t, err)

	if isTimeboosted == nil {
		t.Fatal("timeboosted field should exist in the receipt object")
	}
	if isTimeboosted.Timeboosted == nil {
		t.Fatal("timeboosted field should exist in the receipt object")
	}
	if !*isTimeboosted.Timeboosted {
		t.Fatal("tx was not timeboosted, but the field indicates otherwise")
	}
}
