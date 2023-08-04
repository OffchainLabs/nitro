package api

import (
	"math/big"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/ethereum/go-ethereum/common"
)

type Assertion struct {
	ConfirmPeriodBlocks uint64                 `json:"confirmPeriodBlocks"`
	RequiredStake       string                 `json:"requiredStake"`
	ParentAssertionHash common.Hash            `json:"parentAssertionHash"`
	InboxMaxCount       string                 `json:"inboxMaxCount"`
	AfterInboxBatchAcc  common.Hash            `json:"afterInboxBatchAcc"`
	AssertionHash       common.Hash            `json:"assertionHash"`
	WasmModuleRoot      common.Hash            `json:"wasmModuleRoot"`
	ChallengeManager    common.Address         `json:"challengeManager"`
	CreationBlock       uint64                 `json:"creationBlock"`
	TransactionHash     common.Hash            `json:"transactionHash"`
	L2State             protocol.GoGlobalState `json:"L2State"`
}

func AssertionCreatedInfoToAssertion(aci *protocol.AssertionCreatedInfo) *Assertion {
	if aci == nil {
		return nil
	}

	return &Assertion{
		ConfirmPeriodBlocks: aci.ConfirmPeriodBlocks,
		RequiredStake:       big.NewInt(0).Set(aci.RequiredStake).String(),
		ParentAssertionHash: aci.ParentAssertionHash,
		InboxMaxCount:       big.NewInt(0).Set(aci.InboxMaxCount).String(),
		AfterInboxBatchAcc:  aci.AfterInboxBatchAcc,
		AssertionHash:       aci.AssertionHash,
		WasmModuleRoot:      aci.WasmModuleRoot,
		ChallengeManager:    aci.ChallengeManager,
		CreationBlock:       aci.CreationBlock,
		TransactionHash:     aci.TransactionHash,
		L2State:             protocol.GoGlobalStateFromSolidity(aci.AfterState.GlobalState),
	}
}
