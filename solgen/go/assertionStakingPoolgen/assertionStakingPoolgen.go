// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package assertionStakingPoolgen

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

// AssertionInputs is an auto generated low-level Go binding around an user-defined struct.
type AssertionInputs struct {
	BeforeStateData BeforeStateData
	BeforeState     AssertionState
	AfterState      AssertionState
}

// AssertionState is an auto generated low-level Go binding around an user-defined struct.
type AssertionState struct {
	GlobalState    GlobalState
	MachineStatus  uint8
	EndHistoryRoot [32]byte
}

// BeforeStateData is an auto generated low-level Go binding around an user-defined struct.
type BeforeStateData struct {
	PrevPrevAssertionHash [32]byte
	SequencerBatchAcc     [32]byte
	ConfigData            ConfigData
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
	Level          uint8
	EndHistoryRoot [32]byte
	EndHeight      *big.Int
	ClaimId        [32]byte
	PrefixProof    []byte
	Proof          []byte
}

// GlobalState is an auto generated low-level Go binding around an user-defined struct.
type GlobalState struct {
	Bytes32Vals [2][32]byte
	U64Vals     [2]uint64
}

// AbsBoldStakingPoolMetaData contains all meta data concerning the AbsBoldStakingPool contract.
var AbsBoldStakingPoolMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"name\":\"AmountExceedsBalance\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAmount\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"StakeDeposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"StakeWithdrawn\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"depositBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"depositIntoPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stakeToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawFromPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawFromPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// AbsBoldStakingPoolABI is the input ABI used to generate the binding from.
// Deprecated: Use AbsBoldStakingPoolMetaData.ABI instead.
var AbsBoldStakingPoolABI = AbsBoldStakingPoolMetaData.ABI

// AbsBoldStakingPool is an auto generated Go binding around an Ethereum contract.
type AbsBoldStakingPool struct {
	AbsBoldStakingPoolCaller     // Read-only binding to the contract
	AbsBoldStakingPoolTransactor // Write-only binding to the contract
	AbsBoldStakingPoolFilterer   // Log filterer for contract events
}

// AbsBoldStakingPoolCaller is an auto generated read-only Go binding around an Ethereum contract.
type AbsBoldStakingPoolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbsBoldStakingPoolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AbsBoldStakingPoolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbsBoldStakingPoolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AbsBoldStakingPoolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AbsBoldStakingPoolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AbsBoldStakingPoolSession struct {
	Contract     *AbsBoldStakingPool // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// AbsBoldStakingPoolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AbsBoldStakingPoolCallerSession struct {
	Contract *AbsBoldStakingPoolCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// AbsBoldStakingPoolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AbsBoldStakingPoolTransactorSession struct {
	Contract     *AbsBoldStakingPoolTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// AbsBoldStakingPoolRaw is an auto generated low-level Go binding around an Ethereum contract.
type AbsBoldStakingPoolRaw struct {
	Contract *AbsBoldStakingPool // Generic contract binding to access the raw methods on
}

// AbsBoldStakingPoolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AbsBoldStakingPoolCallerRaw struct {
	Contract *AbsBoldStakingPoolCaller // Generic read-only contract binding to access the raw methods on
}

// AbsBoldStakingPoolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AbsBoldStakingPoolTransactorRaw struct {
	Contract *AbsBoldStakingPoolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAbsBoldStakingPool creates a new instance of AbsBoldStakingPool, bound to a specific deployed contract.
func NewAbsBoldStakingPool(address common.Address, backend bind.ContractBackend) (*AbsBoldStakingPool, error) {
	contract, err := bindAbsBoldStakingPool(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AbsBoldStakingPool{AbsBoldStakingPoolCaller: AbsBoldStakingPoolCaller{contract: contract}, AbsBoldStakingPoolTransactor: AbsBoldStakingPoolTransactor{contract: contract}, AbsBoldStakingPoolFilterer: AbsBoldStakingPoolFilterer{contract: contract}}, nil
}

// NewAbsBoldStakingPoolCaller creates a new read-only instance of AbsBoldStakingPool, bound to a specific deployed contract.
func NewAbsBoldStakingPoolCaller(address common.Address, caller bind.ContractCaller) (*AbsBoldStakingPoolCaller, error) {
	contract, err := bindAbsBoldStakingPool(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AbsBoldStakingPoolCaller{contract: contract}, nil
}

// NewAbsBoldStakingPoolTransactor creates a new write-only instance of AbsBoldStakingPool, bound to a specific deployed contract.
func NewAbsBoldStakingPoolTransactor(address common.Address, transactor bind.ContractTransactor) (*AbsBoldStakingPoolTransactor, error) {
	contract, err := bindAbsBoldStakingPool(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AbsBoldStakingPoolTransactor{contract: contract}, nil
}

// NewAbsBoldStakingPoolFilterer creates a new log filterer instance of AbsBoldStakingPool, bound to a specific deployed contract.
func NewAbsBoldStakingPoolFilterer(address common.Address, filterer bind.ContractFilterer) (*AbsBoldStakingPoolFilterer, error) {
	contract, err := bindAbsBoldStakingPool(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AbsBoldStakingPoolFilterer{contract: contract}, nil
}

// bindAbsBoldStakingPool binds a generic wrapper to an already deployed contract.
func bindAbsBoldStakingPool(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AbsBoldStakingPoolMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AbsBoldStakingPool *AbsBoldStakingPoolRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AbsBoldStakingPool.Contract.AbsBoldStakingPoolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AbsBoldStakingPool *AbsBoldStakingPoolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AbsBoldStakingPool.Contract.AbsBoldStakingPoolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AbsBoldStakingPool *AbsBoldStakingPoolRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AbsBoldStakingPool.Contract.AbsBoldStakingPoolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AbsBoldStakingPool *AbsBoldStakingPoolCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AbsBoldStakingPool.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AbsBoldStakingPool *AbsBoldStakingPoolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AbsBoldStakingPool.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AbsBoldStakingPool *AbsBoldStakingPoolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AbsBoldStakingPool.Contract.contract.Transact(opts, method, params...)
}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_AbsBoldStakingPool *AbsBoldStakingPoolCaller) DepositBalance(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _AbsBoldStakingPool.contract.Call(opts, &out, "depositBalance", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_AbsBoldStakingPool *AbsBoldStakingPoolSession) DepositBalance(arg0 common.Address) (*big.Int, error) {
	return _AbsBoldStakingPool.Contract.DepositBalance(&_AbsBoldStakingPool.CallOpts, arg0)
}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_AbsBoldStakingPool *AbsBoldStakingPoolCallerSession) DepositBalance(arg0 common.Address) (*big.Int, error) {
	return _AbsBoldStakingPool.Contract.DepositBalance(&_AbsBoldStakingPool.CallOpts, arg0)
}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_AbsBoldStakingPool *AbsBoldStakingPoolCaller) StakeToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AbsBoldStakingPool.contract.Call(opts, &out, "stakeToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_AbsBoldStakingPool *AbsBoldStakingPoolSession) StakeToken() (common.Address, error) {
	return _AbsBoldStakingPool.Contract.StakeToken(&_AbsBoldStakingPool.CallOpts)
}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_AbsBoldStakingPool *AbsBoldStakingPoolCallerSession) StakeToken() (common.Address, error) {
	return _AbsBoldStakingPool.Contract.StakeToken(&_AbsBoldStakingPool.CallOpts)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 amount) returns()
func (_AbsBoldStakingPool *AbsBoldStakingPoolTransactor) DepositIntoPool(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _AbsBoldStakingPool.contract.Transact(opts, "depositIntoPool", amount)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 amount) returns()
func (_AbsBoldStakingPool *AbsBoldStakingPoolSession) DepositIntoPool(amount *big.Int) (*types.Transaction, error) {
	return _AbsBoldStakingPool.Contract.DepositIntoPool(&_AbsBoldStakingPool.TransactOpts, amount)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 amount) returns()
func (_AbsBoldStakingPool *AbsBoldStakingPoolTransactorSession) DepositIntoPool(amount *big.Int) (*types.Transaction, error) {
	return _AbsBoldStakingPool.Contract.DepositIntoPool(&_AbsBoldStakingPool.TransactOpts, amount)
}

// WithdrawFromPool26c0e5c5 is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_AbsBoldStakingPool *AbsBoldStakingPoolTransactor) WithdrawFromPool26c0e5c5(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AbsBoldStakingPool.contract.Transact(opts, "withdrawFromPool")
}

// WithdrawFromPool26c0e5c5 is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_AbsBoldStakingPool *AbsBoldStakingPoolSession) WithdrawFromPool26c0e5c5() (*types.Transaction, error) {
	return _AbsBoldStakingPool.Contract.WithdrawFromPool26c0e5c5(&_AbsBoldStakingPool.TransactOpts)
}

// WithdrawFromPool26c0e5c5 is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_AbsBoldStakingPool *AbsBoldStakingPoolTransactorSession) WithdrawFromPool26c0e5c5() (*types.Transaction, error) {
	return _AbsBoldStakingPool.Contract.WithdrawFromPool26c0e5c5(&_AbsBoldStakingPool.TransactOpts)
}

// WithdrawFromPool30fc43ed is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 amount) returns()
func (_AbsBoldStakingPool *AbsBoldStakingPoolTransactor) WithdrawFromPool30fc43ed(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _AbsBoldStakingPool.contract.Transact(opts, "withdrawFromPool0", amount)
}

// WithdrawFromPool30fc43ed is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 amount) returns()
func (_AbsBoldStakingPool *AbsBoldStakingPoolSession) WithdrawFromPool30fc43ed(amount *big.Int) (*types.Transaction, error) {
	return _AbsBoldStakingPool.Contract.WithdrawFromPool30fc43ed(&_AbsBoldStakingPool.TransactOpts, amount)
}

// WithdrawFromPool30fc43ed is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 amount) returns()
func (_AbsBoldStakingPool *AbsBoldStakingPoolTransactorSession) WithdrawFromPool30fc43ed(amount *big.Int) (*types.Transaction, error) {
	return _AbsBoldStakingPool.Contract.WithdrawFromPool30fc43ed(&_AbsBoldStakingPool.TransactOpts, amount)
}

