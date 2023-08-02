// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package challengegen

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

// GlobalState is an auto generated low-level Go binding around an user-defined struct.
type GlobalState struct {
	Bytes32Vals [2][32]byte
	U64Vals     [2]uint64
}

// OldChallengeLibChallenge is an auto generated low-level Go binding around an user-defined struct.
type OldChallengeLibChallenge struct {
	Current            OldChallengeLibParticipant
	Next               OldChallengeLibParticipant
	LastMoveTimestamp  *big.Int
	WasmModuleRoot     [32]byte
	ChallengeStateHash [32]byte
	MaxInboxMessages   uint64
	Mode               uint8
}

// OldChallengeLibParticipant is an auto generated low-level Go binding around an user-defined struct.
type OldChallengeLibParticipant struct {
	Addr     common.Address
	TimeLeft *big.Int
}

// OldChallengeLibSegmentSelection is an auto generated low-level Go binding around an user-defined struct.
type OldChallengeLibSegmentSelection struct {
	OldSegmentsStart  *big.Int
	OldSegmentsLength *big.Int
	OldSegments       [][32]byte
	ChallengePosition *big.Int
}

// IOldChallengeManagerMetaData contains all meta data concerning the IOldChallengeManager contract.
var IOldChallengeManagerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"challengeRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"challengedSegmentStart\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"challengedSegmentLength\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32[]\",\"name\":\"chainHashes\",\"type\":\"bytes32[]\"}],\"name\":\"Bisected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"enumIOldChallengeManager.ChallengeTerminationType\",\"name\":\"kind\",\"type\":\"uint8\"}],\"name\":\"ChallengeEnded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockSteps\",\"type\":\"uint256\"}],\"name\":\"ExecutionChallengeBegun\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"indexed\":false,\"internalType\":\"structGlobalState\",\"name\":\"startState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"indexed\":false,\"internalType\":\"structGlobalState\",\"name\":\"endState\",\"type\":\"tuple\"}],\"name\":\"InitiatedChallenge\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"OneStepProofCompleted\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex_\",\"type\":\"uint64\"}],\"name\":\"challengeInfo\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timeLeft\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.Participant\",\"name\":\"current\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timeLeft\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.Participant\",\"name\":\"next\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"lastMoveTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"challengeStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"maxInboxMessages\",\"type\":\"uint64\"},{\"internalType\":\"enumOldChallengeLib.ChallengeMode\",\"name\":\"mode\",\"type\":\"uint8\"}],\"internalType\":\"structOldChallengeLib.Challenge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex_\",\"type\":\"uint64\"}],\"name\":\"clearChallenge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot_\",\"type\":\"bytes32\"},{\"internalType\":\"enumMachineStatus[2]\",\"name\":\"startAndEndMachineStatuses_\",\"type\":\"uint8[2]\"},{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState[2]\",\"name\":\"startAndEndGlobalStates_\",\"type\":\"tuple[2]\"},{\"internalType\":\"uint64\",\"name\":\"numBlocks\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"asserter_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"challenger_\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asserterTimeLeft_\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"challengerTimeLeft_\",\"type\":\"uint256\"}],\"name\":\"createChallenge\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"currentResponder\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOldChallengeResultReceiver\",\"name\":\"resultReceiver_\",\"type\":\"address\"},{\"internalType\":\"contractISequencerInbox\",\"name\":\"sequencerInbox_\",\"type\":\"address\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge_\",\"type\":\"address\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"osp_\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"isTimedOut\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex_\",\"type\":\"uint64\"}],\"name\":\"timeout\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IOldChallengeManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use IOldChallengeManagerMetaData.ABI instead.
var IOldChallengeManagerABI = IOldChallengeManagerMetaData.ABI

// IOldChallengeManager is an auto generated Go binding around an Ethereum contract.
type IOldChallengeManager struct {
	IOldChallengeManagerCaller     // Read-only binding to the contract
	IOldChallengeManagerTransactor // Write-only binding to the contract
	IOldChallengeManagerFilterer   // Log filterer for contract events
}

// IOldChallengeManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type IOldChallengeManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOldChallengeManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IOldChallengeManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOldChallengeManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IOldChallengeManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOldChallengeManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IOldChallengeManagerSession struct {
	Contract     *IOldChallengeManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// IOldChallengeManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IOldChallengeManagerCallerSession struct {
	Contract *IOldChallengeManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// IOldChallengeManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IOldChallengeManagerTransactorSession struct {
	Contract     *IOldChallengeManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// IOldChallengeManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type IOldChallengeManagerRaw struct {
	Contract *IOldChallengeManager // Generic contract binding to access the raw methods on
}

// IOldChallengeManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IOldChallengeManagerCallerRaw struct {
	Contract *IOldChallengeManagerCaller // Generic read-only contract binding to access the raw methods on
}

// IOldChallengeManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IOldChallengeManagerTransactorRaw struct {
	Contract *IOldChallengeManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIOldChallengeManager creates a new instance of IOldChallengeManager, bound to a specific deployed contract.
func NewIOldChallengeManager(address common.Address, backend bind.ContractBackend) (*IOldChallengeManager, error) {
	contract, err := bindIOldChallengeManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeManager{IOldChallengeManagerCaller: IOldChallengeManagerCaller{contract: contract}, IOldChallengeManagerTransactor: IOldChallengeManagerTransactor{contract: contract}, IOldChallengeManagerFilterer: IOldChallengeManagerFilterer{contract: contract}}, nil
}

// NewIOldChallengeManagerCaller creates a new read-only instance of IOldChallengeManager, bound to a specific deployed contract.
func NewIOldChallengeManagerCaller(address common.Address, caller bind.ContractCaller) (*IOldChallengeManagerCaller, error) {
	contract, err := bindIOldChallengeManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeManagerCaller{contract: contract}, nil
}

// NewIOldChallengeManagerTransactor creates a new write-only instance of IOldChallengeManager, bound to a specific deployed contract.
func NewIOldChallengeManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*IOldChallengeManagerTransactor, error) {
	contract, err := bindIOldChallengeManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeManagerTransactor{contract: contract}, nil
}

// NewIOldChallengeManagerFilterer creates a new log filterer instance of IOldChallengeManager, bound to a specific deployed contract.
func NewIOldChallengeManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*IOldChallengeManagerFilterer, error) {
	contract, err := bindIOldChallengeManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeManagerFilterer{contract: contract}, nil
}

// bindIOldChallengeManager binds a generic wrapper to an already deployed contract.
func bindIOldChallengeManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IOldChallengeManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOldChallengeManager *IOldChallengeManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOldChallengeManager.Contract.IOldChallengeManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOldChallengeManager *IOldChallengeManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.IOldChallengeManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOldChallengeManager *IOldChallengeManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.IOldChallengeManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOldChallengeManager *IOldChallengeManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOldChallengeManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOldChallengeManager *IOldChallengeManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOldChallengeManager *IOldChallengeManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.contract.Transact(opts, method, params...)
}

// ChallengeInfo is a free data retrieval call binding the contract method 0x7fd07a9c.
//
// Solidity: function challengeInfo(uint64 challengeIndex_) view returns(((address,uint256),(address,uint256),uint256,bytes32,bytes32,uint64,uint8))
func (_IOldChallengeManager *IOldChallengeManagerCaller) ChallengeInfo(opts *bind.CallOpts, challengeIndex_ uint64) (OldChallengeLibChallenge, error) {
	var out []interface{}
	err := _IOldChallengeManager.contract.Call(opts, &out, "challengeInfo", challengeIndex_)

	if err != nil {
		return *new(OldChallengeLibChallenge), err
	}

	out0 := *abi.ConvertType(out[0], new(OldChallengeLibChallenge)).(*OldChallengeLibChallenge)

	return out0, err

}

// ChallengeInfo is a free data retrieval call binding the contract method 0x7fd07a9c.
//
// Solidity: function challengeInfo(uint64 challengeIndex_) view returns(((address,uint256),(address,uint256),uint256,bytes32,bytes32,uint64,uint8))
func (_IOldChallengeManager *IOldChallengeManagerSession) ChallengeInfo(challengeIndex_ uint64) (OldChallengeLibChallenge, error) {
	return _IOldChallengeManager.Contract.ChallengeInfo(&_IOldChallengeManager.CallOpts, challengeIndex_)
}

// ChallengeInfo is a free data retrieval call binding the contract method 0x7fd07a9c.
//
// Solidity: function challengeInfo(uint64 challengeIndex_) view returns(((address,uint256),(address,uint256),uint256,bytes32,bytes32,uint64,uint8))
func (_IOldChallengeManager *IOldChallengeManagerCallerSession) ChallengeInfo(challengeIndex_ uint64) (OldChallengeLibChallenge, error) {
	return _IOldChallengeManager.Contract.ChallengeInfo(&_IOldChallengeManager.CallOpts, challengeIndex_)
}

// CurrentResponder is a free data retrieval call binding the contract method 0x23a9ef23.
//
// Solidity: function currentResponder(uint64 challengeIndex) view returns(address)
func (_IOldChallengeManager *IOldChallengeManagerCaller) CurrentResponder(opts *bind.CallOpts, challengeIndex uint64) (common.Address, error) {
	var out []interface{}
	err := _IOldChallengeManager.contract.Call(opts, &out, "currentResponder", challengeIndex)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CurrentResponder is a free data retrieval call binding the contract method 0x23a9ef23.
//
// Solidity: function currentResponder(uint64 challengeIndex) view returns(address)
func (_IOldChallengeManager *IOldChallengeManagerSession) CurrentResponder(challengeIndex uint64) (common.Address, error) {
	return _IOldChallengeManager.Contract.CurrentResponder(&_IOldChallengeManager.CallOpts, challengeIndex)
}

// CurrentResponder is a free data retrieval call binding the contract method 0x23a9ef23.
//
// Solidity: function currentResponder(uint64 challengeIndex) view returns(address)
func (_IOldChallengeManager *IOldChallengeManagerCallerSession) CurrentResponder(challengeIndex uint64) (common.Address, error) {
	return _IOldChallengeManager.Contract.CurrentResponder(&_IOldChallengeManager.CallOpts, challengeIndex)
}

// IsTimedOut is a free data retrieval call binding the contract method 0x9ede42b9.
//
// Solidity: function isTimedOut(uint64 challengeIndex) view returns(bool)
func (_IOldChallengeManager *IOldChallengeManagerCaller) IsTimedOut(opts *bind.CallOpts, challengeIndex uint64) (bool, error) {
	var out []interface{}
	err := _IOldChallengeManager.contract.Call(opts, &out, "isTimedOut", challengeIndex)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsTimedOut is a free data retrieval call binding the contract method 0x9ede42b9.
//
// Solidity: function isTimedOut(uint64 challengeIndex) view returns(bool)
func (_IOldChallengeManager *IOldChallengeManagerSession) IsTimedOut(challengeIndex uint64) (bool, error) {
	return _IOldChallengeManager.Contract.IsTimedOut(&_IOldChallengeManager.CallOpts, challengeIndex)
}

// IsTimedOut is a free data retrieval call binding the contract method 0x9ede42b9.
//
// Solidity: function isTimedOut(uint64 challengeIndex) view returns(bool)
func (_IOldChallengeManager *IOldChallengeManagerCallerSession) IsTimedOut(challengeIndex uint64) (bool, error) {
	return _IOldChallengeManager.Contract.IsTimedOut(&_IOldChallengeManager.CallOpts, challengeIndex)
}

// ClearChallenge is a paid mutator transaction binding the contract method 0x56e9df97.
//
// Solidity: function clearChallenge(uint64 challengeIndex_) returns()
func (_IOldChallengeManager *IOldChallengeManagerTransactor) ClearChallenge(opts *bind.TransactOpts, challengeIndex_ uint64) (*types.Transaction, error) {
	return _IOldChallengeManager.contract.Transact(opts, "clearChallenge", challengeIndex_)
}

// ClearChallenge is a paid mutator transaction binding the contract method 0x56e9df97.
//
// Solidity: function clearChallenge(uint64 challengeIndex_) returns()
func (_IOldChallengeManager *IOldChallengeManagerSession) ClearChallenge(challengeIndex_ uint64) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.ClearChallenge(&_IOldChallengeManager.TransactOpts, challengeIndex_)
}

// ClearChallenge is a paid mutator transaction binding the contract method 0x56e9df97.
//
// Solidity: function clearChallenge(uint64 challengeIndex_) returns()
func (_IOldChallengeManager *IOldChallengeManagerTransactorSession) ClearChallenge(challengeIndex_ uint64) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.ClearChallenge(&_IOldChallengeManager.TransactOpts, challengeIndex_)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_IOldChallengeManager *IOldChallengeManagerTransactor) CreateChallenge(opts *bind.TransactOpts, wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _IOldChallengeManager.contract.Transact(opts, "createChallenge", wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_IOldChallengeManager *IOldChallengeManagerSession) CreateChallenge(wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.CreateChallenge(&_IOldChallengeManager.TransactOpts, wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_IOldChallengeManager *IOldChallengeManagerTransactorSession) CreateChallenge(wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.CreateChallenge(&_IOldChallengeManager.TransactOpts, wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address resultReceiver_, address sequencerInbox_, address bridge_, address osp_) returns()
func (_IOldChallengeManager *IOldChallengeManagerTransactor) Initialize(opts *bind.TransactOpts, resultReceiver_ common.Address, sequencerInbox_ common.Address, bridge_ common.Address, osp_ common.Address) (*types.Transaction, error) {
	return _IOldChallengeManager.contract.Transact(opts, "initialize", resultReceiver_, sequencerInbox_, bridge_, osp_)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address resultReceiver_, address sequencerInbox_, address bridge_, address osp_) returns()
func (_IOldChallengeManager *IOldChallengeManagerSession) Initialize(resultReceiver_ common.Address, sequencerInbox_ common.Address, bridge_ common.Address, osp_ common.Address) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.Initialize(&_IOldChallengeManager.TransactOpts, resultReceiver_, sequencerInbox_, bridge_, osp_)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address resultReceiver_, address sequencerInbox_, address bridge_, address osp_) returns()
func (_IOldChallengeManager *IOldChallengeManagerTransactorSession) Initialize(resultReceiver_ common.Address, sequencerInbox_ common.Address, bridge_ common.Address, osp_ common.Address) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.Initialize(&_IOldChallengeManager.TransactOpts, resultReceiver_, sequencerInbox_, bridge_, osp_)
}

// Timeout is a paid mutator transaction binding the contract method 0x1b45c86a.
//
// Solidity: function timeout(uint64 challengeIndex_) returns()
func (_IOldChallengeManager *IOldChallengeManagerTransactor) Timeout(opts *bind.TransactOpts, challengeIndex_ uint64) (*types.Transaction, error) {
	return _IOldChallengeManager.contract.Transact(opts, "timeout", challengeIndex_)
}

// Timeout is a paid mutator transaction binding the contract method 0x1b45c86a.
//
// Solidity: function timeout(uint64 challengeIndex_) returns()
func (_IOldChallengeManager *IOldChallengeManagerSession) Timeout(challengeIndex_ uint64) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.Timeout(&_IOldChallengeManager.TransactOpts, challengeIndex_)
}

