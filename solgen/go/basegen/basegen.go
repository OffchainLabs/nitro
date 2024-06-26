// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package basegen

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

// BaseModuleGuardMetaData contains all meta data concerning the BaseModuleGuard contract.
var BaseModuleGuardMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"txHash\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"name\":\"checkAfterModuleExecution\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"checkModuleTransaction\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"moduleTxHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// BaseModuleGuardABI is the input ABI used to generate the binding from.
// Deprecated: Use BaseModuleGuardMetaData.ABI instead.
var BaseModuleGuardABI = BaseModuleGuardMetaData.ABI

// BaseModuleGuard is an auto generated Go binding around an Ethereum contract.
type BaseModuleGuard struct {
	BaseModuleGuardCaller     // Read-only binding to the contract
	BaseModuleGuardTransactor // Write-only binding to the contract
	BaseModuleGuardFilterer   // Log filterer for contract events
}

// BaseModuleGuardCaller is an auto generated read-only Go binding around an Ethereum contract.
type BaseModuleGuardCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseModuleGuardTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BaseModuleGuardTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseModuleGuardFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BaseModuleGuardFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseModuleGuardSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BaseModuleGuardSession struct {
	Contract     *BaseModuleGuard  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BaseModuleGuardCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BaseModuleGuardCallerSession struct {
	Contract *BaseModuleGuardCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// BaseModuleGuardTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BaseModuleGuardTransactorSession struct {
	Contract     *BaseModuleGuardTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// BaseModuleGuardRaw is an auto generated low-level Go binding around an Ethereum contract.
type BaseModuleGuardRaw struct {
	Contract *BaseModuleGuard // Generic contract binding to access the raw methods on
}

// BaseModuleGuardCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BaseModuleGuardCallerRaw struct {
	Contract *BaseModuleGuardCaller // Generic read-only contract binding to access the raw methods on
}

// BaseModuleGuardTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BaseModuleGuardTransactorRaw struct {
	Contract *BaseModuleGuardTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBaseModuleGuard creates a new instance of BaseModuleGuard, bound to a specific deployed contract.
func NewBaseModuleGuard(address common.Address, backend bind.ContractBackend) (*BaseModuleGuard, error) {
	contract, err := bindBaseModuleGuard(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BaseModuleGuard{BaseModuleGuardCaller: BaseModuleGuardCaller{contract: contract}, BaseModuleGuardTransactor: BaseModuleGuardTransactor{contract: contract}, BaseModuleGuardFilterer: BaseModuleGuardFilterer{contract: contract}}, nil
}

// NewBaseModuleGuardCaller creates a new read-only instance of BaseModuleGuard, bound to a specific deployed contract.
func NewBaseModuleGuardCaller(address common.Address, caller bind.ContractCaller) (*BaseModuleGuardCaller, error) {
	contract, err := bindBaseModuleGuard(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BaseModuleGuardCaller{contract: contract}, nil
}

// NewBaseModuleGuardTransactor creates a new write-only instance of BaseModuleGuard, bound to a specific deployed contract.
func NewBaseModuleGuardTransactor(address common.Address, transactor bind.ContractTransactor) (*BaseModuleGuardTransactor, error) {
	contract, err := bindBaseModuleGuard(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BaseModuleGuardTransactor{contract: contract}, nil
}

// NewBaseModuleGuardFilterer creates a new log filterer instance of BaseModuleGuard, bound to a specific deployed contract.
func NewBaseModuleGuardFilterer(address common.Address, filterer bind.ContractFilterer) (*BaseModuleGuardFilterer, error) {
	contract, err := bindBaseModuleGuard(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BaseModuleGuardFilterer{contract: contract}, nil
}

// bindBaseModuleGuard binds a generic wrapper to an already deployed contract.
func bindBaseModuleGuard(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BaseModuleGuardMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BaseModuleGuard *BaseModuleGuardRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BaseModuleGuard.Contract.BaseModuleGuardCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BaseModuleGuard *BaseModuleGuardRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseModuleGuard.Contract.BaseModuleGuardTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BaseModuleGuard *BaseModuleGuardRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BaseModuleGuard.Contract.BaseModuleGuardTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BaseModuleGuard *BaseModuleGuardCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BaseModuleGuard.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BaseModuleGuard *BaseModuleGuardTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseModuleGuard.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BaseModuleGuard *BaseModuleGuardTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BaseModuleGuard.Contract.contract.Transact(opts, method, params...)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_BaseModuleGuard *BaseModuleGuardCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _BaseModuleGuard.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_BaseModuleGuard *BaseModuleGuardSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _BaseModuleGuard.Contract.SupportsInterface(&_BaseModuleGuard.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_BaseModuleGuard *BaseModuleGuardCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _BaseModuleGuard.Contract.SupportsInterface(&_BaseModuleGuard.CallOpts, interfaceId)
}

// CheckAfterModuleExecution is a paid mutator transaction binding the contract method 0x2acc37aa.
//
// Solidity: function checkAfterModuleExecution(bytes32 txHash, bool success) returns()
func (_BaseModuleGuard *BaseModuleGuardTransactor) CheckAfterModuleExecution(opts *bind.TransactOpts, txHash [32]byte, success bool) (*types.Transaction, error) {
	return _BaseModuleGuard.contract.Transact(opts, "checkAfterModuleExecution", txHash, success)
}

// CheckAfterModuleExecution is a paid mutator transaction binding the contract method 0x2acc37aa.
//
// Solidity: function checkAfterModuleExecution(bytes32 txHash, bool success) returns()
func (_BaseModuleGuard *BaseModuleGuardSession) CheckAfterModuleExecution(txHash [32]byte, success bool) (*types.Transaction, error) {
	return _BaseModuleGuard.Contract.CheckAfterModuleExecution(&_BaseModuleGuard.TransactOpts, txHash, success)
}

// CheckAfterModuleExecution is a paid mutator transaction binding the contract method 0x2acc37aa.
//
// Solidity: function checkAfterModuleExecution(bytes32 txHash, bool success) returns()
func (_BaseModuleGuard *BaseModuleGuardTransactorSession) CheckAfterModuleExecution(txHash [32]byte, success bool) (*types.Transaction, error) {
	return _BaseModuleGuard.Contract.CheckAfterModuleExecution(&_BaseModuleGuard.TransactOpts, txHash, success)
}

// CheckModuleTransaction is a paid mutator transaction binding the contract method 0x728c2972.
//
// Solidity: function checkModuleTransaction(address to, uint256 value, bytes data, uint8 operation, address module) returns(bytes32 moduleTxHash)
func (_BaseModuleGuard *BaseModuleGuardTransactor) CheckModuleTransaction(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8, module common.Address) (*types.Transaction, error) {
	return _BaseModuleGuard.contract.Transact(opts, "checkModuleTransaction", to, value, data, operation, module)
}

// CheckModuleTransaction is a paid mutator transaction binding the contract method 0x728c2972.
//
// Solidity: function checkModuleTransaction(address to, uint256 value, bytes data, uint8 operation, address module) returns(bytes32 moduleTxHash)
func (_BaseModuleGuard *BaseModuleGuardSession) CheckModuleTransaction(to common.Address, value *big.Int, data []byte, operation uint8, module common.Address) (*types.Transaction, error) {
	return _BaseModuleGuard.Contract.CheckModuleTransaction(&_BaseModuleGuard.TransactOpts, to, value, data, operation, module)
}

// CheckModuleTransaction is a paid mutator transaction binding the contract method 0x728c2972.
//
// Solidity: function checkModuleTransaction(address to, uint256 value, bytes data, uint8 operation, address module) returns(bytes32 moduleTxHash)
func (_BaseModuleGuard *BaseModuleGuardTransactorSession) CheckModuleTransaction(to common.Address, value *big.Int, data []byte, operation uint8, module common.Address) (*types.Transaction, error) {
	return _BaseModuleGuard.Contract.CheckModuleTransaction(&_BaseModuleGuard.TransactOpts, to, value, data, operation, module)
}

// BaseTransactionGuardMetaData contains all meta data concerning the BaseTransactionGuard contract.
var BaseTransactionGuardMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"name\":\"checkAfterExecution\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"safeTxGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasPrice\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"gasToken\",\"type\":\"address\"},{\"internalType\":\"addresspayable\",\"name\":\"refundReceiver\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"},{\"internalType\":\"address\",\"name\":\"msgSender\",\"type\":\"address\"}],\"name\":\"checkTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// BaseTransactionGuardABI is the input ABI used to generate the binding from.
// Deprecated: Use BaseTransactionGuardMetaData.ABI instead.
var BaseTransactionGuardABI = BaseTransactionGuardMetaData.ABI

// BaseTransactionGuard is an auto generated Go binding around an Ethereum contract.
type BaseTransactionGuard struct {
	BaseTransactionGuardCaller     // Read-only binding to the contract
	BaseTransactionGuardTransactor // Write-only binding to the contract
	BaseTransactionGuardFilterer   // Log filterer for contract events
}

// BaseTransactionGuardCaller is an auto generated read-only Go binding around an Ethereum contract.
type BaseTransactionGuardCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseTransactionGuardTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BaseTransactionGuardTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseTransactionGuardFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BaseTransactionGuardFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseTransactionGuardSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BaseTransactionGuardSession struct {
	Contract     *BaseTransactionGuard // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// BaseTransactionGuardCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BaseTransactionGuardCallerSession struct {
	Contract *BaseTransactionGuardCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// BaseTransactionGuardTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BaseTransactionGuardTransactorSession struct {
	Contract     *BaseTransactionGuardTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// BaseTransactionGuardRaw is an auto generated low-level Go binding around an Ethereum contract.
type BaseTransactionGuardRaw struct {
	Contract *BaseTransactionGuard // Generic contract binding to access the raw methods on
}

// BaseTransactionGuardCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BaseTransactionGuardCallerRaw struct {
	Contract *BaseTransactionGuardCaller // Generic read-only contract binding to access the raw methods on
}

// BaseTransactionGuardTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BaseTransactionGuardTransactorRaw struct {
	Contract *BaseTransactionGuardTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBaseTransactionGuard creates a new instance of BaseTransactionGuard, bound to a specific deployed contract.
func NewBaseTransactionGuard(address common.Address, backend bind.ContractBackend) (*BaseTransactionGuard, error) {
	contract, err := bindBaseTransactionGuard(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BaseTransactionGuard{BaseTransactionGuardCaller: BaseTransactionGuardCaller{contract: contract}, BaseTransactionGuardTransactor: BaseTransactionGuardTransactor{contract: contract}, BaseTransactionGuardFilterer: BaseTransactionGuardFilterer{contract: contract}}, nil
}

// NewBaseTransactionGuardCaller creates a new read-only instance of BaseTransactionGuard, bound to a specific deployed contract.
func NewBaseTransactionGuardCaller(address common.Address, caller bind.ContractCaller) (*BaseTransactionGuardCaller, error) {
	contract, err := bindBaseTransactionGuard(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BaseTransactionGuardCaller{contract: contract}, nil
}

// NewBaseTransactionGuardTransactor creates a new write-only instance of BaseTransactionGuard, bound to a specific deployed contract.
func NewBaseTransactionGuardTransactor(address common.Address, transactor bind.ContractTransactor) (*BaseTransactionGuardTransactor, error) {
	contract, err := bindBaseTransactionGuard(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BaseTransactionGuardTransactor{contract: contract}, nil
}

// NewBaseTransactionGuardFilterer creates a new log filterer instance of BaseTransactionGuard, bound to a specific deployed contract.
func NewBaseTransactionGuardFilterer(address common.Address, filterer bind.ContractFilterer) (*BaseTransactionGuardFilterer, error) {
	contract, err := bindBaseTransactionGuard(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BaseTransactionGuardFilterer{contract: contract}, nil
}

// bindBaseTransactionGuard binds a generic wrapper to an already deployed contract.
func bindBaseTransactionGuard(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BaseTransactionGuardMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BaseTransactionGuard *BaseTransactionGuardRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BaseTransactionGuard.Contract.BaseTransactionGuardCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BaseTransactionGuard *BaseTransactionGuardRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseTransactionGuard.Contract.BaseTransactionGuardTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BaseTransactionGuard *BaseTransactionGuardRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BaseTransactionGuard.Contract.BaseTransactionGuardTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BaseTransactionGuard *BaseTransactionGuardCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BaseTransactionGuard.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BaseTransactionGuard *BaseTransactionGuardTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseTransactionGuard.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BaseTransactionGuard *BaseTransactionGuardTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BaseTransactionGuard.Contract.contract.Transact(opts, method, params...)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_BaseTransactionGuard *BaseTransactionGuardCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _BaseTransactionGuard.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_BaseTransactionGuard *BaseTransactionGuardSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _BaseTransactionGuard.Contract.SupportsInterface(&_BaseTransactionGuard.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_BaseTransactionGuard *BaseTransactionGuardCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _BaseTransactionGuard.Contract.SupportsInterface(&_BaseTransactionGuard.CallOpts, interfaceId)
}

// CheckAfterExecution is a paid mutator transaction binding the contract method 0x93271368.
//
// Solidity: function checkAfterExecution(bytes32 hash, bool success) returns()
func (_BaseTransactionGuard *BaseTransactionGuardTransactor) CheckAfterExecution(opts *bind.TransactOpts, hash [32]byte, success bool) (*types.Transaction, error) {
	return _BaseTransactionGuard.contract.Transact(opts, "checkAfterExecution", hash, success)
}

// CheckAfterExecution is a paid mutator transaction binding the contract method 0x93271368.
//
// Solidity: function checkAfterExecution(bytes32 hash, bool success) returns()
func (_BaseTransactionGuard *BaseTransactionGuardSession) CheckAfterExecution(hash [32]byte, success bool) (*types.Transaction, error) {
	return _BaseTransactionGuard.Contract.CheckAfterExecution(&_BaseTransactionGuard.TransactOpts, hash, success)
}

// CheckAfterExecution is a paid mutator transaction binding the contract method 0x93271368.
//
// Solidity: function checkAfterExecution(bytes32 hash, bool success) returns()
func (_BaseTransactionGuard *BaseTransactionGuardTransactorSession) CheckAfterExecution(hash [32]byte, success bool) (*types.Transaction, error) {
	return _BaseTransactionGuard.Contract.CheckAfterExecution(&_BaseTransactionGuard.TransactOpts, hash, success)
}

// CheckTransaction is a paid mutator transaction binding the contract method 0x75f0bb52.
//
// Solidity: function checkTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures, address msgSender) returns()
func (_BaseTransactionGuard *BaseTransactionGuardTransactor) CheckTransaction(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte, msgSender common.Address) (*types.Transaction, error) {
	return _BaseTransactionGuard.contract.Transact(opts, "checkTransaction", to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures, msgSender)
}

// CheckTransaction is a paid mutator transaction binding the contract method 0x75f0bb52.
//
// Solidity: function checkTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures, address msgSender) returns()
func (_BaseTransactionGuard *BaseTransactionGuardSession) CheckTransaction(to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte, msgSender common.Address) (*types.Transaction, error) {
	return _BaseTransactionGuard.Contract.CheckTransaction(&_BaseTransactionGuard.TransactOpts, to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures, msgSender)
}

// CheckTransaction is a paid mutator transaction binding the contract method 0x75f0bb52.
//
// Solidity: function checkTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures, address msgSender) returns()
func (_BaseTransactionGuard *BaseTransactionGuardTransactorSession) CheckTransaction(to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte, msgSender common.Address) (*types.Transaction, error) {
	return _BaseTransactionGuard.Contract.CheckTransaction(&_BaseTransactionGuard.TransactOpts, to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures, msgSender)
}

// ExecutorMetaData contains all meta data concerning the Executor contract.
var ExecutorMetaData = &bind.MetaData{
	ABI: "[]",
}

// ExecutorABI is the input ABI used to generate the binding from.
// Deprecated: Use ExecutorMetaData.ABI instead.
var ExecutorABI = ExecutorMetaData.ABI

// Executor is an auto generated Go binding around an Ethereum contract.
type Executor struct {
	ExecutorCaller     // Read-only binding to the contract
	ExecutorTransactor // Write-only binding to the contract
	ExecutorFilterer   // Log filterer for contract events
}

// ExecutorCaller is an auto generated read-only Go binding around an Ethereum contract.
type ExecutorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExecutorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ExecutorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExecutorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ExecutorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExecutorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ExecutorSession struct {
	Contract     *Executor         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ExecutorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ExecutorCallerSession struct {
	Contract *ExecutorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ExecutorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ExecutorTransactorSession struct {
	Contract     *ExecutorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ExecutorRaw is an auto generated low-level Go binding around an Ethereum contract.
type ExecutorRaw struct {
	Contract *Executor // Generic contract binding to access the raw methods on
}

// ExecutorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ExecutorCallerRaw struct {
	Contract *ExecutorCaller // Generic read-only contract binding to access the raw methods on
}

// ExecutorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ExecutorTransactorRaw struct {
	Contract *ExecutorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewExecutor creates a new instance of Executor, bound to a specific deployed contract.
func NewExecutor(address common.Address, backend bind.ContractBackend) (*Executor, error) {
	contract, err := bindExecutor(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Executor{ExecutorCaller: ExecutorCaller{contract: contract}, ExecutorTransactor: ExecutorTransactor{contract: contract}, ExecutorFilterer: ExecutorFilterer{contract: contract}}, nil
}

// NewExecutorCaller creates a new read-only instance of Executor, bound to a specific deployed contract.
func NewExecutorCaller(address common.Address, caller bind.ContractCaller) (*ExecutorCaller, error) {
	contract, err := bindExecutor(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ExecutorCaller{contract: contract}, nil
}

// NewExecutorTransactor creates a new write-only instance of Executor, bound to a specific deployed contract.
func NewExecutorTransactor(address common.Address, transactor bind.ContractTransactor) (*ExecutorTransactor, error) {
	contract, err := bindExecutor(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ExecutorTransactor{contract: contract}, nil
}

// NewExecutorFilterer creates a new log filterer instance of Executor, bound to a specific deployed contract.
func NewExecutorFilterer(address common.Address, filterer bind.ContractFilterer) (*ExecutorFilterer, error) {
	contract, err := bindExecutor(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ExecutorFilterer{contract: contract}, nil
}

// bindExecutor binds a generic wrapper to an already deployed contract.
func bindExecutor(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ExecutorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Executor *ExecutorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Executor.Contract.ExecutorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Executor *ExecutorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Executor.Contract.ExecutorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Executor *ExecutorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Executor.Contract.ExecutorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Executor *ExecutorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Executor.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Executor *ExecutorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Executor.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Executor *ExecutorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Executor.Contract.contract.Transact(opts, method, params...)
}

// FallbackManagerMetaData contains all meta data concerning the FallbackManager contract.
var FallbackManagerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"handler\",\"type\":\"address\"}],\"name\":\"ChangedFallbackHandler\",\"type\":\"event\"},{\"stateMutability\":\"nonpayable\",\"type\":\"fallback\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"handler\",\"type\":\"address\"}],\"name\":\"setFallbackHandler\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// FallbackManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use FallbackManagerMetaData.ABI instead.
var FallbackManagerABI = FallbackManagerMetaData.ABI

// FallbackManager is an auto generated Go binding around an Ethereum contract.
type FallbackManager struct {
	FallbackManagerCaller     // Read-only binding to the contract
	FallbackManagerTransactor // Write-only binding to the contract
	FallbackManagerFilterer   // Log filterer for contract events
}

// FallbackManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type FallbackManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FallbackManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FallbackManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FallbackManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FallbackManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FallbackManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FallbackManagerSession struct {
	Contract     *FallbackManager  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// FallbackManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FallbackManagerCallerSession struct {
	Contract *FallbackManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// FallbackManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FallbackManagerTransactorSession struct {
	Contract     *FallbackManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// FallbackManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type FallbackManagerRaw struct {
	Contract *FallbackManager // Generic contract binding to access the raw methods on
}

// FallbackManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FallbackManagerCallerRaw struct {
	Contract *FallbackManagerCaller // Generic read-only contract binding to access the raw methods on
}

// FallbackManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FallbackManagerTransactorRaw struct {
	Contract *FallbackManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFallbackManager creates a new instance of FallbackManager, bound to a specific deployed contract.
func NewFallbackManager(address common.Address, backend bind.ContractBackend) (*FallbackManager, error) {
	contract, err := bindFallbackManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &FallbackManager{FallbackManagerCaller: FallbackManagerCaller{contract: contract}, FallbackManagerTransactor: FallbackManagerTransactor{contract: contract}, FallbackManagerFilterer: FallbackManagerFilterer{contract: contract}}, nil
}

// NewFallbackManagerCaller creates a new read-only instance of FallbackManager, bound to a specific deployed contract.
func NewFallbackManagerCaller(address common.Address, caller bind.ContractCaller) (*FallbackManagerCaller, error) {
	contract, err := bindFallbackManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FallbackManagerCaller{contract: contract}, nil
}

// NewFallbackManagerTransactor creates a new write-only instance of FallbackManager, bound to a specific deployed contract.
func NewFallbackManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*FallbackManagerTransactor, error) {
	contract, err := bindFallbackManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FallbackManagerTransactor{contract: contract}, nil
}

// NewFallbackManagerFilterer creates a new log filterer instance of FallbackManager, bound to a specific deployed contract.
func NewFallbackManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*FallbackManagerFilterer, error) {
	contract, err := bindFallbackManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FallbackManagerFilterer{contract: contract}, nil
}

// bindFallbackManager binds a generic wrapper to an already deployed contract.
func bindFallbackManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := FallbackManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FallbackManager *FallbackManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FallbackManager.Contract.FallbackManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FallbackManager *FallbackManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FallbackManager.Contract.FallbackManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FallbackManager *FallbackManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FallbackManager.Contract.FallbackManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FallbackManager *FallbackManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FallbackManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FallbackManager *FallbackManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FallbackManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FallbackManager *FallbackManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FallbackManager.Contract.contract.Transact(opts, method, params...)
}

// SetFallbackHandler is a paid mutator transaction binding the contract method 0xf08a0323.
//
// Solidity: function setFallbackHandler(address handler) returns()
func (_FallbackManager *FallbackManagerTransactor) SetFallbackHandler(opts *bind.TransactOpts, handler common.Address) (*types.Transaction, error) {
	return _FallbackManager.contract.Transact(opts, "setFallbackHandler", handler)
}

// SetFallbackHandler is a paid mutator transaction binding the contract method 0xf08a0323.
//
// Solidity: function setFallbackHandler(address handler) returns()
func (_FallbackManager *FallbackManagerSession) SetFallbackHandler(handler common.Address) (*types.Transaction, error) {
	return _FallbackManager.Contract.SetFallbackHandler(&_FallbackManager.TransactOpts, handler)
}

// SetFallbackHandler is a paid mutator transaction binding the contract method 0xf08a0323.
//
// Solidity: function setFallbackHandler(address handler) returns()
func (_FallbackManager *FallbackManagerTransactorSession) SetFallbackHandler(handler common.Address) (*types.Transaction, error) {
	return _FallbackManager.Contract.SetFallbackHandler(&_FallbackManager.TransactOpts, handler)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() returns()
func (_FallbackManager *FallbackManagerTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _FallbackManager.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() returns()
func (_FallbackManager *FallbackManagerSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _FallbackManager.Contract.Fallback(&_FallbackManager.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() returns()
func (_FallbackManager *FallbackManagerTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _FallbackManager.Contract.Fallback(&_FallbackManager.TransactOpts, calldata)
}

// FallbackManagerChangedFallbackHandlerIterator is returned from FilterChangedFallbackHandler and is used to iterate over the raw logs and unpacked data for ChangedFallbackHandler events raised by the FallbackManager contract.
type FallbackManagerChangedFallbackHandlerIterator struct {
	Event *FallbackManagerChangedFallbackHandler // Event containing the contract specifics and raw log

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
func (it *FallbackManagerChangedFallbackHandlerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FallbackManagerChangedFallbackHandler)
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
		it.Event = new(FallbackManagerChangedFallbackHandler)
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
func (it *FallbackManagerChangedFallbackHandlerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FallbackManagerChangedFallbackHandlerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FallbackManagerChangedFallbackHandler represents a ChangedFallbackHandler event raised by the FallbackManager contract.
type FallbackManagerChangedFallbackHandler struct {
	Handler common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterChangedFallbackHandler is a free log retrieval operation binding the contract event 0x5ac6c46c93c8d0e53714ba3b53db3e7c046da994313d7ed0d192028bc7c228b0.
//
// Solidity: event ChangedFallbackHandler(address indexed handler)
func (_FallbackManager *FallbackManagerFilterer) FilterChangedFallbackHandler(opts *bind.FilterOpts, handler []common.Address) (*FallbackManagerChangedFallbackHandlerIterator, error) {

	var handlerRule []interface{}
	for _, handlerItem := range handler {
		handlerRule = append(handlerRule, handlerItem)
	}

	logs, sub, err := _FallbackManager.contract.FilterLogs(opts, "ChangedFallbackHandler", handlerRule)
	if err != nil {
		return nil, err
	}
	return &FallbackManagerChangedFallbackHandlerIterator{contract: _FallbackManager.contract, event: "ChangedFallbackHandler", logs: logs, sub: sub}, nil
}

// WatchChangedFallbackHandler is a free log subscription operation binding the contract event 0x5ac6c46c93c8d0e53714ba3b53db3e7c046da994313d7ed0d192028bc7c228b0.
//
// Solidity: event ChangedFallbackHandler(address indexed handler)
func (_FallbackManager *FallbackManagerFilterer) WatchChangedFallbackHandler(opts *bind.WatchOpts, sink chan<- *FallbackManagerChangedFallbackHandler, handler []common.Address) (event.Subscription, error) {

	var handlerRule []interface{}
	for _, handlerItem := range handler {
		handlerRule = append(handlerRule, handlerItem)
	}

	logs, sub, err := _FallbackManager.contract.WatchLogs(opts, "ChangedFallbackHandler", handlerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FallbackManagerChangedFallbackHandler)
				if err := _FallbackManager.contract.UnpackLog(event, "ChangedFallbackHandler", log); err != nil {
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

// ParseChangedFallbackHandler is a log parse operation binding the contract event 0x5ac6c46c93c8d0e53714ba3b53db3e7c046da994313d7ed0d192028bc7c228b0.
//
// Solidity: event ChangedFallbackHandler(address indexed handler)
func (_FallbackManager *FallbackManagerFilterer) ParseChangedFallbackHandler(log types.Log) (*FallbackManagerChangedFallbackHandler, error) {
	event := new(FallbackManagerChangedFallbackHandler)
	if err := _FallbackManager.contract.UnpackLog(event, "ChangedFallbackHandler", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GuardManagerMetaData contains all meta data concerning the GuardManager contract.
var GuardManagerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"guard\",\"type\":\"address\"}],\"name\":\"ChangedGuard\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"guard\",\"type\":\"address\"}],\"name\":\"setGuard\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// GuardManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use GuardManagerMetaData.ABI instead.
var GuardManagerABI = GuardManagerMetaData.ABI

// GuardManager is an auto generated Go binding around an Ethereum contract.
type GuardManager struct {
	GuardManagerCaller     // Read-only binding to the contract
	GuardManagerTransactor // Write-only binding to the contract
	GuardManagerFilterer   // Log filterer for contract events
}

// GuardManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type GuardManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GuardManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GuardManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GuardManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GuardManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GuardManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GuardManagerSession struct {
	Contract     *GuardManager     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// GuardManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GuardManagerCallerSession struct {
	Contract *GuardManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// GuardManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GuardManagerTransactorSession struct {
	Contract     *GuardManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// GuardManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type GuardManagerRaw struct {
	Contract *GuardManager // Generic contract binding to access the raw methods on
}

// GuardManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GuardManagerCallerRaw struct {
	Contract *GuardManagerCaller // Generic read-only contract binding to access the raw methods on
}

// GuardManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GuardManagerTransactorRaw struct {
	Contract *GuardManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGuardManager creates a new instance of GuardManager, bound to a specific deployed contract.
func NewGuardManager(address common.Address, backend bind.ContractBackend) (*GuardManager, error) {
	contract, err := bindGuardManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &GuardManager{GuardManagerCaller: GuardManagerCaller{contract: contract}, GuardManagerTransactor: GuardManagerTransactor{contract: contract}, GuardManagerFilterer: GuardManagerFilterer{contract: contract}}, nil
}

// NewGuardManagerCaller creates a new read-only instance of GuardManager, bound to a specific deployed contract.
func NewGuardManagerCaller(address common.Address, caller bind.ContractCaller) (*GuardManagerCaller, error) {
	contract, err := bindGuardManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GuardManagerCaller{contract: contract}, nil
}

// NewGuardManagerTransactor creates a new write-only instance of GuardManager, bound to a specific deployed contract.
func NewGuardManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*GuardManagerTransactor, error) {
	contract, err := bindGuardManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GuardManagerTransactor{contract: contract}, nil
}

// NewGuardManagerFilterer creates a new log filterer instance of GuardManager, bound to a specific deployed contract.
func NewGuardManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*GuardManagerFilterer, error) {
	contract, err := bindGuardManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GuardManagerFilterer{contract: contract}, nil
}

// bindGuardManager binds a generic wrapper to an already deployed contract.
func bindGuardManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := GuardManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GuardManager *GuardManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GuardManager.Contract.GuardManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GuardManager *GuardManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GuardManager.Contract.GuardManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GuardManager *GuardManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GuardManager.Contract.GuardManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GuardManager *GuardManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GuardManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GuardManager *GuardManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GuardManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GuardManager *GuardManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GuardManager.Contract.contract.Transact(opts, method, params...)
}

// SetGuard is a paid mutator transaction binding the contract method 0xe19a9dd9.
//
// Solidity: function setGuard(address guard) returns()
func (_GuardManager *GuardManagerTransactor) SetGuard(opts *bind.TransactOpts, guard common.Address) (*types.Transaction, error) {
	return _GuardManager.contract.Transact(opts, "setGuard", guard)
}

// SetGuard is a paid mutator transaction binding the contract method 0xe19a9dd9.
//
// Solidity: function setGuard(address guard) returns()
func (_GuardManager *GuardManagerSession) SetGuard(guard common.Address) (*types.Transaction, error) {
	return _GuardManager.Contract.SetGuard(&_GuardManager.TransactOpts, guard)
}

// SetGuard is a paid mutator transaction binding the contract method 0xe19a9dd9.
//
// Solidity: function setGuard(address guard) returns()
func (_GuardManager *GuardManagerTransactorSession) SetGuard(guard common.Address) (*types.Transaction, error) {
	return _GuardManager.Contract.SetGuard(&_GuardManager.TransactOpts, guard)
}

// GuardManagerChangedGuardIterator is returned from FilterChangedGuard and is used to iterate over the raw logs and unpacked data for ChangedGuard events raised by the GuardManager contract.
type GuardManagerChangedGuardIterator struct {
	Event *GuardManagerChangedGuard // Event containing the contract specifics and raw log

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
func (it *GuardManagerChangedGuardIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GuardManagerChangedGuard)
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
		it.Event = new(GuardManagerChangedGuard)
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
func (it *GuardManagerChangedGuardIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GuardManagerChangedGuardIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GuardManagerChangedGuard represents a ChangedGuard event raised by the GuardManager contract.
type GuardManagerChangedGuard struct {
	Guard common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterChangedGuard is a free log retrieval operation binding the contract event 0x1151116914515bc0891ff9047a6cb32cf902546f83066499bcf8ba33d2353fa2.
//
// Solidity: event ChangedGuard(address indexed guard)
func (_GuardManager *GuardManagerFilterer) FilterChangedGuard(opts *bind.FilterOpts, guard []common.Address) (*GuardManagerChangedGuardIterator, error) {

	var guardRule []interface{}
	for _, guardItem := range guard {
		guardRule = append(guardRule, guardItem)
	}

	logs, sub, err := _GuardManager.contract.FilterLogs(opts, "ChangedGuard", guardRule)
	if err != nil {
		return nil, err
	}
	return &GuardManagerChangedGuardIterator{contract: _GuardManager.contract, event: "ChangedGuard", logs: logs, sub: sub}, nil
}

// WatchChangedGuard is a free log subscription operation binding the contract event 0x1151116914515bc0891ff9047a6cb32cf902546f83066499bcf8ba33d2353fa2.
//
// Solidity: event ChangedGuard(address indexed guard)
func (_GuardManager *GuardManagerFilterer) WatchChangedGuard(opts *bind.WatchOpts, sink chan<- *GuardManagerChangedGuard, guard []common.Address) (event.Subscription, error) {

	var guardRule []interface{}
	for _, guardItem := range guard {
		guardRule = append(guardRule, guardItem)
	}

	logs, sub, err := _GuardManager.contract.WatchLogs(opts, "ChangedGuard", guardRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GuardManagerChangedGuard)
				if err := _GuardManager.contract.UnpackLog(event, "ChangedGuard", log); err != nil {
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

// ParseChangedGuard is a log parse operation binding the contract event 0x1151116914515bc0891ff9047a6cb32cf902546f83066499bcf8ba33d2353fa2.
//
// Solidity: event ChangedGuard(address indexed guard)
func (_GuardManager *GuardManagerFilterer) ParseChangedGuard(log types.Log) (*GuardManagerChangedGuard, error) {
	event := new(GuardManagerChangedGuard)
	if err := _GuardManager.contract.UnpackLog(event, "ChangedGuard", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IModuleGuardMetaData contains all meta data concerning the IModuleGuard contract.
var IModuleGuardMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"txHash\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"name\":\"checkAfterModuleExecution\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"checkModuleTransaction\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"moduleTxHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IModuleGuardABI is the input ABI used to generate the binding from.
// Deprecated: Use IModuleGuardMetaData.ABI instead.
var IModuleGuardABI = IModuleGuardMetaData.ABI

// IModuleGuard is an auto generated Go binding around an Ethereum contract.
type IModuleGuard struct {
	IModuleGuardCaller     // Read-only binding to the contract
	IModuleGuardTransactor // Write-only binding to the contract
	IModuleGuardFilterer   // Log filterer for contract events
}

// IModuleGuardCaller is an auto generated read-only Go binding around an Ethereum contract.
type IModuleGuardCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IModuleGuardTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IModuleGuardTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IModuleGuardFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IModuleGuardFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IModuleGuardSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IModuleGuardSession struct {
	Contract     *IModuleGuard     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IModuleGuardCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IModuleGuardCallerSession struct {
	Contract *IModuleGuardCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// IModuleGuardTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IModuleGuardTransactorSession struct {
	Contract     *IModuleGuardTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// IModuleGuardRaw is an auto generated low-level Go binding around an Ethereum contract.
type IModuleGuardRaw struct {
	Contract *IModuleGuard // Generic contract binding to access the raw methods on
}

// IModuleGuardCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IModuleGuardCallerRaw struct {
	Contract *IModuleGuardCaller // Generic read-only contract binding to access the raw methods on
}

// IModuleGuardTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IModuleGuardTransactorRaw struct {
	Contract *IModuleGuardTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIModuleGuard creates a new instance of IModuleGuard, bound to a specific deployed contract.
func NewIModuleGuard(address common.Address, backend bind.ContractBackend) (*IModuleGuard, error) {
	contract, err := bindIModuleGuard(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IModuleGuard{IModuleGuardCaller: IModuleGuardCaller{contract: contract}, IModuleGuardTransactor: IModuleGuardTransactor{contract: contract}, IModuleGuardFilterer: IModuleGuardFilterer{contract: contract}}, nil
}

// NewIModuleGuardCaller creates a new read-only instance of IModuleGuard, bound to a specific deployed contract.
func NewIModuleGuardCaller(address common.Address, caller bind.ContractCaller) (*IModuleGuardCaller, error) {
	contract, err := bindIModuleGuard(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IModuleGuardCaller{contract: contract}, nil
}

// NewIModuleGuardTransactor creates a new write-only instance of IModuleGuard, bound to a specific deployed contract.
func NewIModuleGuardTransactor(address common.Address, transactor bind.ContractTransactor) (*IModuleGuardTransactor, error) {
	contract, err := bindIModuleGuard(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IModuleGuardTransactor{contract: contract}, nil
}

// NewIModuleGuardFilterer creates a new log filterer instance of IModuleGuard, bound to a specific deployed contract.
func NewIModuleGuardFilterer(address common.Address, filterer bind.ContractFilterer) (*IModuleGuardFilterer, error) {
	contract, err := bindIModuleGuard(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IModuleGuardFilterer{contract: contract}, nil
}

// bindIModuleGuard binds a generic wrapper to an already deployed contract.
func bindIModuleGuard(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IModuleGuardMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IModuleGuard *IModuleGuardRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IModuleGuard.Contract.IModuleGuardCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IModuleGuard *IModuleGuardRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IModuleGuard.Contract.IModuleGuardTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IModuleGuard *IModuleGuardRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IModuleGuard.Contract.IModuleGuardTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IModuleGuard *IModuleGuardCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IModuleGuard.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IModuleGuard *IModuleGuardTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IModuleGuard.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IModuleGuard *IModuleGuardTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IModuleGuard.Contract.contract.Transact(opts, method, params...)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_IModuleGuard *IModuleGuardCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _IModuleGuard.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_IModuleGuard *IModuleGuardSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _IModuleGuard.Contract.SupportsInterface(&_IModuleGuard.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_IModuleGuard *IModuleGuardCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _IModuleGuard.Contract.SupportsInterface(&_IModuleGuard.CallOpts, interfaceId)
}

// CheckAfterModuleExecution is a paid mutator transaction binding the contract method 0x2acc37aa.
//
// Solidity: function checkAfterModuleExecution(bytes32 txHash, bool success) returns()
func (_IModuleGuard *IModuleGuardTransactor) CheckAfterModuleExecution(opts *bind.TransactOpts, txHash [32]byte, success bool) (*types.Transaction, error) {
	return _IModuleGuard.contract.Transact(opts, "checkAfterModuleExecution", txHash, success)
}

// CheckAfterModuleExecution is a paid mutator transaction binding the contract method 0x2acc37aa.
//
// Solidity: function checkAfterModuleExecution(bytes32 txHash, bool success) returns()
func (_IModuleGuard *IModuleGuardSession) CheckAfterModuleExecution(txHash [32]byte, success bool) (*types.Transaction, error) {
	return _IModuleGuard.Contract.CheckAfterModuleExecution(&_IModuleGuard.TransactOpts, txHash, success)
}

// CheckAfterModuleExecution is a paid mutator transaction binding the contract method 0x2acc37aa.
//
// Solidity: function checkAfterModuleExecution(bytes32 txHash, bool success) returns()
func (_IModuleGuard *IModuleGuardTransactorSession) CheckAfterModuleExecution(txHash [32]byte, success bool) (*types.Transaction, error) {
	return _IModuleGuard.Contract.CheckAfterModuleExecution(&_IModuleGuard.TransactOpts, txHash, success)
}

// CheckModuleTransaction is a paid mutator transaction binding the contract method 0x728c2972.
//
// Solidity: function checkModuleTransaction(address to, uint256 value, bytes data, uint8 operation, address module) returns(bytes32 moduleTxHash)
func (_IModuleGuard *IModuleGuardTransactor) CheckModuleTransaction(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8, module common.Address) (*types.Transaction, error) {
	return _IModuleGuard.contract.Transact(opts, "checkModuleTransaction", to, value, data, operation, module)
}

// CheckModuleTransaction is a paid mutator transaction binding the contract method 0x728c2972.
//
// Solidity: function checkModuleTransaction(address to, uint256 value, bytes data, uint8 operation, address module) returns(bytes32 moduleTxHash)
func (_IModuleGuard *IModuleGuardSession) CheckModuleTransaction(to common.Address, value *big.Int, data []byte, operation uint8, module common.Address) (*types.Transaction, error) {
	return _IModuleGuard.Contract.CheckModuleTransaction(&_IModuleGuard.TransactOpts, to, value, data, operation, module)
}

// CheckModuleTransaction is a paid mutator transaction binding the contract method 0x728c2972.
//
// Solidity: function checkModuleTransaction(address to, uint256 value, bytes data, uint8 operation, address module) returns(bytes32 moduleTxHash)
func (_IModuleGuard *IModuleGuardTransactorSession) CheckModuleTransaction(to common.Address, value *big.Int, data []byte, operation uint8, module common.Address) (*types.Transaction, error) {
	return _IModuleGuard.Contract.CheckModuleTransaction(&_IModuleGuard.TransactOpts, to, value, data, operation, module)
}

// ITransactionGuardMetaData contains all meta data concerning the ITransactionGuard contract.
var ITransactionGuardMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"name\":\"checkAfterExecution\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"safeTxGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasPrice\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"gasToken\",\"type\":\"address\"},{\"internalType\":\"addresspayable\",\"name\":\"refundReceiver\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"},{\"internalType\":\"address\",\"name\":\"msgSender\",\"type\":\"address\"}],\"name\":\"checkTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// ITransactionGuardABI is the input ABI used to generate the binding from.
// Deprecated: Use ITransactionGuardMetaData.ABI instead.
var ITransactionGuardABI = ITransactionGuardMetaData.ABI

// ITransactionGuard is an auto generated Go binding around an Ethereum contract.
type ITransactionGuard struct {
	ITransactionGuardCaller     // Read-only binding to the contract
	ITransactionGuardTransactor // Write-only binding to the contract
	ITransactionGuardFilterer   // Log filterer for contract events
}

// ITransactionGuardCaller is an auto generated read-only Go binding around an Ethereum contract.
type ITransactionGuardCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ITransactionGuardTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ITransactionGuardTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ITransactionGuardFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ITransactionGuardFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ITransactionGuardSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ITransactionGuardSession struct {
	Contract     *ITransactionGuard // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// ITransactionGuardCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ITransactionGuardCallerSession struct {
	Contract *ITransactionGuardCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// ITransactionGuardTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ITransactionGuardTransactorSession struct {
	Contract     *ITransactionGuardTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// ITransactionGuardRaw is an auto generated low-level Go binding around an Ethereum contract.
type ITransactionGuardRaw struct {
	Contract *ITransactionGuard // Generic contract binding to access the raw methods on
}

// ITransactionGuardCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ITransactionGuardCallerRaw struct {
	Contract *ITransactionGuardCaller // Generic read-only contract binding to access the raw methods on
}

// ITransactionGuardTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ITransactionGuardTransactorRaw struct {
	Contract *ITransactionGuardTransactor // Generic write-only contract binding to access the raw methods on
}

// NewITransactionGuard creates a new instance of ITransactionGuard, bound to a specific deployed contract.
func NewITransactionGuard(address common.Address, backend bind.ContractBackend) (*ITransactionGuard, error) {
	contract, err := bindITransactionGuard(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ITransactionGuard{ITransactionGuardCaller: ITransactionGuardCaller{contract: contract}, ITransactionGuardTransactor: ITransactionGuardTransactor{contract: contract}, ITransactionGuardFilterer: ITransactionGuardFilterer{contract: contract}}, nil
}

// NewITransactionGuardCaller creates a new read-only instance of ITransactionGuard, bound to a specific deployed contract.
func NewITransactionGuardCaller(address common.Address, caller bind.ContractCaller) (*ITransactionGuardCaller, error) {
	contract, err := bindITransactionGuard(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ITransactionGuardCaller{contract: contract}, nil
}

// NewITransactionGuardTransactor creates a new write-only instance of ITransactionGuard, bound to a specific deployed contract.
func NewITransactionGuardTransactor(address common.Address, transactor bind.ContractTransactor) (*ITransactionGuardTransactor, error) {
	contract, err := bindITransactionGuard(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ITransactionGuardTransactor{contract: contract}, nil
}

// NewITransactionGuardFilterer creates a new log filterer instance of ITransactionGuard, bound to a specific deployed contract.
func NewITransactionGuardFilterer(address common.Address, filterer bind.ContractFilterer) (*ITransactionGuardFilterer, error) {
	contract, err := bindITransactionGuard(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ITransactionGuardFilterer{contract: contract}, nil
}

// bindITransactionGuard binds a generic wrapper to an already deployed contract.
func bindITransactionGuard(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ITransactionGuardMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ITransactionGuard *ITransactionGuardRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ITransactionGuard.Contract.ITransactionGuardCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ITransactionGuard *ITransactionGuardRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ITransactionGuard.Contract.ITransactionGuardTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ITransactionGuard *ITransactionGuardRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ITransactionGuard.Contract.ITransactionGuardTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ITransactionGuard *ITransactionGuardCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ITransactionGuard.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ITransactionGuard *ITransactionGuardTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ITransactionGuard.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ITransactionGuard *ITransactionGuardTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ITransactionGuard.Contract.contract.Transact(opts, method, params...)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ITransactionGuard *ITransactionGuardCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _ITransactionGuard.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ITransactionGuard *ITransactionGuardSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ITransactionGuard.Contract.SupportsInterface(&_ITransactionGuard.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ITransactionGuard *ITransactionGuardCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ITransactionGuard.Contract.SupportsInterface(&_ITransactionGuard.CallOpts, interfaceId)
}

// CheckAfterExecution is a paid mutator transaction binding the contract method 0x93271368.
//
// Solidity: function checkAfterExecution(bytes32 hash, bool success) returns()
func (_ITransactionGuard *ITransactionGuardTransactor) CheckAfterExecution(opts *bind.TransactOpts, hash [32]byte, success bool) (*types.Transaction, error) {
	return _ITransactionGuard.contract.Transact(opts, "checkAfterExecution", hash, success)
}

// CheckAfterExecution is a paid mutator transaction binding the contract method 0x93271368.
//
// Solidity: function checkAfterExecution(bytes32 hash, bool success) returns()
func (_ITransactionGuard *ITransactionGuardSession) CheckAfterExecution(hash [32]byte, success bool) (*types.Transaction, error) {
	return _ITransactionGuard.Contract.CheckAfterExecution(&_ITransactionGuard.TransactOpts, hash, success)
}

// CheckAfterExecution is a paid mutator transaction binding the contract method 0x93271368.
//
// Solidity: function checkAfterExecution(bytes32 hash, bool success) returns()
func (_ITransactionGuard *ITransactionGuardTransactorSession) CheckAfterExecution(hash [32]byte, success bool) (*types.Transaction, error) {
	return _ITransactionGuard.Contract.CheckAfterExecution(&_ITransactionGuard.TransactOpts, hash, success)
}

// CheckTransaction is a paid mutator transaction binding the contract method 0x75f0bb52.
//
// Solidity: function checkTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures, address msgSender) returns()
func (_ITransactionGuard *ITransactionGuardTransactor) CheckTransaction(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte, msgSender common.Address) (*types.Transaction, error) {
	return _ITransactionGuard.contract.Transact(opts, "checkTransaction", to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures, msgSender)
}

// CheckTransaction is a paid mutator transaction binding the contract method 0x75f0bb52.
//
// Solidity: function checkTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures, address msgSender) returns()
func (_ITransactionGuard *ITransactionGuardSession) CheckTransaction(to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte, msgSender common.Address) (*types.Transaction, error) {
	return _ITransactionGuard.Contract.CheckTransaction(&_ITransactionGuard.TransactOpts, to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures, msgSender)
}

// CheckTransaction is a paid mutator transaction binding the contract method 0x75f0bb52.
//
// Solidity: function checkTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures, address msgSender) returns()
func (_ITransactionGuard *ITransactionGuardTransactorSession) CheckTransaction(to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte, msgSender common.Address) (*types.Transaction, error) {
	return _ITransactionGuard.Contract.CheckTransaction(&_ITransactionGuard.TransactOpts, to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures, msgSender)
}

// ModuleManagerMetaData contains all meta data concerning the ModuleManager contract.
var ModuleManagerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"moduleGuard\",\"type\":\"address\"}],\"name\":\"ChangedModuleGuard\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"DisabledModule\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"EnabledModule\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"ExecutionFromModuleFailure\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"ExecutionFromModuleSuccess\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"prevModule\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"disableModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"enableModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"}],\"name\":\"execTransactionFromModule\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"}],\"name\":\"execTransactionFromModuleReturnData\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"start\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"pageSize\",\"type\":\"uint256\"}],\"name\":\"getModulesPaginated\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"array\",\"type\":\"address[]\"},{\"internalType\":\"address\",\"name\":\"next\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"isModuleEnabled\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"moduleGuard\",\"type\":\"address\"}],\"name\":\"setModuleGuard\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ModuleManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use ModuleManagerMetaData.ABI instead.
var ModuleManagerABI = ModuleManagerMetaData.ABI

// ModuleManager is an auto generated Go binding around an Ethereum contract.
type ModuleManager struct {
	ModuleManagerCaller     // Read-only binding to the contract
	ModuleManagerTransactor // Write-only binding to the contract
	ModuleManagerFilterer   // Log filterer for contract events
}

// ModuleManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type ModuleManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ModuleManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ModuleManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ModuleManagerSession struct {
	Contract     *ModuleManager    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ModuleManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ModuleManagerCallerSession struct {
	Contract *ModuleManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ModuleManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ModuleManagerTransactorSession struct {
	Contract     *ModuleManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ModuleManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type ModuleManagerRaw struct {
	Contract *ModuleManager // Generic contract binding to access the raw methods on
}

// ModuleManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ModuleManagerCallerRaw struct {
	Contract *ModuleManagerCaller // Generic read-only contract binding to access the raw methods on
}

// ModuleManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ModuleManagerTransactorRaw struct {
	Contract *ModuleManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewModuleManager creates a new instance of ModuleManager, bound to a specific deployed contract.
func NewModuleManager(address common.Address, backend bind.ContractBackend) (*ModuleManager, error) {
	contract, err := bindModuleManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ModuleManager{ModuleManagerCaller: ModuleManagerCaller{contract: contract}, ModuleManagerTransactor: ModuleManagerTransactor{contract: contract}, ModuleManagerFilterer: ModuleManagerFilterer{contract: contract}}, nil
}

// NewModuleManagerCaller creates a new read-only instance of ModuleManager, bound to a specific deployed contract.
func NewModuleManagerCaller(address common.Address, caller bind.ContractCaller) (*ModuleManagerCaller, error) {
	contract, err := bindModuleManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ModuleManagerCaller{contract: contract}, nil
}

// NewModuleManagerTransactor creates a new write-only instance of ModuleManager, bound to a specific deployed contract.
func NewModuleManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*ModuleManagerTransactor, error) {
	contract, err := bindModuleManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ModuleManagerTransactor{contract: contract}, nil
}

// NewModuleManagerFilterer creates a new log filterer instance of ModuleManager, bound to a specific deployed contract.
func NewModuleManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*ModuleManagerFilterer, error) {
	contract, err := bindModuleManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ModuleManagerFilterer{contract: contract}, nil
}

// bindModuleManager binds a generic wrapper to an already deployed contract.
func bindModuleManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ModuleManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ModuleManager *ModuleManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ModuleManager.Contract.ModuleManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ModuleManager *ModuleManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ModuleManager.Contract.ModuleManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ModuleManager *ModuleManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ModuleManager.Contract.ModuleManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ModuleManager *ModuleManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ModuleManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ModuleManager *ModuleManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ModuleManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ModuleManager *ModuleManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ModuleManager.Contract.contract.Transact(opts, method, params...)
}

// GetModulesPaginated is a free data retrieval call binding the contract method 0xcc2f8452.
//
// Solidity: function getModulesPaginated(address start, uint256 pageSize) view returns(address[] array, address next)
func (_ModuleManager *ModuleManagerCaller) GetModulesPaginated(opts *bind.CallOpts, start common.Address, pageSize *big.Int) (struct {
	Array []common.Address
	Next  common.Address
}, error) {
	var out []interface{}
	err := _ModuleManager.contract.Call(opts, &out, "getModulesPaginated", start, pageSize)

	outstruct := new(struct {
		Array []common.Address
		Next  common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Array = *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)
	outstruct.Next = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// GetModulesPaginated is a free data retrieval call binding the contract method 0xcc2f8452.
//
// Solidity: function getModulesPaginated(address start, uint256 pageSize) view returns(address[] array, address next)
func (_ModuleManager *ModuleManagerSession) GetModulesPaginated(start common.Address, pageSize *big.Int) (struct {
	Array []common.Address
	Next  common.Address
}, error) {
	return _ModuleManager.Contract.GetModulesPaginated(&_ModuleManager.CallOpts, start, pageSize)
}

// GetModulesPaginated is a free data retrieval call binding the contract method 0xcc2f8452.
//
// Solidity: function getModulesPaginated(address start, uint256 pageSize) view returns(address[] array, address next)
func (_ModuleManager *ModuleManagerCallerSession) GetModulesPaginated(start common.Address, pageSize *big.Int) (struct {
	Array []common.Address
	Next  common.Address
}, error) {
	return _ModuleManager.Contract.GetModulesPaginated(&_ModuleManager.CallOpts, start, pageSize)
}

// IsModuleEnabled is a free data retrieval call binding the contract method 0x2d9ad53d.
//
// Solidity: function isModuleEnabled(address module) view returns(bool)
func (_ModuleManager *ModuleManagerCaller) IsModuleEnabled(opts *bind.CallOpts, module common.Address) (bool, error) {
	var out []interface{}
	err := _ModuleManager.contract.Call(opts, &out, "isModuleEnabled", module)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsModuleEnabled is a free data retrieval call binding the contract method 0x2d9ad53d.
//
// Solidity: function isModuleEnabled(address module) view returns(bool)
func (_ModuleManager *ModuleManagerSession) IsModuleEnabled(module common.Address) (bool, error) {
	return _ModuleManager.Contract.IsModuleEnabled(&_ModuleManager.CallOpts, module)
}

// IsModuleEnabled is a free data retrieval call binding the contract method 0x2d9ad53d.
//
// Solidity: function isModuleEnabled(address module) view returns(bool)
func (_ModuleManager *ModuleManagerCallerSession) IsModuleEnabled(module common.Address) (bool, error) {
	return _ModuleManager.Contract.IsModuleEnabled(&_ModuleManager.CallOpts, module)
}

// DisableModule is a paid mutator transaction binding the contract method 0xe009cfde.
//
// Solidity: function disableModule(address prevModule, address module) returns()
func (_ModuleManager *ModuleManagerTransactor) DisableModule(opts *bind.TransactOpts, prevModule common.Address, module common.Address) (*types.Transaction, error) {
	return _ModuleManager.contract.Transact(opts, "disableModule", prevModule, module)
}

// DisableModule is a paid mutator transaction binding the contract method 0xe009cfde.
//
// Solidity: function disableModule(address prevModule, address module) returns()
func (_ModuleManager *ModuleManagerSession) DisableModule(prevModule common.Address, module common.Address) (*types.Transaction, error) {
	return _ModuleManager.Contract.DisableModule(&_ModuleManager.TransactOpts, prevModule, module)
}

// DisableModule is a paid mutator transaction binding the contract method 0xe009cfde.
//
// Solidity: function disableModule(address prevModule, address module) returns()
func (_ModuleManager *ModuleManagerTransactorSession) DisableModule(prevModule common.Address, module common.Address) (*types.Transaction, error) {
	return _ModuleManager.Contract.DisableModule(&_ModuleManager.TransactOpts, prevModule, module)
}

// EnableModule is a paid mutator transaction binding the contract method 0x610b5925.
//
// Solidity: function enableModule(address module) returns()
func (_ModuleManager *ModuleManagerTransactor) EnableModule(opts *bind.TransactOpts, module common.Address) (*types.Transaction, error) {
	return _ModuleManager.contract.Transact(opts, "enableModule", module)
}

// EnableModule is a paid mutator transaction binding the contract method 0x610b5925.
//
// Solidity: function enableModule(address module) returns()
func (_ModuleManager *ModuleManagerSession) EnableModule(module common.Address) (*types.Transaction, error) {
	return _ModuleManager.Contract.EnableModule(&_ModuleManager.TransactOpts, module)
}

// EnableModule is a paid mutator transaction binding the contract method 0x610b5925.
//
// Solidity: function enableModule(address module) returns()
func (_ModuleManager *ModuleManagerTransactorSession) EnableModule(module common.Address) (*types.Transaction, error) {
	return _ModuleManager.Contract.EnableModule(&_ModuleManager.TransactOpts, module)
}

// ExecTransactionFromModule is a paid mutator transaction binding the contract method 0x468721a7.
//
// Solidity: function execTransactionFromModule(address to, uint256 value, bytes data, uint8 operation) returns(bool success)
func (_ModuleManager *ModuleManagerTransactor) ExecTransactionFromModule(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ModuleManager.contract.Transact(opts, "execTransactionFromModule", to, value, data, operation)
}

// ExecTransactionFromModule is a paid mutator transaction binding the contract method 0x468721a7.
//
// Solidity: function execTransactionFromModule(address to, uint256 value, bytes data, uint8 operation) returns(bool success)
func (_ModuleManager *ModuleManagerSession) ExecTransactionFromModule(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ModuleManager.Contract.ExecTransactionFromModule(&_ModuleManager.TransactOpts, to, value, data, operation)
}

// ExecTransactionFromModule is a paid mutator transaction binding the contract method 0x468721a7.
//
// Solidity: function execTransactionFromModule(address to, uint256 value, bytes data, uint8 operation) returns(bool success)
func (_ModuleManager *ModuleManagerTransactorSession) ExecTransactionFromModule(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ModuleManager.Contract.ExecTransactionFromModule(&_ModuleManager.TransactOpts, to, value, data, operation)
}

// ExecTransactionFromModuleReturnData is a paid mutator transaction binding the contract method 0x5229073f.
//
// Solidity: function execTransactionFromModuleReturnData(address to, uint256 value, bytes data, uint8 operation) returns(bool success, bytes returnData)
func (_ModuleManager *ModuleManagerTransactor) ExecTransactionFromModuleReturnData(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ModuleManager.contract.Transact(opts, "execTransactionFromModuleReturnData", to, value, data, operation)
}

// ExecTransactionFromModuleReturnData is a paid mutator transaction binding the contract method 0x5229073f.
//
// Solidity: function execTransactionFromModuleReturnData(address to, uint256 value, bytes data, uint8 operation) returns(bool success, bytes returnData)
func (_ModuleManager *ModuleManagerSession) ExecTransactionFromModuleReturnData(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ModuleManager.Contract.ExecTransactionFromModuleReturnData(&_ModuleManager.TransactOpts, to, value, data, operation)
}

// ExecTransactionFromModuleReturnData is a paid mutator transaction binding the contract method 0x5229073f.
//
// Solidity: function execTransactionFromModuleReturnData(address to, uint256 value, bytes data, uint8 operation) returns(bool success, bytes returnData)
func (_ModuleManager *ModuleManagerTransactorSession) ExecTransactionFromModuleReturnData(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ModuleManager.Contract.ExecTransactionFromModuleReturnData(&_ModuleManager.TransactOpts, to, value, data, operation)
}

// SetModuleGuard is a paid mutator transaction binding the contract method 0xe068df37.
//
// Solidity: function setModuleGuard(address moduleGuard) returns()
func (_ModuleManager *ModuleManagerTransactor) SetModuleGuard(opts *bind.TransactOpts, moduleGuard common.Address) (*types.Transaction, error) {
	return _ModuleManager.contract.Transact(opts, "setModuleGuard", moduleGuard)
}

// SetModuleGuard is a paid mutator transaction binding the contract method 0xe068df37.
//
// Solidity: function setModuleGuard(address moduleGuard) returns()
func (_ModuleManager *ModuleManagerSession) SetModuleGuard(moduleGuard common.Address) (*types.Transaction, error) {
	return _ModuleManager.Contract.SetModuleGuard(&_ModuleManager.TransactOpts, moduleGuard)
}

// SetModuleGuard is a paid mutator transaction binding the contract method 0xe068df37.
//
// Solidity: function setModuleGuard(address moduleGuard) returns()
func (_ModuleManager *ModuleManagerTransactorSession) SetModuleGuard(moduleGuard common.Address) (*types.Transaction, error) {
	return _ModuleManager.Contract.SetModuleGuard(&_ModuleManager.TransactOpts, moduleGuard)
}

// ModuleManagerChangedModuleGuardIterator is returned from FilterChangedModuleGuard and is used to iterate over the raw logs and unpacked data for ChangedModuleGuard events raised by the ModuleManager contract.
type ModuleManagerChangedModuleGuardIterator struct {
	Event *ModuleManagerChangedModuleGuard // Event containing the contract specifics and raw log

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
func (it *ModuleManagerChangedModuleGuardIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ModuleManagerChangedModuleGuard)
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
		it.Event = new(ModuleManagerChangedModuleGuard)
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
func (it *ModuleManagerChangedModuleGuardIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ModuleManagerChangedModuleGuardIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ModuleManagerChangedModuleGuard represents a ChangedModuleGuard event raised by the ModuleManager contract.
type ModuleManagerChangedModuleGuard struct {
	ModuleGuard common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChangedModuleGuard is a free log retrieval operation binding the contract event 0xcd1966d6be16bc0c030cc741a06c6e0efaf8d00de2c8b6a9e11827e125de8bb8.
//
// Solidity: event ChangedModuleGuard(address indexed moduleGuard)
func (_ModuleManager *ModuleManagerFilterer) FilterChangedModuleGuard(opts *bind.FilterOpts, moduleGuard []common.Address) (*ModuleManagerChangedModuleGuardIterator, error) {

	var moduleGuardRule []interface{}
	for _, moduleGuardItem := range moduleGuard {
		moduleGuardRule = append(moduleGuardRule, moduleGuardItem)
	}

	logs, sub, err := _ModuleManager.contract.FilterLogs(opts, "ChangedModuleGuard", moduleGuardRule)
	if err != nil {
		return nil, err
	}
	return &ModuleManagerChangedModuleGuardIterator{contract: _ModuleManager.contract, event: "ChangedModuleGuard", logs: logs, sub: sub}, nil
}

// WatchChangedModuleGuard is a free log subscription operation binding the contract event 0xcd1966d6be16bc0c030cc741a06c6e0efaf8d00de2c8b6a9e11827e125de8bb8.
//
// Solidity: event ChangedModuleGuard(address indexed moduleGuard)
func (_ModuleManager *ModuleManagerFilterer) WatchChangedModuleGuard(opts *bind.WatchOpts, sink chan<- *ModuleManagerChangedModuleGuard, moduleGuard []common.Address) (event.Subscription, error) {

	var moduleGuardRule []interface{}
	for _, moduleGuardItem := range moduleGuard {
		moduleGuardRule = append(moduleGuardRule, moduleGuardItem)
	}

	logs, sub, err := _ModuleManager.contract.WatchLogs(opts, "ChangedModuleGuard", moduleGuardRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ModuleManagerChangedModuleGuard)
				if err := _ModuleManager.contract.UnpackLog(event, "ChangedModuleGuard", log); err != nil {
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

// ParseChangedModuleGuard is a log parse operation binding the contract event 0xcd1966d6be16bc0c030cc741a06c6e0efaf8d00de2c8b6a9e11827e125de8bb8.
//
// Solidity: event ChangedModuleGuard(address indexed moduleGuard)
func (_ModuleManager *ModuleManagerFilterer) ParseChangedModuleGuard(log types.Log) (*ModuleManagerChangedModuleGuard, error) {
	event := new(ModuleManagerChangedModuleGuard)
	if err := _ModuleManager.contract.UnpackLog(event, "ChangedModuleGuard", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ModuleManagerDisabledModuleIterator is returned from FilterDisabledModule and is used to iterate over the raw logs and unpacked data for DisabledModule events raised by the ModuleManager contract.
type ModuleManagerDisabledModuleIterator struct {
	Event *ModuleManagerDisabledModule // Event containing the contract specifics and raw log

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
func (it *ModuleManagerDisabledModuleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ModuleManagerDisabledModule)
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
		it.Event = new(ModuleManagerDisabledModule)
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
func (it *ModuleManagerDisabledModuleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ModuleManagerDisabledModuleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ModuleManagerDisabledModule represents a DisabledModule event raised by the ModuleManager contract.
type ModuleManagerDisabledModule struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterDisabledModule is a free log retrieval operation binding the contract event 0xaab4fa2b463f581b2b32cb3b7e3b704b9ce37cc209b5fb4d77e593ace4054276.
//
// Solidity: event DisabledModule(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) FilterDisabledModule(opts *bind.FilterOpts, module []common.Address) (*ModuleManagerDisabledModuleIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ModuleManager.contract.FilterLogs(opts, "DisabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return &ModuleManagerDisabledModuleIterator{contract: _ModuleManager.contract, event: "DisabledModule", logs: logs, sub: sub}, nil
}

// WatchDisabledModule is a free log subscription operation binding the contract event 0xaab4fa2b463f581b2b32cb3b7e3b704b9ce37cc209b5fb4d77e593ace4054276.
//
// Solidity: event DisabledModule(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) WatchDisabledModule(opts *bind.WatchOpts, sink chan<- *ModuleManagerDisabledModule, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ModuleManager.contract.WatchLogs(opts, "DisabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ModuleManagerDisabledModule)
				if err := _ModuleManager.contract.UnpackLog(event, "DisabledModule", log); err != nil {
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

// ParseDisabledModule is a log parse operation binding the contract event 0xaab4fa2b463f581b2b32cb3b7e3b704b9ce37cc209b5fb4d77e593ace4054276.
//
// Solidity: event DisabledModule(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) ParseDisabledModule(log types.Log) (*ModuleManagerDisabledModule, error) {
	event := new(ModuleManagerDisabledModule)
	if err := _ModuleManager.contract.UnpackLog(event, "DisabledModule", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ModuleManagerEnabledModuleIterator is returned from FilterEnabledModule and is used to iterate over the raw logs and unpacked data for EnabledModule events raised by the ModuleManager contract.
type ModuleManagerEnabledModuleIterator struct {
	Event *ModuleManagerEnabledModule // Event containing the contract specifics and raw log

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
func (it *ModuleManagerEnabledModuleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ModuleManagerEnabledModule)
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
		it.Event = new(ModuleManagerEnabledModule)
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
func (it *ModuleManagerEnabledModuleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ModuleManagerEnabledModuleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ModuleManagerEnabledModule represents a EnabledModule event raised by the ModuleManager contract.
type ModuleManagerEnabledModule struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterEnabledModule is a free log retrieval operation binding the contract event 0xecdf3a3effea5783a3c4c2140e677577666428d44ed9d474a0b3a4c9943f8440.
//
// Solidity: event EnabledModule(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) FilterEnabledModule(opts *bind.FilterOpts, module []common.Address) (*ModuleManagerEnabledModuleIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ModuleManager.contract.FilterLogs(opts, "EnabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return &ModuleManagerEnabledModuleIterator{contract: _ModuleManager.contract, event: "EnabledModule", logs: logs, sub: sub}, nil
}

// WatchEnabledModule is a free log subscription operation binding the contract event 0xecdf3a3effea5783a3c4c2140e677577666428d44ed9d474a0b3a4c9943f8440.
//
// Solidity: event EnabledModule(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) WatchEnabledModule(opts *bind.WatchOpts, sink chan<- *ModuleManagerEnabledModule, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ModuleManager.contract.WatchLogs(opts, "EnabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ModuleManagerEnabledModule)
				if err := _ModuleManager.contract.UnpackLog(event, "EnabledModule", log); err != nil {
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

// ParseEnabledModule is a log parse operation binding the contract event 0xecdf3a3effea5783a3c4c2140e677577666428d44ed9d474a0b3a4c9943f8440.
//
// Solidity: event EnabledModule(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) ParseEnabledModule(log types.Log) (*ModuleManagerEnabledModule, error) {
	event := new(ModuleManagerEnabledModule)
	if err := _ModuleManager.contract.UnpackLog(event, "EnabledModule", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ModuleManagerExecutionFromModuleFailureIterator is returned from FilterExecutionFromModuleFailure and is used to iterate over the raw logs and unpacked data for ExecutionFromModuleFailure events raised by the ModuleManager contract.
type ModuleManagerExecutionFromModuleFailureIterator struct {
	Event *ModuleManagerExecutionFromModuleFailure // Event containing the contract specifics and raw log

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
func (it *ModuleManagerExecutionFromModuleFailureIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ModuleManagerExecutionFromModuleFailure)
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
		it.Event = new(ModuleManagerExecutionFromModuleFailure)
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
func (it *ModuleManagerExecutionFromModuleFailureIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ModuleManagerExecutionFromModuleFailureIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ModuleManagerExecutionFromModuleFailure represents a ExecutionFromModuleFailure event raised by the ModuleManager contract.
type ModuleManagerExecutionFromModuleFailure struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterExecutionFromModuleFailure is a free log retrieval operation binding the contract event 0xacd2c8702804128fdb0db2bb49f6d127dd0181c13fd45dbfe16de0930e2bd375.
//
// Solidity: event ExecutionFromModuleFailure(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) FilterExecutionFromModuleFailure(opts *bind.FilterOpts, module []common.Address) (*ModuleManagerExecutionFromModuleFailureIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ModuleManager.contract.FilterLogs(opts, "ExecutionFromModuleFailure", moduleRule)
	if err != nil {
		return nil, err
	}
	return &ModuleManagerExecutionFromModuleFailureIterator{contract: _ModuleManager.contract, event: "ExecutionFromModuleFailure", logs: logs, sub: sub}, nil
}

// WatchExecutionFromModuleFailure is a free log subscription operation binding the contract event 0xacd2c8702804128fdb0db2bb49f6d127dd0181c13fd45dbfe16de0930e2bd375.
//
// Solidity: event ExecutionFromModuleFailure(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) WatchExecutionFromModuleFailure(opts *bind.WatchOpts, sink chan<- *ModuleManagerExecutionFromModuleFailure, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ModuleManager.contract.WatchLogs(opts, "ExecutionFromModuleFailure", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ModuleManagerExecutionFromModuleFailure)
				if err := _ModuleManager.contract.UnpackLog(event, "ExecutionFromModuleFailure", log); err != nil {
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

// ParseExecutionFromModuleFailure is a log parse operation binding the contract event 0xacd2c8702804128fdb0db2bb49f6d127dd0181c13fd45dbfe16de0930e2bd375.
//
// Solidity: event ExecutionFromModuleFailure(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) ParseExecutionFromModuleFailure(log types.Log) (*ModuleManagerExecutionFromModuleFailure, error) {
	event := new(ModuleManagerExecutionFromModuleFailure)
	if err := _ModuleManager.contract.UnpackLog(event, "ExecutionFromModuleFailure", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ModuleManagerExecutionFromModuleSuccessIterator is returned from FilterExecutionFromModuleSuccess and is used to iterate over the raw logs and unpacked data for ExecutionFromModuleSuccess events raised by the ModuleManager contract.
type ModuleManagerExecutionFromModuleSuccessIterator struct {
	Event *ModuleManagerExecutionFromModuleSuccess // Event containing the contract specifics and raw log

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
func (it *ModuleManagerExecutionFromModuleSuccessIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ModuleManagerExecutionFromModuleSuccess)
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
		it.Event = new(ModuleManagerExecutionFromModuleSuccess)
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
func (it *ModuleManagerExecutionFromModuleSuccessIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ModuleManagerExecutionFromModuleSuccessIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ModuleManagerExecutionFromModuleSuccess represents a ExecutionFromModuleSuccess event raised by the ModuleManager contract.
type ModuleManagerExecutionFromModuleSuccess struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterExecutionFromModuleSuccess is a free log retrieval operation binding the contract event 0x6895c13664aa4f67288b25d7a21d7aaa34916e355fb9b6fae0a139a9085becb8.
//
// Solidity: event ExecutionFromModuleSuccess(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) FilterExecutionFromModuleSuccess(opts *bind.FilterOpts, module []common.Address) (*ModuleManagerExecutionFromModuleSuccessIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ModuleManager.contract.FilterLogs(opts, "ExecutionFromModuleSuccess", moduleRule)
	if err != nil {
		return nil, err
	}
	return &ModuleManagerExecutionFromModuleSuccessIterator{contract: _ModuleManager.contract, event: "ExecutionFromModuleSuccess", logs: logs, sub: sub}, nil
}

// WatchExecutionFromModuleSuccess is a free log subscription operation binding the contract event 0x6895c13664aa4f67288b25d7a21d7aaa34916e355fb9b6fae0a139a9085becb8.
//
// Solidity: event ExecutionFromModuleSuccess(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) WatchExecutionFromModuleSuccess(opts *bind.WatchOpts, sink chan<- *ModuleManagerExecutionFromModuleSuccess, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ModuleManager.contract.WatchLogs(opts, "ExecutionFromModuleSuccess", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ModuleManagerExecutionFromModuleSuccess)
				if err := _ModuleManager.contract.UnpackLog(event, "ExecutionFromModuleSuccess", log); err != nil {
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

// ParseExecutionFromModuleSuccess is a log parse operation binding the contract event 0x6895c13664aa4f67288b25d7a21d7aaa34916e355fb9b6fae0a139a9085becb8.
//
// Solidity: event ExecutionFromModuleSuccess(address indexed module)
func (_ModuleManager *ModuleManagerFilterer) ParseExecutionFromModuleSuccess(log types.Log) (*ModuleManagerExecutionFromModuleSuccess, error) {
	event := new(ModuleManagerExecutionFromModuleSuccess)
	if err := _ModuleManager.contract.UnpackLog(event, "ExecutionFromModuleSuccess", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OwnerManagerMetaData contains all meta data concerning the OwnerManager contract.
var OwnerManagerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"AddedOwner\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"threshold\",\"type\":\"uint256\"}],\"name\":\"ChangedThreshold\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"RemovedOwner\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"}],\"name\":\"addOwnerWithThreshold\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"}],\"name\":\"changeThreshold\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getOwners\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getThreshold\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"isOwner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"prevOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"}],\"name\":\"removeOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"prevOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"swapOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// OwnerManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use OwnerManagerMetaData.ABI instead.
var OwnerManagerABI = OwnerManagerMetaData.ABI

// OwnerManager is an auto generated Go binding around an Ethereum contract.
type OwnerManager struct {
	OwnerManagerCaller     // Read-only binding to the contract
	OwnerManagerTransactor // Write-only binding to the contract
	OwnerManagerFilterer   // Log filterer for contract events
}

// OwnerManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type OwnerManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnerManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OwnerManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnerManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OwnerManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnerManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OwnerManagerSession struct {
	Contract     *OwnerManager     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OwnerManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OwnerManagerCallerSession struct {
	Contract *OwnerManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// OwnerManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OwnerManagerTransactorSession struct {
	Contract     *OwnerManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// OwnerManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type OwnerManagerRaw struct {
	Contract *OwnerManager // Generic contract binding to access the raw methods on
}

// OwnerManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OwnerManagerCallerRaw struct {
	Contract *OwnerManagerCaller // Generic read-only contract binding to access the raw methods on
}

// OwnerManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OwnerManagerTransactorRaw struct {
	Contract *OwnerManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOwnerManager creates a new instance of OwnerManager, bound to a specific deployed contract.
func NewOwnerManager(address common.Address, backend bind.ContractBackend) (*OwnerManager, error) {
	contract, err := bindOwnerManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OwnerManager{OwnerManagerCaller: OwnerManagerCaller{contract: contract}, OwnerManagerTransactor: OwnerManagerTransactor{contract: contract}, OwnerManagerFilterer: OwnerManagerFilterer{contract: contract}}, nil
}

// NewOwnerManagerCaller creates a new read-only instance of OwnerManager, bound to a specific deployed contract.
func NewOwnerManagerCaller(address common.Address, caller bind.ContractCaller) (*OwnerManagerCaller, error) {
	contract, err := bindOwnerManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OwnerManagerCaller{contract: contract}, nil
}

// NewOwnerManagerTransactor creates a new write-only instance of OwnerManager, bound to a specific deployed contract.
func NewOwnerManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*OwnerManagerTransactor, error) {
	contract, err := bindOwnerManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OwnerManagerTransactor{contract: contract}, nil
}

// NewOwnerManagerFilterer creates a new log filterer instance of OwnerManager, bound to a specific deployed contract.
func NewOwnerManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*OwnerManagerFilterer, error) {
	contract, err := bindOwnerManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OwnerManagerFilterer{contract: contract}, nil
}

// bindOwnerManager binds a generic wrapper to an already deployed contract.
func bindOwnerManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OwnerManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OwnerManager *OwnerManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OwnerManager.Contract.OwnerManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OwnerManager *OwnerManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OwnerManager.Contract.OwnerManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OwnerManager *OwnerManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OwnerManager.Contract.OwnerManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OwnerManager *OwnerManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OwnerManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OwnerManager *OwnerManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OwnerManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OwnerManager *OwnerManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OwnerManager.Contract.contract.Transact(opts, method, params...)
}

// GetOwners is a free data retrieval call binding the contract method 0xa0e67e2b.
//
// Solidity: function getOwners() view returns(address[])
func (_OwnerManager *OwnerManagerCaller) GetOwners(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _OwnerManager.contract.Call(opts, &out, "getOwners")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetOwners is a free data retrieval call binding the contract method 0xa0e67e2b.
//
// Solidity: function getOwners() view returns(address[])
func (_OwnerManager *OwnerManagerSession) GetOwners() ([]common.Address, error) {
	return _OwnerManager.Contract.GetOwners(&_OwnerManager.CallOpts)
}

// GetOwners is a free data retrieval call binding the contract method 0xa0e67e2b.
//
// Solidity: function getOwners() view returns(address[])
func (_OwnerManager *OwnerManagerCallerSession) GetOwners() ([]common.Address, error) {
	return _OwnerManager.Contract.GetOwners(&_OwnerManager.CallOpts)
}

// GetThreshold is a free data retrieval call binding the contract method 0xe75235b8.
//
// Solidity: function getThreshold() view returns(uint256)
func (_OwnerManager *OwnerManagerCaller) GetThreshold(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OwnerManager.contract.Call(opts, &out, "getThreshold")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetThreshold is a free data retrieval call binding the contract method 0xe75235b8.
//
// Solidity: function getThreshold() view returns(uint256)
func (_OwnerManager *OwnerManagerSession) GetThreshold() (*big.Int, error) {
	return _OwnerManager.Contract.GetThreshold(&_OwnerManager.CallOpts)
}

// GetThreshold is a free data retrieval call binding the contract method 0xe75235b8.
//
// Solidity: function getThreshold() view returns(uint256)
func (_OwnerManager *OwnerManagerCallerSession) GetThreshold() (*big.Int, error) {
	return _OwnerManager.Contract.GetThreshold(&_OwnerManager.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address owner) view returns(bool)
func (_OwnerManager *OwnerManagerCaller) IsOwner(opts *bind.CallOpts, owner common.Address) (bool, error) {
	var out []interface{}
	err := _OwnerManager.contract.Call(opts, &out, "isOwner", owner)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address owner) view returns(bool)
func (_OwnerManager *OwnerManagerSession) IsOwner(owner common.Address) (bool, error) {
	return _OwnerManager.Contract.IsOwner(&_OwnerManager.CallOpts, owner)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address owner) view returns(bool)
func (_OwnerManager *OwnerManagerCallerSession) IsOwner(owner common.Address) (bool, error) {
	return _OwnerManager.Contract.IsOwner(&_OwnerManager.CallOpts, owner)
}

// AddOwnerWithThreshold is a paid mutator transaction binding the contract method 0x0d582f13.
//
// Solidity: function addOwnerWithThreshold(address owner, uint256 _threshold) returns()
func (_OwnerManager *OwnerManagerTransactor) AddOwnerWithThreshold(opts *bind.TransactOpts, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _OwnerManager.contract.Transact(opts, "addOwnerWithThreshold", owner, _threshold)
}

// AddOwnerWithThreshold is a paid mutator transaction binding the contract method 0x0d582f13.
//
// Solidity: function addOwnerWithThreshold(address owner, uint256 _threshold) returns()
func (_OwnerManager *OwnerManagerSession) AddOwnerWithThreshold(owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _OwnerManager.Contract.AddOwnerWithThreshold(&_OwnerManager.TransactOpts, owner, _threshold)
}

// AddOwnerWithThreshold is a paid mutator transaction binding the contract method 0x0d582f13.
//
// Solidity: function addOwnerWithThreshold(address owner, uint256 _threshold) returns()
func (_OwnerManager *OwnerManagerTransactorSession) AddOwnerWithThreshold(owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _OwnerManager.Contract.AddOwnerWithThreshold(&_OwnerManager.TransactOpts, owner, _threshold)
}

// ChangeThreshold is a paid mutator transaction binding the contract method 0x694e80c3.
//
// Solidity: function changeThreshold(uint256 _threshold) returns()
func (_OwnerManager *OwnerManagerTransactor) ChangeThreshold(opts *bind.TransactOpts, _threshold *big.Int) (*types.Transaction, error) {
	return _OwnerManager.contract.Transact(opts, "changeThreshold", _threshold)
}

// ChangeThreshold is a paid mutator transaction binding the contract method 0x694e80c3.
//
// Solidity: function changeThreshold(uint256 _threshold) returns()
func (_OwnerManager *OwnerManagerSession) ChangeThreshold(_threshold *big.Int) (*types.Transaction, error) {
	return _OwnerManager.Contract.ChangeThreshold(&_OwnerManager.TransactOpts, _threshold)
}

// ChangeThreshold is a paid mutator transaction binding the contract method 0x694e80c3.
//
// Solidity: function changeThreshold(uint256 _threshold) returns()
func (_OwnerManager *OwnerManagerTransactorSession) ChangeThreshold(_threshold *big.Int) (*types.Transaction, error) {
	return _OwnerManager.Contract.ChangeThreshold(&_OwnerManager.TransactOpts, _threshold)
}

// RemoveOwner is a paid mutator transaction binding the contract method 0xf8dc5dd9.
//
// Solidity: function removeOwner(address prevOwner, address owner, uint256 _threshold) returns()
func (_OwnerManager *OwnerManagerTransactor) RemoveOwner(opts *bind.TransactOpts, prevOwner common.Address, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _OwnerManager.contract.Transact(opts, "removeOwner", prevOwner, owner, _threshold)
}

// RemoveOwner is a paid mutator transaction binding the contract method 0xf8dc5dd9.
//
// Solidity: function removeOwner(address prevOwner, address owner, uint256 _threshold) returns()
func (_OwnerManager *OwnerManagerSession) RemoveOwner(prevOwner common.Address, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _OwnerManager.Contract.RemoveOwner(&_OwnerManager.TransactOpts, prevOwner, owner, _threshold)
}

// RemoveOwner is a paid mutator transaction binding the contract method 0xf8dc5dd9.
//
// Solidity: function removeOwner(address prevOwner, address owner, uint256 _threshold) returns()
func (_OwnerManager *OwnerManagerTransactorSession) RemoveOwner(prevOwner common.Address, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _OwnerManager.Contract.RemoveOwner(&_OwnerManager.TransactOpts, prevOwner, owner, _threshold)
}

// SwapOwner is a paid mutator transaction binding the contract method 0xe318b52b.
//
// Solidity: function swapOwner(address prevOwner, address oldOwner, address newOwner) returns()
func (_OwnerManager *OwnerManagerTransactor) SwapOwner(opts *bind.TransactOpts, prevOwner common.Address, oldOwner common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _OwnerManager.contract.Transact(opts, "swapOwner", prevOwner, oldOwner, newOwner)
}

// SwapOwner is a paid mutator transaction binding the contract method 0xe318b52b.
//
// Solidity: function swapOwner(address prevOwner, address oldOwner, address newOwner) returns()
func (_OwnerManager *OwnerManagerSession) SwapOwner(prevOwner common.Address, oldOwner common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _OwnerManager.Contract.SwapOwner(&_OwnerManager.TransactOpts, prevOwner, oldOwner, newOwner)
}

// SwapOwner is a paid mutator transaction binding the contract method 0xe318b52b.
//
// Solidity: function swapOwner(address prevOwner, address oldOwner, address newOwner) returns()
func (_OwnerManager *OwnerManagerTransactorSession) SwapOwner(prevOwner common.Address, oldOwner common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _OwnerManager.Contract.SwapOwner(&_OwnerManager.TransactOpts, prevOwner, oldOwner, newOwner)
}

// OwnerManagerAddedOwnerIterator is returned from FilterAddedOwner and is used to iterate over the raw logs and unpacked data for AddedOwner events raised by the OwnerManager contract.
type OwnerManagerAddedOwnerIterator struct {
	Event *OwnerManagerAddedOwner // Event containing the contract specifics and raw log

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
func (it *OwnerManagerAddedOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OwnerManagerAddedOwner)
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
		it.Event = new(OwnerManagerAddedOwner)
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
func (it *OwnerManagerAddedOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OwnerManagerAddedOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OwnerManagerAddedOwner represents a AddedOwner event raised by the OwnerManager contract.
type OwnerManagerAddedOwner struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterAddedOwner is a free log retrieval operation binding the contract event 0x9465fa0c962cc76958e6373a993326400c1c94f8be2fe3a952adfa7f60b2ea26.
//
// Solidity: event AddedOwner(address indexed owner)
func (_OwnerManager *OwnerManagerFilterer) FilterAddedOwner(opts *bind.FilterOpts, owner []common.Address) (*OwnerManagerAddedOwnerIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _OwnerManager.contract.FilterLogs(opts, "AddedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return &OwnerManagerAddedOwnerIterator{contract: _OwnerManager.contract, event: "AddedOwner", logs: logs, sub: sub}, nil
}

// WatchAddedOwner is a free log subscription operation binding the contract event 0x9465fa0c962cc76958e6373a993326400c1c94f8be2fe3a952adfa7f60b2ea26.
//
// Solidity: event AddedOwner(address indexed owner)
func (_OwnerManager *OwnerManagerFilterer) WatchAddedOwner(opts *bind.WatchOpts, sink chan<- *OwnerManagerAddedOwner, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _OwnerManager.contract.WatchLogs(opts, "AddedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OwnerManagerAddedOwner)
				if err := _OwnerManager.contract.UnpackLog(event, "AddedOwner", log); err != nil {
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

// ParseAddedOwner is a log parse operation binding the contract event 0x9465fa0c962cc76958e6373a993326400c1c94f8be2fe3a952adfa7f60b2ea26.
//
// Solidity: event AddedOwner(address indexed owner)
func (_OwnerManager *OwnerManagerFilterer) ParseAddedOwner(log types.Log) (*OwnerManagerAddedOwner, error) {
	event := new(OwnerManagerAddedOwner)
	if err := _OwnerManager.contract.UnpackLog(event, "AddedOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OwnerManagerChangedThresholdIterator is returned from FilterChangedThreshold and is used to iterate over the raw logs and unpacked data for ChangedThreshold events raised by the OwnerManager contract.
type OwnerManagerChangedThresholdIterator struct {
	Event *OwnerManagerChangedThreshold // Event containing the contract specifics and raw log

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
func (it *OwnerManagerChangedThresholdIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OwnerManagerChangedThreshold)
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
		it.Event = new(OwnerManagerChangedThreshold)
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
func (it *OwnerManagerChangedThresholdIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OwnerManagerChangedThresholdIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OwnerManagerChangedThreshold represents a ChangedThreshold event raised by the OwnerManager contract.
type OwnerManagerChangedThreshold struct {
	Threshold *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterChangedThreshold is a free log retrieval operation binding the contract event 0x610f7ff2b304ae8903c3de74c60c6ab1f7d6226b3f52c5161905bb5ad4039c93.
//
// Solidity: event ChangedThreshold(uint256 threshold)
func (_OwnerManager *OwnerManagerFilterer) FilterChangedThreshold(opts *bind.FilterOpts) (*OwnerManagerChangedThresholdIterator, error) {

	logs, sub, err := _OwnerManager.contract.FilterLogs(opts, "ChangedThreshold")
	if err != nil {
		return nil, err
	}
	return &OwnerManagerChangedThresholdIterator{contract: _OwnerManager.contract, event: "ChangedThreshold", logs: logs, sub: sub}, nil
}

// WatchChangedThreshold is a free log subscription operation binding the contract event 0x610f7ff2b304ae8903c3de74c60c6ab1f7d6226b3f52c5161905bb5ad4039c93.
//
// Solidity: event ChangedThreshold(uint256 threshold)
func (_OwnerManager *OwnerManagerFilterer) WatchChangedThreshold(opts *bind.WatchOpts, sink chan<- *OwnerManagerChangedThreshold) (event.Subscription, error) {

	logs, sub, err := _OwnerManager.contract.WatchLogs(opts, "ChangedThreshold")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OwnerManagerChangedThreshold)
				if err := _OwnerManager.contract.UnpackLog(event, "ChangedThreshold", log); err != nil {
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

// ParseChangedThreshold is a log parse operation binding the contract event 0x610f7ff2b304ae8903c3de74c60c6ab1f7d6226b3f52c5161905bb5ad4039c93.
//
// Solidity: event ChangedThreshold(uint256 threshold)
func (_OwnerManager *OwnerManagerFilterer) ParseChangedThreshold(log types.Log) (*OwnerManagerChangedThreshold, error) {
	event := new(OwnerManagerChangedThreshold)
	if err := _OwnerManager.contract.UnpackLog(event, "ChangedThreshold", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OwnerManagerRemovedOwnerIterator is returned from FilterRemovedOwner and is used to iterate over the raw logs and unpacked data for RemovedOwner events raised by the OwnerManager contract.
type OwnerManagerRemovedOwnerIterator struct {
	Event *OwnerManagerRemovedOwner // Event containing the contract specifics and raw log

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
func (it *OwnerManagerRemovedOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OwnerManagerRemovedOwner)
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
		it.Event = new(OwnerManagerRemovedOwner)
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
func (it *OwnerManagerRemovedOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OwnerManagerRemovedOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OwnerManagerRemovedOwner represents a RemovedOwner event raised by the OwnerManager contract.
type OwnerManagerRemovedOwner struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterRemovedOwner is a free log retrieval operation binding the contract event 0xf8d49fc529812e9a7c5c50e69c20f0dccc0db8fa95c98bc58cc9a4f1c1299eaf.
//
// Solidity: event RemovedOwner(address indexed owner)
func (_OwnerManager *OwnerManagerFilterer) FilterRemovedOwner(opts *bind.FilterOpts, owner []common.Address) (*OwnerManagerRemovedOwnerIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _OwnerManager.contract.FilterLogs(opts, "RemovedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return &OwnerManagerRemovedOwnerIterator{contract: _OwnerManager.contract, event: "RemovedOwner", logs: logs, sub: sub}, nil
}

// WatchRemovedOwner is a free log subscription operation binding the contract event 0xf8d49fc529812e9a7c5c50e69c20f0dccc0db8fa95c98bc58cc9a4f1c1299eaf.
//
// Solidity: event RemovedOwner(address indexed owner)
func (_OwnerManager *OwnerManagerFilterer) WatchRemovedOwner(opts *bind.WatchOpts, sink chan<- *OwnerManagerRemovedOwner, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _OwnerManager.contract.WatchLogs(opts, "RemovedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OwnerManagerRemovedOwner)
				if err := _OwnerManager.contract.UnpackLog(event, "RemovedOwner", log); err != nil {
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

// ParseRemovedOwner is a log parse operation binding the contract event 0xf8d49fc529812e9a7c5c50e69c20f0dccc0db8fa95c98bc58cc9a4f1c1299eaf.
//
// Solidity: event RemovedOwner(address indexed owner)
func (_OwnerManager *OwnerManagerFilterer) ParseRemovedOwner(log types.Log) (*OwnerManagerRemovedOwner, error) {
	event := new(OwnerManagerRemovedOwner)
	if err := _OwnerManager.contract.UnpackLog(event, "RemovedOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
