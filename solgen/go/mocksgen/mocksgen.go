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

// ISequencerInboxMaxTimeVariation is an auto generated low-level Go binding around an user-defined struct.
type ISequencerInboxMaxTimeVariation struct {
	DelayBlocks   *big.Int
	FutureBlocks  *big.Int
	DelaySeconds  *big.Int
	FutureSeconds *big.Int
}

// ISequencerInboxTimeBounds is an auto generated low-level Go binding around an user-defined struct.
type ISequencerInboxTimeBounds struct {
	MinTimestamp   uint64
	MaxTimestamp   uint64
	MinBlockNumber uint64
	MaxBlockNumber uint64
}

// OldChallengeLibChallenge is an auto generated low-level Go binding around an user-defined struct.
type OldChallengeLibChallenge struct {
	Current            OldChallengeLibParticipant
	Next               OldChallengeLibParticipant
	LastMoveTimestamp  *big.Int
	WasmModuleRoot     [32]byte
	ChallengeStateHash [32]byte
	MaxInboxMessages   uint64
	Mode               uint8
}

// OldChallengeLibParticipant is an auto generated low-level Go binding around an user-defined struct.
type OldChallengeLibParticipant struct {
	Addr     common.Address
	TimeLeft *big.Int
}

// OldChallengeLibSegmentSelection is an auto generated low-level Go binding around an user-defined struct.
type OldChallengeLibSegmentSelection struct {
	OldSegmentsStart  *big.Int
	OldSegmentsLength *big.Int
	OldSegments       [][32]byte
	ChallengePosition *big.Int
}

// BridgeStubMetaData contains all meta data concerning the BridgeStub contract.
var BridgeStubMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"stored\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"received\",\"type\":\"uint256\"}],\"name\":\"BadSequencerMessageNumber\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"BridgeCallTriggered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"InboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeInboxAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"MessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"OutboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newSequencerInbox\",\"type\":\"address\"}],\"name\":\"SequencerInboxUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"acceptFundsFromOldBridge\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"activeOutbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedDelayedInboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"}],\"name\":\"allowedDelayedInboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedOutboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"allowedOutboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"delayedInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"delayedMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"enqueueDelayedMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"enqueueSequencerMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"seqMessageIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"acc\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"executeCall\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"rollup_\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"sequencerInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerReportedSubMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setDelayedInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"name\":\"setOutbox\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sequencerInbox\",\"type\":\"address\"}],\"name\":\"setSequencerInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"batchPoster\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"}],\"name\":\"submitBatchSpendingReport\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610da0806100206000396000f3fe6080604052600436106101745760003560e01c80639e5d4c49116100cb578063cee3d7281161007f578063e77145f411610059578063e77145f41461020e578063eca067ad14610465578063ee35f3271461047a57600080fd5b8063cee3d72814610406578063d5719dc214610425578063e76f5c8d1461044557600080fd5b8063ae60bd13116100b0578063ae60bd1314610362578063c4d66de81461039e578063cb23bcb5146103e857600080fd5b80639e5d4c4914610314578063ab5d89431461034257600080fd5b80635fca4a161161012d5780638db5993b116101075780638db5993b146102a9578063919cc706146102bc578063945e1147146102dc57600080fd5b80635fca4a16146102305780637a88b1071461024657806386598a561461026957600080fd5b8063413b35bd1161015e578063413b35bd146101bd57806347fb24c5146101ee5780634f61f8501461021057600080fd5b806284120c1461017957806316bf55791461019d575b600080fd5b34801561018557600080fd5b506006545b6040519081526020015b60405180910390f35b3480156101a957600080fd5b5061018a6101b8366004610b04565b61049a565b3480156101c957600080fd5b506101de6101d8366004610b35565b50600190565b6040519015158152602001610194565b3480156101fa57600080fd5b5061020e610209366004610b59565b6104bb565b005b34801561021c57600080fd5b5061020e61022b366004610b35565b61070c565b34801561023c57600080fd5b5061018a60085481565b34801561025257600080fd5b5061018a610261366004610b97565b600092915050565b34801561027557600080fd5b50610289610284366004610bc3565b61076d565b604080519485526020850193909352918301526060820152608001610194565b61018a6102b7366004610bf5565b6108c2565b3480156102c857600080fd5b5061020e6102d7366004610b35565b61093a565b3480156102e857600080fd5b506102fc6102f7366004610b04565b610982565b6040516001600160a01b039091168152602001610194565b34801561032057600080fd5b5061033461032f366004610c3c565b6109ac565b604051610194929190610cc5565b34801561034e57600080fd5b506004546102fc906001600160a01b031681565b34801561036e57600080fd5b506101de61037d366004610b35565b6001600160a01b031660009081526020819052604090206001015460ff1690565b3480156103aa57600080fd5b5061020e6103b9366004610b35565b6001805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0392909216919091179055565b3480156103f457600080fd5b506001546001600160a01b03166102fc565b34801561041257600080fd5b5061020e610421366004610b59565b5050565b34801561043157600080fd5b5061018a610440366004610b04565b6109fa565b34801561045157600080fd5b506102fc610460366004610b04565b610a0a565b34801561047157600080fd5b5060055461018a565b34801561048657600080fd5b506007546102fc906001600160a01b031681565b600681815481106104aa57600080fd5b600091825260209091200154905081565b6001600160a01b03821660008181526020818152604091829020600181015492518515158152909360ff90931692917f6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521910160405180910390a2821515811515036105265750505050565b82156105c157604080518082018252600280548252600160208084018281526001600160a01b038a166000818152928390529582209451855551938201805460ff1916941515949094179093558154908101825591527f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace01805473ffffffffffffffffffffffffffffffffffffffff19169091179055610706565b600280546105d190600190610d1d565b815481106105e1576105e1610d3e565b6000918252602090912001548254600280546001600160a01b0390931692909190811061061057610610610d3e565b9060005260206000200160006101000a8154816001600160a01b0302191690836001600160a01b031602179055508160000154600080600285600001548154811061065d5761065d610d3e565b60009182526020808320909101546001600160a01b03168352820192909252604001902055600280548061069357610693610d54565b6000828152602080822083017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff908101805473ffffffffffffffffffffffffffffffffffffffff191690559092019092556001600160a01b038616825281905260408120908155600101805460ff191690555b50505050565b6007805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0383169081179091556040519081527f8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a9060200160405180910390a150565b600080600080856008541415801561078457508515155b8015610791575060085415155b156107db576008546040517fe2051feb0000000000000000000000000000000000000000000000000000000081526004810191909152602481018790526044015b60405180910390fd5b60088590556006549350831561081957600680546107fb90600190610d1d565b8154811061080b5761080b610d3e565b906000526020600020015492505b861561084a57600561082c600189610d1d565b8154811061083c5761083c610d3e565b906000526020600020015491505b60408051602081018590529081018990526060810183905260800160408051601f198184030181529190528051602090910120600680546001810182556000919091527ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f018190559398929750909550919350915050565b3360009081526020819052604081206001015460ff166109245760405162461bcd60e51b815260206004820152600e60248201527f4e4f545f46524f4d5f494e424f5800000000000000000000000000000000000060448201526064016107d2565b610932848443424887610a1a565b949350505050565b60405162461bcd60e51b815260206004820152600f60248201527f4e4f545f494d504c454d454e544544000000000000000000000000000000000060448201526064016107d2565b6003818154811061099257600080fd5b6000918252602090912001546001600160a01b0316905081565b60405162461bcd60e51b815260206004820152601c60248201527f4e4f545f494d504c454d454e5445445f455845435554455f43414c4c0000000060448201526000906060906064016107d2565b600581815481106104aa57600080fd5b6002818154811061099257600080fd5b60055460408051600060208083018290526021830182905260358301829052603d8301829052604583018290526065830182905260858084018790528451808503909101815260a59093019093528151919092012090919060008215610aa5576005610a87600185610d1d565b81548110610a9757610a97610d3e565b906000526020600020015490505b6005610ab18284610ad5565b81546001810183556000928352602090922090910155509098975050505050505050565b604080516020808201859052818301849052825180830384018152606090920190925280519101205b92915050565b600060208284031215610b1657600080fd5b5035919050565b6001600160a01b0381168114610b3257600080fd5b50565b600060208284031215610b4757600080fd5b8135610b5281610b1d565b9392505050565b60008060408385031215610b6c57600080fd5b8235610b7781610b1d565b915060208301358015158114610b8c57600080fd5b809150509250929050565b60008060408385031215610baa57600080fd5b8235610bb581610b1d565b946020939093013593505050565b60008060008060808587031215610bd957600080fd5b5050823594602084013594506040840135936060013592509050565b600080600060608486031215610c0a57600080fd5b833560ff81168114610c1b57600080fd5b92506020840135610c2b81610b1d565b929592945050506040919091013590565b60008060008060608587031215610c5257600080fd5b8435610c5d81610b1d565b935060208501359250604085013567ffffffffffffffff80821115610c8157600080fd5b818701915087601f830112610c9557600080fd5b813581811115610ca457600080fd5b886020828501011115610cb657600080fd5b95989497505060200194505050565b821515815260006020604081840152835180604085015260005b81811015610cfb57858101830151858201606001528201610cdf565b506000606082860101526060601f19601f830116850101925050509392505050565b81810381811115610afe57634e487b7160e01b600052601160045260246000fd5b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052603160045260246000fdfea26469706673582212209afc73f0d4ec7d2c939dc494cebad63b12b3dd9a1655577d8102c76ab888aea864736f6c63430008110033",
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

// ExecuteCall is a free data retrieval call binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address , uint256 , bytes ) pure returns(bool, bytes)
func (_BridgeStub *BridgeStubCaller) ExecuteCall(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int, arg2 []byte) (bool, []byte, error) {
	var out []interface{}
	err := _BridgeStub.contract.Call(opts, &out, "executeCall", arg0, arg1, arg2)

	if err != nil {
		return *new(bool), *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new([]byte)).(*[]byte)

	return out0, out1, err

}

// ExecuteCall is a free data retrieval call binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address , uint256 , bytes ) pure returns(bool, bytes)
func (_BridgeStub *BridgeStubSession) ExecuteCall(arg0 common.Address, arg1 *big.Int, arg2 []byte) (bool, []byte, error) {
	return _BridgeStub.Contract.ExecuteCall(&_BridgeStub.CallOpts, arg0, arg1, arg2)
}

// ExecuteCall is a free data retrieval call binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address , uint256 , bytes ) pure returns(bool, bytes)
func (_BridgeStub *BridgeStubCallerSession) ExecuteCall(arg0 common.Address, arg1 *big.Int, arg2 []byte) (bool, []byte, error) {
	return _BridgeStub.Contract.ExecuteCall(&_BridgeStub.CallOpts, arg0, arg1, arg2)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
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
// Solidity: function rollup() view returns(address)
func (_BridgeStub *BridgeStubSession) Rollup() (common.Address, error) {
	return _BridgeStub.Contract.Rollup(&_BridgeStub.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
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

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_BridgeStub *BridgeStubTransactor) Initialize(opts *bind.TransactOpts, rollup_ common.Address) (*types.Transaction, error) {
	return _BridgeStub.contract.Transact(opts, "initialize", rollup_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_BridgeStub *BridgeStubSession) Initialize(rollup_ common.Address) (*types.Transaction, error) {
	return _BridgeStub.Contract.Initialize(&_BridgeStub.TransactOpts, rollup_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_BridgeStub *BridgeStubTransactorSession) Initialize(rollup_ common.Address) (*types.Transaction, error) {
	return _BridgeStub.Contract.Initialize(&_BridgeStub.TransactOpts, rollup_)
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
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"stored\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"received\",\"type\":\"uint256\"}],\"name\":\"BadSequencerMessageNumber\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"}],\"name\":\"InvalidOutboxSet\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"NotContract\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotDelayedInbox\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotOutbox\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotRollupOrOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotSequencerInbox\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"BridgeCallTriggered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"InboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeInboxAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"MessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"OutboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newSequencerInbox\",\"type\":\"address\"}],\"name\":\"SequencerInboxUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"acceptFundsFromOldBridge\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"activeOutbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedDelayedInboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"}],\"name\":\"allowedDelayedInboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedOutboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"}],\"name\":\"allowedOutboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"delayedInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"delayedMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"enqueueDelayedMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"enqueueSequencerMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"seqMessageIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"acc\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeCall\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"rollup_\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"sequencerInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerReportedSubMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setDelayedInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setOutbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sequencerInbox\",\"type\":\"address\"}],\"name\":\"setSequencerInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newMsgCount\",\"type\":\"uint256\"}],\"name\":\"setSequencerReportedSubMessageCount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"submitBatchSpendingReport\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"_rollup\",\"type\":\"address\"}],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a06040523060805234801561001457600080fd5b50600580546001600160a01b03199081166001600160a01b031790915560088054909116331790556080516119d961005e60003960008181610b660152610f9401526119d96000f3fe60806040526004361061017f5760003560e01c80639e5d4c49116100d6578063d5719dc21161007f578063eca067ad11610059578063eca067ad14610457578063ee35f3271461046c578063f81ff3b31461048c57600080fd5b8063d5719dc214610417578063e76f5c8d14610437578063e77145f41461023457600080fd5b8063c4d66de8116100b0578063c4d66de8146103b7578063cb23bcb5146103d7578063cee3d728146103f757600080fd5b80639e5d4c4914610337578063ab5d894314610365578063ae60bd131461037a57600080fd5b80635fca4a16116101385780638db5993b116101125780638db5993b146102cc578063919cc706146102df578063945e1147146102ff57600080fd5b80635fca4a16146102565780637a88b1071461026c57806386598a561461028c57600080fd5b8063413b35bd11610169578063413b35bd146101c857806347fb24c5146102145780634f61f8501461023657600080fd5b806284120c1461018457806316bf5579146101a8575b600080fd5b34801561019057600080fd5b506007545b6040519081526020015b60405180910390f35b3480156101b457600080fd5b506101956101c33660046116da565b6104ac565b3480156101d457600080fd5b506102046101e336600461170b565b6001600160a01b031660009081526002602052604090206001015460ff1690565b604051901515815260200161019f565b34801561022057600080fd5b5061023461022f36600461172f565b6104cd565b005b34801561024257600080fd5b5061023461025136600461170b565b6107d3565b34801561026257600080fd5b50610195600a5481565b34801561027857600080fd5b5061019561028736600461176d565b6108fe565b34801561029857600080fd5b506102ac6102a7366004611799565b61095f565b60408051948552602085019390935291830152606082015260800161019f565b6101956102da3660046117cb565b610af8565b3480156102eb57600080fd5b506102346102fa36600461170b565b610b5c565b34801561030b57600080fd5b5061031f61031a3660046116da565b610ca1565b6040516001600160a01b03909116815260200161019f565b34801561034357600080fd5b50610357610352366004611812565b610ccb565b60405161019f92919061189b565b34801561037157600080fd5b5061031f610e78565b34801561038657600080fd5b5061020461039536600461170b565b6001600160a01b03166000908152600160208190526040909120015460ff1690565b3480156103c357600080fd5b506102346103d236600461170b565b610ebb565b3480156103e357600080fd5b5060085461031f906001600160a01b031681565b34801561040357600080fd5b5061023461041236600461172f565b6110ab565b34801561042357600080fd5b506101956104323660046116da565b611419565b34801561044357600080fd5b5061031f6104523660046116da565b611429565b34801561046357600080fd5b50600654610195565b34801561047857600080fd5b5060095461031f906001600160a01b031681565b34801561049857600080fd5b506102346104a73660046116da565b611439565b600781815481106104bc57600080fd5b600091825260209091200154905081565b6008546001600160a01b0316331461059c5760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610529573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061054d91906118f3565b9050336001600160a01b0382161461059a57600854604051630739600760e01b81523360048201526001600160a01b03918216602482015290821660448201526064015b60405180910390fd5b505b6001600160a01b0382166000818152600160208181526040928390209182015492518515158152919360ff90931692917f6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521910160405180910390a2821515811515036106085750505050565b82156106a357604080518082018252600380548252600160208084018281526001600160a01b038a166000818152928490529582209451855551938201805460ff1916941515949094179093558154908101825591527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b01805473ffffffffffffffffffffffffffffffffffffffff191690911790556107cc565b600380546106b390600190611910565b815481106106c3576106c3611931565b6000918252602090912001548254600380546001600160a01b039093169290919081106106f2576106f2611931565b9060005260206000200160006101000a8154816001600160a01b0302191690836001600160a01b03160217905550816000015460016000600385600001548154811061074057610740611931565b60009182526020808320909101546001600160a01b03168352820192909252604001902055600380548061077657610776611947565b600082815260208082208301600019908101805473ffffffffffffffffffffffffffffffffffffffff191690559092019092556001600160a01b03861682526001908190526040822091825501805460ff191690555b50505b5050565b6008546001600160a01b0316331461089d5760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa15801561082f573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061085391906118f3565b9050336001600160a01b0382161461089b57600854604051630739600760e01b81523360048201526001600160a01b0391821660248201529082166044820152606401610591565b505b6009805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0383169081179091556040519081527f8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a9060200160405180910390a150565b6009546000906001600160a01b03163314610947576040517f88f84f04000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b610956600d8443424887611508565b90505b92915050565b6009546000908190819081906001600160a01b031633146109ae576040517f88f84f04000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b85600a54141580156109bf57508515155b80156109cc5750600a5415155b15610a1157600a546040517fe2051feb000000000000000000000000000000000000000000000000000000008152600481019190915260248101879052604401610591565b600a85905560075493508315610a4f5760078054610a3190600190611910565b81548110610a4157610a41611931565b906000526020600020015492505b8615610a80576006610a62600189611910565b81548110610a7257610a72611931565b906000526020600020015491505b60408051602081018590529081018990526060810183905260800160408051601f198184030181529190528051602090910120600780546001810182556000919091527fa66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c688018190559398929750909550919350915050565b3360009081526001602081905260408220015460ff16610b46576040517fb6c60ea3000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b610b54848443424887611508565b949350505050565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610bfa5760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c00000000000000000000000000000000000000006064820152608401610591565b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038054336001600160a01b03821614610c70576040517f23295f0e0000000000000000000000000000000000000000000000000000000081523360048201526001600160a01b0382166024820152604401610591565b50506008805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0392909216919091179055565b60048181548110610cb157600080fd5b6000918252602090912001546001600160a01b0316905081565b3360009081526002602052604081206001015460609060ff16610d1c576040517f32ea82ab000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b8215801590610d3357506001600160a01b0386163b155b15610d75576040517fb5cf5b8f0000000000000000000000000000000000000000000000000000000081526001600160a01b0387166004820152602401610591565b6005805473ffffffffffffffffffffffffffffffffffffffff19811633179091556040516001600160a01b03918216918816908790610db7908890889061195d565b60006040518083038185875af1925050503d8060008114610df4576040519150601f19603f3d011682016040523d82523d6000602084013e610df9565b606091505b506005805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038581169190911790915560405192955090935088169033907f2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d46690610e66908a908a908a9061196d565b60405180910390a35094509492505050565b6005546000906001600160a01b03167fffffffffffffffffffffffff00000000000000000000000000000000000000018101610eb657600091505090565b919050565b600054610100900460ff1615808015610edb5750600054600160ff909116105b80610ef55750303b158015610ef5575060005460ff166001145b610f675760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a65640000000000000000000000000000000000006064820152608401610591565b6000805460ff191660011790558015610f8a576000805461ff0019166101001790555b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036110285760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c00000000000000000000000000000000000000006064820152608401610591565b600580546001600160a01b0373ffffffffffffffffffffffffffffffffffffffff1991821681179092556008805490911691841691909117905580156107cf576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050565b6008546001600160a01b031633146111755760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015611107573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061112b91906118f3565b9050336001600160a01b0382161461117357600854604051630739600760e01b81523360048201526001600160a01b0391821660248201529082166044820152606401610591565b505b7fffffffffffffffffffffffff00000000000000000000000000000000000000016001600160a01b038316016111e2576040517f77abed100000000000000000000000000000000000000000000000000000000081526001600160a01b0383166004820152602401610591565b6001600160a01b038216600081815260026020908152604091829020600181015492518515158152909360ff90931692917f49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa910160405180910390a28215158115150361124f5750505050565b82156112eb57604080518082018252600480548252600160208084018281526001600160a01b038a16600081815260029093529582209451855551938201805460ff1916941515949094179093558154908101825591527f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b01805473ffffffffffffffffffffffffffffffffffffffff191690911790556107cc565b600480546112fb90600190611910565b8154811061130b5761130b611931565b6000918252602090912001548254600480546001600160a01b0390931692909190811061133a5761133a611931565b9060005260206000200160006101000a8154816001600160a01b0302191690836001600160a01b03160217905550816000015460026000600485600001548154811061138857611388611931565b60009182526020808320909101546001600160a01b0316835282019290925260400190205560048054806113be576113be611947565b600082815260208082208301600019908101805473ffffffffffffffffffffffffffffffffffffffff191690559092019092556001600160a01b03861682526002905260408120908155600101805460ff1916905550505050565b600681815481106104bc57600080fd5b60038181548110610cb157600080fd5b6008546001600160a01b031633146115035760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015611495573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114b991906118f3565b9050336001600160a01b0382161461150157600854604051630739600760e01b81523360048201526001600160a01b0391821660248201529082166044820152606401610591565b505b600a55565b600654604080517fff0000000000000000000000000000000000000000000000000000000000000060f88a901b166020808301919091527fffffffffffffffffffffffffffffffffffffffff00000000000000000000000060608a901b1660218301527fffffffffffffffff00000000000000000000000000000000000000000000000060c089811b8216603585015288901b16603d830152604582018490526065820186905260858083018690528351808403909101815260a5909201909252805191012060009190600082156116055760066115e7600185611910565b815481106115f7576115f7611931565b906000526020600020015490505b6040805160208082018490528183018590528251808303840181526060830180855281519190920120600680546001810182556000919091527ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f015533905260ff8c1660808201526001600160a01b038b1660a082015260c0810187905260e0810188905267ffffffffffffffff89166101008201529051829185917f5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1918190036101200190a3509098975050505050505050565b6000602082840312156116ec57600080fd5b5035919050565b6001600160a01b038116811461170857600080fd5b50565b60006020828403121561171d57600080fd5b8135611728816116f3565b9392505050565b6000806040838503121561174257600080fd5b823561174d816116f3565b91506020830135801515811461176257600080fd5b809150509250929050565b6000806040838503121561178057600080fd5b823561178b816116f3565b946020939093013593505050565b600080600080608085870312156117af57600080fd5b5050823594602084013594506040840135936060013592509050565b6000806000606084860312156117e057600080fd5b833560ff811681146117f157600080fd5b92506020840135611801816116f3565b929592945050506040919091013590565b6000806000806060858703121561182857600080fd5b8435611833816116f3565b935060208501359250604085013567ffffffffffffffff8082111561185757600080fd5b818701915087601f83011261186b57600080fd5b81358181111561187a57600080fd5b88602082850101111561188c57600080fd5b95989497505060200194505050565b821515815260006020604081840152835180604085015260005b818110156118d1578581018301518582016060015282016118b5565b506000606082860101526060601f19601f830116850101925050509392505050565b60006020828403121561190557600080fd5b8151611728816116f3565b8181038181111561095957634e487b7160e01b600052601160045260246000fd5b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052603160045260246000fd5b8183823760009101908152919050565b83815260406020820152816040820152818360608301376000818301606090810191909152601f909201601f191601019291505056fea2646970667358221220141a4ccb62f810380b79eaa024f3f8acbf2f1a6f2416a607c2f42eac3755735364736f6c63430008110033",
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
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"calculateRetryableSubmissionFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"createRetryableTicket\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"depositEth\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"_bridge\",\"type\":\"address\"},{\"internalType\":\"contractISequencerInbox\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxDataSize\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"_bridge\",\"type\":\"address\"}],\"name\":\"postUpgradeInit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendContractTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedContractTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedUnsignedTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedUnsignedTransactionToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"messageData\",\"type\":\"bytes\"}],\"name\":\"sendL2Message\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"messageData\",\"type\":\"bytes\"}],\"name\":\"sendL2MessageFromOrigin\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendUnsignedTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"sendUnsignedTransactionToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"sendWithdrawEthToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"contractISequencerInbox\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"unsafeCreateRetryableTicket\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b506201cccc608052608051610b25610033600039600061035f0152610b256000f3fe6080604052600436106101695760003560e01c80636e6e8a6a116100cb578063b75436bb1161007f578063e78cea9211610059578063e78cea92146102fb578063e8eb1dc31461034d578063ee35f3271461038157600080fd5b8063b75436bb146102bd578063c474d2c5146102dd578063e6bd12cf1461025e57600080fd5b80638456cb59116100b05780638456cb59146101c15780638a631aa614610287578063a66b327d146102a257600080fd5b80636e6e8a6a1461025057806370665f141461026c57600080fd5b80635075788b116101225780635e916758116101075780635e91675814610242578063679b6ded1461025057806367ef3ab81461025e57600080fd5b80635075788b1461016e5780635c975abb1461020057600080fd5b80633f4ba83a116101535780633f4ba83a146101c1578063439370b1146101d8578063485cc955146101e057600080fd5b8062f723821461016e5780631fe927cf146101a1575b600080fd5b34801561017a57600080fd5b5061018e610189366004610717565b6103ae565b6040519081526020015b60405180910390f35b3480156101ad57600080fd5b5061018e6101bc366004610794565b6103fe565b3480156101cd57600080fd5b506101d66104a9565b005b61018e6103ae565b3480156101ec57600080fd5b506101d66101fb3660046107d6565b6104f1565b34801561020c57600080fd5b506001546102329074010000000000000000000000000000000000000000900460ff1681565b6040519015158152602001610198565b61018e61018936600461080f565b61018e610189366004610879565b61018e61018936600461091e565b34801561027857600080fd5b5061018e610189366004610991565b34801561029357600080fd5b5061018e6101893660046109de565b3480156102ae57600080fd5b5061018e610189366004610a33565b3480156102c957600080fd5b5061018e6102d8366004610794565b61059f565b3480156102e957600080fd5b506101d66102f8366004610a55565b50565b34801561030757600080fd5b506000546103289073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610198565b34801561035957600080fd5b5061018e7f000000000000000000000000000000000000000000000000000000000000000081565b34801561038d57600080fd5b506001546103289073ffffffffffffffffffffffffffffffffffffffff1681565b60405162461bcd60e51b815260206004820152600f60248201527f4e4f545f494d504c454d454e544544000000000000000000000000000000000060448201526000906064015b60405180910390fd5b600033321461044f5760405162461bcd60e51b815260206004820152600b60248201527f6f726967696e206f6e6c7900000000000000000000000000000000000000000060448201526064016103f5565b60006104746003338686604051610467929190610a79565b60405180910390206105fb565b60405190915081907fab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c90600090a29392505050565b60405162461bcd60e51b815260206004820152600f60248201527f4e4f5420494d504c454d454e544544000000000000000000000000000000000060448201526064016103f5565b60005473ffffffffffffffffffffffffffffffffffffffff16156105575760405162461bcd60e51b815260206004820152600c60248201527f414c52454144595f494e4954000000000000000000000000000000000000000060448201526064016103f5565b50600080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b6000806105b86003338686604051610467929190610a79565b9050807fff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b85856040516105ec929190610a89565b60405180910390a29392505050565b600080546040517f8db5993b00000000000000000000000000000000000000000000000000000000815260ff8616600482015273ffffffffffffffffffffffffffffffffffffffff85811660248301526044820185905290911690638db5993b90349060640160206040518083038185885af115801561067f573d6000803e3d6000fd5b50505050506040513d601f19601f820116820180604052508101906106a49190610ad6565b949350505050565b73ffffffffffffffffffffffffffffffffffffffff811681146102f857600080fd5b60008083601f8401126106e057600080fd5b50813567ffffffffffffffff8111156106f857600080fd5b60208301915083602082850101111561071057600080fd5b9250929050565b600080600080600080600060c0888a03121561073257600080fd5b8735965060208801359550604088013594506060880135610752816106ac565b93506080880135925060a088013567ffffffffffffffff81111561077557600080fd5b6107818a828b016106ce565b989b979a50959850939692959293505050565b600080602083850312156107a757600080fd5b823567ffffffffffffffff8111156107be57600080fd5b6107ca858286016106ce565b90969095509350505050565b600080604083850312156107e957600080fd5b82356107f4816106ac565b91506020830135610804816106ac565b809150509250929050565b60008060008060006080868803121561082757600080fd5b85359450602086013593506040860135610840816106ac565b9250606086013567ffffffffffffffff81111561085c57600080fd5b610868888289016106ce565b969995985093965092949392505050565b60008060008060008060008060006101008a8c03121561089857600080fd5b89356108a3816106ac565b985060208a0135975060408a0135965060608a01356108c1816106ac565b955060808a01356108d1816106ac565b945060a08a0135935060c08a0135925060e08a013567ffffffffffffffff8111156108fb57600080fd5b6109078c828d016106ce565b915080935050809150509295985092959850929598565b60008060008060008060a0878903121561093757600080fd5b8635955060208701359450604087013593506060870135610957816106ac565b9250608087013567ffffffffffffffff81111561097357600080fd5b61097f89828a016106ce565b979a9699509497509295939492505050565b600080600080600060a086880312156109a957600080fd5b8535945060208601359350604086013592506060860135915060808601356109d0816106ac565b809150509295509295909350565b60008060008060008060a087890312156109f757600080fd5b86359550602087013594506040870135610a10816106ac565b935060608701359250608087013567ffffffffffffffff81111561097357600080fd5b60008060408385031215610a4657600080fd5b50508035926020909101359150565b600060208284031215610a6757600080fd5b8135610a72816106ac565b9392505050565b8183823760009101908152919050565b60208152816020820152818360408301376000818301604090810191909152601f9092017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0160101919050565b600060208284031215610ae857600080fd5b505191905056fea2646970667358221220f8da381333a9b8c35e37ed8b1a839ce828de7fd76680407be8b5050ca86c5ca364736f6c63430008110033",
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

// SendUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 , uint256 , uint256 , address , uint256 , bytes ) returns(uint256)
func (_InboxStub *InboxStubTransactor) SendUnsignedTransactionToFork(opts *bind.TransactOpts, arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "sendUnsignedTransactionToFork", arg0, arg1, arg2, arg3, arg4, arg5)
}

// SendUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 , uint256 , uint256 , address , uint256 , bytes ) returns(uint256)
func (_InboxStub *InboxStubSession) SendUnsignedTransactionToFork(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendUnsignedTransactionToFork(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// SendUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 , uint256 , uint256 , address , uint256 , bytes ) returns(uint256)
func (_InboxStub *InboxStubTransactorSession) SendUnsignedTransactionToFork(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*types.Transaction, error) {
	return _InboxStub.Contract.SendUnsignedTransactionToFork(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// SendWithdrawEthToFork is a paid mutator transaction binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 , uint256 , uint256 , uint256 , address ) returns(uint256)
func (_InboxStub *InboxStubTransactor) SendWithdrawEthToFork(opts *bind.TransactOpts, arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 *big.Int, arg4 common.Address) (*types.Transaction, error) {
	return _InboxStub.contract.Transact(opts, "sendWithdrawEthToFork", arg0, arg1, arg2, arg3, arg4)
}

// SendWithdrawEthToFork is a paid mutator transaction binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 , uint256 , uint256 , uint256 , address ) returns(uint256)
func (_InboxStub *InboxStubSession) SendWithdrawEthToFork(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 *big.Int, arg4 common.Address) (*types.Transaction, error) {
	return _InboxStub.Contract.SendWithdrawEthToFork(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4)
}

// SendWithdrawEthToFork is a paid mutator transaction binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 , uint256 , uint256 , uint256 , address ) returns(uint256)
func (_InboxStub *InboxStubTransactorSession) SendWithdrawEthToFork(arg0 *big.Int, arg1 *big.Int, arg2 *big.Int, arg3 *big.Int, arg4 common.Address) (*types.Transaction, error) {
	return _InboxStub.Contract.SendWithdrawEthToFork(&_InboxStub.TransactOpts, arg0, arg1, arg2, arg3, arg4)
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
	Bin: "0x608060405234801561001057600080fd5b506116b8806100206000396000f3fe608060405234801561001057600080fd5b50600436106100885760003560e01c8063c22c47a41161005b578063c22c47a4146100ff578063ca11325314610112578063d230d23f14610125578063e6bcbc651461013857600080fd5b80635fb9c3d41461008d57806367905a7e146100a25780636bd58993146100cb578063bc2f0640146100de575b600080fd5b6100a061009b3660046112de565b61014b565b005b6100b56100b0366004611367565b610161565b6040516100c291906113b5565b60405180910390f35b6100a06100d93660046113f9565b610178565b6100f16100ec366004611453565b61018a565b6040519081526020016100c2565b6100b561010d366004611475565b61019f565b6100f16101203660046114ba565b6101ab565b6100f16101333660046114f7565b6101b6565b6100f16101463660046114f7565b6101c1565b6101598686868686866101cc565b505050505050565b606061016e8484846104f4565b90505b9392505050565b61018484848484610a7e565b50505050565b60006101968383610b0b565b90505b92915050565b60606101968383610c00565b600061019982610c36565b600061019982610dd6565b600061019982610e3f565b600085116102215760405162461bcd60e51b815260206004820152601460248201527f5072652d73697a652063616e6e6f74206265203000000000000000000000000060448201526064015b60405180910390fd5b8561022b83610c36565b146102785760405162461bcd60e51b815260206004820152601b60248201527f50726520657870616e73696f6e20726f6f74206d69736d6174636800000000006044820152606401610218565b8461028283610f85565b146102f55760405162461bcd60e51b815260206004820152602160248201527f5072652073697a6520646f6573206e6f74206d6174636820657870616e73696f60448201527f6e000000000000000000000000000000000000000000000000000000000000006064820152608401610218565b8285106103445760405162461bcd60e51b815260206004820181905260248201527f5072652073697a65206e6f74206c657373207468616e20706f73742073697a656044820152606401610218565b60008590506000806103598560008751610fe0565b90505b8583101561041c5760006103708488610b0b565b9050845183106103c25760405162461bcd60e51b815260206004820152601260248201527f496e646578206f7574206f662072616e676500000000000000000000000000006044820152606401610218565b6103e682828786815181106103d9576103d9611510565b60200260200101516104f4565b91506001811b6103f6818661153c565b9450878511156104085761040861154f565b8361041281611565565b945050505061035c565b8661042682610c36565b146104995760405162461bcd60e51b815260206004820152602260248201527f506f737420657870616e73696f6e20726f6f74206e6f7420657175616c20706f60448201527f73740000000000000000000000000000000000000000000000000000000000006064820152608401610218565b835182146104e95760405162461bcd60e51b815260206004820152601660248201527f496e636f6d706c6574652070726f6f66207573616765000000000000000000006044820152606401610218565b505050505050505050565b6060604083106105465760405162461bcd60e51b815260206004820152600e60248201527f4c6576656c20746f6f20686967680000000000000000000000000000000000006044820152606401610218565b60008290036105975760405162461bcd60e51b815260206004820152601b60248201527f43616e6e6f7420617070656e6420656d707479207375627472656500000000006044820152606401610218565b6040845111156105e95760405162461bcd60e51b815260206004820152601a60248201527f4d65726b6c6520657870616e73696f6e20746f6f206c617267650000000000006044820152606401610218565b83516000036106685760006105ff84600161153c565b67ffffffffffffffff8111156106175761061761121a565b604051908082528060200260200182016040528015610640578160200160208202803683370190505b5090508281858151811061065657610656611510565b60209081029190910101529050610171565b835183106106de5760405162461bcd60e51b815260206004820152603560248201527f4c6576656c2067726561746572207468616e2068696768657374206c6576656c60448201527f206f662063757272656e7420657870616e73696f6e00000000000000000000006064820152608401610218565b8160006106ea86610f85565b905060006106f9866002611663565b610703908361153c565b9050600061071083610e3f565b61071983610e3f565b1161076757875167ffffffffffffffff8111156107385761073861121a565b604051908082528060200260200182016040528015610761578160200160208202803683370190505b506107b7565b875161077490600161153c565b67ffffffffffffffff81111561078c5761078c61121a565b6040519080825280602002602001820160405280156107b5578160200160208202803683370190505b505b905060408151111561080b5760405162461bcd60e51b815260206004820152601c60248201527f417070656e642063726561746573206f76657273697a652074726565000000006044820152606401610218565b60005b88518110156109c757878110156108b55788818151811061083157610831611510565b60200260200101516000801b146108b05760405162461bcd60e51b815260206004820152602260248201527f417070656e642061626f7665206c65617374207369676e69666963616e74206260448201527f69740000000000000000000000000000000000000000000000000000000000006064820152608401610218565b6109b5565b60008590036108fb578881815181106108d0576108d0611510565b60200260200101518282815181106108ea576108ea611510565b6020026020010181815250506109b5565b88818151811061090d5761090d611510565b60200260200101516000801b03610945578482828151811061093157610931611510565b6020908102919091010152600094506109b5565b6000801b82828151811061095b5761095b611510565b60200260200101818152505088818151811061097957610979611510565b60200260200101518560405160200161099c929190918252602082015260400190565b6040516020818303038152906040528051906020012094505b806109bf81611565565b91505061080e565b5083156109fb578381600183516109de919061166f565b815181106109ee576109ee611510565b6020026020010181815250505b8060018251610a0a919061166f565b81518110610a1a57610a1a611510565b60200260200101516000801b03610a735760405162461bcd60e51b815260206004820152600f60248201527f4c61737420656e747279207a65726f00000000000000000000000000000000006044820152606401610218565b979650505050505050565b6000610ab3828486604051602001610a9891815260200190565b6040516020818303038152906040528051906020012061115f565b9050808514610b045760405162461bcd60e51b815260206004820152601760248201527f496e76616c696420696e636c7573696f6e2070726f6f660000000000000000006044820152606401610218565b5050505050565b6000818310610b5c5760405162461bcd60e51b815260206004820152601760248201527f5374617274206e6f74206c657373207468616e20656e640000000000000000006044820152606401610218565b6000610b69838518610e3f565b905060006001610b79838261153c565b6001901b610b87919061166f565b90508481168482168115610ba957610b9e82610dd6565b945050505050610199565b8015610bb857610b9e81610e3f565b60405162461bcd60e51b815260206004820152601b60248201527f426f7468207920616e64207a2063616e6e6f74206265207a65726f00000000006044820152606401610218565b606061019683600084604051602001610c1b91815260200190565b604051602081830303815290604052805190602001206104f4565b600080825111610c885760405162461bcd60e51b815260206004820152601660248201527f456d707479206d65726b6c6520657870616e73696f6e000000000000000000006044820152606401610218565b604082511115610cda5760405162461bcd60e51b815260206004820152601a60248201527f4d65726b6c6520657870616e73696f6e20746f6f206c617267650000000000006044820152606401610218565b6000805b8351811015610dcf576000848281518110610cfb57610cfb611510565b60200260200101519050826000801b03610d67578015610d625780925060018551610d26919061166f565b8214610d6257604051610d49908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b610dbc565b8015610d86576040805160208101839052908101849052606001610d49565b604051610da3908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b5080610dc781611565565b915050610cde565b5092915050565b6000808211610e275760405162461bcd60e51b815260206004820152601c60248201527f5a65726f20686173206e6f207369676e69666963616e742062697473000000006044820152606401610218565b60008280610e3660018261166f565b16189050610171815b600081600003610e915760405162461bcd60e51b815260206004820152601c60248201527f5a65726f20686173206e6f207369676e69666963616e742062697473000000006044820152606401610218565b7001000000000000000000000000000000008210610ebc57608091821c91610eb9908261153c565b90505b680100000000000000008210610edf57604091821c91610edc908261153c565b90505b6401000000008210610efe57602091821c91610efb908261153c565b90505b620100008210610f1b57601091821c91610f18908261153c565b90505b6101008210610f3757600891821c91610f34908261153c565b90505b60108210610f5257600491821c91610f4f908261153c565b90505b60048210610f6d57600291821c91610f6a908261153c565b90505b60028210610f805761019960018261153c565b919050565b600080805b8351811015610dcf57838181518110610fa557610fa5611510565b60200260200101516000801b14610fce57610fc1816002611663565b610fcb908361153c565b91505b80610fd881611565565b915050610f8a565b60608183106110315760405162461bcd60e51b815260206004820152601760248201527f5374617274206e6f74206c657373207468616e20656e640000000000000000006044820152606401610218565b83518211156110a85760405162461bcd60e51b815260206004820152602160248201527f456e64206e6f74206c657373206f7220657175616c207468616e206c656e677460448201527f68000000000000000000000000000000000000000000000000000000000000006064820152608401610218565b60006110b4848461166f565b67ffffffffffffffff8111156110cc576110cc61121a565b6040519080825280602002602001820160405280156110f5578160200160208202803683370190505b509050835b838110156111565785818151811061111457611114611510565b6020026020010151828683611129919061166f565b8151811061113957611139611510565b60209081029190910101528061114e81611565565b9150506110fa565b50949350505050565b82516000906101008111156111ab576040517ffdac331e000000000000000000000000000000000000000000000000000000008152600481018290526101006024820152604401610218565b8260005b828110156112105760008782815181106111cb576111cb611510565b60200260200101519050816001901b87166000036111f757826000528060205260406000209250611207565b8060005282602052604060002092505b506001016111af565b5095945050505050565b634e487b7160e01b600052604160045260246000fd5b600082601f83011261124157600080fd5b8135602067ffffffffffffffff8083111561125e5761125e61121a565b8260051b6040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0603f830116810181811084821117156112a1576112a161121a565b6040529384528581018301938381019250878511156112bf57600080fd5b83870191505b84821015610a73578135835291830191908301906112c5565b60008060008060008060c087890312156112f757600080fd5b86359550602087013594506040870135935060608701359250608087013567ffffffffffffffff8082111561132b57600080fd5b6113378a838b01611230565b935060a089013591508082111561134d57600080fd5b5061135a89828a01611230565b9150509295509295509295565b60008060006060848603121561137c57600080fd5b833567ffffffffffffffff81111561139357600080fd5b61139f86828701611230565b9660208601359650604090950135949350505050565b6020808252825182820181905260009190848201906040850190845b818110156113ed578351835292840192918401916001016113d1565b50909695505050505050565b6000806000806080858703121561140f57600080fd5b843593506020850135925060408501359150606085013567ffffffffffffffff81111561143b57600080fd5b61144787828801611230565b91505092959194509250565b6000806040838503121561146657600080fd5b50508035926020909101359150565b6000806040838503121561148857600080fd5b823567ffffffffffffffff81111561149f57600080fd5b6114ab85828601611230565b95602094909401359450505050565b6000602082840312156114cc57600080fd5b813567ffffffffffffffff8111156114e357600080fd5b6114ef84828501611230565b949350505050565b60006020828403121561150957600080fd5b5035919050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b8082018082111561019957610199611526565b634e487b7160e01b600052600160045260246000fd5b6000600019820361157857611578611526565b5060010190565b600181815b808511156115ba5781600019048211156115a0576115a0611526565b808516156115ad57918102915b93841c9390800290611584565b509250929050565b6000826115d157506001610199565b816115de57506000610199565b81600181146115f457600281146115fe5761161a565b6001915050610199565b60ff84111561160f5761160f611526565b50506001821b610199565b5060208310610133831016604e8410600b841016171561163d575081810a610199565b611647838361157f565b806000190482111561165b5761165b611526565b029392505050565b600061019683836115c2565b818103818111156101995761019961152656fea26469706673582212209ea92f5dd270735a0679d4b9f24152d1e5de0c1e893a07d785e2ee5ab377631064736f6c63430008110033",
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

// MockResultReceiverMetaData contains all meta data concerning the MockResultReceiver contract.
var MockResultReceiverMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIOldChallengeManager\",\"name\":\"manager_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"challengeIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"winner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"loser\",\"type\":\"address\"}],\"name\":\"ChallengeCompleted\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"challengeIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"challengeIndex_\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"winner_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"loser_\",\"type\":\"address\"}],\"name\":\"completeChallenge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot_\",\"type\":\"bytes32\"},{\"internalType\":\"enumMachineStatus[2]\",\"name\":\"startAndEndMachineStatuses_\",\"type\":\"uint8[2]\"},{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState[2]\",\"name\":\"startAndEndGlobalStates_\",\"type\":\"tuple[2]\"},{\"internalType\":\"uint64\",\"name\":\"numBlocks\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"asserter_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"challenger_\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asserterTimeLeft_\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"challengerTimeLeft_\",\"type\":\"uint256\"}],\"name\":\"createChallenge\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"loser\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"manager\",\"outputs\":[{\"internalType\":\"contractIOldChallengeManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"winner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506040516105a33803806105a383398101604081905261002f91610054565b600080546001600160a01b0319166001600160a01b0392909216919091179055610084565b60006020828403121561006657600080fd5b81516001600160a01b038116811461007d57600080fd5b9392505050565b610510806100936000396000f3fe608060405234801561001057600080fd5b50600436106100725760003560e01c8063d685374811610050578063d685374814610102578063dfbf53ae14610119578063e82898b31461013957600080fd5b80630357aa491461007757806314eab5e71461008c578063481c6a75146100bd575b600080fd5b61008a6100853660046102c2565b610159565b005b61009f61009a366004610317565b6101e4565b60405167ffffffffffffffff90911681526020015b60405180910390f35b6000546100dd9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100b4565b61010b60035481565b6040519081526020016100b4565b6001546100dd9073ffffffffffffffffffffffffffffffffffffffff1681565b6002546100dd9073ffffffffffffffffffffffffffffffffffffffff1681565b600180547fffffffffffffffffffffffff000000000000000000000000000000000000000090811673ffffffffffffffffffffffffffffffffffffffff85811691821790935560028054909216928416928317909155600385905560405185907f88cb1f3fe351f3ac338db9c36bff1ece1750423c7ae6dfc427cd194b1c69b12790600090a4505050565b600080546040517f14eab5e700000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff909116906314eab5e790610249908c908c908c908c908c908c908c908c9060040161041c565b6020604051808303816000875af1158015610268573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061028c91906104b6565b9998505050505050505050565b803573ffffffffffffffffffffffffffffffffffffffff811681146102bd57600080fd5b919050565b6000806000606084860312156102d757600080fd5b833592506102e760208501610299565b91506102f560408501610299565b90509250925092565b67ffffffffffffffff8116811461031457600080fd5b50565b600080600080600080600080610200898b03121561033457600080fd5b88359750606089018a81111561034957600080fd5b60208a0197506101608a018b81111561036157600080fd5b9096503561036e816102fe565b945061037d6101808a01610299565b935061038c6101a08a01610299565b92506101c089013591506101e089013590509295985092959890939650565b806000805b60028082106103bf5750610415565b604080858837848101908701845b838110156103fe5782356103e0816102fe565b67ffffffffffffffff168252602092830192909101906001016103cd565b5050506080958601959390930192506001016103b0565b5050505050565b888152610200810160208083018a6000805b600281101561045757823560038110610445578283fd5b8452928401929184019160010161042e565b505050505061046960608301896103ab565b67ffffffffffffffff9690961661016082015273ffffffffffffffffffffffffffffffffffffffff948516610180820152929093166101a08301526101c08201526101e001529392505050565b6000602082840312156104c857600080fd5b81516104d3816102fe565b939250505056fea2646970667358221220a51244b629dfee810f882e09cae3cb26e7a43363389bd7c15f43ddfaf106a11064736f6c63430008110033",
}

// MockResultReceiverABI is the input ABI used to generate the binding from.
// Deprecated: Use MockResultReceiverMetaData.ABI instead.
var MockResultReceiverABI = MockResultReceiverMetaData.ABI

// MockResultReceiverBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MockResultReceiverMetaData.Bin instead.
var MockResultReceiverBin = MockResultReceiverMetaData.Bin

// DeployMockResultReceiver deploys a new Ethereum contract, binding an instance of MockResultReceiver to it.
func DeployMockResultReceiver(auth *bind.TransactOpts, backend bind.ContractBackend, manager_ common.Address) (common.Address, *types.Transaction, *MockResultReceiver, error) {
	parsed, err := MockResultReceiverMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MockResultReceiverBin), backend, manager_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MockResultReceiver{MockResultReceiverCaller: MockResultReceiverCaller{contract: contract}, MockResultReceiverTransactor: MockResultReceiverTransactor{contract: contract}, MockResultReceiverFilterer: MockResultReceiverFilterer{contract: contract}}, nil
}

// MockResultReceiver is an auto generated Go binding around an Ethereum contract.
type MockResultReceiver struct {
	MockResultReceiverCaller     // Read-only binding to the contract
	MockResultReceiverTransactor // Write-only binding to the contract
	MockResultReceiverFilterer   // Log filterer for contract events
}

// MockResultReceiverCaller is an auto generated read-only Go binding around an Ethereum contract.
type MockResultReceiverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockResultReceiverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MockResultReceiverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockResultReceiverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MockResultReceiverFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockResultReceiverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MockResultReceiverSession struct {
	Contract     *MockResultReceiver // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// MockResultReceiverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MockResultReceiverCallerSession struct {
	Contract *MockResultReceiverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// MockResultReceiverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MockResultReceiverTransactorSession struct {
	Contract     *MockResultReceiverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// MockResultReceiverRaw is an auto generated low-level Go binding around an Ethereum contract.
type MockResultReceiverRaw struct {
	Contract *MockResultReceiver // Generic contract binding to access the raw methods on
}

// MockResultReceiverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MockResultReceiverCallerRaw struct {
	Contract *MockResultReceiverCaller // Generic read-only contract binding to access the raw methods on
}

// MockResultReceiverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MockResultReceiverTransactorRaw struct {
	Contract *MockResultReceiverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMockResultReceiver creates a new instance of MockResultReceiver, bound to a specific deployed contract.
func NewMockResultReceiver(address common.Address, backend bind.ContractBackend) (*MockResultReceiver, error) {
	contract, err := bindMockResultReceiver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MockResultReceiver{MockResultReceiverCaller: MockResultReceiverCaller{contract: contract}, MockResultReceiverTransactor: MockResultReceiverTransactor{contract: contract}, MockResultReceiverFilterer: MockResultReceiverFilterer{contract: contract}}, nil
}

// NewMockResultReceiverCaller creates a new read-only instance of MockResultReceiver, bound to a specific deployed contract.
func NewMockResultReceiverCaller(address common.Address, caller bind.ContractCaller) (*MockResultReceiverCaller, error) {
	contract, err := bindMockResultReceiver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MockResultReceiverCaller{contract: contract}, nil
}

// NewMockResultReceiverTransactor creates a new write-only instance of MockResultReceiver, bound to a specific deployed contract.
func NewMockResultReceiverTransactor(address common.Address, transactor bind.ContractTransactor) (*MockResultReceiverTransactor, error) {
	contract, err := bindMockResultReceiver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MockResultReceiverTransactor{contract: contract}, nil
}

// NewMockResultReceiverFilterer creates a new log filterer instance of MockResultReceiver, bound to a specific deployed contract.
func NewMockResultReceiverFilterer(address common.Address, filterer bind.ContractFilterer) (*MockResultReceiverFilterer, error) {
	contract, err := bindMockResultReceiver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MockResultReceiverFilterer{contract: contract}, nil
}

// bindMockResultReceiver binds a generic wrapper to an already deployed contract.
func bindMockResultReceiver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MockResultReceiverMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockResultReceiver *MockResultReceiverRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockResultReceiver.Contract.MockResultReceiverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockResultReceiver *MockResultReceiverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockResultReceiver.Contract.MockResultReceiverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockResultReceiver *MockResultReceiverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockResultReceiver.Contract.MockResultReceiverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockResultReceiver *MockResultReceiverCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockResultReceiver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockResultReceiver *MockResultReceiverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockResultReceiver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockResultReceiver *MockResultReceiverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockResultReceiver.Contract.contract.Transact(opts, method, params...)
}

// ChallengeIndex is a free data retrieval call binding the contract method 0xd6853748.
//
// Solidity: function challengeIndex() view returns(uint256)
func (_MockResultReceiver *MockResultReceiverCaller) ChallengeIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MockResultReceiver.contract.Call(opts, &out, "challengeIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChallengeIndex is a free data retrieval call binding the contract method 0xd6853748.
//
// Solidity: function challengeIndex() view returns(uint256)
func (_MockResultReceiver *MockResultReceiverSession) ChallengeIndex() (*big.Int, error) {
	return _MockResultReceiver.Contract.ChallengeIndex(&_MockResultReceiver.CallOpts)
}

// ChallengeIndex is a free data retrieval call binding the contract method 0xd6853748.
//
// Solidity: function challengeIndex() view returns(uint256)
func (_MockResultReceiver *MockResultReceiverCallerSession) ChallengeIndex() (*big.Int, error) {
	return _MockResultReceiver.Contract.ChallengeIndex(&_MockResultReceiver.CallOpts)
}

// Loser is a free data retrieval call binding the contract method 0xe82898b3.
//
// Solidity: function loser() view returns(address)
func (_MockResultReceiver *MockResultReceiverCaller) Loser(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MockResultReceiver.contract.Call(opts, &out, "loser")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Loser is a free data retrieval call binding the contract method 0xe82898b3.
//
// Solidity: function loser() view returns(address)
func (_MockResultReceiver *MockResultReceiverSession) Loser() (common.Address, error) {
	return _MockResultReceiver.Contract.Loser(&_MockResultReceiver.CallOpts)
}

// Loser is a free data retrieval call binding the contract method 0xe82898b3.
//
// Solidity: function loser() view returns(address)
func (_MockResultReceiver *MockResultReceiverCallerSession) Loser() (common.Address, error) {
	return _MockResultReceiver.Contract.Loser(&_MockResultReceiver.CallOpts)
}

// Manager is a free data retrieval call binding the contract method 0x481c6a75.
//
// Solidity: function manager() view returns(address)
func (_MockResultReceiver *MockResultReceiverCaller) Manager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MockResultReceiver.contract.Call(opts, &out, "manager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Manager is a free data retrieval call binding the contract method 0x481c6a75.
//
// Solidity: function manager() view returns(address)
func (_MockResultReceiver *MockResultReceiverSession) Manager() (common.Address, error) {
	return _MockResultReceiver.Contract.Manager(&_MockResultReceiver.CallOpts)
}

// Manager is a free data retrieval call binding the contract method 0x481c6a75.
//
// Solidity: function manager() view returns(address)
func (_MockResultReceiver *MockResultReceiverCallerSession) Manager() (common.Address, error) {
	return _MockResultReceiver.Contract.Manager(&_MockResultReceiver.CallOpts)
}

// Winner is a free data retrieval call binding the contract method 0xdfbf53ae.
//
// Solidity: function winner() view returns(address)
func (_MockResultReceiver *MockResultReceiverCaller) Winner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MockResultReceiver.contract.Call(opts, &out, "winner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Winner is a free data retrieval call binding the contract method 0xdfbf53ae.
//
// Solidity: function winner() view returns(address)
func (_MockResultReceiver *MockResultReceiverSession) Winner() (common.Address, error) {
	return _MockResultReceiver.Contract.Winner(&_MockResultReceiver.CallOpts)
}

// Winner is a free data retrieval call binding the contract method 0xdfbf53ae.
//
// Solidity: function winner() view returns(address)
func (_MockResultReceiver *MockResultReceiverCallerSession) Winner() (common.Address, error) {
	return _MockResultReceiver.Contract.Winner(&_MockResultReceiver.CallOpts)
}

// CompleteChallenge is a paid mutator transaction binding the contract method 0x0357aa49.
//
// Solidity: function completeChallenge(uint256 challengeIndex_, address winner_, address loser_) returns()
func (_MockResultReceiver *MockResultReceiverTransactor) CompleteChallenge(opts *bind.TransactOpts, challengeIndex_ *big.Int, winner_ common.Address, loser_ common.Address) (*types.Transaction, error) {
	return _MockResultReceiver.contract.Transact(opts, "completeChallenge", challengeIndex_, winner_, loser_)
}

// CompleteChallenge is a paid mutator transaction binding the contract method 0x0357aa49.
//
// Solidity: function completeChallenge(uint256 challengeIndex_, address winner_, address loser_) returns()
func (_MockResultReceiver *MockResultReceiverSession) CompleteChallenge(challengeIndex_ *big.Int, winner_ common.Address, loser_ common.Address) (*types.Transaction, error) {
	return _MockResultReceiver.Contract.CompleteChallenge(&_MockResultReceiver.TransactOpts, challengeIndex_, winner_, loser_)
}

// CompleteChallenge is a paid mutator transaction binding the contract method 0x0357aa49.
//
// Solidity: function completeChallenge(uint256 challengeIndex_, address winner_, address loser_) returns()
func (_MockResultReceiver *MockResultReceiverTransactorSession) CompleteChallenge(challengeIndex_ *big.Int, winner_ common.Address, loser_ common.Address) (*types.Transaction, error) {
	return _MockResultReceiver.Contract.CompleteChallenge(&_MockResultReceiver.TransactOpts, challengeIndex_, winner_, loser_)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_MockResultReceiver *MockResultReceiverTransactor) CreateChallenge(opts *bind.TransactOpts, wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _MockResultReceiver.contract.Transact(opts, "createChallenge", wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_MockResultReceiver *MockResultReceiverSession) CreateChallenge(wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _MockResultReceiver.Contract.CreateChallenge(&_MockResultReceiver.TransactOpts, wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_MockResultReceiver *MockResultReceiverTransactorSession) CreateChallenge(wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _MockResultReceiver.Contract.CreateChallenge(&_MockResultReceiver.TransactOpts, wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// MockResultReceiverChallengeCompletedIterator is returned from FilterChallengeCompleted and is used to iterate over the raw logs and unpacked data for ChallengeCompleted events raised by the MockResultReceiver contract.
type MockResultReceiverChallengeCompletedIterator struct {
	Event *MockResultReceiverChallengeCompleted // Event containing the contract specifics and raw log

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
func (it *MockResultReceiverChallengeCompletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MockResultReceiverChallengeCompleted)
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
		it.Event = new(MockResultReceiverChallengeCompleted)
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
func (it *MockResultReceiverChallengeCompletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MockResultReceiverChallengeCompletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MockResultReceiverChallengeCompleted represents a ChallengeCompleted event raised by the MockResultReceiver contract.
type MockResultReceiverChallengeCompleted struct {
	ChallengeIndex *big.Int
	Winner         common.Address
	Loser          common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterChallengeCompleted is a free log retrieval operation binding the contract event 0x88cb1f3fe351f3ac338db9c36bff1ece1750423c7ae6dfc427cd194b1c69b127.
//
// Solidity: event ChallengeCompleted(uint256 indexed challengeIndex, address indexed winner, address indexed loser)
func (_MockResultReceiver *MockResultReceiverFilterer) FilterChallengeCompleted(opts *bind.FilterOpts, challengeIndex []*big.Int, winner []common.Address, loser []common.Address) (*MockResultReceiverChallengeCompletedIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}
	var winnerRule []interface{}
	for _, winnerItem := range winner {
		winnerRule = append(winnerRule, winnerItem)
	}
	var loserRule []interface{}
	for _, loserItem := range loser {
		loserRule = append(loserRule, loserItem)
	}

	logs, sub, err := _MockResultReceiver.contract.FilterLogs(opts, "ChallengeCompleted", challengeIndexRule, winnerRule, loserRule)
	if err != nil {
		return nil, err
	}
	return &MockResultReceiverChallengeCompletedIterator{contract: _MockResultReceiver.contract, event: "ChallengeCompleted", logs: logs, sub: sub}, nil
}

// WatchChallengeCompleted is a free log subscription operation binding the contract event 0x88cb1f3fe351f3ac338db9c36bff1ece1750423c7ae6dfc427cd194b1c69b127.
//
// Solidity: event ChallengeCompleted(uint256 indexed challengeIndex, address indexed winner, address indexed loser)
func (_MockResultReceiver *MockResultReceiverFilterer) WatchChallengeCompleted(opts *bind.WatchOpts, sink chan<- *MockResultReceiverChallengeCompleted, challengeIndex []*big.Int, winner []common.Address, loser []common.Address) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}
	var winnerRule []interface{}
	for _, winnerItem := range winner {
		winnerRule = append(winnerRule, winnerItem)
	}
	var loserRule []interface{}
	for _, loserItem := range loser {
		loserRule = append(loserRule, loserItem)
	}

	logs, sub, err := _MockResultReceiver.contract.WatchLogs(opts, "ChallengeCompleted", challengeIndexRule, winnerRule, loserRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MockResultReceiverChallengeCompleted)
				if err := _MockResultReceiver.contract.UnpackLog(event, "ChallengeCompleted", log); err != nil {
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

// ParseChallengeCompleted is a log parse operation binding the contract event 0x88cb1f3fe351f3ac338db9c36bff1ece1750423c7ae6dfc427cd194b1c69b127.
//
// Solidity: event ChallengeCompleted(uint256 indexed challengeIndex, address indexed winner, address indexed loser)
func (_MockResultReceiver *MockResultReceiverFilterer) ParseChallengeCompleted(log types.Log) (*MockResultReceiverChallengeCompleted, error) {
	event := new(MockResultReceiverChallengeCompleted)
	if err := _MockResultReceiver.contract.UnpackLog(event, "ChallengeCompleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxStubMetaData contains all meta data concerning the SequencerInboxStub contract.
var SequencerInboxStubMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"bridge_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"sequencer_\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"maxDataSize_\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AlreadyInit\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"AlreadyValidDASKeyset\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"stored\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"received\",\"type\":\"uint256\"}],\"name\":\"BadSequencerNumber\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DataNotAuthenticated\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"dataLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxDataLength\",\"type\":\"uint256\"}],\"name\":\"DataTooLarge\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelayedBackwards\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelayedTooFar\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ForceIncludeBlockTooSoon\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ForceIncludeTimeTooSoon\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HadZeroInit\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"IncorrectMessagePreimage\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"NoSuchKeyset\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotBatchPoster\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotForked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotOrigin\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotOwner\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keysetHash\",\"type\":\"bytes32\"}],\"name\":\"InvalidateKeyset\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"OwnerFunctionCalled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchSequenceNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"SequencerBatchData\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchSequenceNumber\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"afterAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"minTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"minBlockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxBlockNumber\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structISequencerInbox.TimeBounds\",\"name\":\"timeBounds\",\"type\":\"tuple\"},{\"indexed\":false,\"internalType\":\"enumISequencerInbox.BatchDataLocation\",\"name\":\"dataLocation\",\"type\":\"uint8\"}],\"name\":\"SequencerBatchDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keysetHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"SetValidKeyset\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DATA_AUTHENTICATED_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"HEADER_LENGTH\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"name\":\"addInitMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2Batch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"}],\"name\":\"addSequencerL2BatchFromOrigin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2BatchFromOrigin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batchCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"dasKeySetInfo\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"isValidKeyset\",\"type\":\"bool\"},{\"internalType\":\"uint64\",\"name\":\"creationBlock\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"uint64[2]\",\"name\":\"l1BlockAndTime\",\"type\":\"uint64[2]\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"forceInclusion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"getKeysetCreationBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"inboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"bridge_\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"invalidateKeysetHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isBatchPoster\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isSequencer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"isValidKeysetHash\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxDataSize\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxTimeVariation\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"removeDelayAfterFork\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isBatchPoster_\",\"type\":\"bool\"}],\"name\":\"setIsBatchPoster\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isSequencer_\",\"type\":\"bool\"}],\"name\":\"setIsSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"}],\"name\":\"setMaxTimeVariation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"setValidKeyset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalDelayedMessagesRead\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x610100604052306080524660c05262000023620000c5602090811b62001ed217901c565b151560e0523480156200003557600080fd5b506040516200313a3803806200313a83398101604081905262000058916200017b565b60a052600180546001600160a01b039485166001600160a01b03199182161782556002805490911633179055815160045560208083015160055560408084015160065560609093015160075592909316600090815260039092529020805460ff1916909117905562000265565b60408051600481526024810182526020810180516001600160e01b03166302881c7960e11b17905290516000918291829160649162000105919062000234565b600060405180830381855afa9150503d806000811462000142576040519150601f19603f3d011682016040523d82523d6000602084013e62000147565b606091505b50915091508180156200015b575080516020145b9250505090565b6001600160a01b03811681146200017857600080fd5b50565b60008060008084860360e08112156200019357600080fd5b8551620001a08162000162565b6020870151909550620001b38162000162565b93506080603f1982011215620001c857600080fd5b50604051608081016001600160401b0381118282101715620001fa57634e487b7160e01b600052604160045260246000fd5b6040908152868101518252606080880151602084015260808801519183019190915260a08701519082015260c09095015193969295505050565b6000825160005b818110156200025757602081860181015185830152016200023b565b506000920191825250919050565b60805160a05160c05160e051612e7f620002bb600039600081816116fb0152612186015260006113840152600081816104480152818161246a01526124bf0152600081816105470152610af30152612e7f6000f3fe608060405234801561001057600080fd5b50600436106101b95760003560e01c80637fa3a40e116100f9578063d9dd67ab11610097578063e78cea9211610071578063e78cea9214610430578063e8eb1dc314610443578063ebea461d1461046a578063f1981578146104a057600080fd5b8063d9dd67ab146103b2578063e0bc9729146103c5578063e5a358c8146103d857600080fd5b806396cc5c78116100d357806396cc5c7814610359578063b31761f814610361578063cb23bcb514610374578063d1ce8da81461039f57600080fd5b80637fa3a40e1461032a57806384420860146103335780638f111f3c1461034657600080fd5b80636633ae85116101665780636e7df3e7116101405780636e7df3e71461028d5780636f12b0c9146102a0578063715ea34b146102b357806371c3e6fe1461030757600080fd5b80636633ae851461024f5780636ae71f12146102625780636d46e9871461026a57600080fd5b80631f956632116101975780631f95663214610221578063258f04951461023457806327957a491461024757600080fd5b806306f13056146101be5780631637be48146101d95780631f7a92b21461020c575b600080fd5b6101c66104b3565b6040519081526020015b60405180910390f35b6101fc6101e7366004612737565b60009081526008602052604090205460ff1690565b60405190151581526020016101d0565b61021f61021a366004612768565b61053d565b005b61021f61022f3660046127b7565b610752565b6101c6610242366004612737565b61085f565b6101c6602881565b61021f61025d366004612737565b6108cc565b61021f610ae9565b6101fc6102783660046127f0565b60096020526000908152604090205460ff1681565b61021f61029b3660046127b7565b610c8c565b61021f6102ae36600461285d565b610d99565b6102e76102c1366004612737565b60086020526000908152604090205460ff811690610100900467ffffffffffffffff1682565b60408051921515835267ffffffffffffffff9091166020830152016101d0565b6101fc6103153660046127f0565b60036020526000908152604090205460ff1681565b6101c660005481565b61021f610341366004612737565b610fd0565b61021f6103543660046128c8565b61112a565b61021f611381565b61021f61036f366004612945565b611410565b600254610387906001600160a01b031681565b6040516001600160a01b0390911681526020016101d0565b61021f6103ad3660046129b9565b611517565b6101c66103c0366004612737565b61187b565b61021f6103d33660046128c8565b611908565b6103ff7f400000000000000000000000000000000000000000000000000000000000000081565b6040517fff0000000000000000000000000000000000000000000000000000000000000090911681526020016101d0565b600154610387906001600160a01b031681565b6101c67f000000000000000000000000000000000000000000000000000000000000000081565b6004546005546006546007546104809392919084565b6040805194855260208501939093529183015260608201526080016101d0565b61021f6104ae3660046129fb565b611a6f565b600154604080517e84120c00000000000000000000000000000000000000000000000000000000815290516000926001600160a01b0316916284120c9160048083019260209291908290030181865afa158015610514573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105389190612a6b565b905090565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036105e05760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084015b60405180910390fd5b6001546001600160a01b031615610623576040517fef34ca5c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001600160a01b038216610663576040517f1ad0f74300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038416908117909155604080517fcb23bcb5000000000000000000000000000000000000000000000000000000008152905163cb23bcb5916004808201926020929091908290030181865afa1580156106e2573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107069190612a84565b6002805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03929092169190911790558035600455602081013560055560408101356006556060013560075550565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156107a5573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107c99190612a84565b6001600160a01b0316336001600160a01b03161461080f57600254604051631194af8760e11b81523360048201526001600160a01b0390911660248201526044016105d7565b6001600160a01b038216600090815260096020526040808220805460ff1916841515179055516004917fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e91a25050565b600081815260086020908152604080832081518083019092525460ff811615158252610100900467ffffffffffffffff169181018290529082036108b85760405162f20c5d60e01b8152600481018490526024016105d7565b6020015167ffffffffffffffff1692915050565b6000816040516020016108e191815260200190565b60408051808303601f1901815290829052600154815160208301207f8db5993b000000000000000000000000000000000000000000000000000000008452600b6004850152600060248501819052604485019190915291935090916001600160a01b0390911690638db5993b906064016020604051808303816000875af1158015610970573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906109949190612a6b565b905080156109e45760405162461bcd60e51b815260206004820152601460248201527f414c52454144595f44454c415945445f494e495400000000000000000000000060448201526064016105d7565b807fff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b83604051610a149190612ac5565b60405180910390a2600080610a296001611f98565b91509150600080600080610a438660016000806001611fdd565b935093509350935083600014610a9b5760405162461bcd60e51b815260206004820152601060248201527f414c52454144595f5345515f494e49540000000000000000000000000000000060448201526064016105d7565b8083857f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548a6002604051610ad69493929190612af8565b60405180910390a4505050505050505050565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610b875760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084016105d7565b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038054336001600160a01b03821614610be457604051631194af8760e11b81523360048201526001600160a01b03821660248201526044016105d7565b600160009054906101000a90046001600160a01b03166001600160a01b031663cb23bcb56040518163ffffffff1660e01b8152600401602060405180830381865afa158015610c37573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610c5b9190612a84565b6002805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03929092169190911790555050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610cdf573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610d039190612a84565b6001600160a01b0316336001600160a01b031614610d4957600254604051631194af8760e11b81523360048201526001600160a01b0390911660248201526044016105d7565b6001600160a01b038216600090815260036020526040808220805460ff1916841515179055516001917fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e91a25050565b8060005a9050333214610dd8576040517ffeb3d07100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526003602052604090205460ff16610e0857604051632dd9fc9760e01b815260040160405180910390fd5b600080610e16888888612434565b90925090506000808080610e2d868b8d8480611fdd565b93509350935093508c8414610e5f5760405163ac7411c960e01b815260048101859052602481018e90526044016105d7565b80838e7f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548a6000604051610e9a9493929190612af8565b60405180910390a4505050506001600160a01b038416159150610fc79050573660006020610ec983601f612b83565b610ed39190612b96565b9050610200610ee3600283612c9c565b610eed9190612b96565b610ef8826006612cab565b610f029190612b83565b610f0c9084612b83565b9250333214610f1a57600091505b836001600160a01b031663e3db8a49335a610f359087612cc2565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af1158015610f9f573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610fc39190612cd5565b5050505b50505050505050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611023573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906110479190612a84565b6001600160a01b0316336001600160a01b03161461108d57600254604051631194af8760e11b81523360048201526001600160a01b0390911660248201526044016105d7565b60008181526008602052604090205460ff166110be5760405162f20c5d60e01b8152600481018290526024016105d7565b600081815260086020526040808220805460ff191690555182917f5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a91a26040516003907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a250565b8260005a9050333214611169576040517ffeb3d07100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526003602052604090205460ff1661119957604051632dd9fc9760e01b815260040160405180910390fd5b6000806111a78a8a8a612434565b90925090508a81838b8b8a8a60008080806111c589888a8989611fdd565b93509350935093508a84141580156111df57506000198b14155b156112075760405163ac7411c960e01b815260048101859052602481018c90526044016105d7565b8083857f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548f60006040516112429493929190612af8565b60405180910390a4505050506001600160a01b038b1615985061137697505050505050505057366000602061127883601f612b83565b6112829190612b96565b9050610200611292600283612c9c565b61129c9190612b96565b6112a7826006612cab565b6112b19190612b83565b6112bb9084612b83565b92503332146112c957600091505b836001600160a01b031663e3db8a49335a6112e49087612cc2565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af115801561134e573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906113729190612cd5565b5050505b505050505050505050565b467f0000000000000000000000000000000000000000000000000000000000000000036113da576040517fa301bb0600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60408051608081018252600180825260208201819052918101829052606001819052600481905560058190556006819055600755565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611463573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114879190612a84565b6001600160a01b0316336001600160a01b0316146114cd57600254604051631194af8760e11b81523360048201526001600160a01b0390911660248201526044016105d7565b805160045560208101516005556040808201516006556060820151600755516000907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e908290a250565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561156a573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061158e9190612a84565b6001600160a01b0316336001600160a01b0316146115d457600254604051631194af8760e11b81523360048201526001600160a01b0390911660248201526044016105d7565b600082826040516115e6929190612cf2565b6040519081900381207ffe000000000000000000000000000000000000000000000000000000000000006020830152602182015260410160408051601f19818403018152919052805160209091012090507f800000000000000000000000000000000000000000000000000000000000000081186201000083106116ac5760405162461bcd60e51b815260206004820152601360248201527f6b657973657420697320746f6f206c617267650000000000000000000000000060448201526064016105d7565b60008181526008602052604090205460ff16156116f8576040517ffa2fddda000000000000000000000000000000000000000000000000000000008152600481018290526024016105d7565b437f0000000000000000000000000000000000000000000000000000000000000000156117855760646001600160a01b031663a3b1b31d6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561175e573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906117829190612a6b565b90505b6040805180820182526001815267ffffffffffffffff8381166020808401918252600087815260089091528490209251835491517fffffffffffffffffffffffffffffffffffffffffffffff0000000000000000009092169015157fffffffffffffffffffffffffffffffffffffffffffffff0000000000000000ff161761010091909216021790555182907fabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722906118409088908890612d02565b60405180910390a26040516002907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a25050505050565b6001546040517f16bf5579000000000000000000000000000000000000000000000000000000008152600481018390526000916001600160a01b0316906316bf557990602401602060405180830381865afa1580156118de573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906119029190612a6b565b92915050565b8260005a3360009081526003602052604090205490915060ff1615801561193a57506002546001600160a01b03163314155b1561195857604051632dd9fc9760e01b815260040160405180910390fd5b6000806119668a8a8a612434565b909250905060008b82848b8a8a8680806119838787838888611fdd565b929c5090945092509050888a1480159061199f57506000198914155b156119c75760405163ac7411c960e01b8152600481018b9052602481018a90526044016105d7565b80838b7f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548d6001604051611a029493929190612af8565b60405180910390a4505050505050505050807ffe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c208c8c604051611a45929190612d02565b60405180910390a25050506001600160a01b0382161561137657366000602061127883601f612b83565b6000548611611aaa576040517f7d73e6fa00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000611b8f8684611abe6020890189612d47565b611ace60408a0160208b01612d47565b611ad960018d612cc2565b6040805160f89690961b7fff000000000000000000000000000000000000000000000000000000000000001660208088019190915260609590951b6bffffffffffffffffffffffff1916602187015260c093841b7fffffffffffffffff00000000000000000000000000000000000000000000000090811660358801529290931b909116603d85015260458401526065830188905260858084018790528151808503909101815260a59093019052815191012090565b6004549091504390611ba46020880188612d47565b67ffffffffffffffff16611bb89190612b83565b10611bef576040517fad3515d900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6006544290611c046040880160208901612d47565b67ffffffffffffffff16611c189190612b83565b10611c4f576040517fc76d17e500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006001881115611cd8576001546001600160a01b031663d5719dc2611c7660028b612cc2565b6040518263ffffffff1660e01b8152600401611c9491815260200190565b602060405180830381865afa158015611cb1573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611cd59190612a6b565b90505b60408051602080820184905281830185905282518083038401815260609092019092528051910120600180546001600160a01b03169063d5719dc290611d1e908c612cc2565b6040518263ffffffff1660e01b8152600401611d3c91815260200190565b602060405180830381865afa158015611d59573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611d7d9190612a6b565b14611db4576040517f13947fd700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600080611dc08a611f98565b9150915060008a90506000600160009054906101000a90046001600160a01b03166001600160a01b0316635fca4a166040518163ffffffff1660e01b8152600401602060405180830381865afa158015611e1e573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611e429190612a6b565b9050600080548d83611e549190612b83565b611e5e9190612cc2565b9050600080600080611e74898860008989611fdd565b93509350935093508083857f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548d6002604051611eb79493929190612af8565b60405180910390a45050505050505050505050505050505050565b60408051600481526024810182526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f051038f200000000000000000000000000000000000000000000000000000000179052905160009182918291606491611f3e9190612d71565b600060405180830381855afa9150503d8060008114611f79576040519150601f19603f3d011682016040523d82523d6000602084013e611f7e565b606091505b5091509150818015611f91575080516020145b9250505090565b6040805160808101825260008082526020820181905291810182905260608101829052600080611fc78561267c565b8151602090920191909120969095509350505050565b60008060008060005488101561201f576040517f7d73e6fa00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600160009054906101000a90046001600160a01b03166001600160a01b031663eca067ad6040518163ffffffff1660e01b8152600401602060405180830381865afa158015612072573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906120969190612a6b565b8811156120cf576040517f925f8bd300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001546040517f86598a56000000000000000000000000000000000000000000000000000000008152600481018b9052602481018a905260448101889052606481018790526001600160a01b03909116906386598a56906084016080604051808303816000875af1158015612148573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061216c9190612d8d565b60008c9055929650909450925090508615612428573360607f0000000000000000000000000000000000000000000000000000000000000000156122f4576000606c6001600160a01b031663c6f7de0e6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156121eb573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061220f9190612a6b565b9050600061221d4883612b96565b905067ffffffffffffffff8111156122775760405162461bcd60e51b815260206004820152601160248201527f4c315f4741535f4e4f545f55494e54363400000000000000000000000000000060448201526064016105d7565b60408051426020820152606086901b6bffffffffffffffffffffffff191681830152605481018f9052607481018a905248609482015260c09290921b7fffffffffffffffff0000000000000000000000000000000000000000000000001660b48301528051609c81840301815260bc909201905291506123429050565b604080514260208201526bffffffffffffffffffffffff19606085901b1691810191909152605481018c90526074810187905248609482015260b40160405160208183030381529060405290505b600154815160208301206040517f7a88b1070000000000000000000000000000000000000000000000000000000081526000926001600160a01b031691637a88b107916123a79187916004016001600160a01b03929092168252602082015260400190565b6020604051808303816000875af11580156123c6573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906123ea9190612a6b565b9050807fff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b8360405161241c9190612ac5565b60405180910390a25050505b95509550955095915050565b604080516080810182526000808252602082018190529181018290526060810182905284846000612466826028612b83565b90507f00000000000000000000000000000000000000000000000000000000000000008111156124eb576040517f4634691b000000000000000000000000000000000000000000000000000000008152600481018290527f000000000000000000000000000000000000000000000000000000000000000060248201526044016105d7565b811580159061255757507f400000000000000000000000000000000000000000000000000000000000000080848460008161252857612528612d31565b9050013560f81c60f81b167effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916145b1561258e576040517f1f97007f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b602182108015906125d85750828260008181106125ad576125ad612d31565b909101357f800000000000000000000000000000000000000000000000000000000000000016151590505b1561262c5760006125ed602160018587612dc3565b6125f691612ded565b60008181526008602052604090205490915060ff1661262a5760405162f20c5d60e01b8152600481018290526024016105d7565b505b6000806126388861267c565b915091506000828b8b60405160200161265393929190612e0b565b60408051808303601f1901815291905280516020909101209b919a509098505050505050505050565b604080516080808201835260008083526020808401829052838501829052606080850183905285519384018652828452838201839052838601839052838101839052855191820183905260288201839052603082018390526038820183905260c087901b7fffffffffffffffff00000000000000000000000000000000000000000000000016958201959095526048016040516020818303038152906040529050602881511461272e5761272e612e33565b94909350915050565b60006020828403121561274957600080fd5b5035919050565b6001600160a01b038116811461276557600080fd5b50565b60008082840360a081121561277c57600080fd5b833561278781612750565b92506080601f198201121561279b57600080fd5b506020830190509250929050565b801515811461276557600080fd5b600080604083850312156127ca57600080fd5b82356127d581612750565b915060208301356127e5816127a9565b809150509250929050565b60006020828403121561280257600080fd5b813561280d81612750565b9392505050565b60008083601f84011261282657600080fd5b50813567ffffffffffffffff81111561283e57600080fd5b60208301915083602082850101111561285657600080fd5b9250929050565b60008060008060006080868803121561287557600080fd5b85359450602086013567ffffffffffffffff81111561289357600080fd5b61289f88828901612814565b9095509350506040860135915060608601356128ba81612750565b809150509295509295909350565b600080600080600080600060c0888a0312156128e357600080fd5b87359650602088013567ffffffffffffffff81111561290157600080fd5b61290d8a828b01612814565b90975095505060408801359350606088013561292881612750565b969995985093969295946080840135945060a09093013592915050565b60006080828403121561295757600080fd5b6040516080810181811067ffffffffffffffff8211171561298857634e487b7160e01b600052604160045260246000fd5b8060405250823581526020830135602082015260408301356040820152606083013560608201528091505092915050565b600080602083850312156129cc57600080fd5b823567ffffffffffffffff8111156129e357600080fd5b6129ef85828601612814565b90969095509350505050565b60008060008060008060e08789031215612a1457600080fd5b86359550602087013560ff81168114612a2c57600080fd5b94506080870188811115612a3f57600080fd5b60408801945035925060a0870135612a5681612750565b8092505060c087013590509295509295509295565b600060208284031215612a7d57600080fd5b5051919050565b600060208284031215612a9657600080fd5b815161280d81612750565b60005b83811015612abc578181015183820152602001612aa4565b50506000910152565b6020815260008251806020840152612ae4816040850160208701612aa1565b601f01601f19169190910160400192915050565b600060e08201905085825284602083015267ffffffffffffffff8085511660408401528060208601511660608401528060408601511660808401528060608601511660a08401525060038310612b5e57634e487b7160e01b600052602160045260246000fd5b8260c083015295945050505050565b634e487b7160e01b600052601160045260246000fd5b8082018082111561190257611902612b6d565b600082612bb357634e487b7160e01b600052601260045260246000fd5b500490565b600181815b80851115612bf3578160001904821115612bd957612bd9612b6d565b80851615612be657918102915b93841c9390800290612bbd565b509250929050565b600082612c0a57506001611902565b81612c1757506000611902565b8160018114612c2d5760028114612c3757612c53565b6001915050611902565b60ff841115612c4857612c48612b6d565b50506001821b611902565b5060208310610133831016604e8410600b8410161715612c76575081810a611902565b612c808383612bb8565b8060001904821115612c9457612c94612b6d565b029392505050565b600061280d60ff841683612bfb565b808202811582820484141761190257611902612b6d565b8181038181111561190257611902612b6d565b600060208284031215612ce757600080fd5b815161280d816127a9565b8183823760009101908152919050565b60208152816020820152818360408301376000818301604090810191909152601f909201601f19160101919050565b634e487b7160e01b600052603260045260246000fd5b600060208284031215612d5957600080fd5b813567ffffffffffffffff8116811461280d57600080fd5b60008251612d83818460208701612aa1565b9190910192915050565b60008060008060808587031215612da357600080fd5b505082516020840151604085015160609095015191969095509092509050565b60008085851115612dd357600080fd5b83861115612de057600080fd5b5050820193919092039150565b8035602083101561190257600019602084900360031b1b1692915050565b60008451612e1d818460208901612aa1565b8201838582376000930192835250909392505050565b634e487b7160e01b600052600160045260246000fdfea26469706673582212202a7695310a90203bbe2578b9d0123e1b0bec6f7c577557cf5bb56ea331b9be4064736f6c63430008110033",
}

// SequencerInboxStubABI is the input ABI used to generate the binding from.
// Deprecated: Use SequencerInboxStubMetaData.ABI instead.
var SequencerInboxStubABI = SequencerInboxStubMetaData.ABI

// SequencerInboxStubBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SequencerInboxStubMetaData.Bin instead.
var SequencerInboxStubBin = SequencerInboxStubMetaData.Bin

// DeploySequencerInboxStub deploys a new Ethereum contract, binding an instance of SequencerInboxStub to it.
func DeploySequencerInboxStub(auth *bind.TransactOpts, backend bind.ContractBackend, bridge_ common.Address, sequencer_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation, maxDataSize_ *big.Int) (common.Address, *types.Transaction, *SequencerInboxStub, error) {
	parsed, err := SequencerInboxStubMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SequencerInboxStubBin), backend, bridge_, sequencer_, maxTimeVariation_, maxDataSize_)
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
// Solidity: function maxTimeVariation() view returns(uint256 delayBlocks, uint256 futureBlocks, uint256 delaySeconds, uint256 futureSeconds)
func (_SequencerInboxStub *SequencerInboxStubCaller) MaxTimeVariation(opts *bind.CallOpts) (struct {
	DelayBlocks   *big.Int
	FutureBlocks  *big.Int
	DelaySeconds  *big.Int
	FutureSeconds *big.Int
}, error) {
	var out []interface{}
	err := _SequencerInboxStub.contract.Call(opts, &out, "maxTimeVariation")

	outstruct := new(struct {
		DelayBlocks   *big.Int
		FutureBlocks  *big.Int
		DelaySeconds  *big.Int
		FutureSeconds *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.DelayBlocks = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FutureBlocks = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.DelaySeconds = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.FutureSeconds = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256 delayBlocks, uint256 futureBlocks, uint256 delaySeconds, uint256 futureSeconds)
func (_SequencerInboxStub *SequencerInboxStubSession) MaxTimeVariation() (struct {
	DelayBlocks   *big.Int
	FutureBlocks  *big.Int
	DelaySeconds  *big.Int
	FutureSeconds *big.Int
}, error) {
	return _SequencerInboxStub.Contract.MaxTimeVariation(&_SequencerInboxStub.CallOpts)
}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256 delayBlocks, uint256 futureBlocks, uint256 delaySeconds, uint256 futureSeconds)
func (_SequencerInboxStub *SequencerInboxStubCallerSession) MaxTimeVariation() (struct {
	DelayBlocks   *big.Int
	FutureBlocks  *big.Int
	DelaySeconds  *big.Int
	FutureSeconds *big.Int
}, error) {
	return _SequencerInboxStub.Contract.MaxTimeVariation(&_SequencerInboxStub.CallOpts)
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

// AddSequencerL2BatchFromOrigin is a paid mutator transaction binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) AddSequencerL2BatchFromOrigin(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "addSequencerL2BatchFromOrigin", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder)
}

// AddSequencerL2BatchFromOrigin is a paid mutator transaction binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) AddSequencerL2BatchFromOrigin(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromOrigin(&_SequencerInboxStub.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder)
}

// AddSequencerL2BatchFromOrigin is a paid mutator transaction binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) AddSequencerL2BatchFromOrigin(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.AddSequencerL2BatchFromOrigin(&_SequencerInboxStub.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder)
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

// Initialize is a paid mutator transaction binding the contract method 0x1f7a92b2.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactor) Initialize(opts *bind.TransactOpts, bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInboxStub.contract.Transact(opts, "initialize", bridge_, maxTimeVariation_)
}

// Initialize is a paid mutator transaction binding the contract method 0x1f7a92b2.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInboxStub *SequencerInboxStubSession) Initialize(bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.Initialize(&_SequencerInboxStub.TransactOpts, bridge_, maxTimeVariation_)
}

// Initialize is a paid mutator transaction binding the contract method 0x1f7a92b2.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInboxStub *SequencerInboxStubTransactorSession) Initialize(bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInboxStub.Contract.Initialize(&_SequencerInboxStub.TransactOpts, bridge_, maxTimeVariation_)
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
	TimeBounds               ISequencerInboxTimeBounds
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
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"count\",\"type\":\"uint64\"}],\"name\":\"CounterEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"NullEvent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"redeemer\",\"type\":\"address\"}],\"name\":\"RedeemedEvent\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"checkBlockHashes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"useTopLevel\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"directCase\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"staticCase\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"delegateCase\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"callcodeCase\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"callCase\",\"type\":\"bool\"}],\"name\":\"checkCalls\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"useTopLevel\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"expected\",\"type\":\"bool\"}],\"name\":\"checkIsTopLevelOrWasAliased\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"counter\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"emitNullEvent\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBlockDifficulty\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"increment\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"incrementEmit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"incrementRedeem\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"noop\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pleaseRevert\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610cc5806100206000396000f3fe608060405234801561001057600080fd5b50600436106100c95760003560e01c80635dfc2e4a11610081578063b226a9641161005b578063b226a96414610145578063d09de08a1461014d578063ded5ecad1461015557600080fd5b80635dfc2e4a146100d657806361bc221a146101105780639ff5ccac1461013d57600080fd5b806312e05dd1116100b257806312e05dd1146100e057806344c25fba146100f55780635677c11e1461010857600080fd5b806305795f73146100ce5780630e8c389f146100d8575b600080fd5b6100d6610168565b005b6100d66101b5565b445b6040519081526020015b60405180910390f35b6100d6610103366004610adc565b6103c5565b6100e2610868565b6000546101249067ffffffffffffffff1681565b60405167ffffffffffffffff90911681526020016100ec565b6100d66108d4565b6100d6610943565b6100d661096e565b6100d6610163366004610b5e565b6109b0565b60405162461bcd60e51b815260206004820152601260248201527f534f4c49444954595f524556455254494e47000000000000000000000000000060448201526064015b60405180910390fd5b3332146102045760405162461bcd60e51b815260206004820152601160248201527f53454e4445525f4e4f545f4f524947494e00000000000000000000000000000060448201526064016101ac565b606473ffffffffffffffffffffffffffffffffffffffff1663175a260b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610250573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906102749190610b97565b6102c05760405162461bcd60e51b815260206004820152600b60248201527f4e4f545f414c494153454400000000000000000000000000000000000000000060448201526064016101ac565b6000805467ffffffffffffffff1690806102d983610bea565b91906101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550507f773c78bf96e65f61c1a2622b47d76e78bfe70dd59cf4f11470c4c121c315941333606e73ffffffffffffffffffffffffffffffffffffffff1663de4ba2b36040518163ffffffff1660e01b8152600401602060405180830381865afa15801561036e573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906103929190610c11565b6040805173ffffffffffffffffffffffffffffffffffffffff9384168152929091166020830152015b60405180910390a1565b851561049257841515606473ffffffffffffffffffffffffffffffffffffffff166308bd624c6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561041a573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061043e9190610b97565b15151461048d5760405162461bcd60e51b815260206004820152601160248201527f554e45585045435445445f524553554c5400000000000000000000000000000060448201526064016101ac565b610554565b841515606473ffffffffffffffffffffffffffffffffffffffff1663175a260b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156104e1573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105059190610b97565b1515146105545760405162461bcd60e51b815260206004820152601160248201527f554e45585045435445445f524553554c5400000000000000000000000000000060448201526064016101ac565b60405163ded5ecad60e01b815286151560048201528415156024820152309063ded5ecad9060440160006040518083038186803b15801561059457600080fd5b505afa1580156105a8573d6000803e3d6000fd5b505060408051891515602482015286151560448083019190915282518083039091018152606490910182526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1663ded5ecad60e01b1790529051909250600091503090610618908490610c47565b600060405180830381855af49150503d8060008114610653576040519150601f19603f3d011682016040523d82523d6000602084013e610658565b606091505b50509050806106a95760405162461bcd60e51b815260206004820152601460248201527f44454c45474154455f43414c4c5f4641494c454400000000000000000000000060448201526064016101ac565b6040805189151560248201528515156044808301919091528251808303909101815260649091019091526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1663ded5ecad60e01b1781528151919350600091829182305af290508061075f5760405162461bcd60e51b815260206004820152600f60248201527f43414c4c434f44455f4641494c4544000000000000000000000000000000000060448201526064016101ac565b60408051891515602482015284151560448083019190915282518083039091018152606490910182526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1663ded5ecad60e01b179052905190925030906107c9908490610c47565b6000604051808303816000865af19150503d8060008114610806576040519150601f19603f3d011682016040523d82523d6000602084013e61080b565b606091505b5050809150508061085e5760405162461bcd60e51b815260206004820152600b60248201527f43414c4c5f4641494c454400000000000000000000000000000000000000000060448201526064016101ac565b5050505050505050565b6000610875600243610c76565b40610881600143610c76565b40036108cf5760405162461bcd60e51b815260206004820152600f60248201527f53414d455f424c4f434b5f48415348000000000000000000000000000000000060448201526064016101ac565b504390565b6000805467ffffffffffffffff1690806108ed83610bea565b82546101009290920a67ffffffffffffffff818102199093169183160217909155600054604051911681527fa45d7e79cb3c6044f30c8dd891e6571301d6b8b6618df519c987905ec70742e791506020016103bb565b6040517f6f59c82101949290205a9ae9d0c657e6dae1a71c301ae76d385c2792294585fe90600090a1565b6000805467ffffffffffffffff16908061098783610bea565b91906101000a81548167ffffffffffffffff021916908367ffffffffffffffff16021790555050565b8115610a7c57801515606473ffffffffffffffffffffffffffffffffffffffff166308bd624c6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610a05573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610a299190610b97565b151514610a785760405162461bcd60e51b815260206004820152601160248201527f554e45585045435445445f524553554c5400000000000000000000000000000060448201526064016101ac565b5050565b801515606473ffffffffffffffffffffffffffffffffffffffff1663175a260b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610a05573d6000803e3d6000fd5b8015158114610ad957600080fd5b50565b60008060008060008060c08789031215610af557600080fd5b8635610b0081610acb565b95506020870135610b1081610acb565b94506040870135610b2081610acb565b93506060870135610b3081610acb565b92506080870135610b4081610acb565b915060a0870135610b5081610acb565b809150509295509295509295565b60008060408385031215610b7157600080fd5b8235610b7c81610acb565b91506020830135610b8c81610acb565b809150509250929050565b600060208284031215610ba957600080fd5b8151610bb481610acb565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600067ffffffffffffffff808316818103610c0757610c07610bbb565b6001019392505050565b600060208284031215610c2357600080fd5b815173ffffffffffffffffffffffffffffffffffffffff81168114610bb457600080fd5b6000825160005b81811015610c685760208186018101518583015201610c4e565b506000920191825250919050565b81810381811115610c8957610c89610bbb565b9291505056fea26469706673582212208a01c3eb4bf7ce53cdb77b91c7953601bd35e32c3907b2b5be380678ee6ada4464736f6c63430008110033",
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
	ABI: "[{\"inputs\":[],\"name\":\"STEPS_PER_BATCH\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"}],\"internalType\":\"structExecutionState\",\"name\":\"execState\",\"type\":\"tuple\"}],\"name\":\"getMachineHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"initialWasmModuleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"step\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"proveOneStep\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"afterHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506108fa806100206000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c80639c2009cd14610046578063b5112fd21461006d578063c39619c41461008e575b600080fd5b61004f6107d081565b60405167ffffffffffffffff90911681526020015b60405180910390f35b61008061007b3660046105ba565b6100a1565b604051908152602001610064565b61008061009c366004610656565b6103c5565b60008181036100f75760405162461bcd60e51b815260206004820152600b60248201527f454d5054595f50524f4f4600000000000000000000000000000000000000000060448201526064015b60405180910390fd5b6100ff610577565b600061010c8585836104eb565b602084015167ffffffffffffffff909216909152905061012d8585836104eb565b60208481015167ffffffffffffffff9093169201919091529050861580159061019657508560001a60f81b7fff00000000000000000000000000000000000000000000000000000000000000161580610196575061018a82610553565b67ffffffffffffffff16155b156101a55785925050506103bc565b87356101b083610569565b67ffffffffffffffff16106101c95785925050506103bc565b8151805160209182015182850151805190840151604080517f476c6f62616c2073746174653a0000000000000000000000000000000000000081880152602d810195909552604d8501939093527fffffffffffffffff00000000000000000000000000000000000000000000000060c092831b8116606d860152911b1660758301528051808303605d018152607d9092019052805191012086146102af5760405162461bcd60e51b815260206004820152600960248201527f4241445f50524f4f46000000000000000000000000000000000000000000000060448201526064016100ee565b602082810151018051906102c28261069a565b67ffffffffffffffff1690525060208281015101516102e4906107d0906106c1565b67ffffffffffffffff1660000361031f5760208201518051906103068261069a565b67ffffffffffffffff1690525060208281015160009101525b8151805160209182015182850151805190840151604080517f476c6f62616c2073746174653a0000000000000000000000000000000000000081880152602d810195909552604d8501939093527fffffffffffffffff00000000000000000000000000000000000000000000000060c092831b8116606d860152911b1660758301528051808303605d018152607d90920190528051910120925050505b95945050505050565b600060016103d960a084016080850161070c565b60028111156103ea576103ea6106f6565b146104375760405162461bcd60e51b815260206004820152601260248201527f4241445f4d414348494e455f535441545553000000000000000000000000000060448201526064016100ee565b6104e5610449368490038401846107f3565b8051805160209182015192820151805190830151604080517f476c6f62616c2073746174653a0000000000000000000000000000000000000081870152602d810194909452604d8401959095527fffffffffffffffff00000000000000000000000000000000000000000000000060c092831b8116606d850152911b1660758201528251808203605d018152607d909101909252815191012090565b92915050565b600081815b600881101561054a5760088367ffffffffffffffff16901b925085858381811061051c5761051c61066e565b919091013560f81c939093179250816105348161088c565b92505080806105429061088c565b9150506104f0565b50935093915050565b602081015160009060015b602002015192915050565b60208101516000908161055e565b604051806040016040528061058a61059c565b815260200161059761059c565b905290565b60405180604001604052806002906020820280368337509192915050565b600080600080600085870360c08112156105d357600080fd5b60608112156105e157600080fd5b50859450606086013593506080860135925060a086013567ffffffffffffffff8082111561060e57600080fd5b818801915088601f83011261062257600080fd5b81358181111561063157600080fd5b89602082850101111561064357600080fd5b9699959850939650602001949392505050565b600060a0828403121561066857600080fd5b50919050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b600067ffffffffffffffff8083168181036106b7576106b7610684565b6001019392505050565b600067ffffffffffffffff808416806106ea57634e487b7160e01b600052601260045260246000fd5b92169190910692915050565b634e487b7160e01b600052602160045260246000fd5b60006020828403121561071e57600080fd5b81356003811061072d57600080fd5b9392505050565b634e487b7160e01b600052604160045260246000fd5b6040805190810167ffffffffffffffff8111828210171561076d5761076d610734565b60405290565b600082601f83011261078457600080fd5b6040516040810167ffffffffffffffff82821081831117156107a8576107a8610734565b604091825282918501868111156107be57600080fd5b855b818110156107e757803583811681146107d95760008081fd5b8452602093840193016107c0565b50929695505050505050565b60006080828403121561080557600080fd5b6040516040810181811067ffffffffffffffff8211171561082857610828610734565b604052601f8301841361083a57600080fd5b61084261074a565b80604085018681111561085457600080fd5b855b8181101561086e578035845260209384019301610856565b5081845261087c8782610773565b6020850152509195945050505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036108bd576108bd610684565b506001019056fea2646970667358221220174b7c204b232861bfe6663867a4401c9aecf255f4e2741a8875231ea57e044e64736f6c63430008110033",
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
	Bin: "0x60a060405234801561001057600080fd5b5060405161011d38038061011d83398101604081905261002f91610040565b6001600160a01b0316608052610070565b60006020828403121561005257600080fd5b81516001600160a01b038116811461006957600080fd5b9392505050565b608051609561008860003960006017015260956000f3fe608060405236601057600e6013565b005b600e5b603a7f0000000000000000000000000000000000000000000000000000000000000000603c565b565b3660008037600080366000845af43d6000803e808015605a573d6000f35b3d6000fdfea2646970667358221220f7a591d7c78165ab26b2469cfd455a69fb64ba59095a2cbc0894e967d0b3c6dc64736f6c63430008110033",
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

// SingleExecutionChallengeMetaData contains all meta data concerning the SingleExecutionChallenge contract.
var SingleExecutionChallengeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"osp_\",\"type\":\"address\"},{\"internalType\":\"contractIOldChallengeResultReceiver\",\"name\":\"resultReceiver_\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"maxInboxMessagesRead_\",\"type\":\"uint64\"},{\"internalType\":\"bytes32[2]\",\"name\":\"startAndEndHashes\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint256\",\"name\":\"numSteps_\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"asserter_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"challenger_\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asserterTimeLeft_\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"challengerTimeLeft_\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"challengeRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"challengedSegmentStart\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"challengedSegmentLength\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32[]\",\"name\":\"chainHashes\",\"type\":\"bytes32[]\"}],\"name\":\"Bisected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"enumIOldChallengeManager.ChallengeTerminationType\",\"name\":\"kind\",\"type\":\"uint8\"}],\"name\":\"ChallengeEnded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockSteps\",\"type\":\"uint256\"}],\"name\":\"ExecutionChallengeBegun\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"indexed\":false,\"internalType\":\"structGlobalState\",\"name\":\"startState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"indexed\":false,\"internalType\":\"structGlobalState\",\"name\":\"endState\",\"type\":\"tuple\"}],\"name\":\"InitiatedChallenge\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"OneStepProofCompleted\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"oldSegmentsStart\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oldSegmentsLength\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"oldSegments\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"challengePosition\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.SegmentSelection\",\"name\":\"selection\",\"type\":\"tuple\"},{\"internalType\":\"bytes32[]\",\"name\":\"newSegments\",\"type\":\"bytes32[]\"}],\"name\":\"bisectExecution\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"oldSegmentsStart\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oldSegmentsLength\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"oldSegments\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"challengePosition\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.SegmentSelection\",\"name\":\"selection\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus[2]\",\"name\":\"machineStatuses\",\"type\":\"uint8[2]\"},{\"internalType\":\"bytes32[2]\",\"name\":\"globalStateHashes\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint256\",\"name\":\"numSteps\",\"type\":\"uint256\"}],\"name\":\"challengeExecution\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"challengeInfo\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timeLeft\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.Participant\",\"name\":\"current\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timeLeft\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.Participant\",\"name\":\"next\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"lastMoveTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"challengeStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"maxInboxMessages\",\"type\":\"uint64\"},{\"internalType\":\"enumOldChallengeLib.ChallengeMode\",\"name\":\"mode\",\"type\":\"uint8\"}],\"internalType\":\"structOldChallengeLib.Challenge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"challenges\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timeLeft\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.Participant\",\"name\":\"current\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timeLeft\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.Participant\",\"name\":\"next\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"lastMoveTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"challengeStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"maxInboxMessages\",\"type\":\"uint64\"},{\"internalType\":\"enumOldChallengeLib.ChallengeMode\",\"name\":\"mode\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"clearChallenge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot_\",\"type\":\"bytes32\"},{\"internalType\":\"enumMachineStatus[2]\",\"name\":\"startAndEndMachineStatuses_\",\"type\":\"uint8[2]\"},{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState[2]\",\"name\":\"startAndEndGlobalStates_\",\"type\":\"tuple[2]\"},{\"internalType\":\"uint64\",\"name\":\"numBlocks\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"asserter_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"challenger_\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asserterTimeLeft_\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"challengerTimeLeft_\",\"type\":\"uint256\"}],\"name\":\"createChallenge\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"currentResponder\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOldChallengeResultReceiver\",\"name\":\"resultReceiver_\",\"type\":\"address\"},{\"internalType\":\"contractISequencerInbox\",\"name\":\"sequencerInbox_\",\"type\":\"address\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge_\",\"type\":\"address\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"osp_\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"isTimedOut\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"oldSegmentsStart\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"oldSegmentsLength\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"oldSegments\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"challengePosition\",\"type\":\"uint256\"}],\"internalType\":\"structOldChallengeLib.SegmentSelection\",\"name\":\"selection\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"oneStepProveExecution\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"osp\",\"outputs\":[{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"resultReceiver\",\"outputs\":[{\"internalType\":\"contractIOldChallengeResultReceiver\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"contractISequencerInbox\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"challengeIndex\",\"type\":\"uint64\"}],\"name\":\"timeout\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalChallengesCreated\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a0604052306080523480156200001557600080fd5b50604051620039be380380620039be8339810160408190526200003891620002bc565b600580546001600160a01b03808c166001600160a01b03199283161790925560028054928b1692909116919091179055600080548190819062000084906001600160401b0316620003cd565b82546101009290920a6001600160401b03818102199093168284169182021790935560009283526001602090815260408085206007810180546001600160401b031916958f16959095179094558051600280825260608201835293965093949392918301908036833750508a518251929350918391506000906200010c576200010c6200040a565b60209081029190910101528860016020020151816001815181106200013557620001356200040a565b60200260200101818152505060006200015c60008a846200024360201b62001f091760201c565b600684018190556040805180820182526001600160a01b038b811680835260209283018b90526002880180546001600160a01b03199081169092179055600388018b905583518085018552918c168083529190920189905286549091161785556001850187905542600486015560078501805460ff60401b1916680200000000000000001790555190915081906001600160401b038616907f86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d689062000228906000908e90889062000420565b60405180910390a350505050505050505050505050620004bb565b60008383836040516020016200025c9392919062000477565b6040516020818303038152906040528051906020012090509392505050565b6001600160a01b03811681146200029157600080fd5b50565b634e487b7160e01b600052604160045260246000fd5b8051620002b7816200027b565b919050565b60008060008060008060008060006101408a8c031215620002dc57600080fd5b8951620002e9816200027b565b809950506020808b0151620002fe816200027b565b60408c01519099506001600160401b0380821682146200031d57600080fd5b8199508d607f8e01126200033057600080fd5b604051915060408201828110828211171562000350576200035062000294565b604052508060a08d018e8111156200036757600080fd5b60608e015b818110156200038557805183529184019184016200036c565b50519198509096506200039e91505060c08b01620002aa565b9350620003ae60e08b01620002aa565b92506101008a015191506101208a015190509295985092959850929598565b60006001600160401b038281166002600160401b031981016200040057634e487b7160e01b600052601160045260246000fd5b6001019392505050565b634e487b7160e01b600052603260045260246000fd5b6000606082018583526020858185015260606040850152818551808452608086019150828701935060005b8181101562000469578451835293830193918301916001016200044b565b509098975050505050505050565b83815260006020848184015260408301845182860160005b82811015620004ad578151845292840192908401906001016200048f565b509198975050505050505050565b6080516134e7620004d7600039600061155501526134e76000f3fe608060405234801561001057600080fd5b50600436106101005760003560e01c80639ede42b911610097578063ee35f32711610066578063ee35f327146102f0578063f26a62c614610303578063f8c8765e14610316578063fb7be0a11461032957600080fd5b80639ede42b914610294578063a521b032146102b7578063d248d124146102ca578063e78cea92146102dd57600080fd5b806356e9df97116100d357806356e9df97146101a95780635ef489e6146101bc5780637fd07a9c146101d05780638f1d3776146101f057600080fd5b806314eab5e7146101055780631b45c86a1461013657806323a9ef231461014b5780633504f1d714610196575b600080fd5b610118610113366004612b8c565b61033c565b60405167ffffffffffffffff90911681526020015b60405180910390f35b610149610144366004612c1f565b61064c565b005b61017e610159366004612c1f565b67ffffffffffffffff166000908152600160205260409020546001600160a01b031690565b6040516001600160a01b03909116815260200161012d565b60025461017e906001600160a01b031681565b6101496101b7366004612c1f565b61072a565b6000546101189067ffffffffffffffff1681565b6101e36101de366004612c1f565b6108b4565b60405161012d9190612c64565b6102816101fe366004612cee565b6001602081815260009283526040928390208351808501855281546001600160a01b03908116825293820154818401528451808601909552600282015490931684526003810154918401919091526004810154600582015460068301546007909301549394939192909167ffffffffffffffff811690600160401b900460ff1687565b60405161012d9796959493929190612d07565b6102a76102a2366004612c1f565b6109dc565b604051901515815260200161012d565b6101496102c5366004612d7f565b610a04565b6101496102d8366004612e24565b610fbb565b60045461017e906001600160a01b031681565b60035461017e906001600160a01b031681565b60055461017e906001600160a01b031681565b610149610324366004612eb7565b61154b565b610149610337366004612f13565b6116f5565b6002546000906001600160a01b0316331461039e5760405162461bcd60e51b815260206004820152601060248201527f4f4e4c595f524f4c4c55505f4348414c0000000000000000000000000000000060448201526064015b60405180910390fd5b6040805160028082526060820183526000926020830190803683370190505090506103f46103cf60208b018b612fb8565b6103ef8a60005b608002018036038101906103ea9190613079565b611f40565b611fe9565b8160008151811061040757610407612fa2565b602090810291909101015261043689600160200201602081019061042b9190612fb8565b6103ef8a60016103d6565b8160018151811061044957610449612fa2565b6020908102919091010152600080548190819061046f9067ffffffffffffffff16613128565b825467ffffffffffffffff8083166101009490940a848102910219909116179092559091506104a0576104a061314f565b67ffffffffffffffff81166000908152600160205260408120600581018d9055906104db6104d6368d90038d0160808e01613079565b6120ee565b905060026104ef60408e0160208f01612fb8565b600281111561050057610500612c3a565b148061052f5750600061052361051e368e90038e0160808f01613079565b612103565b67ffffffffffffffff16115b15610542578061053e81613128565b9150505b6007820180546040805180820182526001600160a01b038d811680835260209283018d905260028801805473ffffffffffffffffffffffffffffffffffffffff199081169092179055600388018d905583518085018552918e16808352919092018b905286549091161785556001850189905542600486015567ffffffffffffffff84811668ffffffffffffffffff1990931692909217600160401b179092559051908416907f76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a9061061a908e9060808201906131ad565b60405180910390a26106398360008c67ffffffffffffffff1687612112565b5090925050505b98975050505050505050565b600067ffffffffffffffff8216600090815260016020526040902060070154600160401b900460ff16600281111561068657610686612c3a565b1415604051806040016040528060078152602001661393d7d0d2105360ca1b815250906106c65760405162461bcd60e51b815260040161039591906131c9565b506106d0816109dc565b61071c5760405162461bcd60e51b815260206004820152601060248201527f54494d454f55545f444541444c494e45000000000000000000000000000000006044820152606401610395565b6107278160006121a9565b50565b6002546001600160a01b031633146107845760405162461bcd60e51b815260206004820152601060248201527f4e4f545f5245535f5245434549564552000000000000000000000000000000006044820152606401610395565b600067ffffffffffffffff8216600090815260016020526040902060070154600160401b900460ff1660028111156107be576107be612c3a565b1415604051806040016040528060078152602001661393d7d0d2105360ca1b815250906107fe5760405162461bcd60e51b815260040161039591906131c9565b5067ffffffffffffffff81166000818152600160208190526040808320805473ffffffffffffffffffffffffffffffffffffffff1990811682559281018490556002810180549093169092556003808301849055600483018490556005830184905560068301939093556007909101805468ffffffffffffffffff19169055517ffdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40916108a991613217565b60405180910390a250565b6040805161012081018252600060e0820181815261010083018290528252825180840184528181526020808201839052830152918101829052606081018290526080810182905260a0810182905260c081019190915267ffffffffffffffff82811660009081526001602081815260409283902083516101208101855281546001600160a01b0390811660e0830190815294830154610100830152938152845180860186526002808401549095168152600383015481850152928101929092526004810154938201939093526005830154606082015260068301546080820152600783015493841660a08201529260c0840191600160401b90910460ff16908111156109c2576109c2612c3a565b60028111156109d3576109d3612c3a565b90525092915050565b67ffffffffffffffff811660009081526001602052604081206109fe906122ff565b92915050565b67ffffffffffffffff8416600090815260016020526040812080548692869290916001600160a01b03163314610a7c5760405162461bcd60e51b815260206004820152600b60248201527f4348414c5f53454e4445520000000000000000000000000000000000000000006044820152606401610395565b610a85846109dc565b15610ad25760405162461bcd60e51b815260206004820152600d60248201527f4348414c5f444541444c494e45000000000000000000000000000000000000006044820152606401610395565b6000826002811115610ae657610ae6612c3a565b03610b535760006007820154600160401b900460ff166002811115610b0d57610b0d612c3a565b1415604051806040016040528060078152602001661393d7d0d2105360ca1b81525090610b4d5760405162461bcd60e51b815260040161039591906131c9565b50610c70565b6001826002811115610b6757610b67612c3a565b03610be05760016007820154600160401b900460ff166002811115610b8e57610b8e612c3a565b14610bdb5760405162461bcd60e51b815260206004820152600e60248201527f4348414c5f4e4f545f424c4f434b0000000000000000000000000000000000006044820152606401610395565b610c70565b6002826002811115610bf457610bf4612c3a565b03610c685760026007820154600160401b900460ff166002811115610c1b57610c1b612c3a565b14610bdb5760405162461bcd60e51b815260206004820152601260248201527f4348414c5f4e4f545f455845435554494f4e00000000000000000000000000006044820152606401610395565b610c7061314f565b610cbe83356020850135610c876040870187613231565b80806020026020016040519081016040528093929190818152602001838360200280828437600092019190915250611f0992505050565b816006015414610d105760405162461bcd60e51b815260206004820152600960248201527f4249535f535441544500000000000000000000000000000000000000000000006044820152606401610395565b6002610d1f6040850185613231565b90501080610d4a57506001610d376040850185613231565b610d429291506132a0565b836060013510155b15610d975760405162461bcd60e51b815260206004820152601160248201527f4241445f4348414c4c454e47455f504f530000000000000000000000000000006044820152606401610395565b600080610da389612317565b9150915060018111610df75760405162461bcd60e51b815260206004820152600960248201527f544f4f5f53484f525400000000000000000000000000000000000000000000006044820152606401610395565b806028811115610e05575060285b610e108160016132b3565b8814610e5e5760405162461bcd60e51b815260206004820152600c60248201527f57524f4e475f44454752454500000000000000000000000000000000000000006044820152606401610395565b50610ea88989896000818110610e7657610e76612fa2565b602002919091013590508a8a610e8d6001826132a0565b818110610e9c57610e9c612fa2565b905060200201356123a7565b610ee78a83838b8b8080602002602001604051908101604052809392919081815260200183836020028082843760009201919091525061211292505050565b50600090505b6007820154600160401b900460ff166002811115610f0d57610f0d612c3a565b03610f185750610fb2565b6040805180820190915281546001600160a01b03168152600182015460208201526004820154610f4890426132a0565b81602001818151610f5991906132a0565b90525060028201805483546001600160a01b0380831673ffffffffffffffffffffffffffffffffffffffff1992831617865560038601805460018801558551929093169116179091556020909101519055426004909101555b50505050505050565b67ffffffffffffffff84166000908152600160205260409020805485918591600291906001600160a01b031633146110355760405162461bcd60e51b815260206004820152600b60248201527f4348414c5f53454e4445520000000000000000000000000000000000000000006044820152606401610395565b61103e846109dc565b1561108b5760405162461bcd60e51b815260206004820152600d60248201527f4348414c5f444541444c494e45000000000000000000000000000000000000006044820152606401610395565b600082600281111561109f5761109f612c3a565b0361110c5760006007820154600160401b900460ff1660028111156110c6576110c6612c3a565b1415604051806040016040528060078152602001661393d7d0d2105360ca1b815250906111065760405162461bcd60e51b815260040161039591906131c9565b50611229565b600182600281111561112057611120612c3a565b036111995760016007820154600160401b900460ff16600281111561114757611147612c3a565b146111945760405162461bcd60e51b815260206004820152600e60248201527f4348414c5f4e4f545f424c4f434b0000000000000000000000000000000000006044820152606401610395565b611229565b60028260028111156111ad576111ad612c3a565b036112215760026007820154600160401b900460ff1660028111156111d4576111d4612c3a565b146111945760405162461bcd60e51b815260206004820152601260248201527f4348414c5f4e4f545f455845435554494f4e00000000000000000000000000006044820152606401610395565b61122961314f565b61124083356020850135610c876040870187613231565b8160060154146112925760405162461bcd60e51b815260206004820152600960248201527f4249535f535441544500000000000000000000000000000000000000000000006044820152606401610395565b60026112a16040850185613231565b905010806112cc575060016112b96040850185613231565b6112c49291506132a0565b836060013510155b156113195760405162461bcd60e51b815260206004820152601160248201527f4241445f4348414c4c454e47455f504f530000000000000000000000000000006044820152606401610395565b67ffffffffffffffff88166000908152600160205260408120908061133d8a612317565b9092509050600181146113925760405162461bcd60e51b815260206004820152600860248201527f544f4f5f4c4f4e470000000000000000000000000000000000000000000000006044820152606401610395565b506005805460408051606081018252600786015467ffffffffffffffff1681526004546001600160a01b03908116602083015293860154818301526000939092169163b5112fd29185906113e8908f018f613231565b8f606001358181106113fc576113fc612fa2565b905060200201358d8d6040518663ffffffff1660e01b81526004016114259594939291906132c6565b602060405180830381865afa158015611442573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114669190613328565b905061147560408b018b613231565b61148460608d013560016132b3565b81811061149357611493612fa2565b9050602002013581036114e85760405162461bcd60e51b815260206004820152600c60248201527f53414d455f4f53505f454e4400000000000000000000000000000000000000006044820152606401610395565b60405167ffffffffffffffff8c16907fc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e90600090a261153f8b67ffffffffffffffff16600090815260016020526040812060060155565b5060009150610eed9050565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036115e95760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c00000000000000000000000000000000000000006064820152608401610395565b6002546001600160a01b0316156116425760405162461bcd60e51b815260206004820152600c60248201527f414c52454144595f494e495400000000000000000000000000000000000000006044820152606401610395565b6001600160a01b0384166116985760405162461bcd60e51b815260206004820152601260248201527f4e4f5f524553554c545f524543454956455200000000000000000000000000006044820152606401610395565b600280546001600160a01b0395861673ffffffffffffffffffffffffffffffffffffffff19918216179091556003805494861694821694909417909355600480549285169284169290921790915560058054919093169116179055565b67ffffffffffffffff8516600090815260016020819052604090912080548792879290916001600160a01b031633146117705760405162461bcd60e51b815260206004820152600b60248201527f4348414c5f53454e4445520000000000000000000000000000000000000000006044820152606401610395565b611779846109dc565b156117c65760405162461bcd60e51b815260206004820152600d60248201527f4348414c5f444541444c494e45000000000000000000000000000000000000006044820152606401610395565b60008260028111156117da576117da612c3a565b036118475760006007820154600160401b900460ff16600281111561180157611801612c3a565b1415604051806040016040528060078152602001661393d7d0d2105360ca1b815250906118415760405162461bcd60e51b815260040161039591906131c9565b50611964565b600182600281111561185b5761185b612c3a565b036118d45760016007820154600160401b900460ff16600281111561188257611882612c3a565b146118cf5760405162461bcd60e51b815260206004820152600e60248201527f4348414c5f4e4f545f424c4f434b0000000000000000000000000000000000006044820152606401610395565b611964565b60028260028111156118e8576118e8612c3a565b0361195c5760026007820154600160401b900460ff16600281111561190f5761190f612c3a565b146118cf5760405162461bcd60e51b815260206004820152601260248201527f4348414c5f4e4f545f455845435554494f4e00000000000000000000000000006044820152606401610395565b61196461314f565b61197b83356020850135610c876040870187613231565b8160060154146119cd5760405162461bcd60e51b815260206004820152600960248201527f4249535f535441544500000000000000000000000000000000000000000000006044820152606401610395565b60026119dc6040850185613231565b90501080611a07575060016119f46040850185613231565b6119ff9291506132a0565b836060013510155b15611a545760405162461bcd60e51b815260206004820152601160248201527f4241445f4348414c4c454e47455f504f530000000000000000000000000000006044820152606401610395565b6001851015611aa55760405162461bcd60e51b815260206004820152601360248201527f4348414c4c454e47455f544f4f5f53484f5254000000000000000000000000006044820152606401610395565b65080000000000851115611afb5760405162461bcd60e51b815260206004820152601260248201527f4348414c4c454e47455f544f4f5f4c4f4e4700000000000000000000000000006044820152606401610395565b611b3d88611b1d611b0f60208b018b612fb8565b8960005b6020020135611fe9565b611b38611b3060408c0160208d01612fb8565b8a6001611b13565b6123a7565b67ffffffffffffffff891660009081526001602052604081209080611b618b612317565b9150915080600114611bb55760405162461bcd60e51b815260206004820152600860248201527f544f4f5f4c4f4e470000000000000000000000000000000000000000000000006044820152606401610395565b6001611bc460208c018c612fb8565b6002811115611bd557611bd5612c3a565b14611ca057611bea60408b0160208c01612fb8565b6002811115611bfb57611bfb612c3a565b611c0860208c018c612fb8565b6002811115611c1957611c19612c3a565b148015611c2a5750883560208a0135145b611c765760405162461bcd60e51b815260206004820152600d60248201527f48414c5445445f4348414e4745000000000000000000000000000000000000006044820152606401610395565b611c988c67ffffffffffffffff16600090815260016020526040812060060155565b505050611e38565b6002611cb260408c0160208d01612fb8565b6002811115611cc357611cc3612c3a565b03611d1c57883560208a013514611d1c5760405162461bcd60e51b815260206004820152600c60248201527f4552524f525f4348414e474500000000000000000000000000000000000000006044820152606401610395565b6040805160028082526060820183526000926020830190803683375050506005850154909150611d4e908b35906124a2565b81600081518110611d6157611d61612fa2565b6020908102919091010152611d8f8b6001602002016020810190611d859190612fb8565b60208c013561268a565b81600181518110611da257611da2612fa2565b60209081029190910101526007840180547fffffffffffffffffffffffffffffffffffffffffffffff00ffffffffffffffff1668020000000000000000179055611def8d60008b84612112565b8c67ffffffffffffffff167f24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db84604051611e2b91815260200190565b60405180910390a2505050505b60006007820154600160401b900460ff166002811115611e5a57611e5a612c3a565b03611e655750611eff565b6040805180820190915281546001600160a01b03168152600182015460208201526004820154611e9590426132a0565b81602001818151611ea691906132a0565b90525060028201805483546001600160a01b0380831673ffffffffffffffffffffffffffffffffffffffff1992831617865560038601805460018801558551929093169116179091556020909101519055426004909101555b5050505050505050565b6000838383604051602001611f2093929190613341565b6040516020818303038152906040528051906020012090505b9392505050565b80518051602091820151828401518051908401516040517f476c6f62616c2073746174653a0000000000000000000000000000000000000095810195909552602d850193909352604d8401919091527fffffffffffffffff00000000000000000000000000000000000000000000000060c091821b8116606d85015291901b166075820152600090607d015b604051602081830303815290604052805190602001209050919050565b60006001836002811115611fff57611fff612c3a565b03612055576040517f426c6f636b2073746174653a00000000000000000000000000000000000000006020820152602c8101839052604c015b6040516020818303038152906040528051906020012090506109fe565b600283600281111561206957612069612c3a565b036120a6576040517f426c6f636b2073746174652c206572726f7265643a0000000000000000000000602082015260358101839052605501612038565b60405162461bcd60e51b815260206004820152601060248201527f4241445f424c4f434b5f535441545553000000000000000000000000000000006044820152606401610395565b6020810151600090815b602002015192915050565b602081015160009060016120f8565b60018210156121235761212361314f565b6002815110156121355761213561314f565b6000612142848484611f09565b67ffffffffffffffff8616600081815260016020526040908190206006018390555191925082917f86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d689061219a90889088908890613383565b60405180910390a35050505050565b67ffffffffffffffff8216600081815260016020819052604080832060028082018054835473ffffffffffffffffffffffffffffffffffffffff19808216865596850188905595811690915560038301869055600480840187905560058401879055600684019690965560078301805468ffffffffffffffffff19169055905492517f0357aa49000000000000000000000000000000000000000000000000000000008152948501959095526001600160a01b03948516602485018190529285166044850181905290949293909290911690630357aa4990606401600060405180830381600087803b15801561229e57600080fd5b505af11580156122b2573d6000803e3d6000fd5b505050508467ffffffffffffffff167ffdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40856040516122f09190613217565b60405180910390a25050505050565b600181015460009061231083612727565b1192915050565b60008080600161232a6040860186613231565b6123359291506132a0565b90506123458160208601356133ee565b9150612355606085013583613402565b6123609085356132b3565b925060026123716040860186613231565b61237c9291506132a0565b8460600135036123a157612394816020860135613419565b61239e90836132b3565b91505b50915091565b816123b56040850185613231565b85606001358181106123c9576123c9612fa2565b905060200201351461241d5760405162461bcd60e51b815260206004820152600b60248201527f57524f4e475f53544152540000000000000000000000000000000000000000006044820152606401610395565b8061242b6040850185613231565b61243a606087013560016132b3565b81811061244957612449612fa2565b905060200201350361249d5760405162461bcd60e51b815260206004820152600860248201527f53414d455f454e440000000000000000000000000000000000000000000000006044820152606401610395565b505050565b60408051600380825260808201909252600091829190816020015b60408051808201909152600080825260208201528152602001906001900390816124bd57505060408051808201825260008082526020918201819052825180840190935260048352908201529091508160008151811061251f5761251f612fa2565b60200260200101819052506125626000604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b8160018151811061257557612575612fa2565b60200260200101819052506125b86000604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b816002815181106125cb576125cb612fa2565b602090810291909101810191909152604080518083018252838152815180830190925280825260009282019290925261261b60408051606080820183529181019182529081526000602082015290565b604080518082018252606080825260006020808401829052845161012081018652828152908101879052938401859052908301829052608083018a905260a0830181905260c0830181905260e083015261010082018890529061267d81612739565b9998505050505050505050565b600060018360028111156126a0576126a0612c3a565b036126dd576040517f4d616368696e652066696e69736865643a000000000000000000000000000000602082015260318101839052605101612038565b60028360028111156126f1576126f1612c3a565b036120a6576040517f4d616368696e65206572726f7265643a000000000000000000000000000000006020820152603001612038565b60008160040154426109fe91906132a0565b6000808251600281111561274f5761274f612c3a565b03612829576127618260200151612926565b61276e8360400151612926565b61277b84606001516129bc565b608085015160a086015160c087015160e0808901516101008a01516040517f4d616368696e652072756e6e696e673a00000000000000000000000000000000602082015260308101999099526050890197909752607088019590955260908701939093527fffffffff0000000000000000000000000000000000000000000000000000000091831b821660b0870152821b811660b486015291901b1660b883015260bc82015260dc01611fcc565b60018251600281111561283e5761283e612c3a565b036128815760808201516040517f4d616368696e652066696e69736865643a00000000000000000000000000000060208201526031810191909152605101611fcc565b60028251600281111561289657612896612c3a565b036128d95760808201516040517f4d616368696e65206572726f7265643a0000000000000000000000000000000060208201526030810191909152605001611fcc565b60405162461bcd60e51b815260206004820152600f60248201527f4241445f4d4143485f53544154555300000000000000000000000000000000006044820152606401610395565b919050565b60208101518151515160005b818110156129b557835161294f9061294a9083612a60565b612a98565b6040517f56616c756520737461636b3a00000000000000000000000000000000000000006020820152602c810191909152604c8101849052606c0160405160208183030381529060405280519060200120925080806129ad9061342d565b915050612932565b5050919050565b602081015160005b825151811015612a5a576129f4836000015182815181106129e7576129e7612fa2565b6020026020010151612ab5565b6040517f537461636b206672616d6520737461636b3a000000000000000000000000000060208201526032810191909152605281018390526072016040516020818303038152906040528051906020012091508080612a529061342d565b9150506129c4565b50919050565b60408051808201909152600080825260208201528251805183908110612a8857612a88612fa2565b6020026020010151905092915050565b600081600001518260200151604051602001611fcc929190613465565b6000612ac48260000151612a98565b602080840151604080860151606087015191517f537461636b206672616d653a000000000000000000000000000000000000000094810194909452602c840194909452604c8301919091527fffffffff0000000000000000000000000000000000000000000000000000000060e093841b8116606c840152921b9091166070820152607401611fcc565b80604081018310156109fe57600080fd5b803567ffffffffffffffff8116811461292157600080fd5b6001600160a01b038116811461072757600080fd5b600080600080600080600080610200898b031215612ba957600080fd5b88359750612bba8a60208b01612b4e565b965061016089018a811115612bce57600080fd5b60608a019650612bdd81612b5f565b955050610180890135612bef81612b77565b93506101a0890135612c0081612b77565b979a96995094979396929592945050506101c0820135916101e0013590565b600060208284031215612c3157600080fd5b611f3982612b5f565b634e487b7160e01b600052602160045260246000fd5b60038110612c6057612c60612c3a565b9052565b815180516001600160a01b0316825260209081015190820152610120810160208381015180516001600160a01b031660408501529081015160608401525060408301516080830152606083015160a0830152608083015160c083015267ffffffffffffffff60a08401511660e083015260c0830151612ce7610100840182612c50565b5092915050565b600060208284031215612d0057600080fd5b5035919050565b87516001600160a01b0316815260208089015190820152610120810187516001600160a01b03166040830152602088015160608301528660808301528560a08301528460c083015267ffffffffffffffff841660e0830152610640610100830184612c50565b600060808284031215612a5a57600080fd5b60008060008060608587031215612d9557600080fd5b612d9e85612b5f565b9350602085013567ffffffffffffffff80821115612dbb57600080fd5b612dc788838901612d6d565b94506040870135915080821115612ddd57600080fd5b818701915087601f830112612df157600080fd5b813581811115612e0057600080fd5b8860208260051b8501011115612e1557600080fd5b95989497505060200194505050565b60008060008060608587031215612e3a57600080fd5b612e4385612b5f565b9350602085013567ffffffffffffffff80821115612e6057600080fd5b612e6c88838901612d6d565b94506040870135915080821115612e8257600080fd5b818701915087601f830112612e9657600080fd5b813581811115612ea557600080fd5b886020828501011115612e1557600080fd5b60008060008060808587031215612ecd57600080fd5b8435612ed881612b77565b93506020850135612ee881612b77565b92506040850135612ef881612b77565b91506060850135612f0881612b77565b939692955090935050565b600080600080600060e08688031215612f2b57600080fd5b612f3486612b5f565b9450602086013567ffffffffffffffff811115612f5057600080fd5b612f5c88828901612d6d565b945050612f6c8760408801612b4e565b9250612f7b8760808801612b4e565b9497939650919460c0013592915050565b634e487b7160e01b600052604160045260246000fd5b634e487b7160e01b600052603260045260246000fd5b600060208284031215612fca57600080fd5b813560038110611f3957600080fd5b6040805190810167ffffffffffffffff81118282101715612ffc57612ffc612f8c565b60405290565b600082601f83011261301357600080fd5b6040516040810181811067ffffffffffffffff8211171561303657613036612f8c565b806040525080604084018581111561304d57600080fd5b845b8181101561306e5761306081612b5f565b83526020928301920161304f565b509195945050505050565b60006080828403121561308b57600080fd5b6040516040810181811067ffffffffffffffff821117156130ae576130ae612f8c565b604052601f830184136130c057600080fd5b6130c8612fd9565b8060408501868111156130da57600080fd5b855b818110156130f45780358452602093840193016130dc565b508184526131028782613002565b6020850152509195945050505050565b634e487b7160e01b600052601160045260246000fd5b600067ffffffffffffffff80831681810361314557613145613112565b6001019392505050565b634e487b7160e01b600052600160045260246000fd5b6040818337604082016040820160005b60028110156131a65767ffffffffffffffff61319083612b5f565b1683526020928301929190910190600101613175565b5050505050565b61010081016131bc8285613165565b611f396080830184613165565b600060208083528351808285015260005b818110156131f6578581018301518582016040015282016131da565b506000604082860101526040601f19601f8301168501019250505092915050565b602081016004831061322b5761322b612c3a565b91905290565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe184360301811261326657600080fd5b83018035915067ffffffffffffffff82111561328157600080fd5b6020019150600581901b360382131561329957600080fd5b9250929050565b818103818111156109fe576109fe613112565b808201808211156109fe576109fe613112565b855181526001600160a01b0360208701511660208201526040860151604082015284606082015283608082015260c060a08201528160c0820152818360e0830137600081830160e090810191909152601f909201601f19160101949350505050565b60006020828403121561333a57600080fd5b5051919050565b83815260006020848184015260408301845182860160005b8281101561337557815184529284019290840190600101613359565b509198975050505050505050565b6000606082018583526020858185015260606040850152818551808452608086019150828701935060005b818110156133ca578451835293830193918301916001016133ae565b509098975050505050505050565b634e487b7160e01b600052601260045260246000fd5b6000826133fd576133fd6133d8565b500490565b80820281158282048414176109fe576109fe613112565b600082613428576134286133d8565b500690565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361345e5761345e613112565b5060010190565b7f56616c75653a0000000000000000000000000000000000000000000000000000815260006007841061349a5761349a612c3a565b5060f89290921b600683015260078201526027019056fea26469706673582212201a4e28db100105638dcba0617d5083bde1e8a01f2d386d73ef7ba54d08c1841d64736f6c63430008110033",
}

// SingleExecutionChallengeABI is the input ABI used to generate the binding from.
// Deprecated: Use SingleExecutionChallengeMetaData.ABI instead.
var SingleExecutionChallengeABI = SingleExecutionChallengeMetaData.ABI

// SingleExecutionChallengeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SingleExecutionChallengeMetaData.Bin instead.
var SingleExecutionChallengeBin = SingleExecutionChallengeMetaData.Bin

// DeploySingleExecutionChallenge deploys a new Ethereum contract, binding an instance of SingleExecutionChallenge to it.
func DeploySingleExecutionChallenge(auth *bind.TransactOpts, backend bind.ContractBackend, osp_ common.Address, resultReceiver_ common.Address, maxInboxMessagesRead_ uint64, startAndEndHashes [2][32]byte, numSteps_ *big.Int, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (common.Address, *types.Transaction, *SingleExecutionChallenge, error) {
	parsed, err := SingleExecutionChallengeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SingleExecutionChallengeBin), backend, osp_, resultReceiver_, maxInboxMessagesRead_, startAndEndHashes, numSteps_, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SingleExecutionChallenge{SingleExecutionChallengeCaller: SingleExecutionChallengeCaller{contract: contract}, SingleExecutionChallengeTransactor: SingleExecutionChallengeTransactor{contract: contract}, SingleExecutionChallengeFilterer: SingleExecutionChallengeFilterer{contract: contract}}, nil
}

// SingleExecutionChallenge is an auto generated Go binding around an Ethereum contract.
type SingleExecutionChallenge struct {
	SingleExecutionChallengeCaller     // Read-only binding to the contract
	SingleExecutionChallengeTransactor // Write-only binding to the contract
	SingleExecutionChallengeFilterer   // Log filterer for contract events
}

// SingleExecutionChallengeCaller is an auto generated read-only Go binding around an Ethereum contract.
type SingleExecutionChallengeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SingleExecutionChallengeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SingleExecutionChallengeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SingleExecutionChallengeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SingleExecutionChallengeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SingleExecutionChallengeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SingleExecutionChallengeSession struct {
	Contract     *SingleExecutionChallenge // Generic contract binding to set the session for
	CallOpts     bind.CallOpts             // Call options to use throughout this session
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// SingleExecutionChallengeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SingleExecutionChallengeCallerSession struct {
	Contract *SingleExecutionChallengeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                   // Call options to use throughout this session
}

// SingleExecutionChallengeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SingleExecutionChallengeTransactorSession struct {
	Contract     *SingleExecutionChallengeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                   // Transaction auth options to use throughout this session
}

// SingleExecutionChallengeRaw is an auto generated low-level Go binding around an Ethereum contract.
type SingleExecutionChallengeRaw struct {
	Contract *SingleExecutionChallenge // Generic contract binding to access the raw methods on
}

// SingleExecutionChallengeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SingleExecutionChallengeCallerRaw struct {
	Contract *SingleExecutionChallengeCaller // Generic read-only contract binding to access the raw methods on
}

// SingleExecutionChallengeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SingleExecutionChallengeTransactorRaw struct {
	Contract *SingleExecutionChallengeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSingleExecutionChallenge creates a new instance of SingleExecutionChallenge, bound to a specific deployed contract.
func NewSingleExecutionChallenge(address common.Address, backend bind.ContractBackend) (*SingleExecutionChallenge, error) {
	contract, err := bindSingleExecutionChallenge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SingleExecutionChallenge{SingleExecutionChallengeCaller: SingleExecutionChallengeCaller{contract: contract}, SingleExecutionChallengeTransactor: SingleExecutionChallengeTransactor{contract: contract}, SingleExecutionChallengeFilterer: SingleExecutionChallengeFilterer{contract: contract}}, nil
}

// NewSingleExecutionChallengeCaller creates a new read-only instance of SingleExecutionChallenge, bound to a specific deployed contract.
func NewSingleExecutionChallengeCaller(address common.Address, caller bind.ContractCaller) (*SingleExecutionChallengeCaller, error) {
	contract, err := bindSingleExecutionChallenge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SingleExecutionChallengeCaller{contract: contract}, nil
}

// NewSingleExecutionChallengeTransactor creates a new write-only instance of SingleExecutionChallenge, bound to a specific deployed contract.
func NewSingleExecutionChallengeTransactor(address common.Address, transactor bind.ContractTransactor) (*SingleExecutionChallengeTransactor, error) {
	contract, err := bindSingleExecutionChallenge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SingleExecutionChallengeTransactor{contract: contract}, nil
}

// NewSingleExecutionChallengeFilterer creates a new log filterer instance of SingleExecutionChallenge, bound to a specific deployed contract.
func NewSingleExecutionChallengeFilterer(address common.Address, filterer bind.ContractFilterer) (*SingleExecutionChallengeFilterer, error) {
	contract, err := bindSingleExecutionChallenge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SingleExecutionChallengeFilterer{contract: contract}, nil
}

// bindSingleExecutionChallenge binds a generic wrapper to an already deployed contract.
func bindSingleExecutionChallenge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SingleExecutionChallengeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SingleExecutionChallenge *SingleExecutionChallengeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SingleExecutionChallenge.Contract.SingleExecutionChallengeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SingleExecutionChallenge *SingleExecutionChallengeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.SingleExecutionChallengeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SingleExecutionChallenge *SingleExecutionChallengeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.SingleExecutionChallengeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SingleExecutionChallenge *SingleExecutionChallengeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SingleExecutionChallenge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.contract.Transact(opts, method, params...)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SingleExecutionChallenge.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) Bridge() (common.Address, error) {
	return _SingleExecutionChallenge.Contract.Bridge(&_SingleExecutionChallenge.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeCallerSession) Bridge() (common.Address, error) {
	return _SingleExecutionChallenge.Contract.Bridge(&_SingleExecutionChallenge.CallOpts)
}

// ChallengeInfo is a free data retrieval call binding the contract method 0x7fd07a9c.
//
// Solidity: function challengeInfo(uint64 challengeIndex) view returns(((address,uint256),(address,uint256),uint256,bytes32,bytes32,uint64,uint8))
func (_SingleExecutionChallenge *SingleExecutionChallengeCaller) ChallengeInfo(opts *bind.CallOpts, challengeIndex uint64) (OldChallengeLibChallenge, error) {
	var out []interface{}
	err := _SingleExecutionChallenge.contract.Call(opts, &out, "challengeInfo", challengeIndex)

	if err != nil {
		return *new(OldChallengeLibChallenge), err
	}

	out0 := *abi.ConvertType(out[0], new(OldChallengeLibChallenge)).(*OldChallengeLibChallenge)

	return out0, err

}

// ChallengeInfo is a free data retrieval call binding the contract method 0x7fd07a9c.
//
// Solidity: function challengeInfo(uint64 challengeIndex) view returns(((address,uint256),(address,uint256),uint256,bytes32,bytes32,uint64,uint8))
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) ChallengeInfo(challengeIndex uint64) (OldChallengeLibChallenge, error) {
	return _SingleExecutionChallenge.Contract.ChallengeInfo(&_SingleExecutionChallenge.CallOpts, challengeIndex)
}

