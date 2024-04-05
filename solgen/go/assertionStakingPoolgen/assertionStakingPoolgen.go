// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package assertionStakingPoolgen

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

// AssertionInputs is an auto generated low-level Go binding around an user-defined struct.
type AssertionInputs struct {
	BeforeStateData BeforeStateData
	BeforeState     AssertionState
	AfterState      AssertionState
}

// AssertionState is an auto generated low-level Go binding around an user-defined struct.
type AssertionState struct {
	GlobalState    GlobalState
	MachineStatus  uint8
	EndHistoryRoot [32]byte
}

// BeforeStateData is an auto generated low-level Go binding around an user-defined struct.
type BeforeStateData struct {
	PrevPrevAssertionHash [32]byte
	SequencerBatchAcc     [32]byte
	ConfigData            ConfigData
}

// ConfigData is an auto generated low-level Go binding around an user-defined struct.
type ConfigData struct {
	WasmModuleRoot      [32]byte
	RequiredStake       *big.Int
	ChallengeManager    common.Address
	ConfirmPeriodBlocks uint64
	NextInboxPosition   uint64
}

// GlobalState is an auto generated low-level Go binding around an user-defined struct.
type GlobalState struct {
	Bytes32Vals [2][32]byte
	U64Vals     [2]uint64
}

// AssertionStakingPoolMetaData contains all meta data concerning the AssertionStakingPool contract.
var AssertionStakingPoolMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollup\",\"type\":\"address\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"prevPrevAssertionHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"sequencerBatchAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"requiredStake\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"confirmPeriodBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nextInboxPosition\",\"type\":\"uint64\"}],\"internalType\":\"structConfigData\",\"name\":\"configData\",\"type\":\"tuple\"}],\"internalType\":\"structBeforeStateData\",\"name\":\"beforeStateData\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"beforeState\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"afterState\",\"type\":\"tuple\"}],\"internalType\":\"structAssertionInputs\",\"name\":\"_assertionInputs\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"_assertionHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"name\":\"AmountExceedsBalance\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NoBalanceToWithdraw\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"StakeDeposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"StakeWithdrawn\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"assertionHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"assertionInputs\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"prevPrevAssertionHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"sequencerBatchAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"requiredStake\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"confirmPeriodBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nextInboxPosition\",\"type\":\"uint64\"}],\"internalType\":\"structConfigData\",\"name\":\"configData\",\"type\":\"tuple\"}],\"internalType\":\"structBeforeStateData\",\"name\":\"beforeStateData\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"beforeState\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"afterState\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"createAssertion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"depositIntoPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"depositedTokenBalances\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getRequiredStake\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"makeStakeWithdrawable\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"makeStakeWithdrawableAndWithdrawBackIntoPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stakeToken\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawFromPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"withdrawFromPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawStakeBackIntoPool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60e06040523480156200001157600080fd5b50604051620017453803806200174583398101604081905262000034916200050a565b6001600160a01b03838116608090815260a0839052835180516000908155602080830151600155604092830151805160029081558183015160035593810151600480546060840151929098166001600160e01b031990981697909717600160a01b6001600160401b03928316021790965590930151600580546001600160401b03191691909516179093559084015180518051869493600692918391620000dd9183916200021f565b506020820151620000f5906002808401919062000262565b505050602082015160038201805460ff191660018360028111156200011e576200011e6200062b565b021790555060409182015160049091015582015180518051600b8401919082906200014d90829060026200021f565b50602082015162000165906002808401919062000262565b505050602082015160038201805460ff191660018360028111156200018e576200018e6200062b565b02179055506040820151816004015550509050506080516001600160a01b03166351ed6a306040518163ffffffff1660e01b8152600401602060405180830381865afa158015620001e3573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019062000209919062000641565b6001600160a01b031660c0525062000666915050565b826002810192821562000250579160200282015b828111156200025057825182559160200191906001019062000233565b506200025e92915062000304565b5090565b600183019183908215620002505791602002820160005b83821115620002c557835183826101000a8154816001600160401b0302191690836001600160401b03160217905550926020019260080160208160070104928301926001030262000279565b8015620002fa5782816101000a8154906001600160401b030219169055600801602081600701049283019260010302620002c5565b50506200025e9291505b5b808211156200025e576000815560010162000305565b80516001600160a01b03811681146200033357600080fd5b919050565b634e487b7160e01b600052604160045260246000fd5b604051606081016001600160401b038111828210171562000373576200037362000338565b60405290565b604080519081016001600160401b038111828210171562000373576200037362000338565b60405160a081016001600160401b038111828210171562000373576200037362000338565b80516001600160401b03811681146200033357600080fd5b600082601f830112620003ed57600080fd5b620003f762000379565b8060408401858111156200040a57600080fd5b845b818110156200042f576200042081620003c3565b8452602093840193016200040c565b509095945050505050565b8051600381106200033357600080fd5b600081830360c08112156200045e57600080fd5b620004686200034e565b915060808112156200047957600080fd5b506200048462000379565b83601f8401126200049457600080fd5b6200049e62000379565b806040850186811115620004b157600080fd5b855b81811015620004cd578051845260209384019301620004b3565b50818452620004dd8782620003db565b60208501525050508152620004f5608083016200043a565b602082015260a0820151604082015292915050565b60008060008385036102a08112156200052257600080fd5b6200052d856200031b565b9350601f1981016102608112156200054457600080fd5b6200054e6200034e565b60e08212156200055d57600080fd5b620005676200034e565b9150602087015182526040870151602083015260a0605f19840112156200058d57600080fd5b620005976200039e565b92506060870151835260808701516020840152620005b860a088016200031b565b6040840152620005cb60c08801620003c3565b6060840152620005de60e08801620003c3565b6080840152826040830152818152620005fc8861010089016200044a565b602082015262000611886101c089016200044a565b604082015280945050505061028084015190509250925092565b634e487b7160e01b600052602160045260246000fd5b6000602082840312156200065457600080fd5b6200065f826200031b565b9392505050565b60805160a05160c051611078620006cd60003960008181610148015281816102ec0152818161036e01526104b601526000818160e901526104170152600081816101e901528181610390015281816103e50152818161077901526107ee01526110786000f3fe608060405234801561001057600080fd5b50600436106100df5760003560e01c80637476083b1161008c578063930412af11610066578063930412af146101d45780639451944d146101dc578063cb23bcb5146101e4578063f0e978891461020b57600080fd5b80637476083b1461018a578063875b2af01461019d5780639252175b146101bd57600080fd5b80634b7a7538116100bd5780634b7a75381461013b57806351ed6a30146101435780636b74d5151461018257600080fd5b80632113ed21146100e457806326c0e5c51461011e57806330fc43ed14610128575b600080fd5b61010b7f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020015b60405180910390f35b610126610213565b005b610126610136366004610cd7565b61022e565b610126610354565b61016a7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b039091168152602001610115565b610126610474565b610126610198366004610cd7565b610484565b61010b6101ab366004610cf0565b60106020526000908152604090205481565b6101c5610516565b60405161011593929190610dd2565b610126610777565b6101266107ec565b61016a7f000000000000000000000000000000000000000000000000000000000000000081565b60035461010b565b3360009081526010602052604090205461022c9061022e565b565b336000908152601060205260408120549081900361027f576040517fe06b2da50000000000000000000000000000000000000000000000000000000081523360048201526024015b60405180910390fd5b808211156102c9576040517fa47b7c650000000000000000000000000000000000000000000000000000000081523360048201526024810183905260448101829052606401610276565b6102d38282610e68565b3360008181526010602052604090209190915561031b907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169084610873565b60405182815233907f8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc9060200160405180910390a25050565b600061035f60035490565b90506103b56001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000167f000000000000000000000000000000000000000000000000000000000000000083610921565b6040517f7300201c0000000000000000000000000000000000000000000000000000000081526001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001690637300201c9061043f9084906000907f000000000000000000000000000000000000000000000000000000000000000090600401610eee565b600060405180830381600087803b15801561045957600080fd5b505af115801561046d573d6000803e3d6000fd5b5050505050565b61047c610777565b61022c6107ec565b33600090815260106020526040812080548392906104a3908490610f81565b909155506104de90506001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016333084610a05565b60405181815233907f0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc29060200160405180910390a250565b6040805160608082018352600080548352600154602080850191909152845160a08082018752600280548352600354938301939093526004546001600160a01b0381168389015274010000000000000000000000000000000000000000900467ffffffffffffffff9081168387015260055416608083015285870191909152855160e08101968790529495929493600693859391840192859284929186019184919082845b8154815260200190600101908083116105bb575050509183525050604080518082019182905260209092019190600284810191826000855b82829054906101000a900467ffffffffffffffff1667ffffffffffffffff16815260200190600801906020826007010492830192600103820291508084116105f3579050505050919092525050508152600382015460209091019060ff16600281111561066257610662610d19565b600281111561067357610673610d19565b8152600491909101546020909101526040805160e08101909152909190600b82018160608101828160a084018260028282826020028201915b8154815260200190600101908083116106ac575050509183525050604080518082019182905260209092019190600284810191826000855b82829054906101000a900467ffffffffffffffff1667ffffffffffffffff16815260200190600801906020826007010492830192600103820291508084116106e4579050505050919092525050508152600382015460209091019060ff16600281111561075357610753610d19565b600281111561076457610764610d19565b8152602001600482015481525050905083565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03166357ef4ab96040518163ffffffff1660e01b8152600401600060405180830381600087803b1580156107d257600080fd5b505af11580156107e6573d6000803e3d6000fd5b50505050565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031663613739196040518163ffffffff1660e01b81526004016020604051808303816000875af115801561084c573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906108709190610f94565b50565b6040516001600160a01b03831660248201526044810182905261091c9084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff0000000000000000000000000000000000000000000000000000000090931692909217909152610a56565b505050565b6040517fdd62ed3e0000000000000000000000000000000000000000000000000000000081523060048201526001600160a01b038381166024830152600091839186169063dd62ed3e90604401602060405180830381865afa15801561098b573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906109af9190610f94565b6109b99190610f81565b6040516001600160a01b0385166024820152604481018290529091506107e69085907f095ea7b300000000000000000000000000000000000000000000000000000000906064016108b8565b6040516001600160a01b03808516602483015283166044820152606481018290526107e69085907f23b872dd00000000000000000000000000000000000000000000000000000000906084016108b8565b6000610aab826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564815250856001600160a01b0316610b3b9092919063ffffffff16565b80519091501561091c5780806020019051810190610ac99190610fad565b61091c5760405162461bcd60e51b815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f742073756363656564000000000000000000000000000000000000000000006064820152608401610276565b6060610b4a8484600085610b54565b90505b9392505050565b606082471015610bcc5760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c00000000000000000000000000000000000000000000000000006064820152608401610276565b6001600160a01b0385163b610c235760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e74726163740000006044820152606401610276565b600080866001600160a01b03168587604051610c3f9190610ff3565b60006040518083038185875af1925050503d8060008114610c7c576040519150601f19603f3d011682016040523d82523d6000602084013e610c81565b606091505b5091509150610c91828286610c9e565b925050505b949350505050565b60608315610cad575081610b4d565b825115610cbd5782518084602001fd5b8160405162461bcd60e51b8152600401610276919061100f565b600060208284031215610ce957600080fd5b5035919050565b600060208284031215610d0257600080fd5b81356001600160a01b0381168114610b4d57600080fd5b634e487b7160e01b600052602160045260246000fd5b60038110610d4d57634e487b7160e01b600052602160045260246000fd5b9052565b805180518360005b6002811015610d78578251825260209283019290910190600101610d59565b505050602090810151906040840160005b6002811015610db057835167ffffffffffffffff1682529282019290820190600101610d89565b50508201519050610dc46080840182610d2f565b506040015160a09190910152565b6000610260820190508451825260208501516020830152604085015180516040840152602081015160608401526001600160a01b036040820151166080840152606081015167ffffffffffffffff80821660a08601528060808401511660c0860152505050610e4460e0830185610d51565b610c966101a0830184610d51565b634e487b7160e01b600052601160045260246000fd5b81810381811115610e7b57610e7b610e52565b92915050565b818160005b6002811015610ea5578154835260209092019160019182019101610e86565b505050600281015467ffffffffffffffff8082166040850152808260401c166060850152505060ff600382015416610ee06080840182610d2f565b506004015460a09190910152565b8381528254602082015260018301546040820152600283015460608201526003830154608082015260048301546001600160a01b03811660a0808401919091521c67ffffffffffffffff90811660c083015260058401541660e08201526102a08101610f61610100830160068601610e81565b610f726101c08301600b8601610e81565b82610280830152949350505050565b80820180821115610e7b57610e7b610e52565b600060208284031215610fa657600080fd5b5051919050565b600060208284031215610fbf57600080fd5b81518015158114610b4d57600080fd5b60005b83811015610fea578181015183820152602001610fd2565b50506000910152565b60008251611005818460208701610fcf565b9190910192915050565b602081526000825180602084015261102e816040850160208701610fcf565b601f01601f1916919091016040019291505056fea26469706673582212209087fdad40ab4da26397456263ffcb9377057af7edb54d6f02cda01ffdd72c0e64736f6c63430008110033",
}

