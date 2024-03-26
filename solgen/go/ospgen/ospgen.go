// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package ospgen

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

// Instruction is an auto generated low-level Go binding around an user-defined struct.
type Instruction struct {
	Opcode       uint16
	ArgumentData *big.Int
}

// Machine is an auto generated low-level Go binding around an user-defined struct.
type Machine struct {
	Status          uint8
	ValueStack      ValueStack
	InternalStack   ValueStack
	FrameStack      StackFrameWindow
	GlobalStateHash [32]byte
	ModuleIdx       uint32
	FunctionIdx     uint32
	FunctionPc      uint32
	ModulesRoot     [32]byte
}

// Module is an auto generated low-level Go binding around an user-defined struct.
type Module struct {
	GlobalsMerkleRoot   [32]byte
	ModuleMemory        ModuleMemory
	TablesMerkleRoot    [32]byte
	FunctionsMerkleRoot [32]byte
	InternalsOffset     uint32
}

// ModuleMemory is an auto generated low-level Go binding around an user-defined struct.
type ModuleMemory struct {
	Size       uint64
	MaxSize    uint64
	MerkleRoot [32]byte
}

// StackFrame is an auto generated low-level Go binding around an user-defined struct.
type StackFrame struct {
	ReturnPc              Value
	LocalsMerkleRoot      [32]byte
	CallerModule          uint32
	CallerModuleInternals uint32
}

// StackFrameWindow is an auto generated low-level Go binding around an user-defined struct.
type StackFrameWindow struct {
	Proved        []StackFrame
	RemainingHash [32]byte
}

// Value is an auto generated low-level Go binding around an user-defined struct.
type Value struct {
	ValueType uint8
	Contents  *big.Int
}

// ValueArray is an auto generated low-level Go binding around an user-defined struct.
type ValueArray struct {
	Inner []Value
}

// ValueStack is an auto generated low-level Go binding around an user-defined struct.
type ValueStack struct {
	Proved        ValueArray
	RemainingHash [32]byte
}

// HashProofHelperMetaData contains all meta data concerning the HashProofHelper contract.
var HashProofHelperMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fullHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"offset\",\"type\":\"uint64\"}],\"name\":\"NotProven\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"fullHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"offset\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"part\",\"type\":\"bytes\"}],\"name\":\"PreimagePartProven\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"clearSplitProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fullHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"offset\",\"type\":\"uint64\"}],\"name\":\"getPreimagePart\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"keccakStates\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"offset\",\"type\":\"uint64\"},{\"internalType\":\"bytes\",\"name\":\"part\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint64\",\"name\":\"offset\",\"type\":\"uint64\"}],\"name\":\"proveWithFullPreimage\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"fullHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint64\",\"name\":\"offset\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"flags\",\"type\":\"uint256\"}],\"name\":\"proveWithSplitPreimage\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"fullHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611e58806100206000396000f3fe608060405234801561001057600080fd5b50600436106100675760003560e01c8063ae364ac211610050578063ae364ac2146100b6578063b7465799146100c0578063d4e5dd2b146100e257600080fd5b8063740085d71461006c57806379754cba14610095575b600080fd5b61007f61007a36600461194f565b6100f5565b60405161008c91906119c1565b60405180910390f35b6100a86100a3366004611a24565b610204565b60405190815260200161008c565b6100be610767565b005b6100d36100ce366004611a80565b6107af565b60405161008c93929190611ab6565b6100a86100f0366004611ae9565b610866565b60008281526020818152604080832067ffffffffffffffff85168452909152902080546060919060ff1661016d576040517f139647920000000000000000000000000000000000000000000000000000000081526004810185905267ffffffffffffffff841660248201526044015b60405180910390fd5b80600101805461017c90611b3d565b80601f01602080910402602001604051908101604052809291908181526020018280546101a890611b3d565b80156101f55780601f106101ca576101008083540402835291602001916101f5565b820191906000526020600020905b8154815290600101906020018083116101d857829003601f168201915b50505050509150505b92915050565b6000600182161515600283161561025c573360009081526001602081905260408220805467ffffffffffffffff19168155919061024390830182611890565b6102516002830160006118cd565b600982016000905550505b8080610270575061026e608886611b8d565b155b6102d6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601160248201527f4e4f545f424c4f434b5f414c49474e45440000000000000000000000000000006044820152606401610164565b3360009081526001602052604081206009810154909181900361031357815467ffffffffffffffff191667ffffffffffffffff871617825561038a565b815467ffffffffffffffff87811691161461038a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600b60248201527f444946465f4f46465345540000000000000000000000000000000000000000006044820152606401610164565b610396828989866109bd565b806103ac602067ffffffffffffffff8916611bb7565b1180156103c6575081600901548667ffffffffffffffff16105b156104fa576000818767ffffffffffffffff1611156103f6576103f38267ffffffffffffffff8916611bca565b90505b60008261040e602067ffffffffffffffff8b16611bb7565b6104189190611bca565b9050888111156104255750875b815b818110156104f657846001018b8b8381811061044557610445611bdd565b9050013560f81c60f81b908080548061045d90611b3d565b80601f810361047c5783600052602060002060ff1984168155603f9350505b506002919091019091558154600116156104a55790600052602060002090602091828204019190065b909190919091601f036101000a81548160ff021916907f01000000000000000000000000000000000000000000000000000000000000008404021790555080806104ee90611bf3565b915050610427565b5050505b8261050c57506000925061075f915050565b60005b60208110156105dd576000610525600883611c0d565b9050610532600582611c0d565b61053d600583611b8d565b610548906005611c21565b6105529190611bb7565b90506000610561600884611b8d565b61056c906008611c21565b85600201836019811061058157610581611bdd565b600481049091015467ffffffffffffffff6008600390931683026101000a9091041690911c91506105b3908490611c21565b6105be9060f8611bca565b60ff909116901b959095179450806105d581611bf3565b91505061050f565b50604051806040016040528060011515815260200183600101805461060190611b3d565b80601f016020809104026020016040519081016040528092919081815260200182805461062d90611b3d565b801561067a5780601f1061064f5761010080835404028352916020019161067a565b820191906000526020600020905b81548152906001019060200180831161065d57829003601f168201915b505050919092525050600085815260208181526040808320865467ffffffffffffffff16845282529091208251815460ff19169015151781559082015160018201906106c69082611c9d565b5050825460405167ffffffffffffffff909116915085907ff88493e8ac6179d3c1ba8712068367d7ecdd6f30d3b5de01198e7a449fe2802c9061070d906001870190611d5d565b60405180910390a33360009081526001602081905260408220805467ffffffffffffffff19168155919061074390830182611890565b6107516002830160006118cd565b600982016000905550505050505b949350505050565b3360009081526001602081905260408220805467ffffffffffffffff19168155919061079590830182611890565b6107a36002830160006118cd565b60098201600090555050565b60016020819052600091825260409091208054918101805467ffffffffffffffff909316926107dd90611b3d565b80601f016020809104026020016040519081016040528092919081815260200182805461080990611b3d565b80156108565780601f1061082b57610100808354040283529160200191610856565b820191906000526020600020905b81548152906001019060200180831161083957829003601f168201915b5050505050908060090154905083565b60008383604051610878929190611de8565b6040519081900390209050606067ffffffffffffffff83168411156109195760006108ad67ffffffffffffffff851686611bca565b905060208111156108bc575060205b8567ffffffffffffffff8516866108d38483611bb7565b926108e093929190611df8565b8080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250929450505050505b60408051808201825260018082526020808301858152600087815280835285812067ffffffffffffffff8a1682529092529390208251815460ff191690151517815592519192919082019061096e9082611c9d565b509050508267ffffffffffffffff16827ff88493e8ac6179d3c1ba8712068367d7ecdd6f30d3b5de01198e7a449fe2802c836040516109ad91906119c1565b60405180910390a3509392505050565b828290508460090160008282546109d49190611bb7565b90915550505b811580156109e6575080155b610c3e5760005b6088811015610b1257600083821015610a2357848483818110610a1257610a12611bdd565b919091013560f81c9150610a449050565b838203610a2e576001175b610a3a60016088611bca565b8203610a44576080175b6000610a51600884611c0d565b9050610a5e600582611c0d565b610a69600583611b8d565b610a74906005611c21565b610a7e9190611bb7565b9050610a8b600884611b8d565b610a96906008611c21565b67ffffffffffffffff168260ff1667ffffffffffffffff16901b876002018260198110610ac557610ac5611bdd565b60048104909101805467ffffffffffffffff60086003909416939093026101000a808204841690941883168402929093021990921617905550819050610b0a81611bf3565b9150506109ed565b50610b1b6118dc565b60005b6019811015610b8f57856002018160198110610b3c57610b3c611bdd565b600491828204019190066008029054906101000a900467ffffffffffffffff1667ffffffffffffffff16828260198110610b7857610b78611bdd565b602002015280610b8781611bf3565b915050610b1e565b50610b9981610c44565b905060005b6019811015610c1757818160198110610bb957610bb9611bdd565b6020020151866002018260198110610bd357610bd3611bdd565b600491828204019190066008026101000a81548167ffffffffffffffff021916908367ffffffffffffffff1602179055508080610c0f90611bf3565b915050610b9e565b506088831015610c275750610c3e565b610c348360888187611df8565b93509350506109da565b50505050565b610c4c6118dc565b610c546118fb565b610c5c6118fb565b610c646118dc565b600060405180610300016040528060018152602001618082815260200167800000000000808a8152602001678000000080008000815260200161808b81526020016380000001815260200167800000008000808181526020016780000000000080098152602001608a81526020016088815260200163800080098152602001638000000a8152602001638000808b815260200167800000000000008b8152602001678000000000008089815260200167800000000000800381526020016780000000000080028152602001678000000000000080815260200161800a815260200167800000008000000a81526020016780000000800080818152602001678000000000008080815260200163800000018152602001678000000080008008815250905060005b6018811015611885576080878101516060808a01516040808c01516020808e01518e511890911890921890931889526101208b01516101008c015160e08d015160c08e015160a08f0151181818189089018190526101c08b01516101a08c01516101808d01516101608e01516101408f0151181818189289019283526102608b01516102408c01516102208d01516102008e01516101e08f015118181818918901919091526103008a01516102e08b01516102c08c01516102a08d01516102808e01511818181892880183905267ffffffffffffffff6002820216678000000000000000918290041790921886525104856002602002015160020267ffffffffffffffff16178560006020020151188460016020020152678000000000000000856003602002015181610ebf57610ebf611b77565b04856003602002015160020267ffffffffffffffff16178560016020020151188460026020020152678000000000000000856004602002015181610f0557610f05611b77565b04856004602002015160020267ffffffffffffffff161785600260058110610f2f57610f2f611bdd565b6020020151186060850152845167800000000000000090865160608089015193909204600290910267ffffffffffffffff1617909118608086810191825286518a5118808b5287516020808d018051909218825289516040808f0180519092189091528a518e8801805190911890528a51948e0180519095189094528901805160a08e0180519091189052805160c08e0180519091189052805160e08e018051909118905280516101008e0180519091189052516101208d018051909118905291880180516101408d018051909118905280516101608d018051909118905280516101808d018051909118905280516101a08d0180519091189052516101c08c018051909118905292870180516101e08c018051909118905280516102008c018051909118905280516102208c018051909118905280516102408c0180519091189052516102608b018051909118905281516102808b018051909118905281516102a08b018051909118905281516102c08b018051909118905281516102e08b018051909118905290516103008a018051909118905290845251631000000090602089015167ffffffffffffffff6410000000009091021691900417610100840152604087015167200000000000000090604089015167ffffffffffffffff6008909102169190041761016084015260608701516280000090606089015167ffffffffffffffff65020000000000909102169190041761026084015260808701516540000000000090608089015167ffffffffffffffff6204000090910216919004176102c084015260a08701516780000000000000009004876005602002015160020267ffffffffffffffff1617836002601981106111b1576111b1611bdd565b602002015260c08701516210000081046510000000000090910267ffffffffffffffff9081169190911760a085015260e0880151664000000000000081046104009091028216176101a08501526101008801516208000081046520000000000090910282161761020085015261012088015160048082029092166740000000000000009091041761030085015261014088015161014089015167ffffffffffffffff674000000000000000909102169190041760808401526101608701516704000000000000009061016089015167ffffffffffffffff6040909102169190041760e0840152610180870151622000009061018089015167ffffffffffffffff6508000000000090910216919004176101408401526101a08701516602000000000000906101a089015167ffffffffffffffff61800090910216919004176102408401526101c08701516008906101c089015167ffffffffffffffff67200000000000000090910216919004176102a08401526101e0870151641000000000906101e089015167ffffffffffffffff6310000000909102169190041760208401526102008088015161020089015167ffffffffffffffff668000000000000090910216919004176101208401526102208701516480000000009061022089015167ffffffffffffffff63020000009091021691900417610180840152610240870151650800000000009061024089015167ffffffffffffffff6220000090910216919004176101e08401526102608701516101009061026089015167ffffffffffffffff67010000000000000090910216919004176102e08401526102808701516420000000009061028089015167ffffffffffffffff6308000000909102169190041760608401526102a087015165100000000000906102a089015167ffffffffffffffff62100000909102169190041760c08401526102c08701516302000000906102c089015167ffffffffffffffff64800000000090910216919004176101c08401526102e0870151670100000000000000906102e089015167ffffffffffffffff61010090910216919004176102208401526103008701516604000000000000900487601860200201516140000267ffffffffffffffff1617836014602002015282600a602002015183600560200201511916836000602002015118876000602002015282600b602002015183600660200201511916836001602002015118876001602002015282600c602002015183600760200201511916836002602002015118876002602002015282600d602002015183600860200201511916836003602002015118876003602002015282600e602002015183600960200201511916836004602002015118876004602002015282600f602002015183600a602002015119168360056020020151188760056020020152826010602002015183600b602002015119168360066020020151188760066020020152826011602002015183600c602002015119168360076020020151188760076020020152826012602002015183600d602002015119168360086020020151188760086020020152826013602002015183600e602002015119168360096020020151188760096020020152826014602002015183600f6020020151191683600a60200201511887600a602002015282601560200201518360106020020151191683600b60200201511887600b602002015282601660200201518360116020020151191683600c60200201511887600c602002015282601760200201518360126020020151191683600d60200201511887600d602002015282601860200201518360136020020151191683600e60200201511887600e602002015282600060200201518360146020020151191683600f60200201511887600f602002015282600160200201518360156020020151191683601060200201511887601060200201528260026020020151836016602002015119168360116020020151188760116020020152826003602002015183601760200201511916836012602002015118876012602002015282600460200201518360186020020151191683601360200201511887601360200201528260056020020151836000602002015119168360146020020151188760146020020152826006602002015183600160200201511916836015602002015118876015602002015282600760200201518360026020020151191683601660200201511887601660200201528260086020020151836003602002015119168360176020020151188760176020020152826009602002015183600460200201511916836018602002015118876018602002015281816018811061187357611873611bdd565b60200201518751188752600101610d8a565b509495945050505050565b50805461189c90611b3d565b6000825580601f106118ac575050565b601f0160209004906000526020600020908101906118ca9190611919565b50565b506118ca906007810190611919565b6040518061032001604052806019906020820280368337509192915050565b6040518060a001604052806005906020820280368337509192915050565b5b8082111561192e576000815560010161191a565b5090565b803567ffffffffffffffff8116811461194a57600080fd5b919050565b6000806040838503121561196257600080fd5b8235915061197260208401611932565b90509250929050565b6000815180845260005b818110156119a157602081850181015186830182015201611985565b506000602082860101526020601f19601f83011685010191505092915050565b6020815260006119d4602083018461197b565b9392505050565b60008083601f8401126119ed57600080fd5b50813567ffffffffffffffff811115611a0557600080fd5b602083019150836020828501011115611a1d57600080fd5b9250929050565b60008060008060608587031215611a3a57600080fd5b843567ffffffffffffffff811115611a5157600080fd5b611a5d878288016119db565b9095509350611a70905060208601611932565b9396929550929360400135925050565b600060208284031215611a9257600080fd5b813573ffffffffffffffffffffffffffffffffffffffff811681146119d457600080fd5b67ffffffffffffffff84168152606060208201526000611ad9606083018561197b565b9050826040830152949350505050565b600080600060408486031215611afe57600080fd5b833567ffffffffffffffff811115611b1557600080fd5b611b21868287016119db565b9094509250611b34905060208501611932565b90509250925092565b600181811c90821680611b5157607f821691505b602082108103611b7157634e487b7160e01b600052602260045260246000fd5b50919050565b634e487b7160e01b600052601260045260246000fd5b600082611b9c57611b9c611b77565b500690565b634e487b7160e01b600052601160045260246000fd5b808201808211156101fe576101fe611ba1565b818103818111156101fe576101fe611ba1565b634e487b7160e01b600052603260045260246000fd5b60006000198203611c0657611c06611ba1565b5060010190565b600082611c1c57611c1c611b77565b500490565b80820281158282048414176101fe576101fe611ba1565b634e487b7160e01b600052604160045260246000fd5b601f821115611c9857600081815260208120601f850160051c81016020861015611c755750805b601f850160051c820191505b81811015611c9457828155600101611c81565b5050505b505050565b815167ffffffffffffffff811115611cb757611cb7611c38565b611ccb81611cc58454611b3d565b84611c4e565b602080601f831160018114611d005760008415611ce85750858301515b600019600386901b1c1916600185901b178555611c94565b600085815260208120601f198616915b82811015611d2f57888601518255948401946001909101908401611d10565b5085821015611d4d5787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b6000602080835260008454611d7181611b3d565b80848701526040600180841660008114611d925760018114611dac57611dda565b60ff198516838a01528284151560051b8a01019550611dda565b896000528660002060005b85811015611dd25781548b8201860152908301908801611db7565b8a0184019650505b509398975050505050505050565b8183823760009101908152919050565b60008085851115611e0857600080fd5b83861115611e1557600080fd5b505082019391909203915056fea264697066735822122062fc5c963b0191d18adc1676c966d70ae115ebd72d6ddb33c3dba3d1278f6d0c64736f6c63430008110033",
}

// HashProofHelperABI is the input ABI used to generate the binding from.
// Deprecated: Use HashProofHelperMetaData.ABI instead.
var HashProofHelperABI = HashProofHelperMetaData.ABI

// HashProofHelperBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use HashProofHelperMetaData.Bin instead.
var HashProofHelperBin = HashProofHelperMetaData.Bin

// DeployHashProofHelper deploys a new Ethereum contract, binding an instance of HashProofHelper to it.
func DeployHashProofHelper(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *HashProofHelper, error) {
	parsed, err := HashProofHelperMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(HashProofHelperBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &HashProofHelper{HashProofHelperCaller: HashProofHelperCaller{contract: contract}, HashProofHelperTransactor: HashProofHelperTransactor{contract: contract}, HashProofHelperFilterer: HashProofHelperFilterer{contract: contract}}, nil
}

// HashProofHelper is an auto generated Go binding around an Ethereum contract.
type HashProofHelper struct {
	HashProofHelperCaller     // Read-only binding to the contract
	HashProofHelperTransactor // Write-only binding to the contract
	HashProofHelperFilterer   // Log filterer for contract events
}

// HashProofHelperCaller is an auto generated read-only Go binding around an Ethereum contract.
type HashProofHelperCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HashProofHelperTransactor is an auto generated write-only Go binding around an Ethereum contract.
type HashProofHelperTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HashProofHelperFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type HashProofHelperFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HashProofHelperSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type HashProofHelperSession struct {
	Contract     *HashProofHelper  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// HashProofHelperCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type HashProofHelperCallerSession struct {
	Contract *HashProofHelperCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// HashProofHelperTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type HashProofHelperTransactorSession struct {
	Contract     *HashProofHelperTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// HashProofHelperRaw is an auto generated low-level Go binding around an Ethereum contract.
type HashProofHelperRaw struct {
	Contract *HashProofHelper // Generic contract binding to access the raw methods on
}

// HashProofHelperCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type HashProofHelperCallerRaw struct {
	Contract *HashProofHelperCaller // Generic read-only contract binding to access the raw methods on
}

// HashProofHelperTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type HashProofHelperTransactorRaw struct {
	Contract *HashProofHelperTransactor // Generic write-only contract binding to access the raw methods on
}

