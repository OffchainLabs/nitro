// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/containers"
)

const (
	externalL1StartupTimeout  = 30 * time.Second
	externalL1ShutdownTimeout = 5 * time.Second
	externalL1LogTailBytes    = 16 * 1024
	externalL1GasLimit        = 150_000_000
)

type synchronizedBuffer struct {
	mu sync.Mutex
	b  [externalL1LogTailBytes]byte
	// next is the position of the next byte to overwrite. Once the buffer is
	// full, it also identifies the oldest retained byte.
	next int
	size int
}

type externalL1MiningClient struct {
	mu         sync.Mutex
	inner      rpc.ClientInterface
	blobReader *externalL1BlobReader
	timeOffset time.Duration
}

// AdvanceTime makes future blocks at least d newer and immediately seals an
// empty block. It lets transition tests cross timestamp-activated L1 forks.
func (c *externalL1MiningClient) AdvanceTime(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeOffset += d
	empty := []hexutil.Bytes{}
	_, err := c.commitBlock(ctx, &empty)
	return err
}

type externalL1BlobReader struct {
	mu sync.RWMutex
	// Versioned hashes are content-addressed (the hash commits to the blob),
	// so a flat index is correct regardless of which block carried the tx.
	// This also handles multiple blob txs landing in a single block, which a
	// per-block index recorded at submission time mis-attributed.
	blobs map[common.Hash]kzg4844.Blob
}

func (r *externalL1BlobReader) Initialize(context.Context) error {
	return nil
}

func (r *externalL1BlobReader) GetBlobs(_ context.Context, blockHash common.Hash, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	blobs := make([]kzg4844.Blob, len(versionedHashes))
	for i, versionedHash := range versionedHashes {
		blob, ok := r.blobs[versionedHash]
		if !ok {
			return nil, fmt.Errorf("blob %s not found (requested for L1 block %s)", versionedHash, blockHash)
		}
		blobs[i] = blob
	}
	return blobs, nil
}

func (r *externalL1BlobReader) record(txs []*types.Transaction) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, tx := range txs {
		sidecar := tx.BlobTxSidecar()
		if sidecar == nil {
			continue
		}
		for i, versionedHash := range tx.BlobHashes() {
			if i < len(sidecar.Blobs) {
				r.blobs[versionedHash] = sidecar.Blobs[i]
			}
		}
	}
}

func (c *externalL1MiningClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var sentTx *types.Transaction
	var sentRaw hexutil.Bytes
	if method == "eth_sendRawTransaction" {
		var err error
		sentTx, sentRaw, err = transactionFromRPCArgs(args)
		if err != nil {
			return err
		}
	}
	var finalizedTag string
	for _, arg := range args {
		if blockTag, ok := arg.(string); ok && blockTag == "finalized" {
			var latest hexutil.Uint64
			if err := c.inner.CallContext(ctx, &latest, "eth_blockNumber"); err != nil {
				return err
			}
			finalizedTag = hexutil.EncodeUint64(uint64(latest) / 32 * 32)
			break
		}
	}
	for i, arg := range args {
		if blockTag, ok := arg.(string); ok {
			switch blockTag {
			case "safe":
				args[i] = "latest"
			case "finalized":
				args[i] = finalizedTag
			}
		}
	}
	// After a time jump the head may be ahead of wall clock. Because this
	// harness mines submissions synchronously, latest and pending balances are
	// equivalent and latest avoids geth rejecting the future pending block.
	if method == "eth_getBalance" && c.timeOffset > 0 && len(args) == 2 {
		if blockTag, ok := args[1].(string); ok && blockTag == "pending" {
			args[1] = "latest"
		}
	}
	if err := c.inner.CallContext(ctx, result, method, args...); err != nil {
		return err
	}
	if method == "eth_sendRawTransaction" {
		c.blobReader.record([]*types.Transaction{sentTx})
		_, err := c.commitSentTransactions(ctx, []hexutil.Bytes{sentRaw})
		return err
	}
	return nil
}

