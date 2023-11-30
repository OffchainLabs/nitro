// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package hotshot

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

// BN254G1Point is an auto generated low-level Go binding around an user-defined struct.
type BN254G1Point struct {
	X *big.Int
	Y *big.Int
}

// BN254G2Point is an auto generated low-level Go binding around an user-defined struct.
type BN254G2Point struct {
	X0 *big.Int
	X1 *big.Int
	Y0 *big.Int
	Y1 *big.Int
}

// HotShotQC is an auto generated low-level Go binding around an user-defined struct.
type HotShotQC struct {
	Height          *big.Int
	BlockCommitment *big.Int
	Pad1            *big.Int
	Pad2            *big.Int
}

// HotshotMetaData contains all meta data concerning the Hotshot contract.
var HotshotMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"BLSSigVerificationFailed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expectedBlockNumber\",\"type\":\"uint256\"}],\"name\":\"IncorrectBlockNumber\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"InvalidQC\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NoKeySelected\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotEnoughStake\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"numBlocks\",\"type\":\"uint256\"}],\"name\":\"TooManyBlocks\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"firstBlockNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"numBlocks\",\"type\":\"uint256\"}],\"name\":\"NewBlocks\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"x0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"x1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"y0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"y1\",\"type\":\"uint256\"}],\"indexed\":false,\"internalType\":\"structBN254.G2Point\",\"name\":\"stakingKey\",\"type\":\"tuple\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"NewStakingKey\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MAX_BLOCKS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"x0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"x1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"y0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"y1\",\"type\":\"uint256\"}],\"internalType\":\"structBN254.G2Point\",\"name\":\"stakingKey\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"addNewStakingKey\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"blockHeight\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockHeight\",\"type\":\"uint256\"}],\"name\":\"commitments\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"commitment\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getStakingKey\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"x0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"x1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"y0\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"y1\",\"type\":\"uint256\"}],\"internalType\":\"structBN254.G2Point\",\"name\":\"\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blockCommitment\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"pad1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"pad2\",\"type\":\"uint256\"}],\"internalType\":\"structHotShot.QC[]\",\"name\":\"qcs\",\"type\":\"tuple[]\"}],\"name\":\"newBlocks\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"y\",\"type\":\"uint256\"}],\"internalType\":\"structBN254.G1Point\",\"name\":\"sig\",\"type\":\"tuple\"},{\"internalType\":\"bool[]\",\"name\":\"bitmap\",\"type\":\"bool[]\"},{\"internalType\":\"uint256\",\"name\":\"minStakeThreshold\",\"type\":\"uint256\"}],\"name\":\"verifyAggSig\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// HotshotABI is the input ABI used to generate the binding from.
// Deprecated: Use HotshotMetaData.ABI instead.
var HotshotABI = HotshotMetaData.ABI

// Hotshot is an auto generated Go binding around an Ethereum contract.
type Hotshot struct {
	HotshotCaller     // Read-only binding to the contract
	HotshotTransactor // Write-only binding to the contract
	HotshotFilterer   // Log filterer for contract events
}

// HotshotCaller is an auto generated read-only Go binding around an Ethereum contract.
type HotshotCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HotshotTransactor is an auto generated write-only Go binding around an Ethereum contract.
type HotshotTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HotshotFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type HotshotFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HotshotSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type HotshotSession struct {
	Contract     *Hotshot          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// HotshotCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type HotshotCallerSession struct {
	Contract *HotshotCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// HotshotTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type HotshotTransactorSession struct {
	Contract     *HotshotTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// HotshotRaw is an auto generated low-level Go binding around an Ethereum contract.
type HotshotRaw struct {
	Contract *Hotshot // Generic contract binding to access the raw methods on
}

// HotshotCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type HotshotCallerRaw struct {
	Contract *HotshotCaller // Generic read-only contract binding to access the raw methods on
}

// HotshotTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type HotshotTransactorRaw struct {
	Contract *HotshotTransactor // Generic write-only contract binding to access the raw methods on
}