// AssertionStakingPoolABI is the input ABI used to generate the binding from.
// Deprecated: Use AssertionStakingPoolMetaData.ABI instead.
var AssertionStakingPoolABI = AssertionStakingPoolMetaData.ABI

// AssertionStakingPoolBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AssertionStakingPoolMetaData.Bin instead.
var AssertionStakingPoolBin = AssertionStakingPoolMetaData.Bin

// DeployAssertionStakingPool deploys a new Ethereum contract, binding an instance of AssertionStakingPool to it.
func DeployAssertionStakingPool(auth *bind.TransactOpts, backend bind.ContractBackend, _rollup common.Address, _assertionInputs AssertionInputs, _assertionHash [32]byte) (common.Address, *types.Transaction, *AssertionStakingPool, error) {
	parsed, err := AssertionStakingPoolMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AssertionStakingPoolBin), backend, _rollup, _assertionInputs, _assertionHash)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AssertionStakingPool{AssertionStakingPoolCaller: AssertionStakingPoolCaller{contract: contract}, AssertionStakingPoolTransactor: AssertionStakingPoolTransactor{contract: contract}, AssertionStakingPoolFilterer: AssertionStakingPoolFilterer{contract: contract}}, nil
}

// AssertionStakingPool is an auto generated Go binding around an Ethereum contract.
type AssertionStakingPool struct {
	AssertionStakingPoolCaller     // Read-only binding to the contract
	AssertionStakingPoolTransactor // Write-only binding to the contract
	AssertionStakingPoolFilterer   // Log filterer for contract events
}

// AssertionStakingPoolCaller is an auto generated read-only Go binding around an Ethereum contract.
type AssertionStakingPoolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AssertionStakingPoolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AssertionStakingPoolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AssertionStakingPoolSession struct {
	Contract     *AssertionStakingPool // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// AssertionStakingPoolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AssertionStakingPoolCallerSession struct {
	Contract *AssertionStakingPoolCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// AssertionStakingPoolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AssertionStakingPoolTransactorSession struct {
	Contract     *AssertionStakingPoolTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// AssertionStakingPoolRaw is an auto generated low-level Go binding around an Ethereum contract.
type AssertionStakingPoolRaw struct {
	Contract *AssertionStakingPool // Generic contract binding to access the raw methods on
}

// AssertionStakingPoolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AssertionStakingPoolCallerRaw struct {
	Contract *AssertionStakingPoolCaller // Generic read-only contract binding to access the raw methods on
}

// AssertionStakingPoolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AssertionStakingPoolTransactorRaw struct {
	Contract *AssertionStakingPoolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAssertionStakingPool creates a new instance of AssertionStakingPool, bound to a specific deployed contract.