// commitSentTransactions mines the just-accepted transactions into a block by
// forcing their inclusion. Relying on pool-based selection instead would race
// geth's asynchronous txpool promotion and occasionally mine an empty block.
// Forced inclusion fails for legitimately stale transactions (e.g. a replaced
// nonce), so fall back to pool selection in that case.
func (c *externalL1MiningClient) commitSentTransactions(ctx context.Context, rawTxs []hexutil.Bytes) (common.Hash, error) {
	blockHash, err := c.commitBlock(ctx, &rawTxs)
	if err != nil {
		return c.commitBlock(ctx, nil)
	}
	return blockHash, nil
}

func transactionFromRPCArgs(args []interface{}) (*types.Transaction, hexutil.Bytes, error) {
	if len(args) != 1 {
		return nil, nil, fmt.Errorf("eth_sendRawTransaction expected one argument, got %d", len(args))
	}
	raw, ok := args[0].(string)
	if !ok {
		return nil, nil, fmt.Errorf("eth_sendRawTransaction argument has type %T, want string", args[0])
	}
	encoded, err := hexutil.Decode(raw)
	if err != nil {
		return nil, nil, err
	}
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(encoded); err != nil {
		return nil, nil, err
	}
	return tx, encoded, nil
}

func (c *externalL1MiningClient) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	return c.inner.EthSubscribe(ctx, channel, args...)
}

func (c *externalL1MiningClient) BatchCallContext(ctx context.Context, batch []rpc.BatchElem) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var sentTxs []*types.Transaction
	var sentRaws []hexutil.Bytes
	for _, elem := range batch {
		if elem.Method == "eth_sendRawTransaction" {
			tx, raw, err := transactionFromRPCArgs(elem.Args)
			if err != nil {
				return err
			}
			sentTxs = append(sentTxs, tx)
			sentRaws = append(sentRaws, raw)
		}
	}
	if err := c.inner.BatchCallContext(ctx, batch); err != nil {
		return err
	}
	var acceptedTxs []*types.Transaction
	var acceptedRaws []hexutil.Bytes
	sent := 0
	for _, elem := range batch {
		if elem.Method != "eth_sendRawTransaction" {
			continue
		}
		if elem.Error == nil {
			acceptedTxs = append(acceptedTxs, sentTxs[sent])
			acceptedRaws = append(acceptedRaws, sentRaws[sent])
		}
		sent++
	}
	if len(acceptedRaws) > 0 {
		c.blobReader.record(acceptedTxs)
		_, err := c.commitSentTransactions(ctx, acceptedRaws)
		return err
	}
	return nil
}

func (c *externalL1MiningClient) Close() {
	c.inner.Close()
}

func (c *externalL1MiningClient) CommitTransactions(ctx context.Context, txs []*types.Transaction) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	rawTxs := make([]hexutil.Bytes, 0, len(txs))
	for _, tx := range txs {
		rawTx, err := tx.MarshalBinary()
		if err != nil {
			return err
		}
		rawTxs = append(rawTxs, rawTx)
	}
	_, err := c.commitBlock(ctx, &rawTxs)
	if err == nil {
		c.blobReader.record(txs)
	}
	return err
}

func (c *externalL1MiningClient) commitBlock(ctx context.Context, txs *[]hexutil.Bytes) (common.Hash, error) {
	header, err := ethclient.NewClient(c.inner).HeaderByNumber(ctx, nil)
	if err != nil {
		return common.Hash{}, err
	}
	timestamp := max(header.Time+1, uint64(time.Now().Add(c.timeOffset).Unix())) // #nosec G115 -- current Unix time is non-negative
	slotNumber := uint64(0)
	if header.SlotNumber != nil {
		slotNumber = *header.SlotNumber + 1
	}
	beaconRoot := common.Hash{}
	attributes := engine.PayloadAttributes{
		Timestamp:             timestamp,
		Random:                header.Hash(),
		SuggestedFeeRecipient: header.Coinbase,
		Withdrawals:           []*types.Withdrawal{},
		BeaconRoot:            &beaconRoot,
		SlotNumber:            &slotNumber,
	}
	var blockHash common.Hash
	err = c.inner.CallContext(ctx, &blockHash, "testing_commitBlockV1", attributes, txs, nil)
	return blockHash, err
}

