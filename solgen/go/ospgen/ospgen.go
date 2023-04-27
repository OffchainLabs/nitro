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
	Bin: "0x608060405234801561001057600080fd5b50611d42806100206000396000f3fe608060405234801561001057600080fd5b50600436106100575760003560e01c8063740085d71461005c57806379754cba14610085578063ae364ac2146100a6578063b7465799146100b0578063d4e5dd2b146100d2575b600080fd5b61006f61006a36600461184c565b6100e5565b60405161007c91906118be565b60405180910390f35b610098610093366004611920565b6101d9565b60405190815260200161007c565b6100ae6106c4565b005b6100c36100be36600461197b565b61070c565b60405161007c939291906119a4565b6100986100e03660046119d6565b6107c2565b6000828152602081815260408083206001600160401b0385168452909152902080546060919060ff16610142576040516309cb23c960e11b8152600481018590526001600160401b03841660248201526044015b60405180910390fd5b80600101805461015190611a29565b80601f016020809104026020016040519081016040528092919081815260200182805461017d90611a29565b80156101ca5780601f1061019f576101008083540402835291602001916101ca565b820191906000526020600020905b8154815290600101906020018083116101ad57829003601f168201915b50505050509150505b92915050565b60006001821615156002831615610231573360009081526001602081905260408220805467ffffffffffffffff1916815591906102189083018261178e565b6102266002830160006117cb565b600982016000905550505b80806102455750610243608886611a79565b155b6102855760405162461bcd60e51b81526020600482015260116024820152701393d517d09313d0d2d7d0531251d39151607a1b6044820152606401610139565b336000908152600160205260408120600981015490918190036102c157815467ffffffffffffffff19166001600160401b03871617825561030b565b81546001600160401b0387811691161461030b5760405162461bcd60e51b815260206004820152600b60248201526a1112519197d3d19194d15560aa1b6044820152606401610139565b61031782898986610914565b8061032c60206001600160401b038916611aa3565b11801561034557508160090154866001600160401b0316105b1561045a57600081876001600160401b0316111561037357610370826001600160401b038916611ab6565b90505b60008261038a60206001600160401b038b16611aa3565b6103949190611ab6565b9050888111156103a15750875b815b8181101561045657846001018b8b838181106103c1576103c1611ac9565b9050013560f81c60f81b90808054806103d990611a29565b80601f81036103f85783600052602060002060ff1984168155603f9350505b506002919091019091558154600116156104215790600052602060002090602091828204019190065b909190919091601f036101000a81548160ff02191690600160f81b84040217905550808061044e90611adf565b9150506103a3565b5050505b8261046c5750600092506106bc915050565b60005b602081101561053c576000610485600883611af8565b9050610492600582611af8565b61049d600583611a79565b6104a8906005611b0c565b6104b29190611aa3565b905060006104c1600884611a79565b6104cc906008611b0c565b8560020183601981106104e1576104e1611ac9565b60048104909101546001600160401b036008600390931683026101000a9091041690911c9150610512908490611b0c565b61051d9060f8611ab6565b60ff909116901b9590951794508061053481611adf565b91505061046f565b50604051806040016040528060011515815260200183600101805461056090611a29565b80601f016020809104026020016040519081016040528092919081815260200182805461058c90611a29565b80156105d95780601f106105ae576101008083540402835291602001916105d9565b820191906000526020600020905b8154815290600101906020018083116105bc57829003601f168201915b50505091909252505060008581526020818152604080832086546001600160401b0316845282529091208251815460ff19169015151781559082015160018201906106249082611b88565b505082546040516001600160401b03909116915085907ff88493e8ac6179d3c1ba8712068367d7ecdd6f30d3b5de01198e7a449fe2802c9061066a906001870190611c47565b60405180910390a33360009081526001602081905260408220805467ffffffffffffffff1916815591906106a09083018261178e565b6106ae6002830160006117cb565b600982016000905550505050505b949350505050565b3360009081526001602081905260408220805467ffffffffffffffff1916815591906106f29083018261178e565b6107006002830160006117cb565b60098201600090555050565b6001602081905260009182526040909120805491810180546001600160401b039093169261073990611a29565b80601f016020809104026020016040519081016040528092919081815260200182805461076590611a29565b80156107b25780601f10610787576101008083540402835291602001916107b2565b820191906000526020600020905b81548152906001019060200180831161079557829003601f168201915b5050505050908060090154905083565b600083836040516107d4929190611cd2565b604051908190039020905060606001600160401b0383168411156108725760006108076001600160401b03851686611ab6565b90506020811115610816575060205b856001600160401b0385168661082c8483611aa3565b9261083993929190611ce2565b8080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250929450505050505b6040805180820182526001808252602080830185815260008781528083528581206001600160401b038a1682529092529390208251815460ff19169015151781559251919291908201906108c69082611b88565b50905050826001600160401b0316827ff88493e8ac6179d3c1ba8712068367d7ecdd6f30d3b5de01198e7a449fe2802c8360405161090491906118be565b60405180910390a3509392505050565b8282905084600901600082825461092b9190611aa3565b90915550505b8115801561093d575080155b610b8e5760005b6088811015610a665760008382101561097a5784848381811061096957610969611ac9565b919091013560f81c915061099b9050565b838203610985576001175b61099160016088611ab6565b820361099b576080175b60006109a8600884611af8565b90506109b5600582611af8565b6109c0600583611a79565b6109cb906005611b0c565b6109d59190611aa3565b90506109e2600884611a79565b6109ed906008611b0c565b6001600160401b03168260ff166001600160401b0316901b876002018260198110610a1a57610a1a611ac9565b6004810490910180546001600160401b0360086003909416939093026101000a808204841690941883168402929093021990921617905550819050610a5e81611adf565b915050610944565b50610a6f6117da565b60005b6019811015610ae157856002018160198110610a9057610a90611ac9565b600491828204019190066008029054906101000a90046001600160401b03166001600160401b0316828260198110610aca57610aca611ac9565b602002015280610ad981611adf565b915050610a72565b50610aeb81610b94565b905060005b6019811015610b6757818160198110610b0b57610b0b611ac9565b6020020151866002018260198110610b2557610b25611ac9565b600491828204019190066008026101000a8154816001600160401b0302191690836001600160401b031602179055508080610b5f90611adf565b915050610af0565b506088831015610b775750610b8e565b610b848360888187611ce2565b9350935050610931565b50505050565b610b9c6117da565b610ba46117f9565b610bac6117f9565b610bb46117da565b600060405180610300016040528060018152602001618082815260200167800000000000808a8152602001678000000080008000815260200161808b81526020016380000001815260200167800000008000808181526020016780000000000080098152602001608a81526020016088815260200163800080098152602001638000000a8152602001638000808b815260200167800000000000008b8152602001678000000000008089815260200167800000000000800381526020016780000000000080028152602001678000000000000080815260200161800a815260200167800000008000000a81526020016780000000800080818152602001678000000000008080815260200163800000018152602001678000000080008008815250905060005b6018811015611783576080878101516060808a01516040808c01516020808e01518e511890911890921890931889526101208b01516101008c015160e08d015160c08e015160a08f0151181818189089018190526101c08b01516101a08c01516101808d01516101608e01516101408f0151181818189289019283526102608b01516102408c01516102208d01516102008e01516101e08f015118181818918901919091526103008a01516102e08b01516102c08c01516102a08d01516102808e0151181818189288018390526001600160401b0360028202166001603f1b91829004179092188652510485600260200201516002026001600160401b03161785600060200201511884600160200201526001603f1b856003602002015181610e0557610e05611a63565b0485600360200201516002026001600160401b03161785600160200201511884600260200201526001603f1b856004602002015181610e4657610e46611a63565b0485600460200201516002026001600160401b03161785600260058110610e6f57610e6f611ac9565b602002015118606085015284516001603f1b9086516060808901519390920460029091026001600160401b031617909118608086810191825286518a5118808b5287516020808d018051909218825289516040808f0180519092189091528a518e8801805190911890528a51948e0180519095189094528901805160a08e0180519091189052805160c08e0180519091189052805160e08e018051909118905280516101008e0180519091189052516101208d018051909118905291880180516101408d018051909118905280516101608d018051909118905280516101808d018051909118905280516101a08d0180519091189052516101c08c018051909118905292870180516101e08c018051909118905280516102008c018051909118905280516102208c018051909118905280516102408c0180519091189052516102608b018051909118905281516102808b018051909118905281516102a08b018051909118905281516102c08b018051909118905281516102e08b018051909118905290516103008a01805190911890529084525163100000009060208901516001600160401b03641000000000909102169190041761010084015260408701516001603d1b9060408901516001600160401b03600890910216919004176101608401526060870151628000009060608901516001600160401b036502000000000090910216919004176102608401526080870151654000000000009060808901516001600160401b036204000090910216919004176102c084015260a08701516001603f1b900487600560200201516002026001600160401b031617836002601981106110df576110df611ac9565b602002015260c08701516210000081046001602c1b9091026001600160401b039081169190911760a085015260e0880151664000000000000081046104009091028216176101a08501526101008801516208000081046520000000000090910282161761020085015261012088015160048082029092166001603e1b909104176103008501526101408801516101408901516001600160401b036001603e1b909102169190041760808401526101608701516001603a1b906101608901516001600160401b036040909102169190041760e084015261018087015162200000906101808901516001600160401b036001602b1b90910216919004176101408401526101a08701516602000000000000906101a08901516001600160401b0361800090910216919004176102408401526101c08701516008906101c08901516001600160401b036001603d1b90910216919004176102a08401526101e0870151641000000000906101e08901516001600160401b03631000000090910216919004176020840152610200808801516102008901516001600160401b0366800000000000009091021691900417610120840152610220870151648000000000906102208901516001600160401b03630200000090910216919004176101808401526102408701516001602b1b906102408901516001600160401b036220000090910216919004176101e0840152610260870151610100906102608901516001600160401b03600160381b90910216919004176102e0840152610280870151642000000000906102808901516001600160401b036308000000909102169190041760608401526102a08701516001602c1b906102a08901516001600160401b0362100000909102169190041760c08401526102c08701516302000000906102c08901516001600160401b0364800000000090910216919004176101c08401526102e0870151600160381b906102e08901516001600160401b036101009091021691900417610220840152610300870151660400000000000090048760186020020151614000026001600160401b031617836014602002015282600a602002015183600560200201511916836000602002015118876000602002015282600b602002015183600660200201511916836001602002015118876001602002015282600c602002015183600760200201511916836002602002015118876002602002015282600d602002015183600860200201511916836003602002015118876003602002015282600e602002015183600960200201511916836004602002015118876004602002015282600f602002015183600a602002015119168360056020020151188760056020020152826010602002015183600b602002015119168360066020020151188760066020020152826011602002015183600c602002015119168360076020020151188760076020020152826012602002015183600d602002015119168360086020020151188760086020020152826013602002015183600e602002015119168360096020020151188760096020020152826014602002015183600f6020020151191683600a60200201511887600a602002015282601560200201518360106020020151191683600b60200201511887600b602002015282601660200201518360116020020151191683600c60200201511887600c602002015282601760200201518360126020020151191683600d60200201511887600d602002015282601860200201518360136020020151191683600e60200201511887600e602002015282600060200201518360146020020151191683600f60200201511887600f602002015282600160200201518360156020020151191683601060200201511887601060200201528260026020020151836016602002015119168360116020020151188760116020020152826003602002015183601760200201511916836012602002015118876012602002015282600460200201518360186020020151191683601360200201511887601360200201528260056020020151836000602002015119168360146020020151188760146020020152826006602002015183600160200201511916836015602002015118876015602002015282600760200201518360026020020151191683601660200201511887601660200201528260086020020151836003602002015119168360176020020151188760176020020152826009602002015183600460200201511916836018602002015118876018602002015281816018811061177157611771611ac9565b60200201518751188752600101610cda565b509495945050505050565b50805461179a90611a29565b6000825580601f106117aa575050565b601f0160209004906000526020600020908101906117c89190611817565b50565b506117c8906007810190611817565b6040518061032001604052806019906020820280368337509192915050565b6040518060a001604052806005906020820280368337509192915050565b5b8082111561182c5760008155600101611818565b5090565b80356001600160401b038116811461184757600080fd5b919050565b6000806040838503121561185f57600080fd5b8235915061186f60208401611830565b90509250929050565b6000815180845260005b8181101561189e57602081850181015186830182015201611882565b506000602082860101526020601f19601f83011685010191505092915050565b6020815260006118d16020830184611878565b9392505050565b60008083601f8401126118ea57600080fd5b5081356001600160401b0381111561190157600080fd5b60208301915083602082850101111561191957600080fd5b9250929050565b6000806000806060858703121561193657600080fd5b84356001600160401b0381111561194c57600080fd5b611958878288016118d8565b909550935061196b905060208601611830565b9396929550929360400135925050565b60006020828403121561198d57600080fd5b81356001600160a01b03811681146118d157600080fd5b6001600160401b03841681526060602082015260006119c66060830185611878565b9050826040830152949350505050565b6000806000604084860312156119eb57600080fd5b83356001600160401b03811115611a0157600080fd5b611a0d868287016118d8565b9094509250611a20905060208501611830565b90509250925092565b600181811c90821680611a3d57607f821691505b602082108103611a5d57634e487b7160e01b600052602260045260246000fd5b50919050565b634e487b7160e01b600052601260045260246000fd5b600082611a8857611a88611a63565b500690565b634e487b7160e01b600052601160045260246000fd5b808201808211156101d3576101d3611a8d565b818103818111156101d3576101d3611a8d565b634e487b7160e01b600052603260045260246000fd5b600060018201611af157611af1611a8d565b5060010190565b600082611b0757611b07611a63565b500490565b80820281158282048414176101d3576101d3611a8d565b634e487b7160e01b600052604160045260246000fd5b601f821115611b8357600081815260208120601f850160051c81016020861015611b605750805b601f850160051c820191505b81811015611b7f57828155600101611b6c565b5050505b505050565b81516001600160401b03811115611ba157611ba1611b23565b611bb581611baf8454611a29565b84611b39565b602080601f831160018114611bea5760008415611bd25750858301515b600019600386901b1c1916600185901b178555611b7f565b600085815260208120601f198616915b82811015611c1957888601518255948401946001909101908401611bfa565b5085821015611c375787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b6000602080835260008454611c5b81611a29565b80848701526040600180841660008114611c7c5760018114611c9657611cc4565b60ff1985168984015283151560051b890183019550611cc4565b896000528660002060005b85811015611cbc5781548b8201860152908301908801611ca1565b8a0184019650505b509398975050505050505050565b8183823760009101908152919050565b60008085851115611cf257600080fd5b83861115611cff57600080fd5b505082019391909203915056fea26469706673582212203815be7380ffb93ff272136f4d76ce414d894e1790b2a591fe81746083fa972664736f6c63430008110033",
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
	parsed, err := abi.JSON(strings.NewReader(HashProofHelperABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
	parsed, err := abi.JSON(strings.NewReader(IOneStepProofEntryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
	parsed, err := abi.JSON(strings.NewReader(IOneStepProverABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
	Bin: "0x60806040523480156200001157600080fd5b506040516200291a3803806200291a8339810160408190526200003491620000a5565b600080546001600160a01b039586166001600160a01b031991821617909155600180549486169482169490941790935560028054928516928416929092179091556003805491909316911617905562000102565b80516001600160a01b0381168114620000a057600080fd5b919050565b60008060008060808587031215620000bc57600080fd5b620000c78562000088565b9350620000d76020860162000088565b9250620000e76040860162000088565b9150620000f76060860162000088565b905092959194509250565b61280880620001126000396000f3fe608060405234801561001057600080fd5b50600436106100625760003560e01c80631f128bc01461006757806330a5509f146100975780635f52fd7c146100aa57806366e5d9c3146100bd578063b5112fd2146100d0578063c39619c4146100f1575b600080fd5b60015461007a906001600160a01b031681565b6040516001600160a01b0390911681526020015b60405180910390f35b60005461007a906001600160a01b031681565b60035461007a906001600160a01b031681565b60025461007a906001600160a01b031681565b6100e36100de366004611c5f565b610104565b60405190815260200161008e565b6100e36100ff366004611cfa565b610775565b600061010e611b18565b610116611b8a565b6040805160208101909152606081526040805180820190915260008082526020820152600061014688888361086f565b90955090508861015586610a2a565b1461019d5760405162461bcd60e51b815260206004820152601360248201527209a828690929c8abe848a8c9ea48abe9082a69606b1b60448201526064015b60405180910390fd5b6000855160028111156101b2576101b2611d0c565b1461028b576101bf611bdb565b6101ca898984610b8d565b60808801519093509091506101de82610c68565b1461021e5760405162461bcd60e51b815260206004820152601060248201526f4241445f474c4f42414c5f535441544560801b6044820152606401610194565b60018651600281111561023357610233611d0c565b14801561023e57508a155b801561025e57508b3561025382602001515190565b6001600160401b0316105b156102825761027586608001518d60400135610cd0565b965050505050505061076c565b61027586610a2a565b6508000000000061029d8b6001611d38565b036102bb57600285526102af85610a2a565b9550505050505061076c565b6102c6888883610e45565b90945090506102d6888883610ef1565b80925081945050508461010001516103038660a0015163ffffffff168686610fcb9092919063ffffffff16565b1461033f5760405162461bcd60e51b815260206004820152600c60248201526b1353d115531154d7d493d3d560a21b6044820152606401610194565b6040805160208101909152606081526040805160208101909152606081526103688a8a85611014565b90945092506103788a8a85610ef1565b935091506103878a8a85610ef1565b809450819250505060006103b08860e0015163ffffffff16868561106e9092919063ffffffff16565b905060006103d38960c0015163ffffffff1683856110b49092919063ffffffff16565b90508760600151811461041d5760405162461bcd60e51b815260206004820152601260248201527110905117d1955390d51253d394d7d493d3d560721b6044820152606401610194565b506104309250899150839050818b611d4b565b975097505060008460a0015163ffffffff16905060018560e0018181516104579190611d75565b63ffffffff1690525081516000602861ffff83161080159061047e5750603561ffff831611155b8061049e5750603661ffff83161080159061049e5750603e61ffff831611155b806104ad575061ffff8216603f145b806104bc575061ffff82166040145b156104d357506001546001600160a01b03166106c8565b61ffff8216604514806104ea575061ffff82166050145b806105185750604661ffff831610801590610518575061050c60096046611d99565b61ffff168261ffff1611155b806105465750606761ffff831610801590610546575061053a60026067611d99565b61ffff168261ffff1611155b806105665750606a61ffff8316108015906105665750607861ffff831611155b806105945750605161ffff831610801590610594575061058860096051611d99565b61ffff168261ffff1611155b806105c25750607961ffff8316108015906105c257506105b660026079611d99565b61ffff168261ffff1611155b806105e25750607c61ffff8316108015906105e25750608a61ffff831611155b806105f1575061ffff821660a7145b8061060e575061ffff821660ac148061060e575061ffff821660ad145b8061062e575060c061ffff83161080159061062e575060c461ffff831611155b8061064e575060bc61ffff83161080159061064e575060bf61ffff831611155b1561066557506002546001600160a01b03166106c8565b61801061ffff831610801590610681575061801361ffff831611155b806106a3575061802061ffff8316108015906106a3575061802261ffff831611155b156106ba57506003546001600160a01b03166106c8565b506000546001600160a01b03165b806001600160a01b03166397cc779a8e8989888f8f6040518763ffffffff1660e01b81526004016106fe96959493929190611ef6565b600060405180830381865afa15801561071b573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405261074391908101906124c7565b9097509550610753858488610fcb565b61010088015261076287610a2a565b9750505050505050505b95945050505050565b6000600161078960a08401608085016125e9565b600281111561079a5761079a611d0c565b036107e3576107b66107b13684900384018461260d565b610c68565b6040516020016107c691906126cf565b604051602081830303815290604052805190602001209050919050565b60026107f560a08401608085016125e9565b600281111561080657610806611d0c565b0361082d5761081d6107b13684900384018461260d565b6040516020016107c691906126f4565b60405162461bcd60e51b81526020600482015260126024820152714241445f4d414348494e455f53544154555360701b6044820152606401610194565b919050565b610877611b18565b81600080610886878785611129565b9350905060ff811660000361089e5760009150610904565b8060ff166001036108b25760019150610904565b8060ff166002036108c65760029150610904565b60405162461bcd60e51b8152602060048201526013602482015272554e4b4e4f574e5f4d4143485f53544154555360681b6044820152606401610194565b5061090d611c00565b610915611c00565b60008060008061093660408051808201909152606081526000602082015290565b60006109438e8e8c61115f565b9a5097506109528e8e8c61115f565b9a5096506109618e8e8c61125e565b9a5091506109708e8e8c611386565b9a50955061097f8e8e8c6113a2565b9a50945061098e8e8e8c6113a2565b9a50935061099d8e8e8c6113a2565b9a5092506109ac8e8e8c611386565b809b5081925050506040518061012001604052808a60028111156109d2576109d2611d0c565b81526020018981526020018881526020018381526020018781526020018663ffffffff1681526020018563ffffffff1681526020018463ffffffff168152602001828152509a50505050505050505050935093915050565b60008082516002811115610a4057610a40611d0c565b03610af557610a528260200151611406565b610a5f8360400151611406565b610a6c846060015161148b565b608085015160a086015160c087015160e0808901516101008a01516040516f26b0b1b434b73290393ab73734b7339d60811b602082015260308101999099526050890197909752607088019590955260908701939093526001600160e01b031991831b821660b0870152821b811660b486015291901b1660b883015260bc82015260dc016107c6565b600182516002811115610b0a57610b0a611d0c565b03610b245781608001516040516020016107c691906126cf565b600282516002811115610b3957610b39611d0c565b03610b535781608001516040516020016107c691906126f4565b60405162461bcd60e51b815260206004820152600f60248201526e4241445f4d4143485f53544154555360881b6044820152606401610194565b610b95611bdb565b81610b9e611c1e565b610ba6611c1e565b60005b600260ff82161015610bf157610bc0888886611386565b848360ff1660028110610bd557610bd5612718565b6020020191909152935080610be98161272e565b915050610ba9565b5060005b600260ff82161015610c4b57610c0c888886611524565b838360ff1660028110610c2157610c21612718565b6001600160401b039093166020939093020191909152935080610c438161272e565b915050610bf5565b506040805180820190915291825260208201529590945092505050565b80518051602091820151828401518051908401516040516c23b637b130b61039ba30ba329d60991b95810195909552602d850193909352604d8401919091526001600160c01b031960c091821b8116606d85015291901b166075820152600090607d016107c6565b60408051600380825260808201909252600091829190816020015b6040805180820190915260008082526020820152815260200190600190039081610ceb575050604080518082018252600080825260209182018190528251808401909352600483529082015290915081600081518110610d4d57610d4d612718565b6020026020010181905250610d626000611582565b81600181518110610d7557610d75612718565b6020026020010181905250610d8a6000611582565b81600281518110610d9d57610d9d612718565b6020908102919091018101919091526040805180830182528381528151808301909252808252600092820192909252610dd4611c00565b604080518082018252606080825260006020808401829052845161012081018652828152908101879052938401859052908301829052608083018a905260a0830181905260c0830181905260e0830152610100820188905290610e3681610a2a565b96505050505050505b92915050565b610e4d611b8a565b604080516060810182526000808252602082018190529181018290528391906000806000610e7c8a8a88611386565b96509450610e8b8a8a886115b5565b96509350610e9a8a8a88611386565b96509250610ea98a8a88611386565b96509150610eb88a8a886113a2565b6040805160a08101825297885260208801969096529486019390935250606084015263ffffffff16608083015290969095509350505050565b604080516020810190915260608152816000610f0e868684611129565b92509050600060ff82166001600160401b03811115610f2f57610f2f61208e565b604051908082528060200260200182016040528015610f58578160200160208202803683370190505b50905060005b8260ff168160ff161015610faf57610f77888886611386565b838360ff1681518110610f8c57610f8c612718565b602002602001018196508281525050508080610fa79061272e565b915050610f5e565b5060405180602001604052808281525093505050935093915050565b600061100c8484610fdb85611630565b6040518060400160405280601381526020017226b7b23ab6329036b2b935b632903a3932b29d60691b81525061169d565b949350505050565b604080518082019091526000808252602082015281600080611037878785611772565b935091506110468787856117cb565b6040805180820190915261ffff90941684526020840191909152919791965090945050505050565b600061100c848461107e85611820565b6040518060400160405280601881526020017724b739ba393ab1ba34b7b71036b2b935b632903a3932b29d60411b81525061169d565b60405168233ab731ba34b7b71d60b91b602082015260298101829052600090819060490160405160208183030381529060405280519060200120905061076c85858360405180604001604052806015815260200174233ab731ba34b7b71036b2b935b632903a3932b29d60591b81525061169d565b60008184848281811061113e5761113e612718565b919091013560f81c92508190506111548161274d565b915050935093915050565b611167611c00565b816000611175868684611386565b9250905060006111868787856117cb565b935090506000816001600160401b038111156111a4576111a461208e565b6040519080825280602002602001820160405280156111e957816020015b60408051808201909152600080825260208201528152602001906001900390816111c25790505b50905060005b81518110156112375761120389898761186a565b83838151811061121557611215612718565b602002602001018197508290525050808061122f9061274d565b9150506111ef565b50604080516060810182529081019182529081526020810192909252509590945092505050565b604080518082019091526060815260006020820152816000611281868684611386565b92509050606086868481811061129957611299612718565b909101356001600160f81b03191615905061132157826112b88161274d565b604080516001808252818301909252919550909150816020015b6112da611c3c565b8152602001906001900390816112d25790505090506112fa878785611966565b8260008151811061130d5761130d612718565b602002602001018195508290525050611365565b8261132b8161274d565b60408051600080825260208201909252919550909150611361565b61134e611c3c565b8152602001906001900390816113465790505b5090505b60405180604001604052808281526020018381525093505050935093915050565b600081816113958686846117cb565b9097909650945050505050565b600081815b60048110156113fd5760088363ffffffff16901b92508585838181106113cf576113cf612718565b919091013560f81c939093179250816113e78161274d565b92505080806113f59061274d565b9150506113a7565b50935093915050565b60208101518151515160005b8181101561148457835161142f9061142a90836119ff565b611a37565b6040516b2b30b63ab29039ba30b1b59d60a11b6020820152602c810191909152604c8101849052606c01604051602081830303815290604052805190602001209250808061147c9061274d565b915050611412565b5050919050565b602081015160005b82515181101561151e576114c3836000015182815181106114b6576114b6612718565b6020026020010151611a54565b6040517129ba30b1b590333930b6b29039ba30b1b59d60711b602082015260328101919091526052810183905260720160405160208183030381529060405280519060200120915080806115169061274d565b915050611493565b50919050565b600081815b60088110156113fd576008836001600160401b0316901b925085858381811061155457611554612718565b919091013560f81c9390931792508161156c8161274d565b925050808061157a9061274d565b915050611529565b604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b604080516060810182526000808252602082018190529181019190915281600080806115e2888886611524565b945092506115f1888886611524565b94509150611600888886611386565b604080516060810182526001600160401b0396871681529490951660208501529383015250969095509350505050565b600081600001516116448360200151611ac4565b6040848101516060860151608087015192516626b7b23ab6329d60c91b6020820152602781019590955260478501939093526067840152608783019190915260e01b6001600160e01b03191660a782015260ab016107c6565b8160005b8551518110156117695784600116600003611705578282876000015183815181106116ce576116ce612718565b60200260200101516040516020016116e893929190612766565b604051602081830303815290604052805190602001209150611750565b828660000151828151811061171c5761171c612718565b60200260200101518360405160200161173793929190612766565b6040516020818303038152906040528051906020012091505b60019490941c93806117618161274d565b9150506116a1565b50949350505050565b600081815b60028110156113fd5760088361ffff16901b925085858381811061179d5761179d612718565b919091013560f81c939093179250816117b58161274d565b92505080806117c39061274d565b915050611777565b600081815b60208110156113fd57600883901b92508585838181106117f2576117f2612718565b919091013560f81c9390931792508161180a8161274d565b92505080806118189061274d565b9150506117d0565b6000816000015182602001516040516020016107c69291906b24b739ba393ab1ba34b7b71d60a11b815260f09290921b6001600160f01b031916600c830152600e820152602e0190565b604080518082019091526000808252602082015281600085858381811061189357611893612718565b919091013560f81c91508290506118a98161274d565b9250506118b4600690565b60068111156118c5576118c5611d0c565b60ff168160ff16111561190b5760405162461bcd60e51b815260206004820152600e60248201526d4241445f56414c55455f5459504560901b6044820152606401610194565b60006119188787856117cb565b809450819250505060405180604001604052808360ff16600681111561194057611940611d0c565b600681111561195157611951611d0c565b81526020018281525093505050935093915050565b61196e611c3c565b81611989604080518082019091526000808252602082015290565b600080600061199989898761186a565b955093506119a8898987611386565b955092506119b78989876113a2565b955091506119c68989876113a2565b60408051608081018252968752602087019590955263ffffffff9384169486019490945290911660608401525090969095509350505050565b60408051808201909152600080825260208201528251805183908110611a2757611a27612718565b6020026020010151905092915050565b6000816000015182602001516040516020016107c692919061279d565b6000611a638260000151611a37565b602080840151604080860151606087015191516b29ba30b1b590333930b6b29d60a11b94810194909452602c840194909452604c8301919091526001600160e01b031960e093841b8116606c840152921b90911660708201526074016107c6565b805160208083015160408085015190516626b2b6b7b93c9d60c91b938101939093526001600160c01b031960c094851b811660278501529190931b16602f82015260378101919091526000906057016107c6565b6040805161012081019091528060008152602001611b34611c00565b8152602001611b41611c00565b8152602001611b6160408051808201909152606081526000602082015290565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b6040805160a081019091526000815260208101611bc0604080516060810182526000808252602082018190529181019190915290565b81526000602082018190526040820181905260609091015290565b6040518060400160405280611bee611c1e565b8152602001611bfb611c1e565b905290565b60408051606080820183529181019182529081526000602082015290565b60405180604001604052806002906020820280368337509192915050565b6040805160c0810190915260006080820181815260a08301919091528190611bc0565b600080600080600085870360c0811215611c7857600080fd5b6060811215611c8657600080fd5b50859450606086013593506080860135925060a08601356001600160401b0380821115611cb257600080fd5b818801915088601f830112611cc657600080fd5b813581811115611cd557600080fd5b896020828501011115611ce757600080fd5b9699959850939650602001949392505050565b600060a0828403121561151e57600080fd5b634e487b7160e01b600052602160045260246000fd5b634e487b7160e01b600052601160045260246000fd5b80820180821115610e3f57610e3f611d22565b60008085851115611d5b57600080fd5b83861115611d6857600080fd5b5050820193919092039150565b63ffffffff818116838216019080821115611d9257611d92611d22565b5092915050565b61ffff818116838216019080821115611d9257611d92611d22565b60038110611dc457611dc4611d0c565b9052565b805160078110611dda57611dda611d0c565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b80841015611e3757611e23828651611dc8565b938201936001939093019290850190611e10565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b82811015611eb6578451611e82858251611dc8565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101611e6d565b509687015197909601969096525093949350505050565b81835281816020850137506000828201602090810191909152601f909101601f19169091010190565b8635815260006101c060208901356001600160a01b038116808214611f1a57600080fd5b806020860152505060408901356040840152806060840152611f3f8184018951611db4565b5060208701516101206101e0840152611f5c6102e0840182611de7565b905060408801516101bf198085840301610200860152611f7c8383611de7565b925060608a01519150808584030161022086015250611f9b8282611e4b565b915050608088015161024084015260a0880151611fc161026085018263ffffffff169052565b5060c088015163ffffffff81166102808501525060e088015163ffffffff81166102a0850152506101008801516102c084015261205660808401888051825260208101516001600160401b0380825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b855161ffff1661016084015260208601516101808401528281036101a0840152612081818587611ecd565b9998505050505050505050565b634e487b7160e01b600052604160045260246000fd5b604080519081016001600160401b03811182821017156120c6576120c661208e565b60405290565b604051602081016001600160401b03811182821017156120c6576120c661208e565b604051608081016001600160401b03811182821017156120c6576120c661208e565b60405160a081016001600160401b03811182821017156120c6576120c661208e565b604051606081016001600160401b03811182821017156120c6576120c661208e565b60405161012081016001600160401b03811182821017156120c6576120c661208e565b604051601f8201601f191681016001600160401b038111828210171561219f5761219f61208e565b604052919050565b600381106121b457600080fd5b50565b805161086a816121a7565b60006001600160401b038211156121db576121db61208e565b5060051b60200190565b6000604082840312156121f757600080fd5b6121ff6120a4565b905081516007811061221057600080fd5b808252506020820151602082015292915050565b6000604080838503121561223757600080fd5b61223f6120a4565b915082516001600160401b038082111561225857600080fd5b8185019150602080838803121561226e57600080fd5b6122766120cc565b83518381111561228557600080fd5b80850194505087601f85011261229a57600080fd5b835192506122af6122aa846121c2565b612177565b83815260069390931b840182019282810190898511156122ce57600080fd5b948301945b848610156122f4576122e58a876121e5565b825294860194908301906122d3565b8252508552948501519484019490945250909392505050565b805163ffffffff8116811461086a57600080fd5b6000604080838503121561233457600080fd5b61233c6120a4565b915082516001600160401b0381111561235457600080fd5b8301601f8101851361236557600080fd5b805160206123756122aa836121c2565b82815260a0928302840182019282820191908985111561239457600080fd5b948301945b848610156123fd5780868b0312156123b15760008081fd5b6123b96120ee565b6123c38b886121e5565b8152878701518582015260606123da81890161230d565b898301526123ea6080890161230d565b9082015283529485019491830191612399565b50808752505080860151818601525050505092915050565b6001600160401b03811681146121b457600080fd5b600081830360e081121561243d57600080fd5b612445612110565b8351815291506060601f198201121561245d57600080fd5b50612466612132565b602083015161247481612415565b8152604083015161248481612415565b8060208301525060608301516040820152806020830152506080820151604082015260a082015160608201526124bc60c0830161230d565b608082015292915050565b6000806101008084860312156124dc57600080fd5b83516001600160401b03808211156124f357600080fd5b90850190610120828803121561250857600080fd5b612510612154565b612519836121b7565b815260208301518281111561252d57600080fd5b61253989828601612224565b60208301525060408301518281111561255157600080fd5b61255d89828601612224565b60408301525060608301518281111561257557600080fd5b61258189828601612321565b6060830152506080830151608082015261259d60a0840161230d565b60a08201526125ae60c0840161230d565b60c08201526125bf60e0840161230d565b60e08201528383015184820152809550505050506125e0846020850161242a565b90509250929050565b6000602082840312156125fb57600080fd5b8135612606816121a7565b9392505050565b60006080828403121561261f57600080fd5b6126276120a4565b83601f84011261263657600080fd5b61263e6120a4565b80604085018681111561265057600080fd5b855b8181101561266a578035845260209384019301612652565b5081845286605f87011261267d57600080fd5b6126856120a4565b9250829150608086018781111561269b57600080fd5b808210156126c05781356126ae81612415565b8452602093840193919091019061269b565b50506020830152509392505050565b7026b0b1b434b732903334b734b9b432b21d60791b8152601181019190915260310190565b6f26b0b1b434b7329032b93937b932b21d60811b8152601081019190915260300190565b634e487b7160e01b600052603260045260246000fd5b600060ff821660ff810361274457612744611d22565b60010192915050565b60006001820161275f5761275f611d22565b5060010190565b6000845160005b81811015612787576020818801810151858301520161276d565b5091909101928352506020820152604001919050565b652b30b63ab29d60d11b81526000600784106127bb576127bb611d0c565b5060f89290921b600683015260078201526027019056fea264697066735822122093b1af4e0a9d908b70c6b3339912ca8b5f7e354d6593ef906d0387d93a2c466864736f6c63430008110033",
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
	parsed, err := abi.JSON(strings.NewReader(OneStepProofEntryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220bd50800c5938c8d32471764fef41af95f6e2a83b9b21667b1088e8bbd1ab1fd464736f6c63430008110033",
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
	parsed, err := abi.JSON(strings.NewReader(OneStepProofEntryLibABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
	Bin: "0x608060405234801561001057600080fd5b5061263e806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c806397cc779a14610030575b600080fd5b61004361003e366004611c0b565b61005a565b604051610051929190611df5565b60405180910390f35b610062611a8b565b61006a611b34565b6100738761229d565b9150610084368790038701876123a0565b905060006100956020870187612437565b9050611b7e61ffff82166100ac57506102bd61029f565b60001961ffff8316016100c257506102c861029f565b600e1961ffff8316016100d857506102cf61029f565b600f1961ffff8316016100ee57506103fa61029f565b6180081961ffff831601610105575061049461029f565b6180091961ffff83160161011c575061054061029f565b60101961ffff831601610132575061062c61029f565b6180021961ffff8316016101495750610a0b61029f565b6180031961ffff8316016101605750610a4a61029f565b601f1961ffff8316016101765750610aa861029f565b60201961ffff83160161018c5750610aea61029f565b60221961ffff8316016101a25750610b2f61029f565b60231961ffff8316016101b85750610b5761029f565b6180011961ffff8316016101cf5750610b8761029f565b60191961ffff8316016101e55750610c2461029f565b601a1961ffff8316016101fb5750610c3161029f565b604161ffff8316108015906102155750604461ffff831611155b156102235750610ca061029f565b61ffff8216618005148061023c575061ffff8216618006145b1561024a5750610d9461029f565b6180071961ffff8316016102615750610e6561029f565b60405162461bcd60e51b815260206004820152600e60248201526d494e56414c49445f4f50434f444560901b60448201526064015b60405180910390fd5b6102b084848989898663ffffffff16565b5050965096945050505050565b505060029092525050565b5050505050565b60006102de8660600151610e74565b9050600481515160068111156102f6576102f6611cc6565b0361031c578560025b9081600281111561031257610312611cc6565b81525050506102c8565b6006815151600681111561033257610332611cc6565b146103785760405162461bcd60e51b8152602060048201526016602482015275494e56414c49445f52455455524e5f50435f5459504560501b6044820152606401610296565b805160209081015190819081901c604082901c606083901c156103d65760405162461bcd60e51b8152602060048201526016602482015275494e56414c49445f52455455524e5f50435f4441544160501b6044820152606401610296565b63ffffffff92831660e08b015290821660c08a01521660a088015250505050505050565b61041161040686610f14565b602087015190610f77565b60006104208660600151610f87565b905061043d6104328260400151610fd3565b602088015190610f77565b61044d6104328260600151610fd3565b602084013563ffffffff811681146104775760405162461bcd60e51b81526004016102969061245b565b63ffffffff1660c08701525050600060e090940193909352505050565b6104a061040686610f14565b6104b06104068660a00151610fd3565b6104c06104068560800151610fd3565b6020808401359081901c604082901c1561051c5760405162461bcd60e51b815260206004820152601a60248201527f4241445f43524f53535f4d4f44554c455f43414c4c5f444154410000000000006044820152606401610296565b63ffffffff90811660a08801521660c08601525050600060e0909301929092525050565b61054c61040686610f14565b61055c6104068660a00151610fd3565b61056c6104068560800151610fd3565b600061057b8660600151610f87565b9050806060015163ffffffff16600003610597578560026102ff565b602084013563ffffffff811681146105f15760405162461bcd60e51b815260206004820152601d60248201527f4241445f43414c4c45525f494e5445524e414c5f43414c4c5f444154410000006044820152606401610296565b604082015163ffffffff1660a08801526060820151610611908290612498565b63ffffffff1660c08801525050600060e08601525050505050565b60008061064461063f8860200151611006565b61102b565b90506000806000808060006106656040518060200160405280606081525090565b6106708b8b876110bc565b9550935061067f8b8b87611123565b909650945061068f8b8b8761113f565b9550925061069e8b8b876110bc565b955091506106ad8b8b87611123565b90975094506106bd8b8b87611175565b6040516d21b0b6361034b73234b932b1ba1d60911b60208201526001600160c01b031960c088901b16602e8201526036810189905290965090915060009060560160408051601f19818403018152919052805160209182012091508d013581146107625760405162461bcd60e51b81526020600482015260166024820152754241445f43414c4c5f494e4449524543545f4441544160501b6044820152606401610296565b610778826001600160401b03871686868c61124f565b90508d6040015181146107bf5760405162461bcd60e51b815260206004820152600f60248201526e10905117d51050931154d7d493d3d5608a1b6044820152606401610296565b826001600160401b03168963ffffffff16106107e957505060028d52506102c89650505050505050565b5050505050600061080a604080518082019091526000808252602082015290565b6040805160208101909152606081526108248a8a86611123565b945092506108338a8a866112f1565b945091506108428a8a86611175565b94509050600061085f8263ffffffff808b1690879087906113ed16565b90508681146108a45760405162461bcd60e51b815260206004820152601160248201527010905117d153115351539514d7d493d3d5607a1b6044820152606401610296565b8584146108d4578d60025b908160028111156108c2576108c2611cc6565b815250505050505050505050506102c8565b6004835160068111156108e9576108e9611cc6565b036108f6578d60026108af565b60058351600681111561090b5761090b611cc6565b03610969576020830151985063ffffffff891689146109645760405162461bcd60e51b81526020600482015260156024820152744241445f46554e435f5245465f434f4e54454e545360581b6044820152606401610296565b6109a1565b60405162461bcd60e51b815260206004820152600d60248201526c4241445f454c454d5f5459504560981b6044820152606401610296565b50505050505050506109b561043287610f14565b60006109c48760600151610f87565b90506109e16109d68260400151610fd3565b602089015190610f77565b6109f16109d68260600151610fd3565b5063ffffffff1660c0860152600060e08601525050505050565b602083013563ffffffff81168114610a355760405162461bcd60e51b81526004016102969061245b565b63ffffffff1660e09095019490945250505050565b6000610a5c61063f8760200151611006565b905063ffffffff811615610aa057602084013563ffffffff81168114610a945760405162461bcd60e51b81526004016102969061245b565b63ffffffff1660e08701525b505050505050565b6000610ab78660600151610f87565b90506000610acf826020015186602001358686611487565b6020880151909150610ae19082610f77565b50505050505050565b6000610af98660200151611006565b90506000610b0a8760600151610f87565b9050610b218160200151866020013584878761151f565b602090910152505050505050565b6000610b45856000015185602001358585611487565b6020870151909150610aa09082610f77565b6000610b668660200151611006565b9050610b7d8560000151856020013583868661151f565b9094525050505050565b6000610b968660200151611006565b90506000610ba78760200151611006565b90506000610bb88860200151611006565b905060006040518060800160405280838152602001886020013560001b8152602001610be38561102b565b63ffffffff168152602001610bf78661102b565b63ffffffff168152509050610c19818a606001516115b990919063ffffffff16565b505050505050505050565b610aa08560200151611006565b6000610c4361063f8760200151611006565b90506000610c548760200151611006565b90506000610c658860200151611006565b905063ffffffff831615610c87576020880151610c829082610f77565b610c96565b6020880151610c969083610f77565b5050505050505050565b6000610caf6020850185612437565b9050600060401961ffff831601610cc857506000610d4b565b60411961ffff831601610cdd57506001610d4b565b60421961ffff831601610cf257506002610d4b565b60431961ffff831601610d0757506003610d4b565b60405162461bcd60e51b8152602060048201526019602482015278434f4e53545f505553485f494e56414c49445f4f50434f444560381b6044820152606401610296565b610ae16040518060400160405280836006811115610d6b57610d6b611cc6565b815260200187602001356001600160401b03168152508860200151610f7790919063ffffffff16565b6040805180820190915260008082526020820152618005610db86020860186612437565b61ffff1603610de557610dce8660200151611006565b6040870151909150610de09082610f77565b610aa0565b618006610df56020860186612437565b61ffff1603610e1d57610e0b8660400151611006565b6020870151909150610de09082610f77565b60405162461bcd60e51b815260206004820152601c60248201527f4d4f56455f494e5445524e414c5f494e56414c49445f4f50434f4445000000006044820152606401610296565b6000610b4586602001516116a0565b610e7c611b88565b815151600114610e9e5760405162461bcd60e51b8152600401610296906124bc565b81518051600090610eb157610eb16124e7565b6020026020010151905060006001600160401b03811115610ed457610ed4611f1d565b604051908082528060200260200182016040528015610f0d57816020015b610efa611b88565b815260200190600190039081610ef25790505b5090915290565b604080518082018252600080825260209182015260e083015160c084015160a090940151835180850185526006815263ffffffff90921694831b67ffffffff0000000016949094179390921b63ffffffff60401b16929092179181019190915290565b8151610f8390826116d5565b5050565b610f8f611b88565b815151600114610fb15760405162461bcd60e51b8152600401610296906124bc565b81518051600090610fc457610fc46124e7565b60200260200101519050919050565b604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b604080518082019091526000808252602082015281516110259061179e565b92915050565b6020810151600090818351600681111561104757611047611cc6565b1461107e5760405162461bcd60e51b81526020600482015260076024820152662727aa2fa4999960c91b6044820152606401610296565b64010000000081106110255760405162461bcd60e51b81526020600482015260076024820152662120a22fa4999960c91b6044820152606401610296565b600081815b600881101561111a576008836001600160401b0316901b92508585838181106110ec576110ec6124e7565b919091013560f81c93909317925081611104816124fd565b9250508080611112906124fd565b9150506110c1565b50935093915050565b600081816111328686846118a7565b9097909650945050505050565b600081848482818110611154576111546124e7565b919091013560f81c925081905061116a816124fd565b915050935093915050565b60408051602081019091526060815281600061119286868461113f565b92509050600060ff82166001600160401b038111156111b3576111b3611f1d565b6040519080825280602002602001820160405280156111dc578160200160208202803683370190505b50905060005b8260ff168160ff161015611233576111fb888886611123565b838360ff1681518110611210576112106124e7565b60200260200101819650828152505050808061122b90612516565b9150506111e2565b5060405180602001604052808281525093505050935093915050565b604051652a30b136329d60d11b60208201526001600160f81b031960f885901b1660268201526001600160c01b031960c084901b166027820152602f81018290526000908190604f016040516020818303038152906040528051906020012090506112e6878783604051806040016040528060128152602001712a30b136329036b2b935b632903a3932b29d60711b8152506118fc565b979650505050505050565b604080518082019091526000808252602082015281600085858381811061131a5761131a6124e7565b919091013560f81c9150829050611330816124fd565b92505061133b600690565b600681111561134c5761134c611cc6565b60ff168160ff1611156113925760405162461bcd60e51b815260206004820152600e60248201526d4241445f56414c55455f5459504560901b6044820152606401610296565b600061139f8787856118a7565b809450819250505060405180604001604052808360ff1660068111156113c7576113c7611cc6565b60068111156113d8576113d8611cc6565b81526020018281525093505050935093915050565b600080836113fa846119d1565b6040516d2a30b136329032b632b6b2b73a1d60911b6020820152602e810192909252604e820152606e0160405160208183030381529060405280519060200120905061147d8686836040518060400160405280601a81526020017f5461626c6520656c656d656e74206d65726b6c6520747265653a0000000000008152506118fc565b9695505050505050565b604080518082019091526000808252602082015260006114b7604080518082019091526000808252602082015290565b6040805160208101909152606081526114d18686856112f1565b935091506114e0868685611175565b9350905060006114f1828985611a0b565b90508881146115125760405162461bcd60e51b815260040161029690612535565b5090979650505050505050565b600061153b604080518082019091526000808252602082015290565b60006115536040518060200160405280606081525090565b61155e8686846112f1565b909350915061156e868684611175565b92509050600061157f828a86611a0b565b90508981146115a05760405162461bcd60e51b815260040161029690612535565b6115ab828a8a611a0b565b9a9950505050505050505050565b8151516000906115ca906001612560565b6001600160401b038111156115e1576115e1611f1d565b60405190808252806020026020018201604052801561161a57816020015b611607611b88565b8152602001906001900390816115ff5790505b50905060005b83515181101561167657835180518290811061163e5761163e6124e7565b6020026020010151828281518110611658576116586124e7565b6020026020010181905250808061166e906124fd565b915050611620565b5081818460000151518151811061168f5761168f6124e7565b602090810291909101015290915250565b6040805180820190915260008082526020820152815151516116ce6116c6600183612573565b845190611a53565b9392505050565b8151516000906116e6906001612560565b6001600160401b038111156116fd576116fd611f1d565b60405190808252806020026020018201604052801561174257816020015b604080518082019091526000808252602082015281526020019060019003908161171b5790505b50905060005b835151811015611676578351805182908110611766576117666124e7565b6020026020010151828281518110611780576117806124e7565b60200260200101819052508080611796906124fd565b915050611748565b6040805180820190915260008082526020820152815180516117c290600190612573565b815181106117d2576117d26124e7565b60200260200101519050600060018360000151516117f09190612573565b6001600160401b0381111561180757611807611f1d565b60405190808252806020026020018201604052801561184c57816020015b60408051808201909152600080825260208201528152602001906001900390816118255790505b50905060005b8151811015610f0d57835180518290811061186f5761186f6124e7565b6020026020010151828281518110611889576118896124e7565b6020026020010181905250808061189f906124fd565b915050611852565b600081815b602081101561111a57600883901b92508585838181106118ce576118ce6124e7565b919091013560f81c939093179250816118e6816124fd565b92505080806118f4906124fd565b9150506118ac565b8160005b8551518110156119c857846001166000036119645782828760000151838151811061192d5761192d6124e7565b602002602001015160405160200161194793929190612586565b6040516020818303038152906040528051906020012091506119af565b828660000151828151811061197b5761197b6124e7565b60200260200101518360405160200161199693929190612586565b6040516020818303038152906040528051906020012091505b60019490941c93806119c0816124fd565b915050611900565b50949350505050565b6000816000015182602001516040516020016119ee9291906125bd565b604051602081830303815290604052805190602001209050919050565b6000611a4b8484611a1b856119d1565b604051806040016040528060128152602001712b30b63ab29036b2b935b632903a3932b29d60711b8152506118fc565b949350505050565b60408051808201909152600080825260208201528251805183908110611a7b57611a7b6124e7565b6020026020010151905092915050565b6040805161012081019091528060008152602001611ac060408051606080820183529181019182529081526000602082015290565b8152602001611ae660408051606080820183529181019182529081526000602082015290565b8152602001611b0b604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b6040805160a0810182526000808252825160608101845281815260208181018390529381019190915290918201905b81526000602082018190526040820181905260609091015290565b611b866125f2565b565b6040805160c0810190915260006080820181815260a08301919091528190611b63565b600060408284031215611bbd57600080fd5b50919050565b60008083601f840112611bd557600080fd5b5081356001600160401b03811115611bec57600080fd5b602083019150836020828501011115611c0457600080fd5b9250929050565b6000806000806000808688036101c0811215611c2657600080fd5b6060811215611c3457600080fd5b87965060608801356001600160401b0380821115611c5157600080fd5b90890190610120828c031215611c6657600080fd5b81975060e0607f1984011215611c7b57600080fd5b60808a019650611c8f8b6101608c01611bab565b95506101a08a0135925080831115611ca657600080fd5b5050611cb489828a01611bc3565b979a9699509497509295939492505050565b634e487b7160e01b600052602160045260246000fd5b60038110611cec57611cec611cc6565b9052565b805160078110611d0257611d02611cc6565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b80841015611d5f57611d4b828651611cf0565b938201936001939093019290850190611d38565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b82811015611dde578451611daa858251611cf0565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101611d95565b509687015197909601969096525093949350505050565b6000610100808352611e0a8184018651611cdc565b602085015161012084810152611e24610220850182611d0f565b9050604086015160ff198086840301610140870152611e438383611d0f565b925060608801519150808684030161016087015250611e628282611d73565b915050608086015161018085015260a0860151611e886101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e0860152509085015161020084015290506116ce60208301848051825260208101516001600160401b0380825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b604080519081016001600160401b0381118282101715611f5557611f55611f1d565b60405290565b604051602081016001600160401b0381118282101715611f5557611f55611f1d565b604051608081016001600160401b0381118282101715611f5557611f55611f1d565b60405161012081016001600160401b0381118282101715611f5557611f55611f1d565b60405160a081016001600160401b0381118282101715611f5557611f55611f1d565b604051606081016001600160401b0381118282101715611f5557611f55611f1d565b604051601f8201601f191681016001600160401b038111828210171561202e5761202e611f1d565b604052919050565b80356003811061204557600080fd5b919050565b60006001600160401b0382111561206357612063611f1d565b5060051b60200190565b60006040828403121561207f57600080fd5b612087611f33565b905081356007811061209857600080fd5b808252506020820135602082015292915050565b600060408083850312156120bf57600080fd5b6120c7611f33565b915082356001600160401b03808211156120e057600080fd5b818501915060208083880312156120f657600080fd5b6120fe611f5b565b83358381111561210d57600080fd5b80850194505087601f85011261212257600080fd5b833592506121376121328461204a565b612006565b83815260069390931b8401820192828101908985111561215657600080fd5b948301945b8486101561217c5761216d8a8761206d565b8252948601949083019061215b565b8252508552948501359484019490945250909392505050565b803563ffffffff8116811461204557600080fd5b600060408083850312156121bc57600080fd5b6121c4611f33565b915082356001600160401b038111156121dc57600080fd5b8301601f810185136121ed57600080fd5b803560206121fd6121328361204a565b82815260a0928302840182019282820191908985111561221c57600080fd5b948301945b848610156122855780868b0312156122395760008081fd5b612241611f7d565b61224b8b8861206d565b815287870135858201526060612262818901612195565b8983015261227260808901612195565b9082015283529485019491830191612221565b50808752505080860135818601525050505092915050565b600061012082360312156122b057600080fd5b6122b8611f9f565b6122c183612036565b815260208301356001600160401b03808211156122dd57600080fd5b6122e9368387016120ac565b6020840152604085013591508082111561230257600080fd5b61230e368387016120ac565b6040840152606085013591508082111561232757600080fd5b50612334368286016121a9565b6060830152506080830135608082015261235060a08401612195565b60a082015261236160c08401612195565b60c082015261237260e08401612195565b60e082015261010092830135928101929092525090565b80356001600160401b038116811461204557600080fd5b600081830360e08112156123b357600080fd5b6123bb611fc2565b833581526060601f19830112156123d157600080fd5b6123d9611fe4565b91506123e760208501612389565b82526123f560408501612389565b6020830152606084013560408301528160208201526080840135604082015260a0840135606082015261242a60c08501612195565b6080820152949350505050565b60006020828403121561244957600080fd5b813561ffff811681146116ce57600080fd5b6020808252600d908201526c4241445f43414c4c5f4441544160981b604082015260600190565b634e487b7160e01b600052601160045260246000fd5b63ffffffff8181168382160190808211156124b5576124b5612482565b5092915050565b6020808252601190820152700848288beae929c889eaebe988a9c8ea89607b1b604082015260600190565b634e487b7160e01b600052603260045260246000fd5b60006001820161250f5761250f612482565b5060010190565b600060ff821660ff810361252c5761252c612482565b60010192915050565b60208082526011908201527015d493d391d7d3515492d31157d493d3d5607a1b604082015260600190565b8082018082111561102557611025612482565b8181038181111561102557611025612482565b6000845160005b818110156125a7576020818801810151858301520161258d565b5091909101928352506020820152604001919050565b652b30b63ab29d60d11b81526000600784106125db576125db611cc6565b5060f89290921b6006830152600782015260270190565b634e487b7160e01b600052605160045260246000fdfea2646970667358221220da8af98e7c097b9af8b64d0bc88d672a2fe66c4a0fefb776df8151cf9125fb7a64736f6c63430008110033",
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
	parsed, err := abi.JSON(strings.NewReader(OneStepProver0ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
	Bin: "0x608060405234801561001057600080fd5b5061260f806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c806397cc779a14610030575b600080fd5b61004361003e366004611b58565b61005a565b604051610051929190611d42565b60405180910390f35b610062611a02565b6040805160a0810182526000808252825160608082018552828252602080830184905282860184905284019190915292820181905291810182905260808101919091526100ae876121ea565b91506100bf368790038701876122ed565b905060006100d06020870187612384565b9050611aab61801061ffff8316108015906100f1575061801361ffff831611155b156100ff57506101a8610189565b61801f1961ffff83160161011657506102f0610189565b6180201961ffff83160161012d5750610593610189565b6180211961ffff83160161014457506108b5610189565b60405162461bcd60e51b8152602060048201526015602482015274494e56414c49445f4d454d4f52595f4f50434f444560581b60448201526064015b60405180910390fd5b61019b8a85858a8a8a8763ffffffff16565b5050965096945050505050565b60006101b76020850185612384565b90506101c1611ab5565b60006101ce8585836108c1565b60808a015191935091506101e18361099c565b146102215760405162461bcd60e51b815260206004820152601060248201526f4241445f474c4f42414c5f535441544560801b6044820152606401610180565b61ffff8316618010148061023a575061ffff8316618011145b1561025c57610257888884896102528987818d6123a8565b610a10565b6102d4565b6180111961ffff841601610274576102578883610bb6565b6180121961ffff84160161028c576102578883610c4d565b60405162461bcd60e51b815260206004820152601a60248201527f494e56414c49445f474c4f42414c53544154455f4f50434f44450000000000006044820152606401610180565b6102dd8261099c565b6080909801979097525050505050505050565b60006103076103028760200151610cc3565b610ce8565b63ffffffff16905060006103216103028860200151610cc3565b63ffffffff1690508560200151600001516001600160401b031681602061034891906123e8565b118061035d575061035a602082612411565b15155b15610384578660025b9081600281111561037957610379611c13565b81525050505061058b565b6000610391602083612425565b90506000806103ac6040518060200160405280606081525090565b60208a01516103be90858a8a87610d79565b9094509092509050606060008989868181106103dc576103dc612439565b919091013560f81c91508590506103f28161244f565b9550508060ff166000036104ce5736600061040f8b88818f6123a8565b91509150858282604051610424929190612468565b6040518091039020146104685760405162461bcd60e51b815260206004820152600c60248201526b4241445f505245494d41474560a01b6044820152606401610180565b60006104758b60206123e8565b9050818111156104825750805b61048e818c84866123a8565b8080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525092975061050f95505050505050565b60405162461bcd60e51b81526020600482015260166024820152752aa725a727aba72fa82922a4a6a0a3a2afa82927a7a360511b6044820152606401610180565b60005b82518110156105535761053f858285848151811061053257610532612439565b016020015160f81c610e13565b94508061054b8161244f565b915050610512565b5061055f838786610e99565b60208d01516040015281516105829061057790610f18565b60208f015190610f4b565b50505050505050505b505050505050565b60006105a56103028760200151610cc3565b63ffffffff16905060006105bf6103028860200151610cc3565b63ffffffff16905060006105de6105d98960200151610cc3565b610f5b565b6001600160401b0316905060208601351580156105fc575088358110155b15610624578760025b9081600281111561061857610618611c13565b8152505050505061058b565b602080880151516001600160401b0316906106409084906123e8565b11806106555750610652602083612411565b15155b1561066257876002610605565b600061066f602084612425565b905060008061068a6040518060200160405280606081525090565b60208b015161069c90858b8b87610d79565b90945090925090508888848181106106b6576106b6612439565b909101356001600160f81b0319161590506107095760405162461bcd60e51b81526020600482015260136024820152722aa725a727aba72fa4a72127ac2fa82927a7a360691b6044820152606401610180565b826107138161244f565b9350611aab9050600060208c013561072f57610fec915061076e565b60018c6020013503610745576112be915061076e565b8d60025b9081600281111561075c5761075c611c13565b8152505050505050505050505061058b565b61078e8f888d8d89908092610785939291906123a8565b8663ffffffff16565b90508061079d578d6002610749565b5050828810156107e35760405162461bcd60e51b81526020600482015260116024820152702120a22fa6a2a9a9a0a3a2afa82927a7a360791b6044820152606401610180565b60006107ef848a612478565b905060005b60208163ffffffff1610801561081857508161081663ffffffff83168b6123e8565b105b156108715761085d8463ffffffff83168d8d826108358f8c6123e8565b61083f91906123e8565b81811061084e5761084e612439565b919091013560f81c9050610e13565b9350806108698161248b565b9150506107f4565b61087c838786610e99565b60208e0151604001526108a461089182610f18565b8f60200151610f4b90919063ffffffff16565b505050505050505050505050505050565b50506001909252505050565b6108c9611ab5565b816108d2611ada565b6108da611ada565b60005b600260ff82161015610925576108f4888886611542565b848360ff166002811061090957610909612439565b602002019190915293508061091d816124ae565b9150506108dd565b5060005b600260ff8216101561097f5761094088888661155e565b838360ff166002811061095557610955612439565b6001600160401b039093166020939093020191909152935080610977816124ae565b915050610929565b506040805180820190915291825260208201529590945092505050565b8051805160209182015192820151805190830151604080516c23b637b130b61039ba30ba329d60991b81870152602d810194909452604d8401959095526001600160c01b031960c092831b8116606d850152911b1660758201528251808203605d018152607d909101909252815191012090565b6000610a226103028860200151610cc3565b63ffffffff1690506000610a3c6103028960200151610cc3565b9050600263ffffffff821610610a5457876002610366565b602080880151516001600160401b031690610a709084906123e8565b1180610a855750610a82602083612411565b15155b15610a9257876002610366565b6000610a9f602084612425565b9050600080610aba6040518060200160405280606081525090565b60208b0151610acc90858a8a87610d79565b9094509092509050618010610ae460208b018b612384565b61ffff1603610b2857610b1a848b600001518763ffffffff1660028110610b0d57610b0d612439565b6020020151839190610e99565b60208c015160400152610ba8565b618011610b3860208b018b612384565b61ffff1603610b66578951829063ffffffff871660028110610b5c57610b5c612439565b6020020152610ba8565b60405162461bcd60e51b81526020600482015260176024820152764241445f474c4f42414c5f53544154455f4f50434f444560481b6044820152606401610180565b505050505050505050505050565b6000610bc86103028460200151610cc3565b9050600263ffffffff821610610be057505060029052565b610c48610c3d83602001518363ffffffff1660028110610c0257610c02612439565b602002015160408051808201909152600080825260208201525060408051808201909152600181526001600160401b03909116602082015290565b602085015190610f4b565b505050565b6000610c5f6105d98460200151610cc3565b90506000610c736103028560200151610cc3565b9050600263ffffffff821610610c8d575050600290915250565b8183602001518263ffffffff1660028110610caa57610caa612439565b6001600160401b03909216602092909202015250505050565b60408051808201909152600080825260208201528151610ce2906115c5565b92915050565b60208101516000908183516006811115610d0457610d04611c13565b14610d3b5760405162461bcd60e51b81526020600482015260076024820152662727aa2fa4999960c91b6044820152606401610180565b6401000000008110610ce25760405162461bcd60e51b81526020600482015260076024820152662120a22fa4999960c91b6044820152606401610180565b600080610d926040518060200160405280606081525090565b839150610da0868684611542565b9093509150610db08686846116d5565b925090506000610dc1828986610e99565b905088604001518114610e075760405162461bcd60e51b815260206004820152600e60248201526d15d493d391d7d3515357d493d3d560921b6044820152606401610180565b50955095509592505050565b600060208310610e5d5760405162461bcd60e51b81526020600482015260156024820152740848288bea68aa8be988a828cbe84b2a88abe9288b605b1b6044820152606401610180565b600083610e6c60016020612478565b610e769190612478565b610e819060086124cd565b60ff848116821b911b198616179150505b9392505050565b6040516b26b2b6b7b93c903632b0b31d60a11b6020820152602c81018290526000908190604c01604051602081830303815290604052805190602001209050610f0f8585836040518060400160405280601381526020017226b2b6b7b93c9036b2b935b632903a3932b29d60691b8152506117af565b95945050505050565b604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b8151610f579082611884565b5050565b6020810151600090600183516006811115610f7857610f78611c13565b14610faf5760405162461bcd60e51b81526020600482015260076024820152661393d517d24d8d60ca1b6044820152606401610180565b600160401b8110610ce25760405162461bcd60e51b815260206004820152600760248201526610905117d24d8d60ca1b6044820152606401610180565b600060288210156110345760405162461bcd60e51b81526020600482015260126024820152712120a22fa9a2a8a4a72127ac2fa82927a7a360711b6044820152606401610180565b60006110428484602061155e565b508091505060008484604051611059929190612468565b60405190819003902090506000806001600160401b0388161561110a5761108660408a0160208b016124e4565b6001600160a01b03166316bf557961109f60018b61250d565b6040516001600160e01b031960e084901b1681526001600160401b039091166004820152602401602060405180830381865afa1580156110e3573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906111079190612534565b91505b6001600160401b038416156111ad5761112960408a0160208b016124e4565b6001600160a01b031663d5719dc261114260018761250d565b6040516001600160e01b031960e084901b1681526001600160401b039091166004820152602401602060405180830381865afa158015611186573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906111aa9190612534565b90505b6040805160208101849052908101849052606081018290526000906080016040516020818303038152906040528051906020012090508960200160208101906111f691906124e4565b6040516316bf557960e01b81526001600160401b038b1660048201526001600160a01b0391909116906316bf557990602401602060405180830381865afa158015611245573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906112699190612534565b81146112ae5760405162461bcd60e51b81526020600482015260146024820152734241445f534551494e424f585f4d45535341474560601b6044820152606401610180565b5060019998505050505050505050565b600060718210156113055760405162461bcd60e51b81526020600482015260116024820152702120a22fa222a620aca2a22fa82927a7a360791b6044820152606401610180565b60006001600160401b038516156113aa5761132660408701602088016124e4565b6001600160a01b031663d5719dc261133f60018861250d565b6040516001600160e01b031960e084901b1681526001600160401b039091166004820152602401602060405180830381865afa158015611383573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906113a79190612534565b90505b60006113b984607181886123a8565b6040516113c7929190612468565b604051809103902090506000858560008181106113e6576113e6612439565b9050013560f81c60f81b9050600061140087876001611977565b50905060008282611415607160218b8d6123a8565b8760405160200161142a95949392919061254d565b60408051601f1981840301815282825280516020918201208382018990528383018190528251808503840181526060909401909252825192019190912090915061147a60408c0160208d016124e4565b604051636ab8cee160e11b81526001600160401b038c1660048201526001600160a01b03919091169063d5719dc290602401602060405180830381865afa1580156114c9573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114ed9190612534565b81146115315760405162461bcd60e51b81526020600482015260136024820152724241445f44454c415945445f4d45535341474560681b6044820152606401610180565b5060019a9950505050505050505050565b60008181611551868684611977565b9097909650945050505050565b600081815b60088110156115bc576008836001600160401b0316901b925085858381811061158e5761158e612439565b919091013560f81c939093179250816115a68161244f565b92505080806115b49061244f565b915050611563565b50935093915050565b6040805180820190915260008082526020820152815180516115e990600190612478565b815181106115f9576115f9612439565b60200260200101519050600060018360000151516116179190612478565b6001600160401b0381111561162e5761162e611e6a565b60405190808252806020026020018201604052801561167357816020015b604080518082019091526000808252602082015281526020019060019003908161164c5790505b50905060005b81518110156116ce57835180518290811061169657611696612439565b60200260200101518282815181106116b0576116b0612439565b602002602001018190525080806116c69061244f565b915050611679565b5090915290565b6040805160208101909152606081528160006116f28686846119cc565b92509050600060ff82166001600160401b0381111561171357611713611e6a565b60405190808252806020026020018201604052801561173c578160200160208202803683370190505b50905060005b8260ff168160ff1610156117935761175b888886611542565b838360ff168151811061177057611770612439565b60200260200101819650828152505050808061178b906124ae565b915050611742565b5060405180602001604052808281525093505050935093915050565b8160005b85515181101561187b5784600116600003611817578282876000015183815181106117e0576117e0612439565b60200260200101516040516020016117fa9392919061258c565b604051602081830303815290604052805190602001209150611862565b828660000151828151811061182e5761182e612439565b6020026020010151836040516020016118499392919061258c565b6040516020818303038152906040528051906020012091505b60019490941c93806118738161244f565b9150506117b3565b50949350505050565b8151516000906118959060016123e8565b6001600160401b038111156118ac576118ac611e6a565b6040519080825280602002602001820160405280156118f157816020015b60408051808201909152600080825260208201528152602001906001900390816118ca5790505b50905060005b83515181101561194d57835180518290811061191557611915612439565b602002602001015182828151811061192f5761192f612439565b602002602001018190525080806119459061244f565b9150506118f7565b5081818460000151518151811061196657611966612439565b602090810291909101015290915250565b600081815b60208110156115bc57600883901b925085858381811061199e5761199e612439565b919091013560f81c939093179250816119b68161244f565b92505080806119c49061244f565b91505061197c565b6000818484828181106119e1576119e1612439565b919091013560f81c92508190506119f78161244f565b915050935093915050565b6040805161012081019091528060008152602001611a3760408051606080820183529181019182529081526000602082015290565b8152602001611a5d60408051606080820183529181019182529081526000602082015290565b8152602001611a82604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b611ab36125c3565b565b6040518060400160405280611ac8611ada565b8152602001611ad5611ada565b905290565b60405180604001604052806002906020820280368337509192915050565b600060408284031215611b0a57600080fd5b50919050565b60008083601f840112611b2257600080fd5b5081356001600160401b03811115611b3957600080fd5b602083019150836020828501011115611b5157600080fd5b9250929050565b6000806000806000808688036101c0811215611b7357600080fd5b6060811215611b8157600080fd5b87965060608801356001600160401b0380821115611b9e57600080fd5b90890190610120828c031215611bb357600080fd5b81975060e0607f1984011215611bc857600080fd5b60808a019650611bdc8b6101608c01611af8565b95506101a08a0135925080831115611bf357600080fd5b5050611c0189828a01611b10565b979a9699509497509295939492505050565b634e487b7160e01b600052602160045260246000fd5b60038110611c3957611c39611c13565b9052565b805160078110611c4f57611c4f611c13565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b80841015611cac57611c98828651611c3d565b938201936001939093019290850190611c85565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b82811015611d2b578451611cf7858251611c3d565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101611ce2565b509687015197909601969096525093949350505050565b6000610100808352611d578184018651611c29565b602085015161012084810152611d71610220850182611c5c565b9050604086015160ff198086840301610140870152611d908383611c5c565b925060608801519150808684030161016087015250611daf8282611cc0565b915050608086015161018085015260a0860151611dd56101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e086015250908501516102008401529050610e9260208301848051825260208101516001600160401b0380825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b604080519081016001600160401b0381118282101715611ea257611ea2611e6a565b60405290565b604051602081016001600160401b0381118282101715611ea257611ea2611e6a565b604051608081016001600160401b0381118282101715611ea257611ea2611e6a565b60405161012081016001600160401b0381118282101715611ea257611ea2611e6a565b60405160a081016001600160401b0381118282101715611ea257611ea2611e6a565b604051606081016001600160401b0381118282101715611ea257611ea2611e6a565b604051601f8201601f191681016001600160401b0381118282101715611f7b57611f7b611e6a565b604052919050565b803560038110611f9257600080fd5b919050565b60006001600160401b03821115611fb057611fb0611e6a565b5060051b60200190565b600060408284031215611fcc57600080fd5b611fd4611e80565b9050813560078110611fe557600080fd5b808252506020820135602082015292915050565b6000604080838503121561200c57600080fd5b612014611e80565b915082356001600160401b038082111561202d57600080fd5b8185019150602080838803121561204357600080fd5b61204b611ea8565b83358381111561205a57600080fd5b80850194505087601f85011261206f57600080fd5b8335925061208461207f84611f97565b611f53565b83815260069390931b840182019282810190898511156120a357600080fd5b948301945b848610156120c9576120ba8a87611fba565b825294860194908301906120a8565b8252508552948501359484019490945250909392505050565b803563ffffffff81168114611f9257600080fd5b6000604080838503121561210957600080fd5b612111611e80565b915082356001600160401b0381111561212957600080fd5b8301601f8101851361213a57600080fd5b8035602061214a61207f83611f97565b82815260a0928302840182019282820191908985111561216957600080fd5b948301945b848610156121d25780868b0312156121865760008081fd5b61218e611eca565b6121988b88611fba565b8152878701358582015260606121af8189016120e2565b898301526121bf608089016120e2565b908201528352948501949183019161216e565b50808752505080860135818601525050505092915050565b600061012082360312156121fd57600080fd5b612205611eec565b61220e83611f83565b815260208301356001600160401b038082111561222a57600080fd5b61223636838701611ff9565b6020840152604085013591508082111561224f57600080fd5b61225b36838701611ff9565b6040840152606085013591508082111561227457600080fd5b50612281368286016120f6565b6060830152506080830135608082015261229d60a084016120e2565b60a08201526122ae60c084016120e2565b60c08201526122bf60e084016120e2565b60e082015261010092830135928101929092525090565b80356001600160401b0381168114611f9257600080fd5b600081830360e081121561230057600080fd5b612308611f0f565b833581526060601f198301121561231e57600080fd5b612326611f31565b9150612334602085016122d6565b8252612342604085016122d6565b6020830152606084013560408301528160208201526080840135604082015260a0840135606082015261237760c085016120e2565b6080820152949350505050565b60006020828403121561239657600080fd5b813561ffff81168114610e9257600080fd5b600080858511156123b857600080fd5b838611156123c557600080fd5b5050820193919092039150565b634e487b7160e01b600052601160045260246000fd5b80820180821115610ce257610ce26123d2565b634e487b7160e01b600052601260045260246000fd5b600082612420576124206123fb565b500690565b600082612434576124346123fb565b500490565b634e487b7160e01b600052603260045260246000fd5b600060018201612461576124616123d2565b5060010190565b8183823760009101908152919050565b81810381811115610ce257610ce26123d2565b600063ffffffff8083168181036124a4576124a46123d2565b6001019392505050565b600060ff821660ff81036124c4576124c46123d2565b60010192915050565b8082028115828204841417610ce257610ce26123d2565b6000602082840312156124f657600080fd5b81356001600160a01b0381168114610e9257600080fd5b6001600160401b0382811682821603908082111561252d5761252d6123d2565b5092915050565b60006020828403121561254657600080fd5b5051919050565b6001600160f81b031986168152606085901b6bffffffffffffffffffffffff191660018201528284601583013760159201918201526035019392505050565b6000845160005b818110156125ad5760208188018101518583015201612593565b5091909101928352506020820152604001919050565b634e487b7160e01b600052605160045260246000fdfea2646970667358221220faf2869ec7327a1aea6cb377010cabf441ec49a09ce65fbf4e8c2b57f4e7e14064736f6c63430008110033",
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
	parsed, err := abi.JSON(strings.NewReader(OneStepProverHostIoABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
	Bin: "0x608060405234801561001057600080fd5b5061234c806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c806397cc779a14610030575b600080fd5b61004361003e3660046118e2565b61005a565b604051610051929190611acc565b60405180910390f35b6100626117cf565b6040805160a0810182526000808252825160608082018552828252602080830184905282860184905284019190915292820181905291810182905260808101919091526100ae87611f6f565b91506100bf36879003870187612072565b905060006100d06020870187612109565b905061187861ffff8216604514806100ec575061ffff82166050145b156100fa57506103096102eb565b604661ffff831610801590610122575061011660096046612143565b61ffff168261ffff1611155b15610130575061041f6102eb565b606761ffff831610801590610158575061014c60026067612143565b61ffff168261ffff1611155b1561016657506105026102eb565b606a61ffff8316108015906101805750607861ffff831611155b1561018e575061056a6102eb565b605161ffff8316108015906101b657506101aa60096051612143565b61ffff168261ffff1611155b156101c457506107536102eb565b607961ffff8316108015906101ec57506101e060026079612143565b61ffff168261ffff1611155b156101fa57506107b86102eb565b607c61ffff8316108015906102145750608a61ffff831611155b15610222575061080b6102eb565b60a61961ffff83160161023857506109c26102eb565b61ffff821660ac148061024f575061ffff821660ad145b1561025d57506109e36102eb565b60c061ffff831610801590610277575060c461ffff831611155b156102855750610a366102eb565b60bc61ffff83161080159061029f575060bf61ffff831611155b156102ad5750610c456102eb565b60405162461bcd60e51b815260206004820152600e60248201526d494e56414c49445f4f50434f444560901b60448201526064015b60405180910390fd5b6102fc84848989898663ffffffff16565b5050965096945050505050565b60006103188660200151610dcc565b905060456103296020860186612109565b61ffff1603610369576000815160068111156103475761034761199d565b146103645760405162461bcd60e51b81526004016102e290612165565b6103e5565b60506103786020860186612109565b61ffff16036103b3576001815160068111156103965761039661199d565b146103645760405162461bcd60e51b81526004016102e290612186565b60405162461bcd60e51b81526020600482015260076024820152662120a22fa2a8ad60c91b60448201526064016102e2565b600081602001516000036103fb575060016103ff565b5060005b61041661040b82610df1565b602089015190610e24565b50505050505050565b60006104366104318760200151610dcc565b610e34565b9050600061044a6104318860200151610dcc565b90506000604661045d6020880188612109565b61046791906121a7565b905060008061ffff831660021480610483575061ffff83166004145b80610492575061ffff83166006145b806104a1575061ffff83166008145b156104c1576104af84610eab565b91506104ba85610eab565b90506104cf565b505063ffffffff8083169084165b60006104dc838386610ed7565b90506104f56104ea82611080565b60208d015190610e24565b5050505050505050505050565b60006105146104318760200151610dcc565b9050600060676105276020870187612109565b61053191906121a7565b905060006105478363ffffffff168360206110b3565b905061056061055582610df1565b60208a015190610e24565b5050505050505050565b600061057c6104318760200151610dcc565b905060006105906104318860200151610dcc565b9050600080606a6105a46020890189612109565b6105ae91906121a7565b90508061ffff1660030361062b5763ffffffff841615806105e557508260030b637fffffff191480156105e557508360030b600019145b1561060e578860025b908160028111156106015761060161199d565b815250505050505061074c565b8360030b8360030b81610623576106236121c2565b059150610730565b8061ffff1660050361066a578363ffffffff1660000361064d578860026105ee565b8360030b8360030b81610662576106626121c2565b079150610730565b8061ffff16600a036106895763ffffffff8316601f85161b9150610730565b8061ffff16600c036106a85763ffffffff8316601f85161c9150610730565b8061ffff16600b036106c557600383900b601f85161d9150610730565b8061ffff16600d036106e2576106db8385611277565b9150610730565b8061ffff16600e036106f8576106db83856112b9565b6000806107128563ffffffff168763ffffffff16856112fb565b91509150801561072c575050600289525061074c92505050565b5091505b61074761073c83610df1565b60208b015190610e24565b505050505b5050505050565b600061076a6107658760200151610dcc565b611483565b9050600061077e6107658860200151610dcc565b9050600060516107916020880188612109565b61079b91906121a7565b905060006107aa838584610ed7565b905061074761073c82611080565b60006107ca6107658760200151610dcc565b9050600060796107dd6020870187612109565b6107e791906121a7565b905060006107f7838360406110b3565b63ffffffff169050610560610555826114fa565b600061081d6107658760200151610dcc565b905060006108316107658860200151610dcc565b9050600080607c6108456020890189612109565b61084f91906121a7565b90508061ffff166003036108b7576001600160401b038416158061088d57508260070b677fffffffffffffff1914801561088d57508360070b600019145b1561089a578860026105ee565b8360070b8360070b816108af576108af6121c2565b0591506109b6565b8061ffff166005036108f957836001600160401b03166000036108dc578860026105ee565b8360070b8360070b816108f1576108f16121c2565b0791506109b6565b8061ffff16600a0361091b576001600160401b038316603f85161b91506109b6565b8061ffff16600c0361093d576001600160401b038316603f85161c91506109b6565b8061ffff16600b0361095a57600783900b603f85161d91506109b6565b8061ffff16600d03610977576109708385611530565b91506109b6565b8061ffff16600e0361098d57610970838561157e565b600061099a8486846112fb565b909350905080156109b4575050600288525061074c915050565b505b61074761073c836114fa565b60006109d46107658760200151610dcc565b90508061041661040b82610df1565b60006109f56104318760200151610dcc565b9050600060ac610a086020870187612109565b61ffff1603610a2157610a1a82610eab565b9050610a2a565b5063ffffffff81165b61041661040b826114fa565b60008060c0610a486020870187612109565b61ffff1603610a5d5750600090506008610b30565b60c1610a6c6020870187612109565b61ffff1603610a815750600090506010610b30565b60c2610a906020870187612109565b61ffff1603610aa55750600190506008610b30565b60c3610ab46020870187612109565b61ffff1603610ac95750600190506010610b30565b60c4610ad86020870187612109565b61ffff1603610aed5750600190506020610b30565b60405162461bcd60e51b8152602060048201526018602482015277494e56414c49445f455854454e445f53414d455f5459504560401b60448201526064016102e2565b600080836006811115610b4557610b4561199d565b03610b55575063ffffffff610b5f565b506001600160401b035b6000610b6e8960200151610dcc565b9050836006811115610b8257610b8261199d565b81516006811115610b9557610b9561199d565b14610bde5760405162461bcd60e51b81526020600482015260196024820152784241445f455854454e445f53414d455f545950455f5459504560381b60448201526064016102e2565b6000610bf1600160ff861681901b6121d8565b602083018051821690529050610c086001856121eb565b60ff166001901b826020015116600014610c2a57602082018051821985161790525b60208a0151610c399083610e24565b50505050505050505050565b60008060bc610c576020870187612109565b61ffff1603610c6c5750600090506002610d16565b60bd610c7b6020870187612109565b61ffff1603610c905750600190506003610d16565b60be610c9f6020870187612109565b61ffff1603610cb45750600290506000610d16565b60bf610cc36020870187612109565b61ffff1603610cd85750600390506001610d16565b60405162461bcd60e51b81526020600482015260136024820152721253959053125117d491525395115494149155606a1b60448201526064016102e2565b6000610d258860200151610dcc565b9050816006811115610d3957610d3961199d565b81516006811115610d4c57610d4c61199d565b14610d945760405162461bcd60e51b8152602060048201526018602482015277494e56414c49445f5245494e544552505245545f5459504560401b60448201526064016102e2565b80836006811115610da757610da761199d565b90816006811115610dba57610dba61199d565b90525060208801516105609082610e24565b60408051808201909152600080825260208201528151610deb906115cc565b92915050565b604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b8151610e3090826116dc565b5050565b60208101516000908183516006811115610e5057610e5061199d565b14610e6d5760405162461bcd60e51b81526004016102e290612165565b6401000000008110610deb5760405162461bcd60e51b81526020600482015260076024820152662120a22fa4999960c91b60448201526064016102e2565b60006380000000821615610ecd575063ffffffff1667ffffffff000000001790565b5063ffffffff1690565b600061ffff8216610efe57826001600160401b0316846001600160401b0316149050611079565b60001961ffff831601610f2857826001600160401b0316846001600160401b031614159050611079565b60011961ffff831601610f45578260070b8460070b129050611079565b60021961ffff831601610f6e57826001600160401b0316846001600160401b0316109050611079565b60031961ffff831601610f8b578260070b8460070b139050611079565b60041961ffff831601610fb457826001600160401b0316846001600160401b0316119050611079565b60051961ffff831601610fd2578260070b8460070b13159050611079565b60061961ffff831601610ffc57826001600160401b0316846001600160401b031611159050611079565b60071961ffff83160161101a578260070b8460070b12159050611079565b60081961ffff83160161104457826001600160401b0316846001600160401b031610159050611079565b60405162461bcd60e51b815260206004820152600a6024820152690424144204952454c4f560b41b60448201526064016102e2565b9392505050565b604080518082019091526000808252602082015281156110a457610deb6001610df1565b610deb6000610df1565b919050565b60008161ffff16602014806110cc57508161ffff166040145b6111135760405162461bcd60e51b8152602060048201526018602482015277057524f4e4720555345204f462067656e65726963556e4f760441b60448201526064016102e2565b61ffff83166111845761ffff82165b60008163ffffffff16118015611157575061113e600182612204565b63ffffffff166001901b856001600160401b0316166000145b1561116e57611167600182612204565b9050611122565b61117c8161ffff8516612204565b915050611079565b60001961ffff8416016111dd5760005b8261ffff168163ffffffff161080156111bf5750600163ffffffff82161b85166001600160401b0316155b156111d6576111cf600182612221565b9050611194565b9050611079565b60011961ffff841601611243576000805b8361ffff168263ffffffff16101561123a57600163ffffffff83161b86166001600160401b03161561122857611225600182612221565b90505b816112328161223e565b9250506111ee565b91506110799050565b60405162461bcd60e51b815260206004820152600960248201526804241442049556e4f760bc1b60448201526064016102e2565b6000611284602083612261565b9150611291826020612204565b63ffffffff168363ffffffff16901c8263ffffffff168463ffffffff16901b17905092915050565b60006112c6602083612261565b91506112d3826020612204565b63ffffffff168363ffffffff16901b8263ffffffff168463ffffffff16901c17905092915050565b6000808261ffff16600003611316575050828201600061147b565b8261ffff1660010361132e575050818303600061147b565b8261ffff16600203611346575050828202600061147b565b8261ffff1660040361139c57836001600160401b031660000361136f575060009050600161147b565b836001600160401b0316856001600160401b031681611390576113906121c2565b0460009150915061147b565b8261ffff166006036113f257836001600160401b03166000036113c5575060009050600161147b565b836001600160401b0316856001600160401b0316816113e6576113e66121c2565b0660009150915061147b565b8261ffff1660070361140a575050828216600061147b565b8261ffff16600803611422575050828217600061147b565b8261ffff1660090361143a575050828218600061147b565b60405162461bcd60e51b81526020600482015260166024820152750494e56414c49445f47454e455249435f42494e5f4f560541b60448201526064016102e2565b935093915050565b60208101516000906001835160068111156114a0576114a061199d565b146114bd5760405162461bcd60e51b81526004016102e290612186565b600160401b8110610deb5760405162461bcd60e51b815260206004820152600760248201526610905117d24d8d60ca1b60448201526064016102e2565b60408051808201909152600080825260208201525060408051808201909152600181526001600160401b03909116602082015290565b600061153d604083612284565b915061154a82604061229e565b6001600160401b0316836001600160401b0316901c826001600160401b0316846001600160401b0316901b17905092915050565b600061158b604083612284565b915061159882604061229e565b6001600160401b0316836001600160401b0316901b826001600160401b0316846001600160401b0316901c17905092915050565b6040805180820190915260008082526020820152815180516115f0906001906121d8565b81518110611600576116006122be565b602002602001015190506000600183600001515161161e91906121d8565b6001600160401b0381111561163557611635611bf4565b60405190808252806020026020018201604052801561167a57816020015b60408051808201909152600080825260208201528152602001906001900390816116535790505b50905060005b81518110156116d557835180518290811061169d5761169d6122be565b60200260200101518282815181106116b7576116b76122be565b602002602001018190525080806116cd906122d4565b915050611680565b5090915290565b8151516000906116ed9060016122ed565b6001600160401b0381111561170457611704611bf4565b60405190808252806020026020018201604052801561174957816020015b60408051808201909152600080825260208201528152602001906001900390816117225790505b50905060005b8351518110156117a557835180518290811061176d5761176d6122be565b6020026020010151828281518110611787576117876122be565b6020026020010181905250808061179d906122d4565b91505061174f565b508181846000015151815181106117be576117be6122be565b602090810291909101015290915250565b604080516101208101909152806000815260200161180460408051606080820183529181019182529081526000602082015290565b815260200161182a60408051606080820183529181019182529081526000602082015290565b815260200161184f604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b611880612300565b565b60006040828403121561189457600080fd5b50919050565b60008083601f8401126118ac57600080fd5b5081356001600160401b038111156118c357600080fd5b6020830191508360208285010111156118db57600080fd5b9250929050565b6000806000806000808688036101c08112156118fd57600080fd5b606081121561190b57600080fd5b87965060608801356001600160401b038082111561192857600080fd5b90890190610120828c03121561193d57600080fd5b81975060e0607f198401121561195257600080fd5b60808a0196506119668b6101608c01611882565b95506101a08a013592508083111561197d57600080fd5b505061198b89828a0161189a565b979a9699509497509295939492505050565b634e487b7160e01b600052602160045260246000fd5b600381106119c3576119c361199d565b9052565b8051600781106119d9576119d961199d565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b80841015611a3657611a228286516119c7565b938201936001939093019290850190611a0f565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b82811015611ab5578451611a818582516119c7565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101611a6c565b509687015197909601969096525093949350505050565b6000610100808352611ae181840186516119b3565b602085015161012084810152611afb6102208501826119e6565b9050604086015160ff198086840301610140870152611b1a83836119e6565b925060608801519150808684030161016087015250611b398282611a4a565b915050608086015161018085015260a0860151611b5f6101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e08601525090850151610200840152905061107960208301848051825260208101516001600160401b0380825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b604080519081016001600160401b0381118282101715611c2c57611c2c611bf4565b60405290565b604051602081016001600160401b0381118282101715611c2c57611c2c611bf4565b604051608081016001600160401b0381118282101715611c2c57611c2c611bf4565b60405161012081016001600160401b0381118282101715611c2c57611c2c611bf4565b60405160a081016001600160401b0381118282101715611c2c57611c2c611bf4565b604051606081016001600160401b0381118282101715611c2c57611c2c611bf4565b604051601f8201601f191681016001600160401b0381118282101715611d0557611d05611bf4565b604052919050565b8035600381106110ae57600080fd5b60006001600160401b03821115611d3557611d35611bf4565b5060051b60200190565b600060408284031215611d5157600080fd5b611d59611c0a565b9050813560078110611d6a57600080fd5b808252506020820135602082015292915050565b60006040808385031215611d9157600080fd5b611d99611c0a565b915082356001600160401b0380821115611db257600080fd5b81850191506020808388031215611dc857600080fd5b611dd0611c32565b833583811115611ddf57600080fd5b80850194505087601f850112611df457600080fd5b83359250611e09611e0484611d1c565b611cdd565b83815260069390931b84018201928281019089851115611e2857600080fd5b948301945b84861015611e4e57611e3f8a87611d3f565b82529486019490830190611e2d565b8252508552948501359484019490945250909392505050565b803563ffffffff811681146110ae57600080fd5b60006040808385031215611e8e57600080fd5b611e96611c0a565b915082356001600160401b03811115611eae57600080fd5b8301601f81018513611ebf57600080fd5b80356020611ecf611e0483611d1c565b82815260a09283028401820192828201919089851115611eee57600080fd5b948301945b84861015611f575780868b031215611f0b5760008081fd5b611f13611c54565b611f1d8b88611d3f565b815287870135858201526060611f34818901611e67565b89830152611f4460808901611e67565b9082015283529485019491830191611ef3565b50808752505080860135818601525050505092915050565b60006101208236031215611f8257600080fd5b611f8a611c76565b611f9383611d0d565b815260208301356001600160401b0380821115611faf57600080fd5b611fbb36838701611d7e565b60208401526040850135915080821115611fd457600080fd5b611fe036838701611d7e565b60408401526060850135915080821115611ff957600080fd5b5061200636828601611e7b565b6060830152506080830135608082015261202260a08401611e67565b60a082015261203360c08401611e67565b60c082015261204460e08401611e67565b60e082015261010092830135928101929092525090565b80356001600160401b03811681146110ae57600080fd5b600081830360e081121561208557600080fd5b61208d611c99565b833581526060601f19830112156120a357600080fd5b6120ab611cbb565b91506120b96020850161205b565b82526120c76040850161205b565b6020830152606084013560408301528160208201526080840135604082015260a084013560608201526120fc60c08501611e67565b6080820152949350505050565b60006020828403121561211b57600080fd5b813561ffff8116811461107957600080fd5b634e487b7160e01b600052601160045260246000fd5b61ffff81811683821601908082111561215e5761215e61212d565b5092915050565b6020808252600790820152662727aa2fa4999960c91b604082015260600190565b6020808252600790820152661393d517d24d8d60ca1b604082015260600190565b61ffff82811682821603908082111561215e5761215e61212d565b634e487b7160e01b600052601260045260246000fd5b81810381811115610deb57610deb61212d565b60ff8281168282160390811115610deb57610deb61212d565b63ffffffff82811682821603908082111561215e5761215e61212d565b63ffffffff81811683821601908082111561215e5761215e61212d565b600063ffffffff8083168181036122575761225761212d565b6001019392505050565b600063ffffffff80841680612278576122786121c2565b92169190910692915050565b60006001600160401b0380841680612278576122786121c2565b6001600160401b0382811682821603908082111561215e5761215e61212d565b634e487b7160e01b600052603260045260246000fd5b6000600182016122e6576122e661212d565b5060010190565b80820180821115610deb57610deb61212d565b634e487b7160e01b600052605160045260246000fdfea2646970667358221220bcf30fce266578272a46eaaeeda2c3cdc8526e6f865baa89d57574ea89fe012464736f6c63430008110033",
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
	parsed, err := abi.JSON(strings.NewReader(OneStepProverMathABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
	Bin: "0x608060405234801561001057600080fd5b50611deb806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c806397cc779a14610030575b600080fd5b61004361003e3660046113c1565b61005a565b6040516100519291906115ab565b60405180910390f35b6100626112ae565b6040805160a0810182526000808252825160608082018552828252602080830184905282860184905284019190915292820181905291810182905260808101919091526100ae87611a53565b91506100bf36879003870187611b56565b905060006100d06020870187611bed565b9050611357602861ffff8316108015906100ef5750603561ffff831611155b156100fd57506101b4610196565b603661ffff8316108015906101175750603e61ffff831611155b1561012557506106b1610196565b603e1961ffff83160161013b5750610a52610196565b603f1961ffff8316016101515750610a8a610196565b60405162461bcd60e51b8152602060048201526015602482015274494e56414c49445f4d454d4f52595f4f50434f444560581b60448201526064015b60405180910390fd5b6101a784848989898663ffffffff16565b5050965096945050505050565b6000808060286101c76020880188611bed565b61ffff16036101df5750600091506004905081610427565b60296101ee6020880188611bed565b61ffff1603610207575060019150600890506000610427565b602a6102166020880188611bed565b61ffff160361022f575060029150600490506000610427565b602b61023e6020880188611bed565b61ffff1603610257575060039150600890506000610427565b602c6102666020880188611bed565b61ffff160361027e5750600091506001905080610427565b602d61028d6020880188611bed565b61ffff16036102a55750600091506001905081610427565b602e6102b46020880188611bed565b61ffff16036102cd575060009150600290506001610427565b602f6102dc6020880188611bed565b61ffff16036102f45750600091506002905081610427565b60306103036020880188611bed565b61ffff160361031a57506001915081905080610427565b60316103296020880188611bed565b61ffff16036103415750600191508190506000610427565b60326103506020880188611bed565b61ffff16036103685750600191506002905081610427565b60336103776020880188611bed565b61ffff1603610390575060019150600290506000610427565b603461039f6020880188611bed565b61ffff16036103b75750600191506004905081610427565b60356103c66020880188611bed565b61ffff16036103df575060019150600490506000610427565b60405162461bcd60e51b815260206004820152601a60248201527f494e56414c49445f4d454d4f52595f4c4f41445f4f50434f4445000000000000604482015260640161018d565b600061043e6104398a60200151610b39565b610b5e565b6104529063ffffffff166020890135611c27565b6020890151519091506001600160401b031661046e8483611c27565b111561048257505060028752506106aa9050565b60006000198180805b8781101561052757600061049f8288611c27565b905060006104ae602083611c50565b90508581146104d2576104c88f60200151828f8f8b610bef565b5097509095509350845b60006104df602084611c64565b90506104ec846008611c78565b6001600160401b03166104ff8783610c89565b60ff166001600160401b0316901b85179450505050808061051f90611c8f565b91505061048b565b5085156106665786600114801561054f5750600088600681111561054d5761054d61147c565b145b15610565578060000b63ffffffff169050610666565b866001148015610586575060018860068111156105845761058461147c565b145b156105935760000b610666565b8660021480156105b4575060008860068111156105b2576105b261147c565b145b156105ca578060010b63ffffffff169050610666565b8660021480156105eb575060018860068111156105e9576105e961147c565b145b156105f85760010b610666565b866004148015610619575060018860068111156106175761061761147c565b145b156106265760030b610666565b60405162461bcd60e51b815260206004820152601560248201527410905117d491505117d096551154d7d4d251d39151605a1b604482015260640161018d565b6106a160405180604001604052808a60068111156106865761068661147c565b81526001600160401b0384166020918201528f015190610d03565b50505050505050505b5050505050565b6000808060366106c46020880188611bed565b61ffff16036106d95750600491506000610840565b60376106e86020880188611bed565b61ffff16036106fd5750600891506001610840565b603861070c6020880188611bed565b61ffff16036107215750600491506002610840565b60396107306020880188611bed565b61ffff16036107455750600891506003610840565b603a6107546020880188611bed565b61ffff16036107695750600191506000610840565b603b6107786020880188611bed565b61ffff160361078d5750600291506000610840565b603c61079c6020880188611bed565b61ffff16036107b057506001915081610840565b603d6107bf6020880188611bed565b61ffff16036107d45750600291506001610840565b603e6107e36020880188611bed565b61ffff16036107f85750600491506001610840565b60405162461bcd60e51b815260206004820152601b60248201527f494e56414c49445f4d454d4f52595f53544f52455f4f50434f44450000000000604482015260640161018d565b600061084f8960200151610b39565b90508160068111156108635761086361147c565b815160068111156108765761087661147c565b146108b45760405162461bcd60e51b815260206004820152600e60248201526d4241445f53544f52455f5459504560901b604482015260640161018d565b806020015192506008846001600160401b031610156108ff5760016108da856008611ca8565b6001600160401b031660016001600160401b0316901b6108fa9190611cd3565b831692505b505060006109136104398960200151610b39565b6109279063ffffffff166020880135611c27565b90508660200151600001516001600160401b0316836001600160401b0316826109509190611c27565b111561096257505060028652506106aa565b604080516020810190915260608152600090600019906000805b876001600160401b0316811015610a2f5760006109998288611c27565b905060006109a8602083611c50565b90508581146109ed5760001986146109cf576109c5858786610d13565b60208f0151604001525b6109e08e60200151828e8e8b610bef565b9098509196509094509250845b60006109fa602084611c64565b9050610a0785828c610d94565b945060088a6001600160401b0316901c99505050508080610a2790611c8f565b91505061097c565b50610a3b828483610d13565b60208c015160400152505050505050505050505050565b602084015151600090610a69906201000090611cfa565b9050610a82610a7782610e19565b602088015190610d03565b505050505050565b602084015151600090610aa1906201000090611cfa565b90506000610ab56104398860200151610b39565b90506000610acc63ffffffff808416908516611c27565b90508660200151602001516001600160401b03168111610b2157610af36201000082611c78565b60208801516001600160401b039091169052610b1c610b1184610e19565b60208a015190610d03565b610b2f565b610b2f610b11600019610e19565b5050505050505050565b60408051808201909152600080825260208201528151610b5890610e4c565b92915050565b60208101516000908183516006811115610b7a57610b7a61147c565b14610bb15760405162461bcd60e51b81526020600482015260076024820152662727aa2fa4999960c91b604482015260640161018d565b6401000000008110610b585760405162461bcd60e51b81526020600482015260076024820152662120a22fa4999960c91b604482015260640161018d565b600080610c086040518060200160405280606081525090565b839150610c16868684610f5c565b9093509150610c26868684610f78565b925090506000610c37828986610d13565b905088604001518114610c7d5760405162461bcd60e51b815260206004820152600e60248201526d15d493d391d7d3515357d493d3d560921b604482015260640161018d565b50955095509592505050565b600060208210610cd45760405162461bcd60e51b81526020600482015260166024820152750848288bea0aa9898be988a828cbe84b2a88abe9288b60531b604482015260640161018d565b600082610ce360016020611d20565b610ced9190611d20565b610cf8906008611c78565b9390931c9392505050565b8151610d0f9082611052565b5050565b6040516b26b2b6b7b93c903632b0b31d60a11b6020820152602c81018290526000908190604c01604051602081830303815290604052805190602001209050610d898585836040518060400160405280601381526020017226b2b6b7b93c9036b2b935b632903a3932b29d60691b815250611145565b9150505b9392505050565b600060208310610dde5760405162461bcd60e51b81526020600482015260156024820152740848288bea68aa8be988a828cbe84b2a88abe9288b605b1b604482015260640161018d565b600083610ded60016020611d20565b610df79190611d20565b610e02906008611c78565b60ff848116821b911b198616179150509392505050565b604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b604080518082019091526000808252602082015281518051610e7090600190611d20565b81518110610e8057610e80611d33565b6020026020010151905060006001836000015151610e9e9190611d20565b6001600160401b03811115610eb557610eb56116d3565b604051908082528060200260200182016040528015610efa57816020015b6040805180820190915260008082526020820152815260200190600190039081610ed35790505b50905060005b8151811015610f55578351805182908110610f1d57610f1d611d33565b6020026020010151828281518110610f3757610f37611d33565b60200260200101819052508080610f4d90611c8f565b915050610f00565b5090915290565b60008181610f6b86868461121a565b9097909650945050505050565b604080516020810190915260608152816000610f95868684611278565b92509050600060ff82166001600160401b03811115610fb657610fb66116d3565b604051908082528060200260200182016040528015610fdf578160200160208202803683370190505b50905060005b8260ff168160ff16101561103657610ffe888886610f5c565b838360ff168151811061101357611013611d33565b60200260200101819650828152505050808061102e90611d49565b915050610fe5565b5060405180602001604052808281525093505050935093915050565b815151600090611063906001611c27565b6001600160401b0381111561107a5761107a6116d3565b6040519080825280602002602001820160405280156110bf57816020015b60408051808201909152600080825260208201528152602001906001900390816110985790505b50905060005b83515181101561111b5783518051829081106110e3576110e3611d33565b60200260200101518282815181106110fd576110fd611d33565b6020026020010181905250808061111390611c8f565b9150506110c5565b5081818460000151518151811061113457611134611d33565b602090810291909101015290915250565b8160005b85515181101561121157846001166000036111ad5782828760000151838151811061117657611176611d33565b602002602001015160405160200161119093929190611d68565b6040516020818303038152906040528051906020012091506111f8565b82866000015182815181106111c4576111c4611d33565b6020026020010151836040516020016111df93929190611d68565b6040516020818303038152906040528051906020012091505b60019490941c938061120981611c8f565b915050611149565b50949350505050565b600081815b602081101561126f57600883901b925085858381811061124157611241611d33565b919091013560f81c9390931792508161125981611c8f565b925050808061126790611c8f565b91505061121f565b50935093915050565b60008184848281811061128d5761128d611d33565b919091013560f81c92508190506112a381611c8f565b915050935093915050565b60408051610120810190915280600081526020016112e360408051606080820183529181019182529081526000602082015290565b815260200161130960408051606080820183529181019182529081526000602082015290565b815260200161132e604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b61135f611d9f565b565b60006040828403121561137357600080fd5b50919050565b60008083601f84011261138b57600080fd5b5081356001600160401b038111156113a257600080fd5b6020830191508360208285010111156113ba57600080fd5b9250929050565b6000806000806000808688036101c08112156113dc57600080fd5b60608112156113ea57600080fd5b87965060608801356001600160401b038082111561140757600080fd5b90890190610120828c03121561141c57600080fd5b81975060e0607f198401121561143157600080fd5b60808a0196506114458b6101608c01611361565b95506101a08a013592508083111561145c57600080fd5b505061146a89828a01611379565b979a9699509497509295939492505050565b634e487b7160e01b600052602160045260246000fd5b600381106114a2576114a261147c565b9052565b8051600781106114b8576114b861147c565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b80841015611515576115018286516114a6565b9382019360019390930192908501906114ee565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b828110156115945784516115608582516114a6565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a09093019260010161154b565b509687015197909601969096525093949350505050565b60006101008083526115c08184018651611492565b6020850151610120848101526115da6102208501826114c5565b9050604086015160ff1980868403016101408701526115f983836114c5565b9250606088015191508086840301610160870152506116188282611529565b915050608086015161018085015260a086015161163e6101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e086015250908501516102008401529050610d8d60208301848051825260208101516001600160401b0380825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b604080519081016001600160401b038111828210171561170b5761170b6116d3565b60405290565b604051602081016001600160401b038111828210171561170b5761170b6116d3565b604051608081016001600160401b038111828210171561170b5761170b6116d3565b60405161012081016001600160401b038111828210171561170b5761170b6116d3565b60405160a081016001600160401b038111828210171561170b5761170b6116d3565b604051606081016001600160401b038111828210171561170b5761170b6116d3565b604051601f8201601f191681016001600160401b03811182821017156117e4576117e46116d3565b604052919050565b8035600381106117fb57600080fd5b919050565b60006001600160401b03821115611819576118196116d3565b5060051b60200190565b60006040828403121561183557600080fd5b61183d6116e9565b905081356007811061184e57600080fd5b808252506020820135602082015292915050565b6000604080838503121561187557600080fd5b61187d6116e9565b915082356001600160401b038082111561189657600080fd5b818501915060208083880312156118ac57600080fd5b6118b4611711565b8335838111156118c357600080fd5b80850194505087601f8501126118d857600080fd5b833592506118ed6118e884611800565b6117bc565b83815260069390931b8401820192828101908985111561190c57600080fd5b948301945b84861015611932576119238a87611823565b82529486019490830190611911565b8252508552948501359484019490945250909392505050565b803563ffffffff811681146117fb57600080fd5b6000604080838503121561197257600080fd5b61197a6116e9565b915082356001600160401b0381111561199257600080fd5b8301601f810185136119a357600080fd5b803560206119b36118e883611800565b82815260a092830284018201928282019190898511156119d257600080fd5b948301945b84861015611a3b5780868b0312156119ef5760008081fd5b6119f7611733565b611a018b88611823565b815287870135858201526060611a1881890161194b565b89830152611a286080890161194b565b90820152835294850194918301916119d7565b50808752505080860135818601525050505092915050565b60006101208236031215611a6657600080fd5b611a6e611755565b611a77836117ec565b815260208301356001600160401b0380821115611a9357600080fd5b611a9f36838701611862565b60208401526040850135915080821115611ab857600080fd5b611ac436838701611862565b60408401526060850135915080821115611add57600080fd5b50611aea3682860161195f565b60608301525060808301356080820152611b0660a0840161194b565b60a0820152611b1760c0840161194b565b60c0820152611b2860e0840161194b565b60e082015261010092830135928101929092525090565b80356001600160401b03811681146117fb57600080fd5b600081830360e0811215611b6957600080fd5b611b71611778565b833581526060601f1983011215611b8757600080fd5b611b8f61179a565b9150611b9d60208501611b3f565b8252611bab60408501611b3f565b6020830152606084013560408301528160208201526080840135604082015260a08401356060820152611be060c0850161194b565b6080820152949350505050565b600060208284031215611bff57600080fd5b813561ffff81168114610d8d57600080fd5b634e487b7160e01b600052601160045260246000fd5b80820180821115610b5857610b58611c11565b634e487b7160e01b600052601260045260246000fd5b600082611c5f57611c5f611c3a565b500490565b600082611c7357611c73611c3a565b500690565b8082028115828204841417610b5857610b58611c11565b600060018201611ca157611ca1611c11565b5060010190565b6001600160401b03818116838216028082169190828114611ccb57611ccb611c11565b505092915050565b6001600160401b03828116828216039080821115611cf357611cf3611c11565b5092915050565b60006001600160401b0380841680611d1457611d14611c3a565b92169190910492915050565b81810381811115610b5857610b58611c11565b634e487b7160e01b600052603260045260246000fd5b600060ff821660ff8103611d5f57611d5f611c11565b60010192915050565b6000845160005b81811015611d895760208188018101518583015201611d6f565b5091909101928352506020820152604001919050565b634e487b7160e01b600052605160045260246000fdfea2646970667358221220ad7d72a504422e3d39c49c8dd177d800eec80e494aa8f189cbf3a86324c25c4664736f6c63430008110033",
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
	parsed, err := abi.JSON(strings.NewReader(OneStepProverMemoryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