func NewAssertionStakingPool(address common.Address, backend bind.ContractBackend) (*AssertionStakingPool, error) {
	contract, err := bindAssertionStakingPool(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPool{AssertionStakingPoolCaller: AssertionStakingPoolCaller{contract: contract}, AssertionStakingPoolTransactor: AssertionStakingPoolTransactor{contract: contract}, AssertionStakingPoolFilterer: AssertionStakingPoolFilterer{contract: contract}}, nil
}

// NewAssertionStakingPoolCaller creates a new read-only instance of AssertionStakingPool, bound to a specific deployed contract.
func NewAssertionStakingPoolCaller(address common.Address, caller bind.ContractCaller) (*AssertionStakingPoolCaller, error) {
	contract, err := bindAssertionStakingPool(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCaller{contract: contract}, nil
}

// NewAssertionStakingPoolTransactor creates a new write-only instance of AssertionStakingPool, bound to a specific deployed contract.
func NewAssertionStakingPoolTransactor(address common.Address, transactor bind.ContractTransactor) (*AssertionStakingPoolTransactor, error) {
	contract, err := bindAssertionStakingPool(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolTransactor{contract: contract}, nil
}

// NewAssertionStakingPoolFilterer creates a new log filterer instance of AssertionStakingPool, bound to a specific deployed contract.
func NewAssertionStakingPoolFilterer(address common.Address, filterer bind.ContractFilterer) (*AssertionStakingPoolFilterer, error) {
	contract, err := bindAssertionStakingPool(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolFilterer{contract: contract}, nil
}

// bindAssertionStakingPool binds a generic wrapper to an already deployed contract.
func bindAssertionStakingPool(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AssertionStakingPoolMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AssertionStakingPool *AssertionStakingPoolRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AssertionStakingPool.Contract.AssertionStakingPoolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AssertionStakingPool *AssertionStakingPoolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.AssertionStakingPoolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AssertionStakingPool *AssertionStakingPoolRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.AssertionStakingPoolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AssertionStakingPool *AssertionStakingPoolCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AssertionStakingPool.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AssertionStakingPool *AssertionStakingPoolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AssertionStakingPool *AssertionStakingPoolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.contract.Transact(opts, method, params...)
}

// AssertionHash is a free data retrieval call binding the contract method 0x2113ed21.
//
// Solidity: function assertionHash() view returns(bytes32)
func (_AssertionStakingPool *AssertionStakingPoolCaller) AssertionHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _AssertionStakingPool.contract.Call(opts, &out, "assertionHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// AssertionHash is a free data retrieval call binding the contract method 0x2113ed21.
//
// Solidity: function assertionHash() view returns(bytes32)
func (_AssertionStakingPool *AssertionStakingPoolSession) AssertionHash() ([32]byte, error) {
	return _AssertionStakingPool.Contract.AssertionHash(&_AssertionStakingPool.CallOpts)
}

// AssertionHash is a free data retrieval call binding the contract method 0x2113ed21.
//
// Solidity: function assertionHash() view returns(bytes32)
func (_AssertionStakingPool *AssertionStakingPoolCallerSession) AssertionHash() ([32]byte, error) {
	return _AssertionStakingPool.Contract.AssertionHash(&_AssertionStakingPool.CallOpts)
}

// AssertionInputs is a free data retrieval call binding the contract method 0x9252175b.
//
// Solidity: function assertionInputs() view returns((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)) beforeStateData, ((bytes32[2],uint64[2]),uint8,bytes32) beforeState, ((bytes32[2],uint64[2]),uint8,bytes32) afterState)
func (_AssertionStakingPool *AssertionStakingPoolCaller) AssertionInputs(opts *bind.CallOpts) (struct {
	BeforeStateData BeforeStateData
	BeforeState     AssertionState
	AfterState      AssertionState
}, error) {
	var out []interface{}
	err := _AssertionStakingPool.contract.Call(opts, &out, "assertionInputs")

	outstruct := new(struct {
		BeforeStateData BeforeStateData
		BeforeState     AssertionState
		AfterState      AssertionState
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BeforeStateData = *abi.ConvertType(out[0], new(BeforeStateData)).(*BeforeStateData)
	outstruct.BeforeState = *abi.ConvertType(out[1], new(AssertionState)).(*AssertionState)
	outstruct.AfterState = *abi.ConvertType(out[2], new(AssertionState)).(*AssertionState)

	return *outstruct, err

}

// AssertionInputs is a free data retrieval call binding the contract method 0x9252175b.
//
// Solidity: function assertionInputs() view returns((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)) beforeStateData, ((bytes32[2],uint64[2]),uint8,bytes32) beforeState, ((bytes32[2],uint64[2]),uint8,bytes32) afterState)
func (_AssertionStakingPool *AssertionStakingPoolSession) AssertionInputs() (struct {
	BeforeStateData BeforeStateData
	BeforeState     AssertionState
	AfterState      AssertionState
}, error) {
	return _AssertionStakingPool.Contract.AssertionInputs(&_AssertionStakingPool.CallOpts)
}

// AssertionInputs is a free data retrieval call binding the contract method 0x9252175b.
//
// Solidity: function assertionInputs() view returns((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)) beforeStateData, ((bytes32[2],uint64[2]),uint8,bytes32) beforeState, ((bytes32[2],uint64[2]),uint8,bytes32) afterState)
func (_AssertionStakingPool *AssertionStakingPoolCallerSession) AssertionInputs() (struct {
	BeforeStateData BeforeStateData
	BeforeState     AssertionState
	AfterState      AssertionState
}, error) {
	return _AssertionStakingPool.Contract.AssertionInputs(&_AssertionStakingPool.CallOpts)
}

// DepositedTokenBalances is a free data retrieval call binding the contract method 0x875b2af0.
//
// Solidity: function depositedTokenBalances(address ) view returns(uint256)
func (_AssertionStakingPool *AssertionStakingPoolCaller) DepositedTokenBalances(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _AssertionStakingPool.contract.Call(opts, &out, "depositedTokenBalances", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DepositedTokenBalances is a free data retrieval call binding the contract method 0x875b2af0.
//
// Solidity: function depositedTokenBalances(address ) view returns(uint256)
func (_AssertionStakingPool *AssertionStakingPoolSession) DepositedTokenBalances(arg0 common.Address) (*big.Int, error) {
	return _AssertionStakingPool.Contract.DepositedTokenBalances(&_AssertionStakingPool.CallOpts, arg0)
}

// DepositedTokenBalances is a free data retrieval call binding the contract method 0x875b2af0.
//
// Solidity: function depositedTokenBalances(address ) view returns(uint256)
func (_AssertionStakingPool *AssertionStakingPoolCallerSession) DepositedTokenBalances(arg0 common.Address) (*big.Int, error) {
	return _AssertionStakingPool.Contract.DepositedTokenBalances(&_AssertionStakingPool.CallOpts, arg0)
}

// GetRequiredStake is a free data retrieval call binding the contract method 0xf0e97889.
//
// Solidity: function getRequiredStake() view returns(uint256)
func (_AssertionStakingPool *AssertionStakingPoolCaller) GetRequiredStake(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AssertionStakingPool.contract.Call(opts, &out, "getRequiredStake")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRequiredStake is a free data retrieval call binding the contract method 0xf0e97889.
//
// Solidity: function getRequiredStake() view returns(uint256)
func (_AssertionStakingPool *AssertionStakingPoolSession) GetRequiredStake() (*big.Int, error) {
	return _AssertionStakingPool.Contract.GetRequiredStake(&_AssertionStakingPool.CallOpts)
}

// GetRequiredStake is a free data retrieval call binding the contract method 0xf0e97889.
//
// Solidity: function getRequiredStake() view returns(uint256)
func (_AssertionStakingPool *AssertionStakingPoolCallerSession) GetRequiredStake() (*big.Int, error) {
	return _AssertionStakingPool.Contract.GetRequiredStake(&_AssertionStakingPool.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AssertionStakingPool.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolSession) Rollup() (common.Address, error) {
	return _AssertionStakingPool.Contract.Rollup(&_AssertionStakingPool.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolCallerSession) Rollup() (common.Address, error) {
	return _AssertionStakingPool.Contract.Rollup(&_AssertionStakingPool.CallOpts)
}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolCaller) StakeToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AssertionStakingPool.contract.Call(opts, &out, "stakeToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolSession) StakeToken() (common.Address, error) {
	return _AssertionStakingPool.Contract.StakeToken(&_AssertionStakingPool.CallOpts)
}

// StakeToken is a free data retrieval call binding the contract method 0x51ed6a30.
//
// Solidity: function stakeToken() view returns(address)
func (_AssertionStakingPool *AssertionStakingPoolCallerSession) StakeToken() (common.Address, error) {
	return _AssertionStakingPool.Contract.StakeToken(&_AssertionStakingPool.CallOpts)
}

// CreateAssertion is a paid mutator transaction binding the contract method 0x4b7a7538.
//
// Solidity: function createAssertion() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) CreateAssertion(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "createAssertion")
}

// CreateAssertion is a paid mutator transaction binding the contract method 0x4b7a7538.
//
// Solidity: function createAssertion() returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) CreateAssertion() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.CreateAssertion(&_AssertionStakingPool.TransactOpts)
}

// CreateAssertion is a paid mutator transaction binding the contract method 0x4b7a7538.
//
// Solidity: function createAssertion() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) CreateAssertion() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.CreateAssertion(&_AssertionStakingPool.TransactOpts)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 _amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) DepositIntoPool(opts *bind.TransactOpts, _amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "depositIntoPool", _amount)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 _amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) DepositIntoPool(_amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.DepositIntoPool(&_AssertionStakingPool.TransactOpts, _amount)
}

// DepositIntoPool is a paid mutator transaction binding the contract method 0x7476083b.
//
// Solidity: function depositIntoPool(uint256 _amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) DepositIntoPool(_amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.DepositIntoPool(&_AssertionStakingPool.TransactOpts, _amount)
}

// MakeStakeWithdrawable is a paid mutator transaction binding the contract method 0x930412af.
//
// Solidity: function makeStakeWithdrawable() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) MakeStakeWithdrawable(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "makeStakeWithdrawable")
}

// MakeStakeWithdrawable is a paid mutator transaction binding the contract method 0x930412af.
//
// Solidity: function makeStakeWithdrawable() returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) MakeStakeWithdrawable() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.MakeStakeWithdrawable(&_AssertionStakingPool.TransactOpts)
}

// MakeStakeWithdrawable is a paid mutator transaction binding the contract method 0x930412af.
//
// Solidity: function makeStakeWithdrawable() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) MakeStakeWithdrawable() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.MakeStakeWithdrawable(&_AssertionStakingPool.TransactOpts)
}

// MakeStakeWithdrawableAndWithdrawBackIntoPool is a paid mutator transaction binding the contract method 0x6b74d515.
//
// Solidity: function makeStakeWithdrawableAndWithdrawBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) MakeStakeWithdrawableAndWithdrawBackIntoPool(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "makeStakeWithdrawableAndWithdrawBackIntoPool")
}

// MakeStakeWithdrawableAndWithdrawBackIntoPool is a paid mutator transaction binding the contract method 0x6b74d515.
//
// Solidity: function makeStakeWithdrawableAndWithdrawBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) MakeStakeWithdrawableAndWithdrawBackIntoPool() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.MakeStakeWithdrawableAndWithdrawBackIntoPool(&_AssertionStakingPool.TransactOpts)
}

// MakeStakeWithdrawableAndWithdrawBackIntoPool is a paid mutator transaction binding the contract method 0x6b74d515.
//
// Solidity: function makeStakeWithdrawableAndWithdrawBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) MakeStakeWithdrawableAndWithdrawBackIntoPool() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.MakeStakeWithdrawableAndWithdrawBackIntoPool(&_AssertionStakingPool.TransactOpts)
}

// WithdrawFromPool is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) WithdrawFromPool(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "withdrawFromPool")
}