// ChallengeInfo is a free data retrieval call binding the contract method 0x7fd07a9c.
//
// Solidity: function challengeInfo(uint64 challengeIndex) view returns(((address,uint256),(address,uint256),uint256,bytes32,bytes32,uint64,uint8))
func (_SingleExecutionChallenge *SingleExecutionChallengeCallerSession) ChallengeInfo(challengeIndex uint64) (OldChallengeLibChallenge, error) {
	return _SingleExecutionChallenge.Contract.ChallengeInfo(&_SingleExecutionChallenge.CallOpts, challengeIndex)
}

// Challenges is a free data retrieval call binding the contract method 0x8f1d3776.
//
// Solidity: function challenges(uint256 ) view returns((address,uint256) current, (address,uint256) next, uint256 lastMoveTimestamp, bytes32 wasmModuleRoot, bytes32 challengeStateHash, uint64 maxInboxMessages, uint8 mode)
func (_SingleExecutionChallenge *SingleExecutionChallengeCaller) Challenges(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Current            OldChallengeLibParticipant
	Next               OldChallengeLibParticipant
	LastMoveTimestamp  *big.Int
	WasmModuleRoot     [32]byte
	ChallengeStateHash [32]byte
	MaxInboxMessages   uint64
	Mode               uint8
}, error) {
	var out []interface{}
	err := _SingleExecutionChallenge.contract.Call(opts, &out, "challenges", arg0)

	outstruct := new(struct {
		Current            OldChallengeLibParticipant
		Next               OldChallengeLibParticipant
		LastMoveTimestamp  *big.Int
		WasmModuleRoot     [32]byte
		ChallengeStateHash [32]byte
		MaxInboxMessages   uint64
		Mode               uint8
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Current = *abi.ConvertType(out[0], new(OldChallengeLibParticipant)).(*OldChallengeLibParticipant)
	outstruct.Next = *abi.ConvertType(out[1], new(OldChallengeLibParticipant)).(*OldChallengeLibParticipant)
	outstruct.LastMoveTimestamp = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.WasmModuleRoot = *abi.ConvertType(out[3], new([32]byte)).(*[32]byte)
	outstruct.ChallengeStateHash = *abi.ConvertType(out[4], new([32]byte)).(*[32]byte)
	outstruct.MaxInboxMessages = *abi.ConvertType(out[5], new(uint64)).(*uint64)
	outstruct.Mode = *abi.ConvertType(out[6], new(uint8)).(*uint8)

	return *outstruct, err

}

// Challenges is a free data retrieval call binding the contract method 0x8f1d3776.
//
// Solidity: function challenges(uint256 ) view returns((address,uint256) current, (address,uint256) next, uint256 lastMoveTimestamp, bytes32 wasmModuleRoot, bytes32 challengeStateHash, uint64 maxInboxMessages, uint8 mode)
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) Challenges(arg0 *big.Int) (struct {
	Current            OldChallengeLibParticipant
	Next               OldChallengeLibParticipant
	LastMoveTimestamp  *big.Int
	WasmModuleRoot     [32]byte
	ChallengeStateHash [32]byte
	MaxInboxMessages   uint64
	Mode               uint8
}, error) {
	return _SingleExecutionChallenge.Contract.Challenges(&_SingleExecutionChallenge.CallOpts, arg0)
}

// Challenges is a free data retrieval call binding the contract method 0x8f1d3776.
//
// Solidity: function challenges(uint256 ) view returns((address,uint256) current, (address,uint256) next, uint256 lastMoveTimestamp, bytes32 wasmModuleRoot, bytes32 challengeStateHash, uint64 maxInboxMessages, uint8 mode)
func (_SingleExecutionChallenge *SingleExecutionChallengeCallerSession) Challenges(arg0 *big.Int) (struct {
	Current            OldChallengeLibParticipant
	Next               OldChallengeLibParticipant
	LastMoveTimestamp  *big.Int
	WasmModuleRoot     [32]byte
	ChallengeStateHash [32]byte
	MaxInboxMessages   uint64
	Mode               uint8
}, error) {
	return _SingleExecutionChallenge.Contract.Challenges(&_SingleExecutionChallenge.CallOpts, arg0)
}

// CurrentResponder is a free data retrieval call binding the contract method 0x23a9ef23.
//
// Solidity: function currentResponder(uint64 challengeIndex) view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeCaller) CurrentResponder(opts *bind.CallOpts, challengeIndex uint64) (common.Address, error) {
	var out []interface{}
	err := _SingleExecutionChallenge.contract.Call(opts, &out, "currentResponder", challengeIndex)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CurrentResponder is a free data retrieval call binding the contract method 0x23a9ef23.
//
// Solidity: function currentResponder(uint64 challengeIndex) view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) CurrentResponder(challengeIndex uint64) (common.Address, error) {
	return _SingleExecutionChallenge.Contract.CurrentResponder(&_SingleExecutionChallenge.CallOpts, challengeIndex)
}

