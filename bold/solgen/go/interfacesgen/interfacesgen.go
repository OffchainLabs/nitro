// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package interfacesgen

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

// ERC1155TokenReceiverMetaData contains all meta data concerning the ERC1155TokenReceiver contract.
var ERC1155TokenReceiverMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_operator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"_ids\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_values\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"onERC1155BatchReceived\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_operator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"onERC1155Received\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ERC1155TokenReceiverABI is the input ABI used to generate the binding from.
// Deprecated: Use ERC1155TokenReceiverMetaData.ABI instead.
var ERC1155TokenReceiverABI = ERC1155TokenReceiverMetaData.ABI

// ERC1155TokenReceiver is an auto generated Go binding around an Ethereum contract.
type ERC1155TokenReceiver struct {
	ERC1155TokenReceiverCaller     // Read-only binding to the contract
	ERC1155TokenReceiverTransactor // Write-only binding to the contract
	ERC1155TokenReceiverFilterer   // Log filterer for contract events
}

// ERC1155TokenReceiverCaller is an auto generated read-only Go binding around an Ethereum contract.
type ERC1155TokenReceiverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC1155TokenReceiverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC1155TokenReceiverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC1155TokenReceiverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC1155TokenReceiverFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC1155TokenReceiverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC1155TokenReceiverSession struct {
	Contract     *ERC1155TokenReceiver // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ERC1155TokenReceiverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC1155TokenReceiverCallerSession struct {
	Contract *ERC1155TokenReceiverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// ERC1155TokenReceiverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC1155TokenReceiverTransactorSession struct {
	Contract     *ERC1155TokenReceiverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// ERC1155TokenReceiverRaw is an auto generated low-level Go binding around an Ethereum contract.
type ERC1155TokenReceiverRaw struct {
	Contract *ERC1155TokenReceiver // Generic contract binding to access the raw methods on
}

// ERC1155TokenReceiverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC1155TokenReceiverCallerRaw struct {
	Contract *ERC1155TokenReceiverCaller // Generic read-only contract binding to access the raw methods on
}

// ERC1155TokenReceiverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC1155TokenReceiverTransactorRaw struct {
	Contract *ERC1155TokenReceiverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewERC1155TokenReceiver creates a new instance of ERC1155TokenReceiver, bound to a specific deployed contract.
func NewERC1155TokenReceiver(address common.Address, backend bind.ContractBackend) (*ERC1155TokenReceiver, error) {
	contract, err := bindERC1155TokenReceiver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC1155TokenReceiver{ERC1155TokenReceiverCaller: ERC1155TokenReceiverCaller{contract: contract}, ERC1155TokenReceiverTransactor: ERC1155TokenReceiverTransactor{contract: contract}, ERC1155TokenReceiverFilterer: ERC1155TokenReceiverFilterer{contract: contract}}, nil
}

// NewERC1155TokenReceiverCaller creates a new read-only instance of ERC1155TokenReceiver, bound to a specific deployed contract.
func NewERC1155TokenReceiverCaller(address common.Address, caller bind.ContractCaller) (*ERC1155TokenReceiverCaller, error) {
	contract, err := bindERC1155TokenReceiver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC1155TokenReceiverCaller{contract: contract}, nil
}

// NewERC1155TokenReceiverTransactor creates a new write-only instance of ERC1155TokenReceiver, bound to a specific deployed contract.
func NewERC1155TokenReceiverTransactor(address common.Address, transactor bind.ContractTransactor) (*ERC1155TokenReceiverTransactor, error) {
	contract, err := bindERC1155TokenReceiver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC1155TokenReceiverTransactor{contract: contract}, nil
}

// NewERC1155TokenReceiverFilterer creates a new log filterer instance of ERC1155TokenReceiver, bound to a specific deployed contract.
func NewERC1155TokenReceiverFilterer(address common.Address, filterer bind.ContractFilterer) (*ERC1155TokenReceiverFilterer, error) {
	contract, err := bindERC1155TokenReceiver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC1155TokenReceiverFilterer{contract: contract}, nil
}

// bindERC1155TokenReceiver binds a generic wrapper to an already deployed contract.
func bindERC1155TokenReceiver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ERC1155TokenReceiverMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC1155TokenReceiver *ERC1155TokenReceiverRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC1155TokenReceiver.Contract.ERC1155TokenReceiverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC1155TokenReceiver *ERC1155TokenReceiverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC1155TokenReceiver.Contract.ERC1155TokenReceiverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC1155TokenReceiver *ERC1155TokenReceiverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC1155TokenReceiver.Contract.ERC1155TokenReceiverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC1155TokenReceiver *ERC1155TokenReceiverCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC1155TokenReceiver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC1155TokenReceiver *ERC1155TokenReceiverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC1155TokenReceiver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC1155TokenReceiver *ERC1155TokenReceiverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC1155TokenReceiver.Contract.contract.Transact(opts, method, params...)
}

// OnERC1155BatchReceived is a paid mutator transaction binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address _operator, address _from, uint256[] _ids, uint256[] _values, bytes _data) returns(bytes4)
func (_ERC1155TokenReceiver *ERC1155TokenReceiverTransactor) OnERC1155BatchReceived(opts *bind.TransactOpts, _operator common.Address, _from common.Address, _ids []*big.Int, _values []*big.Int, _data []byte) (*types.Transaction, error) {
	return _ERC1155TokenReceiver.contract.Transact(opts, "onERC1155BatchReceived", _operator, _from, _ids, _values, _data)
}

