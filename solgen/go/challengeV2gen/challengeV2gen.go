// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package challengeV2gen

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// ChallengeEdge is an auto generated low-level Go binding around an user-defined struct.
type ChallengeEdge struct {
	OriginId         [32]byte
	StartHistoryRoot [32]byte
	StartHeight      *big.Int
	EndHistoryRoot   [32]byte
	EndHeight        *big.Int
	LowerChildId     [32]byte
	UpperChildId     [32]byte
	CreatedAtBlock   *big.Int
	ClaimId          [32]byte
	Staker           common.Address
	Status           uint8
	EType            uint8
	Refunded         bool
}

// ConfigData is an auto generated low-level Go binding around an user-defined struct.
type ConfigData struct {
	WasmModuleRoot      [32]byte
	RequiredStake       *big.Int
	ChallengeManager    common.Address
	ConfirmPeriodBlocks uint64
	NextInboxPosition   uint64
}

// CreateEdgeArgs is an auto generated low-level Go binding around an user-defined struct.
type CreateEdgeArgs struct {
	EdgeType       uint8
	EndHistoryRoot [32]byte
	EndHeight      *big.Int
	ClaimId        [32]byte
	PrefixProof    []byte
	Proof          []byte
}

// ExecutionState is an auto generated low-level Go binding around an user-defined struct.
type ExecutionState struct {
	GlobalState   GlobalState
	MachineStatus uint8
}

// ExecutionStateData is an auto generated low-level Go binding around an user-defined struct.
type ExecutionStateData struct {
	ExecutionState    ExecutionState
	PrevAssertionHash [32]byte
	InboxAcc          [32]byte
}

// GlobalState is an auto generated low-level Go binding around an user-defined struct.
type GlobalState struct {
	Bytes32Vals [2][32]byte
	U64Vals     [2]uint64
}

// OneStepData is an auto generated low-level Go binding around an user-defined struct.
type OneStepData struct {
	BeforeHash [32]byte
	Proof      []byte
}

