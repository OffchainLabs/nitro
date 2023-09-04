// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package precompilesgen

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

// ArbAddressTableMetaData contains all meta data concerning the ArbAddressTable contract.
var ArbAddressTableMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"addressExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"compress\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"buf\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"offset\",\"type\":\"uint256\"}],\"name\":\"decompress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"lookup\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"lookupIndex\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"register\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"size\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// ArbAddressTableABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbAddressTableMetaData.ABI instead.
var ArbAddressTableABI = ArbAddressTableMetaData.ABI

// ArbAddressTable is an auto generated Go binding around an Ethereum contract.
type ArbAddressTable struct {
	ArbAddressTableCaller     // Read-only binding to the contract
	ArbAddressTableTransactor // Write-only binding to the contract
	ArbAddressTableFilterer   // Log filterer for contract events
}

// ArbAddressTableCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbAddressTableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbAddressTableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbAddressTableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbAddressTableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbAddressTableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbAddressTableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbAddressTableSession struct {
	Contract     *ArbAddressTable  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbAddressTableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbAddressTableCallerSession struct {
	Contract *ArbAddressTableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// ArbAddressTableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbAddressTableTransactorSession struct {
	Contract     *ArbAddressTableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// ArbAddressTableRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbAddressTableRaw struct {
	Contract *ArbAddressTable // Generic contract binding to access the raw methods on
}

// ArbAddressTableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbAddressTableCallerRaw struct {
	Contract *ArbAddressTableCaller // Generic read-only contract binding to access the raw methods on
}

// ArbAddressTableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbAddressTableTransactorRaw struct {
	Contract *ArbAddressTableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbAddressTable creates a new instance of ArbAddressTable, bound to a specific deployed contract.
func NewArbAddressTable(address common.Address, backend bind.ContractBackend) (*ArbAddressTable, error) {
	contract, err := bindArbAddressTable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbAddressTable{ArbAddressTableCaller: ArbAddressTableCaller{contract: contract}, ArbAddressTableTransactor: ArbAddressTableTransactor{contract: contract}, ArbAddressTableFilterer: ArbAddressTableFilterer{contract: contract}}, nil
}

// NewArbAddressTableCaller creates a new read-only instance of ArbAddressTable, bound to a specific deployed contract.
func NewArbAddressTableCaller(address common.Address, caller bind.ContractCaller) (*ArbAddressTableCaller, error) {
	contract, err := bindArbAddressTable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbAddressTableCaller{contract: contract}, nil
}

// NewArbAddressTableTransactor creates a new write-only instance of ArbAddressTable, bound to a specific deployed contract.
func NewArbAddressTableTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbAddressTableTransactor, error) {
	contract, err := bindArbAddressTable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbAddressTableTransactor{contract: contract}, nil
}

// NewArbAddressTableFilterer creates a new log filterer instance of ArbAddressTable, bound to a specific deployed contract.
func NewArbAddressTableFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbAddressTableFilterer, error) {
	contract, err := bindArbAddressTable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbAddressTableFilterer{contract: contract}, nil
}

// bindArbAddressTable binds a generic wrapper to an already deployed contract.
func bindArbAddressTable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbAddressTableMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbAddressTable *ArbAddressTableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbAddressTable.Contract.ArbAddressTableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbAddressTable *ArbAddressTableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbAddressTable.Contract.ArbAddressTableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbAddressTable *ArbAddressTableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbAddressTable.Contract.ArbAddressTableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbAddressTable *ArbAddressTableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbAddressTable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbAddressTable *ArbAddressTableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbAddressTable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbAddressTable *ArbAddressTableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbAddressTable.Contract.contract.Transact(opts, method, params...)
}

