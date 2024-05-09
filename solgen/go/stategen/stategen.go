// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package stategen

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

// DeserializeMetaData contains all meta data concerning the Deserialize contract.
var DeserializeMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea264697066735822122014f114f817bdd2e4465b7e9fb73bb0d009223b43791499c09c144877d6a7891364736f6c63430008110033",
}

// DeserializeABI is the input ABI used to generate the binding from.
// Deprecated: Use DeserializeMetaData.ABI instead.
var DeserializeABI = DeserializeMetaData.ABI

// DeserializeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DeserializeMetaData.Bin instead.
var DeserializeBin = DeserializeMetaData.Bin

// DeployDeserialize deploys a new Ethereum contract, binding an instance of Deserialize to it.
func DeployDeserialize(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Deserialize, error) {
	parsed, err := DeserializeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DeserializeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Deserialize{DeserializeCaller: DeserializeCaller{contract: contract}, DeserializeTransactor: DeserializeTransactor{contract: contract}, DeserializeFilterer: DeserializeFilterer{contract: contract}}, nil
}

// Deserialize is an auto generated Go binding around an Ethereum contract.
type Deserialize struct {
	DeserializeCaller     // Read-only binding to the contract
	DeserializeTransactor // Write-only binding to the contract
	DeserializeFilterer   // Log filterer for contract events
}

// DeserializeCaller is an auto generated read-only Go binding around an Ethereum contract.
type DeserializeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DeserializeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DeserializeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DeserializeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DeserializeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DeserializeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DeserializeSession struct {
	Contract     *Deserialize      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DeserializeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DeserializeCallerSession struct {
	Contract *DeserializeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// DeserializeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DeserializeTransactorSession struct {
	Contract     *DeserializeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// DeserializeRaw is an auto generated low-level Go binding around an Ethereum contract.
type DeserializeRaw struct {
	Contract *Deserialize // Generic contract binding to access the raw methods on
}

// DeserializeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DeserializeCallerRaw struct {
	Contract *DeserializeCaller // Generic read-only contract binding to access the raw methods on
}

// DeserializeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DeserializeTransactorRaw struct {
	Contract *DeserializeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDeserialize creates a new instance of Deserialize, bound to a specific deployed contract.
func NewDeserialize(address common.Address, backend bind.ContractBackend) (*Deserialize, error) {
	contract, err := bindDeserialize(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Deserialize{DeserializeCaller: DeserializeCaller{contract: contract}, DeserializeTransactor: DeserializeTransactor{contract: contract}, DeserializeFilterer: DeserializeFilterer{contract: contract}}, nil
}

// NewDeserializeCaller creates a new read-only instance of Deserialize, bound to a specific deployed contract.
func NewDeserializeCaller(address common.Address, caller bind.ContractCaller) (*DeserializeCaller, error) {
	contract, err := bindDeserialize(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DeserializeCaller{contract: contract}, nil
}

// NewDeserializeTransactor creates a new write-only instance of Deserialize, bound to a specific deployed contract.
func NewDeserializeTransactor(address common.Address, transactor bind.ContractTransactor) (*DeserializeTransactor, error) {
	contract, err := bindDeserialize(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DeserializeTransactor{contract: contract}, nil
}

// NewDeserializeFilterer creates a new log filterer instance of Deserialize, bound to a specific deployed contract.
func NewDeserializeFilterer(address common.Address, filterer bind.ContractFilterer) (*DeserializeFilterer, error) {
	contract, err := bindDeserialize(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DeserializeFilterer{contract: contract}, nil
}

// bindDeserialize binds a generic wrapper to an already deployed contract.
func bindDeserialize(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DeserializeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Deserialize *DeserializeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Deserialize.Contract.DeserializeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Deserialize *DeserializeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Deserialize.Contract.DeserializeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Deserialize *DeserializeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Deserialize.Contract.DeserializeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Deserialize *DeserializeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Deserialize.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Deserialize *DeserializeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Deserialize.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Deserialize *DeserializeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Deserialize.Contract.contract.Transact(opts, method, params...)
}

// GlobalStateLibMetaData contains all meta data concerning the GlobalStateLib contract.
var GlobalStateLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220d98fed73ef0a1f84db8e72e67fc1d858690396e7564a4490c47cda1449b533d064736f6c63430008110033",
}

// GlobalStateLibABI is the input ABI used to generate the binding from.
// Deprecated: Use GlobalStateLibMetaData.ABI instead.
var GlobalStateLibABI = GlobalStateLibMetaData.ABI

// GlobalStateLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use GlobalStateLibMetaData.Bin instead.
var GlobalStateLibBin = GlobalStateLibMetaData.Bin

// DeployGlobalStateLib deploys a new Ethereum contract, binding an instance of GlobalStateLib to it.
func DeployGlobalStateLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *GlobalStateLib, error) {
	parsed, err := GlobalStateLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(GlobalStateLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &GlobalStateLib{GlobalStateLibCaller: GlobalStateLibCaller{contract: contract}, GlobalStateLibTransactor: GlobalStateLibTransactor{contract: contract}, GlobalStateLibFilterer: GlobalStateLibFilterer{contract: contract}}, nil
}

// GlobalStateLib is an auto generated Go binding around an Ethereum contract.
type GlobalStateLib struct {
	GlobalStateLibCaller     // Read-only binding to the contract
	GlobalStateLibTransactor // Write-only binding to the contract
	GlobalStateLibFilterer   // Log filterer for contract events
}

// GlobalStateLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type GlobalStateLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GlobalStateLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GlobalStateLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GlobalStateLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GlobalStateLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GlobalStateLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GlobalStateLibSession struct {
	Contract     *GlobalStateLib   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// GlobalStateLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GlobalStateLibCallerSession struct {
	Contract *GlobalStateLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// GlobalStateLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GlobalStateLibTransactorSession struct {
	Contract     *GlobalStateLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// GlobalStateLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type GlobalStateLibRaw struct {
	Contract *GlobalStateLib // Generic contract binding to access the raw methods on
}

// GlobalStateLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GlobalStateLibCallerRaw struct {
	Contract *GlobalStateLibCaller // Generic read-only contract binding to access the raw methods on
}

// GlobalStateLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GlobalStateLibTransactorRaw struct {
	Contract *GlobalStateLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGlobalStateLib creates a new instance of GlobalStateLib, bound to a specific deployed contract.
func NewGlobalStateLib(address common.Address, backend bind.ContractBackend) (*GlobalStateLib, error) {
	contract, err := bindGlobalStateLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &GlobalStateLib{GlobalStateLibCaller: GlobalStateLibCaller{contract: contract}, GlobalStateLibTransactor: GlobalStateLibTransactor{contract: contract}, GlobalStateLibFilterer: GlobalStateLibFilterer{contract: contract}}, nil
}

// NewGlobalStateLibCaller creates a new read-only instance of GlobalStateLib, bound to a specific deployed contract.
func NewGlobalStateLibCaller(address common.Address, caller bind.ContractCaller) (*GlobalStateLibCaller, error) {
	contract, err := bindGlobalStateLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GlobalStateLibCaller{contract: contract}, nil
}

// NewGlobalStateLibTransactor creates a new write-only instance of GlobalStateLib, bound to a specific deployed contract.
func NewGlobalStateLibTransactor(address common.Address, transactor bind.ContractTransactor) (*GlobalStateLibTransactor, error) {
	contract, err := bindGlobalStateLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GlobalStateLibTransactor{contract: contract}, nil
}

// NewGlobalStateLibFilterer creates a new log filterer instance of GlobalStateLib, bound to a specific deployed contract.
func NewGlobalStateLibFilterer(address common.Address, filterer bind.ContractFilterer) (*GlobalStateLibFilterer, error) {
	contract, err := bindGlobalStateLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GlobalStateLibFilterer{contract: contract}, nil
}

// bindGlobalStateLib binds a generic wrapper to an already deployed contract.
func bindGlobalStateLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := GlobalStateLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GlobalStateLib *GlobalStateLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GlobalStateLib.Contract.GlobalStateLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GlobalStateLib *GlobalStateLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GlobalStateLib.Contract.GlobalStateLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GlobalStateLib *GlobalStateLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GlobalStateLib.Contract.GlobalStateLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GlobalStateLib *GlobalStateLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GlobalStateLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GlobalStateLib *GlobalStateLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GlobalStateLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GlobalStateLib *GlobalStateLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GlobalStateLib.Contract.contract.Transact(opts, method, params...)
}

// InstructionsMetaData contains all meta data concerning the Instructions contract.
var InstructionsMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220f5ead78e84d301be8be1ac710a0980838d91ad64fee334062db3c2c2ae53f04c64736f6c63430008110033",
}

