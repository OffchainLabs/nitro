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
	MaxInboxMessagesRead *big.Int
	Bridge               common.Address
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
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"machineStep\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"proveOneStep\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"afterHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
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

// ProveOneStep is a free data retrieval call binding the contract method 0x5d3adcfb.
//
// Solidity: function proveOneStep((uint256,address) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_IOneStepProofEntry *IOneStepProofEntryCaller) ProveOneStep(opts *bind.CallOpts, execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	var out []interface{}
	err := _IOneStepProofEntry.contract.Call(opts, &out, "proveOneStep", execCtx, machineStep, beforeHash, proof)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProveOneStep is a free data retrieval call binding the contract method 0x5d3adcfb.
//
// Solidity: function proveOneStep((uint256,address) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_IOneStepProofEntry *IOneStepProofEntrySession) ProveOneStep(execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	return _IOneStepProofEntry.Contract.ProveOneStep(&_IOneStepProofEntry.CallOpts, execCtx, machineStep, beforeHash, proof)
}

// ProveOneStep is a free data retrieval call binding the contract method 0x5d3adcfb.
//
// Solidity: function proveOneStep((uint256,address) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_IOneStepProofEntry *IOneStepProofEntryCallerSession) ProveOneStep(execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	return _IOneStepProofEntry.Contract.ProveOneStep(&_IOneStepProofEntry.CallOpts, execCtx, machineStep, beforeHash, proof)
}

// IOneStepProverMetaData contains all meta data concerning the IOneStepProver contract.
var IOneStepProverMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"mach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"mod\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"opcode\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"argumentData\",\"type\":\"uint256\"}],\"internalType\":\"structInstruction\",\"name\":\"instruction\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"result\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"resultMod\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
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

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod, (uint16,uint256) instruction, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) result, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) resultMod)
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

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod, (uint16,uint256) instruction, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) result, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) resultMod)
func (_IOneStepProver *IOneStepProverSession) ExecuteOneStep(execCtx ExecutionContext, mach Machine, mod Module, instruction Instruction, proof []byte) (struct {
	Result    Machine
	ResultMod Module
}, error) {
	return _IOneStepProver.Contract.ExecuteOneStep(&_IOneStepProver.CallOpts, execCtx, mach, mod, instruction, proof)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod, (uint16,uint256) instruction, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) result, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) resultMod)
func (_IOneStepProver *IOneStepProverCallerSession) ExecuteOneStep(execCtx ExecutionContext, mach Machine, mod Module, instruction Instruction, proof []byte) (struct {
	Result    Machine
	ResultMod Module
}, error) {
	return _IOneStepProver.Contract.ExecuteOneStep(&_IOneStepProver.CallOpts, execCtx, mach, mod, instruction, proof)
}

// OneStepProofEntryMetaData contains all meta data concerning the OneStepProofEntry contract.
var OneStepProofEntryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIOneStepProver\",\"name\":\"prover0_\",\"type\":\"address\"},{\"internalType\":\"contractIOneStepProver\",\"name\":\"proverMem_\",\"type\":\"address\"},{\"internalType\":\"contractIOneStepProver\",\"name\":\"proverMath_\",\"type\":\"address\"},{\"internalType\":\"contractIOneStepProver\",\"name\":\"proverHostIo_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"machineStep\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"proveOneStep\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"afterHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"prover0\",\"outputs\":[{\"internalType\":\"contractIOneStepProver\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proverHostIo\",\"outputs\":[{\"internalType\":\"contractIOneStepProver\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proverMath\",\"outputs\":[{\"internalType\":\"contractIOneStepProver\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proverMem\",\"outputs\":[{\"internalType\":\"contractIOneStepProver\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b506040516200234b3803806200234b8339810160408190526200003491620000a5565b600080546001600160a01b039586166001600160a01b031991821617909155600180549486169482169490941790935560028054928516928416929092179091556003805491909316911617905562000102565b80516001600160a01b0381168114620000a057600080fd5b919050565b60008060008060808587031215620000bc57600080fd5b620000c78562000088565b9350620000d76020860162000088565b9250620000e76040860162000088565b9150620000f76060860162000088565b905092959194509250565b61223980620001126000396000f3fe608060405234801561001057600080fd5b50600436106100575760003560e01c80631f128bc01461005c57806330a5509f1461008c5780635d3adcfb1461009f5780635f52fd7c146100c057806366e5d9c3146100d3575b600080fd5b60015461006f906001600160a01b031681565b6040516001600160a01b0390911681526020015b60405180910390f35b60005461006f906001600160a01b031681565b6100b26100ad3660046117eb565b6100e6565b604051908152602001610083565b60035461006f906001600160a01b031681565b60025461006f906001600160a01b031681565b60006100f06116e7565b6100f8611759565b6040805160208101909152606081526040805180820190915260008082526020820152600061012888888361068c565b9095509050886101378661085b565b1461017f5760405162461bcd60e51b815260206004820152601360248201527209a828690929c8abe848a8c9ea48abe9082a69606b1b60448201526064015b60405180910390fd5b60008551600381111561019457610194611886565b146101ae576101a28561085b565b95505050505050610683565b650800000000006101c08b60016118b2565b036101d257600285526101a28561085b565b6101dd888883610a46565b90945090506101ed888883610af2565b809250819450505084610100015161021a8660a0015163ffffffff168686610bcc9092919063ffffffff16565b146102565760405162461bcd60e51b815260206004820152600c60248201526b1353d115531154d7d493d3d560a21b6044820152606401610176565b60408051602081019091526060815260408051602081019091526060815261027f8a8a85610c15565b909450925061028f8a8a85610af2565b9350915061029e8a8a85610af2565b809450819250505060006102c78860e0015163ffffffff168685610c6f9092919063ffffffff16565b905060006102ea8960c0015163ffffffff168385610cb59092919063ffffffff16565b9050876060015181146103345760405162461bcd60e51b815260206004820152601260248201527110905117d1955390d51253d394d7d493d3d560721b6044820152606401610176565b506103479250899150839050818b6118c5565b975097505060008460a0015163ffffffff16905060018560e00181815161036e91906118ef565b63ffffffff1690525081516000602861ffff8316108015906103955750603561ffff831611155b806103b55750603661ffff8316108015906103b55750603e61ffff831611155b806103c4575061ffff8216603f145b806103d3575061ffff82166040145b156103ea57506001546001600160a01b03166105df565b61ffff821660451480610401575061ffff82166050145b8061042f5750604661ffff83161080159061042f575061042360096046611913565b61ffff168261ffff1611155b8061045d5750606761ffff83161080159061045d575061045160026067611913565b61ffff168261ffff1611155b8061047d5750606a61ffff83161080159061047d5750607861ffff831611155b806104ab5750605161ffff8316108015906104ab575061049f60096051611913565b61ffff168261ffff1611155b806104d95750607961ffff8316108015906104d957506104cd60026079611913565b61ffff168261ffff1611155b806104f95750607c61ffff8316108015906104f95750608a61ffff831611155b80610508575061ffff821660a7145b80610525575061ffff821660ac1480610525575061ffff821660ad145b80610545575060c061ffff831610801590610545575060c461ffff831611155b80610565575060bc61ffff831610801590610565575060bf61ffff831611155b1561057c57506002546001600160a01b03166105df565b61801061ffff831610801590610598575061801361ffff831611155b806105ba575061802061ffff8316108015906105ba575061802261ffff831611155b156105d157506003546001600160a01b03166105df565b506000546001600160a01b03165b806001600160a01b031663da78e7d18e8989888f8f6040518763ffffffff1660e01b815260040161061596959493929190611a70565b600060405180830381865afa158015610632573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f1916820160405261065a9190810190612027565b909750955061066a858488610bcc565b6101008801526106798761085b565b9750505050505050505b95945050505050565b6106946116e7565b816000806106a3878785610d2a565b9350905060ff81166000036106bb5760009150610735565b8060ff166001036106cf5760019150610735565b8060ff166002036106e35760029150610735565b8060ff166003036106f75760039150610735565b60405162461bcd60e51b8152602060048201526013602482015272554e4b4e4f574e5f4d4143485f53544154555360681b6044820152606401610176565b5061073e6117aa565b6107466117aa565b60008060008061076760408051808201909152606081526000602082015290565b60006107748e8e8c610d60565b9a5097506107838e8e8c610d60565b9a5096506107928e8e8c610e5f565b9a5091506107a18e8e8c610f87565b9a5095506107b08e8e8c610fa3565b9a5094506107bf8e8e8c610fa3565b9a5093506107ce8e8e8c610fa3565b9a5092506107dd8e8e8c610f87565b809b5081925050506040518061012001604052808a600381111561080357610803611886565b81526020018981526020018881526020018381526020018781526020018663ffffffff1681526020018563ffffffff1681526020018463ffffffff168152602001828152509a50505050505050505050935093915050565b6000808251600381111561087157610871611886565b0361093f576108838260200151611007565b6108908360400151611007565b61089d846060015161108c565b608085015160a086015160c087015160e0808901516101008a01516040516f26b0b1b434b73290393ab73734b7339d60811b602082015260308101999099526050890197909752607088019590955260908701939093526001600160e01b031991831b821660b0870152821b811660b486015291901b1660b883015260bc82015260dc015b604051602081830303815290604052805190602001209050919050565b60018251600381111561095457610954611886565b0361098b5760808201516040517026b0b1b434b732903334b734b9b432b21d60791b60208201526031810191909152605101610922565b6002825160038111156109a0576109a0611886565b036109c9576040516f26b0b1b434b7329032b93937b932b21d60811b6020820152603001610922565b6003825160038111156109de576109de611886565b03610a07576040516f26b0b1b434b732903a37b7903330b91d60811b6020820152603001610922565b60405162461bcd60e51b815260206004820152600f60248201526e4241445f4d4143485f53544154555360881b6044820152606401610176565b919050565b610a4e611759565b604080516060810182526000808252602082018190529181018290528391906000806000610a7d8a8a88610f87565b96509450610a8c8a8a88611125565b96509350610a9b8a8a88610f87565b96509250610aaa8a8a88610f87565b96509150610ab98a8a88610fa3565b6040805160a08101825297885260208801969096529486019390935250606084015263ffffffff16608083015290969095509350505050565b604080516020810190915260608152816000610b0f868684610d2a565b92509050600060ff82166001600160401b03811115610b3057610b30611bfe565b604051908082528060200260200182016040528015610b59578160200160208202803683370190505b50905060005b8260ff168160ff161015610bb057610b78888886610f87565b838360ff1681518110610b8d57610b8d612149565b602002602001018196508281525050508080610ba89061215f565b915050610b5f565b5060405180602001604052808281525093505050935093915050565b6000610c0d8484610bdc856111a0565b6040518060400160405280601381526020017226b7b23ab6329036b2b935b632903a3932b29d60691b81525061120d565b949350505050565b604080518082019091526000808252602082015281600080610c388787856112e2565b93509150610c4787878561133b565b6040805180820190915261ffff90941684526020840191909152919791965090945050505050565b6000610c0d8484610c7f85611390565b6040518060400160405280601881526020017724b739ba393ab1ba34b7b71036b2b935b632903a3932b29d60411b81525061120d565b60405168233ab731ba34b7b71d60b91b602082015260298101829052600090819060490160405160208183030381529060405280519060200120905061068385858360405180604001604052806015815260200174233ab731ba34b7b71036b2b935b632903a3932b29d60591b81525061120d565b600081848482818110610d3f57610d3f612149565b919091013560f81c9250819050610d558161217e565b915050935093915050565b610d686117aa565b816000610d76868684610f87565b925090506000610d8787878561133b565b935090506000816001600160401b03811115610da557610da5611bfe565b604051908082528060200260200182016040528015610dea57816020015b6040805180820190915260008082526020820152815260200190600190039081610dc35790505b50905060005b8151811015610e3857610e048989876113da565b838381518110610e1657610e16612149565b6020026020010181975082905250508080610e309061217e565b915050610df0565b50604080516060810182529081019182529081526020810192909252509590945092505050565b604080518082019091526060815260006020820152816000610e82868684610f87565b925090506060868684818110610e9a57610e9a612149565b909101356001600160f81b031916159050610f225782610eb98161217e565b604080516001808252818301909252919550909150816020015b610edb6117c8565b815260200190600190039081610ed3579050509050610efb8787856114d6565b82600081518110610f0e57610f0e612149565b602002602001018195508290525050610f66565b82610f2c8161217e565b60408051600080825260208201909252919550909150610f62565b610f4f6117c8565b815260200190600190039081610f475790505b5090505b60405180604001604052808281526020018381525093505050935093915050565b60008181610f9686868461133b565b9097909650945050505050565b600081815b6004811015610ffe5760088363ffffffff16901b9250858583818110610fd057610fd0612149565b919091013560f81c93909317925081610fe88161217e565b9250508080610ff69061217e565b915050610fa8565b50935093915050565b60208101518151515160005b818110156110855783516110309061102b908361156f565b6115a8565b6040516b2b30b63ab29039ba30b1b59d60a11b6020820152602c810191909152604c8101849052606c01604051602081830303815290604052805190602001209250808061107d9061217e565b915050611013565b5050919050565b602081015160005b82515181101561111f576110c4836000015182815181106110b7576110b7612149565b60200260200101516115c5565b6040517129ba30b1b590333930b6b29039ba30b1b59d60711b602082015260328101919091526052810183905260720160405160208183030381529060405280519060200120915080806111179061217e565b915050611094565b50919050565b60408051606081018252600080825260208201819052918101919091528160008080611152888886611635565b94509250611161888886611635565b94509150611170888886610f87565b604080516060810182526001600160401b0396871681529490951660208501529383015250969095509350505050565b600081600001516111b48360200151611693565b6040848101516060860151608087015192516626b7b23ab6329d60c91b6020820152602781019590955260478501939093526067840152608783019190915260e01b6001600160e01b03191660a782015260ab01610922565b8160005b8551518110156112d957846001166000036112755782828760000151838151811061123e5761123e612149565b602002602001015160405160200161125893929190612197565b6040516020818303038152906040528051906020012091506112c0565b828660000151828151811061128c5761128c612149565b6020026020010151836040516020016112a793929190612197565b6040516020818303038152906040528051906020012091505b60019490941c93806112d18161217e565b915050611211565b50949350505050565b600081815b6002811015610ffe5760088361ffff16901b925085858381811061130d5761130d612149565b919091013560f81c939093179250816113258161217e565b92505080806113339061217e565b9150506112e7565b600081815b6020811015610ffe57600883901b925085858381811061136257611362612149565b919091013560f81c9390931792508161137a8161217e565b92505080806113889061217e565b915050611340565b6000816000015182602001516040516020016109229291906b24b739ba393ab1ba34b7b71d60a11b815260f09290921b6001600160f01b031916600c830152600e820152602e0190565b604080518082019091526000808252602082015281600085858381811061140357611403612149565b919091013560f81c91508290506114198161217e565b925050611424600690565b600681111561143557611435611886565b60ff168160ff16111561147b5760405162461bcd60e51b815260206004820152600e60248201526d4241445f56414c55455f5459504560901b6044820152606401610176565b600061148887878561133b565b809450819250505060405180604001604052808360ff1660068111156114b0576114b0611886565b60068111156114c1576114c1611886565b81526020018281525093505050935093915050565b6114de6117c8565b816114f9604080518082019091526000808252602082015290565b60008060006115098989876113da565b95509350611518898987610f87565b95509250611527898987610fa3565b95509150611536898987610fa3565b60408051608081018252968752602087019590955263ffffffff9384169486019490945290911660608401525090969095509350505050565b6040805180820190915260008082526020820152825180518390811061159757611597612149565b602002602001015190505b92915050565b6000816000015182602001516040516020016109229291906121ce565b60006115d482600001516115a8565b602080840151604080860151606087015191516b29ba30b1b590333930b6b29d60a11b94810194909452602c840194909452604c8301919091526001600160e01b031960e093841b8116606c840152921b9091166070820152607401610922565b600081815b6008811015610ffe576008836001600160401b0316901b925085858381811061166557611665612149565b919091013560f81c9390931792508161167d8161217e565b925050808061168b9061217e565b91505061163a565b805160208083015160408085015190516626b2b6b7b93c9d60c91b938101939093526001600160c01b031960c094851b811660278501529190931b16602f8201526037810191909152600090605701610922565b60408051610120810190915280600081526020016117036117aa565b81526020016117106117aa565b815260200161173060408051808201909152606081526000602082015290565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b6040805160a08101909152600081526020810161178f604080516060810182526000808252602082018190529181019190915290565b81526000602082018190526040820181905260609091015290565b60408051606080820183529181019182529081526000602082015290565b6040805160c0810190915260006080820181815260a0830191909152819061178f565b600080600080600085870360a081121561180457600080fd5b604081121561181257600080fd5b50859450604086013593506060860135925060808601356001600160401b038082111561183e57600080fd5b818801915088601f83011261185257600080fd5b81358181111561186157600080fd5b89602082850101111561187357600080fd5b9699959850939650602001949392505050565b634e487b7160e01b600052602160045260246000fd5b634e487b7160e01b600052601160045260246000fd5b808201808211156115a2576115a261189c565b600080858511156118d557600080fd5b838611156118e257600080fd5b5050820193919092039150565b63ffffffff81811683821601908082111561190c5761190c61189c565b5092915050565b61ffff81811683821601908082111561190c5761190c61189c565b6004811061193e5761193e611886565b9052565b80516007811061195457611954611886565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b808410156119b15761199d828651611942565b93820193600193909301929085019061198a565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b82811015611a305784516119fc858251611942565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a0909301926001016119e7565b509687015197909601969096525093949350505050565b81835281816020850137506000828201602090810191909152601f909101601f19169091010190565b8635815260006101a060208901356001600160a01b038116808214611a9457600080fd5b8060208601525050806040840152611aaf818401895161192e565b5060208701516101206101c0840152611acc6102c0840182611961565b9050604088015161019f1980858403016101e0860152611aec8383611961565b925060608a01519150808584030161020086015250611b0b82826119c5565b915050608088015161022084015260a0880151611b3161024085018263ffffffff169052565b5060c088015163ffffffff81166102608501525060e088015163ffffffff8116610280850152506101008801516102a0840152611bc660608401888051825260208101516001600160401b0380825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b855161ffff166101408401526020860151610160840152828103610180840152611bf1818587611a47565b9998505050505050505050565b634e487b7160e01b600052604160045260246000fd5b604080519081016001600160401b0381118282101715611c3657611c36611bfe565b60405290565b604051602081016001600160401b0381118282101715611c3657611c36611bfe565b604051608081016001600160401b0381118282101715611c3657611c36611bfe565b60405160a081016001600160401b0381118282101715611c3657611c36611bfe565b604051606081016001600160401b0381118282101715611c3657611c36611bfe565b60405161012081016001600160401b0381118282101715611c3657611c36611bfe565b604051601f8201601f191681016001600160401b0381118282101715611d0f57611d0f611bfe565b604052919050565b805160048110610a4157600080fd5b60006001600160401b03821115611d3f57611d3f611bfe565b5060051b60200190565b600060408284031215611d5b57600080fd5b611d63611c14565b9050815160078110611d7457600080fd5b808252506020820151602082015292915050565b60006040808385031215611d9b57600080fd5b611da3611c14565b915082516001600160401b0380821115611dbc57600080fd5b81850191506020808388031215611dd257600080fd5b611dda611c3c565b835183811115611de957600080fd5b80850194505087601f850112611dfe57600080fd5b83519250611e13611e0e84611d26565b611ce7565b83815260069390931b84018201928281019089851115611e3257600080fd5b948301945b84861015611e5857611e498a87611d49565b82529486019490830190611e37565b8252508552948501519484019490945250909392505050565b805163ffffffff81168114610a4157600080fd5b60006040808385031215611e9857600080fd5b611ea0611c14565b915082516001600160401b03811115611eb857600080fd5b8301601f81018513611ec957600080fd5b80516020611ed9611e0e83611d26565b82815260a09283028401820192828201919089851115611ef857600080fd5b948301945b84861015611f615780868b031215611f155760008081fd5b611f1d611c5e565b611f278b88611d49565b815287870151858201526060611f3e818901611e71565b89830152611f4e60808901611e71565b9082015283529485019491830191611efd565b50808752505080860151818601525050505092915050565b80516001600160401b0381168114610a4157600080fd5b600081830360e0811215611fa357600080fd5b611fab611c80565b8351815291506060601f1982011215611fc357600080fd5b50611fcc611ca2565b611fd860208401611f79565b8152611fe660408401611f79565b602082015260608301516040820152806020830152506080820151604082015260a0820151606082015261201c60c08301611e71565b608082015292915050565b60008061010080848603121561203c57600080fd5b83516001600160401b038082111561205357600080fd5b90850190610120828803121561206857600080fd5b612070611cc4565b61207983611d17565b815260208301518281111561208d57600080fd5b61209989828601611d88565b6020830152506040830151828111156120b157600080fd5b6120bd89828601611d88565b6040830152506060830151828111156120d557600080fd5b6120e189828601611e85565b606083015250608083015160808201526120fd60a08401611e71565b60a082015261210e60c08401611e71565b60c082015261211f60e08401611e71565b60e08201528383015184820152809550505050506121408460208501611f90565b90509250929050565b634e487b7160e01b600052603260045260246000fd5b600060ff821660ff81036121755761217561189c565b60010192915050565b6000600182016121905761219061189c565b5060010190565b6000845160005b818110156121b8576020818801810151858301520161219e565b5091909101928352506020820152604001919050565b652b30b63ab29d60d11b81526000600784106121ec576121ec611886565b5060f89290921b600683015260078201526027019056fea2646970667358221220d6ae7034ea3eb5459ceba09ef16b047001b646649f398156b82de18d81f1b28e64736f6c63430008110033",
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