// NewHashProofHelper creates a new instance of HashProofHelper, bound to a specific deployed contract.
func NewHashProofHelper(address common.Address, backend bind.ContractBackend) (*HashProofHelper, error) {
	contract, err := bindHashProofHelper(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &HashProofHelper{HashProofHelperCaller: HashProofHelperCaller{contract: contract}, HashProofHelperTransactor: HashProofHelperTransactor{contract: contract}, HashProofHelperFilterer: HashProofHelperFilterer{contract: contract}}, nil
}

// NewHashProofHelperCaller creates a new read-only instance of HashProofHelper, bound to a specific deployed contract.
func NewHashProofHelperCaller(address common.Address, caller bind.ContractCaller) (*HashProofHelperCaller, error) {
	contract, err := bindHashProofHelper(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &HashProofHelperCaller{contract: contract}, nil
}

// NewHashProofHelperTransactor creates a new write-only instance of HashProofHelper, bound to a specific deployed contract.
func NewHashProofHelperTransactor(address common.Address, transactor bind.ContractTransactor) (*HashProofHelperTransactor, error) {
	contract, err := bindHashProofHelper(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &HashProofHelperTransactor{contract: contract}, nil
}

// NewHashProofHelperFilterer creates a new log filterer instance of HashProofHelper, bound to a specific deployed contract.
func NewHashProofHelperFilterer(address common.Address, filterer bind.ContractFilterer) (*HashProofHelperFilterer, error) {
	contract, err := bindHashProofHelper(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &HashProofHelperFilterer{contract: contract}, nil
}

// bindHashProofHelper binds a generic wrapper to an already deployed contract.
func bindHashProofHelper(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := HashProofHelperMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HashProofHelper *HashProofHelperRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HashProofHelper.Contract.HashProofHelperCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HashProofHelper *HashProofHelperRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HashProofHelper.Contract.HashProofHelperTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HashProofHelper *HashProofHelperRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HashProofHelper.Contract.HashProofHelperTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HashProofHelper *HashProofHelperCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HashProofHelper.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HashProofHelper *HashProofHelperTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HashProofHelper.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HashProofHelper *HashProofHelperTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HashProofHelper.Contract.contract.Transact(opts, method, params...)
}

// GetPreimagePart is a free data retrieval call binding the contract method 0x740085d7.
//
// Solidity: function getPreimagePart(bytes32 fullHash, uint64 offset) view returns(bytes)
func (_HashProofHelper *HashProofHelperCaller) GetPreimagePart(opts *bind.CallOpts, fullHash [32]byte, offset uint64) ([]byte, error) {
	var out []interface{}
	err := _HashProofHelper.contract.Call(opts, &out, "getPreimagePart", fullHash, offset)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetPreimagePart is a free data retrieval call binding the contract method 0x740085d7.
//
// Solidity: function getPreimagePart(bytes32 fullHash, uint64 offset) view returns(bytes)
func (_HashProofHelper *HashProofHelperSession) GetPreimagePart(fullHash [32]byte, offset uint64) ([]byte, error) {
	return _HashProofHelper.Contract.GetPreimagePart(&_HashProofHelper.CallOpts, fullHash, offset)
}

// GetPreimagePart is a free data retrieval call binding the contract method 0x740085d7.
//
// Solidity: function getPreimagePart(bytes32 fullHash, uint64 offset) view returns(bytes)
func (_HashProofHelper *HashProofHelperCallerSession) GetPreimagePart(fullHash [32]byte, offset uint64) ([]byte, error) {
	return _HashProofHelper.Contract.GetPreimagePart(&_HashProofHelper.CallOpts, fullHash, offset)
}

// KeccakStates is a free data retrieval call binding the contract method 0xb7465799.
//
// Solidity: function keccakStates(address ) view returns(uint64 offset, bytes part, uint256 length)
func (_HashProofHelper *HashProofHelperCaller) KeccakStates(opts *bind.CallOpts, arg0 common.Address) (struct {
	Offset uint64
	Part   []byte
	Length *big.Int
}, error) {
	var out []interface{}
	err := _HashProofHelper.contract.Call(opts, &out, "keccakStates", arg0)

	outstruct := new(struct {
		Offset uint64
		Part   []byte
		Length *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Offset = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.Part = *abi.ConvertType(out[1], new([]byte)).(*[]byte)
	outstruct.Length = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// KeccakStates is a free data retrieval call binding the contract method 0xb7465799.
//
// Solidity: function keccakStates(address ) view returns(uint64 offset, bytes part, uint256 length)
func (_HashProofHelper *HashProofHelperSession) KeccakStates(arg0 common.Address) (struct {
	Offset uint64
	Part   []byte
	Length *big.Int
}, error) {
	return _HashProofHelper.Contract.KeccakStates(&_HashProofHelper.CallOpts, arg0)
}

// KeccakStates is a free data retrieval call binding the contract method 0xb7465799.
//
// Solidity: function keccakStates(address ) view returns(uint64 offset, bytes part, uint256 length)
func (_HashProofHelper *HashProofHelperCallerSession) KeccakStates(arg0 common.Address) (struct {
	Offset uint64
	Part   []byte
	Length *big.Int
}, error) {
	return _HashProofHelper.Contract.KeccakStates(&_HashProofHelper.CallOpts, arg0)
}

// ClearSplitProof is a paid mutator transaction binding the contract method 0xae364ac2.
//
// Solidity: function clearSplitProof() returns()
func (_HashProofHelper *HashProofHelperTransactor) ClearSplitProof(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HashProofHelper.contract.Transact(opts, "clearSplitProof")
}

// ClearSplitProof is a paid mutator transaction binding the contract method 0xae364ac2.
//
// Solidity: function clearSplitProof() returns()
func (_HashProofHelper *HashProofHelperSession) ClearSplitProof() (*types.Transaction, error) {
	return _HashProofHelper.Contract.ClearSplitProof(&_HashProofHelper.TransactOpts)
}

// ClearSplitProof is a paid mutator transaction binding the contract method 0xae364ac2.
//
// Solidity: function clearSplitProof() returns()
func (_HashProofHelper *HashProofHelperTransactorSession) ClearSplitProof() (*types.Transaction, error) {
	return _HashProofHelper.Contract.ClearSplitProof(&_HashProofHelper.TransactOpts)
}

// ProveWithFullPreimage is a paid mutator transaction binding the contract method 0xd4e5dd2b.
//
// Solidity: function proveWithFullPreimage(bytes data, uint64 offset) returns(bytes32 fullHash)
func (_HashProofHelper *HashProofHelperTransactor) ProveWithFullPreimage(opts *bind.TransactOpts, data []byte, offset uint64) (*types.Transaction, error) {
	return _HashProofHelper.contract.Transact(opts, "proveWithFullPreimage", data, offset)
}

// ProveWithFullPreimage is a paid mutator transaction binding the contract method 0xd4e5dd2b.
//
// Solidity: function proveWithFullPreimage(bytes data, uint64 offset) returns(bytes32 fullHash)
func (_HashProofHelper *HashProofHelperSession) ProveWithFullPreimage(data []byte, offset uint64) (*types.Transaction, error) {
	return _HashProofHelper.Contract.ProveWithFullPreimage(&_HashProofHelper.TransactOpts, data, offset)
}

// ProveWithFullPreimage is a paid mutator transaction binding the contract method 0xd4e5dd2b.
//
// Solidity: function proveWithFullPreimage(bytes data, uint64 offset) returns(bytes32 fullHash)
func (_HashProofHelper *HashProofHelperTransactorSession) ProveWithFullPreimage(data []byte, offset uint64) (*types.Transaction, error) {
	return _HashProofHelper.Contract.ProveWithFullPreimage(&_HashProofHelper.TransactOpts, data, offset)
}

// ProveWithSplitPreimage is a paid mutator transaction binding the contract method 0x79754cba.
//
// Solidity: function proveWithSplitPreimage(bytes data, uint64 offset, uint256 flags) returns(bytes32 fullHash)
func (_HashProofHelper *HashProofHelperTransactor) ProveWithSplitPreimage(opts *bind.TransactOpts, data []byte, offset uint64, flags *big.Int) (*types.Transaction, error) {
	return _HashProofHelper.contract.Transact(opts, "proveWithSplitPreimage", data, offset, flags)
}

// ProveWithSplitPreimage is a paid mutator transaction binding the contract method 0x79754cba.
//
// Solidity: function proveWithSplitPreimage(bytes data, uint64 offset, uint256 flags) returns(bytes32 fullHash)
func (_HashProofHelper *HashProofHelperSession) ProveWithSplitPreimage(data []byte, offset uint64, flags *big.Int) (*types.Transaction, error) {
	return _HashProofHelper.Contract.ProveWithSplitPreimage(&_HashProofHelper.TransactOpts, data, offset, flags)
}

// ProveWithSplitPreimage is a paid mutator transaction binding the contract method 0x79754cba.
//
// Solidity: function proveWithSplitPreimage(bytes data, uint64 offset, uint256 flags) returns(bytes32 fullHash)
func (_HashProofHelper *HashProofHelperTransactorSession) ProveWithSplitPreimage(data []byte, offset uint64, flags *big.Int) (*types.Transaction, error) {
	return _HashProofHelper.Contract.ProveWithSplitPreimage(&_HashProofHelper.TransactOpts, data, offset, flags)
}

// HashProofHelperPreimagePartProvenIterator is returned from FilterPreimagePartProven and is used to iterate over the raw logs and unpacked data for PreimagePartProven events raised by the HashProofHelper contract.
type HashProofHelperPreimagePartProvenIterator struct {
	Event *HashProofHelperPreimagePartProven // Event containing the contract specifics and raw log

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
func (it *HashProofHelperPreimagePartProvenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HashProofHelperPreimagePartProven)
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
		it.Event = new(HashProofHelperPreimagePartProven)
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
func (it *HashProofHelperPreimagePartProvenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HashProofHelperPreimagePartProvenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HashProofHelperPreimagePartProven represents a PreimagePartProven event raised by the HashProofHelper contract.
type HashProofHelperPreimagePartProven struct {
	FullHash [32]byte
	Offset   uint64
	Part     []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterPreimagePartProven is a free log retrieval operation binding the contract event 0xf88493e8ac6179d3c1ba8712068367d7ecdd6f30d3b5de01198e7a449fe2802c.
//
// Solidity: event PreimagePartProven(bytes32 indexed fullHash, uint64 indexed offset, bytes part)
func (_HashProofHelper *HashProofHelperFilterer) FilterPreimagePartProven(opts *bind.FilterOpts, fullHash [][32]byte, offset []uint64) (*HashProofHelperPreimagePartProvenIterator, error) {

	var fullHashRule []interface{}
	for _, fullHashItem := range fullHash {
		fullHashRule = append(fullHashRule, fullHashItem)
	}
	var offsetRule []interface{}
	for _, offsetItem := range offset {
		offsetRule = append(offsetRule, offsetItem)
	}

	logs, sub, err := _HashProofHelper.contract.FilterLogs(opts, "PreimagePartProven", fullHashRule, offsetRule)
	if err != nil {
		return nil, err
	}
	return &HashProofHelperPreimagePartProvenIterator{contract: _HashProofHelper.contract, event: "PreimagePartProven", logs: logs, sub: sub}, nil
}

// WatchPreimagePartProven is a free log subscription operation binding the contract event 0xf88493e8ac6179d3c1ba8712068367d7ecdd6f30d3b5de01198e7a449fe2802c.
//
// Solidity: event PreimagePartProven(bytes32 indexed fullHash, uint64 indexed offset, bytes part)
func (_HashProofHelper *HashProofHelperFilterer) WatchPreimagePartProven(opts *bind.WatchOpts, sink chan<- *HashProofHelperPreimagePartProven, fullHash [][32]byte, offset []uint64) (event.Subscription, error) {

	var fullHashRule []interface{}
	for _, fullHashItem := range fullHash {
		fullHashRule = append(fullHashRule, fullHashItem)
	}
	var offsetRule []interface{}
	for _, offsetItem := range offset {
		offsetRule = append(offsetRule, offsetItem)
	}

	logs, sub, err := _HashProofHelper.contract.WatchLogs(opts, "PreimagePartProven", fullHashRule, offsetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HashProofHelperPreimagePartProven)
				if err := _HashProofHelper.contract.UnpackLog(event, "PreimagePartProven", log); err != nil {
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

// ParsePreimagePartProven is a log parse operation binding the contract event 0xf88493e8ac6179d3c1ba8712068367d7ecdd6f30d3b5de01198e7a449fe2802c.
//
// Solidity: event PreimagePartProven(bytes32 indexed fullHash, uint64 indexed offset, bytes part)
func (_HashProofHelper *HashProofHelperFilterer) ParsePreimagePartProven(log types.Log) (*HashProofHelperPreimagePartProven, error) {
	event := new(HashProofHelperPreimagePartProven)
	if err := _HashProofHelper.contract.UnpackLog(event, "PreimagePartProven", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IOneStepProofEntryMetaData contains all meta data concerning the IOneStepProofEntry contract.
var IOneStepProofEntryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"}],\"internalType\":\"structExecutionState\",\"name\":\"execState\",\"type\":\"tuple\"}],\"name\":\"getMachineHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"initialWasmModuleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"machineStep\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"proveOneStep\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"afterHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IOneStepProofEntryABI is the input ABI used to generate the binding from.
// Deprecated: Use IOneStepProofEntryMetaData.ABI instead.
var IOneStepProofEntryABI = IOneStepProofEntryMetaData.ABI

// IOneStepProofEntry is an auto generated Go binding around an Ethereum contract.
type IOneStepProofEntry struct {
	IOneStepProofEntryCaller     // Read-only binding to the contract
	IOneStepProofEntryTransactor // Write-only binding to the contract
	IOneStepProofEntryFilterer   // Log filterer for contract events
}

// IOneStepProofEntryCaller is an auto generated read-only Go binding around an Ethereum contract.
type IOneStepProofEntryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOneStepProofEntryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IOneStepProofEntryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOneStepProofEntryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IOneStepProofEntryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOneStepProofEntrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IOneStepProofEntrySession struct {
	Contract     *IOneStepProofEntry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// IOneStepProofEntryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IOneStepProofEntryCallerSession struct {
	Contract *IOneStepProofEntryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// IOneStepProofEntryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IOneStepProofEntryTransactorSession struct {
	Contract     *IOneStepProofEntryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// IOneStepProofEntryRaw is an auto generated low-level Go binding around an Ethereum contract.
type IOneStepProofEntryRaw struct {
	Contract *IOneStepProofEntry // Generic contract binding to access the raw methods on
}

// IOneStepProofEntryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IOneStepProofEntryCallerRaw struct {
	Contract *IOneStepProofEntryCaller // Generic read-only contract binding to access the raw methods on
}

// IOneStepProofEntryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IOneStepProofEntryTransactorRaw struct {
	Contract *IOneStepProofEntryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIOneStepProofEntry creates a new instance of IOneStepProofEntry, bound to a specific deployed contract.
func NewIOneStepProofEntry(address common.Address, backend bind.ContractBackend) (*IOneStepProofEntry, error) {
	contract, err := bindIOneStepProofEntry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IOneStepProofEntry{IOneStepProofEntryCaller: IOneStepProofEntryCaller{contract: contract}, IOneStepProofEntryTransactor: IOneStepProofEntryTransactor{contract: contract}, IOneStepProofEntryFilterer: IOneStepProofEntryFilterer{contract: contract}}, nil
}

// NewIOneStepProofEntryCaller creates a new read-only instance of IOneStepProofEntry, bound to a specific deployed contract.
func NewIOneStepProofEntryCaller(address common.Address, caller bind.ContractCaller) (*IOneStepProofEntryCaller, error) {
	contract, err := bindIOneStepProofEntry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IOneStepProofEntryCaller{contract: contract}, nil
}

// NewIOneStepProofEntryTransactor creates a new write-only instance of IOneStepProofEntry, bound to a specific deployed contract.
func NewIOneStepProofEntryTransactor(address common.Address, transactor bind.ContractTransactor) (*IOneStepProofEntryTransactor, error) {
	contract, err := bindIOneStepProofEntry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IOneStepProofEntryTransactor{contract: contract}, nil
}

// NewIOneStepProofEntryFilterer creates a new log filterer instance of IOneStepProofEntry, bound to a specific deployed contract.
func NewIOneStepProofEntryFilterer(address common.Address, filterer bind.ContractFilterer) (*IOneStepProofEntryFilterer, error) {
	contract, err := bindIOneStepProofEntry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IOneStepProofEntryFilterer{contract: contract}, nil
}

// bindIOneStepProofEntry binds a generic wrapper to an already deployed contract.
func bindIOneStepProofEntry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IOneStepProofEntryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOneStepProofEntry *IOneStepProofEntryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOneStepProofEntry.Contract.IOneStepProofEntryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOneStepProofEntry *IOneStepProofEntryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOneStepProofEntry.Contract.IOneStepProofEntryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOneStepProofEntry *IOneStepProofEntryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOneStepProofEntry.Contract.IOneStepProofEntryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOneStepProofEntry *IOneStepProofEntryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOneStepProofEntry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOneStepProofEntry *IOneStepProofEntryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOneStepProofEntry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOneStepProofEntry *IOneStepProofEntryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOneStepProofEntry.Contract.contract.Transact(opts, method, params...)
}

// GetMachineHash is a free data retrieval call binding the contract method 0xc39619c4.
//
// Solidity: function getMachineHash(((bytes32[2],uint64[2]),uint8) execState) pure returns(bytes32)
func (_IOneStepProofEntry *IOneStepProofEntryCaller) GetMachineHash(opts *bind.CallOpts, execState ExecutionState) ([32]byte, error) {
	var out []interface{}
	err := _IOneStepProofEntry.contract.Call(opts, &out, "getMachineHash", execState)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetMachineHash is a free data retrieval call binding the contract method 0xc39619c4.
//
// Solidity: function getMachineHash(((bytes32[2],uint64[2]),uint8) execState) pure returns(bytes32)
func (_IOneStepProofEntry *IOneStepProofEntrySession) GetMachineHash(execState ExecutionState) ([32]byte, error) {
	return _IOneStepProofEntry.Contract.GetMachineHash(&_IOneStepProofEntry.CallOpts, execState)
}

// GetMachineHash is a free data retrieval call binding the contract method 0xc39619c4.
//
// Solidity: function getMachineHash(((bytes32[2],uint64[2]),uint8) execState) pure returns(bytes32)
func (_IOneStepProofEntry *IOneStepProofEntryCallerSession) GetMachineHash(execState ExecutionState) ([32]byte, error) {
	return _IOneStepProofEntry.Contract.GetMachineHash(&_IOneStepProofEntry.CallOpts, execState)
}

// ProveOneStep is a free data retrieval call binding the contract method 0xb5112fd2.
//
// Solidity: function proveOneStep((uint256,address,bytes32) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_IOneStepProofEntry *IOneStepProofEntryCaller) ProveOneStep(opts *bind.CallOpts, execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	var out []interface{}
	err := _IOneStepProofEntry.contract.Call(opts, &out, "proveOneStep", execCtx, machineStep, beforeHash, proof)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProveOneStep is a free data retrieval call binding the contract method 0xb5112fd2.
//
// Solidity: function proveOneStep((uint256,address,bytes32) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_IOneStepProofEntry *IOneStepProofEntrySession) ProveOneStep(execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	return _IOneStepProofEntry.Contract.ProveOneStep(&_IOneStepProofEntry.CallOpts, execCtx, machineStep, beforeHash, proof)
}

// ProveOneStep is a free data retrieval call binding the contract method 0xb5112fd2.
//
// Solidity: function proveOneStep((uint256,address,bytes32) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_IOneStepProofEntry *IOneStepProofEntryCallerSession) ProveOneStep(execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	return _IOneStepProofEntry.Contract.ProveOneStep(&_IOneStepProofEntry.CallOpts, execCtx, machineStep, beforeHash, proof)
}

// IOneStepProverMetaData contains all meta data concerning the IOneStepProver contract.
var IOneStepProverMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"initialWasmModuleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"mach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"mod\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"opcode\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"argumentData\",\"type\":\"uint256\"}],\"internalType\":\"structInstruction\",\"name\":\"instruction\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"result\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"resultMod\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IOneStepProverABI is the input ABI used to generate the binding from.
// Deprecated: Use IOneStepProverMetaData.ABI instead.
var IOneStepProverABI = IOneStepProverMetaData.ABI

// IOneStepProver is an auto generated Go binding around an Ethereum contract.
type IOneStepProver struct {
	IOneStepProverCaller     // Read-only binding to the contract
	IOneStepProverTransactor // Write-only binding to the contract
	IOneStepProverFilterer   // Log filterer for contract events
}

// IOneStepProverCaller is an auto generated read-only Go binding around an Ethereum contract.
type IOneStepProverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOneStepProverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IOneStepProverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOneStepProverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IOneStepProverFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IOneStepProverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IOneStepProverSession struct {
	Contract     *IOneStepProver   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IOneStepProverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IOneStepProverCallerSession struct {
	Contract *IOneStepProverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// IOneStepProverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IOneStepProverTransactorSession struct {
	Contract     *IOneStepProverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// IOneStepProverRaw is an auto generated low-level Go binding around an Ethereum contract.
type IOneStepProverRaw struct {
	Contract *IOneStepProver // Generic contract binding to access the raw methods on
}

// IOneStepProverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IOneStepProverCallerRaw struct {
	Contract *IOneStepProverCaller // Generic read-only contract binding to access the raw methods on
}

// IOneStepProverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IOneStepProverTransactorRaw struct {
	Contract *IOneStepProverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIOneStepProver creates a new instance of IOneStepProver, bound to a specific deployed contract.
func NewIOneStepProver(address common.Address, backend bind.ContractBackend) (*IOneStepProver, error) {
	contract, err := bindIOneStepProver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IOneStepProver{IOneStepProverCaller: IOneStepProverCaller{contract: contract}, IOneStepProverTransactor: IOneStepProverTransactor{contract: contract}, IOneStepProverFilterer: IOneStepProverFilterer{contract: contract}}, nil
}

// NewIOneStepProverCaller creates a new read-only instance of IOneStepProver, bound to a specific deployed contract.
func NewIOneStepProverCaller(address common.Address, caller bind.ContractCaller) (*IOneStepProverCaller, error) {
	contract, err := bindIOneStepProver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IOneStepProverCaller{contract: contract}, nil
}

// NewIOneStepProverTransactor creates a new write-only instance of IOneStepProver, bound to a specific deployed contract.
func NewIOneStepProverTransactor(address common.Address, transactor bind.ContractTransactor) (*IOneStepProverTransactor, error) {
	contract, err := bindIOneStepProver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IOneStepProverTransactor{contract: contract}, nil
}

// NewIOneStepProverFilterer creates a new log filterer instance of IOneStepProver, bound to a specific deployed contract.
func NewIOneStepProverFilterer(address common.Address, filterer bind.ContractFilterer) (*IOneStepProverFilterer, error) {
	contract, err := bindIOneStepProver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IOneStepProverFilterer{contract: contract}, nil
}

// bindIOneStepProver binds a generic wrapper to an already deployed contract.
func bindIOneStepProver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IOneStepProverMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOneStepProver *IOneStepProverRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOneStepProver.Contract.IOneStepProverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOneStepProver *IOneStepProverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOneStepProver.Contract.IOneStepProverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOneStepProver *IOneStepProverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOneStepProver.Contract.IOneStepProverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IOneStepProver *IOneStepProverCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IOneStepProver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IOneStepProver *IOneStepProverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IOneStepProver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IOneStepProver *IOneStepProverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IOneStepProver.Contract.contract.Transact(opts, method, params...)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod, (uint16,uint256) instruction, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) result, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) resultMod)
func (_IOneStepProver *IOneStepProverCaller) ExecuteOneStep(opts *bind.CallOpts, execCtx ExecutionContext, mach Machine, mod Module, instruction Instruction, proof []byte) (struct {
	Result    Machine
	ResultMod Module
}, error) {
	var out []interface{}
	err := _IOneStepProver.contract.Call(opts, &out, "executeOneStep", execCtx, mach, mod, instruction, proof)

	outstruct := new(struct {
		Result    Machine
		ResultMod Module
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Result = *abi.ConvertType(out[0], new(Machine)).(*Machine)
	outstruct.ResultMod = *abi.ConvertType(out[1], new(Module)).(*Module)

	return *outstruct, err

}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod, (uint16,uint256) instruction, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) result, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) resultMod)
func (_IOneStepProver *IOneStepProverSession) ExecuteOneStep(execCtx ExecutionContext, mach Machine, mod Module, instruction Instruction, proof []byte) (struct {
	Result    Machine
	ResultMod Module
}, error) {
	return _IOneStepProver.Contract.ExecuteOneStep(&_IOneStepProver.CallOpts, execCtx, mach, mod, instruction, proof)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod, (uint16,uint256) instruction, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) result, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) resultMod)
func (_IOneStepProver *IOneStepProverCallerSession) ExecuteOneStep(execCtx ExecutionContext, mach Machine, mod Module, instruction Instruction, proof []byte) (struct {
	Result    Machine
	ResultMod Module
}, error) {
	return _IOneStepProver.Contract.ExecuteOneStep(&_IOneStepProver.CallOpts, execCtx, mach, mod, instruction, proof)
}

// OneStepProofEntryMetaData contains all meta data concerning the OneStepProofEntry contract.
var OneStepProofEntryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIOneStepProver\",\"name\":\"prover0_\",\"type\":\"address\"},{\"internalType\":\"contractIOneStepProver\",\"name\":\"proverMem_\",\"type\":\"address\"},{\"internalType\":\"contractIOneStepProver\",\"name\":\"proverMath_\",\"type\":\"address\"},{\"internalType\":\"contractIOneStepProver\",\"name\":\"proverHostIo_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"}],\"internalType\":\"structExecutionState\",\"name\":\"execState\",\"type\":\"tuple\"}],\"name\":\"getMachineHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"initialWasmModuleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"machineStep\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"proveOneStep\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"afterHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"prover0\",\"outputs\":[{\"internalType\":\"contractIOneStepProver\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proverHostIo\",\"outputs\":[{\"internalType\":\"contractIOneStepProver\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proverMath\",\"outputs\":[{\"internalType\":\"contractIOneStepProver\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proverMem\",\"outputs\":[{\"internalType\":\"contractIOneStepProver\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b5060405162002c3b38038062002c3b8339810160408190526200003491620000a5565b600080546001600160a01b039586166001600160a01b031991821617909155600180549486169482169490941790935560028054928516928416929092179091556003805491909316911617905562000102565b80516001600160a01b0381168114620000a057600080fd5b919050565b60008060008060808587031215620000bc57600080fd5b620000c78562000088565b9350620000d76020860162000088565b9250620000e76040860162000088565b9150620000f76060860162000088565b905092959194509250565b612b2980620001126000396000f3fe608060405234801561001057600080fd5b50600436106100725760003560e01c806366e5d9c31161005057806366e5d9c3146100cd578063b5112fd2146100e0578063c39619c41461010157600080fd5b80631f128bc01461007757806330a5509f146100a75780635f52fd7c146100ba575b600080fd5b60015461008a906001600160a01b031681565b6040516001600160a01b0390911681526020015b60405180910390f35b60005461008a906001600160a01b031681565b60035461008a906001600160a01b031681565b60025461008a906001600160a01b031681565b6100f36100ee366004611f66565b610114565b60405190815260200161009e565b6100f361010f366004612002565b6107b9565b600061011e611e0d565b610126611eb6565b6040805160208101909152606081526040805180820190915260008082526020820152600061015688888361090c565b90955090508861016586610afc565b146101b75760405162461bcd60e51b815260206004820152601360248201527f4d414348494e455f4245464f52455f484153480000000000000000000000000060448201526064015b60405180910390fd5b6000855160028111156101cc576101cc612014565b146102b3576101d9611f00565b6101e4898984610ce4565b60808801519093509091506101f882610dc0565b146102455760405162461bcd60e51b815260206004820152601060248201527f4241445f474c4f42414c5f53544154450000000000000000000000000000000060448201526064016101ae565b60018651600281111561025a5761025a612014565b14801561026557508a155b801561028657508b3561027a82602001515190565b67ffffffffffffffff16105b156102aa5761029d86608001518d60400135610e50565b96505050505050506107b0565b61029d86610afc565b650800000000006102c58b6001612040565b036102e357600285526102d785610afc565b955050505050506107b0565b6102ee88888361103a565b90945090506102fe8888836110e6565b809250819450505084610100015161032b8660a0015163ffffffff1686866111c19092919063ffffffff16565b146103785760405162461bcd60e51b815260206004820152600c60248201527f4d4f44554c45535f524f4f54000000000000000000000000000000000000000060448201526064016101ae565b6040805160208101909152606081526040805160208101909152606081526103a18a8a85611214565b90945092506103b18a8a856110e6565b935091506103c08a8a856110e6565b809450819250505060006103e98860e0015163ffffffff16868561126e9092919063ffffffff16565b9050600061040c8960c0015163ffffffff1683856112b99092919063ffffffff16565b9050876060015181146104615760405162461bcd60e51b815260206004820152601260248201527f4241445f46554e4354494f4e535f524f4f54000000000000000000000000000060448201526064016101ae565b506104749250899150839050818b612053565b975097505060008460a0015163ffffffff16905060018560e00181815161049b919061207d565b63ffffffff1690525081516000602861ffff8316108015906104c25750603561ffff831611155b806104e25750603661ffff8316108015906104e25750603e61ffff831611155b806104f1575061ffff8216603f145b80610500575061ffff82166040145b1561051757506001546001600160a01b031661070c565b61ffff82166045148061052e575061ffff82166050145b8061055c5750604661ffff83161080159061055c5750610550600960466120a1565b61ffff168261ffff1611155b8061058a5750606761ffff83161080159061058a575061057e600260676120a1565b61ffff168261ffff1611155b806105aa5750606a61ffff8316108015906105aa5750607861ffff831611155b806105d85750605161ffff8316108015906105d857506105cc600960516120a1565b61ffff168261ffff1611155b806106065750607961ffff83161080159061060657506105fa600260796120a1565b61ffff168261ffff1611155b806106265750607c61ffff8316108015906106265750608a61ffff831611155b80610635575061ffff821660a7145b80610652575061ffff821660ac1480610652575061ffff821660ad145b80610672575060c061ffff831610801590610672575060c461ffff831611155b80610692575060bc61ffff831610801590610692575060bf61ffff831611155b156106a957506002546001600160a01b031661070c565b61801061ffff8316108015906106c5575061801361ffff831611155b806106e7575061802061ffff8316108015906106e7575061802261ffff831611155b156106fe57506003546001600160a01b031661070c565b506000546001600160a01b03165b806001600160a01b03166397cc779a8e8989888f8f6040518763ffffffff1660e01b815260040161074296959493929190612200565b600060405180830381865afa15801561075f573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405261078791908101906127fa565b90975095506107978584886111c1565b6101008801526107a687610afc565b9750505050505050505b95945050505050565b600060016107cd60a084016080850161291d565b60028111156107de576107de612014565b0361084c576107fa6107f536849003840184612941565b610dc0565b6040517f4d616368696e652066696e69736865643a000000000000000000000000000000602082015260318101919091526051015b604051602081830303815290604052805190602001209050919050565b600261085e60a084016080850161291d565b600281111561086f5761086f612014565b036108bf576108866107f536849003840184612941565b6040517f4d616368696e65206572726f7265643a000000000000000000000000000000006020820152603081019190915260500161082f565b60405162461bcd60e51b815260206004820152601260248201527f4241445f4d414348494e455f535441545553000000000000000000000000000060448201526064016101ae565b919050565b610914611e0d565b8160008061092387878561134a565b9350905060ff811660000361093b57600091506109ab565b8060ff1660010361094f57600191506109ab565b8060ff1660020361096357600291506109ab565b60405162461bcd60e51b815260206004820152601360248201527f554e4b4e4f574e5f4d4143485f5354415455530000000000000000000000000060448201526064016101ae565b50604080516060808201835291810191825290815260006020820152604080516060808201835291810191825290815260006020820152600080600080610a08604051806040016040528060608152602001600080191681525090565b6000610a158e8e8c611380565b9a509750610a248e8e8c611380565b9a509650610a338e8e8c611493565b9a509150610a428e8e8c6115d3565b9a509550610a518e8e8c6115ef565b9a509450610a608e8e8c6115ef565b9a509350610a6f8e8e8c6115ef565b9a509250610a7e8e8e8c6115d3565b809b5081925050506040518061012001604052808a6002811115610aa457610aa4612014565b81526020018981526020018881526020018381526020018781526020018663ffffffff1681526020018563ffffffff1681526020018463ffffffff168152602001828152509a50505050505050505050935093915050565b60008082516002811115610b1257610b12612014565b03610bec57610b248260200151611653565b610b318360400151611653565b610b3e84606001516116e9565b608085015160a086015160c087015160e0808901516101008a01516040517f4d616368696e652072756e6e696e673a00000000000000000000000000000000602082015260308101999099526050890197909752607088019590955260908701939093527fffffffff0000000000000000000000000000000000000000000000000000000091831b821660b0870152821b811660b486015291901b1660b883015260bc82015260dc0161082f565b600182516002811115610c0157610c01612014565b03610c445760808201516040517f4d616368696e652066696e69736865643a0000000000000000000000000000006020820152603181019190915260510161082f565b600282516002811115610c5957610c59612014565b03610c9c5760808201516040517f4d616368696e65206572726f7265643a000000000000000000000000000000006020820152603081019190915260500161082f565b60405162461bcd60e51b815260206004820152600f60248201527f4241445f4d4143485f535441545553000000000000000000000000000000000060448201526064016101ae565b610cec611f00565b81610cf5611f25565b610cfd611f25565b60005b600260ff82161015610d4857610d178888866115d3565b848360ff1660028110610d2c57610d2c612a03565b6020020191909152935080610d4081612a19565b915050610d00565b5060005b600260ff82161015610da357610d6388888661178d565b838360ff1660028110610d7857610d78612a03565b67ffffffffffffffff9093166020939093020191909152935080610d9b81612a19565b915050610d4c565b506040805180820190915291825260208201529590945092505050565b80518051602091820151828401518051908401516040517f476c6f62616c2073746174653a0000000000000000000000000000000000000095810195909552602d850193909352604d8401919091527fffffffffffffffff00000000000000000000000000000000000000000000000060c091821b8116606d85015291901b166075820152600090607d0161082f565b60408051600380825260808201909252600091829190816020015b6040805180820190915260008082526020820152815260200190600190039081610e6b575050604080518082018252600080825260209182018190528251808401909352600483529082015290915081600081518110610ecd57610ecd612a03565b6020026020010181905250610f106000604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b81600181518110610f2357610f23612a03565b6020026020010181905250610f666000604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b81600281518110610f7957610f79612a03565b6020908102919091018101919091526040805180830182528381528151808301909252808252600092820192909252610fc960408051606080820183529181019182529081526000602082015290565b604080518082018252606080825260006020808401829052845161012081018652828152908101879052938401859052908301829052608083018a905260a0830181905260c0830181905260e083015261010082018890529061102b81610afc565b96505050505050505b92915050565b611042611eb6565b6040805160608101825260008082526020820181905291810182905283919060008060006110718a8a886115d3565b965094506110808a8a886117ec565b9650935061108f8a8a886115d3565b9650925061109e8a8a886115d3565b965091506110ad8a8a886115ef565b6040805160a08101825297885260208801969096529486019390935250606084015263ffffffff16608083015290969095509350505050565b60408051602081019091526060815281600061110386868461134a565b92509050600060ff821667ffffffffffffffff811115611125576111256123b6565b60405190808252806020026020018201604052801561114e578160200160208202803683370190505b50905060005b8260ff168160ff1610156111a55761116d8888866115d3565b838360ff168151811061118257611182612a03565b60200260200101819650828152505050808061119d90612a19565b915050611154565b5060405180602001604052808281525093505050935093915050565b600061120c84846111d185611868565b6040518060400160405280601381526020017f4d6f64756c65206d65726b6c6520747265653a00000000000000000000000000815250611903565b949350505050565b6040805180820190915260008082526020820152816000806112378787856119d8565b93509150611246878785611a31565b6040805180820190915261ffff90941684526020840191909152919791965090945050505050565b600061120c848461127e85611a86565b6040518060400160405280601881526020017f496e737472756374696f6e206d65726b6c6520747265653a0000000000000000815250611903565b6040517f46756e6374696f6e3a000000000000000000000000000000000000000000000060208201526029810182905260009081906049016040516020818303038152906040528051906020012090506107b08585836040518060400160405280601581526020017f46756e6374696f6e206d65726b6c6520747265653a0000000000000000000000815250611903565b60008184848281811061135f5761135f612a03565b919091013560f81c925081905061137581612a38565b915050935093915050565b6040805160608082018352918101918252908152600060208201528160006113a98686846115d3565b9250905060006113ba878785611a31565b9350905060008167ffffffffffffffff8111156113d9576113d96123b6565b60405190808252806020026020018201604052801561141e57816020015b60408051808201909152600080825260208201528152602001906001900390816113f75790505b50905060005b815181101561146c57611438898987611af9565b83838151811061144a5761144a612a03565b602002602001018197508290525050808061146490612a38565b915050611424565b50604080516060810182529081019182529081526020810192909252509590945092505050565b6040805180820190915260608152600060208201528160006114b68686846115d3565b9250905060608686848181106114ce576114ce612a03565b909101357fff000000000000000000000000000000000000000000000000000000000000001615905061156e578261150581612a38565b604080516001808252818301909252919550909150816020015b611527611f43565b81526020019060019003908161151f579050509050611547878785611c04565b8260008151811061155a5761155a612a03565b6020026020010181955082905250506115b2565b8261157881612a38565b604080516000808252602082019092529195509091506115ae565b61159b611f43565b8152602001906001900390816115935790505b5090505b60405180604001604052808281526020018381525093505050935093915050565b600081816115e2868684611a31565b9097909650945050505050565b600081815b600481101561164a5760088363ffffffff16901b925085858381811061161c5761161c612a03565b919091013560f81c9390931792508161163481612a38565b925050808061164290612a38565b9150506115f4565b50935093915050565b60208101518151515160005b818110156116e257835161167c906116779083611c9d565b611cd5565b6040517f56616c756520737461636b3a00000000000000000000000000000000000000006020820152602c810191909152604c8101849052606c0160405160208183030381529060405280519060200120925080806116da90612a38565b91505061165f565b5050919050565b602081015160005b825151811015611787576117218360000151828151811061171457611714612a03565b6020026020010151611cf2565b6040517f537461636b206672616d6520737461636b3a00000000000000000000000000006020820152603281019190915260528101839052607201604051602081830303815290604052805190602001209150808061177f90612a38565b9150506116f1565b50919050565b600081815b600881101561164a5760088367ffffffffffffffff16901b92508585838181106117be576117be612a03565b919091013560f81c939093179250816117d681612a38565b92505080806117e490612a38565b915050611792565b6040805160608101825260008082526020820181905291810191909152816000808061181988888661178d565b9450925061182888888661178d565b945091506118378888866115d3565b6040805160608101825267ffffffffffffffff96871681529490951660208501529383015250969095509350505050565b6000816000015161187c8360200151611d8b565b6040848101516060860151608087015192517f4d6f64756c653a000000000000000000000000000000000000000000000000006020820152602781019590955260478501939093526067840152608783019190915260e01b7fffffffff000000000000000000000000000000000000000000000000000000001660a782015260ab0161082f565b8160005b8551518110156119cf578460011660000361196b5782828760000151838151811061193457611934612a03565b602002602001015160405160200161194e93929190612a70565b6040516020818303038152906040528051906020012091506119b6565b828660000151828151811061198257611982612a03565b60200260200101518360405160200161199d93929190612a70565b6040516020818303038152906040528051906020012091505b60019490941c93806119c781612a38565b915050611907565b50949350505050565b600081815b600281101561164a5760088361ffff16901b9250858583818110611a0357611a03612a03565b919091013560f81c93909317925081611a1b81612a38565b9250508080611a2990612a38565b9150506119dd565b600081815b602081101561164a57600883901b9250858583818110611a5857611a58612a03565b919091013560f81c93909317925081611a7081612a38565b9250508080611a7e90612a38565b915050611a36565b60008160000151826020015160405160200161082f9291907f496e737472756374696f6e3a0000000000000000000000000000000000000000815260f09290921b7fffff00000000000000000000000000000000000000000000000000000000000016600c830152600e820152602e0190565b6040805180820190915260008082526020820152816000858583818110611b2257611b22612a03565b919091013560f81c9150829050611b3881612a38565b925050611b43600690565b6006811115611b5457611b54612014565b60ff168160ff161115611ba95760405162461bcd60e51b815260206004820152600e60248201527f4241445f56414c55455f5459504500000000000000000000000000000000000060448201526064016101ae565b6000611bb6878785611a31565b809450819250505060405180604001604052808360ff166006811115611bde57611bde612014565b6006811115611bef57611bef612014565b81526020018281525093505050935093915050565b611c0c611f43565b81611c27604080518082019091526000808252602082015290565b6000806000611c37898987611af9565b95509350611c468989876115d3565b95509250611c558989876115ef565b95509150611c648989876115ef565b60408051608081018252968752602087019590955263ffffffff9384169486019490945290911660608401525090969095509350505050565b60408051808201909152600080825260208201528251805183908110611cc557611cc5612a03565b6020026020010151905092915050565b60008160000151826020015160405160200161082f929190612aa7565b6000611d018260000151611cd5565b602080840151604080860151606087015191517f537461636b206672616d653a000000000000000000000000000000000000000094810194909452602c840194909452604c8301919091527fffffffff0000000000000000000000000000000000000000000000000000000060e093841b8116606c840152921b909116607082015260740161082f565b805160208083015160408085015190517f4d656d6f72793a00000000000000000000000000000000000000000000000000938101939093527fffffffffffffffff00000000000000000000000000000000000000000000000060c094851b811660278501529190931b16602f820152603781019190915260009060570161082f565b6040805161012081019091528060008152602001611e4260408051606080820183529181019182529081526000602082015290565b8152602001611e6860408051606080820183529181019182529081526000602082015290565b8152602001611e8d604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b6040805160a0810182526000808252825160608101845281815260208181018390529381019190915290918201905b81526000602082018190526040820181905260609091015290565b6040518060400160405280611f13611f25565b8152602001611f20611f25565b905290565b60405180604001604052806002906020820280368337509192915050565b6040805160c0810190915260006080820181815260a08301919091528190611ee5565b600080600080600085870360c0811215611f7f57600080fd5b6060811215611f8d57600080fd5b50859450606086013593506080860135925060a086013567ffffffffffffffff80821115611fba57600080fd5b818801915088601f830112611fce57600080fd5b813581811115611fdd57600080fd5b896020828501011115611fef57600080fd5b9699959850939650602001949392505050565b600060a0828403121561178757600080fd5b634e487b7160e01b600052602160045260246000fd5b634e487b7160e01b600052601160045260246000fd5b808201808211156110345761103461202a565b6000808585111561206357600080fd5b8386111561207057600080fd5b5050820193919092039150565b63ffffffff81811683821601908082111561209a5761209a61202a565b5092915050565b61ffff81811683821601908082111561209a5761209a61202a565b600381106120cc576120cc612014565b9052565b8051600781106120e2576120e2612014565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b8084101561213f5761212b8286516120d0565b938201936001939093019290850190612118565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b828110156121be57845161218a8582516120d0565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101612175565b509687015197909601969096525093949350505050565b818352818160208501375060006020828401015260006020601f19601f840116840101905092915050565b60006101c08835835260208901356001600160a01b03811680821461222457600080fd5b80602086015250506040890135604084015280606084015261224981840189516120bc565b5060208701516101206101e08401526122666102e08401826120ef565b905060408801517ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe4080858403016102008601526122a383836120ef565b925060608a015191508085840301610220860152506122c28282612153565b915050608088015161024084015260a08801516122e861026085018263ffffffff169052565b5060c088015163ffffffff81166102808501525060e088015163ffffffff81166102a0850152506101008801516102c084015261237e608084018880518252602081015167ffffffffffffffff80825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b855161ffff1661016084015260208601516101808401528281036101a08401526123a98185876121d5565b9998505050505050505050565b634e487b7160e01b600052604160045260246000fd5b6040805190810167ffffffffffffffff811182821017156123ef576123ef6123b6565b60405290565b6040516020810167ffffffffffffffff811182821017156123ef576123ef6123b6565b6040516080810167ffffffffffffffff811182821017156123ef576123ef6123b6565b60405160a0810167ffffffffffffffff811182821017156123ef576123ef6123b6565b6040516060810167ffffffffffffffff811182821017156123ef576123ef6123b6565b604051610120810167ffffffffffffffff811182821017156123ef576123ef6123b6565b604051601f8201601f1916810167ffffffffffffffff811182821017156124ce576124ce6123b6565b604052919050565b600381106124e357600080fd5b50565b8051610907816124d6565b600067ffffffffffffffff82111561250b5761250b6123b6565b5060051b60200190565b60006040828403121561252757600080fd5b61252f6123cc565b905081516007811061254057600080fd5b808252506020820151602082015292915050565b6000604080838503121561256757600080fd5b61256f6123cc565b9150825167ffffffffffffffff8082111561258957600080fd5b8185019150602080838803121561259f57600080fd5b6125a76123f5565b8351838111156125b657600080fd5b80850194505087601f8501126125cb57600080fd5b835192506125e06125db846124f1565b6124a5565b83815260069390931b840182019282810190898511156125ff57600080fd5b948301945b84861015612625576126168a87612515565b82529486019490830190612604565b8252508552948501519484019490945250909392505050565b805163ffffffff8116811461090757600080fd5b6000604080838503121561266557600080fd5b61266d6123cc565b9150825167ffffffffffffffff81111561268657600080fd5b8301601f8101851361269757600080fd5b805160206126a76125db836124f1565b82815260a092830284018201928282019190898511156126c657600080fd5b948301945b8486101561272f5780868b0312156126e35760008081fd5b6126eb612418565b6126f58b88612515565b81528787015185820152606061270c81890161263e565b8983015261271c6080890161263e565b90820152835294850194918301916126cb565b50808752505080860151818601525050505092915050565b67ffffffffffffffff811681146124e357600080fd5b600081830360e081121561277057600080fd5b61277861243b565b9150825182526060601f198201121561279057600080fd5b5061279961245e565b60208301516127a781612747565b815260408301516127b781612747565b8060208301525060608301516040820152806020830152506080820151604082015260a082015160608201526127ef60c0830161263e565b608082015292915050565b60008061010080848603121561280f57600080fd5b835167ffffffffffffffff8082111561282757600080fd5b90850190610120828803121561283c57600080fd5b612844612481565b61284d836124e6565b815260208301518281111561286157600080fd5b61286d89828601612554565b60208301525060408301518281111561288557600080fd5b61289189828601612554565b6040830152506060830151828111156128a957600080fd5b6128b589828601612652565b606083015250608083015160808201526128d160a0840161263e565b60a08201526128e260c0840161263e565b60c08201526128f360e0840161263e565b60e0820152838301518482015280955050505050612914846020850161275d565b90509250929050565b60006020828403121561292f57600080fd5b813561293a816124d6565b9392505050565b60006080828403121561295357600080fd5b61295b6123cc565b83601f84011261296a57600080fd5b6129726123cc565b80604085018681111561298457600080fd5b855b8181101561299e578035845260209384019301612986565b5081845286605f8701126129b157600080fd5b6129b96123cc565b925082915060808601878111156129cf57600080fd5b808210156129f45781356129e281612747565b845260209384019391909101906129cf565b50506020830152509392505050565b634e487b7160e01b600052603260045260246000fd5b600060ff821660ff8103612a2f57612a2f61202a565b60010192915050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203612a6957612a6961202a565b5060010190565b6000845160005b81811015612a915760208188018101518583015201612a77565b5091909101928352506020820152604001919050565b7f56616c75653a00000000000000000000000000000000000000000000000000008152600060078410612adc57612adc612014565b5060f89290921b600683015260078201526027019056fea26469706673582212209418f3a323f28f9fcfc55e8ecd1a364476b75046066acd7f64c37249f0ccb94a64736f6c63430008110033",
}

// OneStepProofEntryABI is the input ABI used to generate the binding from.
// Deprecated: Use OneStepProofEntryMetaData.ABI instead.
var OneStepProofEntryABI = OneStepProofEntryMetaData.ABI

// OneStepProofEntryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OneStepProofEntryMetaData.Bin instead.
var OneStepProofEntryBin = OneStepProofEntryMetaData.Bin

// DeployOneStepProofEntry deploys a new Ethereum contract, binding an instance of OneStepProofEntry to it.
func DeployOneStepProofEntry(auth *bind.TransactOpts, backend bind.ContractBackend, prover0_ common.Address, proverMem_ common.Address, proverMath_ common.Address, proverHostIo_ common.Address) (common.Address, *types.Transaction, *OneStepProofEntry, error) {
	parsed, err := OneStepProofEntryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OneStepProofEntryBin), backend, prover0_, proverMem_, proverMath_, proverHostIo_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OneStepProofEntry{OneStepProofEntryCaller: OneStepProofEntryCaller{contract: contract}, OneStepProofEntryTransactor: OneStepProofEntryTransactor{contract: contract}, OneStepProofEntryFilterer: OneStepProofEntryFilterer{contract: contract}}, nil
}

// OneStepProofEntry is an auto generated Go binding around an Ethereum contract.
type OneStepProofEntry struct {
	OneStepProofEntryCaller     // Read-only binding to the contract
	OneStepProofEntryTransactor // Write-only binding to the contract
	OneStepProofEntryFilterer   // Log filterer for contract events
}

// OneStepProofEntryCaller is an auto generated read-only Go binding around an Ethereum contract.
type OneStepProofEntryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProofEntryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OneStepProofEntryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProofEntryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OneStepProofEntryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProofEntrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OneStepProofEntrySession struct {
	Contract     *OneStepProofEntry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// OneStepProofEntryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OneStepProofEntryCallerSession struct {
	Contract *OneStepProofEntryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// OneStepProofEntryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OneStepProofEntryTransactorSession struct {
	Contract     *OneStepProofEntryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// OneStepProofEntryRaw is an auto generated low-level Go binding around an Ethereum contract.
type OneStepProofEntryRaw struct {
	Contract *OneStepProofEntry // Generic contract binding to access the raw methods on
}

// OneStepProofEntryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OneStepProofEntryCallerRaw struct {
	Contract *OneStepProofEntryCaller // Generic read-only contract binding to access the raw methods on
}

// OneStepProofEntryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OneStepProofEntryTransactorRaw struct {
	Contract *OneStepProofEntryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOneStepProofEntry creates a new instance of OneStepProofEntry, bound to a specific deployed contract.
func NewOneStepProofEntry(address common.Address, backend bind.ContractBackend) (*OneStepProofEntry, error) {
	contract, err := bindOneStepProofEntry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OneStepProofEntry{OneStepProofEntryCaller: OneStepProofEntryCaller{contract: contract}, OneStepProofEntryTransactor: OneStepProofEntryTransactor{contract: contract}, OneStepProofEntryFilterer: OneStepProofEntryFilterer{contract: contract}}, nil
}

// NewOneStepProofEntryCaller creates a new read-only instance of OneStepProofEntry, bound to a specific deployed contract.
func NewOneStepProofEntryCaller(address common.Address, caller bind.ContractCaller) (*OneStepProofEntryCaller, error) {
	contract, err := bindOneStepProofEntry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProofEntryCaller{contract: contract}, nil
}

// NewOneStepProofEntryTransactor creates a new write-only instance of OneStepProofEntry, bound to a specific deployed contract.
func NewOneStepProofEntryTransactor(address common.Address, transactor bind.ContractTransactor) (*OneStepProofEntryTransactor, error) {
	contract, err := bindOneStepProofEntry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProofEntryTransactor{contract: contract}, nil
}

// NewOneStepProofEntryFilterer creates a new log filterer instance of OneStepProofEntry, bound to a specific deployed contract.
func NewOneStepProofEntryFilterer(address common.Address, filterer bind.ContractFilterer) (*OneStepProofEntryFilterer, error) {
	contract, err := bindOneStepProofEntry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OneStepProofEntryFilterer{contract: contract}, nil
}

// bindOneStepProofEntry binds a generic wrapper to an already deployed contract.
func bindOneStepProofEntry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OneStepProofEntryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProofEntry *OneStepProofEntryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProofEntry.Contract.OneStepProofEntryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProofEntry *OneStepProofEntryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProofEntry.Contract.OneStepProofEntryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProofEntry *OneStepProofEntryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProofEntry.Contract.OneStepProofEntryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProofEntry *OneStepProofEntryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProofEntry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProofEntry *OneStepProofEntryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProofEntry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProofEntry *OneStepProofEntryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProofEntry.Contract.contract.Transact(opts, method, params...)
}

// GetMachineHash is a free data retrieval call binding the contract method 0xc39619c4.
//
// Solidity: function getMachineHash(((bytes32[2],uint64[2]),uint8) execState) pure returns(bytes32)
func (_OneStepProofEntry *OneStepProofEntryCaller) GetMachineHash(opts *bind.CallOpts, execState ExecutionState) ([32]byte, error) {
	var out []interface{}
	err := _OneStepProofEntry.contract.Call(opts, &out, "getMachineHash", execState)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetMachineHash is a free data retrieval call binding the contract method 0xc39619c4.
//
// Solidity: function getMachineHash(((bytes32[2],uint64[2]),uint8) execState) pure returns(bytes32)
func (_OneStepProofEntry *OneStepProofEntrySession) GetMachineHash(execState ExecutionState) ([32]byte, error) {
	return _OneStepProofEntry.Contract.GetMachineHash(&_OneStepProofEntry.CallOpts, execState)
}

// GetMachineHash is a free data retrieval call binding the contract method 0xc39619c4.
//
// Solidity: function getMachineHash(((bytes32[2],uint64[2]),uint8) execState) pure returns(bytes32)
func (_OneStepProofEntry *OneStepProofEntryCallerSession) GetMachineHash(execState ExecutionState) ([32]byte, error) {
	return _OneStepProofEntry.Contract.GetMachineHash(&_OneStepProofEntry.CallOpts, execState)
}

// ProveOneStep is a free data retrieval call binding the contract method 0xb5112fd2.
//
// Solidity: function proveOneStep((uint256,address,bytes32) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_OneStepProofEntry *OneStepProofEntryCaller) ProveOneStep(opts *bind.CallOpts, execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	var out []interface{}
	err := _OneStepProofEntry.contract.Call(opts, &out, "proveOneStep", execCtx, machineStep, beforeHash, proof)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProveOneStep is a free data retrieval call binding the contract method 0xb5112fd2.
//
// Solidity: function proveOneStep((uint256,address,bytes32) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_OneStepProofEntry *OneStepProofEntrySession) ProveOneStep(execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	return _OneStepProofEntry.Contract.ProveOneStep(&_OneStepProofEntry.CallOpts, execCtx, machineStep, beforeHash, proof)
}

// ProveOneStep is a free data retrieval call binding the contract method 0xb5112fd2.
//
// Solidity: function proveOneStep((uint256,address,bytes32) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_OneStepProofEntry *OneStepProofEntryCallerSession) ProveOneStep(execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	return _OneStepProofEntry.Contract.ProveOneStep(&_OneStepProofEntry.CallOpts, execCtx, machineStep, beforeHash, proof)
}

// Prover0 is a free data retrieval call binding the contract method 0x30a5509f.
//
// Solidity: function prover0() view returns(address)
func (_OneStepProofEntry *OneStepProofEntryCaller) Prover0(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OneStepProofEntry.contract.Call(opts, &out, "prover0")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Prover0 is a free data retrieval call binding the contract method 0x30a5509f.
//
// Solidity: function prover0() view returns(address)
func (_OneStepProofEntry *OneStepProofEntrySession) Prover0() (common.Address, error) {
	return _OneStepProofEntry.Contract.Prover0(&_OneStepProofEntry.CallOpts)
}

// Prover0 is a free data retrieval call binding the contract method 0x30a5509f.
//
// Solidity: function prover0() view returns(address)
func (_OneStepProofEntry *OneStepProofEntryCallerSession) Prover0() (common.Address, error) {
	return _OneStepProofEntry.Contract.Prover0(&_OneStepProofEntry.CallOpts)
}

// ProverHostIo is a free data retrieval call binding the contract method 0x5f52fd7c.
//
// Solidity: function proverHostIo() view returns(address)
func (_OneStepProofEntry *OneStepProofEntryCaller) ProverHostIo(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OneStepProofEntry.contract.Call(opts, &out, "proverHostIo")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProverHostIo is a free data retrieval call binding the contract method 0x5f52fd7c.
//
// Solidity: function proverHostIo() view returns(address)
func (_OneStepProofEntry *OneStepProofEntrySession) ProverHostIo() (common.Address, error) {
	return _OneStepProofEntry.Contract.ProverHostIo(&_OneStepProofEntry.CallOpts)
}

// ProverHostIo is a free data retrieval call binding the contract method 0x5f52fd7c.
//
// Solidity: function proverHostIo() view returns(address)
func (_OneStepProofEntry *OneStepProofEntryCallerSession) ProverHostIo() (common.Address, error) {
	return _OneStepProofEntry.Contract.ProverHostIo(&_OneStepProofEntry.CallOpts)
}

// ProverMath is a free data retrieval call binding the contract method 0x66e5d9c3.
//
// Solidity: function proverMath() view returns(address)
func (_OneStepProofEntry *OneStepProofEntryCaller) ProverMath(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OneStepProofEntry.contract.Call(opts, &out, "proverMath")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProverMath is a free data retrieval call binding the contract method 0x66e5d9c3.
//
// Solidity: function proverMath() view returns(address)
func (_OneStepProofEntry *OneStepProofEntrySession) ProverMath() (common.Address, error) {
	return _OneStepProofEntry.Contract.ProverMath(&_OneStepProofEntry.CallOpts)
}

// ProverMath is a free data retrieval call binding the contract method 0x66e5d9c3.
//
// Solidity: function proverMath() view returns(address)
func (_OneStepProofEntry *OneStepProofEntryCallerSession) ProverMath() (common.Address, error) {
	return _OneStepProofEntry.Contract.ProverMath(&_OneStepProofEntry.CallOpts)
}

// ProverMem is a free data retrieval call binding the contract method 0x1f128bc0.
//
// Solidity: function proverMem() view returns(address)
func (_OneStepProofEntry *OneStepProofEntryCaller) ProverMem(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OneStepProofEntry.contract.Call(opts, &out, "proverMem")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProverMem is a free data retrieval call binding the contract method 0x1f128bc0.
//
// Solidity: function proverMem() view returns(address)
func (_OneStepProofEntry *OneStepProofEntrySession) ProverMem() (common.Address, error) {
	return _OneStepProofEntry.Contract.ProverMem(&_OneStepProofEntry.CallOpts)
}

// ProverMem is a free data retrieval call binding the contract method 0x1f128bc0.
//
// Solidity: function proverMem() view returns(address)
func (_OneStepProofEntry *OneStepProofEntryCallerSession) ProverMem() (common.Address, error) {
	return _OneStepProofEntry.Contract.ProverMem(&_OneStepProofEntry.CallOpts)
}

// OneStepProofEntryLibMetaData contains all meta data concerning the OneStepProofEntryLib contract.
var OneStepProofEntryLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212201164eb13ea71df8bf202477ed5677c94a6a12b52a8d5eeed231d865b9d1ead7864736f6c63430008110033",
}

// OneStepProofEntryLibABI is the input ABI used to generate the binding from.
// Deprecated: Use OneStepProofEntryLibMetaData.ABI instead.
var OneStepProofEntryLibABI = OneStepProofEntryLibMetaData.ABI

// OneStepProofEntryLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OneStepProofEntryLibMetaData.Bin instead.
var OneStepProofEntryLibBin = OneStepProofEntryLibMetaData.Bin

// DeployOneStepProofEntryLib deploys a new Ethereum contract, binding an instance of OneStepProofEntryLib to it.
func DeployOneStepProofEntryLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OneStepProofEntryLib, error) {
	parsed, err := OneStepProofEntryLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OneStepProofEntryLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OneStepProofEntryLib{OneStepProofEntryLibCaller: OneStepProofEntryLibCaller{contract: contract}, OneStepProofEntryLibTransactor: OneStepProofEntryLibTransactor{contract: contract}, OneStepProofEntryLibFilterer: OneStepProofEntryLibFilterer{contract: contract}}, nil
}

// OneStepProofEntryLib is an auto generated Go binding around an Ethereum contract.
type OneStepProofEntryLib struct {
	OneStepProofEntryLibCaller     // Read-only binding to the contract
	OneStepProofEntryLibTransactor // Write-only binding to the contract
	OneStepProofEntryLibFilterer   // Log filterer for contract events
}

// OneStepProofEntryLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type OneStepProofEntryLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProofEntryLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OneStepProofEntryLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProofEntryLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OneStepProofEntryLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProofEntryLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OneStepProofEntryLibSession struct {
	Contract     *OneStepProofEntryLib // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// OneStepProofEntryLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OneStepProofEntryLibCallerSession struct {
	Contract *OneStepProofEntryLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// OneStepProofEntryLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OneStepProofEntryLibTransactorSession struct {
	Contract     *OneStepProofEntryLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// OneStepProofEntryLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type OneStepProofEntryLibRaw struct {
	Contract *OneStepProofEntryLib // Generic contract binding to access the raw methods on
}

// OneStepProofEntryLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OneStepProofEntryLibCallerRaw struct {
	Contract *OneStepProofEntryLibCaller // Generic read-only contract binding to access the raw methods on
}

// OneStepProofEntryLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OneStepProofEntryLibTransactorRaw struct {
	Contract *OneStepProofEntryLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOneStepProofEntryLib creates a new instance of OneStepProofEntryLib, bound to a specific deployed contract.
func NewOneStepProofEntryLib(address common.Address, backend bind.ContractBackend) (*OneStepProofEntryLib, error) {
	contract, err := bindOneStepProofEntryLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OneStepProofEntryLib{OneStepProofEntryLibCaller: OneStepProofEntryLibCaller{contract: contract}, OneStepProofEntryLibTransactor: OneStepProofEntryLibTransactor{contract: contract}, OneStepProofEntryLibFilterer: OneStepProofEntryLibFilterer{contract: contract}}, nil
}

// NewOneStepProofEntryLibCaller creates a new read-only instance of OneStepProofEntryLib, bound to a specific deployed contract.
func NewOneStepProofEntryLibCaller(address common.Address, caller bind.ContractCaller) (*OneStepProofEntryLibCaller, error) {
	contract, err := bindOneStepProofEntryLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProofEntryLibCaller{contract: contract}, nil
}

// NewOneStepProofEntryLibTransactor creates a new write-only instance of OneStepProofEntryLib, bound to a specific deployed contract.
func NewOneStepProofEntryLibTransactor(address common.Address, transactor bind.ContractTransactor) (*OneStepProofEntryLibTransactor, error) {
	contract, err := bindOneStepProofEntryLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProofEntryLibTransactor{contract: contract}, nil
}

// NewOneStepProofEntryLibFilterer creates a new log filterer instance of OneStepProofEntryLib, bound to a specific deployed contract.
func NewOneStepProofEntryLibFilterer(address common.Address, filterer bind.ContractFilterer) (*OneStepProofEntryLibFilterer, error) {
	contract, err := bindOneStepProofEntryLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OneStepProofEntryLibFilterer{contract: contract}, nil
}

// bindOneStepProofEntryLib binds a generic wrapper to an already deployed contract.
func bindOneStepProofEntryLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OneStepProofEntryLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProofEntryLib *OneStepProofEntryLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProofEntryLib.Contract.OneStepProofEntryLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProofEntryLib *OneStepProofEntryLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProofEntryLib.Contract.OneStepProofEntryLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProofEntryLib *OneStepProofEntryLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProofEntryLib.Contract.OneStepProofEntryLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProofEntryLib *OneStepProofEntryLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProofEntryLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProofEntryLib *OneStepProofEntryLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProofEntryLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProofEntryLib *OneStepProofEntryLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProofEntryLib.Contract.contract.Transact(opts, method, params...)
}

// OneStepProver0MetaData contains all meta data concerning the OneStepProver0 contract.
var OneStepProver0MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"initialWasmModuleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionContext\",\"name\":\"\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"startMach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"startMod\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"opcode\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"argumentData\",\"type\":\"uint256\"}],\"internalType\":\"structInstruction\",\"name\":\"inst\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"mach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"mod\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50612d08806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c806397cc779a14610030575b600080fd5b61004361003e3660046122f0565b61005a565b6040516100519291906124f9565b60405180910390f35b61006261216f565b61006a612218565b610073876129ca565b915061008436879003870187612acf565b905060006100956020870187612b66565b905061226261ffff82166100ac575061046a61044c565b60001961ffff8316016100c2575061047561044c565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff161ffff8316016100f6575061047c61044c565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff061ffff83160161012a57506105b561044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7ff761ffff83160161015e575061073361044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7ff661ffff831601610192575061089761044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffef61ffff8316016101c65750610a3b61044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7ffd61ffff8316016101fa5750610f3b61044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7ffc61ffff83160161022e5750610faa61044c565b601f1961ffff831601610244575061103861044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffdf61ffff831601610278575061107a61044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffdd61ffff8316016102ac57506110bf61044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffdc61ffff8316016102e057506110e761044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7ffe61ffff831601610314575061111761044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe661ffff83160161034857506111b461044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe561ffff83160161037c57506111c161044c565b604161ffff8316108015906103965750604461ffff831611155b156103a4575061123061044c565b61ffff821661800514806103bd575061ffff8216618006145b156103cb57506113a161044c565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7ff861ffff8316016103ff575061147261044c565b60405162461bcd60e51b815260206004820152600e60248201527f494e56414c49445f4f50434f444500000000000000000000000000000000000060448201526064015b60405180910390fd5b61045d84848989898663ffffffff16565b5050965096945050505050565b505060029092525050565b5050505050565b600061048b8660600151611481565b9050600481515160068111156104a3576104a36123ca565b036104c9578560025b908160028111156104bf576104bf6123ca565b8152505050610475565b600681515160068111156104df576104df6123ca565b1461052c5760405162461bcd60e51b815260206004820152601660248201527f494e56414c49445f52455455524e5f50435f54595045000000000000000000006044820152606401610443565b805160209081015190819081901c604082901c606083901c156105915760405162461bcd60e51b815260206004820152601660248201527f494e56414c49445f52455455524e5f50435f44415441000000000000000000006044820152606401610443565b63ffffffff92831660e08b015290821660c08a01521660a088015250505050505050565b604080518082018252600080825260209182015260e087015160c088015160a0890151845180860186526006815263ffffffff90931691841b67ffffffff000000001691909117931b6bffffffff0000000000000000169290921790820152610624905b602087015190611552565b60006106338660600151611562565b905061067e6106738260400151604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b602088015190611552565b6106bc6106738260600151604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b602084013563ffffffff811681146107165760405162461bcd60e51b815260206004820152600d60248201527f4241445f43414c4c5f44415441000000000000000000000000000000000000006044820152606401610443565b63ffffffff1660c08701525050600060e090940193909352505050565b604080518082018252600080825260209182015260e087015160c088015160a0890151845180860186526006815263ffffffff90931691841b67ffffffff000000001691909117931b6bffffffff000000000000000016929092179082015261079b90610619565b6107d96106198660a00151604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b6108176106198560800151604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b6020808401359081901c604082901c156108735760405162461bcd60e51b815260206004820152601a60248201527f4241445f43524f53535f4d4f44554c455f43414c4c5f444154410000000000006044820152606401610443565b63ffffffff90811660a08801521660c08601525050600060e0909301929092525050565b604080518082018252600080825260209182015260e087015160c088015160a0890151845180860186526006815263ffffffff90931691841b67ffffffff000000001691909117931b6bffffffff00000000000000001692909217908201526108ff90610619565b61093d6106198660a00151604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b61097b6106198560800151604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b600061098a8660600151611562565b9050806060015163ffffffff166000036109a6578560026104ac565b602084013563ffffffff81168114610a005760405162461bcd60e51b815260206004820152601d60248201527f4241445f43414c4c45525f494e5445524e414c5f43414c4c5f444154410000006044820152606401610443565b604082015163ffffffff1660a08801526060820151610a20908290612ba0565b63ffffffff1660c08801525050600060e08601525050505050565b600080610a53610a4e88602001516115de565b611603565b9050600080600080806000610a746040518060200160405280606081525090565b610a7f8b8b876116c0565b95509350610a8e8b8b87611728565b9096509450610a9e8b8b87611744565b95509250610aad8b8b876116c0565b95509150610abc8b8b87611728565b9097509450610acc8b8b8761177a565b6040517f43616c6c20696e6469726563743a00000000000000000000000000000000000060208201527fffffffffffffffff00000000000000000000000000000000000000000000000060c088901b16602e8201526036810189905290965090915060009060560160408051601f19818403018152919052805160209182012091508d01358114610b9f5760405162461bcd60e51b815260206004820152601660248201527f4241445f43414c4c5f494e4449524543545f44415441000000000000000000006044820152606401610443565b610bb68267ffffffffffffffff871686868c611855565b90508d604001518114610c0b5760405162461bcd60e51b815260206004820152600f60248201527f4241445f5441424c45535f524f4f5400000000000000000000000000000000006044820152606401610443565b8267ffffffffffffffff168963ffffffff1610610c3657505060028d52506104759650505050505050565b50505050506000610c57604080518082019091526000808252602082015290565b604080516020810190915260608152610c718a8a86611728565b94509250610c808a8a86611949565b94509150610c8f8a8a8661177a565b945090506000610cac8263ffffffff808b169087908790611a5416565b9050868114610cfd5760405162461bcd60e51b815260206004820152601160248201527f4241445f454c454d454e54535f524f4f540000000000000000000000000000006044820152606401610443565b858414610d2d578d60025b90816002811115610d1b57610d1b6123ca565b81525050505050505050505050610475565b600483516006811115610d4257610d426123ca565b03610d4f578d6002610d08565b600583516006811115610d6457610d646123ca565b03610dca576020830151985063ffffffff89168914610dc55760405162461bcd60e51b815260206004820152601560248201527f4241445f46554e435f5245465f434f4e54454e545300000000000000000000006044820152606401610443565b610e12565b60405162461bcd60e51b815260206004820152600d60248201527f4241445f454c454d5f54595045000000000000000000000000000000000000006044820152606401610443565b5050505050505050610e8961067387604080518082018252600080825260209182015260e083015160c084015160a090940151835180850185526006815263ffffffff90921694831b67ffffffff0000000016949094179390921b6bffffffff000000000000000016929092179181019190915290565b6000610e988760600151611562565b9050610ee3610ed88260400151604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b602089015190611552565b610f21610ed88260600151604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b5063ffffffff1660c0860152600060e08601525050505050565b602083013563ffffffff81168114610f955760405162461bcd60e51b815260206004820152600d60248201527f4241445f43414c4c5f44415441000000000000000000000000000000000000006044820152606401610443565b63ffffffff1660e09095019490945250505050565b6000610fbc610a4e87602001516115de565b905063ffffffff81161561103057602084013563ffffffff811681146110245760405162461bcd60e51b815260206004820152600d60248201527f4241445f43414c4c5f44415441000000000000000000000000000000000000006044820152606401610443565b63ffffffff1660e08701525b505050505050565b60006110478660600151611562565b9050600061105f826020015186602001358686611afd565b60208801519091506110719082611552565b50505050505050565b600061108986602001516115de565b9050600061109a8760600151611562565b90506110b181602001518660200135848787611bc5565b602090910152505050505050565b60006110d5856000015185602001358585611afd565b60208701519091506110309082611552565b60006110f686602001516115de565b905061110d85600001518560200135838686611bc5565b9094525050505050565b600061112686602001516115de565b9050600061113787602001516115de565b9050600061114888602001516115de565b905060006040518060800160405280838152602001886020013560001b815260200161117385611603565b63ffffffff16815260200161118786611603565b63ffffffff1681525090506111a9818a60600151611c8f90919063ffffffff16565b505050505050505050565b61103085602001516115de565b60006111d3610a4e87602001516115de565b905060006111e487602001516115de565b905060006111f588602001516115de565b905063ffffffff8316156112175760208801516112129082611552565b611226565b60208801516112269083611552565b5050505050505050565b600061123f6020850185612b66565b905060007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffbf61ffff83160161127657506000611357565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffbe61ffff8316016112a957506001611357565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffbd61ffff8316016112dc57506002611357565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffbc61ffff83160161130f57506003611357565b60405162461bcd60e51b815260206004820152601960248201527f434f4e53545f505553485f494e56414c49445f4f50434f4445000000000000006044820152606401610443565b6110716040518060400160405280836006811115611377576113776123ca565b8152602001876020013567ffffffffffffffff16815250886020015161155290919063ffffffff16565b60408051808201909152600080825260208201526180056113c56020860186612b66565b61ffff16036113f2576113db86602001516115de565b60408701519091506113ed9082611552565b611030565b6180066114026020860186612b66565b61ffff160361142a5761141886604001516115de565b60208701519091506113ed9082611552565b60405162461bcd60e51b815260206004820152601c60248201527f4d4f56455f494e5445524e414c5f494e56414c49445f4f50434f4445000000006044820152606401610443565b60006110d58660200151611d77565b61148961226c565b8151516001146114db5760405162461bcd60e51b815260206004820152601160248201527f4241445f57494e444f575f4c454e4754480000000000000000000000000000006044820152606401610443565b815180516000906114ee576114ee612bc4565b60200260200101519050600067ffffffffffffffff81111561151257611512612640565b60405190808252806020026020018201604052801561154b57816020015b61153861226c565b8152602001906001900390816115305790505b5090915290565b815161155e9082611dac565b5050565b61156a61226c565b8151516001146115bc5760405162461bcd60e51b815260206004820152601160248201527f4241445f57494e444f575f4c454e4754480000000000000000000000000000006044820152606401610443565b815180516000906115cf576115cf612bc4565b60200260200101519050919050565b604080518082019091526000808252602082015281516115fd90611e76565b92915050565b6020810151600090818351600681111561161f5761161f6123ca565b1461166c5760405162461bcd60e51b815260206004820152600760248201527f4e4f545f493332000000000000000000000000000000000000000000000000006044820152606401610443565b64010000000081106115fd5760405162461bcd60e51b815260206004820152600760248201527f4241445f493332000000000000000000000000000000000000000000000000006044820152606401610443565b600081815b600881101561171f5760088367ffffffffffffffff16901b92508585838181106116f1576116f1612bc4565b919091013560f81c9390931792508161170981612bda565b925050808061171790612bda565b9150506116c5565b50935093915050565b60008181611737868684611f80565b9097909650945050505050565b60008184848281811061175957611759612bc4565b919091013560f81c925081905061176f81612bda565b915050935093915050565b604080516020810190915260608152816000611797868684611744565b92509050600060ff821667ffffffffffffffff8111156117b9576117b9612640565b6040519080825280602002602001820160405280156117e2578160200160208202803683370190505b50905060005b8260ff168160ff16101561183957611801888886611728565b838360ff168151811061181657611816612bc4565b60200260200101819650828152505050808061183190612bf4565b9150506117e8565b5060405180602001604052808281525093505050935093915050565b6040517f5461626c653a000000000000000000000000000000000000000000000000000060208201527fff0000000000000000000000000000000000000000000000000000000000000060f885901b1660268201527fffffffffffffffff00000000000000000000000000000000000000000000000060c084901b166027820152602f81018290526000908190604f0160405160208183030381529060405280519060200120905061193e8787836040518060400160405280601281526020017f5461626c65206d65726b6c6520747265653a0000000000000000000000000000815250611fd5565b979650505050505050565b604080518082019091526000808252602082015281600085858381811061197257611972612bc4565b919091013560f81c915082905061198881612bda565b925050611993600690565b60068111156119a4576119a46123ca565b60ff168160ff1611156119f95760405162461bcd60e51b815260206004820152600e60248201527f4241445f56414c55455f545950450000000000000000000000000000000000006044820152606401610443565b6000611a06878785611f80565b809450819250505060405180604001604052808360ff166006811115611a2e57611a2e6123ca565b6006811115611a3f57611a3f6123ca565b81526020018281525093505050935093915050565b60008083611a61846120aa565b6040517f5461626c6520656c656d656e743a0000000000000000000000000000000000006020820152602e810192909252604e820152606e01604051602081830303815290604052805190602001209050611af38686836040518060400160405280601a81526020017f5461626c6520656c656d656e74206d65726b6c6520747265653a000000000000815250611fd5565b9695505050505050565b60408051808201909152600080825260208201526000611b2d604080518082019091526000808252602082015290565b604080516020810190915260608152611b47868685611949565b93509150611b5686868561177a565b935090506000611b678289856120e4565b9050888114611bb85760405162461bcd60e51b815260206004820152601160248201527f57524f4e475f4d45524b4c455f524f4f540000000000000000000000000000006044820152606401610443565b5090979650505050505050565b6000611be1604080518082019091526000808252602082015290565b6000611bf96040518060200160405280606081525090565b611c04868684611949565b9093509150611c1486868461177a565b925090506000611c25828a866120e4565b9050898114611c765760405162461bcd60e51b815260206004820152601160248201527f57524f4e475f4d45524b4c455f524f4f540000000000000000000000000000006044820152606401610443565b611c81828a8a6120e4565b9a9950505050505050505050565b815151600090611ca0906001612c13565b67ffffffffffffffff811115611cb857611cb8612640565b604051908082528060200260200182016040528015611cf157816020015b611cde61226c565b815260200190600190039081611cd65790505b50905060005b835151811015611d4d578351805182908110611d1557611d15612bc4565b6020026020010151828281518110611d2f57611d2f612bc4565b60200260200101819052508080611d4590612bda565b915050611cf7565b50818184600001515181518110611d6657611d66612bc4565b602090810291909101015290915250565b604080518082019091526000808252602082015281515151611da5611d9d600183612c26565b845190612137565b9392505050565b815151600090611dbd906001612c13565b67ffffffffffffffff811115611dd557611dd5612640565b604051908082528060200260200182016040528015611e1a57816020015b6040805180820190915260008082526020820152815260200190600190039081611df35790505b50905060005b835151811015611d4d578351805182908110611e3e57611e3e612bc4565b6020026020010151828281518110611e5857611e58612bc4565b60200260200101819052508080611e6e90612bda565b915050611e20565b604080518082019091526000808252602082015281518051611e9a90600190612c26565b81518110611eaa57611eaa612bc4565b6020026020010151905060006001836000015151611ec89190612c26565b67ffffffffffffffff811115611ee057611ee0612640565b604051908082528060200260200182016040528015611f2557816020015b6040805180820190915260008082526020820152815260200190600190039081611efe5790505b50905060005b815181101561154b578351805182908110611f4857611f48612bc4565b6020026020010151828281518110611f6257611f62612bc4565b60200260200101819052508080611f7890612bda565b915050611f2b565b600081815b602081101561171f57600883901b9250858583818110611fa757611fa7612bc4565b919091013560f81c93909317925081611fbf81612bda565b9250508080611fcd90612bda565b915050611f85565b8160005b8551518110156120a1578460011660000361203d5782828760000151838151811061200657612006612bc4565b602002602001015160405160200161202093929190612c39565b604051602081830303815290604052805190602001209150612088565b828660000151828151811061205457612054612bc4565b60200260200101518360405160200161206f93929190612c39565b6040516020818303038152906040528051906020012091505b60019490941c938061209981612bda565b915050611fd9565b50949350505050565b6000816000015182602001516040516020016120c7929190612c70565b604051602081830303815290604052805190602001209050919050565b600061212f84846120f4856120aa565b6040518060400160405280601281526020017f56616c7565206d65726b6c6520747265653a0000000000000000000000000000815250611fd5565b949350505050565b6040805180820190915260008082526020820152825180518390811061215f5761215f612bc4565b6020026020010151905092915050565b60408051610120810190915280600081526020016121a460408051606080820183529181019182529081526000602082015290565b81526020016121ca60408051606080820183529181019182529081526000602082015290565b81526020016121ef604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b6040805160a0810182526000808252825160608101845281815260208181018390529381019190915290918201905b81526000602082018190526040820181905260609091015290565b61226a612cbc565b565b6040805160c0810190915260006080820181815260a08301919091528190612247565b6000604082840312156122a157600080fd5b50919050565b60008083601f8401126122b957600080fd5b50813567ffffffffffffffff8111156122d157600080fd5b6020830191508360208285010111156122e957600080fd5b9250929050565b6000806000806000808688036101c081121561230b57600080fd5b606081121561231957600080fd5b879650606088013567ffffffffffffffff8082111561233757600080fd5b90890190610120828c03121561234c57600080fd5b81975060e07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808401121561237f57600080fd5b60808a0196506123938b6101608c0161228f565b95506101a08a01359250808311156123aa57600080fd5b50506123b889828a016122a7565b979a9699509497509295939492505050565b634e487b7160e01b600052602160045260246000fd5b600381106123f0576123f06123ca565b9052565b805160078110612406576124066123ca565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b808410156124635761244f8286516123f4565b93820193600193909301929085019061243c565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b828110156124e25784516124ae8582516123f4565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101612499565b509687015197909601969096525093949350505050565b600061010080835261250e81840186516123e0565b602085015161012084810152612528610220850182612413565b905060408601517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0080868403016101408701526125658383612413565b9250606088015191508086840301610160870152506125848282612477565b915050608086015161018085015260a08601516125aa6101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e086015250908501516102008401529050611da5602083018480518252602081015167ffffffffffffffff80825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b6040805190810167ffffffffffffffff8111828210171561267957612679612640565b60405290565b6040516020810167ffffffffffffffff8111828210171561267957612679612640565b6040516080810167ffffffffffffffff8111828210171561267957612679612640565b604051610120810167ffffffffffffffff8111828210171561267957612679612640565b60405160a0810167ffffffffffffffff8111828210171561267957612679612640565b6040516060810167ffffffffffffffff8111828210171561267957612679612640565b604051601f8201601f1916810167ffffffffffffffff8111828210171561275857612758612640565b604052919050565b80356003811061276f57600080fd5b919050565b600067ffffffffffffffff82111561278e5761278e612640565b5060051b60200190565b6000604082840312156127aa57600080fd5b6127b2612656565b90508135600781106127c357600080fd5b808252506020820135602082015292915050565b600060408083850312156127ea57600080fd5b6127f2612656565b9150823567ffffffffffffffff8082111561280c57600080fd5b8185019150602080838803121561282257600080fd5b61282a61267f565b83358381111561283957600080fd5b80850194505087601f85011261284e57600080fd5b8335925061286361285e84612774565b61272f565b83815260069390931b8401820192828101908985111561288257600080fd5b948301945b848610156128a8576128998a87612798565b82529486019490830190612887565b8252508552948501359484019490945250909392505050565b803563ffffffff8116811461276f57600080fd5b600060408083850312156128e857600080fd5b6128f0612656565b9150823567ffffffffffffffff81111561290957600080fd5b8301601f8101851361291a57600080fd5b8035602061292a61285e83612774565b82815260a0928302840182019282820191908985111561294957600080fd5b948301945b848610156129b25780868b0312156129665760008081fd5b61296e6126a2565b6129788b88612798565b81528787013585820152606061298f8189016128c1565b8983015261299f608089016128c1565b908201528352948501949183019161294e565b50808752505080860135818601525050505092915050565b600061012082360312156129dd57600080fd5b6129e56126c5565b6129ee83612760565b8152602083013567ffffffffffffffff80821115612a0b57600080fd5b612a17368387016127d7565b60208401526040850135915080821115612a3057600080fd5b612a3c368387016127d7565b60408401526060850135915080821115612a5557600080fd5b50612a62368286016128d5565b60608301525060808301356080820152612a7e60a084016128c1565b60a0820152612a8f60c084016128c1565b60c0820152612aa060e084016128c1565b60e082015261010092830135928101929092525090565b803567ffffffffffffffff8116811461276f57600080fd5b600081830360e0811215612ae257600080fd5b612aea6126e9565b833581526060601f1983011215612b0057600080fd5b612b0861270c565b9150612b1660208501612ab7565b8252612b2460408501612ab7565b6020830152606084013560408301528160208201526080840135604082015260a08401356060820152612b5960c085016128c1565b6080820152949350505050565b600060208284031215612b7857600080fd5b813561ffff81168114611da557600080fd5b634e487b7160e01b600052601160045260246000fd5b63ffffffff818116838216019080821115612bbd57612bbd612b8a565b5092915050565b634e487b7160e01b600052603260045260246000fd5b60006000198203612bed57612bed612b8a565b5060010190565b600060ff821660ff8103612c0a57612c0a612b8a565b60010192915050565b808201808211156115fd576115fd612b8a565b818103818111156115fd576115fd612b8a565b6000845160005b81811015612c5a5760208188018101518583015201612c40565b5091909101928352506020820152604001919050565b7f56616c75653a00000000000000000000000000000000000000000000000000008152600060078410612ca557612ca56123ca565b5060f89290921b6006830152600782015260270190565b634e487b7160e01b600052605160045260246000fdfea2646970667358221220c8a00303bfbd425a4c1c16797ed8e891936e2bcd4519d39db3d1dc490bfb996f64736f6c63430008110033",
}