// AbsBoldStakingPoolStakeDepositedIterator is returned from FilterStakeDeposited and is used to iterate over the raw logs and unpacked data for StakeDeposited events raised by the AbsBoldStakingPool contract.
type AbsBoldStakingPoolStakeDepositedIterator struct {
	Event *AbsBoldStakingPoolStakeDeposited // Event containing the contract specifics and raw log

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
func (it *AbsBoldStakingPoolStakeDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbsBoldStakingPoolStakeDeposited)
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
		it.Event = new(AbsBoldStakingPoolStakeDeposited)
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
func (it *AbsBoldStakingPoolStakeDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbsBoldStakingPoolStakeDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbsBoldStakingPoolStakeDeposited represents a StakeDeposited event raised by the AbsBoldStakingPool contract.
type AbsBoldStakingPoolStakeDeposited struct {
	Sender common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStakeDeposited is a free log retrieval operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_AbsBoldStakingPool *AbsBoldStakingPoolFilterer) FilterStakeDeposited(opts *bind.FilterOpts, sender []common.Address) (*AbsBoldStakingPoolStakeDepositedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AbsBoldStakingPool.contract.FilterLogs(opts, "StakeDeposited", senderRule)
	if err != nil {
		return nil, err
	}
	return &AbsBoldStakingPoolStakeDepositedIterator{contract: _AbsBoldStakingPool.contract, event: "StakeDeposited", logs: logs, sub: sub}, nil
}

// WatchStakeDeposited is a free log subscription operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_AbsBoldStakingPool *AbsBoldStakingPoolFilterer) WatchStakeDeposited(opts *bind.WatchOpts, sink chan<- *AbsBoldStakingPoolStakeDeposited, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AbsBoldStakingPool.contract.WatchLogs(opts, "StakeDeposited", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbsBoldStakingPoolStakeDeposited)
				if err := _AbsBoldStakingPool.contract.UnpackLog(event, "StakeDeposited", log); err != nil {
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

// ParseStakeDeposited is a log parse operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_AbsBoldStakingPool *AbsBoldStakingPoolFilterer) ParseStakeDeposited(log types.Log) (*AbsBoldStakingPoolStakeDeposited, error) {
	event := new(AbsBoldStakingPoolStakeDeposited)
	if err := _AbsBoldStakingPool.contract.UnpackLog(event, "StakeDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AbsBoldStakingPoolStakeWithdrawnIterator is returned from FilterStakeWithdrawn and is used to iterate over the raw logs and unpacked data for StakeWithdrawn events raised by the AbsBoldStakingPool contract.
type AbsBoldStakingPoolStakeWithdrawnIterator struct {
	Event *AbsBoldStakingPoolStakeWithdrawn // Event containing the contract specifics and raw log

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
func (it *AbsBoldStakingPoolStakeWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AbsBoldStakingPoolStakeWithdrawn)
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
		it.Event = new(AbsBoldStakingPoolStakeWithdrawn)
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
func (it *AbsBoldStakingPoolStakeWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AbsBoldStakingPoolStakeWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AbsBoldStakingPoolStakeWithdrawn represents a StakeWithdrawn event raised by the AbsBoldStakingPool contract.
type AbsBoldStakingPoolStakeWithdrawn struct {
	Sender common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStakeWithdrawn is a free log retrieval operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_AbsBoldStakingPool *AbsBoldStakingPoolFilterer) FilterStakeWithdrawn(opts *bind.FilterOpts, sender []common.Address) (*AbsBoldStakingPoolStakeWithdrawnIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AbsBoldStakingPool.contract.FilterLogs(opts, "StakeWithdrawn", senderRule)
	if err != nil {
		return nil, err
	}
	return &AbsBoldStakingPoolStakeWithdrawnIterator{contract: _AbsBoldStakingPool.contract, event: "StakeWithdrawn", logs: logs, sub: sub}, nil
}

// WatchStakeWithdrawn is a free log subscription operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_AbsBoldStakingPool *AbsBoldStakingPoolFilterer) WatchStakeWithdrawn(opts *bind.WatchOpts, sink chan<- *AbsBoldStakingPoolStakeWithdrawn, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AbsBoldStakingPool.contract.WatchLogs(opts, "StakeWithdrawn", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbsBoldStakingPoolStakeWithdrawn)
				if err := _AbsBoldStakingPool.contract.UnpackLog(event, "StakeWithdrawn", log); err != nil {
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

// ParseStakeWithdrawn is a log parse operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_AbsBoldStakingPool *AbsBoldStakingPoolFilterer) ParseStakeWithdrawn(log types.Log) (*AbsBoldStakingPoolStakeWithdrawn, error) {
	event := new(AbsBoldStakingPoolStakeWithdrawn)
	if err := _AbsBoldStakingPool.contract.UnpackLog(event, "StakeWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AssertionStakingPoolMetaData contains all meta data concerning the AssertionStakingPool contract.
var AssertionStakingPoolMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollup\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"_assertionHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"name\":\"AmountExceedsBalance\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAmount\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"StakeDeposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"StakeWithdrawn\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"assertionHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"prevPrevAssertionHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"sequencerBatchAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"requiredStake\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"confirmPeriodBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nextInboxPosition\",\"type\":\"uint64\"}],\"internalType\":\"structConfigData\",\"name\":\"configData\",\"type\":\"tuple\"}],\"internalType\":\"structBeforeStateData\",\"name\":\"beforeStateData\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"beforeState\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"afterState\",\"type\":\"tuple\"}],\"internalType\":\"structAssertionInputs\",\"name\":\"assertionInputs\",\"type\":\"tuple\"}],\"name\":\"createAssertion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"depositBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"depositIntoPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"makeStakeWithdrawable\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"makeStakeWithdrawableAndWithdrawBackIntoPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stakeToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawFromPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawFromPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawStakeBackIntoPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60e060405234801561001057600080fd5b50604051610ee6380380610ee683398101604081905261002f916100ca565b816001600160a01b03166351ed6a306040518163ffffffff1660e01b8152600401602060405180830381865afa15801561006d573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061009191906100f6565b6001600160a01b039081166080529190911660a05260c052610118565b80516001600160a01b03811681146100c557600080fd5b919050565b600080604083850312156100dd57600080fd5b6100e6836100ae565b9150602083015190509250929050565b60006020828403121561010857600080fd5b610111826100ae565b9392505050565b60805160a05160c051610d6861017e6000396000818160d301526104be0152600081816101c7015281816104380152818161048d01528181610520015261059501526000818161012a015281816102c0015281816103a401526104160152610d686000f3fe608060405234801561001057600080fd5b50600436106100c95760003560e01c80637476083b116100815780639451944d1161005b5780639451944d1461019a578063956501bb146101a2578063cb23bcb5146101c257600080fd5b80637476083b1461016c578063839159711461017f578063930412af1461019257600080fd5b806330fc43ed116100b257806330fc43ed1461011257806351ed6a30146101255780636b74d5151461016457600080fd5b80632113ed21146100ce57806326c0e5c514610108575b600080fd5b6100f57f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020015b60405180910390f35b6101106101e9565b005b610110610120366004610a7c565b610204565b61014c7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b0390911681526020016100ff565b610110610328565b61011061017a366004610a7c565b610338565b61011061018d366004610a95565b610404565b61011061051e565b610110610593565b6100f56101b0366004610aca565b60006020819052908152604090205481565b61014c7f000000000000000000000000000000000000000000000000000000000000000081565b3360009081526020819052604090205461020290610204565b565b8060000361023e576040517f1f2a200500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b336000908152602081905260409020548082111561029d576040517fa47b7c6500000000000000000000000000000000000000000000000000000000815233600482015260248101839052604481018290526064015b60405180910390fd5b6102a78282610b14565b336000818152602081905260409020919091556102ef907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316908461061a565b60405182815233907f8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc9060200160405180910390a25050565b61033061051e565b610202610593565b80600003610372576040517f1f2a200500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526020819052604081208054839290610391908490610b2d565b909155506103cc90506001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163330846106c8565b60405181815233907f0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc29060200160405180910390a250565b606081013561045d6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000167f000000000000000000000000000000000000000000000000000000000000000083610719565b6040517f50f32f680000000000000000000000000000000000000000000000000000000081526001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016906350f32f68906104e890849086907f0000000000000000000000000000000000000000000000000000000000000000903090600401610bbe565b600060405180830381600087803b15801561050257600080fd5b505af1158015610516573d6000803e3d6000fd5b505050505050565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03166357ef4ab96040518163ffffffff1660e01b8152600401600060405180830381600087803b15801561057957600080fd5b505af115801561058d573d6000803e3d6000fd5b50505050565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031663613739196040518163ffffffff1660e01b81526004016020604051808303816000875af11580156105f3573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906106179190610c84565b50565b6040516001600160a01b0383166024820152604481018290526106c39084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff00000000000000000000000000000000000000000000000000000000909316929092179091526107fd565b505050565b6040516001600160a01b038085166024830152831660448201526064810182905261058d9085907f23b872dd000000000000000000000000000000000000000000000000000000009060840161065f565b6040517fdd62ed3e0000000000000000000000000000000000000000000000000000000081523060048201526001600160a01b038381166024830152600091839186169063dd62ed3e90604401602060405180830381865afa158015610783573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107a79190610c84565b6107b19190610b2d565b6040516001600160a01b03851660248201526044810182905290915061058d9085907f095ea7b3000000000000000000000000000000000000000000000000000000009060640161065f565b6000610852826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564815250856001600160a01b03166108e29092919063ffffffff16565b8051909150156106c357808060200190518101906108709190610c9d565b6106c35760405162461bcd60e51b815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f742073756363656564000000000000000000000000000000000000000000006064820152608401610294565b60606108f184846000856108fb565b90505b9392505050565b6060824710156109735760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c00000000000000000000000000000000000000000000000000006064820152608401610294565b6001600160a01b0385163b6109ca5760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e74726163740000006044820152606401610294565b600080866001600160a01b031685876040516109e69190610ce3565b60006040518083038185875af1925050503d8060008114610a23576040519150601f19603f3d011682016040523d82523d6000602084013e610a28565b606091505b5091509150610a38828286610a43565b979650505050505050565b60608315610a525750816108f4565b825115610a625782518084602001fd5b8160405162461bcd60e51b81526004016102949190610cff565b600060208284031215610a8e57600080fd5b5035919050565b60006102608284031215610aa857600080fd5b50919050565b80356001600160a01b0381168114610ac557600080fd5b919050565b600060208284031215610adc57600080fd5b6108f482610aae565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b81810381811115610b2757610b27610ae5565b92915050565b80820180821115610b2757610b27610ae5565b803567ffffffffffffffff81168114610ac557600080fd5b6040818337604082016040820160005b6002811015610b995767ffffffffffffffff610b8383610b40565b1683526020928301929190910190600101610b68565b505050608081013560038110610bae57600080fd5b608083015260a090810135910152565b60006102c082019050858252843560208301526020850135604083015260408501356060830152606085013560808301526001600160a01b03610c0360808701610aae565b1660a0830152610c1560a08601610b40565b67ffffffffffffffff80821660c085015280610c3360c08901610b40565b1660e08501525050610c4c610100830160e08701610b58565b610c5e6101c083016101a08701610b58565b83610280830152610c7b6102a08301846001600160a01b03169052565b95945050505050565b600060208284031215610c9657600080fd5b5051919050565b600060208284031215610caf57600080fd5b815180151581146108f457600080fd5b60005b83811015610cda578181015183820152602001610cc2565b50506000910152565b60008251610cf5818460208701610cbf565b9190910192915050565b6020815260008251806020840152610d1e816040850160208701610cbf565b601f01601f1916919091016040019291505056fea26469706673582212207ff3737da395004ce78939e28894c41d6b5bd0d08c916ef4463a65833d328dd964736f6c63430008110033",
}

// AssertionStakingPoolABI is the input ABI used to generate the binding from.
// Deprecated: Use AssertionStakingPoolMetaData.ABI instead.
var AssertionStakingPoolABI = AssertionStakingPoolMetaData.ABI

// AssertionStakingPoolBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AssertionStakingPoolMetaData.Bin instead.
var AssertionStakingPoolBin = AssertionStakingPoolMetaData.Bin

// DeployAssertionStakingPool deploys a new Ethereum contract, binding an instance of AssertionStakingPool to it.
func DeployAssertionStakingPool(auth *bind.TransactOpts, backend bind.ContractBackend, _rollup common.Address, _assertionHash [32]byte) (common.Address, *types.Transaction, *AssertionStakingPool, error) {
	parsed, err := AssertionStakingPoolMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AssertionStakingPoolBin), backend, _rollup, _assertionHash)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AssertionStakingPool{AssertionStakingPoolCaller: AssertionStakingPoolCaller{contract: contract}, AssertionStakingPoolTransactor: AssertionStakingPoolTransactor{contract: contract}, AssertionStakingPoolFilterer: AssertionStakingPoolFilterer{contract: contract}}, nil
}

// AssertionStakingPool is an auto generated Go binding around an Ethereum contract.
type AssertionStakingPool struct {
	AssertionStakingPoolCaller     // Read-only binding to the contract
	AssertionStakingPoolTransactor // Write-only binding to the contract
	AssertionStakingPoolFilterer   // Log filterer for contract events
}

// AssertionStakingPoolCaller is an auto generated read-only Go binding around an Ethereum contract.
type AssertionStakingPoolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AssertionStakingPoolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AssertionStakingPoolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AssertionStakingPoolSession struct {
	Contract     *AssertionStakingPool // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// AssertionStakingPoolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AssertionStakingPoolCallerSession struct {
	Contract *AssertionStakingPoolCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// AssertionStakingPoolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AssertionStakingPoolTransactorSession struct {
	Contract     *AssertionStakingPoolTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// AssertionStakingPoolRaw is an auto generated low-level Go binding around an Ethereum contract.
type AssertionStakingPoolRaw struct {
	Contract *AssertionStakingPool // Generic contract binding to access the raw methods on
}

// AssertionStakingPoolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AssertionStakingPoolCallerRaw struct {
	Contract *AssertionStakingPoolCaller // Generic read-only contract binding to access the raw methods on
}

// AssertionStakingPoolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AssertionStakingPoolTransactorRaw struct {
	Contract *AssertionStakingPoolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAssertionStakingPool creates a new instance of AssertionStakingPool, bound to a specific deployed contract.
func NewAssertionStakingPool(address common.Address, backend bind.ContractBackend) (*AssertionStakingPool, error) {
	contract, err := bindAssertionStakingPool(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPool{AssertionStakingPoolCaller: AssertionStakingPoolCaller{contract: contract}, AssertionStakingPoolTransactor: AssertionStakingPoolTransactor{contract: contract}, AssertionStakingPoolFilterer: AssertionStakingPoolFilterer{contract: contract}}, nil
}

// NewAssertionStakingPoolCaller creates a new read-only instance of AssertionStakingPool, bound to a specific deployed contract.
func NewAssertionStakingPoolCaller(address common.Address, caller bind.ContractCaller) (*AssertionStakingPoolCaller, error) {
	contract, err := bindAssertionStakingPool(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCaller{contract: contract}, nil
}

// NewAssertionStakingPoolTransactor creates a new write-only instance of AssertionStakingPool, bound to a specific deployed contract.
func NewAssertionStakingPoolTransactor(address common.Address, transactor bind.ContractTransactor) (*AssertionStakingPoolTransactor, error) {
	contract, err := bindAssertionStakingPool(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolTransactor{contract: contract}, nil
}

// NewAssertionStakingPoolFilterer creates a new log filterer instance of AssertionStakingPool, bound to a specific deployed contract.
func NewAssertionStakingPoolFilterer(address common.Address, filterer bind.ContractFilterer) (*AssertionStakingPoolFilterer, error) {
	contract, err := bindAssertionStakingPool(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolFilterer{contract: contract}, nil
}

// bindAssertionStakingPool binds a generic wrapper to an already deployed contract.
func bindAssertionStakingPool(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AssertionStakingPoolMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AssertionStakingPool *AssertionStakingPoolRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AssertionStakingPool.Contract.AssertionStakingPoolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AssertionStakingPool *AssertionStakingPoolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.AssertionStakingPoolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AssertionStakingPool *AssertionStakingPoolRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.AssertionStakingPoolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AssertionStakingPool *AssertionStakingPoolCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AssertionStakingPool.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AssertionStakingPool *AssertionStakingPoolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AssertionStakingPool *AssertionStakingPoolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.contract.Transact(opts, method, params...)
}

// AssertionHash is a free data retrieval call binding the contract method 0x2113ed21.
//
// Solidity: function assertionHash() view returns(bytes32)
func (_AssertionStakingPool *AssertionStakingPoolCaller) AssertionHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _AssertionStakingPool.contract.Call(opts, &out, "assertionHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// AssertionHash is a free data retrieval call binding the contract method 0x2113ed21.
//
// Solidity: function assertionHash() view returns(bytes32)
func (_AssertionStakingPool *AssertionStakingPoolSession) AssertionHash() ([32]byte, error) {
	return _AssertionStakingPool.Contract.AssertionHash(&_AssertionStakingPool.CallOpts)
}

// AssertionHash is a free data retrieval call binding the contract method 0x2113ed21.
//
// Solidity: function assertionHash() view returns(bytes32)
func (_AssertionStakingPool *AssertionStakingPoolCallerSession) AssertionHash() ([32]byte, error) {
	return _AssertionStakingPool.Contract.AssertionHash(&_AssertionStakingPool.CallOpts)
}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_AssertionStakingPool *AssertionStakingPoolCaller) DepositBalance(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _AssertionStakingPool.contract.Call(opts, &out, "depositBalance", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_AssertionStakingPool *AssertionStakingPoolSession) DepositBalance(arg0 common.Address) (*big.Int, error) {
	return _AssertionStakingPool.Contract.DepositBalance(&_AssertionStakingPool.CallOpts, arg0)
}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_AssertionStakingPool *AssertionStakingPoolCallerSession) DepositBalance(arg0 common.Address) (*big.Int, error) {
	return _AssertionStakingPool.Contract.DepositBalance(&_AssertionStakingPool.CallOpts, arg0)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AssertionStakingPool.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolSession) Rollup() (common.Address, error) {
	return _AssertionStakingPool.Contract.Rollup(&_AssertionStakingPool.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolCallerSession) Rollup() (common.Address, error) {
	return _AssertionStakingPool.Contract.Rollup(&_AssertionStakingPool.CallOpts)
}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolCaller) StakeToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AssertionStakingPool.contract.Call(opts, &out, "stakeToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolSession) StakeToken() (common.Address, error) {
	return _AssertionStakingPool.Contract.StakeToken(&_AssertionStakingPool.CallOpts)
}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolCallerSession) StakeToken() (common.Address, error) {
	return _AssertionStakingPool.Contract.StakeToken(&_AssertionStakingPool.CallOpts)
}

// CreateAssertion is a paid mutator transaction binding the contract method 0x83915971.
//
// Solidity: function createAssertion(((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)),((bytes32[2],uint64[2]),uint8,bytes32),((bytes32[2],uint64[2]),uint8,bytes32)) assertionInputs) returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) CreateAssertion(opts *bind.TransactOpts, assertionInputs AssertionInputs) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "createAssertion", assertionInputs)
}

// CreateAssertion is a paid mutator transaction binding the contract method 0x83915971.
//
// Solidity: function createAssertion(((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)),((bytes32[2],uint64[2]),uint8,bytes32),((bytes32[2],uint64[2]),uint8,bytes32)) assertionInputs) returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) CreateAssertion(assertionInputs AssertionInputs) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.CreateAssertion(&_AssertionStakingPool.TransactOpts, assertionInputs)
}

// CreateAssertion is a paid mutator transaction binding the contract method 0x83915971.
//
// Solidity: function createAssertion(((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)),((bytes32[2],uint64[2]),uint8,bytes32),((bytes32[2],uint64[2]),uint8,bytes32)) assertionInputs) returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) CreateAssertion(assertionInputs AssertionInputs) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.CreateAssertion(&_AssertionStakingPool.TransactOpts, assertionInputs)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) DepositIntoPool(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "depositIntoPool", amount)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) DepositIntoPool(amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.DepositIntoPool(&_AssertionStakingPool.TransactOpts, amount)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) DepositIntoPool(amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.DepositIntoPool(&_AssertionStakingPool.TransactOpts, amount)
}

// MakeStakeWithdrawable is a paid mutator transaction binding the contract method 0x930412af.
//
// Solidity: function makeStakeWithdrawable() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) MakeStakeWithdrawable(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "makeStakeWithdrawable")
}

// MakeStakeWithdrawable is a paid mutator transaction binding the contract method 0x930412af.
//
// Solidity: function makeStakeWithdrawable() returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) MakeStakeWithdrawable() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.MakeStakeWithdrawable(&_AssertionStakingPool.TransactOpts)
}

// MakeStakeWithdrawable is a paid mutator transaction binding the contract method 0x930412af.
//
// Solidity: function makeStakeWithdrawable() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) MakeStakeWithdrawable() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.MakeStakeWithdrawable(&_AssertionStakingPool.TransactOpts)
}

// MakeStakeWithdrawableAndWithdrawBackIntoPool is a paid mutator transaction binding the contract method 0x6b74d515.
//
// Solidity: function makeStakeWithdrawableAndWithdrawBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) MakeStakeWithdrawableAndWithdrawBackIntoPool(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "makeStakeWithdrawableAndWithdrawBackIntoPool")
}

// MakeStakeWithdrawableAndWithdrawBackIntoPool is a paid mutator transaction binding the contract method 0x6b74d515.
//
// Solidity: function makeStakeWithdrawableAndWithdrawBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) MakeStakeWithdrawableAndWithdrawBackIntoPool() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.MakeStakeWithdrawableAndWithdrawBackIntoPool(&_AssertionStakingPool.TransactOpts)
}

