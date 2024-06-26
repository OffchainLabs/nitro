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

// IFallbackManagerMetaData contains all meta data concerning the IFallbackManager contract.
var IFallbackManagerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"handler\",\"type\":\"address\"}],\"name\":\"ChangedFallbackHandler\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"handler\",\"type\":\"address\"}],\"name\":\"setFallbackHandler\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IFallbackManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use IFallbackManagerMetaData.ABI instead.
var IFallbackManagerABI = IFallbackManagerMetaData.ABI

// IFallbackManager is an auto generated Go binding around an Ethereum contract.
type IFallbackManager struct {
	IFallbackManagerCaller     // Read-only binding to the contract
	IFallbackManagerTransactor // Write-only binding to the contract
	IFallbackManagerFilterer   // Log filterer for contract events
}

// IFallbackManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type IFallbackManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IFallbackManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IFallbackManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IFallbackManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IFallbackManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IFallbackManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IFallbackManagerSession struct {
	Contract     *IFallbackManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IFallbackManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IFallbackManagerCallerSession struct {
	Contract *IFallbackManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// IFallbackManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IFallbackManagerTransactorSession struct {
	Contract     *IFallbackManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// IFallbackManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type IFallbackManagerRaw struct {
	Contract *IFallbackManager // Generic contract binding to access the raw methods on
}

// IFallbackManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IFallbackManagerCallerRaw struct {
	Contract *IFallbackManagerCaller // Generic read-only contract binding to access the raw methods on
}

// IFallbackManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IFallbackManagerTransactorRaw struct {
	Contract *IFallbackManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIFallbackManager creates a new instance of IFallbackManager, bound to a specific deployed contract.
func NewIFallbackManager(address common.Address, backend bind.ContractBackend) (*IFallbackManager, error) {
	contract, err := bindIFallbackManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IFallbackManager{IFallbackManagerCaller: IFallbackManagerCaller{contract: contract}, IFallbackManagerTransactor: IFallbackManagerTransactor{contract: contract}, IFallbackManagerFilterer: IFallbackManagerFilterer{contract: contract}}, nil
}

// NewIFallbackManagerCaller creates a new read-only instance of IFallbackManager, bound to a specific deployed contract.
func NewIFallbackManagerCaller(address common.Address, caller bind.ContractCaller) (*IFallbackManagerCaller, error) {
	contract, err := bindIFallbackManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IFallbackManagerCaller{contract: contract}, nil
}

// NewIFallbackManagerTransactor creates a new write-only instance of IFallbackManager, bound to a specific deployed contract.
func NewIFallbackManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*IFallbackManagerTransactor, error) {
	contract, err := bindIFallbackManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IFallbackManagerTransactor{contract: contract}, nil
}

// NewIFallbackManagerFilterer creates a new log filterer instance of IFallbackManager, bound to a specific deployed contract.
func NewIFallbackManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*IFallbackManagerFilterer, error) {
	contract, err := bindIFallbackManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IFallbackManagerFilterer{contract: contract}, nil
}

// bindIFallbackManager binds a generic wrapper to an already deployed contract.
func bindIFallbackManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IFallbackManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IFallbackManager *IFallbackManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IFallbackManager.Contract.IFallbackManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IFallbackManager *IFallbackManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IFallbackManager.Contract.IFallbackManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IFallbackManager *IFallbackManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IFallbackManager.Contract.IFallbackManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IFallbackManager *IFallbackManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IFallbackManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IFallbackManager *IFallbackManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IFallbackManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IFallbackManager *IFallbackManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IFallbackManager.Contract.contract.Transact(opts, method, params...)
}

// SetFallbackHandler is a paid mutator transaction binding the contract method 0xf08a0323.
//
// Solidity: function setFallbackHandler(address handler) returns()
func (_IFallbackManager *IFallbackManagerTransactor) SetFallbackHandler(opts *bind.TransactOpts, handler common.Address) (*types.Transaction, error) {
	return _IFallbackManager.contract.Transact(opts, "setFallbackHandler", handler)
}

// SetFallbackHandler is a paid mutator transaction binding the contract method 0xf08a0323.
//
// Solidity: function setFallbackHandler(address handler) returns()
func (_IFallbackManager *IFallbackManagerSession) SetFallbackHandler(handler common.Address) (*types.Transaction, error) {
	return _IFallbackManager.Contract.SetFallbackHandler(&_IFallbackManager.TransactOpts, handler)
}

// SetFallbackHandler is a paid mutator transaction binding the contract method 0xf08a0323.
//
// Solidity: function setFallbackHandler(address handler) returns()
func (_IFallbackManager *IFallbackManagerTransactorSession) SetFallbackHandler(handler common.Address) (*types.Transaction, error) {
	return _IFallbackManager.Contract.SetFallbackHandler(&_IFallbackManager.TransactOpts, handler)
}

