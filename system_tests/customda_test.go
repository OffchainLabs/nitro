// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/customda"
	"github.com/offchainlabs/nitro/daprovider/daclient"
)

// SimpleStorage is an in-memory storage implementation for CustomDA preimages
type SimpleStorage struct {
	mu      sync.Mutex
	byHash  map[common.Hash][]byte
	allData [][]byte
}

func NewSimpleStorage() *SimpleStorage {
	return &SimpleStorage{
		byHash:  make(map[common.Hash][]byte),
		allData: [][]byte{},
	}
}

func (s *SimpleStorage) Store(ctx context.Context, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Hash the data with SHA-256 and use as key
	hashBytes := sha256.Sum256(data)
	hash := common.BytesToHash(hashBytes[:])

	s.byHash[hash] = data
	s.allData = append(s.allData, data)
	return nil
}

func (s *SimpleStorage) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, exists := s.byHash[hash]
	if !exists {
		return nil, fmt.Errorf("preimage not found for hash: %s", hash.Hex())
	}
	return data, nil
}

// CustomDAServer provides a JSON-RPC endpoint for CustomDA
type CustomDAServer struct {
	server    *http.Server
	rpcServer *rpc.Server
	listener  net.Listener
	storage   *SimpleStorage
	validator *customda.DefaultValidator
	writer    *customda.Writer
	reader    *customda.Reader
}

// RPCService provides RPC methods for the CustomDA server
// It also serves as the Validator interface implementation
type RPCService struct {
	reader    *customda.Reader
	writer    *customda.Writer
	validator daprovider.Validator
}

// IsValidHeaderByte checks if the header byte corresponds to CustomDA
func (s *RPCService) IsValidHeaderByte(ctx context.Context, headerByte byte) (*daclient.IsValidHeaderByteResult, error) {
	isValid := s.reader.IsValidHeaderByte(ctx, headerByte)
	log.Debug("CustomDA RPC: received IsValidHeaderByte request",
		"headerByte", headerByte,
		"isValid", isValid)
	return &daclient.IsValidHeaderByteResult{
		IsValid: isValid,
	}, nil
}

// RecoverPayloadFromBatch recovers a payload from a CustomDA batch
func (s *RPCService) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum hexutil.Uint64,
	batchBlockHash common.Hash,
	sequencerMsg hexutil.Bytes,
	preimagesIn interface{},
	validateSeqMsg bool,
) (*daclient.RecoverPayloadFromBatchResult, error) {
	// Log the incoming request for debugging
	log.Debug("CustomDA RPC: received RecoverPayloadFromBatch request",
		"batchNum", batchNum,
		"sequencerMsgLen", len(sequencerMsg),
		"validateSeqMsg", validateSeqMsg)

	// For testing, we're ignoring the incoming preimages format
	// and just using an empty preimages map
	preimagesMap := make(daprovider.PreimagesMap)

	payload, updatedPreimages, err := s.reader.RecoverPayloadFromBatch(
		ctx,
		uint64(batchNum),
		batchBlockHash,
		sequencerMsg,
		preimagesMap,
		validateSeqMsg,
	)
	if err != nil {
		log.Error("CustomDA RPC: failed to recover payload", "error", err)
		return nil, err
	}

	log.Debug("CustomDA RPC: successfully recovered payload",
		"payloadLen", len(payload),
		"numPreimages", len(updatedPreimages))

	return &daclient.RecoverPayloadFromBatchResult{
		Payload:   payload,
		Preimages: updatedPreimages,
	}, nil
}