// AddressExists is a free data retrieval call binding the contract method 0xa5025222.
//
// Solidity: function addressExists(address addr) view returns(bool)
func (_ArbAddressTable *ArbAddressTableCaller) AddressExists(opts *bind.CallOpts, addr common.Address) (bool, error) {
	var out []interface{}
	err := _ArbAddressTable.contract.Call(opts, &out, "addressExists", addr)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AddressExists is a free data retrieval call binding the contract method 0xa5025222.
//
// Solidity: function addressExists(address addr) view returns(bool)
func (_ArbAddressTable *ArbAddressTableSession) AddressExists(addr common.Address) (bool, error) {
	return _ArbAddressTable.Contract.AddressExists(&_ArbAddressTable.CallOpts, addr)
}

// AddressExists is a free data retrieval call binding the contract method 0xa5025222.
//
// Solidity: function addressExists(address addr) view returns(bool)
func (_ArbAddressTable *ArbAddressTableCallerSession) AddressExists(addr common.Address) (bool, error) {
	return _ArbAddressTable.Contract.AddressExists(&_ArbAddressTable.CallOpts, addr)
}

// Decompress is a free data retrieval call binding the contract method 0x31862ada.
//
// Solidity: function decompress(bytes buf, uint256 offset) view returns(address, uint256)
func (_ArbAddressTable *ArbAddressTableCaller) Decompress(opts *bind.CallOpts, buf []byte, offset *big.Int) (common.Address, *big.Int, error) {
	var out []interface{}
	err := _ArbAddressTable.contract.Call(opts, &out, "decompress", buf, offset)

	if err != nil {
		return *new(common.Address), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// Decompress is a free data retrieval call binding the contract method 0x31862ada.
//
// Solidity: function decompress(bytes buf, uint256 offset) view returns(address, uint256)
func (_ArbAddressTable *ArbAddressTableSession) Decompress(buf []byte, offset *big.Int) (common.Address, *big.Int, error) {
	return _ArbAddressTable.Contract.Decompress(&_ArbAddressTable.CallOpts, buf, offset)
}

// Decompress is a free data retrieval call binding the contract method 0x31862ada.
//
// Solidity: function decompress(bytes buf, uint256 offset) view returns(address, uint256)
func (_ArbAddressTable *ArbAddressTableCallerSession) Decompress(buf []byte, offset *big.Int) (common.Address, *big.Int, error) {
	return _ArbAddressTable.Contract.Decompress(&_ArbAddressTable.CallOpts, buf, offset)
}

// Lookup is a free data retrieval call binding the contract method 0xd4b6b5da.
//
// Solidity: function lookup(address addr) view returns(uint256)
func (_ArbAddressTable *ArbAddressTableCaller) Lookup(opts *bind.CallOpts, addr common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ArbAddressTable.contract.Call(opts, &out, "lookup", addr)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Lookup is a free data retrieval call binding the contract method 0xd4b6b5da.
//
// Solidity: function lookup(address addr) view returns(uint256)
func (_ArbAddressTable *ArbAddressTableSession) Lookup(addr common.Address) (*big.Int, error) {
	return _ArbAddressTable.Contract.Lookup(&_ArbAddressTable.CallOpts, addr)
}

// Lookup is a free data retrieval call binding the contract method 0xd4b6b5da.
//
// Solidity: function lookup(address addr) view returns(uint256)
func (_ArbAddressTable *ArbAddressTableCallerSession) Lookup(addr common.Address) (*big.Int, error) {
	return _ArbAddressTable.Contract.Lookup(&_ArbAddressTable.CallOpts, addr)
}

// LookupIndex is a free data retrieval call binding the contract method 0x8a186788.
//
// Solidity: function lookupIndex(uint256 index) view returns(address)
func (_ArbAddressTable *ArbAddressTableCaller) LookupIndex(opts *bind.CallOpts, index *big.Int) (common.Address, error) {
	var out []interface{}
	err := _ArbAddressTable.contract.Call(opts, &out, "lookupIndex", index)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// LookupIndex is a free data retrieval call binding the contract method 0x8a186788.
//
// Solidity: function lookupIndex(uint256 index) view returns(address)
func (_ArbAddressTable *ArbAddressTableSession) LookupIndex(index *big.Int) (common.Address, error) {
	return _ArbAddressTable.Contract.LookupIndex(&_ArbAddressTable.CallOpts, index)
}

// LookupIndex is a free data retrieval call binding the contract method 0x8a186788.
//
// Solidity: function lookupIndex(uint256 index) view returns(address)
func (_ArbAddressTable *ArbAddressTableCallerSession) LookupIndex(index *big.Int) (common.Address, error) {
	return _ArbAddressTable.Contract.LookupIndex(&_ArbAddressTable.CallOpts, index)
}

// Size is a free data retrieval call binding the contract method 0x949d225d.
//
// Solidity: function size() view returns(uint256)
func (_ArbAddressTable *ArbAddressTableCaller) Size(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbAddressTable.contract.Call(opts, &out, "size")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Size is a free data retrieval call binding the contract method 0x949d225d.
//
// Solidity: function size() view returns(uint256)
func (_ArbAddressTable *ArbAddressTableSession) Size() (*big.Int, error) {
	return _ArbAddressTable.Contract.Size(&_ArbAddressTable.CallOpts)
}

// Size is a free data retrieval call binding the contract method 0x949d225d.
//
// Solidity: function size() view returns(uint256)
func (_ArbAddressTable *ArbAddressTableCallerSession) Size() (*big.Int, error) {
	return _ArbAddressTable.Contract.Size(&_ArbAddressTable.CallOpts)
}

// Compress is a paid mutator transaction binding the contract method 0xf6a455a2.
//
// Solidity: function compress(address addr) returns(bytes)
func (_ArbAddressTable *ArbAddressTableTransactor) Compress(opts *bind.TransactOpts, addr common.Address) (*types.Transaction, error) {
	return _ArbAddressTable.contract.Transact(opts, "compress", addr)
}

// Compress is a paid mutator transaction binding the contract method 0xf6a455a2.
//
// Solidity: function compress(address addr) returns(bytes)
func (_ArbAddressTable *ArbAddressTableSession) Compress(addr common.Address) (*types.Transaction, error) {
	return _ArbAddressTable.Contract.Compress(&_ArbAddressTable.TransactOpts, addr)
}

// Compress is a paid mutator transaction binding the contract method 0xf6a455a2.
//
// Solidity: function compress(address addr) returns(bytes)
func (_ArbAddressTable *ArbAddressTableTransactorSession) Compress(addr common.Address) (*types.Transaction, error) {
	return _ArbAddressTable.Contract.Compress(&_ArbAddressTable.TransactOpts, addr)
}

// Register is a paid mutator transaction binding the contract method 0x4420e486.
//
// Solidity: function register(address addr) returns(uint256)
func (_ArbAddressTable *ArbAddressTableTransactor) Register(opts *bind.TransactOpts, addr common.Address) (*types.Transaction, error) {
	return _ArbAddressTable.contract.Transact(opts, "register", addr)
}

// Register is a paid mutator transaction binding the contract method 0x4420e486.
//
// Solidity: function register(address addr) returns(uint256)
func (_ArbAddressTable *ArbAddressTableSession) Register(addr common.Address) (*types.Transaction, error) {
	return _ArbAddressTable.Contract.Register(&_ArbAddressTable.TransactOpts, addr)
}

// Register is a paid mutator transaction binding the contract method 0x4420e486.
//
// Solidity: function register(address addr) returns(uint256)
func (_ArbAddressTable *ArbAddressTableTransactorSession) Register(addr common.Address) (*types.Transaction, error) {
	return _ArbAddressTable.Contract.Register(&_ArbAddressTable.TransactOpts, addr)
}

// ArbAggregatorMetaData contains all meta data concerning the ArbAggregator contract.
var ArbAggregatorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newBatchPoster\",\"type\":\"address\"}],\"name\":\"addBatchPoster\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBatchPosters\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getDefaultAggregator\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"batchPoster\",\"type\":\"address\"}],\"name\":\"getFeeCollector\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"getPreferredAggregator\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"aggregator\",\"type\":\"address\"}],\"name\":\"getTxBaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"batchPoster\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"newFeeCollector\",\"type\":\"address\"}],\"name\":\"setFeeCollector\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"aggregator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"feeInL1Gas\",\"type\":\"uint256\"}],\"name\":\"setTxBaseFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ArbAggregatorABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbAggregatorMetaData.ABI instead.
var ArbAggregatorABI = ArbAggregatorMetaData.ABI

// ArbAggregator is an auto generated Go binding around an Ethereum contract.
type ArbAggregator struct {
	ArbAggregatorCaller     // Read-only binding to the contract
	ArbAggregatorTransactor // Write-only binding to the contract
	ArbAggregatorFilterer   // Log filterer for contract events
}

// ArbAggregatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbAggregatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbAggregatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbAggregatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbAggregatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbAggregatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbAggregatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbAggregatorSession struct {
	Contract     *ArbAggregator    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbAggregatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbAggregatorCallerSession struct {
	Contract *ArbAggregatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ArbAggregatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbAggregatorTransactorSession struct {
	Contract     *ArbAggregatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ArbAggregatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbAggregatorRaw struct {
	Contract *ArbAggregator // Generic contract binding to access the raw methods on
}

// ArbAggregatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbAggregatorCallerRaw struct {
	Contract *ArbAggregatorCaller // Generic read-only contract binding to access the raw methods on
}

// ArbAggregatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbAggregatorTransactorRaw struct {
	Contract *ArbAggregatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbAggregator creates a new instance of ArbAggregator, bound to a specific deployed contract.
func NewArbAggregator(address common.Address, backend bind.ContractBackend) (*ArbAggregator, error) {
	contract, err := bindArbAggregator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbAggregator{ArbAggregatorCaller: ArbAggregatorCaller{contract: contract}, ArbAggregatorTransactor: ArbAggregatorTransactor{contract: contract}, ArbAggregatorFilterer: ArbAggregatorFilterer{contract: contract}}, nil
}

// NewArbAggregatorCaller creates a new read-only instance of ArbAggregator, bound to a specific deployed contract.
func NewArbAggregatorCaller(address common.Address, caller bind.ContractCaller) (*ArbAggregatorCaller, error) {
	contract, err := bindArbAggregator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbAggregatorCaller{contract: contract}, nil
}

// NewArbAggregatorTransactor creates a new write-only instance of ArbAggregator, bound to a specific deployed contract.
func NewArbAggregatorTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbAggregatorTransactor, error) {
	contract, err := bindArbAggregator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbAggregatorTransactor{contract: contract}, nil
}

// NewArbAggregatorFilterer creates a new log filterer instance of ArbAggregator, bound to a specific deployed contract.
func NewArbAggregatorFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbAggregatorFilterer, error) {
	contract, err := bindArbAggregator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbAggregatorFilterer{contract: contract}, nil
}

// bindArbAggregator binds a generic wrapper to an already deployed contract.
func bindArbAggregator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbAggregatorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbAggregator *ArbAggregatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbAggregator.Contract.ArbAggregatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbAggregator *ArbAggregatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbAggregator.Contract.ArbAggregatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbAggregator *ArbAggregatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbAggregator.Contract.ArbAggregatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbAggregator *ArbAggregatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbAggregator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbAggregator *ArbAggregatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbAggregator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbAggregator *ArbAggregatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbAggregator.Contract.contract.Transact(opts, method, params...)
}

// GetBatchPosters is a free data retrieval call binding the contract method 0xe10573a3.
//
// Solidity: function getBatchPosters() view returns(address[])
func (_ArbAggregator *ArbAggregatorCaller) GetBatchPosters(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _ArbAggregator.contract.Call(opts, &out, "getBatchPosters")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetBatchPosters is a free data retrieval call binding the contract method 0xe10573a3.
//
// Solidity: function getBatchPosters() view returns(address[])
func (_ArbAggregator *ArbAggregatorSession) GetBatchPosters() ([]common.Address, error) {
	return _ArbAggregator.Contract.GetBatchPosters(&_ArbAggregator.CallOpts)
}

// GetBatchPosters is a free data retrieval call binding the contract method 0xe10573a3.
//
// Solidity: function getBatchPosters() view returns(address[])
func (_ArbAggregator *ArbAggregatorCallerSession) GetBatchPosters() ([]common.Address, error) {
	return _ArbAggregator.Contract.GetBatchPosters(&_ArbAggregator.CallOpts)
}

// GetDefaultAggregator is a free data retrieval call binding the contract method 0x875883f2.
//
// Solidity: function getDefaultAggregator() view returns(address)
func (_ArbAggregator *ArbAggregatorCaller) GetDefaultAggregator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ArbAggregator.contract.Call(opts, &out, "getDefaultAggregator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetDefaultAggregator is a free data retrieval call binding the contract method 0x875883f2.
//
// Solidity: function getDefaultAggregator() view returns(address)
func (_ArbAggregator *ArbAggregatorSession) GetDefaultAggregator() (common.Address, error) {
	return _ArbAggregator.Contract.GetDefaultAggregator(&_ArbAggregator.CallOpts)
}

// GetDefaultAggregator is a free data retrieval call binding the contract method 0x875883f2.
//
// Solidity: function getDefaultAggregator() view returns(address)
func (_ArbAggregator *ArbAggregatorCallerSession) GetDefaultAggregator() (common.Address, error) {
	return _ArbAggregator.Contract.GetDefaultAggregator(&_ArbAggregator.CallOpts)
}

// GetFeeCollector is a free data retrieval call binding the contract method 0x9c2c5bb5.
//
// Solidity: function getFeeCollector(address batchPoster) view returns(address)
func (_ArbAggregator *ArbAggregatorCaller) GetFeeCollector(opts *bind.CallOpts, batchPoster common.Address) (common.Address, error) {
	var out []interface{}
	err := _ArbAggregator.contract.Call(opts, &out, "getFeeCollector", batchPoster)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetFeeCollector is a free data retrieval call binding the contract method 0x9c2c5bb5.
//
// Solidity: function getFeeCollector(address batchPoster) view returns(address)
func (_ArbAggregator *ArbAggregatorSession) GetFeeCollector(batchPoster common.Address) (common.Address, error) {
	return _ArbAggregator.Contract.GetFeeCollector(&_ArbAggregator.CallOpts, batchPoster)
}

// GetFeeCollector is a free data retrieval call binding the contract method 0x9c2c5bb5.
//
// Solidity: function getFeeCollector(address batchPoster) view returns(address)
func (_ArbAggregator *ArbAggregatorCallerSession) GetFeeCollector(batchPoster common.Address) (common.Address, error) {
	return _ArbAggregator.Contract.GetFeeCollector(&_ArbAggregator.CallOpts, batchPoster)
}

// GetPreferredAggregator is a free data retrieval call binding the contract method 0x52f10740.
//
// Solidity: function getPreferredAggregator(address addr) view returns(address, bool)
func (_ArbAggregator *ArbAggregatorCaller) GetPreferredAggregator(opts *bind.CallOpts, addr common.Address) (common.Address, bool, error) {
	var out []interface{}
	err := _ArbAggregator.contract.Call(opts, &out, "getPreferredAggregator", addr)

	if err != nil {
		return *new(common.Address), *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	out1 := *abi.ConvertType(out[1], new(bool)).(*bool)

	return out0, out1, err

}

// GetPreferredAggregator is a free data retrieval call binding the contract method 0x52f10740.
//
// Solidity: function getPreferredAggregator(address addr) view returns(address, bool)
func (_ArbAggregator *ArbAggregatorSession) GetPreferredAggregator(addr common.Address) (common.Address, bool, error) {
	return _ArbAggregator.Contract.GetPreferredAggregator(&_ArbAggregator.CallOpts, addr)
}

// GetPreferredAggregator is a free data retrieval call binding the contract method 0x52f10740.
//
// Solidity: function getPreferredAggregator(address addr) view returns(address, bool)
func (_ArbAggregator *ArbAggregatorCallerSession) GetPreferredAggregator(addr common.Address) (common.Address, bool, error) {
	return _ArbAggregator.Contract.GetPreferredAggregator(&_ArbAggregator.CallOpts, addr)
}

// GetTxBaseFee is a free data retrieval call binding the contract method 0x049764af.
//
// Solidity: function getTxBaseFee(address aggregator) view returns(uint256)
func (_ArbAggregator *ArbAggregatorCaller) GetTxBaseFee(opts *bind.CallOpts, aggregator common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ArbAggregator.contract.Call(opts, &out, "getTxBaseFee", aggregator)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTxBaseFee is a free data retrieval call binding the contract method 0x049764af.
//
// Solidity: function getTxBaseFee(address aggregator) view returns(uint256)
func (_ArbAggregator *ArbAggregatorSession) GetTxBaseFee(aggregator common.Address) (*big.Int, error) {
	return _ArbAggregator.Contract.GetTxBaseFee(&_ArbAggregator.CallOpts, aggregator)
}

// GetTxBaseFee is a free data retrieval call binding the contract method 0x049764af.
//
// Solidity: function getTxBaseFee(address aggregator) view returns(uint256)
func (_ArbAggregator *ArbAggregatorCallerSession) GetTxBaseFee(aggregator common.Address) (*big.Int, error) {
	return _ArbAggregator.Contract.GetTxBaseFee(&_ArbAggregator.CallOpts, aggregator)
}

// AddBatchPoster is a paid mutator transaction binding the contract method 0xdf41e1e2.
//
// Solidity: function addBatchPoster(address newBatchPoster) returns()
func (_ArbAggregator *ArbAggregatorTransactor) AddBatchPoster(opts *bind.TransactOpts, newBatchPoster common.Address) (*types.Transaction, error) {
	return _ArbAggregator.contract.Transact(opts, "addBatchPoster", newBatchPoster)
}

// AddBatchPoster is a paid mutator transaction binding the contract method 0xdf41e1e2.
//
// Solidity: function addBatchPoster(address newBatchPoster) returns()
func (_ArbAggregator *ArbAggregatorSession) AddBatchPoster(newBatchPoster common.Address) (*types.Transaction, error) {
	return _ArbAggregator.Contract.AddBatchPoster(&_ArbAggregator.TransactOpts, newBatchPoster)
}

// AddBatchPoster is a paid mutator transaction binding the contract method 0xdf41e1e2.
//
// Solidity: function addBatchPoster(address newBatchPoster) returns()
func (_ArbAggregator *ArbAggregatorTransactorSession) AddBatchPoster(newBatchPoster common.Address) (*types.Transaction, error) {
	return _ArbAggregator.Contract.AddBatchPoster(&_ArbAggregator.TransactOpts, newBatchPoster)
}

// SetFeeCollector is a paid mutator transaction binding the contract method 0x29149799.
//
// Solidity: function setFeeCollector(address batchPoster, address newFeeCollector) returns()
func (_ArbAggregator *ArbAggregatorTransactor) SetFeeCollector(opts *bind.TransactOpts, batchPoster common.Address, newFeeCollector common.Address) (*types.Transaction, error) {
	return _ArbAggregator.contract.Transact(opts, "setFeeCollector", batchPoster, newFeeCollector)
}

// SetFeeCollector is a paid mutator transaction binding the contract method 0x29149799.
//
// Solidity: function setFeeCollector(address batchPoster, address newFeeCollector) returns()
func (_ArbAggregator *ArbAggregatorSession) SetFeeCollector(batchPoster common.Address, newFeeCollector common.Address) (*types.Transaction, error) {
	return _ArbAggregator.Contract.SetFeeCollector(&_ArbAggregator.TransactOpts, batchPoster, newFeeCollector)
}

// SetFeeCollector is a paid mutator transaction binding the contract method 0x29149799.
//
// Solidity: function setFeeCollector(address batchPoster, address newFeeCollector) returns()
func (_ArbAggregator *ArbAggregatorTransactorSession) SetFeeCollector(batchPoster common.Address, newFeeCollector common.Address) (*types.Transaction, error) {
	return _ArbAggregator.Contract.SetFeeCollector(&_ArbAggregator.TransactOpts, batchPoster, newFeeCollector)
}

// SetTxBaseFee is a paid mutator transaction binding the contract method 0x5be6888b.
//
// Solidity: function setTxBaseFee(address aggregator, uint256 feeInL1Gas) returns()
func (_ArbAggregator *ArbAggregatorTransactor) SetTxBaseFee(opts *bind.TransactOpts, aggregator common.Address, feeInL1Gas *big.Int) (*types.Transaction, error) {
	return _ArbAggregator.contract.Transact(opts, "setTxBaseFee", aggregator, feeInL1Gas)
}

// SetTxBaseFee is a paid mutator transaction binding the contract method 0x5be6888b.
//
// Solidity: function setTxBaseFee(address aggregator, uint256 feeInL1Gas) returns()
func (_ArbAggregator *ArbAggregatorSession) SetTxBaseFee(aggregator common.Address, feeInL1Gas *big.Int) (*types.Transaction, error) {
	return _ArbAggregator.Contract.SetTxBaseFee(&_ArbAggregator.TransactOpts, aggregator, feeInL1Gas)
}

// SetTxBaseFee is a paid mutator transaction binding the contract method 0x5be6888b.
//
// Solidity: function setTxBaseFee(address aggregator, uint256 feeInL1Gas) returns()
func (_ArbAggregator *ArbAggregatorTransactorSession) SetTxBaseFee(aggregator common.Address, feeInL1Gas *big.Int) (*types.Transaction, error) {
	return _ArbAggregator.Contract.SetTxBaseFee(&_ArbAggregator.TransactOpts, aggregator, feeInL1Gas)
}

// ArbBLSMetaData contains all meta data concerning the ArbBLS contract.
var ArbBLSMetaData = &bind.MetaData{
	ABI: "[]",
}

// ArbBLSABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbBLSMetaData.ABI instead.
var ArbBLSABI = ArbBLSMetaData.ABI

// ArbBLS is an auto generated Go binding around an Ethereum contract.
type ArbBLS struct {
	ArbBLSCaller     // Read-only binding to the contract
	ArbBLSTransactor // Write-only binding to the contract
	ArbBLSFilterer   // Log filterer for contract events
}

// ArbBLSCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbBLSCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbBLSTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbBLSTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbBLSFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbBLSFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbBLSSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbBLSSession struct {
	Contract     *ArbBLS           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbBLSCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbBLSCallerSession struct {
	Contract *ArbBLSCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// ArbBLSTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbBLSTransactorSession struct {
	Contract     *ArbBLSTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbBLSRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbBLSRaw struct {
	Contract *ArbBLS // Generic contract binding to access the raw methods on
}

// ArbBLSCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbBLSCallerRaw struct {
	Contract *ArbBLSCaller // Generic read-only contract binding to access the raw methods on
}

// ArbBLSTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbBLSTransactorRaw struct {
	Contract *ArbBLSTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbBLS creates a new instance of ArbBLS, bound to a specific deployed contract.
func NewArbBLS(address common.Address, backend bind.ContractBackend) (*ArbBLS, error) {
	contract, err := bindArbBLS(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbBLS{ArbBLSCaller: ArbBLSCaller{contract: contract}, ArbBLSTransactor: ArbBLSTransactor{contract: contract}, ArbBLSFilterer: ArbBLSFilterer{contract: contract}}, nil
}

// NewArbBLSCaller creates a new read-only instance of ArbBLS, bound to a specific deployed contract.
func NewArbBLSCaller(address common.Address, caller bind.ContractCaller) (*ArbBLSCaller, error) {
	contract, err := bindArbBLS(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbBLSCaller{contract: contract}, nil
}

// NewArbBLSTransactor creates a new write-only instance of ArbBLS, bound to a specific deployed contract.
func NewArbBLSTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbBLSTransactor, error) {
	contract, err := bindArbBLS(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbBLSTransactor{contract: contract}, nil
}

// NewArbBLSFilterer creates a new log filterer instance of ArbBLS, bound to a specific deployed contract.
func NewArbBLSFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbBLSFilterer, error) {
	contract, err := bindArbBLS(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbBLSFilterer{contract: contract}, nil
}

// bindArbBLS binds a generic wrapper to an already deployed contract.
func bindArbBLS(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbBLSMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbBLS *ArbBLSRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbBLS.Contract.ArbBLSCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbBLS *ArbBLSRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbBLS.Contract.ArbBLSTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbBLS *ArbBLSRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbBLS.Contract.ArbBLSTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbBLS *ArbBLSCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbBLS.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbBLS *ArbBLSTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbBLS.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbBLS *ArbBLSTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbBLS.Contract.contract.Transact(opts, method, params...)
}

// ArbDebugMetaData contains all meta data concerning the ArbDebug contract.
var ArbDebugMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"},{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"},{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"name\":\"Custom\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Unused\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"flag\",\"type\":\"bool\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"}],\"name\":\"Basic\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"flag\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"not\",\"type\":\"bool\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"conn\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"}],\"name\":\"Mixed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"flag\",\"type\":\"bool\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"field\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint24\",\"name\":\"number\",\"type\":\"uint24\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"store\",\"type\":\"bytes\"}],\"name\":\"Store\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"becomeChainOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"number\",\"type\":\"uint64\"}],\"name\":\"customRevert\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"flag\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"}],\"name\":\"events\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
}

// ArbDebugABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbDebugMetaData.ABI instead.
var ArbDebugABI = ArbDebugMetaData.ABI

// ArbDebug is an auto generated Go binding around an Ethereum contract.
type ArbDebug struct {
	ArbDebugCaller     // Read-only binding to the contract
	ArbDebugTransactor // Write-only binding to the contract
	ArbDebugFilterer   // Log filterer for contract events
}

// ArbDebugCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbDebugCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbDebugTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbDebugTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbDebugFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbDebugFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbDebugSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbDebugSession struct {
	Contract     *ArbDebug         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbDebugCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbDebugCallerSession struct {
	Contract *ArbDebugCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ArbDebugTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbDebugTransactorSession struct {
	Contract     *ArbDebugTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ArbDebugRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbDebugRaw struct {
	Contract *ArbDebug // Generic contract binding to access the raw methods on
}

// ArbDebugCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbDebugCallerRaw struct {
	Contract *ArbDebugCaller // Generic read-only contract binding to access the raw methods on
}

// ArbDebugTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbDebugTransactorRaw struct {
	Contract *ArbDebugTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbDebug creates a new instance of ArbDebug, bound to a specific deployed contract.
func NewArbDebug(address common.Address, backend bind.ContractBackend) (*ArbDebug, error) {
	contract, err := bindArbDebug(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbDebug{ArbDebugCaller: ArbDebugCaller{contract: contract}, ArbDebugTransactor: ArbDebugTransactor{contract: contract}, ArbDebugFilterer: ArbDebugFilterer{contract: contract}}, nil
}

// NewArbDebugCaller creates a new read-only instance of ArbDebug, bound to a specific deployed contract.
func NewArbDebugCaller(address common.Address, caller bind.ContractCaller) (*ArbDebugCaller, error) {
	contract, err := bindArbDebug(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbDebugCaller{contract: contract}, nil
}

// NewArbDebugTransactor creates a new write-only instance of ArbDebug, bound to a specific deployed contract.
func NewArbDebugTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbDebugTransactor, error) {
	contract, err := bindArbDebug(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbDebugTransactor{contract: contract}, nil
}

// NewArbDebugFilterer creates a new log filterer instance of ArbDebug, bound to a specific deployed contract.
func NewArbDebugFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbDebugFilterer, error) {
	contract, err := bindArbDebug(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbDebugFilterer{contract: contract}, nil
}

// bindArbDebug binds a generic wrapper to an already deployed contract.
func bindArbDebug(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbDebugMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbDebug *ArbDebugRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbDebug.Contract.ArbDebugCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbDebug *ArbDebugRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbDebug.Contract.ArbDebugTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbDebug *ArbDebugRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbDebug.Contract.ArbDebugTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbDebug *ArbDebugCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbDebug.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbDebug *ArbDebugTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbDebug.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbDebug *ArbDebugTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbDebug.Contract.contract.Transact(opts, method, params...)
}

// CustomRevert is a free data retrieval call binding the contract method 0x7ea89f8b.
//
// Solidity: function customRevert(uint64 number) pure returns()
func (_ArbDebug *ArbDebugCaller) CustomRevert(opts *bind.CallOpts, number uint64) error {
	var out []interface{}
	err := _ArbDebug.contract.Call(opts, &out, "customRevert", number)

	if err != nil {
		return err
	}

	return err

}

// CustomRevert is a free data retrieval call binding the contract method 0x7ea89f8b.
//
// Solidity: function customRevert(uint64 number) pure returns()
func (_ArbDebug *ArbDebugSession) CustomRevert(number uint64) error {
	return _ArbDebug.Contract.CustomRevert(&_ArbDebug.CallOpts, number)
}

// CustomRevert is a free data retrieval call binding the contract method 0x7ea89f8b.
//
// Solidity: function customRevert(uint64 number) pure returns()
func (_ArbDebug *ArbDebugCallerSession) CustomRevert(number uint64) error {
	return _ArbDebug.Contract.CustomRevert(&_ArbDebug.CallOpts, number)
}

// BecomeChainOwner is a paid mutator transaction binding the contract method 0x0e5bbc11.
//
// Solidity: function becomeChainOwner() returns()
func (_ArbDebug *ArbDebugTransactor) BecomeChainOwner(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbDebug.contract.Transact(opts, "becomeChainOwner")
}

// BecomeChainOwner is a paid mutator transaction binding the contract method 0x0e5bbc11.
//
// Solidity: function becomeChainOwner() returns()
func (_ArbDebug *ArbDebugSession) BecomeChainOwner() (*types.Transaction, error) {
	return _ArbDebug.Contract.BecomeChainOwner(&_ArbDebug.TransactOpts)
}

// BecomeChainOwner is a paid mutator transaction binding the contract method 0x0e5bbc11.
//
// Solidity: function becomeChainOwner() returns()
func (_ArbDebug *ArbDebugTransactorSession) BecomeChainOwner() (*types.Transaction, error) {
	return _ArbDebug.Contract.BecomeChainOwner(&_ArbDebug.TransactOpts)
}

// Events is a paid mutator transaction binding the contract method 0x7b9963ef.
//
// Solidity: function events(bool flag, bytes32 value) payable returns(address, uint256)
func (_ArbDebug *ArbDebugTransactor) Events(opts *bind.TransactOpts, flag bool, value [32]byte) (*types.Transaction, error) {
	return _ArbDebug.contract.Transact(opts, "events", flag, value)
}

// Events is a paid mutator transaction binding the contract method 0x7b9963ef.
//
// Solidity: function events(bool flag, bytes32 value) payable returns(address, uint256)
func (_ArbDebug *ArbDebugSession) Events(flag bool, value [32]byte) (*types.Transaction, error) {
	return _ArbDebug.Contract.Events(&_ArbDebug.TransactOpts, flag, value)
}

// Events is a paid mutator transaction binding the contract method 0x7b9963ef.
//
// Solidity: function events(bool flag, bytes32 value) payable returns(address, uint256)
func (_ArbDebug *ArbDebugTransactorSession) Events(flag bool, value [32]byte) (*types.Transaction, error) {
	return _ArbDebug.Contract.Events(&_ArbDebug.TransactOpts, flag, value)
}

// ArbDebugBasicIterator is returned from FilterBasic and is used to iterate over the raw logs and unpacked data for Basic events raised by the ArbDebug contract.
type ArbDebugBasicIterator struct {
	Event *ArbDebugBasic // Event containing the contract specifics and raw log

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
func (it *ArbDebugBasicIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbDebugBasic)
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
		it.Event = new(ArbDebugBasic)
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
func (it *ArbDebugBasicIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbDebugBasicIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbDebugBasic represents a Basic event raised by the ArbDebug contract.
type ArbDebugBasic struct {
	Flag  bool
	Value [32]byte
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterBasic is a free log retrieval operation binding the contract event 0x93c1309077578d0b9ed1398956be51f0c6e1fcc0b91d899836148550215acfe2.
//
// Solidity: event Basic(bool flag, bytes32 indexed value)
func (_ArbDebug *ArbDebugFilterer) FilterBasic(opts *bind.FilterOpts, value [][32]byte) (*ArbDebugBasicIterator, error) {

	var valueRule []interface{}
	for _, valueItem := range value {
		valueRule = append(valueRule, valueItem)
	}

	logs, sub, err := _ArbDebug.contract.FilterLogs(opts, "Basic", valueRule)
	if err != nil {
		return nil, err
	}
	return &ArbDebugBasicIterator{contract: _ArbDebug.contract, event: "Basic", logs: logs, sub: sub}, nil
}

// WatchBasic is a free log subscription operation binding the contract event 0x93c1309077578d0b9ed1398956be51f0c6e1fcc0b91d899836148550215acfe2.
//
// Solidity: event Basic(bool flag, bytes32 indexed value)
func (_ArbDebug *ArbDebugFilterer) WatchBasic(opts *bind.WatchOpts, sink chan<- *ArbDebugBasic, value [][32]byte) (event.Subscription, error) {

	var valueRule []interface{}
	for _, valueItem := range value {
		valueRule = append(valueRule, valueItem)
	}

	logs, sub, err := _ArbDebug.contract.WatchLogs(opts, "Basic", valueRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbDebugBasic)
				if err := _ArbDebug.contract.UnpackLog(event, "Basic", log); err != nil {
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

// ParseBasic is a log parse operation binding the contract event 0x93c1309077578d0b9ed1398956be51f0c6e1fcc0b91d899836148550215acfe2.
//
// Solidity: event Basic(bool flag, bytes32 indexed value)
func (_ArbDebug *ArbDebugFilterer) ParseBasic(log types.Log) (*ArbDebugBasic, error) {
	event := new(ArbDebugBasic)
	if err := _ArbDebug.contract.UnpackLog(event, "Basic", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbDebugMixedIterator is returned from FilterMixed and is used to iterate over the raw logs and unpacked data for Mixed events raised by the ArbDebug contract.
type ArbDebugMixedIterator struct {
	Event *ArbDebugMixed // Event containing the contract specifics and raw log

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
func (it *ArbDebugMixedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbDebugMixed)
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
		it.Event = new(ArbDebugMixed)
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
func (it *ArbDebugMixedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbDebugMixedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbDebugMixed represents a Mixed event raised by the ArbDebug contract.
type ArbDebugMixed struct {
	Flag   bool
	Not    bool
	Value  [32]byte
	Conn   common.Address
	Caller common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterMixed is a free log retrieval operation binding the contract event 0xa6059246508753631072a6a59c1127af99d3f4cc0f8d6370d4fae122b1dd4eaf.
//
// Solidity: event Mixed(bool indexed flag, bool not, bytes32 indexed value, address conn, address indexed caller)
func (_ArbDebug *ArbDebugFilterer) FilterMixed(opts *bind.FilterOpts, flag []bool, value [][32]byte, caller []common.Address) (*ArbDebugMixedIterator, error) {

	var flagRule []interface{}
	for _, flagItem := range flag {
		flagRule = append(flagRule, flagItem)
	}

	var valueRule []interface{}
	for _, valueItem := range value {
		valueRule = append(valueRule, valueItem)
	}

	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}

	logs, sub, err := _ArbDebug.contract.FilterLogs(opts, "Mixed", flagRule, valueRule, callerRule)
	if err != nil {
		return nil, err
	}
	return &ArbDebugMixedIterator{contract: _ArbDebug.contract, event: "Mixed", logs: logs, sub: sub}, nil
}

// WatchMixed is a free log subscription operation binding the contract event 0xa6059246508753631072a6a59c1127af99d3f4cc0f8d6370d4fae122b1dd4eaf.
//
// Solidity: event Mixed(bool indexed flag, bool not, bytes32 indexed value, address conn, address indexed caller)
func (_ArbDebug *ArbDebugFilterer) WatchMixed(opts *bind.WatchOpts, sink chan<- *ArbDebugMixed, flag []bool, value [][32]byte, caller []common.Address) (event.Subscription, error) {

	var flagRule []interface{}
	for _, flagItem := range flag {
		flagRule = append(flagRule, flagItem)
	}

	var valueRule []interface{}
	for _, valueItem := range value {
		valueRule = append(valueRule, valueItem)
	}

	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}

	logs, sub, err := _ArbDebug.contract.WatchLogs(opts, "Mixed", flagRule, valueRule, callerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbDebugMixed)
				if err := _ArbDebug.contract.UnpackLog(event, "Mixed", log); err != nil {
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

// ParseMixed is a log parse operation binding the contract event 0xa6059246508753631072a6a59c1127af99d3f4cc0f8d6370d4fae122b1dd4eaf.
//
// Solidity: event Mixed(bool indexed flag, bool not, bytes32 indexed value, address conn, address indexed caller)
func (_ArbDebug *ArbDebugFilterer) ParseMixed(log types.Log) (*ArbDebugMixed, error) {
	event := new(ArbDebugMixed)
	if err := _ArbDebug.contract.UnpackLog(event, "Mixed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbDebugStoreIterator is returned from FilterStore and is used to iterate over the raw logs and unpacked data for Store events raised by the ArbDebug contract.
type ArbDebugStoreIterator struct {
	Event *ArbDebugStore // Event containing the contract specifics and raw log

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
func (it *ArbDebugStoreIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbDebugStore)
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
		it.Event = new(ArbDebugStore)
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
func (it *ArbDebugStoreIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbDebugStoreIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbDebugStore represents a Store event raised by the ArbDebug contract.
type ArbDebugStore struct {
	Flag   bool
	Field  common.Address
	Number *big.Int
	Value  [32]byte
	Store  []byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStore is a free log retrieval operation binding the contract event 0x9be442d880b83e5d6db765f303a0602546662e34a066734a19a7ee929c028d95.
//
// Solidity: event Store(bool indexed flag, address indexed field, uint24 number, bytes32 value, bytes store)
func (_ArbDebug *ArbDebugFilterer) FilterStore(opts *bind.FilterOpts, flag []bool, field []common.Address) (*ArbDebugStoreIterator, error) {

	var flagRule []interface{}
	for _, flagItem := range flag {
		flagRule = append(flagRule, flagItem)
	}
	var fieldRule []interface{}
	for _, fieldItem := range field {
		fieldRule = append(fieldRule, fieldItem)
	}

	logs, sub, err := _ArbDebug.contract.FilterLogs(opts, "Store", flagRule, fieldRule)
	if err != nil {
		return nil, err
	}
	return &ArbDebugStoreIterator{contract: _ArbDebug.contract, event: "Store", logs: logs, sub: sub}, nil
}

// WatchStore is a free log subscription operation binding the contract event 0x9be442d880b83e5d6db765f303a0602546662e34a066734a19a7ee929c028d95.
//
// Solidity: event Store(bool indexed flag, address indexed field, uint24 number, bytes32 value, bytes store)
func (_ArbDebug *ArbDebugFilterer) WatchStore(opts *bind.WatchOpts, sink chan<- *ArbDebugStore, flag []bool, field []common.Address) (event.Subscription, error) {

	var flagRule []interface{}
	for _, flagItem := range flag {
		flagRule = append(flagRule, flagItem)
	}
	var fieldRule []interface{}
	for _, fieldItem := range field {
		fieldRule = append(fieldRule, fieldItem)
	}

	logs, sub, err := _ArbDebug.contract.WatchLogs(opts, "Store", flagRule, fieldRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbDebugStore)
				if err := _ArbDebug.contract.UnpackLog(event, "Store", log); err != nil {
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

// ParseStore is a log parse operation binding the contract event 0x9be442d880b83e5d6db765f303a0602546662e34a066734a19a7ee929c028d95.
//
// Solidity: event Store(bool indexed flag, address indexed field, uint24 number, bytes32 value, bytes store)
func (_ArbDebug *ArbDebugFilterer) ParseStore(log types.Log) (*ArbDebugStore, error) {
	event := new(ArbDebugStore)
	if err := _ArbDebug.contract.UnpackLog(event, "Store", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbFunctionTableMetaData contains all meta data concerning the ArbFunctionTable contract.
var ArbFunctionTableMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"get\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"size\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"buf\",\"type\":\"bytes\"}],\"name\":\"upload\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ArbFunctionTableABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbFunctionTableMetaData.ABI instead.
var ArbFunctionTableABI = ArbFunctionTableMetaData.ABI

// ArbFunctionTable is an auto generated Go binding around an Ethereum contract.
type ArbFunctionTable struct {
	ArbFunctionTableCaller     // Read-only binding to the contract
	ArbFunctionTableTransactor // Write-only binding to the contract
	ArbFunctionTableFilterer   // Log filterer for contract events
}

// ArbFunctionTableCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbFunctionTableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbFunctionTableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbFunctionTableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbFunctionTableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbFunctionTableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbFunctionTableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbFunctionTableSession struct {
	Contract     *ArbFunctionTable // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbFunctionTableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbFunctionTableCallerSession struct {
	Contract *ArbFunctionTableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// ArbFunctionTableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbFunctionTableTransactorSession struct {
	Contract     *ArbFunctionTableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// ArbFunctionTableRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbFunctionTableRaw struct {
	Contract *ArbFunctionTable // Generic contract binding to access the raw methods on
}

// ArbFunctionTableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbFunctionTableCallerRaw struct {
	Contract *ArbFunctionTableCaller // Generic read-only contract binding to access the raw methods on
}

// ArbFunctionTableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbFunctionTableTransactorRaw struct {
	Contract *ArbFunctionTableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbFunctionTable creates a new instance of ArbFunctionTable, bound to a specific deployed contract.
func NewArbFunctionTable(address common.Address, backend bind.ContractBackend) (*ArbFunctionTable, error) {
	contract, err := bindArbFunctionTable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbFunctionTable{ArbFunctionTableCaller: ArbFunctionTableCaller{contract: contract}, ArbFunctionTableTransactor: ArbFunctionTableTransactor{contract: contract}, ArbFunctionTableFilterer: ArbFunctionTableFilterer{contract: contract}}, nil
}

// NewArbFunctionTableCaller creates a new read-only instance of ArbFunctionTable, bound to a specific deployed contract.
func NewArbFunctionTableCaller(address common.Address, caller bind.ContractCaller) (*ArbFunctionTableCaller, error) {
	contract, err := bindArbFunctionTable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbFunctionTableCaller{contract: contract}, nil
}

// NewArbFunctionTableTransactor creates a new write-only instance of ArbFunctionTable, bound to a specific deployed contract.
func NewArbFunctionTableTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbFunctionTableTransactor, error) {
	contract, err := bindArbFunctionTable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbFunctionTableTransactor{contract: contract}, nil
}

// NewArbFunctionTableFilterer creates a new log filterer instance of ArbFunctionTable, bound to a specific deployed contract.
func NewArbFunctionTableFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbFunctionTableFilterer, error) {
	contract, err := bindArbFunctionTable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbFunctionTableFilterer{contract: contract}, nil
}

// bindArbFunctionTable binds a generic wrapper to an already deployed contract.
func bindArbFunctionTable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbFunctionTableMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbFunctionTable *ArbFunctionTableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbFunctionTable.Contract.ArbFunctionTableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbFunctionTable *ArbFunctionTableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbFunctionTable.Contract.ArbFunctionTableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbFunctionTable *ArbFunctionTableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbFunctionTable.Contract.ArbFunctionTableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbFunctionTable *ArbFunctionTableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbFunctionTable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbFunctionTable *ArbFunctionTableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbFunctionTable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbFunctionTable *ArbFunctionTableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbFunctionTable.Contract.contract.Transact(opts, method, params...)
}

// Get is a free data retrieval call binding the contract method 0xb464631b.
//
// Solidity: function get(address addr, uint256 index) view returns(uint256, bool, uint256)
func (_ArbFunctionTable *ArbFunctionTableCaller) Get(opts *bind.CallOpts, addr common.Address, index *big.Int) (*big.Int, bool, *big.Int, error) {
	var out []interface{}
	err := _ArbFunctionTable.contract.Call(opts, &out, "get", addr, index)

	if err != nil {
		return *new(*big.Int), *new(bool), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(bool)).(*bool)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return out0, out1, out2, err

}

// Get is a free data retrieval call binding the contract method 0xb464631b.
//
// Solidity: function get(address addr, uint256 index) view returns(uint256, bool, uint256)
func (_ArbFunctionTable *ArbFunctionTableSession) Get(addr common.Address, index *big.Int) (*big.Int, bool, *big.Int, error) {
	return _ArbFunctionTable.Contract.Get(&_ArbFunctionTable.CallOpts, addr, index)
}

// Get is a free data retrieval call binding the contract method 0xb464631b.
//
// Solidity: function get(address addr, uint256 index) view returns(uint256, bool, uint256)
func (_ArbFunctionTable *ArbFunctionTableCallerSession) Get(addr common.Address, index *big.Int) (*big.Int, bool, *big.Int, error) {
	return _ArbFunctionTable.Contract.Get(&_ArbFunctionTable.CallOpts, addr, index)
}

// Size is a free data retrieval call binding the contract method 0x88987068.
//
// Solidity: function size(address addr) view returns(uint256)
func (_ArbFunctionTable *ArbFunctionTableCaller) Size(opts *bind.CallOpts, addr common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ArbFunctionTable.contract.Call(opts, &out, "size", addr)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Size is a free data retrieval call binding the contract method 0x88987068.
//
// Solidity: function size(address addr) view returns(uint256)
func (_ArbFunctionTable *ArbFunctionTableSession) Size(addr common.Address) (*big.Int, error) {
	return _ArbFunctionTable.Contract.Size(&_ArbFunctionTable.CallOpts, addr)
}

// Size is a free data retrieval call binding the contract method 0x88987068.
//
// Solidity: function size(address addr) view returns(uint256)
func (_ArbFunctionTable *ArbFunctionTableCallerSession) Size(addr common.Address) (*big.Int, error) {
	return _ArbFunctionTable.Contract.Size(&_ArbFunctionTable.CallOpts, addr)
}

// Upload is a paid mutator transaction binding the contract method 0xce2ae159.
//
// Solidity: function upload(bytes buf) returns()
func (_ArbFunctionTable *ArbFunctionTableTransactor) Upload(opts *bind.TransactOpts, buf []byte) (*types.Transaction, error) {
	return _ArbFunctionTable.contract.Transact(opts, "upload", buf)
}

// Upload is a paid mutator transaction binding the contract method 0xce2ae159.
//
// Solidity: function upload(bytes buf) returns()
func (_ArbFunctionTable *ArbFunctionTableSession) Upload(buf []byte) (*types.Transaction, error) {
	return _ArbFunctionTable.Contract.Upload(&_ArbFunctionTable.TransactOpts, buf)
}

// Upload is a paid mutator transaction binding the contract method 0xce2ae159.
//
// Solidity: function upload(bytes buf) returns()
func (_ArbFunctionTable *ArbFunctionTableTransactorSession) Upload(buf []byte) (*types.Transaction, error) {
	return _ArbFunctionTable.Contract.Upload(&_ArbFunctionTable.TransactOpts, buf)
}

// ArbGasInfoMetaData contains all meta data concerning the ArbGasInfo contract.
var ArbGasInfoMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"getAmortizedCostCapBips\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentTxL1GasFees\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasAccountingParams\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasBacklog\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasBacklogTolerance\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getL1BaseFeeEstimate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getL1BaseFeeEstimateInertia\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getL1FeesAvailable\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getL1GasPriceEstimate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getL1PricingSurplus\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getL1RewardRate\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getL1RewardRecipient\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getMinimumGasPrice\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPerBatchGasCharge\",\"outputs\":[{\"internalType\":\"int64\",\"name\":\"\",\"type\":\"int64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPricesInArbGas\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"aggregator\",\"type\":\"address\"}],\"name\":\"getPricesInArbGasWithAggregator\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPricesInWei\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"aggregator\",\"type\":\"address\"}],\"name\":\"getPricesInWeiWithAggregator\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPricingInertia\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// ArbGasInfoABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbGasInfoMetaData.ABI instead.
var ArbGasInfoABI = ArbGasInfoMetaData.ABI

// ArbGasInfo is an auto generated Go binding around an Ethereum contract.
type ArbGasInfo struct {
	ArbGasInfoCaller     // Read-only binding to the contract
	ArbGasInfoTransactor // Write-only binding to the contract
	ArbGasInfoFilterer   // Log filterer for contract events
}

// ArbGasInfoCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbGasInfoCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbGasInfoTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbGasInfoTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbGasInfoFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbGasInfoFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbGasInfoSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbGasInfoSession struct {
	Contract     *ArbGasInfo       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbGasInfoCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbGasInfoCallerSession struct {
	Contract *ArbGasInfoCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// ArbGasInfoTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbGasInfoTransactorSession struct {
	Contract     *ArbGasInfoTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ArbGasInfoRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbGasInfoRaw struct {
	Contract *ArbGasInfo // Generic contract binding to access the raw methods on
}

// ArbGasInfoCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbGasInfoCallerRaw struct {
	Contract *ArbGasInfoCaller // Generic read-only contract binding to access the raw methods on
}

// ArbGasInfoTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbGasInfoTransactorRaw struct {
	Contract *ArbGasInfoTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbGasInfo creates a new instance of ArbGasInfo, bound to a specific deployed contract.
func NewArbGasInfo(address common.Address, backend bind.ContractBackend) (*ArbGasInfo, error) {
	contract, err := bindArbGasInfo(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbGasInfo{ArbGasInfoCaller: ArbGasInfoCaller{contract: contract}, ArbGasInfoTransactor: ArbGasInfoTransactor{contract: contract}, ArbGasInfoFilterer: ArbGasInfoFilterer{contract: contract}}, nil
}

// NewArbGasInfoCaller creates a new read-only instance of ArbGasInfo, bound to a specific deployed contract.
func NewArbGasInfoCaller(address common.Address, caller bind.ContractCaller) (*ArbGasInfoCaller, error) {
	contract, err := bindArbGasInfo(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbGasInfoCaller{contract: contract}, nil
}

// NewArbGasInfoTransactor creates a new write-only instance of ArbGasInfo, bound to a specific deployed contract.
func NewArbGasInfoTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbGasInfoTransactor, error) {
	contract, err := bindArbGasInfo(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbGasInfoTransactor{contract: contract}, nil
}

// NewArbGasInfoFilterer creates a new log filterer instance of ArbGasInfo, bound to a specific deployed contract.
func NewArbGasInfoFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbGasInfoFilterer, error) {
	contract, err := bindArbGasInfo(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbGasInfoFilterer{contract: contract}, nil
}

// bindArbGasInfo binds a generic wrapper to an already deployed contract.
func bindArbGasInfo(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbGasInfoMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbGasInfo *ArbGasInfoRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbGasInfo.Contract.ArbGasInfoCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbGasInfo *ArbGasInfoRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbGasInfo.Contract.ArbGasInfoTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbGasInfo *ArbGasInfoRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbGasInfo.Contract.ArbGasInfoTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbGasInfo *ArbGasInfoCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbGasInfo.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbGasInfo *ArbGasInfoTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbGasInfo.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbGasInfo *ArbGasInfoTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbGasInfo.Contract.contract.Transact(opts, method, params...)
}

// GetAmortizedCostCapBips is a free data retrieval call binding the contract method 0x7a7d6beb.
//
// Solidity: function getAmortizedCostCapBips() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCaller) GetAmortizedCostCapBips(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getAmortizedCostCapBips")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetAmortizedCostCapBips is a free data retrieval call binding the contract method 0x7a7d6beb.
//
// Solidity: function getAmortizedCostCapBips() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoSession) GetAmortizedCostCapBips() (uint64, error) {
	return _ArbGasInfo.Contract.GetAmortizedCostCapBips(&_ArbGasInfo.CallOpts)
}

// GetAmortizedCostCapBips is a free data retrieval call binding the contract method 0x7a7d6beb.
//
// Solidity: function getAmortizedCostCapBips() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetAmortizedCostCapBips() (uint64, error) {
	return _ArbGasInfo.Contract.GetAmortizedCostCapBips(&_ArbGasInfo.CallOpts)
}

// GetCurrentTxL1GasFees is a free data retrieval call binding the contract method 0xc6f7de0e.
//
// Solidity: function getCurrentTxL1GasFees() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoCaller) GetCurrentTxL1GasFees(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getCurrentTxL1GasFees")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentTxL1GasFees is a free data retrieval call binding the contract method 0xc6f7de0e.
//
// Solidity: function getCurrentTxL1GasFees() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoSession) GetCurrentTxL1GasFees() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetCurrentTxL1GasFees(&_ArbGasInfo.CallOpts)
}

// GetCurrentTxL1GasFees is a free data retrieval call binding the contract method 0xc6f7de0e.
//
// Solidity: function getCurrentTxL1GasFees() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetCurrentTxL1GasFees() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetCurrentTxL1GasFees(&_ArbGasInfo.CallOpts)
}

// GetGasAccountingParams is a free data retrieval call binding the contract method 0x612af178.
//
// Solidity: function getGasAccountingParams() view returns(uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoCaller) GetGasAccountingParams(opts *bind.CallOpts) (*big.Int, *big.Int, *big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getGasAccountingParams")

	if err != nil {
		return *new(*big.Int), *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return out0, out1, out2, err

}

// GetGasAccountingParams is a free data retrieval call binding the contract method 0x612af178.
//
// Solidity: function getGasAccountingParams() view returns(uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoSession) GetGasAccountingParams() (*big.Int, *big.Int, *big.Int, error) {
	return _ArbGasInfo.Contract.GetGasAccountingParams(&_ArbGasInfo.CallOpts)
}

// GetGasAccountingParams is a free data retrieval call binding the contract method 0x612af178.
//
// Solidity: function getGasAccountingParams() view returns(uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetGasAccountingParams() (*big.Int, *big.Int, *big.Int, error) {
	return _ArbGasInfo.Contract.GetGasAccountingParams(&_ArbGasInfo.CallOpts)
}

// GetGasBacklog is a free data retrieval call binding the contract method 0x1d5b5c20.
//
// Solidity: function getGasBacklog() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCaller) GetGasBacklog(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getGasBacklog")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetGasBacklog is a free data retrieval call binding the contract method 0x1d5b5c20.
//
// Solidity: function getGasBacklog() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoSession) GetGasBacklog() (uint64, error) {
	return _ArbGasInfo.Contract.GetGasBacklog(&_ArbGasInfo.CallOpts)
}

// GetGasBacklog is a free data retrieval call binding the contract method 0x1d5b5c20.
//
// Solidity: function getGasBacklog() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetGasBacklog() (uint64, error) {
	return _ArbGasInfo.Contract.GetGasBacklog(&_ArbGasInfo.CallOpts)
}

// GetGasBacklogTolerance is a free data retrieval call binding the contract method 0x25754f91.
//
// Solidity: function getGasBacklogTolerance() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCaller) GetGasBacklogTolerance(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getGasBacklogTolerance")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetGasBacklogTolerance is a free data retrieval call binding the contract method 0x25754f91.
//
// Solidity: function getGasBacklogTolerance() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoSession) GetGasBacklogTolerance() (uint64, error) {
	return _ArbGasInfo.Contract.GetGasBacklogTolerance(&_ArbGasInfo.CallOpts)
}

// GetGasBacklogTolerance is a free data retrieval call binding the contract method 0x25754f91.
//
// Solidity: function getGasBacklogTolerance() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetGasBacklogTolerance() (uint64, error) {
	return _ArbGasInfo.Contract.GetGasBacklogTolerance(&_ArbGasInfo.CallOpts)
}

// GetL1BaseFeeEstimate is a free data retrieval call binding the contract method 0xf5d6ded7.
//
// Solidity: function getL1BaseFeeEstimate() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoCaller) GetL1BaseFeeEstimate(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getL1BaseFeeEstimate")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1BaseFeeEstimate is a free data retrieval call binding the contract method 0xf5d6ded7.
//
// Solidity: function getL1BaseFeeEstimate() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoSession) GetL1BaseFeeEstimate() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetL1BaseFeeEstimate(&_ArbGasInfo.CallOpts)
}

// GetL1BaseFeeEstimate is a free data retrieval call binding the contract method 0xf5d6ded7.
//
// Solidity: function getL1BaseFeeEstimate() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetL1BaseFeeEstimate() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetL1BaseFeeEstimate(&_ArbGasInfo.CallOpts)
}

// GetL1BaseFeeEstimateInertia is a free data retrieval call binding the contract method 0x29eb31ee.
//
// Solidity: function getL1BaseFeeEstimateInertia() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCaller) GetL1BaseFeeEstimateInertia(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getL1BaseFeeEstimateInertia")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetL1BaseFeeEstimateInertia is a free data retrieval call binding the contract method 0x29eb31ee.
//
// Solidity: function getL1BaseFeeEstimateInertia() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoSession) GetL1BaseFeeEstimateInertia() (uint64, error) {
	return _ArbGasInfo.Contract.GetL1BaseFeeEstimateInertia(&_ArbGasInfo.CallOpts)
}

// GetL1BaseFeeEstimateInertia is a free data retrieval call binding the contract method 0x29eb31ee.
//
// Solidity: function getL1BaseFeeEstimateInertia() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetL1BaseFeeEstimateInertia() (uint64, error) {
	return _ArbGasInfo.Contract.GetL1BaseFeeEstimateInertia(&_ArbGasInfo.CallOpts)
}

// GetL1FeesAvailable is a free data retrieval call binding the contract method 0x5b39d23c.
//
// Solidity: function getL1FeesAvailable() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoCaller) GetL1FeesAvailable(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getL1FeesAvailable")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1FeesAvailable is a free data retrieval call binding the contract method 0x5b39d23c.
//
// Solidity: function getL1FeesAvailable() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoSession) GetL1FeesAvailable() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetL1FeesAvailable(&_ArbGasInfo.CallOpts)
}

// GetL1FeesAvailable is a free data retrieval call binding the contract method 0x5b39d23c.
//
// Solidity: function getL1FeesAvailable() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetL1FeesAvailable() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetL1FeesAvailable(&_ArbGasInfo.CallOpts)
}

// GetL1GasPriceEstimate is a free data retrieval call binding the contract method 0x055f362f.
//
// Solidity: function getL1GasPriceEstimate() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoCaller) GetL1GasPriceEstimate(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getL1GasPriceEstimate")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1GasPriceEstimate is a free data retrieval call binding the contract method 0x055f362f.
//
// Solidity: function getL1GasPriceEstimate() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoSession) GetL1GasPriceEstimate() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetL1GasPriceEstimate(&_ArbGasInfo.CallOpts)
}

// GetL1GasPriceEstimate is a free data retrieval call binding the contract method 0x055f362f.
//
// Solidity: function getL1GasPriceEstimate() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetL1GasPriceEstimate() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetL1GasPriceEstimate(&_ArbGasInfo.CallOpts)
}

// GetL1PricingSurplus is a free data retrieval call binding the contract method 0x520acdd7.
//
// Solidity: function getL1PricingSurplus() view returns(int256)
func (_ArbGasInfo *ArbGasInfoCaller) GetL1PricingSurplus(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getL1PricingSurplus")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1PricingSurplus is a free data retrieval call binding the contract method 0x520acdd7.
//
// Solidity: function getL1PricingSurplus() view returns(int256)
func (_ArbGasInfo *ArbGasInfoSession) GetL1PricingSurplus() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetL1PricingSurplus(&_ArbGasInfo.CallOpts)
}

// GetL1PricingSurplus is a free data retrieval call binding the contract method 0x520acdd7.
//
// Solidity: function getL1PricingSurplus() view returns(int256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetL1PricingSurplus() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetL1PricingSurplus(&_ArbGasInfo.CallOpts)
}

// GetL1RewardRate is a free data retrieval call binding the contract method 0x8a5b1d28.
//
// Solidity: function getL1RewardRate() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCaller) GetL1RewardRate(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getL1RewardRate")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetL1RewardRate is a free data retrieval call binding the contract method 0x8a5b1d28.
//
// Solidity: function getL1RewardRate() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoSession) GetL1RewardRate() (uint64, error) {
	return _ArbGasInfo.Contract.GetL1RewardRate(&_ArbGasInfo.CallOpts)
}

// GetL1RewardRate is a free data retrieval call binding the contract method 0x8a5b1d28.
//
// Solidity: function getL1RewardRate() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetL1RewardRate() (uint64, error) {
	return _ArbGasInfo.Contract.GetL1RewardRate(&_ArbGasInfo.CallOpts)
}

// GetL1RewardRecipient is a free data retrieval call binding the contract method 0x9e6d7e31.
//
// Solidity: function getL1RewardRecipient() view returns(address)
func (_ArbGasInfo *ArbGasInfoCaller) GetL1RewardRecipient(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getL1RewardRecipient")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetL1RewardRecipient is a free data retrieval call binding the contract method 0x9e6d7e31.
//
// Solidity: function getL1RewardRecipient() view returns(address)
func (_ArbGasInfo *ArbGasInfoSession) GetL1RewardRecipient() (common.Address, error) {
	return _ArbGasInfo.Contract.GetL1RewardRecipient(&_ArbGasInfo.CallOpts)
}

// GetL1RewardRecipient is a free data retrieval call binding the contract method 0x9e6d7e31.
//
// Solidity: function getL1RewardRecipient() view returns(address)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetL1RewardRecipient() (common.Address, error) {
	return _ArbGasInfo.Contract.GetL1RewardRecipient(&_ArbGasInfo.CallOpts)
}

// GetMinimumGasPrice is a free data retrieval call binding the contract method 0xf918379a.
//
// Solidity: function getMinimumGasPrice() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoCaller) GetMinimumGasPrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getMinimumGasPrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetMinimumGasPrice is a free data retrieval call binding the contract method 0xf918379a.
//
// Solidity: function getMinimumGasPrice() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoSession) GetMinimumGasPrice() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetMinimumGasPrice(&_ArbGasInfo.CallOpts)
}

// GetMinimumGasPrice is a free data retrieval call binding the contract method 0xf918379a.
//
// Solidity: function getMinimumGasPrice() view returns(uint256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetMinimumGasPrice() (*big.Int, error) {
	return _ArbGasInfo.Contract.GetMinimumGasPrice(&_ArbGasInfo.CallOpts)
}

// GetPerBatchGasCharge is a free data retrieval call binding the contract method 0x6ecca45a.
//
// Solidity: function getPerBatchGasCharge() view returns(int64)
func (_ArbGasInfo *ArbGasInfoCaller) GetPerBatchGasCharge(opts *bind.CallOpts) (int64, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getPerBatchGasCharge")

	if err != nil {
		return *new(int64), err
	}

	out0 := *abi.ConvertType(out[0], new(int64)).(*int64)

	return out0, err

}

// GetPerBatchGasCharge is a free data retrieval call binding the contract method 0x6ecca45a.
//
// Solidity: function getPerBatchGasCharge() view returns(int64)
func (_ArbGasInfo *ArbGasInfoSession) GetPerBatchGasCharge() (int64, error) {
	return _ArbGasInfo.Contract.GetPerBatchGasCharge(&_ArbGasInfo.CallOpts)
}

// GetPerBatchGasCharge is a free data retrieval call binding the contract method 0x6ecca45a.
//
// Solidity: function getPerBatchGasCharge() view returns(int64)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetPerBatchGasCharge() (int64, error) {
	return _ArbGasInfo.Contract.GetPerBatchGasCharge(&_ArbGasInfo.CallOpts)
}

// GetPricesInArbGas is a free data retrieval call binding the contract method 0x02199f34.
//
// Solidity: function getPricesInArbGas() view returns(uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoCaller) GetPricesInArbGas(opts *bind.CallOpts) (*big.Int, *big.Int, *big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getPricesInArbGas")

	if err != nil {
		return *new(*big.Int), *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return out0, out1, out2, err

}

// GetPricesInArbGas is a free data retrieval call binding the contract method 0x02199f34.
//
// Solidity: function getPricesInArbGas() view returns(uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoSession) GetPricesInArbGas() (*big.Int, *big.Int, *big.Int, error) {
	return _ArbGasInfo.Contract.GetPricesInArbGas(&_ArbGasInfo.CallOpts)
}

// GetPricesInArbGas is a free data retrieval call binding the contract method 0x02199f34.
//
// Solidity: function getPricesInArbGas() view returns(uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetPricesInArbGas() (*big.Int, *big.Int, *big.Int, error) {
	return _ArbGasInfo.Contract.GetPricesInArbGas(&_ArbGasInfo.CallOpts)
}

// GetPricesInArbGasWithAggregator is a free data retrieval call binding the contract method 0x7a1ea732.
//
// Solidity: function getPricesInArbGasWithAggregator(address aggregator) view returns(uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoCaller) GetPricesInArbGasWithAggregator(opts *bind.CallOpts, aggregator common.Address) (*big.Int, *big.Int, *big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getPricesInArbGasWithAggregator", aggregator)

	if err != nil {
		return *new(*big.Int), *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return out0, out1, out2, err

}

// GetPricesInArbGasWithAggregator is a free data retrieval call binding the contract method 0x7a1ea732.
//
// Solidity: function getPricesInArbGasWithAggregator(address aggregator) view returns(uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoSession) GetPricesInArbGasWithAggregator(aggregator common.Address) (*big.Int, *big.Int, *big.Int, error) {
	return _ArbGasInfo.Contract.GetPricesInArbGasWithAggregator(&_ArbGasInfo.CallOpts, aggregator)
}

// GetPricesInArbGasWithAggregator is a free data retrieval call binding the contract method 0x7a1ea732.
//
// Solidity: function getPricesInArbGasWithAggregator(address aggregator) view returns(uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetPricesInArbGasWithAggregator(aggregator common.Address) (*big.Int, *big.Int, *big.Int, error) {
	return _ArbGasInfo.Contract.GetPricesInArbGasWithAggregator(&_ArbGasInfo.CallOpts, aggregator)
}