// CurrentResponder is a free data retrieval call binding the contract method 0x23a9ef23.
//
// Solidity: function currentResponder(uint64 challengeIndex) view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeCallerSession) CurrentResponder(challengeIndex uint64) (common.Address, error) {
	return _SingleExecutionChallenge.Contract.CurrentResponder(&_SingleExecutionChallenge.CallOpts, challengeIndex)
}

// IsTimedOut is a free data retrieval call binding the contract method 0x9ede42b9.
//
// Solidity: function isTimedOut(uint64 challengeIndex) view returns(bool)
func (_SingleExecutionChallenge *SingleExecutionChallengeCaller) IsTimedOut(opts *bind.CallOpts, challengeIndex uint64) (bool, error) {
	var out []interface{}
	err := _SingleExecutionChallenge.contract.Call(opts, &out, "isTimedOut", challengeIndex)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsTimedOut is a free data retrieval call binding the contract method 0x9ede42b9.
//
// Solidity: function isTimedOut(uint64 challengeIndex) view returns(bool)
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) IsTimedOut(challengeIndex uint64) (bool, error) {
	return _SingleExecutionChallenge.Contract.IsTimedOut(&_SingleExecutionChallenge.CallOpts, challengeIndex)
}

// IsTimedOut is a free data retrieval call binding the contract method 0x9ede42b9.
//
// Solidity: function isTimedOut(uint64 challengeIndex) view returns(bool)
func (_SingleExecutionChallenge *SingleExecutionChallengeCallerSession) IsTimedOut(challengeIndex uint64) (bool, error) {
	return _SingleExecutionChallenge.Contract.IsTimedOut(&_SingleExecutionChallenge.CallOpts, challengeIndex)
}

