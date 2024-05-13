package validator

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/nitro/arbutil"
)

type BatchInfo struct {
	Number    uint64
	BlockHash common.Hash
	Data      []byte
}

type ValidationInput struct {
	Id            uint64
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	Preimages     map[arbutil.PreimageType]map[common.Hash][]byte
	UserWasms     state.UserWasms
	BatchInfo     []BatchInfo
	DelayedMsg    []byte
	StartState    GoGlobalState
	DebugChain    bool
}

func (b BatchInfo) String() string {
	return fmt.Sprintf("Number: %d, Data: %x", b.Number, b.Data)
}

func (v *ValidationInput) String() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("Id: %d\n", v.Id))
	buf.WriteString(fmt.Sprintf("HasDelayedMsg: %v\n", v.HasDelayedMsg))
	buf.WriteString(fmt.Sprintf("DelayedMsgNr: %d\n", v.DelayedMsgNr))

	// Preimages
	buf.WriteString("Preimages:\n")
	for t, pmap := range v.Preimages {
		for h, data := range pmap {
			buf.WriteString(fmt.Sprintf("\tType: %d, Hash: %s, Data: %x\n", t, h.Hex(), data))
		}
	}

	// BatchInfo
	buf.WriteString("BatchInfo:\n")
	for _, bi := range v.BatchInfo {
		buf.WriteString(fmt.Sprintf("\t%s\n", bi))
	}

	buf.WriteString(fmt.Sprintf("DelayedMsg: %x\n", v.DelayedMsg))
	buf.WriteString(fmt.Sprintf("StartState: %s\n", v.StartState))

	return buf.String()
}
