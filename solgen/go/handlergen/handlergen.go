// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package handlergen

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

// CompatibilityFallbackHandlerMetaData contains all meta data concerning the CompatibilityFallbackHandler contract.
var CompatibilityFallbackHandlerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractSafe\",\"name\":\"safe\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"encodeMessageDataForSafe\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"getMessageHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractSafe\",\"name\":\"safe\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"getMessageHashForSafe\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getModules\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_dataHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"_signature\",\"type\":\"bytes\"}],\"name\":\"isValidSignature\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_signature\",\"type\":\"bytes\"}],\"name\":\"isValidSignature\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"onERC1155BatchReceived\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"onERC1155Received\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"onERC721Received\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"targetContract\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"calldataPayload\",\"type\":\"bytes\"}],\"name\":\"simulate\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"response\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"tokensReceived\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611605806100206000396000f3fe608060405234801561001057600080fd5b50600436106100b35760003560e01c8063230316401161007157806323031640146106535780636ac24784146107a7578063b2494df314610896578063bc197c81146108f5578063bd61951d14610a8b578063f23a6e6114610b9d576100b3565b806223de29146100b857806301ffc9a7146101f05780630a1028c414610253578063150b7a02146103225780631626ba7e1461041857806320c13b0b146104ce575b600080fd5b6101ee600480360360c08110156100ce57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291908035906020019064010000000081111561015557600080fd5b82018360208201111561016757600080fd5b8035906020019184600183028401116401000000008311171561018957600080fd5b9091929391929390803590602001906401000000008111156101aa57600080fd5b8201836020820111156101bc57600080fd5b803590602001918460018302840111640100000000831117156101de57600080fd5b9091929391929390505050610c9d565b005b61023b6004803603602081101561020657600080fd5b8101908080357bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19169060200190929190505050610ca7565b60405180821515815260200191505060405180910390f35b61030c6004803603602081101561026957600080fd5b810190808035906020019064010000000081111561028657600080fd5b82018360208201111561029857600080fd5b803590602001918460018302840111640100000000831117156102ba57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050509192919290505050610de1565b6040518082815260200191505060405180910390f35b6103e36004803603608081101561033857600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291908035906020019064010000000081111561039f57600080fd5b8201836020820111156103b157600080fd5b803590602001918460018302840111640100000000831117156103d357600080fd5b9091929391929390505050610df4565b60405180827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200191505060405180910390f35b6104996004803603604081101561042e57600080fd5b81019080803590602001909291908035906020019064010000000081111561045557600080fd5b82018360208201111561046757600080fd5b8035906020019184600183028401116401000000008311171561048957600080fd5b9091929391929390505050610e09565b60405180827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200191505060405180910390f35b61061e600480360360408110156104e457600080fd5b810190808035906020019064010000000081111561050157600080fd5b82018360208201111561051357600080fd5b8035906020019184600183028401116401000000008311171561053557600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192908035906020019064010000000081111561059857600080fd5b8201836020820111156105aa57600080fd5b803590602001918460018302840111640100000000831117156105cc57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050509192919290505050610fc1565b60405180827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200191505060405180910390f35b61072c6004803603604081101561066957600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001906401000000008111156106a657600080fd5b8201836020820111156106b857600080fd5b803590602001918460018302840111640100000000831117156106da57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050509192919290505050611249565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561076c578082015181840152602081019050610751565b50505050905090810190601f1680156107995780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b610880600480360360408110156107bd57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001906401000000008111156107fa57600080fd5b82018360208201111561080c57600080fd5b8035906020019184600183028401116401000000008311171561082e57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192905050506113b5565b6040518082815260200191505060405180910390f35b61089e6113d0565b6040518080602001828103825283818151815260200191508051906020019060200280838360005b838110156108e15780820151818401526020810190506108c6565b505050509050019250505060405180910390f35b610a56600480360360a081101561090b57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019064010000000081111561096857600080fd5b82018360208201111561097a57600080fd5b8035906020019184602083028401116401000000008311171561099c57600080fd5b9091929391929390803590602001906401000000008111156109bd57600080fd5b8201836020820111156109cf57600080fd5b803590602001918460208302840111640100000000831117156109f157600080fd5b909192939192939080359060200190640100000000811115610a1257600080fd5b820183602082011115610a2457600080fd5b80359060200191846001830284011164010000000083111715610a4657600080fd5b9091929391929390505050611537565b60405180827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200191505060405180910390f35b610b2260048036036040811015610aa157600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190640100000000811115610ade57600080fd5b820183602082011115610af057600080fd5b80359060200191846001830284011164010000000083111715610b1257600080fd5b909192939192939050505061154f565b6040518080602001828103825283818151815260200191508051906020019080838360005b83811015610b62578082015181840152602081019050610b47565b50505050905090810190601f168015610b8f5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b610c68600480360360a0811015610bb357600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291908035906020019092919080359060200190640100000000811115610c2457600080fd5b820183602082011115610c3657600080fd5b80359060200191846001830284011164010000000083111715610c5857600080fd5b90919293919293905050506115b9565b60405180827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200191505060405180910390f35b5050505050505050565b60007f4e2312e0000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19161480610d7257507f150b7a02000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916145b80610dda57507f01ffc9a7000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916145b9050919050565b6000610ded33836113b5565b9050919050565b600063150b7a0260e01b905095945050505050565b60008033905060008173ffffffffffffffffffffffffffffffffffffffff166320c13b0b876040516020018082815260200191505060405160208183030381529060405287876040518463ffffffff1660e01b8152600401808060200180602001838103835286818151815260200191508051906020019080838360005b83811015610ea2578082015181840152602081019050610e87565b50505050905090810190601f168015610ecf5780820380516001836020036101000a031916815260200191505b508381038252858582818152602001925080828437600081840152601f19601f8201169050808301925050509550505050505060206040518083038186803b158015610f1a57600080fd5b505afa158015610f2e573d6000803e3d6000fd5b505050506040513d6020811015610f4457600080fd5b810190808051906020019092919050505090506320c13b0b60e01b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916817bffffffffffffffffffffffffffffffffffffffffffffffffffffffff191614610fad57600060e01b610fb6565b631626ba7e60e01b5b925050509392505050565b6000803390506000610fd38286611249565b90506000818051906020012090506000855114156110f25760008373ffffffffffffffffffffffffffffffffffffffff16635ae6bd37836040518263ffffffff1660e01b81526004018082815260200191505060206040518083038186803b15801561103e57600080fd5b505afa158015611052573d6000803e3d6000fd5b505050506040513d602081101561106857600080fd5b810190808051906020019092919050505014156110ed576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260118152602001807f48617368206e6f7420617070726f76656400000000000000000000000000000081525060200191505060405180910390fd5b611236565b8273ffffffffffffffffffffffffffffffffffffffff1663934f3a118284886040518463ffffffff1660e01b8152600401808481526020018060200180602001838103835285818151815260200191508051906020019080838360005b8381101561116a57808201518184015260208101905061114f565b50505050905090810190601f1680156111975780820380516001836020036101000a031916815260200191505b50838103825284818151815260200191508051906020019080838360005b838110156111d05780820151818401526020810190506111b5565b50505050905090810190601f1680156111fd5780820380516001836020036101000a031916815260200191505b509550505050505060006040518083038186803b15801561121d57600080fd5b505afa158015611231573d6000803e3d6000fd5b505050505b6320c13b0b60e01b935050505092915050565b606060007f60b3cbf8b4a223d68d641b3b6ddf9a298e7f33710cf3d3a9d1146b5a6150fbca60001b83805190602001206040516020018083815260200182815260200192505050604051602081830303815290604052805190602001209050601960f81b600160f81b8573ffffffffffffffffffffffffffffffffffffffff1663f698da256040518163ffffffff1660e01b815260040160206040518083038186803b1580156112f857600080fd5b505afa15801561130c573d6000803e3d6000fd5b505050506040513d602081101561132257600080fd5b81019080805190602001909291905050508360405160200180857effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168152600101847effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260010183815260200182815260200194505050505060405160208183030381529060405291505092915050565b60006113c18383611249565b80519060200120905092915050565b6060600033905060008173ffffffffffffffffffffffffffffffffffffffff1663cc2f84526001600a6040518363ffffffff1660e01b8152600401808373ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019250505060006040518083038186803b15801561144a57600080fd5b505afa15801561145e573d6000803e3d6000fd5b505050506040513d6000823e3d601f19601f82011682018060405250604081101561148857600080fd5b81019080805160405193929190846401000000008211156114a857600080fd5b838201915060208201858111156114be57600080fd5b82518660208202830111640100000000821117156114db57600080fd5b8083526020830192505050908051906020019060200280838360005b838110156115125780820151818401526020810190506114f7565b5050505090500160405260200180519060200190929190505050509050809250505090565b600063bc197c8160e01b905098975050505050505050565b60606040517fb4faba09000000000000000000000000000000000000000000000000000000008152600436036004808301376020600036836000335af15060203d036040519250808301604052806020843e6000516115b057825160208401fd5b50509392505050565b600063f23a6e6160e01b9050969550505050505056fea26469706673582212201b4a724e94687b6683f72064694993023a1b3dbf13a3418c0926ebb253c030fe64736f6c63430007060033",
}

