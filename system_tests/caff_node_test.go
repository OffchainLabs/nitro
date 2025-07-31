package arbtest

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/espressostreamer"
	"github.com/offchainlabs/nitro/solgen/go/espressogen"
)

func createCaffNode(
	ctx context.Context,
	t *testing.T,
	existing *NodeBuilder,
	dangerous bool,
) (*NodeBuilder, func(), error) {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	nodeConfig := builder.nodeConfig
	execConfig := builder.execConfig

	// Disable the batch poster because it requires redis if enabled on the 2nd node
	nodeConfig.BatchPoster.Enable = false
	nodeConfig.BlockValidator.Enable = false
	nodeConfig.DelayedSequencer.Enable = false
	nodeConfig.DelayedSequencer.FinalizeDistance = 1
	nodeConfig.Sequencer = false
	nodeConfig.Dangerous.NoSequencerCoordinator = true
	execConfig.Sequencer.Enable = false
	execConfig.ForwardingTarget = existing.l2StackConfig.IPCPath
	execConfig.SecondaryForwardingTarget = []string{}
	nodeConfig.EspressoCaffNode.Enable = true
	nodeConfig.EspressoCaffNode.Namespace = builder.chainConfig.ChainID.Uint64()
	nodeConfig.EspressoCaffNode.NextHotshotBlock = 1
	nodeConfig.EspressoCaffNode.EspressoSGXVerifierAddr = existing.L1Info.GetAddress("EspressoTEEVerifierMock").Hex()
	// reuse the caff node settings so we can set them outside this function.
	nodeConfig.EspressoCaffNode.WaitForFinalization = existing.nodeConfig.EspressoCaffNode.WaitForFinalization
	nodeConfig.EspressoCaffNode.WaitForConfirmations = existing.nodeConfig.EspressoCaffNode.WaitForConfirmations
	nodeConfig.EspressoCaffNode.RequiredBlockDepth = existing.nodeConfig.EspressoCaffNode.RequiredBlockDepth
	nodeConfig.EspressoCaffNode.BatchPosterAddr = "0xb386a74Dcab67b66F8AC07B4f08365d37495Dd23"
	nodeConfig.EspressoCaffNode.FromBlock = 1

	nodeConfig.EspressoCaffNode.StateChecker = arbnode.StateCheckerConfig{
		PollingInterval:        time.Second * 100,
		ErrorToleranceDuration: time.Hour * 1, // Set it to a larger value. That makes the state checker not shut down
		TrustedNodeUrl:         fmt.Sprintf("http://bad-url:%d", 8945),
	}

	nodeConfig.EspressoCaffNode.ForceInclusionChecker = arbnode.ForceInclusionCheckerConfig{
		RetryTime:                time.Second * 2,
		PollingInterval:          time.Second * 1,
		BlockThresholdTolerance:  20,
		SecondThresholdTolerance: 200,
		ErrorToleranceDuration:   time.Minute * 10,
	}

	// for testing, we can use the same hotshot url for both
	nodeConfig.EspressoCaffNode.HotShotUrls = []string{hotShotUrl, hotShotUrl, hotShotUrl, hotShotUrl}
	nodeConfig.EspressoCaffNode.RetryTime = time.Second * 1
	nodeConfig.EspressoCaffNode.HotshotPollingInterval = time.Millisecond * 100
	nodeConfig.ParentChainReader.Enable = true
	nodeConfig.EspressoCaffNode.BlocksToRead = 10000

	builder.l2StackConfig.HTTPPort = getRandomPort(t)
	builder.l2StackConfig.HTTPHost = "0.0.0.0"

	if dangerous {
		nodeConfig.EspressoCaffNode.Dangerous.IgnoreDatabaseHotshotBlock = true
		nodeConfig.EspressoCaffNode.NextHotshotBlock = 0
	}
	cleanup, err := builder.BuildEspressoCaffNode(t, existing)
	return builder, cleanup, err
}

