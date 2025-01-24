// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package yulgen

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

// Reader4844MetaData contains all meta data concerning the Reader4844 contract.
var Reader4844MetaData = &bind.MetaData{
	ABI: "null",
	Bin: "0x605b80600c6000396000f3fe346056576000803560e01c8063e83a2d8214602857631f6d6ef71460205780fd5b6020904a8152f35b50805b804990811560435760019160408260051b015201602b565b60409150602083528060205260051b0190f35b600080fd",
}

// Reader4844ABI is the input ABI used to generate the binding from.
// Deprecated: Use Reader4844MetaData.ABI instead.
var Reader4844ABI = Reader4844MetaData.ABI

// Reader4844Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use Reader4844MetaData.Bin instead.
var Reader4844Bin = Reader4844MetaData.Bin

// DeployReader4844 deploys a new Ethereum contract, binding an instance of Reader4844 to it.
func DeployReader4844(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Reader4844, error) {
	parsed, err := Reader4844MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(Reader4844Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Reader4844{Reader4844Caller: Reader4844Caller{contract: contract}, Reader4844Transactor: Reader4844Transactor{contract: contract}, Reader4844Filterer: Reader4844Filterer{contract: contract}}, nil
}

// Reader4844 is an auto generated Go binding around an Ethereum contract.
type Reader4844 struct {
	Reader4844Caller     // Read-only binding to the contract
	Reader4844Transactor // Write-only binding to the contract
	Reader4844Filterer   // Log filterer for contract events
}

// Reader4844Caller is an auto generated read-only Go binding around an Ethereum contract.
type Reader4844Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Reader4844Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Reader4844Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Reader4844Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Reader4844Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Reader4844Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Reader4844Session struct {
	Contract     *Reader4844       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Reader4844CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Reader4844CallerSession struct {
	Contract *Reader4844Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// Reader4844TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Reader4844TransactorSession struct {
	Contract     *Reader4844Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// Reader4844Raw is an auto generated low-level Go binding around an Ethereum contract.
type Reader4844Raw struct {
	Contract *Reader4844 // Generic contract binding to access the raw methods on
}

// Reader4844CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Reader4844CallerRaw struct {
	Contract *Reader4844Caller // Generic read-only contract binding to access the raw methods on
}

// Reader4844TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Reader4844TransactorRaw struct {
	Contract *Reader4844Transactor // Generic write-only contract binding to access the raw methods on
}

// NewReader4844 creates a new instance of Reader4844, bound to a specific deployed contract.
func NewReader4844(address common.Address, backend bind.ContractBackend) (*Reader4844, error) {
	contract, err := bindReader4844(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Reader4844{Reader4844Caller: Reader4844Caller{contract: contract}, Reader4844Transactor: Reader4844Transactor{contract: contract}, Reader4844Filterer: Reader4844Filterer{contract: contract}}, nil
}

// NewReader4844Caller creates a new read-only instance of Reader4844, bound to a specific deployed contract.
func NewReader4844Caller(address common.Address, caller bind.ContractCaller) (*Reader4844Caller, error) {
	contract, err := bindReader4844(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Reader4844Caller{contract: contract}, nil
}

// NewReader4844Transactor creates a new write-only instance of Reader4844, bound to a specific deployed contract.
func NewReader4844Transactor(address common.Address, transactor bind.ContractTransactor) (*Reader4844Transactor, error) {
	contract, err := bindReader4844(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Reader4844Transactor{contract: contract}, nil
}

// NewReader4844Filterer creates a new log filterer instance of Reader4844, bound to a specific deployed contract.
func NewReader4844Filterer(address common.Address, filterer bind.ContractFilterer) (*Reader4844Filterer, error) {
	contract, err := bindReader4844(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Reader4844Filterer{contract: contract}, nil
}

// bindReader4844 binds a generic wrapper to an already deployed contract.
func bindReader4844(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Reader4844MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Reader4844 *Reader4844Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Reader4844.Contract.Reader4844Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Reader4844 *Reader4844Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Reader4844.Contract.Reader4844Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Reader4844 *Reader4844Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Reader4844.Contract.Reader4844Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Reader4844 *Reader4844CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Reader4844.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Reader4844 *Reader4844TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Reader4844.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Reader4844 *Reader4844TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Reader4844.Contract.contract.Transact(opts, method, params...)
}