// OnERC1155BatchReceived is a paid mutator transaction binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address _operator, address _from, uint256[] _ids, uint256[] _values, bytes _data) returns(bytes4)
func (_ERC1155TokenReceiver *ERC1155TokenReceiverSession) OnERC1155BatchReceived(_operator common.Address, _from common.Address, _ids []*big.Int, _values []*big.Int, _data []byte) (*types.Transaction, error) {
	return _ERC1155TokenReceiver.Contract.OnERC1155BatchReceived(&_ERC1155TokenReceiver.TransactOpts, _operator, _from, _ids, _values, _data)
}

// OnERC1155BatchReceived is a paid mutator transaction binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address _operator, address _from, uint256[] _ids, uint256[] _values, bytes _data) returns(bytes4)
func (_ERC1155TokenReceiver *ERC1155TokenReceiverTransactorSession) OnERC1155BatchReceived(_operator common.Address, _from common.Address, _ids []*big.Int, _values []*big.Int, _data []byte) (*types.Transaction, error) {
	return _ERC1155TokenReceiver.Contract.OnERC1155BatchReceived(&_ERC1155TokenReceiver.TransactOpts, _operator, _from, _ids, _values, _data)
}

// OnERC1155Received is a paid mutator transaction binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address _operator, address _from, uint256 _id, uint256 _value, bytes _data) returns(bytes4)
func (_ERC1155TokenReceiver *ERC1155TokenReceiverTransactor) OnERC1155Received(opts *bind.TransactOpts, _operator common.Address, _from common.Address, _id *big.Int, _value *big.Int, _data []byte) (*types.Transaction, error) {
	return _ERC1155TokenReceiver.contract.Transact(opts, "onERC1155Received", _operator, _from, _id, _value, _data)
}

// OnERC1155Received is a paid mutator transaction binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address _operator, address _from, uint256 _id, uint256 _value, bytes _data) returns(bytes4)
func (_ERC1155TokenReceiver *ERC1155TokenReceiverSession) OnERC1155Received(_operator common.Address, _from common.Address, _id *big.Int, _value *big.Int, _data []byte) (*types.Transaction, error) {
	return _ERC1155TokenReceiver.Contract.OnERC1155Received(&_ERC1155TokenReceiver.TransactOpts, _operator, _from, _id, _value, _data)
}

// OnERC1155Received is a paid mutator transaction binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address _operator, address _from, uint256 _id, uint256 _value, bytes _data) returns(bytes4)
func (_ERC1155TokenReceiver *ERC1155TokenReceiverTransactorSession) OnERC1155Received(_operator common.Address, _from common.Address, _id *big.Int, _value *big.Int, _data []byte) (*types.Transaction, error) {
	return _ERC1155TokenReceiver.Contract.OnERC1155Received(&_ERC1155TokenReceiver.TransactOpts, _operator, _from, _id, _value, _data)
}

// ERC721TokenReceiverMetaData contains all meta data concerning the ERC721TokenReceiver contract.
var ERC721TokenReceiverMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_operator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_tokenId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"onERC721Received\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ERC721TokenReceiverABI is the input ABI used to generate the binding from.
// Deprecated: Use ERC721TokenReceiverMetaData.ABI instead.
var ERC721TokenReceiverABI = ERC721TokenReceiverMetaData.ABI

// ERC721TokenReceiver is an auto generated Go binding around an Ethereum contract.
type ERC721TokenReceiver struct {
	ERC721TokenReceiverCaller     // Read-only binding to the contract
	ERC721TokenReceiverTransactor // Write-only binding to the contract
	ERC721TokenReceiverFilterer   // Log filterer for contract events
}

// ERC721TokenReceiverCaller is an auto generated read-only Go binding around an Ethereum contract.
type ERC721TokenReceiverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC721TokenReceiverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC721TokenReceiverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC721TokenReceiverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC721TokenReceiverFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC721TokenReceiverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC721TokenReceiverSession struct {
	Contract     *ERC721TokenReceiver // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ERC721TokenReceiverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC721TokenReceiverCallerSession struct {
	Contract *ERC721TokenReceiverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// ERC721TokenReceiverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC721TokenReceiverTransactorSession struct {
	Contract     *ERC721TokenReceiverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// ERC721TokenReceiverRaw is an auto generated low-level Go binding around an Ethereum contract.
type ERC721TokenReceiverRaw struct {
	Contract *ERC721TokenReceiver // Generic contract binding to access the raw methods on
}

// ERC721TokenReceiverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC721TokenReceiverCallerRaw struct {
	Contract *ERC721TokenReceiverCaller // Generic read-only contract binding to access the raw methods on
}

// ERC721TokenReceiverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC721TokenReceiverTransactorRaw struct {
	Contract *ERC721TokenReceiverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewERC721TokenReceiver creates a new instance of ERC721TokenReceiver, bound to a specific deployed contract.