func createCaffNodeConfig(ctx context.Context, t *testing.T) *NodeBuilder {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	nodeConfig := builder.nodeConfig
	execConfig := builder.execConfig

	// Disable the batch poster because it requires redis if enabled on the 2nd node
	nodeConfig.BatchPoster.Enable = false
	nodeConfig.BlockValidator.Enable = false
	nodeConfig.DelayedSequencer.Enable = false
	nodeConfig.DelayedSequencer.FinalizeDistance = 1
	nodeConfig.Sequencer = false
	nodeConfig.Dangerous.NoSequencerCoordinator = true
	execConfig.Sequencer.Enable = false
	execConfig.SecondaryForwardingTarget = []string{}
	nodeConfig.EspressoCaffNode.Enable = true
	nodeConfig.EspressoCaffNode.Namespace = builder.chainConfig.ChainID.Uint64()
	nodeConfig.EspressoCaffNode.NextHotshotBlock = 1
	nodeConfig.EspressoCaffNode.BatchPosterAddr = "0xb386a74Dcab67b66F8AC07B4f08365d37495Dd23"

	// for testing, we can use the same hotshot url for both
	nodeConfig.EspressoCaffNode.HotShotUrls = []string{hotShotUrl, hotShotUrl, hotShotUrl, hotShotUrl}
	nodeConfig.EspressoCaffNode.RetryTime = time.Second * 1
	nodeConfig.EspressoCaffNode.HotshotPollingInterval = time.Millisecond * 100
	nodeConfig.EspressoCaffNode.FromBlock = 1
	nodeConfig.ParentChainReader.Enable = true

	return builder
}

// assertEventOrderingHelper is a simple helper fuction that assists in converting the errors presented by the event functions to booleans and passing them back over the channel
func assertEventOrderingHelper(channel chan bool, eventFunc func() error) {
	err := eventFunc()
	if err != nil {
		channel <- false
	} else {
		channel <- true
	}
}

// AssertEventOrdering:
// This function is responsible for asserting that 2 concurrent events happen in a specific order.
//
// Semantics:
// This function will assert that each event can happen only once and will either succeed or fail.
// The only way this function does not fail is if both events succeed in the correct order.
// This would be relatively easy to change to asserting that the second event can fail before the first event succeeds.
//
// Parameters:
// firstEventFunc: A function that can be executed as a goroutine and has an error condition that can be mapped to success vs failure. This should capture the event that should happen first
// secondEventFunc: A function that can be executed as a goroutine and has an error condition that can be mapped to success vs failure. This should capture the event that should happen second
func AssertEventOrdering(t *testing.T, firstEventFunc func() error, secondEventFunc func() error) {
	var firstEventSuccess bool
	var eventOrderSuccess bool
	firstEvent := make(chan bool)
	secondEvent := make(chan bool)
	go assertEventOrderingHelper(firstEvent, firstEventFunc)
	go assertEventOrderingHelper(secondEvent, secondEventFunc)
	for {
		select {
		case success := <-firstEvent:
			if success {
				firstEventSuccess = true
			} else {
				t.Fatal("First event in ordered assert did not succeed")
			}
		case success := <-secondEvent:
			if !success {
				t.Fatal("Second event in ordered assert did not succeed")
			}
			if !firstEventSuccess {
				t.Fatal("Events occurred in an incorrect order according to the assertion")
			} else {
				eventOrderSuccess = true
				break
			}

		}
		if eventOrderSuccess {
			break
		}
	}
}