// ProveOneStep is a free data retrieval call binding the contract method 0x5d3adcfb.
//
// Solidity: function proveOneStep((uint256,address) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_OneStepProofEntry *OneStepProofEntryCaller) ProveOneStep(opts *bind.CallOpts, execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	var out []interface{}
	err := _OneStepProofEntry.contract.Call(opts, &out, "proveOneStep", execCtx, machineStep, beforeHash, proof)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProveOneStep is a free data retrieval call binding the contract method 0x5d3adcfb.
//
// Solidity: function proveOneStep((uint256,address) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
func (_OneStepProofEntry *OneStepProofEntrySession) ProveOneStep(execCtx ExecutionContext, machineStep *big.Int, beforeHash [32]byte, proof []byte) ([32]byte, error) {
	return _OneStepProofEntry.Contract.ProveOneStep(&_OneStepProofEntry.CallOpts, execCtx, machineStep, beforeHash, proof)
}

// ProveOneStep is a free data retrieval call binding the contract method 0x5d3adcfb.
//
// Solidity: function proveOneStep((uint256,address) execCtx, uint256 machineStep, bytes32 beforeHash, bytes proof) view returns(bytes32 afterHash)
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
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212208255c9058e803bcf53c83081c0881febcdee8c6a7e7a679050ba2642c57a132e64736f6c63430008110033",
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
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"startMach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"startMod\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"opcode\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"argumentData\",\"type\":\"uint256\"}],\"internalType\":\"structInstruction\",\"name\":\"inst\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"mach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"mod\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061261e806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063da78e7d114610030575b600080fd5b61004361003e366004611bc3565b61005a565b604051610051929190611dd5565b60405180910390f35b610062611a8b565b61006a611b34565b6100738761227d565b915061008436879003870187612380565b905060006100956020870187612417565b9050611b7e61ffff82166100ac57506102bd61029f565b60001961ffff8316016100c257506102c861029f565b600e1961ffff8316016100d857506102cf61029f565b600f1961ffff8316016100ee57506103fa61029f565b6180081961ffff831601610105575061049461029f565b6180091961ffff83160161011c575061054061029f565b60101961ffff831601610132575061062c61029f565b6180021961ffff8316016101495750610a0b61029f565b6180031961ffff8316016101605750610a4a61029f565b601f1961ffff8316016101765750610aa861029f565b60201961ffff83160161018c5750610aea61029f565b60221961ffff8316016101a25750610b2f61029f565b60231961ffff8316016101b85750610b5761029f565b6180011961ffff8316016101cf5750610b8761029f565b60191961ffff8316016101e55750610c2461029f565b601a1961ffff8316016101fb5750610c3161029f565b604161ffff8316108015906102155750604461ffff831611155b156102235750610ca061029f565b61ffff8216618005148061023c575061ffff8216618006145b1561024a5750610d9461029f565b6180071961ffff8316016102615750610e6561029f565b60405162461bcd60e51b815260206004820152600e60248201526d494e56414c49445f4f50434f444560901b60448201526064015b60405180910390fd5b6102b084848989898663ffffffff16565b5050965096945050505050565b505060029092525050565b5050505050565b60006102de8660600151610e74565b9050600481515160068111156102f6576102f6611ca6565b0361031c578560025b9081600381111561031257610312611ca6565b81525050506102c8565b6006815151600681111561033257610332611ca6565b146103785760405162461bcd60e51b8152602060048201526016602482015275494e56414c49445f52455455524e5f50435f5459504560501b6044820152606401610296565b805160209081015190819081901c604082901c606083901c156103d65760405162461bcd60e51b8152602060048201526016602482015275494e56414c49445f52455455524e5f50435f4441544160501b6044820152606401610296565b63ffffffff92831660e08b015290821660c08a01521660a088015250505050505050565b61041161040686610f14565b602087015190610f77565b60006104208660600151610f87565b905061043d6104328260400151610fd3565b602088015190610f77565b61044d6104328260600151610fd3565b602084013563ffffffff811681146104775760405162461bcd60e51b81526004016102969061243b565b63ffffffff1660c08701525050600060e090940193909352505050565b6104a061040686610f14565b6104b06104068660a00151610fd3565b6104c06104068560800151610fd3565b6020808401359081901c604082901c1561051c5760405162461bcd60e51b815260206004820152601a60248201527f4241445f43524f53535f4d4f44554c455f43414c4c5f444154410000000000006044820152606401610296565b63ffffffff90811660a08801521660c08601525050600060e0909301929092525050565b61054c61040686610f14565b61055c6104068660a00151610fd3565b61056c6104068560800151610fd3565b600061057b8660600151610f87565b9050806060015163ffffffff16600003610597578560026102ff565b602084013563ffffffff811681146105f15760405162461bcd60e51b815260206004820152601d60248201527f4241445f43414c4c45525f494e5445524e414c5f43414c4c5f444154410000006044820152606401610296565b604082015163ffffffff1660a08801526060820151610611908290612478565b63ffffffff1660c08801525050600060e08601525050505050565b60008061064461063f8860200151611006565b61102b565b90506000806000808060006106656040518060200160405280606081525090565b6106708b8b876110bc565b9550935061067f8b8b87611123565b909650945061068f8b8b8761113f565b9550925061069e8b8b876110bc565b955091506106ad8b8b87611123565b90975094506106bd8b8b87611175565b6040516d21b0b6361034b73234b932b1ba1d60911b60208201526001600160c01b031960c088901b16602e8201526036810189905290965090915060009060560160408051601f19818403018152919052805160209182012091508d013581146107625760405162461bcd60e51b81526020600482015260166024820152754241445f43414c4c5f494e4449524543545f4441544160501b6044820152606401610296565b610778826001600160401b03871686868c61124f565b90508d6040015181146107bf5760405162461bcd60e51b815260206004820152600f60248201526e10905117d51050931154d7d493d3d5608a1b6044820152606401610296565b826001600160401b03168963ffffffff16106107e957505060028d52506102c89650505050505050565b5050505050600061080a604080518082019091526000808252602082015290565b6040805160208101909152606081526108248a8a86611123565b945092506108338a8a866112f1565b945091506108428a8a86611175565b94509050600061085f8263ffffffff808b1690879087906113ed16565b90508681146108a45760405162461bcd60e51b815260206004820152601160248201527010905117d153115351539514d7d493d3d5607a1b6044820152606401610296565b8584146108d4578d60025b908160038111156108c2576108c2611ca6565b815250505050505050505050506102c8565b6004835160068111156108e9576108e9611ca6565b036108f6578d60026108af565b60058351600681111561090b5761090b611ca6565b03610969576020830151985063ffffffff891689146109645760405162461bcd60e51b81526020600482015260156024820152744241445f46554e435f5245465f434f4e54454e545360581b6044820152606401610296565b6109a1565b60405162461bcd60e51b815260206004820152600d60248201526c4241445f454c454d5f5459504560981b6044820152606401610296565b50505050505050506109b561043287610f14565b60006109c48760600151610f87565b90506109e16109d68260400151610fd3565b602089015190610f77565b6109f16109d68260600151610fd3565b5063ffffffff1660c0860152600060e08601525050505050565b602083013563ffffffff81168114610a355760405162461bcd60e51b81526004016102969061243b565b63ffffffff1660e09095019490945250505050565b6000610a5c61063f8760200151611006565b905063ffffffff811615610aa057602084013563ffffffff81168114610a945760405162461bcd60e51b81526004016102969061243b565b63ffffffff1660e08701525b505050505050565b6000610ab78660600151610f87565b90506000610acf826020015186602001358686611487565b6020880151909150610ae19082610f77565b50505050505050565b6000610af98660200151611006565b90506000610b0a8760600151610f87565b9050610b218160200151866020013584878761151f565b602090910152505050505050565b6000610b45856000015185602001358585611487565b6020870151909150610aa09082610f77565b6000610b668660200151611006565b9050610b7d8560000151856020013583868661151f565b9094525050505050565b6000610b968660200151611006565b90506000610ba78760200151611006565b90506000610bb88860200151611006565b905060006040518060800160405280838152602001886020013560001b8152602001610be38561102b565b63ffffffff168152602001610bf78661102b565b63ffffffff168152509050610c19818a606001516115b990919063ffffffff16565b505050505050505050565b610aa08560200151611006565b6000610c4361063f8760200151611006565b90506000610c548760200151611006565b90506000610c658860200151611006565b905063ffffffff831615610c87576020880151610c829082610f77565b610c96565b6020880151610c969083610f77565b5050505050505050565b6000610caf6020850185612417565b9050600060401961ffff831601610cc857506000610d4b565b60411961ffff831601610cdd57506001610d4b565b60421961ffff831601610cf257506002610d4b565b60431961ffff831601610d0757506003610d4b565b60405162461bcd60e51b8152602060048201526019602482015278434f4e53545f505553485f494e56414c49445f4f50434f444560381b6044820152606401610296565b610ae16040518060400160405280836006811115610d6b57610d6b611ca6565b815260200187602001356001600160401b03168152508860200151610f7790919063ffffffff16565b6040805180820190915260008082526020820152618005610db86020860186612417565b61ffff1603610de557610dce8660200151611006565b6040870151909150610de09082610f77565b610aa0565b618006610df56020860186612417565b61ffff1603610e1d57610e0b8660400151611006565b6020870151909150610de09082610f77565b60405162461bcd60e51b815260206004820152601c60248201527f4d4f56455f494e5445524e414c5f494e56414c49445f4f50434f4445000000006044820152606401610296565b6000610b4586602001516116a0565b610e7c611b88565b815151600114610e9e5760405162461bcd60e51b81526004016102969061249c565b81518051600090610eb157610eb16124c7565b6020026020010151905060006001600160401b03811115610ed457610ed4611efd565b604051908082528060200260200182016040528015610f0d57816020015b610efa611b88565b815260200190600190039081610ef25790505b5090915290565b604080518082018252600080825260209182015260e083015160c084015160a090940151835180850185526006815263ffffffff90921694831b67ffffffff0000000016949094179390921b63ffffffff60401b16929092179181019190915290565b8151610f8390826116d5565b5050565b610f8f611b88565b815151600114610fb15760405162461bcd60e51b81526004016102969061249c565b81518051600090610fc457610fc46124c7565b60200260200101519050919050565b604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b604080518082019091526000808252602082015281516110259061179e565b92915050565b6020810151600090818351600681111561104757611047611ca6565b1461107e5760405162461bcd60e51b81526020600482015260076024820152662727aa2fa4999960c91b6044820152606401610296565b64010000000081106110255760405162461bcd60e51b81526020600482015260076024820152662120a22fa4999960c91b6044820152606401610296565b600081815b600881101561111a576008836001600160401b0316901b92508585838181106110ec576110ec6124c7565b919091013560f81c93909317925081611104816124dd565b9250508080611112906124dd565b9150506110c1565b50935093915050565b600081816111328686846118a7565b9097909650945050505050565b600081848482818110611154576111546124c7565b919091013560f81c925081905061116a816124dd565b915050935093915050565b60408051602081019091526060815281600061119286868461113f565b92509050600060ff82166001600160401b038111156111b3576111b3611efd565b6040519080825280602002602001820160405280156111dc578160200160208202803683370190505b50905060005b8260ff168160ff161015611233576111fb888886611123565b838360ff1681518110611210576112106124c7565b60200260200101819650828152505050808061122b906124f6565b9150506111e2565b5060405180602001604052808281525093505050935093915050565b604051652a30b136329d60d11b60208201526001600160f81b031960f885901b1660268201526001600160c01b031960c084901b166027820152602f81018290526000908190604f016040516020818303038152906040528051906020012090506112e6878783604051806040016040528060128152602001712a30b136329036b2b935b632903a3932b29d60711b8152506118fc565b979650505050505050565b604080518082019091526000808252602082015281600085858381811061131a5761131a6124c7565b919091013560f81c9150829050611330816124dd565b92505061133b600690565b600681111561134c5761134c611ca6565b60ff168160ff1611156113925760405162461bcd60e51b815260206004820152600e60248201526d4241445f56414c55455f5459504560901b6044820152606401610296565b600061139f8787856118a7565b809450819250505060405180604001604052808360ff1660068111156113c7576113c7611ca6565b60068111156113d8576113d8611ca6565b81526020018281525093505050935093915050565b600080836113fa846119d1565b6040516d2a30b136329032b632b6b2b73a1d60911b6020820152602e810192909252604e820152606e0160405160208183030381529060405280519060200120905061147d8686836040518060400160405280601a81526020017f5461626c6520656c656d656e74206d65726b6c6520747265653a0000000000008152506118fc565b9695505050505050565b604080518082019091526000808252602082015260006114b7604080518082019091526000808252602082015290565b6040805160208101909152606081526114d18686856112f1565b935091506114e0868685611175565b9350905060006114f1828985611a0b565b90508881146115125760405162461bcd60e51b815260040161029690612515565b5090979650505050505050565b600061153b604080518082019091526000808252602082015290565b60006115536040518060200160405280606081525090565b61155e8686846112f1565b909350915061156e868684611175565b92509050600061157f828a86611a0b565b90508981146115a05760405162461bcd60e51b815260040161029690612515565b6115ab828a8a611a0b565b9a9950505050505050505050565b8151516000906115ca906001612540565b6001600160401b038111156115e1576115e1611efd565b60405190808252806020026020018201604052801561161a57816020015b611607611b88565b8152602001906001900390816115ff5790505b50905060005b83515181101561167657835180518290811061163e5761163e6124c7565b6020026020010151828281518110611658576116586124c7565b6020026020010181905250808061166e906124dd565b915050611620565b5081818460000151518151811061168f5761168f6124c7565b602090810291909101015290915250565b6040805180820190915260008082526020820152815151516116ce6116c6600183612553565b845190611a53565b9392505050565b8151516000906116e6906001612540565b6001600160401b038111156116fd576116fd611efd565b60405190808252806020026020018201604052801561174257816020015b604080518082019091526000808252602082015281526020019060019003908161171b5790505b50905060005b835151811015611676578351805182908110611766576117666124c7565b6020026020010151828281518110611780576117806124c7565b60200260200101819052508080611796906124dd565b915050611748565b6040805180820190915260008082526020820152815180516117c290600190612553565b815181106117d2576117d26124c7565b60200260200101519050600060018360000151516117f09190612553565b6001600160401b0381111561180757611807611efd565b60405190808252806020026020018201604052801561184c57816020015b60408051808201909152600080825260208201528152602001906001900390816118255790505b50905060005b8151811015610f0d57835180518290811061186f5761186f6124c7565b6020026020010151828281518110611889576118896124c7565b6020026020010181905250808061189f906124dd565b915050611852565b600081815b602081101561111a57600883901b92508585838181106118ce576118ce6124c7565b919091013560f81c939093179250816118e6816124dd565b92505080806118f4906124dd565b9150506118ac565b8160005b8551518110156119c857846001166000036119645782828760000151838151811061192d5761192d6124c7565b602002602001015160405160200161194793929190612566565b6040516020818303038152906040528051906020012091506119af565b828660000151828151811061197b5761197b6124c7565b60200260200101518360405160200161199693929190612566565b6040516020818303038152906040528051906020012091505b60019490941c93806119c0816124dd565b915050611900565b50949350505050565b6000816000015182602001516040516020016119ee92919061259d565b604051602081830303815290604052805190602001209050919050565b6000611a4b8484611a1b856119d1565b604051806040016040528060128152602001712b30b63ab29036b2b935b632903a3932b29d60711b8152506118fc565b949350505050565b60408051808201909152600080825260208201528251805183908110611a7b57611a7b6124c7565b6020026020010151905092915050565b6040805161012081019091528060008152602001611ac060408051606080820183529181019182529081526000602082015290565b8152602001611ae660408051606080820183529181019182529081526000602082015290565b8152602001611b0b604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b6040805160a0810182526000808252825160608101845281815260208181018390529381019190915290918201905b81526000602082018190526040820181905260609091015290565b611b866125d2565b565b6040805160c0810190915260006080820181815260a08301919091528190611b63565b600060408284031215611bbd57600080fd5b50919050565b6000806000806000808688036101a0811215611bde57600080fd5b611be88989611bab565b965060408801356001600160401b0380821115611c0457600080fd5b90890190610120828c031215611c1957600080fd5b81975060e0605f1984011215611c2e57600080fd5b60608a019650611c428b6101408c01611bab565b95506101808a0135925080831115611c5957600080fd5b828a0192508a601f840112611c6d57600080fd5b8235915080821115611c7e57600080fd5b50896020828401011115611c9157600080fd5b60208201935080925050509295509295509295565b634e487b7160e01b600052602160045260246000fd5b60048110611ccc57611ccc611ca6565b9052565b805160078110611ce257611ce2611ca6565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b80841015611d3f57611d2b828651611cd0565b938201936001939093019290850190611d18565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b82811015611dbe578451611d8a858251611cd0565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101611d75565b509687015197909601969096525093949350505050565b6000610100808352611dea8184018651611cbc565b602085015161012084810152611e04610220850182611cef565b9050604086015160ff198086840301610140870152611e238383611cef565b925060608801519150808684030161016087015250611e428282611d53565b915050608086015161018085015260a0860151611e686101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e0860152509085015161020084015290506116ce60208301848051825260208101516001600160401b0380825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b604080519081016001600160401b0381118282101715611f3557611f35611efd565b60405290565b604051602081016001600160401b0381118282101715611f3557611f35611efd565b604051608081016001600160401b0381118282101715611f3557611f35611efd565b60405161012081016001600160401b0381118282101715611f3557611f35611efd565b60405160a081016001600160401b0381118282101715611f3557611f35611efd565b604051606081016001600160401b0381118282101715611f3557611f35611efd565b604051601f8201601f191681016001600160401b038111828210171561200e5761200e611efd565b604052919050565b80356004811061202557600080fd5b919050565b60006001600160401b0382111561204357612043611efd565b5060051b60200190565b60006040828403121561205f57600080fd5b612067611f13565b905081356007811061207857600080fd5b808252506020820135602082015292915050565b6000604080838503121561209f57600080fd5b6120a7611f13565b915082356001600160401b03808211156120c057600080fd5b818501915060208083880312156120d657600080fd5b6120de611f3b565b8335838111156120ed57600080fd5b80850194505087601f85011261210257600080fd5b833592506121176121128461202a565b611fe6565b83815260069390931b8401820192828101908985111561213657600080fd5b948301945b8486101561215c5761214d8a8761204d565b8252948601949083019061213b565b8252508552948501359484019490945250909392505050565b803563ffffffff8116811461202557600080fd5b6000604080838503121561219c57600080fd5b6121a4611f13565b915082356001600160401b038111156121bc57600080fd5b8301601f810185136121cd57600080fd5b803560206121dd6121128361202a565b82815260a092830284018201928282019190898511156121fc57600080fd5b948301945b848610156122655780868b0312156122195760008081fd5b612221611f5d565b61222b8b8861204d565b815287870135858201526060612242818901612175565b8983015261225260808901612175565b9082015283529485019491830191612201565b50808752505080860135818601525050505092915050565b6000610120823603121561229057600080fd5b612298611f7f565b6122a183612016565b815260208301356001600160401b03808211156122bd57600080fd5b6122c93683870161208c565b602084015260408501359150808211156122e257600080fd5b6122ee3683870161208c565b6040840152606085013591508082111561230757600080fd5b5061231436828601612189565b6060830152506080830135608082015261233060a08401612175565b60a082015261234160c08401612175565b60c082015261235260e08401612175565b60e082015261010092830135928101929092525090565b80356001600160401b038116811461202557600080fd5b600081830360e081121561239357600080fd5b61239b611fa2565b833581526060601f19830112156123b157600080fd5b6123b9611fc4565b91506123c760208501612369565b82526123d560408501612369565b6020830152606084013560408301528160208201526080840135604082015260a0840135606082015261240a60c08501612175565b6080820152949350505050565b60006020828403121561242957600080fd5b813561ffff811681146116ce57600080fd5b6020808252600d908201526c4241445f43414c4c5f4441544160981b604082015260600190565b634e487b7160e01b600052601160045260246000fd5b63ffffffff81811683821601908082111561249557612495612462565b5092915050565b6020808252601190820152700848288beae929c889eaebe988a9c8ea89607b1b604082015260600190565b634e487b7160e01b600052603260045260246000fd5b6000600182016124ef576124ef612462565b5060010190565b600060ff821660ff810361250c5761250c612462565b60010192915050565b60208082526011908201527015d493d391d7d3515492d31157d493d3d5607a1b604082015260600190565b8082018082111561102557611025612462565b8181038181111561102557611025612462565b6000845160005b81811015612587576020818801810151858301520161256d565b5091909101928352506020820152604001919050565b652b30b63ab29d60d11b81526000600784106125bb576125bb611ca6565b5060f89290921b6006830152600782015260270190565b634e487b7160e01b600052605160045260246000fdfea2646970667358221220578ad9138454fc6087bf6a4d637d00c8b8b152a98477f2b180bf20346a9faab064736f6c63430008110033",
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

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
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

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProver0 *OneStepProver0Session) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProver0.Contract.ExecuteOneStep(&_OneStepProver0.CallOpts, arg0, startMach, startMod, inst, proof)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProver0 *OneStepProver0CallerSession) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProver0.Contract.ExecuteOneStep(&_OneStepProver0.CallOpts, arg0, startMach, startMod, inst, proof)
}