// CompatibilityFallbackHandlerABI is the input ABI used to generate the binding from.
// Deprecated: Use CompatibilityFallbackHandlerMetaData.ABI instead.
var CompatibilityFallbackHandlerABI = CompatibilityFallbackHandlerMetaData.ABI

// CompatibilityFallbackHandlerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CompatibilityFallbackHandlerMetaData.Bin instead.
var CompatibilityFallbackHandlerBin = CompatibilityFallbackHandlerMetaData.Bin

// DeployCompatibilityFallbackHandler deploys a new Ethereum contract, binding an instance of CompatibilityFallbackHandler to it.
func DeployCompatibilityFallbackHandler(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *CompatibilityFallbackHandler, error) {
	parsed, err := CompatibilityFallbackHandlerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CompatibilityFallbackHandlerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &CompatibilityFallbackHandler{CompatibilityFallbackHandlerCaller: CompatibilityFallbackHandlerCaller{contract: contract}, CompatibilityFallbackHandlerTransactor: CompatibilityFallbackHandlerTransactor{contract: contract}, CompatibilityFallbackHandlerFilterer: CompatibilityFallbackHandlerFilterer{contract: contract}}, nil
}

// CompatibilityFallbackHandler is an auto generated Go binding around an Ethereum contract.
type CompatibilityFallbackHandler struct {
	CompatibilityFallbackHandlerCaller     // Read-only binding to the contract
	CompatibilityFallbackHandlerTransactor // Write-only binding to the contract
	CompatibilityFallbackHandlerFilterer   // Log filterer for contract events
}

