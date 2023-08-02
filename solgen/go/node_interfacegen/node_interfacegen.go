// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package node_interfacegen

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

// NodeInterfaceDebugRetryableInfo is an auto generated low-level Go binding around an user-defined struct.
type NodeInterfaceDebugRetryableInfo struct {
	Timeout     uint64
	From        common.Address
	To          common.Address
	Value       *big.Int
	Beneficiary common.Address
	Tries       uint64
	Data        []byte
}

// NodeInterfaceMetaData contains all meta data concerning the NodeInterface contract.
var NodeInterfaceMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"leaf\",\"type\":\"uint64\"}],\"name\":\"constructOutboxProof\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"send\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"deposit\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2CallValue\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"excessFeeRefundAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"callValueRefundAddress\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"estimateRetryableTicket\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"blockNum\",\"type\":\"uint64\"}],\"name\":\"findBatchContainingBlock\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"batch\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"contractCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"gasEstimateComponents\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"gasEstimate\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"gasEstimateForL1\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1BaseFeeEstimate\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"contractCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"gasEstimateL1Component\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"gasEstimateForL1\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1BaseFeeEstimate\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"name\":\"getL1Confirmations\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"confirmations\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"batchNum\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"index\",\"type\":\"uint64\"}],\"name\":\"legacyLookupMessageBatchProof\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"path\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1Dest\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"calldataForL1\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nitroGenesisBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"number\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
}

// NodeInterfaceABI is the input ABI used to generate the binding from.
// Deprecated: Use NodeInterfaceMetaData.ABI instead.
var NodeInterfaceABI = NodeInterfaceMetaData.ABI

// NodeInterface is an auto generated Go binding around an Ethereum contract.
type NodeInterface struct {
	NodeInterfaceCaller     // Read-only binding to the contract
	NodeInterfaceTransactor // Write-only binding to the contract
	NodeInterfaceFilterer   // Log filterer for contract events
}

// NodeInterfaceCaller is an auto generated read-only Go binding around an Ethereum contract.
type NodeInterfaceCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodeInterfaceTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NodeInterfaceTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodeInterfaceFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NodeInterfaceFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodeInterfaceSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NodeInterfaceSession struct {
	Contract     *NodeInterface    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// NodeInterfaceCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NodeInterfaceCallerSession struct {
	Contract *NodeInterfaceCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// NodeInterfaceTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NodeInterfaceTransactorSession struct {
	Contract     *NodeInterfaceTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// NodeInterfaceRaw is an auto generated low-level Go binding around an Ethereum contract.
type NodeInterfaceRaw struct {
	Contract *NodeInterface // Generic contract binding to access the raw methods on
}

// NodeInterfaceCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NodeInterfaceCallerRaw struct {
	Contract *NodeInterfaceCaller // Generic read-only contract binding to access the raw methods on
}

// NodeInterfaceTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NodeInterfaceTransactorRaw struct {
	Contract *NodeInterfaceTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNodeInterface creates a new instance of NodeInterface, bound to a specific deployed contract.