// OneStepProverHostIoMetaData contains all meta data concerning the OneStepProverHostIo contract.
var OneStepProverHostIoMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"startMach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"startMod\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"opcode\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"argumentData\",\"type\":\"uint256\"}],\"internalType\":\"structInstruction\",\"name\":\"inst\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"mach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"mod\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506125ef806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063da78e7d114610030575b600080fd5b61004361003e366004611b10565b61005a565b604051610051929190611d22565b60405180910390f35b610062611a02565b6040805160a0810182526000808252825160608082018552828252602080830184905282860184905284019190915292820181905291810182905260808101919091526100ae876121ca565b91506100bf368790038701876122cd565b905060006100d06020870187612364565b9050611aab61801061ffff8316108015906100f1575061801361ffff831611155b156100ff57506101a8610189565b61801f1961ffff83160161011657506102f0610189565b6180201961ffff83160161012d5750610593610189565b6180211961ffff83160161014457506108b5610189565b60405162461bcd60e51b8152602060048201526015602482015274494e56414c49445f4d454d4f52595f4f50434f444560581b60448201526064015b60405180910390fd5b61019b8a85858a8a8a8763ffffffff16565b5050965096945050505050565b60006101b76020850185612364565b90506101c1611ab5565b60006101ce8585836108c1565b60808a015191935091506101e18361099c565b146102215760405162461bcd60e51b815260206004820152601060248201526f4241445f474c4f42414c5f535441544560801b6044820152606401610180565b61ffff8316618010148061023a575061ffff8316618011145b1561025c57610257888884896102528987818d612388565b610a10565b6102d4565b6180111961ffff841601610274576102578883610bb6565b6180121961ffff84160161028c576102578883610c4d565b60405162461bcd60e51b815260206004820152601a60248201527f494e56414c49445f474c4f42414c53544154455f4f50434f44450000000000006044820152606401610180565b6102dd8261099c565b6080909801979097525050505050505050565b60006103076103028760200151610cc3565b610ce8565b63ffffffff16905060006103216103028860200151610cc3565b63ffffffff1690508560200151600001516001600160401b031681602061034891906123c8565b118061035d575061035a6020826123f1565b15155b15610384578660025b9081600381111561037957610379611bf3565b81525050505061058b565b6000610391602083612405565b90506000806103ac6040518060200160405280606081525090565b60208a01516103be90858a8a87610d79565b9094509092509050606060008989868181106103dc576103dc612419565b919091013560f81c91508590506103f28161242f565b9550508060ff166000036104ce5736600061040f8b88818f612388565b91509150858282604051610424929190612448565b6040518091039020146104685760405162461bcd60e51b815260206004820152600c60248201526b4241445f505245494d41474560a01b6044820152606401610180565b60006104758b60206123c8565b9050818111156104825750805b61048e818c8486612388565b8080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525092975061050f95505050505050565b60405162461bcd60e51b81526020600482015260166024820152752aa725a727aba72fa82922a4a6a0a3a2afa82927a7a360511b6044820152606401610180565b60005b82518110156105535761053f858285848151811061053257610532612419565b016020015160f81c610e13565b94508061054b8161242f565b915050610512565b5061055f838786610e99565b60208d01516040015281516105829061057790610f18565b60208f015190610f4b565b50505050505050505b505050505050565b60006105a56103028760200151610cc3565b63ffffffff16905060006105bf6103028860200151610cc3565b63ffffffff16905060006105de6105d98960200151610cc3565b610f5b565b6001600160401b0316905060208601351580156105fc575088358110155b15610624578760035b9081600381111561061857610618611bf3565b8152505050505061058b565b602080880151516001600160401b0316906106409084906123c8565b118061065557506106526020836123f1565b15155b1561066257876002610605565b600061066f602084612405565b905060008061068a6040518060200160405280606081525090565b60208b015161069c90858b8b87610d79565b90945090925090508888848181106106b6576106b6612419565b909101356001600160f81b0319161590506107095760405162461bcd60e51b81526020600482015260136024820152722aa725a727aba72fa4a72127ac2fa82927a7a360691b6044820152606401610180565b826107138161242f565b9350611aab9050600060208c013561072f57610fec915061076e565b60018c6020013503610745576112be915061076e565b8d60025b9081600381111561075c5761075c611bf3565b8152505050505050505050505061058b565b61078e8f888d8d8990809261078593929190612388565b8663ffffffff16565b90508061079d578d6002610749565b5050828810156107e35760405162461bcd60e51b81526020600482015260116024820152702120a22fa6a2a9a9a0a3a2afa82927a7a360791b6044820152606401610180565b60006107ef848a612458565b905060005b60208163ffffffff1610801561081857508161081663ffffffff83168b6123c8565b105b156108715761085d8463ffffffff83168d8d826108358f8c6123c8565b61083f91906123c8565b81811061084e5761084e612419565b919091013560f81c9050610e13565b9350806108698161246b565b9150506107f4565b61087c838786610e99565b60208e0151604001526108a461089182610f18565b8f60200151610f4b90919063ffffffff16565b505050505050505050505050505050565b50506001909252505050565b6108c9611ab5565b816108d2611ada565b6108da611ada565b60005b600260ff82161015610925576108f4888886611542565b848360ff166002811061090957610909612419565b602002019190915293508061091d8161248e565b9150506108dd565b5060005b600260ff8216101561097f5761094088888661155e565b838360ff166002811061095557610955612419565b6001600160401b0390931660209390930201919091529350806109778161248e565b915050610929565b506040805180820190915291825260208201529590945092505050565b8051805160209182015192820151805190830151604080516c23b637b130b61039ba30ba329d60991b81870152602d810194909452604d8401959095526001600160c01b031960c092831b8116606d850152911b1660758201528251808203605d018152607d909101909252815191012090565b6000610a226103028860200151610cc3565b63ffffffff1690506000610a3c6103028960200151610cc3565b9050600263ffffffff821610610a5457876002610366565b602080880151516001600160401b031690610a709084906123c8565b1180610a855750610a826020836123f1565b15155b15610a9257876002610366565b6000610a9f602084612405565b9050600080610aba6040518060200160405280606081525090565b60208b0151610acc90858a8a87610d79565b9094509092509050618010610ae460208b018b612364565b61ffff1603610b2857610b1a848b600001518763ffffffff1660028110610b0d57610b0d612419565b6020020151839190610e99565b60208c015160400152610ba8565b618011610b3860208b018b612364565b61ffff1603610b66578951829063ffffffff871660028110610b5c57610b5c612419565b6020020152610ba8565b60405162461bcd60e51b81526020600482015260176024820152764241445f474c4f42414c5f53544154455f4f50434f444560481b6044820152606401610180565b505050505050505050505050565b6000610bc86103028460200151610cc3565b9050600263ffffffff821610610be057505060029052565b610c48610c3d83602001518363ffffffff1660028110610c0257610c02612419565b602002015160408051808201909152600080825260208201525060408051808201909152600181526001600160401b03909116602082015290565b602085015190610f4b565b505050565b6000610c5f6105d98460200151610cc3565b90506000610c736103028560200151610cc3565b9050600263ffffffff821610610c8d575050600290915250565b8183602001518263ffffffff1660028110610caa57610caa612419565b6001600160401b03909216602092909202015250505050565b60408051808201909152600080825260208201528151610ce2906115c5565b92915050565b60208101516000908183516006811115610d0457610d04611bf3565b14610d3b5760405162461bcd60e51b81526020600482015260076024820152662727aa2fa4999960c91b6044820152606401610180565b6401000000008110610ce25760405162461bcd60e51b81526020600482015260076024820152662120a22fa4999960c91b6044820152606401610180565b600080610d926040518060200160405280606081525090565b839150610da0868684611542565b9093509150610db08686846116d5565b925090506000610dc1828986610e99565b905088604001518114610e075760405162461bcd60e51b815260206004820152600e60248201526d15d493d391d7d3515357d493d3d560921b6044820152606401610180565b50955095509592505050565b600060208310610e5d5760405162461bcd60e51b81526020600482015260156024820152740848288bea68aa8be988a828cbe84b2a88abe9288b605b1b6044820152606401610180565b600083610e6c60016020612458565b610e769190612458565b610e819060086124ad565b60ff848116821b911b198616179150505b9392505050565b6040516b26b2b6b7b93c903632b0b31d60a11b6020820152602c81018290526000908190604c01604051602081830303815290604052805190602001209050610f0f8585836040518060400160405280601381526020017226b2b6b7b93c9036b2b935b632903a3932b29d60691b8152506117af565b95945050505050565b604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b8151610f579082611884565b5050565b6020810151600090600183516006811115610f7857610f78611bf3565b14610faf5760405162461bcd60e51b81526020600482015260076024820152661393d517d24d8d60ca1b6044820152606401610180565b600160401b8110610ce25760405162461bcd60e51b815260206004820152600760248201526610905117d24d8d60ca1b6044820152606401610180565b600060288210156110345760405162461bcd60e51b81526020600482015260126024820152712120a22fa9a2a8a4a72127ac2fa82927a7a360711b6044820152606401610180565b60006110428484602061155e565b508091505060008484604051611059929190612448565b60405190819003902090506000806001600160401b0388161561110a5761108660408a0160208b016124c4565b6001600160a01b03166316bf557961109f60018b6124ed565b6040516001600160e01b031960e084901b1681526001600160401b039091166004820152602401602060405180830381865afa1580156110e3573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906111079190612514565b91505b6001600160401b038416156111ad5761112960408a0160208b016124c4565b6001600160a01b031663d5719dc26111426001876124ed565b6040516001600160e01b031960e084901b1681526001600160401b039091166004820152602401602060405180830381865afa158015611186573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906111aa9190612514565b90505b6040805160208101849052908101849052606081018290526000906080016040516020818303038152906040528051906020012090508960200160208101906111f691906124c4565b6040516316bf557960e01b81526001600160401b038b1660048201526001600160a01b0391909116906316bf557990602401602060405180830381865afa158015611245573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906112699190612514565b81146112ae5760405162461bcd60e51b81526020600482015260146024820152734241445f534551494e424f585f4d45535341474560601b6044820152606401610180565b5060019998505050505050505050565b600060718210156113055760405162461bcd60e51b81526020600482015260116024820152702120a22fa222a620aca2a22fa82927a7a360791b6044820152606401610180565b60006001600160401b038516156113aa5761132660408701602088016124c4565b6001600160a01b031663d5719dc261133f6001886124ed565b6040516001600160e01b031960e084901b1681526001600160401b039091166004820152602401602060405180830381865afa158015611383573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906113a79190612514565b90505b60006113b98460718188612388565b6040516113c7929190612448565b604051809103902090506000858560008181106113e6576113e6612419565b9050013560f81c60f81b9050600061140087876001611977565b50905060008282611415607160218b8d612388565b8760405160200161142a95949392919061252d565b60408051601f1981840301815282825280516020918201208382018990528383018190528251808503840181526060909401909252825192019190912090915061147a60408c0160208d016124c4565b604051636ab8cee160e11b81526001600160401b038c1660048201526001600160a01b03919091169063d5719dc290602401602060405180830381865afa1580156114c9573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114ed9190612514565b81146115315760405162461bcd60e51b81526020600482015260136024820152724241445f44454c415945445f4d45535341474560681b6044820152606401610180565b5060019a9950505050505050505050565b60008181611551868684611977565b9097909650945050505050565b600081815b60088110156115bc576008836001600160401b0316901b925085858381811061158e5761158e612419565b919091013560f81c939093179250816115a68161242f565b92505080806115b49061242f565b915050611563565b50935093915050565b6040805180820190915260008082526020820152815180516115e990600190612458565b815181106115f9576115f9612419565b60200260200101519050600060018360000151516116179190612458565b6001600160401b0381111561162e5761162e611e4a565b60405190808252806020026020018201604052801561167357816020015b604080518082019091526000808252602082015281526020019060019003908161164c5790505b50905060005b81518110156116ce57835180518290811061169657611696612419565b60200260200101518282815181106116b0576116b0612419565b602002602001018190525080806116c69061242f565b915050611679565b5090915290565b6040805160208101909152606081528160006116f28686846119cc565b92509050600060ff82166001600160401b0381111561171357611713611e4a565b60405190808252806020026020018201604052801561173c578160200160208202803683370190505b50905060005b8260ff168160ff1610156117935761175b888886611542565b838360ff168151811061177057611770612419565b60200260200101819650828152505050808061178b9061248e565b915050611742565b5060405180602001604052808281525093505050935093915050565b8160005b85515181101561187b5784600116600003611817578282876000015183815181106117e0576117e0612419565b60200260200101516040516020016117fa9392919061256c565b604051602081830303815290604052805190602001209150611862565b828660000151828151811061182e5761182e612419565b6020026020010151836040516020016118499392919061256c565b6040516020818303038152906040528051906020012091505b60019490941c93806118738161242f565b9150506117b3565b50949350505050565b8151516000906118959060016123c8565b6001600160401b038111156118ac576118ac611e4a565b6040519080825280602002602001820160405280156118f157816020015b60408051808201909152600080825260208201528152602001906001900390816118ca5790505b50905060005b83515181101561194d57835180518290811061191557611915612419565b602002602001015182828151811061192f5761192f612419565b602002602001018190525080806119459061242f565b9150506118f7565b5081818460000151518151811061196657611966612419565b602090810291909101015290915250565b600081815b60208110156115bc57600883901b925085858381811061199e5761199e612419565b919091013560f81c939093179250816119b68161242f565b92505080806119c49061242f565b91505061197c565b6000818484828181106119e1576119e1612419565b919091013560f81c92508190506119f78161242f565b915050935093915050565b6040805161012081019091528060008152602001611a3760408051606080820183529181019182529081526000602082015290565b8152602001611a5d60408051606080820183529181019182529081526000602082015290565b8152602001611a82604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b611ab36125a3565b565b6040518060400160405280611ac8611ada565b8152602001611ad5611ada565b905290565b60405180604001604052806002906020820280368337509192915050565b600060408284031215611b0a57600080fd5b50919050565b6000806000806000808688036101a0811215611b2b57600080fd5b611b358989611af8565b965060408801356001600160401b0380821115611b5157600080fd5b90890190610120828c031215611b6657600080fd5b81975060e0605f1984011215611b7b57600080fd5b60608a019650611b8f8b6101408c01611af8565b95506101808a0135925080831115611ba657600080fd5b828a0192508a601f840112611bba57600080fd5b8235915080821115611bcb57600080fd5b50896020828401011115611bde57600080fd5b60208201935080925050509295509295509295565b634e487b7160e01b600052602160045260246000fd5b60048110611c1957611c19611bf3565b9052565b805160078110611c2f57611c2f611bf3565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b80841015611c8c57611c78828651611c1d565b938201936001939093019290850190611c65565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b82811015611d0b578451611cd7858251611c1d565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101611cc2565b509687015197909601969096525093949350505050565b6000610100808352611d378184018651611c09565b602085015161012084810152611d51610220850182611c3c565b9050604086015160ff198086840301610140870152611d708383611c3c565b925060608801519150808684030161016087015250611d8f8282611ca0565b915050608086015161018085015260a0860151611db56101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e086015250908501516102008401529050610e9260208301848051825260208101516001600160401b0380825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b604080519081016001600160401b0381118282101715611e8257611e82611e4a565b60405290565b604051602081016001600160401b0381118282101715611e8257611e82611e4a565b604051608081016001600160401b0381118282101715611e8257611e82611e4a565b60405161012081016001600160401b0381118282101715611e8257611e82611e4a565b60405160a081016001600160401b0381118282101715611e8257611e82611e4a565b604051606081016001600160401b0381118282101715611e8257611e82611e4a565b604051601f8201601f191681016001600160401b0381118282101715611f5b57611f5b611e4a565b604052919050565b803560048110611f7257600080fd5b919050565b60006001600160401b03821115611f9057611f90611e4a565b5060051b60200190565b600060408284031215611fac57600080fd5b611fb4611e60565b9050813560078110611fc557600080fd5b808252506020820135602082015292915050565b60006040808385031215611fec57600080fd5b611ff4611e60565b915082356001600160401b038082111561200d57600080fd5b8185019150602080838803121561202357600080fd5b61202b611e88565b83358381111561203a57600080fd5b80850194505087601f85011261204f57600080fd5b8335925061206461205f84611f77565b611f33565b83815260069390931b8401820192828101908985111561208357600080fd5b948301945b848610156120a95761209a8a87611f9a565b82529486019490830190612088565b8252508552948501359484019490945250909392505050565b803563ffffffff81168114611f7257600080fd5b600060408083850312156120e957600080fd5b6120f1611e60565b915082356001600160401b0381111561210957600080fd5b8301601f8101851361211a57600080fd5b8035602061212a61205f83611f77565b82815260a0928302840182019282820191908985111561214957600080fd5b948301945b848610156121b25780868b0312156121665760008081fd5b61216e611eaa565b6121788b88611f9a565b81528787013585820152606061218f8189016120c2565b8983015261219f608089016120c2565b908201528352948501949183019161214e565b50808752505080860135818601525050505092915050565b600061012082360312156121dd57600080fd5b6121e5611ecc565b6121ee83611f63565b815260208301356001600160401b038082111561220a57600080fd5b61221636838701611fd9565b6020840152604085013591508082111561222f57600080fd5b61223b36838701611fd9565b6040840152606085013591508082111561225457600080fd5b50612261368286016120d6565b6060830152506080830135608082015261227d60a084016120c2565b60a082015261228e60c084016120c2565b60c082015261229f60e084016120c2565b60e082015261010092830135928101929092525090565b80356001600160401b0381168114611f7257600080fd5b600081830360e08112156122e057600080fd5b6122e8611eef565b833581526060601f19830112156122fe57600080fd5b612306611f11565b9150612314602085016122b6565b8252612322604085016122b6565b6020830152606084013560408301528160208201526080840135604082015260a0840135606082015261235760c085016120c2565b6080820152949350505050565b60006020828403121561237657600080fd5b813561ffff81168114610e9257600080fd5b6000808585111561239857600080fd5b838611156123a557600080fd5b5050820193919092039150565b634e487b7160e01b600052601160045260246000fd5b80820180821115610ce257610ce26123b2565b634e487b7160e01b600052601260045260246000fd5b600082612400576124006123db565b500690565b600082612414576124146123db565b500490565b634e487b7160e01b600052603260045260246000fd5b600060018201612441576124416123b2565b5060010190565b8183823760009101908152919050565b81810381811115610ce257610ce26123b2565b600063ffffffff808316818103612484576124846123b2565b6001019392505050565b600060ff821660ff81036124a4576124a46123b2565b60010192915050565b8082028115828204841417610ce257610ce26123b2565b6000602082840312156124d657600080fd5b81356001600160a01b0381168114610e9257600080fd5b6001600160401b0382811682821603908082111561250d5761250d6123b2565b5092915050565b60006020828403121561252657600080fd5b5051919050565b6001600160f81b031986168152606085901b6bffffffffffffffffffffffff191660018201528284601583013760159201918201526035019392505050565b6000845160005b8181101561258d5760208188018101518583015201612573565b5091909101928352506020820152604001919050565b634e487b7160e01b600052605160045260246000fdfea264697066735822122030268aa8129c4508c78af093455ab2c4e0f27c548dfd3885c6d4ed1d2465569b64736f6c63430008110033",
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

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
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

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverHostIo *OneStepProverHostIoSession) ExecuteOneStep(execCtx ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverHostIo.Contract.ExecuteOneStep(&_OneStepProverHostIo.CallOpts, execCtx, startMach, startMod, inst, proof)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) execCtx, (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) view returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverHostIo *OneStepProverHostIoCallerSession) ExecuteOneStep(execCtx ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverHostIo.Contract.ExecuteOneStep(&_OneStepProverHostIo.CallOpts, execCtx, startMach, startMod, inst, proof)
}