// CompatibilityFallbackHandlerCaller is an auto generated read-only Go binding around an Ethereum contract.
type CompatibilityFallbackHandlerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CompatibilityFallbackHandlerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CompatibilityFallbackHandlerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CompatibilityFallbackHandlerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CompatibilityFallbackHandlerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CompatibilityFallbackHandlerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CompatibilityFallbackHandlerSession struct {
	Contract     *CompatibilityFallbackHandler // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                 // Call options to use throughout this session
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// CompatibilityFallbackHandlerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CompatibilityFallbackHandlerCallerSession struct {
	Contract *CompatibilityFallbackHandlerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                       // Call options to use throughout this session
}

// CompatibilityFallbackHandlerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CompatibilityFallbackHandlerTransactorSession struct {
	Contract     *CompatibilityFallbackHandlerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                       // Transaction auth options to use throughout this session
}

// CompatibilityFallbackHandlerRaw is an auto generated low-level Go binding around an Ethereum contract.
type CompatibilityFallbackHandlerRaw struct {
	Contract *CompatibilityFallbackHandler // Generic contract binding to access the raw methods on
}

// CompatibilityFallbackHandlerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CompatibilityFallbackHandlerCallerRaw struct {
	Contract *CompatibilityFallbackHandlerCaller // Generic read-only contract binding to access the raw methods on
}

// CompatibilityFallbackHandlerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CompatibilityFallbackHandlerTransactorRaw struct {
	Contract *CompatibilityFallbackHandlerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCompatibilityFallbackHandler creates a new instance of CompatibilityFallbackHandler, bound to a specific deployed contract.
func NewCompatibilityFallbackHandler(address common.Address, backend bind.ContractBackend) (*CompatibilityFallbackHandler, error) {
	contract, err := bindCompatibilityFallbackHandler(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CompatibilityFallbackHandler{CompatibilityFallbackHandlerCaller: CompatibilityFallbackHandlerCaller{contract: contract}, CompatibilityFallbackHandlerTransactor: CompatibilityFallbackHandlerTransactor{contract: contract}, CompatibilityFallbackHandlerFilterer: CompatibilityFallbackHandlerFilterer{contract: contract}}, nil
}

// NewCompatibilityFallbackHandlerCaller creates a new read-only instance of CompatibilityFallbackHandler, bound to a specific deployed contract.
func NewCompatibilityFallbackHandlerCaller(address common.Address, caller bind.ContractCaller) (*CompatibilityFallbackHandlerCaller, error) {
	contract, err := bindCompatibilityFallbackHandler(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CompatibilityFallbackHandlerCaller{contract: contract}, nil
}

// NewCompatibilityFallbackHandlerTransactor creates a new write-only instance of CompatibilityFallbackHandler, bound to a specific deployed contract.
func NewCompatibilityFallbackHandlerTransactor(address common.Address, transactor bind.ContractTransactor) (*CompatibilityFallbackHandlerTransactor, error) {
	contract, err := bindCompatibilityFallbackHandler(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CompatibilityFallbackHandlerTransactor{contract: contract}, nil
}

// NewCompatibilityFallbackHandlerFilterer creates a new log filterer instance of CompatibilityFallbackHandler, bound to a specific deployed contract.
func NewCompatibilityFallbackHandlerFilterer(address common.Address, filterer bind.ContractFilterer) (*CompatibilityFallbackHandlerFilterer, error) {
	contract, err := bindCompatibilityFallbackHandler(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CompatibilityFallbackHandlerFilterer{contract: contract}, nil
}

// bindCompatibilityFallbackHandler binds a generic wrapper to an already deployed contract.
func bindCompatibilityFallbackHandler(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := CompatibilityFallbackHandlerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CompatibilityFallbackHandler.Contract.CompatibilityFallbackHandlerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CompatibilityFallbackHandler.Contract.CompatibilityFallbackHandlerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CompatibilityFallbackHandler.Contract.CompatibilityFallbackHandlerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CompatibilityFallbackHandler.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CompatibilityFallbackHandler.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CompatibilityFallbackHandler.Contract.contract.Transact(opts, method, params...)
}

// EncodeMessageDataForSafe is a free data retrieval call binding the contract method 0x23031640.
//
// Solidity: function encodeMessageDataForSafe(address safe, bytes message) view returns(bytes)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) EncodeMessageDataForSafe(opts *bind.CallOpts, safe common.Address, message []byte) ([]byte, error) {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "encodeMessageDataForSafe", safe, message)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// EncodeMessageDataForSafe is a free data retrieval call binding the contract method 0x23031640.
//
// Solidity: function encodeMessageDataForSafe(address safe, bytes message) view returns(bytes)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) EncodeMessageDataForSafe(safe common.Address, message []byte) ([]byte, error) {
	return _CompatibilityFallbackHandler.Contract.EncodeMessageDataForSafe(&_CompatibilityFallbackHandler.CallOpts, safe, message)
}

// EncodeMessageDataForSafe is a free data retrieval call binding the contract method 0x23031640.
//
// Solidity: function encodeMessageDataForSafe(address safe, bytes message) view returns(bytes)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) EncodeMessageDataForSafe(safe common.Address, message []byte) ([]byte, error) {
	return _CompatibilityFallbackHandler.Contract.EncodeMessageDataForSafe(&_CompatibilityFallbackHandler.CallOpts, safe, message)
}

// GetMessageHash is a free data retrieval call binding the contract method 0x0a1028c4.
//
// Solidity: function getMessageHash(bytes message) view returns(bytes32)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) GetMessageHash(opts *bind.CallOpts, message []byte) ([32]byte, error) {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "getMessageHash", message)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetMessageHash is a free data retrieval call binding the contract method 0x0a1028c4.
//
// Solidity: function getMessageHash(bytes message) view returns(bytes32)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) GetMessageHash(message []byte) ([32]byte, error) {
	return _CompatibilityFallbackHandler.Contract.GetMessageHash(&_CompatibilityFallbackHandler.CallOpts, message)
}

// GetMessageHash is a free data retrieval call binding the contract method 0x0a1028c4.
//
// Solidity: function getMessageHash(bytes message) view returns(bytes32)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) GetMessageHash(message []byte) ([32]byte, error) {
	return _CompatibilityFallbackHandler.Contract.GetMessageHash(&_CompatibilityFallbackHandler.CallOpts, message)
}

