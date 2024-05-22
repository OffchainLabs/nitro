// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !wasm
// +build !wasm

package wavmio

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
)

// source for arrayFlags: https://stackoverflow.com/questions/28322997/how-to-get-a-list-of-values-into-a-flag-in-golang
type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	seqMsg             []byte
	seqMsgPos          uint64
	posWithinMsg       uint64
	delayedMsgs        [][]byte
	delayedMsgFirstPos uint64
	lastBlockHash      common.Hash
	hotShotCommitment  [32]byte
	preimages          map[common.Hash][]byte
	seqAdvanced        uint64
	espressoHeight     uint64
)

func parsePreimageBytes(path string) {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	for {
		lenBuf := make([]byte, 8)
		read, err := file.Read(lenBuf)
		if err == io.EOF {
			return
		}
		if err != nil {
			panic(err)
		}
		if read != len(lenBuf) {
			panic(fmt.Sprintf("missing bytes reading len got %d", read))
		}
		fieldSize := int(binary.LittleEndian.Uint64(lenBuf))
		dataBuf := make([]byte, fieldSize)
		read, err = file.Read(dataBuf)
		if err != nil {
			panic(err)
		}
		if read != fieldSize {
			panic("missing bytes reading data")
		}
		hash := crypto.Keccak256Hash(dataBuf)
		preimages[hash] = dataBuf
	}
}

func StubInit() {
	preimages = make(map[common.Hash][]byte)
	var delayedMsgPath arrayFlags
	seqMsgPosFlag := flag.Int("inbox-position", 0, "position for sequencer inbox message")
	posWithinMsgFlag := flag.Int("position-within-message", 0, "position inside sequencer inbox message")
	delayedPositionFlag := flag.Int("delayed-inbox-position", 0, "position for first delayed inbox message")
	lastBlockFlag := flag.String("last-block-hash", "0000000000000000000000000000000000000000000000000000000000000000", "lastBlockHash")
	flag.Var(&delayedMsgPath, "delayed-inbox", "delayed inbox messages (multiple values)")
	inboxPath := flag.String("inbox", "", "file to load sequencer message")
	preimagesPath := flag.String("preimages", "", "file to load preimages from")
	flag.Parse()

	seqMsgPos = uint64(*seqMsgPosFlag)
	posWithinMsg = uint64(*posWithinMsgFlag)
	delayedMsgFirstPos = uint64(*delayedPositionFlag)
	lastBlockHash = common.HexToHash(*lastBlockFlag)
	for _, path := range delayedMsgPath {
		msg, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		delayedMsgs = append(delayedMsgs, msg)
	}
	if *inboxPath != "" {
		msg, err := os.ReadFile(*inboxPath)
		if err != nil {
			panic(err)
		}
		seqMsg = msg
	}
	if *preimagesPath != "" {
		parsePreimageBytes(*preimagesPath)
	}
}

func StubFinal() {
	log.Info("End state", "lastblockHash", lastBlockHash, "InboxPosition", seqMsgPos+seqAdvanced, "positionWithinMessage", posWithinMsg)
}

func GetLastBlockHash() (hash common.Hash) {
	return lastBlockHash
}

func ReadHotShotCommitment(h uint64) [32]byte {
	return hotShotCommitment

}

func GetHotShotAvailability(l1Height uint64) bool {
	return true
}

func GetEspressoHeight() uint64 {
	return espressoHeight
}

func SetEspressoHeight(h uint64) {
	espressoHeight = h
}

func ReadInboxMessage(msgNum uint64) []byte {
	if msgNum != seqMsgPos {
		panic(fmt.Sprintf("trying to read bad msg %d", msgNum))
	}
	return seqMsg
}

func ReadDelayedInboxMessage(seqNum uint64) []byte {
	if seqNum < delayedMsgFirstPos || (int(seqNum-delayedMsgFirstPos) > len(delayedMsgs)) {
		panic(fmt.Sprintf("trying to read bad delayed msg %d", seqNum))
	}
	return delayedMsgs[seqNum-delayedMsgFirstPos]
}

func AdvanceInboxMessage() {
	seqAdvanced++
}

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	val, ok := preimages[hash]
	if !ok {
		return []byte{}, errors.New("preimage not found")
	}
	return val, nil
}

func SetLastBlockHash(hash [32]byte) {
	lastBlockHash = hash
}

func SetSendRoot(hash [32]byte) {
}

func GetPositionWithinMessage() uint64 {
	return posWithinMsg
}

func SetPositionWithinMessage(pos uint64) {
	posWithinMsg = pos
}

func GetInboxPosition() uint64 {
	return seqMsgPos
}
