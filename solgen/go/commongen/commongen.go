// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package commongen

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

// NativeCurrencyPaymentFallbackMetaData contains all meta data concerning the NativeCurrencyPaymentFallback contract.
var NativeCurrencyPaymentFallbackMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"SafeReceived\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
}

// NativeCurrencyPaymentFallbackABI is the input ABI used to generate the binding from.
// Deprecated: Use NativeCurrencyPaymentFallbackMetaData.ABI instead.
var NativeCurrencyPaymentFallbackABI = NativeCurrencyPaymentFallbackMetaData.ABI

// NativeCurrencyPaymentFallback is an auto generated Go binding around an Ethereum contract.
type NativeCurrencyPaymentFallback struct {
	NativeCurrencyPaymentFallbackCaller     // Read-only binding to the contract
	NativeCurrencyPaymentFallbackTransactor // Write-only binding to the contract
	NativeCurrencyPaymentFallbackFilterer   // Log filterer for contract events
}

// NativeCurrencyPaymentFallbackCaller is an auto generated read-only Go binding around an Ethereum contract.
type NativeCurrencyPaymentFallbackCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NativeCurrencyPaymentFallbackTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NativeCurrencyPaymentFallbackTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NativeCurrencyPaymentFallbackFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NativeCurrencyPaymentFallbackFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NativeCurrencyPaymentFallbackSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NativeCurrencyPaymentFallbackSession struct {
	Contract     *NativeCurrencyPaymentFallback // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                  // Call options to use throughout this session
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// NativeCurrencyPaymentFallbackCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NativeCurrencyPaymentFallbackCallerSession struct {
	Contract *NativeCurrencyPaymentFallbackCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                        // Call options to use throughout this session
}

// NativeCurrencyPaymentFallbackTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NativeCurrencyPaymentFallbackTransactorSession struct {
	Contract     *NativeCurrencyPaymentFallbackTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                        // Transaction auth options to use throughout this session
}

// NativeCurrencyPaymentFallbackRaw is an auto generated low-level Go binding around an Ethereum contract.
type NativeCurrencyPaymentFallbackRaw struct {
	Contract *NativeCurrencyPaymentFallback // Generic contract binding to access the raw methods on
}

// NativeCurrencyPaymentFallbackCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NativeCurrencyPaymentFallbackCallerRaw struct {
	Contract *NativeCurrencyPaymentFallbackCaller // Generic read-only contract binding to access the raw methods on
}

// NativeCurrencyPaymentFallbackTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NativeCurrencyPaymentFallbackTransactorRaw struct {
	Contract *NativeCurrencyPaymentFallbackTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNativeCurrencyPaymentFallback creates a new instance of NativeCurrencyPaymentFallback, bound to a specific deployed contract.