// Osp is a free data retrieval call binding the contract method 0xf26a62c6.
//
// Solidity: function osp() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeCaller) Osp(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SingleExecutionChallenge.contract.Call(opts, &out, "osp")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Osp is a free data retrieval call binding the contract method 0xf26a62c6.
//
// Solidity: function osp() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) Osp() (common.Address, error) {
	return _SingleExecutionChallenge.Contract.Osp(&_SingleExecutionChallenge.CallOpts)
}

// Osp is a free data retrieval call binding the contract method 0xf26a62c6.
//
// Solidity: function osp() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeCallerSession) Osp() (common.Address, error) {
	return _SingleExecutionChallenge.Contract.Osp(&_SingleExecutionChallenge.CallOpts)
}

// ResultReceiver is a free data retrieval call binding the contract method 0x3504f1d7.
//
// Solidity: function resultReceiver() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeCaller) ResultReceiver(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SingleExecutionChallenge.contract.Call(opts, &out, "resultReceiver")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ResultReceiver is a free data retrieval call binding the contract method 0x3504f1d7.
//
// Solidity: function resultReceiver() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) ResultReceiver() (common.Address, error) {
	return _SingleExecutionChallenge.Contract.ResultReceiver(&_SingleExecutionChallenge.CallOpts)
}