// OneStepProverMathMetaData contains all meta data concerning the OneStepProverMath contract.
var OneStepProverMathMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"startMach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"startMod\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"opcode\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"argumentData\",\"type\":\"uint256\"}],\"internalType\":\"structInstruction\",\"name\":\"inst\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"mach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"mod\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061232c806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063da78e7d114610030575b600080fd5b61004361003e36600461189a565b61005a565b604051610051929190611aac565b60405180910390f35b6100626117cf565b6040805160a0810182526000808252825160608082018552828252602080830184905282860184905284019190915292820181905291810182905260808101919091526100ae87611f4f565b91506100bf36879003870187612052565b905060006100d060208701876120e9565b905061187861ffff8216604514806100ec575061ffff82166050145b156100fa57506103096102eb565b604661ffff831610801590610122575061011660096046612123565b61ffff168261ffff1611155b15610130575061041f6102eb565b606761ffff831610801590610158575061014c60026067612123565b61ffff168261ffff1611155b1561016657506105026102eb565b606a61ffff8316108015906101805750607861ffff831611155b1561018e575061056a6102eb565b605161ffff8316108015906101b657506101aa60096051612123565b61ffff168261ffff1611155b156101c457506107536102eb565b607961ffff8316108015906101ec57506101e060026079612123565b61ffff168261ffff1611155b156101fa57506107b86102eb565b607c61ffff8316108015906102145750608a61ffff831611155b15610222575061080b6102eb565b60a61961ffff83160161023857506109c26102eb565b61ffff821660ac148061024f575061ffff821660ad145b1561025d57506109e36102eb565b60c061ffff831610801590610277575060c461ffff831611155b156102855750610a366102eb565b60bc61ffff83161080159061029f575060bf61ffff831611155b156102ad5750610c456102eb565b60405162461bcd60e51b815260206004820152600e60248201526d494e56414c49445f4f50434f444560901b60448201526064015b60405180910390fd5b6102fc84848989898663ffffffff16565b5050965096945050505050565b60006103188660200151610dcc565b9050604561032960208601866120e9565b61ffff1603610369576000815160068111156103475761034761197d565b146103645760405162461bcd60e51b81526004016102e290612145565b6103e5565b605061037860208601866120e9565b61ffff16036103b3576001815160068111156103965761039661197d565b146103645760405162461bcd60e51b81526004016102e290612166565b60405162461bcd60e51b81526020600482015260076024820152662120a22fa2a8ad60c91b60448201526064016102e2565b600081602001516000036103fb575060016103ff565b5060005b61041661040b82610df1565b602089015190610e24565b50505050505050565b60006104366104318760200151610dcc565b610e34565b9050600061044a6104318860200151610dcc565b90506000604661045d60208801886120e9565b6104679190612187565b905060008061ffff831660021480610483575061ffff83166004145b80610492575061ffff83166006145b806104a1575061ffff83166008145b156104c1576104af84610eab565b91506104ba85610eab565b90506104cf565b505063ffffffff8083169084165b60006104dc838386610ed7565b90506104f56104ea82611080565b60208d015190610e24565b5050505050505050505050565b60006105146104318760200151610dcc565b90506000606761052760208701876120e9565b6105319190612187565b905060006105478363ffffffff168360206110b3565b905061056061055582610df1565b60208a015190610e24565b5050505050505050565b600061057c6104318760200151610dcc565b905060006105906104318860200151610dcc565b9050600080606a6105a460208901896120e9565b6105ae9190612187565b90508061ffff1660030361062b5763ffffffff841615806105e557508260030b637fffffff191480156105e557508360030b600019145b1561060e578860025b908160038111156106015761060161197d565b815250505050505061074c565b8360030b8360030b81610623576106236121a2565b059150610730565b8061ffff1660050361066a578363ffffffff1660000361064d578860026105ee565b8360030b8360030b81610662576106626121a2565b079150610730565b8061ffff16600a036106895763ffffffff8316601f85161b9150610730565b8061ffff16600c036106a85763ffffffff8316601f85161c9150610730565b8061ffff16600b036106c557600383900b601f85161d9150610730565b8061ffff16600d036106e2576106db8385611277565b9150610730565b8061ffff16600e036106f8576106db83856112b9565b6000806107128563ffffffff168763ffffffff16856112fb565b91509150801561072c575050600289525061074c92505050565b5091505b61074761073c83610df1565b60208b015190610e24565b505050505b5050505050565b600061076a6107658760200151610dcc565b611483565b9050600061077e6107658860200151610dcc565b90506000605161079160208801886120e9565b61079b9190612187565b905060006107aa838584610ed7565b905061074761073c82611080565b60006107ca6107658760200151610dcc565b9050600060796107dd60208701876120e9565b6107e79190612187565b905060006107f7838360406110b3565b63ffffffff169050610560610555826114fa565b600061081d6107658760200151610dcc565b905060006108316107658860200151610dcc565b9050600080607c61084560208901896120e9565b61084f9190612187565b90508061ffff166003036108b7576001600160401b038416158061088d57508260070b677fffffffffffffff1914801561088d57508360070b600019145b1561089a578860026105ee565b8360070b8360070b816108af576108af6121a2565b0591506109b6565b8061ffff166005036108f957836001600160401b03166000036108dc578860026105ee565b8360070b8360070b816108f1576108f16121a2565b0791506109b6565b8061ffff16600a0361091b576001600160401b038316603f85161b91506109b6565b8061ffff16600c0361093d576001600160401b038316603f85161c91506109b6565b8061ffff16600b0361095a57600783900b603f85161d91506109b6565b8061ffff16600d03610977576109708385611530565b91506109b6565b8061ffff16600e0361098d57610970838561157e565b600061099a8486846112fb565b909350905080156109b4575050600288525061074c915050565b505b61074761073c836114fa565b60006109d46107658760200151610dcc565b90508061041661040b82610df1565b60006109f56104318760200151610dcc565b9050600060ac610a0860208701876120e9565b61ffff1603610a2157610a1a82610eab565b9050610a2a565b5063ffffffff81165b61041661040b826114fa565b60008060c0610a4860208701876120e9565b61ffff1603610a5d5750600090506008610b30565b60c1610a6c60208701876120e9565b61ffff1603610a815750600090506010610b30565b60c2610a9060208701876120e9565b61ffff1603610aa55750600190506008610b30565b60c3610ab460208701876120e9565b61ffff1603610ac95750600190506010610b30565b60c4610ad860208701876120e9565b61ffff1603610aed5750600190506020610b30565b60405162461bcd60e51b8152602060048201526018602482015277494e56414c49445f455854454e445f53414d455f5459504560401b60448201526064016102e2565b600080836006811115610b4557610b4561197d565b03610b55575063ffffffff610b5f565b506001600160401b035b6000610b6e8960200151610dcc565b9050836006811115610b8257610b8261197d565b81516006811115610b9557610b9561197d565b14610bde5760405162461bcd60e51b81526020600482015260196024820152784241445f455854454e445f53414d455f545950455f5459504560381b60448201526064016102e2565b6000610bf1600160ff861681901b6121b8565b602083018051821690529050610c086001856121cb565b60ff166001901b826020015116600014610c2a57602082018051821985161790525b60208a0151610c399083610e24565b50505050505050505050565b60008060bc610c5760208701876120e9565b61ffff1603610c6c5750600090506002610d16565b60bd610c7b60208701876120e9565b61ffff1603610c905750600190506003610d16565b60be610c9f60208701876120e9565b61ffff1603610cb45750600290506000610d16565b60bf610cc360208701876120e9565b61ffff1603610cd85750600390506001610d16565b60405162461bcd60e51b81526020600482015260136024820152721253959053125117d491525395115494149155606a1b60448201526064016102e2565b6000610d258860200151610dcc565b9050816006811115610d3957610d3961197d565b81516006811115610d4c57610d4c61197d565b14610d945760405162461bcd60e51b8152602060048201526018602482015277494e56414c49445f5245494e544552505245545f5459504560401b60448201526064016102e2565b80836006811115610da757610da761197d565b90816006811115610dba57610dba61197d565b90525060208801516105609082610e24565b60408051808201909152600080825260208201528151610deb906115cc565b92915050565b604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b8151610e3090826116dc565b5050565b60208101516000908183516006811115610e5057610e5061197d565b14610e6d5760405162461bcd60e51b81526004016102e290612145565b6401000000008110610deb5760405162461bcd60e51b81526020600482015260076024820152662120a22fa4999960c91b60448201526064016102e2565b60006380000000821615610ecd575063ffffffff1667ffffffff000000001790565b5063ffffffff1690565b600061ffff8216610efe57826001600160401b0316846001600160401b0316149050611079565b60001961ffff831601610f2857826001600160401b0316846001600160401b031614159050611079565b60011961ffff831601610f45578260070b8460070b129050611079565b60021961ffff831601610f6e57826001600160401b0316846001600160401b0316109050611079565b60031961ffff831601610f8b578260070b8460070b139050611079565b60041961ffff831601610fb457826001600160401b0316846001600160401b0316119050611079565b60051961ffff831601610fd2578260070b8460070b13159050611079565b60061961ffff831601610ffc57826001600160401b0316846001600160401b031611159050611079565b60071961ffff83160161101a578260070b8460070b12159050611079565b60081961ffff83160161104457826001600160401b0316846001600160401b031610159050611079565b60405162461bcd60e51b815260206004820152600a6024820152690424144204952454c4f560b41b60448201526064016102e2565b9392505050565b604080518082019091526000808252602082015281156110a457610deb6001610df1565b610deb6000610df1565b919050565b60008161ffff16602014806110cc57508161ffff166040145b6111135760405162461bcd60e51b8152602060048201526018602482015277057524f4e4720555345204f462067656e65726963556e4f760441b60448201526064016102e2565b61ffff83166111845761ffff82165b60008163ffffffff16118015611157575061113e6001826121e4565b63ffffffff166001901b856001600160401b0316166000145b1561116e576111676001826121e4565b9050611122565b61117c8161ffff85166121e4565b915050611079565b60001961ffff8416016111dd5760005b8261ffff168163ffffffff161080156111bf5750600163ffffffff82161b85166001600160401b0316155b156111d6576111cf600182612201565b9050611194565b9050611079565b60011961ffff841601611243576000805b8361ffff168263ffffffff16101561123a57600163ffffffff83161b86166001600160401b03161561122857611225600182612201565b90505b816112328161221e565b9250506111ee565b91506110799050565b60405162461bcd60e51b815260206004820152600960248201526804241442049556e4f760bc1b60448201526064016102e2565b6000611284602083612241565b91506112918260206121e4565b63ffffffff168363ffffffff16901c8263ffffffff168463ffffffff16901b17905092915050565b60006112c6602083612241565b91506112d38260206121e4565b63ffffffff168363ffffffff16901b8263ffffffff168463ffffffff16901c17905092915050565b6000808261ffff16600003611316575050828201600061147b565b8261ffff1660010361132e575050818303600061147b565b8261ffff16600203611346575050828202600061147b565b8261ffff1660040361139c57836001600160401b031660000361136f575060009050600161147b565b836001600160401b0316856001600160401b031681611390576113906121a2565b0460009150915061147b565b8261ffff166006036113f257836001600160401b03166000036113c5575060009050600161147b565b836001600160401b0316856001600160401b0316816113e6576113e66121a2565b0660009150915061147b565b8261ffff1660070361140a575050828216600061147b565b8261ffff16600803611422575050828217600061147b565b8261ffff1660090361143a575050828218600061147b565b60405162461bcd60e51b81526020600482015260166024820152750494e56414c49445f47454e455249435f42494e5f4f560541b60448201526064016102e2565b935093915050565b60208101516000906001835160068111156114a0576114a061197d565b146114bd5760405162461bcd60e51b81526004016102e290612166565b600160401b8110610deb5760405162461bcd60e51b815260206004820152600760248201526610905117d24d8d60ca1b60448201526064016102e2565b60408051808201909152600080825260208201525060408051808201909152600181526001600160401b03909116602082015290565b600061153d604083612264565b915061154a82604061227e565b6001600160401b0316836001600160401b0316901c826001600160401b0316846001600160401b0316901b17905092915050565b600061158b604083612264565b915061159882604061227e565b6001600160401b0316836001600160401b0316901b826001600160401b0316846001600160401b0316901c17905092915050565b6040805180820190915260008082526020820152815180516115f0906001906121b8565b815181106116005761160061229e565b602002602001015190506000600183600001515161161e91906121b8565b6001600160401b0381111561163557611635611bd4565b60405190808252806020026020018201604052801561167a57816020015b60408051808201909152600080825260208201528152602001906001900390816116535790505b50905060005b81518110156116d557835180518290811061169d5761169d61229e565b60200260200101518282815181106116b7576116b761229e565b602002602001018190525080806116cd906122b4565b915050611680565b5090915290565b8151516000906116ed9060016122cd565b6001600160401b0381111561170457611704611bd4565b60405190808252806020026020018201604052801561174957816020015b60408051808201909152600080825260208201528152602001906001900390816117225790505b50905060005b8351518110156117a557835180518290811061176d5761176d61229e565b60200260200101518282815181106117875761178761229e565b6020026020010181905250808061179d906122b4565b91505061174f565b508181846000015151815181106117be576117be61229e565b602090810291909101015290915250565b604080516101208101909152806000815260200161180460408051606080820183529181019182529081526000602082015290565b815260200161182a60408051606080820183529181019182529081526000602082015290565b815260200161184f604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b6118806122e0565b565b60006040828403121561189457600080fd5b50919050565b6000806000806000808688036101a08112156118b557600080fd5b6118bf8989611882565b965060408801356001600160401b03808211156118db57600080fd5b90890190610120828c0312156118f057600080fd5b81975060e0605f198401121561190557600080fd5b60608a0196506119198b6101408c01611882565b95506101808a013592508083111561193057600080fd5b828a0192508a601f84011261194457600080fd5b823591508082111561195557600080fd5b5089602082840101111561196857600080fd5b60208201935080925050509295509295509295565b634e487b7160e01b600052602160045260246000fd5b600481106119a3576119a361197d565b9052565b8051600781106119b9576119b961197d565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b80841015611a1657611a028286516119a7565b9382019360019390930192908501906119ef565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b82811015611a95578451611a618582516119a7565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a090930192600101611a4c565b509687015197909601969096525093949350505050565b6000610100808352611ac18184018651611993565b602085015161012084810152611adb6102208501826119c6565b9050604086015160ff198086840301610140870152611afa83836119c6565b925060608801519150808684030161016087015250611b198282611a2a565b915050608086015161018085015260a0860151611b3f6101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e08601525090850151610200840152905061107960208301848051825260208101516001600160401b0380825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b604080519081016001600160401b0381118282101715611c0c57611c0c611bd4565b60405290565b604051602081016001600160401b0381118282101715611c0c57611c0c611bd4565b604051608081016001600160401b0381118282101715611c0c57611c0c611bd4565b60405161012081016001600160401b0381118282101715611c0c57611c0c611bd4565b60405160a081016001600160401b0381118282101715611c0c57611c0c611bd4565b604051606081016001600160401b0381118282101715611c0c57611c0c611bd4565b604051601f8201601f191681016001600160401b0381118282101715611ce557611ce5611bd4565b604052919050565b8035600481106110ae57600080fd5b60006001600160401b03821115611d1557611d15611bd4565b5060051b60200190565b600060408284031215611d3157600080fd5b611d39611bea565b9050813560078110611d4a57600080fd5b808252506020820135602082015292915050565b60006040808385031215611d7157600080fd5b611d79611bea565b915082356001600160401b0380821115611d9257600080fd5b81850191506020808388031215611da857600080fd5b611db0611c12565b833583811115611dbf57600080fd5b80850194505087601f850112611dd457600080fd5b83359250611de9611de484611cfc565b611cbd565b83815260069390931b84018201928281019089851115611e0857600080fd5b948301945b84861015611e2e57611e1f8a87611d1f565b82529486019490830190611e0d565b8252508552948501359484019490945250909392505050565b803563ffffffff811681146110ae57600080fd5b60006040808385031215611e6e57600080fd5b611e76611bea565b915082356001600160401b03811115611e8e57600080fd5b8301601f81018513611e9f57600080fd5b80356020611eaf611de483611cfc565b82815260a09283028401820192828201919089851115611ece57600080fd5b948301945b84861015611f375780868b031215611eeb5760008081fd5b611ef3611c34565b611efd8b88611d1f565b815287870135858201526060611f14818901611e47565b89830152611f2460808901611e47565b9082015283529485019491830191611ed3565b50808752505080860135818601525050505092915050565b60006101208236031215611f6257600080fd5b611f6a611c56565b611f7383611ced565b815260208301356001600160401b0380821115611f8f57600080fd5b611f9b36838701611d5e565b60208401526040850135915080821115611fb457600080fd5b611fc036838701611d5e565b60408401526060850135915080821115611fd957600080fd5b50611fe636828601611e5b565b6060830152506080830135608082015261200260a08401611e47565b60a082015261201360c08401611e47565b60c082015261202460e08401611e47565b60e082015261010092830135928101929092525090565b80356001600160401b03811681146110ae57600080fd5b600081830360e081121561206557600080fd5b61206d611c79565b833581526060601f198301121561208357600080fd5b61208b611c9b565b91506120996020850161203b565b82526120a76040850161203b565b6020830152606084013560408301528160208201526080840135604082015260a084013560608201526120dc60c08501611e47565b6080820152949350505050565b6000602082840312156120fb57600080fd5b813561ffff8116811461107957600080fd5b634e487b7160e01b600052601160045260246000fd5b61ffff81811683821601908082111561213e5761213e61210d565b5092915050565b6020808252600790820152662727aa2fa4999960c91b604082015260600190565b6020808252600790820152661393d517d24d8d60ca1b604082015260600190565b61ffff82811682821603908082111561213e5761213e61210d565b634e487b7160e01b600052601260045260246000fd5b81810381811115610deb57610deb61210d565b60ff8281168282160390811115610deb57610deb61210d565b63ffffffff82811682821603908082111561213e5761213e61210d565b63ffffffff81811683821601908082111561213e5761213e61210d565b600063ffffffff8083168181036122375761223761210d565b6001019392505050565b600063ffffffff80841680612258576122586121a2565b92169190910692915050565b60006001600160401b0380841680612258576122586121a2565b6001600160401b0382811682821603908082111561213e5761213e61210d565b634e487b7160e01b600052603260045260246000fd5b6000600182016122c6576122c661210d565b5060010190565b80820180821115610deb57610deb61210d565b634e487b7160e01b600052605160045260246000fdfea2646970667358221220e5724c85ee283d4c928e74dd7c58a32d563deaf62530db3b80400b073185cf3064736f6c63430008110033",
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

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
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

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverMath *OneStepProverMathSession) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverMath.Contract.ExecuteOneStep(&_OneStepProverMath.CallOpts, arg0, startMach, startMod, inst, proof)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverMath *OneStepProverMathCallerSession) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverMath.Contract.ExecuteOneStep(&_OneStepProverMath.CallOpts, arg0, startMach, startMod, inst, proof)
}