// OneStepProver0ABI is the input ABI used to generate the binding from.
// Deprecated: Use OneStepProver0MetaData.ABI instead.
var OneStepProver0ABI = OneStepProver0MetaData.ABI

// OneStepProver0Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OneStepProver0MetaData.Bin instead.
var OneStepProver0Bin = OneStepProver0MetaData.Bin

// DeployOneStepProver0 deploys a new Ethereum contract, binding an instance of OneStepProver0 to it.
func DeployOneStepProver0(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OneStepProver0, error) {
	parsed, err := OneStepProver0MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OneStepProver0Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OneStepProver0{OneStepProver0Caller: OneStepProver0Caller{contract: contract}, OneStepProver0Transactor: OneStepProver0Transactor{contract: contract}, OneStepProver0Filterer: OneStepProver0Filterer{contract: contract}}, nil
}

// OneStepProver0 is an auto generated Go binding around an Ethereum contract.
type OneStepProver0 struct {
	OneStepProver0Caller     // Read-only binding to the contract
	OneStepProver0Transactor // Write-only binding to the contract
	OneStepProver0Filterer   // Log filterer for contract events
}

// OneStepProver0Caller is an auto generated read-only Go binding around an Ethereum contract.
type OneStepProver0Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProver0Transactor is an auto generated write-only Go binding around an Ethereum contract.
type OneStepProver0Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProver0Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OneStepProver0Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProver0Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OneStepProver0Session struct {
	Contract     *OneStepProver0   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OneStepProver0CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OneStepProver0CallerSession struct {
	Contract *OneStepProver0Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// OneStepProver0TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OneStepProver0TransactorSession struct {
	Contract     *OneStepProver0Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// OneStepProver0Raw is an auto generated low-level Go binding around an Ethereum contract.
type OneStepProver0Raw struct {
	Contract *OneStepProver0 // Generic contract binding to access the raw methods on
}

// OneStepProver0CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OneStepProver0CallerRaw struct {
	Contract *OneStepProver0Caller // Generic read-only contract binding to access the raw methods on
}

// OneStepProver0TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OneStepProver0TransactorRaw struct {
	Contract *OneStepProver0Transactor // Generic write-only contract binding to access the raw methods on
}