func TestEspressoCaffNode(t *testing.T) {
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

	err = checkTransferTxOnL2(t, ctx, builder.L2, "User14", builder.L2Info)
	Require(t, err)
	err = checkTransferTxOnL2(t, ctx, builder.L2, "User15", builder.L2Info)
	Require(t, err)

	newAccount := "User16"
	l2Info := builder.L2Info
	l2Info.GenerateAccount(newAccount)
	addr := l2Info.GetAddress(newAccount)

	// Transfer via the delayed inbox
	delayedTx := l2Info.PrepareTx("Owner", newAccount, 3e7, transferAmount, nil)
	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		WrapL2ForDelayed(t, delayedTx, builder.L1Info, "Faucet", 100000),
	})

	err = waitForWith(ctx, 240*time.Second, 10*time.Second, func() bool {
		balance := builder.L2.GetBalance(t, addr)
		log.Info("waiting for balance", "account", newAccount, "addr", addr, "balance", balance)
		return balance.Cmp(transferAmount) >= 0
	})
	Require(t, err)

	log.Info("Starting the caff node")
	// don't make the caff node wait for finalization during the default test.
	builder.nodeConfig.EspressoCaffNode.WaitForFinalization = false
	// start the node
	builder, cleanupCaffNode, err := createCaffNode(ctx, t, builder, false)
	Require(t, err)
	builderCaffNode := builder.L2
	defer cleanupCaffNode()

	err = waitForWith(ctx, 10*time.Minute, 10*time.Second, func() bool {
		balance1 := builderCaffNode.GetBalance(t, l2Info.GetAddress("User14"))
		balance2 := builderCaffNode.GetBalance(t, l2Info.GetAddress("User15"))
		log.Info("waiting for balance", "account", "User14", "balance", balance1, "account", "User15", "balance", balance2)
		return balance1.Cmp(transferAmount) > 0 && balance2.Cmp(transferAmount) > 0
	})
	Require(t, err)

	err = waitForWith(ctx, 240*time.Second, 10*time.Second, func() bool {
		balance := builderCaffNode.GetBalance(t, addr)
		log.Info("waiting for balance", "account", newAccount, "addr", addr, "balance", balance)
		if balance.Cmp(transferAmount) >= 0 {
			log.Info("Balance has entered account", "balance", balance, "account", newAccount)
		}
		return balance.Cmp(transferAmount) >= 0
	})
	Require(t, err)

	rpcClient := builderCaffNode.Client.Client()
	startTime := time.Now()
	// Wait till we have two blocks created
	for {
		var lastBlock map[string]interface{}
		err = rpcClient.CallContext(ctx, &lastBlock, "eth_getBlockByNumber", "latest", false)
		Require(t, err)
		if lastBlock == nil {
			// fail
			t.Fatal("last block is nil")
		}
		log.Info("last block", "lastBlock", lastBlock)
		numberString, ok := lastBlock["number"].(string)
		if !ok {
			t.Fatal("number is not a string")
		}
		// convert number to uint
		number, err := strconv.ParseInt(numberString, 0, 64)
		Require(t, err)
		if number >= 3 {
			break
		}
		if time.Since(startTime) > 10*time.Minute {
			t.Fatal("timeout waiting for node to create blocks")
		}
		time.Sleep(time.Second * 5)
	}

	// Send transaction to CaffNode and it should works later
	err = checkTransferTxOnL2(t, ctx, builderCaffNode, "User17", l2Info)
	Require(t, err)

	err = rpcClient.CallContext(ctx, nil, "eth_getBlockByNumber", "latest", false)
	Require(t, err)

	err = rpcClient.CallContext(ctx, nil, "eth_getBlockByNumber", "finalized", false)
	Require(t, err)

	err = rpcClient.CallContext(ctx, nil, "eth_getBlockByNumber", "safe", false)
	Require(t, err)

	// start the trusted node
	trustedPort := 9000
	trustedCleanup := mockTrustedNode(t, ctx, trustedPort)
	defer trustedCleanup()

	time.Sleep(10 * time.Second)

	fatalErrChan := make(chan error)
	// Check the state checker
	port := builder.l2StackConfig.HTTPPort
	// Set the trusted node url to the L1 node
	// This is to simulate the trusted url returning a different block
	stateChecker := arbnode.NewStateChecker(
		arbnode.StateCheckerConfig{
			PollingInterval:        time.Second * 1,
			TrustedNodeUrl:         fmt.Sprintf("http://localhost:%d", trustedPort),
			ErrorToleranceDuration: time.Second * 100,
		},
		port,
		fatalErrChan,
	)
	// Start the monitoring task without initial checking
	err = stateChecker.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-fatalErrChan:
		if err == nil {
			t.Fatal("expected an error from fatalErrChan, got nil")
		} else {
			t.Logf("received error as expected: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("did not receive error from fatalErrChan within timeout")
	}
}

func mockTrustedNode(t *testing.T, ctx context.Context, port int) func() {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.l2StackConfig.HTTPPort = port
	builder.l2StackConfig.HTTPHost = "0.0.0.0"
	return builder.BuildL2(t)
}