// OneStepProverMemoryMetaData contains all meta data concerning the OneStepProverMemory contract.
var OneStepProverMemoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"startMach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"startMod\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"opcode\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"argumentData\",\"type\":\"uint256\"}],\"internalType\":\"structInstruction\",\"name\":\"inst\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"components\":[{\"internalType\":\"enumMachineStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"valueStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue[]\",\"name\":\"inner\",\"type\":\"tuple[]\"}],\"internalType\":\"structValueArray\",\"name\":\"proved\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structValueStack\",\"name\":\"internalStack\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"components\":[{\"internalType\":\"enumValueType\",\"name\":\"valueType\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"contents\",\"type\":\"uint256\"}],\"internalType\":\"structValue\",\"name\":\"returnPc\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"localsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"callerModule\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"callerModuleInternals\",\"type\":\"uint32\"}],\"internalType\":\"structStackFrame[]\",\"name\":\"proved\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"remainingHash\",\"type\":\"bytes32\"}],\"internalType\":\"structStackFrameWindow\",\"name\":\"frameStack\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"globalStateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"moduleIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionIdx\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"functionPc\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"modulesRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structMachine\",\"name\":\"mach\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"globalsMerkleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"maxSize\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structModuleMemory\",\"name\":\"moduleMemory\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"tablesMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"functionsMerkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"internalsOffset\",\"type\":\"uint32\"}],\"internalType\":\"structModule\",\"name\":\"mod\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611dcb806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063da78e7d114610030575b600080fd5b61004361003e366004611379565b61005a565b60405161005192919061158b565b60405180910390f35b6100626112ae565b6040805160a0810182526000808252825160608082018552828252602080830184905282860184905284019190915292820181905291810182905260808101919091526100ae87611a33565b91506100bf36879003870187611b36565b905060006100d06020870187611bcd565b9050611357602861ffff8316108015906100ef5750603561ffff831611155b156100fd57506101b4610196565b603661ffff8316108015906101175750603e61ffff831611155b1561012557506106b1610196565b603e1961ffff83160161013b5750610a52610196565b603f1961ffff8316016101515750610a8a610196565b60405162461bcd60e51b8152602060048201526015602482015274494e56414c49445f4d454d4f52595f4f50434f444560581b60448201526064015b60405180910390fd5b6101a784848989898663ffffffff16565b5050965096945050505050565b6000808060286101c76020880188611bcd565b61ffff16036101df5750600091506004905081610427565b60296101ee6020880188611bcd565b61ffff1603610207575060019150600890506000610427565b602a6102166020880188611bcd565b61ffff160361022f575060029150600490506000610427565b602b61023e6020880188611bcd565b61ffff1603610257575060039150600890506000610427565b602c6102666020880188611bcd565b61ffff160361027e5750600091506001905080610427565b602d61028d6020880188611bcd565b61ffff16036102a55750600091506001905081610427565b602e6102b46020880188611bcd565b61ffff16036102cd575060009150600290506001610427565b602f6102dc6020880188611bcd565b61ffff16036102f45750600091506002905081610427565b60306103036020880188611bcd565b61ffff160361031a57506001915081905080610427565b60316103296020880188611bcd565b61ffff16036103415750600191508190506000610427565b60326103506020880188611bcd565b61ffff16036103685750600191506002905081610427565b60336103776020880188611bcd565b61ffff1603610390575060019150600290506000610427565b603461039f6020880188611bcd565b61ffff16036103b75750600191506004905081610427565b60356103c66020880188611bcd565b61ffff16036103df575060019150600490506000610427565b60405162461bcd60e51b815260206004820152601a60248201527f494e56414c49445f4d454d4f52595f4c4f41445f4f50434f4445000000000000604482015260640161018d565b600061043e6104398a60200151610b39565b610b5e565b6104529063ffffffff166020890135611c07565b6020890151519091506001600160401b031661046e8483611c07565b111561048257505060028752506106aa9050565b60006000198180805b8781101561052757600061049f8288611c07565b905060006104ae602083611c30565b90508581146104d2576104c88f60200151828f8f8b610bef565b5097509095509350845b60006104df602084611c44565b90506104ec846008611c58565b6001600160401b03166104ff8783610c89565b60ff166001600160401b0316901b85179450505050808061051f90611c6f565b91505061048b565b5085156106665786600114801561054f5750600088600681111561054d5761054d61145c565b145b15610565578060000b63ffffffff169050610666565b866001148015610586575060018860068111156105845761058461145c565b145b156105935760000b610666565b8660021480156105b4575060008860068111156105b2576105b261145c565b145b156105ca578060010b63ffffffff169050610666565b8660021480156105eb575060018860068111156105e9576105e961145c565b145b156105f85760010b610666565b866004148015610619575060018860068111156106175761061761145c565b145b156106265760030b610666565b60405162461bcd60e51b815260206004820152601560248201527410905117d491505117d096551154d7d4d251d39151605a1b604482015260640161018d565b6106a160405180604001604052808a60068111156106865761068661145c565b81526001600160401b0384166020918201528f015190610d03565b50505050505050505b5050505050565b6000808060366106c46020880188611bcd565b61ffff16036106d95750600491506000610840565b60376106e86020880188611bcd565b61ffff16036106fd5750600891506001610840565b603861070c6020880188611bcd565b61ffff16036107215750600491506002610840565b60396107306020880188611bcd565b61ffff16036107455750600891506003610840565b603a6107546020880188611bcd565b61ffff16036107695750600191506000610840565b603b6107786020880188611bcd565b61ffff160361078d5750600291506000610840565b603c61079c6020880188611bcd565b61ffff16036107b057506001915081610840565b603d6107bf6020880188611bcd565b61ffff16036107d45750600291506001610840565b603e6107e36020880188611bcd565b61ffff16036107f85750600491506001610840565b60405162461bcd60e51b815260206004820152601b60248201527f494e56414c49445f4d454d4f52595f53544f52455f4f50434f44450000000000604482015260640161018d565b600061084f8960200151610b39565b90508160068111156108635761086361145c565b815160068111156108765761087661145c565b146108b45760405162461bcd60e51b815260206004820152600e60248201526d4241445f53544f52455f5459504560901b604482015260640161018d565b806020015192506008846001600160401b031610156108ff5760016108da856008611c88565b6001600160401b031660016001600160401b0316901b6108fa9190611cb3565b831692505b505060006109136104398960200151610b39565b6109279063ffffffff166020880135611c07565b90508660200151600001516001600160401b0316836001600160401b0316826109509190611c07565b111561096257505060028652506106aa565b604080516020810190915260608152600090600019906000805b876001600160401b0316811015610a2f5760006109998288611c07565b905060006109a8602083611c30565b90508581146109ed5760001986146109cf576109c5858786610d13565b60208f0151604001525b6109e08e60200151828e8e8b610bef565b9098509196509094509250845b60006109fa602084611c44565b9050610a0785828c610d94565b945060088a6001600160401b0316901c99505050508080610a2790611c6f565b91505061097c565b50610a3b828483610d13565b60208c015160400152505050505050505050505050565b602084015151600090610a69906201000090611cda565b9050610a82610a7782610e19565b602088015190610d03565b505050505050565b602084015151600090610aa1906201000090611cda565b90506000610ab56104398860200151610b39565b90506000610acc63ffffffff808416908516611c07565b90508660200151602001516001600160401b03168111610b2157610af36201000082611c58565b60208801516001600160401b039091169052610b1c610b1184610e19565b60208a015190610d03565b610b2f565b610b2f610b11600019610e19565b5050505050505050565b60408051808201909152600080825260208201528151610b5890610e4c565b92915050565b60208101516000908183516006811115610b7a57610b7a61145c565b14610bb15760405162461bcd60e51b81526020600482015260076024820152662727aa2fa4999960c91b604482015260640161018d565b6401000000008110610b585760405162461bcd60e51b81526020600482015260076024820152662120a22fa4999960c91b604482015260640161018d565b600080610c086040518060200160405280606081525090565b839150610c16868684610f5c565b9093509150610c26868684610f78565b925090506000610c37828986610d13565b905088604001518114610c7d5760405162461bcd60e51b815260206004820152600e60248201526d15d493d391d7d3515357d493d3d560921b604482015260640161018d565b50955095509592505050565b600060208210610cd45760405162461bcd60e51b81526020600482015260166024820152750848288bea0aa9898be988a828cbe84b2a88abe9288b60531b604482015260640161018d565b600082610ce360016020611d00565b610ced9190611d00565b610cf8906008611c58565b9390931c9392505050565b8151610d0f9082611052565b5050565b6040516b26b2b6b7b93c903632b0b31d60a11b6020820152602c81018290526000908190604c01604051602081830303815290604052805190602001209050610d898585836040518060400160405280601381526020017226b2b6b7b93c9036b2b935b632903a3932b29d60691b815250611145565b9150505b9392505050565b600060208310610dde5760405162461bcd60e51b81526020600482015260156024820152740848288bea68aa8be988a828cbe84b2a88abe9288b605b1b604482015260640161018d565b600083610ded60016020611d00565b610df79190611d00565b610e02906008611c58565b60ff848116821b911b198616179150509392505050565b604080518082019091526000808252602082015250604080518082019091526000815263ffffffff909116602082015290565b604080518082019091526000808252602082015281518051610e7090600190611d00565b81518110610e8057610e80611d13565b6020026020010151905060006001836000015151610e9e9190611d00565b6001600160401b03811115610eb557610eb56116b3565b604051908082528060200260200182016040528015610efa57816020015b6040805180820190915260008082526020820152815260200190600190039081610ed35790505b50905060005b8151811015610f55578351805182908110610f1d57610f1d611d13565b6020026020010151828281518110610f3757610f37611d13565b60200260200101819052508080610f4d90611c6f565b915050610f00565b5090915290565b60008181610f6b86868461121a565b9097909650945050505050565b604080516020810190915260608152816000610f95868684611278565b92509050600060ff82166001600160401b03811115610fb657610fb66116b3565b604051908082528060200260200182016040528015610fdf578160200160208202803683370190505b50905060005b8260ff168160ff16101561103657610ffe888886610f5c565b838360ff168151811061101357611013611d13565b60200260200101819650828152505050808061102e90611d29565b915050610fe5565b5060405180602001604052808281525093505050935093915050565b815151600090611063906001611c07565b6001600160401b0381111561107a5761107a6116b3565b6040519080825280602002602001820160405280156110bf57816020015b60408051808201909152600080825260208201528152602001906001900390816110985790505b50905060005b83515181101561111b5783518051829081106110e3576110e3611d13565b60200260200101518282815181106110fd576110fd611d13565b6020026020010181905250808061111390611c6f565b9150506110c5565b5081818460000151518151811061113457611134611d13565b602090810291909101015290915250565b8160005b85515181101561121157846001166000036111ad5782828760000151838151811061117657611176611d13565b602002602001015160405160200161119093929190611d48565b6040516020818303038152906040528051906020012091506111f8565b82866000015182815181106111c4576111c4611d13565b6020026020010151836040516020016111df93929190611d48565b6040516020818303038152906040528051906020012091505b60019490941c938061120981611c6f565b915050611149565b50949350505050565b600081815b602081101561126f57600883901b925085858381811061124157611241611d13565b919091013560f81c9390931792508161125981611c6f565b925050808061126790611c6f565b91505061121f565b50935093915050565b60008184848281811061128d5761128d611d13565b919091013560f81c92508190506112a381611c6f565b915050935093915050565b60408051610120810190915280600081526020016112e360408051606080820183529181019182529081526000602082015290565b815260200161130960408051606080820183529181019182529081526000602082015290565b815260200161132e604051806040016040528060608152602001600080191681525090565b815260006020820181905260408201819052606082018190526080820181905260a09091015290565b61135f611d7f565b565b60006040828403121561137357600080fd5b50919050565b6000806000806000808688036101a081121561139457600080fd5b61139e8989611361565b965060408801356001600160401b03808211156113ba57600080fd5b90890190610120828c0312156113cf57600080fd5b81975060e0605f19840112156113e457600080fd5b60608a0196506113f88b6101408c01611361565b95506101808a013592508083111561140f57600080fd5b828a0192508a601f84011261142357600080fd5b823591508082111561143457600080fd5b5089602082840101111561144757600080fd5b60208201935080925050509295509295509295565b634e487b7160e01b600052602160045260246000fd5b600481106114825761148261145c565b9052565b8051600781106114985761149861145c565b8252602090810151910152565b805160408084529051602084830181905281516060860181905260009392820191849160808801905b808410156114f5576114e1828651611486565b9382019360019390930192908501906114ce565b509581015196019590955250919392505050565b8051604080845281518482018190526000926060916020918201918388019190865b82811015611574578451611540858251611486565b80830151858901528781015163ffffffff90811688870152908701511660808501529381019360a09093019260010161152b565b509687015197909601969096525093949350505050565b60006101008083526115a08184018651611472565b6020850151610120848101526115ba6102208501826114a5565b9050604086015160ff1980868403016101408701526115d983836114a5565b9250606088015191508086840301610160870152506115f88282611509565b915050608086015161018085015260a086015161161e6101a086018263ffffffff169052565b5060c086015163ffffffff81166101c08601525060e086015163ffffffff81166101e086015250908501516102008401529050610d8d60208301848051825260208101516001600160401b0380825116602085015280602083015116604085015250604081015160608401525060408101516080830152606081015160a083015263ffffffff60808201511660c08301525050565b634e487b7160e01b600052604160045260246000fd5b604080519081016001600160401b03811182821017156116eb576116eb6116b3565b60405290565b604051602081016001600160401b03811182821017156116eb576116eb6116b3565b604051608081016001600160401b03811182821017156116eb576116eb6116b3565b60405161012081016001600160401b03811182821017156116eb576116eb6116b3565b60405160a081016001600160401b03811182821017156116eb576116eb6116b3565b604051606081016001600160401b03811182821017156116eb576116eb6116b3565b604051601f8201601f191681016001600160401b03811182821017156117c4576117c46116b3565b604052919050565b8035600481106117db57600080fd5b919050565b60006001600160401b038211156117f9576117f96116b3565b5060051b60200190565b60006040828403121561181557600080fd5b61181d6116c9565b905081356007811061182e57600080fd5b808252506020820135602082015292915050565b6000604080838503121561185557600080fd5b61185d6116c9565b915082356001600160401b038082111561187657600080fd5b8185019150602080838803121561188c57600080fd5b6118946116f1565b8335838111156118a357600080fd5b80850194505087601f8501126118b857600080fd5b833592506118cd6118c8846117e0565b61179c565b83815260069390931b840182019282810190898511156118ec57600080fd5b948301945b84861015611912576119038a87611803565b825294860194908301906118f1565b8252508552948501359484019490945250909392505050565b803563ffffffff811681146117db57600080fd5b6000604080838503121561195257600080fd5b61195a6116c9565b915082356001600160401b0381111561197257600080fd5b8301601f8101851361198357600080fd5b803560206119936118c8836117e0565b82815260a092830284018201928282019190898511156119b257600080fd5b948301945b84861015611a1b5780868b0312156119cf5760008081fd5b6119d7611713565b6119e18b88611803565b8152878701358582015260606119f881890161192b565b89830152611a086080890161192b565b90820152835294850194918301916119b7565b50808752505080860135818601525050505092915050565b60006101208236031215611a4657600080fd5b611a4e611735565b611a57836117cc565b815260208301356001600160401b0380821115611a7357600080fd5b611a7f36838701611842565b60208401526040850135915080821115611a9857600080fd5b611aa436838701611842565b60408401526060850135915080821115611abd57600080fd5b50611aca3682860161193f565b60608301525060808301356080820152611ae660a0840161192b565b60a0820152611af760c0840161192b565b60c0820152611b0860e0840161192b565b60e082015261010092830135928101929092525090565b80356001600160401b03811681146117db57600080fd5b600081830360e0811215611b4957600080fd5b611b51611758565b833581526060601f1983011215611b6757600080fd5b611b6f61177a565b9150611b7d60208501611b1f565b8252611b8b60408501611b1f565b6020830152606084013560408301528160208201526080840135604082015260a08401356060820152611bc060c0850161192b565b6080820152949350505050565b600060208284031215611bdf57600080fd5b813561ffff81168114610d8d57600080fd5b634e487b7160e01b600052601160045260246000fd5b80820180821115610b5857610b58611bf1565b634e487b7160e01b600052601260045260246000fd5b600082611c3f57611c3f611c1a565b500490565b600082611c5357611c53611c1a565b500690565b8082028115828204841417610b5857610b58611bf1565b600060018201611c8157611c81611bf1565b5060010190565b6001600160401b03818116838216028082169190828114611cab57611cab611bf1565b505092915050565b6001600160401b03828116828216039080821115611cd357611cd3611bf1565b5092915050565b60006001600160401b0380841680611cf457611cf4611c1a565b92169190910492915050565b81810381811115610b5857610b58611bf1565b634e487b7160e01b600052603260045260246000fd5b600060ff821660ff8103611d3f57611d3f611bf1565b60010192915050565b6000845160005b81811015611d695760208188018101518583015201611d4f565b5091909101928352506020820152604001919050565b634e487b7160e01b600052605160045260246000fdfea2646970667358221220cebbb206491573beaa332ef8289718507bb42151df558dd1ccc4fda001ea193a64736f6c63430008110033",
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

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
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

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverMemory *OneStepProverMemorySession) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverMemory.Contract.ExecuteOneStep(&_OneStepProverMemory.CallOpts, arg0, startMach, startMod, inst, proof)
}

// ExecuteOneStep is a free data retrieval call binding the contract method 0xda78e7d1.
//
// Solidity: function executeOneStep((uint256,address) , (uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) startMach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) startMod, (uint16,uint256) inst, bytes proof) pure returns((uint8,(((uint8,uint256)[]),bytes32),(((uint8,uint256)[]),bytes32),(((uint8,uint256),bytes32,uint32,uint32)[],bytes32),bytes32,uint32,uint32,uint32,bytes32) mach, (bytes32,(uint64,uint64,bytes32),bytes32,bytes32,uint32) mod)
func (_OneStepProverMemory *OneStepProverMemoryCallerSession) ExecuteOneStep(arg0 ExecutionContext, startMach Machine, startMod Module, inst Instruction, proof []byte) (struct {
	Mach Machine
	Mod  Module
}, error) {
	return _OneStepProverMemory.Contract.ExecuteOneStep(&_OneStepProverMemory.CallOpts, arg0, startMach, startMod, inst, proof)
}
