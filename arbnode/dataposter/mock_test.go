// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dataposter

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
	"github.com/offchainlabs/nitro/arbnode/dataposter/slice"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/util/headerreader"
)

// rpcCall records a single RPC method invocation with its arguments.
// Used by testStubClient to enable assertions on both which methods were
// called and what arguments were passed (e.g. verifying "finalized" vs
// "latest" block tag in updateNonce).
type rpcCall struct {
	Method string
	Args   []interface{}
}

// testStubClient implements rpc.ClientInterface for use in unit tests.
// It intercepts CallContext and routes by RPC method name.
//
// Concurrency: This stub is accessed exclusively from the test goroutine.
// None of the functions under test spawn background goroutines, we never call
// Start() on the DataPoster/HeaderReader/ParentChain, and all ethclient
// methods are synchronous (they call CallContext and block). Therefore no
// mutex is needed. If a future test requires concurrent access (e.g. testing
// the Start() loop), a mutex should be added at that point.
type testStubClient struct {
	// Base values returned by RPC methods.
	senderNonce        uint64
	suggestedGasTipCap *big.Int
	latestBlockNumber  uint64
	latestHeader       *types.Header
	balance            *big.Int

	// Configurable errors.
	headerByNumberErr error // returned by eth_getBlockByNumber
	nonceAtErr        error // returned by eth_getTransactionCount
	sendTxErr         error // returned by eth_sendRawTransaction
	txByHashErr       error // returned by eth_getTransactionByHash

	// Optional callback for block-tag-specific nonce queries.
	// Called with the block tag string ("latest", "finalized", or a hex block number).
	// If nil, all NonceAt calls return senderNonce.
	nonceAtFunc func(blockTag string) uint64

	// Recording fields for test assertions.
	sendTxCount int       // number of eth_sendRawTransaction calls
	calls       []rpcCall // method names and args in call order
}

// CallContext implements rpc.ClientInterface. It dispatches by RPC method name
// and populates the result pointer according to each method's ethclient
// expectations (verified against go-ethereum/ethclient/ethclient.go).
func (c *testStubClient) CallContext(_ context.Context, result interface{}, method string, args ...interface{}) error {
	c.calls = append(c.calls, rpcCall{Method: method, Args: args})

	switch method {
	case "eth_getTransactionCount":
		// ethclient.NonceAt passes *hexutil.Uint64.
		// args: [account, blockTag string]
		if c.nonceAtErr != nil {
			return c.nonceAtErr
		}
		nonce := c.senderNonce
		if c.nonceAtFunc != nil && len(args) >= 2 {
			if tag, ok := args[1].(string); ok {
				nonce = c.nonceAtFunc(tag)
			}
		}
		ptr, ok := result.(*hexutil.Uint64)
		if !ok {
			panic(fmt.Sprintf("eth_getTransactionCount: result is %T, want *hexutil.Uint64", result))
		}
		*ptr = hexutil.Uint64(nonce)

	case "eth_maxPriorityFeePerGas":
		// ethclient.SuggestGasTipCap passes *hexutil.Big.
		ptr, ok := result.(*hexutil.Big)
		if !ok {
			panic(fmt.Sprintf("eth_maxPriorityFeePerGas: result is %T, want *hexutil.Big", result))
		}
		if c.suggestedGasTipCap != nil {
			*ptr = hexutil.Big(*c.suggestedGasTipCap)
		}

	case "eth_getBlockByNumber":
		// ethclient.HeaderByNumber: var head *types.Header; CallContext(ctx, &head, ...).
		// So result is **types.Header.
		if c.headerByNumberErr != nil {
			return c.headerByNumberErr
		}
		if c.latestHeader == nil {
			return errors.New("eth_getBlockByNumber: no latestHeader configured in stub")
		}
		ptrptr, ok := result.(**types.Header)
		if !ok {
			panic(fmt.Sprintf("eth_getBlockByNumber: result is %T, want **types.Header", result))
		}
		*ptrptr = c.latestHeader

	case "eth_blockNumber":
		// ethclient.BlockNumber passes *hexutil.Uint64.
		ptr, ok := result.(*hexutil.Uint64)
		if !ok {
			panic(fmt.Sprintf("eth_blockNumber: result is %T, want *hexutil.Uint64", result))
		}
		*ptr = hexutil.Uint64(c.latestBlockNumber)

	case "eth_getBalance":
		// ethclient.BalanceAt passes *hexutil.Big.
		ptr, ok := result.(*hexutil.Big)
		if !ok {
			panic(fmt.Sprintf("eth_getBalance: result is %T, want *hexutil.Big", result))
		}
		if c.balance != nil {
			*ptr = hexutil.Big(*c.balance)
		}

	case "eth_sendRawTransaction":
		// ethclient.SendTransaction passes nil as result.
		c.sendTxCount++
		return c.sendTxErr

	case "eth_getTransactionByHash":
		// ethclient.TransactionByHash: var json *rpcTransaction; CallContext(ctx, &json, ...).
		// rpcTransaction is private to ethclient, so we can only simulate:
		//   - "not found": leave result as nil → ethclient returns ethereum.NotFound
		//   - "RPC error": return c.txByHashErr
		// We cannot simulate "tx found" without constructing the private type.
		return c.txByHashErr

	case "eth_getCode":
		// ethclient.CodeAt passes *hexutil.Bytes. Return empty (no contract).
		ptr, ok := result.(*hexutil.Bytes)
		if !ok {
			panic(fmt.Sprintf("eth_getCode: result is %T, want *hexutil.Bytes", result))
		}
		*ptr = hexutil.Bytes{}

	case "eth_config":
		// parent.pollEthConfig calls L1Reader.Client().Client().CallContext(ctx, &resp, "eth_config").
		// resp is parent.ethConfigResponse (unexported), so we can't populate it.
		// Returning an error causes NewParentChain to fall back to static config
		// from knownConfigs[1337] = params.AllDevChainProtocolChanges.
		return errors.New("eth_config not supported by test stub")
	}

	return nil
}