func NewERC721TokenReceiver(address common.Address, backend bind.ContractBackend) (*ERC721TokenReceiver, error) {
	contract, err := bindERC721TokenReceiver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC721TokenReceiver{ERC721TokenReceiverCaller: ERC721TokenReceiverCaller{contract: contract}, ERC721TokenReceiverTransactor: ERC721TokenReceiverTransactor{contract: contract}, ERC721TokenReceiverFilterer: ERC721TokenReceiverFilterer{contract: contract}}, nil
}

// NewERC721TokenReceiverCaller creates a new read-only instance of ERC721TokenReceiver, bound to a specific deployed contract.
func NewERC721TokenReceiverCaller(address common.Address, caller bind.ContractCaller) (*ERC721TokenReceiverCaller, error) {
	contract, err := bindERC721TokenReceiver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC721TokenReceiverCaller{contract: contract}, nil
}

// NewERC721TokenReceiverTransactor creates a new write-only instance of ERC721TokenReceiver, bound to a specific deployed contract.
func NewERC721TokenReceiverTransactor(address common.Address, transactor bind.ContractTransactor) (*ERC721TokenReceiverTransactor, error) {
	contract, err := bindERC721TokenReceiver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC721TokenReceiverTransactor{contract: contract}, nil
}

// NewERC721TokenReceiverFilterer creates a new log filterer instance of ERC721TokenReceiver, bound to a specific deployed contract.
func NewERC721TokenReceiverFilterer(address common.Address, filterer bind.ContractFilterer) (*ERC721TokenReceiverFilterer, error) {
	contract, err := bindERC721TokenReceiver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC721TokenReceiverFilterer{contract: contract}, nil
}

// bindERC721TokenReceiver binds a generic wrapper to an already deployed contract.
func bindERC721TokenReceiver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ERC721TokenReceiverMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC721TokenReceiver *ERC721TokenReceiverRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC721TokenReceiver.Contract.ERC721TokenReceiverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC721TokenReceiver *ERC721TokenReceiverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC721TokenReceiver.Contract.ERC721TokenReceiverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC721TokenReceiver *ERC721TokenReceiverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC721TokenReceiver.Contract.ERC721TokenReceiverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC721TokenReceiver *ERC721TokenReceiverCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC721TokenReceiver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC721TokenReceiver *ERC721TokenReceiverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC721TokenReceiver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC721TokenReceiver *ERC721TokenReceiverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC721TokenReceiver.Contract.contract.Transact(opts, method, params...)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address _operator, address _from, uint256 _tokenId, bytes _data) returns(bytes4)
func (_ERC721TokenReceiver *ERC721TokenReceiverTransactor) OnERC721Received(opts *bind.TransactOpts, _operator common.Address, _from common.Address, _tokenId *big.Int, _data []byte) (*types.Transaction, error) {
	return _ERC721TokenReceiver.contract.Transact(opts, "onERC721Received", _operator, _from, _tokenId, _data)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address _operator, address _from, uint256 _tokenId, bytes _data) returns(bytes4)
func (_ERC721TokenReceiver *ERC721TokenReceiverSession) OnERC721Received(_operator common.Address, _from common.Address, _tokenId *big.Int, _data []byte) (*types.Transaction, error) {
	return _ERC721TokenReceiver.Contract.OnERC721Received(&_ERC721TokenReceiver.TransactOpts, _operator, _from, _tokenId, _data)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address _operator, address _from, uint256 _tokenId, bytes _data) returns(bytes4)
func (_ERC721TokenReceiver *ERC721TokenReceiverTransactorSession) OnERC721Received(_operator common.Address, _from common.Address, _tokenId *big.Int, _data []byte) (*types.Transaction, error) {
	return _ERC721TokenReceiver.Contract.OnERC721Received(&_ERC721TokenReceiver.TransactOpts, _operator, _from, _tokenId, _data)
}

// ERC777TokensRecipientMetaData contains all meta data concerning the ERC777TokensRecipient contract.
var ERC777TokensRecipientMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"operatorData\",\"type\":\"bytes\"}],\"name\":\"tokensReceived\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ERC777TokensRecipientABI is the input ABI used to generate the binding from.
// Deprecated: Use ERC777TokensRecipientMetaData.ABI instead.
var ERC777TokensRecipientABI = ERC777TokensRecipientMetaData.ABI

// ERC777TokensRecipient is an auto generated Go binding around an Ethereum contract.
type ERC777TokensRecipient struct {
	ERC777TokensRecipientCaller     // Read-only binding to the contract
	ERC777TokensRecipientTransactor // Write-only binding to the contract
	ERC777TokensRecipientFilterer   // Log filterer for contract events
}

// ERC777TokensRecipientCaller is an auto generated read-only Go binding around an Ethereum contract.
type ERC777TokensRecipientCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC777TokensRecipientTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC777TokensRecipientTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC777TokensRecipientFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC777TokensRecipientFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC777TokensRecipientSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC777TokensRecipientSession struct {
	Contract     *ERC777TokensRecipient // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// ERC777TokensRecipientCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC777TokensRecipientCallerSession struct {
	Contract *ERC777TokensRecipientCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// ERC777TokensRecipientTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC777TokensRecipientTransactorSession struct {
	Contract     *ERC777TokensRecipientTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// ERC777TokensRecipientRaw is an auto generated low-level Go binding around an Ethereum contract.
type ERC777TokensRecipientRaw struct {
	Contract *ERC777TokensRecipient // Generic contract binding to access the raw methods on
}

// ERC777TokensRecipientCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC777TokensRecipientCallerRaw struct {
	Contract *ERC777TokensRecipientCaller // Generic read-only contract binding to access the raw methods on
}