// IFallbackManagerChangedFallbackHandlerIterator is returned from FilterChangedFallbackHandler and is used to iterate over the raw logs and unpacked data for ChangedFallbackHandler events raised by the IFallbackManager contract.
type IFallbackManagerChangedFallbackHandlerIterator struct {
	Event *IFallbackManagerChangedFallbackHandler // Event containing the contract specifics and raw log

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
func (it *IFallbackManagerChangedFallbackHandlerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IFallbackManagerChangedFallbackHandler)
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
		it.Event = new(IFallbackManagerChangedFallbackHandler)
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
func (it *IFallbackManagerChangedFallbackHandlerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IFallbackManagerChangedFallbackHandlerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IFallbackManagerChangedFallbackHandler represents a ChangedFallbackHandler event raised by the IFallbackManager contract.
type IFallbackManagerChangedFallbackHandler struct {
	Handler common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterChangedFallbackHandler is a free log retrieval operation binding the contract event 0x5ac6c46c93c8d0e53714ba3b53db3e7c046da994313d7ed0d192028bc7c228b0.
//
// Solidity: event ChangedFallbackHandler(address indexed handler)
func (_IFallbackManager *IFallbackManagerFilterer) FilterChangedFallbackHandler(opts *bind.FilterOpts, handler []common.Address) (*IFallbackManagerChangedFallbackHandlerIterator, error) {

	var handlerRule []interface{}
	for _, handlerItem := range handler {
		handlerRule = append(handlerRule, handlerItem)
	}

	logs, sub, err := _IFallbackManager.contract.FilterLogs(opts, "ChangedFallbackHandler", handlerRule)
	if err != nil {
		return nil, err
	}
	return &IFallbackManagerChangedFallbackHandlerIterator{contract: _IFallbackManager.contract, event: "ChangedFallbackHandler", logs: logs, sub: sub}, nil
}

// WatchChangedFallbackHandler is a free log subscription operation binding the contract event 0x5ac6c46c93c8d0e53714ba3b53db3e7c046da994313d7ed0d192028bc7c228b0.
//
// Solidity: event ChangedFallbackHandler(address indexed handler)
func (_IFallbackManager *IFallbackManagerFilterer) WatchChangedFallbackHandler(opts *bind.WatchOpts, sink chan<- *IFallbackManagerChangedFallbackHandler, handler []common.Address) (event.Subscription, error) {

	var handlerRule []interface{}
	for _, handlerItem := range handler {
		handlerRule = append(handlerRule, handlerItem)
	}

	logs, sub, err := _IFallbackManager.contract.WatchLogs(opts, "ChangedFallbackHandler", handlerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IFallbackManagerChangedFallbackHandler)
				if err := _IFallbackManager.contract.UnpackLog(event, "ChangedFallbackHandler", log); err != nil {
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
func (_IFallbackManager *IFallbackManagerFilterer) ParseChangedFallbackHandler(log types.Log) (*IFallbackManagerChangedFallbackHandler, error) {
	event := new(IFallbackManagerChangedFallbackHandler)
	if err := _IFallbackManager.contract.UnpackLog(event, "ChangedFallbackHandler", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IGuardManagerMetaData contains all meta data concerning the IGuardManager contract.
var IGuardManagerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"guard\",\"type\":\"address\"}],\"name\":\"ChangedGuard\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"guard\",\"type\":\"address\"}],\"name\":\"setGuard\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IGuardManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use IGuardManagerMetaData.ABI instead.
var IGuardManagerABI = IGuardManagerMetaData.ABI

// IGuardManager is an auto generated Go binding around an Ethereum contract.
type IGuardManager struct {
	IGuardManagerCaller     // Read-only binding to the contract
	IGuardManagerTransactor // Write-only binding to the contract
	IGuardManagerFilterer   // Log filterer for contract events
}

// IGuardManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type IGuardManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IGuardManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IGuardManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IGuardManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IGuardManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IGuardManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IGuardManagerSession struct {
	Contract     *IGuardManager    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IGuardManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IGuardManagerCallerSession struct {
	Contract *IGuardManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// IGuardManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IGuardManagerTransactorSession struct {
	Contract     *IGuardManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// IGuardManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type IGuardManagerRaw struct {
	Contract *IGuardManager // Generic contract binding to access the raw methods on
}

// IGuardManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IGuardManagerCallerRaw struct {
	Contract *IGuardManagerCaller // Generic read-only contract binding to access the raw methods on
}

// IGuardManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IGuardManagerTransactorRaw struct {
	Contract *IGuardManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIGuardManager creates a new instance of IGuardManager, bound to a specific deployed contract.
func NewIGuardManager(address common.Address, backend bind.ContractBackend) (*IGuardManager, error) {
	contract, err := bindIGuardManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IGuardManager{IGuardManagerCaller: IGuardManagerCaller{contract: contract}, IGuardManagerTransactor: IGuardManagerTransactor{contract: contract}, IGuardManagerFilterer: IGuardManagerFilterer{contract: contract}}, nil
}

// NewIGuardManagerCaller creates a new read-only instance of IGuardManager, bound to a specific deployed contract.
func NewIGuardManagerCaller(address common.Address, caller bind.ContractCaller) (*IGuardManagerCaller, error) {
	contract, err := bindIGuardManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IGuardManagerCaller{contract: contract}, nil
}

// NewIGuardManagerTransactor creates a new write-only instance of IGuardManager, bound to a specific deployed contract.
func NewIGuardManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*IGuardManagerTransactor, error) {
	contract, err := bindIGuardManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IGuardManagerTransactor{contract: contract}, nil
}

// NewIGuardManagerFilterer creates a new log filterer instance of IGuardManager, bound to a specific deployed contract.
func NewIGuardManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*IGuardManagerFilterer, error) {
	contract, err := bindIGuardManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IGuardManagerFilterer{contract: contract}, nil
}

// bindIGuardManager binds a generic wrapper to an already deployed contract.
func bindIGuardManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IGuardManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IGuardManager *IGuardManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IGuardManager.Contract.IGuardManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IGuardManager *IGuardManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IGuardManager.Contract.IGuardManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IGuardManager *IGuardManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IGuardManager.Contract.IGuardManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IGuardManager *IGuardManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IGuardManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IGuardManager *IGuardManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IGuardManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IGuardManager *IGuardManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IGuardManager.Contract.contract.Transact(opts, method, params...)
}

// SetGuard is a paid mutator transaction binding the contract method 0xe19a9dd9.
//
// Solidity: function setGuard(address guard) returns()
func (_IGuardManager *IGuardManagerTransactor) SetGuard(opts *bind.TransactOpts, guard common.Address) (*types.Transaction, error) {
	return _IGuardManager.contract.Transact(opts, "setGuard", guard)
}

// SetGuard is a paid mutator transaction binding the contract method 0xe19a9dd9.
//
// Solidity: function setGuard(address guard) returns()
func (_IGuardManager *IGuardManagerSession) SetGuard(guard common.Address) (*types.Transaction, error) {
	return _IGuardManager.Contract.SetGuard(&_IGuardManager.TransactOpts, guard)
}

// SetGuard is a paid mutator transaction binding the contract method 0xe19a9dd9.
//
// Solidity: function setGuard(address guard) returns()
func (_IGuardManager *IGuardManagerTransactorSession) SetGuard(guard common.Address) (*types.Transaction, error) {
	return _IGuardManager.Contract.SetGuard(&_IGuardManager.TransactOpts, guard)
}

// IGuardManagerChangedGuardIterator is returned from FilterChangedGuard and is used to iterate over the raw logs and unpacked data for ChangedGuard events raised by the IGuardManager contract.
type IGuardManagerChangedGuardIterator struct {
	Event *IGuardManagerChangedGuard // Event containing the contract specifics and raw log

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
func (it *IGuardManagerChangedGuardIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IGuardManagerChangedGuard)
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
		it.Event = new(IGuardManagerChangedGuard)
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
func (it *IGuardManagerChangedGuardIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IGuardManagerChangedGuardIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IGuardManagerChangedGuard represents a ChangedGuard event raised by the IGuardManager contract.
type IGuardManagerChangedGuard struct {
	Guard common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterChangedGuard is a free log retrieval operation binding the contract event 0x1151116914515bc0891ff9047a6cb32cf902546f83066499bcf8ba33d2353fa2.
//
// Solidity: event ChangedGuard(address indexed guard)
func (_IGuardManager *IGuardManagerFilterer) FilterChangedGuard(opts *bind.FilterOpts, guard []common.Address) (*IGuardManagerChangedGuardIterator, error) {

	var guardRule []interface{}
	for _, guardItem := range guard {
		guardRule = append(guardRule, guardItem)
	}

	logs, sub, err := _IGuardManager.contract.FilterLogs(opts, "ChangedGuard", guardRule)
	if err != nil {
		return nil, err
	}
	return &IGuardManagerChangedGuardIterator{contract: _IGuardManager.contract, event: "ChangedGuard", logs: logs, sub: sub}, nil
}

// WatchChangedGuard is a free log subscription operation binding the contract event 0x1151116914515bc0891ff9047a6cb32cf902546f83066499bcf8ba33d2353fa2.
//
// Solidity: event ChangedGuard(address indexed guard)
func (_IGuardManager *IGuardManagerFilterer) WatchChangedGuard(opts *bind.WatchOpts, sink chan<- *IGuardManagerChangedGuard, guard []common.Address) (event.Subscription, error) {

	var guardRule []interface{}
	for _, guardItem := range guard {
		guardRule = append(guardRule, guardItem)
	}

	logs, sub, err := _IGuardManager.contract.WatchLogs(opts, "ChangedGuard", guardRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IGuardManagerChangedGuard)
				if err := _IGuardManager.contract.UnpackLog(event, "ChangedGuard", log); err != nil {
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
func (_IGuardManager *IGuardManagerFilterer) ParseChangedGuard(log types.Log) (*IGuardManagerChangedGuard, error) {
	event := new(IGuardManagerChangedGuard)
	if err := _IGuardManager.contract.UnpackLog(event, "ChangedGuard", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IModuleManagerMetaData contains all meta data concerning the IModuleManager contract.
var IModuleManagerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"moduleGuard\",\"type\":\"address\"}],\"name\":\"ChangedModuleGuard\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"DisabledModule\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"EnabledModule\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"ExecutionFromModuleFailure\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"ExecutionFromModuleSuccess\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"prevModule\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"disableModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"enableModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"}],\"name\":\"execTransactionFromModule\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"}],\"name\":\"execTransactionFromModuleReturnData\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"start\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"pageSize\",\"type\":\"uint256\"}],\"name\":\"getModulesPaginated\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"array\",\"type\":\"address[]\"},{\"internalType\":\"address\",\"name\":\"next\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"isModuleEnabled\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"moduleGuard\",\"type\":\"address\"}],\"name\":\"setModuleGuard\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IModuleManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use IModuleManagerMetaData.ABI instead.
var IModuleManagerABI = IModuleManagerMetaData.ABI

// IModuleManager is an auto generated Go binding around an Ethereum contract.
type IModuleManager struct {
	IModuleManagerCaller     // Read-only binding to the contract
	IModuleManagerTransactor // Write-only binding to the contract
	IModuleManagerFilterer   // Log filterer for contract events
}

// IModuleManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type IModuleManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IModuleManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IModuleManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IModuleManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IModuleManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IModuleManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IModuleManagerSession struct {
	Contract     *IModuleManager   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IModuleManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IModuleManagerCallerSession struct {
	Contract *IModuleManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// IModuleManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IModuleManagerTransactorSession struct {
	Contract     *IModuleManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// IModuleManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type IModuleManagerRaw struct {
	Contract *IModuleManager // Generic contract binding to access the raw methods on
}

// IModuleManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IModuleManagerCallerRaw struct {
	Contract *IModuleManagerCaller // Generic read-only contract binding to access the raw methods on
}

// IModuleManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IModuleManagerTransactorRaw struct {
	Contract *IModuleManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIModuleManager creates a new instance of IModuleManager, bound to a specific deployed contract.
func NewIModuleManager(address common.Address, backend bind.ContractBackend) (*IModuleManager, error) {
	contract, err := bindIModuleManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IModuleManager{IModuleManagerCaller: IModuleManagerCaller{contract: contract}, IModuleManagerTransactor: IModuleManagerTransactor{contract: contract}, IModuleManagerFilterer: IModuleManagerFilterer{contract: contract}}, nil
}

// NewIModuleManagerCaller creates a new read-only instance of IModuleManager, bound to a specific deployed contract.
func NewIModuleManagerCaller(address common.Address, caller bind.ContractCaller) (*IModuleManagerCaller, error) {
	contract, err := bindIModuleManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IModuleManagerCaller{contract: contract}, nil
}

// NewIModuleManagerTransactor creates a new write-only instance of IModuleManager, bound to a specific deployed contract.
func NewIModuleManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*IModuleManagerTransactor, error) {
	contract, err := bindIModuleManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IModuleManagerTransactor{contract: contract}, nil
}

// NewIModuleManagerFilterer creates a new log filterer instance of IModuleManager, bound to a specific deployed contract.
func NewIModuleManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*IModuleManagerFilterer, error) {
	contract, err := bindIModuleManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IModuleManagerFilterer{contract: contract}, nil
}

// bindIModuleManager binds a generic wrapper to an already deployed contract.
func bindIModuleManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IModuleManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IModuleManager *IModuleManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IModuleManager.Contract.IModuleManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IModuleManager *IModuleManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IModuleManager.Contract.IModuleManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IModuleManager *IModuleManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IModuleManager.Contract.IModuleManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IModuleManager *IModuleManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IModuleManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IModuleManager *IModuleManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IModuleManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IModuleManager *IModuleManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IModuleManager.Contract.contract.Transact(opts, method, params...)
}

// GetModulesPaginated is a free data retrieval call binding the contract method 0xcc2f8452.
//
// Solidity: function getModulesPaginated(address start, uint256 pageSize) view returns(address[] array, address next)
func (_IModuleManager *IModuleManagerCaller) GetModulesPaginated(opts *bind.CallOpts, start common.Address, pageSize *big.Int) (struct {
	Array []common.Address
	Next  common.Address
}, error) {
	var out []interface{}
	err := _IModuleManager.contract.Call(opts, &out, "getModulesPaginated", start, pageSize)

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
func (_IModuleManager *IModuleManagerSession) GetModulesPaginated(start common.Address, pageSize *big.Int) (struct {
	Array []common.Address
	Next  common.Address
}, error) {
	return _IModuleManager.Contract.GetModulesPaginated(&_IModuleManager.CallOpts, start, pageSize)
}

// GetModulesPaginated is a free data retrieval call binding the contract method 0xcc2f8452.
//
// Solidity: function getModulesPaginated(address start, uint256 pageSize) view returns(address[] array, address next)
func (_IModuleManager *IModuleManagerCallerSession) GetModulesPaginated(start common.Address, pageSize *big.Int) (struct {
	Array []common.Address
	Next  common.Address
}, error) {
	return _IModuleManager.Contract.GetModulesPaginated(&_IModuleManager.CallOpts, start, pageSize)
}

// IsModuleEnabled is a free data retrieval call binding the contract method 0x2d9ad53d.
//
// Solidity: function isModuleEnabled(address module) view returns(bool)
func (_IModuleManager *IModuleManagerCaller) IsModuleEnabled(opts *bind.CallOpts, module common.Address) (bool, error) {
	var out []interface{}
	err := _IModuleManager.contract.Call(opts, &out, "isModuleEnabled", module)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsModuleEnabled is a free data retrieval call binding the contract method 0x2d9ad53d.
//
// Solidity: function isModuleEnabled(address module) view returns(bool)
func (_IModuleManager *IModuleManagerSession) IsModuleEnabled(module common.Address) (bool, error) {
	return _IModuleManager.Contract.IsModuleEnabled(&_IModuleManager.CallOpts, module)
}

// IsModuleEnabled is a free data retrieval call binding the contract method 0x2d9ad53d.
//
// Solidity: function isModuleEnabled(address module) view returns(bool)
func (_IModuleManager *IModuleManagerCallerSession) IsModuleEnabled(module common.Address) (bool, error) {
	return _IModuleManager.Contract.IsModuleEnabled(&_IModuleManager.CallOpts, module)
}

// DisableModule is a paid mutator transaction binding the contract method 0xe009cfde.
//
// Solidity: function disableModule(address prevModule, address module) returns()
func (_IModuleManager *IModuleManagerTransactor) DisableModule(opts *bind.TransactOpts, prevModule common.Address, module common.Address) (*types.Transaction, error) {
	return _IModuleManager.contract.Transact(opts, "disableModule", prevModule, module)
}

// DisableModule is a paid mutator transaction binding the contract method 0xe009cfde.
//
// Solidity: function disableModule(address prevModule, address module) returns()
func (_IModuleManager *IModuleManagerSession) DisableModule(prevModule common.Address, module common.Address) (*types.Transaction, error) {
	return _IModuleManager.Contract.DisableModule(&_IModuleManager.TransactOpts, prevModule, module)
}

// DisableModule is a paid mutator transaction binding the contract method 0xe009cfde.
//
// Solidity: function disableModule(address prevModule, address module) returns()
func (_IModuleManager *IModuleManagerTransactorSession) DisableModule(prevModule common.Address, module common.Address) (*types.Transaction, error) {
	return _IModuleManager.Contract.DisableModule(&_IModuleManager.TransactOpts, prevModule, module)
}

// EnableModule is a paid mutator transaction binding the contract method 0x610b5925.
//
// Solidity: function enableModule(address module) returns()
func (_IModuleManager *IModuleManagerTransactor) EnableModule(opts *bind.TransactOpts, module common.Address) (*types.Transaction, error) {
	return _IModuleManager.contract.Transact(opts, "enableModule", module)
}

// EnableModule is a paid mutator transaction binding the contract method 0x610b5925.
//
// Solidity: function enableModule(address module) returns()
func (_IModuleManager *IModuleManagerSession) EnableModule(module common.Address) (*types.Transaction, error) {
	return _IModuleManager.Contract.EnableModule(&_IModuleManager.TransactOpts, module)
}

// EnableModule is a paid mutator transaction binding the contract method 0x610b5925.
//
// Solidity: function enableModule(address module) returns()
func (_IModuleManager *IModuleManagerTransactorSession) EnableModule(module common.Address) (*types.Transaction, error) {
	return _IModuleManager.Contract.EnableModule(&_IModuleManager.TransactOpts, module)
}

// ExecTransactionFromModule is a paid mutator transaction binding the contract method 0x468721a7.
//
// Solidity: function execTransactionFromModule(address to, uint256 value, bytes data, uint8 operation) returns(bool success)
func (_IModuleManager *IModuleManagerTransactor) ExecTransactionFromModule(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _IModuleManager.contract.Transact(opts, "execTransactionFromModule", to, value, data, operation)
}

// ExecTransactionFromModule is a paid mutator transaction binding the contract method 0x468721a7.
//
// Solidity: function execTransactionFromModule(address to, uint256 value, bytes data, uint8 operation) returns(bool success)
func (_IModuleManager *IModuleManagerSession) ExecTransactionFromModule(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _IModuleManager.Contract.ExecTransactionFromModule(&_IModuleManager.TransactOpts, to, value, data, operation)
}

// ExecTransactionFromModule is a paid mutator transaction binding the contract method 0x468721a7.
//
// Solidity: function execTransactionFromModule(address to, uint256 value, bytes data, uint8 operation) returns(bool success)
func (_IModuleManager *IModuleManagerTransactorSession) ExecTransactionFromModule(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _IModuleManager.Contract.ExecTransactionFromModule(&_IModuleManager.TransactOpts, to, value, data, operation)
}

// ExecTransactionFromModuleReturnData is a paid mutator transaction binding the contract method 0x5229073f.
//
// Solidity: function execTransactionFromModuleReturnData(address to, uint256 value, bytes data, uint8 operation) returns(bool success, bytes returnData)
func (_IModuleManager *IModuleManagerTransactor) ExecTransactionFromModuleReturnData(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _IModuleManager.contract.Transact(opts, "execTransactionFromModuleReturnData", to, value, data, operation)
}

// ExecTransactionFromModuleReturnData is a paid mutator transaction binding the contract method 0x5229073f.
//
// Solidity: function execTransactionFromModuleReturnData(address to, uint256 value, bytes data, uint8 operation) returns(bool success, bytes returnData)
func (_IModuleManager *IModuleManagerSession) ExecTransactionFromModuleReturnData(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _IModuleManager.Contract.ExecTransactionFromModuleReturnData(&_IModuleManager.TransactOpts, to, value, data, operation)
}

// ExecTransactionFromModuleReturnData is a paid mutator transaction binding the contract method 0x5229073f.
//
// Solidity: function execTransactionFromModuleReturnData(address to, uint256 value, bytes data, uint8 operation) returns(bool success, bytes returnData)
func (_IModuleManager *IModuleManagerTransactorSession) ExecTransactionFromModuleReturnData(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _IModuleManager.Contract.ExecTransactionFromModuleReturnData(&_IModuleManager.TransactOpts, to, value, data, operation)
}

// SetModuleGuard is a paid mutator transaction binding the contract method 0xe068df37.
//
// Solidity: function setModuleGuard(address moduleGuard) returns()
func (_IModuleManager *IModuleManagerTransactor) SetModuleGuard(opts *bind.TransactOpts, moduleGuard common.Address) (*types.Transaction, error) {
	return _IModuleManager.contract.Transact(opts, "setModuleGuard", moduleGuard)
}

// SetModuleGuard is a paid mutator transaction binding the contract method 0xe068df37.
//
// Solidity: function setModuleGuard(address moduleGuard) returns()
func (_IModuleManager *IModuleManagerSession) SetModuleGuard(moduleGuard common.Address) (*types.Transaction, error) {
	return _IModuleManager.Contract.SetModuleGuard(&_IModuleManager.TransactOpts, moduleGuard)
}

// SetModuleGuard is a paid mutator transaction binding the contract method 0xe068df37.
//
// Solidity: function setModuleGuard(address moduleGuard) returns()
func (_IModuleManager *IModuleManagerTransactorSession) SetModuleGuard(moduleGuard common.Address) (*types.Transaction, error) {
	return _IModuleManager.Contract.SetModuleGuard(&_IModuleManager.TransactOpts, moduleGuard)
}

// IModuleManagerChangedModuleGuardIterator is returned from FilterChangedModuleGuard and is used to iterate over the raw logs and unpacked data for ChangedModuleGuard events raised by the IModuleManager contract.
type IModuleManagerChangedModuleGuardIterator struct {
	Event *IModuleManagerChangedModuleGuard // Event containing the contract specifics and raw log

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
func (it *IModuleManagerChangedModuleGuardIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IModuleManagerChangedModuleGuard)
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
		it.Event = new(IModuleManagerChangedModuleGuard)
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
func (it *IModuleManagerChangedModuleGuardIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IModuleManagerChangedModuleGuardIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IModuleManagerChangedModuleGuard represents a ChangedModuleGuard event raised by the IModuleManager contract.
type IModuleManagerChangedModuleGuard struct {
	ModuleGuard common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChangedModuleGuard is a free log retrieval operation binding the contract event 0xcd1966d6be16bc0c030cc741a06c6e0efaf8d00de2c8b6a9e11827e125de8bb8.
//
// Solidity: event ChangedModuleGuard(address indexed moduleGuard)
func (_IModuleManager *IModuleManagerFilterer) FilterChangedModuleGuard(opts *bind.FilterOpts, moduleGuard []common.Address) (*IModuleManagerChangedModuleGuardIterator, error) {

	var moduleGuardRule []interface{}
	for _, moduleGuardItem := range moduleGuard {
		moduleGuardRule = append(moduleGuardRule, moduleGuardItem)
	}

	logs, sub, err := _IModuleManager.contract.FilterLogs(opts, "ChangedModuleGuard", moduleGuardRule)
	if err != nil {
		return nil, err
	}
	return &IModuleManagerChangedModuleGuardIterator{contract: _IModuleManager.contract, event: "ChangedModuleGuard", logs: logs, sub: sub}, nil
}

// WatchChangedModuleGuard is a free log subscription operation binding the contract event 0xcd1966d6be16bc0c030cc741a06c6e0efaf8d00de2c8b6a9e11827e125de8bb8.
//
// Solidity: event ChangedModuleGuard(address indexed moduleGuard)
func (_IModuleManager *IModuleManagerFilterer) WatchChangedModuleGuard(opts *bind.WatchOpts, sink chan<- *IModuleManagerChangedModuleGuard, moduleGuard []common.Address) (event.Subscription, error) {

	var moduleGuardRule []interface{}
	for _, moduleGuardItem := range moduleGuard {
		moduleGuardRule = append(moduleGuardRule, moduleGuardItem)
	}

	logs, sub, err := _IModuleManager.contract.WatchLogs(opts, "ChangedModuleGuard", moduleGuardRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IModuleManagerChangedModuleGuard)
				if err := _IModuleManager.contract.UnpackLog(event, "ChangedModuleGuard", log); err != nil {
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
func (_IModuleManager *IModuleManagerFilterer) ParseChangedModuleGuard(log types.Log) (*IModuleManagerChangedModuleGuard, error) {
	event := new(IModuleManagerChangedModuleGuard)
	if err := _IModuleManager.contract.UnpackLog(event, "ChangedModuleGuard", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IModuleManagerDisabledModuleIterator is returned from FilterDisabledModule and is used to iterate over the raw logs and unpacked data for DisabledModule events raised by the IModuleManager contract.
type IModuleManagerDisabledModuleIterator struct {
	Event *IModuleManagerDisabledModule // Event containing the contract specifics and raw log

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
func (it *IModuleManagerDisabledModuleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IModuleManagerDisabledModule)
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
		it.Event = new(IModuleManagerDisabledModule)
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
func (it *IModuleManagerDisabledModuleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IModuleManagerDisabledModuleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IModuleManagerDisabledModule represents a DisabledModule event raised by the IModuleManager contract.
type IModuleManagerDisabledModule struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterDisabledModule is a free log retrieval operation binding the contract event 0xaab4fa2b463f581b2b32cb3b7e3b704b9ce37cc209b5fb4d77e593ace4054276.
//
// Solidity: event DisabledModule(address indexed module)
func (_IModuleManager *IModuleManagerFilterer) FilterDisabledModule(opts *bind.FilterOpts, module []common.Address) (*IModuleManagerDisabledModuleIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _IModuleManager.contract.FilterLogs(opts, "DisabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return &IModuleManagerDisabledModuleIterator{contract: _IModuleManager.contract, event: "DisabledModule", logs: logs, sub: sub}, nil
}

// WatchDisabledModule is a free log subscription operation binding the contract event 0xaab4fa2b463f581b2b32cb3b7e3b704b9ce37cc209b5fb4d77e593ace4054276.
//
// Solidity: event DisabledModule(address indexed module)
func (_IModuleManager *IModuleManagerFilterer) WatchDisabledModule(opts *bind.WatchOpts, sink chan<- *IModuleManagerDisabledModule, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _IModuleManager.contract.WatchLogs(opts, "DisabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IModuleManagerDisabledModule)
				if err := _IModuleManager.contract.UnpackLog(event, "DisabledModule", log); err != nil {
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
func (_IModuleManager *IModuleManagerFilterer) ParseDisabledModule(log types.Log) (*IModuleManagerDisabledModule, error) {
	event := new(IModuleManagerDisabledModule)
	if err := _IModuleManager.contract.UnpackLog(event, "DisabledModule", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IModuleManagerEnabledModuleIterator is returned from FilterEnabledModule and is used to iterate over the raw logs and unpacked data for EnabledModule events raised by the IModuleManager contract.
type IModuleManagerEnabledModuleIterator struct {
	Event *IModuleManagerEnabledModule // Event containing the contract specifics and raw log

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
func (it *IModuleManagerEnabledModuleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IModuleManagerEnabledModule)
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
		it.Event = new(IModuleManagerEnabledModule)
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
func (it *IModuleManagerEnabledModuleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IModuleManagerEnabledModuleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IModuleManagerEnabledModule represents a EnabledModule event raised by the IModuleManager contract.
type IModuleManagerEnabledModule struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterEnabledModule is a free log retrieval operation binding the contract event 0xecdf3a3effea5783a3c4c2140e677577666428d44ed9d474a0b3a4c9943f8440.
//
// Solidity: event EnabledModule(address indexed module)
func (_IModuleManager *IModuleManagerFilterer) FilterEnabledModule(opts *bind.FilterOpts, module []common.Address) (*IModuleManagerEnabledModuleIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _IModuleManager.contract.FilterLogs(opts, "EnabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return &IModuleManagerEnabledModuleIterator{contract: _IModuleManager.contract, event: "EnabledModule", logs: logs, sub: sub}, nil
}

// WatchEnabledModule is a free log subscription operation binding the contract event 0xecdf3a3effea5783a3c4c2140e677577666428d44ed9d474a0b3a4c9943f8440.
//
// Solidity: event EnabledModule(address indexed module)
func (_IModuleManager *IModuleManagerFilterer) WatchEnabledModule(opts *bind.WatchOpts, sink chan<- *IModuleManagerEnabledModule, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _IModuleManager.contract.WatchLogs(opts, "EnabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IModuleManagerEnabledModule)
				if err := _IModuleManager.contract.UnpackLog(event, "EnabledModule", log); err != nil {
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
func (_IModuleManager *IModuleManagerFilterer) ParseEnabledModule(log types.Log) (*IModuleManagerEnabledModule, error) {
	event := new(IModuleManagerEnabledModule)
	if err := _IModuleManager.contract.UnpackLog(event, "EnabledModule", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IModuleManagerExecutionFromModuleFailureIterator is returned from FilterExecutionFromModuleFailure and is used to iterate over the raw logs and unpacked data for ExecutionFromModuleFailure events raised by the IModuleManager contract.
type IModuleManagerExecutionFromModuleFailureIterator struct {
	Event *IModuleManagerExecutionFromModuleFailure // Event containing the contract specifics and raw log

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
func (it *IModuleManagerExecutionFromModuleFailureIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IModuleManagerExecutionFromModuleFailure)
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
		it.Event = new(IModuleManagerExecutionFromModuleFailure)
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
func (it *IModuleManagerExecutionFromModuleFailureIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IModuleManagerExecutionFromModuleFailureIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IModuleManagerExecutionFromModuleFailure represents a ExecutionFromModuleFailure event raised by the IModuleManager contract.
type IModuleManagerExecutionFromModuleFailure struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterExecutionFromModuleFailure is a free log retrieval operation binding the contract event 0xacd2c8702804128fdb0db2bb49f6d127dd0181c13fd45dbfe16de0930e2bd375.
//
// Solidity: event ExecutionFromModuleFailure(address indexed module)
func (_IModuleManager *IModuleManagerFilterer) FilterExecutionFromModuleFailure(opts *bind.FilterOpts, module []common.Address) (*IModuleManagerExecutionFromModuleFailureIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _IModuleManager.contract.FilterLogs(opts, "ExecutionFromModuleFailure", moduleRule)
	if err != nil {
		return nil, err
	}
	return &IModuleManagerExecutionFromModuleFailureIterator{contract: _IModuleManager.contract, event: "ExecutionFromModuleFailure", logs: logs, sub: sub}, nil
}

// WatchExecutionFromModuleFailure is a free log subscription operation binding the contract event 0xacd2c8702804128fdb0db2bb49f6d127dd0181c13fd45dbfe16de0930e2bd375.
//
// Solidity: event ExecutionFromModuleFailure(address indexed module)
func (_IModuleManager *IModuleManagerFilterer) WatchExecutionFromModuleFailure(opts *bind.WatchOpts, sink chan<- *IModuleManagerExecutionFromModuleFailure, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _IModuleManager.contract.WatchLogs(opts, "ExecutionFromModuleFailure", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IModuleManagerExecutionFromModuleFailure)
				if err := _IModuleManager.contract.UnpackLog(event, "ExecutionFromModuleFailure", log); err != nil {
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
func (_IModuleManager *IModuleManagerFilterer) ParseExecutionFromModuleFailure(log types.Log) (*IModuleManagerExecutionFromModuleFailure, error) {
	event := new(IModuleManagerExecutionFromModuleFailure)
	if err := _IModuleManager.contract.UnpackLog(event, "ExecutionFromModuleFailure", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IModuleManagerExecutionFromModuleSuccessIterator is returned from FilterExecutionFromModuleSuccess and is used to iterate over the raw logs and unpacked data for ExecutionFromModuleSuccess events raised by the IModuleManager contract.
type IModuleManagerExecutionFromModuleSuccessIterator struct {
	Event *IModuleManagerExecutionFromModuleSuccess // Event containing the contract specifics and raw log

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
func (it *IModuleManagerExecutionFromModuleSuccessIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IModuleManagerExecutionFromModuleSuccess)
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
		it.Event = new(IModuleManagerExecutionFromModuleSuccess)
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
func (it *IModuleManagerExecutionFromModuleSuccessIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IModuleManagerExecutionFromModuleSuccessIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IModuleManagerExecutionFromModuleSuccess represents a ExecutionFromModuleSuccess event raised by the IModuleManager contract.
type IModuleManagerExecutionFromModuleSuccess struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterExecutionFromModuleSuccess is a free log retrieval operation binding the contract event 0x6895c13664aa4f67288b25d7a21d7aaa34916e355fb9b6fae0a139a9085becb8.
//
// Solidity: event ExecutionFromModuleSuccess(address indexed module)
func (_IModuleManager *IModuleManagerFilterer) FilterExecutionFromModuleSuccess(opts *bind.FilterOpts, module []common.Address) (*IModuleManagerExecutionFromModuleSuccessIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _IModuleManager.contract.FilterLogs(opts, "ExecutionFromModuleSuccess", moduleRule)
	if err != nil {
		return nil, err
	}
	return &IModuleManagerExecutionFromModuleSuccessIterator{contract: _IModuleManager.contract, event: "ExecutionFromModuleSuccess", logs: logs, sub: sub}, nil
}

// WatchExecutionFromModuleSuccess is a free log subscription operation binding the contract event 0x6895c13664aa4f67288b25d7a21d7aaa34916e355fb9b6fae0a139a9085becb8.
//
// Solidity: event ExecutionFromModuleSuccess(address indexed module)
func (_IModuleManager *IModuleManagerFilterer) WatchExecutionFromModuleSuccess(opts *bind.WatchOpts, sink chan<- *IModuleManagerExecutionFromModuleSuccess, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _IModuleManager.contract.WatchLogs(opts, "ExecutionFromModuleSuccess", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IModuleManagerExecutionFromModuleSuccess)
				if err := _IModuleManager.contract.UnpackLog(event, "ExecutionFromModuleSuccess", log); err != nil {
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
func (_IModuleManager *IModuleManagerFilterer) ParseExecutionFromModuleSuccess(log types.Log) (*IModuleManagerExecutionFromModuleSuccess, error) {
	event := new(IModuleManagerExecutionFromModuleSuccess)
	if err := _IModuleManager.contract.UnpackLog(event, "ExecutionFromModuleSuccess", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOwnerManagerMetaData contains all meta data concerning the IOwnerManager contract.
var IOwnerManagerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"AddedOwner\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"threshold\",\"type\":\"uint256\"}],\"name\":\"ChangedThreshold\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"RemovedOwner\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"}],\"name\":\"addOwnerWithThreshold\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"}],\"name\":\"changeThreshold\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getOwners\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getThreshold\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"isOwner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"prevOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"}],\"name\":\"removeOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"prevOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"swapOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IOwnerManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use IOwnerManagerMetaData.ABI instead.
var IOwnerManagerABI = IOwnerManagerMetaData.ABI

// IOwnerManager is an auto generated Go binding around an Ethereum contract.
type IOwnerManager struct {
	IOwnerManagerCaller     // Read-only binding to the contract
	IOwnerManagerTransactor // Write-only binding to the contract
	IOwnerManagerFilterer   // Log filterer for contract events
}

// IOwnerManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type IOwnerManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOwnerManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IOwnerManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOwnerManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IOwnerManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOwnerManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IOwnerManagerSession struct {
	Contract     *IOwnerManager    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IOwnerManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IOwnerManagerCallerSession struct {
	Contract *IOwnerManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// IOwnerManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IOwnerManagerTransactorSession struct {
	Contract     *IOwnerManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// IOwnerManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type IOwnerManagerRaw struct {
	Contract *IOwnerManager // Generic contract binding to access the raw methods on
}

// IOwnerManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IOwnerManagerCallerRaw struct {
	Contract *IOwnerManagerCaller // Generic read-only contract binding to access the raw methods on
}

// IOwnerManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IOwnerManagerTransactorRaw struct {
	Contract *IOwnerManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIOwnerManager creates a new instance of IOwnerManager, bound to a specific deployed contract.
func NewIOwnerManager(address common.Address, backend bind.ContractBackend) (*IOwnerManager, error) {
	contract, err := bindIOwnerManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IOwnerManager{IOwnerManagerCaller: IOwnerManagerCaller{contract: contract}, IOwnerManagerTransactor: IOwnerManagerTransactor{contract: contract}, IOwnerManagerFilterer: IOwnerManagerFilterer{contract: contract}}, nil
}

// NewIOwnerManagerCaller creates a new read-only instance of IOwnerManager, bound to a specific deployed contract.
func NewIOwnerManagerCaller(address common.Address, caller bind.ContractCaller) (*IOwnerManagerCaller, error) {
	contract, err := bindIOwnerManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IOwnerManagerCaller{contract: contract}, nil
}

// NewIOwnerManagerTransactor creates a new write-only instance of IOwnerManager, bound to a specific deployed contract.
func NewIOwnerManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*IOwnerManagerTransactor, error) {
	contract, err := bindIOwnerManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IOwnerManagerTransactor{contract: contract}, nil
}

// NewIOwnerManagerFilterer creates a new log filterer instance of IOwnerManager, bound to a specific deployed contract.
func NewIOwnerManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*IOwnerManagerFilterer, error) {
	contract, err := bindIOwnerManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IOwnerManagerFilterer{contract: contract}, nil
}

// bindIOwnerManager binds a generic wrapper to an already deployed contract.
func bindIOwnerManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IOwnerManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOwnerManager *IOwnerManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOwnerManager.Contract.IOwnerManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOwnerManager *IOwnerManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOwnerManager.Contract.IOwnerManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOwnerManager *IOwnerManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOwnerManager.Contract.IOwnerManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOwnerManager *IOwnerManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOwnerManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOwnerManager *IOwnerManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOwnerManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOwnerManager *IOwnerManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOwnerManager.Contract.contract.Transact(opts, method, params...)
}

// GetOwners is a free data retrieval call binding the contract method 0xa0e67e2b.
//
// Solidity: function getOwners() view returns(address[])
func (_IOwnerManager *IOwnerManagerCaller) GetOwners(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _IOwnerManager.contract.Call(opts, &out, "getOwners")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetOwners is a free data retrieval call binding the contract method 0xa0e67e2b.
//
// Solidity: function getOwners() view returns(address[])
func (_IOwnerManager *IOwnerManagerSession) GetOwners() ([]common.Address, error) {
	return _IOwnerManager.Contract.GetOwners(&_IOwnerManager.CallOpts)
}

// GetOwners is a free data retrieval call binding the contract method 0xa0e67e2b.
//
// Solidity: function getOwners() view returns(address[])
func (_IOwnerManager *IOwnerManagerCallerSession) GetOwners() ([]common.Address, error) {
	return _IOwnerManager.Contract.GetOwners(&_IOwnerManager.CallOpts)
}

// GetThreshold is a free data retrieval call binding the contract method 0xe75235b8.
//
// Solidity: function getThreshold() view returns(uint256)
func (_IOwnerManager *IOwnerManagerCaller) GetThreshold(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IOwnerManager.contract.Call(opts, &out, "getThreshold")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetThreshold is a free data retrieval call binding the contract method 0xe75235b8.
//
// Solidity: function getThreshold() view returns(uint256)
func (_IOwnerManager *IOwnerManagerSession) GetThreshold() (*big.Int, error) {
	return _IOwnerManager.Contract.GetThreshold(&_IOwnerManager.CallOpts)
}

// GetThreshold is a free data retrieval call binding the contract method 0xe75235b8.
//
// Solidity: function getThreshold() view returns(uint256)
func (_IOwnerManager *IOwnerManagerCallerSession) GetThreshold() (*big.Int, error) {
	return _IOwnerManager.Contract.GetThreshold(&_IOwnerManager.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address owner) view returns(bool)
func (_IOwnerManager *IOwnerManagerCaller) IsOwner(opts *bind.CallOpts, owner common.Address) (bool, error) {
	var out []interface{}
	err := _IOwnerManager.contract.Call(opts, &out, "isOwner", owner)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address owner) view returns(bool)
func (_IOwnerManager *IOwnerManagerSession) IsOwner(owner common.Address) (bool, error) {
	return _IOwnerManager.Contract.IsOwner(&_IOwnerManager.CallOpts, owner)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address owner) view returns(bool)
func (_IOwnerManager *IOwnerManagerCallerSession) IsOwner(owner common.Address) (bool, error) {
	return _IOwnerManager.Contract.IsOwner(&_IOwnerManager.CallOpts, owner)
}

// AddOwnerWithThreshold is a paid mutator transaction binding the contract method 0x0d582f13.
//
// Solidity: function addOwnerWithThreshold(address owner, uint256 _threshold) returns()
func (_IOwnerManager *IOwnerManagerTransactor) AddOwnerWithThreshold(opts *bind.TransactOpts, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _IOwnerManager.contract.Transact(opts, "addOwnerWithThreshold", owner, _threshold)
}

// AddOwnerWithThreshold is a paid mutator transaction binding the contract method 0x0d582f13.
//
// Solidity: function addOwnerWithThreshold(address owner, uint256 _threshold) returns()
func (_IOwnerManager *IOwnerManagerSession) AddOwnerWithThreshold(owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _IOwnerManager.Contract.AddOwnerWithThreshold(&_IOwnerManager.TransactOpts, owner, _threshold)
}

// AddOwnerWithThreshold is a paid mutator transaction binding the contract method 0x0d582f13.
//
// Solidity: function addOwnerWithThreshold(address owner, uint256 _threshold) returns()
func (_IOwnerManager *IOwnerManagerTransactorSession) AddOwnerWithThreshold(owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _IOwnerManager.Contract.AddOwnerWithThreshold(&_IOwnerManager.TransactOpts, owner, _threshold)
}

// ChangeThreshold is a paid mutator transaction binding the contract method 0x694e80c3.
//
// Solidity: function changeThreshold(uint256 _threshold) returns()
func (_IOwnerManager *IOwnerManagerTransactor) ChangeThreshold(opts *bind.TransactOpts, _threshold *big.Int) (*types.Transaction, error) {
	return _IOwnerManager.contract.Transact(opts, "changeThreshold", _threshold)
}

// ChangeThreshold is a paid mutator transaction binding the contract method 0x694e80c3.
//
// Solidity: function changeThreshold(uint256 _threshold) returns()
func (_IOwnerManager *IOwnerManagerSession) ChangeThreshold(_threshold *big.Int) (*types.Transaction, error) {
	return _IOwnerManager.Contract.ChangeThreshold(&_IOwnerManager.TransactOpts, _threshold)
}

// ChangeThreshold is a paid mutator transaction binding the contract method 0x694e80c3.
//
// Solidity: function changeThreshold(uint256 _threshold) returns()
func (_IOwnerManager *IOwnerManagerTransactorSession) ChangeThreshold(_threshold *big.Int) (*types.Transaction, error) {
	return _IOwnerManager.Contract.ChangeThreshold(&_IOwnerManager.TransactOpts, _threshold)
}

// RemoveOwner is a paid mutator transaction binding the contract method 0xf8dc5dd9.
//
// Solidity: function removeOwner(address prevOwner, address owner, uint256 _threshold) returns()
func (_IOwnerManager *IOwnerManagerTransactor) RemoveOwner(opts *bind.TransactOpts, prevOwner common.Address, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _IOwnerManager.contract.Transact(opts, "removeOwner", prevOwner, owner, _threshold)
}

// RemoveOwner is a paid mutator transaction binding the contract method 0xf8dc5dd9.
//
// Solidity: function removeOwner(address prevOwner, address owner, uint256 _threshold) returns()
func (_IOwnerManager *IOwnerManagerSession) RemoveOwner(prevOwner common.Address, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _IOwnerManager.Contract.RemoveOwner(&_IOwnerManager.TransactOpts, prevOwner, owner, _threshold)
}

// RemoveOwner is a paid mutator transaction binding the contract method 0xf8dc5dd9.
//
// Solidity: function removeOwner(address prevOwner, address owner, uint256 _threshold) returns()
func (_IOwnerManager *IOwnerManagerTransactorSession) RemoveOwner(prevOwner common.Address, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _IOwnerManager.Contract.RemoveOwner(&_IOwnerManager.TransactOpts, prevOwner, owner, _threshold)
}

// SwapOwner is a paid mutator transaction binding the contract method 0xe318b52b.
//
// Solidity: function swapOwner(address prevOwner, address oldOwner, address newOwner) returns()
func (_IOwnerManager *IOwnerManagerTransactor) SwapOwner(opts *bind.TransactOpts, prevOwner common.Address, oldOwner common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _IOwnerManager.contract.Transact(opts, "swapOwner", prevOwner, oldOwner, newOwner)
}

// SwapOwner is a paid mutator transaction binding the contract method 0xe318b52b.
//
// Solidity: function swapOwner(address prevOwner, address oldOwner, address newOwner) returns()
func (_IOwnerManager *IOwnerManagerSession) SwapOwner(prevOwner common.Address, oldOwner common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _IOwnerManager.Contract.SwapOwner(&_IOwnerManager.TransactOpts, prevOwner, oldOwner, newOwner)
}

// SwapOwner is a paid mutator transaction binding the contract method 0xe318b52b.
//
// Solidity: function swapOwner(address prevOwner, address oldOwner, address newOwner) returns()
func (_IOwnerManager *IOwnerManagerTransactorSession) SwapOwner(prevOwner common.Address, oldOwner common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _IOwnerManager.Contract.SwapOwner(&_IOwnerManager.TransactOpts, prevOwner, oldOwner, newOwner)
}

// IOwnerManagerAddedOwnerIterator is returned from FilterAddedOwner and is used to iterate over the raw logs and unpacked data for AddedOwner events raised by the IOwnerManager contract.
type IOwnerManagerAddedOwnerIterator struct {
	Event *IOwnerManagerAddedOwner // Event containing the contract specifics and raw log

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
func (it *IOwnerManagerAddedOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IOwnerManagerAddedOwner)
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
		it.Event = new(IOwnerManagerAddedOwner)
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
func (it *IOwnerManagerAddedOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IOwnerManagerAddedOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IOwnerManagerAddedOwner represents a AddedOwner event raised by the IOwnerManager contract.
type IOwnerManagerAddedOwner struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterAddedOwner is a free log retrieval operation binding the contract event 0x9465fa0c962cc76958e6373a993326400c1c94f8be2fe3a952adfa7f60b2ea26.
//
// Solidity: event AddedOwner(address indexed owner)
func (_IOwnerManager *IOwnerManagerFilterer) FilterAddedOwner(opts *bind.FilterOpts, owner []common.Address) (*IOwnerManagerAddedOwnerIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _IOwnerManager.contract.FilterLogs(opts, "AddedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return &IOwnerManagerAddedOwnerIterator{contract: _IOwnerManager.contract, event: "AddedOwner", logs: logs, sub: sub}, nil
}

// WatchAddedOwner is a free log subscription operation binding the contract event 0x9465fa0c962cc76958e6373a993326400c1c94f8be2fe3a952adfa7f60b2ea26.
//
// Solidity: event AddedOwner(address indexed owner)
func (_IOwnerManager *IOwnerManagerFilterer) WatchAddedOwner(opts *bind.WatchOpts, sink chan<- *IOwnerManagerAddedOwner, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _IOwnerManager.contract.WatchLogs(opts, "AddedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IOwnerManagerAddedOwner)
				if err := _IOwnerManager.contract.UnpackLog(event, "AddedOwner", log); err != nil {
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
func (_IOwnerManager *IOwnerManagerFilterer) ParseAddedOwner(log types.Log) (*IOwnerManagerAddedOwner, error) {
	event := new(IOwnerManagerAddedOwner)
	if err := _IOwnerManager.contract.UnpackLog(event, "AddedOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOwnerManagerChangedThresholdIterator is returned from FilterChangedThreshold and is used to iterate over the raw logs and unpacked data for ChangedThreshold events raised by the IOwnerManager contract.
type IOwnerManagerChangedThresholdIterator struct {
	Event *IOwnerManagerChangedThreshold // Event containing the contract specifics and raw log

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
func (it *IOwnerManagerChangedThresholdIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IOwnerManagerChangedThreshold)
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
		it.Event = new(IOwnerManagerChangedThreshold)
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
func (it *IOwnerManagerChangedThresholdIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IOwnerManagerChangedThresholdIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IOwnerManagerChangedThreshold represents a ChangedThreshold event raised by the IOwnerManager contract.
type IOwnerManagerChangedThreshold struct {
	Threshold *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterChangedThreshold is a free log retrieval operation binding the contract event 0x610f7ff2b304ae8903c3de74c60c6ab1f7d6226b3f52c5161905bb5ad4039c93.
//
// Solidity: event ChangedThreshold(uint256 threshold)
func (_IOwnerManager *IOwnerManagerFilterer) FilterChangedThreshold(opts *bind.FilterOpts) (*IOwnerManagerChangedThresholdIterator, error) {

	logs, sub, err := _IOwnerManager.contract.FilterLogs(opts, "ChangedThreshold")
	if err != nil {
		return nil, err
	}
	return &IOwnerManagerChangedThresholdIterator{contract: _IOwnerManager.contract, event: "ChangedThreshold", logs: logs, sub: sub}, nil
}

// WatchChangedThreshold is a free log subscription operation binding the contract event 0x610f7ff2b304ae8903c3de74c60c6ab1f7d6226b3f52c5161905bb5ad4039c93.
//
// Solidity: event ChangedThreshold(uint256 threshold)
func (_IOwnerManager *IOwnerManagerFilterer) WatchChangedThreshold(opts *bind.WatchOpts, sink chan<- *IOwnerManagerChangedThreshold) (event.Subscription, error) {

	logs, sub, err := _IOwnerManager.contract.WatchLogs(opts, "ChangedThreshold")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IOwnerManagerChangedThreshold)
				if err := _IOwnerManager.contract.UnpackLog(event, "ChangedThreshold", log); err != nil {
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
func (_IOwnerManager *IOwnerManagerFilterer) ParseChangedThreshold(log types.Log) (*IOwnerManagerChangedThreshold, error) {
	event := new(IOwnerManagerChangedThreshold)
	if err := _IOwnerManager.contract.UnpackLog(event, "ChangedThreshold", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOwnerManagerRemovedOwnerIterator is returned from FilterRemovedOwner and is used to iterate over the raw logs and unpacked data for RemovedOwner events raised by the IOwnerManager contract.
type IOwnerManagerRemovedOwnerIterator struct {
	Event *IOwnerManagerRemovedOwner // Event containing the contract specifics and raw log

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
func (it *IOwnerManagerRemovedOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IOwnerManagerRemovedOwner)
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
		it.Event = new(IOwnerManagerRemovedOwner)
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
func (it *IOwnerManagerRemovedOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IOwnerManagerRemovedOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IOwnerManagerRemovedOwner represents a RemovedOwner event raised by the IOwnerManager contract.
type IOwnerManagerRemovedOwner struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterRemovedOwner is a free log retrieval operation binding the contract event 0xf8d49fc529812e9a7c5c50e69c20f0dccc0db8fa95c98bc58cc9a4f1c1299eaf.
//
// Solidity: event RemovedOwner(address indexed owner)
func (_IOwnerManager *IOwnerManagerFilterer) FilterRemovedOwner(opts *bind.FilterOpts, owner []common.Address) (*IOwnerManagerRemovedOwnerIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _IOwnerManager.contract.FilterLogs(opts, "RemovedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return &IOwnerManagerRemovedOwnerIterator{contract: _IOwnerManager.contract, event: "RemovedOwner", logs: logs, sub: sub}, nil
}

// WatchRemovedOwner is a free log subscription operation binding the contract event 0xf8d49fc529812e9a7c5c50e69c20f0dccc0db8fa95c98bc58cc9a4f1c1299eaf.
//
// Solidity: event RemovedOwner(address indexed owner)
func (_IOwnerManager *IOwnerManagerFilterer) WatchRemovedOwner(opts *bind.WatchOpts, sink chan<- *IOwnerManagerRemovedOwner, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _IOwnerManager.contract.WatchLogs(opts, "RemovedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IOwnerManagerRemovedOwner)
				if err := _IOwnerManager.contract.UnpackLog(event, "RemovedOwner", log); err != nil {
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
func (_IOwnerManager *IOwnerManagerFilterer) ParseRemovedOwner(log types.Log) (*IOwnerManagerRemovedOwner, error) {
	event := new(IOwnerManagerRemovedOwner)
	if err := _IOwnerManager.contract.UnpackLog(event, "RemovedOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeMetaData contains all meta data concerning the ISafe contract.
var ISafeMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"AddedOwner\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"approvedHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"ApproveHash\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"handler\",\"type\":\"address\"}],\"name\":\"ChangedFallbackHandler\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"guard\",\"type\":\"address\"}],\"name\":\"ChangedGuard\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"moduleGuard\",\"type\":\"address\"}],\"name\":\"ChangedModuleGuard\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"threshold\",\"type\":\"uint256\"}],\"name\":\"ChangedThreshold\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"DisabledModule\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"EnabledModule\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"txHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"payment\",\"type\":\"uint256\"}],\"name\":\"ExecutionFailure\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"ExecutionFromModuleFailure\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"ExecutionFromModuleSuccess\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"txHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"payment\",\"type\":\"uint256\"}],\"name\":\"ExecutionSuccess\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"RemovedOwner\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"initiator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address[]\",\"name\":\"owners\",\"type\":\"address[]\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"threshold\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"initializer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"fallbackHandler\",\"type\":\"address\"}],\"name\":\"SafeSetup\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"SignMsg\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"VERSION\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"}],\"name\":\"addOwnerWithThreshold\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"hashToApprove\",\"type\":\"bytes32\"}],\"name\":\"approveHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"approvedHashes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"}],\"name\":\"changeThreshold\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"executor\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"requiredSignatures\",\"type\":\"uint256\"}],\"name\":\"checkNSignatures\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"checkSignatures\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"prevModule\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"disableModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"domainSeparator\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"enableModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"safeTxGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasPrice\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"gasToken\",\"type\":\"address\"},{\"internalType\":\"addresspayable\",\"name\":\"refundReceiver\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"execTransaction\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"}],\"name\":\"execTransactionFromModule\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"}],\"name\":\"execTransactionFromModuleReturnData\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"start\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"pageSize\",\"type\":\"uint256\"}],\"name\":\"getModulesPaginated\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"array\",\"type\":\"address[]\"},{\"internalType\":\"address\",\"name\":\"next\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getOwners\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getThreshold\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"safeTxGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasPrice\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"gasToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"refundReceiver\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"}],\"name\":\"getTransactionHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"isModuleEnabled\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"isOwner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"prevOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"}],\"name\":\"removeOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"handler\",\"type\":\"address\"}],\"name\":\"setFallbackHandler\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"guard\",\"type\":\"address\"}],\"name\":\"setGuard\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"moduleGuard\",\"type\":\"address\"}],\"name\":\"setModuleGuard\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"_owners\",\"type\":\"address[]\"},{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"address\",\"name\":\"fallbackHandler\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"paymentToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"payment\",\"type\":\"uint256\"},{\"internalType\":\"addresspayable\",\"name\":\"paymentReceiver\",\"type\":\"address\"}],\"name\":\"setup\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"signedMessages\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"prevOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"swapOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ISafeABI is the input ABI used to generate the binding from.
// Deprecated: Use ISafeMetaData.ABI instead.
var ISafeABI = ISafeMetaData.ABI

// ISafe is an auto generated Go binding around an Ethereum contract.
type ISafe struct {
	ISafeCaller     // Read-only binding to the contract
	ISafeTransactor // Write-only binding to the contract
	ISafeFilterer   // Log filterer for contract events
}

// ISafeCaller is an auto generated read-only Go binding around an Ethereum contract.
type ISafeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISafeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ISafeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISafeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ISafeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISafeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ISafeSession struct {
	Contract     *ISafe            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ISafeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ISafeCallerSession struct {
	Contract *ISafeCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// ISafeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ISafeTransactorSession struct {
	Contract     *ISafeTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ISafeRaw is an auto generated low-level Go binding around an Ethereum contract.
type ISafeRaw struct {
	Contract *ISafe // Generic contract binding to access the raw methods on
}

// ISafeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ISafeCallerRaw struct {
	Contract *ISafeCaller // Generic read-only contract binding to access the raw methods on
}

// ISafeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ISafeTransactorRaw struct {
	Contract *ISafeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewISafe creates a new instance of ISafe, bound to a specific deployed contract.
func NewISafe(address common.Address, backend bind.ContractBackend) (*ISafe, error) {
	contract, err := bindISafe(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ISafe{ISafeCaller: ISafeCaller{contract: contract}, ISafeTransactor: ISafeTransactor{contract: contract}, ISafeFilterer: ISafeFilterer{contract: contract}}, nil
}

// NewISafeCaller creates a new read-only instance of ISafe, bound to a specific deployed contract.
func NewISafeCaller(address common.Address, caller bind.ContractCaller) (*ISafeCaller, error) {
	contract, err := bindISafe(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ISafeCaller{contract: contract}, nil
}

// NewISafeTransactor creates a new write-only instance of ISafe, bound to a specific deployed contract.
func NewISafeTransactor(address common.Address, transactor bind.ContractTransactor) (*ISafeTransactor, error) {
	contract, err := bindISafe(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ISafeTransactor{contract: contract}, nil
}

// NewISafeFilterer creates a new log filterer instance of ISafe, bound to a specific deployed contract.
func NewISafeFilterer(address common.Address, filterer bind.ContractFilterer) (*ISafeFilterer, error) {
	contract, err := bindISafe(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ISafeFilterer{contract: contract}, nil
}

// bindISafe binds a generic wrapper to an already deployed contract.
func bindISafe(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ISafeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ISafe *ISafeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ISafe.Contract.ISafeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ISafe *ISafeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISafe.Contract.ISafeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ISafe *ISafeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ISafe.Contract.ISafeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ISafe *ISafeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ISafe.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ISafe *ISafeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISafe.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ISafe *ISafeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ISafe.Contract.contract.Transact(opts, method, params...)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(string)
func (_ISafe *ISafeCaller) VERSION(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "VERSION")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(string)
func (_ISafe *ISafeSession) VERSION() (string, error) {
	return _ISafe.Contract.VERSION(&_ISafe.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(string)
func (_ISafe *ISafeCallerSession) VERSION() (string, error) {
	return _ISafe.Contract.VERSION(&_ISafe.CallOpts)
}

// ApprovedHashes is a free data retrieval call binding the contract method 0x7d832974.
//
// Solidity: function approvedHashes(address owner, bytes32 messageHash) view returns(uint256)
func (_ISafe *ISafeCaller) ApprovedHashes(opts *bind.CallOpts, owner common.Address, messageHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "approvedHashes", owner, messageHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ApprovedHashes is a free data retrieval call binding the contract method 0x7d832974.
//
// Solidity: function approvedHashes(address owner, bytes32 messageHash) view returns(uint256)
func (_ISafe *ISafeSession) ApprovedHashes(owner common.Address, messageHash [32]byte) (*big.Int, error) {
	return _ISafe.Contract.ApprovedHashes(&_ISafe.CallOpts, owner, messageHash)
}

// ApprovedHashes is a free data retrieval call binding the contract method 0x7d832974.
//
// Solidity: function approvedHashes(address owner, bytes32 messageHash) view returns(uint256)
func (_ISafe *ISafeCallerSession) ApprovedHashes(owner common.Address, messageHash [32]byte) (*big.Int, error) {
	return _ISafe.Contract.ApprovedHashes(&_ISafe.CallOpts, owner, messageHash)
}

// CheckNSignatures is a free data retrieval call binding the contract method 0x1fcac7f3.
//
// Solidity: function checkNSignatures(address executor, bytes32 dataHash, bytes signatures, uint256 requiredSignatures) view returns()
func (_ISafe *ISafeCaller) CheckNSignatures(opts *bind.CallOpts, executor common.Address, dataHash [32]byte, signatures []byte, requiredSignatures *big.Int) error {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "checkNSignatures", executor, dataHash, signatures, requiredSignatures)

	if err != nil {
		return err
	}

	return err

}

// CheckNSignatures is a free data retrieval call binding the contract method 0x1fcac7f3.
//
// Solidity: function checkNSignatures(address executor, bytes32 dataHash, bytes signatures, uint256 requiredSignatures) view returns()
func (_ISafe *ISafeSession) CheckNSignatures(executor common.Address, dataHash [32]byte, signatures []byte, requiredSignatures *big.Int) error {
	return _ISafe.Contract.CheckNSignatures(&_ISafe.CallOpts, executor, dataHash, signatures, requiredSignatures)
}

// CheckNSignatures is a free data retrieval call binding the contract method 0x1fcac7f3.
//
// Solidity: function checkNSignatures(address executor, bytes32 dataHash, bytes signatures, uint256 requiredSignatures) view returns()
func (_ISafe *ISafeCallerSession) CheckNSignatures(executor common.Address, dataHash [32]byte, signatures []byte, requiredSignatures *big.Int) error {
	return _ISafe.Contract.CheckNSignatures(&_ISafe.CallOpts, executor, dataHash, signatures, requiredSignatures)
}

// CheckSignatures is a free data retrieval call binding the contract method 0xed516d51.
//
// Solidity: function checkSignatures(bytes32 dataHash, bytes signatures) view returns()
func (_ISafe *ISafeCaller) CheckSignatures(opts *bind.CallOpts, dataHash [32]byte, signatures []byte) error {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "checkSignatures", dataHash, signatures)

	if err != nil {
		return err
	}

	return err

}

// CheckSignatures is a free data retrieval call binding the contract method 0xed516d51.
//
// Solidity: function checkSignatures(bytes32 dataHash, bytes signatures) view returns()
func (_ISafe *ISafeSession) CheckSignatures(dataHash [32]byte, signatures []byte) error {
	return _ISafe.Contract.CheckSignatures(&_ISafe.CallOpts, dataHash, signatures)
}

// CheckSignatures is a free data retrieval call binding the contract method 0xed516d51.
//
// Solidity: function checkSignatures(bytes32 dataHash, bytes signatures) view returns()
func (_ISafe *ISafeCallerSession) CheckSignatures(dataHash [32]byte, signatures []byte) error {
	return _ISafe.Contract.CheckSignatures(&_ISafe.CallOpts, dataHash, signatures)
}

// DomainSeparator is a free data retrieval call binding the contract method 0xf698da25.
//
// Solidity: function domainSeparator() view returns(bytes32)
func (_ISafe *ISafeCaller) DomainSeparator(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "domainSeparator")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DomainSeparator is a free data retrieval call binding the contract method 0xf698da25.
//
// Solidity: function domainSeparator() view returns(bytes32)
func (_ISafe *ISafeSession) DomainSeparator() ([32]byte, error) {
	return _ISafe.Contract.DomainSeparator(&_ISafe.CallOpts)
}

// DomainSeparator is a free data retrieval call binding the contract method 0xf698da25.
//
// Solidity: function domainSeparator() view returns(bytes32)
func (_ISafe *ISafeCallerSession) DomainSeparator() ([32]byte, error) {
	return _ISafe.Contract.DomainSeparator(&_ISafe.CallOpts)
}

// GetModulesPaginated is a free data retrieval call binding the contract method 0xcc2f8452.
//
// Solidity: function getModulesPaginated(address start, uint256 pageSize) view returns(address[] array, address next)
func (_ISafe *ISafeCaller) GetModulesPaginated(opts *bind.CallOpts, start common.Address, pageSize *big.Int) (struct {
	Array []common.Address
	Next  common.Address
}, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "getModulesPaginated", start, pageSize)

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
func (_ISafe *ISafeSession) GetModulesPaginated(start common.Address, pageSize *big.Int) (struct {
	Array []common.Address
	Next  common.Address
}, error) {
	return _ISafe.Contract.GetModulesPaginated(&_ISafe.CallOpts, start, pageSize)
}

// GetModulesPaginated is a free data retrieval call binding the contract method 0xcc2f8452.
//
// Solidity: function getModulesPaginated(address start, uint256 pageSize) view returns(address[] array, address next)
func (_ISafe *ISafeCallerSession) GetModulesPaginated(start common.Address, pageSize *big.Int) (struct {
	Array []common.Address
	Next  common.Address
}, error) {
	return _ISafe.Contract.GetModulesPaginated(&_ISafe.CallOpts, start, pageSize)
}

// GetOwners is a free data retrieval call binding the contract method 0xa0e67e2b.
//
// Solidity: function getOwners() view returns(address[])
func (_ISafe *ISafeCaller) GetOwners(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "getOwners")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetOwners is a free data retrieval call binding the contract method 0xa0e67e2b.
//
// Solidity: function getOwners() view returns(address[])
func (_ISafe *ISafeSession) GetOwners() ([]common.Address, error) {
	return _ISafe.Contract.GetOwners(&_ISafe.CallOpts)
}

// GetOwners is a free data retrieval call binding the contract method 0xa0e67e2b.
//
// Solidity: function getOwners() view returns(address[])
func (_ISafe *ISafeCallerSession) GetOwners() ([]common.Address, error) {
	return _ISafe.Contract.GetOwners(&_ISafe.CallOpts)
}

// GetThreshold is a free data retrieval call binding the contract method 0xe75235b8.
//
// Solidity: function getThreshold() view returns(uint256)
func (_ISafe *ISafeCaller) GetThreshold(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "getThreshold")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetThreshold is a free data retrieval call binding the contract method 0xe75235b8.
//
// Solidity: function getThreshold() view returns(uint256)
func (_ISafe *ISafeSession) GetThreshold() (*big.Int, error) {
	return _ISafe.Contract.GetThreshold(&_ISafe.CallOpts)
}

// GetThreshold is a free data retrieval call binding the contract method 0xe75235b8.
//
// Solidity: function getThreshold() view returns(uint256)
func (_ISafe *ISafeCallerSession) GetThreshold() (*big.Int, error) {
	return _ISafe.Contract.GetThreshold(&_ISafe.CallOpts)
}

// GetTransactionHash is a free data retrieval call binding the contract method 0xd8d11f78.
//
// Solidity: function getTransactionHash(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, uint256 _nonce) view returns(bytes32)
func (_ISafe *ISafeCaller) GetTransactionHash(opts *bind.CallOpts, to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, _nonce *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "getTransactionHash", to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, _nonce)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetTransactionHash is a free data retrieval call binding the contract method 0xd8d11f78.
//
// Solidity: function getTransactionHash(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, uint256 _nonce) view returns(bytes32)
func (_ISafe *ISafeSession) GetTransactionHash(to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, _nonce *big.Int) ([32]byte, error) {
	return _ISafe.Contract.GetTransactionHash(&_ISafe.CallOpts, to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, _nonce)
}

// GetTransactionHash is a free data retrieval call binding the contract method 0xd8d11f78.
//
// Solidity: function getTransactionHash(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, uint256 _nonce) view returns(bytes32)
func (_ISafe *ISafeCallerSession) GetTransactionHash(to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, _nonce *big.Int) ([32]byte, error) {
	return _ISafe.Contract.GetTransactionHash(&_ISafe.CallOpts, to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, _nonce)
}

// IsModuleEnabled is a free data retrieval call binding the contract method 0x2d9ad53d.
//
// Solidity: function isModuleEnabled(address module) view returns(bool)
func (_ISafe *ISafeCaller) IsModuleEnabled(opts *bind.CallOpts, module common.Address) (bool, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "isModuleEnabled", module)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsModuleEnabled is a free data retrieval call binding the contract method 0x2d9ad53d.
//
// Solidity: function isModuleEnabled(address module) view returns(bool)
func (_ISafe *ISafeSession) IsModuleEnabled(module common.Address) (bool, error) {
	return _ISafe.Contract.IsModuleEnabled(&_ISafe.CallOpts, module)
}

// IsModuleEnabled is a free data retrieval call binding the contract method 0x2d9ad53d.
//
// Solidity: function isModuleEnabled(address module) view returns(bool)
func (_ISafe *ISafeCallerSession) IsModuleEnabled(module common.Address) (bool, error) {
	return _ISafe.Contract.IsModuleEnabled(&_ISafe.CallOpts, module)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address owner) view returns(bool)
func (_ISafe *ISafeCaller) IsOwner(opts *bind.CallOpts, owner common.Address) (bool, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "isOwner", owner)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address owner) view returns(bool)
func (_ISafe *ISafeSession) IsOwner(owner common.Address) (bool, error) {
	return _ISafe.Contract.IsOwner(&_ISafe.CallOpts, owner)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address owner) view returns(bool)
func (_ISafe *ISafeCallerSession) IsOwner(owner common.Address) (bool, error) {
	return _ISafe.Contract.IsOwner(&_ISafe.CallOpts, owner)
}

// Nonce is a free data retrieval call binding the contract method 0xaffed0e0.
//
// Solidity: function nonce() view returns(uint256)
func (_ISafe *ISafeCaller) Nonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "nonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Nonce is a free data retrieval call binding the contract method 0xaffed0e0.
//
// Solidity: function nonce() view returns(uint256)
func (_ISafe *ISafeSession) Nonce() (*big.Int, error) {
	return _ISafe.Contract.Nonce(&_ISafe.CallOpts)
}

// Nonce is a free data retrieval call binding the contract method 0xaffed0e0.
//
// Solidity: function nonce() view returns(uint256)
func (_ISafe *ISafeCallerSession) Nonce() (*big.Int, error) {
	return _ISafe.Contract.Nonce(&_ISafe.CallOpts)
}

// SignedMessages is a free data retrieval call binding the contract method 0x5ae6bd37.
//
// Solidity: function signedMessages(bytes32 messageHash) view returns(uint256)
func (_ISafe *ISafeCaller) SignedMessages(opts *bind.CallOpts, messageHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _ISafe.contract.Call(opts, &out, "signedMessages", messageHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SignedMessages is a free data retrieval call binding the contract method 0x5ae6bd37.
//
// Solidity: function signedMessages(bytes32 messageHash) view returns(uint256)
func (_ISafe *ISafeSession) SignedMessages(messageHash [32]byte) (*big.Int, error) {
	return _ISafe.Contract.SignedMessages(&_ISafe.CallOpts, messageHash)
}

// SignedMessages is a free data retrieval call binding the contract method 0x5ae6bd37.
//
// Solidity: function signedMessages(bytes32 messageHash) view returns(uint256)
func (_ISafe *ISafeCallerSession) SignedMessages(messageHash [32]byte) (*big.Int, error) {
	return _ISafe.Contract.SignedMessages(&_ISafe.CallOpts, messageHash)
}

// AddOwnerWithThreshold is a paid mutator transaction binding the contract method 0x0d582f13.
//
// Solidity: function addOwnerWithThreshold(address owner, uint256 _threshold) returns()
func (_ISafe *ISafeTransactor) AddOwnerWithThreshold(opts *bind.TransactOpts, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "addOwnerWithThreshold", owner, _threshold)
}

// AddOwnerWithThreshold is a paid mutator transaction binding the contract method 0x0d582f13.
//
// Solidity: function addOwnerWithThreshold(address owner, uint256 _threshold) returns()
func (_ISafe *ISafeSession) AddOwnerWithThreshold(owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _ISafe.Contract.AddOwnerWithThreshold(&_ISafe.TransactOpts, owner, _threshold)
}

// AddOwnerWithThreshold is a paid mutator transaction binding the contract method 0x0d582f13.
//
// Solidity: function addOwnerWithThreshold(address owner, uint256 _threshold) returns()
func (_ISafe *ISafeTransactorSession) AddOwnerWithThreshold(owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _ISafe.Contract.AddOwnerWithThreshold(&_ISafe.TransactOpts, owner, _threshold)
}

// ApproveHash is a paid mutator transaction binding the contract method 0xd4d9bdcd.
//
// Solidity: function approveHash(bytes32 hashToApprove) returns()
func (_ISafe *ISafeTransactor) ApproveHash(opts *bind.TransactOpts, hashToApprove [32]byte) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "approveHash", hashToApprove)
}

// ApproveHash is a paid mutator transaction binding the contract method 0xd4d9bdcd.
//
// Solidity: function approveHash(bytes32 hashToApprove) returns()
func (_ISafe *ISafeSession) ApproveHash(hashToApprove [32]byte) (*types.Transaction, error) {
	return _ISafe.Contract.ApproveHash(&_ISafe.TransactOpts, hashToApprove)
}

// ApproveHash is a paid mutator transaction binding the contract method 0xd4d9bdcd.
//
// Solidity: function approveHash(bytes32 hashToApprove) returns()
func (_ISafe *ISafeTransactorSession) ApproveHash(hashToApprove [32]byte) (*types.Transaction, error) {
	return _ISafe.Contract.ApproveHash(&_ISafe.TransactOpts, hashToApprove)
}

// ChangeThreshold is a paid mutator transaction binding the contract method 0x694e80c3.
//
// Solidity: function changeThreshold(uint256 _threshold) returns()
func (_ISafe *ISafeTransactor) ChangeThreshold(opts *bind.TransactOpts, _threshold *big.Int) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "changeThreshold", _threshold)
}

// ChangeThreshold is a paid mutator transaction binding the contract method 0x694e80c3.
//
// Solidity: function changeThreshold(uint256 _threshold) returns()
func (_ISafe *ISafeSession) ChangeThreshold(_threshold *big.Int) (*types.Transaction, error) {
	return _ISafe.Contract.ChangeThreshold(&_ISafe.TransactOpts, _threshold)
}

// ChangeThreshold is a paid mutator transaction binding the contract method 0x694e80c3.
//
// Solidity: function changeThreshold(uint256 _threshold) returns()
func (_ISafe *ISafeTransactorSession) ChangeThreshold(_threshold *big.Int) (*types.Transaction, error) {
	return _ISafe.Contract.ChangeThreshold(&_ISafe.TransactOpts, _threshold)
}

// DisableModule is a paid mutator transaction binding the contract method 0xe009cfde.
//
// Solidity: function disableModule(address prevModule, address module) returns()
func (_ISafe *ISafeTransactor) DisableModule(opts *bind.TransactOpts, prevModule common.Address, module common.Address) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "disableModule", prevModule, module)
}

// DisableModule is a paid mutator transaction binding the contract method 0xe009cfde.
//
// Solidity: function disableModule(address prevModule, address module) returns()
func (_ISafe *ISafeSession) DisableModule(prevModule common.Address, module common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.DisableModule(&_ISafe.TransactOpts, prevModule, module)
}

// DisableModule is a paid mutator transaction binding the contract method 0xe009cfde.
//
// Solidity: function disableModule(address prevModule, address module) returns()
func (_ISafe *ISafeTransactorSession) DisableModule(prevModule common.Address, module common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.DisableModule(&_ISafe.TransactOpts, prevModule, module)
}

// EnableModule is a paid mutator transaction binding the contract method 0x610b5925.
//
// Solidity: function enableModule(address module) returns()
func (_ISafe *ISafeTransactor) EnableModule(opts *bind.TransactOpts, module common.Address) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "enableModule", module)
}

// EnableModule is a paid mutator transaction binding the contract method 0x610b5925.
//
// Solidity: function enableModule(address module) returns()
func (_ISafe *ISafeSession) EnableModule(module common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.EnableModule(&_ISafe.TransactOpts, module)
}

// EnableModule is a paid mutator transaction binding the contract method 0x610b5925.
//
// Solidity: function enableModule(address module) returns()
func (_ISafe *ISafeTransactorSession) EnableModule(module common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.EnableModule(&_ISafe.TransactOpts, module)
}

// ExecTransaction is a paid mutator transaction binding the contract method 0x6a761202.
//
// Solidity: function execTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures) payable returns(bool success)
func (_ISafe *ISafeTransactor) ExecTransaction(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "execTransaction", to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures)
}

// ExecTransaction is a paid mutator transaction binding the contract method 0x6a761202.
//
// Solidity: function execTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures) payable returns(bool success)
func (_ISafe *ISafeSession) ExecTransaction(to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte) (*types.Transaction, error) {
	return _ISafe.Contract.ExecTransaction(&_ISafe.TransactOpts, to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures)
}

// ExecTransaction is a paid mutator transaction binding the contract method 0x6a761202.
//
// Solidity: function execTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures) payable returns(bool success)
func (_ISafe *ISafeTransactorSession) ExecTransaction(to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte) (*types.Transaction, error) {
	return _ISafe.Contract.ExecTransaction(&_ISafe.TransactOpts, to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures)
}

// ExecTransactionFromModule is a paid mutator transaction binding the contract method 0x468721a7.
//
// Solidity: function execTransactionFromModule(address to, uint256 value, bytes data, uint8 operation) returns(bool success)
func (_ISafe *ISafeTransactor) ExecTransactionFromModule(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "execTransactionFromModule", to, value, data, operation)
}

// ExecTransactionFromModule is a paid mutator transaction binding the contract method 0x468721a7.
//
// Solidity: function execTransactionFromModule(address to, uint256 value, bytes data, uint8 operation) returns(bool success)
func (_ISafe *ISafeSession) ExecTransactionFromModule(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ISafe.Contract.ExecTransactionFromModule(&_ISafe.TransactOpts, to, value, data, operation)
}

// ExecTransactionFromModule is a paid mutator transaction binding the contract method 0x468721a7.
//
// Solidity: function execTransactionFromModule(address to, uint256 value, bytes data, uint8 operation) returns(bool success)
func (_ISafe *ISafeTransactorSession) ExecTransactionFromModule(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ISafe.Contract.ExecTransactionFromModule(&_ISafe.TransactOpts, to, value, data, operation)
}

// ExecTransactionFromModuleReturnData is a paid mutator transaction binding the contract method 0x5229073f.
//
// Solidity: function execTransactionFromModuleReturnData(address to, uint256 value, bytes data, uint8 operation) returns(bool success, bytes returnData)
func (_ISafe *ISafeTransactor) ExecTransactionFromModuleReturnData(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "execTransactionFromModuleReturnData", to, value, data, operation)
}

// ExecTransactionFromModuleReturnData is a paid mutator transaction binding the contract method 0x5229073f.
//
// Solidity: function execTransactionFromModuleReturnData(address to, uint256 value, bytes data, uint8 operation) returns(bool success, bytes returnData)
func (_ISafe *ISafeSession) ExecTransactionFromModuleReturnData(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ISafe.Contract.ExecTransactionFromModuleReturnData(&_ISafe.TransactOpts, to, value, data, operation)
}

// ExecTransactionFromModuleReturnData is a paid mutator transaction binding the contract method 0x5229073f.
//
// Solidity: function execTransactionFromModuleReturnData(address to, uint256 value, bytes data, uint8 operation) returns(bool success, bytes returnData)
func (_ISafe *ISafeTransactorSession) ExecTransactionFromModuleReturnData(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _ISafe.Contract.ExecTransactionFromModuleReturnData(&_ISafe.TransactOpts, to, value, data, operation)
}

// RemoveOwner is a paid mutator transaction binding the contract method 0xf8dc5dd9.
//
// Solidity: function removeOwner(address prevOwner, address owner, uint256 _threshold) returns()
func (_ISafe *ISafeTransactor) RemoveOwner(opts *bind.TransactOpts, prevOwner common.Address, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "removeOwner", prevOwner, owner, _threshold)
}

// RemoveOwner is a paid mutator transaction binding the contract method 0xf8dc5dd9.
//
// Solidity: function removeOwner(address prevOwner, address owner, uint256 _threshold) returns()
func (_ISafe *ISafeSession) RemoveOwner(prevOwner common.Address, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _ISafe.Contract.RemoveOwner(&_ISafe.TransactOpts, prevOwner, owner, _threshold)
}

// RemoveOwner is a paid mutator transaction binding the contract method 0xf8dc5dd9.
//
// Solidity: function removeOwner(address prevOwner, address owner, uint256 _threshold) returns()
func (_ISafe *ISafeTransactorSession) RemoveOwner(prevOwner common.Address, owner common.Address, _threshold *big.Int) (*types.Transaction, error) {
	return _ISafe.Contract.RemoveOwner(&_ISafe.TransactOpts, prevOwner, owner, _threshold)
}

// SetFallbackHandler is a paid mutator transaction binding the contract method 0xf08a0323.
//
// Solidity: function setFallbackHandler(address handler) returns()
func (_ISafe *ISafeTransactor) SetFallbackHandler(opts *bind.TransactOpts, handler common.Address) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "setFallbackHandler", handler)
}

// SetFallbackHandler is a paid mutator transaction binding the contract method 0xf08a0323.
//
// Solidity: function setFallbackHandler(address handler) returns()
func (_ISafe *ISafeSession) SetFallbackHandler(handler common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.SetFallbackHandler(&_ISafe.TransactOpts, handler)
}

// SetFallbackHandler is a paid mutator transaction binding the contract method 0xf08a0323.
//
// Solidity: function setFallbackHandler(address handler) returns()
func (_ISafe *ISafeTransactorSession) SetFallbackHandler(handler common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.SetFallbackHandler(&_ISafe.TransactOpts, handler)
}

// SetGuard is a paid mutator transaction binding the contract method 0xe19a9dd9.
//
// Solidity: function setGuard(address guard) returns()
func (_ISafe *ISafeTransactor) SetGuard(opts *bind.TransactOpts, guard common.Address) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "setGuard", guard)
}

// SetGuard is a paid mutator transaction binding the contract method 0xe19a9dd9.
//
// Solidity: function setGuard(address guard) returns()
func (_ISafe *ISafeSession) SetGuard(guard common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.SetGuard(&_ISafe.TransactOpts, guard)
}

// SetGuard is a paid mutator transaction binding the contract method 0xe19a9dd9.
//
// Solidity: function setGuard(address guard) returns()
func (_ISafe *ISafeTransactorSession) SetGuard(guard common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.SetGuard(&_ISafe.TransactOpts, guard)
}

// SetModuleGuard is a paid mutator transaction binding the contract method 0xe068df37.
//
// Solidity: function setModuleGuard(address moduleGuard) returns()
func (_ISafe *ISafeTransactor) SetModuleGuard(opts *bind.TransactOpts, moduleGuard common.Address) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "setModuleGuard", moduleGuard)
}

// SetModuleGuard is a paid mutator transaction binding the contract method 0xe068df37.
//
// Solidity: function setModuleGuard(address moduleGuard) returns()
func (_ISafe *ISafeSession) SetModuleGuard(moduleGuard common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.SetModuleGuard(&_ISafe.TransactOpts, moduleGuard)
}

// SetModuleGuard is a paid mutator transaction binding the contract method 0xe068df37.
//
// Solidity: function setModuleGuard(address moduleGuard) returns()
func (_ISafe *ISafeTransactorSession) SetModuleGuard(moduleGuard common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.SetModuleGuard(&_ISafe.TransactOpts, moduleGuard)
}

// Setup is a paid mutator transaction binding the contract method 0xb63e800d.
//
// Solidity: function setup(address[] _owners, uint256 _threshold, address to, bytes data, address fallbackHandler, address paymentToken, uint256 payment, address paymentReceiver) returns()
func (_ISafe *ISafeTransactor) Setup(opts *bind.TransactOpts, _owners []common.Address, _threshold *big.Int, to common.Address, data []byte, fallbackHandler common.Address, paymentToken common.Address, payment *big.Int, paymentReceiver common.Address) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "setup", _owners, _threshold, to, data, fallbackHandler, paymentToken, payment, paymentReceiver)
}

// Setup is a paid mutator transaction binding the contract method 0xb63e800d.
//
// Solidity: function setup(address[] _owners, uint256 _threshold, address to, bytes data, address fallbackHandler, address paymentToken, uint256 payment, address paymentReceiver) returns()
func (_ISafe *ISafeSession) Setup(_owners []common.Address, _threshold *big.Int, to common.Address, data []byte, fallbackHandler common.Address, paymentToken common.Address, payment *big.Int, paymentReceiver common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.Setup(&_ISafe.TransactOpts, _owners, _threshold, to, data, fallbackHandler, paymentToken, payment, paymentReceiver)
}

// Setup is a paid mutator transaction binding the contract method 0xb63e800d.
//
// Solidity: function setup(address[] _owners, uint256 _threshold, address to, bytes data, address fallbackHandler, address paymentToken, uint256 payment, address paymentReceiver) returns()
func (_ISafe *ISafeTransactorSession) Setup(_owners []common.Address, _threshold *big.Int, to common.Address, data []byte, fallbackHandler common.Address, paymentToken common.Address, payment *big.Int, paymentReceiver common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.Setup(&_ISafe.TransactOpts, _owners, _threshold, to, data, fallbackHandler, paymentToken, payment, paymentReceiver)
}

// SwapOwner is a paid mutator transaction binding the contract method 0xe318b52b.
//
// Solidity: function swapOwner(address prevOwner, address oldOwner, address newOwner) returns()
func (_ISafe *ISafeTransactor) SwapOwner(opts *bind.TransactOpts, prevOwner common.Address, oldOwner common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _ISafe.contract.Transact(opts, "swapOwner", prevOwner, oldOwner, newOwner)
}

// SwapOwner is a paid mutator transaction binding the contract method 0xe318b52b.
//
// Solidity: function swapOwner(address prevOwner, address oldOwner, address newOwner) returns()
func (_ISafe *ISafeSession) SwapOwner(prevOwner common.Address, oldOwner common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.SwapOwner(&_ISafe.TransactOpts, prevOwner, oldOwner, newOwner)
}

// SwapOwner is a paid mutator transaction binding the contract method 0xe318b52b.
//
// Solidity: function swapOwner(address prevOwner, address oldOwner, address newOwner) returns()
func (_ISafe *ISafeTransactorSession) SwapOwner(prevOwner common.Address, oldOwner common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _ISafe.Contract.SwapOwner(&_ISafe.TransactOpts, prevOwner, oldOwner, newOwner)
}

// ISafeAddedOwnerIterator is returned from FilterAddedOwner and is used to iterate over the raw logs and unpacked data for AddedOwner events raised by the ISafe contract.
type ISafeAddedOwnerIterator struct {
	Event *ISafeAddedOwner // Event containing the contract specifics and raw log

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
func (it *ISafeAddedOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeAddedOwner)
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
		it.Event = new(ISafeAddedOwner)
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
func (it *ISafeAddedOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeAddedOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeAddedOwner represents a AddedOwner event raised by the ISafe contract.
type ISafeAddedOwner struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterAddedOwner is a free log retrieval operation binding the contract event 0x9465fa0c962cc76958e6373a993326400c1c94f8be2fe3a952adfa7f60b2ea26.
//
// Solidity: event AddedOwner(address indexed owner)
func (_ISafe *ISafeFilterer) FilterAddedOwner(opts *bind.FilterOpts, owner []common.Address) (*ISafeAddedOwnerIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "AddedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return &ISafeAddedOwnerIterator{contract: _ISafe.contract, event: "AddedOwner", logs: logs, sub: sub}, nil
}

// WatchAddedOwner is a free log subscription operation binding the contract event 0x9465fa0c962cc76958e6373a993326400c1c94f8be2fe3a952adfa7f60b2ea26.
//
// Solidity: event AddedOwner(address indexed owner)
func (_ISafe *ISafeFilterer) WatchAddedOwner(opts *bind.WatchOpts, sink chan<- *ISafeAddedOwner, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "AddedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeAddedOwner)
				if err := _ISafe.contract.UnpackLog(event, "AddedOwner", log); err != nil {
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
func (_ISafe *ISafeFilterer) ParseAddedOwner(log types.Log) (*ISafeAddedOwner, error) {
	event := new(ISafeAddedOwner)
	if err := _ISafe.contract.UnpackLog(event, "AddedOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeApproveHashIterator is returned from FilterApproveHash and is used to iterate over the raw logs and unpacked data for ApproveHash events raised by the ISafe contract.
type ISafeApproveHashIterator struct {
	Event *ISafeApproveHash // Event containing the contract specifics and raw log

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
func (it *ISafeApproveHashIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeApproveHash)
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
		it.Event = new(ISafeApproveHash)
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
func (it *ISafeApproveHashIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeApproveHashIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeApproveHash represents a ApproveHash event raised by the ISafe contract.
type ISafeApproveHash struct {
	ApprovedHash [32]byte
	Owner        common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterApproveHash is a free log retrieval operation binding the contract event 0xf2a0eb156472d1440255b0d7c1e19cc07115d1051fe605b0dce69acfec884d9c.
//
// Solidity: event ApproveHash(bytes32 indexed approvedHash, address indexed owner)
func (_ISafe *ISafeFilterer) FilterApproveHash(opts *bind.FilterOpts, approvedHash [][32]byte, owner []common.Address) (*ISafeApproveHashIterator, error) {

	var approvedHashRule []interface{}
	for _, approvedHashItem := range approvedHash {
		approvedHashRule = append(approvedHashRule, approvedHashItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "ApproveHash", approvedHashRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return &ISafeApproveHashIterator{contract: _ISafe.contract, event: "ApproveHash", logs: logs, sub: sub}, nil
}

// WatchApproveHash is a free log subscription operation binding the contract event 0xf2a0eb156472d1440255b0d7c1e19cc07115d1051fe605b0dce69acfec884d9c.
//
// Solidity: event ApproveHash(bytes32 indexed approvedHash, address indexed owner)
func (_ISafe *ISafeFilterer) WatchApproveHash(opts *bind.WatchOpts, sink chan<- *ISafeApproveHash, approvedHash [][32]byte, owner []common.Address) (event.Subscription, error) {

	var approvedHashRule []interface{}
	for _, approvedHashItem := range approvedHash {
		approvedHashRule = append(approvedHashRule, approvedHashItem)
	}
	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "ApproveHash", approvedHashRule, ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeApproveHash)
				if err := _ISafe.contract.UnpackLog(event, "ApproveHash", log); err != nil {
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

// ParseApproveHash is a log parse operation binding the contract event 0xf2a0eb156472d1440255b0d7c1e19cc07115d1051fe605b0dce69acfec884d9c.
//
// Solidity: event ApproveHash(bytes32 indexed approvedHash, address indexed owner)
func (_ISafe *ISafeFilterer) ParseApproveHash(log types.Log) (*ISafeApproveHash, error) {
	event := new(ISafeApproveHash)
	if err := _ISafe.contract.UnpackLog(event, "ApproveHash", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeChangedFallbackHandlerIterator is returned from FilterChangedFallbackHandler and is used to iterate over the raw logs and unpacked data for ChangedFallbackHandler events raised by the ISafe contract.
type ISafeChangedFallbackHandlerIterator struct {
	Event *ISafeChangedFallbackHandler // Event containing the contract specifics and raw log

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
func (it *ISafeChangedFallbackHandlerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeChangedFallbackHandler)
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
		it.Event = new(ISafeChangedFallbackHandler)
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
func (it *ISafeChangedFallbackHandlerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeChangedFallbackHandlerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeChangedFallbackHandler represents a ChangedFallbackHandler event raised by the ISafe contract.
type ISafeChangedFallbackHandler struct {
	Handler common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterChangedFallbackHandler is a free log retrieval operation binding the contract event 0x5ac6c46c93c8d0e53714ba3b53db3e7c046da994313d7ed0d192028bc7c228b0.
//
// Solidity: event ChangedFallbackHandler(address indexed handler)
func (_ISafe *ISafeFilterer) FilterChangedFallbackHandler(opts *bind.FilterOpts, handler []common.Address) (*ISafeChangedFallbackHandlerIterator, error) {

	var handlerRule []interface{}
	for _, handlerItem := range handler {
		handlerRule = append(handlerRule, handlerItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "ChangedFallbackHandler", handlerRule)
	if err != nil {
		return nil, err
	}
	return &ISafeChangedFallbackHandlerIterator{contract: _ISafe.contract, event: "ChangedFallbackHandler", logs: logs, sub: sub}, nil
}

// WatchChangedFallbackHandler is a free log subscription operation binding the contract event 0x5ac6c46c93c8d0e53714ba3b53db3e7c046da994313d7ed0d192028bc7c228b0.
//
// Solidity: event ChangedFallbackHandler(address indexed handler)
func (_ISafe *ISafeFilterer) WatchChangedFallbackHandler(opts *bind.WatchOpts, sink chan<- *ISafeChangedFallbackHandler, handler []common.Address) (event.Subscription, error) {

	var handlerRule []interface{}
	for _, handlerItem := range handler {
		handlerRule = append(handlerRule, handlerItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "ChangedFallbackHandler", handlerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeChangedFallbackHandler)
				if err := _ISafe.contract.UnpackLog(event, "ChangedFallbackHandler", log); err != nil {
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
func (_ISafe *ISafeFilterer) ParseChangedFallbackHandler(log types.Log) (*ISafeChangedFallbackHandler, error) {
	event := new(ISafeChangedFallbackHandler)
	if err := _ISafe.contract.UnpackLog(event, "ChangedFallbackHandler", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeChangedGuardIterator is returned from FilterChangedGuard and is used to iterate over the raw logs and unpacked data for ChangedGuard events raised by the ISafe contract.
type ISafeChangedGuardIterator struct {
	Event *ISafeChangedGuard // Event containing the contract specifics and raw log

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
func (it *ISafeChangedGuardIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeChangedGuard)
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
		it.Event = new(ISafeChangedGuard)
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
func (it *ISafeChangedGuardIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeChangedGuardIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeChangedGuard represents a ChangedGuard event raised by the ISafe contract.
type ISafeChangedGuard struct {
	Guard common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterChangedGuard is a free log retrieval operation binding the contract event 0x1151116914515bc0891ff9047a6cb32cf902546f83066499bcf8ba33d2353fa2.
//
// Solidity: event ChangedGuard(address indexed guard)
func (_ISafe *ISafeFilterer) FilterChangedGuard(opts *bind.FilterOpts, guard []common.Address) (*ISafeChangedGuardIterator, error) {

	var guardRule []interface{}
	for _, guardItem := range guard {
		guardRule = append(guardRule, guardItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "ChangedGuard", guardRule)
	if err != nil {
		return nil, err
	}
	return &ISafeChangedGuardIterator{contract: _ISafe.contract, event: "ChangedGuard", logs: logs, sub: sub}, nil
}

// WatchChangedGuard is a free log subscription operation binding the contract event 0x1151116914515bc0891ff9047a6cb32cf902546f83066499bcf8ba33d2353fa2.
//
// Solidity: event ChangedGuard(address indexed guard)
func (_ISafe *ISafeFilterer) WatchChangedGuard(opts *bind.WatchOpts, sink chan<- *ISafeChangedGuard, guard []common.Address) (event.Subscription, error) {

	var guardRule []interface{}
	for _, guardItem := range guard {
		guardRule = append(guardRule, guardItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "ChangedGuard", guardRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeChangedGuard)
				if err := _ISafe.contract.UnpackLog(event, "ChangedGuard", log); err != nil {
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
func (_ISafe *ISafeFilterer) ParseChangedGuard(log types.Log) (*ISafeChangedGuard, error) {
	event := new(ISafeChangedGuard)
	if err := _ISafe.contract.UnpackLog(event, "ChangedGuard", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeChangedModuleGuardIterator is returned from FilterChangedModuleGuard and is used to iterate over the raw logs and unpacked data for ChangedModuleGuard events raised by the ISafe contract.
type ISafeChangedModuleGuardIterator struct {
	Event *ISafeChangedModuleGuard // Event containing the contract specifics and raw log

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
func (it *ISafeChangedModuleGuardIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeChangedModuleGuard)
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
		it.Event = new(ISafeChangedModuleGuard)
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
func (it *ISafeChangedModuleGuardIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeChangedModuleGuardIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeChangedModuleGuard represents a ChangedModuleGuard event raised by the ISafe contract.
type ISafeChangedModuleGuard struct {
	ModuleGuard common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChangedModuleGuard is a free log retrieval operation binding the contract event 0xcd1966d6be16bc0c030cc741a06c6e0efaf8d00de2c8b6a9e11827e125de8bb8.
//
// Solidity: event ChangedModuleGuard(address indexed moduleGuard)
func (_ISafe *ISafeFilterer) FilterChangedModuleGuard(opts *bind.FilterOpts, moduleGuard []common.Address) (*ISafeChangedModuleGuardIterator, error) {

	var moduleGuardRule []interface{}
	for _, moduleGuardItem := range moduleGuard {
		moduleGuardRule = append(moduleGuardRule, moduleGuardItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "ChangedModuleGuard", moduleGuardRule)
	if err != nil {
		return nil, err
	}
	return &ISafeChangedModuleGuardIterator{contract: _ISafe.contract, event: "ChangedModuleGuard", logs: logs, sub: sub}, nil
}

// WatchChangedModuleGuard is a free log subscription operation binding the contract event 0xcd1966d6be16bc0c030cc741a06c6e0efaf8d00de2c8b6a9e11827e125de8bb8.
//
// Solidity: event ChangedModuleGuard(address indexed moduleGuard)
func (_ISafe *ISafeFilterer) WatchChangedModuleGuard(opts *bind.WatchOpts, sink chan<- *ISafeChangedModuleGuard, moduleGuard []common.Address) (event.Subscription, error) {

	var moduleGuardRule []interface{}
	for _, moduleGuardItem := range moduleGuard {
		moduleGuardRule = append(moduleGuardRule, moduleGuardItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "ChangedModuleGuard", moduleGuardRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeChangedModuleGuard)
				if err := _ISafe.contract.UnpackLog(event, "ChangedModuleGuard", log); err != nil {
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
func (_ISafe *ISafeFilterer) ParseChangedModuleGuard(log types.Log) (*ISafeChangedModuleGuard, error) {
	event := new(ISafeChangedModuleGuard)
	if err := _ISafe.contract.UnpackLog(event, "ChangedModuleGuard", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeChangedThresholdIterator is returned from FilterChangedThreshold and is used to iterate over the raw logs and unpacked data for ChangedThreshold events raised by the ISafe contract.
type ISafeChangedThresholdIterator struct {
	Event *ISafeChangedThreshold // Event containing the contract specifics and raw log

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
func (it *ISafeChangedThresholdIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeChangedThreshold)
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
		it.Event = new(ISafeChangedThreshold)
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
func (it *ISafeChangedThresholdIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeChangedThresholdIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeChangedThreshold represents a ChangedThreshold event raised by the ISafe contract.
type ISafeChangedThreshold struct {
	Threshold *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterChangedThreshold is a free log retrieval operation binding the contract event 0x610f7ff2b304ae8903c3de74c60c6ab1f7d6226b3f52c5161905bb5ad4039c93.
//
// Solidity: event ChangedThreshold(uint256 threshold)
func (_ISafe *ISafeFilterer) FilterChangedThreshold(opts *bind.FilterOpts) (*ISafeChangedThresholdIterator, error) {

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "ChangedThreshold")
	if err != nil {
		return nil, err
	}
	return &ISafeChangedThresholdIterator{contract: _ISafe.contract, event: "ChangedThreshold", logs: logs, sub: sub}, nil
}

// WatchChangedThreshold is a free log subscription operation binding the contract event 0x610f7ff2b304ae8903c3de74c60c6ab1f7d6226b3f52c5161905bb5ad4039c93.
//
// Solidity: event ChangedThreshold(uint256 threshold)
func (_ISafe *ISafeFilterer) WatchChangedThreshold(opts *bind.WatchOpts, sink chan<- *ISafeChangedThreshold) (event.Subscription, error) {

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "ChangedThreshold")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeChangedThreshold)
				if err := _ISafe.contract.UnpackLog(event, "ChangedThreshold", log); err != nil {
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
func (_ISafe *ISafeFilterer) ParseChangedThreshold(log types.Log) (*ISafeChangedThreshold, error) {
	event := new(ISafeChangedThreshold)
	if err := _ISafe.contract.UnpackLog(event, "ChangedThreshold", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeDisabledModuleIterator is returned from FilterDisabledModule and is used to iterate over the raw logs and unpacked data for DisabledModule events raised by the ISafe contract.
type ISafeDisabledModuleIterator struct {
	Event *ISafeDisabledModule // Event containing the contract specifics and raw log

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
func (it *ISafeDisabledModuleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeDisabledModule)
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
		it.Event = new(ISafeDisabledModule)
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
func (it *ISafeDisabledModuleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeDisabledModuleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeDisabledModule represents a DisabledModule event raised by the ISafe contract.
type ISafeDisabledModule struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterDisabledModule is a free log retrieval operation binding the contract event 0xaab4fa2b463f581b2b32cb3b7e3b704b9ce37cc209b5fb4d77e593ace4054276.
//
// Solidity: event DisabledModule(address indexed module)
func (_ISafe *ISafeFilterer) FilterDisabledModule(opts *bind.FilterOpts, module []common.Address) (*ISafeDisabledModuleIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "DisabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return &ISafeDisabledModuleIterator{contract: _ISafe.contract, event: "DisabledModule", logs: logs, sub: sub}, nil
}

// WatchDisabledModule is a free log subscription operation binding the contract event 0xaab4fa2b463f581b2b32cb3b7e3b704b9ce37cc209b5fb4d77e593ace4054276.
//
// Solidity: event DisabledModule(address indexed module)
func (_ISafe *ISafeFilterer) WatchDisabledModule(opts *bind.WatchOpts, sink chan<- *ISafeDisabledModule, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "DisabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeDisabledModule)
				if err := _ISafe.contract.UnpackLog(event, "DisabledModule", log); err != nil {
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
func (_ISafe *ISafeFilterer) ParseDisabledModule(log types.Log) (*ISafeDisabledModule, error) {
	event := new(ISafeDisabledModule)
	if err := _ISafe.contract.UnpackLog(event, "DisabledModule", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeEnabledModuleIterator is returned from FilterEnabledModule and is used to iterate over the raw logs and unpacked data for EnabledModule events raised by the ISafe contract.
type ISafeEnabledModuleIterator struct {
	Event *ISafeEnabledModule // Event containing the contract specifics and raw log

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
func (it *ISafeEnabledModuleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeEnabledModule)
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
		it.Event = new(ISafeEnabledModule)
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
func (it *ISafeEnabledModuleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeEnabledModuleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeEnabledModule represents a EnabledModule event raised by the ISafe contract.
type ISafeEnabledModule struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterEnabledModule is a free log retrieval operation binding the contract event 0xecdf3a3effea5783a3c4c2140e677577666428d44ed9d474a0b3a4c9943f8440.
//
// Solidity: event EnabledModule(address indexed module)
func (_ISafe *ISafeFilterer) FilterEnabledModule(opts *bind.FilterOpts, module []common.Address) (*ISafeEnabledModuleIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "EnabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return &ISafeEnabledModuleIterator{contract: _ISafe.contract, event: "EnabledModule", logs: logs, sub: sub}, nil
}

// WatchEnabledModule is a free log subscription operation binding the contract event 0xecdf3a3effea5783a3c4c2140e677577666428d44ed9d474a0b3a4c9943f8440.
//
// Solidity: event EnabledModule(address indexed module)
func (_ISafe *ISafeFilterer) WatchEnabledModule(opts *bind.WatchOpts, sink chan<- *ISafeEnabledModule, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "EnabledModule", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeEnabledModule)
				if err := _ISafe.contract.UnpackLog(event, "EnabledModule", log); err != nil {
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
func (_ISafe *ISafeFilterer) ParseEnabledModule(log types.Log) (*ISafeEnabledModule, error) {
	event := new(ISafeEnabledModule)
	if err := _ISafe.contract.UnpackLog(event, "EnabledModule", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeExecutionFailureIterator is returned from FilterExecutionFailure and is used to iterate over the raw logs and unpacked data for ExecutionFailure events raised by the ISafe contract.
type ISafeExecutionFailureIterator struct {
	Event *ISafeExecutionFailure // Event containing the contract specifics and raw log

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
func (it *ISafeExecutionFailureIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeExecutionFailure)
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
		it.Event = new(ISafeExecutionFailure)
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
func (it *ISafeExecutionFailureIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeExecutionFailureIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeExecutionFailure represents a ExecutionFailure event raised by the ISafe contract.
type ISafeExecutionFailure struct {
	TxHash  [32]byte
	Payment *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterExecutionFailure is a free log retrieval operation binding the contract event 0x23428b18acfb3ea64b08dc0c1d296ea9c09702c09083ca5272e64d115b687d23.
//
// Solidity: event ExecutionFailure(bytes32 indexed txHash, uint256 payment)
func (_ISafe *ISafeFilterer) FilterExecutionFailure(opts *bind.FilterOpts, txHash [][32]byte) (*ISafeExecutionFailureIterator, error) {

	var txHashRule []interface{}
	for _, txHashItem := range txHash {
		txHashRule = append(txHashRule, txHashItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "ExecutionFailure", txHashRule)
	if err != nil {
		return nil, err
	}
	return &ISafeExecutionFailureIterator{contract: _ISafe.contract, event: "ExecutionFailure", logs: logs, sub: sub}, nil
}

// WatchExecutionFailure is a free log subscription operation binding the contract event 0x23428b18acfb3ea64b08dc0c1d296ea9c09702c09083ca5272e64d115b687d23.
//
// Solidity: event ExecutionFailure(bytes32 indexed txHash, uint256 payment)
func (_ISafe *ISafeFilterer) WatchExecutionFailure(opts *bind.WatchOpts, sink chan<- *ISafeExecutionFailure, txHash [][32]byte) (event.Subscription, error) {

	var txHashRule []interface{}
	for _, txHashItem := range txHash {
		txHashRule = append(txHashRule, txHashItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "ExecutionFailure", txHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeExecutionFailure)
				if err := _ISafe.contract.UnpackLog(event, "ExecutionFailure", log); err != nil {
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

// ParseExecutionFailure is a log parse operation binding the contract event 0x23428b18acfb3ea64b08dc0c1d296ea9c09702c09083ca5272e64d115b687d23.
//
// Solidity: event ExecutionFailure(bytes32 indexed txHash, uint256 payment)
func (_ISafe *ISafeFilterer) ParseExecutionFailure(log types.Log) (*ISafeExecutionFailure, error) {
	event := new(ISafeExecutionFailure)
	if err := _ISafe.contract.UnpackLog(event, "ExecutionFailure", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeExecutionFromModuleFailureIterator is returned from FilterExecutionFromModuleFailure and is used to iterate over the raw logs and unpacked data for ExecutionFromModuleFailure events raised by the ISafe contract.
type ISafeExecutionFromModuleFailureIterator struct {
	Event *ISafeExecutionFromModuleFailure // Event containing the contract specifics and raw log

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
func (it *ISafeExecutionFromModuleFailureIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeExecutionFromModuleFailure)
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
		it.Event = new(ISafeExecutionFromModuleFailure)
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
func (it *ISafeExecutionFromModuleFailureIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeExecutionFromModuleFailureIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeExecutionFromModuleFailure represents a ExecutionFromModuleFailure event raised by the ISafe contract.
type ISafeExecutionFromModuleFailure struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterExecutionFromModuleFailure is a free log retrieval operation binding the contract event 0xacd2c8702804128fdb0db2bb49f6d127dd0181c13fd45dbfe16de0930e2bd375.
//
// Solidity: event ExecutionFromModuleFailure(address indexed module)
func (_ISafe *ISafeFilterer) FilterExecutionFromModuleFailure(opts *bind.FilterOpts, module []common.Address) (*ISafeExecutionFromModuleFailureIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "ExecutionFromModuleFailure", moduleRule)
	if err != nil {
		return nil, err
	}
	return &ISafeExecutionFromModuleFailureIterator{contract: _ISafe.contract, event: "ExecutionFromModuleFailure", logs: logs, sub: sub}, nil
}

// WatchExecutionFromModuleFailure is a free log subscription operation binding the contract event 0xacd2c8702804128fdb0db2bb49f6d127dd0181c13fd45dbfe16de0930e2bd375.
//
// Solidity: event ExecutionFromModuleFailure(address indexed module)
func (_ISafe *ISafeFilterer) WatchExecutionFromModuleFailure(opts *bind.WatchOpts, sink chan<- *ISafeExecutionFromModuleFailure, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "ExecutionFromModuleFailure", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeExecutionFromModuleFailure)
				if err := _ISafe.contract.UnpackLog(event, "ExecutionFromModuleFailure", log); err != nil {
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
func (_ISafe *ISafeFilterer) ParseExecutionFromModuleFailure(log types.Log) (*ISafeExecutionFromModuleFailure, error) {
	event := new(ISafeExecutionFromModuleFailure)
	if err := _ISafe.contract.UnpackLog(event, "ExecutionFromModuleFailure", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeExecutionFromModuleSuccessIterator is returned from FilterExecutionFromModuleSuccess and is used to iterate over the raw logs and unpacked data for ExecutionFromModuleSuccess events raised by the ISafe contract.
type ISafeExecutionFromModuleSuccessIterator struct {
	Event *ISafeExecutionFromModuleSuccess // Event containing the contract specifics and raw log

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
func (it *ISafeExecutionFromModuleSuccessIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeExecutionFromModuleSuccess)
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
		it.Event = new(ISafeExecutionFromModuleSuccess)
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
func (it *ISafeExecutionFromModuleSuccessIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeExecutionFromModuleSuccessIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeExecutionFromModuleSuccess represents a ExecutionFromModuleSuccess event raised by the ISafe contract.
type ISafeExecutionFromModuleSuccess struct {
	Module common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterExecutionFromModuleSuccess is a free log retrieval operation binding the contract event 0x6895c13664aa4f67288b25d7a21d7aaa34916e355fb9b6fae0a139a9085becb8.
//
// Solidity: event ExecutionFromModuleSuccess(address indexed module)
func (_ISafe *ISafeFilterer) FilterExecutionFromModuleSuccess(opts *bind.FilterOpts, module []common.Address) (*ISafeExecutionFromModuleSuccessIterator, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "ExecutionFromModuleSuccess", moduleRule)
	if err != nil {
		return nil, err
	}
	return &ISafeExecutionFromModuleSuccessIterator{contract: _ISafe.contract, event: "ExecutionFromModuleSuccess", logs: logs, sub: sub}, nil
}

// WatchExecutionFromModuleSuccess is a free log subscription operation binding the contract event 0x6895c13664aa4f67288b25d7a21d7aaa34916e355fb9b6fae0a139a9085becb8.
//
// Solidity: event ExecutionFromModuleSuccess(address indexed module)
func (_ISafe *ISafeFilterer) WatchExecutionFromModuleSuccess(opts *bind.WatchOpts, sink chan<- *ISafeExecutionFromModuleSuccess, module []common.Address) (event.Subscription, error) {

	var moduleRule []interface{}
	for _, moduleItem := range module {
		moduleRule = append(moduleRule, moduleItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "ExecutionFromModuleSuccess", moduleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeExecutionFromModuleSuccess)
				if err := _ISafe.contract.UnpackLog(event, "ExecutionFromModuleSuccess", log); err != nil {
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
func (_ISafe *ISafeFilterer) ParseExecutionFromModuleSuccess(log types.Log) (*ISafeExecutionFromModuleSuccess, error) {
	event := new(ISafeExecutionFromModuleSuccess)
	if err := _ISafe.contract.UnpackLog(event, "ExecutionFromModuleSuccess", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeExecutionSuccessIterator is returned from FilterExecutionSuccess and is used to iterate over the raw logs and unpacked data for ExecutionSuccess events raised by the ISafe contract.
type ISafeExecutionSuccessIterator struct {
	Event *ISafeExecutionSuccess // Event containing the contract specifics and raw log

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
func (it *ISafeExecutionSuccessIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeExecutionSuccess)
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
		it.Event = new(ISafeExecutionSuccess)
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
func (it *ISafeExecutionSuccessIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeExecutionSuccessIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeExecutionSuccess represents a ExecutionSuccess event raised by the ISafe contract.
type ISafeExecutionSuccess struct {
	TxHash  [32]byte
	Payment *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterExecutionSuccess is a free log retrieval operation binding the contract event 0x442e715f626346e8c54381002da614f62bee8d27386535b2521ec8540898556e.
//
// Solidity: event ExecutionSuccess(bytes32 indexed txHash, uint256 payment)
func (_ISafe *ISafeFilterer) FilterExecutionSuccess(opts *bind.FilterOpts, txHash [][32]byte) (*ISafeExecutionSuccessIterator, error) {

	var txHashRule []interface{}
	for _, txHashItem := range txHash {
		txHashRule = append(txHashRule, txHashItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "ExecutionSuccess", txHashRule)
	if err != nil {
		return nil, err
	}
	return &ISafeExecutionSuccessIterator{contract: _ISafe.contract, event: "ExecutionSuccess", logs: logs, sub: sub}, nil
}

// WatchExecutionSuccess is a free log subscription operation binding the contract event 0x442e715f626346e8c54381002da614f62bee8d27386535b2521ec8540898556e.
//
// Solidity: event ExecutionSuccess(bytes32 indexed txHash, uint256 payment)
func (_ISafe *ISafeFilterer) WatchExecutionSuccess(opts *bind.WatchOpts, sink chan<- *ISafeExecutionSuccess, txHash [][32]byte) (event.Subscription, error) {

	var txHashRule []interface{}
	for _, txHashItem := range txHash {
		txHashRule = append(txHashRule, txHashItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "ExecutionSuccess", txHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeExecutionSuccess)
				if err := _ISafe.contract.UnpackLog(event, "ExecutionSuccess", log); err != nil {
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

// ParseExecutionSuccess is a log parse operation binding the contract event 0x442e715f626346e8c54381002da614f62bee8d27386535b2521ec8540898556e.
//
// Solidity: event ExecutionSuccess(bytes32 indexed txHash, uint256 payment)
func (_ISafe *ISafeFilterer) ParseExecutionSuccess(log types.Log) (*ISafeExecutionSuccess, error) {
	event := new(ISafeExecutionSuccess)
	if err := _ISafe.contract.UnpackLog(event, "ExecutionSuccess", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeRemovedOwnerIterator is returned from FilterRemovedOwner and is used to iterate over the raw logs and unpacked data for RemovedOwner events raised by the ISafe contract.
type ISafeRemovedOwnerIterator struct {
	Event *ISafeRemovedOwner // Event containing the contract specifics and raw log

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
func (it *ISafeRemovedOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeRemovedOwner)
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
		it.Event = new(ISafeRemovedOwner)
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
func (it *ISafeRemovedOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeRemovedOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeRemovedOwner represents a RemovedOwner event raised by the ISafe contract.
type ISafeRemovedOwner struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterRemovedOwner is a free log retrieval operation binding the contract event 0xf8d49fc529812e9a7c5c50e69c20f0dccc0db8fa95c98bc58cc9a4f1c1299eaf.
//
// Solidity: event RemovedOwner(address indexed owner)
func (_ISafe *ISafeFilterer) FilterRemovedOwner(opts *bind.FilterOpts, owner []common.Address) (*ISafeRemovedOwnerIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "RemovedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return &ISafeRemovedOwnerIterator{contract: _ISafe.contract, event: "RemovedOwner", logs: logs, sub: sub}, nil
}

// WatchRemovedOwner is a free log subscription operation binding the contract event 0xf8d49fc529812e9a7c5c50e69c20f0dccc0db8fa95c98bc58cc9a4f1c1299eaf.
//
// Solidity: event RemovedOwner(address indexed owner)
func (_ISafe *ISafeFilterer) WatchRemovedOwner(opts *bind.WatchOpts, sink chan<- *ISafeRemovedOwner, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "RemovedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeRemovedOwner)
				if err := _ISafe.contract.UnpackLog(event, "RemovedOwner", log); err != nil {
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
func (_ISafe *ISafeFilterer) ParseRemovedOwner(log types.Log) (*ISafeRemovedOwner, error) {
	event := new(ISafeRemovedOwner)
	if err := _ISafe.contract.UnpackLog(event, "RemovedOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeSafeSetupIterator is returned from FilterSafeSetup and is used to iterate over the raw logs and unpacked data for SafeSetup events raised by the ISafe contract.
type ISafeSafeSetupIterator struct {
	Event *ISafeSafeSetup // Event containing the contract specifics and raw log

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
func (it *ISafeSafeSetupIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeSafeSetup)
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
		it.Event = new(ISafeSafeSetup)
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
func (it *ISafeSafeSetupIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeSafeSetupIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeSafeSetup represents a SafeSetup event raised by the ISafe contract.
type ISafeSafeSetup struct {
	Initiator       common.Address
	Owners          []common.Address
	Threshold       *big.Int
	Initializer     common.Address
	FallbackHandler common.Address
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterSafeSetup is a free log retrieval operation binding the contract event 0x141df868a6331af528e38c83b7aa03edc19be66e37ae67f9285bf4f8e3c6a1a8.
//
// Solidity: event SafeSetup(address indexed initiator, address[] owners, uint256 threshold, address initializer, address fallbackHandler)
func (_ISafe *ISafeFilterer) FilterSafeSetup(opts *bind.FilterOpts, initiator []common.Address) (*ISafeSafeSetupIterator, error) {

	var initiatorRule []interface{}
	for _, initiatorItem := range initiator {
		initiatorRule = append(initiatorRule, initiatorItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "SafeSetup", initiatorRule)
	if err != nil {
		return nil, err
	}
	return &ISafeSafeSetupIterator{contract: _ISafe.contract, event: "SafeSetup", logs: logs, sub: sub}, nil
}

// WatchSafeSetup is a free log subscription operation binding the contract event 0x141df868a6331af528e38c83b7aa03edc19be66e37ae67f9285bf4f8e3c6a1a8.
//
// Solidity: event SafeSetup(address indexed initiator, address[] owners, uint256 threshold, address initializer, address fallbackHandler)
func (_ISafe *ISafeFilterer) WatchSafeSetup(opts *bind.WatchOpts, sink chan<- *ISafeSafeSetup, initiator []common.Address) (event.Subscription, error) {

	var initiatorRule []interface{}
	for _, initiatorItem := range initiator {
		initiatorRule = append(initiatorRule, initiatorItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "SafeSetup", initiatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeSafeSetup)
				if err := _ISafe.contract.UnpackLog(event, "SafeSetup", log); err != nil {
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

// ParseSafeSetup is a log parse operation binding the contract event 0x141df868a6331af528e38c83b7aa03edc19be66e37ae67f9285bf4f8e3c6a1a8.
//
// Solidity: event SafeSetup(address indexed initiator, address[] owners, uint256 threshold, address initializer, address fallbackHandler)
func (_ISafe *ISafeFilterer) ParseSafeSetup(log types.Log) (*ISafeSafeSetup, error) {
	event := new(ISafeSafeSetup)
	if err := _ISafe.contract.UnpackLog(event, "SafeSetup", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeSignMsgIterator is returned from FilterSignMsg and is used to iterate over the raw logs and unpacked data for SignMsg events raised by the ISafe contract.
type ISafeSignMsgIterator struct {
	Event *ISafeSignMsg // Event containing the contract specifics and raw log

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
func (it *ISafeSignMsgIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISafeSignMsg)
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
		it.Event = new(ISafeSignMsg)
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
func (it *ISafeSignMsgIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISafeSignMsgIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISafeSignMsg represents a SignMsg event raised by the ISafe contract.
type ISafeSignMsg struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterSignMsg is a free log retrieval operation binding the contract event 0xe7f4675038f4f6034dfcbbb24c4dc08e4ebf10eb9d257d3d02c0f38d122ac6e4.
//
// Solidity: event SignMsg(bytes32 indexed msgHash)
func (_ISafe *ISafeFilterer) FilterSignMsg(opts *bind.FilterOpts, msgHash [][32]byte) (*ISafeSignMsgIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _ISafe.contract.FilterLogs(opts, "SignMsg", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &ISafeSignMsgIterator{contract: _ISafe.contract, event: "SignMsg", logs: logs, sub: sub}, nil
}

// WatchSignMsg is a free log subscription operation binding the contract event 0xe7f4675038f4f6034dfcbbb24c4dc08e4ebf10eb9d257d3d02c0f38d122ac6e4.
//
// Solidity: event SignMsg(bytes32 indexed msgHash)
func (_ISafe *ISafeFilterer) WatchSignMsg(opts *bind.WatchOpts, sink chan<- *ISafeSignMsg, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _ISafe.contract.WatchLogs(opts, "SignMsg", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISafeSignMsg)
				if err := _ISafe.contract.UnpackLog(event, "SignMsg", log); err != nil {
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

// ParseSignMsg is a log parse operation binding the contract event 0xe7f4675038f4f6034dfcbbb24c4dc08e4ebf10eb9d257d3d02c0f38d122ac6e4.
//
// Solidity: event SignMsg(bytes32 indexed msgHash)
func (_ISafe *ISafeFilterer) ParseSignMsg(log types.Log) (*ISafeSignMsg, error) {
	event := new(ISafeSignMsg)
	if err := _ISafe.contract.UnpackLog(event, "SignMsg", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISignatureValidatorMetaData contains all meta data concerning the ISignatureValidator contract.
var ISignatureValidatorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_hash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"_signature\",\"type\":\"bytes\"}],\"name\":\"isValidSignature\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
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

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 _hash, bytes _signature) view returns(bytes4)
func (_ISignatureValidator *ISignatureValidatorCaller) IsValidSignature(opts *bind.CallOpts, _hash [32]byte, _signature []byte) ([4]byte, error) {
	var out []interface{}
	err := _ISignatureValidator.contract.Call(opts, &out, "isValidSignature", _hash, _signature)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 _hash, bytes _signature) view returns(bytes4)
func (_ISignatureValidator *ISignatureValidatorSession) IsValidSignature(_hash [32]byte, _signature []byte) ([4]byte, error) {
	return _ISignatureValidator.Contract.IsValidSignature(&_ISignatureValidator.CallOpts, _hash, _signature)
}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 _hash, bytes _signature) view returns(bytes4)
func (_ISignatureValidator *ISignatureValidatorCallerSession) IsValidSignature(_hash [32]byte, _signature []byte) ([4]byte, error) {
	return _ISignatureValidator.Contract.IsValidSignature(&_ISignatureValidator.CallOpts, _hash, _signature)
}

// ISignatureValidatorConstantsMetaData contains all meta data concerning the ISignatureValidatorConstants contract.
var ISignatureValidatorConstantsMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x6080604052348015600f57600080fd5b50603f80601d6000396000f3fe6080604052600080fdfea2646970667358221220bad1115a484680c6b92eba806697a18753355a1b6c3143bc504130df45e7039264736f6c63430007060033",
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