// GetPricesInWei is a free data retrieval call binding the contract method 0x41b247a8.
//
// Solidity: function getPricesInWei() view returns(uint256, uint256, uint256, uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoCaller) GetPricesInWei(opts *bind.CallOpts) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getPricesInWei")

	if err != nil {
		return *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	out3 := *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	out4 := *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	out5 := *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	return out0, out1, out2, out3, out4, out5, err

}

// GetPricesInWei is a free data retrieval call binding the contract method 0x41b247a8.
//
// Solidity: function getPricesInWei() view returns(uint256, uint256, uint256, uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoSession) GetPricesInWei() (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _ArbGasInfo.Contract.GetPricesInWei(&_ArbGasInfo.CallOpts)
}

// GetPricesInWei is a free data retrieval call binding the contract method 0x41b247a8.
//
// Solidity: function getPricesInWei() view returns(uint256, uint256, uint256, uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetPricesInWei() (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _ArbGasInfo.Contract.GetPricesInWei(&_ArbGasInfo.CallOpts)
}

// GetPricesInWeiWithAggregator is a free data retrieval call binding the contract method 0xba9c916e.
//
// Solidity: function getPricesInWeiWithAggregator(address aggregator) view returns(uint256, uint256, uint256, uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoCaller) GetPricesInWeiWithAggregator(opts *bind.CallOpts, aggregator common.Address) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getPricesInWeiWithAggregator", aggregator)

	if err != nil {
		return *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	out3 := *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	out4 := *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	out5 := *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	return out0, out1, out2, out3, out4, out5, err

}

// GetPricesInWeiWithAggregator is a free data retrieval call binding the contract method 0xba9c916e.
//
// Solidity: function getPricesInWeiWithAggregator(address aggregator) view returns(uint256, uint256, uint256, uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoSession) GetPricesInWeiWithAggregator(aggregator common.Address) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _ArbGasInfo.Contract.GetPricesInWeiWithAggregator(&_ArbGasInfo.CallOpts, aggregator)
}

// GetPricesInWeiWithAggregator is a free data retrieval call binding the contract method 0xba9c916e.
//
// Solidity: function getPricesInWeiWithAggregator(address aggregator) view returns(uint256, uint256, uint256, uint256, uint256, uint256)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetPricesInWeiWithAggregator(aggregator common.Address) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _ArbGasInfo.Contract.GetPricesInWeiWithAggregator(&_ArbGasInfo.CallOpts, aggregator)
}

// GetPricingInertia is a free data retrieval call binding the contract method 0x3dfb45b9.
//
// Solidity: function getPricingInertia() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCaller) GetPricingInertia(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ArbGasInfo.contract.Call(opts, &out, "getPricingInertia")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetPricingInertia is a free data retrieval call binding the contract method 0x3dfb45b9.
//
// Solidity: function getPricingInertia() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoSession) GetPricingInertia() (uint64, error) {
	return _ArbGasInfo.Contract.GetPricingInertia(&_ArbGasInfo.CallOpts)
}

// GetPricingInertia is a free data retrieval call binding the contract method 0x3dfb45b9.
//
// Solidity: function getPricingInertia() view returns(uint64)
func (_ArbGasInfo *ArbGasInfoCallerSession) GetPricingInertia() (uint64, error) {
	return _ArbGasInfo.Contract.GetPricingInertia(&_ArbGasInfo.CallOpts)
}

// ArbInfoMetaData contains all meta data concerning the ArbInfo contract.
var ArbInfoMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"getBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"getCode\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// ArbInfoABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbInfoMetaData.ABI instead.
var ArbInfoABI = ArbInfoMetaData.ABI

// ArbInfo is an auto generated Go binding around an Ethereum contract.
type ArbInfo struct {
	ArbInfoCaller     // Read-only binding to the contract
	ArbInfoTransactor // Write-only binding to the contract
	ArbInfoFilterer   // Log filterer for contract events
}

// ArbInfoCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbInfoCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbInfoTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbInfoTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbInfoFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbInfoFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbInfoSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbInfoSession struct {
	Contract     *ArbInfo          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbInfoCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbInfoCallerSession struct {
	Contract *ArbInfoCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// ArbInfoTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbInfoTransactorSession struct {
	Contract     *ArbInfoTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// ArbInfoRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbInfoRaw struct {
	Contract *ArbInfo // Generic contract binding to access the raw methods on
}

// ArbInfoCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbInfoCallerRaw struct {
	Contract *ArbInfoCaller // Generic read-only contract binding to access the raw methods on
}

// ArbInfoTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbInfoTransactorRaw struct {
	Contract *ArbInfoTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbInfo creates a new instance of ArbInfo, bound to a specific deployed contract.
