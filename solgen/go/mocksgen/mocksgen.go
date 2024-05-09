// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package mocksgen

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

// BufferConfig is an auto generated low-level Go binding around an user-defined struct.
type BufferConfig struct {
	Threshold            uint64
	Max                  uint64
	ReplenishRateInBasis uint64
}

// DelayProof is an auto generated low-level Go binding around an user-defined struct.
type DelayProof struct {
	BeforeDelayedAcc [32]byte
	DelayedMessage   MessagesMessage
}

// ExecutionContext is an auto generated low-level Go binding around an user-defined struct.
type ExecutionContext struct {
	MaxInboxMessagesRead  *big.Int
	Bridge                common.Address
	InitialWasmModuleRoot [32]byte
}

// ExecutionState is an auto generated low-level Go binding around an user-defined struct.
type ExecutionState struct {
	GlobalState   GlobalState
	MachineStatus uint8
}

// GlobalState is an auto generated low-level Go binding around an user-defined struct.
type GlobalState struct {
	Bytes32Vals [2][32]byte
	U64Vals     [2]uint64
}

// IBridgeTimeBounds is an auto generated low-level Go binding around an user-defined struct.
type IBridgeTimeBounds struct {
	MinTimestamp   uint64
	MaxTimestamp   uint64
	MinBlockNumber uint64
	MaxBlockNumber uint64
}

// ISequencerInboxMaxTimeVariation is an auto generated low-level Go binding around an user-defined struct.
type ISequencerInboxMaxTimeVariation struct {
	DelayBlocks   *big.Int
	FutureBlocks  *big.Int
	DelaySeconds  *big.Int
	FutureSeconds *big.Int
}

// MessagesMessage is an auto generated low-level Go binding around an user-defined struct.
type MessagesMessage struct {
	Kind            uint8
	Sender          common.Address
	BlockNumber     uint64
	Timestamp       uint64
	InboxSeqNum     *big.Int
	BaseFeeL1       *big.Int
	MessageDataHash [32]byte
}

// BridgeStubMetaData contains all meta data concerning the BridgeStub contract.
var BridgeStubMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"stored\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"received\",\"type\":\"uint256\"}],\"name\":\"BadSequencerMessageNumber\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"BridgeCallTriggered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"InboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeInboxAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"MessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"OutboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"}],\"name\":\"RollupUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newSequencerInbox\",\"type\":\"address\"}],\"name\":\"SequencerInboxUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"acceptFundsFromOldBridge\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"activeOutbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedDelayedInboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"}],\"name\":\"allowedDelayedInboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedOutboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"allowedOutboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"delayedInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"delayedMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"enqueueDelayedMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"enqueueSequencerMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"seqMessageIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"acc\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeCall\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"sequencerInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerReportedSubMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setDelayedInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"name\":\"setOutbox\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sequencerInbox\",\"type\":\"address\"}],\"name\":\"setSequencerInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"batchPoster\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"}],\"name\":\"submitBatchSpendingReport\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610ea4806100206000396000f3fe6080604052600436106101745760003560e01c80639e5d4c49116100cb578063cee3d7281161007f578063e77145f411610059578063e77145f41461020d578063eca067ad1461040d578063ee35f3271461042257600080fd5b8063cee3d728146103b2578063d5719dc2146103cd578063e76f5c8d146103ed57600080fd5b8063ae60bd13116100b0578063ae60bd1314610361578063c4d66de8146102bb578063cb23bcb51461039d57600080fd5b80639e5d4c4914610313578063ab5d89431461034157600080fd5b80635fca4a161161012d5780638db5993b116101075780638db5993b146102a8578063919cc706146102bb578063945e1147146102db57600080fd5b80635fca4a161461022f5780637a88b1071461024557806386598a561461026857600080fd5b8063413b35bd1161015e578063413b35bd146101bd57806347fb24c5146101ed5780634f61f8501461020f57600080fd5b806284120c1461017957806316bf55791461019d575b600080fd5b34801561018557600080fd5b506005545b6040519081526020015b60405180910390f35b3480156101a957600080fd5b5061018a6101b8366004610bae565b610442565b3480156101c957600080fd5b506101dd6101d8366004610bdf565b610463565b6040519015158152602001610194565b3480156101f957600080fd5b5061020d610208366004610c03565b6104b3565b005b34801561021b57600080fd5b5061020d61022a366004610bdf565b610701565b34801561023b57600080fd5b5061018a60075481565b34801561025157600080fd5b5061018a610260366004610c41565b600092915050565b34801561027457600080fd5b50610288610283366004610c6d565b610762565b604080519485526020850193909352918301526060820152608001610194565b61018a6102b6366004610c9f565b6108b2565b3480156102c757600080fd5b5061020d6102d6366004610bdf565b61092a565b3480156102e757600080fd5b506102fb6102f6366004610bae565b610972565b6040516001600160a01b039091168152602001610194565b34801561031f57600080fd5b5061033361032e366004610ce6565b61099c565b604051610194929190610d93565b34801561034d57600080fd5b506003546102fb906001600160a01b031681565b34801561036d57600080fd5b506101dd61037c366004610bdf565b6001600160a01b031660009081526020819052604090206001015460ff1690565b3480156103a957600080fd5b506102fb610463565b3480156103be57600080fd5b5061020d6102d6366004610c03565b3480156103d957600080fd5b5061018a6103e8366004610bae565b610a38565b3480156103f957600080fd5b506102fb610408366004610bae565b610a48565b34801561041957600080fd5b5060045461018a565b34801561042e57600080fd5b506006546102fb906001600160a01b031681565b6005818154811061045257600080fd5b600091825260209091200154905081565b60405162461bcd60e51b815260206004820152600f60248201527f4e4f545f494d504c454d454e544544000000000000000000000000000000000060448201526000906064015b60405180910390fd5b6001600160a01b03821660008181526020818152604091829020600181015492518515158152909360ff90931692917f6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521910160405180910390a28215158115150361051e5750505050565b82156105b75760408051808201825260018054825260208083018281526001600160a01b0389166000818152928390529482209351845551928201805460ff1916931515939093179092558054808201825591527fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf601805473ffffffffffffffffffffffffffffffffffffffff191690911790556106fb565b600180546105c6908290610dcf565b815481106105d6576105d6610df0565b6000918252602090912001548254600180546001600160a01b0390931692909190811061060557610605610df0565b9060005260206000200160006101000a8154816001600160a01b0302191690836001600160a01b031602179055508160000154600080600185600001548154811061065257610652610df0565b60009182526020808320909101546001600160a01b03168352820192909252604001902055600180548061068857610688610e06565b6000828152602080822083017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff908101805473ffffffffffffffffffffffffffffffffffffffff191690559092019092556001600160a01b038616825281905260408120908155600101805460ff191690555b50505050565b6006805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0383169081179091556040519081527f8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a9060200160405180910390a150565b600080600080856007541415801561077957508515155b8015610786575060075415155b156107cb576007546040517fe2051feb0000000000000000000000000000000000000000000000000000000081526004810191909152602481018790526044016104aa565b60078590556005549350831561080957600580546107eb90600190610dcf565b815481106107fb576107fb610df0565b906000526020600020015492505b861561083a57600461081c600189610dcf565b8154811061082c5761082c610df0565b906000526020600020015491505b60408051602081018590529081018990526060810183905260800160408051601f198184030181529190528051602090910120600580546001810182556000919091527f036b6384b5eca791c62761152d0c79bb0604c104a5fb6f4eb0703f3154bb3db0018190559398929750909550919350915050565b3360009081526020819052604081206001015460ff166109145760405162461bcd60e51b815260206004820152600e60248201527f4e4f545f46524f4d5f494e424f5800000000000000000000000000000000000060448201526064016104aa565b610922848443424887610a58565b949350505050565b60405162461bcd60e51b815260206004820152600f60248201527f4e4f545f494d504c454d454e544544000000000000000000000000000000000060448201526064016104aa565b6002818154811061098257600080fd5b6000918252602090912001546001600160a01b0316905081565b600060606109e1868686868080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250610b1392505050565b60405191935091506001600160a01b0387169033907f2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d46690610a2790899089908990610e1c565b60405180910390a394509492505050565b6004818154811061045257600080fd5b6001818154811061098257600080fd5b60045460408051600060208083018290526021830182905260358301829052603d8301829052604583018290526065830182905260858084018790528451808503909101815260a59093019093528151919092012090919060008215610ae3576004610ac5600185610dcf565b81548110610ad557610ad5610df0565b906000526020600020015490505b6004610aef8284610b7f565b81546001810183556000928352602090922090910155509098975050505050505050565b60006060846001600160a01b03168484604051610b309190610e52565b60006040518083038185875af1925050503d8060008114610b6d576040519150601f19603f3d011682016040523d82523d6000602084013e610b72565b606091505b5090969095509350505050565b604080516020808201859052818301849052825180830384018152606090920190925280519101205b92915050565b600060208284031215610bc057600080fd5b5035919050565b6001600160a01b0381168114610bdc57600080fd5b50565b600060208284031215610bf157600080fd5b8135610bfc81610bc7565b9392505050565b60008060408385031215610c1657600080fd5b8235610c2181610bc7565b915060208301358015158114610c3657600080fd5b809150509250929050565b60008060408385031215610c5457600080fd5b8235610c5f81610bc7565b946020939093013593505050565b60008060008060808587031215610c8357600080fd5b5050823594602084013594506040840135936060013592509050565b600080600060608486031215610cb457600080fd5b833560ff81168114610cc557600080fd5b92506020840135610cd581610bc7565b929592945050506040919091013590565b60008060008060608587031215610cfc57600080fd5b8435610d0781610bc7565b935060208501359250604085013567ffffffffffffffff80821115610d2b57600080fd5b818701915087601f830112610d3f57600080fd5b813581811115610d4e57600080fd5b886020828501011115610d6057600080fd5b95989497505060200194505050565b60005b83811015610d8a578181015183820152602001610d72565b50506000910152565b82151581526040602082015260008251806040840152610dba816060850160208701610d6f565b601f01601f1916919091016060019392505050565b81810381811115610ba857634e487b7160e01b600052601160045260246000fd5b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052603160045260246000fd5b83815260406020820152816040820152818360608301376000818301606090810191909152601f909201601f1916010192915050565b60008251610e64818460208701610d6f565b919091019291505056fea2646970667358221220c206ace8ef1bc041371fc4242f3ff4d3767518b8a19fceb1cfc3f29d7db891ee64736f6c63430008110033",
}

// BridgeStubABI is the input ABI used to generate the binding from.
// Deprecated: Use BridgeStubMetaData.ABI instead.
var BridgeStubABI = BridgeStubMetaData.ABI

// BridgeStubBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BridgeStubMetaData.Bin instead.
var BridgeStubBin = BridgeStubMetaData.Bin

// DeployBridgeStub deploys a new Ethereum contract, binding an instance of BridgeStub to it.
func DeployBridgeStub(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BridgeStub, error) {
	parsed, err := BridgeStubMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BridgeStubBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BridgeStub{BridgeStubCaller: BridgeStubCaller{contract: contract}, BridgeStubTransactor: BridgeStubTransactor{contract: contract}, BridgeStubFilterer: BridgeStubFilterer{contract: contract}}, nil
}

// BridgeStub is an auto generated Go binding around an Ethereum contract.
type BridgeStub struct {
	BridgeStubCaller     // Read-only binding to the contract
	BridgeStubTransactor // Write-only binding to the contract
	BridgeStubFilterer   // Log filterer for contract events
}

// BridgeStubCaller is an auto generated read-only Go binding around an Ethereum contract.
type BridgeStubCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeStubTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BridgeStubTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeStubFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BridgeStubFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeStubSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BridgeStubSession struct {
	Contract     *BridgeStub       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BridgeStubCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BridgeStubCallerSession struct {
	Contract *BridgeStubCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// BridgeStubTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BridgeStubTransactorSession struct {
	Contract     *BridgeStubTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// BridgeStubRaw is an auto generated low-level Go binding around an Ethereum contract.
type BridgeStubRaw struct {
	Contract *BridgeStub // Generic contract binding to access the raw methods on
}

// BridgeStubCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BridgeStubCallerRaw struct {
	Contract *BridgeStubCaller // Generic read-only contract binding to access the raw methods on
}

// BridgeStubTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BridgeStubTransactorRaw struct {
	Contract *BridgeStubTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBridgeStub creates a new instance of BridgeStub, bound to a specific deployed contract.
func NewBridgeStub(address common.Address, backend bind.ContractBackend) (*BridgeStub, error) {
	contract, err := bindBridgeStub(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BridgeStub{BridgeStubCaller: BridgeStubCaller{contract: contract}, BridgeStubTransactor: BridgeStubTransactor{contract: contract}, BridgeStubFilterer: BridgeStubFilterer{contract: contract}}, nil
}

// NewBridgeStubCaller creates a new read-only instance of BridgeStub, bound to a specific deployed contract.
func NewBridgeStubCaller(address common.Address, caller bind.ContractCaller) (*BridgeStubCaller, error) {
	contract, err := bindBridgeStub(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeStubCaller{contract: contract}, nil
}

// NewBridgeStubTransactor creates a new write-only instance of BridgeStub, bound to a specific deployed contract.
func NewBridgeStubTransactor(address common.Address, transactor bind.ContractTransactor) (*BridgeStubTransactor, error) {
	contract, err := bindBridgeStub(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeStubTransactor{contract: contract}, nil
}

// NewBridgeStubFilterer creates a new log filterer instance of BridgeStub, bound to a specific deployed contract.
func NewBridgeStubFilterer(address common.Address, filterer bind.ContractFilterer) (*BridgeStubFilterer, error) {
	contract, err := bindBridgeStub(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BridgeStubFilterer{contract: contract}, nil
}

// bindBridgeStub binds a generic wrapper to an already deployed contract.
func bindBridgeStub(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BridgeStubMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeStub *BridgeStubRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeStub.Contract.BridgeStubCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeStub *BridgeStubRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeStub.Contract.BridgeStubTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeStub *BridgeStubRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeStub.Contract.BridgeStubTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeStub *BridgeStubCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeStub.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeStub *BridgeStubTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeStub.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeStub *BridgeStubTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeStub.Contract.contract.Transact(opts, method, params...)
}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_BridgeStub *BridgeStubCaller) ActiveOutbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "activeOutbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_BridgeStub *BridgeStubSession) ActiveOutbox() (common.Address, error) {
	return _BridgeStub.Contract.ActiveOutbox(&_BridgeStub.CallOpts)
}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_BridgeStub *BridgeStubCallerSession) ActiveOutbox() (common.Address, error) {
	return _BridgeStub.Contract.ActiveOutbox(&_BridgeStub.CallOpts)
}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_BridgeStub *BridgeStubCaller) AllowedDelayedInboxList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "allowedDelayedInboxList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_BridgeStub *BridgeStubSession) AllowedDelayedInboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeStub.Contract.AllowedDelayedInboxList(&_BridgeStub.CallOpts, arg0)
}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_BridgeStub *BridgeStubCallerSession) AllowedDelayedInboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeStub.Contract.AllowedDelayedInboxList(&_BridgeStub.CallOpts, arg0)
}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_BridgeStub *BridgeStubCaller) AllowedDelayedInboxes(opts *bind.CallOpts, inbox common.Address) (bool, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "allowedDelayedInboxes", inbox)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_BridgeStub *BridgeStubSession) AllowedDelayedInboxes(inbox common.Address) (bool, error) {
	return _BridgeStub.Contract.AllowedDelayedInboxes(&_BridgeStub.CallOpts, inbox)
}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_BridgeStub *BridgeStubCallerSession) AllowedDelayedInboxes(inbox common.Address) (bool, error) {
	return _BridgeStub.Contract.AllowedDelayedInboxes(&_BridgeStub.CallOpts, inbox)
}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_BridgeStub *BridgeStubCaller) AllowedOutboxList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "allowedOutboxList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_BridgeStub *BridgeStubSession) AllowedOutboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeStub.Contract.AllowedOutboxList(&_BridgeStub.CallOpts, arg0)
}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_BridgeStub *BridgeStubCallerSession) AllowedOutboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeStub.Contract.AllowedOutboxList(&_BridgeStub.CallOpts, arg0)
}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address ) pure returns(bool)
func (_BridgeStub *BridgeStubCaller) AllowedOutboxes(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "allowedOutboxes", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address ) pure returns(bool)
func (_BridgeStub *BridgeStubSession) AllowedOutboxes(arg0 common.Address) (bool, error) {
	return _BridgeStub.Contract.AllowedOutboxes(&_BridgeStub.CallOpts, arg0)
}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address ) pure returns(bool)
func (_BridgeStub *BridgeStubCallerSession) AllowedOutboxes(arg0 common.Address) (bool, error) {
	return _BridgeStub.Contract.AllowedOutboxes(&_BridgeStub.CallOpts, arg0)
}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeStub *BridgeStubCaller) DelayedInboxAccs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "delayedInboxAccs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeStub *BridgeStubSession) DelayedInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeStub.Contract.DelayedInboxAccs(&_BridgeStub.CallOpts, arg0)
}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeStub *BridgeStubCallerSession) DelayedInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeStub.Contract.DelayedInboxAccs(&_BridgeStub.CallOpts, arg0)
}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_BridgeStub *BridgeStubCaller) DelayedMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "delayedMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_BridgeStub *BridgeStubSession) DelayedMessageCount() (*big.Int, error) {
	return _BridgeStub.Contract.DelayedMessageCount(&_BridgeStub.CallOpts)
}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_BridgeStub *BridgeStubCallerSession) DelayedMessageCount() (*big.Int, error) {
	return _BridgeStub.Contract.DelayedMessageCount(&_BridgeStub.CallOpts)
}

// Initialize is a free data retrieval call binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address ) pure returns()
func (_BridgeStub *BridgeStubCaller) Initialize(opts *bind.CallOpts, arg0 common.Address) error {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "initialize", arg0)

	if err != nil {
		return err
	}

	return err

}

// Initialize is a free data retrieval call binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address ) pure returns()
func (_BridgeStub *BridgeStubSession) Initialize(arg0 common.Address) error {
	return _BridgeStub.Contract.Initialize(&_BridgeStub.CallOpts, arg0)
}

// Initialize is a free data retrieval call binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address ) pure returns()
func (_BridgeStub *BridgeStubCallerSession) Initialize(arg0 common.Address) error {
	return _BridgeStub.Contract.Initialize(&_BridgeStub.CallOpts, arg0)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() pure returns(address)
func (_BridgeStub *BridgeStubCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() pure returns(address)
func (_BridgeStub *BridgeStubSession) Rollup() (common.Address, error) {
	return _BridgeStub.Contract.Rollup(&_BridgeStub.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() pure returns(address)
func (_BridgeStub *BridgeStubCallerSession) Rollup() (common.Address, error) {
	return _BridgeStub.Contract.Rollup(&_BridgeStub.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_BridgeStub *BridgeStubCaller) SequencerInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "sequencerInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_BridgeStub *BridgeStubSession) SequencerInbox() (common.Address, error) {
	return _BridgeStub.Contract.SequencerInbox(&_BridgeStub.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_BridgeStub *BridgeStubCallerSession) SequencerInbox() (common.Address, error) {
	return _BridgeStub.Contract.SequencerInbox(&_BridgeStub.CallOpts)
}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeStub *BridgeStubCaller) SequencerInboxAccs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "sequencerInboxAccs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeStub *BridgeStubSession) SequencerInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeStub.Contract.SequencerInboxAccs(&_BridgeStub.CallOpts, arg0)
}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeStub *BridgeStubCallerSession) SequencerInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeStub.Contract.SequencerInboxAccs(&_BridgeStub.CallOpts, arg0)
}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_BridgeStub *BridgeStubCaller) SequencerMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "sequencerMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_BridgeStub *BridgeStubSession) SequencerMessageCount() (*big.Int, error) {
	return _BridgeStub.Contract.SequencerMessageCount(&_BridgeStub.CallOpts)
}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_BridgeStub *BridgeStubCallerSession) SequencerMessageCount() (*big.Int, error) {
	return _BridgeStub.Contract.SequencerMessageCount(&_BridgeStub.CallOpts)
}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_BridgeStub *BridgeStubCaller) SequencerReportedSubMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "sequencerReportedSubMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_BridgeStub *BridgeStubSession) SequencerReportedSubMessageCount() (*big.Int, error) {
	return _BridgeStub.Contract.SequencerReportedSubMessageCount(&_BridgeStub.CallOpts)
}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_BridgeStub *BridgeStubCallerSession) SequencerReportedSubMessageCount() (*big.Int, error) {
	return _BridgeStub.Contract.SequencerReportedSubMessageCount(&_BridgeStub.CallOpts)
}

// SetOutbox is a free data retrieval call binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address , bool ) pure returns()
func (_BridgeStub *BridgeStubCaller) SetOutbox(opts *bind.CallOpts, arg0 common.Address, arg1 bool) error {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "setOutbox", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// SetOutbox is a free data retrieval call binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address , bool ) pure returns()
func (_BridgeStub *BridgeStubSession) SetOutbox(arg0 common.Address, arg1 bool) error {
	return _BridgeStub.Contract.SetOutbox(&_BridgeStub.CallOpts, arg0, arg1)
}

// SetOutbox is a free data retrieval call binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address , bool ) pure returns()
func (_BridgeStub *BridgeStubCallerSession) SetOutbox(arg0 common.Address, arg1 bool) error {
	return _BridgeStub.Contract.SetOutbox(&_BridgeStub.CallOpts, arg0, arg1)
}

// UpdateRollupAddress is a free data retrieval call binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address ) pure returns()
func (_BridgeStub *BridgeStubCaller) UpdateRollupAddress(opts *bind.CallOpts, arg0 common.Address) error {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "updateRollupAddress", arg0)

	if err != nil {
		return err
	}

	return err

}

// UpdateRollupAddress is a free data retrieval call binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address ) pure returns()
func (_BridgeStub *BridgeStubSession) UpdateRollupAddress(arg0 common.Address) error {
	return _BridgeStub.Contract.UpdateRollupAddress(&_BridgeStub.CallOpts, arg0)
}

// UpdateRollupAddress is a free data retrieval call binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address ) pure returns()
func (_BridgeStub *BridgeStubCallerSession) UpdateRollupAddress(arg0 common.Address) error {
	return _BridgeStub.Contract.UpdateRollupAddress(&_BridgeStub.CallOpts, arg0)
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_BridgeStub *BridgeStubTransactor) AcceptFundsFromOldBridge(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeStub.contract.Transact(opts, "acceptFundsFromOldBridge")
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_BridgeStub *BridgeStubSession) AcceptFundsFromOldBridge() (*types.Transaction, error) {
	return _BridgeStub.Contract.AcceptFundsFromOldBridge(&_BridgeStub.TransactOpts)
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_BridgeStub *BridgeStubTransactorSession) AcceptFundsFromOldBridge() (*types.Transaction, error) {
	return _BridgeStub.Contract.AcceptFundsFromOldBridge(&_BridgeStub.TransactOpts)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_BridgeStub *BridgeStubTransactor) EnqueueDelayedMessage(opts *bind.TransactOpts, kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeStub.contract.Transact(opts, "enqueueDelayedMessage", kind, sender, messageDataHash)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_BridgeStub *BridgeStubSession) EnqueueDelayedMessage(kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeStub.Contract.EnqueueDelayedMessage(&_BridgeStub.TransactOpts, kind, sender, messageDataHash)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_BridgeStub *BridgeStubTransactorSession) EnqueueDelayedMessage(kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeStub.Contract.EnqueueDelayedMessage(&_BridgeStub.TransactOpts, kind, sender, messageDataHash)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_BridgeStub *BridgeStubTransactor) EnqueueSequencerMessage(opts *bind.TransactOpts, dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _BridgeStub.contract.Transact(opts, "enqueueSequencerMessage", dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_BridgeStub *BridgeStubSession) EnqueueSequencerMessage(dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _BridgeStub.Contract.EnqueueSequencerMessage(&_BridgeStub.TransactOpts, dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_BridgeStub *BridgeStubTransactorSession) EnqueueSequencerMessage(dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _BridgeStub.Contract.EnqueueSequencerMessage(&_BridgeStub.TransactOpts, dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_BridgeStub *BridgeStubTransactor) ExecuteCall(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _BridgeStub.contract.Transact(opts, "executeCall", to, value, data)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_BridgeStub *BridgeStubSession) ExecuteCall(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _BridgeStub.Contract.ExecuteCall(&_BridgeStub.TransactOpts, to, value, data)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_BridgeStub *BridgeStubTransactorSession) ExecuteCall(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _BridgeStub.Contract.ExecuteCall(&_BridgeStub.TransactOpts, to, value, data)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_BridgeStub *BridgeStubTransactor) SetDelayedInbox(opts *bind.TransactOpts, inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeStub.contract.Transact(opts, "setDelayedInbox", inbox, enabled)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_BridgeStub *BridgeStubSession) SetDelayedInbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeStub.Contract.SetDelayedInbox(&_BridgeStub.TransactOpts, inbox, enabled)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_BridgeStub *BridgeStubTransactorSession) SetDelayedInbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeStub.Contract.SetDelayedInbox(&_BridgeStub.TransactOpts, inbox, enabled)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_BridgeStub *BridgeStubTransactor) SetSequencerInbox(opts *bind.TransactOpts, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _BridgeStub.contract.Transact(opts, "setSequencerInbox", _sequencerInbox)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_BridgeStub *BridgeStubSession) SetSequencerInbox(_sequencerInbox common.Address) (*types.Transaction, error) {
	return _BridgeStub.Contract.SetSequencerInbox(&_BridgeStub.TransactOpts, _sequencerInbox)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_BridgeStub *BridgeStubTransactorSession) SetSequencerInbox(_sequencerInbox common.Address) (*types.Transaction, error) {
	return _BridgeStub.Contract.SetSequencerInbox(&_BridgeStub.TransactOpts, _sequencerInbox)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address batchPoster, bytes32 dataHash) returns(uint256)
func (_BridgeStub *BridgeStubTransactor) SubmitBatchSpendingReport(opts *bind.TransactOpts, batchPoster common.Address, dataHash [32]byte) (*types.Transaction, error) {
	return _BridgeStub.contract.Transact(opts, "submitBatchSpendingReport", batchPoster, dataHash)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address batchPoster, bytes32 dataHash) returns(uint256)
func (_BridgeStub *BridgeStubSession) SubmitBatchSpendingReport(batchPoster common.Address, dataHash [32]byte) (*types.Transaction, error) {
	return _BridgeStub.Contract.SubmitBatchSpendingReport(&_BridgeStub.TransactOpts, batchPoster, dataHash)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address batchPoster, bytes32 dataHash) returns(uint256)
func (_BridgeStub *BridgeStubTransactorSession) SubmitBatchSpendingReport(batchPoster common.Address, dataHash [32]byte) (*types.Transaction, error) {
	return _BridgeStub.Contract.SubmitBatchSpendingReport(&_BridgeStub.TransactOpts, batchPoster, dataHash)
}

// BridgeStubBridgeCallTriggeredIterator is returned from FilterBridgeCallTriggered and is used to iterate over the raw logs and unpacked data for BridgeCallTriggered events raised by the BridgeStub contract.
type BridgeStubBridgeCallTriggeredIterator struct {
	Event *BridgeStubBridgeCallTriggered // Event containing the contract specifics and raw log

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
func (it *BridgeStubBridgeCallTriggeredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeStubBridgeCallTriggered)
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
		it.Event = new(BridgeStubBridgeCallTriggered)
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
func (it *BridgeStubBridgeCallTriggeredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeStubBridgeCallTriggeredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeStubBridgeCallTriggered represents a BridgeCallTriggered event raised by the BridgeStub contract.
type BridgeStubBridgeCallTriggered struct {
	Outbox common.Address
	To     common.Address
	Value  *big.Int
	Data   []byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBridgeCallTriggered is a free log retrieval operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_BridgeStub *BridgeStubFilterer) FilterBridgeCallTriggered(opts *bind.FilterOpts, outbox []common.Address, to []common.Address) (*BridgeStubBridgeCallTriggeredIterator, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _BridgeStub.contract.FilterLogs(opts, "BridgeCallTriggered", outboxRule, toRule)
	if err != nil {
		return nil, err
	}
	return &BridgeStubBridgeCallTriggeredIterator{contract: _BridgeStub.contract, event: "BridgeCallTriggered", logs: logs, sub: sub}, nil
}

// WatchBridgeCallTriggered is a free log subscription operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_BridgeStub *BridgeStubFilterer) WatchBridgeCallTriggered(opts *bind.WatchOpts, sink chan<- *BridgeStubBridgeCallTriggered, outbox []common.Address, to []common.Address) (event.Subscription, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _BridgeStub.contract.WatchLogs(opts, "BridgeCallTriggered", outboxRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeStubBridgeCallTriggered)
				if err := _BridgeStub.contract.UnpackLog(event, "BridgeCallTriggered", log); err != nil {
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

// ParseBridgeCallTriggered is a log parse operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_BridgeStub *BridgeStubFilterer) ParseBridgeCallTriggered(log types.Log) (*BridgeStubBridgeCallTriggered, error) {
	event := new(BridgeStubBridgeCallTriggered)
	if err := _BridgeStub.contract.UnpackLog(event, "BridgeCallTriggered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeStubInboxToggleIterator is returned from FilterInboxToggle and is used to iterate over the raw logs and unpacked data for InboxToggle events raised by the BridgeStub contract.
type BridgeStubInboxToggleIterator struct {
	Event *BridgeStubInboxToggle // Event containing the contract specifics and raw log

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
func (it *BridgeStubInboxToggleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeStubInboxToggle)
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
		it.Event = new(BridgeStubInboxToggle)
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
func (it *BridgeStubInboxToggleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeStubInboxToggleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeStubInboxToggle represents a InboxToggle event raised by the BridgeStub contract.
type BridgeStubInboxToggle struct {
	Inbox   common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInboxToggle is a free log retrieval operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_BridgeStub *BridgeStubFilterer) FilterInboxToggle(opts *bind.FilterOpts, inbox []common.Address) (*BridgeStubInboxToggleIterator, error) {

	var inboxRule []interface{}
	for _, inboxItem := range inbox {
		inboxRule = append(inboxRule, inboxItem)
	}

	logs, sub, err := _BridgeStub.contract.FilterLogs(opts, "InboxToggle", inboxRule)
	if err != nil {
		return nil, err
	}
	return &BridgeStubInboxToggleIterator{contract: _BridgeStub.contract, event: "InboxToggle", logs: logs, sub: sub}, nil
}

// WatchInboxToggle is a free log subscription operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_BridgeStub *BridgeStubFilterer) WatchInboxToggle(opts *bind.WatchOpts, sink chan<- *BridgeStubInboxToggle, inbox []common.Address) (event.Subscription, error) {

	var inboxRule []interface{}
	for _, inboxItem := range inbox {
		inboxRule = append(inboxRule, inboxItem)
	}

	logs, sub, err := _BridgeStub.contract.WatchLogs(opts, "InboxToggle", inboxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeStubInboxToggle)
				if err := _BridgeStub.contract.UnpackLog(event, "InboxToggle", log); err != nil {
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

// ParseInboxToggle is a log parse operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_BridgeStub *BridgeStubFilterer) ParseInboxToggle(log types.Log) (*BridgeStubInboxToggle, error) {
	event := new(BridgeStubInboxToggle)
	if err := _BridgeStub.contract.UnpackLog(event, "InboxToggle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeStubMessageDeliveredIterator is returned from FilterMessageDelivered and is used to iterate over the raw logs and unpacked data for MessageDelivered events raised by the BridgeStub contract.
type BridgeStubMessageDeliveredIterator struct {
	Event *BridgeStubMessageDelivered // Event containing the contract specifics and raw log

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
func (it *BridgeStubMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeStubMessageDelivered)
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
		it.Event = new(BridgeStubMessageDelivered)
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
func (it *BridgeStubMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeStubMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeStubMessageDelivered represents a MessageDelivered event raised by the BridgeStub contract.
type BridgeStubMessageDelivered struct {
	MessageIndex    *big.Int
	BeforeInboxAcc  [32]byte
	Inbox           common.Address
	Kind            uint8
	Sender          common.Address
	MessageDataHash [32]byte
	BaseFeeL1       *big.Int
	Timestamp       uint64
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterMessageDelivered is a free log retrieval operation binding the contract event 0x5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1.
//
// Solidity: event MessageDelivered(uint256 indexed messageIndex, bytes32 indexed beforeInboxAcc, address inbox, uint8 kind, address sender, bytes32 messageDataHash, uint256 baseFeeL1, uint64 timestamp)
func (_BridgeStub *BridgeStubFilterer) FilterMessageDelivered(opts *bind.FilterOpts, messageIndex []*big.Int, beforeInboxAcc [][32]byte) (*BridgeStubMessageDeliveredIterator, error) {

	var messageIndexRule []interface{}
	for _, messageIndexItem := range messageIndex {
		messageIndexRule = append(messageIndexRule, messageIndexItem)
	}
	var beforeInboxAccRule []interface{}
	for _, beforeInboxAccItem := range beforeInboxAcc {
		beforeInboxAccRule = append(beforeInboxAccRule, beforeInboxAccItem)
	}

	logs, sub, err := _BridgeStub.contract.FilterLogs(opts, "MessageDelivered", messageIndexRule, beforeInboxAccRule)
	if err != nil {
		return nil, err
	}
	return &BridgeStubMessageDeliveredIterator{contract: _BridgeStub.contract, event: "MessageDelivered", logs: logs, sub: sub}, nil
}

// WatchMessageDelivered is a free log subscription operation binding the contract event 0x5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1.
//
// Solidity: event MessageDelivered(uint256 indexed messageIndex, bytes32 indexed beforeInboxAcc, address inbox, uint8 kind, address sender, bytes32 messageDataHash, uint256 baseFeeL1, uint64 timestamp)
func (_BridgeStub *BridgeStubFilterer) WatchMessageDelivered(opts *bind.WatchOpts, sink chan<- *BridgeStubMessageDelivered, messageIndex []*big.Int, beforeInboxAcc [][32]byte) (event.Subscription, error) {

	var messageIndexRule []interface{}
	for _, messageIndexItem := range messageIndex {
		messageIndexRule = append(messageIndexRule, messageIndexItem)
	}
	var beforeInboxAccRule []interface{}
	for _, beforeInboxAccItem := range beforeInboxAcc {
		beforeInboxAccRule = append(beforeInboxAccRule, beforeInboxAccItem)
	}

	logs, sub, err := _BridgeStub.contract.WatchLogs(opts, "MessageDelivered", messageIndexRule, beforeInboxAccRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeStubMessageDelivered)
				if err := _BridgeStub.contract.UnpackLog(event, "MessageDelivered", log); err != nil {
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

// ParseMessageDelivered is a log parse operation binding the contract event 0x5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1.
//
// Solidity: event MessageDelivered(uint256 indexed messageIndex, bytes32 indexed beforeInboxAcc, address inbox, uint8 kind, address sender, bytes32 messageDataHash, uint256 baseFeeL1, uint64 timestamp)
func (_BridgeStub *BridgeStubFilterer) ParseMessageDelivered(log types.Log) (*BridgeStubMessageDelivered, error) {
	event := new(BridgeStubMessageDelivered)
	if err := _BridgeStub.contract.UnpackLog(event, "MessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeStubOutboxToggleIterator is returned from FilterOutboxToggle and is used to iterate over the raw logs and unpacked data for OutboxToggle events raised by the BridgeStub contract.
type BridgeStubOutboxToggleIterator struct {
	Event *BridgeStubOutboxToggle // Event containing the contract specifics and raw log

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
func (it *BridgeStubOutboxToggleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeStubOutboxToggle)
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
		it.Event = new(BridgeStubOutboxToggle)
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
func (it *BridgeStubOutboxToggleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeStubOutboxToggleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeStubOutboxToggle represents a OutboxToggle event raised by the BridgeStub contract.
type BridgeStubOutboxToggle struct {
	Outbox  common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOutboxToggle is a free log retrieval operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_BridgeStub *BridgeStubFilterer) FilterOutboxToggle(opts *bind.FilterOpts, outbox []common.Address) (*BridgeStubOutboxToggleIterator, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}

	logs, sub, err := _BridgeStub.contract.FilterLogs(opts, "OutboxToggle", outboxRule)
	if err != nil {
		return nil, err
	}
	return &BridgeStubOutboxToggleIterator{contract: _BridgeStub.contract, event: "OutboxToggle", logs: logs, sub: sub}, nil
}

// WatchOutboxToggle is a free log subscription operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_BridgeStub *BridgeStubFilterer) WatchOutboxToggle(opts *bind.WatchOpts, sink chan<- *BridgeStubOutboxToggle, outbox []common.Address) (event.Subscription, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}

	logs, sub, err := _BridgeStub.contract.WatchLogs(opts, "OutboxToggle", outboxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeStubOutboxToggle)
				if err := _BridgeStub.contract.UnpackLog(event, "OutboxToggle", log); err != nil {
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

// ParseOutboxToggle is a log parse operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_BridgeStub *BridgeStubFilterer) ParseOutboxToggle(log types.Log) (*BridgeStubOutboxToggle, error) {
	event := new(BridgeStubOutboxToggle)
	if err := _BridgeStub.contract.UnpackLog(event, "OutboxToggle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeStubRollupUpdatedIterator is returned from FilterRollupUpdated and is used to iterate over the raw logs and unpacked data for RollupUpdated events raised by the BridgeStub contract.
type BridgeStubRollupUpdatedIterator struct {
	Event *BridgeStubRollupUpdated // Event containing the contract specifics and raw log

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
func (it *BridgeStubRollupUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeStubRollupUpdated)
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
		it.Event = new(BridgeStubRollupUpdated)
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
func (it *BridgeStubRollupUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeStubRollupUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeStubRollupUpdated represents a RollupUpdated event raised by the BridgeStub contract.
type BridgeStubRollupUpdated struct {
	Rollup common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterRollupUpdated is a free log retrieval operation binding the contract event 0xae1f5aa15f6ff844896347ceca2a3c24c8d3a27785efdeacd581a0a95172784a.
//
// Solidity: event RollupUpdated(address rollup)
func (_BridgeStub *BridgeStubFilterer) FilterRollupUpdated(opts *bind.FilterOpts) (*BridgeStubRollupUpdatedIterator, error) {

	logs, sub, err := _BridgeStub.contract.FilterLogs(opts, "RollupUpdated")
	if err != nil {
		return nil, err
	}
	return &BridgeStubRollupUpdatedIterator{contract: _BridgeStub.contract, event: "RollupUpdated", logs: logs, sub: sub}, nil
}

// WatchRollupUpdated is a free log subscription operation binding the contract event 0xae1f5aa15f6ff844896347ceca2a3c24c8d3a27785efdeacd581a0a95172784a.
//
// Solidity: event RollupUpdated(address rollup)
func (_BridgeStub *BridgeStubFilterer) WatchRollupUpdated(opts *bind.WatchOpts, sink chan<- *BridgeStubRollupUpdated) (event.Subscription, error) {

	logs, sub, err := _BridgeStub.contract.WatchLogs(opts, "RollupUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeStubRollupUpdated)
				if err := _BridgeStub.contract.UnpackLog(event, "RollupUpdated", log); err != nil {
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

// ParseRollupUpdated is a log parse operation binding the contract event 0xae1f5aa15f6ff844896347ceca2a3c24c8d3a27785efdeacd581a0a95172784a.
//
// Solidity: event RollupUpdated(address rollup)
func (_BridgeStub *BridgeStubFilterer) ParseRollupUpdated(log types.Log) (*BridgeStubRollupUpdated, error) {
	event := new(BridgeStubRollupUpdated)
	if err := _BridgeStub.contract.UnpackLog(event, "RollupUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeStubSequencerInboxUpdatedIterator is returned from FilterSequencerInboxUpdated and is used to iterate over the raw logs and unpacked data for SequencerInboxUpdated events raised by the BridgeStub contract.
type BridgeStubSequencerInboxUpdatedIterator struct {
	Event *BridgeStubSequencerInboxUpdated // Event containing the contract specifics and raw log

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
func (it *BridgeStubSequencerInboxUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeStubSequencerInboxUpdated)
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
		it.Event = new(BridgeStubSequencerInboxUpdated)
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
func (it *BridgeStubSequencerInboxUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeStubSequencerInboxUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeStubSequencerInboxUpdated represents a SequencerInboxUpdated event raised by the BridgeStub contract.
type BridgeStubSequencerInboxUpdated struct {
	NewSequencerInbox common.Address
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterSequencerInboxUpdated is a free log retrieval operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_BridgeStub *BridgeStubFilterer) FilterSequencerInboxUpdated(opts *bind.FilterOpts) (*BridgeStubSequencerInboxUpdatedIterator, error) {

	logs, sub, err := _BridgeStub.contract.FilterLogs(opts, "SequencerInboxUpdated")
	if err != nil {
		return nil, err
	}
	return &BridgeStubSequencerInboxUpdatedIterator{contract: _BridgeStub.contract, event: "SequencerInboxUpdated", logs: logs, sub: sub}, nil
}

// WatchSequencerInboxUpdated is a free log subscription operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_BridgeStub *BridgeStubFilterer) WatchSequencerInboxUpdated(opts *bind.WatchOpts, sink chan<- *BridgeStubSequencerInboxUpdated) (event.Subscription, error) {

	logs, sub, err := _BridgeStub.contract.WatchLogs(opts, "SequencerInboxUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeStubSequencerInboxUpdated)
				if err := _BridgeStub.contract.UnpackLog(event, "SequencerInboxUpdated", log); err != nil {
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

// ParseSequencerInboxUpdated is a log parse operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_BridgeStub *BridgeStubFilterer) ParseSequencerInboxUpdated(log types.Log) (*BridgeStubSequencerInboxUpdated, error) {
	event := new(BridgeStubSequencerInboxUpdated)
	if err := _BridgeStub.contract.UnpackLog(event, "SequencerInboxUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeUnproxiedMetaData contains all meta data concerning the BridgeUnproxied contract.
var BridgeUnproxiedMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"stored\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"received\",\"type\":\"uint256\"}],\"name\":\"BadSequencerMessageNumber\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"}],\"name\":\"InvalidOutboxSet\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"NotContract\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotDelayedInbox\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotOutbox\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotRollupOrOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotSequencerInbox\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"BridgeCallTriggered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"InboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeInboxAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"MessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"OutboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"}],\"name\":\"RollupUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newSequencerInbox\",\"type\":\"address\"}],\"name\":\"SequencerInboxUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"acceptFundsFromOldBridge\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"activeOutbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedDelayedInboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"}],\"name\":\"allowedDelayedInboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedOutboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"}],\"name\":\"allowedOutboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"delayedInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"delayedMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"enqueueDelayedMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"enqueueSequencerMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"seqMessageIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"acc\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeCall\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"rollup_\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"sequencerInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerReportedSubMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setDelayedInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setOutbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sequencerInbox\",\"type\":\"address\"}],\"name\":\"setSequencerInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newMsgCount\",\"type\":\"uint256\"}],\"name\":\"setSequencerReportedSubMessageCount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"submitBatchSpendingReport\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"_rollup\",\"type\":\"address\"}],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a06040523060805234801561001457600080fd5b50600580546001600160a01b03199081166001600160a01b03179091556008805490911633179055608051611a716100576000396000610f2a0152611a716000f3fe60806040526004361061017f5760003560e01c80639e5d4c49116100d6578063d5719dc21161007f578063eca067ad11610059578063eca067ad14610457578063ee35f3271461046c578063f81ff3b31461048c57600080fd5b8063d5719dc214610417578063e76f5c8d14610437578063e77145f41461023457600080fd5b8063c4d66de8116100b0578063c4d66de8146103b7578063cb23bcb5146103d7578063cee3d728146103f757600080fd5b80639e5d4c4914610337578063ab5d894314610365578063ae60bd131461037a57600080fd5b80635fca4a16116101385780638db5993b116101125780638db5993b146102cc578063919cc706146102df578063945e1147146102ff57600080fd5b80635fca4a16146102565780637a88b1071461026c57806386598a561461028c57600080fd5b8063413b35bd11610169578063413b35bd146101c857806347fb24c5146102145780634f61f8501461023657600080fd5b806284120c1461018457806316bf5579146101a8575b600080fd5b34801561019057600080fd5b506007545b6040519081526020015b60405180910390f35b3480156101b457600080fd5b506101956101c3366004611761565b6104ac565b3480156101d457600080fd5b506102046101e336600461178f565b6001600160a01b031660009081526002602052604090206001015460ff1690565b604051901515815260200161019f565b34801561022057600080fd5b5061023461022f3660046117b3565b6104cd565b005b34801561024257600080fd5b5061023461025136600461178f565b6107d3565b34801561026257600080fd5b50610195600a5481565b34801561027857600080fd5b506101956102873660046117f1565b6108ff565b34801561029857600080fd5b506102ac6102a736600461181d565b610960565b60408051948552602085019390935291830152606082015260800161019f565b6101956102da36600461184f565b610af9565b3480156102eb57600080fd5b506102346102fa36600461178f565b610b0f565b34801561030b57600080fd5b5061031f61031a366004611761565b610c34565b6040516001600160a01b03909116815260200161019f565b34801561034357600080fd5b50610357610352366004611896565b610c5e565b60405161019f929190611943565b34801561037157600080fd5b5061031f610df4565b34801561038657600080fd5b5061020461039536600461178f565b6001600160a01b03166000908152600160208190526040909120015460ff1690565b3480156103c357600080fd5b506102346103d236600461178f565b610e37565b3480156103e357600080fd5b5060085461031f906001600160a01b031681565b34801561040357600080fd5b506102346104123660046117b3565b61105b565b34801561042357600080fd5b50610195610432366004611761565b6113c9565b34801561044357600080fd5b5061031f610452366004611761565b6113d9565b34801561046357600080fd5b50600654610195565b34801561047857600080fd5b5060095461031f906001600160a01b031681565b34801561049857600080fd5b506102346104a7366004611761565b6113e9565b600781815481106104bc57600080fd5b600091825260209091200154905081565b6008546001600160a01b0316331461059c5760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610529573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061054d919061197f565b9050336001600160a01b0382161461059a57600854604051630739600760e01b81523360048201526001600160a01b03918216602482015290821660448201526064015b60405180910390fd5b505b6001600160a01b0382166000818152600160208181526040928390209182015492518515158152919360ff90931692917f6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521910160405180910390a2821515811515036106085750505050565b82156106a357604080518082018252600380548252600160208084018281526001600160a01b038a166000818152928490529582209451855551938201805460ff1916941515949094179093558154908101825591527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b01805473ffffffffffffffffffffffffffffffffffffffff191690911790556107cc565b600380546106b39060019061199c565b815481106106c3576106c36119bd565b6000918252602090912001548254600380546001600160a01b039093169290919081106106f2576106f26119bd565b9060005260206000200160006101000a8154816001600160a01b0302191690836001600160a01b031602179055508160000154600160006003856000015481548110610740576107406119bd565b60009182526020808320909101546001600160a01b031683528201929092526040019020556003805480610776576107766119d3565b600082815260208082208301600019908101805473ffffffffffffffffffffffffffffffffffffffff191690559092019092556001600160a01b03861682526001908190526040822091825501805460ff191690555b50505b5050565b6008546001600160a01b0316331461089d5760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa15801561082f573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610853919061197f565b9050336001600160a01b0382161461089b57600854604051630739600760e01b81523360048201526001600160a01b0391821660248201529082166044820152606401610591565b505b6009805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0383169081179091556040519081527f8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a906020015b60405180910390a150565b6009546000906001600160a01b03163314610948576040517f88f84f04000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b610957600d84434248876114b8565b90505b92915050565b6009546000908190819081906001600160a01b031633146109af576040517f88f84f04000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b85600a54141580156109c057508515155b80156109cd5750600a5415155b15610a1257600a546040517fe2051feb000000000000000000000000000000000000000000000000000000008152600481019190915260248101879052604401610591565b600a85905560075493508315610a505760078054610a329060019061199c565b81548110610a4257610a426119bd565b906000526020600020015492505b8615610a81576006610a6360018961199c565b81548110610a7357610a736119bd565b906000526020600020015491505b60408051602081018590529081018990526060810183905260800160408051601f198184030181529190528051602090910120600780546001810182556000919091527fa66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c688018190559398929750909550919350915050565b6000610b078484843461168a565b949350505050565b6008546001600160a01b03163314610bd95760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610b6b573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b8f919061197f565b9050336001600160a01b03821614610bd757600854604051630739600760e01b81523360048201526001600160a01b0391821660248201529082166044820152606401610591565b505b6008805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0383169081179091556040519081527fae1f5aa15f6ff844896347ceca2a3c24c8d3a27785efdeacd581a0a95172784a906020016108f4565b60048181548110610c4457600080fd5b6000918252602090912001546001600160a01b0316905081565b3360009081526002602052604081206001015460609060ff16610caf576040517f32ea82ab000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b8215801590610cc657506001600160a01b0386163b155b15610d08576040517fb5cf5b8f0000000000000000000000000000000000000000000000000000000081526001600160a01b0387166004820152602401610591565b6005805473ffffffffffffffffffffffffffffffffffffffff1981163317909155604080516020601f87018190048102820181019092528581526001600160a01b0390921691610d76918991899189908990819084018382808284376000920191909152506116f292505050565b6005805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038581169190911790915560405192955090935088169033907f2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d46690610de2908a908a908a906119e9565b60405180910390a35094509492505050565b6005546000906001600160a01b03167fffffffffffffffffffffffff00000000000000000000000000000000000000018101610e3257600091505090565b919050565b600054610100900460ff1615808015610e575750600054600160ff909116105b80610e715750303b158015610e71575060005460ff166001145b610efd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a65640000000000000000000000000000000000006064820152608401610591565b6000805460ff191660011790558015610f20576000805461ff0019166101001790555b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610fd8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c00000000000000000000000000000000000000006064820152608401610591565b600580546001600160a01b0373ffffffffffffffffffffffffffffffffffffffff1991821681179092556008805490911691841691909117905580156107cf576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050565b6008546001600160a01b031633146111255760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa1580156110b7573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906110db919061197f565b9050336001600160a01b0382161461112357600854604051630739600760e01b81523360048201526001600160a01b0391821660248201529082166044820152606401610591565b505b7fffffffffffffffffffffffff00000000000000000000000000000000000000016001600160a01b03831601611192576040517f77abed100000000000000000000000000000000000000000000000000000000081526001600160a01b0383166004820152602401610591565b6001600160a01b038216600081815260026020908152604091829020600181015492518515158152909360ff90931692917f49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa910160405180910390a2821515811515036111ff5750505050565b821561129b57604080518082018252600480548252600160208084018281526001600160a01b038a16600081815260029093529582209451855551938201805460ff1916941515949094179093558154908101825591527f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b01805473ffffffffffffffffffffffffffffffffffffffff191690911790556107cc565b600480546112ab9060019061199c565b815481106112bb576112bb6119bd565b6000918252602090912001548254600480546001600160a01b039093169290919081106112ea576112ea6119bd565b9060005260206000200160006101000a8154816001600160a01b0302191690836001600160a01b031602179055508160000154600260006004856000015481548110611338576113386119bd565b60009182526020808320909101546001600160a01b03168352820192909252604001902055600480548061136e5761136e6119d3565b600082815260208082208301600019908101805473ffffffffffffffffffffffffffffffffffffffff191690559092019092556001600160a01b03861682526002905260408120908155600101805460ff1916905550505050565b600681815481106104bc57600080fd5b60038181548110610c4457600080fd5b6008546001600160a01b031633146114b35760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015611445573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611469919061197f565b9050336001600160a01b038216146114b157600854604051630739600760e01b81523360048201526001600160a01b0391821660248201529082166044820152606401610591565b505b600a55565b600654604080517fff0000000000000000000000000000000000000000000000000000000000000060f88a901b166020808301919091527fffffffffffffffffffffffffffffffffffffffff00000000000000000000000060608a901b1660218301527fffffffffffffffff00000000000000000000000000000000000000000000000060c089811b8216603585015288901b16603d830152604582018490526065820186905260858083018690528351808403909101815260a5909201909252805191012060009190600082156115b557600661159760018561199c565b815481106115a7576115a76119bd565b906000526020600020015490505b6040805160208082018490528183018590528251808303840181526060830180855281519190920120600680546001810182556000919091527ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f015533905260ff8c1660808201526001600160a01b038b1660a082015260c0810187905260e0810188905267ffffffffffffffff89166101008201529051829185917f5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1918190036101200190a3509098975050505050505050565b3360009081526001602081905260408220015460ff166116d8576040517fb6c60ea3000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b60006116e88686434248896114b8565b9695505050505050565b60006060846001600160a01b0316848460405161170f9190611a1f565b60006040518083038185875af1925050503d806000811461174c576040519150601f19603f3d011682016040523d82523d6000602084013e611751565b606091505b5090969095509350505050565b50565b60006020828403121561177357600080fd5b5035919050565b6001600160a01b038116811461175e57600080fd5b6000602082840312156117a157600080fd5b81356117ac8161177a565b9392505050565b600080604083850312156117c657600080fd5b82356117d18161177a565b9150602083013580151581146117e657600080fd5b809150509250929050565b6000806040838503121561180457600080fd5b823561180f8161177a565b946020939093013593505050565b6000806000806080858703121561183357600080fd5b5050823594602084013594506040840135936060013592509050565b60008060006060848603121561186457600080fd5b833560ff8116811461187557600080fd5b925060208401356118858161177a565b929592945050506040919091013590565b600080600080606085870312156118ac57600080fd5b84356118b78161177a565b935060208501359250604085013567ffffffffffffffff808211156118db57600080fd5b818701915087601f8301126118ef57600080fd5b8135818111156118fe57600080fd5b88602082850101111561191057600080fd5b95989497505060200194505050565b60005b8381101561193a578181015183820152602001611922565b50506000910152565b8215158152604060208201526000825180604084015261196a81606085016020870161191f565b601f01601f1916919091016060019392505050565b60006020828403121561199157600080fd5b81516117ac8161177a565b8181038181111561095a57634e487b7160e01b600052601160045260246000fd5b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052603160045260246000fd5b83815260406020820152816040820152818360608301376000818301606090810191909152601f909201601f1916010192915050565b60008251611a3181846020870161191f565b919091019291505056fea2646970667358221220008e1314f8b78e21874d9edd59b44e4c84387fd6ef25113e4f5d323fc72ab6e164736f6c63430008110033",
}

// BridgeUnproxiedABI is the input ABI used to generate the binding from.
// Deprecated: Use BridgeUnproxiedMetaData.ABI instead.
var BridgeUnproxiedABI = BridgeUnproxiedMetaData.ABI

// BridgeUnproxiedBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BridgeUnproxiedMetaData.Bin instead.
var BridgeUnproxiedBin = BridgeUnproxiedMetaData.Bin

// DeployBridgeUnproxied deploys a new Ethereum contract, binding an instance of BridgeUnproxied to it.
func DeployBridgeUnproxied(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BridgeUnproxied, error) {
	parsed, err := BridgeUnproxiedMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BridgeUnproxiedBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BridgeUnproxied{BridgeUnproxiedCaller: BridgeUnproxiedCaller{contract: contract}, BridgeUnproxiedTransactor: BridgeUnproxiedTransactor{contract: contract}, BridgeUnproxiedFilterer: BridgeUnproxiedFilterer{contract: contract}}, nil
}

// BridgeUnproxied is an auto generated Go binding around an Ethereum contract.
type BridgeUnproxied struct {
	BridgeUnproxiedCaller     // Read-only binding to the contract
	BridgeUnproxiedTransactor // Write-only binding to the contract
	BridgeUnproxiedFilterer   // Log filterer for contract events
}

// BridgeUnproxiedCaller is an auto generated read-only Go binding around an Ethereum contract.
type BridgeUnproxiedCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeUnproxiedTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BridgeUnproxiedTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeUnproxiedFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BridgeUnproxiedFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeUnproxiedSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BridgeUnproxiedSession struct {
	Contract     *BridgeUnproxied  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BridgeUnproxiedCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BridgeUnproxiedCallerSession struct {
	Contract *BridgeUnproxiedCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// BridgeUnproxiedTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BridgeUnproxiedTransactorSession struct {
	Contract     *BridgeUnproxiedTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// BridgeUnproxiedRaw is an auto generated low-level Go binding around an Ethereum contract.
type BridgeUnproxiedRaw struct {
	Contract *BridgeUnproxied // Generic contract binding to access the raw methods on
}

// BridgeUnproxiedCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BridgeUnproxiedCallerRaw struct {
	Contract *BridgeUnproxiedCaller // Generic read-only contract binding to access the raw methods on
}

// BridgeUnproxiedTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BridgeUnproxiedTransactorRaw struct {
	Contract *BridgeUnproxiedTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBridgeUnproxied creates a new instance of BridgeUnproxied, bound to a specific deployed contract.
func NewBridgeUnproxied(address common.Address, backend bind.ContractBackend) (*BridgeUnproxied, error) {
	contract, err := bindBridgeUnproxied(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxied{BridgeUnproxiedCaller: BridgeUnproxiedCaller{contract: contract}, BridgeUnproxiedTransactor: BridgeUnproxiedTransactor{contract: contract}, BridgeUnproxiedFilterer: BridgeUnproxiedFilterer{contract: contract}}, nil
}

// NewBridgeUnproxiedCaller creates a new read-only instance of BridgeUnproxied, bound to a specific deployed contract.
func NewBridgeUnproxiedCaller(address common.Address, caller bind.ContractCaller) (*BridgeUnproxiedCaller, error) {
	contract, err := bindBridgeUnproxied(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxiedCaller{contract: contract}, nil
}

// NewBridgeUnproxiedTransactor creates a new write-only instance of BridgeUnproxied, bound to a specific deployed contract.
func NewBridgeUnproxiedTransactor(address common.Address, transactor bind.ContractTransactor) (*BridgeUnproxiedTransactor, error) {
	contract, err := bindBridgeUnproxied(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxiedTransactor{contract: contract}, nil
}

// NewBridgeUnproxiedFilterer creates a new log filterer instance of BridgeUnproxied, bound to a specific deployed contract.
func NewBridgeUnproxiedFilterer(address common.Address, filterer bind.ContractFilterer) (*BridgeUnproxiedFilterer, error) {
	contract, err := bindBridgeUnproxied(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxiedFilterer{contract: contract}, nil
}

// bindBridgeUnproxied binds a generic wrapper to an already deployed contract.
func bindBridgeUnproxied(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BridgeUnproxiedMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeUnproxied *BridgeUnproxiedRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeUnproxied.Contract.BridgeUnproxiedCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeUnproxied *BridgeUnproxiedRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.BridgeUnproxiedTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeUnproxied *BridgeUnproxiedRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.BridgeUnproxiedTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeUnproxied *BridgeUnproxiedCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeUnproxied.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeUnproxied *BridgeUnproxiedTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeUnproxied *BridgeUnproxiedTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.contract.Transact(opts, method, params...)
}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedCaller) ActiveOutbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "activeOutbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedSession) ActiveOutbox() (common.Address, error) {
	return _BridgeUnproxied.Contract.ActiveOutbox(&_BridgeUnproxied.CallOpts)
}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) ActiveOutbox() (common.Address, error) {
	return _BridgeUnproxied.Contract.ActiveOutbox(&_BridgeUnproxied.CallOpts)
}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedCaller) AllowedDelayedInboxList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "allowedDelayedInboxList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedSession) AllowedDelayedInboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeUnproxied.Contract.AllowedDelayedInboxList(&_BridgeUnproxied.CallOpts, arg0)
}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) AllowedDelayedInboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeUnproxied.Contract.AllowedDelayedInboxList(&_BridgeUnproxied.CallOpts, arg0)
}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_BridgeUnproxied *BridgeUnproxiedCaller) AllowedDelayedInboxes(opts *bind.CallOpts, inbox common.Address) (bool, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "allowedDelayedInboxes", inbox)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_BridgeUnproxied *BridgeUnproxiedSession) AllowedDelayedInboxes(inbox common.Address) (bool, error) {
	return _BridgeUnproxied.Contract.AllowedDelayedInboxes(&_BridgeUnproxied.CallOpts, inbox)
}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) AllowedDelayedInboxes(inbox common.Address) (bool, error) {
	return _BridgeUnproxied.Contract.AllowedDelayedInboxes(&_BridgeUnproxied.CallOpts, inbox)
}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedCaller) AllowedOutboxList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "allowedOutboxList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedSession) AllowedOutboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeUnproxied.Contract.AllowedOutboxList(&_BridgeUnproxied.CallOpts, arg0)
}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) AllowedOutboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeUnproxied.Contract.AllowedOutboxList(&_BridgeUnproxied.CallOpts, arg0)
}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_BridgeUnproxied *BridgeUnproxiedCaller) AllowedOutboxes(opts *bind.CallOpts, outbox common.Address) (bool, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "allowedOutboxes", outbox)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_BridgeUnproxied *BridgeUnproxiedSession) AllowedOutboxes(outbox common.Address) (bool, error) {
	return _BridgeUnproxied.Contract.AllowedOutboxes(&_BridgeUnproxied.CallOpts, outbox)
}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) AllowedOutboxes(outbox common.Address) (bool, error) {
	return _BridgeUnproxied.Contract.AllowedOutboxes(&_BridgeUnproxied.CallOpts, outbox)
}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeUnproxied *BridgeUnproxiedCaller) DelayedInboxAccs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "delayedInboxAccs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeUnproxied *BridgeUnproxiedSession) DelayedInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeUnproxied.Contract.DelayedInboxAccs(&_BridgeUnproxied.CallOpts, arg0)
}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) DelayedInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeUnproxied.Contract.DelayedInboxAccs(&_BridgeUnproxied.CallOpts, arg0)
}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedCaller) DelayedMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "delayedMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedSession) DelayedMessageCount() (*big.Int, error) {
	return _BridgeUnproxied.Contract.DelayedMessageCount(&_BridgeUnproxied.CallOpts)
}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) DelayedMessageCount() (*big.Int, error) {
	return _BridgeUnproxied.Contract.DelayedMessageCount(&_BridgeUnproxied.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedSession) Rollup() (common.Address, error) {
	return _BridgeUnproxied.Contract.Rollup(&_BridgeUnproxied.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) Rollup() (common.Address, error) {
	return _BridgeUnproxied.Contract.Rollup(&_BridgeUnproxied.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedCaller) SequencerInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "sequencerInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedSession) SequencerInbox() (common.Address, error) {
	return _BridgeUnproxied.Contract.SequencerInbox(&_BridgeUnproxied.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) SequencerInbox() (common.Address, error) {
	return _BridgeUnproxied.Contract.SequencerInbox(&_BridgeUnproxied.CallOpts)
}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeUnproxied *BridgeUnproxiedCaller) SequencerInboxAccs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "sequencerInboxAccs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeUnproxied *BridgeUnproxiedSession) SequencerInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeUnproxied.Contract.SequencerInboxAccs(&_BridgeUnproxied.CallOpts, arg0)
}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) SequencerInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeUnproxied.Contract.SequencerInboxAccs(&_BridgeUnproxied.CallOpts, arg0)
}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedCaller) SequencerMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "sequencerMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedSession) SequencerMessageCount() (*big.Int, error) {
	return _BridgeUnproxied.Contract.SequencerMessageCount(&_BridgeUnproxied.CallOpts)
}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) SequencerMessageCount() (*big.Int, error) {
	return _BridgeUnproxied.Contract.SequencerMessageCount(&_BridgeUnproxied.CallOpts)
}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedCaller) SequencerReportedSubMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeUnproxied.contract.Call(opts, &out, "sequencerReportedSubMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedSession) SequencerReportedSubMessageCount() (*big.Int, error) {
	return _BridgeUnproxied.Contract.SequencerReportedSubMessageCount(&_BridgeUnproxied.CallOpts)
}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedCallerSession) SequencerReportedSubMessageCount() (*big.Int, error) {
	return _BridgeUnproxied.Contract.SequencerReportedSubMessageCount(&_BridgeUnproxied.CallOpts)
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactor) AcceptFundsFromOldBridge(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "acceptFundsFromOldBridge")
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_BridgeUnproxied *BridgeUnproxiedSession) AcceptFundsFromOldBridge() (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.AcceptFundsFromOldBridge(&_BridgeUnproxied.TransactOpts)
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) AcceptFundsFromOldBridge() (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.AcceptFundsFromOldBridge(&_BridgeUnproxied.TransactOpts)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedTransactor) EnqueueDelayedMessage(opts *bind.TransactOpts, kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "enqueueDelayedMessage", kind, sender, messageDataHash)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedSession) EnqueueDelayedMessage(kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.EnqueueDelayedMessage(&_BridgeUnproxied.TransactOpts, kind, sender, messageDataHash)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) EnqueueDelayedMessage(kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.EnqueueDelayedMessage(&_BridgeUnproxied.TransactOpts, kind, sender, messageDataHash)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_BridgeUnproxied *BridgeUnproxiedTransactor) EnqueueSequencerMessage(opts *bind.TransactOpts, dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "enqueueSequencerMessage", dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_BridgeUnproxied *BridgeUnproxiedSession) EnqueueSequencerMessage(dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.EnqueueSequencerMessage(&_BridgeUnproxied.TransactOpts, dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) EnqueueSequencerMessage(dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.EnqueueSequencerMessage(&_BridgeUnproxied.TransactOpts, dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_BridgeUnproxied *BridgeUnproxiedTransactor) ExecuteCall(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "executeCall", to, value, data)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_BridgeUnproxied *BridgeUnproxiedSession) ExecuteCall(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.ExecuteCall(&_BridgeUnproxied.TransactOpts, to, value, data)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) ExecuteCall(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.ExecuteCall(&_BridgeUnproxied.TransactOpts, to, value, data)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactor) Initialize(opts *bind.TransactOpts, rollup_ common.Address) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "initialize", rollup_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_BridgeUnproxied *BridgeUnproxiedSession) Initialize(rollup_ common.Address) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.Initialize(&_BridgeUnproxied.TransactOpts, rollup_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) Initialize(rollup_ common.Address) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.Initialize(&_BridgeUnproxied.TransactOpts, rollup_)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactor) SetDelayedInbox(opts *bind.TransactOpts, inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "setDelayedInbox", inbox, enabled)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_BridgeUnproxied *BridgeUnproxiedSession) SetDelayedInbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.SetDelayedInbox(&_BridgeUnproxied.TransactOpts, inbox, enabled)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) SetDelayedInbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.SetDelayedInbox(&_BridgeUnproxied.TransactOpts, inbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address outbox, bool enabled) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactor) SetOutbox(opts *bind.TransactOpts, outbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "setOutbox", outbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address outbox, bool enabled) returns()
func (_BridgeUnproxied *BridgeUnproxiedSession) SetOutbox(outbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.SetOutbox(&_BridgeUnproxied.TransactOpts, outbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address outbox, bool enabled) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) SetOutbox(outbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.SetOutbox(&_BridgeUnproxied.TransactOpts, outbox, enabled)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactor) SetSequencerInbox(opts *bind.TransactOpts, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "setSequencerInbox", _sequencerInbox)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_BridgeUnproxied *BridgeUnproxiedSession) SetSequencerInbox(_sequencerInbox common.Address) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.SetSequencerInbox(&_BridgeUnproxied.TransactOpts, _sequencerInbox)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) SetSequencerInbox(_sequencerInbox common.Address) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.SetSequencerInbox(&_BridgeUnproxied.TransactOpts, _sequencerInbox)
}

// SetSequencerReportedSubMessageCount is a paid mutator transaction binding the contract method 0xf81ff3b3.
//
// Solidity: function setSequencerReportedSubMessageCount(uint256 newMsgCount) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactor) SetSequencerReportedSubMessageCount(opts *bind.TransactOpts, newMsgCount *big.Int) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "setSequencerReportedSubMessageCount", newMsgCount)
}

// SetSequencerReportedSubMessageCount is a paid mutator transaction binding the contract method 0xf81ff3b3.
//
// Solidity: function setSequencerReportedSubMessageCount(uint256 newMsgCount) returns()
func (_BridgeUnproxied *BridgeUnproxiedSession) SetSequencerReportedSubMessageCount(newMsgCount *big.Int) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.SetSequencerReportedSubMessageCount(&_BridgeUnproxied.TransactOpts, newMsgCount)
}

// SetSequencerReportedSubMessageCount is a paid mutator transaction binding the contract method 0xf81ff3b3.
//
// Solidity: function setSequencerReportedSubMessageCount(uint256 newMsgCount) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) SetSequencerReportedSubMessageCount(newMsgCount *big.Int) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.SetSequencerReportedSubMessageCount(&_BridgeUnproxied.TransactOpts, newMsgCount)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address sender, bytes32 messageDataHash) returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedTransactor) SubmitBatchSpendingReport(opts *bind.TransactOpts, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "submitBatchSpendingReport", sender, messageDataHash)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address sender, bytes32 messageDataHash) returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedSession) SubmitBatchSpendingReport(sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.SubmitBatchSpendingReport(&_BridgeUnproxied.TransactOpts, sender, messageDataHash)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address sender, bytes32 messageDataHash) returns(uint256)
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) SubmitBatchSpendingReport(sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.SubmitBatchSpendingReport(&_BridgeUnproxied.TransactOpts, sender, messageDataHash)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactor) UpdateRollupAddress(opts *bind.TransactOpts, _rollup common.Address) (*types.Transaction, error) {
	return _BridgeUnproxied.contract.Transact(opts, "updateRollupAddress", _rollup)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_BridgeUnproxied *BridgeUnproxiedSession) UpdateRollupAddress(_rollup common.Address) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.UpdateRollupAddress(&_BridgeUnproxied.TransactOpts, _rollup)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_BridgeUnproxied *BridgeUnproxiedTransactorSession) UpdateRollupAddress(_rollup common.Address) (*types.Transaction, error) {
	return _BridgeUnproxied.Contract.UpdateRollupAddress(&_BridgeUnproxied.TransactOpts, _rollup)
}

// BridgeUnproxiedBridgeCallTriggeredIterator is returned from FilterBridgeCallTriggered and is used to iterate over the raw logs and unpacked data for BridgeCallTriggered events raised by the BridgeUnproxied contract.
type BridgeUnproxiedBridgeCallTriggeredIterator struct {
	Event *BridgeUnproxiedBridgeCallTriggered // Event containing the contract specifics and raw log

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
func (it *BridgeUnproxiedBridgeCallTriggeredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeUnproxiedBridgeCallTriggered)
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
		it.Event = new(BridgeUnproxiedBridgeCallTriggered)
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
func (it *BridgeUnproxiedBridgeCallTriggeredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeUnproxiedBridgeCallTriggeredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeUnproxiedBridgeCallTriggered represents a BridgeCallTriggered event raised by the BridgeUnproxied contract.
type BridgeUnproxiedBridgeCallTriggered struct {
	Outbox common.Address
	To     common.Address
	Value  *big.Int
	Data   []byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBridgeCallTriggered is a free log retrieval operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) FilterBridgeCallTriggered(opts *bind.FilterOpts, outbox []common.Address, to []common.Address) (*BridgeUnproxiedBridgeCallTriggeredIterator, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _BridgeUnproxied.contract.FilterLogs(opts, "BridgeCallTriggered", outboxRule, toRule)
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxiedBridgeCallTriggeredIterator{contract: _BridgeUnproxied.contract, event: "BridgeCallTriggered", logs: logs, sub: sub}, nil
}

// WatchBridgeCallTriggered is a free log subscription operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) WatchBridgeCallTriggered(opts *bind.WatchOpts, sink chan<- *BridgeUnproxiedBridgeCallTriggered, outbox []common.Address, to []common.Address) (event.Subscription, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _BridgeUnproxied.contract.WatchLogs(opts, "BridgeCallTriggered", outboxRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeUnproxiedBridgeCallTriggered)
				if err := _BridgeUnproxied.contract.UnpackLog(event, "BridgeCallTriggered", log); err != nil {
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

// ParseBridgeCallTriggered is a log parse operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) ParseBridgeCallTriggered(log types.Log) (*BridgeUnproxiedBridgeCallTriggered, error) {
	event := new(BridgeUnproxiedBridgeCallTriggered)
	if err := _BridgeUnproxied.contract.UnpackLog(event, "BridgeCallTriggered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeUnproxiedInboxToggleIterator is returned from FilterInboxToggle and is used to iterate over the raw logs and unpacked data for InboxToggle events raised by the BridgeUnproxied contract.
type BridgeUnproxiedInboxToggleIterator struct {
	Event *BridgeUnproxiedInboxToggle // Event containing the contract specifics and raw log

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
func (it *BridgeUnproxiedInboxToggleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeUnproxiedInboxToggle)
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
		it.Event = new(BridgeUnproxiedInboxToggle)
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
func (it *BridgeUnproxiedInboxToggleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeUnproxiedInboxToggleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeUnproxiedInboxToggle represents a InboxToggle event raised by the BridgeUnproxied contract.
type BridgeUnproxiedInboxToggle struct {
	Inbox   common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInboxToggle is a free log retrieval operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) FilterInboxToggle(opts *bind.FilterOpts, inbox []common.Address) (*BridgeUnproxiedInboxToggleIterator, error) {

	var inboxRule []interface{}
	for _, inboxItem := range inbox {
		inboxRule = append(inboxRule, inboxItem)
	}

	logs, sub, err := _BridgeUnproxied.contract.FilterLogs(opts, "InboxToggle", inboxRule)
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxiedInboxToggleIterator{contract: _BridgeUnproxied.contract, event: "InboxToggle", logs: logs, sub: sub}, nil
}

// WatchInboxToggle is a free log subscription operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) WatchInboxToggle(opts *bind.WatchOpts, sink chan<- *BridgeUnproxiedInboxToggle, inbox []common.Address) (event.Subscription, error) {

	var inboxRule []interface{}
	for _, inboxItem := range inbox {
		inboxRule = append(inboxRule, inboxItem)
	}

	logs, sub, err := _BridgeUnproxied.contract.WatchLogs(opts, "InboxToggle", inboxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeUnproxiedInboxToggle)
				if err := _BridgeUnproxied.contract.UnpackLog(event, "InboxToggle", log); err != nil {
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

// ParseInboxToggle is a log parse operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) ParseInboxToggle(log types.Log) (*BridgeUnproxiedInboxToggle, error) {
	event := new(BridgeUnproxiedInboxToggle)
	if err := _BridgeUnproxied.contract.UnpackLog(event, "InboxToggle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeUnproxiedInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the BridgeUnproxied contract.
type BridgeUnproxiedInitializedIterator struct {
	Event *BridgeUnproxiedInitialized // Event containing the contract specifics and raw log

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
func (it *BridgeUnproxiedInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeUnproxiedInitialized)
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
		it.Event = new(BridgeUnproxiedInitialized)
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
func (it *BridgeUnproxiedInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeUnproxiedInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeUnproxiedInitialized represents a Initialized event raised by the BridgeUnproxied contract.
type BridgeUnproxiedInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) FilterInitialized(opts *bind.FilterOpts) (*BridgeUnproxiedInitializedIterator, error) {

	logs, sub, err := _BridgeUnproxied.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxiedInitializedIterator{contract: _BridgeUnproxied.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *BridgeUnproxiedInitialized) (event.Subscription, error) {

	logs, sub, err := _BridgeUnproxied.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeUnproxiedInitialized)
				if err := _BridgeUnproxied.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) ParseInitialized(log types.Log) (*BridgeUnproxiedInitialized, error) {
	event := new(BridgeUnproxiedInitialized)
	if err := _BridgeUnproxied.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeUnproxiedMessageDeliveredIterator is returned from FilterMessageDelivered and is used to iterate over the raw logs and unpacked data for MessageDelivered events raised by the BridgeUnproxied contract.
type BridgeUnproxiedMessageDeliveredIterator struct {
	Event *BridgeUnproxiedMessageDelivered // Event containing the contract specifics and raw log

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
func (it *BridgeUnproxiedMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeUnproxiedMessageDelivered)
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
		it.Event = new(BridgeUnproxiedMessageDelivered)
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
func (it *BridgeUnproxiedMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeUnproxiedMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeUnproxiedMessageDelivered represents a MessageDelivered event raised by the BridgeUnproxied contract.
type BridgeUnproxiedMessageDelivered struct {
	MessageIndex    *big.Int
	BeforeInboxAcc  [32]byte
	Inbox           common.Address
	Kind            uint8
	Sender          common.Address
	MessageDataHash [32]byte
	BaseFeeL1       *big.Int
	Timestamp       uint64
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterMessageDelivered is a free log retrieval operation binding the contract event 0x5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1.
//
// Solidity: event MessageDelivered(uint256 indexed messageIndex, bytes32 indexed beforeInboxAcc, address inbox, uint8 kind, address sender, bytes32 messageDataHash, uint256 baseFeeL1, uint64 timestamp)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) FilterMessageDelivered(opts *bind.FilterOpts, messageIndex []*big.Int, beforeInboxAcc [][32]byte) (*BridgeUnproxiedMessageDeliveredIterator, error) {

	var messageIndexRule []interface{}
	for _, messageIndexItem := range messageIndex {
		messageIndexRule = append(messageIndexRule, messageIndexItem)
	}
	var beforeInboxAccRule []interface{}
	for _, beforeInboxAccItem := range beforeInboxAcc {
		beforeInboxAccRule = append(beforeInboxAccRule, beforeInboxAccItem)
	}

	logs, sub, err := _BridgeUnproxied.contract.FilterLogs(opts, "MessageDelivered", messageIndexRule, beforeInboxAccRule)
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxiedMessageDeliveredIterator{contract: _BridgeUnproxied.contract, event: "MessageDelivered", logs: logs, sub: sub}, nil
}

// WatchMessageDelivered is a free log subscription operation binding the contract event 0x5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1.
//
// Solidity: event MessageDelivered(uint256 indexed messageIndex, bytes32 indexed beforeInboxAcc, address inbox, uint8 kind, address sender, bytes32 messageDataHash, uint256 baseFeeL1, uint64 timestamp)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) WatchMessageDelivered(opts *bind.WatchOpts, sink chan<- *BridgeUnproxiedMessageDelivered, messageIndex []*big.Int, beforeInboxAcc [][32]byte) (event.Subscription, error) {

	var messageIndexRule []interface{}
	for _, messageIndexItem := range messageIndex {
		messageIndexRule = append(messageIndexRule, messageIndexItem)
	}
	var beforeInboxAccRule []interface{}
	for _, beforeInboxAccItem := range beforeInboxAcc {
		beforeInboxAccRule = append(beforeInboxAccRule, beforeInboxAccItem)
	}

	logs, sub, err := _BridgeUnproxied.contract.WatchLogs(opts, "MessageDelivered", messageIndexRule, beforeInboxAccRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeUnproxiedMessageDelivered)
				if err := _BridgeUnproxied.contract.UnpackLog(event, "MessageDelivered", log); err != nil {
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

// ParseMessageDelivered is a log parse operation binding the contract event 0x5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1.
//
// Solidity: event MessageDelivered(uint256 indexed messageIndex, bytes32 indexed beforeInboxAcc, address inbox, uint8 kind, address sender, bytes32 messageDataHash, uint256 baseFeeL1, uint64 timestamp)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) ParseMessageDelivered(log types.Log) (*BridgeUnproxiedMessageDelivered, error) {
	event := new(BridgeUnproxiedMessageDelivered)
	if err := _BridgeUnproxied.contract.UnpackLog(event, "MessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeUnproxiedOutboxToggleIterator is returned from FilterOutboxToggle and is used to iterate over the raw logs and unpacked data for OutboxToggle events raised by the BridgeUnproxied contract.
type BridgeUnproxiedOutboxToggleIterator struct {
	Event *BridgeUnproxiedOutboxToggle // Event containing the contract specifics and raw log

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
func (it *BridgeUnproxiedOutboxToggleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeUnproxiedOutboxToggle)
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
		it.Event = new(BridgeUnproxiedOutboxToggle)
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
func (it *BridgeUnproxiedOutboxToggleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeUnproxiedOutboxToggleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeUnproxiedOutboxToggle represents a OutboxToggle event raised by the BridgeUnproxied contract.
type BridgeUnproxiedOutboxToggle struct {
	Outbox  common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOutboxToggle is a free log retrieval operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) FilterOutboxToggle(opts *bind.FilterOpts, outbox []common.Address) (*BridgeUnproxiedOutboxToggleIterator, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}

	logs, sub, err := _BridgeUnproxied.contract.FilterLogs(opts, "OutboxToggle", outboxRule)
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxiedOutboxToggleIterator{contract: _BridgeUnproxied.contract, event: "OutboxToggle", logs: logs, sub: sub}, nil
}

// WatchOutboxToggle is a free log subscription operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) WatchOutboxToggle(opts *bind.WatchOpts, sink chan<- *BridgeUnproxiedOutboxToggle, outbox []common.Address) (event.Subscription, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}

	logs, sub, err := _BridgeUnproxied.contract.WatchLogs(opts, "OutboxToggle", outboxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeUnproxiedOutboxToggle)
				if err := _BridgeUnproxied.contract.UnpackLog(event, "OutboxToggle", log); err != nil {
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

// ParseOutboxToggle is a log parse operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) ParseOutboxToggle(log types.Log) (*BridgeUnproxiedOutboxToggle, error) {
	event := new(BridgeUnproxiedOutboxToggle)
	if err := _BridgeUnproxied.contract.UnpackLog(event, "OutboxToggle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeUnproxiedRollupUpdatedIterator is returned from FilterRollupUpdated and is used to iterate over the raw logs and unpacked data for RollupUpdated events raised by the BridgeUnproxied contract.
type BridgeUnproxiedRollupUpdatedIterator struct {
	Event *BridgeUnproxiedRollupUpdated // Event containing the contract specifics and raw log

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
func (it *BridgeUnproxiedRollupUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeUnproxiedRollupUpdated)
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
		it.Event = new(BridgeUnproxiedRollupUpdated)
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
func (it *BridgeUnproxiedRollupUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeUnproxiedRollupUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeUnproxiedRollupUpdated represents a RollupUpdated event raised by the BridgeUnproxied contract.
type BridgeUnproxiedRollupUpdated struct {
	Rollup common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterRollupUpdated is a free log retrieval operation binding the contract event 0xae1f5aa15f6ff844896347ceca2a3c24c8d3a27785efdeacd581a0a95172784a.
//
// Solidity: event RollupUpdated(address rollup)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) FilterRollupUpdated(opts *bind.FilterOpts) (*BridgeUnproxiedRollupUpdatedIterator, error) {

	logs, sub, err := _BridgeUnproxied.contract.FilterLogs(opts, "RollupUpdated")
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxiedRollupUpdatedIterator{contract: _BridgeUnproxied.contract, event: "RollupUpdated", logs: logs, sub: sub}, nil
}

// WatchRollupUpdated is a free log subscription operation binding the contract event 0xae1f5aa15f6ff844896347ceca2a3c24c8d3a27785efdeacd581a0a95172784a.
//
// Solidity: event RollupUpdated(address rollup)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) WatchRollupUpdated(opts *bind.WatchOpts, sink chan<- *BridgeUnproxiedRollupUpdated) (event.Subscription, error) {

	logs, sub, err := _BridgeUnproxied.contract.WatchLogs(opts, "RollupUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeUnproxiedRollupUpdated)
				if err := _BridgeUnproxied.contract.UnpackLog(event, "RollupUpdated", log); err != nil {
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

// ParseRollupUpdated is a log parse operation binding the contract event 0xae1f5aa15f6ff844896347ceca2a3c24c8d3a27785efdeacd581a0a95172784a.
//
// Solidity: event RollupUpdated(address rollup)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) ParseRollupUpdated(log types.Log) (*BridgeUnproxiedRollupUpdated, error) {
	event := new(BridgeUnproxiedRollupUpdated)
	if err := _BridgeUnproxied.contract.UnpackLog(event, "RollupUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeUnproxiedSequencerInboxUpdatedIterator is returned from FilterSequencerInboxUpdated and is used to iterate over the raw logs and unpacked data for SequencerInboxUpdated events raised by the BridgeUnproxied contract.
type BridgeUnproxiedSequencerInboxUpdatedIterator struct {
	Event *BridgeUnproxiedSequencerInboxUpdated // Event containing the contract specifics and raw log

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
func (it *BridgeUnproxiedSequencerInboxUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeUnproxiedSequencerInboxUpdated)
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
		it.Event = new(BridgeUnproxiedSequencerInboxUpdated)
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
func (it *BridgeUnproxiedSequencerInboxUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeUnproxiedSequencerInboxUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeUnproxiedSequencerInboxUpdated represents a SequencerInboxUpdated event raised by the BridgeUnproxied contract.
type BridgeUnproxiedSequencerInboxUpdated struct {
	NewSequencerInbox common.Address
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterSequencerInboxUpdated is a free log retrieval operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) FilterSequencerInboxUpdated(opts *bind.FilterOpts) (*BridgeUnproxiedSequencerInboxUpdatedIterator, error) {

	logs, sub, err := _BridgeUnproxied.contract.FilterLogs(opts, "SequencerInboxUpdated")
	if err != nil {
		return nil, err
	}
	return &BridgeUnproxiedSequencerInboxUpdatedIterator{contract: _BridgeUnproxied.contract, event: "SequencerInboxUpdated", logs: logs, sub: sub}, nil
}

// WatchSequencerInboxUpdated is a free log subscription operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) WatchSequencerInboxUpdated(opts *bind.WatchOpts, sink chan<- *BridgeUnproxiedSequencerInboxUpdated) (event.Subscription, error) {

	logs, sub, err := _BridgeUnproxied.contract.WatchLogs(opts, "SequencerInboxUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeUnproxiedSequencerInboxUpdated)
				if err := _BridgeUnproxied.contract.UnpackLog(event, "SequencerInboxUpdated", log); err != nil {
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

// ParseSequencerInboxUpdated is a log parse operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_BridgeUnproxied *BridgeUnproxiedFilterer) ParseSequencerInboxUpdated(log types.Log) (*BridgeUnproxiedSequencerInboxUpdated, error) {
	event := new(BridgeUnproxiedSequencerInboxUpdated)
	if err := _BridgeUnproxied.contract.UnpackLog(event, "SequencerInboxUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IWETH9MetaData contains all meta data concerning the IWETH9 contract.
var IWETH9MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"deposit\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IWETH9ABI is the input ABI used to generate the binding from.
// Deprecated: Use IWETH9MetaData.ABI instead.
var IWETH9ABI = IWETH9MetaData.ABI

// IWETH9 is an auto generated Go binding around an Ethereum contract.
type IWETH9 struct {
	IWETH9Caller     // Read-only binding to the contract
	IWETH9Transactor // Write-only binding to the contract
	IWETH9Filterer   // Log filterer for contract events
}

// IWETH9Caller is an auto generated read-only Go binding around an Ethereum contract.
type IWETH9Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IWETH9Transactor is an auto generated write-only Go binding around an Ethereum contract.
type IWETH9Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IWETH9Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IWETH9Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IWETH9Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IWETH9Session struct {
	Contract     *IWETH9           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IWETH9CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IWETH9CallerSession struct {
	Contract *IWETH9Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// IWETH9TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IWETH9TransactorSession struct {
	Contract     *IWETH9Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IWETH9Raw is an auto generated low-level Go binding around an Ethereum contract.
type IWETH9Raw struct {
	Contract *IWETH9 // Generic contract binding to access the raw methods on
}

// IWETH9CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IWETH9CallerRaw struct {
	Contract *IWETH9Caller // Generic read-only contract binding to access the raw methods on
}

// IWETH9TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IWETH9TransactorRaw struct {
	Contract *IWETH9Transactor // Generic write-only contract binding to access the raw methods on
}

// NewIWETH9 creates a new instance of IWETH9, bound to a specific deployed contract.
func NewIWETH9(address common.Address, backend bind.ContractBackend) (*IWETH9, error) {
	contract, err := bindIWETH9(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IWETH9{IWETH9Caller: IWETH9Caller{contract: contract}, IWETH9Transactor: IWETH9Transactor{contract: contract}, IWETH9Filterer: IWETH9Filterer{contract: contract}}, nil
}

// NewIWETH9Caller creates a new read-only instance of IWETH9, bound to a specific deployed contract.
func NewIWETH9Caller(address common.Address, caller bind.ContractCaller) (*IWETH9Caller, error) {
	contract, err := bindIWETH9(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IWETH9Caller{contract: contract}, nil
}

// NewIWETH9Transactor creates a new write-only instance of IWETH9, bound to a specific deployed contract.
func NewIWETH9Transactor(address common.Address, transactor bind.ContractTransactor) (*IWETH9Transactor, error) {
	contract, err := bindIWETH9(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IWETH9Transactor{contract: contract}, nil
}

// NewIWETH9Filterer creates a new log filterer instance of IWETH9, bound to a specific deployed contract.
func NewIWETH9Filterer(address common.Address, filterer bind.ContractFilterer) (*IWETH9Filterer, error) {
	contract, err := bindIWETH9(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IWETH9Filterer{contract: contract}, nil
}

// bindIWETH9 binds a generic wrapper to an already deployed contract.
func bindIWETH9(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IWETH9MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IWETH9 *IWETH9Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IWETH9.Contract.IWETH9Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IWETH9 *IWETH9Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IWETH9.Contract.IWETH9Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IWETH9 *IWETH9Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IWETH9.Contract.IWETH9Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IWETH9 *IWETH9CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IWETH9.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IWETH9 *IWETH9TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IWETH9.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IWETH9 *IWETH9TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IWETH9.Contract.contract.Transact(opts, method, params...)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_IWETH9 *IWETH9Transactor) Deposit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IWETH9.contract.Transact(opts, "deposit")
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_IWETH9 *IWETH9Session) Deposit() (*types.Transaction, error) {
	return _IWETH9.Contract.Deposit(&_IWETH9.TransactOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_IWETH9 *IWETH9TransactorSession) Deposit() (*types.Transaction, error) {
	return _IWETH9.Contract.Deposit(&_IWETH9.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 _amount) returns()
func (_IWETH9 *IWETH9Transactor) Withdraw(opts *bind.TransactOpts, _amount *big.Int) (*types.Transaction, error) {
	return _IWETH9.contract.Transact(opts, "withdraw", _amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 _amount) returns()
func (_IWETH9 *IWETH9Session) Withdraw(_amount *big.Int) (*types.Transaction, error) {
	return _IWETH9.Contract.Withdraw(&_IWETH9.TransactOpts, _amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 _amount) returns()
func (_IWETH9 *IWETH9TransactorSession) Withdraw(_amount *big.Int) (*types.Transaction, error) {
	return _IWETH9.Contract.Withdraw(&_IWETH9.TransactOpts, _amount)
}

// InboxStubMetaData contains all meta data concerning the InboxStub contract.
var InboxStubMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"allowListEnabled\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"calculateRetryableSubmissionFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"createRetryableTicket\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"depositEth\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getProxyAdmin\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"_bridge\",\"type\":\"address\"},{\"internalType\":\"contractISequencerInbox\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isAllowed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxDataSize\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"_bridge\",\"type\":\"address\"}],\"name\":\"postUpgradeInit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendContractTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedContractTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedUnsignedTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedUnsignedTransactionToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"messageData\",\"type\":\"bytes\"}],\"name\":\"sendL2Message\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"messageData\",\"type\":\"bytes\"}],\"name\":\"sendL2MessageFromOrigin\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendUnsignedTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendUnsignedTransactionToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"sendWithdrawEthToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"contractISequencerInbox\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"},{\"internalType\":\"bool[]\",\"name\":\"\",\"type\":\"bool[]\"}],\"name\":\"setAllowList\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"name\":\"setAllowListEnabled\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"unsafeCreateRetryableTicket\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b506201cccc608052608051610dee61003360003960006103fb0152610dee6000f3fe6080604052600436106101a05760003560e01c80638456cb59116100e1578063c474d2c51161008a578063e78cea9211610064578063e78cea92146103bc578063e8eb1dc3146103e9578063ee35f3271461041d578063efeadb6d1461044a57600080fd5b8063c474d2c51461037e578063e3de72a51461039c578063e6bd12cf146102aa57600080fd5b8063a66b327d116100bb578063a66b327d14610328578063b75436bb14610343578063babcc5391461036357600080fd5b80638456cb591461021d5780638a631aa6146102d35780638b3240a0146102ee57600080fd5b80635075788b1161014e578063679b6ded11610128578063679b6ded1461029c57806367ef3ab8146102aa5780636e6e8a6a1461029c57806370665f14146102b857600080fd5b80635075788b146101a55780635c975abb1461025c5780635e9167581461028e57600080fd5b80633f4ba83a1161017f5780633f4ba83a1461021d578063439370b114610234578063485cc9551461023c57600080fd5b8062f72382146101a55780631fe927cf146101d857806322bd5c1c146101f8575b600080fd5b3480156101b157600080fd5b506101c56101c0366004610816565b610465565b6040519081526020015b60405180910390f35b3480156101e457600080fd5b506101c56101f3366004610893565b6104b5565b34801561020457600080fd5b5061020d610465565b60405190151581526020016101cf565b34801561022957600080fd5b50610232610560565b005b6101c5610465565b34801561024857600080fd5b506102326102573660046108d5565b6105a8565b34801561026857600080fd5b5060015461020d9074010000000000000000000000000000000000000000900460ff1681565b6101c56101c036600461090e565b6101c56101c0366004610978565b6101c56101c0366004610a1d565b3480156102c457600080fd5b506101c56101c0366004610a90565b3480156102df57600080fd5b506101c56101c0366004610add565b3480156102fa57600080fd5b50610303610465565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101cf565b34801561033457600080fd5b506101c56101c0366004610b32565b34801561034f57600080fd5b506101c561035e366004610893565b610656565b34801561036f57600080fd5b5061020d6101c0366004610b54565b34801561038a57600080fd5b50610232610399366004610b54565b50565b3480156103a857600080fd5b506102326103b7366004610c83565b6106b2565b3480156103c857600080fd5b506000546103039073ffffffffffffffffffffffffffffffffffffffff1681565b3480156103f557600080fd5b506101c57f000000000000000000000000000000000000000000000000000000000000000081565b34801561042957600080fd5b506001546103039073ffffffffffffffffffffffffffffffffffffffff1681565b34801561045657600080fd5b506102326103b7366004610d45565b60405162461bcd60e51b815260206004820152600f60248201527f4e4f545f494d504c454d454e544544000000000000000000000000000000000060448201526000906064015b60405180910390fd5b60003332146105065760405162461bcd60e51b815260206004820152600b60248201527f6f726967696e206f6e6c7900000000000000000000000000000000000000000060448201526064016104ac565b600061052b600333868660405161051e929190610d60565b60405180910390206106fa565b60405190915081907fab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c90600090a29392505050565b60405162461bcd60e51b815260206004820152600f60248201527f4e4f5420494d504c454d454e544544000000000000000000000000000000000060448201526064016104ac565b60005473ffffffffffffffffffffffffffffffffffffffff161561060e5760405162461bcd60e51b815260206004820152600c60248201527f414c52454144595f494e4954000000000000000000000000000000000000000060448201526064016104ac565b50600080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b60008061066f600333868660405161051e929190610d60565b9050807fff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b85856040516106a3929190610d70565b60405180910390a29392505050565b60405162461bcd60e51b815260206004820152600f60248201527f4e4f545f494d504c454d454e544544000000000000000000000000000000000060448201526064016104ac565b600080546040517f8db5993b00000000000000000000000000000000000000000000000000000000815260ff8616600482015273ffffffffffffffffffffffffffffffffffffffff85811660248301526044820185905290911690638db5993b90349060640160206040518083038185885af115801561077e573d6000803e3d6000fd5b50505050506040513d601f19601f820116820180604052508101906107a39190610d9f565b949350505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461039957600080fd5b60008083601f8401126107df57600080fd5b50813567ffffffffffffffff8111156107f757600080fd5b60208301915083602082850101111561080f57600080fd5b9250929050565b600080600080600080600060c0888a03121561083157600080fd5b8735965060208801359550604088013594506060880135610851816107ab565b93506080880135925060a088013567ffffffffffffffff81111561087457600080fd5b6108808a828b016107cd565b989b979a50959850939692959293505050565b600080602083850312156108a657600080fd5b823567ffffffffffffffff8111156108bd57600080fd5b6108c9858286016107cd565b90969095509350505050565b600080604083850312156108e857600080fd5b82356108f3816107ab565b91506020830135610903816107ab565b809150509250929050565b60008060008060006080868803121561092657600080fd5b8535945060208601359350604086013561093f816107ab565b9250606086013567ffffffffffffffff81111561095b57600080fd5b610967888289016107cd565b969995985093965092949392505050565b60008060008060008060008060006101008a8c03121561099757600080fd5b89356109a2816107ab565b985060208a0135975060408a0135965060608a01356109c0816107ab565b955060808a01356109d0816107ab565b945060a08a0135935060c08a0135925060e08a013567ffffffffffffffff8111156109fa57600080fd5b610a068c828d016107cd565b915080935050809150509295985092959850929598565b60008060008060008060a08789031215610a3657600080fd5b8635955060208701359450604087013593506060870135610a56816107ab565b9250608087013567ffffffffffffffff811115610a7257600080fd5b610a7e89828a016107cd565b979a9699509497509295939492505050565b600080600080600060a08688031215610aa857600080fd5b853594506020860135935060408601359250606086013591506080860135610acf816107ab565b809150509295509295909350565b60008060008060008060a08789031215610af657600080fd5b86359550602087013594506040870135610b0f816107ab565b935060608701359250608087013567ffffffffffffffff811115610a7257600080fd5b60008060408385031215610b4557600080fd5b50508035926020909101359150565b600060208284031215610b6657600080fd5b8135610b71816107ab565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604051601f8201601f1916810167ffffffffffffffff81118282101715610bd057610bd0610b78565b604052919050565b600067ffffffffffffffff821115610bf257610bf2610b78565b5060051b60200190565b80358015158114610c0c57600080fd5b919050565b600082601f830112610c2257600080fd5b81356020610c37610c3283610bd8565b610ba7565b82815260059290921b84018101918181019086841115610c5657600080fd5b8286015b84811015610c7857610c6b81610bfc565b8352918301918301610c5a565b509695505050505050565b60008060408385031215610c9657600080fd5b823567ffffffffffffffff80821115610cae57600080fd5b818501915085601f830112610cc257600080fd5b81356020610cd2610c3283610bd8565b82815260059290921b84018101918181019089841115610cf157600080fd5b948201945b83861015610d18578535610d09816107ab565b82529482019490820190610cf6565b96505086013592505080821115610d2e57600080fd5b50610d3b85828601610c11565b9150509250929050565b600060208284031215610d5757600080fd5b610b7182610bfc565b8183823760009101908152919050565b60208152816020820152818360408301376000818301604090810191909152601f909201601f19160101919050565b600060208284031215610db157600080fd5b505191905056fea2646970667358221220b8cb682595ef29ccca9ed25f4047d14df285855aa0ab1b091e7f99a4f7a99ce964736f6c63430008110033",
}

// InboxStubABI is the input ABI used to generate the binding from.
// Deprecated: Use InboxStubMetaData.ABI instead.
var InboxStubABI = InboxStubMetaData.ABI

// InboxStubBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use InboxStubMetaData.Bin instead.
var InboxStubBin = InboxStubMetaData.Bin

// DeployInboxStub deploys a new Ethereum contract, binding an instance of InboxStub to it.
func DeployInboxStub(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *InboxStub, error) {
	parsed, err := InboxStubMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(InboxStubBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &InboxStub{InboxStubCaller: InboxStubCaller{contract: contract}, InboxStubTransactor: InboxStubTransactor{contract: contract}, InboxStubFilterer: InboxStubFilterer{contract: contract}}, nil
}

// InboxStub is an auto generated Go binding around an Ethereum contract.
type InboxStub struct {
	InboxStubCaller     // Read-only binding to the contract
	InboxStubTransactor // Write-only binding to the contract
	InboxStubFilterer   // Log filterer for contract events
}

// InboxStubCaller is an auto generated read-only Go binding around an Ethereum contract.
type InboxStubCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InboxStubTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InboxStubTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InboxStubFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InboxStubFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InboxStubSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InboxStubSession struct {
	Contract     *InboxStub        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InboxStubCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InboxStubCallerSession struct {
	Contract *InboxStubCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// InboxStubTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InboxStubTransactorSession struct {
	Contract     *InboxStubTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// InboxStubRaw is an auto generated low-level Go binding around an Ethereum contract.
type InboxStubRaw struct {
	Contract *InboxStub // Generic contract binding to access the raw methods on
}

// InboxStubCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InboxStubCallerRaw struct {
	Contract *InboxStubCaller // Generic read-only contract binding to access the raw methods on
}

// InboxStubTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InboxStubTransactorRaw struct {
	Contract *InboxStubTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInboxStub creates a new instance of InboxStub, bound to a specific deployed contract.
func NewInboxStub(address common.Address, backend bind.ContractBackend) (*InboxStub, error) {
	contract, err := bindInboxStub(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &InboxStub{InboxStubCaller: InboxStubCaller{contract: contract}, InboxStubTransactor: InboxStubTransactor{contract: contract}, InboxStubFilterer: InboxStubFilterer{contract: contract}}, nil
}

// NewInboxStubCaller creates a new read-only instance of InboxStub, bound to a specific deployed contract.
func NewInboxStubCaller(address common.Address, caller bind.ContractCaller) (*InboxStubCaller, error) {
	contract, err := bindInboxStub(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InboxStubCaller{contract: contract}, nil
}

// NewInboxStubTransactor creates a new write-only instance of InboxStub, bound to a specific deployed contract.
func NewInboxStubTransactor(address common.Address, transactor bind.ContractTransactor) (*InboxStubTransactor, error) {
	contract, err := bindInboxStub(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InboxStubTransactor{contract: contract}, nil
}

// NewInboxStubFilterer creates a new log filterer instance of InboxStub, bound to a specific deployed contract.
func NewInboxStubFilterer(address common.Address, filterer bind.ContractFilterer) (*InboxStubFilterer, error) {
	contract, err := bindInboxStub(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InboxStubFilterer{contract: contract}, nil
}

// bindInboxStub binds a generic wrapper to an already deployed contract.
func bindInboxStub(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := InboxStubMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InboxStub *InboxStubRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InboxStub.Contract.InboxStubCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InboxStub *InboxStubRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InboxStub.Contract.InboxStubTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InboxStub *InboxStubRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InboxStub.Contract.InboxStubTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InboxStub *InboxStubCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InboxStub.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InboxStub *InboxStubTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InboxStub.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InboxStub *InboxStubTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InboxStub.Contract.contract.Transact(opts, method, params...)
}

// AllowListEnabled is a free data retrieval call binding the contract method 0x22bd5c1c.
//
// Solidity: function allowListEnabled() pure returns(bool)
func (_InboxStub *InboxStubCaller) AllowListEnabled(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "allowListEnabled")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowListEnabled is a free data retrieval call binding the contract method 0x22bd5c1c.
//
// Solidity: function allowListEnabled() pure returns(bool)
func (_InboxStub *InboxStubSession) AllowListEnabled() (bool, error) {
	return _InboxStub.Contract.AllowListEnabled(&_InboxStub.CallOpts)
}

// AllowListEnabled is a free data retrieval call binding the contract method 0x22bd5c1c.
//
// Solidity: function allowListEnabled() pure returns(bool)
func (_InboxStub *InboxStubCallerSession) AllowListEnabled() (bool, error) {
	return _InboxStub.Contract.AllowListEnabled(&_InboxStub.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_InboxStub *InboxStubCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_InboxStub *InboxStubSession) Bridge() (common.Address, error) {
	return _InboxStub.Contract.Bridge(&_InboxStub.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_InboxStub *InboxStubCallerSession) Bridge() (common.Address, error) {
	return _InboxStub.Contract.Bridge(&_InboxStub.CallOpts)
}

// CalculateRetryableSubmissionFee is a free data retrieval call binding the contract method 0xa66b327d.
//
// Solidity: function calculateRetryableSubmissionFee(uint256 , uint256 ) pure returns(uint256)
func (_InboxStub *InboxStubCaller) CalculateRetryableSubmissionFee(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "calculateRetryableSubmissionFee", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CalculateRetryableSubmissionFee is a free data retrieval call binding the contract method 0xa66b327d.
//
// Solidity: function calculateRetryableSubmissionFee(uint256 , uint256 ) pure returns(uint256)
func (_InboxStub *InboxStubSession) CalculateRetryableSubmissionFee(arg0 *big.Int, arg1 *big.Int) (*big.Int, error) {
	return _InboxStub.Contract.CalculateRetryableSubmissionFee(&_InboxStub.CallOpts, arg0, arg1)
}

// CalculateRetryableSubmissionFee is a free data retrieval call binding the contract method 0xa66b327d.
//
// Solidity: function calculateRetryableSubmissionFee(uint256 , uint256 ) pure returns(uint256)
func (_InboxStub *InboxStubCallerSession) CalculateRetryableSubmissionFee(arg0 *big.Int, arg1 *big.Int) (*big.Int, error) {
	return _InboxStub.Contract.CalculateRetryableSubmissionFee(&_InboxStub.CallOpts, arg0, arg1)
}

// GetProxyAdmin is a free data retrieval call binding the contract method 0x8b3240a0.
//
// Solidity: function getProxyAdmin() pure returns(address)
func (_InboxStub *InboxStubCaller) GetProxyAdmin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "getProxyAdmin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetProxyAdmin is a free data retrieval call binding the contract method 0x8b3240a0.
//
// Solidity: function getProxyAdmin() pure returns(address)
func (_InboxStub *InboxStubSession) GetProxyAdmin() (common.Address, error) {
	return _InboxStub.Contract.GetProxyAdmin(&_InboxStub.CallOpts)
}

// GetProxyAdmin is a free data retrieval call binding the contract method 0x8b3240a0.
//
// Solidity: function getProxyAdmin() pure returns(address)
func (_InboxStub *InboxStubCallerSession) GetProxyAdmin() (common.Address, error) {
	return _InboxStub.Contract.GetProxyAdmin(&_InboxStub.CallOpts)
}

// IsAllowed is a free data retrieval call binding the contract method 0xbabcc539.
//
// Solidity: function isAllowed(address ) pure returns(bool)
func (_InboxStub *InboxStubCaller) IsAllowed(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "isAllowed", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAllowed is a free data retrieval call binding the contract method 0xbabcc539.
//
// Solidity: function isAllowed(address ) pure returns(bool)
func (_InboxStub *InboxStubSession) IsAllowed(arg0 common.Address) (bool, error) {
	return _InboxStub.Contract.IsAllowed(&_InboxStub.CallOpts, arg0)
}

// IsAllowed is a free data retrieval call binding the contract method 0xbabcc539.
//
// Solidity: function isAllowed(address ) pure returns(bool)
func (_InboxStub *InboxStubCallerSession) IsAllowed(arg0 common.Address) (bool, error) {
	return _InboxStub.Contract.IsAllowed(&_InboxStub.CallOpts, arg0)
}

// MaxDataSize is a free data retrieval call binding the contract method 0xe8eb1dc3.
//
// Solidity: function maxDataSize() view returns(uint256)
func (_InboxStub *InboxStubCaller) MaxDataSize(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "maxDataSize")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxDataSize is a free data retrieval call binding the contract method 0xe8eb1dc3.
//
// Solidity: function maxDataSize() view returns(uint256)
func (_InboxStub *InboxStubSession) MaxDataSize() (*big.Int, error) {
	return _InboxStub.Contract.MaxDataSize(&_InboxStub.CallOpts)
}

// MaxDataSize is a free data retrieval call binding the contract method 0xe8eb1dc3.
//
// Solidity: function maxDataSize() view returns(uint256)
func (_InboxStub *InboxStubCallerSession) MaxDataSize() (*big.Int, error) {
	return _InboxStub.Contract.MaxDataSize(&_InboxStub.CallOpts)
}

// Pause is a free data retrieval call binding the contract method 0x8456cb59.
//
// Solidity: function pause() pure returns()
func (_InboxStub *InboxStubCaller) Pause(opts *bind.CallOpts) error {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "pause")

	if err != nil {
		return err
	}

	return err

}

// Pause is a free data retrieval call binding the contract method 0x8456cb59.
//
// Solidity: function pause() pure returns()
func (_InboxStub *InboxStubSession) Pause() error {
	return _InboxStub.Contract.Pause(&_InboxStub.CallOpts)
}

// Pause is a free data retrieval call binding the contract method 0x8456cb59.
//
// Solidity: function pause() pure returns()
func (_InboxStub *InboxStubCallerSession) Pause() error {
	return _InboxStub.Contract.Pause(&_InboxStub.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_InboxStub *InboxStubCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_InboxStub *InboxStubSession) Paused() (bool, error) {
	return _InboxStub.Contract.Paused(&_InboxStub.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_InboxStub *InboxStubCallerSession) Paused() (bool, error) {
	return _InboxStub.Contract.Paused(&_InboxStub.CallOpts)
}

// SendContractTransaction is a free data retrieval call binding the contract method 0x8a631aa6.
//
// Solidity: function sendContractTransaction(uint256 , uint256 , address , uint256 , bytes ) pure returns(uint256)
func (_InboxStub *InboxStubCaller) SendContractTransaction(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int, arg2 common.Address, arg3 *big.Int, arg4 []byte) (*big.Int, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "sendContractTransaction", arg0, arg1, arg2, arg3, arg4)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SendContractTransaction is a free data retrieval call binding the contract method 0x8a631aa6.
//
// Solidity: function sendContractTransaction(uint256 , uint256 , address , uint256 , bytes ) pure returns(uint256)
func (_InboxStub *InboxStubSession) SendContractTransaction(arg0 *big.Int, arg1 *big.Int, arg2 common.Address, arg3 *big.Int, arg4 []byte) (*big.Int, error) {
	return _InboxStub.Contract.SendContractTransaction(&_InboxStub.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// SendContractTransaction is a free data retrieval call binding the contract method 0x8a631aa6.
//
// Solidity: function sendContractTransaction(uint256 , uint256 , address , uint256 , bytes ) pure returns(uint256)
func (_InboxStub *InboxStubCallerSession) SendContractTransaction(arg0 *big.Int, arg1 *big.Int, arg2 common.Address, arg3 *big.Int, arg4 []byte) (*big.Int, error) {
	return _InboxStub.Contract.SendContractTransaction(&_InboxStub.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// SendUnsignedTransaction is a free data retrieval call binding the contract method 0x5075788b.
//
// Solidity: function sendUnsignedTransaction(uint256 , uint256 , uint256 , address , uint256 , bytes ) pure returns(uint256)
func (_InboxStub *InboxStubCaller) SendUnsignedTransaction(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*big.Int, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "sendUnsignedTransaction", arg0, arg1, arg2, arg3, arg4, arg5)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SendUnsignedTransaction is a free data retrieval call binding the contract method 0x5075788b.
//
// Solidity: function sendUnsignedTransaction(uint256 , uint256 , uint256 , address , uint256 , bytes ) pure returns(uint256)
func (_InboxStub *InboxStubSession) SendUnsignedTransaction(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*big.Int, error) {
	return _InboxStub.Contract.SendUnsignedTransaction(&_InboxStub.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// SendUnsignedTransaction is a free data retrieval call binding the contract method 0x5075788b.
//
// Solidity: function sendUnsignedTransaction(uint256 , uint256 , uint256 , address , uint256 , bytes ) pure returns(uint256)
func (_InboxStub *InboxStubCallerSession) SendUnsignedTransaction(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*big.Int, error) {
	return _InboxStub.Contract.SendUnsignedTransaction(&_InboxStub.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// SendUnsignedTransactionToFork is a free data retrieval call binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 , uint256 , uint256 , address , uint256 , bytes ) pure returns(uint256)
func (_InboxStub *InboxStubCaller) SendUnsignedTransactionToFork(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*big.Int, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "sendUnsignedTransactionToFork", arg0, arg1, arg2, arg3, arg4, arg5)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SendUnsignedTransactionToFork is a free data retrieval call binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 , uint256 , uint256 , address , uint256 , bytes ) pure returns(uint256)
func (_InboxStub *InboxStubSession) SendUnsignedTransactionToFork(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*big.Int, error) {
	return _InboxStub.Contract.SendUnsignedTransactionToFork(&_InboxStub.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// SendUnsignedTransactionToFork is a free data retrieval call binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 , uint256 , uint256 , address , uint256 , bytes ) pure returns(uint256)
func (_InboxStub *InboxStubCallerSession) SendUnsignedTransactionToFork(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*big.Int, error) {
	return _InboxStub.Contract.SendUnsignedTransactionToFork(&_InboxStub.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// SendWithdrawEthToFork is a free data retrieval call binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 , uint256 , uint256 , uint256 , address ) pure returns(uint256)
func (_InboxStub *InboxStubCaller) SendWithdrawEthToFork(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 *big.Int, arg4 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "sendWithdrawEthToFork", arg0, arg1, arg2, arg3, arg4)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SendWithdrawEthToFork is a free data retrieval call binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 , uint256 , uint256 , uint256 , address ) pure returns(uint256)
func (_InboxStub *InboxStubSession) SendWithdrawEthToFork(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 *big.Int, arg4 common.Address) (*big.Int, error) {
	return _InboxStub.Contract.SendWithdrawEthToFork(&_InboxStub.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// SendWithdrawEthToFork is a free data retrieval call binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 , uint256 , uint256 , uint256 , address ) pure returns(uint256)
func (_InboxStub *InboxStubCallerSession) SendWithdrawEthToFork(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 *big.Int, arg4 common.Address) (*big.Int, error) {
	return _InboxStub.Contract.SendWithdrawEthToFork(&_InboxStub.CallOpts, arg0, arg1, arg2, arg3, arg4)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_InboxStub *InboxStubCaller) SequencerInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "sequencerInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_InboxStub *InboxStubSession) SequencerInbox() (common.Address, error) {
	return _InboxStub.Contract.SequencerInbox(&_InboxStub.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_InboxStub *InboxStubCallerSession) SequencerInbox() (common.Address, error) {
	return _InboxStub.Contract.SequencerInbox(&_InboxStub.CallOpts)
}

// SetAllowList is a free data retrieval call binding the contract method 0xe3de72a5.
//
// Solidity: function setAllowList(address[] , bool[] ) pure returns()
func (_InboxStub *InboxStubCaller) SetAllowList(opts *bind.CallOpts, arg0 []common.Address, arg1 []bool) error {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "setAllowList", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// SetAllowList is a free data retrieval call binding the contract method 0xe3de72a5.
//
// Solidity: function setAllowList(address[] , bool[] ) pure returns()
func (_InboxStub *InboxStubSession) SetAllowList(arg0 []common.Address, arg1 []bool) error {
	return _InboxStub.Contract.SetAllowList(&_InboxStub.CallOpts, arg0, arg1)
}

// SetAllowList is a free data retrieval call binding the contract method 0xe3de72a5.
//
// Solidity: function setAllowList(address[] , bool[] ) pure returns()
func (_InboxStub *InboxStubCallerSession) SetAllowList(arg0 []common.Address, arg1 []bool) error {
	return _InboxStub.Contract.SetAllowList(&_InboxStub.CallOpts, arg0, arg1)
}

// SetAllowListEnabled is a free data retrieval call binding the contract method 0xefeadb6d.
//
// Solidity: function setAllowListEnabled(bool ) pure returns()
func (_InboxStub *InboxStubCaller) SetAllowListEnabled(opts *bind.CallOpts, arg0 bool) error {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "setAllowListEnabled", arg0)

	if err != nil {
		return err
	}

	return err

}

// SetAllowListEnabled is a free data retrieval call binding the contract method 0xefeadb6d.
//
// Solidity: function setAllowListEnabled(bool ) pure returns()
func (_InboxStub *InboxStubSession) SetAllowListEnabled(arg0 bool) error {
	return _InboxStub.Contract.SetAllowListEnabled(&_InboxStub.CallOpts, arg0)
}

// SetAllowListEnabled is a free data retrieval call binding the contract method 0xefeadb6d.
//
// Solidity: function setAllowListEnabled(bool ) pure returns()
func (_InboxStub *InboxStubCallerSession) SetAllowListEnabled(arg0 bool) error {
	return _InboxStub.Contract.SetAllowListEnabled(&_InboxStub.CallOpts, arg0)
}

// Unpause is a free data retrieval call binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() pure returns()
func (_InboxStub *InboxStubCaller) Unpause(opts *bind.CallOpts) error {
	var out []interface{}
	err := _InboxStub.contract.Call(opts, &out, "unpause")

	if err != nil {
		return err
	}

	return err

}

// Unpause is a free data retrieval call binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() pure returns()
func (_InboxStub *InboxStubSession) Unpause() error {
	return _InboxStub.Contract.Unpause(&_InboxStub.CallOpts)
}

// Unpause is a free data retrieval call binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() pure returns()
func (_InboxStub *InboxStubCallerSession) Unpause() error {
	return _InboxStub.Contract.Unpause(&_InboxStub.CallOpts)
}

// CreateRetryableTicket is a paid mutator transaction binding the contract method 0x679b6ded.
//
// Solidity: function createRetryableTicket(address , uint256 , uint256 , address , address , uint256 , uint256 , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubTransactor) CreateRetryableTicket(opts *bind.TransactOpts, arg0 common.Address, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 common.Address, arg5 *big.Int, arg6 *big.Int, arg7 []byte) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "createRetryableTicket", arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
}

// CreateRetryableTicket is a paid mutator transaction binding the contract method 0x679b6ded.
//
// Solidity: function createRetryableTicket(address , uint256 , uint256 , address , address , uint256 , uint256 , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubSession) CreateRetryableTicket(arg0 common.Address, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 common.Address, arg5 *big.Int, arg6 *big.Int, arg7 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.CreateRetryableTicket(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
}

// CreateRetryableTicket is a paid mutator transaction binding the contract method 0x679b6ded.
//
// Solidity: function createRetryableTicket(address , uint256 , uint256 , address , address , uint256 , uint256 , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubTransactorSession) CreateRetryableTicket(arg0 common.Address, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 common.Address, arg5 *big.Int, arg6 *big.Int, arg7 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.CreateRetryableTicket(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
}

// DepositEth is a paid mutator transaction binding the contract method 0x439370b1.
//
// Solidity: function depositEth() payable returns(uint256)
func (_InboxStub *InboxStubTransactor) DepositEth(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "depositEth")
}

// DepositEth is a paid mutator transaction binding the contract method 0x439370b1.
//
// Solidity: function depositEth() payable returns(uint256)
func (_InboxStub *InboxStubSession) DepositEth() (*types.Transaction, error) {
	return _InboxStub.Contract.DepositEth(&_InboxStub.TransactOpts)
}

// DepositEth is a paid mutator transaction binding the contract method 0x439370b1.
//
// Solidity: function depositEth() payable returns(uint256)
func (_InboxStub *InboxStubTransactorSession) DepositEth() (*types.Transaction, error) {
	return _InboxStub.Contract.DepositEth(&_InboxStub.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _bridge, address ) returns()
func (_InboxStub *InboxStubTransactor) Initialize(opts *bind.TransactOpts, _bridge common.Address, arg1 common.Address) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "initialize", _bridge, arg1)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _bridge, address ) returns()
func (_InboxStub *InboxStubSession) Initialize(_bridge common.Address, arg1 common.Address) (*types.Transaction, error) {
	return _InboxStub.Contract.Initialize(&_InboxStub.TransactOpts, _bridge, arg1)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _bridge, address ) returns()
func (_InboxStub *InboxStubTransactorSession) Initialize(_bridge common.Address, arg1 common.Address) (*types.Transaction, error) {
	return _InboxStub.Contract.Initialize(&_InboxStub.TransactOpts, _bridge, arg1)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xc474d2c5.
//
// Solidity: function postUpgradeInit(address _bridge) returns()
func (_InboxStub *InboxStubTransactor) PostUpgradeInit(opts *bind.TransactOpts, _bridge common.Address) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "postUpgradeInit", _bridge)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xc474d2c5.
//
// Solidity: function postUpgradeInit(address _bridge) returns()
func (_InboxStub *InboxStubSession) PostUpgradeInit(_bridge common.Address) (*types.Transaction, error) {
	return _InboxStub.Contract.PostUpgradeInit(&_InboxStub.TransactOpts, _bridge)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xc474d2c5.
//
// Solidity: function postUpgradeInit(address _bridge) returns()
func (_InboxStub *InboxStubTransactorSession) PostUpgradeInit(_bridge common.Address) (*types.Transaction, error) {
	return _InboxStub.Contract.PostUpgradeInit(&_InboxStub.TransactOpts, _bridge)
}

// SendL1FundedContractTransaction is a paid mutator transaction binding the contract method 0x5e916758.
//
// Solidity: function sendL1FundedContractTransaction(uint256 , uint256 , address , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubTransactor) SendL1FundedContractTransaction(opts *bind.TransactOpts, arg0 *big.Int, arg1 *big.Int, arg2 common.Address, arg3 []byte) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "sendL1FundedContractTransaction", arg0, arg1, arg2, arg3)
}

// SendL1FundedContractTransaction is a paid mutator transaction binding the contract method 0x5e916758.
//
// Solidity: function sendL1FundedContractTransaction(uint256 , uint256 , address , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubSession) SendL1FundedContractTransaction(arg0 *big.Int, arg1 *big.Int, arg2 common.Address, arg3 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendL1FundedContractTransaction(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3)
}

// SendL1FundedContractTransaction is a paid mutator transaction binding the contract method 0x5e916758.
//
// Solidity: function sendL1FundedContractTransaction(uint256 , uint256 , address , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubTransactorSession) SendL1FundedContractTransaction(arg0 *big.Int, arg1 *big.Int, arg2 common.Address, arg3 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendL1FundedContractTransaction(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3)
}

// SendL1FundedUnsignedTransaction is a paid mutator transaction binding the contract method 0x67ef3ab8.
//
// Solidity: function sendL1FundedUnsignedTransaction(uint256 , uint256 , uint256 , address , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubTransactor) SendL1FundedUnsignedTransaction(opts *bind.TransactOpts, arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 []byte) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "sendL1FundedUnsignedTransaction", arg0, arg1, arg2, arg3, arg4)
}

// SendL1FundedUnsignedTransaction is a paid mutator transaction binding the contract method 0x67ef3ab8.
//
// Solidity: function sendL1FundedUnsignedTransaction(uint256 , uint256 , uint256 , address , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubSession) SendL1FundedUnsignedTransaction(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendL1FundedUnsignedTransaction(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4)
}

// SendL1FundedUnsignedTransaction is a paid mutator transaction binding the contract method 0x67ef3ab8.
//
// Solidity: function sendL1FundedUnsignedTransaction(uint256 , uint256 , uint256 , address , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubTransactorSession) SendL1FundedUnsignedTransaction(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendL1FundedUnsignedTransaction(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4)
}

// SendL1FundedUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0xe6bd12cf.
//
// Solidity: function sendL1FundedUnsignedTransactionToFork(uint256 , uint256 , uint256 , address , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubTransactor) SendL1FundedUnsignedTransactionToFork(opts *bind.TransactOpts, arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 []byte) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "sendL1FundedUnsignedTransactionToFork", arg0, arg1, arg2, arg3, arg4)
}

// SendL1FundedUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0xe6bd12cf.
//
// Solidity: function sendL1FundedUnsignedTransactionToFork(uint256 , uint256 , uint256 , address , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubSession) SendL1FundedUnsignedTransactionToFork(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendL1FundedUnsignedTransactionToFork(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4)
}

// SendL1FundedUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0xe6bd12cf.
//
// Solidity: function sendL1FundedUnsignedTransactionToFork(uint256 , uint256 , uint256 , address , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubTransactorSession) SendL1FundedUnsignedTransactionToFork(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendL1FundedUnsignedTransactionToFork(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4)
}

// SendL2Message is a paid mutator transaction binding the contract method 0xb75436bb.
//
// Solidity: function sendL2Message(bytes messageData) returns(uint256)
func (_InboxStub *InboxStubTransactor) SendL2Message(opts *bind.TransactOpts, messageData []byte) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "sendL2Message", messageData)
}

// SendL2Message is a paid mutator transaction binding the contract method 0xb75436bb.
//
// Solidity: function sendL2Message(bytes messageData) returns(uint256)
func (_InboxStub *InboxStubSession) SendL2Message(messageData []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendL2Message(&_InboxStub.TransactOpts, messageData)
}

// SendL2Message is a paid mutator transaction binding the contract method 0xb75436bb.
//
// Solidity: function sendL2Message(bytes messageData) returns(uint256)
func (_InboxStub *InboxStubTransactorSession) SendL2Message(messageData []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendL2Message(&_InboxStub.TransactOpts, messageData)
}

// SendL2MessageFromOrigin is a paid mutator transaction binding the contract method 0x1fe927cf.
//
// Solidity: function sendL2MessageFromOrigin(bytes messageData) returns(uint256)
func (_InboxStub *InboxStubTransactor) SendL2MessageFromOrigin(opts *bind.TransactOpts, messageData []byte) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "sendL2MessageFromOrigin", messageData)
}

// SendL2MessageFromOrigin is a paid mutator transaction binding the contract method 0x1fe927cf.
//
// Solidity: function sendL2MessageFromOrigin(bytes messageData) returns(uint256)
func (_InboxStub *InboxStubSession) SendL2MessageFromOrigin(messageData []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendL2MessageFromOrigin(&_InboxStub.TransactOpts, messageData)
}

// SendL2MessageFromOrigin is a paid mutator transaction binding the contract method 0x1fe927cf.
//
// Solidity: function sendL2MessageFromOrigin(bytes messageData) returns(uint256)
func (_InboxStub *InboxStubTransactorSession) SendL2MessageFromOrigin(messageData []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendL2MessageFromOrigin(&_InboxStub.TransactOpts, messageData)
}

// UnsafeCreateRetryableTicket is a paid mutator transaction binding the contract method 0x6e6e8a6a.
//
// Solidity: function unsafeCreateRetryableTicket(address , uint256 , uint256 , address , address , uint256 , uint256 , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubTransactor) UnsafeCreateRetryableTicket(opts *bind.TransactOpts, arg0 common.Address, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 common.Address, arg5 *big.Int, arg6 *big.Int, arg7 []byte) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "unsafeCreateRetryableTicket", arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
}

// UnsafeCreateRetryableTicket is a paid mutator transaction binding the contract method 0x6e6e8a6a.
//
// Solidity: function unsafeCreateRetryableTicket(address , uint256 , uint256 , address , address , uint256 , uint256 , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubSession) UnsafeCreateRetryableTicket(arg0 common.Address, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 common.Address, arg5 *big.Int, arg6 *big.Int, arg7 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.UnsafeCreateRetryableTicket(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
}

// UnsafeCreateRetryableTicket is a paid mutator transaction binding the contract method 0x6e6e8a6a.
//
// Solidity: function unsafeCreateRetryableTicket(address , uint256 , uint256 , address , address , uint256 , uint256 , bytes ) payable returns(uint256)
func (_InboxStub *InboxStubTransactorSession) UnsafeCreateRetryableTicket(arg0 common.Address, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 common.Address, arg5 *big.Int, arg6 *big.Int, arg7 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.UnsafeCreateRetryableTicket(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
}

// InboxStubInboxMessageDeliveredIterator is returned from FilterInboxMessageDelivered and is used to iterate over the raw logs and unpacked data for InboxMessageDelivered events raised by the InboxStub contract.
type InboxStubInboxMessageDeliveredIterator struct {
	Event *InboxStubInboxMessageDelivered // Event containing the contract specifics and raw log

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
func (it *InboxStubInboxMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InboxStubInboxMessageDelivered)
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
		it.Event = new(InboxStubInboxMessageDelivered)
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
func (it *InboxStubInboxMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InboxStubInboxMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InboxStubInboxMessageDelivered represents a InboxMessageDelivered event raised by the InboxStub contract.
type InboxStubInboxMessageDelivered struct {
	MessageNum *big.Int
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDelivered is a free log retrieval operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_InboxStub *InboxStubFilterer) FilterInboxMessageDelivered(opts *bind.FilterOpts, messageNum []*big.Int) (*InboxStubInboxMessageDeliveredIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _InboxStub.contract.FilterLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &InboxStubInboxMessageDeliveredIterator{contract: _InboxStub.contract, event: "InboxMessageDelivered", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDelivered is a free log subscription operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_InboxStub *InboxStubFilterer) WatchInboxMessageDelivered(opts *bind.WatchOpts, sink chan<- *InboxStubInboxMessageDelivered, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _InboxStub.contract.WatchLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InboxStubInboxMessageDelivered)
				if err := _InboxStub.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
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

// ParseInboxMessageDelivered is a log parse operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_InboxStub *InboxStubFilterer) ParseInboxMessageDelivered(log types.Log) (*InboxStubInboxMessageDelivered, error) {
	event := new(InboxStubInboxMessageDelivered)
	if err := _InboxStub.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InboxStubInboxMessageDeliveredFromOriginIterator is returned from FilterInboxMessageDeliveredFromOrigin and is used to iterate over the raw logs and unpacked data for InboxMessageDeliveredFromOrigin events raised by the InboxStub contract.
type InboxStubInboxMessageDeliveredFromOriginIterator struct {
	Event *InboxStubInboxMessageDeliveredFromOrigin // Event containing the contract specifics and raw log

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
func (it *InboxStubInboxMessageDeliveredFromOriginIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InboxStubInboxMessageDeliveredFromOrigin)
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
		it.Event = new(InboxStubInboxMessageDeliveredFromOrigin)
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
func (it *InboxStubInboxMessageDeliveredFromOriginIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InboxStubInboxMessageDeliveredFromOriginIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InboxStubInboxMessageDeliveredFromOrigin represents a InboxMessageDeliveredFromOrigin event raised by the InboxStub contract.
type InboxStubInboxMessageDeliveredFromOrigin struct {
	MessageNum *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDeliveredFromOrigin is a free log retrieval operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_InboxStub *InboxStubFilterer) FilterInboxMessageDeliveredFromOrigin(opts *bind.FilterOpts, messageNum []*big.Int) (*InboxStubInboxMessageDeliveredFromOriginIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _InboxStub.contract.FilterLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &InboxStubInboxMessageDeliveredFromOriginIterator{contract: _InboxStub.contract, event: "InboxMessageDeliveredFromOrigin", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDeliveredFromOrigin is a free log subscription operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_InboxStub *InboxStubFilterer) WatchInboxMessageDeliveredFromOrigin(opts *bind.WatchOpts, sink chan<- *InboxStubInboxMessageDeliveredFromOrigin, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _InboxStub.contract.WatchLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InboxStubInboxMessageDeliveredFromOrigin)
				if err := _InboxStub.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
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

// ParseInboxMessageDeliveredFromOrigin is a log parse operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_InboxStub *InboxStubFilterer) ParseInboxMessageDeliveredFromOrigin(log types.Log) (*InboxStubInboxMessageDeliveredFromOrigin, error) {
	event := new(InboxStubInboxMessageDeliveredFromOrigin)
	if err := _InboxStub.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MerkleTreeAccessMetaData contains all meta data concerning the MerkleTreeAccess contract.
var MerkleTreeAccessMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"actualLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxProofLength\",\"type\":\"uint256\"}],\"name\":\"MerkleProofTooLong\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"me\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"level\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"subtreeRoot\",\"type\":\"bytes32\"}],\"name\":\"appendCompleteSubTree\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"me\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32\",\"name\":\"leaf\",\"type\":\"bytes32\"}],\"name\":\"appendLeaf\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"leastSignificantBit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"startSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"endSize\",\"type\":\"uint256\"}],\"name\":\"maximumAppendBetween\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"mostSignificantBit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"me\",\"type\":\"bytes32[]\"}],\"name\":\"root\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"leaf\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"}],\"name\":\"verifyInclusionProof\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"preRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"preSize\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"postRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"postSize\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"preExpansion\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"}],\"name\":\"verifyPrefixProof\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506116b8806100206000396000f3fe608060405234801561001057600080fd5b50600436106100885760003560e01c8063c22c47a41161005b578063c22c47a4146100ff578063ca11325314610112578063d230d23f14610125578063e6bcbc651461013857600080fd5b80635fb9c3d41461008d57806367905a7e146100a25780636bd58993146100cb578063bc2f0640146100de575b600080fd5b6100a061009b3660046112de565b61014b565b005b6100b56100b0366004611367565b610161565b6040516100c291906113b5565b60405180910390f35b6100a06100d93660046113f9565b610178565b6100f16100ec366004611453565b61018a565b6040519081526020016100c2565b6100b561010d366004611475565b61019f565b6100f16101203660046114ba565b6101ab565b6100f16101333660046114f7565b6101b6565b6100f16101463660046114f7565b6101c1565b6101598686868686866101cc565b505050505050565b606061016e8484846104f4565b90505b9392505050565b61018484848484610a7e565b50505050565b60006101968383610b0b565b90505b92915050565b60606101968383610c00565b600061019982610c36565b600061019982610dd6565b600061019982610e3f565b600085116102215760405162461bcd60e51b815260206004820152601460248201527f5072652d73697a652063616e6e6f74206265203000000000000000000000000060448201526064015b60405180910390fd5b8561022b83610c36565b146102785760405162461bcd60e51b815260206004820152601b60248201527f50726520657870616e73696f6e20726f6f74206d69736d6174636800000000006044820152606401610218565b8461028283610f85565b146102f55760405162461bcd60e51b815260206004820152602160248201527f5072652073697a6520646f6573206e6f74206d6174636820657870616e73696f60448201527f6e000000000000000000000000000000000000000000000000000000000000006064820152608401610218565b8285106103445760405162461bcd60e51b815260206004820181905260248201527f5072652073697a65206e6f74206c657373207468616e20706f73742073697a656044820152606401610218565b60008590506000806103598560008751610fe0565b90505b8583101561041c5760006103708488610b0b565b9050845183106103c25760405162461bcd60e51b815260206004820152601260248201527f496e646578206f7574206f662072616e676500000000000000000000000000006044820152606401610218565b6103e682828786815181106103d9576103d9611510565b60200260200101516104f4565b91506001811b6103f6818661153c565b9450878511156104085761040861154f565b8361041281611565565b945050505061035c565b8661042682610c36565b146104995760405162461bcd60e51b815260206004820152602260248201527f506f737420657870616e73696f6e20726f6f74206e6f7420657175616c20706f60448201527f73740000000000000000000000000000000000000000000000000000000000006064820152608401610218565b835182146104e95760405162461bcd60e51b815260206004820152601660248201527f496e636f6d706c6574652070726f6f66207573616765000000000000000000006044820152606401610218565b505050505050505050565b6060604083106105465760405162461bcd60e51b815260206004820152600e60248201527f4c6576656c20746f6f20686967680000000000000000000000000000000000006044820152606401610218565b60008290036105975760405162461bcd60e51b815260206004820152601b60248201527f43616e6e6f7420617070656e6420656d707479207375627472656500000000006044820152606401610218565b6040845111156105e95760405162461bcd60e51b815260206004820152601a60248201527f4d65726b6c6520657870616e73696f6e20746f6f206c617267650000000000006044820152606401610218565b83516000036106685760006105ff84600161153c565b67ffffffffffffffff8111156106175761061761121a565b604051908082528060200260200182016040528015610640578160200160208202803683370190505b5090508281858151811061065657610656611510565b60209081029190910101529050610171565b835183106106de5760405162461bcd60e51b815260206004820152603560248201527f4c6576656c2067726561746572207468616e2068696768657374206c6576656c60448201527f206f662063757272656e7420657870616e73696f6e00000000000000000000006064820152608401610218565b8160006106ea86610f85565b905060006106f9866002611663565b610703908361153c565b9050600061071083610e3f565b61071983610e3f565b1161076757875167ffffffffffffffff8111156107385761073861121a565b604051908082528060200260200182016040528015610761578160200160208202803683370190505b506107b7565b875161077490600161153c565b67ffffffffffffffff81111561078c5761078c61121a565b6040519080825280602002602001820160405280156107b5578160200160208202803683370190505b505b905060408151111561080b5760405162461bcd60e51b815260206004820152601c60248201527f417070656e642063726561746573206f76657273697a652074726565000000006044820152606401610218565b60005b88518110156109c757878110156108b55788818151811061083157610831611510565b60200260200101516000801b146108b05760405162461bcd60e51b815260206004820152602260248201527f417070656e642061626f7665206c65617374207369676e69666963616e74206260448201527f69740000000000000000000000000000000000000000000000000000000000006064820152608401610218565b6109b5565b60008590036108fb578881815181106108d0576108d0611510565b60200260200101518282815181106108ea576108ea611510565b6020026020010181815250506109b5565b88818151811061090d5761090d611510565b60200260200101516000801b03610945578482828151811061093157610931611510565b6020908102919091010152600094506109b5565b6000801b82828151811061095b5761095b611510565b60200260200101818152505088818151811061097957610979611510565b60200260200101518560405160200161099c929190918252602082015260400190565b6040516020818303038152906040528051906020012094505b806109bf81611565565b91505061080e565b5083156109fb578381600183516109de919061166f565b815181106109ee576109ee611510565b6020026020010181815250505b8060018251610a0a919061166f565b81518110610a1a57610a1a611510565b60200260200101516000801b03610a735760405162461bcd60e51b815260206004820152600f60248201527f4c61737420656e747279207a65726f00000000000000000000000000000000006044820152606401610218565b979650505050505050565b6000610ab3828486604051602001610a9891815260200190565b6040516020818303038152906040528051906020012061115f565b9050808514610b045760405162461bcd60e51b815260206004820152601760248201527f496e76616c696420696e636c7573696f6e2070726f6f660000000000000000006044820152606401610218565b5050505050565b6000818310610b5c5760405162461bcd60e51b815260206004820152601760248201527f5374617274206e6f74206c657373207468616e20656e640000000000000000006044820152606401610218565b6000610b69838518610e3f565b905060006001610b79838261153c565b6001901b610b87919061166f565b90508481168482168115610ba957610b9e82610dd6565b945050505050610199565b8015610bb857610b9e81610e3f565b60405162461bcd60e51b815260206004820152601b60248201527f426f7468207920616e64207a2063616e6e6f74206265207a65726f00000000006044820152606401610218565b606061019683600084604051602001610c1b91815260200190565b604051602081830303815290604052805190602001206104f4565b600080825111610c885760405162461bcd60e51b815260206004820152601660248201527f456d707479206d65726b6c6520657870616e73696f6e000000000000000000006044820152606401610218565b604082511115610cda5760405162461bcd60e51b815260206004820152601a60248201527f4d65726b6c6520657870616e73696f6e20746f6f206c617267650000000000006044820152606401610218565b6000805b8351811015610dcf576000848281518110610cfb57610cfb611510565b60200260200101519050826000801b03610d67578015610d625780925060018551610d26919061166f565b8214610d6257604051610d49908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b610dbc565b8015610d86576040805160208101839052908101849052606001610d49565b604051610da3908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b5080610dc781611565565b915050610cde565b5092915050565b6000808211610e275760405162461bcd60e51b815260206004820152601c60248201527f5a65726f20686173206e6f207369676e69666963616e742062697473000000006044820152606401610218565b60008280610e3660018261166f565b16189050610171815b600081600003610e915760405162461bcd60e51b815260206004820152601c60248201527f5a65726f20686173206e6f207369676e69666963616e742062697473000000006044820152606401610218565b7001000000000000000000000000000000008210610ebc57608091821c91610eb9908261153c565b90505b680100000000000000008210610edf57604091821c91610edc908261153c565b90505b6401000000008210610efe57602091821c91610efb908261153c565b90505b620100008210610f1b57601091821c91610f18908261153c565b90505b6101008210610f3757600891821c91610f34908261153c565b90505b60108210610f5257600491821c91610f4f908261153c565b90505b60048210610f6d57600291821c91610f6a908261153c565b90505b60028210610f805761019960018261153c565b919050565b600080805b8351811015610dcf57838181518110610fa557610fa5611510565b60200260200101516000801b14610fce57610fc1816002611663565b610fcb908361153c565b91505b80610fd881611565565b915050610f8a565b60608183106110315760405162461bcd60e51b815260206004820152601760248201527f5374617274206e6f74206c657373207468616e20656e640000000000000000006044820152606401610218565b83518211156110a85760405162461bcd60e51b815260206004820152602160248201527f456e64206e6f74206c657373206f7220657175616c207468616e206c656e677460448201527f68000000000000000000000000000000000000000000000000000000000000006064820152608401610218565b60006110b4848461166f565b67ffffffffffffffff8111156110cc576110cc61121a565b6040519080825280602002602001820160405280156110f5578160200160208202803683370190505b509050835b838110156111565785818151811061111457611114611510565b6020026020010151828683611129919061166f565b8151811061113957611139611510565b60209081029190910101528061114e81611565565b9150506110fa565b50949350505050565b82516000906101008111156111ab576040517ffdac331e000000000000000000000000000000000000000000000000000000008152600481018290526101006024820152604401610218565b8260005b828110156112105760008782815181106111cb576111cb611510565b60200260200101519050816001901b87166000036111f757826000528060205260406000209250611207565b8060005282602052604060002092505b506001016111af565b5095945050505050565b634e487b7160e01b600052604160045260246000fd5b600082601f83011261124157600080fd5b8135602067ffffffffffffffff8083111561125e5761125e61121a565b8260051b6040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0603f830116810181811084821117156112a1576112a161121a565b6040529384528581018301938381019250878511156112bf57600080fd5b83870191505b84821015610a73578135835291830191908301906112c5565b60008060008060008060c087890312156112f757600080fd5b86359550602087013594506040870135935060608701359250608087013567ffffffffffffffff8082111561132b57600080fd5b6113378a838b01611230565b935060a089013591508082111561134d57600080fd5b5061135a89828a01611230565b9150509295509295509295565b60008060006060848603121561137c57600080fd5b833567ffffffffffffffff81111561139357600080fd5b61139f86828701611230565b9660208601359650604090950135949350505050565b6020808252825182820181905260009190848201906040850190845b818110156113ed578351835292840192918401916001016113d1565b50909695505050505050565b6000806000806080858703121561140f57600080fd5b843593506020850135925060408501359150606085013567ffffffffffffffff81111561143b57600080fd5b61144787828801611230565b91505092959194509250565b6000806040838503121561146657600080fd5b50508035926020909101359150565b6000806040838503121561148857600080fd5b823567ffffffffffffffff81111561149f57600080fd5b6114ab85828601611230565b95602094909401359450505050565b6000602082840312156114cc57600080fd5b813567ffffffffffffffff8111156114e357600080fd5b6114ef84828501611230565b949350505050565b60006020828403121561150957600080fd5b5035919050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b8082018082111561019957610199611526565b634e487b7160e01b600052600160045260246000fd5b6000600019820361157857611578611526565b5060010190565b600181815b808511156115ba5781600019048211156115a0576115a0611526565b808516156115ad57918102915b93841c9390800290611584565b509250929050565b6000826115d157506001610199565b816115de57506000610199565b81600181146115f457600281146115fe5761161a565b6001915050610199565b60ff84111561160f5761160f611526565b50506001821b610199565b5060208310610133831016604e8410600b841016171561163d575081810a610199565b611647838361157f565b806000190482111561165b5761165b611526565b029392505050565b600061019683836115c2565b818103818111156101995761019961152656fea26469706673582212205bf41bd80017925143bb42a7d2d22486ca1f3b7a0251a436657a538ecf059c2a64736f6c63430008110033",
}

// MerkleTreeAccessABI is the input ABI used to generate the binding from.
// Deprecated: Use MerkleTreeAccessMetaData.ABI instead.
var MerkleTreeAccessABI = MerkleTreeAccessMetaData.ABI

// MerkleTreeAccessBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MerkleTreeAccessMetaData.Bin instead.
var MerkleTreeAccessBin = MerkleTreeAccessMetaData.Bin

// DeployMerkleTreeAccess deploys a new Ethereum contract, binding an instance of MerkleTreeAccess to it.
func DeployMerkleTreeAccess(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MerkleTreeAccess, error) {
	parsed, err := MerkleTreeAccessMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MerkleTreeAccessBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MerkleTreeAccess{MerkleTreeAccessCaller: MerkleTreeAccessCaller{contract: contract}, MerkleTreeAccessTransactor: MerkleTreeAccessTransactor{contract: contract}, MerkleTreeAccessFilterer: MerkleTreeAccessFilterer{contract: contract}}, nil
}

// MerkleTreeAccess is an auto generated Go binding around an Ethereum contract.
type MerkleTreeAccess struct {
	MerkleTreeAccessCaller     // Read-only binding to the contract
	MerkleTreeAccessTransactor // Write-only binding to the contract
	MerkleTreeAccessFilterer   // Log filterer for contract events
}

// MerkleTreeAccessCaller is an auto generated read-only Go binding around an Ethereum contract.
type MerkleTreeAccessCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleTreeAccessTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MerkleTreeAccessTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleTreeAccessFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MerkleTreeAccessFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MerkleTreeAccessSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MerkleTreeAccessSession struct {
	Contract     *MerkleTreeAccess // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MerkleTreeAccessCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MerkleTreeAccessCallerSession struct {
	Contract *MerkleTreeAccessCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// MerkleTreeAccessTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MerkleTreeAccessTransactorSession struct {
	Contract     *MerkleTreeAccessTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// MerkleTreeAccessRaw is an auto generated low-level Go binding around an Ethereum contract.
type MerkleTreeAccessRaw struct {
	Contract *MerkleTreeAccess // Generic contract binding to access the raw methods on
}

// MerkleTreeAccessCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MerkleTreeAccessCallerRaw struct {
	Contract *MerkleTreeAccessCaller // Generic read-only contract binding to access the raw methods on
}

// MerkleTreeAccessTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MerkleTreeAccessTransactorRaw struct {
	Contract *MerkleTreeAccessTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMerkleTreeAccess creates a new instance of MerkleTreeAccess, bound to a specific deployed contract.
func NewMerkleTreeAccess(address common.Address, backend bind.ContractBackend) (*MerkleTreeAccess, error) {
	contract, err := bindMerkleTreeAccess(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MerkleTreeAccess{MerkleTreeAccessCaller: MerkleTreeAccessCaller{contract: contract}, MerkleTreeAccessTransactor: MerkleTreeAccessTransactor{contract: contract}, MerkleTreeAccessFilterer: MerkleTreeAccessFilterer{contract: contract}}, nil
}

// NewMerkleTreeAccessCaller creates a new read-only instance of MerkleTreeAccess, bound to a specific deployed contract.
func NewMerkleTreeAccessCaller(address common.Address, caller bind.ContractCaller) (*MerkleTreeAccessCaller, error) {
	contract, err := bindMerkleTreeAccess(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MerkleTreeAccessCaller{contract: contract}, nil
}

// NewMerkleTreeAccessTransactor creates a new write-only instance of MerkleTreeAccess, bound to a specific deployed contract.
func NewMerkleTreeAccessTransactor(address common.Address, transactor bind.ContractTransactor) (*MerkleTreeAccessTransactor, error) {
	contract, err := bindMerkleTreeAccess(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MerkleTreeAccessTransactor{contract: contract}, nil
}

// NewMerkleTreeAccessFilterer creates a new log filterer instance of MerkleTreeAccess, bound to a specific deployed contract.
func NewMerkleTreeAccessFilterer(address common.Address, filterer bind.ContractFilterer) (*MerkleTreeAccessFilterer, error) {
	contract, err := bindMerkleTreeAccess(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MerkleTreeAccessFilterer{contract: contract}, nil
}

// bindMerkleTreeAccess binds a generic wrapper to an already deployed contract.
func bindMerkleTreeAccess(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MerkleTreeAccessMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MerkleTreeAccess *MerkleTreeAccessRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MerkleTreeAccess.Contract.MerkleTreeAccessCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MerkleTreeAccess *MerkleTreeAccessRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MerkleTreeAccess.Contract.MerkleTreeAccessTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MerkleTreeAccess *MerkleTreeAccessRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MerkleTreeAccess.Contract.MerkleTreeAccessTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MerkleTreeAccess *MerkleTreeAccessCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MerkleTreeAccess.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MerkleTreeAccess *MerkleTreeAccessTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MerkleTreeAccess.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MerkleTreeAccess *MerkleTreeAccessTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MerkleTreeAccess.Contract.contract.Transact(opts, method, params...)
}

// AppendCompleteSubTree is a free data retrieval call binding the contract method 0x67905a7e.
//
// Solidity: function appendCompleteSubTree(bytes32[] me, uint256 level, bytes32 subtreeRoot) pure returns(bytes32[])
func (_MerkleTreeAccess *MerkleTreeAccessCaller) AppendCompleteSubTree(opts *bind.CallOpts, me [][32]byte, level *big.Int, subtreeRoot [32]byte) ([][32]byte, error) {
	var out []interface{}
	err := _MerkleTreeAccess.contract.Call(opts, &out, "appendCompleteSubTree", me, level, subtreeRoot)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// AppendCompleteSubTree is a free data retrieval call binding the contract method 0x67905a7e.
//
// Solidity: function appendCompleteSubTree(bytes32[] me, uint256 level, bytes32 subtreeRoot) pure returns(bytes32[])
func (_MerkleTreeAccess *MerkleTreeAccessSession) AppendCompleteSubTree(me [][32]byte, level *big.Int, subtreeRoot [32]byte) ([][32]byte, error) {
	return _MerkleTreeAccess.Contract.AppendCompleteSubTree(&_MerkleTreeAccess.CallOpts, me, level, subtreeRoot)
}

// AppendCompleteSubTree is a free data retrieval call binding the contract method 0x67905a7e.
//
// Solidity: function appendCompleteSubTree(bytes32[] me, uint256 level, bytes32 subtreeRoot) pure returns(bytes32[])
func (_MerkleTreeAccess *MerkleTreeAccessCallerSession) AppendCompleteSubTree(me [][32]byte, level *big.Int, subtreeRoot [32]byte) ([][32]byte, error) {
	return _MerkleTreeAccess.Contract.AppendCompleteSubTree(&_MerkleTreeAccess.CallOpts, me, level, subtreeRoot)
}

// AppendLeaf is a free data retrieval call binding the contract method 0xc22c47a4.
//
// Solidity: function appendLeaf(bytes32[] me, bytes32 leaf) pure returns(bytes32[])
func (_MerkleTreeAccess *MerkleTreeAccessCaller) AppendLeaf(opts *bind.CallOpts, me [][32]byte, leaf [32]byte) ([][32]byte, error) {
	var out []interface{}
	err := _MerkleTreeAccess.contract.Call(opts, &out, "appendLeaf", me, leaf)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// AppendLeaf is a free data retrieval call binding the contract method 0xc22c47a4.
//
// Solidity: function appendLeaf(bytes32[] me, bytes32 leaf) pure returns(bytes32[])
func (_MerkleTreeAccess *MerkleTreeAccessSession) AppendLeaf(me [][32]byte, leaf [32]byte) ([][32]byte, error) {
	return _MerkleTreeAccess.Contract.AppendLeaf(&_MerkleTreeAccess.CallOpts, me, leaf)
}

// AppendLeaf is a free data retrieval call binding the contract method 0xc22c47a4.
//
// Solidity: function appendLeaf(bytes32[] me, bytes32 leaf) pure returns(bytes32[])
func (_MerkleTreeAccess *MerkleTreeAccessCallerSession) AppendLeaf(me [][32]byte, leaf [32]byte) ([][32]byte, error) {
	return _MerkleTreeAccess.Contract.AppendLeaf(&_MerkleTreeAccess.CallOpts, me, leaf)
}

// LeastSignificantBit is a free data retrieval call binding the contract method 0xd230d23f.
//
// Solidity: function leastSignificantBit(uint256 x) pure returns(uint256)
func (_MerkleTreeAccess *MerkleTreeAccessCaller) LeastSignificantBit(opts *bind.CallOpts, x *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _MerkleTreeAccess.contract.Call(opts, &out, "leastSignificantBit", x)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LeastSignificantBit is a free data retrieval call binding the contract method 0xd230d23f.
//
// Solidity: function leastSignificantBit(uint256 x) pure returns(uint256)
func (_MerkleTreeAccess *MerkleTreeAccessSession) LeastSignificantBit(x *big.Int) (*big.Int, error) {
	return _MerkleTreeAccess.Contract.LeastSignificantBit(&_MerkleTreeAccess.CallOpts, x)
}

// LeastSignificantBit is a free data retrieval call binding the contract method 0xd230d23f.
//
// Solidity: function leastSignificantBit(uint256 x) pure returns(uint256)
func (_MerkleTreeAccess *MerkleTreeAccessCallerSession) LeastSignificantBit(x *big.Int) (*big.Int, error) {
	return _MerkleTreeAccess.Contract.LeastSignificantBit(&_MerkleTreeAccess.CallOpts, x)
}

// MaximumAppendBetween is a free data retrieval call binding the contract method 0xbc2f0640.
//
// Solidity: function maximumAppendBetween(uint256 startSize, uint256 endSize) pure returns(uint256)
func (_MerkleTreeAccess *MerkleTreeAccessCaller) MaximumAppendBetween(opts *bind.CallOpts, startSize *big.Int, endSize *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _MerkleTreeAccess.contract.Call(opts, &out, "maximumAppendBetween", startSize, endSize)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaximumAppendBetween is a free data retrieval call binding the contract method 0xbc2f0640.
//
// Solidity: function maximumAppendBetween(uint256 startSize, uint256 endSize) pure returns(uint256)
func (_MerkleTreeAccess *MerkleTreeAccessSession) MaximumAppendBetween(startSize *big.Int, endSize *big.Int) (*big.Int, error) {
	return _MerkleTreeAccess.Contract.MaximumAppendBetween(&_MerkleTreeAccess.CallOpts, startSize, endSize)
}

// MaximumAppendBetween is a free data retrieval call binding the contract method 0xbc2f0640.
//
// Solidity: function maximumAppendBetween(uint256 startSize, uint256 endSize) pure returns(uint256)
func (_MerkleTreeAccess *MerkleTreeAccessCallerSession) MaximumAppendBetween(startSize *big.Int, endSize *big.Int) (*big.Int, error) {
	return _MerkleTreeAccess.Contract.MaximumAppendBetween(&_MerkleTreeAccess.CallOpts, startSize, endSize)
}

// MostSignificantBit is a free data retrieval call binding the contract method 0xe6bcbc65.
//
// Solidity: function mostSignificantBit(uint256 x) pure returns(uint256)
func (_MerkleTreeAccess *MerkleTreeAccessCaller) MostSignificantBit(opts *bind.CallOpts, x *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _MerkleTreeAccess.contract.Call(opts, &out, "mostSignificantBit", x)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MostSignificantBit is a free data retrieval call binding the contract method 0xe6bcbc65.
//
// Solidity: function mostSignificantBit(uint256 x) pure returns(uint256)
func (_MerkleTreeAccess *MerkleTreeAccessSession) MostSignificantBit(x *big.Int) (*big.Int, error) {
	return _MerkleTreeAccess.Contract.MostSignificantBit(&_MerkleTreeAccess.CallOpts, x)
}

// MostSignificantBit is a free data retrieval call binding the contract method 0xe6bcbc65.
//
// Solidity: function mostSignificantBit(uint256 x) pure returns(uint256)
func (_MerkleTreeAccess *MerkleTreeAccessCallerSession) MostSignificantBit(x *big.Int) (*big.Int, error) {
	return _MerkleTreeAccess.Contract.MostSignificantBit(&_MerkleTreeAccess.CallOpts, x)
}

// Root is a free data retrieval call binding the contract method 0xca113253.
//
// Solidity: function root(bytes32[] me) pure returns(bytes32)
func (_MerkleTreeAccess *MerkleTreeAccessCaller) Root(opts *bind.CallOpts, me [][32]byte) ([32]byte, error) {
	var out []interface{}
	err := _MerkleTreeAccess.contract.Call(opts, &out, "root", me)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Root is a free data retrieval call binding the contract method 0xca113253.
//
// Solidity: function root(bytes32[] me) pure returns(bytes32)
func (_MerkleTreeAccess *MerkleTreeAccessSession) Root(me [][32]byte) ([32]byte, error) {
	return _MerkleTreeAccess.Contract.Root(&_MerkleTreeAccess.CallOpts, me)
}

// Root is a free data retrieval call binding the contract method 0xca113253.
//
// Solidity: function root(bytes32[] me) pure returns(bytes32)
func (_MerkleTreeAccess *MerkleTreeAccessCallerSession) Root(me [][32]byte) ([32]byte, error) {
	return _MerkleTreeAccess.Contract.Root(&_MerkleTreeAccess.CallOpts, me)
}

// VerifyInclusionProof is a free data retrieval call binding the contract method 0x6bd58993.
//
// Solidity: function verifyInclusionProof(bytes32 rootHash, bytes32 leaf, uint256 index, bytes32[] proof) pure returns()
func (_MerkleTreeAccess *MerkleTreeAccessCaller) VerifyInclusionProof(opts *bind.CallOpts, rootHash [32]byte, leaf [32]byte, index *big.Int, proof [][32]byte) error {
	var out []interface{}
	err := _MerkleTreeAccess.contract.Call(opts, &out, "verifyInclusionProof", rootHash, leaf, index, proof)

	if err != nil {
		return err
	}

	return err

}

// VerifyInclusionProof is a free data retrieval call binding the contract method 0x6bd58993.
//
// Solidity: function verifyInclusionProof(bytes32 rootHash, bytes32 leaf, uint256 index, bytes32[] proof) pure returns()
func (_MerkleTreeAccess *MerkleTreeAccessSession) VerifyInclusionProof(rootHash [32]byte, leaf [32]byte, index *big.Int, proof [][32]byte) error {
	return _MerkleTreeAccess.Contract.VerifyInclusionProof(&_MerkleTreeAccess.CallOpts, rootHash, leaf, index, proof)
}

// VerifyInclusionProof is a free data retrieval call binding the contract method 0x6bd58993.
//
// Solidity: function verifyInclusionProof(bytes32 rootHash, bytes32 leaf, uint256 index, bytes32[] proof) pure returns()
func (_MerkleTreeAccess *MerkleTreeAccessCallerSession) VerifyInclusionProof(rootHash [32]byte, leaf [32]byte, index *big.Int, proof [][32]byte) error {
	return _MerkleTreeAccess.Contract.VerifyInclusionProof(&_MerkleTreeAccess.CallOpts, rootHash, leaf, index, proof)
}

// VerifyPrefixProof is a free data retrieval call binding the contract method 0x5fb9c3d4.
//
// Solidity: function verifyPrefixProof(bytes32 preRoot, uint256 preSize, bytes32 postRoot, uint256 postSize, bytes32[] preExpansion, bytes32[] proof) pure returns()
func (_MerkleTreeAccess *MerkleTreeAccessCaller) VerifyPrefixProof(opts *bind.CallOpts, preRoot [32]byte, preSize *big.Int, postRoot [32]byte, postSize *big.Int, preExpansion [][32]byte, proof [][32]byte) error {
	var out []interface{}
	err := _MerkleTreeAccess.contract.Call(opts, &out, "verifyPrefixProof", preRoot, preSize, postRoot, postSize, preExpansion, proof)

	if err != nil {
		return err
	}

	return err

}

// VerifyPrefixProof is a free data retrieval call binding the contract method 0x5fb9c3d4.
//
// Solidity: function verifyPrefixProof(bytes32 preRoot, uint256 preSize, bytes32 postRoot, uint256 postSize, bytes32[] preExpansion, bytes32[] proof) pure returns()
func (_MerkleTreeAccess *MerkleTreeAccessSession) VerifyPrefixProof(preRoot [32]byte, preSize *big.Int, postRoot [32]byte, postSize *big.Int, preExpansion [][32]byte, proof [][32]byte) error {
	return _MerkleTreeAccess.Contract.VerifyPrefixProof(&_MerkleTreeAccess.CallOpts, preRoot, preSize, postRoot, postSize, preExpansion, proof)
}

// VerifyPrefixProof is a free data retrieval call binding the contract method 0x5fb9c3d4.
//
// Solidity: function verifyPrefixProof(bytes32 preRoot, uint256 preSize, bytes32 postRoot, uint256 postSize, bytes32[] preExpansion, bytes32[] proof) pure returns()
func (_MerkleTreeAccess *MerkleTreeAccessCallerSession) VerifyPrefixProof(preRoot [32]byte, preSize *big.Int, postRoot [32]byte, postSize *big.Int, preExpansion [][32]byte, proof [][32]byte) error {
	return _MerkleTreeAccess.Contract.VerifyPrefixProof(&_MerkleTreeAccess.CallOpts, preRoot, preSize, postRoot, postSize, preExpansion, proof)
}

// MockRollupEventInboxMetaData contains all meta data concerning the MockRollupEventInbox contract.
var MockRollupEventInboxMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"AlreadyInit\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HadZeroInit\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotOwner\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"_bridge\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"chainConfig\",\"type\":\"string\"}],\"name\":\"rollupInitialized\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a06040523060805234801561001457600080fd5b506080516109ae6100366000396000818160e801526102a701526109ae6000f3fe608060405234801561001057600080fd5b50600436106100675760003560e01c8063cb23bcb511610050578063cb23bcb514610089578063cf8d56d6146100b8578063e78cea92146100cb57600080fd5b80636ae71f121461006c578063c4d66de814610076575b600080fd5b6100746100de565b005b6100746100843660046107a2565b61029d565b60015461009c906001600160a01b031681565b6040516001600160a01b03909116815260200160405180910390f35b6100746100c63660046107c6565b610491565b60005461009c906001600160a01b031681565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036101815760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084015b60405180910390fd5b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038054336001600160a01b038216146101f7576040517f23295f0e0000000000000000000000000000000000000000000000000000000081523360048201526001600160a01b0382166024820152604401610178565b60008054906101000a90046001600160a01b03166001600160a01b031663cb23bcb56040518163ffffffff1660e01b8152600401602060405180830381865afa158015610248573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061026c9190610842565b6001805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03929092169190911790555050565b6001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016300361033b5760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c00000000000000000000000000000000000000006064820152608401610178565b6000546001600160a01b03161561037e576040517fef34ca5c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001600160a01b0381166103be576040517f1ad0f74300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038316908117909155604080517fcb23bcb5000000000000000000000000000000000000000000000000000000008152905163cb23bcb5916004808201926020929091908290030181865afa15801561043d573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906104619190610842565b6001805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b039290921691909117905550565b6001546001600160a01b031633146104eb5760405162461bcd60e51b815260206004820152600b60248201527f4f4e4c595f524f4c4c55500000000000000000000000000000000000000000006044820152606401610178565b806105385760405162461bcd60e51b815260206004820152601260248201527f454d5054595f434841494e5f434f4e46494700000000000000000000000000006044820152606401610178565b6001806105436106c4565b156105b857606c6001600160a01b031663f5d6ded76040518163ffffffff1660e01b8152600401602060405180830381865afa158015610587573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105ab919061085f565b6105b59082610878565b90505b600085838387876040516020016105d39594939291906108b8565b60408051808303601f190181529082905260008054825160208401207f8db5993b000000000000000000000000000000000000000000000000000000008552600b6004860152602485018390526044850152919350916001600160a01b0390911690638db5993b906064016020604051808303816000875af115801561065d573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610681919061085f565b9050807fff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b836040516106b39190610929565b60405180910390a250505050505050565b60408051600481526024810182526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f051038f200000000000000000000000000000000000000000000000000000000179052905160009182918291606491610730919061095c565b600060405180830381855afa9150503d806000811461076b576040519150601f19603f3d011682016040523d82523d6000602084013e610770565b606091505b5091509150818015610783575080516020145b9250505090565b6001600160a01b038116811461079f57600080fd5b50565b6000602082840312156107b457600080fd5b81356107bf8161078a565b9392505050565b6000806000604084860312156107db57600080fd5b83359250602084013567ffffffffffffffff808211156107fa57600080fd5b818601915086601f83011261080e57600080fd5b81358181111561081d57600080fd5b87602082850101111561082f57600080fd5b6020830194508093505050509250925092565b60006020828403121561085457600080fd5b81516107bf8161078a565b60006020828403121561087157600080fd5b5051919050565b808201808211156108b2577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b92915050565b8581527fff000000000000000000000000000000000000000000000000000000000000008560f81b1660208201528360218201528183604183013760009101604101908152949350505050565b60005b83811015610920578181015183820152602001610908565b50506000910152565b6020815260008251806020840152610948816040850160208701610905565b601f01601f19169190910160400192915050565b6000825161096e818460208701610905565b919091019291505056fea26469706673582212206c4fb82025280762ddba386b4cb981c9a953416bbc43795073076c20e07ba86864736f6c63430008110033",
}

// MockRollupEventInboxABI is the input ABI used to generate the binding from.
// Deprecated: Use MockRollupEventInboxMetaData.ABI instead.
var MockRollupEventInboxABI = MockRollupEventInboxMetaData.ABI

// MockRollupEventInboxBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MockRollupEventInboxMetaData.Bin instead.
var MockRollupEventInboxBin = MockRollupEventInboxMetaData.Bin

// DeployMockRollupEventInbox deploys a new Ethereum contract, binding an instance of MockRollupEventInbox to it.
func DeployMockRollupEventInbox(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MockRollupEventInbox, error) {
	parsed, err := MockRollupEventInboxMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MockRollupEventInboxBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MockRollupEventInbox{MockRollupEventInboxCaller: MockRollupEventInboxCaller{contract: contract}, MockRollupEventInboxTransactor: MockRollupEventInboxTransactor{contract: contract}, MockRollupEventInboxFilterer: MockRollupEventInboxFilterer{contract: contract}}, nil
}

// MockRollupEventInbox is an auto generated Go binding around an Ethereum contract.
type MockRollupEventInbox struct {
	MockRollupEventInboxCaller     // Read-only binding to the contract
	MockRollupEventInboxTransactor // Write-only binding to the contract
	MockRollupEventInboxFilterer   // Log filterer for contract events
}

// MockRollupEventInboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type MockRollupEventInboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockRollupEventInboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MockRollupEventInboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockRollupEventInboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MockRollupEventInboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockRollupEventInboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MockRollupEventInboxSession struct {
	Contract     *MockRollupEventInbox // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// MockRollupEventInboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MockRollupEventInboxCallerSession struct {
	Contract *MockRollupEventInboxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// MockRollupEventInboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MockRollupEventInboxTransactorSession struct {
	Contract     *MockRollupEventInboxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// MockRollupEventInboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type MockRollupEventInboxRaw struct {
	Contract *MockRollupEventInbox // Generic contract binding to access the raw methods on
}

// MockRollupEventInboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MockRollupEventInboxCallerRaw struct {
	Contract *MockRollupEventInboxCaller // Generic read-only contract binding to access the raw methods on
}

// MockRollupEventInboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MockRollupEventInboxTransactorRaw struct {
	Contract *MockRollupEventInboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMockRollupEventInbox creates a new instance of MockRollupEventInbox, bound to a specific deployed contract.
func NewMockRollupEventInbox(address common.Address, backend bind.ContractBackend) (*MockRollupEventInbox, error) {
	contract, err := bindMockRollupEventInbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MockRollupEventInbox{MockRollupEventInboxCaller: MockRollupEventInboxCaller{contract: contract}, MockRollupEventInboxTransactor: MockRollupEventInboxTransactor{contract: contract}, MockRollupEventInboxFilterer: MockRollupEventInboxFilterer{contract: contract}}, nil
}

// NewMockRollupEventInboxCaller creates a new read-only instance of MockRollupEventInbox, bound to a specific deployed contract.
func NewMockRollupEventInboxCaller(address common.Address, caller bind.ContractCaller) (*MockRollupEventInboxCaller, error) {
	contract, err := bindMockRollupEventInbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MockRollupEventInboxCaller{contract: contract}, nil
}

// NewMockRollupEventInboxTransactor creates a new write-only instance of MockRollupEventInbox, bound to a specific deployed contract.
func NewMockRollupEventInboxTransactor(address common.Address, transactor bind.ContractTransactor) (*MockRollupEventInboxTransactor, error) {
	contract, err := bindMockRollupEventInbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MockRollupEventInboxTransactor{contract: contract}, nil
}

// NewMockRollupEventInboxFilterer creates a new log filterer instance of MockRollupEventInbox, bound to a specific deployed contract.
func NewMockRollupEventInboxFilterer(address common.Address, filterer bind.ContractFilterer) (*MockRollupEventInboxFilterer, error) {
	contract, err := bindMockRollupEventInbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MockRollupEventInboxFilterer{contract: contract}, nil
}

// bindMockRollupEventInbox binds a generic wrapper to an already deployed contract.
func bindMockRollupEventInbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MockRollupEventInboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockRollupEventInbox *MockRollupEventInboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockRollupEventInbox.Contract.MockRollupEventInboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockRollupEventInbox *MockRollupEventInboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockRollupEventInbox.Contract.MockRollupEventInboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockRollupEventInbox *MockRollupEventInboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockRollupEventInbox.Contract.MockRollupEventInboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockRollupEventInbox *MockRollupEventInboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockRollupEventInbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockRollupEventInbox *MockRollupEventInboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockRollupEventInbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockRollupEventInbox *MockRollupEventInboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockRollupEventInbox.Contract.contract.Transact(opts, method, params...)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_MockRollupEventInbox *MockRollupEventInboxCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MockRollupEventInbox.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_MockRollupEventInbox *MockRollupEventInboxSession) Bridge() (common.Address, error) {
	return _MockRollupEventInbox.Contract.Bridge(&_MockRollupEventInbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_MockRollupEventInbox *MockRollupEventInboxCallerSession) Bridge() (common.Address, error) {
	return _MockRollupEventInbox.Contract.Bridge(&_MockRollupEventInbox.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_MockRollupEventInbox *MockRollupEventInboxCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MockRollupEventInbox.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_MockRollupEventInbox *MockRollupEventInboxSession) Rollup() (common.Address, error) {
	return _MockRollupEventInbox.Contract.Rollup(&_MockRollupEventInbox.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_MockRollupEventInbox *MockRollupEventInboxCallerSession) Rollup() (common.Address, error) {
	return _MockRollupEventInbox.Contract.Rollup(&_MockRollupEventInbox.CallOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_MockRollupEventInbox *MockRollupEventInboxTransactor) Initialize(opts *bind.TransactOpts, _bridge common.Address) (*types.Transaction, error) {
	return _MockRollupEventInbox.contract.Transact(opts, "initialize", _bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_MockRollupEventInbox *MockRollupEventInboxSession) Initialize(_bridge common.Address) (*types.Transaction, error) {
	return _MockRollupEventInbox.Contract.Initialize(&_MockRollupEventInbox.TransactOpts, _bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_MockRollupEventInbox *MockRollupEventInboxTransactorSession) Initialize(_bridge common.Address) (*types.Transaction, error) {
	return _MockRollupEventInbox.Contract.Initialize(&_MockRollupEventInbox.TransactOpts, _bridge)
}

// RollupInitialized is a paid mutator transaction binding the contract method 0xcf8d56d6.
//
// Solidity: function rollupInitialized(uint256 chainId, string chainConfig) returns()
func (_MockRollupEventInbox *MockRollupEventInboxTransactor) RollupInitialized(opts *bind.TransactOpts, chainId *big.Int, chainConfig string) (*types.Transaction, error) {
	return _MockRollupEventInbox.contract.Transact(opts, "rollupInitialized", chainId, chainConfig)
}

// RollupInitialized is a paid mutator transaction binding the contract method 0xcf8d56d6.
//
// Solidity: function rollupInitialized(uint256 chainId, string chainConfig) returns()
func (_MockRollupEventInbox *MockRollupEventInboxSession) RollupInitialized(chainId *big.Int, chainConfig string) (*types.Transaction, error) {
	return _MockRollupEventInbox.Contract.RollupInitialized(&_MockRollupEventInbox.TransactOpts, chainId, chainConfig)
}

// RollupInitialized is a paid mutator transaction binding the contract method 0xcf8d56d6.
//
// Solidity: function rollupInitialized(uint256 chainId, string chainConfig) returns()
func (_MockRollupEventInbox *MockRollupEventInboxTransactorSession) RollupInitialized(chainId *big.Int, chainConfig string) (*types.Transaction, error) {
	return _MockRollupEventInbox.Contract.RollupInitialized(&_MockRollupEventInbox.TransactOpts, chainId, chainConfig)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_MockRollupEventInbox *MockRollupEventInboxTransactor) UpdateRollupAddress(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockRollupEventInbox.contract.Transact(opts, "updateRollupAddress")
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_MockRollupEventInbox *MockRollupEventInboxSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _MockRollupEventInbox.Contract.UpdateRollupAddress(&_MockRollupEventInbox.TransactOpts)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_MockRollupEventInbox *MockRollupEventInboxTransactorSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _MockRollupEventInbox.Contract.UpdateRollupAddress(&_MockRollupEventInbox.TransactOpts)
}

// MockRollupEventInboxInboxMessageDeliveredIterator is returned from FilterInboxMessageDelivered and is used to iterate over the raw logs and unpacked data for InboxMessageDelivered events raised by the MockRollupEventInbox contract.
type MockRollupEventInboxInboxMessageDeliveredIterator struct {
	Event *MockRollupEventInboxInboxMessageDelivered // Event containing the contract specifics and raw log

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
func (it *MockRollupEventInboxInboxMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MockRollupEventInboxInboxMessageDelivered)
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
		it.Event = new(MockRollupEventInboxInboxMessageDelivered)
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
func (it *MockRollupEventInboxInboxMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MockRollupEventInboxInboxMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MockRollupEventInboxInboxMessageDelivered represents a InboxMessageDelivered event raised by the MockRollupEventInbox contract.
type MockRollupEventInboxInboxMessageDelivered struct {
	MessageNum *big.Int
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDelivered is a free log retrieval operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_MockRollupEventInbox *MockRollupEventInboxFilterer) FilterInboxMessageDelivered(opts *bind.FilterOpts, messageNum []*big.Int) (*MockRollupEventInboxInboxMessageDeliveredIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _MockRollupEventInbox.contract.FilterLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &MockRollupEventInboxInboxMessageDeliveredIterator{contract: _MockRollupEventInbox.contract, event: "InboxMessageDelivered", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDelivered is a free log subscription operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_MockRollupEventInbox *MockRollupEventInboxFilterer) WatchInboxMessageDelivered(opts *bind.WatchOpts, sink chan<- *MockRollupEventInboxInboxMessageDelivered, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _MockRollupEventInbox.contract.WatchLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MockRollupEventInboxInboxMessageDelivered)
				if err := _MockRollupEventInbox.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
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

// ParseInboxMessageDelivered is a log parse operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_MockRollupEventInbox *MockRollupEventInboxFilterer) ParseInboxMessageDelivered(log types.Log) (*MockRollupEventInboxInboxMessageDelivered, error) {
	event := new(MockRollupEventInboxInboxMessageDelivered)
	if err := _MockRollupEventInbox.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MockRollupEventInboxInboxMessageDeliveredFromOriginIterator is returned from FilterInboxMessageDeliveredFromOrigin and is used to iterate over the raw logs and unpacked data for InboxMessageDeliveredFromOrigin events raised by the MockRollupEventInbox contract.
type MockRollupEventInboxInboxMessageDeliveredFromOriginIterator struct {
	Event *MockRollupEventInboxInboxMessageDeliveredFromOrigin // Event containing the contract specifics and raw log

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
func (it *MockRollupEventInboxInboxMessageDeliveredFromOriginIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MockRollupEventInboxInboxMessageDeliveredFromOrigin)
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
		it.Event = new(MockRollupEventInboxInboxMessageDeliveredFromOrigin)
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
func (it *MockRollupEventInboxInboxMessageDeliveredFromOriginIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MockRollupEventInboxInboxMessageDeliveredFromOriginIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MockRollupEventInboxInboxMessageDeliveredFromOrigin represents a InboxMessageDeliveredFromOrigin event raised by the MockRollupEventInbox contract.
type MockRollupEventInboxInboxMessageDeliveredFromOrigin struct {
	MessageNum *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDeliveredFromOrigin is a free log retrieval operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_MockRollupEventInbox *MockRollupEventInboxFilterer) FilterInboxMessageDeliveredFromOrigin(opts *bind.FilterOpts, messageNum []*big.Int) (*MockRollupEventInboxInboxMessageDeliveredFromOriginIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _MockRollupEventInbox.contract.FilterLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &MockRollupEventInboxInboxMessageDeliveredFromOriginIterator{contract: _MockRollupEventInbox.contract, event: "InboxMessageDeliveredFromOrigin", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDeliveredFromOrigin is a free log subscription operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_MockRollupEventInbox *MockRollupEventInboxFilterer) WatchInboxMessageDeliveredFromOrigin(opts *bind.WatchOpts, sink chan<- *MockRollupEventInboxInboxMessageDeliveredFromOrigin, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _MockRollupEventInbox.contract.WatchLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MockRollupEventInboxInboxMessageDeliveredFromOrigin)
				if err := _MockRollupEventInbox.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
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

// ParseInboxMessageDeliveredFromOrigin is a log parse operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_MockRollupEventInbox *MockRollupEventInboxFilterer) ParseInboxMessageDeliveredFromOrigin(log types.Log) (*MockRollupEventInboxInboxMessageDeliveredFromOrigin, error) {
	event := new(MockRollupEventInboxInboxMessageDeliveredFromOrigin)
	if err := _MockRollupEventInbox.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ProxyAdminForBindingMetaData contains all meta data concerning the ProxyAdminForBinding contract.
var ProxyAdminForBindingMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"contractTransparentUpgradeableProxy\",\"name\":\"proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"changeProxyAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractTransparentUpgradeableProxy\",\"name\":\"proxy\",\"type\":\"address\"}],\"name\":\"getProxyAdmin\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractTransparentUpgradeableProxy\",\"name\":\"proxy\",\"type\":\"address\"}],\"name\":\"getProxyImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractTransparentUpgradeableProxy\",\"name\":\"proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"upgrade\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractTransparentUpgradeableProxy\",\"name\":\"proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"upgradeAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061001a3361001f565b61006f565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b61078d8061007e6000396000f3fe60806040526004361061007b5760003560e01c80639623609d1161004e5780639623609d1461011157806399a88ec414610124578063f2fde38b14610144578063f3b7dead1461016457600080fd5b8063204e1c7a14610080578063715018a6146100bc5780637eff275e146100d35780638da5cb5b146100f3575b600080fd5b34801561008c57600080fd5b506100a061009b366004610579565b610184565b6040516001600160a01b03909116815260200160405180910390f35b3480156100c857600080fd5b506100d161022e565b005b3480156100df57600080fd5b506100d16100ee36600461059d565b610242565b3480156100ff57600080fd5b506000546001600160a01b03166100a0565b6100d161011f366004610605565b6102c3565b34801561013057600080fd5b506100d161013f36600461059d565b61034b565b34801561015057600080fd5b506100d161015f366004610579565b61039a565b34801561017057600080fd5b506100a061017f366004610579565b610449565b6000806000836001600160a01b03166040516101c3907f5c60da1b00000000000000000000000000000000000000000000000000000000815260040190565b600060405180830381855afa9150503d80600081146101fe576040519150601f19603f3d011682016040523d82523d6000602084013e610203565b606091505b50915091508161021257600080fd5b8080602001905181019061022691906106db565b949350505050565b610236610488565b61024060006104fc565b565b61024a610488565b6040517f8f2839700000000000000000000000000000000000000000000000000000000081526001600160a01b038281166004830152831690638f283970906024015b600060405180830381600087803b1580156102a757600080fd5b505af11580156102bb573d6000803e3d6000fd5b505050505050565b6102cb610488565b6040517f4f1ef2860000000000000000000000000000000000000000000000000000000081526001600160a01b03841690634f1ef28690349061031490869086906004016106f8565b6000604051808303818588803b15801561032d57600080fd5b505af1158015610341573d6000803e3d6000fd5b5050505050505050565b610353610488565b6040517f3659cfe60000000000000000000000000000000000000000000000000000000081526001600160a01b038281166004830152831690633659cfe69060240161028d565b6103a2610488565b6001600160a01b03811661043d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b610446816104fc565b50565b6000806000836001600160a01b03166040516101c3907ff851a44000000000000000000000000000000000000000000000000000000000815260040190565b6000546001600160a01b03163314610240576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401610434565b600080546001600160a01b038381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b6001600160a01b038116811461044657600080fd5b60006020828403121561058b57600080fd5b813561059681610564565b9392505050565b600080604083850312156105b057600080fd5b82356105bb81610564565b915060208301356105cb81610564565b809150509250929050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60008060006060848603121561061a57600080fd5b833561062581610564565b9250602084013561063581610564565b9150604084013567ffffffffffffffff8082111561065257600080fd5b818601915086601f83011261066657600080fd5b813581811115610678576106786105d6565b604051601f8201601f19908116603f011681019083821181831017156106a0576106a06105d6565b816040528281528960208487010111156106b957600080fd5b8260208601602083013760006020848301015280955050505050509250925092565b6000602082840312156106ed57600080fd5b815161059681610564565b6001600160a01b038316815260006020604081840152835180604085015260005b8181101561073557858101830151858201606001528201610719565b506000606082860101526060601f19601f83011685010192505050939250505056fea264697066735822122076d272eb6456a016cf2af94ef883f02aa3ffcaa0f2aeaae4c2d10b1a127ace0464736f6c63430008110033",
}

// ProxyAdminForBindingABI is the input ABI used to generate the binding from.
// Deprecated: Use ProxyAdminForBindingMetaData.ABI instead.
var ProxyAdminForBindingABI = ProxyAdminForBindingMetaData.ABI

// ProxyAdminForBindingBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ProxyAdminForBindingMetaData.Bin instead.
var ProxyAdminForBindingBin = ProxyAdminForBindingMetaData.Bin

// DeployProxyAdminForBinding deploys a new Ethereum contract, binding an instance of ProxyAdminForBinding to it.
func DeployProxyAdminForBinding(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ProxyAdminForBinding, error) {
	parsed, err := ProxyAdminForBindingMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ProxyAdminForBindingBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ProxyAdminForBinding{ProxyAdminForBindingCaller: ProxyAdminForBindingCaller{contract: contract}, ProxyAdminForBindingTransactor: ProxyAdminForBindingTransactor{contract: contract}, ProxyAdminForBindingFilterer: ProxyAdminForBindingFilterer{contract: contract}}, nil
}

// ProxyAdminForBinding is an auto generated Go binding around an Ethereum contract.
type ProxyAdminForBinding struct {
	ProxyAdminForBindingCaller     // Read-only binding to the contract
	ProxyAdminForBindingTransactor // Write-only binding to the contract
	ProxyAdminForBindingFilterer   // Log filterer for contract events
}

// ProxyAdminForBindingCaller is an auto generated read-only Go binding around an Ethereum contract.
type ProxyAdminForBindingCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProxyAdminForBindingTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ProxyAdminForBindingTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProxyAdminForBindingFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ProxyAdminForBindingFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProxyAdminForBindingSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ProxyAdminForBindingSession struct {
	Contract     *ProxyAdminForBinding // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ProxyAdminForBindingCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ProxyAdminForBindingCallerSession struct {
	Contract *ProxyAdminForBindingCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// ProxyAdminForBindingTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ProxyAdminForBindingTransactorSession struct {
	Contract     *ProxyAdminForBindingTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// ProxyAdminForBindingRaw is an auto generated low-level Go binding around an Ethereum contract.
type ProxyAdminForBindingRaw struct {
	Contract *ProxyAdminForBinding // Generic contract binding to access the raw methods on
}

// ProxyAdminForBindingCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ProxyAdminForBindingCallerRaw struct {
	Contract *ProxyAdminForBindingCaller // Generic read-only contract binding to access the raw methods on
}

// ProxyAdminForBindingTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ProxyAdminForBindingTransactorRaw struct {
	Contract *ProxyAdminForBindingTransactor // Generic write-only contract binding to access the raw methods on
}

// NewProxyAdminForBinding creates a new instance of ProxyAdminForBinding, bound to a specific deployed contract.
func NewProxyAdminForBinding(address common.Address, backend bind.ContractBackend) (*ProxyAdminForBinding, error) {
	contract, err := bindProxyAdminForBinding(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ProxyAdminForBinding{ProxyAdminForBindingCaller: ProxyAdminForBindingCaller{contract: contract}, ProxyAdminForBindingTransactor: ProxyAdminForBindingTransactor{contract: contract}, ProxyAdminForBindingFilterer: ProxyAdminForBindingFilterer{contract: contract}}, nil
}

// NewProxyAdminForBindingCaller creates a new read-only instance of ProxyAdminForBinding, bound to a specific deployed contract.
func NewProxyAdminForBindingCaller(address common.Address, caller bind.ContractCaller) (*ProxyAdminForBindingCaller, error) {
	contract, err := bindProxyAdminForBinding(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ProxyAdminForBindingCaller{contract: contract}, nil
}

// NewProxyAdminForBindingTransactor creates a new write-only instance of ProxyAdminForBinding, bound to a specific deployed contract.
func NewProxyAdminForBindingTransactor(address common.Address, transactor bind.ContractTransactor) (*ProxyAdminForBindingTransactor, error) {
	contract, err := bindProxyAdminForBinding(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ProxyAdminForBindingTransactor{contract: contract}, nil
}

// NewProxyAdminForBindingFilterer creates a new log filterer instance of ProxyAdminForBinding, bound to a specific deployed contract.
func NewProxyAdminForBindingFilterer(address common.Address, filterer bind.ContractFilterer) (*ProxyAdminForBindingFilterer, error) {
	contract, err := bindProxyAdminForBinding(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ProxyAdminForBindingFilterer{contract: contract}, nil
}

// bindProxyAdminForBinding binds a generic wrapper to an already deployed contract.
func bindProxyAdminForBinding(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ProxyAdminForBindingMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProxyAdminForBinding *ProxyAdminForBindingRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProxyAdminForBinding.Contract.ProxyAdminForBindingCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProxyAdminForBinding *ProxyAdminForBindingRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.ProxyAdminForBindingTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProxyAdminForBinding *ProxyAdminForBindingRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.ProxyAdminForBindingTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProxyAdminForBinding *ProxyAdminForBindingCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProxyAdminForBinding.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.contract.Transact(opts, method, params...)
}

// GetProxyAdmin is a free data retrieval call binding the contract method 0xf3b7dead.
//
// Solidity: function getProxyAdmin(address proxy) view returns(address)
func (_ProxyAdminForBinding *ProxyAdminForBindingCaller) GetProxyAdmin(opts *bind.CallOpts, proxy common.Address) (common.Address, error) {
	var out []interface{}
	err := _ProxyAdminForBinding.contract.Call(opts, &out, "getProxyAdmin", proxy)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetProxyAdmin is a free data retrieval call binding the contract method 0xf3b7dead.
//
// Solidity: function getProxyAdmin(address proxy) view returns(address)
func (_ProxyAdminForBinding *ProxyAdminForBindingSession) GetProxyAdmin(proxy common.Address) (common.Address, error) {
	return _ProxyAdminForBinding.Contract.GetProxyAdmin(&_ProxyAdminForBinding.CallOpts, proxy)
}

// GetProxyAdmin is a free data retrieval call binding the contract method 0xf3b7dead.
//
// Solidity: function getProxyAdmin(address proxy) view returns(address)
func (_ProxyAdminForBinding *ProxyAdminForBindingCallerSession) GetProxyAdmin(proxy common.Address) (common.Address, error) {
	return _ProxyAdminForBinding.Contract.GetProxyAdmin(&_ProxyAdminForBinding.CallOpts, proxy)
}

// GetProxyImplementation is a free data retrieval call binding the contract method 0x204e1c7a.
//
// Solidity: function getProxyImplementation(address proxy) view returns(address)
func (_ProxyAdminForBinding *ProxyAdminForBindingCaller) GetProxyImplementation(opts *bind.CallOpts, proxy common.Address) (common.Address, error) {
	var out []interface{}
	err := _ProxyAdminForBinding.contract.Call(opts, &out, "getProxyImplementation", proxy)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetProxyImplementation is a free data retrieval call binding the contract method 0x204e1c7a.
//
// Solidity: function getProxyImplementation(address proxy) view returns(address)
func (_ProxyAdminForBinding *ProxyAdminForBindingSession) GetProxyImplementation(proxy common.Address) (common.Address, error) {
	return _ProxyAdminForBinding.Contract.GetProxyImplementation(&_ProxyAdminForBinding.CallOpts, proxy)
}

// GetProxyImplementation is a free data retrieval call binding the contract method 0x204e1c7a.
//
// Solidity: function getProxyImplementation(address proxy) view returns(address)
func (_ProxyAdminForBinding *ProxyAdminForBindingCallerSession) GetProxyImplementation(proxy common.Address) (common.Address, error) {
	return _ProxyAdminForBinding.Contract.GetProxyImplementation(&_ProxyAdminForBinding.CallOpts, proxy)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ProxyAdminForBinding *ProxyAdminForBindingCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ProxyAdminForBinding.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ProxyAdminForBinding *ProxyAdminForBindingSession) Owner() (common.Address, error) {
	return _ProxyAdminForBinding.Contract.Owner(&_ProxyAdminForBinding.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ProxyAdminForBinding *ProxyAdminForBindingCallerSession) Owner() (common.Address, error) {
	return _ProxyAdminForBinding.Contract.Owner(&_ProxyAdminForBinding.CallOpts)
}

// ChangeProxyAdmin is a paid mutator transaction binding the contract method 0x7eff275e.
//
// Solidity: function changeProxyAdmin(address proxy, address newAdmin) returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactor) ChangeProxyAdmin(opts *bind.TransactOpts, proxy common.Address, newAdmin common.Address) (*types.Transaction, error) {
	return _ProxyAdminForBinding.contract.Transact(opts, "changeProxyAdmin", proxy, newAdmin)
}

// ChangeProxyAdmin is a paid mutator transaction binding the contract method 0x7eff275e.
//
// Solidity: function changeProxyAdmin(address proxy, address newAdmin) returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingSession) ChangeProxyAdmin(proxy common.Address, newAdmin common.Address) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.ChangeProxyAdmin(&_ProxyAdminForBinding.TransactOpts, proxy, newAdmin)
}

// ChangeProxyAdmin is a paid mutator transaction binding the contract method 0x7eff275e.
//
// Solidity: function changeProxyAdmin(address proxy, address newAdmin) returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactorSession) ChangeProxyAdmin(proxy common.Address, newAdmin common.Address) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.ChangeProxyAdmin(&_ProxyAdminForBinding.TransactOpts, proxy, newAdmin)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProxyAdminForBinding.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingSession) RenounceOwnership() (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.RenounceOwnership(&_ProxyAdminForBinding.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.RenounceOwnership(&_ProxyAdminForBinding.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _ProxyAdminForBinding.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.TransferOwnership(&_ProxyAdminForBinding.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.TransferOwnership(&_ProxyAdminForBinding.TransactOpts, newOwner)
}

// Upgrade is a paid mutator transaction binding the contract method 0x99a88ec4.
//
// Solidity: function upgrade(address proxy, address implementation) returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactor) Upgrade(opts *bind.TransactOpts, proxy common.Address, implementation common.Address) (*types.Transaction, error) {
	return _ProxyAdminForBinding.contract.Transact(opts, "upgrade", proxy, implementation)
}

// Upgrade is a paid mutator transaction binding the contract method 0x99a88ec4.
//
// Solidity: function upgrade(address proxy, address implementation) returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingSession) Upgrade(proxy common.Address, implementation common.Address) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.Upgrade(&_ProxyAdminForBinding.TransactOpts, proxy, implementation)
}

// Upgrade is a paid mutator transaction binding the contract method 0x99a88ec4.
//
// Solidity: function upgrade(address proxy, address implementation) returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactorSession) Upgrade(proxy common.Address, implementation common.Address) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.Upgrade(&_ProxyAdminForBinding.TransactOpts, proxy, implementation)
}

// UpgradeAndCall is a paid mutator transaction binding the contract method 0x9623609d.
//
// Solidity: function upgradeAndCall(address proxy, address implementation, bytes data) payable returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactor) UpgradeAndCall(opts *bind.TransactOpts, proxy common.Address, implementation common.Address, data []byte) (*types.Transaction, error) {
	return _ProxyAdminForBinding.contract.Transact(opts, "upgradeAndCall", proxy, implementation, data)
}

// UpgradeAndCall is a paid mutator transaction binding the contract method 0x9623609d.
//
// Solidity: function upgradeAndCall(address proxy, address implementation, bytes data) payable returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingSession) UpgradeAndCall(proxy common.Address, implementation common.Address, data []byte) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.UpgradeAndCall(&_ProxyAdminForBinding.TransactOpts, proxy, implementation, data)
}

// UpgradeAndCall is a paid mutator transaction binding the contract method 0x9623609d.
//
// Solidity: function upgradeAndCall(address proxy, address implementation, bytes data) payable returns()
func (_ProxyAdminForBinding *ProxyAdminForBindingTransactorSession) UpgradeAndCall(proxy common.Address, implementation common.Address, data []byte) (*types.Transaction, error) {
	return _ProxyAdminForBinding.Contract.UpgradeAndCall(&_ProxyAdminForBinding.TransactOpts, proxy, implementation, data)
}

// ProxyAdminForBindingOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the ProxyAdminForBinding contract.
type ProxyAdminForBindingOwnershipTransferredIterator struct {
	Event *ProxyAdminForBindingOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *ProxyAdminForBindingOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ProxyAdminForBindingOwnershipTransferred)
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
		it.Event = new(ProxyAdminForBindingOwnershipTransferred)
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
func (it *ProxyAdminForBindingOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ProxyAdminForBindingOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ProxyAdminForBindingOwnershipTransferred represents a OwnershipTransferred event raised by the ProxyAdminForBinding contract.
type ProxyAdminForBindingOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ProxyAdminForBinding *ProxyAdminForBindingFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ProxyAdminForBindingOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ProxyAdminForBinding.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ProxyAdminForBindingOwnershipTransferredIterator{contract: _ProxyAdminForBinding.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ProxyAdminForBinding *ProxyAdminForBindingFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ProxyAdminForBindingOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ProxyAdminForBinding.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ProxyAdminForBindingOwnershipTransferred)
				if err := _ProxyAdminForBinding.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ProxyAdminForBinding *ProxyAdminForBindingFilterer) ParseOwnershipTransferred(log types.Log) (*ProxyAdminForBindingOwnershipTransferred, error) {
	event := new(ProxyAdminForBindingOwnershipTransferred)
	if err := _ProxyAdminForBinding.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxBlobMockMetaData contains all meta data concerning the SequencerInboxBlobMock contract.
var SequencerInboxBlobMockMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"maxDataSize_\",\"type\":\"uint256\"},{\"internalType\":\"contractIReader4844\",\"name\":\"reader_\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isUsingFeeToken_\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"isDelayBufferable_\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AlreadyInit\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"AlreadyValidDASKeyset\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BadBufferConfig\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BadMaxTimeVariation\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"stored\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"received\",\"type\":\"uint256\"}],\"name\":\"BadSequencerNumber\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DataBlobsNotSupported\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"dataLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxDataLength\",\"type\":\"uint256\"}],\"name\":\"DataTooLarge\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelayProofRequired\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelayedBackwards\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelayedTooFar\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Deprecated\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ExtraGasNotUint64\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ForceIncludeBlockTooSoon\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HadZeroInit\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"IncorrectMessagePreimage\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"}],\"name\":\"InitParamZero\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidDelayedAccPreimage\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"name\":\"InvalidHeaderFlag\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"KeysetTooLarge\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MissingDataHashes\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NativeTokenMismatch\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"NoSuchKeyset\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotBatchPoster\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"NotBatchPosterManager\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotDelayBufferable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotForked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotOrigin\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotOwner\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RollupNotChanged\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keysetHash\",\"type\":\"bytes32\"}],\"name\":\"InvalidateKeyset\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"OwnerFunctionCalled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchSequenceNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"SequencerBatchData\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchSequenceNumber\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"afterAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"minTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"minBlockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxBlockNumber\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structIBridge.TimeBounds\",\"name\":\"timeBounds\",\"type\":\"tuple\"},{\"indexed\":false,\"internalType\":\"enumIBridge.BatchDataLocation\",\"name\":\"dataLocation\",\"type\":\"uint8\"}],\"name\":\"SequencerBatchDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keysetHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"SetValidKeyset\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"BROTLI_MESSAGE_HEADER_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DAS_MESSAGE_HEADER_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DATA_AUTHENTICATED_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DATA_BLOB_HEADER_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"HEADER_LENGTH\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"TREE_DAS_MESSAGE_HEADER_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ZERO_HEAVY_MESSAGE_HEADER_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2Batch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"beforeDelayedAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"blockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"inboxSeqNum\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"internalType\":\"structMessages.Message\",\"name\":\"delayedMessage\",\"type\":\"tuple\"}],\"internalType\":\"structDelayProof\",\"name\":\"delayProof\",\"type\":\"tuple\"}],\"name\":\"addSequencerL2BatchDelayProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2BatchFromBlobs\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"beforeDelayedAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"blockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"inboxSeqNum\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"internalType\":\"structMessages.Message\",\"name\":\"delayedMessage\",\"type\":\"tuple\"}],\"internalType\":\"structDelayProof\",\"name\":\"delayProof\",\"type\":\"tuple\"}],\"name\":\"addSequencerL2BatchFromBlobsDelayProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"addSequencerL2BatchFromOrigin\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2BatchFromOrigin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"beforeDelayedAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"blockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"inboxSeqNum\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"internalType\":\"structMessages.Message\",\"name\":\"delayedMessage\",\"type\":\"tuple\"}],\"internalType\":\"structDelayProof\",\"name\":\"delayProof\",\"type\":\"tuple\"}],\"name\":\"addSequencerL2BatchFromOriginDelayProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batchCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batchPosterManager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"buffer\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"bufferBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"max\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"threshold\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"prevBlockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"replenishRateInBasis\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"prevSequencedBlockNumber\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"dasKeySetInfo\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"isValidKeyset\",\"type\":\"bool\"},{\"internalType\":\"uint64\",\"name\":\"creationBlock\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"uint64[2]\",\"name\":\"l1BlockAndTime\",\"type\":\"uint64[2]\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"forceInclusion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"blockNumber\",\"type\":\"uint64\"}],\"name\":\"forceInclusionDeadline\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"getKeysetCreationBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"inboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"bridge_\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"threshold\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"max\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"replenishRateInBasis\",\"type\":\"uint64\"}],\"internalType\":\"structBufferConfig\",\"name\":\"bufferConfig_\",\"type\":\"tuple\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"invalidateKeysetHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isBatchPoster\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isDelayBufferable\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isSequencer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isUsingFeeToken\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"isValidKeysetHash\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxDataSize\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxTimeVariation\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint64\",\"name\":\"threshold\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"max\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"replenishRateInBasis\",\"type\":\"uint64\"}],\"internalType\":\"structBufferConfig\",\"name\":\"bufferConfig_\",\"type\":\"tuple\"}],\"name\":\"postUpgradeInit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"reader4844\",\"outputs\":[{\"internalType\":\"contractIReader4844\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"removeDelayAfterFork\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newBatchPosterManager\",\"type\":\"address\"}],\"name\":\"setBatchPosterManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint64\",\"name\":\"threshold\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"max\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"replenishRateInBasis\",\"type\":\"uint64\"}],\"internalType\":\"structBufferConfig\",\"name\":\"bufferConfig_\",\"type\":\"tuple\"}],\"name\":\"setBufferConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isBatchPoster_\",\"type\":\"bool\"}],\"name\":\"setIsBatchPoster\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isSequencer_\",\"type\":\"bool\"}],\"name\":\"setIsSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"}],\"name\":\"setMaxTimeVariation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"setValidKeyset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalDelayedMessagesRead\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x610180604052306080526202000060a05246610100526200002b62000115602090811b62002d7717901c565b1515610120523480156200003e57600080fd5b50604051620052be380380620052be8339810160408190526200006191620001c8565b838383838360e081815250506101205115620000a6576001600160a01b03831615620000a0576040516386657a5360e01b815260040160405180910390fd5b620000ef565b6001600160a01b038316620000ef576040516380fc2c0360e01b815260206004820152600a60248201526914995859195c8d0e0d0d60b21b604482015260640160405180910390fd5b6001600160a01b0390921660c052151561014052151561016052506200025b9350505050565b60408051600481526024810182526020810180516001600160e01b03166302881c7960e11b1790529051600091829182916064916200015591906200022a565b600060405180830381855afa9150503d806000811462000192576040519150601f19603f3d011682016040523d82523d6000602084013e62000197565b606091505b5091509150818015620001ab575080516020145b9250505090565b80518015158114620001c357600080fd5b919050565b60008060008060808587031215620001df57600080fd5b845160208601519094506001600160a01b0381168114620001ff57600080fd5b92506200020f60408601620001b2565b91506200021f60608601620001b2565b905092959194509250565b6000825160005b818110156200024d576020818601810151858301520162000231565b506000920191825250919050565b60805160a05160c05160e05161010051610120516101405161016051614f5c620003626000396000818161043701528181610b86015281816113550152818161184401528181611ed701528181612339015281816128c201528181612a5701528181612f6b01526131ad0152600081816105f401528181610a450152818161331b01526139950152600081816126a3015281816132b90152613b7b015260008181612194015261360c01526000818161070e01528181613e640152613eb901526000818161058f01528181610ffd0152611e800152600081816111dd0152818161151101528181611d75015261208a01526000818161089301526122130152614f5c6000f3fe608060405234801561001057600080fd5b50600436106102f45760003560e01c80637fa3a40e11610191578063d1ce8da8116100e3578063e78cea9211610097578063edaafe2011610071578063edaafe2014610758578063f1981578146107e1578063f60a5091146107f457600080fd5b8063e78cea92146106f6578063e8eb1dc314610709578063ebea461d1461073057600080fd5b8063dd44e6e0116100c8578063dd44e6e014610690578063e0bc9729146106bc578063e5a358c8146106cf57600080fd5b8063d1ce8da81461066a578063d9dd67ab1461067d57600080fd5b806392d9f78211610145578063b31761f81161011f578063b31761f814610631578063cb23bcb514610644578063cc2a1a0c1461065757600080fd5b806392d9f782146105ef57806396cc5c7814610616578063a655d9371461061e57600080fd5b80638d910dde116101765780638d910dde1461058a5780638f111f3c146105c9578063917cf8ac146105dc57600080fd5b80637fa3a40e1461056e578063844208601461057757600080fd5b80633e5aa0821161024a5780636d46e987116101fe5780636f12b0c9116101d85780636f12b0c9146104e4578063715ea34b146104f757806371c3e6fe1461054b57600080fd5b80636d46e9871461049b5780636e620055146104be5780636e7df3e7146104d157600080fd5b806369cacded1161022f57806369cacded146104595780636ae71f121461046c5780636c8904501461047457600080fd5b80633e5aa0821461041f5780634b678a661461043257600080fd5b80631f956632116102ac57806327957a491161028657806327957a49146103dd5780632cbf74e5146103e55780632f3985a71461040c57600080fd5b80631f956632146103a45780631ff64790146103b7578063258f0495146103ca57600080fd5b80631637be48116102dd5780631637be481461035457806316af91a7146103875780631ad87e451461038f57600080fd5b806302c99275146102f957806306f130561461033e575b600080fd5b6103207f200000000000000000000000000000000000000000000000000000000000000081565b6040516001600160f81b031990911681526020015b60405180910390f35b6103466107ff565b604051908152602001610335565b610377610362366004614385565b60009081526008602052604090205460ff1690565b6040519015158152602001610335565b610320600081565b6103a261039d3660046144d2565b610889565b005b6103a26103b2366004614533565b610bb9565b6103a26103c536600461456c565b610ce5565b6103466103d8366004614385565b610e7d565b610346602881565b6103207f500000000000000000000000000000000000000000000000000000000000000081565b6103a261041a366004614589565b610eea565b6103a261042d3660046145a5565b610ffa565b6103777f000000000000000000000000000000000000000000000000000000000000000081565b6103a2610467366004614650565b6112e3565b6103a2611619565b6103207f080000000000000000000000000000000000000000000000000000000000000081565b6103776104a936600461456c565b60096020526000908152604090205460ff1681565b6103a26104cc366004614650565b6117f1565b6103a26104df366004614533565b6118a3565b6103a26104f23660046146de565b6119cf565b61052b610505366004614385565b60086020526000908152604090205460ff811690610100900467ffffffffffffffff1682565b60408051921515835267ffffffffffffffff909116602083015201610335565b61037761055936600461456c565b60036020526000908152604090205460ff1681565b61034660005481565b6103a2610585366004614385565b611a01565b6105b17f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b039091168152602001610335565b6103a26105d7366004614749565b611b76565b6103a26105ea3660046147c6565b611e7d565b6103777f000000000000000000000000000000000000000000000000000000000000000081565b6103a2612191565b6103a261062c366004614589565b612209565b6103a261063f366004614822565b6123c7565b6002546105b1906001600160a01b031681565b600b546105b1906001600160a01b031681565b6103a2610678366004614888565b6124d6565b61034661068b366004614385565b612823565b6106a361069e3660046148ca565b6128b0565b60405167ffffffffffffffff9091168152602001610335565b6103a26106ca366004614749565b612913565b6103207f400000000000000000000000000000000000000000000000000000000000000081565b6001546105b1906001600160a01b031681565b6103467f000000000000000000000000000000000000000000000000000000000000000081565b61073861299b565b604080519485526020850193909352918301526060820152608001610335565b600c54600d5461079e9167ffffffffffffffff8082169268010000000000000000808404831693600160801b8104841693600160c01b9091048116928082169290041686565b6040805167ffffffffffffffff978816815295871660208701529386169385019390935290841660608401528316608083015290911660a082015260c001610335565b6103a26107ef3660046148f6565b6129d4565b610320600160ff1b81565b600154604080517e84120c00000000000000000000000000000000000000000000000000000000815290516000926001600160a01b0316916284120c9160048083019260209291908290030181865afa158015610860573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610884919061495e565b905090565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610946576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084015b60405180910390fd5b6001546001600160a01b031615610989576040517fef34ca5c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001600160a01b0383166109c9576040517f1ad0f74300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000836001600160a01b031663e1758bd86040518163ffffffff1660e01b8152600401602060405180830381865afa925050508015610a25575060408051601f3d908101601f19168201909252610a2291810190614977565b60015b15610a40576001600160a01b03811615610a3e57600191505b505b8015157f0000000000000000000000000000000000000000000000000000000000000000151514610a9d576040517fc3e31f8d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038616908117909155604080517fcb23bcb5000000000000000000000000000000000000000000000000000000008152905163cb23bcb5916004808201926020929091908290030181865afa158015610b1c573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b409190614977565b6002805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0392909216919091179055610b84610b7f36859003850185614822565b612e3d565b7f000000000000000000000000000000000000000000000000000000000000000015610bb357610bb382612f69565b50505050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610c0c573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610c309190614977565b6001600160a01b0316336001600160a01b031614158015610c5c5750600b546001600160a01b03163314155b15610c95576040517f660b3b4200000000000000000000000000000000000000000000000000000000815233600482015260240161093d565b6001600160a01b038216600090815260096020526040808220805460ff1916841515179055516004917fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e91a25050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610d38573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610d5c9190614977565b6001600160a01b0316336001600160a01b031614610e265760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dbd573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610de19190614977565b6040517f23295f0e0000000000000000000000000000000000000000000000000000000081526001600160a01b0392831660048201529116602482015260440161093d565b600b805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0383161790556040516005907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a250565b600081815260086020908152604080832081518083019092525460ff811615158252610100900467ffffffffffffffff16918101829052908203610ed65760405162f20c5d60e01b81526004810184905260240161093d565b6020015167ffffffffffffffff1692915050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610f3d573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610f619190614977565b6001600160a01b0316336001600160a01b031614610fc25760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dbd573d6000803e3d6000fd5b610fcb81612f69565b6040516006907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a250565b827f000000000000000000000000000000000000000000000000000000000000000060005a3360009081526003602052604090205490915060ff1661105257604051632dd9fc9760e01b815260040160405180910390fd5b61105b876131a9565b1561107957604051630e5da8fb60e01b815260040160405180910390fd5b611085888887876131f1565b6001600160a01b038316156112d95736600060206110a483601f6149aa565b6110ae91906149bd565b90506102006110be600283614ac3565b6110c891906149bd565b6110d3826006614ad2565b6110dd91906149aa565b6110e790846149aa565b92503332146110f9576000915061122c565b6001600160a01b0384161561122c57836001600160a01b031663e83a2d826040518163ffffffff1660e01b8152600401600060405180830381865afa92505050801561116757506040513d6000823e601f3d908101601f191682016040526111649190810190614ae9565b60015b1561122c5780511561122a576000856001600160a01b0316631f6d6ef76040518163ffffffff1660e01b8152600401602060405180830381865afa1580156111b3573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906111d7919061495e565b905048817f000000000000000000000000000000000000000000000000000000000000000084516112089190614ad2565b6112129190614ad2565b61121c91906149bd565b61122690866149aa565b9450505b505b846001600160a01b031663e3db8a49335a6112479087614b8f565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af11580156112b1573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906112d59190614ba2565b5050505b5050505050505050565b836000805a9050333214611323576040517ffeb3d07100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526003602052604090205460ff1661135357604051632dd9fc9760e01b815260040160405180910390fd5b7f000000000000000000000000000000000000000000000000000000000000000061139157604051631192b39960e31b815260040160405180910390fd5b6113a9886113a436879003870187614bbf565b61334d565b6113b98b8b8b8b8a8a600161345a565b6001600160a01b038316156112d55736600060206113d883601f6149aa565b6113e291906149bd565b90506102006113f2600283614ac3565b6113fc91906149bd565b611407826006614ad2565b61141191906149aa565b61141b90846149aa565b925033321461142d5760009150611560565b6001600160a01b0384161561156057836001600160a01b031663e83a2d826040518163ffffffff1660e01b8152600401600060405180830381865afa92505050801561149b57506040513d6000823e601f3d908101601f191682016040526114989190810190614ae9565b60015b156115605780511561155e576000856001600160a01b0316631f6d6ef76040518163ffffffff1660e01b8152600401602060405180830381865afa1580156114e7573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061150b919061495e565b905048817f0000000000000000000000000000000000000000000000000000000000000000845161153c9190614ad2565b6115469190614ad2565b61155091906149bd565b61155a90866149aa565b9450505b505b846001600160a01b031663e3db8a49335a61157b9087614b8f565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af11580156115e5573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906116099190614ba2565b5050505050505050505050505050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561166c573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906116909190614977565b6001600160a01b0316336001600160a01b0316146116f15760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dbd573d6000803e3d6000fd5b600154604080517fcb23bcb500000000000000000000000000000000000000000000000000000000815290516000926001600160a01b03169163cb23bcb59160048083019260209291908290030181865afa158015611754573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906117789190614977565b6002549091506001600160a01b038083169116036117c2576040517fd054909f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6002805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0392909216919091179055565b836000805a3360009081526003602052604090205490915060ff1615801561182457506002546001600160a01b03163314155b1561184257604051632dd9fc9760e01b815260040160405180910390fd5b7f000000000000000000000000000000000000000000000000000000000000000061188057604051631192b39960e31b815260040160405180910390fd5b611893886113a436879003870187614bbf565b6113b98b8b8b8b8a8a600061345a565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156118f6573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061191a9190614977565b6001600160a01b0316336001600160a01b0316141580156119465750600b546001600160a01b03163314155b1561197f576040517f660b3b4200000000000000000000000000000000000000000000000000000000815233600482015260240161093d565b6001600160a01b038216600090815260036020526040808220805460ff1916841515179055516001917fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e91a25050565b6040517fc73b9d7c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611a54573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611a789190614977565b6001600160a01b0316336001600160a01b031614611ad95760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dbd573d6000803e3d6000fd5b60008181526008602052604090205460ff16611b0a5760405162f20c5d60e01b81526004810182905260240161093d565b600081815260086020526040808220805460ff191690555182917f5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a91a26040516003907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a250565b826000805a9050333214611bb6576040517ffeb3d07100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526003602052604090205460ff16611be657604051632dd9fc9760e01b815260040160405180910390fd5b611bef876131a9565b15611c0d57604051630e5da8fb60e01b815260040160405180910390fd5b611c1d8a8a8a8a8989600161345a565b6001600160a01b03831615611e71573660006020611c3c83601f6149aa565b611c4691906149bd565b9050610200611c56600283614ac3565b611c6091906149bd565b611c6b826006614ad2565b611c7591906149aa565b611c7f90846149aa565b9250333214611c915760009150611dc4565b6001600160a01b03841615611dc457836001600160a01b031663e83a2d826040518163ffffffff1660e01b8152600401600060405180830381865afa925050508015611cff57506040513d6000823e601f3d908101601f19168201604052611cfc9190810190614ae9565b60015b15611dc457805115611dc2576000856001600160a01b0316631f6d6ef76040518163ffffffff1660e01b8152600401602060405180830381865afa158015611d4b573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611d6f919061495e565b905048817f00000000000000000000000000000000000000000000000000000000000000008451611da09190614ad2565b611daa9190614ad2565b611db491906149bd565b611dbe90866149aa565b9450505b505b846001600160a01b031663e3db8a49335a611ddf9087614b8f565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af1158015611e49573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611e6d9190614ba2565b5050505b50505050505050505050565b837f000000000000000000000000000000000000000000000000000000000000000060005a3360009081526003602052604090205490915060ff16611ed557604051632dd9fc9760e01b815260040160405180910390fd5b7f0000000000000000000000000000000000000000000000000000000000000000611f1357604051631192b39960e31b815260040160405180910390fd5b611f26886113a436879003870187614bbf565b611f32898988886131f1565b6001600160a01b03831615612186573660006020611f5183601f6149aa565b611f5b91906149bd565b9050610200611f6b600283614ac3565b611f7591906149bd565b611f80826006614ad2565b611f8a91906149aa565b611f9490846149aa565b9250333214611fa657600091506120d9565b6001600160a01b038416156120d957836001600160a01b031663e83a2d826040518163ffffffff1660e01b8152600401600060405180830381865afa92505050801561201457506040513d6000823e601f3d908101601f191682016040526120119190810190614ae9565b60015b156120d9578051156120d7576000856001600160a01b0316631f6d6ef76040518163ffffffff1660e01b8152600401602060405180830381865afa158015612060573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612084919061495e565b905048817f000000000000000000000000000000000000000000000000000000000000000084516120b59190614ad2565b6120bf9190614ad2565b6120c991906149bd565b6120d390866149aa565b9450505b505b846001600160a01b031663e3db8a49335a6120f49087614b8f565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af115801561215e573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906121829190614ba2565b5050505b505050505050505050565b467f0000000000000000000000000000000000000000000000000000000000000000036121ea576040517fa301bb0600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b7801000000000000000100000000000000010000000000000001600a55565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036122c1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c0000000000000000000000000000000000000000606482015260840161093d565b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038054336001600160a01b03821614612337576040517f23295f0e0000000000000000000000000000000000000000000000000000000081523360048201526001600160a01b038216602482015260440161093d565b7f000000000000000000000000000000000000000000000000000000000000000061237557604051631192b39960e31b815260040160405180910390fd5b600c5467ffffffffffffffff16156123b9576040517fef34ca5c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6123c283612f69565b505050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561241a573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061243e9190614977565b6001600160a01b0316336001600160a01b03161461249f5760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dbd573d6000803e3d6000fd5b6124a881612e3d565b6040516000907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e908290a250565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015612529573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061254d9190614977565b6001600160a01b0316336001600160a01b0316146125ae5760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dbd573d6000803e3d6000fd5b600082826040516125c0929190614c6d565b6040519081900381207ffe000000000000000000000000000000000000000000000000000000000000006020830152602182015260410160408051601f1981840301815291905280516020909101209050600160ff1b8118620100008310612654576040517fb3d1f41200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008181526008602052604090205460ff16156126a0576040517ffa2fddda0000000000000000000000000000000000000000000000000000000081526004810182905260240161093d565b437f00000000000000000000000000000000000000000000000000000000000000001561272d5760646001600160a01b031663a3b1b31d6040518163ffffffff1660e01b8152600401602060405180830381865afa158015612706573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061272a919061495e565b90505b6040805180820182526001815267ffffffffffffffff8381166020808401918252600087815260089091528490209251835491517fffffffffffffffffffffffffffffffffffffffffffffff0000000000000000009092169015157fffffffffffffffffffffffffffffffffffffffffffffff0000000000000000ff161761010091909216021790555182907fabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722906127e89088908890614c7d565b60405180910390a26040516002907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a25050505050565b6001546040517f16bf5579000000000000000000000000000000000000000000000000000000008152600481018390526000916001600160a01b0316906316bf557990602401602060405180830381865afa158015612886573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906128aa919061495e565b92915050565b600a5460009067ffffffffffffffff167f0000000000000000000000000000000000000000000000000000000000000000156129025760006128f3600c85613585565b90506128fe816135d4565b9150505b61290c8184614cac565b9392505050565b826000805a3360009081526003602052604090205490915060ff1615801561294657506002546001600160a01b03163314155b1561296457604051632dd9fc9760e01b815260040160405180910390fd5b61296d876131a9565b1561298b57604051630e5da8fb60e01b815260040160405180910390fd5b611c1d8a8a8a8a8989600061345a565b6000806000806000806000806129af613604565b67ffffffffffffffff9384169b50918316995082169750169450505050505b90919293565b6000548611612a0f576040517f7d73e6fa00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000612a458684612a2360208901896148ca565b612a3360408a0160208b016148ca565b612a3e60018d614b8f565b898861367b565b600a5490915067ffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000015612ab657612a93612a8b60208801886148ca565b600c90613720565b600c54612aa99067ffffffffffffffff166135d4565b67ffffffffffffffff1690505b4381612ac560208901896148ca565b67ffffffffffffffff16612ad991906149aa565b10612b10576040517fad3515d900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006001891115612b99576001546001600160a01b031663d5719dc2612b3760028c614b8f565b6040518263ffffffff1660e01b8152600401612b5591815260200190565b602060405180830381865afa158015612b72573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612b96919061495e565b90505b60408051602080820184905281830186905282518083038401815260609092019092528051910120600180546001600160a01b03169063d5719dc290612bdf908d614b8f565b6040518263ffffffff1660e01b8152600401612bfd91815260200190565b602060405180830381865afa158015612c1a573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612c3e919061495e565b14612c75576040517f13947fd700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600080612c818b6137a6565b9150915060008b90506000600160009054906101000a90046001600160a01b03166001600160a01b0316635fca4a166040518163ffffffff1660e01b8152600401602060405180830381865afa158015612cdf573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612d03919061495e565b9050806000808080612d1889888388806137eb565b93509350935093508083857f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548d6002604051612d5b9493929190614cea565b60405180910390a4505050505050505050505050505050505050565b60408051600481526024810182526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f051038f200000000000000000000000000000000000000000000000000000000179052905160009182918291606491612de39190614d83565b600060405180830381855afa9150503d8060008114612e1e576040519150601f19603f3d011682016040523d82523d6000602084013e612e23565b606091505b5091509150818015612e36575080516020145b9250505090565b805167ffffffffffffffff1080612e5f5750602081015167ffffffffffffffff105b80612e755750604081015167ffffffffffffffff105b80612e8b5750606081015167ffffffffffffffff105b15612ec2576040517f09cfba7500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8051600a80546020840151604085015160609095015167ffffffffffffffff908116600160c01b0277ffffffffffffffffffffffffffffffffffffffffffffffff968216600160801b02969096166fffffffffffffffffffffffffffffffff92821668010000000000000000027fffffffffffffffffffffffffffffffff000000000000000000000000000000009094169190951617919091171691909117919091179055565b7f0000000000000000000000000000000000000000000000000000000000000000612fa757604051631192b39960e31b815260040160405180910390fd5b612fb0816139d4565b612fe6576040517fda1c8eb600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600c5467ffffffffffffffff16158061301257506020810151600c5467ffffffffffffffff9182169116115b1561303e576020810151600c805467ffffffffffffffff191667ffffffffffffffff9092169190911790555b8051600c5467ffffffffffffffff9182169116101561307b578051600c805467ffffffffffffffff191667ffffffffffffffff9092169190911790555b602081810151600c805484517fffffffffffffffff00000000000000000000000000000000ffffffffffffffff9091166801000000000000000067ffffffffffffffff948516027fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff1617600160801b91841691909102179055604080840151600d805467ffffffffffffffff1916919093161790915560005460015482517feca067ad000000000000000000000000000000000000000000000000000000008152925191936001600160a01b039091169263eca067ad92600480830193928290030181865afa158015613172573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190613196919061495e565b036131a6576131a6600c43613720565b50565b60007f000000000000000000000000000000000000000000000000000000000000000080156131d9575060005482115b80156128aa57506131ea600c613a3c565b1592915050565b60008060006131ff86613a6f565b925092509250600080600080613219878b60008c8c6137eb565b93509350935093508a841415801561323357506000198b14155b15613274576040517fac7411c900000000000000000000000000000000000000000000000000000000815260048101859052602481018c905260440161093d565b80838c7f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548b60036040516132af9493929190614cea565b60405180910390a47f00000000000000000000000000000000000000000000000000000000000000001561330f576040517f86657a5300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b333214801561333c57507f0000000000000000000000000000000000000000000000000000000000000000155b156112d5576112d587854888613b78565b60005482111561345657613361600c613dba565b1561345657600154600080546040517fd5719dc200000000000000000000000000000000000000000000000000000000815291926001600160a01b03169163d5719dc2916133b59160040190815260200190565b602060405180830381865afa1580156133d2573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906133f6919061495e565b905061340b8183600001518460200151613deb565b613441576040517fc334542d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6020820151604001516123c290600c90613720565b5050565b600080613468888888613e30565b9150915060008060008061348c868b89613483576000613485565b8d5b8c8c6137eb565b93509350935093508c84141580156134a657506000198d14155b156134e7576040517fac7411c900000000000000000000000000000000000000000000000000000000815260048101859052602481018e905260440161093d565b8083857f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548a8d61351c57600161351f565b60005b60405161352f9493929190614cea565b60405180910390a486611e6d57837ffe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c208d8d60405161356e929190614c7d565b60405180910390a250505050505050505050505050565b8154600183015460009161290c9167ffffffffffffffff600160c01b8304811692868216928282169268010000000000000000808304821693600160801b81048316939190048216911661403e565b600a5460009067ffffffffffffffff9081169083161061360057600a5467ffffffffffffffff166128aa565b5090565b6000808080467f000000000000000000000000000000000000000000000000000000000000000014613641575060019250829150819050806129ce565b5050600a5467ffffffffffffffff808216935068010000000000000000820481169250600160801b8204811691600160c01b9004166129ce565b6040516001600160f81b031960f889901b1660208201526bffffffffffffffffffffffff19606088901b1660218201527fffffffffffffffff00000000000000000000000000000000000000000000000060c087811b8216603584015286901b16603d82015260458101849052606581018390526085810182905260009060a5016040516020818303038152906040528051906020012090505b979650505050505050565b61372a8282613585565b825467ffffffffffffffff928316600160c01b0277ffffffffffffffffffffffffffffffff000000000000000090911691831691909117178255600190910180544390921668010000000000000000027fffffffffffffffffffffffffffffffff0000000000000000ffffffffffffffff909216919091179055565b60408051608081018252600080825260208201819052918101829052606081018290526000806137d585614105565b8151602090920191909120969095509350505050565b60008060008060005488101561382d576040517f7d73e6fa00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600160009054906101000a90046001600160a01b03166001600160a01b031663eca067ad6040518163ffffffff1660e01b8152600401602060405180830381865afa158015613880573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906138a4919061495e565b8811156138dd576040517f925f8bd300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001546040517f86598a56000000000000000000000000000000000000000000000000000000008152600481018b9052602481018a905260448101889052606481018790526001600160a01b03909116906386598a56906084016080604051808303816000875af1158015613956573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061397a9190614d9f565b60008c90559296509094509250905086158015906139b657507f0000000000000000000000000000000000000000000000000000000000000000155b156139c8576139c88985486000613b78565b95509550955095915050565b805160009067ffffffffffffffff16158015906139fe5750602082015167ffffffffffffffff1615155b8015613a1a5750612710826040015167ffffffffffffffff1611155b80156128aa5750506020810151905167ffffffffffffffff9182169116111590565b805460009067ffffffffffffffff600160801b8204811691613a6791600160c01b9091041643614b8f565b111592915050565b604080516080810182526000808252602082018190529181018290526060810182905260408051606081018252600080825260208201819052918101829052600080613aba87614105565b9092509050633b9aca0060006003613ad56202000084614ad2565b613adf9190614ad2565b60405190915084907f500000000000000000000000000000000000000000000000000000000000000090613b17908890602001614dd5565b60408051601f1981840301815290829052613b36939291602001614e09565b604051602081830303815290604052805190602001208360004811613b5c576000613b66565b613b6648846149bd565b97509750975050505050509193909250565b327f000000000000000000000000000000000000000000000000000000000000000015613c1e576000606c6001600160a01b031663c6f7de0e6040518163ffffffff1660e01b8152600401602060405180830381865afa158015613be0573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190613c04919061495e565b9050613c1048826149bd565b613c1a90846149aa565b9250505b67ffffffffffffffff821115613c60576040517f04d5501200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b604080514260208201526bffffffffffffffffffffffff19606084901b16918101919091526054810186905260748101859052609481018490527fffffffffffffffff00000000000000000000000000000000000000000000000060c084901b1660b482015260009060bc0160408051808303601f1901815290829052600154815160208301207f7a88b1070000000000000000000000000000000000000000000000000000000084526001600160a01b0386811660048601526024850191909152919350600092911690637a88b107906044016020604051808303816000875af1158015613d53573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190613d77919061495e565b9050807fff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b83604051613da99190614e4c565b60405180910390a250505050505050565b6000613dc582613a3c565b15806128aa5750505467ffffffffffffffff680100000000000000008204811691161090565b6000613e2683613dfa846141dd565b604080516020808201949094528082019290925280518083038201815260609092019052805191012090565b9093149392505050565b60408051608081018252600080825260208201819052918101829052606081018290526000613e608560286149aa565b90507f0000000000000000000000000000000000000000000000000000000000000000811115613ee5576040517f4634691b000000000000000000000000000000000000000000000000000000008152600481018290527f0000000000000000000000000000000000000000000000000000000000000000602482015260440161093d565b600080613ef186614105565b9092509050861561400457613f2188886000818110613f1257613f12614cd4565b9050013560f81c60f81b61420a565b613f795787876000818110613f3857613f38614cd4565b6040517f6b3333560000000000000000000000000000000000000000000000000000000081529201356001600160f81b03191660048301525060240161093d565b600160ff1b8888600081613f8f57613f8f614cd4565b6001600160f81b031992013592909216161580159150613fb0575060218710155b15614004576000613fc5602160018a8c614e7f565b613fce91614ea9565b60008181526008602052604090205490915060ff166140025760405162f20c5d60e01b81526004810182905260240161093d565b505b81888860405160200161401993929190614ec7565b60408051601f1981840301815291905280516020909101209890975095505050505050565b60008088881161404f576000614059565b6140598989614b8f565b9050600089871161406b576000614075565b6140758a88614b8f565b90506127106140848584614ad2565b61408e91906149bd565b61409890896149aa565b975060008682116140aa5760006140b4565b6140b48783614b8f565b9050828111156140c15750815b808911156140f6576140d3818a614b8f565b9850868911156140f6578589116140ea57886140ec565b855b9350505050613715565b50949998505050505050505050565b6040805160808101825260008082526020820181905291810182905260608082018390529161413261429d565b905060008160000151826020015183604001518460600151886040516020016141b295949392919060c095861b7fffffffffffffffff000000000000000000000000000000000000000000000000908116825294861b8516600882015292851b8416601084015290841b8316601883015290921b16602082015260280190565b604051602081830303815290604052905060288151146141d4576141d4614eef565b94909350915050565b60006128aa826000015183602001518460400151856060015186608001518760a001518860c0015161367b565b60006001600160f81b03198216158061423057506001600160f81b03198216600160ff1b145b8061426457506001600160f81b031982167f8800000000000000000000000000000000000000000000000000000000000000145b806128aa57506001600160f81b031982167f20000000000000000000000000000000000000000000000000000000000000001492915050565b6040805160808101825260008082526020820181905291810182905260608101919091526040805160808101825260008082526020820181905291810182905260608101919091526000806000806142f3613604565b93509350935093508167ffffffffffffffff16421115614324576143178242614f05565b67ffffffffffffffff1685525b61432e8142614cac565b67ffffffffffffffff90811660208701528416431115614362576143528443614f05565b67ffffffffffffffff1660408601525b61436c8343614cac565b67ffffffffffffffff1660608601525092949350505050565b60006020828403121561439757600080fd5b5035919050565b6001600160a01b03811681146131a657600080fd5b634e487b7160e01b600052604160045260246000fd5b6040805190810167ffffffffffffffff811182821017156143ec576143ec6143b3565b60405290565b60405160e0810167ffffffffffffffff811182821017156143ec576143ec6143b3565b604051601f8201601f1916810167ffffffffffffffff8111828210171561443e5761443e6143b3565b604052919050565b803567ffffffffffffffff8116811461445e57600080fd5b919050565b60006060828403121561447557600080fd5b6040516060810181811067ffffffffffffffff82111715614498576144986143b3565b6040529050806144a783614446565b81526144b560208401614446565b60208201526144c660408401614446565b60408201525092915050565b60008060008385036101008112156144e957600080fd5b84356144f48161439e565b93506080601f198201121561450857600080fd5b5060208401915061451c8560a08601614463565b90509250925092565b80151581146131a657600080fd5b6000806040838503121561454657600080fd5b82356145518161439e565b9150602083013561456181614525565b809150509250929050565b60006020828403121561457e57600080fd5b813561290c8161439e565b60006060828403121561459b57600080fd5b61290c8383614463565b600080600080600060a086880312156145bd57600080fd5b853594506020860135935060408601356145d68161439e565b94979396509394606081013594506080013592915050565b60008083601f84011261460057600080fd5b50813567ffffffffffffffff81111561461857600080fd5b60208301915083602082850101111561463057600080fd5b9250929050565b6000610100828403121561464a57600080fd5b50919050565b6000806000806000806000806101c0898b03121561466d57600080fd5b88359750602089013567ffffffffffffffff81111561468b57600080fd5b6146978b828c016145ee565b9098509650506040890135945060608901356146b28161439e565b93506080890135925060a089013591506146cf8a60c08b01614637565b90509295985092959890939650565b6000806000806000608086880312156146f657600080fd5b85359450602086013567ffffffffffffffff81111561471457600080fd5b614720888289016145ee565b90955093505060408601359150606086013561473b8161439e565b809150509295509295909350565b600080600080600080600060c0888a03121561476457600080fd5b87359650602088013567ffffffffffffffff81111561478257600080fd5b61478e8a828b016145ee565b9097509550506040880135935060608801356147a98161439e565b969995985093969295946080840135945060a09093013592915050565b6000806000806000806101a087890312156147e057600080fd5b863595506020870135945060408701356147f98161439e565b935060608701359250608087013591506148168860a08901614637565b90509295509295509295565b60006080828403121561483457600080fd5b6040516080810181811067ffffffffffffffff82111715614857576148576143b3565b8060405250823581526020830135602082015260408301356040820152606083013560608201528091505092915050565b6000806020838503121561489b57600080fd5b823567ffffffffffffffff8111156148b257600080fd5b6148be858286016145ee565b90969095509350505050565b6000602082840312156148dc57600080fd5b61290c82614446565b803560ff8116811461445e57600080fd5b60008060008060008060e0878903121561490f57600080fd5b8635955061491f602088016148e5565b9450608087018881111561493257600080fd5b60408801945035925060a08701356149498161439e565b8092505060c087013590509295509295509295565b60006020828403121561497057600080fd5b5051919050565b60006020828403121561498957600080fd5b815161290c8161439e565b634e487b7160e01b600052601160045260246000fd5b808201808211156128aa576128aa614994565b6000826149da57634e487b7160e01b600052601260045260246000fd5b500490565b600181815b80851115614a1a578160001904821115614a0057614a00614994565b80851615614a0d57918102915b93841c93908002906149e4565b509250929050565b600082614a31575060016128aa565b81614a3e575060006128aa565b8160018114614a545760028114614a5e57614a7a565b60019150506128aa565b60ff841115614a6f57614a6f614994565b50506001821b6128aa565b5060208310610133831016604e8410600b8410161715614a9d575081810a6128aa565b614aa783836149df565b8060001904821115614abb57614abb614994565b029392505050565b600061290c60ff841683614a22565b80820281158282048414176128aa576128aa614994565b60006020808385031215614afc57600080fd5b825167ffffffffffffffff80821115614b1457600080fd5b818501915085601f830112614b2857600080fd5b815181811115614b3a57614b3a6143b3565b8060051b9150614b4b848301614415565b8181529183018401918481019088841115614b6557600080fd5b938501935b83851015614b8357845182529385019390850190614b6a565b98975050505050505050565b818103818111156128aa576128aa614994565b600060208284031215614bb457600080fd5b815161290c81614525565b6000818303610100811215614bd357600080fd5b614bdb6143c9565b8335815260e0601f1983011215614bf157600080fd5b614bf96143f2565b9150614c07602085016148e5565b82526040840135614c178161439e565b6020830152614c2860608501614446565b6040830152614c3960808501614446565b606083015260a0840135608083015260c084013560a083015260e084013560c0830152816020820152809250505092915050565b8183823760009101908152919050565b60208152816020820152818360408301376000818301604090810191909152601f909201601f19160101919050565b67ffffffffffffffff818116838216019080821115614ccd57614ccd614994565b5092915050565b634e487b7160e01b600052603260045260246000fd5b600060e08201905085825284602083015267ffffffffffffffff8085511660408401528060208601511660608401528060408601511660808401528060608601511660a08401525060048310614d5057634e487b7160e01b600052602160045260246000fd5b8260c083015295945050505050565b60005b83811015614d7a578181015183820152602001614d62565b50506000910152565b60008251614d95818460208701614d5f565b9190910192915050565b60008060008060808587031215614db557600080fd5b505082516020840151604085015160609095015191969095509092509050565b60008183825b6003811015614dfa578151835260209283019290910190600101614ddb565b50505060608201905092915050565b60008451614e1b818460208901614d5f565b6001600160f81b031985169083019081528351614e3f816001840160208801614d5f565b0160010195945050505050565b6020815260008251806020840152614e6b816040850160208701614d5f565b601f01601f19169190910160400192915050565b60008085851115614e8f57600080fd5b83861115614e9c57600080fd5b5050820193919092039150565b803560208310156128aa57600019602084900360031b1b1692915050565b60008451614ed9818460208901614d5f565b8201838582376000930192835250909392505050565b634e487b7160e01b600052600160045260246000fd5b67ffffffffffffffff828116828216039080821115614ccd57614ccd61499456fea264697066735822122000b1be9ec8040251bfdf78e1846cf01c07b8b19096fa40fe826a0430b66f36f364736f6c63430008110033",
}

// SequencerInboxBlobMockABI is the input ABI used to generate the binding from.
// Deprecated: Use SequencerInboxBlobMockMetaData.ABI instead.
var SequencerInboxBlobMockABI = SequencerInboxBlobMockMetaData.ABI

// SequencerInboxBlobMockBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SequencerInboxBlobMockMetaData.Bin instead.
var SequencerInboxBlobMockBin = SequencerInboxBlobMockMetaData.Bin

// DeploySequencerInboxBlobMock deploys a new Ethereum contract, binding an instance of SequencerInboxBlobMock to it.
func DeploySequencerInboxBlobMock(auth *bind.TransactOpts, backend bind.ContractBackend, maxDataSize_ *big.Int, reader_ common.Address, isUsingFeeToken_ bool, isDelayBufferable_ bool) (common.Address, *types.Transaction, *SequencerInboxBlobMock, error) {
	parsed, err := SequencerInboxBlobMockMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SequencerInboxBlobMockBin), backend, maxDataSize_, reader_, isUsingFeeToken_, isDelayBufferable_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SequencerInboxBlobMock{SequencerInboxBlobMockCaller: SequencerInboxBlobMockCaller{contract: contract}, SequencerInboxBlobMockTransactor: SequencerInboxBlobMockTransactor{contract: contract}, SequencerInboxBlobMockFilterer: SequencerInboxBlobMockFilterer{contract: contract}}, nil
}

// SequencerInboxBlobMock is an auto generated Go binding around an Ethereum contract.
type SequencerInboxBlobMock struct {
	SequencerInboxBlobMockCaller     // Read-only binding to the contract
	SequencerInboxBlobMockTransactor // Write-only binding to the contract
	SequencerInboxBlobMockFilterer   // Log filterer for contract events
}

// SequencerInboxBlobMockCaller is an auto generated read-only Go binding around an Ethereum contract.
type SequencerInboxBlobMockCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerInboxBlobMockTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SequencerInboxBlobMockTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerInboxBlobMockFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SequencerInboxBlobMockFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerInboxBlobMockSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SequencerInboxBlobMockSession struct {
	Contract     *SequencerInboxBlobMock // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// SequencerInboxBlobMockCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SequencerInboxBlobMockCallerSession struct {
	Contract *SequencerInboxBlobMockCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// SequencerInboxBlobMockTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SequencerInboxBlobMockTransactorSession struct {
	Contract     *SequencerInboxBlobMockTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// SequencerInboxBlobMockRaw is an auto generated low-level Go binding around an Ethereum contract.
type SequencerInboxBlobMockRaw struct {
	Contract *SequencerInboxBlobMock // Generic contract binding to access the raw methods on
}

// SequencerInboxBlobMockCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SequencerInboxBlobMockCallerRaw struct {
	Contract *SequencerInboxBlobMockCaller // Generic read-only contract binding to access the raw methods on
}

// SequencerInboxBlobMockTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SequencerInboxBlobMockTransactorRaw struct {
	Contract *SequencerInboxBlobMockTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSequencerInboxBlobMock creates a new instance of SequencerInboxBlobMock, bound to a specific deployed contract.
func NewSequencerInboxBlobMock(address common.Address, backend bind.ContractBackend) (*SequencerInboxBlobMock, error) {
	contract, err := bindSequencerInboxBlobMock(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMock{SequencerInboxBlobMockCaller: SequencerInboxBlobMockCaller{contract: contract}, SequencerInboxBlobMockTransactor: SequencerInboxBlobMockTransactor{contract: contract}, SequencerInboxBlobMockFilterer: SequencerInboxBlobMockFilterer{contract: contract}}, nil
}

// NewSequencerInboxBlobMockCaller creates a new read-only instance of SequencerInboxBlobMock, bound to a specific deployed contract.
func NewSequencerInboxBlobMockCaller(address common.Address, caller bind.ContractCaller) (*SequencerInboxBlobMockCaller, error) {
	contract, err := bindSequencerInboxBlobMock(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMockCaller{contract: contract}, nil
}

// NewSequencerInboxBlobMockTransactor creates a new write-only instance of SequencerInboxBlobMock, bound to a specific deployed contract.
func NewSequencerInboxBlobMockTransactor(address common.Address, transactor bind.ContractTransactor) (*SequencerInboxBlobMockTransactor, error) {
	contract, err := bindSequencerInboxBlobMock(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMockTransactor{contract: contract}, nil
}

// NewSequencerInboxBlobMockFilterer creates a new log filterer instance of SequencerInboxBlobMock, bound to a specific deployed contract.
func NewSequencerInboxBlobMockFilterer(address common.Address, filterer bind.ContractFilterer) (*SequencerInboxBlobMockFilterer, error) {
	contract, err := bindSequencerInboxBlobMock(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMockFilterer{contract: contract}, nil
}

// bindSequencerInboxBlobMock binds a generic wrapper to an already deployed contract.
func bindSequencerInboxBlobMock(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SequencerInboxBlobMockMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SequencerInboxBlobMock *SequencerInboxBlobMockRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SequencerInboxBlobMock.Contract.SequencerInboxBlobMockCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SequencerInboxBlobMock *SequencerInboxBlobMockRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SequencerInboxBlobMockTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SequencerInboxBlobMock *SequencerInboxBlobMockRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SequencerInboxBlobMockTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SequencerInboxBlobMock.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.contract.Transact(opts, method, params...)
}

// BROTLIMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x16af91a7.
//
// Solidity: function BROTLI_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) BROTLIMESSAGEHEADERFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "BROTLI_MESSAGE_HEADER_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// BROTLIMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x16af91a7.
//
// Solidity: function BROTLI_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) BROTLIMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.BROTLIMESSAGEHEADERFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// BROTLIMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x16af91a7.
//
// Solidity: function BROTLI_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) BROTLIMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.BROTLIMESSAGEHEADERFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// DASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0xf60a5091.
//
// Solidity: function DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) DASMESSAGEHEADERFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "DAS_MESSAGE_HEADER_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// DASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0xf60a5091.
//
// Solidity: function DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) DASMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.DASMESSAGEHEADERFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// DASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0xf60a5091.
//
// Solidity: function DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) DASMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.DASMESSAGEHEADERFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) DATAAUTHENTICATEDFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "DATA_AUTHENTICATED_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) DATAAUTHENTICATEDFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.DATAAUTHENTICATEDFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) DATAAUTHENTICATEDFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.DATAAUTHENTICATEDFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// DATABLOBHEADERFLAG is a free data retrieval call binding the contract method 0x2cbf74e5.
//
// Solidity: function DATA_BLOB_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) DATABLOBHEADERFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "DATA_BLOB_HEADER_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// DATABLOBHEADERFLAG is a free data retrieval call binding the contract method 0x2cbf74e5.
//
// Solidity: function DATA_BLOB_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) DATABLOBHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.DATABLOBHEADERFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// DATABLOBHEADERFLAG is a free data retrieval call binding the contract method 0x2cbf74e5.
//
// Solidity: function DATA_BLOB_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) DATABLOBHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.DATABLOBHEADERFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) HEADERLENGTH(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "HEADER_LENGTH")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) HEADERLENGTH() (*big.Int, error) {
	return _SequencerInboxBlobMock.Contract.HEADERLENGTH(&_SequencerInboxBlobMock.CallOpts)
}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) HEADERLENGTH() (*big.Int, error) {
	return _SequencerInboxBlobMock.Contract.HEADERLENGTH(&_SequencerInboxBlobMock.CallOpts)
}

// TREEDASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x6c890450.
//
// Solidity: function TREE_DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) TREEDASMESSAGEHEADERFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "TREE_DAS_MESSAGE_HEADER_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// TREEDASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x6c890450.
//
// Solidity: function TREE_DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) TREEDASMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.TREEDASMESSAGEHEADERFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// TREEDASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x6c890450.
//
// Solidity: function TREE_DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) TREEDASMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.TREEDASMESSAGEHEADERFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// ZEROHEAVYMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x02c99275.
//
// Solidity: function ZERO_HEAVY_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) ZEROHEAVYMESSAGEHEADERFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "ZERO_HEAVY_MESSAGE_HEADER_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// ZEROHEAVYMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x02c99275.
//
// Solidity: function ZERO_HEAVY_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) ZEROHEAVYMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.ZEROHEAVYMESSAGEHEADERFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// ZEROHEAVYMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x02c99275.
//
// Solidity: function ZERO_HEAVY_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) ZEROHEAVYMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxBlobMock.Contract.ZEROHEAVYMESSAGEHEADERFLAG(&_SequencerInboxBlobMock.CallOpts)
}

// AddSequencerL2BatchFromOrigin is a free data retrieval call binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 , bytes , uint256 , address ) pure returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) AddSequencerL2BatchFromOrigin(opts *bind.CallOpts, arg0 *big.Int, arg1 []byte, arg2 *big.Int, arg3 common.Address) error {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "addSequencerL2BatchFromOrigin", arg0, arg1, arg2, arg3)

	if err != nil {
		return err
	}

	return err

}

// AddSequencerL2BatchFromOrigin is a free data retrieval call binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 , bytes , uint256 , address ) pure returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) AddSequencerL2BatchFromOrigin(arg0 *big.Int, arg1 []byte, arg2 *big.Int, arg3 common.Address) error {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchFromOrigin(&_SequencerInboxBlobMock.CallOpts, arg0, arg1, arg2, arg3)
}

// AddSequencerL2BatchFromOrigin is a free data retrieval call binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 , bytes , uint256 , address ) pure returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) AddSequencerL2BatchFromOrigin(arg0 *big.Int, arg1 []byte, arg2 *big.Int, arg3 common.Address) error {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchFromOrigin(&_SequencerInboxBlobMock.CallOpts, arg0, arg1, arg2, arg3)
}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) BatchCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "batchCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) BatchCount() (*big.Int, error) {
	return _SequencerInboxBlobMock.Contract.BatchCount(&_SequencerInboxBlobMock.CallOpts)
}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) BatchCount() (*big.Int, error) {
	return _SequencerInboxBlobMock.Contract.BatchCount(&_SequencerInboxBlobMock.CallOpts)
}

// BatchPosterManager is a free data retrieval call binding the contract method 0xcc2a1a0c.
//
// Solidity: function batchPosterManager() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) BatchPosterManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "batchPosterManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BatchPosterManager is a free data retrieval call binding the contract method 0xcc2a1a0c.
//
// Solidity: function batchPosterManager() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) BatchPosterManager() (common.Address, error) {
	return _SequencerInboxBlobMock.Contract.BatchPosterManager(&_SequencerInboxBlobMock.CallOpts)
}

// BatchPosterManager is a free data retrieval call binding the contract method 0xcc2a1a0c.
//
// Solidity: function batchPosterManager() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) BatchPosterManager() (common.Address, error) {
	return _SequencerInboxBlobMock.Contract.BatchPosterManager(&_SequencerInboxBlobMock.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) Bridge() (common.Address, error) {
	return _SequencerInboxBlobMock.Contract.Bridge(&_SequencerInboxBlobMock.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) Bridge() (common.Address, error) {
	return _SequencerInboxBlobMock.Contract.Bridge(&_SequencerInboxBlobMock.CallOpts)
}

// Buffer is a free data retrieval call binding the contract method 0xedaafe20.
//
// Solidity: function buffer() view returns(uint64 bufferBlocks, uint64 max, uint64 threshold, uint64 prevBlockNumber, uint64 replenishRateInBasis, uint64 prevSequencedBlockNumber)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) Buffer(opts *bind.CallOpts) (struct {
	BufferBlocks             uint64
	Max                      uint64
	Threshold                uint64
	PrevBlockNumber          uint64
	ReplenishRateInBasis     uint64
	PrevSequencedBlockNumber uint64
}, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "buffer")

	outstruct := new(struct {
		BufferBlocks             uint64
		Max                      uint64
		Threshold                uint64
		PrevBlockNumber          uint64
		ReplenishRateInBasis     uint64
		PrevSequencedBlockNumber uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BufferBlocks = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.Max = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.Threshold = *abi.ConvertType(out[2], new(uint64)).(*uint64)
	outstruct.PrevBlockNumber = *abi.ConvertType(out[3], new(uint64)).(*uint64)
	outstruct.ReplenishRateInBasis = *abi.ConvertType(out[4], new(uint64)).(*uint64)
	outstruct.PrevSequencedBlockNumber = *abi.ConvertType(out[5], new(uint64)).(*uint64)

	return *outstruct, err

}

// Buffer is a free data retrieval call binding the contract method 0xedaafe20.
//
// Solidity: function buffer() view returns(uint64 bufferBlocks, uint64 max, uint64 threshold, uint64 prevBlockNumber, uint64 replenishRateInBasis, uint64 prevSequencedBlockNumber)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) Buffer() (struct {
	BufferBlocks             uint64
	Max                      uint64
	Threshold                uint64
	PrevBlockNumber          uint64
	ReplenishRateInBasis     uint64
	PrevSequencedBlockNumber uint64
}, error) {
	return _SequencerInboxBlobMock.Contract.Buffer(&_SequencerInboxBlobMock.CallOpts)
}

// Buffer is a free data retrieval call binding the contract method 0xedaafe20.
//
// Solidity: function buffer() view returns(uint64 bufferBlocks, uint64 max, uint64 threshold, uint64 prevBlockNumber, uint64 replenishRateInBasis, uint64 prevSequencedBlockNumber)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) Buffer() (struct {
	BufferBlocks             uint64
	Max                      uint64
	Threshold                uint64
	PrevBlockNumber          uint64
	ReplenishRateInBasis     uint64
	PrevSequencedBlockNumber uint64
}, error) {
	return _SequencerInboxBlobMock.Contract.Buffer(&_SequencerInboxBlobMock.CallOpts)
}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool isValidKeyset, uint64 creationBlock)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) DasKeySetInfo(opts *bind.CallOpts, arg0 [32]byte) (struct {
	IsValidKeyset bool
	CreationBlock uint64
}, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "dasKeySetInfo", arg0)

	outstruct := new(struct {
		IsValidKeyset bool
		CreationBlock uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.IsValidKeyset = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.CreationBlock = *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return *outstruct, err

}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool isValidKeyset, uint64 creationBlock)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) DasKeySetInfo(arg0 [32]byte) (struct {
	IsValidKeyset bool
	CreationBlock uint64
}, error) {
	return _SequencerInboxBlobMock.Contract.DasKeySetInfo(&_SequencerInboxBlobMock.CallOpts, arg0)
}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool isValidKeyset, uint64 creationBlock)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) DasKeySetInfo(arg0 [32]byte) (struct {
	IsValidKeyset bool
	CreationBlock uint64
}, error) {
	return _SequencerInboxBlobMock.Contract.DasKeySetInfo(&_SequencerInboxBlobMock.CallOpts, arg0)
}

// ForceInclusionDeadline is a free data retrieval call binding the contract method 0xdd44e6e0.
//
// Solidity: function forceInclusionDeadline(uint64 blockNumber) view returns(uint64)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) ForceInclusionDeadline(opts *bind.CallOpts, blockNumber uint64) (uint64, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "forceInclusionDeadline", blockNumber)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// ForceInclusionDeadline is a free data retrieval call binding the contract method 0xdd44e6e0.
//
// Solidity: function forceInclusionDeadline(uint64 blockNumber) view returns(uint64)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) ForceInclusionDeadline(blockNumber uint64) (uint64, error) {
	return _SequencerInboxBlobMock.Contract.ForceInclusionDeadline(&_SequencerInboxBlobMock.CallOpts, blockNumber)
}

// ForceInclusionDeadline is a free data retrieval call binding the contract method 0xdd44e6e0.
//
// Solidity: function forceInclusionDeadline(uint64 blockNumber) view returns(uint64)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) ForceInclusionDeadline(blockNumber uint64) (uint64, error) {
	return _SequencerInboxBlobMock.Contract.ForceInclusionDeadline(&_SequencerInboxBlobMock.CallOpts, blockNumber)
}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) GetKeysetCreationBlock(opts *bind.CallOpts, ksHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "getKeysetCreationBlock", ksHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) GetKeysetCreationBlock(ksHash [32]byte) (*big.Int, error) {
	return _SequencerInboxBlobMock.Contract.GetKeysetCreationBlock(&_SequencerInboxBlobMock.CallOpts, ksHash)
}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) GetKeysetCreationBlock(ksHash [32]byte) (*big.Int, error) {
	return _SequencerInboxBlobMock.Contract.GetKeysetCreationBlock(&_SequencerInboxBlobMock.CallOpts, ksHash)
}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) InboxAccs(opts *bind.CallOpts, index *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "inboxAccs", index)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) InboxAccs(index *big.Int) ([32]byte, error) {
	return _SequencerInboxBlobMock.Contract.InboxAccs(&_SequencerInboxBlobMock.CallOpts, index)
}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) InboxAccs(index *big.Int) ([32]byte, error) {
	return _SequencerInboxBlobMock.Contract.InboxAccs(&_SequencerInboxBlobMock.CallOpts, index)
}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) IsBatchPoster(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "isBatchPoster", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) IsBatchPoster(arg0 common.Address) (bool, error) {
	return _SequencerInboxBlobMock.Contract.IsBatchPoster(&_SequencerInboxBlobMock.CallOpts, arg0)
}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) IsBatchPoster(arg0 common.Address) (bool, error) {
	return _SequencerInboxBlobMock.Contract.IsBatchPoster(&_SequencerInboxBlobMock.CallOpts, arg0)
}

// IsDelayBufferable is a free data retrieval call binding the contract method 0x4b678a66.
//
// Solidity: function isDelayBufferable() view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) IsDelayBufferable(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "isDelayBufferable")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsDelayBufferable is a free data retrieval call binding the contract method 0x4b678a66.
//
// Solidity: function isDelayBufferable() view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) IsDelayBufferable() (bool, error) {
	return _SequencerInboxBlobMock.Contract.IsDelayBufferable(&_SequencerInboxBlobMock.CallOpts)
}

// IsDelayBufferable is a free data retrieval call binding the contract method 0x4b678a66.
//
// Solidity: function isDelayBufferable() view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) IsDelayBufferable() (bool, error) {
	return _SequencerInboxBlobMock.Contract.IsDelayBufferable(&_SequencerInboxBlobMock.CallOpts)
}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) IsSequencer(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "isSequencer", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) IsSequencer(arg0 common.Address) (bool, error) {
	return _SequencerInboxBlobMock.Contract.IsSequencer(&_SequencerInboxBlobMock.CallOpts, arg0)
}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) IsSequencer(arg0 common.Address) (bool, error) {
	return _SequencerInboxBlobMock.Contract.IsSequencer(&_SequencerInboxBlobMock.CallOpts, arg0)
}

// IsUsingFeeToken is a free data retrieval call binding the contract method 0x92d9f782.
//
// Solidity: function isUsingFeeToken() view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) IsUsingFeeToken(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "isUsingFeeToken")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsUsingFeeToken is a free data retrieval call binding the contract method 0x92d9f782.
//
// Solidity: function isUsingFeeToken() view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) IsUsingFeeToken() (bool, error) {
	return _SequencerInboxBlobMock.Contract.IsUsingFeeToken(&_SequencerInboxBlobMock.CallOpts)
}

// IsUsingFeeToken is a free data retrieval call binding the contract method 0x92d9f782.
//
// Solidity: function isUsingFeeToken() view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) IsUsingFeeToken() (bool, error) {
	return _SequencerInboxBlobMock.Contract.IsUsingFeeToken(&_SequencerInboxBlobMock.CallOpts)
}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) IsValidKeysetHash(opts *bind.CallOpts, ksHash [32]byte) (bool, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "isValidKeysetHash", ksHash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) IsValidKeysetHash(ksHash [32]byte) (bool, error) {
	return _SequencerInboxBlobMock.Contract.IsValidKeysetHash(&_SequencerInboxBlobMock.CallOpts, ksHash)
}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) IsValidKeysetHash(ksHash [32]byte) (bool, error) {
	return _SequencerInboxBlobMock.Contract.IsValidKeysetHash(&_SequencerInboxBlobMock.CallOpts, ksHash)
}

// MaxDataSize is a free data retrieval call binding the contract method 0xe8eb1dc3.
//
// Solidity: function maxDataSize() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) MaxDataSize(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "maxDataSize")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxDataSize is a free data retrieval call binding the contract method 0xe8eb1dc3.
//
// Solidity: function maxDataSize() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) MaxDataSize() (*big.Int, error) {
	return _SequencerInboxBlobMock.Contract.MaxDataSize(&_SequencerInboxBlobMock.CallOpts)
}

// MaxDataSize is a free data retrieval call binding the contract method 0xe8eb1dc3.
//
// Solidity: function maxDataSize() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) MaxDataSize() (*big.Int, error) {
	return _SequencerInboxBlobMock.Contract.MaxDataSize(&_SequencerInboxBlobMock.CallOpts)
}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256, uint256, uint256, uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) MaxTimeVariation(opts *bind.CallOpts) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "maxTimeVariation")

	if err != nil {
		return *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	out3 := *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return out0, out1, out2, out3, err

}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256, uint256, uint256, uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) MaxTimeVariation() (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _SequencerInboxBlobMock.Contract.MaxTimeVariation(&_SequencerInboxBlobMock.CallOpts)
}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256, uint256, uint256, uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) MaxTimeVariation() (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _SequencerInboxBlobMock.Contract.MaxTimeVariation(&_SequencerInboxBlobMock.CallOpts)
}

// Reader4844 is a free data retrieval call binding the contract method 0x8d910dde.
//
// Solidity: function reader4844() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) Reader4844(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "reader4844")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Reader4844 is a free data retrieval call binding the contract method 0x8d910dde.
//
// Solidity: function reader4844() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) Reader4844() (common.Address, error) {
	return _SequencerInboxBlobMock.Contract.Reader4844(&_SequencerInboxBlobMock.CallOpts)
}

// Reader4844 is a free data retrieval call binding the contract method 0x8d910dde.
//
// Solidity: function reader4844() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) Reader4844() (common.Address, error) {
	return _SequencerInboxBlobMock.Contract.Reader4844(&_SequencerInboxBlobMock.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) Rollup() (common.Address, error) {
	return _SequencerInboxBlobMock.Contract.Rollup(&_SequencerInboxBlobMock.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) Rollup() (common.Address, error) {
	return _SequencerInboxBlobMock.Contract.Rollup(&_SequencerInboxBlobMock.CallOpts)
}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCaller) TotalDelayedMessagesRead(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInboxBlobMock.contract.Call(opts, &out, "totalDelayedMessagesRead")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) TotalDelayedMessagesRead() (*big.Int, error) {
	return _SequencerInboxBlobMock.Contract.TotalDelayedMessagesRead(&_SequencerInboxBlobMock.CallOpts)
}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockCallerSession) TotalDelayedMessagesRead() (*big.Int, error) {
	return _SequencerInboxBlobMock.Contract.TotalDelayedMessagesRead(&_SequencerInboxBlobMock.CallOpts)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) AddSequencerL2Batch(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "addSequencerL2Batch", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) AddSequencerL2Batch(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2Batch(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) AddSequencerL2Batch(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2Batch(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchDelayProof is a paid mutator transaction binding the contract method 0x6e620055.
//
// Solidity: function addSequencerL2BatchDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) AddSequencerL2BatchDelayProof(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "addSequencerL2BatchDelayProof", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchDelayProof is a paid mutator transaction binding the contract method 0x6e620055.
//
// Solidity: function addSequencerL2BatchDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) AddSequencerL2BatchDelayProof(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchDelayProof(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchDelayProof is a paid mutator transaction binding the contract method 0x6e620055.
//
// Solidity: function addSequencerL2BatchDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) AddSequencerL2BatchDelayProof(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchDelayProof(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromBlobs is a paid mutator transaction binding the contract method 0x3e5aa082.
//
// Solidity: function addSequencerL2BatchFromBlobs(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) AddSequencerL2BatchFromBlobs(opts *bind.TransactOpts, sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "addSequencerL2BatchFromBlobs", sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromBlobs is a paid mutator transaction binding the contract method 0x3e5aa082.
//
// Solidity: function addSequencerL2BatchFromBlobs(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) AddSequencerL2BatchFromBlobs(sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchFromBlobs(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromBlobs is a paid mutator transaction binding the contract method 0x3e5aa082.
//
// Solidity: function addSequencerL2BatchFromBlobs(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) AddSequencerL2BatchFromBlobs(sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchFromBlobs(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromBlobsDelayProof is a paid mutator transaction binding the contract method 0x917cf8ac.
//
// Solidity: function addSequencerL2BatchFromBlobsDelayProof(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) AddSequencerL2BatchFromBlobsDelayProof(opts *bind.TransactOpts, sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "addSequencerL2BatchFromBlobsDelayProof", sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromBlobsDelayProof is a paid mutator transaction binding the contract method 0x917cf8ac.
//
// Solidity: function addSequencerL2BatchFromBlobsDelayProof(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) AddSequencerL2BatchFromBlobsDelayProof(sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchFromBlobsDelayProof(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromBlobsDelayProof is a paid mutator transaction binding the contract method 0x917cf8ac.
//
// Solidity: function addSequencerL2BatchFromBlobsDelayProof(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) AddSequencerL2BatchFromBlobsDelayProof(sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchFromBlobsDelayProof(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromOrigin0 is a paid mutator transaction binding the contract method 0x8f111f3c.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) AddSequencerL2BatchFromOrigin0(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "addSequencerL2BatchFromOrigin0", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromOrigin0 is a paid mutator transaction binding the contract method 0x8f111f3c.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) AddSequencerL2BatchFromOrigin0(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchFromOrigin0(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromOrigin0 is a paid mutator transaction binding the contract method 0x8f111f3c.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) AddSequencerL2BatchFromOrigin0(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchFromOrigin0(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromOriginDelayProof is a paid mutator transaction binding the contract method 0x69cacded.
//
// Solidity: function addSequencerL2BatchFromOriginDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) AddSequencerL2BatchFromOriginDelayProof(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "addSequencerL2BatchFromOriginDelayProof", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromOriginDelayProof is a paid mutator transaction binding the contract method 0x69cacded.
//
// Solidity: function addSequencerL2BatchFromOriginDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) AddSequencerL2BatchFromOriginDelayProof(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchFromOriginDelayProof(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromOriginDelayProof is a paid mutator transaction binding the contract method 0x69cacded.
//
// Solidity: function addSequencerL2BatchFromOriginDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) AddSequencerL2BatchFromOriginDelayProof(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.AddSequencerL2BatchFromOriginDelayProof(&_SequencerInboxBlobMock.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) ForceInclusion(opts *bind.TransactOpts, _totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "forceInclusion", _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) ForceInclusion(_totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.ForceInclusion(&_SequencerInboxBlobMock.TransactOpts, _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) ForceInclusion(_totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.ForceInclusion(&_SequencerInboxBlobMock.TransactOpts, _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// Initialize is a paid mutator transaction binding the contract method 0x1ad87e45.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_, (uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) Initialize(opts *bind.TransactOpts, bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation, bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "initialize", bridge_, maxTimeVariation_, bufferConfig_)
}

// Initialize is a paid mutator transaction binding the contract method 0x1ad87e45.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_, (uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) Initialize(bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation, bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.Initialize(&_SequencerInboxBlobMock.TransactOpts, bridge_, maxTimeVariation_, bufferConfig_)
}

// Initialize is a paid mutator transaction binding the contract method 0x1ad87e45.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_, (uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) Initialize(bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation, bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.Initialize(&_SequencerInboxBlobMock.TransactOpts, bridge_, maxTimeVariation_, bufferConfig_)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) InvalidateKeysetHash(opts *bind.TransactOpts, ksHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "invalidateKeysetHash", ksHash)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) InvalidateKeysetHash(ksHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.InvalidateKeysetHash(&_SequencerInboxBlobMock.TransactOpts, ksHash)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) InvalidateKeysetHash(ksHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.InvalidateKeysetHash(&_SequencerInboxBlobMock.TransactOpts, ksHash)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xa655d937.
//
// Solidity: function postUpgradeInit((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) PostUpgradeInit(opts *bind.TransactOpts, bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "postUpgradeInit", bufferConfig_)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xa655d937.
//
// Solidity: function postUpgradeInit((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) PostUpgradeInit(bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.PostUpgradeInit(&_SequencerInboxBlobMock.TransactOpts, bufferConfig_)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xa655d937.
//
// Solidity: function postUpgradeInit((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) PostUpgradeInit(bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.PostUpgradeInit(&_SequencerInboxBlobMock.TransactOpts, bufferConfig_)
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) RemoveDelayAfterFork(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "removeDelayAfterFork")
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) RemoveDelayAfterFork() (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.RemoveDelayAfterFork(&_SequencerInboxBlobMock.TransactOpts)
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) RemoveDelayAfterFork() (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.RemoveDelayAfterFork(&_SequencerInboxBlobMock.TransactOpts)
}

// SetBatchPosterManager is a paid mutator transaction binding the contract method 0x1ff64790.
//
// Solidity: function setBatchPosterManager(address newBatchPosterManager) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) SetBatchPosterManager(opts *bind.TransactOpts, newBatchPosterManager common.Address) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "setBatchPosterManager", newBatchPosterManager)
}

// SetBatchPosterManager is a paid mutator transaction binding the contract method 0x1ff64790.
//
// Solidity: function setBatchPosterManager(address newBatchPosterManager) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) SetBatchPosterManager(newBatchPosterManager common.Address) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetBatchPosterManager(&_SequencerInboxBlobMock.TransactOpts, newBatchPosterManager)
}

// SetBatchPosterManager is a paid mutator transaction binding the contract method 0x1ff64790.
//
// Solidity: function setBatchPosterManager(address newBatchPosterManager) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) SetBatchPosterManager(newBatchPosterManager common.Address) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetBatchPosterManager(&_SequencerInboxBlobMock.TransactOpts, newBatchPosterManager)
}

// SetBufferConfig is a paid mutator transaction binding the contract method 0x2f3985a7.
//
// Solidity: function setBufferConfig((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) SetBufferConfig(opts *bind.TransactOpts, bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "setBufferConfig", bufferConfig_)
}

// SetBufferConfig is a paid mutator transaction binding the contract method 0x2f3985a7.
//
// Solidity: function setBufferConfig((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) SetBufferConfig(bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetBufferConfig(&_SequencerInboxBlobMock.TransactOpts, bufferConfig_)
}

// SetBufferConfig is a paid mutator transaction binding the contract method 0x2f3985a7.
//
// Solidity: function setBufferConfig((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) SetBufferConfig(bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetBufferConfig(&_SequencerInboxBlobMock.TransactOpts, bufferConfig_)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) SetIsBatchPoster(opts *bind.TransactOpts, addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "setIsBatchPoster", addr, isBatchPoster_)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) SetIsBatchPoster(addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetIsBatchPoster(&_SequencerInboxBlobMock.TransactOpts, addr, isBatchPoster_)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) SetIsBatchPoster(addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetIsBatchPoster(&_SequencerInboxBlobMock.TransactOpts, addr, isBatchPoster_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) SetIsSequencer(opts *bind.TransactOpts, addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "setIsSequencer", addr, isSequencer_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) SetIsSequencer(addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetIsSequencer(&_SequencerInboxBlobMock.TransactOpts, addr, isSequencer_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) SetIsSequencer(addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetIsSequencer(&_SequencerInboxBlobMock.TransactOpts, addr, isSequencer_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) SetMaxTimeVariation(opts *bind.TransactOpts, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "setMaxTimeVariation", maxTimeVariation_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) SetMaxTimeVariation(maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetMaxTimeVariation(&_SequencerInboxBlobMock.TransactOpts, maxTimeVariation_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) SetMaxTimeVariation(maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetMaxTimeVariation(&_SequencerInboxBlobMock.TransactOpts, maxTimeVariation_)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) SetValidKeyset(opts *bind.TransactOpts, keysetBytes []byte) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "setValidKeyset", keysetBytes)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) SetValidKeyset(keysetBytes []byte) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetValidKeyset(&_SequencerInboxBlobMock.TransactOpts, keysetBytes)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) SetValidKeyset(keysetBytes []byte) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.SetValidKeyset(&_SequencerInboxBlobMock.TransactOpts, keysetBytes)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactor) UpdateRollupAddress(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInboxBlobMock.contract.Transact(opts, "updateRollupAddress")
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.UpdateRollupAddress(&_SequencerInboxBlobMock.TransactOpts)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_SequencerInboxBlobMock *SequencerInboxBlobMockTransactorSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _SequencerInboxBlobMock.Contract.UpdateRollupAddress(&_SequencerInboxBlobMock.TransactOpts)
}

// SequencerInboxBlobMockInboxMessageDeliveredIterator is returned from FilterInboxMessageDelivered and is used to iterate over the raw logs and unpacked data for InboxMessageDelivered events raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockInboxMessageDeliveredIterator struct {
	Event *SequencerInboxBlobMockInboxMessageDelivered // Event containing the contract specifics and raw log

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
func (it *SequencerInboxBlobMockInboxMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxBlobMockInboxMessageDelivered)
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
		it.Event = new(SequencerInboxBlobMockInboxMessageDelivered)
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
func (it *SequencerInboxBlobMockInboxMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxBlobMockInboxMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxBlobMockInboxMessageDelivered represents a InboxMessageDelivered event raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockInboxMessageDelivered struct {
	MessageNum *big.Int
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDelivered is a free log retrieval operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) FilterInboxMessageDelivered(opts *bind.FilterOpts, messageNum []*big.Int) (*SequencerInboxBlobMockInboxMessageDeliveredIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.FilterLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMockInboxMessageDeliveredIterator{contract: _SequencerInboxBlobMock.contract, event: "InboxMessageDelivered", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDelivered is a free log subscription operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) WatchInboxMessageDelivered(opts *bind.WatchOpts, sink chan<- *SequencerInboxBlobMockInboxMessageDelivered, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.WatchLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxBlobMockInboxMessageDelivered)
				if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
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

// ParseInboxMessageDelivered is a log parse operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) ParseInboxMessageDelivered(log types.Log) (*SequencerInboxBlobMockInboxMessageDelivered, error) {
	event := new(SequencerInboxBlobMockInboxMessageDelivered)
	if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxBlobMockInboxMessageDeliveredFromOriginIterator is returned from FilterInboxMessageDeliveredFromOrigin and is used to iterate over the raw logs and unpacked data for InboxMessageDeliveredFromOrigin events raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockInboxMessageDeliveredFromOriginIterator struct {
	Event *SequencerInboxBlobMockInboxMessageDeliveredFromOrigin // Event containing the contract specifics and raw log

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
func (it *SequencerInboxBlobMockInboxMessageDeliveredFromOriginIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxBlobMockInboxMessageDeliveredFromOrigin)
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
		it.Event = new(SequencerInboxBlobMockInboxMessageDeliveredFromOrigin)
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
func (it *SequencerInboxBlobMockInboxMessageDeliveredFromOriginIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxBlobMockInboxMessageDeliveredFromOriginIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxBlobMockInboxMessageDeliveredFromOrigin represents a InboxMessageDeliveredFromOrigin event raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockInboxMessageDeliveredFromOrigin struct {
	MessageNum *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDeliveredFromOrigin is a free log retrieval operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) FilterInboxMessageDeliveredFromOrigin(opts *bind.FilterOpts, messageNum []*big.Int) (*SequencerInboxBlobMockInboxMessageDeliveredFromOriginIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.FilterLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMockInboxMessageDeliveredFromOriginIterator{contract: _SequencerInboxBlobMock.contract, event: "InboxMessageDeliveredFromOrigin", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDeliveredFromOrigin is a free log subscription operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) WatchInboxMessageDeliveredFromOrigin(opts *bind.WatchOpts, sink chan<- *SequencerInboxBlobMockInboxMessageDeliveredFromOrigin, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.WatchLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxBlobMockInboxMessageDeliveredFromOrigin)
				if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
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

// ParseInboxMessageDeliveredFromOrigin is a log parse operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) ParseInboxMessageDeliveredFromOrigin(log types.Log) (*SequencerInboxBlobMockInboxMessageDeliveredFromOrigin, error) {
	event := new(SequencerInboxBlobMockInboxMessageDeliveredFromOrigin)
	if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxBlobMockInvalidateKeysetIterator is returned from FilterInvalidateKeyset and is used to iterate over the raw logs and unpacked data for InvalidateKeyset events raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockInvalidateKeysetIterator struct {
	Event *SequencerInboxBlobMockInvalidateKeyset // Event containing the contract specifics and raw log

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
func (it *SequencerInboxBlobMockInvalidateKeysetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxBlobMockInvalidateKeyset)
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
		it.Event = new(SequencerInboxBlobMockInvalidateKeyset)
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
func (it *SequencerInboxBlobMockInvalidateKeysetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxBlobMockInvalidateKeysetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxBlobMockInvalidateKeyset represents a InvalidateKeyset event raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockInvalidateKeyset struct {
	KeysetHash [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInvalidateKeyset is a free log retrieval operation binding the contract event 0x5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a.
//
// Solidity: event InvalidateKeyset(bytes32 indexed keysetHash)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) FilterInvalidateKeyset(opts *bind.FilterOpts, keysetHash [][32]byte) (*SequencerInboxBlobMockInvalidateKeysetIterator, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.FilterLogs(opts, "InvalidateKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMockInvalidateKeysetIterator{contract: _SequencerInboxBlobMock.contract, event: "InvalidateKeyset", logs: logs, sub: sub}, nil
}

// WatchInvalidateKeyset is a free log subscription operation binding the contract event 0x5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a.
//
// Solidity: event InvalidateKeyset(bytes32 indexed keysetHash)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) WatchInvalidateKeyset(opts *bind.WatchOpts, sink chan<- *SequencerInboxBlobMockInvalidateKeyset, keysetHash [][32]byte) (event.Subscription, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.WatchLogs(opts, "InvalidateKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxBlobMockInvalidateKeyset)
				if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "InvalidateKeyset", log); err != nil {
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

// ParseInvalidateKeyset is a log parse operation binding the contract event 0x5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a.
//
// Solidity: event InvalidateKeyset(bytes32 indexed keysetHash)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) ParseInvalidateKeyset(log types.Log) (*SequencerInboxBlobMockInvalidateKeyset, error) {
	event := new(SequencerInboxBlobMockInvalidateKeyset)
	if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "InvalidateKeyset", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxBlobMockOwnerFunctionCalledIterator is returned from FilterOwnerFunctionCalled and is used to iterate over the raw logs and unpacked data for OwnerFunctionCalled events raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockOwnerFunctionCalledIterator struct {
	Event *SequencerInboxBlobMockOwnerFunctionCalled // Event containing the contract specifics and raw log

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
func (it *SequencerInboxBlobMockOwnerFunctionCalledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxBlobMockOwnerFunctionCalled)
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
		it.Event = new(SequencerInboxBlobMockOwnerFunctionCalled)
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
func (it *SequencerInboxBlobMockOwnerFunctionCalledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxBlobMockOwnerFunctionCalledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxBlobMockOwnerFunctionCalled represents a OwnerFunctionCalled event raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockOwnerFunctionCalled struct {
	Id  *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterOwnerFunctionCalled is a free log retrieval operation binding the contract event 0xea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e.
//
// Solidity: event OwnerFunctionCalled(uint256 indexed id)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) FilterOwnerFunctionCalled(opts *bind.FilterOpts, id []*big.Int) (*SequencerInboxBlobMockOwnerFunctionCalledIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.FilterLogs(opts, "OwnerFunctionCalled", idRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMockOwnerFunctionCalledIterator{contract: _SequencerInboxBlobMock.contract, event: "OwnerFunctionCalled", logs: logs, sub: sub}, nil
}

// WatchOwnerFunctionCalled is a free log subscription operation binding the contract event 0xea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e.
//
// Solidity: event OwnerFunctionCalled(uint256 indexed id)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) WatchOwnerFunctionCalled(opts *bind.WatchOpts, sink chan<- *SequencerInboxBlobMockOwnerFunctionCalled, id []*big.Int) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.WatchLogs(opts, "OwnerFunctionCalled", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxBlobMockOwnerFunctionCalled)
				if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "OwnerFunctionCalled", log); err != nil {
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

// ParseOwnerFunctionCalled is a log parse operation binding the contract event 0xea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e.
//
// Solidity: event OwnerFunctionCalled(uint256 indexed id)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) ParseOwnerFunctionCalled(log types.Log) (*SequencerInboxBlobMockOwnerFunctionCalled, error) {
	event := new(SequencerInboxBlobMockOwnerFunctionCalled)
	if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "OwnerFunctionCalled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxBlobMockSequencerBatchDataIterator is returned from FilterSequencerBatchData and is used to iterate over the raw logs and unpacked data for SequencerBatchData events raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockSequencerBatchDataIterator struct {
	Event *SequencerInboxBlobMockSequencerBatchData // Event containing the contract specifics and raw log

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
func (it *SequencerInboxBlobMockSequencerBatchDataIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxBlobMockSequencerBatchData)
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
		it.Event = new(SequencerInboxBlobMockSequencerBatchData)
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
func (it *SequencerInboxBlobMockSequencerBatchDataIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxBlobMockSequencerBatchDataIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxBlobMockSequencerBatchData represents a SequencerBatchData event raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockSequencerBatchData struct {
	BatchSequenceNumber *big.Int
	Data                []byte
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterSequencerBatchData is a free log retrieval operation binding the contract event 0xfe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c20.
//
// Solidity: event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) FilterSequencerBatchData(opts *bind.FilterOpts, batchSequenceNumber []*big.Int) (*SequencerInboxBlobMockSequencerBatchDataIterator, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.FilterLogs(opts, "SequencerBatchData", batchSequenceNumberRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMockSequencerBatchDataIterator{contract: _SequencerInboxBlobMock.contract, event: "SequencerBatchData", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchData is a free log subscription operation binding the contract event 0xfe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c20.
//
// Solidity: event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) WatchSequencerBatchData(opts *bind.WatchOpts, sink chan<- *SequencerInboxBlobMockSequencerBatchData, batchSequenceNumber []*big.Int) (event.Subscription, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.WatchLogs(opts, "SequencerBatchData", batchSequenceNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxBlobMockSequencerBatchData)
				if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "SequencerBatchData", log); err != nil {
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

// ParseSequencerBatchData is a log parse operation binding the contract event 0xfe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c20.
//
// Solidity: event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) ParseSequencerBatchData(log types.Log) (*SequencerInboxBlobMockSequencerBatchData, error) {
	event := new(SequencerInboxBlobMockSequencerBatchData)
	if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "SequencerBatchData", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxBlobMockSequencerBatchDeliveredIterator is returned from FilterSequencerBatchDelivered and is used to iterate over the raw logs and unpacked data for SequencerBatchDelivered events raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockSequencerBatchDeliveredIterator struct {
	Event *SequencerInboxBlobMockSequencerBatchDelivered // Event containing the contract specifics and raw log

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
func (it *SequencerInboxBlobMockSequencerBatchDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxBlobMockSequencerBatchDelivered)
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
		it.Event = new(SequencerInboxBlobMockSequencerBatchDelivered)
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
func (it *SequencerInboxBlobMockSequencerBatchDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxBlobMockSequencerBatchDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxBlobMockSequencerBatchDelivered represents a SequencerBatchDelivered event raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockSequencerBatchDelivered struct {
	BatchSequenceNumber      *big.Int
	BeforeAcc                [32]byte
	AfterAcc                 [32]byte
	DelayedAcc               [32]byte
	AfterDelayedMessagesRead *big.Int
	TimeBounds               IBridgeTimeBounds
	DataLocation             uint8
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterSequencerBatchDelivered is a free log retrieval operation binding the contract event 0x7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7.
//
// Solidity: event SequencerBatchDelivered(uint256 indexed batchSequenceNumber, bytes32 indexed beforeAcc, bytes32 indexed afterAcc, bytes32 delayedAcc, uint256 afterDelayedMessagesRead, (uint64,uint64,uint64,uint64) timeBounds, uint8 dataLocation)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) FilterSequencerBatchDelivered(opts *bind.FilterOpts, batchSequenceNumber []*big.Int, beforeAcc [][32]byte, afterAcc [][32]byte) (*SequencerInboxBlobMockSequencerBatchDeliveredIterator, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}
	var beforeAccRule []interface{}
	for _, beforeAccItem := range beforeAcc {
		beforeAccRule = append(beforeAccRule, beforeAccItem)
	}
	var afterAccRule []interface{}
	for _, afterAccItem := range afterAcc {
		afterAccRule = append(afterAccRule, afterAccItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.FilterLogs(opts, "SequencerBatchDelivered", batchSequenceNumberRule, beforeAccRule, afterAccRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMockSequencerBatchDeliveredIterator{contract: _SequencerInboxBlobMock.contract, event: "SequencerBatchDelivered", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchDelivered is a free log subscription operation binding the contract event 0x7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7.
//
// Solidity: event SequencerBatchDelivered(uint256 indexed batchSequenceNumber, bytes32 indexed beforeAcc, bytes32 indexed afterAcc, bytes32 delayedAcc, uint256 afterDelayedMessagesRead, (uint64,uint64,uint64,uint64) timeBounds, uint8 dataLocation)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) WatchSequencerBatchDelivered(opts *bind.WatchOpts, sink chan<- *SequencerInboxBlobMockSequencerBatchDelivered, batchSequenceNumber []*big.Int, beforeAcc [][32]byte, afterAcc [][32]byte) (event.Subscription, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}
	var beforeAccRule []interface{}
	for _, beforeAccItem := range beforeAcc {
		beforeAccRule = append(beforeAccRule, beforeAccItem)
	}
	var afterAccRule []interface{}
	for _, afterAccItem := range afterAcc {
		afterAccRule = append(afterAccRule, afterAccItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.WatchLogs(opts, "SequencerBatchDelivered", batchSequenceNumberRule, beforeAccRule, afterAccRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxBlobMockSequencerBatchDelivered)
				if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "SequencerBatchDelivered", log); err != nil {
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

// ParseSequencerBatchDelivered is a log parse operation binding the contract event 0x7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7.
//
// Solidity: event SequencerBatchDelivered(uint256 indexed batchSequenceNumber, bytes32 indexed beforeAcc, bytes32 indexed afterAcc, bytes32 delayedAcc, uint256 afterDelayedMessagesRead, (uint64,uint64,uint64,uint64) timeBounds, uint8 dataLocation)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) ParseSequencerBatchDelivered(log types.Log) (*SequencerInboxBlobMockSequencerBatchDelivered, error) {
	event := new(SequencerInboxBlobMockSequencerBatchDelivered)
	if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "SequencerBatchDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxBlobMockSetValidKeysetIterator is returned from FilterSetValidKeyset and is used to iterate over the raw logs and unpacked data for SetValidKeyset events raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockSetValidKeysetIterator struct {
	Event *SequencerInboxBlobMockSetValidKeyset // Event containing the contract specifics and raw log

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
func (it *SequencerInboxBlobMockSetValidKeysetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxBlobMockSetValidKeyset)
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
		it.Event = new(SequencerInboxBlobMockSetValidKeyset)
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
func (it *SequencerInboxBlobMockSetValidKeysetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxBlobMockSetValidKeysetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxBlobMockSetValidKeyset represents a SetValidKeyset event raised by the SequencerInboxBlobMock contract.
type SequencerInboxBlobMockSetValidKeyset struct {
	KeysetHash  [32]byte
	KeysetBytes []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSetValidKeyset is a free log retrieval operation binding the contract event 0xabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722.
//
// Solidity: event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) FilterSetValidKeyset(opts *bind.FilterOpts, keysetHash [][32]byte) (*SequencerInboxBlobMockSetValidKeysetIterator, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.FilterLogs(opts, "SetValidKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxBlobMockSetValidKeysetIterator{contract: _SequencerInboxBlobMock.contract, event: "SetValidKeyset", logs: logs, sub: sub}, nil
}

// WatchSetValidKeyset is a free log subscription operation binding the contract event 0xabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722.
//
// Solidity: event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) WatchSetValidKeyset(opts *bind.WatchOpts, sink chan<- *SequencerInboxBlobMockSetValidKeyset, keysetHash [][32]byte) (event.Subscription, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInboxBlobMock.contract.WatchLogs(opts, "SetValidKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxBlobMockSetValidKeyset)
				if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "SetValidKeyset", log); err != nil {
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

// ParseSetValidKeyset is a log parse operation binding the contract event 0xabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722.
//
// Solidity: event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes)
func (_SequencerInboxBlobMock *SequencerInboxBlobMockFilterer) ParseSetValidKeyset(log types.Log) (*SequencerInboxBlobMockSetValidKeyset, error) {
	event := new(SequencerInboxBlobMockSetValidKeyset)
	if err := _SequencerInboxBlobMock.contract.UnpackLog(event, "SetValidKeyset", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxStubMetaData contains all meta data concerning the SequencerInboxStub contract.
var SequencerInboxStubMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"bridge_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"sequencer_\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"maxDataSize_\",\"type\":\"uint256\"},{\"internalType\":\"contractIReader4844\",\"name\":\"reader4844_\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isUsingFeeToken_\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"isDelayBufferable_\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AlreadyInit\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"AlreadyValidDASKeyset\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BadBufferConfig\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BadMaxTimeVariation\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"stored\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"received\",\"type\":\"uint256\"}],\"name\":\"BadSequencerNumber\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DataBlobsNotSupported\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"dataLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxDataLength\",\"type\":\"uint256\"}],\"name\":\"DataTooLarge\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelayProofRequired\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelayedBackwards\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelayedTooFar\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Deprecated\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ExtraGasNotUint64\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ForceIncludeBlockTooSoon\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HadZeroInit\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"IncorrectMessagePreimage\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"}],\"name\":\"InitParamZero\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidDelayedAccPreimage\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"name\":\"InvalidHeaderFlag\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"KeysetTooLarge\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MissingDataHashes\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NativeTokenMismatch\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"NoSuchKeyset\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotBatchPoster\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"NotBatchPosterManager\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotDelayBufferable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotForked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotOrigin\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotOwner\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RollupNotChanged\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keysetHash\",\"type\":\"bytes32\"}],\"name\":\"InvalidateKeyset\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"OwnerFunctionCalled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchSequenceNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"SequencerBatchData\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchSequenceNumber\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"afterAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"minTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"minBlockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxBlockNumber\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structIBridge.TimeBounds\",\"name\":\"timeBounds\",\"type\":\"tuple\"},{\"indexed\":false,\"internalType\":\"enumIBridge.BatchDataLocation\",\"name\":\"dataLocation\",\"type\":\"uint8\"}],\"name\":\"SequencerBatchDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keysetHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"SetValidKeyset\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"BROTLI_MESSAGE_HEADER_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DAS_MESSAGE_HEADER_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DATA_AUTHENTICATED_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DATA_BLOB_HEADER_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"HEADER_LENGTH\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"TREE_DAS_MESSAGE_HEADER_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ZERO_HEAVY_MESSAGE_HEADER_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"name\":\"addInitMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2Batch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"beforeDelayedAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"blockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"inboxSeqNum\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"internalType\":\"structMessages.Message\",\"name\":\"delayedMessage\",\"type\":\"tuple\"}],\"internalType\":\"structDelayProof\",\"name\":\"delayProof\",\"type\":\"tuple\"}],\"name\":\"addSequencerL2BatchDelayProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2BatchFromBlobs\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"beforeDelayedAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"blockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"inboxSeqNum\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"internalType\":\"structMessages.Message\",\"name\":\"delayedMessage\",\"type\":\"tuple\"}],\"internalType\":\"structDelayProof\",\"name\":\"delayProof\",\"type\":\"tuple\"}],\"name\":\"addSequencerL2BatchFromBlobsDelayProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"addSequencerL2BatchFromOrigin\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2BatchFromOrigin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"beforeDelayedAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"blockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"inboxSeqNum\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"internalType\":\"structMessages.Message\",\"name\":\"delayedMessage\",\"type\":\"tuple\"}],\"internalType\":\"structDelayProof\",\"name\":\"delayProof\",\"type\":\"tuple\"}],\"name\":\"addSequencerL2BatchFromOriginDelayProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batchCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batchPosterManager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"buffer\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"bufferBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"max\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"threshold\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"prevBlockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"replenishRateInBasis\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"prevSequencedBlockNumber\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"dasKeySetInfo\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"isValidKeyset\",\"type\":\"bool\"},{\"internalType\":\"uint64\",\"name\":\"creationBlock\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"uint64[2]\",\"name\":\"l1BlockAndTime\",\"type\":\"uint64[2]\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"forceInclusion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"blockNumber\",\"type\":\"uint64\"}],\"name\":\"forceInclusionDeadline\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"getKeysetCreationBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"inboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"bridge_\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"threshold\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"max\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"replenishRateInBasis\",\"type\":\"uint64\"}],\"internalType\":\"structBufferConfig\",\"name\":\"bufferConfig_\",\"type\":\"tuple\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"invalidateKeysetHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isBatchPoster\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isDelayBufferable\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isSequencer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isUsingFeeToken\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"isValidKeysetHash\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxDataSize\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxTimeVariation\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint64\",\"name\":\"threshold\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"max\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"replenishRateInBasis\",\"type\":\"uint64\"}],\"internalType\":\"structBufferConfig\",\"name\":\"bufferConfig_\",\"type\":\"tuple\"}],\"name\":\"postUpgradeInit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"reader4844\",\"outputs\":[{\"internalType\":\"contractIReader4844\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"removeDelayAfterFork\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newBatchPosterManager\",\"type\":\"address\"}],\"name\":\"setBatchPosterManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint64\",\"name\":\"threshold\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"max\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"replenishRateInBasis\",\"type\":\"uint64\"}],\"internalType\":\"structBufferConfig\",\"name\":\"bufferConfig_\",\"type\":\"tuple\"}],\"name\":\"setBufferConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isBatchPoster_\",\"type\":\"bool\"}],\"name\":\"setIsBatchPoster\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isSequencer_\",\"type\":\"bool\"}],\"name\":\"setIsSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"}],\"name\":\"setMaxTimeVariation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"setValidKeyset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalDelayedMessagesRead\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x610180604052306080526202000060a05246610100526200002b620001c8602090811b62002f7e17901c565b1515610120523480156200003e57600080fd5b5060405162005644380380620056448339810160408190526200006191620002a1565b838383838360e081815250506101205115620000a6576001600160a01b03831615620000a0576040516386657a5360e01b815260040160405180910390fd5b620000ef565b6001600160a01b038316620000ef576040516380fc2c0360e01b815260206004820152600a60248201526914995859195c8d0e0d0d60b21b604482015260640160405180910390fd5b6001600160a01b0392831660c05290151561014052151561016052600180549982166001600160a01b03199a8b1617815560028054909a1633179099558651600a80546020808b01516040808d01516060909d01516001600160401b03908116600160c01b026001600160c01b039e8216600160801b029e909e166001600160801b0393821668010000000000000000026001600160801b0319909616919097161793909317169390931799909917905597166000908152600390975250505091909220805460ff191690931790925550620003ca9050565b60408051600481526024810182526020810180516001600160e01b03166302881c7960e11b17905290516000918291829160649162000208919062000399565b600060405180830381855afa9150503d806000811462000245576040519150601f19603f3d011682016040523d82523d6000602084013e6200024a565b606091505b50915091508180156200025e575080516020145b9250505090565b6001600160a01b03811681146200027b57600080fd5b50565b80516200028b8162000265565b919050565b805180151581146200028b57600080fd5b6000806000806000806000878903610140811215620002bf57600080fd5b8851620002cc8162000265565b60208a0151909850620002df8162000265565b96506080603f1982011215620002f457600080fd5b50604051608081016001600160401b03811182821017156200032657634e487b7160e01b600052604160045260246000fd5b806040525060408901518152606089015160208201526080890151604082015260a089015160608201528095505060c088015193506200036960e089016200027e565b92506200037a610100890162000290565b91506200038b610120890162000290565b905092959891949750929550565b6000825160005b81811015620003bc5760208186018101518583015201620003a0565b506000920191825250919050565b60805160a05160c05160e05161010051610120516101405161016051615165620004df6000396000818161044201528181610b8a0152818161157601528181611a65015281816120f80152818161254001528181612ac901528181612c5e0152818161317201526133b401526000818161061201528181610a490152818161352201526137430152600081816128aa015281816134c00152613ea80152600081816123b50152613a4101526000818161072c0152818161424c01526142a10152600081816105ad01528181611001015281816120a101528181613c9e0152613d790152600081816111e10152818161173201528181611f9601526122ab0152600081816108b1015261243401526151656000f3fe608060405234801561001057600080fd5b50600436106102ff5760003560e01c806371c3e6fe1161019c578063cc2a1a0c116100ee578063e78cea9211610097578063edaafe2011610071578063edaafe2014610776578063f1981578146107ff578063f60a50911461081257600080fd5b8063e78cea9214610714578063e8eb1dc314610727578063ebea461d1461074e57600080fd5b8063dd44e6e0116100c8578063dd44e6e0146106ae578063e0bc9729146106da578063e5a358c8146106ed57600080fd5b8063cc2a1a0c14610675578063d1ce8da814610688578063d9dd67ab1461069b57600080fd5b8063917cf8ac11610150578063a655d9371161012a578063a655d9371461063c578063b31761f81461064f578063cb23bcb51461066257600080fd5b8063917cf8ac146105fa57806392d9f7821461060d57806396cc5c781461063457600080fd5b8063844208601161018157806384420860146105955780638d910dde146105a85780638f111f3c146105e757600080fd5b806371c3e6fe146105695780637fa3a40e1461058c57600080fd5b80633e5aa082116102555780636c890450116102095780636e7df3e7116101e35780636e7df3e7146104ef5780636f12b0c914610502578063715ea34b1461051557600080fd5b80636c890450146104925780636d46e987146104b95780636e620055146104dc57600080fd5b80636633ae851161023a5780636633ae851461046457806369cacded146104775780636ae71f121461048a57600080fd5b80633e5aa0821461042a5780634b678a661461043d57600080fd5b80631f956632116102b757806327957a491161029157806327957a49146103e85780632cbf74e5146103f05780632f3985a71461041757600080fd5b80631f956632146103af5780631ff64790146103c2578063258f0495146103d557600080fd5b80631637be48116102e85780631637be481461035f57806316af91a7146103925780631ad87e451461039a57600080fd5b806302c992751461030457806306f1305614610349575b600080fd5b61032b7f200000000000000000000000000000000000000000000000000000000000000081565b6040516001600160f81b031990911681526020015b60405180910390f35b61035161081d565b604051908152602001610340565b61038261036d3660046145ad565b60009081526008602052604090205460ff1690565b6040519015158152602001610340565b61032b600081565b6103ad6103a83660046146fa565b6108a7565b005b6103ad6103bd36600461475b565b610bbd565b6103ad6103d0366004614794565b610ce9565b6103516103e33660046145ad565b610e81565b610351602881565b61032b7f500000000000000000000000000000000000000000000000000000000000000081565b6103ad6104253660046147b1565b610eee565b6103ad6104383660046147cd565b610ffe565b6103827f000000000000000000000000000000000000000000000000000000000000000081565b6103ad6104723660046145ad565b6112e7565b6103ad610485366004614878565b611504565b6103ad61183a565b61032b7f080000000000000000000000000000000000000000000000000000000000000081565b6103826104c7366004614794565b60096020526000908152604090205460ff1681565b6103ad6104ea366004614878565b611a12565b6103ad6104fd36600461475b565b611ac4565b6103ad610510366004614906565b611bf0565b6105496105233660046145ad565b60086020526000908152604090205460ff811690610100900467ffffffffffffffff1682565b60408051921515835267ffffffffffffffff909116602083015201610340565b610382610577366004614794565b60036020526000908152604090205460ff1681565b61035160005481565b6103ad6105a33660046145ad565b611c22565b6105cf7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b039091168152602001610340565b6103ad6105f5366004614971565b611d97565b6103ad6106083660046149ee565b61209e565b6103827f000000000000000000000000000000000000000000000000000000000000000081565b6103ad6123b2565b6103ad61064a3660046147b1565b61242a565b6103ad61065d366004614a4a565b6125ce565b6002546105cf906001600160a01b031681565b600b546105cf906001600160a01b031681565b6103ad610696366004614ab0565b6126dd565b6103516106a93660046145ad565b612a2a565b6106c16106bc366004614af2565b612ab7565b60405167ffffffffffffffff9091168152602001610340565b6103ad6106e8366004614971565b612b1a565b61032b7f400000000000000000000000000000000000000000000000000000000000000081565b6001546105cf906001600160a01b031681565b6103517f000000000000000000000000000000000000000000000000000000000000000081565b610756612ba2565b604080519485526020850193909352918301526060820152608001610340565b600c54600d546107bc9167ffffffffffffffff8082169268010000000000000000808404831693600160801b8104841693600160c01b9091048116928082169290041686565b6040805167ffffffffffffffff978816815295871660208701529386169385019390935290841660608401528316608083015290911660a082015260c001610340565b6103ad61080d366004614b1e565b612bdb565b61032b600160ff1b81565b600154604080517e84120c00000000000000000000000000000000000000000000000000000000815290516000926001600160a01b0316916284120c9160048083019260209291908290030181865afa15801561087e573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906108a29190614b86565b905090565b6001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016300361094a5760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084015b60405180910390fd5b6001546001600160a01b03161561098d576040517fef34ca5c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001600160a01b0383166109cd576040517f1ad0f74300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000836001600160a01b031663e1758bd86040518163ffffffff1660e01b8152600401602060405180830381865afa925050508015610a29575060408051601f3d908101601f19168201909252610a2691810190614b9f565b60015b15610a44576001600160a01b03811615610a4257600191505b505b8015157f0000000000000000000000000000000000000000000000000000000000000000151514610aa1576040517fc3e31f8d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038616908117909155604080517fcb23bcb5000000000000000000000000000000000000000000000000000000008152905163cb23bcb5916004808201926020929091908290030181865afa158015610b20573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b449190614b9f565b6002805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0392909216919091179055610b88610b8336859003850185614a4a565b613044565b7f000000000000000000000000000000000000000000000000000000000000000015610bb757610bb782613170565b50505050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610c10573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610c349190614b9f565b6001600160a01b0316336001600160a01b031614158015610c605750600b546001600160a01b03163314155b15610c99576040517f660b3b42000000000000000000000000000000000000000000000000000000008152336004820152602401610941565b6001600160a01b038216600090815260096020526040808220805460ff1916841515179055516004917fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e91a25050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610d3c573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610d609190614b9f565b6001600160a01b0316336001600160a01b031614610e2a5760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dc1573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610de59190614b9f565b6040517f23295f0e0000000000000000000000000000000000000000000000000000000081526001600160a01b03928316600482015291166024820152604401610941565b600b805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0383161790556040516005907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a250565b600081815260086020908152604080832081518083019092525460ff811615158252610100900467ffffffffffffffff16918101829052908203610eda5760405162f20c5d60e01b815260048101849052602401610941565b6020015167ffffffffffffffff1692915050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610f41573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610f659190614b9f565b6001600160a01b0316336001600160a01b031614610fc65760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dc1573d6000803e3d6000fd5b610fcf81613170565b6040516006907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a250565b827f000000000000000000000000000000000000000000000000000000000000000060005a3360009081526003602052604090205490915060ff1661105657604051632dd9fc9760e01b815260040160405180910390fd5b61105f876133b0565b1561107d57604051630e5da8fb60e01b815260040160405180910390fd5b611089888887876133f8565b6001600160a01b038316156112dd5736600060206110a883601f614bd2565b6110b29190614be5565b90506102006110c2600283614ceb565b6110cc9190614be5565b6110d7826006614cfa565b6110e19190614bd2565b6110eb9084614bd2565b92503332146110fd5760009150611230565b6001600160a01b0384161561123057836001600160a01b031663e83a2d826040518163ffffffff1660e01b8152600401600060405180830381865afa92505050801561116b57506040513d6000823e601f3d908101601f191682016040526111689190810190614d11565b60015b156112305780511561122e576000856001600160a01b0316631f6d6ef76040518163ffffffff1660e01b8152600401602060405180830381865afa1580156111b7573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906111db9190614b86565b905048817f0000000000000000000000000000000000000000000000000000000000000000845161120c9190614cfa565b6112169190614cfa565b6112209190614be5565b61122a9086614bd2565b9450505b505b846001600160a01b031663e3db8a49335a61124b9087614db7565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af11580156112b5573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906112d99190614dca565b5050505b5050505050505050565b6000816040516020016112fc91815260200190565b60408051808303601f1901815290829052600154815160208301207f8db5993b000000000000000000000000000000000000000000000000000000008452600b6004850152600060248501819052604485019190915291935090916001600160a01b0390911690638db5993b906064016020604051808303816000875af115801561138b573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906113af9190614b86565b905080156113ff5760405162461bcd60e51b815260206004820152601460248201527f414c52454144595f44454c415945445f494e49540000000000000000000000006044820152606401610941565b807fff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b8360405161142f9190614e0b565b60405180910390a26000806114446001613554565b9150915060008060008061145e8660016000806001613599565b9350935093509350836000146114b65760405162461bcd60e51b815260206004820152601060248201527f414c52454144595f5345515f494e4954000000000000000000000000000000006044820152606401610941565b8083857f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548a60026040516114f19493929190614e3e565b60405180910390a4505050505050505050565b836000805a9050333214611544576040517ffeb3d07100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526003602052604090205460ff1661157457604051632dd9fc9760e01b815260040160405180910390fd5b7f00000000000000000000000000000000000000000000000000000000000000006115b257604051631192b39960e31b815260040160405180910390fd5b6115ca886115c536879003870187614eb3565b613782565b6115da8b8b8b8b8a8a600161388f565b6001600160a01b038316156112d95736600060206115f983601f614bd2565b6116039190614be5565b9050610200611613600283614ceb565b61161d9190614be5565b611628826006614cfa565b6116329190614bd2565b61163c9084614bd2565b925033321461164e5760009150611781565b6001600160a01b0384161561178157836001600160a01b031663e83a2d826040518163ffffffff1660e01b8152600401600060405180830381865afa9250505080156116bc57506040513d6000823e601f3d908101601f191682016040526116b99190810190614d11565b60015b156117815780511561177f576000856001600160a01b0316631f6d6ef76040518163ffffffff1660e01b8152600401602060405180830381865afa158015611708573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061172c9190614b86565b905048817f0000000000000000000000000000000000000000000000000000000000000000845161175d9190614cfa565b6117679190614cfa565b6117719190614be5565b61177b9086614bd2565b9450505b505b846001600160a01b031663e3db8a49335a61179c9087614db7565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af1158015611806573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061182a9190614dca565b5050505050505050505050505050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561188d573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906118b19190614b9f565b6001600160a01b0316336001600160a01b0316146119125760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dc1573d6000803e3d6000fd5b600154604080517fcb23bcb500000000000000000000000000000000000000000000000000000000815290516000926001600160a01b03169163cb23bcb59160048083019260209291908290030181865afa158015611975573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906119999190614b9f565b6002549091506001600160a01b038083169116036119e3576040517fd054909f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6002805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0392909216919091179055565b836000805a3360009081526003602052604090205490915060ff16158015611a4557506002546001600160a01b03163314155b15611a6357604051632dd9fc9760e01b815260040160405180910390fd5b7f0000000000000000000000000000000000000000000000000000000000000000611aa157604051631192b39960e31b815260040160405180910390fd5b611ab4886115c536879003870187614eb3565b6115da8b8b8b8b8a8a600061388f565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611b17573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611b3b9190614b9f565b6001600160a01b0316336001600160a01b031614158015611b675750600b546001600160a01b03163314155b15611ba0576040517f660b3b42000000000000000000000000000000000000000000000000000000008152336004820152602401610941565b6001600160a01b038216600090815260036020526040808220805460ff1916841515179055516001917fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e91a25050565b6040517fc73b9d7c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611c75573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611c999190614b9f565b6001600160a01b0316336001600160a01b031614611cfa5760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dc1573d6000803e3d6000fd5b60008181526008602052604090205460ff16611d2b5760405162f20c5d60e01b815260048101829052602401610941565b600081815260086020526040808220805460ff191690555182917f5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a91a26040516003907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a250565b826000805a9050333214611dd7576040517ffeb3d07100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526003602052604090205460ff16611e0757604051632dd9fc9760e01b815260040160405180910390fd5b611e10876133b0565b15611e2e57604051630e5da8fb60e01b815260040160405180910390fd5b611e3e8a8a8a8a8989600161388f565b6001600160a01b03831615612092573660006020611e5d83601f614bd2565b611e679190614be5565b9050610200611e77600283614ceb565b611e819190614be5565b611e8c826006614cfa565b611e969190614bd2565b611ea09084614bd2565b9250333214611eb25760009150611fe5565b6001600160a01b03841615611fe557836001600160a01b031663e83a2d826040518163ffffffff1660e01b8152600401600060405180830381865afa925050508015611f2057506040513d6000823e601f3d908101601f19168201604052611f1d9190810190614d11565b60015b15611fe557805115611fe3576000856001600160a01b0316631f6d6ef76040518163ffffffff1660e01b8152600401602060405180830381865afa158015611f6c573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611f909190614b86565b905048817f00000000000000000000000000000000000000000000000000000000000000008451611fc19190614cfa565b611fcb9190614cfa565b611fd59190614be5565b611fdf9086614bd2565b9450505b505b846001600160a01b031663e3db8a49335a6120009087614db7565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af115801561206a573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061208e9190614dca565b5050505b50505050505050505050565b837f000000000000000000000000000000000000000000000000000000000000000060005a3360009081526003602052604090205490915060ff166120f657604051632dd9fc9760e01b815260040160405180910390fd5b7f000000000000000000000000000000000000000000000000000000000000000061213457604051631192b39960e31b815260040160405180910390fd5b612147886115c536879003870187614eb3565b612153898988886133f8565b6001600160a01b038316156123a757366000602061217283601f614bd2565b61217c9190614be5565b905061020061218c600283614ceb565b6121969190614be5565b6121a1826006614cfa565b6121ab9190614bd2565b6121b59084614bd2565b92503332146121c757600091506122fa565b6001600160a01b038416156122fa57836001600160a01b031663e83a2d826040518163ffffffff1660e01b8152600401600060405180830381865afa92505050801561223557506040513d6000823e601f3d908101601f191682016040526122329190810190614d11565b60015b156122fa578051156122f8576000856001600160a01b0316631f6d6ef76040518163ffffffff1660e01b8152600401602060405180830381865afa158015612281573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906122a59190614b86565b905048817f000000000000000000000000000000000000000000000000000000000000000084516122d69190614cfa565b6122e09190614cfa565b6122ea9190614be5565b6122f49086614bd2565b9450505b505b846001600160a01b031663e3db8a49335a6123159087614db7565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af115801561237f573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906123a39190614dca565b5050505b505050505050505050565b467f00000000000000000000000000000000000000000000000000000000000000000361240b576040517fa301bb0600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b7801000000000000000100000000000000010000000000000001600a55565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036124c85760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c00000000000000000000000000000000000000006064820152608401610941565b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038054336001600160a01b0382161461253e576040517f23295f0e0000000000000000000000000000000000000000000000000000000081523360048201526001600160a01b0382166024820152604401610941565b7f000000000000000000000000000000000000000000000000000000000000000061257c57604051631192b39960e31b815260040160405180910390fd5b600c5467ffffffffffffffff16156125c0576040517fef34ca5c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6125c983613170565b505050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015612621573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906126459190614b9f565b6001600160a01b0316336001600160a01b0316146126a65760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dc1573d6000803e3d6000fd5b6126af81613044565b6040516000907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e908290a250565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015612730573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906127549190614b9f565b6001600160a01b0316336001600160a01b0316146127b55760025460408051638da5cb5b60e01b8152905133926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610dc1573d6000803e3d6000fd5b600082826040516127c7929190614f61565b6040519081900381207ffe000000000000000000000000000000000000000000000000000000000000006020830152602182015260410160408051601f1981840301815291905280516020909101209050600160ff1b811862010000831061285b576040517fb3d1f41200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008181526008602052604090205460ff16156128a7576040517ffa2fddda00000000000000000000000000000000000000000000000000000000815260048101829052602401610941565b437f0000000000000000000000000000000000000000000000000000000000000000156129345760646001600160a01b031663a3b1b31d6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561290d573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906129319190614b86565b90505b6040805180820182526001815267ffffffffffffffff8381166020808401918252600087815260089091528490209251835491517fffffffffffffffffffffffffffffffffffffffffffffff0000000000000000009092169015157fffffffffffffffffffffffffffffffffffffffffffffff0000000000000000ff161761010091909216021790555182907fabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722906129ef9088908890614f71565b60405180910390a26040516002907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a25050505050565b6001546040517f16bf5579000000000000000000000000000000000000000000000000000000008152600481018390526000916001600160a01b0316906316bf557990602401602060405180830381865afa158015612a8d573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612ab19190614b86565b92915050565b600a5460009067ffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000015612b09576000612afa600c856139ba565b9050612b0581613a09565b9150505b612b138184614fa0565b9392505050565b826000805a3360009081526003602052604090205490915060ff16158015612b4d57506002546001600160a01b03163314155b15612b6b57604051632dd9fc9760e01b815260040160405180910390fd5b612b74876133b0565b15612b9257604051630e5da8fb60e01b815260040160405180910390fd5b611e3e8a8a8a8a8989600061388f565b600080600080600080600080612bb6613a39565b67ffffffffffffffff9384169b50918316995082169750169450505050505b90919293565b6000548611612c16576040517f7d73e6fa00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000612c4c8684612c2a6020890189614af2565b612c3a60408a0160208b01614af2565b612c4560018d614db7565b8988613ab0565b600a5490915067ffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000015612cbd57612c9a612c926020880188614af2565b600c90613b55565b600c54612cb09067ffffffffffffffff16613a09565b67ffffffffffffffff1690505b4381612ccc6020890189614af2565b67ffffffffffffffff16612ce09190614bd2565b10612d17576040517fad3515d900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006001891115612da0576001546001600160a01b031663d5719dc2612d3e60028c614db7565b6040518263ffffffff1660e01b8152600401612d5c91815260200190565b602060405180830381865afa158015612d79573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612d9d9190614b86565b90505b60408051602080820184905281830186905282518083038401815260609092019092528051910120600180546001600160a01b03169063d5719dc290612de6908d614db7565b6040518263ffffffff1660e01b8152600401612e0491815260200190565b602060405180830381865afa158015612e21573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612e459190614b86565b14612e7c576040517f13947fd700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600080612e888b613554565b9150915060008b90506000600160009054906101000a90046001600160a01b03166001600160a01b0316635fca4a166040518163ffffffff1660e01b8152600401602060405180830381865afa158015612ee6573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612f0a9190614b86565b9050806000808080612f1f8988838880613599565b93509350935093508083857f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548d6002604051612f629493929190614e3e565b60405180910390a4505050505050505050505050505050505050565b60408051600481526024810182526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f051038f200000000000000000000000000000000000000000000000000000000179052905160009182918291606491612fea9190614fde565b600060405180830381855afa9150503d8060008114613025576040519150601f19603f3d011682016040523d82523d6000602084013e61302a565b606091505b509150915081801561303d575080516020145b9250505090565b805167ffffffffffffffff10806130665750602081015167ffffffffffffffff105b8061307c5750604081015167ffffffffffffffff105b806130925750606081015167ffffffffffffffff105b156130c9576040517f09cfba7500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8051600a80546020840151604085015160609095015167ffffffffffffffff908116600160c01b0277ffffffffffffffffffffffffffffffffffffffffffffffff968216600160801b02969096166fffffffffffffffffffffffffffffffff92821668010000000000000000027fffffffffffffffffffffffffffffffff000000000000000000000000000000009094169190951617919091171691909117919091179055565b7f00000000000000000000000000000000000000000000000000000000000000006131ae57604051631192b39960e31b815260040160405180910390fd5b6131b781613bdb565b6131ed576040517fda1c8eb600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600c5467ffffffffffffffff16158061321957506020810151600c5467ffffffffffffffff9182169116115b15613245576020810151600c805467ffffffffffffffff191667ffffffffffffffff9092169190911790555b8051600c5467ffffffffffffffff91821691161015613282578051600c805467ffffffffffffffff191667ffffffffffffffff9092169190911790555b602081810151600c805484517fffffffffffffffff00000000000000000000000000000000ffffffffffffffff9091166801000000000000000067ffffffffffffffff948516027fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff1617600160801b91841691909102179055604080840151600d805467ffffffffffffffff1916919093161790915560005460015482517feca067ad000000000000000000000000000000000000000000000000000000008152925191936001600160a01b039091169263eca067ad92600480830193928290030181865afa158015613379573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061339d9190614b86565b036133ad576133ad600c43613b55565b50565b60007f000000000000000000000000000000000000000000000000000000000000000080156133e0575060005482115b8015612ab157506133f1600c613c43565b1592915050565b600080600061340686613c76565b925092509250600080600080613420878b60008c8c613599565b93509350935093508a841415801561343a57506000198b14155b1561347b576040517fac7411c900000000000000000000000000000000000000000000000000000000815260048101859052602481018c9052604401610941565b80838c7f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548b60036040516134b69493929190614e3e565b60405180910390a47f000000000000000000000000000000000000000000000000000000000000000015613516576040517f86657a5300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b333214801561354357507f0000000000000000000000000000000000000000000000000000000000000000155b156112d9576112d987854888613ea5565b6040805160808101825260008082526020820181905291810182905260608101829052600080613583856140e7565b8151602090920191909120969095509350505050565b6000806000806000548810156135db576040517f7d73e6fa00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600160009054906101000a90046001600160a01b03166001600160a01b031663eca067ad6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561362e573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906136529190614b86565b88111561368b576040517f925f8bd300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001546040517f86598a56000000000000000000000000000000000000000000000000000000008152600481018b9052602481018a905260448101889052606481018790526001600160a01b03909116906386598a56906084016080604051808303816000875af1158015613704573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906137289190614ffa565b60008c905592965090945092509050861580159061376457507f0000000000000000000000000000000000000000000000000000000000000000155b15613776576137768985486000613ea5565b95509550955095915050565b60005482111561388b57613796600c6141a2565b1561388b57600154600080546040517fd5719dc200000000000000000000000000000000000000000000000000000000815291926001600160a01b03169163d5719dc2916137ea9160040190815260200190565b602060405180830381865afa158015613807573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061382b9190614b86565b905061384081836000015184602001516141d3565b613876576040517fc334542d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6020820151604001516125c990600c90613b55565b5050565b60008061389d888888614218565b915091506000806000806138c1868b896138b85760006138ba565b8d5b8c8c613599565b93509350935093508c84141580156138db57506000198d14155b1561391c576040517fac7411c900000000000000000000000000000000000000000000000000000000815260048101859052602481018e9052604401610941565b8083857f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548a8d613951576001613954565b60005b6040516139649493929190614e3e565b60405180910390a48661208e57837ffe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c208d8d6040516139a3929190614f71565b60405180910390a250505050505050505050505050565b81546001830154600091612b139167ffffffffffffffff600160c01b8304811692868216928282169268010000000000000000808304821693600160801b810483169391900482169116614426565b600a5460009067ffffffffffffffff90811690831610613a3557600a5467ffffffffffffffff16612ab1565b5090565b6000808080467f000000000000000000000000000000000000000000000000000000000000000014613a7657506001925082915081905080612bd5565b5050600a5467ffffffffffffffff808216935068010000000000000000820481169250600160801b8204811691600160c01b900416612bd5565b6040516001600160f81b031960f889901b1660208201526bffffffffffffffffffffffff19606088901b1660218201527fffffffffffffffff00000000000000000000000000000000000000000000000060c087811b8216603584015286901b16603d82015260458101849052606581018390526085810182905260009060a5016040516020818303038152906040528051906020012090505b979650505050505050565b613b5f82826139ba565b825467ffffffffffffffff928316600160c01b0277ffffffffffffffffffffffffffffffff000000000000000090911691831691909117178255600190910180544390921668010000000000000000027fffffffffffffffffffffffffffffffff0000000000000000ffffffffffffffff909216919091179055565b805160009067ffffffffffffffff1615801590613c055750602082015167ffffffffffffffff1615155b8015613c215750612710826040015167ffffffffffffffff1611155b8015612ab15750506020810151905167ffffffffffffffff9182169116111590565b805460009067ffffffffffffffff600160801b8204811691613c6e91600160c01b9091041643614db7565b111592915050565b60408051608081018252600080825260208201819052918101829052606081018290526000807f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031663e83a2d826040518163ffffffff1660e01b8152600401600060405180830381865afa158015613cfa573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f19168201604052613d229190810190614d11565b90508051600003613d5f576040517f3cd27eb600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600080613d6b876140e7565b9150915060008351620200007f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316631f6d6ef76040518163ffffffff1660e01b8152600401602060405180830381865afa158015613dd5573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190613df99190614b86565b613e039190614cfa565b613e0d9190614cfa565b60405190915083907f500000000000000000000000000000000000000000000000000000000000000090613e45908790602001615030565b60408051601f1981840301815290829052613e64939291602001615066565b604051602081830303815290604052805190602001208260004811613e8a576000613e94565b613e944884614be5565b965096509650505050509193909250565b327f000000000000000000000000000000000000000000000000000000000000000015613f4b576000606c6001600160a01b031663c6f7de0e6040518163ffffffff1660e01b8152600401602060405180830381865afa158015613f0d573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190613f319190614b86565b9050613f3d4882614be5565b613f479084614bd2565b9250505b67ffffffffffffffff821115613f8d576040517f04d5501200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b604080514260208201526bffffffffffffffffffffffff19606084901b16918101919091526054810186905260748101859052609481018490527fffffffffffffffff00000000000000000000000000000000000000000000000060c084901b1660b482015260009060bc0160408051808303601f1901815290829052600154815160208301207f7a88b1070000000000000000000000000000000000000000000000000000000084526001600160a01b0386811660048601526024850191909152919350600092911690637a88b107906044016020604051808303816000875af1158015614080573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906140a49190614b86565b9050807fff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b836040516140d69190614e0b565b60405180910390a250505050505050565b604080516080808201835260008083526020808401829052838501829052606080850183905285519384018652828452838201839052838601839052838101839052855191820183905260288201839052603082018390526038820183905260c087901b7fffffffffffffffff000000000000000000000000000000000000000000000000169582019590955260480160405160208183030381529060405290506028815114614199576141996150a9565b94909350915050565b60006141ad82613c43565b1580612ab15750505467ffffffffffffffff680100000000000000008204811691161090565b600061420e836141e2846144ed565b604080516020808201949094528082019290925280518083038201815260609092019052805191012090565b9093149392505050565b60408051608081018252600080825260208201819052918101829052606081018290526000614248856028614bd2565b90507f00000000000000000000000000000000000000000000000000000000000000008111156142cd576040517f4634691b000000000000000000000000000000000000000000000000000000008152600481018290527f00000000000000000000000000000000000000000000000000000000000000006024820152604401610941565b6000806142d9866140e7565b909250905086156143ec57614309888860008181106142fa576142fa614fc8565b9050013560f81c60f81b61451a565b614361578787600081811061432057614320614fc8565b6040517f6b3333560000000000000000000000000000000000000000000000000000000081529201356001600160f81b031916600483015250602401610941565b600160ff1b888860008161437757614377614fc8565b6001600160f81b031992013592909216161580159150614398575060218710155b156143ec5760006143ad602160018a8c6150bf565b6143b6916150e9565b60008181526008602052604090205490915060ff166143ea5760405162f20c5d60e01b815260048101829052602401610941565b505b81888860405160200161440193929190615107565b60408051601f1981840301815291905280516020909101209890975095505050505050565b600080888811614437576000614441565b6144418989614db7565b9050600089871161445357600061445d565b61445d8a88614db7565b905061271061446c8584614cfa565b6144769190614be5565b6144809089614bd2565b9750600086821161449257600061449c565b61449c8783614db7565b9050828111156144a95750815b808911156144de576144bb818a614db7565b9850868911156144de578589116144d257886144d4565b855b9350505050613b4a565b50949998505050505050505050565b6000612ab1826000015183602001518460400151856060015186608001518760a001518860c00151613ab0565b60006001600160f81b03198216158061454057506001600160f81b03198216600160ff1b145b8061457457506001600160f81b031982167f8800000000000000000000000000000000000000000000000000000000000000145b80612ab157506001600160f81b031982167f20000000000000000000000000000000000000000000000000000000000000001492915050565b6000602082840312156145bf57600080fd5b5035919050565b6001600160a01b03811681146133ad57600080fd5b634e487b7160e01b600052604160045260246000fd5b6040805190810167ffffffffffffffff81118282101715614614576146146145db565b60405290565b60405160e0810167ffffffffffffffff81118282101715614614576146146145db565b604051601f8201601f1916810167ffffffffffffffff81118282101715614666576146666145db565b604052919050565b803567ffffffffffffffff8116811461468657600080fd5b919050565b60006060828403121561469d57600080fd5b6040516060810181811067ffffffffffffffff821117156146c0576146c06145db565b6040529050806146cf8361466e565b81526146dd6020840161466e565b60208201526146ee6040840161466e565b60408201525092915050565b600080600083850361010081121561471157600080fd5b843561471c816145c6565b93506080601f198201121561473057600080fd5b506020840191506147448560a0860161468b565b90509250925092565b80151581146133ad57600080fd5b6000806040838503121561476e57600080fd5b8235614779816145c6565b915060208301356147898161474d565b809150509250929050565b6000602082840312156147a657600080fd5b8135612b13816145c6565b6000606082840312156147c357600080fd5b612b13838361468b565b600080600080600060a086880312156147e557600080fd5b853594506020860135935060408601356147fe816145c6565b94979396509394606081013594506080013592915050565b60008083601f84011261482857600080fd5b50813567ffffffffffffffff81111561484057600080fd5b60208301915083602082850101111561485857600080fd5b9250929050565b6000610100828403121561487257600080fd5b50919050565b6000806000806000806000806101c0898b03121561489557600080fd5b88359750602089013567ffffffffffffffff8111156148b357600080fd5b6148bf8b828c01614816565b9098509650506040890135945060608901356148da816145c6565b93506080890135925060a089013591506148f78a60c08b0161485f565b90509295985092959890939650565b60008060008060006080868803121561491e57600080fd5b85359450602086013567ffffffffffffffff81111561493c57600080fd5b61494888828901614816565b909550935050604086013591506060860135614963816145c6565b809150509295509295909350565b600080600080600080600060c0888a03121561498c57600080fd5b87359650602088013567ffffffffffffffff8111156149aa57600080fd5b6149b68a828b01614816565b9097509550506040880135935060608801356149d1816145c6565b969995985093969295946080840135945060a09093013592915050565b6000806000806000806101a08789031215614a0857600080fd5b86359550602087013594506040870135614a21816145c6565b93506060870135925060808701359150614a3e8860a0890161485f565b90509295509295509295565b600060808284031215614a5c57600080fd5b6040516080810181811067ffffffffffffffff82111715614a7f57614a7f6145db565b8060405250823581526020830135602082015260408301356040820152606083013560608201528091505092915050565b60008060208385031215614ac357600080fd5b823567ffffffffffffffff811115614ada57600080fd5b614ae685828601614816565b90969095509350505050565b600060208284031215614b0457600080fd5b612b138261466e565b803560ff8116811461468657600080fd5b60008060008060008060e08789031215614b3757600080fd5b86359550614b4760208801614b0d565b94506080870188811115614b5a57600080fd5b60408801945035925060a0870135614b71816145c6565b8092505060c087013590509295509295509295565b600060208284031215614b9857600080fd5b5051919050565b600060208284031215614bb157600080fd5b8151612b13816145c6565b634e487b7160e01b600052601160045260246000fd5b80820180821115612ab157612ab1614bbc565b600082614c0257634e487b7160e01b600052601260045260246000fd5b500490565b600181815b80851115614c42578160001904821115614c2857614c28614bbc565b80851615614c3557918102915b93841c9390800290614c0c565b509250929050565b600082614c5957506001612ab1565b81614c6657506000612ab1565b8160018114614c7c5760028114614c8657614ca2565b6001915050612ab1565b60ff841115614c9757614c97614bbc565b50506001821b612ab1565b5060208310610133831016604e8410600b8410161715614cc5575081810a612ab1565b614ccf8383614c07565b8060001904821115614ce357614ce3614bbc565b029392505050565b6000612b1360ff841683614c4a565b8082028115828204841417612ab157612ab1614bbc565b60006020808385031215614d2457600080fd5b825167ffffffffffffffff80821115614d3c57600080fd5b818501915085601f830112614d5057600080fd5b815181811115614d6257614d626145db565b8060051b9150614d7384830161463d565b8181529183018401918481019088841115614d8d57600080fd5b938501935b83851015614dab57845182529385019390850190614d92565b98975050505050505050565b81810381811115612ab157612ab1614bbc565b600060208284031215614ddc57600080fd5b8151612b138161474d565b60005b83811015614e02578181015183820152602001614dea565b50506000910152565b6020815260008251806020840152614e2a816040850160208701614de7565b601f01601f19169190910160400192915050565b600060e08201905085825284602083015267ffffffffffffffff8085511660408401528060208601511660608401528060408601511660808401528060608601511660a08401525060048310614ea457634e487b7160e01b600052602160045260246000fd5b8260c083015295945050505050565b6000818303610100811215614ec757600080fd5b614ecf6145f1565b8335815260e0601f1983011215614ee557600080fd5b614eed61461a565b9150614efb60208501614b0d565b82526040840135614f0b816145c6565b6020830152614f1c6060850161466e565b6040830152614f2d6080850161466e565b606083015260a0840135608083015260c084013560a083015260e084013560c0830152816020820152809250505092915050565b8183823760009101908152919050565b60208152816020820152818360408301376000818301604090810191909152601f909201601f19160101919050565b67ffffffffffffffff818116838216019080821115614fc157614fc1614bbc565b5092915050565b634e487b7160e01b600052603260045260246000fd5b60008251614ff0818460208701614de7565b9190910192915050565b6000806000806080858703121561501057600080fd5b505082516020840151604085015160609095015191969095509092509050565b815160009082906020808601845b8381101561505a5781518552938201939082019060010161503e565b50929695505050505050565b60008451615078818460208901614de7565b6001600160f81b03198516908301908152835161509c816001840160208801614de7565b0160010195945050505050565b634e487b7160e01b600052600160045260246000fd5b600080858511156150cf57600080fd5b838611156150dc57600080fd5b5050820193919092039150565b80356020831015612ab157600019602084900360031b1b1692915050565b60008451615119818460208901614de7565b820183858237600093019283525090939250505056fea26469706673582212204a352d17908339cbf1a8ff29b7adbd67b410653eec554019cb4fdc918c9bdf4564736f6c63430008110033",
}

// SequencerInboxStubABI is the input ABI used to generate the binding from.
// Deprecated: Use SequencerInboxStubMetaData.ABI instead.
var SequencerInboxStubABI = SequencerInboxStubMetaData.ABI

// SequencerInboxStubBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SequencerInboxStubMetaData.Bin instead.
var SequencerInboxStubBin = SequencerInboxStubMetaData.Bin

// DeploySequencerInboxStub deploys a new Ethereum contract, binding an instance of SequencerInboxStub to it.
func DeploySequencerInboxStub(auth *bind.TransactOpts, backend bind.ContractBackend, bridge_ common.Address, sequencer_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation, maxDataSize_ *big.Int, reader4844_ common.Address, isUsingFeeToken_ bool, isDelayBufferable_ bool) (common.Address, *types.Transaction, *SequencerInboxStub, error) {
	parsed, err := SequencerInboxStubMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SequencerInboxStubBin), backend, bridge_, sequencer_, maxTimeVariation_, maxDataSize_, reader4844_, isUsingFeeToken_, isDelayBufferable_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SequencerInboxStub{SequencerInboxStubCaller: SequencerInboxStubCaller{contract: contract}, SequencerInboxStubTransactor: SequencerInboxStubTransactor{contract: contract}, SequencerInboxStubFilterer: SequencerInboxStubFilterer{contract: contract}}, nil
}

// SequencerInboxStub is an auto generated Go binding around an Ethereum contract.
type SequencerInboxStub struct {
	SequencerInboxStubCaller     // Read-only binding to the contract
	SequencerInboxStubTransactor // Write-only binding to the contract
	SequencerInboxStubFilterer   // Log filterer for contract events
}

// SequencerInboxStubCaller is an auto generated read-only Go binding around an Ethereum contract.
type SequencerInboxStubCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerInboxStubTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SequencerInboxStubTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerInboxStubFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SequencerInboxStubFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerInboxStubSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SequencerInboxStubSession struct {
	Contract     *SequencerInboxStub // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// SequencerInboxStubCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SequencerInboxStubCallerSession struct {
	Contract *SequencerInboxStubCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// SequencerInboxStubTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SequencerInboxStubTransactorSession struct {
	Contract     *SequencerInboxStubTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// SequencerInboxStubRaw is an auto generated low-level Go binding around an Ethereum contract.
type SequencerInboxStubRaw struct {
	Contract *SequencerInboxStub // Generic contract binding to access the raw methods on
}

// SequencerInboxStubCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SequencerInboxStubCallerRaw struct {
	Contract *SequencerInboxStubCaller // Generic read-only contract binding to access the raw methods on
}

// SequencerInboxStubTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SequencerInboxStubTransactorRaw struct {
	Contract *SequencerInboxStubTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSequencerInboxStub creates a new instance of SequencerInboxStub, bound to a specific deployed contract.
func NewSequencerInboxStub(address common.Address, backend bind.ContractBackend) (*SequencerInboxStub, error) {
	contract, err := bindSequencerInboxStub(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStub{SequencerInboxStubCaller: SequencerInboxStubCaller{contract: contract}, SequencerInboxStubTransactor: SequencerInboxStubTransactor{contract: contract}, SequencerInboxStubFilterer: SequencerInboxStubFilterer{contract: contract}}, nil
}

// NewSequencerInboxStubCaller creates a new read-only instance of SequencerInboxStub, bound to a specific deployed contract.
func NewSequencerInboxStubCaller(address common.Address, caller bind.ContractCaller) (*SequencerInboxStubCaller, error) {
	contract, err := bindSequencerInboxStub(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStubCaller{contract: contract}, nil
}

// NewSequencerInboxStubTransactor creates a new write-only instance of SequencerInboxStub, bound to a specific deployed contract.
func NewSequencerInboxStubTransactor(address common.Address, transactor bind.ContractTransactor) (*SequencerInboxStubTransactor, error) {
	contract, err := bindSequencerInboxStub(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStubTransactor{contract: contract}, nil
}

// NewSequencerInboxStubFilterer creates a new log filterer instance of SequencerInboxStub, bound to a specific deployed contract.
func NewSequencerInboxStubFilterer(address common.Address, filterer bind.ContractFilterer) (*SequencerInboxStubFilterer, error) {
	contract, err := bindSequencerInboxStub(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStubFilterer{contract: contract}, nil
}

// bindSequencerInboxStub binds a generic wrapper to an already deployed contract.
func bindSequencerInboxStub(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SequencerInboxStubMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SequencerInboxStub *SequencerInboxStubRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SequencerInboxStub.Contract.SequencerInboxStubCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SequencerInboxStub *SequencerInboxStubRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SequencerInboxStubTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SequencerInboxStub *SequencerInboxStubRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SequencerInboxStubTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SequencerInboxStub *SequencerInboxStubCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SequencerInboxStub.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SequencerInboxStub *SequencerInboxStubTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SequencerInboxStub *SequencerInboxStubTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.contract.Transact(opts, method, params...)
}

// BROTLIMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x16af91a7.
//
// Solidity: function BROTLI_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCaller) BROTLIMESSAGEHEADERFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "BROTLI_MESSAGE_HEADER_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// BROTLIMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x16af91a7.
//
// Solidity: function BROTLI_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubSession) BROTLIMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.BROTLIMESSAGEHEADERFLAG(&_SequencerInboxStub.CallOpts)
}

// BROTLIMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x16af91a7.
//
// Solidity: function BROTLI_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) BROTLIMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.BROTLIMESSAGEHEADERFLAG(&_SequencerInboxStub.CallOpts)
}

// DASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0xf60a5091.
//
// Solidity: function DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCaller) DASMESSAGEHEADERFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "DAS_MESSAGE_HEADER_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// DASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0xf60a5091.
//
// Solidity: function DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubSession) DASMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.DASMESSAGEHEADERFLAG(&_SequencerInboxStub.CallOpts)
}

// DASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0xf60a5091.
//
// Solidity: function DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) DASMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.DASMESSAGEHEADERFLAG(&_SequencerInboxStub.CallOpts)
}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCaller) DATAAUTHENTICATEDFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "DATA_AUTHENTICATED_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubSession) DATAAUTHENTICATEDFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.DATAAUTHENTICATEDFLAG(&_SequencerInboxStub.CallOpts)
}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) DATAAUTHENTICATEDFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.DATAAUTHENTICATEDFLAG(&_SequencerInboxStub.CallOpts)
}

// DATABLOBHEADERFLAG is a free data retrieval call binding the contract method 0x2cbf74e5.
//
// Solidity: function DATA_BLOB_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCaller) DATABLOBHEADERFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "DATA_BLOB_HEADER_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// DATABLOBHEADERFLAG is a free data retrieval call binding the contract method 0x2cbf74e5.
//
// Solidity: function DATA_BLOB_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubSession) DATABLOBHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.DATABLOBHEADERFLAG(&_SequencerInboxStub.CallOpts)
}

// DATABLOBHEADERFLAG is a free data retrieval call binding the contract method 0x2cbf74e5.
//
// Solidity: function DATA_BLOB_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) DATABLOBHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.DATABLOBHEADERFLAG(&_SequencerInboxStub.CallOpts)
}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubCaller) HEADERLENGTH(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "HEADER_LENGTH")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubSession) HEADERLENGTH() (*big.Int, error) {
	return _SequencerInboxStub.Contract.HEADERLENGTH(&_SequencerInboxStub.CallOpts)
}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) HEADERLENGTH() (*big.Int, error) {
	return _SequencerInboxStub.Contract.HEADERLENGTH(&_SequencerInboxStub.CallOpts)
}

// TREEDASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x6c890450.
//
// Solidity: function TREE_DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCaller) TREEDASMESSAGEHEADERFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "TREE_DAS_MESSAGE_HEADER_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// TREEDASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x6c890450.
//
// Solidity: function TREE_DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubSession) TREEDASMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.TREEDASMESSAGEHEADERFLAG(&_SequencerInboxStub.CallOpts)
}

// TREEDASMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x6c890450.
//
// Solidity: function TREE_DAS_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) TREEDASMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.TREEDASMESSAGEHEADERFLAG(&_SequencerInboxStub.CallOpts)
}

// ZEROHEAVYMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x02c99275.
//
// Solidity: function ZERO_HEAVY_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCaller) ZEROHEAVYMESSAGEHEADERFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "ZERO_HEAVY_MESSAGE_HEADER_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// ZEROHEAVYMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x02c99275.
//
// Solidity: function ZERO_HEAVY_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubSession) ZEROHEAVYMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.ZEROHEAVYMESSAGEHEADERFLAG(&_SequencerInboxStub.CallOpts)
}

// ZEROHEAVYMESSAGEHEADERFLAG is a free data retrieval call binding the contract method 0x02c99275.
//
// Solidity: function ZERO_HEAVY_MESSAGE_HEADER_FLAG() view returns(bytes1)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) ZEROHEAVYMESSAGEHEADERFLAG() ([1]byte, error) {
	return _SequencerInboxStub.Contract.ZEROHEAVYMESSAGEHEADERFLAG(&_SequencerInboxStub.CallOpts)
}

// AddSequencerL2BatchFromOrigin is a free data retrieval call binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 , bytes , uint256 , address ) pure returns()
func (_SequencerInboxStub *SequencerInboxStubCaller) AddSequencerL2BatchFromOrigin(opts *bind.CallOpts, arg0 *big.Int, arg1 []byte, arg2 *big.Int, arg3 common.Address) error {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "addSequencerL2BatchFromOrigin", arg0, arg1, arg2, arg3)

	if err != nil {
		return err
	}

	return err

}

// AddSequencerL2BatchFromOrigin is a free data retrieval call binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 , bytes , uint256 , address ) pure returns()
func (_SequencerInboxStub *SequencerInboxStubSession) AddSequencerL2BatchFromOrigin(arg0 *big.Int, arg1 []byte, arg2 *big.Int, arg3 common.Address) error {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromOrigin(&_SequencerInboxStub.CallOpts, arg0, arg1, arg2, arg3)
}

// AddSequencerL2BatchFromOrigin is a free data retrieval call binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 , bytes , uint256 , address ) pure returns()
func (_SequencerInboxStub *SequencerInboxStubCallerSession) AddSequencerL2BatchFromOrigin(arg0 *big.Int, arg1 []byte, arg2 *big.Int, arg3 common.Address) error {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromOrigin(&_SequencerInboxStub.CallOpts, arg0, arg1, arg2, arg3)
}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubCaller) BatchCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "batchCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubSession) BatchCount() (*big.Int, error) {
	return _SequencerInboxStub.Contract.BatchCount(&_SequencerInboxStub.CallOpts)
}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) BatchCount() (*big.Int, error) {
	return _SequencerInboxStub.Contract.BatchCount(&_SequencerInboxStub.CallOpts)
}

// BatchPosterManager is a free data retrieval call binding the contract method 0xcc2a1a0c.
//
// Solidity: function batchPosterManager() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubCaller) BatchPosterManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "batchPosterManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BatchPosterManager is a free data retrieval call binding the contract method 0xcc2a1a0c.
//
// Solidity: function batchPosterManager() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubSession) BatchPosterManager() (common.Address, error) {
	return _SequencerInboxStub.Contract.BatchPosterManager(&_SequencerInboxStub.CallOpts)
}

// BatchPosterManager is a free data retrieval call binding the contract method 0xcc2a1a0c.
//
// Solidity: function batchPosterManager() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) BatchPosterManager() (common.Address, error) {
	return _SequencerInboxStub.Contract.BatchPosterManager(&_SequencerInboxStub.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubSession) Bridge() (common.Address, error) {
	return _SequencerInboxStub.Contract.Bridge(&_SequencerInboxStub.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) Bridge() (common.Address, error) {
	return _SequencerInboxStub.Contract.Bridge(&_SequencerInboxStub.CallOpts)
}

// Buffer is a free data retrieval call binding the contract method 0xedaafe20.
//
// Solidity: function buffer() view returns(uint64 bufferBlocks, uint64 max, uint64 threshold, uint64 prevBlockNumber, uint64 replenishRateInBasis, uint64 prevSequencedBlockNumber)
func (_SequencerInboxStub *SequencerInboxStubCaller) Buffer(opts *bind.CallOpts) (struct {
	BufferBlocks             uint64
	Max                      uint64
	Threshold                uint64
	PrevBlockNumber          uint64
	ReplenishRateInBasis     uint64
	PrevSequencedBlockNumber uint64
}, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "buffer")

	outstruct := new(struct {
		BufferBlocks             uint64
		Max                      uint64
		Threshold                uint64
		PrevBlockNumber          uint64
		ReplenishRateInBasis     uint64
		PrevSequencedBlockNumber uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BufferBlocks = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.Max = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.Threshold = *abi.ConvertType(out[2], new(uint64)).(*uint64)
	outstruct.PrevBlockNumber = *abi.ConvertType(out[3], new(uint64)).(*uint64)
	outstruct.ReplenishRateInBasis = *abi.ConvertType(out[4], new(uint64)).(*uint64)
	outstruct.PrevSequencedBlockNumber = *abi.ConvertType(out[5], new(uint64)).(*uint64)

	return *outstruct, err

}

// Buffer is a free data retrieval call binding the contract method 0xedaafe20.
//
// Solidity: function buffer() view returns(uint64 bufferBlocks, uint64 max, uint64 threshold, uint64 prevBlockNumber, uint64 replenishRateInBasis, uint64 prevSequencedBlockNumber)
func (_SequencerInboxStub *SequencerInboxStubSession) Buffer() (struct {
	BufferBlocks             uint64
	Max                      uint64
	Threshold                uint64
	PrevBlockNumber          uint64
	ReplenishRateInBasis     uint64
	PrevSequencedBlockNumber uint64
}, error) {
	return _SequencerInboxStub.Contract.Buffer(&_SequencerInboxStub.CallOpts)
}

// Buffer is a free data retrieval call binding the contract method 0xedaafe20.
//
// Solidity: function buffer() view returns(uint64 bufferBlocks, uint64 max, uint64 threshold, uint64 prevBlockNumber, uint64 replenishRateInBasis, uint64 prevSequencedBlockNumber)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) Buffer() (struct {
	BufferBlocks             uint64
	Max                      uint64
	Threshold                uint64
	PrevBlockNumber          uint64
	ReplenishRateInBasis     uint64
	PrevSequencedBlockNumber uint64
}, error) {
	return _SequencerInboxStub.Contract.Buffer(&_SequencerInboxStub.CallOpts)
}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool isValidKeyset, uint64 creationBlock)
func (_SequencerInboxStub *SequencerInboxStubCaller) DasKeySetInfo(opts *bind.CallOpts, arg0 [32]byte) (struct {
	IsValidKeyset bool
	CreationBlock uint64
}, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "dasKeySetInfo", arg0)

	outstruct := new(struct {
		IsValidKeyset bool
		CreationBlock uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.IsValidKeyset = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.CreationBlock = *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return *outstruct, err

}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool isValidKeyset, uint64 creationBlock)
func (_SequencerInboxStub *SequencerInboxStubSession) DasKeySetInfo(arg0 [32]byte) (struct {
	IsValidKeyset bool
	CreationBlock uint64
}, error) {
	return _SequencerInboxStub.Contract.DasKeySetInfo(&_SequencerInboxStub.CallOpts, arg0)
}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool isValidKeyset, uint64 creationBlock)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) DasKeySetInfo(arg0 [32]byte) (struct {
	IsValidKeyset bool
	CreationBlock uint64
}, error) {
	return _SequencerInboxStub.Contract.DasKeySetInfo(&_SequencerInboxStub.CallOpts, arg0)
}

// ForceInclusionDeadline is a free data retrieval call binding the contract method 0xdd44e6e0.
//
// Solidity: function forceInclusionDeadline(uint64 blockNumber) view returns(uint64)
func (_SequencerInboxStub *SequencerInboxStubCaller) ForceInclusionDeadline(opts *bind.CallOpts, blockNumber uint64) (uint64, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "forceInclusionDeadline", blockNumber)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// ForceInclusionDeadline is a free data retrieval call binding the contract method 0xdd44e6e0.
//
// Solidity: function forceInclusionDeadline(uint64 blockNumber) view returns(uint64)
func (_SequencerInboxStub *SequencerInboxStubSession) ForceInclusionDeadline(blockNumber uint64) (uint64, error) {
	return _SequencerInboxStub.Contract.ForceInclusionDeadline(&_SequencerInboxStub.CallOpts, blockNumber)
}

// ForceInclusionDeadline is a free data retrieval call binding the contract method 0xdd44e6e0.
//
// Solidity: function forceInclusionDeadline(uint64 blockNumber) view returns(uint64)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) ForceInclusionDeadline(blockNumber uint64) (uint64, error) {
	return _SequencerInboxStub.Contract.ForceInclusionDeadline(&_SequencerInboxStub.CallOpts, blockNumber)
}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubCaller) GetKeysetCreationBlock(opts *bind.CallOpts, ksHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "getKeysetCreationBlock", ksHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubSession) GetKeysetCreationBlock(ksHash [32]byte) (*big.Int, error) {
	return _SequencerInboxStub.Contract.GetKeysetCreationBlock(&_SequencerInboxStub.CallOpts, ksHash)
}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) GetKeysetCreationBlock(ksHash [32]byte) (*big.Int, error) {
	return _SequencerInboxStub.Contract.GetKeysetCreationBlock(&_SequencerInboxStub.CallOpts, ksHash)
}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_SequencerInboxStub *SequencerInboxStubCaller) InboxAccs(opts *bind.CallOpts, index *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "inboxAccs", index)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_SequencerInboxStub *SequencerInboxStubSession) InboxAccs(index *big.Int) ([32]byte, error) {
	return _SequencerInboxStub.Contract.InboxAccs(&_SequencerInboxStub.CallOpts, index)
}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) InboxAccs(index *big.Int) ([32]byte, error) {
	return _SequencerInboxStub.Contract.InboxAccs(&_SequencerInboxStub.CallOpts, index)
}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubCaller) IsBatchPoster(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "isBatchPoster", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubSession) IsBatchPoster(arg0 common.Address) (bool, error) {
	return _SequencerInboxStub.Contract.IsBatchPoster(&_SequencerInboxStub.CallOpts, arg0)
}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) IsBatchPoster(arg0 common.Address) (bool, error) {
	return _SequencerInboxStub.Contract.IsBatchPoster(&_SequencerInboxStub.CallOpts, arg0)
}

// IsDelayBufferable is a free data retrieval call binding the contract method 0x4b678a66.
//
// Solidity: function isDelayBufferable() view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubCaller) IsDelayBufferable(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "isDelayBufferable")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsDelayBufferable is a free data retrieval call binding the contract method 0x4b678a66.
//
// Solidity: function isDelayBufferable() view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubSession) IsDelayBufferable() (bool, error) {
	return _SequencerInboxStub.Contract.IsDelayBufferable(&_SequencerInboxStub.CallOpts)
}

// IsDelayBufferable is a free data retrieval call binding the contract method 0x4b678a66.
//
// Solidity: function isDelayBufferable() view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) IsDelayBufferable() (bool, error) {
	return _SequencerInboxStub.Contract.IsDelayBufferable(&_SequencerInboxStub.CallOpts)
}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubCaller) IsSequencer(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "isSequencer", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubSession) IsSequencer(arg0 common.Address) (bool, error) {
	return _SequencerInboxStub.Contract.IsSequencer(&_SequencerInboxStub.CallOpts, arg0)
}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) IsSequencer(arg0 common.Address) (bool, error) {
	return _SequencerInboxStub.Contract.IsSequencer(&_SequencerInboxStub.CallOpts, arg0)
}

// IsUsingFeeToken is a free data retrieval call binding the contract method 0x92d9f782.
//
// Solidity: function isUsingFeeToken() view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubCaller) IsUsingFeeToken(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "isUsingFeeToken")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsUsingFeeToken is a free data retrieval call binding the contract method 0x92d9f782.
//
// Solidity: function isUsingFeeToken() view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubSession) IsUsingFeeToken() (bool, error) {
	return _SequencerInboxStub.Contract.IsUsingFeeToken(&_SequencerInboxStub.CallOpts)
}

// IsUsingFeeToken is a free data retrieval call binding the contract method 0x92d9f782.
//
// Solidity: function isUsingFeeToken() view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) IsUsingFeeToken() (bool, error) {
	return _SequencerInboxStub.Contract.IsUsingFeeToken(&_SequencerInboxStub.CallOpts)
}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubCaller) IsValidKeysetHash(opts *bind.CallOpts, ksHash [32]byte) (bool, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "isValidKeysetHash", ksHash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubSession) IsValidKeysetHash(ksHash [32]byte) (bool, error) {
	return _SequencerInboxStub.Contract.IsValidKeysetHash(&_SequencerInboxStub.CallOpts, ksHash)
}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) IsValidKeysetHash(ksHash [32]byte) (bool, error) {
	return _SequencerInboxStub.Contract.IsValidKeysetHash(&_SequencerInboxStub.CallOpts, ksHash)
}

// MaxDataSize is a free data retrieval call binding the contract method 0xe8eb1dc3.
//
// Solidity: function maxDataSize() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubCaller) MaxDataSize(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "maxDataSize")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxDataSize is a free data retrieval call binding the contract method 0xe8eb1dc3.
//
// Solidity: function maxDataSize() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubSession) MaxDataSize() (*big.Int, error) {
	return _SequencerInboxStub.Contract.MaxDataSize(&_SequencerInboxStub.CallOpts)
}

// MaxDataSize is a free data retrieval call binding the contract method 0xe8eb1dc3.
//
// Solidity: function maxDataSize() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) MaxDataSize() (*big.Int, error) {
	return _SequencerInboxStub.Contract.MaxDataSize(&_SequencerInboxStub.CallOpts)
}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256, uint256, uint256, uint256)
func (_SequencerInboxStub *SequencerInboxStubCaller) MaxTimeVariation(opts *bind.CallOpts) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "maxTimeVariation")

	if err != nil {
		return *new(*big.Int), *new(*big.Int), *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	out3 := *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return out0, out1, out2, out3, err

}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256, uint256, uint256, uint256)
func (_SequencerInboxStub *SequencerInboxStubSession) MaxTimeVariation() (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _SequencerInboxStub.Contract.MaxTimeVariation(&_SequencerInboxStub.CallOpts)
}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256, uint256, uint256, uint256)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) MaxTimeVariation() (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _SequencerInboxStub.Contract.MaxTimeVariation(&_SequencerInboxStub.CallOpts)
}

// Reader4844 is a free data retrieval call binding the contract method 0x8d910dde.
//
// Solidity: function reader4844() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubCaller) Reader4844(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "reader4844")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Reader4844 is a free data retrieval call binding the contract method 0x8d910dde.
//
// Solidity: function reader4844() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubSession) Reader4844() (common.Address, error) {
	return _SequencerInboxStub.Contract.Reader4844(&_SequencerInboxStub.CallOpts)
}

// Reader4844 is a free data retrieval call binding the contract method 0x8d910dde.
//
// Solidity: function reader4844() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) Reader4844() (common.Address, error) {
	return _SequencerInboxStub.Contract.Reader4844(&_SequencerInboxStub.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubSession) Rollup() (common.Address, error) {
	return _SequencerInboxStub.Contract.Rollup(&_SequencerInboxStub.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) Rollup() (common.Address, error) {
	return _SequencerInboxStub.Contract.Rollup(&_SequencerInboxStub.CallOpts)
}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubCaller) TotalDelayedMessagesRead(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "totalDelayedMessagesRead")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubSession) TotalDelayedMessagesRead() (*big.Int, error) {
	return _SequencerInboxStub.Contract.TotalDelayedMessagesRead(&_SequencerInboxStub.CallOpts)
}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) TotalDelayedMessagesRead() (*big.Int, error) {
	return _SequencerInboxStub.Contract.TotalDelayedMessagesRead(&_SequencerInboxStub.CallOpts)
}

// AddInitMessage is a paid mutator transaction binding the contract method 0x6633ae85.
//
// Solidity: function addInitMessage(uint256 chainId) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) AddInitMessage(opts *bind.TransactOpts, chainId *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "addInitMessage", chainId)
}

// AddInitMessage is a paid mutator transaction binding the contract method 0x6633ae85.
//
// Solidity: function addInitMessage(uint256 chainId) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) AddInitMessage(chainId *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddInitMessage(&_SequencerInboxStub.TransactOpts, chainId)
}

// AddInitMessage is a paid mutator transaction binding the contract method 0x6633ae85.
//
// Solidity: function addInitMessage(uint256 chainId) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) AddInitMessage(chainId *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddInitMessage(&_SequencerInboxStub.TransactOpts, chainId)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) AddSequencerL2Batch(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "addSequencerL2Batch", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) AddSequencerL2Batch(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2Batch(&_SequencerInboxStub.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) AddSequencerL2Batch(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2Batch(&_SequencerInboxStub.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchDelayProof is a paid mutator transaction binding the contract method 0x6e620055.
//
// Solidity: function addSequencerL2BatchDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) AddSequencerL2BatchDelayProof(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "addSequencerL2BatchDelayProof", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchDelayProof is a paid mutator transaction binding the contract method 0x6e620055.
//
// Solidity: function addSequencerL2BatchDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) AddSequencerL2BatchDelayProof(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchDelayProof(&_SequencerInboxStub.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchDelayProof is a paid mutator transaction binding the contract method 0x6e620055.
//
// Solidity: function addSequencerL2BatchDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) AddSequencerL2BatchDelayProof(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchDelayProof(&_SequencerInboxStub.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromBlobs is a paid mutator transaction binding the contract method 0x3e5aa082.
//
// Solidity: function addSequencerL2BatchFromBlobs(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) AddSequencerL2BatchFromBlobs(opts *bind.TransactOpts, sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "addSequencerL2BatchFromBlobs", sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromBlobs is a paid mutator transaction binding the contract method 0x3e5aa082.
//
// Solidity: function addSequencerL2BatchFromBlobs(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) AddSequencerL2BatchFromBlobs(sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromBlobs(&_SequencerInboxStub.TransactOpts, sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromBlobs is a paid mutator transaction binding the contract method 0x3e5aa082.
//
// Solidity: function addSequencerL2BatchFromBlobs(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) AddSequencerL2BatchFromBlobs(sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromBlobs(&_SequencerInboxStub.TransactOpts, sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromBlobsDelayProof is a paid mutator transaction binding the contract method 0x917cf8ac.
//
// Solidity: function addSequencerL2BatchFromBlobsDelayProof(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) AddSequencerL2BatchFromBlobsDelayProof(opts *bind.TransactOpts, sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "addSequencerL2BatchFromBlobsDelayProof", sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromBlobsDelayProof is a paid mutator transaction binding the contract method 0x917cf8ac.
//
// Solidity: function addSequencerL2BatchFromBlobsDelayProof(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) AddSequencerL2BatchFromBlobsDelayProof(sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromBlobsDelayProof(&_SequencerInboxStub.TransactOpts, sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromBlobsDelayProof is a paid mutator transaction binding the contract method 0x917cf8ac.
//
// Solidity: function addSequencerL2BatchFromBlobsDelayProof(uint256 sequenceNumber, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) AddSequencerL2BatchFromBlobsDelayProof(sequenceNumber *big.Int, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromBlobsDelayProof(&_SequencerInboxStub.TransactOpts, sequenceNumber, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromOrigin0 is a paid mutator transaction binding the contract method 0x8f111f3c.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) AddSequencerL2BatchFromOrigin0(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "addSequencerL2BatchFromOrigin0", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromOrigin0 is a paid mutator transaction binding the contract method 0x8f111f3c.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) AddSequencerL2BatchFromOrigin0(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromOrigin0(&_SequencerInboxStub.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromOrigin0 is a paid mutator transaction binding the contract method 0x8f111f3c.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) AddSequencerL2BatchFromOrigin0(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromOrigin0(&_SequencerInboxStub.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromOriginDelayProof is a paid mutator transaction binding the contract method 0x69cacded.
//
// Solidity: function addSequencerL2BatchFromOriginDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) AddSequencerL2BatchFromOriginDelayProof(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "addSequencerL2BatchFromOriginDelayProof", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromOriginDelayProof is a paid mutator transaction binding the contract method 0x69cacded.
//
// Solidity: function addSequencerL2BatchFromOriginDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) AddSequencerL2BatchFromOriginDelayProof(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromOriginDelayProof(&_SequencerInboxStub.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// AddSequencerL2BatchFromOriginDelayProof is a paid mutator transaction binding the contract method 0x69cacded.
//
// Solidity: function addSequencerL2BatchFromOriginDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, (bytes32,(uint8,address,uint64,uint64,uint256,uint256,bytes32)) delayProof) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) AddSequencerL2BatchFromOriginDelayProof(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int, delayProof DelayProof) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromOriginDelayProof(&_SequencerInboxStub.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount, delayProof)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) ForceInclusion(opts *bind.TransactOpts, _totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "forceInclusion", _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) ForceInclusion(_totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.ForceInclusion(&_SequencerInboxStub.TransactOpts, _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) ForceInclusion(_totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.ForceInclusion(&_SequencerInboxStub.TransactOpts, _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// Initialize is a paid mutator transaction binding the contract method 0x1ad87e45.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_, (uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) Initialize(opts *bind.TransactOpts, bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation, bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "initialize", bridge_, maxTimeVariation_, bufferConfig_)
}

// Initialize is a paid mutator transaction binding the contract method 0x1ad87e45.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_, (uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) Initialize(bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation, bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.Initialize(&_SequencerInboxStub.TransactOpts, bridge_, maxTimeVariation_, bufferConfig_)
}

// Initialize is a paid mutator transaction binding the contract method 0x1ad87e45.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_, (uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) Initialize(bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation, bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.Initialize(&_SequencerInboxStub.TransactOpts, bridge_, maxTimeVariation_, bufferConfig_)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) InvalidateKeysetHash(opts *bind.TransactOpts, ksHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "invalidateKeysetHash", ksHash)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) InvalidateKeysetHash(ksHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.InvalidateKeysetHash(&_SequencerInboxStub.TransactOpts, ksHash)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) InvalidateKeysetHash(ksHash [32]byte) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.InvalidateKeysetHash(&_SequencerInboxStub.TransactOpts, ksHash)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xa655d937.
//
// Solidity: function postUpgradeInit((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) PostUpgradeInit(opts *bind.TransactOpts, bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "postUpgradeInit", bufferConfig_)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xa655d937.
//
// Solidity: function postUpgradeInit((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) PostUpgradeInit(bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.PostUpgradeInit(&_SequencerInboxStub.TransactOpts, bufferConfig_)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xa655d937.
//
// Solidity: function postUpgradeInit((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) PostUpgradeInit(bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.PostUpgradeInit(&_SequencerInboxStub.TransactOpts, bufferConfig_)
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) RemoveDelayAfterFork(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "removeDelayAfterFork")
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_SequencerInboxStub *SequencerInboxStubSession) RemoveDelayAfterFork() (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.RemoveDelayAfterFork(&_SequencerInboxStub.TransactOpts)
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) RemoveDelayAfterFork() (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.RemoveDelayAfterFork(&_SequencerInboxStub.TransactOpts)
}

// SetBatchPosterManager is a paid mutator transaction binding the contract method 0x1ff64790.
//
// Solidity: function setBatchPosterManager(address newBatchPosterManager) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) SetBatchPosterManager(opts *bind.TransactOpts, newBatchPosterManager common.Address) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "setBatchPosterManager", newBatchPosterManager)
}

// SetBatchPosterManager is a paid mutator transaction binding the contract method 0x1ff64790.
//
// Solidity: function setBatchPosterManager(address newBatchPosterManager) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) SetBatchPosterManager(newBatchPosterManager common.Address) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetBatchPosterManager(&_SequencerInboxStub.TransactOpts, newBatchPosterManager)
}

// SetBatchPosterManager is a paid mutator transaction binding the contract method 0x1ff64790.
//
// Solidity: function setBatchPosterManager(address newBatchPosterManager) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) SetBatchPosterManager(newBatchPosterManager common.Address) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetBatchPosterManager(&_SequencerInboxStub.TransactOpts, newBatchPosterManager)
}

// SetBufferConfig is a paid mutator transaction binding the contract method 0x2f3985a7.
//
// Solidity: function setBufferConfig((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) SetBufferConfig(opts *bind.TransactOpts, bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "setBufferConfig", bufferConfig_)
}

// SetBufferConfig is a paid mutator transaction binding the contract method 0x2f3985a7.
//
// Solidity: function setBufferConfig((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) SetBufferConfig(bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetBufferConfig(&_SequencerInboxStub.TransactOpts, bufferConfig_)
}

// SetBufferConfig is a paid mutator transaction binding the contract method 0x2f3985a7.
//
// Solidity: function setBufferConfig((uint64,uint64,uint64) bufferConfig_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) SetBufferConfig(bufferConfig_ BufferConfig) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetBufferConfig(&_SequencerInboxStub.TransactOpts, bufferConfig_)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) SetIsBatchPoster(opts *bind.TransactOpts, addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "setIsBatchPoster", addr, isBatchPoster_)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) SetIsBatchPoster(addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetIsBatchPoster(&_SequencerInboxStub.TransactOpts, addr, isBatchPoster_)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) SetIsBatchPoster(addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetIsBatchPoster(&_SequencerInboxStub.TransactOpts, addr, isBatchPoster_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) SetIsSequencer(opts *bind.TransactOpts, addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "setIsSequencer", addr, isSequencer_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) SetIsSequencer(addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetIsSequencer(&_SequencerInboxStub.TransactOpts, addr, isSequencer_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) SetIsSequencer(addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetIsSequencer(&_SequencerInboxStub.TransactOpts, addr, isSequencer_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) SetMaxTimeVariation(opts *bind.TransactOpts, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "setMaxTimeVariation", maxTimeVariation_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) SetMaxTimeVariation(maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetMaxTimeVariation(&_SequencerInboxStub.TransactOpts, maxTimeVariation_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) SetMaxTimeVariation(maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetMaxTimeVariation(&_SequencerInboxStub.TransactOpts, maxTimeVariation_)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) SetValidKeyset(opts *bind.TransactOpts, keysetBytes []byte) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "setValidKeyset", keysetBytes)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) SetValidKeyset(keysetBytes []byte) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetValidKeyset(&_SequencerInboxStub.TransactOpts, keysetBytes)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) SetValidKeyset(keysetBytes []byte) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.SetValidKeyset(&_SequencerInboxStub.TransactOpts, keysetBytes)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) UpdateRollupAddress(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "updateRollupAddress")
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_SequencerInboxStub *SequencerInboxStubSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.UpdateRollupAddress(&_SequencerInboxStub.TransactOpts)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.UpdateRollupAddress(&_SequencerInboxStub.TransactOpts)
}

// SequencerInboxStubInboxMessageDeliveredIterator is returned from FilterInboxMessageDelivered and is used to iterate over the raw logs and unpacked data for InboxMessageDelivered events raised by the SequencerInboxStub contract.
type SequencerInboxStubInboxMessageDeliveredIterator struct {
	Event *SequencerInboxStubInboxMessageDelivered // Event containing the contract specifics and raw log

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
func (it *SequencerInboxStubInboxMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxStubInboxMessageDelivered)
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
		it.Event = new(SequencerInboxStubInboxMessageDelivered)
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
func (it *SequencerInboxStubInboxMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxStubInboxMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxStubInboxMessageDelivered represents a InboxMessageDelivered event raised by the SequencerInboxStub contract.
type SequencerInboxStubInboxMessageDelivered struct {
	MessageNum *big.Int
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDelivered is a free log retrieval operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_SequencerInboxStub *SequencerInboxStubFilterer) FilterInboxMessageDelivered(opts *bind.FilterOpts, messageNum []*big.Int) (*SequencerInboxStubInboxMessageDeliveredIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.FilterLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStubInboxMessageDeliveredIterator{contract: _SequencerInboxStub.contract, event: "InboxMessageDelivered", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDelivered is a free log subscription operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_SequencerInboxStub *SequencerInboxStubFilterer) WatchInboxMessageDelivered(opts *bind.WatchOpts, sink chan<- *SequencerInboxStubInboxMessageDelivered, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.WatchLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxStubInboxMessageDelivered)
				if err := _SequencerInboxStub.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
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

// ParseInboxMessageDelivered is a log parse operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_SequencerInboxStub *SequencerInboxStubFilterer) ParseInboxMessageDelivered(log types.Log) (*SequencerInboxStubInboxMessageDelivered, error) {
	event := new(SequencerInboxStubInboxMessageDelivered)
	if err := _SequencerInboxStub.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxStubInboxMessageDeliveredFromOriginIterator is returned from FilterInboxMessageDeliveredFromOrigin and is used to iterate over the raw logs and unpacked data for InboxMessageDeliveredFromOrigin events raised by the SequencerInboxStub contract.
type SequencerInboxStubInboxMessageDeliveredFromOriginIterator struct {
	Event *SequencerInboxStubInboxMessageDeliveredFromOrigin // Event containing the contract specifics and raw log

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
func (it *SequencerInboxStubInboxMessageDeliveredFromOriginIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxStubInboxMessageDeliveredFromOrigin)
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
		it.Event = new(SequencerInboxStubInboxMessageDeliveredFromOrigin)
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
func (it *SequencerInboxStubInboxMessageDeliveredFromOriginIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxStubInboxMessageDeliveredFromOriginIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxStubInboxMessageDeliveredFromOrigin represents a InboxMessageDeliveredFromOrigin event raised by the SequencerInboxStub contract.
type SequencerInboxStubInboxMessageDeliveredFromOrigin struct {
	MessageNum *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDeliveredFromOrigin is a free log retrieval operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_SequencerInboxStub *SequencerInboxStubFilterer) FilterInboxMessageDeliveredFromOrigin(opts *bind.FilterOpts, messageNum []*big.Int) (*SequencerInboxStubInboxMessageDeliveredFromOriginIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.FilterLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStubInboxMessageDeliveredFromOriginIterator{contract: _SequencerInboxStub.contract, event: "InboxMessageDeliveredFromOrigin", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDeliveredFromOrigin is a free log subscription operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_SequencerInboxStub *SequencerInboxStubFilterer) WatchInboxMessageDeliveredFromOrigin(opts *bind.WatchOpts, sink chan<- *SequencerInboxStubInboxMessageDeliveredFromOrigin, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.WatchLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxStubInboxMessageDeliveredFromOrigin)
				if err := _SequencerInboxStub.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
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

// ParseInboxMessageDeliveredFromOrigin is a log parse operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_SequencerInboxStub *SequencerInboxStubFilterer) ParseInboxMessageDeliveredFromOrigin(log types.Log) (*SequencerInboxStubInboxMessageDeliveredFromOrigin, error) {
	event := new(SequencerInboxStubInboxMessageDeliveredFromOrigin)
	if err := _SequencerInboxStub.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxStubInvalidateKeysetIterator is returned from FilterInvalidateKeyset and is used to iterate over the raw logs and unpacked data for InvalidateKeyset events raised by the SequencerInboxStub contract.
type SequencerInboxStubInvalidateKeysetIterator struct {
	Event *SequencerInboxStubInvalidateKeyset // Event containing the contract specifics and raw log

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
func (it *SequencerInboxStubInvalidateKeysetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxStubInvalidateKeyset)
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
		it.Event = new(SequencerInboxStubInvalidateKeyset)
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
func (it *SequencerInboxStubInvalidateKeysetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxStubInvalidateKeysetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxStubInvalidateKeyset represents a InvalidateKeyset event raised by the SequencerInboxStub contract.
type SequencerInboxStubInvalidateKeyset struct {
	KeysetHash [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInvalidateKeyset is a free log retrieval operation binding the contract event 0x5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a.
//
// Solidity: event InvalidateKeyset(bytes32 indexed keysetHash)
func (_SequencerInboxStub *SequencerInboxStubFilterer) FilterInvalidateKeyset(opts *bind.FilterOpts, keysetHash [][32]byte) (*SequencerInboxStubInvalidateKeysetIterator, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.FilterLogs(opts, "InvalidateKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStubInvalidateKeysetIterator{contract: _SequencerInboxStub.contract, event: "InvalidateKeyset", logs: logs, sub: sub}, nil
}

// WatchInvalidateKeyset is a free log subscription operation binding the contract event 0x5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a.
//
// Solidity: event InvalidateKeyset(bytes32 indexed keysetHash)
func (_SequencerInboxStub *SequencerInboxStubFilterer) WatchInvalidateKeyset(opts *bind.WatchOpts, sink chan<- *SequencerInboxStubInvalidateKeyset, keysetHash [][32]byte) (event.Subscription, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.WatchLogs(opts, "InvalidateKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxStubInvalidateKeyset)
				if err := _SequencerInboxStub.contract.UnpackLog(event, "InvalidateKeyset", log); err != nil {
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

// ParseInvalidateKeyset is a log parse operation binding the contract event 0x5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a.
//
// Solidity: event InvalidateKeyset(bytes32 indexed keysetHash)
func (_SequencerInboxStub *SequencerInboxStubFilterer) ParseInvalidateKeyset(log types.Log) (*SequencerInboxStubInvalidateKeyset, error) {
	event := new(SequencerInboxStubInvalidateKeyset)
	if err := _SequencerInboxStub.contract.UnpackLog(event, "InvalidateKeyset", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxStubOwnerFunctionCalledIterator is returned from FilterOwnerFunctionCalled and is used to iterate over the raw logs and unpacked data for OwnerFunctionCalled events raised by the SequencerInboxStub contract.
type SequencerInboxStubOwnerFunctionCalledIterator struct {
	Event *SequencerInboxStubOwnerFunctionCalled // Event containing the contract specifics and raw log

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
func (it *SequencerInboxStubOwnerFunctionCalledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxStubOwnerFunctionCalled)
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
		it.Event = new(SequencerInboxStubOwnerFunctionCalled)
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
func (it *SequencerInboxStubOwnerFunctionCalledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxStubOwnerFunctionCalledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxStubOwnerFunctionCalled represents a OwnerFunctionCalled event raised by the SequencerInboxStub contract.
type SequencerInboxStubOwnerFunctionCalled struct {
	Id  *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterOwnerFunctionCalled is a free log retrieval operation binding the contract event 0xea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e.
//
// Solidity: event OwnerFunctionCalled(uint256 indexed id)
func (_SequencerInboxStub *SequencerInboxStubFilterer) FilterOwnerFunctionCalled(opts *bind.FilterOpts, id []*big.Int) (*SequencerInboxStubOwnerFunctionCalledIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.FilterLogs(opts, "OwnerFunctionCalled", idRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStubOwnerFunctionCalledIterator{contract: _SequencerInboxStub.contract, event: "OwnerFunctionCalled", logs: logs, sub: sub}, nil
}

// WatchOwnerFunctionCalled is a free log subscription operation binding the contract event 0xea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e.
//
// Solidity: event OwnerFunctionCalled(uint256 indexed id)
func (_SequencerInboxStub *SequencerInboxStubFilterer) WatchOwnerFunctionCalled(opts *bind.WatchOpts, sink chan<- *SequencerInboxStubOwnerFunctionCalled, id []*big.Int) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.WatchLogs(opts, "OwnerFunctionCalled", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxStubOwnerFunctionCalled)
				if err := _SequencerInboxStub.contract.UnpackLog(event, "OwnerFunctionCalled", log); err != nil {
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

// ParseOwnerFunctionCalled is a log parse operation binding the contract event 0xea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e.
//
// Solidity: event OwnerFunctionCalled(uint256 indexed id)
func (_SequencerInboxStub *SequencerInboxStubFilterer) ParseOwnerFunctionCalled(log types.Log) (*SequencerInboxStubOwnerFunctionCalled, error) {
	event := new(SequencerInboxStubOwnerFunctionCalled)
	if err := _SequencerInboxStub.contract.UnpackLog(event, "OwnerFunctionCalled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxStubSequencerBatchDataIterator is returned from FilterSequencerBatchData and is used to iterate over the raw logs and unpacked data for SequencerBatchData events raised by the SequencerInboxStub contract.
type SequencerInboxStubSequencerBatchDataIterator struct {
	Event *SequencerInboxStubSequencerBatchData // Event containing the contract specifics and raw log

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
func (it *SequencerInboxStubSequencerBatchDataIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxStubSequencerBatchData)
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
		it.Event = new(SequencerInboxStubSequencerBatchData)
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
func (it *SequencerInboxStubSequencerBatchDataIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxStubSequencerBatchDataIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxStubSequencerBatchData represents a SequencerBatchData event raised by the SequencerInboxStub contract.
type SequencerInboxStubSequencerBatchData struct {
	BatchSequenceNumber *big.Int
	Data                []byte
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterSequencerBatchData is a free log retrieval operation binding the contract event 0xfe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c20.
//
// Solidity: event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data)
func (_SequencerInboxStub *SequencerInboxStubFilterer) FilterSequencerBatchData(opts *bind.FilterOpts, batchSequenceNumber []*big.Int) (*SequencerInboxStubSequencerBatchDataIterator, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.FilterLogs(opts, "SequencerBatchData", batchSequenceNumberRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStubSequencerBatchDataIterator{contract: _SequencerInboxStub.contract, event: "SequencerBatchData", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchData is a free log subscription operation binding the contract event 0xfe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c20.
//
// Solidity: event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data)
func (_SequencerInboxStub *SequencerInboxStubFilterer) WatchSequencerBatchData(opts *bind.WatchOpts, sink chan<- *SequencerInboxStubSequencerBatchData, batchSequenceNumber []*big.Int) (event.Subscription, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.WatchLogs(opts, "SequencerBatchData", batchSequenceNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxStubSequencerBatchData)
				if err := _SequencerInboxStub.contract.UnpackLog(event, "SequencerBatchData", log); err != nil {
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

// ParseSequencerBatchData is a log parse operation binding the contract event 0xfe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c20.
//
// Solidity: event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data)
func (_SequencerInboxStub *SequencerInboxStubFilterer) ParseSequencerBatchData(log types.Log) (*SequencerInboxStubSequencerBatchData, error) {
	event := new(SequencerInboxStubSequencerBatchData)
	if err := _SequencerInboxStub.contract.UnpackLog(event, "SequencerBatchData", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxStubSequencerBatchDeliveredIterator is returned from FilterSequencerBatchDelivered and is used to iterate over the raw logs and unpacked data for SequencerBatchDelivered events raised by the SequencerInboxStub contract.
type SequencerInboxStubSequencerBatchDeliveredIterator struct {
	Event *SequencerInboxStubSequencerBatchDelivered // Event containing the contract specifics and raw log

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
func (it *SequencerInboxStubSequencerBatchDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxStubSequencerBatchDelivered)
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
		it.Event = new(SequencerInboxStubSequencerBatchDelivered)
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
func (it *SequencerInboxStubSequencerBatchDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxStubSequencerBatchDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxStubSequencerBatchDelivered represents a SequencerBatchDelivered event raised by the SequencerInboxStub contract.
type SequencerInboxStubSequencerBatchDelivered struct {
	BatchSequenceNumber      *big.Int
	BeforeAcc                [32]byte
	AfterAcc                 [32]byte
	DelayedAcc               [32]byte
	AfterDelayedMessagesRead *big.Int
	TimeBounds               IBridgeTimeBounds
	DataLocation             uint8
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterSequencerBatchDelivered is a free log retrieval operation binding the contract event 0x7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7.
//
// Solidity: event SequencerBatchDelivered(uint256 indexed batchSequenceNumber, bytes32 indexed beforeAcc, bytes32 indexed afterAcc, bytes32 delayedAcc, uint256 afterDelayedMessagesRead, (uint64,uint64,uint64,uint64) timeBounds, uint8 dataLocation)
func (_SequencerInboxStub *SequencerInboxStubFilterer) FilterSequencerBatchDelivered(opts *bind.FilterOpts, batchSequenceNumber []*big.Int, beforeAcc [][32]byte, afterAcc [][32]byte) (*SequencerInboxStubSequencerBatchDeliveredIterator, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}
	var beforeAccRule []interface{}
	for _, beforeAccItem := range beforeAcc {
		beforeAccRule = append(beforeAccRule, beforeAccItem)
	}
	var afterAccRule []interface{}
	for _, afterAccItem := range afterAcc {
		afterAccRule = append(afterAccRule, afterAccItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.FilterLogs(opts, "SequencerBatchDelivered", batchSequenceNumberRule, beforeAccRule, afterAccRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStubSequencerBatchDeliveredIterator{contract: _SequencerInboxStub.contract, event: "SequencerBatchDelivered", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchDelivered is a free log subscription operation binding the contract event 0x7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7.
//
// Solidity: event SequencerBatchDelivered(uint256 indexed batchSequenceNumber, bytes32 indexed beforeAcc, bytes32 indexed afterAcc, bytes32 delayedAcc, uint256 afterDelayedMessagesRead, (uint64,uint64,uint64,uint64) timeBounds, uint8 dataLocation)
func (_SequencerInboxStub *SequencerInboxStubFilterer) WatchSequencerBatchDelivered(opts *bind.WatchOpts, sink chan<- *SequencerInboxStubSequencerBatchDelivered, batchSequenceNumber []*big.Int, beforeAcc [][32]byte, afterAcc [][32]byte) (event.Subscription, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}
	var beforeAccRule []interface{}
	for _, beforeAccItem := range beforeAcc {
		beforeAccRule = append(beforeAccRule, beforeAccItem)
	}
	var afterAccRule []interface{}
	for _, afterAccItem := range afterAcc {
		afterAccRule = append(afterAccRule, afterAccItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.WatchLogs(opts, "SequencerBatchDelivered", batchSequenceNumberRule, beforeAccRule, afterAccRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxStubSequencerBatchDelivered)
				if err := _SequencerInboxStub.contract.UnpackLog(event, "SequencerBatchDelivered", log); err != nil {
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

// ParseSequencerBatchDelivered is a log parse operation binding the contract event 0x7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7.
//
// Solidity: event SequencerBatchDelivered(uint256 indexed batchSequenceNumber, bytes32 indexed beforeAcc, bytes32 indexed afterAcc, bytes32 delayedAcc, uint256 afterDelayedMessagesRead, (uint64,uint64,uint64,uint64) timeBounds, uint8 dataLocation)
func (_SequencerInboxStub *SequencerInboxStubFilterer) ParseSequencerBatchDelivered(log types.Log) (*SequencerInboxStubSequencerBatchDelivered, error) {
	event := new(SequencerInboxStubSequencerBatchDelivered)
	if err := _SequencerInboxStub.contract.UnpackLog(event, "SequencerBatchDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxStubSetValidKeysetIterator is returned from FilterSetValidKeyset and is used to iterate over the raw logs and unpacked data for SetValidKeyset events raised by the SequencerInboxStub contract.
type SequencerInboxStubSetValidKeysetIterator struct {
	Event *SequencerInboxStubSetValidKeyset // Event containing the contract specifics and raw log

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
func (it *SequencerInboxStubSetValidKeysetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxStubSetValidKeyset)
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
		it.Event = new(SequencerInboxStubSetValidKeyset)
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
func (it *SequencerInboxStubSetValidKeysetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxStubSetValidKeysetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxStubSetValidKeyset represents a SetValidKeyset event raised by the SequencerInboxStub contract.
type SequencerInboxStubSetValidKeyset struct {
	KeysetHash  [32]byte
	KeysetBytes []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSetValidKeyset is a free log retrieval operation binding the contract event 0xabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722.
//
// Solidity: event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes)
func (_SequencerInboxStub *SequencerInboxStubFilterer) FilterSetValidKeyset(opts *bind.FilterOpts, keysetHash [][32]byte) (*SequencerInboxStubSetValidKeysetIterator, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.FilterLogs(opts, "SetValidKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxStubSetValidKeysetIterator{contract: _SequencerInboxStub.contract, event: "SetValidKeyset", logs: logs, sub: sub}, nil
}

// WatchSetValidKeyset is a free log subscription operation binding the contract event 0xabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722.
//
// Solidity: event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes)
func (_SequencerInboxStub *SequencerInboxStubFilterer) WatchSetValidKeyset(opts *bind.WatchOpts, sink chan<- *SequencerInboxStubSetValidKeyset, keysetHash [][32]byte) (event.Subscription, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInboxStub.contract.WatchLogs(opts, "SetValidKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxStubSetValidKeyset)
				if err := _SequencerInboxStub.contract.UnpackLog(event, "SetValidKeyset", log); err != nil {
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

// ParseSetValidKeyset is a log parse operation binding the contract event 0xabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722.
//
// Solidity: event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes)
func (_SequencerInboxStub *SequencerInboxStubFilterer) ParseSetValidKeyset(log types.Log) (*SequencerInboxStubSetValidKeyset, error) {
	event := new(SequencerInboxStubSetValidKeyset)
	if err := _SequencerInboxStub.contract.UnpackLog(event, "SetValidKeyset", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SimpleMetaData contains all meta data concerning the Simple contract.
var SimpleMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"count\",\"type\":\"uint64\"}],\"name\":\"CounterEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"expected\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"have\",\"type\":\"uint256\"}],\"name\":\"LogAndIncrementCalled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"NullEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"redeemer\",\"type\":\"address\"}],\"name\":\"RedeemedEvent\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"checkBlockHashes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"useTopLevel\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"directCase\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"staticCase\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"delegateCase\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"callcodeCase\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"callCase\",\"type\":\"bool\"}],\"name\":\"checkCalls\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"input\",\"type\":\"bytes\"}],\"name\":\"checkGasUsed\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"useTopLevel\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"expected\",\"type\":\"bool\"}],\"name\":\"checkIsTopLevelOrWasAliased\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"counter\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"difficulty\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"emitNullEvent\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBlockDifficulty\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"increment\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"incrementEmit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"incrementRedeem\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"expected\",\"type\":\"uint256\"}],\"name\":\"logAndIncrement\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"noop\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pleaseRevert\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractISequencerInbox\",\"name\":\"sequencerInbox\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"batchData\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"numberToPost\",\"type\":\"uint256\"}],\"name\":\"postManyBatches\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"storeDifficulty\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506111e4806100206000396000f3fe608060405234801561001057600080fd5b50600436106101005760003560e01c806361bc221a11610097578063b226a96411610066578063b226a964146101c0578063cff36f2d146101c8578063d09de08a146101d1578063ded5ecad146101d957600080fd5b806361bc221a146101655780638a390877146101925780639ff5ccac146101a5578063b1948fc3146101ad57600080fd5b80631a2f8a92116100d35780631a2f8a921461013757806344c25fba1461014a5780635677c11e1461015d5780635dfc2e4a1461010d57600080fd5b806305795f73146101055780630e8c389f1461010f57806312e05dd11461011757806319cae4621461012e575b600080fd5b61010d6101ec565b005b61010d610239565b6001545b6040519081526020015b60405180910390f35b61011b60015481565b61011b610145366004610d9b565b610422565b61010d610158366004610e2e565b6104a6565b61011b61092f565b6000546101799067ffffffffffffffff1681565b60405167ffffffffffffffff9091168152602001610125565b61010d6101a0366004610eb0565b61099b565b61010d610a24565b61010d6101bb366004610ef8565b610a93565b61010d610c15565b61010d44600155565b61010d610c40565b61010d6101e7366004610fc5565b610c82565b60405162461bcd60e51b815260206004820152601260248201527f534f4c49444954595f524556455254494e47000000000000000000000000000060448201526064015b60405180910390fd5b3332146102885760405162461bcd60e51b815260206004820152601160248201527f53454e4445525f4e4f545f4f524947494e0000000000000000000000000000006044820152606401610230565b60646001600160a01b031663175a260b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156102c7573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906102eb9190610ffe565b6103375760405162461bcd60e51b815260206004820152600b60248201527f4e4f545f414c49415345440000000000000000000000000000000000000000006044820152606401610230565b6000805467ffffffffffffffff16908061035083611051565b91906101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550507f773c78bf96e65f61c1a2622b47d76e78bfe70dd59cf4f11470c4c121c315941333606e6001600160a01b031663de4ba2b36040518163ffffffff1660e01b8152600401602060405180830381865afa1580156103d8573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906103fc9190611078565b604080516001600160a01b039384168152929091166020830152015b60405180910390a1565b6000805a90506001600160a01b03851661043e61271083611095565b858560405161044e9291906110ae565b6000604051808303818686fa925050503d806000811461048a576040519150601f19603f3d011682016040523d82523d6000602084013e61048f565b606091505b5050505a61049d9082611095565b95945050505050565b85156105665784151560646001600160a01b03166308bd624c6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156104ee573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105129190610ffe565b1515146105615760405162461bcd60e51b815260206004820152601160248201527f554e45585045435445445f524553554c540000000000000000000000000000006044820152606401610230565b61061b565b84151560646001600160a01b031663175a260b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105a8573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105cc9190610ffe565b15151461061b5760405162461bcd60e51b815260206004820152601160248201527f554e45585045435445445f524553554c540000000000000000000000000000006044820152606401610230565b60405163ded5ecad60e01b815286151560048201528415156024820152309063ded5ecad9060440160006040518083038186803b15801561065b57600080fd5b505afa15801561066f573d6000803e3d6000fd5b505060408051891515602482015286151560448083019190915282518083039091018152606490910182526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1663ded5ecad60e01b17905290519092506000915030906106df9084906110e2565b600060405180830381855af49150503d806000811461071a576040519150601f19603f3d011682016040523d82523d6000602084013e61071f565b606091505b50509050806107705760405162461bcd60e51b815260206004820152601460248201527f44454c45474154455f43414c4c5f4641494c45440000000000000000000000006044820152606401610230565b6040805189151560248201528515156044808301919091528251808303909101815260649091019091526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1663ded5ecad60e01b1781528151919350600091829182305af29050806108265760405162461bcd60e51b815260206004820152600f60248201527f43414c4c434f44455f4641494c454400000000000000000000000000000000006044820152606401610230565b60408051891515602482015284151560448083019190915282518083039091018152606490910182526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1663ded5ecad60e01b179052905190925030906108909084906110e2565b6000604051808303816000865af19150503d80600081146108cd576040519150601f19603f3d011682016040523d82523d6000602084013e6108d2565b606091505b505080915050806109255760405162461bcd60e51b815260206004820152600b60248201527f43414c4c5f4641494c45440000000000000000000000000000000000000000006044820152606401610230565b5050505050505050565b600061093c600243611095565b40610948600143611095565b40036109965760405162461bcd60e51b815260206004820152600f60248201527f53414d455f424c4f434b5f4841534800000000000000000000000000000000006044820152606401610230565b504390565b6000546040805183815267ffffffffffffffff90921660208301527f8df8e492f407b078593c5d8fd7e65ef68505999d911d5b99b017c0b7077398b9910160405180910390a16000805467ffffffffffffffff1690806109fa83611051565b91906101000a81548167ffffffffffffffff021916908367ffffffffffffffff1602179055505050565b6000805467ffffffffffffffff169080610a3d83611051565b82546101009290920a67ffffffffffffffff818102199093169183160217909155600054604051911681527fa45d7e79cb3c6044f30c8dd891e6571301d6b8b6618df519c987905ec70742e79150602001610418565b6000836001600160a01b03166306f130566040518163ffffffff1660e01b8152600401602060405180830381865afa158015610ad3573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610af791906110fe565b90506000846001600160a01b0316637fa3a40e6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610b39573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b5d91906110fe565b905060005b83811015610c0d576040517fe0bc97290000000000000000000000000000000000000000000000000000000081526001600160a01b0387169063e0bc972990610bba9086908990879060009081908190600401611117565b600060405180830381600087803b158015610bd457600080fd5b505af1158015610be8573d6000803e3d6000fd5b505050508280610bf790611176565b9350508080610c0590611176565b915050610b62565b505050505050565b6040517f6f59c82101949290205a9ae9d0c657e6dae1a71c301ae76d385c2792294585fe90600090a1565b6000805467ffffffffffffffff169080610c5983611051565b91906101000a81548167ffffffffffffffff021916908367ffffffffffffffff16021790555050565b8115610d415780151560646001600160a01b03166308bd624c6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610cca573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610cee9190610ffe565b151514610d3d5760405162461bcd60e51b815260206004820152601160248201527f554e45585045435445445f524553554c540000000000000000000000000000006044820152606401610230565b5050565b80151560646001600160a01b031663175a260b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610cca573d6000803e3d6000fd5b6001600160a01b0381168114610d9857600080fd5b50565b600080600060408486031215610db057600080fd5b8335610dbb81610d83565b9250602084013567ffffffffffffffff80821115610dd857600080fd5b818601915086601f830112610dec57600080fd5b813581811115610dfb57600080fd5b876020828501011115610e0d57600080fd5b6020830194508093505050509250925092565b8015158114610d9857600080fd5b60008060008060008060c08789031215610e4757600080fd5b8635610e5281610e20565b95506020870135610e6281610e20565b94506040870135610e7281610e20565b93506060870135610e8281610e20565b92506080870135610e9281610e20565b915060a0870135610ea281610e20565b809150509295509295509295565b600060208284031215610ec257600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600080600060608486031215610f0d57600080fd5b8335610f1881610d83565b9250602084013567ffffffffffffffff80821115610f3557600080fd5b818601915086601f830112610f4957600080fd5b813581811115610f5b57610f5b610ec9565b604051601f8201601f19908116603f01168101908382118183101715610f8357610f83610ec9565b81604052828152896020848701011115610f9c57600080fd5b826020860160208301376000602084830101528096505050505050604084013590509250925092565b60008060408385031215610fd857600080fd5b8235610fe381610e20565b91506020830135610ff381610e20565b809150509250929050565b60006020828403121561101057600080fd5b815161101b81610e20565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600067ffffffffffffffff80831681810361106e5761106e611022565b6001019392505050565b60006020828403121561108a57600080fd5b815161101b81610d83565b818103818111156110a8576110a8611022565b92915050565b8183823760009101908152919050565b60005b838110156110d95781810151838201526020016110c1565b50506000910152565b600082516110f48184602087016110be565b9190910192915050565b60006020828403121561111057600080fd5b5051919050565b86815260c06020820152600086518060c084015261113c8160e0850160208b016110be565b6040830196909652506001600160a01b03939093166060840152608083019190915260a082015260e0601f909201601f1916010192915050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036111a7576111a7611022565b506001019056fea2646970667358221220e36929500ec40a6fbe0e66c9e5e7797bdb762fd4f042a2cd446c18d6ec7e325764736f6c63430008110033",
}

// SimpleABI is the input ABI used to generate the binding from.
// Deprecated: Use SimpleMetaData.ABI instead.
var SimpleABI = SimpleMetaData.ABI

// SimpleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SimpleMetaData.Bin instead.
var SimpleBin = SimpleMetaData.Bin

// DeploySimple deploys a new Ethereum contract, binding an instance of Simple to it.
func DeploySimple(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Simple, error) {
	parsed, err := SimpleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SimpleBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Simple{SimpleCaller: SimpleCaller{contract: contract}, SimpleTransactor: SimpleTransactor{contract: contract}, SimpleFilterer: SimpleFilterer{contract: contract}}, nil
}

// Simple is an auto generated Go binding around an Ethereum contract.
type Simple struct {
	SimpleCaller     // Read-only binding to the contract
	SimpleTransactor // Write-only binding to the contract
	SimpleFilterer   // Log filterer for contract events
}

// SimpleCaller is an auto generated read-only Go binding around an Ethereum contract.
type SimpleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SimpleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SimpleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SimpleSession struct {
	Contract     *Simple           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SimpleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SimpleCallerSession struct {
	Contract *SimpleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// SimpleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SimpleTransactorSession struct {
	Contract     *SimpleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SimpleRaw is an auto generated low-level Go binding around an Ethereum contract.
type SimpleRaw struct {
	Contract *Simple // Generic contract binding to access the raw methods on
}

// SimpleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SimpleCallerRaw struct {
	Contract *SimpleCaller // Generic read-only contract binding to access the raw methods on
}

// SimpleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SimpleTransactorRaw struct {
	Contract *SimpleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSimple creates a new instance of Simple, bound to a specific deployed contract.
func NewSimple(address common.Address, backend bind.ContractBackend) (*Simple, error) {
	contract, err := bindSimple(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Simple{SimpleCaller: SimpleCaller{contract: contract}, SimpleTransactor: SimpleTransactor{contract: contract}, SimpleFilterer: SimpleFilterer{contract: contract}}, nil
}

// NewSimpleCaller creates a new read-only instance of Simple, bound to a specific deployed contract.
func NewSimpleCaller(address common.Address, caller bind.ContractCaller) (*SimpleCaller, error) {
	contract, err := bindSimple(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleCaller{contract: contract}, nil
}

// NewSimpleTransactor creates a new write-only instance of Simple, bound to a specific deployed contract.
func NewSimpleTransactor(address common.Address, transactor bind.ContractTransactor) (*SimpleTransactor, error) {
	contract, err := bindSimple(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleTransactor{contract: contract}, nil
}

// NewSimpleFilterer creates a new log filterer instance of Simple, bound to a specific deployed contract.
func NewSimpleFilterer(address common.Address, filterer bind.ContractFilterer) (*SimpleFilterer, error) {
	contract, err := bindSimple(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SimpleFilterer{contract: contract}, nil
}

// bindSimple binds a generic wrapper to an already deployed contract.
func bindSimple(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SimpleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Simple *SimpleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Simple.Contract.SimpleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Simple *SimpleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simple.Contract.SimpleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Simple *SimpleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Simple.Contract.SimpleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Simple *SimpleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Simple.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Simple *SimpleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simple.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Simple *SimpleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Simple.Contract.contract.Transact(opts, method, params...)
}

// CheckBlockHashes is a free data retrieval call binding the contract method 0x5677c11e.
//
// Solidity: function checkBlockHashes() view returns(uint256)
func (_Simple *SimpleCaller) CheckBlockHashes(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Simple.contract.Call(opts, &out, "checkBlockHashes")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CheckBlockHashes is a free data retrieval call binding the contract method 0x5677c11e.
//
// Solidity: function checkBlockHashes() view returns(uint256)
func (_Simple *SimpleSession) CheckBlockHashes() (*big.Int, error) {
	return _Simple.Contract.CheckBlockHashes(&_Simple.CallOpts)
}

// CheckBlockHashes is a free data retrieval call binding the contract method 0x5677c11e.
//
// Solidity: function checkBlockHashes() view returns(uint256)
func (_Simple *SimpleCallerSession) CheckBlockHashes() (*big.Int, error) {
	return _Simple.Contract.CheckBlockHashes(&_Simple.CallOpts)
}

// CheckGasUsed is a free data retrieval call binding the contract method 0x1a2f8a92.
//
// Solidity: function checkGasUsed(address to, bytes input) view returns(uint256)
func (_Simple *SimpleCaller) CheckGasUsed(opts *bind.CallOpts, to common.Address, input []byte) (*big.Int, error) {
	var out []interface{}
	err := _Simple.contract.Call(opts, &out, "checkGasUsed", to, input)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CheckGasUsed is a free data retrieval call binding the contract method 0x1a2f8a92.
//
// Solidity: function checkGasUsed(address to, bytes input) view returns(uint256)
func (_Simple *SimpleSession) CheckGasUsed(to common.Address, input []byte) (*big.Int, error) {
	return _Simple.Contract.CheckGasUsed(&_Simple.CallOpts, to, input)
}

// CheckGasUsed is a free data retrieval call binding the contract method 0x1a2f8a92.
//
// Solidity: function checkGasUsed(address to, bytes input) view returns(uint256)
func (_Simple *SimpleCallerSession) CheckGasUsed(to common.Address, input []byte) (*big.Int, error) {
	return _Simple.Contract.CheckGasUsed(&_Simple.CallOpts, to, input)
}

// CheckIsTopLevelOrWasAliased is a free data retrieval call binding the contract method 0xded5ecad.
//
// Solidity: function checkIsTopLevelOrWasAliased(bool useTopLevel, bool expected) view returns()
func (_Simple *SimpleCaller) CheckIsTopLevelOrWasAliased(opts *bind.CallOpts, useTopLevel bool, expected bool) error {
	var out []interface{}
	err := _Simple.contract.Call(opts, &out, "checkIsTopLevelOrWasAliased", useTopLevel, expected)

	if err != nil {
		return err
	}

	return err

}

// CheckIsTopLevelOrWasAliased is a free data retrieval call binding the contract method 0xded5ecad.
//
// Solidity: function checkIsTopLevelOrWasAliased(bool useTopLevel, bool expected) view returns()
func (_Simple *SimpleSession) CheckIsTopLevelOrWasAliased(useTopLevel bool, expected bool) error {
	return _Simple.Contract.CheckIsTopLevelOrWasAliased(&_Simple.CallOpts, useTopLevel, expected)
}

// CheckIsTopLevelOrWasAliased is a free data retrieval call binding the contract method 0xded5ecad.
//
// Solidity: function checkIsTopLevelOrWasAliased(bool useTopLevel, bool expected) view returns()
func (_Simple *SimpleCallerSession) CheckIsTopLevelOrWasAliased(useTopLevel bool, expected bool) error {
	return _Simple.Contract.CheckIsTopLevelOrWasAliased(&_Simple.CallOpts, useTopLevel, expected)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint64)
func (_Simple *SimpleCaller) Counter(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Simple.contract.Call(opts, &out, "counter")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint64)
func (_Simple *SimpleSession) Counter() (uint64, error) {
	return _Simple.Contract.Counter(&_Simple.CallOpts)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint64)
func (_Simple *SimpleCallerSession) Counter() (uint64, error) {
	return _Simple.Contract.Counter(&_Simple.CallOpts)
}

// Difficulty is a free data retrieval call binding the contract method 0x19cae462.
//
// Solidity: function difficulty() view returns(uint256)
func (_Simple *SimpleCaller) Difficulty(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Simple.contract.Call(opts, &out, "difficulty")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Difficulty is a free data retrieval call binding the contract method 0x19cae462.
//
// Solidity: function difficulty() view returns(uint256)
func (_Simple *SimpleSession) Difficulty() (*big.Int, error) {
	return _Simple.Contract.Difficulty(&_Simple.CallOpts)
}

// Difficulty is a free data retrieval call binding the contract method 0x19cae462.
//
// Solidity: function difficulty() view returns(uint256)
func (_Simple *SimpleCallerSession) Difficulty() (*big.Int, error) {
	return _Simple.Contract.Difficulty(&_Simple.CallOpts)
}

// GetBlockDifficulty is a free data retrieval call binding the contract method 0x12e05dd1.
//
// Solidity: function getBlockDifficulty() view returns(uint256)
func (_Simple *SimpleCaller) GetBlockDifficulty(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Simple.contract.Call(opts, &out, "getBlockDifficulty")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBlockDifficulty is a free data retrieval call binding the contract method 0x12e05dd1.
//
// Solidity: function getBlockDifficulty() view returns(uint256)
func (_Simple *SimpleSession) GetBlockDifficulty() (*big.Int, error) {
	return _Simple.Contract.GetBlockDifficulty(&_Simple.CallOpts)
}

// GetBlockDifficulty is a free data retrieval call binding the contract method 0x12e05dd1.
//
// Solidity: function getBlockDifficulty() view returns(uint256)
func (_Simple *SimpleCallerSession) GetBlockDifficulty() (*big.Int, error) {
	return _Simple.Contract.GetBlockDifficulty(&_Simple.CallOpts)
}

// Noop is a free data retrieval call binding the contract method 0x5dfc2e4a.
//
// Solidity: function noop() pure returns()
func (_Simple *SimpleCaller) Noop(opts *bind.CallOpts) error {
	var out []interface{}
	err := _Simple.contract.Call(opts, &out, "noop")

	if err != nil {
		return err
	}

	return err

}

// Noop is a free data retrieval call binding the contract method 0x5dfc2e4a.
//
// Solidity: function noop() pure returns()
func (_Simple *SimpleSession) Noop() error {
	return _Simple.Contract.Noop(&_Simple.CallOpts)
}

// Noop is a free data retrieval call binding the contract method 0x5dfc2e4a.
//
// Solidity: function noop() pure returns()
func (_Simple *SimpleCallerSession) Noop() error {
	return _Simple.Contract.Noop(&_Simple.CallOpts)
}

// PleaseRevert is a free data retrieval call binding the contract method 0x05795f73.
//
// Solidity: function pleaseRevert() pure returns()
func (_Simple *SimpleCaller) PleaseRevert(opts *bind.CallOpts) error {
	var out []interface{}
	err := _Simple.contract.Call(opts, &out, "pleaseRevert")

	if err != nil {
		return err
	}

	return err

}

// PleaseRevert is a free data retrieval call binding the contract method 0x05795f73.
//
// Solidity: function pleaseRevert() pure returns()
func (_Simple *SimpleSession) PleaseRevert() error {
	return _Simple.Contract.PleaseRevert(&_Simple.CallOpts)
}

// PleaseRevert is a free data retrieval call binding the contract method 0x05795f73.
//
// Solidity: function pleaseRevert() pure returns()
func (_Simple *SimpleCallerSession) PleaseRevert() error {
	return _Simple.Contract.PleaseRevert(&_Simple.CallOpts)
}

// CheckCalls is a paid mutator transaction binding the contract method 0x44c25fba.
//
// Solidity: function checkCalls(bool useTopLevel, bool directCase, bool staticCase, bool delegateCase, bool callcodeCase, bool callCase) returns()
func (_Simple *SimpleTransactor) CheckCalls(opts *bind.TransactOpts, useTopLevel bool, directCase bool, staticCase bool, delegateCase bool, callcodeCase bool, callCase bool) (*types.Transaction, error) {
	return _Simple.contract.Transact(opts, "checkCalls", useTopLevel, directCase, staticCase, delegateCase, callcodeCase, callCase)
}

// CheckCalls is a paid mutator transaction binding the contract method 0x44c25fba.
//
// Solidity: function checkCalls(bool useTopLevel, bool directCase, bool staticCase, bool delegateCase, bool callcodeCase, bool callCase) returns()
func (_Simple *SimpleSession) CheckCalls(useTopLevel bool, directCase bool, staticCase bool, delegateCase bool, callcodeCase bool, callCase bool) (*types.Transaction, error) {
	return _Simple.Contract.CheckCalls(&_Simple.TransactOpts, useTopLevel, directCase, staticCase, delegateCase, callcodeCase, callCase)
}

// CheckCalls is a paid mutator transaction binding the contract method 0x44c25fba.
//
// Solidity: function checkCalls(bool useTopLevel, bool directCase, bool staticCase, bool delegateCase, bool callcodeCase, bool callCase) returns()
func (_Simple *SimpleTransactorSession) CheckCalls(useTopLevel bool, directCase bool, staticCase bool, delegateCase bool, callcodeCase bool, callCase bool) (*types.Transaction, error) {
	return _Simple.Contract.CheckCalls(&_Simple.TransactOpts, useTopLevel, directCase, staticCase, delegateCase, callcodeCase, callCase)
}

// EmitNullEvent is a paid mutator transaction binding the contract method 0xb226a964.
//
// Solidity: function emitNullEvent() returns()
func (_Simple *SimpleTransactor) EmitNullEvent(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simple.contract.Transact(opts, "emitNullEvent")
}

// EmitNullEvent is a paid mutator transaction binding the contract method 0xb226a964.
//
// Solidity: function emitNullEvent() returns()
func (_Simple *SimpleSession) EmitNullEvent() (*types.Transaction, error) {
	return _Simple.Contract.EmitNullEvent(&_Simple.TransactOpts)
}

// EmitNullEvent is a paid mutator transaction binding the contract method 0xb226a964.
//
// Solidity: function emitNullEvent() returns()
func (_Simple *SimpleTransactorSession) EmitNullEvent() (*types.Transaction, error) {
	return _Simple.Contract.EmitNullEvent(&_Simple.TransactOpts)
}

// Increment is a paid mutator transaction binding the contract method 0xd09de08a.
//
// Solidity: function increment() returns()
func (_Simple *SimpleTransactor) Increment(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simple.contract.Transact(opts, "increment")
}

// Increment is a paid mutator transaction binding the contract method 0xd09de08a.
//
// Solidity: function increment() returns()
func (_Simple *SimpleSession) Increment() (*types.Transaction, error) {
	return _Simple.Contract.Increment(&_Simple.TransactOpts)
}

// Increment is a paid mutator transaction binding the contract method 0xd09de08a.
//
// Solidity: function increment() returns()
func (_Simple *SimpleTransactorSession) Increment() (*types.Transaction, error) {
	return _Simple.Contract.Increment(&_Simple.TransactOpts)
}

// IncrementEmit is a paid mutator transaction binding the contract method 0x9ff5ccac.
//
// Solidity: function incrementEmit() returns()
func (_Simple *SimpleTransactor) IncrementEmit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simple.contract.Transact(opts, "incrementEmit")
}

// IncrementEmit is a paid mutator transaction binding the contract method 0x9ff5ccac.
//
// Solidity: function incrementEmit() returns()
func (_Simple *SimpleSession) IncrementEmit() (*types.Transaction, error) {
	return _Simple.Contract.IncrementEmit(&_Simple.TransactOpts)
}

// IncrementEmit is a paid mutator transaction binding the contract method 0x9ff5ccac.
//
// Solidity: function incrementEmit() returns()
func (_Simple *SimpleTransactorSession) IncrementEmit() (*types.Transaction, error) {
	return _Simple.Contract.IncrementEmit(&_Simple.TransactOpts)
}

// IncrementRedeem is a paid mutator transaction binding the contract method 0x0e8c389f.
//
// Solidity: function incrementRedeem() returns()
func (_Simple *SimpleTransactor) IncrementRedeem(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simple.contract.Transact(opts, "incrementRedeem")
}

// IncrementRedeem is a paid mutator transaction binding the contract method 0x0e8c389f.
//
// Solidity: function incrementRedeem() returns()
func (_Simple *SimpleSession) IncrementRedeem() (*types.Transaction, error) {
	return _Simple.Contract.IncrementRedeem(&_Simple.TransactOpts)
}

// IncrementRedeem is a paid mutator transaction binding the contract method 0x0e8c389f.
//
// Solidity: function incrementRedeem() returns()
func (_Simple *SimpleTransactorSession) IncrementRedeem() (*types.Transaction, error) {
	return _Simple.Contract.IncrementRedeem(&_Simple.TransactOpts)
}

// LogAndIncrement is a paid mutator transaction binding the contract method 0x8a390877.
//
// Solidity: function logAndIncrement(uint256 expected) returns()
func (_Simple *SimpleTransactor) LogAndIncrement(opts *bind.TransactOpts, expected *big.Int) (*types.Transaction, error) {
	return _Simple.contract.Transact(opts, "logAndIncrement", expected)
}

// LogAndIncrement is a paid mutator transaction binding the contract method 0x8a390877.
//
// Solidity: function logAndIncrement(uint256 expected) returns()
func (_Simple *SimpleSession) LogAndIncrement(expected *big.Int) (*types.Transaction, error) {
	return _Simple.Contract.LogAndIncrement(&_Simple.TransactOpts, expected)
}

// LogAndIncrement is a paid mutator transaction binding the contract method 0x8a390877.
//
// Solidity: function logAndIncrement(uint256 expected) returns()
func (_Simple *SimpleTransactorSession) LogAndIncrement(expected *big.Int) (*types.Transaction, error) {
	return _Simple.Contract.LogAndIncrement(&_Simple.TransactOpts, expected)
}

// PostManyBatches is a paid mutator transaction binding the contract method 0xb1948fc3.
//
// Solidity: function postManyBatches(address sequencerInbox, bytes batchData, uint256 numberToPost) returns()
func (_Simple *SimpleTransactor) PostManyBatches(opts *bind.TransactOpts, sequencerInbox common.Address, batchData []byte, numberToPost *big.Int) (*types.Transaction, error) {
	return _Simple.contract.Transact(opts, "postManyBatches", sequencerInbox, batchData, numberToPost)
}

// PostManyBatches is a paid mutator transaction binding the contract method 0xb1948fc3.
//
// Solidity: function postManyBatches(address sequencerInbox, bytes batchData, uint256 numberToPost) returns()
func (_Simple *SimpleSession) PostManyBatches(sequencerInbox common.Address, batchData []byte, numberToPost *big.Int) (*types.Transaction, error) {
	return _Simple.Contract.PostManyBatches(&_Simple.TransactOpts, sequencerInbox, batchData, numberToPost)
}

// PostManyBatches is a paid mutator transaction binding the contract method 0xb1948fc3.
//
// Solidity: function postManyBatches(address sequencerInbox, bytes batchData, uint256 numberToPost) returns()
func (_Simple *SimpleTransactorSession) PostManyBatches(sequencerInbox common.Address, batchData []byte, numberToPost *big.Int) (*types.Transaction, error) {
	return _Simple.Contract.PostManyBatches(&_Simple.TransactOpts, sequencerInbox, batchData, numberToPost)
}

// StoreDifficulty is a paid mutator transaction binding the contract method 0xcff36f2d.
//
// Solidity: function storeDifficulty() returns()
func (_Simple *SimpleTransactor) StoreDifficulty(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simple.contract.Transact(opts, "storeDifficulty")
}

// StoreDifficulty is a paid mutator transaction binding the contract method 0xcff36f2d.
//
// Solidity: function storeDifficulty() returns()
func (_Simple *SimpleSession) StoreDifficulty() (*types.Transaction, error) {
	return _Simple.Contract.StoreDifficulty(&_Simple.TransactOpts)
}

// StoreDifficulty is a paid mutator transaction binding the contract method 0xcff36f2d.
//
// Solidity: function storeDifficulty() returns()
func (_Simple *SimpleTransactorSession) StoreDifficulty() (*types.Transaction, error) {
	return _Simple.Contract.StoreDifficulty(&_Simple.TransactOpts)
}

// SimpleCounterEventIterator is returned from FilterCounterEvent and is used to iterate over the raw logs and unpacked data for CounterEvent events raised by the Simple contract.
type SimpleCounterEventIterator struct {
	Event *SimpleCounterEvent // Event containing the contract specifics and raw log

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
func (it *SimpleCounterEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SimpleCounterEvent)
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
		it.Event = new(SimpleCounterEvent)
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
func (it *SimpleCounterEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SimpleCounterEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SimpleCounterEvent represents a CounterEvent event raised by the Simple contract.
type SimpleCounterEvent struct {
	Count uint64
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterCounterEvent is a free log retrieval operation binding the contract event 0xa45d7e79cb3c6044f30c8dd891e6571301d6b8b6618df519c987905ec70742e7.
//
// Solidity: event CounterEvent(uint64 count)
func (_Simple *SimpleFilterer) FilterCounterEvent(opts *bind.FilterOpts) (*SimpleCounterEventIterator, error) {

	logs, sub, err := _Simple.contract.FilterLogs(opts, "CounterEvent")
	if err != nil {
		return nil, err
	}
	return &SimpleCounterEventIterator{contract: _Simple.contract, event: "CounterEvent", logs: logs, sub: sub}, nil
}

// WatchCounterEvent is a free log subscription operation binding the contract event 0xa45d7e79cb3c6044f30c8dd891e6571301d6b8b6618df519c987905ec70742e7.
//
// Solidity: event CounterEvent(uint64 count)
func (_Simple *SimpleFilterer) WatchCounterEvent(opts *bind.WatchOpts, sink chan<- *SimpleCounterEvent) (event.Subscription, error) {

	logs, sub, err := _Simple.contract.WatchLogs(opts, "CounterEvent")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SimpleCounterEvent)
				if err := _Simple.contract.UnpackLog(event, "CounterEvent", log); err != nil {
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

// ParseCounterEvent is a log parse operation binding the contract event 0xa45d7e79cb3c6044f30c8dd891e6571301d6b8b6618df519c987905ec70742e7.
//
// Solidity: event CounterEvent(uint64 count)
func (_Simple *SimpleFilterer) ParseCounterEvent(log types.Log) (*SimpleCounterEvent, error) {
	event := new(SimpleCounterEvent)
	if err := _Simple.contract.UnpackLog(event, "CounterEvent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SimpleLogAndIncrementCalledIterator is returned from FilterLogAndIncrementCalled and is used to iterate over the raw logs and unpacked data for LogAndIncrementCalled events raised by the Simple contract.
type SimpleLogAndIncrementCalledIterator struct {
	Event *SimpleLogAndIncrementCalled // Event containing the contract specifics and raw log

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
func (it *SimpleLogAndIncrementCalledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SimpleLogAndIncrementCalled)
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
		it.Event = new(SimpleLogAndIncrementCalled)
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
func (it *SimpleLogAndIncrementCalledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SimpleLogAndIncrementCalledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SimpleLogAndIncrementCalled represents a LogAndIncrementCalled event raised by the Simple contract.
type SimpleLogAndIncrementCalled struct {
	Expected *big.Int
	Have     *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterLogAndIncrementCalled is a free log retrieval operation binding the contract event 0x8df8e492f407b078593c5d8fd7e65ef68505999d911d5b99b017c0b7077398b9.
//
// Solidity: event LogAndIncrementCalled(uint256 expected, uint256 have)
func (_Simple *SimpleFilterer) FilterLogAndIncrementCalled(opts *bind.FilterOpts) (*SimpleLogAndIncrementCalledIterator, error) {

	logs, sub, err := _Simple.contract.FilterLogs(opts, "LogAndIncrementCalled")
	if err != nil {
		return nil, err
	}
	return &SimpleLogAndIncrementCalledIterator{contract: _Simple.contract, event: "LogAndIncrementCalled", logs: logs, sub: sub}, nil
}

// WatchLogAndIncrementCalled is a free log subscription operation binding the contract event 0x8df8e492f407b078593c5d8fd7e65ef68505999d911d5b99b017c0b7077398b9.
//
// Solidity: event LogAndIncrementCalled(uint256 expected, uint256 have)
func (_Simple *SimpleFilterer) WatchLogAndIncrementCalled(opts *bind.WatchOpts, sink chan<- *SimpleLogAndIncrementCalled) (event.Subscription, error) {

	logs, sub, err := _Simple.contract.WatchLogs(opts, "LogAndIncrementCalled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SimpleLogAndIncrementCalled)
				if err := _Simple.contract.UnpackLog(event, "LogAndIncrementCalled", log); err != nil {
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

// ParseLogAndIncrementCalled is a log parse operation binding the contract event 0x8df8e492f407b078593c5d8fd7e65ef68505999d911d5b99b017c0b7077398b9.
//
// Solidity: event LogAndIncrementCalled(uint256 expected, uint256 have)
func (_Simple *SimpleFilterer) ParseLogAndIncrementCalled(log types.Log) (*SimpleLogAndIncrementCalled, error) {
	event := new(SimpleLogAndIncrementCalled)
	if err := _Simple.contract.UnpackLog(event, "LogAndIncrementCalled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SimpleNullEventIterator is returned from FilterNullEvent and is used to iterate over the raw logs and unpacked data for NullEvent events raised by the Simple contract.
type SimpleNullEventIterator struct {
	Event *SimpleNullEvent // Event containing the contract specifics and raw log

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
func (it *SimpleNullEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SimpleNullEvent)
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
		it.Event = new(SimpleNullEvent)
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
func (it *SimpleNullEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SimpleNullEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SimpleNullEvent represents a NullEvent event raised by the Simple contract.
type SimpleNullEvent struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterNullEvent is a free log retrieval operation binding the contract event 0x6f59c82101949290205a9ae9d0c657e6dae1a71c301ae76d385c2792294585fe.
//
// Solidity: event NullEvent()
func (_Simple *SimpleFilterer) FilterNullEvent(opts *bind.FilterOpts) (*SimpleNullEventIterator, error) {

	logs, sub, err := _Simple.contract.FilterLogs(opts, "NullEvent")
	if err != nil {
		return nil, err
	}
	return &SimpleNullEventIterator{contract: _Simple.contract, event: "NullEvent", logs: logs, sub: sub}, nil
}

// WatchNullEvent is a free log subscription operation binding the contract event 0x6f59c82101949290205a9ae9d0c657e6dae1a71c301ae76d385c2792294585fe.
//
// Solidity: event NullEvent()
func (_Simple *SimpleFilterer) WatchNullEvent(opts *bind.WatchOpts, sink chan<- *SimpleNullEvent) (event.Subscription, error) {

	logs, sub, err := _Simple.contract.WatchLogs(opts, "NullEvent")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SimpleNullEvent)
				if err := _Simple.contract.UnpackLog(event, "NullEvent", log); err != nil {
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

// ParseNullEvent is a log parse operation binding the contract event 0x6f59c82101949290205a9ae9d0c657e6dae1a71c301ae76d385c2792294585fe.
//
// Solidity: event NullEvent()
func (_Simple *SimpleFilterer) ParseNullEvent(log types.Log) (*SimpleNullEvent, error) {
	event := new(SimpleNullEvent)
	if err := _Simple.contract.UnpackLog(event, "NullEvent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SimpleRedeemedEventIterator is returned from FilterRedeemedEvent and is used to iterate over the raw logs and unpacked data for RedeemedEvent events raised by the Simple contract.
type SimpleRedeemedEventIterator struct {
	Event *SimpleRedeemedEvent // Event containing the contract specifics and raw log

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
func (it *SimpleRedeemedEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SimpleRedeemedEvent)
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
		it.Event = new(SimpleRedeemedEvent)
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
func (it *SimpleRedeemedEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SimpleRedeemedEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SimpleRedeemedEvent represents a RedeemedEvent event raised by the Simple contract.
type SimpleRedeemedEvent struct {
	Caller   common.Address
	Redeemer common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterRedeemedEvent is a free log retrieval operation binding the contract event 0x773c78bf96e65f61c1a2622b47d76e78bfe70dd59cf4f11470c4c121c3159413.
//
// Solidity: event RedeemedEvent(address caller, address redeemer)
func (_Simple *SimpleFilterer) FilterRedeemedEvent(opts *bind.FilterOpts) (*SimpleRedeemedEventIterator, error) {

	logs, sub, err := _Simple.contract.FilterLogs(opts, "RedeemedEvent")
	if err != nil {
		return nil, err
	}
	return &SimpleRedeemedEventIterator{contract: _Simple.contract, event: "RedeemedEvent", logs: logs, sub: sub}, nil
}

// WatchRedeemedEvent is a free log subscription operation binding the contract event 0x773c78bf96e65f61c1a2622b47d76e78bfe70dd59cf4f11470c4c121c3159413.
//
// Solidity: event RedeemedEvent(address caller, address redeemer)
func (_Simple *SimpleFilterer) WatchRedeemedEvent(opts *bind.WatchOpts, sink chan<- *SimpleRedeemedEvent) (event.Subscription, error) {

	logs, sub, err := _Simple.contract.WatchLogs(opts, "RedeemedEvent")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SimpleRedeemedEvent)
				if err := _Simple.contract.UnpackLog(event, "RedeemedEvent", log); err != nil {
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

// ParseRedeemedEvent is a log parse operation binding the contract event 0x773c78bf96e65f61c1a2622b47d76e78bfe70dd59cf4f11470c4c121c3159413.
//
// Solidity: event RedeemedEvent(address caller, address redeemer)
func (_Simple *SimpleFilterer) ParseRedeemedEvent(log types.Log) (*SimpleRedeemedEvent, error) {
	event := new(SimpleRedeemedEvent)
	if err := _Simple.contract.UnpackLog(event, "RedeemedEvent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SimpleOneStepProofEntryMetaData contains all meta data concerning the SimpleOneStepProofEntry contract.
var SimpleOneStepProofEntryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"STEPS_PER_BATCH\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"}],\"internalType\":\"structExecutionState\",\"name\":\"execState\",\"type\":\"tuple\"}],\"name\":\"getMachineHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"}],\"name\":\"getStartMachineHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"initialWasmModuleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"step\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"proveOneStep\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"afterHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610990806100206000396000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c806304997be4146100515780639c2009cd146100cd578063b5112fd2146100ef578063c39619c414610102575b600080fd5b6100ba61005f36600461062e565b6040517f4d616368696e653a0000000000000000000000000000000000000000000000006020820152602881018390526048810182905260009060680160405160208183030381529060405280519060200120905092915050565b6040519081526020015b60405180910390f35b6100d66107d081565b60405167ffffffffffffffff90911681526020016100c4565b6100ba6100fd366004610650565b610115565b6100ba6101103660046106ec565b610439565b600081810361016b5760405162461bcd60e51b815260206004820152600b60248201527f454d5054595f50524f4f4600000000000000000000000000000000000000000060448201526064015b60405180910390fd5b6101736105eb565b600061018085858361055f565b602084015167ffffffffffffffff90921690915290506101a185858361055f565b60208481015167ffffffffffffffff9093169201919091529050861580159061020a57508560001a60f81b7fff0000000000000000000000000000000000000000000000000000000000000016158061020a57506101fe826105c7565b67ffffffffffffffff16155b15610219578592505050610430565b8735610224836105dd565b67ffffffffffffffff161061023d578592505050610430565b8151805160209182015182850151805190840151604080517f476c6f62616c2073746174653a0000000000000000000000000000000000000081880152602d810195909552604d8501939093527fffffffffffffffff00000000000000000000000000000000000000000000000060c092831b8116606d860152911b1660758301528051808303605d018152607d9092019052805191012086146103235760405162461bcd60e51b815260206004820152600960248201527f4241445f50524f4f4600000000000000000000000000000000000000000000006044820152606401610162565b6020828101510180519061033682610730565b67ffffffffffffffff169052506020828101510151610358906107d090610757565b67ffffffffffffffff1660000361039357602082015180519061037a82610730565b67ffffffffffffffff1690525060208281015160009101525b8151805160209182015182850151805190840151604080517f476c6f62616c2073746174653a0000000000000000000000000000000000000081880152602d810195909552604d8501939093527fffffffffffffffff00000000000000000000000000000000000000000000000060c092831b8116606d860152911b1660758301528051808303605d018152607d90920190528051910120925050505b95945050505050565b6000600161044d60a08401608085016107a2565b600281111561045e5761045e61078c565b146104ab5760405162461bcd60e51b815260206004820152601260248201527f4241445f4d414348494e455f53544154555300000000000000000000000000006044820152606401610162565b6105596104bd36849003840184610889565b8051805160209182015192820151805190830151604080517f476c6f62616c2073746174653a0000000000000000000000000000000000000081870152602d810194909452604d8401959095527fffffffffffffffff00000000000000000000000000000000000000000000000060c092831b8116606d850152911b1660758201528251808203605d018152607d909101909252815191012090565b92915050565b600081815b60088110156105be5760088367ffffffffffffffff16901b925085858381811061059057610590610704565b919091013560f81c939093179250816105a881610922565b92505080806105b690610922565b915050610564565b50935093915050565b602081015160009060015b602002015192915050565b6020810151600090816105d2565b60405180604001604052806105fe610610565b815260200161060b610610565b905290565b60405180604001604052806002906020820280368337509192915050565b6000806040838503121561064157600080fd5b50508035926020909101359150565b600080600080600085870360c081121561066957600080fd5b606081121561067757600080fd5b50859450606086013593506080860135925060a086013567ffffffffffffffff808211156106a457600080fd5b818801915088601f8301126106b857600080fd5b8135818111156106c757600080fd5b8960208285010111156106d957600080fd5b9699959850939650602001949392505050565b600060a082840312156106fe57600080fd5b50919050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b600067ffffffffffffffff80831681810361074d5761074d61071a565b6001019392505050565b600067ffffffffffffffff8084168061078057634e487b7160e01b600052601260045260246000fd5b92169190910692915050565b634e487b7160e01b600052602160045260246000fd5b6000602082840312156107b457600080fd5b8135600381106107c357600080fd5b9392505050565b634e487b7160e01b600052604160045260246000fd5b6040805190810167ffffffffffffffff81118282101715610803576108036107ca565b60405290565b600082601f83011261081a57600080fd5b6040516040810167ffffffffffffffff828210818311171561083e5761083e6107ca565b6040918252829185018681111561085457600080fd5b855b8181101561087d578035838116811461086f5760008081fd5b845260209384019301610856565b50929695505050505050565b60006080828403121561089b57600080fd5b6040516040810181811067ffffffffffffffff821117156108be576108be6107ca565b604052601f830184136108d057600080fd5b6108d86107e0565b8060408501868111156108ea57600080fd5b855b818110156109045780358452602093840193016108ec565b508184526109128782610809565b6020850152509195945050505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036109535761095361071a565b506001019056fea264697066735822122059d89145b06524df7cee6ef328068e39a4f4fbd511226ff4e6892d9c94e29d8664736f6c63430008110033",
}

// SimpleOneStepProofEntryABI is the input ABI used to generate the binding from.
// Deprecated: Use SimpleOneStepProofEntryMetaData.ABI instead.
var SimpleOneStepProofEntryABI = SimpleOneStepProofEntryMetaData.ABI

// SimpleOneStepProofEntryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SimpleOneStepProofEntryMetaData.Bin instead.
var SimpleOneStepProofEntryBin = SimpleOneStepProofEntryMetaData.Bin

// DeploySimpleOneStepProofEntry deploys a new Ethereum contract, binding an instance of SimpleOneStepProofEntry to it.
func DeploySimpleOneStepProofEntry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SimpleOneStepProofEntry, error) {
	parsed, err := SimpleOneStepProofEntryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SimpleOneStepProofEntryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SimpleOneStepProofEntry{SimpleOneStepProofEntryCaller: SimpleOneStepProofEntryCaller{contract: contract}, SimpleOneStepProofEntryTransactor: SimpleOneStepProofEntryTransactor{contract: contract}, SimpleOneStepProofEntryFilterer: SimpleOneStepProofEntryFilterer{contract: contract}}, nil
}

// SimpleOneStepProofEntry is an auto generated Go binding around an Ethereum contract.
type SimpleOneStepProofEntry struct {
	SimpleOneStepProofEntryCaller     // Read-only binding to the contract
	SimpleOneStepProofEntryTransactor // Write-only binding to the contract
	SimpleOneStepProofEntryFilterer   // Log filterer for contract events
}

// SimpleOneStepProofEntryCaller is an auto generated read-only Go binding around an Ethereum contract.
type SimpleOneStepProofEntryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleOneStepProofEntryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SimpleOneStepProofEntryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleOneStepProofEntryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SimpleOneStepProofEntryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleOneStepProofEntrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SimpleOneStepProofEntrySession struct {
	Contract     *SimpleOneStepProofEntry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts            // Call options to use throughout this session
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// SimpleOneStepProofEntryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SimpleOneStepProofEntryCallerSession struct {
	Contract *SimpleOneStepProofEntryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                  // Call options to use throughout this session
}

// SimpleOneStepProofEntryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SimpleOneStepProofEntryTransactorSession struct {
	Contract     *SimpleOneStepProofEntryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                  // Transaction auth options to use throughout this session
}

// SimpleOneStepProofEntryRaw is an auto generated low-level Go binding around an Ethereum contract.
type SimpleOneStepProofEntryRaw struct {
	Contract *SimpleOneStepProofEntry // Generic contract binding to access the raw methods on
}

// SimpleOneStepProofEntryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SimpleOneStepProofEntryCallerRaw struct {
	Contract *SimpleOneStepProofEntryCaller // Generic read-only contract binding to access the raw methods on
}

// SimpleOneStepProofEntryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SimpleOneStepProofEntryTransactorRaw struct {
	Contract *SimpleOneStepProofEntryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSimpleOneStepProofEntry creates a new instance of SimpleOneStepProofEntry, bound to a specific deployed contract.
func NewSimpleOneStepProofEntry(address common.Address, backend bind.ContractBackend) (*SimpleOneStepProofEntry, error) {
	contract, err := bindSimpleOneStepProofEntry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SimpleOneStepProofEntry{SimpleOneStepProofEntryCaller: SimpleOneStepProofEntryCaller{contract: contract}, SimpleOneStepProofEntryTransactor: SimpleOneStepProofEntryTransactor{contract: contract}, SimpleOneStepProofEntryFilterer: SimpleOneStepProofEntryFilterer{contract: contract}}, nil
}

// NewSimpleOneStepProofEntryCaller creates a new read-only instance of SimpleOneStepProofEntry, bound to a specific deployed contract.
func NewSimpleOneStepProofEntryCaller(address common.Address, caller bind.ContractCaller) (*SimpleOneStepProofEntryCaller, error) {
	contract, err := bindSimpleOneStepProofEntry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleOneStepProofEntryCaller{contract: contract}, nil
}

// NewSimpleOneStepProofEntryTransactor creates a new write-only instance of SimpleOneStepProofEntry, bound to a specific deployed contract.
func NewSimpleOneStepProofEntryTransactor(address common.Address, transactor bind.ContractTransactor) (*SimpleOneStepProofEntryTransactor, error) {
	contract, err := bindSimpleOneStepProofEntry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleOneStepProofEntryTransactor{contract: contract}, nil
}

// NewSimpleOneStepProofEntryFilterer creates a new log filterer instance of SimpleOneStepProofEntry, bound to a specific deployed contract.
func NewSimpleOneStepProofEntryFilterer(address common.Address, filterer bind.ContractFilterer) (*SimpleOneStepProofEntryFilterer, error) {
	contract, err := bindSimpleOneStepProofEntry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SimpleOneStepProofEntryFilterer{contract: contract}, nil
}

// bindSimpleOneStepProofEntry binds a generic wrapper to an already deployed contract.
func bindSimpleOneStepProofEntry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SimpleOneStepProofEntryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SimpleOneStepProofEntry.Contract.SimpleOneStepProofEntryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimpleOneStepProofEntry.Contract.SimpleOneStepProofEntryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SimpleOneStepProofEntry.Contract.SimpleOneStepProofEntryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SimpleOneStepProofEntry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimpleOneStepProofEntry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SimpleOneStepProofEntry.Contract.contract.Transact(opts, method, params...)
}

// STEPSPERBATCH is a free data retrieval call binding the contract method 0x9c2009cd.
//
// Solidity: function STEPS_PER_BATCH() view returns(uint64)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryCaller) STEPSPERBATCH(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _SimpleOneStepProofEntry.contract.Call(opts, &out, "STEPS_PER_BATCH")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// STEPSPERBATCH is a free data retrieval call binding the contract method 0x9c2009cd.
//
// Solidity: function STEPS_PER_BATCH() view returns(uint64)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntrySession) STEPSPERBATCH() (uint64, error) {
	return _SimpleOneStepProofEntry.Contract.STEPSPERBATCH(&_SimpleOneStepProofEntry.CallOpts)
}

// STEPSPERBATCH is a free data retrieval call binding the contract method 0x9c2009cd.
//
// Solidity: function STEPS_PER_BATCH() view returns(uint64)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryCallerSession) STEPSPERBATCH() (uint64, error) {
	return _SimpleOneStepProofEntry.Contract.STEPSPERBATCH(&_SimpleOneStepProofEntry.CallOpts)
}

// GetMachineHash is a free data retrieval call binding the contract method 0xc39619c4.
//
// Solidity: function getMachineHash(((bytes32[2],uint64[2]),uint8) execState) pure returns(bytes32)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryCaller) GetMachineHash(opts *bind.CallOpts, execState ExecutionState) ([32]byte, error) {
	var out []interface{}
	err := _SimpleOneStepProofEntry.contract.Call(opts, &out, "getMachineHash", execState)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetMachineHash is a free data retrieval call binding the contract method 0xc39619c4.
//
// Solidity: function getMachineHash(((bytes32[2],uint64[2]),uint8) execState) pure returns(bytes32)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntrySession) GetMachineHash(execState ExecutionState) ([32]byte, error) {
	return _SimpleOneStepProofEntry.Contract.GetMachineHash(&_SimpleOneStepProofEntry.CallOpts, execState)
}

// GetMachineHash is a free data retrieval call binding the contract method 0xc39619c4.
//
// Solidity: function getMachineHash(((bytes32[2],uint64[2]),uint8) execState) pure returns(bytes32)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryCallerSession) GetMachineHash(execState ExecutionState) ([32]byte, error) {
	return _SimpleOneStepProofEntry.Contract.GetMachineHash(&_SimpleOneStepProofEntry.CallOpts, execState)
}

// GetStartMachineHash is a free data retrieval call binding the contract method 0x04997be4.
//
// Solidity: function getStartMachineHash(bytes32 globalStateHash, bytes32 wasmModuleRoot) pure returns(bytes32)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryCaller) GetStartMachineHash(opts *bind.CallOpts, globalStateHash [32]byte, wasmModuleRoot [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _SimpleOneStepProofEntry.contract.Call(opts, &out, "getStartMachineHash", globalStateHash, wasmModuleRoot)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetStartMachineHash is a free data retrieval call binding the contract method 0x04997be4.
//
// Solidity: function getStartMachineHash(bytes32 globalStateHash, bytes32 wasmModuleRoot) pure returns(bytes32)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntrySession) GetStartMachineHash(globalStateHash [32]byte, wasmModuleRoot [32]byte) ([32]byte, error) {
	return _SimpleOneStepProofEntry.Contract.GetStartMachineHash(&_SimpleOneStepProofEntry.CallOpts, globalStateHash, wasmModuleRoot)
}

// GetStartMachineHash is a free data retrieval call binding the contract method 0x04997be4.
//
// Solidity: function getStartMachineHash(bytes32 globalStateHash, bytes32 wasmModuleRoot) pure returns(bytes32)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryCallerSession) GetStartMachineHash(globalStateHash [32]byte, wasmModuleRoot [32]byte) ([32]byte, error) {
	return _SimpleOneStepProofEntry.Contract.GetStartMachineHash(&_SimpleOneStepProofEntry.CallOpts, globalStateHash, wasmModuleRoot)
}

// ProveOneStep is a free data retrieval call binding the contract method 0xb5112fd2.
//
// Solidity: function proveOneStep((uint256,address,bytes32) execCtx, uint256 step, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryCaller) ProveOneStep(opts *bind.CallOpts, execCtx ExecutionContext, step *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	var out []interface{}
	err := _SimpleOneStepProofEntry.contract.Call(opts, &out, "proveOneStep", execCtx, step, beforeHash, proof)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProveOneStep is a free data retrieval call binding the contract method 0xb5112fd2.
//
// Solidity: function proveOneStep((uint256,address,bytes32) execCtx, uint256 step, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntrySession) ProveOneStep(execCtx ExecutionContext, step *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	return _SimpleOneStepProofEntry.Contract.ProveOneStep(&_SimpleOneStepProofEntry.CallOpts, execCtx, step, beforeHash, proof)
}

// ProveOneStep is a free data retrieval call binding the contract method 0xb5112fd2.
//
// Solidity: function proveOneStep((uint256,address,bytes32) execCtx, uint256 step, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_SimpleOneStepProofEntry *SimpleOneStepProofEntryCallerSession) ProveOneStep(execCtx ExecutionContext, step *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	return _SimpleOneStepProofEntry.Contract.ProveOneStep(&_SimpleOneStepProofEntry.CallOpts, execCtx, step, beforeHash, proof)
}

// SimpleProxyMetaData contains all meta data concerning the SimpleProxy contract.
var SimpleProxyMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"impl_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x60a060405234801561001057600080fd5b5060405161011d38038061011d83398101604081905261002f91610040565b6001600160a01b0316608052610070565b60006020828403121561005257600080fd5b81516001600160a01b038116811461006957600080fd5b9392505050565b608051609561008860003960006017015260956000f3fe608060405236601057600e6013565b005b600e5b603a7f0000000000000000000000000000000000000000000000000000000000000000603c565b565b3660008037600080366000845af43d6000803e808015605a573d6000f35b3d6000fdfea264697066735822122065e1224d690bac6df0886c1fe017c80772ed1b674f697f11e0bf9ae9563587e764736f6c63430008110033",
}

// SimpleProxyABI is the input ABI used to generate the binding from.
// Deprecated: Use SimpleProxyMetaData.ABI instead.
var SimpleProxyABI = SimpleProxyMetaData.ABI

// SimpleProxyBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SimpleProxyMetaData.Bin instead.
var SimpleProxyBin = SimpleProxyMetaData.Bin

// DeploySimpleProxy deploys a new Ethereum contract, binding an instance of SimpleProxy to it.
func DeploySimpleProxy(auth *bind.TransactOpts, backend bind.ContractBackend, impl_ common.Address) (common.Address, *types.Transaction, *SimpleProxy, error) {
	parsed, err := SimpleProxyMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SimpleProxyBin), backend, impl_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SimpleProxy{SimpleProxyCaller: SimpleProxyCaller{contract: contract}, SimpleProxyTransactor: SimpleProxyTransactor{contract: contract}, SimpleProxyFilterer: SimpleProxyFilterer{contract: contract}}, nil
}

// SimpleProxy is an auto generated Go binding around an Ethereum contract.
type SimpleProxy struct {
	SimpleProxyCaller     // Read-only binding to the contract
	SimpleProxyTransactor // Write-only binding to the contract
	SimpleProxyFilterer   // Log filterer for contract events
}

// SimpleProxyCaller is an auto generated read-only Go binding around an Ethereum contract.
type SimpleProxyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleProxyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SimpleProxyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleProxyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SimpleProxyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleProxySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SimpleProxySession struct {
	Contract     *SimpleProxy      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SimpleProxyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SimpleProxyCallerSession struct {
	Contract *SimpleProxyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// SimpleProxyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SimpleProxyTransactorSession struct {
	Contract     *SimpleProxyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// SimpleProxyRaw is an auto generated low-level Go binding around an Ethereum contract.
type SimpleProxyRaw struct {
	Contract *SimpleProxy // Generic contract binding to access the raw methods on
}

// SimpleProxyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SimpleProxyCallerRaw struct {
	Contract *SimpleProxyCaller // Generic read-only contract binding to access the raw methods on
}

// SimpleProxyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SimpleProxyTransactorRaw struct {
	Contract *SimpleProxyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSimpleProxy creates a new instance of SimpleProxy, bound to a specific deployed contract.
func NewSimpleProxy(address common.Address, backend bind.ContractBackend) (*SimpleProxy, error) {
	contract, err := bindSimpleProxy(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SimpleProxy{SimpleProxyCaller: SimpleProxyCaller{contract: contract}, SimpleProxyTransactor: SimpleProxyTransactor{contract: contract}, SimpleProxyFilterer: SimpleProxyFilterer{contract: contract}}, nil
}

// NewSimpleProxyCaller creates a new read-only instance of SimpleProxy, bound to a specific deployed contract.
func NewSimpleProxyCaller(address common.Address, caller bind.ContractCaller) (*SimpleProxyCaller, error) {
	contract, err := bindSimpleProxy(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleProxyCaller{contract: contract}, nil
}

// NewSimpleProxyTransactor creates a new write-only instance of SimpleProxy, bound to a specific deployed contract.
func NewSimpleProxyTransactor(address common.Address, transactor bind.ContractTransactor) (*SimpleProxyTransactor, error) {
	contract, err := bindSimpleProxy(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleProxyTransactor{contract: contract}, nil
}

// NewSimpleProxyFilterer creates a new log filterer instance of SimpleProxy, bound to a specific deployed contract.
func NewSimpleProxyFilterer(address common.Address, filterer bind.ContractFilterer) (*SimpleProxyFilterer, error) {
	contract, err := bindSimpleProxy(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SimpleProxyFilterer{contract: contract}, nil
}

// bindSimpleProxy binds a generic wrapper to an already deployed contract.
func bindSimpleProxy(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SimpleProxyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SimpleProxy *SimpleProxyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SimpleProxy.Contract.SimpleProxyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SimpleProxy *SimpleProxyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimpleProxy.Contract.SimpleProxyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SimpleProxy *SimpleProxyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SimpleProxy.Contract.SimpleProxyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SimpleProxy *SimpleProxyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SimpleProxy.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SimpleProxy *SimpleProxyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimpleProxy.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SimpleProxy *SimpleProxyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SimpleProxy.Contract.contract.Transact(opts, method, params...)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SimpleProxy *SimpleProxyTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _SimpleProxy.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SimpleProxy *SimpleProxySession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _SimpleProxy.Contract.Fallback(&_SimpleProxy.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SimpleProxy *SimpleProxyTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _SimpleProxy.Contract.Fallback(&_SimpleProxy.TransactOpts, calldata)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_SimpleProxy *SimpleProxyTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimpleProxy.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_SimpleProxy *SimpleProxySession) Receive() (*types.Transaction, error) {
	return _SimpleProxy.Contract.Receive(&_SimpleProxy.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_SimpleProxy *SimpleProxyTransactorSession) Receive() (*types.Transaction, error) {
	return _SimpleProxy.Contract.Receive(&_SimpleProxy.TransactOpts)
}

// TestWETH9MetaData contains all meta data concerning the TestWETH9 contract.
var TestWETH9MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name_\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol_\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"deposit\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b506040516200107838038062001078833981016040819052620000349162000123565b818160036200004483826200021c565b5060046200005382826200021c565b5050505050620002e8565b634e487b7160e01b600052604160045260246000fd5b600082601f8301126200008657600080fd5b81516001600160401b0380821115620000a357620000a36200005e565b604051601f8301601f19908116603f01168101908282118183101715620000ce57620000ce6200005e565b81604052838152602092508683858801011115620000eb57600080fd5b600091505b838210156200010f5785820183015181830184015290820190620000f0565b600093810190920192909252949350505050565b600080604083850312156200013757600080fd5b82516001600160401b03808211156200014f57600080fd5b6200015d8683870162000074565b935060208501519150808211156200017457600080fd5b50620001838582860162000074565b9150509250929050565b600181811c90821680620001a257607f821691505b602082108103620001c357634e487b7160e01b600052602260045260246000fd5b50919050565b601f8211156200021757600081815260208120601f850160051c81016020861015620001f25750805b601f850160051c820191505b818110156200021357828155600101620001fe565b5050505b505050565b81516001600160401b038111156200023857620002386200005e565b62000250816200024984546200018d565b84620001c9565b602080601f8311600181146200028857600084156200026f5750858301515b600019600386901b1c1916600185901b17855562000213565b600085815260208120601f198616915b82811015620002b95788860151825594840194600190910190840162000298565b5085821015620002d85787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b610d8080620002f86000396000f3fe6080604052600436106100d25760003560e01c8063395093511161007f578063a457c2d711610059578063a457c2d71461021a578063a9059cbb1461023a578063d0e30db01461025a578063dd62ed3e1461026257600080fd5b806339509351146101af57806370a08231146101cf57806395d89b411461020557600080fd5b806323b872dd116100b057806323b872dd146101515780632e1a7d4d14610171578063313ce5671461019357600080fd5b806306fdde03146100d7578063095ea7b31461010257806318160ddd14610132575b600080fd5b3480156100e357600080fd5b506100ec6102a8565b6040516100f99190610b46565b60405180910390f35b34801561010e57600080fd5b5061012261011d366004610bce565b61033a565b60405190151581526020016100f9565b34801561013e57600080fd5b506002545b6040519081526020016100f9565b34801561015d57600080fd5b5061012261016c366004610bf8565b610354565b34801561017d57600080fd5b5061019161018c366004610c34565b610378565b005b34801561019f57600080fd5b50604051601281526020016100f9565b3480156101bb57600080fd5b506101226101ca366004610bce565b6103b3565b3480156101db57600080fd5b506101436101ea366004610c4d565b6001600160a01b031660009081526020819052604090205490565b34801561021157600080fd5b506100ec6103f2565b34801561022657600080fd5b50610122610235366004610bce565b610401565b34801561024657600080fd5b50610122610255366004610bce565b6104b0565b6101916104be565b34801561026e57600080fd5b5061014361027d366004610c6f565b6001600160a01b03918216600090815260016020908152604080832093909416825291909152205490565b6060600380546102b790610ca2565b80601f01602080910402602001604051908101604052809291908181526020018280546102e390610ca2565b80156103305780601f1061030557610100808354040283529160200191610330565b820191906000526020600020905b81548152906001019060200180831161031357829003601f168201915b5050505050905090565b6000336103488185856104ca565b60019150505b92915050565b600033610362858285610623565b61036d8585856106d3565b506001949350505050565b61038233826108ea565b604051339082156108fc029083906000818181858888f193505050501580156103af573d6000803e3d6000fd5b5050565b3360008181526001602090815260408083206001600160a01b038716845290915281205490919061034890829086906103ed908790610d24565b6104ca565b6060600480546102b790610ca2565b3360008181526001602090815260408083206001600160a01b0387168452909152812054909190838110156104a35760405162461bcd60e51b815260206004820152602560248201527f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760448201527f207a65726f00000000000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b61036d82868684036104ca565b6000336103488185856106d3565b6104c83334610a67565b565b6001600160a01b0383166105455760405162461bcd60e51b8152602060048201526024808201527f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460448201527f7265737300000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b0382166105c15760405162461bcd60e51b815260206004820152602260248201527f45524332303a20617070726f766520746f20746865207a65726f20616464726560448201527f7373000000000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b0383811660008181526001602090815260408083209487168084529482529182902085905590518481527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92591015b60405180910390a3505050565b6001600160a01b038381166000908152600160209081526040808320938616835292905220547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81146106cd57818110156106c05760405162461bcd60e51b815260206004820152601d60248201527f45524332303a20696e73756666696369656e7420616c6c6f77616e6365000000604482015260640161049a565b6106cd84848484036104ca565b50505050565b6001600160a01b03831661074f5760405162461bcd60e51b815260206004820152602560248201527f45524332303a207472616e736665722066726f6d20746865207a65726f20616460448201527f6472657373000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b0382166107cb5760405162461bcd60e51b815260206004820152602360248201527f45524332303a207472616e7366657220746f20746865207a65726f206164647260448201527f6573730000000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b0383166000908152602081905260409020548181101561085a5760405162461bcd60e51b815260206004820152602660248201527f45524332303a207472616e7366657220616d6f756e742065786365656473206260448201527f616c616e63650000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b03808516600090815260208190526040808220858503905591851681529081208054849290610891908490610d24565b92505081905550826001600160a01b0316846001600160a01b03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040516108dd91815260200190565b60405180910390a36106cd565b6001600160a01b0382166109665760405162461bcd60e51b815260206004820152602160248201527f45524332303a206275726e2066726f6d20746865207a65726f2061646472657360448201527f7300000000000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b038216600090815260208190526040902054818110156109f55760405162461bcd60e51b815260206004820152602260248201527f45524332303a206275726e20616d6f756e7420657863656564732062616c616e60448201527f6365000000000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b0383166000908152602081905260408120838303905560028054849290610a24908490610d37565b90915550506040518281526000906001600160a01b038516907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef90602001610616565b6001600160a01b038216610abd5760405162461bcd60e51b815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f206164647265737300604482015260640161049a565b8060026000828254610acf9190610d24565b90915550506001600160a01b03821660009081526020819052604081208054839290610afc908490610d24565b90915550506040518181526001600160a01b038316906000907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9060200160405180910390a35050565b600060208083528351808285015260005b81811015610b7357858101830151858201604001528201610b57565b5060006040828601015260407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f8301168501019250505092915050565b80356001600160a01b0381168114610bc957600080fd5b919050565b60008060408385031215610be157600080fd5b610bea83610bb2565b946020939093013593505050565b600080600060608486031215610c0d57600080fd5b610c1684610bb2565b9250610c2460208501610bb2565b9150604084013590509250925092565b600060208284031215610c4657600080fd5b5035919050565b600060208284031215610c5f57600080fd5b610c6882610bb2565b9392505050565b60008060408385031215610c8257600080fd5b610c8b83610bb2565b9150610c9960208401610bb2565b90509250929050565b600181811c90821680610cb657607f821691505b602082108103610cef577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b8082018082111561034e5761034e610cf5565b8181038181111561034e5761034e610cf556fea2646970667358221220975244aa934d68cda9d7d535c488636ae49585ad4cd02c6a4f6512749d4e001764736f6c63430008110033",
}

// TestWETH9ABI is the input ABI used to generate the binding from.
// Deprecated: Use TestWETH9MetaData.ABI instead.
var TestWETH9ABI = TestWETH9MetaData.ABI

// TestWETH9Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TestWETH9MetaData.Bin instead.
var TestWETH9Bin = TestWETH9MetaData.Bin

// DeployTestWETH9 deploys a new Ethereum contract, binding an instance of TestWETH9 to it.
func DeployTestWETH9(auth *bind.TransactOpts, backend bind.ContractBackend, name_ string, symbol_ string) (common.Address, *types.Transaction, *TestWETH9, error) {
	parsed, err := TestWETH9MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TestWETH9Bin), backend, name_, symbol_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TestWETH9{TestWETH9Caller: TestWETH9Caller{contract: contract}, TestWETH9Transactor: TestWETH9Transactor{contract: contract}, TestWETH9Filterer: TestWETH9Filterer{contract: contract}}, nil
}

// TestWETH9 is an auto generated Go binding around an Ethereum contract.
type TestWETH9 struct {
	TestWETH9Caller     // Read-only binding to the contract
	TestWETH9Transactor // Write-only binding to the contract
	TestWETH9Filterer   // Log filterer for contract events
}

// TestWETH9Caller is an auto generated read-only Go binding around an Ethereum contract.
type TestWETH9Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestWETH9Transactor is an auto generated write-only Go binding around an Ethereum contract.
type TestWETH9Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestWETH9Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TestWETH9Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestWETH9Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TestWETH9Session struct {
	Contract     *TestWETH9        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TestWETH9CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TestWETH9CallerSession struct {
	Contract *TestWETH9Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// TestWETH9TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TestWETH9TransactorSession struct {
	Contract     *TestWETH9Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// TestWETH9Raw is an auto generated low-level Go binding around an Ethereum contract.
type TestWETH9Raw struct {
	Contract *TestWETH9 // Generic contract binding to access the raw methods on
}

// TestWETH9CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TestWETH9CallerRaw struct {
	Contract *TestWETH9Caller // Generic read-only contract binding to access the raw methods on
}

// TestWETH9TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TestWETH9TransactorRaw struct {
	Contract *TestWETH9Transactor // Generic write-only contract binding to access the raw methods on
}

// NewTestWETH9 creates a new instance of TestWETH9, bound to a specific deployed contract.
func NewTestWETH9(address common.Address, backend bind.ContractBackend) (*TestWETH9, error) {
	contract, err := bindTestWETH9(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TestWETH9{TestWETH9Caller: TestWETH9Caller{contract: contract}, TestWETH9Transactor: TestWETH9Transactor{contract: contract}, TestWETH9Filterer: TestWETH9Filterer{contract: contract}}, nil
}

// NewTestWETH9Caller creates a new read-only instance of TestWETH9, bound to a specific deployed contract.
func NewTestWETH9Caller(address common.Address, caller bind.ContractCaller) (*TestWETH9Caller, error) {
	contract, err := bindTestWETH9(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TestWETH9Caller{contract: contract}, nil
}

// NewTestWETH9Transactor creates a new write-only instance of TestWETH9, bound to a specific deployed contract.
func NewTestWETH9Transactor(address common.Address, transactor bind.ContractTransactor) (*TestWETH9Transactor, error) {
	contract, err := bindTestWETH9(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TestWETH9Transactor{contract: contract}, nil
}

// NewTestWETH9Filterer creates a new log filterer instance of TestWETH9, bound to a specific deployed contract.
func NewTestWETH9Filterer(address common.Address, filterer bind.ContractFilterer) (*TestWETH9Filterer, error) {
	contract, err := bindTestWETH9(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TestWETH9Filterer{contract: contract}, nil
}

// bindTestWETH9 binds a generic wrapper to an already deployed contract.
func bindTestWETH9(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TestWETH9MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TestWETH9 *TestWETH9Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TestWETH9.Contract.TestWETH9Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TestWETH9 *TestWETH9Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TestWETH9.Contract.TestWETH9Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TestWETH9 *TestWETH9Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TestWETH9.Contract.TestWETH9Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TestWETH9 *TestWETH9CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TestWETH9.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TestWETH9 *TestWETH9TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TestWETH9.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TestWETH9 *TestWETH9TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TestWETH9.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_TestWETH9 *TestWETH9Caller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TestWETH9.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_TestWETH9 *TestWETH9Session) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _TestWETH9.Contract.Allowance(&_TestWETH9.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_TestWETH9 *TestWETH9CallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _TestWETH9.Contract.Allowance(&_TestWETH9.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_TestWETH9 *TestWETH9Caller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TestWETH9.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_TestWETH9 *TestWETH9Session) BalanceOf(account common.Address) (*big.Int, error) {
	return _TestWETH9.Contract.BalanceOf(&_TestWETH9.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_TestWETH9 *TestWETH9CallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _TestWETH9.Contract.BalanceOf(&_TestWETH9.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_TestWETH9 *TestWETH9Caller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _TestWETH9.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_TestWETH9 *TestWETH9Session) Decimals() (uint8, error) {
	return _TestWETH9.Contract.Decimals(&_TestWETH9.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_TestWETH9 *TestWETH9CallerSession) Decimals() (uint8, error) {
	return _TestWETH9.Contract.Decimals(&_TestWETH9.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TestWETH9 *TestWETH9Caller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TestWETH9.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TestWETH9 *TestWETH9Session) Name() (string, error) {
	return _TestWETH9.Contract.Name(&_TestWETH9.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TestWETH9 *TestWETH9CallerSession) Name() (string, error) {
	return _TestWETH9.Contract.Name(&_TestWETH9.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TestWETH9 *TestWETH9Caller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TestWETH9.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TestWETH9 *TestWETH9Session) Symbol() (string, error) {
	return _TestWETH9.Contract.Symbol(&_TestWETH9.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TestWETH9 *TestWETH9CallerSession) Symbol() (string, error) {
	return _TestWETH9.Contract.Symbol(&_TestWETH9.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TestWETH9 *TestWETH9Caller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TestWETH9.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TestWETH9 *TestWETH9Session) TotalSupply() (*big.Int, error) {
	return _TestWETH9.Contract.TotalSupply(&_TestWETH9.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TestWETH9 *TestWETH9CallerSession) TotalSupply() (*big.Int, error) {
	return _TestWETH9.Contract.TotalSupply(&_TestWETH9.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_TestWETH9 *TestWETH9Transactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_TestWETH9 *TestWETH9Session) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.Approve(&_TestWETH9.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_TestWETH9 *TestWETH9TransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.Approve(&_TestWETH9.TransactOpts, spender, amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_TestWETH9 *TestWETH9Transactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _TestWETH9.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_TestWETH9 *TestWETH9Session) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.DecreaseAllowance(&_TestWETH9.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_TestWETH9 *TestWETH9TransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.DecreaseAllowance(&_TestWETH9.TransactOpts, spender, subtractedValue)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_TestWETH9 *TestWETH9Transactor) Deposit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TestWETH9.contract.Transact(opts, "deposit")
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_TestWETH9 *TestWETH9Session) Deposit() (*types.Transaction, error) {
	return _TestWETH9.Contract.Deposit(&_TestWETH9.TransactOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_TestWETH9 *TestWETH9TransactorSession) Deposit() (*types.Transaction, error) {
	return _TestWETH9.Contract.Deposit(&_TestWETH9.TransactOpts)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_TestWETH9 *TestWETH9Transactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _TestWETH9.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_TestWETH9 *TestWETH9Session) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.IncreaseAllowance(&_TestWETH9.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_TestWETH9 *TestWETH9TransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.IncreaseAllowance(&_TestWETH9.TransactOpts, spender, addedValue)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_TestWETH9 *TestWETH9Transactor) Transfer(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.contract.Transact(opts, "transfer", to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_TestWETH9 *TestWETH9Session) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.Transfer(&_TestWETH9.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_TestWETH9 *TestWETH9TransactorSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.Transfer(&_TestWETH9.TransactOpts, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_TestWETH9 *TestWETH9Transactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.contract.Transact(opts, "transferFrom", from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_TestWETH9 *TestWETH9Session) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.TransferFrom(&_TestWETH9.TransactOpts, from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_TestWETH9 *TestWETH9TransactorSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.TransferFrom(&_TestWETH9.TransactOpts, from, to, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 _amount) returns()
func (_TestWETH9 *TestWETH9Transactor) Withdraw(opts *bind.TransactOpts, _amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.contract.Transact(opts, "withdraw", _amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 _amount) returns()
func (_TestWETH9 *TestWETH9Session) Withdraw(_amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.Withdraw(&_TestWETH9.TransactOpts, _amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 _amount) returns()
func (_TestWETH9 *TestWETH9TransactorSession) Withdraw(_amount *big.Int) (*types.Transaction, error) {
	return _TestWETH9.Contract.Withdraw(&_TestWETH9.TransactOpts, _amount)
}

// TestWETH9ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the TestWETH9 contract.
type TestWETH9ApprovalIterator struct {
	Event *TestWETH9Approval // Event containing the contract specifics and raw log

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
func (it *TestWETH9ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TestWETH9Approval)
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
		it.Event = new(TestWETH9Approval)
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
func (it *TestWETH9ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TestWETH9ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TestWETH9Approval represents a Approval event raised by the TestWETH9 contract.
type TestWETH9Approval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_TestWETH9 *TestWETH9Filterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*TestWETH9ApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _TestWETH9.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &TestWETH9ApprovalIterator{contract: _TestWETH9.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_TestWETH9 *TestWETH9Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *TestWETH9Approval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _TestWETH9.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TestWETH9Approval)
				if err := _TestWETH9.contract.UnpackLog(event, "Approval", log); err != nil {
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
func (_TestWETH9 *TestWETH9Filterer) ParseApproval(log types.Log) (*TestWETH9Approval, error) {
	event := new(TestWETH9Approval)
	if err := _TestWETH9.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TestWETH9TransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the TestWETH9 contract.
type TestWETH9TransferIterator struct {
	Event *TestWETH9Transfer // Event containing the contract specifics and raw log

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
func (it *TestWETH9TransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TestWETH9Transfer)
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
		it.Event = new(TestWETH9Transfer)
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
func (it *TestWETH9TransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TestWETH9TransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TestWETH9Transfer represents a Transfer event raised by the TestWETH9 contract.
type TestWETH9Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_TestWETH9 *TestWETH9Filterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*TestWETH9TransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TestWETH9.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &TestWETH9TransferIterator{contract: _TestWETH9.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_TestWETH9 *TestWETH9Filterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *TestWETH9Transfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TestWETH9.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TestWETH9Transfer)
				if err := _TestWETH9.contract.UnpackLog(event, "Transfer", log); err != nil {
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
func (_TestWETH9 *TestWETH9Filterer) ParseTransfer(log types.Log) (*TestWETH9Transfer, error) {
	event := new(TestWETH9Transfer)
	if err := _TestWETH9.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UpgradeExecutorMockMetaData contains all meta data concerning the UpgradeExecutorMock contract.
var UpgradeExecutorMockMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"TargetCallExecuted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"upgrade\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"UpgradeExecuted\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"EXECUTOR_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"upgrade\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"upgradeCallData\",\"type\":\"bytes\"}],\"name\":\"execute\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"targetCallData\",\"type\":\"bytes\"}],\"name\":\"executeCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"executors\",\"type\":\"address[]\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506001609755600054610100900460ff16158080156100365750600054600160ff909116105b80610061575061004f3061013760201b6108881760201c565b158015610061575060005460ff166001145b6100c85760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b606482015260840160405180910390fd5b6000805460ff1916600117905580156100eb576000805461ff0019166101001790555b8015610131576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b50610146565b6001600160a01b03163b151590565b61145680620001566000396000f3fe6080604052600436106100c75760003560e01c806375b238fc11610074578063a217fddf1161004e578063a217fddf14610262578063bca8c7b514610277578063d547741f1461028a57600080fd5b806375b238fc146101c857806391d14854146101fc578063946d92041461024257600080fd5b8063248a9ca3116100a5578063248a9ca3146101585780632f2ff15d1461018857806336568abe146101a857600080fd5b806301ffc9a7146100cc57806307bd0265146101015780631cff79cd14610143575b600080fd5b3480156100d857600080fd5b506100ec6100e7366004610fbd565b6102aa565b60405190151581526020015b60405180910390f35b34801561010d57600080fd5b506101357fd8aa0f3194971a2a116679f7c2090f6939c8d4e01a2a8d7e41d55e5351469e6381565b6040519081526020016100f8565b610156610151366004611062565b610343565b005b34801561016457600080fd5b50610135610173366004611108565b60009081526065602052604090206001015490565b34801561019457600080fd5b506101566101a3366004611121565b610448565b3480156101b457600080fd5b506101566101c3366004611121565b610472565b3480156101d457600080fd5b506101357fa49807205ce4d355092ef5a8a18f56e8913cf4a201fbe287825b095693c2177581565b34801561020857600080fd5b506100ec610217366004611121565b60009182526065602090815260408084206001600160a01b0393909316845291905290205460ff1690565b34801561024e57600080fd5b5061015661025d36600461114d565b6104fe565b34801561026e57600080fd5b50610135600081565b610156610285366004611062565b610773565b34801561029657600080fd5b506101566102a5366004611121565b610863565b60007fffffffff0000000000000000000000000000000000000000000000000000000082167f7965db0b00000000000000000000000000000000000000000000000000000000148061033d57507f01ffc9a7000000000000000000000000000000000000000000000000000000007fffffffff000000000000000000000000000000000000000000000000000000008316145b92915050565b7fd8aa0f3194971a2a116679f7c2090f6939c8d4e01a2a8d7e41d55e5351469e6361036d81610897565b6002609754036103c45760405162461bcd60e51b815260206004820152601f60248201527f5265656e7472616e637947756172643a207265656e7472616e742063616c6c0060448201526064015b60405180910390fd5b60026097819055506103fa826040518060600160405280603a81526020016113e7603a91396001600160a01b03861691906108a4565b50826001600160a01b03167f49f6851d1cd01a518db5bdea5cffbbe90276baa2595f74250b7472b96806302e348460405161043692919061125d565b60405180910390a25050600160975550565b60008281526065602052604090206001015461046381610897565b61046d838361099a565b505050565b6001600160a01b03811633146104f05760405162461bcd60e51b815260206004820152602f60248201527f416363657373436f6e74726f6c3a2063616e206f6e6c792072656e6f756e636560448201527f20726f6c657320666f722073656c66000000000000000000000000000000000060648201526084016103bb565b6104fa8282610a3c565b5050565b600054610100900460ff161580801561051e5750600054600160ff909116105b806105385750303b158015610538575060005460ff166001145b6105aa5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016103bb565b6000805460ff1916600117905580156105cd576000805461ff0019166101001790555b6001600160a01b0383166106235760405162461bcd60e51b815260206004820152601b60248201527f557067726164654578656375746f723a207a65726f2061646d696e000000000060448201526064016103bb565b61062b610abf565b6106557fa49807205ce4d355092ef5a8a18f56e8913cf4a201fbe287825b095693c2177580610b3e565b61069f7fd8aa0f3194971a2a116679f7c2090f6939c8d4e01a2a8d7e41d55e5351469e637fa49807205ce4d355092ef5a8a18f56e8913cf4a201fbe287825b095693c21775610b3e565b6106c97fa49807205ce4d355092ef5a8a18f56e8913cf4a201fbe287825b095693c2177584610b89565b60005b8251811015610728576107187fd8aa0f3194971a2a116679f7c2090f6939c8d4e01a2a8d7e41d55e5351469e6384838151811061070b5761070b61127e565b6020026020010151610b89565b610721816112aa565b90506106cc565b50801561046d576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a1505050565b7fd8aa0f3194971a2a116679f7c2090f6939c8d4e01a2a8d7e41d55e5351469e6361079d81610897565b6002609754036107ef5760405162461bcd60e51b815260206004820152601f60248201527f5265656e7472616e637947756172643a207265656e7472616e742063616c6c0060448201526064016103bb565b600260978190555061082782346040518060600160405280603181526020016113b6603191396001600160a01b038716929190610b93565b50826001600160a01b03167f4d7dbdcc249630ec373f584267f10abf44938de920c32562f5aee93959c25258348460405161043692919061125d565b60008281526065602052604090206001015461087e81610897565b61046d8383610a3c565b6001600160a01b03163b151590565b6108a18133610cdb565b50565b60606001600160a01b0384163b6109235760405162461bcd60e51b815260206004820152602660248201527f416464726573733a2064656c65676174652063616c6c20746f206e6f6e2d636f60448201527f6e7472616374000000000000000000000000000000000000000000000000000060648201526084016103bb565b600080856001600160a01b03168560405161093e91906112c4565b600060405180830381855af49150503d8060008114610979576040519150601f19603f3d011682016040523d82523d6000602084013e61097e565b606091505b509150915061098e828286610d5b565b925050505b9392505050565b60008281526065602090815260408083206001600160a01b038516845290915290205460ff166104fa5760008281526065602090815260408083206001600160a01b03851684529091529020805460ff191660011790556109f83390565b6001600160a01b0316816001600160a01b0316837f2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d60405160405180910390a45050565b60008281526065602090815260408083206001600160a01b038516845290915290205460ff16156104fa5760008281526065602090815260408083206001600160a01b0385168085529252808320805460ff1916905551339285917ff6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b9190a45050565b600054610100900460ff16610b3c5760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016103bb565b565b600082815260656020526040808220600101805490849055905190918391839186917fbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff9190a4505050565b6104fa828261099a565b606082471015610c0b5760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c000000000000000000000000000000000000000000000000000060648201526084016103bb565b6001600160a01b0385163b610c625760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e747261637400000060448201526064016103bb565b600080866001600160a01b03168587604051610c7e91906112c4565b60006040518083038185875af1925050503d8060008114610cbb576040519150601f19603f3d011682016040523d82523d6000602084013e610cc0565b606091505b5091509150610cd0828286610d5b565b979650505050505050565b60008281526065602090815260408083206001600160a01b038516845290915290205460ff166104fa57610d19816001600160a01b03166014610d94565b610d24836020610d94565b604051602001610d359291906112e0565b60408051601f198184030181529082905262461bcd60e51b82526103bb91600401611361565b60608315610d6a575081610993565b825115610d7a5782518084602001fd5b8160405162461bcd60e51b81526004016103bb9190611361565b60606000610da3836002611374565b610dae90600261138b565b67ffffffffffffffff811115610dc657610dc661101b565b6040519080825280601f01601f191660200182016040528015610df0576020820181803683370190505b5090507f300000000000000000000000000000000000000000000000000000000000000081600081518110610e2757610e2761127e565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053507f780000000000000000000000000000000000000000000000000000000000000081600181518110610e8a57610e8a61127e565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053506000610ec6846002611374565b610ed190600161138b565b90505b6001811115610f6e577f303132333435363738396162636465660000000000000000000000000000000085600f1660108110610f1257610f1261127e565b1a60f81b828281518110610f2857610f2861127e565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a90535060049490941c93610f678161139e565b9050610ed4565b5083156109935760405162461bcd60e51b815260206004820181905260248201527f537472696e67733a20686578206c656e67746820696e73756666696369656e7460448201526064016103bb565b600060208284031215610fcf57600080fd5b81357fffffffff000000000000000000000000000000000000000000000000000000008116811461099357600080fd5b80356001600160a01b038116811461101657600080fd5b919050565b634e487b7160e01b600052604160045260246000fd5b604051601f8201601f1916810167ffffffffffffffff8111828210171561105a5761105a61101b565b604052919050565b6000806040838503121561107557600080fd5b61107e83610fff565b915060208084013567ffffffffffffffff8082111561109c57600080fd5b818601915086601f8301126110b057600080fd5b8135818111156110c2576110c261101b565b6110d484601f19601f84011601611031565b915080825287848285010111156110ea57600080fd5b80848401858401376000848284010152508093505050509250929050565b60006020828403121561111a57600080fd5b5035919050565b6000806040838503121561113457600080fd5b8235915061114460208401610fff565b90509250929050565b6000806040838503121561116057600080fd5b61116983610fff565b915060208084013567ffffffffffffffff8082111561118757600080fd5b818601915086601f83011261119b57600080fd5b8135818111156111ad576111ad61101b565b8060051b91506111be848301611031565b81815291830184019184810190898411156111d857600080fd5b938501935b838510156111fd576111ee85610fff565b825293850193908501906111dd565b8096505050505050509250929050565b60005b83811015611228578181015183820152602001611210565b50506000910152565b6000815180845261124981602086016020860161120d565b601f01601f19169290920160200192915050565b8281526040602082015260006112766040830184611231565b949350505050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b600060001982036112bd576112bd611294565b5060010190565b600082516112d681846020870161120d565b9190910192915050565b7f416363657373436f6e74726f6c3a206163636f756e742000000000000000000081526000835161131881601785016020880161120d565b7f206973206d697373696e6720726f6c6520000000000000000000000000000000601791840191820152835161135581602884016020880161120d565b01602801949350505050565b6020815260006109936020830184611231565b808202811582820484141761033d5761033d611294565b8082018082111561033d5761033d611294565b6000816113ad576113ad611294565b50600019019056fe557067726164654578656375746f723a20696e6e65722063616c6c206661696c656420776974686f757420726561736f6e557067726164654578656375746f723a20696e6e65722064656c65676174652063616c6c206661696c656420776974686f757420726561736f6ea2646970667358221220d53b0fa8acd9ad301d9b731974463329347be3d4fd5442f7c49359b5c3967c4064736f6c63430008110033",
}

// UpgradeExecutorMockABI is the input ABI used to generate the binding from.
// Deprecated: Use UpgradeExecutorMockMetaData.ABI instead.
var UpgradeExecutorMockABI = UpgradeExecutorMockMetaData.ABI

// UpgradeExecutorMockBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use UpgradeExecutorMockMetaData.Bin instead.
var UpgradeExecutorMockBin = UpgradeExecutorMockMetaData.Bin

// DeployUpgradeExecutorMock deploys a new Ethereum contract, binding an instance of UpgradeExecutorMock to it.
func DeployUpgradeExecutorMock(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *UpgradeExecutorMock, error) {
	parsed, err := UpgradeExecutorMockMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(UpgradeExecutorMockBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &UpgradeExecutorMock{UpgradeExecutorMockCaller: UpgradeExecutorMockCaller{contract: contract}, UpgradeExecutorMockTransactor: UpgradeExecutorMockTransactor{contract: contract}, UpgradeExecutorMockFilterer: UpgradeExecutorMockFilterer{contract: contract}}, nil
}

// UpgradeExecutorMock is an auto generated Go binding around an Ethereum contract.
type UpgradeExecutorMock struct {
	UpgradeExecutorMockCaller     // Read-only binding to the contract
	UpgradeExecutorMockTransactor // Write-only binding to the contract
	UpgradeExecutorMockFilterer   // Log filterer for contract events
}

// UpgradeExecutorMockCaller is an auto generated read-only Go binding around an Ethereum contract.
type UpgradeExecutorMockCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UpgradeExecutorMockTransactor is an auto generated write-only Go binding around an Ethereum contract.
type UpgradeExecutorMockTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UpgradeExecutorMockFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type UpgradeExecutorMockFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UpgradeExecutorMockSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type UpgradeExecutorMockSession struct {
	Contract     *UpgradeExecutorMock // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// UpgradeExecutorMockCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type UpgradeExecutorMockCallerSession struct {
	Contract *UpgradeExecutorMockCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// UpgradeExecutorMockTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type UpgradeExecutorMockTransactorSession struct {
	Contract     *UpgradeExecutorMockTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// UpgradeExecutorMockRaw is an auto generated low-level Go binding around an Ethereum contract.
type UpgradeExecutorMockRaw struct {
	Contract *UpgradeExecutorMock // Generic contract binding to access the raw methods on
}

// UpgradeExecutorMockCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type UpgradeExecutorMockCallerRaw struct {
	Contract *UpgradeExecutorMockCaller // Generic read-only contract binding to access the raw methods on
}

// UpgradeExecutorMockTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type UpgradeExecutorMockTransactorRaw struct {
	Contract *UpgradeExecutorMockTransactor // Generic write-only contract binding to access the raw methods on
}

// NewUpgradeExecutorMock creates a new instance of UpgradeExecutorMock, bound to a specific deployed contract.
func NewUpgradeExecutorMock(address common.Address, backend bind.ContractBackend) (*UpgradeExecutorMock, error) {
	contract, err := bindUpgradeExecutorMock(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &UpgradeExecutorMock{UpgradeExecutorMockCaller: UpgradeExecutorMockCaller{contract: contract}, UpgradeExecutorMockTransactor: UpgradeExecutorMockTransactor{contract: contract}, UpgradeExecutorMockFilterer: UpgradeExecutorMockFilterer{contract: contract}}, nil
}

// NewUpgradeExecutorMockCaller creates a new read-only instance of UpgradeExecutorMock, bound to a specific deployed contract.
func NewUpgradeExecutorMockCaller(address common.Address, caller bind.ContractCaller) (*UpgradeExecutorMockCaller, error) {
	contract, err := bindUpgradeExecutorMock(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &UpgradeExecutorMockCaller{contract: contract}, nil
}

// NewUpgradeExecutorMockTransactor creates a new write-only instance of UpgradeExecutorMock, bound to a specific deployed contract.
func NewUpgradeExecutorMockTransactor(address common.Address, transactor bind.ContractTransactor) (*UpgradeExecutorMockTransactor, error) {
	contract, err := bindUpgradeExecutorMock(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &UpgradeExecutorMockTransactor{contract: contract}, nil
}

// NewUpgradeExecutorMockFilterer creates a new log filterer instance of UpgradeExecutorMock, bound to a specific deployed contract.
func NewUpgradeExecutorMockFilterer(address common.Address, filterer bind.ContractFilterer) (*UpgradeExecutorMockFilterer, error) {
	contract, err := bindUpgradeExecutorMock(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &UpgradeExecutorMockFilterer{contract: contract}, nil
}

// bindUpgradeExecutorMock binds a generic wrapper to an already deployed contract.
func bindUpgradeExecutorMock(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := UpgradeExecutorMockMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_UpgradeExecutorMock *UpgradeExecutorMockRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _UpgradeExecutorMock.Contract.UpgradeExecutorMockCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_UpgradeExecutorMock *UpgradeExecutorMockRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.UpgradeExecutorMockTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_UpgradeExecutorMock *UpgradeExecutorMockRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.UpgradeExecutorMockTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_UpgradeExecutorMock *UpgradeExecutorMockCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _UpgradeExecutorMock.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.contract.Transact(opts, method, params...)
}

// ADMINROLE is a free data retrieval call binding the contract method 0x75b238fc.
//
// Solidity: function ADMIN_ROLE() view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockCaller) ADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _UpgradeExecutorMock.contract.Call(opts, &out, "ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ADMINROLE is a free data retrieval call binding the contract method 0x75b238fc.
//
// Solidity: function ADMIN_ROLE() view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) ADMINROLE() ([32]byte, error) {
	return _UpgradeExecutorMock.Contract.ADMINROLE(&_UpgradeExecutorMock.CallOpts)
}

// ADMINROLE is a free data retrieval call binding the contract method 0x75b238fc.
//
// Solidity: function ADMIN_ROLE() view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockCallerSession) ADMINROLE() ([32]byte, error) {
	return _UpgradeExecutorMock.Contract.ADMINROLE(&_UpgradeExecutorMock.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _UpgradeExecutorMock.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _UpgradeExecutorMock.Contract.DEFAULTADMINROLE(&_UpgradeExecutorMock.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _UpgradeExecutorMock.Contract.DEFAULTADMINROLE(&_UpgradeExecutorMock.CallOpts)
}

// EXECUTORROLE is a free data retrieval call binding the contract method 0x07bd0265.
//
// Solidity: function EXECUTOR_ROLE() view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockCaller) EXECUTORROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _UpgradeExecutorMock.contract.Call(opts, &out, "EXECUTOR_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// EXECUTORROLE is a free data retrieval call binding the contract method 0x07bd0265.
//
// Solidity: function EXECUTOR_ROLE() view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) EXECUTORROLE() ([32]byte, error) {
	return _UpgradeExecutorMock.Contract.EXECUTORROLE(&_UpgradeExecutorMock.CallOpts)
}

// EXECUTORROLE is a free data retrieval call binding the contract method 0x07bd0265.
//
// Solidity: function EXECUTOR_ROLE() view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockCallerSession) EXECUTORROLE() ([32]byte, error) {
	return _UpgradeExecutorMock.Contract.EXECUTORROLE(&_UpgradeExecutorMock.CallOpts)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _UpgradeExecutorMock.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _UpgradeExecutorMock.Contract.GetRoleAdmin(&_UpgradeExecutorMock.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_UpgradeExecutorMock *UpgradeExecutorMockCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _UpgradeExecutorMock.Contract.GetRoleAdmin(&_UpgradeExecutorMock.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_UpgradeExecutorMock *UpgradeExecutorMockCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _UpgradeExecutorMock.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _UpgradeExecutorMock.Contract.HasRole(&_UpgradeExecutorMock.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_UpgradeExecutorMock *UpgradeExecutorMockCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _UpgradeExecutorMock.Contract.HasRole(&_UpgradeExecutorMock.CallOpts, role, account)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_UpgradeExecutorMock *UpgradeExecutorMockCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _UpgradeExecutorMock.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _UpgradeExecutorMock.Contract.SupportsInterface(&_UpgradeExecutorMock.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_UpgradeExecutorMock *UpgradeExecutorMockCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _UpgradeExecutorMock.Contract.SupportsInterface(&_UpgradeExecutorMock.CallOpts, interfaceId)
}

// Execute is a paid mutator transaction binding the contract method 0x1cff79cd.
//
// Solidity: function execute(address upgrade, bytes upgradeCallData) payable returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactor) Execute(opts *bind.TransactOpts, upgrade common.Address, upgradeCallData []byte) (*types.Transaction, error) {
	return _UpgradeExecutorMock.contract.Transact(opts, "execute", upgrade, upgradeCallData)
}

// Execute is a paid mutator transaction binding the contract method 0x1cff79cd.
//
// Solidity: function execute(address upgrade, bytes upgradeCallData) payable returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) Execute(upgrade common.Address, upgradeCallData []byte) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.Execute(&_UpgradeExecutorMock.TransactOpts, upgrade, upgradeCallData)
}

// Execute is a paid mutator transaction binding the contract method 0x1cff79cd.
//
// Solidity: function execute(address upgrade, bytes upgradeCallData) payable returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactorSession) Execute(upgrade common.Address, upgradeCallData []byte) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.Execute(&_UpgradeExecutorMock.TransactOpts, upgrade, upgradeCallData)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0xbca8c7b5.
//
// Solidity: function executeCall(address target, bytes targetCallData) payable returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactor) ExecuteCall(opts *bind.TransactOpts, target common.Address, targetCallData []byte) (*types.Transaction, error) {
	return _UpgradeExecutorMock.contract.Transact(opts, "executeCall", target, targetCallData)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0xbca8c7b5.
//
// Solidity: function executeCall(address target, bytes targetCallData) payable returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) ExecuteCall(target common.Address, targetCallData []byte) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.ExecuteCall(&_UpgradeExecutorMock.TransactOpts, target, targetCallData)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0xbca8c7b5.
//
// Solidity: function executeCall(address target, bytes targetCallData) payable returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactorSession) ExecuteCall(target common.Address, targetCallData []byte) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.ExecuteCall(&_UpgradeExecutorMock.TransactOpts, target, targetCallData)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.GrantRole(&_UpgradeExecutorMock.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.GrantRole(&_UpgradeExecutorMock.TransactOpts, role, account)
}

// Initialize is a paid mutator transaction binding the contract method 0x946d9204.
//
// Solidity: function initialize(address admin, address[] executors) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactor) Initialize(opts *bind.TransactOpts, admin common.Address, executors []common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.contract.Transact(opts, "initialize", admin, executors)
}

// Initialize is a paid mutator transaction binding the contract method 0x946d9204.
//
// Solidity: function initialize(address admin, address[] executors) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) Initialize(admin common.Address, executors []common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.Initialize(&_UpgradeExecutorMock.TransactOpts, admin, executors)
}

// Initialize is a paid mutator transaction binding the contract method 0x946d9204.
//
// Solidity: function initialize(address admin, address[] executors) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactorSession) Initialize(admin common.Address, executors []common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.Initialize(&_UpgradeExecutorMock.TransactOpts, admin, executors)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.RenounceRole(&_UpgradeExecutorMock.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.RenounceRole(&_UpgradeExecutorMock.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.RevokeRole(&_UpgradeExecutorMock.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_UpgradeExecutorMock *UpgradeExecutorMockTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _UpgradeExecutorMock.Contract.RevokeRole(&_UpgradeExecutorMock.TransactOpts, role, account)
}

// UpgradeExecutorMockInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockInitializedIterator struct {
	Event *UpgradeExecutorMockInitialized // Event containing the contract specifics and raw log

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
func (it *UpgradeExecutorMockInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UpgradeExecutorMockInitialized)
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
		it.Event = new(UpgradeExecutorMockInitialized)
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
func (it *UpgradeExecutorMockInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UpgradeExecutorMockInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UpgradeExecutorMockInitialized represents a Initialized event raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) FilterInitialized(opts *bind.FilterOpts) (*UpgradeExecutorMockInitializedIterator, error) {

	logs, sub, err := _UpgradeExecutorMock.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &UpgradeExecutorMockInitializedIterator{contract: _UpgradeExecutorMock.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *UpgradeExecutorMockInitialized) (event.Subscription, error) {

	logs, sub, err := _UpgradeExecutorMock.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UpgradeExecutorMockInitialized)
				if err := _UpgradeExecutorMock.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) ParseInitialized(log types.Log) (*UpgradeExecutorMockInitialized, error) {
	event := new(UpgradeExecutorMockInitialized)
	if err := _UpgradeExecutorMock.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UpgradeExecutorMockRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockRoleAdminChangedIterator struct {
	Event *UpgradeExecutorMockRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *UpgradeExecutorMockRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UpgradeExecutorMockRoleAdminChanged)
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
		it.Event = new(UpgradeExecutorMockRoleAdminChanged)
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
func (it *UpgradeExecutorMockRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UpgradeExecutorMockRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UpgradeExecutorMockRoleAdminChanged represents a RoleAdminChanged event raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*UpgradeExecutorMockRoleAdminChangedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _UpgradeExecutorMock.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &UpgradeExecutorMockRoleAdminChangedIterator{contract: _UpgradeExecutorMock.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *UpgradeExecutorMockRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _UpgradeExecutorMock.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UpgradeExecutorMockRoleAdminChanged)
				if err := _UpgradeExecutorMock.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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

// ParseRoleAdminChanged is a log parse operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) ParseRoleAdminChanged(log types.Log) (*UpgradeExecutorMockRoleAdminChanged, error) {
	event := new(UpgradeExecutorMockRoleAdminChanged)
	if err := _UpgradeExecutorMock.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UpgradeExecutorMockRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockRoleGrantedIterator struct {
	Event *UpgradeExecutorMockRoleGranted // Event containing the contract specifics and raw log

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
func (it *UpgradeExecutorMockRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UpgradeExecutorMockRoleGranted)
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
		it.Event = new(UpgradeExecutorMockRoleGranted)
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
func (it *UpgradeExecutorMockRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UpgradeExecutorMockRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UpgradeExecutorMockRoleGranted represents a RoleGranted event raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*UpgradeExecutorMockRoleGrantedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _UpgradeExecutorMock.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &UpgradeExecutorMockRoleGrantedIterator{contract: _UpgradeExecutorMock.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *UpgradeExecutorMockRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _UpgradeExecutorMock.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UpgradeExecutorMockRoleGranted)
				if err := _UpgradeExecutorMock.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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

// ParseRoleGranted is a log parse operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) ParseRoleGranted(log types.Log) (*UpgradeExecutorMockRoleGranted, error) {
	event := new(UpgradeExecutorMockRoleGranted)
	if err := _UpgradeExecutorMock.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UpgradeExecutorMockRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockRoleRevokedIterator struct {
	Event *UpgradeExecutorMockRoleRevoked // Event containing the contract specifics and raw log

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
func (it *UpgradeExecutorMockRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UpgradeExecutorMockRoleRevoked)
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
		it.Event = new(UpgradeExecutorMockRoleRevoked)
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
func (it *UpgradeExecutorMockRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UpgradeExecutorMockRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UpgradeExecutorMockRoleRevoked represents a RoleRevoked event raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*UpgradeExecutorMockRoleRevokedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _UpgradeExecutorMock.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &UpgradeExecutorMockRoleRevokedIterator{contract: _UpgradeExecutorMock.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *UpgradeExecutorMockRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _UpgradeExecutorMock.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UpgradeExecutorMockRoleRevoked)
				if err := _UpgradeExecutorMock.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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

// ParseRoleRevoked is a log parse operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) ParseRoleRevoked(log types.Log) (*UpgradeExecutorMockRoleRevoked, error) {
	event := new(UpgradeExecutorMockRoleRevoked)
	if err := _UpgradeExecutorMock.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UpgradeExecutorMockTargetCallExecutedIterator is returned from FilterTargetCallExecuted and is used to iterate over the raw logs and unpacked data for TargetCallExecuted events raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockTargetCallExecutedIterator struct {
	Event *UpgradeExecutorMockTargetCallExecuted // Event containing the contract specifics and raw log

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
func (it *UpgradeExecutorMockTargetCallExecutedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UpgradeExecutorMockTargetCallExecuted)
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
		it.Event = new(UpgradeExecutorMockTargetCallExecuted)
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
func (it *UpgradeExecutorMockTargetCallExecutedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UpgradeExecutorMockTargetCallExecutedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UpgradeExecutorMockTargetCallExecuted represents a TargetCallExecuted event raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockTargetCallExecuted struct {
	Target common.Address
	Value  *big.Int
	Data   []byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTargetCallExecuted is a free log retrieval operation binding the contract event 0x4d7dbdcc249630ec373f584267f10abf44938de920c32562f5aee93959c25258.
//
// Solidity: event TargetCallExecuted(address indexed target, uint256 value, bytes data)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) FilterTargetCallExecuted(opts *bind.FilterOpts, target []common.Address) (*UpgradeExecutorMockTargetCallExecutedIterator, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _UpgradeExecutorMock.contract.FilterLogs(opts, "TargetCallExecuted", targetRule)
	if err != nil {
		return nil, err
	}
	return &UpgradeExecutorMockTargetCallExecutedIterator{contract: _UpgradeExecutorMock.contract, event: "TargetCallExecuted", logs: logs, sub: sub}, nil
}

// WatchTargetCallExecuted is a free log subscription operation binding the contract event 0x4d7dbdcc249630ec373f584267f10abf44938de920c32562f5aee93959c25258.
//
// Solidity: event TargetCallExecuted(address indexed target, uint256 value, bytes data)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) WatchTargetCallExecuted(opts *bind.WatchOpts, sink chan<- *UpgradeExecutorMockTargetCallExecuted, target []common.Address) (event.Subscription, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _UpgradeExecutorMock.contract.WatchLogs(opts, "TargetCallExecuted", targetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UpgradeExecutorMockTargetCallExecuted)
				if err := _UpgradeExecutorMock.contract.UnpackLog(event, "TargetCallExecuted", log); err != nil {
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

// ParseTargetCallExecuted is a log parse operation binding the contract event 0x4d7dbdcc249630ec373f584267f10abf44938de920c32562f5aee93959c25258.
//
// Solidity: event TargetCallExecuted(address indexed target, uint256 value, bytes data)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) ParseTargetCallExecuted(log types.Log) (*UpgradeExecutorMockTargetCallExecuted, error) {
	event := new(UpgradeExecutorMockTargetCallExecuted)
	if err := _UpgradeExecutorMock.contract.UnpackLog(event, "TargetCallExecuted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UpgradeExecutorMockUpgradeExecutedIterator is returned from FilterUpgradeExecuted and is used to iterate over the raw logs and unpacked data for UpgradeExecuted events raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockUpgradeExecutedIterator struct {
	Event *UpgradeExecutorMockUpgradeExecuted // Event containing the contract specifics and raw log

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
func (it *UpgradeExecutorMockUpgradeExecutedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UpgradeExecutorMockUpgradeExecuted)
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
		it.Event = new(UpgradeExecutorMockUpgradeExecuted)
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
func (it *UpgradeExecutorMockUpgradeExecutedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UpgradeExecutorMockUpgradeExecutedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UpgradeExecutorMockUpgradeExecuted represents a UpgradeExecuted event raised by the UpgradeExecutorMock contract.
type UpgradeExecutorMockUpgradeExecuted struct {
	Upgrade common.Address
	Value   *big.Int
	Data    []byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUpgradeExecuted is a free log retrieval operation binding the contract event 0x49f6851d1cd01a518db5bdea5cffbbe90276baa2595f74250b7472b96806302e.
//
// Solidity: event UpgradeExecuted(address indexed upgrade, uint256 value, bytes data)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) FilterUpgradeExecuted(opts *bind.FilterOpts, upgrade []common.Address) (*UpgradeExecutorMockUpgradeExecutedIterator, error) {

	var upgradeRule []interface{}
	for _, upgradeItem := range upgrade {
		upgradeRule = append(upgradeRule, upgradeItem)
	}

	logs, sub, err := _UpgradeExecutorMock.contract.FilterLogs(opts, "UpgradeExecuted", upgradeRule)
	if err != nil {
		return nil, err
	}
	return &UpgradeExecutorMockUpgradeExecutedIterator{contract: _UpgradeExecutorMock.contract, event: "UpgradeExecuted", logs: logs, sub: sub}, nil
}

// WatchUpgradeExecuted is a free log subscription operation binding the contract event 0x49f6851d1cd01a518db5bdea5cffbbe90276baa2595f74250b7472b96806302e.
//
// Solidity: event UpgradeExecuted(address indexed upgrade, uint256 value, bytes data)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) WatchUpgradeExecuted(opts *bind.WatchOpts, sink chan<- *UpgradeExecutorMockUpgradeExecuted, upgrade []common.Address) (event.Subscription, error) {

	var upgradeRule []interface{}
	for _, upgradeItem := range upgrade {
		upgradeRule = append(upgradeRule, upgradeItem)
	}

	logs, sub, err := _UpgradeExecutorMock.contract.WatchLogs(opts, "UpgradeExecuted", upgradeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UpgradeExecutorMockUpgradeExecuted)
				if err := _UpgradeExecutorMock.contract.UnpackLog(event, "UpgradeExecuted", log); err != nil {
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

// ParseUpgradeExecuted is a log parse operation binding the contract event 0x49f6851d1cd01a518db5bdea5cffbbe90276baa2595f74250b7472b96806302e.
//
// Solidity: event UpgradeExecuted(address indexed upgrade, uint256 value, bytes data)
func (_UpgradeExecutorMock *UpgradeExecutorMockFilterer) ParseUpgradeExecuted(log types.Log) (*UpgradeExecutorMockUpgradeExecuted, error) {
	event := new(UpgradeExecutorMockUpgradeExecuted)
	if err := _UpgradeExecutorMock.contract.UnpackLog(event, "UpgradeExecuted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