// InstructionsABI is the input ABI used to generate the binding from.
// Deprecated: Use InstructionsMetaData.ABI instead.
var InstructionsABI = InstructionsMetaData.ABI

// InstructionsBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use InstructionsMetaData.Bin instead.
var InstructionsBin = InstructionsMetaData.Bin

// DeployInstructions deploys a new Ethereum contract, binding an instance of Instructions to it.
func DeployInstructions(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Instructions, error) {
	parsed, err := InstructionsMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(InstructionsBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Instructions{InstructionsCaller: InstructionsCaller{contract: contract}, InstructionsTransactor: InstructionsTransactor{contract: contract}, InstructionsFilterer: InstructionsFilterer{contract: contract}}, nil
}

// Instructions is an auto generated Go binding around an Ethereum contract.
type Instructions struct {
	InstructionsCaller     // Read-only binding to the contract
	InstructionsTransactor // Write-only binding to the contract
	InstructionsFilterer   // Log filterer for contract events
}

// InstructionsCaller is an auto generated read-only Go binding around an Ethereum contract.
type InstructionsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InstructionsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InstructionsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InstructionsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InstructionsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InstructionsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InstructionsSession struct {
	Contract     *Instructions     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InstructionsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InstructionsCallerSession struct {
	Contract *InstructionsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// InstructionsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InstructionsTransactorSession struct {
	Contract     *InstructionsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// InstructionsRaw is an auto generated low-level Go binding around an Ethereum contract.
type InstructionsRaw struct {
	Contract *Instructions // Generic contract binding to access the raw methods on
}

// InstructionsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InstructionsCallerRaw struct {
	Contract *InstructionsCaller // Generic read-only contract binding to access the raw methods on
}

// InstructionsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InstructionsTransactorRaw struct {
	Contract *InstructionsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInstructions creates a new instance of Instructions, bound to a specific deployed contract.
func NewInstructions(address common.Address, backend bind.ContractBackend) (*Instructions, error) {
	contract, err := bindInstructions(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Instructions{InstructionsCaller: InstructionsCaller{contract: contract}, InstructionsTransactor: InstructionsTransactor{contract: contract}, InstructionsFilterer: InstructionsFilterer{contract: contract}}, nil
}

// NewInstructionsCaller creates a new read-only instance of Instructions, bound to a specific deployed contract.
func NewInstructionsCaller(address common.Address, caller bind.ContractCaller) (*InstructionsCaller, error) {
	contract, err := bindInstructions(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InstructionsCaller{contract: contract}, nil
}

// NewInstructionsTransactor creates a new write-only instance of Instructions, bound to a specific deployed contract.
func NewInstructionsTransactor(address common.Address, transactor bind.ContractTransactor) (*InstructionsTransactor, error) {
	contract, err := bindInstructions(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InstructionsTransactor{contract: contract}, nil
}

// NewInstructionsFilterer creates a new log filterer instance of Instructions, bound to a specific deployed contract.
func NewInstructionsFilterer(address common.Address, filterer bind.ContractFilterer) (*InstructionsFilterer, error) {
	contract, err := bindInstructions(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InstructionsFilterer{contract: contract}, nil
}

// bindInstructions binds a generic wrapper to an already deployed contract.
func bindInstructions(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := InstructionsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Instructions *InstructionsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Instructions.Contract.InstructionsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Instructions *InstructionsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Instructions.Contract.InstructionsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Instructions *InstructionsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Instructions.Contract.InstructionsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Instructions *InstructionsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Instructions.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Instructions *InstructionsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Instructions.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Instructions *InstructionsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Instructions.Contract.contract.Transact(opts, method, params...)
}

// MachineLibMetaData contains all meta data concerning the MachineLib contract.
var MachineLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212201829b0956e2e634bd9b3cd8e74daf12565245f850e3d8f833a5ab37ac51057bc64736f6c63430008110033",
}

// MachineLibABI is the input ABI used to generate the binding from.
// Deprecated: Use MachineLibMetaData.ABI instead.
var MachineLibABI = MachineLibMetaData.ABI

// MachineLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MachineLibMetaData.Bin instead.
var MachineLibBin = MachineLibMetaData.Bin

// DeployMachineLib deploys a new Ethereum contract, binding an instance of MachineLib to it.
func DeployMachineLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MachineLib, error) {
	parsed, err := MachineLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MachineLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MachineLib{MachineLibCaller: MachineLibCaller{contract: contract}, MachineLibTransactor: MachineLibTransactor{contract: contract}, MachineLibFilterer: MachineLibFilterer{contract: contract}}, nil
}

// MachineLib is an auto generated Go binding around an Ethereum contract.
type MachineLib struct {
	MachineLibCaller     // Read-only binding to the contract
	MachineLibTransactor // Write-only binding to the contract
	MachineLibFilterer   // Log filterer for contract events
}

// MachineLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type MachineLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MachineLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MachineLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MachineLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MachineLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MachineLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MachineLibSession struct {
	Contract     *MachineLib       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MachineLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MachineLibCallerSession struct {
	Contract *MachineLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// MachineLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MachineLibTransactorSession struct {
	Contract     *MachineLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// MachineLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type MachineLibRaw struct {
	Contract *MachineLib // Generic contract binding to access the raw methods on
}

// MachineLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MachineLibCallerRaw struct {
	Contract *MachineLibCaller // Generic read-only contract binding to access the raw methods on
}

// MachineLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MachineLibTransactorRaw struct {
	Contract *MachineLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMachineLib creates a new instance of MachineLib, bound to a specific deployed contract.
func NewMachineLib(address common.Address, backend bind.ContractBackend) (*MachineLib, error) {
	contract, err := bindMachineLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MachineLib{MachineLibCaller: MachineLibCaller{contract: contract}, MachineLibTransactor: MachineLibTransactor{contract: contract}, MachineLibFilterer: MachineLibFilterer{contract: contract}}, nil
}

// NewMachineLibCaller creates a new read-only instance of MachineLib, bound to a specific deployed contract.
func NewMachineLibCaller(address common.Address, caller bind.ContractCaller) (*MachineLibCaller, error) {
	contract, err := bindMachineLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MachineLibCaller{contract: contract}, nil
}

// NewMachineLibTransactor creates a new write-only instance of MachineLib, bound to a specific deployed contract.
func NewMachineLibTransactor(address common.Address, transactor bind.ContractTransactor) (*MachineLibTransactor, error) {
	contract, err := bindMachineLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MachineLibTransactor{contract: contract}, nil
}

// NewMachineLibFilterer creates a new log filterer instance of MachineLib, bound to a specific deployed contract.
func NewMachineLibFilterer(address common.Address, filterer bind.ContractFilterer) (*MachineLibFilterer, error) {
	contract, err := bindMachineLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MachineLibFilterer{contract: contract}, nil
}

// bindMachineLib binds a generic wrapper to an already deployed contract.
func bindMachineLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MachineLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MachineLib *MachineLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MachineLib.Contract.MachineLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MachineLib *MachineLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MachineLib.Contract.MachineLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MachineLib *MachineLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MachineLib.Contract.MachineLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MachineLib *MachineLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MachineLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MachineLib *MachineLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MachineLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MachineLib *MachineLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MachineLib.Contract.contract.Transact(opts, method, params...)
}

// MerkleProofLibMetaData contains all meta data concerning the MerkleProofLib contract.
var MerkleProofLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220b4f706b98fb8eda361d16ba3527dad53de973ff311a3cb9c18115cd1a3fd231564736f6c63430008110033",
}

// MerkleProofLibABI is the input ABI used to generate the binding from.
// Deprecated: Use MerkleProofLibMetaData.ABI instead.
var MerkleProofLibABI = MerkleProofLibMetaData.ABI

// MerkleProofLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MerkleProofLibMetaData.Bin instead.
var MerkleProofLibBin = MerkleProofLibMetaData.Bin

// DeployMerkleProofLib deploys a new Ethereum contract, binding an instance of MerkleProofLib to it.
func DeployMerkleProofLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MerkleProofLib, error) {
	parsed, err := MerkleProofLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MerkleProofLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MerkleProofLib{MerkleProofLibCaller: MerkleProofLibCaller{contract: contract}, MerkleProofLibTransactor: MerkleProofLibTransactor{contract: contract}, MerkleProofLibFilterer: MerkleProofLibFilterer{contract: contract}}, nil
}

