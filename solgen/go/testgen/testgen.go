// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package testgen

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

// UserOperation is an auto generated low-level Go binding around an user-defined struct.
type UserOperation struct {
	Sender               common.Address
	Nonce                *big.Int
	InitCode             []byte
	CallData             []byte
	CallGasLimit         *big.Int
	VerificationGasLimit *big.Int
	PreVerificationGas   *big.Int
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
	PaymasterAndData     []byte
	Signature            []byte
}

// DelegateCallerMetaData contains all meta data concerning the DelegateCaller contract.
var DelegateCallerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_called\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_calldata\",\"type\":\"bytes\"}],\"name\":\"makeDelegatecall\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610296806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063e632e17214610030575b600080fd5b6101096004803603604081101561004657600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019064010000000081111561008357600080fd5b82018360208201111561009557600080fd5b803590602001918460018302840111640100000000831117156100b757600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f82011690508083019250505050505050919291929050505061018d565b60405180831515815260200180602001828103825283818151815260200191508051906020019080838360005b83811015610151578082015181840152602081019050610136565b50505050905090810190601f16801561017e5780820380516001836020036101000a031916815260200191505b50935050505060405180910390f35b600060608373ffffffffffffffffffffffffffffffffffffffff16836040518082805190602001908083835b602083106101dc57805182526020820191506020810190506020830392506101b9565b6001836020036101000a038019825116818451168082178552505050505050905001915050600060405180830381855af49150503d806000811461023c576040519150601f19603f3d011682016040523d82523d6000602084013e610241565b606091505b50809250819350505081610259573d6000803e3d6000fd5b925092905056fea2646970667358221220588e0081c5d3e0cfad27619350765395f20d7d3c6598ac5f61dd4fb24aa8a67364736f6c63430007060033",
}

// DelegateCallerABI is the input ABI used to generate the binding from.
// Deprecated: Use DelegateCallerMetaData.ABI instead.
var DelegateCallerABI = DelegateCallerMetaData.ABI

// DelegateCallerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DelegateCallerMetaData.Bin instead.
var DelegateCallerBin = DelegateCallerMetaData.Bin

// DeployDelegateCaller deploys a new Ethereum contract, binding an instance of DelegateCaller to it.
func DeployDelegateCaller(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *DelegateCaller, error) {
	parsed, err := DelegateCallerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DelegateCallerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DelegateCaller{DelegateCallerCaller: DelegateCallerCaller{contract: contract}, DelegateCallerTransactor: DelegateCallerTransactor{contract: contract}, DelegateCallerFilterer: DelegateCallerFilterer{contract: contract}}, nil
}

// DelegateCaller is an auto generated Go binding around an Ethereum contract.
type DelegateCaller struct {
	DelegateCallerCaller     // Read-only binding to the contract
	DelegateCallerTransactor // Write-only binding to the contract
	DelegateCallerFilterer   // Log filterer for contract events
}