func NewArbInfo(address common.Address, backend bind.ContractBackend) (*ArbInfo, error) {
	contract, err := bindArbInfo(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbInfo{ArbInfoCaller: ArbInfoCaller{contract: contract}, ArbInfoTransactor: ArbInfoTransactor{contract: contract}, ArbInfoFilterer: ArbInfoFilterer{contract: contract}}, nil
}

// NewArbInfoCaller creates a new read-only instance of ArbInfo, bound to a specific deployed contract.
func NewArbInfoCaller(address common.Address, caller bind.ContractCaller) (*ArbInfoCaller, error) {
	contract, err := bindArbInfo(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbInfoCaller{contract: contract}, nil
}

// NewArbInfoTransactor creates a new write-only instance of ArbInfo, bound to a specific deployed contract.
func NewArbInfoTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbInfoTransactor, error) {
	contract, err := bindArbInfo(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbInfoTransactor{contract: contract}, nil
}

// NewArbInfoFilterer creates a new log filterer instance of ArbInfo, bound to a specific deployed contract.
func NewArbInfoFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbInfoFilterer, error) {
	contract, err := bindArbInfo(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbInfoFilterer{contract: contract}, nil
}

// bindArbInfo binds a generic wrapper to an already deployed contract.
func bindArbInfo(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbInfoMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbInfo *ArbInfoRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbInfo.Contract.ArbInfoCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbInfo *ArbInfoRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbInfo.Contract.ArbInfoTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbInfo *ArbInfoRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbInfo.Contract.ArbInfoTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbInfo *ArbInfoCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbInfo.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbInfo *ArbInfoTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbInfo.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbInfo *ArbInfoTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbInfo.Contract.contract.Transact(opts, method, params...)
}

// GetBalance is a free data retrieval call binding the contract method 0xf8b2cb4f.
//
// Solidity: function getBalance(address account) view returns(uint256)
func (_ArbInfo *ArbInfoCaller) GetBalance(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ArbInfo.contract.Call(opts, &out, "getBalance", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBalance is a free data retrieval call binding the contract method 0xf8b2cb4f.
//
// Solidity: function getBalance(address account) view returns(uint256)
func (_ArbInfo *ArbInfoSession) GetBalance(account common.Address) (*big.Int, error) {
	return _ArbInfo.Contract.GetBalance(&_ArbInfo.CallOpts, account)
}

// GetBalance is a free data retrieval call binding the contract method 0xf8b2cb4f.
//
// Solidity: function getBalance(address account) view returns(uint256)
func (_ArbInfo *ArbInfoCallerSession) GetBalance(account common.Address) (*big.Int, error) {
	return _ArbInfo.Contract.GetBalance(&_ArbInfo.CallOpts, account)
}

// GetCode is a free data retrieval call binding the contract method 0x7e105ce2.
//
// Solidity: function getCode(address account) view returns(bytes)
func (_ArbInfo *ArbInfoCaller) GetCode(opts *bind.CallOpts, account common.Address) ([]byte, error) {
	var out []interface{}
	err := _ArbInfo.contract.Call(opts, &out, "getCode", account)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetCode is a free data retrieval call binding the contract method 0x7e105ce2.
//
// Solidity: function getCode(address account) view returns(bytes)
func (_ArbInfo *ArbInfoSession) GetCode(account common.Address) ([]byte, error) {
	return _ArbInfo.Contract.GetCode(&_ArbInfo.CallOpts, account)
}

// GetCode is a free data retrieval call binding the contract method 0x7e105ce2.
//
// Solidity: function getCode(address account) view returns(bytes)
func (_ArbInfo *ArbInfoCallerSession) GetCode(account common.Address) ([]byte, error) {
	return _ArbInfo.Contract.GetCode(&_ArbInfo.CallOpts, account)
}

// ArbOwnerMetaData contains all meta data concerning the ArbOwner contract.
var ArbOwnerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes4\",\"name\":\"method\",\"type\":\"bytes4\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"OwnerActs\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"addChainOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getAllChainOwners\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getInfraFeeAccount\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNetworkFeeAccount\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"isChainOwner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"maxWeiToRelease\",\"type\":\"uint256\"}],\"name\":\"releaseL1PricerSurplusFunds\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"ownerToRemove\",\"type\":\"address\"}],\"name\":\"removeChainOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"newVersion\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"scheduleArbOSUpgrade\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"cap\",\"type\":\"uint64\"}],\"name\":\"setAmortizedCostCapBips\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"chainConfig\",\"type\":\"string\"}],\"name\":\"setChainConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newInfraFeeAccount\",\"type\":\"address\"}],\"name\":\"setInfraFeeAccount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"inertia\",\"type\":\"uint64\"}],\"name\":\"setL1BaseFeeEstimateInertia\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"pricePerUnit\",\"type\":\"uint256\"}],\"name\":\"setL1PricePerUnit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"equilibrationUnits\",\"type\":\"uint256\"}],\"name\":\"setL1PricingEquilibrationUnits\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"inertia\",\"type\":\"uint64\"}],\"name\":\"setL1PricingInertia\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"weiPerUnit\",\"type\":\"uint64\"}],\"name\":\"setL1PricingRewardRate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"setL1PricingRewardRecipient\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"priceInWei\",\"type\":\"uint256\"}],\"name\":\"setL2BaseFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"sec\",\"type\":\"uint64\"}],\"name\":\"setL2GasBacklogTolerance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"sec\",\"type\":\"uint64\"}],\"name\":\"setL2GasPricingInertia\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"limit\",\"type\":\"uint64\"}],\"name\":\"setMaxTxGasLimit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"priceInWei\",\"type\":\"uint256\"}],\"name\":\"setMinimumL2BaseFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newNetworkFeeAccount\",\"type\":\"address\"}],\"name\":\"setNetworkFeeAccount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int64\",\"name\":\"cost\",\"type\":\"int64\"}],\"name\":\"setPerBatchGasCharge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"limit\",\"type\":\"uint64\"}],\"name\":\"setSpeedLimit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ArbOwnerABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbOwnerMetaData.ABI instead.
var ArbOwnerABI = ArbOwnerMetaData.ABI

// ArbOwner is an auto generated Go binding around an Ethereum contract.
type ArbOwner struct {
	ArbOwnerCaller     // Read-only binding to the contract
	ArbOwnerTransactor // Write-only binding to the contract
	ArbOwnerFilterer   // Log filterer for contract events
}

// ArbOwnerCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbOwnerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbOwnerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbOwnerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbOwnerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbOwnerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbOwnerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbOwnerSession struct {
	Contract     *ArbOwner         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbOwnerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbOwnerCallerSession struct {
	Contract *ArbOwnerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ArbOwnerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbOwnerTransactorSession struct {
	Contract     *ArbOwnerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ArbOwnerRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbOwnerRaw struct {
	Contract *ArbOwner // Generic contract binding to access the raw methods on
}

// ArbOwnerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbOwnerCallerRaw struct {
	Contract *ArbOwnerCaller // Generic read-only contract binding to access the raw methods on
}

// ArbOwnerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbOwnerTransactorRaw struct {
	Contract *ArbOwnerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbOwner creates a new instance of ArbOwner, bound to a specific deployed contract.
func NewArbOwner(address common.Address, backend bind.ContractBackend) (*ArbOwner, error) {
	contract, err := bindArbOwner(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbOwner{ArbOwnerCaller: ArbOwnerCaller{contract: contract}, ArbOwnerTransactor: ArbOwnerTransactor{contract: contract}, ArbOwnerFilterer: ArbOwnerFilterer{contract: contract}}, nil
}

// NewArbOwnerCaller creates a new read-only instance of ArbOwner, bound to a specific deployed contract.
func NewArbOwnerCaller(address common.Address, caller bind.ContractCaller) (*ArbOwnerCaller, error) {
	contract, err := bindArbOwner(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbOwnerCaller{contract: contract}, nil
}

// NewArbOwnerTransactor creates a new write-only instance of ArbOwner, bound to a specific deployed contract.
func NewArbOwnerTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbOwnerTransactor, error) {
	contract, err := bindArbOwner(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbOwnerTransactor{contract: contract}, nil
}

// NewArbOwnerFilterer creates a new log filterer instance of ArbOwner, bound to a specific deployed contract.
func NewArbOwnerFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbOwnerFilterer, error) {
	contract, err := bindArbOwner(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbOwnerFilterer{contract: contract}, nil
}

// bindArbOwner binds a generic wrapper to an already deployed contract.
func bindArbOwner(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbOwnerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbOwner *ArbOwnerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbOwner.Contract.ArbOwnerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbOwner *ArbOwnerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbOwner.Contract.ArbOwnerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbOwner *ArbOwnerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbOwner.Contract.ArbOwnerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbOwner *ArbOwnerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbOwner.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbOwner *ArbOwnerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbOwner.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbOwner *ArbOwnerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbOwner.Contract.contract.Transact(opts, method, params...)
}

// GetAllChainOwners is a free data retrieval call binding the contract method 0x516b4e0f.
//
// Solidity: function getAllChainOwners() view returns(address[])
func (_ArbOwner *ArbOwnerCaller) GetAllChainOwners(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _ArbOwner.contract.Call(opts, &out, "getAllChainOwners")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetAllChainOwners is a free data retrieval call binding the contract method 0x516b4e0f.
//
// Solidity: function getAllChainOwners() view returns(address[])
func (_ArbOwner *ArbOwnerSession) GetAllChainOwners() ([]common.Address, error) {
	return _ArbOwner.Contract.GetAllChainOwners(&_ArbOwner.CallOpts)
}

// GetAllChainOwners is a free data retrieval call binding the contract method 0x516b4e0f.
//
// Solidity: function getAllChainOwners() view returns(address[])
func (_ArbOwner *ArbOwnerCallerSession) GetAllChainOwners() ([]common.Address, error) {
	return _ArbOwner.Contract.GetAllChainOwners(&_ArbOwner.CallOpts)
}

// GetInfraFeeAccount is a free data retrieval call binding the contract method 0xee95a824.
//
// Solidity: function getInfraFeeAccount() view returns(address)
func (_ArbOwner *ArbOwnerCaller) GetInfraFeeAccount(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ArbOwner.contract.Call(opts, &out, "getInfraFeeAccount")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetInfraFeeAccount is a free data retrieval call binding the contract method 0xee95a824.
//
// Solidity: function getInfraFeeAccount() view returns(address)
func (_ArbOwner *ArbOwnerSession) GetInfraFeeAccount() (common.Address, error) {
	return _ArbOwner.Contract.GetInfraFeeAccount(&_ArbOwner.CallOpts)
}

// GetInfraFeeAccount is a free data retrieval call binding the contract method 0xee95a824.
//
// Solidity: function getInfraFeeAccount() view returns(address)
func (_ArbOwner *ArbOwnerCallerSession) GetInfraFeeAccount() (common.Address, error) {
	return _ArbOwner.Contract.GetInfraFeeAccount(&_ArbOwner.CallOpts)
}

// GetNetworkFeeAccount is a free data retrieval call binding the contract method 0x2d9125e9.
//
// Solidity: function getNetworkFeeAccount() view returns(address)
func (_ArbOwner *ArbOwnerCaller) GetNetworkFeeAccount(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ArbOwner.contract.Call(opts, &out, "getNetworkFeeAccount")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetNetworkFeeAccount is a free data retrieval call binding the contract method 0x2d9125e9.
//
// Solidity: function getNetworkFeeAccount() view returns(address)
func (_ArbOwner *ArbOwnerSession) GetNetworkFeeAccount() (common.Address, error) {
	return _ArbOwner.Contract.GetNetworkFeeAccount(&_ArbOwner.CallOpts)
}

// GetNetworkFeeAccount is a free data retrieval call binding the contract method 0x2d9125e9.
//
// Solidity: function getNetworkFeeAccount() view returns(address)
func (_ArbOwner *ArbOwnerCallerSession) GetNetworkFeeAccount() (common.Address, error) {
	return _ArbOwner.Contract.GetNetworkFeeAccount(&_ArbOwner.CallOpts)
}

// IsChainOwner is a free data retrieval call binding the contract method 0x26ef7f68.
//
// Solidity: function isChainOwner(address addr) view returns(bool)
func (_ArbOwner *ArbOwnerCaller) IsChainOwner(opts *bind.CallOpts, addr common.Address) (bool, error) {
	var out []interface{}
	err := _ArbOwner.contract.Call(opts, &out, "isChainOwner", addr)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsChainOwner is a free data retrieval call binding the contract method 0x26ef7f68.
//
// Solidity: function isChainOwner(address addr) view returns(bool)
func (_ArbOwner *ArbOwnerSession) IsChainOwner(addr common.Address) (bool, error) {
	return _ArbOwner.Contract.IsChainOwner(&_ArbOwner.CallOpts, addr)
}

// IsChainOwner is a free data retrieval call binding the contract method 0x26ef7f68.
//
// Solidity: function isChainOwner(address addr) view returns(bool)
func (_ArbOwner *ArbOwnerCallerSession) IsChainOwner(addr common.Address) (bool, error) {
	return _ArbOwner.Contract.IsChainOwner(&_ArbOwner.CallOpts, addr)
}

// AddChainOwner is a paid mutator transaction binding the contract method 0x481f8dbf.
//
// Solidity: function addChainOwner(address newOwner) returns()
func (_ArbOwner *ArbOwnerTransactor) AddChainOwner(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "addChainOwner", newOwner)
}

// AddChainOwner is a paid mutator transaction binding the contract method 0x481f8dbf.
//
// Solidity: function addChainOwner(address newOwner) returns()
func (_ArbOwner *ArbOwnerSession) AddChainOwner(newOwner common.Address) (*types.Transaction, error) {
	return _ArbOwner.Contract.AddChainOwner(&_ArbOwner.TransactOpts, newOwner)
}

// AddChainOwner is a paid mutator transaction binding the contract method 0x481f8dbf.
//
// Solidity: function addChainOwner(address newOwner) returns()
func (_ArbOwner *ArbOwnerTransactorSession) AddChainOwner(newOwner common.Address) (*types.Transaction, error) {
	return _ArbOwner.Contract.AddChainOwner(&_ArbOwner.TransactOpts, newOwner)
}

// ReleaseL1PricerSurplusFunds is a paid mutator transaction binding the contract method 0x314bcf05.
//
// Solidity: function releaseL1PricerSurplusFunds(uint256 maxWeiToRelease) returns(uint256)
func (_ArbOwner *ArbOwnerTransactor) ReleaseL1PricerSurplusFunds(opts *bind.TransactOpts, maxWeiToRelease *big.Int) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "releaseL1PricerSurplusFunds", maxWeiToRelease)
}

// ReleaseL1PricerSurplusFunds is a paid mutator transaction binding the contract method 0x314bcf05.
//
// Solidity: function releaseL1PricerSurplusFunds(uint256 maxWeiToRelease) returns(uint256)
func (_ArbOwner *ArbOwnerSession) ReleaseL1PricerSurplusFunds(maxWeiToRelease *big.Int) (*types.Transaction, error) {
	return _ArbOwner.Contract.ReleaseL1PricerSurplusFunds(&_ArbOwner.TransactOpts, maxWeiToRelease)
}

// ReleaseL1PricerSurplusFunds is a paid mutator transaction binding the contract method 0x314bcf05.
//
// Solidity: function releaseL1PricerSurplusFunds(uint256 maxWeiToRelease) returns(uint256)
func (_ArbOwner *ArbOwnerTransactorSession) ReleaseL1PricerSurplusFunds(maxWeiToRelease *big.Int) (*types.Transaction, error) {
	return _ArbOwner.Contract.ReleaseL1PricerSurplusFunds(&_ArbOwner.TransactOpts, maxWeiToRelease)
}

// RemoveChainOwner is a paid mutator transaction binding the contract method 0x8792701a.
//
// Solidity: function removeChainOwner(address ownerToRemove) returns()
func (_ArbOwner *ArbOwnerTransactor) RemoveChainOwner(opts *bind.TransactOpts, ownerToRemove common.Address) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "removeChainOwner", ownerToRemove)
}

// RemoveChainOwner is a paid mutator transaction binding the contract method 0x8792701a.
//
// Solidity: function removeChainOwner(address ownerToRemove) returns()
func (_ArbOwner *ArbOwnerSession) RemoveChainOwner(ownerToRemove common.Address) (*types.Transaction, error) {
	return _ArbOwner.Contract.RemoveChainOwner(&_ArbOwner.TransactOpts, ownerToRemove)
}

// RemoveChainOwner is a paid mutator transaction binding the contract method 0x8792701a.
//
// Solidity: function removeChainOwner(address ownerToRemove) returns()
func (_ArbOwner *ArbOwnerTransactorSession) RemoveChainOwner(ownerToRemove common.Address) (*types.Transaction, error) {
	return _ArbOwner.Contract.RemoveChainOwner(&_ArbOwner.TransactOpts, ownerToRemove)
}

// ScheduleArbOSUpgrade is a paid mutator transaction binding the contract method 0xe388b381.
//
// Solidity: function scheduleArbOSUpgrade(uint64 newVersion, uint64 timestamp) returns()
func (_ArbOwner *ArbOwnerTransactor) ScheduleArbOSUpgrade(opts *bind.TransactOpts, newVersion uint64, timestamp uint64) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "scheduleArbOSUpgrade", newVersion, timestamp)
}

// ScheduleArbOSUpgrade is a paid mutator transaction binding the contract method 0xe388b381.
//
// Solidity: function scheduleArbOSUpgrade(uint64 newVersion, uint64 timestamp) returns()
func (_ArbOwner *ArbOwnerSession) ScheduleArbOSUpgrade(newVersion uint64, timestamp uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.ScheduleArbOSUpgrade(&_ArbOwner.TransactOpts, newVersion, timestamp)
}

// ScheduleArbOSUpgrade is a paid mutator transaction binding the contract method 0xe388b381.
//
// Solidity: function scheduleArbOSUpgrade(uint64 newVersion, uint64 timestamp) returns()
func (_ArbOwner *ArbOwnerTransactorSession) ScheduleArbOSUpgrade(newVersion uint64, timestamp uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.ScheduleArbOSUpgrade(&_ArbOwner.TransactOpts, newVersion, timestamp)
}

// SetAmortizedCostCapBips is a paid mutator transaction binding the contract method 0x56191cc3.
//
// Solidity: function setAmortizedCostCapBips(uint64 cap) returns()
func (_ArbOwner *ArbOwnerTransactor) SetAmortizedCostCapBips(opts *bind.TransactOpts, cap uint64) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setAmortizedCostCapBips", cap)
}

// SetAmortizedCostCapBips is a paid mutator transaction binding the contract method 0x56191cc3.
//
// Solidity: function setAmortizedCostCapBips(uint64 cap) returns()
func (_ArbOwner *ArbOwnerSession) SetAmortizedCostCapBips(cap uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetAmortizedCostCapBips(&_ArbOwner.TransactOpts, cap)
}

// SetAmortizedCostCapBips is a paid mutator transaction binding the contract method 0x56191cc3.
//
// Solidity: function setAmortizedCostCapBips(uint64 cap) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetAmortizedCostCapBips(cap uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetAmortizedCostCapBips(&_ArbOwner.TransactOpts, cap)
}

// SetChainConfig is a paid mutator transaction binding the contract method 0xeda73212.
//
// Solidity: function setChainConfig(string chainConfig) returns()
func (_ArbOwner *ArbOwnerTransactor) SetChainConfig(opts *bind.TransactOpts, chainConfig string) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setChainConfig", chainConfig)
}

// SetChainConfig is a paid mutator transaction binding the contract method 0xeda73212.
//
// Solidity: function setChainConfig(string chainConfig) returns()
func (_ArbOwner *ArbOwnerSession) SetChainConfig(chainConfig string) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetChainConfig(&_ArbOwner.TransactOpts, chainConfig)
}

// SetChainConfig is a paid mutator transaction binding the contract method 0xeda73212.
//
// Solidity: function setChainConfig(string chainConfig) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetChainConfig(chainConfig string) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetChainConfig(&_ArbOwner.TransactOpts, chainConfig)
}

// SetInfraFeeAccount is a paid mutator transaction binding the contract method 0x57f585db.
//
// Solidity: function setInfraFeeAccount(address newInfraFeeAccount) returns()
func (_ArbOwner *ArbOwnerTransactor) SetInfraFeeAccount(opts *bind.TransactOpts, newInfraFeeAccount common.Address) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setInfraFeeAccount", newInfraFeeAccount)
}

// SetInfraFeeAccount is a paid mutator transaction binding the contract method 0x57f585db.
//
// Solidity: function setInfraFeeAccount(address newInfraFeeAccount) returns()
func (_ArbOwner *ArbOwnerSession) SetInfraFeeAccount(newInfraFeeAccount common.Address) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetInfraFeeAccount(&_ArbOwner.TransactOpts, newInfraFeeAccount)
}

// SetInfraFeeAccount is a paid mutator transaction binding the contract method 0x57f585db.
//
// Solidity: function setInfraFeeAccount(address newInfraFeeAccount) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetInfraFeeAccount(newInfraFeeAccount common.Address) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetInfraFeeAccount(&_ArbOwner.TransactOpts, newInfraFeeAccount)
}

// SetL1BaseFeeEstimateInertia is a paid mutator transaction binding the contract method 0x718f7805.
//
// Solidity: function setL1BaseFeeEstimateInertia(uint64 inertia) returns()
func (_ArbOwner *ArbOwnerTransactor) SetL1BaseFeeEstimateInertia(opts *bind.TransactOpts, inertia uint64) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setL1BaseFeeEstimateInertia", inertia)
}

// SetL1BaseFeeEstimateInertia is a paid mutator transaction binding the contract method 0x718f7805.
//
// Solidity: function setL1BaseFeeEstimateInertia(uint64 inertia) returns()
func (_ArbOwner *ArbOwnerSession) SetL1BaseFeeEstimateInertia(inertia uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1BaseFeeEstimateInertia(&_ArbOwner.TransactOpts, inertia)
}

// SetL1BaseFeeEstimateInertia is a paid mutator transaction binding the contract method 0x718f7805.
//
// Solidity: function setL1BaseFeeEstimateInertia(uint64 inertia) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetL1BaseFeeEstimateInertia(inertia uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1BaseFeeEstimateInertia(&_ArbOwner.TransactOpts, inertia)
}

// SetL1PricePerUnit is a paid mutator transaction binding the contract method 0x2b352fae.
//
// Solidity: function setL1PricePerUnit(uint256 pricePerUnit) returns()
func (_ArbOwner *ArbOwnerTransactor) SetL1PricePerUnit(opts *bind.TransactOpts, pricePerUnit *big.Int) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setL1PricePerUnit", pricePerUnit)
}

// SetL1PricePerUnit is a paid mutator transaction binding the contract method 0x2b352fae.
//
// Solidity: function setL1PricePerUnit(uint256 pricePerUnit) returns()
func (_ArbOwner *ArbOwnerSession) SetL1PricePerUnit(pricePerUnit *big.Int) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1PricePerUnit(&_ArbOwner.TransactOpts, pricePerUnit)
}

// SetL1PricePerUnit is a paid mutator transaction binding the contract method 0x2b352fae.
//
// Solidity: function setL1PricePerUnit(uint256 pricePerUnit) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetL1PricePerUnit(pricePerUnit *big.Int) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1PricePerUnit(&_ArbOwner.TransactOpts, pricePerUnit)
}

// SetL1PricingEquilibrationUnits is a paid mutator transaction binding the contract method 0x152db696.
//
// Solidity: function setL1PricingEquilibrationUnits(uint256 equilibrationUnits) returns()
func (_ArbOwner *ArbOwnerTransactor) SetL1PricingEquilibrationUnits(opts *bind.TransactOpts, equilibrationUnits *big.Int) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setL1PricingEquilibrationUnits", equilibrationUnits)
}

// SetL1PricingEquilibrationUnits is a paid mutator transaction binding the contract method 0x152db696.
//
// Solidity: function setL1PricingEquilibrationUnits(uint256 equilibrationUnits) returns()
func (_ArbOwner *ArbOwnerSession) SetL1PricingEquilibrationUnits(equilibrationUnits *big.Int) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1PricingEquilibrationUnits(&_ArbOwner.TransactOpts, equilibrationUnits)
}

// SetL1PricingEquilibrationUnits is a paid mutator transaction binding the contract method 0x152db696.
//
// Solidity: function setL1PricingEquilibrationUnits(uint256 equilibrationUnits) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetL1PricingEquilibrationUnits(equilibrationUnits *big.Int) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1PricingEquilibrationUnits(&_ArbOwner.TransactOpts, equilibrationUnits)
}

// SetL1PricingInertia is a paid mutator transaction binding the contract method 0x775a82e9.
//
// Solidity: function setL1PricingInertia(uint64 inertia) returns()
func (_ArbOwner *ArbOwnerTransactor) SetL1PricingInertia(opts *bind.TransactOpts, inertia uint64) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setL1PricingInertia", inertia)
}

// SetL1PricingInertia is a paid mutator transaction binding the contract method 0x775a82e9.
//
// Solidity: function setL1PricingInertia(uint64 inertia) returns()
func (_ArbOwner *ArbOwnerSession) SetL1PricingInertia(inertia uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1PricingInertia(&_ArbOwner.TransactOpts, inertia)
}

// SetL1PricingInertia is a paid mutator transaction binding the contract method 0x775a82e9.
//
// Solidity: function setL1PricingInertia(uint64 inertia) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetL1PricingInertia(inertia uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1PricingInertia(&_ArbOwner.TransactOpts, inertia)
}

// SetL1PricingRewardRate is a paid mutator transaction binding the contract method 0xf6739500.
//
// Solidity: function setL1PricingRewardRate(uint64 weiPerUnit) returns()
func (_ArbOwner *ArbOwnerTransactor) SetL1PricingRewardRate(opts *bind.TransactOpts, weiPerUnit uint64) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setL1PricingRewardRate", weiPerUnit)
}

// SetL1PricingRewardRate is a paid mutator transaction binding the contract method 0xf6739500.
//
// Solidity: function setL1PricingRewardRate(uint64 weiPerUnit) returns()
func (_ArbOwner *ArbOwnerSession) SetL1PricingRewardRate(weiPerUnit uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1PricingRewardRate(&_ArbOwner.TransactOpts, weiPerUnit)
}

// SetL1PricingRewardRate is a paid mutator transaction binding the contract method 0xf6739500.
//
// Solidity: function setL1PricingRewardRate(uint64 weiPerUnit) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetL1PricingRewardRate(weiPerUnit uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1PricingRewardRate(&_ArbOwner.TransactOpts, weiPerUnit)
}

// SetL1PricingRewardRecipient is a paid mutator transaction binding the contract method 0x934be07d.
//
// Solidity: function setL1PricingRewardRecipient(address recipient) returns()
func (_ArbOwner *ArbOwnerTransactor) SetL1PricingRewardRecipient(opts *bind.TransactOpts, recipient common.Address) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setL1PricingRewardRecipient", recipient)
}

// SetL1PricingRewardRecipient is a paid mutator transaction binding the contract method 0x934be07d.
//
// Solidity: function setL1PricingRewardRecipient(address recipient) returns()
func (_ArbOwner *ArbOwnerSession) SetL1PricingRewardRecipient(recipient common.Address) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1PricingRewardRecipient(&_ArbOwner.TransactOpts, recipient)
}

// SetL1PricingRewardRecipient is a paid mutator transaction binding the contract method 0x934be07d.
//
// Solidity: function setL1PricingRewardRecipient(address recipient) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetL1PricingRewardRecipient(recipient common.Address) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL1PricingRewardRecipient(&_ArbOwner.TransactOpts, recipient)
}

// SetL2BaseFee is a paid mutator transaction binding the contract method 0xd99bc80e.
//
// Solidity: function setL2BaseFee(uint256 priceInWei) returns()
func (_ArbOwner *ArbOwnerTransactor) SetL2BaseFee(opts *bind.TransactOpts, priceInWei *big.Int) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setL2BaseFee", priceInWei)
}

// SetL2BaseFee is a paid mutator transaction binding the contract method 0xd99bc80e.
//
// Solidity: function setL2BaseFee(uint256 priceInWei) returns()
func (_ArbOwner *ArbOwnerSession) SetL2BaseFee(priceInWei *big.Int) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL2BaseFee(&_ArbOwner.TransactOpts, priceInWei)
}

// SetL2BaseFee is a paid mutator transaction binding the contract method 0xd99bc80e.
//
// Solidity: function setL2BaseFee(uint256 priceInWei) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetL2BaseFee(priceInWei *big.Int) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL2BaseFee(&_ArbOwner.TransactOpts, priceInWei)
}

// SetL2GasBacklogTolerance is a paid mutator transaction binding the contract method 0x198e7157.
//
// Solidity: function setL2GasBacklogTolerance(uint64 sec) returns()
func (_ArbOwner *ArbOwnerTransactor) SetL2GasBacklogTolerance(opts *bind.TransactOpts, sec uint64) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setL2GasBacklogTolerance", sec)
}