// MerkleProofLib is an auto generated Go binding around an Ethereum contract.
type MerkleProofLib struct {
	MerkleProofLibCaller     // Read-only binding to the contract
	MerkleProofLibTransactor // Write-only binding to the contract
	MerkleProofLibFilterer   // Log filterer for contract events
}

// MerkleProofLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type MerkleProofLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleProofLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MerkleProofLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleProofLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MerkleProofLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleProofLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MerkleProofLibSession struct {
	Contract     *MerkleProofLib   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MerkleProofLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MerkleProofLibCallerSession struct {
	Contract *MerkleProofLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// MerkleProofLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MerkleProofLibTransactorSession struct {
	Contract     *MerkleProofLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// MerkleProofLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type MerkleProofLibRaw struct {
	Contract *MerkleProofLib // Generic contract binding to access the raw methods on
}

// MerkleProofLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MerkleProofLibCallerRaw struct {
	Contract *MerkleProofLibCaller // Generic read-only contract binding to access the raw methods on
}

// MerkleProofLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MerkleProofLibTransactorRaw struct {
	Contract *MerkleProofLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMerkleProofLib creates a new instance of MerkleProofLib, bound to a specific deployed contract.
func NewMerkleProofLib(address common.Address, backend bind.ContractBackend) (*MerkleProofLib, error) {
	contract, err := bindMerkleProofLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MerkleProofLib{MerkleProofLibCaller: MerkleProofLibCaller{contract: contract}, MerkleProofLibTransactor: MerkleProofLibTransactor{contract: contract}, MerkleProofLibFilterer: MerkleProofLibFilterer{contract: contract}}, nil
}

// NewMerkleProofLibCaller creates a new read-only instance of MerkleProofLib, bound to a specific deployed contract.
func NewMerkleProofLibCaller(address common.Address, caller bind.ContractCaller) (*MerkleProofLibCaller, error) {
	contract, err := bindMerkleProofLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MerkleProofLibCaller{contract: contract}, nil
}

// NewMerkleProofLibTransactor creates a new write-only instance of MerkleProofLib, bound to a specific deployed contract.
func NewMerkleProofLibTransactor(address common.Address, transactor bind.ContractTransactor) (*MerkleProofLibTransactor, error) {
	contract, err := bindMerkleProofLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MerkleProofLibTransactor{contract: contract}, nil
}

// NewMerkleProofLibFilterer creates a new log filterer instance of MerkleProofLib, bound to a specific deployed contract.
func NewMerkleProofLibFilterer(address common.Address, filterer bind.ContractFilterer) (*MerkleProofLibFilterer, error) {
	contract, err := bindMerkleProofLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MerkleProofLibFilterer{contract: contract}, nil
}

// bindMerkleProofLib binds a generic wrapper to an already deployed contract.
func bindMerkleProofLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MerkleProofLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MerkleProofLib *MerkleProofLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MerkleProofLib.Contract.MerkleProofLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MerkleProofLib *MerkleProofLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MerkleProofLib.Contract.MerkleProofLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MerkleProofLib *MerkleProofLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MerkleProofLib.Contract.MerkleProofLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MerkleProofLib *MerkleProofLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MerkleProofLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MerkleProofLib *MerkleProofLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MerkleProofLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MerkleProofLib *MerkleProofLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MerkleProofLib.Contract.contract.Transact(opts, method, params...)
}

// ModuleLibMetaData contains all meta data concerning the ModuleLib contract.
var ModuleLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220d61c55edc0f529bb53a0171044606caf812895a4028c5104f008fc104652a1c364736f6c63430008110033",
}

// ModuleLibABI is the input ABI used to generate the binding from.
// Deprecated: Use ModuleLibMetaData.ABI instead.
var ModuleLibABI = ModuleLibMetaData.ABI

// ModuleLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ModuleLibMetaData.Bin instead.
var ModuleLibBin = ModuleLibMetaData.Bin

// DeployModuleLib deploys a new Ethereum contract, binding an instance of ModuleLib to it.
func DeployModuleLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ModuleLib, error) {
	parsed, err := ModuleLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ModuleLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ModuleLib{ModuleLibCaller: ModuleLibCaller{contract: contract}, ModuleLibTransactor: ModuleLibTransactor{contract: contract}, ModuleLibFilterer: ModuleLibFilterer{contract: contract}}, nil
}

// ModuleLib is an auto generated Go binding around an Ethereum contract.
type ModuleLib struct {
	ModuleLibCaller     // Read-only binding to the contract
	ModuleLibTransactor // Write-only binding to the contract
	ModuleLibFilterer   // Log filterer for contract events
}

// ModuleLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type ModuleLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ModuleLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ModuleLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ModuleLibSession struct {
	Contract     *ModuleLib        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ModuleLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ModuleLibCallerSession struct {
	Contract *ModuleLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// ModuleLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ModuleLibTransactorSession struct {
	Contract     *ModuleLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ModuleLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type ModuleLibRaw struct {
	Contract *ModuleLib // Generic contract binding to access the raw methods on
}

// ModuleLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ModuleLibCallerRaw struct {
	Contract *ModuleLibCaller // Generic read-only contract binding to access the raw methods on
}

// ModuleLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ModuleLibTransactorRaw struct {
	Contract *ModuleLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewModuleLib creates a new instance of ModuleLib, bound to a specific deployed contract.