// ResultReceiver is a free data retrieval call binding the contract method 0x3504f1d7.
//
// Solidity: function resultReceiver() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeCallerSession) ResultReceiver() (common.Address, error) {
	return _SingleExecutionChallenge.Contract.ResultReceiver(&_SingleExecutionChallenge.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeCaller) SequencerInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SingleExecutionChallenge.contract.Call(opts, &out, "sequencerInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) SequencerInbox() (common.Address, error) {
	return _SingleExecutionChallenge.Contract.SequencerInbox(&_SingleExecutionChallenge.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_SingleExecutionChallenge *SingleExecutionChallengeCallerSession) SequencerInbox() (common.Address, error) {
	return _SingleExecutionChallenge.Contract.SequencerInbox(&_SingleExecutionChallenge.CallOpts)
}

// TotalChallengesCreated is a free data retrieval call binding the contract method 0x5ef489e6.
//
// Solidity: function totalChallengesCreated() view returns(uint64)
func (_SingleExecutionChallenge *SingleExecutionChallengeCaller) TotalChallengesCreated(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _SingleExecutionChallenge.contract.Call(opts, &out, "totalChallengesCreated")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// TotalChallengesCreated is a free data retrieval call binding the contract method 0x5ef489e6.
//
// Solidity: function totalChallengesCreated() view returns(uint64)
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) TotalChallengesCreated() (uint64, error) {
	return _SingleExecutionChallenge.Contract.TotalChallengesCreated(&_SingleExecutionChallenge.CallOpts)
}

// TotalChallengesCreated is a free data retrieval call binding the contract method 0x5ef489e6.
//
// Solidity: function totalChallengesCreated() view returns(uint64)
func (_SingleExecutionChallenge *SingleExecutionChallengeCallerSession) TotalChallengesCreated() (uint64, error) {
	return _SingleExecutionChallenge.Contract.TotalChallengesCreated(&_SingleExecutionChallenge.CallOpts)
}

// BisectExecution is a paid mutator transaction binding the contract method 0xa521b032.
//
// Solidity: function bisectExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes32[] newSegments) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactor) BisectExecution(opts *bind.TransactOpts, challengeIndex uint64, selection OldChallengeLibSegmentSelection, newSegments [][32]byte) (*types.Transaction, error) {
	return _SingleExecutionChallenge.contract.Transact(opts, "bisectExecution", challengeIndex, selection, newSegments)
}

