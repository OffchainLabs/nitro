// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package librariesgen

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

// AddressAliasHelperMetaData contains all meta data concerning the AddressAliasHelper contract.
var AddressAliasHelperMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212201b1c7c1eed6308a3167edcc47c3ecd54d20200159e384b2fa53f8ed918ceaeee64736f6c63430008110033",
}

// AddressAliasHelperABI is the input ABI used to generate the binding from.
// Deprecated: Use AddressAliasHelperMetaData.ABI instead.
var AddressAliasHelperABI = AddressAliasHelperMetaData.ABI

// AddressAliasHelperBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AddressAliasHelperMetaData.Bin instead.
var AddressAliasHelperBin = AddressAliasHelperMetaData.Bin

// DeployAddressAliasHelper deploys a new Ethereum contract, binding an instance of AddressAliasHelper to it.
func DeployAddressAliasHelper(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *AddressAliasHelper, error) {
	parsed, err := AddressAliasHelperMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AddressAliasHelperBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AddressAliasHelper{AddressAliasHelperCaller: AddressAliasHelperCaller{contract: contract}, AddressAliasHelperTransactor: AddressAliasHelperTransactor{contract: contract}, AddressAliasHelperFilterer: AddressAliasHelperFilterer{contract: contract}}, nil
}

// AddressAliasHelper is an auto generated Go binding around an Ethereum contract.
type AddressAliasHelper struct {
	AddressAliasHelperCaller     // Read-only binding to the contract
	AddressAliasHelperTransactor // Write-only binding to the contract
	AddressAliasHelperFilterer   // Log filterer for contract events
}

// AddressAliasHelperCaller is an auto generated read-only Go binding around an Ethereum contract.
type AddressAliasHelperCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddressAliasHelperTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AddressAliasHelperTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddressAliasHelperFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AddressAliasHelperFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddressAliasHelperSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AddressAliasHelperSession struct {
	Contract     *AddressAliasHelper // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// AddressAliasHelperCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AddressAliasHelperCallerSession struct {
	Contract *AddressAliasHelperCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// AddressAliasHelperTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AddressAliasHelperTransactorSession struct {
	Contract     *AddressAliasHelperTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// AddressAliasHelperRaw is an auto generated low-level Go binding around an Ethereum contract.
type AddressAliasHelperRaw struct {
	Contract *AddressAliasHelper // Generic contract binding to access the raw methods on
}

// AddressAliasHelperCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AddressAliasHelperCallerRaw struct {
	Contract *AddressAliasHelperCaller // Generic read-only contract binding to access the raw methods on
}

// AddressAliasHelperTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AddressAliasHelperTransactorRaw struct {
	Contract *AddressAliasHelperTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAddressAliasHelper creates a new instance of AddressAliasHelper, bound to a specific deployed contract.