func NewModuleLib(address common.Address, backend bind.ContractBackend) (*ModuleLib, error) {
	contract, err := bindModuleLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ModuleLib{ModuleLibCaller: ModuleLibCaller{contract: contract}, ModuleLibTransactor: ModuleLibTransactor{contract: contract}, ModuleLibFilterer: ModuleLibFilterer{contract: contract}}, nil
}

// NewModuleLibCaller creates a new read-only instance of ModuleLib, bound to a specific deployed contract.
func NewModuleLibCaller(address common.Address, caller bind.ContractCaller) (*ModuleLibCaller, error) {
	contract, err := bindModuleLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ModuleLibCaller{contract: contract}, nil
}

// NewModuleLibTransactor creates a new write-only instance of ModuleLib, bound to a specific deployed contract.
func NewModuleLibTransactor(address common.Address, transactor bind.ContractTransactor) (*ModuleLibTransactor, error) {
	contract, err := bindModuleLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ModuleLibTransactor{contract: contract}, nil
}

// NewModuleLibFilterer creates a new log filterer instance of ModuleLib, bound to a specific deployed contract.
func NewModuleLibFilterer(address common.Address, filterer bind.ContractFilterer) (*ModuleLibFilterer, error) {
	contract, err := bindModuleLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ModuleLibFilterer{contract: contract}, nil
}

// bindModuleLib binds a generic wrapper to an already deployed contract.
func bindModuleLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ModuleLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ModuleLib *ModuleLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ModuleLib.Contract.ModuleLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ModuleLib *ModuleLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ModuleLib.Contract.ModuleLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ModuleLib *ModuleLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ModuleLib.Contract.ModuleLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ModuleLib *ModuleLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ModuleLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ModuleLib *ModuleLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ModuleLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ModuleLib *ModuleLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ModuleLib.Contract.contract.Transact(opts, method, params...)
}

// ModuleMemoryCompactLibMetaData contains all meta data concerning the ModuleMemoryCompactLib contract.
var ModuleMemoryCompactLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212209b3f783462018ab63c3cf93540eab2692b1748a4a0ad3515dd3a13481d4a105464736f6c63430008110033",
}

// ModuleMemoryCompactLibABI is the input ABI used to generate the binding from.
// Deprecated: Use ModuleMemoryCompactLibMetaData.ABI instead.
var ModuleMemoryCompactLibABI = ModuleMemoryCompactLibMetaData.ABI

// ModuleMemoryCompactLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ModuleMemoryCompactLibMetaData.Bin instead.
var ModuleMemoryCompactLibBin = ModuleMemoryCompactLibMetaData.Bin

// DeployModuleMemoryCompactLib deploys a new Ethereum contract, binding an instance of ModuleMemoryCompactLib to it.
func DeployModuleMemoryCompactLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ModuleMemoryCompactLib, error) {
	parsed, err := ModuleMemoryCompactLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ModuleMemoryCompactLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ModuleMemoryCompactLib{ModuleMemoryCompactLibCaller: ModuleMemoryCompactLibCaller{contract: contract}, ModuleMemoryCompactLibTransactor: ModuleMemoryCompactLibTransactor{contract: contract}, ModuleMemoryCompactLibFilterer: ModuleMemoryCompactLibFilterer{contract: contract}}, nil
}

// ModuleMemoryCompactLib is an auto generated Go binding around an Ethereum contract.
type ModuleMemoryCompactLib struct {
	ModuleMemoryCompactLibCaller     // Read-only binding to the contract
	ModuleMemoryCompactLibTransactor // Write-only binding to the contract
	ModuleMemoryCompactLibFilterer   // Log filterer for contract events
}

// ModuleMemoryCompactLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type ModuleMemoryCompactLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleMemoryCompactLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ModuleMemoryCompactLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleMemoryCompactLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ModuleMemoryCompactLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleMemoryCompactLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ModuleMemoryCompactLibSession struct {
	Contract     *ModuleMemoryCompactLib // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// ModuleMemoryCompactLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ModuleMemoryCompactLibCallerSession struct {
	Contract *ModuleMemoryCompactLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// ModuleMemoryCompactLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ModuleMemoryCompactLibTransactorSession struct {
	Contract     *ModuleMemoryCompactLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// ModuleMemoryCompactLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type ModuleMemoryCompactLibRaw struct {
	Contract *ModuleMemoryCompactLib // Generic contract binding to access the raw methods on
}

// ModuleMemoryCompactLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ModuleMemoryCompactLibCallerRaw struct {
	Contract *ModuleMemoryCompactLibCaller // Generic read-only contract binding to access the raw methods on
}

// ModuleMemoryCompactLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ModuleMemoryCompactLibTransactorRaw struct {
	Contract *ModuleMemoryCompactLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewModuleMemoryCompactLib creates a new instance of ModuleMemoryCompactLib, bound to a specific deployed contract.
func NewModuleMemoryCompactLib(address common.Address, backend bind.ContractBackend) (*ModuleMemoryCompactLib, error) {
	contract, err := bindModuleMemoryCompactLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ModuleMemoryCompactLib{ModuleMemoryCompactLibCaller: ModuleMemoryCompactLibCaller{contract: contract}, ModuleMemoryCompactLibTransactor: ModuleMemoryCompactLibTransactor{contract: contract}, ModuleMemoryCompactLibFilterer: ModuleMemoryCompactLibFilterer{contract: contract}}, nil
}

// NewModuleMemoryCompactLibCaller creates a new read-only instance of ModuleMemoryCompactLib, bound to a specific deployed contract.
func NewModuleMemoryCompactLibCaller(address common.Address, caller bind.ContractCaller) (*ModuleMemoryCompactLibCaller, error) {
	contract, err := bindModuleMemoryCompactLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ModuleMemoryCompactLibCaller{contract: contract}, nil
}

// NewModuleMemoryCompactLibTransactor creates a new write-only instance of ModuleMemoryCompactLib, bound to a specific deployed contract.
func NewModuleMemoryCompactLibTransactor(address common.Address, transactor bind.ContractTransactor) (*ModuleMemoryCompactLibTransactor, error) {
	contract, err := bindModuleMemoryCompactLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ModuleMemoryCompactLibTransactor{contract: contract}, nil
}

// NewModuleMemoryCompactLibFilterer creates a new log filterer instance of ModuleMemoryCompactLib, bound to a specific deployed contract.
func NewModuleMemoryCompactLibFilterer(address common.Address, filterer bind.ContractFilterer) (*ModuleMemoryCompactLibFilterer, error) {
	contract, err := bindModuleMemoryCompactLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ModuleMemoryCompactLibFilterer{contract: contract}, nil
}

// bindModuleMemoryCompactLib binds a generic wrapper to an already deployed contract.
func bindModuleMemoryCompactLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ModuleMemoryCompactLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ModuleMemoryCompactLib *ModuleMemoryCompactLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ModuleMemoryCompactLib.Contract.ModuleMemoryCompactLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ModuleMemoryCompactLib *ModuleMemoryCompactLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ModuleMemoryCompactLib.Contract.ModuleMemoryCompactLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ModuleMemoryCompactLib *ModuleMemoryCompactLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ModuleMemoryCompactLib.Contract.ModuleMemoryCompactLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ModuleMemoryCompactLib *ModuleMemoryCompactLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ModuleMemoryCompactLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ModuleMemoryCompactLib *ModuleMemoryCompactLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ModuleMemoryCompactLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ModuleMemoryCompactLib *ModuleMemoryCompactLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ModuleMemoryCompactLib.Contract.contract.Transact(opts, method, params...)
}