// NewHotshot creates a new instance of Hotshot, bound to a specific deployed contract.
func NewHotshot(address common.Address, backend bind.ContractBackend) (*Hotshot, error) {
	contract, err := bindHotshot(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Hotshot{HotshotCaller: HotshotCaller{contract: contract}, HotshotTransactor: HotshotTransactor{contract: contract}, HotshotFilterer: HotshotFilterer{contract: contract}}, nil
}

// NewHotshotCaller creates a new read-only instance of Hotshot, bound to a specific deployed contract.
func NewHotshotCaller(address common.Address, caller bind.ContractCaller) (*HotshotCaller, error) {
	contract, err := bindHotshot(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &HotshotCaller{contract: contract}, nil
}

// NewHotshotTransactor creates a new write-only instance of Hotshot, bound to a specific deployed contract.
func NewHotshotTransactor(address common.Address, transactor bind.ContractTransactor) (*HotshotTransactor, error) {
	contract, err := bindHotshot(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &HotshotTransactor{contract: contract}, nil
}

// NewHotshotFilterer creates a new log filterer instance of Hotshot, bound to a specific deployed contract.
func NewHotshotFilterer(address common.Address, filterer bind.ContractFilterer) (*HotshotFilterer, error) {
	contract, err := bindHotshot(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &HotshotFilterer{contract: contract}, nil
}

// bindHotshot binds a generic wrapper to an already deployed contract.
func bindHotshot(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := HotshotMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Hotshot *HotshotRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Hotshot.Contract.HotshotCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Hotshot *HotshotRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Hotshot.Contract.HotshotTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Hotshot *HotshotRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Hotshot.Contract.HotshotTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Hotshot *HotshotCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Hotshot.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Hotshot *HotshotTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Hotshot.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Hotshot *HotshotTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Hotshot.Contract.contract.Transact(opts, method, params...)
}

// MAXBLOCKS is a free data retrieval call binding the contract method 0x26833dcc.
//
// Solidity: function MAX_BLOCKS() view returns(uint256)
func (_Hotshot *HotshotCaller) MAXBLOCKS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Hotshot.contract.Call(opts, &out, "MAX_BLOCKS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXBLOCKS is a free data retrieval call binding the contract method 0x26833dcc.
//
// Solidity: function MAX_BLOCKS() view returns(uint256)
func (_Hotshot *HotshotSession) MAXBLOCKS() (*big.Int, error) {
	return _Hotshot.Contract.MAXBLOCKS(&_Hotshot.CallOpts)
}

// MAXBLOCKS is a free data retrieval call binding the contract method 0x26833dcc.
//
// Solidity: function MAX_BLOCKS() view returns(uint256)
func (_Hotshot *HotshotCallerSession) MAXBLOCKS() (*big.Int, error) {
	return _Hotshot.Contract.MAXBLOCKS(&_Hotshot.CallOpts)
}

// BlockHeight is a free data retrieval call binding the contract method 0xf44ff712.
//
// Solidity: function blockHeight() view returns(uint256)
func (_Hotshot *HotshotCaller) BlockHeight(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Hotshot.contract.Call(opts, &out, "blockHeight")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BlockHeight is a free data retrieval call binding the contract method 0xf44ff712.
//
// Solidity: function blockHeight() view returns(uint256)
func (_Hotshot *HotshotSession) BlockHeight() (*big.Int, error) {
	return _Hotshot.Contract.BlockHeight(&_Hotshot.CallOpts)
}

// BlockHeight is a free data retrieval call binding the contract method 0xf44ff712.
//
// Solidity: function blockHeight() view returns(uint256)
func (_Hotshot *HotshotCallerSession) BlockHeight() (*big.Int, error) {
	return _Hotshot.Contract.BlockHeight(&_Hotshot.CallOpts)
}

// Commitments is a free data retrieval call binding the contract method 0x49ce8997.
//
// Solidity: function commitments(uint256 blockHeight) view returns(uint256 commitment)
func (_Hotshot *HotshotCaller) Commitments(opts *bind.CallOpts, blockHeight *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Hotshot.contract.Call(opts, &out, "commitments", blockHeight)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Commitments is a free data retrieval call binding the contract method 0x49ce8997.
//
// Solidity: function commitments(uint256 blockHeight) view returns(uint256 commitment)
func (_Hotshot *HotshotSession) Commitments(blockHeight *big.Int) (*big.Int, error) {
	return _Hotshot.Contract.Commitments(&_Hotshot.CallOpts, blockHeight)
}

// Commitments is a free data retrieval call binding the contract method 0x49ce8997.
//
// Solidity: function commitments(uint256 blockHeight) view returns(uint256 commitment)
func (_Hotshot *HotshotCallerSession) Commitments(blockHeight *big.Int) (*big.Int, error) {
	return _Hotshot.Contract.Commitments(&_Hotshot.CallOpts, blockHeight)
}

// GetStakingKey is a free data retrieval call binding the contract method 0x67a21e70.
//
// Solidity: function getStakingKey(uint256 index) view returns((uint256,uint256,uint256,uint256), uint256)
func (_Hotshot *HotshotCaller) GetStakingKey(opts *bind.CallOpts, index *big.Int) (BN254G2Point, *big.Int, error) {
	var out []interface{}
	err := _Hotshot.contract.Call(opts, &out, "getStakingKey", index)

	if err != nil {
		return *new(BN254G2Point), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(BN254G2Point)).(*BN254G2Point)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// GetStakingKey is a free data retrieval call binding the contract method 0x67a21e70.
//
// Solidity: function getStakingKey(uint256 index) view returns((uint256,uint256,uint256,uint256), uint256)
func (_Hotshot *HotshotSession) GetStakingKey(index *big.Int) (BN254G2Point, *big.Int, error) {
	return _Hotshot.Contract.GetStakingKey(&_Hotshot.CallOpts, index)
}

// GetStakingKey is a free data retrieval call binding the contract method 0x67a21e70.
//
// Solidity: function getStakingKey(uint256 index) view returns((uint256,uint256,uint256,uint256), uint256)
func (_Hotshot *HotshotCallerSession) GetStakingKey(index *big.Int) (BN254G2Point, *big.Int, error) {
	return _Hotshot.Contract.GetStakingKey(&_Hotshot.CallOpts, index)
}

// VerifyAggSig is a free data retrieval call binding the contract method 0x0340961e.
//
// Solidity: function verifyAggSig(bytes message, (uint256,uint256) sig, bool[] bitmap, uint256 minStakeThreshold) view returns()
func (_Hotshot *HotshotCaller) VerifyAggSig(opts *bind.CallOpts, message []byte, sig BN254G1Point, bitmap []bool, minStakeThreshold *big.Int) error {
	var out []interface{}
	err := _Hotshot.contract.Call(opts, &out, "verifyAggSig", message, sig, bitmap, minStakeThreshold)

	if err != nil {
		return err
	}

	return err

}

// VerifyAggSig is a free data retrieval call binding the contract method 0x0340961e.
//
// Solidity: function verifyAggSig(bytes message, (uint256,uint256) sig, bool[] bitmap, uint256 minStakeThreshold) view returns()
func (_Hotshot *HotshotSession) VerifyAggSig(message []byte, sig BN254G1Point, bitmap []bool, minStakeThreshold *big.Int) error {
	return _Hotshot.Contract.VerifyAggSig(&_Hotshot.CallOpts, message, sig, bitmap, minStakeThreshold)
}

// VerifyAggSig is a free data retrieval call binding the contract method 0x0340961e.
//
// Solidity: function verifyAggSig(bytes message, (uint256,uint256) sig, bool[] bitmap, uint256 minStakeThreshold) view returns()
func (_Hotshot *HotshotCallerSession) VerifyAggSig(message []byte, sig BN254G1Point, bitmap []bool, minStakeThreshold *big.Int) error {
	return _Hotshot.Contract.VerifyAggSig(&_Hotshot.CallOpts, message, sig, bitmap, minStakeThreshold)
}

// AddNewStakingKey is a paid mutator transaction binding the contract method 0xf1f45d99.
//
// Solidity: function addNewStakingKey((uint256,uint256,uint256,uint256) stakingKey, uint256 amount) returns()
func (_Hotshot *HotshotTransactor) AddNewStakingKey(opts *bind.TransactOpts, stakingKey BN254G2Point, amount *big.Int) (*types.Transaction, error) {
	return _Hotshot.contract.Transact(opts, "addNewStakingKey", stakingKey, amount)
}

// AddNewStakingKey is a paid mutator transaction binding the contract method 0xf1f45d99.
//
// Solidity: function addNewStakingKey((uint256,uint256,uint256,uint256) stakingKey, uint256 amount) returns()
func (_Hotshot *HotshotSession) AddNewStakingKey(stakingKey BN254G2Point, amount *big.Int) (*types.Transaction, error) {
	return _Hotshot.Contract.AddNewStakingKey(&_Hotshot.TransactOpts, stakingKey, amount)
}

// AddNewStakingKey is a paid mutator transaction binding the contract method 0xf1f45d99.
//
// Solidity: function addNewStakingKey((uint256,uint256,uint256,uint256) stakingKey, uint256 amount) returns()
func (_Hotshot *HotshotTransactorSession) AddNewStakingKey(stakingKey BN254G2Point, amount *big.Int) (*types.Transaction, error) {
	return _Hotshot.Contract.AddNewStakingKey(&_Hotshot.TransactOpts, stakingKey, amount)
}

// NewBlocks is a paid mutator transaction binding the contract method 0x0a321cff.
//
// Solidity: function newBlocks((uint256,uint256,uint256,uint256)[] qcs) returns()
func (_Hotshot *HotshotTransactor) NewBlocks(opts *bind.TransactOpts, qcs []HotShotQC) (*types.Transaction, error) {
	return _Hotshot.contract.Transact(opts, "newBlocks", qcs)
}

// NewBlocks is a paid mutator transaction binding the contract method 0x0a321cff.
//
// Solidity: function newBlocks((uint256,uint256,uint256,uint256)[] qcs) returns()
func (_Hotshot *HotshotSession) NewBlocks(qcs []HotShotQC) (*types.Transaction, error) {
	return _Hotshot.Contract.NewBlocks(&_Hotshot.TransactOpts, qcs)
}

// NewBlocks is a paid mutator transaction binding the contract method 0x0a321cff.
//
// Solidity: function newBlocks((uint256,uint256,uint256,uint256)[] qcs) returns()
func (_Hotshot *HotshotTransactorSession) NewBlocks(qcs []HotShotQC) (*types.Transaction, error) {
	return _Hotshot.Contract.NewBlocks(&_Hotshot.TransactOpts, qcs)
}

// HotshotNewBlocksIterator is returned from FilterNewBlocks and is used to iterate over the raw logs and unpacked data for NewBlocks events raised by the Hotshot contract.
type HotshotNewBlocksIterator struct {
	Event *HotshotNewBlocks // Event containing the contract specifics and raw log

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
func (it *HotshotNewBlocksIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HotshotNewBlocks)
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
		it.Event = new(HotshotNewBlocks)
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
func (it *HotshotNewBlocksIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HotshotNewBlocksIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HotshotNewBlocks represents a NewBlocks event raised by the Hotshot contract.
type HotshotNewBlocks struct {
	FirstBlockNumber *big.Int
	NumBlocks        *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterNewBlocks is a free log retrieval operation binding the contract event 0x8203a21e4f95f72e5081d5e0929b1a8c52141e123f9a14e1e74b0260fa5f52f1.
//
// Solidity: event NewBlocks(uint256 firstBlockNumber, uint256 numBlocks)
func (_Hotshot *HotshotFilterer) FilterNewBlocks(opts *bind.FilterOpts) (*HotshotNewBlocksIterator, error) {

	logs, sub, err := _Hotshot.contract.FilterLogs(opts, "NewBlocks")
	if err != nil {
		return nil, err
	}
	return &HotshotNewBlocksIterator{contract: _Hotshot.contract, event: "NewBlocks", logs: logs, sub: sub}, nil
}

// WatchNewBlocks is a free log subscription operation binding the contract event 0x8203a21e4f95f72e5081d5e0929b1a8c52141e123f9a14e1e74b0260fa5f52f1.
//
// Solidity: event NewBlocks(uint256 firstBlockNumber, uint256 numBlocks)
func (_Hotshot *HotshotFilterer) WatchNewBlocks(opts *bind.WatchOpts, sink chan<- *HotshotNewBlocks) (event.Subscription, error) {

	logs, sub, err := _Hotshot.contract.WatchLogs(opts, "NewBlocks")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HotshotNewBlocks)
				if err := _Hotshot.contract.UnpackLog(event, "NewBlocks", log); err != nil {
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

// ParseNewBlocks is a log parse operation binding the contract event 0x8203a21e4f95f72e5081d5e0929b1a8c52141e123f9a14e1e74b0260fa5f52f1.
//
// Solidity: event NewBlocks(uint256 firstBlockNumber, uint256 numBlocks)
func (_Hotshot *HotshotFilterer) ParseNewBlocks(log types.Log) (*HotshotNewBlocks, error) {
	event := new(HotshotNewBlocks)
	if err := _Hotshot.contract.UnpackLog(event, "NewBlocks", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HotshotNewStakingKeyIterator is returned from FilterNewStakingKey and is used to iterate over the raw logs and unpacked data for NewStakingKey events raised by the Hotshot contract.
type HotshotNewStakingKeyIterator struct {
	Event *HotshotNewStakingKey // Event containing the contract specifics and raw log

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
func (it *HotshotNewStakingKeyIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HotshotNewStakingKey)
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
		it.Event = new(HotshotNewStakingKey)
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
func (it *HotshotNewStakingKeyIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HotshotNewStakingKeyIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HotshotNewStakingKey represents a NewStakingKey event raised by the Hotshot contract.
type HotshotNewStakingKey struct {
	StakingKey BN254G2Point
	Amount     *big.Int
	Index      *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterNewStakingKey is a free log retrieval operation binding the contract event 0xd72fe1ac57d3e6d51c922ae4d811cc50aa3ad7026283aea637494a073252565a.
//
// Solidity: event NewStakingKey((uint256,uint256,uint256,uint256) stakingKey, uint256 amount, uint256 index)
func (_Hotshot *HotshotFilterer) FilterNewStakingKey(opts *bind.FilterOpts) (*HotshotNewStakingKeyIterator, error) {

	logs, sub, err := _Hotshot.contract.FilterLogs(opts, "NewStakingKey")
	if err != nil {
		return nil, err
	}
	return &HotshotNewStakingKeyIterator{contract: _Hotshot.contract, event: "NewStakingKey", logs: logs, sub: sub}, nil
}

// WatchNewStakingKey is a free log subscription operation binding the contract event 0xd72fe1ac57d3e6d51c922ae4d811cc50aa3ad7026283aea637494a073252565a.
//
// Solidity: event NewStakingKey((uint256,uint256,uint256,uint256) stakingKey, uint256 amount, uint256 index)
func (_Hotshot *HotshotFilterer) WatchNewStakingKey(opts *bind.WatchOpts, sink chan<- *HotshotNewStakingKey) (event.Subscription, error) {

	logs, sub, err := _Hotshot.contract.WatchLogs(opts, "NewStakingKey")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HotshotNewStakingKey)
				if err := _Hotshot.contract.UnpackLog(event, "NewStakingKey", log); err != nil {
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

// ParseNewStakingKey is a log parse operation binding the contract event 0xd72fe1ac57d3e6d51c922ae4d811cc50aa3ad7026283aea637494a073252565a.
//
// Solidity: event NewStakingKey((uint256,uint256,uint256,uint256) stakingKey, uint256 amount, uint256 index)
func (_Hotshot *HotshotFilterer) ParseNewStakingKey(log types.Log) (*HotshotNewStakingKey, error) {
	event := new(HotshotNewStakingKey)
	if err := _Hotshot.contract.UnpackLog(event, "NewStakingKey", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