func (c *testStubClient) EthSubscribe(_ context.Context, _ interface{}, _ ...interface{}) (*rpc.ClientSubscription, error) {
	return nil, errors.New("subscriptions not supported by test stub")
}

func (c *testStubClient) BatchCallContext(_ context.Context, _ []rpc.BatchElem) error {
	return errors.New("batch calls not supported by test stub")
}

func (c *testStubClient) Close() {}

// clearConstructionCalls clears the RPC call log and send counter.
// Called once after construction (which makes eth_config calls via
// NewParentChain) to give tests a clean recording slate. This is NOT
// for resetting between test cases — construct a fresh stub instead.
func (c *testStubClient) clearConstructionCalls() {
	c.sendTxCount = 0
	c.calls = nil
}

// defaultTestHeader returns a types.Header with fields set to reasonable
// values for post-Cancun Ethereum. Tests can override individual fields
// on the returned header since all fields are public.
func defaultTestHeader() *types.Header {
	excessBlobGas := uint64(0)
	blobGasUsed := uint64(0)
	return &types.Header{
		Number:        big.NewInt(100),
		Time:          uint64(time.Now().Unix()),
		BaseFee:       big.NewInt(10 * params.GWei),
		ExcessBlobGas: &excessBlobGas,
		BlobGasUsed:   &blobGasUsed,
	}
}

// defaultTestStub returns a testStubClient with sensible defaults.
// Tests should override fields as needed for their specific scenario.
func defaultTestStub() *testStubClient {
	return &testStubClient{
		senderNonce:        0,
		suggestedGasTipCap: big.NewInt(2 * params.GWei),
		latestBlockNumber:  100,
		latestHeader:       defaultTestHeader(),
		balance:            new(big.Int).Mul(big.NewInt(10), big.NewInt(params.Ether)),
	}
}