// ModuleMemoryLibMetaData contains all meta data concerning the ModuleMemoryLib contract.
var ModuleMemoryLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212207cccfb717452c5963f76e7f1cba1db14fafca285a9525dfb2016affa0a2ea06664736f6c63430008110033",
}

// ModuleMemoryLibABI is the input ABI used to generate the binding from.
// Deprecated: Use ModuleMemoryLibMetaData.ABI instead.
var ModuleMemoryLibABI = ModuleMemoryLibMetaData.ABI

// ModuleMemoryLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ModuleMemoryLibMetaData.Bin instead.
var ModuleMemoryLibBin = ModuleMemoryLibMetaData.Bin

// DeployModuleMemoryLib deploys a new Ethereum contract, binding an instance of ModuleMemoryLib to it.
func DeployModuleMemoryLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ModuleMemoryLib, error) {
	parsed, err := ModuleMemoryLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ModuleMemoryLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ModuleMemoryLib{ModuleMemoryLibCaller: ModuleMemoryLibCaller{contract: contract}, ModuleMemoryLibTransactor: ModuleMemoryLibTransactor{contract: contract}, ModuleMemoryLibFilterer: ModuleMemoryLibFilterer{contract: contract}}, nil
}

// ModuleMemoryLib is an auto generated Go binding around an Ethereum contract.
type ModuleMemoryLib struct {
	ModuleMemoryLibCaller     // Read-only binding to the contract
	ModuleMemoryLibTransactor // Write-only binding to the contract
	ModuleMemoryLibFilterer   // Log filterer for contract events
}

// ModuleMemoryLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type ModuleMemoryLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleMemoryLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ModuleMemoryLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleMemoryLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ModuleMemoryLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ModuleMemoryLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ModuleMemoryLibSession struct {
	Contract     *ModuleMemoryLib  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ModuleMemoryLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ModuleMemoryLibCallerSession struct {
	Contract *ModuleMemoryLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// ModuleMemoryLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ModuleMemoryLibTransactorSession struct {
	Contract     *ModuleMemoryLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// ModuleMemoryLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type ModuleMemoryLibRaw struct {
	Contract *ModuleMemoryLib // Generic contract binding to access the raw methods on
}

// ModuleMemoryLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ModuleMemoryLibCallerRaw struct {
	Contract *ModuleMemoryLibCaller // Generic read-only contract binding to access the raw methods on
}

// ModuleMemoryLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ModuleMemoryLibTransactorRaw struct {
	Contract *ModuleMemoryLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewModuleMemoryLib creates a new instance of ModuleMemoryLib, bound to a specific deployed contract.
func NewModuleMemoryLib(address common.Address, backend bind.ContractBackend) (*ModuleMemoryLib, error) {
	contract, err := bindModuleMemoryLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ModuleMemoryLib{ModuleMemoryLibCaller: ModuleMemoryLibCaller{contract: contract}, ModuleMemoryLibTransactor: ModuleMemoryLibTransactor{contract: contract}, ModuleMemoryLibFilterer: ModuleMemoryLibFilterer{contract: contract}}, nil
}

// NewModuleMemoryLibCaller creates a new read-only instance of ModuleMemoryLib, bound to a specific deployed contract.
func NewModuleMemoryLibCaller(address common.Address, caller bind.ContractCaller) (*ModuleMemoryLibCaller, error) {
	contract, err := bindModuleMemoryLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ModuleMemoryLibCaller{contract: contract}, nil
}

// NewModuleMemoryLibTransactor creates a new write-only instance of ModuleMemoryLib, bound to a specific deployed contract.
func NewModuleMemoryLibTransactor(address common.Address, transactor bind.ContractTransactor) (*ModuleMemoryLibTransactor, error) {
	contract, err := bindModuleMemoryLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ModuleMemoryLibTransactor{contract: contract}, nil
}

// NewModuleMemoryLibFilterer creates a new log filterer instance of ModuleMemoryLib, bound to a specific deployed contract.
func NewModuleMemoryLibFilterer(address common.Address, filterer bind.ContractFilterer) (*ModuleMemoryLibFilterer, error) {
	contract, err := bindModuleMemoryLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ModuleMemoryLibFilterer{contract: contract}, nil
}

// bindModuleMemoryLib binds a generic wrapper to an already deployed contract.
func bindModuleMemoryLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ModuleMemoryLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ModuleMemoryLib *ModuleMemoryLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ModuleMemoryLib.Contract.ModuleMemoryLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ModuleMemoryLib *ModuleMemoryLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ModuleMemoryLib.Contract.ModuleMemoryLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ModuleMemoryLib *ModuleMemoryLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ModuleMemoryLib.Contract.ModuleMemoryLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ModuleMemoryLib *ModuleMemoryLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ModuleMemoryLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ModuleMemoryLib *ModuleMemoryLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ModuleMemoryLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ModuleMemoryLib *ModuleMemoryLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ModuleMemoryLib.Contract.contract.Transact(opts, method, params...)
}

// MultiStackLibMetaData contains all meta data concerning the MultiStackLib contract.
var MultiStackLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212205ffbd2933f4af71722c4cb15f5f37de90673d9b9e065daaa3fdcb973103d7d4964736f6c63430008110033",
}

// MultiStackLibABI is the input ABI used to generate the binding from.
// Deprecated: Use MultiStackLibMetaData.ABI instead.
var MultiStackLibABI = MultiStackLibMetaData.ABI

// MultiStackLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MultiStackLibMetaData.Bin instead.
var MultiStackLibBin = MultiStackLibMetaData.Bin

// DeployMultiStackLib deploys a new Ethereum contract, binding an instance of MultiStackLib to it.
func DeployMultiStackLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MultiStackLib, error) {
	parsed, err := MultiStackLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MultiStackLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MultiStackLib{MultiStackLibCaller: MultiStackLibCaller{contract: contract}, MultiStackLibTransactor: MultiStackLibTransactor{contract: contract}, MultiStackLibFilterer: MultiStackLibFilterer{contract: contract}}, nil
}

// MultiStackLib is an auto generated Go binding around an Ethereum contract.
type MultiStackLib struct {
	MultiStackLibCaller     // Read-only binding to the contract
	MultiStackLibTransactor // Write-only binding to the contract
	MultiStackLibFilterer   // Log filterer for contract events
}

// MultiStackLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type MultiStackLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiStackLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MultiStackLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiStackLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MultiStackLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiStackLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MultiStackLibSession struct {
	Contract     *MultiStackLib    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MultiStackLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MultiStackLibCallerSession struct {
	Contract *MultiStackLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// MultiStackLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MultiStackLibTransactorSession struct {
	Contract     *MultiStackLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// MultiStackLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type MultiStackLibRaw struct {
	Contract *MultiStackLib // Generic contract binding to access the raw methods on
}

// MultiStackLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MultiStackLibCallerRaw struct {
	Contract *MultiStackLibCaller // Generic read-only contract binding to access the raw methods on
}

// MultiStackLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MultiStackLibTransactorRaw struct {
	Contract *MultiStackLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMultiStackLib creates a new instance of MultiStackLib, bound to a specific deployed contract.
