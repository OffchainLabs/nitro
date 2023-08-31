// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bridgegen

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

// BridgeMetaData contains all meta data concerning the Bridge contract.
var BridgeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"stored\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"received\",\"type\":\"uint256\"}],\"name\":\"BadSequencerMessageNumber\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"}],\"name\":\"InvalidOutboxSet\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"NotContract\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotDelayedInbox\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotOutbox\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotRollupOrOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotSequencerInbox\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"BridgeCallTriggered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"InboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeInboxAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"MessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"OutboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newSequencerInbox\",\"type\":\"address\"}],\"name\":\"SequencerInboxUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"acceptFundsFromOldBridge\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"activeOutbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedDelayedInboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"}],\"name\":\"allowedDelayedInboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedOutboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"}],\"name\":\"allowedOutboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"delayedInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"delayedMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"enqueueDelayedMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"enqueueSequencerMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"seqMessageIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"acc\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeCall\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"rollup_\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"sequencerInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerReportedSubMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setDelayedInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setOutbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sequencerInbox\",\"type\":\"address\"}],\"name\":\"setSequencerInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newMsgCount\",\"type\":\"uint256\"}],\"name\":\"setSequencerReportedSubMessageCount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"submitBatchSpendingReport\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"_rollup\",\"type\":\"address\"}],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a06040523060805234801561001457600080fd5b506080516119d961003760003960008181610b660152610f9401526119d96000f3fe60806040526004361061017f5760003560e01c80639e5d4c49116100d6578063d5719dc21161007f578063eca067ad11610059578063eca067ad14610457578063ee35f3271461046c578063f81ff3b31461048c57600080fd5b8063d5719dc214610417578063e76f5c8d14610437578063e77145f41461023457600080fd5b8063c4d66de8116100b0578063c4d66de8146103b7578063cb23bcb5146103d7578063cee3d728146103f757600080fd5b80639e5d4c4914610337578063ab5d894314610365578063ae60bd131461037a57600080fd5b80635fca4a16116101385780638db5993b116101125780638db5993b146102cc578063919cc706146102df578063945e1147146102ff57600080fd5b80635fca4a16146102565780637a88b1071461026c57806386598a561461028c57600080fd5b8063413b35bd11610169578063413b35bd146101c857806347fb24c5146102145780634f61f8501461023657600080fd5b806284120c1461018457806316bf5579146101a8575b600080fd5b34801561019057600080fd5b506007545b6040519081526020015b60405180910390f35b3480156101b457600080fd5b506101956101c33660046116da565b6104ac565b3480156101d457600080fd5b506102046101e336600461170b565b6001600160a01b031660009081526002602052604090206001015460ff1690565b604051901515815260200161019f565b34801561022057600080fd5b5061023461022f36600461172f565b6104cd565b005b34801561024257600080fd5b5061023461025136600461170b565b6107d3565b34801561026257600080fd5b50610195600a5481565b34801561027857600080fd5b5061019561028736600461176d565b6108fe565b34801561029857600080fd5b506102ac6102a7366004611799565b61095f565b60408051948552602085019390935291830152606082015260800161019f565b6101956102da3660046117cb565b610af8565b3480156102eb57600080fd5b506102346102fa36600461170b565b610b5c565b34801561030b57600080fd5b5061031f61031a3660046116da565b610ca1565b6040516001600160a01b03909116815260200161019f565b34801561034357600080fd5b50610357610352366004611812565b610ccb565b60405161019f92919061189b565b34801561037157600080fd5b5061031f610e78565b34801561038657600080fd5b5061020461039536600461170b565b6001600160a01b03166000908152600160208190526040909120015460ff1690565b3480156103c357600080fd5b506102346103d236600461170b565b610ebb565b3480156103e357600080fd5b5060085461031f906001600160a01b031681565b34801561040357600080fd5b5061023461041236600461172f565b6110ab565b34801561042357600080fd5b506101956104323660046116da565b611419565b34801561044357600080fd5b5061031f6104523660046116da565b611429565b34801561046357600080fd5b50600654610195565b34801561047857600080fd5b5060095461031f906001600160a01b031681565b34801561049857600080fd5b506102346104a73660046116da565b611439565b600781815481106104bc57600080fd5b600091825260209091200154905081565b6008546001600160a01b0316331461059c5760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015610529573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061054d91906118f3565b9050336001600160a01b0382161461059a57600854604051630739600760e01b81523360048201526001600160a01b03918216602482015290821660448201526064015b60405180910390fd5b505b6001600160a01b0382166000818152600160208181526040928390209182015492518515158152919360ff90931692917f6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521910160405180910390a2821515811515036106085750505050565b82156106a357604080518082018252600380548252600160208084018281526001600160a01b038a166000818152928490529582209451855551938201805460ff1916941515949094179093558154908101825591527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b01805473ffffffffffffffffffffffffffffffffffffffff191690911790556107cc565b600380546106b390600190611910565b815481106106c3576106c3611931565b6000918252602090912001548254600380546001600160a01b039093169290919081106106f2576106f2611931565b9060005260206000200160006101000a8154816001600160a01b0302191690836001600160a01b03160217905550816000015460016000600385600001548154811061074057610740611931565b60009182526020808320909101546001600160a01b03168352820192909252604001902055600380548061077657610776611947565b600082815260208082208301600019908101805473ffffffffffffffffffffffffffffffffffffffff191690559092019092556001600160a01b03861682526001908190526040822091825501805460ff191690555b50505b5050565b6008546001600160a01b0316331461089d5760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa15801561082f573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061085391906118f3565b9050336001600160a01b0382161461089b57600854604051630739600760e01b81523360048201526001600160a01b0391821660248201529082166044820152606401610591565b505b6009805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0383169081179091556040519081527f8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a9060200160405180910390a150565b6009546000906001600160a01b03163314610947576040517f88f84f04000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b610956600d8443424887611508565b90505b92915050565b6009546000908190819081906001600160a01b031633146109ae576040517f88f84f04000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b85600a54141580156109bf57508515155b80156109cc5750600a5415155b15610a1157600a546040517fe2051feb000000000000000000000000000000000000000000000000000000008152600481019190915260248101879052604401610591565b600a85905560075493508315610a4f5760078054610a3190600190611910565b81548110610a4157610a41611931565b906000526020600020015492505b8615610a80576006610a62600189611910565b81548110610a7257610a72611931565b906000526020600020015491505b60408051602081018590529081018990526060810183905260800160408051601f198184030181529190528051602090910120600780546001810182556000919091527fa66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c688018190559398929750909550919350915050565b3360009081526001602081905260408220015460ff16610b46576040517fb6c60ea3000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b610b54848443424887611508565b949350505050565b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610bfa5760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c00000000000000000000000000000000000000006064820152608401610591565b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038054336001600160a01b03821614610c70576040517f23295f0e0000000000000000000000000000000000000000000000000000000081523360048201526001600160a01b0382166024820152604401610591565b50506008805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0392909216919091179055565b60048181548110610cb157600080fd5b6000918252602090912001546001600160a01b0316905081565b3360009081526002602052604081206001015460609060ff16610d1c576040517f32ea82ab000000000000000000000000000000000000000000000000000000008152336004820152602401610591565b8215801590610d3357506001600160a01b0386163b155b15610d75576040517fb5cf5b8f0000000000000000000000000000000000000000000000000000000081526001600160a01b0387166004820152602401610591565b6005805473ffffffffffffffffffffffffffffffffffffffff19811633179091556040516001600160a01b03918216918816908790610db7908890889061195d565b60006040518083038185875af1925050503d8060008114610df4576040519150601f19603f3d011682016040523d82523d6000602084013e610df9565b606091505b506005805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038581169190911790915560405192955090935088169033907f2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d46690610e66908a908a908a9061196d565b60405180910390a35094509492505050565b6005546000906001600160a01b03167fffffffffffffffffffffffff00000000000000000000000000000000000000018101610eb657600091505090565b919050565b600054610100900460ff1615808015610edb5750600054600160ff909116105b80610ef55750303b158015610ef5575060005460ff166001145b610f675760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a65640000000000000000000000000000000000006064820152608401610591565b6000805460ff191660011790558015610f8a576000805461ff0019166101001790555b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036110285760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c00000000000000000000000000000000000000006064820152608401610591565b600580546001600160a01b0373ffffffffffffffffffffffffffffffffffffffff1991821681179092556008805490911691841691909117905580156107cf576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050565b6008546001600160a01b031633146111755760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015611107573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061112b91906118f3565b9050336001600160a01b0382161461117357600854604051630739600760e01b81523360048201526001600160a01b0391821660248201529082166044820152606401610591565b505b7fffffffffffffffffffffffff00000000000000000000000000000000000000016001600160a01b038316016111e2576040517f77abed100000000000000000000000000000000000000000000000000000000081526001600160a01b0383166004820152602401610591565b6001600160a01b038216600081815260026020908152604091829020600181015492518515158152909360ff90931692917f49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa910160405180910390a28215158115150361124f5750505050565b82156112eb57604080518082018252600480548252600160208084018281526001600160a01b038a16600081815260029093529582209451855551938201805460ff1916941515949094179093558154908101825591527f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b01805473ffffffffffffffffffffffffffffffffffffffff191690911790556107cc565b600480546112fb90600190611910565b8154811061130b5761130b611931565b6000918252602090912001548254600480546001600160a01b0390931692909190811061133a5761133a611931565b9060005260206000200160006101000a8154816001600160a01b0302191690836001600160a01b03160217905550816000015460026000600485600001548154811061138857611388611931565b60009182526020808320909101546001600160a01b0316835282019290925260400190205560048054806113be576113be611947565b600082815260208082208301600019908101805473ffffffffffffffffffffffffffffffffffffffff191690559092019092556001600160a01b03861682526002905260408120908155600101805460ff1916905550505050565b600681815481106104bc57600080fd5b60038181548110610cb157600080fd5b6008546001600160a01b031633146115035760085460408051638da5cb5b60e01b815290516000926001600160a01b031691638da5cb5b9160048083019260209291908290030181865afa158015611495573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114b991906118f3565b9050336001600160a01b0382161461150157600854604051630739600760e01b81523360048201526001600160a01b0391821660248201529082166044820152606401610591565b505b600a55565b600654604080517fff0000000000000000000000000000000000000000000000000000000000000060f88a901b166020808301919091527fffffffffffffffffffffffffffffffffffffffff00000000000000000000000060608a901b1660218301527fffffffffffffffff00000000000000000000000000000000000000000000000060c089811b8216603585015288901b16603d830152604582018490526065820186905260858083018690528351808403909101815260a5909201909252805191012060009190600082156116055760066115e7600185611910565b815481106115f7576115f7611931565b906000526020600020015490505b6040805160208082018490528183018590528251808303840181526060830180855281519190920120600680546001810182556000919091527ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f015533905260ff8c1660808201526001600160a01b038b1660a082015260c0810187905260e0810188905267ffffffffffffffff89166101008201529051829185917f5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1918190036101200190a3509098975050505050505050565b6000602082840312156116ec57600080fd5b5035919050565b6001600160a01b038116811461170857600080fd5b50565b60006020828403121561171d57600080fd5b8135611728816116f3565b9392505050565b6000806040838503121561174257600080fd5b823561174d816116f3565b91506020830135801515811461176257600080fd5b809150509250929050565b6000806040838503121561178057600080fd5b823561178b816116f3565b946020939093013593505050565b600080600080608085870312156117af57600080fd5b5050823594602084013594506040840135936060013592509050565b6000806000606084860312156117e057600080fd5b833560ff811681146117f157600080fd5b92506020840135611801816116f3565b929592945050506040919091013590565b6000806000806060858703121561182857600080fd5b8435611833816116f3565b935060208501359250604085013567ffffffffffffffff8082111561185757600080fd5b818701915087601f83011261186b57600080fd5b81358181111561187a57600080fd5b88602082850101111561188c57600080fd5b95989497505060200194505050565b821515815260006020604081840152835180604085015260005b818110156118d1578581018301518582016060015282016118b5565b506000606082860101526060601f19601f830116850101925050509392505050565b60006020828403121561190557600080fd5b8151611728816116f3565b8181038181111561095957634e487b7160e01b600052601160045260246000fd5b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052603160045260246000fd5b8183823760009101908152919050565b83815260406020820152816040820152818360608301376000818301606090810191909152601f909201601f191601019291505056fea2646970667358221220e17c4fa516aef1025bbd9042f6a81bce4320d01116e045245bb984afe41977fb64736f6c63430008110033",
}

// BridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use BridgeMetaData.ABI instead.
var BridgeABI = BridgeMetaData.ABI

// BridgeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BridgeMetaData.Bin instead.
var BridgeBin = BridgeMetaData.Bin

// DeployBridge deploys a new Ethereum contract, binding an instance of Bridge to it.
func DeployBridge(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Bridge, error) {
	parsed, err := BridgeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BridgeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Bridge{BridgeCaller: BridgeCaller{contract: contract}, BridgeTransactor: BridgeTransactor{contract: contract}, BridgeFilterer: BridgeFilterer{contract: contract}}, nil
}

// Bridge is an auto generated Go binding around an Ethereum contract.
type Bridge struct {
	BridgeCaller     // Read-only binding to the contract
	BridgeTransactor // Write-only binding to the contract
	BridgeFilterer   // Log filterer for contract events
}

// BridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type BridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BridgeSession struct {
	Contract     *Bridge           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BridgeCallerSession struct {
	Contract *BridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// BridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BridgeTransactorSession struct {
	Contract     *BridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type BridgeRaw struct {
	Contract *Bridge // Generic contract binding to access the raw methods on
}

// BridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BridgeCallerRaw struct {
	Contract *BridgeCaller // Generic read-only contract binding to access the raw methods on
}

// BridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BridgeTransactorRaw struct {
	Contract *BridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBridge creates a new instance of Bridge, bound to a specific deployed contract.
func NewBridge(address common.Address, backend bind.ContractBackend) (*Bridge, error) {
	contract, err := bindBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Bridge{BridgeCaller: BridgeCaller{contract: contract}, BridgeTransactor: BridgeTransactor{contract: contract}, BridgeFilterer: BridgeFilterer{contract: contract}}, nil
}

// NewBridgeCaller creates a new read-only instance of Bridge, bound to a specific deployed contract.
func NewBridgeCaller(address common.Address, caller bind.ContractCaller) (*BridgeCaller, error) {
	contract, err := bindBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeCaller{contract: contract}, nil
}

// NewBridgeTransactor creates a new write-only instance of Bridge, bound to a specific deployed contract.
func NewBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*BridgeTransactor, error) {
	contract, err := bindBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeTransactor{contract: contract}, nil
}

// NewBridgeFilterer creates a new log filterer instance of Bridge, bound to a specific deployed contract.
func NewBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*BridgeFilterer, error) {
	contract, err := bindBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BridgeFilterer{contract: contract}, nil
}

// bindBridge binds a generic wrapper to an already deployed contract.
func bindBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Bridge *BridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Bridge.Contract.BridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Bridge *BridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bridge.Contract.BridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Bridge *BridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Bridge.Contract.BridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Bridge *BridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Bridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Bridge *BridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Bridge *BridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Bridge.Contract.contract.Transact(opts, method, params...)
}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_Bridge *BridgeCaller) ActiveOutbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "activeOutbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_Bridge *BridgeSession) ActiveOutbox() (common.Address, error) {
	return _Bridge.Contract.ActiveOutbox(&_Bridge.CallOpts)
}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_Bridge *BridgeCallerSession) ActiveOutbox() (common.Address, error) {
	return _Bridge.Contract.ActiveOutbox(&_Bridge.CallOpts)
}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_Bridge *BridgeCaller) AllowedDelayedInboxList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "allowedDelayedInboxList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_Bridge *BridgeSession) AllowedDelayedInboxList(arg0 *big.Int) (common.Address, error) {
	return _Bridge.Contract.AllowedDelayedInboxList(&_Bridge.CallOpts, arg0)
}

// AllowedDelayedInboxList is a free data retrieval call binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) view returns(address)
func (_Bridge *BridgeCallerSession) AllowedDelayedInboxList(arg0 *big.Int) (common.Address, error) {
	return _Bridge.Contract.AllowedDelayedInboxList(&_Bridge.CallOpts, arg0)
}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_Bridge *BridgeCaller) AllowedDelayedInboxes(opts *bind.CallOpts, inbox common.Address) (bool, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "allowedDelayedInboxes", inbox)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_Bridge *BridgeSession) AllowedDelayedInboxes(inbox common.Address) (bool, error) {
	return _Bridge.Contract.AllowedDelayedInboxes(&_Bridge.CallOpts, inbox)
}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_Bridge *BridgeCallerSession) AllowedDelayedInboxes(inbox common.Address) (bool, error) {
	return _Bridge.Contract.AllowedDelayedInboxes(&_Bridge.CallOpts, inbox)
}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_Bridge *BridgeCaller) AllowedOutboxList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "allowedOutboxList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_Bridge *BridgeSession) AllowedOutboxList(arg0 *big.Int) (common.Address, error) {
	return _Bridge.Contract.AllowedOutboxList(&_Bridge.CallOpts, arg0)
}

// AllowedOutboxList is a free data retrieval call binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) view returns(address)
func (_Bridge *BridgeCallerSession) AllowedOutboxList(arg0 *big.Int) (common.Address, error) {
	return _Bridge.Contract.AllowedOutboxList(&_Bridge.CallOpts, arg0)
}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_Bridge *BridgeCaller) AllowedOutboxes(opts *bind.CallOpts, outbox common.Address) (bool, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "allowedOutboxes", outbox)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_Bridge *BridgeSession) AllowedOutboxes(outbox common.Address) (bool, error) {
	return _Bridge.Contract.AllowedOutboxes(&_Bridge.CallOpts, outbox)
}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_Bridge *BridgeCallerSession) AllowedOutboxes(outbox common.Address) (bool, error) {
	return _Bridge.Contract.AllowedOutboxes(&_Bridge.CallOpts, outbox)
}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_Bridge *BridgeCaller) DelayedInboxAccs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "delayedInboxAccs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_Bridge *BridgeSession) DelayedInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _Bridge.Contract.DelayedInboxAccs(&_Bridge.CallOpts, arg0)
}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_Bridge *BridgeCallerSession) DelayedInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _Bridge.Contract.DelayedInboxAccs(&_Bridge.CallOpts, arg0)
}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_Bridge *BridgeCaller) DelayedMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "delayedMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_Bridge *BridgeSession) DelayedMessageCount() (*big.Int, error) {
	return _Bridge.Contract.DelayedMessageCount(&_Bridge.CallOpts)
}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_Bridge *BridgeCallerSession) DelayedMessageCount() (*big.Int, error) {
	return _Bridge.Contract.DelayedMessageCount(&_Bridge.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_Bridge *BridgeCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_Bridge *BridgeSession) Rollup() (common.Address, error) {
	return _Bridge.Contract.Rollup(&_Bridge.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_Bridge *BridgeCallerSession) Rollup() (common.Address, error) {
	return _Bridge.Contract.Rollup(&_Bridge.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_Bridge *BridgeCaller) SequencerInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "sequencerInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_Bridge *BridgeSession) SequencerInbox() (common.Address, error) {
	return _Bridge.Contract.SequencerInbox(&_Bridge.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_Bridge *BridgeCallerSession) SequencerInbox() (common.Address, error) {
	return _Bridge.Contract.SequencerInbox(&_Bridge.CallOpts)
}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_Bridge *BridgeCaller) SequencerInboxAccs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "sequencerInboxAccs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_Bridge *BridgeSession) SequencerInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _Bridge.Contract.SequencerInboxAccs(&_Bridge.CallOpts, arg0)
}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_Bridge *BridgeCallerSession) SequencerInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _Bridge.Contract.SequencerInboxAccs(&_Bridge.CallOpts, arg0)
}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_Bridge *BridgeCaller) SequencerMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "sequencerMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_Bridge *BridgeSession) SequencerMessageCount() (*big.Int, error) {
	return _Bridge.Contract.SequencerMessageCount(&_Bridge.CallOpts)
}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_Bridge *BridgeCallerSession) SequencerMessageCount() (*big.Int, error) {
	return _Bridge.Contract.SequencerMessageCount(&_Bridge.CallOpts)
}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_Bridge *BridgeCaller) SequencerReportedSubMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Bridge.contract.Call(opts, &out, "sequencerReportedSubMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_Bridge *BridgeSession) SequencerReportedSubMessageCount() (*big.Int, error) {
	return _Bridge.Contract.SequencerReportedSubMessageCount(&_Bridge.CallOpts)
}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_Bridge *BridgeCallerSession) SequencerReportedSubMessageCount() (*big.Int, error) {
	return _Bridge.Contract.SequencerReportedSubMessageCount(&_Bridge.CallOpts)
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_Bridge *BridgeTransactor) AcceptFundsFromOldBridge(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "acceptFundsFromOldBridge")
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_Bridge *BridgeSession) AcceptFundsFromOldBridge() (*types.Transaction, error) {
	return _Bridge.Contract.AcceptFundsFromOldBridge(&_Bridge.TransactOpts)
}

// AcceptFundsFromOldBridge is a paid mutator transaction binding the contract method 0xe77145f4.
//
// Solidity: function acceptFundsFromOldBridge() payable returns()
func (_Bridge *BridgeTransactorSession) AcceptFundsFromOldBridge() (*types.Transaction, error) {
	return _Bridge.Contract.AcceptFundsFromOldBridge(&_Bridge.TransactOpts)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_Bridge *BridgeTransactor) EnqueueDelayedMessage(opts *bind.TransactOpts, kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "enqueueDelayedMessage", kind, sender, messageDataHash)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_Bridge *BridgeSession) EnqueueDelayedMessage(kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _Bridge.Contract.EnqueueDelayedMessage(&_Bridge.TransactOpts, kind, sender, messageDataHash)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_Bridge *BridgeTransactorSession) EnqueueDelayedMessage(kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _Bridge.Contract.EnqueueDelayedMessage(&_Bridge.TransactOpts, kind, sender, messageDataHash)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_Bridge *BridgeTransactor) EnqueueSequencerMessage(opts *bind.TransactOpts, dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "enqueueSequencerMessage", dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_Bridge *BridgeSession) EnqueueSequencerMessage(dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _Bridge.Contract.EnqueueSequencerMessage(&_Bridge.TransactOpts, dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_Bridge *BridgeTransactorSession) EnqueueSequencerMessage(dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _Bridge.Contract.EnqueueSequencerMessage(&_Bridge.TransactOpts, dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_Bridge *BridgeTransactor) ExecuteCall(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "executeCall", to, value, data)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_Bridge *BridgeSession) ExecuteCall(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Bridge.Contract.ExecuteCall(&_Bridge.TransactOpts, to, value, data)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_Bridge *BridgeTransactorSession) ExecuteCall(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Bridge.Contract.ExecuteCall(&_Bridge.TransactOpts, to, value, data)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_Bridge *BridgeTransactor) Initialize(opts *bind.TransactOpts, rollup_ common.Address) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "initialize", rollup_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_Bridge *BridgeSession) Initialize(rollup_ common.Address) (*types.Transaction, error) {
	return _Bridge.Contract.Initialize(&_Bridge.TransactOpts, rollup_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_Bridge *BridgeTransactorSession) Initialize(rollup_ common.Address) (*types.Transaction, error) {
	return _Bridge.Contract.Initialize(&_Bridge.TransactOpts, rollup_)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_Bridge *BridgeTransactor) SetDelayedInbox(opts *bind.TransactOpts, inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "setDelayedInbox", inbox, enabled)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_Bridge *BridgeSession) SetDelayedInbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _Bridge.Contract.SetDelayedInbox(&_Bridge.TransactOpts, inbox, enabled)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_Bridge *BridgeTransactorSession) SetDelayedInbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _Bridge.Contract.SetDelayedInbox(&_Bridge.TransactOpts, inbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address outbox, bool enabled) returns()
func (_Bridge *BridgeTransactor) SetOutbox(opts *bind.TransactOpts, outbox common.Address, enabled bool) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "setOutbox", outbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address outbox, bool enabled) returns()
func (_Bridge *BridgeSession) SetOutbox(outbox common.Address, enabled bool) (*types.Transaction, error) {
	return _Bridge.Contract.SetOutbox(&_Bridge.TransactOpts, outbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address outbox, bool enabled) returns()
func (_Bridge *BridgeTransactorSession) SetOutbox(outbox common.Address, enabled bool) (*types.Transaction, error) {
	return _Bridge.Contract.SetOutbox(&_Bridge.TransactOpts, outbox, enabled)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_Bridge *BridgeTransactor) SetSequencerInbox(opts *bind.TransactOpts, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "setSequencerInbox", _sequencerInbox)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_Bridge *BridgeSession) SetSequencerInbox(_sequencerInbox common.Address) (*types.Transaction, error) {
	return _Bridge.Contract.SetSequencerInbox(&_Bridge.TransactOpts, _sequencerInbox)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_Bridge *BridgeTransactorSession) SetSequencerInbox(_sequencerInbox common.Address) (*types.Transaction, error) {
	return _Bridge.Contract.SetSequencerInbox(&_Bridge.TransactOpts, _sequencerInbox)
}

// SetSequencerReportedSubMessageCount is a paid mutator transaction binding the contract method 0xf81ff3b3.
//
// Solidity: function setSequencerReportedSubMessageCount(uint256 newMsgCount) returns()
func (_Bridge *BridgeTransactor) SetSequencerReportedSubMessageCount(opts *bind.TransactOpts, newMsgCount *big.Int) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "setSequencerReportedSubMessageCount", newMsgCount)
}

// SetSequencerReportedSubMessageCount is a paid mutator transaction binding the contract method 0xf81ff3b3.
//
// Solidity: function setSequencerReportedSubMessageCount(uint256 newMsgCount) returns()
func (_Bridge *BridgeSession) SetSequencerReportedSubMessageCount(newMsgCount *big.Int) (*types.Transaction, error) {
	return _Bridge.Contract.SetSequencerReportedSubMessageCount(&_Bridge.TransactOpts, newMsgCount)
}

// SetSequencerReportedSubMessageCount is a paid mutator transaction binding the contract method 0xf81ff3b3.
//
// Solidity: function setSequencerReportedSubMessageCount(uint256 newMsgCount) returns()
func (_Bridge *BridgeTransactorSession) SetSequencerReportedSubMessageCount(newMsgCount *big.Int) (*types.Transaction, error) {
	return _Bridge.Contract.SetSequencerReportedSubMessageCount(&_Bridge.TransactOpts, newMsgCount)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address sender, bytes32 messageDataHash) returns(uint256)
func (_Bridge *BridgeTransactor) SubmitBatchSpendingReport(opts *bind.TransactOpts, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "submitBatchSpendingReport", sender, messageDataHash)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address sender, bytes32 messageDataHash) returns(uint256)
func (_Bridge *BridgeSession) SubmitBatchSpendingReport(sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _Bridge.Contract.SubmitBatchSpendingReport(&_Bridge.TransactOpts, sender, messageDataHash)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address sender, bytes32 messageDataHash) returns(uint256)
func (_Bridge *BridgeTransactorSession) SubmitBatchSpendingReport(sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _Bridge.Contract.SubmitBatchSpendingReport(&_Bridge.TransactOpts, sender, messageDataHash)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_Bridge *BridgeTransactor) UpdateRollupAddress(opts *bind.TransactOpts, _rollup common.Address) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "updateRollupAddress", _rollup)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_Bridge *BridgeSession) UpdateRollupAddress(_rollup common.Address) (*types.Transaction, error) {
	return _Bridge.Contract.UpdateRollupAddress(&_Bridge.TransactOpts, _rollup)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_Bridge *BridgeTransactorSession) UpdateRollupAddress(_rollup common.Address) (*types.Transaction, error) {
	return _Bridge.Contract.UpdateRollupAddress(&_Bridge.TransactOpts, _rollup)
}

// BridgeBridgeCallTriggeredIterator is returned from FilterBridgeCallTriggered and is used to iterate over the raw logs and unpacked data for BridgeCallTriggered events raised by the Bridge contract.
type BridgeBridgeCallTriggeredIterator struct {
	Event *BridgeBridgeCallTriggered // Event containing the contract specifics and raw log

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
func (it *BridgeBridgeCallTriggeredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeBridgeCallTriggered)
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
		it.Event = new(BridgeBridgeCallTriggered)
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
func (it *BridgeBridgeCallTriggeredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeBridgeCallTriggeredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeBridgeCallTriggered represents a BridgeCallTriggered event raised by the Bridge contract.
type BridgeBridgeCallTriggered struct {
	Outbox common.Address
	To     common.Address
	Value  *big.Int
	Data   []byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBridgeCallTriggered is a free log retrieval operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_Bridge *BridgeFilterer) FilterBridgeCallTriggered(opts *bind.FilterOpts, outbox []common.Address, to []common.Address) (*BridgeBridgeCallTriggeredIterator, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "BridgeCallTriggered", outboxRule, toRule)
	if err != nil {
		return nil, err
	}
	return &BridgeBridgeCallTriggeredIterator{contract: _Bridge.contract, event: "BridgeCallTriggered", logs: logs, sub: sub}, nil
}

// WatchBridgeCallTriggered is a free log subscription operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_Bridge *BridgeFilterer) WatchBridgeCallTriggered(opts *bind.WatchOpts, sink chan<- *BridgeBridgeCallTriggered, outbox []common.Address, to []common.Address) (event.Subscription, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "BridgeCallTriggered", outboxRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeBridgeCallTriggered)
				if err := _Bridge.contract.UnpackLog(event, "BridgeCallTriggered", log); err != nil {
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
func (_Bridge *BridgeFilterer) ParseBridgeCallTriggered(log types.Log) (*BridgeBridgeCallTriggered, error) {
	event := new(BridgeBridgeCallTriggered)
	if err := _Bridge.contract.UnpackLog(event, "BridgeCallTriggered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeInboxToggleIterator is returned from FilterInboxToggle and is used to iterate over the raw logs and unpacked data for InboxToggle events raised by the Bridge contract.
type BridgeInboxToggleIterator struct {
	Event *BridgeInboxToggle // Event containing the contract specifics and raw log

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
func (it *BridgeInboxToggleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeInboxToggle)
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
		it.Event = new(BridgeInboxToggle)
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
func (it *BridgeInboxToggleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeInboxToggleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeInboxToggle represents a InboxToggle event raised by the Bridge contract.
type BridgeInboxToggle struct {
	Inbox   common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInboxToggle is a free log retrieval operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_Bridge *BridgeFilterer) FilterInboxToggle(opts *bind.FilterOpts, inbox []common.Address) (*BridgeInboxToggleIterator, error) {

	var inboxRule []interface{}
	for _, inboxItem := range inbox {
		inboxRule = append(inboxRule, inboxItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "InboxToggle", inboxRule)
	if err != nil {
		return nil, err
	}
	return &BridgeInboxToggleIterator{contract: _Bridge.contract, event: "InboxToggle", logs: logs, sub: sub}, nil
}

// WatchInboxToggle is a free log subscription operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_Bridge *BridgeFilterer) WatchInboxToggle(opts *bind.WatchOpts, sink chan<- *BridgeInboxToggle, inbox []common.Address) (event.Subscription, error) {

	var inboxRule []interface{}
	for _, inboxItem := range inbox {
		inboxRule = append(inboxRule, inboxItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "InboxToggle", inboxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeInboxToggle)
				if err := _Bridge.contract.UnpackLog(event, "InboxToggle", log); err != nil {
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
func (_Bridge *BridgeFilterer) ParseInboxToggle(log types.Log) (*BridgeInboxToggle, error) {
	event := new(BridgeInboxToggle)
	if err := _Bridge.contract.UnpackLog(event, "InboxToggle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Bridge contract.
type BridgeInitializedIterator struct {
	Event *BridgeInitialized // Event containing the contract specifics and raw log

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
func (it *BridgeInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeInitialized)
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
		it.Event = new(BridgeInitialized)
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
func (it *BridgeInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeInitialized represents a Initialized event raised by the Bridge contract.
type BridgeInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_Bridge *BridgeFilterer) FilterInitialized(opts *bind.FilterOpts) (*BridgeInitializedIterator, error) {

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &BridgeInitializedIterator{contract: _Bridge.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_Bridge *BridgeFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *BridgeInitialized) (event.Subscription, error) {

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeInitialized)
				if err := _Bridge.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_Bridge *BridgeFilterer) ParseInitialized(log types.Log) (*BridgeInitialized, error) {
	event := new(BridgeInitialized)
	if err := _Bridge.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeMessageDeliveredIterator is returned from FilterMessageDelivered and is used to iterate over the raw logs and unpacked data for MessageDelivered events raised by the Bridge contract.
type BridgeMessageDeliveredIterator struct {
	Event *BridgeMessageDelivered // Event containing the contract specifics and raw log

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
func (it *BridgeMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeMessageDelivered)
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
		it.Event = new(BridgeMessageDelivered)
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
func (it *BridgeMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeMessageDelivered represents a MessageDelivered event raised by the Bridge contract.
type BridgeMessageDelivered struct {
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
func (_Bridge *BridgeFilterer) FilterMessageDelivered(opts *bind.FilterOpts, messageIndex []*big.Int, beforeInboxAcc [][32]byte) (*BridgeMessageDeliveredIterator, error) {

	var messageIndexRule []interface{}
	for _, messageIndexItem := range messageIndex {
		messageIndexRule = append(messageIndexRule, messageIndexItem)
	}
	var beforeInboxAccRule []interface{}
	for _, beforeInboxAccItem := range beforeInboxAcc {
		beforeInboxAccRule = append(beforeInboxAccRule, beforeInboxAccItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "MessageDelivered", messageIndexRule, beforeInboxAccRule)
	if err != nil {
		return nil, err
	}
	return &BridgeMessageDeliveredIterator{contract: _Bridge.contract, event: "MessageDelivered", logs: logs, sub: sub}, nil
}

// WatchMessageDelivered is a free log subscription operation binding the contract event 0x5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1.
//
// Solidity: event MessageDelivered(uint256 indexed messageIndex, bytes32 indexed beforeInboxAcc, address inbox, uint8 kind, address sender, bytes32 messageDataHash, uint256 baseFeeL1, uint64 timestamp)
func (_Bridge *BridgeFilterer) WatchMessageDelivered(opts *bind.WatchOpts, sink chan<- *BridgeMessageDelivered, messageIndex []*big.Int, beforeInboxAcc [][32]byte) (event.Subscription, error) {

	var messageIndexRule []interface{}
	for _, messageIndexItem := range messageIndex {
		messageIndexRule = append(messageIndexRule, messageIndexItem)
	}
	var beforeInboxAccRule []interface{}
	for _, beforeInboxAccItem := range beforeInboxAcc {
		beforeInboxAccRule = append(beforeInboxAccRule, beforeInboxAccItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "MessageDelivered", messageIndexRule, beforeInboxAccRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeMessageDelivered)
				if err := _Bridge.contract.UnpackLog(event, "MessageDelivered", log); err != nil {
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
func (_Bridge *BridgeFilterer) ParseMessageDelivered(log types.Log) (*BridgeMessageDelivered, error) {
	event := new(BridgeMessageDelivered)
	if err := _Bridge.contract.UnpackLog(event, "MessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeOutboxToggleIterator is returned from FilterOutboxToggle and is used to iterate over the raw logs and unpacked data for OutboxToggle events raised by the Bridge contract.
type BridgeOutboxToggleIterator struct {
	Event *BridgeOutboxToggle // Event containing the contract specifics and raw log

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
func (it *BridgeOutboxToggleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeOutboxToggle)
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
		it.Event = new(BridgeOutboxToggle)
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
func (it *BridgeOutboxToggleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeOutboxToggleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeOutboxToggle represents a OutboxToggle event raised by the Bridge contract.
type BridgeOutboxToggle struct {
	Outbox  common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOutboxToggle is a free log retrieval operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_Bridge *BridgeFilterer) FilterOutboxToggle(opts *bind.FilterOpts, outbox []common.Address) (*BridgeOutboxToggleIterator, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "OutboxToggle", outboxRule)
	if err != nil {
		return nil, err
	}
	return &BridgeOutboxToggleIterator{contract: _Bridge.contract, event: "OutboxToggle", logs: logs, sub: sub}, nil
}

// WatchOutboxToggle is a free log subscription operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_Bridge *BridgeFilterer) WatchOutboxToggle(opts *bind.WatchOpts, sink chan<- *BridgeOutboxToggle, outbox []common.Address) (event.Subscription, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "OutboxToggle", outboxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeOutboxToggle)
				if err := _Bridge.contract.UnpackLog(event, "OutboxToggle", log); err != nil {
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
func (_Bridge *BridgeFilterer) ParseOutboxToggle(log types.Log) (*BridgeOutboxToggle, error) {
	event := new(BridgeOutboxToggle)
	if err := _Bridge.contract.UnpackLog(event, "OutboxToggle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeSequencerInboxUpdatedIterator is returned from FilterSequencerInboxUpdated and is used to iterate over the raw logs and unpacked data for SequencerInboxUpdated events raised by the Bridge contract.
type BridgeSequencerInboxUpdatedIterator struct {
	Event *BridgeSequencerInboxUpdated // Event containing the contract specifics and raw log

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
func (it *BridgeSequencerInboxUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeSequencerInboxUpdated)
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
		it.Event = new(BridgeSequencerInboxUpdated)
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
func (it *BridgeSequencerInboxUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeSequencerInboxUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeSequencerInboxUpdated represents a SequencerInboxUpdated event raised by the Bridge contract.
type BridgeSequencerInboxUpdated struct {
	NewSequencerInbox common.Address
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterSequencerInboxUpdated is a free log retrieval operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_Bridge *BridgeFilterer) FilterSequencerInboxUpdated(opts *bind.FilterOpts) (*BridgeSequencerInboxUpdatedIterator, error) {

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "SequencerInboxUpdated")
	if err != nil {
		return nil, err
	}
	return &BridgeSequencerInboxUpdatedIterator{contract: _Bridge.contract, event: "SequencerInboxUpdated", logs: logs, sub: sub}, nil
}

// WatchSequencerInboxUpdated is a free log subscription operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_Bridge *BridgeFilterer) WatchSequencerInboxUpdated(opts *bind.WatchOpts, sink chan<- *BridgeSequencerInboxUpdated) (event.Subscription, error) {

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "SequencerInboxUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeSequencerInboxUpdated)
				if err := _Bridge.contract.UnpackLog(event, "SequencerInboxUpdated", log); err != nil {
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
func (_Bridge *BridgeFilterer) ParseSequencerInboxUpdated(log types.Log) (*BridgeSequencerInboxUpdated, error) {
	event := new(BridgeSequencerInboxUpdated)
	if err := _Bridge.contract.UnpackLog(event, "SequencerInboxUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IBridgeMetaData contains all meta data concerning the IBridge contract.
var IBridgeMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"BridgeCallTriggered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"InboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeInboxAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"MessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"OutboxToggle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newSequencerInbox\",\"type\":\"address\"}],\"name\":\"SequencerInboxUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"activeOutbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedDelayedInboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"}],\"name\":\"allowedDelayedInboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allowedOutboxList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"outbox\",\"type\":\"address\"}],\"name\":\"allowedOutboxes\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"delayedInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"delayedMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"enqueueDelayedMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"enqueueSequencerMessage\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"seqMessageIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"acc\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeCall\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"rollup_\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"sequencerInboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerReportedSubMessageCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setDelayedInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"inbox\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setOutbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sequencerInbox\",\"type\":\"address\"}],\"name\":\"setSequencerInbox\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"batchPoster\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"}],\"name\":\"submitBatchSpendingReport\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"msgNum\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"_rollup\",\"type\":\"address\"}],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IBridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use IBridgeMetaData.ABI instead.
var IBridgeABI = IBridgeMetaData.ABI

// IBridge is an auto generated Go binding around an Ethereum contract.
type IBridge struct {
	IBridgeCaller     // Read-only binding to the contract
	IBridgeTransactor // Write-only binding to the contract
	IBridgeFilterer   // Log filterer for contract events
}

// IBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type IBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IBridgeSession struct {
	Contract     *IBridge          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IBridgeCallerSession struct {
	Contract *IBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// IBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IBridgeTransactorSession struct {
	Contract     *IBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// IBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type IBridgeRaw struct {
	Contract *IBridge // Generic contract binding to access the raw methods on
}

// IBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IBridgeCallerRaw struct {
	Contract *IBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// IBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IBridgeTransactorRaw struct {
	Contract *IBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIBridge creates a new instance of IBridge, bound to a specific deployed contract.
func NewIBridge(address common.Address, backend bind.ContractBackend) (*IBridge, error) {
	contract, err := bindIBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IBridge{IBridgeCaller: IBridgeCaller{contract: contract}, IBridgeTransactor: IBridgeTransactor{contract: contract}, IBridgeFilterer: IBridgeFilterer{contract: contract}}, nil
}

// NewIBridgeCaller creates a new read-only instance of IBridge, bound to a specific deployed contract.
func NewIBridgeCaller(address common.Address, caller bind.ContractCaller) (*IBridgeCaller, error) {
	contract, err := bindIBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IBridgeCaller{contract: contract}, nil
}

// NewIBridgeTransactor creates a new write-only instance of IBridge, bound to a specific deployed contract.
func NewIBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*IBridgeTransactor, error) {
	contract, err := bindIBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IBridgeTransactor{contract: contract}, nil
}

// NewIBridgeFilterer creates a new log filterer instance of IBridge, bound to a specific deployed contract.
func NewIBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*IBridgeFilterer, error) {
	contract, err := bindIBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IBridgeFilterer{contract: contract}, nil
}

// bindIBridge binds a generic wrapper to an already deployed contract.
func bindIBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IBridge *IBridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IBridge.Contract.IBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IBridge *IBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IBridge.Contract.IBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IBridge *IBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IBridge.Contract.IBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IBridge *IBridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IBridge *IBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IBridge *IBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IBridge.Contract.contract.Transact(opts, method, params...)
}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_IBridge *IBridgeCaller) ActiveOutbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IBridge.contract.Call(opts, &out, "activeOutbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_IBridge *IBridgeSession) ActiveOutbox() (common.Address, error) {
	return _IBridge.Contract.ActiveOutbox(&_IBridge.CallOpts)
}

// ActiveOutbox is a free data retrieval call binding the contract method 0xab5d8943.
//
// Solidity: function activeOutbox() view returns(address)
func (_IBridge *IBridgeCallerSession) ActiveOutbox() (common.Address, error) {
	return _IBridge.Contract.ActiveOutbox(&_IBridge.CallOpts)
}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_IBridge *IBridgeCaller) AllowedDelayedInboxes(opts *bind.CallOpts, inbox common.Address) (bool, error) {
	var out []interface{}
	err := _IBridge.contract.Call(opts, &out, "allowedDelayedInboxes", inbox)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_IBridge *IBridgeSession) AllowedDelayedInboxes(inbox common.Address) (bool, error) {
	return _IBridge.Contract.AllowedDelayedInboxes(&_IBridge.CallOpts, inbox)
}

// AllowedDelayedInboxes is a free data retrieval call binding the contract method 0xae60bd13.
//
// Solidity: function allowedDelayedInboxes(address inbox) view returns(bool)
func (_IBridge *IBridgeCallerSession) AllowedDelayedInboxes(inbox common.Address) (bool, error) {
	return _IBridge.Contract.AllowedDelayedInboxes(&_IBridge.CallOpts, inbox)
}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_IBridge *IBridgeCaller) AllowedOutboxes(opts *bind.CallOpts, outbox common.Address) (bool, error) {
	var out []interface{}
	err := _IBridge.contract.Call(opts, &out, "allowedOutboxes", outbox)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_IBridge *IBridgeSession) AllowedOutboxes(outbox common.Address) (bool, error) {
	return _IBridge.Contract.AllowedOutboxes(&_IBridge.CallOpts, outbox)
}

// AllowedOutboxes is a free data retrieval call binding the contract method 0x413b35bd.
//
// Solidity: function allowedOutboxes(address outbox) view returns(bool)
func (_IBridge *IBridgeCallerSession) AllowedOutboxes(outbox common.Address) (bool, error) {
	return _IBridge.Contract.AllowedOutboxes(&_IBridge.CallOpts, outbox)
}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_IBridge *IBridgeCaller) DelayedInboxAccs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _IBridge.contract.Call(opts, &out, "delayedInboxAccs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_IBridge *IBridgeSession) DelayedInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _IBridge.Contract.DelayedInboxAccs(&_IBridge.CallOpts, arg0)
}

// DelayedInboxAccs is a free data retrieval call binding the contract method 0xd5719dc2.
//
// Solidity: function delayedInboxAccs(uint256 ) view returns(bytes32)
func (_IBridge *IBridgeCallerSession) DelayedInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _IBridge.Contract.DelayedInboxAccs(&_IBridge.CallOpts, arg0)
}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_IBridge *IBridgeCaller) DelayedMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IBridge.contract.Call(opts, &out, "delayedMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_IBridge *IBridgeSession) DelayedMessageCount() (*big.Int, error) {
	return _IBridge.Contract.DelayedMessageCount(&_IBridge.CallOpts)
}

// DelayedMessageCount is a free data retrieval call binding the contract method 0xeca067ad.
//
// Solidity: function delayedMessageCount() view returns(uint256)
func (_IBridge *IBridgeCallerSession) DelayedMessageCount() (*big.Int, error) {
	return _IBridge.Contract.DelayedMessageCount(&_IBridge.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_IBridge *IBridgeCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IBridge.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_IBridge *IBridgeSession) Rollup() (common.Address, error) {
	return _IBridge.Contract.Rollup(&_IBridge.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_IBridge *IBridgeCallerSession) Rollup() (common.Address, error) {
	return _IBridge.Contract.Rollup(&_IBridge.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_IBridge *IBridgeCaller) SequencerInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IBridge.contract.Call(opts, &out, "sequencerInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_IBridge *IBridgeSession) SequencerInbox() (common.Address, error) {
	return _IBridge.Contract.SequencerInbox(&_IBridge.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_IBridge *IBridgeCallerSession) SequencerInbox() (common.Address, error) {
	return _IBridge.Contract.SequencerInbox(&_IBridge.CallOpts)
}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_IBridge *IBridgeCaller) SequencerInboxAccs(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _IBridge.contract.Call(opts, &out, "sequencerInboxAccs", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_IBridge *IBridgeSession) SequencerInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _IBridge.Contract.SequencerInboxAccs(&_IBridge.CallOpts, arg0)
}

// SequencerInboxAccs is a free data retrieval call binding the contract method 0x16bf5579.
//
// Solidity: function sequencerInboxAccs(uint256 ) view returns(bytes32)
func (_IBridge *IBridgeCallerSession) SequencerInboxAccs(arg0 *big.Int) ([32]byte, error) {
	return _IBridge.Contract.SequencerInboxAccs(&_IBridge.CallOpts, arg0)
}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_IBridge *IBridgeCaller) SequencerMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IBridge.contract.Call(opts, &out, "sequencerMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_IBridge *IBridgeSession) SequencerMessageCount() (*big.Int, error) {
	return _IBridge.Contract.SequencerMessageCount(&_IBridge.CallOpts)
}

// SequencerMessageCount is a free data retrieval call binding the contract method 0x0084120c.
//
// Solidity: function sequencerMessageCount() view returns(uint256)
func (_IBridge *IBridgeCallerSession) SequencerMessageCount() (*big.Int, error) {
	return _IBridge.Contract.SequencerMessageCount(&_IBridge.CallOpts)
}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_IBridge *IBridgeCaller) SequencerReportedSubMessageCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IBridge.contract.Call(opts, &out, "sequencerReportedSubMessageCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_IBridge *IBridgeSession) SequencerReportedSubMessageCount() (*big.Int, error) {
	return _IBridge.Contract.SequencerReportedSubMessageCount(&_IBridge.CallOpts)
}

// SequencerReportedSubMessageCount is a free data retrieval call binding the contract method 0x5fca4a16.
//
// Solidity: function sequencerReportedSubMessageCount() view returns(uint256)
func (_IBridge *IBridgeCallerSession) SequencerReportedSubMessageCount() (*big.Int, error) {
	return _IBridge.Contract.SequencerReportedSubMessageCount(&_IBridge.CallOpts)
}

// AllowedDelayedInboxList is a paid mutator transaction binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) returns(address)
func (_IBridge *IBridgeTransactor) AllowedDelayedInboxList(opts *bind.TransactOpts, arg0 *big.Int) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "allowedDelayedInboxList", arg0)
}

// AllowedDelayedInboxList is a paid mutator transaction binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) returns(address)
func (_IBridge *IBridgeSession) AllowedDelayedInboxList(arg0 *big.Int) (*types.Transaction, error) {
	return _IBridge.Contract.AllowedDelayedInboxList(&_IBridge.TransactOpts, arg0)
}

// AllowedDelayedInboxList is a paid mutator transaction binding the contract method 0xe76f5c8d.
//
// Solidity: function allowedDelayedInboxList(uint256 ) returns(address)
func (_IBridge *IBridgeTransactorSession) AllowedDelayedInboxList(arg0 *big.Int) (*types.Transaction, error) {
	return _IBridge.Contract.AllowedDelayedInboxList(&_IBridge.TransactOpts, arg0)
}

// AllowedOutboxList is a paid mutator transaction binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) returns(address)
func (_IBridge *IBridgeTransactor) AllowedOutboxList(opts *bind.TransactOpts, arg0 *big.Int) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "allowedOutboxList", arg0)
}

// AllowedOutboxList is a paid mutator transaction binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) returns(address)
func (_IBridge *IBridgeSession) AllowedOutboxList(arg0 *big.Int) (*types.Transaction, error) {
	return _IBridge.Contract.AllowedOutboxList(&_IBridge.TransactOpts, arg0)
}

// AllowedOutboxList is a paid mutator transaction binding the contract method 0x945e1147.
//
// Solidity: function allowedOutboxList(uint256 ) returns(address)
func (_IBridge *IBridgeTransactorSession) AllowedOutboxList(arg0 *big.Int) (*types.Transaction, error) {
	return _IBridge.Contract.AllowedOutboxList(&_IBridge.TransactOpts, arg0)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_IBridge *IBridgeTransactor) EnqueueDelayedMessage(opts *bind.TransactOpts, kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "enqueueDelayedMessage", kind, sender, messageDataHash)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_IBridge *IBridgeSession) EnqueueDelayedMessage(kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _IBridge.Contract.EnqueueDelayedMessage(&_IBridge.TransactOpts, kind, sender, messageDataHash)
}

// EnqueueDelayedMessage is a paid mutator transaction binding the contract method 0x8db5993b.
//
// Solidity: function enqueueDelayedMessage(uint8 kind, address sender, bytes32 messageDataHash) payable returns(uint256)
func (_IBridge *IBridgeTransactorSession) EnqueueDelayedMessage(kind uint8, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _IBridge.Contract.EnqueueDelayedMessage(&_IBridge.TransactOpts, kind, sender, messageDataHash)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_IBridge *IBridgeTransactor) EnqueueSequencerMessage(opts *bind.TransactOpts, dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "enqueueSequencerMessage", dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_IBridge *IBridgeSession) EnqueueSequencerMessage(dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _IBridge.Contract.EnqueueSequencerMessage(&_IBridge.TransactOpts, dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// EnqueueSequencerMessage is a paid mutator transaction binding the contract method 0x86598a56.
//
// Solidity: function enqueueSequencerMessage(bytes32 dataHash, uint256 afterDelayedMessagesRead, uint256 prevMessageCount, uint256 newMessageCount) returns(uint256 seqMessageIndex, bytes32 beforeAcc, bytes32 delayedAcc, bytes32 acc)
func (_IBridge *IBridgeTransactorSession) EnqueueSequencerMessage(dataHash [32]byte, afterDelayedMessagesRead *big.Int, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _IBridge.Contract.EnqueueSequencerMessage(&_IBridge.TransactOpts, dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_IBridge *IBridgeTransactor) ExecuteCall(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "executeCall", to, value, data)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_IBridge *IBridgeSession) ExecuteCall(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IBridge.Contract.ExecuteCall(&_IBridge.TransactOpts, to, value, data)
}

// ExecuteCall is a paid mutator transaction binding the contract method 0x9e5d4c49.
//
// Solidity: function executeCall(address to, uint256 value, bytes data) returns(bool success, bytes returnData)
func (_IBridge *IBridgeTransactorSession) ExecuteCall(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IBridge.Contract.ExecuteCall(&_IBridge.TransactOpts, to, value, data)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_IBridge *IBridgeTransactor) Initialize(opts *bind.TransactOpts, rollup_ common.Address) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "initialize", rollup_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_IBridge *IBridgeSession) Initialize(rollup_ common.Address) (*types.Transaction, error) {
	return _IBridge.Contract.Initialize(&_IBridge.TransactOpts, rollup_)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address rollup_) returns()
func (_IBridge *IBridgeTransactorSession) Initialize(rollup_ common.Address) (*types.Transaction, error) {
	return _IBridge.Contract.Initialize(&_IBridge.TransactOpts, rollup_)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_IBridge *IBridgeTransactor) SetDelayedInbox(opts *bind.TransactOpts, inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "setDelayedInbox", inbox, enabled)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_IBridge *IBridgeSession) SetDelayedInbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _IBridge.Contract.SetDelayedInbox(&_IBridge.TransactOpts, inbox, enabled)
}

// SetDelayedInbox is a paid mutator transaction binding the contract method 0x47fb24c5.
//
// Solidity: function setDelayedInbox(address inbox, bool enabled) returns()
func (_IBridge *IBridgeTransactorSession) SetDelayedInbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _IBridge.Contract.SetDelayedInbox(&_IBridge.TransactOpts, inbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address inbox, bool enabled) returns()
func (_IBridge *IBridgeTransactor) SetOutbox(opts *bind.TransactOpts, inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "setOutbox", inbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address inbox, bool enabled) returns()
func (_IBridge *IBridgeSession) SetOutbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _IBridge.Contract.SetOutbox(&_IBridge.TransactOpts, inbox, enabled)
}

// SetOutbox is a paid mutator transaction binding the contract method 0xcee3d728.
//
// Solidity: function setOutbox(address inbox, bool enabled) returns()
func (_IBridge *IBridgeTransactorSession) SetOutbox(inbox common.Address, enabled bool) (*types.Transaction, error) {
	return _IBridge.Contract.SetOutbox(&_IBridge.TransactOpts, inbox, enabled)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_IBridge *IBridgeTransactor) SetSequencerInbox(opts *bind.TransactOpts, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "setSequencerInbox", _sequencerInbox)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_IBridge *IBridgeSession) SetSequencerInbox(_sequencerInbox common.Address) (*types.Transaction, error) {
	return _IBridge.Contract.SetSequencerInbox(&_IBridge.TransactOpts, _sequencerInbox)
}

// SetSequencerInbox is a paid mutator transaction binding the contract method 0x4f61f850.
//
// Solidity: function setSequencerInbox(address _sequencerInbox) returns()
func (_IBridge *IBridgeTransactorSession) SetSequencerInbox(_sequencerInbox common.Address) (*types.Transaction, error) {
	return _IBridge.Contract.SetSequencerInbox(&_IBridge.TransactOpts, _sequencerInbox)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address batchPoster, bytes32 dataHash) returns(uint256 msgNum)
func (_IBridge *IBridgeTransactor) SubmitBatchSpendingReport(opts *bind.TransactOpts, batchPoster common.Address, dataHash [32]byte) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "submitBatchSpendingReport", batchPoster, dataHash)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address batchPoster, bytes32 dataHash) returns(uint256 msgNum)
func (_IBridge *IBridgeSession) SubmitBatchSpendingReport(batchPoster common.Address, dataHash [32]byte) (*types.Transaction, error) {
	return _IBridge.Contract.SubmitBatchSpendingReport(&_IBridge.TransactOpts, batchPoster, dataHash)
}

// SubmitBatchSpendingReport is a paid mutator transaction binding the contract method 0x7a88b107.
//
// Solidity: function submitBatchSpendingReport(address batchPoster, bytes32 dataHash) returns(uint256 msgNum)
func (_IBridge *IBridgeTransactorSession) SubmitBatchSpendingReport(batchPoster common.Address, dataHash [32]byte) (*types.Transaction, error) {
	return _IBridge.Contract.SubmitBatchSpendingReport(&_IBridge.TransactOpts, batchPoster, dataHash)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_IBridge *IBridgeTransactor) UpdateRollupAddress(opts *bind.TransactOpts, _rollup common.Address) (*types.Transaction, error) {
	return _IBridge.contract.Transact(opts, "updateRollupAddress", _rollup)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_IBridge *IBridgeSession) UpdateRollupAddress(_rollup common.Address) (*types.Transaction, error) {
	return _IBridge.Contract.UpdateRollupAddress(&_IBridge.TransactOpts, _rollup)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x919cc706.
//
// Solidity: function updateRollupAddress(address _rollup) returns()
func (_IBridge *IBridgeTransactorSession) UpdateRollupAddress(_rollup common.Address) (*types.Transaction, error) {
	return _IBridge.Contract.UpdateRollupAddress(&_IBridge.TransactOpts, _rollup)
}

// IBridgeBridgeCallTriggeredIterator is returned from FilterBridgeCallTriggered and is used to iterate over the raw logs and unpacked data for BridgeCallTriggered events raised by the IBridge contract.
type IBridgeBridgeCallTriggeredIterator struct {
	Event *IBridgeBridgeCallTriggered // Event containing the contract specifics and raw log

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
func (it *IBridgeBridgeCallTriggeredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IBridgeBridgeCallTriggered)
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
		it.Event = new(IBridgeBridgeCallTriggered)
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
func (it *IBridgeBridgeCallTriggeredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IBridgeBridgeCallTriggeredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IBridgeBridgeCallTriggered represents a BridgeCallTriggered event raised by the IBridge contract.
type IBridgeBridgeCallTriggered struct {
	Outbox common.Address
	To     common.Address
	Value  *big.Int
	Data   []byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBridgeCallTriggered is a free log retrieval operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_IBridge *IBridgeFilterer) FilterBridgeCallTriggered(opts *bind.FilterOpts, outbox []common.Address, to []common.Address) (*IBridgeBridgeCallTriggeredIterator, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _IBridge.contract.FilterLogs(opts, "BridgeCallTriggered", outboxRule, toRule)
	if err != nil {
		return nil, err
	}
	return &IBridgeBridgeCallTriggeredIterator{contract: _IBridge.contract, event: "BridgeCallTriggered", logs: logs, sub: sub}, nil
}

// WatchBridgeCallTriggered is a free log subscription operation binding the contract event 0x2d9d115ef3e4a606d698913b1eae831a3cdfe20d9a83d48007b0526749c3d466.
//
// Solidity: event BridgeCallTriggered(address indexed outbox, address indexed to, uint256 value, bytes data)
func (_IBridge *IBridgeFilterer) WatchBridgeCallTriggered(opts *bind.WatchOpts, sink chan<- *IBridgeBridgeCallTriggered, outbox []common.Address, to []common.Address) (event.Subscription, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _IBridge.contract.WatchLogs(opts, "BridgeCallTriggered", outboxRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IBridgeBridgeCallTriggered)
				if err := _IBridge.contract.UnpackLog(event, "BridgeCallTriggered", log); err != nil {
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
func (_IBridge *IBridgeFilterer) ParseBridgeCallTriggered(log types.Log) (*IBridgeBridgeCallTriggered, error) {
	event := new(IBridgeBridgeCallTriggered)
	if err := _IBridge.contract.UnpackLog(event, "BridgeCallTriggered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IBridgeInboxToggleIterator is returned from FilterInboxToggle and is used to iterate over the raw logs and unpacked data for InboxToggle events raised by the IBridge contract.
type IBridgeInboxToggleIterator struct {
	Event *IBridgeInboxToggle // Event containing the contract specifics and raw log

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
func (it *IBridgeInboxToggleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IBridgeInboxToggle)
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
		it.Event = new(IBridgeInboxToggle)
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
func (it *IBridgeInboxToggleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IBridgeInboxToggleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IBridgeInboxToggle represents a InboxToggle event raised by the IBridge contract.
type IBridgeInboxToggle struct {
	Inbox   common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInboxToggle is a free log retrieval operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_IBridge *IBridgeFilterer) FilterInboxToggle(opts *bind.FilterOpts, inbox []common.Address) (*IBridgeInboxToggleIterator, error) {

	var inboxRule []interface{}
	for _, inboxItem := range inbox {
		inboxRule = append(inboxRule, inboxItem)
	}

	logs, sub, err := _IBridge.contract.FilterLogs(opts, "InboxToggle", inboxRule)
	if err != nil {
		return nil, err
	}
	return &IBridgeInboxToggleIterator{contract: _IBridge.contract, event: "InboxToggle", logs: logs, sub: sub}, nil
}

// WatchInboxToggle is a free log subscription operation binding the contract event 0x6675ce8882cb71637de5903a193d218cc0544be9c0650cb83e0955f6aa2bf521.
//
// Solidity: event InboxToggle(address indexed inbox, bool enabled)
func (_IBridge *IBridgeFilterer) WatchInboxToggle(opts *bind.WatchOpts, sink chan<- *IBridgeInboxToggle, inbox []common.Address) (event.Subscription, error) {

	var inboxRule []interface{}
	for _, inboxItem := range inbox {
		inboxRule = append(inboxRule, inboxItem)
	}

	logs, sub, err := _IBridge.contract.WatchLogs(opts, "InboxToggle", inboxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IBridgeInboxToggle)
				if err := _IBridge.contract.UnpackLog(event, "InboxToggle", log); err != nil {
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
func (_IBridge *IBridgeFilterer) ParseInboxToggle(log types.Log) (*IBridgeInboxToggle, error) {
	event := new(IBridgeInboxToggle)
	if err := _IBridge.contract.UnpackLog(event, "InboxToggle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IBridgeMessageDeliveredIterator is returned from FilterMessageDelivered and is used to iterate over the raw logs and unpacked data for MessageDelivered events raised by the IBridge contract.
type IBridgeMessageDeliveredIterator struct {
	Event *IBridgeMessageDelivered // Event containing the contract specifics and raw log

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
func (it *IBridgeMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IBridgeMessageDelivered)
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
		it.Event = new(IBridgeMessageDelivered)
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
func (it *IBridgeMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IBridgeMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IBridgeMessageDelivered represents a MessageDelivered event raised by the IBridge contract.
type IBridgeMessageDelivered struct {
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
func (_IBridge *IBridgeFilterer) FilterMessageDelivered(opts *bind.FilterOpts, messageIndex []*big.Int, beforeInboxAcc [][32]byte) (*IBridgeMessageDeliveredIterator, error) {

	var messageIndexRule []interface{}
	for _, messageIndexItem := range messageIndex {
		messageIndexRule = append(messageIndexRule, messageIndexItem)
	}
	var beforeInboxAccRule []interface{}
	for _, beforeInboxAccItem := range beforeInboxAcc {
		beforeInboxAccRule = append(beforeInboxAccRule, beforeInboxAccItem)
	}

	logs, sub, err := _IBridge.contract.FilterLogs(opts, "MessageDelivered", messageIndexRule, beforeInboxAccRule)
	if err != nil {
		return nil, err
	}
	return &IBridgeMessageDeliveredIterator{contract: _IBridge.contract, event: "MessageDelivered", logs: logs, sub: sub}, nil
}

// WatchMessageDelivered is a free log subscription operation binding the contract event 0x5e3c1311ea442664e8b1611bfabef659120ea7a0a2cfc0667700bebc69cbffe1.
//
// Solidity: event MessageDelivered(uint256 indexed messageIndex, bytes32 indexed beforeInboxAcc, address inbox, uint8 kind, address sender, bytes32 messageDataHash, uint256 baseFeeL1, uint64 timestamp)
func (_IBridge *IBridgeFilterer) WatchMessageDelivered(opts *bind.WatchOpts, sink chan<- *IBridgeMessageDelivered, messageIndex []*big.Int, beforeInboxAcc [][32]byte) (event.Subscription, error) {

	var messageIndexRule []interface{}
	for _, messageIndexItem := range messageIndex {
		messageIndexRule = append(messageIndexRule, messageIndexItem)
	}
	var beforeInboxAccRule []interface{}
	for _, beforeInboxAccItem := range beforeInboxAcc {
		beforeInboxAccRule = append(beforeInboxAccRule, beforeInboxAccItem)
	}

	logs, sub, err := _IBridge.contract.WatchLogs(opts, "MessageDelivered", messageIndexRule, beforeInboxAccRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IBridgeMessageDelivered)
				if err := _IBridge.contract.UnpackLog(event, "MessageDelivered", log); err != nil {
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
func (_IBridge *IBridgeFilterer) ParseMessageDelivered(log types.Log) (*IBridgeMessageDelivered, error) {
	event := new(IBridgeMessageDelivered)
	if err := _IBridge.contract.UnpackLog(event, "MessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IBridgeOutboxToggleIterator is returned from FilterOutboxToggle and is used to iterate over the raw logs and unpacked data for OutboxToggle events raised by the IBridge contract.
type IBridgeOutboxToggleIterator struct {
	Event *IBridgeOutboxToggle // Event containing the contract specifics and raw log

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
func (it *IBridgeOutboxToggleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IBridgeOutboxToggle)
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
		it.Event = new(IBridgeOutboxToggle)
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
func (it *IBridgeOutboxToggleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IBridgeOutboxToggleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IBridgeOutboxToggle represents a OutboxToggle event raised by the IBridge contract.
type IBridgeOutboxToggle struct {
	Outbox  common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOutboxToggle is a free log retrieval operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_IBridge *IBridgeFilterer) FilterOutboxToggle(opts *bind.FilterOpts, outbox []common.Address) (*IBridgeOutboxToggleIterator, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}

	logs, sub, err := _IBridge.contract.FilterLogs(opts, "OutboxToggle", outboxRule)
	if err != nil {
		return nil, err
	}
	return &IBridgeOutboxToggleIterator{contract: _IBridge.contract, event: "OutboxToggle", logs: logs, sub: sub}, nil
}

// WatchOutboxToggle is a free log subscription operation binding the contract event 0x49477e7356dbcb654ab85d7534b50126772d938130d1350e23e2540370c8dffa.
//
// Solidity: event OutboxToggle(address indexed outbox, bool enabled)
func (_IBridge *IBridgeFilterer) WatchOutboxToggle(opts *bind.WatchOpts, sink chan<- *IBridgeOutboxToggle, outbox []common.Address) (event.Subscription, error) {

	var outboxRule []interface{}
	for _, outboxItem := range outbox {
		outboxRule = append(outboxRule, outboxItem)
	}

	logs, sub, err := _IBridge.contract.WatchLogs(opts, "OutboxToggle", outboxRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IBridgeOutboxToggle)
				if err := _IBridge.contract.UnpackLog(event, "OutboxToggle", log); err != nil {
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
func (_IBridge *IBridgeFilterer) ParseOutboxToggle(log types.Log) (*IBridgeOutboxToggle, error) {
	event := new(IBridgeOutboxToggle)
	if err := _IBridge.contract.UnpackLog(event, "OutboxToggle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IBridgeSequencerInboxUpdatedIterator is returned from FilterSequencerInboxUpdated and is used to iterate over the raw logs and unpacked data for SequencerInboxUpdated events raised by the IBridge contract.
type IBridgeSequencerInboxUpdatedIterator struct {
	Event *IBridgeSequencerInboxUpdated // Event containing the contract specifics and raw log

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
func (it *IBridgeSequencerInboxUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IBridgeSequencerInboxUpdated)
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
		it.Event = new(IBridgeSequencerInboxUpdated)
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
func (it *IBridgeSequencerInboxUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IBridgeSequencerInboxUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IBridgeSequencerInboxUpdated represents a SequencerInboxUpdated event raised by the IBridge contract.
type IBridgeSequencerInboxUpdated struct {
	NewSequencerInbox common.Address
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterSequencerInboxUpdated is a free log retrieval operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_IBridge *IBridgeFilterer) FilterSequencerInboxUpdated(opts *bind.FilterOpts) (*IBridgeSequencerInboxUpdatedIterator, error) {

	logs, sub, err := _IBridge.contract.FilterLogs(opts, "SequencerInboxUpdated")
	if err != nil {
		return nil, err
	}
	return &IBridgeSequencerInboxUpdatedIterator{contract: _IBridge.contract, event: "SequencerInboxUpdated", logs: logs, sub: sub}, nil
}

// WatchSequencerInboxUpdated is a free log subscription operation binding the contract event 0x8c1e6003ed33ca6748d4ad3dd4ecc949065c89dceb31fdf546a5289202763c6a.
//
// Solidity: event SequencerInboxUpdated(address newSequencerInbox)
func (_IBridge *IBridgeFilterer) WatchSequencerInboxUpdated(opts *bind.WatchOpts, sink chan<- *IBridgeSequencerInboxUpdated) (event.Subscription, error) {

	logs, sub, err := _IBridge.contract.WatchLogs(opts, "SequencerInboxUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IBridgeSequencerInboxUpdated)
				if err := _IBridge.contract.UnpackLog(event, "SequencerInboxUpdated", log); err != nil {
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
func (_IBridge *IBridgeFilterer) ParseSequencerInboxUpdated(log types.Log) (*IBridgeSequencerInboxUpdated, error) {
	event := new(IBridgeSequencerInboxUpdated)
	if err := _IBridge.contract.UnpackLog(event, "SequencerInboxUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IDelayedMessageProviderMetaData contains all meta data concerning the IDelayedMessageProvider contract.
var IDelayedMessageProviderMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"}]",
}

// IDelayedMessageProviderABI is the input ABI used to generate the binding from.
// Deprecated: Use IDelayedMessageProviderMetaData.ABI instead.
var IDelayedMessageProviderABI = IDelayedMessageProviderMetaData.ABI

// IDelayedMessageProvider is an auto generated Go binding around an Ethereum contract.
type IDelayedMessageProvider struct {
	IDelayedMessageProviderCaller     // Read-only binding to the contract
	IDelayedMessageProviderTransactor // Write-only binding to the contract
	IDelayedMessageProviderFilterer   // Log filterer for contract events
}

// IDelayedMessageProviderCaller is an auto generated read-only Go binding around an Ethereum contract.
type IDelayedMessageProviderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IDelayedMessageProviderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IDelayedMessageProviderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IDelayedMessageProviderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IDelayedMessageProviderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IDelayedMessageProviderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IDelayedMessageProviderSession struct {
	Contract     *IDelayedMessageProvider // Generic contract binding to set the session for
	CallOpts     bind.CallOpts            // Call options to use throughout this session
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// IDelayedMessageProviderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IDelayedMessageProviderCallerSession struct {
	Contract *IDelayedMessageProviderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                  // Call options to use throughout this session
}

// IDelayedMessageProviderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IDelayedMessageProviderTransactorSession struct {
	Contract     *IDelayedMessageProviderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                  // Transaction auth options to use throughout this session
}

// IDelayedMessageProviderRaw is an auto generated low-level Go binding around an Ethereum contract.
type IDelayedMessageProviderRaw struct {
	Contract *IDelayedMessageProvider // Generic contract binding to access the raw methods on
}

// IDelayedMessageProviderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IDelayedMessageProviderCallerRaw struct {
	Contract *IDelayedMessageProviderCaller // Generic read-only contract binding to access the raw methods on
}

// IDelayedMessageProviderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IDelayedMessageProviderTransactorRaw struct {
	Contract *IDelayedMessageProviderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIDelayedMessageProvider creates a new instance of IDelayedMessageProvider, bound to a specific deployed contract.
func NewIDelayedMessageProvider(address common.Address, backend bind.ContractBackend) (*IDelayedMessageProvider, error) {
	contract, err := bindIDelayedMessageProvider(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IDelayedMessageProvider{IDelayedMessageProviderCaller: IDelayedMessageProviderCaller{contract: contract}, IDelayedMessageProviderTransactor: IDelayedMessageProviderTransactor{contract: contract}, IDelayedMessageProviderFilterer: IDelayedMessageProviderFilterer{contract: contract}}, nil
}

// NewIDelayedMessageProviderCaller creates a new read-only instance of IDelayedMessageProvider, bound to a specific deployed contract.
func NewIDelayedMessageProviderCaller(address common.Address, caller bind.ContractCaller) (*IDelayedMessageProviderCaller, error) {
	contract, err := bindIDelayedMessageProvider(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IDelayedMessageProviderCaller{contract: contract}, nil
}

// NewIDelayedMessageProviderTransactor creates a new write-only instance of IDelayedMessageProvider, bound to a specific deployed contract.
func NewIDelayedMessageProviderTransactor(address common.Address, transactor bind.ContractTransactor) (*IDelayedMessageProviderTransactor, error) {
	contract, err := bindIDelayedMessageProvider(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IDelayedMessageProviderTransactor{contract: contract}, nil
}

// NewIDelayedMessageProviderFilterer creates a new log filterer instance of IDelayedMessageProvider, bound to a specific deployed contract.
func NewIDelayedMessageProviderFilterer(address common.Address, filterer bind.ContractFilterer) (*IDelayedMessageProviderFilterer, error) {
	contract, err := bindIDelayedMessageProvider(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IDelayedMessageProviderFilterer{contract: contract}, nil
}

// bindIDelayedMessageProvider binds a generic wrapper to an already deployed contract.
func bindIDelayedMessageProvider(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IDelayedMessageProviderMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IDelayedMessageProvider *IDelayedMessageProviderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IDelayedMessageProvider.Contract.IDelayedMessageProviderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IDelayedMessageProvider *IDelayedMessageProviderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IDelayedMessageProvider.Contract.IDelayedMessageProviderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IDelayedMessageProvider *IDelayedMessageProviderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IDelayedMessageProvider.Contract.IDelayedMessageProviderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IDelayedMessageProvider *IDelayedMessageProviderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IDelayedMessageProvider.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IDelayedMessageProvider *IDelayedMessageProviderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IDelayedMessageProvider.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IDelayedMessageProvider *IDelayedMessageProviderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IDelayedMessageProvider.Contract.contract.Transact(opts, method, params...)
}

// IDelayedMessageProviderInboxMessageDeliveredIterator is returned from FilterInboxMessageDelivered and is used to iterate over the raw logs and unpacked data for InboxMessageDelivered events raised by the IDelayedMessageProvider contract.
type IDelayedMessageProviderInboxMessageDeliveredIterator struct {
	Event *IDelayedMessageProviderInboxMessageDelivered // Event containing the contract specifics and raw log

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
func (it *IDelayedMessageProviderInboxMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IDelayedMessageProviderInboxMessageDelivered)
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
		it.Event = new(IDelayedMessageProviderInboxMessageDelivered)
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
func (it *IDelayedMessageProviderInboxMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IDelayedMessageProviderInboxMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IDelayedMessageProviderInboxMessageDelivered represents a InboxMessageDelivered event raised by the IDelayedMessageProvider contract.
type IDelayedMessageProviderInboxMessageDelivered struct {
	MessageNum *big.Int
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDelivered is a free log retrieval operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_IDelayedMessageProvider *IDelayedMessageProviderFilterer) FilterInboxMessageDelivered(opts *bind.FilterOpts, messageNum []*big.Int) (*IDelayedMessageProviderInboxMessageDeliveredIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _IDelayedMessageProvider.contract.FilterLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &IDelayedMessageProviderInboxMessageDeliveredIterator{contract: _IDelayedMessageProvider.contract, event: "InboxMessageDelivered", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDelivered is a free log subscription operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_IDelayedMessageProvider *IDelayedMessageProviderFilterer) WatchInboxMessageDelivered(opts *bind.WatchOpts, sink chan<- *IDelayedMessageProviderInboxMessageDelivered, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _IDelayedMessageProvider.contract.WatchLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IDelayedMessageProviderInboxMessageDelivered)
				if err := _IDelayedMessageProvider.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
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
func (_IDelayedMessageProvider *IDelayedMessageProviderFilterer) ParseInboxMessageDelivered(log types.Log) (*IDelayedMessageProviderInboxMessageDelivered, error) {
	event := new(IDelayedMessageProviderInboxMessageDelivered)
	if err := _IDelayedMessageProvider.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IDelayedMessageProviderInboxMessageDeliveredFromOriginIterator is returned from FilterInboxMessageDeliveredFromOrigin and is used to iterate over the raw logs and unpacked data for InboxMessageDeliveredFromOrigin events raised by the IDelayedMessageProvider contract.
type IDelayedMessageProviderInboxMessageDeliveredFromOriginIterator struct {
	Event *IDelayedMessageProviderInboxMessageDeliveredFromOrigin // Event containing the contract specifics and raw log

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
func (it *IDelayedMessageProviderInboxMessageDeliveredFromOriginIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IDelayedMessageProviderInboxMessageDeliveredFromOrigin)
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
		it.Event = new(IDelayedMessageProviderInboxMessageDeliveredFromOrigin)
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
func (it *IDelayedMessageProviderInboxMessageDeliveredFromOriginIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IDelayedMessageProviderInboxMessageDeliveredFromOriginIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IDelayedMessageProviderInboxMessageDeliveredFromOrigin represents a InboxMessageDeliveredFromOrigin event raised by the IDelayedMessageProvider contract.
type IDelayedMessageProviderInboxMessageDeliveredFromOrigin struct {
	MessageNum *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDeliveredFromOrigin is a free log retrieval operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_IDelayedMessageProvider *IDelayedMessageProviderFilterer) FilterInboxMessageDeliveredFromOrigin(opts *bind.FilterOpts, messageNum []*big.Int) (*IDelayedMessageProviderInboxMessageDeliveredFromOriginIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _IDelayedMessageProvider.contract.FilterLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &IDelayedMessageProviderInboxMessageDeliveredFromOriginIterator{contract: _IDelayedMessageProvider.contract, event: "InboxMessageDeliveredFromOrigin", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDeliveredFromOrigin is a free log subscription operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_IDelayedMessageProvider *IDelayedMessageProviderFilterer) WatchInboxMessageDeliveredFromOrigin(opts *bind.WatchOpts, sink chan<- *IDelayedMessageProviderInboxMessageDeliveredFromOrigin, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _IDelayedMessageProvider.contract.WatchLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IDelayedMessageProviderInboxMessageDeliveredFromOrigin)
				if err := _IDelayedMessageProvider.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
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
func (_IDelayedMessageProvider *IDelayedMessageProviderFilterer) ParseInboxMessageDeliveredFromOrigin(log types.Log) (*IDelayedMessageProviderInboxMessageDeliveredFromOrigin, error) {
	event := new(IDelayedMessageProviderInboxMessageDeliveredFromOrigin)
	if err := _IDelayedMessageProvider.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IInboxMetaData contains all meta data concerning the IInbox contract.
var IInboxMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"dataLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"}],\"name\":\"calculateRetryableSubmissionFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2CallValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxSubmissionCost\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"excessFeeRefundAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"callValueRefundAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"createRetryableTicket\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"depositEth\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"_bridge\",\"type\":\"address\"},{\"internalType\":\"contractISequencerInbox\",\"name\":\"_sequencerInbox\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"_bridge\",\"type\":\"address\"}],\"name\":\"postUpgradeInit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendContractTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedContractTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedUnsignedTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedUnsignedTransactionToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"messageData\",\"type\":\"bytes\"}],\"name\":\"sendL2Message\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"messageData\",\"type\":\"bytes\"}],\"name\":\"sendL2MessageFromOrigin\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendUnsignedTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendUnsignedTransactionToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"withdrawTo\",\"type\":\"address\"}],\"name\":\"sendWithdrawEthToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"contractISequencerInbox\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2CallValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxSubmissionCost\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"excessFeeRefundAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"callValueRefundAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"unsafeCreateRetryableTicket\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
}

// IInboxABI is the input ABI used to generate the binding from.
// Deprecated: Use IInboxMetaData.ABI instead.
var IInboxABI = IInboxMetaData.ABI

// IInbox is an auto generated Go binding around an Ethereum contract.
type IInbox struct {
	IInboxCaller     // Read-only binding to the contract
	IInboxTransactor // Write-only binding to the contract
	IInboxFilterer   // Log filterer for contract events
}

// IInboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type IInboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IInboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IInboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IInboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IInboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IInboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IInboxSession struct {
	Contract     *IInbox           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IInboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IInboxCallerSession struct {
	Contract *IInboxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// IInboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IInboxTransactorSession struct {
	Contract     *IInboxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IInboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type IInboxRaw struct {
	Contract *IInbox // Generic contract binding to access the raw methods on
}

// IInboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IInboxCallerRaw struct {
	Contract *IInboxCaller // Generic read-only contract binding to access the raw methods on
}

// IInboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IInboxTransactorRaw struct {
	Contract *IInboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIInbox creates a new instance of IInbox, bound to a specific deployed contract.
func NewIInbox(address common.Address, backend bind.ContractBackend) (*IInbox, error) {
	contract, err := bindIInbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IInbox{IInboxCaller: IInboxCaller{contract: contract}, IInboxTransactor: IInboxTransactor{contract: contract}, IInboxFilterer: IInboxFilterer{contract: contract}}, nil
}

// NewIInboxCaller creates a new read-only instance of IInbox, bound to a specific deployed contract.
func NewIInboxCaller(address common.Address, caller bind.ContractCaller) (*IInboxCaller, error) {
	contract, err := bindIInbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IInboxCaller{contract: contract}, nil
}

// NewIInboxTransactor creates a new write-only instance of IInbox, bound to a specific deployed contract.
func NewIInboxTransactor(address common.Address, transactor bind.ContractTransactor) (*IInboxTransactor, error) {
	contract, err := bindIInbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IInboxTransactor{contract: contract}, nil
}

// NewIInboxFilterer creates a new log filterer instance of IInbox, bound to a specific deployed contract.
func NewIInboxFilterer(address common.Address, filterer bind.ContractFilterer) (*IInboxFilterer, error) {
	contract, err := bindIInbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IInboxFilterer{contract: contract}, nil
}

// bindIInbox binds a generic wrapper to an already deployed contract.
func bindIInbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IInboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IInbox *IInboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IInbox.Contract.IInboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IInbox *IInboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IInbox.Contract.IInboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IInbox *IInboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IInbox.Contract.IInboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IInbox *IInboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IInbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IInbox *IInboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IInbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IInbox *IInboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IInbox.Contract.contract.Transact(opts, method, params...)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_IInbox *IInboxCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IInbox.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_IInbox *IInboxSession) Bridge() (common.Address, error) {
	return _IInbox.Contract.Bridge(&_IInbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_IInbox *IInboxCallerSession) Bridge() (common.Address, error) {
	return _IInbox.Contract.Bridge(&_IInbox.CallOpts)
}

// CalculateRetryableSubmissionFee is a free data retrieval call binding the contract method 0xa66b327d.
//
// Solidity: function calculateRetryableSubmissionFee(uint256 dataLength, uint256 baseFee) view returns(uint256)
func (_IInbox *IInboxCaller) CalculateRetryableSubmissionFee(opts *bind.CallOpts, dataLength *big.Int, baseFee *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _IInbox.contract.Call(opts, &out, "calculateRetryableSubmissionFee", dataLength, baseFee)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CalculateRetryableSubmissionFee is a free data retrieval call binding the contract method 0xa66b327d.
//
// Solidity: function calculateRetryableSubmissionFee(uint256 dataLength, uint256 baseFee) view returns(uint256)
func (_IInbox *IInboxSession) CalculateRetryableSubmissionFee(dataLength *big.Int, baseFee *big.Int) (*big.Int, error) {
	return _IInbox.Contract.CalculateRetryableSubmissionFee(&_IInbox.CallOpts, dataLength, baseFee)
}

// CalculateRetryableSubmissionFee is a free data retrieval call binding the contract method 0xa66b327d.
//
// Solidity: function calculateRetryableSubmissionFee(uint256 dataLength, uint256 baseFee) view returns(uint256)
func (_IInbox *IInboxCallerSession) CalculateRetryableSubmissionFee(dataLength *big.Int, baseFee *big.Int) (*big.Int, error) {
	return _IInbox.Contract.CalculateRetryableSubmissionFee(&_IInbox.CallOpts, dataLength, baseFee)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_IInbox *IInboxCaller) SequencerInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IInbox.contract.Call(opts, &out, "sequencerInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_IInbox *IInboxSession) SequencerInbox() (common.Address, error) {
	return _IInbox.Contract.SequencerInbox(&_IInbox.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_IInbox *IInboxCallerSession) SequencerInbox() (common.Address, error) {
	return _IInbox.Contract.SequencerInbox(&_IInbox.CallOpts)
}

// CreateRetryableTicket is a paid mutator transaction binding the contract method 0x679b6ded.
//
// Solidity: function createRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_IInbox *IInboxTransactor) CreateRetryableTicket(opts *bind.TransactOpts, to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "createRetryableTicket", to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// CreateRetryableTicket is a paid mutator transaction binding the contract method 0x679b6ded.
//
// Solidity: function createRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_IInbox *IInboxSession) CreateRetryableTicket(to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.CreateRetryableTicket(&_IInbox.TransactOpts, to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// CreateRetryableTicket is a paid mutator transaction binding the contract method 0x679b6ded.
//
// Solidity: function createRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_IInbox *IInboxTransactorSession) CreateRetryableTicket(to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.CreateRetryableTicket(&_IInbox.TransactOpts, to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// DepositEth is a paid mutator transaction binding the contract method 0x439370b1.
//
// Solidity: function depositEth() payable returns(uint256)
func (_IInbox *IInboxTransactor) DepositEth(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "depositEth")
}

// DepositEth is a paid mutator transaction binding the contract method 0x439370b1.
//
// Solidity: function depositEth() payable returns(uint256)
func (_IInbox *IInboxSession) DepositEth() (*types.Transaction, error) {
	return _IInbox.Contract.DepositEth(&_IInbox.TransactOpts)
}

// DepositEth is a paid mutator transaction binding the contract method 0x439370b1.
//
// Solidity: function depositEth() payable returns(uint256)
func (_IInbox *IInboxTransactorSession) DepositEth() (*types.Transaction, error) {
	return _IInbox.Contract.DepositEth(&_IInbox.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _bridge, address _sequencerInbox) returns()
func (_IInbox *IInboxTransactor) Initialize(opts *bind.TransactOpts, _bridge common.Address, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "initialize", _bridge, _sequencerInbox)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _bridge, address _sequencerInbox) returns()
func (_IInbox *IInboxSession) Initialize(_bridge common.Address, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _IInbox.Contract.Initialize(&_IInbox.TransactOpts, _bridge, _sequencerInbox)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _bridge, address _sequencerInbox) returns()
func (_IInbox *IInboxTransactorSession) Initialize(_bridge common.Address, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _IInbox.Contract.Initialize(&_IInbox.TransactOpts, _bridge, _sequencerInbox)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_IInbox *IInboxTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_IInbox *IInboxSession) Pause() (*types.Transaction, error) {
	return _IInbox.Contract.Pause(&_IInbox.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_IInbox *IInboxTransactorSession) Pause() (*types.Transaction, error) {
	return _IInbox.Contract.Pause(&_IInbox.TransactOpts)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xc474d2c5.
//
// Solidity: function postUpgradeInit(address _bridge) returns()
func (_IInbox *IInboxTransactor) PostUpgradeInit(opts *bind.TransactOpts, _bridge common.Address) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "postUpgradeInit", _bridge)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xc474d2c5.
//
// Solidity: function postUpgradeInit(address _bridge) returns()
func (_IInbox *IInboxSession) PostUpgradeInit(_bridge common.Address) (*types.Transaction, error) {
	return _IInbox.Contract.PostUpgradeInit(&_IInbox.TransactOpts, _bridge)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xc474d2c5.
//
// Solidity: function postUpgradeInit(address _bridge) returns()
func (_IInbox *IInboxTransactorSession) PostUpgradeInit(_bridge common.Address) (*types.Transaction, error) {
	return _IInbox.Contract.PostUpgradeInit(&_IInbox.TransactOpts, _bridge)
}

// SendContractTransaction is a paid mutator transaction binding the contract method 0x8a631aa6.
//
// Solidity: function sendContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, uint256 value, bytes data) returns(uint256)
func (_IInbox *IInboxTransactor) SendContractTransaction(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "sendContractTransaction", gasLimit, maxFeePerGas, to, value, data)
}

// SendContractTransaction is a paid mutator transaction binding the contract method 0x8a631aa6.
//
// Solidity: function sendContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, uint256 value, bytes data) returns(uint256)
func (_IInbox *IInboxSession) SendContractTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendContractTransaction(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, to, value, data)
}

// SendContractTransaction is a paid mutator transaction binding the contract method 0x8a631aa6.
//
// Solidity: function sendContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, uint256 value, bytes data) returns(uint256)
func (_IInbox *IInboxTransactorSession) SendContractTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendContractTransaction(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, to, value, data)
}

// SendL1FundedContractTransaction is a paid mutator transaction binding the contract method 0x5e916758.
//
// Solidity: function sendL1FundedContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, bytes data) payable returns(uint256)
func (_IInbox *IInboxTransactor) SendL1FundedContractTransaction(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "sendL1FundedContractTransaction", gasLimit, maxFeePerGas, to, data)
}

// SendL1FundedContractTransaction is a paid mutator transaction binding the contract method 0x5e916758.
//
// Solidity: function sendL1FundedContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, bytes data) payable returns(uint256)
func (_IInbox *IInboxSession) SendL1FundedContractTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendL1FundedContractTransaction(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, to, data)
}

// SendL1FundedContractTransaction is a paid mutator transaction binding the contract method 0x5e916758.
//
// Solidity: function sendL1FundedContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, bytes data) payable returns(uint256)
func (_IInbox *IInboxTransactorSession) SendL1FundedContractTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendL1FundedContractTransaction(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, to, data)
}

// SendL1FundedUnsignedTransaction is a paid mutator transaction binding the contract method 0x67ef3ab8.
//
// Solidity: function sendL1FundedUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_IInbox *IInboxTransactor) SendL1FundedUnsignedTransaction(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "sendL1FundedUnsignedTransaction", gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL1FundedUnsignedTransaction is a paid mutator transaction binding the contract method 0x67ef3ab8.
//
// Solidity: function sendL1FundedUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_IInbox *IInboxSession) SendL1FundedUnsignedTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendL1FundedUnsignedTransaction(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL1FundedUnsignedTransaction is a paid mutator transaction binding the contract method 0x67ef3ab8.
//
// Solidity: function sendL1FundedUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_IInbox *IInboxTransactorSession) SendL1FundedUnsignedTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendL1FundedUnsignedTransaction(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL1FundedUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0xe6bd12cf.
//
// Solidity: function sendL1FundedUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_IInbox *IInboxTransactor) SendL1FundedUnsignedTransactionToFork(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "sendL1FundedUnsignedTransactionToFork", gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL1FundedUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0xe6bd12cf.
//
// Solidity: function sendL1FundedUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_IInbox *IInboxSession) SendL1FundedUnsignedTransactionToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendL1FundedUnsignedTransactionToFork(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL1FundedUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0xe6bd12cf.
//
// Solidity: function sendL1FundedUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_IInbox *IInboxTransactorSession) SendL1FundedUnsignedTransactionToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendL1FundedUnsignedTransactionToFork(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL2Message is a paid mutator transaction binding the contract method 0xb75436bb.
//
// Solidity: function sendL2Message(bytes messageData) returns(uint256)
func (_IInbox *IInboxTransactor) SendL2Message(opts *bind.TransactOpts, messageData []byte) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "sendL2Message", messageData)
}

// SendL2Message is a paid mutator transaction binding the contract method 0xb75436bb.
//
// Solidity: function sendL2Message(bytes messageData) returns(uint256)
func (_IInbox *IInboxSession) SendL2Message(messageData []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendL2Message(&_IInbox.TransactOpts, messageData)
}

// SendL2Message is a paid mutator transaction binding the contract method 0xb75436bb.
//
// Solidity: function sendL2Message(bytes messageData) returns(uint256)
func (_IInbox *IInboxTransactorSession) SendL2Message(messageData []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendL2Message(&_IInbox.TransactOpts, messageData)
}

// SendL2MessageFromOrigin is a paid mutator transaction binding the contract method 0x1fe927cf.
//
// Solidity: function sendL2MessageFromOrigin(bytes messageData) returns(uint256)
func (_IInbox *IInboxTransactor) SendL2MessageFromOrigin(opts *bind.TransactOpts, messageData []byte) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "sendL2MessageFromOrigin", messageData)
}

// SendL2MessageFromOrigin is a paid mutator transaction binding the contract method 0x1fe927cf.
//
// Solidity: function sendL2MessageFromOrigin(bytes messageData) returns(uint256)
func (_IInbox *IInboxSession) SendL2MessageFromOrigin(messageData []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendL2MessageFromOrigin(&_IInbox.TransactOpts, messageData)
}

// SendL2MessageFromOrigin is a paid mutator transaction binding the contract method 0x1fe927cf.
//
// Solidity: function sendL2MessageFromOrigin(bytes messageData) returns(uint256)
func (_IInbox *IInboxTransactorSession) SendL2MessageFromOrigin(messageData []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendL2MessageFromOrigin(&_IInbox.TransactOpts, messageData)
}

// SendUnsignedTransaction is a paid mutator transaction binding the contract method 0x5075788b.
//
// Solidity: function sendUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_IInbox *IInboxTransactor) SendUnsignedTransaction(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "sendUnsignedTransaction", gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendUnsignedTransaction is a paid mutator transaction binding the contract method 0x5075788b.
//
// Solidity: function sendUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_IInbox *IInboxSession) SendUnsignedTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendUnsignedTransaction(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendUnsignedTransaction is a paid mutator transaction binding the contract method 0x5075788b.
//
// Solidity: function sendUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_IInbox *IInboxTransactorSession) SendUnsignedTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendUnsignedTransaction(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_IInbox *IInboxTransactor) SendUnsignedTransactionToFork(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "sendUnsignedTransactionToFork", gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_IInbox *IInboxSession) SendUnsignedTransactionToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendUnsignedTransactionToFork(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_IInbox *IInboxTransactorSession) SendUnsignedTransactionToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.SendUnsignedTransactionToFork(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendWithdrawEthToFork is a paid mutator transaction binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, uint256 value, address withdrawTo) returns(uint256)
func (_IInbox *IInboxTransactor) SendWithdrawEthToFork(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, value *big.Int, withdrawTo common.Address) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "sendWithdrawEthToFork", gasLimit, maxFeePerGas, nonce, value, withdrawTo)
}

// SendWithdrawEthToFork is a paid mutator transaction binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, uint256 value, address withdrawTo) returns(uint256)
func (_IInbox *IInboxSession) SendWithdrawEthToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, value *big.Int, withdrawTo common.Address) (*types.Transaction, error) {
	return _IInbox.Contract.SendWithdrawEthToFork(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, nonce, value, withdrawTo)
}

// SendWithdrawEthToFork is a paid mutator transaction binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, uint256 value, address withdrawTo) returns(uint256)
func (_IInbox *IInboxTransactorSession) SendWithdrawEthToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, value *big.Int, withdrawTo common.Address) (*types.Transaction, error) {
	return _IInbox.Contract.SendWithdrawEthToFork(&_IInbox.TransactOpts, gasLimit, maxFeePerGas, nonce, value, withdrawTo)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_IInbox *IInboxTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_IInbox *IInboxSession) Unpause() (*types.Transaction, error) {
	return _IInbox.Contract.Unpause(&_IInbox.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_IInbox *IInboxTransactorSession) Unpause() (*types.Transaction, error) {
	return _IInbox.Contract.Unpause(&_IInbox.TransactOpts)
}

// UnsafeCreateRetryableTicket is a paid mutator transaction binding the contract method 0x6e6e8a6a.
//
// Solidity: function unsafeCreateRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_IInbox *IInboxTransactor) UnsafeCreateRetryableTicket(opts *bind.TransactOpts, to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "unsafeCreateRetryableTicket", to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// UnsafeCreateRetryableTicket is a paid mutator transaction binding the contract method 0x6e6e8a6a.
//
// Solidity: function unsafeCreateRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_IInbox *IInboxSession) UnsafeCreateRetryableTicket(to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.UnsafeCreateRetryableTicket(&_IInbox.TransactOpts, to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// UnsafeCreateRetryableTicket is a paid mutator transaction binding the contract method 0x6e6e8a6a.
//
// Solidity: function unsafeCreateRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_IInbox *IInboxTransactorSession) UnsafeCreateRetryableTicket(to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _IInbox.Contract.UnsafeCreateRetryableTicket(&_IInbox.TransactOpts, to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// IInboxInboxMessageDeliveredIterator is returned from FilterInboxMessageDelivered and is used to iterate over the raw logs and unpacked data for InboxMessageDelivered events raised by the IInbox contract.
type IInboxInboxMessageDeliveredIterator struct {
	Event *IInboxInboxMessageDelivered // Event containing the contract specifics and raw log

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
func (it *IInboxInboxMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IInboxInboxMessageDelivered)
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
		it.Event = new(IInboxInboxMessageDelivered)
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
func (it *IInboxInboxMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IInboxInboxMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IInboxInboxMessageDelivered represents a InboxMessageDelivered event raised by the IInbox contract.
type IInboxInboxMessageDelivered struct {
	MessageNum *big.Int
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDelivered is a free log retrieval operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_IInbox *IInboxFilterer) FilterInboxMessageDelivered(opts *bind.FilterOpts, messageNum []*big.Int) (*IInboxInboxMessageDeliveredIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _IInbox.contract.FilterLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &IInboxInboxMessageDeliveredIterator{contract: _IInbox.contract, event: "InboxMessageDelivered", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDelivered is a free log subscription operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_IInbox *IInboxFilterer) WatchInboxMessageDelivered(opts *bind.WatchOpts, sink chan<- *IInboxInboxMessageDelivered, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _IInbox.contract.WatchLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IInboxInboxMessageDelivered)
				if err := _IInbox.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
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
func (_IInbox *IInboxFilterer) ParseInboxMessageDelivered(log types.Log) (*IInboxInboxMessageDelivered, error) {
	event := new(IInboxInboxMessageDelivered)
	if err := _IInbox.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IInboxInboxMessageDeliveredFromOriginIterator is returned from FilterInboxMessageDeliveredFromOrigin and is used to iterate over the raw logs and unpacked data for InboxMessageDeliveredFromOrigin events raised by the IInbox contract.
type IInboxInboxMessageDeliveredFromOriginIterator struct {
	Event *IInboxInboxMessageDeliveredFromOrigin // Event containing the contract specifics and raw log

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
func (it *IInboxInboxMessageDeliveredFromOriginIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IInboxInboxMessageDeliveredFromOrigin)
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
		it.Event = new(IInboxInboxMessageDeliveredFromOrigin)
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
func (it *IInboxInboxMessageDeliveredFromOriginIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IInboxInboxMessageDeliveredFromOriginIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IInboxInboxMessageDeliveredFromOrigin represents a InboxMessageDeliveredFromOrigin event raised by the IInbox contract.
type IInboxInboxMessageDeliveredFromOrigin struct {
	MessageNum *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDeliveredFromOrigin is a free log retrieval operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_IInbox *IInboxFilterer) FilterInboxMessageDeliveredFromOrigin(opts *bind.FilterOpts, messageNum []*big.Int) (*IInboxInboxMessageDeliveredFromOriginIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _IInbox.contract.FilterLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &IInboxInboxMessageDeliveredFromOriginIterator{contract: _IInbox.contract, event: "InboxMessageDeliveredFromOrigin", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDeliveredFromOrigin is a free log subscription operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_IInbox *IInboxFilterer) WatchInboxMessageDeliveredFromOrigin(opts *bind.WatchOpts, sink chan<- *IInboxInboxMessageDeliveredFromOrigin, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _IInbox.contract.WatchLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IInboxInboxMessageDeliveredFromOrigin)
				if err := _IInbox.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
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
func (_IInbox *IInboxFilterer) ParseInboxMessageDeliveredFromOrigin(log types.Log) (*IInboxInboxMessageDeliveredFromOrigin, error) {
	event := new(IInboxInboxMessageDeliveredFromOrigin)
	if err := _IInbox.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOutboxMetaData contains all meta data concerning the IOutbox contract.
var IOutboxMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"zero\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"transactionIndex\",\"type\":\"uint256\"}],\"name\":\"OutBoxTransactionExecuted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"l2BlockHash\",\"type\":\"bytes32\"}],\"name\":\"SendRootUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"OUTBOX_VERSION\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l2Timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"calculateItemHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"path\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"item\",\"type\":\"bytes32\"}],\"name\":\"calculateMerkleRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l2Timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l2Timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeTransactionSimulation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"isSpent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1Block\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1EthBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1OutputId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1Sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1Timestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"roots\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"spent\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"sendRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"l2BlockHash\",\"type\":\"bytes32\"}],\"name\":\"updateSendRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IOutboxABI is the input ABI used to generate the binding from.
// Deprecated: Use IOutboxMetaData.ABI instead.
var IOutboxABI = IOutboxMetaData.ABI

// IOutbox is an auto generated Go binding around an Ethereum contract.
type IOutbox struct {
	IOutboxCaller     // Read-only binding to the contract
	IOutboxTransactor // Write-only binding to the contract
	IOutboxFilterer   // Log filterer for contract events
}

// IOutboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type IOutboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOutboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IOutboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOutboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IOutboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOutboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IOutboxSession struct {
	Contract     *IOutbox          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IOutboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IOutboxCallerSession struct {
	Contract *IOutboxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// IOutboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IOutboxTransactorSession struct {
	Contract     *IOutboxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// IOutboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type IOutboxRaw struct {
	Contract *IOutbox // Generic contract binding to access the raw methods on
}

// IOutboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IOutboxCallerRaw struct {
	Contract *IOutboxCaller // Generic read-only contract binding to access the raw methods on
}

// IOutboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IOutboxTransactorRaw struct {
	Contract *IOutboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIOutbox creates a new instance of IOutbox, bound to a specific deployed contract.
func NewIOutbox(address common.Address, backend bind.ContractBackend) (*IOutbox, error) {
	contract, err := bindIOutbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IOutbox{IOutboxCaller: IOutboxCaller{contract: contract}, IOutboxTransactor: IOutboxTransactor{contract: contract}, IOutboxFilterer: IOutboxFilterer{contract: contract}}, nil
}

// NewIOutboxCaller creates a new read-only instance of IOutbox, bound to a specific deployed contract.
func NewIOutboxCaller(address common.Address, caller bind.ContractCaller) (*IOutboxCaller, error) {
	contract, err := bindIOutbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IOutboxCaller{contract: contract}, nil
}

// NewIOutboxTransactor creates a new write-only instance of IOutbox, bound to a specific deployed contract.
func NewIOutboxTransactor(address common.Address, transactor bind.ContractTransactor) (*IOutboxTransactor, error) {
	contract, err := bindIOutbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IOutboxTransactor{contract: contract}, nil
}

// NewIOutboxFilterer creates a new log filterer instance of IOutbox, bound to a specific deployed contract.
func NewIOutboxFilterer(address common.Address, filterer bind.ContractFilterer) (*IOutboxFilterer, error) {
	contract, err := bindIOutbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IOutboxFilterer{contract: contract}, nil
}

// bindIOutbox binds a generic wrapper to an already deployed contract.
func bindIOutbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IOutboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOutbox *IOutboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOutbox.Contract.IOutboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOutbox *IOutboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOutbox.Contract.IOutboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOutbox *IOutboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOutbox.Contract.IOutboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOutbox *IOutboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOutbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOutbox *IOutboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOutbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOutbox *IOutboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOutbox.Contract.contract.Transact(opts, method, params...)
}

// OUTBOXVERSION is a free data retrieval call binding the contract method 0xc75184df.
//
// Solidity: function OUTBOX_VERSION() view returns(uint128)
func (_IOutbox *IOutboxCaller) OUTBOXVERSION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "OUTBOX_VERSION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OUTBOXVERSION is a free data retrieval call binding the contract method 0xc75184df.
//
// Solidity: function OUTBOX_VERSION() view returns(uint128)
func (_IOutbox *IOutboxSession) OUTBOXVERSION() (*big.Int, error) {
	return _IOutbox.Contract.OUTBOXVERSION(&_IOutbox.CallOpts)
}

// OUTBOXVERSION is a free data retrieval call binding the contract method 0xc75184df.
//
// Solidity: function OUTBOX_VERSION() view returns(uint128)
func (_IOutbox *IOutboxCallerSession) OUTBOXVERSION() (*big.Int, error) {
	return _IOutbox.Contract.OUTBOXVERSION(&_IOutbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_IOutbox *IOutboxCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_IOutbox *IOutboxSession) Bridge() (common.Address, error) {
	return _IOutbox.Contract.Bridge(&_IOutbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_IOutbox *IOutboxCallerSession) Bridge() (common.Address, error) {
	return _IOutbox.Contract.Bridge(&_IOutbox.CallOpts)
}

// CalculateItemHash is a free data retrieval call binding the contract method 0x9f0c04bf.
//
// Solidity: function calculateItemHash(address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) pure returns(bytes32)
func (_IOutbox *IOutboxCaller) CalculateItemHash(opts *bind.CallOpts, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) ([32]byte, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "calculateItemHash", l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateItemHash is a free data retrieval call binding the contract method 0x9f0c04bf.
//
// Solidity: function calculateItemHash(address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) pure returns(bytes32)
func (_IOutbox *IOutboxSession) CalculateItemHash(l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) ([32]byte, error) {
	return _IOutbox.Contract.CalculateItemHash(&_IOutbox.CallOpts, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// CalculateItemHash is a free data retrieval call binding the contract method 0x9f0c04bf.
//
// Solidity: function calculateItemHash(address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) pure returns(bytes32)
func (_IOutbox *IOutboxCallerSession) CalculateItemHash(l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) ([32]byte, error) {
	return _IOutbox.Contract.CalculateItemHash(&_IOutbox.CallOpts, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// CalculateMerkleRoot is a free data retrieval call binding the contract method 0x007436d3.
//
// Solidity: function calculateMerkleRoot(bytes32[] proof, uint256 path, bytes32 item) pure returns(bytes32)
func (_IOutbox *IOutboxCaller) CalculateMerkleRoot(opts *bind.CallOpts, proof [][32]byte, path *big.Int, item [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "calculateMerkleRoot", proof, path, item)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateMerkleRoot is a free data retrieval call binding the contract method 0x007436d3.
//
// Solidity: function calculateMerkleRoot(bytes32[] proof, uint256 path, bytes32 item) pure returns(bytes32)
func (_IOutbox *IOutboxSession) CalculateMerkleRoot(proof [][32]byte, path *big.Int, item [32]byte) ([32]byte, error) {
	return _IOutbox.Contract.CalculateMerkleRoot(&_IOutbox.CallOpts, proof, path, item)
}

// CalculateMerkleRoot is a free data retrieval call binding the contract method 0x007436d3.
//
// Solidity: function calculateMerkleRoot(bytes32[] proof, uint256 path, bytes32 item) pure returns(bytes32)
func (_IOutbox *IOutboxCallerSession) CalculateMerkleRoot(proof [][32]byte, path *big.Int, item [32]byte) ([32]byte, error) {
	return _IOutbox.Contract.CalculateMerkleRoot(&_IOutbox.CallOpts, proof, path, item)
}

// IsSpent is a free data retrieval call binding the contract method 0x5a129efe.
//
// Solidity: function isSpent(uint256 index) view returns(bool)
func (_IOutbox *IOutboxCaller) IsSpent(opts *bind.CallOpts, index *big.Int) (bool, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "isSpent", index)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSpent is a free data retrieval call binding the contract method 0x5a129efe.
//
// Solidity: function isSpent(uint256 index) view returns(bool)
func (_IOutbox *IOutboxSession) IsSpent(index *big.Int) (bool, error) {
	return _IOutbox.Contract.IsSpent(&_IOutbox.CallOpts, index)
}

// IsSpent is a free data retrieval call binding the contract method 0x5a129efe.
//
// Solidity: function isSpent(uint256 index) view returns(bool)
func (_IOutbox *IOutboxCallerSession) IsSpent(index *big.Int) (bool, error) {
	return _IOutbox.Contract.IsSpent(&_IOutbox.CallOpts, index)
}

// L2ToL1Block is a free data retrieval call binding the contract method 0x46547790.
//
// Solidity: function l2ToL1Block() view returns(uint256)
func (_IOutbox *IOutboxCaller) L2ToL1Block(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "l2ToL1Block")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1Block is a free data retrieval call binding the contract method 0x46547790.
//
// Solidity: function l2ToL1Block() view returns(uint256)
func (_IOutbox *IOutboxSession) L2ToL1Block() (*big.Int, error) {
	return _IOutbox.Contract.L2ToL1Block(&_IOutbox.CallOpts)
}

// L2ToL1Block is a free data retrieval call binding the contract method 0x46547790.
//
// Solidity: function l2ToL1Block() view returns(uint256)
func (_IOutbox *IOutboxCallerSession) L2ToL1Block() (*big.Int, error) {
	return _IOutbox.Contract.L2ToL1Block(&_IOutbox.CallOpts)
}

// L2ToL1EthBlock is a free data retrieval call binding the contract method 0x8515bc6a.
//
// Solidity: function l2ToL1EthBlock() view returns(uint256)
func (_IOutbox *IOutboxCaller) L2ToL1EthBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "l2ToL1EthBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1EthBlock is a free data retrieval call binding the contract method 0x8515bc6a.
//
// Solidity: function l2ToL1EthBlock() view returns(uint256)
func (_IOutbox *IOutboxSession) L2ToL1EthBlock() (*big.Int, error) {
	return _IOutbox.Contract.L2ToL1EthBlock(&_IOutbox.CallOpts)
}

// L2ToL1EthBlock is a free data retrieval call binding the contract method 0x8515bc6a.
//
// Solidity: function l2ToL1EthBlock() view returns(uint256)
func (_IOutbox *IOutboxCallerSession) L2ToL1EthBlock() (*big.Int, error) {
	return _IOutbox.Contract.L2ToL1EthBlock(&_IOutbox.CallOpts)
}

// L2ToL1OutputId is a free data retrieval call binding the contract method 0x72f2a8c7.
//
// Solidity: function l2ToL1OutputId() view returns(bytes32)
func (_IOutbox *IOutboxCaller) L2ToL1OutputId(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "l2ToL1OutputId")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L2ToL1OutputId is a free data retrieval call binding the contract method 0x72f2a8c7.
//
// Solidity: function l2ToL1OutputId() view returns(bytes32)
func (_IOutbox *IOutboxSession) L2ToL1OutputId() ([32]byte, error) {
	return _IOutbox.Contract.L2ToL1OutputId(&_IOutbox.CallOpts)
}

// L2ToL1OutputId is a free data retrieval call binding the contract method 0x72f2a8c7.
//
// Solidity: function l2ToL1OutputId() view returns(bytes32)
func (_IOutbox *IOutboxCallerSession) L2ToL1OutputId() ([32]byte, error) {
	return _IOutbox.Contract.L2ToL1OutputId(&_IOutbox.CallOpts)
}

// L2ToL1Sender is a free data retrieval call binding the contract method 0x80648b02.
//
// Solidity: function l2ToL1Sender() view returns(address)
func (_IOutbox *IOutboxCaller) L2ToL1Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "l2ToL1Sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2ToL1Sender is a free data retrieval call binding the contract method 0x80648b02.
//
// Solidity: function l2ToL1Sender() view returns(address)
func (_IOutbox *IOutboxSession) L2ToL1Sender() (common.Address, error) {
	return _IOutbox.Contract.L2ToL1Sender(&_IOutbox.CallOpts)
}

// L2ToL1Sender is a free data retrieval call binding the contract method 0x80648b02.
//
// Solidity: function l2ToL1Sender() view returns(address)
func (_IOutbox *IOutboxCallerSession) L2ToL1Sender() (common.Address, error) {
	return _IOutbox.Contract.L2ToL1Sender(&_IOutbox.CallOpts)
}

// L2ToL1Timestamp is a free data retrieval call binding the contract method 0xb0f30537.
//
// Solidity: function l2ToL1Timestamp() view returns(uint256)
func (_IOutbox *IOutboxCaller) L2ToL1Timestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "l2ToL1Timestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1Timestamp is a free data retrieval call binding the contract method 0xb0f30537.
//
// Solidity: function l2ToL1Timestamp() view returns(uint256)
func (_IOutbox *IOutboxSession) L2ToL1Timestamp() (*big.Int, error) {
	return _IOutbox.Contract.L2ToL1Timestamp(&_IOutbox.CallOpts)
}

// L2ToL1Timestamp is a free data retrieval call binding the contract method 0xb0f30537.
//
// Solidity: function l2ToL1Timestamp() view returns(uint256)
func (_IOutbox *IOutboxCallerSession) L2ToL1Timestamp() (*big.Int, error) {
	return _IOutbox.Contract.L2ToL1Timestamp(&_IOutbox.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_IOutbox *IOutboxCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_IOutbox *IOutboxSession) Rollup() (common.Address, error) {
	return _IOutbox.Contract.Rollup(&_IOutbox.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_IOutbox *IOutboxCallerSession) Rollup() (common.Address, error) {
	return _IOutbox.Contract.Rollup(&_IOutbox.CallOpts)
}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(bytes32)
func (_IOutbox *IOutboxCaller) Roots(opts *bind.CallOpts, arg0 [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "roots", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(bytes32)
func (_IOutbox *IOutboxSession) Roots(arg0 [32]byte) ([32]byte, error) {
	return _IOutbox.Contract.Roots(&_IOutbox.CallOpts, arg0)
}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(bytes32)
func (_IOutbox *IOutboxCallerSession) Roots(arg0 [32]byte) ([32]byte, error) {
	return _IOutbox.Contract.Roots(&_IOutbox.CallOpts, arg0)
}

// Spent is a free data retrieval call binding the contract method 0xd5b5cc23.
//
// Solidity: function spent(uint256 ) view returns(bytes32)
func (_IOutbox *IOutboxCaller) Spent(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _IOutbox.contract.Call(opts, &out, "spent", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Spent is a free data retrieval call binding the contract method 0xd5b5cc23.
//
// Solidity: function spent(uint256 ) view returns(bytes32)
func (_IOutbox *IOutboxSession) Spent(arg0 *big.Int) ([32]byte, error) {
	return _IOutbox.Contract.Spent(&_IOutbox.CallOpts, arg0)
}

// Spent is a free data retrieval call binding the contract method 0xd5b5cc23.
//
// Solidity: function spent(uint256 ) view returns(bytes32)
func (_IOutbox *IOutboxCallerSession) Spent(arg0 *big.Int) ([32]byte, error) {
	return _IOutbox.Contract.Spent(&_IOutbox.CallOpts, arg0)
}

// ExecuteTransaction is a paid mutator transaction binding the contract method 0x08635a95.
//
// Solidity: function executeTransaction(bytes32[] proof, uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_IOutbox *IOutboxTransactor) ExecuteTransaction(opts *bind.TransactOpts, proof [][32]byte, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IOutbox.contract.Transact(opts, "executeTransaction", proof, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransaction is a paid mutator transaction binding the contract method 0x08635a95.
//
// Solidity: function executeTransaction(bytes32[] proof, uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_IOutbox *IOutboxSession) ExecuteTransaction(proof [][32]byte, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IOutbox.Contract.ExecuteTransaction(&_IOutbox.TransactOpts, proof, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransaction is a paid mutator transaction binding the contract method 0x08635a95.
//
// Solidity: function executeTransaction(bytes32[] proof, uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_IOutbox *IOutboxTransactorSession) ExecuteTransaction(proof [][32]byte, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IOutbox.Contract.ExecuteTransaction(&_IOutbox.TransactOpts, proof, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransactionSimulation is a paid mutator transaction binding the contract method 0x288e5b10.
//
// Solidity: function executeTransactionSimulation(uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_IOutbox *IOutboxTransactor) ExecuteTransactionSimulation(opts *bind.TransactOpts, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IOutbox.contract.Transact(opts, "executeTransactionSimulation", index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransactionSimulation is a paid mutator transaction binding the contract method 0x288e5b10.
//
// Solidity: function executeTransactionSimulation(uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_IOutbox *IOutboxSession) ExecuteTransactionSimulation(index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IOutbox.Contract.ExecuteTransactionSimulation(&_IOutbox.TransactOpts, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransactionSimulation is a paid mutator transaction binding the contract method 0x288e5b10.
//
// Solidity: function executeTransactionSimulation(uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_IOutbox *IOutboxTransactorSession) ExecuteTransactionSimulation(index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _IOutbox.Contract.ExecuteTransactionSimulation(&_IOutbox.TransactOpts, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_IOutbox *IOutboxTransactor) UpdateRollupAddress(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOutbox.contract.Transact(opts, "updateRollupAddress")
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_IOutbox *IOutboxSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _IOutbox.Contract.UpdateRollupAddress(&_IOutbox.TransactOpts)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_IOutbox *IOutboxTransactorSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _IOutbox.Contract.UpdateRollupAddress(&_IOutbox.TransactOpts)
}

// UpdateSendRoot is a paid mutator transaction binding the contract method 0xa04cee60.
//
// Solidity: function updateSendRoot(bytes32 sendRoot, bytes32 l2BlockHash) returns()
func (_IOutbox *IOutboxTransactor) UpdateSendRoot(opts *bind.TransactOpts, sendRoot [32]byte, l2BlockHash [32]byte) (*types.Transaction, error) {
	return _IOutbox.contract.Transact(opts, "updateSendRoot", sendRoot, l2BlockHash)
}

// UpdateSendRoot is a paid mutator transaction binding the contract method 0xa04cee60.
//
// Solidity: function updateSendRoot(bytes32 sendRoot, bytes32 l2BlockHash) returns()
func (_IOutbox *IOutboxSession) UpdateSendRoot(sendRoot [32]byte, l2BlockHash [32]byte) (*types.Transaction, error) {
	return _IOutbox.Contract.UpdateSendRoot(&_IOutbox.TransactOpts, sendRoot, l2BlockHash)
}

// UpdateSendRoot is a paid mutator transaction binding the contract method 0xa04cee60.
//
// Solidity: function updateSendRoot(bytes32 sendRoot, bytes32 l2BlockHash) returns()
func (_IOutbox *IOutboxTransactorSession) UpdateSendRoot(sendRoot [32]byte, l2BlockHash [32]byte) (*types.Transaction, error) {
	return _IOutbox.Contract.UpdateSendRoot(&_IOutbox.TransactOpts, sendRoot, l2BlockHash)
}

// IOutboxOutBoxTransactionExecutedIterator is returned from FilterOutBoxTransactionExecuted and is used to iterate over the raw logs and unpacked data for OutBoxTransactionExecuted events raised by the IOutbox contract.
type IOutboxOutBoxTransactionExecutedIterator struct {
	Event *IOutboxOutBoxTransactionExecuted // Event containing the contract specifics and raw log

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
func (it *IOutboxOutBoxTransactionExecutedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IOutboxOutBoxTransactionExecuted)
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
		it.Event = new(IOutboxOutBoxTransactionExecuted)
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
func (it *IOutboxOutBoxTransactionExecutedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IOutboxOutBoxTransactionExecutedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IOutboxOutBoxTransactionExecuted represents a OutBoxTransactionExecuted event raised by the IOutbox contract.
type IOutboxOutBoxTransactionExecuted struct {
	To               common.Address
	L2Sender         common.Address
	Zero             *big.Int
	TransactionIndex *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterOutBoxTransactionExecuted is a free log retrieval operation binding the contract event 0x20af7f3bbfe38132b8900ae295cd9c8d1914be7052d061a511f3f728dab18964.
//
// Solidity: event OutBoxTransactionExecuted(address indexed to, address indexed l2Sender, uint256 indexed zero, uint256 transactionIndex)
func (_IOutbox *IOutboxFilterer) FilterOutBoxTransactionExecuted(opts *bind.FilterOpts, to []common.Address, l2Sender []common.Address, zero []*big.Int) (*IOutboxOutBoxTransactionExecutedIterator, error) {

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

	logs, sub, err := _IOutbox.contract.FilterLogs(opts, "OutBoxTransactionExecuted", toRule, l2SenderRule, zeroRule)
	if err != nil {
		return nil, err
	}
	return &IOutboxOutBoxTransactionExecutedIterator{contract: _IOutbox.contract, event: "OutBoxTransactionExecuted", logs: logs, sub: sub}, nil
}

// WatchOutBoxTransactionExecuted is a free log subscription operation binding the contract event 0x20af7f3bbfe38132b8900ae295cd9c8d1914be7052d061a511f3f728dab18964.
//
// Solidity: event OutBoxTransactionExecuted(address indexed to, address indexed l2Sender, uint256 indexed zero, uint256 transactionIndex)
func (_IOutbox *IOutboxFilterer) WatchOutBoxTransactionExecuted(opts *bind.WatchOpts, sink chan<- *IOutboxOutBoxTransactionExecuted, to []common.Address, l2Sender []common.Address, zero []*big.Int) (event.Subscription, error) {

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

	logs, sub, err := _IOutbox.contract.WatchLogs(opts, "OutBoxTransactionExecuted", toRule, l2SenderRule, zeroRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IOutboxOutBoxTransactionExecuted)
				if err := _IOutbox.contract.UnpackLog(event, "OutBoxTransactionExecuted", log); err != nil {
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
func (_IOutbox *IOutboxFilterer) ParseOutBoxTransactionExecuted(log types.Log) (*IOutboxOutBoxTransactionExecuted, error) {
	event := new(IOutboxOutBoxTransactionExecuted)
	if err := _IOutbox.contract.UnpackLog(event, "OutBoxTransactionExecuted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOutboxSendRootUpdatedIterator is returned from FilterSendRootUpdated and is used to iterate over the raw logs and unpacked data for SendRootUpdated events raised by the IOutbox contract.
type IOutboxSendRootUpdatedIterator struct {
	Event *IOutboxSendRootUpdated // Event containing the contract specifics and raw log

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
func (it *IOutboxSendRootUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IOutboxSendRootUpdated)
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
		it.Event = new(IOutboxSendRootUpdated)
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
func (it *IOutboxSendRootUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IOutboxSendRootUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IOutboxSendRootUpdated represents a SendRootUpdated event raised by the IOutbox contract.
type IOutboxSendRootUpdated struct {
	OutputRoot  [32]byte
	L2BlockHash [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSendRootUpdated is a free log retrieval operation binding the contract event 0xb4df3847300f076a369cd76d2314b470a1194d9e8a6bb97f1860aee88a5f6748.
//
// Solidity: event SendRootUpdated(bytes32 indexed outputRoot, bytes32 indexed l2BlockHash)
func (_IOutbox *IOutboxFilterer) FilterSendRootUpdated(opts *bind.FilterOpts, outputRoot [][32]byte, l2BlockHash [][32]byte) (*IOutboxSendRootUpdatedIterator, error) {

	var outputRootRule []interface{}
	for _, outputRootItem := range outputRoot {
		outputRootRule = append(outputRootRule, outputRootItem)
	}
	var l2BlockHashRule []interface{}
	for _, l2BlockHashItem := range l2BlockHash {
		l2BlockHashRule = append(l2BlockHashRule, l2BlockHashItem)
	}

	logs, sub, err := _IOutbox.contract.FilterLogs(opts, "SendRootUpdated", outputRootRule, l2BlockHashRule)
	if err != nil {
		return nil, err
	}
	return &IOutboxSendRootUpdatedIterator{contract: _IOutbox.contract, event: "SendRootUpdated", logs: logs, sub: sub}, nil
}

// WatchSendRootUpdated is a free log subscription operation binding the contract event 0xb4df3847300f076a369cd76d2314b470a1194d9e8a6bb97f1860aee88a5f6748.
//
// Solidity: event SendRootUpdated(bytes32 indexed outputRoot, bytes32 indexed l2BlockHash)
func (_IOutbox *IOutboxFilterer) WatchSendRootUpdated(opts *bind.WatchOpts, sink chan<- *IOutboxSendRootUpdated, outputRoot [][32]byte, l2BlockHash [][32]byte) (event.Subscription, error) {

	var outputRootRule []interface{}
	for _, outputRootItem := range outputRoot {
		outputRootRule = append(outputRootRule, outputRootItem)
	}
	var l2BlockHashRule []interface{}
	for _, l2BlockHashItem := range l2BlockHash {
		l2BlockHashRule = append(l2BlockHashRule, l2BlockHashItem)
	}

	logs, sub, err := _IOutbox.contract.WatchLogs(opts, "SendRootUpdated", outputRootRule, l2BlockHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IOutboxSendRootUpdated)
				if err := _IOutbox.contract.UnpackLog(event, "SendRootUpdated", log); err != nil {
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
func (_IOutbox *IOutboxFilterer) ParseSendRootUpdated(log types.Log) (*IOutboxSendRootUpdated, error) {
	event := new(IOutboxSendRootUpdated)
	if err := _IOutbox.contract.UnpackLog(event, "SendRootUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOwnableMetaData contains all meta data concerning the IOwnable contract.
var IOwnableMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IOwnableABI is the input ABI used to generate the binding from.
// Deprecated: Use IOwnableMetaData.ABI instead.
var IOwnableABI = IOwnableMetaData.ABI

// IOwnable is an auto generated Go binding around an Ethereum contract.
type IOwnable struct {
	IOwnableCaller     // Read-only binding to the contract
	IOwnableTransactor // Write-only binding to the contract
	IOwnableFilterer   // Log filterer for contract events
}

// IOwnableCaller is an auto generated read-only Go binding around an Ethereum contract.
type IOwnableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOwnableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IOwnableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOwnableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IOwnableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOwnableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IOwnableSession struct {
	Contract     *IOwnable         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IOwnableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IOwnableCallerSession struct {
	Contract *IOwnableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// IOwnableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IOwnableTransactorSession struct {
	Contract     *IOwnableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// IOwnableRaw is an auto generated low-level Go binding around an Ethereum contract.
type IOwnableRaw struct {
	Contract *IOwnable // Generic contract binding to access the raw methods on
}

// IOwnableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IOwnableCallerRaw struct {
	Contract *IOwnableCaller // Generic read-only contract binding to access the raw methods on
}

// IOwnableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IOwnableTransactorRaw struct {
	Contract *IOwnableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIOwnable creates a new instance of IOwnable, bound to a specific deployed contract.
func NewIOwnable(address common.Address, backend bind.ContractBackend) (*IOwnable, error) {
	contract, err := bindIOwnable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IOwnable{IOwnableCaller: IOwnableCaller{contract: contract}, IOwnableTransactor: IOwnableTransactor{contract: contract}, IOwnableFilterer: IOwnableFilterer{contract: contract}}, nil
}

// NewIOwnableCaller creates a new read-only instance of IOwnable, bound to a specific deployed contract.
func NewIOwnableCaller(address common.Address, caller bind.ContractCaller) (*IOwnableCaller, error) {
	contract, err := bindIOwnable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IOwnableCaller{contract: contract}, nil
}

// NewIOwnableTransactor creates a new write-only instance of IOwnable, bound to a specific deployed contract.
func NewIOwnableTransactor(address common.Address, transactor bind.ContractTransactor) (*IOwnableTransactor, error) {
	contract, err := bindIOwnable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IOwnableTransactor{contract: contract}, nil
}

// NewIOwnableFilterer creates a new log filterer instance of IOwnable, bound to a specific deployed contract.
func NewIOwnableFilterer(address common.Address, filterer bind.ContractFilterer) (*IOwnableFilterer, error) {
	contract, err := bindIOwnable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IOwnableFilterer{contract: contract}, nil
}

// bindIOwnable binds a generic wrapper to an already deployed contract.
func bindIOwnable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IOwnableMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOwnable *IOwnableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOwnable.Contract.IOwnableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOwnable *IOwnableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOwnable.Contract.IOwnableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOwnable *IOwnableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOwnable.Contract.IOwnableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOwnable *IOwnableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOwnable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOwnable *IOwnableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOwnable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOwnable *IOwnableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOwnable.Contract.contract.Transact(opts, method, params...)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_IOwnable *IOwnableCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IOwnable.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_IOwnable *IOwnableSession) Owner() (common.Address, error) {
	return _IOwnable.Contract.Owner(&_IOwnable.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_IOwnable *IOwnableCallerSession) Owner() (common.Address, error) {
	return _IOwnable.Contract.Owner(&_IOwnable.CallOpts)
}

// ISequencerInboxMetaData contains all meta data concerning the ISequencerInbox contract.
var ISequencerInboxMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keysetHash\",\"type\":\"bytes32\"}],\"name\":\"InvalidateKeyset\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"OwnerFunctionCalled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchSequenceNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"SequencerBatchData\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchSequenceNumber\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"afterAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"minTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"minBlockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxBlockNumber\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structISequencerInbox.TimeBounds\",\"name\":\"timeBounds\",\"type\":\"tuple\"},{\"indexed\":false,\"internalType\":\"enumISequencerInbox.BatchDataLocation\",\"name\":\"dataLocation\",\"type\":\"uint8\"}],\"name\":\"SequencerBatchDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keysetHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"SetValidKeyset\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DATA_AUTHENTICATED_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"HEADER_LENGTH\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2Batch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"}],\"name\":\"addSequencerL2BatchFromOrigin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batchCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"dasKeySetInfo\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"uint64[2]\",\"name\":\"l1BlockAndTime\",\"type\":\"uint64[2]\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"forceInclusion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"getKeysetCreationBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"inboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"bridge_\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"invalidateKeysetHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isBatchPoster\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isSequencer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"isValidKeysetHash\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxTimeVariation\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"removeDelayAfterFork\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isBatchPoster_\",\"type\":\"bool\"}],\"name\":\"setIsBatchPoster\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isSequencer_\",\"type\":\"bool\"}],\"name\":\"setIsSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"}],\"name\":\"setMaxTimeVariation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"setValidKeyset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalDelayedMessagesRead\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ISequencerInboxABI is the input ABI used to generate the binding from.
// Deprecated: Use ISequencerInboxMetaData.ABI instead.
var ISequencerInboxABI = ISequencerInboxMetaData.ABI

// ISequencerInbox is an auto generated Go binding around an Ethereum contract.
type ISequencerInbox struct {
	ISequencerInboxCaller     // Read-only binding to the contract
	ISequencerInboxTransactor // Write-only binding to the contract
	ISequencerInboxFilterer   // Log filterer for contract events
}

// ISequencerInboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type ISequencerInboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISequencerInboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ISequencerInboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISequencerInboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ISequencerInboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISequencerInboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ISequencerInboxSession struct {
	Contract     *ISequencerInbox  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ISequencerInboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ISequencerInboxCallerSession struct {
	Contract *ISequencerInboxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// ISequencerInboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ISequencerInboxTransactorSession struct {
	Contract     *ISequencerInboxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// ISequencerInboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type ISequencerInboxRaw struct {
	Contract *ISequencerInbox // Generic contract binding to access the raw methods on
}

// ISequencerInboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ISequencerInboxCallerRaw struct {
	Contract *ISequencerInboxCaller // Generic read-only contract binding to access the raw methods on
}

// ISequencerInboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ISequencerInboxTransactorRaw struct {
	Contract *ISequencerInboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewISequencerInbox creates a new instance of ISequencerInbox, bound to a specific deployed contract.
func NewISequencerInbox(address common.Address, backend bind.ContractBackend) (*ISequencerInbox, error) {
	contract, err := bindISequencerInbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ISequencerInbox{ISequencerInboxCaller: ISequencerInboxCaller{contract: contract}, ISequencerInboxTransactor: ISequencerInboxTransactor{contract: contract}, ISequencerInboxFilterer: ISequencerInboxFilterer{contract: contract}}, nil
}

// NewISequencerInboxCaller creates a new read-only instance of ISequencerInbox, bound to a specific deployed contract.
func NewISequencerInboxCaller(address common.Address, caller bind.ContractCaller) (*ISequencerInboxCaller, error) {
	contract, err := bindISequencerInbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ISequencerInboxCaller{contract: contract}, nil
}

// NewISequencerInboxTransactor creates a new write-only instance of ISequencerInbox, bound to a specific deployed contract.
func NewISequencerInboxTransactor(address common.Address, transactor bind.ContractTransactor) (*ISequencerInboxTransactor, error) {
	contract, err := bindISequencerInbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ISequencerInboxTransactor{contract: contract}, nil
}

// NewISequencerInboxFilterer creates a new log filterer instance of ISequencerInbox, bound to a specific deployed contract.
func NewISequencerInboxFilterer(address common.Address, filterer bind.ContractFilterer) (*ISequencerInboxFilterer, error) {
	contract, err := bindISequencerInbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ISequencerInboxFilterer{contract: contract}, nil
}

// bindISequencerInbox binds a generic wrapper to an already deployed contract.
func bindISequencerInbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ISequencerInboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ISequencerInbox *ISequencerInboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ISequencerInbox.Contract.ISequencerInboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ISequencerInbox *ISequencerInboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.ISequencerInboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ISequencerInbox *ISequencerInboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.ISequencerInboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ISequencerInbox *ISequencerInboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ISequencerInbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ISequencerInbox *ISequencerInboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ISequencerInbox *ISequencerInboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.contract.Transact(opts, method, params...)
}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_ISequencerInbox *ISequencerInboxCaller) DATAAUTHENTICATEDFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "DATA_AUTHENTICATED_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_ISequencerInbox *ISequencerInboxSession) DATAAUTHENTICATEDFLAG() ([1]byte, error) {
	return _ISequencerInbox.Contract.DATAAUTHENTICATEDFLAG(&_ISequencerInbox.CallOpts)
}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_ISequencerInbox *ISequencerInboxCallerSession) DATAAUTHENTICATEDFLAG() ([1]byte, error) {
	return _ISequencerInbox.Contract.DATAAUTHENTICATEDFLAG(&_ISequencerInbox.CallOpts)
}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_ISequencerInbox *ISequencerInboxCaller) HEADERLENGTH(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "HEADER_LENGTH")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_ISequencerInbox *ISequencerInboxSession) HEADERLENGTH() (*big.Int, error) {
	return _ISequencerInbox.Contract.HEADERLENGTH(&_ISequencerInbox.CallOpts)
}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_ISequencerInbox *ISequencerInboxCallerSession) HEADERLENGTH() (*big.Int, error) {
	return _ISequencerInbox.Contract.HEADERLENGTH(&_ISequencerInbox.CallOpts)
}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_ISequencerInbox *ISequencerInboxCaller) BatchCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "batchCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_ISequencerInbox *ISequencerInboxSession) BatchCount() (*big.Int, error) {
	return _ISequencerInbox.Contract.BatchCount(&_ISequencerInbox.CallOpts)
}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_ISequencerInbox *ISequencerInboxCallerSession) BatchCount() (*big.Int, error) {
	return _ISequencerInbox.Contract.BatchCount(&_ISequencerInbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_ISequencerInbox *ISequencerInboxCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_ISequencerInbox *ISequencerInboxSession) Bridge() (common.Address, error) {
	return _ISequencerInbox.Contract.Bridge(&_ISequencerInbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_ISequencerInbox *ISequencerInboxCallerSession) Bridge() (common.Address, error) {
	return _ISequencerInbox.Contract.Bridge(&_ISequencerInbox.CallOpts)
}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool, uint64)
func (_ISequencerInbox *ISequencerInboxCaller) DasKeySetInfo(opts *bind.CallOpts, arg0 [32]byte) (bool, uint64, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "dasKeySetInfo", arg0)

	if err != nil {
		return *new(bool), *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return out0, out1, err

}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool, uint64)
func (_ISequencerInbox *ISequencerInboxSession) DasKeySetInfo(arg0 [32]byte) (bool, uint64, error) {
	return _ISequencerInbox.Contract.DasKeySetInfo(&_ISequencerInbox.CallOpts, arg0)
}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool, uint64)
func (_ISequencerInbox *ISequencerInboxCallerSession) DasKeySetInfo(arg0 [32]byte) (bool, uint64, error) {
	return _ISequencerInbox.Contract.DasKeySetInfo(&_ISequencerInbox.CallOpts, arg0)
}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_ISequencerInbox *ISequencerInboxCaller) GetKeysetCreationBlock(opts *bind.CallOpts, ksHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "getKeysetCreationBlock", ksHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_ISequencerInbox *ISequencerInboxSession) GetKeysetCreationBlock(ksHash [32]byte) (*big.Int, error) {
	return _ISequencerInbox.Contract.GetKeysetCreationBlock(&_ISequencerInbox.CallOpts, ksHash)
}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_ISequencerInbox *ISequencerInboxCallerSession) GetKeysetCreationBlock(ksHash [32]byte) (*big.Int, error) {
	return _ISequencerInbox.Contract.GetKeysetCreationBlock(&_ISequencerInbox.CallOpts, ksHash)
}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_ISequencerInbox *ISequencerInboxCaller) InboxAccs(opts *bind.CallOpts, index *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "inboxAccs", index)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_ISequencerInbox *ISequencerInboxSession) InboxAccs(index *big.Int) ([32]byte, error) {
	return _ISequencerInbox.Contract.InboxAccs(&_ISequencerInbox.CallOpts, index)
}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_ISequencerInbox *ISequencerInboxCallerSession) InboxAccs(index *big.Int) ([32]byte, error) {
	return _ISequencerInbox.Contract.InboxAccs(&_ISequencerInbox.CallOpts, index)
}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_ISequencerInbox *ISequencerInboxCaller) IsBatchPoster(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "isBatchPoster", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_ISequencerInbox *ISequencerInboxSession) IsBatchPoster(arg0 common.Address) (bool, error) {
	return _ISequencerInbox.Contract.IsBatchPoster(&_ISequencerInbox.CallOpts, arg0)
}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_ISequencerInbox *ISequencerInboxCallerSession) IsBatchPoster(arg0 common.Address) (bool, error) {
	return _ISequencerInbox.Contract.IsBatchPoster(&_ISequencerInbox.CallOpts, arg0)
}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_ISequencerInbox *ISequencerInboxCaller) IsSequencer(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "isSequencer", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_ISequencerInbox *ISequencerInboxSession) IsSequencer(arg0 common.Address) (bool, error) {
	return _ISequencerInbox.Contract.IsSequencer(&_ISequencerInbox.CallOpts, arg0)
}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_ISequencerInbox *ISequencerInboxCallerSession) IsSequencer(arg0 common.Address) (bool, error) {
	return _ISequencerInbox.Contract.IsSequencer(&_ISequencerInbox.CallOpts, arg0)
}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_ISequencerInbox *ISequencerInboxCaller) IsValidKeysetHash(opts *bind.CallOpts, ksHash [32]byte) (bool, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "isValidKeysetHash", ksHash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_ISequencerInbox *ISequencerInboxSession) IsValidKeysetHash(ksHash [32]byte) (bool, error) {
	return _ISequencerInbox.Contract.IsValidKeysetHash(&_ISequencerInbox.CallOpts, ksHash)
}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_ISequencerInbox *ISequencerInboxCallerSession) IsValidKeysetHash(ksHash [32]byte) (bool, error) {
	return _ISequencerInbox.Contract.IsValidKeysetHash(&_ISequencerInbox.CallOpts, ksHash)
}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256, uint256, uint256, uint256)
func (_ISequencerInbox *ISequencerInboxCaller) MaxTimeVariation(opts *bind.CallOpts) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "maxTimeVariation")

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
func (_ISequencerInbox *ISequencerInboxSession) MaxTimeVariation() (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _ISequencerInbox.Contract.MaxTimeVariation(&_ISequencerInbox.CallOpts)
}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256, uint256, uint256, uint256)
func (_ISequencerInbox *ISequencerInboxCallerSession) MaxTimeVariation() (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	return _ISequencerInbox.Contract.MaxTimeVariation(&_ISequencerInbox.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_ISequencerInbox *ISequencerInboxCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_ISequencerInbox *ISequencerInboxSession) Rollup() (common.Address, error) {
	return _ISequencerInbox.Contract.Rollup(&_ISequencerInbox.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_ISequencerInbox *ISequencerInboxCallerSession) Rollup() (common.Address, error) {
	return _ISequencerInbox.Contract.Rollup(&_ISequencerInbox.CallOpts)
}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_ISequencerInbox *ISequencerInboxCaller) TotalDelayedMessagesRead(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ISequencerInbox.contract.Call(opts, &out, "totalDelayedMessagesRead")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_ISequencerInbox *ISequencerInboxSession) TotalDelayedMessagesRead() (*big.Int, error) {
	return _ISequencerInbox.Contract.TotalDelayedMessagesRead(&_ISequencerInbox.CallOpts)
}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_ISequencerInbox *ISequencerInboxCallerSession) TotalDelayedMessagesRead() (*big.Int, error) {
	return _ISequencerInbox.Contract.TotalDelayedMessagesRead(&_ISequencerInbox.CallOpts)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_ISequencerInbox *ISequencerInboxTransactor) AddSequencerL2Batch(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "addSequencerL2Batch", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_ISequencerInbox *ISequencerInboxSession) AddSequencerL2Batch(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.AddSequencerL2Batch(&_ISequencerInbox.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) AddSequencerL2Batch(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.AddSequencerL2Batch(&_ISequencerInbox.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromOrigin is a paid mutator transaction binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder) returns()
func (_ISequencerInbox *ISequencerInboxTransactor) AddSequencerL2BatchFromOrigin(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "addSequencerL2BatchFromOrigin", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder)
}

// AddSequencerL2BatchFromOrigin is a paid mutator transaction binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder) returns()
func (_ISequencerInbox *ISequencerInboxSession) AddSequencerL2BatchFromOrigin(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.AddSequencerL2BatchFromOrigin(&_ISequencerInbox.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder)
}

// AddSequencerL2BatchFromOrigin is a paid mutator transaction binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder) returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) AddSequencerL2BatchFromOrigin(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.AddSequencerL2BatchFromOrigin(&_ISequencerInbox.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_ISequencerInbox *ISequencerInboxTransactor) ForceInclusion(opts *bind.TransactOpts, _totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "forceInclusion", _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_ISequencerInbox *ISequencerInboxSession) ForceInclusion(_totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.ForceInclusion(&_ISequencerInbox.TransactOpts, _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) ForceInclusion(_totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.ForceInclusion(&_ISequencerInbox.TransactOpts, _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// Initialize is a paid mutator transaction binding the contract method 0x1f7a92b2.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_ISequencerInbox *ISequencerInboxTransactor) Initialize(opts *bind.TransactOpts, bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "initialize", bridge_, maxTimeVariation_)
}

// Initialize is a paid mutator transaction binding the contract method 0x1f7a92b2.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_ISequencerInbox *ISequencerInboxSession) Initialize(bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.Initialize(&_ISequencerInbox.TransactOpts, bridge_, maxTimeVariation_)
}

// Initialize is a paid mutator transaction binding the contract method 0x1f7a92b2.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) Initialize(bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.Initialize(&_ISequencerInbox.TransactOpts, bridge_, maxTimeVariation_)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_ISequencerInbox *ISequencerInboxTransactor) InvalidateKeysetHash(opts *bind.TransactOpts, ksHash [32]byte) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "invalidateKeysetHash", ksHash)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_ISequencerInbox *ISequencerInboxSession) InvalidateKeysetHash(ksHash [32]byte) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.InvalidateKeysetHash(&_ISequencerInbox.TransactOpts, ksHash)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) InvalidateKeysetHash(ksHash [32]byte) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.InvalidateKeysetHash(&_ISequencerInbox.TransactOpts, ksHash)
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_ISequencerInbox *ISequencerInboxTransactor) RemoveDelayAfterFork(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "removeDelayAfterFork")
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_ISequencerInbox *ISequencerInboxSession) RemoveDelayAfterFork() (*types.Transaction, error) {
	return _ISequencerInbox.Contract.RemoveDelayAfterFork(&_ISequencerInbox.TransactOpts)
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) RemoveDelayAfterFork() (*types.Transaction, error) {
	return _ISequencerInbox.Contract.RemoveDelayAfterFork(&_ISequencerInbox.TransactOpts)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_ISequencerInbox *ISequencerInboxTransactor) SetIsBatchPoster(opts *bind.TransactOpts, addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "setIsBatchPoster", addr, isBatchPoster_)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_ISequencerInbox *ISequencerInboxSession) SetIsBatchPoster(addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.SetIsBatchPoster(&_ISequencerInbox.TransactOpts, addr, isBatchPoster_)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) SetIsBatchPoster(addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.SetIsBatchPoster(&_ISequencerInbox.TransactOpts, addr, isBatchPoster_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_ISequencerInbox *ISequencerInboxTransactor) SetIsSequencer(opts *bind.TransactOpts, addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "setIsSequencer", addr, isSequencer_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_ISequencerInbox *ISequencerInboxSession) SetIsSequencer(addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.SetIsSequencer(&_ISequencerInbox.TransactOpts, addr, isSequencer_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) SetIsSequencer(addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.SetIsSequencer(&_ISequencerInbox.TransactOpts, addr, isSequencer_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_ISequencerInbox *ISequencerInboxTransactor) SetMaxTimeVariation(opts *bind.TransactOpts, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "setMaxTimeVariation", maxTimeVariation_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_ISequencerInbox *ISequencerInboxSession) SetMaxTimeVariation(maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.SetMaxTimeVariation(&_ISequencerInbox.TransactOpts, maxTimeVariation_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) SetMaxTimeVariation(maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.SetMaxTimeVariation(&_ISequencerInbox.TransactOpts, maxTimeVariation_)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_ISequencerInbox *ISequencerInboxTransactor) SetValidKeyset(opts *bind.TransactOpts, keysetBytes []byte) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "setValidKeyset", keysetBytes)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_ISequencerInbox *ISequencerInboxSession) SetValidKeyset(keysetBytes []byte) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.SetValidKeyset(&_ISequencerInbox.TransactOpts, keysetBytes)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) SetValidKeyset(keysetBytes []byte) (*types.Transaction, error) {
	return _ISequencerInbox.Contract.SetValidKeyset(&_ISequencerInbox.TransactOpts, keysetBytes)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_ISequencerInbox *ISequencerInboxTransactor) UpdateRollupAddress(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISequencerInbox.contract.Transact(opts, "updateRollupAddress")
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_ISequencerInbox *ISequencerInboxSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _ISequencerInbox.Contract.UpdateRollupAddress(&_ISequencerInbox.TransactOpts)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_ISequencerInbox *ISequencerInboxTransactorSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _ISequencerInbox.Contract.UpdateRollupAddress(&_ISequencerInbox.TransactOpts)
}

// ISequencerInboxInboxMessageDeliveredIterator is returned from FilterInboxMessageDelivered and is used to iterate over the raw logs and unpacked data for InboxMessageDelivered events raised by the ISequencerInbox contract.
type ISequencerInboxInboxMessageDeliveredIterator struct {
	Event *ISequencerInboxInboxMessageDelivered // Event containing the contract specifics and raw log

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
func (it *ISequencerInboxInboxMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISequencerInboxInboxMessageDelivered)
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
		it.Event = new(ISequencerInboxInboxMessageDelivered)
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
func (it *ISequencerInboxInboxMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISequencerInboxInboxMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISequencerInboxInboxMessageDelivered represents a InboxMessageDelivered event raised by the ISequencerInbox contract.
type ISequencerInboxInboxMessageDelivered struct {
	MessageNum *big.Int
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDelivered is a free log retrieval operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_ISequencerInbox *ISequencerInboxFilterer) FilterInboxMessageDelivered(opts *bind.FilterOpts, messageNum []*big.Int) (*ISequencerInboxInboxMessageDeliveredIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _ISequencerInbox.contract.FilterLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &ISequencerInboxInboxMessageDeliveredIterator{contract: _ISequencerInbox.contract, event: "InboxMessageDelivered", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDelivered is a free log subscription operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_ISequencerInbox *ISequencerInboxFilterer) WatchInboxMessageDelivered(opts *bind.WatchOpts, sink chan<- *ISequencerInboxInboxMessageDelivered, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _ISequencerInbox.contract.WatchLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISequencerInboxInboxMessageDelivered)
				if err := _ISequencerInbox.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
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
func (_ISequencerInbox *ISequencerInboxFilterer) ParseInboxMessageDelivered(log types.Log) (*ISequencerInboxInboxMessageDelivered, error) {
	event := new(ISequencerInboxInboxMessageDelivered)
	if err := _ISequencerInbox.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISequencerInboxInboxMessageDeliveredFromOriginIterator is returned from FilterInboxMessageDeliveredFromOrigin and is used to iterate over the raw logs and unpacked data for InboxMessageDeliveredFromOrigin events raised by the ISequencerInbox contract.
type ISequencerInboxInboxMessageDeliveredFromOriginIterator struct {
	Event *ISequencerInboxInboxMessageDeliveredFromOrigin // Event containing the contract specifics and raw log

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
func (it *ISequencerInboxInboxMessageDeliveredFromOriginIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISequencerInboxInboxMessageDeliveredFromOrigin)
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
		it.Event = new(ISequencerInboxInboxMessageDeliveredFromOrigin)
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
func (it *ISequencerInboxInboxMessageDeliveredFromOriginIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISequencerInboxInboxMessageDeliveredFromOriginIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISequencerInboxInboxMessageDeliveredFromOrigin represents a InboxMessageDeliveredFromOrigin event raised by the ISequencerInbox contract.
type ISequencerInboxInboxMessageDeliveredFromOrigin struct {
	MessageNum *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDeliveredFromOrigin is a free log retrieval operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_ISequencerInbox *ISequencerInboxFilterer) FilterInboxMessageDeliveredFromOrigin(opts *bind.FilterOpts, messageNum []*big.Int) (*ISequencerInboxInboxMessageDeliveredFromOriginIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _ISequencerInbox.contract.FilterLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &ISequencerInboxInboxMessageDeliveredFromOriginIterator{contract: _ISequencerInbox.contract, event: "InboxMessageDeliveredFromOrigin", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDeliveredFromOrigin is a free log subscription operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_ISequencerInbox *ISequencerInboxFilterer) WatchInboxMessageDeliveredFromOrigin(opts *bind.WatchOpts, sink chan<- *ISequencerInboxInboxMessageDeliveredFromOrigin, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _ISequencerInbox.contract.WatchLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISequencerInboxInboxMessageDeliveredFromOrigin)
				if err := _ISequencerInbox.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
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
func (_ISequencerInbox *ISequencerInboxFilterer) ParseInboxMessageDeliveredFromOrigin(log types.Log) (*ISequencerInboxInboxMessageDeliveredFromOrigin, error) {
	event := new(ISequencerInboxInboxMessageDeliveredFromOrigin)
	if err := _ISequencerInbox.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISequencerInboxInvalidateKeysetIterator is returned from FilterInvalidateKeyset and is used to iterate over the raw logs and unpacked data for InvalidateKeyset events raised by the ISequencerInbox contract.
type ISequencerInboxInvalidateKeysetIterator struct {
	Event *ISequencerInboxInvalidateKeyset // Event containing the contract specifics and raw log

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
func (it *ISequencerInboxInvalidateKeysetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISequencerInboxInvalidateKeyset)
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
		it.Event = new(ISequencerInboxInvalidateKeyset)
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
func (it *ISequencerInboxInvalidateKeysetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISequencerInboxInvalidateKeysetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISequencerInboxInvalidateKeyset represents a InvalidateKeyset event raised by the ISequencerInbox contract.
type ISequencerInboxInvalidateKeyset struct {
	KeysetHash [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInvalidateKeyset is a free log retrieval operation binding the contract event 0x5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a.
//
// Solidity: event InvalidateKeyset(bytes32 indexed keysetHash)
func (_ISequencerInbox *ISequencerInboxFilterer) FilterInvalidateKeyset(opts *bind.FilterOpts, keysetHash [][32]byte) (*ISequencerInboxInvalidateKeysetIterator, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _ISequencerInbox.contract.FilterLogs(opts, "InvalidateKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return &ISequencerInboxInvalidateKeysetIterator{contract: _ISequencerInbox.contract, event: "InvalidateKeyset", logs: logs, sub: sub}, nil
}

// WatchInvalidateKeyset is a free log subscription operation binding the contract event 0x5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a.
//
// Solidity: event InvalidateKeyset(bytes32 indexed keysetHash)
func (_ISequencerInbox *ISequencerInboxFilterer) WatchInvalidateKeyset(opts *bind.WatchOpts, sink chan<- *ISequencerInboxInvalidateKeyset, keysetHash [][32]byte) (event.Subscription, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _ISequencerInbox.contract.WatchLogs(opts, "InvalidateKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISequencerInboxInvalidateKeyset)
				if err := _ISequencerInbox.contract.UnpackLog(event, "InvalidateKeyset", log); err != nil {
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
func (_ISequencerInbox *ISequencerInboxFilterer) ParseInvalidateKeyset(log types.Log) (*ISequencerInboxInvalidateKeyset, error) {
	event := new(ISequencerInboxInvalidateKeyset)
	if err := _ISequencerInbox.contract.UnpackLog(event, "InvalidateKeyset", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISequencerInboxOwnerFunctionCalledIterator is returned from FilterOwnerFunctionCalled and is used to iterate over the raw logs and unpacked data for OwnerFunctionCalled events raised by the ISequencerInbox contract.
type ISequencerInboxOwnerFunctionCalledIterator struct {
	Event *ISequencerInboxOwnerFunctionCalled // Event containing the contract specifics and raw log

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
func (it *ISequencerInboxOwnerFunctionCalledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISequencerInboxOwnerFunctionCalled)
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
		it.Event = new(ISequencerInboxOwnerFunctionCalled)
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
func (it *ISequencerInboxOwnerFunctionCalledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISequencerInboxOwnerFunctionCalledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISequencerInboxOwnerFunctionCalled represents a OwnerFunctionCalled event raised by the ISequencerInbox contract.
type ISequencerInboxOwnerFunctionCalled struct {
	Id  *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterOwnerFunctionCalled is a free log retrieval operation binding the contract event 0xea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e.
//
// Solidity: event OwnerFunctionCalled(uint256 indexed id)
func (_ISequencerInbox *ISequencerInboxFilterer) FilterOwnerFunctionCalled(opts *bind.FilterOpts, id []*big.Int) (*ISequencerInboxOwnerFunctionCalledIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _ISequencerInbox.contract.FilterLogs(opts, "OwnerFunctionCalled", idRule)
	if err != nil {
		return nil, err
	}
	return &ISequencerInboxOwnerFunctionCalledIterator{contract: _ISequencerInbox.contract, event: "OwnerFunctionCalled", logs: logs, sub: sub}, nil
}

// WatchOwnerFunctionCalled is a free log subscription operation binding the contract event 0xea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e.
//
// Solidity: event OwnerFunctionCalled(uint256 indexed id)
func (_ISequencerInbox *ISequencerInboxFilterer) WatchOwnerFunctionCalled(opts *bind.WatchOpts, sink chan<- *ISequencerInboxOwnerFunctionCalled, id []*big.Int) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _ISequencerInbox.contract.WatchLogs(opts, "OwnerFunctionCalled", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISequencerInboxOwnerFunctionCalled)
				if err := _ISequencerInbox.contract.UnpackLog(event, "OwnerFunctionCalled", log); err != nil {
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
func (_ISequencerInbox *ISequencerInboxFilterer) ParseOwnerFunctionCalled(log types.Log) (*ISequencerInboxOwnerFunctionCalled, error) {
	event := new(ISequencerInboxOwnerFunctionCalled)
	if err := _ISequencerInbox.contract.UnpackLog(event, "OwnerFunctionCalled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISequencerInboxSequencerBatchDataIterator is returned from FilterSequencerBatchData and is used to iterate over the raw logs and unpacked data for SequencerBatchData events raised by the ISequencerInbox contract.
type ISequencerInboxSequencerBatchDataIterator struct {
	Event *ISequencerInboxSequencerBatchData // Event containing the contract specifics and raw log

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
func (it *ISequencerInboxSequencerBatchDataIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISequencerInboxSequencerBatchData)
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
		it.Event = new(ISequencerInboxSequencerBatchData)
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
func (it *ISequencerInboxSequencerBatchDataIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISequencerInboxSequencerBatchDataIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISequencerInboxSequencerBatchData represents a SequencerBatchData event raised by the ISequencerInbox contract.
type ISequencerInboxSequencerBatchData struct {
	BatchSequenceNumber *big.Int
	Data                []byte
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterSequencerBatchData is a free log retrieval operation binding the contract event 0xfe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c20.
//
// Solidity: event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data)
func (_ISequencerInbox *ISequencerInboxFilterer) FilterSequencerBatchData(opts *bind.FilterOpts, batchSequenceNumber []*big.Int) (*ISequencerInboxSequencerBatchDataIterator, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}

	logs, sub, err := _ISequencerInbox.contract.FilterLogs(opts, "SequencerBatchData", batchSequenceNumberRule)
	if err != nil {
		return nil, err
	}
	return &ISequencerInboxSequencerBatchDataIterator{contract: _ISequencerInbox.contract, event: "SequencerBatchData", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchData is a free log subscription operation binding the contract event 0xfe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c20.
//
// Solidity: event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data)
func (_ISequencerInbox *ISequencerInboxFilterer) WatchSequencerBatchData(opts *bind.WatchOpts, sink chan<- *ISequencerInboxSequencerBatchData, batchSequenceNumber []*big.Int) (event.Subscription, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}

	logs, sub, err := _ISequencerInbox.contract.WatchLogs(opts, "SequencerBatchData", batchSequenceNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISequencerInboxSequencerBatchData)
				if err := _ISequencerInbox.contract.UnpackLog(event, "SequencerBatchData", log); err != nil {
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
func (_ISequencerInbox *ISequencerInboxFilterer) ParseSequencerBatchData(log types.Log) (*ISequencerInboxSequencerBatchData, error) {
	event := new(ISequencerInboxSequencerBatchData)
	if err := _ISequencerInbox.contract.UnpackLog(event, "SequencerBatchData", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISequencerInboxSequencerBatchDeliveredIterator is returned from FilterSequencerBatchDelivered and is used to iterate over the raw logs and unpacked data for SequencerBatchDelivered events raised by the ISequencerInbox contract.
type ISequencerInboxSequencerBatchDeliveredIterator struct {
	Event *ISequencerInboxSequencerBatchDelivered // Event containing the contract specifics and raw log

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
func (it *ISequencerInboxSequencerBatchDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISequencerInboxSequencerBatchDelivered)
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
		it.Event = new(ISequencerInboxSequencerBatchDelivered)
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
func (it *ISequencerInboxSequencerBatchDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISequencerInboxSequencerBatchDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISequencerInboxSequencerBatchDelivered represents a SequencerBatchDelivered event raised by the ISequencerInbox contract.
type ISequencerInboxSequencerBatchDelivered struct {
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
func (_ISequencerInbox *ISequencerInboxFilterer) FilterSequencerBatchDelivered(opts *bind.FilterOpts, batchSequenceNumber []*big.Int, beforeAcc [][32]byte, afterAcc [][32]byte) (*ISequencerInboxSequencerBatchDeliveredIterator, error) {

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

	logs, sub, err := _ISequencerInbox.contract.FilterLogs(opts, "SequencerBatchDelivered", batchSequenceNumberRule, beforeAccRule, afterAccRule)
	if err != nil {
		return nil, err
	}
	return &ISequencerInboxSequencerBatchDeliveredIterator{contract: _ISequencerInbox.contract, event: "SequencerBatchDelivered", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchDelivered is a free log subscription operation binding the contract event 0x7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7.
//
// Solidity: event SequencerBatchDelivered(uint256 indexed batchSequenceNumber, bytes32 indexed beforeAcc, bytes32 indexed afterAcc, bytes32 delayedAcc, uint256 afterDelayedMessagesRead, (uint64,uint64,uint64,uint64) timeBounds, uint8 dataLocation)
func (_ISequencerInbox *ISequencerInboxFilterer) WatchSequencerBatchDelivered(opts *bind.WatchOpts, sink chan<- *ISequencerInboxSequencerBatchDelivered, batchSequenceNumber []*big.Int, beforeAcc [][32]byte, afterAcc [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _ISequencerInbox.contract.WatchLogs(opts, "SequencerBatchDelivered", batchSequenceNumberRule, beforeAccRule, afterAccRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISequencerInboxSequencerBatchDelivered)
				if err := _ISequencerInbox.contract.UnpackLog(event, "SequencerBatchDelivered", log); err != nil {
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
func (_ISequencerInbox *ISequencerInboxFilterer) ParseSequencerBatchDelivered(log types.Log) (*ISequencerInboxSequencerBatchDelivered, error) {
	event := new(ISequencerInboxSequencerBatchDelivered)
	if err := _ISequencerInbox.contract.UnpackLog(event, "SequencerBatchDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ISequencerInboxSetValidKeysetIterator is returned from FilterSetValidKeyset and is used to iterate over the raw logs and unpacked data for SetValidKeyset events raised by the ISequencerInbox contract.
type ISequencerInboxSetValidKeysetIterator struct {
	Event *ISequencerInboxSetValidKeyset // Event containing the contract specifics and raw log

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
func (it *ISequencerInboxSetValidKeysetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ISequencerInboxSetValidKeyset)
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
		it.Event = new(ISequencerInboxSetValidKeyset)
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
func (it *ISequencerInboxSetValidKeysetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ISequencerInboxSetValidKeysetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ISequencerInboxSetValidKeyset represents a SetValidKeyset event raised by the ISequencerInbox contract.
type ISequencerInboxSetValidKeyset struct {
	KeysetHash  [32]byte
	KeysetBytes []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSetValidKeyset is a free log retrieval operation binding the contract event 0xabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722.
//
// Solidity: event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes)
func (_ISequencerInbox *ISequencerInboxFilterer) FilterSetValidKeyset(opts *bind.FilterOpts, keysetHash [][32]byte) (*ISequencerInboxSetValidKeysetIterator, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _ISequencerInbox.contract.FilterLogs(opts, "SetValidKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return &ISequencerInboxSetValidKeysetIterator{contract: _ISequencerInbox.contract, event: "SetValidKeyset", logs: logs, sub: sub}, nil
}

// WatchSetValidKeyset is a free log subscription operation binding the contract event 0xabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722.
//
// Solidity: event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes)
func (_ISequencerInbox *ISequencerInboxFilterer) WatchSetValidKeyset(opts *bind.WatchOpts, sink chan<- *ISequencerInboxSetValidKeyset, keysetHash [][32]byte) (event.Subscription, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _ISequencerInbox.contract.WatchLogs(opts, "SetValidKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ISequencerInboxSetValidKeyset)
				if err := _ISequencerInbox.contract.UnpackLog(event, "SetValidKeyset", log); err != nil {
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
func (_ISequencerInbox *ISequencerInboxFilterer) ParseSetValidKeyset(log types.Log) (*ISequencerInboxSetValidKeyset, error) {
	event := new(ISequencerInboxSetValidKeyset)
	if err := _ISequencerInbox.contract.UnpackLog(event, "SetValidKeyset", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InboxMetaData contains all meta data concerning the Inbox contract.
var InboxMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"dataLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxDataLength\",\"type\":\"uint256\"}],\"name\":\"DataTooLarge\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"GasLimitTooLarge\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"expected\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"actual\",\"type\":\"uint256\"}],\"name\":\"InsufficientSubmissionCost\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"expected\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"actual\",\"type\":\"uint256\"}],\"name\":\"InsufficientValue\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"L1Forked\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"origin\",\"type\":\"address\"}],\"name\":\"NotAllowedOrigin\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotForked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotOrigin\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotRollupOrOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2CallValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deposit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxSubmissionCost\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"excessFeeRefundAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"callValueRefundAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"RetryableData\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"val\",\"type\":\"bool\"}],\"name\":\"AllowListAddressSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"isEnabled\",\"type\":\"bool\"}],\"name\":\"AllowListEnabledUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"allowListEnabled\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"dataLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"}],\"name\":\"calculateRetryableSubmissionFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2CallValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxSubmissionCost\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"excessFeeRefundAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"callValueRefundAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"createRetryableTicket\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2CallValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxSubmissionCost\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"excessFeeRefundAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"callValueRefundAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"createRetryableTicketNoRefundAliasRewrite\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"depositEth\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"depositEth\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"_bridge\",\"type\":\"address\"},{\"internalType\":\"contractISequencerInbox\",\"name\":\"_sequencerInbox\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isAllowed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"postUpgradeInit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendContractTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedContractTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedUnsignedTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendL1FundedUnsignedTransactionToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"messageData\",\"type\":\"bytes\"}],\"name\":\"sendL2Message\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"messageData\",\"type\":\"bytes\"}],\"name\":\"sendL2MessageFromOrigin\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendUnsignedTransaction\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendUnsignedTransactionToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"withdrawTo\",\"type\":\"address\"}],\"name\":\"sendWithdrawEthToFork\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencerInbox\",\"outputs\":[{\"internalType\":\"contractISequencerInbox\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"user\",\"type\":\"address[]\"},{\"internalType\":\"bool[]\",\"name\":\"val\",\"type\":\"bool[]\"}],\"name\":\"setAllowList\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_allowListEnabled\",\"type\":\"bool\"}],\"name\":\"setAllowListEnabled\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2CallValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxSubmissionCost\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"excessFeeRefundAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"callValueRefundAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxFeePerGas\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"unsafeCreateRetryableTicket\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x60c0604052306080524660a05234801561001857600080fd5b5060805160a05161291d610061600039600081816104fd015281816107140152818161117c01528181611516015261199d015260008181610b0a01526115b8015261291d6000f3fe6080604052600436106101a05760003560e01c80636e6e8a6a116100e1578063babcc5391161008a578063e6bd12cf11610064578063e6bd12cf14610417578063e78cea921461042a578063ee35f32714610462578063efeadb6d1461048257600080fd5b8063babcc539146103a7578063c474d2c5146103d7578063e3de72a5146103f757600080fd5b80638a631aa6116100bb5780638a631aa614610347578063a66b327d14610367578063b75436bb1461038757600080fd5b80636e6e8a6a146102ff57806370665f14146103125780638456cb591461033257600080fd5b8063439370b11161014e5780635c975abb116101285780635c975abb146102ae5780635e916758146102c6578063679b6ded146102d957806367ef3ab8146102ec57600080fd5b8063439370b114610266578063485cc9551461026e5780635075788b1461028e57600080fd5b80631fe927cf1161017f5780631fe927cf146101fe57806322bd5c1c1461021e5780633f4ba83a1461024f57600080fd5b8062f72382146101a55780630f4d14e9146101d85780631b871c8d146101eb575b600080fd5b3480156101b157600080fd5b506101c56101c0366004612073565b6104a2565b6040519081526020015b60405180910390f35b6101c56101e63660046120f0565b6105e9565b6101c56101f9366004612109565b61064c565b34801561020a57600080fd5b506101c56102193660046121ae565b6106be565b34801561022a57600080fd5b5060665461023f90600160a01b900460ff1681565b60405190151581526020016101cf565b34801561025b57600080fd5b5061026461082f565b005b6101c5610964565b34801561027a57600080fd5b506102646102893660046121f0565b610a31565b34801561029a57600080fd5b506101c56102a9366004612073565b610c5a565b3480156102ba57600080fd5b5060335460ff1661023f565b6101c56102d4366004612229565b610d05565b6101c56102e7366004612109565b610db8565b6101c56102fa366004612293565b610eee565b6101c561030d366004612109565b610fa4565b34801561031e57600080fd5b506101c561032d366004612306565b611126565b34801561033e57600080fd5b506102646112ad565b34801561035357600080fd5b506101c5610362366004612353565b6113df565b34801561037357600080fd5b506101c56103823660046123a8565b611488565b34801561039357600080fd5b506101c56103a23660046121ae565b6114c0565b3480156103b357600080fd5b5061023f6103c23660046123ca565b60676020526000908152604090205460ff1681565b3480156103e357600080fd5b506102646103f23660046123ca565b6115ae565b34801561040357600080fd5b506102646104123660046124d4565b6116c2565b6101c5610425366004612293565b611947565b34801561043657600080fd5b5060655461044a906001600160a01b031681565b6040516001600160a01b0390911681526020016101cf565b34801561046e57600080fd5b5060665461044a906001600160a01b031681565b34801561048e57600080fd5b5061026461049d366004612596565b611a69565b60006104ac611c66565b606654600160a01b900460ff1680156104d557503260009081526067602052604090205460ff16155b156104fa57604051630f51ed7160e41b81523260048201526024015b60405180910390fd5b467f00000000000000000000000000000000000000000000000000000000000000000361053a57604051635180dd8360e11b815260040160405180910390fd5b33321461055a5760405163feb3d07160e01b815260040160405180910390fd5b67ffffffffffffffff8811156105835760405163107c527b60e01b815260040160405180910390fd5b6105dd600373111100000000000000000000000000000000111019330160008b8b8b8b6001600160a01b03168b8b8b6040516020016105c99897969594939291906125b1565b604051602081830303815290604052611cbb565b98975050505050505050565b60006105f3611c66565b606654600160a01b900460ff16801561061c57503260009081526067602052604090205460ff16155b1561063c57604051630f51ed7160e41b81523260048201526024016104f1565b610644610964565b90505b919050565b6000610656611c66565b606654600160a01b900460ff16801561067f57503260009081526067602052604090205460ff16155b1561069f57604051630f51ed7160e41b81523260048201526024016104f1565b6106b08a8a8a8a8a8a8a8a8a610fa4565b9a9950505050505050505050565b60006106c8611c66565b606654600160a01b900460ff1680156106f157503260009081526067602052604090205460ff16155b1561071157604051630f51ed7160e41b81523260048201526024016104f1565b467f00000000000000000000000000000000000000000000000000000000000000001461076a576040517fc6ea680300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b33321461078a5760405163feb3d07160e01b815260040160405180910390fd5b6201cccc8211156107d3576040517f4634691b000000000000000000000000000000000000000000000000000000008152600481018390526201cccc60248201526044016104f1565b60006107f860033386866040516107eb929190612613565b6040518091039020611d60565b60405190915081907fab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c90600090a290505b92915050565b6065546040805163cb23bcb560e01b815290516000926001600160a01b03169163cb23bcb59160048083019260209291908290030181865afa158015610879573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061089d9190612623565b9050336001600160a01b03821614610959576000816001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156108ef573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906109139190612623565b9050336001600160a01b0382161461095757604051630739600760e01b81523360048201526001600160a01b038084166024830152821660448201526064016104f1565b505b610961611e26565b50565b600061096e611c66565b606654600160a01b900460ff16801561099757503260009081526067602052604090205460ff16155b156109b757604051630f51ed7160e41b81523260048201526024016104f1565b33803b1515806109c75750323314155b156109e5575033731111000000000000000000000000000000001111015b6040517fffffffffffffffffffffffffffffffffffffffff000000000000000000000000606083901b166020820152346034820152610a2b90600c9033906054016105c9565b91505090565b600054610100900460ff1615808015610a515750600054600160ff909116105b80610a6b5750303b158015610a6b575060005460ff166001145b610add5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016104f1565b6000805460ff191660011790558015610b00576000805461ff0019166101001790555b6001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000163003610b9e5760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084016104f1565b606580546001600160a01b038086167fffffffffffffffffffffffff000000000000000000000000000000000000000090921691909117909155606680547fffffffffffffffffffffff00000000000000000000000000000000000000000016918416919091179055610c0f611e78565b8015610c55576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b505050565b6000610c64611c66565b606654600160a01b900460ff168015610c8d57503260009081526067602052604090205460ff16155b15610cad57604051630f51ed7160e41b81523260048201526024016104f1565b67ffffffffffffffff881115610cd65760405163107c527b60e01b815260040160405180910390fd5b6105dd60033360008b8b8b8b6001600160a01b03168b8b8b6040516020016105c99897969594939291906125b1565b6000610d0f611c66565b606654600160a01b900460ff168015610d3857503260009081526067602052604090205460ff16155b15610d5857604051630f51ed7160e41b81523260048201526024016104f1565b67ffffffffffffffff861115610d815760405163107c527b60e01b815260040160405180910390fd5b610dae60073360018989896001600160a01b0316348a8a6040516020016105c99796959493929190612640565b9695505050505050565b6000610dc2611c66565b606654600160a01b900460ff168015610deb57503260009081526067602052604090205460ff16155b15610e0b57604051630f51ed7160e41b81523260048201526024016104f1565b610e1584866126b1565b610e1f8a8a6126c8565b610e2991906126c8565b341015610e8a57610e3a84866126b1565b610e448a8a6126c8565b610e4e91906126c8565b6040517f7040b58c00000000000000000000000000000000000000000000000000000000815260048101919091523460248201526044016104f1565b6001600160a01b0387163b15610eb457731111000000000000000000000000000000001111870196505b6001600160a01b0386163b1561069f57731111000000000000000000000000000000001111860195506106b08a8a8a8a8a8a8a8a8a610fa4565b6000610ef8611c66565b606654600160a01b900460ff168015610f2157503260009081526067602052604090205460ff16155b15610f4157604051630f51ed7160e41b81523260048201526024016104f1565b67ffffffffffffffff871115610f6a5760405163107c527b60e01b815260040160405180910390fd5b610f9960073360008a8a8a8a6001600160a01b0316348b8b6040516020016105c99897969594939291906125b1565b979650505050505050565b6000610fae611c66565b606654600160a01b900460ff168015610fd757503260009081526067602052604090205460ff16155b15610ff757604051630f51ed7160e41b81523260048201526024016104f1565b84600114806110065750836001145b1561105357338a8a348b8b8b8b8b8b8b6040517f07c266e30000000000000000000000000000000000000000000000000000000081526004016104f19b9a999897969594939291906126db565b67ffffffffffffffff85111561107c5760405163107c527b60e01b815260040160405180910390fd5b60006110888348611488565b9050808910156110ce576040517ffadf238a00000000000000000000000000000000000000000000000000000000815260048101829052602481018a90526044016104f1565b6111176009338d6001600160a01b03168d348e8e6001600160a01b03168e6001600160a01b03168e8e8e8e90508f8f6040516020016105c99b9a99989796959493929190612761565b9b9a5050505050505050505050565b6000611130611c66565b606654600160a01b900460ff16801561115957503260009081526067602052604090205460ff16155b1561117957604051630f51ed7160e41b81523260048201526024016104f1565b467f0000000000000000000000000000000000000000000000000000000000000000036111b957604051635180dd8360e11b815260040160405180910390fd5b3332146111d95760405163feb3d07160e01b815260040160405180910390fd5b67ffffffffffffffff8611156112025760405163107c527b60e01b815260040160405180910390fd5b604080516001600160a01b0384166024808301919091528251808303909101815260449091018252602080820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f25e16063000000000000000000000000000000000000000000000000000000001790529151610dae92600392337311110000000000000000000000000000000011101901926105c9926000928d928d928d926064928e92016127df565b6065546040805163cb23bcb560e01b815290516000926001600160a01b03169163cb23bcb59160048083019260209291908290030181865afa1580156112f7573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061131b9190612623565b9050336001600160a01b038216146113d7576000816001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa15801561136d573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906113919190612623565b9050336001600160a01b038216146113d557604051630739600760e01b81523360048201526001600160a01b038084166024830152821660448201526064016104f1565b505b610961611efd565b60006113e9611c66565b606654600160a01b900460ff16801561141257503260009081526067602052604090205460ff16155b1561143257604051630f51ed7160e41b81523260048201526024016104f1565b67ffffffffffffffff87111561145b5760405163107c527b60e01b815260040160405180910390fd5b610f9960033360018a8a8a6001600160a01b03168a8a8a6040516020016105c99796959493929190612640565b600081156114965781611498565b485b6114a38460066126b1565b6114af906105786126c8565b6114b991906126b1565b9392505050565b60006114ca611c66565b606654600160a01b900460ff1680156114f357503260009081526067602052604090205460ff16155b1561151357604051630f51ed7160e41b81523260048201526024016104f1565b467f00000000000000000000000000000000000000000000000000000000000000001461156c576040517fc6ea680300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6114b960033385858080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250611cbb92505050565b6001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016300361164c5760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084016104f1565b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038054336001600160a01b03821614610c55576040517f23295f0e0000000000000000000000000000000000000000000000000000000081523360048201526001600160a01b03821660248201526044016104f1565b6065546040805163cb23bcb560e01b815290516000926001600160a01b03169163cb23bcb59160048083019260209291908290030181865afa15801561170c573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906117309190612623565b9050336001600160a01b038216146117ec576000816001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611782573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906117a69190612623565b9050336001600160a01b038216146117ea57604051630739600760e01b81523360048201526001600160a01b038084166024830152821660448201526064016104f1565b505b815183511461183d5760405162461bcd60e51b815260206004820152600d60248201527f494e56414c49445f494e5055540000000000000000000000000000000000000060448201526064016104f1565b60005b83518110156119415782818151811061185b5761185b61284d565b6020026020010151606760008684815181106118795761187961284d565b60200260200101516001600160a01b03166001600160a01b0316815260200190815260200160002060006101000a81548160ff0219169083151502179055508381815181106118ca576118ca61284d565b60200260200101516001600160a01b03167fd9739f45a01ce092c5cdb3d68f63d63d21676b1c6c0b4f9cbc6be4cf5449595a84838151811061190e5761190e61284d565b6020026020010151604051611927911515815260200190565b60405180910390a28061193981612863565b915050611840565b50505050565b6000611951611c66565b606654600160a01b900460ff16801561197a57503260009081526067602052604090205460ff16155b1561199a57604051630f51ed7160e41b81523260048201526024016104f1565b467f0000000000000000000000000000000000000000000000000000000000000000036119da57604051635180dd8360e11b815260040160405180910390fd5b3332146119fa5760405163feb3d07160e01b815260040160405180910390fd5b67ffffffffffffffff871115611a235760405163107c527b60e01b815260040160405180910390fd5b610f99600773111100000000000000000000000000000000111019330160008a8a8a8a6001600160a01b0316348b8b6040516020016105c99897969594939291906125b1565b6065546040805163cb23bcb560e01b815290516000926001600160a01b03169163cb23bcb59160048083019260209291908290030181865afa158015611ab3573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611ad79190612623565b9050336001600160a01b03821614611b93576000816001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611b29573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611b4d9190612623565b9050336001600160a01b03821614611b9157604051630739600760e01b81523360048201526001600160a01b038084166024830152821660448201526064016104f1565b505b606654600160a01b900460ff16151582151503611bf25760405162461bcd60e51b815260206004820152600b60248201527f414c52454144595f53455400000000000000000000000000000000000000000060448201526064016104f1565b60668054831515600160a01b027fffffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffff9091161790556040517f16435b45f7482047f839a6a19d291442627200f52cad2803c595150d0d440eb390611c5a90841515815260200190565b60405180910390a15050565b60335460ff1615611cb95760405162461bcd60e51b815260206004820152601060248201527f5061757361626c653a207061757365640000000000000000000000000000000060448201526064016104f1565b565b60006201cccc82511115611d0a5781516040517f4634691b00000000000000000000000000000000000000000000000000000000815260048101919091526201cccc60248201526044016104f1565b6000611d1e85858580519060200120611d60565b9050807fff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b84604051611d50919061289b565b60405180910390a2949350505050565b6065546000906001600160a01b0316638db5993b348673111100000000000000000000000000000000111187016040517fffffffff0000000000000000000000000000000000000000000000000000000060e086901b16815260ff90921660048301526001600160a01b031660248201526044810186905260640160206040518083038185885af1158015611df9573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190611e1e91906128ce565b949350505050565b611e2e611f3a565b6033805460ff191690557f5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa335b6040516001600160a01b03909116815260200160405180910390a1565b600054610100900460ff16611ef55760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016104f1565b611cb9611f8c565b611f05611c66565b6033805460ff191660011790557f62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258611e5b3390565b60335460ff16611cb95760405162461bcd60e51b815260206004820152601460248201527f5061757361626c653a206e6f742070617573656400000000000000000000000060448201526064016104f1565b600054610100900460ff166120095760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016104f1565b6033805460ff19169055565b6001600160a01b038116811461096157600080fd5b60008083601f84011261203c57600080fd5b50813567ffffffffffffffff81111561205457600080fd5b60208301915083602082850101111561206c57600080fd5b9250929050565b600080600080600080600060c0888a03121561208e57600080fd5b87359650602088013595506040880135945060608801356120ae81612015565b93506080880135925060a088013567ffffffffffffffff8111156120d157600080fd5b6120dd8a828b0161202a565b989b979a50959850939692959293505050565b60006020828403121561210257600080fd5b5035919050565b60008060008060008060008060006101008a8c03121561212857600080fd5b893561213381612015565b985060208a0135975060408a0135965060608a013561215181612015565b955060808a013561216181612015565b945060a08a0135935060c08a0135925060e08a013567ffffffffffffffff81111561218b57600080fd5b6121978c828d0161202a565b915080935050809150509295985092959850929598565b600080602083850312156121c157600080fd5b823567ffffffffffffffff8111156121d857600080fd5b6121e48582860161202a565b90969095509350505050565b6000806040838503121561220357600080fd5b823561220e81612015565b9150602083013561221e81612015565b809150509250929050565b60008060008060006080868803121561224157600080fd5b8535945060208601359350604086013561225a81612015565b9250606086013567ffffffffffffffff81111561227657600080fd5b6122828882890161202a565b969995985093965092949392505050565b60008060008060008060a087890312156122ac57600080fd5b86359550602087013594506040870135935060608701356122cc81612015565b9250608087013567ffffffffffffffff8111156122e857600080fd5b6122f489828a0161202a565b979a9699509497509295939492505050565b600080600080600060a0868803121561231e57600080fd5b85359450602086013593506040860135925060608601359150608086013561234581612015565b809150509295509295909350565b60008060008060008060a0878903121561236c57600080fd5b8635955060208701359450604087013561238581612015565b935060608701359250608087013567ffffffffffffffff8111156122e857600080fd5b600080604083850312156123bb57600080fd5b50508035926020909101359150565b6000602082840312156123dc57600080fd5b81356114b981612015565b634e487b7160e01b600052604160045260246000fd5b604051601f8201601f1916810167ffffffffffffffff81118282101715612426576124266123e7565b604052919050565b600067ffffffffffffffff821115612448576124486123e7565b5060051b60200190565b8035801515811461064757600080fd5b600082601f83011261247357600080fd5b813560206124886124838361242e565b6123fd565b82815260059290921b840181019181810190868411156124a757600080fd5b8286015b848110156124c9576124bc81612452565b83529183019183016124ab565b509695505050505050565b600080604083850312156124e757600080fd5b823567ffffffffffffffff808211156124ff57600080fd5b818501915085601f83011261251357600080fd5b813560206125236124838361242e565b82815260059290921b8401810191818101908984111561254257600080fd5b948201945b8386101561256957853561255a81612015565b82529482019490820190612547565b9650508601359250508082111561257f57600080fd5b5061258c85828601612462565b9150509250929050565b6000602082840312156125a857600080fd5b6114b982612452565b7fff000000000000000000000000000000000000000000000000000000000000008960f81b168152876001820152866021820152856041820152846061820152836081820152818360a18301376000910160a101908152979650505050505050565b8183823760009101908152919050565b60006020828403121561263557600080fd5b81516114b981612015565b7fff000000000000000000000000000000000000000000000000000000000000008860f81b16815286600182015285602182015284604182015283606182015281836081830137600091016081019081529695505050505050565b634e487b7160e01b600052601160045260246000fd5b80820281158282048414176108295761082961269b565b808201808211156108295761082961269b565b60006101406001600160a01b03808f168452808e1660208501528c60408501528b60608501528a6080850152808a1660a085015280891660c0850152508660e084015285610100840152806101208401528381840152506101608385828501376000838501820152601f909301601f19169091019091019b9a5050505050505050505050565b8b81528a60208201528960408201528860608201528760808201528660a08201528560c08201528460e08201528361010082015260006101208385828501376000929093019092019081529b9a5050505050505050505050565b60005b838110156127d65781810151838201526020016127be565b50506000910152565b7fff000000000000000000000000000000000000000000000000000000000000008860f81b1681528660018201528560218201528460418201528360618201528260818201526000825161283a8160a18501602087016127bb565b9190910160a10198975050505050505050565b634e487b7160e01b600052603260045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036128945761289461269b565b5060010190565b60208152600082518060208401526128ba8160408501602087016127bb565b601f01601f19169190910160400192915050565b6000602082840312156128e057600080fd5b505191905056fea2646970667358221220278a12ea135265c77dd913755a9dbe44f40aa604a88a095aa6ba49831b1bda3964736f6c63430008110033",
}

// InboxABI is the input ABI used to generate the binding from.
// Deprecated: Use InboxMetaData.ABI instead.
var InboxABI = InboxMetaData.ABI

// InboxBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use InboxMetaData.Bin instead.
var InboxBin = InboxMetaData.Bin

// DeployInbox deploys a new Ethereum contract, binding an instance of Inbox to it.
func DeployInbox(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Inbox, error) {
	parsed, err := InboxMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(InboxBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Inbox{InboxCaller: InboxCaller{contract: contract}, InboxTransactor: InboxTransactor{contract: contract}, InboxFilterer: InboxFilterer{contract: contract}}, nil
}

// Inbox is an auto generated Go binding around an Ethereum contract.
type Inbox struct {
	InboxCaller     // Read-only binding to the contract
	InboxTransactor // Write-only binding to the contract
	InboxFilterer   // Log filterer for contract events
}

// InboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type InboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InboxSession struct {
	Contract     *Inbox            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InboxCallerSession struct {
	Contract *InboxCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// InboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InboxTransactorSession struct {
	Contract     *InboxTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type InboxRaw struct {
	Contract *Inbox // Generic contract binding to access the raw methods on
}

// InboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InboxCallerRaw struct {
	Contract *InboxCaller // Generic read-only contract binding to access the raw methods on
}

// InboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InboxTransactorRaw struct {
	Contract *InboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInbox creates a new instance of Inbox, bound to a specific deployed contract.
func NewInbox(address common.Address, backend bind.ContractBackend) (*Inbox, error) {
	contract, err := bindInbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Inbox{InboxCaller: InboxCaller{contract: contract}, InboxTransactor: InboxTransactor{contract: contract}, InboxFilterer: InboxFilterer{contract: contract}}, nil
}

// NewInboxCaller creates a new read-only instance of Inbox, bound to a specific deployed contract.
func NewInboxCaller(address common.Address, caller bind.ContractCaller) (*InboxCaller, error) {
	contract, err := bindInbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InboxCaller{contract: contract}, nil
}

// NewInboxTransactor creates a new write-only instance of Inbox, bound to a specific deployed contract.
func NewInboxTransactor(address common.Address, transactor bind.ContractTransactor) (*InboxTransactor, error) {
	contract, err := bindInbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InboxTransactor{contract: contract}, nil
}

// NewInboxFilterer creates a new log filterer instance of Inbox, bound to a specific deployed contract.
func NewInboxFilterer(address common.Address, filterer bind.ContractFilterer) (*InboxFilterer, error) {
	contract, err := bindInbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InboxFilterer{contract: contract}, nil
}

// bindInbox binds a generic wrapper to an already deployed contract.
func bindInbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := InboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Inbox *InboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Inbox.Contract.InboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Inbox *InboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Inbox.Contract.InboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Inbox *InboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Inbox.Contract.InboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Inbox *InboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Inbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Inbox *InboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Inbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Inbox *InboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Inbox.Contract.contract.Transact(opts, method, params...)
}

// AllowListEnabled is a free data retrieval call binding the contract method 0x22bd5c1c.
//
// Solidity: function allowListEnabled() view returns(bool)
func (_Inbox *InboxCaller) AllowListEnabled(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Inbox.contract.Call(opts, &out, "allowListEnabled")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AllowListEnabled is a free data retrieval call binding the contract method 0x22bd5c1c.
//
// Solidity: function allowListEnabled() view returns(bool)
func (_Inbox *InboxSession) AllowListEnabled() (bool, error) {
	return _Inbox.Contract.AllowListEnabled(&_Inbox.CallOpts)
}

// AllowListEnabled is a free data retrieval call binding the contract method 0x22bd5c1c.
//
// Solidity: function allowListEnabled() view returns(bool)
func (_Inbox *InboxCallerSession) AllowListEnabled() (bool, error) {
	return _Inbox.Contract.AllowListEnabled(&_Inbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_Inbox *InboxCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Inbox.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_Inbox *InboxSession) Bridge() (common.Address, error) {
	return _Inbox.Contract.Bridge(&_Inbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_Inbox *InboxCallerSession) Bridge() (common.Address, error) {
	return _Inbox.Contract.Bridge(&_Inbox.CallOpts)
}

// CalculateRetryableSubmissionFee is a free data retrieval call binding the contract method 0xa66b327d.
//
// Solidity: function calculateRetryableSubmissionFee(uint256 dataLength, uint256 baseFee) view returns(uint256)
func (_Inbox *InboxCaller) CalculateRetryableSubmissionFee(opts *bind.CallOpts, dataLength *big.Int, baseFee *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Inbox.contract.Call(opts, &out, "calculateRetryableSubmissionFee", dataLength, baseFee)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CalculateRetryableSubmissionFee is a free data retrieval call binding the contract method 0xa66b327d.
//
// Solidity: function calculateRetryableSubmissionFee(uint256 dataLength, uint256 baseFee) view returns(uint256)
func (_Inbox *InboxSession) CalculateRetryableSubmissionFee(dataLength *big.Int, baseFee *big.Int) (*big.Int, error) {
	return _Inbox.Contract.CalculateRetryableSubmissionFee(&_Inbox.CallOpts, dataLength, baseFee)
}

// CalculateRetryableSubmissionFee is a free data retrieval call binding the contract method 0xa66b327d.
//
// Solidity: function calculateRetryableSubmissionFee(uint256 dataLength, uint256 baseFee) view returns(uint256)
func (_Inbox *InboxCallerSession) CalculateRetryableSubmissionFee(dataLength *big.Int, baseFee *big.Int) (*big.Int, error) {
	return _Inbox.Contract.CalculateRetryableSubmissionFee(&_Inbox.CallOpts, dataLength, baseFee)
}

// IsAllowed is a free data retrieval call binding the contract method 0xbabcc539.
//
// Solidity: function isAllowed(address ) view returns(bool)
func (_Inbox *InboxCaller) IsAllowed(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _Inbox.contract.Call(opts, &out, "isAllowed", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAllowed is a free data retrieval call binding the contract method 0xbabcc539.
//
// Solidity: function isAllowed(address ) view returns(bool)
func (_Inbox *InboxSession) IsAllowed(arg0 common.Address) (bool, error) {
	return _Inbox.Contract.IsAllowed(&_Inbox.CallOpts, arg0)
}

// IsAllowed is a free data retrieval call binding the contract method 0xbabcc539.
//
// Solidity: function isAllowed(address ) view returns(bool)
func (_Inbox *InboxCallerSession) IsAllowed(arg0 common.Address) (bool, error) {
	return _Inbox.Contract.IsAllowed(&_Inbox.CallOpts, arg0)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Inbox *InboxCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Inbox.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Inbox *InboxSession) Paused() (bool, error) {
	return _Inbox.Contract.Paused(&_Inbox.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Inbox *InboxCallerSession) Paused() (bool, error) {
	return _Inbox.Contract.Paused(&_Inbox.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_Inbox *InboxCaller) SequencerInbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Inbox.contract.Call(opts, &out, "sequencerInbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_Inbox *InboxSession) SequencerInbox() (common.Address, error) {
	return _Inbox.Contract.SequencerInbox(&_Inbox.CallOpts)
}

// SequencerInbox is a free data retrieval call binding the contract method 0xee35f327.
//
// Solidity: function sequencerInbox() view returns(address)
func (_Inbox *InboxCallerSession) SequencerInbox() (common.Address, error) {
	return _Inbox.Contract.SequencerInbox(&_Inbox.CallOpts)
}

// CreateRetryableTicket is a paid mutator transaction binding the contract method 0x679b6ded.
//
// Solidity: function createRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactor) CreateRetryableTicket(opts *bind.TransactOpts, to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "createRetryableTicket", to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// CreateRetryableTicket is a paid mutator transaction binding the contract method 0x679b6ded.
//
// Solidity: function createRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_Inbox *InboxSession) CreateRetryableTicket(to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.CreateRetryableTicket(&_Inbox.TransactOpts, to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// CreateRetryableTicket is a paid mutator transaction binding the contract method 0x679b6ded.
//
// Solidity: function createRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactorSession) CreateRetryableTicket(to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.CreateRetryableTicket(&_Inbox.TransactOpts, to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// CreateRetryableTicketNoRefundAliasRewrite is a paid mutator transaction binding the contract method 0x1b871c8d.
//
// Solidity: function createRetryableTicketNoRefundAliasRewrite(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactor) CreateRetryableTicketNoRefundAliasRewrite(opts *bind.TransactOpts, to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "createRetryableTicketNoRefundAliasRewrite", to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// CreateRetryableTicketNoRefundAliasRewrite is a paid mutator transaction binding the contract method 0x1b871c8d.
//
// Solidity: function createRetryableTicketNoRefundAliasRewrite(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_Inbox *InboxSession) CreateRetryableTicketNoRefundAliasRewrite(to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.CreateRetryableTicketNoRefundAliasRewrite(&_Inbox.TransactOpts, to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// CreateRetryableTicketNoRefundAliasRewrite is a paid mutator transaction binding the contract method 0x1b871c8d.
//
// Solidity: function createRetryableTicketNoRefundAliasRewrite(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactorSession) CreateRetryableTicketNoRefundAliasRewrite(to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.CreateRetryableTicketNoRefundAliasRewrite(&_Inbox.TransactOpts, to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// DepositEth is a paid mutator transaction binding the contract method 0x0f4d14e9.
//
// Solidity: function depositEth(uint256 ) payable returns(uint256)
func (_Inbox *InboxTransactor) DepositEth(opts *bind.TransactOpts, arg0 *big.Int) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "depositEth", arg0)
}

// DepositEth is a paid mutator transaction binding the contract method 0x0f4d14e9.
//
// Solidity: function depositEth(uint256 ) payable returns(uint256)
func (_Inbox *InboxSession) DepositEth(arg0 *big.Int) (*types.Transaction, error) {
	return _Inbox.Contract.DepositEth(&_Inbox.TransactOpts, arg0)
}

// DepositEth is a paid mutator transaction binding the contract method 0x0f4d14e9.
//
// Solidity: function depositEth(uint256 ) payable returns(uint256)
func (_Inbox *InboxTransactorSession) DepositEth(arg0 *big.Int) (*types.Transaction, error) {
	return _Inbox.Contract.DepositEth(&_Inbox.TransactOpts, arg0)
}

// DepositEth0 is a paid mutator transaction binding the contract method 0x439370b1.
//
// Solidity: function depositEth() payable returns(uint256)
func (_Inbox *InboxTransactor) DepositEth0(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "depositEth0")
}

// DepositEth0 is a paid mutator transaction binding the contract method 0x439370b1.
//
// Solidity: function depositEth() payable returns(uint256)
func (_Inbox *InboxSession) DepositEth0() (*types.Transaction, error) {
	return _Inbox.Contract.DepositEth0(&_Inbox.TransactOpts)
}

// DepositEth0 is a paid mutator transaction binding the contract method 0x439370b1.
//
// Solidity: function depositEth() payable returns(uint256)
func (_Inbox *InboxTransactorSession) DepositEth0() (*types.Transaction, error) {
	return _Inbox.Contract.DepositEth0(&_Inbox.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _bridge, address _sequencerInbox) returns()
func (_Inbox *InboxTransactor) Initialize(opts *bind.TransactOpts, _bridge common.Address, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "initialize", _bridge, _sequencerInbox)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _bridge, address _sequencerInbox) returns()
func (_Inbox *InboxSession) Initialize(_bridge common.Address, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _Inbox.Contract.Initialize(&_Inbox.TransactOpts, _bridge, _sequencerInbox)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _bridge, address _sequencerInbox) returns()
func (_Inbox *InboxTransactorSession) Initialize(_bridge common.Address, _sequencerInbox common.Address) (*types.Transaction, error) {
	return _Inbox.Contract.Initialize(&_Inbox.TransactOpts, _bridge, _sequencerInbox)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Inbox *InboxTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Inbox *InboxSession) Pause() (*types.Transaction, error) {
	return _Inbox.Contract.Pause(&_Inbox.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Inbox *InboxTransactorSession) Pause() (*types.Transaction, error) {
	return _Inbox.Contract.Pause(&_Inbox.TransactOpts)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xc474d2c5.
//
// Solidity: function postUpgradeInit(address ) returns()
func (_Inbox *InboxTransactor) PostUpgradeInit(opts *bind.TransactOpts, arg0 common.Address) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "postUpgradeInit", arg0)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xc474d2c5.
//
// Solidity: function postUpgradeInit(address ) returns()
func (_Inbox *InboxSession) PostUpgradeInit(arg0 common.Address) (*types.Transaction, error) {
	return _Inbox.Contract.PostUpgradeInit(&_Inbox.TransactOpts, arg0)
}

// PostUpgradeInit is a paid mutator transaction binding the contract method 0xc474d2c5.
//
// Solidity: function postUpgradeInit(address ) returns()
func (_Inbox *InboxTransactorSession) PostUpgradeInit(arg0 common.Address) (*types.Transaction, error) {
	return _Inbox.Contract.PostUpgradeInit(&_Inbox.TransactOpts, arg0)
}

// SendContractTransaction is a paid mutator transaction binding the contract method 0x8a631aa6.
//
// Solidity: function sendContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, uint256 value, bytes data) returns(uint256)
func (_Inbox *InboxTransactor) SendContractTransaction(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "sendContractTransaction", gasLimit, maxFeePerGas, to, value, data)
}

// SendContractTransaction is a paid mutator transaction binding the contract method 0x8a631aa6.
//
// Solidity: function sendContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, uint256 value, bytes data) returns(uint256)
func (_Inbox *InboxSession) SendContractTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendContractTransaction(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, to, value, data)
}

// SendContractTransaction is a paid mutator transaction binding the contract method 0x8a631aa6.
//
// Solidity: function sendContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, uint256 value, bytes data) returns(uint256)
func (_Inbox *InboxTransactorSession) SendContractTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendContractTransaction(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, to, value, data)
}

// SendL1FundedContractTransaction is a paid mutator transaction binding the contract method 0x5e916758.
//
// Solidity: function sendL1FundedContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactor) SendL1FundedContractTransaction(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "sendL1FundedContractTransaction", gasLimit, maxFeePerGas, to, data)
}

// SendL1FundedContractTransaction is a paid mutator transaction binding the contract method 0x5e916758.
//
// Solidity: function sendL1FundedContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, bytes data) payable returns(uint256)
func (_Inbox *InboxSession) SendL1FundedContractTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendL1FundedContractTransaction(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, to, data)
}

// SendL1FundedContractTransaction is a paid mutator transaction binding the contract method 0x5e916758.
//
// Solidity: function sendL1FundedContractTransaction(uint256 gasLimit, uint256 maxFeePerGas, address to, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactorSession) SendL1FundedContractTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendL1FundedContractTransaction(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, to, data)
}

// SendL1FundedUnsignedTransaction is a paid mutator transaction binding the contract method 0x67ef3ab8.
//
// Solidity: function sendL1FundedUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactor) SendL1FundedUnsignedTransaction(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "sendL1FundedUnsignedTransaction", gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL1FundedUnsignedTransaction is a paid mutator transaction binding the contract method 0x67ef3ab8.
//
// Solidity: function sendL1FundedUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_Inbox *InboxSession) SendL1FundedUnsignedTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendL1FundedUnsignedTransaction(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL1FundedUnsignedTransaction is a paid mutator transaction binding the contract method 0x67ef3ab8.
//
// Solidity: function sendL1FundedUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactorSession) SendL1FundedUnsignedTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendL1FundedUnsignedTransaction(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL1FundedUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0xe6bd12cf.
//
// Solidity: function sendL1FundedUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactor) SendL1FundedUnsignedTransactionToFork(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "sendL1FundedUnsignedTransactionToFork", gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL1FundedUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0xe6bd12cf.
//
// Solidity: function sendL1FundedUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_Inbox *InboxSession) SendL1FundedUnsignedTransactionToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendL1FundedUnsignedTransactionToFork(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL1FundedUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0xe6bd12cf.
//
// Solidity: function sendL1FundedUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactorSession) SendL1FundedUnsignedTransactionToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendL1FundedUnsignedTransactionToFork(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, data)
}

// SendL2Message is a paid mutator transaction binding the contract method 0xb75436bb.
//
// Solidity: function sendL2Message(bytes messageData) returns(uint256)
func (_Inbox *InboxTransactor) SendL2Message(opts *bind.TransactOpts, messageData []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "sendL2Message", messageData)
}

// SendL2Message is a paid mutator transaction binding the contract method 0xb75436bb.
//
// Solidity: function sendL2Message(bytes messageData) returns(uint256)
func (_Inbox *InboxSession) SendL2Message(messageData []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendL2Message(&_Inbox.TransactOpts, messageData)
}

// SendL2Message is a paid mutator transaction binding the contract method 0xb75436bb.
//
// Solidity: function sendL2Message(bytes messageData) returns(uint256)
func (_Inbox *InboxTransactorSession) SendL2Message(messageData []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendL2Message(&_Inbox.TransactOpts, messageData)
}

// SendL2MessageFromOrigin is a paid mutator transaction binding the contract method 0x1fe927cf.
//
// Solidity: function sendL2MessageFromOrigin(bytes messageData) returns(uint256)
func (_Inbox *InboxTransactor) SendL2MessageFromOrigin(opts *bind.TransactOpts, messageData []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "sendL2MessageFromOrigin", messageData)
}

// SendL2MessageFromOrigin is a paid mutator transaction binding the contract method 0x1fe927cf.
//
// Solidity: function sendL2MessageFromOrigin(bytes messageData) returns(uint256)
func (_Inbox *InboxSession) SendL2MessageFromOrigin(messageData []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendL2MessageFromOrigin(&_Inbox.TransactOpts, messageData)
}

// SendL2MessageFromOrigin is a paid mutator transaction binding the contract method 0x1fe927cf.
//
// Solidity: function sendL2MessageFromOrigin(bytes messageData) returns(uint256)
func (_Inbox *InboxTransactorSession) SendL2MessageFromOrigin(messageData []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendL2MessageFromOrigin(&_Inbox.TransactOpts, messageData)
}

// SendUnsignedTransaction is a paid mutator transaction binding the contract method 0x5075788b.
//
// Solidity: function sendUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_Inbox *InboxTransactor) SendUnsignedTransaction(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "sendUnsignedTransaction", gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendUnsignedTransaction is a paid mutator transaction binding the contract method 0x5075788b.
//
// Solidity: function sendUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_Inbox *InboxSession) SendUnsignedTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendUnsignedTransaction(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendUnsignedTransaction is a paid mutator transaction binding the contract method 0x5075788b.
//
// Solidity: function sendUnsignedTransaction(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_Inbox *InboxTransactorSession) SendUnsignedTransaction(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendUnsignedTransaction(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_Inbox *InboxTransactor) SendUnsignedTransactionToFork(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "sendUnsignedTransactionToFork", gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_Inbox *InboxSession) SendUnsignedTransactionToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendUnsignedTransactionToFork(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendUnsignedTransactionToFork is a paid mutator transaction binding the contract method 0x00f72382.
//
// Solidity: function sendUnsignedTransactionToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, address to, uint256 value, bytes data) returns(uint256)
func (_Inbox *InboxTransactorSession) SendUnsignedTransactionToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.SendUnsignedTransactionToFork(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, nonce, to, value, data)
}

// SendWithdrawEthToFork is a paid mutator transaction binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, uint256 value, address withdrawTo) returns(uint256)
func (_Inbox *InboxTransactor) SendWithdrawEthToFork(opts *bind.TransactOpts, gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, value *big.Int, withdrawTo common.Address) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "sendWithdrawEthToFork", gasLimit, maxFeePerGas, nonce, value, withdrawTo)
}

// SendWithdrawEthToFork is a paid mutator transaction binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, uint256 value, address withdrawTo) returns(uint256)
func (_Inbox *InboxSession) SendWithdrawEthToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, value *big.Int, withdrawTo common.Address) (*types.Transaction, error) {
	return _Inbox.Contract.SendWithdrawEthToFork(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, nonce, value, withdrawTo)
}

// SendWithdrawEthToFork is a paid mutator transaction binding the contract method 0x70665f14.
//
// Solidity: function sendWithdrawEthToFork(uint256 gasLimit, uint256 maxFeePerGas, uint256 nonce, uint256 value, address withdrawTo) returns(uint256)
func (_Inbox *InboxTransactorSession) SendWithdrawEthToFork(gasLimit *big.Int, maxFeePerGas *big.Int, nonce *big.Int, value *big.Int, withdrawTo common.Address) (*types.Transaction, error) {
	return _Inbox.Contract.SendWithdrawEthToFork(&_Inbox.TransactOpts, gasLimit, maxFeePerGas, nonce, value, withdrawTo)
}

// SetAllowList is a paid mutator transaction binding the contract method 0xe3de72a5.
//
// Solidity: function setAllowList(address[] user, bool[] val) returns()
func (_Inbox *InboxTransactor) SetAllowList(opts *bind.TransactOpts, user []common.Address, val []bool) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "setAllowList", user, val)
}

// SetAllowList is a paid mutator transaction binding the contract method 0xe3de72a5.
//
// Solidity: function setAllowList(address[] user, bool[] val) returns()
func (_Inbox *InboxSession) SetAllowList(user []common.Address, val []bool) (*types.Transaction, error) {
	return _Inbox.Contract.SetAllowList(&_Inbox.TransactOpts, user, val)
}

// SetAllowList is a paid mutator transaction binding the contract method 0xe3de72a5.
//
// Solidity: function setAllowList(address[] user, bool[] val) returns()
func (_Inbox *InboxTransactorSession) SetAllowList(user []common.Address, val []bool) (*types.Transaction, error) {
	return _Inbox.Contract.SetAllowList(&_Inbox.TransactOpts, user, val)
}

// SetAllowListEnabled is a paid mutator transaction binding the contract method 0xefeadb6d.
//
// Solidity: function setAllowListEnabled(bool _allowListEnabled) returns()
func (_Inbox *InboxTransactor) SetAllowListEnabled(opts *bind.TransactOpts, _allowListEnabled bool) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "setAllowListEnabled", _allowListEnabled)
}

// SetAllowListEnabled is a paid mutator transaction binding the contract method 0xefeadb6d.
//
// Solidity: function setAllowListEnabled(bool _allowListEnabled) returns()
func (_Inbox *InboxSession) SetAllowListEnabled(_allowListEnabled bool) (*types.Transaction, error) {
	return _Inbox.Contract.SetAllowListEnabled(&_Inbox.TransactOpts, _allowListEnabled)
}

// SetAllowListEnabled is a paid mutator transaction binding the contract method 0xefeadb6d.
//
// Solidity: function setAllowListEnabled(bool _allowListEnabled) returns()
func (_Inbox *InboxTransactorSession) SetAllowListEnabled(_allowListEnabled bool) (*types.Transaction, error) {
	return _Inbox.Contract.SetAllowListEnabled(&_Inbox.TransactOpts, _allowListEnabled)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_Inbox *InboxTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_Inbox *InboxSession) Unpause() (*types.Transaction, error) {
	return _Inbox.Contract.Unpause(&_Inbox.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_Inbox *InboxTransactorSession) Unpause() (*types.Transaction, error) {
	return _Inbox.Contract.Unpause(&_Inbox.TransactOpts)
}

// UnsafeCreateRetryableTicket is a paid mutator transaction binding the contract method 0x6e6e8a6a.
//
// Solidity: function unsafeCreateRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactor) UnsafeCreateRetryableTicket(opts *bind.TransactOpts, to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.contract.Transact(opts, "unsafeCreateRetryableTicket", to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// UnsafeCreateRetryableTicket is a paid mutator transaction binding the contract method 0x6e6e8a6a.
//
// Solidity: function unsafeCreateRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_Inbox *InboxSession) UnsafeCreateRetryableTicket(to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.UnsafeCreateRetryableTicket(&_Inbox.TransactOpts, to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// UnsafeCreateRetryableTicket is a paid mutator transaction binding the contract method 0x6e6e8a6a.
//
// Solidity: function unsafeCreateRetryableTicket(address to, uint256 l2CallValue, uint256 maxSubmissionCost, address excessFeeRefundAddress, address callValueRefundAddress, uint256 gasLimit, uint256 maxFeePerGas, bytes data) payable returns(uint256)
func (_Inbox *InboxTransactorSession) UnsafeCreateRetryableTicket(to common.Address, l2CallValue *big.Int, maxSubmissionCost *big.Int, excessFeeRefundAddress common.Address, callValueRefundAddress common.Address, gasLimit *big.Int, maxFeePerGas *big.Int, data []byte) (*types.Transaction, error) {
	return _Inbox.Contract.UnsafeCreateRetryableTicket(&_Inbox.TransactOpts, to, l2CallValue, maxSubmissionCost, excessFeeRefundAddress, callValueRefundAddress, gasLimit, maxFeePerGas, data)
}

// InboxAllowListAddressSetIterator is returned from FilterAllowListAddressSet and is used to iterate over the raw logs and unpacked data for AllowListAddressSet events raised by the Inbox contract.
type InboxAllowListAddressSetIterator struct {
	Event *InboxAllowListAddressSet // Event containing the contract specifics and raw log

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
func (it *InboxAllowListAddressSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InboxAllowListAddressSet)
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
		it.Event = new(InboxAllowListAddressSet)
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
func (it *InboxAllowListAddressSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InboxAllowListAddressSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InboxAllowListAddressSet represents a AllowListAddressSet event raised by the Inbox contract.
type InboxAllowListAddressSet struct {
	User common.Address
	Val  bool
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterAllowListAddressSet is a free log retrieval operation binding the contract event 0xd9739f45a01ce092c5cdb3d68f63d63d21676b1c6c0b4f9cbc6be4cf5449595a.
//
// Solidity: event AllowListAddressSet(address indexed user, bool val)
func (_Inbox *InboxFilterer) FilterAllowListAddressSet(opts *bind.FilterOpts, user []common.Address) (*InboxAllowListAddressSetIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _Inbox.contract.FilterLogs(opts, "AllowListAddressSet", userRule)
	if err != nil {
		return nil, err
	}
	return &InboxAllowListAddressSetIterator{contract: _Inbox.contract, event: "AllowListAddressSet", logs: logs, sub: sub}, nil
}

// WatchAllowListAddressSet is a free log subscription operation binding the contract event 0xd9739f45a01ce092c5cdb3d68f63d63d21676b1c6c0b4f9cbc6be4cf5449595a.
//
// Solidity: event AllowListAddressSet(address indexed user, bool val)
func (_Inbox *InboxFilterer) WatchAllowListAddressSet(opts *bind.WatchOpts, sink chan<- *InboxAllowListAddressSet, user []common.Address) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _Inbox.contract.WatchLogs(opts, "AllowListAddressSet", userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InboxAllowListAddressSet)
				if err := _Inbox.contract.UnpackLog(event, "AllowListAddressSet", log); err != nil {
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

// ParseAllowListAddressSet is a log parse operation binding the contract event 0xd9739f45a01ce092c5cdb3d68f63d63d21676b1c6c0b4f9cbc6be4cf5449595a.
//
// Solidity: event AllowListAddressSet(address indexed user, bool val)
func (_Inbox *InboxFilterer) ParseAllowListAddressSet(log types.Log) (*InboxAllowListAddressSet, error) {
	event := new(InboxAllowListAddressSet)
	if err := _Inbox.contract.UnpackLog(event, "AllowListAddressSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InboxAllowListEnabledUpdatedIterator is returned from FilterAllowListEnabledUpdated and is used to iterate over the raw logs and unpacked data for AllowListEnabledUpdated events raised by the Inbox contract.
type InboxAllowListEnabledUpdatedIterator struct {
	Event *InboxAllowListEnabledUpdated // Event containing the contract specifics and raw log

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
func (it *InboxAllowListEnabledUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InboxAllowListEnabledUpdated)
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
		it.Event = new(InboxAllowListEnabledUpdated)
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
func (it *InboxAllowListEnabledUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InboxAllowListEnabledUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InboxAllowListEnabledUpdated represents a AllowListEnabledUpdated event raised by the Inbox contract.
type InboxAllowListEnabledUpdated struct {
	IsEnabled bool
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAllowListEnabledUpdated is a free log retrieval operation binding the contract event 0x16435b45f7482047f839a6a19d291442627200f52cad2803c595150d0d440eb3.
//
// Solidity: event AllowListEnabledUpdated(bool isEnabled)
func (_Inbox *InboxFilterer) FilterAllowListEnabledUpdated(opts *bind.FilterOpts) (*InboxAllowListEnabledUpdatedIterator, error) {

	logs, sub, err := _Inbox.contract.FilterLogs(opts, "AllowListEnabledUpdated")
	if err != nil {
		return nil, err
	}
	return &InboxAllowListEnabledUpdatedIterator{contract: _Inbox.contract, event: "AllowListEnabledUpdated", logs: logs, sub: sub}, nil
}

// WatchAllowListEnabledUpdated is a free log subscription operation binding the contract event 0x16435b45f7482047f839a6a19d291442627200f52cad2803c595150d0d440eb3.
//
// Solidity: event AllowListEnabledUpdated(bool isEnabled)
func (_Inbox *InboxFilterer) WatchAllowListEnabledUpdated(opts *bind.WatchOpts, sink chan<- *InboxAllowListEnabledUpdated) (event.Subscription, error) {

	logs, sub, err := _Inbox.contract.WatchLogs(opts, "AllowListEnabledUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InboxAllowListEnabledUpdated)
				if err := _Inbox.contract.UnpackLog(event, "AllowListEnabledUpdated", log); err != nil {
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

// ParseAllowListEnabledUpdated is a log parse operation binding the contract event 0x16435b45f7482047f839a6a19d291442627200f52cad2803c595150d0d440eb3.
//
// Solidity: event AllowListEnabledUpdated(bool isEnabled)
func (_Inbox *InboxFilterer) ParseAllowListEnabledUpdated(log types.Log) (*InboxAllowListEnabledUpdated, error) {
	event := new(InboxAllowListEnabledUpdated)
	if err := _Inbox.contract.UnpackLog(event, "AllowListEnabledUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InboxInboxMessageDeliveredIterator is returned from FilterInboxMessageDelivered and is used to iterate over the raw logs and unpacked data for InboxMessageDelivered events raised by the Inbox contract.
type InboxInboxMessageDeliveredIterator struct {
	Event *InboxInboxMessageDelivered // Event containing the contract specifics and raw log

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
func (it *InboxInboxMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InboxInboxMessageDelivered)
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
		it.Event = new(InboxInboxMessageDelivered)
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
func (it *InboxInboxMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InboxInboxMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InboxInboxMessageDelivered represents a InboxMessageDelivered event raised by the Inbox contract.
type InboxInboxMessageDelivered struct {
	MessageNum *big.Int
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDelivered is a free log retrieval operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_Inbox *InboxFilterer) FilterInboxMessageDelivered(opts *bind.FilterOpts, messageNum []*big.Int) (*InboxInboxMessageDeliveredIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _Inbox.contract.FilterLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &InboxInboxMessageDeliveredIterator{contract: _Inbox.contract, event: "InboxMessageDelivered", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDelivered is a free log subscription operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_Inbox *InboxFilterer) WatchInboxMessageDelivered(opts *bind.WatchOpts, sink chan<- *InboxInboxMessageDelivered, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _Inbox.contract.WatchLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InboxInboxMessageDelivered)
				if err := _Inbox.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
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
func (_Inbox *InboxFilterer) ParseInboxMessageDelivered(log types.Log) (*InboxInboxMessageDelivered, error) {
	event := new(InboxInboxMessageDelivered)
	if err := _Inbox.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InboxInboxMessageDeliveredFromOriginIterator is returned from FilterInboxMessageDeliveredFromOrigin and is used to iterate over the raw logs and unpacked data for InboxMessageDeliveredFromOrigin events raised by the Inbox contract.
type InboxInboxMessageDeliveredFromOriginIterator struct {
	Event *InboxInboxMessageDeliveredFromOrigin // Event containing the contract specifics and raw log

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
func (it *InboxInboxMessageDeliveredFromOriginIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InboxInboxMessageDeliveredFromOrigin)
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
		it.Event = new(InboxInboxMessageDeliveredFromOrigin)
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
func (it *InboxInboxMessageDeliveredFromOriginIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InboxInboxMessageDeliveredFromOriginIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InboxInboxMessageDeliveredFromOrigin represents a InboxMessageDeliveredFromOrigin event raised by the Inbox contract.
type InboxInboxMessageDeliveredFromOrigin struct {
	MessageNum *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDeliveredFromOrigin is a free log retrieval operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_Inbox *InboxFilterer) FilterInboxMessageDeliveredFromOrigin(opts *bind.FilterOpts, messageNum []*big.Int) (*InboxInboxMessageDeliveredFromOriginIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _Inbox.contract.FilterLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &InboxInboxMessageDeliveredFromOriginIterator{contract: _Inbox.contract, event: "InboxMessageDeliveredFromOrigin", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDeliveredFromOrigin is a free log subscription operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_Inbox *InboxFilterer) WatchInboxMessageDeliveredFromOrigin(opts *bind.WatchOpts, sink chan<- *InboxInboxMessageDeliveredFromOrigin, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _Inbox.contract.WatchLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InboxInboxMessageDeliveredFromOrigin)
				if err := _Inbox.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
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
func (_Inbox *InboxFilterer) ParseInboxMessageDeliveredFromOrigin(log types.Log) (*InboxInboxMessageDeliveredFromOrigin, error) {
	event := new(InboxInboxMessageDeliveredFromOrigin)
	if err := _Inbox.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InboxInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Inbox contract.
type InboxInitializedIterator struct {
	Event *InboxInitialized // Event containing the contract specifics and raw log

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
func (it *InboxInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InboxInitialized)
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
		it.Event = new(InboxInitialized)
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
func (it *InboxInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InboxInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InboxInitialized represents a Initialized event raised by the Inbox contract.
type InboxInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_Inbox *InboxFilterer) FilterInitialized(opts *bind.FilterOpts) (*InboxInitializedIterator, error) {

	logs, sub, err := _Inbox.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &InboxInitializedIterator{contract: _Inbox.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_Inbox *InboxFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *InboxInitialized) (event.Subscription, error) {

	logs, sub, err := _Inbox.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InboxInitialized)
				if err := _Inbox.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_Inbox *InboxFilterer) ParseInitialized(log types.Log) (*InboxInitialized, error) {
	event := new(InboxInitialized)
	if err := _Inbox.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InboxPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the Inbox contract.
type InboxPausedIterator struct {
	Event *InboxPaused // Event containing the contract specifics and raw log

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
func (it *InboxPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InboxPaused)
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
		it.Event = new(InboxPaused)
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
func (it *InboxPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InboxPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InboxPaused represents a Paused event raised by the Inbox contract.
type InboxPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Inbox *InboxFilterer) FilterPaused(opts *bind.FilterOpts) (*InboxPausedIterator, error) {

	logs, sub, err := _Inbox.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &InboxPausedIterator{contract: _Inbox.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Inbox *InboxFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *InboxPaused) (event.Subscription, error) {

	logs, sub, err := _Inbox.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InboxPaused)
				if err := _Inbox.contract.UnpackLog(event, "Paused", log); err != nil {
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

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Inbox *InboxFilterer) ParsePaused(log types.Log) (*InboxPaused, error) {
	event := new(InboxPaused)
	if err := _Inbox.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InboxUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the Inbox contract.
type InboxUnpausedIterator struct {
	Event *InboxUnpaused // Event containing the contract specifics and raw log

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
func (it *InboxUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InboxUnpaused)
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
		it.Event = new(InboxUnpaused)
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
func (it *InboxUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InboxUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InboxUnpaused represents a Unpaused event raised by the Inbox contract.
type InboxUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Inbox *InboxFilterer) FilterUnpaused(opts *bind.FilterOpts) (*InboxUnpausedIterator, error) {

	logs, sub, err := _Inbox.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &InboxUnpausedIterator{contract: _Inbox.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Inbox *InboxFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *InboxUnpaused) (event.Subscription, error) {

	logs, sub, err := _Inbox.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InboxUnpaused)
				if err := _Inbox.contract.UnpackLog(event, "Unpaused", log); err != nil {
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

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Inbox *InboxFilterer) ParseUnpaused(log types.Log) (*InboxUnpaused, error) {
	event := new(InboxUnpaused)
	if err := _Inbox.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MessagesMetaData contains all meta data concerning the Messages contract.
var MessagesMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220bb427be497f5b76f3003d3528754ab3047e12d8e877d217dd77853d9c24206fd64736f6c63430008110033",
}

// MessagesABI is the input ABI used to generate the binding from.
// Deprecated: Use MessagesMetaData.ABI instead.
var MessagesABI = MessagesMetaData.ABI

// MessagesBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MessagesMetaData.Bin instead.
var MessagesBin = MessagesMetaData.Bin

// DeployMessages deploys a new Ethereum contract, binding an instance of Messages to it.
func DeployMessages(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Messages, error) {
	parsed, err := MessagesMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MessagesBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Messages{MessagesCaller: MessagesCaller{contract: contract}, MessagesTransactor: MessagesTransactor{contract: contract}, MessagesFilterer: MessagesFilterer{contract: contract}}, nil
}

// Messages is an auto generated Go binding around an Ethereum contract.
type Messages struct {
	MessagesCaller     // Read-only binding to the contract
	MessagesTransactor // Write-only binding to the contract
	MessagesFilterer   // Log filterer for contract events
}

// MessagesCaller is an auto generated read-only Go binding around an Ethereum contract.
type MessagesCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MessagesTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MessagesTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MessagesFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MessagesFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MessagesSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MessagesSession struct {
	Contract     *Messages         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MessagesCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MessagesCallerSession struct {
	Contract *MessagesCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// MessagesTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MessagesTransactorSession struct {
	Contract     *MessagesTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// MessagesRaw is an auto generated low-level Go binding around an Ethereum contract.
type MessagesRaw struct {
	Contract *Messages // Generic contract binding to access the raw methods on
}

// MessagesCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MessagesCallerRaw struct {
	Contract *MessagesCaller // Generic read-only contract binding to access the raw methods on
}

// MessagesTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MessagesTransactorRaw struct {
	Contract *MessagesTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMessages creates a new instance of Messages, bound to a specific deployed contract.
func NewMessages(address common.Address, backend bind.ContractBackend) (*Messages, error) {
	contract, err := bindMessages(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Messages{MessagesCaller: MessagesCaller{contract: contract}, MessagesTransactor: MessagesTransactor{contract: contract}, MessagesFilterer: MessagesFilterer{contract: contract}}, nil
}

// NewMessagesCaller creates a new read-only instance of Messages, bound to a specific deployed contract.
func NewMessagesCaller(address common.Address, caller bind.ContractCaller) (*MessagesCaller, error) {
	contract, err := bindMessages(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MessagesCaller{contract: contract}, nil
}

// NewMessagesTransactor creates a new write-only instance of Messages, bound to a specific deployed contract.
func NewMessagesTransactor(address common.Address, transactor bind.ContractTransactor) (*MessagesTransactor, error) {
	contract, err := bindMessages(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MessagesTransactor{contract: contract}, nil
}

// NewMessagesFilterer creates a new log filterer instance of Messages, bound to a specific deployed contract.
func NewMessagesFilterer(address common.Address, filterer bind.ContractFilterer) (*MessagesFilterer, error) {
	contract, err := bindMessages(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MessagesFilterer{contract: contract}, nil
}

// bindMessages binds a generic wrapper to an already deployed contract.
func bindMessages(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MessagesMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Messages *MessagesRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Messages.Contract.MessagesCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Messages *MessagesRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Messages.Contract.MessagesTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Messages *MessagesRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Messages.Contract.MessagesTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Messages *MessagesCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Messages.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Messages *MessagesTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Messages.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Messages *MessagesTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Messages.Contract.contract.Transact(opts, method, params...)
}

// OutboxMetaData contains all meta data concerning the Outbox contract.
var OutboxMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"AlreadyInit\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"AlreadySpent\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BridgeCallFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HadZeroInit\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"actualLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxProofLength\",\"type\":\"uint256\"}],\"name\":\"MerkleProofTooLong\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"}],\"name\":\"NotRollup\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxIndex\",\"type\":\"uint256\"}],\"name\":\"PathNotMinimal\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"proofLength\",\"type\":\"uint256\"}],\"name\":\"ProofTooLong\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SimulationOnlyEntrypoint\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"}],\"name\":\"UnknownRoot\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"zero\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"transactionIndex\",\"type\":\"uint256\"}],\"name\":\"OutBoxTransactionExecuted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"l2BlockHash\",\"type\":\"bytes32\"}],\"name\":\"SendRootUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"OUTBOX_VERSION\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l2Timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"calculateItemHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"path\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"item\",\"type\":\"bytes32\"}],\"name\":\"calculateMerkleRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l2Timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"l2Sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l1Block\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"l2Timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"executeTransactionSimulation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"_bridge\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"isSpent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1BatchNum\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1Block\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1EthBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1OutputId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1Sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2ToL1Timestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"roots\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"spent\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"l2BlockHash\",\"type\":\"bytes32\"}],\"name\":\"updateSendRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a06040523060805234801561001457600080fd5b506080516117cd6100376000396000818161043c01526107bf01526117cd6000f3fe608060405234801561001057600080fd5b50600436106101505760003560e01c80638515bc6a116100cd578063c4d66de811610081578063cb23bcb511610066578063cb23bcb51461028e578063d5b5cc23146102a1578063e78cea92146102c157600080fd5b8063c4d66de81461025b578063c75184df1461026e57600080fd5b8063a04cee60116100b2578063a04cee6014610220578063ae6dead714610233578063b0f305371461025357600080fd5b80638515bc6a146102055780639f0c04bf1461020d57600080fd5b806346547790116101245780636ae71f12116101095780636ae71f12146101d557806372f2a8c7146101dd57806380648b02146101e557600080fd5b806346547790146101aa5780635a129efe146101b257600080fd5b80627436d31461015557806308635a951461017b5780631198527114610190578063288e5b1014610197575b600080fd5b6101686101633660046110f6565b6102d4565b6040519081526020015b60405180910390f35b61018e610189366004611218565b610311565b005b6000610168565b61018e6101a536600461130d565b610384565b6101686103d8565b6101c56101c03660046113a9565b610415565b6040519015158152602001610172565b61018e610432565b61016861060d565b6101ed610628565b6040516001600160a01b039091168152602001610172565b610168610666565b61016861021b3660046113c2565b6106a9565b61018e61022e366004611451565b6106ee565b6101686102413660046113a9565b60036020526000908152604090205481565b610168610786565b61018e610269366004611473565b6107b5565b610276600281565b6040516001600160801b039091168152602001610172565b6000546101ed906001600160a01b031681565b6101686102af3660046113a9565b60026020526000908152604090205481565b6001546101ed906001600160a01b031681565b60006103098484846040516020016102ee91815260200190565b60405160208183030381529060405280519060200120610a34565b949350505050565b600061032389898989898989896106a9565b90506103658c8c808060200260200160405190810160405280939291908181526020018383602002808284376000920191909152508e9250859150610aef9050565b6103768a8a8a8a8a8a8a8a8a610c5c565b505050505050505050505050565b33156103bc576040517f0e13b69d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6103cd898989898989898989610c5c565b505050505050505050565b6004546000906001600160801b03166ffffffffffffffffffffffffffffffffe19810161040757600091505090565b6001600160801b0316919050565b600080600061042384610f78565b92509250506103098282610fb5565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036104ef576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084015b60405180910390fd5b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038054336001600160a01b03821614610565576040517f23295f0e0000000000000000000000000000000000000000000000000000000081523360048201526001600160a01b03821660248201526044016104e6565b600160009054906101000a90046001600160a01b03166001600160a01b031663cb23bcb56040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105b8573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105dc9190611497565b6000805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03929092169190911790555050565b6006546000906001810161062357506000919050565b919050565b6007546000906001600160a01b03167fffffffffffffffffffffffff0000000000000000000000000000000000000001810161062357600091505090565b60045460009070010000000000000000000000000000000090046001600160801b03166ffffffffffffffffffffffffffffffffe19810161040757600091505090565b600088888888888888886040516020016106ca9897969594939291906114b4565b60405160208183030381529060405280519060200120905098975050505050505050565b6000546001600160a01b03163314610747576000546040517f3933c6fc0000000000000000000000000000000000000000000000000000000081523360048201526001600160a01b0390911660248201526044016104e6565b60008281526003602052604080822083905551829184917fb4df3847300f076a369cd76d2314b470a1194d9e8a6bb97f1860aee88a5f67489190a35050565b6005546000906001600160801b03166ffffffffffffffffffffffffffffffffe19810161040757600091505090565b6001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016300361086d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084016104e6565b6001600160a01b0381166108ad576040517f1ad0f74300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001546001600160a01b0316156108f0576040517fef34ca5c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040805160a0810182526001600160801b038082526020808301829052828401829052600019606084018190526001600160a01b0360809094018490526004818155600580547fffffffffffffffffffffffffffffffff00000000000000000000000000000000169094179093556006556007805473ffffffffffffffffffffffffffffffffffffffff19908116851790915560018054948716949091168417905583517fcb23bcb50000000000000000000000000000000000000000000000000000000081529351929363cb23bcb5938184019390918290030181865afa1580156109e0573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610a049190611497565b6000805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b039290921691909117905550565b8251600090610100811115610a80576040517ffdac331e0000000000000000000000000000000000000000000000000000000081526004810182905261010060248201526044016104e6565b8260005b82811015610ae5576000878281518110610aa057610aa0611520565b60200260200101519050816001901b8716600003610acc57826000528060205260406000209250610adc565b8060005282602052604060002092505b50600101610a84565b5095945050505050565b610100835110610b305782516040517fab6a06830000000000000000000000000000000000000000000000000000000081526004016104e691815260200190565b8251610b3d906002611630565b8210610b8d578183516002610b529190611630565b6040517f0b8a724b000000000000000000000000000000000000000000000000000000008152600481019290925260248201526044016104e6565b6000610b9a8484846102d4565b600081815260036020526040902054909150610be5576040517f8730d7c8000000000000000000000000000000000000000000000000000000008152600481018290526024016104e6565b6000806000610bf386610f78565b925092509250610c038282610fb5565b15610c3d576040517f9715b8d3000000000000000000000000000000000000000000000000000000008152600481018790526024016104e6565b600092835260026020526040909220600190911b909117905550505050565b6000886001600160a01b0316886001600160a01b03167f20af7f3bbfe38132b8900ae295cd9c8d1914be7052d061a511f3f728dab189648c604051610ca391815260200190565b60405180910390a4600060046040518060a00160405290816000820160009054906101000a90046001600160801b03166001600160801b03166001600160801b031681526020016000820160109054906101000a90046001600160801b03166001600160801b03166001600160801b031681526020016001820160009054906101000a90046001600160801b03166001600160801b03166001600160801b03168152602001600282015481526020016003820160009054906101000a90046001600160a01b03166001600160a01b03166001600160a01b03168152505090506040518060a00160405280886001600160801b03168152602001876001600160801b03168152602001866001600160801b031681526020018b60001b81526020018a6001600160a01b0316815250600460008201518160000160006101000a8154816001600160801b0302191690836001600160801b0316021790555060208201518160000160106101000a8154816001600160801b0302191690836001600160801b0316021790555060408201518160010160006101000a8154816001600160801b0302191690836001600160801b031602179055506060820151816002015560808201518160030160006101000a8154816001600160a01b0302191690836001600160a01b03160217905550905050610ed4888585858080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250610fc492505050565b805160208201516001600160801b0391821670010000000000000000000000000000000091831691909102176004556040820151600580547fffffffffffffffffffffffffffffffff0000000000000000000000000000000016919092161790556060810151600655608001516007805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03909216919091179055505050505050505050565b6000808080610f8860ff86611652565b90506000610f9760ff87611666565b60008381526002602052604090205492979096509194509092505050565b80821c60011615155b92915050565b6001546040517f9e5d4c4900000000000000000000000000000000000000000000000000000000815260009182916001600160a01b0390911690639e5d4c49906110169088908890889060040161169e565b6000604051808303816000875af1158015611035573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405261105d91908101906116e8565b91509150816110a8578051156110765780518082602001fd5b6040517f376fb55a00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b5050505050565b634e487b7160e01b600052604160045260246000fd5b604051601f8201601f1916810167ffffffffffffffff811182821017156110ee576110ee6110af565b604052919050565b60008060006060848603121561110b57600080fd5b833567ffffffffffffffff8082111561112357600080fd5b818601915086601f83011261113757600080fd5b813560208282111561114b5761114b6110af565b8160051b925061115c8184016110c5565b828152928401810192818101908a85111561117657600080fd5b948201945b848610156111945785358252948201949082019061117b565b9a918901359950506040909701359695505050505050565b6001600160a01b03811681146111c157600080fd5b50565b8035610623816111ac565b60008083601f8401126111e157600080fd5b50813567ffffffffffffffff8111156111f957600080fd5b60208301915083602082850101111561121157600080fd5b9250929050565b60008060008060008060008060008060006101208c8e03121561123a57600080fd5b8b3567ffffffffffffffff8082111561125257600080fd5b818e0191508e601f83011261126657600080fd5b81358181111561127557600080fd5b8f60208260051b850101111561128a57600080fd5b60208381019e50909c508e01359a506112a560408f016111c4565b99506112b360608f016111c4565b985060808e0135975060a08e0135965060c08e0135955060e08e013594506101008e01359150808211156112e657600080fd5b506112f38e828f016111cf565b915080935050809150509295989b509295989b9093969950565b60008060008060008060008060006101008a8c03121561132c57600080fd5b8935985060208a013561133e816111ac565b975060408a013561134e816111ac565b965060608a0135955060808a0135945060a08a0135935060c08a0135925060e08a013567ffffffffffffffff81111561138657600080fd5b6113928c828d016111cf565b915080935050809150509295985092959850929598565b6000602082840312156113bb57600080fd5b5035919050565b60008060008060008060008060e0898b0312156113de57600080fd5b88356113e9816111ac565b975060208901356113f9816111ac565b965060408901359550606089013594506080890135935060a0890135925060c089013567ffffffffffffffff81111561143157600080fd5b61143d8b828c016111cf565b999c989b5096995094979396929594505050565b6000806040838503121561146457600080fd5b50508035926020909101359150565b60006020828403121561148557600080fd5b8135611490816111ac565b9392505050565b6000602082840312156114a957600080fd5b8151611490816111ac565b60007fffffffffffffffffffffffffffffffffffffffff000000000000000000000000808b60601b168352808a60601b16601484015250876028830152866048830152856068830152846088830152828460a8840137506000910160a801908152979650505050505050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b600181815b8085111561158757816000190482111561156d5761156d611536565b8085161561157a57918102915b93841c9390800290611551565b509250929050565b60008261159e57506001610fbe565b816115ab57506000610fbe565b81600181146115c157600281146115cb576115e7565b6001915050610fbe565b60ff8411156115dc576115dc611536565b50506001821b610fbe565b5060208310610133831016604e8410600b841016171561160a575081810a610fbe565b611614838361154c565b806000190482111561162857611628611536565b029392505050565b6000611490838361158f565b634e487b7160e01b600052601260045260246000fd5b6000826116615761166161163c565b500490565b6000826116755761167561163c565b500690565b60005b8381101561169557818101518382015260200161167d565b50506000910152565b6001600160a01b038416815282602082015260606040820152600082518060608401526116d281608085016020870161167a565b601f01601f191691909101608001949350505050565b600080604083850312156116fb57600080fd5b8251801515811461170b57600080fd5b602084015190925067ffffffffffffffff8082111561172957600080fd5b818501915085601f83011261173d57600080fd5b81518181111561174f5761174f6110af565b6117626020601f19601f840116016110c5565b915080825286602082850101111561177957600080fd5b61178a81602084016020860161167a565b508092505050925092905056fea2646970667358221220f97735c6ce65cfe41023d1626f05c807e67ba1b01a8fd4a0ace2ebdb6cc1e2fd64736f6c63430008110033",
}

// OutboxABI is the input ABI used to generate the binding from.
// Deprecated: Use OutboxMetaData.ABI instead.
var OutboxABI = OutboxMetaData.ABI

// OutboxBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OutboxMetaData.Bin instead.
var OutboxBin = OutboxMetaData.Bin

// DeployOutbox deploys a new Ethereum contract, binding an instance of Outbox to it.
func DeployOutbox(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Outbox, error) {
	parsed, err := OutboxMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OutboxBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Outbox{OutboxCaller: OutboxCaller{contract: contract}, OutboxTransactor: OutboxTransactor{contract: contract}, OutboxFilterer: OutboxFilterer{contract: contract}}, nil
}

// Outbox is an auto generated Go binding around an Ethereum contract.
type Outbox struct {
	OutboxCaller     // Read-only binding to the contract
	OutboxTransactor // Write-only binding to the contract
	OutboxFilterer   // Log filterer for contract events
}

// OutboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type OutboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OutboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OutboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OutboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OutboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OutboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OutboxSession struct {
	Contract     *Outbox           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OutboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OutboxCallerSession struct {
	Contract *OutboxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// OutboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OutboxTransactorSession struct {
	Contract     *OutboxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OutboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type OutboxRaw struct {
	Contract *Outbox // Generic contract binding to access the raw methods on
}

// OutboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OutboxCallerRaw struct {
	Contract *OutboxCaller // Generic read-only contract binding to access the raw methods on
}

// OutboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OutboxTransactorRaw struct {
	Contract *OutboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOutbox creates a new instance of Outbox, bound to a specific deployed contract.
func NewOutbox(address common.Address, backend bind.ContractBackend) (*Outbox, error) {
	contract, err := bindOutbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Outbox{OutboxCaller: OutboxCaller{contract: contract}, OutboxTransactor: OutboxTransactor{contract: contract}, OutboxFilterer: OutboxFilterer{contract: contract}}, nil
}

// NewOutboxCaller creates a new read-only instance of Outbox, bound to a specific deployed contract.
func NewOutboxCaller(address common.Address, caller bind.ContractCaller) (*OutboxCaller, error) {
	contract, err := bindOutbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OutboxCaller{contract: contract}, nil
}

// NewOutboxTransactor creates a new write-only instance of Outbox, bound to a specific deployed contract.
func NewOutboxTransactor(address common.Address, transactor bind.ContractTransactor) (*OutboxTransactor, error) {
	contract, err := bindOutbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OutboxTransactor{contract: contract}, nil
}

// NewOutboxFilterer creates a new log filterer instance of Outbox, bound to a specific deployed contract.
func NewOutboxFilterer(address common.Address, filterer bind.ContractFilterer) (*OutboxFilterer, error) {
	contract, err := bindOutbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OutboxFilterer{contract: contract}, nil
}

// bindOutbox binds a generic wrapper to an already deployed contract.
func bindOutbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OutboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Outbox *OutboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Outbox.Contract.OutboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Outbox *OutboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Outbox.Contract.OutboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Outbox *OutboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Outbox.Contract.OutboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Outbox *OutboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Outbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Outbox *OutboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Outbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Outbox *OutboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Outbox.Contract.contract.Transact(opts, method, params...)
}

// OUTBOXVERSION is a free data retrieval call binding the contract method 0xc75184df.
//
// Solidity: function OUTBOX_VERSION() view returns(uint128)
func (_Outbox *OutboxCaller) OUTBOXVERSION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "OUTBOX_VERSION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OUTBOXVERSION is a free data retrieval call binding the contract method 0xc75184df.
//
// Solidity: function OUTBOX_VERSION() view returns(uint128)
func (_Outbox *OutboxSession) OUTBOXVERSION() (*big.Int, error) {
	return _Outbox.Contract.OUTBOXVERSION(&_Outbox.CallOpts)
}

// OUTBOXVERSION is a free data retrieval call binding the contract method 0xc75184df.
//
// Solidity: function OUTBOX_VERSION() view returns(uint128)
func (_Outbox *OutboxCallerSession) OUTBOXVERSION() (*big.Int, error) {
	return _Outbox.Contract.OUTBOXVERSION(&_Outbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_Outbox *OutboxCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_Outbox *OutboxSession) Bridge() (common.Address, error) {
	return _Outbox.Contract.Bridge(&_Outbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_Outbox *OutboxCallerSession) Bridge() (common.Address, error) {
	return _Outbox.Contract.Bridge(&_Outbox.CallOpts)
}

// CalculateItemHash is a free data retrieval call binding the contract method 0x9f0c04bf.
//
// Solidity: function calculateItemHash(address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) pure returns(bytes32)
func (_Outbox *OutboxCaller) CalculateItemHash(opts *bind.CallOpts, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) ([32]byte, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "calculateItemHash", l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateItemHash is a free data retrieval call binding the contract method 0x9f0c04bf.
//
// Solidity: function calculateItemHash(address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) pure returns(bytes32)
func (_Outbox *OutboxSession) CalculateItemHash(l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) ([32]byte, error) {
	return _Outbox.Contract.CalculateItemHash(&_Outbox.CallOpts, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// CalculateItemHash is a free data retrieval call binding the contract method 0x9f0c04bf.
//
// Solidity: function calculateItemHash(address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) pure returns(bytes32)
func (_Outbox *OutboxCallerSession) CalculateItemHash(l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) ([32]byte, error) {
	return _Outbox.Contract.CalculateItemHash(&_Outbox.CallOpts, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// CalculateMerkleRoot is a free data retrieval call binding the contract method 0x007436d3.
//
// Solidity: function calculateMerkleRoot(bytes32[] proof, uint256 path, bytes32 item) pure returns(bytes32)
func (_Outbox *OutboxCaller) CalculateMerkleRoot(opts *bind.CallOpts, proof [][32]byte, path *big.Int, item [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "calculateMerkleRoot", proof, path, item)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateMerkleRoot is a free data retrieval call binding the contract method 0x007436d3.
//
// Solidity: function calculateMerkleRoot(bytes32[] proof, uint256 path, bytes32 item) pure returns(bytes32)
func (_Outbox *OutboxSession) CalculateMerkleRoot(proof [][32]byte, path *big.Int, item [32]byte) ([32]byte, error) {
	return _Outbox.Contract.CalculateMerkleRoot(&_Outbox.CallOpts, proof, path, item)
}

// CalculateMerkleRoot is a free data retrieval call binding the contract method 0x007436d3.
//
// Solidity: function calculateMerkleRoot(bytes32[] proof, uint256 path, bytes32 item) pure returns(bytes32)
func (_Outbox *OutboxCallerSession) CalculateMerkleRoot(proof [][32]byte, path *big.Int, item [32]byte) ([32]byte, error) {
	return _Outbox.Contract.CalculateMerkleRoot(&_Outbox.CallOpts, proof, path, item)
}

// IsSpent is a free data retrieval call binding the contract method 0x5a129efe.
//
// Solidity: function isSpent(uint256 index) view returns(bool)
func (_Outbox *OutboxCaller) IsSpent(opts *bind.CallOpts, index *big.Int) (bool, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "isSpent", index)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSpent is a free data retrieval call binding the contract method 0x5a129efe.
//
// Solidity: function isSpent(uint256 index) view returns(bool)
func (_Outbox *OutboxSession) IsSpent(index *big.Int) (bool, error) {
	return _Outbox.Contract.IsSpent(&_Outbox.CallOpts, index)
}

// IsSpent is a free data retrieval call binding the contract method 0x5a129efe.
//
// Solidity: function isSpent(uint256 index) view returns(bool)
func (_Outbox *OutboxCallerSession) IsSpent(index *big.Int) (bool, error) {
	return _Outbox.Contract.IsSpent(&_Outbox.CallOpts, index)
}

// L2ToL1BatchNum is a free data retrieval call binding the contract method 0x11985271.
//
// Solidity: function l2ToL1BatchNum() pure returns(uint256)
func (_Outbox *OutboxCaller) L2ToL1BatchNum(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "l2ToL1BatchNum")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1BatchNum is a free data retrieval call binding the contract method 0x11985271.
//
// Solidity: function l2ToL1BatchNum() pure returns(uint256)
func (_Outbox *OutboxSession) L2ToL1BatchNum() (*big.Int, error) {
	return _Outbox.Contract.L2ToL1BatchNum(&_Outbox.CallOpts)
}

// L2ToL1BatchNum is a free data retrieval call binding the contract method 0x11985271.
//
// Solidity: function l2ToL1BatchNum() pure returns(uint256)
func (_Outbox *OutboxCallerSession) L2ToL1BatchNum() (*big.Int, error) {
	return _Outbox.Contract.L2ToL1BatchNum(&_Outbox.CallOpts)
}

// L2ToL1Block is a free data retrieval call binding the contract method 0x46547790.
//
// Solidity: function l2ToL1Block() view returns(uint256)
func (_Outbox *OutboxCaller) L2ToL1Block(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "l2ToL1Block")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1Block is a free data retrieval call binding the contract method 0x46547790.
//
// Solidity: function l2ToL1Block() view returns(uint256)
func (_Outbox *OutboxSession) L2ToL1Block() (*big.Int, error) {
	return _Outbox.Contract.L2ToL1Block(&_Outbox.CallOpts)
}

// L2ToL1Block is a free data retrieval call binding the contract method 0x46547790.
//
// Solidity: function l2ToL1Block() view returns(uint256)
func (_Outbox *OutboxCallerSession) L2ToL1Block() (*big.Int, error) {
	return _Outbox.Contract.L2ToL1Block(&_Outbox.CallOpts)
}

// L2ToL1EthBlock is a free data retrieval call binding the contract method 0x8515bc6a.
//
// Solidity: function l2ToL1EthBlock() view returns(uint256)
func (_Outbox *OutboxCaller) L2ToL1EthBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "l2ToL1EthBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1EthBlock is a free data retrieval call binding the contract method 0x8515bc6a.
//
// Solidity: function l2ToL1EthBlock() view returns(uint256)
func (_Outbox *OutboxSession) L2ToL1EthBlock() (*big.Int, error) {
	return _Outbox.Contract.L2ToL1EthBlock(&_Outbox.CallOpts)
}

// L2ToL1EthBlock is a free data retrieval call binding the contract method 0x8515bc6a.
//
// Solidity: function l2ToL1EthBlock() view returns(uint256)
func (_Outbox *OutboxCallerSession) L2ToL1EthBlock() (*big.Int, error) {
	return _Outbox.Contract.L2ToL1EthBlock(&_Outbox.CallOpts)
}

// L2ToL1OutputId is a free data retrieval call binding the contract method 0x72f2a8c7.
//
// Solidity: function l2ToL1OutputId() view returns(bytes32)
func (_Outbox *OutboxCaller) L2ToL1OutputId(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "l2ToL1OutputId")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L2ToL1OutputId is a free data retrieval call binding the contract method 0x72f2a8c7.
//
// Solidity: function l2ToL1OutputId() view returns(bytes32)
func (_Outbox *OutboxSession) L2ToL1OutputId() ([32]byte, error) {
	return _Outbox.Contract.L2ToL1OutputId(&_Outbox.CallOpts)
}

// L2ToL1OutputId is a free data retrieval call binding the contract method 0x72f2a8c7.
//
// Solidity: function l2ToL1OutputId() view returns(bytes32)
func (_Outbox *OutboxCallerSession) L2ToL1OutputId() ([32]byte, error) {
	return _Outbox.Contract.L2ToL1OutputId(&_Outbox.CallOpts)
}

// L2ToL1Sender is a free data retrieval call binding the contract method 0x80648b02.
//
// Solidity: function l2ToL1Sender() view returns(address)
func (_Outbox *OutboxCaller) L2ToL1Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "l2ToL1Sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2ToL1Sender is a free data retrieval call binding the contract method 0x80648b02.
//
// Solidity: function l2ToL1Sender() view returns(address)
func (_Outbox *OutboxSession) L2ToL1Sender() (common.Address, error) {
	return _Outbox.Contract.L2ToL1Sender(&_Outbox.CallOpts)
}

// L2ToL1Sender is a free data retrieval call binding the contract method 0x80648b02.
//
// Solidity: function l2ToL1Sender() view returns(address)
func (_Outbox *OutboxCallerSession) L2ToL1Sender() (common.Address, error) {
	return _Outbox.Contract.L2ToL1Sender(&_Outbox.CallOpts)
}

// L2ToL1Timestamp is a free data retrieval call binding the contract method 0xb0f30537.
//
// Solidity: function l2ToL1Timestamp() view returns(uint256)
func (_Outbox *OutboxCaller) L2ToL1Timestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "l2ToL1Timestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2ToL1Timestamp is a free data retrieval call binding the contract method 0xb0f30537.
//
// Solidity: function l2ToL1Timestamp() view returns(uint256)
func (_Outbox *OutboxSession) L2ToL1Timestamp() (*big.Int, error) {
	return _Outbox.Contract.L2ToL1Timestamp(&_Outbox.CallOpts)
}

// L2ToL1Timestamp is a free data retrieval call binding the contract method 0xb0f30537.
//
// Solidity: function l2ToL1Timestamp() view returns(uint256)
func (_Outbox *OutboxCallerSession) L2ToL1Timestamp() (*big.Int, error) {
	return _Outbox.Contract.L2ToL1Timestamp(&_Outbox.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_Outbox *OutboxCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_Outbox *OutboxSession) Rollup() (common.Address, error) {
	return _Outbox.Contract.Rollup(&_Outbox.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_Outbox *OutboxCallerSession) Rollup() (common.Address, error) {
	return _Outbox.Contract.Rollup(&_Outbox.CallOpts)
}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(bytes32)
func (_Outbox *OutboxCaller) Roots(opts *bind.CallOpts, arg0 [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "roots", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(bytes32)
func (_Outbox *OutboxSession) Roots(arg0 [32]byte) ([32]byte, error) {
	return _Outbox.Contract.Roots(&_Outbox.CallOpts, arg0)
}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(bytes32)
func (_Outbox *OutboxCallerSession) Roots(arg0 [32]byte) ([32]byte, error) {
	return _Outbox.Contract.Roots(&_Outbox.CallOpts, arg0)
}

// Spent is a free data retrieval call binding the contract method 0xd5b5cc23.
//
// Solidity: function spent(uint256 ) view returns(bytes32)
func (_Outbox *OutboxCaller) Spent(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Outbox.contract.Call(opts, &out, "spent", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Spent is a free data retrieval call binding the contract method 0xd5b5cc23.
//
// Solidity: function spent(uint256 ) view returns(bytes32)
func (_Outbox *OutboxSession) Spent(arg0 *big.Int) ([32]byte, error) {
	return _Outbox.Contract.Spent(&_Outbox.CallOpts, arg0)
}

// Spent is a free data retrieval call binding the contract method 0xd5b5cc23.
//
// Solidity: function spent(uint256 ) view returns(bytes32)
func (_Outbox *OutboxCallerSession) Spent(arg0 *big.Int) ([32]byte, error) {
	return _Outbox.Contract.Spent(&_Outbox.CallOpts, arg0)
}

// ExecuteTransaction is a paid mutator transaction binding the contract method 0x08635a95.
//
// Solidity: function executeTransaction(bytes32[] proof, uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_Outbox *OutboxTransactor) ExecuteTransaction(opts *bind.TransactOpts, proof [][32]byte, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Outbox.contract.Transact(opts, "executeTransaction", proof, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransaction is a paid mutator transaction binding the contract method 0x08635a95.
//
// Solidity: function executeTransaction(bytes32[] proof, uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_Outbox *OutboxSession) ExecuteTransaction(proof [][32]byte, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Outbox.Contract.ExecuteTransaction(&_Outbox.TransactOpts, proof, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransaction is a paid mutator transaction binding the contract method 0x08635a95.
//
// Solidity: function executeTransaction(bytes32[] proof, uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_Outbox *OutboxTransactorSession) ExecuteTransaction(proof [][32]byte, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Outbox.Contract.ExecuteTransaction(&_Outbox.TransactOpts, proof, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransactionSimulation is a paid mutator transaction binding the contract method 0x288e5b10.
//
// Solidity: function executeTransactionSimulation(uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_Outbox *OutboxTransactor) ExecuteTransactionSimulation(opts *bind.TransactOpts, index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Outbox.contract.Transact(opts, "executeTransactionSimulation", index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransactionSimulation is a paid mutator transaction binding the contract method 0x288e5b10.
//
// Solidity: function executeTransactionSimulation(uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_Outbox *OutboxSession) ExecuteTransactionSimulation(index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Outbox.Contract.ExecuteTransactionSimulation(&_Outbox.TransactOpts, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// ExecuteTransactionSimulation is a paid mutator transaction binding the contract method 0x288e5b10.
//
// Solidity: function executeTransactionSimulation(uint256 index, address l2Sender, address to, uint256 l2Block, uint256 l1Block, uint256 l2Timestamp, uint256 value, bytes data) returns()
func (_Outbox *OutboxTransactorSession) ExecuteTransactionSimulation(index *big.Int, l2Sender common.Address, to common.Address, l2Block *big.Int, l1Block *big.Int, l2Timestamp *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Outbox.Contract.ExecuteTransactionSimulation(&_Outbox.TransactOpts, index, l2Sender, to, l2Block, l1Block, l2Timestamp, value, data)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_Outbox *OutboxTransactor) Initialize(opts *bind.TransactOpts, _bridge common.Address) (*types.Transaction, error) {
	return _Outbox.contract.Transact(opts, "initialize", _bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_Outbox *OutboxSession) Initialize(_bridge common.Address) (*types.Transaction, error) {
	return _Outbox.Contract.Initialize(&_Outbox.TransactOpts, _bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_Outbox *OutboxTransactorSession) Initialize(_bridge common.Address) (*types.Transaction, error) {
	return _Outbox.Contract.Initialize(&_Outbox.TransactOpts, _bridge)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_Outbox *OutboxTransactor) UpdateRollupAddress(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Outbox.contract.Transact(opts, "updateRollupAddress")
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_Outbox *OutboxSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _Outbox.Contract.UpdateRollupAddress(&_Outbox.TransactOpts)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_Outbox *OutboxTransactorSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _Outbox.Contract.UpdateRollupAddress(&_Outbox.TransactOpts)
}

// UpdateSendRoot is a paid mutator transaction binding the contract method 0xa04cee60.
//
// Solidity: function updateSendRoot(bytes32 root, bytes32 l2BlockHash) returns()
func (_Outbox *OutboxTransactor) UpdateSendRoot(opts *bind.TransactOpts, root [32]byte, l2BlockHash [32]byte) (*types.Transaction, error) {
	return _Outbox.contract.Transact(opts, "updateSendRoot", root, l2BlockHash)
}

// UpdateSendRoot is a paid mutator transaction binding the contract method 0xa04cee60.
//
// Solidity: function updateSendRoot(bytes32 root, bytes32 l2BlockHash) returns()
func (_Outbox *OutboxSession) UpdateSendRoot(root [32]byte, l2BlockHash [32]byte) (*types.Transaction, error) {
	return _Outbox.Contract.UpdateSendRoot(&_Outbox.TransactOpts, root, l2BlockHash)
}

// UpdateSendRoot is a paid mutator transaction binding the contract method 0xa04cee60.
//
// Solidity: function updateSendRoot(bytes32 root, bytes32 l2BlockHash) returns()
func (_Outbox *OutboxTransactorSession) UpdateSendRoot(root [32]byte, l2BlockHash [32]byte) (*types.Transaction, error) {
	return _Outbox.Contract.UpdateSendRoot(&_Outbox.TransactOpts, root, l2BlockHash)
}

// OutboxOutBoxTransactionExecutedIterator is returned from FilterOutBoxTransactionExecuted and is used to iterate over the raw logs and unpacked data for OutBoxTransactionExecuted events raised by the Outbox contract.
type OutboxOutBoxTransactionExecutedIterator struct {
	Event *OutboxOutBoxTransactionExecuted // Event containing the contract specifics and raw log

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
func (it *OutboxOutBoxTransactionExecutedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OutboxOutBoxTransactionExecuted)
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
		it.Event = new(OutboxOutBoxTransactionExecuted)
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
func (it *OutboxOutBoxTransactionExecutedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OutboxOutBoxTransactionExecutedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OutboxOutBoxTransactionExecuted represents a OutBoxTransactionExecuted event raised by the Outbox contract.
type OutboxOutBoxTransactionExecuted struct {
	To               common.Address
	L2Sender         common.Address
	Zero             *big.Int
	TransactionIndex *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterOutBoxTransactionExecuted is a free log retrieval operation binding the contract event 0x20af7f3bbfe38132b8900ae295cd9c8d1914be7052d061a511f3f728dab18964.
//
// Solidity: event OutBoxTransactionExecuted(address indexed to, address indexed l2Sender, uint256 indexed zero, uint256 transactionIndex)
func (_Outbox *OutboxFilterer) FilterOutBoxTransactionExecuted(opts *bind.FilterOpts, to []common.Address, l2Sender []common.Address, zero []*big.Int) (*OutboxOutBoxTransactionExecutedIterator, error) {

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

	logs, sub, err := _Outbox.contract.FilterLogs(opts, "OutBoxTransactionExecuted", toRule, l2SenderRule, zeroRule)
	if err != nil {
		return nil, err
	}
	return &OutboxOutBoxTransactionExecutedIterator{contract: _Outbox.contract, event: "OutBoxTransactionExecuted", logs: logs, sub: sub}, nil
}

// WatchOutBoxTransactionExecuted is a free log subscription operation binding the contract event 0x20af7f3bbfe38132b8900ae295cd9c8d1914be7052d061a511f3f728dab18964.
//
// Solidity: event OutBoxTransactionExecuted(address indexed to, address indexed l2Sender, uint256 indexed zero, uint256 transactionIndex)
func (_Outbox *OutboxFilterer) WatchOutBoxTransactionExecuted(opts *bind.WatchOpts, sink chan<- *OutboxOutBoxTransactionExecuted, to []common.Address, l2Sender []common.Address, zero []*big.Int) (event.Subscription, error) {

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

	logs, sub, err := _Outbox.contract.WatchLogs(opts, "OutBoxTransactionExecuted", toRule, l2SenderRule, zeroRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OutboxOutBoxTransactionExecuted)
				if err := _Outbox.contract.UnpackLog(event, "OutBoxTransactionExecuted", log); err != nil {
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
func (_Outbox *OutboxFilterer) ParseOutBoxTransactionExecuted(log types.Log) (*OutboxOutBoxTransactionExecuted, error) {
	event := new(OutboxOutBoxTransactionExecuted)
	if err := _Outbox.contract.UnpackLog(event, "OutBoxTransactionExecuted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OutboxSendRootUpdatedIterator is returned from FilterSendRootUpdated and is used to iterate over the raw logs and unpacked data for SendRootUpdated events raised by the Outbox contract.
type OutboxSendRootUpdatedIterator struct {
	Event *OutboxSendRootUpdated // Event containing the contract specifics and raw log

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
func (it *OutboxSendRootUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OutboxSendRootUpdated)
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
		it.Event = new(OutboxSendRootUpdated)
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
func (it *OutboxSendRootUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OutboxSendRootUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OutboxSendRootUpdated represents a SendRootUpdated event raised by the Outbox contract.
type OutboxSendRootUpdated struct {
	OutputRoot  [32]byte
	L2BlockHash [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSendRootUpdated is a free log retrieval operation binding the contract event 0xb4df3847300f076a369cd76d2314b470a1194d9e8a6bb97f1860aee88a5f6748.
//
// Solidity: event SendRootUpdated(bytes32 indexed outputRoot, bytes32 indexed l2BlockHash)
func (_Outbox *OutboxFilterer) FilterSendRootUpdated(opts *bind.FilterOpts, outputRoot [][32]byte, l2BlockHash [][32]byte) (*OutboxSendRootUpdatedIterator, error) {

	var outputRootRule []interface{}
	for _, outputRootItem := range outputRoot {
		outputRootRule = append(outputRootRule, outputRootItem)
	}
	var l2BlockHashRule []interface{}
	for _, l2BlockHashItem := range l2BlockHash {
		l2BlockHashRule = append(l2BlockHashRule, l2BlockHashItem)
	}

	logs, sub, err := _Outbox.contract.FilterLogs(opts, "SendRootUpdated", outputRootRule, l2BlockHashRule)
	if err != nil {
		return nil, err
	}
	return &OutboxSendRootUpdatedIterator{contract: _Outbox.contract, event: "SendRootUpdated", logs: logs, sub: sub}, nil
}

// WatchSendRootUpdated is a free log subscription operation binding the contract event 0xb4df3847300f076a369cd76d2314b470a1194d9e8a6bb97f1860aee88a5f6748.
//
// Solidity: event SendRootUpdated(bytes32 indexed outputRoot, bytes32 indexed l2BlockHash)
func (_Outbox *OutboxFilterer) WatchSendRootUpdated(opts *bind.WatchOpts, sink chan<- *OutboxSendRootUpdated, outputRoot [][32]byte, l2BlockHash [][32]byte) (event.Subscription, error) {

	var outputRootRule []interface{}
	for _, outputRootItem := range outputRoot {
		outputRootRule = append(outputRootRule, outputRootItem)
	}
	var l2BlockHashRule []interface{}
	for _, l2BlockHashItem := range l2BlockHash {
		l2BlockHashRule = append(l2BlockHashRule, l2BlockHashItem)
	}

	logs, sub, err := _Outbox.contract.WatchLogs(opts, "SendRootUpdated", outputRootRule, l2BlockHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OutboxSendRootUpdated)
				if err := _Outbox.contract.UnpackLog(event, "SendRootUpdated", log); err != nil {
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
func (_Outbox *OutboxFilterer) ParseSendRootUpdated(log types.Log) (*OutboxSendRootUpdated, error) {
	event := new(OutboxSendRootUpdated)
	if err := _Outbox.contract.UnpackLog(event, "SendRootUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxMetaData contains all meta data concerning the SequencerInbox contract.
var SequencerInboxMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"AlreadyInit\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"AlreadyValidDASKeyset\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"stored\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"received\",\"type\":\"uint256\"}],\"name\":\"BadSequencerNumber\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DataNotAuthenticated\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"dataLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxDataLength\",\"type\":\"uint256\"}],\"name\":\"DataTooLarge\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelayedBackwards\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DelayedTooFar\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ForceIncludeBlockTooSoon\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ForceIncludeTimeTooSoon\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"HadZeroInit\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"IncorrectMessagePreimage\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"NoSuchKeyset\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotBatchPoster\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotForked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotOrigin\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"NotOwner\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"InboxMessageDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNum\",\"type\":\"uint256\"}],\"name\":\"InboxMessageDeliveredFromOrigin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keysetHash\",\"type\":\"bytes32\"}],\"name\":\"InvalidateKeyset\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"OwnerFunctionCalled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchSequenceNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"SequencerBatchData\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchSequenceNumber\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"beforeAcc\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"afterAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"delayedAcc\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"minTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"minBlockNumber\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxBlockNumber\",\"type\":\"uint64\"}],\"indexed\":false,\"internalType\":\"structISequencerInbox.TimeBounds\",\"name\":\"timeBounds\",\"type\":\"tuple\"},{\"indexed\":false,\"internalType\":\"enumISequencerInbox.BatchDataLocation\",\"name\":\"dataLocation\",\"type\":\"uint8\"}],\"name\":\"SequencerBatchDelivered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"keysetHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"SetValidKeyset\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DATA_AUTHENTICATED_FLAG\",\"outputs\":[{\"internalType\":\"bytes1\",\"name\":\"\",\"type\":\"bytes1\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"HEADER_LENGTH\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2Batch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"}],\"name\":\"addSequencerL2BatchFromOrigin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sequenceNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"afterDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIGasRefunder\",\"name\":\"gasRefunder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"prevMessageCount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"newMessageCount\",\"type\":\"uint256\"}],\"name\":\"addSequencerL2BatchFromOrigin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batchCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"dasKeySetInfo\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"isValidKeyset\",\"type\":\"bool\"},{\"internalType\":\"uint64\",\"name\":\"creationBlock\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalDelayedMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"kind\",\"type\":\"uint8\"},{\"internalType\":\"uint64[2]\",\"name\":\"l1BlockAndTime\",\"type\":\"uint64[2]\"},{\"internalType\":\"uint256\",\"name\":\"baseFeeL1\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"messageDataHash\",\"type\":\"bytes32\"}],\"name\":\"forceInclusion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"getKeysetCreationBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"inboxAccs\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIBridge\",\"name\":\"bridge_\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"invalidateKeysetHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isBatchPoster\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isSequencer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"ksHash\",\"type\":\"bytes32\"}],\"name\":\"isValidKeysetHash\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxTimeVariation\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"removeDelayAfterFork\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"contractIOwnable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isBatchPoster_\",\"type\":\"bool\"}],\"name\":\"setIsBatchPoster\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isSequencer_\",\"type\":\"bool\"}],\"name\":\"setIsSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delayBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"delaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"futureSeconds\",\"type\":\"uint256\"}],\"internalType\":\"structISequencerInbox.MaxTimeVariation\",\"name\":\"maxTimeVariation_\",\"type\":\"tuple\"}],\"name\":\"setMaxTimeVariation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"setValidKeyset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalDelayedMessagesRead\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"updateRollupAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60e0604052306080524660a052620000226200003b602090811b62001c6517901c565b151560c0523480156200003457600080fd5b5062000109565b60408051600481526024810182526020810180516001600160e01b03166302881c7960e11b1790529051600091829182916064916200007b9190620000d8565b600060405180830381855afa9150503d8060008114620000b8576040519150601f19603f3d011682016040523d82523d6000602084013e620000bd565b606091505b5091509150818015620000d1575080516020145b9250505090565b6000825160005b81811015620000fb5760208186018101518583015201620000df565b506000920191825250919050565b60805160a05160c051612cc5620001476000396000818161148e01526120e2015260006111170152600081816104f701526108860152612cc56000f3fe608060405234801561001057600080fd5b50600436106101a35760003560e01c80637fa3a40e116100ee578063d1ce8da811610097578063e5a358c811610071578063e5a358c8146103af578063e78cea9214610407578063ebea461d1461041a578063f19815781461045057600080fd5b8063d1ce8da814610376578063d9dd67ab14610389578063e0bc97291461039c57600080fd5b806396cc5c78116100c857806396cc5c7814610330578063b31761f814610338578063cb23bcb51461034b57600080fd5b80637fa3a40e14610301578063844208601461030a5780638f111f3c1461031d57600080fd5b80636ae71f12116101505780636f12b0c91161012a5780636f12b0c914610277578063715ea34b1461028a57806371c3e6fe146102de57600080fd5b80636ae71f12146102395780636d46e987146102415780636e7df3e71461026457600080fd5b80631f956632116101815780631f9566321461020b578063258f04951461021e57806327957a491461023157600080fd5b806306f13056146101a85780631637be48146101c35780631f7a92b2146101f6575b600080fd5b6101b0610463565b6040519081526020015b60405180910390f35b6101e66101d136600461257d565b60009081526008602052604090205460ff1690565b60405190151581526020016101ba565b6102096102043660046125ae565b6104ed565b005b6102096102193660046125fd565b610702565b6101b061022c36600461257d565b61080f565b6101b0602881565b61020961087c565b6101e661024f366004612636565b60096020526000908152604090205460ff1681565b6102096102723660046125fd565b610a1f565b6102096102853660046126a3565b610b2c565b6102be61029836600461257d565b60086020526000908152604090205460ff811690610100900467ffffffffffffffff1682565b60408051921515835267ffffffffffffffff9091166020830152016101ba565b6101e66102ec366004612636565b60036020526000908152604090205460ff1681565b6101b060005481565b61020961031836600461257d565b610d63565b61020961032b36600461270e565b610ebd565b610209611114565b61020961034636600461278b565b6111a3565b60025461035e906001600160a01b031681565b6040516001600160a01b0390911681526020016101ba565b6102096103843660046127ff565b6112aa565b6101b061039736600461257d565b61160e565b6102096103aa36600461270e565b61169b565b6103d67f400000000000000000000000000000000000000000000000000000000000000081565b6040517fff0000000000000000000000000000000000000000000000000000000000000090911681526020016101ba565b60015461035e906001600160a01b031681565b6004546005546006546007546104309392919084565b6040805194855260208501939093529183015260608201526080016101ba565b61020961045e366004612841565b611802565b600154604080517e84120c00000000000000000000000000000000000000000000000000000000815290516000926001600160a01b0316916284120c9160048083019260209291908290030181865afa1580156104c4573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906104e891906128b1565b905090565b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036105905760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084015b60405180910390fd5b6001546001600160a01b0316156105d3576040517fef34ca5c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001600160a01b038216610613576040517f1ad0f74300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038416908117909155604080517fcb23bcb5000000000000000000000000000000000000000000000000000000008152905163cb23bcb5916004808201926020929091908290030181865afa158015610692573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906106b691906128ca565b6002805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03929092169190911790558035600455602081013560055560408101356006556060013560075550565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610755573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061077991906128ca565b6001600160a01b0316336001600160a01b0316146107bf57600254604051631194af8760e11b81523360048201526001600160a01b039091166024820152604401610587565b6001600160a01b038216600090815260096020526040808220805460ff1916841515179055516004917fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e91a25050565b600081815260086020908152604080832081518083019092525460ff811615158252610100900467ffffffffffffffff169181018290529082036108685760405162f20c5d60e01b815260048101849052602401610587565b6020015167ffffffffffffffff1692915050565b6001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016300361091a5760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c00000000000000000000000000000000000000006064820152608401610587565b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038054336001600160a01b0382161461097757604051631194af8760e11b81523360048201526001600160a01b0382166024820152604401610587565b600160009054906101000a90046001600160a01b03166001600160a01b031663cb23bcb56040518163ffffffff1660e01b8152600401602060405180830381865afa1580156109ca573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906109ee91906128ca565b6002805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03929092169190911790555050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610a72573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610a9691906128ca565b6001600160a01b0316336001600160a01b031614610adc57600254604051631194af8760e11b81523360048201526001600160a01b039091166024820152604401610587565b6001600160a01b038216600090815260036020526040808220805460ff1916841515179055516001917fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e91a25050565b8060005a9050333214610b6b576040517ffeb3d07100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526003602052604090205460ff16610b9b57604051632dd9fc9760e01b815260040160405180910390fd5b600080610ba9888888611d2b565b90925090506000808080610bc0868b8d8480611f39565b93509350935093508c8414610bf25760405163ac7411c960e01b815260048101859052602481018e9052604401610587565b80838e7f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548a6000604051610c2d94939291906128e7565b60405180910390a4505050506001600160a01b038416159150610d5a9050573660006020610c5c83601f612972565b610c669190612985565b9050610200610c76600283612a8b565b610c809190612985565b610c8b826006612a9a565b610c959190612972565b610c9f9084612972565b9250333214610cad57600091505b836001600160a01b031663e3db8a49335a610cc89087612ab1565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af1158015610d32573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610d569190612ac4565b5050505b50505050505050565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610db6573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610dda91906128ca565b6001600160a01b0316336001600160a01b031614610e2057600254604051631194af8760e11b81523360048201526001600160a01b039091166024820152604401610587565b60008181526008602052604090205460ff16610e515760405162f20c5d60e01b815260048101829052602401610587565b600081815260086020526040808220805460ff191690555182917f5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a91a26040516003907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a250565b8260005a9050333214610efc576040517ffeb3d07100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b3360009081526003602052604090205460ff16610f2c57604051632dd9fc9760e01b815260040160405180910390fd5b600080610f3a8a8a8a611d2b565b90925090508a81838b8b8a8a6000808080610f5889888a8989611f39565b93509350935093508a8414158015610f7257506000198b14155b15610f9a5760405163ac7411c960e01b815260048101859052602481018c9052604401610587565b8083857f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548f6000604051610fd594939291906128e7565b60405180910390a4505050506001600160a01b038b1615985061110997505050505050505057366000602061100b83601f612972565b6110159190612985565b9050610200611025600283612a8b565b61102f9190612985565b61103a826006612a9a565b6110449190612972565b61104e9084612972565b925033321461105c57600091505b836001600160a01b031663e3db8a49335a6110779087612ab1565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e085901b1681526001600160a01b0390921660048301526024820152604481018590526064016020604051808303816000875af11580156110e1573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906111059190612ac4565b5050505b505050505050505050565b467f00000000000000000000000000000000000000000000000000000000000000000361116d576040517fa301bb0600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60408051608081018252600180825260208201819052918101829052606001819052600481905560058190556006819055600755565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156111f6573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061121a91906128ca565b6001600160a01b0316336001600160a01b03161461126057600254604051631194af8760e11b81523360048201526001600160a01b039091166024820152604401610587565b805160045560208101516005556040808201516006556060820151600755516000907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e908290a250565b600260009054906101000a90046001600160a01b03166001600160a01b0316638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156112fd573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061132191906128ca565b6001600160a01b0316336001600160a01b03161461136757600254604051631194af8760e11b81523360048201526001600160a01b039091166024820152604401610587565b60008282604051611379929190612ae1565b6040519081900381207ffe000000000000000000000000000000000000000000000000000000000000006020830152602182015260410160408051601f19818403018152919052805160209091012090507f8000000000000000000000000000000000000000000000000000000000000000811862010000831061143f5760405162461bcd60e51b815260206004820152601360248201527f6b657973657420697320746f6f206c61726765000000000000000000000000006044820152606401610587565b60008181526008602052604090205460ff161561148b576040517ffa2fddda00000000000000000000000000000000000000000000000000000000815260048101829052602401610587565b437f0000000000000000000000000000000000000000000000000000000000000000156115185760646001600160a01b031663a3b1b31d6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156114f1573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061151591906128b1565b90505b6040805180820182526001815267ffffffffffffffff8381166020808401918252600087815260089091528490209251835491517fffffffffffffffffffffffffffffffffffffffffffffff0000000000000000009092169015157fffffffffffffffffffffffffffffffffffffffffffffff0000000000000000ff161761010091909216021790555182907fabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722906115d39088908890612af1565b60405180910390a26040516002907fea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e90600090a25050505050565b6001546040517f16bf5579000000000000000000000000000000000000000000000000000000008152600481018390526000916001600160a01b0316906316bf557990602401602060405180830381865afa158015611671573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061169591906128b1565b92915050565b8260005a3360009081526003602052604090205490915060ff161580156116cd57506002546001600160a01b03163314155b156116eb57604051632dd9fc9760e01b815260040160405180910390fd5b6000806116f98a8a8a611d2b565b909250905060008b82848b8a8a8680806117168787838888611f39565b929c5090945092509050888a1480159061173257506000198914155b1561175a5760405163ac7411c960e01b8152600481018b9052602481018a9052604401610587565b80838b7f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548d600160405161179594939291906128e7565b60405180910390a4505050505050505050807ffe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c208c8c6040516117d8929190612af1565b60405180910390a25050506001600160a01b0382161561110957366000602061100b83601f612972565b600054861161183d576040517f7d73e6fa00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600061192286846118516020890189612b36565b61186160408a0160208b01612b36565b61186c60018d612ab1565b6040805160f89690961b7fff000000000000000000000000000000000000000000000000000000000000001660208088019190915260609590951b6bffffffffffffffffffffffff1916602187015260c093841b7fffffffffffffffff00000000000000000000000000000000000000000000000090811660358801529290931b909116603d85015260458401526065830188905260858084018790528151808503909101815260a59093019052815191012090565b60045490915043906119376020880188612b36565b67ffffffffffffffff1661194b9190612972565b10611982576040517fad3515d900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60065442906119976040880160208901612b36565b67ffffffffffffffff166119ab9190612972565b106119e2576040517fc76d17e500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006001881115611a6b576001546001600160a01b031663d5719dc2611a0960028b612ab1565b6040518263ffffffff1660e01b8152600401611a2791815260200190565b602060405180830381865afa158015611a44573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611a6891906128b1565b90505b60408051602080820184905281830185905282518083038401815260609092019092528051910120600180546001600160a01b03169063d5719dc290611ab1908c612ab1565b6040518263ffffffff1660e01b8152600401611acf91815260200190565b602060405180830381865afa158015611aec573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611b1091906128b1565b14611b47576040517f13947fd700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600080611b538a612390565b9150915060008a90506000600160009054906101000a90046001600160a01b03166001600160a01b0316635fca4a166040518163ffffffff1660e01b8152600401602060405180830381865afa158015611bb1573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611bd591906128b1565b9050600080548d83611be79190612972565b611bf19190612ab1565b9050600080600080611c07898860008989611f39565b93509350935093508083857f7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7856000548d6002604051611c4a94939291906128e7565b60405180910390a45050505050505050505050505050505050565b60408051600481526024810182526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f051038f200000000000000000000000000000000000000000000000000000000179052905160009182918291606491611cd19190612b84565b600060405180830381855afa9150503d8060008114611d0c576040519150601f19603f3d011682016040523d82523d6000602084013e611d11565b606091505b5091509150818015611d24575080516020145b9250505090565b604080516080810182526000808252602082018190529181018290526060810182905284846000611d5d826028612972565b90506201cccc811115611da8576040517f4634691b000000000000000000000000000000000000000000000000000000008152600481018290526201cccc6024820152604401610587565b8115801590611e1457507f4000000000000000000000000000000000000000000000000000000000000000808484600081611de557611de5612b20565b9050013560f81c60f81b167effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916145b15611e4b576040517f1f97007f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60218210801590611e95575082826000818110611e6a57611e6a612b20565b909101357f800000000000000000000000000000000000000000000000000000000000000016151590505b15611ee9576000611eaa602160018587612ba0565b611eb391612bca565b60008181526008602052604090205490915060ff16611ee75760405162f20c5d60e01b815260048101829052602401610587565b505b600080611ef5886123d5565b915091506000828b8b604051602001611f1093929190612be8565b60408051808303601f1901815291905280516020909101209b919a509098505050505050505050565b600080600080600054881015611f7b576040517f7d73e6fa00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600160009054906101000a90046001600160a01b03166001600160a01b031663eca067ad6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611fce573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611ff291906128b1565b88111561202b576040517f925f8bd300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001546040517f86598a56000000000000000000000000000000000000000000000000000000008152600481018b9052602481018a905260448101889052606481018790526001600160a01b03909116906386598a56906084016080604051808303816000875af11580156120a4573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906120c89190612c10565b60008c9055929650909450925090508615612384573360607f000000000000000000000000000000000000000000000000000000000000000015612250576000606c6001600160a01b031663c6f7de0e6040518163ffffffff1660e01b8152600401602060405180830381865afa158015612147573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061216b91906128b1565b905060006121794883612985565b905067ffffffffffffffff8111156121d35760405162461bcd60e51b815260206004820152601160248201527f4c315f4741535f4e4f545f55494e5436340000000000000000000000000000006044820152606401610587565b60408051426020820152606086901b6bffffffffffffffffffffffff191681830152605481018f9052607481018a905248609482015260c09290921b7fffffffffffffffff0000000000000000000000000000000000000000000000001660b48301528051609c81840301815260bc9092019052915061229e9050565b604080514260208201526bffffffffffffffffffffffff19606085901b1691810191909152605481018c90526074810187905248609482015260b40160405160208183030381529060405290505b600154815160208301206040517f7a88b1070000000000000000000000000000000000000000000000000000000081526000926001600160a01b031691637a88b107916123039187916004016001600160a01b03929092168252602082015260400190565b6020604051808303816000875af1158015612322573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061234691906128b1565b9050807fff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b836040516123789190612c46565b60405180910390a25050505b95509550955095915050565b60408051608081018252600080825260208201819052918101829052606081018290526000806123bf856123d5565b8151602090920191909120969095509350505050565b604080516080810182526000808252602082018190529181018290526060808201839052916124026124ad565b9050600081600001518260200151836040015184606001518860405160200161248295949392919060c095861b7fffffffffffffffff000000000000000000000000000000000000000000000000908116825294861b8516600882015292851b8416601084015290841b8316601883015290921b16602082015260280190565b604051602081830303815290604052905060288151146124a4576124a4612c79565b94909350915050565b6040805160808101825260008082526020820181905291810182905260608101919091526040805160808101825260008082526020820181905291810182905260608101919091526006544211156125195760065461250c9042612ab1565b67ffffffffffffffff1681525b6007546125269042612972565b67ffffffffffffffff16602082015260045443111561255c5760045461254c9043612ab1565b67ffffffffffffffff1660408201525b6005546125699043612972565b67ffffffffffffffff166060820152919050565b60006020828403121561258f57600080fd5b5035919050565b6001600160a01b03811681146125ab57600080fd5b50565b60008082840360a08112156125c257600080fd5b83356125cd81612596565b92506080601f19820112156125e157600080fd5b506020830190509250929050565b80151581146125ab57600080fd5b6000806040838503121561261057600080fd5b823561261b81612596565b9150602083013561262b816125ef565b809150509250929050565b60006020828403121561264857600080fd5b813561265381612596565b9392505050565b60008083601f84011261266c57600080fd5b50813567ffffffffffffffff81111561268457600080fd5b60208301915083602082850101111561269c57600080fd5b9250929050565b6000806000806000608086880312156126bb57600080fd5b85359450602086013567ffffffffffffffff8111156126d957600080fd5b6126e58882890161265a565b90955093505060408601359150606086013561270081612596565b809150509295509295909350565b600080600080600080600060c0888a03121561272957600080fd5b87359650602088013567ffffffffffffffff81111561274757600080fd5b6127538a828b0161265a565b90975095505060408801359350606088013561276e81612596565b969995985093969295946080840135945060a09093013592915050565b60006080828403121561279d57600080fd5b6040516080810181811067ffffffffffffffff821117156127ce57634e487b7160e01b600052604160045260246000fd5b8060405250823581526020830135602082015260408301356040820152606083013560608201528091505092915050565b6000806020838503121561281257600080fd5b823567ffffffffffffffff81111561282957600080fd5b6128358582860161265a565b90969095509350505050565b60008060008060008060e0878903121561285a57600080fd5b86359550602087013560ff8116811461287257600080fd5b9450608087018881111561288557600080fd5b60408801945035925060a087013561289c81612596565b8092505060c087013590509295509295509295565b6000602082840312156128c357600080fd5b5051919050565b6000602082840312156128dc57600080fd5b815161265381612596565b600060e08201905085825284602083015267ffffffffffffffff8085511660408401528060208601511660608401528060408601511660808401528060608601511660a0840152506003831061294d57634e487b7160e01b600052602160045260246000fd5b8260c083015295945050505050565b634e487b7160e01b600052601160045260246000fd5b808201808211156116955761169561295c565b6000826129a257634e487b7160e01b600052601260045260246000fd5b500490565b600181815b808511156129e25781600019048211156129c8576129c861295c565b808516156129d557918102915b93841c93908002906129ac565b509250929050565b6000826129f957506001611695565b81612a0657506000611695565b8160018114612a1c5760028114612a2657612a42565b6001915050611695565b60ff841115612a3757612a3761295c565b50506001821b611695565b5060208310610133831016604e8410600b8410161715612a65575081810a611695565b612a6f83836129a7565b8060001904821115612a8357612a8361295c565b029392505050565b600061265360ff8416836129ea565b80820281158282048414176116955761169561295c565b818103818111156116955761169561295c565b600060208284031215612ad657600080fd5b8151612653816125ef565b8183823760009101908152919050565b60208152816020820152818360408301376000818301604090810191909152601f909201601f19160101919050565b634e487b7160e01b600052603260045260246000fd5b600060208284031215612b4857600080fd5b813567ffffffffffffffff8116811461265357600080fd5b60005b83811015612b7b578181015183820152602001612b63565b50506000910152565b60008251612b96818460208701612b60565b9190910192915050565b60008085851115612bb057600080fd5b83861115612bbd57600080fd5b5050820193919092039150565b8035602083101561169557600019602084900360031b1b1692915050565b60008451612bfa818460208901612b60565b8201838582376000930192835250909392505050565b60008060008060808587031215612c2657600080fd5b505082516020840151604085015160609095015191969095509092509050565b6020815260008251806020840152612c65816040850160208701612b60565b601f01601f19169190910160400192915050565b634e487b7160e01b600052600160045260246000fdfea264697066735822122020aeb41661ac5beefc3edbfe3afaf75b6e0b32bb01d2ae7e0abc3870e37d712c64736f6c63430008110033",
}

// SequencerInboxABI is the input ABI used to generate the binding from.
// Deprecated: Use SequencerInboxMetaData.ABI instead.
var SequencerInboxABI = SequencerInboxMetaData.ABI

// SequencerInboxBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SequencerInboxMetaData.Bin instead.
var SequencerInboxBin = SequencerInboxMetaData.Bin

// DeploySequencerInbox deploys a new Ethereum contract, binding an instance of SequencerInbox to it.
func DeploySequencerInbox(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SequencerInbox, error) {
	parsed, err := SequencerInboxMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SequencerInboxBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SequencerInbox{SequencerInboxCaller: SequencerInboxCaller{contract: contract}, SequencerInboxTransactor: SequencerInboxTransactor{contract: contract}, SequencerInboxFilterer: SequencerInboxFilterer{contract: contract}}, nil
}

// SequencerInbox is an auto generated Go binding around an Ethereum contract.
type SequencerInbox struct {
	SequencerInboxCaller     // Read-only binding to the contract
	SequencerInboxTransactor // Write-only binding to the contract
	SequencerInboxFilterer   // Log filterer for contract events
}

// SequencerInboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type SequencerInboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerInboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SequencerInboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerInboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SequencerInboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerInboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SequencerInboxSession struct {
	Contract     *SequencerInbox   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SequencerInboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SequencerInboxCallerSession struct {
	Contract *SequencerInboxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// SequencerInboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SequencerInboxTransactorSession struct {
	Contract     *SequencerInboxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// SequencerInboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type SequencerInboxRaw struct {
	Contract *SequencerInbox // Generic contract binding to access the raw methods on
}

// SequencerInboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SequencerInboxCallerRaw struct {
	Contract *SequencerInboxCaller // Generic read-only contract binding to access the raw methods on
}

// SequencerInboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SequencerInboxTransactorRaw struct {
	Contract *SequencerInboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSequencerInbox creates a new instance of SequencerInbox, bound to a specific deployed contract.
func NewSequencerInbox(address common.Address, backend bind.ContractBackend) (*SequencerInbox, error) {
	contract, err := bindSequencerInbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SequencerInbox{SequencerInboxCaller: SequencerInboxCaller{contract: contract}, SequencerInboxTransactor: SequencerInboxTransactor{contract: contract}, SequencerInboxFilterer: SequencerInboxFilterer{contract: contract}}, nil
}

// NewSequencerInboxCaller creates a new read-only instance of SequencerInbox, bound to a specific deployed contract.
func NewSequencerInboxCaller(address common.Address, caller bind.ContractCaller) (*SequencerInboxCaller, error) {
	contract, err := bindSequencerInbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxCaller{contract: contract}, nil
}

// NewSequencerInboxTransactor creates a new write-only instance of SequencerInbox, bound to a specific deployed contract.
func NewSequencerInboxTransactor(address common.Address, transactor bind.ContractTransactor) (*SequencerInboxTransactor, error) {
	contract, err := bindSequencerInbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxTransactor{contract: contract}, nil
}

// NewSequencerInboxFilterer creates a new log filterer instance of SequencerInbox, bound to a specific deployed contract.
func NewSequencerInboxFilterer(address common.Address, filterer bind.ContractFilterer) (*SequencerInboxFilterer, error) {
	contract, err := bindSequencerInbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxFilterer{contract: contract}, nil
}

// bindSequencerInbox binds a generic wrapper to an already deployed contract.
func bindSequencerInbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SequencerInboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SequencerInbox *SequencerInboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SequencerInbox.Contract.SequencerInboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SequencerInbox *SequencerInboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInbox.Contract.SequencerInboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SequencerInbox *SequencerInboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SequencerInbox.Contract.SequencerInboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SequencerInbox *SequencerInboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SequencerInbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SequencerInbox *SequencerInboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SequencerInbox *SequencerInboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SequencerInbox.Contract.contract.Transact(opts, method, params...)
}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_SequencerInbox *SequencerInboxCaller) DATAAUTHENTICATEDFLAG(opts *bind.CallOpts) ([1]byte, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "DATA_AUTHENTICATED_FLAG")

	if err != nil {
		return *new([1]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)

	return out0, err

}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_SequencerInbox *SequencerInboxSession) DATAAUTHENTICATEDFLAG() ([1]byte, error) {
	return _SequencerInbox.Contract.DATAAUTHENTICATEDFLAG(&_SequencerInbox.CallOpts)
}

// DATAAUTHENTICATEDFLAG is a free data retrieval call binding the contract method 0xe5a358c8.
//
// Solidity: function DATA_AUTHENTICATED_FLAG() view returns(bytes1)
func (_SequencerInbox *SequencerInboxCallerSession) DATAAUTHENTICATEDFLAG() ([1]byte, error) {
	return _SequencerInbox.Contract.DATAAUTHENTICATEDFLAG(&_SequencerInbox.CallOpts)
}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_SequencerInbox *SequencerInboxCaller) HEADERLENGTH(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "HEADER_LENGTH")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_SequencerInbox *SequencerInboxSession) HEADERLENGTH() (*big.Int, error) {
	return _SequencerInbox.Contract.HEADERLENGTH(&_SequencerInbox.CallOpts)
}

// HEADERLENGTH is a free data retrieval call binding the contract method 0x27957a49.
//
// Solidity: function HEADER_LENGTH() view returns(uint256)
func (_SequencerInbox *SequencerInboxCallerSession) HEADERLENGTH() (*big.Int, error) {
	return _SequencerInbox.Contract.HEADERLENGTH(&_SequencerInbox.CallOpts)
}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_SequencerInbox *SequencerInboxCaller) BatchCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "batchCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_SequencerInbox *SequencerInboxSession) BatchCount() (*big.Int, error) {
	return _SequencerInbox.Contract.BatchCount(&_SequencerInbox.CallOpts)
}

// BatchCount is a free data retrieval call binding the contract method 0x06f13056.
//
// Solidity: function batchCount() view returns(uint256)
func (_SequencerInbox *SequencerInboxCallerSession) BatchCount() (*big.Int, error) {
	return _SequencerInbox.Contract.BatchCount(&_SequencerInbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SequencerInbox *SequencerInboxCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SequencerInbox *SequencerInboxSession) Bridge() (common.Address, error) {
	return _SequencerInbox.Contract.Bridge(&_SequencerInbox.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_SequencerInbox *SequencerInboxCallerSession) Bridge() (common.Address, error) {
	return _SequencerInbox.Contract.Bridge(&_SequencerInbox.CallOpts)
}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool isValidKeyset, uint64 creationBlock)
func (_SequencerInbox *SequencerInboxCaller) DasKeySetInfo(opts *bind.CallOpts, arg0 [32]byte) (struct {
	IsValidKeyset bool
	CreationBlock uint64
}, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "dasKeySetInfo", arg0)

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
func (_SequencerInbox *SequencerInboxSession) DasKeySetInfo(arg0 [32]byte) (struct {
	IsValidKeyset bool
	CreationBlock uint64
}, error) {
	return _SequencerInbox.Contract.DasKeySetInfo(&_SequencerInbox.CallOpts, arg0)
}

// DasKeySetInfo is a free data retrieval call binding the contract method 0x715ea34b.
//
// Solidity: function dasKeySetInfo(bytes32 ) view returns(bool isValidKeyset, uint64 creationBlock)
func (_SequencerInbox *SequencerInboxCallerSession) DasKeySetInfo(arg0 [32]byte) (struct {
	IsValidKeyset bool
	CreationBlock uint64
}, error) {
	return _SequencerInbox.Contract.DasKeySetInfo(&_SequencerInbox.CallOpts, arg0)
}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_SequencerInbox *SequencerInboxCaller) GetKeysetCreationBlock(opts *bind.CallOpts, ksHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "getKeysetCreationBlock", ksHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_SequencerInbox *SequencerInboxSession) GetKeysetCreationBlock(ksHash [32]byte) (*big.Int, error) {
	return _SequencerInbox.Contract.GetKeysetCreationBlock(&_SequencerInbox.CallOpts, ksHash)
}

// GetKeysetCreationBlock is a free data retrieval call binding the contract method 0x258f0495.
//
// Solidity: function getKeysetCreationBlock(bytes32 ksHash) view returns(uint256)
func (_SequencerInbox *SequencerInboxCallerSession) GetKeysetCreationBlock(ksHash [32]byte) (*big.Int, error) {
	return _SequencerInbox.Contract.GetKeysetCreationBlock(&_SequencerInbox.CallOpts, ksHash)
}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_SequencerInbox *SequencerInboxCaller) InboxAccs(opts *bind.CallOpts, index *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "inboxAccs", index)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_SequencerInbox *SequencerInboxSession) InboxAccs(index *big.Int) ([32]byte, error) {
	return _SequencerInbox.Contract.InboxAccs(&_SequencerInbox.CallOpts, index)
}

// InboxAccs is a free data retrieval call binding the contract method 0xd9dd67ab.
//
// Solidity: function inboxAccs(uint256 index) view returns(bytes32)
func (_SequencerInbox *SequencerInboxCallerSession) InboxAccs(index *big.Int) ([32]byte, error) {
	return _SequencerInbox.Contract.InboxAccs(&_SequencerInbox.CallOpts, index)
}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_SequencerInbox *SequencerInboxCaller) IsBatchPoster(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "isBatchPoster", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_SequencerInbox *SequencerInboxSession) IsBatchPoster(arg0 common.Address) (bool, error) {
	return _SequencerInbox.Contract.IsBatchPoster(&_SequencerInbox.CallOpts, arg0)
}

// IsBatchPoster is a free data retrieval call binding the contract method 0x71c3e6fe.
//
// Solidity: function isBatchPoster(address ) view returns(bool)
func (_SequencerInbox *SequencerInboxCallerSession) IsBatchPoster(arg0 common.Address) (bool, error) {
	return _SequencerInbox.Contract.IsBatchPoster(&_SequencerInbox.CallOpts, arg0)
}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_SequencerInbox *SequencerInboxCaller) IsSequencer(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "isSequencer", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_SequencerInbox *SequencerInboxSession) IsSequencer(arg0 common.Address) (bool, error) {
	return _SequencerInbox.Contract.IsSequencer(&_SequencerInbox.CallOpts, arg0)
}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_SequencerInbox *SequencerInboxCallerSession) IsSequencer(arg0 common.Address) (bool, error) {
	return _SequencerInbox.Contract.IsSequencer(&_SequencerInbox.CallOpts, arg0)
}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_SequencerInbox *SequencerInboxCaller) IsValidKeysetHash(opts *bind.CallOpts, ksHash [32]byte) (bool, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "isValidKeysetHash", ksHash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_SequencerInbox *SequencerInboxSession) IsValidKeysetHash(ksHash [32]byte) (bool, error) {
	return _SequencerInbox.Contract.IsValidKeysetHash(&_SequencerInbox.CallOpts, ksHash)
}

// IsValidKeysetHash is a free data retrieval call binding the contract method 0x1637be48.
//
// Solidity: function isValidKeysetHash(bytes32 ksHash) view returns(bool)
func (_SequencerInbox *SequencerInboxCallerSession) IsValidKeysetHash(ksHash [32]byte) (bool, error) {
	return _SequencerInbox.Contract.IsValidKeysetHash(&_SequencerInbox.CallOpts, ksHash)
}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256 delayBlocks, uint256 futureBlocks, uint256 delaySeconds, uint256 futureSeconds)
func (_SequencerInbox *SequencerInboxCaller) MaxTimeVariation(opts *bind.CallOpts) (struct {
	DelayBlocks   *big.Int
	FutureBlocks  *big.Int
	DelaySeconds  *big.Int
	FutureSeconds *big.Int
}, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "maxTimeVariation")

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
func (_SequencerInbox *SequencerInboxSession) MaxTimeVariation() (struct {
	DelayBlocks   *big.Int
	FutureBlocks  *big.Int
	DelaySeconds  *big.Int
	FutureSeconds *big.Int
}, error) {
	return _SequencerInbox.Contract.MaxTimeVariation(&_SequencerInbox.CallOpts)
}

// MaxTimeVariation is a free data retrieval call binding the contract method 0xebea461d.
//
// Solidity: function maxTimeVariation() view returns(uint256 delayBlocks, uint256 futureBlocks, uint256 delaySeconds, uint256 futureSeconds)
func (_SequencerInbox *SequencerInboxCallerSession) MaxTimeVariation() (struct {
	DelayBlocks   *big.Int
	FutureBlocks  *big.Int
	DelaySeconds  *big.Int
	FutureSeconds *big.Int
}, error) {
	return _SequencerInbox.Contract.MaxTimeVariation(&_SequencerInbox.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_SequencerInbox *SequencerInboxCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_SequencerInbox *SequencerInboxSession) Rollup() (common.Address, error) {
	return _SequencerInbox.Contract.Rollup(&_SequencerInbox.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_SequencerInbox *SequencerInboxCallerSession) Rollup() (common.Address, error) {
	return _SequencerInbox.Contract.Rollup(&_SequencerInbox.CallOpts)
}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_SequencerInbox *SequencerInboxCaller) TotalDelayedMessagesRead(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerInbox.contract.Call(opts, &out, "totalDelayedMessagesRead")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_SequencerInbox *SequencerInboxSession) TotalDelayedMessagesRead() (*big.Int, error) {
	return _SequencerInbox.Contract.TotalDelayedMessagesRead(&_SequencerInbox.CallOpts)
}

// TotalDelayedMessagesRead is a free data retrieval call binding the contract method 0x7fa3a40e.
//
// Solidity: function totalDelayedMessagesRead() view returns(uint256)
func (_SequencerInbox *SequencerInboxCallerSession) TotalDelayedMessagesRead() (*big.Int, error) {
	return _SequencerInbox.Contract.TotalDelayedMessagesRead(&_SequencerInbox.CallOpts)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInbox *SequencerInboxTransactor) AddSequencerL2Batch(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "addSequencerL2Batch", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInbox *SequencerInboxSession) AddSequencerL2Batch(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInbox.Contract.AddSequencerL2Batch(&_SequencerInbox.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2Batch is a paid mutator transaction binding the contract method 0xe0bc9729.
//
// Solidity: function addSequencerL2Batch(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInbox *SequencerInboxTransactorSession) AddSequencerL2Batch(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInbox.Contract.AddSequencerL2Batch(&_SequencerInbox.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromOrigin is a paid mutator transaction binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder) returns()
func (_SequencerInbox *SequencerInboxTransactor) AddSequencerL2BatchFromOrigin(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "addSequencerL2BatchFromOrigin", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder)
}

// AddSequencerL2BatchFromOrigin is a paid mutator transaction binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder) returns()
func (_SequencerInbox *SequencerInboxSession) AddSequencerL2BatchFromOrigin(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address) (*types.Transaction, error) {
	return _SequencerInbox.Contract.AddSequencerL2BatchFromOrigin(&_SequencerInbox.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder)
}

// AddSequencerL2BatchFromOrigin is a paid mutator transaction binding the contract method 0x6f12b0c9.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder) returns()
func (_SequencerInbox *SequencerInboxTransactorSession) AddSequencerL2BatchFromOrigin(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address) (*types.Transaction, error) {
	return _SequencerInbox.Contract.AddSequencerL2BatchFromOrigin(&_SequencerInbox.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder)
}

// AddSequencerL2BatchFromOrigin0 is a paid mutator transaction binding the contract method 0x8f111f3c.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInbox *SequencerInboxTransactor) AddSequencerL2BatchFromOrigin0(opts *bind.TransactOpts, sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "addSequencerL2BatchFromOrigin0", sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromOrigin0 is a paid mutator transaction binding the contract method 0x8f111f3c.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInbox *SequencerInboxSession) AddSequencerL2BatchFromOrigin0(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInbox.Contract.AddSequencerL2BatchFromOrigin0(&_SequencerInbox.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// AddSequencerL2BatchFromOrigin0 is a paid mutator transaction binding the contract method 0x8f111f3c.
//
// Solidity: function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount) returns()
func (_SequencerInbox *SequencerInboxTransactorSession) AddSequencerL2BatchFromOrigin0(sequenceNumber *big.Int, data []byte, afterDelayedMessagesRead *big.Int, gasRefunder common.Address, prevMessageCount *big.Int, newMessageCount *big.Int) (*types.Transaction, error) {
	return _SequencerInbox.Contract.AddSequencerL2BatchFromOrigin0(&_SequencerInbox.TransactOpts, sequenceNumber, data, afterDelayedMessagesRead, gasRefunder, prevMessageCount, newMessageCount)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_SequencerInbox *SequencerInboxTransactor) ForceInclusion(opts *bind.TransactOpts, _totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "forceInclusion", _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_SequencerInbox *SequencerInboxSession) ForceInclusion(_totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _SequencerInbox.Contract.ForceInclusion(&_SequencerInbox.TransactOpts, _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// ForceInclusion is a paid mutator transaction binding the contract method 0xf1981578.
//
// Solidity: function forceInclusion(uint256 _totalDelayedMessagesRead, uint8 kind, uint64[2] l1BlockAndTime, uint256 baseFeeL1, address sender, bytes32 messageDataHash) returns()
func (_SequencerInbox *SequencerInboxTransactorSession) ForceInclusion(_totalDelayedMessagesRead *big.Int, kind uint8, l1BlockAndTime [2]uint64, baseFeeL1 *big.Int, sender common.Address, messageDataHash [32]byte) (*types.Transaction, error) {
	return _SequencerInbox.Contract.ForceInclusion(&_SequencerInbox.TransactOpts, _totalDelayedMessagesRead, kind, l1BlockAndTime, baseFeeL1, sender, messageDataHash)
}

// Initialize is a paid mutator transaction binding the contract method 0x1f7a92b2.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInbox *SequencerInboxTransactor) Initialize(opts *bind.TransactOpts, bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "initialize", bridge_, maxTimeVariation_)
}

// Initialize is a paid mutator transaction binding the contract method 0x1f7a92b2.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInbox *SequencerInboxSession) Initialize(bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInbox.Contract.Initialize(&_SequencerInbox.TransactOpts, bridge_, maxTimeVariation_)
}

// Initialize is a paid mutator transaction binding the contract method 0x1f7a92b2.
//
// Solidity: function initialize(address bridge_, (uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInbox *SequencerInboxTransactorSession) Initialize(bridge_ common.Address, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInbox.Contract.Initialize(&_SequencerInbox.TransactOpts, bridge_, maxTimeVariation_)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_SequencerInbox *SequencerInboxTransactor) InvalidateKeysetHash(opts *bind.TransactOpts, ksHash [32]byte) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "invalidateKeysetHash", ksHash)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_SequencerInbox *SequencerInboxSession) InvalidateKeysetHash(ksHash [32]byte) (*types.Transaction, error) {
	return _SequencerInbox.Contract.InvalidateKeysetHash(&_SequencerInbox.TransactOpts, ksHash)
}

// InvalidateKeysetHash is a paid mutator transaction binding the contract method 0x84420860.
//
// Solidity: function invalidateKeysetHash(bytes32 ksHash) returns()
func (_SequencerInbox *SequencerInboxTransactorSession) InvalidateKeysetHash(ksHash [32]byte) (*types.Transaction, error) {
	return _SequencerInbox.Contract.InvalidateKeysetHash(&_SequencerInbox.TransactOpts, ksHash)
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_SequencerInbox *SequencerInboxTransactor) RemoveDelayAfterFork(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "removeDelayAfterFork")
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_SequencerInbox *SequencerInboxSession) RemoveDelayAfterFork() (*types.Transaction, error) {
	return _SequencerInbox.Contract.RemoveDelayAfterFork(&_SequencerInbox.TransactOpts)
}

// RemoveDelayAfterFork is a paid mutator transaction binding the contract method 0x96cc5c78.
//
// Solidity: function removeDelayAfterFork() returns()
func (_SequencerInbox *SequencerInboxTransactorSession) RemoveDelayAfterFork() (*types.Transaction, error) {
	return _SequencerInbox.Contract.RemoveDelayAfterFork(&_SequencerInbox.TransactOpts)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_SequencerInbox *SequencerInboxTransactor) SetIsBatchPoster(opts *bind.TransactOpts, addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "setIsBatchPoster", addr, isBatchPoster_)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_SequencerInbox *SequencerInboxSession) SetIsBatchPoster(addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _SequencerInbox.Contract.SetIsBatchPoster(&_SequencerInbox.TransactOpts, addr, isBatchPoster_)
}

// SetIsBatchPoster is a paid mutator transaction binding the contract method 0x6e7df3e7.
//
// Solidity: function setIsBatchPoster(address addr, bool isBatchPoster_) returns()
func (_SequencerInbox *SequencerInboxTransactorSession) SetIsBatchPoster(addr common.Address, isBatchPoster_ bool) (*types.Transaction, error) {
	return _SequencerInbox.Contract.SetIsBatchPoster(&_SequencerInbox.TransactOpts, addr, isBatchPoster_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_SequencerInbox *SequencerInboxTransactor) SetIsSequencer(opts *bind.TransactOpts, addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "setIsSequencer", addr, isSequencer_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_SequencerInbox *SequencerInboxSession) SetIsSequencer(addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _SequencerInbox.Contract.SetIsSequencer(&_SequencerInbox.TransactOpts, addr, isSequencer_)
}

// SetIsSequencer is a paid mutator transaction binding the contract method 0x1f956632.
//
// Solidity: function setIsSequencer(address addr, bool isSequencer_) returns()
func (_SequencerInbox *SequencerInboxTransactorSession) SetIsSequencer(addr common.Address, isSequencer_ bool) (*types.Transaction, error) {
	return _SequencerInbox.Contract.SetIsSequencer(&_SequencerInbox.TransactOpts, addr, isSequencer_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInbox *SequencerInboxTransactor) SetMaxTimeVariation(opts *bind.TransactOpts, maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "setMaxTimeVariation", maxTimeVariation_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInbox *SequencerInboxSession) SetMaxTimeVariation(maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInbox.Contract.SetMaxTimeVariation(&_SequencerInbox.TransactOpts, maxTimeVariation_)
}

// SetMaxTimeVariation is a paid mutator transaction binding the contract method 0xb31761f8.
//
// Solidity: function setMaxTimeVariation((uint256,uint256,uint256,uint256) maxTimeVariation_) returns()
func (_SequencerInbox *SequencerInboxTransactorSession) SetMaxTimeVariation(maxTimeVariation_ ISequencerInboxMaxTimeVariation) (*types.Transaction, error) {
	return _SequencerInbox.Contract.SetMaxTimeVariation(&_SequencerInbox.TransactOpts, maxTimeVariation_)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_SequencerInbox *SequencerInboxTransactor) SetValidKeyset(opts *bind.TransactOpts, keysetBytes []byte) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "setValidKeyset", keysetBytes)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_SequencerInbox *SequencerInboxSession) SetValidKeyset(keysetBytes []byte) (*types.Transaction, error) {
	return _SequencerInbox.Contract.SetValidKeyset(&_SequencerInbox.TransactOpts, keysetBytes)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_SequencerInbox *SequencerInboxTransactorSession) SetValidKeyset(keysetBytes []byte) (*types.Transaction, error) {
	return _SequencerInbox.Contract.SetValidKeyset(&_SequencerInbox.TransactOpts, keysetBytes)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_SequencerInbox *SequencerInboxTransactor) UpdateRollupAddress(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerInbox.contract.Transact(opts, "updateRollupAddress")
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_SequencerInbox *SequencerInboxSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _SequencerInbox.Contract.UpdateRollupAddress(&_SequencerInbox.TransactOpts)
}

// UpdateRollupAddress is a paid mutator transaction binding the contract method 0x6ae71f12.
//
// Solidity: function updateRollupAddress() returns()
func (_SequencerInbox *SequencerInboxTransactorSession) UpdateRollupAddress() (*types.Transaction, error) {
	return _SequencerInbox.Contract.UpdateRollupAddress(&_SequencerInbox.TransactOpts)
}

// SequencerInboxInboxMessageDeliveredIterator is returned from FilterInboxMessageDelivered and is used to iterate over the raw logs and unpacked data for InboxMessageDelivered events raised by the SequencerInbox contract.
type SequencerInboxInboxMessageDeliveredIterator struct {
	Event *SequencerInboxInboxMessageDelivered // Event containing the contract specifics and raw log

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
func (it *SequencerInboxInboxMessageDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxInboxMessageDelivered)
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
		it.Event = new(SequencerInboxInboxMessageDelivered)
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
func (it *SequencerInboxInboxMessageDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxInboxMessageDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxInboxMessageDelivered represents a InboxMessageDelivered event raised by the SequencerInbox contract.
type SequencerInboxInboxMessageDelivered struct {
	MessageNum *big.Int
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDelivered is a free log retrieval operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_SequencerInbox *SequencerInboxFilterer) FilterInboxMessageDelivered(opts *bind.FilterOpts, messageNum []*big.Int) (*SequencerInboxInboxMessageDeliveredIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInbox.contract.FilterLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxInboxMessageDeliveredIterator{contract: _SequencerInbox.contract, event: "InboxMessageDelivered", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDelivered is a free log subscription operation binding the contract event 0xff64905f73a67fb594e0f940a8075a860db489ad991e032f48c81123eb52d60b.
//
// Solidity: event InboxMessageDelivered(uint256 indexed messageNum, bytes data)
func (_SequencerInbox *SequencerInboxFilterer) WatchInboxMessageDelivered(opts *bind.WatchOpts, sink chan<- *SequencerInboxInboxMessageDelivered, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInbox.contract.WatchLogs(opts, "InboxMessageDelivered", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxInboxMessageDelivered)
				if err := _SequencerInbox.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
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
func (_SequencerInbox *SequencerInboxFilterer) ParseInboxMessageDelivered(log types.Log) (*SequencerInboxInboxMessageDelivered, error) {
	event := new(SequencerInboxInboxMessageDelivered)
	if err := _SequencerInbox.contract.UnpackLog(event, "InboxMessageDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxInboxMessageDeliveredFromOriginIterator is returned from FilterInboxMessageDeliveredFromOrigin and is used to iterate over the raw logs and unpacked data for InboxMessageDeliveredFromOrigin events raised by the SequencerInbox contract.
type SequencerInboxInboxMessageDeliveredFromOriginIterator struct {
	Event *SequencerInboxInboxMessageDeliveredFromOrigin // Event containing the contract specifics and raw log

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
func (it *SequencerInboxInboxMessageDeliveredFromOriginIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxInboxMessageDeliveredFromOrigin)
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
		it.Event = new(SequencerInboxInboxMessageDeliveredFromOrigin)
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
func (it *SequencerInboxInboxMessageDeliveredFromOriginIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxInboxMessageDeliveredFromOriginIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxInboxMessageDeliveredFromOrigin represents a InboxMessageDeliveredFromOrigin event raised by the SequencerInbox contract.
type SequencerInboxInboxMessageDeliveredFromOrigin struct {
	MessageNum *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInboxMessageDeliveredFromOrigin is a free log retrieval operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_SequencerInbox *SequencerInboxFilterer) FilterInboxMessageDeliveredFromOrigin(opts *bind.FilterOpts, messageNum []*big.Int) (*SequencerInboxInboxMessageDeliveredFromOriginIterator, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInbox.contract.FilterLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxInboxMessageDeliveredFromOriginIterator{contract: _SequencerInbox.contract, event: "InboxMessageDeliveredFromOrigin", logs: logs, sub: sub}, nil
}

// WatchInboxMessageDeliveredFromOrigin is a free log subscription operation binding the contract event 0xab532385be8f1005a4b6ba8fa20a2245facb346134ac739fe9a5198dc1580b9c.
//
// Solidity: event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum)
func (_SequencerInbox *SequencerInboxFilterer) WatchInboxMessageDeliveredFromOrigin(opts *bind.WatchOpts, sink chan<- *SequencerInboxInboxMessageDeliveredFromOrigin, messageNum []*big.Int) (event.Subscription, error) {

	var messageNumRule []interface{}
	for _, messageNumItem := range messageNum {
		messageNumRule = append(messageNumRule, messageNumItem)
	}

	logs, sub, err := _SequencerInbox.contract.WatchLogs(opts, "InboxMessageDeliveredFromOrigin", messageNumRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxInboxMessageDeliveredFromOrigin)
				if err := _SequencerInbox.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
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
func (_SequencerInbox *SequencerInboxFilterer) ParseInboxMessageDeliveredFromOrigin(log types.Log) (*SequencerInboxInboxMessageDeliveredFromOrigin, error) {
	event := new(SequencerInboxInboxMessageDeliveredFromOrigin)
	if err := _SequencerInbox.contract.UnpackLog(event, "InboxMessageDeliveredFromOrigin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxInvalidateKeysetIterator is returned from FilterInvalidateKeyset and is used to iterate over the raw logs and unpacked data for InvalidateKeyset events raised by the SequencerInbox contract.
type SequencerInboxInvalidateKeysetIterator struct {
	Event *SequencerInboxInvalidateKeyset // Event containing the contract specifics and raw log

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
func (it *SequencerInboxInvalidateKeysetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxInvalidateKeyset)
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
		it.Event = new(SequencerInboxInvalidateKeyset)
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
func (it *SequencerInboxInvalidateKeysetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxInvalidateKeysetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxInvalidateKeyset represents a InvalidateKeyset event raised by the SequencerInbox contract.
type SequencerInboxInvalidateKeyset struct {
	KeysetHash [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterInvalidateKeyset is a free log retrieval operation binding the contract event 0x5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a.
//
// Solidity: event InvalidateKeyset(bytes32 indexed keysetHash)
func (_SequencerInbox *SequencerInboxFilterer) FilterInvalidateKeyset(opts *bind.FilterOpts, keysetHash [][32]byte) (*SequencerInboxInvalidateKeysetIterator, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInbox.contract.FilterLogs(opts, "InvalidateKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxInvalidateKeysetIterator{contract: _SequencerInbox.contract, event: "InvalidateKeyset", logs: logs, sub: sub}, nil
}

// WatchInvalidateKeyset is a free log subscription operation binding the contract event 0x5cb4218b272fd214168ac43e90fb4d05d6c36f0b17ffb4c2dd07c234d744eb2a.
//
// Solidity: event InvalidateKeyset(bytes32 indexed keysetHash)
func (_SequencerInbox *SequencerInboxFilterer) WatchInvalidateKeyset(opts *bind.WatchOpts, sink chan<- *SequencerInboxInvalidateKeyset, keysetHash [][32]byte) (event.Subscription, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInbox.contract.WatchLogs(opts, "InvalidateKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxInvalidateKeyset)
				if err := _SequencerInbox.contract.UnpackLog(event, "InvalidateKeyset", log); err != nil {
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
func (_SequencerInbox *SequencerInboxFilterer) ParseInvalidateKeyset(log types.Log) (*SequencerInboxInvalidateKeyset, error) {
	event := new(SequencerInboxInvalidateKeyset)
	if err := _SequencerInbox.contract.UnpackLog(event, "InvalidateKeyset", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxOwnerFunctionCalledIterator is returned from FilterOwnerFunctionCalled and is used to iterate over the raw logs and unpacked data for OwnerFunctionCalled events raised by the SequencerInbox contract.
type SequencerInboxOwnerFunctionCalledIterator struct {
	Event *SequencerInboxOwnerFunctionCalled // Event containing the contract specifics and raw log

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
func (it *SequencerInboxOwnerFunctionCalledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxOwnerFunctionCalled)
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
		it.Event = new(SequencerInboxOwnerFunctionCalled)
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
func (it *SequencerInboxOwnerFunctionCalledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxOwnerFunctionCalledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxOwnerFunctionCalled represents a OwnerFunctionCalled event raised by the SequencerInbox contract.
type SequencerInboxOwnerFunctionCalled struct {
	Id  *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterOwnerFunctionCalled is a free log retrieval operation binding the contract event 0xea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e.
//
// Solidity: event OwnerFunctionCalled(uint256 indexed id)
func (_SequencerInbox *SequencerInboxFilterer) FilterOwnerFunctionCalled(opts *bind.FilterOpts, id []*big.Int) (*SequencerInboxOwnerFunctionCalledIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _SequencerInbox.contract.FilterLogs(opts, "OwnerFunctionCalled", idRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxOwnerFunctionCalledIterator{contract: _SequencerInbox.contract, event: "OwnerFunctionCalled", logs: logs, sub: sub}, nil
}

// WatchOwnerFunctionCalled is a free log subscription operation binding the contract event 0xea8787f128d10b2cc0317b0c3960f9ad447f7f6c1ed189db1083ccffd20f456e.
//
// Solidity: event OwnerFunctionCalled(uint256 indexed id)
func (_SequencerInbox *SequencerInboxFilterer) WatchOwnerFunctionCalled(opts *bind.WatchOpts, sink chan<- *SequencerInboxOwnerFunctionCalled, id []*big.Int) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _SequencerInbox.contract.WatchLogs(opts, "OwnerFunctionCalled", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxOwnerFunctionCalled)
				if err := _SequencerInbox.contract.UnpackLog(event, "OwnerFunctionCalled", log); err != nil {
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
func (_SequencerInbox *SequencerInboxFilterer) ParseOwnerFunctionCalled(log types.Log) (*SequencerInboxOwnerFunctionCalled, error) {
	event := new(SequencerInboxOwnerFunctionCalled)
	if err := _SequencerInbox.contract.UnpackLog(event, "OwnerFunctionCalled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxSequencerBatchDataIterator is returned from FilterSequencerBatchData and is used to iterate over the raw logs and unpacked data for SequencerBatchData events raised by the SequencerInbox contract.
type SequencerInboxSequencerBatchDataIterator struct {
	Event *SequencerInboxSequencerBatchData // Event containing the contract specifics and raw log

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
func (it *SequencerInboxSequencerBatchDataIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxSequencerBatchData)
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
		it.Event = new(SequencerInboxSequencerBatchData)
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
func (it *SequencerInboxSequencerBatchDataIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxSequencerBatchDataIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxSequencerBatchData represents a SequencerBatchData event raised by the SequencerInbox contract.
type SequencerInboxSequencerBatchData struct {
	BatchSequenceNumber *big.Int
	Data                []byte
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterSequencerBatchData is a free log retrieval operation binding the contract event 0xfe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c20.
//
// Solidity: event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data)
func (_SequencerInbox *SequencerInboxFilterer) FilterSequencerBatchData(opts *bind.FilterOpts, batchSequenceNumber []*big.Int) (*SequencerInboxSequencerBatchDataIterator, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}

	logs, sub, err := _SequencerInbox.contract.FilterLogs(opts, "SequencerBatchData", batchSequenceNumberRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxSequencerBatchDataIterator{contract: _SequencerInbox.contract, event: "SequencerBatchData", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchData is a free log subscription operation binding the contract event 0xfe325ca1efe4c5c1062c981c3ee74b781debe4ea9440306a96d2a55759c66c20.
//
// Solidity: event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data)
func (_SequencerInbox *SequencerInboxFilterer) WatchSequencerBatchData(opts *bind.WatchOpts, sink chan<- *SequencerInboxSequencerBatchData, batchSequenceNumber []*big.Int) (event.Subscription, error) {

	var batchSequenceNumberRule []interface{}
	for _, batchSequenceNumberItem := range batchSequenceNumber {
		batchSequenceNumberRule = append(batchSequenceNumberRule, batchSequenceNumberItem)
	}

	logs, sub, err := _SequencerInbox.contract.WatchLogs(opts, "SequencerBatchData", batchSequenceNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxSequencerBatchData)
				if err := _SequencerInbox.contract.UnpackLog(event, "SequencerBatchData", log); err != nil {
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
func (_SequencerInbox *SequencerInboxFilterer) ParseSequencerBatchData(log types.Log) (*SequencerInboxSequencerBatchData, error) {
	event := new(SequencerInboxSequencerBatchData)
	if err := _SequencerInbox.contract.UnpackLog(event, "SequencerBatchData", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxSequencerBatchDeliveredIterator is returned from FilterSequencerBatchDelivered and is used to iterate over the raw logs and unpacked data for SequencerBatchDelivered events raised by the SequencerInbox contract.
type SequencerInboxSequencerBatchDeliveredIterator struct {
	Event *SequencerInboxSequencerBatchDelivered // Event containing the contract specifics and raw log

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
func (it *SequencerInboxSequencerBatchDeliveredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxSequencerBatchDelivered)
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
		it.Event = new(SequencerInboxSequencerBatchDelivered)
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
func (it *SequencerInboxSequencerBatchDeliveredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxSequencerBatchDeliveredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxSequencerBatchDelivered represents a SequencerBatchDelivered event raised by the SequencerInbox contract.
type SequencerInboxSequencerBatchDelivered struct {
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
func (_SequencerInbox *SequencerInboxFilterer) FilterSequencerBatchDelivered(opts *bind.FilterOpts, batchSequenceNumber []*big.Int, beforeAcc [][32]byte, afterAcc [][32]byte) (*SequencerInboxSequencerBatchDeliveredIterator, error) {

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

	logs, sub, err := _SequencerInbox.contract.FilterLogs(opts, "SequencerBatchDelivered", batchSequenceNumberRule, beforeAccRule, afterAccRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxSequencerBatchDeliveredIterator{contract: _SequencerInbox.contract, event: "SequencerBatchDelivered", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchDelivered is a free log subscription operation binding the contract event 0x7394f4a19a13c7b92b5bb71033245305946ef78452f7b4986ac1390b5df4ebd7.
//
// Solidity: event SequencerBatchDelivered(uint256 indexed batchSequenceNumber, bytes32 indexed beforeAcc, bytes32 indexed afterAcc, bytes32 delayedAcc, uint256 afterDelayedMessagesRead, (uint64,uint64,uint64,uint64) timeBounds, uint8 dataLocation)
func (_SequencerInbox *SequencerInboxFilterer) WatchSequencerBatchDelivered(opts *bind.WatchOpts, sink chan<- *SequencerInboxSequencerBatchDelivered, batchSequenceNumber []*big.Int, beforeAcc [][32]byte, afterAcc [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _SequencerInbox.contract.WatchLogs(opts, "SequencerBatchDelivered", batchSequenceNumberRule, beforeAccRule, afterAccRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxSequencerBatchDelivered)
				if err := _SequencerInbox.contract.UnpackLog(event, "SequencerBatchDelivered", log); err != nil {
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
func (_SequencerInbox *SequencerInboxFilterer) ParseSequencerBatchDelivered(log types.Log) (*SequencerInboxSequencerBatchDelivered, error) {
	event := new(SequencerInboxSequencerBatchDelivered)
	if err := _SequencerInbox.contract.UnpackLog(event, "SequencerBatchDelivered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerInboxSetValidKeysetIterator is returned from FilterSetValidKeyset and is used to iterate over the raw logs and unpacked data for SetValidKeyset events raised by the SequencerInbox contract.
type SequencerInboxSetValidKeysetIterator struct {
	Event *SequencerInboxSetValidKeyset // Event containing the contract specifics and raw log

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
func (it *SequencerInboxSetValidKeysetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerInboxSetValidKeyset)
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
		it.Event = new(SequencerInboxSetValidKeyset)
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
func (it *SequencerInboxSetValidKeysetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerInboxSetValidKeysetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerInboxSetValidKeyset represents a SetValidKeyset event raised by the SequencerInbox contract.
type SequencerInboxSetValidKeyset struct {
	KeysetHash  [32]byte
	KeysetBytes []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSetValidKeyset is a free log retrieval operation binding the contract event 0xabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722.
//
// Solidity: event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes)
func (_SequencerInbox *SequencerInboxFilterer) FilterSetValidKeyset(opts *bind.FilterOpts, keysetHash [][32]byte) (*SequencerInboxSetValidKeysetIterator, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInbox.contract.FilterLogs(opts, "SetValidKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return &SequencerInboxSetValidKeysetIterator{contract: _SequencerInbox.contract, event: "SetValidKeyset", logs: logs, sub: sub}, nil
}

// WatchSetValidKeyset is a free log subscription operation binding the contract event 0xabca9b7986bc22ad0160eb0cb88ae75411eacfba4052af0b457a9335ef655722.
//
// Solidity: event SetValidKeyset(bytes32 indexed keysetHash, bytes keysetBytes)
func (_SequencerInbox *SequencerInboxFilterer) WatchSetValidKeyset(opts *bind.WatchOpts, sink chan<- *SequencerInboxSetValidKeyset, keysetHash [][32]byte) (event.Subscription, error) {

	var keysetHashRule []interface{}
	for _, keysetHashItem := range keysetHash {
		keysetHashRule = append(keysetHashRule, keysetHashItem)
	}

	logs, sub, err := _SequencerInbox.contract.WatchLogs(opts, "SetValidKeyset", keysetHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerInboxSetValidKeyset)
				if err := _SequencerInbox.contract.UnpackLog(event, "SetValidKeyset", log); err != nil {
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
func (_SequencerInbox *SequencerInboxFilterer) ParseSetValidKeyset(log types.Log) (*SequencerInboxSetValidKeyset, error) {
	event := new(SequencerInboxSetValidKeyset)
	if err := _SequencerInbox.contract.UnpackLog(event, "SetValidKeyset", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