// WithdrawFromPool is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) WithdrawFromPool() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawFromPool(&_AssertionStakingPool.TransactOpts)
}

// WithdrawFromPool is a paid mutator transaction binding the contract method 0x26c0e5c5.
//
// Solidity: function withdrawFromPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) WithdrawFromPool() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawFromPool(&_AssertionStakingPool.TransactOpts)
}

// WithdrawFromPool0 is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 _amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) WithdrawFromPool0(opts *bind.TransactOpts, _amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "withdrawFromPool0", _amount)
}

// WithdrawFromPool0 is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 _amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) WithdrawFromPool0(_amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawFromPool0(&_AssertionStakingPool.TransactOpts, _amount)
}

// WithdrawFromPool0 is a paid mutator transaction binding the contract method 0x30fc43ed.
//
// Solidity: function withdrawFromPool(uint256 _amount) returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) WithdrawFromPool0(_amount *big.Int) (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawFromPool0(&_AssertionStakingPool.TransactOpts, _amount)
}

// WithdrawStakeBackIntoPool is a paid mutator transaction binding the contract method 0x9451944d.
//
// Solidity: function withdrawStakeBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactor) WithdrawStakeBackIntoPool(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPool.contract.Transact(opts, "withdrawStakeBackIntoPool")
}

// WithdrawStakeBackIntoPool is a paid mutator transaction binding the contract method 0x9451944d.
//
// Solidity: function withdrawStakeBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolSession) WithdrawStakeBackIntoPool() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawStakeBackIntoPool(&_AssertionStakingPool.TransactOpts)
}

// WithdrawStakeBackIntoPool is a paid mutator transaction binding the contract method 0x9451944d.
//
// Solidity: function withdrawStakeBackIntoPool() returns()
func (_AssertionStakingPool *AssertionStakingPoolTransactorSession) WithdrawStakeBackIntoPool() (*types.Transaction, error) {
	return _AssertionStakingPool.Contract.WithdrawStakeBackIntoPool(&_AssertionStakingPool.TransactOpts)
}