// DelegateCallerCaller is an auto generated read-only Go binding around an Ethereum contract.
type DelegateCallerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateCallerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DelegateCallerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateCallerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DelegateCallerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateCallerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DelegateCallerSession struct {
	Contract     *DelegateCaller   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DelegateCallerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DelegateCallerCallerSession struct {
	Contract *DelegateCallerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// DelegateCallerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DelegateCallerTransactorSession struct {
	Contract     *DelegateCallerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// DelegateCallerRaw is an auto generated low-level Go binding around an Ethereum contract.
type DelegateCallerRaw struct {
	Contract *DelegateCaller // Generic contract binding to access the raw methods on
}

// DelegateCallerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DelegateCallerCallerRaw struct {
	Contract *DelegateCallerCaller // Generic read-only contract binding to access the raw methods on
}

// DelegateCallerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DelegateCallerTransactorRaw struct {
	Contract *DelegateCallerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDelegateCaller creates a new instance of DelegateCaller, bound to a specific deployed contract.
func NewDelegateCaller(address common.Address, backend bind.ContractBackend) (*DelegateCaller, error) {
	contract, err := bindDelegateCaller(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DelegateCaller{DelegateCallerCaller: DelegateCallerCaller{contract: contract}, DelegateCallerTransactor: DelegateCallerTransactor{contract: contract}, DelegateCallerFilterer: DelegateCallerFilterer{contract: contract}}, nil
}

// NewDelegateCallerCaller creates a new read-only instance of DelegateCaller, bound to a specific deployed contract.
func NewDelegateCallerCaller(address common.Address, caller bind.ContractCaller) (*DelegateCallerCaller, error) {
	contract, err := bindDelegateCaller(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DelegateCallerCaller{contract: contract}, nil
}

// NewDelegateCallerTransactor creates a new write-only instance of DelegateCaller, bound to a specific deployed contract.
func NewDelegateCallerTransactor(address common.Address, transactor bind.ContractTransactor) (*DelegateCallerTransactor, error) {
	contract, err := bindDelegateCaller(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DelegateCallerTransactor{contract: contract}, nil
}

// NewDelegateCallerFilterer creates a new log filterer instance of DelegateCaller, bound to a specific deployed contract.
func NewDelegateCallerFilterer(address common.Address, filterer bind.ContractFilterer) (*DelegateCallerFilterer, error) {
	contract, err := bindDelegateCaller(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DelegateCallerFilterer{contract: contract}, nil
}

// bindDelegateCaller binds a generic wrapper to an already deployed contract.
func bindDelegateCaller(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DelegateCallerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelegateCaller *DelegateCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelegateCaller.Contract.DelegateCallerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelegateCaller *DelegateCallerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateCaller.Contract.DelegateCallerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelegateCaller *DelegateCallerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelegateCaller.Contract.DelegateCallerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelegateCaller *DelegateCallerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelegateCaller.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelegateCaller *DelegateCallerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateCaller.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelegateCaller *DelegateCallerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelegateCaller.Contract.contract.Transact(opts, method, params...)
}

// MakeDelegatecall is a paid mutator transaction binding the contract method 0xe632e172.
//
// Solidity: function makeDelegatecall(address _called, bytes _calldata) returns(bool success, bytes returnData)
func (_DelegateCaller *DelegateCallerTransactor) MakeDelegatecall(opts *bind.TransactOpts, _called common.Address, _calldata []byte) (*types.Transaction, error) {
	return _DelegateCaller.contract.Transact(opts, "makeDelegatecall", _called, _calldata)
}

// MakeDelegatecall is a paid mutator transaction binding the contract method 0xe632e172.
//
// Solidity: function makeDelegatecall(address _called, bytes _calldata) returns(bool success, bytes returnData)
func (_DelegateCaller *DelegateCallerSession) MakeDelegatecall(_called common.Address, _calldata []byte) (*types.Transaction, error) {
	return _DelegateCaller.Contract.MakeDelegatecall(&_DelegateCaller.TransactOpts, _called, _calldata)
}

// MakeDelegatecall is a paid mutator transaction binding the contract method 0xe632e172.
//
// Solidity: function makeDelegatecall(address _called, bytes _calldata) returns(bool success, bytes returnData)
func (_DelegateCaller *DelegateCallerTransactorSession) MakeDelegatecall(_called common.Address, _calldata []byte) (*types.Transaction, error) {
	return _DelegateCaller.Contract.MakeDelegatecall(&_DelegateCaller.TransactOpts, _called, _calldata)
}

// ERC1155TokenMetaData contains all meta data concerning the ERC1155Token contract.
var ERC1155TokenMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610ae3806100206000396000f3fe608060405234801561001057600080fd5b50600436106100405760003560e01c8062fdd58e14610045578063731133e9146100a7578063f242432a14610154575b600080fd5b6100916004803603604081101561005b57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610221565b6040518082815260200191505060405180910390f35b610152600480360360808110156100bd57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190803590602001909291908035906020019064010000000081111561010e57600080fd5b82018360208201111561012057600080fd5b8035906020019184600183028401116401000000008311171561014257600080fd5b9091929391929390505050610300565b005b61021f600480360360a081101561016a57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019092919080359060200190929190803590602001906401000000008111156101db57600080fd5b8201836020820111156101ed57600080fd5b8035906020019184600183028401116401000000008311171561020f57600080fd5b9091929391929390505050610485565b005b60008073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1614156102a8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252602b8152602001806109d1602b913960400191505060405180910390fd5b60008083815260200190815260200160002060008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b600073ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff161415610386576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526021815260200180610a5c6021913960400191505060405180910390fd5b60008085815260200190815260200160002060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054830160008086815260200190815260200160002060008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555061047e33600087878787878080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050506107cb565b5050505050565b600073ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff16141561050b576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260288152602001806109fc6028913960400191505060405180910390fd5b3373ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff1614806105d2575060011515600160008873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff161515145b610627576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526038815260200180610a246038913960400191505060405180910390fd5b8260008086815260200190815260200160002060008873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020540360008086815260200190815260200160002060008873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555060008085815260200190815260200160002060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054830160008086815260200190815260200160002060008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506107c3338787878787878080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050506107cb565b505050505050565b6107d4846109bd565b156109b55763f23a6e6160e01b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168473ffffffffffffffffffffffffffffffffffffffff1663f23a6e6188888787876040518663ffffffff1660e01b8152600401808673ffffffffffffffffffffffffffffffffffffffff1681526020018573ffffffffffffffffffffffffffffffffffffffff16815260200184815260200183815260200180602001828103825283818151815260200191508051906020019080838360005b838110156108b4578082015181840152602081019050610899565b50505050905090810190601f1680156108e15780820380516001836020036101000a031916815260200191505b509650505050505050602060405180830381600087803b15801561090457600080fd5b505af1158015610918573d6000803e3d6000fd5b505050506040513d602081101561092e57600080fd5b81019080805190602001909291905050507bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916146109b4576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526031815260200180610a7d6031913960400191505060405180910390fd5b5b505050505050565b600080823b90506000811191505091905056fe455243313135353a2062616c616e636520717565727920666f7220746865207a65726f2061646472657373455243313135353a207461726765742061646472657373206d757374206265206e6f6e2d7a65726f455243313135353a206e656564206f70657261746f7220617070726f76616c20666f7220337264207061727479207472616e73666572732e455243313135353a206d696e7420746f20746865207a65726f2061646472657373455243313135353a20676f7420756e6b6e6f776e2076616c75652066726f6d206f6e455243313135355265636569766564a264697066735822122049533ab32faee05169ec29223ecd1687274b175b3e256847f999f1971c1bc00d64736f6c63430007060033",
}

// ERC1155TokenABI is the input ABI used to generate the binding from.
// Deprecated: Use ERC1155TokenMetaData.ABI instead.
var ERC1155TokenABI = ERC1155TokenMetaData.ABI

// ERC1155TokenBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ERC1155TokenMetaData.Bin instead.
var ERC1155TokenBin = ERC1155TokenMetaData.Bin

// DeployERC1155Token deploys a new Ethereum contract, binding an instance of ERC1155Token to it.
func DeployERC1155Token(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ERC1155Token, error) {
	parsed, err := ERC1155TokenMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ERC1155TokenBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ERC1155Token{ERC1155TokenCaller: ERC1155TokenCaller{contract: contract}, ERC1155TokenTransactor: ERC1155TokenTransactor{contract: contract}, ERC1155TokenFilterer: ERC1155TokenFilterer{contract: contract}}, nil
}

// ERC1155Token is an auto generated Go binding around an Ethereum contract.
type ERC1155Token struct {
	ERC1155TokenCaller     // Read-only binding to the contract
	ERC1155TokenTransactor // Write-only binding to the contract
	ERC1155TokenFilterer   // Log filterer for contract events
}

// ERC1155TokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type ERC1155TokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC1155TokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC1155TokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC1155TokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC1155TokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC1155TokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC1155TokenSession struct {
	Contract     *ERC1155Token     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ERC1155TokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC1155TokenCallerSession struct {
	Contract *ERC1155TokenCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// ERC1155TokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC1155TokenTransactorSession struct {
	Contract     *ERC1155TokenTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// ERC1155TokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type ERC1155TokenRaw struct {
	Contract *ERC1155Token // Generic contract binding to access the raw methods on
}

// ERC1155TokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC1155TokenCallerRaw struct {
	Contract *ERC1155TokenCaller // Generic read-only contract binding to access the raw methods on
}

// ERC1155TokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC1155TokenTransactorRaw struct {
	Contract *ERC1155TokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewERC1155Token creates a new instance of ERC1155Token, bound to a specific deployed contract.
func NewERC1155Token(address common.Address, backend bind.ContractBackend) (*ERC1155Token, error) {
	contract, err := bindERC1155Token(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC1155Token{ERC1155TokenCaller: ERC1155TokenCaller{contract: contract}, ERC1155TokenTransactor: ERC1155TokenTransactor{contract: contract}, ERC1155TokenFilterer: ERC1155TokenFilterer{contract: contract}}, nil
}

// NewERC1155TokenCaller creates a new read-only instance of ERC1155Token, bound to a specific deployed contract.
func NewERC1155TokenCaller(address common.Address, caller bind.ContractCaller) (*ERC1155TokenCaller, error) {
	contract, err := bindERC1155Token(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC1155TokenCaller{contract: contract}, nil
}

// NewERC1155TokenTransactor creates a new write-only instance of ERC1155Token, bound to a specific deployed contract.
func NewERC1155TokenTransactor(address common.Address, transactor bind.ContractTransactor) (*ERC1155TokenTransactor, error) {
	contract, err := bindERC1155Token(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC1155TokenTransactor{contract: contract}, nil
}

// NewERC1155TokenFilterer creates a new log filterer instance of ERC1155Token, bound to a specific deployed contract.
func NewERC1155TokenFilterer(address common.Address, filterer bind.ContractFilterer) (*ERC1155TokenFilterer, error) {
	contract, err := bindERC1155Token(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC1155TokenFilterer{contract: contract}, nil
}

// bindERC1155Token binds a generic wrapper to an already deployed contract.
func bindERC1155Token(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ERC1155TokenMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC1155Token *ERC1155TokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC1155Token.Contract.ERC1155TokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC1155Token *ERC1155TokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC1155Token.Contract.ERC1155TokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC1155Token *ERC1155TokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC1155Token.Contract.ERC1155TokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC1155Token *ERC1155TokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC1155Token.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC1155Token *ERC1155TokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC1155Token.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC1155Token *ERC1155TokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC1155Token.Contract.contract.Transact(opts, method, params...)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address owner, uint256 id) view returns(uint256)
func (_ERC1155Token *ERC1155TokenCaller) BalanceOf(opts *bind.CallOpts, owner common.Address, id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _ERC1155Token.contract.Call(opts, &out, "balanceOf", owner, id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address owner, uint256 id) view returns(uint256)
func (_ERC1155Token *ERC1155TokenSession) BalanceOf(owner common.Address, id *big.Int) (*big.Int, error) {
	return _ERC1155Token.Contract.BalanceOf(&_ERC1155Token.CallOpts, owner, id)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address owner, uint256 id) view returns(uint256)
func (_ERC1155Token *ERC1155TokenCallerSession) BalanceOf(owner common.Address, id *big.Int) (*big.Int, error) {
	return _ERC1155Token.Contract.BalanceOf(&_ERC1155Token.CallOpts, owner, id)
}

// Mint is a paid mutator transaction binding the contract method 0x731133e9.
//
// Solidity: function mint(address to, uint256 id, uint256 value, bytes data) returns()
func (_ERC1155Token *ERC1155TokenTransactor) Mint(opts *bind.TransactOpts, to common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _ERC1155Token.contract.Transact(opts, "mint", to, id, value, data)
}

// Mint is a paid mutator transaction binding the contract method 0x731133e9.
//
// Solidity: function mint(address to, uint256 id, uint256 value, bytes data) returns()
func (_ERC1155Token *ERC1155TokenSession) Mint(to common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _ERC1155Token.Contract.Mint(&_ERC1155Token.TransactOpts, to, id, value, data)
}

// Mint is a paid mutator transaction binding the contract method 0x731133e9.
//
// Solidity: function mint(address to, uint256 id, uint256 value, bytes data) returns()
func (_ERC1155Token *ERC1155TokenTransactorSession) Mint(to common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _ERC1155Token.Contract.Mint(&_ERC1155Token.TransactOpts, to, id, value, data)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0xf242432a.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, uint256 value, bytes data) returns()
func (_ERC1155Token *ERC1155TokenTransactor) SafeTransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _ERC1155Token.contract.Transact(opts, "safeTransferFrom", from, to, id, value, data)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0xf242432a.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, uint256 value, bytes data) returns()
func (_ERC1155Token *ERC1155TokenSession) SafeTransferFrom(from common.Address, to common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _ERC1155Token.Contract.SafeTransferFrom(&_ERC1155Token.TransactOpts, from, to, id, value, data)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0xf242432a.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, uint256 value, bytes data) returns()
func (_ERC1155Token *ERC1155TokenTransactorSession) SafeTransferFrom(from common.Address, to common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _ERC1155Token.Contract.SafeTransferFrom(&_ERC1155Token.TransactOpts, from, to, id, value, data)
}

// ERC20TokenMetaData contains all meta data concerning the ERC20Token contract.
var ERC20TokenMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b506040518060400160405280600981526020017f54657374546f6b656e00000000000000000000000000000000000000000000008152506040518060400160405280600281526020017f545400000000000000000000000000000000000000000000000000000000000081525081600390805190602001906200009692919062000359565b508060049080519060200190620000af92919062000359565b506012600560006101000a81548160ff021916908360ff1602179055505050620000e73366038d7ea4c68000620000ed60201b60201c565b6200040f565b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16141562000191576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601f8152602001807f45524332303a206d696e7420746f20746865207a65726f20616464726573730081525060200191505060405180910390fd5b620001a560008383620002cb60201b60201c565b620001c181600254620002d060201b620009a01790919060201c565b6002819055506200021f816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054620002d060201b620009a01790919060201c565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a35050565b505050565b6000808284019050838110156200034f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601b8152602001807f536166654d6174683a206164646974696f6e206f766572666c6f77000000000081525060200191505060405180910390fd5b8091505092915050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282620003915760008555620003dd565b82601f10620003ac57805160ff1916838001178555620003dd565b82800160010185558215620003dd579182015b82811115620003dc578251825591602001919060010190620003bf565b5b509050620003ec9190620003f0565b5090565b5b808211156200040b576000816000905550600101620003f1565b5090565b6110de806200041f6000396000f3fe608060405234801561001057600080fd5b50600436106100a95760003560e01c80633950935111610071578063395093511461025857806370a08231146102bc57806395d89b4114610314578063a457c2d714610397578063a9059cbb146103fb578063dd62ed3e1461045f576100a9565b806306fdde03146100ae578063095ea7b31461013157806318160ddd1461019557806323b872dd146101b3578063313ce56714610237575b600080fd5b6100b66104d7565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156100f65780820151818401526020810190506100db565b50505050905090810190601f1680156101235780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b61017d6004803603604081101561014757600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610579565b60405180821515815260200191505060405180910390f35b61019d610597565b6040518082815260200191505060405180910390f35b61021f600480360360608110156101c957600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506105a1565b60405180821515815260200191505060405180910390f35b61023f61067a565b604051808260ff16815260200191505060405180910390f35b6102a46004803603604081101561026e57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610691565b60405180821515815260200191505060405180910390f35b6102fe600480360360208110156102d257600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610744565b6040518082815260200191505060405180910390f35b61031c61078c565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561035c578082015181840152602081019050610341565b50505050905090810190601f1680156103895780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b6103e3600480360360408110156103ad57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019092919050505061082e565b60405180821515815260200191505060405180910390f35b6104476004803603604081101561041157600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506108fb565b60405180821515815260200191505060405180910390f35b6104c16004803603604081101561047557600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610919565b6040518082815260200191505060405180910390f35b606060038054600181600116156101000203166002900480601f01602080910402602001604051908101604052809291908181526020018280546001816001161561010002031660029004801561056f5780601f106105445761010080835404028352916020019161056f565b820191906000526020600020905b81548152906001019060200180831161055257829003601f168201915b5050505050905090565b600061058d610586610a28565b8484610a30565b6001905092915050565b6000600254905090565b60006105ae848484610c27565b61066f846105ba610a28565b61066a8560405180606001604052806028815260200161101360289139600160008b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000610620610a28565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610ee89092919063ffffffff16565b610a30565b600190509392505050565b6000600560009054906101000a900460ff16905090565b600061073a61069e610a28565b8461073585600160006106af610a28565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008973ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546109a090919063ffffffff16565b610a30565b6001905092915050565b60008060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b606060048054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156108245780601f106107f957610100808354040283529160200191610824565b820191906000526020600020905b81548152906001019060200180831161080757829003601f168201915b5050505050905090565b60006108f161083b610a28565b846108ec856040518060600160405280602581526020016110846025913960016000610865610a28565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008a73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610ee89092919063ffffffff16565b610a30565b6001905092915050565b600061090f610908610a28565b8484610c27565b6001905092915050565b6000600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b600080828401905083811015610a1e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601b8152602001807f536166654d6174683a206164646974696f6e206f766572666c6f77000000000081525060200191505060405180910390fd5b8091505092915050565b600033905090565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415610ab6576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260248152602001806110606024913960400191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff161415610b3c576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526022815260200180610fcb6022913960400191505060405180910390fd5b80600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925836040518082815260200191505060405180910390a3505050565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415610cad576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252602581526020018061103b6025913960400191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff161415610d33576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526023815260200180610fa86023913960400191505060405180910390fd5b610d3e838383610fa2565b610da981604051806060016040528060268152602001610fed602691396000808773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610ee89092919063ffffffff16565b6000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550610e3c816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546109a090919063ffffffff16565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a3505050565b6000838311158290610f95576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825283818151815260200191508051906020019080838360005b83811015610f5a578082015181840152602081019050610f3f565b50505050905090810190601f168015610f875780820380516001836020036101000a031916815260200191505b509250505060405180910390fd5b5082840390509392505050565b50505056fe45524332303a207472616e7366657220746f20746865207a65726f206164647265737345524332303a20617070726f766520746f20746865207a65726f206164647265737345524332303a207472616e7366657220616d6f756e7420657863656564732062616c616e636545524332303a207472616e7366657220616d6f756e74206578636565647320616c6c6f77616e636545524332303a207472616e736665722066726f6d20746865207a65726f206164647265737345524332303a20617070726f76652066726f6d20746865207a65726f206164647265737345524332303a2064656372656173656420616c6c6f77616e63652062656c6f77207a65726fa2646970667358221220cf162c0eb75aec5620498383551a57a1bf95abf691ad5c8a2e2f10543ccbf45e64736f6c63430007060033",
}

// ERC20TokenABI is the input ABI used to generate the binding from.
// Deprecated: Use ERC20TokenMetaData.ABI instead.
var ERC20TokenABI = ERC20TokenMetaData.ABI

// ERC20TokenBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ERC20TokenMetaData.Bin instead.
var ERC20TokenBin = ERC20TokenMetaData.Bin

// DeployERC20Token deploys a new Ethereum contract, binding an instance of ERC20Token to it.
func DeployERC20Token(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ERC20Token, error) {
	parsed, err := ERC20TokenMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ERC20TokenBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ERC20Token{ERC20TokenCaller: ERC20TokenCaller{contract: contract}, ERC20TokenTransactor: ERC20TokenTransactor{contract: contract}, ERC20TokenFilterer: ERC20TokenFilterer{contract: contract}}, nil
}

// ERC20Token is an auto generated Go binding around an Ethereum contract.
type ERC20Token struct {
	ERC20TokenCaller     // Read-only binding to the contract
	ERC20TokenTransactor // Write-only binding to the contract
	ERC20TokenFilterer   // Log filterer for contract events
}

// ERC20TokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type ERC20TokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20TokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC20TokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20TokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC20TokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20TokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC20TokenSession struct {
	Contract     *ERC20Token       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ERC20TokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC20TokenCallerSession struct {
	Contract *ERC20TokenCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// ERC20TokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC20TokenTransactorSession struct {
	Contract     *ERC20TokenTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ERC20TokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type ERC20TokenRaw struct {
	Contract *ERC20Token // Generic contract binding to access the raw methods on
}

// ERC20TokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC20TokenCallerRaw struct {
	Contract *ERC20TokenCaller // Generic read-only contract binding to access the raw methods on
}

// ERC20TokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC20TokenTransactorRaw struct {
	Contract *ERC20TokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewERC20Token creates a new instance of ERC20Token, bound to a specific deployed contract.
func NewERC20Token(address common.Address, backend bind.ContractBackend) (*ERC20Token, error) {
	contract, err := bindERC20Token(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC20Token{ERC20TokenCaller: ERC20TokenCaller{contract: contract}, ERC20TokenTransactor: ERC20TokenTransactor{contract: contract}, ERC20TokenFilterer: ERC20TokenFilterer{contract: contract}}, nil
}

// NewERC20TokenCaller creates a new read-only instance of ERC20Token, bound to a specific deployed contract.
func NewERC20TokenCaller(address common.Address, caller bind.ContractCaller) (*ERC20TokenCaller, error) {
	contract, err := bindERC20Token(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC20TokenCaller{contract: contract}, nil
}

// NewERC20TokenTransactor creates a new write-only instance of ERC20Token, bound to a specific deployed contract.
func NewERC20TokenTransactor(address common.Address, transactor bind.ContractTransactor) (*ERC20TokenTransactor, error) {
	contract, err := bindERC20Token(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC20TokenTransactor{contract: contract}, nil
}

// NewERC20TokenFilterer creates a new log filterer instance of ERC20Token, bound to a specific deployed contract.
func NewERC20TokenFilterer(address common.Address, filterer bind.ContractFilterer) (*ERC20TokenFilterer, error) {
	contract, err := bindERC20Token(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC20TokenFilterer{contract: contract}, nil
}

// bindERC20Token binds a generic wrapper to an already deployed contract.
func bindERC20Token(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ERC20TokenMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC20Token *ERC20TokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC20Token.Contract.ERC20TokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC20Token *ERC20TokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC20Token.Contract.ERC20TokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC20Token *ERC20TokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC20Token.Contract.ERC20TokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC20Token *ERC20TokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC20Token.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC20Token *ERC20TokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC20Token.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC20Token *ERC20TokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC20Token.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_ERC20Token *ERC20TokenCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Token.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_ERC20Token *ERC20TokenSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _ERC20Token.Contract.Allowance(&_ERC20Token.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_ERC20Token *ERC20TokenCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _ERC20Token.Contract.Allowance(&_ERC20Token.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ERC20Token *ERC20TokenCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Token.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ERC20Token *ERC20TokenSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _ERC20Token.Contract.BalanceOf(&_ERC20Token.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ERC20Token *ERC20TokenCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _ERC20Token.Contract.BalanceOf(&_ERC20Token.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ERC20Token *ERC20TokenCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _ERC20Token.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ERC20Token *ERC20TokenSession) Decimals() (uint8, error) {
	return _ERC20Token.Contract.Decimals(&_ERC20Token.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ERC20Token *ERC20TokenCallerSession) Decimals() (uint8, error) {
	return _ERC20Token.Contract.Decimals(&_ERC20Token.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_ERC20Token *ERC20TokenCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ERC20Token.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_ERC20Token *ERC20TokenSession) Name() (string, error) {
	return _ERC20Token.Contract.Name(&_ERC20Token.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_ERC20Token *ERC20TokenCallerSession) Name() (string, error) {
	return _ERC20Token.Contract.Name(&_ERC20Token.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ERC20Token *ERC20TokenCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ERC20Token.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ERC20Token *ERC20TokenSession) Symbol() (string, error) {
	return _ERC20Token.Contract.Symbol(&_ERC20Token.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ERC20Token *ERC20TokenCallerSession) Symbol() (string, error) {
	return _ERC20Token.Contract.Symbol(&_ERC20Token.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_ERC20Token *ERC20TokenCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Token.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_ERC20Token *ERC20TokenSession) TotalSupply() (*big.Int, error) {
	return _ERC20Token.Contract.TotalSupply(&_ERC20Token.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_ERC20Token *ERC20TokenCallerSession) TotalSupply() (*big.Int, error) {
	return _ERC20Token.Contract.TotalSupply(&_ERC20Token.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_ERC20Token *ERC20TokenTransactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Token.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_ERC20Token *ERC20TokenSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.Approve(&_ERC20Token.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_ERC20Token *ERC20TokenTransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.Approve(&_ERC20Token.TransactOpts, spender, amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_ERC20Token *ERC20TokenTransactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Token.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_ERC20Token *ERC20TokenSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.DecreaseAllowance(&_ERC20Token.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_ERC20Token *ERC20TokenTransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.DecreaseAllowance(&_ERC20Token.TransactOpts, spender, subtractedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_ERC20Token *ERC20TokenTransactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Token.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_ERC20Token *ERC20TokenSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.IncreaseAllowance(&_ERC20Token.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_ERC20Token *ERC20TokenTransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.IncreaseAllowance(&_ERC20Token.TransactOpts, spender, addedValue)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_ERC20Token *ERC20TokenTransactor) Transfer(opts *bind.TransactOpts, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Token.contract.Transact(opts, "transfer", recipient, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_ERC20Token *ERC20TokenSession) Transfer(recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.Transfer(&_ERC20Token.TransactOpts, recipient, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_ERC20Token *ERC20TokenTransactorSession) Transfer(recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.Transfer(&_ERC20Token.TransactOpts, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_ERC20Token *ERC20TokenTransactor) TransferFrom(opts *bind.TransactOpts, sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Token.contract.Transact(opts, "transferFrom", sender, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_ERC20Token *ERC20TokenSession) TransferFrom(sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.TransferFrom(&_ERC20Token.TransactOpts, sender, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_ERC20Token *ERC20TokenTransactorSession) TransferFrom(sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Token.Contract.TransferFrom(&_ERC20Token.TransactOpts, sender, recipient, amount)
}

// ERC20TokenApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the ERC20Token contract.
type ERC20TokenApprovalIterator struct {
	Event *ERC20TokenApproval // Event containing the contract specifics and raw log

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
func (it *ERC20TokenApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC20TokenApproval)
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
		it.Event = new(ERC20TokenApproval)
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
func (it *ERC20TokenApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC20TokenApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC20TokenApproval represents a Approval event raised by the ERC20Token contract.
type ERC20TokenApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_ERC20Token *ERC20TokenFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*ERC20TokenApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _ERC20Token.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &ERC20TokenApprovalIterator{contract: _ERC20Token.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_ERC20Token *ERC20TokenFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *ERC20TokenApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _ERC20Token.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC20TokenApproval)
				if err := _ERC20Token.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_ERC20Token *ERC20TokenFilterer) ParseApproval(log types.Log) (*ERC20TokenApproval, error) {
	event := new(ERC20TokenApproval)
	if err := _ERC20Token.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ERC20TokenTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the ERC20Token contract.
type ERC20TokenTransferIterator struct {
	Event *ERC20TokenTransfer // Event containing the contract specifics and raw log

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
func (it *ERC20TokenTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC20TokenTransfer)
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
		it.Event = new(ERC20TokenTransfer)
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
func (it *ERC20TokenTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC20TokenTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC20TokenTransfer represents a Transfer event raised by the ERC20Token contract.
type ERC20TokenTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Token *ERC20TokenFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*ERC20TokenTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _ERC20Token.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &ERC20TokenTransferIterator{contract: _ERC20Token.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Token *ERC20TokenFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *ERC20TokenTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _ERC20Token.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC20TokenTransfer)
				if err := _ERC20Token.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Token *ERC20TokenFilterer) ParseTransfer(log types.Log) (*ERC20TokenTransfer, error) {
	event := new(ERC20TokenTransfer)
	if err := _ERC20Token.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISafeMetaData contains all meta data concerning the ISafe contract.
var ISafeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"module\",\"type\":\"address\"}],\"name\":\"enableModule\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint8\",\"name\":\"operation\",\"type\":\"uint8\"}],\"name\":\"execTransactionFromModule\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
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

// Test4337ModuleAndHandlerMetaData contains all meta data concerning the Test4337ModuleAndHandler contract.
var Test4337ModuleAndHandlerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"entryPointAddress\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"ENTRYPOINT\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MY_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"enableMyself\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"execTransaction\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"initCode\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"callGasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"verificationGasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"preVerificationGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxPriorityFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"paymasterAndData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structUserOperation\",\"name\":\"userOp\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"missingAccountFunds\",\"type\":\"uint256\"}],\"name\":\"validateUserOp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"validationData\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60c060405234801561001057600080fd5b506040516109d93803806109d9833981810160405281019061003291906100bb565b8073ffffffffffffffffffffffffffffffffffffffff1660a08173ffffffffffffffffffffffffffffffffffffffff1660601b815250503073ffffffffffffffffffffffffffffffffffffffff1660808173ffffffffffffffffffffffffffffffffffffffff1660601b815250505061012d565b6000815190506100b581610116565b92915050565b6000602082840312156100cd57600080fd5b60006100db848285016100a6565b91505092915050565b60006100ef826100f6565b9050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b61011f816100e4565b811461012a57600080fd5b50565b60805160601c60a05160601c6108796101606000398061017c5280610391525080610117528061023b52506108796000f3fe60806040526004361061004a5760003560e01c80633a756cec1461004f5780633a871cdd1461007a578063a798b2b1146100b7578063ab4ed83e146100ce578063e8eb3cc6146100ea575b600080fd5b34801561005b57600080fd5b50610064610115565b6040516100719190610646565b60405180910390f35b34801561008657600080fd5b506100a1600480360381019061009c919061052b565b610139565b6040516100ae9190610719565b60405180910390f35b3480156100c357600080fd5b506100cc61021d565b005b6100e860048036038101906100e39190610496565b6102aa565b005b3480156100f657600080fd5b506100ff61038f565b60405161010c9190610646565b60405180910390f35b7f000000000000000000000000000000000000000000000000000000000000000081565b60008084600001602081019061014f919061046d565b9050600081905060008414610210578073ffffffffffffffffffffffffffffffffffffffff1663468721a77f00000000000000000000000000000000000000000000000000000000000000008660006040518463ffffffff1660e01b81526004016101bc939291906106af565b602060405180830381600087803b1580156101d657600080fd5b505af11580156101ea573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061020e9190610502565b505b6000925050509392505050565b3073ffffffffffffffffffffffffffffffffffffffff1663610b59257f00000000000000000000000000000000000000000000000000000000000000006040518263ffffffff1660e01b81526004016102769190610646565b600060405180830381600087803b15801561029057600080fd5b505af11580156102a4573d6000803e3d6000fd5b50505050565b600033905060008190508073ffffffffffffffffffffffffffffffffffffffff1663468721a78787878760006040518663ffffffff1660e01b81526004016102f6959493929190610661565b602060405180830381600087803b15801561031057600080fd5b505af1158015610324573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906103489190610502565b610387576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161037e906106f9565b60405180910390fd5b505050505050565b7f000000000000000000000000000000000000000000000000000000000000000081565b6000813590506103c2816107e7565b92915050565b6000815190506103d7816107fe565b92915050565b6000813590506103ec81610815565b92915050565b60008083601f84011261040457600080fd5b8235905067ffffffffffffffff81111561041d57600080fd5b60208301915083600182028301111561043557600080fd5b9250929050565b6000610160828403121561044f57600080fd5b81905092915050565b6000813590506104678161082c565b92915050565b60006020828403121561047f57600080fd5b600061048d848285016103b3565b91505092915050565b600080600080606085870312156104ac57600080fd5b60006104ba878288016103b3565b94505060206104cb87828801610458565b935050604085013567ffffffffffffffff8111156104e857600080fd5b6104f4878288016103f2565b925092505092959194509250565b60006020828403121561051457600080fd5b6000610522848285016103c8565b91505092915050565b60008060006060848603121561054057600080fd5b600084013567ffffffffffffffff81111561055a57600080fd5b6105668682870161043c565b9350506020610577868287016103dd565b925050604061058886828701610458565b9150509250925092565b61059b81610756565b82525050565b60006105ad8385610734565b93506105ba8385846107c7565b6105c3836107d6565b840190509392505050565b6105d7816107b5565b82525050565b60006105ea600983610745565b91507f7478206661696c656400000000000000000000000000000000000000000000006000830152602082019050919050565b600061062a600083610734565b9150600082019050919050565b6106408161079e565b82525050565b600060208201905061065b6000830184610592565b92915050565b60006080820190506106766000830188610592565b6106836020830187610637565b81810360408301526106968185876105a1565b90506106a560608301846105ce565b9695505050505050565b60006080820190506106c46000830186610592565b6106d16020830185610637565b81810360408301526106e28161061d565b90506106f160608301846105ce565b949350505050565b60006020820190508181036000830152610712816105dd565b9050919050565b600060208201905061072e6000830184610637565b92915050565b600082825260208201905092915050565b600082825260208201905092915050565b60006107618261077e565b9050919050565b60008115159050919050565b6000819050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b600060ff82169050919050565b60006107c0826107a8565b9050919050565b82818337600083830152505050565b6000601f19601f8301169050919050565b6107f081610756565b81146107fb57600080fd5b50565b61080781610768565b811461081257600080fd5b50565b61081e81610774565b811461082957600080fd5b50565b6108358161079e565b811461084057600080fd5b5056fea264697066735822122077ce9f3a41b2586f42b9efe16a2fcd6ff83c8d4c03453502cc93e2e97668704c64736f6c63430007060033",
}

// Test4337ModuleAndHandlerABI is the input ABI used to generate the binding from.
// Deprecated: Use Test4337ModuleAndHandlerMetaData.ABI instead.
var Test4337ModuleAndHandlerABI = Test4337ModuleAndHandlerMetaData.ABI

// Test4337ModuleAndHandlerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use Test4337ModuleAndHandlerMetaData.Bin instead.
var Test4337ModuleAndHandlerBin = Test4337ModuleAndHandlerMetaData.Bin

// DeployTest4337ModuleAndHandler deploys a new Ethereum contract, binding an instance of Test4337ModuleAndHandler to it.
func DeployTest4337ModuleAndHandler(auth *bind.TransactOpts, backend bind.ContractBackend, entryPointAddress common.Address) (common.Address, *types.Transaction, *Test4337ModuleAndHandler, error) {
	parsed, err := Test4337ModuleAndHandlerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(Test4337ModuleAndHandlerBin), backend, entryPointAddress)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Test4337ModuleAndHandler{Test4337ModuleAndHandlerCaller: Test4337ModuleAndHandlerCaller{contract: contract}, Test4337ModuleAndHandlerTransactor: Test4337ModuleAndHandlerTransactor{contract: contract}, Test4337ModuleAndHandlerFilterer: Test4337ModuleAndHandlerFilterer{contract: contract}}, nil
}

// Test4337ModuleAndHandler is an auto generated Go binding around an Ethereum contract.
type Test4337ModuleAndHandler struct {
	Test4337ModuleAndHandlerCaller     // Read-only binding to the contract
	Test4337ModuleAndHandlerTransactor // Write-only binding to the contract
	Test4337ModuleAndHandlerFilterer   // Log filterer for contract events
}

// Test4337ModuleAndHandlerCaller is an auto generated read-only Go binding around an Ethereum contract.
type Test4337ModuleAndHandlerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Test4337ModuleAndHandlerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type Test4337ModuleAndHandlerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Test4337ModuleAndHandlerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Test4337ModuleAndHandlerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Test4337ModuleAndHandlerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Test4337ModuleAndHandlerSession struct {
	Contract     *Test4337ModuleAndHandler // Generic contract binding to set the session for
	CallOpts     bind.CallOpts             // Call options to use throughout this session
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// Test4337ModuleAndHandlerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Test4337ModuleAndHandlerCallerSession struct {
	Contract *Test4337ModuleAndHandlerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                   // Call options to use throughout this session
}

// Test4337ModuleAndHandlerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Test4337ModuleAndHandlerTransactorSession struct {
	Contract     *Test4337ModuleAndHandlerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                   // Transaction auth options to use throughout this session
}

// Test4337ModuleAndHandlerRaw is an auto generated low-level Go binding around an Ethereum contract.
type Test4337ModuleAndHandlerRaw struct {
	Contract *Test4337ModuleAndHandler // Generic contract binding to access the raw methods on
}

// Test4337ModuleAndHandlerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Test4337ModuleAndHandlerCallerRaw struct {
	Contract *Test4337ModuleAndHandlerCaller // Generic read-only contract binding to access the raw methods on
}

// Test4337ModuleAndHandlerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Test4337ModuleAndHandlerTransactorRaw struct {
	Contract *Test4337ModuleAndHandlerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTest4337ModuleAndHandler creates a new instance of Test4337ModuleAndHandler, bound to a specific deployed contract.
func NewTest4337ModuleAndHandler(address common.Address, backend bind.ContractBackend) (*Test4337ModuleAndHandler, error) {
	contract, err := bindTest4337ModuleAndHandler(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Test4337ModuleAndHandler{Test4337ModuleAndHandlerCaller: Test4337ModuleAndHandlerCaller{contract: contract}, Test4337ModuleAndHandlerTransactor: Test4337ModuleAndHandlerTransactor{contract: contract}, Test4337ModuleAndHandlerFilterer: Test4337ModuleAndHandlerFilterer{contract: contract}}, nil
}

// NewTest4337ModuleAndHandlerCaller creates a new read-only instance of Test4337ModuleAndHandler, bound to a specific deployed contract.
func NewTest4337ModuleAndHandlerCaller(address common.Address, caller bind.ContractCaller) (*Test4337ModuleAndHandlerCaller, error) {
	contract, err := bindTest4337ModuleAndHandler(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Test4337ModuleAndHandlerCaller{contract: contract}, nil
}

// NewTest4337ModuleAndHandlerTransactor creates a new write-only instance of Test4337ModuleAndHandler, bound to a specific deployed contract.
func NewTest4337ModuleAndHandlerTransactor(address common.Address, transactor bind.ContractTransactor) (*Test4337ModuleAndHandlerTransactor, error) {
	contract, err := bindTest4337ModuleAndHandler(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Test4337ModuleAndHandlerTransactor{contract: contract}, nil
}

// NewTest4337ModuleAndHandlerFilterer creates a new log filterer instance of Test4337ModuleAndHandler, bound to a specific deployed contract.
func NewTest4337ModuleAndHandlerFilterer(address common.Address, filterer bind.ContractFilterer) (*Test4337ModuleAndHandlerFilterer, error) {
	contract, err := bindTest4337ModuleAndHandler(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Test4337ModuleAndHandlerFilterer{contract: contract}, nil
}

// bindTest4337ModuleAndHandler binds a generic wrapper to an already deployed contract.
func bindTest4337ModuleAndHandler(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Test4337ModuleAndHandlerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Test4337ModuleAndHandler.Contract.Test4337ModuleAndHandlerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.Contract.Test4337ModuleAndHandlerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.Contract.Test4337ModuleAndHandlerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Test4337ModuleAndHandler.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.Contract.contract.Transact(opts, method, params...)
}

// ENTRYPOINT is a free data retrieval call binding the contract method 0xe8eb3cc6.
//
// Solidity: function ENTRYPOINT() view returns(address)
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerCaller) ENTRYPOINT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Test4337ModuleAndHandler.contract.Call(opts, &out, "ENTRYPOINT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ENTRYPOINT is a free data retrieval call binding the contract method 0xe8eb3cc6.
//
// Solidity: function ENTRYPOINT() view returns(address)
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerSession) ENTRYPOINT() (common.Address, error) {
	return _Test4337ModuleAndHandler.Contract.ENTRYPOINT(&_Test4337ModuleAndHandler.CallOpts)
}

// ENTRYPOINT is a free data retrieval call binding the contract method 0xe8eb3cc6.
//
// Solidity: function ENTRYPOINT() view returns(address)
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerCallerSession) ENTRYPOINT() (common.Address, error) {
	return _Test4337ModuleAndHandler.Contract.ENTRYPOINT(&_Test4337ModuleAndHandler.CallOpts)
}

// MYADDRESS is a free data retrieval call binding the contract method 0x3a756cec.
//
// Solidity: function MY_ADDRESS() view returns(address)
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerCaller) MYADDRESS(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Test4337ModuleAndHandler.contract.Call(opts, &out, "MY_ADDRESS")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MYADDRESS is a free data retrieval call binding the contract method 0x3a756cec.
//
// Solidity: function MY_ADDRESS() view returns(address)
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerSession) MYADDRESS() (common.Address, error) {
	return _Test4337ModuleAndHandler.Contract.MYADDRESS(&_Test4337ModuleAndHandler.CallOpts)
}

// MYADDRESS is a free data retrieval call binding the contract method 0x3a756cec.
//
// Solidity: function MY_ADDRESS() view returns(address)
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerCallerSession) MYADDRESS() (common.Address, error) {
	return _Test4337ModuleAndHandler.Contract.MYADDRESS(&_Test4337ModuleAndHandler.CallOpts)
}

// EnableMyself is a paid mutator transaction binding the contract method 0xa798b2b1.
//
// Solidity: function enableMyself() returns()
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerTransactor) EnableMyself(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.contract.Transact(opts, "enableMyself")
}

// EnableMyself is a paid mutator transaction binding the contract method 0xa798b2b1.
//
// Solidity: function enableMyself() returns()
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerSession) EnableMyself() (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.Contract.EnableMyself(&_Test4337ModuleAndHandler.TransactOpts)
}

// EnableMyself is a paid mutator transaction binding the contract method 0xa798b2b1.
//
// Solidity: function enableMyself() returns()
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerTransactorSession) EnableMyself() (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.Contract.EnableMyself(&_Test4337ModuleAndHandler.TransactOpts)
}

// ExecTransaction is a paid mutator transaction binding the contract method 0xab4ed83e.
//
// Solidity: function execTransaction(address to, uint256 value, bytes data) payable returns()
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerTransactor) ExecTransaction(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.contract.Transact(opts, "execTransaction", to, value, data)
}

// ExecTransaction is a paid mutator transaction binding the contract method 0xab4ed83e.
//
// Solidity: function execTransaction(address to, uint256 value, bytes data) payable returns()
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerSession) ExecTransaction(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.Contract.ExecTransaction(&_Test4337ModuleAndHandler.TransactOpts, to, value, data)
}

// ExecTransaction is a paid mutator transaction binding the contract method 0xab4ed83e.
//
// Solidity: function execTransaction(address to, uint256 value, bytes data) payable returns()
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerTransactorSession) ExecTransaction(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.Contract.ExecTransaction(&_Test4337ModuleAndHandler.TransactOpts, to, value, data)
}

// ValidateUserOp is a paid mutator transaction binding the contract method 0x3a871cdd.
//
// Solidity: function validateUserOp((address,uint256,bytes,bytes,uint256,uint256,uint256,uint256,uint256,bytes,bytes) userOp, bytes32 , uint256 missingAccountFunds) returns(uint256 validationData)
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerTransactor) ValidateUserOp(opts *bind.TransactOpts, userOp UserOperation, arg1 [32]byte, missingAccountFunds *big.Int) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.contract.Transact(opts, "validateUserOp", userOp, arg1, missingAccountFunds)
}

// ValidateUserOp is a paid mutator transaction binding the contract method 0x3a871cdd.
//
// Solidity: function validateUserOp((address,uint256,bytes,bytes,uint256,uint256,uint256,uint256,uint256,bytes,bytes) userOp, bytes32 , uint256 missingAccountFunds) returns(uint256 validationData)
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerSession) ValidateUserOp(userOp UserOperation, arg1 [32]byte, missingAccountFunds *big.Int) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.Contract.ValidateUserOp(&_Test4337ModuleAndHandler.TransactOpts, userOp, arg1, missingAccountFunds)
}

// ValidateUserOp is a paid mutator transaction binding the contract method 0x3a871cdd.
//
// Solidity: function validateUserOp((address,uint256,bytes,bytes,uint256,uint256,uint256,uint256,uint256,bytes,bytes) userOp, bytes32 , uint256 missingAccountFunds) returns(uint256 validationData)
func (_Test4337ModuleAndHandler *Test4337ModuleAndHandlerTransactorSession) ValidateUserOp(userOp UserOperation, arg1 [32]byte, missingAccountFunds *big.Int) (*types.Transaction, error) {
	return _Test4337ModuleAndHandler.Contract.ValidateUserOp(&_Test4337ModuleAndHandler.TransactOpts, userOp, arg1, missingAccountFunds)
}

// TestHandlerMetaData contains all meta data concerning the TestHandler contract.
var TestHandlerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"dudududu\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5060e08061001f6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c806354955e5914602d575b600080fd5b6033607c565b604051808373ffffffffffffffffffffffffffffffffffffffff1681526020018273ffffffffffffffffffffffffffffffffffffffff1681526020019250505060405180910390f35b60008060856093565b608b60a2565b915091509091565b6000601436033560601c905090565b60003390509056fea26469706673582212203bb05fdff8e545f51a34df027dbc60c2153b635de1cfa5db672db08e62d4823364736f6c63430007060033",
}

// TestHandlerABI is the input ABI used to generate the binding from.
// Deprecated: Use TestHandlerMetaData.ABI instead.
var TestHandlerABI = TestHandlerMetaData.ABI

// TestHandlerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TestHandlerMetaData.Bin instead.
var TestHandlerBin = TestHandlerMetaData.Bin

// DeployTestHandler deploys a new Ethereum contract, binding an instance of TestHandler to it.
func DeployTestHandler(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *TestHandler, error) {
	parsed, err := TestHandlerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TestHandlerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TestHandler{TestHandlerCaller: TestHandlerCaller{contract: contract}, TestHandlerTransactor: TestHandlerTransactor{contract: contract}, TestHandlerFilterer: TestHandlerFilterer{contract: contract}}, nil
}

// TestHandler is an auto generated Go binding around an Ethereum contract.
type TestHandler struct {
	TestHandlerCaller     // Read-only binding to the contract
	TestHandlerTransactor // Write-only binding to the contract
	TestHandlerFilterer   // Log filterer for contract events
}

// TestHandlerCaller is an auto generated read-only Go binding around an Ethereum contract.
type TestHandlerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestHandlerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TestHandlerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestHandlerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TestHandlerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestHandlerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TestHandlerSession struct {
	Contract     *TestHandler      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TestHandlerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TestHandlerCallerSession struct {
	Contract *TestHandlerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// TestHandlerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TestHandlerTransactorSession struct {
	Contract     *TestHandlerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// TestHandlerRaw is an auto generated low-level Go binding around an Ethereum contract.
type TestHandlerRaw struct {
	Contract *TestHandler // Generic contract binding to access the raw methods on
}

// TestHandlerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TestHandlerCallerRaw struct {
	Contract *TestHandlerCaller // Generic read-only contract binding to access the raw methods on
}

// TestHandlerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TestHandlerTransactorRaw struct {
	Contract *TestHandlerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTestHandler creates a new instance of TestHandler, bound to a specific deployed contract.
func NewTestHandler(address common.Address, backend bind.ContractBackend) (*TestHandler, error) {
	contract, err := bindTestHandler(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TestHandler{TestHandlerCaller: TestHandlerCaller{contract: contract}, TestHandlerTransactor: TestHandlerTransactor{contract: contract}, TestHandlerFilterer: TestHandlerFilterer{contract: contract}}, nil
}

// NewTestHandlerCaller creates a new read-only instance of TestHandler, bound to a specific deployed contract.
func NewTestHandlerCaller(address common.Address, caller bind.ContractCaller) (*TestHandlerCaller, error) {
	contract, err := bindTestHandler(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TestHandlerCaller{contract: contract}, nil
}

// NewTestHandlerTransactor creates a new write-only instance of TestHandler, bound to a specific deployed contract.
func NewTestHandlerTransactor(address common.Address, transactor bind.ContractTransactor) (*TestHandlerTransactor, error) {
	contract, err := bindTestHandler(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TestHandlerTransactor{contract: contract}, nil
}

// NewTestHandlerFilterer creates a new log filterer instance of TestHandler, bound to a specific deployed contract.
func NewTestHandlerFilterer(address common.Address, filterer bind.ContractFilterer) (*TestHandlerFilterer, error) {
	contract, err := bindTestHandler(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TestHandlerFilterer{contract: contract}, nil
}

// bindTestHandler binds a generic wrapper to an already deployed contract.
func bindTestHandler(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TestHandlerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TestHandler *TestHandlerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TestHandler.Contract.TestHandlerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TestHandler *TestHandlerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TestHandler.Contract.TestHandlerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TestHandler *TestHandlerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TestHandler.Contract.TestHandlerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TestHandler *TestHandlerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TestHandler.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TestHandler *TestHandlerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TestHandler.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TestHandler *TestHandlerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TestHandler.Contract.contract.Transact(opts, method, params...)
}

// Dudududu is a free data retrieval call binding the contract method 0x54955e59.
//
// Solidity: function dudududu() view returns(address sender, address manager)
func (_TestHandler *TestHandlerCaller) Dudududu(opts *bind.CallOpts) (struct {
	Sender  common.Address
	Manager common.Address
}, error) {
	var out []interface{}
	err := _TestHandler.contract.Call(opts, &out, "dudududu")

	outstruct := new(struct {
		Sender  common.Address
		Manager common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Sender = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Manager = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// Dudududu is a free data retrieval call binding the contract method 0x54955e59.
//
// Solidity: function dudududu() view returns(address sender, address manager)
func (_TestHandler *TestHandlerSession) Dudududu() (struct {
	Sender  common.Address
	Manager common.Address
}, error) {
	return _TestHandler.Contract.Dudududu(&_TestHandler.CallOpts)
}

// Dudududu is a free data retrieval call binding the contract method 0x54955e59.
//
// Solidity: function dudududu() view returns(address sender, address manager)
func (_TestHandler *TestHandlerCallerSession) Dudududu() (struct {
	Sender  common.Address
	Manager common.Address
}, error) {
	return _TestHandler.Contract.Dudududu(&_TestHandler.CallOpts)
}

// TestNativeTokenReceiverMetaData contains all meta data concerning the TestNativeTokenReceiver contract.
var TestNativeTokenReceiverMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"forwardedGas\",\"type\":\"uint256\"}],\"name\":\"BreadReceived\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"}]",
	Bin: "0x6080604052348015600f57600080fd5b50609280601d6000396000f3fe60806040523373ffffffffffffffffffffffffffffffffffffffff167f16549311ba52796916987df5401f791fb06b998524a5a8684010010415850bb3345a604051808381526020018281526020019250505060405180910390a200fea264697066735822122035663a4184b682e3d2c1649228db3273b6a2439d885e4203ca9ef996501e7b4c64736f6c63430007060033",
}

// TestNativeTokenReceiverABI is the input ABI used to generate the binding from.
// Deprecated: Use TestNativeTokenReceiverMetaData.ABI instead.
var TestNativeTokenReceiverABI = TestNativeTokenReceiverMetaData.ABI

// TestNativeTokenReceiverBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TestNativeTokenReceiverMetaData.Bin instead.
var TestNativeTokenReceiverBin = TestNativeTokenReceiverMetaData.Bin

// DeployTestNativeTokenReceiver deploys a new Ethereum contract, binding an instance of TestNativeTokenReceiver to it.
func DeployTestNativeTokenReceiver(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *TestNativeTokenReceiver, error) {
	parsed, err := TestNativeTokenReceiverMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TestNativeTokenReceiverBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TestNativeTokenReceiver{TestNativeTokenReceiverCaller: TestNativeTokenReceiverCaller{contract: contract}, TestNativeTokenReceiverTransactor: TestNativeTokenReceiverTransactor{contract: contract}, TestNativeTokenReceiverFilterer: TestNativeTokenReceiverFilterer{contract: contract}}, nil
}

// TestNativeTokenReceiver is an auto generated Go binding around an Ethereum contract.
type TestNativeTokenReceiver struct {
	TestNativeTokenReceiverCaller     // Read-only binding to the contract
	TestNativeTokenReceiverTransactor // Write-only binding to the contract
	TestNativeTokenReceiverFilterer   // Log filterer for contract events
}

// TestNativeTokenReceiverCaller is an auto generated read-only Go binding around an Ethereum contract.
type TestNativeTokenReceiverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestNativeTokenReceiverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TestNativeTokenReceiverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestNativeTokenReceiverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TestNativeTokenReceiverFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestNativeTokenReceiverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TestNativeTokenReceiverSession struct {
	Contract     *TestNativeTokenReceiver // Generic contract binding to set the session for
	CallOpts     bind.CallOpts            // Call options to use throughout this session
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// TestNativeTokenReceiverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TestNativeTokenReceiverCallerSession struct {
	Contract *TestNativeTokenReceiverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                  // Call options to use throughout this session
}

// TestNativeTokenReceiverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TestNativeTokenReceiverTransactorSession struct {
	Contract     *TestNativeTokenReceiverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                  // Transaction auth options to use throughout this session
}

// TestNativeTokenReceiverRaw is an auto generated low-level Go binding around an Ethereum contract.
type TestNativeTokenReceiverRaw struct {
	Contract *TestNativeTokenReceiver // Generic contract binding to access the raw methods on
}

// TestNativeTokenReceiverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TestNativeTokenReceiverCallerRaw struct {
	Contract *TestNativeTokenReceiverCaller // Generic read-only contract binding to access the raw methods on
}

// TestNativeTokenReceiverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TestNativeTokenReceiverTransactorRaw struct {
	Contract *TestNativeTokenReceiverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTestNativeTokenReceiver creates a new instance of TestNativeTokenReceiver, bound to a specific deployed contract.
func NewTestNativeTokenReceiver(address common.Address, backend bind.ContractBackend) (*TestNativeTokenReceiver, error) {
	contract, err := bindTestNativeTokenReceiver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TestNativeTokenReceiver{TestNativeTokenReceiverCaller: TestNativeTokenReceiverCaller{contract: contract}, TestNativeTokenReceiverTransactor: TestNativeTokenReceiverTransactor{contract: contract}, TestNativeTokenReceiverFilterer: TestNativeTokenReceiverFilterer{contract: contract}}, nil
}

// NewTestNativeTokenReceiverCaller creates a new read-only instance of TestNativeTokenReceiver, bound to a specific deployed contract.
func NewTestNativeTokenReceiverCaller(address common.Address, caller bind.ContractCaller) (*TestNativeTokenReceiverCaller, error) {
	contract, err := bindTestNativeTokenReceiver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TestNativeTokenReceiverCaller{contract: contract}, nil
}

// NewTestNativeTokenReceiverTransactor creates a new write-only instance of TestNativeTokenReceiver, bound to a specific deployed contract.
func NewTestNativeTokenReceiverTransactor(address common.Address, transactor bind.ContractTransactor) (*TestNativeTokenReceiverTransactor, error) {
	contract, err := bindTestNativeTokenReceiver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TestNativeTokenReceiverTransactor{contract: contract}, nil
}

// NewTestNativeTokenReceiverFilterer creates a new log filterer instance of TestNativeTokenReceiver, bound to a specific deployed contract.
func NewTestNativeTokenReceiverFilterer(address common.Address, filterer bind.ContractFilterer) (*TestNativeTokenReceiverFilterer, error) {
	contract, err := bindTestNativeTokenReceiver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TestNativeTokenReceiverFilterer{contract: contract}, nil
}

// bindTestNativeTokenReceiver binds a generic wrapper to an already deployed contract.
func bindTestNativeTokenReceiver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TestNativeTokenReceiverMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TestNativeTokenReceiver *TestNativeTokenReceiverRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TestNativeTokenReceiver.Contract.TestNativeTokenReceiverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TestNativeTokenReceiver *TestNativeTokenReceiverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TestNativeTokenReceiver.Contract.TestNativeTokenReceiverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TestNativeTokenReceiver *TestNativeTokenReceiverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TestNativeTokenReceiver.Contract.TestNativeTokenReceiverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TestNativeTokenReceiver *TestNativeTokenReceiverCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TestNativeTokenReceiver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TestNativeTokenReceiver *TestNativeTokenReceiverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TestNativeTokenReceiver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TestNativeTokenReceiver *TestNativeTokenReceiverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TestNativeTokenReceiver.Contract.contract.Transact(opts, method, params...)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_TestNativeTokenReceiver *TestNativeTokenReceiverTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _TestNativeTokenReceiver.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_TestNativeTokenReceiver *TestNativeTokenReceiverSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _TestNativeTokenReceiver.Contract.Fallback(&_TestNativeTokenReceiver.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_TestNativeTokenReceiver *TestNativeTokenReceiverTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _TestNativeTokenReceiver.Contract.Fallback(&_TestNativeTokenReceiver.TransactOpts, calldata)
}

// TestNativeTokenReceiverBreadReceivedIterator is returned from FilterBreadReceived and is used to iterate over the raw logs and unpacked data for BreadReceived events raised by the TestNativeTokenReceiver contract.
type TestNativeTokenReceiverBreadReceivedIterator struct {
	Event *TestNativeTokenReceiverBreadReceived // Event containing the contract specifics and raw log

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
func (it *TestNativeTokenReceiverBreadReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TestNativeTokenReceiverBreadReceived)
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
		it.Event = new(TestNativeTokenReceiverBreadReceived)
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
func (it *TestNativeTokenReceiverBreadReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TestNativeTokenReceiverBreadReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TestNativeTokenReceiverBreadReceived represents a BreadReceived event raised by the TestNativeTokenReceiver contract.
type TestNativeTokenReceiverBreadReceived struct {
	From         common.Address
	Amount       *big.Int
	ForwardedGas *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBreadReceived is a free log retrieval operation binding the contract event 0x16549311ba52796916987df5401f791fb06b998524a5a8684010010415850bb3.
//
// Solidity: event BreadReceived(address indexed from, uint256 amount, uint256 forwardedGas)
func (_TestNativeTokenReceiver *TestNativeTokenReceiverFilterer) FilterBreadReceived(opts *bind.FilterOpts, from []common.Address) (*TestNativeTokenReceiverBreadReceivedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _TestNativeTokenReceiver.contract.FilterLogs(opts, "BreadReceived", fromRule)
	if err != nil {
		return nil, err
	}
	return &TestNativeTokenReceiverBreadReceivedIterator{contract: _TestNativeTokenReceiver.contract, event: "BreadReceived", logs: logs, sub: sub}, nil
}

// WatchBreadReceived is a free log subscription operation binding the contract event 0x16549311ba52796916987df5401f791fb06b998524a5a8684010010415850bb3.
//
// Solidity: event BreadReceived(address indexed from, uint256 amount, uint256 forwardedGas)
func (_TestNativeTokenReceiver *TestNativeTokenReceiverFilterer) WatchBreadReceived(opts *bind.WatchOpts, sink chan<- *TestNativeTokenReceiverBreadReceived, from []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _TestNativeTokenReceiver.contract.WatchLogs(opts, "BreadReceived", fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TestNativeTokenReceiverBreadReceived)
				if err := _TestNativeTokenReceiver.contract.UnpackLog(event, "BreadReceived", log); err != nil {
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

// ParseBreadReceived is a log parse operation binding the contract event 0x16549311ba52796916987df5401f791fb06b998524a5a8684010010415850bb3.
//
// Solidity: event BreadReceived(address indexed from, uint256 amount, uint256 forwardedGas)
func (_TestNativeTokenReceiver *TestNativeTokenReceiverFilterer) ParseBreadReceived(log types.Log) (*TestNativeTokenReceiverBreadReceived, error) {
	event := new(TestNativeTokenReceiverBreadReceived)
	if err := _TestNativeTokenReceiver.contract.UnpackLog(event, "BreadReceived", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenMetaData contains all meta data concerning the Token contract.
var TokenMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// TokenABI is the input ABI used to generate the binding from.
// Deprecated: Use TokenMetaData.ABI instead.
var TokenABI = TokenMetaData.ABI

// Token is an auto generated Go binding around an Ethereum contract.
type Token struct {
	TokenCaller     // Read-only binding to the contract
	TokenTransactor // Write-only binding to the contract
	TokenFilterer   // Log filterer for contract events
}

// TokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type TokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TokenSession struct {
	Contract     *Token            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TokenCallerSession struct {
	Contract *TokenCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// TokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TokenTransactorSession struct {
	Contract     *TokenTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type TokenRaw struct {
	Contract *Token // Generic contract binding to access the raw methods on
}

// TokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TokenCallerRaw struct {
	Contract *TokenCaller // Generic read-only contract binding to access the raw methods on
}

// TokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TokenTransactorRaw struct {
	Contract *TokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewToken creates a new instance of Token, bound to a specific deployed contract.
func NewToken(address common.Address, backend bind.ContractBackend) (*Token, error) {
	contract, err := bindToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Token{TokenCaller: TokenCaller{contract: contract}, TokenTransactor: TokenTransactor{contract: contract}, TokenFilterer: TokenFilterer{contract: contract}}, nil
}

// NewTokenCaller creates a new read-only instance of Token, bound to a specific deployed contract.
func NewTokenCaller(address common.Address, caller bind.ContractCaller) (*TokenCaller, error) {
	contract, err := bindToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TokenCaller{contract: contract}, nil
}

// NewTokenTransactor creates a new write-only instance of Token, bound to a specific deployed contract.
func NewTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*TokenTransactor, error) {
	contract, err := bindToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TokenTransactor{contract: contract}, nil
}

// NewTokenFilterer creates a new log filterer instance of Token, bound to a specific deployed contract.
func NewTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*TokenFilterer, error) {
	contract, err := bindToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TokenFilterer{contract: contract}, nil
}

// bindToken binds a generic wrapper to an already deployed contract.
func bindToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TokenMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Token *TokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Token.Contract.TokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Token *TokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.Contract.TokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Token *TokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Token.Contract.TokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Token *TokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Token.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Token *TokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Token *TokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Token.Contract.contract.Transact(opts, method, params...)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 value) returns(bool)
func (_Token *TokenTransactor) Transfer(opts *bind.TransactOpts, _to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "transfer", _to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 value) returns(bool)
func (_Token *TokenSession) Transfer(_to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Transfer(&_Token.TransactOpts, _to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 value) returns(bool)
func (_Token *TokenTransactorSession) Transfer(_to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Transfer(&_Token.TransactOpts, _to, value)
}
