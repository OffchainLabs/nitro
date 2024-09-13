// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbostypes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const (
	L1MessageType_L2Message             = 3
	L1MessageType_EndOfBlock            = 6
	L1MessageType_L2FundedByL1          = 7
	L1MessageType_RollupEvent           = 8
	L1MessageType_SubmitRetryable       = 9
	L1MessageType_BatchForGasEstimation = 10 // probably won't use this in practice
	L1MessageType_Initialize            = 11
	L1MessageType_EthDeposit            = 12
	L1MessageType_BatchPostingReport    = 13
	L1MessageType_Invalid               = 0xFF
)

const MaxL2MessageSize = 256 * 1024

type L1IncomingMessageHeader struct {
	Kind        uint8          `json:"kind"`
	Poster      common.Address `json:"sender"`
	BlockNumber uint64         `json:"blockNumber"`
	Timestamp   uint64         `json:"timestamp"`
	RequestId   *common.Hash   `json:"requestId" rlp:"nilList"`
	L1BaseFee   *big.Int       `json:"baseFeeL1"`
}

func (h L1IncomingMessageHeader) SeqNum() (uint64, error) {
	if h.RequestId == nil {
		return 0, errors.New("no requestId")
	}
	seqNumBig := h.RequestId.Big()
	if !seqNumBig.IsUint64() {
		return 0, errors.New("bad requestId")
	}
	return seqNumBig.Uint64(), nil
}

type L1IncomingMessage struct {
	Header *L1IncomingMessageHeader `json:"header"`
	L2msg  []byte                   `json:"l2Msg"`

	// Only used for `L1MessageType_BatchPostingReport`
	BatchGasCost *uint64 `json:"batchGasCost,omitempty" rlp:"optional"`
}

var EmptyTestIncomingMessage = L1IncomingMessage{
	Header: &L1IncomingMessageHeader{},
}

var TestIncomingMessageWithRequestId = L1IncomingMessage{
	Header: &L1IncomingMessageHeader{
		Kind:      L1MessageType_Invalid,
		RequestId: &common.Hash{},
		L1BaseFee: big.NewInt(0),
	},
}

var InvalidL1Message = &L1IncomingMessage{
	Header: &L1IncomingMessageHeader{
		Kind: L1MessageType_Invalid,
	},
	L2msg: []byte{},
}

func (msg *L1IncomingMessage) Serialize() ([]byte, error) {
	wr := &bytes.Buffer{}
	if err := wr.WriteByte(msg.Header.Kind); err != nil {
		return nil, err
	}

	if err := util.AddressTo256ToWriter(msg.Header.Poster, wr); err != nil {
		return nil, err
	}

	if err := util.Uint64ToWriter(msg.Header.BlockNumber, wr); err != nil {
		return nil, err
	}

	if err := util.Uint64ToWriter(msg.Header.Timestamp, wr); err != nil {
		return nil, err
	}

	if msg.Header.RequestId == nil {
		return nil, errors.New("cannot serialize L1IncomingMessage without RequestId")
	}
	requestId := *msg.Header.RequestId
	if err := util.HashToWriter(requestId, wr); err != nil {
		return nil, err
	}

	var l1BaseFeeHash common.Hash
	if msg.Header.L1BaseFee == nil {
		return nil, errors.New("cannot serialize L1IncomingMessage without L1BaseFee")
	}
	l1BaseFeeHash = common.BigToHash(msg.Header.L1BaseFee)
	if err := util.HashToWriter(l1BaseFeeHash, wr); err != nil {
		return nil, err
	}

	if _, err := wr.Write(msg.L2msg); err != nil {
		return nil, err
	}

	return wr.Bytes(), nil
}

func (msg *L1IncomingMessage) Equals(other *L1IncomingMessage) bool {
	return msg.Header.Equals(other.Header) && bytes.Equal(msg.L2msg, other.L2msg)
}

func hashesEqual(ha, hb *common.Hash) bool {
	if (ha == nil) != (hb == nil) {
		return false
	}
	return (ha == nil && hb == nil) || *ha == *hb
}

func (h *L1IncomingMessageHeader) Equals(other *L1IncomingMessageHeader) bool {
	// These are all non-pointer types so it's safe to use the == operator
	return h.Kind == other.Kind &&
		h.Poster == other.Poster &&
		h.BlockNumber == other.BlockNumber &&
		h.Timestamp == other.Timestamp &&
		hashesEqual(h.RequestId, other.RequestId) &&
		arbmath.BigEquals(h.L1BaseFee, other.L1BaseFee)
}

func ComputeBatchGasCost(data []byte) uint64 {
	var gas uint64
	for _, b := range data {
		if b == 0 {
			gas += params.TxDataZeroGas
		} else {
			gas += params.TxDataNonZeroGasEIP2028
		}
	}

	// the poster also pays to keccak the batch and place it and a batch-posting report into the inbox
	keccakWords := arbmath.WordsForBytes(uint64(len(data)))
	gas += params.Keccak256Gas + (keccakWords * params.Keccak256WordGas)
	gas += 2 * params.SstoreSetGasEIP2200
	return gas
}