func Setup(t *testing.T) (context.Context, common.Address, info, string, context.CancelFunc, func(), *NodeBuilder, func(), func()) {
	ctx, cancel := context.WithCancel(context.Background())

	valNodeCleanup := createValidationNode(ctx, t, true)

	builder, cleanup := createL1AndL2Node(ctx, t, true, false)

	err := waitForL1Node(ctx)
	Require(t, err)

	cleanEspresso := runEspresso()

	// wait for the builder
	err = waitForEspressoNode(ctx)
	Require(t, err)

	newAccount := "User16"
	l2Info := builder.L2Info
	l2Info.GenerateAccount(newAccount)
	addr := l2Info.GetAddress(newAccount)
	return ctx, addr, l2Info, newAccount, cancel, valNodeCleanup, builder, cleanup, cleanEspresso
}

func TestEspressoCaffNodeDelayedMessagesConfirmations(t *testing.T) {
	ctx, addr, l2Info, newAccount, cancel, valNodeCleanup, builder, cleanup, cleanEspresso := Setup(t)
	defer cancel()
	defer valNodeCleanup()
	defer cleanup()
	defer cleanEspresso()

	// Set caff node config variables
	builder.nodeConfig.EspressoCaffNode.WaitForConfirmations = true
	builder.nodeConfig.EspressoCaffNode.RequiredBlockDepth = 6
	builder.nodeConfig.EspressoCaffNode.WaitForFinalization = false

	// start the node
	log.Info("Starting the caff node")
	builder2, cleanupCaffNode, err := createCaffNode(ctx, t, builder, false)
	Require(t, err)
	builderCaffNode := builder2.L2
	defer cleanupCaffNode()

	// Transfer via the delayed inbox
	delayedTx := l2Info.PrepareTx("Owner", newAccount, 3e7, transferAmount, nil)
	log.Info("Delayed tx", "delayedtx", delayedTx)
	tx := builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		WrapL2ForDelayed(t, delayedTx, builder.L1Info, "Faucet", 100000),
	})

	// Check the caff node RPC for tx. assert that it is not there.
	_, _, err = builderCaffNode.Client.TransactionByHash(ctx, tx[0].TxHash)
	ExpectErr(t, err, ethereum.NotFound)

	// Create the event function closures for the assert statement.
	firstEvent := func() error {
		err := waitForWith(ctx, 240*time.Second, 1*time.Second, func() bool {
			header, err := builder.L1.Client.HeaderByNumber(ctx, nil) // get the latest header to check tx block depth
			Require(t, err)
			return header.Number.Uint64() >= tx[0].BlockNumber.Uint64()+builder.nodeConfig.EspressoCaffNode.RequiredBlockDepth // check that the tx is at least RequiredBlockDepth blocks deep in the parent chains state.
		})
		return err
	}
	secondEvent := func() error {
		err := waitForWith(ctx, 240*time.Second, 10*time.Second, func() bool {
			balance := builderCaffNode.GetBalance(t, addr)
			log.Info("waiting for balance", "account", newAccount, "addr", addr, "balance", balance)
			if balance.Cmp(transferAmount) >= 0 {
				log.Info("Balance has entered account", "balance", balance, "account", newAccount)
			}
			return balance.Cmp(transferAmount) >= 0
		})
		return err
	}
	// Assert that the delayed message should reach the required block depth before the balance appears on the caff node.
	AssertEventOrdering(t, firstEvent, secondEvent)
	log.Info("Concurrent events finished in the correct order!")
}

