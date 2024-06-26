// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package accessorsgen

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

// SimulateTxAccessorMetaData contains all meta data concerning the SimulateTxAccessor contract.
var SimulateTxAccessorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"enumEnum.Operation\",\"name\":\"operation\",\"type\":\"uint8\"}],\"name\":\"simulate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"estimate\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b503073ffffffffffffffffffffffffffffffffffffffff1660808173ffffffffffffffffffffffffffffffffffffffff1660601b8152505060805160601c6103526100656000398061017052506103526000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c80631c5fb21114610030575b600080fd5b6100de6004803603608081101561004657600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291908035906020019064010000000081111561008d57600080fd5b82018360208201111561009f57600080fd5b803590602001918460018302840111640100000000831117156100c157600080fd5b9091929391929390803560ff169060200190929190505050610169565b60405180848152602001831515815260200180602001828103825283818151815260200191508051906020019080838360005b8381101561012c578082015181840152602081019050610111565b50505050905090810190601f1680156101595780820380516001836020036101000a031916815260200191505b5094505050505060405180910390f35b60008060607f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff163073ffffffffffffffffffffffffffffffffffffffff161415610213576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260398152602001806102e46039913960400191505060405180910390fd5b60005a9050610269898989898080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f82011690508083019250505050505050885a610297565b92505a8103935060405160203d0181016040523d81523d6000602083013e8092505050955095509592505050565b60006001808111156102a557fe5b8360018111156102b157fe5b14156102ca576000808551602087018986f490506102da565b600080855160208701888a87f190505b9594505050505056fe53696d756c61746554784163636573736f722073686f756c64206f6e6c792062652063616c6c6564207669612064656c656761746563616c6ca2646970667358221220fd430144adb49f3c41232dfb6304536647edd8ba193872e74bf6864e1b781cca64736f6c63430007060033",
}

// SimulateTxAccessorABI is the input ABI used to generate the binding from.
// Deprecated: Use SimulateTxAccessorMetaData.ABI instead.
var SimulateTxAccessorABI = SimulateTxAccessorMetaData.ABI

// SimulateTxAccessorBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SimulateTxAccessorMetaData.Bin instead.
var SimulateTxAccessorBin = SimulateTxAccessorMetaData.Bin

// DeploySimulateTxAccessor deploys a new Ethereum contract, binding an instance of SimulateTxAccessor to it.
func DeploySimulateTxAccessor(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SimulateTxAccessor, error) {
	parsed, err := SimulateTxAccessorMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SimulateTxAccessorBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SimulateTxAccessor{SimulateTxAccessorCaller: SimulateTxAccessorCaller{contract: contract}, SimulateTxAccessorTransactor: SimulateTxAccessorTransactor{contract: contract}, SimulateTxAccessorFilterer: SimulateTxAccessorFilterer{contract: contract}}, nil
}

// SimulateTxAccessor is an auto generated Go binding around an Ethereum contract.
type SimulateTxAccessor struct {
	SimulateTxAccessorCaller     // Read-only binding to the contract
	SimulateTxAccessorTransactor // Write-only binding to the contract
	SimulateTxAccessorFilterer   // Log filterer for contract events
}

// SimulateTxAccessorCaller is an auto generated read-only Go binding around an Ethereum contract.
type SimulateTxAccessorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimulateTxAccessorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SimulateTxAccessorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimulateTxAccessorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SimulateTxAccessorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimulateTxAccessorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SimulateTxAccessorSession struct {
	Contract     *SimulateTxAccessor // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// SimulateTxAccessorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SimulateTxAccessorCallerSession struct {
	Contract *SimulateTxAccessorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// SimulateTxAccessorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SimulateTxAccessorTransactorSession struct {
	Contract     *SimulateTxAccessorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// SimulateTxAccessorRaw is an auto generated low-level Go binding around an Ethereum contract.
type SimulateTxAccessorRaw struct {
	Contract *SimulateTxAccessor // Generic contract binding to access the raw methods on
}

// SimulateTxAccessorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SimulateTxAccessorCallerRaw struct {
	Contract *SimulateTxAccessorCaller // Generic read-only contract binding to access the raw methods on
}

// SimulateTxAccessorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SimulateTxAccessorTransactorRaw struct {
	Contract *SimulateTxAccessorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSimulateTxAccessor creates a new instance of SimulateTxAccessor, bound to a specific deployed contract.
func NewSimulateTxAccessor(address common.Address, backend bind.ContractBackend) (*SimulateTxAccessor, error) {
	contract, err := bindSimulateTxAccessor(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SimulateTxAccessor{SimulateTxAccessorCaller: SimulateTxAccessorCaller{contract: contract}, SimulateTxAccessorTransactor: SimulateTxAccessorTransactor{contract: contract}, SimulateTxAccessorFilterer: SimulateTxAccessorFilterer{contract: contract}}, nil
}

// NewSimulateTxAccessorCaller creates a new read-only instance of SimulateTxAccessor, bound to a specific deployed contract.
func NewSimulateTxAccessorCaller(address common.Address, caller bind.ContractCaller) (*SimulateTxAccessorCaller, error) {
	contract, err := bindSimulateTxAccessor(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SimulateTxAccessorCaller{contract: contract}, nil
}

// NewSimulateTxAccessorTransactor creates a new write-only instance of SimulateTxAccessor, bound to a specific deployed contract.
func NewSimulateTxAccessorTransactor(address common.Address, transactor bind.ContractTransactor) (*SimulateTxAccessorTransactor, error) {
	contract, err := bindSimulateTxAccessor(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SimulateTxAccessorTransactor{contract: contract}, nil
}

// NewSimulateTxAccessorFilterer creates a new log filterer instance of SimulateTxAccessor, bound to a specific deployed contract.
func NewSimulateTxAccessorFilterer(address common.Address, filterer bind.ContractFilterer) (*SimulateTxAccessorFilterer, error) {
	contract, err := bindSimulateTxAccessor(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SimulateTxAccessorFilterer{contract: contract}, nil
}

// bindSimulateTxAccessor binds a generic wrapper to an already deployed contract.
func bindSimulateTxAccessor(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SimulateTxAccessorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SimulateTxAccessor *SimulateTxAccessorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SimulateTxAccessor.Contract.SimulateTxAccessorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SimulateTxAccessor *SimulateTxAccessorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimulateTxAccessor.Contract.SimulateTxAccessorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SimulateTxAccessor *SimulateTxAccessorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SimulateTxAccessor.Contract.SimulateTxAccessorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SimulateTxAccessor *SimulateTxAccessorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SimulateTxAccessor.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SimulateTxAccessor *SimulateTxAccessorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimulateTxAccessor.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SimulateTxAccessor *SimulateTxAccessorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SimulateTxAccessor.Contract.contract.Transact(opts, method, params...)
}

// Simulate is a paid mutator transaction binding the contract method 0x1c5fb211.
//
// Solidity: function simulate(address to, uint256 value, bytes data, uint8 operation) returns(uint256 estimate, bool success, bytes returnData)
func (_SimulateTxAccessor *SimulateTxAccessorTransactor) Simulate(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _SimulateTxAccessor.contract.Transact(opts, "simulate", to, value, data, operation)
}

// Simulate is a paid mutator transaction binding the contract method 0x1c5fb211.
//
// Solidity: function simulate(address to, uint256 value, bytes data, uint8 operation) returns(uint256 estimate, bool success, bytes returnData)
func (_SimulateTxAccessor *SimulateTxAccessorSession) Simulate(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _SimulateTxAccessor.Contract.Simulate(&_SimulateTxAccessor.TransactOpts, to, value, data, operation)
}

// Simulate is a paid mutator transaction binding the contract method 0x1c5fb211.
//
// Solidity: function simulate(address to, uint256 value, bytes data, uint8 operation) returns(uint256 estimate, bool success, bytes returnData)
func (_SimulateTxAccessor *SimulateTxAccessorTransactorSession) Simulate(to common.Address, value *big.Int, data []byte, operation uint8) (*types.Transaction, error) {
	return _SimulateTxAccessor.Contract.Simulate(&_SimulateTxAccessor.TransactOpts, to, value, data, operation)
}
