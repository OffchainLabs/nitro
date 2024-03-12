package api

import (
	"reflect"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/ethereum/go-ethereum/common"
)

type JsonAssertion struct {
	Hash                     common.Hash            `json:"hash" db:"Hash"`
	ConfirmPeriodBlocks      uint64                 `json:"confirmPeriodBlocks" db:"ConfirmPeriodBlocks"`
	RequiredStake            string                 `json:"requiredStake" db:"RequiredStake"`
	ParentAssertionHash      common.Hash            `json:"parentAssertionHash" db:"ParentAssertionHash"`
	InboxMaxCount            string                 `json:"inboxMaxCount" db:"InboxMaxCount"`
	AfterInboxBatchAcc       common.Hash            `json:"afterInboxBatchAcc" db:"AfterInboxBatchAcc"`
	WasmModuleRoot           common.Hash            `json:"wasmModuleRoot" db:"WasmModuleRoot"`
	ChallengeManager         common.Address         `json:"challengeManager" db:"ChallengeManager"`
	CreationBlock            uint64                 `json:"creationBlock" db:"CreationBlock"`
	TransactionHash          common.Hash            `json:"transactionHash" db:"TransactionHash"`
	BeforeStateBlockHash     common.Hash            `json:"beforeStateBlockHash" db:"BeforeStateBlockHash"`
	BeforeStateSendRoot      common.Hash            `json:"beforeStateSendRoot" db:"BeforeStateSendRoot"`
	BeforeStateBatch         uint64                 `json:"beforeStateBatch" db:"BeforeStateBatch"`
	BeforeStatePosInBatch    uint64                 `json:"beforeStatePosInBatch" db:"BeforeStatePosInBatch"`
	BeforeStateMachineStatus protocol.MachineStatus `json:"beforeStateMachineStatus" db:"BeforeStateMachineStatus"`
	AfterStateBlockHash      common.Hash            `json:"afterStateBlockHash" db:"AfterStateBlockHash"`
	AfterStateSendRoot       common.Hash            `json:"afterStateSendRoot" db:"AfterStateSendRoot"`
	AfterStateBatch          uint64                 `json:"afterStateBatch" db:"AfterStateBatch"`
	AfterStatePosInBatch     uint64                 `json:"afterStatePosInBatch" db:"AfterStatePosInBatch"`
	AfterStateMachineStatus  protocol.MachineStatus `json:"afterStateMachineStatus" db:"AfterStateMachineStatus"`
	FirstChildBlock          *uint64                `json:"firstChildBlock" db:"FirstChildBlock"`
	SecondChildBlock         *uint64                `json:"secondChildBlock" db:"SecondChildBlock"`
	IsFirstChild             bool                   `json:"isFirstChild" db:"IsFirstChild"`
	Status                   string                 `json:"status" db:"Status"`
	LastUpdatedAt            time.Time              `json:"lastUpdatedAt" db:"LastUpdatedAt"`
}