// SetL2GasBacklogTolerance is a paid mutator transaction binding the contract method 0x198e7157.
//
// Solidity: function setL2GasBacklogTolerance(uint64 sec) returns()
func (_ArbOwner *ArbOwnerSession) SetL2GasBacklogTolerance(sec uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL2GasBacklogTolerance(&_ArbOwner.TransactOpts, sec)
}

// SetL2GasBacklogTolerance is a paid mutator transaction binding the contract method 0x198e7157.
//
// Solidity: function setL2GasBacklogTolerance(uint64 sec) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetL2GasBacklogTolerance(sec uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL2GasBacklogTolerance(&_ArbOwner.TransactOpts, sec)
}

// SetL2GasPricingInertia is a paid mutator transaction binding the contract method 0x3fd62a29.
//
// Solidity: function setL2GasPricingInertia(uint64 sec) returns()
func (_ArbOwner *ArbOwnerTransactor) SetL2GasPricingInertia(opts *bind.TransactOpts, sec uint64) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setL2GasPricingInertia", sec)
}

// SetL2GasPricingInertia is a paid mutator transaction binding the contract method 0x3fd62a29.
//
// Solidity: function setL2GasPricingInertia(uint64 sec) returns()
func (_ArbOwner *ArbOwnerSession) SetL2GasPricingInertia(sec uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL2GasPricingInertia(&_ArbOwner.TransactOpts, sec)
}

// SetL2GasPricingInertia is a paid mutator transaction binding the contract method 0x3fd62a29.
//
// Solidity: function setL2GasPricingInertia(uint64 sec) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetL2GasPricingInertia(sec uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetL2GasPricingInertia(&_ArbOwner.TransactOpts, sec)
}

// SetMaxTxGasLimit is a paid mutator transaction binding the contract method 0x39673611.
//
// Solidity: function setMaxTxGasLimit(uint64 limit) returns()
func (_ArbOwner *ArbOwnerTransactor) SetMaxTxGasLimit(opts *bind.TransactOpts, limit uint64) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setMaxTxGasLimit", limit)
}

// SetMaxTxGasLimit is a paid mutator transaction binding the contract method 0x39673611.
//
// Solidity: function setMaxTxGasLimit(uint64 limit) returns()
func (_ArbOwner *ArbOwnerSession) SetMaxTxGasLimit(limit uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetMaxTxGasLimit(&_ArbOwner.TransactOpts, limit)
}

// SetMaxTxGasLimit is a paid mutator transaction binding the contract method 0x39673611.
//
// Solidity: function setMaxTxGasLimit(uint64 limit) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetMaxTxGasLimit(limit uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetMaxTxGasLimit(&_ArbOwner.TransactOpts, limit)
}

// SetMinimumL2BaseFee is a paid mutator transaction binding the contract method 0xa0188cdb.
//
// Solidity: function setMinimumL2BaseFee(uint256 priceInWei) returns()
func (_ArbOwner *ArbOwnerTransactor) SetMinimumL2BaseFee(opts *bind.TransactOpts, priceInWei *big.Int) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setMinimumL2BaseFee", priceInWei)
}

// SetMinimumL2BaseFee is a paid mutator transaction binding the contract method 0xa0188cdb.
//
// Solidity: function setMinimumL2BaseFee(uint256 priceInWei) returns()
func (_ArbOwner *ArbOwnerSession) SetMinimumL2BaseFee(priceInWei *big.Int) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetMinimumL2BaseFee(&_ArbOwner.TransactOpts, priceInWei)
}

// SetMinimumL2BaseFee is a paid mutator transaction binding the contract method 0xa0188cdb.
//
// Solidity: function setMinimumL2BaseFee(uint256 priceInWei) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetMinimumL2BaseFee(priceInWei *big.Int) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetMinimumL2BaseFee(&_ArbOwner.TransactOpts, priceInWei)
}

// SetNetworkFeeAccount is a paid mutator transaction binding the contract method 0xfcdde2b4.
//
// Solidity: function setNetworkFeeAccount(address newNetworkFeeAccount) returns()
func (_ArbOwner *ArbOwnerTransactor) SetNetworkFeeAccount(opts *bind.TransactOpts, newNetworkFeeAccount common.Address) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setNetworkFeeAccount", newNetworkFeeAccount)
}

// SetNetworkFeeAccount is a paid mutator transaction binding the contract method 0xfcdde2b4.
//
// Solidity: function setNetworkFeeAccount(address newNetworkFeeAccount) returns()
func (_ArbOwner *ArbOwnerSession) SetNetworkFeeAccount(newNetworkFeeAccount common.Address) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetNetworkFeeAccount(&_ArbOwner.TransactOpts, newNetworkFeeAccount)
}

// SetNetworkFeeAccount is a paid mutator transaction binding the contract method 0xfcdde2b4.
//
// Solidity: function setNetworkFeeAccount(address newNetworkFeeAccount) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetNetworkFeeAccount(newNetworkFeeAccount common.Address) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetNetworkFeeAccount(&_ArbOwner.TransactOpts, newNetworkFeeAccount)
}

// SetPerBatchGasCharge is a paid mutator transaction binding the contract method 0xfad7f20b.
//
// Solidity: function setPerBatchGasCharge(int64 cost) returns()
func (_ArbOwner *ArbOwnerTransactor) SetPerBatchGasCharge(opts *bind.TransactOpts, cost int64) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setPerBatchGasCharge", cost)
}

// SetPerBatchGasCharge is a paid mutator transaction binding the contract method 0xfad7f20b.
//
// Solidity: function setPerBatchGasCharge(int64 cost) returns()
func (_ArbOwner *ArbOwnerSession) SetPerBatchGasCharge(cost int64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetPerBatchGasCharge(&_ArbOwner.TransactOpts, cost)
}

// SetPerBatchGasCharge is a paid mutator transaction binding the contract method 0xfad7f20b.
//
// Solidity: function setPerBatchGasCharge(int64 cost) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetPerBatchGasCharge(cost int64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetPerBatchGasCharge(&_ArbOwner.TransactOpts, cost)
}

// SetSpeedLimit is a paid mutator transaction binding the contract method 0x4d7a060d.
//
// Solidity: function setSpeedLimit(uint64 limit) returns()
func (_ArbOwner *ArbOwnerTransactor) SetSpeedLimit(opts *bind.TransactOpts, limit uint64) (*types.Transaction, error) {
	return _ArbOwner.contract.Transact(opts, "setSpeedLimit", limit)
}

// SetSpeedLimit is a paid mutator transaction binding the contract method 0x4d7a060d.
//
// Solidity: function setSpeedLimit(uint64 limit) returns()
func (_ArbOwner *ArbOwnerSession) SetSpeedLimit(limit uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetSpeedLimit(&_ArbOwner.TransactOpts, limit)
}

// SetSpeedLimit is a paid mutator transaction binding the contract method 0x4d7a060d.
//
// Solidity: function setSpeedLimit(uint64 limit) returns()
func (_ArbOwner *ArbOwnerTransactorSession) SetSpeedLimit(limit uint64) (*types.Transaction, error) {
	return _ArbOwner.Contract.SetSpeedLimit(&_ArbOwner.TransactOpts, limit)
}

