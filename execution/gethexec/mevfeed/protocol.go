// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package mevfeed

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	Magic           = "RMEV"
	ProtocolVersion = uint16(1)
	HeaderSize      = 24
)

type FrameKind byte

const (
	FrameHello FrameKind = iota + 1
	FrameBlockBegin
	FrameTransaction
	FrameBlockEnd
	FrameReorg
	FrameGap
	FrameHeartbeat
)

var crcTable = crc32.MakeTable(crc32.Castagnoli)

type frame struct {
	kind     FrameKind
	flags    byte
	sequence uint64
	payload  []byte
}

// Frame is the wire representation of one protocol frame. Payload is copied by
// DecodeFrame so callers can safely retain it after the input buffer is reused.
type Frame struct {
	Kind     FrameKind
	Flags    byte
	Sequence uint64
	Payload  []byte
}

func EncodeFrame(f Frame, max uint32) ([]byte, error) {
	if f.Kind < FrameHello || f.Kind > FrameHeartbeat {
		return nil, fmt.Errorf("invalid frame kind: %d", f.Kind)
	}
	return encodeFrame(frame{kind: f.Kind, flags: f.Flags, sequence: f.Sequence, payload: f.Payload}, max)
}

func DecodeFrame(data []byte, max uint32) (Frame, error) {
	if len(data) < HeaderSize {
		return Frame{}, fmt.Errorf("frame is truncated")
	}
	if string(data[:4]) != Magic {
		return Frame{}, fmt.Errorf("invalid frame magic")
	}
	if binary.BigEndian.Uint16(data[4:6]) != ProtocolVersion {
		return Frame{}, fmt.Errorf("unsupported protocol version")
	}
	if FrameKind(data[6]) < FrameHello || FrameKind(data[6]) > FrameHeartbeat {
		return Frame{}, fmt.Errorf("invalid frame kind")
	}
	length := binary.BigEndian.Uint32(data[16:20])
	if max < HeaderSize || length > max-HeaderSize || int(length) != len(data)-HeaderSize {
		return Frame{}, fmt.Errorf("invalid frame payload length")
	}
	payload := data[HeaderSize:]
	if crc32.Checksum(payload, crcTable) != binary.BigEndian.Uint32(data[20:24]) {
		return Frame{}, fmt.Errorf("frame CRC mismatch")
	}
	return Frame{Kind: FrameKind(data[6]), Flags: data[7], Sequence: binary.BigEndian.Uint64(data[8:16]), Payload: append([]byte(nil), payload...)}, nil
}

func encodeFrame(f frame, max uint32) ([]byte, error) {
	if f.kind < FrameHello || f.kind > FrameHeartbeat {
		return nil, fmt.Errorf("invalid frame kind: %d", f.kind)
	}
	if max < HeaderSize || uint64(len(f.payload)) > uint64(max)-HeaderSize {
		return nil, fmt.Errorf("payload exceeds configured frame limit: %d", len(f.payload))
	}
	out := make([]byte, HeaderSize+len(f.payload))
	copy(out[0:4], Magic)
	binary.BigEndian.PutUint16(out[4:6], ProtocolVersion)
	out[6] = byte(f.kind)
	out[7] = f.flags
	binary.BigEndian.PutUint64(out[8:16], f.sequence)
	binary.BigEndian.PutUint32(out[16:20], uint32(len(f.payload)))
	binary.BigEndian.PutUint32(out[20:24], crc32.Checksum(f.payload, crcTable))
	copy(out[24:], f.payload)
	return out, nil
}

func putHash(b *bytes.Buffer, h common.Hash)       { b.Write(h[:]) }
func putAddress(b *bytes.Buffer, a common.Address) { b.Write(a[:]) }
func putU256(b *bytes.Buffer, n *big.Int) {
	var v [32]byte
	if n != nil {
		n.FillBytes(v[:])
	}
	b.Write(v[:])
}