// GetMessageHashForSafe is a free data retrieval call binding the contract method 0x6ac24784.
//
// Solidity: function getMessageHashForSafe(address safe, bytes message) view returns(bytes32)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) GetMessageHashForSafe(opts *bind.CallOpts, safe common.Address, message []byte) ([32]byte, error) {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "getMessageHashForSafe", safe, message)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetMessageHashForSafe is a free data retrieval call binding the contract method 0x6ac24784.
//
// Solidity: function getMessageHashForSafe(address safe, bytes message) view returns(bytes32)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) GetMessageHashForSafe(safe common.Address, message []byte) ([32]byte, error) {
	return _CompatibilityFallbackHandler.Contract.GetMessageHashForSafe(&_CompatibilityFallbackHandler.CallOpts, safe, message)
}

// GetMessageHashForSafe is a free data retrieval call binding the contract method 0x6ac24784.
//
// Solidity: function getMessageHashForSafe(address safe, bytes message) view returns(bytes32)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) GetMessageHashForSafe(safe common.Address, message []byte) ([32]byte, error) {
	return _CompatibilityFallbackHandler.Contract.GetMessageHashForSafe(&_CompatibilityFallbackHandler.CallOpts, safe, message)
}

// GetModules is a free data retrieval call binding the contract method 0xb2494df3.
//
// Solidity: function getModules() view returns(address[])
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) GetModules(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "getModules")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetModules is a free data retrieval call binding the contract method 0xb2494df3.
//
// Solidity: function getModules() view returns(address[])
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) GetModules() ([]common.Address, error) {
	return _CompatibilityFallbackHandler.Contract.GetModules(&_CompatibilityFallbackHandler.CallOpts)
}

// GetModules is a free data retrieval call binding the contract method 0xb2494df3.
//
// Solidity: function getModules() view returns(address[])
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) GetModules() ([]common.Address, error) {
	return _CompatibilityFallbackHandler.Contract.GetModules(&_CompatibilityFallbackHandler.CallOpts)
}

// IsValidSignature1626ba7e is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 _dataHash, bytes _signature) view returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) IsValidSignature1626ba7e(opts *bind.CallOpts, _dataHash [32]byte, _signature []byte) ([4]byte, error) {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "isValidSignature", _dataHash, _signature)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// IsValidSignature1626ba7e is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 _dataHash, bytes _signature) view returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) IsValidSignature1626ba7e(_dataHash [32]byte, _signature []byte) ([4]byte, error) {
	return _CompatibilityFallbackHandler.Contract.IsValidSignature1626ba7e(&_CompatibilityFallbackHandler.CallOpts, _dataHash, _signature)
}

// IsValidSignature1626ba7e is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 _dataHash, bytes _signature) view returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) IsValidSignature1626ba7e(_dataHash [32]byte, _signature []byte) ([4]byte, error) {
	return _CompatibilityFallbackHandler.Contract.IsValidSignature1626ba7e(&_CompatibilityFallbackHandler.CallOpts, _dataHash, _signature)
}

// IsValidSignature20c13b0b is a free data retrieval call binding the contract method 0x20c13b0b.
//
// Solidity: function isValidSignature(bytes _data, bytes _signature) view returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) IsValidSignature20c13b0b(opts *bind.CallOpts, _data []byte, _signature []byte) ([4]byte, error) {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "isValidSignature0", _data, _signature)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// IsValidSignature20c13b0b is a free data retrieval call binding the contract method 0x20c13b0b.
//
// Solidity: function isValidSignature(bytes _data, bytes _signature) view returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) IsValidSignature20c13b0b(_data []byte, _signature []byte) ([4]byte, error) {
	return _CompatibilityFallbackHandler.Contract.IsValidSignature20c13b0b(&_CompatibilityFallbackHandler.CallOpts, _data, _signature)
}

// IsValidSignature20c13b0b is a free data retrieval call binding the contract method 0x20c13b0b.
//
// Solidity: function isValidSignature(bytes _data, bytes _signature) view returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) IsValidSignature20c13b0b(_data []byte, _signature []byte) ([4]byte, error) {
	return _CompatibilityFallbackHandler.Contract.IsValidSignature20c13b0b(&_CompatibilityFallbackHandler.CallOpts, _data, _signature)
}