// AssertionStakingPoolStakeDepositedIterator is returned from FilterStakeDeposited and is used to iterate over the raw logs and unpacked data for StakeDeposited events raised by the AssertionStakingPool contract.
type AssertionStakingPoolStakeDepositedIterator struct {
	Event *AssertionStakingPoolStakeDeposited // Event containing the contract specifics and raw log

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
func (it *AssertionStakingPoolStakeDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AssertionStakingPoolStakeDeposited)
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
		it.Event = new(AssertionStakingPoolStakeDeposited)
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
func (it *AssertionStakingPoolStakeDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AssertionStakingPoolStakeDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AssertionStakingPoolStakeDeposited represents a StakeDeposited event raised by the AssertionStakingPool contract.
type AssertionStakingPoolStakeDeposited struct {
	Sender common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStakeDeposited is a free log retrieval operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) FilterStakeDeposited(opts *bind.FilterOpts, sender []common.Address) (*AssertionStakingPoolStakeDepositedIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AssertionStakingPool.contract.FilterLogs(opts, "StakeDeposited", senderRule)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolStakeDepositedIterator{contract: _AssertionStakingPool.contract, event: "StakeDeposited", logs: logs, sub: sub}, nil
}

// WatchStakeDeposited is a free log subscription operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) WatchStakeDeposited(opts *bind.WatchOpts, sink chan<- *AssertionStakingPoolStakeDeposited, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AssertionStakingPool.contract.WatchLogs(opts, "StakeDeposited", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AssertionStakingPoolStakeDeposited)
				if err := _AssertionStakingPool.contract.UnpackLog(event, "StakeDeposited", log); err != nil {
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

// ParseStakeDeposited is a log parse operation binding the contract event 0x0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc2.
//
// Solidity: event StakeDeposited(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) ParseStakeDeposited(log types.Log) (*AssertionStakingPoolStakeDeposited, error) {
	event := new(AssertionStakingPoolStakeDeposited)
	if err := _AssertionStakingPool.contract.UnpackLog(event, "StakeDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AssertionStakingPoolStakeWithdrawnIterator is returned from FilterStakeWithdrawn and is used to iterate over the raw logs and unpacked data for StakeWithdrawn events raised by the AssertionStakingPool contract.
type AssertionStakingPoolStakeWithdrawnIterator struct {
	Event *AssertionStakingPoolStakeWithdrawn // Event containing the contract specifics and raw log

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
func (it *AssertionStakingPoolStakeWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AssertionStakingPoolStakeWithdrawn)
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
		it.Event = new(AssertionStakingPoolStakeWithdrawn)
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
func (it *AssertionStakingPoolStakeWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AssertionStakingPoolStakeWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AssertionStakingPoolStakeWithdrawn represents a StakeWithdrawn event raised by the AssertionStakingPool contract.
type AssertionStakingPoolStakeWithdrawn struct {
	Sender common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterStakeWithdrawn is a free log retrieval operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) FilterStakeWithdrawn(opts *bind.FilterOpts, sender []common.Address) (*AssertionStakingPoolStakeWithdrawnIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AssertionStakingPool.contract.FilterLogs(opts, "StakeWithdrawn", senderRule)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolStakeWithdrawnIterator{contract: _AssertionStakingPool.contract, event: "StakeWithdrawn", logs: logs, sub: sub}, nil
}

// WatchStakeWithdrawn is a free log subscription operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) WatchStakeWithdrawn(opts *bind.WatchOpts, sink chan<- *AssertionStakingPoolStakeWithdrawn, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _AssertionStakingPool.contract.WatchLogs(opts, "StakeWithdrawn", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AssertionStakingPoolStakeWithdrawn)
				if err := _AssertionStakingPool.contract.UnpackLog(event, "StakeWithdrawn", log); err != nil {
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

// ParseStakeWithdrawn is a log parse operation binding the contract event 0x8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc.
//
// Solidity: event StakeWithdrawn(address indexed sender, uint256 amount)
func (_AssertionStakingPool *AssertionStakingPoolFilterer) ParseStakeWithdrawn(log types.Log) (*AssertionStakingPoolStakeWithdrawn, error) {
	event := new(AssertionStakingPoolStakeWithdrawn)
	if err := _AssertionStakingPool.contract.UnpackLog(event, "StakeWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AssertionStakingPoolCreatorMetaData contains all meta data concerning the AssertionStakingPoolCreator contract.
var AssertionStakingPoolCreatorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"prevPrevAssertionHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"sequencerBatchAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"requiredStake\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"confirmPeriodBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nextInboxPosition\",\"type\":\"uint64\"}],\"internalType\":\"structConfigData\",\"name\":\"configData\",\"type\":\"tuple\"}],\"internalType\":\"structBeforeStateData\",\"name\":\"beforeStateData\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"beforeState\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"afterState\",\"type\":\"tuple\"}],\"internalType\":\"structAssertionInputs\",\"name\":\"assertionInputs\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"assertionHash\",\"type\":\"bytes32\"}],\"name\":\"PoolDoesntExist\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"rollup\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_assertionHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"assertionPool\",\"type\":\"address\"}],\"name\":\"NewAssertionPoolCreated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollup\",\"type\":\"address\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"prevPrevAssertionHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"sequencerBatchAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"requiredStake\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"confirmPeriodBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nextInboxPosition\",\"type\":\"uint64\"}],\"internalType\":\"structConfigData\",\"name\":\"configData\",\"type\":\"tuple\"}],\"internalType\":\"structBeforeStateData\",\"name\":\"beforeStateData\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"beforeState\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"afterState\",\"type\":\"tuple\"}],\"internalType\":\"structAssertionInputs\",\"name\":\"_assertionInputs\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"_assertionHash\",\"type\":\"bytes32\"}],\"name\":\"createPoolForAssertion\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollup\",\"type\":\"address\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"prevPrevAssertionHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"sequencerBatchAcc\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"wasmModuleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"requiredStake\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"challengeManager\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"confirmPeriodBlocks\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nextInboxPosition\",\"type\":\"uint64\"}],\"internalType\":\"structConfigData\",\"name\":\"configData\",\"type\":\"tuple\"}],\"internalType\":\"structBeforeStateData\",\"name\":\"beforeStateData\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"beforeState\",\"type\":\"tuple\"},{\"components\":[{\"components\":[{\"internalType\":\"bytes32[2]\",\"name\":\"bytes32Vals\",\"type\":\"bytes32[2]\"},{\"internalType\":\"uint64[2]\",\"name\":\"u64Vals\",\"type\":\"uint64[2]\"}],\"internalType\":\"structGlobalState\",\"name\":\"globalState\",\"type\":\"tuple\"},{\"internalType\":\"enumMachineStatus\",\"name\":\"machineStatus\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structAssertionState\",\"name\":\"afterState\",\"type\":\"tuple\"}],\"internalType\":\"structAssertionInputs\",\"name\":\"_assertionInputs\",\"type\":\"tuple\"},{\"internalType\":\"bytes32\",\"name\":\"_assertionHash\",\"type\":\"bytes32\"}],\"name\":\"getPool\",\"outputs\":[{\"internalType\":\"contractAssertionStakingPool\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611fc4806100206000396000f3fe60806040523480156200001157600080fd5b50600436106200003a5760003560e01c806341dd1eaa146200003f57806361841ec9146200007f575b600080fd5b620000566200005036600462000539565b62000096565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200160405180910390f35b620000566200009036600462000539565b620001ac565b600080620000a685858562000257565b90506000620000b78686866200028f565b604080517fff000000000000000000000000000000000000000000000000000000000000006020808301919091527fffffffffffffffffffffffffffffffffffffffff0000000000000000000000003060601b166021830152603582018690526055808301859052835180840390910181526075909201909252805191012090915073ffffffffffffffffffffffffffffffffffffffff81163b1562000162579250620001a5915050565b8686866040517fc070882d0000000000000000000000000000000000000000000000000000000081526004016200019c939291906200073a565b60405180910390fd5b9392505050565b600080620001bc85858562000257565b858585604051620001cd9062000312565b620001db939291906200073a565b8190604051809103906000f5905080158015620001fc573d6000803e3d6000fd5b5060405173ffffffffffffffffffffffffffffffffffffffff808316825291925084918716907fd628317c6ebae87acc5dbfadeb835cb97692cc6935ea72bf37461e14a0bbee1e9060200160405180910390a3949350505050565b600083838360405160200162000270939291906200073a565b6040516020818303038152906040528051906020012090509392505050565b60008060405180602001620002a49062000312565b6020820181038252601f19601f82011660405250905080858585604051602001620002d2939291906200073a565b60408051601f1981840301815290829052620002f2929160200162000828565b604051602081830303815290604052805190602001209150509392505050565b611745806200084a83390190565b803573ffffffffffffffffffffffffffffffffffffffff811681146200034557600080fd5b919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040516060810167ffffffffffffffff811182821017156200039f576200039f6200034a565b60405290565b6040805190810167ffffffffffffffff811182821017156200039f576200039f6200034a565b60405160a0810167ffffffffffffffff811182821017156200039f576200039f6200034a565b803567ffffffffffffffff811681146200034557600080fd5b600082601f8301126200041c57600080fd5b62000426620003a5565b8060408401858111156200043957600080fd5b845b818110156200045e576200044f81620003f1565b8452602093840193016200043b565b509095945050505050565b8035600381106200034557600080fd5b600081830360c08112156200048d57600080fd5b6200049762000379565b91506080811215620004a857600080fd5b50620004b3620003a5565b83601f840112620004c357600080fd5b620004cd620003a5565b806040850186811115620004e057600080fd5b855b81811015620004fc578035845260209384019301620004e2565b508184526200050c87826200040a565b60208501525050508152620005246080830162000469565b602082015260a0820135604082015292915050565b60008060008385036102a08112156200055157600080fd5b6200055c8562000320565b9350601f1981016102608112156200057357600080fd5b6200057d62000379565b60e08212156200058c57600080fd5b6200059662000379565b9150602087013582526040870135602083015260a07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa084011215620005da57600080fd5b620005e4620003cb565b925060608701358352608087013560208401526200060560a0880162000320565b60408401526200061860c08801620003f1565b60608401526200062b60e08801620003f1565b60808401528260408301528181526200064988610100890162000479565b60208201526200065e886101c0890162000479565b6040820152949794965050505061028092909201359150565b60038110620006af577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b9052565b805180518360005b6002811015620006dc578251825260209283019290910190600101620006bb565b505050602090810151906040840160005b60028110156200071657835167ffffffffffffffff1682529282019290820190600101620006ed565b505082015190506200072c608084018262000677565b506040015160a09190910152565b60006102a08201905073ffffffffffffffffffffffffffffffffffffffff8086168352845180516020850152602081015160408501526040810151905080516060850152602081015160808501528160408201511660a08501526060810151915067ffffffffffffffff80831660c08601528060808301511660e08601525050506020840151620007d0610100840182620006b3565b506040840151620007e66101c0840182620006b3565b5082610280830152949350505050565b6000815160005b81811015620008195760208185018101518683015201620007fd565b50600093019283525090919050565b6000620008416200083a8386620007f6565b84620007f6565b94935050505056fe60e06040523480156200001157600080fd5b50604051620017453803806200174583398101604081905262000034916200050a565b6001600160a01b03838116608090815260a0839052835180516000908155602080830151600155604092830151805160029081558183015160035593810151600480546060840151929098166001600160e01b031990981697909717600160a01b6001600160401b03928316021790965590930151600580546001600160401b03191691909516179093559084015180518051869493600692918391620000dd9183916200021f565b506020820151620000f5906002808401919062000262565b505050602082015160038201805460ff191660018360028111156200011e576200011e6200062b565b021790555060409182015160049091015582015180518051600b8401919082906200014d90829060026200021f565b50602082015162000165906002808401919062000262565b505050602082015160038201805460ff191660018360028111156200018e576200018e6200062b565b02179055506040820151816004015550509050506080516001600160a01b03166351ed6a306040518163ffffffff1660e01b8152600401602060405180830381865afa158015620001e3573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019062000209919062000641565b6001600160a01b031660c0525062000666915050565b826002810192821562000250579160200282015b828111156200025057825182559160200191906001019062000233565b506200025e92915062000304565b5090565b600183019183908215620002505791602002820160005b83821115620002c557835183826101000a8154816001600160401b0302191690836001600160401b03160217905550926020019260080160208160070104928301926001030262000279565b8015620002fa5782816101000a8154906001600160401b030219169055600801602081600701049283019260010302620002c5565b50506200025e9291505b5b808211156200025e576000815560010162000305565b80516001600160a01b03811681146200033357600080fd5b919050565b634e487b7160e01b600052604160045260246000fd5b604051606081016001600160401b038111828210171562000373576200037362000338565b60405290565b604080519081016001600160401b038111828210171562000373576200037362000338565b60405160a081016001600160401b038111828210171562000373576200037362000338565b80516001600160401b03811681146200033357600080fd5b600082601f830112620003ed57600080fd5b620003f762000379565b8060408401858111156200040a57600080fd5b845b818110156200042f576200042081620003c3565b8452602093840193016200040c565b509095945050505050565b8051600381106200033357600080fd5b600081830360c08112156200045e57600080fd5b620004686200034e565b915060808112156200047957600080fd5b506200048462000379565b83601f8401126200049457600080fd5b6200049e62000379565b806040850186811115620004b157600080fd5b855b81811015620004cd578051845260209384019301620004b3565b50818452620004dd8782620003db565b60208501525050508152620004f5608083016200043a565b602082015260a0820151604082015292915050565b60008060008385036102a08112156200052257600080fd5b6200052d856200031b565b9350601f1981016102608112156200054457600080fd5b6200054e6200034e565b60e08212156200055d57600080fd5b620005676200034e565b9150602087015182526040870151602083015260a0605f19840112156200058d57600080fd5b620005976200039e565b92506060870151835260808701516020840152620005b860a088016200031b565b6040840152620005cb60c08801620003c3565b6060840152620005de60e08801620003c3565b6080840152826040830152818152620005fc8861010089016200044a565b602082015262000611886101c089016200044a565b604082015280945050505061028084015190509250925092565b634e487b7160e01b600052602160045260246000fd5b6000602082840312156200065457600080fd5b6200065f826200031b565b9392505050565b60805160a05160c051611078620006cd60003960008181610148015281816102ec0152818161036e01526104b601526000818160e901526104170152600081816101e901528181610390015281816103e50152818161077901526107ee01526110786000f3fe608060405234801561001057600080fd5b50600436106100df5760003560e01c80637476083b1161008c578063930412af11610066578063930412af146101d45780639451944d146101dc578063cb23bcb5146101e4578063f0e978891461020b57600080fd5b80637476083b1461018a578063875b2af01461019d5780639252175b146101bd57600080fd5b80634b7a7538116100bd5780634b7a75381461013b57806351ed6a30146101435780636b74d5151461018257600080fd5b80632113ed21146100e457806326c0e5c51461011e57806330fc43ed14610128575b600080fd5b61010b7f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020015b60405180910390f35b610126610213565b005b610126610136366004610cd7565b61022e565b610126610354565b61016a7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b039091168152602001610115565b610126610474565b610126610198366004610cd7565b610484565b61010b6101ab366004610cf0565b60106020526000908152604090205481565b6101c5610516565b60405161011593929190610dd2565b610126610777565b6101266107ec565b61016a7f000000000000000000000000000000000000000000000000000000000000000081565b60035461010b565b3360009081526010602052604090205461022c9061022e565b565b336000908152601060205260408120549081900361027f576040517fe06b2da50000000000000000000000000000000000000000000000000000000081523360048201526024015b60405180910390fd5b808211156102c9576040517fa47b7c650000000000000000000000000000000000000000000000000000000081523360048201526024810183905260448101829052606401610276565b6102d38282610e68565b3360008181526010602052604090209190915561031b907f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169084610873565b60405182815233907f8108595eb6bad3acefa9da467d90cc2217686d5c5ac85460f8b7849c840645fc9060200160405180910390a25050565b600061035f60035490565b90506103b56001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000167f000000000000000000000000000000000000000000000000000000000000000083610921565b6040517f7300201c0000000000000000000000000000000000000000000000000000000081526001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001690637300201c9061043f9084906000907f000000000000000000000000000000000000000000000000000000000000000090600401610eee565b600060405180830381600087803b15801561045957600080fd5b505af115801561046d573d6000803e3d6000fd5b5050505050565b61047c610777565b61022c6107ec565b33600090815260106020526040812080548392906104a3908490610f81565b909155506104de90506001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016333084610a05565b60405181815233907f0a7bb2e28cc4698aac06db79cf9163bfcc20719286cf59fa7d492ceda1b8edc29060200160405180910390a250565b6040805160608082018352600080548352600154602080850191909152845160a08082018752600280548352600354938301939093526004546001600160a01b0381168389015274010000000000000000000000000000000000000000900467ffffffffffffffff9081168387015260055416608083015285870191909152855160e08101968790529495929493600693859391840192859284929186019184919082845b8154815260200190600101908083116105bb575050509183525050604080518082019182905260209092019190600284810191826000855b82829054906101000a900467ffffffffffffffff1667ffffffffffffffff16815260200190600801906020826007010492830192600103820291508084116105f3579050505050919092525050508152600382015460209091019060ff16600281111561066257610662610d19565b600281111561067357610673610d19565b8152600491909101546020909101526040805160e08101909152909190600b82018160608101828160a084018260028282826020028201915b8154815260200190600101908083116106ac575050509183525050604080518082019182905260209092019190600284810191826000855b82829054906101000a900467ffffffffffffffff1667ffffffffffffffff16815260200190600801906020826007010492830192600103820291508084116106e4579050505050919092525050508152600382015460209091019060ff16600281111561075357610753610d19565b600281111561076457610764610d19565b8152602001600482015481525050905083565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03166357ef4ab96040518163ffffffff1660e01b8152600401600060405180830381600087803b1580156107d257600080fd5b505af11580156107e6573d6000803e3d6000fd5b50505050565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031663613739196040518163ffffffff1660e01b81526004016020604051808303816000875af115801561084c573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906108709190610f94565b50565b6040516001600160a01b03831660248201526044810182905261091c9084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff0000000000000000000000000000000000000000000000000000000090931692909217909152610a56565b505050565b6040517fdd62ed3e0000000000000000000000000000000000000000000000000000000081523060048201526001600160a01b038381166024830152600091839186169063dd62ed3e90604401602060405180830381865afa15801561098b573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906109af9190610f94565b6109b99190610f81565b6040516001600160a01b0385166024820152604481018290529091506107e69085907f095ea7b300000000000000000000000000000000000000000000000000000000906064016108b8565b6040516001600160a01b03808516602483015283166044820152606481018290526107e69085907f23b872dd00000000000000000000000000000000000000000000000000000000906084016108b8565b6000610aab826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564815250856001600160a01b0316610b3b9092919063ffffffff16565b80519091501561091c5780806020019051810190610ac99190610fad565b61091c5760405162461bcd60e51b815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f742073756363656564000000000000000000000000000000000000000000006064820152608401610276565b6060610b4a8484600085610b54565b90505b9392505050565b606082471015610bcc5760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c00000000000000000000000000000000000000000000000000006064820152608401610276565b6001600160a01b0385163b610c235760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e74726163740000006044820152606401610276565b600080866001600160a01b03168587604051610c3f9190610ff3565b60006040518083038185875af1925050503d8060008114610c7c576040519150601f19603f3d011682016040523d82523d6000602084013e610c81565b606091505b5091509150610c91828286610c9e565b925050505b949350505050565b60608315610cad575081610b4d565b825115610cbd5782518084602001fd5b8160405162461bcd60e51b8152600401610276919061100f565b600060208284031215610ce957600080fd5b5035919050565b600060208284031215610d0257600080fd5b81356001600160a01b0381168114610b4d57600080fd5b634e487b7160e01b600052602160045260246000fd5b60038110610d4d57634e487b7160e01b600052602160045260246000fd5b9052565b805180518360005b6002811015610d78578251825260209283019290910190600101610d59565b505050602090810151906040840160005b6002811015610db057835167ffffffffffffffff1682529282019290820190600101610d89565b50508201519050610dc46080840182610d2f565b506040015160a09190910152565b6000610260820190508451825260208501516020830152604085015180516040840152602081015160608401526001600160a01b036040820151166080840152606081015167ffffffffffffffff80821660a08601528060808401511660c0860152505050610e4460e0830185610d51565b610c966101a0830184610d51565b634e487b7160e01b600052601160045260246000fd5b81810381811115610e7b57610e7b610e52565b92915050565b818160005b6002811015610ea5578154835260209092019160019182019101610e86565b505050600281015467ffffffffffffffff8082166040850152808260401c166060850152505060ff600382015416610ee06080840182610d2f565b506004015460a09190910152565b8381528254602082015260018301546040820152600283015460608201526003830154608082015260048301546001600160a01b03811660a0808401919091521c67ffffffffffffffff90811660c083015260058401541660e08201526102a08101610f61610100830160068601610e81565b610f726101c08301600b8601610e81565b82610280830152949350505050565b80820180821115610e7b57610e7b610e52565b600060208284031215610fa657600080fd5b5051919050565b600060208284031215610fbf57600080fd5b81518015158114610b4d57600080fd5b60005b83811015610fea578181015183820152602001610fd2565b50506000910152565b60008251611005818460208701610fcf565b9190910192915050565b602081526000825180602084015261102e816040850160208701610fcf565b601f01601f1916919091016040019291505056fea26469706673582212209087fdad40ab4da26397456263ffcb9377057af7edb54d6f02cda01ffdd72c0e64736f6c63430008110033a264697066735822122052476b12d560aa91318683075894d1d85091672ff176aade555a8ea430d5abe464736f6c63430008110033",
}

// AssertionStakingPoolCreatorABI is the input ABI used to generate the binding from.
// Deprecated: Use AssertionStakingPoolCreatorMetaData.ABI instead.
var AssertionStakingPoolCreatorABI = AssertionStakingPoolCreatorMetaData.ABI

// AssertionStakingPoolCreatorBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AssertionStakingPoolCreatorMetaData.Bin instead.
var AssertionStakingPoolCreatorBin = AssertionStakingPoolCreatorMetaData.Bin

// DeployAssertionStakingPoolCreator deploys a new Ethereum contract, binding an instance of AssertionStakingPoolCreator to it.
func DeployAssertionStakingPoolCreator(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *AssertionStakingPoolCreator, error) {
	parsed, err := AssertionStakingPoolCreatorMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AssertionStakingPoolCreatorBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AssertionStakingPoolCreator{AssertionStakingPoolCreatorCaller: AssertionStakingPoolCreatorCaller{contract: contract}, AssertionStakingPoolCreatorTransactor: AssertionStakingPoolCreatorTransactor{contract: contract}, AssertionStakingPoolCreatorFilterer: AssertionStakingPoolCreatorFilterer{contract: contract}}, nil
}

// AssertionStakingPoolCreator is an auto generated Go binding around an Ethereum contract.
type AssertionStakingPoolCreator struct {
	AssertionStakingPoolCreatorCaller     // Read-only binding to the contract
	AssertionStakingPoolCreatorTransactor // Write-only binding to the contract
	AssertionStakingPoolCreatorFilterer   // Log filterer for contract events
}

// AssertionStakingPoolCreatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type AssertionStakingPoolCreatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolCreatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AssertionStakingPoolCreatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolCreatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AssertionStakingPoolCreatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionStakingPoolCreatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AssertionStakingPoolCreatorSession struct {
	Contract     *AssertionStakingPoolCreator // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                // Call options to use throughout this session
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// AssertionStakingPoolCreatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AssertionStakingPoolCreatorCallerSession struct {
	Contract *AssertionStakingPoolCreatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                      // Call options to use throughout this session
}

// AssertionStakingPoolCreatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AssertionStakingPoolCreatorTransactorSession struct {
	Contract     *AssertionStakingPoolCreatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                      // Transaction auth options to use throughout this session
}

// AssertionStakingPoolCreatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type AssertionStakingPoolCreatorRaw struct {
	Contract *AssertionStakingPoolCreator // Generic contract binding to access the raw methods on
}

// AssertionStakingPoolCreatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AssertionStakingPoolCreatorCallerRaw struct {
	Contract *AssertionStakingPoolCreatorCaller // Generic read-only contract binding to access the raw methods on
}

// AssertionStakingPoolCreatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AssertionStakingPoolCreatorTransactorRaw struct {
	Contract *AssertionStakingPoolCreatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAssertionStakingPoolCreator creates a new instance of AssertionStakingPoolCreator, bound to a specific deployed contract.
func NewAssertionStakingPoolCreator(address common.Address, backend bind.ContractBackend) (*AssertionStakingPoolCreator, error) {
	contract, err := bindAssertionStakingPoolCreator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCreator{AssertionStakingPoolCreatorCaller: AssertionStakingPoolCreatorCaller{contract: contract}, AssertionStakingPoolCreatorTransactor: AssertionStakingPoolCreatorTransactor{contract: contract}, AssertionStakingPoolCreatorFilterer: AssertionStakingPoolCreatorFilterer{contract: contract}}, nil
}

// NewAssertionStakingPoolCreatorCaller creates a new read-only instance of AssertionStakingPoolCreator, bound to a specific deployed contract.
func NewAssertionStakingPoolCreatorCaller(address common.Address, caller bind.ContractCaller) (*AssertionStakingPoolCreatorCaller, error) {
	contract, err := bindAssertionStakingPoolCreator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCreatorCaller{contract: contract}, nil
}

// NewAssertionStakingPoolCreatorTransactor creates a new write-only instance of AssertionStakingPoolCreator, bound to a specific deployed contract.
func NewAssertionStakingPoolCreatorTransactor(address common.Address, transactor bind.ContractTransactor) (*AssertionStakingPoolCreatorTransactor, error) {
	contract, err := bindAssertionStakingPoolCreator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCreatorTransactor{contract: contract}, nil
}

// NewAssertionStakingPoolCreatorFilterer creates a new log filterer instance of AssertionStakingPoolCreator, bound to a specific deployed contract.
func NewAssertionStakingPoolCreatorFilterer(address common.Address, filterer bind.ContractFilterer) (*AssertionStakingPoolCreatorFilterer, error) {
	contract, err := bindAssertionStakingPoolCreator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCreatorFilterer{contract: contract}, nil
}

// bindAssertionStakingPoolCreator binds a generic wrapper to an already deployed contract.
func bindAssertionStakingPoolCreator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AssertionStakingPoolCreatorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AssertionStakingPoolCreator.Contract.AssertionStakingPoolCreatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.AssertionStakingPoolCreatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.AssertionStakingPoolCreatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AssertionStakingPoolCreator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.contract.Transact(opts, method, params...)
}

// GetPool is a free data retrieval call binding the contract method 0x41dd1eaa.
//
// Solidity: function getPool(address _rollup, ((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)),((bytes32[2],uint64[2]),uint8,bytes32),((bytes32[2],uint64[2]),uint8,bytes32)) _assertionInputs, bytes32 _assertionHash) view returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorCaller) GetPool(opts *bind.CallOpts, _rollup common.Address, _assertionInputs AssertionInputs, _assertionHash [32]byte) (common.Address, error) {
	var out []interface{}
	err := _AssertionStakingPoolCreator.contract.Call(opts, &out, "getPool", _rollup, _assertionInputs, _assertionHash)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetPool is a free data retrieval call binding the contract method 0x41dd1eaa.
//
// Solidity: function getPool(address _rollup, ((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)),((bytes32[2],uint64[2]),uint8,bytes32),((bytes32[2],uint64[2]),uint8,bytes32)) _assertionInputs, bytes32 _assertionHash) view returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorSession) GetPool(_rollup common.Address, _assertionInputs AssertionInputs, _assertionHash [32]byte) (common.Address, error) {
	return _AssertionStakingPoolCreator.Contract.GetPool(&_AssertionStakingPoolCreator.CallOpts, _rollup, _assertionInputs, _assertionHash)
}

// GetPool is a free data retrieval call binding the contract method 0x41dd1eaa.
//
// Solidity: function getPool(address _rollup, ((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)),((bytes32[2],uint64[2]),uint8,bytes32),((bytes32[2],uint64[2]),uint8,bytes32)) _assertionInputs, bytes32 _assertionHash) view returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorCallerSession) GetPool(_rollup common.Address, _assertionInputs AssertionInputs, _assertionHash [32]byte) (common.Address, error) {
	return _AssertionStakingPoolCreator.Contract.GetPool(&_AssertionStakingPoolCreator.CallOpts, _rollup, _assertionInputs, _assertionHash)
}

// CreatePoolForAssertion is a paid mutator transaction binding the contract method 0x61841ec9.
//
// Solidity: function createPoolForAssertion(address _rollup, ((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)),((bytes32[2],uint64[2]),uint8,bytes32),((bytes32[2],uint64[2]),uint8,bytes32)) _assertionInputs, bytes32 _assertionHash) returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorTransactor) CreatePoolForAssertion(opts *bind.TransactOpts, _rollup common.Address, _assertionInputs AssertionInputs, _assertionHash [32]byte) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.contract.Transact(opts, "createPoolForAssertion", _rollup, _assertionInputs, _assertionHash)
}