// MakeStakeWithdrawableAndWithdrawBackIntoPool is a paid mutator transaction binding the contract method 0x6b74d515.
//
// Solidity: function makeStakeWithdrawableAndWithdrawBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) MakeStakeWithdrawableAndWithdrawBackIntoPool() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.MakeStakeWithdrawableAndWithdrawBackIntoPool(&_AssertionStakingPool.TransactOpts)
}

// WithdrawFromPool26c0e5c5 is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) WithdrawFromPool26c0e5c5(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "withdrawFromPool")
}

// WithdrawFromPool26c0e5c5 is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) WithdrawFromPool26c0e5c5() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawFromPool26c0e5c5(&_AssertionStakingPool.TransactOpts)
}

// WithdrawFromPool26c0e5c5 is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) WithdrawFromPool26c0e5c5() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawFromPool26c0e5c5(&_AssertionStakingPool.TransactOpts)
}

// WithdrawFromPool30fc43ed is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) WithdrawFromPool30fc43ed(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "withdrawFromPool0", amount)
}

// WithdrawFromPool30fc43ed is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) WithdrawFromPool30fc43ed(amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawFromPool30fc43ed(&_AssertionStakingPool.TransactOpts, amount)
}

// WithdrawFromPool30fc43ed is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) WithdrawFromPool30fc43ed(amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawFromPool30fc43ed(&_AssertionStakingPool.TransactOpts, amount)
}

// WithdrawStakeBackIntoPool is a paid mutator transaction binding the contract method 0x9451944d.
//
// Solidity: function withdrawStakeBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) WithdrawStakeBackIntoPool(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "withdrawStakeBackIntoPool")
}

// WithdrawStakeBackIntoPool is a paid mutator transaction binding the contract method 0x9451944d.
//
// Solidity: function withdrawStakeBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) WithdrawStakeBackIntoPool() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawStakeBackIntoPool(&_AssertionStakingPool.TransactOpts)
}

// WithdrawStakeBackIntoPool is a paid mutator transaction binding the contract method 0x9451944d.
//
// Solidity: function withdrawStakeBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) WithdrawStakeBackIntoPool() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawStakeBackIntoPool(&_AssertionStakingPool.TransactOpts)
}