// BisectExecution is a paid mutator transaction binding the contract method 0xa521b032.
//
// Solidity: function bisectExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes32[] newSegments) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) BisectExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, newSegments [][32]byte) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.BisectExecution(&_SingleExecutionChallenge.TransactOpts, challengeIndex, selection, newSegments)
}

// BisectExecution is a paid mutator transaction binding the contract method 0xa521b032.
//
// Solidity: function bisectExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes32[] newSegments) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactorSession) BisectExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, newSegments [][32]byte) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.BisectExecution(&_SingleExecutionChallenge.TransactOpts, challengeIndex, selection, newSegments)
}

// ChallengeExecution is a paid mutator transaction binding the contract method 0xfb7be0a1.
//
// Solidity: function challengeExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, uint8[2] machineStatuses, bytes32[2] globalStateHashes, uint256 numSteps) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactor) ChallengeExecution(opts *bind.TransactOpts, challengeIndex uint64, selection OldChallengeLibSegmentSelection, machineStatuses [2]uint8, globalStateHashes [2][32]byte, numSteps *big.Int) (*types.Transaction, error) {
	return _SingleExecutionChallenge.contract.Transact(opts, "challengeExecution", challengeIndex, selection, machineStatuses, globalStateHashes, numSteps)
}

// ChallengeExecution is a paid mutator transaction binding the contract method 0xfb7be0a1.
//
// Solidity: function challengeExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, uint8[2] machineStatuses, bytes32[2] globalStateHashes, uint256 numSteps) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) ChallengeExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, machineStatuses [2]uint8, globalStateHashes [2][32]byte, numSteps *big.Int) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.ChallengeExecution(&_SingleExecutionChallenge.TransactOpts, challengeIndex, selection, machineStatuses, globalStateHashes, numSteps)
}

// ChallengeExecution is a paid mutator transaction binding the contract method 0xfb7be0a1.
//
// Solidity: function challengeExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, uint8[2] machineStatuses, bytes32[2] globalStateHashes, uint256 numSteps) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactorSession) ChallengeExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, machineStatuses [2]uint8, globalStateHashes [2][32]byte, numSteps *big.Int) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.ChallengeExecution(&_SingleExecutionChallenge.TransactOpts, challengeIndex, selection, machineStatuses, globalStateHashes, numSteps)
}

// ClearChallenge is a paid mutator transaction binding the contract method 0x56e9df97.
//
// Solidity: function clearChallenge(uint64 challengeIndex) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactor) ClearChallenge(opts *bind.TransactOpts, challengeIndex uint64) (*types.Transaction, error) {
	return _SingleExecutionChallenge.contract.Transact(opts, "clearChallenge", challengeIndex)
}

// ClearChallenge is a paid mutator transaction binding the contract method 0x56e9df97.
//
// Solidity: function clearChallenge(uint64 challengeIndex) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) ClearChallenge(challengeIndex uint64) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.ClearChallenge(&_SingleExecutionChallenge.TransactOpts, challengeIndex)
}

// ClearChallenge is a paid mutator transaction binding the contract method 0x56e9df97.
//
// Solidity: function clearChallenge(uint64 challengeIndex) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactorSession) ClearChallenge(challengeIndex uint64) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.ClearChallenge(&_SingleExecutionChallenge.TransactOpts, challengeIndex)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactor) CreateChallenge(opts *bind.TransactOpts, wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _SingleExecutionChallenge.contract.Transact(opts, "createChallenge", wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) CreateChallenge(wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.CreateChallenge(&_SingleExecutionChallenge.TransactOpts, wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0x14eab5e7.
//
// Solidity: function createChallenge(bytes32 wasmModuleRoot_, uint8[2] startAndEndMachineStatuses_, (bytes32[2],uint64[2])[2] startAndEndGlobalStates_, uint64 numBlocks, address asserter_, address challenger_, uint256 asserterTimeLeft_, uint256 challengerTimeLeft_) returns(uint64)
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactorSession) CreateChallenge(wasmModuleRoot_ [32]byte, startAndEndMachineStatuses_ [2]uint8, startAndEndGlobalStates_ [2]GlobalState, numBlocks uint64, asserter_ common.Address, challenger_ common.Address, asserterTimeLeft_ *big.Int, challengerTimeLeft_ *big.Int) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.CreateChallenge(&_SingleExecutionChallenge.TransactOpts, wasmModuleRoot_, startAndEndMachineStatuses_, startAndEndGlobalStates_, numBlocks, asserter_, challenger_, asserterTimeLeft_, challengerTimeLeft_)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address resultReceiver_, address sequencerInbox_, address bridge_, address osp_) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactor) Initialize(opts *bind.TransactOpts, resultReceiver_ common.Address, sequencerInbox_ common.Address, bridge_ common.Address, osp_ common.Address) (*types.Transaction, error) {
	return _SingleExecutionChallenge.contract.Transact(opts, "initialize", resultReceiver_, sequencerInbox_, bridge_, osp_)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address resultReceiver_, address sequencerInbox_, address bridge_, address osp_) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) Initialize(resultReceiver_ common.Address, sequencerInbox_ common.Address, bridge_ common.Address, osp_ common.Address) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.Initialize(&_SingleExecutionChallenge.TransactOpts, resultReceiver_, sequencerInbox_, bridge_, osp_)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address resultReceiver_, address sequencerInbox_, address bridge_, address osp_) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactorSession) Initialize(resultReceiver_ common.Address, sequencerInbox_ common.Address, bridge_ common.Address, osp_ common.Address) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.Initialize(&_SingleExecutionChallenge.TransactOpts, resultReceiver_, sequencerInbox_, bridge_, osp_)
}

// OneStepProveExecution is a paid mutator transaction binding the contract method 0xd248d124.
//
// Solidity: function oneStepProveExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes proof) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactor) OneStepProveExecution(opts *bind.TransactOpts, challengeIndex uint64, selection OldChallengeLibSegmentSelection, proof []byte) (*types.Transaction, error) {
	return _SingleExecutionChallenge.contract.Transact(opts, "oneStepProveExecution", challengeIndex, selection, proof)
}

// OneStepProveExecution is a paid mutator transaction binding the contract method 0xd248d124.
//
// Solidity: function oneStepProveExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes proof) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) OneStepProveExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, proof []byte) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.OneStepProveExecution(&_SingleExecutionChallenge.TransactOpts, challengeIndex, selection, proof)
}

// OneStepProveExecution is a paid mutator transaction binding the contract method 0xd248d124.
//
// Solidity: function oneStepProveExecution(uint64 challengeIndex, (uint256,uint256,bytes32[],uint256) selection, bytes proof) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactorSession) OneStepProveExecution(challengeIndex uint64, selection OldChallengeLibSegmentSelection, proof []byte) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.OneStepProveExecution(&_SingleExecutionChallenge.TransactOpts, challengeIndex, selection, proof)
}

// Timeout is a paid mutator transaction binding the contract method 0x1b45c86a.
//
// Solidity: function timeout(uint64 challengeIndex) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactor) Timeout(opts *bind.TransactOpts, challengeIndex uint64) (*types.Transaction, error) {
	return _SingleExecutionChallenge.contract.Transact(opts, "timeout", challengeIndex)
}

// Timeout is a paid mutator transaction binding the contract method 0x1b45c86a.
//
// Solidity: function timeout(uint64 challengeIndex) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeSession) Timeout(challengeIndex uint64) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.Timeout(&_SingleExecutionChallenge.TransactOpts, challengeIndex)
}

