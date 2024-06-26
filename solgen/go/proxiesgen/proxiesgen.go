// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package proxiesgen

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

// IProxyMetaData contains all meta data concerning the IProxy contract.
var IProxyMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"masterCopy\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IProxyABI is the input ABI used to generate the binding from.
// Deprecated: Use IProxyMetaData.ABI instead.
var IProxyABI = IProxyMetaData.ABI

// IProxy is an auto generated Go binding around an Ethereum contract.
type IProxy struct {
	IProxyCaller     // Read-only binding to the contract
	IProxyTransactor // Write-only binding to the contract
	IProxyFilterer   // Log filterer for contract events
}

// IProxyCaller is an auto generated read-only Go binding around an Ethereum contract.
type IProxyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IProxyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IProxyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IProxyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IProxyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IProxySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IProxySession struct {
	Contract     *IProxy           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IProxyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IProxyCallerSession struct {
	Contract *IProxyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// IProxyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IProxyTransactorSession struct {
	Contract     *IProxyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IProxyRaw is an auto generated low-level Go binding around an Ethereum contract.
type IProxyRaw struct {
	Contract *IProxy // Generic contract binding to access the raw methods on
}

// IProxyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IProxyCallerRaw struct {
	Contract *IProxyCaller // Generic read-only contract binding to access the raw methods on
}

// IProxyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IProxyTransactorRaw struct {
	Contract *IProxyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIProxy creates a new instance of IProxy, bound to a specific deployed contract.
func NewIProxy(address common.Address, backend bind.ContractBackend) (*IProxy, error) {
	contract, err := bindIProxy(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IProxy{IProxyCaller: IProxyCaller{contract: contract}, IProxyTransactor: IProxyTransactor{contract: contract}, IProxyFilterer: IProxyFilterer{contract: contract}}, nil
}

// NewIProxyCaller creates a new read-only instance of IProxy, bound to a specific deployed contract.
func NewIProxyCaller(address common.Address, caller bind.ContractCaller) (*IProxyCaller, error) {
	contract, err := bindIProxy(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IProxyCaller{contract: contract}, nil
}

// NewIProxyTransactor creates a new write-only instance of IProxy, bound to a specific deployed contract.
func NewIProxyTransactor(address common.Address, transactor bind.ContractTransactor) (*IProxyTransactor, error) {
	contract, err := bindIProxy(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IProxyTransactor{contract: contract}, nil
}

// NewIProxyFilterer creates a new log filterer instance of IProxy, bound to a specific deployed contract.
func NewIProxyFilterer(address common.Address, filterer bind.ContractFilterer) (*IProxyFilterer, error) {
	contract, err := bindIProxy(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IProxyFilterer{contract: contract}, nil
}

// bindIProxy binds a generic wrapper to an already deployed contract.
func bindIProxy(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IProxyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IProxy *IProxyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IProxy.Contract.IProxyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IProxy *IProxyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IProxy.Contract.IProxyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IProxy *IProxyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IProxy.Contract.IProxyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IProxy *IProxyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IProxy.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IProxy *IProxyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IProxy.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IProxy *IProxyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IProxy.Contract.contract.Transact(opts, method, params...)
}

// MasterCopy is a free data retrieval call binding the contract method 0xa619486e.
//
// Solidity: function masterCopy() view returns(address)
func (_IProxy *IProxyCaller) MasterCopy(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IProxy.contract.Call(opts, &out, "masterCopy")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MasterCopy is a free data retrieval call binding the contract method 0xa619486e.
//
// Solidity: function masterCopy() view returns(address)
func (_IProxy *IProxySession) MasterCopy() (common.Address, error) {
	return _IProxy.Contract.MasterCopy(&_IProxy.CallOpts)
}

// MasterCopy is a free data retrieval call binding the contract method 0xa619486e.
//
// Solidity: function masterCopy() view returns(address)
func (_IProxy *IProxyCallerSession) MasterCopy() (common.Address, error) {
	return _IProxy.Contract.MasterCopy(&_IProxy.CallOpts)
}

// IProxyCreationCallbackMetaData contains all meta data concerning the IProxyCreationCallback contract.
var IProxyCreationCallbackMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractSafeProxy\",\"name\":\"proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_singleton\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"initializer\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"saltNonce\",\"type\":\"uint256\"}],\"name\":\"proxyCreated\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IProxyCreationCallbackABI is the input ABI used to generate the binding from.
// Deprecated: Use IProxyCreationCallbackMetaData.ABI instead.
var IProxyCreationCallbackABI = IProxyCreationCallbackMetaData.ABI

// IProxyCreationCallback is an auto generated Go binding around an Ethereum contract.
type IProxyCreationCallback struct {
	IProxyCreationCallbackCaller     // Read-only binding to the contract
	IProxyCreationCallbackTransactor // Write-only binding to the contract
	IProxyCreationCallbackFilterer   // Log filterer for contract events
}

// IProxyCreationCallbackCaller is an auto generated read-only Go binding around an Ethereum contract.
type IProxyCreationCallbackCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IProxyCreationCallbackTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IProxyCreationCallbackTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IProxyCreationCallbackFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IProxyCreationCallbackFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IProxyCreationCallbackSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IProxyCreationCallbackSession struct {
	Contract     *IProxyCreationCallback // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// IProxyCreationCallbackCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IProxyCreationCallbackCallerSession struct {
	Contract *IProxyCreationCallbackCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// IProxyCreationCallbackTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IProxyCreationCallbackTransactorSession struct {
	Contract     *IProxyCreationCallbackTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// IProxyCreationCallbackRaw is an auto generated low-level Go binding around an Ethereum contract.
type IProxyCreationCallbackRaw struct {
	Contract *IProxyCreationCallback // Generic contract binding to access the raw methods on
}

// IProxyCreationCallbackCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IProxyCreationCallbackCallerRaw struct {
	Contract *IProxyCreationCallbackCaller // Generic read-only contract binding to access the raw methods on
}

// IProxyCreationCallbackTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IProxyCreationCallbackTransactorRaw struct {
	Contract *IProxyCreationCallbackTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIProxyCreationCallback creates a new instance of IProxyCreationCallback, bound to a specific deployed contract.
func NewIProxyCreationCallback(address common.Address, backend bind.ContractBackend) (*IProxyCreationCallback, error) {
	contract, err := bindIProxyCreationCallback(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IProxyCreationCallback{IProxyCreationCallbackCaller: IProxyCreationCallbackCaller{contract: contract}, IProxyCreationCallbackTransactor: IProxyCreationCallbackTransactor{contract: contract}, IProxyCreationCallbackFilterer: IProxyCreationCallbackFilterer{contract: contract}}, nil
}

// NewIProxyCreationCallbackCaller creates a new read-only instance of IProxyCreationCallback, bound to a specific deployed contract.
func NewIProxyCreationCallbackCaller(address common.Address, caller bind.ContractCaller) (*IProxyCreationCallbackCaller, error) {
	contract, err := bindIProxyCreationCallback(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IProxyCreationCallbackCaller{contract: contract}, nil
}

// NewIProxyCreationCallbackTransactor creates a new write-only instance of IProxyCreationCallback, bound to a specific deployed contract.
func NewIProxyCreationCallbackTransactor(address common.Address, transactor bind.ContractTransactor) (*IProxyCreationCallbackTransactor, error) {
	contract, err := bindIProxyCreationCallback(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IProxyCreationCallbackTransactor{contract: contract}, nil
}

// NewIProxyCreationCallbackFilterer creates a new log filterer instance of IProxyCreationCallback, bound to a specific deployed contract.
func NewIProxyCreationCallbackFilterer(address common.Address, filterer bind.ContractFilterer) (*IProxyCreationCallbackFilterer, error) {
	contract, err := bindIProxyCreationCallback(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IProxyCreationCallbackFilterer{contract: contract}, nil
}

// bindIProxyCreationCallback binds a generic wrapper to an already deployed contract.
func bindIProxyCreationCallback(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IProxyCreationCallbackMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IProxyCreationCallback *IProxyCreationCallbackRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IProxyCreationCallback.Contract.IProxyCreationCallbackCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IProxyCreationCallback *IProxyCreationCallbackRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IProxyCreationCallback.Contract.IProxyCreationCallbackTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IProxyCreationCallback *IProxyCreationCallbackRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IProxyCreationCallback.Contract.IProxyCreationCallbackTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IProxyCreationCallback *IProxyCreationCallbackCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IProxyCreationCallback.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IProxyCreationCallback *IProxyCreationCallbackTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IProxyCreationCallback.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IProxyCreationCallback *IProxyCreationCallbackTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IProxyCreationCallback.Contract.contract.Transact(opts, method, params...)
}

// ProxyCreated is a paid mutator transaction binding the contract method 0x1e52b518.
//
// Solidity: function proxyCreated(address proxy, address _singleton, bytes initializer, uint256 saltNonce) returns()
func (_IProxyCreationCallback *IProxyCreationCallbackTransactor) ProxyCreated(opts *bind.TransactOpts, proxy common.Address, _singleton common.Address, initializer []byte, saltNonce *big.Int) (*types.Transaction, error) {
	return _IProxyCreationCallback.contract.Transact(opts, "proxyCreated", proxy, _singleton, initializer, saltNonce)
}

// ProxyCreated is a paid mutator transaction binding the contract method 0x1e52b518.
//
// Solidity: function proxyCreated(address proxy, address _singleton, bytes initializer, uint256 saltNonce) returns()
func (_IProxyCreationCallback *IProxyCreationCallbackSession) ProxyCreated(proxy common.Address, _singleton common.Address, initializer []byte, saltNonce *big.Int) (*types.Transaction, error) {
	return _IProxyCreationCallback.Contract.ProxyCreated(&_IProxyCreationCallback.TransactOpts, proxy, _singleton, initializer, saltNonce)
}

// ProxyCreated is a paid mutator transaction binding the contract method 0x1e52b518.
//
// Solidity: function proxyCreated(address proxy, address _singleton, bytes initializer, uint256 saltNonce) returns()
func (_IProxyCreationCallback *IProxyCreationCallbackTransactorSession) ProxyCreated(proxy common.Address, _singleton common.Address, initializer []byte, saltNonce *big.Int) (*types.Transaction, error) {
	return _IProxyCreationCallback.Contract.ProxyCreated(&_IProxyCreationCallback.TransactOpts, proxy, _singleton, initializer, saltNonce)
}

// SafeProxyMetaData contains all meta data concerning the SafeProxy contract.
var SafeProxyMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_singleton\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"}]",
	Bin: "0x608060405234801561001057600080fd5b506040516101d63803806101d68339818101604052602081101561003357600080fd5b8101908080519060200190929190505050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614156100ca576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260228152602001806101b46022913960400191505060405180910390fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050609b806101196000396000f3fe60806040526000547fa619486e00000000000000000000000000000000000000000000000000000000600035141560405780600c1b600c1c60005260206000f35b3660008037600080366000845af43d6000803e60008114156060573d6000fd5b3d6000f3fea2646970667358221220bfbe5e66dfccd59d80684323ec36a561ddc5ef3b39a33a941f25cabefff21eb964736f6c63430007060033496e76616c69642073696e676c65746f6e20616464726573732070726f7669646564",
}

// SafeProxyABI is the input ABI used to generate the binding from.
// Deprecated: Use SafeProxyMetaData.ABI instead.
var SafeProxyABI = SafeProxyMetaData.ABI

// SafeProxyBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SafeProxyMetaData.Bin instead.
var SafeProxyBin = SafeProxyMetaData.Bin

// DeploySafeProxy deploys a new Ethereum contract, binding an instance of SafeProxy to it.
func DeploySafeProxy(auth *bind.TransactOpts, backend bind.ContractBackend, _singleton common.Address) (common.Address, *types.Transaction, *SafeProxy, error) {
	parsed, err := SafeProxyMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SafeProxyBin), backend, _singleton)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SafeProxy{SafeProxyCaller: SafeProxyCaller{contract: contract}, SafeProxyTransactor: SafeProxyTransactor{contract: contract}, SafeProxyFilterer: SafeProxyFilterer{contract: contract}}, nil
}

// SafeProxy is an auto generated Go binding around an Ethereum contract.
type SafeProxy struct {
	SafeProxyCaller     // Read-only binding to the contract
	SafeProxyTransactor // Write-only binding to the contract
	SafeProxyFilterer   // Log filterer for contract events
}

// SafeProxyCaller is an auto generated read-only Go binding around an Ethereum contract.
type SafeProxyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeProxyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SafeProxyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeProxyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SafeProxyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeProxySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SafeProxySession struct {
	Contract     *SafeProxy        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SafeProxyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SafeProxyCallerSession struct {
	Contract *SafeProxyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// SafeProxyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SafeProxyTransactorSession struct {
	Contract     *SafeProxyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// SafeProxyRaw is an auto generated low-level Go binding around an Ethereum contract.
type SafeProxyRaw struct {
	Contract *SafeProxy // Generic contract binding to access the raw methods on
}

// SafeProxyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SafeProxyCallerRaw struct {
	Contract *SafeProxyCaller // Generic read-only contract binding to access the raw methods on
}

// SafeProxyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SafeProxyTransactorRaw struct {
	Contract *SafeProxyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSafeProxy creates a new instance of SafeProxy, bound to a specific deployed contract.
func NewSafeProxy(address common.Address, backend bind.ContractBackend) (*SafeProxy, error) {
	contract, err := bindSafeProxy(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SafeProxy{SafeProxyCaller: SafeProxyCaller{contract: contract}, SafeProxyTransactor: SafeProxyTransactor{contract: contract}, SafeProxyFilterer: SafeProxyFilterer{contract: contract}}, nil
}

// NewSafeProxyCaller creates a new read-only instance of SafeProxy, bound to a specific deployed contract.
func NewSafeProxyCaller(address common.Address, caller bind.ContractCaller) (*SafeProxyCaller, error) {
	contract, err := bindSafeProxy(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SafeProxyCaller{contract: contract}, nil
}

// NewSafeProxyTransactor creates a new write-only instance of SafeProxy, bound to a specific deployed contract.
func NewSafeProxyTransactor(address common.Address, transactor bind.ContractTransactor) (*SafeProxyTransactor, error) {
	contract, err := bindSafeProxy(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SafeProxyTransactor{contract: contract}, nil
}

// NewSafeProxyFilterer creates a new log filterer instance of SafeProxy, bound to a specific deployed contract.
func NewSafeProxyFilterer(address common.Address, filterer bind.ContractFilterer) (*SafeProxyFilterer, error) {
	contract, err := bindSafeProxy(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SafeProxyFilterer{contract: contract}, nil
}

// bindSafeProxy binds a generic wrapper to an already deployed contract.
func bindSafeProxy(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SafeProxyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeProxy *SafeProxyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SafeProxy.Contract.SafeProxyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeProxy *SafeProxyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeProxy.Contract.SafeProxyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeProxy *SafeProxyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeProxy.Contract.SafeProxyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeProxy *SafeProxyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SafeProxy.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeProxy *SafeProxyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeProxy.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeProxy *SafeProxyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeProxy.Contract.contract.Transact(opts, method, params...)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SafeProxy *SafeProxyTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _SafeProxy.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SafeProxy *SafeProxySession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _SafeProxy.Contract.Fallback(&_SafeProxy.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SafeProxy *SafeProxyTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _SafeProxy.Contract.Fallback(&_SafeProxy.TransactOpts, calldata)
}

// SafeProxyFactoryMetaData contains all meta data concerning the SafeProxyFactory contract.
var SafeProxyFactoryMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"contractSafeProxy\",\"name\":\"proxy\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"singleton\",\"type\":\"address\"}],\"name\":\"ProxyCreation\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_singleton\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"initializer\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"saltNonce\",\"type\":\"uint256\"}],\"name\":\"createChainSpecificProxyWithNonce\",\"outputs\":[{\"internalType\":\"contractSafeProxy\",\"name\":\"proxy\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_singleton\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"initializer\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"saltNonce\",\"type\":\"uint256\"},{\"internalType\":\"contractIProxyCreationCallback\",\"name\":\"callback\",\"type\":\"address\"}],\"name\":\"createProxyWithCallback\",\"outputs\":[{\"internalType\":\"contractSafeProxy\",\"name\":\"proxy\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_singleton\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"initializer\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"saltNonce\",\"type\":\"uint256\"}],\"name\":\"createProxyWithNonce\",\"outputs\":[{\"internalType\":\"contractSafeProxy\",\"name\":\"proxy\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proxyCreationCode\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610bde806100206000396000f3fe608060405234801561001057600080fd5b50600436106100575760003560e01c80631688f0b91461005c5780633408e4701461016b57806353e5d93514610189578063d18af54d1461020c578063ec9e80bb1461033b575b600080fd5b61013f6004803603606081101561007257600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001906401000000008111156100af57600080fd5b8201836020820111156100c157600080fd5b803590602001918460018302840111640100000000831117156100e357600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192908035906020019092919050505061044a565b604051808273ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6101736104fe565b6040518082815260200191505060405180910390f35b61019161050b565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156101d15780820151818401526020810190506101b6565b50505050905090810190601f1680156101fe5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b61030f6004803603608081101561022257600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019064010000000081111561025f57600080fd5b82018360208201111561027157600080fd5b8035906020019184600183028401116401000000008311171561029357600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f82011690508083019250505050505050919291929080359060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610536565b604051808273ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b61041e6004803603606081101561035157600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019064010000000081111561038e57600080fd5b8201836020820111156103a057600080fd5b803590602001918460018302840111640100000000831117156103c257600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050509192919290803590602001909291905050506106e5565b604051808273ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b60008083805190602001208360405160200180838152602001828152602001925050506040516020818303038152906040528051906020012090506104908585836107a8565b91508173ffffffffffffffffffffffffffffffffffffffff167f4f51faf6c4561ff95f067657e43439f0f856d97c04d9ec9070a6199ad418e23586604051808273ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a2509392505050565b6000804690508091505090565b60606040518060200161051d906109c5565b6020820181038252601f19601f82011660405250905090565b6000808383604051602001808381526020018273ffffffffffffffffffffffffffffffffffffffff1660601b8152601401925050506040516020818303038152906040528051906020012060001c905061059186868361044a565b9150600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16146106dc578273ffffffffffffffffffffffffffffffffffffffff16631e52b518838888886040518563ffffffff1660e01b8152600401808573ffffffffffffffffffffffffffffffffffffffff1681526020018473ffffffffffffffffffffffffffffffffffffffff16815260200180602001838152602001828103825284818151815260200191508051906020019080838360005b83811015610674578082015181840152602081019050610659565b50505050905090810190601f1680156106a15780820380516001836020036101000a031916815260200191505b5095505050505050600060405180830381600087803b1580156106c357600080fd5b505af11580156106d7573d6000803e3d6000fd5b505050505b50949350505050565b6000808380519060200120836106f96104fe565b60405160200180848152602001838152602001828152602001935050505060405160208183030381529060405280519060200120905061073a8585836107a8565b91508173ffffffffffffffffffffffffffffffffffffffff167f4f51faf6c4561ff95f067657e43439f0f856d97c04d9ec9070a6199ad418e23586604051808273ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a2509392505050565b60006107b3846109b2565b610825576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601f8152602001807f53696e676c65746f6e20636f6e7472616374206e6f74206465706c6f7965640081525060200191505060405180910390fd5b600060405180602001610837906109c5565b6020820181038252601f19601f820116604052508573ffffffffffffffffffffffffffffffffffffffff166040516020018083805190602001908083835b602083106108985780518252602082019150602081019050602083039250610875565b6001836020036101000a038019825116818451168082178552505050505050905001828152602001925050506040516020818303038152906040529050828151826020016000f59150600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff161415610984576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260138152602001807f437265617465322063616c6c206661696c65640000000000000000000000000081525060200191505060405180910390fd5b6000845111156109aa5760008060008651602088016000875af114156109a957600080fd5b5b509392505050565b600080823b905060008111915050919050565b6101d6806109d38339019056fe608060405234801561001057600080fd5b506040516101d63803806101d68339818101604052602081101561003357600080fd5b8101908080519060200190929190505050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614156100ca576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260228152602001806101b46022913960400191505060405180910390fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050609b806101196000396000f3fe60806040526000547fa619486e00000000000000000000000000000000000000000000000000000000600035141560405780600c1b600c1c60005260206000f35b3660008037600080366000845af43d6000803e60008114156060573d6000fd5b3d6000f3fea2646970667358221220bfbe5e66dfccd59d80684323ec36a561ddc5ef3b39a33a941f25cabefff21eb964736f6c63430007060033496e76616c69642073696e676c65746f6e20616464726573732070726f7669646564a2646970667358221220149b0d7527b1b8b9ef516314484ca0dc26d512f1a355e1835863922f9dc5953564736f6c63430007060033",
}

// SafeProxyFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use SafeProxyFactoryMetaData.ABI instead.
var SafeProxyFactoryABI = SafeProxyFactoryMetaData.ABI

// SafeProxyFactoryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SafeProxyFactoryMetaData.Bin instead.
var SafeProxyFactoryBin = SafeProxyFactoryMetaData.Bin

// DeploySafeProxyFactory deploys a new Ethereum contract, binding an instance of SafeProxyFactory to it.
func DeploySafeProxyFactory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SafeProxyFactory, error) {
	parsed, err := SafeProxyFactoryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SafeProxyFactoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SafeProxyFactory{SafeProxyFactoryCaller: SafeProxyFactoryCaller{contract: contract}, SafeProxyFactoryTransactor: SafeProxyFactoryTransactor{contract: contract}, SafeProxyFactoryFilterer: SafeProxyFactoryFilterer{contract: contract}}, nil
}

// SafeProxyFactory is an auto generated Go binding around an Ethereum contract.
type SafeProxyFactory struct {
	SafeProxyFactoryCaller     // Read-only binding to the contract
	SafeProxyFactoryTransactor // Write-only binding to the contract
	SafeProxyFactoryFilterer   // Log filterer for contract events
}

// SafeProxyFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type SafeProxyFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeProxyFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SafeProxyFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeProxyFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SafeProxyFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeProxyFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SafeProxyFactorySession struct {
	Contract     *SafeProxyFactory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SafeProxyFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SafeProxyFactoryCallerSession struct {
	Contract *SafeProxyFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// SafeProxyFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SafeProxyFactoryTransactorSession struct {
	Contract     *SafeProxyFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// SafeProxyFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type SafeProxyFactoryRaw struct {
	Contract *SafeProxyFactory // Generic contract binding to access the raw methods on
}

// SafeProxyFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SafeProxyFactoryCallerRaw struct {
	Contract *SafeProxyFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// SafeProxyFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SafeProxyFactoryTransactorRaw struct {
	Contract *SafeProxyFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSafeProxyFactory creates a new instance of SafeProxyFactory, bound to a specific deployed contract.
func NewSafeProxyFactory(address common.Address, backend bind.ContractBackend) (*SafeProxyFactory, error) {
	contract, err := bindSafeProxyFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SafeProxyFactory{SafeProxyFactoryCaller: SafeProxyFactoryCaller{contract: contract}, SafeProxyFactoryTransactor: SafeProxyFactoryTransactor{contract: contract}, SafeProxyFactoryFilterer: SafeProxyFactoryFilterer{contract: contract}}, nil
}

// NewSafeProxyFactoryCaller creates a new read-only instance of SafeProxyFactory, bound to a specific deployed contract.
func NewSafeProxyFactoryCaller(address common.Address, caller bind.ContractCaller) (*SafeProxyFactoryCaller, error) {
	contract, err := bindSafeProxyFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SafeProxyFactoryCaller{contract: contract}, nil
}

// NewSafeProxyFactoryTransactor creates a new write-only instance of SafeProxyFactory, bound to a specific deployed contract.
func NewSafeProxyFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*SafeProxyFactoryTransactor, error) {
	contract, err := bindSafeProxyFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SafeProxyFactoryTransactor{contract: contract}, nil
}

// NewSafeProxyFactoryFilterer creates a new log filterer instance of SafeProxyFactory, bound to a specific deployed contract.
func NewSafeProxyFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*SafeProxyFactoryFilterer, error) {
	contract, err := bindSafeProxyFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SafeProxyFactoryFilterer{contract: contract}, nil
}

// bindSafeProxyFactory binds a generic wrapper to an already deployed contract.
func bindSafeProxyFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SafeProxyFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeProxyFactory *SafeProxyFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SafeProxyFactory.Contract.SafeProxyFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeProxyFactory *SafeProxyFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeProxyFactory.Contract.SafeProxyFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeProxyFactory *SafeProxyFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeProxyFactory.Contract.SafeProxyFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeProxyFactory *SafeProxyFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SafeProxyFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeProxyFactory *SafeProxyFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeProxyFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeProxyFactory *SafeProxyFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeProxyFactory.Contract.contract.Transact(opts, method, params...)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256)
func (_SafeProxyFactory *SafeProxyFactoryCaller) GetChainId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SafeProxyFactory.contract.Call(opts, &out, "getChainId")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256)
func (_SafeProxyFactory *SafeProxyFactorySession) GetChainId() (*big.Int, error) {
	return _SafeProxyFactory.Contract.GetChainId(&_SafeProxyFactory.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256)
func (_SafeProxyFactory *SafeProxyFactoryCallerSession) GetChainId() (*big.Int, error) {
	return _SafeProxyFactory.Contract.GetChainId(&_SafeProxyFactory.CallOpts)
}

// ProxyCreationCode is a free data retrieval call binding the contract method 0x53e5d935.
//
// Solidity: function proxyCreationCode() pure returns(bytes)
func (_SafeProxyFactory *SafeProxyFactoryCaller) ProxyCreationCode(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _SafeProxyFactory.contract.Call(opts, &out, "proxyCreationCode")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// ProxyCreationCode is a free data retrieval call binding the contract method 0x53e5d935.
//
// Solidity: function proxyCreationCode() pure returns(bytes)
func (_SafeProxyFactory *SafeProxyFactorySession) ProxyCreationCode() ([]byte, error) {
	return _SafeProxyFactory.Contract.ProxyCreationCode(&_SafeProxyFactory.CallOpts)
}

// ProxyCreationCode is a free data retrieval call binding the contract method 0x53e5d935.
//
// Solidity: function proxyCreationCode() pure returns(bytes)
func (_SafeProxyFactory *SafeProxyFactoryCallerSession) ProxyCreationCode() ([]byte, error) {
	return _SafeProxyFactory.Contract.ProxyCreationCode(&_SafeProxyFactory.CallOpts)
}

// CreateChainSpecificProxyWithNonce is a paid mutator transaction binding the contract method 0xec9e80bb.
//
// Solidity: function createChainSpecificProxyWithNonce(address _singleton, bytes initializer, uint256 saltNonce) returns(address proxy)
func (_SafeProxyFactory *SafeProxyFactoryTransactor) CreateChainSpecificProxyWithNonce(opts *bind.TransactOpts, _singleton common.Address, initializer []byte, saltNonce *big.Int) (*types.Transaction, error) {
	return _SafeProxyFactory.contract.Transact(opts, "createChainSpecificProxyWithNonce", _singleton, initializer, saltNonce)
}

// CreateChainSpecificProxyWithNonce is a paid mutator transaction binding the contract method 0xec9e80bb.
//
// Solidity: function createChainSpecificProxyWithNonce(address _singleton, bytes initializer, uint256 saltNonce) returns(address proxy)
func (_SafeProxyFactory *SafeProxyFactorySession) CreateChainSpecificProxyWithNonce(_singleton common.Address, initializer []byte, saltNonce *big.Int) (*types.Transaction, error) {
	return _SafeProxyFactory.Contract.CreateChainSpecificProxyWithNonce(&_SafeProxyFactory.TransactOpts, _singleton, initializer, saltNonce)
}

// CreateChainSpecificProxyWithNonce is a paid mutator transaction binding the contract method 0xec9e80bb.
//
// Solidity: function createChainSpecificProxyWithNonce(address _singleton, bytes initializer, uint256 saltNonce) returns(address proxy)
func (_SafeProxyFactory *SafeProxyFactoryTransactorSession) CreateChainSpecificProxyWithNonce(_singleton common.Address, initializer []byte, saltNonce *big.Int) (*types.Transaction, error) {
	return _SafeProxyFactory.Contract.CreateChainSpecificProxyWithNonce(&_SafeProxyFactory.TransactOpts, _singleton, initializer, saltNonce)
}

// CreateProxyWithCallback is a paid mutator transaction binding the contract method 0xd18af54d.
//
// Solidity: function createProxyWithCallback(address _singleton, bytes initializer, uint256 saltNonce, address callback) returns(address proxy)
func (_SafeProxyFactory *SafeProxyFactoryTransactor) CreateProxyWithCallback(opts *bind.TransactOpts, _singleton common.Address, initializer []byte, saltNonce *big.Int, callback common.Address) (*types.Transaction, error) {
	return _SafeProxyFactory.contract.Transact(opts, "createProxyWithCallback", _singleton, initializer, saltNonce, callback)
}

// CreateProxyWithCallback is a paid mutator transaction binding the contract method 0xd18af54d.
//
// Solidity: function createProxyWithCallback(address _singleton, bytes initializer, uint256 saltNonce, address callback) returns(address proxy)
func (_SafeProxyFactory *SafeProxyFactorySession) CreateProxyWithCallback(_singleton common.Address, initializer []byte, saltNonce *big.Int, callback common.Address) (*types.Transaction, error) {
	return _SafeProxyFactory.Contract.CreateProxyWithCallback(&_SafeProxyFactory.TransactOpts, _singleton, initializer, saltNonce, callback)
}

// CreateProxyWithCallback is a paid mutator transaction binding the contract method 0xd18af54d.
//
// Solidity: function createProxyWithCallback(address _singleton, bytes initializer, uint256 saltNonce, address callback) returns(address proxy)
func (_SafeProxyFactory *SafeProxyFactoryTransactorSession) CreateProxyWithCallback(_singleton common.Address, initializer []byte, saltNonce *big.Int, callback common.Address) (*types.Transaction, error) {
	return _SafeProxyFactory.Contract.CreateProxyWithCallback(&_SafeProxyFactory.TransactOpts, _singleton, initializer, saltNonce, callback)
}

// CreateProxyWithNonce is a paid mutator transaction binding the contract method 0x1688f0b9.
//
// Solidity: function createProxyWithNonce(address _singleton, bytes initializer, uint256 saltNonce) returns(address proxy)
func (_SafeProxyFactory *SafeProxyFactoryTransactor) CreateProxyWithNonce(opts *bind.TransactOpts, _singleton common.Address, initializer []byte, saltNonce *big.Int) (*types.Transaction, error) {
	return _SafeProxyFactory.contract.Transact(opts, "createProxyWithNonce", _singleton, initializer, saltNonce)
}

// CreateProxyWithNonce is a paid mutator transaction binding the contract method 0x1688f0b9.
//
// Solidity: function createProxyWithNonce(address _singleton, bytes initializer, uint256 saltNonce) returns(address proxy)
func (_SafeProxyFactory *SafeProxyFactorySession) CreateProxyWithNonce(_singleton common.Address, initializer []byte, saltNonce *big.Int) (*types.Transaction, error) {
	return _SafeProxyFactory.Contract.CreateProxyWithNonce(&_SafeProxyFactory.TransactOpts, _singleton, initializer, saltNonce)
}

// CreateProxyWithNonce is a paid mutator transaction binding the contract method 0x1688f0b9.
//
// Solidity: function createProxyWithNonce(address _singleton, bytes initializer, uint256 saltNonce) returns(address proxy)
func (_SafeProxyFactory *SafeProxyFactoryTransactorSession) CreateProxyWithNonce(_singleton common.Address, initializer []byte, saltNonce *big.Int) (*types.Transaction, error) {
	return _SafeProxyFactory.Contract.CreateProxyWithNonce(&_SafeProxyFactory.TransactOpts, _singleton, initializer, saltNonce)
}

// SafeProxyFactoryProxyCreationIterator is returned from FilterProxyCreation and is used to iterate over the raw logs and unpacked data for ProxyCreation events raised by the SafeProxyFactory contract.
type SafeProxyFactoryProxyCreationIterator struct {
	Event *SafeProxyFactoryProxyCreation // Event containing the contract specifics and raw log

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
func (it *SafeProxyFactoryProxyCreationIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SafeProxyFactoryProxyCreation)
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
		it.Event = new(SafeProxyFactoryProxyCreation)
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
func (it *SafeProxyFactoryProxyCreationIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SafeProxyFactoryProxyCreationIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SafeProxyFactoryProxyCreation represents a ProxyCreation event raised by the SafeProxyFactory contract.
type SafeProxyFactoryProxyCreation struct {
	Proxy     common.Address
	Singleton common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterProxyCreation is a free log retrieval operation binding the contract event 0x4f51faf6c4561ff95f067657e43439f0f856d97c04d9ec9070a6199ad418e235.
//
// Solidity: event ProxyCreation(address indexed proxy, address singleton)
func (_SafeProxyFactory *SafeProxyFactoryFilterer) FilterProxyCreation(opts *bind.FilterOpts, proxy []common.Address) (*SafeProxyFactoryProxyCreationIterator, error) {

	var proxyRule []interface{}
	for _, proxyItem := range proxy {
		proxyRule = append(proxyRule, proxyItem)
	}

	logs, sub, err := _SafeProxyFactory.contract.FilterLogs(opts, "ProxyCreation", proxyRule)
	if err != nil {
		return nil, err
	}
	return &SafeProxyFactoryProxyCreationIterator{contract: _SafeProxyFactory.contract, event: "ProxyCreation", logs: logs, sub: sub}, nil
}

// WatchProxyCreation is a free log subscription operation binding the contract event 0x4f51faf6c4561ff95f067657e43439f0f856d97c04d9ec9070a6199ad418e235.
//
// Solidity: event ProxyCreation(address indexed proxy, address singleton)
func (_SafeProxyFactory *SafeProxyFactoryFilterer) WatchProxyCreation(opts *bind.WatchOpts, sink chan<- *SafeProxyFactoryProxyCreation, proxy []common.Address) (event.Subscription, error) {

	var proxyRule []interface{}
	for _, proxyItem := range proxy {
		proxyRule = append(proxyRule, proxyItem)
	}

	logs, sub, err := _SafeProxyFactory.contract.WatchLogs(opts, "ProxyCreation", proxyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SafeProxyFactoryProxyCreation)
				if err := _SafeProxyFactory.contract.UnpackLog(event, "ProxyCreation", log); err != nil {
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

// ParseProxyCreation is a log parse operation binding the contract event 0x4f51faf6c4561ff95f067657e43439f0f856d97c04d9ec9070a6199ad418e235.
//
// Solidity: event ProxyCreation(address indexed proxy, address singleton)
func (_SafeProxyFactory *SafeProxyFactoryFilterer) ParseProxyCreation(log types.Log) (*SafeProxyFactoryProxyCreation, error) {
	event := new(SafeProxyFactoryProxyCreation)
	if err := _SafeProxyFactory.contract.UnpackLog(event, "ProxyCreation", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