// AssertionStakingPoolStakeDepositedIterator is returned from FilterStakeDeposited and is used to iterate over the raw logs and unpacked data for StakeDeposited events raised by the AssertionStakingPool contract.
type AssertionStakingPoolStakeDepositedIterator struct {
	Event *AssertionStakingPoolStakeDeposited // Event containing the contract specifics and raw log

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
func (it *AssertionStakingPoolStakeDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AssertionStakingPoolStakeDeposited)
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
		it.Event = new(AssertionStakingPoolStakeDeposited)
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
func (it *AssertionStakingPoolStakeDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AssertionStakingPoolStakeDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AssertionStakingPoolStakeDeposited represents a StakeDeposited event raised by the AssertionStakingPool contract.
type AssertionStakingPoolStakeDeposited struct {
	Sender common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStakeDeposited is a free log retrieval operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) FilterStakeDeposited(opts *bind.FilterOpts, sender []common.Address) (*AssertionStakingPoolStakeDepositedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AssertionStakingPool.contract.FilterLogs(opts, "StakeDeposited", senderRule)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolStakeDepositedIterator{contract: _AssertionStakingPool.contract, event: "StakeDeposited", logs: logs, sub: sub}, nil
}

// WatchStakeDeposited is a free log subscription operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) WatchStakeDeposited(opts *bind.WatchOpts, sink chan<- *AssertionStakingPoolStakeDeposited, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AssertionStakingPool.contract.WatchLogs(opts, "StakeDeposited", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AssertionStakingPoolStakeDeposited)
				if err := _AssertionStakingPool.contract.UnpackLog(event, "StakeDeposited", log); err != nil {
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

// ParseStakeDeposited is a log parse operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) ParseStakeDeposited(log types.Log) (*AssertionStakingPoolStakeDeposited, error) {
	event := new(AssertionStakingPoolStakeDeposited)
	if err := _AssertionStakingPool.contract.UnpackLog(event, "StakeDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AssertionStakingPoolStakeWithdrawnIterator is returned from FilterStakeWithdrawn and is used to iterate over the raw logs and unpacked data for StakeWithdrawn events raised by the AssertionStakingPool contract.
type AssertionStakingPoolStakeWithdrawnIterator struct {
	Event *AssertionStakingPoolStakeWithdrawn // Event containing the contract specifics and raw log

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
func (it *AssertionStakingPoolStakeWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AssertionStakingPoolStakeWithdrawn)
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
		it.Event = new(AssertionStakingPoolStakeWithdrawn)
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
func (it *AssertionStakingPoolStakeWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AssertionStakingPoolStakeWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AssertionStakingPoolStakeWithdrawn represents a StakeWithdrawn event raised by the AssertionStakingPool contract.
type AssertionStakingPoolStakeWithdrawn struct {
	Sender common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStakeWithdrawn is a free log retrieval operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) FilterStakeWithdrawn(opts *bind.FilterOpts, sender []common.Address) (*AssertionStakingPoolStakeWithdrawnIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AssertionStakingPool.contract.FilterLogs(opts, "StakeWithdrawn", senderRule)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolStakeWithdrawnIterator{contract: _AssertionStakingPool.contract, event: "StakeWithdrawn", logs: logs, sub: sub}, nil
}

// WatchStakeWithdrawn is a free log subscription operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) WatchStakeWithdrawn(opts *bind.WatchOpts, sink chan<- *AssertionStakingPoolStakeWithdrawn, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AssertionStakingPool.contract.WatchLogs(opts, "StakeWithdrawn", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AssertionStakingPoolStakeWithdrawn)
				if err := _AssertionStakingPool.contract.UnpackLog(event, "StakeWithdrawn", log); err != nil {
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

// ParseStakeWithdrawn is a log parse operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) ParseStakeWithdrawn(log types.Log) (*AssertionStakingPoolStakeWithdrawn, error) {
	event := new(AssertionStakingPoolStakeWithdrawn)
	if err := _AssertionStakingPool.contract.UnpackLog(event, "StakeWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AssertionStakingPoolCreatorMetaData contains all meta data concerning the AssertionStakingPoolCreator contract.
var AssertionStakingPoolCreatorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"PoolDoesntExist\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_assertionHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"assertionPool\",\"type\":\"address\"}],\"name\":\"NewAssertionPoolCreated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollup\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"_assertionHash\",\"type\":\"bytes32\"}],\"name\":\"createPool\",\"outputs\":[{\"internalType\":\"contractIAssertionStakingPool\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollup\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"_assertionHash\",\"type\":\"bytes32\"}],\"name\":\"getPool\",\"outputs\":[{\"internalType\":\"contractIAssertionStakingPool\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611263806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80639b505aa11461003b578063dc082ad314610077575b600080fd5b61004e6100493660046102b5565b61008a565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200160405180910390f35b61004e6100853660046102b5565b61013d565b6000806000801b848460405161009f906102a8565b73ffffffffffffffffffffffffffffffffffffffff909216825260208201526040018190604051809103906000f59050801580156100e1573d6000803e3d6000fd5b5060405173ffffffffffffffffffffffffffffffffffffffff808316825291925084918616907fd628317c6ebae87acc5dbfadeb835cb97692cc6935ea72bf37461e14a0bbee1e9060200160405180910390a390505b92915050565b600061019f60405180602001610152906102a8565b601f1982820381018352601f90910116604081815273ffffffffffffffffffffffffffffffffffffffff8716602083015281018590526060016040516020818303038152906040526101a6565b9392505050565b60008083836040516020016101bc92919061032a565b60408051808303601f1901815282825280516020918201207fff00000000000000000000000000000000000000000000000000000000000000828501523060601b7fffffffffffffffffffffffffffffffffffffffff000000000000000000000000166021850152600060358501526055808501829052835180860390910181526075909401909252825192019190912090915073ffffffffffffffffffffffffffffffffffffffff81163b156102765791506101379050565b6040517f215db33100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b610ee68061034883390190565b600080604083850312156102c857600080fd5b823573ffffffffffffffffffffffffffffffffffffffff811681146102ec57600080fd5b946020939093013593505050565b6000815160005b8181101561031b5760208185018101518683015201610301565b50600093019283525090919050565b600061033f61033983866102fa565b846102fa565b94935050505056fe60e060405234801561001057600080fd5b50604051610ee6380380610ee683398101604081905261002f916100ca565b816001600160a01b03166351ed6a306040518163ffffffff1660e01b8152600401602060405180830381865afa15801561006d573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061009191906100f6565b6001600160a01b039081166080529190911660a05260c052610118565b80516001600160a01b03811681146100c557600080fd5b919050565b600080604083850312156100dd57600080fd5b6100e6836100ae565b9150602083015190509250929050565b60006020828403121561010857600080fd5b610111826100ae565b9392505050565b60805160a05160c051610d6861017e6000396000818160d301526104be0152600081816101c7015281816104380152818161048d01528181610520015261059501526000818161012a015281816102c0015281816103a401526104160152610d686000f3fe608060405234801561001057600080fd5b50600436106100c95760003560e01c80637476083b116100815780639451944d1161005b5780639451944d1461019a578063956501bb146101a2578063cb23bcb5146101c257600080fd5b80637476083b1461016c578063839159711461017f578063930412af1461019257600080fd5b806330fc43ed116100b257806330fc43ed1461011257806351ed6a30146101255780636b74d5151461016457600080fd5b80632113ed21146100ce57806326c0e5c514610108575b600080fd5b6100f57f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020015b60405180910390f35b6101106101e9565b005b610110610120366004610a7c565b610204565b61014c7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b0390911681526020016100ff565b610110610328565b61011061017a366004610a7c565b610338565b61011061018d366004610a95565b610404565b61011061051e565b610110610593565b6100f56101b0366004610aca565b60006020819052908152604090205481565b61014c7f000000000000000000000000000000000000000000000000000000000000000081565b3360009081526020819052604090205461020290610204565b565b8060000361023e576040517f1f2a200500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b336000908152602081905260409020548082111561029d576040517fa47b7c6500000000000000000000000000000000000000000000000000000000815233600482015260248101839052604481018290526064015b60405180910390fd5b6102a78282610b14565b336000818152602081905260409020919091556102ef907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316908461061a565b60405182815233907f8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc9060200160405180910390a25050565b61033061051e565b610202610593565b80600003610372576040517f1f2a200500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526020819052604081208054839290610391908490610b2d565b909155506103cc90506001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163330846106c8565b60405181815233907f0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc29060200160405180910390a250565b606081013561045d6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000167f000000000000000000000000000000000000000000000000000000000000000083610719565b6040517f50f32f680000000000000000000000000000000000000000000000000000000081526001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016906350f32f68906104e890849086907f0000000000000000000000000000000000000000000000000000000000000000903090600401610bbe565b600060405180830381600087803b15801561050257600080fd5b505af1158015610516573d6000803e3d6000fd5b505050505050565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03166357ef4ab96040518163ffffffff1660e01b8152600401600060405180830381600087803b15801561057957600080fd5b505af115801561058d573d6000803e3d6000fd5b50505050565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031663613739196040518163ffffffff1660e01b81526004016020604051808303816000875af11580156105f3573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906106179190610c84565b50565b6040516001600160a01b0383166024820152604481018290526106c39084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff00000000000000000000000000000000000000000000000000000000909316929092179091526107fd565b505050565b6040516001600160a01b038085166024830152831660448201526064810182905261058d9085907f23b872dd000000000000000000000000000000000000000000000000000000009060840161065f565b6040517fdd62ed3e0000000000000000000000000000000000000000000000000000000081523060048201526001600160a01b038381166024830152600091839186169063dd62ed3e90604401602060405180830381865afa158015610783573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107a79190610c84565b6107b19190610b2d565b6040516001600160a01b03851660248201526044810182905290915061058d9085907f095ea7b3000000000000000000000000000000000000000000000000000000009060640161065f565b6000610852826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564815250856001600160a01b03166108e29092919063ffffffff16565b8051909150156106c357808060200190518101906108709190610c9d565b6106c35760405162461bcd60e51b815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f742073756363656564000000000000000000000000000000000000000000006064820152608401610294565b60606108f184846000856108fb565b90505b9392505050565b6060824710156109735760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c00000000000000000000000000000000000000000000000000006064820152608401610294565b6001600160a01b0385163b6109ca5760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e74726163740000006044820152606401610294565b600080866001600160a01b031685876040516109e69190610ce3565b60006040518083038185875af1925050503d8060008114610a23576040519150601f19603f3d011682016040523d82523d6000602084013e610a28565b606091505b5091509150610a38828286610a43565b979650505050505050565b60608315610a525750816108f4565b825115610a625782518084602001fd5b8160405162461bcd60e51b81526004016102949190610cff565b600060208284031215610a8e57600080fd5b5035919050565b60006102608284031215610aa857600080fd5b50919050565b80356001600160a01b0381168114610ac557600080fd5b919050565b600060208284031215610adc57600080fd5b6108f482610aae565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b81810381811115610b2757610b27610ae5565b92915050565b80820180821115610b2757610b27610ae5565b803567ffffffffffffffff81168114610ac557600080fd5b6040818337604082016040820160005b6002811015610b995767ffffffffffffffff610b8383610b40565b1683526020928301929190910190600101610b68565b505050608081013560038110610bae57600080fd5b608083015260a090810135910152565b60006102c082019050858252843560208301526020850135604083015260408501356060830152606085013560808301526001600160a01b03610c0360808701610aae565b1660a0830152610c1560a08601610b40565b67ffffffffffffffff80821660c085015280610c3360c08901610b40565b1660e08501525050610c4c610100830160e08701610b58565b610c5e6101c083016101a08701610b58565b83610280830152610c7b6102a08301846001600160a01b03169052565b95945050505050565b600060208284031215610c9657600080fd5b5051919050565b600060208284031215610caf57600080fd5b815180151581146108f457600080fd5b60005b83811015610cda578181015183820152602001610cc2565b50506000910152565b60008251610cf5818460208701610cbf565b9190910192915050565b6020815260008251806020840152610d1e816040850160208701610cbf565b601f01601f1916919091016040019291505056fea26469706673582212207ff3737da395004ce78939e28894c41d6b5bd0d08c916ef4463a65833d328dd964736f6c63430008110033a2646970667358221220c49a2118677c91524459a1d45a0e62ead52fc3df98e24081e040f233d6ca276464736f6c63430008110033",
}

// AssertionStakingPoolCreatorABI is the input ABI used to generate the binding from.
// Deprecated: Use AssertionStakingPoolCreatorMetaData.ABI instead.
var AssertionStakingPoolCreatorABI = AssertionStakingPoolCreatorMetaData.ABI

// AssertionStakingPoolCreatorBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AssertionStakingPoolCreatorMetaData.Bin instead.
var AssertionStakingPoolCreatorBin = AssertionStakingPoolCreatorMetaData.Bin

// DeployAssertionStakingPoolCreator deploys a new Ethereum contract, binding an instance of AssertionStakingPoolCreator to it.
func DeployAssertionStakingPoolCreator(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *AssertionStakingPoolCreator, error) {
	parsed, err := AssertionStakingPoolCreatorMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AssertionStakingPoolCreatorBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AssertionStakingPoolCreator{AssertionStakingPoolCreatorCaller: AssertionStakingPoolCreatorCaller{contract: contract}, AssertionStakingPoolCreatorTransactor: AssertionStakingPoolCreatorTransactor{contract: contract}, AssertionStakingPoolCreatorFilterer: AssertionStakingPoolCreatorFilterer{contract: contract}}, nil
}

// AssertionStakingPoolCreator is an auto generated Go binding around an Ethereum contract.
type AssertionStakingPoolCreator struct {
	AssertionStakingPoolCreatorCaller     // Read-only binding to the contract
	AssertionStakingPoolCreatorTransactor // Write-only binding to the contract
	AssertionStakingPoolCreatorFilterer   // Log filterer for contract events
}

// AssertionStakingPoolCreatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type AssertionStakingPoolCreatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolCreatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AssertionStakingPoolCreatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolCreatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AssertionStakingPoolCreatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolCreatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AssertionStakingPoolCreatorSession struct {
	Contract     *AssertionStakingPoolCreator // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                // Call options to use throughout this session
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// AssertionStakingPoolCreatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AssertionStakingPoolCreatorCallerSession struct {
	Contract *AssertionStakingPoolCreatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                      // Call options to use throughout this session
}

// AssertionStakingPoolCreatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AssertionStakingPoolCreatorTransactorSession struct {
	Contract     *AssertionStakingPoolCreatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                      // Transaction auth options to use throughout this session
}

// AssertionStakingPoolCreatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type AssertionStakingPoolCreatorRaw struct {
	Contract *AssertionStakingPoolCreator // Generic contract binding to access the raw methods on
}

// AssertionStakingPoolCreatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AssertionStakingPoolCreatorCallerRaw struct {
	Contract *AssertionStakingPoolCreatorCaller // Generic read-only contract binding to access the raw methods on
}

// AssertionStakingPoolCreatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AssertionStakingPoolCreatorTransactorRaw struct {
	Contract *AssertionStakingPoolCreatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAssertionStakingPoolCreator creates a new instance of AssertionStakingPoolCreator, bound to a specific deployed contract.
func NewAssertionStakingPoolCreator(address common.Address, backend bind.ContractBackend) (*AssertionStakingPoolCreator, error) {
	contract, err := bindAssertionStakingPoolCreator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCreator{AssertionStakingPoolCreatorCaller: AssertionStakingPoolCreatorCaller{contract: contract}, AssertionStakingPoolCreatorTransactor: AssertionStakingPoolCreatorTransactor{contract: contract}, AssertionStakingPoolCreatorFilterer: AssertionStakingPoolCreatorFilterer{contract: contract}}, nil
}

// NewAssertionStakingPoolCreatorCaller creates a new read-only instance of AssertionStakingPoolCreator, bound to a specific deployed contract.
func NewAssertionStakingPoolCreatorCaller(address common.Address, caller bind.ContractCaller) (*AssertionStakingPoolCreatorCaller, error) {
	contract, err := bindAssertionStakingPoolCreator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCreatorCaller{contract: contract}, nil
}

// NewAssertionStakingPoolCreatorTransactor creates a new write-only instance of AssertionStakingPoolCreator, bound to a specific deployed contract.
func NewAssertionStakingPoolCreatorTransactor(address common.Address, transactor bind.ContractTransactor) (*AssertionStakingPoolCreatorTransactor, error) {
	contract, err := bindAssertionStakingPoolCreator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCreatorTransactor{contract: contract}, nil
}

// NewAssertionStakingPoolCreatorFilterer creates a new log filterer instance of AssertionStakingPoolCreator, bound to a specific deployed contract.
func NewAssertionStakingPoolCreatorFilterer(address common.Address, filterer bind.ContractFilterer) (*AssertionStakingPoolCreatorFilterer, error) {
	contract, err := bindAssertionStakingPoolCreator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCreatorFilterer{contract: contract}, nil
}

// bindAssertionStakingPoolCreator binds a generic wrapper to an already deployed contract.
func bindAssertionStakingPoolCreator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AssertionStakingPoolCreatorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AssertionStakingPoolCreator.Contract.AssertionStakingPoolCreatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.AssertionStakingPoolCreatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.AssertionStakingPoolCreatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AssertionStakingPoolCreator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.contract.Transact(opts, method, params...)
}

// GetPool is a free data retrieval call binding the contract method 0xdc082ad3.
//
// Solidity: function getPool(address _rollup, bytes32 _assertionHash) view returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorCaller) GetPool(opts *bind.CallOpts, _rollup common.Address, _assertionHash [32]byte) (common.Address, error) {
	var out []interface{}
	err := _AssertionStakingPoolCreator.contract.Call(opts, &out, "getPool", _rollup, _assertionHash)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetPool is a free data retrieval call binding the contract method 0xdc082ad3.
//
// Solidity: function getPool(address _rollup, bytes32 _assertionHash) view returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorSession) GetPool(_rollup common.Address, _assertionHash [32]byte) (common.Address, error) {
	return _AssertionStakingPoolCreator.Contract.GetPool(&_AssertionStakingPoolCreator.CallOpts, _rollup, _assertionHash)
}

// GetPool is a free data retrieval call binding the contract method 0xdc082ad3.
//
// Solidity: function getPool(address _rollup, bytes32 _assertionHash) view returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorCallerSession) GetPool(_rollup common.Address, _assertionHash [32]byte) (common.Address, error) {
	return _AssertionStakingPoolCreator.Contract.GetPool(&_AssertionStakingPoolCreator.CallOpts, _rollup, _assertionHash)
}

// CreatePool is a paid mutator transaction binding the contract method 0x9b505aa1.
//
// Solidity: function createPool(address _rollup, bytes32 _assertionHash) returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorTransactor) CreatePool(opts *bind.TransactOpts, _rollup common.Address, _assertionHash [32]byte) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.contract.Transact(opts, "createPool", _rollup, _assertionHash)
}