// Timeout is a paid mutator transaction binding the contract method 0x1b45c86a.
//
// Solidity: function timeout(uint64 challengeIndex_) returns()
func (_IOldChallengeManager *IOldChallengeManagerTransactorSession) Timeout(challengeIndex_ uint64) (*types.Transaction, error) {
	return _IOldChallengeManager.Contract.Timeout(&_IOldChallengeManager.TransactOpts, challengeIndex_)
}

// IOldChallengeManagerBisectedIterator is returned from FilterBisected and is used to iterate over the raw logs and unpacked data for Bisected events raised by the IOldChallengeManager contract.
type IOldChallengeManagerBisectedIterator struct {
	Event *IOldChallengeManagerBisected // Event containing the contract specifics and raw log

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
func (it *IOldChallengeManagerBisectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IOldChallengeManagerBisected)
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
		it.Event = new(IOldChallengeManagerBisected)
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
func (it *IOldChallengeManagerBisectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IOldChallengeManagerBisectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IOldChallengeManagerBisected represents a Bisected event raised by the IOldChallengeManager contract.
type IOldChallengeManagerBisected struct {
	ChallengeIndex          uint64
	ChallengeRoot           [32]byte
	ChallengedSegmentStart  *big.Int
	ChallengedSegmentLength *big.Int
	ChainHashes             [][32]byte
	Raw                     types.Log // Blockchain specific contextual infos
}

// FilterBisected is a free log retrieval operation binding the contract event 0x86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d68.
//
// Solidity: event Bisected(uint64 indexed challengeIndex, bytes32 indexed challengeRoot, uint256 challengedSegmentStart, uint256 challengedSegmentLength, bytes32[] chainHashes)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) FilterBisected(opts *bind.FilterOpts, challengeIndex []uint64, challengeRoot [][32]byte) (*IOldChallengeManagerBisectedIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}
	var challengeRootRule []interface{}
	for _, challengeRootItem := range challengeRoot {
		challengeRootRule = append(challengeRootRule, challengeRootItem)
	}

	logs, sub, err := _IOldChallengeManager.contract.FilterLogs(opts, "Bisected", challengeIndexRule, challengeRootRule)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeManagerBisectedIterator{contract: _IOldChallengeManager.contract, event: "Bisected", logs: logs, sub: sub}, nil
}

// WatchBisected is a free log subscription operation binding the contract event 0x86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d68.
//
// Solidity: event Bisected(uint64 indexed challengeIndex, bytes32 indexed challengeRoot, uint256 challengedSegmentStart, uint256 challengedSegmentLength, bytes32[] chainHashes)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) WatchBisected(opts *bind.WatchOpts, sink chan<- *IOldChallengeManagerBisected, challengeIndex []uint64, challengeRoot [][32]byte) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}
	var challengeRootRule []interface{}
	for _, challengeRootItem := range challengeRoot {
		challengeRootRule = append(challengeRootRule, challengeRootItem)
	}

	logs, sub, err := _IOldChallengeManager.contract.WatchLogs(opts, "Bisected", challengeIndexRule, challengeRootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IOldChallengeManagerBisected)
				if err := _IOldChallengeManager.contract.UnpackLog(event, "Bisected", log); err != nil {
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

// ParseBisected is a log parse operation binding the contract event 0x86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d68.
//
// Solidity: event Bisected(uint64 indexed challengeIndex, bytes32 indexed challengeRoot, uint256 challengedSegmentStart, uint256 challengedSegmentLength, bytes32[] chainHashes)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) ParseBisected(log types.Log) (*IOldChallengeManagerBisected, error) {
	event := new(IOldChallengeManagerBisected)
	if err := _IOldChallengeManager.contract.UnpackLog(event, "Bisected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOldChallengeManagerChallengeEndedIterator is returned from FilterChallengeEnded and is used to iterate over the raw logs and unpacked data for ChallengeEnded events raised by the IOldChallengeManager contract.
type IOldChallengeManagerChallengeEndedIterator struct {
	Event *IOldChallengeManagerChallengeEnded // Event containing the contract specifics and raw log

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
func (it *IOldChallengeManagerChallengeEndedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IOldChallengeManagerChallengeEnded)
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
		it.Event = new(IOldChallengeManagerChallengeEnded)
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
func (it *IOldChallengeManagerChallengeEndedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IOldChallengeManagerChallengeEndedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IOldChallengeManagerChallengeEnded represents a ChallengeEnded event raised by the IOldChallengeManager contract.
type IOldChallengeManagerChallengeEnded struct {
	ChallengeIndex uint64
	Kind           uint8
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterChallengeEnded is a free log retrieval operation binding the contract event 0xfdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40.
//
// Solidity: event ChallengeEnded(uint64 indexed challengeIndex, uint8 kind)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) FilterChallengeEnded(opts *bind.FilterOpts, challengeIndex []uint64) (*IOldChallengeManagerChallengeEndedIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _IOldChallengeManager.contract.FilterLogs(opts, "ChallengeEnded", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeManagerChallengeEndedIterator{contract: _IOldChallengeManager.contract, event: "ChallengeEnded", logs: logs, sub: sub}, nil
}

// WatchChallengeEnded is a free log subscription operation binding the contract event 0xfdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40.
//
// Solidity: event ChallengeEnded(uint64 indexed challengeIndex, uint8 kind)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) WatchChallengeEnded(opts *bind.WatchOpts, sink chan<- *IOldChallengeManagerChallengeEnded, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _IOldChallengeManager.contract.WatchLogs(opts, "ChallengeEnded", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IOldChallengeManagerChallengeEnded)
				if err := _IOldChallengeManager.contract.UnpackLog(event, "ChallengeEnded", log); err != nil {
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

// ParseChallengeEnded is a log parse operation binding the contract event 0xfdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40.
//
// Solidity: event ChallengeEnded(uint64 indexed challengeIndex, uint8 kind)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) ParseChallengeEnded(log types.Log) (*IOldChallengeManagerChallengeEnded, error) {
	event := new(IOldChallengeManagerChallengeEnded)
	if err := _IOldChallengeManager.contract.UnpackLog(event, "ChallengeEnded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOldChallengeManagerExecutionChallengeBegunIterator is returned from FilterExecutionChallengeBegun and is used to iterate over the raw logs and unpacked data for ExecutionChallengeBegun events raised by the IOldChallengeManager contract.
type IOldChallengeManagerExecutionChallengeBegunIterator struct {
	Event *IOldChallengeManagerExecutionChallengeBegun // Event containing the contract specifics and raw log

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
func (it *IOldChallengeManagerExecutionChallengeBegunIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IOldChallengeManagerExecutionChallengeBegun)
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
		it.Event = new(IOldChallengeManagerExecutionChallengeBegun)
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
func (it *IOldChallengeManagerExecutionChallengeBegunIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IOldChallengeManagerExecutionChallengeBegunIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IOldChallengeManagerExecutionChallengeBegun represents a ExecutionChallengeBegun event raised by the IOldChallengeManager contract.
type IOldChallengeManagerExecutionChallengeBegun struct {
	ChallengeIndex uint64
	BlockSteps     *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterExecutionChallengeBegun is a free log retrieval operation binding the contract event 0x24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db.
//
// Solidity: event ExecutionChallengeBegun(uint64 indexed challengeIndex, uint256 blockSteps)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) FilterExecutionChallengeBegun(opts *bind.FilterOpts, challengeIndex []uint64) (*IOldChallengeManagerExecutionChallengeBegunIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _IOldChallengeManager.contract.FilterLogs(opts, "ExecutionChallengeBegun", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeManagerExecutionChallengeBegunIterator{contract: _IOldChallengeManager.contract, event: "ExecutionChallengeBegun", logs: logs, sub: sub}, nil
}

// WatchExecutionChallengeBegun is a free log subscription operation binding the contract event 0x24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db.
//
// Solidity: event ExecutionChallengeBegun(uint64 indexed challengeIndex, uint256 blockSteps)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) WatchExecutionChallengeBegun(opts *bind.WatchOpts, sink chan<- *IOldChallengeManagerExecutionChallengeBegun, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _IOldChallengeManager.contract.WatchLogs(opts, "ExecutionChallengeBegun", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IOldChallengeManagerExecutionChallengeBegun)
				if err := _IOldChallengeManager.contract.UnpackLog(event, "ExecutionChallengeBegun", log); err != nil {
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

// ParseExecutionChallengeBegun is a log parse operation binding the contract event 0x24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db.
//
// Solidity: event ExecutionChallengeBegun(uint64 indexed challengeIndex, uint256 blockSteps)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) ParseExecutionChallengeBegun(log types.Log) (*IOldChallengeManagerExecutionChallengeBegun, error) {
	event := new(IOldChallengeManagerExecutionChallengeBegun)
	if err := _IOldChallengeManager.contract.UnpackLog(event, "ExecutionChallengeBegun", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOldChallengeManagerInitiatedChallengeIterator is returned from FilterInitiatedChallenge and is used to iterate over the raw logs and unpacked data for InitiatedChallenge events raised by the IOldChallengeManager contract.
type IOldChallengeManagerInitiatedChallengeIterator struct {
	Event *IOldChallengeManagerInitiatedChallenge // Event containing the contract specifics and raw log

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
func (it *IOldChallengeManagerInitiatedChallengeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IOldChallengeManagerInitiatedChallenge)
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
		it.Event = new(IOldChallengeManagerInitiatedChallenge)
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
func (it *IOldChallengeManagerInitiatedChallengeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IOldChallengeManagerInitiatedChallengeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IOldChallengeManagerInitiatedChallenge represents a InitiatedChallenge event raised by the IOldChallengeManager contract.
type IOldChallengeManagerInitiatedChallenge struct {
	ChallengeIndex uint64
	StartState     GlobalState
	EndState       GlobalState
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterInitiatedChallenge is a free log retrieval operation binding the contract event 0x76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a.
//
// Solidity: event InitiatedChallenge(uint64 indexed challengeIndex, (bytes32[2],uint64[2]) startState, (bytes32[2],uint64[2]) endState)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) FilterInitiatedChallenge(opts *bind.FilterOpts, challengeIndex []uint64) (*IOldChallengeManagerInitiatedChallengeIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _IOldChallengeManager.contract.FilterLogs(opts, "InitiatedChallenge", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeManagerInitiatedChallengeIterator{contract: _IOldChallengeManager.contract, event: "InitiatedChallenge", logs: logs, sub: sub}, nil
}

// WatchInitiatedChallenge is a free log subscription operation binding the contract event 0x76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a.
//
// Solidity: event InitiatedChallenge(uint64 indexed challengeIndex, (bytes32[2],uint64[2]) startState, (bytes32[2],uint64[2]) endState)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) WatchInitiatedChallenge(opts *bind.WatchOpts, sink chan<- *IOldChallengeManagerInitiatedChallenge, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _IOldChallengeManager.contract.WatchLogs(opts, "InitiatedChallenge", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IOldChallengeManagerInitiatedChallenge)
				if err := _IOldChallengeManager.contract.UnpackLog(event, "InitiatedChallenge", log); err != nil {
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

// ParseInitiatedChallenge is a log parse operation binding the contract event 0x76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a.
//
// Solidity: event InitiatedChallenge(uint64 indexed challengeIndex, (bytes32[2],uint64[2]) startState, (bytes32[2],uint64[2]) endState)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) ParseInitiatedChallenge(log types.Log) (*IOldChallengeManagerInitiatedChallenge, error) {
	event := new(IOldChallengeManagerInitiatedChallenge)
	if err := _IOldChallengeManager.contract.UnpackLog(event, "InitiatedChallenge", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOldChallengeManagerOneStepProofCompletedIterator is returned from FilterOneStepProofCompleted and is used to iterate over the raw logs and unpacked data for OneStepProofCompleted events raised by the IOldChallengeManager contract.
type IOldChallengeManagerOneStepProofCompletedIterator struct {
	Event *IOldChallengeManagerOneStepProofCompleted // Event containing the contract specifics and raw log

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
func (it *IOldChallengeManagerOneStepProofCompletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IOldChallengeManagerOneStepProofCompleted)
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
		it.Event = new(IOldChallengeManagerOneStepProofCompleted)
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
func (it *IOldChallengeManagerOneStepProofCompletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IOldChallengeManagerOneStepProofCompletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IOldChallengeManagerOneStepProofCompleted represents a OneStepProofCompleted event raised by the IOldChallengeManager contract.
type IOldChallengeManagerOneStepProofCompleted struct {
	ChallengeIndex uint64
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterOneStepProofCompleted is a free log retrieval operation binding the contract event 0xc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e.
//
// Solidity: event OneStepProofCompleted(uint64 indexed challengeIndex)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) FilterOneStepProofCompleted(opts *bind.FilterOpts, challengeIndex []uint64) (*IOldChallengeManagerOneStepProofCompletedIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _IOldChallengeManager.contract.FilterLogs(opts, "OneStepProofCompleted", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeManagerOneStepProofCompletedIterator{contract: _IOldChallengeManager.contract, event: "OneStepProofCompleted", logs: logs, sub: sub}, nil
}

// WatchOneStepProofCompleted is a free log subscription operation binding the contract event 0xc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e.
//
// Solidity: event OneStepProofCompleted(uint64 indexed challengeIndex)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) WatchOneStepProofCompleted(opts *bind.WatchOpts, sink chan<- *IOldChallengeManagerOneStepProofCompleted, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _IOldChallengeManager.contract.WatchLogs(opts, "OneStepProofCompleted", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IOldChallengeManagerOneStepProofCompleted)
				if err := _IOldChallengeManager.contract.UnpackLog(event, "OneStepProofCompleted", log); err != nil {
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

// ParseOneStepProofCompleted is a log parse operation binding the contract event 0xc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e.
//
// Solidity: event OneStepProofCompleted(uint64 indexed challengeIndex)
func (_IOldChallengeManager *IOldChallengeManagerFilterer) ParseOneStepProofCompleted(log types.Log) (*IOldChallengeManagerOneStepProofCompleted, error) {
	event := new(IOldChallengeManagerOneStepProofCompleted)
	if err := _IOldChallengeManager.contract.UnpackLog(event, "OneStepProofCompleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOldChallengeResultReceiverMetaData contains all meta data concerning the IOldChallengeResultReceiver contract.
var IOldChallengeResultReceiverMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"challengeIndex\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"winner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"loser\",\"type\":\"address\"}],\"name\":\"completeChallenge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IOldChallengeResultReceiverABI is the input ABI used to generate the binding from.
// Deprecated: Use IOldChallengeResultReceiverMetaData.ABI instead.
var IOldChallengeResultReceiverABI = IOldChallengeResultReceiverMetaData.ABI

// IOldChallengeResultReceiver is an auto generated Go binding around an Ethereum contract.
type IOldChallengeResultReceiver struct {
	IOldChallengeResultReceiverCaller     // Read-only binding to the contract
	IOldChallengeResultReceiverTransactor // Write-only binding to the contract
	IOldChallengeResultReceiverFilterer   // Log filterer for contract events
}

// IOldChallengeResultReceiverCaller is an auto generated read-only Go binding around an Ethereum contract.
type IOldChallengeResultReceiverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOldChallengeResultReceiverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IOldChallengeResultReceiverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOldChallengeResultReceiverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IOldChallengeResultReceiverFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOldChallengeResultReceiverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IOldChallengeResultReceiverSession struct {
	Contract     *IOldChallengeResultReceiver // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                // Call options to use throughout this session
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// IOldChallengeResultReceiverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IOldChallengeResultReceiverCallerSession struct {
	Contract *IOldChallengeResultReceiverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                      // Call options to use throughout this session
}

// IOldChallengeResultReceiverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IOldChallengeResultReceiverTransactorSession struct {
	Contract     *IOldChallengeResultReceiverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                      // Transaction auth options to use throughout this session
}

// IOldChallengeResultReceiverRaw is an auto generated low-level Go binding around an Ethereum contract.
type IOldChallengeResultReceiverRaw struct {
	Contract *IOldChallengeResultReceiver // Generic contract binding to access the raw methods on
}

// IOldChallengeResultReceiverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IOldChallengeResultReceiverCallerRaw struct {
	Contract *IOldChallengeResultReceiverCaller // Generic read-only contract binding to access the raw methods on
}

// IOldChallengeResultReceiverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IOldChallengeResultReceiverTransactorRaw struct {
	Contract *IOldChallengeResultReceiverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIOldChallengeResultReceiver creates a new instance of IOldChallengeResultReceiver, bound to a specific deployed contract.
func NewIOldChallengeResultReceiver(address common.Address, backend bind.ContractBackend) (*IOldChallengeResultReceiver, error) {
	contract, err := bindIOldChallengeResultReceiver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeResultReceiver{IOldChallengeResultReceiverCaller: IOldChallengeResultReceiverCaller{contract: contract}, IOldChallengeResultReceiverTransactor: IOldChallengeResultReceiverTransactor{contract: contract}, IOldChallengeResultReceiverFilterer: IOldChallengeResultReceiverFilterer{contract: contract}}, nil
}

// NewIOldChallengeResultReceiverCaller creates a new read-only instance of IOldChallengeResultReceiver, bound to a specific deployed contract.
func NewIOldChallengeResultReceiverCaller(address common.Address, caller bind.ContractCaller) (*IOldChallengeResultReceiverCaller, error) {
	contract, err := bindIOldChallengeResultReceiver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeResultReceiverCaller{contract: contract}, nil
}

// NewIOldChallengeResultReceiverTransactor creates a new write-only instance of IOldChallengeResultReceiver, bound to a specific deployed contract.
func NewIOldChallengeResultReceiverTransactor(address common.Address, transactor bind.ContractTransactor) (*IOldChallengeResultReceiverTransactor, error) {
	contract, err := bindIOldChallengeResultReceiver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeResultReceiverTransactor{contract: contract}, nil
}

// NewIOldChallengeResultReceiverFilterer creates a new log filterer instance of IOldChallengeResultReceiver, bound to a specific deployed contract.
func NewIOldChallengeResultReceiverFilterer(address common.Address, filterer bind.ContractFilterer) (*IOldChallengeResultReceiverFilterer, error) {
	contract, err := bindIOldChallengeResultReceiver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IOldChallengeResultReceiverFilterer{contract: contract}, nil
}

// bindIOldChallengeResultReceiver binds a generic wrapper to an already deployed contract.
func bindIOldChallengeResultReceiver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IOldChallengeResultReceiverMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOldChallengeResultReceiver *IOldChallengeResultReceiverRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOldChallengeResultReceiver.Contract.IOldChallengeResultReceiverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOldChallengeResultReceiver *IOldChallengeResultReceiverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOldChallengeResultReceiver.Contract.IOldChallengeResultReceiverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOldChallengeResultReceiver *IOldChallengeResultReceiverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOldChallengeResultReceiver.Contract.IOldChallengeResultReceiverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOldChallengeResultReceiver *IOldChallengeResultReceiverCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOldChallengeResultReceiver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOldChallengeResultReceiver *IOldChallengeResultReceiverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOldChallengeResultReceiver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOldChallengeResultReceiver *IOldChallengeResultReceiverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOldChallengeResultReceiver.Contract.contract.Transact(opts, method, params...)
}

// CompleteChallenge is a paid mutator transaction binding the contract method 0x0357aa49.
//
// Solidity: function completeChallenge(uint256 challengeIndex, address winner, address loser) returns()
func (_IOldChallengeResultReceiver *IOldChallengeResultReceiverTransactor) CompleteChallenge(opts *bind.TransactOpts, challengeIndex *big.Int, winner common.Address, loser common.Address) (*types.Transaction, error) {
	return _IOldChallengeResultReceiver.contract.Transact(opts, "completeChallenge", challengeIndex, winner, loser)
}

// CompleteChallenge is a paid mutator transaction binding the contract method 0x0357aa49.
//
// Solidity: function completeChallenge(uint256 challengeIndex, address winner, address loser) returns()
func (_IOldChallengeResultReceiver *IOldChallengeResultReceiverSession) CompleteChallenge(challengeIndex *big.Int, winner common.Address, loser common.Address) (*types.Transaction, error) {
	return _IOldChallengeResultReceiver.Contract.CompleteChallenge(&_IOldChallengeResultReceiver.TransactOpts, challengeIndex, winner, loser)
}

// CompleteChallenge is a paid mutator transaction binding the contract method 0x0357aa49.
//
// Solidity: function completeChallenge(uint256 challengeIndex, address winner, address loser) returns()
func (_IOldChallengeResultReceiver *IOldChallengeResultReceiverTransactorSession) CompleteChallenge(challengeIndex *big.Int, winner common.Address, loser common.Address) (*types.Transaction, error) {
	return _IOldChallengeResultReceiver.Contract.CompleteChallenge(&_IOldChallengeResultReceiver.TransactOpts, challengeIndex, winner, loser)
}

// OldChallengeLibMetaData contains all meta data concerning the OldChallengeLib contract.
var OldChallengeLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220effda62e1c79d6b9c6200a4afbfee0e200460f4cc70b00cd87b25953cca3ba2464736f6c63430008110033",
}

// OldChallengeLibABI is the input ABI used to generate the binding from.
// Deprecated: Use OldChallengeLibMetaData.ABI instead.
var OldChallengeLibABI = OldChallengeLibMetaData.ABI

// OldChallengeLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OldChallengeLibMetaData.Bin instead.
var OldChallengeLibBin = OldChallengeLibMetaData.Bin

// DeployOldChallengeLib deploys a new Ethereum contract, binding an instance of OldChallengeLib to it.
func DeployOldChallengeLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OldChallengeLib, error) {
	parsed, err := OldChallengeLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OldChallengeLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OldChallengeLib{OldChallengeLibCaller: OldChallengeLibCaller{contract: contract}, OldChallengeLibTransactor: OldChallengeLibTransactor{contract: contract}, OldChallengeLibFilterer: OldChallengeLibFilterer{contract: contract}}, nil
}

// OldChallengeLib is an auto generated Go binding around an Ethereum contract.
type OldChallengeLib struct {
	OldChallengeLibCaller     // Read-only binding to the contract
	OldChallengeLibTransactor // Write-only binding to the contract
	OldChallengeLibFilterer   // Log filterer for contract events
}

// OldChallengeLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type OldChallengeLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OldChallengeLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OldChallengeLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OldChallengeLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OldChallengeLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OldChallengeLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OldChallengeLibSession struct {
	Contract     *OldChallengeLib  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OldChallengeLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OldChallengeLibCallerSession struct {
	Contract *OldChallengeLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// OldChallengeLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OldChallengeLibTransactorSession struct {
	Contract     *OldChallengeLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// OldChallengeLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type OldChallengeLibRaw struct {
	Contract *OldChallengeLib // Generic contract binding to access the raw methods on
}

// OldChallengeLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OldChallengeLibCallerRaw struct {
	Contract *OldChallengeLibCaller // Generic read-only contract binding to access the raw methods on
}

// OldChallengeLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OldChallengeLibTransactorRaw struct {
	Contract *OldChallengeLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOldChallengeLib creates a new instance of OldChallengeLib, bound to a specific deployed contract.
func NewOldChallengeLib(address common.Address, backend bind.ContractBackend) (*OldChallengeLib, error) {
	contract, err := bindOldChallengeLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OldChallengeLib{OldChallengeLibCaller: OldChallengeLibCaller{contract: contract}, OldChallengeLibTransactor: OldChallengeLibTransactor{contract: contract}, OldChallengeLibFilterer: OldChallengeLibFilterer{contract: contract}}, nil
}

// NewOldChallengeLibCaller creates a new read-only instance of OldChallengeLib, bound to a specific deployed contract.
func NewOldChallengeLibCaller(address common.Address, caller bind.ContractCaller) (*OldChallengeLibCaller, error) {
	contract, err := bindOldChallengeLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OldChallengeLibCaller{contract: contract}, nil
}

// NewOldChallengeLibTransactor creates a new write-only instance of OldChallengeLib, bound to a specific deployed contract.
func NewOldChallengeLibTransactor(address common.Address, transactor bind.ContractTransactor) (*OldChallengeLibTransactor, error) {
	contract, err := bindOldChallengeLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OldChallengeLibTransactor{contract: contract}, nil
}

// NewOldChallengeLibFilterer creates a new log filterer instance of OldChallengeLib, bound to a specific deployed contract.
func NewOldChallengeLibFilterer(address common.Address, filterer bind.ContractFilterer) (*OldChallengeLibFilterer, error) {
	contract, err := bindOldChallengeLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OldChallengeLibFilterer{contract: contract}, nil
}

// bindOldChallengeLib binds a generic wrapper to an already deployed contract.
func bindOldChallengeLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OldChallengeLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OldChallengeLib *OldChallengeLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OldChallengeLib.Contract.OldChallengeLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OldChallengeLib *OldChallengeLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OldChallengeLib.Contract.OldChallengeLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OldChallengeLib *OldChallengeLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OldChallengeLib.Contract.OldChallengeLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OldChallengeLib *OldChallengeLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OldChallengeLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OldChallengeLib *OldChallengeLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OldChallengeLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OldChallengeLib *OldChallengeLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OldChallengeLib.Contract.contract.Transact(opts, method, params...)
}

// OldChallengeManagerMetaData contains all meta data concerning the OldChallengeManager contract.
var OldChallengeManagerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"challengeRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"challengedSegmentStart\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"challengedSegmentLength\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32[]\",\"name\":\"chainHashes\",\"type\":\"bytes32[]\"}],\"name\":\"Bisected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"enumIOldChallengeManager.ChallengeTerminationType\",\"name\":\"kind\",\"type\":\"uint8\"}],\"name\":\"ChallengeEnded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockSteps\",\"type\":\"uint256\"}],\"name\":\"ExecutionChallengeBegun\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"indexed\":false,\"internalType\":\"structGlobalState\",\"name\":\"startState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"indexed\":false,\"internalType\":\"structGlobalState\",\"name\":\"endState\",\"type\":\"tuple\"}],\"name\":\"InitiatedChallenge\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"OneStepProofCompleted\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"oldSegmentsStart\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oldSegmentsLength\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"oldSegments\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"challengePosition\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.SegmentSelection\",\"name\":\"selection\",\"type\":\"tuple\"},{\"internalType\":\"bytes32[]\",\"name\":\"newSegments\",\"type\":\"bytes32[]\"}],\"name\":\"bisectExecution\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"oldSegmentsStart\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oldSegmentsLength\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"oldSegments\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"challengePosition\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.SegmentSelection\",\"name\":\"selection\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus[2]\",\"name\":\"machineStatuses\",\"type\":\"uint8[2]\"},{\"internalType\":\"bytes32[2]\",\"name\":\"globalStateHashes\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint256\",\"name\":\"numSteps\",\"type\":\"uint256\"}],\"name\":\"challengeExecution\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"challengeInfo\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timeLeft\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.Participant\",\"name\":\"current\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timeLeft\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.Participant\",\"name\":\"next\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"lastMoveTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"challengeStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"maxInboxMessages\",\"type\":\"uint64\"},{\"internalType\":\"enumOldChallengeLib.ChallengeMode\",\"name\":\"mode\",\"type\":\"uint8\"}],\"internalType\":\"structOldChallengeLib.Challenge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"challenges\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timeLeft\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.Participant\",\"name\":\"current\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timeLeft\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.Participant\",\"name\":\"next\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"lastMoveTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"challengeStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"maxInboxMessages\",\"type\":\"uint64\"},{\"internalType\":\"enumOldChallengeLib.ChallengeMode\",\"name\":\"mode\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"clearChallenge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot_\",\"type\":\"bytes32\"},{\"internalType\":\"enumMachineStatus[2]\",\"name\":\"startAndEndMachineStatuses_\",\"type\":\"uint8[2]\"},{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState[2]\",\"name\":\"startAndEndGlobalStates_\",\"type\":\"tuple[2]\"},{\"internalType\":\"uint64\",\"name\":\"numBlocks\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"asserter_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"challenger_\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asserterTimeLeft_\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"challengerTimeLeft_\",\"type\":\"uint256\"}],\"name\":\"createChallenge\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"currentResponder\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOldChallengeResultReceiver\",\"name\":\"resultReceiver_\",\"type\":\"address\"},{\"internalType\":\"contractISequencerInbox\",\"name\":\"sequencerInbox_\",\"type\":\"address\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge_\",\"type\":\"address\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"osp_\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"isTimedOut\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"oldSegmentsStart\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oldSegmentsLength\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"oldSegments\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"challengePosition\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.SegmentSelection\",\"name\":\"selection\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"oneStepProveExecution\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"osp\",\"outputs\":[{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"resultReceiver\",\"outputs\":[{\"internalType\":\"contractIOldChallengeResultReceiver\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"contractISequencerInbox\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"timeout\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalChallengesCreated\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a06040523060805234801561001457600080fd5b506080516134e7610030600039600061155501526134e76000f3fe608060405234801561001057600080fd5b50600436106101005760003560e01c80639ede42b911610097578063ee35f32711610066578063ee35f327146102f0578063f26a62c614610303578063f8c8765e14610316578063fb7be0a11461032957600080fd5b80639ede42b914610294578063a521b032146102b7578063d248d124146102ca578063e78cea92146102dd57600080fd5b806356e9df97116100d357806356e9df97146101a95780635ef489e6146101bc5780637fd07a9c146101d05780638f1d3776146101f057600080fd5b806314eab5e7146101055780631b45c86a1461013657806323a9ef231461014b5780633504f1d714610196575b600080fd5b610118610113366004612b8c565b61033c565b60405167ffffffffffffffff90911681526020015b60405180910390f35b610149610144366004612c1f565b61064c565b005b61017e610159366004612c1f565b67ffffffffffffffff166000908152600160205260409020546001600160a01b031690565b6040516001600160a01b03909116815260200161012d565b60025461017e906001600160a01b031681565b6101496101b7366004612c1f565b61072a565b6000546101189067ffffffffffffffff1681565b6101e36101de366004612c1f565b6108b4565b60405161012d9190612c64565b6102816101fe366004612cee565b6001602081815260009283526040928390208351808501855281546001600160a01b03908116825293820154818401528451808601909552600282015490931684526003810154918401919091526004810154600582015460068301546007909301549394939192909167ffffffffffffffff811690600160401b900460ff1687565b60405161012d9796959493929190612d07565b6102a76102a2366004612c1f565b6109dc565b604051901515815260200161012d565b6101496102c5366004612d7f565b610a04565b6101496102d8366004612e24565b610fbb565b60045461017e906001600160a01b031681565b60035461017e906001600160a01b031681565b60055461017e906001600160a01b031681565b610149610324366004612eb7565b61154b565b610149610337366004612f13565b6116f5565b6002546000906001600160a01b0316331461039e5760405162461bcd60e51b815260206004820152601060248201527f4f4e4c595f524f4c4c55505f4348414c0000000000000000000000000000000060448201526064015b60405180910390fd5b6040805160028082526060820183526000926020830190803683370190505090506103f46103cf60208b018b612fb8565b6103ef8a60005b608002018036038101906103ea9190613079565b611f09565b611fb2565b8160008151811061040757610407612fa2565b602090810291909101015261043689600160200201602081019061042b9190612fb8565b6103ef8a60016103d6565b8160018151811061044957610449612fa2565b6020908102919091010152600080548190819061046f9067ffffffffffffffff16613128565b825467ffffffffffffffff8083166101009490940a848102910219909116179092559091506104a0576104a061314f565b67ffffffffffffffff81166000908152600160205260408120600581018d9055906104db6104d6368d90038d0160808e01613079565b6120b7565b905060026104ef60408e0160208f01612fb8565b600281111561050057610500612c3a565b148061052f5750600061052361051e368e90038e0160808f01613079565b6120cc565b67ffffffffffffffff16115b15610542578061053e81613128565b9150505b6007820180546040805180820182526001600160a01b038d811680835260209283018d905260028801805473ffffffffffffffffffffffffffffffffffffffff199081169092179055600388018d905583518085018552918e16808352919092018b905286549091161785556001850189905542600486015567ffffffffffffffff84811668ffffffffffffffffff1990931692909217600160401b179092559051908416907f76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a9061061a908e9060808201906131ad565b60405180910390a26106398360008c67ffffffffffffffff16876120db565b5090925050505b98975050505050505050565b600067ffffffffffffffff8216600090815260016020526040902060070154600160401b900460ff16600281111561068657610686612c3a565b1415604051806040016040528060078152602001661393d7d0d2105360ca1b815250906106c65760405162461bcd60e51b815260040161039591906131c9565b506106d0816109dc565b61071c5760405162461bcd60e51b815260206004820152601060248201527f54494d454f55545f444541444c494e45000000000000000000000000000000006044820152606401610395565b610727816000612172565b50565b6002546001600160a01b031633146107845760405162461bcd60e51b815260206004820152601060248201527f4e4f545f5245535f5245434549564552000000000000000000000000000000006044820152606401610395565b600067ffffffffffffffff8216600090815260016020526040902060070154600160401b900460ff1660028111156107be576107be612c3a565b1415604051806040016040528060078152602001661393d7d0d2105360ca1b815250906107fe5760405162461bcd60e51b815260040161039591906131c9565b5067ffffffffffffffff81166000818152600160208190526040808320805473ffffffffffffffffffffffffffffffffffffffff1990811682559281018490556002810180549093169092556003808301849055600483018490556005830184905560068301939093556007909101805468ffffffffffffffffff19169055517ffdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40916108a991613217565b60405180910390a250565b6040805161012081018252600060e0820181815261010083018290528252825180840184528181526020808201839052830152918101829052606081018290526080810182905260a0810182905260c081019190915267ffffffffffffffff82811660009081526001602081815260409283902083516101208101855281546001600160a01b0390811660e0830190815294830154610100830152938152845180860186526002808401549095168152600383015481850152928101929092526004810154938201939093526005830154606082015260068301546080820152600783015493841660a08201529260c0840191600160401b90910460ff16908111156109c2576109c2612c3a565b60028111156109d3576109d3612c3a565b90525092915050565b67ffffffffffffffff811660009081526001602052604081206109fe906122c8565b92915050565b67ffffffffffffffff8416600090815260016020526040812080548692869290916001600160a01b03163314610a7c5760405162461bcd60e51b815260206004820152600b60248201527f4348414c5f53454e4445520000000000000000000000000000000000000000006044820152606401610395565b610a85846109dc565b15610ad25760405162461bcd60e51b815260206004820152600d60248201527f4348414c5f444541444c494e45000000000000000000000000000000000000006044820152606401610395565b6000826002811115610ae657610ae6612c3a565b03610b535760006007820154600160401b900460ff166002811115610b0d57610b0d612c3a565b1415604051806040016040528060078152602001661393d7d0d2105360ca1b81525090610b4d5760405162461bcd60e51b815260040161039591906131c9565b50610c70565b6001826002811115610b6757610b67612c3a565b03610be05760016007820154600160401b900460ff166002811115610b8e57610b8e612c3a565b14610bdb5760405162461bcd60e51b815260206004820152600e60248201527f4348414c5f4e4f545f424c4f434b0000000000000000000000000000000000006044820152606401610395565b610c70565b6002826002811115610bf457610bf4612c3a565b03610c685760026007820154600160401b900460ff166002811115610c1b57610c1b612c3a565b14610bdb5760405162461bcd60e51b815260206004820152601260248201527f4348414c5f4e4f545f455845435554494f4e00000000000000000000000000006044820152606401610395565b610c7061314f565b610cbe83356020850135610c876040870187613231565b808060200260200160405190810160405280939291908181526020018383602002808284376000920191909152506122e092505050565b816006015414610d105760405162461bcd60e51b815260206004820152600960248201527f4249535f535441544500000000000000000000000000000000000000000000006044820152606401610395565b6002610d1f6040850185613231565b90501080610d4a57506001610d376040850185613231565b610d429291506132a0565b836060013510155b15610d975760405162461bcd60e51b815260206004820152601160248201527f4241445f4348414c4c454e47455f504f530000000000000000000000000000006044820152606401610395565b600080610da389612317565b9150915060018111610df75760405162461bcd60e51b815260206004820152600960248201527f544f4f5f53484f525400000000000000000000000000000000000000000000006044820152606401610395565b806028811115610e05575060285b610e108160016132b3565b8814610e5e5760405162461bcd60e51b815260206004820152600c60248201527f57524f4e475f44454752454500000000000000000000000000000000000000006044820152606401610395565b50610ea88989896000818110610e7657610e76612fa2565b602002919091013590508a8a610e8d6001826132a0565b818110610e9c57610e9c612fa2565b905060200201356123a7565b610ee78a83838b8b808060200260200160405190810160405280939291908181526020018383602002808284376000920191909152506120db92505050565b50600090505b6007820154600160401b900460ff166002811115610f0d57610f0d612c3a565b03610f185750610fb2565b6040805180820190915281546001600160a01b03168152600182015460208201526004820154610f4890426132a0565b81602001818151610f5991906132a0565b90525060028201805483546001600160a01b0380831673ffffffffffffffffffffffffffffffffffffffff1992831617865560038601805460018801558551929093169116179091556020909101519055426004909101555b50505050505050565b67ffffffffffffffff84166000908152600160205260409020805485918591600291906001600160a01b031633146110355760405162461bcd60e51b815260206004820152600b60248201527f4348414c5f53454e4445520000000000000000000000000000000000000000006044820152606401610395565b61103e846109dc565b1561108b5760405162461bcd60e51b815260206004820152600d60248201527f4348414c5f444541444c494e45000000000000000000000000000000000000006044820152606401610395565b600082600281111561109f5761109f612c3a565b0361110c5760006007820154600160401b900460ff1660028111156110c6576110c6612c3a565b1415604051806040016040528060078152602001661393d7d0d2105360ca1b815250906111065760405162461bcd60e51b815260040161039591906131c9565b50611229565b600182600281111561112057611120612c3a565b036111995760016007820154600160401b900460ff16600281111561114757611147612c3a565b146111945760405162461bcd60e51b815260206004820152600e60248201527f4348414c5f4e4f545f424c4f434b0000000000000000000000000000000000006044820152606401610395565b611229565b60028260028111156111ad576111ad612c3a565b036112215760026007820154600160401b900460ff1660028111156111d4576111d4612c3a565b146111945760405162461bcd60e51b815260206004820152601260248201527f4348414c5f4e4f545f455845435554494f4e00000000000000000000000000006044820152606401610395565b61122961314f565b61124083356020850135610c876040870187613231565b8160060154146112925760405162461bcd60e51b815260206004820152600960248201527f4249535f535441544500000000000000000000000000000000000000000000006044820152606401610395565b60026112a16040850185613231565b905010806112cc575060016112b96040850185613231565b6112c49291506132a0565b836060013510155b156113195760405162461bcd60e51b815260206004820152601160248201527f4241445f4348414c4c454e47455f504f530000000000000000000000000000006044820152606401610395565b67ffffffffffffffff88166000908152600160205260408120908061133d8a612317565b9092509050600181146113925760405162461bcd60e51b815260206004820152600860248201527f544f4f5f4c4f4e470000000000000000000000000000000000000000000000006044820152606401610395565b506005805460408051606081018252600786015467ffffffffffffffff1681526004546001600160a01b03908116602083015293860154818301526000939092169163b5112fd29185906113e8908f018f613231565b8f606001358181106113fc576113fc612fa2565b905060200201358d8d6040518663ffffffff1660e01b81526004016114259594939291906132c6565b602060405180830381865afa158015611442573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114669190613328565b905061147560408b018b613231565b61148460608d013560016132b3565b81811061149357611493612fa2565b9050602002013581036114e85760405162461bcd60e51b815260206004820152600c60248201527f53414d455f4f53505f454e4400000000000000000000000000000000000000006044820152606401610395565b60405167ffffffffffffffff8c16907fc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e90600090a261153f8b67ffffffffffffffff16600090815260016020526040812060060155565b5060009150610eed9050565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036115e95760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c00000000000000000000000000000000000000006064820152608401610395565b6002546001600160a01b0316156116425760405162461bcd60e51b815260206004820152600c60248201527f414c52454144595f494e495400000000000000000000000000000000000000006044820152606401610395565b6001600160a01b0384166116985760405162461bcd60e51b815260206004820152601260248201527f4e4f5f524553554c545f524543454956455200000000000000000000000000006044820152606401610395565b600280546001600160a01b0395861673ffffffffffffffffffffffffffffffffffffffff19918216179091556003805494861694821694909417909355600480549285169284169290921790915560058054919093169116179055565b67ffffffffffffffff8516600090815260016020819052604090912080548792879290916001600160a01b031633146117705760405162461bcd60e51b815260206004820152600b60248201527f4348414c5f53454e4445520000000000000000000000000000000000000000006044820152606401610395565b611779846109dc565b156117c65760405162461bcd60e51b815260206004820152600d60248201527f4348414c5f444541444c494e45000000000000000000000000000000000000006044820152606401610395565b60008260028111156117da576117da612c3a565b036118475760006007820154600160401b900460ff16600281111561180157611801612c3a565b1415604051806040016040528060078152602001661393d7d0d2105360ca1b815250906118415760405162461bcd60e51b815260040161039591906131c9565b50611964565b600182600281111561185b5761185b612c3a565b036118d45760016007820154600160401b900460ff16600281111561188257611882612c3a565b146118cf5760405162461bcd60e51b815260206004820152600e60248201527f4348414c5f4e4f545f424c4f434b0000000000000000000000000000000000006044820152606401610395565b611964565b60028260028111156118e8576118e8612c3a565b0361195c5760026007820154600160401b900460ff16600281111561190f5761190f612c3a565b146118cf5760405162461bcd60e51b815260206004820152601260248201527f4348414c5f4e4f545f455845435554494f4e00000000000000000000000000006044820152606401610395565b61196461314f565b61197b83356020850135610c876040870187613231565b8160060154146119cd5760405162461bcd60e51b815260206004820152600960248201527f4249535f535441544500000000000000000000000000000000000000000000006044820152606401610395565b60026119dc6040850185613231565b90501080611a07575060016119f46040850185613231565b6119ff9291506132a0565b836060013510155b15611a545760405162461bcd60e51b815260206004820152601160248201527f4241445f4348414c4c454e47455f504f530000000000000000000000000000006044820152606401610395565b6001851015611aa55760405162461bcd60e51b815260206004820152601360248201527f4348414c4c454e47455f544f4f5f53484f5254000000000000000000000000006044820152606401610395565b65080000000000851115611afb5760405162461bcd60e51b815260206004820152601260248201527f4348414c4c454e47455f544f4f5f4c4f4e4700000000000000000000000000006044820152606401610395565b611b3d88611b1d611b0f60208b018b612fb8565b8960005b6020020135611fb2565b611b38611b3060408c0160208d01612fb8565b8a6001611b13565b6123a7565b67ffffffffffffffff891660009081526001602052604081209080611b618b612317565b9150915080600114611bb55760405162461bcd60e51b815260206004820152600860248201527f544f4f5f4c4f4e470000000000000000000000000000000000000000000000006044820152606401610395565b6001611bc460208c018c612fb8565b6002811115611bd557611bd5612c3a565b14611ca057611bea60408b0160208c01612fb8565b6002811115611bfb57611bfb612c3a565b611c0860208c018c612fb8565b6002811115611c1957611c19612c3a565b148015611c2a5750883560208a0135145b611c765760405162461bcd60e51b815260206004820152600d60248201527f48414c5445445f4348414e4745000000000000000000000000000000000000006044820152606401610395565b611c988c67ffffffffffffffff16600090815260016020526040812060060155565b505050611e38565b6002611cb260408c0160208d01612fb8565b6002811115611cc357611cc3612c3a565b03611d1c57883560208a013514611d1c5760405162461bcd60e51b815260206004820152600c60248201527f4552524f525f4348414e474500000000000000000000000000000000000000006044820152606401610395565b6040805160028082526060820183526000926020830190803683375050506005850154909150611d4e908b35906124a2565b81600081518110611d6157611d61612fa2565b6020908102919091010152611d8f8b6001602002016020810190611d859190612fb8565b60208c013561268a565b81600181518110611da257611da2612fa2565b60209081029190910101526007840180547fffffffffffffffffffffffffffffffffffffffffffffff00ffffffffffffffff1668020000000000000000179055611def8d60008b846120db565b8c67ffffffffffffffff167f24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db84604051611e2b91815260200190565b60405180910390a2505050505b60006007820154600160401b900460ff166002811115611e5a57611e5a612c3a565b03611e655750611eff565b6040805180820190915281546001600160a01b03168152600182015460208201526004820154611e9590426132a0565b81602001818151611ea691906132a0565b90525060028201805483546001600160a01b0380831673ffffffffffffffffffffffffffffffffffffffff1992831617865560038601805460018801558551929093169116179091556020909101519055426004909101555b5050505050505050565b80518051602091820151828401518051908401516040517f476c6f62616c2073746174653a0000000000000000000000000000000000000095810195909552602d850193909352604d8401919091527fffffffffffffffff00000000000000000000000000000000000000000000000060c091821b8116606d85015291901b166075820152600090607d015b604051602081830303815290604052805190602001209050919050565b60006001836002811115611fc857611fc8612c3a565b0361201e576040517f426c6f636b2073746174653a00000000000000000000000000000000000000006020820152602c8101839052604c015b6040516020818303038152906040528051906020012090506109fe565b600283600281111561203257612032612c3a565b0361206f576040517f426c6f636b2073746174652c206572726f7265643a0000000000000000000000602082015260358101839052605501612001565b60405162461bcd60e51b815260206004820152601060248201527f4241445f424c4f434b5f535441545553000000000000000000000000000000006044820152606401610395565b6020810151600090815b602002015192915050565b602081015160009060016120c1565b60018210156120ec576120ec61314f565b6002815110156120fe576120fe61314f565b600061210b8484846122e0565b67ffffffffffffffff8616600081815260016020526040908190206006018390555191925082917f86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d689061216390889088908890613341565b60405180910390a35050505050565b67ffffffffffffffff8216600081815260016020819052604080832060028082018054835473ffffffffffffffffffffffffffffffffffffffff19808216865596850188905595811690915560038301869055600480840187905560058401879055600684019690965560078301805468ffffffffffffffffff19169055905492517f0357aa49000000000000000000000000000000000000000000000000000000008152948501959095526001600160a01b03948516602485018190529285166044850181905290949293909290911690630357aa4990606401600060405180830381600087803b15801561226757600080fd5b505af115801561227b573d6000803e3d6000fd5b505050508467ffffffffffffffff167ffdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40856040516122b99190613217565b60405180910390a25050505050565b60018101546000906122d983612727565b1192915050565b60008383836040516020016122f793929190613396565b6040516020818303038152906040528051906020012090505b9392505050565b60008080600161232a6040860186613231565b6123359291506132a0565b90506123458160208601356133ee565b9150612355606085013583613402565b6123609085356132b3565b925060026123716040860186613231565b61237c9291506132a0565b8460600135036123a157612394816020860135613419565b61239e90836132b3565b91505b50915091565b816123b56040850185613231565b85606001358181106123c9576123c9612fa2565b905060200201351461241d5760405162461bcd60e51b815260206004820152600b60248201527f57524f4e475f53544152540000000000000000000000000000000000000000006044820152606401610395565b8061242b6040850185613231565b61243a606087013560016132b3565b81811061244957612449612fa2565b905060200201350361249d5760405162461bcd60e51b815260206004820152600860248201527f53414d455f454e440000000000000000000000000000000000000000000000006044820152606401610395565b505050565b60408051600380825260808201909252600091829190816020015b60408051808201909152600080825260208201528152602001906001900390816124bd57505060408051808201825260008082526020918201819052825180840190935260048352908201529091508160008151811061251f5761251f612fa2565b60200260200101819052506125626000604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b8160018151811061257557612575612fa2565b60200260200101819052506125b86000604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b816002815181106125cb576125cb612fa2565b602090810291909101810191909152604080518083018252838152815180830190925280825260009282019290925261261b60408051606080820183529181019182529081526000602082015290565b604080518082018252606080825260006020808401829052845161012081018652828152908101879052938401859052908301829052608083018a905260a0830181905260c0830181905260e083015261010082018890529061267d81612739565b9998505050505050505050565b600060018360028111156126a0576126a0612c3a565b036126dd576040517f4d616368696e652066696e69736865643a000000000000000000000000000000602082015260318101839052605101612001565b60028360028111156126f1576126f1612c3a565b0361206f576040517f4d616368696e65206572726f7265643a000000000000000000000000000000006020820152603001612001565b60008160040154426109fe91906132a0565b6000808251600281111561274f5761274f612c3a565b03612829576127618260200151612926565b61276e8360400151612926565b61277b84606001516129bc565b608085015160a086015160c087015160e0808901516101008a01516040517f4d616368696e652072756e6e696e673a00000000000000000000000000000000602082015260308101999099526050890197909752607088019590955260908701939093527fffffffff0000000000000000000000000000000000000000000000000000000091831b821660b0870152821b811660b486015291901b1660b883015260bc82015260dc01611f95565b60018251600281111561283e5761283e612c3a565b036128815760808201516040517f4d616368696e652066696e69736865643a00000000000000000000000000000060208201526031810191909152605101611f95565b60028251600281111561289657612896612c3a565b036128d95760808201516040517f4d616368696e65206572726f7265643a0000000000000000000000000000000060208201526030810191909152605001611f95565b60405162461bcd60e51b815260206004820152600f60248201527f4241445f4d4143485f53544154555300000000000000000000000000000000006044820152606401610395565b919050565b60208101518151515160005b818110156129b557835161294f9061294a9083612a60565b612a98565b6040517f56616c756520737461636b3a00000000000000000000000000000000000000006020820152602c810191909152604c8101849052606c0160405160208183030381529060405280519060200120925080806129ad9061342d565b915050612932565b5050919050565b602081015160005b825151811015612a5a576129f4836000015182815181106129e7576129e7612fa2565b6020026020010151612ab5565b6040517f537461636b206672616d6520737461636b3a000000000000000000000000000060208201526032810191909152605281018390526072016040516020818303038152906040528051906020012091508080612a529061342d565b9150506129c4565b50919050565b60408051808201909152600080825260208201528251805183908110612a8857612a88612fa2565b6020026020010151905092915050565b600081600001518260200151604051602001611f95929190613465565b6000612ac48260000151612a98565b602080840151604080860151606087015191517f537461636b206672616d653a000000000000000000000000000000000000000094810194909452602c840194909452604c8301919091527fffffffff0000000000000000000000000000000000000000000000000000000060e093841b8116606c840152921b9091166070820152607401611f95565b80604081018310156109fe57600080fd5b803567ffffffffffffffff8116811461292157600080fd5b6001600160a01b038116811461072757600080fd5b600080600080600080600080610200898b031215612ba957600080fd5b88359750612bba8a60208b01612b4e565b965061016089018a811115612bce57600080fd5b60608a019650612bdd81612b5f565b955050610180890135612bef81612b77565b93506101a0890135612c0081612b77565b979a96995094979396929592945050506101c0820135916101e0013590565b600060208284031215612c3157600080fd5b61231082612b5f565b634e487b7160e01b600052602160045260246000fd5b60038110612c6057612c60612c3a565b9052565b815180516001600160a01b0316825260209081015190820152610120810160208381015180516001600160a01b031660408501529081015160608401525060408301516080830152606083015160a0830152608083015160c083015267ffffffffffffffff60a08401511660e083015260c0830151612ce7610100840182612c50565b5092915050565b600060208284031215612d0057600080fd5b5035919050565b87516001600160a01b0316815260208089015190820152610120810187516001600160a01b03166040830152602088015160608301528660808301528560a08301528460c083015267ffffffffffffffff841660e0830152610640610100830184612c50565b600060808284031215612a5a57600080fd5b60008060008060608587031215612d9557600080fd5b612d9e85612b5f565b9350602085013567ffffffffffffffff80821115612dbb57600080fd5b612dc788838901612d6d565b94506040870135915080821115612ddd57600080fd5b818701915087601f830112612df157600080fd5b813581811115612e0057600080fd5b8860208260051b8501011115612e1557600080fd5b95989497505060200194505050565b60008060008060608587031215612e3a57600080fd5b612e4385612b5f565b9350602085013567ffffffffffffffff80821115612e6057600080fd5b612e6c88838901612d6d565b94506040870135915080821115612e8257600080fd5b818701915087601f830112612e9657600080fd5b813581811115612ea557600080fd5b886020828501011115612e1557600080fd5b60008060008060808587031215612ecd57600080fd5b8435612ed881612b77565b93506020850135612ee881612b77565b92506040850135612ef881612b77565b91506060850135612f0881612b77565b939692955090935050565b600080600080600060e08688031215612f2b57600080fd5b612f3486612b5f565b9450602086013567ffffffffffffffff811115612f5057600080fd5b612f5c88828901612d6d565b945050612f6c8760408801612b4e565b9250612f7b8760808801612b4e565b9497939650919460c0013592915050565b634e487b7160e01b600052604160045260246000fd5b634e487b7160e01b600052603260045260246000fd5b600060208284031215612fca57600080fd5b81356003811061231057600080fd5b6040805190810167ffffffffffffffff81118282101715612ffc57612ffc612f8c565b60405290565b600082601f83011261301357600080fd5b6040516040810181811067ffffffffffffffff8211171561303657613036612f8c565b806040525080604084018581111561304d57600080fd5b845b8181101561306e5761306081612b5f565b83526020928301920161304f565b509195945050505050565b60006080828403121561308b57600080fd5b6040516040810181811067ffffffffffffffff821117156130ae576130ae612f8c565b604052601f830184136130c057600080fd5b6130c8612fd9565b8060408501868111156130da57600080fd5b855b818110156130f45780358452602093840193016130dc565b508184526131028782613002565b6020850152509195945050505050565b634e487b7160e01b600052601160045260246000fd5b600067ffffffffffffffff80831681810361314557613145613112565b6001019392505050565b634e487b7160e01b600052600160045260246000fd5b6040818337604082016040820160005b60028110156131a65767ffffffffffffffff61319083612b5f565b1683526020928301929190910190600101613175565b5050505050565b61010081016131bc8285613165565b6123106080830184613165565b600060208083528351808285015260005b818110156131f6578581018301518582016040015282016131da565b506000604082860101526040601f19601f8301168501019250505092915050565b602081016004831061322b5761322b612c3a565b91905290565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe184360301811261326657600080fd5b83018035915067ffffffffffffffff82111561328157600080fd5b6020019150600581901b360382131561329957600080fd5b9250929050565b818103818111156109fe576109fe613112565b808201808211156109fe576109fe613112565b855181526001600160a01b0360208701511660208201526040860151604082015284606082015283608082015260c060a08201528160c0820152818360e0830137600081830160e090810191909152601f909201601f19160101949350505050565b60006020828403121561333a57600080fd5b5051919050565b6000606082018583526020858185015260606040850152818551808452608086019150828701935060005b818110156133885784518352938301939183019160010161336c565b509098975050505050505050565b83815260006020848184015260408301845182860160005b828110156133ca578151845292840192908401906001016133ae565b509198975050505050505050565b634e487b7160e01b600052601260045260246000fd5b6000826133fd576133fd6133d8565b500490565b80820281158282048414176109fe576109fe613112565b600082613428576134286133d8565b500690565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361345e5761345e613112565b5060010190565b7f56616c75653a0000000000000000000000000000000000000000000000000000815260006007841061349a5761349a612c3a565b5060f89290921b600683015260078201526027019056fea264697066735822122056218a60df42fc01923e69513d8e9c0826eb41ef1ef9e99107eca4331a09f55e64736f6c63430008110033",
}

// OldChallengeManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use OldChallengeManagerMetaData.ABI instead.
var OldChallengeManagerABI = OldChallengeManagerMetaData.ABI

// OldChallengeManagerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OldChallengeManagerMetaData.Bin instead.
var OldChallengeManagerBin = OldChallengeManagerMetaData.Bin

// DeployOldChallengeManager deploys a new Ethereum contract, binding an instance of OldChallengeManager to it.
func DeployOldChallengeManager(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OldChallengeManager, error) {
	parsed, err := OldChallengeManagerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OldChallengeManagerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OldChallengeManager{OldChallengeManagerCaller: OldChallengeManagerCaller{contract: contract}, OldChallengeManagerTransactor: OldChallengeManagerTransactor{contract: contract}, OldChallengeManagerFilterer: OldChallengeManagerFilterer{contract: contract}}, nil
}

// OldChallengeManager is an auto generated Go binding around an Ethereum contract.
type OldChallengeManager struct {
	OldChallengeManagerCaller     // Read-only binding to the contract
	OldChallengeManagerTransactor // Write-only binding to the contract
	OldChallengeManagerFilterer   // Log filterer for contract events
}

// OldChallengeManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type OldChallengeManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OldChallengeManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OldChallengeManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OldChallengeManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OldChallengeManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OldChallengeManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OldChallengeManagerSession struct {
	Contract     *OldChallengeManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// OldChallengeManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OldChallengeManagerCallerSession struct {
	Contract *OldChallengeManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// OldChallengeManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OldChallengeManagerTransactorSession struct {
	Contract     *OldChallengeManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// OldChallengeManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type OldChallengeManagerRaw struct {
	Contract *OldChallengeManager // Generic contract binding to access the raw methods on
}

// OldChallengeManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OldChallengeManagerCallerRaw struct {
	Contract *OldChallengeManagerCaller // Generic read-only contract binding to access the raw methods on
}

// OldChallengeManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OldChallengeManagerTransactorRaw struct {
	Contract *OldChallengeManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOldChallengeManager creates a new instance of OldChallengeManager, bound to a specific deployed contract.
func NewOldChallengeManager(address common.Address, backend bind.ContractBackend) (*OldChallengeManager, error) {
	contract, err := bindOldChallengeManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OldChallengeManager{OldChallengeManagerCaller: OldChallengeManagerCaller{contract: contract}, OldChallengeManagerTransactor: OldChallengeManagerTransactor{contract: contract}, OldChallengeManagerFilterer: OldChallengeManagerFilterer{contract: contract}}, nil
}

// NewOldChallengeManagerCaller creates a new read-only instance of OldChallengeManager, bound to a specific deployed contract.
func NewOldChallengeManagerCaller(address common.Address, caller bind.ContractCaller) (*OldChallengeManagerCaller, error) {
	contract, err := bindOldChallengeManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OldChallengeManagerCaller{contract: contract}, nil
}

// NewOldChallengeManagerTransactor creates a new write-only instance of OldChallengeManager, bound to a specific deployed contract.
func NewOldChallengeManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*OldChallengeManagerTransactor, error) {
	contract, err := bindOldChallengeManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OldChallengeManagerTransactor{contract: contract}, nil
}

// NewOldChallengeManagerFilterer creates a new log filterer instance of OldChallengeManager, bound to a specific deployed contract.
func NewOldChallengeManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*OldChallengeManagerFilterer, error) {
	contract, err := bindOldChallengeManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OldChallengeManagerFilterer{contract: contract}, nil
}

// bindOldChallengeManager binds a generic wrapper to an already deployed contract.
func bindOldChallengeManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OldChallengeManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OldChallengeManager *OldChallengeManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OldChallengeManager.Contract.OldChallengeManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OldChallengeManager *OldChallengeManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.OldChallengeManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OldChallengeManager *OldChallengeManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.OldChallengeManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OldChallengeManager *OldChallengeManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OldChallengeManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OldChallengeManager *OldChallengeManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OldChallengeManager *OldChallengeManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.contract.Transact(opts, method, params...)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OldChallengeManager *OldChallengeManagerCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OldChallengeManager.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OldChallengeManager *OldChallengeManagerSession) Bridge() (common.Address, error) {
	return _OldChallengeManager.Contract.Bridge(&_OldChallengeManager.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OldChallengeManager *OldChallengeManagerCallerSession) Bridge() (common.Address, error) {
	return _OldChallengeManager.Contract.Bridge(&_OldChallengeManager.CallOpts)
}

// ChallengeInfo is a free data retrieval call binding the contract method 0x7fd07a9c.
//
// Solidity: function challengeInfo(uint64 challengeIndex) view returns(((address,uint256),(address,uint256),uint256,bytes32,bytes32,uint64,uint8))
func (_OldChallengeManager *OldChallengeManagerCaller) ChallengeInfo(opts *bind.CallOpts, challengeIndex uint64) (OldChallengeLibChallenge, error) {
	var out []interface{}
	err := _OldChallengeManager.contract.Call(opts, &out, "challengeInfo", challengeIndex)

	if err != nil {
		return *new(OldChallengeLibChallenge), err
	}

	out0 := *abi.ConvertType(out[0], new(OldChallengeLibChallenge)).(*OldChallengeLibChallenge)

	return out0, err

}

// ChallengeInfo is a free data retrieval call binding the contract method 0x7fd07a9c.
//
// Solidity: function challengeInfo(uint64 challengeIndex) view returns(((address,uint256),(address,uint256),uint256,bytes32,bytes32,uint64,uint8))
func (_OldChallengeManager *OldChallengeManagerSession) ChallengeInfo(challengeIndex uint64) (OldChallengeLibChallenge, error) {
	return _OldChallengeManager.Contract.ChallengeInfo(&_OldChallengeManager.CallOpts, challengeIndex)
}

// ChallengeInfo is a free data retrieval call binding the contract method 0x7fd07a9c.
//
// Solidity: function challengeInfo(uint64 challengeIndex) view returns(((address,uint256),(address,uint256),uint256,bytes32,bytes32,uint64,uint8))
func (_OldChallengeManager *OldChallengeManagerCallerSession) ChallengeInfo(challengeIndex uint64) (OldChallengeLibChallenge, error) {
	return _OldChallengeManager.Contract.ChallengeInfo(&_OldChallengeManager.CallOpts, challengeIndex)
}

// Challenges is a free data retrieval call binding the contract method 0x8f1d3776.
//
// Solidity: function challenges(uint256 ) view returns((address,uint256) current, (address,uint256) next, uint256 lastMoveTimestamp, bytes32 wasmModuleRoot, bytes32 challengeStateHash, uint64 maxInboxMessages, uint8 mode)
func (_OldChallengeManager *OldChallengeManagerCaller) Challenges(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Current            OldChallengeLibParticipant
	Next               OldChallengeLibParticipant
	LastMoveTimestamp  *big.Int
	WasmModuleRoot     [32]byte
	ChallengeStateHash [32]byte
	MaxInboxMessages   uint64
	Mode               uint8
}, error) {
	var out []interface{}
	err := _OldChallengeManager.contract.Call(opts, &out, "challenges", arg0)

	outstruct := new(struct {
		Current            OldChallengeLibParticipant
		Next               OldChallengeLibParticipant
		LastMoveTimestamp  *big.Int
		WasmModuleRoot     [32]byte
		ChallengeStateHash [32]byte
		MaxInboxMessages   uint64
		Mode               uint8
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Current = *abi.ConvertType(out[0], new(OldChallengeLibParticipant)).(*OldChallengeLibParticipant)
	outstruct.Next = *abi.ConvertType(out[1], new(OldChallengeLibParticipant)).(*OldChallengeLibParticipant)
	outstruct.LastMoveTimestamp = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.WasmModuleRoot = *abi.ConvertType(out[3], new([32]byte)).(*[32]byte)
	outstruct.ChallengeStateHash = *abi.ConvertType(out[4], new([32]byte)).(*[32]byte)
	outstruct.MaxInboxMessages = *abi.ConvertType(out[5], new(uint64)).(*uint64)
	outstruct.Mode = *abi.ConvertType(out[6], new(uint8)).(*uint8)

	return *outstruct, err

}

// Challenges is a free data retrieval call binding the contract method 0x8f1d3776.
//
// Solidity: function challenges(uint256 ) view returns((address,uint256) current, (address,uint256) next, uint256 lastMoveTimestamp, bytes32 wasmModuleRoot, bytes32 challengeStateHash, uint64 maxInboxMessages, uint8 mode)
func (_OldChallengeManager *OldChallengeManagerSession) Challenges(arg0 *big.Int) (struct {
	Current            OldChallengeLibParticipant
	Next               OldChallengeLibParticipant
	LastMoveTimestamp  *big.Int
	WasmModuleRoot     [32]byte
	ChallengeStateHash [32]byte
	MaxInboxMessages   uint64
	Mode               uint8
}, error) {
	return _OldChallengeManager.Contract.Challenges(&_OldChallengeManager.CallOpts, arg0)
}

// Challenges is a free data retrieval call binding the contract method 0x8f1d3776.
//
// Solidity: function challenges(uint256 ) view returns((address,uint256) current, (address,uint256) next, uint256 lastMoveTimestamp, bytes32 wasmModuleRoot, bytes32 challengeStateHash, uint64 maxInboxMessages, uint8 mode)
func (_OldChallengeManager *OldChallengeManagerCallerSession) Challenges(arg0 *big.Int) (struct {
	Current            OldChallengeLibParticipant
	Next               OldChallengeLibParticipant
	LastMoveTimestamp  *big.Int
	WasmModuleRoot     [32]byte
	ChallengeStateHash [32]byte
	MaxInboxMessages   uint64
	Mode               uint8
}, error) {
	return _OldChallengeManager.Contract.Challenges(&_OldChallengeManager.CallOpts, arg0)
}

// CurrentResponder is a free data retrieval call binding the contract method 0x23a9ef23.
//
// Solidity: function currentResponder(uint64 challengeIndex) view returns(address)
func (_OldChallengeManager *OldChallengeManagerCaller) CurrentResponder(opts *bind.CallOpts, challengeIndex uint64) (common.Address, error) {
	var out []interface{}
	err := _OldChallengeManager.contract.Call(opts, &out, "currentResponder", challengeIndex)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CurrentResponder is a free data retrieval call binding the contract method 0x23a9ef23.
//
// Solidity: function currentResponder(uint64 challengeIndex) view returns(address)
func (_OldChallengeManager *OldChallengeManagerSession) CurrentResponder(challengeIndex uint64) (common.Address, error) {
	return _OldChallengeManager.Contract.CurrentResponder(&_OldChallengeManager.CallOpts, challengeIndex)
}

// CurrentResponder is a free data retrieval call binding the contract method 0x23a9ef23.
//
// Solidity: function currentResponder(uint64 challengeIndex) view returns(address)
func (_OldChallengeManager *OldChallengeManagerCallerSession) CurrentResponder(challengeIndex uint64) (common.Address, error) {
	return _OldChallengeManager.Contract.CurrentResponder(&_OldChallengeManager.CallOpts, challengeIndex)
}

// IsTimedOut is a free data retrieval call binding the contract method 0x9ede42b9.
//
// Solidity: function isTimedOut(uint64 challengeIndex) view returns(bool)
func (_OldChallengeManager *OldChallengeManagerCaller) IsTimedOut(opts *bind.CallOpts, challengeIndex uint64) (bool, error) {
	var out []interface{}
	err := _OldChallengeManager.contract.Call(opts, &out, "isTimedOut", challengeIndex)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsTimedOut is a free data retrieval call binding the contract method 0x9ede42b9.
//
// Solidity: function isTimedOut(uint64 challengeIndex) view returns(bool)
func (_OldChallengeManager *OldChallengeManagerSession) IsTimedOut(challengeIndex uint64) (bool, error) {
	return _OldChallengeManager.Contract.IsTimedOut(&_OldChallengeManager.CallOpts, challengeIndex)
}

// IsTimedOut is a free data retrieval call binding the contract method 0x9ede42b9.
//
// Solidity: function isTimedOut(uint64 challengeIndex) view returns(bool)
func (_OldChallengeManager *OldChallengeManagerCallerSession) IsTimedOut(challengeIndex uint64) (bool, error) {
	return _OldChallengeManager.Contract.IsTimedOut(&_OldChallengeManager.CallOpts, challengeIndex)
}

// Osp is a free data retrieval call binding the contract method 0xf26a62c6.
//
// Solidity: function osp() view returns(address)
func (_OldChallengeManager *OldChallengeManagerCaller) Osp(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OldChallengeManager.contract.Call(opts, &out, "osp")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Osp is a free data retrieval call binding the contract method 0xf26a62c6.
//
// Solidity: function osp() view returns(address)
func (_OldChallengeManager *OldChallengeManagerSession) Osp() (common.Address, error) {
	return _OldChallengeManager.Contract.Osp(&_OldChallengeManager.CallOpts)
}

// Osp is a free data retrieval call binding the contract method 0xf26a62c6.
//
// Solidity: function osp() view returns(address)
func (_OldChallengeManager *OldChallengeManagerCallerSession) Osp() (common.Address, error) {
	return _OldChallengeManager.Contract.Osp(&_OldChallengeManager.CallOpts)
}

// ResultReceiver is a free data retrieval call binding the contract method 0x3504f1d7.
//
// Solidity: function resultReceiver() view returns(address)
func (_OldChallengeManager *OldChallengeManagerCaller) ResultReceiver(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OldChallengeManager.contract.Call(opts, &out, "resultReceiver")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ResultReceiver is a free data retrieval call binding the contract method 0x3504f1d7.
//
// Solidity: function resultReceiver() view returns(address)
func (_OldChallengeManager *OldChallengeManagerSession) ResultReceiver() (common.Address, error) {
	return _OldChallengeManager.Contract.ResultReceiver(&_OldChallengeManager.CallOpts)
}

// ResultReceiver is a free data retrieval call binding the contract method 0x3504f1d7.
//
// Solidity: function resultReceiver() view returns(address)
func (_OldChallengeManager *OldChallengeManagerCallerSession) ResultReceiver() (common.Address, error) {
	return _OldChallengeManager.Contract.ResultReceiver(&_OldChallengeManager.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_OldChallengeManager *OldChallengeManagerCaller) SequencerInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OldChallengeManager.contract.Call(opts, &out, "sequencerInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_OldChallengeManager *OldChallengeManagerSession) SequencerInbox() (common.Address, error) {
	return _OldChallengeManager.Contract.SequencerInbox(&_OldChallengeManager.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_OldChallengeManager *OldChallengeManagerCallerSession) SequencerInbox() (common.Address, error) {
	return _OldChallengeManager.Contract.SequencerInbox(&_OldChallengeManager.CallOpts)
}

// TotalChallengesCreated is a free data retrieval call binding the contract method 0x5ef489e6.
//
// Solidity: function totalChallengesCreated() view returns(uint64)
func (_OldChallengeManager *OldChallengeManagerCaller) TotalChallengesCreated(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _OldChallengeManager.contract.Call(opts, &out, "totalChallengesCreated")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// TotalChallengesCreated is a free data retrieval call binding the contract method 0x5ef489e6.
//
// Solidity: function totalChallengesCreated() view returns(uint64)
func (_OldChallengeManager *OldChallengeManagerSession) TotalChallengesCreated() (uint64, error) {
	return _OldChallengeManager.Contract.TotalChallengesCreated(&_OldChallengeManager.CallOpts)
}

// TotalChallengesCreated is a free data retrieval call binding the contract method 0x5ef489e6.
//
// Solidity: function totalChallengesCreated() view returns(uint64)
func (_OldChallengeManager *OldChallengeManagerCallerSession) TotalChallengesCreated() (uint64, error) {
	return _OldChallengeManager.Contract.TotalChallengesCreated(&_OldChallengeManager.CallOpts)
}

// BisectExecution is a paid mutator transaction binding the contract method 0xa521b032.
//
// Solidity: function bisectExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes32[] newSegments) returns()
func (_OldChallengeManager *OldChallengeManagerTransactor) BisectExecution(opts *bind.TransactOpts, challengeIndex uint64, selection OldChallengeLibSegmentSelection, newSegments [][32]byte) (*types.Transaction, error) {
	return _OldChallengeManager.contract.Transact(opts, "bisectExecution", challengeIndex, selection, newSegments)
}

// BisectExecution is a paid mutator transaction binding the contract method 0xa521b032.
//
// Solidity: function bisectExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes32[] newSegments) returns()
func (_OldChallengeManager *OldChallengeManagerSession) BisectExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, newSegments [][32]byte) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.BisectExecution(&_OldChallengeManager.TransactOpts, challengeIndex, selection, newSegments)
}

// BisectExecution is a paid mutator transaction binding the contract method 0xa521b032.
//
// Solidity: function bisectExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes32[] newSegments) returns()
func (_OldChallengeManager *OldChallengeManagerTransactorSession) BisectExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, newSegments [][32]byte) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.BisectExecution(&_OldChallengeManager.TransactOpts, challengeIndex, selection, newSegments)
}

// ChallengeExecution is a paid mutator transaction binding the contract method 0xfb7be0a1.
//
// Solidity: function challengeExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, uint8[2] machineStatuses, bytes32[2] globalStateHashes, uint256 numSteps) returns()
func (_OldChallengeManager *OldChallengeManagerTransactor) ChallengeExecution(opts *bind.TransactOpts, challengeIndex uint64, selection OldChallengeLibSegmentSelection, machineStatuses [2]uint8, globalStateHashes [2][32]byte, numSteps *big.Int) (*types.Transaction, error) {
	return _OldChallengeManager.contract.Transact(opts, "challengeExecution", challengeIndex, selection, machineStatuses, globalStateHashes, numSteps)
}

// ChallengeExecution is a paid mutator transaction binding the contract method 0xfb7be0a1.
//
// Solidity: function challengeExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, uint8[2] machineStatuses, bytes32[2] globalStateHashes, uint256 numSteps) returns()
func (_OldChallengeManager *OldChallengeManagerSession) ChallengeExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, machineStatuses [2]uint8, globalStateHashes [2][32]byte, numSteps *big.Int) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.ChallengeExecution(&_OldChallengeManager.TransactOpts, challengeIndex, selection, machineStatuses, globalStateHashes, numSteps)
}

// ChallengeExecution is a paid mutator transaction binding the contract method 0xfb7be0a1.
//
// Solidity: function challengeExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, uint8[2] machineStatuses, bytes32[2] globalStateHashes, uint256 numSteps) returns()
func (_OldChallengeManager *OldChallengeManagerTransactorSession) ChallengeExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, machineStatuses [2]uint8, globalStateHashes [2][32]byte, numSteps *big.Int) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.ChallengeExecution(&_OldChallengeManager.TransactOpts, challengeIndex, selection, machineStatuses, globalStateHashes, numSteps)
}

// ClearChallenge is a paid mutator transaction binding the contract method 0x56e9df97.
//
// Solidity: function clearChallenge(uint64 challengeIndex) returns()
func (_OldChallengeManager *OldChallengeManagerTransactor) ClearChallenge(opts *bind.TransactOpts, challengeIndex uint64) (*types.Transaction, error) {
	return _OldChallengeManager.contract.Transact(opts, "clearChallenge", challengeIndex)
}

// ClearChallenge is a paid mutator transaction binding the contract method 0x56e9df97.
//
// Solidity: function clearChallenge(uint64 challengeIndex) returns()
func (_OldChallengeManager *OldChallengeManagerSession) ClearChallenge(challengeIndex uint64) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.ClearChallenge(&_OldChallengeManager.TransactOpts, challengeIndex)
}

// ClearChallenge is a paid mutator transaction binding the contract method 0x56e9df97.
//
// Solidity: function clearChallenge(uint64 challengeIndex) returns()
func (_OldChallengeManager *OldChallengeManagerTransactorSession) ClearChallenge(challengeIndex uint64) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.ClearChallenge(&_OldChallengeManager.TransactOpts, challengeIndex)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_OldChallengeManager *OldChallengeManagerTransactor) CreateChallenge(opts *bind.TransactOpts, wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _OldChallengeManager.contract.Transact(opts, "createChallenge", wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_OldChallengeManager *OldChallengeManagerSession) CreateChallenge(wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.CreateChallenge(&_OldChallengeManager.TransactOpts, wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_OldChallengeManager *OldChallengeManagerTransactorSession) CreateChallenge(wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.CreateChallenge(&_OldChallengeManager.TransactOpts, wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address resultReceiver_, address sequencerInbox_, address bridge_, address osp_) returns()
func (_OldChallengeManager *OldChallengeManagerTransactor) Initialize(opts *bind.TransactOpts, resultReceiver_ common.Address, sequencerInbox_ common.Address, bridge_ common.Address, osp_ common.Address) (*types.Transaction, error) {
	return _OldChallengeManager.contract.Transact(opts, "initialize", resultReceiver_, sequencerInbox_, bridge_, osp_)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address resultReceiver_, address sequencerInbox_, address bridge_, address osp_) returns()
func (_OldChallengeManager *OldChallengeManagerSession) Initialize(resultReceiver_ common.Address, sequencerInbox_ common.Address, bridge_ common.Address, osp_ common.Address) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.Initialize(&_OldChallengeManager.TransactOpts, resultReceiver_, sequencerInbox_, bridge_, osp_)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address resultReceiver_, address sequencerInbox_, address bridge_, address osp_) returns()
func (_OldChallengeManager *OldChallengeManagerTransactorSession) Initialize(resultReceiver_ common.Address, sequencerInbox_ common.Address, bridge_ common.Address, osp_ common.Address) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.Initialize(&_OldChallengeManager.TransactOpts, resultReceiver_, sequencerInbox_, bridge_, osp_)
}

// OneStepProveExecution is a paid mutator transaction binding the contract method 0xd248d124.
//
// Solidity: function oneStepProveExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes proof) returns()
func (_OldChallengeManager *OldChallengeManagerTransactor) OneStepProveExecution(opts *bind.TransactOpts, challengeIndex uint64, selection OldChallengeLibSegmentSelection, proof []byte) (*types.Transaction, error) {
	return _OldChallengeManager.contract.Transact(opts, "oneStepProveExecution", challengeIndex, selection, proof)
}

// OneStepProveExecution is a paid mutator transaction binding the contract method 0xd248d124.
//
// Solidity: function oneStepProveExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes proof) returns()
func (_OldChallengeManager *OldChallengeManagerSession) OneStepProveExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, proof []byte) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.OneStepProveExecution(&_OldChallengeManager.TransactOpts, challengeIndex, selection, proof)
}

// OneStepProveExecution is a paid mutator transaction binding the contract method 0xd248d124.
//
// Solidity: function oneStepProveExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes proof) returns()
func (_OldChallengeManager *OldChallengeManagerTransactorSession) OneStepProveExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, proof []byte) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.OneStepProveExecution(&_OldChallengeManager.TransactOpts, challengeIndex, selection, proof)
}

// Timeout is a paid mutator transaction binding the contract method 0x1b45c86a.
//
// Solidity: function timeout(uint64 challengeIndex) returns()
func (_OldChallengeManager *OldChallengeManagerTransactor) Timeout(opts *bind.TransactOpts, challengeIndex uint64) (*types.Transaction, error) {
	return _OldChallengeManager.contract.Transact(opts, "timeout", challengeIndex)
}

// Timeout is a paid mutator transaction binding the contract method 0x1b45c86a.
//
// Solidity: function timeout(uint64 challengeIndex) returns()
func (_OldChallengeManager *OldChallengeManagerSession) Timeout(challengeIndex uint64) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.Timeout(&_OldChallengeManager.TransactOpts, challengeIndex)
}

// Timeout is a paid mutator transaction binding the contract method 0x1b45c86a.
//
// Solidity: function timeout(uint64 challengeIndex) returns()
func (_OldChallengeManager *OldChallengeManagerTransactorSession) Timeout(challengeIndex uint64) (*types.Transaction, error) {
	return _OldChallengeManager.Contract.Timeout(&_OldChallengeManager.TransactOpts, challengeIndex)
}

// OldChallengeManagerBisectedIterator is returned from FilterBisected and is used to iterate over the raw logs and unpacked data for Bisected events raised by the OldChallengeManager contract.
type OldChallengeManagerBisectedIterator struct {
	Event *OldChallengeManagerBisected // Event containing the contract specifics and raw log

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
func (it *OldChallengeManagerBisectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OldChallengeManagerBisected)
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
		it.Event = new(OldChallengeManagerBisected)
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
func (it *OldChallengeManagerBisectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OldChallengeManagerBisectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OldChallengeManagerBisected represents a Bisected event raised by the OldChallengeManager contract.
type OldChallengeManagerBisected struct {
	ChallengeIndex          uint64
	ChallengeRoot           [32]byte
	ChallengedSegmentStart  *big.Int
	ChallengedSegmentLength *big.Int
	ChainHashes             [][32]byte
	Raw                     types.Log // Blockchain specific contextual infos
}

// FilterBisected is a free log retrieval operation binding the contract event 0x86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d68.
//
// Solidity: event Bisected(uint64 indexed challengeIndex, bytes32 indexed challengeRoot, uint256 challengedSegmentStart, uint256 challengedSegmentLength, bytes32[] chainHashes)
func (_OldChallengeManager *OldChallengeManagerFilterer) FilterBisected(opts *bind.FilterOpts, challengeIndex []uint64, challengeRoot [][32]byte) (*OldChallengeManagerBisectedIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}
	var challengeRootRule []interface{}
	for _, challengeRootItem := range challengeRoot {
		challengeRootRule = append(challengeRootRule, challengeRootItem)
	}

	logs, sub, err := _OldChallengeManager.contract.FilterLogs(opts, "Bisected", challengeIndexRule, challengeRootRule)
	if err != nil {
		return nil, err
	}
	return &OldChallengeManagerBisectedIterator{contract: _OldChallengeManager.contract, event: "Bisected", logs: logs, sub: sub}, nil
}

// WatchBisected is a free log subscription operation binding the contract event 0x86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d68.
//
// Solidity: event Bisected(uint64 indexed challengeIndex, bytes32 indexed challengeRoot, uint256 challengedSegmentStart, uint256 challengedSegmentLength, bytes32[] chainHashes)
func (_OldChallengeManager *OldChallengeManagerFilterer) WatchBisected(opts *bind.WatchOpts, sink chan<- *OldChallengeManagerBisected, challengeIndex []uint64, challengeRoot [][32]byte) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}
	var challengeRootRule []interface{}
	for _, challengeRootItem := range challengeRoot {
		challengeRootRule = append(challengeRootRule, challengeRootItem)
	}

	logs, sub, err := _OldChallengeManager.contract.WatchLogs(opts, "Bisected", challengeIndexRule, challengeRootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OldChallengeManagerBisected)
				if err := _OldChallengeManager.contract.UnpackLog(event, "Bisected", log); err != nil {
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

// ParseBisected is a log parse operation binding the contract event 0x86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d68.
//
// Solidity: event Bisected(uint64 indexed challengeIndex, bytes32 indexed challengeRoot, uint256 challengedSegmentStart, uint256 challengedSegmentLength, bytes32[] chainHashes)
func (_OldChallengeManager *OldChallengeManagerFilterer) ParseBisected(log types.Log) (*OldChallengeManagerBisected, error) {
	event := new(OldChallengeManagerBisected)
	if err := _OldChallengeManager.contract.UnpackLog(event, "Bisected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OldChallengeManagerChallengeEndedIterator is returned from FilterChallengeEnded and is used to iterate over the raw logs and unpacked data for ChallengeEnded events raised by the OldChallengeManager contract.
type OldChallengeManagerChallengeEndedIterator struct {
	Event *OldChallengeManagerChallengeEnded // Event containing the contract specifics and raw log

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
func (it *OldChallengeManagerChallengeEndedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OldChallengeManagerChallengeEnded)
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
		it.Event = new(OldChallengeManagerChallengeEnded)
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
func (it *OldChallengeManagerChallengeEndedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OldChallengeManagerChallengeEndedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OldChallengeManagerChallengeEnded represents a ChallengeEnded event raised by the OldChallengeManager contract.
type OldChallengeManagerChallengeEnded struct {
	ChallengeIndex uint64
	Kind           uint8
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterChallengeEnded is a free log retrieval operation binding the contract event 0xfdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40.
//
// Solidity: event ChallengeEnded(uint64 indexed challengeIndex, uint8 kind)
func (_OldChallengeManager *OldChallengeManagerFilterer) FilterChallengeEnded(opts *bind.FilterOpts, challengeIndex []uint64) (*OldChallengeManagerChallengeEndedIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _OldChallengeManager.contract.FilterLogs(opts, "ChallengeEnded", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &OldChallengeManagerChallengeEndedIterator{contract: _OldChallengeManager.contract, event: "ChallengeEnded", logs: logs, sub: sub}, nil
}

// WatchChallengeEnded is a free log subscription operation binding the contract event 0xfdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40.
//
// Solidity: event ChallengeEnded(uint64 indexed challengeIndex, uint8 kind)
func (_OldChallengeManager *OldChallengeManagerFilterer) WatchChallengeEnded(opts *bind.WatchOpts, sink chan<- *OldChallengeManagerChallengeEnded, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _OldChallengeManager.contract.WatchLogs(opts, "ChallengeEnded", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OldChallengeManagerChallengeEnded)
				if err := _OldChallengeManager.contract.UnpackLog(event, "ChallengeEnded", log); err != nil {
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

// ParseChallengeEnded is a log parse operation binding the contract event 0xfdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40.
//
// Solidity: event ChallengeEnded(uint64 indexed challengeIndex, uint8 kind)
func (_OldChallengeManager *OldChallengeManagerFilterer) ParseChallengeEnded(log types.Log) (*OldChallengeManagerChallengeEnded, error) {
	event := new(OldChallengeManagerChallengeEnded)
	if err := _OldChallengeManager.contract.UnpackLog(event, "ChallengeEnded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OldChallengeManagerExecutionChallengeBegunIterator is returned from FilterExecutionChallengeBegun and is used to iterate over the raw logs and unpacked data for ExecutionChallengeBegun events raised by the OldChallengeManager contract.
type OldChallengeManagerExecutionChallengeBegunIterator struct {
	Event *OldChallengeManagerExecutionChallengeBegun // Event containing the contract specifics and raw log

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
func (it *OldChallengeManagerExecutionChallengeBegunIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OldChallengeManagerExecutionChallengeBegun)
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
		it.Event = new(OldChallengeManagerExecutionChallengeBegun)
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
func (it *OldChallengeManagerExecutionChallengeBegunIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OldChallengeManagerExecutionChallengeBegunIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OldChallengeManagerExecutionChallengeBegun represents a ExecutionChallengeBegun event raised by the OldChallengeManager contract.
type OldChallengeManagerExecutionChallengeBegun struct {
	ChallengeIndex uint64
	BlockSteps     *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterExecutionChallengeBegun is a free log retrieval operation binding the contract event 0x24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db.
//
// Solidity: event ExecutionChallengeBegun(uint64 indexed challengeIndex, uint256 blockSteps)
func (_OldChallengeManager *OldChallengeManagerFilterer) FilterExecutionChallengeBegun(opts *bind.FilterOpts, challengeIndex []uint64) (*OldChallengeManagerExecutionChallengeBegunIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _OldChallengeManager.contract.FilterLogs(opts, "ExecutionChallengeBegun", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &OldChallengeManagerExecutionChallengeBegunIterator{contract: _OldChallengeManager.contract, event: "ExecutionChallengeBegun", logs: logs, sub: sub}, nil
}

// WatchExecutionChallengeBegun is a free log subscription operation binding the contract event 0x24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db.
//
// Solidity: event ExecutionChallengeBegun(uint64 indexed challengeIndex, uint256 blockSteps)
func (_OldChallengeManager *OldChallengeManagerFilterer) WatchExecutionChallengeBegun(opts *bind.WatchOpts, sink chan<- *OldChallengeManagerExecutionChallengeBegun, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _OldChallengeManager.contract.WatchLogs(opts, "ExecutionChallengeBegun", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OldChallengeManagerExecutionChallengeBegun)
				if err := _OldChallengeManager.contract.UnpackLog(event, "ExecutionChallengeBegun", log); err != nil {
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

// ParseExecutionChallengeBegun is a log parse operation binding the contract event 0x24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db.
//
// Solidity: event ExecutionChallengeBegun(uint64 indexed challengeIndex, uint256 blockSteps)
func (_OldChallengeManager *OldChallengeManagerFilterer) ParseExecutionChallengeBegun(log types.Log) (*OldChallengeManagerExecutionChallengeBegun, error) {
	event := new(OldChallengeManagerExecutionChallengeBegun)
	if err := _OldChallengeManager.contract.UnpackLog(event, "ExecutionChallengeBegun", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OldChallengeManagerInitiatedChallengeIterator is returned from FilterInitiatedChallenge and is used to iterate over the raw logs and unpacked data for InitiatedChallenge events raised by the OldChallengeManager contract.
type OldChallengeManagerInitiatedChallengeIterator struct {
	Event *OldChallengeManagerInitiatedChallenge // Event containing the contract specifics and raw log

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
func (it *OldChallengeManagerInitiatedChallengeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OldChallengeManagerInitiatedChallenge)
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
		it.Event = new(OldChallengeManagerInitiatedChallenge)
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
func (it *OldChallengeManagerInitiatedChallengeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OldChallengeManagerInitiatedChallengeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OldChallengeManagerInitiatedChallenge represents a InitiatedChallenge event raised by the OldChallengeManager contract.
type OldChallengeManagerInitiatedChallenge struct {
	ChallengeIndex uint64
	StartState     GlobalState
	EndState       GlobalState
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterInitiatedChallenge is a free log retrieval operation binding the contract event 0x76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a.
//
// Solidity: event InitiatedChallenge(uint64 indexed challengeIndex, (bytes32[2],uint64[2]) startState, (bytes32[2],uint64[2]) endState)
func (_OldChallengeManager *OldChallengeManagerFilterer) FilterInitiatedChallenge(opts *bind.FilterOpts, challengeIndex []uint64) (*OldChallengeManagerInitiatedChallengeIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _OldChallengeManager.contract.FilterLogs(opts, "InitiatedChallenge", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &OldChallengeManagerInitiatedChallengeIterator{contract: _OldChallengeManager.contract, event: "InitiatedChallenge", logs: logs, sub: sub}, nil
}

// WatchInitiatedChallenge is a free log subscription operation binding the contract event 0x76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a.
//
// Solidity: event InitiatedChallenge(uint64 indexed challengeIndex, (bytes32[2],uint64[2]) startState, (bytes32[2],uint64[2]) endState)
func (_OldChallengeManager *OldChallengeManagerFilterer) WatchInitiatedChallenge(opts *bind.WatchOpts, sink chan<- *OldChallengeManagerInitiatedChallenge, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _OldChallengeManager.contract.WatchLogs(opts, "InitiatedChallenge", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OldChallengeManagerInitiatedChallenge)
				if err := _OldChallengeManager.contract.UnpackLog(event, "InitiatedChallenge", log); err != nil {
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

// ParseInitiatedChallenge is a log parse operation binding the contract event 0x76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a.
//
// Solidity: event InitiatedChallenge(uint64 indexed challengeIndex, (bytes32[2],uint64[2]) startState, (bytes32[2],uint64[2]) endState)
func (_OldChallengeManager *OldChallengeManagerFilterer) ParseInitiatedChallenge(log types.Log) (*OldChallengeManagerInitiatedChallenge, error) {
	event := new(OldChallengeManagerInitiatedChallenge)
	if err := _OldChallengeManager.contract.UnpackLog(event, "InitiatedChallenge", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OldChallengeManagerOneStepProofCompletedIterator is returned from FilterOneStepProofCompleted and is used to iterate over the raw logs and unpacked data for OneStepProofCompleted events raised by the OldChallengeManager contract.
type OldChallengeManagerOneStepProofCompletedIterator struct {
	Event *OldChallengeManagerOneStepProofCompleted // Event containing the contract specifics and raw log

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
func (it *OldChallengeManagerOneStepProofCompletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OldChallengeManagerOneStepProofCompleted)
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
		it.Event = new(OldChallengeManagerOneStepProofCompleted)
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
func (it *OldChallengeManagerOneStepProofCompletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OldChallengeManagerOneStepProofCompletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OldChallengeManagerOneStepProofCompleted represents a OneStepProofCompleted event raised by the OldChallengeManager contract.
type OldChallengeManagerOneStepProofCompleted struct {
	ChallengeIndex uint64
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterOneStepProofCompleted is a free log retrieval operation binding the contract event 0xc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e.
//
// Solidity: event OneStepProofCompleted(uint64 indexed challengeIndex)
func (_OldChallengeManager *OldChallengeManagerFilterer) FilterOneStepProofCompleted(opts *bind.FilterOpts, challengeIndex []uint64) (*OldChallengeManagerOneStepProofCompletedIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _OldChallengeManager.contract.FilterLogs(opts, "OneStepProofCompleted", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &OldChallengeManagerOneStepProofCompletedIterator{contract: _OldChallengeManager.contract, event: "OneStepProofCompleted", logs: logs, sub: sub}, nil
}

// WatchOneStepProofCompleted is a free log subscription operation binding the contract event 0xc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e.
//
// Solidity: event OneStepProofCompleted(uint64 indexed challengeIndex)
func (_OldChallengeManager *OldChallengeManagerFilterer) WatchOneStepProofCompleted(opts *bind.WatchOpts, sink chan<- *OldChallengeManagerOneStepProofCompleted, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _OldChallengeManager.contract.WatchLogs(opts, "OneStepProofCompleted", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OldChallengeManagerOneStepProofCompleted)
				if err := _OldChallengeManager.contract.UnpackLog(event, "OneStepProofCompleted", log); err != nil {
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

// ParseOneStepProofCompleted is a log parse operation binding the contract event 0xc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e.
//
// Solidity: event OneStepProofCompleted(uint64 indexed challengeIndex)
func (_OldChallengeManager *OldChallengeManagerFilterer) ParseOneStepProofCompleted(log types.Log) (*OldChallengeManagerOneStepProofCompleted, error) {
	event := new(OldChallengeManagerOneStepProofCompleted)
	if err := _OldChallengeManager.contract.UnpackLog(event, "OneStepProofCompleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