// EdgeChallengeManagerMetaData contains all meta data concerning the EdgeChallengeManager contract.
var EdgeChallengeManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AssertionHashEmpty\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"h1\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"h2\",\"type\":\"bytes32\"}],\"name\":\"AssertionHashMismatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AssertionNoSibling\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AssertionNotPending\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"lowerChildId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"upperChildId\",\"type\":\"bytes32\"}],\"name\":\"ChildrenAlreadySet\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"argType\",\"type\":\"uint8\"},{\"internalType\":\"enumEdgeType\",\"name\":\"claimType\",\"type\":\"uint8\"}],\"name\":\"ClaimEdgeInvalidType\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"}],\"name\":\"ClaimEdgeNotLengthOneRival\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ClaimEdgeNotPending\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"EdgeAlreadyExists\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"EdgeAlreadyRefunded\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimingEdgeId\",\"type\":\"bytes32\"}],\"name\":\"EdgeClaimMismatch\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"lowerChildId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"upperChildId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"ancestorEdgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"}],\"name\":\"EdgeNotAncestor\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"enumEdgeStatus\",\"name\":\"\",\"type\":\"uint8\"}],\"name\":\"EdgeNotConfirmed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"EdgeNotExists\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"}],\"name\":\"EdgeNotLayerZero\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"}],\"name\":\"EdgeNotLengthOne\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"enumEdgeStatus\",\"name\":\"status\",\"type\":\"uint8\"}],\"name\":\"EdgeNotPending\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId1\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"edgeId2\",\"type\":\"bytes32\"},{\"internalType\":\"enumEdgeType\",\"name\":\"type1\",\"type\":\"uint8\"},{\"internalType\":\"enumEdgeType\",\"name\":\"type2\",\"type\":\"uint8\"}],\"name\":\"EdgeTypeInvalid\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"eType\",\"type\":\"uint8\"}],\"name\":\"EdgeTypeNotBlock\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"eType\",\"type\":\"uint8\"}],\"name\":\"EdgeTypeNotSmallStep\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"EdgeUnrivaled\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyAssertionChain\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyChallengePeriod\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyClaimId\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyEdgeSpecificProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyEndMachineStatus\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyEndRoot\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyFirstRival\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyOneStepProofEntry\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyOriginId\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyPrefixProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyStakeReceiver\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyStaker\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyStartMachineStatus\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyStartRoot\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"h1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"h2\",\"type\":\"uint256\"}],\"name\":\"HeightDiffLtTwo\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"totalBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"thresholdBlocks\",\"type\":\"uint256\"}],\"name\":\"InsufficientConfirmationBlocks\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"actualHeight\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expectedHeight\",\"type\":\"uint256\"}],\"name\":\"InvalidEndHeight\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"start\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"end\",\"type\":\"uint256\"}],\"name\":\"InvalidHeights\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"actualLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxProofLength\",\"type\":\"uint256\"}],\"name\":\"MerkleProofTooLong\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"val\",\"type\":\"uint256\"}],\"name\":\"NotPowerOfTwo\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"mutualId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"}],\"name\":\"OriginIdMutualIdMismatch\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"mutualId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"enumEdgeType\",\"name\":\"eType\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"hasRival\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"isLayerZero\",\"type\":\"bool\"}],\"name\":\"EdgeAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"lowerChildId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"upperChildId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"lowerChildAlreadyExists\",\"type\":\"bool\"}],\"name\":\"EdgeBisected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"mutualId\",\"type\":\"bytes32\"}],\"name\":\"EdgeConfirmedByChildren\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"mutualId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"claimingEdgeId\",\"type\":\"bytes32\"}],\"name\":\"EdgeConfirmedByClaim\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"mutualId\",\"type\":\"bytes32\"}],\"name\":\"EdgeConfirmedByOneStepProof\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"mutualId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalTimeUnrivaled\",\"type\":\"uint256\"}],\"name\":\"EdgeConfirmedByTime\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"mutualId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"stakeToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"stakeAmount\",\"type\":\"uint256\"}],\"name\":\"EdgeRefunded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"LAYERZERO_BIGSTEPEDGE_HEIGHT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"LAYERZERO_BLOCKEDGE_HEIGHT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"LAYERZERO_SMALLSTEPEDGE_HEIGHT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"assertionChain\",\"outputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"bisectionHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"bisectEdge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"name\":\"calculateEdgeId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"}],\"name\":\"calculateMutualId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"challengePeriodBlocks\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"confirmEdgeByChildren\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimingEdgeId\",\"type\":\"bytes32\"}],\"name\":\"confirmEdgeByClaim\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"internalType\":\"structOneStepData\",\"name\":\"oneStepData\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"requiredStake\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"confirmPeriodBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nextInboxPosition\",\"type\":\"uint64\"}],\"internalType\":\"structConfigData\",\"name\":\"prevConfig\",\"type\":\"tuple\"},{\"internalType\":\"bytes32[]\",\"name\":\"beforeHistoryInclusionProof\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"afterHistoryInclusionProof\",\"type\":\"bytes32[]\"}],\"name\":\"confirmEdgeByOneStepProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"ancestorEdges\",\"type\":\"bytes32[]\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"}],\"internalType\":\"structExecutionState\",\"name\":\"executionState\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"prevAssertionHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"inboxAcc\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionStateData\",\"name\":\"claimStateData\",\"type\":\"tuple\"}],\"name\":\"confirmEdgeByTime\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"internalType\":\"structCreateEdgeArgs\",\"name\":\"args\",\"type\":\"tuple\"}],\"name\":\"createLayerZeroEdge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"edgeExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"edgeLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"excessStakeReceiver\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"firstRival\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"getEdge\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"lowerChildId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"upperChildId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"createdAtBlock\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"enumEdgeStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"enumEdgeType\",\"name\":\"eType\",\"type\":\"uint8\"},{\"internalType\":\"bool\",\"name\":\"refunded\",\"type\":\"bool\"}],\"internalType\":\"structChallengeEdge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"eType\",\"type\":\"uint8\"}],\"name\":\"getLayerZeroEndHeight\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"getPrevAssertionHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"hasLengthOneRival\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"hasRival\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriodBlocks\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"layerZeroBlockEdgeHeight\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"layerZeroBigStepEdgeHeight\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"layerZeroSmallStepEdgeHeight\",\"type\":\"uint256\"},{\"internalType\":\"contractIERC20\",\"name\":\"_stakeToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_stakeAmount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_excessStakeReceiver\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"oneStepProofEntry\",\"outputs\":[{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"refundStake\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stakeAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stakeToken\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"timeUnrivaled\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b506200001c62000022565b620000e4565b600054610100900460ff16156200008f5760405162461bcd60e51b815260206004820152602760248201527f496e697469616c697a61626c653a20636f6e747261637420697320696e697469604482015266616c697a696e6760c81b606482015260840160405180910390fd5b60005460ff9081161015620000e2576000805460ff191660ff9081179091556040519081527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b565b615f1b80620000f46000396000f3fe608060405234801561001057600080fd5b50600436106101c35760003560e01c80635a48e0f4116100f9578063bce6f54f11610097578063e94e051e11610071578063e94e051e146103c7578063eae0328b146103da578063f8ee77d6146103ed578063fda2892e146103f657600080fd5b8063bce6f54f1461036c578063c32d8c631461038c578063c8bc4e431461039f57600080fd5b8063748926f3116100d3578063748926f314610320578063750e0c0f146103335780638c1b3a4014610346578063908517e91461035957600080fd5b80635a48e0f4146102f157806360c7dc471461030457806364deed591461030d57600080fd5b806342e1aaa81161016657806348923bc51161014057806348923bc51461027d57806348dd2924146102a857806351ed6a30146102bb57806354b64151146102ce57600080fd5b806342e1aaa81461024e578063453e969b1461026157806346c2781a1461027457600080fd5b80631dce5166116101a25780631dce5166146102165780632eaa00431461021f5780633e35f5e814610232578063416e66571461024557600080fd5b80624d8efe146101c857806305fae141146101ee5780630f73bfad14610201575b600080fd5b6101db6101d6366004615017565b610416565b6040519081526020015b60405180910390f35b6101db6101fc366004615063565b610431565b61021461020f36600461509e565b610848565b005b6101db60095481565b61021461022d3660046150c0565b6108a9565b6101db6102403660046150c0565b6108f9565b6101db600a5481565b6101db61025c3660046150d9565b61090c565b61021461026f36600461510b565b6109c2565b6101db60065481565b600854610290906001600160a01b031681565b6040516001600160a01b0390911681526020016101e5565b600754610290906001600160a01b031681565b600454610290906001600160a01b031681565b6102e16102dc3660046150c0565b610d2a565b60405190151581526020016101e5565b6101db6102ff3660046150c0565b610d37565b6101db60055481565b61021461031b3660046152bc565b610d44565b61021461032e3660046150c0565b611114565b6102e16103413660046150c0565b6111ce565b61021461035436600461536c565b6111e7565b6102e16103673660046150c0565b6113a4565b6101db61037a3660046150c0565b60009081526002602052604090205490565b6101db61039a366004615435565b6113b1565b6103b26103ad366004615479565b6113ca565b604080519283526020830191909152016101e5565b600354610290906001600160a01b031681565b6101db6103e83660046150c0565b61153c565b6101db600b5481565b6104096104043660046150c0565b611551565b6040516101e5919061553c565b60006104268787878787876116e3565b979650505050505050565b6040805161010081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052600061048361025c60208601866150d9565b905061048d614f61565b600061049c60208701876150d9565b60028111156104ad576104ad6154f9565b0361075e576104bf60a08601866155f5565b90506000036104e157604051630c9ccac560e41b815260040160405180910390fd5b6000806104f160a08801886155f5565b8101906104fe9190615795565b60075481516020830151604080850151905163f9cee9df60e01b81529598509396506001600160a01b03909216945063f9cee9df936105469360608e0135939160040161586a565b60006040518083038186803b15801561055e57600080fd5b505afa158015610572573d6000803e3d6000fd5b5050600754602080850151865191870151604080890151905163f9cee9df60e01b81526001600160a01b03909516965063f9cee9df95506105b89492939260040161586a565b60006040518083038186803b1580156105d057600080fd5b505afa1580156105e4573d6000803e3d6000fd5b50506040805160c08101825260608b01358082526020868101519083015260075483517fe531d8c7000000000000000000000000000000000000000000000000000000008152600481019290925291945091840192506001600160a01b03169063e531d8c790602401602060405180830381865afa15801561066a573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061068e9190615891565b151581526007546020848101516040517f56bbc9e600000000000000000000000000000000000000000000000000000000815260048101919091529201916000916001600160a01b0316906356bbc9e690602401602060405180830381865afa1580156106ff573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061072391906158b3565b11815283516020820152825160409091015260085490935061075590600190899086906001600160a01b031688611728565b9450505061077e565b60085461077b90600190879084906001600160a01b031686611728565b92505b6004546005546001600160a01b0390911690811580159061079e57508015155b156107da5760008560c001516107b457306107c1565b6003546001600160a01b03165b90506107d86001600160a01b0384163383856117b0565b505b8460400151856020015186600001517faa4b66b1ce938c06e2a3f8466bae10ef62e747630e3859889f4719fc6427b5a4886060015189608001518a60a001518b60c001518c60e001516040516108349594939291906158cc565b60405180910390a450509151949350505050565b61085460018383611867565b600082815260016020526040902061086b906119c6565b827fb924f3aa473645c7cf5b10262f927ae4ccf869d7fc239c17144b0c67490d1c738360405161089d91815260200190565b60405180910390a35050565b6108b46001826119f6565b60008181526001602052604090206108cb906119c6565b60405182907f0d27fcaf1adc41547a5cfc99d2364f6c0dc7e81c9fc3fe8cb38abb409b48358a90600090a350565b6000610906600183611b90565b92915050565b600080826002811115610921576109216154f9565b0361092e57505060095490565b6001826002811115610942576109426154f9565b0361094f575050600a5490565b6002826002811115610963576109636154f9565b03610970575050600b5490565b60405162461bcd60e51b815260206004820152601660248201527f556e7265636f676e69736564206564676520747970650000000000000000000060448201526064015b60405180910390fd5b919050565b600054610100900460ff16158080156109e25750600054600160ff909116105b806109fc5750303b1580156109fc575060005460ff166001145b610a6e5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016109b4565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790558015610aaf576000805461ff0019166101001790555b6001600160a01b038a16610aef576040517fc83e086200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6007805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038c8116919091179091558816610b54576040517ffb60b0ef00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6008805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038a161790556000899003610bb7576040517f8a0eedcc00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60068990556004805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038681169190911790915560058490558216610c26576040517f0f0ec8a800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6003805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038416179055610c5787611d01565b610c7757604051633abfb6ff60e21b8152600481018890526024016109b4565b6009879055610c8586611d01565b610ca557604051633abfb6ff60e21b8152600481018790526024016109b4565b600a869055610cb385611d01565b610cd357604051633abfb6ff60e21b8152600481018690526024016109b4565b600b8590558015610d1e576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b50505050505050505050565b6000610906600183611d2b565b6000610906600183611d60565b600080835111610d545783610d7c565b8260018451610d639190615917565b81518110610d7357610d7361592a565b60200260200101515b90506000610d8b600183611e70565b905060006009820154600160a81b900460ff166002811115610daf57610daf6154f9565b14610df65760098101546040517fec72dc5d0000000000000000000000000000000000000000000000000000000081526109b491600160a81b900460ff1690600401615940565b610dff81611eb4565b610e6157610e0c81611ed8565b600982015460088301546040517fe58c830800000000000000000000000000000000000000000000000000000000815260048101939093526001600160a01b03909116602483015260448201526064016109b4565b60075460088201546040517f3083622800000000000000000000000000000000000000000000000000000000815260009283926001600160a01b0390911691633083622891610eb69160040190815260200190565b602060405180830381865afa158015610ed3573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610ef79190615891565b9050801561109357600754600884015460405163f9cee9df60e01b81526001600160a01b039092169163f9cee9df91610f4091899060a08201359060c083013590600401615953565b60006040518083038186803b158015610f5857600080fd5b505afa158015610f6c573d6000803e3d6000fd5b50506007546040517f1171558500000000000000000000000000000000000000000000000000000000815260a089013560048201526001600160a01b03909116925063117155859150602401602060405180830381865afa158015610fd5573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610ff991906158b3565b6007546040517f56bbc9e600000000000000000000000000000000000000000000000000000000815260a088013560048201526001600160a01b03909116906356bbc9e690602401602060405180830381865afa15801561105e573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061108291906158b3565b61108c9190615917565b9150611098565b600091505b60006110b68888856006546001611f0d90949392919063ffffffff16565b60008981526001602052604090209091506110d0906119c6565b887f2e0808830a22204cb3fb8f8d784b28bc97e9ce2e39d2f9cde2860de0957d68eb8360405161110291815260200190565b60405180910390a35050505050505050565b6000611121600183611e70565b905061112c8161214b565b6004546005546001600160a01b0390911690811580159061114c57508015155b1561116d57600983015461116d906001600160a01b03848116911683612268565b6000848152600160205260409020611184906119c6565b604080516001600160a01b03851681526020810184905286917fa635398959ddb5ce3b14537edfc25b2e671274c9b8cad0f4bd634752e69007b6910160405180910390a350505050565b6000818152600160205260408120600701541515610906565b60006111f4600189611d60565b6007546040517f04972af90000000000000000000000000000000000000000000000000000000081529192506001600160a01b0316906304972af9906112409084908a906004016159cf565b60006040518083038186803b15801561125857600080fd5b505afa15801561126c573d6000803e3d6000fd5b50505050600060405180606001604052808860800160208101906112909190615a42565b67ffffffffffffffff168152600754604080517fe78cea9200000000000000000000000000000000000000000000000000000000815290516020938401936001600160a01b039093169263e78cea9292600480820193918290030181865afa158015611300573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906113249190615a5d565b6001600160a01b0390811682528935602090920191909152600854919250611357916001918c91168b858b8b8b8b6122b1565b600089815260016020526040902061136e906119c6565b6040518a907fe11db4b27bc8c6ea5943ecbb205ae1ca8d56c42c719717aaf8a53d43d0cee7c290600090a3505050505050505050565b6000610906600183612511565b60006113c086868686866125e0565b9695505050505050565b6000806000806000611419898989898080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250600195949392505061261c9050565b81519295509093509150158061148c578260400151836020015184600001517faa4b66b1ce938c06e2a3f8466bae10ef62e747630e3859889f4719fc6427b5a4866060015187608001518860a001518960c001518a60e001516040516114839594939291906158cc565b60405180910390a45b8160400151826020015183600001517faa4b66b1ce938c06e2a3f8466bae10ef62e747630e3859889f4719fc6427b5a4856060015186608001518760a001518860c001518960e001516040516114e69594939291906158cc565b60405180910390a48151604051821515815285908c907f7340510d24b7ec9b5c100f5500d93429d80d00d46f0d18e4e85d0c4cc22b99249060200160405180910390a45051919350909150505b94509492505050565b600061090661154c600184611e70565b612a1c565b6115bf604080516101a081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e081018290526101008101829052610120810182905290610140820190815260200160008152600060209091015290565b6115ca600183611e70565b604080516101a0810182528254815260018084015460208301526002840154928201929092526003830154606082015260048301546080820152600583015460a0820152600683015460c0820152600783015460e0820152600883015461010082015260098301546001600160a01b038116610120830152909291610140840191600160a01b900460ff1690811115611665576116656154f9565b6001811115611676576116766154f9565b81526020016009820160159054906101000a900460ff16600281111561169e5761169e6154f9565b60028111156116af576116af6154f9565b815260099190910154760100000000000000000000000000000000000000000000900460ff16151560209091015292915050565b60006116f287878787876125e0565b60408051602081019290925281018390526060016040516020818303038152906040528051906020012090509695505050505050565b6040805161010081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052908061177688888888612a61565b915091506000611787838987612fd1565b9050600061179683838b613116565b90506117a28a826131b7565b9a9950505050505050505050565b6040516001600160a01b03808516602483015283166044820152606481018290526118619085907f23b872dd00000000000000000000000000000000000000000000000000000000906084015b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff00000000000000000000000000000000000000000000000000000000909316929092179091526134f8565b50505050565b6000828152602084905260409020600701546118985760405162a7b02b60e01b8152600481018390526024016109b4565b6000818152602084905260409020600701546118c95760405162a7b02b60e01b8152600481018390526024016109b4565b6001600082815260208590526040902060090154600160a01b900460ff1660018111156118f8576118f86154f9565b146119375760008181526020849052604090819020600901549051633bc499ed60e21b81526109b4918391600160a01b90910460ff1690600401615a7a565b6119428383836135dd565b60008181526020849052604090206008015482146119aa57600081815260208490526040908190206008015490517fc2adc3e80000000000000000000000000000000000000000000000000000000081526109b4918491600401918252602082015260400190565b60008281526020849052604090206119c190613746565b505050565b60006109068260090160159054906101000a900460ff1683600001548460020154856001015486600401546125e0565b600081815260208390526040902060070154611a275760405162a7b02b60e01b8152600481018290526024016109b4565b60008181526020839052604080822060050154808352912060070154611a625760405162a7b02b60e01b8152600481018290526024016109b4565b6001600082815260208590526040902060090154600160a01b900460ff166001811115611a9157611a916154f9565b14611ad05760008181526020849052604090819020600901549051633bc499ed60e21b81526109b4918391600160a01b90910460ff1690600401615a7a565b60008281526020849052604080822060060154808352912060070154611b0b5760405162a7b02b60e01b8152600481018290526024016109b4565b6001600082815260208690526040902060090154600160a01b900460ff166001811115611b3a57611b3a6154f9565b14611b795760008181526020859052604090819020600901549051633bc499ed60e21b81526109b4918391600160a01b90910460ff1690600401615a7a565b600083815260208590526040902061186190613746565b600081815260208390526040812060070154611bc15760405162a7b02b60e01b8152600481018390526024016109b4565b6000828152602084905260408120611bd8906119c6565b6000818152600186016020526040812054919250819003611c25576040517fda10f67c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040516815539492559053115160ba1b6020820152602901604051602081830303815290604052805190602001208103611c7e57600084815260208690526040902060070154611c759043615917565b92505050610906565b600081815260208690526040902060070154611caf5760405162a7b02b60e01b8152600481018290526024016109b4565b600081815260208690526040808220600790810154878452919092209091015480821115611cec57611ce18183615917565b945050505050610906565b6000945050505050610906565b505092915050565b600081600003611d1357506000919050565b6000611d20600184615917565b929092161592915050565b6000611d378383612511565b8015611d5957506000828152602084905260409020611d5590612a1c565b6001145b9392505050565b600080611d6d8484611e70565b905060026009820154600160a81b900460ff166002811115611d9157611d916154f9565b03611db75780546000908152600185016020526040902054611db38582611e70565b9150505b60016009820154600160a81b900460ff166002811115611dd957611dd96154f9565b03611dff5780546000908152600185016020526040902054611dfb8582611e70565b9150505b60006009820154600160a81b900460ff166002811115611e2157611e216154f9565b14611e685760098101546040517fec72dc5d0000000000000000000000000000000000000000000000000000000081526109b491600160a81b900460ff1690600401615940565b549392505050565b600081815260208390526040812060070154611ea15760405162a7b02b60e01b8152600481018390526024016109b4565b5060009081526020919091526040902090565b600881015460009015801590610906575050600901546001600160a01b0316151590565b60006109068260090160159054906101000a900460ff16836000015484600201548560010154866004015487600301546116e3565b600084815260208690526040812060070154611f3e5760405162a7b02b60e01b8152600481018690526024016109b4565b846000611f4b8883611b90565b905060005b86518110156120e3576000611f7e8a898481518110611f7157611f7161592a565b6020026020010151611e70565b90508381600501541480611f955750838160060154145b15611fd957611fac8a611fa783611ed8565b611b90565b611fb69084615a8e565b9250878281518110611fca57611fca61592a565b602002602001015193506120d0565b600084815260208b905260409020600801548851899084908110611fff57611fff61592a565b60200260200101510361203d576120308a8984815181106120225761202261592a565b6020026020010151866135dd565b611fac8a611fa783611ed8565b83816005015482600601548a858151811061205a5761205a61592a565b60200260200101518d6000016000898152602001908152602001600020600801546040517f6ebd28c90000000000000000000000000000000000000000000000000000000081526004016109b4959493929190948552602085019390935260408401919091526060830152608082015260a00190565b50806120db81615aa1565b915050611f50565b506120ee8582615a8e565b905083811015612134576040517f11a8d4d000000000000000000000000000000000000000000000000000000000815260048101829052602481018590526044016109b4565b600087815260208990526040902061042690613746565b60016009820154600160a01b900460ff16600181111561216d5761216d6154f9565b146121a55761217b81611ed8565b6009820154604051633bc499ed60e21b81526109b49291600160a01b900460ff1690600401615a7a565b6121ae81611eb4565b6121bb57610e0c81611ed8565b6009810154760100000000000000000000000000000000000000000000900460ff161515600103612224576121ef81611ed8565b6040517f307f76690000000000000000000000000000000000000000000000000000000081526004016109b491815260200190565b60090180547fffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffffffff16760100000000000000000000000000000000000000000000179055565b6040516001600160a01b0383166024820152604481018290526119c19084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064016117fd565b60006122bd8a8a611e70565b600290810154915060008a815260208c90526040902060090154600160a81b900460ff1660028111156122f2576122f26154f9565b1461234757600089815260208b90526040908190206009015490517f348aefdf0000000000000000000000000000000000000000000000000000000081526109b491600160a81b900460ff1690600401615940565b600089815260208b90526040902061235e90612a1c565b6001146123b157600089815260208b90526040902061237c90612a1c565b6040517f6b595e500000000000000000000000000000000000000000000000000000000081526004016109b491815260200190565b61240c8a60000160008b8152602001908152602001600020600101548860000135838888808060200260200160405190810160405280939291908181526020018383602002808284376000920191909152506137ea92505050565b60006001600160a01b03891663b5112fd288848b3561242e60208e018e6155f5565b6040518663ffffffff1660e01b815260040161244e959493929190615abb565b602060405180830381865afa15801561246b573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061248f91906158b3565b60008b815260208d905260409020600301549091506124ed90826124b4856001615a8e565b8787808060200260200160405190810160405280939291908181526020018383602002808284376000920191909152506137ea92505050565b60008a815260208c90526040902061250490613746565b5050505050505050505050565b6000818152602083905260408120600701546125425760405162a7b02b60e01b8152600481018390526024016109b4565b6000828152602084905260408120612559906119c6565b60008181526001860160205260408120549192508190036125a6576040517fda10f67c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040516815539492559053115160ba1b602082015260290160408051601f1981840301815291905280516020909101201415949350505050565b600085858585856040516020016125fb959493929190615b1d565b60405160208183030381529060405280519060200120905095945050505050565b6040805161010081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e081018290526040805161010081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e081019190915260008087815260208990526040902060090154600160a01b900460ff1660018111156126cb576126cb6154f9565b1461272357600086815260208890526040908190206009015490517f23f8405d0000000000000000000000000000000000000000000000000000000081526109b4918891600160a01b90910460ff1690600401615a7a565b61272d8787612511565b612766576040517f80e07e45000000000000000000000000000000000000000000000000000000008152600481018790526024016109b4565b6000868152602088905260408120604080516101a0810182528254815260018084015460208301526002840154928201929092526003830154606082015260048301546080820152600583015460a0820152600683015460c0820152600783015460e0820152600883015461010082015260098301546001600160a01b038116610120830152909291610140840191600160a01b900460ff169081111561280f5761280f6154f9565b6001811115612820576128206154f9565b81526020016009820160159054906101000a900460ff166002811115612848576128486154f9565b6002811115612859576128596154f9565b81526020016009820160169054906101000a900460ff1615151515815250509050600061288e82604001518360800151613877565b9050600080878060200190518101906128a79190615ba9565b90925090506128d7896128bb856001615a8e565b606087015160808801516128d0906001615a8e565b8686613924565b50506040805161010081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052600061293a8560000151866020015187604001518d888a6101600151613c47565b905061294581613d43565b600081815260208e9052604090206007015490935061296b576129688c826131b7565b91505b506040805161010081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e081019190915260006129ce86600001518c8789606001518a608001518b6101600151613c47565b90506129da8d826131b7565b915050612a0a8382600001518e60000160008f8152602001908152602001600020613d6c9092919063ffffffff16565b919b909a509098509650505050505050565b60008082600201548360040154612a339190615917565b90508060000361090657612a4683611ed8565b60405162a7b02b60e01b81526004016109b491815260200190565b60408051606080820183526000808352602083015291810191909152600080612a8d60208701876150d9565b6002811115612a9e57612a9e6154f9565b03612dc65760208401518451600003612ae3576040517fe96bc61a00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8451606087013514612b315784516040517f5b177a3c0000000000000000000000000000000000000000000000000000000081526004810191909152606087013560248201526044016109b4565b8460400151612b6c576040517fc169243600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8460600151612ba7576040517fb45d1c3a00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b612bb460a08701876155f5565b9050600003612bd657604051630c9ccac560e41b815260040160405180910390fd5b6000612be560a08801886155f5565b810190612bf29190615795565b50909150600090508660800151602001516002811115612c1457612c146154f9565b03612c4b576040517f46365e5200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008660a00151602001516002811115612c6757612c676154f9565b03612c9e576040517f8999857d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60808601516040517fc39619c40000000000000000000000000000000000000000000000000000000081526000916001600160a01b0388169163c39619c491612ce991600401615c0d565b602060405180830381865afa158015612d06573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612d2a91906158b3565b90506000866001600160a01b031663c39619c48960a001516040518263ffffffff1660e01b8152600401612d5e9190615c0d565b602060405180830381865afa158015612d7b573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612d9f91906158b3565b60408051606081018252938452602084019190915282019290925293509091506115339050565b612dd4868660600135611d2b565b612e10576040517fff6d9bd7000000000000000000000000000000000000000000000000000000008152606086013560048201526024016109b4565b6060850135600090815260208790526040812090612e2d826119c6565b905060006009830154600160a01b900460ff166001811115612e5157612e516154f9565b14612e88576040517f12459ffd00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6009820154612ea090600160a81b900460ff16613dec565b6002811115612eb157612eb16154f9565b612ebe60208901896150d9565b6002811115612ecf57612ecf6154f9565b14612f2457612ee160208801886150d9565b60098301546040517fcb6a33e30000000000000000000000000000000000000000000000000000000081526109b49291600160a81b900460ff1690600401615c1b565b612f3160a08801886155f5565b9050600003612f5357604051630c9ccac560e41b815260040160405180910390fd5b600080808080612f6660a08d018d6155f5565b810190612f739190615c41565b94509450945094509450612f918760010154868960020154866137ea565b612fa58760030154858960040154856137ea565b604051806060016040528086815260200185815260200182815250869850985050505050505050611533565b604080516000808252602082019092528190612ff790612ff2908751613ed8565b613f0e565b905061300283611d01565b61302257604051633abfb6ff60e21b8152600481018490526024016109b4565b8284604001351461306c57604080517fdfcc62bc000000000000000000000000000000000000000000000000000000008152908501356004820152602481018490526044016109b4565b61308884602001358660200151866040013588604001516137ea565b61309560808501856155f5565b90506000036130d0576040517f342a075200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000806130e060808701876155f5565b8101906130ed9190615cdd565b909250905061310b83600160208901356128d060408b013583615a8e565b509095945050505050565b613184604080516101a081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e081018290526101008101829052610120810182905290610140820190815260200160008152600060209091015290565b6131af848460006020860180359060408801359060608901359033906131aa908b6150d9565b6140ae565b949350505050565b6040805161010081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e081018290529061320183613d43565b60008181526020869052604090206007015490915015613250576040517fbcedf3de000000000000000000000000000000000000000000000000000000008152600481018290526024016109b4565b600081815260208581526040918290208551815590850151600180830191909155918501516002820155606085015160038201556080850151600482015560a0850151600582015560c0850151600682015560e0850151600782015561010085015160088201556101208501516009820180546001600160a01b0390921673ffffffffffffffffffffffffffffffffffffffff19831681178255610140880151889590937fffffffffffffffffffffff0000000000000000000000000000000000000000001690911790600160a01b908490811115613331576133316154f9565b02179055506101608201516009820180547fffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffffff16600160a81b83600281111561337b5761337b6154f9565b02179055506101808201518160090160166101000a81548160ff02191690831515021790555090505060006133c884610160015185600001518660400151876020015188608001516125e0565b6000818152600187016020526040812054919250819003613427576040516815539492559053115160ba1b602082015260290160408051601f198184030181529181528151602092830120600085815260018a0190935291205561346f565b6040516815539492559053115160ba1b602082015260290160405160208183030381529060405280519060200120810361346f57600082815260018701602052604090208390555b6040518061010001604052808481526020018381526020018660000151815260200186610100015181526020016134b9886000016000878152602001908152602001600020612a1c565b815260200186610160015160028111156134d5576134d56154f9565b815291151560208301526101009590950151151560409091015250919392505050565b600061354d826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564815250856001600160a01b031661423c9092919063ffffffff16565b8051909150156119c1578080602001905181019061356b9190615891565b6119c15760405162461bcd60e51b815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f7420737563636565640000000000000000000000000000000000000000000060648201526084016109b4565b6000818152602084905260408082205484835291206135fb906119c6565b14613662576000828152602084905260409020613617906119c6565b600082815260208590526040908190205490517fe2e27f87000000000000000000000000000000000000000000000000000000008152600481019290925260248201526044016109b4565b600081815260208490526040902060090154600160a81b900460ff16600281111561368f5761368f6154f9565b6000838152602085905260409020600901546136b490600160a81b900460ff16613dec565b60028111156136c5576136c56154f9565b146119c157600082815260208490526040902060090154829082906136f390600160a81b900460ff16613dec565b600084815260208790526040908190206009015490517fdaac88170000000000000000000000000000000000000000000000000000000081526109b494939291600160a81b900460ff1690600401615d37565b60006009820154600160a01b900460ff166001811115613768576137686154f9565b146137b95761377681611ed8565b60098201546040517f23f8405d0000000000000000000000000000000000000000000000000000000081526109b49291600160a01b900460ff1690600401615a7a565b60090180547fffffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffff16600160a01b179055565b600061381f82848660405160200161380491815260200190565b6040516020818303038152906040528051906020012061424b565b90508085146138705760405162461bcd60e51b815260206004820152601760248201527f496e76616c696420696e636c7573696f6e2070726f6f6600000000000000000060448201526064016109b4565b5050505050565b600060026138858484615917565b10156138c7576040517f9029859400000000000000000000000000000000000000000000000000000000815260048101849052602481018390526044016109b4565b6138d18383615917565b6002036138ea576138e3836001615a8e565b9050610906565b6000836138f8600185615917565b189050600061390682614306565b9050600019811b80613919600187615917565b169695505050505050565b600085116139745760405162461bcd60e51b815260206004820152601460248201527f5072652d73697a652063616e6e6f74206265203000000000000000000000000060448201526064016109b4565b8561397e83613f0e565b146139cb5760405162461bcd60e51b815260206004820152601b60248201527f50726520657870616e73696f6e20726f6f74206d69736d61746368000000000060448201526064016109b4565b846139d583614447565b14613a485760405162461bcd60e51b815260206004820152602160248201527f5072652073697a6520646f6573206e6f74206d6174636820657870616e73696f60448201527f6e0000000000000000000000000000000000000000000000000000000000000060648201526084016109b4565b828510613a975760405162461bcd60e51b815260206004820181905260248201527f5072652073697a65206e6f74206c657373207468616e20706f73742073697a6560448201526064016109b4565b6000859050600080613aac85600087516144a2565b90505b85831015613b6f576000613ac38488614621565b905084518310613b155760405162461bcd60e51b815260206004820152601260248201527f496e646578206f7574206f662072616e6765000000000000000000000000000060448201526064016109b4565b613b398282878681518110613b2c57613b2c61592a565b602002602001015161470b565b91506001811b613b498186615a8e565b945087851115613b5b57613b5b615d6c565b83613b6581615aa1565b9450505050613aaf565b86613b7982613f0e565b14613bec5760405162461bcd60e51b815260206004820152602260248201527f506f737420657870616e73696f6e20726f6f74206e6f7420657175616c20706f60448201527f737400000000000000000000000000000000000000000000000000000000000060648201526084016109b4565b83518214613c3c5760405162461bcd60e51b815260206004820152601660248201527f496e636f6d706c6574652070726f6f662075736167650000000000000000000060448201526064016109b4565b505050505050505050565b613cb5604080516101a081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e081018290526101008101829052610120810182905290610140820190815260200160008152600060209091015290565b613cc28787878787614c8a565b604080516101a081018252888152602081018890529081018690526060810185905260808101849052600060a0820181905260c082018190524360e0830152610100820181905261012082018190526101408201526101608101836002811115613d2e57613d2e6154f9565b81526000602090910152979650505050505050565b6000610906826101600151836000015184604001518560200151866080015187606001516116e3565b6005830154151580613d815750600683015415155b15613ddc57613d8f83611ed8565b600584015460068501546040517f8b0e71d00000000000000000000000000000000000000000000000000000000081526004810193909352602483019190915260448201526064016109b4565b6005830191909155600690910155565b600080826002811115613e0157613e016154f9565b03613e0e57506001919050565b6001826002811115613e2257613e226154f9565b03613e2f57506002919050565b6002826002811115613e4357613e436154f9565b03613e905760405162461bcd60e51b815260206004820152601c60248201527f4e6f206e657874207479706520616674657220536d616c6c537465700000000060448201526064016109b4565b60405162461bcd60e51b815260206004820152601460248201527f556e65787065637465642065646765207479706500000000000000000000000060448201526064016109b4565b6060611d5983600084604051602001613ef391815260200190565b6040516020818303038152906040528051906020012061470b565b600080825111613f605760405162461bcd60e51b815260206004820152601660248201527f456d707479206d65726b6c6520657870616e73696f6e0000000000000000000060448201526064016109b4565b604082511115613fb25760405162461bcd60e51b815260206004820152601a60248201527f4d65726b6c6520657870616e73696f6e20746f6f206c6172676500000000000060448201526064016109b4565b6000805b83518110156140a7576000848281518110613fd357613fd361592a565b60200260200101519050826000801b0361403f57801561403a5780925060018551613ffe9190615917565b821461403a57604051614021908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b614094565b801561405e576040805160208101839052908101849052606001614021565b60405161407b908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b508061409f81615aa1565b915050613fb6565b5092915050565b61411c604080516101a081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e081018290526101008101829052610120810182905290610140820190815260200160008152600060209091015290565b6001600160a01b03831661415c576040517ff289e65700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000849003614197576040517f6932bcfd00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6141a48989898989614c8a565b604051806101a001604052808a81526020018981526020018881526020018781526020018681526020016000801b81526020016000801b8152602001438152602001858152602001846001600160a01b031681526020016000600181111561420e5761420e6154f9565b8152602001836002811115614225576142256154f9565b815260006020909101529998505050505050505050565b60606131af8484600085614d7e565b8251600090610100811115614297576040517ffdac331e0000000000000000000000000000000000000000000000000000000081526004810182905261010060248201526044016109b4565b8260005b828110156142fc5760008782815181106142b7576142b761592a565b60200260200101519050816001901b87166000036142e3578260005280602052604060002092506142f3565b8060005282602052604060002092505b5060010161429b565b5095945050505050565b6000816000036143585760405162461bcd60e51b815260206004820152601c60248201527f5a65726f20686173206e6f207369676e69666963616e7420626974730000000060448201526064016109b4565b700100000000000000000000000000000000821061438357608091821c916143809082615a8e565b90505b6801000000000000000082106143a657604091821c916143a39082615a8e565b90505b64010000000082106143c557602091821c916143c29082615a8e565b90505b6201000082106143e257601091821c916143df9082615a8e565b90505b61010082106143fe57600891821c916143fb9082615a8e565b90505b6010821061441957600491821c916144169082615a8e565b90505b6004821061443457600291821c916144319082615a8e565b90505b600282106109bd57610906600182615a8e565b600080805b83518110156140a7578381815181106144675761446761592a565b60200260200101516000801b1461449057614483816002615e66565b61448d9083615a8e565b91505b8061449a81615aa1565b91505061444c565b60608183106144f35760405162461bcd60e51b815260206004820152601760248201527f5374617274206e6f74206c657373207468616e20656e6400000000000000000060448201526064016109b4565b835182111561456a5760405162461bcd60e51b815260206004820152602160248201527f456e64206e6f74206c657373206f7220657175616c207468616e206c656e677460448201527f680000000000000000000000000000000000000000000000000000000000000060648201526084016109b4565b60006145768484615917565b67ffffffffffffffff81111561458e5761458e61519a565b6040519080825280602002602001820160405280156145b7578160200160208202803683370190505b509050835b83811015614618578581815181106145d6576145d661592a565b60200260200101518286836145eb9190615917565b815181106145fb576145fb61592a565b60209081029190910101528061461081615aa1565b9150506145bc565b50949350505050565b60008183106146725760405162461bcd60e51b815260206004820152601760248201527f5374617274206e6f74206c657373207468616e20656e6400000000000000000060448201526064016109b4565b600061467f838518614306565b90506000600161468f8382615a8e565b6001901b61469d9190615917565b905084811684821681156146b457611ce182614ebb565b80156146c357611ce181614306565b60405162461bcd60e51b815260206004820152601b60248201527f426f7468207920616e64207a2063616e6e6f74206265207a65726f000000000060448201526064016109b4565b60606040831061475d5760405162461bcd60e51b815260206004820152600e60248201527f4c6576656c20746f6f206869676800000000000000000000000000000000000060448201526064016109b4565b60008290036147ae5760405162461bcd60e51b815260206004820152601b60248201527f43616e6e6f7420617070656e6420656d7074792073756274726565000000000060448201526064016109b4565b6040845111156148005760405162461bcd60e51b815260206004820152601a60248201527f4d65726b6c6520657870616e73696f6e20746f6f206c6172676500000000000060448201526064016109b4565b835160000361487f576000614816846001615a8e565b67ffffffffffffffff81111561482e5761482e61519a565b604051908082528060200260200182016040528015614857578160200160208202803683370190505b5090508281858151811061486d5761486d61592a565b60209081029190910101529050611d59565b835183106148f55760405162461bcd60e51b815260206004820152603560248201527f4c6576656c2067726561746572207468616e2068696768657374206c6576656c60448201527f206f662063757272656e7420657870616e73696f6e000000000000000000000060648201526084016109b4565b81600061490186614447565b90506000614910866002615e66565b61491a9083615a8e565b9050600061492783614306565b61493083614306565b1161497e57875167ffffffffffffffff81111561494f5761494f61519a565b604051908082528060200260200182016040528015614978578160200160208202803683370190505b506149ce565b875161498b906001615a8e565b67ffffffffffffffff8111156149a3576149a361519a565b6040519080825280602002602001820160405280156149cc578160200160208202803683370190505b505b9050604081511115614a225760405162461bcd60e51b815260206004820152601c60248201527f417070656e642063726561746573206f76657273697a6520747265650000000060448201526064016109b4565b60005b8851811015614bde5787811015614acc57888181518110614a4857614a4861592a565b60200260200101516000801b14614ac75760405162461bcd60e51b815260206004820152602260248201527f417070656e642061626f7665206c65617374207369676e69666963616e74206260448201527f697400000000000000000000000000000000000000000000000000000000000060648201526084016109b4565b614bcc565b6000859003614b1257888181518110614ae757614ae761592a565b6020026020010151828281518110614b0157614b0161592a565b602002602001018181525050614bcc565b888181518110614b2457614b2461592a565b60200260200101516000801b03614b5c5784828281518110614b4857614b4861592a565b602090810291909101015260009450614bcc565b6000801b828281518110614b7257614b7261592a565b602002602001018181525050888181518110614b9057614b9061592a565b602002602001015185604051602001614bb3929190918252602082015260400190565b6040516020818303038152906040528051906020012094505b80614bd681615aa1565b915050614a25565b508315614c1257838160018351614bf59190615917565b81518110614c0557614c0561592a565b6020026020010181815250505b8060018251614c219190615917565b81518110614c3157614c3161592a565b60200260200101516000801b036104265760405162461bcd60e51b815260206004820152600f60248201527f4c61737420656e747279207a65726f000000000000000000000000000000000060448201526064016109b4565b6000859003614cc5576040517f8d79dbbc00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b828111614d08576040517f2060faf400000000000000000000000000000000000000000000000000000000815260048101849052602481018290526044016109b4565b6000849003614d43576040517f83c683e400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000829003613870576040517f5cb6e5bb00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b606082471015614df65760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c000000000000000000000000000000000000000000000000000060648201526084016109b4565b6001600160a01b0385163b614e4d5760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e747261637400000060448201526064016109b4565b600080866001600160a01b03168587604051614e699190615e96565b60006040518083038185875af1925050503d8060008114614ea6576040519150601f19603f3d011682016040523d82523d6000602084013e614eab565b606091505b5091509150610426828286614f28565b6000808211614f0c5760405162461bcd60e51b815260206004820152601c60248201527f5a65726f20686173206e6f207369676e69666963616e7420626974730000000060448201526064016109b4565b60008280614f1b600182615917565b16189050611d5981614306565b60608315614f37575081611d59565b825115614f475782518084602001fd5b8160405162461bcd60e51b81526004016109b49190615eb2565b6040805160c081018252600080825260208201819052918101829052606081019190915260808101614f91614fa3565b8152602001614f9e614fa3565b905290565b6040518060400160405280614fb6614fc2565b81526020016000905290565b6040518060400160405280614fd5614fde565b8152602001614f9e5b60405180604001604052806002906020820280368337509192915050565b6003811061500957600080fd5b50565b80356109bd81614ffc565b60008060008060008060c0878903121561503057600080fd5b863561503b81614ffc565b9860208801359850604088013597606081013597506080810135965060a00135945092505050565b60006020828403121561507557600080fd5b813567ffffffffffffffff81111561508c57600080fd5b820160c08185031215611d5957600080fd5b600080604083850312156150b157600080fd5b50508035926020909101359150565b6000602082840312156150d257600080fd5b5035919050565b6000602082840312156150eb57600080fd5b8135611d5981614ffc565b6001600160a01b038116811461500957600080fd5b60008060008060008060008060006101208a8c03121561512a57600080fd5b8935615135816150f6565b985060208a0135975060408a013561514c816150f6565b965060608a0135955060808a0135945060a08a0135935060c08a0135615171816150f6565b925060e08a013591506101008a0135615189816150f6565b809150509295985092959850929598565b634e487b7160e01b600052604160045260246000fd5b6040516060810167ffffffffffffffff811182821017156151d3576151d361519a565b60405290565b6040805190810167ffffffffffffffff811182821017156151d3576151d361519a565b604051601f8201601f1916810167ffffffffffffffff811182821017156152255761522561519a565b604052919050565b600067ffffffffffffffff8211156152475761524761519a565b5060051b60200190565b600082601f83011261526257600080fd5b813560206152776152728361522d565b6151fc565b82815260059290921b8401810191818101908684111561529657600080fd5b8286015b848110156152b1578035835291830191830161529a565b509695505050505050565b60008060008385036101208112156152d357600080fd5b84359350602085013567ffffffffffffffff8111156152f157600080fd5b6152fd87828801615251565b93505060e0603f198201121561531257600080fd5b506040840190509250925092565b60008083601f84011261533257600080fd5b50813567ffffffffffffffff81111561534a57600080fd5b6020830191508360208260051b850101111561536557600080fd5b9250929050565b600080600080600080600087890361012081121561538957600080fd5b88359750602089013567ffffffffffffffff808211156153a857600080fd5b908a01906040828d0312156153bc57600080fd5b81985060a0603f19840112156153d157600080fd5b60408b01975060e08b01359250808311156153eb57600080fd5b6153f78c848d01615320565b90975095506101008b013592508691508083111561541457600080fd5b50506154228a828b01615320565b989b979a50959850939692959293505050565b600080600080600060a0868803121561544d57600080fd5b853561545881614ffc565b97602087013597506040870135966060810135965060800135945092505050565b6000806000806060858703121561548f57600080fd5b8435935060208501359250604085013567ffffffffffffffff808211156154b557600080fd5b818701915087601f8301126154c957600080fd5b8135818111156154d857600080fd5b8860208285010111156154ea57600080fd5b95989497505060200194505050565b634e487b7160e01b600052602160045260246000fd5b6002811061551f5761551f6154f9565b9052565b60038110615009576150096154f9565b61551f81615523565b60006101a082019050825182526020830151602083015260408301516040830152606083015160608301526080830151608083015260a083015160a083015260c083015160c083015260e083015160e0830152610100808401518184015250610120808401516155b6828501826001600160a01b03169052565b5050610140808401516155cb8285018261550f565b5050610160808401516155e082850182615533565b50506101808381015180151584830152611cf9565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe184360301811261562a57600080fd5b83018035915067ffffffffffffffff82111561564557600080fd5b60200191503681900382131561536557600080fd5b803567ffffffffffffffff811681146109bd57600080fd5b600082601f83011261568357600080fd5b61568b6151d9565b80604084018581111561569d57600080fd5b845b8181101561310b576156b08161565a565b84526020938401930161569f565b600081830360e08112156156d157600080fd5b6156d96151b0565b915060a08112156156e957600080fd5b6156f16151d9565b60808212156156ff57600080fd5b6157076151d9565b915084601f85011261571857600080fd5b6157206151d9565b80604086018781111561573257600080fd5b865b8181101561574c578035845260209384019301615734565b5081855261575a8882615672565b60208601525050508181526157716080850161500c565b6020820152808352505060a0820135602082015260c0820135604082015292915050565b60008060006101e084860312156157ab57600080fd5b833567ffffffffffffffff8111156157c257600080fd5b6157ce86828701615251565b9350506157de85602086016156be565b91506157ee8561010086016156be565b90509250925092565b805180518360005b600281101561581e5782518252602092830192909101906001016157ff565b505050602090810151906040840160005b600281101561585657835167ffffffffffffffff168252928201929082019060010161582f565b505082015190506119c16080840182615533565b848152610100810161587f60208301866157f7565b60c082019390935260e0015292915050565b6000602082840312156158a357600080fd5b81518015158114611d5957600080fd5b6000602082840312156158c557600080fd5b5051919050565b8581526020810185905260a081016158e385615523565b60408201949094529115156060830152151560809091015292915050565b634e487b7160e01b600052601160045260246000fd5b8181038181111561090657610906615901565b634e487b7160e01b600052603260045260246000fd5b6020810161594d83615523565b91905290565b8481526101008101602060408682850137606083016040870160005b600281101561599d5767ffffffffffffffff61598a8361565a565b168352918301919083019060010161596f565b5050505060808501356159af81614ffc565b6159b881615523565b60a083015260c082019390935260e0015292915050565b600060c082019050838252823560208301526020830135604083015260408301356159f9816150f6565b6001600160a01b038116606084015250615a156060840161565a565b67ffffffffffffffff808216608085015280615a336080870161565a565b1660a085015250509392505050565b600060208284031215615a5457600080fd5b611d598261565a565b600060208284031215615a6f57600080fd5b8151611d59816150f6565b82815260408101611d59602083018461550f565b8082018082111561090657610906615901565b60006000198203615ab457615ab4615901565b5060010190565b855181526001600160a01b0360208701511660208201526040860151604082015284606082015283608082015260c060a08201528160c0820152818360e0830137600081830160e090810191909152601f909201601f19160101949350505050565b615b2686615523565b60f89590951b8552600185019390935260218401919091526041830152606182015260810190565b600082601f830112615b5f57600080fd5b81516020615b6f6152728361522d565b82815260059290921b84018101918181019086841115615b8e57600080fd5b8286015b848110156152b15780518352918301918301615b92565b60008060408385031215615bbc57600080fd5b825167ffffffffffffffff80821115615bd457600080fd5b615be086838701615b4e565b93506020850151915080821115615bf657600080fd5b50615c0385828601615b4e565b9150509250929050565b60a0810161090682846157f7565b60408101615c2884615523565b838252615c3483615523565b8260208301529392505050565b600080600080600060a08688031215615c5957600080fd5b8535945060208601359350604086013567ffffffffffffffff80821115615c7f57600080fd5b615c8b89838a01615251565b94506060880135915080821115615ca157600080fd5b615cad89838a01615251565b93506080880135915080821115615cc357600080fd5b50615cd088828901615251565b9150509295509295909350565b60008060408385031215615cf057600080fd5b823567ffffffffffffffff80821115615d0857600080fd5b615d1486838701615251565b93506020850135915080821115615d2a57600080fd5b50615c0385828601615251565b8481526020810184905260808101615d4e84615523565b836040830152615d5d83615523565b82606083015295945050505050565b634e487b7160e01b600052600160045260246000fd5b600181815b80851115615dbd578160001904821115615da357615da3615901565b80851615615db057918102915b93841c9390800290615d87565b509250929050565b600082615dd457506001610906565b81615de157506000610906565b8160018114615df75760028114615e0157615e1d565b6001915050610906565b60ff841115615e1257615e12615901565b50506001821b610906565b5060208310610133831016604e8410600b8410161715615e40575081810a610906565b615e4a8383615d82565b8060001904821115615e5e57615e5e615901565b029392505050565b6000611d598383615dc5565b60005b83811015615e8d578181015183820152602001615e75565b50506000910152565b60008251615ea8818460208701615e72565b9190910192915050565b6020815260008251806020840152615ed1816040850160208701615e72565b601f01601f1916919091016040019291505056fea2646970667358221220a8678cca71244284fac502d21cc098885c64a366fa488c07557a2eb3c4d2399964736f6c63430008110033",
}

// EdgeChallengeManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use EdgeChallengeManagerMetaData.ABI instead.
var EdgeChallengeManagerABI = EdgeChallengeManagerMetaData.ABI

// EdgeChallengeManagerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use EdgeChallengeManagerMetaData.Bin instead.
var EdgeChallengeManagerBin = EdgeChallengeManagerMetaData.Bin

// DeployEdgeChallengeManager deploys a new Ethereum contract, binding an instance of EdgeChallengeManager to it.
func DeployEdgeChallengeManager(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *EdgeChallengeManager, error) {
	parsed, err := EdgeChallengeManagerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(EdgeChallengeManagerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &EdgeChallengeManager{EdgeChallengeManagerCaller: EdgeChallengeManagerCaller{contract: contract}, EdgeChallengeManagerTransactor: EdgeChallengeManagerTransactor{contract: contract}, EdgeChallengeManagerFilterer: EdgeChallengeManagerFilterer{contract: contract}}, nil
}

// EdgeChallengeManager is an auto generated Go binding around an Ethereum contract.
type EdgeChallengeManager struct {
	EdgeChallengeManagerCaller     // Read-only binding to the contract
	EdgeChallengeManagerTransactor // Write-only binding to the contract
	EdgeChallengeManagerFilterer   // Log filterer for contract events
}

// EdgeChallengeManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type EdgeChallengeManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeChallengeManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type EdgeChallengeManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeChallengeManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type EdgeChallengeManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeChallengeManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type EdgeChallengeManagerSession struct {
	Contract     *EdgeChallengeManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// EdgeChallengeManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type EdgeChallengeManagerCallerSession struct {
	Contract *EdgeChallengeManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// EdgeChallengeManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type EdgeChallengeManagerTransactorSession struct {
	Contract     *EdgeChallengeManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// EdgeChallengeManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type EdgeChallengeManagerRaw struct {
	Contract *EdgeChallengeManager // Generic contract binding to access the raw methods on
}

// EdgeChallengeManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type EdgeChallengeManagerCallerRaw struct {
	Contract *EdgeChallengeManagerCaller // Generic read-only contract binding to access the raw methods on
}

// EdgeChallengeManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type EdgeChallengeManagerTransactorRaw struct {
	Contract *EdgeChallengeManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewEdgeChallengeManager creates a new instance of EdgeChallengeManager, bound to a specific deployed contract.
func NewEdgeChallengeManager(address common.Address, backend bind.ContractBackend) (*EdgeChallengeManager, error) {
	contract, err := bindEdgeChallengeManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManager{EdgeChallengeManagerCaller: EdgeChallengeManagerCaller{contract: contract}, EdgeChallengeManagerTransactor: EdgeChallengeManagerTransactor{contract: contract}, EdgeChallengeManagerFilterer: EdgeChallengeManagerFilterer{contract: contract}}, nil
}

// NewEdgeChallengeManagerCaller creates a new read-only instance of EdgeChallengeManager, bound to a specific deployed contract.
func NewEdgeChallengeManagerCaller(address common.Address, caller bind.ContractCaller) (*EdgeChallengeManagerCaller, error) {
	contract, err := bindEdgeChallengeManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerCaller{contract: contract}, nil
}

// NewEdgeChallengeManagerTransactor creates a new write-only instance of EdgeChallengeManager, bound to a specific deployed contract.
func NewEdgeChallengeManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*EdgeChallengeManagerTransactor, error) {
	contract, err := bindEdgeChallengeManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerTransactor{contract: contract}, nil
}

// NewEdgeChallengeManagerFilterer creates a new log filterer instance of EdgeChallengeManager, bound to a specific deployed contract.
func NewEdgeChallengeManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*EdgeChallengeManagerFilterer, error) {
	contract, err := bindEdgeChallengeManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerFilterer{contract: contract}, nil
}

// bindEdgeChallengeManager binds a generic wrapper to an already deployed contract.
func bindEdgeChallengeManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := EdgeChallengeManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EdgeChallengeManager *EdgeChallengeManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EdgeChallengeManager.Contract.EdgeChallengeManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EdgeChallengeManager *EdgeChallengeManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.EdgeChallengeManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EdgeChallengeManager *EdgeChallengeManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.EdgeChallengeManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EdgeChallengeManager *EdgeChallengeManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EdgeChallengeManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.contract.Transact(opts, method, params...)
}

// LAYERZEROBIGSTEPEDGEHEIGHT is a free data retrieval call binding the contract method 0x416e6657.
//
// Solidity: function LAYERZERO_BIGSTEPEDGE_HEIGHT() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) LAYERZEROBIGSTEPEDGEHEIGHT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "LAYERZERO_BIGSTEPEDGE_HEIGHT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LAYERZEROBIGSTEPEDGEHEIGHT is a free data retrieval call binding the contract method 0x416e6657.
//
// Solidity: function LAYERZERO_BIGSTEPEDGE_HEIGHT() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) LAYERZEROBIGSTEPEDGEHEIGHT() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.LAYERZEROBIGSTEPEDGEHEIGHT(&_EdgeChallengeManager.CallOpts)
}

// LAYERZEROBIGSTEPEDGEHEIGHT is a free data retrieval call binding the contract method 0x416e6657.
//
// Solidity: function LAYERZERO_BIGSTEPEDGE_HEIGHT() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) LAYERZEROBIGSTEPEDGEHEIGHT() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.LAYERZEROBIGSTEPEDGEHEIGHT(&_EdgeChallengeManager.CallOpts)
}

// LAYERZEROBLOCKEDGEHEIGHT is a free data retrieval call binding the contract method 0x1dce5166.
//
// Solidity: function LAYERZERO_BLOCKEDGE_HEIGHT() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) LAYERZEROBLOCKEDGEHEIGHT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "LAYERZERO_BLOCKEDGE_HEIGHT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LAYERZEROBLOCKEDGEHEIGHT is a free data retrieval call binding the contract method 0x1dce5166.
//
// Solidity: function LAYERZERO_BLOCKEDGE_HEIGHT() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) LAYERZEROBLOCKEDGEHEIGHT() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.LAYERZEROBLOCKEDGEHEIGHT(&_EdgeChallengeManager.CallOpts)
}

// LAYERZEROBLOCKEDGEHEIGHT is a free data retrieval call binding the contract method 0x1dce5166.
//
// Solidity: function LAYERZERO_BLOCKEDGE_HEIGHT() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) LAYERZEROBLOCKEDGEHEIGHT() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.LAYERZEROBLOCKEDGEHEIGHT(&_EdgeChallengeManager.CallOpts)
}

// LAYERZEROSMALLSTEPEDGEHEIGHT is a free data retrieval call binding the contract method 0xf8ee77d6.
//
// Solidity: function LAYERZERO_SMALLSTEPEDGE_HEIGHT() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) LAYERZEROSMALLSTEPEDGEHEIGHT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "LAYERZERO_SMALLSTEPEDGE_HEIGHT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LAYERZEROSMALLSTEPEDGEHEIGHT is a free data retrieval call binding the contract method 0xf8ee77d6.
//
// Solidity: function LAYERZERO_SMALLSTEPEDGE_HEIGHT() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) LAYERZEROSMALLSTEPEDGEHEIGHT() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.LAYERZEROSMALLSTEPEDGEHEIGHT(&_EdgeChallengeManager.CallOpts)
}

// LAYERZEROSMALLSTEPEDGEHEIGHT is a free data retrieval call binding the contract method 0xf8ee77d6.
//
// Solidity: function LAYERZERO_SMALLSTEPEDGE_HEIGHT() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) LAYERZEROSMALLSTEPEDGEHEIGHT() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.LAYERZEROSMALLSTEPEDGEHEIGHT(&_EdgeChallengeManager.CallOpts)
}

// AssertionChain is a free data retrieval call binding the contract method 0x48dd2924.
//
// Solidity: function assertionChain() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) AssertionChain(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "assertionChain")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AssertionChain is a free data retrieval call binding the contract method 0x48dd2924.
//
// Solidity: function assertionChain() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) AssertionChain() (common.Address, error) {
	return _EdgeChallengeManager.Contract.AssertionChain(&_EdgeChallengeManager.CallOpts)
}

// AssertionChain is a free data retrieval call binding the contract method 0x48dd2924.
//
// Solidity: function assertionChain() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) AssertionChain() (common.Address, error) {
	return _EdgeChallengeManager.Contract.AssertionChain(&_EdgeChallengeManager.CallOpts)
}

// CalculateEdgeId is a free data retrieval call binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) CalculateEdgeId(opts *bind.CallOpts, edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "calculateEdgeId", edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateEdgeId is a free data retrieval call binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) CalculateEdgeId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.CalculateEdgeId(&_EdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
}

// CalculateEdgeId is a free data retrieval call binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) CalculateEdgeId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.CalculateEdgeId(&_EdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
}

// CalculateMutualId is a free data retrieval call binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) CalculateMutualId(opts *bind.CallOpts, edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "calculateMutualId", edgeType, originId, startHeight, startHistoryRoot, endHeight)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateMutualId is a free data retrieval call binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) CalculateMutualId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.CalculateMutualId(&_EdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight)
}

// CalculateMutualId is a free data retrieval call binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) CalculateMutualId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.CalculateMutualId(&_EdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight)
}

// ChallengePeriodBlocks is a free data retrieval call binding the contract method 0x46c2781a.
//
// Solidity: function challengePeriodBlocks() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) ChallengePeriodBlocks(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "challengePeriodBlocks")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChallengePeriodBlocks is a free data retrieval call binding the contract method 0x46c2781a.
//
// Solidity: function challengePeriodBlocks() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ChallengePeriodBlocks() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.ChallengePeriodBlocks(&_EdgeChallengeManager.CallOpts)
}

// ChallengePeriodBlocks is a free data retrieval call binding the contract method 0x46c2781a.
//
// Solidity: function challengePeriodBlocks() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) ChallengePeriodBlocks() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.ChallengePeriodBlocks(&_EdgeChallengeManager.CallOpts)
}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) EdgeExists(opts *bind.CallOpts, edgeId [32]byte) (bool, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "edgeExists", edgeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) EdgeExists(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.EdgeExists(&_EdgeChallengeManager.CallOpts, edgeId)
}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) EdgeExists(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.EdgeExists(&_EdgeChallengeManager.CallOpts, edgeId)
}

// EdgeLength is a free data retrieval call binding the contract method 0xeae0328b.
//
// Solidity: function edgeLength(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) EdgeLength(opts *bind.CallOpts, edgeId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "edgeLength", edgeId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EdgeLength is a free data retrieval call binding the contract method 0xeae0328b.
//
// Solidity: function edgeLength(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) EdgeLength(edgeId [32]byte) (*big.Int, error) {
	return _EdgeChallengeManager.Contract.EdgeLength(&_EdgeChallengeManager.CallOpts, edgeId)
}

// EdgeLength is a free data retrieval call binding the contract method 0xeae0328b.
//
// Solidity: function edgeLength(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) EdgeLength(edgeId [32]byte) (*big.Int, error) {
	return _EdgeChallengeManager.Contract.EdgeLength(&_EdgeChallengeManager.CallOpts, edgeId)
}

// ExcessStakeReceiver is a free data retrieval call binding the contract method 0xe94e051e.
//
// Solidity: function excessStakeReceiver() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) ExcessStakeReceiver(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "excessStakeReceiver")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ExcessStakeReceiver is a free data retrieval call binding the contract method 0xe94e051e.
//
// Solidity: function excessStakeReceiver() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ExcessStakeReceiver() (common.Address, error) {
	return _EdgeChallengeManager.Contract.ExcessStakeReceiver(&_EdgeChallengeManager.CallOpts)
}

// ExcessStakeReceiver is a free data retrieval call binding the contract method 0xe94e051e.
//
// Solidity: function excessStakeReceiver() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) ExcessStakeReceiver() (common.Address, error) {
	return _EdgeChallengeManager.Contract.ExcessStakeReceiver(&_EdgeChallengeManager.CallOpts)
}

// FirstRival is a free data retrieval call binding the contract method 0xbce6f54f.
//
// Solidity: function firstRival(bytes32 edgeId) view returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) FirstRival(opts *bind.CallOpts, edgeId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "firstRival", edgeId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// FirstRival is a free data retrieval call binding the contract method 0xbce6f54f.
//
// Solidity: function firstRival(bytes32 edgeId) view returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) FirstRival(edgeId [32]byte) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.FirstRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// FirstRival is a free data retrieval call binding the contract method 0xbce6f54f.
//
// Solidity: function firstRival(bytes32 edgeId) view returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) FirstRival(edgeId [32]byte) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.FirstRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 edgeId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8,bool))
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) GetEdge(opts *bind.CallOpts, edgeId [32]byte) (ChallengeEdge, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "getEdge", edgeId)

	if err != nil {
		return *new(ChallengeEdge), err
	}

	out0 := *abi.ConvertType(out[0], new(ChallengeEdge)).(*ChallengeEdge)

	return out0, err

}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 edgeId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8,bool))
func (_EdgeChallengeManager *EdgeChallengeManagerSession) GetEdge(edgeId [32]byte) (ChallengeEdge, error) {
	return _EdgeChallengeManager.Contract.GetEdge(&_EdgeChallengeManager.CallOpts, edgeId)
}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 edgeId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8,bool))
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) GetEdge(edgeId [32]byte) (ChallengeEdge, error) {
	return _EdgeChallengeManager.Contract.GetEdge(&_EdgeChallengeManager.CallOpts, edgeId)
}

// GetLayerZeroEndHeight is a free data retrieval call binding the contract method 0x42e1aaa8.
//
// Solidity: function getLayerZeroEndHeight(uint8 eType) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) GetLayerZeroEndHeight(opts *bind.CallOpts, eType uint8) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "getLayerZeroEndHeight", eType)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLayerZeroEndHeight is a free data retrieval call binding the contract method 0x42e1aaa8.
//
// Solidity: function getLayerZeroEndHeight(uint8 eType) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) GetLayerZeroEndHeight(eType uint8) (*big.Int, error) {
	return _EdgeChallengeManager.Contract.GetLayerZeroEndHeight(&_EdgeChallengeManager.CallOpts, eType)
}

// GetLayerZeroEndHeight is a free data retrieval call binding the contract method 0x42e1aaa8.
//
// Solidity: function getLayerZeroEndHeight(uint8 eType) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) GetLayerZeroEndHeight(eType uint8) (*big.Int, error) {
	return _EdgeChallengeManager.Contract.GetLayerZeroEndHeight(&_EdgeChallengeManager.CallOpts, eType)
}

// GetPrevAssertionHash is a free data retrieval call binding the contract method 0x5a48e0f4.
//
// Solidity: function getPrevAssertionHash(bytes32 edgeId) view returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) GetPrevAssertionHash(opts *bind.CallOpts, edgeId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "getPrevAssertionHash", edgeId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetPrevAssertionHash is a free data retrieval call binding the contract method 0x5a48e0f4.
//
// Solidity: function getPrevAssertionHash(bytes32 edgeId) view returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) GetPrevAssertionHash(edgeId [32]byte) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.GetPrevAssertionHash(&_EdgeChallengeManager.CallOpts, edgeId)
}

// GetPrevAssertionHash is a free data retrieval call binding the contract method 0x5a48e0f4.
//
// Solidity: function getPrevAssertionHash(bytes32 edgeId) view returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) GetPrevAssertionHash(edgeId [32]byte) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.GetPrevAssertionHash(&_EdgeChallengeManager.CallOpts, edgeId)
}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) HasLengthOneRival(opts *bind.CallOpts, edgeId [32]byte) (bool, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "hasLengthOneRival", edgeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) HasLengthOneRival(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.HasLengthOneRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) HasLengthOneRival(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.HasLengthOneRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) HasRival(opts *bind.CallOpts, edgeId [32]byte) (bool, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "hasRival", edgeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) HasRival(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.HasRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) HasRival(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.HasRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// OneStepProofEntry is a free data retrieval call binding the contract method 0x48923bc5.
//
// Solidity: function oneStepProofEntry() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) OneStepProofEntry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "oneStepProofEntry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OneStepProofEntry is a free data retrieval call binding the contract method 0x48923bc5.
//
// Solidity: function oneStepProofEntry() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) OneStepProofEntry() (common.Address, error) {
	return _EdgeChallengeManager.Contract.OneStepProofEntry(&_EdgeChallengeManager.CallOpts)
}