func NewNativeCurrencyPaymentFallback(address common.Address, backend bind.ContractBackend) (*NativeCurrencyPaymentFallback, error) {
	contract, err := bindNativeCurrencyPaymentFallback(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NativeCurrencyPaymentFallback{NativeCurrencyPaymentFallbackCaller: NativeCurrencyPaymentFallbackCaller{contract: contract}, NativeCurrencyPaymentFallbackTransactor: NativeCurrencyPaymentFallbackTransactor{contract: contract}, NativeCurrencyPaymentFallbackFilterer: NativeCurrencyPaymentFallbackFilterer{contract: contract}}, nil
}

// NewNativeCurrencyPaymentFallbackCaller creates a new read-only instance of NativeCurrencyPaymentFallback, bound to a specific deployed contract.
func NewNativeCurrencyPaymentFallbackCaller(address common.Address, caller bind.ContractCaller) (*NativeCurrencyPaymentFallbackCaller, error) {
	contract, err := bindNativeCurrencyPaymentFallback(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NativeCurrencyPaymentFallbackCaller{contract: contract}, nil
}

// NewNativeCurrencyPaymentFallbackTransactor creates a new write-only instance of NativeCurrencyPaymentFallback, bound to a specific deployed contract.
func NewNativeCurrencyPaymentFallbackTransactor(address common.Address, transactor bind.ContractTransactor) (*NativeCurrencyPaymentFallbackTransactor, error) {
	contract, err := bindNativeCurrencyPaymentFallback(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NativeCurrencyPaymentFallbackTransactor{contract: contract}, nil
}

// NewNativeCurrencyPaymentFallbackFilterer creates a new log filterer instance of NativeCurrencyPaymentFallback, bound to a specific deployed contract.
func NewNativeCurrencyPaymentFallbackFilterer(address common.Address, filterer bind.ContractFilterer) (*NativeCurrencyPaymentFallbackFilterer, error) {
	contract, err := bindNativeCurrencyPaymentFallback(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NativeCurrencyPaymentFallbackFilterer{contract: contract}, nil
}

// bindNativeCurrencyPaymentFallback binds a generic wrapper to an already deployed contract.
func bindNativeCurrencyPaymentFallback(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NativeCurrencyPaymentFallbackMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NativeCurrencyPaymentFallback.Contract.NativeCurrencyPaymentFallbackCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NativeCurrencyPaymentFallback.Contract.NativeCurrencyPaymentFallbackTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NativeCurrencyPaymentFallback.Contract.NativeCurrencyPaymentFallbackTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NativeCurrencyPaymentFallback.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NativeCurrencyPaymentFallback.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NativeCurrencyPaymentFallback.Contract.contract.Transact(opts, method, params...)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NativeCurrencyPaymentFallback.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackSession) Receive() (*types.Transaction, error) {
	return _NativeCurrencyPaymentFallback.Contract.Receive(&_NativeCurrencyPaymentFallback.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackTransactorSession) Receive() (*types.Transaction, error) {
	return _NativeCurrencyPaymentFallback.Contract.Receive(&_NativeCurrencyPaymentFallback.TransactOpts)
}

// NativeCurrencyPaymentFallbackSafeReceivedIterator is returned from FilterSafeReceived and is used to iterate over the raw logs and unpacked data for SafeReceived events raised by the NativeCurrencyPaymentFallback contract.
type NativeCurrencyPaymentFallbackSafeReceivedIterator struct {
	Event *NativeCurrencyPaymentFallbackSafeReceived // Event containing the contract specifics and raw log

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
func (it *NativeCurrencyPaymentFallbackSafeReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NativeCurrencyPaymentFallbackSafeReceived)
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
		it.Event = new(NativeCurrencyPaymentFallbackSafeReceived)
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
func (it *NativeCurrencyPaymentFallbackSafeReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NativeCurrencyPaymentFallbackSafeReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NativeCurrencyPaymentFallbackSafeReceived represents a SafeReceived event raised by the NativeCurrencyPaymentFallback contract.
type NativeCurrencyPaymentFallbackSafeReceived struct {
	Sender common.Address
	Value  *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterSafeReceived is a free log retrieval operation binding the contract event 0x3d0ce9bfc3ed7d6862dbb28b2dea94561fe714a1b4d019aa8af39730d1ad7c3d.
//
// Solidity: event SafeReceived(address indexed sender, uint256 value)
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackFilterer) FilterSafeReceived(opts *bind.FilterOpts, sender []common.Address) (*NativeCurrencyPaymentFallbackSafeReceivedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NativeCurrencyPaymentFallback.contract.FilterLogs(opts, "SafeReceived", senderRule)
	if err != nil {
		return nil, err
	}
	return &NativeCurrencyPaymentFallbackSafeReceivedIterator{contract: _NativeCurrencyPaymentFallback.contract, event: "SafeReceived", logs: logs, sub: sub}, nil
}

// WatchSafeReceived is a free log subscription operation binding the contract event 0x3d0ce9bfc3ed7d6862dbb28b2dea94561fe714a1b4d019aa8af39730d1ad7c3d.
//
// Solidity: event SafeReceived(address indexed sender, uint256 value)
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackFilterer) WatchSafeReceived(opts *bind.WatchOpts, sink chan<- *NativeCurrencyPaymentFallbackSafeReceived, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _NativeCurrencyPaymentFallback.contract.WatchLogs(opts, "SafeReceived", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NativeCurrencyPaymentFallbackSafeReceived)
				if err := _NativeCurrencyPaymentFallback.contract.UnpackLog(event, "SafeReceived", log); err != nil {
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

// ParseSafeReceived is a log parse operation binding the contract event 0x3d0ce9bfc3ed7d6862dbb28b2dea94561fe714a1b4d019aa8af39730d1ad7c3d.
//
// Solidity: event SafeReceived(address indexed sender, uint256 value)
func (_NativeCurrencyPaymentFallback *NativeCurrencyPaymentFallbackFilterer) ParseSafeReceived(log types.Log) (*NativeCurrencyPaymentFallbackSafeReceived, error) {
	event := new(NativeCurrencyPaymentFallbackSafeReceived)
	if err := _NativeCurrencyPaymentFallback.contract.UnpackLog(event, "SafeReceived", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SecuredTokenTransferMetaData contains all meta data concerning the SecuredTokenTransfer contract.
var SecuredTokenTransferMetaData = &bind.MetaData{
	ABI: "[]",
}

// SecuredTokenTransferABI is the input ABI used to generate the binding from.
// Deprecated: Use SecuredTokenTransferMetaData.ABI instead.
var SecuredTokenTransferABI = SecuredTokenTransferMetaData.ABI

// SecuredTokenTransfer is an auto generated Go binding around an Ethereum contract.
type SecuredTokenTransfer struct {
	SecuredTokenTransferCaller     // Read-only binding to the contract
	SecuredTokenTransferTransactor // Write-only binding to the contract
	SecuredTokenTransferFilterer   // Log filterer for contract events
}

// SecuredTokenTransferCaller is an auto generated read-only Go binding around an Ethereum contract.
type SecuredTokenTransferCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SecuredTokenTransferTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SecuredTokenTransferTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SecuredTokenTransferFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SecuredTokenTransferFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SecuredTokenTransferSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SecuredTokenTransferSession struct {
	Contract     *SecuredTokenTransfer // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// SecuredTokenTransferCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SecuredTokenTransferCallerSession struct {
	Contract *SecuredTokenTransferCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// SecuredTokenTransferTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SecuredTokenTransferTransactorSession struct {
	Contract     *SecuredTokenTransferTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// SecuredTokenTransferRaw is an auto generated low-level Go binding around an Ethereum contract.
type SecuredTokenTransferRaw struct {
	Contract *SecuredTokenTransfer // Generic contract binding to access the raw methods on
}

// SecuredTokenTransferCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SecuredTokenTransferCallerRaw struct {
	Contract *SecuredTokenTransferCaller // Generic read-only contract binding to access the raw methods on
}

// SecuredTokenTransferTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SecuredTokenTransferTransactorRaw struct {
	Contract *SecuredTokenTransferTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSecuredTokenTransfer creates a new instance of SecuredTokenTransfer, bound to a specific deployed contract.
func NewSecuredTokenTransfer(address common.Address, backend bind.ContractBackend) (*SecuredTokenTransfer, error) {
	contract, err := bindSecuredTokenTransfer(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SecuredTokenTransfer{SecuredTokenTransferCaller: SecuredTokenTransferCaller{contract: contract}, SecuredTokenTransferTransactor: SecuredTokenTransferTransactor{contract: contract}, SecuredTokenTransferFilterer: SecuredTokenTransferFilterer{contract: contract}}, nil
}

// NewSecuredTokenTransferCaller creates a new read-only instance of SecuredTokenTransfer, bound to a specific deployed contract.
func NewSecuredTokenTransferCaller(address common.Address, caller bind.ContractCaller) (*SecuredTokenTransferCaller, error) {
	contract, err := bindSecuredTokenTransfer(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SecuredTokenTransferCaller{contract: contract}, nil
}

// NewSecuredTokenTransferTransactor creates a new write-only instance of SecuredTokenTransfer, bound to a specific deployed contract.
func NewSecuredTokenTransferTransactor(address common.Address, transactor bind.ContractTransactor) (*SecuredTokenTransferTransactor, error) {
	contract, err := bindSecuredTokenTransfer(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SecuredTokenTransferTransactor{contract: contract}, nil
}

// NewSecuredTokenTransferFilterer creates a new log filterer instance of SecuredTokenTransfer, bound to a specific deployed contract.
func NewSecuredTokenTransferFilterer(address common.Address, filterer bind.ContractFilterer) (*SecuredTokenTransferFilterer, error) {
	contract, err := bindSecuredTokenTransfer(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SecuredTokenTransferFilterer{contract: contract}, nil
}

// bindSecuredTokenTransfer binds a generic wrapper to an already deployed contract.
func bindSecuredTokenTransfer(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SecuredTokenTransferMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SecuredTokenTransfer *SecuredTokenTransferRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SecuredTokenTransfer.Contract.SecuredTokenTransferCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SecuredTokenTransfer *SecuredTokenTransferRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SecuredTokenTransfer.Contract.SecuredTokenTransferTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SecuredTokenTransfer *SecuredTokenTransferRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SecuredTokenTransfer.Contract.SecuredTokenTransferTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SecuredTokenTransfer *SecuredTokenTransferCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SecuredTokenTransfer.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SecuredTokenTransfer *SecuredTokenTransferTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SecuredTokenTransfer.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SecuredTokenTransfer *SecuredTokenTransferTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SecuredTokenTransfer.Contract.contract.Transact(opts, method, params...)
}

// SelfAuthorizedMetaData contains all meta data concerning the SelfAuthorized contract.
var SelfAuthorizedMetaData = &bind.MetaData{
	ABI: "[]",
}

// SelfAuthorizedABI is the input ABI used to generate the binding from.
// Deprecated: Use SelfAuthorizedMetaData.ABI instead.
var SelfAuthorizedABI = SelfAuthorizedMetaData.ABI

// SelfAuthorized is an auto generated Go binding around an Ethereum contract.
type SelfAuthorized struct {
	SelfAuthorizedCaller     // Read-only binding to the contract
	SelfAuthorizedTransactor // Write-only binding to the contract
	SelfAuthorizedFilterer   // Log filterer for contract events
}

// SelfAuthorizedCaller is an auto generated read-only Go binding around an Ethereum contract.
type SelfAuthorizedCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SelfAuthorizedTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SelfAuthorizedTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SelfAuthorizedFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SelfAuthorizedFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SelfAuthorizedSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SelfAuthorizedSession struct {
	Contract     *SelfAuthorized   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SelfAuthorizedCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SelfAuthorizedCallerSession struct {
	Contract *SelfAuthorizedCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// SelfAuthorizedTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SelfAuthorizedTransactorSession struct {
	Contract     *SelfAuthorizedTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// SelfAuthorizedRaw is an auto generated low-level Go binding around an Ethereum contract.
type SelfAuthorizedRaw struct {
	Contract *SelfAuthorized // Generic contract binding to access the raw methods on
}

// SelfAuthorizedCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SelfAuthorizedCallerRaw struct {
	Contract *SelfAuthorizedCaller // Generic read-only contract binding to access the raw methods on
}

// SelfAuthorizedTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SelfAuthorizedTransactorRaw struct {
	Contract *SelfAuthorizedTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSelfAuthorized creates a new instance of SelfAuthorized, bound to a specific deployed contract.
func NewSelfAuthorized(address common.Address, backend bind.ContractBackend) (*SelfAuthorized, error) {
	contract, err := bindSelfAuthorized(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SelfAuthorized{SelfAuthorizedCaller: SelfAuthorizedCaller{contract: contract}, SelfAuthorizedTransactor: SelfAuthorizedTransactor{contract: contract}, SelfAuthorizedFilterer: SelfAuthorizedFilterer{contract: contract}}, nil
}

// NewSelfAuthorizedCaller creates a new read-only instance of SelfAuthorized, bound to a specific deployed contract.
func NewSelfAuthorizedCaller(address common.Address, caller bind.ContractCaller) (*SelfAuthorizedCaller, error) {
	contract, err := bindSelfAuthorized(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SelfAuthorizedCaller{contract: contract}, nil
}

// NewSelfAuthorizedTransactor creates a new write-only instance of SelfAuthorized, bound to a specific deployed contract.
func NewSelfAuthorizedTransactor(address common.Address, transactor bind.ContractTransactor) (*SelfAuthorizedTransactor, error) {
	contract, err := bindSelfAuthorized(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SelfAuthorizedTransactor{contract: contract}, nil
}

// NewSelfAuthorizedFilterer creates a new log filterer instance of SelfAuthorized, bound to a specific deployed contract.
func NewSelfAuthorizedFilterer(address common.Address, filterer bind.ContractFilterer) (*SelfAuthorizedFilterer, error) {
	contract, err := bindSelfAuthorized(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SelfAuthorizedFilterer{contract: contract}, nil
}

// bindSelfAuthorized binds a generic wrapper to an already deployed contract.
func bindSelfAuthorized(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SelfAuthorizedMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SelfAuthorized *SelfAuthorizedRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SelfAuthorized.Contract.SelfAuthorizedCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SelfAuthorized *SelfAuthorizedRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SelfAuthorized.Contract.SelfAuthorizedTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SelfAuthorized *SelfAuthorizedRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SelfAuthorized.Contract.SelfAuthorizedTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SelfAuthorized *SelfAuthorizedCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SelfAuthorized.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SelfAuthorized *SelfAuthorizedTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SelfAuthorized.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SelfAuthorized *SelfAuthorizedTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SelfAuthorized.Contract.contract.Transact(opts, method, params...)
}

// SignatureDecoderMetaData contains all meta data concerning the SignatureDecoder contract.
var SignatureDecoderMetaData = &bind.MetaData{
	ABI: "[]",
}

// SignatureDecoderABI is the input ABI used to generate the binding from.
// Deprecated: Use SignatureDecoderMetaData.ABI instead.
var SignatureDecoderABI = SignatureDecoderMetaData.ABI

// SignatureDecoder is an auto generated Go binding around an Ethereum contract.
type SignatureDecoder struct {
	SignatureDecoderCaller     // Read-only binding to the contract
	SignatureDecoderTransactor // Write-only binding to the contract
	SignatureDecoderFilterer   // Log filterer for contract events
}

// SignatureDecoderCaller is an auto generated read-only Go binding around an Ethereum contract.
type SignatureDecoderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SignatureDecoderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SignatureDecoderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SignatureDecoderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SignatureDecoderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SignatureDecoderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SignatureDecoderSession struct {
	Contract     *SignatureDecoder // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SignatureDecoderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SignatureDecoderCallerSession struct {
	Contract *SignatureDecoderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// SignatureDecoderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SignatureDecoderTransactorSession struct {
	Contract     *SignatureDecoderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// SignatureDecoderRaw is an auto generated low-level Go binding around an Ethereum contract.
type SignatureDecoderRaw struct {
	Contract *SignatureDecoder // Generic contract binding to access the raw methods on
}

// SignatureDecoderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SignatureDecoderCallerRaw struct {
	Contract *SignatureDecoderCaller // Generic read-only contract binding to access the raw methods on
}

// SignatureDecoderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SignatureDecoderTransactorRaw struct {
	Contract *SignatureDecoderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSignatureDecoder creates a new instance of SignatureDecoder, bound to a specific deployed contract.
func NewSignatureDecoder(address common.Address, backend bind.ContractBackend) (*SignatureDecoder, error) {
	contract, err := bindSignatureDecoder(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SignatureDecoder{SignatureDecoderCaller: SignatureDecoderCaller{contract: contract}, SignatureDecoderTransactor: SignatureDecoderTransactor{contract: contract}, SignatureDecoderFilterer: SignatureDecoderFilterer{contract: contract}}, nil
}

// NewSignatureDecoderCaller creates a new read-only instance of SignatureDecoder, bound to a specific deployed contract.
func NewSignatureDecoderCaller(address common.Address, caller bind.ContractCaller) (*SignatureDecoderCaller, error) {
	contract, err := bindSignatureDecoder(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SignatureDecoderCaller{contract: contract}, nil
}

// NewSignatureDecoderTransactor creates a new write-only instance of SignatureDecoder, bound to a specific deployed contract.
func NewSignatureDecoderTransactor(address common.Address, transactor bind.ContractTransactor) (*SignatureDecoderTransactor, error) {
	contract, err := bindSignatureDecoder(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SignatureDecoderTransactor{contract: contract}, nil
}

// NewSignatureDecoderFilterer creates a new log filterer instance of SignatureDecoder, bound to a specific deployed contract.
func NewSignatureDecoderFilterer(address common.Address, filterer bind.ContractFilterer) (*SignatureDecoderFilterer, error) {
	contract, err := bindSignatureDecoder(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SignatureDecoderFilterer{contract: contract}, nil
}

// bindSignatureDecoder binds a generic wrapper to an already deployed contract.
func bindSignatureDecoder(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SignatureDecoderMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SignatureDecoder *SignatureDecoderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SignatureDecoder.Contract.SignatureDecoderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SignatureDecoder *SignatureDecoderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SignatureDecoder.Contract.SignatureDecoderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SignatureDecoder *SignatureDecoderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SignatureDecoder.Contract.SignatureDecoderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SignatureDecoder *SignatureDecoderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SignatureDecoder.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SignatureDecoder *SignatureDecoderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SignatureDecoder.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SignatureDecoder *SignatureDecoderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SignatureDecoder.Contract.contract.Transact(opts, method, params...)
}

// SingletonMetaData contains all meta data concerning the Singleton contract.
var SingletonMetaData = &bind.MetaData{
	ABI: "[]",
}

// SingletonABI is the input ABI used to generate the binding from.
// Deprecated: Use SingletonMetaData.ABI instead.
var SingletonABI = SingletonMetaData.ABI

// Singleton is an auto generated Go binding around an Ethereum contract.
type Singleton struct {
	SingletonCaller     // Read-only binding to the contract
	SingletonTransactor // Write-only binding to the contract
	SingletonFilterer   // Log filterer for contract events
}

// SingletonCaller is an auto generated read-only Go binding around an Ethereum contract.
type SingletonCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SingletonTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SingletonTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SingletonFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SingletonFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SingletonSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SingletonSession struct {
	Contract     *Singleton        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SingletonCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SingletonCallerSession struct {
	Contract *SingletonCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// SingletonTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SingletonTransactorSession struct {
	Contract     *SingletonTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// SingletonRaw is an auto generated low-level Go binding around an Ethereum contract.
type SingletonRaw struct {
	Contract *Singleton // Generic contract binding to access the raw methods on
}

// SingletonCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SingletonCallerRaw struct {
	Contract *SingletonCaller // Generic read-only contract binding to access the raw methods on
}

// SingletonTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SingletonTransactorRaw struct {
	Contract *SingletonTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSingleton creates a new instance of Singleton, bound to a specific deployed contract.
func NewSingleton(address common.Address, backend bind.ContractBackend) (*Singleton, error) {
	contract, err := bindSingleton(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Singleton{SingletonCaller: SingletonCaller{contract: contract}, SingletonTransactor: SingletonTransactor{contract: contract}, SingletonFilterer: SingletonFilterer{contract: contract}}, nil
}

// NewSingletonCaller creates a new read-only instance of Singleton, bound to a specific deployed contract.
func NewSingletonCaller(address common.Address, caller bind.ContractCaller) (*SingletonCaller, error) {
	contract, err := bindSingleton(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SingletonCaller{contract: contract}, nil
}

// NewSingletonTransactor creates a new write-only instance of Singleton, bound to a specific deployed contract.
func NewSingletonTransactor(address common.Address, transactor bind.ContractTransactor) (*SingletonTransactor, error) {
	contract, err := bindSingleton(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SingletonTransactor{contract: contract}, nil
}

// NewSingletonFilterer creates a new log filterer instance of Singleton, bound to a specific deployed contract.
func NewSingletonFilterer(address common.Address, filterer bind.ContractFilterer) (*SingletonFilterer, error) {
	contract, err := bindSingleton(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SingletonFilterer{contract: contract}, nil
}

// bindSingleton binds a generic wrapper to an already deployed contract.
func bindSingleton(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SingletonMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Singleton *SingletonRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Singleton.Contract.SingletonCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Singleton *SingletonRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Singleton.Contract.SingletonTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Singleton *SingletonRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Singleton.Contract.SingletonTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Singleton *SingletonCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Singleton.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Singleton *SingletonTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Singleton.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Singleton *SingletonTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Singleton.Contract.contract.Transact(opts, method, params...)
}

// StorageAccessibleMetaData contains all meta data concerning the StorageAccessible contract.
var StorageAccessibleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"offset\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"}],\"name\":\"getStorageAt\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"targetContract\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"calldataPayload\",\"type\":\"bytes\"}],\"name\":\"simulateAndRevert\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// StorageAccessibleABI is the input ABI used to generate the binding from.
// Deprecated: Use StorageAccessibleMetaData.ABI instead.
var StorageAccessibleABI = StorageAccessibleMetaData.ABI

// StorageAccessible is an auto generated Go binding around an Ethereum contract.
type StorageAccessible struct {
	StorageAccessibleCaller     // Read-only binding to the contract
	StorageAccessibleTransactor // Write-only binding to the contract
	StorageAccessibleFilterer   // Log filterer for contract events
}

// StorageAccessibleCaller is an auto generated read-only Go binding around an Ethereum contract.
type StorageAccessibleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageAccessibleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StorageAccessibleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageAccessibleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StorageAccessibleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageAccessibleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StorageAccessibleSession struct {
	Contract     *StorageAccessible // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// StorageAccessibleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StorageAccessibleCallerSession struct {
	Contract *StorageAccessibleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// StorageAccessibleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StorageAccessibleTransactorSession struct {
	Contract     *StorageAccessibleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// StorageAccessibleRaw is an auto generated low-level Go binding around an Ethereum contract.
type StorageAccessibleRaw struct {
	Contract *StorageAccessible // Generic contract binding to access the raw methods on
}

// StorageAccessibleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StorageAccessibleCallerRaw struct {
	Contract *StorageAccessibleCaller // Generic read-only contract binding to access the raw methods on
}

// StorageAccessibleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StorageAccessibleTransactorRaw struct {
	Contract *StorageAccessibleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStorageAccessible creates a new instance of StorageAccessible, bound to a specific deployed contract.
func NewStorageAccessible(address common.Address, backend bind.ContractBackend) (*StorageAccessible, error) {
	contract, err := bindStorageAccessible(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &StorageAccessible{StorageAccessibleCaller: StorageAccessibleCaller{contract: contract}, StorageAccessibleTransactor: StorageAccessibleTransactor{contract: contract}, StorageAccessibleFilterer: StorageAccessibleFilterer{contract: contract}}, nil
}

// NewStorageAccessibleCaller creates a new read-only instance of StorageAccessible, bound to a specific deployed contract.
func NewStorageAccessibleCaller(address common.Address, caller bind.ContractCaller) (*StorageAccessibleCaller, error) {
	contract, err := bindStorageAccessible(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StorageAccessibleCaller{contract: contract}, nil
}

// NewStorageAccessibleTransactor creates a new write-only instance of StorageAccessible, bound to a specific deployed contract.
func NewStorageAccessibleTransactor(address common.Address, transactor bind.ContractTransactor) (*StorageAccessibleTransactor, error) {
	contract, err := bindStorageAccessible(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StorageAccessibleTransactor{contract: contract}, nil
}

// NewStorageAccessibleFilterer creates a new log filterer instance of StorageAccessible, bound to a specific deployed contract.
func NewStorageAccessibleFilterer(address common.Address, filterer bind.ContractFilterer) (*StorageAccessibleFilterer, error) {
	contract, err := bindStorageAccessible(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StorageAccessibleFilterer{contract: contract}, nil
}

// bindStorageAccessible binds a generic wrapper to an already deployed contract.
func bindStorageAccessible(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StorageAccessibleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StorageAccessible *StorageAccessibleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StorageAccessible.Contract.StorageAccessibleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StorageAccessible *StorageAccessibleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StorageAccessible.Contract.StorageAccessibleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StorageAccessible *StorageAccessibleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StorageAccessible.Contract.StorageAccessibleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StorageAccessible *StorageAccessibleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StorageAccessible.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StorageAccessible *StorageAccessibleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StorageAccessible.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StorageAccessible *StorageAccessibleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StorageAccessible.Contract.contract.Transact(opts, method, params...)
}

// GetStorageAt is a free data retrieval call binding the contract method 0x5624b25b.
//
// Solidity: function getStorageAt(uint256 offset, uint256 length) view returns(bytes)
func (_StorageAccessible *StorageAccessibleCaller) GetStorageAt(opts *bind.CallOpts, offset *big.Int, length *big.Int) ([]byte, error) {
	var out []interface{}
	err := _StorageAccessible.contract.Call(opts, &out, "getStorageAt", offset, length)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetStorageAt is a free data retrieval call binding the contract method 0x5624b25b.
//
// Solidity: function getStorageAt(uint256 offset, uint256 length) view returns(bytes)
func (_StorageAccessible *StorageAccessibleSession) GetStorageAt(offset *big.Int, length *big.Int) ([]byte, error) {
	return _StorageAccessible.Contract.GetStorageAt(&_StorageAccessible.CallOpts, offset, length)
}

// GetStorageAt is a free data retrieval call binding the contract method 0x5624b25b.
//
// Solidity: function getStorageAt(uint256 offset, uint256 length) view returns(bytes)
func (_StorageAccessible *StorageAccessibleCallerSession) GetStorageAt(offset *big.Int, length *big.Int) ([]byte, error) {
	return _StorageAccessible.Contract.GetStorageAt(&_StorageAccessible.CallOpts, offset, length)
}

// SimulateAndRevert is a paid mutator transaction binding the contract method 0xb4faba09.
//
// Solidity: function simulateAndRevert(address targetContract, bytes calldataPayload) returns()
func (_StorageAccessible *StorageAccessibleTransactor) SimulateAndRevert(opts *bind.TransactOpts, targetContract common.Address, calldataPayload []byte) (*types.Transaction, error) {
	return _StorageAccessible.contract.Transact(opts, "simulateAndRevert", targetContract, calldataPayload)
}

// SimulateAndRevert is a paid mutator transaction binding the contract method 0xb4faba09.
//
// Solidity: function simulateAndRevert(address targetContract, bytes calldataPayload) returns()
func (_StorageAccessible *StorageAccessibleSession) SimulateAndRevert(targetContract common.Address, calldataPayload []byte) (*types.Transaction, error) {
	return _StorageAccessible.Contract.SimulateAndRevert(&_StorageAccessible.TransactOpts, targetContract, calldataPayload)
}

// SimulateAndRevert is a paid mutator transaction binding the contract method 0xb4faba09.
//
// Solidity: function simulateAndRevert(address targetContract, bytes calldataPayload) returns()
func (_StorageAccessible *StorageAccessibleTransactorSession) SimulateAndRevert(targetContract common.Address, calldataPayload []byte) (*types.Transaction, error) {
	return _StorageAccessible.Contract.SimulateAndRevert(&_StorageAccessible.TransactOpts, targetContract, calldataPayload)
}