type JsonEdge struct {
	Id                common.Hash    `json:"id" db:"Id"`
	ChallengeLevel    uint8          `json:"challengeLevel" db:"ChallengeLevel"`
	StartHistoryRoot  common.Hash    `json:"startHistoryRoot" db:"StartHistoryRoot"`
	StartHeight       uint64         `json:"startHeight" db:"StartHeight"`
	EndHistoryRoot    common.Hash    `json:"endHistoryRoot" db:"EndHistoryRoot"`
	EndHeight         uint64         `json:"endHeight" db:"EndHeight"`
	CreatedAtBlock    uint64         `json:"createdAtBlock" db:"CreatedAtBlock"`
	MutualId          common.Hash    `json:"mutualId" db:"MutualId"`
	OriginId          common.Hash    `json:"originId" db:"OriginId"`
	ClaimId           common.Hash    `json:"claimId" db:"ClaimId"`
	HasChildren       bool           `json:"hasChildren" db:"HasChildren"`
	LowerChildId      common.Hash    `json:"lowerChildId" db:"LowerChildId"`
	UpperChildId      common.Hash    `json:"upperChildId" db:"UpperChildId"`
	MiniStaker        common.Address `json:"miniStaker" db:"MiniStaker"`
	AssertionHash     common.Hash    `json:"assertionHash" db:"AssertionHash"`
	TimeUnrivaled     uint64         `json:"timeUnrivaled" db:"TimeUnrivaled"`
	HasRival          bool           `json:"hasRival" db:"HasRival"`
	Status            string         `json:"status" db:"Status"`
	HasLengthOneRival bool           `json:"hasLengthOneRival" db:"HasLengthOneRival"`
	LastUpdatedAt     time.Time      `json:"lastUpdatedAt" db:"LastUpdatedAt"`
	// Honest validator's point of view
	Ancestors           []common.Hash `json:"ancestors"`
	RawAncestors        string        `json:"-" db:"RawAncestors"`
	IsRoyal             bool          `json:"isRoyal" db:"IsRoyal"`
	CumulativePathTimer uint64        `json:"cumulativePathTimer" db:"CumulativePathTimer"`
	InheritedTimer      uint64        `json:"inheritedTimer" db:"InheritedTimer"`
	RefersTo            string        `json:"refersTo" db:"RefersTo"`
	FSMState            string        `json:"fsmState"`
	FSMError            string        `json:"fsmError"`
}

type JsonTrackedRoyalEdge struct {
	Id               common.Hash    `json:"id"`
	ChallengeLevel   uint8          `json:"challengeLevel"`
	StartHistoryRoot common.Hash    `json:"startHistoryRoot"`
	StartHeight      uint64         `json:"startHeight"`
	EndHistoryRoot   common.Hash    `json:"endHistoryRoot"`
	EndHeight        uint64         `json:"endHeight"`
	CreatedAtBlock   uint64         `json:"createdAtBlock"`
	MutualId         common.Hash    `json:"mutualId"`
	OriginId         common.Hash    `json:"originId"`
	ClaimId          common.Hash    `json:"claimId"`
	MiniStaker       common.Address `json:"miniStaker" db:"MiniStaker"`
	AssertionHash    common.Hash    `json:"assertionHash" db:"AssertionHash"`
	TimeUnrivaled    uint64         `json:"timeUnrivaled" db:"TimeUnrivaled"`
	HasRival         bool           `json:"hasRival" db:"HasRival"`
	Ancestors        []common.Hash  `json:"ancestors"`
}

type JsonEdgesByChallengedAssertion struct {
	AssertionHash common.Hash             `json:"challengedAssertionHash"`
	RoyalEdges    []*JsonTrackedRoyalEdge `json:"royalEdges"`
}

type JsonMiniStakes struct {
	ChallengedAssertionHash common.Hash                                      `json:"challengedAssertionHash"`
	StakesByLvlAndOrigin    map[protocol.ChallengeLevel][]*JsonMiniStakeInfo `json:"stakesByLvlAndOrigin"`
}

type JsonMiniStakeInfo struct {
	ChallengeOriginId  common.Hash      `json:"challengeOriginId"`
	StakerAddresses    []common.Address `json:"stakerAddresses"`
	NumberOfMiniStakes uint64           `json:"numberOfMiniStakes"`
}

type JsonCollectMachineHashes struct {
	WasmModuleRoot       common.Hash `json:"wasmModuleRoot" db:"WasmModuleRoot"`
	FromBatch            uint64      `json:"fromBatch" db:"FromBatch"`
	BlockChallengeHeight uint64      `json:"blockChallengeHeight" db:"BlockChallengeHeight"`
	StepHeights          []uint64    `json:"stepHeights"`
	RawStepHeights       string      `json:"-" db:"RawStepHeights"`
	NumDesiredHashes     uint64      `json:"numDesiredHashes" db:"NumDesiredHashes"`
	MachineStartIndex    uint64      `json:"machineStartIndex" db:"MachineStartIndex"`
	StepSize             uint64      `json:"stepSize" db:"StepSize"`
	StartTime            time.Time   `json:"startTime" db:"StartTime"`
	FinishTime           *time.Time  `json:"finishTime" db:"FinishTime"`
}

func IsNil(i any) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}