// ERC777TokensRecipientTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC777TokensRecipientTransactorRaw struct {
	Contract *ERC777TokensRecipientTransactor // Generic write-only contract binding to access the raw methods on
}

// NewERC777TokensRecipient creates a new instance of ERC777TokensRecipient, bound to a specific deployed contract.
func NewERC777TokensRecipient(address common.Address, backend bind.ContractBackend) (*ERC777TokensRecipient, error) {
	contract, err := bindERC777TokensRecipient(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC777TokensRecipient{ERC777TokensRecipientCaller: ERC777TokensRecipientCaller{contract: contract}, ERC777TokensRecipientTransactor: ERC777TokensRecipientTransactor{contract: contract}, ERC777TokensRecipientFilterer: ERC777TokensRecipientFilterer{contract: contract}}, nil
}

// NewERC777TokensRecipientCaller creates a new read-only instance of ERC777TokensRecipient, bound to a specific deployed contract.
func NewERC777TokensRecipientCaller(address common.Address, caller bind.ContractCaller) (*ERC777TokensRecipientCaller, error) {
	contract, err := bindERC777TokensRecipient(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC777TokensRecipientCaller{contract: contract}, nil
}

// NewERC777TokensRecipientTransactor creates a new write-only instance of ERC777TokensRecipient, bound to a specific deployed contract.
func NewERC777TokensRecipientTransactor(address common.Address, transactor bind.ContractTransactor) (*ERC777TokensRecipientTransactor, error) {
	contract, err := bindERC777TokensRecipient(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC777TokensRecipientTransactor{contract: contract}, nil
}

// NewERC777TokensRecipientFilterer creates a new log filterer instance of ERC777TokensRecipient, bound to a specific deployed contract.
func NewERC777TokensRecipientFilterer(address common.Address, filterer bind.ContractFilterer) (*ERC777TokensRecipientFilterer, error) {
	contract, err := bindERC777TokensRecipient(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC777TokensRecipientFilterer{contract: contract}, nil
}

// bindERC777TokensRecipient binds a generic wrapper to an already deployed contract.
func bindERC777TokensRecipient(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ERC777TokensRecipientMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC777TokensRecipient *ERC777TokensRecipientRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC777TokensRecipient.Contract.ERC777TokensRecipientCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC777TokensRecipient *ERC777TokensRecipientRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC777TokensRecipient.Contract.ERC777TokensRecipientTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC777TokensRecipient *ERC777TokensRecipientRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC777TokensRecipient.Contract.ERC777TokensRecipientTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC777TokensRecipient *ERC777TokensRecipientCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC777TokensRecipient.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC777TokensRecipient *ERC777TokensRecipientTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC777TokensRecipient.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC777TokensRecipient *ERC777TokensRecipientTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC777TokensRecipient.Contract.contract.Transact(opts, method, params...)
}

// TokensReceived is a paid mutator transaction binding the contract method 0x0023de29.
//
// Solidity: function tokensReceived(address operator, address from, address to, uint256 amount, bytes data, bytes operatorData) returns()
func (_ERC777TokensRecipient *ERC777TokensRecipientTransactor) TokensReceived(opts *bind.TransactOpts, operator common.Address, from common.Address, to common.Address, amount *big.Int, data []byte, operatorData []byte) (*types.Transaction, error) {
	return _ERC777TokensRecipient.contract.Transact(opts, "tokensReceived", operator, from, to, amount, data, operatorData)
}

// TokensReceived is a paid mutator transaction binding the contract method 0x0023de29.
//
// Solidity: function tokensReceived(address operator, address from, address to, uint256 amount, bytes data, bytes operatorData) returns()
func (_ERC777TokensRecipient *ERC777TokensRecipientSession) TokensReceived(operator common.Address, from common.Address, to common.Address, amount *big.Int, data []byte, operatorData []byte) (*types.Transaction, error) {
	return _ERC777TokensRecipient.Contract.TokensReceived(&_ERC777TokensRecipient.TransactOpts, operator, from, to, amount, data, operatorData)
}

// TokensReceived is a paid mutator transaction binding the contract method 0x0023de29.
//
// Solidity: function tokensReceived(address operator, address from, address to, uint256 amount, bytes data, bytes operatorData) returns()
func (_ERC777TokensRecipient *ERC777TokensRecipientTransactorSession) TokensReceived(operator common.Address, from common.Address, to common.Address, amount *big.Int, data []byte, operatorData []byte) (*types.Transaction, error) {
	return _ERC777TokensRecipient.Contract.TokensReceived(&_ERC777TokensRecipient.TransactOpts, operator, from, to, amount, data, operatorData)
}

// IERC165MetaData contains all meta data concerning the IERC165 contract.
var IERC165MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IERC165ABI is the input ABI used to generate the binding from.
// Deprecated: Use IERC165MetaData.ABI instead.
var IERC165ABI = IERC165MetaData.ABI

// IERC165 is an auto generated Go binding around an Ethereum contract.
type IERC165 struct {
	IERC165Caller     // Read-only binding to the contract
	IERC165Transactor // Write-only binding to the contract
	IERC165Filterer   // Log filterer for contract events
}

// IERC165Caller is an auto generated read-only Go binding around an Ethereum contract.
type IERC165Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IERC165Transactor is an auto generated write-only Go binding around an Ethereum contract.
type IERC165Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IERC165Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IERC165Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IERC165Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IERC165Session struct {
	Contract     *IERC165          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IERC165CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IERC165CallerSession struct {
	Contract *IERC165Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// IERC165TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IERC165TransactorSession struct {
	Contract     *IERC165Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// IERC165Raw is an auto generated low-level Go binding around an Ethereum contract.
type IERC165Raw struct {
	Contract *IERC165 // Generic contract binding to access the raw methods on
}

// IERC165CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IERC165CallerRaw struct {
	Contract *IERC165Caller // Generic read-only contract binding to access the raw methods on
}

// IERC165TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IERC165TransactorRaw struct {
	Contract *IERC165Transactor // Generic write-only contract binding to access the raw methods on
}

// NewIERC165 creates a new instance of IERC165, bound to a specific deployed contract.
func NewIERC165(address common.Address, backend bind.ContractBackend) (*IERC165, error) {
	contract, err := bindIERC165(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IERC165{IERC165Caller: IERC165Caller{contract: contract}, IERC165Transactor: IERC165Transactor{contract: contract}, IERC165Filterer: IERC165Filterer{contract: contract}}, nil
}

// NewIERC165Caller creates a new read-only instance of IERC165, bound to a specific deployed contract.
func NewIERC165Caller(address common.Address, caller bind.ContractCaller) (*IERC165Caller, error) {
	contract, err := bindIERC165(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IERC165Caller{contract: contract}, nil
}

// NewIERC165Transactor creates a new write-only instance of IERC165, bound to a specific deployed contract.
func NewIERC165Transactor(address common.Address, transactor bind.ContractTransactor) (*IERC165Transactor, error) {
	contract, err := bindIERC165(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IERC165Transactor{contract: contract}, nil
}

// NewIERC165Filterer creates a new log filterer instance of IERC165, bound to a specific deployed contract.
func NewIERC165Filterer(address common.Address, filterer bind.ContractFilterer) (*IERC165Filterer, error) {
	contract, err := bindIERC165(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IERC165Filterer{contract: contract}, nil
}

// bindIERC165 binds a generic wrapper to an already deployed contract.
func bindIERC165(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IERC165MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IERC165 *IERC165Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IERC165.Contract.IERC165Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IERC165 *IERC165Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IERC165.Contract.IERC165Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IERC165 *IERC165Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IERC165.Contract.IERC165Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IERC165 *IERC165CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IERC165.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IERC165 *IERC165TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IERC165.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IERC165 *IERC165TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IERC165.Contract.contract.Transact(opts, method, params...)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_IERC165 *IERC165Caller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _IERC165.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_IERC165 *IERC165Session) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _IERC165.Contract.SupportsInterface(&_IERC165.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_IERC165 *IERC165CallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _IERC165.Contract.SupportsInterface(&_IERC165.CallOpts, interfaceId)
}

// ISignatureValidatorMetaData contains all meta data concerning the ISignatureValidator contract.
var ISignatureValidatorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_signature\",\"type\":\"bytes\"}],\"name\":\"isValidSignature\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// ISignatureValidatorABI is the input ABI used to generate the binding from.
// Deprecated: Use ISignatureValidatorMetaData.ABI instead.
var ISignatureValidatorABI = ISignatureValidatorMetaData.ABI

// ISignatureValidator is an auto generated Go binding around an Ethereum contract.
type ISignatureValidator struct {
	ISignatureValidatorCaller     // Read-only binding to the contract
	ISignatureValidatorTransactor // Write-only binding to the contract
	ISignatureValidatorFilterer   // Log filterer for contract events
}

// ISignatureValidatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type ISignatureValidatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISignatureValidatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ISignatureValidatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISignatureValidatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ISignatureValidatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISignatureValidatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ISignatureValidatorSession struct {
	Contract     *ISignatureValidator // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ISignatureValidatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ISignatureValidatorCallerSession struct {
	Contract *ISignatureValidatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// ISignatureValidatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ISignatureValidatorTransactorSession struct {
	Contract     *ISignatureValidatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// ISignatureValidatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type ISignatureValidatorRaw struct {
	Contract *ISignatureValidator // Generic contract binding to access the raw methods on
}

// ISignatureValidatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ISignatureValidatorCallerRaw struct {
	Contract *ISignatureValidatorCaller // Generic read-only contract binding to access the raw methods on
}

// ISignatureValidatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ISignatureValidatorTransactorRaw struct {
	Contract *ISignatureValidatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewISignatureValidator creates a new instance of ISignatureValidator, bound to a specific deployed contract.
func NewISignatureValidator(address common.Address, backend bind.ContractBackend) (*ISignatureValidator, error) {
	contract, err := bindISignatureValidator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ISignatureValidator{ISignatureValidatorCaller: ISignatureValidatorCaller{contract: contract}, ISignatureValidatorTransactor: ISignatureValidatorTransactor{contract: contract}, ISignatureValidatorFilterer: ISignatureValidatorFilterer{contract: contract}}, nil
}

// NewISignatureValidatorCaller creates a new read-only instance of ISignatureValidator, bound to a specific deployed contract.
func NewISignatureValidatorCaller(address common.Address, caller bind.ContractCaller) (*ISignatureValidatorCaller, error) {
	contract, err := bindISignatureValidator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ISignatureValidatorCaller{contract: contract}, nil
}

// NewISignatureValidatorTransactor creates a new write-only instance of ISignatureValidator, bound to a specific deployed contract.
func NewISignatureValidatorTransactor(address common.Address, transactor bind.ContractTransactor) (*ISignatureValidatorTransactor, error) {
	contract, err := bindISignatureValidator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ISignatureValidatorTransactor{contract: contract}, nil
}

// NewISignatureValidatorFilterer creates a new log filterer instance of ISignatureValidator, bound to a specific deployed contract.
func NewISignatureValidatorFilterer(address common.Address, filterer bind.ContractFilterer) (*ISignatureValidatorFilterer, error) {
	contract, err := bindISignatureValidator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ISignatureValidatorFilterer{contract: contract}, nil
}

// bindISignatureValidator binds a generic wrapper to an already deployed contract.
func bindISignatureValidator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ISignatureValidatorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ISignatureValidator *ISignatureValidatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ISignatureValidator.Contract.ISignatureValidatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ISignatureValidator *ISignatureValidatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISignatureValidator.Contract.ISignatureValidatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ISignatureValidator *ISignatureValidatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ISignatureValidator.Contract.ISignatureValidatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ISignatureValidator *ISignatureValidatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ISignatureValidator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ISignatureValidator *ISignatureValidatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISignatureValidator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ISignatureValidator *ISignatureValidatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ISignatureValidator.Contract.contract.Transact(opts, method, params...)
}

// IsValidSignature is a free data retrieval call binding the contract method 0x20c13b0b.
//
// Solidity: function isValidSignature(bytes _data, bytes _signature) view returns(bytes4)
func (_ISignatureValidator *ISignatureValidatorCaller) IsValidSignature(opts *bind.CallOpts, _data []byte, _signature []byte) ([4]byte, error) {
	var out []interface{}
	err := _ISignatureValidator.contract.Call(opts, &out, "isValidSignature", _data, _signature)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// IsValidSignature is a free data retrieval call binding the contract method 0x20c13b0b.
//
// Solidity: function isValidSignature(bytes _data, bytes _signature) view returns(bytes4)
func (_ISignatureValidator *ISignatureValidatorSession) IsValidSignature(_data []byte, _signature []byte) ([4]byte, error) {
	return _ISignatureValidator.Contract.IsValidSignature(&_ISignatureValidator.CallOpts, _data, _signature)
}

// IsValidSignature is a free data retrieval call binding the contract method 0x20c13b0b.
//
// Solidity: function isValidSignature(bytes _data, bytes _signature) view returns(bytes4)
func (_ISignatureValidator *ISignatureValidatorCallerSession) IsValidSignature(_data []byte, _signature []byte) ([4]byte, error) {
	return _ISignatureValidator.Contract.IsValidSignature(&_ISignatureValidator.CallOpts, _data, _signature)
}

// ISignatureValidatorConstantsMetaData contains all meta data concerning the ISignatureValidatorConstants contract.
var ISignatureValidatorConstantsMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x6080604052348015600f57600080fd5b50603f80601d6000396000f3fe6080604052600080fdfea26469706673582212203615497132de5030615c9512b13fb6dd2def86745a8c743d2776c6f56e468de564736f6c63430007060033",
}

// ISignatureValidatorConstantsABI is the input ABI used to generate the binding from.
// Deprecated: Use ISignatureValidatorConstantsMetaData.ABI instead.
var ISignatureValidatorConstantsABI = ISignatureValidatorConstantsMetaData.ABI

// ISignatureValidatorConstantsBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ISignatureValidatorConstantsMetaData.Bin instead.
var ISignatureValidatorConstantsBin = ISignatureValidatorConstantsMetaData.Bin

// DeployISignatureValidatorConstants deploys a new Ethereum contract, binding an instance of ISignatureValidatorConstants to it.
func DeployISignatureValidatorConstants(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ISignatureValidatorConstants, error) {
	parsed, err := ISignatureValidatorConstantsMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ISignatureValidatorConstantsBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ISignatureValidatorConstants{ISignatureValidatorConstantsCaller: ISignatureValidatorConstantsCaller{contract: contract}, ISignatureValidatorConstantsTransactor: ISignatureValidatorConstantsTransactor{contract: contract}, ISignatureValidatorConstantsFilterer: ISignatureValidatorConstantsFilterer{contract: contract}}, nil
}

// ISignatureValidatorConstants is an auto generated Go binding around an Ethereum contract.
type ISignatureValidatorConstants struct {
	ISignatureValidatorConstantsCaller     // Read-only binding to the contract
	ISignatureValidatorConstantsTransactor // Write-only binding to the contract
	ISignatureValidatorConstantsFilterer   // Log filterer for contract events
}

// ISignatureValidatorConstantsCaller is an auto generated read-only Go binding around an Ethereum contract.
type ISignatureValidatorConstantsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISignatureValidatorConstantsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ISignatureValidatorConstantsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISignatureValidatorConstantsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ISignatureValidatorConstantsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISignatureValidatorConstantsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ISignatureValidatorConstantsSession struct {
	Contract     *ISignatureValidatorConstants // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                 // Call options to use throughout this session
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// ISignatureValidatorConstantsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ISignatureValidatorConstantsCallerSession struct {
	Contract *ISignatureValidatorConstantsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                       // Call options to use throughout this session
}

// ISignatureValidatorConstantsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ISignatureValidatorConstantsTransactorSession struct {
	Contract     *ISignatureValidatorConstantsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                       // Transaction auth options to use throughout this session
}

// ISignatureValidatorConstantsRaw is an auto generated low-level Go binding around an Ethereum contract.
type ISignatureValidatorConstantsRaw struct {
	Contract *ISignatureValidatorConstants // Generic contract binding to access the raw methods on
}

// ISignatureValidatorConstantsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ISignatureValidatorConstantsCallerRaw struct {
	Contract *ISignatureValidatorConstantsCaller // Generic read-only contract binding to access the raw methods on
}

// ISignatureValidatorConstantsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ISignatureValidatorConstantsTransactorRaw struct {
	Contract *ISignatureValidatorConstantsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewISignatureValidatorConstants creates a new instance of ISignatureValidatorConstants, bound to a specific deployed contract.
func NewISignatureValidatorConstants(address common.Address, backend bind.ContractBackend) (*ISignatureValidatorConstants, error) {
	contract, err := bindISignatureValidatorConstants(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ISignatureValidatorConstants{ISignatureValidatorConstantsCaller: ISignatureValidatorConstantsCaller{contract: contract}, ISignatureValidatorConstantsTransactor: ISignatureValidatorConstantsTransactor{contract: contract}, ISignatureValidatorConstantsFilterer: ISignatureValidatorConstantsFilterer{contract: contract}}, nil
}

// NewISignatureValidatorConstantsCaller creates a new read-only instance of ISignatureValidatorConstants, bound to a specific deployed contract.
func NewISignatureValidatorConstantsCaller(address common.Address, caller bind.ContractCaller) (*ISignatureValidatorConstantsCaller, error) {
	contract, err := bindISignatureValidatorConstants(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ISignatureValidatorConstantsCaller{contract: contract}, nil
}

// NewISignatureValidatorConstantsTransactor creates a new write-only instance of ISignatureValidatorConstants, bound to a specific deployed contract.
func NewISignatureValidatorConstantsTransactor(address common.Address, transactor bind.ContractTransactor) (*ISignatureValidatorConstantsTransactor, error) {
	contract, err := bindISignatureValidatorConstants(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ISignatureValidatorConstantsTransactor{contract: contract}, nil
}

// NewISignatureValidatorConstantsFilterer creates a new log filterer instance of ISignatureValidatorConstants, bound to a specific deployed contract.
func NewISignatureValidatorConstantsFilterer(address common.Address, filterer bind.ContractFilterer) (*ISignatureValidatorConstantsFilterer, error) {
	contract, err := bindISignatureValidatorConstants(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ISignatureValidatorConstantsFilterer{contract: contract}, nil
}

// bindISignatureValidatorConstants binds a generic wrapper to an already deployed contract.
func bindISignatureValidatorConstants(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ISignatureValidatorConstantsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ISignatureValidatorConstants *ISignatureValidatorConstantsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ISignatureValidatorConstants.Contract.ISignatureValidatorConstantsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ISignatureValidatorConstants *ISignatureValidatorConstantsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISignatureValidatorConstants.Contract.ISignatureValidatorConstantsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ISignatureValidatorConstants *ISignatureValidatorConstantsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ISignatureValidatorConstants.Contract.ISignatureValidatorConstantsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ISignatureValidatorConstants *ISignatureValidatorConstantsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ISignatureValidatorConstants.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ISignatureValidatorConstants *ISignatureValidatorConstantsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISignatureValidatorConstants.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ISignatureValidatorConstants *ISignatureValidatorConstantsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ISignatureValidatorConstants.Contract.contract.Transact(opts, method, params...)
}

// ViewStorageAccessibleMetaData contains all meta data concerning the ViewStorageAccessible contract.
var ViewStorageAccessibleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"targetContract\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"calldataPayload\",\"type\":\"bytes\"}],\"name\":\"simulate\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// ViewStorageAccessibleABI is the input ABI used to generate the binding from.
// Deprecated: Use ViewStorageAccessibleMetaData.ABI instead.
var ViewStorageAccessibleABI = ViewStorageAccessibleMetaData.ABI

// ViewStorageAccessible is an auto generated Go binding around an Ethereum contract.
type ViewStorageAccessible struct {
	ViewStorageAccessibleCaller     // Read-only binding to the contract
	ViewStorageAccessibleTransactor // Write-only binding to the contract
	ViewStorageAccessibleFilterer   // Log filterer for contract events
}

// ViewStorageAccessibleCaller is an auto generated read-only Go binding around an Ethereum contract.
type ViewStorageAccessibleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ViewStorageAccessibleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ViewStorageAccessibleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ViewStorageAccessibleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ViewStorageAccessibleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ViewStorageAccessibleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ViewStorageAccessibleSession struct {
	Contract     *ViewStorageAccessible // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// ViewStorageAccessibleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ViewStorageAccessibleCallerSession struct {
	Contract *ViewStorageAccessibleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// ViewStorageAccessibleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ViewStorageAccessibleTransactorSession struct {
	Contract     *ViewStorageAccessibleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// ViewStorageAccessibleRaw is an auto generated low-level Go binding around an Ethereum contract.
type ViewStorageAccessibleRaw struct {
	Contract *ViewStorageAccessible // Generic contract binding to access the raw methods on
}

// ViewStorageAccessibleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ViewStorageAccessibleCallerRaw struct {
	Contract *ViewStorageAccessibleCaller // Generic read-only contract binding to access the raw methods on
}

// ViewStorageAccessibleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ViewStorageAccessibleTransactorRaw struct {
	Contract *ViewStorageAccessibleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewViewStorageAccessible creates a new instance of ViewStorageAccessible, bound to a specific deployed contract.
func NewViewStorageAccessible(address common.Address, backend bind.ContractBackend) (*ViewStorageAccessible, error) {
	contract, err := bindViewStorageAccessible(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ViewStorageAccessible{ViewStorageAccessibleCaller: ViewStorageAccessibleCaller{contract: contract}, ViewStorageAccessibleTransactor: ViewStorageAccessibleTransactor{contract: contract}, ViewStorageAccessibleFilterer: ViewStorageAccessibleFilterer{contract: contract}}, nil
}

// NewViewStorageAccessibleCaller creates a new read-only instance of ViewStorageAccessible, bound to a specific deployed contract.
func NewViewStorageAccessibleCaller(address common.Address, caller bind.ContractCaller) (*ViewStorageAccessibleCaller, error) {
	contract, err := bindViewStorageAccessible(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ViewStorageAccessibleCaller{contract: contract}, nil
}

// NewViewStorageAccessibleTransactor creates a new write-only instance of ViewStorageAccessible, bound to a specific deployed contract.
func NewViewStorageAccessibleTransactor(address common.Address, transactor bind.ContractTransactor) (*ViewStorageAccessibleTransactor, error) {
	contract, err := bindViewStorageAccessible(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ViewStorageAccessibleTransactor{contract: contract}, nil
}

// NewViewStorageAccessibleFilterer creates a new log filterer instance of ViewStorageAccessible, bound to a specific deployed contract.
func NewViewStorageAccessibleFilterer(address common.Address, filterer bind.ContractFilterer) (*ViewStorageAccessibleFilterer, error) {
	contract, err := bindViewStorageAccessible(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ViewStorageAccessibleFilterer{contract: contract}, nil
}

// bindViewStorageAccessible binds a generic wrapper to an already deployed contract.
func bindViewStorageAccessible(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ViewStorageAccessibleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ViewStorageAccessible *ViewStorageAccessibleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ViewStorageAccessible.Contract.ViewStorageAccessibleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ViewStorageAccessible *ViewStorageAccessibleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ViewStorageAccessible.Contract.ViewStorageAccessibleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ViewStorageAccessible *ViewStorageAccessibleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ViewStorageAccessible.Contract.ViewStorageAccessibleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ViewStorageAccessible *ViewStorageAccessibleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ViewStorageAccessible.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ViewStorageAccessible *ViewStorageAccessibleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ViewStorageAccessible.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ViewStorageAccessible *ViewStorageAccessibleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ViewStorageAccessible.Contract.contract.Transact(opts, method, params...)
}

// Simulate is a free data retrieval call binding the contract method 0xbd61951d.
//
// Solidity: function simulate(address targetContract, bytes calldataPayload) view returns(bytes)
func (_ViewStorageAccessible *ViewStorageAccessibleCaller) Simulate(opts *bind.CallOpts, targetContract common.Address, calldataPayload []byte) ([]byte, error) {
	var out []interface{}
	err := _ViewStorageAccessible.contract.Call(opts, &out, "simulate", targetContract, calldataPayload)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// Simulate is a free data retrieval call binding the contract method 0xbd61951d.
//
// Solidity: function simulate(address targetContract, bytes calldataPayload) view returns(bytes)
func (_ViewStorageAccessible *ViewStorageAccessibleSession) Simulate(targetContract common.Address, calldataPayload []byte) ([]byte, error) {
	return _ViewStorageAccessible.Contract.Simulate(&_ViewStorageAccessible.CallOpts, targetContract, calldataPayload)
}

// Simulate is a free data retrieval call binding the contract method 0xbd61951d.
//
// Solidity: function simulate(address targetContract, bytes calldataPayload) view returns(bytes)
func (_ViewStorageAccessible *ViewStorageAccessibleCallerSession) Simulate(targetContract common.Address, calldataPayload []byte) ([]byte, error) {
	return _ViewStorageAccessible.Contract.Simulate(&_ViewStorageAccessible.CallOpts, targetContract, calldataPayload)
}