// OnERC1155BatchReceived is a free data retrieval call binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address , address , uint256[] , uint256[] , bytes ) pure returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) OnERC1155BatchReceived(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 []*big.Int, arg3 []*big.Int, arg4 []byte) ([4]byte, error) {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "onERC1155BatchReceived", arg0, arg1, arg2, arg3, arg4)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// OnERC1155BatchReceived is a free data retrieval call binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address , address , uint256[] , uint256[] , bytes ) pure returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) OnERC1155BatchReceived(arg0 common.Address, arg1 common.Address, arg2 []*big.Int, arg3 []*big.Int, arg4 []byte) ([4]byte, error) {
	return _CompatibilityFallbackHandler.Contract.OnERC1155BatchReceived(&_CompatibilityFallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// OnERC1155BatchReceived is a free data retrieval call binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address , address , uint256[] , uint256[] , bytes ) pure returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) OnERC1155BatchReceived(arg0 common.Address, arg1 common.Address, arg2 []*big.Int, arg3 []*big.Int, arg4 []byte) ([4]byte, error) {
	return _CompatibilityFallbackHandler.Contract.OnERC1155BatchReceived(&_CompatibilityFallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// OnERC1155Received is a free data retrieval call binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address , address , uint256 , uint256 , bytes ) pure returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) OnERC1155Received(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte) ([4]byte, error) {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "onERC1155Received", arg0, arg1, arg2, arg3, arg4)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// OnERC1155Received is a free data retrieval call binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address , address , uint256 , uint256 , bytes ) pure returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) OnERC1155Received(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte) ([4]byte, error) {
	return _CompatibilityFallbackHandler.Contract.OnERC1155Received(&_CompatibilityFallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// OnERC1155Received is a free data retrieval call binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address , address , uint256 , uint256 , bytes ) pure returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) OnERC1155Received(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte) ([4]byte, error) {
	return _CompatibilityFallbackHandler.Contract.OnERC1155Received(&_CompatibilityFallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// OnERC721Received is a free data retrieval call binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address , uint256 , bytes ) pure returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) OnERC721Received(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 []byte) ([4]byte, error) {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "onERC721Received", arg0, arg1, arg2, arg3)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// OnERC721Received is a free data retrieval call binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address , uint256 , bytes ) pure returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) OnERC721Received(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 []byte) ([4]byte, error) {
	return _CompatibilityFallbackHandler.Contract.OnERC721Received(&_CompatibilityFallbackHandler.CallOpts, arg0, arg1, arg2, arg3)
}

// OnERC721Received is a free data retrieval call binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address , uint256 , bytes ) pure returns(bytes4)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) OnERC721Received(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 []byte) ([4]byte, error) {
	return _CompatibilityFallbackHandler.Contract.OnERC721Received(&_CompatibilityFallbackHandler.CallOpts, arg0, arg1, arg2, arg3)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _CompatibilityFallbackHandler.Contract.SupportsInterface(&_CompatibilityFallbackHandler.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _CompatibilityFallbackHandler.Contract.SupportsInterface(&_CompatibilityFallbackHandler.CallOpts, interfaceId)
}

// TokensReceived is a free data retrieval call binding the contract method 0x0023de29.
//
// Solidity: function tokensReceived(address , address , address , uint256 , bytes , bytes ) pure returns()
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCaller) TokensReceived(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 common.Address, arg3 *big.Int, arg4 []byte, arg5 []byte) error {
	var out []interface{}
	err := _CompatibilityFallbackHandler.contract.Call(opts, &out, "tokensReceived", arg0, arg1, arg2, arg3, arg4, arg5)

	if err != nil {
		return err
	}

	return err

}

// TokensReceived is a free data retrieval call binding the contract method 0x0023de29.
//
// Solidity: function tokensReceived(address , address , address , uint256 , bytes , bytes ) pure returns()
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) TokensReceived(arg0 common.Address, arg1 common.Address, arg2 common.Address, arg3 *big.Int, arg4 []byte, arg5 []byte) error {
	return _CompatibilityFallbackHandler.Contract.TokensReceived(&_CompatibilityFallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// TokensReceived is a free data retrieval call binding the contract method 0x0023de29.
//
// Solidity: function tokensReceived(address , address , address , uint256 , bytes , bytes ) pure returns()
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerCallerSession) TokensReceived(arg0 common.Address, arg1 common.Address, arg2 common.Address, arg3 *big.Int, arg4 []byte, arg5 []byte) error {
	return _CompatibilityFallbackHandler.Contract.TokensReceived(&_CompatibilityFallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// Simulate is a paid mutator transaction binding the contract method 0xbd61951d.
//
// Solidity: function simulate(address targetContract, bytes calldataPayload) returns(bytes response)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerTransactor) Simulate(opts *bind.TransactOpts, targetContract common.Address, calldataPayload []byte) (*types.Transaction, error) {
	return _CompatibilityFallbackHandler.contract.Transact(opts, "simulate", targetContract, calldataPayload)
}

// Simulate is a paid mutator transaction binding the contract method 0xbd61951d.
//
// Solidity: function simulate(address targetContract, bytes calldataPayload) returns(bytes response)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerSession) Simulate(targetContract common.Address, calldataPayload []byte) (*types.Transaction, error) {
	return _CompatibilityFallbackHandler.Contract.Simulate(&_CompatibilityFallbackHandler.TransactOpts, targetContract, calldataPayload)
}

// Simulate is a paid mutator transaction binding the contract method 0xbd61951d.
//
// Solidity: function simulate(address targetContract, bytes calldataPayload) returns(bytes response)
func (_CompatibilityFallbackHandler *CompatibilityFallbackHandlerTransactorSession) Simulate(targetContract common.Address, calldataPayload []byte) (*types.Transaction, error) {
	return _CompatibilityFallbackHandler.Contract.Simulate(&_CompatibilityFallbackHandler.TransactOpts, targetContract, calldataPayload)
}

// HandlerContextMetaData contains all meta data concerning the HandlerContext contract.
var HandlerContextMetaData = &bind.MetaData{
	ABI: "[]",
}

// HandlerContextABI is the input ABI used to generate the binding from.
// Deprecated: Use HandlerContextMetaData.ABI instead.
var HandlerContextABI = HandlerContextMetaData.ABI

// HandlerContext is an auto generated Go binding around an Ethereum contract.
type HandlerContext struct {
	HandlerContextCaller     // Read-only binding to the contract
	HandlerContextTransactor // Write-only binding to the contract
	HandlerContextFilterer   // Log filterer for contract events
}

// HandlerContextCaller is an auto generated read-only Go binding around an Ethereum contract.
type HandlerContextCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HandlerContextTransactor is an auto generated write-only Go binding around an Ethereum contract.
type HandlerContextTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HandlerContextFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type HandlerContextFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HandlerContextSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type HandlerContextSession struct {
	Contract     *HandlerContext   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// HandlerContextCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type HandlerContextCallerSession struct {
	Contract *HandlerContextCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// HandlerContextTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type HandlerContextTransactorSession struct {
	Contract     *HandlerContextTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// HandlerContextRaw is an auto generated low-level Go binding around an Ethereum contract.
type HandlerContextRaw struct {
	Contract *HandlerContext // Generic contract binding to access the raw methods on
}

// HandlerContextCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type HandlerContextCallerRaw struct {
	Contract *HandlerContextCaller // Generic read-only contract binding to access the raw methods on
}

// HandlerContextTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type HandlerContextTransactorRaw struct {
	Contract *HandlerContextTransactor // Generic write-only contract binding to access the raw methods on
}

// NewHandlerContext creates a new instance of HandlerContext, bound to a specific deployed contract.
func NewHandlerContext(address common.Address, backend bind.ContractBackend) (*HandlerContext, error) {
	contract, err := bindHandlerContext(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &HandlerContext{HandlerContextCaller: HandlerContextCaller{contract: contract}, HandlerContextTransactor: HandlerContextTransactor{contract: contract}, HandlerContextFilterer: HandlerContextFilterer{contract: contract}}, nil
}

// NewHandlerContextCaller creates a new read-only instance of HandlerContext, bound to a specific deployed contract.
func NewHandlerContextCaller(address common.Address, caller bind.ContractCaller) (*HandlerContextCaller, error) {
	contract, err := bindHandlerContext(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &HandlerContextCaller{contract: contract}, nil
}

// NewHandlerContextTransactor creates a new write-only instance of HandlerContext, bound to a specific deployed contract.
func NewHandlerContextTransactor(address common.Address, transactor bind.ContractTransactor) (*HandlerContextTransactor, error) {
	contract, err := bindHandlerContext(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &HandlerContextTransactor{contract: contract}, nil
}

// NewHandlerContextFilterer creates a new log filterer instance of HandlerContext, bound to a specific deployed contract.
func NewHandlerContextFilterer(address common.Address, filterer bind.ContractFilterer) (*HandlerContextFilterer, error) {
	contract, err := bindHandlerContext(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &HandlerContextFilterer{contract: contract}, nil
}

// bindHandlerContext binds a generic wrapper to an already deployed contract.
func bindHandlerContext(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := HandlerContextMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HandlerContext *HandlerContextRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HandlerContext.Contract.HandlerContextCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HandlerContext *HandlerContextRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HandlerContext.Contract.HandlerContextTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HandlerContext *HandlerContextRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HandlerContext.Contract.HandlerContextTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HandlerContext *HandlerContextCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HandlerContext.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HandlerContext *HandlerContextTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HandlerContext.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HandlerContext *HandlerContextTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HandlerContext.Contract.contract.Transact(opts, method, params...)
}

// TokenCallbackHandlerMetaData contains all meta data concerning the TokenCallbackHandler contract.
var TokenCallbackHandlerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"onERC1155BatchReceived\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"onERC1155Received\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"onERC721Received\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"tokensReceived\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061073f806100206000396000f3fe608060405234801561001057600080fd5b50600436106100565760003560e01c806223de291461005b57806301ffc9a714610193578063150b7a02146101f6578063bc197c81146102ec578063f23a6e6114610482575b600080fd5b610191600480360360c081101561007157600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190803590602001906401000000008111156100f857600080fd5b82018360208201111561010a57600080fd5b8035906020019184600183028401116401000000008311171561012c57600080fd5b90919293919293908035906020019064010000000081111561014d57600080fd5b82018360208201111561015f57600080fd5b8035906020019184600183028401116401000000008311171561018157600080fd5b9091929391929390505050610582565b005b6101de600480360360208110156101a957600080fd5b8101908080357bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916906020019092919050505061058c565b60405180821515815260200191505060405180910390f35b6102b76004803603608081101561020c57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291908035906020019064010000000081111561027357600080fd5b82018360208201111561028557600080fd5b803590602001918460018302840111640100000000831117156102a757600080fd5b90919293919293905050506106c6565b60405180827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200191505060405180910390f35b61044d600480360360a081101561030257600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019064010000000081111561035f57600080fd5b82018360208201111561037157600080fd5b8035906020019184602083028401116401000000008311171561039357600080fd5b9091929391929390803590602001906401000000008111156103b457600080fd5b8201836020820111156103c657600080fd5b803590602001918460208302840111640100000000831117156103e857600080fd5b90919293919293908035906020019064010000000081111561040957600080fd5b82018360208201111561041b57600080fd5b8035906020019184600183028401116401000000008311171561043d57600080fd5b90919293919293905050506106db565b60405180827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200191505060405180910390f35b61054d600480360360a081101561049857600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190803590602001909291908035906020019064010000000081111561050957600080fd5b82018360208201111561051b57600080fd5b8035906020019184600183028401116401000000008311171561053d57600080fd5b90919293919293905050506106f3565b60405180827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200191505060405180910390f35b5050505050505050565b60007f4e2312e0000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916148061065757507f150b7a02000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916145b806106bf57507f01ffc9a7000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916145b9050919050565b600063150b7a0260e01b905095945050505050565b600063bc197c8160e01b905098975050505050505050565b600063f23a6e6160e01b9050969550505050505056fea2646970667358221220f62cd059f3672bb04062df149e7ae71534a8512cca0172e695d98a43cff0c53564736f6c63430007060033",
}

// TokenCallbackHandlerABI is the input ABI used to generate the binding from.
// Deprecated: Use TokenCallbackHandlerMetaData.ABI instead.
var TokenCallbackHandlerABI = TokenCallbackHandlerMetaData.ABI

// TokenCallbackHandlerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TokenCallbackHandlerMetaData.Bin instead.
var TokenCallbackHandlerBin = TokenCallbackHandlerMetaData.Bin

// DeployTokenCallbackHandler deploys a new Ethereum contract, binding an instance of TokenCallbackHandler to it.
func DeployTokenCallbackHandler(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *TokenCallbackHandler, error) {
	parsed, err := TokenCallbackHandlerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TokenCallbackHandlerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TokenCallbackHandler{TokenCallbackHandlerCaller: TokenCallbackHandlerCaller{contract: contract}, TokenCallbackHandlerTransactor: TokenCallbackHandlerTransactor{contract: contract}, TokenCallbackHandlerFilterer: TokenCallbackHandlerFilterer{contract: contract}}, nil
}

// TokenCallbackHandler is an auto generated Go binding around an Ethereum contract.
type TokenCallbackHandler struct {
	TokenCallbackHandlerCaller     // Read-only binding to the contract
	TokenCallbackHandlerTransactor // Write-only binding to the contract
	TokenCallbackHandlerFilterer   // Log filterer for contract events
}

// TokenCallbackHandlerCaller is an auto generated read-only Go binding around an Ethereum contract.
type TokenCallbackHandlerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenCallbackHandlerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TokenCallbackHandlerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenCallbackHandlerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TokenCallbackHandlerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenCallbackHandlerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TokenCallbackHandlerSession struct {
	Contract     *TokenCallbackHandler // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// TokenCallbackHandlerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TokenCallbackHandlerCallerSession struct {
	Contract *TokenCallbackHandlerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// TokenCallbackHandlerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TokenCallbackHandlerTransactorSession struct {
	Contract     *TokenCallbackHandlerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// TokenCallbackHandlerRaw is an auto generated low-level Go binding around an Ethereum contract.
type TokenCallbackHandlerRaw struct {
	Contract *TokenCallbackHandler // Generic contract binding to access the raw methods on
}

// TokenCallbackHandlerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TokenCallbackHandlerCallerRaw struct {
	Contract *TokenCallbackHandlerCaller // Generic read-only contract binding to access the raw methods on
}

// TokenCallbackHandlerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TokenCallbackHandlerTransactorRaw struct {
	Contract *TokenCallbackHandlerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTokenCallbackHandler creates a new instance of TokenCallbackHandler, bound to a specific deployed contract.
func NewTokenCallbackHandler(address common.Address, backend bind.ContractBackend) (*TokenCallbackHandler, error) {
	contract, err := bindTokenCallbackHandler(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TokenCallbackHandler{TokenCallbackHandlerCaller: TokenCallbackHandlerCaller{contract: contract}, TokenCallbackHandlerTransactor: TokenCallbackHandlerTransactor{contract: contract}, TokenCallbackHandlerFilterer: TokenCallbackHandlerFilterer{contract: contract}}, nil
}

// NewTokenCallbackHandlerCaller creates a new read-only instance of TokenCallbackHandler, bound to a specific deployed contract.
func NewTokenCallbackHandlerCaller(address common.Address, caller bind.ContractCaller) (*TokenCallbackHandlerCaller, error) {
	contract, err := bindTokenCallbackHandler(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TokenCallbackHandlerCaller{contract: contract}, nil
}

// NewTokenCallbackHandlerTransactor creates a new write-only instance of TokenCallbackHandler, bound to a specific deployed contract.
func NewTokenCallbackHandlerTransactor(address common.Address, transactor bind.ContractTransactor) (*TokenCallbackHandlerTransactor, error) {
	contract, err := bindTokenCallbackHandler(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TokenCallbackHandlerTransactor{contract: contract}, nil
}

// NewTokenCallbackHandlerFilterer creates a new log filterer instance of TokenCallbackHandler, bound to a specific deployed contract.
func NewTokenCallbackHandlerFilterer(address common.Address, filterer bind.ContractFilterer) (*TokenCallbackHandlerFilterer, error) {
	contract, err := bindTokenCallbackHandler(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TokenCallbackHandlerFilterer{contract: contract}, nil
}

// bindTokenCallbackHandler binds a generic wrapper to an already deployed contract.
func bindTokenCallbackHandler(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TokenCallbackHandlerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TokenCallbackHandler *TokenCallbackHandlerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TokenCallbackHandler.Contract.TokenCallbackHandlerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TokenCallbackHandler *TokenCallbackHandlerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TokenCallbackHandler.Contract.TokenCallbackHandlerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TokenCallbackHandler *TokenCallbackHandlerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TokenCallbackHandler.Contract.TokenCallbackHandlerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TokenCallbackHandler *TokenCallbackHandlerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TokenCallbackHandler.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TokenCallbackHandler *TokenCallbackHandlerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TokenCallbackHandler.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TokenCallbackHandler *TokenCallbackHandlerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TokenCallbackHandler.Contract.contract.Transact(opts, method, params...)
}

// OnERC1155BatchReceived is a free data retrieval call binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address , address , uint256[] , uint256[] , bytes ) pure returns(bytes4)
func (_TokenCallbackHandler *TokenCallbackHandlerCaller) OnERC1155BatchReceived(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 []*big.Int, arg3 []*big.Int, arg4 []byte) ([4]byte, error) {
	var out []interface{}
	err := _TokenCallbackHandler.contract.Call(opts, &out, "onERC1155BatchReceived", arg0, arg1, arg2, arg3, arg4)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// OnERC1155BatchReceived is a free data retrieval call binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address , address , uint256[] , uint256[] , bytes ) pure returns(bytes4)
func (_TokenCallbackHandler *TokenCallbackHandlerSession) OnERC1155BatchReceived(arg0 common.Address, arg1 common.Address, arg2 []*big.Int, arg3 []*big.Int, arg4 []byte) ([4]byte, error) {
	return _TokenCallbackHandler.Contract.OnERC1155BatchReceived(&_TokenCallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// OnERC1155BatchReceived is a free data retrieval call binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address , address , uint256[] , uint256[] , bytes ) pure returns(bytes4)
func (_TokenCallbackHandler *TokenCallbackHandlerCallerSession) OnERC1155BatchReceived(arg0 common.Address, arg1 common.Address, arg2 []*big.Int, arg3 []*big.Int, arg4 []byte) ([4]byte, error) {
	return _TokenCallbackHandler.Contract.OnERC1155BatchReceived(&_TokenCallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// OnERC1155Received is a free data retrieval call binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address , address , uint256 , uint256 , bytes ) pure returns(bytes4)
func (_TokenCallbackHandler *TokenCallbackHandlerCaller) OnERC1155Received(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte) ([4]byte, error) {
	var out []interface{}
	err := _TokenCallbackHandler.contract.Call(opts, &out, "onERC1155Received", arg0, arg1, arg2, arg3, arg4)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// OnERC1155Received is a free data retrieval call binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address , address , uint256 , uint256 , bytes ) pure returns(bytes4)
func (_TokenCallbackHandler *TokenCallbackHandlerSession) OnERC1155Received(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte) ([4]byte, error) {
	return _TokenCallbackHandler.Contract.OnERC1155Received(&_TokenCallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// OnERC1155Received is a free data retrieval call binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address , address , uint256 , uint256 , bytes ) pure returns(bytes4)
func (_TokenCallbackHandler *TokenCallbackHandlerCallerSession) OnERC1155Received(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte) ([4]byte, error) {
	return _TokenCallbackHandler.Contract.OnERC1155Received(&_TokenCallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// OnERC721Received is a free data retrieval call binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address , uint256 , bytes ) pure returns(bytes4)
func (_TokenCallbackHandler *TokenCallbackHandlerCaller) OnERC721Received(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 []byte) ([4]byte, error) {
	var out []interface{}
	err := _TokenCallbackHandler.contract.Call(opts, &out, "onERC721Received", arg0, arg1, arg2, arg3)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// OnERC721Received is a free data retrieval call binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address , uint256 , bytes ) pure returns(bytes4)
func (_TokenCallbackHandler *TokenCallbackHandlerSession) OnERC721Received(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 []byte) ([4]byte, error) {
	return _TokenCallbackHandler.Contract.OnERC721Received(&_TokenCallbackHandler.CallOpts, arg0, arg1, arg2, arg3)
}

// OnERC721Received is a free data retrieval call binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address , uint256 , bytes ) pure returns(bytes4)
func (_TokenCallbackHandler *TokenCallbackHandlerCallerSession) OnERC721Received(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 []byte) ([4]byte, error) {
	return _TokenCallbackHandler.Contract.OnERC721Received(&_TokenCallbackHandler.CallOpts, arg0, arg1, arg2, arg3)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_TokenCallbackHandler *TokenCallbackHandlerCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _TokenCallbackHandler.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_TokenCallbackHandler *TokenCallbackHandlerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _TokenCallbackHandler.Contract.SupportsInterface(&_TokenCallbackHandler.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_TokenCallbackHandler *TokenCallbackHandlerCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _TokenCallbackHandler.Contract.SupportsInterface(&_TokenCallbackHandler.CallOpts, interfaceId)
}

// TokensReceived is a free data retrieval call binding the contract method 0x0023de29.
//
// Solidity: function tokensReceived(address , address , address , uint256 , bytes , bytes ) pure returns()
func (_TokenCallbackHandler *TokenCallbackHandlerCaller) TokensReceived(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 common.Address, arg3 *big.Int, arg4 []byte, arg5 []byte) error {
	var out []interface{}
	err := _TokenCallbackHandler.contract.Call(opts, &out, "tokensReceived", arg0, arg1, arg2, arg3, arg4, arg5)

	if err != nil {
		return err
	}

	return err

}

// TokensReceived is a free data retrieval call binding the contract method 0x0023de29.
//
// Solidity: function tokensReceived(address , address , address , uint256 , bytes , bytes ) pure returns()
func (_TokenCallbackHandler *TokenCallbackHandlerSession) TokensReceived(arg0 common.Address, arg1 common.Address, arg2 common.Address, arg3 *big.Int, arg4 []byte, arg5 []byte) error {
	return _TokenCallbackHandler.Contract.TokensReceived(&_TokenCallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// TokensReceived is a free data retrieval call binding the contract method 0x0023de29.
//
// Solidity: function tokensReceived(address , address , address , uint256 , bytes , bytes ) pure returns()
func (_TokenCallbackHandler *TokenCallbackHandlerCallerSession) TokensReceived(arg0 common.Address, arg1 common.Address, arg2 common.Address, arg3 *big.Int, arg4 []byte, arg5 []byte) error {
	return _TokenCallbackHandler.Contract.TokensReceived(&_TokenCallbackHandler.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}