func NewMultiStackLib(address common.Address, backend bind.ContractBackend) (*MultiStackLib, error) {
	contract, err := bindMultiStackLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MultiStackLib{MultiStackLibCaller: MultiStackLibCaller{contract: contract}, MultiStackLibTransactor: MultiStackLibTransactor{contract: contract}, MultiStackLibFilterer: MultiStackLibFilterer{contract: contract}}, nil
}

// NewMultiStackLibCaller creates a new read-only instance of MultiStackLib, bound to a specific deployed contract.
func NewMultiStackLibCaller(address common.Address, caller bind.ContractCaller) (*MultiStackLibCaller, error) {
	contract, err := bindMultiStackLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MultiStackLibCaller{contract: contract}, nil
}

// NewMultiStackLibTransactor creates a new write-only instance of MultiStackLib, bound to a specific deployed contract.
func NewMultiStackLibTransactor(address common.Address, transactor bind.ContractTransactor) (*MultiStackLibTransactor, error) {
	contract, err := bindMultiStackLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MultiStackLibTransactor{contract: contract}, nil
}

// NewMultiStackLibFilterer creates a new log filterer instance of MultiStackLib, bound to a specific deployed contract.
func NewMultiStackLibFilterer(address common.Address, filterer bind.ContractFilterer) (*MultiStackLibFilterer, error) {
	contract, err := bindMultiStackLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MultiStackLibFilterer{contract: contract}, nil
}

// bindMultiStackLib binds a generic wrapper to an already deployed contract.
func bindMultiStackLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MultiStackLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiStackLib *MultiStackLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiStackLib.Contract.MultiStackLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiStackLib *MultiStackLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiStackLib.Contract.MultiStackLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiStackLib *MultiStackLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiStackLib.Contract.MultiStackLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiStackLib *MultiStackLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiStackLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiStackLib *MultiStackLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiStackLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiStackLib *MultiStackLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiStackLib.Contract.contract.Transact(opts, method, params...)
}

// PcArrayLibMetaData contains all meta data concerning the PcArrayLib contract.
var PcArrayLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220e3dc270b6b90bfde6651dcd425c63985ef40e8fd37e14a5c513d95ce2d8d927364736f6c63430008110033",
}

// PcArrayLibABI is the input ABI used to generate the binding from.
// Deprecated: Use PcArrayLibMetaData.ABI instead.
var PcArrayLibABI = PcArrayLibMetaData.ABI

// PcArrayLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use PcArrayLibMetaData.Bin instead.
var PcArrayLibBin = PcArrayLibMetaData.Bin

// DeployPcArrayLib deploys a new Ethereum contract, binding an instance of PcArrayLib to it.
func DeployPcArrayLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *PcArrayLib, error) {
	parsed, err := PcArrayLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PcArrayLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PcArrayLib{PcArrayLibCaller: PcArrayLibCaller{contract: contract}, PcArrayLibTransactor: PcArrayLibTransactor{contract: contract}, PcArrayLibFilterer: PcArrayLibFilterer{contract: contract}}, nil
}

// PcArrayLib is an auto generated Go binding around an Ethereum contract.
type PcArrayLib struct {
	PcArrayLibCaller     // Read-only binding to the contract
	PcArrayLibTransactor // Write-only binding to the contract
	PcArrayLibFilterer   // Log filterer for contract events
}

// PcArrayLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type PcArrayLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PcArrayLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PcArrayLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PcArrayLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PcArrayLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PcArrayLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PcArrayLibSession struct {
	Contract     *PcArrayLib       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PcArrayLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PcArrayLibCallerSession struct {
	Contract *PcArrayLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// PcArrayLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PcArrayLibTransactorSession struct {
	Contract     *PcArrayLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// PcArrayLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type PcArrayLibRaw struct {
	Contract *PcArrayLib // Generic contract binding to access the raw methods on
}

// PcArrayLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PcArrayLibCallerRaw struct {
	Contract *PcArrayLibCaller // Generic read-only contract binding to access the raw methods on
}

// PcArrayLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PcArrayLibTransactorRaw struct {
	Contract *PcArrayLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPcArrayLib creates a new instance of PcArrayLib, bound to a specific deployed contract.
func NewPcArrayLib(address common.Address, backend bind.ContractBackend) (*PcArrayLib, error) {
	contract, err := bindPcArrayLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PcArrayLib{PcArrayLibCaller: PcArrayLibCaller{contract: contract}, PcArrayLibTransactor: PcArrayLibTransactor{contract: contract}, PcArrayLibFilterer: PcArrayLibFilterer{contract: contract}}, nil
}

// NewPcArrayLibCaller creates a new read-only instance of PcArrayLib, bound to a specific deployed contract.
func NewPcArrayLibCaller(address common.Address, caller bind.ContractCaller) (*PcArrayLibCaller, error) {
	contract, err := bindPcArrayLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PcArrayLibCaller{contract: contract}, nil
}

// NewPcArrayLibTransactor creates a new write-only instance of PcArrayLib, bound to a specific deployed contract.
func NewPcArrayLibTransactor(address common.Address, transactor bind.ContractTransactor) (*PcArrayLibTransactor, error) {
	contract, err := bindPcArrayLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PcArrayLibTransactor{contract: contract}, nil
}

// NewPcArrayLibFilterer creates a new log filterer instance of PcArrayLib, bound to a specific deployed contract.
func NewPcArrayLibFilterer(address common.Address, filterer bind.ContractFilterer) (*PcArrayLibFilterer, error) {
	contract, err := bindPcArrayLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PcArrayLibFilterer{contract: contract}, nil
}

// bindPcArrayLib binds a generic wrapper to an already deployed contract.
func bindPcArrayLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PcArrayLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PcArrayLib *PcArrayLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PcArrayLib.Contract.PcArrayLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PcArrayLib *PcArrayLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PcArrayLib.Contract.PcArrayLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PcArrayLib *PcArrayLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PcArrayLib.Contract.PcArrayLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PcArrayLib *PcArrayLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PcArrayLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PcArrayLib *PcArrayLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PcArrayLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PcArrayLib *PcArrayLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PcArrayLib.Contract.contract.Transact(opts, method, params...)
}

// StackFrameLibMetaData contains all meta data concerning the StackFrameLib contract.
var StackFrameLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212203c536423870080be6a01a5bfb1af2e74ed8610b80394b39f75d1947d16201bad64736f6c63430008110033",
}

// StackFrameLibABI is the input ABI used to generate the binding from.
// Deprecated: Use StackFrameLibMetaData.ABI instead.
var StackFrameLibABI = StackFrameLibMetaData.ABI

// StackFrameLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use StackFrameLibMetaData.Bin instead.
var StackFrameLibBin = StackFrameLibMetaData.Bin

// DeployStackFrameLib deploys a new Ethereum contract, binding an instance of StackFrameLib to it.
func DeployStackFrameLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *StackFrameLib, error) {
	parsed, err := StackFrameLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(StackFrameLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &StackFrameLib{StackFrameLibCaller: StackFrameLibCaller{contract: contract}, StackFrameLibTransactor: StackFrameLibTransactor{contract: contract}, StackFrameLibFilterer: StackFrameLibFilterer{contract: contract}}, nil
}

// StackFrameLib is an auto generated Go binding around an Ethereum contract.
type StackFrameLib struct {
	StackFrameLibCaller     // Read-only binding to the contract
	StackFrameLibTransactor // Write-only binding to the contract
	StackFrameLibFilterer   // Log filterer for contract events
}

// StackFrameLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type StackFrameLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StackFrameLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StackFrameLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StackFrameLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StackFrameLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StackFrameLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StackFrameLibSession struct {
	Contract     *StackFrameLib    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StackFrameLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StackFrameLibCallerSession struct {
	Contract *StackFrameLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// StackFrameLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StackFrameLibTransactorSession struct {
	Contract     *StackFrameLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// StackFrameLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type StackFrameLibRaw struct {
	Contract *StackFrameLib // Generic contract binding to access the raw methods on
}

// StackFrameLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StackFrameLibCallerRaw struct {
	Contract *StackFrameLibCaller // Generic read-only contract binding to access the raw methods on
}

// StackFrameLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StackFrameLibTransactorRaw struct {
	Contract *StackFrameLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStackFrameLib creates a new instance of StackFrameLib, bound to a specific deployed contract.
func NewStackFrameLib(address common.Address, backend bind.ContractBackend) (*StackFrameLib, error) {
	contract, err := bindStackFrameLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &StackFrameLib{StackFrameLibCaller: StackFrameLibCaller{contract: contract}, StackFrameLibTransactor: StackFrameLibTransactor{contract: contract}, StackFrameLibFilterer: StackFrameLibFilterer{contract: contract}}, nil
}

// NewStackFrameLibCaller creates a new read-only instance of StackFrameLib, bound to a specific deployed contract.
func NewStackFrameLibCaller(address common.Address, caller bind.ContractCaller) (*StackFrameLibCaller, error) {
	contract, err := bindStackFrameLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StackFrameLibCaller{contract: contract}, nil
}

// NewStackFrameLibTransactor creates a new write-only instance of StackFrameLib, bound to a specific deployed contract.
func NewStackFrameLibTransactor(address common.Address, transactor bind.ContractTransactor) (*StackFrameLibTransactor, error) {
	contract, err := bindStackFrameLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StackFrameLibTransactor{contract: contract}, nil
}

// NewStackFrameLibFilterer creates a new log filterer instance of StackFrameLib, bound to a specific deployed contract.
func NewStackFrameLibFilterer(address common.Address, filterer bind.ContractFilterer) (*StackFrameLibFilterer, error) {
	contract, err := bindStackFrameLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StackFrameLibFilterer{contract: contract}, nil
}

// bindStackFrameLib binds a generic wrapper to an already deployed contract.
func bindStackFrameLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StackFrameLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StackFrameLib *StackFrameLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StackFrameLib.Contract.StackFrameLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StackFrameLib *StackFrameLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StackFrameLib.Contract.StackFrameLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StackFrameLib *StackFrameLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StackFrameLib.Contract.StackFrameLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StackFrameLib *StackFrameLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StackFrameLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StackFrameLib *StackFrameLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StackFrameLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StackFrameLib *StackFrameLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StackFrameLib.Contract.contract.Transact(opts, method, params...)
}

// ValueArrayLibMetaData contains all meta data concerning the ValueArrayLib contract.
var ValueArrayLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212207fef79a6782773361e287f346ea7724af9ddf32f9df51c985f5e692c761a77d164736f6c63430008110033",
}

// ValueArrayLibABI is the input ABI used to generate the binding from.
// Deprecated: Use ValueArrayLibMetaData.ABI instead.
var ValueArrayLibABI = ValueArrayLibMetaData.ABI

// ValueArrayLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ValueArrayLibMetaData.Bin instead.
var ValueArrayLibBin = ValueArrayLibMetaData.Bin

// DeployValueArrayLib deploys a new Ethereum contract, binding an instance of ValueArrayLib to it.
func DeployValueArrayLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ValueArrayLib, error) {
	parsed, err := ValueArrayLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ValueArrayLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ValueArrayLib{ValueArrayLibCaller: ValueArrayLibCaller{contract: contract}, ValueArrayLibTransactor: ValueArrayLibTransactor{contract: contract}, ValueArrayLibFilterer: ValueArrayLibFilterer{contract: contract}}, nil
}

// ValueArrayLib is an auto generated Go binding around an Ethereum contract.
type ValueArrayLib struct {
	ValueArrayLibCaller     // Read-only binding to the contract
	ValueArrayLibTransactor // Write-only binding to the contract
	ValueArrayLibFilterer   // Log filterer for contract events
}

// ValueArrayLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type ValueArrayLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueArrayLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ValueArrayLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueArrayLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ValueArrayLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueArrayLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ValueArrayLibSession struct {
	Contract     *ValueArrayLib    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ValueArrayLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ValueArrayLibCallerSession struct {
	Contract *ValueArrayLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ValueArrayLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ValueArrayLibTransactorSession struct {
	Contract     *ValueArrayLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ValueArrayLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type ValueArrayLibRaw struct {
	Contract *ValueArrayLib // Generic contract binding to access the raw methods on
}

// ValueArrayLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ValueArrayLibCallerRaw struct {
	Contract *ValueArrayLibCaller // Generic read-only contract binding to access the raw methods on
}

// ValueArrayLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ValueArrayLibTransactorRaw struct {
	Contract *ValueArrayLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewValueArrayLib creates a new instance of ValueArrayLib, bound to a specific deployed contract.
func NewValueArrayLib(address common.Address, backend bind.ContractBackend) (*ValueArrayLib, error) {
	contract, err := bindValueArrayLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ValueArrayLib{ValueArrayLibCaller: ValueArrayLibCaller{contract: contract}, ValueArrayLibTransactor: ValueArrayLibTransactor{contract: contract}, ValueArrayLibFilterer: ValueArrayLibFilterer{contract: contract}}, nil
}

// NewValueArrayLibCaller creates a new read-only instance of ValueArrayLib, bound to a specific deployed contract.
func NewValueArrayLibCaller(address common.Address, caller bind.ContractCaller) (*ValueArrayLibCaller, error) {
	contract, err := bindValueArrayLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ValueArrayLibCaller{contract: contract}, nil
}

// NewValueArrayLibTransactor creates a new write-only instance of ValueArrayLib, bound to a specific deployed contract.
func NewValueArrayLibTransactor(address common.Address, transactor bind.ContractTransactor) (*ValueArrayLibTransactor, error) {
	contract, err := bindValueArrayLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ValueArrayLibTransactor{contract: contract}, nil
}

// NewValueArrayLibFilterer creates a new log filterer instance of ValueArrayLib, bound to a specific deployed contract.
func NewValueArrayLibFilterer(address common.Address, filterer bind.ContractFilterer) (*ValueArrayLibFilterer, error) {
	contract, err := bindValueArrayLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ValueArrayLibFilterer{contract: contract}, nil
}

// bindValueArrayLib binds a generic wrapper to an already deployed contract.
func bindValueArrayLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ValueArrayLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValueArrayLib *ValueArrayLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValueArrayLib.Contract.ValueArrayLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValueArrayLib *ValueArrayLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValueArrayLib.Contract.ValueArrayLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValueArrayLib *ValueArrayLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValueArrayLib.Contract.ValueArrayLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValueArrayLib *ValueArrayLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValueArrayLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValueArrayLib *ValueArrayLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValueArrayLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValueArrayLib *ValueArrayLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValueArrayLib.Contract.contract.Transact(opts, method, params...)
}

// ValueLibMetaData contains all meta data concerning the ValueLib contract.
var ValueLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212206f8b156d50ca622cf4d5f8a312657be6b72c883de3034690dcffe1ed145cbeb664736f6c63430008110033",
}