// newTestDataPoster constructs a DataPoster backed by the given stub, suitable
// for unit tests. It returns the DataPoster and its InternalState
// (so tests can Lock/Unlock to access LockedInternalState directly).
//
// The cfg parameter overrides the default config. Pass nil to use TestDataPosterConfig.
//
// Construction order:
//  1. ethclient.NewClient(stub) → wraps stub as *ethclient.Client
//  2. headerreader.New(ctx, client, TestConfig, nil) → not Started, so
//     LastHeader() falls through to client.HeaderByNumber → stub
//  3. parent.NewParentChain(ctx, big.NewInt(1337), hr) → calls eth_config
//     via stub (returns error → falls back to knownConfigs[1337])
//  4. slice.NewStorage → in-memory queue (non-persistent)
//  5. state.NewInternalState(queue)
//  6. Compile MaxFeeCapFormula → *govaluate.EvaluableExpression
//  7. Assemble DataPoster with passthrough signer
//
// After construction, stub.clearConstructionCalls() is called to clear the eth_config
// call from the recording, giving tests a clean slate.
func newTestDataPoster(t testing.TB, stub *testStubClient, cfg *config.DataPosterConfig) (*DataPoster, *state.InternalState) {
	t.Helper()

	if cfg == nil {
		c := config.TestDataPosterConfig
		cfg = &c
	}

	ctx := context.Background()
	client := ethclient.NewClient(stub)

	hr, err := headerreader.New(
		ctx,
		client,
		func() *headerreader.Config { c := headerreader.TestConfig; return &c },
		nil, // arbSysPrecompile=nil → isParentChainArbitrum=false, no CodeAt RPC
	)
	if err != nil {
		t.Fatalf("headerreader.New: %v", err)
	}

	pc := parent.NewParentChain(ctx, big.NewInt(1337), hr)

	sliceStorage := slice.NewStorage(func() storage.EncoderDecoderInterface {
		return &storage.EncoderDecoder{}
	})
	internalState := state.NewInternalState(sliceStorage)

	expression, err := govaluate.NewEvaluableExpression(config.DefaultDataPosterConfig.MaxFeeCapFormula)
	if err != nil {
		t.Fatalf("govaluate.NewEvaluableExpression: %v", err)
	}

	parentChainID256, _ := uint256.FromBig(big.NewInt(1337))

	dp := &DataPoster{
		headerReader: hr,
		client:       client,
		auth:         &bind.TransactOpts{From: common.Address{}},
		signer: func(_ context.Context, _ common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return tx, nil // passthrough: return the transaction as-is
		},
		config:              func() *config.DataPosterConfig { return cfg },
		extraBacklog:        func() uint64 { return 0 },
		parentChainID256:    parentChainID256,
		parentChain:         pc,
		internalState:       internalState,
		maxFeeCapExpression: expression,
	}

	// Clear RPC calls recorded during construction (eth_config from NewParentChain).
	stub.clearConstructionCalls()

	return dp, internalState
}

// makeTestQueuedTx creates a non-blob QueuedTransaction for testing.
// Callers can override public fields after creation (e.g. tx.Created for
// elapsed time tests, tx.NextReplacement for replacement scheduling tests).
func makeTestQueuedTx(nonce uint64, feeCap, tipCap *big.Int, sent bool) *storage.QueuedTransaction {
	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Gas:       300_000,
		To:        &common.Address{},
		Value:     big.NewInt(0),
	})
	return &storage.QueuedTransaction{
		FullTx:  tx,
		Sent:    sent,
		Created: time.Now(),
		DeprecatedData: types.DynamicFeeTx{
			GasFeeCap: feeCap,
			GasTipCap: tipCap,
		},
	}
}

// makeTestBlobQueuedTx creates a BlobTxType QueuedTransaction for testing.
// Note: BlobTx.Sidecar has `rlp:"-"` tag and is excluded from encoding,
// so nil sidecar is fine. ChainID is set to 1337 to match the test chain
// (required for MarshalBinary in SendTransaction).
// Callers can override public fields after creation.
func makeTestBlobQueuedTx(nonce uint64, feeCap, tipCap, blobFeeCap *big.Int, numBlobs int, sent bool) *storage.QueuedTransaction {
	blobHashes := make([]common.Hash, numBlobs)
	for i := range blobHashes {
		blobHashes[i] = common.Hash{}
	}
	tx := types.NewTx(&types.BlobTx{
		ChainID:    uint256.NewInt(1337),
		Nonce:      nonce,
		GasTipCap:  uint256.MustFromBig(tipCap),
		GasFeeCap:  uint256.MustFromBig(feeCap),
		BlobFeeCap: uint256.MustFromBig(blobFeeCap),
		Gas:        300_000,
		To:         common.Address{},
		Value:      uint256.NewInt(0),
		BlobHashes: blobHashes,
	})
	return &storage.QueuedTransaction{
		FullTx:  tx,
		Sent:    sent,
		Created: time.Now(),
		DeprecatedData: types.DynamicFeeTx{
			GasFeeCap: feeCap,
			GasTipCap: tipCap,
		},
	}
}

// putTxInQueue inserts a QueuedTransaction into the queue at the given nonce.
// It handles both appending (prevTx=nil for new entries) and replacement
// (prevTx must match what's stored). The queue must be locked via
// s = internalState.Lock() before calling.
func putTxInQueue(t testing.TB, ctx context.Context, s *state.LockedInternalState, nonce uint64, prevTx, newTx *storage.QueuedTransaction) {
	t.Helper()
	if err := s.Queue.Put(ctx, nonce, prevTx, newTx); err != nil {
		t.Fatalf("putTxInQueue(nonce=%d): %v", nonce, err)
	}
}