func TestEspressoCaffNodeDelayedMessagesFinalized(t *testing.T) {
	ctx, addr, l2Info, newAccount, cancel, valNodeCleanup, builder, cleanup, cleanEspresso := Setup(t)
	defer cancel()
	defer valNodeCleanup()
	defer cleanup()
	defer cleanEspresso()

	// Wait for l1 node
	err := waitForL1Node(ctx)
	Require(t, err)

	// Set caff node config vars
	builder.nodeConfig.EspressoCaffNode.WaitForConfirmations = false
	builder.nodeConfig.EspressoCaffNode.RequiredBlockDepth = 6
	builder.nodeConfig.EspressoCaffNode.WaitForFinalization = true
	// start the node
	log.Info("Starting the caff node")
	builder2, cleanupCaffNode, err := createCaffNode(ctx, t, builder, false)
	Require(t, err)
	builderCaffNode := builder2.L2
	defer cleanupCaffNode()

	// Transfer via the delayed inbox
	delayedTx := l2Info.PrepareTx("Owner", newAccount, 3e7, transferAmount, nil)
	log.Info("Delayed tx", "delayedtx", delayedTx)
	tx := builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		WrapL2ForDelayed(t, delayedTx, builder.L1Info, "Faucet", 100000),
	})
	// Check the caff node RPC for tx. assert that it is not there.
	_, _, err = builderCaffNode.Client.TransactionByHash(ctx, tx[0].TxHash)
	ExpectErr(t, err, ethereum.NotFound)
	// Wait for the tx header to be finalized.

	firstEvent := func() error {
		err := waitForWith(ctx, 240*time.Second, 1*time.Second, func() bool {
			header, err := builder.L1.Client.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
			Require(t, err)
			return header.Number.Int64() >= tx[0].BlockNumber.Int64()
		})
		return err
	}
	secondEvent := func() error {
		err := waitForWith(ctx, 240*time.Second, 10*time.Second, func() bool {
			balance := builderCaffNode.GetBalance(t, addr)
			log.Info("waiting for balance", "account", newAccount, "addr", addr, "balance", balance)
			if balance.Cmp(transferAmount) >= 0 {
				log.Info("Balance has entered account", "balance", balance, "account", newAccount)
			}
			return balance.Cmp(transferAmount) >= 0
		})
		return err
	}
	AssertEventOrdering(t, firstEvent, secondEvent)
	log.Info("Concurrent events finished in the correct order!")
}

func TestEspressoCaffNodeUnfinalizedDelayedMessages(t *testing.T) {
	ctx, addr, l2Info, newAccount, cancel, valNodeCleanup, builder, cleanup, cleanEspresso := Setup(t)
	defer cancel()
	defer valNodeCleanup()
	defer cleanup()
	defer cleanEspresso()
	// set caff node config vars
	builder.nodeConfig.EspressoCaffNode.WaitForConfirmations = false
	builder.nodeConfig.EspressoCaffNode.RequiredBlockDepth = 6
	builder.nodeConfig.EspressoCaffNode.WaitForFinalization = false

	// start the node
	log.Info("Starting the caff node")
	builder2, cleanupCaffNode, err := createCaffNode(ctx, t, builder, false)
	Require(t, err)
	builderCaffNode := builder2.L2
	defer cleanupCaffNode()

	// Transfer via the delayed inbox
	delayedTx3 := l2Info.PrepareTx("Owner", newAccount, 3e7, transferAmount, nil)
	tx3 := builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		WrapL2ForDelayed(t, delayedTx3, builder.L1Info, "Faucet", 100000),
	})
	// Wait for the tx to appear on the caff node
	err = waitForWith(ctx, 240*time.Second, 10*time.Second, func() bool {
		balance := builderCaffNode.GetBalance(t, addr)
		log.Info("waiting for balance", "account", newAccount, "addr", addr, "balance", balance)
		if balance.Cmp(transferAmount) >= 0 {
			log.Info("Balance has entered account", "balance", balance, "account", newAccount)
		}
		return balance.Cmp(transferAmount) >= 0
	})
	Require(t, err)

	finalizedHeader, err := builder.L1.Client.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
	if tx3[0].BlockNumber.Int64() <= finalizedHeader.Number.Int64() {
		t.Fatal("Tx finalized before appearing in the caff node")
	}
	Require(t, err)
}

// RequireErr:
// This serves to assert that we should be expecting some error during the test, and if there is not an error, fail the test.
func RequireErr(t *testing.T, err error, expectedError error) {
	t.Helper()
	if err == nil {
		log.Error("expected an error to occur", "expected error", expectedError)
		t.Fatal(err, expectedError)
	}
}