func blockBeginPayload(block *types.Block, chainID uint64) []byte {
	h := block.Header()
	var b bytes.Buffer
	var id [32]byte
	binary.BigEndian.PutUint64(id[24:], chainID)
	b.Write(id[:])
	binary.Write(&b, binary.BigEndian, block.NumberU64())
	putHash(&b, block.Hash())
	putHash(&b, h.ParentHash)
	putHash(&b, h.Root)
	binary.Write(&b, binary.BigEndian, h.Time)
	binary.Write(&b, binary.BigEndian, h.GasLimit)
	putU256(&b, h.BaseFee)
	putAddress(&b, h.Coinbase)
	putHash(&b, h.MixDigest)
	binary.Write(&b, binary.BigEndian, uint32(len(block.Transactions())))
	return b.Bytes()
}

func transactionPayload(tx *types.Transaction, receipt *types.Receipt, txIndex uint32, chainID uint64) ([]byte, error) {
	raw, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if uint64(len(raw)) > math.MaxUint32 || uint64(len(receipt.Logs)) > math.MaxUint32 {
		return nil, fmt.Errorf("transaction record exceeds protocol bounds")
	}
	var sender common.Address
	if chainID != 0 {
		if s, e := types.Sender(types.LatestSignerForChainID(new(big.Int).SetUint64(chainID)), tx); e == nil {
			sender = s
		}
	}
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, txIndex)
	putHash(&b, tx.Hash())
	putAddress(&b, sender)
	binary.Write(&b, binary.BigEndian, uint32(len(raw)))
	b.Write(raw)
	b.WriteByte(byte(receipt.Status))
	binary.Write(&b, binary.BigEndian, receipt.CumulativeGasUsed)
	binary.Write(&b, binary.BigEndian, receipt.GasUsed)
	binary.Write(&b, binary.BigEndian, receipt.GasUsedForL1)
	putU256(&b, receipt.EffectiveGasPrice)
	if receipt.ContractAddress == (common.Address{}) {
		putAddress(&b, common.Address{})
	} else {
		putAddress(&b, receipt.ContractAddress)
	}
	binary.Write(&b, binary.BigEndian, uint32(len(receipt.Logs)))
	for _, l := range receipt.Logs {
		if len(l.Topics) > math.MaxUint8 || uint64(len(l.Data)) > math.MaxUint32 {
			return nil, fmt.Errorf("log record exceeds protocol bounds")
		}
		putAddress(&b, l.Address)
		b.WriteByte(byte(len(l.Topics)))
		for _, topic := range l.Topics {
			putHash(&b, topic)
		}
		binary.Write(&b, binary.BigEndian, uint32(len(l.Data)))
		b.Write(l.Data)
		binary.Write(&b, binary.BigEndian, txIndex)
		binary.Write(&b, binary.BigEndian, uint32(l.Index))
		if l.Removed {
			b.WriteByte(1)
		} else {
			b.WriteByte(0)
		}
	}
	return b.Bytes(), nil
}

func blockEndPayload(block *types.Block, count uint32, payloadCRC uint32) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, block.NumberU64())
	putHash(&b, block.Hash())
	binary.Write(&b, binary.BigEndian, count)
	binary.Write(&b, binary.BigEndian, payloadCRC)
	return b.Bytes()
}

func helloPayload(session [16]byte, chainID, headNumber uint64, headHash common.Hash, gap bool) []byte {
	var b bytes.Buffer
	b.Write(session[:])
	var id [32]byte
	binary.BigEndian.PutUint64(id[24:], chainID)
	b.Write(id[:])
	binary.Write(&b, binary.BigEndian, headNumber)
	putHash(&b, headHash)
	if gap {
		b.WriteByte(1)
	} else {
		b.WriteByte(0)
	}
	return b.Bytes()
}

func reorgPayload(oldNum, newNum uint64, oldHash, newHash, parent common.Hash) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, oldNum)
	putHash(&b, oldHash)
	binary.Write(&b, binary.BigEndian, newNum)
	putHash(&b, newHash)
	putHash(&b, parent)
	return b.Bytes()
}

func gapPayload(lastNum, headNum uint64, lastHash, headHash common.Hash) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, lastNum)
	putHash(&b, lastHash)
	binary.Write(&b, binary.BigEndian, headNum)
	putHash(&b, headHash)
	return b.Bytes()
}