// ValueLibABI is the input ABI used to generate the binding from.
// Deprecated: Use ValueLibMetaData.ABI instead.
var ValueLibABI = ValueLibMetaData.ABI

// ValueLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ValueLibMetaData.Bin instead.
var ValueLibBin = ValueLibMetaData.Bin

// DeployValueLib deploys a new Ethereum contract, binding an instance of ValueLib to it.
func DeployValueLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ValueLib, error) {
	parsed, err := ValueLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ValueLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ValueLib{ValueLibCaller: ValueLibCaller{contract: contract}, ValueLibTransactor: ValueLibTransactor{contract: contract}, ValueLibFilterer: ValueLibFilterer{contract: contract}}, nil
}

// ValueLib is an auto generated Go binding around an Ethereum contract.
type ValueLib struct {
	ValueLibCaller     // Read-only binding to the contract
	ValueLibTransactor // Write-only binding to the contract
	ValueLibFilterer   // Log filterer for contract events
}

// ValueLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type ValueLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ValueLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ValueLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ValueLibSession struct {
	Contract     *ValueLib         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ValueLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ValueLibCallerSession struct {
	Contract *ValueLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ValueLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ValueLibTransactorSession struct {
	Contract     *ValueLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ValueLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type ValueLibRaw struct {
	Contract *ValueLib // Generic contract binding to access the raw methods on
}

// ValueLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ValueLibCallerRaw struct {
	Contract *ValueLibCaller // Generic read-only contract binding to access the raw methods on
}

// ValueLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ValueLibTransactorRaw struct {
	Contract *ValueLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewValueLib creates a new instance of ValueLib, bound to a specific deployed contract.
func NewValueLib(address common.Address, backend bind.ContractBackend) (*ValueLib, error) {
	contract, err := bindValueLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ValueLib{ValueLibCaller: ValueLibCaller{contract: contract}, ValueLibTransactor: ValueLibTransactor{contract: contract}, ValueLibFilterer: ValueLibFilterer{contract: contract}}, nil
}

// NewValueLibCaller creates a new read-only instance of ValueLib, bound to a specific deployed contract.
func NewValueLibCaller(address common.Address, caller bind.ContractCaller) (*ValueLibCaller, error) {
	contract, err := bindValueLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ValueLibCaller{contract: contract}, nil
}

// NewValueLibTransactor creates a new write-only instance of ValueLib, bound to a specific deployed contract.
func NewValueLibTransactor(address common.Address, transactor bind.ContractTransactor) (*ValueLibTransactor, error) {
	contract, err := bindValueLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ValueLibTransactor{contract: contract}, nil
}

// NewValueLibFilterer creates a new log filterer instance of ValueLib, bound to a specific deployed contract.
func NewValueLibFilterer(address common.Address, filterer bind.ContractFilterer) (*ValueLibFilterer, error) {
	contract, err := bindValueLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ValueLibFilterer{contract: contract}, nil
}

// bindValueLib binds a generic wrapper to an already deployed contract.
func bindValueLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ValueLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValueLib *ValueLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValueLib.Contract.ValueLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValueLib *ValueLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValueLib.Contract.ValueLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValueLib *ValueLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValueLib.Contract.ValueLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValueLib *ValueLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValueLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValueLib *ValueLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValueLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValueLib *ValueLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValueLib.Contract.contract.Transact(opts, method, params...)
}

// ValueStackLibMetaData contains all meta data concerning the ValueStackLib contract.
var ValueStackLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212200549d9e927d4e3e31c9f826b907e243dc2bc3f486c1bf899468099ed9ec19e5564736f6c63430008110033",
}

// ValueStackLibABI is the input ABI used to generate the binding from.
// Deprecated: Use ValueStackLibMetaData.ABI instead.
var ValueStackLibABI = ValueStackLibMetaData.ABI

// ValueStackLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ValueStackLibMetaData.Bin instead.
var ValueStackLibBin = ValueStackLibMetaData.Bin

// DeployValueStackLib deploys a new Ethereum contract, binding an instance of ValueStackLib to it.
func DeployValueStackLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ValueStackLib, error) {
	parsed, err := ValueStackLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ValueStackLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ValueStackLib{ValueStackLibCaller: ValueStackLibCaller{contract: contract}, ValueStackLibTransactor: ValueStackLibTransactor{contract: contract}, ValueStackLibFilterer: ValueStackLibFilterer{contract: contract}}, nil
}

// ValueStackLib is an auto generated Go binding around an Ethereum contract.
type ValueStackLib struct {
	ValueStackLibCaller     // Read-only binding to the contract
	ValueStackLibTransactor // Write-only binding to the contract
	ValueStackLibFilterer   // Log filterer for contract events
}

// ValueStackLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type ValueStackLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueStackLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ValueStackLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueStackLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ValueStackLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueStackLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ValueStackLibSession struct {
	Contract     *ValueStackLib    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ValueStackLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ValueStackLibCallerSession struct {
	Contract *ValueStackLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ValueStackLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ValueStackLibTransactorSession struct {
	Contract     *ValueStackLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ValueStackLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type ValueStackLibRaw struct {
	Contract *ValueStackLib // Generic contract binding to access the raw methods on
}

// ValueStackLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ValueStackLibCallerRaw struct {
	Contract *ValueStackLibCaller // Generic read-only contract binding to access the raw methods on
}

// ValueStackLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ValueStackLibTransactorRaw struct {
	Contract *ValueStackLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewValueStackLib creates a new instance of ValueStackLib, bound to a specific deployed contract.
func NewValueStackLib(address common.Address, backend bind.ContractBackend) (*ValueStackLib, error) {
	contract, err := bindValueStackLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ValueStackLib{ValueStackLibCaller: ValueStackLibCaller{contract: contract}, ValueStackLibTransactor: ValueStackLibTransactor{contract: contract}, ValueStackLibFilterer: ValueStackLibFilterer{contract: contract}}, nil
}

// NewValueStackLibCaller creates a new read-only instance of ValueStackLib, bound to a specific deployed contract.
func NewValueStackLibCaller(address common.Address, caller bind.ContractCaller) (*ValueStackLibCaller, error) {
	contract, err := bindValueStackLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ValueStackLibCaller{contract: contract}, nil
}

// NewValueStackLibTransactor creates a new write-only instance of ValueStackLib, bound to a specific deployed contract.
func NewValueStackLibTransactor(address common.Address, transactor bind.ContractTransactor) (*ValueStackLibTransactor, error) {
	contract, err := bindValueStackLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ValueStackLibTransactor{contract: contract}, nil
}

// NewValueStackLibFilterer creates a new log filterer instance of ValueStackLib, bound to a specific deployed contract.
func NewValueStackLibFilterer(address common.Address, filterer bind.ContractFilterer) (*ValueStackLibFilterer, error) {
	contract, err := bindValueStackLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ValueStackLibFilterer{contract: contract}, nil
}

// bindValueStackLib binds a generic wrapper to an already deployed contract.
func bindValueStackLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ValueStackLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValueStackLib *ValueStackLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValueStackLib.Contract.ValueStackLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValueStackLib *ValueStackLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValueStackLib.Contract.ValueStackLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValueStackLib *ValueStackLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValueStackLib.Contract.ValueStackLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValueStackLib *ValueStackLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValueStackLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValueStackLib *ValueStackLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValueStackLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValueStackLib *ValueStackLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValueStackLib.Contract.contract.Transact(opts, method, params...)
}