// ExpectErr:
// This serves to assert that we should be expecting a specific error during the test, and if the error does not match, fail the test.
func ExpectErr(t *testing.T, err error, expectedError error) {
	t.Helper()
	if !errors.Is(err, expectedError) {
		t.Fatal(err, expectedError)
	}
}

// This tests that the caff node config validates that known versions of arb sequencers are not enabled if the caff node is.
func TestEspressoCaffNodeConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := createCaffNodeConfig(ctx, t)
	err := builder.nodeConfig.Validate()
	Require(t, err)

	expectedErr := errors.New("cannot start a Caff node with any sequencer enabled")
	// Test if this node is attempting to be a sequencer
	builder.nodeConfig.Sequencer = true
	err = builder.nodeConfig.Validate()
	RequireErr(t, err, expectedErr)
	// Test the delayed sequencer
	builder.nodeConfig.Sequencer = false
	builder.nodeConfig.DelayedSequencer.Enable = true

	err = builder.nodeConfig.Validate()
	RequireErr(t, err, expectedErr)

	builder.nodeConfig.DelayedSequencer.Enable = false
	builder.nodeConfig.SeqCoordinator.Enable = true

	err = builder.nodeConfig.Validate()
	RequireErr(t, err, expectedErr)

}

func TestEspressoCaffNodeDangerousConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := createL1AndL2Node(ctx, t, true, false)
	defer cleanup()

	// start the node
	_, cleanupCaffNode, err := createCaffNode(ctx, t, builder, true)
	if cleanupCaffNode != nil {
		defer cleanupCaffNode()
	}

	// The actual error is wrapped in an array, so we need to check for that
	expectedErrMsg := "no next hotshot block found in database or dangerous.ignore-database-hotshot-block is set to true, please set config.CaffNodeConfig.NextHotshotBlock"

	if err == nil {
		t.Fatal("Expected an error but got nil")
	}

	// Check if the error contains the expected message (since it might be wrapped)
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error to contain %q, got %q", expectedErrMsg, err.Error())
	}
}

type mockSgxTeeVerifier struct {
	mock.Mock

	time time.Time
}

func (v *mockSgxTeeVerifier) Verify(opts *bind.CallOpts, attestation []byte, signature [32]byte) (espressogen.EnclaveReport, error) {
	if time.Since(v.time) < 1*time.Minute {
		return espressogen.EnclaveReport{}, rpc.HTTPError{StatusCode: 500, Status: "Internal Server Error", Body: []byte("Internal Server Error")}
	}
	return espressogen.EnclaveReport{}, nil
}

func NewMockSgxTeeVerifier() *mockSgxTeeVerifier {
	return &mockSgxTeeVerifier{
		time: time.Now(),
	}
}

func TestEspressoCaffNodeSGXVerifierShouldRetryWhenEncounterRPCError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := createL1AndL2Node(ctx, t, true, false)
	defer cleanup()

	cleanEspresso := runEspresso()
	defer cleanEspresso()

	// wait for the builder
	err := waitForEspressoNode(ctx)
	Require(t, err)

	err = checkTransferTxOnL2(t, ctx, builder.L2, "User14", builder.L2Info)
	Require(t, err)
	err = checkTransferTxOnL2(t, ctx, builder.L2, "User15", builder.L2Info)
	Require(t, err)

	builder2, cleanupCaffNode, err := createCaffNode(ctx, t, builder, false)
	defer cleanupCaffNode()
	Require(t, err)

	err = waitForWith(ctx, 10*time.Minute, 10*time.Second, func() bool {
		balance1 := builder2.L2.GetBalance(t, builder.L2Info.GetAddress("User14"))
		balance2 := builder2.L2.GetBalance(t, builder.L2Info.GetAddress("User15"))
		log.Info("waiting for balance", "account", "User14", "balance", balance1, "account", "User15", "balance", balance2)
		return balance1.Cmp(transferAmount) > 0 && balance2.Cmp(transferAmount) > 0
	})
	Require(t, err)

	espressoStreamerInterface := builder2.L2.ConsensusNode.EspressoCaffNode.GetEspressoStreamer()
	espressoStreamer, ok := espressoStreamerInterface.(*espressostreamer.EspressoStreamer)
	if !ok {
		t.Fatal("espresso streamer is not of type EspressoStreamer")
	}

	err = checkTransferTxOnL2(t, ctx, builder.L2, "User16", builder.L2Info)
	Require(t, err)
	err = checkTransferTxOnL2(t, ctx, builder.L2, "User17", builder.L2Info)
	Require(t, err)

	espressoStreamer.SetSGXVerifier(NewMockSgxTeeVerifier())
	// Set this will cause the caff node to use the sgx verifier
	espressoStreamer.SetBatcherAddressesFetcher(func(l1Height uint64) []common.Address { return []common.Address{{}} })

	err = waitForWith(ctx, 10*time.Minute, 10*time.Second, func() bool {
		balance1 := builder2.L2.GetBalance(t, builder.L2Info.GetAddress("User16"))
		balance2 := builder2.L2.GetBalance(t, builder.L2Info.GetAddress("User17"))
		log.Info("waiting for balance", "account", "User16", "balance", balance1, "account", "User17", "balance", balance2)
		return balance1.Cmp(transferAmount) > 0 && balance2.Cmp(transferAmount) > 0
	})
	Require(t, err)

}