// OneStepProofEntry is a free data retrieval call binding the contract method 0x48923bc5.
//
// Solidity: function oneStepProofEntry() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) OneStepProofEntry() (common.Address, error) {
	return _EdgeChallengeManager.Contract.OneStepProofEntry(&_EdgeChallengeManager.CallOpts)
}

// StakeAmount is a free data retrieval call binding the contract method 0x60c7dc47.
//
// Solidity: function stakeAmount() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) StakeAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "stakeAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StakeAmount is a free data retrieval call binding the contract method 0x60c7dc47.
//
// Solidity: function stakeAmount() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) StakeAmount() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.StakeAmount(&_EdgeChallengeManager.CallOpts)
}

// StakeAmount is a free data retrieval call binding the contract method 0x60c7dc47.
//
// Solidity: function stakeAmount() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) StakeAmount() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.StakeAmount(&_EdgeChallengeManager.CallOpts)
}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) StakeToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "stakeToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) StakeToken() (common.Address, error) {
	return _EdgeChallengeManager.Contract.StakeToken(&_EdgeChallengeManager.CallOpts)
}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) StakeToken() (common.Address, error) {
	return _EdgeChallengeManager.Contract.StakeToken(&_EdgeChallengeManager.CallOpts)
}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) TimeUnrivaled(opts *bind.CallOpts, edgeId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "timeUnrivaled", edgeId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) TimeUnrivaled(edgeId [32]byte) (*big.Int, error) {
	return _EdgeChallengeManager.Contract.TimeUnrivaled(&_EdgeChallengeManager.CallOpts, edgeId)
}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) TimeUnrivaled(edgeId [32]byte) (*big.Int, error) {
	return _EdgeChallengeManager.Contract.TimeUnrivaled(&_EdgeChallengeManager.CallOpts, edgeId)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) BisectEdge(opts *bind.TransactOpts, edgeId [32]byte, bisectionHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "bisectEdge", edgeId, bisectionHistoryRoot, prefixProof)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) BisectEdge(edgeId [32]byte, bisectionHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.BisectEdge(&_EdgeChallengeManager.TransactOpts, edgeId, bisectionHistoryRoot, prefixProof)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) BisectEdge(edgeId [32]byte, bisectionHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.BisectEdge(&_EdgeChallengeManager.TransactOpts, edgeId, bisectionHistoryRoot, prefixProof)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 edgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) ConfirmEdgeByChildren(opts *bind.TransactOpts, edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "confirmEdgeByChildren", edgeId)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 edgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ConfirmEdgeByChildren(edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByChildren(&_EdgeChallengeManager.TransactOpts, edgeId)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 edgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) ConfirmEdgeByChildren(edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByChildren(&_EdgeChallengeManager.TransactOpts, edgeId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) ConfirmEdgeByClaim(opts *bind.TransactOpts, edgeId [32]byte, claimingEdgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "confirmEdgeByClaim", edgeId, claimingEdgeId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ConfirmEdgeByClaim(edgeId [32]byte, claimingEdgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByClaim(&_EdgeChallengeManager.TransactOpts, edgeId, claimingEdgeId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) ConfirmEdgeByClaim(edgeId [32]byte, claimingEdgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByClaim(&_EdgeChallengeManager.TransactOpts, edgeId, claimingEdgeId)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0x8c1b3a40.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, (bytes32,bytes) oneStepData, (bytes32,uint256,address,uint64,uint64) prevConfig, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) ConfirmEdgeByOneStepProof(opts *bind.TransactOpts, edgeId [32]byte, oneStepData OneStepData, prevConfig ConfigData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "confirmEdgeByOneStepProof", edgeId, oneStepData, prevConfig, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0x8c1b3a40.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, (bytes32,bytes) oneStepData, (bytes32,uint256,address,uint64,uint64) prevConfig, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ConfirmEdgeByOneStepProof(edgeId [32]byte, oneStepData OneStepData, prevConfig ConfigData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByOneStepProof(&_EdgeChallengeManager.TransactOpts, edgeId, oneStepData, prevConfig, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0x8c1b3a40.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, (bytes32,bytes) oneStepData, (bytes32,uint256,address,uint64,uint64) prevConfig, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) ConfirmEdgeByOneStepProof(edgeId [32]byte, oneStepData OneStepData, prevConfig ConfigData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByOneStepProof(&_EdgeChallengeManager.TransactOpts, edgeId, oneStepData, prevConfig, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x64deed59.
//
// Solidity: function confirmEdgeByTime(bytes32 edgeId, bytes32[] ancestorEdges, (((bytes32[2],uint64[2]),uint8),bytes32,bytes32) claimStateData) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) ConfirmEdgeByTime(opts *bind.TransactOpts, edgeId [32]byte, ancestorEdges [][32]byte, claimStateData ExecutionStateData) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "confirmEdgeByTime", edgeId, ancestorEdges, claimStateData)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x64deed59.
//
// Solidity: function confirmEdgeByTime(bytes32 edgeId, bytes32[] ancestorEdges, (((bytes32[2],uint64[2]),uint8),bytes32,bytes32) claimStateData) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ConfirmEdgeByTime(edgeId [32]byte, ancestorEdges [][32]byte, claimStateData ExecutionStateData) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByTime(&_EdgeChallengeManager.TransactOpts, edgeId, ancestorEdges, claimStateData)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x64deed59.
//
// Solidity: function confirmEdgeByTime(bytes32 edgeId, bytes32[] ancestorEdges, (((bytes32[2],uint64[2]),uint8),bytes32,bytes32) claimStateData) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) ConfirmEdgeByTime(edgeId [32]byte, ancestorEdges [][32]byte, claimStateData ExecutionStateData) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByTime(&_EdgeChallengeManager.TransactOpts, edgeId, ancestorEdges, claimStateData)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x05fae141.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,bytes,bytes) args) returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) CreateLayerZeroEdge(opts *bind.TransactOpts, args CreateEdgeArgs) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "createLayerZeroEdge", args)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x05fae141.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,bytes,bytes) args) returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) CreateLayerZeroEdge(args CreateEdgeArgs) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.CreateLayerZeroEdge(&_EdgeChallengeManager.TransactOpts, args)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x05fae141.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,bytes,bytes) args) returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) CreateLayerZeroEdge(args CreateEdgeArgs) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.CreateLayerZeroEdge(&_EdgeChallengeManager.TransactOpts, args)
}

// Initialize is a paid mutator transaction binding the contract method 0x453e969b.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodBlocks, address _oneStepProofEntry, uint256 layerZeroBlockEdgeHeight, uint256 layerZeroBigStepEdgeHeight, uint256 layerZeroSmallStepEdgeHeight, address _stakeToken, uint256 _stakeAmount, address _excessStakeReceiver) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) Initialize(opts *bind.TransactOpts, _assertionChain common.Address, _challengePeriodBlocks *big.Int, _oneStepProofEntry common.Address, layerZeroBlockEdgeHeight *big.Int, layerZeroBigStepEdgeHeight *big.Int, layerZeroSmallStepEdgeHeight *big.Int, _stakeToken common.Address, _stakeAmount *big.Int, _excessStakeReceiver common.Address) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "initialize", _assertionChain, _challengePeriodBlocks, _oneStepProofEntry, layerZeroBlockEdgeHeight, layerZeroBigStepEdgeHeight, layerZeroSmallStepEdgeHeight, _stakeToken, _stakeAmount, _excessStakeReceiver)
}

// Initialize is a paid mutator transaction binding the contract method 0x453e969b.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodBlocks, address _oneStepProofEntry, uint256 layerZeroBlockEdgeHeight, uint256 layerZeroBigStepEdgeHeight, uint256 layerZeroSmallStepEdgeHeight, address _stakeToken, uint256 _stakeAmount, address _excessStakeReceiver) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) Initialize(_assertionChain common.Address, _challengePeriodBlocks *big.Int, _oneStepProofEntry common.Address, layerZeroBlockEdgeHeight *big.Int, layerZeroBigStepEdgeHeight *big.Int, layerZeroSmallStepEdgeHeight *big.Int, _stakeToken common.Address, _stakeAmount *big.Int, _excessStakeReceiver common.Address) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.Initialize(&_EdgeChallengeManager.TransactOpts, _assertionChain, _challengePeriodBlocks, _oneStepProofEntry, layerZeroBlockEdgeHeight, layerZeroBigStepEdgeHeight, layerZeroSmallStepEdgeHeight, _stakeToken, _stakeAmount, _excessStakeReceiver)
}

// Initialize is a paid mutator transaction binding the contract method 0x453e969b.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodBlocks, address _oneStepProofEntry, uint256 layerZeroBlockEdgeHeight, uint256 layerZeroBigStepEdgeHeight, uint256 layerZeroSmallStepEdgeHeight, address _stakeToken, uint256 _stakeAmount, address _excessStakeReceiver) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) Initialize(_assertionChain common.Address, _challengePeriodBlocks *big.Int, _oneStepProofEntry common.Address, layerZeroBlockEdgeHeight *big.Int, layerZeroBigStepEdgeHeight *big.Int, layerZeroSmallStepEdgeHeight *big.Int, _stakeToken common.Address, _stakeAmount *big.Int, _excessStakeReceiver common.Address) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.Initialize(&_EdgeChallengeManager.TransactOpts, _assertionChain, _challengePeriodBlocks, _oneStepProofEntry, layerZeroBlockEdgeHeight, layerZeroBigStepEdgeHeight, layerZeroSmallStepEdgeHeight, _stakeToken, _stakeAmount, _excessStakeReceiver)
}

// RefundStake is a paid mutator transaction binding the contract method 0x748926f3.
//
// Solidity: function refundStake(bytes32 edgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) RefundStake(opts *bind.TransactOpts, edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "refundStake", edgeId)
}

// RefundStake is a paid mutator transaction binding the contract method 0x748926f3.
//
// Solidity: function refundStake(bytes32 edgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) RefundStake(edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.RefundStake(&_EdgeChallengeManager.TransactOpts, edgeId)
}

// RefundStake is a paid mutator transaction binding the contract method 0x748926f3.
//
// Solidity: function refundStake(bytes32 edgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) RefundStake(edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.RefundStake(&_EdgeChallengeManager.TransactOpts, edgeId)
}