// CreatePoolForAssertion is a paid mutator transaction binding the contract method 0x61841ec9.
//
// Solidity: function createPoolForAssertion(address _rollup, ((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)),((bytes32[2],uint64[2]),uint8,bytes32),((bytes32[2],uint64[2]),uint8,bytes32)) _assertionInputs, bytes32 _assertionHash) returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorSession) CreatePoolForAssertion(_rollup common.Address, _assertionInputs AssertionInputs, _assertionHash [32]byte) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.CreatePoolForAssertion(&_AssertionStakingPoolCreator.TransactOpts, _rollup, _assertionInputs, _assertionHash)
}

// CreatePoolForAssertion is a paid mutator transaction binding the contract method 0x61841ec9.
//
// Solidity: function createPoolForAssertion(address _rollup, ((bytes32,bytes32,(bytes32,uint256,address,uint64,uint64)),((bytes32[2],uint64[2]),uint8,bytes32),((bytes32[2],uint64[2]),uint8,bytes32)) _assertionInputs, bytes32 _assertionHash) returns(address)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorTransactorSession) CreatePoolForAssertion(_rollup common.Address, _assertionInputs AssertionInputs, _assertionHash [32]byte) (*types.Transaction, error) {
	return _AssertionStakingPoolCreator.Contract.CreatePoolForAssertion(&_AssertionStakingPoolCreator.TransactOpts, _rollup, _assertionInputs, _assertionHash)
}

// AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator is returned from FilterNewAssertionPoolCreated and is used to iterate over the raw logs and unpacked data for NewAssertionPoolCreated events raised by the AssertionStakingPoolCreator contract.
type AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator struct {
	Event *AssertionStakingPoolCreatorNewAssertionPoolCreated // Event containing the contract specifics and raw log

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
func (it *AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AssertionStakingPoolCreatorNewAssertionPoolCreated)
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
		it.Event = new(AssertionStakingPoolCreatorNewAssertionPoolCreated)
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
func (it *AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AssertionStakingPoolCreatorNewAssertionPoolCreated represents a NewAssertionPoolCreated event raised by the AssertionStakingPoolCreator contract.
type AssertionStakingPoolCreatorNewAssertionPoolCreated struct {
	Rollup        common.Address
	AssertionHash [32]byte
	AssertionPool common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterNewAssertionPoolCreated is a free log retrieval operation binding the contract event 0xd628317c6ebae87acc5dbfadeb835cb97692cc6935ea72bf37461e14a0bbee1e.
//
// Solidity: event NewAssertionPoolCreated(address indexed rollup, bytes32 indexed _assertionHash, address assertionPool)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorFilterer) FilterNewAssertionPoolCreated(opts *bind.FilterOpts, rollup []common.Address, _assertionHash [][32]byte) (*AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator, error) {

	var rollupRule []interface{}
	for _, rollupItem := range rollup {
		rollupRule = append(rollupRule, rollupItem)
	}
	var _assertionHashRule []interface{}
	for _, _assertionHashItem := range _assertionHash {
		_assertionHashRule = append(_assertionHashRule, _assertionHashItem)
	}

	logs, sub, err := _AssertionStakingPoolCreator.contract.FilterLogs(opts, "NewAssertionPoolCreated", rollupRule, _assertionHashRule)
	if err != nil {
		return nil, err
	}
	return &AssertionStakingPoolCreatorNewAssertionPoolCreatedIterator{contract: _AssertionStakingPoolCreator.contract, event: "NewAssertionPoolCreated", logs: logs, sub: sub}, nil
}

// WatchNewAssertionPoolCreated is a free log subscription operation binding the contract event 0xd628317c6ebae87acc5dbfadeb835cb97692cc6935ea72bf37461e14a0bbee1e.
//
// Solidity: event NewAssertionPoolCreated(address indexed rollup, bytes32 indexed _assertionHash, address assertionPool)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorFilterer) WatchNewAssertionPoolCreated(opts *bind.WatchOpts, sink chan<- *AssertionStakingPoolCreatorNewAssertionPoolCreated, rollup []common.Address, _assertionHash [][32]byte) (event.Subscription, error) {

	var rollupRule []interface{}
	for _, rollupItem := range rollup {
		rollupRule = append(rollupRule, rollupItem)
	}
	var _assertionHashRule []interface{}
	for _, _assertionHashItem := range _assertionHash {
		_assertionHashRule = append(_assertionHashRule, _assertionHashItem)
	}

	logs, sub, err := _AssertionStakingPoolCreator.contract.WatchLogs(opts, "NewAssertionPoolCreated", rollupRule, _assertionHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AssertionStakingPoolCreatorNewAssertionPoolCreated)
				if err := _AssertionStakingPoolCreator.contract.UnpackLog(event, "NewAssertionPoolCreated", log); err != nil {
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

// ParseNewAssertionPoolCreated is a log parse operation binding the contract event 0xd628317c6ebae87acc5dbfadeb835cb97692cc6935ea72bf37461e14a0bbee1e.
//
// Solidity: event NewAssertionPoolCreated(address indexed rollup, bytes32 indexed _assertionHash, address assertionPool)
func (_AssertionStakingPoolCreator *AssertionStakingPoolCreatorFilterer) ParseNewAssertionPoolCreated(log types.Log) (*AssertionStakingPoolCreatorNewAssertionPoolCreated, error) {
	event := new(AssertionStakingPoolCreatorNewAssertionPoolCreated)
	if err := _AssertionStakingPoolCreator.contract.UnpackLog(event, "NewAssertionPoolCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