func (msg *L1IncomingMessage) FillInBatchGasCost(batchFetcher FallibleBatchFetcher) error {
	if batchFetcher == nil || msg.Header.Kind != L1MessageType_BatchPostingReport || msg.BatchGasCost != nil {
		return nil
	}
	_, _, batchHash, batchNum, _, _, err := ParseBatchPostingReportMessageFields(bytes.NewReader(msg.L2msg))
	if err != nil {
		return fmt.Errorf("failed to parse batch posting report: %w", err)
	}
	batchData, err := batchFetcher(batchNum)
	if err != nil {
		return fmt.Errorf("failed to fetch batch mentioned by batch posting report: %w", err)
	}
	gotHash := crypto.Keccak256Hash(batchData)
	if gotHash != batchHash {
		return fmt.Errorf("batch fetcher returned incorrect data hash %v (wanted %v for batch %v)", gotHash, batchHash, batchNum)
	}
	gas := ComputeBatchGasCost(batchData)
	msg.BatchGasCost = &gas
	return nil
}

func (msg *L1IncomingMessage) PastBatchesRequired() ([]uint64, error) {
	if msg.Header.Kind != L1MessageType_BatchPostingReport {
		return nil, nil
	}
	_, _, _, batchNum, _, _, err := ParseBatchPostingReportMessageFields(bytes.NewReader(msg.L2msg))
	if err != nil {
		return nil, fmt.Errorf("failed to parse batch posting report: %w", err)
	}
	return []uint64{batchNum}, nil
}

func ParseIncomingL1Message(rd io.Reader, batchFetcher FallibleBatchFetcher) (*L1IncomingMessage, error) {
	var kindBuf [1]byte
	_, err := rd.Read(kindBuf[:])
	if err != nil {
		return nil, err
	}
	kind := kindBuf[0]

	sender, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}

	blockNumber, err := util.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}

	timestamp, err := util.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}

	requestId, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	baseFeeL1, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	msg := &L1IncomingMessage{
		&L1IncomingMessageHeader{
			kind,
			sender,
			blockNumber,
			timestamp,
			&requestId,
			baseFeeL1.Big(),
		},
		data,
		nil,
	}
	err = msg.FillInBatchGasCost(batchFetcher)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

type FallibleBatchFetcher func(batchNum uint64) ([]byte, error)

type ParsedInitMessage struct {
	ChainId          *big.Int
	InitialL1BaseFee *big.Int

	// These may be nil
	ChainConfig           *params.ChainConfig
	SerializedChainConfig []byte
}

// The initial L1 pricing basefee starts at 50 GWei unless set in the init message
var DefaultInitialL1BaseFee = big.NewInt(50 * params.GWei)

var TestInitMessage = &ParsedInitMessage{
	ChainId:          params.ArbitrumDevTestChainConfig().ChainID,
	InitialL1BaseFee: DefaultInitialL1BaseFee,
}

// ParseInitMessage returns the chain id on success
func (msg *L1IncomingMessage) ParseInitMessage() (*ParsedInitMessage, error) {
	if msg.Header.Kind != L1MessageType_Initialize {
		return nil, fmt.Errorf("invalid init message kind %v", msg.Header.Kind)
	}
	basefee := new(big.Int).Set(DefaultInitialL1BaseFee)
	var chainConfig params.ChainConfig
	var chainId *big.Int
	if len(msg.L2msg) == 32 {
		chainId = new(big.Int).SetBytes(msg.L2msg[:32])
		return &ParsedInitMessage{chainId, basefee, nil, nil}, nil
	}
	if len(msg.L2msg) > 32 {
		chainId = new(big.Int).SetBytes(msg.L2msg[:32])
		version := msg.L2msg[32]
		reader := bytes.NewReader(msg.L2msg[33:])
		switch version {
		case 1:
			var err error
			basefee, err = util.Uint256FromReader(reader)
			if err != nil {
				return nil, err
			}
			fallthrough
		case 0:
			serializedChainConfig, err := io.ReadAll(reader)
			if err != nil {
				return nil, err
			}
			err = json.Unmarshal(serializedChainConfig, &chainConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to parse init message, err: %w, message data: %v", err, string(msg.L2msg))
			}
			return &ParsedInitMessage{chainId, basefee, &chainConfig, serializedChainConfig}, nil
		}
	}
	return nil, fmt.Errorf("invalid init message data %v", string(msg.L2msg))
}

func ParseBatchPostingReportMessageFields(rd io.Reader) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error) {
	batchTimestamp, err := util.HashFromReader(rd)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, 0, err
	}
	batchPosterAddr, err := util.AddressFromReader(rd)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, 0, err
	}
	dataHash, err := util.HashFromReader(rd)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, 0, err
	}
	batchNum, err := util.HashFromReader(rd)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, 0, err
	}
	l1BaseFee, err := util.HashFromReader(rd)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, 0, err
	}
	extraGas, err := util.Uint64FromReader(rd)
	if errors.Is(err, io.EOF) {
		// This field isn't always present
		extraGas = 0
		err = nil
	}
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, 0, err
	}
	batchNumBig := batchNum.Big()
	if !batchNumBig.IsUint64() {
		return nil, common.Address{}, common.Hash{}, 0, nil, 0, fmt.Errorf("batch number %v is not a uint64", batchNumBig)
	}
	return batchTimestamp.Big(), batchPosterAddr, dataHash, batchNumBig.Uint64(), l1BaseFee.Big(), extraGas, nil
}