// EdgeChallengeManagerEdgeAddedIterator is returned from FilterEdgeAdded and is used to iterate over the raw logs and unpacked data for EdgeAdded events raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeAddedIterator struct {
	Event *EdgeChallengeManagerEdgeAdded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EdgeChallengeManagerEdgeAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeChallengeManagerEdgeAdded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EdgeChallengeManagerEdgeAdded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EdgeChallengeManagerEdgeAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeChallengeManagerEdgeAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeChallengeManagerEdgeAdded represents a EdgeAdded event raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeAdded struct {
	EdgeId      [32]byte
	MutualId    [32]byte
	OriginId    [32]byte
	ClaimId     [32]byte
	Length      *big.Int
	EType       uint8
	HasRival    bool
	IsLayerZero bool
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterEdgeAdded is a free log retrieval operation binding the contract event 0xaa4b66b1ce938c06e2a3f8466bae10ef62e747630e3859889f4719fc6427b5a4.
//
// Solidity: event EdgeAdded(bytes32 indexed edgeId, bytes32 indexed mutualId, bytes32 indexed originId, bytes32 claimId, uint256 length, uint8 eType, bool hasRival, bool isLayerZero)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) FilterEdgeAdded(opts *bind.FilterOpts, edgeId [][32]byte, mutualId [][32]byte, originId [][32]byte) (*EdgeChallengeManagerEdgeAddedIterator, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}
	var originIdRule []interface{}
	for _, originIdItem := range originId {
		originIdRule = append(originIdRule, originIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.FilterLogs(opts, "EdgeAdded", edgeIdRule, mutualIdRule, originIdRule)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerEdgeAddedIterator{contract: _EdgeChallengeManager.contract, event: "EdgeAdded", logs: logs, sub: sub}, nil
}

// WatchEdgeAdded is a free log subscription operation binding the contract event 0xaa4b66b1ce938c06e2a3f8466bae10ef62e747630e3859889f4719fc6427b5a4.
//
// Solidity: event EdgeAdded(bytes32 indexed edgeId, bytes32 indexed mutualId, bytes32 indexed originId, bytes32 claimId, uint256 length, uint8 eType, bool hasRival, bool isLayerZero)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) WatchEdgeAdded(opts *bind.WatchOpts, sink chan<- *EdgeChallengeManagerEdgeAdded, edgeId [][32]byte, mutualId [][32]byte, originId [][32]byte) (event.Subscription, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}
	var originIdRule []interface{}
	for _, originIdItem := range originId {
		originIdRule = append(originIdRule, originIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.WatchLogs(opts, "EdgeAdded", edgeIdRule, mutualIdRule, originIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeChallengeManagerEdgeAdded)
				if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeAdded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseEdgeAdded is a log parse operation binding the contract event 0xaa4b66b1ce938c06e2a3f8466bae10ef62e747630e3859889f4719fc6427b5a4.
//
// Solidity: event EdgeAdded(bytes32 indexed edgeId, bytes32 indexed mutualId, bytes32 indexed originId, bytes32 claimId, uint256 length, uint8 eType, bool hasRival, bool isLayerZero)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) ParseEdgeAdded(log types.Log) (*EdgeChallengeManagerEdgeAdded, error) {
	event := new(EdgeChallengeManagerEdgeAdded)
	if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeChallengeManagerEdgeBisectedIterator is returned from FilterEdgeBisected and is used to iterate over the raw logs and unpacked data for EdgeBisected events raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeBisectedIterator struct {
	Event *EdgeChallengeManagerEdgeBisected // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EdgeChallengeManagerEdgeBisectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeChallengeManagerEdgeBisected)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EdgeChallengeManagerEdgeBisected)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EdgeChallengeManagerEdgeBisectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeChallengeManagerEdgeBisectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeChallengeManagerEdgeBisected represents a EdgeBisected event raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeBisected struct {
	EdgeId                  [32]byte
	LowerChildId            [32]byte
	UpperChildId            [32]byte
	LowerChildAlreadyExists bool
	Raw                     types.Log // Blockchain specific contextual infos
}

// FilterEdgeBisected is a free log retrieval operation binding the contract event 0x7340510d24b7ec9b5c100f5500d93429d80d00d46f0d18e4e85d0c4cc22b9924.
//
// Solidity: event EdgeBisected(bytes32 indexed edgeId, bytes32 indexed lowerChildId, bytes32 indexed upperChildId, bool lowerChildAlreadyExists)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) FilterEdgeBisected(opts *bind.FilterOpts, edgeId [][32]byte, lowerChildId [][32]byte, upperChildId [][32]byte) (*EdgeChallengeManagerEdgeBisectedIterator, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var lowerChildIdRule []interface{}
	for _, lowerChildIdItem := range lowerChildId {
		lowerChildIdRule = append(lowerChildIdRule, lowerChildIdItem)
	}
	var upperChildIdRule []interface{}
	for _, upperChildIdItem := range upperChildId {
		upperChildIdRule = append(upperChildIdRule, upperChildIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.FilterLogs(opts, "EdgeBisected", edgeIdRule, lowerChildIdRule, upperChildIdRule)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerEdgeBisectedIterator{contract: _EdgeChallengeManager.contract, event: "EdgeBisected", logs: logs, sub: sub}, nil
}

// WatchEdgeBisected is a free log subscription operation binding the contract event 0x7340510d24b7ec9b5c100f5500d93429d80d00d46f0d18e4e85d0c4cc22b9924.
//
// Solidity: event EdgeBisected(bytes32 indexed edgeId, bytes32 indexed lowerChildId, bytes32 indexed upperChildId, bool lowerChildAlreadyExists)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) WatchEdgeBisected(opts *bind.WatchOpts, sink chan<- *EdgeChallengeManagerEdgeBisected, edgeId [][32]byte, lowerChildId [][32]byte, upperChildId [][32]byte) (event.Subscription, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var lowerChildIdRule []interface{}
	for _, lowerChildIdItem := range lowerChildId {
		lowerChildIdRule = append(lowerChildIdRule, lowerChildIdItem)
	}
	var upperChildIdRule []interface{}
	for _, upperChildIdItem := range upperChildId {
		upperChildIdRule = append(upperChildIdRule, upperChildIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.WatchLogs(opts, "EdgeBisected", edgeIdRule, lowerChildIdRule, upperChildIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeChallengeManagerEdgeBisected)
				if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeBisected", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseEdgeBisected is a log parse operation binding the contract event 0x7340510d24b7ec9b5c100f5500d93429d80d00d46f0d18e4e85d0c4cc22b9924.
//
// Solidity: event EdgeBisected(bytes32 indexed edgeId, bytes32 indexed lowerChildId, bytes32 indexed upperChildId, bool lowerChildAlreadyExists)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) ParseEdgeBisected(log types.Log) (*EdgeChallengeManagerEdgeBisected, error) {
	event := new(EdgeChallengeManagerEdgeBisected)
	if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeBisected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeChallengeManagerEdgeConfirmedByChildrenIterator is returned from FilterEdgeConfirmedByChildren and is used to iterate over the raw logs and unpacked data for EdgeConfirmedByChildren events raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeConfirmedByChildrenIterator struct {
	Event *EdgeChallengeManagerEdgeConfirmedByChildren // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EdgeChallengeManagerEdgeConfirmedByChildrenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeChallengeManagerEdgeConfirmedByChildren)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EdgeChallengeManagerEdgeConfirmedByChildren)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EdgeChallengeManagerEdgeConfirmedByChildrenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeChallengeManagerEdgeConfirmedByChildrenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeChallengeManagerEdgeConfirmedByChildren represents a EdgeConfirmedByChildren event raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeConfirmedByChildren struct {
	EdgeId   [32]byte
	MutualId [32]byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterEdgeConfirmedByChildren is a free log retrieval operation binding the contract event 0x0d27fcaf1adc41547a5cfc99d2364f6c0dc7e81c9fc3fe8cb38abb409b48358a.
//
// Solidity: event EdgeConfirmedByChildren(bytes32 indexed edgeId, bytes32 indexed mutualId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) FilterEdgeConfirmedByChildren(opts *bind.FilterOpts, edgeId [][32]byte, mutualId [][32]byte) (*EdgeChallengeManagerEdgeConfirmedByChildrenIterator, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.FilterLogs(opts, "EdgeConfirmedByChildren", edgeIdRule, mutualIdRule)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerEdgeConfirmedByChildrenIterator{contract: _EdgeChallengeManager.contract, event: "EdgeConfirmedByChildren", logs: logs, sub: sub}, nil
}

// WatchEdgeConfirmedByChildren is a free log subscription operation binding the contract event 0x0d27fcaf1adc41547a5cfc99d2364f6c0dc7e81c9fc3fe8cb38abb409b48358a.
//
// Solidity: event EdgeConfirmedByChildren(bytes32 indexed edgeId, bytes32 indexed mutualId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) WatchEdgeConfirmedByChildren(opts *bind.WatchOpts, sink chan<- *EdgeChallengeManagerEdgeConfirmedByChildren, edgeId [][32]byte, mutualId [][32]byte) (event.Subscription, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.WatchLogs(opts, "EdgeConfirmedByChildren", edgeIdRule, mutualIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeChallengeManagerEdgeConfirmedByChildren)
				if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeConfirmedByChildren", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseEdgeConfirmedByChildren is a log parse operation binding the contract event 0x0d27fcaf1adc41547a5cfc99d2364f6c0dc7e81c9fc3fe8cb38abb409b48358a.
//
// Solidity: event EdgeConfirmedByChildren(bytes32 indexed edgeId, bytes32 indexed mutualId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) ParseEdgeConfirmedByChildren(log types.Log) (*EdgeChallengeManagerEdgeConfirmedByChildren, error) {
	event := new(EdgeChallengeManagerEdgeConfirmedByChildren)
	if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeConfirmedByChildren", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeChallengeManagerEdgeConfirmedByClaimIterator is returned from FilterEdgeConfirmedByClaim and is used to iterate over the raw logs and unpacked data for EdgeConfirmedByClaim events raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeConfirmedByClaimIterator struct {
	Event *EdgeChallengeManagerEdgeConfirmedByClaim // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EdgeChallengeManagerEdgeConfirmedByClaimIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeChallengeManagerEdgeConfirmedByClaim)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EdgeChallengeManagerEdgeConfirmedByClaim)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EdgeChallengeManagerEdgeConfirmedByClaimIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeChallengeManagerEdgeConfirmedByClaimIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeChallengeManagerEdgeConfirmedByClaim represents a EdgeConfirmedByClaim event raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeConfirmedByClaim struct {
	EdgeId         [32]byte
	MutualId       [32]byte
	ClaimingEdgeId [32]byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterEdgeConfirmedByClaim is a free log retrieval operation binding the contract event 0xb924f3aa473645c7cf5b10262f927ae4ccf869d7fc239c17144b0c67490d1c73.
//
// Solidity: event EdgeConfirmedByClaim(bytes32 indexed edgeId, bytes32 indexed mutualId, bytes32 claimingEdgeId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) FilterEdgeConfirmedByClaim(opts *bind.FilterOpts, edgeId [][32]byte, mutualId [][32]byte) (*EdgeChallengeManagerEdgeConfirmedByClaimIterator, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.FilterLogs(opts, "EdgeConfirmedByClaim", edgeIdRule, mutualIdRule)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerEdgeConfirmedByClaimIterator{contract: _EdgeChallengeManager.contract, event: "EdgeConfirmedByClaim", logs: logs, sub: sub}, nil
}

// WatchEdgeConfirmedByClaim is a free log subscription operation binding the contract event 0xb924f3aa473645c7cf5b10262f927ae4ccf869d7fc239c17144b0c67490d1c73.
//
// Solidity: event EdgeConfirmedByClaim(bytes32 indexed edgeId, bytes32 indexed mutualId, bytes32 claimingEdgeId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) WatchEdgeConfirmedByClaim(opts *bind.WatchOpts, sink chan<- *EdgeChallengeManagerEdgeConfirmedByClaim, edgeId [][32]byte, mutualId [][32]byte) (event.Subscription, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.WatchLogs(opts, "EdgeConfirmedByClaim", edgeIdRule, mutualIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeChallengeManagerEdgeConfirmedByClaim)
				if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeConfirmedByClaim", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseEdgeConfirmedByClaim is a log parse operation binding the contract event 0xb924f3aa473645c7cf5b10262f927ae4ccf869d7fc239c17144b0c67490d1c73.
//
// Solidity: event EdgeConfirmedByClaim(bytes32 indexed edgeId, bytes32 indexed mutualId, bytes32 claimingEdgeId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) ParseEdgeConfirmedByClaim(log types.Log) (*EdgeChallengeManagerEdgeConfirmedByClaim, error) {
	event := new(EdgeChallengeManagerEdgeConfirmedByClaim)
	if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeConfirmedByClaim", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeChallengeManagerEdgeConfirmedByOneStepProofIterator is returned from FilterEdgeConfirmedByOneStepProof and is used to iterate over the raw logs and unpacked data for EdgeConfirmedByOneStepProof events raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeConfirmedByOneStepProofIterator struct {
	Event *EdgeChallengeManagerEdgeConfirmedByOneStepProof // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EdgeChallengeManagerEdgeConfirmedByOneStepProofIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeChallengeManagerEdgeConfirmedByOneStepProof)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EdgeChallengeManagerEdgeConfirmedByOneStepProof)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EdgeChallengeManagerEdgeConfirmedByOneStepProofIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeChallengeManagerEdgeConfirmedByOneStepProofIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeChallengeManagerEdgeConfirmedByOneStepProof represents a EdgeConfirmedByOneStepProof event raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeConfirmedByOneStepProof struct {
	EdgeId   [32]byte
	MutualId [32]byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterEdgeConfirmedByOneStepProof is a free log retrieval operation binding the contract event 0xe11db4b27bc8c6ea5943ecbb205ae1ca8d56c42c719717aaf8a53d43d0cee7c2.
//
// Solidity: event EdgeConfirmedByOneStepProof(bytes32 indexed edgeId, bytes32 indexed mutualId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) FilterEdgeConfirmedByOneStepProof(opts *bind.FilterOpts, edgeId [][32]byte, mutualId [][32]byte) (*EdgeChallengeManagerEdgeConfirmedByOneStepProofIterator, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.FilterLogs(opts, "EdgeConfirmedByOneStepProof", edgeIdRule, mutualIdRule)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerEdgeConfirmedByOneStepProofIterator{contract: _EdgeChallengeManager.contract, event: "EdgeConfirmedByOneStepProof", logs: logs, sub: sub}, nil
}

// WatchEdgeConfirmedByOneStepProof is a free log subscription operation binding the contract event 0xe11db4b27bc8c6ea5943ecbb205ae1ca8d56c42c719717aaf8a53d43d0cee7c2.
//
// Solidity: event EdgeConfirmedByOneStepProof(bytes32 indexed edgeId, bytes32 indexed mutualId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) WatchEdgeConfirmedByOneStepProof(opts *bind.WatchOpts, sink chan<- *EdgeChallengeManagerEdgeConfirmedByOneStepProof, edgeId [][32]byte, mutualId [][32]byte) (event.Subscription, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.WatchLogs(opts, "EdgeConfirmedByOneStepProof", edgeIdRule, mutualIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeChallengeManagerEdgeConfirmedByOneStepProof)
				if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeConfirmedByOneStepProof", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseEdgeConfirmedByOneStepProof is a log parse operation binding the contract event 0xe11db4b27bc8c6ea5943ecbb205ae1ca8d56c42c719717aaf8a53d43d0cee7c2.
//
// Solidity: event EdgeConfirmedByOneStepProof(bytes32 indexed edgeId, bytes32 indexed mutualId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) ParseEdgeConfirmedByOneStepProof(log types.Log) (*EdgeChallengeManagerEdgeConfirmedByOneStepProof, error) {
	event := new(EdgeChallengeManagerEdgeConfirmedByOneStepProof)
	if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeConfirmedByOneStepProof", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeChallengeManagerEdgeConfirmedByTimeIterator is returned from FilterEdgeConfirmedByTime and is used to iterate over the raw logs and unpacked data for EdgeConfirmedByTime events raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeConfirmedByTimeIterator struct {
	Event *EdgeChallengeManagerEdgeConfirmedByTime // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EdgeChallengeManagerEdgeConfirmedByTimeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeChallengeManagerEdgeConfirmedByTime)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EdgeChallengeManagerEdgeConfirmedByTime)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EdgeChallengeManagerEdgeConfirmedByTimeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeChallengeManagerEdgeConfirmedByTimeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeChallengeManagerEdgeConfirmedByTime represents a EdgeConfirmedByTime event raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeConfirmedByTime struct {
	EdgeId             [32]byte
	MutualId           [32]byte
	TotalTimeUnrivaled *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterEdgeConfirmedByTime is a free log retrieval operation binding the contract event 0x2e0808830a22204cb3fb8f8d784b28bc97e9ce2e39d2f9cde2860de0957d68eb.
//
// Solidity: event EdgeConfirmedByTime(bytes32 indexed edgeId, bytes32 indexed mutualId, uint256 totalTimeUnrivaled)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) FilterEdgeConfirmedByTime(opts *bind.FilterOpts, edgeId [][32]byte, mutualId [][32]byte) (*EdgeChallengeManagerEdgeConfirmedByTimeIterator, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.FilterLogs(opts, "EdgeConfirmedByTime", edgeIdRule, mutualIdRule)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerEdgeConfirmedByTimeIterator{contract: _EdgeChallengeManager.contract, event: "EdgeConfirmedByTime", logs: logs, sub: sub}, nil
}

// WatchEdgeConfirmedByTime is a free log subscription operation binding the contract event 0x2e0808830a22204cb3fb8f8d784b28bc97e9ce2e39d2f9cde2860de0957d68eb.
//
// Solidity: event EdgeConfirmedByTime(bytes32 indexed edgeId, bytes32 indexed mutualId, uint256 totalTimeUnrivaled)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) WatchEdgeConfirmedByTime(opts *bind.WatchOpts, sink chan<- *EdgeChallengeManagerEdgeConfirmedByTime, edgeId [][32]byte, mutualId [][32]byte) (event.Subscription, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.WatchLogs(opts, "EdgeConfirmedByTime", edgeIdRule, mutualIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeChallengeManagerEdgeConfirmedByTime)
				if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeConfirmedByTime", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseEdgeConfirmedByTime is a log parse operation binding the contract event 0x2e0808830a22204cb3fb8f8d784b28bc97e9ce2e39d2f9cde2860de0957d68eb.
//
// Solidity: event EdgeConfirmedByTime(bytes32 indexed edgeId, bytes32 indexed mutualId, uint256 totalTimeUnrivaled)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) ParseEdgeConfirmedByTime(log types.Log) (*EdgeChallengeManagerEdgeConfirmedByTime, error) {
	event := new(EdgeChallengeManagerEdgeConfirmedByTime)
	if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeConfirmedByTime", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeChallengeManagerEdgeRefundedIterator is returned from FilterEdgeRefunded and is used to iterate over the raw logs and unpacked data for EdgeRefunded events raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeRefundedIterator struct {
	Event *EdgeChallengeManagerEdgeRefunded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EdgeChallengeManagerEdgeRefundedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeChallengeManagerEdgeRefunded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EdgeChallengeManagerEdgeRefunded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EdgeChallengeManagerEdgeRefundedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeChallengeManagerEdgeRefundedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeChallengeManagerEdgeRefunded represents a EdgeRefunded event raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerEdgeRefunded struct {
	EdgeId      [32]byte
	MutualId    [32]byte
	StakeToken  common.Address
	StakeAmount *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterEdgeRefunded is a free log retrieval operation binding the contract event 0xa635398959ddb5ce3b14537edfc25b2e671274c9b8cad0f4bd634752e69007b6.
//
// Solidity: event EdgeRefunded(bytes32 indexed edgeId, bytes32 indexed mutualId, address stakeToken, uint256 stakeAmount)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) FilterEdgeRefunded(opts *bind.FilterOpts, edgeId [][32]byte, mutualId [][32]byte) (*EdgeChallengeManagerEdgeRefundedIterator, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.FilterLogs(opts, "EdgeRefunded", edgeIdRule, mutualIdRule)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerEdgeRefundedIterator{contract: _EdgeChallengeManager.contract, event: "EdgeRefunded", logs: logs, sub: sub}, nil
}

// WatchEdgeRefunded is a free log subscription operation binding the contract event 0xa635398959ddb5ce3b14537edfc25b2e671274c9b8cad0f4bd634752e69007b6.
//
// Solidity: event EdgeRefunded(bytes32 indexed edgeId, bytes32 indexed mutualId, address stakeToken, uint256 stakeAmount)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) WatchEdgeRefunded(opts *bind.WatchOpts, sink chan<- *EdgeChallengeManagerEdgeRefunded, edgeId [][32]byte, mutualId [][32]byte) (event.Subscription, error) {

	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}
	var mutualIdRule []interface{}
	for _, mutualIdItem := range mutualId {
		mutualIdRule = append(mutualIdRule, mutualIdItem)
	}

	logs, sub, err := _EdgeChallengeManager.contract.WatchLogs(opts, "EdgeRefunded", edgeIdRule, mutualIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeChallengeManagerEdgeRefunded)
				if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeRefunded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseEdgeRefunded is a log parse operation binding the contract event 0xa635398959ddb5ce3b14537edfc25b2e671274c9b8cad0f4bd634752e69007b6.
//
// Solidity: event EdgeRefunded(bytes32 indexed edgeId, bytes32 indexed mutualId, address stakeToken, uint256 stakeAmount)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) ParseEdgeRefunded(log types.Log) (*EdgeChallengeManagerEdgeRefunded, error) {
	event := new(EdgeChallengeManagerEdgeRefunded)
	if err := _EdgeChallengeManager.contract.UnpackLog(event, "EdgeRefunded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeChallengeManagerInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerInitializedIterator struct {
	Event *EdgeChallengeManagerInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EdgeChallengeManagerInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeChallengeManagerInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EdgeChallengeManagerInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EdgeChallengeManagerInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeChallengeManagerInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeChallengeManagerInitialized represents a Initialized event raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) FilterInitialized(opts *bind.FilterOpts) (*EdgeChallengeManagerInitializedIterator, error) {

	logs, sub, err := _EdgeChallengeManager.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerInitializedIterator{contract: _EdgeChallengeManager.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *EdgeChallengeManagerInitialized) (event.Subscription, error) {

	logs, sub, err := _EdgeChallengeManager.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeChallengeManagerInitialized)
				if err := _EdgeChallengeManager.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) ParseInitialized(log types.Log) (*EdgeChallengeManagerInitialized, error) {
	event := new(EdgeChallengeManagerInitialized)
	if err := _EdgeChallengeManager.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAssertionChainMetaData contains all meta data concerning the IAssertionChain contract.
var IAssertionChainMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionHash\",\"type\":\"bytes32\"}],\"name\":\"getFirstChildCreationBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionHash\",\"type\":\"bytes32\"}],\"name\":\"getSecondChildCreationBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionHash\",\"type\":\"bytes32\"}],\"name\":\"isFirstChild\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionHash\",\"type\":\"bytes32\"}],\"name\":\"isPending\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionHash\",\"type\":\"bytes32\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"}],\"internalType\":\"structExecutionState\",\"name\":\"state\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"prevAssertionHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"inboxAcc\",\"type\":\"bytes32\"}],\"name\":\"validateAssertionHash\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionHash\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"requiredStake\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"confirmPeriodBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nextInboxPosition\",\"type\":\"uint64\"}],\"internalType\":\"structConfigData\",\"name\":\"configData\",\"type\":\"tuple\"}],\"name\":\"validateConfig\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IAssertionChainABI is the input ABI used to generate the binding from.
// Deprecated: Use IAssertionChainMetaData.ABI instead.
var IAssertionChainABI = IAssertionChainMetaData.ABI

// IAssertionChain is an auto generated Go binding around an Ethereum contract.
type IAssertionChain struct {
	IAssertionChainCaller     // Read-only binding to the contract
	IAssertionChainTransactor // Write-only binding to the contract
	IAssertionChainFilterer   // Log filterer for contract events
}

// IAssertionChainCaller is an auto generated read-only Go binding around an Ethereum contract.
type IAssertionChainCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAssertionChainTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IAssertionChainTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAssertionChainFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IAssertionChainFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAssertionChainSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IAssertionChainSession struct {
	Contract     *IAssertionChain  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IAssertionChainCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IAssertionChainCallerSession struct {
	Contract *IAssertionChainCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// IAssertionChainTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IAssertionChainTransactorSession struct {
	Contract     *IAssertionChainTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// IAssertionChainRaw is an auto generated low-level Go binding around an Ethereum contract.
type IAssertionChainRaw struct {
	Contract *IAssertionChain // Generic contract binding to access the raw methods on
}

// IAssertionChainCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IAssertionChainCallerRaw struct {
	Contract *IAssertionChainCaller // Generic read-only contract binding to access the raw methods on
}

// IAssertionChainTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IAssertionChainTransactorRaw struct {
	Contract *IAssertionChainTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIAssertionChain creates a new instance of IAssertionChain, bound to a specific deployed contract.
func NewIAssertionChain(address common.Address, backend bind.ContractBackend) (*IAssertionChain, error) {
	contract, err := bindIAssertionChain(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IAssertionChain{IAssertionChainCaller: IAssertionChainCaller{contract: contract}, IAssertionChainTransactor: IAssertionChainTransactor{contract: contract}, IAssertionChainFilterer: IAssertionChainFilterer{contract: contract}}, nil
}

// NewIAssertionChainCaller creates a new read-only instance of IAssertionChain, bound to a specific deployed contract.
func NewIAssertionChainCaller(address common.Address, caller bind.ContractCaller) (*IAssertionChainCaller, error) {
	contract, err := bindIAssertionChain(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IAssertionChainCaller{contract: contract}, nil
}

// NewIAssertionChainTransactor creates a new write-only instance of IAssertionChain, bound to a specific deployed contract.
func NewIAssertionChainTransactor(address common.Address, transactor bind.ContractTransactor) (*IAssertionChainTransactor, error) {
	contract, err := bindIAssertionChain(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IAssertionChainTransactor{contract: contract}, nil
}

// NewIAssertionChainFilterer creates a new log filterer instance of IAssertionChain, bound to a specific deployed contract.
func NewIAssertionChainFilterer(address common.Address, filterer bind.ContractFilterer) (*IAssertionChainFilterer, error) {
	contract, err := bindIAssertionChain(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IAssertionChainFilterer{contract: contract}, nil
}

// bindIAssertionChain binds a generic wrapper to an already deployed contract.
func bindIAssertionChain(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IAssertionChainMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IAssertionChain *IAssertionChainRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IAssertionChain.Contract.IAssertionChainCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IAssertionChain *IAssertionChainRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IAssertionChain.Contract.IAssertionChainTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IAssertionChain *IAssertionChainRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IAssertionChain.Contract.IAssertionChainTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IAssertionChain *IAssertionChainCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IAssertionChain.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IAssertionChain *IAssertionChainTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IAssertionChain.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IAssertionChain *IAssertionChainTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IAssertionChain.Contract.contract.Transact(opts, method, params...)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_IAssertionChain *IAssertionChainCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_IAssertionChain *IAssertionChainSession) Bridge() (common.Address, error) {
	return _IAssertionChain.Contract.Bridge(&_IAssertionChain.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_IAssertionChain *IAssertionChainCallerSession) Bridge() (common.Address, error) {
	return _IAssertionChain.Contract.Bridge(&_IAssertionChain.CallOpts)
}

// GetFirstChildCreationBlock is a free data retrieval call binding the contract method 0x11715585.
//
// Solidity: function getFirstChildCreationBlock(bytes32 assertionHash) view returns(uint256)
func (_IAssertionChain *IAssertionChainCaller) GetFirstChildCreationBlock(opts *bind.CallOpts, assertionHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "getFirstChildCreationBlock", assertionHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetFirstChildCreationBlock is a free data retrieval call binding the contract method 0x11715585.
//
// Solidity: function getFirstChildCreationBlock(bytes32 assertionHash) view returns(uint256)
func (_IAssertionChain *IAssertionChainSession) GetFirstChildCreationBlock(assertionHash [32]byte) (*big.Int, error) {
	return _IAssertionChain.Contract.GetFirstChildCreationBlock(&_IAssertionChain.CallOpts, assertionHash)
}

// GetFirstChildCreationBlock is a free data retrieval call binding the contract method 0x11715585.
//
// Solidity: function getFirstChildCreationBlock(bytes32 assertionHash) view returns(uint256)
func (_IAssertionChain *IAssertionChainCallerSession) GetFirstChildCreationBlock(assertionHash [32]byte) (*big.Int, error) {
	return _IAssertionChain.Contract.GetFirstChildCreationBlock(&_IAssertionChain.CallOpts, assertionHash)
}

// GetSecondChildCreationBlock is a free data retrieval call binding the contract method 0x56bbc9e6.
//
// Solidity: function getSecondChildCreationBlock(bytes32 assertionHash) view returns(uint256)
func (_IAssertionChain *IAssertionChainCaller) GetSecondChildCreationBlock(opts *bind.CallOpts, assertionHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "getSecondChildCreationBlock", assertionHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetSecondChildCreationBlock is a free data retrieval call binding the contract method 0x56bbc9e6.
//
// Solidity: function getSecondChildCreationBlock(bytes32 assertionHash) view returns(uint256)
func (_IAssertionChain *IAssertionChainSession) GetSecondChildCreationBlock(assertionHash [32]byte) (*big.Int, error) {
	return _IAssertionChain.Contract.GetSecondChildCreationBlock(&_IAssertionChain.CallOpts, assertionHash)
}

// GetSecondChildCreationBlock is a free data retrieval call binding the contract method 0x56bbc9e6.
//
// Solidity: function getSecondChildCreationBlock(bytes32 assertionHash) view returns(uint256)
func (_IAssertionChain *IAssertionChainCallerSession) GetSecondChildCreationBlock(assertionHash [32]byte) (*big.Int, error) {
	return _IAssertionChain.Contract.GetSecondChildCreationBlock(&_IAssertionChain.CallOpts, assertionHash)
}

// IsFirstChild is a free data retrieval call binding the contract method 0x30836228.
//
// Solidity: function isFirstChild(bytes32 assertionHash) view returns(bool)
func (_IAssertionChain *IAssertionChainCaller) IsFirstChild(opts *bind.CallOpts, assertionHash [32]byte) (bool, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "isFirstChild", assertionHash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsFirstChild is a free data retrieval call binding the contract method 0x30836228.
//
// Solidity: function isFirstChild(bytes32 assertionHash) view returns(bool)
func (_IAssertionChain *IAssertionChainSession) IsFirstChild(assertionHash [32]byte) (bool, error) {
	return _IAssertionChain.Contract.IsFirstChild(&_IAssertionChain.CallOpts, assertionHash)
}

// IsFirstChild is a free data retrieval call binding the contract method 0x30836228.
//
// Solidity: function isFirstChild(bytes32 assertionHash) view returns(bool)
func (_IAssertionChain *IAssertionChainCallerSession) IsFirstChild(assertionHash [32]byte) (bool, error) {
	return _IAssertionChain.Contract.IsFirstChild(&_IAssertionChain.CallOpts, assertionHash)
}

// IsPending is a free data retrieval call binding the contract method 0xe531d8c7.
//
// Solidity: function isPending(bytes32 assertionHash) view returns(bool)
func (_IAssertionChain *IAssertionChainCaller) IsPending(opts *bind.CallOpts, assertionHash [32]byte) (bool, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "isPending", assertionHash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsPending is a free data retrieval call binding the contract method 0xe531d8c7.
//
// Solidity: function isPending(bytes32 assertionHash) view returns(bool)
func (_IAssertionChain *IAssertionChainSession) IsPending(assertionHash [32]byte) (bool, error) {
	return _IAssertionChain.Contract.IsPending(&_IAssertionChain.CallOpts, assertionHash)
}

// IsPending is a free data retrieval call binding the contract method 0xe531d8c7.
//
// Solidity: function isPending(bytes32 assertionHash) view returns(bool)
func (_IAssertionChain *IAssertionChainCallerSession) IsPending(assertionHash [32]byte) (bool, error) {
	return _IAssertionChain.Contract.IsPending(&_IAssertionChain.CallOpts, assertionHash)
}

// ValidateAssertionHash is a free data retrieval call binding the contract method 0xf9cee9df.
//
// Solidity: function validateAssertionHash(bytes32 assertionHash, ((bytes32[2],uint64[2]),uint8) state, bytes32 prevAssertionHash, bytes32 inboxAcc) view returns()
func (_IAssertionChain *IAssertionChainCaller) ValidateAssertionHash(opts *bind.CallOpts, assertionHash [32]byte, state ExecutionState, prevAssertionHash [32]byte, inboxAcc [32]byte) error {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "validateAssertionHash", assertionHash, state, prevAssertionHash, inboxAcc)

	if err != nil {
		return err
	}

	return err

}

// ValidateAssertionHash is a free data retrieval call binding the contract method 0xf9cee9df.
//
// Solidity: function validateAssertionHash(bytes32 assertionHash, ((bytes32[2],uint64[2]),uint8) state, bytes32 prevAssertionHash, bytes32 inboxAcc) view returns()
func (_IAssertionChain *IAssertionChainSession) ValidateAssertionHash(assertionHash [32]byte, state ExecutionState, prevAssertionHash [32]byte, inboxAcc [32]byte) error {
	return _IAssertionChain.Contract.ValidateAssertionHash(&_IAssertionChain.CallOpts, assertionHash, state, prevAssertionHash, inboxAcc)
}

// ValidateAssertionHash is a free data retrieval call binding the contract method 0xf9cee9df.
//
// Solidity: function validateAssertionHash(bytes32 assertionHash, ((bytes32[2],uint64[2]),uint8) state, bytes32 prevAssertionHash, bytes32 inboxAcc) view returns()
func (_IAssertionChain *IAssertionChainCallerSession) ValidateAssertionHash(assertionHash [32]byte, state ExecutionState, prevAssertionHash [32]byte, inboxAcc [32]byte) error {
	return _IAssertionChain.Contract.ValidateAssertionHash(&_IAssertionChain.CallOpts, assertionHash, state, prevAssertionHash, inboxAcc)
}

// ValidateConfig is a free data retrieval call binding the contract method 0x04972af9.
//
// Solidity: function validateConfig(bytes32 assertionHash, (bytes32,uint256,address,uint64,uint64) configData) view returns()
func (_IAssertionChain *IAssertionChainCaller) ValidateConfig(opts *bind.CallOpts, assertionHash [32]byte, configData ConfigData) error {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "validateConfig", assertionHash, configData)

	if err != nil {
		return err
	}

	return err

}

// ValidateConfig is a free data retrieval call binding the contract method 0x04972af9.
//
// Solidity: function validateConfig(bytes32 assertionHash, (bytes32,uint256,address,uint64,uint64) configData) view returns()
func (_IAssertionChain *IAssertionChainSession) ValidateConfig(assertionHash [32]byte, configData ConfigData) error {
	return _IAssertionChain.Contract.ValidateConfig(&_IAssertionChain.CallOpts, assertionHash, configData)
}

// ValidateConfig is a free data retrieval call binding the contract method 0x04972af9.
//
// Solidity: function validateConfig(bytes32 assertionHash, (bytes32,uint256,address,uint64,uint64) configData) view returns()
func (_IAssertionChain *IAssertionChainCallerSession) ValidateConfig(assertionHash [32]byte, configData ConfigData) error {
	return _IAssertionChain.Contract.ValidateConfig(&_IAssertionChain.CallOpts, assertionHash, configData)
}

// IEdgeChallengeManagerMetaData contains all meta data concerning the IEdgeChallengeManager contract.
var IEdgeChallengeManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"bisectionHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"bisectEdge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"name\":\"calculateEdgeId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"}],\"name\":\"calculateMutualId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"challengePeriodBlocks\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"confirmEdgeByChildren\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimingEdgeId\",\"type\":\"bytes32\"}],\"name\":\"confirmEdgeByClaim\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"internalType\":\"structOneStepData\",\"name\":\"oneStepData\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"requiredStake\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"confirmPeriodBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nextInboxPosition\",\"type\":\"uint64\"}],\"internalType\":\"structConfigData\",\"name\":\"prevConfig\",\"type\":\"tuple\"},{\"internalType\":\"bytes32[]\",\"name\":\"beforeHistoryInclusionProof\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"afterHistoryInclusionProof\",\"type\":\"bytes32[]\"}],\"name\":\"confirmEdgeByOneStepProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"ancestorEdgeIds\",\"type\":\"bytes32[]\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"}],\"internalType\":\"structExecutionState\",\"name\":\"executionState\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"prevAssertionHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"inboxAcc\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionStateData\",\"name\":\"claimStateData\",\"type\":\"tuple\"}],\"name\":\"confirmEdgeByTime\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"internalType\":\"structCreateEdgeArgs\",\"name\":\"args\",\"type\":\"tuple\"}],\"name\":\"createLayerZeroEdge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"edgeExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"edgeLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"firstRival\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"getEdge\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"lowerChildId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"upperChildId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"createdAtBlock\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"enumEdgeStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"enumEdgeType\",\"name\":\"eType\",\"type\":\"uint8\"},{\"internalType\":\"bool\",\"name\":\"refunded\",\"type\":\"bool\"}],\"internalType\":\"structChallengeEdge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"eType\",\"type\":\"uint8\"}],\"name\":\"getLayerZeroEndHeight\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"getPrevAssertionHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"hasLengthOneRival\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"hasRival\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriodBlocks\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"layerZeroBlockEdgeHeight\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"layerZeroBigStepEdgeHeight\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"layerZeroSmallStepEdgeHeight\",\"type\":\"uint256\"},{\"internalType\":\"contractIERC20\",\"name\":\"_stakeToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_stakeAmount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_excessStakeReceiver\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"oneStepProofEntry\",\"outputs\":[{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"refundStake\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"timeUnrivaled\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IEdgeChallengeManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use IEdgeChallengeManagerMetaData.ABI instead.
var IEdgeChallengeManagerABI = IEdgeChallengeManagerMetaData.ABI

// IEdgeChallengeManager is an auto generated Go binding around an Ethereum contract.
type IEdgeChallengeManager struct {
	IEdgeChallengeManagerCaller     // Read-only binding to the contract
	IEdgeChallengeManagerTransactor // Write-only binding to the contract
	IEdgeChallengeManagerFilterer   // Log filterer for contract events
}

// IEdgeChallengeManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type IEdgeChallengeManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IEdgeChallengeManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IEdgeChallengeManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IEdgeChallengeManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IEdgeChallengeManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IEdgeChallengeManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IEdgeChallengeManagerSession struct {
	Contract     *IEdgeChallengeManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// IEdgeChallengeManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IEdgeChallengeManagerCallerSession struct {
	Contract *IEdgeChallengeManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// IEdgeChallengeManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IEdgeChallengeManagerTransactorSession struct {
	Contract     *IEdgeChallengeManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// IEdgeChallengeManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type IEdgeChallengeManagerRaw struct {
	Contract *IEdgeChallengeManager // Generic contract binding to access the raw methods on
}

// IEdgeChallengeManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IEdgeChallengeManagerCallerRaw struct {
	Contract *IEdgeChallengeManagerCaller // Generic read-only contract binding to access the raw methods on
}

// IEdgeChallengeManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IEdgeChallengeManagerTransactorRaw struct {
	Contract *IEdgeChallengeManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIEdgeChallengeManager creates a new instance of IEdgeChallengeManager, bound to a specific deployed contract.
func NewIEdgeChallengeManager(address common.Address, backend bind.ContractBackend) (*IEdgeChallengeManager, error) {
	contract, err := bindIEdgeChallengeManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IEdgeChallengeManager{IEdgeChallengeManagerCaller: IEdgeChallengeManagerCaller{contract: contract}, IEdgeChallengeManagerTransactor: IEdgeChallengeManagerTransactor{contract: contract}, IEdgeChallengeManagerFilterer: IEdgeChallengeManagerFilterer{contract: contract}}, nil
}

// NewIEdgeChallengeManagerCaller creates a new read-only instance of IEdgeChallengeManager, bound to a specific deployed contract.
func NewIEdgeChallengeManagerCaller(address common.Address, caller bind.ContractCaller) (*IEdgeChallengeManagerCaller, error) {
	contract, err := bindIEdgeChallengeManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IEdgeChallengeManagerCaller{contract: contract}, nil
}

// NewIEdgeChallengeManagerTransactor creates a new write-only instance of IEdgeChallengeManager, bound to a specific deployed contract.
func NewIEdgeChallengeManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*IEdgeChallengeManagerTransactor, error) {
	contract, err := bindIEdgeChallengeManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IEdgeChallengeManagerTransactor{contract: contract}, nil
}

// NewIEdgeChallengeManagerFilterer creates a new log filterer instance of IEdgeChallengeManager, bound to a specific deployed contract.
func NewIEdgeChallengeManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*IEdgeChallengeManagerFilterer, error) {
	contract, err := bindIEdgeChallengeManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IEdgeChallengeManagerFilterer{contract: contract}, nil
}

// bindIEdgeChallengeManager binds a generic wrapper to an already deployed contract.
func bindIEdgeChallengeManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IEdgeChallengeManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IEdgeChallengeManager *IEdgeChallengeManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IEdgeChallengeManager.Contract.IEdgeChallengeManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IEdgeChallengeManager *IEdgeChallengeManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.IEdgeChallengeManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IEdgeChallengeManager *IEdgeChallengeManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.IEdgeChallengeManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IEdgeChallengeManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.contract.Transact(opts, method, params...)
}

// CalculateEdgeId is a free data retrieval call binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) pure returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) CalculateEdgeId(opts *bind.CallOpts, edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "calculateEdgeId", edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateEdgeId is a free data retrieval call binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) pure returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) CalculateEdgeId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) ([32]byte, error) {
	return _IEdgeChallengeManager.Contract.CalculateEdgeId(&_IEdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
}

// CalculateEdgeId is a free data retrieval call binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) pure returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) CalculateEdgeId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) ([32]byte, error) {
	return _IEdgeChallengeManager.Contract.CalculateEdgeId(&_IEdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
}

// CalculateMutualId is a free data retrieval call binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) pure returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) CalculateMutualId(opts *bind.CallOpts, edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "calculateMutualId", edgeType, originId, startHeight, startHistoryRoot, endHeight)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateMutualId is a free data retrieval call binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) pure returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) CalculateMutualId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) ([32]byte, error) {
	return _IEdgeChallengeManager.Contract.CalculateMutualId(&_IEdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight)
}