func NewNodeInterface(address common.Address, backend bind.ContractBackend) (*NodeInterface, error) {
	contract, err := bindNodeInterface(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NodeInterface{NodeInterfaceCaller: NodeInterfaceCaller{contract: contract}, NodeInterfaceTransactor: NodeInterfaceTransactor{contract: contract}, NodeInterfaceFilterer: NodeInterfaceFilterer{contract: contract}}, nil
}

// NewNodeInterfaceCaller creates a new read-only instance of NodeInterface, bound to a specific deployed contract.
func NewNodeInterfaceCaller(address common.Address, caller bind.ContractCaller) (*NodeInterfaceCaller, error) {
	contract, err := bindNodeInterface(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NodeInterfaceCaller{contract: contract}, nil
}

// NewNodeInterfaceTransactor creates a new write-only instance of NodeInterface, bound to a specific deployed contract.
func NewNodeInterfaceTransactor(address common.Address, transactor bind.ContractTransactor) (*NodeInterfaceTransactor, error) {
	contract, err := bindNodeInterface(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NodeInterfaceTransactor{contract: contract}, nil
}

// NewNodeInterfaceFilterer creates a new log filterer instance of NodeInterface, bound to a specific deployed contract.
func NewNodeInterfaceFilterer(address common.Address, filterer bind.ContractFilterer) (*NodeInterfaceFilterer, error) {
	contract, err := bindNodeInterface(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NodeInterfaceFilterer{contract: contract}, nil
}

// bindNodeInterface binds a generic wrapper to an already deployed contract.
func bindNodeInterface(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NodeInterfaceMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NodeInterface *NodeInterfaceRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NodeInterface.Contract.NodeInterfaceCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NodeInterface *NodeInterfaceRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NodeInterface.Contract.NodeInterfaceTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NodeInterface *NodeInterfaceRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NodeInterface.Contract.NodeInterfaceTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NodeInterface *NodeInterfaceCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NodeInterface.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NodeInterface *NodeInterfaceTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NodeInterface.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NodeInterface *NodeInterfaceTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NodeInterface.Contract.contract.Transact(opts, method, params...)
}

// ConstructOutboxProof is a free data retrieval call binding the contract method 0x42696350.
//
// Solidity: function constructOutboxProof(uint64 size, uint64 leaf) view returns(bytes32 send, bytes32 root, bytes32[] proof)
func (_NodeInterface *NodeInterfaceCaller) ConstructOutboxProof(opts *bind.CallOpts, size uint64, leaf uint64) (struct {
	Send  [32]byte
	Root  [32]byte
	Proof [][32]byte
}, error) {
	var out []interface{}
	err := _NodeInterface.contract.Call(opts, &out, "constructOutboxProof", size, leaf)

	outstruct := new(struct {
		Send  [32]byte
		Root  [32]byte
		Proof [][32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Send = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Root = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.Proof = *abi.ConvertType(out[2], new([][32]byte)).(*[][32]byte)

	return *outstruct, err

}

// ConstructOutboxProof is a free data retrieval call binding the contract method 0x42696350.
//
// Solidity: function constructOutboxProof(uint64 size, uint64 leaf) view returns(bytes32 send, bytes32 root, bytes32[] proof)
func (_NodeInterface *NodeInterfaceSession) ConstructOutboxProof(size uint64, leaf uint64) (struct {
	Send  [32]byte
	Root  [32]byte
	Proof [][32]byte
}, error) {
	return _NodeInterface.Contract.ConstructOutboxProof(&_NodeInterface.CallOpts, size, leaf)
}

// ConstructOutboxProof is a free data retrieval call binding the contract method 0x42696350.
//
// Solidity: function constructOutboxProof(uint64 size, uint64 leaf) view returns(bytes32 send, bytes32 root, bytes32[] proof)
func (_NodeInterface *NodeInterfaceCallerSession) ConstructOutboxProof(size uint64, leaf uint64) (struct {
	Send  [32]byte
	Root  [32]byte
	Proof [][32]byte
}, error) {
	return _NodeInterface.Contract.ConstructOutboxProof(&_NodeInterface.CallOpts, size, leaf)
}

// FindBatchContainingBlock is a free data retrieval call binding the contract method 0x81f1adaf.
//
// Solidity: function findBatchContainingBlock(uint64 blockNum) view returns(uint64 batch)
func (_NodeInterface *NodeInterfaceCaller) FindBatchContainingBlock(opts *bind.CallOpts, blockNum uint64) (uint64, error) {
	var out []interface{}
	err := _NodeInterface.contract.Call(opts, &out, "findBatchContainingBlock", blockNum)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// FindBatchContainingBlock is a free data retrieval call binding the contract method 0x81f1adaf.
//
// Solidity: function findBatchContainingBlock(uint64 blockNum) view returns(uint64 batch)
func (_NodeInterface *NodeInterfaceSession) FindBatchContainingBlock(blockNum uint64) (uint64, error) {
	return _NodeInterface.Contract.FindBatchContainingBlock(&_NodeInterface.CallOpts, blockNum)
}

// FindBatchContainingBlock is a free data retrieval call binding the contract method 0x81f1adaf.
//
// Solidity: function findBatchContainingBlock(uint64 blockNum) view returns(uint64 batch)
func (_NodeInterface *NodeInterfaceCallerSession) FindBatchContainingBlock(blockNum uint64) (uint64, error) {
	return _NodeInterface.Contract.FindBatchContainingBlock(&_NodeInterface.CallOpts, blockNum)
}

// GetL1Confirmations is a free data retrieval call binding the contract method 0xe5ca238c.
//
// Solidity: function getL1Confirmations(bytes32 blockHash) view returns(uint64 confirmations)
func (_NodeInterface *NodeInterfaceCaller) GetL1Confirmations(opts *bind.CallOpts, blockHash [32]byte) (uint64, error) {
	var out []interface{}
	err := _NodeInterface.contract.Call(opts, &out, "getL1Confirmations", blockHash)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetL1Confirmations is a free data retrieval call binding the contract method 0xe5ca238c.
//
// Solidity: function getL1Confirmations(bytes32 blockHash) view returns(uint64 confirmations)
func (_NodeInterface *NodeInterfaceSession) GetL1Confirmations(blockHash [32]byte) (uint64, error) {
	return _NodeInterface.Contract.GetL1Confirmations(&_NodeInterface.CallOpts, blockHash)
}

// GetL1Confirmations is a free data retrieval call binding the contract method 0xe5ca238c.
//
// Solidity: function getL1Confirmations(bytes32 blockHash) view returns(uint64 confirmations)
func (_NodeInterface *NodeInterfaceCallerSession) GetL1Confirmations(blockHash [32]byte) (uint64, error) {
	return _NodeInterface.Contract.GetL1Confirmations(&_NodeInterface.CallOpts, blockHash)
}

// LegacyLookupMessageBatchProof is a free data retrieval call binding the contract method 0x89496270.
//
// Solidity: function legacyLookupMessageBatchProof(uint256 batchNum, uint64 index) view returns(bytes32[] proof, uint256 path, address l2Sender, address l1Dest, uint256 l2Block, uint256 l1Block, uint256 timestamp, uint256 amount, bytes calldataForL1)
func (_NodeInterface *NodeInterfaceCaller) LegacyLookupMessageBatchProof(opts *bind.CallOpts, batchNum *big.Int, index uint64) (struct {
	Proof         [][32]byte
	Path          *big.Int
	L2Sender      common.Address
	L1Dest        common.Address
	L2Block       *big.Int
	L1Block       *big.Int
	Timestamp     *big.Int
	Amount        *big.Int
	CalldataForL1 []byte
}, error) {
	var out []interface{}
	err := _NodeInterface.contract.Call(opts, &out, "legacyLookupMessageBatchProof", batchNum, index)

	outstruct := new(struct {
		Proof         [][32]byte
		Path          *big.Int
		L2Sender      common.Address
		L1Dest        common.Address
		L2Block       *big.Int
		L1Block       *big.Int
		Timestamp     *big.Int
		Amount        *big.Int
		CalldataForL1 []byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Proof = *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)
	outstruct.Path = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.L2Sender = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.L1Dest = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)
	outstruct.L2Block = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.L1Block = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.Timestamp = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)
	outstruct.Amount = *abi.ConvertType(out[7], new(*big.Int)).(**big.Int)
	outstruct.CalldataForL1 = *abi.ConvertType(out[8], new([]byte)).(*[]byte)

	return *outstruct, err

}

// LegacyLookupMessageBatchProof is a free data retrieval call binding the contract method 0x89496270.
//
// Solidity: function legacyLookupMessageBatchProof(uint256 batchNum, uint64 index) view returns(bytes32[] proof, uint256 path, address l2Sender, address l1Dest, uint256 l2Block, uint256 l1Block, uint256 timestamp, uint256 amount, bytes calldataForL1)
func (_NodeInterface *NodeInterfaceSession) LegacyLookupMessageBatchProof(batchNum *big.Int, index uint64) (struct {
	Proof         [][32]byte
	Path          *big.Int
	L2Sender      common.Address
	L1Dest        common.Address
	L2Block       *big.Int
	L1Block       *big.Int
	Timestamp     *big.Int
	Amount        *big.Int
	CalldataForL1 []byte
}, error) {
	return _NodeInterface.Contract.LegacyLookupMessageBatchProof(&_NodeInterface.CallOpts, batchNum, index)
}

// LegacyLookupMessageBatchProof is a free data retrieval call binding the contract method 0x89496270.
//
// Solidity: function legacyLookupMessageBatchProof(uint256 batchNum, uint64 index) view returns(bytes32[] proof, uint256 path, address l2Sender, address l1Dest, uint256 l2Block, uint256 l1Block, uint256 timestamp, uint256 amount, bytes calldataForL1)
func (_NodeInterface *NodeInterfaceCallerSession) LegacyLookupMessageBatchProof(batchNum *big.Int, index uint64) (struct {
	Proof         [][32]byte
	Path          *big.Int
	L2Sender      common.Address
	L1Dest        common.Address
	L2Block       *big.Int
	L1Block       *big.Int
	Timestamp     *big.Int
	Amount        *big.Int
	CalldataForL1 []byte
}, error) {
	return _NodeInterface.Contract.LegacyLookupMessageBatchProof(&_NodeInterface.CallOpts, batchNum, index)
}

// NitroGenesisBlock is a free data retrieval call binding the contract method 0x93a2fe21.
//
// Solidity: function nitroGenesisBlock() pure returns(uint256 number)
func (_NodeInterface *NodeInterfaceCaller) NitroGenesisBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NodeInterface.contract.Call(opts, &out, "nitroGenesisBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NitroGenesisBlock is a free data retrieval call binding the contract method 0x93a2fe21.
//
// Solidity: function nitroGenesisBlock() pure returns(uint256 number)
func (_NodeInterface *NodeInterfaceSession) NitroGenesisBlock() (*big.Int, error) {
	return _NodeInterface.Contract.NitroGenesisBlock(&_NodeInterface.CallOpts)
}

// NitroGenesisBlock is a free data retrieval call binding the contract method 0x93a2fe21.
//
// Solidity: function nitroGenesisBlock() pure returns(uint256 number)
func (_NodeInterface *NodeInterfaceCallerSession) NitroGenesisBlock() (*big.Int, error) {
	return _NodeInterface.Contract.NitroGenesisBlock(&_NodeInterface.CallOpts)
}

// EstimateRetryableTicket is a paid mutator transaction binding the contract method 0xc3dc5879.
//
// Solidity: function estimateRetryableTicket(address sender, uint256 deposit, address to, uint256 l2CallValue, address excessFeeRefundAddress, address callValueRefundAddress, bytes data) returns()
func (_NodeInterface *NodeInterfaceTransactor) EstimateRetryableTicket(opts *bind.TransactOpts, sender common.Address, deposit *big.Int, to common.Address, l2CallValue *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, data []byte) (*types.Transaction, error) {
	return _NodeInterface.contract.Transact(opts, "estimateRetryableTicket", sender, deposit, to, l2CallValue, excessFeeRefundAddress, callValueRefundAddress, data)
}

// EstimateRetryableTicket is a paid mutator transaction binding the contract method 0xc3dc5879.
//
// Solidity: function estimateRetryableTicket(address sender, uint256 deposit, address to, uint256 l2CallValue, address excessFeeRefundAddress, address callValueRefundAddress, bytes data) returns()
func (_NodeInterface *NodeInterfaceSession) EstimateRetryableTicket(sender common.Address, deposit *big.Int, to common.Address, l2CallValue *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, data []byte) (*types.Transaction, error) {
	return _NodeInterface.Contract.EstimateRetryableTicket(&_NodeInterface.TransactOpts, sender, deposit, to, l2CallValue, excessFeeRefundAddress, callValueRefundAddress, data)
}

// EstimateRetryableTicket is a paid mutator transaction binding the contract method 0xc3dc5879.
//
// Solidity: function estimateRetryableTicket(address sender, uint256 deposit, address to, uint256 l2CallValue, address excessFeeRefundAddress, address callValueRefundAddress, bytes data) returns()
func (_NodeInterface *NodeInterfaceTransactorSession) EstimateRetryableTicket(sender common.Address, deposit *big.Int, to common.Address, l2CallValue *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, data []byte) (*types.Transaction, error) {
	return _NodeInterface.Contract.EstimateRetryableTicket(&_NodeInterface.TransactOpts, sender, deposit, to, l2CallValue, excessFeeRefundAddress, callValueRefundAddress, data)
}

// GasEstimateComponents is a paid mutator transaction binding the contract method 0xc94e6eeb.
//
// Solidity: function gasEstimateComponents(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimate, uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_NodeInterface *NodeInterfaceTransactor) GasEstimateComponents(opts *bind.TransactOpts, to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _NodeInterface.contract.Transact(opts, "gasEstimateComponents", to, contractCreation, data)
}

// GasEstimateComponents is a paid mutator transaction binding the contract method 0xc94e6eeb.
//
// Solidity: function gasEstimateComponents(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimate, uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_NodeInterface *NodeInterfaceSession) GasEstimateComponents(to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _NodeInterface.Contract.GasEstimateComponents(&_NodeInterface.TransactOpts, to, contractCreation, data)
}

// GasEstimateComponents is a paid mutator transaction binding the contract method 0xc94e6eeb.
//
// Solidity: function gasEstimateComponents(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimate, uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_NodeInterface *NodeInterfaceTransactorSession) GasEstimateComponents(to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _NodeInterface.Contract.GasEstimateComponents(&_NodeInterface.TransactOpts, to, contractCreation, data)
}

// GasEstimateL1Component is a paid mutator transaction binding the contract method 0x77d488a2.
//
// Solidity: function gasEstimateL1Component(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_NodeInterface *NodeInterfaceTransactor) GasEstimateL1Component(opts *bind.TransactOpts, to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _NodeInterface.contract.Transact(opts, "gasEstimateL1Component", to, contractCreation, data)
}

// GasEstimateL1Component is a paid mutator transaction binding the contract method 0x77d488a2.
//
// Solidity: function gasEstimateL1Component(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_NodeInterface *NodeInterfaceSession) GasEstimateL1Component(to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _NodeInterface.Contract.GasEstimateL1Component(&_NodeInterface.TransactOpts, to, contractCreation, data)
}

// GasEstimateL1Component is a paid mutator transaction binding the contract method 0x77d488a2.
//
// Solidity: function gasEstimateL1Component(address to, bool contractCreation, bytes data) payable returns(uint64 gasEstimateForL1, uint256 baseFee, uint256 l1BaseFeeEstimate)
func (_NodeInterface *NodeInterfaceTransactorSession) GasEstimateL1Component(to common.Address, contractCreation bool, data []byte) (*types.Transaction, error) {
	return _NodeInterface.Contract.GasEstimateL1Component(&_NodeInterface.TransactOpts, to, contractCreation, data)
}

// NodeInterfaceDebugMetaData contains all meta data concerning the NodeInterfaceDebug contract.
var NodeInterfaceDebugMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ticket\",\"type\":\"bytes32\"}],\"name\":\"getRetryable\",\"outputs\":[{\"components\":[{\"internalType\":\"uint64\",\"name\":\"timeout\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"beneficiary\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"tries\",\"type\":\"uint64\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structNodeInterfaceDebug.RetryableInfo\",\"name\":\"retryable\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// NodeInterfaceDebugABI is the input ABI used to generate the binding from.
// Deprecated: Use NodeInterfaceDebugMetaData.ABI instead.
var NodeInterfaceDebugABI = NodeInterfaceDebugMetaData.ABI

// NodeInterfaceDebug is an auto generated Go binding around an Ethereum contract.
type NodeInterfaceDebug struct {
	NodeInterfaceDebugCaller     // Read-only binding to the contract
	NodeInterfaceDebugTransactor // Write-only binding to the contract
	NodeInterfaceDebugFilterer   // Log filterer for contract events
}

// NodeInterfaceDebugCaller is an auto generated read-only Go binding around an Ethereum contract.
type NodeInterfaceDebugCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodeInterfaceDebugTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NodeInterfaceDebugTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodeInterfaceDebugFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NodeInterfaceDebugFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodeInterfaceDebugSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NodeInterfaceDebugSession struct {
	Contract     *NodeInterfaceDebug // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// NodeInterfaceDebugCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NodeInterfaceDebugCallerSession struct {
	Contract *NodeInterfaceDebugCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// NodeInterfaceDebugTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NodeInterfaceDebugTransactorSession struct {
	Contract     *NodeInterfaceDebugTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// NodeInterfaceDebugRaw is an auto generated low-level Go binding around an Ethereum contract.
type NodeInterfaceDebugRaw struct {
	Contract *NodeInterfaceDebug // Generic contract binding to access the raw methods on
}

// NodeInterfaceDebugCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NodeInterfaceDebugCallerRaw struct {
	Contract *NodeInterfaceDebugCaller // Generic read-only contract binding to access the raw methods on
}

// NodeInterfaceDebugTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NodeInterfaceDebugTransactorRaw struct {
	Contract *NodeInterfaceDebugTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNodeInterfaceDebug creates a new instance of NodeInterfaceDebug, bound to a specific deployed contract.
func NewNodeInterfaceDebug(address common.Address, backend bind.ContractBackend) (*NodeInterfaceDebug, error) {
	contract, err := bindNodeInterfaceDebug(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NodeInterfaceDebug{NodeInterfaceDebugCaller: NodeInterfaceDebugCaller{contract: contract}, NodeInterfaceDebugTransactor: NodeInterfaceDebugTransactor{contract: contract}, NodeInterfaceDebugFilterer: NodeInterfaceDebugFilterer{contract: contract}}, nil
}

// NewNodeInterfaceDebugCaller creates a new read-only instance of NodeInterfaceDebug, bound to a specific deployed contract.
func NewNodeInterfaceDebugCaller(address common.Address, caller bind.ContractCaller) (*NodeInterfaceDebugCaller, error) {
	contract, err := bindNodeInterfaceDebug(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NodeInterfaceDebugCaller{contract: contract}, nil
}

// NewNodeInterfaceDebugTransactor creates a new write-only instance of NodeInterfaceDebug, bound to a specific deployed contract.
func NewNodeInterfaceDebugTransactor(address common.Address, transactor bind.ContractTransactor) (*NodeInterfaceDebugTransactor, error) {
	contract, err := bindNodeInterfaceDebug(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NodeInterfaceDebugTransactor{contract: contract}, nil
}

// NewNodeInterfaceDebugFilterer creates a new log filterer instance of NodeInterfaceDebug, bound to a specific deployed contract.
func NewNodeInterfaceDebugFilterer(address common.Address, filterer bind.ContractFilterer) (*NodeInterfaceDebugFilterer, error) {
	contract, err := bindNodeInterfaceDebug(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NodeInterfaceDebugFilterer{contract: contract}, nil
}

// bindNodeInterfaceDebug binds a generic wrapper to an already deployed contract.
func bindNodeInterfaceDebug(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NodeInterfaceDebugMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NodeInterfaceDebug *NodeInterfaceDebugRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NodeInterfaceDebug.Contract.NodeInterfaceDebugCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NodeInterfaceDebug *NodeInterfaceDebugRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NodeInterfaceDebug.Contract.NodeInterfaceDebugTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NodeInterfaceDebug *NodeInterfaceDebugRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NodeInterfaceDebug.Contract.NodeInterfaceDebugTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NodeInterfaceDebug *NodeInterfaceDebugCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NodeInterfaceDebug.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NodeInterfaceDebug *NodeInterfaceDebugTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NodeInterfaceDebug.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NodeInterfaceDebug *NodeInterfaceDebugTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NodeInterfaceDebug.Contract.contract.Transact(opts, method, params...)
}

// GetRetryable is a free data retrieval call binding the contract method 0x4d7953ad.
//
// Solidity: function getRetryable(bytes32 ticket) view returns((uint64,address,address,uint256,address,uint64,bytes) retryable)
func (_NodeInterfaceDebug *NodeInterfaceDebugCaller) GetRetryable(opts *bind.CallOpts, ticket [32]byte) (NodeInterfaceDebugRetryableInfo, error) {
	var out []interface{}
	err := _NodeInterfaceDebug.contract.Call(opts, &out, "getRetryable", ticket)

	if err != nil {
		return *new(NodeInterfaceDebugRetryableInfo), err
	}

	out0 := *abi.ConvertType(out[0], new(NodeInterfaceDebugRetryableInfo)).(*NodeInterfaceDebugRetryableInfo)

	return out0, err

}

// GetRetryable is a free data retrieval call binding the contract method 0x4d7953ad.
//
// Solidity: function getRetryable(bytes32 ticket) view returns((uint64,address,address,uint256,address,uint64,bytes) retryable)
func (_NodeInterfaceDebug *NodeInterfaceDebugSession) GetRetryable(ticket [32]byte) (NodeInterfaceDebugRetryableInfo, error) {
	return _NodeInterfaceDebug.Contract.GetRetryable(&_NodeInterfaceDebug.CallOpts, ticket)
}

// GetRetryable is a free data retrieval call binding the contract method 0x4d7953ad.
//
// Solidity: function getRetryable(bytes32 ticket) view returns((uint64,address,address,uint256,address,uint64,bytes) retryable)
func (_NodeInterfaceDebug *NodeInterfaceDebugCallerSession) GetRetryable(ticket [32]byte) (NodeInterfaceDebugRetryableInfo, error) {
	return _NodeInterfaceDebug.Contract.GetRetryable(&_NodeInterfaceDebug.CallOpts, ticket)
}