func TestEspressoForceInclusionChecker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	addr := builder.addresses.SequencerInbox
	seqInbox, err := bridgegen.NewSequencerInbox(addr, builder.L1.Client)
	if err != nil {
		t.Fatal(err)
	}

	mockSeqInbox := &MockSeqInbox{
		MaxDelayBlocks:  big.NewInt(20),
		MaxDelaySeconds: big.NewInt(200),
		seqInbox:        seqInbox,
	}

	config := arbnode.ForceInclusionCheckerConfig{
		RetryTime:                time.Second * 2,
		PollingInterval:          time.Second * 1,
		BlockThresholdTolerance:  20,
		SecondThresholdTolerance: 200,
		ErrorToleranceDuration:   time.Minute * 10,
	}

	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.addresses.Bridge, builder.addresses.DeployedAt)
	Require(t, err)

	if builder.addresses.DeployedAt > math.MaxInt64 {
		t.Fatal("deployedAt is greater than max int64")
	}

	// nolint:all
	seqInboxInterface, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, int64(builder.addresses.DeployedAt))
	Require(t, err)

	reader := builder.L2.ConsensusNode.L1Reader

	delayedMessageFetcher := arbnode.NewDelayedMessageFetcher(
		delayedBridge,
		reader,
		builder.L2.ConsensusNode.ArbDB,
		100,
		false,
		false,
		10,
		0,
		seqInboxInterface,
	)

	fatalErrChan := make(chan error)

	forceInclusionChecker := arbnode.NewForceInclusionChecker(mockSeqInbox, config, reader, delayedMessageFetcher, fatalErrChan)
	err = forceInclusionChecker.Start(ctx)
	Require(t, err)

	delayedTx := builder.L2Info.PrepareTx("Faucet", "Owner", 3e7, transferAmount, nil)
	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		WrapL2ForDelayed(t, delayedTx, builder.L1Info, "Faucet", 100000),
	})

	select {
	case err := <-fatalErrChan:
		if err == nil {
			t.Fatal("expected an error from fatalErrChan, got nil")
		} else {
			t.Logf("received error as expected: %v", err)
		}
	case <-time.After(100 * time.Second):
		t.Fatal("did not receive error from fatalErrChan within timeout")
	}
}

// MockSeqInbox is a mock implementation of the sequencer inbox interface,
// allowing customizable time variation values for testing purposes.
// This is useful because the real contract hardcodes MaxTimeVariation when deployBold is disabled.
type MockSeqInbox struct {
	MaxDelayBlocks  *big.Int
	MaxDelaySeconds *big.Int
	seqInbox        *bridgegen.SequencerInbox
}

func (m *MockSeqInbox) MaxTimeVariation(ctx context.Context) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	return m.MaxDelayBlocks, nil, m.MaxDelaySeconds, nil, nil
}

func (m *MockSeqInbox) TotalDelayedMessagesRead(ctx context.Context) (*big.Int, error) {
	return m.seqInbox.TotalDelayedMessagesRead(&bind.CallOpts{Context: ctx})
}