// CalculateMutualId is a free data retrieval call binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) pure returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) CalculateMutualId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) ([32]byte, error) {
	return _IEdgeChallengeManager.Contract.CalculateMutualId(&_IEdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight)
}

// ChallengePeriodBlocks is a free data retrieval call binding the contract method 0x46c2781a.
//
// Solidity: function challengePeriodBlocks() view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) ChallengePeriodBlocks(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "challengePeriodBlocks")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChallengePeriodBlocks is a free data retrieval call binding the contract method 0x46c2781a.
//
// Solidity: function challengePeriodBlocks() view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) ChallengePeriodBlocks() (*big.Int, error) {
	return _IEdgeChallengeManager.Contract.ChallengePeriodBlocks(&_IEdgeChallengeManager.CallOpts)
}

// ChallengePeriodBlocks is a free data retrieval call binding the contract method 0x46c2781a.
//
// Solidity: function challengePeriodBlocks() view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) ChallengePeriodBlocks() (*big.Int, error) {
	return _IEdgeChallengeManager.Contract.ChallengePeriodBlocks(&_IEdgeChallengeManager.CallOpts)
}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 edgeId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) EdgeExists(opts *bind.CallOpts, edgeId [32]byte) (bool, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "edgeExists", edgeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 edgeId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) EdgeExists(edgeId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.EdgeExists(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 edgeId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) EdgeExists(edgeId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.EdgeExists(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// EdgeLength is a free data retrieval call binding the contract method 0xeae0328b.
//
// Solidity: function edgeLength(bytes32 edgeId) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) EdgeLength(opts *bind.CallOpts, edgeId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "edgeLength", edgeId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EdgeLength is a free data retrieval call binding the contract method 0xeae0328b.
//
// Solidity: function edgeLength(bytes32 edgeId) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) EdgeLength(edgeId [32]byte) (*big.Int, error) {
	return _IEdgeChallengeManager.Contract.EdgeLength(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// EdgeLength is a free data retrieval call binding the contract method 0xeae0328b.
//
// Solidity: function edgeLength(bytes32 edgeId) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) EdgeLength(edgeId [32]byte) (*big.Int, error) {
	return _IEdgeChallengeManager.Contract.EdgeLength(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// FirstRival is a free data retrieval call binding the contract method 0xbce6f54f.
//
// Solidity: function firstRival(bytes32 edgeId) view returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) FirstRival(opts *bind.CallOpts, edgeId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "firstRival", edgeId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// FirstRival is a free data retrieval call binding the contract method 0xbce6f54f.
//
// Solidity: function firstRival(bytes32 edgeId) view returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) FirstRival(edgeId [32]byte) ([32]byte, error) {
	return _IEdgeChallengeManager.Contract.FirstRival(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// FirstRival is a free data retrieval call binding the contract method 0xbce6f54f.
//
// Solidity: function firstRival(bytes32 edgeId) view returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) FirstRival(edgeId [32]byte) ([32]byte, error) {
	return _IEdgeChallengeManager.Contract.FirstRival(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 edgeId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8,bool))
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) GetEdge(opts *bind.CallOpts, edgeId [32]byte) (ChallengeEdge, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "getEdge", edgeId)

	if err != nil {
		return *new(ChallengeEdge), err
	}

	out0 := *abi.ConvertType(out[0], new(ChallengeEdge)).(*ChallengeEdge)

	return out0, err

}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 edgeId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8,bool))
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) GetEdge(edgeId [32]byte) (ChallengeEdge, error) {
	return _IEdgeChallengeManager.Contract.GetEdge(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 edgeId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8,bool))
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) GetEdge(edgeId [32]byte) (ChallengeEdge, error) {
	return _IEdgeChallengeManager.Contract.GetEdge(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// GetLayerZeroEndHeight is a free data retrieval call binding the contract method 0x42e1aaa8.
//
// Solidity: function getLayerZeroEndHeight(uint8 eType) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) GetLayerZeroEndHeight(opts *bind.CallOpts, eType uint8) (*big.Int, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "getLayerZeroEndHeight", eType)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLayerZeroEndHeight is a free data retrieval call binding the contract method 0x42e1aaa8.
//
// Solidity: function getLayerZeroEndHeight(uint8 eType) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) GetLayerZeroEndHeight(eType uint8) (*big.Int, error) {
	return _IEdgeChallengeManager.Contract.GetLayerZeroEndHeight(&_IEdgeChallengeManager.CallOpts, eType)
}

// GetLayerZeroEndHeight is a free data retrieval call binding the contract method 0x42e1aaa8.
//
// Solidity: function getLayerZeroEndHeight(uint8 eType) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) GetLayerZeroEndHeight(eType uint8) (*big.Int, error) {
	return _IEdgeChallengeManager.Contract.GetLayerZeroEndHeight(&_IEdgeChallengeManager.CallOpts, eType)
}