// Timeout is a paid mutator transaction binding the contract method 0x1b45c86a.
//
// Solidity: function timeout(uint64 challengeIndex) returns()
func (_SingleExecutionChallenge *SingleExecutionChallengeTransactorSession) Timeout(challengeIndex uint64) (*types.Transaction, error) {
	return _SingleExecutionChallenge.Contract.Timeout(&_SingleExecutionChallenge.TransactOpts, challengeIndex)
}

// SingleExecutionChallengeBisectedIterator is returned from FilterBisected and is used to iterate over the raw logs and unpacked data for Bisected events raised by the SingleExecutionChallenge contract.
type SingleExecutionChallengeBisectedIterator struct {
	Event *SingleExecutionChallengeBisected // Event containing the contract specifics and raw log

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
func (it *SingleExecutionChallengeBisectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingleExecutionChallengeBisected)
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
		it.Event = new(SingleExecutionChallengeBisected)
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
func (it *SingleExecutionChallengeBisectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingleExecutionChallengeBisectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingleExecutionChallengeBisected represents a Bisected event raised by the SingleExecutionChallenge contract.
type SingleExecutionChallengeBisected struct {
	ChallengeIndex          uint64
	ChallengeRoot           [32]byte
	ChallengedSegmentStart  *big.Int
	ChallengedSegmentLength *big.Int
	ChainHashes             [][32]byte
	Raw                     types.Log // Blockchain specific contextual infos
}

// FilterBisected is a free log retrieval operation binding the contract event 0x86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d68.
//
// Solidity: event Bisected(uint64 indexed challengeIndex, bytes32 indexed challengeRoot, uint256 challengedSegmentStart, uint256 challengedSegmentLength, bytes32[] chainHashes)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) FilterBisected(opts *bind.FilterOpts, challengeIndex []uint64, challengeRoot [][32]byte) (*SingleExecutionChallengeBisectedIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}
	var challengeRootRule []interface{}
	for _, challengeRootItem := range challengeRoot {
		challengeRootRule = append(challengeRootRule, challengeRootItem)
	}

	logs, sub, err := _SingleExecutionChallenge.contract.FilterLogs(opts, "Bisected", challengeIndexRule, challengeRootRule)
	if err != nil {
		return nil, err
	}
	return &SingleExecutionChallengeBisectedIterator{contract: _SingleExecutionChallenge.contract, event: "Bisected", logs: logs, sub: sub}, nil
}

// WatchBisected is a free log subscription operation binding the contract event 0x86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d68.
//
// Solidity: event Bisected(uint64 indexed challengeIndex, bytes32 indexed challengeRoot, uint256 challengedSegmentStart, uint256 challengedSegmentLength, bytes32[] chainHashes)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) WatchBisected(opts *bind.WatchOpts, sink chan<- *SingleExecutionChallengeBisected, challengeIndex []uint64, challengeRoot [][32]byte) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}
	var challengeRootRule []interface{}
	for _, challengeRootItem := range challengeRoot {
		challengeRootRule = append(challengeRootRule, challengeRootItem)
	}

	logs, sub, err := _SingleExecutionChallenge.contract.WatchLogs(opts, "Bisected", challengeIndexRule, challengeRootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingleExecutionChallengeBisected)
				if err := _SingleExecutionChallenge.contract.UnpackLog(event, "Bisected", log); err != nil {
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

// ParseBisected is a log parse operation binding the contract event 0x86b34e9455464834eca718f62d4481437603bb929d8a78ccde5d1bc79fa06d68.
//
// Solidity: event Bisected(uint64 indexed challengeIndex, bytes32 indexed challengeRoot, uint256 challengedSegmentStart, uint256 challengedSegmentLength, bytes32[] chainHashes)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) ParseBisected(log types.Log) (*SingleExecutionChallengeBisected, error) {
	event := new(SingleExecutionChallengeBisected)
	if err := _SingleExecutionChallenge.contract.UnpackLog(event, "Bisected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SingleExecutionChallengeChallengeEndedIterator is returned from FilterChallengeEnded and is used to iterate over the raw logs and unpacked data for ChallengeEnded events raised by the SingleExecutionChallenge contract.
type SingleExecutionChallengeChallengeEndedIterator struct {
	Event *SingleExecutionChallengeChallengeEnded // Event containing the contract specifics and raw log

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
func (it *SingleExecutionChallengeChallengeEndedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingleExecutionChallengeChallengeEnded)
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
		it.Event = new(SingleExecutionChallengeChallengeEnded)
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
func (it *SingleExecutionChallengeChallengeEndedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingleExecutionChallengeChallengeEndedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingleExecutionChallengeChallengeEnded represents a ChallengeEnded event raised by the SingleExecutionChallenge contract.
type SingleExecutionChallengeChallengeEnded struct {
	ChallengeIndex uint64
	Kind           uint8
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterChallengeEnded is a free log retrieval operation binding the contract event 0xfdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40.
//
// Solidity: event ChallengeEnded(uint64 indexed challengeIndex, uint8 kind)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) FilterChallengeEnded(opts *bind.FilterOpts, challengeIndex []uint64) (*SingleExecutionChallengeChallengeEndedIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _SingleExecutionChallenge.contract.FilterLogs(opts, "ChallengeEnded", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &SingleExecutionChallengeChallengeEndedIterator{contract: _SingleExecutionChallenge.contract, event: "ChallengeEnded", logs: logs, sub: sub}, nil
}

// WatchChallengeEnded is a free log subscription operation binding the contract event 0xfdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40.
//
// Solidity: event ChallengeEnded(uint64 indexed challengeIndex, uint8 kind)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) WatchChallengeEnded(opts *bind.WatchOpts, sink chan<- *SingleExecutionChallengeChallengeEnded, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _SingleExecutionChallenge.contract.WatchLogs(opts, "ChallengeEnded", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingleExecutionChallengeChallengeEnded)
				if err := _SingleExecutionChallenge.contract.UnpackLog(event, "ChallengeEnded", log); err != nil {
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

// ParseChallengeEnded is a log parse operation binding the contract event 0xfdaece6c274a4b56af16761f83fd6b1062823192630ea08e019fdf9b2d747f40.
//
// Solidity: event ChallengeEnded(uint64 indexed challengeIndex, uint8 kind)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) ParseChallengeEnded(log types.Log) (*SingleExecutionChallengeChallengeEnded, error) {
	event := new(SingleExecutionChallengeChallengeEnded)
	if err := _SingleExecutionChallenge.contract.UnpackLog(event, "ChallengeEnded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SingleExecutionChallengeExecutionChallengeBegunIterator is returned from FilterExecutionChallengeBegun and is used to iterate over the raw logs and unpacked data for ExecutionChallengeBegun events raised by the SingleExecutionChallenge contract.
type SingleExecutionChallengeExecutionChallengeBegunIterator struct {
	Event *SingleExecutionChallengeExecutionChallengeBegun // Event containing the contract specifics and raw log

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
func (it *SingleExecutionChallengeExecutionChallengeBegunIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingleExecutionChallengeExecutionChallengeBegun)
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
		it.Event = new(SingleExecutionChallengeExecutionChallengeBegun)
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
func (it *SingleExecutionChallengeExecutionChallengeBegunIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingleExecutionChallengeExecutionChallengeBegunIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingleExecutionChallengeExecutionChallengeBegun represents a ExecutionChallengeBegun event raised by the SingleExecutionChallenge contract.
type SingleExecutionChallengeExecutionChallengeBegun struct {
	ChallengeIndex uint64
	BlockSteps     *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterExecutionChallengeBegun is a free log retrieval operation binding the contract event 0x24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db.
//
// Solidity: event ExecutionChallengeBegun(uint64 indexed challengeIndex, uint256 blockSteps)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) FilterExecutionChallengeBegun(opts *bind.FilterOpts, challengeIndex []uint64) (*SingleExecutionChallengeExecutionChallengeBegunIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _SingleExecutionChallenge.contract.FilterLogs(opts, "ExecutionChallengeBegun", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &SingleExecutionChallengeExecutionChallengeBegunIterator{contract: _SingleExecutionChallenge.contract, event: "ExecutionChallengeBegun", logs: logs, sub: sub}, nil
}

// WatchExecutionChallengeBegun is a free log subscription operation binding the contract event 0x24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db.
//
// Solidity: event ExecutionChallengeBegun(uint64 indexed challengeIndex, uint256 blockSteps)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) WatchExecutionChallengeBegun(opts *bind.WatchOpts, sink chan<- *SingleExecutionChallengeExecutionChallengeBegun, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _SingleExecutionChallenge.contract.WatchLogs(opts, "ExecutionChallengeBegun", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingleExecutionChallengeExecutionChallengeBegun)
				if err := _SingleExecutionChallenge.contract.UnpackLog(event, "ExecutionChallengeBegun", log); err != nil {
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

// ParseExecutionChallengeBegun is a log parse operation binding the contract event 0x24e032e170243bbea97e140174b22dc7e54fb85925afbf52c70e001cd6af16db.
//
// Solidity: event ExecutionChallengeBegun(uint64 indexed challengeIndex, uint256 blockSteps)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) ParseExecutionChallengeBegun(log types.Log) (*SingleExecutionChallengeExecutionChallengeBegun, error) {
	event := new(SingleExecutionChallengeExecutionChallengeBegun)
	if err := _SingleExecutionChallenge.contract.UnpackLog(event, "ExecutionChallengeBegun", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SingleExecutionChallengeInitiatedChallengeIterator is returned from FilterInitiatedChallenge and is used to iterate over the raw logs and unpacked data for InitiatedChallenge events raised by the SingleExecutionChallenge contract.
type SingleExecutionChallengeInitiatedChallengeIterator struct {
	Event *SingleExecutionChallengeInitiatedChallenge // Event containing the contract specifics and raw log

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
func (it *SingleExecutionChallengeInitiatedChallengeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingleExecutionChallengeInitiatedChallenge)
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
		it.Event = new(SingleExecutionChallengeInitiatedChallenge)
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
func (it *SingleExecutionChallengeInitiatedChallengeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingleExecutionChallengeInitiatedChallengeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingleExecutionChallengeInitiatedChallenge represents a InitiatedChallenge event raised by the SingleExecutionChallenge contract.
type SingleExecutionChallengeInitiatedChallenge struct {
	ChallengeIndex uint64
	StartState     GlobalState
	EndState       GlobalState
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterInitiatedChallenge is a free log retrieval operation binding the contract event 0x76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a.
//
// Solidity: event InitiatedChallenge(uint64 indexed challengeIndex, (bytes32[2],uint64[2]) startState, (bytes32[2],uint64[2]) endState)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) FilterInitiatedChallenge(opts *bind.FilterOpts, challengeIndex []uint64) (*SingleExecutionChallengeInitiatedChallengeIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _SingleExecutionChallenge.contract.FilterLogs(opts, "InitiatedChallenge", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &SingleExecutionChallengeInitiatedChallengeIterator{contract: _SingleExecutionChallenge.contract, event: "InitiatedChallenge", logs: logs, sub: sub}, nil
}

// WatchInitiatedChallenge is a free log subscription operation binding the contract event 0x76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a.
//
// Solidity: event InitiatedChallenge(uint64 indexed challengeIndex, (bytes32[2],uint64[2]) startState, (bytes32[2],uint64[2]) endState)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) WatchInitiatedChallenge(opts *bind.WatchOpts, sink chan<- *SingleExecutionChallengeInitiatedChallenge, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _SingleExecutionChallenge.contract.WatchLogs(opts, "InitiatedChallenge", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingleExecutionChallengeInitiatedChallenge)
				if err := _SingleExecutionChallenge.contract.UnpackLog(event, "InitiatedChallenge", log); err != nil {
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

// ParseInitiatedChallenge is a log parse operation binding the contract event 0x76604fe17af46c9b5f53ffe99ff23e0f655dab91886b07ac1fc0254319f7145a.
//
// Solidity: event InitiatedChallenge(uint64 indexed challengeIndex, (bytes32[2],uint64[2]) startState, (bytes32[2],uint64[2]) endState)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) ParseInitiatedChallenge(log types.Log) (*SingleExecutionChallengeInitiatedChallenge, error) {
	event := new(SingleExecutionChallengeInitiatedChallenge)
	if err := _SingleExecutionChallenge.contract.UnpackLog(event, "InitiatedChallenge", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SingleExecutionChallengeOneStepProofCompletedIterator is returned from FilterOneStepProofCompleted and is used to iterate over the raw logs and unpacked data for OneStepProofCompleted events raised by the SingleExecutionChallenge contract.
type SingleExecutionChallengeOneStepProofCompletedIterator struct {
	Event *SingleExecutionChallengeOneStepProofCompleted // Event containing the contract specifics and raw log

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
func (it *SingleExecutionChallengeOneStepProofCompletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingleExecutionChallengeOneStepProofCompleted)
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
		it.Event = new(SingleExecutionChallengeOneStepProofCompleted)
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
func (it *SingleExecutionChallengeOneStepProofCompletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingleExecutionChallengeOneStepProofCompletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingleExecutionChallengeOneStepProofCompleted represents a OneStepProofCompleted event raised by the SingleExecutionChallenge contract.
type SingleExecutionChallengeOneStepProofCompleted struct {
	ChallengeIndex uint64
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterOneStepProofCompleted is a free log retrieval operation binding the contract event 0xc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e.
//
// Solidity: event OneStepProofCompleted(uint64 indexed challengeIndex)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) FilterOneStepProofCompleted(opts *bind.FilterOpts, challengeIndex []uint64) (*SingleExecutionChallengeOneStepProofCompletedIterator, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _SingleExecutionChallenge.contract.FilterLogs(opts, "OneStepProofCompleted", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return &SingleExecutionChallengeOneStepProofCompletedIterator{contract: _SingleExecutionChallenge.contract, event: "OneStepProofCompleted", logs: logs, sub: sub}, nil
}

// WatchOneStepProofCompleted is a free log subscription operation binding the contract event 0xc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e.
//
// Solidity: event OneStepProofCompleted(uint64 indexed challengeIndex)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) WatchOneStepProofCompleted(opts *bind.WatchOpts, sink chan<- *SingleExecutionChallengeOneStepProofCompleted, challengeIndex []uint64) (event.Subscription, error) {

	var challengeIndexRule []interface{}
	for _, challengeIndexItem := range challengeIndex {
		challengeIndexRule = append(challengeIndexRule, challengeIndexItem)
	}

	logs, sub, err := _SingleExecutionChallenge.contract.WatchLogs(opts, "OneStepProofCompleted", challengeIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingleExecutionChallengeOneStepProofCompleted)
				if err := _SingleExecutionChallenge.contract.UnpackLog(event, "OneStepProofCompleted", log); err != nil {
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

// ParseOneStepProofCompleted is a log parse operation binding the contract event 0xc2cc42e04ff8c36de71c6a2937ea9f161dd0dd9e175f00caa26e5200643c781e.
//
// Solidity: event OneStepProofCompleted(uint64 indexed challengeIndex)
func (_SingleExecutionChallenge *SingleExecutionChallengeFilterer) ParseOneStepProofCompleted(log types.Log) (*SingleExecutionChallengeOneStepProofCompleted, error) {
	event := new(SingleExecutionChallengeOneStepProofCompleted)
	if err := _SingleExecutionChallenge.contract.UnpackLog(event, "OneStepProofCompleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TestWETH9MetaData contains all meta data concerning the TestWETH9 contract.
var TestWETH9MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name_\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol_\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"deposit\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b506040516200107838038062001078833981016040819052620000349162000123565b818160036200004483826200021c565b5060046200005382826200021c565b5050505050620002e8565b634e487b7160e01b600052604160045260246000fd5b600082601f8301126200008657600080fd5b81516001600160401b0380821115620000a357620000a36200005e565b604051601f8301601f19908116603f01168101908282118183101715620000ce57620000ce6200005e565b81604052838152602092508683858801011115620000eb57600080fd5b600091505b838210156200010f5785820183015181830184015290820190620000f0565b600093810190920192909252949350505050565b600080604083850312156200013757600080fd5b82516001600160401b03808211156200014f57600080fd5b6200015d8683870162000074565b935060208501519150808211156200017457600080fd5b50620001838582860162000074565b9150509250929050565b600181811c90821680620001a257607f821691505b602082108103620001c357634e487b7160e01b600052602260045260246000fd5b50919050565b601f8211156200021757600081815260208120601f850160051c81016020861015620001f25750805b601f850160051c820191505b818110156200021357828155600101620001fe565b5050505b505050565b81516001600160401b038111156200023857620002386200005e565b62000250816200024984546200018d565b84620001c9565b602080601f8311600181146200028857600084156200026f5750858301515b600019600386901b1c1916600185901b17855562000213565b600085815260208120601f198616915b82811015620002b95788860151825594840194600190910190840162000298565b5085821015620002d85787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b610d8080620002f86000396000f3fe6080604052600436106100d25760003560e01c8063395093511161007f578063a457c2d711610059578063a457c2d71461021a578063a9059cbb1461023a578063d0e30db01461025a578063dd62ed3e1461026257600080fd5b806339509351146101af57806370a08231146101cf57806395d89b411461020557600080fd5b806323b872dd116100b057806323b872dd146101515780632e1a7d4d14610171578063313ce5671461019357600080fd5b806306fdde03146100d7578063095ea7b31461010257806318160ddd14610132575b600080fd5b3480156100e357600080fd5b506100ec6102a8565b6040516100f99190610b46565b60405180910390f35b34801561010e57600080fd5b5061012261011d366004610bce565b61033a565b60405190151581526020016100f9565b34801561013e57600080fd5b506002545b6040519081526020016100f9565b34801561015d57600080fd5b5061012261016c366004610bf8565b610354565b34801561017d57600080fd5b5061019161018c366004610c34565b610378565b005b34801561019f57600080fd5b50604051601281526020016100f9565b3480156101bb57600080fd5b506101226101ca366004610bce565b6103b3565b3480156101db57600080fd5b506101436101ea366004610c4d565b6001600160a01b031660009081526020819052604090205490565b34801561021157600080fd5b506100ec6103f2565b34801561022657600080fd5b50610122610235366004610bce565b610401565b34801561024657600080fd5b50610122610255366004610bce565b6104b0565b6101916104be565b34801561026e57600080fd5b5061014361027d366004610c6f565b6001600160a01b03918216600090815260016020908152604080832093909416825291909152205490565b6060600380546102b790610ca2565b80601f01602080910402602001604051908101604052809291908181526020018280546102e390610ca2565b80156103305780601f1061030557610100808354040283529160200191610330565b820191906000526020600020905b81548152906001019060200180831161031357829003601f168201915b5050505050905090565b6000336103488185856104ca565b60019150505b92915050565b600033610362858285610623565b61036d8585856106d3565b506001949350505050565b61038233826108ea565b604051339082156108fc029083906000818181858888f193505050501580156103af573d6000803e3d6000fd5b5050565b3360008181526001602090815260408083206001600160a01b038716845290915281205490919061034890829086906103ed908790610d24565b6104ca565b6060600480546102b790610ca2565b3360008181526001602090815260408083206001600160a01b0387168452909152812054909190838110156104a35760405162461bcd60e51b815260206004820152602560248201527f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760448201527f207a65726f00000000000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b61036d82868684036104ca565b6000336103488185856106d3565b6104c83334610a67565b565b6001600160a01b0383166105455760405162461bcd60e51b8152602060048201526024808201527f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460448201527f7265737300000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b0382166105c15760405162461bcd60e51b815260206004820152602260248201527f45524332303a20617070726f766520746f20746865207a65726f20616464726560448201527f7373000000000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b0383811660008181526001602090815260408083209487168084529482529182902085905590518481527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92591015b60405180910390a3505050565b6001600160a01b038381166000908152600160209081526040808320938616835292905220547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81146106cd57818110156106c05760405162461bcd60e51b815260206004820152601d60248201527f45524332303a20696e73756666696369656e7420616c6c6f77616e6365000000604482015260640161049a565b6106cd84848484036104ca565b50505050565b6001600160a01b03831661074f5760405162461bcd60e51b815260206004820152602560248201527f45524332303a207472616e736665722066726f6d20746865207a65726f20616460448201527f6472657373000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b0382166107cb5760405162461bcd60e51b815260206004820152602360248201527f45524332303a207472616e7366657220746f20746865207a65726f206164647260448201527f6573730000000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b0383166000908152602081905260409020548181101561085a5760405162461bcd60e51b815260206004820152602660248201527f45524332303a207472616e7366657220616d6f756e742065786365656473206260448201527f616c616e63650000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b03808516600090815260208190526040808220858503905591851681529081208054849290610891908490610d24565b92505081905550826001600160a01b0316846001600160a01b03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040516108dd91815260200190565b60405180910390a36106cd565b6001600160a01b0382166109665760405162461bcd60e51b815260206004820152602160248201527f45524332303a206275726e2066726f6d20746865207a65726f2061646472657360448201527f7300000000000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b038216600090815260208190526040902054818110156109f55760405162461bcd60e51b815260206004820152602260248201527f45524332303a206275726e20616d6f756e7420657863656564732062616c616e60448201527f6365000000000000000000000000000000000000000000000000000000000000606482015260840161049a565b6001600160a01b0383166000908152602081905260408120838303905560028054849290610a24908490610d37565b90915550506040518281526000906001600160a01b038516907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef90602001610616565b6001600160a01b038216610abd5760405162461bcd60e51b815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f206164647265737300604482015260640161049a565b8060026000828254610acf9190610d24565b90915550506001600160a01b03821660009081526020819052604081208054839290610afc908490610d24565b90915550506040518181526001600160a01b038316906000907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9060200160405180910390a35050565b600060208083528351808285015260005b81811015610b7357858101830151858201604001528201610b57565b5060006040828601015260407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f8301168501019250505092915050565b80356001600160a01b0381168114610bc957600080fd5b919050565b60008060408385031215610be157600080fd5b610bea83610bb2565b946020939093013593505050565b600080600060608486031215610c0d57600080fd5b610c1684610bb2565b9250610c2460208501610bb2565b9150604084013590509250925092565b600060208284031215610c4657600080fd5b5035919050565b600060208284031215610c5f57600080fd5b610c6882610bb2565b9392505050565b60008060408385031215610c8257600080fd5b610c8b83610bb2565b9150610c9960208401610bb2565b90509250929050565b600181811c90821680610cb657607f821691505b602082108103610cef577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b8082018082111561034e5761034e610cf5565b8181038181111561034e5761034e610cf556fea264697066735822122072827c8c0faea0de8b1e5a02360be312f4bdc3d1a0c4421d768d72b64f3b947a64736f6c63430008110033",
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