func NewAddressAliasHelper(address common.Address, backend bind.ContractBackend) (*AddressAliasHelper, error) {
	contract, err := bindAddressAliasHelper(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AddressAliasHelper{AddressAliasHelperCaller: AddressAliasHelperCaller{contract: contract}, AddressAliasHelperTransactor: AddressAliasHelperTransactor{contract: contract}, AddressAliasHelperFilterer: AddressAliasHelperFilterer{contract: contract}}, nil
}

// NewAddressAliasHelperCaller creates a new read-only instance of AddressAliasHelper, bound to a specific deployed contract.
func NewAddressAliasHelperCaller(address common.Address, caller bind.ContractCaller) (*AddressAliasHelperCaller, error) {
	contract, err := bindAddressAliasHelper(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AddressAliasHelperCaller{contract: contract}, nil
}

// NewAddressAliasHelperTransactor creates a new write-only instance of AddressAliasHelper, bound to a specific deployed contract.
func NewAddressAliasHelperTransactor(address common.Address, transactor bind.ContractTransactor) (*AddressAliasHelperTransactor, error) {
	contract, err := bindAddressAliasHelper(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AddressAliasHelperTransactor{contract: contract}, nil
}

// NewAddressAliasHelperFilterer creates a new log filterer instance of AddressAliasHelper, bound to a specific deployed contract.
func NewAddressAliasHelperFilterer(address common.Address, filterer bind.ContractFilterer) (*AddressAliasHelperFilterer, error) {
	contract, err := bindAddressAliasHelper(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AddressAliasHelperFilterer{contract: contract}, nil
}

// bindAddressAliasHelper binds a generic wrapper to an already deployed contract.
func bindAddressAliasHelper(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AddressAliasHelperMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AddressAliasHelper *AddressAliasHelperRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AddressAliasHelper.Contract.AddressAliasHelperCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AddressAliasHelper *AddressAliasHelperRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AddressAliasHelper.Contract.AddressAliasHelperTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AddressAliasHelper *AddressAliasHelperRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AddressAliasHelper.Contract.AddressAliasHelperTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AddressAliasHelper *AddressAliasHelperCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AddressAliasHelper.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AddressAliasHelper *AddressAliasHelperTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AddressAliasHelper.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AddressAliasHelper *AddressAliasHelperTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AddressAliasHelper.Contract.contract.Transact(opts, method, params...)
}

// AdminFallbackProxyMetaData contains all meta data concerning the AdminFallbackProxy contract.
var AdminFallbackProxyMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"previousAdmin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"AdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"beacon\",\"type\":\"address\"}],\"name\":\"BeaconUpgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"UpgradedSecondary\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610245806100206000396000f3fe60806040523661001357610011610017565b005b6100115b610027610022610029565b61015b565b565b6000600436101561009b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600b60248201527f4e4f5f46554e435f53494700000000000000000000000000000000000000000060448201526064015b60405180910390fd5b6000336100a661017f565b73ffffffffffffffffffffffffffffffffffffffff16036100ce576100c96101bf565b6100d6565b6100d66101e7565b905073ffffffffffffffffffffffffffffffffffffffff81163b610156576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601360248201527f5441524745545f4e4f545f434f4e5452414354000000000000000000000000006044820152606401610092565b919050565b3660008037600080366000845af43d6000803e80801561017a573d6000f35b3d6000fd5b60007fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61035b5473ffffffffffffffffffffffffffffffffffffffff16919050565b60007f360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc6101a3565b60007f2b1dbce74324248c222f0ec2d5ed7bd323cfc425b336f0253c5ccfda7265546d6101a356fea26469706673582212203170ab15fbcb10688949c192cd5e26073e1877dee31cd823a939d33a39fc1a6e64736f6c63430008110033",
}

// AdminFallbackProxyABI is the input ABI used to generate the binding from.
// Deprecated: Use AdminFallbackProxyMetaData.ABI instead.
var AdminFallbackProxyABI = AdminFallbackProxyMetaData.ABI

// AdminFallbackProxyBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AdminFallbackProxyMetaData.Bin instead.
var AdminFallbackProxyBin = AdminFallbackProxyMetaData.Bin

// DeployAdminFallbackProxy deploys a new Ethereum contract, binding an instance of AdminFallbackProxy to it.
func DeployAdminFallbackProxy(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *AdminFallbackProxy, error) {
	parsed, err := AdminFallbackProxyMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AdminFallbackProxyBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AdminFallbackProxy{AdminFallbackProxyCaller: AdminFallbackProxyCaller{contract: contract}, AdminFallbackProxyTransactor: AdminFallbackProxyTransactor{contract: contract}, AdminFallbackProxyFilterer: AdminFallbackProxyFilterer{contract: contract}}, nil
}

// AdminFallbackProxy is an auto generated Go binding around an Ethereum contract.
type AdminFallbackProxy struct {
	AdminFallbackProxyCaller     // Read-only binding to the contract
	AdminFallbackProxyTransactor // Write-only binding to the contract
	AdminFallbackProxyFilterer   // Log filterer for contract events
}

// AdminFallbackProxyCaller is an auto generated read-only Go binding around an Ethereum contract.
type AdminFallbackProxyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AdminFallbackProxyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AdminFallbackProxyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AdminFallbackProxyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AdminFallbackProxyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AdminFallbackProxySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AdminFallbackProxySession struct {
	Contract     *AdminFallbackProxy // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// AdminFallbackProxyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AdminFallbackProxyCallerSession struct {
	Contract *AdminFallbackProxyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// AdminFallbackProxyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AdminFallbackProxyTransactorSession struct {
	Contract     *AdminFallbackProxyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// AdminFallbackProxyRaw is an auto generated low-level Go binding around an Ethereum contract.
type AdminFallbackProxyRaw struct {
	Contract *AdminFallbackProxy // Generic contract binding to access the raw methods on
}

// AdminFallbackProxyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AdminFallbackProxyCallerRaw struct {
	Contract *AdminFallbackProxyCaller // Generic read-only contract binding to access the raw methods on
}

// AdminFallbackProxyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AdminFallbackProxyTransactorRaw struct {
	Contract *AdminFallbackProxyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAdminFallbackProxy creates a new instance of AdminFallbackProxy, bound to a specific deployed contract.
func NewAdminFallbackProxy(address common.Address, backend bind.ContractBackend) (*AdminFallbackProxy, error) {
	contract, err := bindAdminFallbackProxy(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AdminFallbackProxy{AdminFallbackProxyCaller: AdminFallbackProxyCaller{contract: contract}, AdminFallbackProxyTransactor: AdminFallbackProxyTransactor{contract: contract}, AdminFallbackProxyFilterer: AdminFallbackProxyFilterer{contract: contract}}, nil
}

// NewAdminFallbackProxyCaller creates a new read-only instance of AdminFallbackProxy, bound to a specific deployed contract.
func NewAdminFallbackProxyCaller(address common.Address, caller bind.ContractCaller) (*AdminFallbackProxyCaller, error) {
	contract, err := bindAdminFallbackProxy(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AdminFallbackProxyCaller{contract: contract}, nil
}

// NewAdminFallbackProxyTransactor creates a new write-only instance of AdminFallbackProxy, bound to a specific deployed contract.
func NewAdminFallbackProxyTransactor(address common.Address, transactor bind.ContractTransactor) (*AdminFallbackProxyTransactor, error) {
	contract, err := bindAdminFallbackProxy(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AdminFallbackProxyTransactor{contract: contract}, nil
}

// NewAdminFallbackProxyFilterer creates a new log filterer instance of AdminFallbackProxy, bound to a specific deployed contract.
func NewAdminFallbackProxyFilterer(address common.Address, filterer bind.ContractFilterer) (*AdminFallbackProxyFilterer, error) {
	contract, err := bindAdminFallbackProxy(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AdminFallbackProxyFilterer{contract: contract}, nil
}

// bindAdminFallbackProxy binds a generic wrapper to an already deployed contract.
func bindAdminFallbackProxy(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AdminFallbackProxyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AdminFallbackProxy *AdminFallbackProxyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AdminFallbackProxy.Contract.AdminFallbackProxyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AdminFallbackProxy *AdminFallbackProxyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AdminFallbackProxy.Contract.AdminFallbackProxyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AdminFallbackProxy *AdminFallbackProxyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AdminFallbackProxy.Contract.AdminFallbackProxyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AdminFallbackProxy *AdminFallbackProxyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AdminFallbackProxy.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AdminFallbackProxy *AdminFallbackProxyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AdminFallbackProxy.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AdminFallbackProxy *AdminFallbackProxyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AdminFallbackProxy.Contract.contract.Transact(opts, method, params...)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_AdminFallbackProxy *AdminFallbackProxyTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _AdminFallbackProxy.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_AdminFallbackProxy *AdminFallbackProxySession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _AdminFallbackProxy.Contract.Fallback(&_AdminFallbackProxy.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_AdminFallbackProxy *AdminFallbackProxyTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _AdminFallbackProxy.Contract.Fallback(&_AdminFallbackProxy.TransactOpts, calldata)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_AdminFallbackProxy *AdminFallbackProxyTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AdminFallbackProxy.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_AdminFallbackProxy *AdminFallbackProxySession) Receive() (*types.Transaction, error) {
	return _AdminFallbackProxy.Contract.Receive(&_AdminFallbackProxy.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_AdminFallbackProxy *AdminFallbackProxyTransactorSession) Receive() (*types.Transaction, error) {
	return _AdminFallbackProxy.Contract.Receive(&_AdminFallbackProxy.TransactOpts)
}

// AdminFallbackProxyAdminChangedIterator is returned from FilterAdminChanged and is used to iterate over the raw logs and unpacked data for AdminChanged events raised by the AdminFallbackProxy contract.
type AdminFallbackProxyAdminChangedIterator struct {
	Event *AdminFallbackProxyAdminChanged // Event containing the contract specifics and raw log

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
func (it *AdminFallbackProxyAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AdminFallbackProxyAdminChanged)
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
		it.Event = new(AdminFallbackProxyAdminChanged)
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
func (it *AdminFallbackProxyAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AdminFallbackProxyAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AdminFallbackProxyAdminChanged represents a AdminChanged event raised by the AdminFallbackProxy contract.
type AdminFallbackProxyAdminChanged struct {
	PreviousAdmin common.Address
	NewAdmin      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterAdminChanged is a free log retrieval operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) FilterAdminChanged(opts *bind.FilterOpts) (*AdminFallbackProxyAdminChangedIterator, error) {

	logs, sub, err := _AdminFallbackProxy.contract.FilterLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return &AdminFallbackProxyAdminChangedIterator{contract: _AdminFallbackProxy.contract, event: "AdminChanged", logs: logs, sub: sub}, nil
}

// WatchAdminChanged is a free log subscription operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) WatchAdminChanged(opts *bind.WatchOpts, sink chan<- *AdminFallbackProxyAdminChanged) (event.Subscription, error) {

	logs, sub, err := _AdminFallbackProxy.contract.WatchLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AdminFallbackProxyAdminChanged)
				if err := _AdminFallbackProxy.contract.UnpackLog(event, "AdminChanged", log); err != nil {
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

// ParseAdminChanged is a log parse operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) ParseAdminChanged(log types.Log) (*AdminFallbackProxyAdminChanged, error) {
	event := new(AdminFallbackProxyAdminChanged)
	if err := _AdminFallbackProxy.contract.UnpackLog(event, "AdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AdminFallbackProxyBeaconUpgradedIterator is returned from FilterBeaconUpgraded and is used to iterate over the raw logs and unpacked data for BeaconUpgraded events raised by the AdminFallbackProxy contract.
type AdminFallbackProxyBeaconUpgradedIterator struct {
	Event *AdminFallbackProxyBeaconUpgraded // Event containing the contract specifics and raw log

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
func (it *AdminFallbackProxyBeaconUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AdminFallbackProxyBeaconUpgraded)
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
		it.Event = new(AdminFallbackProxyBeaconUpgraded)
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
func (it *AdminFallbackProxyBeaconUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AdminFallbackProxyBeaconUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AdminFallbackProxyBeaconUpgraded represents a BeaconUpgraded event raised by the AdminFallbackProxy contract.
type AdminFallbackProxyBeaconUpgraded struct {
	Beacon common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBeaconUpgraded is a free log retrieval operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) FilterBeaconUpgraded(opts *bind.FilterOpts, beacon []common.Address) (*AdminFallbackProxyBeaconUpgradedIterator, error) {

	var beaconRule []interface{}
	for _, beaconItem := range beacon {
		beaconRule = append(beaconRule, beaconItem)
	}

	logs, sub, err := _AdminFallbackProxy.contract.FilterLogs(opts, "BeaconUpgraded", beaconRule)
	if err != nil {
		return nil, err
	}
	return &AdminFallbackProxyBeaconUpgradedIterator{contract: _AdminFallbackProxy.contract, event: "BeaconUpgraded", logs: logs, sub: sub}, nil
}

// WatchBeaconUpgraded is a free log subscription operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) WatchBeaconUpgraded(opts *bind.WatchOpts, sink chan<- *AdminFallbackProxyBeaconUpgraded, beacon []common.Address) (event.Subscription, error) {

	var beaconRule []interface{}
	for _, beaconItem := range beacon {
		beaconRule = append(beaconRule, beaconItem)
	}

	logs, sub, err := _AdminFallbackProxy.contract.WatchLogs(opts, "BeaconUpgraded", beaconRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AdminFallbackProxyBeaconUpgraded)
				if err := _AdminFallbackProxy.contract.UnpackLog(event, "BeaconUpgraded", log); err != nil {
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

// ParseBeaconUpgraded is a log parse operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) ParseBeaconUpgraded(log types.Log) (*AdminFallbackProxyBeaconUpgraded, error) {
	event := new(AdminFallbackProxyBeaconUpgraded)
	if err := _AdminFallbackProxy.contract.UnpackLog(event, "BeaconUpgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AdminFallbackProxyUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the AdminFallbackProxy contract.
type AdminFallbackProxyUpgradedIterator struct {
	Event *AdminFallbackProxyUpgraded // Event containing the contract specifics and raw log

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
func (it *AdminFallbackProxyUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AdminFallbackProxyUpgraded)
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
		it.Event = new(AdminFallbackProxyUpgraded)
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
func (it *AdminFallbackProxyUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AdminFallbackProxyUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AdminFallbackProxyUpgraded represents a Upgraded event raised by the AdminFallbackProxy contract.
type AdminFallbackProxyUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*AdminFallbackProxyUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _AdminFallbackProxy.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &AdminFallbackProxyUpgradedIterator{contract: _AdminFallbackProxy.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *AdminFallbackProxyUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _AdminFallbackProxy.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AdminFallbackProxyUpgraded)
				if err := _AdminFallbackProxy.contract.UnpackLog(event, "Upgraded", log); err != nil {
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

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) ParseUpgraded(log types.Log) (*AdminFallbackProxyUpgraded, error) {
	event := new(AdminFallbackProxyUpgraded)
	if err := _AdminFallbackProxy.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AdminFallbackProxyUpgradedSecondaryIterator is returned from FilterUpgradedSecondary and is used to iterate over the raw logs and unpacked data for UpgradedSecondary events raised by the AdminFallbackProxy contract.
type AdminFallbackProxyUpgradedSecondaryIterator struct {
	Event *AdminFallbackProxyUpgradedSecondary // Event containing the contract specifics and raw log

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
func (it *AdminFallbackProxyUpgradedSecondaryIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AdminFallbackProxyUpgradedSecondary)
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
		it.Event = new(AdminFallbackProxyUpgradedSecondary)
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
func (it *AdminFallbackProxyUpgradedSecondaryIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AdminFallbackProxyUpgradedSecondaryIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AdminFallbackProxyUpgradedSecondary represents a UpgradedSecondary event raised by the AdminFallbackProxy contract.
type AdminFallbackProxyUpgradedSecondary struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgradedSecondary is a free log retrieval operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) FilterUpgradedSecondary(opts *bind.FilterOpts, implementation []common.Address) (*AdminFallbackProxyUpgradedSecondaryIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _AdminFallbackProxy.contract.FilterLogs(opts, "UpgradedSecondary", implementationRule)
	if err != nil {
		return nil, err
	}
	return &AdminFallbackProxyUpgradedSecondaryIterator{contract: _AdminFallbackProxy.contract, event: "UpgradedSecondary", logs: logs, sub: sub}, nil
}

// WatchUpgradedSecondary is a free log subscription operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) WatchUpgradedSecondary(opts *bind.WatchOpts, sink chan<- *AdminFallbackProxyUpgradedSecondary, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _AdminFallbackProxy.contract.WatchLogs(opts, "UpgradedSecondary", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AdminFallbackProxyUpgradedSecondary)
				if err := _AdminFallbackProxy.contract.UnpackLog(event, "UpgradedSecondary", log); err != nil {
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

// ParseUpgradedSecondary is a log parse operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_AdminFallbackProxy *AdminFallbackProxyFilterer) ParseUpgradedSecondary(log types.Log) (*AdminFallbackProxyUpgradedSecondary, error) {
	event := new(AdminFallbackProxyUpgradedSecondary)
	if err := _AdminFallbackProxy.contract.UnpackLog(event, "UpgradedSecondary", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbitrumCheckerMetaData contains all meta data concerning the ArbitrumChecker contract.
var ArbitrumCheckerMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea264697066735822122040c6bb794acf80741366e0fd21ce6d4bc9275edf89631a495ced7d2c12534ada64736f6c63430008110033",
}

// ArbitrumCheckerABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbitrumCheckerMetaData.ABI instead.
var ArbitrumCheckerABI = ArbitrumCheckerMetaData.ABI

// ArbitrumCheckerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ArbitrumCheckerMetaData.Bin instead.
var ArbitrumCheckerBin = ArbitrumCheckerMetaData.Bin

// DeployArbitrumChecker deploys a new Ethereum contract, binding an instance of ArbitrumChecker to it.
func DeployArbitrumChecker(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ArbitrumChecker, error) {
	parsed, err := ArbitrumCheckerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ArbitrumCheckerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ArbitrumChecker{ArbitrumCheckerCaller: ArbitrumCheckerCaller{contract: contract}, ArbitrumCheckerTransactor: ArbitrumCheckerTransactor{contract: contract}, ArbitrumCheckerFilterer: ArbitrumCheckerFilterer{contract: contract}}, nil
}

// ArbitrumChecker is an auto generated Go binding around an Ethereum contract.
type ArbitrumChecker struct {
	ArbitrumCheckerCaller     // Read-only binding to the contract
	ArbitrumCheckerTransactor // Write-only binding to the contract
	ArbitrumCheckerFilterer   // Log filterer for contract events
}

// ArbitrumCheckerCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbitrumCheckerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbitrumCheckerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbitrumCheckerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbitrumCheckerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbitrumCheckerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbitrumCheckerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbitrumCheckerSession struct {
	Contract     *ArbitrumChecker  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbitrumCheckerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbitrumCheckerCallerSession struct {
	Contract *ArbitrumCheckerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// ArbitrumCheckerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbitrumCheckerTransactorSession struct {
	Contract     *ArbitrumCheckerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// ArbitrumCheckerRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbitrumCheckerRaw struct {
	Contract *ArbitrumChecker // Generic contract binding to access the raw methods on
}

// ArbitrumCheckerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbitrumCheckerCallerRaw struct {
	Contract *ArbitrumCheckerCaller // Generic read-only contract binding to access the raw methods on
}

// ArbitrumCheckerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbitrumCheckerTransactorRaw struct {
	Contract *ArbitrumCheckerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbitrumChecker creates a new instance of ArbitrumChecker, bound to a specific deployed contract.
func NewArbitrumChecker(address common.Address, backend bind.ContractBackend) (*ArbitrumChecker, error) {
	contract, err := bindArbitrumChecker(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbitrumChecker{ArbitrumCheckerCaller: ArbitrumCheckerCaller{contract: contract}, ArbitrumCheckerTransactor: ArbitrumCheckerTransactor{contract: contract}, ArbitrumCheckerFilterer: ArbitrumCheckerFilterer{contract: contract}}, nil
}

// NewArbitrumCheckerCaller creates a new read-only instance of ArbitrumChecker, bound to a specific deployed contract.
func NewArbitrumCheckerCaller(address common.Address, caller bind.ContractCaller) (*ArbitrumCheckerCaller, error) {
	contract, err := bindArbitrumChecker(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbitrumCheckerCaller{contract: contract}, nil
}

// NewArbitrumCheckerTransactor creates a new write-only instance of ArbitrumChecker, bound to a specific deployed contract.
func NewArbitrumCheckerTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbitrumCheckerTransactor, error) {
	contract, err := bindArbitrumChecker(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbitrumCheckerTransactor{contract: contract}, nil
}

// NewArbitrumCheckerFilterer creates a new log filterer instance of ArbitrumChecker, bound to a specific deployed contract.
func NewArbitrumCheckerFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbitrumCheckerFilterer, error) {
	contract, err := bindArbitrumChecker(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbitrumCheckerFilterer{contract: contract}, nil
}

// bindArbitrumChecker binds a generic wrapper to an already deployed contract.
func bindArbitrumChecker(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbitrumCheckerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbitrumChecker *ArbitrumCheckerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbitrumChecker.Contract.ArbitrumCheckerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbitrumChecker *ArbitrumCheckerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbitrumChecker.Contract.ArbitrumCheckerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbitrumChecker *ArbitrumCheckerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbitrumChecker.Contract.ArbitrumCheckerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbitrumChecker *ArbitrumCheckerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbitrumChecker.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbitrumChecker *ArbitrumCheckerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbitrumChecker.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbitrumChecker *ArbitrumCheckerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbitrumChecker.Contract.contract.Transact(opts, method, params...)
}

// CryptographyPrimitivesMetaData contains all meta data concerning the CryptographyPrimitives contract.
var CryptographyPrimitivesMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220cf259683d0f53e6866060206401ed3afb138c743830ab040df6197c5b8f3cc8e64736f6c63430008110033",
}

// CryptographyPrimitivesABI is the input ABI used to generate the binding from.
// Deprecated: Use CryptographyPrimitivesMetaData.ABI instead.
var CryptographyPrimitivesABI = CryptographyPrimitivesMetaData.ABI

// CryptographyPrimitivesBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CryptographyPrimitivesMetaData.Bin instead.
var CryptographyPrimitivesBin = CryptographyPrimitivesMetaData.Bin

// DeployCryptographyPrimitives deploys a new Ethereum contract, binding an instance of CryptographyPrimitives to it.
func DeployCryptographyPrimitives(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *CryptographyPrimitives, error) {
	parsed, err := CryptographyPrimitivesMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CryptographyPrimitivesBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &CryptographyPrimitives{CryptographyPrimitivesCaller: CryptographyPrimitivesCaller{contract: contract}, CryptographyPrimitivesTransactor: CryptographyPrimitivesTransactor{contract: contract}, CryptographyPrimitivesFilterer: CryptographyPrimitivesFilterer{contract: contract}}, nil
}

// CryptographyPrimitives is an auto generated Go binding around an Ethereum contract.
type CryptographyPrimitives struct {
	CryptographyPrimitivesCaller     // Read-only binding to the contract
	CryptographyPrimitivesTransactor // Write-only binding to the contract
	CryptographyPrimitivesFilterer   // Log filterer for contract events
}

// CryptographyPrimitivesCaller is an auto generated read-only Go binding around an Ethereum contract.
type CryptographyPrimitivesCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptographyPrimitivesTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CryptographyPrimitivesTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptographyPrimitivesFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CryptographyPrimitivesFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptographyPrimitivesSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CryptographyPrimitivesSession struct {
	Contract     *CryptographyPrimitives // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// CryptographyPrimitivesCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CryptographyPrimitivesCallerSession struct {
	Contract *CryptographyPrimitivesCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// CryptographyPrimitivesTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CryptographyPrimitivesTransactorSession struct {
	Contract     *CryptographyPrimitivesTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// CryptographyPrimitivesRaw is an auto generated low-level Go binding around an Ethereum contract.
type CryptographyPrimitivesRaw struct {
	Contract *CryptographyPrimitives // Generic contract binding to access the raw methods on
}

// CryptographyPrimitivesCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CryptographyPrimitivesCallerRaw struct {
	Contract *CryptographyPrimitivesCaller // Generic read-only contract binding to access the raw methods on
}

// CryptographyPrimitivesTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CryptographyPrimitivesTransactorRaw struct {
	Contract *CryptographyPrimitivesTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCryptographyPrimitives creates a new instance of CryptographyPrimitives, bound to a specific deployed contract.
func NewCryptographyPrimitives(address common.Address, backend bind.ContractBackend) (*CryptographyPrimitives, error) {
	contract, err := bindCryptographyPrimitives(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CryptographyPrimitives{CryptographyPrimitivesCaller: CryptographyPrimitivesCaller{contract: contract}, CryptographyPrimitivesTransactor: CryptographyPrimitivesTransactor{contract: contract}, CryptographyPrimitivesFilterer: CryptographyPrimitivesFilterer{contract: contract}}, nil
}

// NewCryptographyPrimitivesCaller creates a new read-only instance of CryptographyPrimitives, bound to a specific deployed contract.
func NewCryptographyPrimitivesCaller(address common.Address, caller bind.ContractCaller) (*CryptographyPrimitivesCaller, error) {
	contract, err := bindCryptographyPrimitives(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CryptographyPrimitivesCaller{contract: contract}, nil
}

// NewCryptographyPrimitivesTransactor creates a new write-only instance of CryptographyPrimitives, bound to a specific deployed contract.
func NewCryptographyPrimitivesTransactor(address common.Address, transactor bind.ContractTransactor) (*CryptographyPrimitivesTransactor, error) {
	contract, err := bindCryptographyPrimitives(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CryptographyPrimitivesTransactor{contract: contract}, nil
}

// NewCryptographyPrimitivesFilterer creates a new log filterer instance of CryptographyPrimitives, bound to a specific deployed contract.
func NewCryptographyPrimitivesFilterer(address common.Address, filterer bind.ContractFilterer) (*CryptographyPrimitivesFilterer, error) {
	contract, err := bindCryptographyPrimitives(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CryptographyPrimitivesFilterer{contract: contract}, nil
}

// bindCryptographyPrimitives binds a generic wrapper to an already deployed contract.
func bindCryptographyPrimitives(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := CryptographyPrimitivesMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CryptographyPrimitives *CryptographyPrimitivesRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CryptographyPrimitives.Contract.CryptographyPrimitivesCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CryptographyPrimitives *CryptographyPrimitivesRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptographyPrimitives.Contract.CryptographyPrimitivesTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CryptographyPrimitives *CryptographyPrimitivesRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CryptographyPrimitives.Contract.CryptographyPrimitivesTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CryptographyPrimitives *CryptographyPrimitivesCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CryptographyPrimitives.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CryptographyPrimitives *CryptographyPrimitivesTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptographyPrimitives.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CryptographyPrimitives *CryptographyPrimitivesTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CryptographyPrimitives.Contract.contract.Transact(opts, method, params...)
}

// DelegateCallAwareMetaData contains all meta data concerning the DelegateCallAware contract.
var DelegateCallAwareMetaData = &bind.MetaData{
	ABI: "[]",
}

// DelegateCallAwareABI is the input ABI used to generate the binding from.
// Deprecated: Use DelegateCallAwareMetaData.ABI instead.
var DelegateCallAwareABI = DelegateCallAwareMetaData.ABI

// DelegateCallAware is an auto generated Go binding around an Ethereum contract.
type DelegateCallAware struct {
	DelegateCallAwareCaller     // Read-only binding to the contract
	DelegateCallAwareTransactor // Write-only binding to the contract
	DelegateCallAwareFilterer   // Log filterer for contract events
}

// DelegateCallAwareCaller is an auto generated read-only Go binding around an Ethereum contract.
type DelegateCallAwareCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateCallAwareTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DelegateCallAwareTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateCallAwareFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DelegateCallAwareFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateCallAwareSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DelegateCallAwareSession struct {
	Contract     *DelegateCallAware // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// DelegateCallAwareCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DelegateCallAwareCallerSession struct {
	Contract *DelegateCallAwareCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// DelegateCallAwareTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DelegateCallAwareTransactorSession struct {
	Contract     *DelegateCallAwareTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// DelegateCallAwareRaw is an auto generated low-level Go binding around an Ethereum contract.
type DelegateCallAwareRaw struct {
	Contract *DelegateCallAware // Generic contract binding to access the raw methods on
}

// DelegateCallAwareCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DelegateCallAwareCallerRaw struct {
	Contract *DelegateCallAwareCaller // Generic read-only contract binding to access the raw methods on
}

// DelegateCallAwareTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DelegateCallAwareTransactorRaw struct {
	Contract *DelegateCallAwareTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDelegateCallAware creates a new instance of DelegateCallAware, bound to a specific deployed contract.
func NewDelegateCallAware(address common.Address, backend bind.ContractBackend) (*DelegateCallAware, error) {
	contract, err := bindDelegateCallAware(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DelegateCallAware{DelegateCallAwareCaller: DelegateCallAwareCaller{contract: contract}, DelegateCallAwareTransactor: DelegateCallAwareTransactor{contract: contract}, DelegateCallAwareFilterer: DelegateCallAwareFilterer{contract: contract}}, nil
}

// NewDelegateCallAwareCaller creates a new read-only instance of DelegateCallAware, bound to a specific deployed contract.
func NewDelegateCallAwareCaller(address common.Address, caller bind.ContractCaller) (*DelegateCallAwareCaller, error) {
	contract, err := bindDelegateCallAware(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DelegateCallAwareCaller{contract: contract}, nil
}

// NewDelegateCallAwareTransactor creates a new write-only instance of DelegateCallAware, bound to a specific deployed contract.
func NewDelegateCallAwareTransactor(address common.Address, transactor bind.ContractTransactor) (*DelegateCallAwareTransactor, error) {
	contract, err := bindDelegateCallAware(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DelegateCallAwareTransactor{contract: contract}, nil
}

// NewDelegateCallAwareFilterer creates a new log filterer instance of DelegateCallAware, bound to a specific deployed contract.
func NewDelegateCallAwareFilterer(address common.Address, filterer bind.ContractFilterer) (*DelegateCallAwareFilterer, error) {
	contract, err := bindDelegateCallAware(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DelegateCallAwareFilterer{contract: contract}, nil
}

// bindDelegateCallAware binds a generic wrapper to an already deployed contract.
func bindDelegateCallAware(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DelegateCallAwareMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelegateCallAware *DelegateCallAwareRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelegateCallAware.Contract.DelegateCallAwareCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelegateCallAware *DelegateCallAwareRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateCallAware.Contract.DelegateCallAwareTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelegateCallAware *DelegateCallAwareRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelegateCallAware.Contract.DelegateCallAwareTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelegateCallAware *DelegateCallAwareCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelegateCallAware.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelegateCallAware *DelegateCallAwareTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateCallAware.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelegateCallAware *DelegateCallAwareTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelegateCallAware.Contract.contract.Transact(opts, method, params...)
}

// DoubleLogicERC1967UpgradeMetaData contains all meta data concerning the DoubleLogicERC1967Upgrade contract.
var DoubleLogicERC1967UpgradeMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"previousAdmin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"AdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"beacon\",\"type\":\"address\"}],\"name\":\"BeaconUpgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"UpgradedSecondary\",\"type\":\"event\"}]",
}

// DoubleLogicERC1967UpgradeABI is the input ABI used to generate the binding from.
// Deprecated: Use DoubleLogicERC1967UpgradeMetaData.ABI instead.
var DoubleLogicERC1967UpgradeABI = DoubleLogicERC1967UpgradeMetaData.ABI

// DoubleLogicERC1967Upgrade is an auto generated Go binding around an Ethereum contract.
type DoubleLogicERC1967Upgrade struct {
	DoubleLogicERC1967UpgradeCaller     // Read-only binding to the contract
	DoubleLogicERC1967UpgradeTransactor // Write-only binding to the contract
	DoubleLogicERC1967UpgradeFilterer   // Log filterer for contract events
}

// DoubleLogicERC1967UpgradeCaller is an auto generated read-only Go binding around an Ethereum contract.
type DoubleLogicERC1967UpgradeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DoubleLogicERC1967UpgradeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DoubleLogicERC1967UpgradeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DoubleLogicERC1967UpgradeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DoubleLogicERC1967UpgradeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DoubleLogicERC1967UpgradeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DoubleLogicERC1967UpgradeSession struct {
	Contract     *DoubleLogicERC1967Upgrade // Generic contract binding to set the session for
	CallOpts     bind.CallOpts              // Call options to use throughout this session
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// DoubleLogicERC1967UpgradeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DoubleLogicERC1967UpgradeCallerSession struct {
	Contract *DoubleLogicERC1967UpgradeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                    // Call options to use throughout this session
}

// DoubleLogicERC1967UpgradeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DoubleLogicERC1967UpgradeTransactorSession struct {
	Contract     *DoubleLogicERC1967UpgradeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                    // Transaction auth options to use throughout this session
}

// DoubleLogicERC1967UpgradeRaw is an auto generated low-level Go binding around an Ethereum contract.
type DoubleLogicERC1967UpgradeRaw struct {
	Contract *DoubleLogicERC1967Upgrade // Generic contract binding to access the raw methods on
}

// DoubleLogicERC1967UpgradeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DoubleLogicERC1967UpgradeCallerRaw struct {
	Contract *DoubleLogicERC1967UpgradeCaller // Generic read-only contract binding to access the raw methods on
}

// DoubleLogicERC1967UpgradeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DoubleLogicERC1967UpgradeTransactorRaw struct {
	Contract *DoubleLogicERC1967UpgradeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDoubleLogicERC1967Upgrade creates a new instance of DoubleLogicERC1967Upgrade, bound to a specific deployed contract.
func NewDoubleLogicERC1967Upgrade(address common.Address, backend bind.ContractBackend) (*DoubleLogicERC1967Upgrade, error) {
	contract, err := bindDoubleLogicERC1967Upgrade(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicERC1967Upgrade{DoubleLogicERC1967UpgradeCaller: DoubleLogicERC1967UpgradeCaller{contract: contract}, DoubleLogicERC1967UpgradeTransactor: DoubleLogicERC1967UpgradeTransactor{contract: contract}, DoubleLogicERC1967UpgradeFilterer: DoubleLogicERC1967UpgradeFilterer{contract: contract}}, nil
}

// NewDoubleLogicERC1967UpgradeCaller creates a new read-only instance of DoubleLogicERC1967Upgrade, bound to a specific deployed contract.
func NewDoubleLogicERC1967UpgradeCaller(address common.Address, caller bind.ContractCaller) (*DoubleLogicERC1967UpgradeCaller, error) {
	contract, err := bindDoubleLogicERC1967Upgrade(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicERC1967UpgradeCaller{contract: contract}, nil
}

// NewDoubleLogicERC1967UpgradeTransactor creates a new write-only instance of DoubleLogicERC1967Upgrade, bound to a specific deployed contract.
func NewDoubleLogicERC1967UpgradeTransactor(address common.Address, transactor bind.ContractTransactor) (*DoubleLogicERC1967UpgradeTransactor, error) {
	contract, err := bindDoubleLogicERC1967Upgrade(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicERC1967UpgradeTransactor{contract: contract}, nil
}

// NewDoubleLogicERC1967UpgradeFilterer creates a new log filterer instance of DoubleLogicERC1967Upgrade, bound to a specific deployed contract.
func NewDoubleLogicERC1967UpgradeFilterer(address common.Address, filterer bind.ContractFilterer) (*DoubleLogicERC1967UpgradeFilterer, error) {
	contract, err := bindDoubleLogicERC1967Upgrade(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicERC1967UpgradeFilterer{contract: contract}, nil
}

// bindDoubleLogicERC1967Upgrade binds a generic wrapper to an already deployed contract.
func bindDoubleLogicERC1967Upgrade(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DoubleLogicERC1967UpgradeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DoubleLogicERC1967Upgrade.Contract.DoubleLogicERC1967UpgradeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DoubleLogicERC1967Upgrade.Contract.DoubleLogicERC1967UpgradeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DoubleLogicERC1967Upgrade.Contract.DoubleLogicERC1967UpgradeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DoubleLogicERC1967Upgrade.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DoubleLogicERC1967Upgrade.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DoubleLogicERC1967Upgrade.Contract.contract.Transact(opts, method, params...)
}

// DoubleLogicERC1967UpgradeAdminChangedIterator is returned from FilterAdminChanged and is used to iterate over the raw logs and unpacked data for AdminChanged events raised by the DoubleLogicERC1967Upgrade contract.
type DoubleLogicERC1967UpgradeAdminChangedIterator struct {
	Event *DoubleLogicERC1967UpgradeAdminChanged // Event containing the contract specifics and raw log

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
func (it *DoubleLogicERC1967UpgradeAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DoubleLogicERC1967UpgradeAdminChanged)
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
		it.Event = new(DoubleLogicERC1967UpgradeAdminChanged)
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
func (it *DoubleLogicERC1967UpgradeAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DoubleLogicERC1967UpgradeAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DoubleLogicERC1967UpgradeAdminChanged represents a AdminChanged event raised by the DoubleLogicERC1967Upgrade contract.
type DoubleLogicERC1967UpgradeAdminChanged struct {
	PreviousAdmin common.Address
	NewAdmin      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterAdminChanged is a free log retrieval operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) FilterAdminChanged(opts *bind.FilterOpts) (*DoubleLogicERC1967UpgradeAdminChangedIterator, error) {

	logs, sub, err := _DoubleLogicERC1967Upgrade.contract.FilterLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return &DoubleLogicERC1967UpgradeAdminChangedIterator{contract: _DoubleLogicERC1967Upgrade.contract, event: "AdminChanged", logs: logs, sub: sub}, nil
}

// WatchAdminChanged is a free log subscription operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) WatchAdminChanged(opts *bind.WatchOpts, sink chan<- *DoubleLogicERC1967UpgradeAdminChanged) (event.Subscription, error) {

	logs, sub, err := _DoubleLogicERC1967Upgrade.contract.WatchLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DoubleLogicERC1967UpgradeAdminChanged)
				if err := _DoubleLogicERC1967Upgrade.contract.UnpackLog(event, "AdminChanged", log); err != nil {
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

// ParseAdminChanged is a log parse operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) ParseAdminChanged(log types.Log) (*DoubleLogicERC1967UpgradeAdminChanged, error) {
	event := new(DoubleLogicERC1967UpgradeAdminChanged)
	if err := _DoubleLogicERC1967Upgrade.contract.UnpackLog(event, "AdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DoubleLogicERC1967UpgradeBeaconUpgradedIterator is returned from FilterBeaconUpgraded and is used to iterate over the raw logs and unpacked data for BeaconUpgraded events raised by the DoubleLogicERC1967Upgrade contract.
type DoubleLogicERC1967UpgradeBeaconUpgradedIterator struct {
	Event *DoubleLogicERC1967UpgradeBeaconUpgraded // Event containing the contract specifics and raw log

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
func (it *DoubleLogicERC1967UpgradeBeaconUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DoubleLogicERC1967UpgradeBeaconUpgraded)
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
		it.Event = new(DoubleLogicERC1967UpgradeBeaconUpgraded)
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
func (it *DoubleLogicERC1967UpgradeBeaconUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DoubleLogicERC1967UpgradeBeaconUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DoubleLogicERC1967UpgradeBeaconUpgraded represents a BeaconUpgraded event raised by the DoubleLogicERC1967Upgrade contract.
type DoubleLogicERC1967UpgradeBeaconUpgraded struct {
	Beacon common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBeaconUpgraded is a free log retrieval operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) FilterBeaconUpgraded(opts *bind.FilterOpts, beacon []common.Address) (*DoubleLogicERC1967UpgradeBeaconUpgradedIterator, error) {

	var beaconRule []interface{}
	for _, beaconItem := range beacon {
		beaconRule = append(beaconRule, beaconItem)
	}

	logs, sub, err := _DoubleLogicERC1967Upgrade.contract.FilterLogs(opts, "BeaconUpgraded", beaconRule)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicERC1967UpgradeBeaconUpgradedIterator{contract: _DoubleLogicERC1967Upgrade.contract, event: "BeaconUpgraded", logs: logs, sub: sub}, nil
}

// WatchBeaconUpgraded is a free log subscription operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) WatchBeaconUpgraded(opts *bind.WatchOpts, sink chan<- *DoubleLogicERC1967UpgradeBeaconUpgraded, beacon []common.Address) (event.Subscription, error) {

	var beaconRule []interface{}
	for _, beaconItem := range beacon {
		beaconRule = append(beaconRule, beaconItem)
	}

	logs, sub, err := _DoubleLogicERC1967Upgrade.contract.WatchLogs(opts, "BeaconUpgraded", beaconRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DoubleLogicERC1967UpgradeBeaconUpgraded)
				if err := _DoubleLogicERC1967Upgrade.contract.UnpackLog(event, "BeaconUpgraded", log); err != nil {
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

// ParseBeaconUpgraded is a log parse operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) ParseBeaconUpgraded(log types.Log) (*DoubleLogicERC1967UpgradeBeaconUpgraded, error) {
	event := new(DoubleLogicERC1967UpgradeBeaconUpgraded)
	if err := _DoubleLogicERC1967Upgrade.contract.UnpackLog(event, "BeaconUpgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DoubleLogicERC1967UpgradeUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the DoubleLogicERC1967Upgrade contract.
type DoubleLogicERC1967UpgradeUpgradedIterator struct {
	Event *DoubleLogicERC1967UpgradeUpgraded // Event containing the contract specifics and raw log

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
func (it *DoubleLogicERC1967UpgradeUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DoubleLogicERC1967UpgradeUpgraded)
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
		it.Event = new(DoubleLogicERC1967UpgradeUpgraded)
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
func (it *DoubleLogicERC1967UpgradeUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DoubleLogicERC1967UpgradeUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DoubleLogicERC1967UpgradeUpgraded represents a Upgraded event raised by the DoubleLogicERC1967Upgrade contract.
type DoubleLogicERC1967UpgradeUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*DoubleLogicERC1967UpgradeUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _DoubleLogicERC1967Upgrade.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicERC1967UpgradeUpgradedIterator{contract: _DoubleLogicERC1967Upgrade.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *DoubleLogicERC1967UpgradeUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _DoubleLogicERC1967Upgrade.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DoubleLogicERC1967UpgradeUpgraded)
				if err := _DoubleLogicERC1967Upgrade.contract.UnpackLog(event, "Upgraded", log); err != nil {
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

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) ParseUpgraded(log types.Log) (*DoubleLogicERC1967UpgradeUpgraded, error) {
	event := new(DoubleLogicERC1967UpgradeUpgraded)
	if err := _DoubleLogicERC1967Upgrade.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DoubleLogicERC1967UpgradeUpgradedSecondaryIterator is returned from FilterUpgradedSecondary and is used to iterate over the raw logs and unpacked data for UpgradedSecondary events raised by the DoubleLogicERC1967Upgrade contract.
type DoubleLogicERC1967UpgradeUpgradedSecondaryIterator struct {
	Event *DoubleLogicERC1967UpgradeUpgradedSecondary // Event containing the contract specifics and raw log

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
func (it *DoubleLogicERC1967UpgradeUpgradedSecondaryIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DoubleLogicERC1967UpgradeUpgradedSecondary)
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
		it.Event = new(DoubleLogicERC1967UpgradeUpgradedSecondary)
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
func (it *DoubleLogicERC1967UpgradeUpgradedSecondaryIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DoubleLogicERC1967UpgradeUpgradedSecondaryIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DoubleLogicERC1967UpgradeUpgradedSecondary represents a UpgradedSecondary event raised by the DoubleLogicERC1967Upgrade contract.
type DoubleLogicERC1967UpgradeUpgradedSecondary struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgradedSecondary is a free log retrieval operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) FilterUpgradedSecondary(opts *bind.FilterOpts, implementation []common.Address) (*DoubleLogicERC1967UpgradeUpgradedSecondaryIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _DoubleLogicERC1967Upgrade.contract.FilterLogs(opts, "UpgradedSecondary", implementationRule)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicERC1967UpgradeUpgradedSecondaryIterator{contract: _DoubleLogicERC1967Upgrade.contract, event: "UpgradedSecondary", logs: logs, sub: sub}, nil
}

// WatchUpgradedSecondary is a free log subscription operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) WatchUpgradedSecondary(opts *bind.WatchOpts, sink chan<- *DoubleLogicERC1967UpgradeUpgradedSecondary, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _DoubleLogicERC1967Upgrade.contract.WatchLogs(opts, "UpgradedSecondary", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DoubleLogicERC1967UpgradeUpgradedSecondary)
				if err := _DoubleLogicERC1967Upgrade.contract.UnpackLog(event, "UpgradedSecondary", log); err != nil {
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

// ParseUpgradedSecondary is a log parse operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_DoubleLogicERC1967Upgrade *DoubleLogicERC1967UpgradeFilterer) ParseUpgradedSecondary(log types.Log) (*DoubleLogicERC1967UpgradeUpgradedSecondary, error) {
	event := new(DoubleLogicERC1967UpgradeUpgradedSecondary)
	if err := _DoubleLogicERC1967Upgrade.contract.UnpackLog(event, "UpgradedSecondary", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DoubleLogicUUPSUpgradeableMetaData contains all meta data concerning the DoubleLogicUUPSUpgradeable contract.
var DoubleLogicUUPSUpgradeableMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"previousAdmin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"AdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"beacon\",\"type\":\"address\"}],\"name\":\"BeaconUpgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"UpgradedSecondary\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"proxiableUUID\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation\",\"type\":\"address\"}],\"name\":\"upgradeSecondaryTo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"upgradeSecondaryToAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation\",\"type\":\"address\"}],\"name\":\"upgradeTo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"upgradeToAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
}

// DoubleLogicUUPSUpgradeableABI is the input ABI used to generate the binding from.
// Deprecated: Use DoubleLogicUUPSUpgradeableMetaData.ABI instead.
var DoubleLogicUUPSUpgradeableABI = DoubleLogicUUPSUpgradeableMetaData.ABI

// DoubleLogicUUPSUpgradeable is an auto generated Go binding around an Ethereum contract.
type DoubleLogicUUPSUpgradeable struct {
	DoubleLogicUUPSUpgradeableCaller     // Read-only binding to the contract
	DoubleLogicUUPSUpgradeableTransactor // Write-only binding to the contract
	DoubleLogicUUPSUpgradeableFilterer   // Log filterer for contract events
}

// DoubleLogicUUPSUpgradeableCaller is an auto generated read-only Go binding around an Ethereum contract.
type DoubleLogicUUPSUpgradeableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DoubleLogicUUPSUpgradeableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DoubleLogicUUPSUpgradeableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DoubleLogicUUPSUpgradeableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DoubleLogicUUPSUpgradeableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DoubleLogicUUPSUpgradeableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DoubleLogicUUPSUpgradeableSession struct {
	Contract     *DoubleLogicUUPSUpgradeable // Generic contract binding to set the session for
	CallOpts     bind.CallOpts               // Call options to use throughout this session
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// DoubleLogicUUPSUpgradeableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DoubleLogicUUPSUpgradeableCallerSession struct {
	Contract *DoubleLogicUUPSUpgradeableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                     // Call options to use throughout this session
}

// DoubleLogicUUPSUpgradeableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DoubleLogicUUPSUpgradeableTransactorSession struct {
	Contract     *DoubleLogicUUPSUpgradeableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                     // Transaction auth options to use throughout this session
}

// DoubleLogicUUPSUpgradeableRaw is an auto generated low-level Go binding around an Ethereum contract.
type DoubleLogicUUPSUpgradeableRaw struct {
	Contract *DoubleLogicUUPSUpgradeable // Generic contract binding to access the raw methods on
}

// DoubleLogicUUPSUpgradeableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DoubleLogicUUPSUpgradeableCallerRaw struct {
	Contract *DoubleLogicUUPSUpgradeableCaller // Generic read-only contract binding to access the raw methods on
}

// DoubleLogicUUPSUpgradeableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DoubleLogicUUPSUpgradeableTransactorRaw struct {
	Contract *DoubleLogicUUPSUpgradeableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDoubleLogicUUPSUpgradeable creates a new instance of DoubleLogicUUPSUpgradeable, bound to a specific deployed contract.
func NewDoubleLogicUUPSUpgradeable(address common.Address, backend bind.ContractBackend) (*DoubleLogicUUPSUpgradeable, error) {
	contract, err := bindDoubleLogicUUPSUpgradeable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicUUPSUpgradeable{DoubleLogicUUPSUpgradeableCaller: DoubleLogicUUPSUpgradeableCaller{contract: contract}, DoubleLogicUUPSUpgradeableTransactor: DoubleLogicUUPSUpgradeableTransactor{contract: contract}, DoubleLogicUUPSUpgradeableFilterer: DoubleLogicUUPSUpgradeableFilterer{contract: contract}}, nil
}

// NewDoubleLogicUUPSUpgradeableCaller creates a new read-only instance of DoubleLogicUUPSUpgradeable, bound to a specific deployed contract.
func NewDoubleLogicUUPSUpgradeableCaller(address common.Address, caller bind.ContractCaller) (*DoubleLogicUUPSUpgradeableCaller, error) {
	contract, err := bindDoubleLogicUUPSUpgradeable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicUUPSUpgradeableCaller{contract: contract}, nil
}

// NewDoubleLogicUUPSUpgradeableTransactor creates a new write-only instance of DoubleLogicUUPSUpgradeable, bound to a specific deployed contract.
func NewDoubleLogicUUPSUpgradeableTransactor(address common.Address, transactor bind.ContractTransactor) (*DoubleLogicUUPSUpgradeableTransactor, error) {
	contract, err := bindDoubleLogicUUPSUpgradeable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicUUPSUpgradeableTransactor{contract: contract}, nil
}

// NewDoubleLogicUUPSUpgradeableFilterer creates a new log filterer instance of DoubleLogicUUPSUpgradeable, bound to a specific deployed contract.
func NewDoubleLogicUUPSUpgradeableFilterer(address common.Address, filterer bind.ContractFilterer) (*DoubleLogicUUPSUpgradeableFilterer, error) {
	contract, err := bindDoubleLogicUUPSUpgradeable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicUUPSUpgradeableFilterer{contract: contract}, nil
}

// bindDoubleLogicUUPSUpgradeable binds a generic wrapper to an already deployed contract.
func bindDoubleLogicUUPSUpgradeable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DoubleLogicUUPSUpgradeableMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DoubleLogicUUPSUpgradeable.Contract.DoubleLogicUUPSUpgradeableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.DoubleLogicUUPSUpgradeableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.DoubleLogicUUPSUpgradeableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DoubleLogicUUPSUpgradeable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.contract.Transact(opts, method, params...)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableCaller) ProxiableUUID(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _DoubleLogicUUPSUpgradeable.contract.Call(opts, &out, "proxiableUUID")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableSession) ProxiableUUID() ([32]byte, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.ProxiableUUID(&_DoubleLogicUUPSUpgradeable.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableCallerSession) ProxiableUUID() ([32]byte, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.ProxiableUUID(&_DoubleLogicUUPSUpgradeable.CallOpts)
}

// UpgradeSecondaryTo is a paid mutator transaction binding the contract method 0x0d40a0fd.
//
// Solidity: function upgradeSecondaryTo(address newImplementation) returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableTransactor) UpgradeSecondaryTo(opts *bind.TransactOpts, newImplementation common.Address) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.contract.Transact(opts, "upgradeSecondaryTo", newImplementation)
}

// UpgradeSecondaryTo is a paid mutator transaction binding the contract method 0x0d40a0fd.
//
// Solidity: function upgradeSecondaryTo(address newImplementation) returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableSession) UpgradeSecondaryTo(newImplementation common.Address) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.UpgradeSecondaryTo(&_DoubleLogicUUPSUpgradeable.TransactOpts, newImplementation)
}

// UpgradeSecondaryTo is a paid mutator transaction binding the contract method 0x0d40a0fd.
//
// Solidity: function upgradeSecondaryTo(address newImplementation) returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableTransactorSession) UpgradeSecondaryTo(newImplementation common.Address) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.UpgradeSecondaryTo(&_DoubleLogicUUPSUpgradeable.TransactOpts, newImplementation)
}

// UpgradeSecondaryToAndCall is a paid mutator transaction binding the contract method 0x9846129a.
//
// Solidity: function upgradeSecondaryToAndCall(address newImplementation, bytes data) payable returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableTransactor) UpgradeSecondaryToAndCall(opts *bind.TransactOpts, newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.contract.Transact(opts, "upgradeSecondaryToAndCall", newImplementation, data)
}

// UpgradeSecondaryToAndCall is a paid mutator transaction binding the contract method 0x9846129a.
//
// Solidity: function upgradeSecondaryToAndCall(address newImplementation, bytes data) payable returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableSession) UpgradeSecondaryToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.UpgradeSecondaryToAndCall(&_DoubleLogicUUPSUpgradeable.TransactOpts, newImplementation, data)
}

// UpgradeSecondaryToAndCall is a paid mutator transaction binding the contract method 0x9846129a.
//
// Solidity: function upgradeSecondaryToAndCall(address newImplementation, bytes data) payable returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableTransactorSession) UpgradeSecondaryToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.UpgradeSecondaryToAndCall(&_DoubleLogicUUPSUpgradeable.TransactOpts, newImplementation, data)
}

// UpgradeTo is a paid mutator transaction binding the contract method 0x3659cfe6.
//
// Solidity: function upgradeTo(address newImplementation) returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableTransactor) UpgradeTo(opts *bind.TransactOpts, newImplementation common.Address) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.contract.Transact(opts, "upgradeTo", newImplementation)
}

// UpgradeTo is a paid mutator transaction binding the contract method 0x3659cfe6.
//
// Solidity: function upgradeTo(address newImplementation) returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableSession) UpgradeTo(newImplementation common.Address) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.UpgradeTo(&_DoubleLogicUUPSUpgradeable.TransactOpts, newImplementation)
}

// UpgradeTo is a paid mutator transaction binding the contract method 0x3659cfe6.
//
// Solidity: function upgradeTo(address newImplementation) returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableTransactorSession) UpgradeTo(newImplementation common.Address) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.UpgradeTo(&_DoubleLogicUUPSUpgradeable.TransactOpts, newImplementation)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableTransactor) UpgradeToAndCall(opts *bind.TransactOpts, newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.contract.Transact(opts, "upgradeToAndCall", newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableSession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.UpgradeToAndCall(&_DoubleLogicUUPSUpgradeable.TransactOpts, newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableTransactorSession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _DoubleLogicUUPSUpgradeable.Contract.UpgradeToAndCall(&_DoubleLogicUUPSUpgradeable.TransactOpts, newImplementation, data)
}

// DoubleLogicUUPSUpgradeableAdminChangedIterator is returned from FilterAdminChanged and is used to iterate over the raw logs and unpacked data for AdminChanged events raised by the DoubleLogicUUPSUpgradeable contract.
type DoubleLogicUUPSUpgradeableAdminChangedIterator struct {
	Event *DoubleLogicUUPSUpgradeableAdminChanged // Event containing the contract specifics and raw log

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
func (it *DoubleLogicUUPSUpgradeableAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DoubleLogicUUPSUpgradeableAdminChanged)
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
		it.Event = new(DoubleLogicUUPSUpgradeableAdminChanged)
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
func (it *DoubleLogicUUPSUpgradeableAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DoubleLogicUUPSUpgradeableAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DoubleLogicUUPSUpgradeableAdminChanged represents a AdminChanged event raised by the DoubleLogicUUPSUpgradeable contract.
type DoubleLogicUUPSUpgradeableAdminChanged struct {
	PreviousAdmin common.Address
	NewAdmin      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterAdminChanged is a free log retrieval operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) FilterAdminChanged(opts *bind.FilterOpts) (*DoubleLogicUUPSUpgradeableAdminChangedIterator, error) {

	logs, sub, err := _DoubleLogicUUPSUpgradeable.contract.FilterLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return &DoubleLogicUUPSUpgradeableAdminChangedIterator{contract: _DoubleLogicUUPSUpgradeable.contract, event: "AdminChanged", logs: logs, sub: sub}, nil
}

// WatchAdminChanged is a free log subscription operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) WatchAdminChanged(opts *bind.WatchOpts, sink chan<- *DoubleLogicUUPSUpgradeableAdminChanged) (event.Subscription, error) {

	logs, sub, err := _DoubleLogicUUPSUpgradeable.contract.WatchLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DoubleLogicUUPSUpgradeableAdminChanged)
				if err := _DoubleLogicUUPSUpgradeable.contract.UnpackLog(event, "AdminChanged", log); err != nil {
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

// ParseAdminChanged is a log parse operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) ParseAdminChanged(log types.Log) (*DoubleLogicUUPSUpgradeableAdminChanged, error) {
	event := new(DoubleLogicUUPSUpgradeableAdminChanged)
	if err := _DoubleLogicUUPSUpgradeable.contract.UnpackLog(event, "AdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DoubleLogicUUPSUpgradeableBeaconUpgradedIterator is returned from FilterBeaconUpgraded and is used to iterate over the raw logs and unpacked data for BeaconUpgraded events raised by the DoubleLogicUUPSUpgradeable contract.
type DoubleLogicUUPSUpgradeableBeaconUpgradedIterator struct {
	Event *DoubleLogicUUPSUpgradeableBeaconUpgraded // Event containing the contract specifics and raw log

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
func (it *DoubleLogicUUPSUpgradeableBeaconUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DoubleLogicUUPSUpgradeableBeaconUpgraded)
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
		it.Event = new(DoubleLogicUUPSUpgradeableBeaconUpgraded)
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
func (it *DoubleLogicUUPSUpgradeableBeaconUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DoubleLogicUUPSUpgradeableBeaconUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DoubleLogicUUPSUpgradeableBeaconUpgraded represents a BeaconUpgraded event raised by the DoubleLogicUUPSUpgradeable contract.
type DoubleLogicUUPSUpgradeableBeaconUpgraded struct {
	Beacon common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBeaconUpgraded is a free log retrieval operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) FilterBeaconUpgraded(opts *bind.FilterOpts, beacon []common.Address) (*DoubleLogicUUPSUpgradeableBeaconUpgradedIterator, error) {

	var beaconRule []interface{}
	for _, beaconItem := range beacon {
		beaconRule = append(beaconRule, beaconItem)
	}

	logs, sub, err := _DoubleLogicUUPSUpgradeable.contract.FilterLogs(opts, "BeaconUpgraded", beaconRule)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicUUPSUpgradeableBeaconUpgradedIterator{contract: _DoubleLogicUUPSUpgradeable.contract, event: "BeaconUpgraded", logs: logs, sub: sub}, nil
}

// WatchBeaconUpgraded is a free log subscription operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) WatchBeaconUpgraded(opts *bind.WatchOpts, sink chan<- *DoubleLogicUUPSUpgradeableBeaconUpgraded, beacon []common.Address) (event.Subscription, error) {

	var beaconRule []interface{}
	for _, beaconItem := range beacon {
		beaconRule = append(beaconRule, beaconItem)
	}

	logs, sub, err := _DoubleLogicUUPSUpgradeable.contract.WatchLogs(opts, "BeaconUpgraded", beaconRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DoubleLogicUUPSUpgradeableBeaconUpgraded)
				if err := _DoubleLogicUUPSUpgradeable.contract.UnpackLog(event, "BeaconUpgraded", log); err != nil {
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

// ParseBeaconUpgraded is a log parse operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) ParseBeaconUpgraded(log types.Log) (*DoubleLogicUUPSUpgradeableBeaconUpgraded, error) {
	event := new(DoubleLogicUUPSUpgradeableBeaconUpgraded)
	if err := _DoubleLogicUUPSUpgradeable.contract.UnpackLog(event, "BeaconUpgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DoubleLogicUUPSUpgradeableUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the DoubleLogicUUPSUpgradeable contract.
type DoubleLogicUUPSUpgradeableUpgradedIterator struct {
	Event *DoubleLogicUUPSUpgradeableUpgraded // Event containing the contract specifics and raw log

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
func (it *DoubleLogicUUPSUpgradeableUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DoubleLogicUUPSUpgradeableUpgraded)
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
		it.Event = new(DoubleLogicUUPSUpgradeableUpgraded)
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
func (it *DoubleLogicUUPSUpgradeableUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DoubleLogicUUPSUpgradeableUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DoubleLogicUUPSUpgradeableUpgraded represents a Upgraded event raised by the DoubleLogicUUPSUpgradeable contract.
type DoubleLogicUUPSUpgradeableUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*DoubleLogicUUPSUpgradeableUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _DoubleLogicUUPSUpgradeable.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicUUPSUpgradeableUpgradedIterator{contract: _DoubleLogicUUPSUpgradeable.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *DoubleLogicUUPSUpgradeableUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _DoubleLogicUUPSUpgradeable.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DoubleLogicUUPSUpgradeableUpgraded)
				if err := _DoubleLogicUUPSUpgradeable.contract.UnpackLog(event, "Upgraded", log); err != nil {
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

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) ParseUpgraded(log types.Log) (*DoubleLogicUUPSUpgradeableUpgraded, error) {
	event := new(DoubleLogicUUPSUpgradeableUpgraded)
	if err := _DoubleLogicUUPSUpgradeable.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DoubleLogicUUPSUpgradeableUpgradedSecondaryIterator is returned from FilterUpgradedSecondary and is used to iterate over the raw logs and unpacked data for UpgradedSecondary events raised by the DoubleLogicUUPSUpgradeable contract.
type DoubleLogicUUPSUpgradeableUpgradedSecondaryIterator struct {
	Event *DoubleLogicUUPSUpgradeableUpgradedSecondary // Event containing the contract specifics and raw log

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
func (it *DoubleLogicUUPSUpgradeableUpgradedSecondaryIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DoubleLogicUUPSUpgradeableUpgradedSecondary)
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
		it.Event = new(DoubleLogicUUPSUpgradeableUpgradedSecondary)
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
func (it *DoubleLogicUUPSUpgradeableUpgradedSecondaryIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DoubleLogicUUPSUpgradeableUpgradedSecondaryIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DoubleLogicUUPSUpgradeableUpgradedSecondary represents a UpgradedSecondary event raised by the DoubleLogicUUPSUpgradeable contract.
type DoubleLogicUUPSUpgradeableUpgradedSecondary struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgradedSecondary is a free log retrieval operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) FilterUpgradedSecondary(opts *bind.FilterOpts, implementation []common.Address) (*DoubleLogicUUPSUpgradeableUpgradedSecondaryIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _DoubleLogicUUPSUpgradeable.contract.FilterLogs(opts, "UpgradedSecondary", implementationRule)
	if err != nil {
		return nil, err
	}
	return &DoubleLogicUUPSUpgradeableUpgradedSecondaryIterator{contract: _DoubleLogicUUPSUpgradeable.contract, event: "UpgradedSecondary", logs: logs, sub: sub}, nil
}

// WatchUpgradedSecondary is a free log subscription operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) WatchUpgradedSecondary(opts *bind.WatchOpts, sink chan<- *DoubleLogicUUPSUpgradeableUpgradedSecondary, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _DoubleLogicUUPSUpgradeable.contract.WatchLogs(opts, "UpgradedSecondary", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DoubleLogicUUPSUpgradeableUpgradedSecondary)
				if err := _DoubleLogicUUPSUpgradeable.contract.UnpackLog(event, "UpgradedSecondary", log); err != nil {
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

// ParseUpgradedSecondary is a log parse operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_DoubleLogicUUPSUpgradeable *DoubleLogicUUPSUpgradeableFilterer) ParseUpgradedSecondary(log types.Log) (*DoubleLogicUUPSUpgradeableUpgradedSecondary, error) {
	event := new(DoubleLogicUUPSUpgradeableUpgradedSecondary)
	if err := _DoubleLogicUUPSUpgradeable.contract.UnpackLog(event, "UpgradedSecondary", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GasRefundEnabledMetaData contains all meta data concerning the GasRefundEnabled contract.
var GasRefundEnabledMetaData = &bind.MetaData{
	ABI: "[]",
}

// GasRefundEnabledABI is the input ABI used to generate the binding from.
// Deprecated: Use GasRefundEnabledMetaData.ABI instead.
var GasRefundEnabledABI = GasRefundEnabledMetaData.ABI

// GasRefundEnabled is an auto generated Go binding around an Ethereum contract.
type GasRefundEnabled struct {
	GasRefundEnabledCaller     // Read-only binding to the contract
	GasRefundEnabledTransactor // Write-only binding to the contract
	GasRefundEnabledFilterer   // Log filterer for contract events
}

// GasRefundEnabledCaller is an auto generated read-only Go binding around an Ethereum contract.
type GasRefundEnabledCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GasRefundEnabledTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GasRefundEnabledTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GasRefundEnabledFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GasRefundEnabledFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GasRefundEnabledSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GasRefundEnabledSession struct {
	Contract     *GasRefundEnabled // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// GasRefundEnabledCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GasRefundEnabledCallerSession struct {
	Contract *GasRefundEnabledCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// GasRefundEnabledTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GasRefundEnabledTransactorSession struct {
	Contract     *GasRefundEnabledTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// GasRefundEnabledRaw is an auto generated low-level Go binding around an Ethereum contract.
type GasRefundEnabledRaw struct {
	Contract *GasRefundEnabled // Generic contract binding to access the raw methods on
}

// GasRefundEnabledCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GasRefundEnabledCallerRaw struct {
	Contract *GasRefundEnabledCaller // Generic read-only contract binding to access the raw methods on
}

// GasRefundEnabledTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GasRefundEnabledTransactorRaw struct {
	Contract *GasRefundEnabledTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGasRefundEnabled creates a new instance of GasRefundEnabled, bound to a specific deployed contract.
func NewGasRefundEnabled(address common.Address, backend bind.ContractBackend) (*GasRefundEnabled, error) {
	contract, err := bindGasRefundEnabled(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &GasRefundEnabled{GasRefundEnabledCaller: GasRefundEnabledCaller{contract: contract}, GasRefundEnabledTransactor: GasRefundEnabledTransactor{contract: contract}, GasRefundEnabledFilterer: GasRefundEnabledFilterer{contract: contract}}, nil
}

// NewGasRefundEnabledCaller creates a new read-only instance of GasRefundEnabled, bound to a specific deployed contract.
func NewGasRefundEnabledCaller(address common.Address, caller bind.ContractCaller) (*GasRefundEnabledCaller, error) {
	contract, err := bindGasRefundEnabled(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GasRefundEnabledCaller{contract: contract}, nil
}

// NewGasRefundEnabledTransactor creates a new write-only instance of GasRefundEnabled, bound to a specific deployed contract.
func NewGasRefundEnabledTransactor(address common.Address, transactor bind.ContractTransactor) (*GasRefundEnabledTransactor, error) {
	contract, err := bindGasRefundEnabled(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GasRefundEnabledTransactor{contract: contract}, nil
}

// NewGasRefundEnabledFilterer creates a new log filterer instance of GasRefundEnabled, bound to a specific deployed contract.
func NewGasRefundEnabledFilterer(address common.Address, filterer bind.ContractFilterer) (*GasRefundEnabledFilterer, error) {
	contract, err := bindGasRefundEnabled(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GasRefundEnabledFilterer{contract: contract}, nil
}

// bindGasRefundEnabled binds a generic wrapper to an already deployed contract.
func bindGasRefundEnabled(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := GasRefundEnabledMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GasRefundEnabled *GasRefundEnabledRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GasRefundEnabled.Contract.GasRefundEnabledCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GasRefundEnabled *GasRefundEnabledRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GasRefundEnabled.Contract.GasRefundEnabledTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GasRefundEnabled *GasRefundEnabledRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GasRefundEnabled.Contract.GasRefundEnabledTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GasRefundEnabled *GasRefundEnabledCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GasRefundEnabled.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GasRefundEnabled *GasRefundEnabledTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GasRefundEnabled.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GasRefundEnabled *GasRefundEnabledTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GasRefundEnabled.Contract.contract.Transact(opts, method, params...)
}

// IGasRefunderMetaData contains all meta data concerning the IGasRefunder contract.
var IGasRefunderMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"gasUsed\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"calldataSize\",\"type\":\"uint256\"}],\"name\":\"onGasSpent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IGasRefunderABI is the input ABI used to generate the binding from.
// Deprecated: Use IGasRefunderMetaData.ABI instead.
var IGasRefunderABI = IGasRefunderMetaData.ABI

// IGasRefunder is an auto generated Go binding around an Ethereum contract.
type IGasRefunder struct {
	IGasRefunderCaller     // Read-only binding to the contract
	IGasRefunderTransactor // Write-only binding to the contract
	IGasRefunderFilterer   // Log filterer for contract events
}

// IGasRefunderCaller is an auto generated read-only Go binding around an Ethereum contract.
type IGasRefunderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IGasRefunderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IGasRefunderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IGasRefunderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IGasRefunderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IGasRefunderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IGasRefunderSession struct {
	Contract     *IGasRefunder     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IGasRefunderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IGasRefunderCallerSession struct {
	Contract *IGasRefunderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// IGasRefunderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IGasRefunderTransactorSession struct {
	Contract     *IGasRefunderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// IGasRefunderRaw is an auto generated low-level Go binding around an Ethereum contract.
type IGasRefunderRaw struct {
	Contract *IGasRefunder // Generic contract binding to access the raw methods on
}

// IGasRefunderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IGasRefunderCallerRaw struct {
	Contract *IGasRefunderCaller // Generic read-only contract binding to access the raw methods on
}

// IGasRefunderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IGasRefunderTransactorRaw struct {
	Contract *IGasRefunderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIGasRefunder creates a new instance of IGasRefunder, bound to a specific deployed contract.
func NewIGasRefunder(address common.Address, backend bind.ContractBackend) (*IGasRefunder, error) {
	contract, err := bindIGasRefunder(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IGasRefunder{IGasRefunderCaller: IGasRefunderCaller{contract: contract}, IGasRefunderTransactor: IGasRefunderTransactor{contract: contract}, IGasRefunderFilterer: IGasRefunderFilterer{contract: contract}}, nil
}

// NewIGasRefunderCaller creates a new read-only instance of IGasRefunder, bound to a specific deployed contract.
func NewIGasRefunderCaller(address common.Address, caller bind.ContractCaller) (*IGasRefunderCaller, error) {
	contract, err := bindIGasRefunder(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IGasRefunderCaller{contract: contract}, nil
}

// NewIGasRefunderTransactor creates a new write-only instance of IGasRefunder, bound to a specific deployed contract.
func NewIGasRefunderTransactor(address common.Address, transactor bind.ContractTransactor) (*IGasRefunderTransactor, error) {
	contract, err := bindIGasRefunder(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IGasRefunderTransactor{contract: contract}, nil
}

// NewIGasRefunderFilterer creates a new log filterer instance of IGasRefunder, bound to a specific deployed contract.
func NewIGasRefunderFilterer(address common.Address, filterer bind.ContractFilterer) (*IGasRefunderFilterer, error) {
	contract, err := bindIGasRefunder(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IGasRefunderFilterer{contract: contract}, nil
}

// bindIGasRefunder binds a generic wrapper to an already deployed contract.
func bindIGasRefunder(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IGasRefunderMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IGasRefunder *IGasRefunderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IGasRefunder.Contract.IGasRefunderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IGasRefunder *IGasRefunderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IGasRefunder.Contract.IGasRefunderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IGasRefunder *IGasRefunderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IGasRefunder.Contract.IGasRefunderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IGasRefunder *IGasRefunderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IGasRefunder.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IGasRefunder *IGasRefunderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IGasRefunder.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IGasRefunder *IGasRefunderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IGasRefunder.Contract.contract.Transact(opts, method, params...)
}

// OnGasSpent is a paid mutator transaction binding the contract method 0xe3db8a49.
//
// Solidity: function onGasSpent(address spender, uint256 gasUsed, uint256 calldataSize) returns(bool success)
func (_IGasRefunder *IGasRefunderTransactor) OnGasSpent(opts *bind.TransactOpts, spender common.Address, gasUsed *big.Int, calldataSize *big.Int) (*types.Transaction, error) {
	return _IGasRefunder.contract.Transact(opts, "onGasSpent", spender, gasUsed, calldataSize)
}

// OnGasSpent is a paid mutator transaction binding the contract method 0xe3db8a49.
//
// Solidity: function onGasSpent(address spender, uint256 gasUsed, uint256 calldataSize) returns(bool success)
func (_IGasRefunder *IGasRefunderSession) OnGasSpent(spender common.Address, gasUsed *big.Int, calldataSize *big.Int) (*types.Transaction, error) {
	return _IGasRefunder.Contract.OnGasSpent(&_IGasRefunder.TransactOpts, spender, gasUsed, calldataSize)
}

// OnGasSpent is a paid mutator transaction binding the contract method 0xe3db8a49.
//
// Solidity: function onGasSpent(address spender, uint256 gasUsed, uint256 calldataSize) returns(bool success)
func (_IGasRefunder *IGasRefunderTransactorSession) OnGasSpent(spender common.Address, gasUsed *big.Int, calldataSize *big.Int) (*types.Transaction, error) {
	return _IGasRefunder.Contract.OnGasSpent(&_IGasRefunder.TransactOpts, spender, gasUsed, calldataSize)
}

// IReader4844MetaData contains all meta data concerning the IReader4844 contract.
var IReader4844MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"getBlobBaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getDataHashes\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IReader4844ABI is the input ABI used to generate the binding from.
// Deprecated: Use IReader4844MetaData.ABI instead.
var IReader4844ABI = IReader4844MetaData.ABI

// IReader4844 is an auto generated Go binding around an Ethereum contract.
type IReader4844 struct {
	IReader4844Caller     // Read-only binding to the contract
	IReader4844Transactor // Write-only binding to the contract
	IReader4844Filterer   // Log filterer for contract events
}

// IReader4844Caller is an auto generated read-only Go binding around an Ethereum contract.
type IReader4844Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IReader4844Transactor is an auto generated write-only Go binding around an Ethereum contract.
type IReader4844Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IReader4844Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IReader4844Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IReader4844Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IReader4844Session struct {
	Contract     *IReader4844      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IReader4844CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IReader4844CallerSession struct {
	Contract *IReader4844Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// IReader4844TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IReader4844TransactorSession struct {
	Contract     *IReader4844Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// IReader4844Raw is an auto generated low-level Go binding around an Ethereum contract.
type IReader4844Raw struct {
	Contract *IReader4844 // Generic contract binding to access the raw methods on
}

// IReader4844CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IReader4844CallerRaw struct {
	Contract *IReader4844Caller // Generic read-only contract binding to access the raw methods on
}

// IReader4844TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IReader4844TransactorRaw struct {
	Contract *IReader4844Transactor // Generic write-only contract binding to access the raw methods on
}

// NewIReader4844 creates a new instance of IReader4844, bound to a specific deployed contract.
func NewIReader4844(address common.Address, backend bind.ContractBackend) (*IReader4844, error) {
	contract, err := bindIReader4844(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IReader4844{IReader4844Caller: IReader4844Caller{contract: contract}, IReader4844Transactor: IReader4844Transactor{contract: contract}, IReader4844Filterer: IReader4844Filterer{contract: contract}}, nil
}

// NewIReader4844Caller creates a new read-only instance of IReader4844, bound to a specific deployed contract.
func NewIReader4844Caller(address common.Address, caller bind.ContractCaller) (*IReader4844Caller, error) {
	contract, err := bindIReader4844(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IReader4844Caller{contract: contract}, nil
}

// NewIReader4844Transactor creates a new write-only instance of IReader4844, bound to a specific deployed contract.
func NewIReader4844Transactor(address common.Address, transactor bind.ContractTransactor) (*IReader4844Transactor, error) {
	contract, err := bindIReader4844(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IReader4844Transactor{contract: contract}, nil
}

// NewIReader4844Filterer creates a new log filterer instance of IReader4844, bound to a specific deployed contract.
func NewIReader4844Filterer(address common.Address, filterer bind.ContractFilterer) (*IReader4844Filterer, error) {
	contract, err := bindIReader4844(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IReader4844Filterer{contract: contract}, nil
}

// bindIReader4844 binds a generic wrapper to an already deployed contract.
func bindIReader4844(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IReader4844MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IReader4844 *IReader4844Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IReader4844.Contract.IReader4844Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IReader4844 *IReader4844Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IReader4844.Contract.IReader4844Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IReader4844 *IReader4844Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IReader4844.Contract.IReader4844Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IReader4844 *IReader4844CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IReader4844.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IReader4844 *IReader4844TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IReader4844.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IReader4844 *IReader4844TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IReader4844.Contract.contract.Transact(opts, method, params...)
}

// GetBlobBaseFee is a free data retrieval call binding the contract method 0x1f6d6ef7.
//
// Solidity: function getBlobBaseFee() view returns(uint256)
func (_IReader4844 *IReader4844Caller) GetBlobBaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IReader4844.contract.Call(opts, &out, "getBlobBaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBlobBaseFee is a free data retrieval call binding the contract method 0x1f6d6ef7.
//
// Solidity: function getBlobBaseFee() view returns(uint256)
func (_IReader4844 *IReader4844Session) GetBlobBaseFee() (*big.Int, error) {
	return _IReader4844.Contract.GetBlobBaseFee(&_IReader4844.CallOpts)
}

// GetBlobBaseFee is a free data retrieval call binding the contract method 0x1f6d6ef7.
//
// Solidity: function getBlobBaseFee() view returns(uint256)
func (_IReader4844 *IReader4844CallerSession) GetBlobBaseFee() (*big.Int, error) {
	return _IReader4844.Contract.GetBlobBaseFee(&_IReader4844.CallOpts)
}

// GetDataHashes is a free data retrieval call binding the contract method 0xe83a2d82.
//
// Solidity: function getDataHashes() view returns(bytes32[])
func (_IReader4844 *IReader4844Caller) GetDataHashes(opts *bind.CallOpts) ([][32]byte, error) {
	var out []interface{}
	err := _IReader4844.contract.Call(opts, &out, "getDataHashes")

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetDataHashes is a free data retrieval call binding the contract method 0xe83a2d82.
//
// Solidity: function getDataHashes() view returns(bytes32[])
func (_IReader4844 *IReader4844Session) GetDataHashes() ([][32]byte, error) {
	return _IReader4844.Contract.GetDataHashes(&_IReader4844.CallOpts)
}

// GetDataHashes is a free data retrieval call binding the contract method 0xe83a2d82.
//
// Solidity: function getDataHashes() view returns(bytes32[])
func (_IReader4844 *IReader4844CallerSession) GetDataHashes() ([][32]byte, error) {
	return _IReader4844.Contract.GetDataHashes(&_IReader4844.CallOpts)
}

// MerkleLibMetaData contains all meta data concerning the MerkleLib contract.
var MerkleLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220d8ba33cb85457acc7fa0a70694265877c906b0737c0d44f7640ca6c62470c3c864736f6c63430008110033",
}

// MerkleLibABI is the input ABI used to generate the binding from.
// Deprecated: Use MerkleLibMetaData.ABI instead.
var MerkleLibABI = MerkleLibMetaData.ABI

// MerkleLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MerkleLibMetaData.Bin instead.
var MerkleLibBin = MerkleLibMetaData.Bin

// DeployMerkleLib deploys a new Ethereum contract, binding an instance of MerkleLib to it.
func DeployMerkleLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MerkleLib, error) {
	parsed, err := MerkleLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MerkleLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MerkleLib{MerkleLibCaller: MerkleLibCaller{contract: contract}, MerkleLibTransactor: MerkleLibTransactor{contract: contract}, MerkleLibFilterer: MerkleLibFilterer{contract: contract}}, nil
}

// MerkleLib is an auto generated Go binding around an Ethereum contract.
type MerkleLib struct {
	MerkleLibCaller     // Read-only binding to the contract
	MerkleLibTransactor // Write-only binding to the contract
	MerkleLibFilterer   // Log filterer for contract events
}

// MerkleLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type MerkleLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MerkleLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MerkleLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MerkleLibSession struct {
	Contract     *MerkleLib        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MerkleLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MerkleLibCallerSession struct {
	Contract *MerkleLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// MerkleLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MerkleLibTransactorSession struct {
	Contract     *MerkleLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// MerkleLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type MerkleLibRaw struct {
	Contract *MerkleLib // Generic contract binding to access the raw methods on
}

// MerkleLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MerkleLibCallerRaw struct {
	Contract *MerkleLibCaller // Generic read-only contract binding to access the raw methods on
}

// MerkleLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MerkleLibTransactorRaw struct {
	Contract *MerkleLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMerkleLib creates a new instance of MerkleLib, bound to a specific deployed contract.
func NewMerkleLib(address common.Address, backend bind.ContractBackend) (*MerkleLib, error) {
	contract, err := bindMerkleLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MerkleLib{MerkleLibCaller: MerkleLibCaller{contract: contract}, MerkleLibTransactor: MerkleLibTransactor{contract: contract}, MerkleLibFilterer: MerkleLibFilterer{contract: contract}}, nil
}

// NewMerkleLibCaller creates a new read-only instance of MerkleLib, bound to a specific deployed contract.
func NewMerkleLibCaller(address common.Address, caller bind.ContractCaller) (*MerkleLibCaller, error) {
	contract, err := bindMerkleLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MerkleLibCaller{contract: contract}, nil
}

// NewMerkleLibTransactor creates a new write-only instance of MerkleLib, bound to a specific deployed contract.
func NewMerkleLibTransactor(address common.Address, transactor bind.ContractTransactor) (*MerkleLibTransactor, error) {
	contract, err := bindMerkleLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MerkleLibTransactor{contract: contract}, nil
}

// NewMerkleLibFilterer creates a new log filterer instance of MerkleLib, bound to a specific deployed contract.
func NewMerkleLibFilterer(address common.Address, filterer bind.ContractFilterer) (*MerkleLibFilterer, error) {
	contract, err := bindMerkleLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MerkleLibFilterer{contract: contract}, nil
}

// bindMerkleLib binds a generic wrapper to an already deployed contract.
func bindMerkleLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MerkleLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MerkleLib *MerkleLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MerkleLib.Contract.MerkleLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MerkleLib *MerkleLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MerkleLib.Contract.MerkleLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MerkleLib *MerkleLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MerkleLib.Contract.MerkleLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MerkleLib *MerkleLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MerkleLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MerkleLib *MerkleLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MerkleLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MerkleLib *MerkleLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MerkleLib.Contract.contract.Transact(opts, method, params...)
}

// UUPSNotUpgradeableMetaData contains all meta data concerning the UUPSNotUpgradeable contract.
var UUPSNotUpgradeableMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"previousAdmin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"AdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"beacon\",\"type\":\"address\"}],\"name\":\"BeaconUpgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"UpgradedSecondary\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"proxiableUUID\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// UUPSNotUpgradeableABI is the input ABI used to generate the binding from.
// Deprecated: Use UUPSNotUpgradeableMetaData.ABI instead.
var UUPSNotUpgradeableABI = UUPSNotUpgradeableMetaData.ABI

// UUPSNotUpgradeable is an auto generated Go binding around an Ethereum contract.
type UUPSNotUpgradeable struct {
	UUPSNotUpgradeableCaller     // Read-only binding to the contract
	UUPSNotUpgradeableTransactor // Write-only binding to the contract
	UUPSNotUpgradeableFilterer   // Log filterer for contract events
}

// UUPSNotUpgradeableCaller is an auto generated read-only Go binding around an Ethereum contract.
type UUPSNotUpgradeableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UUPSNotUpgradeableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type UUPSNotUpgradeableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UUPSNotUpgradeableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type UUPSNotUpgradeableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UUPSNotUpgradeableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type UUPSNotUpgradeableSession struct {
	Contract     *UUPSNotUpgradeable // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// UUPSNotUpgradeableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type UUPSNotUpgradeableCallerSession struct {
	Contract *UUPSNotUpgradeableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// UUPSNotUpgradeableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type UUPSNotUpgradeableTransactorSession struct {
	Contract     *UUPSNotUpgradeableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// UUPSNotUpgradeableRaw is an auto generated low-level Go binding around an Ethereum contract.
type UUPSNotUpgradeableRaw struct {
	Contract *UUPSNotUpgradeable // Generic contract binding to access the raw methods on
}

// UUPSNotUpgradeableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type UUPSNotUpgradeableCallerRaw struct {
	Contract *UUPSNotUpgradeableCaller // Generic read-only contract binding to access the raw methods on
}

// UUPSNotUpgradeableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type UUPSNotUpgradeableTransactorRaw struct {
	Contract *UUPSNotUpgradeableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewUUPSNotUpgradeable creates a new instance of UUPSNotUpgradeable, bound to a specific deployed contract.
func NewUUPSNotUpgradeable(address common.Address, backend bind.ContractBackend) (*UUPSNotUpgradeable, error) {
	contract, err := bindUUPSNotUpgradeable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &UUPSNotUpgradeable{UUPSNotUpgradeableCaller: UUPSNotUpgradeableCaller{contract: contract}, UUPSNotUpgradeableTransactor: UUPSNotUpgradeableTransactor{contract: contract}, UUPSNotUpgradeableFilterer: UUPSNotUpgradeableFilterer{contract: contract}}, nil
}

// NewUUPSNotUpgradeableCaller creates a new read-only instance of UUPSNotUpgradeable, bound to a specific deployed contract.
func NewUUPSNotUpgradeableCaller(address common.Address, caller bind.ContractCaller) (*UUPSNotUpgradeableCaller, error) {
	contract, err := bindUUPSNotUpgradeable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &UUPSNotUpgradeableCaller{contract: contract}, nil
}

// NewUUPSNotUpgradeableTransactor creates a new write-only instance of UUPSNotUpgradeable, bound to a specific deployed contract.
func NewUUPSNotUpgradeableTransactor(address common.Address, transactor bind.ContractTransactor) (*UUPSNotUpgradeableTransactor, error) {
	contract, err := bindUUPSNotUpgradeable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &UUPSNotUpgradeableTransactor{contract: contract}, nil
}

// NewUUPSNotUpgradeableFilterer creates a new log filterer instance of UUPSNotUpgradeable, bound to a specific deployed contract.
func NewUUPSNotUpgradeableFilterer(address common.Address, filterer bind.ContractFilterer) (*UUPSNotUpgradeableFilterer, error) {
	contract, err := bindUUPSNotUpgradeable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &UUPSNotUpgradeableFilterer{contract: contract}, nil
}

// bindUUPSNotUpgradeable binds a generic wrapper to an already deployed contract.
func bindUUPSNotUpgradeable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := UUPSNotUpgradeableMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_UUPSNotUpgradeable *UUPSNotUpgradeableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _UUPSNotUpgradeable.Contract.UUPSNotUpgradeableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_UUPSNotUpgradeable *UUPSNotUpgradeableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _UUPSNotUpgradeable.Contract.UUPSNotUpgradeableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_UUPSNotUpgradeable *UUPSNotUpgradeableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _UUPSNotUpgradeable.Contract.UUPSNotUpgradeableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_UUPSNotUpgradeable *UUPSNotUpgradeableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _UUPSNotUpgradeable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_UUPSNotUpgradeable *UUPSNotUpgradeableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _UUPSNotUpgradeable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_UUPSNotUpgradeable *UUPSNotUpgradeableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _UUPSNotUpgradeable.Contract.contract.Transact(opts, method, params...)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableCaller) ProxiableUUID(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _UUPSNotUpgradeable.contract.Call(opts, &out, "proxiableUUID")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableSession) ProxiableUUID() ([32]byte, error) {
	return _UUPSNotUpgradeable.Contract.ProxiableUUID(&_UUPSNotUpgradeable.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableCallerSession) ProxiableUUID() ([32]byte, error) {
	return _UUPSNotUpgradeable.Contract.ProxiableUUID(&_UUPSNotUpgradeable.CallOpts)
}

// UUPSNotUpgradeableAdminChangedIterator is returned from FilterAdminChanged and is used to iterate over the raw logs and unpacked data for AdminChanged events raised by the UUPSNotUpgradeable contract.
type UUPSNotUpgradeableAdminChangedIterator struct {
	Event *UUPSNotUpgradeableAdminChanged // Event containing the contract specifics and raw log

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
func (it *UUPSNotUpgradeableAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UUPSNotUpgradeableAdminChanged)
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
		it.Event = new(UUPSNotUpgradeableAdminChanged)
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
func (it *UUPSNotUpgradeableAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UUPSNotUpgradeableAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UUPSNotUpgradeableAdminChanged represents a AdminChanged event raised by the UUPSNotUpgradeable contract.
type UUPSNotUpgradeableAdminChanged struct {
	PreviousAdmin common.Address
	NewAdmin      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterAdminChanged is a free log retrieval operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) FilterAdminChanged(opts *bind.FilterOpts) (*UUPSNotUpgradeableAdminChangedIterator, error) {

	logs, sub, err := _UUPSNotUpgradeable.contract.FilterLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return &UUPSNotUpgradeableAdminChangedIterator{contract: _UUPSNotUpgradeable.contract, event: "AdminChanged", logs: logs, sub: sub}, nil
}

// WatchAdminChanged is a free log subscription operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) WatchAdminChanged(opts *bind.WatchOpts, sink chan<- *UUPSNotUpgradeableAdminChanged) (event.Subscription, error) {

	logs, sub, err := _UUPSNotUpgradeable.contract.WatchLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UUPSNotUpgradeableAdminChanged)
				if err := _UUPSNotUpgradeable.contract.UnpackLog(event, "AdminChanged", log); err != nil {
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

// ParseAdminChanged is a log parse operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) ParseAdminChanged(log types.Log) (*UUPSNotUpgradeableAdminChanged, error) {
	event := new(UUPSNotUpgradeableAdminChanged)
	if err := _UUPSNotUpgradeable.contract.UnpackLog(event, "AdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UUPSNotUpgradeableBeaconUpgradedIterator is returned from FilterBeaconUpgraded and is used to iterate over the raw logs and unpacked data for BeaconUpgraded events raised by the UUPSNotUpgradeable contract.
type UUPSNotUpgradeableBeaconUpgradedIterator struct {
	Event *UUPSNotUpgradeableBeaconUpgraded // Event containing the contract specifics and raw log

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
func (it *UUPSNotUpgradeableBeaconUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UUPSNotUpgradeableBeaconUpgraded)
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
		it.Event = new(UUPSNotUpgradeableBeaconUpgraded)
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
func (it *UUPSNotUpgradeableBeaconUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UUPSNotUpgradeableBeaconUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UUPSNotUpgradeableBeaconUpgraded represents a BeaconUpgraded event raised by the UUPSNotUpgradeable contract.
type UUPSNotUpgradeableBeaconUpgraded struct {
	Beacon common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBeaconUpgraded is a free log retrieval operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) FilterBeaconUpgraded(opts *bind.FilterOpts, beacon []common.Address) (*UUPSNotUpgradeableBeaconUpgradedIterator, error) {

	var beaconRule []interface{}
	for _, beaconItem := range beacon {
		beaconRule = append(beaconRule, beaconItem)
	}

	logs, sub, err := _UUPSNotUpgradeable.contract.FilterLogs(opts, "BeaconUpgraded", beaconRule)
	if err != nil {
		return nil, err
	}
	return &UUPSNotUpgradeableBeaconUpgradedIterator{contract: _UUPSNotUpgradeable.contract, event: "BeaconUpgraded", logs: logs, sub: sub}, nil
}

// WatchBeaconUpgraded is a free log subscription operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) WatchBeaconUpgraded(opts *bind.WatchOpts, sink chan<- *UUPSNotUpgradeableBeaconUpgraded, beacon []common.Address) (event.Subscription, error) {

	var beaconRule []interface{}
	for _, beaconItem := range beacon {
		beaconRule = append(beaconRule, beaconItem)
	}

	logs, sub, err := _UUPSNotUpgradeable.contract.WatchLogs(opts, "BeaconUpgraded", beaconRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UUPSNotUpgradeableBeaconUpgraded)
				if err := _UUPSNotUpgradeable.contract.UnpackLog(event, "BeaconUpgraded", log); err != nil {
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

// ParseBeaconUpgraded is a log parse operation binding the contract event 0x1cf3b03a6cf19fa2baba4df148e9dcabedea7f8a5c07840e207e5c089be95d3e.
//
// Solidity: event BeaconUpgraded(address indexed beacon)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) ParseBeaconUpgraded(log types.Log) (*UUPSNotUpgradeableBeaconUpgraded, error) {
	event := new(UUPSNotUpgradeableBeaconUpgraded)
	if err := _UUPSNotUpgradeable.contract.UnpackLog(event, "BeaconUpgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UUPSNotUpgradeableUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the UUPSNotUpgradeable contract.
type UUPSNotUpgradeableUpgradedIterator struct {
	Event *UUPSNotUpgradeableUpgraded // Event containing the contract specifics and raw log

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
func (it *UUPSNotUpgradeableUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UUPSNotUpgradeableUpgraded)
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
		it.Event = new(UUPSNotUpgradeableUpgraded)
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
func (it *UUPSNotUpgradeableUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UUPSNotUpgradeableUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UUPSNotUpgradeableUpgraded represents a Upgraded event raised by the UUPSNotUpgradeable contract.
type UUPSNotUpgradeableUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*UUPSNotUpgradeableUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _UUPSNotUpgradeable.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &UUPSNotUpgradeableUpgradedIterator{contract: _UUPSNotUpgradeable.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *UUPSNotUpgradeableUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _UUPSNotUpgradeable.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UUPSNotUpgradeableUpgraded)
				if err := _UUPSNotUpgradeable.contract.UnpackLog(event, "Upgraded", log); err != nil {
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

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) ParseUpgraded(log types.Log) (*UUPSNotUpgradeableUpgraded, error) {
	event := new(UUPSNotUpgradeableUpgraded)
	if err := _UUPSNotUpgradeable.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UUPSNotUpgradeableUpgradedSecondaryIterator is returned from FilterUpgradedSecondary and is used to iterate over the raw logs and unpacked data for UpgradedSecondary events raised by the UUPSNotUpgradeable contract.
type UUPSNotUpgradeableUpgradedSecondaryIterator struct {
	Event *UUPSNotUpgradeableUpgradedSecondary // Event containing the contract specifics and raw log

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
func (it *UUPSNotUpgradeableUpgradedSecondaryIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UUPSNotUpgradeableUpgradedSecondary)
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
		it.Event = new(UUPSNotUpgradeableUpgradedSecondary)
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
func (it *UUPSNotUpgradeableUpgradedSecondaryIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UUPSNotUpgradeableUpgradedSecondaryIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UUPSNotUpgradeableUpgradedSecondary represents a UpgradedSecondary event raised by the UUPSNotUpgradeable contract.
type UUPSNotUpgradeableUpgradedSecondary struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgradedSecondary is a free log retrieval operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) FilterUpgradedSecondary(opts *bind.FilterOpts, implementation []common.Address) (*UUPSNotUpgradeableUpgradedSecondaryIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _UUPSNotUpgradeable.contract.FilterLogs(opts, "UpgradedSecondary", implementationRule)
	if err != nil {
		return nil, err
	}
	return &UUPSNotUpgradeableUpgradedSecondaryIterator{contract: _UUPSNotUpgradeable.contract, event: "UpgradedSecondary", logs: logs, sub: sub}, nil
}

// WatchUpgradedSecondary is a free log subscription operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) WatchUpgradedSecondary(opts *bind.WatchOpts, sink chan<- *UUPSNotUpgradeableUpgradedSecondary, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _UUPSNotUpgradeable.contract.WatchLogs(opts, "UpgradedSecondary", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UUPSNotUpgradeableUpgradedSecondary)
				if err := _UUPSNotUpgradeable.contract.UnpackLog(event, "UpgradedSecondary", log); err != nil {
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

// ParseUpgradedSecondary is a log parse operation binding the contract event 0xf7eed2a7fabbf1bec8d55ed5e785cc76622376dde5df4ff15470551e030b8134.
//
// Solidity: event UpgradedSecondary(address indexed implementation)
func (_UUPSNotUpgradeable *UUPSNotUpgradeableFilterer) ParseUpgradedSecondary(log types.Log) (*UUPSNotUpgradeableUpgradedSecondary, error) {
	event := new(UUPSNotUpgradeableUpgradedSecondary)
	if err := _UUPSNotUpgradeable.contract.UnpackLog(event, "UpgradedSecondary", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