// ArbOwnerOwnerActsIterator is returned from FilterOwnerActs and is used to iterate over the raw logs and unpacked data for OwnerActs events raised by the ArbOwner contract.
type ArbOwnerOwnerActsIterator struct {
	Event *ArbOwnerOwnerActs // Event containing the contract specifics and raw log

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
func (it *ArbOwnerOwnerActsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbOwnerOwnerActs)
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
		it.Event = new(ArbOwnerOwnerActs)
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
func (it *ArbOwnerOwnerActsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbOwnerOwnerActsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbOwnerOwnerActs represents a OwnerActs event raised by the ArbOwner contract.
type ArbOwnerOwnerActs struct {
	Method [4]byte
	Owner  common.Address
	Data   []byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterOwnerActs is a free log retrieval operation binding the contract event 0x3c9e6a772755407311e3b35b3ee56799df8f87395941b3a658eee9e08a67ebda.
//
// Solidity: event OwnerActs(bytes4 indexed method, address indexed owner, bytes data)
func (_ArbOwner *ArbOwnerFilterer) FilterOwnerActs(opts *bind.FilterOpts, method [][4]byte, owner []common.Address) (*ArbOwnerOwnerActsIterator, error) {

	var methodRule []interface{}
	for _, methodItem := range method {
		methodRule = append(methodRule, methodItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _ArbOwner.contract.FilterLogs(opts, "OwnerActs", methodRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &ArbOwnerOwnerActsIterator{contract: _ArbOwner.contract, event: "OwnerActs", logs: logs, sub: sub}, nil
}

// WatchOwnerActs is a free log subscription operation binding the contract event 0x3c9e6a772755407311e3b35b3ee56799df8f87395941b3a658eee9e08a67ebda.
//
// Solidity: event OwnerActs(bytes4 indexed method, address indexed owner, bytes data)
func (_ArbOwner *ArbOwnerFilterer) WatchOwnerActs(opts *bind.WatchOpts, sink chan<- *ArbOwnerOwnerActs, method [][4]byte, owner []common.Address) (event.Subscription, error) {

	var methodRule []interface{}
	for _, methodItem := range method {
		methodRule = append(methodRule, methodItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _ArbOwner.contract.WatchLogs(opts, "OwnerActs", methodRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbOwnerOwnerActs)
				if err := _ArbOwner.contract.UnpackLog(event, "OwnerActs", log); err != nil {
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

// ParseOwnerActs is a log parse operation binding the contract event 0x3c9e6a772755407311e3b35b3ee56799df8f87395941b3a658eee9e08a67ebda.
//
// Solidity: event OwnerActs(bytes4 indexed method, address indexed owner, bytes data)
func (_ArbOwner *ArbOwnerFilterer) ParseOwnerActs(log types.Log) (*ArbOwnerOwnerActs, error) {
	event := new(ArbOwnerOwnerActs)
	if err := _ArbOwner.contract.UnpackLog(event, "OwnerActs", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbOwnerPublicMetaData contains all meta data concerning the ArbOwnerPublic contract.
var ArbOwnerPublicMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rectifiedOwner\",\"type\":\"address\"}],\"name\":\"ChainOwnerRectified\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"getAllChainOwners\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getInfraFeeAccount\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNetworkFeeAccount\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"isChainOwner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"ownerToRectify\",\"type\":\"address\"}],\"name\":\"rectifyChainOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ArbOwnerPublicABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbOwnerPublicMetaData.ABI instead.
var ArbOwnerPublicABI = ArbOwnerPublicMetaData.ABI

// ArbOwnerPublic is an auto generated Go binding around an Ethereum contract.
type ArbOwnerPublic struct {
	ArbOwnerPublicCaller     // Read-only binding to the contract
	ArbOwnerPublicTransactor // Write-only binding to the contract
	ArbOwnerPublicFilterer   // Log filterer for contract events
}

// ArbOwnerPublicCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbOwnerPublicCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbOwnerPublicTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbOwnerPublicTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbOwnerPublicFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbOwnerPublicFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbOwnerPublicSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbOwnerPublicSession struct {
	Contract     *ArbOwnerPublic   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbOwnerPublicCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbOwnerPublicCallerSession struct {
	Contract *ArbOwnerPublicCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// ArbOwnerPublicTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbOwnerPublicTransactorSession struct {
	Contract     *ArbOwnerPublicTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// ArbOwnerPublicRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbOwnerPublicRaw struct {
	Contract *ArbOwnerPublic // Generic contract binding to access the raw methods on
}

// ArbOwnerPublicCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbOwnerPublicCallerRaw struct {
	Contract *ArbOwnerPublicCaller // Generic read-only contract binding to access the raw methods on
}

// ArbOwnerPublicTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbOwnerPublicTransactorRaw struct {
	Contract *ArbOwnerPublicTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbOwnerPublic creates a new instance of ArbOwnerPublic, bound to a specific deployed contract.
func NewArbOwnerPublic(address common.Address, backend bind.ContractBackend) (*ArbOwnerPublic, error) {
	contract, err := bindArbOwnerPublic(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbOwnerPublic{ArbOwnerPublicCaller: ArbOwnerPublicCaller{contract: contract}, ArbOwnerPublicTransactor: ArbOwnerPublicTransactor{contract: contract}, ArbOwnerPublicFilterer: ArbOwnerPublicFilterer{contract: contract}}, nil
}

// NewArbOwnerPublicCaller creates a new read-only instance of ArbOwnerPublic, bound to a specific deployed contract.
func NewArbOwnerPublicCaller(address common.Address, caller bind.ContractCaller) (*ArbOwnerPublicCaller, error) {
	contract, err := bindArbOwnerPublic(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbOwnerPublicCaller{contract: contract}, nil
}

// NewArbOwnerPublicTransactor creates a new write-only instance of ArbOwnerPublic, bound to a specific deployed contract.
func NewArbOwnerPublicTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbOwnerPublicTransactor, error) {
	contract, err := bindArbOwnerPublic(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbOwnerPublicTransactor{contract: contract}, nil
}

// NewArbOwnerPublicFilterer creates a new log filterer instance of ArbOwnerPublic, bound to a specific deployed contract.
func NewArbOwnerPublicFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbOwnerPublicFilterer, error) {
	contract, err := bindArbOwnerPublic(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbOwnerPublicFilterer{contract: contract}, nil
}

// bindArbOwnerPublic binds a generic wrapper to an already deployed contract.
func bindArbOwnerPublic(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbOwnerPublicMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbOwnerPublic *ArbOwnerPublicRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbOwnerPublic.Contract.ArbOwnerPublicCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbOwnerPublic *ArbOwnerPublicRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbOwnerPublic.Contract.ArbOwnerPublicTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbOwnerPublic *ArbOwnerPublicRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbOwnerPublic.Contract.ArbOwnerPublicTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbOwnerPublic *ArbOwnerPublicCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbOwnerPublic.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbOwnerPublic *ArbOwnerPublicTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbOwnerPublic.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbOwnerPublic *ArbOwnerPublicTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbOwnerPublic.Contract.contract.Transact(opts, method, params...)
}

// GetAllChainOwners is a free data retrieval call binding the contract method 0x516b4e0f.
//
// Solidity: function getAllChainOwners() view returns(address[])
func (_ArbOwnerPublic *ArbOwnerPublicCaller) GetAllChainOwners(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _ArbOwnerPublic.contract.Call(opts, &out, "getAllChainOwners")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetAllChainOwners is a free data retrieval call binding the contract method 0x516b4e0f.
//
// Solidity: function getAllChainOwners() view returns(address[])
func (_ArbOwnerPublic *ArbOwnerPublicSession) GetAllChainOwners() ([]common.Address, error) {
	return _ArbOwnerPublic.Contract.GetAllChainOwners(&_ArbOwnerPublic.CallOpts)
}

// GetAllChainOwners is a free data retrieval call binding the contract method 0x516b4e0f.
//
// Solidity: function getAllChainOwners() view returns(address[])
func (_ArbOwnerPublic *ArbOwnerPublicCallerSession) GetAllChainOwners() ([]common.Address, error) {
	return _ArbOwnerPublic.Contract.GetAllChainOwners(&_ArbOwnerPublic.CallOpts)
}

// GetInfraFeeAccount is a free data retrieval call binding the contract method 0xee95a824.
//
// Solidity: function getInfraFeeAccount() view returns(address)
func (_ArbOwnerPublic *ArbOwnerPublicCaller) GetInfraFeeAccount(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ArbOwnerPublic.contract.Call(opts, &out, "getInfraFeeAccount")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetInfraFeeAccount is a free data retrieval call binding the contract method 0xee95a824.
//
// Solidity: function getInfraFeeAccount() view returns(address)
func (_ArbOwnerPublic *ArbOwnerPublicSession) GetInfraFeeAccount() (common.Address, error) {
	return _ArbOwnerPublic.Contract.GetInfraFeeAccount(&_ArbOwnerPublic.CallOpts)
}

// GetInfraFeeAccount is a free data retrieval call binding the contract method 0xee95a824.
//
// Solidity: function getInfraFeeAccount() view returns(address)
func (_ArbOwnerPublic *ArbOwnerPublicCallerSession) GetInfraFeeAccount() (common.Address, error) {
	return _ArbOwnerPublic.Contract.GetInfraFeeAccount(&_ArbOwnerPublic.CallOpts)
}

// GetNetworkFeeAccount is a free data retrieval call binding the contract method 0x2d9125e9.
//
// Solidity: function getNetworkFeeAccount() view returns(address)
func (_ArbOwnerPublic *ArbOwnerPublicCaller) GetNetworkFeeAccount(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ArbOwnerPublic.contract.Call(opts, &out, "getNetworkFeeAccount")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetNetworkFeeAccount is a free data retrieval call binding the contract method 0x2d9125e9.
//
// Solidity: function getNetworkFeeAccount() view returns(address)
func (_ArbOwnerPublic *ArbOwnerPublicSession) GetNetworkFeeAccount() (common.Address, error) {
	return _ArbOwnerPublic.Contract.GetNetworkFeeAccount(&_ArbOwnerPublic.CallOpts)
}

// GetNetworkFeeAccount is a free data retrieval call binding the contract method 0x2d9125e9.
//
// Solidity: function getNetworkFeeAccount() view returns(address)
func (_ArbOwnerPublic *ArbOwnerPublicCallerSession) GetNetworkFeeAccount() (common.Address, error) {
	return _ArbOwnerPublic.Contract.GetNetworkFeeAccount(&_ArbOwnerPublic.CallOpts)
}

// IsChainOwner is a free data retrieval call binding the contract method 0x26ef7f68.
//
// Solidity: function isChainOwner(address addr) view returns(bool)
func (_ArbOwnerPublic *ArbOwnerPublicCaller) IsChainOwner(opts *bind.CallOpts, addr common.Address) (bool, error) {
	var out []interface{}
	err := _ArbOwnerPublic.contract.Call(opts, &out, "isChainOwner", addr)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsChainOwner is a free data retrieval call binding the contract method 0x26ef7f68.
//
// Solidity: function isChainOwner(address addr) view returns(bool)
func (_ArbOwnerPublic *ArbOwnerPublicSession) IsChainOwner(addr common.Address) (bool, error) {
	return _ArbOwnerPublic.Contract.IsChainOwner(&_ArbOwnerPublic.CallOpts, addr)
}

// IsChainOwner is a free data retrieval call binding the contract method 0x26ef7f68.
//
// Solidity: function isChainOwner(address addr) view returns(bool)
func (_ArbOwnerPublic *ArbOwnerPublicCallerSession) IsChainOwner(addr common.Address) (bool, error) {
	return _ArbOwnerPublic.Contract.IsChainOwner(&_ArbOwnerPublic.CallOpts, addr)
}

// RectifyChainOwner is a paid mutator transaction binding the contract method 0x6fe86373.
//
// Solidity: function rectifyChainOwner(address ownerToRectify) returns()
func (_ArbOwnerPublic *ArbOwnerPublicTransactor) RectifyChainOwner(opts *bind.TransactOpts, ownerToRectify common.Address) (*types.Transaction, error) {
	return _ArbOwnerPublic.contract.Transact(opts, "rectifyChainOwner", ownerToRectify)
}

// RectifyChainOwner is a paid mutator transaction binding the contract method 0x6fe86373.
//
// Solidity: function rectifyChainOwner(address ownerToRectify) returns()
func (_ArbOwnerPublic *ArbOwnerPublicSession) RectifyChainOwner(ownerToRectify common.Address) (*types.Transaction, error) {
	return _ArbOwnerPublic.Contract.RectifyChainOwner(&_ArbOwnerPublic.TransactOpts, ownerToRectify)
}

// RectifyChainOwner is a paid mutator transaction binding the contract method 0x6fe86373.
//
// Solidity: function rectifyChainOwner(address ownerToRectify) returns()
func (_ArbOwnerPublic *ArbOwnerPublicTransactorSession) RectifyChainOwner(ownerToRectify common.Address) (*types.Transaction, error) {
	return _ArbOwnerPublic.Contract.RectifyChainOwner(&_ArbOwnerPublic.TransactOpts, ownerToRectify)
}

// ArbOwnerPublicChainOwnerRectifiedIterator is returned from FilterChainOwnerRectified and is used to iterate over the raw logs and unpacked data for ChainOwnerRectified events raised by the ArbOwnerPublic contract.
type ArbOwnerPublicChainOwnerRectifiedIterator struct {
	Event *ArbOwnerPublicChainOwnerRectified // Event containing the contract specifics and raw log

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
func (it *ArbOwnerPublicChainOwnerRectifiedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbOwnerPublicChainOwnerRectified)
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
		it.Event = new(ArbOwnerPublicChainOwnerRectified)
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
func (it *ArbOwnerPublicChainOwnerRectifiedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbOwnerPublicChainOwnerRectifiedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbOwnerPublicChainOwnerRectified represents a ChainOwnerRectified event raised by the ArbOwnerPublic contract.
type ArbOwnerPublicChainOwnerRectified struct {
	RectifiedOwner common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterChainOwnerRectified is a free log retrieval operation binding the contract event 0x14c7c9cb05f84448a0f2fc5775a4048a7210cb040a35fd84cd45b2b863d04d82.
//
// Solidity: event ChainOwnerRectified(address rectifiedOwner)
func (_ArbOwnerPublic *ArbOwnerPublicFilterer) FilterChainOwnerRectified(opts *bind.FilterOpts) (*ArbOwnerPublicChainOwnerRectifiedIterator, error) {

	logs, sub, err := _ArbOwnerPublic.contract.FilterLogs(opts, "ChainOwnerRectified")
	if err != nil {
		return nil, err
	}
	return &ArbOwnerPublicChainOwnerRectifiedIterator{contract: _ArbOwnerPublic.contract, event: "ChainOwnerRectified", logs: logs, sub: sub}, nil
}

// WatchChainOwnerRectified is a free log subscription operation binding the contract event 0x14c7c9cb05f84448a0f2fc5775a4048a7210cb040a35fd84cd45b2b863d04d82.
//
// Solidity: event ChainOwnerRectified(address rectifiedOwner)
func (_ArbOwnerPublic *ArbOwnerPublicFilterer) WatchChainOwnerRectified(opts *bind.WatchOpts, sink chan<- *ArbOwnerPublicChainOwnerRectified) (event.Subscription, error) {

	logs, sub, err := _ArbOwnerPublic.contract.WatchLogs(opts, "ChainOwnerRectified")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbOwnerPublicChainOwnerRectified)
				if err := _ArbOwnerPublic.contract.UnpackLog(event, "ChainOwnerRectified", log); err != nil {
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

// ParseChainOwnerRectified is a log parse operation binding the contract event 0x14c7c9cb05f84448a0f2fc5775a4048a7210cb040a35fd84cd45b2b863d04d82.
//
// Solidity: event ChainOwnerRectified(address rectifiedOwner)
func (_ArbOwnerPublic *ArbOwnerPublicFilterer) ParseChainOwnerRectified(log types.Log) (*ArbOwnerPublicChainOwnerRectified, error) {
	event := new(ArbOwnerPublicChainOwnerRectified)
	if err := _ArbOwnerPublic.contract.UnpackLog(event, "ChainOwnerRectified", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbRetryableTxMetaData contains all meta data concerning the ArbRetryableTx contract.
var ArbRetryableTxMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"NoTicketWithID\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotCallable\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"ticketId\",\"type\":\"bytes32\"}],\"name\":\"Canceled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"ticketId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newTimeout\",\"type\":\"uint256\"}],\"name\":\"LifetimeExtended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"ticketId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"retryTxHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"sequenceNum\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"donatedGas\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"gasDonor\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"maxRefund\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"submissionFeeRefund\",\"type\":\"uint256\"}],\"name\":\"RedeemScheduled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"userTxHash\",\"type\":\"bytes32\"}],\"name\":\"Redeemed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"ticketId\",\"type\":\"bytes32\"}],\"name\":\"TicketCreated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ticketId\",\"type\":\"bytes32\"}],\"name\":\"cancel\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ticketId\",\"type\":\"bytes32\"}],\"name\":\"getBeneficiary\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentRedeemer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLifetime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ticketId\",\"type\":\"bytes32\"}],\"name\":\"getTimeout\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ticketId\",\"type\":\"bytes32\"}],\"name\":\"keepalive\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ticketId\",\"type\":\"bytes32\"}],\"name\":\"redeem\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"requestId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"l1BaseFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deposit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callvalue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasFeeCap\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"gasLimit\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"maxSubmissionFee\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"feeRefundAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"beneficiary\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"retryTo\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"retryData\",\"type\":\"bytes\"}],\"name\":\"submitRetryable\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ArbRetryableTxABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbRetryableTxMetaData.ABI instead.
var ArbRetryableTxABI = ArbRetryableTxMetaData.ABI

// ArbRetryableTx is an auto generated Go binding around an Ethereum contract.
type ArbRetryableTx struct {
	ArbRetryableTxCaller     // Read-only binding to the contract
	ArbRetryableTxTransactor // Write-only binding to the contract
	ArbRetryableTxFilterer   // Log filterer for contract events
}

// ArbRetryableTxCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbRetryableTxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbRetryableTxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbRetryableTxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbRetryableTxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbRetryableTxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbRetryableTxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbRetryableTxSession struct {
	Contract     *ArbRetryableTx   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbRetryableTxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbRetryableTxCallerSession struct {
	Contract *ArbRetryableTxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// ArbRetryableTxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbRetryableTxTransactorSession struct {
	Contract     *ArbRetryableTxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// ArbRetryableTxRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbRetryableTxRaw struct {
	Contract *ArbRetryableTx // Generic contract binding to access the raw methods on
}

// ArbRetryableTxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbRetryableTxCallerRaw struct {
	Contract *ArbRetryableTxCaller // Generic read-only contract binding to access the raw methods on
}

// ArbRetryableTxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbRetryableTxTransactorRaw struct {
	Contract *ArbRetryableTxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbRetryableTx creates a new instance of ArbRetryableTx, bound to a specific deployed contract.
func NewArbRetryableTx(address common.Address, backend bind.ContractBackend) (*ArbRetryableTx, error) {
	contract, err := bindArbRetryableTx(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbRetryableTx{ArbRetryableTxCaller: ArbRetryableTxCaller{contract: contract}, ArbRetryableTxTransactor: ArbRetryableTxTransactor{contract: contract}, ArbRetryableTxFilterer: ArbRetryableTxFilterer{contract: contract}}, nil
}

// NewArbRetryableTxCaller creates a new read-only instance of ArbRetryableTx, bound to a specific deployed contract.
func NewArbRetryableTxCaller(address common.Address, caller bind.ContractCaller) (*ArbRetryableTxCaller, error) {
	contract, err := bindArbRetryableTx(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbRetryableTxCaller{contract: contract}, nil
}

// NewArbRetryableTxTransactor creates a new write-only instance of ArbRetryableTx, bound to a specific deployed contract.
func NewArbRetryableTxTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbRetryableTxTransactor, error) {
	contract, err := bindArbRetryableTx(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbRetryableTxTransactor{contract: contract}, nil
}

// NewArbRetryableTxFilterer creates a new log filterer instance of ArbRetryableTx, bound to a specific deployed contract.
func NewArbRetryableTxFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbRetryableTxFilterer, error) {
	contract, err := bindArbRetryableTx(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbRetryableTxFilterer{contract: contract}, nil
}

// bindArbRetryableTx binds a generic wrapper to an already deployed contract.
func bindArbRetryableTx(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbRetryableTxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbRetryableTx *ArbRetryableTxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbRetryableTx.Contract.ArbRetryableTxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbRetryableTx *ArbRetryableTxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.ArbRetryableTxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbRetryableTx *ArbRetryableTxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.ArbRetryableTxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbRetryableTx *ArbRetryableTxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbRetryableTx.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbRetryableTx *ArbRetryableTxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbRetryableTx *ArbRetryableTxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.contract.Transact(opts, method, params...)
}

// GetBeneficiary is a free data retrieval call binding the contract method 0xba20dda4.
//
// Solidity: function getBeneficiary(bytes32 ticketId) view returns(address)
func (_ArbRetryableTx *ArbRetryableTxCaller) GetBeneficiary(opts *bind.CallOpts, ticketId [32]byte) (common.Address, error) {
	var out []interface{}
	err := _ArbRetryableTx.contract.Call(opts, &out, "getBeneficiary", ticketId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetBeneficiary is a free data retrieval call binding the contract method 0xba20dda4.
//
// Solidity: function getBeneficiary(bytes32 ticketId) view returns(address)
func (_ArbRetryableTx *ArbRetryableTxSession) GetBeneficiary(ticketId [32]byte) (common.Address, error) {
	return _ArbRetryableTx.Contract.GetBeneficiary(&_ArbRetryableTx.CallOpts, ticketId)
}

// GetBeneficiary is a free data retrieval call binding the contract method 0xba20dda4.
//
// Solidity: function getBeneficiary(bytes32 ticketId) view returns(address)
func (_ArbRetryableTx *ArbRetryableTxCallerSession) GetBeneficiary(ticketId [32]byte) (common.Address, error) {
	return _ArbRetryableTx.Contract.GetBeneficiary(&_ArbRetryableTx.CallOpts, ticketId)
}

// GetCurrentRedeemer is a free data retrieval call binding the contract method 0xde4ba2b3.
//
// Solidity: function getCurrentRedeemer() view returns(address)
func (_ArbRetryableTx *ArbRetryableTxCaller) GetCurrentRedeemer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ArbRetryableTx.contract.Call(opts, &out, "getCurrentRedeemer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetCurrentRedeemer is a free data retrieval call binding the contract method 0xde4ba2b3.
//
// Solidity: function getCurrentRedeemer() view returns(address)
func (_ArbRetryableTx *ArbRetryableTxSession) GetCurrentRedeemer() (common.Address, error) {
	return _ArbRetryableTx.Contract.GetCurrentRedeemer(&_ArbRetryableTx.CallOpts)
}

// GetCurrentRedeemer is a free data retrieval call binding the contract method 0xde4ba2b3.
//
// Solidity: function getCurrentRedeemer() view returns(address)
func (_ArbRetryableTx *ArbRetryableTxCallerSession) GetCurrentRedeemer() (common.Address, error) {
	return _ArbRetryableTx.Contract.GetCurrentRedeemer(&_ArbRetryableTx.CallOpts)
}

// GetLifetime is a free data retrieval call binding the contract method 0x81e6e083.
//
// Solidity: function getLifetime() view returns(uint256)
func (_ArbRetryableTx *ArbRetryableTxCaller) GetLifetime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbRetryableTx.contract.Call(opts, &out, "getLifetime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLifetime is a free data retrieval call binding the contract method 0x81e6e083.
//
// Solidity: function getLifetime() view returns(uint256)
func (_ArbRetryableTx *ArbRetryableTxSession) GetLifetime() (*big.Int, error) {
	return _ArbRetryableTx.Contract.GetLifetime(&_ArbRetryableTx.CallOpts)
}

// GetLifetime is a free data retrieval call binding the contract method 0x81e6e083.
//
// Solidity: function getLifetime() view returns(uint256)
func (_ArbRetryableTx *ArbRetryableTxCallerSession) GetLifetime() (*big.Int, error) {
	return _ArbRetryableTx.Contract.GetLifetime(&_ArbRetryableTx.CallOpts)
}

// GetTimeout is a free data retrieval call binding the contract method 0x9f1025c6.
//
// Solidity: function getTimeout(bytes32 ticketId) view returns(uint256)
func (_ArbRetryableTx *ArbRetryableTxCaller) GetTimeout(opts *bind.CallOpts, ticketId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _ArbRetryableTx.contract.Call(opts, &out, "getTimeout", ticketId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTimeout is a free data retrieval call binding the contract method 0x9f1025c6.
//
// Solidity: function getTimeout(bytes32 ticketId) view returns(uint256)
func (_ArbRetryableTx *ArbRetryableTxSession) GetTimeout(ticketId [32]byte) (*big.Int, error) {
	return _ArbRetryableTx.Contract.GetTimeout(&_ArbRetryableTx.CallOpts, ticketId)
}

// GetTimeout is a free data retrieval call binding the contract method 0x9f1025c6.
//
// Solidity: function getTimeout(bytes32 ticketId) view returns(uint256)
func (_ArbRetryableTx *ArbRetryableTxCallerSession) GetTimeout(ticketId [32]byte) (*big.Int, error) {
	return _ArbRetryableTx.Contract.GetTimeout(&_ArbRetryableTx.CallOpts, ticketId)
}

// Cancel is a paid mutator transaction binding the contract method 0xc4d252f5.
//
// Solidity: function cancel(bytes32 ticketId) returns()
func (_ArbRetryableTx *ArbRetryableTxTransactor) Cancel(opts *bind.TransactOpts, ticketId [32]byte) (*types.Transaction, error) {
	return _ArbRetryableTx.contract.Transact(opts, "cancel", ticketId)
}

// Cancel is a paid mutator transaction binding the contract method 0xc4d252f5.
//
// Solidity: function cancel(bytes32 ticketId) returns()
func (_ArbRetryableTx *ArbRetryableTxSession) Cancel(ticketId [32]byte) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.Cancel(&_ArbRetryableTx.TransactOpts, ticketId)
}

// Cancel is a paid mutator transaction binding the contract method 0xc4d252f5.
//
// Solidity: function cancel(bytes32 ticketId) returns()
func (_ArbRetryableTx *ArbRetryableTxTransactorSession) Cancel(ticketId [32]byte) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.Cancel(&_ArbRetryableTx.TransactOpts, ticketId)
}

// Keepalive is a paid mutator transaction binding the contract method 0xf0b21a41.
//
// Solidity: function keepalive(bytes32 ticketId) returns(uint256)
func (_ArbRetryableTx *ArbRetryableTxTransactor) Keepalive(opts *bind.TransactOpts, ticketId [32]byte) (*types.Transaction, error) {
	return _ArbRetryableTx.contract.Transact(opts, "keepalive", ticketId)
}

// Keepalive is a paid mutator transaction binding the contract method 0xf0b21a41.
//
// Solidity: function keepalive(bytes32 ticketId) returns(uint256)
func (_ArbRetryableTx *ArbRetryableTxSession) Keepalive(ticketId [32]byte) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.Keepalive(&_ArbRetryableTx.TransactOpts, ticketId)
}

// Keepalive is a paid mutator transaction binding the contract method 0xf0b21a41.
//
// Solidity: function keepalive(bytes32 ticketId) returns(uint256)
func (_ArbRetryableTx *ArbRetryableTxTransactorSession) Keepalive(ticketId [32]byte) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.Keepalive(&_ArbRetryableTx.TransactOpts, ticketId)
}

// Redeem is a paid mutator transaction binding the contract method 0xeda1122c.
//
// Solidity: function redeem(bytes32 ticketId) returns(bytes32)
func (_ArbRetryableTx *ArbRetryableTxTransactor) Redeem(opts *bind.TransactOpts, ticketId [32]byte) (*types.Transaction, error) {
	return _ArbRetryableTx.contract.Transact(opts, "redeem", ticketId)
}

// Redeem is a paid mutator transaction binding the contract method 0xeda1122c.
//
// Solidity: function redeem(bytes32 ticketId) returns(bytes32)
func (_ArbRetryableTx *ArbRetryableTxSession) Redeem(ticketId [32]byte) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.Redeem(&_ArbRetryableTx.TransactOpts, ticketId)
}

// Redeem is a paid mutator transaction binding the contract method 0xeda1122c.
//
// Solidity: function redeem(bytes32 ticketId) returns(bytes32)
func (_ArbRetryableTx *ArbRetryableTxTransactorSession) Redeem(ticketId [32]byte) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.Redeem(&_ArbRetryableTx.TransactOpts, ticketId)
}

// SubmitRetryable is a paid mutator transaction binding the contract method 0xc9f95d32.
//
// Solidity: function submitRetryable(bytes32 requestId, uint256 l1BaseFee, uint256 deposit, uint256 callvalue, uint256 gasFeeCap, uint64 gasLimit, uint256 maxSubmissionFee, address feeRefundAddress, address beneficiary, address retryTo, bytes retryData) returns()
func (_ArbRetryableTx *ArbRetryableTxTransactor) SubmitRetryable(opts *bind.TransactOpts, requestId [32]byte, l1BaseFee *big.Int, deposit *big.Int, callvalue *big.Int, gasFeeCap *big.Int, gasLimit uint64, maxSubmissionFee *big.Int, feeRefundAddress common.Address, beneficiary common.Address, retryTo common.Address, retryData []byte) (*types.Transaction, error) {
	return _ArbRetryableTx.contract.Transact(opts, "submitRetryable", requestId, l1BaseFee, deposit, callvalue, gasFeeCap, gasLimit, maxSubmissionFee, feeRefundAddress, beneficiary, retryTo, retryData)
}

// SubmitRetryable is a paid mutator transaction binding the contract method 0xc9f95d32.
//
// Solidity: function submitRetryable(bytes32 requestId, uint256 l1BaseFee, uint256 deposit, uint256 callvalue, uint256 gasFeeCap, uint64 gasLimit, uint256 maxSubmissionFee, address feeRefundAddress, address beneficiary, address retryTo, bytes retryData) returns()
func (_ArbRetryableTx *ArbRetryableTxSession) SubmitRetryable(requestId [32]byte, l1BaseFee *big.Int, deposit *big.Int, callvalue *big.Int, gasFeeCap *big.Int, gasLimit uint64, maxSubmissionFee *big.Int, feeRefundAddress common.Address, beneficiary common.Address, retryTo common.Address, retryData []byte) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.SubmitRetryable(&_ArbRetryableTx.TransactOpts, requestId, l1BaseFee, deposit, callvalue, gasFeeCap, gasLimit, maxSubmissionFee, feeRefundAddress, beneficiary, retryTo, retryData)
}

// SubmitRetryable is a paid mutator transaction binding the contract method 0xc9f95d32.
//
// Solidity: function submitRetryable(bytes32 requestId, uint256 l1BaseFee, uint256 deposit, uint256 callvalue, uint256 gasFeeCap, uint64 gasLimit, uint256 maxSubmissionFee, address feeRefundAddress, address beneficiary, address retryTo, bytes retryData) returns()
func (_ArbRetryableTx *ArbRetryableTxTransactorSession) SubmitRetryable(requestId [32]byte, l1BaseFee *big.Int, deposit *big.Int, callvalue *big.Int, gasFeeCap *big.Int, gasLimit uint64, maxSubmissionFee *big.Int, feeRefundAddress common.Address, beneficiary common.Address, retryTo common.Address, retryData []byte) (*types.Transaction, error) {
	return _ArbRetryableTx.Contract.SubmitRetryable(&_ArbRetryableTx.TransactOpts, requestId, l1BaseFee, deposit, callvalue, gasFeeCap, gasLimit, maxSubmissionFee, feeRefundAddress, beneficiary, retryTo, retryData)
}

// ArbRetryableTxCanceledIterator is returned from FilterCanceled and is used to iterate over the raw logs and unpacked data for Canceled events raised by the ArbRetryableTx contract.
type ArbRetryableTxCanceledIterator struct {
	Event *ArbRetryableTxCanceled // Event containing the contract specifics and raw log

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
func (it *ArbRetryableTxCanceledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbRetryableTxCanceled)
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
		it.Event = new(ArbRetryableTxCanceled)
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
func (it *ArbRetryableTxCanceledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbRetryableTxCanceledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbRetryableTxCanceled represents a Canceled event raised by the ArbRetryableTx contract.
type ArbRetryableTxCanceled struct {
	TicketId [32]byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterCanceled is a free log retrieval operation binding the contract event 0x134fdd648feeaf30251f0157f9624ef8608ff9a042aad6d13e73f35d21d3f88d.
//
// Solidity: event Canceled(bytes32 indexed ticketId)
func (_ArbRetryableTx *ArbRetryableTxFilterer) FilterCanceled(opts *bind.FilterOpts, ticketId [][32]byte) (*ArbRetryableTxCanceledIterator, error) {

	var ticketIdRule []interface{}
	for _, ticketIdItem := range ticketId {
		ticketIdRule = append(ticketIdRule, ticketIdItem)
	}

	logs, sub, err := _ArbRetryableTx.contract.FilterLogs(opts, "Canceled", ticketIdRule)
	if err != nil {
		return nil, err
	}
	return &ArbRetryableTxCanceledIterator{contract: _ArbRetryableTx.contract, event: "Canceled", logs: logs, sub: sub}, nil
}

// WatchCanceled is a free log subscription operation binding the contract event 0x134fdd648feeaf30251f0157f9624ef8608ff9a042aad6d13e73f35d21d3f88d.
//
// Solidity: event Canceled(bytes32 indexed ticketId)
func (_ArbRetryableTx *ArbRetryableTxFilterer) WatchCanceled(opts *bind.WatchOpts, sink chan<- *ArbRetryableTxCanceled, ticketId [][32]byte) (event.Subscription, error) {

	var ticketIdRule []interface{}
	for _, ticketIdItem := range ticketId {
		ticketIdRule = append(ticketIdRule, ticketIdItem)
	}

	logs, sub, err := _ArbRetryableTx.contract.WatchLogs(opts, "Canceled", ticketIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbRetryableTxCanceled)
				if err := _ArbRetryableTx.contract.UnpackLog(event, "Canceled", log); err != nil {
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

// ParseCanceled is a log parse operation binding the contract event 0x134fdd648feeaf30251f0157f9624ef8608ff9a042aad6d13e73f35d21d3f88d.
//
// Solidity: event Canceled(bytes32 indexed ticketId)
func (_ArbRetryableTx *ArbRetryableTxFilterer) ParseCanceled(log types.Log) (*ArbRetryableTxCanceled, error) {
	event := new(ArbRetryableTxCanceled)
	if err := _ArbRetryableTx.contract.UnpackLog(event, "Canceled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbRetryableTxLifetimeExtendedIterator is returned from FilterLifetimeExtended and is used to iterate over the raw logs and unpacked data for LifetimeExtended events raised by the ArbRetryableTx contract.
type ArbRetryableTxLifetimeExtendedIterator struct {
	Event *ArbRetryableTxLifetimeExtended // Event containing the contract specifics and raw log

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
func (it *ArbRetryableTxLifetimeExtendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbRetryableTxLifetimeExtended)
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
		it.Event = new(ArbRetryableTxLifetimeExtended)
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
func (it *ArbRetryableTxLifetimeExtendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbRetryableTxLifetimeExtendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbRetryableTxLifetimeExtended represents a LifetimeExtended event raised by the ArbRetryableTx contract.
type ArbRetryableTxLifetimeExtended struct {
	TicketId   [32]byte
	NewTimeout *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterLifetimeExtended is a free log retrieval operation binding the contract event 0xf4c40a5f930e1469fcc053bf25f045253a7bad2fcc9b88c05ec1fca8e2066b83.
//
// Solidity: event LifetimeExtended(bytes32 indexed ticketId, uint256 newTimeout)
func (_ArbRetryableTx *ArbRetryableTxFilterer) FilterLifetimeExtended(opts *bind.FilterOpts, ticketId [][32]byte) (*ArbRetryableTxLifetimeExtendedIterator, error) {

	var ticketIdRule []interface{}
	for _, ticketIdItem := range ticketId {
		ticketIdRule = append(ticketIdRule, ticketIdItem)
	}

	logs, sub, err := _ArbRetryableTx.contract.FilterLogs(opts, "LifetimeExtended", ticketIdRule)
	if err != nil {
		return nil, err
	}
	return &ArbRetryableTxLifetimeExtendedIterator{contract: _ArbRetryableTx.contract, event: "LifetimeExtended", logs: logs, sub: sub}, nil
}

// WatchLifetimeExtended is a free log subscription operation binding the contract event 0xf4c40a5f930e1469fcc053bf25f045253a7bad2fcc9b88c05ec1fca8e2066b83.
//
// Solidity: event LifetimeExtended(bytes32 indexed ticketId, uint256 newTimeout)
func (_ArbRetryableTx *ArbRetryableTxFilterer) WatchLifetimeExtended(opts *bind.WatchOpts, sink chan<- *ArbRetryableTxLifetimeExtended, ticketId [][32]byte) (event.Subscription, error) {

	var ticketIdRule []interface{}
	for _, ticketIdItem := range ticketId {
		ticketIdRule = append(ticketIdRule, ticketIdItem)
	}

	logs, sub, err := _ArbRetryableTx.contract.WatchLogs(opts, "LifetimeExtended", ticketIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbRetryableTxLifetimeExtended)
				if err := _ArbRetryableTx.contract.UnpackLog(event, "LifetimeExtended", log); err != nil {
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

// ParseLifetimeExtended is a log parse operation binding the contract event 0xf4c40a5f930e1469fcc053bf25f045253a7bad2fcc9b88c05ec1fca8e2066b83.
//
// Solidity: event LifetimeExtended(bytes32 indexed ticketId, uint256 newTimeout)
func (_ArbRetryableTx *ArbRetryableTxFilterer) ParseLifetimeExtended(log types.Log) (*ArbRetryableTxLifetimeExtended, error) {
	event := new(ArbRetryableTxLifetimeExtended)
	if err := _ArbRetryableTx.contract.UnpackLog(event, "LifetimeExtended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbRetryableTxRedeemScheduledIterator is returned from FilterRedeemScheduled and is used to iterate over the raw logs and unpacked data for RedeemScheduled events raised by the ArbRetryableTx contract.
type ArbRetryableTxRedeemScheduledIterator struct {
	Event *ArbRetryableTxRedeemScheduled // Event containing the contract specifics and raw log

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
func (it *ArbRetryableTxRedeemScheduledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbRetryableTxRedeemScheduled)
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
		it.Event = new(ArbRetryableTxRedeemScheduled)
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
func (it *ArbRetryableTxRedeemScheduledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbRetryableTxRedeemScheduledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbRetryableTxRedeemScheduled represents a RedeemScheduled event raised by the ArbRetryableTx contract.
type ArbRetryableTxRedeemScheduled struct {
	TicketId            [32]byte
	RetryTxHash         [32]byte
	SequenceNum         uint64
	DonatedGas          uint64
	GasDonor            common.Address
	MaxRefund           *big.Int
	SubmissionFeeRefund *big.Int
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterRedeemScheduled is a free log retrieval operation binding the contract event 0x5ccd009502509cf28762c67858994d85b163bb6e451f5e9df7c5e18c9c2e123e.
//
// Solidity: event RedeemScheduled(bytes32 indexed ticketId, bytes32 indexed retryTxHash, uint64 indexed sequenceNum, uint64 donatedGas, address gasDonor, uint256 maxRefund, uint256 submissionFeeRefund)
func (_ArbRetryableTx *ArbRetryableTxFilterer) FilterRedeemScheduled(opts *bind.FilterOpts, ticketId [][32]byte, retryTxHash [][32]byte, sequenceNum []uint64) (*ArbRetryableTxRedeemScheduledIterator, error) {

	var ticketIdRule []interface{}
	for _, ticketIdItem := range ticketId {
		ticketIdRule = append(ticketIdRule, ticketIdItem)
	}
	var retryTxHashRule []interface{}
	for _, retryTxHashItem := range retryTxHash {
		retryTxHashRule = append(retryTxHashRule, retryTxHashItem)
	}
	var sequenceNumRule []interface{}
	for _, sequenceNumItem := range sequenceNum {
		sequenceNumRule = append(sequenceNumRule, sequenceNumItem)
	}

	logs, sub, err := _ArbRetryableTx.contract.FilterLogs(opts, "RedeemScheduled", ticketIdRule, retryTxHashRule, sequenceNumRule)
	if err != nil {
		return nil, err
	}
	return &ArbRetryableTxRedeemScheduledIterator{contract: _ArbRetryableTx.contract, event: "RedeemScheduled", logs: logs, sub: sub}, nil
}

// WatchRedeemScheduled is a free log subscription operation binding the contract event 0x5ccd009502509cf28762c67858994d85b163bb6e451f5e9df7c5e18c9c2e123e.
//
// Solidity: event RedeemScheduled(bytes32 indexed ticketId, bytes32 indexed retryTxHash, uint64 indexed sequenceNum, uint64 donatedGas, address gasDonor, uint256 maxRefund, uint256 submissionFeeRefund)
func (_ArbRetryableTx *ArbRetryableTxFilterer) WatchRedeemScheduled(opts *bind.WatchOpts, sink chan<- *ArbRetryableTxRedeemScheduled, ticketId [][32]byte, retryTxHash [][32]byte, sequenceNum []uint64) (event.Subscription, error) {

	var ticketIdRule []interface{}
	for _, ticketIdItem := range ticketId {
		ticketIdRule = append(ticketIdRule, ticketIdItem)
	}
	var retryTxHashRule []interface{}
	for _, retryTxHashItem := range retryTxHash {
		retryTxHashRule = append(retryTxHashRule, retryTxHashItem)
	}
	var sequenceNumRule []interface{}
	for _, sequenceNumItem := range sequenceNum {
		sequenceNumRule = append(sequenceNumRule, sequenceNumItem)
	}

	logs, sub, err := _ArbRetryableTx.contract.WatchLogs(opts, "RedeemScheduled", ticketIdRule, retryTxHashRule, sequenceNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbRetryableTxRedeemScheduled)
				if err := _ArbRetryableTx.contract.UnpackLog(event, "RedeemScheduled", log); err != nil {
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

// ParseRedeemScheduled is a log parse operation binding the contract event 0x5ccd009502509cf28762c67858994d85b163bb6e451f5e9df7c5e18c9c2e123e.
//
// Solidity: event RedeemScheduled(bytes32 indexed ticketId, bytes32 indexed retryTxHash, uint64 indexed sequenceNum, uint64 donatedGas, address gasDonor, uint256 maxRefund, uint256 submissionFeeRefund)
func (_ArbRetryableTx *ArbRetryableTxFilterer) ParseRedeemScheduled(log types.Log) (*ArbRetryableTxRedeemScheduled, error) {
	event := new(ArbRetryableTxRedeemScheduled)
	if err := _ArbRetryableTx.contract.UnpackLog(event, "RedeemScheduled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbRetryableTxRedeemedIterator is returned from FilterRedeemed and is used to iterate over the raw logs and unpacked data for Redeemed events raised by the ArbRetryableTx contract.
type ArbRetryableTxRedeemedIterator struct {
	Event *ArbRetryableTxRedeemed // Event containing the contract specifics and raw log

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
func (it *ArbRetryableTxRedeemedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbRetryableTxRedeemed)
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
		it.Event = new(ArbRetryableTxRedeemed)
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
func (it *ArbRetryableTxRedeemedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbRetryableTxRedeemedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbRetryableTxRedeemed represents a Redeemed event raised by the ArbRetryableTx contract.
type ArbRetryableTxRedeemed struct {
	UserTxHash [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterRedeemed is a free log retrieval operation binding the contract event 0x27fc6cca2a0e9eb6f4876c01fc7779b00cdeb7277a770ac2b844db5932449578.
//
// Solidity: event Redeemed(bytes32 indexed userTxHash)
func (_ArbRetryableTx *ArbRetryableTxFilterer) FilterRedeemed(opts *bind.FilterOpts, userTxHash [][32]byte) (*ArbRetryableTxRedeemedIterator, error) {

	var userTxHashRule []interface{}
	for _, userTxHashItem := range userTxHash {
		userTxHashRule = append(userTxHashRule, userTxHashItem)
	}

	logs, sub, err := _ArbRetryableTx.contract.FilterLogs(opts, "Redeemed", userTxHashRule)
	if err != nil {
		return nil, err
	}
	return &ArbRetryableTxRedeemedIterator{contract: _ArbRetryableTx.contract, event: "Redeemed", logs: logs, sub: sub}, nil
}

// WatchRedeemed is a free log subscription operation binding the contract event 0x27fc6cca2a0e9eb6f4876c01fc7779b00cdeb7277a770ac2b844db5932449578.
//
// Solidity: event Redeemed(bytes32 indexed userTxHash)
func (_ArbRetryableTx *ArbRetryableTxFilterer) WatchRedeemed(opts *bind.WatchOpts, sink chan<- *ArbRetryableTxRedeemed, userTxHash [][32]byte) (event.Subscription, error) {

	var userTxHashRule []interface{}
	for _, userTxHashItem := range userTxHash {
		userTxHashRule = append(userTxHashRule, userTxHashItem)
	}

	logs, sub, err := _ArbRetryableTx.contract.WatchLogs(opts, "Redeemed", userTxHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbRetryableTxRedeemed)
				if err := _ArbRetryableTx.contract.UnpackLog(event, "Redeemed", log); err != nil {
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

// ParseRedeemed is a log parse operation binding the contract event 0x27fc6cca2a0e9eb6f4876c01fc7779b00cdeb7277a770ac2b844db5932449578.
//
// Solidity: event Redeemed(bytes32 indexed userTxHash)
func (_ArbRetryableTx *ArbRetryableTxFilterer) ParseRedeemed(log types.Log) (*ArbRetryableTxRedeemed, error) {
	event := new(ArbRetryableTxRedeemed)
	if err := _ArbRetryableTx.contract.UnpackLog(event, "Redeemed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbRetryableTxTicketCreatedIterator is returned from FilterTicketCreated and is used to iterate over the raw logs and unpacked data for TicketCreated events raised by the ArbRetryableTx contract.
type ArbRetryableTxTicketCreatedIterator struct {
	Event *ArbRetryableTxTicketCreated // Event containing the contract specifics and raw log

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
func (it *ArbRetryableTxTicketCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbRetryableTxTicketCreated)
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
		it.Event = new(ArbRetryableTxTicketCreated)
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
func (it *ArbRetryableTxTicketCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbRetryableTxTicketCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbRetryableTxTicketCreated represents a TicketCreated event raised by the ArbRetryableTx contract.
type ArbRetryableTxTicketCreated struct {
	TicketId [32]byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterTicketCreated is a free log retrieval operation binding the contract event 0x7c793cced5743dc5f531bbe2bfb5a9fa3f40adef29231e6ab165c08a29e3dd89.
//
// Solidity: event TicketCreated(bytes32 indexed ticketId)
func (_ArbRetryableTx *ArbRetryableTxFilterer) FilterTicketCreated(opts *bind.FilterOpts, ticketId [][32]byte) (*ArbRetryableTxTicketCreatedIterator, error) {

	var ticketIdRule []interface{}
	for _, ticketIdItem := range ticketId {
		ticketIdRule = append(ticketIdRule, ticketIdItem)
	}

	logs, sub, err := _ArbRetryableTx.contract.FilterLogs(opts, "TicketCreated", ticketIdRule)
	if err != nil {
		return nil, err
	}
	return &ArbRetryableTxTicketCreatedIterator{contract: _ArbRetryableTx.contract, event: "TicketCreated", logs: logs, sub: sub}, nil
}

// WatchTicketCreated is a free log subscription operation binding the contract event 0x7c793cced5743dc5f531bbe2bfb5a9fa3f40adef29231e6ab165c08a29e3dd89.
//
// Solidity: event TicketCreated(bytes32 indexed ticketId)
func (_ArbRetryableTx *ArbRetryableTxFilterer) WatchTicketCreated(opts *bind.WatchOpts, sink chan<- *ArbRetryableTxTicketCreated, ticketId [][32]byte) (event.Subscription, error) {

	var ticketIdRule []interface{}
	for _, ticketIdItem := range ticketId {
		ticketIdRule = append(ticketIdRule, ticketIdItem)
	}

	logs, sub, err := _ArbRetryableTx.contract.WatchLogs(opts, "TicketCreated", ticketIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbRetryableTxTicketCreated)
				if err := _ArbRetryableTx.contract.UnpackLog(event, "TicketCreated", log); err != nil {
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

// ParseTicketCreated is a log parse operation binding the contract event 0x7c793cced5743dc5f531bbe2bfb5a9fa3f40adef29231e6ab165c08a29e3dd89.
//
// Solidity: event TicketCreated(bytes32 indexed ticketId)
func (_ArbRetryableTx *ArbRetryableTxFilterer) ParseTicketCreated(log types.Log) (*ArbRetryableTxTicketCreated, error) {
	event := new(ArbRetryableTxTicketCreated)
	if err := _ArbRetryableTx.contract.UnpackLog(event, "TicketCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbStatisticsMetaData contains all meta data concerning the ArbStatistics contract.
var ArbStatisticsMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"getStats\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// ArbStatisticsABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbStatisticsMetaData.ABI instead.
var ArbStatisticsABI = ArbStatisticsMetaData.ABI

// ArbStatistics is an auto generated Go binding around an Ethereum contract.
type ArbStatistics struct {
	ArbStatisticsCaller     // Read-only binding to the contract
	ArbStatisticsTransactor // Write-only binding to the contract
	ArbStatisticsFilterer   // Log filterer for contract events
}

// ArbStatisticsCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbStatisticsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbStatisticsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbStatisticsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbStatisticsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbStatisticsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbStatisticsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbStatisticsSession struct {
	Contract     *ArbStatistics    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbStatisticsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbStatisticsCallerSession struct {
	Contract *ArbStatisticsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ArbStatisticsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbStatisticsTransactorSession struct {
	Contract     *ArbStatisticsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ArbStatisticsRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbStatisticsRaw struct {
	Contract *ArbStatistics // Generic contract binding to access the raw methods on
}

// ArbStatisticsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbStatisticsCallerRaw struct {
	Contract *ArbStatisticsCaller // Generic read-only contract binding to access the raw methods on
}

// ArbStatisticsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbStatisticsTransactorRaw struct {
	Contract *ArbStatisticsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbStatistics creates a new instance of ArbStatistics, bound to a specific deployed contract.
func NewArbStatistics(address common.Address, backend bind.ContractBackend) (*ArbStatistics, error) {
	contract, err := bindArbStatistics(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbStatistics{ArbStatisticsCaller: ArbStatisticsCaller{contract: contract}, ArbStatisticsTransactor: ArbStatisticsTransactor{contract: contract}, ArbStatisticsFilterer: ArbStatisticsFilterer{contract: contract}}, nil
}

// NewArbStatisticsCaller creates a new read-only instance of ArbStatistics, bound to a specific deployed contract.
func NewArbStatisticsCaller(address common.Address, caller bind.ContractCaller) (*ArbStatisticsCaller, error) {
	contract, err := bindArbStatistics(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbStatisticsCaller{contract: contract}, nil
}

// NewArbStatisticsTransactor creates a new write-only instance of ArbStatistics, bound to a specific deployed contract.
func NewArbStatisticsTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbStatisticsTransactor, error) {
	contract, err := bindArbStatistics(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbStatisticsTransactor{contract: contract}, nil
}

// NewArbStatisticsFilterer creates a new log filterer instance of ArbStatistics, bound to a specific deployed contract.
func NewArbStatisticsFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbStatisticsFilterer, error) {
	contract, err := bindArbStatistics(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbStatisticsFilterer{contract: contract}, nil
}

// bindArbStatistics binds a generic wrapper to an already deployed contract.
func bindArbStatistics(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbStatisticsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbStatistics *ArbStatisticsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbStatistics.Contract.ArbStatisticsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbStatistics *ArbStatisticsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbStatistics.Contract.ArbStatisticsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbStatistics *ArbStatisticsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbStatistics.Contract.ArbStatisticsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbStatistics *ArbStatisticsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbStatistics.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbStatistics *ArbStatisticsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbStatistics.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbStatistics *ArbStatisticsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbStatistics.Contract.contract.Transact(opts, method, params...)
}

// GetStats is a free data retrieval call binding the contract method 0xc59d4847.
//
// Solidity: function getStats() view returns(uint256, uint256, uint256, uint256, uint256, uint256)
func (_ArbStatistics *ArbStatisticsCaller) GetStats(opts *bind.CallOpts) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	var out []interface{}
	err := _ArbStatistics.contract.Call(opts, &out, "getStats")

	if err != nil {
		return *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	out3 := *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	out4 := *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	out5 := *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	return out0, out1, out2, out3, out4, out5, err

}

// GetStats is a free data retrieval call binding the contract method 0xc59d4847.
//
// Solidity: function getStats() view returns(uint256, uint256, uint256, uint256, uint256, uint256)
func (_ArbStatistics *ArbStatisticsSession) GetStats() (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _ArbStatistics.Contract.GetStats(&_ArbStatistics.CallOpts)
}

// GetStats is a free data retrieval call binding the contract method 0xc59d4847.
//
// Solidity: function getStats() view returns(uint256, uint256, uint256, uint256, uint256, uint256)
func (_ArbStatistics *ArbStatisticsCallerSession) GetStats() (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _ArbStatistics.Contract.GetStats(&_ArbStatistics.CallOpts)
}

// ArbSysMetaData contains all meta data concerning the ArbSys contract.
var ArbSysMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"destination\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"uniqueId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"indexInBatch\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"arbBlockNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"ethBlockNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"callvalue\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"L2ToL1Transaction\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"destination\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"hash\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"position\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"arbBlockNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"ethBlockNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"callvalue\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"L2ToL1Tx\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"reserved\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"position\",\"type\":\"uint256\"}],\"name\":\"SendMerkleUpdate\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"arbBlockNum\",\"type\":\"uint256\"}],\"name\":\"arbBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"arbBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"arbChainID\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"arbOSVersion\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getStorageGasAvailable\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isTopLevelCall\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"unused\",\"type\":\"address\"}],\"name\":\"mapL1SenderContractAddressToL2Alias\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"myCallersAddressWithoutAliasing\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sendMerkleTreeState\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"size\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"partials\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"destination\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendTxToL1\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wasMyCallersAddressAliased\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"destination\",\"type\":\"address\"}],\"name\":\"withdrawEth\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
}

// ArbSysABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbSysMetaData.ABI instead.
var ArbSysABI = ArbSysMetaData.ABI

// ArbSys is an auto generated Go binding around an Ethereum contract.
type ArbSys struct {
	ArbSysCaller     // Read-only binding to the contract
	ArbSysTransactor // Write-only binding to the contract
	ArbSysFilterer   // Log filterer for contract events
}

// ArbSysCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbSysCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbSysTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbSysTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbSysFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbSysFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbSysSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbSysSession struct {
	Contract     *ArbSys           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbSysCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbSysCallerSession struct {
	Contract *ArbSysCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// ArbSysTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbSysTransactorSession struct {
	Contract     *ArbSysTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbSysRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbSysRaw struct {
	Contract *ArbSys // Generic contract binding to access the raw methods on
}

// ArbSysCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbSysCallerRaw struct {
	Contract *ArbSysCaller // Generic read-only contract binding to access the raw methods on
}

// ArbSysTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbSysTransactorRaw struct {
	Contract *ArbSysTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbSys creates a new instance of ArbSys, bound to a specific deployed contract.
func NewArbSys(address common.Address, backend bind.ContractBackend) (*ArbSys, error) {
	contract, err := bindArbSys(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbSys{ArbSysCaller: ArbSysCaller{contract: contract}, ArbSysTransactor: ArbSysTransactor{contract: contract}, ArbSysFilterer: ArbSysFilterer{contract: contract}}, nil
}

// NewArbSysCaller creates a new read-only instance of ArbSys, bound to a specific deployed contract.
func NewArbSysCaller(address common.Address, caller bind.ContractCaller) (*ArbSysCaller, error) {
	contract, err := bindArbSys(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbSysCaller{contract: contract}, nil
}

// NewArbSysTransactor creates a new write-only instance of ArbSys, bound to a specific deployed contract.
func NewArbSysTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbSysTransactor, error) {
	contract, err := bindArbSys(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbSysTransactor{contract: contract}, nil
}

// NewArbSysFilterer creates a new log filterer instance of ArbSys, bound to a specific deployed contract.
func NewArbSysFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbSysFilterer, error) {
	contract, err := bindArbSys(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbSysFilterer{contract: contract}, nil
}

// bindArbSys binds a generic wrapper to an already deployed contract.
func bindArbSys(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbSysMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbSys *ArbSysRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbSys.Contract.ArbSysCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbSys *ArbSysRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbSys.Contract.ArbSysTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbSys *ArbSysRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbSys.Contract.ArbSysTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbSys *ArbSysCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbSys.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbSys *ArbSysTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbSys.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbSys *ArbSysTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbSys.Contract.contract.Transact(opts, method, params...)
}

// ArbBlockHash is a free data retrieval call binding the contract method 0x2b407a82.
//
// Solidity: function arbBlockHash(uint256 arbBlockNum) view returns(bytes32)
func (_ArbSys *ArbSysCaller) ArbBlockHash(opts *bind.CallOpts, arbBlockNum *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _ArbSys.contract.Call(opts, &out, "arbBlockHash", arbBlockNum)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ArbBlockHash is a free data retrieval call binding the contract method 0x2b407a82.
//
// Solidity: function arbBlockHash(uint256 arbBlockNum) view returns(bytes32)
func (_ArbSys *ArbSysSession) ArbBlockHash(arbBlockNum *big.Int) ([32]byte, error) {
	return _ArbSys.Contract.ArbBlockHash(&_ArbSys.CallOpts, arbBlockNum)
}

// ArbBlockHash is a free data retrieval call binding the contract method 0x2b407a82.
//
// Solidity: function arbBlockHash(uint256 arbBlockNum) view returns(bytes32)
func (_ArbSys *ArbSysCallerSession) ArbBlockHash(arbBlockNum *big.Int) ([32]byte, error) {
	return _ArbSys.Contract.ArbBlockHash(&_ArbSys.CallOpts, arbBlockNum)
}

// ArbBlockNumber is a free data retrieval call binding the contract method 0xa3b1b31d.
//
// Solidity: function arbBlockNumber() view returns(uint256)
func (_ArbSys *ArbSysCaller) ArbBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbSys.contract.Call(opts, &out, "arbBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ArbBlockNumber is a free data retrieval call binding the contract method 0xa3b1b31d.
//
// Solidity: function arbBlockNumber() view returns(uint256)
func (_ArbSys *ArbSysSession) ArbBlockNumber() (*big.Int, error) {
	return _ArbSys.Contract.ArbBlockNumber(&_ArbSys.CallOpts)
}

// ArbBlockNumber is a free data retrieval call binding the contract method 0xa3b1b31d.
//
// Solidity: function arbBlockNumber() view returns(uint256)
func (_ArbSys *ArbSysCallerSession) ArbBlockNumber() (*big.Int, error) {
	return _ArbSys.Contract.ArbBlockNumber(&_ArbSys.CallOpts)
}

// ArbChainID is a free data retrieval call binding the contract method 0xd127f54a.
//
// Solidity: function arbChainID() view returns(uint256)
func (_ArbSys *ArbSysCaller) ArbChainID(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbSys.contract.Call(opts, &out, "arbChainID")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ArbChainID is a free data retrieval call binding the contract method 0xd127f54a.
//
// Solidity: function arbChainID() view returns(uint256)
func (_ArbSys *ArbSysSession) ArbChainID() (*big.Int, error) {
	return _ArbSys.Contract.ArbChainID(&_ArbSys.CallOpts)
}

// ArbChainID is a free data retrieval call binding the contract method 0xd127f54a.
//
// Solidity: function arbChainID() view returns(uint256)
func (_ArbSys *ArbSysCallerSession) ArbChainID() (*big.Int, error) {
	return _ArbSys.Contract.ArbChainID(&_ArbSys.CallOpts)
}

// ArbOSVersion is a free data retrieval call binding the contract method 0x051038f2.
//
// Solidity: function arbOSVersion() view returns(uint256)
func (_ArbSys *ArbSysCaller) ArbOSVersion(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbSys.contract.Call(opts, &out, "arbOSVersion")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ArbOSVersion is a free data retrieval call binding the contract method 0x051038f2.
//
// Solidity: function arbOSVersion() view returns(uint256)
func (_ArbSys *ArbSysSession) ArbOSVersion() (*big.Int, error) {
	return _ArbSys.Contract.ArbOSVersion(&_ArbSys.CallOpts)
}

// ArbOSVersion is a free data retrieval call binding the contract method 0x051038f2.
//
// Solidity: function arbOSVersion() view returns(uint256)
func (_ArbSys *ArbSysCallerSession) ArbOSVersion() (*big.Int, error) {
	return _ArbSys.Contract.ArbOSVersion(&_ArbSys.CallOpts)
}

// GetStorageGasAvailable is a free data retrieval call binding the contract method 0xa94597ff.
//
// Solidity: function getStorageGasAvailable() view returns(uint256)
func (_ArbSys *ArbSysCaller) GetStorageGasAvailable(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ArbSys.contract.Call(opts, &out, "getStorageGasAvailable")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetStorageGasAvailable is a free data retrieval call binding the contract method 0xa94597ff.
//
// Solidity: function getStorageGasAvailable() view returns(uint256)
func (_ArbSys *ArbSysSession) GetStorageGasAvailable() (*big.Int, error) {
	return _ArbSys.Contract.GetStorageGasAvailable(&_ArbSys.CallOpts)
}

// GetStorageGasAvailable is a free data retrieval call binding the contract method 0xa94597ff.
//
// Solidity: function getStorageGasAvailable() view returns(uint256)
func (_ArbSys *ArbSysCallerSession) GetStorageGasAvailable() (*big.Int, error) {
	return _ArbSys.Contract.GetStorageGasAvailable(&_ArbSys.CallOpts)
}

// IsTopLevelCall is a free data retrieval call binding the contract method 0x08bd624c.
//
// Solidity: function isTopLevelCall() view returns(bool)
func (_ArbSys *ArbSysCaller) IsTopLevelCall(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _ArbSys.contract.Call(opts, &out, "isTopLevelCall")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsTopLevelCall is a free data retrieval call binding the contract method 0x08bd624c.
//
// Solidity: function isTopLevelCall() view returns(bool)
func (_ArbSys *ArbSysSession) IsTopLevelCall() (bool, error) {
	return _ArbSys.Contract.IsTopLevelCall(&_ArbSys.CallOpts)
}

// IsTopLevelCall is a free data retrieval call binding the contract method 0x08bd624c.
//
// Solidity: function isTopLevelCall() view returns(bool)
func (_ArbSys *ArbSysCallerSession) IsTopLevelCall() (bool, error) {
	return _ArbSys.Contract.IsTopLevelCall(&_ArbSys.CallOpts)
}

// MapL1SenderContractAddressToL2Alias is a free data retrieval call binding the contract method 0x4dbbd506.
//
// Solidity: function mapL1SenderContractAddressToL2Alias(address sender, address unused) pure returns(address)
func (_ArbSys *ArbSysCaller) MapL1SenderContractAddressToL2Alias(opts *bind.CallOpts, sender common.Address, unused common.Address) (common.Address, error) {
	var out []interface{}
	err := _ArbSys.contract.Call(opts, &out, "mapL1SenderContractAddressToL2Alias", sender, unused)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MapL1SenderContractAddressToL2Alias is a free data retrieval call binding the contract method 0x4dbbd506.
//
// Solidity: function mapL1SenderContractAddressToL2Alias(address sender, address unused) pure returns(address)
func (_ArbSys *ArbSysSession) MapL1SenderContractAddressToL2Alias(sender common.Address, unused common.Address) (common.Address, error) {
	return _ArbSys.Contract.MapL1SenderContractAddressToL2Alias(&_ArbSys.CallOpts, sender, unused)
}

// MapL1SenderContractAddressToL2Alias is a free data retrieval call binding the contract method 0x4dbbd506.
//
// Solidity: function mapL1SenderContractAddressToL2Alias(address sender, address unused) pure returns(address)
func (_ArbSys *ArbSysCallerSession) MapL1SenderContractAddressToL2Alias(sender common.Address, unused common.Address) (common.Address, error) {
	return _ArbSys.Contract.MapL1SenderContractAddressToL2Alias(&_ArbSys.CallOpts, sender, unused)
}

// MyCallersAddressWithoutAliasing is a free data retrieval call binding the contract method 0xd74523b3.
//
// Solidity: function myCallersAddressWithoutAliasing() view returns(address)
func (_ArbSys *ArbSysCaller) MyCallersAddressWithoutAliasing(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ArbSys.contract.Call(opts, &out, "myCallersAddressWithoutAliasing")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MyCallersAddressWithoutAliasing is a free data retrieval call binding the contract method 0xd74523b3.
//
// Solidity: function myCallersAddressWithoutAliasing() view returns(address)
func (_ArbSys *ArbSysSession) MyCallersAddressWithoutAliasing() (common.Address, error) {
	return _ArbSys.Contract.MyCallersAddressWithoutAliasing(&_ArbSys.CallOpts)
}

// MyCallersAddressWithoutAliasing is a free data retrieval call binding the contract method 0xd74523b3.
//
// Solidity: function myCallersAddressWithoutAliasing() view returns(address)
func (_ArbSys *ArbSysCallerSession) MyCallersAddressWithoutAliasing() (common.Address, error) {
	return _ArbSys.Contract.MyCallersAddressWithoutAliasing(&_ArbSys.CallOpts)
}

// SendMerkleTreeState is a free data retrieval call binding the contract method 0x7aeecd2a.
//
// Solidity: function sendMerkleTreeState() view returns(uint256 size, bytes32 root, bytes32[] partials)
func (_ArbSys *ArbSysCaller) SendMerkleTreeState(opts *bind.CallOpts) (struct {
	Size     *big.Int
	Root     [32]byte
	Partials [][32]byte
}, error) {
	var out []interface{}
	err := _ArbSys.contract.Call(opts, &out, "sendMerkleTreeState")

	outstruct := new(struct {
		Size     *big.Int
		Root     [32]byte
		Partials [][32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Size = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Root = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.Partials = *abi.ConvertType(out[2], new([][32]byte)).(*[][32]byte)

	return *outstruct, err

}

// SendMerkleTreeState is a free data retrieval call binding the contract method 0x7aeecd2a.
//
// Solidity: function sendMerkleTreeState() view returns(uint256 size, bytes32 root, bytes32[] partials)
func (_ArbSys *ArbSysSession) SendMerkleTreeState() (struct {
	Size     *big.Int
	Root     [32]byte
	Partials [][32]byte
}, error) {
	return _ArbSys.Contract.SendMerkleTreeState(&_ArbSys.CallOpts)
}

// SendMerkleTreeState is a free data retrieval call binding the contract method 0x7aeecd2a.
//
// Solidity: function sendMerkleTreeState() view returns(uint256 size, bytes32 root, bytes32[] partials)
func (_ArbSys *ArbSysCallerSession) SendMerkleTreeState() (struct {
	Size     *big.Int
	Root     [32]byte
	Partials [][32]byte
}, error) {
	return _ArbSys.Contract.SendMerkleTreeState(&_ArbSys.CallOpts)
}

// WasMyCallersAddressAliased is a free data retrieval call binding the contract method 0x175a260b.
//
// Solidity: function wasMyCallersAddressAliased() view returns(bool)
func (_ArbSys *ArbSysCaller) WasMyCallersAddressAliased(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _ArbSys.contract.Call(opts, &out, "wasMyCallersAddressAliased")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// WasMyCallersAddressAliased is a free data retrieval call binding the contract method 0x175a260b.
//
// Solidity: function wasMyCallersAddressAliased() view returns(bool)
func (_ArbSys *ArbSysSession) WasMyCallersAddressAliased() (bool, error) {
	return _ArbSys.Contract.WasMyCallersAddressAliased(&_ArbSys.CallOpts)
}

// WasMyCallersAddressAliased is a free data retrieval call binding the contract method 0x175a260b.
//
// Solidity: function wasMyCallersAddressAliased() view returns(bool)
func (_ArbSys *ArbSysCallerSession) WasMyCallersAddressAliased() (bool, error) {
	return _ArbSys.Contract.WasMyCallersAddressAliased(&_ArbSys.CallOpts)
}

// SendTxToL1 is a paid mutator transaction binding the contract method 0x928c169a.
//
// Solidity: function sendTxToL1(address destination, bytes data) payable returns(uint256)
func (_ArbSys *ArbSysTransactor) SendTxToL1(opts *bind.TransactOpts, destination common.Address, data []byte) (*types.Transaction, error) {
	return _ArbSys.contract.Transact(opts, "sendTxToL1", destination, data)
}

// SendTxToL1 is a paid mutator transaction binding the contract method 0x928c169a.
//
// Solidity: function sendTxToL1(address destination, bytes data) payable returns(uint256)
func (_ArbSys *ArbSysSession) SendTxToL1(destination common.Address, data []byte) (*types.Transaction, error) {
	return _ArbSys.Contract.SendTxToL1(&_ArbSys.TransactOpts, destination, data)
}

// SendTxToL1 is a paid mutator transaction binding the contract method 0x928c169a.
//
// Solidity: function sendTxToL1(address destination, bytes data) payable returns(uint256)
func (_ArbSys *ArbSysTransactorSession) SendTxToL1(destination common.Address, data []byte) (*types.Transaction, error) {
	return _ArbSys.Contract.SendTxToL1(&_ArbSys.TransactOpts, destination, data)
}

// WithdrawEth is a paid mutator transaction binding the contract method 0x25e16063.
//
// Solidity: function withdrawEth(address destination) payable returns(uint256)
func (_ArbSys *ArbSysTransactor) WithdrawEth(opts *bind.TransactOpts, destination common.Address) (*types.Transaction, error) {
	return _ArbSys.contract.Transact(opts, "withdrawEth", destination)
}

// WithdrawEth is a paid mutator transaction binding the contract method 0x25e16063.
//
// Solidity: function withdrawEth(address destination) payable returns(uint256)
func (_ArbSys *ArbSysSession) WithdrawEth(destination common.Address) (*types.Transaction, error) {
	return _ArbSys.Contract.WithdrawEth(&_ArbSys.TransactOpts, destination)
}

// WithdrawEth is a paid mutator transaction binding the contract method 0x25e16063.
//
// Solidity: function withdrawEth(address destination) payable returns(uint256)
func (_ArbSys *ArbSysTransactorSession) WithdrawEth(destination common.Address) (*types.Transaction, error) {
	return _ArbSys.Contract.WithdrawEth(&_ArbSys.TransactOpts, destination)
}

// ArbSysL2ToL1TransactionIterator is returned from FilterL2ToL1Transaction and is used to iterate over the raw logs and unpacked data for L2ToL1Transaction events raised by the ArbSys contract.
type ArbSysL2ToL1TransactionIterator struct {
	Event *ArbSysL2ToL1Transaction // Event containing the contract specifics and raw log

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
func (it *ArbSysL2ToL1TransactionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbSysL2ToL1Transaction)
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
		it.Event = new(ArbSysL2ToL1Transaction)
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
func (it *ArbSysL2ToL1TransactionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbSysL2ToL1TransactionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbSysL2ToL1Transaction represents a L2ToL1Transaction event raised by the ArbSys contract.
type ArbSysL2ToL1Transaction struct {
	Caller       common.Address
	Destination  common.Address
	UniqueId     *big.Int
	BatchNumber  *big.Int
	IndexInBatch *big.Int
	ArbBlockNum  *big.Int
	EthBlockNum  *big.Int
	Timestamp    *big.Int
	Callvalue    *big.Int
	Data         []byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterL2ToL1Transaction is a free log retrieval operation binding the contract event 0x5baaa87db386365b5c161be377bc3d8e317e8d98d71a3ca7ed7d555340c8f767.
//
// Solidity: event L2ToL1Transaction(address caller, address indexed destination, uint256 indexed uniqueId, uint256 indexed batchNumber, uint256 indexInBatch, uint256 arbBlockNum, uint256 ethBlockNum, uint256 timestamp, uint256 callvalue, bytes data)
func (_ArbSys *ArbSysFilterer) FilterL2ToL1Transaction(opts *bind.FilterOpts, destination []common.Address, uniqueId []*big.Int, batchNumber []*big.Int) (*ArbSysL2ToL1TransactionIterator, error) {

	var destinationRule []interface{}
	for _, destinationItem := range destination {
		destinationRule = append(destinationRule, destinationItem)
	}
	var uniqueIdRule []interface{}
	for _, uniqueIdItem := range uniqueId {
		uniqueIdRule = append(uniqueIdRule, uniqueIdItem)
	}
	var batchNumberRule []interface{}
	for _, batchNumberItem := range batchNumber {
		batchNumberRule = append(batchNumberRule, batchNumberItem)
	}

	logs, sub, err := _ArbSys.contract.FilterLogs(opts, "L2ToL1Transaction", destinationRule, uniqueIdRule, batchNumberRule)
	if err != nil {
		return nil, err
	}
	return &ArbSysL2ToL1TransactionIterator{contract: _ArbSys.contract, event: "L2ToL1Transaction", logs: logs, sub: sub}, nil
}

// WatchL2ToL1Transaction is a free log subscription operation binding the contract event 0x5baaa87db386365b5c161be377bc3d8e317e8d98d71a3ca7ed7d555340c8f767.
//
// Solidity: event L2ToL1Transaction(address caller, address indexed destination, uint256 indexed uniqueId, uint256 indexed batchNumber, uint256 indexInBatch, uint256 arbBlockNum, uint256 ethBlockNum, uint256 timestamp, uint256 callvalue, bytes data)
func (_ArbSys *ArbSysFilterer) WatchL2ToL1Transaction(opts *bind.WatchOpts, sink chan<- *ArbSysL2ToL1Transaction, destination []common.Address, uniqueId []*big.Int, batchNumber []*big.Int) (event.Subscription, error) {

	var destinationRule []interface{}
	for _, destinationItem := range destination {
		destinationRule = append(destinationRule, destinationItem)
	}
	var uniqueIdRule []interface{}
	for _, uniqueIdItem := range uniqueId {
		uniqueIdRule = append(uniqueIdRule, uniqueIdItem)
	}
	var batchNumberRule []interface{}
	for _, batchNumberItem := range batchNumber {
		batchNumberRule = append(batchNumberRule, batchNumberItem)
	}

	logs, sub, err := _ArbSys.contract.WatchLogs(opts, "L2ToL1Transaction", destinationRule, uniqueIdRule, batchNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbSysL2ToL1Transaction)
				if err := _ArbSys.contract.UnpackLog(event, "L2ToL1Transaction", log); err != nil {
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

// ParseL2ToL1Transaction is a log parse operation binding the contract event 0x5baaa87db386365b5c161be377bc3d8e317e8d98d71a3ca7ed7d555340c8f767.
//
// Solidity: event L2ToL1Transaction(address caller, address indexed destination, uint256 indexed uniqueId, uint256 indexed batchNumber, uint256 indexInBatch, uint256 arbBlockNum, uint256 ethBlockNum, uint256 timestamp, uint256 callvalue, bytes data)
func (_ArbSys *ArbSysFilterer) ParseL2ToL1Transaction(log types.Log) (*ArbSysL2ToL1Transaction, error) {
	event := new(ArbSysL2ToL1Transaction)
	if err := _ArbSys.contract.UnpackLog(event, "L2ToL1Transaction", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbSysL2ToL1TxIterator is returned from FilterL2ToL1Tx and is used to iterate over the raw logs and unpacked data for L2ToL1Tx events raised by the ArbSys contract.
type ArbSysL2ToL1TxIterator struct {
	Event *ArbSysL2ToL1Tx // Event containing the contract specifics and raw log

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
func (it *ArbSysL2ToL1TxIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbSysL2ToL1Tx)
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
		it.Event = new(ArbSysL2ToL1Tx)
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
func (it *ArbSysL2ToL1TxIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbSysL2ToL1TxIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbSysL2ToL1Tx represents a L2ToL1Tx event raised by the ArbSys contract.
type ArbSysL2ToL1Tx struct {
	Caller      common.Address
	Destination common.Address
	Hash        *big.Int
	Position    *big.Int
	ArbBlockNum *big.Int
	EthBlockNum *big.Int
	Timestamp   *big.Int
	Callvalue   *big.Int
	Data        []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterL2ToL1Tx is a free log retrieval operation binding the contract event 0x3e7aafa77dbf186b7fd488006beff893744caa3c4f6f299e8a709fa2087374fc.
//
// Solidity: event L2ToL1Tx(address caller, address indexed destination, uint256 indexed hash, uint256 indexed position, uint256 arbBlockNum, uint256 ethBlockNum, uint256 timestamp, uint256 callvalue, bytes data)
func (_ArbSys *ArbSysFilterer) FilterL2ToL1Tx(opts *bind.FilterOpts, destination []common.Address, hash []*big.Int, position []*big.Int) (*ArbSysL2ToL1TxIterator, error) {

	var destinationRule []interface{}
	for _, destinationItem := range destination {
		destinationRule = append(destinationRule, destinationItem)
	}
	var hashRule []interface{}
	for _, hashItem := range hash {
		hashRule = append(hashRule, hashItem)
	}
	var positionRule []interface{}
	for _, positionItem := range position {
		positionRule = append(positionRule, positionItem)
	}

	logs, sub, err := _ArbSys.contract.FilterLogs(opts, "L2ToL1Tx", destinationRule, hashRule, positionRule)
	if err != nil {
		return nil, err
	}
	return &ArbSysL2ToL1TxIterator{contract: _ArbSys.contract, event: "L2ToL1Tx", logs: logs, sub: sub}, nil
}

// WatchL2ToL1Tx is a free log subscription operation binding the contract event 0x3e7aafa77dbf186b7fd488006beff893744caa3c4f6f299e8a709fa2087374fc.
//
// Solidity: event L2ToL1Tx(address caller, address indexed destination, uint256 indexed hash, uint256 indexed position, uint256 arbBlockNum, uint256 ethBlockNum, uint256 timestamp, uint256 callvalue, bytes data)
func (_ArbSys *ArbSysFilterer) WatchL2ToL1Tx(opts *bind.WatchOpts, sink chan<- *ArbSysL2ToL1Tx, destination []common.Address, hash []*big.Int, position []*big.Int) (event.Subscription, error) {

	var destinationRule []interface{}
	for _, destinationItem := range destination {
		destinationRule = append(destinationRule, destinationItem)
	}
	var hashRule []interface{}
	for _, hashItem := range hash {
		hashRule = append(hashRule, hashItem)
	}
	var positionRule []interface{}
	for _, positionItem := range position {
		positionRule = append(positionRule, positionItem)
	}

	logs, sub, err := _ArbSys.contract.WatchLogs(opts, "L2ToL1Tx", destinationRule, hashRule, positionRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbSysL2ToL1Tx)
				if err := _ArbSys.contract.UnpackLog(event, "L2ToL1Tx", log); err != nil {
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

// ParseL2ToL1Tx is a log parse operation binding the contract event 0x3e7aafa77dbf186b7fd488006beff893744caa3c4f6f299e8a709fa2087374fc.
//
// Solidity: event L2ToL1Tx(address caller, address indexed destination, uint256 indexed hash, uint256 indexed position, uint256 arbBlockNum, uint256 ethBlockNum, uint256 timestamp, uint256 callvalue, bytes data)
func (_ArbSys *ArbSysFilterer) ParseL2ToL1Tx(log types.Log) (*ArbSysL2ToL1Tx, error) {
	event := new(ArbSysL2ToL1Tx)
	if err := _ArbSys.contract.UnpackLog(event, "L2ToL1Tx", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbSysSendMerkleUpdateIterator is returned from FilterSendMerkleUpdate and is used to iterate over the raw logs and unpacked data for SendMerkleUpdate events raised by the ArbSys contract.
type ArbSysSendMerkleUpdateIterator struct {
	Event *ArbSysSendMerkleUpdate // Event containing the contract specifics and raw log

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
func (it *ArbSysSendMerkleUpdateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ArbSysSendMerkleUpdate)
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
		it.Event = new(ArbSysSendMerkleUpdate)
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
func (it *ArbSysSendMerkleUpdateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ArbSysSendMerkleUpdateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ArbSysSendMerkleUpdate represents a SendMerkleUpdate event raised by the ArbSys contract.
type ArbSysSendMerkleUpdate struct {
	Reserved *big.Int
	Hash     [32]byte
	Position *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterSendMerkleUpdate is a free log retrieval operation binding the contract event 0xe9e13da364699fb5b0496ff5a0fc70760ad5836e93ba96568a4e42b9914a8b95.
//
// Solidity: event SendMerkleUpdate(uint256 indexed reserved, bytes32 indexed hash, uint256 indexed position)
func (_ArbSys *ArbSysFilterer) FilterSendMerkleUpdate(opts *bind.FilterOpts, reserved []*big.Int, hash [][32]byte, position []*big.Int) (*ArbSysSendMerkleUpdateIterator, error) {

	var reservedRule []interface{}
	for _, reservedItem := range reserved {
		reservedRule = append(reservedRule, reservedItem)
	}
	var hashRule []interface{}
	for _, hashItem := range hash {
		hashRule = append(hashRule, hashItem)
	}
	var positionRule []interface{}
	for _, positionItem := range position {
		positionRule = append(positionRule, positionItem)
	}

	logs, sub, err := _ArbSys.contract.FilterLogs(opts, "SendMerkleUpdate", reservedRule, hashRule, positionRule)
	if err != nil {
		return nil, err
	}
	return &ArbSysSendMerkleUpdateIterator{contract: _ArbSys.contract, event: "SendMerkleUpdate", logs: logs, sub: sub}, nil
}

// WatchSendMerkleUpdate is a free log subscription operation binding the contract event 0xe9e13da364699fb5b0496ff5a0fc70760ad5836e93ba96568a4e42b9914a8b95.
//
// Solidity: event SendMerkleUpdate(uint256 indexed reserved, bytes32 indexed hash, uint256 indexed position)
func (_ArbSys *ArbSysFilterer) WatchSendMerkleUpdate(opts *bind.WatchOpts, sink chan<- *ArbSysSendMerkleUpdate, reserved []*big.Int, hash [][32]byte, position []*big.Int) (event.Subscription, error) {

	var reservedRule []interface{}
	for _, reservedItem := range reserved {
		reservedRule = append(reservedRule, reservedItem)
	}
	var hashRule []interface{}
	for _, hashItem := range hash {
		hashRule = append(hashRule, hashItem)
	}
	var positionRule []interface{}
	for _, positionItem := range position {
		positionRule = append(positionRule, positionItem)
	}

	logs, sub, err := _ArbSys.contract.WatchLogs(opts, "SendMerkleUpdate", reservedRule, hashRule, positionRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ArbSysSendMerkleUpdate)
				if err := _ArbSys.contract.UnpackLog(event, "SendMerkleUpdate", log); err != nil {
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

// ParseSendMerkleUpdate is a log parse operation binding the contract event 0xe9e13da364699fb5b0496ff5a0fc70760ad5836e93ba96568a4e42b9914a8b95.
//
// Solidity: event SendMerkleUpdate(uint256 indexed reserved, bytes32 indexed hash, uint256 indexed position)
func (_ArbSys *ArbSysFilterer) ParseSendMerkleUpdate(log types.Log) (*ArbSysSendMerkleUpdate, error) {
	event := new(ArbSysSendMerkleUpdate)
	if err := _ArbSys.contract.UnpackLog(event, "SendMerkleUpdate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ArbosActsMetaData contains all meta data concerning the ArbosActs contract.
var ArbosActsMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"CallerNotArbOS\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"batchTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"batchPosterAddress\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"batchNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"batchDataGas\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"l1BaseFeeWei\",\"type\":\"uint256\"}],\"name\":\"batchPostingReport\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"l1BaseFee\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"l1BlockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"l2BlockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timePassed\",\"type\":\"uint64\"}],\"name\":\"startBlock\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ArbosActsABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbosActsMetaData.ABI instead.
var ArbosActsABI = ArbosActsMetaData.ABI

// ArbosActs is an auto generated Go binding around an Ethereum contract.
type ArbosActs struct {
	ArbosActsCaller     // Read-only binding to the contract
	ArbosActsTransactor // Write-only binding to the contract
	ArbosActsFilterer   // Log filterer for contract events
}

// ArbosActsCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbosActsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbosActsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbosActsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbosActsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbosActsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbosActsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbosActsSession struct {
	Contract     *ArbosActs        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbosActsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbosActsCallerSession struct {
	Contract *ArbosActsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// ArbosActsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbosActsTransactorSession struct {
	Contract     *ArbosActsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ArbosActsRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbosActsRaw struct {
	Contract *ArbosActs // Generic contract binding to access the raw methods on
}

// ArbosActsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbosActsCallerRaw struct {
	Contract *ArbosActsCaller // Generic read-only contract binding to access the raw methods on
}

// ArbosActsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbosActsTransactorRaw struct {
	Contract *ArbosActsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbosActs creates a new instance of ArbosActs, bound to a specific deployed contract.
func NewArbosActs(address common.Address, backend bind.ContractBackend) (*ArbosActs, error) {
	contract, err := bindArbosActs(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbosActs{ArbosActsCaller: ArbosActsCaller{contract: contract}, ArbosActsTransactor: ArbosActsTransactor{contract: contract}, ArbosActsFilterer: ArbosActsFilterer{contract: contract}}, nil
}

// NewArbosActsCaller creates a new read-only instance of ArbosActs, bound to a specific deployed contract.
func NewArbosActsCaller(address common.Address, caller bind.ContractCaller) (*ArbosActsCaller, error) {
	contract, err := bindArbosActs(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbosActsCaller{contract: contract}, nil
}

// NewArbosActsTransactor creates a new write-only instance of ArbosActs, bound to a specific deployed contract.
func NewArbosActsTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbosActsTransactor, error) {
	contract, err := bindArbosActs(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbosActsTransactor{contract: contract}, nil
}

// NewArbosActsFilterer creates a new log filterer instance of ArbosActs, bound to a specific deployed contract.
func NewArbosActsFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbosActsFilterer, error) {
	contract, err := bindArbosActs(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbosActsFilterer{contract: contract}, nil
}

// bindArbosActs binds a generic wrapper to an already deployed contract.
func bindArbosActs(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbosActsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbosActs *ArbosActsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbosActs.Contract.ArbosActsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbosActs *ArbosActsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbosActs.Contract.ArbosActsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbosActs *ArbosActsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbosActs.Contract.ArbosActsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbosActs *ArbosActsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbosActs.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbosActs *ArbosActsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbosActs.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbosActs *ArbosActsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbosActs.Contract.contract.Transact(opts, method, params...)
}

// BatchPostingReport is a paid mutator transaction binding the contract method 0xb6693771.
//
// Solidity: function batchPostingReport(uint256 batchTimestamp, address batchPosterAddress, uint64 batchNumber, uint64 batchDataGas, uint256 l1BaseFeeWei) returns()
func (_ArbosActs *ArbosActsTransactor) BatchPostingReport(opts *bind.TransactOpts, batchTimestamp *big.Int, batchPosterAddress common.Address, batchNumber uint64, batchDataGas uint64, l1BaseFeeWei *big.Int) (*types.Transaction, error) {
	return _ArbosActs.contract.Transact(opts, "batchPostingReport", batchTimestamp, batchPosterAddress, batchNumber, batchDataGas, l1BaseFeeWei)
}

// BatchPostingReport is a paid mutator transaction binding the contract method 0xb6693771.
//
// Solidity: function batchPostingReport(uint256 batchTimestamp, address batchPosterAddress, uint64 batchNumber, uint64 batchDataGas, uint256 l1BaseFeeWei) returns()
func (_ArbosActs *ArbosActsSession) BatchPostingReport(batchTimestamp *big.Int, batchPosterAddress common.Address, batchNumber uint64, batchDataGas uint64, l1BaseFeeWei *big.Int) (*types.Transaction, error) {
	return _ArbosActs.Contract.BatchPostingReport(&_ArbosActs.TransactOpts, batchTimestamp, batchPosterAddress, batchNumber, batchDataGas, l1BaseFeeWei)
}

// BatchPostingReport is a paid mutator transaction binding the contract method 0xb6693771.
//
// Solidity: function batchPostingReport(uint256 batchTimestamp, address batchPosterAddress, uint64 batchNumber, uint64 batchDataGas, uint256 l1BaseFeeWei) returns()
func (_ArbosActs *ArbosActsTransactorSession) BatchPostingReport(batchTimestamp *big.Int, batchPosterAddress common.Address, batchNumber uint64, batchDataGas uint64, l1BaseFeeWei *big.Int) (*types.Transaction, error) {
	return _ArbosActs.Contract.BatchPostingReport(&_ArbosActs.TransactOpts, batchTimestamp, batchPosterAddress, batchNumber, batchDataGas, l1BaseFeeWei)
}

// StartBlock is a paid mutator transaction binding the contract method 0x6bf6a42d.
//
// Solidity: function startBlock(uint256 l1BaseFee, uint64 l1BlockNumber, uint64 l2BlockNumber, uint64 timePassed) returns()
func (_ArbosActs *ArbosActsTransactor) StartBlock(opts *bind.TransactOpts, l1BaseFee *big.Int, l1BlockNumber uint64, l2BlockNumber uint64, timePassed uint64) (*types.Transaction, error) {
	return _ArbosActs.contract.Transact(opts, "startBlock", l1BaseFee, l1BlockNumber, l2BlockNumber, timePassed)
}

// StartBlock is a paid mutator transaction binding the contract method 0x6bf6a42d.
//
// Solidity: function startBlock(uint256 l1BaseFee, uint64 l1BlockNumber, uint64 l2BlockNumber, uint64 timePassed) returns()
func (_ArbosActs *ArbosActsSession) StartBlock(l1BaseFee *big.Int, l1BlockNumber uint64, l2BlockNumber uint64, timePassed uint64) (*types.Transaction, error) {
	return _ArbosActs.Contract.StartBlock(&_ArbosActs.TransactOpts, l1BaseFee, l1BlockNumber, l2BlockNumber, timePassed)
}

// StartBlock is a paid mutator transaction binding the contract method 0x6bf6a42d.
//
// Solidity: function startBlock(uint256 l1BaseFee, uint64 l1BlockNumber, uint64 l2BlockNumber, uint64 timePassed) returns()
func (_ArbosActs *ArbosActsTransactorSession) StartBlock(l1BaseFee *big.Int, l1BlockNumber uint64, l2BlockNumber uint64, timePassed uint64) (*types.Transaction, error) {
	return _ArbosActs.Contract.StartBlock(&_ArbosActs.TransactOpts, l1BaseFee, l1BlockNumber, l2BlockNumber, timePassed)
}

// ArbosTestMetaData contains all meta data concerning the ArbosTest contract.
var ArbosTestMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasAmount\",\"type\":\"uint256\"}],\"name\":\"burnArbGas\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
}

// ArbosTestABI is the input ABI used to generate the binding from.
// Deprecated: Use ArbosTestMetaData.ABI instead.
var ArbosTestABI = ArbosTestMetaData.ABI

// ArbosTest is an auto generated Go binding around an Ethereum contract.
type ArbosTest struct {
	ArbosTestCaller     // Read-only binding to the contract
	ArbosTestTransactor // Write-only binding to the contract
	ArbosTestFilterer   // Log filterer for contract events
}

// ArbosTestCaller is an auto generated read-only Go binding around an Ethereum contract.
type ArbosTestCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbosTestTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ArbosTestTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbosTestFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ArbosTestFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ArbosTestSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ArbosTestSession struct {
	Contract     *ArbosTest        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ArbosTestCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ArbosTestCallerSession struct {
	Contract *ArbosTestCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// ArbosTestTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ArbosTestTransactorSession struct {
	Contract     *ArbosTestTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ArbosTestRaw is an auto generated low-level Go binding around an Ethereum contract.
type ArbosTestRaw struct {
	Contract *ArbosTest // Generic contract binding to access the raw methods on
}

// ArbosTestCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ArbosTestCallerRaw struct {
	Contract *ArbosTestCaller // Generic read-only contract binding to access the raw methods on
}

// ArbosTestTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ArbosTestTransactorRaw struct {
	Contract *ArbosTestTransactor // Generic write-only contract binding to access the raw methods on
}

// NewArbosTest creates a new instance of ArbosTest, bound to a specific deployed contract.
func NewArbosTest(address common.Address, backend bind.ContractBackend) (*ArbosTest, error) {
	contract, err := bindArbosTest(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ArbosTest{ArbosTestCaller: ArbosTestCaller{contract: contract}, ArbosTestTransactor: ArbosTestTransactor{contract: contract}, ArbosTestFilterer: ArbosTestFilterer{contract: contract}}, nil
}

// NewArbosTestCaller creates a new read-only instance of ArbosTest, bound to a specific deployed contract.
func NewArbosTestCaller(address common.Address, caller bind.ContractCaller) (*ArbosTestCaller, error) {
	contract, err := bindArbosTest(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ArbosTestCaller{contract: contract}, nil
}

// NewArbosTestTransactor creates a new write-only instance of ArbosTest, bound to a specific deployed contract.
func NewArbosTestTransactor(address common.Address, transactor bind.ContractTransactor) (*ArbosTestTransactor, error) {
	contract, err := bindArbosTest(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ArbosTestTransactor{contract: contract}, nil
}

// NewArbosTestFilterer creates a new log filterer instance of ArbosTest, bound to a specific deployed contract.
func NewArbosTestFilterer(address common.Address, filterer bind.ContractFilterer) (*ArbosTestFilterer, error) {
	contract, err := bindArbosTest(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ArbosTestFilterer{contract: contract}, nil
}

// bindArbosTest binds a generic wrapper to an already deployed contract.
func bindArbosTest(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ArbosTestMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbosTest *ArbosTestRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbosTest.Contract.ArbosTestCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbosTest *ArbosTestRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbosTest.Contract.ArbosTestTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbosTest *ArbosTestRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbosTest.Contract.ArbosTestTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ArbosTest *ArbosTestCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ArbosTest.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ArbosTest *ArbosTestTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ArbosTest.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ArbosTest *ArbosTestTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ArbosTest.Contract.contract.Transact(opts, method, params...)
}

// BurnArbGas is a free data retrieval call binding the contract method 0xbb3480f9.
//
// Solidity: function burnArbGas(uint256 gasAmount) pure returns()
func (_ArbosTest *ArbosTestCaller) BurnArbGas(opts *bind.CallOpts, gasAmount *big.Int) error {
	var out []interface{}
	err := _ArbosTest.contract.Call(opts, &out, "burnArbGas", gasAmount)

	if err != nil {
		return err
	}

	return err

}

// BurnArbGas is a free data retrieval call binding the contract method 0xbb3480f9.
//
// Solidity: function burnArbGas(uint256 gasAmount) pure returns()
func (_ArbosTest *ArbosTestSession) BurnArbGas(gasAmount *big.Int) error {
	return _ArbosTest.Contract.BurnArbGas(&_ArbosTest.CallOpts, gasAmount)
}

// BurnArbGas is a free data retrieval call binding the contract method 0xbb3480f9.
//
// Solidity: function burnArbGas(uint256 gasAmount) pure returns()
func (_ArbosTest *ArbosTestCallerSession) BurnArbGas(gasAmount *big.Int) error {
	return _ArbosTest.Contract.BurnArbGas(&_ArbosTest.CallOpts, gasAmount)
}