// Store stores a message in the CustomDA system
func (s *RPCService) Store(
	ctx context.Context,
	message hexutil.Bytes,
	timeout hexutil.Uint64,
	disableFallbackStoreDataOnChain bool,
) (*daclient.StoreResult, error) {
	log.Info("CustomDA RPC: received Store request",
		"messageLen", len(message),
		"timeout", timeout,
		"disableFallback", disableFallbackStoreDataOnChain,
		"firstByte", func() string {
			if len(message) > 0 {
				return fmt.Sprintf("0x%x", message[0])
			}
			return "none"
		}())

	if len(message) == 0 {
		log.Error("CustomDA RPC: empty message received")
		return nil, fmt.Errorf("empty message")
	}

	// Make sure the message has the correct header byte
	if message[0] != daprovider.CustomDAMessageHeaderFlag {
		log.Info("CustomDA RPC: message doesn't have CustomDA header, adding it",
			"expected", fmt.Sprintf("0x%x", daprovider.CustomDAMessageHeaderFlag),
			"got", fmt.Sprintf("0x%x", message[0]))
		// For testing, add the header if it's missing
		message = append([]byte{daprovider.CustomDAMessageHeaderFlag}, message...)
	}

	log.Info("CustomDA RPC: storing message with writer",
		"writer", fmt.Sprintf("%T", s.writer),
		"validator", fmt.Sprintf("%T", s.validator),
		"messageLen", len(message))

	serializedDACert, err := s.writer.Store(
		ctx,
		message,
		uint64(timeout),
		disableFallbackStoreDataOnChain,
	)
	if err != nil {
		log.Error("CustomDA RPC: failed to store message", "error", err)
		return nil, err
	}

	log.Info("CustomDA RPC: successfully stored message",
		"certLen", len(serializedDACert),
		"totalStoredPreimages", func() int {
			if ss, ok := s.validator.(*customda.DefaultValidator); ok {
				if storage, ok := ss.Storage().(*SimpleStorage); ok {
					return len(storage.byHash)
				}
			}
			return -1
		}())

	return &daclient.StoreResult{
		SerializedDACert: serializedDACert,
	}, nil
}

// RecordPreimages implements the Validator interface to record preimages from a batch
func (s *RPCService) RecordPreimages(ctx context.Context, batch []byte) ([]daprovider.PreimageWithType, error) {
	log.Debug("CustomDA RPC: received RecordPreimages request",
		"batchLen", len(batch))

	return s.validator.RecordPreimages(ctx, batch)
}

// GenerateProof implements the Validator interface to generate proofs for preimages
func (s *RPCService) GenerateProof(ctx context.Context, preimageType arbutil.PreimageType, hash common.Hash, offset uint64) ([]byte, error) {
	log.Debug("CustomDA RPC: received GenerateProof request",
		"preimageType", preimageType,
		"hash", hash.Hex(),
		"offset", offset)

	return s.validator.GenerateProof(ctx, preimageType, hash, offset)
}

// StartCustomDAServer starts a CustomDA server with JSON-RPC API
func startCustomDAServer(t *testing.T, ctx context.Context) (*CustomDAServer, string) {
	// Create the storage and validator
	storage := NewSimpleStorage()
	validator := customda.NewDefaultValidator(storage)

	// Create the writer and reader
	writer := customda.NewWriter(validator)
	reader := customda.NewReader(validator)

	// Create the RPC service
	rpcService := &RPCService{
		reader:    reader,
		writer:    writer,
		validator: validator,
	}

	// Create the RPC server
	rpcServer := rpc.NewServer()
	err := rpcServer.RegisterName("daprovider", rpcService)
	Require(t, err)

	// Start the HTTP server
	listener, err := net.Listen("tcp", "localhost:0")
	Require(t, err)

	server := &http.Server{
		Handler:           rpcServer,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("CustomDA server error", "err", err)
		}
	}()

	customDAServer := &CustomDAServer{
		server:    server,
		rpcServer: rpcServer,
		listener:  listener,
		storage:   storage,
		validator: validator,
		writer:    writer,
		reader:    reader,
	}

	serverURL := "http://" + listener.Addr().String()
	log.Info("Started CustomDA server", "url", serverURL)

	return customDAServer, serverURL
}