func (b *synchronizedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	written := len(data)
	if written >= len(b.b) {
		copy(b.b[:], data[written-len(b.b):])
		b.next = 0
		b.size = len(b.b)
		return written, nil
	}
	for len(data) > 0 {
		copied := copy(b.b[b.next:], data)
		b.next = (b.next + copied) % len(b.b)
		data = data[copied:]
	}
	b.size = min(len(b.b), b.size+written)
	return written, nil
}

func (b *synchronizedBuffer) tail(limit int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	limit = min(limit, b.size)
	start := (b.next - limit + len(b.b)) % len(b.b)
	if start+limit <= len(b.b) {
		return string(b.b[start : start+limit])
	}
	data := make([]byte, limit)
	copied := copy(data, b.b[start:])
	copy(data[copied:], b.b[:limit-copied])
	return string(data)
}

type ExternalL1Params struct {
	// Info is the L1 test info to use; nil creates a fresh NewL1TestInfo.
	Info       *BlockchainTestInfo
	GethBinary string
	// WithClientWrapper wraps the returned client so tests can intercept and
	// stall L1 RPC traffic.
	WithClientWrapper bool
	// GenesisOverrides is merged into the genesis chain config (nil deletes a
	// key). A non-nil map makes the caller authoritative for the fork schedule.
	GenesisOverrides map[string]interface{}
}

type ExternalL1 struct {
	Info          *BlockchainTestInfo
	Client        *ethclient.Client
	GethClient    *gethclient.Client
	ClientWrapper *ClientWrapper // nil unless WithClientWrapper was set
	MiningClient  *externalL1MiningClient
	BlobReader    containers.Option[daprovider.BlobReader]
	// Close shuts down the geth process. It is idempotent and also registered
	// via t.Cleanup.
	Close func()
}

