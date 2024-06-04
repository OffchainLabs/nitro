// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package test_helpersgen

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

// BridgeTesterMetaData contains all meta data concerning the BridgeTester contract.
var BridgeTesterMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"NotContract\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotDelayedInbox\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotOutbox\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotRollupOrOwner\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"BridgeCallTriggered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"InboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeInboxAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"MessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"OutboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"}],\"name\":\"RollupUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newSequencerInbox\",\"type\":\"address\"}],\"name\":\"SequencerInboxUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"acceptFundsFromOldBridge\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"activeOutbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedDelayedInboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"}],\"name\":\"allowedDelayedInboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedOutboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"}],\"name\":\"allowedOutboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"delayedInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"delayedMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"enqueueDelayedMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"enqueueSequencerMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"seqMessageIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"acc\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeCall\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"rollup_\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"sequencerInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerReportedSubMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setDelayedInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setOutbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sequencerInbox\",\"type\":\"address\"}],\"name\":\"setSequencerInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"batchPoster\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"}],\"name\":\"submitBatchSpendingReport\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"_rollup\",\"type\":\"address\"}],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x60a06040523060805234801561001457600080fd5b5060805161151b61002d6000396000505061151b6000f3fe6080604052600436106101785760003560e01c80639e5d4c49116100cb578063cee3d7281161007f578063e77145f411610059578063e77145f414610234578063eca067ad1461048e578063ee35f327146104a357600080fd5b8063cee3d7281461042e578063d5719dc21461044e578063e76f5c8d1461046e57600080fd5b8063ae60bd13116100b0578063ae60bd13146103b1578063c4d66de8146103ee578063cb23bcb51461040e57600080fd5b80639e5d4c491461036e578063ab5d89431461039c57600080fd5b80635fca4a161161012d5780638db5993b116101075780638db5993b146102d9578063919cc706146102ec578063945e11471461033657600080fd5b80635fca4a16146102565780637a88b1071461026c57806386598a561461028f57600080fd5b8063413b35bd1161015e578063413b35bd146101c857806347fb24c5146102145780634f61f8501461023657600080fd5b806284120c1461018457806316bf5579146101a857600080fd5b3661017f57005b600080fd5b34801561019057600080fd5b506009545b6040519081526020015b60405180910390f35b3480156101b457600080fd5b506101956101c336600461121c565b6104c3565b3480156101d457600080fd5b506102046101e336600461124d565b6001600160a01b031660009081526002602052604090206001015460ff1690565b604051901515815260200161019f565b34801561022057600080fd5b5061023461022f366004611271565b6104e4565b005b34801561024257600080fd5b5061023461025136600461124d565b610800565b34801561026257600080fd5b50610195600a5481565b34801561027857600080fd5b506101956102873660046112af565b600092915050565b34801561029b57600080fd5b506102b96102aa3660046112db565b50600093849350839250829150565b60408051948552602085019390935291830152606082015260800161019f565b6101956102e736600461130d565b61092b565b3480156102f857600080fd5b5061023461030736600461124d565b6006805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0392909216919091179055565b34801561034257600080fd5b5061035661035136600461121c565b61098f565b6040516001600160a01b03909116815260200161019f565b34801561037a57600080fd5b5061038e610389366004611354565b6109b9565b60405161019f9291906113dd565b3480156103a857600080fd5b50610356610b66565b3480156103bd57600080fd5b506102046103cc36600461124d565b6001600160a01b03166000908152600160208190526040909120015460ff1690565b3480156103fa57600080fd5b5061023461040936600461124d565b610bb1565b34801561041a57600080fd5b50600654610356906001600160a01b031681565b34801561043a57600080fd5b50610234610449366004611271565b610d1d565b34801561045a57600080fd5b5061019561046936600461121c565b611034565b34801561047a57600080fd5b5061035661048936600461121c565b611044565b34801561049a57600080fd5b50600854610195565b3480156104af57600080fd5b50600754610356906001600160a01b031681565b600981815481106104d357600080fd5b600091825260209091200154905081565b6006546001600160a01b031633146105b35760065460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610540573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105649190611435565b9050336001600160a01b038216146105b157600654604051630739600760e01b81523360048201526001600160a01b03918216602482015290821660448201526064015b60405180910390fd5b505b6001600160a01b0382166000818152600160208181526040928390209182015492518515158152919360ff90931692917f6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521910160405180910390a28080156106185750825b8061062a57508015801561062a575082155b156106355750505050565b82156106d057604080518082018252600380548252600160208084018281526001600160a01b038a166000818152928490529582209451855551938201805460ff1916941515949094179093558154908101825591527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b01805473ffffffffffffffffffffffffffffffffffffffff191690911790556107f9565b600380546106e090600190611452565b815481106106f0576106f0611473565b6000918252602090912001548254600380546001600160a01b0390931692909190811061071f5761071f611473565b9060005260206000200160006101000a8154816001600160a01b0302191690836001600160a01b03160217905550816000015460016000600385600001548154811061076d5761076d611473565b60009182526020808320909101546001600160a01b0316835282019290925260400190205560038054806107a3576107a3611489565b600082815260208082208301600019908101805473ffffffffffffffffffffffffffffffffffffffff191690559092019092556001600160a01b03861682526001908190526040822091825501805460ff191690555b50505b5050565b6006546001600160a01b031633146108ca5760065460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa15801561085c573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906108809190611435565b9050336001600160a01b038216146108c857600654604051630739600760e01b81523360048201526001600160a01b03918216602482015290821660448201526064016105a8565b505b6007805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0383169081179091556040519081527f8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a9060200160405180910390a150565b3360009081526001602081905260408220015460ff16610979576040517fb6c60ea30000000000000000000000000000000000000000000000000000000081523360048201526024016105a8565b610987848443424887611054565b949350505050565b6004818154811061099f57600080fd5b6000918252602090912001546001600160a01b0316905081565b3360009081526002602052604081206001015460609060ff16610a0a576040517f32ea82ab0000000000000000000000000000000000000000000000000000000081523360048201526024016105a8565b8215801590610a2157506001600160a01b0386163b155b15610a63576040517fb5cf5b8f0000000000000000000000000000000000000000000000000000000081526001600160a01b03871660048201526024016105a8565b6005805473ffffffffffffffffffffffffffffffffffffffff19811633179091556040516001600160a01b03918216918816908790610aa5908890889061149f565b60006040518083038185875af1925050503d8060008114610ae2576040519150601f19603f3d011682016040523d82523d6000602084013e610ae7565b606091505b506005805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038581169190911790915560405192955090935088169033907f2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d46690610b54908a908a908a906114af565b60405180910390a35094509492505050565b6005546000906001600160a01b03167fffffffffffffffffffffffff000000000000000000000000000000000000000101610ba15750600090565b506005546001600160a01b031690565b600054610100900460ff1615808015610bd15750600054600160ff909116105b80610beb5750303b158015610beb575060005460ff166001145b610c77576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016105a8565b6000805460ff191660011790558015610c9a576000805461ff0019166101001790555b600580546001600160a01b0373ffffffffffffffffffffffffffffffffffffffff1991821681179092556006805490911691841691909117905580156107fc576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050565b6006546001600160a01b03163314610de75760065460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610d79573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610d9d9190611435565b9050336001600160a01b03821614610de557600654604051630739600760e01b81523360048201526001600160a01b03918216602482015290821660448201526064016105a8565b505b6001600160a01b038216600081815260026020908152604091829020600181015492518515158152909360ff90931692917f49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa910160405180910390a2808015610e4d5750825b80610e5f575080158015610e5f575082155b15610e6a5750505050565b8215610f0657604080518082018252600480548252600160208084018281526001600160a01b038a16600081815260029093529582209451855551938201805460ff1916941515949094179093558154908101825591527f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b01805473ffffffffffffffffffffffffffffffffffffffff191690911790556107f9565b60048054610f1690600190611452565b81548110610f2657610f26611473565b6000918252602090912001548254600480546001600160a01b03909316929091908110610f5557610f55611473565b9060005260206000200160006101000a8154816001600160a01b0302191690836001600160a01b031602179055508160000154600260006004856000015481548110610fa357610fa3611473565b60009182526020808320909101546001600160a01b031683528201929092526040019020556004805480610fd957610fd9611489565b600082815260208082208301600019908101805473ffffffffffffffffffffffffffffffffffffffff191690559092019092556001600160a01b03861682526002905260408120908155600101805460ff1916905550505050565b600881815481106104d357600080fd5b6003818154811061099f57600080fd5b600854604080517fff0000000000000000000000000000000000000000000000000000000000000060f88a901b166020808301919091527fffffffffffffffffffffffffffffffffffffffff00000000000000000000000060608a901b1660218301527fffffffffffffffff00000000000000000000000000000000000000000000000060c089811b8216603585015288901b16603d830152604582018490526065820186905260858083018690528351808403909101815260a590920190925280519101206000919060008215611151576008611133600185611452565b8154811061114357611143611473565b906000526020600020015490505b600861115d82846111ed565b8154600181018355600092835260209283902001556040805133815260ff8d16928101929092526001600160a01b038b1682820152606082018790526080820188905267ffffffffffffffff891660a083015251829185917f5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe19181900360c00190a3509098975050505050505050565b604080516020808201859052818301849052825180830384018152606090920190925280519101205b92915050565b60006020828403121561122e57600080fd5b5035919050565b6001600160a01b038116811461124a57600080fd5b50565b60006020828403121561125f57600080fd5b813561126a81611235565b9392505050565b6000806040838503121561128457600080fd5b823561128f81611235565b9150602083013580151581146112a457600080fd5b809150509250929050565b600080604083850312156112c257600080fd5b82356112cd81611235565b946020939093013593505050565b600080600080608085870312156112f157600080fd5b5050823594602084013594506040840135936060013592509050565b60008060006060848603121561132257600080fd5b833560ff8116811461133357600080fd5b9250602084013561134381611235565b929592945050506040919091013590565b6000806000806060858703121561136a57600080fd5b843561137581611235565b935060208501359250604085013567ffffffffffffffff8082111561139957600080fd5b818701915087601f8301126113ad57600080fd5b8135818111156113bc57600080fd5b8860208285010111156113ce57600080fd5b95989497505060200194505050565b821515815260006020604081840152835180604085015260005b81811015611413578581018301518582016060015282016113f7565b506000606082860101526060601f19601f830116850101925050509392505050565b60006020828403121561144757600080fd5b815161126a81611235565b8181038181111561121657634e487b7160e01b600052601160045260246000fd5b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052603160045260246000fd5b8183823760009101908152919050565b83815260406020820152816040820152818360608301376000818301606090810191909152601f909201601f191601019291505056fea264697066735822122036a77fb8da25232c1499bd97780f1b8fc63c7125caa167d04d26c589757bad6064736f6c63430008110033",
}

// BridgeTesterABI is the input ABI used to generate the binding from.
// Deprecated: Use BridgeTesterMetaData.ABI instead.
var BridgeTesterABI = BridgeTesterMetaData.ABI

// BridgeTesterBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BridgeTesterMetaData.Bin instead.
var BridgeTesterBin = BridgeTesterMetaData.Bin

// DeployBridgeTester deploys a new Ethereum contract, binding an instance of BridgeTester to it.
func DeployBridgeTester(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BridgeTester, error) {
	parsed, err := BridgeTesterMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BridgeTesterBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BridgeTester{BridgeTesterCaller: BridgeTesterCaller{contract: contract}, BridgeTesterTransactor: BridgeTesterTransactor{contract: contract}, BridgeTesterFilterer: BridgeTesterFilterer{contract: contract}}, nil
}

// BridgeTester is an auto generated Go binding around an Ethereum contract.
type BridgeTester struct {
	BridgeTesterCaller     // Read-only binding to the contract
	BridgeTesterTransactor // Write-only binding to the contract
	BridgeTesterFilterer   // Log filterer for contract events
}