// ConfigureNodeForCustomDA configures a node to use CustomDA
func configureNodeForCustomDA(nodeConfig *arbnode.Config, serverURL string) {
	// Configure the batch poster and enable it
	nodeConfig.BatchPoster.Enable = true
	nodeConfig.BatchPoster.UseCustomDA = true
	nodeConfig.BatchPoster.DisableDapFallbackStoreDataOnChain = false

	// Disable traditional DAS (Anytrust) data availability service
	nodeConfig.DataAvailability.Enable = false

	// Enable custom DA provider
	nodeConfig.DAProvider.Enable = true
	nodeConfig.DAProvider.WithWriter = true // This is critical! Enables batch posting to our CustomDA server
	nodeConfig.DAProvider.RPC.URL = serverURL

	// Note: We're keeping default log levels for now

	log.Info("Configured node for CustomDA",
		"serverURL", serverURL,
		"batchPosterEnabled", nodeConfig.BatchPoster.Enable,
		"batchPosterUseCustomDA", nodeConfig.BatchPoster.UseCustomDA,
		"daProviderEnabled", nodeConfig.DAProvider.Enable,
		"daProviderWithWriter", nodeConfig.DAProvider.WithWriter,
		"daProviderURL", nodeConfig.DAProvider.RPC.URL,
		"dataAvailabilityEnabled", nodeConfig.DataAvailability.Enable)
}

// TestCustomDABasic tests basic CustomDA functionality
func TestCustomDABasic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// Enable sequencer mode so it can create batches
	builder.nodeConfig.Sequencer = true
	builder.BuildL1(t)

	// Setup CustomDA server
	customDAServer, serverURL := startCustomDAServer(t, ctx)
	defer func() {
		err := customDAServer.server.Shutdown(ctx)
		if err != nil {
			log.Error("Error shutting down CustomDA server", "err", err)
		}
	}()

	// Setup sequencer node with CustomDA
	configureNodeForCustomDA(builder.nodeConfig, serverURL)

	// Setup L2 chain
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	// Test transferring funds via L2
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)

	// Wait for the batch to be posted and processed
	time.Sleep(time.Millisecond * 500)

	// Force L1 blocks to be created
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, builder.L1.Client, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	//	time.Sleep(15 * time.Second)

	// Log status of CustomDA storage
	log.Info("CustomDA test status after L1 blocks",
		"receivedBatches", len(customDAServer.storage.allData),
		"storedPreimages", len(customDAServer.storage.byHash),
		"batchPosterUseCustomDA", builder.nodeConfig.BatchPoster.UseCustomDA,
		"dataAvailabilityEnabled", builder.nodeConfig.DataAvailability.Enable,
		"daProviderEnabled", builder.nodeConfig.DAProvider.Enable,
		"daProviderURL", builder.nodeConfig.DAProvider.RPC.URL)

	// Verify that preimages were properly recorded in the CustomDA storage
	if len(customDAServer.storage.allData) == 0 {
		Fatal(t, "No batches were stored in CustomDA storage")
	}

	// Validate that we can generate and verify proofs
	for hash := range customDAServer.storage.byHash {
		// Generate a proof for this preimage
		proof, err := customDAServer.validator.GenerateProof(ctx, arbutil.CustomDAPreimageType, hash, 0)
		Require(t, err)

		// Verify the proof by checking it contains a valid preimage
		// The default implementation uses a simple structure where the first byte is the proof type (0)
		// and the rest is the raw preimage data
		if len(proof) <= 1 {
			Fatal(t, "Proof is too short")
		}

		// Extract the preimage data from the proof
		proofType := proof[0]
		preimageData := proof[1:]

		// Validate the proof type
		if proofType != 0 {
			Fatal(t, "Unexpected proof type:", proofType)
		}

		// Verify that the preimage's hash matches
		hashBytes := sha256.Sum256(preimageData)
		computedHash := common.BytesToHash(hashBytes[:])

		if computedHash != hash {
			Fatal(t, "Proof validation failed, hash mismatch", "expected", hash.Hex(), "got", computedHash.Hex())
		}

		log.Info("CustomDA proof validated successfully", "hash", hash.Hex(), "proofSize", len(proof))
	}

	log.Info("CustomDA basic test completed successfully")
}