func CreateExternalL1(t *testing.T, ctx context.Context, params ExternalL1Params) *ExternalL1 {
	t.Helper()

	l1info := params.Info
	gethBinary := params.GethBinary
	withClientWrapper := params.WithClientWrapper
	genesisOverrides := params.GenesisOverrides
	if l1info == nil {
		l1info = NewL1TestInfo(t)
	}
	if !l1info.HasAccount("Faucet") {
		l1info.GenerateAccount("Faucet")
	}
	for _, acct := range DefaultChainAccounts {
		if !l1info.HasAccount(acct) {
			l1info.GenerateAccount(acct)
		}
	}
	gethBinary, err := filepath.Abs(gethBinary)
	Require(t, err)
	if stat, err := os.Stat(gethBinary); err != nil {
		t.Fatalf("External geth binary is unavailable at %q: %v; run make build-upstream-geth", gethBinary, err)
	} else if stat.IsDir() {
		t.Fatalf("External geth binary path %q is a directory; run make build-upstream-geth", gethBinary)
	}

	dataDir := t.TempDir()
	genesisPath := filepath.Join(dataDir, "genesis.json")
	ipcFile, err := os.CreateTemp("", "nitro-l1-*.ipc")
	Require(t, err)
	ipcPath := ipcFile.Name()
	Require(t, ipcFile.Close())
	Require(t, os.Remove(ipcPath))
	t.Cleanup(func() {
		if err := os.Remove(ipcPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Logf("Could not remove external geth IPC socket: %v", err)
		}
	})
	if genesisOverrides == nil {
		// The selected upstream developer genesis schedules Bogota at genesis.
		// Remove it so Osaka remains the latest active fork in Nitro's current
		// external-L1 preset.
		genesisOverrides = map[string]interface{}{"bogotaTime": nil}
	}
	writeExternalL1Genesis(t, gethBinary, genesisPath, l1info.GetAddress("Faucet"), genesisOverrides)
	initArgs := []string{"init", "--datadir", dataDir, genesisPath}
	if initOutput, err := exec.Command(gethBinary, initArgs...).CombinedOutput(); err != nil {
		t.Fatalf("Could not initialize external geth genesis: %v\n%s", err, initOutput)
	}

	// Dev mode registers only the simulated beacon APIs, so run a regular node
	// instead: it serves both the testing and engine namespaces over IPC, and
	// the engine API is what ReorgToOldBlock needs to rewind the head without
	// truncating the abandoned blocks.
	var gethLog synchronizedBuffer
	runArgs := []string{
		"--syncmode", "full",
		"--nodiscover",
		"--maxpeers", "0",
		"--port", "0",
		"--nat", "none",
		"--authrpc.port", "0",
		"--miner.gaslimit", strconv.FormatUint(externalL1GasLimit, 10),
		"--rpc.gascap", strconv.FormatUint(externalL1GasLimit, 10),
		"--rpc.txfeecap", "0",
		// Reorg tests deliberately rewind further than the default 32-block
		// engine API limit.
		"--engine.maxreorgdepth", "0",
		"--datadir", dataDir,
		"--ipcpath", ipcPath,
		"--verbosity", "4",
	}
	cmd := exec.Command(gethBinary, runArgs...)
	cmd.Stdout = &gethLog
	cmd.Stderr = &gethLog
	Require(t, cmd.Start())

	var processErr error
	processExited := make(chan struct{})
	go func() {
		processErr = cmd.Wait()
		close(processExited)
	}()

	var rpcClient *rpc.Client
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if t.Failed() {
				t.Logf("External geth log tail:\n%s", gethLog.tail(externalL1LogTailBytes))
			}
			if rpcClient != nil {
				rpcClient.Close()
			}
			select {
			case <-processExited:
				return
			default:
			}
			if err := cmd.Process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
				t.Logf("Could not interrupt external geth: %v", err)
			}
			select {
			case <-processExited:
			case <-time.After(externalL1ShutdownTimeout):
				if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
					t.Logf("Could not kill external geth: %v", err)
				}
				<-processExited
			}
		})
	}
	t.Cleanup(cleanup)

	startupCtx, cancel := context.WithTimeout(ctx, externalL1StartupTimeout)
	defer cancel()
	for rpcClient == nil {
		select {
		case <-processExited:
			t.Fatalf("External geth exited during startup: %v\n%s", processErr, gethLog.tail(externalL1LogTailBytes))
		case <-startupCtx.Done():
			t.Fatalf("Timed out waiting for external geth IPC: %v\n%s", startupCtx.Err(), gethLog.tail(externalL1LogTailBytes))
		default:
		}
		client, err := rpc.DialIPC(startupCtx, ipcPath)
		if err == nil {
			rpcClient = client
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	gethClient := gethclient.New(rpcClient)
	client := ethclient.NewClient(rpcClient)
	chainID, err := client.ChainID(startupCtx)
	Require(t, err)
	if chainID.Cmp(bigChainID1337) != 0 {
		t.Fatalf("Unexpected external L1 chain ID: got %s, want %s", chainID, bigChainID1337)
	}
	verifyExternalL1HeaderHash(t, startupCtx, rpcClient)
	markExternalL1Synced(t, startupCtx, rpcClient)
	blobReader := &externalL1BlobReader{blobs: make(map[common.Hash]kzg4844.Blob)}
	miningClient := &externalL1MiningClient{inner: rpcClient, blobReader: blobReader}
	client = ethclient.NewClient(miningClient)

	var fundingTxs []*types.Transaction
	externalAccountBalance := new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	for _, acct := range DefaultChainAccounts {
		fundingTxs = append(fundingTxs, l1info.PrepareTx(
			"Faucet",
			acct,
			l1info.TransferGas,
			new(big.Int).Set(externalAccountBalance),
			nil,
		))
	}
	faucetAddress := l1info.GetAddress("Faucet")
	for _, account := range l1info.ArbInitData.Accounts {
		if account.Addr == faucetAddress || account.EthBalance == nil || account.EthBalance.Sign() == 0 {
			continue
		}
		address := account.Addr
		fundingTxs = append(fundingTxs, l1info.PrepareTxTo(
			"Faucet",
			&address,
			l1info.TransferGas,
			new(big.Int).Set(account.EthBalance),
			nil,
		))
	}
	SendWaitTestTransactions(t, ctx, client, fundingTxs)

	var clientWrapper *ClientWrapper
	if withClientWrapper {
		clientWrapper = NewClientWrapper(miningClient, l1info)
		client = ethclient.NewClient(clientWrapper)
	}
	t.Logf("Using external L1 from %s", gethBinary)
	return &ExternalL1{
		Info:          l1info,
		Client:        client,
		GethClient:    gethClient,
		ClientWrapper: clientWrapper,
		MiningClient:  miningClient,
		BlobReader:    containers.Some[daprovider.BlobReader](blobReader),
		Close:         cleanup,
	}
}

var bigChainID1337 = new(big.Int).SetUint64(1337)

// writeExternalL1Genesis derives the external L1 genesis from the pinned
// geth's developer preset, pre-funding the faucet so the chain can run as a
// regular (non-dev) node.
func writeExternalL1Genesis(t *testing.T, gethBinary, genesisPath string, faucet common.Address, configOverrides map[string]interface{}) {
	t.Helper()
	devGenesis, err := exec.Command(gethBinary, "--dev", "dumpgenesis").Output()
	Require(t, err, "could not dump the developer genesis")
	var genesis map[string]interface{}
	Require(t, json.Unmarshal(devGenesis, &genesis))
	alloc, ok := genesis["alloc"].(map[string]interface{})
	if !ok {
		t.Fatal("Developer genesis has no alloc")
	}
	faucetBalance := new(big.Int).Exp(big.NewInt(10), big.NewInt(45), nil)
	alloc[faucet.Hex()] = map[string]interface{}{"balance": hexutil.EncodeBig(faucetBalance)}
	genesis["gasLimit"] = hexutil.Uint64(externalL1GasLimit).String()
	if len(configOverrides) > 0 {
		config, ok := genesis["config"].(map[string]interface{})
		if !ok {
			t.Fatal("Developer genesis has no chain config")
		}
		for key, value := range configOverrides {
			if value == nil {
				delete(config, key)
			} else {
				config[key] = value
			}
		}
	}
	encoded, err := json.Marshal(genesis)
	Require(t, err)
	Require(t, os.WriteFile(genesisPath, encoded, 0o600))
}

// markExternalL1Synced issues a forkchoice update to the current head. The
// no-op update flips geth's synced flag, without which the engine API
// silently ignores forkchoice updates that rewind the head (see
// eth/catalyst.(*ConsensusAPI).forkchoiceUpdated).
func markExternalL1Synced(t *testing.T, ctx context.Context, rpcClient *rpc.Client) {
	t.Helper()
	head, err := ethclient.NewClient(rpcClient).HeaderByNumber(ctx, nil)
	Require(t, err)
	var response engine.ForkChoiceResponse
	Require(t, rpcClient.CallContext(
		ctx,
		&response,
		"engine_forkchoiceUpdatedV3",
		engine.ForkchoiceStateV1{HeadBlockHash: head.Hash()},
		nil,
	))
	if response.PayloadStatus.Status != engine.VALID {
		t.Fatalf("Priming forkchoice update returned status %s", response.PayloadStatus.Status)
	}
}

func verifyExternalL1HeaderHash(t *testing.T, ctx context.Context, rpcClient rpc.ClientInterface) {
	t.Helper()
	var block map[string]json.RawMessage
	Require(t, rpcClient.CallContext(ctx, &block, "eth_getBlockByNumber", "latest", false))
	var rpcHash common.Hash
	Require(t, json.Unmarshal(block["hash"], &rpcHash))
	header, err := ethclient.NewClient(rpcClient).HeaderByHash(ctx, rpcHash)
	Require(t, err)
	if header.Hash() != rpcHash {
		t.Fatalf("Typed external L1 header hash mismatch: got %s, want %s", header.Hash(), rpcHash)
	}
}