// GetPrevAssertionHash is a free data retrieval call binding the contract method 0x5a48e0f4.
//
// Solidity: function getPrevAssertionHash(bytes32 edgeId) view returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) GetPrevAssertionHash(opts *bind.CallOpts, edgeId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "getPrevAssertionHash", edgeId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetPrevAssertionHash is a free data retrieval call binding the contract method 0x5a48e0f4.
//
// Solidity: function getPrevAssertionHash(bytes32 edgeId) view returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) GetPrevAssertionHash(edgeId [32]byte) ([32]byte, error) {
	return _IEdgeChallengeManager.Contract.GetPrevAssertionHash(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// GetPrevAssertionHash is a free data retrieval call binding the contract method 0x5a48e0f4.
//
// Solidity: function getPrevAssertionHash(bytes32 edgeId) view returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) GetPrevAssertionHash(edgeId [32]byte) ([32]byte, error) {
	return _IEdgeChallengeManager.Contract.GetPrevAssertionHash(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 edgeId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) HasLengthOneRival(opts *bind.CallOpts, edgeId [32]byte) (bool, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "hasLengthOneRival", edgeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 edgeId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) HasLengthOneRival(edgeId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.HasLengthOneRival(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 edgeId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) HasLengthOneRival(edgeId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.HasLengthOneRival(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 edgeId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) HasRival(opts *bind.CallOpts, edgeId [32]byte) (bool, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "hasRival", edgeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 edgeId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) HasRival(edgeId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.HasRival(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 edgeId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) HasRival(edgeId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.HasRival(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// OneStepProofEntry is a free data retrieval call binding the contract method 0x48923bc5.
//
// Solidity: function oneStepProofEntry() view returns(address)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) OneStepProofEntry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "oneStepProofEntry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OneStepProofEntry is a free data retrieval call binding the contract method 0x48923bc5.
//
// Solidity: function oneStepProofEntry() view returns(address)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) OneStepProofEntry() (common.Address, error) {
	return _IEdgeChallengeManager.Contract.OneStepProofEntry(&_IEdgeChallengeManager.CallOpts)
}

// OneStepProofEntry is a free data retrieval call binding the contract method 0x48923bc5.
//
// Solidity: function oneStepProofEntry() view returns(address)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) OneStepProofEntry() (common.Address, error) {
	return _IEdgeChallengeManager.Contract.OneStepProofEntry(&_IEdgeChallengeManager.CallOpts)
}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 edgeId) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) TimeUnrivaled(opts *bind.CallOpts, edgeId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "timeUnrivaled", edgeId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 edgeId) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) TimeUnrivaled(edgeId [32]byte) (*big.Int, error) {
	return _IEdgeChallengeManager.Contract.TimeUnrivaled(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 edgeId) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) TimeUnrivaled(edgeId [32]byte) (*big.Int, error) {
	return _IEdgeChallengeManager.Contract.TimeUnrivaled(&_IEdgeChallengeManager.CallOpts, edgeId)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) BisectEdge(opts *bind.TransactOpts, edgeId [32]byte, bisectionHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "bisectEdge", edgeId, bisectionHistoryRoot, prefixProof)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) BisectEdge(edgeId [32]byte, bisectionHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.BisectEdge(&_IEdgeChallengeManager.TransactOpts, edgeId, bisectionHistoryRoot, prefixProof)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) BisectEdge(edgeId [32]byte, bisectionHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.BisectEdge(&_IEdgeChallengeManager.TransactOpts, edgeId, bisectionHistoryRoot, prefixProof)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 edgeId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) ConfirmEdgeByChildren(opts *bind.TransactOpts, edgeId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "confirmEdgeByChildren", edgeId)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 edgeId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) ConfirmEdgeByChildren(edgeId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByChildren(&_IEdgeChallengeManager.TransactOpts, edgeId)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 edgeId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) ConfirmEdgeByChildren(edgeId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByChildren(&_IEdgeChallengeManager.TransactOpts, edgeId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) ConfirmEdgeByClaim(opts *bind.TransactOpts, edgeId [32]byte, claimingEdgeId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "confirmEdgeByClaim", edgeId, claimingEdgeId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) ConfirmEdgeByClaim(edgeId [32]byte, claimingEdgeId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByClaim(&_IEdgeChallengeManager.TransactOpts, edgeId, claimingEdgeId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) ConfirmEdgeByClaim(edgeId [32]byte, claimingEdgeId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByClaim(&_IEdgeChallengeManager.TransactOpts, edgeId, claimingEdgeId)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0x8c1b3a40.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, (bytes32,bytes) oneStepData, (bytes32,uint256,address,uint64,uint64) prevConfig, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) ConfirmEdgeByOneStepProof(opts *bind.TransactOpts, edgeId [32]byte, oneStepData OneStepData, prevConfig ConfigData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "confirmEdgeByOneStepProof", edgeId, oneStepData, prevConfig, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0x8c1b3a40.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, (bytes32,bytes) oneStepData, (bytes32,uint256,address,uint64,uint64) prevConfig, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) ConfirmEdgeByOneStepProof(edgeId [32]byte, oneStepData OneStepData, prevConfig ConfigData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByOneStepProof(&_IEdgeChallengeManager.TransactOpts, edgeId, oneStepData, prevConfig, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0x8c1b3a40.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, (bytes32,bytes) oneStepData, (bytes32,uint256,address,uint64,uint64) prevConfig, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) ConfirmEdgeByOneStepProof(edgeId [32]byte, oneStepData OneStepData, prevConfig ConfigData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByOneStepProof(&_IEdgeChallengeManager.TransactOpts, edgeId, oneStepData, prevConfig, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x64deed59.
//
// Solidity: function confirmEdgeByTime(bytes32 edgeId, bytes32[] ancestorEdgeIds, (((bytes32[2],uint64[2]),uint8),bytes32,bytes32) claimStateData) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) ConfirmEdgeByTime(opts *bind.TransactOpts, edgeId [32]byte, ancestorEdgeIds [][32]byte, claimStateData ExecutionStateData) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "confirmEdgeByTime", edgeId, ancestorEdgeIds, claimStateData)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x64deed59.
//
// Solidity: function confirmEdgeByTime(bytes32 edgeId, bytes32[] ancestorEdgeIds, (((bytes32[2],uint64[2]),uint8),bytes32,bytes32) claimStateData) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) ConfirmEdgeByTime(edgeId [32]byte, ancestorEdgeIds [][32]byte, claimStateData ExecutionStateData) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByTime(&_IEdgeChallengeManager.TransactOpts, edgeId, ancestorEdgeIds, claimStateData)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x64deed59.
//
// Solidity: function confirmEdgeByTime(bytes32 edgeId, bytes32[] ancestorEdgeIds, (((bytes32[2],uint64[2]),uint8),bytes32,bytes32) claimStateData) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) ConfirmEdgeByTime(edgeId [32]byte, ancestorEdgeIds [][32]byte, claimStateData ExecutionStateData) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByTime(&_IEdgeChallengeManager.TransactOpts, edgeId, ancestorEdgeIds, claimStateData)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x05fae141.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,bytes,bytes) args) returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) CreateLayerZeroEdge(opts *bind.TransactOpts, args CreateEdgeArgs) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "createLayerZeroEdge", args)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x05fae141.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,bytes,bytes) args) returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) CreateLayerZeroEdge(args CreateEdgeArgs) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.CreateLayerZeroEdge(&_IEdgeChallengeManager.TransactOpts, args)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x05fae141.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,bytes,bytes) args) returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) CreateLayerZeroEdge(args CreateEdgeArgs) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.CreateLayerZeroEdge(&_IEdgeChallengeManager.TransactOpts, args)
}

// Initialize is a paid mutator transaction binding the contract method 0x453e969b.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodBlocks, address _oneStepProofEntry, uint256 layerZeroBlockEdgeHeight, uint256 layerZeroBigStepEdgeHeight, uint256 layerZeroSmallStepEdgeHeight, address _stakeToken, uint256 _stakeAmount, address _excessStakeReceiver) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) Initialize(opts *bind.TransactOpts, _assertionChain common.Address, _challengePeriodBlocks *big.Int, _oneStepProofEntry common.Address, layerZeroBlockEdgeHeight *big.Int, layerZeroBigStepEdgeHeight *big.Int, layerZeroSmallStepEdgeHeight *big.Int, _stakeToken common.Address, _stakeAmount *big.Int, _excessStakeReceiver common.Address) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "initialize", _assertionChain, _challengePeriodBlocks, _oneStepProofEntry, layerZeroBlockEdgeHeight, layerZeroBigStepEdgeHeight, layerZeroSmallStepEdgeHeight, _stakeToken, _stakeAmount, _excessStakeReceiver)
}

// Initialize is a paid mutator transaction binding the contract method 0x453e969b.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodBlocks, address _oneStepProofEntry, uint256 layerZeroBlockEdgeHeight, uint256 layerZeroBigStepEdgeHeight, uint256 layerZeroSmallStepEdgeHeight, address _stakeToken, uint256 _stakeAmount, address _excessStakeReceiver) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) Initialize(_assertionChain common.Address, _challengePeriodBlocks *big.Int, _oneStepProofEntry common.Address, layerZeroBlockEdgeHeight *big.Int, layerZeroBigStepEdgeHeight *big.Int, layerZeroSmallStepEdgeHeight *big.Int, _stakeToken common.Address, _stakeAmount *big.Int, _excessStakeReceiver common.Address) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.Initialize(&_IEdgeChallengeManager.TransactOpts, _assertionChain, _challengePeriodBlocks, _oneStepProofEntry, layerZeroBlockEdgeHeight, layerZeroBigStepEdgeHeight, layerZeroSmallStepEdgeHeight, _stakeToken, _stakeAmount, _excessStakeReceiver)
}

// Initialize is a paid mutator transaction binding the contract method 0x453e969b.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodBlocks, address _oneStepProofEntry, uint256 layerZeroBlockEdgeHeight, uint256 layerZeroBigStepEdgeHeight, uint256 layerZeroSmallStepEdgeHeight, address _stakeToken, uint256 _stakeAmount, address _excessStakeReceiver) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) Initialize(_assertionChain common.Address, _challengePeriodBlocks *big.Int, _oneStepProofEntry common.Address, layerZeroBlockEdgeHeight *big.Int, layerZeroBigStepEdgeHeight *big.Int, layerZeroSmallStepEdgeHeight *big.Int, _stakeToken common.Address, _stakeAmount *big.Int, _excessStakeReceiver common.Address) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.Initialize(&_IEdgeChallengeManager.TransactOpts, _assertionChain, _challengePeriodBlocks, _oneStepProofEntry, layerZeroBlockEdgeHeight, layerZeroBigStepEdgeHeight, layerZeroSmallStepEdgeHeight, _stakeToken, _stakeAmount, _excessStakeReceiver)
}

// RefundStake is a paid mutator transaction binding the contract method 0x748926f3.
//
// Solidity: function refundStake(bytes32 edgeId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) RefundStake(opts *bind.TransactOpts, edgeId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "refundStake", edgeId)
}

// RefundStake is a paid mutator transaction binding the contract method 0x748926f3.
//
// Solidity: function refundStake(bytes32 edgeId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) RefundStake(edgeId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.RefundStake(&_IEdgeChallengeManager.TransactOpts, edgeId)
}

// RefundStake is a paid mutator transaction binding the contract method 0x748926f3.
//
// Solidity: function refundStake(bytes32 edgeId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) RefundStake(edgeId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.RefundStake(&_IEdgeChallengeManager.TransactOpts, edgeId)
}