// CreatePool is a paid mutator transaction binding the contract method 0x9b505aa1.
//
// Solidity: function createPool(address _rollup, bytes32 _assertionHash) returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorSession) CreatePool(_rollup common.Address, _assertionHash [32]byte) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.CreatePool(&_AssertionStakingPoolCreator.TransactOpts, _rollup, _assertionHash)
}

// CreatePool is a paid mutator transaction binding the contract method 0x9b505aa1.
//
// Solidity: function createPool(address _rollup, bytes32 _assertionHash) returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorTransactorSession) CreatePool(_rollup common.Address, _assertionHash [32]byte) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.CreatePool(&_AssertionStakingPoolCreator.TransactOpts, _rollup, _assertionHash)
}

// AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator is returned from FilterNewAssertionPoolCreated and is used to iterate over the raw logs and unpacked data for NewAssertionPoolCreated events raised by the AssertionStakingPoolCreator contract.
type AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator struct {
	Event *AssertionStakingPoolCreatorNewAssertionPoolCreated // Event containing the contract specifics and raw log

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
func (it *AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AssertionStakingPoolCreatorNewAssertionPoolCreated)
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
		it.Event = new(AssertionStakingPoolCreatorNewAssertionPoolCreated)
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
func (it *AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AssertionStakingPoolCreatorNewAssertionPoolCreated represents a NewAssertionPoolCreated event raised by the AssertionStakingPoolCreator contract.
type AssertionStakingPoolCreatorNewAssertionPoolCreated struct {
	Rollup        common.Address
	AssertionHash [32]byte
	AssertionPool common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterNewAssertionPoolCreated is a free log retrieval operation binding the contract event 0xd628317c6ebae87acc5dbfadeb835cb97692cc6935ea72bf37461e14a0bbee1e.
//
// Solidity: event NewAssertionPoolCreated(address indexed rollup, bytes32 indexed _assertionHash, address assertionPool)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorFilterer) FilterNewAssertionPoolCreated(opts *bind.FilterOpts, rollup []common.Address, _assertionHash [][32]byte) (*AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator, error) {

	var rollupRule []interface{}
	for _, rollupItem := range rollup {
		rollupRule = append(rollupRule, rollupItem)
	}
	var _assertionHashRule []interface{}
	for _, _assertionHashItem := range _assertionHash {
		_assertionHashRule = append(_assertionHashRule, _assertionHashItem)
	}

	logs, sub, err := _AssertionStakingPoolCreator.contract.FilterLogs(opts, "NewAssertionPoolCreated", rollupRule, _assertionHashRule)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator{contract: _AssertionStakingPoolCreator.contract, event: "NewAssertionPoolCreated", logs: logs, sub: sub}, nil
}

// WatchNewAssertionPoolCreated is a free log subscription operation binding the contract event 0xd628317c6ebae87acc5dbfadeb835cb97692cc6935ea72bf37461e14a0bbee1e.
//
// Solidity: event NewAssertionPoolCreated(address indexed rollup, bytes32 indexed _assertionHash, address assertionPool)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorFilterer) WatchNewAssertionPoolCreated(opts *bind.WatchOpts, sink chan<- *AssertionStakingPoolCreatorNewAssertionPoolCreated, rollup []common.Address, _assertionHash [][32]byte) (event.Subscription, error) {

	var rollupRule []interface{}
	for _, rollupItem := range rollup {
		rollupRule = append(rollupRule, rollupItem)
	}
	var _assertionHashRule []interface{}
	for _, _assertionHashItem := range _assertionHash {
		_assertionHashRule = append(_assertionHashRule, _assertionHashItem)
	}

	logs, sub, err := _AssertionStakingPoolCreator.contract.WatchLogs(opts, "NewAssertionPoolCreated", rollupRule, _assertionHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AssertionStakingPoolCreatorNewAssertionPoolCreated)
				if err := _AssertionStakingPoolCreator.contract.UnpackLog(event, "NewAssertionPoolCreated", log); err != nil {
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

// ParseNewAssertionPoolCreated is a log parse operation binding the contract event 0xd628317c6ebae87acc5dbfadeb835cb97692cc6935ea72bf37461e14a0bbee1e.
//
// Solidity: event NewAssertionPoolCreated(address indexed rollup, bytes32 indexed _assertionHash, address assertionPool)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorFilterer) ParseNewAssertionPoolCreated(log types.Log) (*AssertionStakingPoolCreatorNewAssertionPoolCreated, error) {
	event := new(AssertionStakingPoolCreatorNewAssertionPoolCreated)
	if err := _AssertionStakingPoolCreator.contract.UnpackLog(event, "NewAssertionPoolCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeStakingPoolMetaData contains all meta data concerning the EdgeStakingPool contract.
var EdgeStakingPoolMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_challengeManager\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"_edgeId\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"name\":\"AmountExceedsBalance\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"actual\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"expected\",\"type\":\"bytes32\"}],\"name\":\"IncorrectEdgeId\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAmount\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"StakeDeposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"StakeWithdrawn\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"challengeManager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint8\",\"name\":\"level\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"internalType\":\"structCreateEdgeArgs\",\"name\":\"args\",\"type\":\"tuple\"}],\"name\":\"createEdge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"depositBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"depositIntoPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"edgeId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stakeToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawFromPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawFromPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60e060405234801561001057600080fd5b50604051610ec8380380610ec883398101604081905261002f916100c6565b816001600160a01b03166351ed6a306040518163ffffffff1660e01b8152600401602060405180830381865afa15801561006d573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061009191906100f4565b6001600160a01b039081166080529190911660a05260c052610118565b6001600160a01b03811681146100c357600080fd5b50565b600080604083850312156100d957600080fd5b82516100e4816100ae565b6020939093015192949293505050565b60006020828403121561010657600080fd5b8151610111816100ae565b9392505050565b60805160a05160c051610d4b61017d6000396000818161015b0152818161055801526105ac0152600081816092015281816103a70152818161048401526104dc01526000818160f3015281816102670152818161033b01526104620152610d4b6000f3fe608060405234801561001057600080fd5b50600436106100885760003560e01c80637476083b1161005b5780637476083b14610115578063956501bb146101285780639cfa2a2a14610156578063bd3eec7d1461017d57600080fd5b8063023a96fe1461008d57806326c0e5c5146100d157806330fc43ed146100db57806351ed6a30146100ee575b600080fd5b6100b47f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b0390911681526020015b60405180910390f35b6100d9610190565b005b6100d96100e9366004610a40565b6101ab565b6100b47f000000000000000000000000000000000000000000000000000000000000000081565b6100d9610123366004610a40565b6102cf565b610148610136366004610a59565b60006020819052908152604090205481565b6040519081526020016100c8565b6101487f000000000000000000000000000000000000000000000000000000000000000081565b6100d961018b366004610a82565b61039b565b336000908152602081905260409020546101a9906101ab565b565b806000036101e5576040517f1f2a200500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526020819052604090205480821115610244576040517fa47b7c6500000000000000000000000000000000000000000000000000000000815233600482015260248101839052604481018290526064015b60405180910390fd5b61024e8282610aec565b33600081815260208190526040902091909155610296907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031690846105dd565b60405182815233907f8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc9060200160405180910390a25050565b80600003610309576040517f1f2a200500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526020819052604081208054839290610328908490610b05565b9091555061036390506001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016333084610686565b60405181815233907f0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc29060200160405180910390a250565b60006001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016631c1b4f3a6103d96020850185610b2e565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e084901b16815260ff9091166004820152602401602060405180830381865afa15801561042f573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906104539190610b49565b90506104a96001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000167f0000000000000000000000000000000000000000000000000000000000000000836106dd565b6040517f05fae1410000000000000000000000000000000000000000000000000000000081526000906001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016906305fae14190610511908690600401610bf8565b6020604051808303816000875af1158015610530573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105549190610b49565b90507f000000000000000000000000000000000000000000000000000000000000000081146105d8576040517f75c0811b000000000000000000000000000000000000000000000000000000008152600481018290527f0000000000000000000000000000000000000000000000000000000000000000602482015260440161023b565b505050565b6040516001600160a01b0383166024820152604481018290526105d89084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff00000000000000000000000000000000000000000000000000000000909316929092179091526107c1565b6040516001600160a01b03808516602483015283166044820152606481018290526106d79085907f23b872dd0000000000000000000000000000000000000000000000000000000090608401610622565b50505050565b6040517fdd62ed3e0000000000000000000000000000000000000000000000000000000081523060048201526001600160a01b038381166024830152600091839186169063dd62ed3e90604401602060405180830381865afa158015610747573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061076b9190610b49565b6107759190610b05565b6040516001600160a01b0385166024820152604481018290529091506106d79085907f095ea7b30000000000000000000000000000000000000000000000000000000090606401610622565b6000610816826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564815250856001600160a01b03166108a69092919063ffffffff16565b8051909150156105d857808060200190518101906108349190610c80565b6105d85760405162461bcd60e51b815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f74207375636365656400000000000000000000000000000000000000000000606482015260840161023b565b60606108b584846000856108bf565b90505b9392505050565b6060824710156109375760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c0000000000000000000000000000000000000000000000000000606482015260840161023b565b6001600160a01b0385163b61098e5760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e7472616374000000604482015260640161023b565b600080866001600160a01b031685876040516109aa9190610cc6565b60006040518083038185875af1925050503d80600081146109e7576040519150601f19603f3d011682016040523d82523d6000602084013e6109ec565b606091505b50915091506109fc828286610a07565b979650505050505050565b60608315610a165750816108b8565b825115610a265782518084602001fd5b8160405162461bcd60e51b815260040161023b9190610ce2565b600060208284031215610a5257600080fd5b5035919050565b600060208284031215610a6b57600080fd5b81356001600160a01b03811681146108b857600080fd5b600060208284031215610a9457600080fd5b813567ffffffffffffffff811115610aab57600080fd5b820160c081850312156108b857600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b81810381811115610aff57610aff610abd565b92915050565b80820180821115610aff57610aff610abd565b803560ff81168114610b2957600080fd5b919050565b600060208284031215610b4057600080fd5b6108b882610b18565b600060208284031215610b5b57600080fd5b5051919050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112610b9757600080fd5b830160208101925035905067ffffffffffffffff811115610bb757600080fd5b803603821315610bc657600080fd5b9250929050565b818352818160208501375060006020828401015260006020601f19601f840116840101905092915050565b6020815260ff610c0783610b18565b1660208201526020820135604082015260408201356060820152606082013560808201526000610c3a6080840184610b62565b60c060a0850152610c4f60e085018284610bcd565b915050610c5f60a0850185610b62565b601f198584030160c0860152610c76838284610bcd565b9695505050505050565b600060208284031215610c9257600080fd5b815180151581146108b857600080fd5b60005b83811015610cbd578181015183820152602001610ca5565b50506000910152565b60008251610cd8818460208701610ca2565b9190910192915050565b6020815260008251806020840152610d01816040850160208701610ca2565b601f01601f1916919091016040019291505056fea2646970667358221220045bdaf31a036831af5ac9e940af092e3bee03f7326979b895e5c2146fd111a764736f6c63430008110033",
}

// EdgeStakingPoolABI is the input ABI used to generate the binding from.
// Deprecated: Use EdgeStakingPoolMetaData.ABI instead.
var EdgeStakingPoolABI = EdgeStakingPoolMetaData.ABI

// EdgeStakingPoolBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use EdgeStakingPoolMetaData.Bin instead.
var EdgeStakingPoolBin = EdgeStakingPoolMetaData.Bin

// DeployEdgeStakingPool deploys a new Ethereum contract, binding an instance of EdgeStakingPool to it.
func DeployEdgeStakingPool(auth *bind.TransactOpts, backend bind.ContractBackend, _challengeManager common.Address, _edgeId [32]byte) (common.Address, *types.Transaction, *EdgeStakingPool, error) {
	parsed, err := EdgeStakingPoolMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(EdgeStakingPoolBin), backend, _challengeManager, _edgeId)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &EdgeStakingPool{EdgeStakingPoolCaller: EdgeStakingPoolCaller{contract: contract}, EdgeStakingPoolTransactor: EdgeStakingPoolTransactor{contract: contract}, EdgeStakingPoolFilterer: EdgeStakingPoolFilterer{contract: contract}}, nil
}

// EdgeStakingPool is an auto generated Go binding around an Ethereum contract.
type EdgeStakingPool struct {
	EdgeStakingPoolCaller     // Read-only binding to the contract
	EdgeStakingPoolTransactor // Write-only binding to the contract
	EdgeStakingPoolFilterer   // Log filterer for contract events
}

// EdgeStakingPoolCaller is an auto generated read-only Go binding around an Ethereum contract.
type EdgeStakingPoolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeStakingPoolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type EdgeStakingPoolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeStakingPoolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type EdgeStakingPoolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeStakingPoolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type EdgeStakingPoolSession struct {
	Contract     *EdgeStakingPool  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// EdgeStakingPoolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type EdgeStakingPoolCallerSession struct {
	Contract *EdgeStakingPoolCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// EdgeStakingPoolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type EdgeStakingPoolTransactorSession struct {
	Contract     *EdgeStakingPoolTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// EdgeStakingPoolRaw is an auto generated low-level Go binding around an Ethereum contract.
type EdgeStakingPoolRaw struct {
	Contract *EdgeStakingPool // Generic contract binding to access the raw methods on
}

// EdgeStakingPoolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type EdgeStakingPoolCallerRaw struct {
	Contract *EdgeStakingPoolCaller // Generic read-only contract binding to access the raw methods on
}

// EdgeStakingPoolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type EdgeStakingPoolTransactorRaw struct {
	Contract *EdgeStakingPoolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewEdgeStakingPool creates a new instance of EdgeStakingPool, bound to a specific deployed contract.
func NewEdgeStakingPool(address common.Address, backend bind.ContractBackend) (*EdgeStakingPool, error) {
	contract, err := bindEdgeStakingPool(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPool{EdgeStakingPoolCaller: EdgeStakingPoolCaller{contract: contract}, EdgeStakingPoolTransactor: EdgeStakingPoolTransactor{contract: contract}, EdgeStakingPoolFilterer: EdgeStakingPoolFilterer{contract: contract}}, nil
}

// NewEdgeStakingPoolCaller creates a new read-only instance of EdgeStakingPool, bound to a specific deployed contract.
func NewEdgeStakingPoolCaller(address common.Address, caller bind.ContractCaller) (*EdgeStakingPoolCaller, error) {
	contract, err := bindEdgeStakingPool(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPoolCaller{contract: contract}, nil
}

// NewEdgeStakingPoolTransactor creates a new write-only instance of EdgeStakingPool, bound to a specific deployed contract.
func NewEdgeStakingPoolTransactor(address common.Address, transactor bind.ContractTransactor) (*EdgeStakingPoolTransactor, error) {
	contract, err := bindEdgeStakingPool(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPoolTransactor{contract: contract}, nil
}

// NewEdgeStakingPoolFilterer creates a new log filterer instance of EdgeStakingPool, bound to a specific deployed contract.
func NewEdgeStakingPoolFilterer(address common.Address, filterer bind.ContractFilterer) (*EdgeStakingPoolFilterer, error) {
	contract, err := bindEdgeStakingPool(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPoolFilterer{contract: contract}, nil
}

// bindEdgeStakingPool binds a generic wrapper to an already deployed contract.
func bindEdgeStakingPool(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := EdgeStakingPoolMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EdgeStakingPool *EdgeStakingPoolRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EdgeStakingPool.Contract.EdgeStakingPoolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EdgeStakingPool *EdgeStakingPoolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.EdgeStakingPoolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EdgeStakingPool *EdgeStakingPoolRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.EdgeStakingPoolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EdgeStakingPool *EdgeStakingPoolCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EdgeStakingPool.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EdgeStakingPool *EdgeStakingPoolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EdgeStakingPool *EdgeStakingPoolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.contract.Transact(opts, method, params...)
}

// ChallengeManager is a free data retrieval call binding the contract method 0x023a96fe.
//
// Solidity: function challengeManager() view returns(address)
func (_EdgeStakingPool *EdgeStakingPoolCaller) ChallengeManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _EdgeStakingPool.contract.Call(opts, &out, "challengeManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ChallengeManager is a free data retrieval call binding the contract method 0x023a96fe.
//
// Solidity: function challengeManager() view returns(address)
func (_EdgeStakingPool *EdgeStakingPoolSession) ChallengeManager() (common.Address, error) {
	return _EdgeStakingPool.Contract.ChallengeManager(&_EdgeStakingPool.CallOpts)
}

// ChallengeManager is a free data retrieval call binding the contract method 0x023a96fe.
//
// Solidity: function challengeManager() view returns(address)
func (_EdgeStakingPool *EdgeStakingPoolCallerSession) ChallengeManager() (common.Address, error) {
	return _EdgeStakingPool.Contract.ChallengeManager(&_EdgeStakingPool.CallOpts)
}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_EdgeStakingPool *EdgeStakingPoolCaller) DepositBalance(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _EdgeStakingPool.contract.Call(opts, &out, "depositBalance", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_EdgeStakingPool *EdgeStakingPoolSession) DepositBalance(arg0 common.Address) (*big.Int, error) {
	return _EdgeStakingPool.Contract.DepositBalance(&_EdgeStakingPool.CallOpts, arg0)
}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_EdgeStakingPool *EdgeStakingPoolCallerSession) DepositBalance(arg0 common.Address) (*big.Int, error) {
	return _EdgeStakingPool.Contract.DepositBalance(&_EdgeStakingPool.CallOpts, arg0)
}

// EdgeId is a free data retrieval call binding the contract method 0x9cfa2a2a.
//
// Solidity: function edgeId() view returns(bytes32)
func (_EdgeStakingPool *EdgeStakingPoolCaller) EdgeId(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _EdgeStakingPool.contract.Call(opts, &out, "edgeId")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// EdgeId is a free data retrieval call binding the contract method 0x9cfa2a2a.
//
// Solidity: function edgeId() view returns(bytes32)
func (_EdgeStakingPool *EdgeStakingPoolSession) EdgeId() ([32]byte, error) {
	return _EdgeStakingPool.Contract.EdgeId(&_EdgeStakingPool.CallOpts)
}

// EdgeId is a free data retrieval call binding the contract method 0x9cfa2a2a.
//
// Solidity: function edgeId() view returns(bytes32)
func (_EdgeStakingPool *EdgeStakingPoolCallerSession) EdgeId() ([32]byte, error) {
	return _EdgeStakingPool.Contract.EdgeId(&_EdgeStakingPool.CallOpts)
}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_EdgeStakingPool *EdgeStakingPoolCaller) StakeToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _EdgeStakingPool.contract.Call(opts, &out, "stakeToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_EdgeStakingPool *EdgeStakingPoolSession) StakeToken() (common.Address, error) {
	return _EdgeStakingPool.Contract.StakeToken(&_EdgeStakingPool.CallOpts)
}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_EdgeStakingPool *EdgeStakingPoolCallerSession) StakeToken() (common.Address, error) {
	return _EdgeStakingPool.Contract.StakeToken(&_EdgeStakingPool.CallOpts)
}

// CreateEdge is a paid mutator transaction binding the contract method 0xbd3eec7d.
//
// Solidity: function createEdge((uint8,bytes32,uint256,bytes32,bytes,bytes) args) returns()
func (_EdgeStakingPool *EdgeStakingPoolTransactor) CreateEdge(opts *bind.TransactOpts, args CreateEdgeArgs) (*types.Transaction, error) {
	return _EdgeStakingPool.contract.Transact(opts, "createEdge", args)
}

// CreateEdge is a paid mutator transaction binding the contract method 0xbd3eec7d.
//
// Solidity: function createEdge((uint8,bytes32,uint256,bytes32,bytes,bytes) args) returns()
func (_EdgeStakingPool *EdgeStakingPoolSession) CreateEdge(args CreateEdgeArgs) (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.CreateEdge(&_EdgeStakingPool.TransactOpts, args)
}

// CreateEdge is a paid mutator transaction binding the contract method 0xbd3eec7d.
//
// Solidity: function createEdge((uint8,bytes32,uint256,bytes32,bytes,bytes) args) returns()
func (_EdgeStakingPool *EdgeStakingPoolTransactorSession) CreateEdge(args CreateEdgeArgs) (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.CreateEdge(&_EdgeStakingPool.TransactOpts, args)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 amount) returns()
func (_EdgeStakingPool *EdgeStakingPoolTransactor) DepositIntoPool(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _EdgeStakingPool.contract.Transact(opts, "depositIntoPool", amount)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 amount) returns()
func (_EdgeStakingPool *EdgeStakingPoolSession) DepositIntoPool(amount *big.Int) (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.DepositIntoPool(&_EdgeStakingPool.TransactOpts, amount)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 amount) returns()
func (_EdgeStakingPool *EdgeStakingPoolTransactorSession) DepositIntoPool(amount *big.Int) (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.DepositIntoPool(&_EdgeStakingPool.TransactOpts, amount)
}

// WithdrawFromPool26c0e5c5 is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_EdgeStakingPool *EdgeStakingPoolTransactor) WithdrawFromPool26c0e5c5(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EdgeStakingPool.contract.Transact(opts, "withdrawFromPool")
}

// WithdrawFromPool26c0e5c5 is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_EdgeStakingPool *EdgeStakingPoolSession) WithdrawFromPool26c0e5c5() (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.WithdrawFromPool26c0e5c5(&_EdgeStakingPool.TransactOpts)
}

// WithdrawFromPool26c0e5c5 is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_EdgeStakingPool *EdgeStakingPoolTransactorSession) WithdrawFromPool26c0e5c5() (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.WithdrawFromPool26c0e5c5(&_EdgeStakingPool.TransactOpts)
}

// WithdrawFromPool30fc43ed is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 amount) returns()
func (_EdgeStakingPool *EdgeStakingPoolTransactor) WithdrawFromPool30fc43ed(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _EdgeStakingPool.contract.Transact(opts, "withdrawFromPool0", amount)
}

// WithdrawFromPool30fc43ed is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 amount) returns()
func (_EdgeStakingPool *EdgeStakingPoolSession) WithdrawFromPool30fc43ed(amount *big.Int) (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.WithdrawFromPool30fc43ed(&_EdgeStakingPool.TransactOpts, amount)
}

// WithdrawFromPool30fc43ed is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 amount) returns()
func (_EdgeStakingPool *EdgeStakingPoolTransactorSession) WithdrawFromPool30fc43ed(amount *big.Int) (*types.Transaction, error) {
	return _EdgeStakingPool.Contract.WithdrawFromPool30fc43ed(&_EdgeStakingPool.TransactOpts, amount)
}

// EdgeStakingPoolStakeDepositedIterator is returned from FilterStakeDeposited and is used to iterate over the raw logs and unpacked data for StakeDeposited events raised by the EdgeStakingPool contract.
type EdgeStakingPoolStakeDepositedIterator struct {
	Event *EdgeStakingPoolStakeDeposited // Event containing the contract specifics and raw log

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
func (it *EdgeStakingPoolStakeDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeStakingPoolStakeDeposited)
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
		it.Event = new(EdgeStakingPoolStakeDeposited)
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
func (it *EdgeStakingPoolStakeDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeStakingPoolStakeDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeStakingPoolStakeDeposited represents a StakeDeposited event raised by the EdgeStakingPool contract.
type EdgeStakingPoolStakeDeposited struct {
	Sender common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStakeDeposited is a free log retrieval operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_EdgeStakingPool *EdgeStakingPoolFilterer) FilterStakeDeposited(opts *bind.FilterOpts, sender []common.Address) (*EdgeStakingPoolStakeDepositedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _EdgeStakingPool.contract.FilterLogs(opts, "StakeDeposited", senderRule)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPoolStakeDepositedIterator{contract: _EdgeStakingPool.contract, event: "StakeDeposited", logs: logs, sub: sub}, nil
}

// WatchStakeDeposited is a free log subscription operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_EdgeStakingPool *EdgeStakingPoolFilterer) WatchStakeDeposited(opts *bind.WatchOpts, sink chan<- *EdgeStakingPoolStakeDeposited, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _EdgeStakingPool.contract.WatchLogs(opts, "StakeDeposited", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeStakingPoolStakeDeposited)
				if err := _EdgeStakingPool.contract.UnpackLog(event, "StakeDeposited", log); err != nil {
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

// ParseStakeDeposited is a log parse operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_EdgeStakingPool *EdgeStakingPoolFilterer) ParseStakeDeposited(log types.Log) (*EdgeStakingPoolStakeDeposited, error) {
	event := new(EdgeStakingPoolStakeDeposited)
	if err := _EdgeStakingPool.contract.UnpackLog(event, "StakeDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeStakingPoolStakeWithdrawnIterator is returned from FilterStakeWithdrawn and is used to iterate over the raw logs and unpacked data for StakeWithdrawn events raised by the EdgeStakingPool contract.
type EdgeStakingPoolStakeWithdrawnIterator struct {
	Event *EdgeStakingPoolStakeWithdrawn // Event containing the contract specifics and raw log

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
func (it *EdgeStakingPoolStakeWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeStakingPoolStakeWithdrawn)
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
		it.Event = new(EdgeStakingPoolStakeWithdrawn)
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
func (it *EdgeStakingPoolStakeWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeStakingPoolStakeWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeStakingPoolStakeWithdrawn represents a StakeWithdrawn event raised by the EdgeStakingPool contract.
type EdgeStakingPoolStakeWithdrawn struct {
	Sender common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStakeWithdrawn is a free log retrieval operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_EdgeStakingPool *EdgeStakingPoolFilterer) FilterStakeWithdrawn(opts *bind.FilterOpts, sender []common.Address) (*EdgeStakingPoolStakeWithdrawnIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _EdgeStakingPool.contract.FilterLogs(opts, "StakeWithdrawn", senderRule)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPoolStakeWithdrawnIterator{contract: _EdgeStakingPool.contract, event: "StakeWithdrawn", logs: logs, sub: sub}, nil
}

// WatchStakeWithdrawn is a free log subscription operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_EdgeStakingPool *EdgeStakingPoolFilterer) WatchStakeWithdrawn(opts *bind.WatchOpts, sink chan<- *EdgeStakingPoolStakeWithdrawn, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _EdgeStakingPool.contract.WatchLogs(opts, "StakeWithdrawn", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeStakingPoolStakeWithdrawn)
				if err := _EdgeStakingPool.contract.UnpackLog(event, "StakeWithdrawn", log); err != nil {
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

// ParseStakeWithdrawn is a log parse operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_EdgeStakingPool *EdgeStakingPoolFilterer) ParseStakeWithdrawn(log types.Log) (*EdgeStakingPoolStakeWithdrawn, error) {
	event := new(EdgeStakingPoolStakeWithdrawn)
	if err := _EdgeStakingPool.contract.UnpackLog(event, "StakeWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeStakingPoolCreatorMetaData contains all meta data concerning the EdgeStakingPoolCreator contract.
var EdgeStakingPoolCreatorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"PoolDoesntExist\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"NewEdgeStakingPoolCreated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"createPool\",\"outputs\":[{\"internalType\":\"contractIEdgeStakingPool\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"getPool\",\"outputs\":[{\"internalType\":\"contractIEdgeStakingPool\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611239806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80639b505aa11461003b578063dc082ad314610077575b600080fd5b61004e6100493660046102a9565b61008a565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200160405180910390f35b61004e6100853660046102a9565b610131565b6000806000801b848460405161009f9061029c565b73ffffffffffffffffffffffffffffffffffffffff909216825260208201526040018190604051809103906000f59050801580156100e1573d6000803e3d6000fd5b509050828473ffffffffffffffffffffffffffffffffffffffff167f15e71db3d71eb3b7985105d763101e1d6c1c491ab3e6a0d682558c12cc0bb8d660405160405180910390a390505b92915050565b6000610193604051806020016101469061029c565b601f1982820381018352601f90910116604081815273ffffffffffffffffffffffffffffffffffffffff87166020830152810185905260600160405160208183030381529060405261019a565b9392505050565b60008083836040516020016101b092919061031e565b60408051808303601f1901815282825280516020918201207fff00000000000000000000000000000000000000000000000000000000000000828501523060601b7fffffffffffffffffffffffffffffffffffffffff000000000000000000000000166021850152600060358501526055808501829052835180860390910181526075909401909252825192019190912090915073ffffffffffffffffffffffffffffffffffffffff81163b1561026a57915061012b9050565b6040517f215db33100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b610ec88061033c83390190565b600080604083850312156102bc57600080fd5b823573ffffffffffffffffffffffffffffffffffffffff811681146102e057600080fd5b946020939093013593505050565b6000815160005b8181101561030f57602081850181015186830152016102f5565b50600093019283525090919050565b600061033361032d83866102ee565b846102ee565b94935050505056fe60e060405234801561001057600080fd5b50604051610ec8380380610ec883398101604081905261002f916100c6565b816001600160a01b03166351ed6a306040518163ffffffff1660e01b8152600401602060405180830381865afa15801561006d573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061009191906100f4565b6001600160a01b039081166080529190911660a05260c052610118565b6001600160a01b03811681146100c357600080fd5b50565b600080604083850312156100d957600080fd5b82516100e4816100ae565b6020939093015192949293505050565b60006020828403121561010657600080fd5b8151610111816100ae565b9392505050565b60805160a05160c051610d4b61017d6000396000818161015b0152818161055801526105ac0152600081816092015281816103a70152818161048401526104dc01526000818160f3015281816102670152818161033b01526104620152610d4b6000f3fe608060405234801561001057600080fd5b50600436106100885760003560e01c80637476083b1161005b5780637476083b14610115578063956501bb146101285780639cfa2a2a14610156578063bd3eec7d1461017d57600080fd5b8063023a96fe1461008d57806326c0e5c5146100d157806330fc43ed146100db57806351ed6a30146100ee575b600080fd5b6100b47f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b0390911681526020015b60405180910390f35b6100d9610190565b005b6100d96100e9366004610a40565b6101ab565b6100b47f000000000000000000000000000000000000000000000000000000000000000081565b6100d9610123366004610a40565b6102cf565b610148610136366004610a59565b60006020819052908152604090205481565b6040519081526020016100c8565b6101487f000000000000000000000000000000000000000000000000000000000000000081565b6100d961018b366004610a82565b61039b565b336000908152602081905260409020546101a9906101ab565b565b806000036101e5576040517f1f2a200500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526020819052604090205480821115610244576040517fa47b7c6500000000000000000000000000000000000000000000000000000000815233600482015260248101839052604481018290526064015b60405180910390fd5b61024e8282610aec565b33600081815260208190526040902091909155610296907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031690846105dd565b60405182815233907f8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc9060200160405180910390a25050565b80600003610309576040517f1f2a200500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526020819052604081208054839290610328908490610b05565b9091555061036390506001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016333084610686565b60405181815233907f0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc29060200160405180910390a250565b60006001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016631c1b4f3a6103d96020850185610b2e565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e084901b16815260ff9091166004820152602401602060405180830381865afa15801561042f573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906104539190610b49565b90506104a96001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000167f0000000000000000000000000000000000000000000000000000000000000000836106dd565b6040517f05fae1410000000000000000000000000000000000000000000000000000000081526000906001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016906305fae14190610511908690600401610bf8565b6020604051808303816000875af1158015610530573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105549190610b49565b90507f000000000000000000000000000000000000000000000000000000000000000081146105d8576040517f75c0811b000000000000000000000000000000000000000000000000000000008152600481018290527f0000000000000000000000000000000000000000000000000000000000000000602482015260440161023b565b505050565b6040516001600160a01b0383166024820152604481018290526105d89084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff00000000000000000000000000000000000000000000000000000000909316929092179091526107c1565b6040516001600160a01b03808516602483015283166044820152606481018290526106d79085907f23b872dd0000000000000000000000000000000000000000000000000000000090608401610622565b50505050565b6040517fdd62ed3e0000000000000000000000000000000000000000000000000000000081523060048201526001600160a01b038381166024830152600091839186169063dd62ed3e90604401602060405180830381865afa158015610747573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061076b9190610b49565b6107759190610b05565b6040516001600160a01b0385166024820152604481018290529091506106d79085907f095ea7b30000000000000000000000000000000000000000000000000000000090606401610622565b6000610816826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564815250856001600160a01b03166108a69092919063ffffffff16565b8051909150156105d857808060200190518101906108349190610c80565b6105d85760405162461bcd60e51b815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f74207375636365656400000000000000000000000000000000000000000000606482015260840161023b565b60606108b584846000856108bf565b90505b9392505050565b6060824710156109375760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c0000000000000000000000000000000000000000000000000000606482015260840161023b565b6001600160a01b0385163b61098e5760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e7472616374000000604482015260640161023b565b600080866001600160a01b031685876040516109aa9190610cc6565b60006040518083038185875af1925050503d80600081146109e7576040519150601f19603f3d011682016040523d82523d6000602084013e6109ec565b606091505b50915091506109fc828286610a07565b979650505050505050565b60608315610a165750816108b8565b825115610a265782518084602001fd5b8160405162461bcd60e51b815260040161023b9190610ce2565b600060208284031215610a5257600080fd5b5035919050565b600060208284031215610a6b57600080fd5b81356001600160a01b03811681146108b857600080fd5b600060208284031215610a9457600080fd5b813567ffffffffffffffff811115610aab57600080fd5b820160c081850312156108b857600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b81810381811115610aff57610aff610abd565b92915050565b80820180821115610aff57610aff610abd565b803560ff81168114610b2957600080fd5b919050565b600060208284031215610b4057600080fd5b6108b882610b18565b600060208284031215610b5b57600080fd5b5051919050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112610b9757600080fd5b830160208101925035905067ffffffffffffffff811115610bb757600080fd5b803603821315610bc657600080fd5b9250929050565b818352818160208501375060006020828401015260006020601f19601f840116840101905092915050565b6020815260ff610c0783610b18565b1660208201526020820135604082015260408201356060820152606082013560808201526000610c3a6080840184610b62565b60c060a0850152610c4f60e085018284610bcd565b915050610c5f60a0850185610b62565b601f198584030160c0860152610c76838284610bcd565b9695505050505050565b600060208284031215610c9257600080fd5b815180151581146108b857600080fd5b60005b83811015610cbd578181015183820152602001610ca5565b50506000910152565b60008251610cd8818460208701610ca2565b9190910192915050565b6020815260008251806020840152610d01816040850160208701610ca2565b601f01601f1916919091016040019291505056fea2646970667358221220045bdaf31a036831af5ac9e940af092e3bee03f7326979b895e5c2146fd111a764736f6c63430008110033a2646970667358221220e47674c71abbea0b747f24f2ae9050c2849f75f1d39c214e5c0ba3605b21e2b964736f6c63430008110033",
}

// EdgeStakingPoolCreatorABI is the input ABI used to generate the binding from.
// Deprecated: Use EdgeStakingPoolCreatorMetaData.ABI instead.
var EdgeStakingPoolCreatorABI = EdgeStakingPoolCreatorMetaData.ABI

// EdgeStakingPoolCreatorBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use EdgeStakingPoolCreatorMetaData.Bin instead.
var EdgeStakingPoolCreatorBin = EdgeStakingPoolCreatorMetaData.Bin

// DeployEdgeStakingPoolCreator deploys a new Ethereum contract, binding an instance of EdgeStakingPoolCreator to it.
func DeployEdgeStakingPoolCreator(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *EdgeStakingPoolCreator, error) {
	parsed, err := EdgeStakingPoolCreatorMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(EdgeStakingPoolCreatorBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &EdgeStakingPoolCreator{EdgeStakingPoolCreatorCaller: EdgeStakingPoolCreatorCaller{contract: contract}, EdgeStakingPoolCreatorTransactor: EdgeStakingPoolCreatorTransactor{contract: contract}, EdgeStakingPoolCreatorFilterer: EdgeStakingPoolCreatorFilterer{contract: contract}}, nil
}

// EdgeStakingPoolCreator is an auto generated Go binding around an Ethereum contract.
type EdgeStakingPoolCreator struct {
	EdgeStakingPoolCreatorCaller     // Read-only binding to the contract
	EdgeStakingPoolCreatorTransactor // Write-only binding to the contract
	EdgeStakingPoolCreatorFilterer   // Log filterer for contract events
}

// EdgeStakingPoolCreatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type EdgeStakingPoolCreatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeStakingPoolCreatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type EdgeStakingPoolCreatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeStakingPoolCreatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type EdgeStakingPoolCreatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeStakingPoolCreatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type EdgeStakingPoolCreatorSession struct {
	Contract     *EdgeStakingPoolCreator // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// EdgeStakingPoolCreatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type EdgeStakingPoolCreatorCallerSession struct {
	Contract *EdgeStakingPoolCreatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// EdgeStakingPoolCreatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type EdgeStakingPoolCreatorTransactorSession struct {
	Contract     *EdgeStakingPoolCreatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// EdgeStakingPoolCreatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type EdgeStakingPoolCreatorRaw struct {
	Contract *EdgeStakingPoolCreator // Generic contract binding to access the raw methods on
}

// EdgeStakingPoolCreatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type EdgeStakingPoolCreatorCallerRaw struct {
	Contract *EdgeStakingPoolCreatorCaller // Generic read-only contract binding to access the raw methods on
}

// EdgeStakingPoolCreatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type EdgeStakingPoolCreatorTransactorRaw struct {
	Contract *EdgeStakingPoolCreatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewEdgeStakingPoolCreator creates a new instance of EdgeStakingPoolCreator, bound to a specific deployed contract.
func NewEdgeStakingPoolCreator(address common.Address, backend bind.ContractBackend) (*EdgeStakingPoolCreator, error) {
	contract, err := bindEdgeStakingPoolCreator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPoolCreator{EdgeStakingPoolCreatorCaller: EdgeStakingPoolCreatorCaller{contract: contract}, EdgeStakingPoolCreatorTransactor: EdgeStakingPoolCreatorTransactor{contract: contract}, EdgeStakingPoolCreatorFilterer: EdgeStakingPoolCreatorFilterer{contract: contract}}, nil
}

// NewEdgeStakingPoolCreatorCaller creates a new read-only instance of EdgeStakingPoolCreator, bound to a specific deployed contract.
func NewEdgeStakingPoolCreatorCaller(address common.Address, caller bind.ContractCaller) (*EdgeStakingPoolCreatorCaller, error) {
	contract, err := bindEdgeStakingPoolCreator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPoolCreatorCaller{contract: contract}, nil
}

// NewEdgeStakingPoolCreatorTransactor creates a new write-only instance of EdgeStakingPoolCreator, bound to a specific deployed contract.
func NewEdgeStakingPoolCreatorTransactor(address common.Address, transactor bind.ContractTransactor) (*EdgeStakingPoolCreatorTransactor, error) {
	contract, err := bindEdgeStakingPoolCreator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPoolCreatorTransactor{contract: contract}, nil
}

// NewEdgeStakingPoolCreatorFilterer creates a new log filterer instance of EdgeStakingPoolCreator, bound to a specific deployed contract.
func NewEdgeStakingPoolCreatorFilterer(address common.Address, filterer bind.ContractFilterer) (*EdgeStakingPoolCreatorFilterer, error) {
	contract, err := bindEdgeStakingPoolCreator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPoolCreatorFilterer{contract: contract}, nil
}

// bindEdgeStakingPoolCreator binds a generic wrapper to an already deployed contract.
func bindEdgeStakingPoolCreator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := EdgeStakingPoolCreatorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EdgeStakingPoolCreator.Contract.EdgeStakingPoolCreatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EdgeStakingPoolCreator.Contract.EdgeStakingPoolCreatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EdgeStakingPoolCreator.Contract.EdgeStakingPoolCreatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EdgeStakingPoolCreator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EdgeStakingPoolCreator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EdgeStakingPoolCreator.Contract.contract.Transact(opts, method, params...)
}

// GetPool is a free data retrieval call binding the contract method 0xdc082ad3.
//
// Solidity: function getPool(address challengeManager, bytes32 edgeId) view returns(address)
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorCaller) GetPool(opts *bind.CallOpts, challengeManager common.Address, edgeId [32]byte) (common.Address, error) {
	var out []interface{}
	err := _EdgeStakingPoolCreator.contract.Call(opts, &out, "getPool", challengeManager, edgeId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetPool is a free data retrieval call binding the contract method 0xdc082ad3.
//
// Solidity: function getPool(address challengeManager, bytes32 edgeId) view returns(address)
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorSession) GetPool(challengeManager common.Address, edgeId [32]byte) (common.Address, error) {
	return _EdgeStakingPoolCreator.Contract.GetPool(&_EdgeStakingPoolCreator.CallOpts, challengeManager, edgeId)
}

// GetPool is a free data retrieval call binding the contract method 0xdc082ad3.
//
// Solidity: function getPool(address challengeManager, bytes32 edgeId) view returns(address)
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorCallerSession) GetPool(challengeManager common.Address, edgeId [32]byte) (common.Address, error) {
	return _EdgeStakingPoolCreator.Contract.GetPool(&_EdgeStakingPoolCreator.CallOpts, challengeManager, edgeId)
}

// CreatePool is a paid mutator transaction binding the contract method 0x9b505aa1.
//
// Solidity: function createPool(address challengeManager, bytes32 edgeId) returns(address)
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorTransactor) CreatePool(opts *bind.TransactOpts, challengeManager common.Address, edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeStakingPoolCreator.contract.Transact(opts, "createPool", challengeManager, edgeId)
}

// CreatePool is a paid mutator transaction binding the contract method 0x9b505aa1.
//
// Solidity: function createPool(address challengeManager, bytes32 edgeId) returns(address)
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorSession) CreatePool(challengeManager common.Address, edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeStakingPoolCreator.Contract.CreatePool(&_EdgeStakingPoolCreator.TransactOpts, challengeManager, edgeId)
}

// CreatePool is a paid mutator transaction binding the contract method 0x9b505aa1.
//
// Solidity: function createPool(address challengeManager, bytes32 edgeId) returns(address)
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorTransactorSession) CreatePool(challengeManager common.Address, edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeStakingPoolCreator.Contract.CreatePool(&_EdgeStakingPoolCreator.TransactOpts, challengeManager, edgeId)
}

// EdgeStakingPoolCreatorNewEdgeStakingPoolCreatedIterator is returned from FilterNewEdgeStakingPoolCreated and is used to iterate over the raw logs and unpacked data for NewEdgeStakingPoolCreated events raised by the EdgeStakingPoolCreator contract.
type EdgeStakingPoolCreatorNewEdgeStakingPoolCreatedIterator struct {
	Event *EdgeStakingPoolCreatorNewEdgeStakingPoolCreated // Event containing the contract specifics and raw log

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
func (it *EdgeStakingPoolCreatorNewEdgeStakingPoolCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeStakingPoolCreatorNewEdgeStakingPoolCreated)
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
		it.Event = new(EdgeStakingPoolCreatorNewEdgeStakingPoolCreated)
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
func (it *EdgeStakingPoolCreatorNewEdgeStakingPoolCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeStakingPoolCreatorNewEdgeStakingPoolCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeStakingPoolCreatorNewEdgeStakingPoolCreated represents a NewEdgeStakingPoolCreated event raised by the EdgeStakingPoolCreator contract.
type EdgeStakingPoolCreatorNewEdgeStakingPoolCreated struct {
	ChallengeManager common.Address
	EdgeId           [32]byte
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterNewEdgeStakingPoolCreated is a free log retrieval operation binding the contract event 0x15e71db3d71eb3b7985105d763101e1d6c1c491ab3e6a0d682558c12cc0bb8d6.
//
// Solidity: event NewEdgeStakingPoolCreated(address indexed challengeManager, bytes32 indexed edgeId)
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorFilterer) FilterNewEdgeStakingPoolCreated(opts *bind.FilterOpts, challengeManager []common.Address, edgeId [][32]byte) (*EdgeStakingPoolCreatorNewEdgeStakingPoolCreatedIterator, error) {

	var challengeManagerRule []interface{}
	for _, challengeManagerItem := range challengeManager {
		challengeManagerRule = append(challengeManagerRule, challengeManagerItem)
	}
	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}

	logs, sub, err := _EdgeStakingPoolCreator.contract.FilterLogs(opts, "NewEdgeStakingPoolCreated", challengeManagerRule, edgeIdRule)
	if err != nil {
		return nil, err
	}
	return &EdgeStakingPoolCreatorNewEdgeStakingPoolCreatedIterator{contract: _EdgeStakingPoolCreator.contract, event: "NewEdgeStakingPoolCreated", logs: logs, sub: sub}, nil
}

// WatchNewEdgeStakingPoolCreated is a free log subscription operation binding the contract event 0x15e71db3d71eb3b7985105d763101e1d6c1c491ab3e6a0d682558c12cc0bb8d6.
//
// Solidity: event NewEdgeStakingPoolCreated(address indexed challengeManager, bytes32 indexed edgeId)
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorFilterer) WatchNewEdgeStakingPoolCreated(opts *bind.WatchOpts, sink chan<- *EdgeStakingPoolCreatorNewEdgeStakingPoolCreated, challengeManager []common.Address, edgeId [][32]byte) (event.Subscription, error) {

	var challengeManagerRule []interface{}
	for _, challengeManagerItem := range challengeManager {
		challengeManagerRule = append(challengeManagerRule, challengeManagerItem)
	}
	var edgeIdRule []interface{}
	for _, edgeIdItem := range edgeId {
		edgeIdRule = append(edgeIdRule, edgeIdItem)
	}

	logs, sub, err := _EdgeStakingPoolCreator.contract.WatchLogs(opts, "NewEdgeStakingPoolCreated", challengeManagerRule, edgeIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeStakingPoolCreatorNewEdgeStakingPoolCreated)
				if err := _EdgeStakingPoolCreator.contract.UnpackLog(event, "NewEdgeStakingPoolCreated", log); err != nil {
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

// ParseNewEdgeStakingPoolCreated is a log parse operation binding the contract event 0x15e71db3d71eb3b7985105d763101e1d6c1c491ab3e6a0d682558c12cc0bb8d6.
//
// Solidity: event NewEdgeStakingPoolCreated(address indexed challengeManager, bytes32 indexed edgeId)
func (_EdgeStakingPoolCreator *EdgeStakingPoolCreatorFilterer) ParseNewEdgeStakingPoolCreated(log types.Log) (*EdgeStakingPoolCreatorNewEdgeStakingPoolCreated, error) {
	event := new(EdgeStakingPoolCreatorNewEdgeStakingPoolCreated)
	if err := _EdgeStakingPoolCreator.contract.UnpackLog(event, "NewEdgeStakingPoolCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakingPoolCreatorUtilsMetaData contains all meta data concerning the StakingPoolCreatorUtils contract.
var StakingPoolCreatorUtilsMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"PoolDoesntExist\",\"type\":\"error\"}]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea264697066735822122057f86fed56cc6470e19ac9a3135a9794f9cd5bd5a657dc4bb3706094100337a364736f6c63430008110033",
}

// StakingPoolCreatorUtilsABI is the input ABI used to generate the binding from.
// Deprecated: Use StakingPoolCreatorUtilsMetaData.ABI instead.
var StakingPoolCreatorUtilsABI = StakingPoolCreatorUtilsMetaData.ABI

// StakingPoolCreatorUtilsBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use StakingPoolCreatorUtilsMetaData.Bin instead.
var StakingPoolCreatorUtilsBin = StakingPoolCreatorUtilsMetaData.Bin

// DeployStakingPoolCreatorUtils deploys a new Ethereum contract, binding an instance of StakingPoolCreatorUtils to it.
func DeployStakingPoolCreatorUtils(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *StakingPoolCreatorUtils, error) {
	parsed, err := StakingPoolCreatorUtilsMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(StakingPoolCreatorUtilsBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &StakingPoolCreatorUtils{StakingPoolCreatorUtilsCaller: StakingPoolCreatorUtilsCaller{contract: contract}, StakingPoolCreatorUtilsTransactor: StakingPoolCreatorUtilsTransactor{contract: contract}, StakingPoolCreatorUtilsFilterer: StakingPoolCreatorUtilsFilterer{contract: contract}}, nil
}

// StakingPoolCreatorUtils is an auto generated Go binding around an Ethereum contract.
type StakingPoolCreatorUtils struct {
	StakingPoolCreatorUtilsCaller     // Read-only binding to the contract
	StakingPoolCreatorUtilsTransactor // Write-only binding to the contract
	StakingPoolCreatorUtilsFilterer   // Log filterer for contract events
}

// StakingPoolCreatorUtilsCaller is an auto generated read-only Go binding around an Ethereum contract.
type StakingPoolCreatorUtilsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakingPoolCreatorUtilsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StakingPoolCreatorUtilsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakingPoolCreatorUtilsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StakingPoolCreatorUtilsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakingPoolCreatorUtilsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StakingPoolCreatorUtilsSession struct {
	Contract     *StakingPoolCreatorUtils // Generic contract binding to set the session for
	CallOpts     bind.CallOpts            // Call options to use throughout this session
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// StakingPoolCreatorUtilsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StakingPoolCreatorUtilsCallerSession struct {
	Contract *StakingPoolCreatorUtilsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                  // Call options to use throughout this session
}

// StakingPoolCreatorUtilsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StakingPoolCreatorUtilsTransactorSession struct {
	Contract     *StakingPoolCreatorUtilsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                  // Transaction auth options to use throughout this session
}

// StakingPoolCreatorUtilsRaw is an auto generated low-level Go binding around an Ethereum contract.
type StakingPoolCreatorUtilsRaw struct {
	Contract *StakingPoolCreatorUtils // Generic contract binding to access the raw methods on
}

// StakingPoolCreatorUtilsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StakingPoolCreatorUtilsCallerRaw struct {
	Contract *StakingPoolCreatorUtilsCaller // Generic read-only contract binding to access the raw methods on
}

// StakingPoolCreatorUtilsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StakingPoolCreatorUtilsTransactorRaw struct {
	Contract *StakingPoolCreatorUtilsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStakingPoolCreatorUtils creates a new instance of StakingPoolCreatorUtils, bound to a specific deployed contract.
func NewStakingPoolCreatorUtils(address common.Address, backend bind.ContractBackend) (*StakingPoolCreatorUtils, error) {
	contract, err := bindStakingPoolCreatorUtils(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &StakingPoolCreatorUtils{StakingPoolCreatorUtilsCaller: StakingPoolCreatorUtilsCaller{contract: contract}, StakingPoolCreatorUtilsTransactor: StakingPoolCreatorUtilsTransactor{contract: contract}, StakingPoolCreatorUtilsFilterer: StakingPoolCreatorUtilsFilterer{contract: contract}}, nil
}

// NewStakingPoolCreatorUtilsCaller creates a new read-only instance of StakingPoolCreatorUtils, bound to a specific deployed contract.
func NewStakingPoolCreatorUtilsCaller(address common.Address, caller bind.ContractCaller) (*StakingPoolCreatorUtilsCaller, error) {
	contract, err := bindStakingPoolCreatorUtils(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StakingPoolCreatorUtilsCaller{contract: contract}, nil
}

// NewStakingPoolCreatorUtilsTransactor creates a new write-only instance of StakingPoolCreatorUtils, bound to a specific deployed contract.
func NewStakingPoolCreatorUtilsTransactor(address common.Address, transactor bind.ContractTransactor) (*StakingPoolCreatorUtilsTransactor, error) {
	contract, err := bindStakingPoolCreatorUtils(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StakingPoolCreatorUtilsTransactor{contract: contract}, nil
}

// NewStakingPoolCreatorUtilsFilterer creates a new log filterer instance of StakingPoolCreatorUtils, bound to a specific deployed contract.
func NewStakingPoolCreatorUtilsFilterer(address common.Address, filterer bind.ContractFilterer) (*StakingPoolCreatorUtilsFilterer, error) {
	contract, err := bindStakingPoolCreatorUtils(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StakingPoolCreatorUtilsFilterer{contract: contract}, nil
}

// bindStakingPoolCreatorUtils binds a generic wrapper to an already deployed contract.
func bindStakingPoolCreatorUtils(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StakingPoolCreatorUtilsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StakingPoolCreatorUtils *StakingPoolCreatorUtilsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StakingPoolCreatorUtils.Contract.StakingPoolCreatorUtilsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StakingPoolCreatorUtils *StakingPoolCreatorUtilsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StakingPoolCreatorUtils.Contract.StakingPoolCreatorUtilsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StakingPoolCreatorUtils *StakingPoolCreatorUtilsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StakingPoolCreatorUtils.Contract.StakingPoolCreatorUtilsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StakingPoolCreatorUtils *StakingPoolCreatorUtilsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StakingPoolCreatorUtils.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StakingPoolCreatorUtils *StakingPoolCreatorUtilsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StakingPoolCreatorUtils.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StakingPoolCreatorUtils *StakingPoolCreatorUtilsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StakingPoolCreatorUtils.Contract.contract.Transact(opts, method, params...)
}