// BridgeTesterCaller is an auto generated read-only Go binding around an Ethereum contract.
type BridgeTesterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeTesterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BridgeTesterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeTesterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BridgeTesterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeTesterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BridgeTesterSession struct {
	Contract     *BridgeTester     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BridgeTesterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BridgeTesterCallerSession struct {
	Contract *BridgeTesterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// BridgeTesterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BridgeTesterTransactorSession struct {
	Contract     *BridgeTesterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// BridgeTesterRaw is an auto generated low-level Go binding around an Ethereum contract.
type BridgeTesterRaw struct {
	Contract *BridgeTester // Generic contract binding to access the raw methods on
}

// BridgeTesterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BridgeTesterCallerRaw struct {
	Contract *BridgeTesterCaller // Generic read-only contract binding to access the raw methods on
}

// BridgeTesterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BridgeTesterTransactorRaw struct {
	Contract *BridgeTesterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBridgeTester creates a new instance of BridgeTester, bound to a specific deployed contract.
func NewBridgeTester(address common.Address, backend bind.ContractBackend) (*BridgeTester, error) {
	contract, err := bindBridgeTester(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BridgeTester{BridgeTesterCaller: BridgeTesterCaller{contract: contract}, BridgeTesterTransactor: BridgeTesterTransactor{contract: contract}, BridgeTesterFilterer: BridgeTesterFilterer{contract: contract}}, nil
}

// NewBridgeTesterCaller creates a new read-only instance of BridgeTester, bound to a specific deployed contract.
func NewBridgeTesterCaller(address common.Address, caller bind.ContractCaller) (*BridgeTesterCaller, error) {
	contract, err := bindBridgeTester(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeTesterCaller{contract: contract}, nil
}

// NewBridgeTesterTransactor creates a new write-only instance of BridgeTester, bound to a specific deployed contract.
func NewBridgeTesterTransactor(address common.Address, transactor bind.ContractTransactor) (*BridgeTesterTransactor, error) {
	contract, err := bindBridgeTester(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeTesterTransactor{contract: contract}, nil
}

// NewBridgeTesterFilterer creates a new log filterer instance of BridgeTester, bound to a specific deployed contract.
func NewBridgeTesterFilterer(address common.Address, filterer bind.ContractFilterer) (*BridgeTesterFilterer, error) {
	contract, err := bindBridgeTester(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BridgeTesterFilterer{contract: contract}, nil
}

// bindBridgeTester binds a generic wrapper to an already deployed contract.
func bindBridgeTester(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BridgeTesterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeTester *BridgeTesterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeTester.Contract.BridgeTesterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeTester *BridgeTesterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeTester.Contract.BridgeTesterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeTester *BridgeTesterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeTester.Contract.BridgeTesterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeTester *BridgeTesterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeTester.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeTester *BridgeTesterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeTester.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeTester *BridgeTesterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeTester.Contract.contract.Transact(opts, method, params...)
}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_BridgeTester *BridgeTesterCaller) ActiveOutbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "activeOutbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_BridgeTester *BridgeTesterSession) ActiveOutbox() (common.Address, error) {
	return _BridgeTester.Contract.ActiveOutbox(&_BridgeTester.CallOpts)
}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_BridgeTester *BridgeTesterCallerSession) ActiveOutbox() (common.Address, error) {
	return _BridgeTester.Contract.ActiveOutbox(&_BridgeTester.CallOpts)
}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_BridgeTester *BridgeTesterCaller) AllowedDelayedInboxList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "allowedDelayedInboxList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_BridgeTester *BridgeTesterSession) AllowedDelayedInboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeTester.Contract.AllowedDelayedInboxList(&_BridgeTester.CallOpts, arg0)
}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_BridgeTester *BridgeTesterCallerSession) AllowedDelayedInboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeTester.Contract.AllowedDelayedInboxList(&_BridgeTester.CallOpts, arg0)
}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_BridgeTester *BridgeTesterCaller) AllowedDelayedInboxes(opts *bind.CallOpts, inbox common.Address) (bool, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "allowedDelayedInboxes", inbox)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_BridgeTester *BridgeTesterSession) AllowedDelayedInboxes(inbox common.Address) (bool, error) {
	return _BridgeTester.Contract.AllowedDelayedInboxes(&_BridgeTester.CallOpts, inbox)
}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_BridgeTester *BridgeTesterCallerSession) AllowedDelayedInboxes(inbox common.Address) (bool, error) {
	return _BridgeTester.Contract.AllowedDelayedInboxes(&_BridgeTester.CallOpts, inbox)
}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_BridgeTester *BridgeTesterCaller) AllowedOutboxList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "allowedOutboxList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_BridgeTester *BridgeTesterSession) AllowedOutboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeTester.Contract.AllowedOutboxList(&_BridgeTester.CallOpts, arg0)
}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_BridgeTester *BridgeTesterCallerSession) AllowedOutboxList(arg0 *big.Int) (common.Address, error) {
	return _BridgeTester.Contract.AllowedOutboxList(&_BridgeTester.CallOpts, arg0)
}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_BridgeTester *BridgeTesterCaller) AllowedOutboxes(opts *bind.CallOpts, outbox common.Address) (bool, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "allowedOutboxes", outbox)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_BridgeTester *BridgeTesterSession) AllowedOutboxes(outbox common.Address) (bool, error) {
	return _BridgeTester.Contract.AllowedOutboxes(&_BridgeTester.CallOpts, outbox)
}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_BridgeTester *BridgeTesterCallerSession) AllowedOutboxes(outbox common.Address) (bool, error) {
	return _BridgeTester.Contract.AllowedOutboxes(&_BridgeTester.CallOpts, outbox)
}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeTester *BridgeTesterCaller) DelayedInboxAccs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "delayedInboxAccs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeTester *BridgeTesterSession) DelayedInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeTester.Contract.DelayedInboxAccs(&_BridgeTester.CallOpts, arg0)
}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeTester *BridgeTesterCallerSession) DelayedInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeTester.Contract.DelayedInboxAccs(&_BridgeTester.CallOpts, arg0)
}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_BridgeTester *BridgeTesterCaller) DelayedMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "delayedMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_BridgeTester *BridgeTesterSession) DelayedMessageCount() (*big.Int, error) {
	return _BridgeTester.Contract.DelayedMessageCount(&_BridgeTester.CallOpts)
}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_BridgeTester *BridgeTesterCallerSession) DelayedMessageCount() (*big.Int, error) {
	return _BridgeTester.Contract.DelayedMessageCount(&_BridgeTester.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_BridgeTester *BridgeTesterCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_BridgeTester *BridgeTesterSession) Rollup() (common.Address, error) {
	return _BridgeTester.Contract.Rollup(&_BridgeTester.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_BridgeTester *BridgeTesterCallerSession) Rollup() (common.Address, error) {
	return _BridgeTester.Contract.Rollup(&_BridgeTester.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_BridgeTester *BridgeTesterCaller) SequencerInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "sequencerInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_BridgeTester *BridgeTesterSession) SequencerInbox() (common.Address, error) {
	return _BridgeTester.Contract.SequencerInbox(&_BridgeTester.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_BridgeTester *BridgeTesterCallerSession) SequencerInbox() (common.Address, error) {
	return _BridgeTester.Contract.SequencerInbox(&_BridgeTester.CallOpts)
}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeTester *BridgeTesterCaller) SequencerInboxAccs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "sequencerInboxAccs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeTester *BridgeTesterSession) SequencerInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeTester.Contract.SequencerInboxAccs(&_BridgeTester.CallOpts, arg0)
}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_BridgeTester *BridgeTesterCallerSession) SequencerInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _BridgeTester.Contract.SequencerInboxAccs(&_BridgeTester.CallOpts, arg0)
}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_BridgeTester *BridgeTesterCaller) SequencerMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "sequencerMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_BridgeTester *BridgeTesterSession) SequencerMessageCount() (*big.Int, error) {
	return _BridgeTester.Contract.SequencerMessageCount(&_BridgeTester.CallOpts)
}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_BridgeTester *BridgeTesterCallerSession) SequencerMessageCount() (*big.Int, error) {
	return _BridgeTester.Contract.SequencerMessageCount(&_BridgeTester.CallOpts)
}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_BridgeTester *BridgeTesterCaller) SequencerReportedSubMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeTester.contract.Call(opts, &out, "sequencerReportedSubMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_BridgeTester *BridgeTesterSession) SequencerReportedSubMessageCount() (*big.Int, error) {
	return _BridgeTester.Contract.SequencerReportedSubMessageCount(&_BridgeTester.CallOpts)
}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_BridgeTester *BridgeTesterCallerSession) SequencerReportedSubMessageCount() (*big.Int, error) {
	return _BridgeTester.Contract.SequencerReportedSubMessageCount(&_BridgeTester.CallOpts)
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_BridgeTester *BridgeTesterTransactor) AcceptFundsFromOldBridge(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeTester.contract.Transact(opts, "acceptFundsFromOldBridge")
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_BridgeTester *BridgeTesterSession) AcceptFundsFromOldBridge() (*types.Transaction, error) {
	return _BridgeTester.Contract.AcceptFundsFromOldBridge(&_BridgeTester.TransactOpts)
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_BridgeTester *BridgeTesterTransactorSession) AcceptFundsFromOldBridge() (*types.Transaction, error) {
	return _BridgeTester.Contract.AcceptFundsFromOldBridge(&_BridgeTester.TransactOpts)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_BridgeTester *BridgeTesterTransactor) EnqueueDelayedMessage(opts *bind.TransactOpts, kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeTester.contract.Transact(opts, "enqueueDelayedMessage", kind, sender, messageDataHash)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_BridgeTester *BridgeTesterSession) EnqueueDelayedMessage(kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeTester.Contract.EnqueueDelayedMessage(&_BridgeTester.TransactOpts, kind, sender, messageDataHash)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_BridgeTester *BridgeTesterTransactorSession) EnqueueDelayedMessage(kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _BridgeTester.Contract.EnqueueDelayedMessage(&_BridgeTester.TransactOpts, kind, sender, messageDataHash)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_BridgeTester *BridgeTesterTransactor) EnqueueSequencerMessage(opts *bind.TransactOpts, dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _BridgeTester.contract.Transact(opts, "enqueueSequencerMessage", dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_BridgeTester *BridgeTesterSession) EnqueueSequencerMessage(dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _BridgeTester.Contract.EnqueueSequencerMessage(&_BridgeTester.TransactOpts, dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_BridgeTester *BridgeTesterTransactorSession) EnqueueSequencerMessage(dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _BridgeTester.Contract.EnqueueSequencerMessage(&_BridgeTester.TransactOpts, dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_BridgeTester *BridgeTesterTransactor) ExecuteCall(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _BridgeTester.contract.Transact(opts, "executeCall", to, value, data)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_BridgeTester *BridgeTesterSession) ExecuteCall(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _BridgeTester.Contract.ExecuteCall(&_BridgeTester.TransactOpts, to, value, data)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_BridgeTester *BridgeTesterTransactorSession) ExecuteCall(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _BridgeTester.Contract.ExecuteCall(&_BridgeTester.TransactOpts, to, value, data)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_BridgeTester *BridgeTesterTransactor) Initialize(opts *bind.TransactOpts, rollup_ common.Address) (*types.Transaction, error) {
	return _BridgeTester.contract.Transact(opts, "initialize", rollup_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_BridgeTester *BridgeTesterSession) Initialize(rollup_ common.Address) (*types.Transaction, error) {
	return _BridgeTester.Contract.Initialize(&_BridgeTester.TransactOpts, rollup_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_BridgeTester *BridgeTesterTransactorSession) Initialize(rollup_ common.Address) (*types.Transaction, error) {
	return _BridgeTester.Contract.Initialize(&_BridgeTester.TransactOpts, rollup_)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_BridgeTester *BridgeTesterTransactor) SetDelayedInbox(opts *bind.TransactOpts, inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeTester.contract.Transact(opts, "setDelayedInbox", inbox, enabled)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_BridgeTester *BridgeTesterSession) SetDelayedInbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeTester.Contract.SetDelayedInbox(&_BridgeTester.TransactOpts, inbox, enabled)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_BridgeTester *BridgeTesterTransactorSession) SetDelayedInbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeTester.Contract.SetDelayedInbox(&_BridgeTester.TransactOpts, inbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address outbox, bool enabled) returns()
func (_BridgeTester *BridgeTesterTransactor) SetOutbox(opts *bind.TransactOpts, outbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeTester.contract.Transact(opts, "setOutbox", outbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address outbox, bool enabled) returns()
func (_BridgeTester *BridgeTesterSession) SetOutbox(outbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeTester.Contract.SetOutbox(&_BridgeTester.TransactOpts, outbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address outbox, bool enabled) returns()
func (_BridgeTester *BridgeTesterTransactorSession) SetOutbox(outbox common.Address, enabled bool) (*types.Transaction, error) {
	return _BridgeTester.Contract.SetOutbox(&_BridgeTester.TransactOpts, outbox, enabled)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_BridgeTester *BridgeTesterTransactor) SetSequencerInbox(opts *bind.TransactOpts, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _BridgeTester.contract.Transact(opts, "setSequencerInbox", _sequencerInbox)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_BridgeTester *BridgeTesterSession) SetSequencerInbox(_sequencerInbox common.Address) (*types.Transaction, error) {
	return _BridgeTester.Contract.SetSequencerInbox(&_BridgeTester.TransactOpts, _sequencerInbox)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_BridgeTester *BridgeTesterTransactorSession) SetSequencerInbox(_sequencerInbox common.Address) (*types.Transaction, error) {
	return _BridgeTester.Contract.SetSequencerInbox(&_BridgeTester.TransactOpts, _sequencerInbox)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address batchPoster, bytes32 dataHash) returns(uint256)
func (_BridgeTester *BridgeTesterTransactor) SubmitBatchSpendingReport(opts *bind.TransactOpts, batchPoster common.Address, dataHash [32]byte) (*types.Transaction, error) {
	return _BridgeTester.contract.Transact(opts, "submitBatchSpendingReport", batchPoster, dataHash)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address batchPoster, bytes32 dataHash) returns(uint256)
func (_BridgeTester *BridgeTesterSession) SubmitBatchSpendingReport(batchPoster common.Address, dataHash [32]byte) (*types.Transaction, error) {
	return _BridgeTester.Contract.SubmitBatchSpendingReport(&_BridgeTester.TransactOpts, batchPoster, dataHash)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address batchPoster, bytes32 dataHash) returns(uint256)
func (_BridgeTester *BridgeTesterTransactorSession) SubmitBatchSpendingReport(batchPoster common.Address, dataHash [32]byte) (*types.Transaction, error) {
	return _BridgeTester.Contract.SubmitBatchSpendingReport(&_BridgeTester.TransactOpts, batchPoster, dataHash)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_BridgeTester *BridgeTesterTransactor) UpdateRollupAddress(opts *bind.TransactOpts, _rollup common.Address) (*types.Transaction, error) {
	return _BridgeTester.contract.Transact(opts, "updateRollupAddress", _rollup)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_BridgeTester *BridgeTesterSession) UpdateRollupAddress(_rollup common.Address) (*types.Transaction, error) {
	return _BridgeTester.Contract.UpdateRollupAddress(&_BridgeTester.TransactOpts, _rollup)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_BridgeTester *BridgeTesterTransactorSession) UpdateRollupAddress(_rollup common.Address) (*types.Transaction, error) {
	return _BridgeTester.Contract.UpdateRollupAddress(&_BridgeTester.TransactOpts, _rollup)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BridgeTester *BridgeTesterTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeTester.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BridgeTester *BridgeTesterSession) Receive() (*types.Transaction, error) {
	return _BridgeTester.Contract.Receive(&_BridgeTester.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BridgeTester *BridgeTesterTransactorSession) Receive() (*types.Transaction, error) {
	return _BridgeTester.Contract.Receive(&_BridgeTester.TransactOpts)
}

// BridgeTesterBridgeCallTriggeredIterator is returned from FilterBridgeCallTriggered and is used to iterate over the raw logs and unpacked data for BridgeCallTriggered events raised by the BridgeTester contract.
type BridgeTesterBridgeCallTriggeredIterator struct {
	Event *BridgeTesterBridgeCallTriggered // Event containing the contract specifics and raw log

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
func (it *BridgeTesterBridgeCallTriggeredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeTesterBridgeCallTriggered)
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
		it.Event = new(BridgeTesterBridgeCallTriggered)
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
func (it *BridgeTesterBridgeCallTriggeredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeTesterBridgeCallTriggeredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeTesterBridgeCallTriggered represents a BridgeCallTriggered event raised by the BridgeTester contract.
type BridgeTesterBridgeCallTriggered struct {
	Outbox common.Address
	To     common.Address
	Value  *big.Int
	Data   []byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBridgeCallTriggered is a free log retrieval operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_BridgeTester *BridgeTesterFilterer) FilterBridgeCallTriggered(opts *bind.FilterOpts, outbox []common.Address, to []common.Address) (*BridgeTesterBridgeCallTriggeredIterator, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _BridgeTester.contract.FilterLogs(opts, "BridgeCallTriggered", outboxRule, toRule)
	if err != nil {
		return nil, err
	}
	return &BridgeTesterBridgeCallTriggeredIterator{contract: _BridgeTester.contract, event: "BridgeCallTriggered", logs: logs, sub: sub}, nil
}

// WatchBridgeCallTriggered is a free log subscription operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_BridgeTester *BridgeTesterFilterer) WatchBridgeCallTriggered(opts *bind.WatchOpts, sink chan<- *BridgeTesterBridgeCallTriggered, outbox []common.Address, to []common.Address) (event.Subscription, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _BridgeTester.contract.WatchLogs(opts, "BridgeCallTriggered", outboxRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeTesterBridgeCallTriggered)
				if err := _BridgeTester.contract.UnpackLog(event, "BridgeCallTriggered", log); err != nil {
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
func (_BridgeTester *BridgeTesterFilterer) ParseBridgeCallTriggered(log types.Log) (*BridgeTesterBridgeCallTriggered, error) {
	event := new(BridgeTesterBridgeCallTriggered)
	if err := _BridgeTester.contract.UnpackLog(event, "BridgeCallTriggered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeTesterInboxToggleIterator is returned from FilterInboxToggle and is used to iterate over the raw logs and unpacked data for InboxToggle events raised by the BridgeTester contract.
type BridgeTesterInboxToggleIterator struct {
	Event *BridgeTesterInboxToggle // Event containing the contract specifics and raw log

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
func (it *BridgeTesterInboxToggleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeTesterInboxToggle)
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
		it.Event = new(BridgeTesterInboxToggle)
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
func (it *BridgeTesterInboxToggleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeTesterInboxToggleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeTesterInboxToggle represents a InboxToggle event raised by the BridgeTester contract.
type BridgeTesterInboxToggle struct {
	Inbox   common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInboxToggle is a free log retrieval operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_BridgeTester *BridgeTesterFilterer) FilterInboxToggle(opts *bind.FilterOpts, inbox []common.Address) (*BridgeTesterInboxToggleIterator, error) {

	var inboxRule []interface{}
	for _, inboxItem := range inbox {
		inboxRule = append(inboxRule, inboxItem)
	}

	logs, sub, err := _BridgeTester.contract.FilterLogs(opts, "InboxToggle", inboxRule)
	if err != nil {
		return nil, err
	}
	return &BridgeTesterInboxToggleIterator{contract: _BridgeTester.contract, event: "InboxToggle", logs: logs, sub: sub}, nil
}

// WatchInboxToggle is a free log subscription operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_BridgeTester *BridgeTesterFilterer) WatchInboxToggle(opts *bind.WatchOpts, sink chan<- *BridgeTesterInboxToggle, inbox []common.Address) (event.Subscription, error) {

	var inboxRule []interface{}
	for _, inboxItem := range inbox {
		inboxRule = append(inboxRule, inboxItem)
	}

	logs, sub, err := _BridgeTester.contract.WatchLogs(opts, "InboxToggle", inboxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeTesterInboxToggle)
				if err := _BridgeTester.contract.UnpackLog(event, "InboxToggle", log); err != nil {
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
func (_BridgeTester *BridgeTesterFilterer) ParseInboxToggle(log types.Log) (*BridgeTesterInboxToggle, error) {
	event := new(BridgeTesterInboxToggle)
	if err := _BridgeTester.contract.UnpackLog(event, "InboxToggle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeTesterInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the BridgeTester contract.
type BridgeTesterInitializedIterator struct {
	Event *BridgeTesterInitialized // Event containing the contract specifics and raw log

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
func (it *BridgeTesterInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeTesterInitialized)
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
		it.Event = new(BridgeTesterInitialized)
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
func (it *BridgeTesterInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeTesterInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeTesterInitialized represents a Initialized event raised by the BridgeTester contract.
type BridgeTesterInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_BridgeTester *BridgeTesterFilterer) FilterInitialized(opts *bind.FilterOpts) (*BridgeTesterInitializedIterator, error) {

	logs, sub, err := _BridgeTester.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &BridgeTesterInitializedIterator{contract: _BridgeTester.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_BridgeTester *BridgeTesterFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *BridgeTesterInitialized) (event.Subscription, error) {

	logs, sub, err := _BridgeTester.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeTesterInitialized)
				if err := _BridgeTester.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_BridgeTester *BridgeTesterFilterer) ParseInitialized(log types.Log) (*BridgeTesterInitialized, error) {
	event := new(BridgeTesterInitialized)
	if err := _BridgeTester.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeTesterMessageDeliveredIterator is returned from FilterMessageDelivered and is used to iterate over the raw logs and unpacked data for MessageDelivered events raised by the BridgeTester contract.
type BridgeTesterMessageDeliveredIterator struct {
	Event *BridgeTesterMessageDelivered // Event containing the contract specifics and raw log

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
func (it *BridgeTesterMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeTesterMessageDelivered)
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
		it.Event = new(BridgeTesterMessageDelivered)
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
func (it *BridgeTesterMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeTesterMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeTesterMessageDelivered represents a MessageDelivered event raised by the BridgeTester contract.
type BridgeTesterMessageDelivered struct {
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
func (_BridgeTester *BridgeTesterFilterer) FilterMessageDelivered(opts *bind.FilterOpts, messageIndex []*big.Int, beforeInboxAcc [][32]byte) (*BridgeTesterMessageDeliveredIterator, error) {

	var messageIndexRule []interface{}
	for _, messageIndexItem := range messageIndex {
		messageIndexRule = append(messageIndexRule, messageIndexItem)
	}
	var beforeInboxAccRule []interface{}
	for _, beforeInboxAccItem := range beforeInboxAcc {
		beforeInboxAccRule = append(beforeInboxAccRule, beforeInboxAccItem)
	}

	logs, sub, err := _BridgeTester.contract.FilterLogs(opts, "MessageDelivered", messageIndexRule, beforeInboxAccRule)
	if err != nil {
		return nil, err
	}
	return &BridgeTesterMessageDeliveredIterator{contract: _BridgeTester.contract, event: "MessageDelivered", logs: logs, sub: sub}, nil
}

// WatchMessageDelivered is a free log subscription operation binding the contract event 0x5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1.
//
// Solidity: event MessageDelivered(uint256 indexed messageIndex, bytes32 indexed beforeInboxAcc, address inbox, uint8 kind, address sender, bytes32 messageDataHash, uint256 baseFeeL1, uint64 timestamp)
func (_BridgeTester *BridgeTesterFilterer) WatchMessageDelivered(opts *bind.WatchOpts, sink chan<- *BridgeTesterMessageDelivered, messageIndex []*big.Int, beforeInboxAcc [][32]byte) (event.Subscription, error) {

	var messageIndexRule []interface{}
	for _, messageIndexItem := range messageIndex {
		messageIndexRule = append(messageIndexRule, messageIndexItem)
	}
	var beforeInboxAccRule []interface{}
	for _, beforeInboxAccItem := range beforeInboxAcc {
		beforeInboxAccRule = append(beforeInboxAccRule, beforeInboxAccItem)
	}

	logs, sub, err := _BridgeTester.contract.WatchLogs(opts, "MessageDelivered", messageIndexRule, beforeInboxAccRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeTesterMessageDelivered)
				if err := _BridgeTester.contract.UnpackLog(event, "MessageDelivered", log); err != nil {
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
func (_BridgeTester *BridgeTesterFilterer) ParseMessageDelivered(log types.Log) (*BridgeTesterMessageDelivered, error) {
	event := new(BridgeTesterMessageDelivered)
	if err := _BridgeTester.contract.UnpackLog(event, "MessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeTesterOutboxToggleIterator is returned from FilterOutboxToggle and is used to iterate over the raw logs and unpacked data for OutboxToggle events raised by the BridgeTester contract.
type BridgeTesterOutboxToggleIterator struct {
	Event *BridgeTesterOutboxToggle // Event containing the contract specifics and raw log

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
func (it *BridgeTesterOutboxToggleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeTesterOutboxToggle)
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
		it.Event = new(BridgeTesterOutboxToggle)
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
func (it *BridgeTesterOutboxToggleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeTesterOutboxToggleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeTesterOutboxToggle represents a OutboxToggle event raised by the BridgeTester contract.
type BridgeTesterOutboxToggle struct {
	Outbox  common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOutboxToggle is a free log retrieval operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_BridgeTester *BridgeTesterFilterer) FilterOutboxToggle(opts *bind.FilterOpts, outbox []common.Address) (*BridgeTesterOutboxToggleIterator, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}

	logs, sub, err := _BridgeTester.contract.FilterLogs(opts, "OutboxToggle", outboxRule)
	if err != nil {
		return nil, err
	}
	return &BridgeTesterOutboxToggleIterator{contract: _BridgeTester.contract, event: "OutboxToggle", logs: logs, sub: sub}, nil
}

// WatchOutboxToggle is a free log subscription operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_BridgeTester *BridgeTesterFilterer) WatchOutboxToggle(opts *bind.WatchOpts, sink chan<- *BridgeTesterOutboxToggle, outbox []common.Address) (event.Subscription, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}

	logs, sub, err := _BridgeTester.contract.WatchLogs(opts, "OutboxToggle", outboxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeTesterOutboxToggle)
				if err := _BridgeTester.contract.UnpackLog(event, "OutboxToggle", log); err != nil {
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
func (_BridgeTester *BridgeTesterFilterer) ParseOutboxToggle(log types.Log) (*BridgeTesterOutboxToggle, error) {
	event := new(BridgeTesterOutboxToggle)
	if err := _BridgeTester.contract.UnpackLog(event, "OutboxToggle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeTesterRollupUpdatedIterator is returned from FilterRollupUpdated and is used to iterate over the raw logs and unpacked data for RollupUpdated events raised by the BridgeTester contract.
type BridgeTesterRollupUpdatedIterator struct {
	Event *BridgeTesterRollupUpdated // Event containing the contract specifics and raw log

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
func (it *BridgeTesterRollupUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeTesterRollupUpdated)
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
		it.Event = new(BridgeTesterRollupUpdated)
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
func (it *BridgeTesterRollupUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeTesterRollupUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeTesterRollupUpdated represents a RollupUpdated event raised by the BridgeTester contract.
type BridgeTesterRollupUpdated struct {
	Rollup common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterRollupUpdated is a free log retrieval operation binding the contract event 0xae1f5aa15f6ff844896347ceca2a3c24c8d3a27785efdeacd581a0a95172784a.
//
// Solidity: event RollupUpdated(address rollup)
func (_BridgeTester *BridgeTesterFilterer) FilterRollupUpdated(opts *bind.FilterOpts) (*BridgeTesterRollupUpdatedIterator, error) {

	logs, sub, err := _BridgeTester.contract.FilterLogs(opts, "RollupUpdated")
	if err != nil {
		return nil, err
	}
	return &BridgeTesterRollupUpdatedIterator{contract: _BridgeTester.contract, event: "RollupUpdated", logs: logs, sub: sub}, nil
}

// WatchRollupUpdated is a free log subscription operation binding the contract event 0xae1f5aa15f6ff844896347ceca2a3c24c8d3a27785efdeacd581a0a95172784a.
//
// Solidity: event RollupUpdated(address rollup)
func (_BridgeTester *BridgeTesterFilterer) WatchRollupUpdated(opts *bind.WatchOpts, sink chan<- *BridgeTesterRollupUpdated) (event.Subscription, error) {

	logs, sub, err := _BridgeTester.contract.WatchLogs(opts, "RollupUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeTesterRollupUpdated)
				if err := _BridgeTester.contract.UnpackLog(event, "RollupUpdated", log); err != nil {
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
func (_BridgeTester *BridgeTesterFilterer) ParseRollupUpdated(log types.Log) (*BridgeTesterRollupUpdated, error) {
	event := new(BridgeTesterRollupUpdated)
	if err := _BridgeTester.contract.UnpackLog(event, "RollupUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeTesterSequencerInboxUpdatedIterator is returned from FilterSequencerInboxUpdated and is used to iterate over the raw logs and unpacked data for SequencerInboxUpdated events raised by the BridgeTester contract.
type BridgeTesterSequencerInboxUpdatedIterator struct {
	Event *BridgeTesterSequencerInboxUpdated // Event containing the contract specifics and raw log

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
func (it *BridgeTesterSequencerInboxUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeTesterSequencerInboxUpdated)
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
		it.Event = new(BridgeTesterSequencerInboxUpdated)
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
func (it *BridgeTesterSequencerInboxUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeTesterSequencerInboxUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeTesterSequencerInboxUpdated represents a SequencerInboxUpdated event raised by the BridgeTester contract.
type BridgeTesterSequencerInboxUpdated struct {
	NewSequencerInbox common.Address
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterSequencerInboxUpdated is a free log retrieval operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_BridgeTester *BridgeTesterFilterer) FilterSequencerInboxUpdated(opts *bind.FilterOpts) (*BridgeTesterSequencerInboxUpdatedIterator, error) {

	logs, sub, err := _BridgeTester.contract.FilterLogs(opts, "SequencerInboxUpdated")
	if err != nil {
		return nil, err
	}
	return &BridgeTesterSequencerInboxUpdatedIterator{contract: _BridgeTester.contract, event: "SequencerInboxUpdated", logs: logs, sub: sub}, nil
}

// WatchSequencerInboxUpdated is a free log subscription operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_BridgeTester *BridgeTesterFilterer) WatchSequencerInboxUpdated(opts *bind.WatchOpts, sink chan<- *BridgeTesterSequencerInboxUpdated) (event.Subscription, error) {

	logs, sub, err := _BridgeTester.contract.WatchLogs(opts, "SequencerInboxUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeTesterSequencerInboxUpdated)
				if err := _BridgeTester.contract.UnpackLog(event, "SequencerInboxUpdated", log); err != nil {
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
func (_BridgeTester *BridgeTesterFilterer) ParseSequencerInboxUpdated(log types.Log) (*BridgeTesterSequencerInboxUpdated, error) {
	event := new(BridgeTesterSequencerInboxUpdated)
	if err := _BridgeTester.contract.UnpackLog(event, "SequencerInboxUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CryptographyPrimitivesTesterMetaData contains all meta data concerning the CryptographyPrimitivesTester contract.
var CryptographyPrimitivesTesterMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256[25]\",\"name\":\"input\",\"type\":\"uint256[25]\"}],\"name\":\"keccakF\",\"outputs\":[{\"internalType\":\"uint256[25]\",\"name\":\"\",\"type\":\"uint256[25]\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[2]\",\"name\":\"inputChunk\",\"type\":\"bytes32[2]\"},{\"internalType\":\"bytes32\",\"name\":\"hashState\",\"type\":\"bytes32\"}],\"name\":\"sha256Block\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x61173b61003a600b82828239805160001a60731461002d57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600436106100405760003560e01c8063ac90ed4614610045578063e479f5321461006e575b600080fd5b610058610053366004611555565b61008f565b60405161006591906115e2565b60405180910390f35b61008161007c366004611614565b6100a6565b604051908152602001610065565b6100976114da565b6100a0826100e9565b92915050565b60006100e26040518060400160405280856000600281106100c9576100c96116a7565b6020908102919091015182528681015191015283610d35565b9392505050565b6100f16114da565b6100f96114f9565b6101016114f9565b6101096114da565b600060405180610300016040528060018152602001618082815260200167800000000000808a8152602001678000000080008000815260200161808b81526020016380000001815260200167800000008000808181526020016780000000000080098152602001608a81526020016088815260200163800080098152602001638000000a8152602001638000808b815260200167800000000000008b8152602001678000000000008089815260200167800000000000800381526020016780000000000080028152602001678000000000000080815260200161800a815260200167800000008000000a81526020016780000000800080818152602001678000000000008080815260200163800000018152602001678000000080008008815250905060005b6018811015610d2a576080878101516060808a01516040808c01516020808e01518e511890911890921890931889526101208b01516101008c015160e08d015160c08e015160a08f0151181818189089018190526101c08b01516101a08c01516101808d01516101608e01516101408f0151181818189289019283526102608b01516102408c01516102208d01516102008e01516101e08f015118181818918901919091526103008a01516102e08b01516102c08c01516102a08d01516102808e01511818181892880183905267ffffffffffffffff6002820216678000000000000000918290041790921886525104856002602002015160020267ffffffffffffffff16178560006020020151188460016020020152678000000000000000856003602002015181610364576103646116bd565b04856003602002015160020267ffffffffffffffff161785600160200201511884600260200201526780000000000000008560046020020151816103aa576103aa6116bd565b04856004602002015160020267ffffffffffffffff1617856002600581106103d4576103d46116a7565b6020020151186060850152845167800000000000000090865160608089015193909204600290910267ffffffffffffffff1617909118608086810191825286518a5118808b5287516020808d018051909218825289516040808f0180519092189091528a518e8801805190911890528a51948e0180519095189094528901805160a08e0180519091189052805160c08e0180519091189052805160e08e018051909118905280516101008e0180519091189052516101208d018051909118905291880180516101408d018051909118905280516101608d018051909118905280516101808d018051909118905280516101a08d0180519091189052516101c08c018051909118905292870180516101e08c018051909118905280516102008c018051909118905280516102208c018051909118905280516102408c0180519091189052516102608b018051909118905281516102808b018051909118905281516102a08b018051909118905281516102c08b018051909118905281516102e08b018051909118905290516103008a018051909118905290845251631000000090602089015167ffffffffffffffff6410000000009091021691900417610100840152604087015167200000000000000090604089015167ffffffffffffffff6008909102169190041761016084015260608701516280000090606089015167ffffffffffffffff65020000000000909102169190041761026084015260808701516540000000000090608089015167ffffffffffffffff6204000090910216919004176102c084015260a08701516780000000000000009004876005602002015160020267ffffffffffffffff161783600260198110610656576106566116a7565b602002015260c08701516210000081046510000000000090910267ffffffffffffffff9081169190911760a085015260e0880151664000000000000081046104009091028216176101a08501526101008801516208000081046520000000000090910282161761020085015261012088015160048082029092166740000000000000009091041761030085015261014088015161014089015167ffffffffffffffff674000000000000000909102169190041760808401526101608701516704000000000000009061016089015167ffffffffffffffff6040909102169190041760e0840152610180870151622000009061018089015167ffffffffffffffff6508000000000090910216919004176101408401526101a08701516602000000000000906101a089015167ffffffffffffffff61800090910216919004176102408401526101c08701516008906101c089015167ffffffffffffffff67200000000000000090910216919004176102a08401526101e0870151641000000000906101e089015167ffffffffffffffff6310000000909102169190041760208401526102008088015161020089015167ffffffffffffffff668000000000000090910216919004176101208401526102208701516480000000009061022089015167ffffffffffffffff63020000009091021691900417610180840152610240870151650800000000009061024089015167ffffffffffffffff6220000090910216919004176101e08401526102608701516101009061026089015167ffffffffffffffff67010000000000000090910216919004176102e08401526102808701516420000000009061028089015167ffffffffffffffff6308000000909102169190041760608401526102a087015165100000000000906102a089015167ffffffffffffffff62100000909102169190041760c08401526102c08701516302000000906102c089015167ffffffffffffffff64800000000090910216919004176101c08401526102e0870151670100000000000000906102e089015167ffffffffffffffff61010090910216919004176102208401526103008701516604000000000000900487601860200201516140000267ffffffffffffffff1617836014602002015282600a602002015183600560200201511916836000602002015118876000602002015282600b602002015183600660200201511916836001602002015118876001602002015282600c602002015183600760200201511916836002602002015118876002602002015282600d602002015183600860200201511916836003602002015118876003602002015282600e602002015183600960200201511916836004602002015118876004602002015282600f602002015183600a602002015119168360056020020151188760056020020152826010602002015183600b602002015119168360066020020151188760066020020152826011602002015183600c602002015119168360076020020151188760076020020152826012602002015183600d602002015119168360086020020151188760086020020152826013602002015183600e602002015119168360096020020151188760096020020152826014602002015183600f6020020151191683600a60200201511887600a602002015282601560200201518360106020020151191683600b60200201511887600b602002015282601660200201518360116020020151191683600c60200201511887600c602002015282601760200201518360126020020151191683600d60200201511887600d602002015282601860200201518360136020020151191683600e60200201511887600e602002015282600060200201518360146020020151191683600f60200201511887600f6020020152826001602002015183601560200201511916836010602002015118876010602002015282600260200201518360166020020151191683601160200201511887601160200201528260036020020151836017602002015119168360126020020151188760126020020152826004602002015183601860200201511916836013602002015118876013602002015282600560200201518360006020020151191683601460200201511887601460200201528260066020020151836001602002015119168360156020020151188760156020020152826007602002015183600260200201511916836016602002015118876016602002015282600860200201518360036020020151191683601760200201511887601760200201528260096020020151836004602002015119168360186020020151188760186020020152818160188110610d1857610d186116a7565b6020020151875118875260010161022f565b509495945050505050565b604080516108008101825263428a2f9881526371374491602082015263b5c0fbcf9181019190915263e9b5dba56060820152633956c25b60808201526359f111f160a082015263923f82a460c082015263ab1c5ed560e082015263d807aa986101008201526312835b0161012082015263243185be61014082015263550c7dc36101608201526372be5d746101808201526380deb1fe6101a0820152639bdc06a76101c082015263c19bf1746101e082015263e49b69c161020082015263efbe4786610220820152630fc19dc661024082015263240ca1cc610260820152632de92c6f610280820152634a7484aa6102a0820152635cb0a9dc6102c08201526376f988da6102e082015263983e515261030082015263a831c66d61032082015263b00327c861034082015263bf597fc761036082015263c6e00bf361038082015263d5a791476103a08201526306ca63516103c082015263142929676103e08201526327b70a85610400820152632e1b2138610420820152634d2c6dfc6104408201526353380d1361046082015263650a735461048082015263766a0abb6104a08201526381c2c92e6104c08201526392722c856104e082015263a2bfe8a161050082015263a81a664b61052082015263c24b8b7061054082015263c76c51a361056082015263d192e81961058082015263d69906246105a082015263f40e35856105c082015263106aa0706105e08201526319a4c116610600820152631e376c08610620820152632748774c6106408201526334b0bcb561066082015263391c0cb3610680820152634ed8aa4a6106a0820152635b9cca4f6106c082015263682e6ff36106e082015263748f82ee6107008201526378a5636f6107208201526384c87814610740820152638cc702086107608201526390befffa61078082015263a4506ceb6107a082015263bef9a3f76107c082015263c67178f26107e0820152600090611002611517565b60005b60088163ffffffff16101561109b5763ffffffff6020820260e003168660006020020151901c828263ffffffff1660408110611043576110436116a7565b63ffffffff92831660209182029290920191909152820260e003168660016020020151901c828260080163ffffffff1660408110611083576110836116a7565b63ffffffff9092166020929092020152600101611005565b5060106000805b60408363ffffffff16101561122d57600384600f850363ffffffff16604081106110ce576110ce6116a7565b602002015163ffffffff16901c61110585600f860363ffffffff16604081106110f9576110f96116a7565b602002015160126114a5565b61112f86600f870363ffffffff1660408110611123576111236116a7565b602002015160076114a5565b18189150600a846002850363ffffffff1660408110611150576111506116a7565b602002015163ffffffff16901c611187856002860363ffffffff166040811061117b5761117b6116a7565b602002015160136114a5565b6111b1866002870363ffffffff16604081106111a5576111a56116a7565b602002015160116114a5565b1818905080846007850363ffffffff16604081106111d1576111d16116a7565b602002015183866010870363ffffffff16604081106111f2576111f26116a7565b6020020151010101848463ffffffff1660408110611212576112126116a7565b63ffffffff90921660209290920201526001909201916110a2565b611235611536565b600093505b60088463ffffffff16101561128c578360200260e00363ffffffff1688901c818563ffffffff1660088110611271576112716116a7565b63ffffffff909216602092909202015260019093019261123a565b60008060008096505b60408763ffffffff1610156113e85760808401516112b49060196114a5565b60808501516112c490600b6114a5565b60808601516112d49060066114a5565b18189450878763ffffffff16604081106112f0576112f06116a7565b6020020151898863ffffffff166040811061130d5761130d6116a7565b6020020151608086015160a087015160c0880151821916911618878760076020020151010101019250611348846000602002015160166114a5565b845161135590600d6114a5565b85516113629060026114a5565b6040870180516020890180518a5160c08c01805163ffffffff90811660e08f015260a08e018051821690925260808e018051821690925260608e0180518e01821690925280861690915280831690955284811690925280831891909116911618929091189290921881810186810190931687526001999099019897509092509050611295565b600096505b60088763ffffffff161015611442578660200260e00363ffffffff168b901c848863ffffffff1660088110611424576114246116a7565b60200201805163ffffffff92019190911690526001909601956113ed565b60008097505b60088863ffffffff161015611495578760200260e00363ffffffff16858963ffffffff166008811061147c5761147c6116a7565b602002015160019099019863ffffffff16901b17611448565b9c9b505050505050505050505050565b60006114b28260206116d3565b63ffffffff168363ffffffff16901b8263ffffffff168463ffffffff16901c17905092915050565b6040518061032001604052806019906020820280368337509192915050565b6040518060a001604052806005906020820280368337509192915050565b6040518061080001604052806040906020820280368337509192915050565b6040518061010001604052806008906020820280368337509192915050565b600061032080838503121561156957600080fd5b83601f84011261157857600080fd5b60405181810181811067ffffffffffffffff821117156115a857634e487b7160e01b600052604160045260246000fd5b6040529083019080858311156115bd57600080fd5b845b838110156115d75780358252602091820191016115bf565b509095945050505050565b6103208101818360005b601981101561160b5781518352602092830192909101906001016115ec565b50505092915050565b6000806060838503121561162757600080fd5b83601f84011261163657600080fd5b6040516040810181811067ffffffffffffffff8211171561166757634e487b7160e01b600052604160045260246000fd5b806040525080604085018681111561167e57600080fd5b855b81811015611698578035835260209283019201611680565b50919691359550909350505050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601260045260246000fd5b63ffffffff8281168282160390808211156116fe57634e487b7160e01b600052601160045260246000fd5b509291505056fea264697066735822122099b67cb168dcef4e82d8fbdadb227e1cb3d7d28070f5b9a4a7ec76385d134b9d64736f6c63430008110033",
}

// CryptographyPrimitivesTesterABI is the input ABI used to generate the binding from.
// Deprecated: Use CryptographyPrimitivesTesterMetaData.ABI instead.
var CryptographyPrimitivesTesterABI = CryptographyPrimitivesTesterMetaData.ABI

// CryptographyPrimitivesTesterBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CryptographyPrimitivesTesterMetaData.Bin instead.
var CryptographyPrimitivesTesterBin = CryptographyPrimitivesTesterMetaData.Bin

// DeployCryptographyPrimitivesTester deploys a new Ethereum contract, binding an instance of CryptographyPrimitivesTester to it.
func DeployCryptographyPrimitivesTester(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *CryptographyPrimitivesTester, error) {
	parsed, err := CryptographyPrimitivesTesterMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CryptographyPrimitivesTesterBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &CryptographyPrimitivesTester{CryptographyPrimitivesTesterCaller: CryptographyPrimitivesTesterCaller{contract: contract}, CryptographyPrimitivesTesterTransactor: CryptographyPrimitivesTesterTransactor{contract: contract}, CryptographyPrimitivesTesterFilterer: CryptographyPrimitivesTesterFilterer{contract: contract}}, nil
}

// CryptographyPrimitivesTester is an auto generated Go binding around an Ethereum contract.
type CryptographyPrimitivesTester struct {
	CryptographyPrimitivesTesterCaller     // Read-only binding to the contract
	CryptographyPrimitivesTesterTransactor // Write-only binding to the contract
	CryptographyPrimitivesTesterFilterer   // Log filterer for contract events
}

// CryptographyPrimitivesTesterCaller is an auto generated read-only Go binding around an Ethereum contract.
type CryptographyPrimitivesTesterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptographyPrimitivesTesterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CryptographyPrimitivesTesterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptographyPrimitivesTesterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CryptographyPrimitivesTesterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CryptographyPrimitivesTesterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CryptographyPrimitivesTesterSession struct {
	Contract     *CryptographyPrimitivesTester // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                 // Call options to use throughout this session
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// CryptographyPrimitivesTesterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CryptographyPrimitivesTesterCallerSession struct {
	Contract *CryptographyPrimitivesTesterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                       // Call options to use throughout this session
}

// CryptographyPrimitivesTesterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CryptographyPrimitivesTesterTransactorSession struct {
	Contract     *CryptographyPrimitivesTesterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                       // Transaction auth options to use throughout this session
}

// CryptographyPrimitivesTesterRaw is an auto generated low-level Go binding around an Ethereum contract.
type CryptographyPrimitivesTesterRaw struct {
	Contract *CryptographyPrimitivesTester // Generic contract binding to access the raw methods on
}

// CryptographyPrimitivesTesterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CryptographyPrimitivesTesterCallerRaw struct {
	Contract *CryptographyPrimitivesTesterCaller // Generic read-only contract binding to access the raw methods on
}

// CryptographyPrimitivesTesterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CryptographyPrimitivesTesterTransactorRaw struct {
	Contract *CryptographyPrimitivesTesterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCryptographyPrimitivesTester creates a new instance of CryptographyPrimitivesTester, bound to a specific deployed contract.
func NewCryptographyPrimitivesTester(address common.Address, backend bind.ContractBackend) (*CryptographyPrimitivesTester, error) {
	contract, err := bindCryptographyPrimitivesTester(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CryptographyPrimitivesTester{CryptographyPrimitivesTesterCaller: CryptographyPrimitivesTesterCaller{contract: contract}, CryptographyPrimitivesTesterTransactor: CryptographyPrimitivesTesterTransactor{contract: contract}, CryptographyPrimitivesTesterFilterer: CryptographyPrimitivesTesterFilterer{contract: contract}}, nil
}

// NewCryptographyPrimitivesTesterCaller creates a new read-only instance of CryptographyPrimitivesTester, bound to a specific deployed contract.
func NewCryptographyPrimitivesTesterCaller(address common.Address, caller bind.ContractCaller) (*CryptographyPrimitivesTesterCaller, error) {
	contract, err := bindCryptographyPrimitivesTester(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CryptographyPrimitivesTesterCaller{contract: contract}, nil
}

// NewCryptographyPrimitivesTesterTransactor creates a new write-only instance of CryptographyPrimitivesTester, bound to a specific deployed contract.
func NewCryptographyPrimitivesTesterTransactor(address common.Address, transactor bind.ContractTransactor) (*CryptographyPrimitivesTesterTransactor, error) {
	contract, err := bindCryptographyPrimitivesTester(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CryptographyPrimitivesTesterTransactor{contract: contract}, nil
}

// NewCryptographyPrimitivesTesterFilterer creates a new log filterer instance of CryptographyPrimitivesTester, bound to a specific deployed contract.
func NewCryptographyPrimitivesTesterFilterer(address common.Address, filterer bind.ContractFilterer) (*CryptographyPrimitivesTesterFilterer, error) {
	contract, err := bindCryptographyPrimitivesTester(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CryptographyPrimitivesTesterFilterer{contract: contract}, nil
}

// bindCryptographyPrimitivesTester binds a generic wrapper to an already deployed contract.
func bindCryptographyPrimitivesTester(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := CryptographyPrimitivesTesterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CryptographyPrimitivesTester.Contract.CryptographyPrimitivesTesterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptographyPrimitivesTester.Contract.CryptographyPrimitivesTesterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CryptographyPrimitivesTester.Contract.CryptographyPrimitivesTesterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CryptographyPrimitivesTester.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CryptographyPrimitivesTester.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CryptographyPrimitivesTester.Contract.contract.Transact(opts, method, params...)
}

// KeccakF is a free data retrieval call binding the contract method 0xac90ed46.
//
// Solidity: function keccakF(uint256[25] input) pure returns(uint256[25])
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterCaller) KeccakF(opts *bind.CallOpts, input [25]*big.Int) ([25]*big.Int, error) {
	var out []interface{}
	err := _CryptographyPrimitivesTester.contract.Call(opts, &out, "keccakF", input)

	if err != nil {
		return *new([25]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([25]*big.Int)).(*[25]*big.Int)

	return out0, err

}

// KeccakF is a free data retrieval call binding the contract method 0xac90ed46.
//
// Solidity: function keccakF(uint256[25] input) pure returns(uint256[25])
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterSession) KeccakF(input [25]*big.Int) ([25]*big.Int, error) {
	return _CryptographyPrimitivesTester.Contract.KeccakF(&_CryptographyPrimitivesTester.CallOpts, input)
}

// KeccakF is a free data retrieval call binding the contract method 0xac90ed46.
//
// Solidity: function keccakF(uint256[25] input) pure returns(uint256[25])
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterCallerSession) KeccakF(input [25]*big.Int) ([25]*big.Int, error) {
	return _CryptographyPrimitivesTester.Contract.KeccakF(&_CryptographyPrimitivesTester.CallOpts, input)
}

// Sha256Block is a free data retrieval call binding the contract method 0xe479f532.
//
// Solidity: function sha256Block(bytes32[2] inputChunk, bytes32 hashState) pure returns(bytes32)
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterCaller) Sha256Block(opts *bind.CallOpts, inputChunk [2][32]byte, hashState [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _CryptographyPrimitivesTester.contract.Call(opts, &out, "sha256Block", inputChunk, hashState)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Sha256Block is a free data retrieval call binding the contract method 0xe479f532.
//
// Solidity: function sha256Block(bytes32[2] inputChunk, bytes32 hashState) pure returns(bytes32)
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterSession) Sha256Block(inputChunk [2][32]byte, hashState [32]byte) ([32]byte, error) {
	return _CryptographyPrimitivesTester.Contract.Sha256Block(&_CryptographyPrimitivesTester.CallOpts, inputChunk, hashState)
}

// Sha256Block is a free data retrieval call binding the contract method 0xe479f532.
//
// Solidity: function sha256Block(bytes32[2] inputChunk, bytes32 hashState) pure returns(bytes32)
func (_CryptographyPrimitivesTester *CryptographyPrimitivesTesterCallerSession) Sha256Block(inputChunk [2][32]byte, hashState [32]byte) ([32]byte, error) {
	return _CryptographyPrimitivesTester.Contract.Sha256Block(&_CryptographyPrimitivesTester.CallOpts, inputChunk, hashState)
}

// EthVaultMetaData contains all meta data concerning the EthVault contract.
var EthVaultMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"justRevert\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_version\",\"type\":\"uint256\"}],\"name\":\"setVersion\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040526000805534801561001457600080fd5b50610127806100246000396000f3fe60806040526004361060305760003560e01c8063408def1e14603557806350b23fd214604757806354fd4d5014604d575b600080fd5b6045604036600460d9565b600055565b005b60456073565b348015605857600080fd5b50606160005481565b60405190815260200160405180910390f35b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600360248201527f6279650000000000000000000000000000000000000000000000000000000000604482015260640160405180910390fd5b60006020828403121560ea57600080fd5b503591905056fea2646970667358221220e1d9ce9e406acb30a5ad7b5c6451d68e3651f071cba9865d6f61ca4a69e77a6864736f6c63430008110033",
}

// EthVaultABI is the input ABI used to generate the binding from.
// Deprecated: Use EthVaultMetaData.ABI instead.
var EthVaultABI = EthVaultMetaData.ABI

// EthVaultBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use EthVaultMetaData.Bin instead.
var EthVaultBin = EthVaultMetaData.Bin

// DeployEthVault deploys a new Ethereum contract, binding an instance of EthVault to it.
func DeployEthVault(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *EthVault, error) {
	parsed, err := EthVaultMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(EthVaultBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &EthVault{EthVaultCaller: EthVaultCaller{contract: contract}, EthVaultTransactor: EthVaultTransactor{contract: contract}, EthVaultFilterer: EthVaultFilterer{contract: contract}}, nil
}

// EthVault is an auto generated Go binding around an Ethereum contract.
type EthVault struct {
	EthVaultCaller     // Read-only binding to the contract
	EthVaultTransactor // Write-only binding to the contract
	EthVaultFilterer   // Log filterer for contract events
}

// EthVaultCaller is an auto generated read-only Go binding around an Ethereum contract.
type EthVaultCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EthVaultTransactor is an auto generated write-only Go binding around an Ethereum contract.
type EthVaultTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EthVaultFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type EthVaultFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EthVaultSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type EthVaultSession struct {
	Contract     *EthVault         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// EthVaultCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type EthVaultCallerSession struct {
	Contract *EthVaultCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// EthVaultTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type EthVaultTransactorSession struct {
	Contract     *EthVaultTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// EthVaultRaw is an auto generated low-level Go binding around an Ethereum contract.
type EthVaultRaw struct {
	Contract *EthVault // Generic contract binding to access the raw methods on
}

// EthVaultCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type EthVaultCallerRaw struct {
	Contract *EthVaultCaller // Generic read-only contract binding to access the raw methods on
}

// EthVaultTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type EthVaultTransactorRaw struct {
	Contract *EthVaultTransactor // Generic write-only contract binding to access the raw methods on
}

// NewEthVault creates a new instance of EthVault, bound to a specific deployed contract.
func NewEthVault(address common.Address, backend bind.ContractBackend) (*EthVault, error) {
	contract, err := bindEthVault(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &EthVault{EthVaultCaller: EthVaultCaller{contract: contract}, EthVaultTransactor: EthVaultTransactor{contract: contract}, EthVaultFilterer: EthVaultFilterer{contract: contract}}, nil
}

// NewEthVaultCaller creates a new read-only instance of EthVault, bound to a specific deployed contract.
func NewEthVaultCaller(address common.Address, caller bind.ContractCaller) (*EthVaultCaller, error) {
	contract, err := bindEthVault(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &EthVaultCaller{contract: contract}, nil
}

// NewEthVaultTransactor creates a new write-only instance of EthVault, bound to a specific deployed contract.
func NewEthVaultTransactor(address common.Address, transactor bind.ContractTransactor) (*EthVaultTransactor, error) {
	contract, err := bindEthVault(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &EthVaultTransactor{contract: contract}, nil
}

// NewEthVaultFilterer creates a new log filterer instance of EthVault, bound to a specific deployed contract.
func NewEthVaultFilterer(address common.Address, filterer bind.ContractFilterer) (*EthVaultFilterer, error) {
	contract, err := bindEthVault(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &EthVaultFilterer{contract: contract}, nil
}

// bindEthVault binds a generic wrapper to an already deployed contract.
func bindEthVault(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := EthVaultMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EthVault *EthVaultRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EthVault.Contract.EthVaultCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EthVault *EthVaultRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EthVault.Contract.EthVaultTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EthVault *EthVaultRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EthVault.Contract.EthVaultTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EthVault *EthVaultCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EthVault.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EthVault *EthVaultTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EthVault.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EthVault *EthVaultTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EthVault.Contract.contract.Transact(opts, method, params...)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(uint256)
func (_EthVault *EthVaultCaller) Version(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _EthVault.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(uint256)
func (_EthVault *EthVaultSession) Version() (*big.Int, error) {
	return _EthVault.Contract.Version(&_EthVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(uint256)
func (_EthVault *EthVaultCallerSession) Version() (*big.Int, error) {
	return _EthVault.Contract.Version(&_EthVault.CallOpts)
}

// JustRevert is a paid mutator transaction binding the contract method 0x50b23fd2.
//
// Solidity: function justRevert() payable returns()
func (_EthVault *EthVaultTransactor) JustRevert(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EthVault.contract.Transact(opts, "justRevert")
}

// JustRevert is a paid mutator transaction binding the contract method 0x50b23fd2.
//
// Solidity: function justRevert() payable returns()
func (_EthVault *EthVaultSession) JustRevert() (*types.Transaction, error) {
	return _EthVault.Contract.JustRevert(&_EthVault.TransactOpts)
}

// JustRevert is a paid mutator transaction binding the contract method 0x50b23fd2.
//
// Solidity: function justRevert() payable returns()
func (_EthVault *EthVaultTransactorSession) JustRevert() (*types.Transaction, error) {
	return _EthVault.Contract.JustRevert(&_EthVault.TransactOpts)
}

// SetVersion is a paid mutator transaction binding the contract method 0x408def1e.
//
// Solidity: function setVersion(uint256 _version) payable returns()
func (_EthVault *EthVaultTransactor) SetVersion(opts *bind.TransactOpts, _version *big.Int) (*types.Transaction, error) {
	return _EthVault.contract.Transact(opts, "setVersion", _version)
}

// SetVersion is a paid mutator transaction binding the contract method 0x408def1e.
//
// Solidity: function setVersion(uint256 _version) payable returns()
func (_EthVault *EthVaultSession) SetVersion(_version *big.Int) (*types.Transaction, error) {
	return _EthVault.Contract.SetVersion(&_EthVault.TransactOpts, _version)
}

// SetVersion is a paid mutator transaction binding the contract method 0x408def1e.
//
// Solidity: function setVersion(uint256 _version) payable returns()
func (_EthVault *EthVaultTransactorSession) SetVersion(_version *big.Int) (*types.Transaction, error) {
	return _EthVault.Contract.SetVersion(&_EthVault.TransactOpts, _version)
}

// MessageTesterMetaData contains all meta data concerning the MessageTester contract.
var MessageTesterMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"inbox\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"message\",\"type\":\"bytes32\"}],\"name\":\"accumulateInboxMessage\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"messageType\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"blockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"inboxSeqNum\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasPriceL1\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"messageHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610267806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80638f3c79c01461003b578063bf00905214610087575b600080fd5b610075610049366004610160565b604080516020808201949094528082019290925280518083038201815260609092019052805191012090565b60405190815260200160405180910390f35b61007561009536600461019f565b6040805160f89890981b7fff00000000000000000000000000000000000000000000000000000000000000166020808a019190915260609790971b7fffffffffffffffffffffffffffffffffffffffff00000000000000000000000016602189015260c095861b7fffffffffffffffff00000000000000000000000000000000000000000000000090811660358a01529490951b909316603d870152604586019190915260658501526085808501919091528151808503909101815260a59093019052815191012090565b6000806040838503121561017357600080fd5b50508035926020909101359150565b803567ffffffffffffffff8116811461019a57600080fd5b919050565b600080600080600080600060e0888a0312156101ba57600080fd5b873560ff811681146101cb57600080fd5b9650602088013573ffffffffffffffffffffffffffffffffffffffff811681146101f457600080fd5b955061020260408901610182565b945061021060608901610182565b9699959850939660808101359560a0820135955060c090910135935091505056fea2646970667358221220b2a2bc5b3c96ebf221b893e18d58d6c2dadab91879606e5c2b786f54a6179cbd64736f6c63430008110033",
}

// MessageTesterABI is the input ABI used to generate the binding from.
// Deprecated: Use MessageTesterMetaData.ABI instead.
var MessageTesterABI = MessageTesterMetaData.ABI

// MessageTesterBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MessageTesterMetaData.Bin instead.
var MessageTesterBin = MessageTesterMetaData.Bin

// DeployMessageTester deploys a new Ethereum contract, binding an instance of MessageTester to it.
func DeployMessageTester(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MessageTester, error) {
	parsed, err := MessageTesterMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MessageTesterBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MessageTester{MessageTesterCaller: MessageTesterCaller{contract: contract}, MessageTesterTransactor: MessageTesterTransactor{contract: contract}, MessageTesterFilterer: MessageTesterFilterer{contract: contract}}, nil
}

// MessageTester is an auto generated Go binding around an Ethereum contract.
type MessageTester struct {
	MessageTesterCaller     // Read-only binding to the contract
	MessageTesterTransactor // Write-only binding to the contract
	MessageTesterFilterer   // Log filterer for contract events
}

// MessageTesterCaller is an auto generated read-only Go binding around an Ethereum contract.
type MessageTesterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MessageTesterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MessageTesterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MessageTesterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MessageTesterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MessageTesterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MessageTesterSession struct {
	Contract     *MessageTester    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MessageTesterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MessageTesterCallerSession struct {
	Contract *MessageTesterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// MessageTesterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MessageTesterTransactorSession struct {
	Contract     *MessageTesterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// MessageTesterRaw is an auto generated low-level Go binding around an Ethereum contract.
type MessageTesterRaw struct {
	Contract *MessageTester // Generic contract binding to access the raw methods on
}

// MessageTesterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MessageTesterCallerRaw struct {
	Contract *MessageTesterCaller // Generic read-only contract binding to access the raw methods on
}

// MessageTesterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MessageTesterTransactorRaw struct {
	Contract *MessageTesterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMessageTester creates a new instance of MessageTester, bound to a specific deployed contract.
func NewMessageTester(address common.Address, backend bind.ContractBackend) (*MessageTester, error) {
	contract, err := bindMessageTester(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MessageTester{MessageTesterCaller: MessageTesterCaller{contract: contract}, MessageTesterTransactor: MessageTesterTransactor{contract: contract}, MessageTesterFilterer: MessageTesterFilterer{contract: contract}}, nil
}

// NewMessageTesterCaller creates a new read-only instance of MessageTester, bound to a specific deployed contract.
func NewMessageTesterCaller(address common.Address, caller bind.ContractCaller) (*MessageTesterCaller, error) {
	contract, err := bindMessageTester(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MessageTesterCaller{contract: contract}, nil
}

// NewMessageTesterTransactor creates a new write-only instance of MessageTester, bound to a specific deployed contract.
func NewMessageTesterTransactor(address common.Address, transactor bind.ContractTransactor) (*MessageTesterTransactor, error) {
	contract, err := bindMessageTester(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MessageTesterTransactor{contract: contract}, nil
}

// NewMessageTesterFilterer creates a new log filterer instance of MessageTester, bound to a specific deployed contract.
func NewMessageTesterFilterer(address common.Address, filterer bind.ContractFilterer) (*MessageTesterFilterer, error) {
	contract, err := bindMessageTester(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MessageTesterFilterer{contract: contract}, nil
}

// bindMessageTester binds a generic wrapper to an already deployed contract.
func bindMessageTester(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MessageTesterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MessageTester *MessageTesterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MessageTester.Contract.MessageTesterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MessageTester *MessageTesterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MessageTester.Contract.MessageTesterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MessageTester *MessageTesterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MessageTester.Contract.MessageTesterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MessageTester *MessageTesterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MessageTester.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MessageTester *MessageTesterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MessageTester.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MessageTester *MessageTesterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MessageTester.Contract.contract.Transact(opts, method, params...)
}

// AccumulateInboxMessage is a free data retrieval call binding the contract method 0x8f3c79c0.
//
// Solidity: function accumulateInboxMessage(bytes32 inbox, bytes32 message) pure returns(bytes32)
func (_MessageTester *MessageTesterCaller) AccumulateInboxMessage(opts *bind.CallOpts, inbox [32]byte, message [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _MessageTester.contract.Call(opts, &out, "accumulateInboxMessage", inbox, message)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// AccumulateInboxMessage is a free data retrieval call binding the contract method 0x8f3c79c0.
//
// Solidity: function accumulateInboxMessage(bytes32 inbox, bytes32 message) pure returns(bytes32)
func (_MessageTester *MessageTesterSession) AccumulateInboxMessage(inbox [32]byte, message [32]byte) ([32]byte, error) {
	return _MessageTester.Contract.AccumulateInboxMessage(&_MessageTester.CallOpts, inbox, message)
}

// AccumulateInboxMessage is a free data retrieval call binding the contract method 0x8f3c79c0.
//
// Solidity: function accumulateInboxMessage(bytes32 inbox, bytes32 message) pure returns(bytes32)
func (_MessageTester *MessageTesterCallerSession) AccumulateInboxMessage(inbox [32]byte, message [32]byte) ([32]byte, error) {
	return _MessageTester.Contract.AccumulateInboxMessage(&_MessageTester.CallOpts, inbox, message)
}

// MessageHash is a free data retrieval call binding the contract method 0xbf009052.
//
// Solidity: function messageHash(uint8 messageType, address sender, uint64 blockNumber, uint64 timestamp, uint256 inboxSeqNum, uint256 gasPriceL1, bytes32 messageDataHash) pure returns(bytes32)
func (_MessageTester *MessageTesterCaller) MessageHash(opts *bind.CallOpts, messageType uint8, sender common.Address, blockNumber uint64, timestamp uint64, inboxSeqNum *big.Int, gasPriceL1 *big.Int, messageDataHash [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _MessageTester.contract.Call(opts, &out, "messageHash", messageType, sender, blockNumber, timestamp, inboxSeqNum, gasPriceL1, messageDataHash)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// MessageHash is a free data retrieval call binding the contract method 0xbf009052.
//
// Solidity: function messageHash(uint8 messageType, address sender, uint64 blockNumber, uint64 timestamp, uint256 inboxSeqNum, uint256 gasPriceL1, bytes32 messageDataHash) pure returns(bytes32)
func (_MessageTester *MessageTesterSession) MessageHash(messageType uint8, sender common.Address, blockNumber uint64, timestamp uint64, inboxSeqNum *big.Int, gasPriceL1 *big.Int, messageDataHash [32]byte) ([32]byte, error) {
	return _MessageTester.Contract.MessageHash(&_MessageTester.CallOpts, messageType, sender, blockNumber, timestamp, inboxSeqNum, gasPriceL1, messageDataHash)
}

// MessageHash is a free data retrieval call binding the contract method 0xbf009052.
//
// Solidity: function messageHash(uint8 messageType, address sender, uint64 blockNumber, uint64 timestamp, uint256 inboxSeqNum, uint256 gasPriceL1, bytes32 messageDataHash) pure returns(bytes32)
func (_MessageTester *MessageTesterCallerSession) MessageHash(messageType uint8, sender common.Address, blockNumber uint64, timestamp uint64, inboxSeqNum *big.Int, gasPriceL1 *big.Int, messageDataHash [32]byte) ([32]byte, error) {
	return _MessageTester.Contract.MessageHash(&_MessageTester.CallOpts, messageType, sender, blockNumber, timestamp, inboxSeqNum, gasPriceL1, messageDataHash)
}

// OutboxWithoutOptTesterMetaData contains all meta data concerning the OutboxWithoutOptTester contract.
var OutboxWithoutOptTesterMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"AlreadyInit\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"AlreadySpent\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BridgeCallFailed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"actualLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxProofLength\",\"type\":\"uint256\"}],\"name\":\"MerkleProofTooLong\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxIndex\",\"type\":\"uint256\"}],\"name\":\"PathNotMinimal\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"proofLength\",\"type\":\"uint256\"}],\"name\":\"ProofTooLong\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"}],\"name\":\"UnknownRoot\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"zero\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"transactionIndex\",\"type\":\"uint256\"}],\"name\":\"OutBoxTransactionExecuted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"l2BlockHash\",\"type\":\"bytes32\"}],\"name\":\"SendRootUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"OUTBOX_VERSION\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l2Timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"calculateItemHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"path\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"item\",\"type\":\"bytes32\"}],\"name\":\"calculateMerkleRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l2Timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"executeTransactionSimulation\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"_bridge\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"isSpent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1BatchNum\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1Block\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1EthBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1OutputId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1Sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1Timestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"postUpgradeInit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"roots\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"spent\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"l2BlockHash\",\"type\":\"bytes32\"}],\"name\":\"updateSendRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a06040523060805234801561001457600080fd5b506080516114d7610030600039600061072e01526114d76000f3fe608060405234801561001057600080fd5b506004361061016b5760003560e01c806395fcea78116100cd578063c4d66de811610081578063cb23bcb511610066578063cb23bcb5146102ed578063d5b5cc2314610300578063e78cea921461031357600080fd5b8063c4d66de8146102ba578063c75184df146102cd57600080fd5b8063a04cee60116100b2578063a04cee6014610276578063ae6dead714610289578063b0f30537146102a957600080fd5b806395fcea78146101a95780639f0c04bf1461026357600080fd5b80635a129efe1161012457806372f2a8c71161010957806372f2a8c71461021157806380648b02146102195780638515bc6a1461023e57600080fd5b80635a129efe146101d65780636ae71f121461020957600080fd5b8063119852711161015557806311985271146101ab578063288e5b10146101b257806346547790146101c557600080fd5b80627436d31461017057806308635a9514610196575b600080fd5b61018361017e366004610e37565b610326565b6040519081526020015b60405180910390f35b6101a96101a4366004610f5e565b610363565b005b6000610183565b6101a96101c0366004611053565b6106d7565b6004546001600160801b0316610183565b6101f96101e43660046110ef565b60026020526000908152604090205460ff1681565b604051901515815260200161018d565b6101a9610724565b600654610183565b6007546001600160a01b03165b6040516001600160a01b03909116815260200161018d565b60045470010000000000000000000000000000000090046001600160801b0316610183565b610183610271366004611108565b6108e0565b6101a9610284366004611197565b610925565b6101836102973660046110ef565b60036020526000908152604090205481565b6005546001600160801b0316610183565b6101a96102c83660046111b9565b610964565b6102d5600281565b6040516001600160801b03909116815260200161018d565b600054610226906001600160a01b031681565b61018361030e3660046110ef565b610a7a565b600154610226906001600160a01b031681565b600061035b84848460405160200161034091815260200190565b60405160208183030381529060405280519060200120610ac5565b949350505050565b6000806103768a8a8a8a8a8a8a8a6108e0565b90506103b88d8d808060200260200160405190810160405280939291908181526020018383602002808284376000920191909152508f9250859150610b809050565b915060008a6001600160a01b03168a6001600160a01b03167f20af7f3bbfe38132b8900ae295cd9c8d1914be7052d061a511f3f728dab189648e60405161040191815260200190565b60405180910390a450600060046040518060a00160405290816000820160009054906101000a90046001600160801b03166001600160801b03166001600160801b031681526020016000820160109054906101000a90046001600160801b03166001600160801b03166001600160801b031681526020016001820160009054906101000a90046001600160801b03166001600160801b03166001600160801b03168152602001600282015481526020016003820160009054906101000a90046001600160a01b03166001600160a01b03166001600160a01b03168152505090506040518060a00160405280896001600160801b03168152602001886001600160801b03168152602001876001600160801b031681526020018381526020018b6001600160a01b0316815250600460008201518160000160006101000a8154816001600160801b0302191690836001600160801b0316021790555060208201518160000160106101000a8154816001600160801b0302191690836001600160801b0316021790555060408201518160010160006101000a8154816001600160801b0302191690836001600160801b031602179055506060820151816002015560808201518160030160006101000a8154816001600160a01b0302191690836001600160a01b03160217905550905050610630898686868080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250610d0592505050565b805160208201516001600160801b0391821670010000000000000000000000000000000091831691909102176004556040820151600580547fffffffffffffffffffffffffffffffff0000000000000000000000000000000016919092161790556060810151600655608001516007805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03909216919091179055505050505050505050505050565b60405162461bcd60e51b815260206004820152600f60248201527f4e6f7420696d706c656d656e746564000000000000000000000000000000000060448201526064015b60405180910390fd5b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036107c25760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c0000000000000000000000000000000000000000606482015260840161071b565b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038054336001600160a01b03821614610838576040517f23295f0e0000000000000000000000000000000000000000000000000000000081523360048201526001600160a01b038216602482015260440161071b565b600160009054906101000a90046001600160a01b03166001600160a01b031663cb23bcb56040518163ffffffff1660e01b8152600401602060405180830381865afa15801561088b573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906108af91906111dd565b6000805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03929092169190911790555050565b600088888888888888886040516020016109019897969594939291906111fa565b60405160208183030381529060405280519060200120905098975050505050505050565b60008281526003602052604080822083905551829184917fb4df3847300f076a369cd76d2314b470a1194d9e8a6bb97f1860aee88a5f67489190a35050565b6001546001600160a01b0316156109a7576040517fef34ca5c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038316908117909155604080517fcb23bcb5000000000000000000000000000000000000000000000000000000008152905163cb23bcb5916004808201926020929091908290030181865afa158015610a26573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610a4a91906111dd565b6000805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b039290921691909117905550565b60405162461bcd60e51b815260206004820152600e60248201527f4e4f545f494d504c454d45544544000000000000000000000000000000000000604482015260009060640161071b565b8251600090610100811115610b11576040517ffdac331e00000000000000000000000000000000000000000000000000000000815260048101829052610100602482015260440161071b565b8260005b82811015610b76576000878281518110610b3157610b31611266565b60200260200101519050816001901b8716600003610b5d57826000528060205260406000209250610b6d565b8060005282602052604060002092505b50600101610b15565b5095945050505050565b6000610100845110610bc35783516040517fab6a068300000000000000000000000000000000000000000000000000000000815260040161071b91815260200190565b8351610bd0906002611378565b8310610c20578284516002610be59190611378565b6040517f0b8a724b0000000000000000000000000000000000000000000000000000000081526004810192909252602482015260440161071b565b6000610c2d858585610326565b600081815260036020526040902054909150610c78576040517f8730d7c80000000000000000000000000000000000000000000000000000000081526004810182905260240161071b565b60008481526002602052604090205460ff1615610cc4576040517f9715b8d30000000000000000000000000000000000000000000000000000000081526004810185905260240161071b565b5050600082815260026020526040902080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001179055819392505050565b6001546040517f9e5d4c4900000000000000000000000000000000000000000000000000000000815260009182916001600160a01b0390911690639e5d4c4990610d57908890889088906004016113a8565b6000604051808303816000875af1158015610d76573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f19168201604052610d9e91908101906113f2565b9150915081610de957805115610db75780518082602001fd5b6040517f376fb55a00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b5050505050565b634e487b7160e01b600052604160045260246000fd5b604051601f8201601f1916810167ffffffffffffffff81118282101715610e2f57610e2f610df0565b604052919050565b600080600060608486031215610e4c57600080fd5b833567ffffffffffffffff80821115610e6457600080fd5b818601915086601f830112610e7857600080fd5b8135602082821115610e8c57610e8c610df0565b8160051b9250610e9d818401610e06565b828152928401810192818101908a851115610eb757600080fd5b948201945b84861015610ed557853582529482019490820190610ebc565b9a918901359950506040909701359695505050505050565b6001600160a01b0381168114610f0257600080fd5b50565b8035610f1081610eed565b919050565b60008083601f840112610f2757600080fd5b50813567ffffffffffffffff811115610f3f57600080fd5b602083019150836020828501011115610f5757600080fd5b9250929050565b60008060008060008060008060008060006101208c8e031215610f8057600080fd5b8b3567ffffffffffffffff80821115610f9857600080fd5b818e0191508e601f830112610fac57600080fd5b813581811115610fbb57600080fd5b8f60208260051b8501011115610fd057600080fd5b60208381019e50909c508e01359a50610feb60408f01610f05565b9950610ff960608f01610f05565b985060808e0135975060a08e0135965060c08e0135955060e08e013594506101008e013591508082111561102c57600080fd5b506110398e828f01610f15565b915080935050809150509295989b509295989b9093969950565b60008060008060008060008060006101008a8c03121561107257600080fd5b8935985060208a013561108481610eed565b975060408a013561109481610eed565b965060608a0135955060808a0135945060a08a0135935060c08a0135925060e08a013567ffffffffffffffff8111156110cc57600080fd5b6110d88c828d01610f15565b915080935050809150509295985092959850929598565b60006020828403121561110157600080fd5b5035919050565b60008060008060008060008060e0898b03121561112457600080fd5b883561112f81610eed565b9750602089013561113f81610eed565b965060408901359550606089013594506080890135935060a0890135925060c089013567ffffffffffffffff81111561117757600080fd5b6111838b828c01610f15565b999c989b5096995094979396929594505050565b600080604083850312156111aa57600080fd5b50508035926020909101359150565b6000602082840312156111cb57600080fd5b81356111d681610eed565b9392505050565b6000602082840312156111ef57600080fd5b81516111d681610eed565b60007fffffffffffffffffffffffffffffffffffffffff000000000000000000000000808b60601b168352808a60601b16601484015250876028830152866048830152856068830152846088830152828460a8840137506000910160a801908152979650505050505050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b600181815b808511156112cd5781600019048211156112b3576112b361127c565b808516156112c057918102915b93841c9390800290611297565b509250929050565b6000826112e457506001611372565b816112f157506000611372565b816001811461130757600281146113115761132d565b6001915050611372565b60ff8411156113225761132261127c565b50506001821b611372565b5060208310610133831016604e8410600b8410161715611350575081810a611372565b61135a8383611292565b806000190482111561136e5761136e61127c565b0290505b92915050565b60006111d683836112d5565b60005b8381101561139f578181015183820152602001611387565b50506000910152565b6001600160a01b038416815282602082015260606040820152600082518060608401526113dc816080850160208701611384565b601f01601f191691909101608001949350505050565b6000806040838503121561140557600080fd5b8251801515811461141557600080fd5b602084015190925067ffffffffffffffff8082111561143357600080fd5b818501915085601f83011261144757600080fd5b81518181111561145957611459610df0565b61146c6020601f19601f84011601610e06565b915080825286602082850101111561148357600080fd5b611494816020840160208601611384565b508092505050925092905056fea26469706673582212203835b38301f1d4bf75f1900d701a4bd72ad5d907f69084ad418e99c556915ca964736f6c63430008110033",
}

// OutboxWithoutOptTesterABI is the input ABI used to generate the binding from.
// Deprecated: Use OutboxWithoutOptTesterMetaData.ABI instead.
var OutboxWithoutOptTesterABI = OutboxWithoutOptTesterMetaData.ABI

// OutboxWithoutOptTesterBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OutboxWithoutOptTesterMetaData.Bin instead.
var OutboxWithoutOptTesterBin = OutboxWithoutOptTesterMetaData.Bin

// DeployOutboxWithoutOptTester deploys a new Ethereum contract, binding an instance of OutboxWithoutOptTester to it.
func DeployOutboxWithoutOptTester(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OutboxWithoutOptTester, error) {
	parsed, err := OutboxWithoutOptTesterMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OutboxWithoutOptTesterBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OutboxWithoutOptTester{OutboxWithoutOptTesterCaller: OutboxWithoutOptTesterCaller{contract: contract}, OutboxWithoutOptTesterTransactor: OutboxWithoutOptTesterTransactor{contract: contract}, OutboxWithoutOptTesterFilterer: OutboxWithoutOptTesterFilterer{contract: contract}}, nil
}

// OutboxWithoutOptTester is an auto generated Go binding around an Ethereum contract.
type OutboxWithoutOptTester struct {
	OutboxWithoutOptTesterCaller     // Read-only binding to the contract
	OutboxWithoutOptTesterTransactor // Write-only binding to the contract
	OutboxWithoutOptTesterFilterer   // Log filterer for contract events
}

// OutboxWithoutOptTesterCaller is an auto generated read-only Go binding around an Ethereum contract.
type OutboxWithoutOptTesterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OutboxWithoutOptTesterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OutboxWithoutOptTesterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OutboxWithoutOptTesterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OutboxWithoutOptTesterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OutboxWithoutOptTesterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OutboxWithoutOptTesterSession struct {
	Contract     *OutboxWithoutOptTester // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// OutboxWithoutOptTesterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OutboxWithoutOptTesterCallerSession struct {
	Contract *OutboxWithoutOptTesterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// OutboxWithoutOptTesterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OutboxWithoutOptTesterTransactorSession struct {
	Contract     *OutboxWithoutOptTesterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// OutboxWithoutOptTesterRaw is an auto generated low-level Go binding around an Ethereum contract.
type OutboxWithoutOptTesterRaw struct {
	Contract *OutboxWithoutOptTester // Generic contract binding to access the raw methods on
}

// OutboxWithoutOptTesterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OutboxWithoutOptTesterCallerRaw struct {
	Contract *OutboxWithoutOptTesterCaller // Generic read-only contract binding to access the raw methods on
}

// OutboxWithoutOptTesterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OutboxWithoutOptTesterTransactorRaw struct {
	Contract *OutboxWithoutOptTesterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOutboxWithoutOptTester creates a new instance of OutboxWithoutOptTester, bound to a specific deployed contract.
func NewOutboxWithoutOptTester(address common.Address, backend bind.ContractBackend) (*OutboxWithoutOptTester, error) {
	contract, err := bindOutboxWithoutOptTester(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OutboxWithoutOptTester{OutboxWithoutOptTesterCaller: OutboxWithoutOptTesterCaller{contract: contract}, OutboxWithoutOptTesterTransactor: OutboxWithoutOptTesterTransactor{contract: contract}, OutboxWithoutOptTesterFilterer: OutboxWithoutOptTesterFilterer{contract: contract}}, nil
}

// NewOutboxWithoutOptTesterCaller creates a new read-only instance of OutboxWithoutOptTester, bound to a specific deployed contract.
func NewOutboxWithoutOptTesterCaller(address common.Address, caller bind.ContractCaller) (*OutboxWithoutOptTesterCaller, error) {
	contract, err := bindOutboxWithoutOptTester(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OutboxWithoutOptTesterCaller{contract: contract}, nil
}

// NewOutboxWithoutOptTesterTransactor creates a new write-only instance of OutboxWithoutOptTester, bound to a specific deployed contract.
func NewOutboxWithoutOptTesterTransactor(address common.Address, transactor bind.ContractTransactor) (*OutboxWithoutOptTesterTransactor, error) {
	contract, err := bindOutboxWithoutOptTester(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OutboxWithoutOptTesterTransactor{contract: contract}, nil
}

// NewOutboxWithoutOptTesterFilterer creates a new log filterer instance of OutboxWithoutOptTester, bound to a specific deployed contract.
func NewOutboxWithoutOptTesterFilterer(address common.Address, filterer bind.ContractFilterer) (*OutboxWithoutOptTesterFilterer, error) {
	contract, err := bindOutboxWithoutOptTester(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OutboxWithoutOptTesterFilterer{contract: contract}, nil
}

// bindOutboxWithoutOptTester binds a generic wrapper to an already deployed contract.
func bindOutboxWithoutOptTester(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OutboxWithoutOptTesterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OutboxWithoutOptTester.Contract.OutboxWithoutOptTesterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.OutboxWithoutOptTesterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.OutboxWithoutOptTesterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OutboxWithoutOptTester.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.contract.Transact(opts, method, params...)
}

// OUTBOXVERSION is a free data retrieval call binding the contract method 0xc75184df.
//
// Solidity: function OUTBOX_VERSION() view returns(uint128)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) OUTBOXVERSION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "OUTBOX_VERSION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OUTBOXVERSION is a free data retrieval call binding the contract method 0xc75184df.
//
// Solidity: function OUTBOX_VERSION() view returns(uint128)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) OUTBOXVERSION() (*big.Int, error) {
	return _OutboxWithoutOptTester.Contract.OUTBOXVERSION(&_OutboxWithoutOptTester.CallOpts)
}

// OUTBOXVERSION is a free data retrieval call binding the contract method 0xc75184df.
//
// Solidity: function OUTBOX_VERSION() view returns(uint128)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) OUTBOXVERSION() (*big.Int, error) {
	return _OutboxWithoutOptTester.Contract.OUTBOXVERSION(&_OutboxWithoutOptTester.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) Bridge() (common.Address, error) {
	return _OutboxWithoutOptTester.Contract.Bridge(&_OutboxWithoutOptTester.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) Bridge() (common.Address, error) {
	return _OutboxWithoutOptTester.Contract.Bridge(&_OutboxWithoutOptTester.CallOpts)
}

// CalculateItemHash is a free data retrieval call binding the contract method 0x9f0c04bf.
//
// Solidity: function calculateItemHash(address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) pure returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) CalculateItemHash(opts *bind.CallOpts, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) ([32]byte, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "calculateItemHash", l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateItemHash is a free data retrieval call binding the contract method 0x9f0c04bf.
//
// Solidity: function calculateItemHash(address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) pure returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) CalculateItemHash(l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) ([32]byte, error) {
	return _OutboxWithoutOptTester.Contract.CalculateItemHash(&_OutboxWithoutOptTester.CallOpts, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// CalculateItemHash is a free data retrieval call binding the contract method 0x9f0c04bf.
//
// Solidity: function calculateItemHash(address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) pure returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) CalculateItemHash(l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) ([32]byte, error) {
	return _OutboxWithoutOptTester.Contract.CalculateItemHash(&_OutboxWithoutOptTester.CallOpts, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// CalculateMerkleRoot is a free data retrieval call binding the contract method 0x007436d3.
//
// Solidity: function calculateMerkleRoot(bytes32[] proof, uint256 path, bytes32 item) pure returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) CalculateMerkleRoot(opts *bind.CallOpts, proof [][32]byte, path *big.Int, item [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "calculateMerkleRoot", proof, path, item)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateMerkleRoot is a free data retrieval call binding the contract method 0x007436d3.
//
// Solidity: function calculateMerkleRoot(bytes32[] proof, uint256 path, bytes32 item) pure returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) CalculateMerkleRoot(proof [][32]byte, path *big.Int, item [32]byte) ([32]byte, error) {
	return _OutboxWithoutOptTester.Contract.CalculateMerkleRoot(&_OutboxWithoutOptTester.CallOpts, proof, path, item)
}

// CalculateMerkleRoot is a free data retrieval call binding the contract method 0x007436d3.
//
// Solidity: function calculateMerkleRoot(bytes32[] proof, uint256 path, bytes32 item) pure returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) CalculateMerkleRoot(proof [][32]byte, path *big.Int, item [32]byte) ([32]byte, error) {
	return _OutboxWithoutOptTester.Contract.CalculateMerkleRoot(&_OutboxWithoutOptTester.CallOpts, proof, path, item)
}

// ExecuteTransactionSimulation is a free data retrieval call binding the contract method 0x288e5b10.
//
// Solidity: function executeTransactionSimulation(uint256 , address , address , uint256 , uint256 , uint256 , uint256 , bytes ) pure returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) ExecuteTransactionSimulation(opts *bind.CallOpts, arg0 *big.Int, arg1 common.Address, arg2 common.Address, arg3 *big.Int, arg4 *big.Int, arg5 *big.Int, arg6 *big.Int, arg7 []byte) error {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "executeTransactionSimulation", arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)

	if err != nil {
		return err
	}

	return err

}

// ExecuteTransactionSimulation is a free data retrieval call binding the contract method 0x288e5b10.
//
// Solidity: function executeTransactionSimulation(uint256 , address , address , uint256 , uint256 , uint256 , uint256 , bytes ) pure returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) ExecuteTransactionSimulation(arg0 *big.Int, arg1 common.Address, arg2 common.Address, arg3 *big.Int, arg4 *big.Int, arg5 *big.Int, arg6 *big.Int, arg7 []byte) error {
	return _OutboxWithoutOptTester.Contract.ExecuteTransactionSimulation(&_OutboxWithoutOptTester.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
}

// ExecuteTransactionSimulation is a free data retrieval call binding the contract method 0x288e5b10.
//
// Solidity: function executeTransactionSimulation(uint256 , address , address , uint256 , uint256 , uint256 , uint256 , bytes ) pure returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) ExecuteTransactionSimulation(arg0 *big.Int, arg1 common.Address, arg2 common.Address, arg3 *big.Int, arg4 *big.Int, arg5 *big.Int, arg6 *big.Int, arg7 []byte) error {
	return _OutboxWithoutOptTester.Contract.ExecuteTransactionSimulation(&_OutboxWithoutOptTester.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
}

// IsSpent is a free data retrieval call binding the contract method 0x5a129efe.
//
// Solidity: function isSpent(uint256 ) view returns(bool)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) IsSpent(opts *bind.CallOpts, arg0 *big.Int) (bool, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "isSpent", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSpent is a free data retrieval call binding the contract method 0x5a129efe.
//
// Solidity: function isSpent(uint256 ) view returns(bool)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) IsSpent(arg0 *big.Int) (bool, error) {
	return _OutboxWithoutOptTester.Contract.IsSpent(&_OutboxWithoutOptTester.CallOpts, arg0)
}

// IsSpent is a free data retrieval call binding the contract method 0x5a129efe.
//
// Solidity: function isSpent(uint256 ) view returns(bool)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) IsSpent(arg0 *big.Int) (bool, error) {
	return _OutboxWithoutOptTester.Contract.IsSpent(&_OutboxWithoutOptTester.CallOpts, arg0)
}

// L2ToL1BatchNum is a free data retrieval call binding the contract method 0x11985271.
//
// Solidity: function l2ToL1BatchNum() pure returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) L2ToL1BatchNum(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "l2ToL1BatchNum")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1BatchNum is a free data retrieval call binding the contract method 0x11985271.
//
// Solidity: function l2ToL1BatchNum() pure returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) L2ToL1BatchNum() (*big.Int, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1BatchNum(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1BatchNum is a free data retrieval call binding the contract method 0x11985271.
//
// Solidity: function l2ToL1BatchNum() pure returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) L2ToL1BatchNum() (*big.Int, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1BatchNum(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1Block is a free data retrieval call binding the contract method 0x46547790.
//
// Solidity: function l2ToL1Block() view returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) L2ToL1Block(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "l2ToL1Block")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1Block is a free data retrieval call binding the contract method 0x46547790.
//
// Solidity: function l2ToL1Block() view returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) L2ToL1Block() (*big.Int, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1Block(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1Block is a free data retrieval call binding the contract method 0x46547790.
//
// Solidity: function l2ToL1Block() view returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) L2ToL1Block() (*big.Int, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1Block(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1EthBlock is a free data retrieval call binding the contract method 0x8515bc6a.
//
// Solidity: function l2ToL1EthBlock() view returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) L2ToL1EthBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "l2ToL1EthBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1EthBlock is a free data retrieval call binding the contract method 0x8515bc6a.
//
// Solidity: function l2ToL1EthBlock() view returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) L2ToL1EthBlock() (*big.Int, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1EthBlock(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1EthBlock is a free data retrieval call binding the contract method 0x8515bc6a.
//
// Solidity: function l2ToL1EthBlock() view returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) L2ToL1EthBlock() (*big.Int, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1EthBlock(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1OutputId is a free data retrieval call binding the contract method 0x72f2a8c7.
//
// Solidity: function l2ToL1OutputId() view returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) L2ToL1OutputId(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "l2ToL1OutputId")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L2ToL1OutputId is a free data retrieval call binding the contract method 0x72f2a8c7.
//
// Solidity: function l2ToL1OutputId() view returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) L2ToL1OutputId() ([32]byte, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1OutputId(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1OutputId is a free data retrieval call binding the contract method 0x72f2a8c7.
//
// Solidity: function l2ToL1OutputId() view returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) L2ToL1OutputId() ([32]byte, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1OutputId(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1Sender is a free data retrieval call binding the contract method 0x80648b02.
//
// Solidity: function l2ToL1Sender() view returns(address)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) L2ToL1Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "l2ToL1Sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2ToL1Sender is a free data retrieval call binding the contract method 0x80648b02.
//
// Solidity: function l2ToL1Sender() view returns(address)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) L2ToL1Sender() (common.Address, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1Sender(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1Sender is a free data retrieval call binding the contract method 0x80648b02.
//
// Solidity: function l2ToL1Sender() view returns(address)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) L2ToL1Sender() (common.Address, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1Sender(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1Timestamp is a free data retrieval call binding the contract method 0xb0f30537.
//
// Solidity: function l2ToL1Timestamp() view returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) L2ToL1Timestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "l2ToL1Timestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1Timestamp is a free data retrieval call binding the contract method 0xb0f30537.
//
// Solidity: function l2ToL1Timestamp() view returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) L2ToL1Timestamp() (*big.Int, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1Timestamp(&_OutboxWithoutOptTester.CallOpts)
}

// L2ToL1Timestamp is a free data retrieval call binding the contract method 0xb0f30537.
//
// Solidity: function l2ToL1Timestamp() view returns(uint256)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) L2ToL1Timestamp() (*big.Int, error) {
	return _OutboxWithoutOptTester.Contract.L2ToL1Timestamp(&_OutboxWithoutOptTester.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) Rollup() (common.Address, error) {
	return _OutboxWithoutOptTester.Contract.Rollup(&_OutboxWithoutOptTester.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) Rollup() (common.Address, error) {
	return _OutboxWithoutOptTester.Contract.Rollup(&_OutboxWithoutOptTester.CallOpts)
}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) Roots(opts *bind.CallOpts, arg0 [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "roots", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) Roots(arg0 [32]byte) ([32]byte, error) {
	return _OutboxWithoutOptTester.Contract.Roots(&_OutboxWithoutOptTester.CallOpts, arg0)
}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) Roots(arg0 [32]byte) ([32]byte, error) {
	return _OutboxWithoutOptTester.Contract.Roots(&_OutboxWithoutOptTester.CallOpts, arg0)
}

// Spent is a free data retrieval call binding the contract method 0xd5b5cc23.
//
// Solidity: function spent(uint256 ) pure returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCaller) Spent(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _OutboxWithoutOptTester.contract.Call(opts, &out, "spent", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Spent is a free data retrieval call binding the contract method 0xd5b5cc23.
//
// Solidity: function spent(uint256 ) pure returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) Spent(arg0 *big.Int) ([32]byte, error) {
	return _OutboxWithoutOptTester.Contract.Spent(&_OutboxWithoutOptTester.CallOpts, arg0)
}

// Spent is a free data retrieval call binding the contract method 0xd5b5cc23.
//
// Solidity: function spent(uint256 ) pure returns(bytes32)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterCallerSession) Spent(arg0 *big.Int) ([32]byte, error) {
	return _OutboxWithoutOptTester.Contract.Spent(&_OutboxWithoutOptTester.CallOpts, arg0)
}

// ExecuteTransaction is a paid mutator transaction binding the contract method 0x08635a95.
//
// Solidity: function executeTransaction(bytes32[] proof, uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactor) ExecuteTransaction(opts *bind.TransactOpts, proof [][32]byte, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.contract.Transact(opts, "executeTransaction", proof, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransaction is a paid mutator transaction binding the contract method 0x08635a95.
//
// Solidity: function executeTransaction(bytes32[] proof, uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) ExecuteTransaction(proof [][32]byte, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.ExecuteTransaction(&_OutboxWithoutOptTester.TransactOpts, proof, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransaction is a paid mutator transaction binding the contract method 0x08635a95.
//
// Solidity: function executeTransaction(bytes32[] proof, uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactorSession) ExecuteTransaction(proof [][32]byte, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.ExecuteTransaction(&_OutboxWithoutOptTester.TransactOpts, proof, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactor) Initialize(opts *bind.TransactOpts, _bridge common.Address) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.contract.Transact(opts, "initialize", _bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) Initialize(_bridge common.Address) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.Initialize(&_OutboxWithoutOptTester.TransactOpts, _bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactorSession) Initialize(_bridge common.Address) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.Initialize(&_OutboxWithoutOptTester.TransactOpts, _bridge)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0x95fcea78.
//
// Solidity: function postUpgradeInit() returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactor) PostUpgradeInit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.contract.Transact(opts, "postUpgradeInit")
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0x95fcea78.
//
// Solidity: function postUpgradeInit() returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) PostUpgradeInit() (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.PostUpgradeInit(&_OutboxWithoutOptTester.TransactOpts)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0x95fcea78.
//
// Solidity: function postUpgradeInit() returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactorSession) PostUpgradeInit() (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.PostUpgradeInit(&_OutboxWithoutOptTester.TransactOpts)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactor) UpdateRollupAddress(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.contract.Transact(opts, "updateRollupAddress")
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.UpdateRollupAddress(&_OutboxWithoutOptTester.TransactOpts)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactorSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.UpdateRollupAddress(&_OutboxWithoutOptTester.TransactOpts)
}

// UpdateSendRoot is a paid mutator transaction binding the contract method 0xa04cee60.
//
// Solidity: function updateSendRoot(bytes32 root, bytes32 l2BlockHash) returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactor) UpdateSendRoot(opts *bind.TransactOpts, root [32]byte, l2BlockHash [32]byte) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.contract.Transact(opts, "updateSendRoot", root, l2BlockHash)
}

// UpdateSendRoot is a paid mutator transaction binding the contract method 0xa04cee60.
//
// Solidity: function updateSendRoot(bytes32 root, bytes32 l2BlockHash) returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterSession) UpdateSendRoot(root [32]byte, l2BlockHash [32]byte) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.UpdateSendRoot(&_OutboxWithoutOptTester.TransactOpts, root, l2BlockHash)
}

// UpdateSendRoot is a paid mutator transaction binding the contract method 0xa04cee60.
//
// Solidity: function updateSendRoot(bytes32 root, bytes32 l2BlockHash) returns()
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterTransactorSession) UpdateSendRoot(root [32]byte, l2BlockHash [32]byte) (*types.Transaction, error) {
	return _OutboxWithoutOptTester.Contract.UpdateSendRoot(&_OutboxWithoutOptTester.TransactOpts, root, l2BlockHash)
}

// OutboxWithoutOptTesterOutBoxTransactionExecutedIterator is returned from FilterOutBoxTransactionExecuted and is used to iterate over the raw logs and unpacked data for OutBoxTransactionExecuted events raised by the OutboxWithoutOptTester contract.
type OutboxWithoutOptTesterOutBoxTransactionExecutedIterator struct {
	Event *OutboxWithoutOptTesterOutBoxTransactionExecuted // Event containing the contract specifics and raw log

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
func (it *OutboxWithoutOptTesterOutBoxTransactionExecutedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OutboxWithoutOptTesterOutBoxTransactionExecuted)
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
		it.Event = new(OutboxWithoutOptTesterOutBoxTransactionExecuted)
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
func (it *OutboxWithoutOptTesterOutBoxTransactionExecutedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OutboxWithoutOptTesterOutBoxTransactionExecutedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OutboxWithoutOptTesterOutBoxTransactionExecuted represents a OutBoxTransactionExecuted event raised by the OutboxWithoutOptTester contract.
type OutboxWithoutOptTesterOutBoxTransactionExecuted struct {
	To               common.Address
	L2Sender         common.Address
	Zero             *big.Int
	TransactionIndex *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterOutBoxTransactionExecuted is a free log retrieval operation binding the contract event 0x20af7f3bbfe38132b8900ae295cd9c8d1914be7052d061a511f3f728dab18964.
//
// Solidity: event OutBoxTransactionExecuted(address indexed to, address indexed l2Sender, uint256 indexed zero, uint256 transactionIndex)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterFilterer) FilterOutBoxTransactionExecuted(opts *bind.FilterOpts, to []common.Address, l2Sender []common.Address, zero []*big.Int) (*OutboxWithoutOptTesterOutBoxTransactionExecutedIterator, error) {

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var l2SenderRule []interface{}
	for _, l2SenderItem := range l2Sender {
		l2SenderRule = append(l2SenderRule, l2SenderItem)
	}
	var zeroRule []interface{}
	for _, zeroItem := range zero {
		zeroRule = append(zeroRule, zeroItem)
	}

	logs, sub, err := _OutboxWithoutOptTester.contract.FilterLogs(opts, "OutBoxTransactionExecuted", toRule, l2SenderRule, zeroRule)
	if err != nil {
		return nil, err
	}
	return &OutboxWithoutOptTesterOutBoxTransactionExecutedIterator{contract: _OutboxWithoutOptTester.contract, event: "OutBoxTransactionExecuted", logs: logs, sub: sub}, nil
}

// WatchOutBoxTransactionExecuted is a free log subscription operation binding the contract event 0x20af7f3bbfe38132b8900ae295cd9c8d1914be7052d061a511f3f728dab18964.
//
// Solidity: event OutBoxTransactionExecuted(address indexed to, address indexed l2Sender, uint256 indexed zero, uint256 transactionIndex)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterFilterer) WatchOutBoxTransactionExecuted(opts *bind.WatchOpts, sink chan<- *OutboxWithoutOptTesterOutBoxTransactionExecuted, to []common.Address, l2Sender []common.Address, zero []*big.Int) (event.Subscription, error) {

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var l2SenderRule []interface{}
	for _, l2SenderItem := range l2Sender {
		l2SenderRule = append(l2SenderRule, l2SenderItem)
	}
	var zeroRule []interface{}
	for _, zeroItem := range zero {
		zeroRule = append(zeroRule, zeroItem)
	}

	logs, sub, err := _OutboxWithoutOptTester.contract.WatchLogs(opts, "OutBoxTransactionExecuted", toRule, l2SenderRule, zeroRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OutboxWithoutOptTesterOutBoxTransactionExecuted)
				if err := _OutboxWithoutOptTester.contract.UnpackLog(event, "OutBoxTransactionExecuted", log); err != nil {
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

// ParseOutBoxTransactionExecuted is a log parse operation binding the contract event 0x20af7f3bbfe38132b8900ae295cd9c8d1914be7052d061a511f3f728dab18964.
//
// Solidity: event OutBoxTransactionExecuted(address indexed to, address indexed l2Sender, uint256 indexed zero, uint256 transactionIndex)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterFilterer) ParseOutBoxTransactionExecuted(log types.Log) (*OutboxWithoutOptTesterOutBoxTransactionExecuted, error) {
	event := new(OutboxWithoutOptTesterOutBoxTransactionExecuted)
	if err := _OutboxWithoutOptTester.contract.UnpackLog(event, "OutBoxTransactionExecuted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OutboxWithoutOptTesterSendRootUpdatedIterator is returned from FilterSendRootUpdated and is used to iterate over the raw logs and unpacked data for SendRootUpdated events raised by the OutboxWithoutOptTester contract.
type OutboxWithoutOptTesterSendRootUpdatedIterator struct {
	Event *OutboxWithoutOptTesterSendRootUpdated // Event containing the contract specifics and raw log

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
func (it *OutboxWithoutOptTesterSendRootUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OutboxWithoutOptTesterSendRootUpdated)
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
		it.Event = new(OutboxWithoutOptTesterSendRootUpdated)
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
func (it *OutboxWithoutOptTesterSendRootUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OutboxWithoutOptTesterSendRootUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OutboxWithoutOptTesterSendRootUpdated represents a SendRootUpdated event raised by the OutboxWithoutOptTester contract.
type OutboxWithoutOptTesterSendRootUpdated struct {
	OutputRoot  [32]byte
	L2BlockHash [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSendRootUpdated is a free log retrieval operation binding the contract event 0xb4df3847300f076a369cd76d2314b470a1194d9e8a6bb97f1860aee88a5f6748.
//
// Solidity: event SendRootUpdated(bytes32 indexed outputRoot, bytes32 indexed l2BlockHash)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterFilterer) FilterSendRootUpdated(opts *bind.FilterOpts, outputRoot [][32]byte, l2BlockHash [][32]byte) (*OutboxWithoutOptTesterSendRootUpdatedIterator, error) {

	var outputRootRule []interface{}
	for _, outputRootItem := range outputRoot {
		outputRootRule = append(outputRootRule, outputRootItem)
	}
	var l2BlockHashRule []interface{}
	for _, l2BlockHashItem := range l2BlockHash {
		l2BlockHashRule = append(l2BlockHashRule, l2BlockHashItem)
	}

	logs, sub, err := _OutboxWithoutOptTester.contract.FilterLogs(opts, "SendRootUpdated", outputRootRule, l2BlockHashRule)
	if err != nil {
		return nil, err
	}
	return &OutboxWithoutOptTesterSendRootUpdatedIterator{contract: _OutboxWithoutOptTester.contract, event: "SendRootUpdated", logs: logs, sub: sub}, nil
}

// WatchSendRootUpdated is a free log subscription operation binding the contract event 0xb4df3847300f076a369cd76d2314b470a1194d9e8a6bb97f1860aee88a5f6748.
//
// Solidity: event SendRootUpdated(bytes32 indexed outputRoot, bytes32 indexed l2BlockHash)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterFilterer) WatchSendRootUpdated(opts *bind.WatchOpts, sink chan<- *OutboxWithoutOptTesterSendRootUpdated, outputRoot [][32]byte, l2BlockHash [][32]byte) (event.Subscription, error) {

	var outputRootRule []interface{}
	for _, outputRootItem := range outputRoot {
		outputRootRule = append(outputRootRule, outputRootItem)
	}
	var l2BlockHashRule []interface{}
	for _, l2BlockHashItem := range l2BlockHash {
		l2BlockHashRule = append(l2BlockHashRule, l2BlockHashItem)
	}

	logs, sub, err := _OutboxWithoutOptTester.contract.WatchLogs(opts, "SendRootUpdated", outputRootRule, l2BlockHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OutboxWithoutOptTesterSendRootUpdated)
				if err := _OutboxWithoutOptTester.contract.UnpackLog(event, "SendRootUpdated", log); err != nil {
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

// ParseSendRootUpdated is a log parse operation binding the contract event 0xb4df3847300f076a369cd76d2314b470a1194d9e8a6bb97f1860aee88a5f6748.
//
// Solidity: event SendRootUpdated(bytes32 indexed outputRoot, bytes32 indexed l2BlockHash)
func (_OutboxWithoutOptTester *OutboxWithoutOptTesterFilterer) ParseSendRootUpdated(log types.Log) (*OutboxWithoutOptTesterSendRootUpdated, error) {
	event := new(OutboxWithoutOptTesterSendRootUpdated)
	if err := _OutboxWithoutOptTester.contract.UnpackLog(event, "SendRootUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RollupMockMetaData contains all meta data concerning the RollupMock contract.
var RollupMockMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"WithdrawTriggered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"ZombieTriggered\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawStakerFunds\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5060405161018c38038061018c83398101604081905261002f91610054565b600080546001600160a01b0319166001600160a01b0392909216919091179055610084565b60006020828403121561006657600080fd5b81516001600160a01b038116811461007d57600080fd5b9392505050565b60fa806100926000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c8063613739191460375780638da5cb5b146050575b600080fd5b603d6093565b6040519081526020015b60405180910390f35b600054606f9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016047565b6040516000907f1c09fbbf7cfd024f5e4e5472dd87afd5d67ee5db6a0ca715bf508d96abce309f908290a15060009056fea2646970667358221220e36130a8e9119bfb252dbac8ceec14eb5df19884c0e88f45682b2a726751b65364736f6c63430008110033",
}

// RollupMockABI is the input ABI used to generate the binding from.
// Deprecated: Use RollupMockMetaData.ABI instead.
var RollupMockABI = RollupMockMetaData.ABI

// RollupMockBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use RollupMockMetaData.Bin instead.
var RollupMockBin = RollupMockMetaData.Bin

// DeployRollupMock deploys a new Ethereum contract, binding an instance of RollupMock to it.
func DeployRollupMock(auth *bind.TransactOpts, backend bind.ContractBackend, _owner common.Address) (common.Address, *types.Transaction, *RollupMock, error) {
	parsed, err := RollupMockMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(RollupMockBin), backend, _owner)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &RollupMock{RollupMockCaller: RollupMockCaller{contract: contract}, RollupMockTransactor: RollupMockTransactor{contract: contract}, RollupMockFilterer: RollupMockFilterer{contract: contract}}, nil
}

// RollupMock is an auto generated Go binding around an Ethereum contract.
type RollupMock struct {
	RollupMockCaller     // Read-only binding to the contract
	RollupMockTransactor // Write-only binding to the contract
	RollupMockFilterer   // Log filterer for contract events
}

// RollupMockCaller is an auto generated read-only Go binding around an Ethereum contract.
type RollupMockCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RollupMockTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RollupMockTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RollupMockFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RollupMockFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RollupMockSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RollupMockSession struct {
	Contract     *RollupMock       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RollupMockCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RollupMockCallerSession struct {
	Contract *RollupMockCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// RollupMockTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RollupMockTransactorSession struct {
	Contract     *RollupMockTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// RollupMockRaw is an auto generated low-level Go binding around an Ethereum contract.
type RollupMockRaw struct {
	Contract *RollupMock // Generic contract binding to access the raw methods on
}

// RollupMockCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RollupMockCallerRaw struct {
	Contract *RollupMockCaller // Generic read-only contract binding to access the raw methods on
}

// RollupMockTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RollupMockTransactorRaw struct {
	Contract *RollupMockTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRollupMock creates a new instance of RollupMock, bound to a specific deployed contract.
func NewRollupMock(address common.Address, backend bind.ContractBackend) (*RollupMock, error) {
	contract, err := bindRollupMock(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RollupMock{RollupMockCaller: RollupMockCaller{contract: contract}, RollupMockTransactor: RollupMockTransactor{contract: contract}, RollupMockFilterer: RollupMockFilterer{contract: contract}}, nil
}

// NewRollupMockCaller creates a new read-only instance of RollupMock, bound to a specific deployed contract.
func NewRollupMockCaller(address common.Address, caller bind.ContractCaller) (*RollupMockCaller, error) {
	contract, err := bindRollupMock(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RollupMockCaller{contract: contract}, nil
}

// NewRollupMockTransactor creates a new write-only instance of RollupMock, bound to a specific deployed contract.
func NewRollupMockTransactor(address common.Address, transactor bind.ContractTransactor) (*RollupMockTransactor, error) {
	contract, err := bindRollupMock(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RollupMockTransactor{contract: contract}, nil
}

// NewRollupMockFilterer creates a new log filterer instance of RollupMock, bound to a specific deployed contract.
func NewRollupMockFilterer(address common.Address, filterer bind.ContractFilterer) (*RollupMockFilterer, error) {
	contract, err := bindRollupMock(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RollupMockFilterer{contract: contract}, nil
}

// bindRollupMock binds a generic wrapper to an already deployed contract.
func bindRollupMock(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RollupMockMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RollupMock *RollupMockRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RollupMock.Contract.RollupMockCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RollupMock *RollupMockRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RollupMock.Contract.RollupMockTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RollupMock *RollupMockRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RollupMock.Contract.RollupMockTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RollupMock *RollupMockCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RollupMock.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RollupMock *RollupMockTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RollupMock.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RollupMock *RollupMockTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RollupMock.Contract.contract.Transact(opts, method, params...)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_RollupMock *RollupMockCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _RollupMock.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_RollupMock *RollupMockSession) Owner() (common.Address, error) {
	return _RollupMock.Contract.Owner(&_RollupMock.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_RollupMock *RollupMockCallerSession) Owner() (common.Address, error) {
	return _RollupMock.Contract.Owner(&_RollupMock.CallOpts)
}

// WithdrawStakerFunds is a paid mutator transaction binding the contract method 0x61373919.
//
// Solidity: function withdrawStakerFunds() returns(uint256)
func (_RollupMock *RollupMockTransactor) WithdrawStakerFunds(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RollupMock.contract.Transact(opts, "withdrawStakerFunds")
}

// WithdrawStakerFunds is a paid mutator transaction binding the contract method 0x61373919.
//
// Solidity: function withdrawStakerFunds() returns(uint256)
func (_RollupMock *RollupMockSession) WithdrawStakerFunds() (*types.Transaction, error) {
	return _RollupMock.Contract.WithdrawStakerFunds(&_RollupMock.TransactOpts)
}

// WithdrawStakerFunds is a paid mutator transaction binding the contract method 0x61373919.
//
// Solidity: function withdrawStakerFunds() returns(uint256)
func (_RollupMock *RollupMockTransactorSession) WithdrawStakerFunds() (*types.Transaction, error) {
	return _RollupMock.Contract.WithdrawStakerFunds(&_RollupMock.TransactOpts)
}

// RollupMockWithdrawTriggeredIterator is returned from FilterWithdrawTriggered and is used to iterate over the raw logs and unpacked data for WithdrawTriggered events raised by the RollupMock contract.
type RollupMockWithdrawTriggeredIterator struct {
	Event *RollupMockWithdrawTriggered // Event containing the contract specifics and raw log

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
func (it *RollupMockWithdrawTriggeredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RollupMockWithdrawTriggered)
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
		it.Event = new(RollupMockWithdrawTriggered)
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
func (it *RollupMockWithdrawTriggeredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RollupMockWithdrawTriggeredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RollupMockWithdrawTriggered represents a WithdrawTriggered event raised by the RollupMock contract.
type RollupMockWithdrawTriggered struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterWithdrawTriggered is a free log retrieval operation binding the contract event 0x1c09fbbf7cfd024f5e4e5472dd87afd5d67ee5db6a0ca715bf508d96abce309f.
//
// Solidity: event WithdrawTriggered()
func (_RollupMock *RollupMockFilterer) FilterWithdrawTriggered(opts *bind.FilterOpts) (*RollupMockWithdrawTriggeredIterator, error) {

	logs, sub, err := _RollupMock.contract.FilterLogs(opts, "WithdrawTriggered")
	if err != nil {
		return nil, err
	}
	return &RollupMockWithdrawTriggeredIterator{contract: _RollupMock.contract, event: "WithdrawTriggered", logs: logs, sub: sub}, nil
}

// WatchWithdrawTriggered is a free log subscription operation binding the contract event 0x1c09fbbf7cfd024f5e4e5472dd87afd5d67ee5db6a0ca715bf508d96abce309f.
//
// Solidity: event WithdrawTriggered()
func (_RollupMock *RollupMockFilterer) WatchWithdrawTriggered(opts *bind.WatchOpts, sink chan<- *RollupMockWithdrawTriggered) (event.Subscription, error) {

	logs, sub, err := _RollupMock.contract.WatchLogs(opts, "WithdrawTriggered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RollupMockWithdrawTriggered)
				if err := _RollupMock.contract.UnpackLog(event, "WithdrawTriggered", log); err != nil {
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

// ParseWithdrawTriggered is a log parse operation binding the contract event 0x1c09fbbf7cfd024f5e4e5472dd87afd5d67ee5db6a0ca715bf508d96abce309f.
//
// Solidity: event WithdrawTriggered()
func (_RollupMock *RollupMockFilterer) ParseWithdrawTriggered(log types.Log) (*RollupMockWithdrawTriggered, error) {
	event := new(RollupMockWithdrawTriggered)
	if err := _RollupMock.contract.UnpackLog(event, "WithdrawTriggered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RollupMockZombieTriggeredIterator is returned from FilterZombieTriggered and is used to iterate over the raw logs and unpacked data for ZombieTriggered events raised by the RollupMock contract.
type RollupMockZombieTriggeredIterator struct {
	Event *RollupMockZombieTriggered // Event containing the contract specifics and raw log

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
func (it *RollupMockZombieTriggeredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RollupMockZombieTriggered)
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
		it.Event = new(RollupMockZombieTriggered)
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
func (it *RollupMockZombieTriggeredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RollupMockZombieTriggeredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RollupMockZombieTriggered represents a ZombieTriggered event raised by the RollupMock contract.
type RollupMockZombieTriggered struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterZombieTriggered is a free log retrieval operation binding the contract event 0xb774f793432a37585a7638b9afe49e91c478887a2c0fef32877508bf2f76429d.
//
// Solidity: event ZombieTriggered()
func (_RollupMock *RollupMockFilterer) FilterZombieTriggered(opts *bind.FilterOpts) (*RollupMockZombieTriggeredIterator, error) {

	logs, sub, err := _RollupMock.contract.FilterLogs(opts, "ZombieTriggered")
	if err != nil {
		return nil, err
	}
	return &RollupMockZombieTriggeredIterator{contract: _RollupMock.contract, event: "ZombieTriggered", logs: logs, sub: sub}, nil
}

// WatchZombieTriggered is a free log subscription operation binding the contract event 0xb774f793432a37585a7638b9afe49e91c478887a2c0fef32877508bf2f76429d.
//
// Solidity: event ZombieTriggered()
func (_RollupMock *RollupMockFilterer) WatchZombieTriggered(opts *bind.WatchOpts, sink chan<- *RollupMockZombieTriggered) (event.Subscription, error) {

	logs, sub, err := _RollupMock.contract.WatchLogs(opts, "ZombieTriggered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RollupMockZombieTriggered)
				if err := _RollupMock.contract.UnpackLog(event, "ZombieTriggered", log); err != nil {
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

// ParseZombieTriggered is a log parse operation binding the contract event 0xb774f793432a37585a7638b9afe49e91c478887a2c0fef32877508bf2f76429d.
//
// Solidity: event ZombieTriggered()
func (_RollupMock *RollupMockFilterer) ParseZombieTriggered(log types.Log) (*RollupMockZombieTriggered, error) {
	event := new(RollupMockZombieTriggered)
	if err := _RollupMock.contract.UnpackLog(event, "ZombieTriggered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TestTokenMetaData contains all meta data concerning the TestToken contract.
var TestTokenMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"initialSupply\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b5060405162000d4238038062000d4283398101604081905262000034916200019a565b604051806040016040528060098152602001682a32b9ba2a37b5b2b760b91b81525060405180604001604052806002815260200161151560f21b815250816003908162000082919062000258565b50600462000091828262000258565b505050620000a63382620000ad60201b60201c565b506200034c565b6001600160a01b038216620001085760405162461bcd60e51b815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f206164647265737300604482015260640160405180910390fd5b80600260008282546200011c919062000324565b90915550506001600160a01b038216600090815260208190526040812080548392906200014b90849062000324565b90915550506040518181526001600160a01b038316906000907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9060200160405180910390a35050565b505050565b600060208284031215620001ad57600080fd5b5051919050565b634e487b7160e01b600052604160045260246000fd5b600181811c90821680620001df57607f821691505b6020821081036200020057634e487b7160e01b600052602260045260246000fd5b50919050565b601f8211156200019557600081815260208120601f850160051c810160208610156200022f5750805b601f850160051c820191505b8181101562000250578281556001016200023b565b505050505050565b81516001600160401b03811115620002745762000274620001b4565b6200028c81620002858454620001ca565b8462000206565b602080601f831160018114620002c45760008415620002ab5750858301515b600019600386901b1c1916600185901b17855562000250565b600085815260208120601f198616915b82811015620002f557888601518255948401946001909101908401620002d4565b5085821015620003145787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b808201808211156200034657634e487b7160e01b600052601160045260246000fd5b92915050565b6109e6806200035c6000396000f3fe608060405234801561001057600080fd5b50600436106100c95760003560e01c80633950935111610081578063a457c2d71161005b578063a457c2d714610187578063a9059cbb1461019a578063dd62ed3e146101ad57600080fd5b8063395093511461014357806370a082311461015657806395d89b411461017f57600080fd5b806318160ddd116100b257806318160ddd1461010f57806323b872dd14610121578063313ce5671461013457600080fd5b806306fdde03146100ce578063095ea7b3146100ec575b600080fd5b6100d66101e6565b6040516100e391906107e0565b60405180910390f35b6100ff6100fa366004610868565b610278565b60405190151581526020016100e3565b6002545b6040519081526020016100e3565b6100ff61012f366004610892565b610292565b604051601281526020016100e3565b6100ff610151366004610868565b6102b6565b6101136101643660046108ce565b6001600160a01b031660009081526020819052604090205490565b6100d66102f5565b6100ff610195366004610868565b610304565b6100ff6101a8366004610868565b6103b3565b6101136101bb3660046108f0565b6001600160a01b03918216600090815260016020908152604080832093909416825291909152205490565b6060600380546101f590610923565b80601f016020809104026020016040519081016040528092919081815260200182805461022190610923565b801561026e5780601f106102435761010080835404028352916020019161026e565b820191906000526020600020905b81548152906001019060200180831161025157829003601f168201915b5050505050905090565b6000336102868185856103c1565b60019150505b92915050565b6000336102a0858285610519565b6102ab8585856105c9565b506001949350505050565b3360008181526001602090815260408083206001600160a01b038716845290915281205490919061028690829086906102f0908790610976565b6103c1565b6060600480546101f590610923565b3360008181526001602090815260408083206001600160a01b0387168452909152812054909190838110156103a65760405162461bcd60e51b815260206004820152602560248201527f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760448201527f207a65726f00000000000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b6102ab82868684036103c1565b6000336102868185856105c9565b6001600160a01b03831661043c5760405162461bcd60e51b8152602060048201526024808201527f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460448201527f7265737300000000000000000000000000000000000000000000000000000000606482015260840161039d565b6001600160a01b0382166104b85760405162461bcd60e51b815260206004820152602260248201527f45524332303a20617070726f766520746f20746865207a65726f20616464726560448201527f7373000000000000000000000000000000000000000000000000000000000000606482015260840161039d565b6001600160a01b0383811660008181526001602090815260408083209487168084529482529182902085905590518481527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925910160405180910390a3505050565b6001600160a01b038381166000908152600160209081526040808320938616835292905220547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81146105c357818110156105b65760405162461bcd60e51b815260206004820152601d60248201527f45524332303a20696e73756666696369656e7420616c6c6f77616e6365000000604482015260640161039d565b6105c384848484036103c1565b50505050565b6001600160a01b0383166106455760405162461bcd60e51b815260206004820152602560248201527f45524332303a207472616e736665722066726f6d20746865207a65726f20616460448201527f6472657373000000000000000000000000000000000000000000000000000000606482015260840161039d565b6001600160a01b0382166106c15760405162461bcd60e51b815260206004820152602360248201527f45524332303a207472616e7366657220746f20746865207a65726f206164647260448201527f6573730000000000000000000000000000000000000000000000000000000000606482015260840161039d565b6001600160a01b038316600090815260208190526040902054818110156107505760405162461bcd60e51b815260206004820152602660248201527f45524332303a207472616e7366657220616d6f756e742065786365656473206260448201527f616c616e63650000000000000000000000000000000000000000000000000000606482015260840161039d565b6001600160a01b03808516600090815260208190526040808220858503905591851681529081208054849290610787908490610976565b92505081905550826001600160a01b0316846001600160a01b03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040516107d391815260200190565b60405180910390a36105c3565b600060208083528351808285015260005b8181101561080d578581018301518582016040015282016107f1565b5060006040828601015260407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f8301168501019250505092915050565b80356001600160a01b038116811461086357600080fd5b919050565b6000806040838503121561087b57600080fd5b6108848361084c565b946020939093013593505050565b6000806000606084860312156108a757600080fd5b6108b08461084c565b92506108be6020850161084c565b9150604084013590509250925092565b6000602082840312156108e057600080fd5b6108e98261084c565b9392505050565b6000806040838503121561090357600080fd5b61090c8361084c565b915061091a6020840161084c565b90509250929050565b600181811c9082168061093757607f821691505b602082108103610970577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b8082018082111561028c577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fdfea2646970667358221220dae20d256ed58deb03c3e523c577c150397283006a35da4c09547ffba3ebd58d64736f6c63430008110033",
}

// TestTokenABI is the input ABI used to generate the binding from.
// Deprecated: Use TestTokenMetaData.ABI instead.
var TestTokenABI = TestTokenMetaData.ABI

// TestTokenBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TestTokenMetaData.Bin instead.
var TestTokenBin = TestTokenMetaData.Bin

// DeployTestToken deploys a new Ethereum contract, binding an instance of TestToken to it.
func DeployTestToken(auth *bind.TransactOpts, backend bind.ContractBackend, initialSupply *big.Int) (common.Address, *types.Transaction, *TestToken, error) {
	parsed, err := TestTokenMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TestTokenBin), backend, initialSupply)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TestToken{TestTokenCaller: TestTokenCaller{contract: contract}, TestTokenTransactor: TestTokenTransactor{contract: contract}, TestTokenFilterer: TestTokenFilterer{contract: contract}}, nil
}

// TestToken is an auto generated Go binding around an Ethereum contract.
type TestToken struct {
	TestTokenCaller     // Read-only binding to the contract
	TestTokenTransactor // Write-only binding to the contract
	TestTokenFilterer   // Log filterer for contract events
}

// TestTokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type TestTokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestTokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TestTokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestTokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TestTokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestTokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TestTokenSession struct {
	Contract     *TestToken        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TestTokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TestTokenCallerSession struct {
	Contract *TestTokenCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// TestTokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TestTokenTransactorSession struct {
	Contract     *TestTokenTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// TestTokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type TestTokenRaw struct {
	Contract *TestToken // Generic contract binding to access the raw methods on
}

// TestTokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TestTokenCallerRaw struct {
	Contract *TestTokenCaller // Generic read-only contract binding to access the raw methods on
}

// TestTokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TestTokenTransactorRaw struct {
	Contract *TestTokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTestToken creates a new instance of TestToken, bound to a specific deployed contract.
func NewTestToken(address common.Address, backend bind.ContractBackend) (*TestToken, error) {
	contract, err := bindTestToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TestToken{TestTokenCaller: TestTokenCaller{contract: contract}, TestTokenTransactor: TestTokenTransactor{contract: contract}, TestTokenFilterer: TestTokenFilterer{contract: contract}}, nil
}

// NewTestTokenCaller creates a new read-only instance of TestToken, bound to a specific deployed contract.
func NewTestTokenCaller(address common.Address, caller bind.ContractCaller) (*TestTokenCaller, error) {
	contract, err := bindTestToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TestTokenCaller{contract: contract}, nil
}

// NewTestTokenTransactor creates a new write-only instance of TestToken, bound to a specific deployed contract.
func NewTestTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*TestTokenTransactor, error) {
	contract, err := bindTestToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TestTokenTransactor{contract: contract}, nil
}

// NewTestTokenFilterer creates a new log filterer instance of TestToken, bound to a specific deployed contract.
func NewTestTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*TestTokenFilterer, error) {
	contract, err := bindTestToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TestTokenFilterer{contract: contract}, nil
}

// bindTestToken binds a generic wrapper to an already deployed contract.
func bindTestToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TestTokenMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TestToken *TestTokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TestToken.Contract.TestTokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TestToken *TestTokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TestToken.Contract.TestTokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TestToken *TestTokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TestToken.Contract.TestTokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TestToken *TestTokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TestToken.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TestToken *TestTokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TestToken.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TestToken *TestTokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TestToken.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_TestToken *TestTokenCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TestToken.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_TestToken *TestTokenSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _TestToken.Contract.Allowance(&_TestToken.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_TestToken *TestTokenCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _TestToken.Contract.Allowance(&_TestToken.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_TestToken *TestTokenCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TestToken.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_TestToken *TestTokenSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _TestToken.Contract.BalanceOf(&_TestToken.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_TestToken *TestTokenCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _TestToken.Contract.BalanceOf(&_TestToken.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_TestToken *TestTokenCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _TestToken.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_TestToken *TestTokenSession) Decimals() (uint8, error) {
	return _TestToken.Contract.Decimals(&_TestToken.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_TestToken *TestTokenCallerSession) Decimals() (uint8, error) {
	return _TestToken.Contract.Decimals(&_TestToken.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TestToken *TestTokenCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TestToken.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TestToken *TestTokenSession) Name() (string, error) {
	return _TestToken.Contract.Name(&_TestToken.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TestToken *TestTokenCallerSession) Name() (string, error) {
	return _TestToken.Contract.Name(&_TestToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TestToken *TestTokenCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TestToken.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TestToken *TestTokenSession) Symbol() (string, error) {
	return _TestToken.Contract.Symbol(&_TestToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TestToken *TestTokenCallerSession) Symbol() (string, error) {
	return _TestToken.Contract.Symbol(&_TestToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TestToken *TestTokenCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TestToken.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TestToken *TestTokenSession) TotalSupply() (*big.Int, error) {
	return _TestToken.Contract.TotalSupply(&_TestToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TestToken *TestTokenCallerSession) TotalSupply() (*big.Int, error) {
	return _TestToken.Contract.TotalSupply(&_TestToken.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_TestToken *TestTokenTransactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestToken.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_TestToken *TestTokenSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestToken.Contract.Approve(&_TestToken.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_TestToken *TestTokenTransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestToken.Contract.Approve(&_TestToken.TransactOpts, spender, amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_TestToken *TestTokenTransactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _TestToken.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_TestToken *TestTokenSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _TestToken.Contract.DecreaseAllowance(&_TestToken.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_TestToken *TestTokenTransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _TestToken.Contract.DecreaseAllowance(&_TestToken.TransactOpts, spender, subtractedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_TestToken *TestTokenTransactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _TestToken.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_TestToken *TestTokenSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _TestToken.Contract.IncreaseAllowance(&_TestToken.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_TestToken *TestTokenTransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _TestToken.Contract.IncreaseAllowance(&_TestToken.TransactOpts, spender, addedValue)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_TestToken *TestTokenTransactor) Transfer(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestToken.contract.Transact(opts, "transfer", to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_TestToken *TestTokenSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestToken.Contract.Transfer(&_TestToken.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_TestToken *TestTokenTransactorSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestToken.Contract.Transfer(&_TestToken.TransactOpts, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_TestToken *TestTokenTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestToken.contract.Transact(opts, "transferFrom", from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_TestToken *TestTokenSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestToken.Contract.TransferFrom(&_TestToken.TransactOpts, from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_TestToken *TestTokenTransactorSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TestToken.Contract.TransferFrom(&_TestToken.TransactOpts, from, to, amount)
}

// TestTokenApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the TestToken contract.
type TestTokenApprovalIterator struct {
	Event *TestTokenApproval // Event containing the contract specifics and raw log

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
func (it *TestTokenApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TestTokenApproval)
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
		it.Event = new(TestTokenApproval)
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
func (it *TestTokenApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TestTokenApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TestTokenApproval represents a Approval event raised by the TestToken contract.
type TestTokenApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_TestToken *TestTokenFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*TestTokenApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _TestToken.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &TestTokenApprovalIterator{contract: _TestToken.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_TestToken *TestTokenFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *TestTokenApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _TestToken.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TestTokenApproval)
				if err := _TestToken.contract.UnpackLog(event, "Approval", log); err != nil {
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
func (_TestToken *TestTokenFilterer) ParseApproval(log types.Log) (*TestTokenApproval, error) {
	event := new(TestTokenApproval)
	if err := _TestToken.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TestTokenTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the TestToken contract.
type TestTokenTransferIterator struct {
	Event *TestTokenTransfer // Event containing the contract specifics and raw log

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
func (it *TestTokenTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TestTokenTransfer)
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
		it.Event = new(TestTokenTransfer)
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
func (it *TestTokenTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TestTokenTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TestTokenTransfer represents a Transfer event raised by the TestToken contract.
type TestTokenTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_TestToken *TestTokenFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*TestTokenTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TestToken.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &TestTokenTransferIterator{contract: _TestToken.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_TestToken *TestTokenFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *TestTokenTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TestToken.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TestTokenTransfer)
				if err := _TestToken.contract.UnpackLog(event, "Transfer", log); err != nil {
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
func (_TestToken *TestTokenFilterer) ParseTransfer(log types.Log) (*TestTokenTransfer, error) {
	event := new(TestTokenTransfer)
	if err := _TestToken.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ValueArrayTesterMetaData contains all meta data concerning the ValueArrayTester contract.
var ValueArrayTesterMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"test\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061080a806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063f8a8fd6d14610030575b600080fd5b61003861003a565b005b6040805160026020820181815260808301845260009383929083015b604080518082019091526000808252602082015281526020019060019003908161005657505090528051519091506002146100d85760405162461bcd60e51b815260206004820152600960248201527f53544152545f4c454e000000000000000000000000000000000000000000000060448201526064015b60405180910390fd5b60408051808201825260008082526020918201819052825180840190935280835260019183019190915261010f915b8391906104b9565b604080518082018252600080825260209182018190528251808401909352825260029082015261014190600190610107565b60408051808201825260008082526020918201819052825180840190935282526003908201526101729082906104e0565b8051516003146101c45760405162461bcd60e51b815260206004820152600860248201527f505553485f4c454e00000000000000000000000000000000000000000000000060448201526064016100cf565b60005b8151518110156102b15760006101dd83836105d4565b90506000815160068111156101f4576101f4610734565b146102415760405162461bcd60e51b815260206004820152600d60248201527f505553485f56414c5f545950450000000000000000000000000000000000000060448201526064016100cf565b61024c826001610760565b81602001511461029e5760405162461bcd60e51b815260206004820152601160248201527f505553485f56414c5f434f4e54454e545300000000000000000000000000000060448201526064016100cf565b50806102a981610773565b9150506101c7565b5060006102bd8261060d565b90506000815160068111156102d4576102d4610734565b146103215760405162461bcd60e51b815260206004820152600c60248201527f504f505f5245545f54595045000000000000000000000000000000000000000060448201526064016100cf565b80602001516003146103755760405162461bcd60e51b815260206004820152601060248201527f504f505f5245545f434f4e54454e54530000000000000000000000000000000060448201526064016100cf565b8151516002146103c75760405162461bcd60e51b815260206004820152600760248201527f504f505f4c454e0000000000000000000000000000000000000000000000000060448201526064016100cf565b60005b8251518110156104b45760006103e084836105d4565b90506000815160068111156103f7576103f7610734565b146104445760405162461bcd60e51b815260206004820152600c60248201527f504f505f56414c5f54595045000000000000000000000000000000000000000060448201526064016100cf565b61044f826001610760565b8160200151146104a15760405162461bcd60e51b815260206004820152601060248201527f504f505f56414c5f434f4e54454e54530000000000000000000000000000000060448201526064016100cf565b50806104ac81610773565b9150506103ca565b505050565b80836000015183815181106104d0576104d06107ab565b6020026020010181905250505050565b8151516000906104f1906001610760565b67ffffffffffffffff8111156105095761050961071e565b60405190808252806020026020018201604052801561054e57816020015b60408051808201909152600080825260208201528152602001906001900390816105275790505b50905060005b8351518110156105aa578351805182908110610572576105726107ab565b602002602001015182828151811061058c5761058c6107ab565b602002602001018190525080806105a290610773565b915050610554565b508181846000015151815181106105c3576105c36107ab565b602090810291909101015290915250565b604080518082019091526000808252602082015282518051839081106105fc576105fc6107ab565b602002602001015190505b92915050565b604080518082019091526000808252602082015281518051610631906001906107c1565b81518110610641576106416107ab565b602002602001015190506000600183600001515161065f91906107c1565b67ffffffffffffffff8111156106775761067761071e565b6040519080825280602002602001820160405280156106bc57816020015b60408051808201909152600080825260208201528152602001906001900390816106955790505b50905060005b81518110156107175783518051829081106106df576106df6107ab565b60200260200101518282815181106106f9576106f96107ab565b6020026020010181905250808061070f90610773565b9150506106c2565b5090915290565b634e487b7160e01b600052604160045260246000fd5b634e487b7160e01b600052602160045260246000fd5b634e487b7160e01b600052601160045260246000fd5b808201808211156106075761060761074a565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036107a4576107a461074a565b5060010190565b634e487b7160e01b600052603260045260246000fd5b818103818111156106075761060761074a56fea264697066735822122014d2f7ea2b733ccb457fe08467c70539099e5bf463951c5af3ebc5dd93564fb264736f6c63430008110033",
}

// ValueArrayTesterABI is the input ABI used to generate the binding from.
// Deprecated: Use ValueArrayTesterMetaData.ABI instead.
var ValueArrayTesterABI = ValueArrayTesterMetaData.ABI

// ValueArrayTesterBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ValueArrayTesterMetaData.Bin instead.
var ValueArrayTesterBin = ValueArrayTesterMetaData.Bin

// DeployValueArrayTester deploys a new Ethereum contract, binding an instance of ValueArrayTester to it.
func DeployValueArrayTester(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ValueArrayTester, error) {
	parsed, err := ValueArrayTesterMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ValueArrayTesterBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ValueArrayTester{ValueArrayTesterCaller: ValueArrayTesterCaller{contract: contract}, ValueArrayTesterTransactor: ValueArrayTesterTransactor{contract: contract}, ValueArrayTesterFilterer: ValueArrayTesterFilterer{contract: contract}}, nil
}

// ValueArrayTester is an auto generated Go binding around an Ethereum contract.
type ValueArrayTester struct {
	ValueArrayTesterCaller     // Read-only binding to the contract
	ValueArrayTesterTransactor // Write-only binding to the contract
	ValueArrayTesterFilterer   // Log filterer for contract events
}

// ValueArrayTesterCaller is an auto generated read-only Go binding around an Ethereum contract.
type ValueArrayTesterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueArrayTesterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ValueArrayTesterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueArrayTesterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ValueArrayTesterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValueArrayTesterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ValueArrayTesterSession struct {
	Contract     *ValueArrayTester // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ValueArrayTesterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ValueArrayTesterCallerSession struct {
	Contract *ValueArrayTesterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// ValueArrayTesterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ValueArrayTesterTransactorSession struct {
	Contract     *ValueArrayTesterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// ValueArrayTesterRaw is an auto generated low-level Go binding around an Ethereum contract.
type ValueArrayTesterRaw struct {
	Contract *ValueArrayTester // Generic contract binding to access the raw methods on
}

// ValueArrayTesterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ValueArrayTesterCallerRaw struct {
	Contract *ValueArrayTesterCaller // Generic read-only contract binding to access the raw methods on
}

// ValueArrayTesterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ValueArrayTesterTransactorRaw struct {
	Contract *ValueArrayTesterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewValueArrayTester creates a new instance of ValueArrayTester, bound to a specific deployed contract.
func NewValueArrayTester(address common.Address, backend bind.ContractBackend) (*ValueArrayTester, error) {
	contract, err := bindValueArrayTester(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ValueArrayTester{ValueArrayTesterCaller: ValueArrayTesterCaller{contract: contract}, ValueArrayTesterTransactor: ValueArrayTesterTransactor{contract: contract}, ValueArrayTesterFilterer: ValueArrayTesterFilterer{contract: contract}}, nil
}

// NewValueArrayTesterCaller creates a new read-only instance of ValueArrayTester, bound to a specific deployed contract.
func NewValueArrayTesterCaller(address common.Address, caller bind.ContractCaller) (*ValueArrayTesterCaller, error) {
	contract, err := bindValueArrayTester(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ValueArrayTesterCaller{contract: contract}, nil
}

// NewValueArrayTesterTransactor creates a new write-only instance of ValueArrayTester, bound to a specific deployed contract.
func NewValueArrayTesterTransactor(address common.Address, transactor bind.ContractTransactor) (*ValueArrayTesterTransactor, error) {
	contract, err := bindValueArrayTester(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ValueArrayTesterTransactor{contract: contract}, nil
}

// NewValueArrayTesterFilterer creates a new log filterer instance of ValueArrayTester, bound to a specific deployed contract.
func NewValueArrayTesterFilterer(address common.Address, filterer bind.ContractFilterer) (*ValueArrayTesterFilterer, error) {
	contract, err := bindValueArrayTester(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ValueArrayTesterFilterer{contract: contract}, nil
}

// bindValueArrayTester binds a generic wrapper to an already deployed contract.
func bindValueArrayTester(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ValueArrayTesterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValueArrayTester *ValueArrayTesterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValueArrayTester.Contract.ValueArrayTesterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValueArrayTester *ValueArrayTesterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValueArrayTester.Contract.ValueArrayTesterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValueArrayTester *ValueArrayTesterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValueArrayTester.Contract.ValueArrayTesterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ValueArrayTester *ValueArrayTesterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ValueArrayTester.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ValueArrayTester *ValueArrayTesterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ValueArrayTester.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ValueArrayTester *ValueArrayTesterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ValueArrayTester.Contract.contract.Transact(opts, method, params...)
}

// Test is a free data retrieval call binding the contract method 0xf8a8fd6d.
//
// Solidity: function test() pure returns()
func (_ValueArrayTester *ValueArrayTesterCaller) Test(opts *bind.CallOpts) error {
	var out []interface{}
	err := _ValueArrayTester.contract.Call(opts, &out, "test")

	if err != nil {
		return err
	}

	return err

}

// Test is a free data retrieval call binding the contract method 0xf8a8fd6d.
//
// Solidity: function test() pure returns()
func (_ValueArrayTester *ValueArrayTesterSession) Test() error {
	return _ValueArrayTester.Contract.Test(&_ValueArrayTester.CallOpts)
}

// Test is a free data retrieval call binding the contract method 0xf8a8fd6d.
//
// Solidity: function test() pure returns()
func (_ValueArrayTester *ValueArrayTesterCallerSession) Test() error {
	return _ValueArrayTester.Contract.Test(&_ValueArrayTester.CallOpts)
}