// NewOneStepProver0 creates a new instance of OneStepProver0, bound to a specific deployed contract.
func NewOneStepProver0(address common.Address, backend bind.ContractBackend) (*OneStepProver0, error) {
	contract, err := bindOneStepProver0(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OneStepProver0{OneStepProver0Caller: OneStepProver0Caller{contract: contract}, OneStepProver0Transactor: OneStepProver0Transactor{contract: contract}, OneStepProver0Filterer: OneStepProver0Filterer{contract: contract}}, nil
}

// NewOneStepProver0Caller creates a new read-only instance of OneStepProver0, bound to a specific deployed contract.
func NewOneStepProver0Caller(address common.Address, caller bind.ContractCaller) (*OneStepProver0Caller, error) {
	contract, err := bindOneStepProver0(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProver0Caller{contract: contract}, nil
}

// NewOneStepProver0Transactor creates a new write-only instance of OneStepProver0, bound to a specific deployed contract.
func NewOneStepProver0Transactor(address common.Address, transactor bind.ContractTransactor) (*OneStepProver0Transactor, error) {
	contract, err := bindOneStepProver0(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProver0Transactor{contract: contract}, nil
}

// NewOneStepProver0Filterer creates a new log filterer instance of OneStepProver0, bound to a specific deployed contract.
func NewOneStepProver0Filterer(address common.Address, filterer bind.ContractFilterer) (*OneStepProver0Filterer, error) {
	contract, err := bindOneStepProver0(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OneStepProver0Filterer{contract: contract}, nil
}

// bindOneStepProver0 binds a generic wrapper to an already deployed contract.
func bindOneStepProver0(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OneStepProver0MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProver0 *OneStepProver0Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProver0.Contract.OneStepProver0Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProver0 *OneStepProver0Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProver0.Contract.OneStepProver0Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProver0 *OneStepProver0Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProver0.Contract.OneStepProver0Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProver0 *OneStepProver0CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProver0.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProver0 *OneStepProver0TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProver0.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProver0 *OneStepProver0TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProver0.Contract.contract.Transact(opts, method, params...)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProver0 *OneStepProver0Caller) ExecuteOneStep(opts *bind.CallOpts, arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	var out []interface{}
	err := _OneStepProver0.contract.Call(opts, &out, "executeOneStep", arg0, startMach, startMod, inst, proof)

	outstruct := new(struct {
		Mach Machine
		Mod  Module
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Mach = *abi.ConvertType(out[0], new(Machine)).(*Machine)
	outstruct.Mod = *abi.ConvertType(out[1], new(Module)).(*Module)

	return *outstruct, err

}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProver0 *OneStepProver0Session) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProver0.Contract.ExecuteOneStep(&_OneStepProver0.CallOpts, arg0, startMach, startMod, inst, proof)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProver0 *OneStepProver0CallerSession) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProver0.Contract.ExecuteOneStep(&_OneStepProver0.CallOpts, arg0, startMach, startMod, inst, proof)
}

// OneStepProverHostIoMetaData contains all meta data concerning the OneStepProverHostIo contract.
var OneStepProverHostIoMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"initialWasmModuleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"startMach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"startMod\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"opcode\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"argumentData\",\"type\":\"uint256\"}],\"internalType\":\"structInstruction\",\"name\":\"inst\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"mach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"mod\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506129f0806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c806397cc779a14610030575b600080fd5b61004361003e366004611e98565b61005a565b6040516100519291906120a1565b60405180910390f35b610062611d41565b6040805160a0810182526000808252825160608082018552828252602080830184905282860184905284019190915292820181905291810182905260808101919091526100ae87612572565b91506100bf36879003870187612677565b905060006100d0602087018761270e565b9050611dea61801061ffff8316108015906100f1575061801361ffff831611155b156100ff57506102076101e8565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7fe061ffff83160161013357506104b56101e8565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7fdf61ffff83160161016757506107916101e8565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7fde61ffff83160161019b5750610b116101e8565b60405162461bcd60e51b815260206004820152601560248201527f494e56414c49445f4d454d4f52595f4f50434f4445000000000000000000000060448201526064015b60405180910390fd5b6101fa8a85858a8a8a8763ffffffff16565b5050965096945050505050565b6000610216602085018561270e565b9050610220611df4565b600061022d858583610b1d565b60808a01518251805160209182015182860151805190840151604080517f476c6f62616c2073746174653a0000000000000000000000000000000000000081880152602d810195909552604d8501939093527fffffffffffffffff00000000000000000000000000000000000000000000000060c092831b8116606d860152911b1660758301528051808303605d018152607d909201905280519101209294509092501461031d5760405162461bcd60e51b815260206004820152601060248201527f4241445f474c4f42414c5f53544154450000000000000000000000000000000060448201526064016101df565b61ffff83166180101480610336575061ffff8316618011145b15610358576103538888848961034e8987818d612732565b610bf9565b61040a565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7fee61ffff84160161038d576103538883610da6565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7fed61ffff8416016103c2576103538883610e3e565b60405162461bcd60e51b815260206004820152601a60248201527f494e56414c49445f474c4f42414c53544154455f4f50434f444500000000000060448201526064016101df565b8151805160209182015182850151805190840151604080517f476c6f62616c2073746174653a0000000000000000000000000000000000000081880152602d810195909552604d8501939093527fffffffffffffffff00000000000000000000000000000000000000000000000060c092831b8116606d860152911b1660758301528051808303605d018152607d909201905280519101206080909801979097525050505050505050565b60006104cc6104c78760200151610eb5565b610eda565b63ffffffff16905060006104e66104c78860200151610eb5565b63ffffffff16905085602001516000015167ffffffffffffffff1681602061050e9190612772565b1180610523575061052060208261279b565b15155b1561054a578660025b9081600281111561053f5761053f611f72565b815250505050610789565b60006105576020836127af565b90506000806105726040518060200160405280606081525090565b60208a015161058490858a8a87610f97565b9094509092509050606060008989868181106105a2576105a26127c3565b919091013560f81c91508590506105b8816127d9565b9550508060ff166000036106a5573660006105d58b88818f612732565b915091508582826040516105ea929190612811565b60405180910390201461063f5760405162461bcd60e51b815260206004820152600c60248201527f4241445f505245494d414745000000000000000000000000000000000000000060448201526064016101df565b600061064c8b6020612772565b9050818111156106595750805b610665818c8486612732565b8080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152509297506106ed95505050505050565b60405162461bcd60e51b815260206004820152601660248201527f554e4b4e4f574e5f505245494d4147455f50524f4f460000000000000000000060448201526064016101df565b60005b82518110156107315761071d8582858481518110610710576107106127c3565b016020015160f81c611040565b945080610729816127d9565b9150506106f0565b5061073d8387866110ce565b60208d81015160409081019290925283518251808401845260008082529083018190528351808501909452835263ffffffff16828201528e015161078091611168565b50505050505050505b505050505050565b60006107a36104c78760200151610eb5565b63ffffffff16905060006107bd6104c78860200151610eb5565b63ffffffff16905060006107dc6107d78960200151610eb5565b611178565b67ffffffffffffffff16905060208601351580156107fb575088358110155b15610823578760025b9081600281111561081757610817611f72565b81525050505050610789565b6020808801515167ffffffffffffffff1690610840908490612772565b1180610855575061085260208361279b565b15155b1561086257876002610804565b600061086f6020846127af565b905060008061088a6040518060200160405280606081525090565b60208b015161089c90858b8b87610f97565b90945090925090508888848181106108b6576108b66127c3565b909101357fff000000000000000000000000000000000000000000000000000000000000001615905061092b5760405162461bcd60e51b815260206004820152601360248201527f554e4b4e4f574e5f494e424f585f50524f4f460000000000000000000000000060448201526064016101df565b82610935816127d9565b9350611dea9050600060208c01356109515761123a9150610990565b60018c6020013503610967576115959150610990565b8d60025b9081600281111561097e5761097e611f72565b81525050505050505050505050610789565b6109b08f888d8d899080926109a793929190612732565b8663ffffffff16565b9050806109bf578d600261096b565b505082881015610a115760405162461bcd60e51b815260206004820152601160248201527f4241445f4d4553534147455f50524f4f4600000000000000000000000000000060448201526064016101df565b6000610a1d848a612821565b905060005b60208163ffffffff16108015610a46575081610a4463ffffffff83168b612772565b105b15610a9f57610a8b8463ffffffff83168d8d82610a638f8c612772565b610a6d9190612772565b818110610a7c57610a7c6127c3565b919091013560f81c9050611040565b935080610a9781612834565b915050610a22565b610aaa8387866110ce565b60208e015160400152610b00610aed82604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b8f6020015161116890919063ffffffff16565b505050505050505050505050505050565b50506001909252505050565b610b25611df4565b81610b2e611e19565b610b36611e19565b60005b600260ff82161015610b8157610b5088888661187d565b848360ff1660028110610b6557610b656127c3565b6020020191909152935080610b7981612857565b915050610b39565b5060005b600260ff82161015610bdc57610b9c888886611899565b838360ff1660028110610bb157610bb16127c3565b67ffffffffffffffff9093166020939093020191909152935080610bd481612857565b915050610b85565b506040805180820190915291825260208201529590945092505050565b6000610c0b6104c78860200151610eb5565b63ffffffff1690506000610c256104c78960200151610eb5565b9050600263ffffffff821610610c3d5787600261052c565b6020808801515167ffffffffffffffff1690610c5a908490612772565b1180610c6f5750610c6c60208361279b565b15155b15610c7c5787600261052c565b6000610c896020846127af565b9050600080610ca46040518060200160405280606081525090565b60208b0151610cb690858a8a87610f97565b9094509092509050618010610cce60208b018b61270e565b61ffff1603610d1257610d04848b600001518763ffffffff1660028110610cf757610cf76127c3565b60200201518391906110ce565b60208c015160400152610d98565b618011610d2260208b018b61270e565b61ffff1603610d50578951829063ffffffff871660028110610d4657610d466127c3565b6020020152610d98565b60405162461bcd60e51b815260206004820152601760248201527f4241445f474c4f42414c5f53544154455f4f50434f444500000000000000000060448201526064016101df565b505050505050505050505050565b6000610db86104c78460200151610eb5565b9050600263ffffffff821610610dd057505060029052565b610e39610e2e83602001518363ffffffff1660028110610df257610df26127c3565b6020020151604080518082019091526000808252602082015250604080518082019091526001815267ffffffffffffffff909116602082015290565b602085015190611168565b505050565b6000610e506107d78460200151610eb5565b90506000610e646104c78560200151610eb5565b9050600263ffffffff821610610e7e575050600290915250565b8183602001518263ffffffff1660028110610e9b57610e9b6127c3565b67ffffffffffffffff909216602092909202015250505050565b60408051808201909152600080825260208201528151610ed490611901565b92915050565b60208101516000908183516006811115610ef657610ef6611f72565b14610f435760405162461bcd60e51b815260206004820152600760248201527f4e4f545f4933320000000000000000000000000000000000000000000000000060448201526064016101df565b6401000000008110610ed45760405162461bcd60e51b815260206004820152600760248201527f4241445f4933320000000000000000000000000000000000000000000000000060448201526064016101df565b600080610fb06040518060200160405280606081525090565b839150610fbe86868461187d565b9093509150610fce868684611a12565b925090506000610fdf8289866110ce565b9050886040015181146110345760405162461bcd60e51b815260206004820152600e60248201527f57524f4e475f4d454d5f524f4f5400000000000000000000000000000000000060448201526064016101df565b50955095509592505050565b6000602083106110925760405162461bcd60e51b815260206004820152601560248201527f4241445f5345545f4c4541465f425954455f494458000000000000000000000060448201526064016101df565b6000836110a160016020612821565b6110ab9190612821565b6110b6906008612876565b60ff848116821b911b198616179150505b9392505050565b6040517f4d656d6f7279206c6561663a00000000000000000000000000000000000000006020820152602c81018290526000908190604c0160405160208183030381529060405280519060200120905061115f8585836040518060400160405280601381526020017f4d656d6f7279206d65726b6c6520747265653a00000000000000000000000000815250611aed565b95945050505050565b81516111749082611bc2565b5050565b602081015160009060018351600681111561119557611195611f72565b146111e25760405162461bcd60e51b815260206004820152600760248201527f4e4f545f4936340000000000000000000000000000000000000000000000000060448201526064016101df565b680100000000000000008110610ed45760405162461bcd60e51b815260206004820152600760248201527f4241445f4936340000000000000000000000000000000000000000000000000060448201526064016101df565b6000602882101561128d5760405162461bcd60e51b815260206004820152601260248201527f4241445f534551494e424f585f50524f4f46000000000000000000000000000060448201526064016101df565b600061129b84846020611899565b5080915050600084846040516112b2929190612811565b604051908190039020905060008067ffffffffffffffff88161561138a576112e060408a0160208b0161288d565b73ffffffffffffffffffffffffffffffffffffffff166316bf557961130660018b6128c3565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e084901b16815267ffffffffffffffff9091166004820152602401602060405180830381865afa158015611363573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061138791906128eb565b91505b67ffffffffffffffff841615611454576113aa60408a0160208b0161288d565b73ffffffffffffffffffffffffffffffffffffffff1663d5719dc26113d06001876128c3565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e084901b16815267ffffffffffffffff9091166004820152602401602060405180830381865afa15801561142d573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061145191906128eb565b90505b60408051602081018490529081018490526060810182905260009060800160405160208183030381529060405280519060200120905089602001602081019061149d919061288d565b6040517f16bf557900000000000000000000000000000000000000000000000000000000815267ffffffffffffffff8b16600482015273ffffffffffffffffffffffffffffffffffffffff91909116906316bf557990602401602060405180830381865afa158015611513573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061153791906128eb565b81146115855760405162461bcd60e51b815260206004820152601460248201527f4241445f534551494e424f585f4d45535341474500000000000000000000000060448201526064016101df565b5060019998505050505050505050565b600060718210156115e85760405162461bcd60e51b815260206004820152601160248201527f4241445f44454c415945445f50524f4f4600000000000000000000000000000060448201526064016101df565b600067ffffffffffffffff8516156116b45761160a604087016020880161288d565b73ffffffffffffffffffffffffffffffffffffffff1663d5719dc26116306001886128c3565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e084901b16815267ffffffffffffffff9091166004820152602401602060405180830381865afa15801561168d573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906116b191906128eb565b90505b60006116c38460718188612732565b6040516116d1929190612811565b604051809103902090506000858560008181106116f0576116f06127c3565b9050013560f81c60f81b9050600061170a87876001611cb6565b5090506000828261171f607160218b8d612732565b87604051602001611734959493929190612904565b60408051601f1981840301815282825280516020918201208382018990528383018190528251808503840181526060909401909252825192019190912090915061178460408c0160208d0161288d565b6040517fd5719dc200000000000000000000000000000000000000000000000000000000815267ffffffffffffffff8c16600482015273ffffffffffffffffffffffffffffffffffffffff919091169063d5719dc290602401602060405180830381865afa1580156117fa573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061181e91906128eb565b811461186c5760405162461bcd60e51b815260206004820152601360248201527f4241445f44454c415945445f4d4553534147450000000000000000000000000060448201526064016101df565b5060019a9950505050505050505050565b6000818161188c868684611cb6565b9097909650945050505050565b600081815b60088110156118f85760088367ffffffffffffffff16901b92508585838181106118ca576118ca6127c3565b919091013560f81c939093179250816118e2816127d9565b92505080806118f0906127d9565b91505061189e565b50935093915050565b60408051808201909152600080825260208201528151805161192590600190612821565b81518110611935576119356127c3565b60200260200101519050600060018360000151516119539190612821565b67ffffffffffffffff81111561196b5761196b6121e8565b6040519080825280602002602001820160405280156119b057816020015b60408051808201909152600080825260208201528152602001906001900390816119895790505b50905060005b8151811015611a0b5783518051829081106119d3576119d36127c3565b60200260200101518282815181106119ed576119ed6127c3565b60200260200101819052508080611a03906127d9565b9150506119b6565b5090915290565b604080516020810190915260608152816000611a2f868684611d0b565b92509050600060ff821667ffffffffffffffff811115611a5157611a516121e8565b604051908082528060200260200182016040528015611a7a578160200160208202803683370190505b50905060005b8260ff168160ff161015611ad157611a9988888661187d565b838360ff1681518110611aae57611aae6127c3565b602002602001018196508281525050508080611ac990612857565b915050611a80565b5060405180602001604052808281525093505050935093915050565b8160005b855151811015611bb95784600116600003611b5557828287600001518381518110611b1e57611b1e6127c3565b6020026020010151604051602001611b389392919061296d565b604051602081830303815290604052805190602001209150611ba0565b8286600001518281518110611b6c57611b6c6127c3565b602002602001015183604051602001611b879392919061296d565b6040516020818303038152906040528051906020012091505b60019490941c9380611bb1816127d9565b915050611af1565b50949350505050565b815151600090611bd3906001612772565b67ffffffffffffffff811115611beb57611beb6121e8565b604051908082528060200260200182016040528015611c3057816020015b6040805180820190915260008082526020820152815260200190600190039081611c095790505b50905060005b835151811015611c8c578351805182908110611c5457611c546127c3565b6020026020010151828281518110611c6e57611c6e6127c3565b60200260200101819052508080611c84906127d9565b915050611c36565b50818184600001515181518110611ca557611ca56127c3565b602090810291909101015290915250565b600081815b60208110156118f857600883901b9250858583818110611cdd57611cdd6127c3565b919091013560f81c93909317925081611cf5816127d9565b9250508080611d03906127d9565b915050611cbb565b600081848482818110611d2057611d206127c3565b919091013560f81c9250819050611d36816127d9565b915050935093915050565b6040805161012081019091528060008152602001611d7660408051606080820183529181019182529081526000602082015290565b8152602001611d9c60408051606080820183529181019182529081526000602082015290565b8152602001611dc1604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b611df26129a4565b565b6040518060400160405280611e07611e19565b8152602001611e14611e19565b905290565b60405180604001604052806002906020820280368337509192915050565b600060408284031215611e4957600080fd5b50919050565b60008083601f840112611e6157600080fd5b50813567ffffffffffffffff811115611e7957600080fd5b602083019150836020828501011115611e9157600080fd5b9250929050565b6000806000806000808688036101c0811215611eb357600080fd5b6060811215611ec157600080fd5b879650606088013567ffffffffffffffff80821115611edf57600080fd5b90890190610120828c031215611ef457600080fd5b81975060e07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8084011215611f2757600080fd5b60808a019650611f3b8b6101608c01611e37565b95506101a08a0135925080831115611f5257600080fd5b5050611f6089828a01611e4f565b979a9699509497509295939492505050565b634e487b7160e01b600052602160045260246000fd5b60038110611f9857611f98611f72565b9052565b805160078110611fae57611fae611f72565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b8084101561200b57611ff7828651611f9c565b938201936001939093019290850190611fe4565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b8281101561208a578451612056858251611f9c565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101612041565b509687015197909601969096525093949350505050565b60006101008083526120b68184018651611f88565b6020850151610120848101526120d0610220850182611fbb565b905060408601517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00808684030161014087015261210d8383611fbb565b92506060880151915080868403016101608701525061212c828261201f565b915050608086015161018085015260a08601516121526101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e0860152509085015161020084015290506110c7602083018480518252602081015167ffffffffffffffff80825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b6040805190810167ffffffffffffffff81118282101715612221576122216121e8565b60405290565b6040516020810167ffffffffffffffff81118282101715612221576122216121e8565b6040516080810167ffffffffffffffff81118282101715612221576122216121e8565b604051610120810167ffffffffffffffff81118282101715612221576122216121e8565b60405160a0810167ffffffffffffffff81118282101715612221576122216121e8565b6040516060810167ffffffffffffffff81118282101715612221576122216121e8565b604051601f8201601f1916810167ffffffffffffffff81118282101715612300576123006121e8565b604052919050565b80356003811061231757600080fd5b919050565b600067ffffffffffffffff821115612336576123366121e8565b5060051b60200190565b60006040828403121561235257600080fd5b61235a6121fe565b905081356007811061236b57600080fd5b808252506020820135602082015292915050565b6000604080838503121561239257600080fd5b61239a6121fe565b9150823567ffffffffffffffff808211156123b457600080fd5b818501915060208083880312156123ca57600080fd5b6123d2612227565b8335838111156123e157600080fd5b80850194505087601f8501126123f657600080fd5b8335925061240b6124068461231c565b6122d7565b83815260069390931b8401820192828101908985111561242a57600080fd5b948301945b84861015612450576124418a87612340565b8252948601949083019061242f565b8252508552948501359484019490945250909392505050565b803563ffffffff8116811461231757600080fd5b6000604080838503121561249057600080fd5b6124986121fe565b9150823567ffffffffffffffff8111156124b157600080fd5b8301601f810185136124c257600080fd5b803560206124d26124068361231c565b82815260a092830284018201928282019190898511156124f157600080fd5b948301945b8486101561255a5780868b03121561250e5760008081fd5b61251661224a565b6125208b88612340565b815287870135858201526060612537818901612469565b8983015261254760808901612469565b90820152835294850194918301916124f6565b50808752505080860135818601525050505092915050565b6000610120823603121561258557600080fd5b61258d61226d565b61259683612308565b8152602083013567ffffffffffffffff808211156125b357600080fd5b6125bf3683870161237f565b602084015260408501359150808211156125d857600080fd5b6125e43683870161237f565b604084015260608501359150808211156125fd57600080fd5b5061260a3682860161247d565b6060830152506080830135608082015261262660a08401612469565b60a082015261263760c08401612469565b60c082015261264860e08401612469565b60e082015261010092830135928101929092525090565b803567ffffffffffffffff8116811461231757600080fd5b600081830360e081121561268a57600080fd5b612692612291565b833581526060601f19830112156126a857600080fd5b6126b06122b4565b91506126be6020850161265f565b82526126cc6040850161265f565b6020830152606084013560408301528160208201526080840135604082015260a0840135606082015261270160c08501612469565b6080820152949350505050565b60006020828403121561272057600080fd5b813561ffff811681146110c757600080fd5b6000808585111561274257600080fd5b8386111561274f57600080fd5b5050820193919092039150565b634e487b7160e01b600052601160045260246000fd5b80820180821115610ed457610ed461275c565b634e487b7160e01b600052601260045260246000fd5b6000826127aa576127aa612785565b500690565b6000826127be576127be612785565b500490565b634e487b7160e01b600052603260045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361280a5761280a61275c565b5060010190565b8183823760009101908152919050565b81810381811115610ed457610ed461275c565b600063ffffffff80831681810361284d5761284d61275c565b6001019392505050565b600060ff821660ff810361286d5761286d61275c565b60010192915050565b8082028115828204841417610ed457610ed461275c565b60006020828403121561289f57600080fd5b813573ffffffffffffffffffffffffffffffffffffffff811681146110c757600080fd5b67ffffffffffffffff8281168282160390808211156128e4576128e461275c565b5092915050565b6000602082840312156128fd57600080fd5b5051919050565b7fff00000000000000000000000000000000000000000000000000000000000000861681527fffffffffffffffffffffffffffffffffffffffff0000000000000000000000008560601b1660018201528284601583013760159201918201526035019392505050565b6000845160005b8181101561298e5760208188018101518583015201612974565b5091909101928352506020820152604001919050565b634e487b7160e01b600052605160045260246000fdfea2646970667358221220ea7b1576baf26084cc90e3c1e0e39aa8b3052c93e8adae1fc8547a89e583cbfe64736f6c63430008110033",
}

// OneStepProverHostIoABI is the input ABI used to generate the binding from.
// Deprecated: Use OneStepProverHostIoMetaData.ABI instead.
var OneStepProverHostIoABI = OneStepProverHostIoMetaData.ABI

// OneStepProverHostIoBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OneStepProverHostIoMetaData.Bin instead.
var OneStepProverHostIoBin = OneStepProverHostIoMetaData.Bin

// DeployOneStepProverHostIo deploys a new Ethereum contract, binding an instance of OneStepProverHostIo to it.
func DeployOneStepProverHostIo(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OneStepProverHostIo, error) {
	parsed, err := OneStepProverHostIoMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OneStepProverHostIoBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OneStepProverHostIo{OneStepProverHostIoCaller: OneStepProverHostIoCaller{contract: contract}, OneStepProverHostIoTransactor: OneStepProverHostIoTransactor{contract: contract}, OneStepProverHostIoFilterer: OneStepProverHostIoFilterer{contract: contract}}, nil
}

// OneStepProverHostIo is an auto generated Go binding around an Ethereum contract.
type OneStepProverHostIo struct {
	OneStepProverHostIoCaller     // Read-only binding to the contract
	OneStepProverHostIoTransactor // Write-only binding to the contract
	OneStepProverHostIoFilterer   // Log filterer for contract events
}

// OneStepProverHostIoCaller is an auto generated read-only Go binding around an Ethereum contract.
type OneStepProverHostIoCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProverHostIoTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OneStepProverHostIoTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProverHostIoFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OneStepProverHostIoFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProverHostIoSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OneStepProverHostIoSession struct {
	Contract     *OneStepProverHostIo // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// OneStepProverHostIoCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OneStepProverHostIoCallerSession struct {
	Contract *OneStepProverHostIoCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// OneStepProverHostIoTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OneStepProverHostIoTransactorSession struct {
	Contract     *OneStepProverHostIoTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// OneStepProverHostIoRaw is an auto generated low-level Go binding around an Ethereum contract.
type OneStepProverHostIoRaw struct {
	Contract *OneStepProverHostIo // Generic contract binding to access the raw methods on
}

// OneStepProverHostIoCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OneStepProverHostIoCallerRaw struct {
	Contract *OneStepProverHostIoCaller // Generic read-only contract binding to access the raw methods on
}

// OneStepProverHostIoTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OneStepProverHostIoTransactorRaw struct {
	Contract *OneStepProverHostIoTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOneStepProverHostIo creates a new instance of OneStepProverHostIo, bound to a specific deployed contract.
func NewOneStepProverHostIo(address common.Address, backend bind.ContractBackend) (*OneStepProverHostIo, error) {
	contract, err := bindOneStepProverHostIo(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OneStepProverHostIo{OneStepProverHostIoCaller: OneStepProverHostIoCaller{contract: contract}, OneStepProverHostIoTransactor: OneStepProverHostIoTransactor{contract: contract}, OneStepProverHostIoFilterer: OneStepProverHostIoFilterer{contract: contract}}, nil
}

// NewOneStepProverHostIoCaller creates a new read-only instance of OneStepProverHostIo, bound to a specific deployed contract.
func NewOneStepProverHostIoCaller(address common.Address, caller bind.ContractCaller) (*OneStepProverHostIoCaller, error) {
	contract, err := bindOneStepProverHostIo(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProverHostIoCaller{contract: contract}, nil
}

// NewOneStepProverHostIoTransactor creates a new write-only instance of OneStepProverHostIo, bound to a specific deployed contract.
func NewOneStepProverHostIoTransactor(address common.Address, transactor bind.ContractTransactor) (*OneStepProverHostIoTransactor, error) {
	contract, err := bindOneStepProverHostIo(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProverHostIoTransactor{contract: contract}, nil
}

// NewOneStepProverHostIoFilterer creates a new log filterer instance of OneStepProverHostIo, bound to a specific deployed contract.
func NewOneStepProverHostIoFilterer(address common.Address, filterer bind.ContractFilterer) (*OneStepProverHostIoFilterer, error) {
	contract, err := bindOneStepProverHostIo(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OneStepProverHostIoFilterer{contract: contract}, nil
}

// bindOneStepProverHostIo binds a generic wrapper to an already deployed contract.
func bindOneStepProverHostIo(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OneStepProverHostIoMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProverHostIo *OneStepProverHostIoRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProverHostIo.Contract.OneStepProverHostIoCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProverHostIo *OneStepProverHostIoRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProverHostIo.Contract.OneStepProverHostIoTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProverHostIo *OneStepProverHostIoRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProverHostIo.Contract.OneStepProverHostIoTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProverHostIo *OneStepProverHostIoCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProverHostIo.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProverHostIo *OneStepProverHostIoTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProverHostIo.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProverHostIo *OneStepProverHostIoTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProverHostIo.Contract.contract.Transact(opts, method, params...)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverHostIo *OneStepProverHostIoCaller) ExecuteOneStep(opts *bind.CallOpts, execCtx ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	var out []interface{}
	err := _OneStepProverHostIo.contract.Call(opts, &out, "executeOneStep", execCtx, startMach, startMod, inst, proof)

	outstruct := new(struct {
		Mach Machine
		Mod  Module
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Mach = *abi.ConvertType(out[0], new(Machine)).(*Machine)
	outstruct.Mod = *abi.ConvertType(out[1], new(Module)).(*Module)

	return *outstruct, err

}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverHostIo *OneStepProverHostIoSession) ExecuteOneStep(execCtx ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverHostIo.Contract.ExecuteOneStep(&_OneStepProverHostIo.CallOpts, execCtx, startMach, startMod, inst, proof)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverHostIo *OneStepProverHostIoCallerSession) ExecuteOneStep(execCtx ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverHostIo.Contract.ExecuteOneStep(&_OneStepProverHostIo.CallOpts, execCtx, startMach, startMod, inst, proof)
}

// OneStepProverMathMetaData contains all meta data concerning the OneStepProverMath contract.
var OneStepProverMathMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"initialWasmModuleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionContext\",\"name\":\"\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"startMach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"startMod\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"opcode\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"argumentData\",\"type\":\"uint256\"}],\"internalType\":\"structInstruction\",\"name\":\"inst\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"mach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"mod\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506126ed806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c806397cc779a14610030575b600080fd5b61004361003e366004611c78565b61005a565b604051610051929190611e81565b60405180910390f35b610062611b64565b6040805160a0810182526000808252825160608082018552828252602080830184905282860184905284019190915292820181905291810182905260808101919091526100ae8761234d565b91506100bf36879003870187612452565b905060006100d060208701876124e9565b9050611c0d61ffff8216604514806100ec575061ffff82166050145b156100fa5750610336610318565b604661ffff831610801590610122575061011660096046612523565b61ffff168261ffff1611155b1561013057506104e6610318565b606761ffff831610801590610158575061014c60026067612523565b61ffff168261ffff1611155b1561016657506105c9610318565b606a61ffff8316108015906101805750607861ffff831611155b1561018e5750610656610318565b605161ffff8316108015906101b657506101aa60096051612523565b61ffff168261ffff1611155b156101c4575061087e610318565b607961ffff8316108015906101ec57506101e060026079612523565b61ffff168261ffff1611155b156101fa57506108e3610318565b607c61ffff8316108015906102145750608a61ffff831611155b15610222575061095d610318565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff5961ffff8316016102565750610b5b610318565b61ffff821660ac148061026d575061ffff821660ad145b1561027b5750610ba6610318565b60c061ffff831610801590610295575060c461ffff831611155b156102a35750610c25610318565b60bc61ffff8316108015906102bd575060bf61ffff831611155b156102cb5750610e3e610318565b60405162461bcd60e51b815260206004820152600e60248201527f494e56414c49445f4f50434f444500000000000000000000000000000000000060448201526064015b60405180910390fd5b61032984848989898663ffffffff16565b5050965096945050505050565b60006103458660200151610fd4565b9050604561035660208601866124e9565b61ffff16036103c65760008151600681111561037457610374611d52565b146103c15760405162461bcd60e51b815260206004820152600760248201527f4e4f545f49333200000000000000000000000000000000000000000000000000604482015260640161030f565b610488565b60506103d560208601866124e9565b61ffff1603610440576001815160068111156103f3576103f3611d52565b146103c15760405162461bcd60e51b815260206004820152600760248201527f4e4f545f49363400000000000000000000000000000000000000000000000000604482015260640161030f565b60405162461bcd60e51b815260206004820152600760248201527f4241445f45515a00000000000000000000000000000000000000000000000000604482015260640161030f565b6000816020015160000361049e575060016104a2565b5060005b604080518082018252600080825260209182018190528251808401909352825263ffffffff8316908201526104dd905b602089015190610ff9565b50505050505050565b60006104fd6104f88760200151610fd4565b611009565b905060006105116104f88860200151610fd4565b90506000604661052460208801886124e9565b61052e9190612545565b905060008061ffff83166002148061054a575061ffff83166004145b80610559575061ffff83166006145b80610568575061ffff83166008145b1561058857610576846110c6565b9150610581856110c6565b9050610596565b505063ffffffff8083169084165b60006105a38383866110f2565b90506105bc6105b18261138c565b60208d015190610ff9565b5050505050505050505050565b60006105db6104f88760200151610fd4565b9050600060676105ee60208701876124e9565b6105f89190612545565b9050600061060e8363ffffffff16836020611400565b604080518082018252600080825260209182018190528251808401909352825263ffffffff83169082015290915061064c905b60208a015190610ff9565b5050505050505050565b60006106686104f88760200151610fd4565b9050600061067c6104f88860200151610fd4565b9050600080606a61069060208901896124e9565b61069a9190612545565b90508061ffff166003036107325763ffffffff841615806106ec57508260030b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffff800000001480156106ec57508360030b600019145b15610715578860025b9081600281111561070857610708611d52565b8152505050505050610877565b8360030b8360030b8161072a5761072a612560565b059150610837565b8061ffff16600503610771578363ffffffff16600003610754578860026106f5565b8360030b8360030b8161076957610769612560565b079150610837565b8061ffff16600a036107905763ffffffff8316601f85161b9150610837565b8061ffff16600c036107af5763ffffffff8316601f85161c9150610837565b8061ffff16600b036107cc57600383900b601f85161d9150610837565b8061ffff16600d036107e9576107e283856115e0565b9150610837565b8061ffff16600e036107ff576107e28385611622565b6000806108198563ffffffff168763ffffffff1685611664565b915091508015610833575050600289525061087792505050565b5091505b604080518082018252600080825260209182018190528251808401909352825263ffffffff841690820152610872905b60208b015190610ff9565b505050505b5050505050565b60006108956108908760200151610fd4565b6117f9565b905060006108a96108908860200151610fd4565b9050600060516108bc60208801886124e9565b6108c69190612545565b905060006108d58385846110f2565b90506108726108678261138c565b60006108f56108908760200151610fd4565b90506000607961090860208701876124e9565b6109129190612545565b9050600061092283836040611400565b604080518082018252600080825260209182015281518083019092526001825263ffffffff9290921691810182905290915061064c90610641565b600061096f6108908760200151610fd4565b905060006109836108908860200151610fd4565b9050600080607c61099760208901896124e9565b6109a19190612545565b90508061ffff16600303610a215767ffffffffffffffff841615806109f757508260070b7fffffffffffffffffffffffffffffffffffffffffffffffff80000000000000001480156109f757508360070b600019145b15610a04578860026106f5565b8360070b8360070b81610a1957610a19612560565b059150610b23565b8061ffff16600503610a64578367ffffffffffffffff16600003610a47578860026106f5565b8360070b8360070b81610a5c57610a5c612560565b079150610b23565b8061ffff16600a03610a875767ffffffffffffffff8316603f85161b9150610b23565b8061ffff16600c03610aaa5767ffffffffffffffff8316603f85161c9150610b23565b8061ffff16600b03610ac757600783900b603f85161d9150610b23565b8061ffff16600d03610ae457610add83856118bb565b9150610b23565b8061ffff16600e03610afa57610add838561190d565b6000610b07848684611664565b90935090508015610b215750506002885250610877915050565b505b604080518082018252600080825260209182015281518083019092526001825267ffffffffffffffff84169082015261087290610867565b6000610b6d6108908760200151610fd4565b604080518082018252600080825260209182018190528251808401909352825263ffffffff83169082015290915081906104dd906104d2565b6000610bb86104f88760200151610fd4565b9050600060ac610bcb60208701876124e9565b61ffff1603610be457610bdd826110c6565b9050610bed565b5063ffffffff81165b604080518082018252600080825260209182015281518083019092526001825267ffffffffffffffff8316908201526104dd906104d2565b60008060c0610c3760208701876124e9565b61ffff1603610c4c5750600090506008610d24565b60c1610c5b60208701876124e9565b61ffff1603610c705750600090506010610d24565b60c2610c7f60208701876124e9565b61ffff1603610c945750600190506008610d24565b60c3610ca360208701876124e9565b61ffff1603610cb85750600190506010610d24565b60c4610cc760208701876124e9565b61ffff1603610cdc5750600190506020610d24565b60405162461bcd60e51b815260206004820152601860248201527f494e56414c49445f455854454e445f53414d455f545950450000000000000000604482015260640161030f565b600080836006811115610d3957610d39611d52565b03610d49575063ffffffff610d54565b5067ffffffffffffffff5b6000610d638960200151610fd4565b9050836006811115610d7757610d77611d52565b81516006811115610d8a57610d8a611d52565b14610dd75760405162461bcd60e51b815260206004820152601960248201527f4241445f455854454e445f53414d455f545950455f5459504500000000000000604482015260640161030f565b6000610dea600160ff861681901b612576565b602083018051821690529050610e01600185612589565b60ff166001901b826020015116600014610e2357602082018051821985161790525b60208a0151610e329083610ff9565b50505050505050505050565b60008060bc610e5060208701876124e9565b61ffff1603610e655750600090506002610f19565b60bd610e7460208701876124e9565b61ffff1603610e895750600190506003610f19565b60be610e9860208701876124e9565b61ffff1603610ead5750600290506000610f19565b60bf610ebc60208701876124e9565b61ffff1603610ed15750600390506001610f19565b60405162461bcd60e51b815260206004820152601360248201527f494e56414c49445f5245494e5445525052455400000000000000000000000000604482015260640161030f565b6000610f288860200151610fd4565b9050816006811115610f3c57610f3c611d52565b81516006811115610f4f57610f4f611d52565b14610f9c5760405162461bcd60e51b815260206004820152601860248201527f494e56414c49445f5245494e544552505245545f545950450000000000000000604482015260640161030f565b80836006811115610faf57610faf611d52565b90816006811115610fc257610fc2611d52565b905250602088015161064c9082610ff9565b60408051808201909152600080825260208201528151610ff39061195f565b92915050565b81516110059082611a70565b5050565b6020810151600090818351600681111561102557611025611d52565b146110725760405162461bcd60e51b815260206004820152600760248201527f4e4f545f49333200000000000000000000000000000000000000000000000000604482015260640161030f565b6401000000008110610ff35760405162461bcd60e51b815260206004820152600760248201527f4241445f49333200000000000000000000000000000000000000000000000000604482015260640161030f565b600063800000008216156110e8575063ffffffff1667ffffffff000000001790565b5063ffffffff1690565b600061ffff821661111b578267ffffffffffffffff168467ffffffffffffffff16149050611385565b60001961ffff831601611147578267ffffffffffffffff168467ffffffffffffffff1614159050611385565b60011961ffff831601611164578260070b8460070b129050611385565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd61ffff8316016111ad578267ffffffffffffffff168467ffffffffffffffff16109050611385565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc61ffff8316016111e8578260070b8460070b139050611385565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb61ffff831601611231578267ffffffffffffffff168467ffffffffffffffff16119050611385565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa61ffff83160161126d578260070b8460070b13159050611385565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff961ffff8316016112b7578267ffffffffffffffff168467ffffffffffffffff1611159050611385565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff861ffff8316016112f3578260070b8460070b12159050611385565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff761ffff83160161133d578267ffffffffffffffff168467ffffffffffffffff1610159050611385565b60405162461bcd60e51b815260206004820152600a60248201527f424144204952454c4f5000000000000000000000000000000000000000000000604482015260640161030f565b9392505050565b604080518082019091526000808252602082015281156113d1576040805180820182526000808252602091820181905282518084019093528252600190820152610ff3565b60408051808201825260008082526020918201819052825180840190935280835290820152610ff3565b919050565b60008161ffff166020148061141957508161ffff166040145b6114655760405162461bcd60e51b815260206004820152601860248201527f57524f4e4720555345204f462067656e65726963556e4f700000000000000000604482015260640161030f565b61ffff83166114d75761ffff82165b60008163ffffffff161180156114aa57506114906001826125a2565b63ffffffff166001901b8567ffffffffffffffff16166000145b156114c1576114ba6001826125a2565b9050611474565b6114cf8161ffff85166125a2565b915050611385565b60001961ffff8416016115315760005b8261ffff168163ffffffff161080156115135750600163ffffffff82161b851667ffffffffffffffff16155b1561152a576115236001826125bf565b90506114e7565b9050611385565b60011961ffff841601611598576000805b8361ffff168263ffffffff16101561158f57600163ffffffff83161b861667ffffffffffffffff161561157d5761157a6001826125bf565b90505b81611587816125dc565b925050611542565b91506113859050565b60405162461bcd60e51b815260206004820152600960248201527f4241442049556e4f700000000000000000000000000000000000000000000000604482015260640161030f565b60006115ed6020836125ff565b91506115fa8260206125a2565b63ffffffff168363ffffffff16901c8263ffffffff168463ffffffff16901b17905092915050565b600061162f6020836125ff565b915061163c8260206125a2565b63ffffffff168363ffffffff16901b8263ffffffff168463ffffffff16901c17905092915050565b6000808261ffff1660000361167f57505082820160006117f1565b8261ffff1660010361169757505081830360006117f1565b8261ffff166002036116af57505082820260006117f1565b8261ffff16600403611708578367ffffffffffffffff166000036116d957506000905060016117f1565b8367ffffffffffffffff168567ffffffffffffffff16816116fc576116fc612560565b046000915091506117f1565b8261ffff16600603611761578367ffffffffffffffff1660000361173257506000905060016117f1565b8367ffffffffffffffff168567ffffffffffffffff168161175557611755612560565b066000915091506117f1565b8261ffff1660070361177957505082821660006117f1565b8261ffff1660080361179157505082821760006117f1565b8261ffff166009036117a957505082821860006117f1565b60405162461bcd60e51b815260206004820152601660248201527f494e56414c49445f47454e455249435f42494e5f4f5000000000000000000000604482015260640161030f565b935093915050565b602081015160009060018351600681111561181657611816611d52565b146118635760405162461bcd60e51b815260206004820152600760248201527f4e4f545f49363400000000000000000000000000000000000000000000000000604482015260640161030f565b680100000000000000008110610ff35760405162461bcd60e51b815260206004820152600760248201527f4241445f49363400000000000000000000000000000000000000000000000000604482015260640161030f565b60006118c8604083612622565b91506118d582604061263d565b67ffffffffffffffff168367ffffffffffffffff16901c8267ffffffffffffffff168467ffffffffffffffff16901b17905092915050565b600061191a604083612622565b915061192782604061263d565b67ffffffffffffffff168367ffffffffffffffff16901b8267ffffffffffffffff168467ffffffffffffffff16901c17905092915050565b60408051808201909152600080825260208201528151805161198390600190612576565b815181106119935761199361265e565b60200260200101519050600060018360000151516119b19190612576565b67ffffffffffffffff8111156119c9576119c9611fc8565b604051908082528060200260200182016040528015611a0e57816020015b60408051808201909152600080825260208201528152602001906001900390816119e75790505b50905060005b8151811015611a69578351805182908110611a3157611a3161265e565b6020026020010151828281518110611a4b57611a4b61265e565b60200260200101819052508080611a6190612674565b915050611a14565b5090915290565b815151600090611a8190600161268e565b67ffffffffffffffff811115611a9957611a99611fc8565b604051908082528060200260200182016040528015611ade57816020015b6040805180820190915260008082526020820152815260200190600190039081611ab75790505b50905060005b835151811015611b3a578351805182908110611b0257611b0261265e565b6020026020010151828281518110611b1c57611b1c61265e565b60200260200101819052508080611b3290612674565b915050611ae4565b50818184600001515181518110611b5357611b5361265e565b602090810291909101015290915250565b6040805161012081019091528060008152602001611b9960408051606080820183529181019182529081526000602082015290565b8152602001611bbf60408051606080820183529181019182529081526000602082015290565b8152602001611be4604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b611c156126a1565b565b600060408284031215611c2957600080fd5b50919050565b60008083601f840112611c4157600080fd5b50813567ffffffffffffffff811115611c5957600080fd5b602083019150836020828501011115611c7157600080fd5b9250929050565b6000806000806000808688036101c0811215611c9357600080fd5b6060811215611ca157600080fd5b879650606088013567ffffffffffffffff80821115611cbf57600080fd5b90890190610120828c031215611cd457600080fd5b81975060e07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8084011215611d0757600080fd5b60808a019650611d1b8b6101608c01611c17565b95506101a08a0135925080831115611d3257600080fd5b5050611d4089828a01611c2f565b979a9699509497509295939492505050565b634e487b7160e01b600052602160045260246000fd5b60038110611d7857611d78611d52565b9052565b805160078110611d8e57611d8e611d52565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b80841015611deb57611dd7828651611d7c565b938201936001939093019290850190611dc4565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b82811015611e6a578451611e36858251611d7c565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101611e21565b509687015197909601969096525093949350505050565b6000610100808352611e968184018651611d68565b602085015161012084810152611eb0610220850182611d9b565b905060408601517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff008086840301610140870152611eed8383611d9b565b925060608801519150808684030161016087015250611f0c8282611dff565b915050608086015161018085015260a0860151611f326101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e086015250908501516102008401529050611385602083018480518252602081015167ffffffffffffffff80825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b6040805190810167ffffffffffffffff8111828210171561200157612001611fc8565b60405290565b6040516020810167ffffffffffffffff8111828210171561200157612001611fc8565b6040516080810167ffffffffffffffff8111828210171561200157612001611fc8565b604051610120810167ffffffffffffffff8111828210171561200157612001611fc8565b60405160a0810167ffffffffffffffff8111828210171561200157612001611fc8565b6040516060810167ffffffffffffffff8111828210171561200157612001611fc8565b604051601f8201601f1916810167ffffffffffffffff811182821017156120e0576120e0611fc8565b604052919050565b8035600381106113fb57600080fd5b600067ffffffffffffffff82111561211157612111611fc8565b5060051b60200190565b60006040828403121561212d57600080fd5b612135611fde565b905081356007811061214657600080fd5b808252506020820135602082015292915050565b6000604080838503121561216d57600080fd5b612175611fde565b9150823567ffffffffffffffff8082111561218f57600080fd5b818501915060208083880312156121a557600080fd5b6121ad612007565b8335838111156121bc57600080fd5b80850194505087601f8501126121d157600080fd5b833592506121e66121e1846120f7565b6120b7565b83815260069390931b8401820192828101908985111561220557600080fd5b948301945b8486101561222b5761221c8a8761211b565b8252948601949083019061220a565b8252508552948501359484019490945250909392505050565b803563ffffffff811681146113fb57600080fd5b6000604080838503121561226b57600080fd5b612273611fde565b9150823567ffffffffffffffff81111561228c57600080fd5b8301601f8101851361229d57600080fd5b803560206122ad6121e1836120f7565b82815260a092830284018201928282019190898511156122cc57600080fd5b948301945b848610156123355780868b0312156122e95760008081fd5b6122f161202a565b6122fb8b8861211b565b815287870135858201526060612312818901612244565b8983015261232260808901612244565b90820152835294850194918301916122d1565b50808752505080860135818601525050505092915050565b6000610120823603121561236057600080fd5b61236861204d565b612371836120e8565b8152602083013567ffffffffffffffff8082111561238e57600080fd5b61239a3683870161215a565b602084015260408501359150808211156123b357600080fd5b6123bf3683870161215a565b604084015260608501359150808211156123d857600080fd5b506123e536828601612258565b6060830152506080830135608082015261240160a08401612244565b60a082015261241260c08401612244565b60c082015261242360e08401612244565b60e082015261010092830135928101929092525090565b803567ffffffffffffffff811681146113fb57600080fd5b600081830360e081121561246557600080fd5b61246d612071565b833581526060601f198301121561248357600080fd5b61248b612094565b91506124996020850161243a565b82526124a76040850161243a565b6020830152606084013560408301528160208201526080840135604082015260a084013560608201526124dc60c08501612244565b6080820152949350505050565b6000602082840312156124fb57600080fd5b813561ffff8116811461138557600080fd5b634e487b7160e01b600052601160045260246000fd5b61ffff81811683821601908082111561253e5761253e61250d565b5092915050565b61ffff82811682821603908082111561253e5761253e61250d565b634e487b7160e01b600052601260045260246000fd5b81810381811115610ff357610ff361250d565b60ff8281168282160390811115610ff357610ff361250d565b63ffffffff82811682821603908082111561253e5761253e61250d565b63ffffffff81811683821601908082111561253e5761253e61250d565b600063ffffffff8083168181036125f5576125f561250d565b6001019392505050565b600063ffffffff8084168061261657612616612560565b92169190910692915050565b600067ffffffffffffffff8084168061261657612616612560565b67ffffffffffffffff82811682821603908082111561253e5761253e61250d565b634e487b7160e01b600052603260045260246000fd5b600060001982036126875761268761250d565b5060010190565b80820180821115610ff357610ff361250d565b634e487b7160e01b600052605160045260246000fdfea26469706673582212203454c1c30c97ca6d3e22c07fe2f741872668b8d35ac2b22c3d0f5a6dd237388d64736f6c63430008110033",
}

// OneStepProverMathABI is the input ABI used to generate the binding from.
// Deprecated: Use OneStepProverMathMetaData.ABI instead.
var OneStepProverMathABI = OneStepProverMathMetaData.ABI

// OneStepProverMathBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OneStepProverMathMetaData.Bin instead.
var OneStepProverMathBin = OneStepProverMathMetaData.Bin

// DeployOneStepProverMath deploys a new Ethereum contract, binding an instance of OneStepProverMath to it.
func DeployOneStepProverMath(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OneStepProverMath, error) {
	parsed, err := OneStepProverMathMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OneStepProverMathBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OneStepProverMath{OneStepProverMathCaller: OneStepProverMathCaller{contract: contract}, OneStepProverMathTransactor: OneStepProverMathTransactor{contract: contract}, OneStepProverMathFilterer: OneStepProverMathFilterer{contract: contract}}, nil
}

// OneStepProverMath is an auto generated Go binding around an Ethereum contract.
type OneStepProverMath struct {
	OneStepProverMathCaller     // Read-only binding to the contract
	OneStepProverMathTransactor // Write-only binding to the contract
	OneStepProverMathFilterer   // Log filterer for contract events
}

// OneStepProverMathCaller is an auto generated read-only Go binding around an Ethereum contract.
type OneStepProverMathCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProverMathTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OneStepProverMathTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProverMathFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OneStepProverMathFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProverMathSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OneStepProverMathSession struct {
	Contract     *OneStepProverMath // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// OneStepProverMathCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OneStepProverMathCallerSession struct {
	Contract *OneStepProverMathCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// OneStepProverMathTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OneStepProverMathTransactorSession struct {
	Contract     *OneStepProverMathTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// OneStepProverMathRaw is an auto generated low-level Go binding around an Ethereum contract.
type OneStepProverMathRaw struct {
	Contract *OneStepProverMath // Generic contract binding to access the raw methods on
}

// OneStepProverMathCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OneStepProverMathCallerRaw struct {
	Contract *OneStepProverMathCaller // Generic read-only contract binding to access the raw methods on
}

// OneStepProverMathTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OneStepProverMathTransactorRaw struct {
	Contract *OneStepProverMathTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOneStepProverMath creates a new instance of OneStepProverMath, bound to a specific deployed contract.
func NewOneStepProverMath(address common.Address, backend bind.ContractBackend) (*OneStepProverMath, error) {
	contract, err := bindOneStepProverMath(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OneStepProverMath{OneStepProverMathCaller: OneStepProverMathCaller{contract: contract}, OneStepProverMathTransactor: OneStepProverMathTransactor{contract: contract}, OneStepProverMathFilterer: OneStepProverMathFilterer{contract: contract}}, nil
}

// NewOneStepProverMathCaller creates a new read-only instance of OneStepProverMath, bound to a specific deployed contract.
func NewOneStepProverMathCaller(address common.Address, caller bind.ContractCaller) (*OneStepProverMathCaller, error) {
	contract, err := bindOneStepProverMath(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProverMathCaller{contract: contract}, nil
}

// NewOneStepProverMathTransactor creates a new write-only instance of OneStepProverMath, bound to a specific deployed contract.
func NewOneStepProverMathTransactor(address common.Address, transactor bind.ContractTransactor) (*OneStepProverMathTransactor, error) {
	contract, err := bindOneStepProverMath(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProverMathTransactor{contract: contract}, nil
}

// NewOneStepProverMathFilterer creates a new log filterer instance of OneStepProverMath, bound to a specific deployed contract.
func NewOneStepProverMathFilterer(address common.Address, filterer bind.ContractFilterer) (*OneStepProverMathFilterer, error) {
	contract, err := bindOneStepProverMath(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OneStepProverMathFilterer{contract: contract}, nil
}

// bindOneStepProverMath binds a generic wrapper to an already deployed contract.
func bindOneStepProverMath(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OneStepProverMathMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProverMath *OneStepProverMathRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProverMath.Contract.OneStepProverMathCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProverMath *OneStepProverMathRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProverMath.Contract.OneStepProverMathTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProverMath *OneStepProverMathRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProverMath.Contract.OneStepProverMathTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProverMath *OneStepProverMathCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProverMath.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProverMath *OneStepProverMathTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProverMath.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProverMath *OneStepProverMathTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProverMath.Contract.contract.Transact(opts, method, params...)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverMath *OneStepProverMathCaller) ExecuteOneStep(opts *bind.CallOpts, arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	var out []interface{}
	err := _OneStepProverMath.contract.Call(opts, &out, "executeOneStep", arg0, startMach, startMod, inst, proof)

	outstruct := new(struct {
		Mach Machine
		Mod  Module
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Mach = *abi.ConvertType(out[0], new(Machine)).(*Machine)
	outstruct.Mod = *abi.ConvertType(out[1], new(Module)).(*Module)

	return *outstruct, err

}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverMath *OneStepProverMathSession) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverMath.Contract.ExecuteOneStep(&_OneStepProverMath.CallOpts, arg0, startMach, startMod, inst, proof)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverMath *OneStepProverMathCallerSession) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverMath.Contract.ExecuteOneStep(&_OneStepProverMath.CallOpts, arg0, startMach, startMod, inst, proof)
}

// OneStepProverMemoryMetaData contains all meta data concerning the OneStepProverMemory contract.
var OneStepProverMemoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"initialWasmModuleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structExecutionContext\",\"name\":\"\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"startMach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"startMod\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"opcode\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"argumentData\",\"type\":\"uint256\"}],\"internalType\":\"structInstruction\",\"name\":\"inst\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"mach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"mod\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611f4b806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c806397cc779a14610030575b600080fd5b61004361003e3660046114d3565b61005a565b6040516100519291906116dc565b60405180910390f35b6100626113bf565b6040805160a0810182526000808252825160608082018552828252602080830184905282860184905284019190915292820181905291810182905260808101919091526100ae87611bad565b91506100bf36879003870187611cb2565b905060006100d06020870187611d49565b9050611468602861ffff8316108015906100ef5750603561ffff831611155b156100fd57506101f86101da565b603661ffff8316108015906101175750603e61ffff831611155b1561012557506107016101da565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc161ffff8316016101595750610ab86101da565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc061ffff83160161018d5750610b126101da565b60405162461bcd60e51b815260206004820152601560248201527f494e56414c49445f4d454d4f52595f4f50434f4445000000000000000000000060448201526064015b60405180910390fd5b6101eb84848989898663ffffffff16565b5050965096945050505050565b60008080602861020b6020880188611d49565b61ffff1603610223575060009150600490508161046b565b60296102326020880188611d49565b61ffff160361024b57506001915060089050600061046b565b602a61025a6020880188611d49565b61ffff160361027357506002915060049050600061046b565b602b6102826020880188611d49565b61ffff160361029b57506003915060089050600061046b565b602c6102aa6020880188611d49565b61ffff16036102c2575060009150600190508061046b565b602d6102d16020880188611d49565b61ffff16036102e9575060009150600190508161046b565b602e6102f86020880188611d49565b61ffff160361031157506000915060029050600161046b565b602f6103206020880188611d49565b61ffff1603610338575060009150600290508161046b565b60306103476020880188611d49565b61ffff160361035e5750600191508190508061046b565b603161036d6020880188611d49565b61ffff1603610385575060019150819050600061046b565b60326103946020880188611d49565b61ffff16036103ac575060019150600290508161046b565b60336103bb6020880188611d49565b61ffff16036103d457506001915060029050600061046b565b60346103e36020880188611d49565b61ffff16036103fb575060019150600490508161046b565b603561040a6020880188611d49565b61ffff160361042357506001915060049050600061046b565b60405162461bcd60e51b815260206004820152601a60248201527f494e56414c49445f4d454d4f52595f4c4f41445f4f50434f444500000000000060448201526064016101d1565b600061048261047d8a60200151610c15565b610c3a565b6104969063ffffffff166020890135611d83565b60208901515190915067ffffffffffffffff166104b38483611d83565b11156104c757505060028752506106fa9050565b60006000198180805b8781101561056e5760006104e48288611d83565b905060006104f3602083611dac565b90508581146105175761050d8f60200151828f8f8b610cf7565b5097509095509350845b6000610524602084611dc0565b9050610531846008611dd4565b67ffffffffffffffff166105458783610da0565b60ff1667ffffffffffffffff16901b85179450505050808061056690611deb565b9150506104d0565b5085156106b55786600114801561059657506000886006811115610594576105946115ad565b145b156105ac578060000b63ffffffff1690506106b5565b8660011480156105cd575060018860068111156105cb576105cb6115ad565b145b156105da5760000b6106b5565b8660021480156105fb575060008860068111156105f9576105f96115ad565b145b15610611578060010b63ffffffff1690506106b5565b86600214801561063257506001886006811115610630576106306115ad565b145b1561063f5760010b6106b5565b8660041480156106605750600188600681111561065e5761065e6115ad565b145b1561066d5760030b6106b5565b60405162461bcd60e51b815260206004820152601560248201527f4241445f524541445f42595445535f5349474e4544000000000000000000000060448201526064016101d1565b6106f160405180604001604052808a60068111156106d5576106d56115ad565b815267ffffffffffffffff84166020918201528f015190610e21565b50505050505050505b5050505050565b6000808060366107146020880188611d49565b61ffff16036107295750600491506000610890565b60376107386020880188611d49565b61ffff160361074d5750600891506001610890565b603861075c6020880188611d49565b61ffff16036107715750600491506002610890565b60396107806020880188611d49565b61ffff16036107955750600891506003610890565b603a6107a46020880188611d49565b61ffff16036107b95750600191506000610890565b603b6107c86020880188611d49565b61ffff16036107dd5750600291506000610890565b603c6107ec6020880188611d49565b61ffff160361080057506001915081610890565b603d61080f6020880188611d49565b61ffff16036108245750600291506001610890565b603e6108336020880188611d49565b61ffff16036108485750600491506001610890565b60405162461bcd60e51b815260206004820152601b60248201527f494e56414c49445f4d454d4f52595f53544f52455f4f50434f4445000000000060448201526064016101d1565b600061089f8960200151610c15565b90508160068111156108b3576108b36115ad565b815160068111156108c6576108c66115ad565b146109135760405162461bcd60e51b815260206004820152600e60248201527f4241445f53544f52455f5459504500000000000000000000000000000000000060448201526064016101d1565b8060200151925060088467ffffffffffffffff16101561096157600161093a856008611e05565b67ffffffffffffffff16600167ffffffffffffffff16901b61095c9190611e31565b831692505b5050600061097561047d8960200151610c15565b6109899063ffffffff166020880135611d83565b905086602001516000015167ffffffffffffffff168367ffffffffffffffff16826109b49190611d83565b11156109c657505060028652506106fa565b604080516020810190915260608152600090600019906000805b8767ffffffffffffffff16811015610a955760006109fe8288611d83565b90506000610a0d602083611dac565b9050858114610a52576000198614610a3457610a2a858786610e31565b60208f0151604001525b610a458e60200151828e8e8b610cf7565b9098509196509094509250845b6000610a5f602084611dc0565b9050610a6c85828c610ecd565b945060088a67ffffffffffffffff16901c99505050508080610a8d90611deb565b9150506109e0565b50610aa1828483610e31565b60208c015160400152505050505050505050505050565b602084015151600090610acf906201000090611e59565b604080518082018252600080825260209182018190528251808401909352825263ffffffff831682820152880151919250610b0a9190610e21565b505050505050565b602084015151600090610b29906201000090611e59565b90506000610b3d61047d8860200151610c15565b90506000610b5463ffffffff808416908516611d83565b905086602001516020015167ffffffffffffffff168111610bd957610b7c6201000082611dd4565b602088015167ffffffffffffffff9091169052610bd4610bc984604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b60208a015190610e21565b610c0b565b604080518082018252600080825260209182018190528251808401909352825263ffffffff90820152610c0b90610bc9565b5050505050505050565b60408051808201909152600080825260208201528151610c3490610f5a565b92915050565b60208101516000908183516006811115610c5657610c566115ad565b14610ca35760405162461bcd60e51b815260206004820152600760248201527f4e4f545f4933320000000000000000000000000000000000000000000000000060448201526064016101d1565b6401000000008110610c345760405162461bcd60e51b815260206004820152600760248201527f4241445f4933320000000000000000000000000000000000000000000000000060448201526064016101d1565b600080610d106040518060200160405280606081525090565b839150610d1e86868461106b565b9093509150610d2e868684611087565b925090506000610d3f828986610e31565b905088604001518114610d945760405162461bcd60e51b815260206004820152600e60248201527f57524f4e475f4d454d5f524f4f5400000000000000000000000000000000000060448201526064016101d1565b50955095509592505050565b600060208210610df25760405162461bcd60e51b815260206004820152601660248201527f4241445f50554c4c5f4c4541465f425954455f4944580000000000000000000060448201526064016101d1565b600082610e0160016020611e80565b610e0b9190611e80565b610e16906008611dd4565b9390931c9392505050565b8151610e2d9082611162565b5050565b6040517f4d656d6f7279206c6561663a00000000000000000000000000000000000000006020820152602c81018290526000908190604c01604051602081830303815290604052805190602001209050610ec28585836040518060400160405280601381526020017f4d656d6f7279206d65726b6c6520747265653a00000000000000000000000000815250611256565b9150505b9392505050565b600060208310610f1f5760405162461bcd60e51b815260206004820152601560248201527f4241445f5345545f4c4541465f425954455f494458000000000000000000000060448201526064016101d1565b600083610f2e60016020611e80565b610f389190611e80565b610f43906008611dd4565b60ff848116821b911b198616179150509392505050565b604080518082019091526000808252602082015281518051610f7e90600190611e80565b81518110610f8e57610f8e611e93565b6020026020010151905060006001836000015151610fac9190611e80565b67ffffffffffffffff811115610fc457610fc4611823565b60405190808252806020026020018201604052801561100957816020015b6040805180820190915260008082526020820152815260200190600190039081610fe25790505b50905060005b815181101561106457835180518290811061102c5761102c611e93565b602002602001015182828151811061104657611046611e93565b6020026020010181905250808061105c90611deb565b91505061100f565b5090915290565b6000818161107a86868461132b565b9097909650945050505050565b6040805160208101909152606081528160006110a4868684611389565b92509050600060ff821667ffffffffffffffff8111156110c6576110c6611823565b6040519080825280602002602001820160405280156110ef578160200160208202803683370190505b50905060005b8260ff168160ff1610156111465761110e88888661106b565b838360ff168151811061112357611123611e93565b60200260200101819650828152505050808061113e90611ea9565b9150506110f5565b5060405180602001604052808281525093505050935093915050565b815151600090611173906001611d83565b67ffffffffffffffff81111561118b5761118b611823565b6040519080825280602002602001820160405280156111d057816020015b60408051808201909152600080825260208201528152602001906001900390816111a95790505b50905060005b83515181101561122c5783518051829081106111f4576111f4611e93565b602002602001015182828151811061120e5761120e611e93565b6020026020010181905250808061122490611deb565b9150506111d6565b5081818460000151518151811061124557611245611e93565b602090810291909101015290915250565b8160005b85515181101561132257846001166000036112be5782828760000151838151811061128757611287611e93565b60200260200101516040516020016112a193929190611ec8565b604051602081830303815290604052805190602001209150611309565b82866000015182815181106112d5576112d5611e93565b6020026020010151836040516020016112f093929190611ec8565b6040516020818303038152906040528051906020012091505b60019490941c938061131a81611deb565b91505061125a565b50949350505050565b600081815b602081101561138057600883901b925085858381811061135257611352611e93565b919091013560f81c9390931792508161136a81611deb565b925050808061137890611deb565b915050611330565b50935093915050565b60008184848281811061139e5761139e611e93565b919091013560f81c92508190506113b481611deb565b915050935093915050565b60408051610120810190915280600081526020016113f460408051606080820183529181019182529081526000602082015290565b815260200161141a60408051606080820183529181019182529081526000602082015290565b815260200161143f604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b611470611eff565b565b60006040828403121561148457600080fd5b50919050565b60008083601f84011261149c57600080fd5b50813567ffffffffffffffff8111156114b457600080fd5b6020830191508360208285010111156114cc57600080fd5b9250929050565b6000806000806000808688036101c08112156114ee57600080fd5b60608112156114fc57600080fd5b879650606088013567ffffffffffffffff8082111561151a57600080fd5b90890190610120828c03121561152f57600080fd5b81975060e07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808401121561156257600080fd5b60808a0196506115768b6101608c01611472565b95506101a08a013592508083111561158d57600080fd5b505061159b89828a0161148a565b979a9699509497509295939492505050565b634e487b7160e01b600052602160045260246000fd5b600381106115d3576115d36115ad565b9052565b8051600781106115e9576115e96115ad565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b80841015611646576116328286516115d7565b93820193600193909301929085019061161f565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b828110156116c55784516116918582516115d7565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a09093019260010161167c565b509687015197909601969096525093949350505050565b60006101008083526116f181840186516115c3565b60208501516101208481015261170b6102208501826115f6565b905060408601517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00808684030161014087015261174883836115f6565b925060608801519150808684030161016087015250611767828261165a565b915050608086015161018085015260a086015161178d6101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e086015250908501516102008401529050610ec6602083018480518252602081015167ffffffffffffffff80825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b6040805190810167ffffffffffffffff8111828210171561185c5761185c611823565b60405290565b6040516020810167ffffffffffffffff8111828210171561185c5761185c611823565b6040516080810167ffffffffffffffff8111828210171561185c5761185c611823565b604051610120810167ffffffffffffffff8111828210171561185c5761185c611823565b60405160a0810167ffffffffffffffff8111828210171561185c5761185c611823565b6040516060810167ffffffffffffffff8111828210171561185c5761185c611823565b604051601f8201601f1916810167ffffffffffffffff8111828210171561193b5761193b611823565b604052919050565b80356003811061195257600080fd5b919050565b600067ffffffffffffffff82111561197157611971611823565b5060051b60200190565b60006040828403121561198d57600080fd5b611995611839565b90508135600781106119a657600080fd5b808252506020820135602082015292915050565b600060408083850312156119cd57600080fd5b6119d5611839565b9150823567ffffffffffffffff808211156119ef57600080fd5b81850191506020808388031215611a0557600080fd5b611a0d611862565b833583811115611a1c57600080fd5b80850194505087601f850112611a3157600080fd5b83359250611a46611a4184611957565b611912565b83815260069390931b84018201928281019089851115611a6557600080fd5b948301945b84861015611a8b57611a7c8a8761197b565b82529486019490830190611a6a565b8252508552948501359484019490945250909392505050565b803563ffffffff8116811461195257600080fd5b60006040808385031215611acb57600080fd5b611ad3611839565b9150823567ffffffffffffffff811115611aec57600080fd5b8301601f81018513611afd57600080fd5b80356020611b0d611a4183611957565b82815260a09283028401820192828201919089851115611b2c57600080fd5b948301945b84861015611b955780868b031215611b495760008081fd5b611b51611885565b611b5b8b8861197b565b815287870135858201526060611b72818901611aa4565b89830152611b8260808901611aa4565b9082015283529485019491830191611b31565b50808752505080860135818601525050505092915050565b60006101208236031215611bc057600080fd5b611bc86118a8565b611bd183611943565b8152602083013567ffffffffffffffff80821115611bee57600080fd5b611bfa368387016119ba565b60208401526040850135915080821115611c1357600080fd5b611c1f368387016119ba565b60408401526060850135915080821115611c3857600080fd5b50611c4536828601611ab8565b60608301525060808301356080820152611c6160a08401611aa4565b60a0820152611c7260c08401611aa4565b60c0820152611c8360e08401611aa4565b60e082015261010092830135928101929092525090565b803567ffffffffffffffff8116811461195257600080fd5b600081830360e0811215611cc557600080fd5b611ccd6118cc565b833581526060601f1983011215611ce357600080fd5b611ceb6118ef565b9150611cf960208501611c9a565b8252611d0760408501611c9a565b6020830152606084013560408301528160208201526080840135604082015260a08401356060820152611d3c60c08501611aa4565b6080820152949350505050565b600060208284031215611d5b57600080fd5b813561ffff81168114610ec657600080fd5b634e487b7160e01b600052601160045260246000fd5b80820180821115610c3457610c34611d6d565b634e487b7160e01b600052601260045260246000fd5b600082611dbb57611dbb611d96565b500490565b600082611dcf57611dcf611d96565b500690565b8082028115828204841417610c3457610c34611d6d565b60006000198203611dfe57611dfe611d6d565b5060010190565b67ffffffffffffffff818116838216028082169190828114611e2957611e29611d6d565b505092915050565b67ffffffffffffffff828116828216039080821115611e5257611e52611d6d565b5092915050565b600067ffffffffffffffff80841680611e7457611e74611d96565b92169190910492915050565b81810381811115610c3457610c34611d6d565b634e487b7160e01b600052603260045260246000fd5b600060ff821660ff8103611ebf57611ebf611d6d565b60010192915050565b6000845160005b81811015611ee95760208188018101518583015201611ecf565b5091909101928352506020820152604001919050565b634e487b7160e01b600052605160045260246000fdfea26469706673582212204c9570fa43a330c3369dbd598f2fc7f0dc097e49facea4fb5f4d58080493939764736f6c63430008110033",
}

// OneStepProverMemoryABI is the input ABI used to generate the binding from.
// Deprecated: Use OneStepProverMemoryMetaData.ABI instead.
var OneStepProverMemoryABI = OneStepProverMemoryMetaData.ABI

// OneStepProverMemoryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OneStepProverMemoryMetaData.Bin instead.
var OneStepProverMemoryBin = OneStepProverMemoryMetaData.Bin

// DeployOneStepProverMemory deploys a new Ethereum contract, binding an instance of OneStepProverMemory to it.
func DeployOneStepProverMemory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OneStepProverMemory, error) {
	parsed, err := OneStepProverMemoryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OneStepProverMemoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OneStepProverMemory{OneStepProverMemoryCaller: OneStepProverMemoryCaller{contract: contract}, OneStepProverMemoryTransactor: OneStepProverMemoryTransactor{contract: contract}, OneStepProverMemoryFilterer: OneStepProverMemoryFilterer{contract: contract}}, nil
}

// OneStepProverMemory is an auto generated Go binding around an Ethereum contract.
type OneStepProverMemory struct {
	OneStepProverMemoryCaller     // Read-only binding to the contract
	OneStepProverMemoryTransactor // Write-only binding to the contract
	OneStepProverMemoryFilterer   // Log filterer for contract events
}

// OneStepProverMemoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type OneStepProverMemoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProverMemoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OneStepProverMemoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProverMemoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OneStepProverMemoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProverMemorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OneStepProverMemorySession struct {
	Contract     *OneStepProverMemory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// OneStepProverMemoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OneStepProverMemoryCallerSession struct {
	Contract *OneStepProverMemoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// OneStepProverMemoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OneStepProverMemoryTransactorSession struct {
	Contract     *OneStepProverMemoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// OneStepProverMemoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type OneStepProverMemoryRaw struct {
	Contract *OneStepProverMemory // Generic contract binding to access the raw methods on
}

// OneStepProverMemoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OneStepProverMemoryCallerRaw struct {
	Contract *OneStepProverMemoryCaller // Generic read-only contract binding to access the raw methods on
}

// OneStepProverMemoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OneStepProverMemoryTransactorRaw struct {
	Contract *OneStepProverMemoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOneStepProverMemory creates a new instance of OneStepProverMemory, bound to a specific deployed contract.
func NewOneStepProverMemory(address common.Address, backend bind.ContractBackend) (*OneStepProverMemory, error) {
	contract, err := bindOneStepProverMemory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OneStepProverMemory{OneStepProverMemoryCaller: OneStepProverMemoryCaller{contract: contract}, OneStepProverMemoryTransactor: OneStepProverMemoryTransactor{contract: contract}, OneStepProverMemoryFilterer: OneStepProverMemoryFilterer{contract: contract}}, nil
}

// NewOneStepProverMemoryCaller creates a new read-only instance of OneStepProverMemory, bound to a specific deployed contract.
func NewOneStepProverMemoryCaller(address common.Address, caller bind.ContractCaller) (*OneStepProverMemoryCaller, error) {
	contract, err := bindOneStepProverMemory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProverMemoryCaller{contract: contract}, nil
}

// NewOneStepProverMemoryTransactor creates a new write-only instance of OneStepProverMemory, bound to a specific deployed contract.
func NewOneStepProverMemoryTransactor(address common.Address, transactor bind.ContractTransactor) (*OneStepProverMemoryTransactor, error) {
	contract, err := bindOneStepProverMemory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProverMemoryTransactor{contract: contract}, nil
}

// NewOneStepProverMemoryFilterer creates a new log filterer instance of OneStepProverMemory, bound to a specific deployed contract.
func NewOneStepProverMemoryFilterer(address common.Address, filterer bind.ContractFilterer) (*OneStepProverMemoryFilterer, error) {
	contract, err := bindOneStepProverMemory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OneStepProverMemoryFilterer{contract: contract}, nil
}

// bindOneStepProverMemory binds a generic wrapper to an already deployed contract.
func bindOneStepProverMemory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OneStepProverMemoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProverMemory *OneStepProverMemoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProverMemory.Contract.OneStepProverMemoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProverMemory *OneStepProverMemoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProverMemory.Contract.OneStepProverMemoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProverMemory *OneStepProverMemoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProverMemory.Contract.OneStepProverMemoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProverMemory *OneStepProverMemoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProverMemory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProverMemory *OneStepProverMemoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProverMemory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProverMemory *OneStepProverMemoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProverMemory.Contract.contract.Transact(opts, method, params...)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverMemory *OneStepProverMemoryCaller) ExecuteOneStep(opts *bind.CallOpts, arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	var out []interface{}
	err := _OneStepProverMemory.contract.Call(opts, &out, "executeOneStep", arg0, startMach, startMod, inst, proof)

	outstruct := new(struct {
		Mach Machine
		Mod  Module
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Mach = *abi.ConvertType(out[0], new(Machine)).(*Machine)
	outstruct.Mod = *abi.ConvertType(out[1], new(Module)).(*Module)

	return *outstruct, err

}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverMemory *OneStepProverMemorySession) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverMemory.Contract.ExecuteOneStep(&_OneStepProverMemory.CallOpts, arg0, startMach, startMod, inst, proof)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0x97cc779a.
//
// Solidity: function executeOneStep((uint256,address,bytes32) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverMemory *OneStepProverMemoryCallerSession) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverMemory.Contract.ExecuteOneStep(&_OneStepProverMemory.CallOpts, arg0, startMach, startMod, inst, proof)
}
