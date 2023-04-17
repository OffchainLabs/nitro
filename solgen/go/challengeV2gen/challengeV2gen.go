// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package challengeV2gen

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

// AddLeafArgs is an auto generated low-level Go binding around an user-defined struct.
type AddLeafArgs struct {
	ChallengeId            [32]byte
	ClaimId                [32]byte
	Height                 *big.Int
	HistoryRoot            [32]byte
	FirstState             [32]byte
	FirstStatehistoryProof [][32]byte
	LastState              [32]byte
	LastStatehistoryProof  [][32]byte
}

// Assertion is an auto generated low-level Go binding around an user-defined struct.
type Assertion struct {
	PredecessorId           [32]byte
	SuccessionChallenge     [32]byte
	IsFirstChild            bool
	SecondChildCreationTime *big.Int
	FirstChildCreationTime  *big.Int
	StateHash               [32]byte
	Height                  *big.Int
	Status                  uint8
	InboxMsgCountSeen       *big.Int
}

// Challenge is an auto generated low-level Go binding around an user-defined struct.
type Challenge struct {
	RootId        [32]byte
	WinningClaim  [32]byte
	ChallengeType uint8
	Challenger    common.Address
}

// ChallengeEdge is an auto generated low-level Go binding around an user-defined struct.
type ChallengeEdge struct {
	OriginId         [32]byte
	StartHistoryRoot [32]byte
	StartHeight      *big.Int
	EndHistoryRoot   [32]byte
	EndHeight        *big.Int
	LowerChildId     [32]byte
	UpperChildId     [32]byte
	CreatedWhen      *big.Int
	ClaimId          [32]byte
	Staker           common.Address
	Status           uint8
	EType            uint8
}

// ChallengeVertex is an auto generated low-level Go binding around an user-defined struct.
type ChallengeVertex struct {
	ChallengeId             [32]byte
	HistoryRoot             [32]byte
	Height                  *big.Int
	SuccessionChallenge     [32]byte
	PredecessorId           [32]byte
	ClaimId                 [32]byte
	Staker                  common.Address
	Status                  uint8
	PsId                    [32]byte
	PsLastUpdatedTimestamp  *big.Int
	FlushedPsTimeSec        *big.Int
	LowestHeightSuccessorId [32]byte
}

// CreateEdgeArgs is an auto generated low-level Go binding around an user-defined struct.
type CreateEdgeArgs struct {
	EdgeType         uint8
	StartHistoryRoot [32]byte
	StartHeight      *big.Int
	EndHistoryRoot   [32]byte
	EndHeight        *big.Int
	ClaimId          [32]byte
}

// ExecutionContext is an auto generated low-level Go binding around an user-defined struct.
type ExecutionContext struct {
	MaxInboxMessagesRead *big.Int
	Bridge               common.Address
}

// OneStepData is an auto generated low-level Go binding around an user-defined struct.
type OneStepData struct {
	ExecCtx     ExecutionContext
	MachineStep *big.Int
	BeforeHash  [32]byte
	Proof       []byte
}

// AssertionChainMetaData contains all meta data concerning the AssertionChain contract.
var AssertionChainMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"stateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriodSeconds\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"NotConfirmable\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"NotRejectable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"addStake\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"assertionExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"assertions\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"predecessorId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"successionChallenge\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"isFirstChild\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"secondChildCreationTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"firstChildCreationTime\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"stateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"enumStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"inboxMsgCountSeen\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"challengeManagerAddr\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"challengePeriodSeconds\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"confirmAssertion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"stateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"predecessorId\",\"type\":\"bytes32\"}],\"name\":\"createNewAssertion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"createSuccessionChallenge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"getAssertion\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"predecessorId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"successionChallenge\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"isFirstChild\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"secondChildCreationTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"firstChildCreationTime\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"stateHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"enumStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"inboxMsgCountSeen\",\"type\":\"uint256\"}],\"internalType\":\"structAssertion\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getFirstChildCreationTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getHeight\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getInboxMsgCountSeen\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getPredecessorId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getStateHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getSuccessionChallenge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"isFirstChild\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"rejectAssertion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stakeAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIChallengeManager\",\"name\":\"_challengeManager\",\"type\":\"address\"}],\"name\":\"updateChallengeManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a060405268056bc75e2d6310000060805234801561001d57600080fd5b5060405161162c38038061162c83398101604081905261003c916101f9565b6002818155604080516101208101825260008082526020808301828152938301828152606084018381526080850184815260a086018a815260c08701868152600160e089018181526101008a018990528880529681905288517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4990815599517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4a5594517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4b805491151560ff1992831617905593517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4c5591517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4d55517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4e55517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4f5591517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb50805494979596959194909316919084908111156101de576101de61021d565b02179055506101008201518160080155905050505050610233565b6000806040838503121561020c57600080fd5b505080516020909101519092909150565b634e487b7160e01b600052602160045260246000fd5b6080516113d76102556000396000818161025f01526108b801526113d76000f3fe6080604052600436106101045760003560e01c80636894bdd5116100a05780639ca565d4116100645780639ca565d41461032e578063d60715b51461034e578063f9bce634146103cf578063fb601294146103ef578063ff8aef871461040557600080fd5b80636894bdd51461028157806375dc6098146102a15780637cfd5ab9146102c157806388302884146102e1578063896efbf21461030e57600080fd5b8063295dfd32146101095780632fefa18a14610148578063308362281461017b57806343ed6ad9146101ab57806349635f9a146101cb5780635625c360146101eb5780635a4038f5146102135780635a627dbc1461024557806360c7dc471461024d575b600080fd5b34801561011557600080fd5b506101466101243660046111a2565b600080546001600160a01b0319166001600160a01b0392909216919091179055565b005b34801561015457600080fd5b506101686101633660046111d2565b610425565b6040519081526020015b60405180910390f35b34801561018757600080fd5b5061019b6101963660046111d2565b610473565b6040519015158152602001610172565b3480156101b757600080fd5b506101686101c63660046111d2565b6104ba565b3480156101d757600080fd5b506101466101e63660046111eb565b6104fe565b3480156101f757600080fd5b506000546040516001600160a01b039091168152602001610172565b34801561021f57600080fd5b5061019b61022e3660046111d2565b600090815260016020526040902060050154151590565b6101466108b6565b34801561025957600080fd5b506101687f000000000000000000000000000000000000000000000000000000000000000081565b34801561028d57600080fd5b5061014661029c3660046111d2565b610927565b3480156102ad57600080fd5b506101466102bc3660046111d2565b610b38565b3480156102cd57600080fd5b506101686102dc3660046111d2565b610d43565b3480156102ed57600080fd5b506103016102fc3660046111d2565b610d87565b604051610172919061124f565b34801561031a57600080fd5b506101686103293660046111d2565b610e7b565b34801561033a57600080fd5b506101686103493660046111d2565b610ebf565b34801561035a57600080fd5b506103ba6103693660046111d2565b6001602081905260009182526040909120805491810154600282015460038301546004840154600585015460068601546007870154600890970154959660ff95861696949593949293919291169089565b604051610172999897969594939291906112bc565b3480156103db57600080fd5b506101686103ea3660046111d2565b610f00565b3480156103fb57600080fd5b5061016860025481565b34801561041157600080fd5b506101466104203660046111d2565b610f44565b60008181526001602052604081206005015461045c5760405162461bcd60e51b815260040161045390611310565b60405180910390fd5b506000908152600160208190526040909120015490565b6000818152600160205260408120600501546104a15760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090206002015460ff1690565b6000818152600160205260408120600501546104e85760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090206004015490565b60408051602081018590529081018390526060810182905260009060800160405160208183030381529060405280519060200120905061054f81600090815260016020526040902060050154151590565b156105975760405162461bcd60e51b8152602060048201526018602482015277417373657274696f6e20616c72656164792065786973747360401b6044820152606401610453565b6000828152600160205260409020600501546105ff5760405162461bcd60e51b815260206004820152602160248201527f50726576696f757320617373657274696f6e20646f6573206e6f7420657869736044820152601d60fa1b6064820152608401610453565b600260008281526001602052604080822054825290206007015460ff16600281111561062d5761062d611217565b0361067a5760405162461bcd60e51b815260206004820152601b60248201527f50726576696f757320617373657274696f6e2072656a656374656400000000006044820152606401610453565b6000818152600160205260408082205482529020839060060154106106ed5760405162461bcd60e51b815260206004820152602360248201527f486569676874206e6f742067726561746572207468616e20707265646563657360448201526239b7b960e91b6064820152608401610453565b600082815260016020526040902060040154151580610720576000838152600160205260409020426004909101556107c1565b6002546000838152600160205260408082205482529020600401546107459190611358565b42106107935760405162461bcd60e51b815260206004820152601a60248201527f546f6f206c61746520746f20637265617465207369626c696e670000000000006044820152606401610453565b60008381526001602052604081206003015490036107c1576000838152600160205260409020426003909101555b6040518061012001604052808481526020016000801b815260200182151515815260200160008152602001600081526020018681526020018581526020016000600281111561081257610812611217565b8152600060209182018190528481526001808352604091829020845181559284015183820155908301516002808401805492151560ff19938416179055606085015160038501556080850151600485015560a0850151600585015560c0850151600685015560e0850151600785018054919490939190911691849081111561089c5761089c611217565b021790555061010082015181600801559050505050505050565b7f000000000000000000000000000000000000000000000000000000000000000034146109255760405162461bcd60e51b815260206004820152601a60248201527f436f7272656374207374616b65206e6f742070726f76696465640000000000006044820152606401610453565b565b6000818152600160205260409020600501546109555760405162461bcd60e51b815260040161045390611310565b600160008281526001602052604080822054825290206007015460ff16600281111561098357610983611217565b146109d05760405162461bcd60e51b815260206004820181905260248201527f50726576696f757320617373657274696f6e206e6f7420636f6e6669726d65646044820152606401610453565b600081815260016020526040808220548252902060030154158015610a185750600254600082815260016020526040808220548252902060040154610a159190611358565b42115b15610a42576000818152600160208190526040909120600701805460ff191682805b021790555050565b60008181526001602081905260408083205483528220015490819003610a7e57604051631895e8f560e21b815260048101839052602401610453565b60008054604051630e7a2a9d60e31b8152600481018490526001600160a01b03909116906373d154e890602401602060405180830381865afa158015610ac8573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610aec9190611371565b9050828114610b1157604051632158b7ff60e11b815260048101849052602401610453565b6000838152600160208190526040909120600701805460ff191682805b0217905550505050565b600081815260016020526040902060050154610b665760405162461bcd60e51b815260040161045390611310565b60008181526001602052604081206007015460ff166002811115610b8c57610b8c611217565b14610bd45760405162461bcd60e51b8152602060048201526018602482015277417373657274696f6e206973206e6f742070656e64696e6760401b6044820152606401610453565b600260008281526001602052604080822054825290206007015460ff166002811115610c0257610c02611217565b03610c2d576000818152600160208190526040909120600701805460029260ff199091169083610a3a565b60008181526001602081905260408083205483528220015490819003610c6957604051632158b7ff60e11b815260048101839052602401610453565b60008054604051630e7a2a9d60e31b8152600481018490526001600160a01b03909116906373d154e890602401602060405180830381865afa158015610cb3573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610cd79190611371565b905080610cfa57604051632158b7ff60e11b815260048101849052602401610453565b828103610d1d57604051632158b7ff60e11b815260048101849052602401610453565b6000838152600160208190526040909120600701805460029260ff199091169083610b2e565b600081815260016020526040812060050154610d715760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090206008015490565b6040805161012081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e0810182905261010081019190915260008281526001602081815260409283902083516101208101855281548152928101549183019190915260028082015460ff9081161515948401949094526003820154606084015260048201546080840152600582015460a0840152600682015460c084015260078201549293919260e08501921690811115610e5557610e55611217565b6002811115610e6657610e66611217565b81526020016008820154815250509050919050565b600081815260016020526040812060050154610ea95760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090206006015490565b600081815260016020526040812060050154610eed5760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090205490565b600081815260016020526040812060050154610f2e5760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090206005015490565b600081815260016020526040902060050154610f725760405162461bcd60e51b815260040161045390611310565b600260008281526001602052604090206007015460ff166002811115610f9a57610f9a611217565b03610fe75760405162461bcd60e51b815260206004820152601a60248201527f417373657274696f6e20616c72656164792072656a65637465640000000000006044820152606401610453565b60008181526001602081905260409091200154156110435760405162461bcd60e51b815260206004820152601960248201527810da185b1b195b99d948185b1c9958591e4818dc99585d1959603a1b6044820152606401610453565b60008181526001602052604081206003015490036110ad5760405162461bcd60e51b815260206004820152602160248201527f4174206c656173742074776f206368696c6472656e206e6f74206372656174656044820152601960fa1b6064820152608401610453565b600280546110ba9161138a565b6000828152600160205260409020600401546110d69190611358565b421061111c5760405162461bcd60e51b8152602060048201526015602482015274546f6f206c61746520746f206368616c6c656e676560581b6044820152606401610453565b60005460405163f696dc5560e01b8152600481018390526001600160a01b039091169063f696dc55906024016020604051808303816000875af1158015611167573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061118b9190611371565b600091825260016020819052604090922090910155565b6000602082840312156111b457600080fd5b81356001600160a01b03811681146111cb57600080fd5b9392505050565b6000602082840312156111e457600080fd5b5035919050565b60008060006060848603121561120057600080fd5b505081359360208301359350604090920135919050565b634e487b7160e01b600052602160045260246000fd5b6003811061124b57634e487b7160e01b600052602160045260246000fd5b9052565b6000610120820190508251825260208301516020830152604083015115156040830152606083015160608301526080830151608083015260a083015160a083015260c083015160c083015260e08301516112ac60e084018261122d565b5061010092830151919092015290565b6000610120820190508a825289602083015288151560408301528760608301528660808301528560a08301528460c08301526112fb60e083018561122d565b826101008301529a9950505050505050505050565b602080825260189082015277105cdcd95c9d1a5bdb88191bd95cc81b9bdd08195e1a5cdd60421b604082015260600190565b634e487b7160e01b600052601160045260246000fd5b8082018082111561136b5761136b611342565b92915050565b60006020828403121561138357600080fd5b5051919050565b808202811582820484141761136b5761136b61134256fea2646970667358221220000e6758e0f66a6705c32861772d94213f4d182b3e6675f82283f1a3f0be967b64736f6c63430008110033",
}

// AssertionChainABI is the input ABI used to generate the binding from.
// Deprecated: Use AssertionChainMetaData.ABI instead.
var AssertionChainABI = AssertionChainMetaData.ABI

// AssertionChainBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AssertionChainMetaData.Bin instead.
var AssertionChainBin = AssertionChainMetaData.Bin

// DeployAssertionChain deploys a new Ethereum contract, binding an instance of AssertionChain to it.
func DeployAssertionChain(auth *bind.TransactOpts, backend bind.ContractBackend, stateHash [32]byte, _challengePeriodSeconds *big.Int) (common.Address, *types.Transaction, *AssertionChain, error) {
	parsed, err := AssertionChainMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AssertionChainBin), backend, stateHash, _challengePeriodSeconds)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AssertionChain{AssertionChainCaller: AssertionChainCaller{contract: contract}, AssertionChainTransactor: AssertionChainTransactor{contract: contract}, AssertionChainFilterer: AssertionChainFilterer{contract: contract}}, nil
}

// AssertionChain is an auto generated Go binding around an Ethereum contract.
type AssertionChain struct {
	AssertionChainCaller     // Read-only binding to the contract
	AssertionChainTransactor // Write-only binding to the contract
	AssertionChainFilterer   // Log filterer for contract events
}

// AssertionChainCaller is an auto generated read-only Go binding around an Ethereum contract.
type AssertionChainCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionChainTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AssertionChainTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionChainFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AssertionChainFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AssertionChainSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AssertionChainSession struct {
	Contract     *AssertionChain   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AssertionChainCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AssertionChainCallerSession struct {
	Contract *AssertionChainCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// AssertionChainTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AssertionChainTransactorSession struct {
	Contract     *AssertionChainTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// AssertionChainRaw is an auto generated low-level Go binding around an Ethereum contract.
type AssertionChainRaw struct {
	Contract *AssertionChain // Generic contract binding to access the raw methods on
}

// AssertionChainCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AssertionChainCallerRaw struct {
	Contract *AssertionChainCaller // Generic read-only contract binding to access the raw methods on
}

// AssertionChainTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AssertionChainTransactorRaw struct {
	Contract *AssertionChainTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAssertionChain creates a new instance of AssertionChain, bound to a specific deployed contract.
func NewAssertionChain(address common.Address, backend bind.ContractBackend) (*AssertionChain, error) {
	contract, err := bindAssertionChain(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AssertionChain{AssertionChainCaller: AssertionChainCaller{contract: contract}, AssertionChainTransactor: AssertionChainTransactor{contract: contract}, AssertionChainFilterer: AssertionChainFilterer{contract: contract}}, nil
}

// NewAssertionChainCaller creates a new read-only instance of AssertionChain, bound to a specific deployed contract.
func NewAssertionChainCaller(address common.Address, caller bind.ContractCaller) (*AssertionChainCaller, error) {
	contract, err := bindAssertionChain(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AssertionChainCaller{contract: contract}, nil
}

// NewAssertionChainTransactor creates a new write-only instance of AssertionChain, bound to a specific deployed contract.
func NewAssertionChainTransactor(address common.Address, transactor bind.ContractTransactor) (*AssertionChainTransactor, error) {
	contract, err := bindAssertionChain(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AssertionChainTransactor{contract: contract}, nil
}

// NewAssertionChainFilterer creates a new log filterer instance of AssertionChain, bound to a specific deployed contract.
func NewAssertionChainFilterer(address common.Address, filterer bind.ContractFilterer) (*AssertionChainFilterer, error) {
	contract, err := bindAssertionChain(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AssertionChainFilterer{contract: contract}, nil
}

// bindAssertionChain binds a generic wrapper to an already deployed contract.
func bindAssertionChain(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AssertionChainABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AssertionChain *AssertionChainRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AssertionChain.Contract.AssertionChainCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AssertionChain *AssertionChainRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionChain.Contract.AssertionChainTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AssertionChain *AssertionChainRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AssertionChain.Contract.AssertionChainTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AssertionChain *AssertionChainCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AssertionChain.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AssertionChain *AssertionChainTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionChain.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AssertionChain *AssertionChainTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AssertionChain.Contract.contract.Transact(opts, method, params...)
}

// AssertionExists is a free data retrieval call binding the contract method 0x5a4038f5.
//
// Solidity: function assertionExists(bytes32 assertionId) view returns(bool)
func (_AssertionChain *AssertionChainCaller) AssertionExists(opts *bind.CallOpts, assertionId [32]byte) (bool, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "assertionExists", assertionId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AssertionExists is a free data retrieval call binding the contract method 0x5a4038f5.
//
// Solidity: function assertionExists(bytes32 assertionId) view returns(bool)
func (_AssertionChain *AssertionChainSession) AssertionExists(assertionId [32]byte) (bool, error) {
	return _AssertionChain.Contract.AssertionExists(&_AssertionChain.CallOpts, assertionId)
}

// AssertionExists is a free data retrieval call binding the contract method 0x5a4038f5.
//
// Solidity: function assertionExists(bytes32 assertionId) view returns(bool)
func (_AssertionChain *AssertionChainCallerSession) AssertionExists(assertionId [32]byte) (bool, error) {
	return _AssertionChain.Contract.AssertionExists(&_AssertionChain.CallOpts, assertionId)
}

// Assertions is a free data retrieval call binding the contract method 0xd60715b5.
//
// Solidity: function assertions(bytes32 ) view returns(bytes32 predecessorId, bytes32 successionChallenge, bool isFirstChild, uint256 secondChildCreationTime, uint256 firstChildCreationTime, bytes32 stateHash, uint256 height, uint8 status, uint256 inboxMsgCountSeen)
func (_AssertionChain *AssertionChainCaller) Assertions(opts *bind.CallOpts, arg0 [32]byte) (struct {
	PredecessorId           [32]byte
	SuccessionChallenge     [32]byte
	IsFirstChild            bool
	SecondChildCreationTime *big.Int
	FirstChildCreationTime  *big.Int
	StateHash               [32]byte
	Height                  *big.Int
	Status                  uint8
	InboxMsgCountSeen       *big.Int
}, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "assertions", arg0)

	outstruct := new(struct {
		PredecessorId           [32]byte
		SuccessionChallenge     [32]byte
		IsFirstChild            bool
		SecondChildCreationTime *big.Int
		FirstChildCreationTime  *big.Int
		StateHash               [32]byte
		Height                  *big.Int
		Status                  uint8
		InboxMsgCountSeen       *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.PredecessorId = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.SuccessionChallenge = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.IsFirstChild = *abi.ConvertType(out[2], new(bool)).(*bool)
	outstruct.SecondChildCreationTime = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.FirstChildCreationTime = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.StateHash = *abi.ConvertType(out[5], new([32]byte)).(*[32]byte)
	outstruct.Height = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)
	outstruct.Status = *abi.ConvertType(out[7], new(uint8)).(*uint8)
	outstruct.InboxMsgCountSeen = *abi.ConvertType(out[8], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Assertions is a free data retrieval call binding the contract method 0xd60715b5.
//
// Solidity: function assertions(bytes32 ) view returns(bytes32 predecessorId, bytes32 successionChallenge, bool isFirstChild, uint256 secondChildCreationTime, uint256 firstChildCreationTime, bytes32 stateHash, uint256 height, uint8 status, uint256 inboxMsgCountSeen)
func (_AssertionChain *AssertionChainSession) Assertions(arg0 [32]byte) (struct {
	PredecessorId           [32]byte
	SuccessionChallenge     [32]byte
	IsFirstChild            bool
	SecondChildCreationTime *big.Int
	FirstChildCreationTime  *big.Int
	StateHash               [32]byte
	Height                  *big.Int
	Status                  uint8
	InboxMsgCountSeen       *big.Int
}, error) {
	return _AssertionChain.Contract.Assertions(&_AssertionChain.CallOpts, arg0)
}

// Assertions is a free data retrieval call binding the contract method 0xd60715b5.
//
// Solidity: function assertions(bytes32 ) view returns(bytes32 predecessorId, bytes32 successionChallenge, bool isFirstChild, uint256 secondChildCreationTime, uint256 firstChildCreationTime, bytes32 stateHash, uint256 height, uint8 status, uint256 inboxMsgCountSeen)
func (_AssertionChain *AssertionChainCallerSession) Assertions(arg0 [32]byte) (struct {
	PredecessorId           [32]byte
	SuccessionChallenge     [32]byte
	IsFirstChild            bool
	SecondChildCreationTime *big.Int
	FirstChildCreationTime  *big.Int
	StateHash               [32]byte
	Height                  *big.Int
	Status                  uint8
	InboxMsgCountSeen       *big.Int
}, error) {
	return _AssertionChain.Contract.Assertions(&_AssertionChain.CallOpts, arg0)
}

// ChallengeManagerAddr is a free data retrieval call binding the contract method 0x5625c360.
//
// Solidity: function challengeManagerAddr() view returns(address)
func (_AssertionChain *AssertionChainCaller) ChallengeManagerAddr(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "challengeManagerAddr")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ChallengeManagerAddr is a free data retrieval call binding the contract method 0x5625c360.
//
// Solidity: function challengeManagerAddr() view returns(address)
func (_AssertionChain *AssertionChainSession) ChallengeManagerAddr() (common.Address, error) {
	return _AssertionChain.Contract.ChallengeManagerAddr(&_AssertionChain.CallOpts)
}

// ChallengeManagerAddr is a free data retrieval call binding the contract method 0x5625c360.
//
// Solidity: function challengeManagerAddr() view returns(address)
func (_AssertionChain *AssertionChainCallerSession) ChallengeManagerAddr() (common.Address, error) {
	return _AssertionChain.Contract.ChallengeManagerAddr(&_AssertionChain.CallOpts)
}

// ChallengePeriodSeconds is a free data retrieval call binding the contract method 0xfb601294.
//
// Solidity: function challengePeriodSeconds() view returns(uint256)
func (_AssertionChain *AssertionChainCaller) ChallengePeriodSeconds(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "challengePeriodSeconds")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChallengePeriodSeconds is a free data retrieval call binding the contract method 0xfb601294.
//
// Solidity: function challengePeriodSeconds() view returns(uint256)
func (_AssertionChain *AssertionChainSession) ChallengePeriodSeconds() (*big.Int, error) {
	return _AssertionChain.Contract.ChallengePeriodSeconds(&_AssertionChain.CallOpts)
}

// ChallengePeriodSeconds is a free data retrieval call binding the contract method 0xfb601294.
//
// Solidity: function challengePeriodSeconds() view returns(uint256)
func (_AssertionChain *AssertionChainCallerSession) ChallengePeriodSeconds() (*big.Int, error) {
	return _AssertionChain.Contract.ChallengePeriodSeconds(&_AssertionChain.CallOpts)
}

// GetAssertion is a free data retrieval call binding the contract method 0x88302884.
//
// Solidity: function getAssertion(bytes32 id) view returns((bytes32,bytes32,bool,uint256,uint256,bytes32,uint256,uint8,uint256))
func (_AssertionChain *AssertionChainCaller) GetAssertion(opts *bind.CallOpts, id [32]byte) (Assertion, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "getAssertion", id)

	if err != nil {
		return *new(Assertion), err
	}

	out0 := *abi.ConvertType(out[0], new(Assertion)).(*Assertion)

	return out0, err

}

// GetAssertion is a free data retrieval call binding the contract method 0x88302884.
//
// Solidity: function getAssertion(bytes32 id) view returns((bytes32,bytes32,bool,uint256,uint256,bytes32,uint256,uint8,uint256))
func (_AssertionChain *AssertionChainSession) GetAssertion(id [32]byte) (Assertion, error) {
	return _AssertionChain.Contract.GetAssertion(&_AssertionChain.CallOpts, id)
}

// GetAssertion is a free data retrieval call binding the contract method 0x88302884.
//
// Solidity: function getAssertion(bytes32 id) view returns((bytes32,bytes32,bool,uint256,uint256,bytes32,uint256,uint8,uint256))
func (_AssertionChain *AssertionChainCallerSession) GetAssertion(id [32]byte) (Assertion, error) {
	return _AssertionChain.Contract.GetAssertion(&_AssertionChain.CallOpts, id)
}

// GetFirstChildCreationTime is a free data retrieval call binding the contract method 0x43ed6ad9.
//
// Solidity: function getFirstChildCreationTime(bytes32 assertionId) view returns(uint256)
func (_AssertionChain *AssertionChainCaller) GetFirstChildCreationTime(opts *bind.CallOpts, assertionId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "getFirstChildCreationTime", assertionId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetFirstChildCreationTime is a free data retrieval call binding the contract method 0x43ed6ad9.
//
// Solidity: function getFirstChildCreationTime(bytes32 assertionId) view returns(uint256)
func (_AssertionChain *AssertionChainSession) GetFirstChildCreationTime(assertionId [32]byte) (*big.Int, error) {
	return _AssertionChain.Contract.GetFirstChildCreationTime(&_AssertionChain.CallOpts, assertionId)
}

// GetFirstChildCreationTime is a free data retrieval call binding the contract method 0x43ed6ad9.
//
// Solidity: function getFirstChildCreationTime(bytes32 assertionId) view returns(uint256)
func (_AssertionChain *AssertionChainCallerSession) GetFirstChildCreationTime(assertionId [32]byte) (*big.Int, error) {
	return _AssertionChain.Contract.GetFirstChildCreationTime(&_AssertionChain.CallOpts, assertionId)
}

// GetHeight is a free data retrieval call binding the contract method 0x896efbf2.
//
// Solidity: function getHeight(bytes32 assertionId) view returns(uint256)
func (_AssertionChain *AssertionChainCaller) GetHeight(opts *bind.CallOpts, assertionId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "getHeight", assertionId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetHeight is a free data retrieval call binding the contract method 0x896efbf2.
//
// Solidity: function getHeight(bytes32 assertionId) view returns(uint256)
func (_AssertionChain *AssertionChainSession) GetHeight(assertionId [32]byte) (*big.Int, error) {
	return _AssertionChain.Contract.GetHeight(&_AssertionChain.CallOpts, assertionId)
}

// GetHeight is a free data retrieval call binding the contract method 0x896efbf2.
//
// Solidity: function getHeight(bytes32 assertionId) view returns(uint256)
func (_AssertionChain *AssertionChainCallerSession) GetHeight(assertionId [32]byte) (*big.Int, error) {
	return _AssertionChain.Contract.GetHeight(&_AssertionChain.CallOpts, assertionId)
}

// GetInboxMsgCountSeen is a free data retrieval call binding the contract method 0x7cfd5ab9.
//
// Solidity: function getInboxMsgCountSeen(bytes32 assertionId) view returns(uint256)
func (_AssertionChain *AssertionChainCaller) GetInboxMsgCountSeen(opts *bind.CallOpts, assertionId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "getInboxMsgCountSeen", assertionId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetInboxMsgCountSeen is a free data retrieval call binding the contract method 0x7cfd5ab9.
//
// Solidity: function getInboxMsgCountSeen(bytes32 assertionId) view returns(uint256)
func (_AssertionChain *AssertionChainSession) GetInboxMsgCountSeen(assertionId [32]byte) (*big.Int, error) {
	return _AssertionChain.Contract.GetInboxMsgCountSeen(&_AssertionChain.CallOpts, assertionId)
}

// GetInboxMsgCountSeen is a free data retrieval call binding the contract method 0x7cfd5ab9.
//
// Solidity: function getInboxMsgCountSeen(bytes32 assertionId) view returns(uint256)
func (_AssertionChain *AssertionChainCallerSession) GetInboxMsgCountSeen(assertionId [32]byte) (*big.Int, error) {
	return _AssertionChain.Contract.GetInboxMsgCountSeen(&_AssertionChain.CallOpts, assertionId)
}

// GetPredecessorId is a free data retrieval call binding the contract method 0x9ca565d4.
//
// Solidity: function getPredecessorId(bytes32 assertionId) view returns(bytes32)
func (_AssertionChain *AssertionChainCaller) GetPredecessorId(opts *bind.CallOpts, assertionId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "getPredecessorId", assertionId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetPredecessorId is a free data retrieval call binding the contract method 0x9ca565d4.
//
// Solidity: function getPredecessorId(bytes32 assertionId) view returns(bytes32)
func (_AssertionChain *AssertionChainSession) GetPredecessorId(assertionId [32]byte) ([32]byte, error) {
	return _AssertionChain.Contract.GetPredecessorId(&_AssertionChain.CallOpts, assertionId)
}

// GetPredecessorId is a free data retrieval call binding the contract method 0x9ca565d4.
//
// Solidity: function getPredecessorId(bytes32 assertionId) view returns(bytes32)
func (_AssertionChain *AssertionChainCallerSession) GetPredecessorId(assertionId [32]byte) ([32]byte, error) {
	return _AssertionChain.Contract.GetPredecessorId(&_AssertionChain.CallOpts, assertionId)
}

// GetStateHash is a free data retrieval call binding the contract method 0xf9bce634.
//
// Solidity: function getStateHash(bytes32 assertionId) view returns(bytes32)
func (_AssertionChain *AssertionChainCaller) GetStateHash(opts *bind.CallOpts, assertionId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "getStateHash", assertionId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetStateHash is a free data retrieval call binding the contract method 0xf9bce634.
//
// Solidity: function getStateHash(bytes32 assertionId) view returns(bytes32)
func (_AssertionChain *AssertionChainSession) GetStateHash(assertionId [32]byte) ([32]byte, error) {
	return _AssertionChain.Contract.GetStateHash(&_AssertionChain.CallOpts, assertionId)
}

// GetStateHash is a free data retrieval call binding the contract method 0xf9bce634.
//
// Solidity: function getStateHash(bytes32 assertionId) view returns(bytes32)
func (_AssertionChain *AssertionChainCallerSession) GetStateHash(assertionId [32]byte) ([32]byte, error) {
	return _AssertionChain.Contract.GetStateHash(&_AssertionChain.CallOpts, assertionId)
}

// GetSuccessionChallenge is a free data retrieval call binding the contract method 0x2fefa18a.
//
// Solidity: function getSuccessionChallenge(bytes32 assertionId) view returns(bytes32)
func (_AssertionChain *AssertionChainCaller) GetSuccessionChallenge(opts *bind.CallOpts, assertionId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "getSuccessionChallenge", assertionId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetSuccessionChallenge is a free data retrieval call binding the contract method 0x2fefa18a.
//
// Solidity: function getSuccessionChallenge(bytes32 assertionId) view returns(bytes32)
func (_AssertionChain *AssertionChainSession) GetSuccessionChallenge(assertionId [32]byte) ([32]byte, error) {
	return _AssertionChain.Contract.GetSuccessionChallenge(&_AssertionChain.CallOpts, assertionId)
}

// GetSuccessionChallenge is a free data retrieval call binding the contract method 0x2fefa18a.
//
// Solidity: function getSuccessionChallenge(bytes32 assertionId) view returns(bytes32)
func (_AssertionChain *AssertionChainCallerSession) GetSuccessionChallenge(assertionId [32]byte) ([32]byte, error) {
	return _AssertionChain.Contract.GetSuccessionChallenge(&_AssertionChain.CallOpts, assertionId)
}

// IsFirstChild is a free data retrieval call binding the contract method 0x30836228.
//
// Solidity: function isFirstChild(bytes32 assertionId) view returns(bool)
func (_AssertionChain *AssertionChainCaller) IsFirstChild(opts *bind.CallOpts, assertionId [32]byte) (bool, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "isFirstChild", assertionId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsFirstChild is a free data retrieval call binding the contract method 0x30836228.
//
// Solidity: function isFirstChild(bytes32 assertionId) view returns(bool)
func (_AssertionChain *AssertionChainSession) IsFirstChild(assertionId [32]byte) (bool, error) {
	return _AssertionChain.Contract.IsFirstChild(&_AssertionChain.CallOpts, assertionId)
}

// IsFirstChild is a free data retrieval call binding the contract method 0x30836228.
//
// Solidity: function isFirstChild(bytes32 assertionId) view returns(bool)
func (_AssertionChain *AssertionChainCallerSession) IsFirstChild(assertionId [32]byte) (bool, error) {
	return _AssertionChain.Contract.IsFirstChild(&_AssertionChain.CallOpts, assertionId)
}

// StakeAmount is a free data retrieval call binding the contract method 0x60c7dc47.
//
// Solidity: function stakeAmount() view returns(uint256)
func (_AssertionChain *AssertionChainCaller) StakeAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AssertionChain.contract.Call(opts, &out, "stakeAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StakeAmount is a free data retrieval call binding the contract method 0x60c7dc47.
//
// Solidity: function stakeAmount() view returns(uint256)
func (_AssertionChain *AssertionChainSession) StakeAmount() (*big.Int, error) {
	return _AssertionChain.Contract.StakeAmount(&_AssertionChain.CallOpts)
}

// StakeAmount is a free data retrieval call binding the contract method 0x60c7dc47.
//
// Solidity: function stakeAmount() view returns(uint256)
func (_AssertionChain *AssertionChainCallerSession) StakeAmount() (*big.Int, error) {
	return _AssertionChain.Contract.StakeAmount(&_AssertionChain.CallOpts)
}

// AddStake is a paid mutator transaction binding the contract method 0x5a627dbc.
//
// Solidity: function addStake() payable returns()
func (_AssertionChain *AssertionChainTransactor) AddStake(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AssertionChain.contract.Transact(opts, "addStake")
}

// AddStake is a paid mutator transaction binding the contract method 0x5a627dbc.
//
// Solidity: function addStake() payable returns()
func (_AssertionChain *AssertionChainSession) AddStake() (*types.Transaction, error) {
	return _AssertionChain.Contract.AddStake(&_AssertionChain.TransactOpts)
}

// AddStake is a paid mutator transaction binding the contract method 0x5a627dbc.
//
// Solidity: function addStake() payable returns()
func (_AssertionChain *AssertionChainTransactorSession) AddStake() (*types.Transaction, error) {
	return _AssertionChain.Contract.AddStake(&_AssertionChain.TransactOpts)
}

// ConfirmAssertion is a paid mutator transaction binding the contract method 0x6894bdd5.
//
// Solidity: function confirmAssertion(bytes32 assertionId) returns()
func (_AssertionChain *AssertionChainTransactor) ConfirmAssertion(opts *bind.TransactOpts, assertionId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.contract.Transact(opts, "confirmAssertion", assertionId)
}

// ConfirmAssertion is a paid mutator transaction binding the contract method 0x6894bdd5.
//
// Solidity: function confirmAssertion(bytes32 assertionId) returns()
func (_AssertionChain *AssertionChainSession) ConfirmAssertion(assertionId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.Contract.ConfirmAssertion(&_AssertionChain.TransactOpts, assertionId)
}

// ConfirmAssertion is a paid mutator transaction binding the contract method 0x6894bdd5.
//
// Solidity: function confirmAssertion(bytes32 assertionId) returns()
func (_AssertionChain *AssertionChainTransactorSession) ConfirmAssertion(assertionId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.Contract.ConfirmAssertion(&_AssertionChain.TransactOpts, assertionId)
}

// CreateNewAssertion is a paid mutator transaction binding the contract method 0x49635f9a.
//
// Solidity: function createNewAssertion(bytes32 stateHash, uint256 height, bytes32 predecessorId) returns()
func (_AssertionChain *AssertionChainTransactor) CreateNewAssertion(opts *bind.TransactOpts, stateHash [32]byte, height *big.Int, predecessorId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.contract.Transact(opts, "createNewAssertion", stateHash, height, predecessorId)
}

// CreateNewAssertion is a paid mutator transaction binding the contract method 0x49635f9a.
//
// Solidity: function createNewAssertion(bytes32 stateHash, uint256 height, bytes32 predecessorId) returns()
func (_AssertionChain *AssertionChainSession) CreateNewAssertion(stateHash [32]byte, height *big.Int, predecessorId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.Contract.CreateNewAssertion(&_AssertionChain.TransactOpts, stateHash, height, predecessorId)
}

// CreateNewAssertion is a paid mutator transaction binding the contract method 0x49635f9a.
//
// Solidity: function createNewAssertion(bytes32 stateHash, uint256 height, bytes32 predecessorId) returns()
func (_AssertionChain *AssertionChainTransactorSession) CreateNewAssertion(stateHash [32]byte, height *big.Int, predecessorId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.Contract.CreateNewAssertion(&_AssertionChain.TransactOpts, stateHash, height, predecessorId)
}

// CreateSuccessionChallenge is a paid mutator transaction binding the contract method 0xff8aef87.
//
// Solidity: function createSuccessionChallenge(bytes32 assertionId) returns()
func (_AssertionChain *AssertionChainTransactor) CreateSuccessionChallenge(opts *bind.TransactOpts, assertionId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.contract.Transact(opts, "createSuccessionChallenge", assertionId)
}

// CreateSuccessionChallenge is a paid mutator transaction binding the contract method 0xff8aef87.
//
// Solidity: function createSuccessionChallenge(bytes32 assertionId) returns()
func (_AssertionChain *AssertionChainSession) CreateSuccessionChallenge(assertionId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.Contract.CreateSuccessionChallenge(&_AssertionChain.TransactOpts, assertionId)
}

// CreateSuccessionChallenge is a paid mutator transaction binding the contract method 0xff8aef87.
//
// Solidity: function createSuccessionChallenge(bytes32 assertionId) returns()
func (_AssertionChain *AssertionChainTransactorSession) CreateSuccessionChallenge(assertionId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.Contract.CreateSuccessionChallenge(&_AssertionChain.TransactOpts, assertionId)
}

// RejectAssertion is a paid mutator transaction binding the contract method 0x75dc6098.
//
// Solidity: function rejectAssertion(bytes32 assertionId) returns()
func (_AssertionChain *AssertionChainTransactor) RejectAssertion(opts *bind.TransactOpts, assertionId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.contract.Transact(opts, "rejectAssertion", assertionId)
}

// RejectAssertion is a paid mutator transaction binding the contract method 0x75dc6098.
//
// Solidity: function rejectAssertion(bytes32 assertionId) returns()
func (_AssertionChain *AssertionChainSession) RejectAssertion(assertionId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.Contract.RejectAssertion(&_AssertionChain.TransactOpts, assertionId)
}

// RejectAssertion is a paid mutator transaction binding the contract method 0x75dc6098.
//
// Solidity: function rejectAssertion(bytes32 assertionId) returns()
func (_AssertionChain *AssertionChainTransactorSession) RejectAssertion(assertionId [32]byte) (*types.Transaction, error) {
	return _AssertionChain.Contract.RejectAssertion(&_AssertionChain.TransactOpts, assertionId)
}

// UpdateChallengeManager is a paid mutator transaction binding the contract method 0x295dfd32.
//
// Solidity: function updateChallengeManager(address _challengeManager) returns()
func (_AssertionChain *AssertionChainTransactor) UpdateChallengeManager(opts *bind.TransactOpts, _challengeManager common.Address) (*types.Transaction, error) {
	return _AssertionChain.contract.Transact(opts, "updateChallengeManager", _challengeManager)
}

// UpdateChallengeManager is a paid mutator transaction binding the contract method 0x295dfd32.
//
// Solidity: function updateChallengeManager(address _challengeManager) returns()
func (_AssertionChain *AssertionChainSession) UpdateChallengeManager(_challengeManager common.Address) (*types.Transaction, error) {
	return _AssertionChain.Contract.UpdateChallengeManager(&_AssertionChain.TransactOpts, _challengeManager)
}

// UpdateChallengeManager is a paid mutator transaction binding the contract method 0x295dfd32.
//
// Solidity: function updateChallengeManager(address _challengeManager) returns()
func (_AssertionChain *AssertionChainTransactorSession) UpdateChallengeManager(_challengeManager common.Address) (*types.Transaction, error) {
	return _AssertionChain.Contract.UpdateChallengeManager(&_AssertionChain.TransactOpts, _challengeManager)
}

// ChallengeManagerImplMetaData contains all meta data concerning the ChallengeManagerImpl contract.
var ChallengeManagerImplMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_miniStakeValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriodSec\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"actualLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxProofLength\",\"type\":\"uint256\"}],\"name\":\"MerkleProofTooLong\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"fromId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"toId\",\"type\":\"bytes32\"}],\"name\":\"Bisected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"ChallengeCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"fromId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"toId\",\"type\":\"bytes32\"}],\"name\":\"Merged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"vertexId\",\"type\":\"bytes32\"}],\"name\":\"VertexAdded\",\"type\":\"event\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"historyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"firstState\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"firstStatehistoryProof\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32\",\"name\":\"lastState\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"lastStatehistoryProof\",\"type\":\"bytes32[]\"}],\"internalType\":\"structAddLeafArgs\",\"name\":\"leafData\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof1\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof2\",\"type\":\"bytes\"}],\"name\":\"addLeaf\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"assertionChain\",\"outputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"prefixHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"bisect\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"},{\"internalType\":\"enumChallengeType\",\"name\":\"typ\",\"type\":\"uint8\"}],\"name\":\"calculateChallengeId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"commitmentMerkle\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"commitmentHeight\",\"type\":\"uint256\"}],\"name\":\"calculateChallengeVertexId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"challengeExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"challengePeriodSec\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"challenges\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"winningClaim\",\"type\":\"bytes32\"},{\"internalType\":\"enumChallengeType\",\"name\":\"challengeType\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"challenger\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"childrenAreAtOneStepFork\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"confirmForPsTimer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"confirmForSucessionChallengeWin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"createChallenge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"createSubChallenge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"winnerVId\",\"type\":\"bytes32\"},{\"components\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"machineStep\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"internalType\":\"structOneStepData\",\"name\":\"oneStepData\",\"type\":\"tuple\"},{\"internalType\":\"bytes32[]\",\"name\":\"beforeHistoryInclusionProof\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"afterHistoryInclusionProof\",\"type\":\"bytes32[]\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"getChallenge\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"rootId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"winningClaim\",\"type\":\"bytes32\"},{\"internalType\":\"enumChallengeType\",\"name\":\"challengeType\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"challenger\",\"type\":\"address\"}],\"internalType\":\"structChallenge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"getCurrentPsTimer\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"getVertex\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"historyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"successionChallenge\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"predecessorId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"enumVertexStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"psId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"psLastUpdatedTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"flushedPsTimeSec\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"lowestHeightSuccessorId\",\"type\":\"bytes32\"}],\"internalType\":\"structChallengeVertex\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"hasConfirmedSibling\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_miniStakeValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriodSec\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"isPresumptiveSuccessor\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"prefixHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"merge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"miniStakeValue\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"vertexExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"vertices\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"historyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"successionChallenge\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"predecessorId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"enumVertexStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"psId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"psLastUpdatedTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"flushedPsTimeSec\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"lowestHeightSuccessorId\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"winningClaim\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b5060405162005b6d38038062005b6d8339810160408190526200003491620000ec565b62000042848484846200004c565b505050506200013d565b6002546001600160a01b031615620000995760405162461bcd60e51b815260206004820152600c60248201526b1053149150511657d253925560a21b604482015260640160405180910390fd5b600280546001600160a01b039586166001600160a01b03199182161790915560049390935560059190915560038054919093169116179055565b6001600160a01b0381168114620000e957600080fd5b50565b600080600080608085870312156200010357600080fd5b84516200011081620000d3565b80945050602085015192506040850151915060608501516200013281620000d3565b939692955090935050565b615a20806200014d6000396000f3fe6080604052600436106101565760003560e01c806373d154e8116100c15780639e7cee541161007a5780639e7cee54146103fe578063bd62325114610411578063c1e69b6614610431578063d1bac9a41461048f578063e41b5058146104af578063f4f81db2146104cf578063f696dc551461056c57600080fd5b806373d154e8146103205780637a4d47dc1461035157806386f048ed146103715780638ac043491461039e57806398b67d59146103be5780639e3d87cd146103de57600080fd5b806348dd29241161011357806348dd29241461025c5780634a65878814610294578063597e1e0b146102b457806359c69996146102d4578063654f0dc2146102ea5780636b0b25921461030057600080fd5b80631421ba3d1461015b57806316ef55341461018e5780631b7bbecb146101ae5780631d5618ac146101ed578063359076cf1461020f578063458d2bf11461022f575b600080fd5b34801561016757600080fd5b5061017b610176366004614c67565b61058c565b6040519081526020015b60405180910390f35b34801561019a57600080fd5b5061017b6101a9366004614d0c565b6105d1565b3480156101ba57600080fd5b506101dd6101c9366004614d40565b600090815260016020526040902054151590565b6040519015158152602001610185565b3480156101f957600080fd5b5061020d610208366004614d40565b6105e6565b005b34801561021b57600080fd5b5061017b61022a366004614dc8565b610600565b34801561023b57600080fd5b5061024f61024a366004614d40565b6106cd565b6040516101859190614e99565b34801561026857600080fd5b5060025461027c906001600160a01b031681565b6040516001600160a01b039091168152602001610185565b3480156102a057600080fd5b5061017b6102af366004614ed9565b6107ca565b3480156102c057600080fd5b5061017b6102cf366004614dc8565b6107df565b3480156102e057600080fd5b5061017b60045481565b3480156102f657600080fd5b5061017b60055481565b34801561030c57600080fd5b506101dd61031b366004614d40565b610877565b34801561032c57600080fd5b5061017b61033b366004614d40565b6000908152600160208190526040909120015490565b34801561035d57600080fd5b506101dd61036c366004614d40565b610890565b34801561037d57600080fd5b5061039161038c366004614d40565b6108a4565b6040516101859190614f15565b3480156103aa57600080fd5b5061017b6103b9366004614d40565b6109a6565b3480156103ca57600080fd5b506101dd6103d9366004614d40565b6109b2565b3480156103ea57600080fd5b5061020d6103f9366004614fca565b610b81565b61017b61040c366004615055565b610c03565b34801561041d57600080fd5b5061017b61042c366004614d40565b610f2d565b34801561043d57600080fd5b5061047f61044c366004614d40565b600160208190526000918252604090912080549181015460029091015460ff81169061010090046001600160a01b031684565b60405161018594939291906150f2565b34801561049b57600080fd5b5061020d6104aa366004614d40565b61110e565b3480156104bb57600080fd5b506101dd6104ca366004614d40565b61111b565b3480156104db57600080fd5b506105546104ea366004614d40565b600060208190529081526040902080546001820154600283015460038401546004850154600586015460068701546007880154600889015460098a0154600a909a01549899979896979596949593946001600160a01b03841694600160a01b90940460ff1693908c565b6040516101859c9b9a99989796959493929190615127565b34801561057857600080fd5b5061017b610587366004614d40565b61119b565b60035460009081906105b19082906001906001600160a01b03168b8b8b8b8b8b611415565b600090815260016020819052604090912001979097559695505050505050565b60006105dd838361172b565b90505b92915050565b6105f460008260055461175e565b6105fd8161184b565b50565b6000806000610614600060018888886118a6565b600088815260208190526040812060040154929450909250610636818961192b565b6000898152602081905260408120549192509061065590898685611a36565b905061067181846005546000611b02909392919063ffffffff16565b506005546106859060009087908c90611c3e565b604080518a8152602081018790527f69d5465c81edf7aaaf2e5c6c8829500df87d84c87f8d5b1221b59eaeaca70d27910160405180910390a1509293505050505b9392505050565b60408051608081018252600080825260208201819052918101829052606081019190915260008281526001602052604090205461074c5760405162461bcd60e51b815260206004820152601860248201527710da185b1b195b99d948191bd95cc81b9bdd08195e1a5cdd60421b60448201526064015b60405180910390fd5b60008281526001602081815260409283902083516080810185528154815292810154918301919091526002810154919290919083019060ff16600381111561079657610796614e6f565b60038111156107a7576107a7614e6f565b81526002919091015461010090046001600160a01b031660209091015292915050565b60006107d78484846120e2565b949350505050565b6000806107f160006001878787612119565b50905061080e81866005546000611c3e909392919063ffffffff16565b60008181526020819052604080822060040154878352908220600901546108369291906121fc565b60408051868152602081018390527f72b50597145599e4288d411331c925b40b33b0fa3cccadc1f57d2a1ab973553a910160405180910390a1949350505050565b60008181526020819052604081206001015415156105e0565b600061089c8183612349565b506001919050565b6108ac614bbb565b6000828152602081905260409020600101546108da5760405162461bcd60e51b81526004016107439061519b565b6000828152602081815260409182902082516101808101845281548152600180830154938201939093526002820154938101939093526003810154606084015260048101546080840152600581015460a084015260068101546001600160a01b03811660c0850152909160e0840191600160a01b900460ff169081111561096357610963614e6f565b600181111561097457610974614e6f565b8152600782015460208201526008820154604082015260098201546060820152600a9091015460809091015292915050565b60006105e0818361192b565b6000818152602081905260408120600101546109e05760405162461bcd60e51b81526004016107439061519b565b60008281526020819052604080822060040154808352912060010154610a185760405162461bcd60e51b8152600401610743906151ca565b6000818152602081905260409020600301548015610af457600081815260016020819052604090912001548015610af257600081815260208190526040902060010154610aa75760405162461bcd60e51b815260206004820152601c60248201527f57696e6e696e6720636c61696d20646f6573206e6f74206578697374000000006044820152606401610743565b848103610ab957506000949350505050565b6001600082815260208190526040902060060154600160a01b900460ff166001811115610ae857610ae8614e6f565b1495945050505050565b505b6000828152602081905260409020600701548015610b7657600081815260208190526040902060010154610aa75760405162461bcd60e51b8152602060048201526024808201527f50726573756d707469766520737563636573736f7220646f6573206e6f7420656044820152631e1a5cdd60e21b6064820152608401610743565b506000949350505050565b6002546001600160a01b031615610bc95760405162461bcd60e51b815260206004820152600c60248201526b1053149150511657d253925560a21b6044820152606401610743565b600280546001600160a01b039586166001600160a01b03199182161790915560049390935560059190915560038054919093169116179055565b600080863560009081526001602052604090206002015460ff166003811115610c2e57610c2e614e6f565b03610d2c576000610cea600060016040518060a00160405280600454815260200160055481526020018b610c619061528f565b81526020018a8a8080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250505090825250604080516020601f8b0181900481028201810190925289815291810191908a908a908190840183828082843760009201919091525050509152506002546001600160a01b0316612569565b90507f4383ba11a7cd16be5880c5f674b93be38b3b1fcafd7a7b06151998fa2a67534981604051610d1d91815260200190565b60405180910390a19050610f24565b6001863560009081526001602052604090206002015460ff166003811115610d5657610d56614e6f565b03610e06576000610cea600060016040518060a00160405280600454815260200160055481526020018b610d899061528f565b81526020018a8a8080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250505090825250604080516020601f8b0181900481028201810190925289815291810191908a908a9081908401838280828437600092019190915250505091525061287b565b6002863560009081526001602052604090206002015460ff166003811115610e3057610e30614e6f565b03610ee0576000610cea600060016040518060a00160405280600454815260200160055481526020018b610e639061528f565b81526020018a8a8080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250505090825250604080516020601f8b0181900481028201810190925289815291810191908a908a908190840183828082843760009201919091525050509152506129e6565b60405162461bcd60e51b8152602060048201526019602482015278556e6578706563746564206368616c6c656e6765207479706560381b6044820152606401610743565b95945050505050565b6000806000610f426000600186600554612b35565b600086815260208190526040812060010154929450909250610f65848388612c89565b90506000610f7282612d34565b600081815260208181526040918290208551815590850151600180830191909155918501516002820155606085015160038201556080850151600482015560a0850151600582015560c08501516006820180546001600160a01b039092166001600160a01b031983168117825560e08801519596508795939491926001600160a81b0319161790600160a01b90849081111561101057611010614e6f565b021790555061010082015160078201556101208201516008820155610140820151600982015561016090910151600a90910155604080516080810182528281526000602082015290810185600381111561106c5761106c614e6f565b81523360209182015260008781526001808352604091829020845181559284015183820155908301516002830180549192909160ff1916908360038111156110b6576110b6614e6f565b021790555060609190910151600290910180546001600160a01b0390921661010002610100600160a81b031990921691909117905560008781526020819052604090206111039086612d4d565b509295945050505050565b6105f46000600183612d9c565b6000818152602081905260408120600101546111495760405162461bcd60e51b81526004016107439061519b565b600082815260208190526040808220600401548083529120600101546111815760405162461bcd60e51b815260040161074390615338565b600090815260208190526040902060070154909114919050565b60025460009081906111ba9060019085906001600160a01b0316612e67565b600254604051633e6f398d60e21b8152600481018690529192506000916001600160a01b039091169063f9bce63490602401602060405180830381865afa158015611209573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061122d9190615379565b90506000611264838360405160200161124891815260200190565b6040516020818303038152906040528051906020012087612c89565b9050600061127182612d34565b600081815260208181526040918290208551815590850151600180830191909155918501516002820155606085015160038201556080850151600482015560a0850151600582015560c08501516006820180546001600160a01b039092166001600160a01b031983168117825560e08801519596508795939491926001600160a81b0319161790600160a01b90849081111561130f5761130f614e6f565b021790555061010082015160078201556101208201516008820155610140820151600982015561016090910151600a90910155604080516080810182528281526000602080830182815283850183815233606086015289845260019283905294909220835181559151828201559251600282018054939492939192909160ff1916908360038111156113a3576113a3614e6f565b021790555060609190910151600290910180546001600160a01b0390921661010002610100600160a81b03199092169190911790556040518481527f867c977ac47adb20fcc4fb6b981269b44d23560057a29eed03cd5afb81750b349060200160405180910390a15091949350505050565b600086815260208a905260408120600101546114435760405162461bcd60e51b81526004016107439061519b565b600087815260208b905260408082206004015480835291206001015461147b5760405162461bcd60e51b8152600401610743906151ca565b600081815260208c90526040812060030154908190036114ad5760405162461bcd60e51b815260040161074390615392565b6003600082815260208d9052604090206002015460ff1660038111156114d5576114d5614e6f565b146115375760405162461bcd60e51b815260206004820152602c60248201527f4368616c6c656e6765206973206e6f74206174206f6e6520737465702065786560448201526b18dd5d1a5bdb881c1bda5b9d60a21b6064820152608401610743565b6115938c60008481526020019081526020016000206001015489606001358a604001358a8a80806020026020016040519081016040528093929190818152602001838360200280828437600092019190915250612f1092505050565b6115df5760405162461bcd60e51b815260206004820152601b60248201527f4265666f7265207374617465206e6f7420696e20686973746f727900000000006044820152606401610743565b60006001600160a01b038b16635d3adcfb8a6040810135606082013561160860808401846153d5565b6040518663ffffffff1660e01b815260040161162895949392919061541b565b602060405180830381865afa158015611645573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906116699190615379565b90506116cf8d60008c815260200190815260200160002060010154828b6040013560016116969190615498565b898980806020026020016040519081016040528093929190818152602001838360200280828437600092019190915250612f1092505050565b61171b5760405162461bcd60e51b815260206004820152601a60248201527f4166746572207374617465206e6f7420696e20686973746f72790000000000006044820152606401610743565b509b9a5050505050505050505050565b600082826040516020016117409291906154ab565b60405160208183030381529060405280519060200120905092915050565b6117688383612f52565b600082815260208490526040808220600401548252902060030154156117dc5760405162461bcd60e51b815260206004820152602360248201527f53756363657373696f6e206368616c6c656e676520616c7265616479206f70656044820152621b995960ea1b6064820152608401610743565b806117e7848461192b565b116118465760405162461bcd60e51b815260206004820152602960248201527f507354696d6572206e6f742067726561746572207468616e206368616c6c656e60448201526819d9481c195c9a5bd960ba1b6064820152608401610743565b505050565b6000818152602081905260409020611862906130a3565b600081815260208190526040902080549061187c90613163565b156118a25760008281526020818152604080832060050154848452600192839052922001555b5050565b6000806000806118b989898989896131eb565b600082815260208c9052604090206001015491935091501561191d5760405162461bcd60e51b815260206004820152601f60248201527f426973656374696f6e2076657274657820616c726561647920657869737473006044820152606401610743565b909890975095505050505050565b6000818152602083905260408120600101546119945760405162461bcd60e51b815260206004820152602260248201527f56657274657820646f6573206e6f7420657869737420666f722070732074696d60448201526132b960f11b6064820152608401610743565b600082815260208490526040808220600401548083529120600101546119cc5760405162461bcd60e51b815260040161074390615338565b600081815260208590526040902060070154839003611a1d5760008381526020859052604080822060090154838352912060080154611a0b90426154d4565b611a159190615498565b9150506105e0565b50506000818152602083905260409020600901546105e0565b611a3e614bbb565b6000859003611a5f5760405162461bcd60e51b8152600401610743906154e7565b6000849003611a805760405162461bcd60e51b815260040161074390615512565b82600003611aa05760405162461bcd60e51b81526004016107439061553d565b5060408051610180810182529485526020850193909352918301526000606083018190526080830181905260a0830181905260c0830181905260e083018190526101008301819052610120830181905261014083019190915261016082015290565b600080611b0e85612d34565b60008181526020889052604090206001015490915015611b685760405162461bcd60e51b815260206004820152601560248201527456657274657820616c72656164792065786973747360581b6044820152606401610743565b600081815260208781526040918290208751815590870151600180830191909155918701516002820155606087015160038201556080870151600482015560a0870151600582015560c08701516006820180546001600160a01b039092166001600160a01b031983168117825560e08a01518a9590936001600160a81b03191690911790600160a01b908490811115611c0357611c03614e6f565b021790555061010082015160078201556101208201516008820155610140820151600982015561016090910151600a90910155610f24868583865b600083815260208590526040902060010154611c9c5760405162461bcd60e51b815260206004820152601b60248201527f53746172742076657274657820646f6573206e6f7420657869737400000000006044820152606401610743565b6000838152602085905260409020611cb390613163565b15611d0c5760405162461bcd60e51b8152602060048201526024808201527f43616e6e6f7420636f6e6e656374206120737563636573736f7220746f2061206044820152633632b0b360e11b6064820152608401610743565b600082815260208590526040902060010154611d665760405162461bcd60e51b8152602060048201526019602482015278115b99081d995c9d195e08191bd95cc81b9bdd08195e1a5cdd603a1b6044820152606401610743565b600082815260208590526040902060040154839003611dc75760405162461bcd60e51b815260206004820152601a60248201527f566572746963657320616c726561647920636f6e6e65637465640000000000006044820152606401610743565b600082815260208590526040808220600290810154868452919092209091015410611e435760405162461bcd60e51b815260206004820152602660248201527f537461727420686569676874206e6f74206c6f776572207468616e20656e64206044820152651a195a59da1d60d21b6064820152608401610743565b6000828152602085905260408082205485835291205414611ec45760405162461bcd60e51b815260206004820152603560248201527f5072656465636573736f7220616e6420737563636573736f722061726520696e60448201527420646966666572656e74206368616c6c656e67657360581b6064820152608401610743565b6000828152602085905260409020611edc9084613390565b6000838152602085905260408120600a01549003611f1d57611f00848460006121fc565b6000838152602085905260409020611f18908361345c565b6120dc565b600082815260208590526040808220600290810154868452828420600a0154845291909220909101548082101561201157611f59868685613537565b15611fe65760405162461bcd60e51b815260206004820152605160248201527f5374617274207665727465782068617320707320776974682074696d6572206760448201527f726561746572207468616e206368616c6c656e676520706572696f642c2063616064820152706e6e6f7420736574206c6f77657220707360781b608482015260a401610743565b611ff2868660006121fc565b600085815260208790526040902061200a908561345c565b50506120dc565b8082036120d957612023868685613537565b156120b65760405162461bcd60e51b815260206004820152605760248201527f5374617274207665727465782068617320707320776974682074696d6572206760448201527f726561746572207468616e206368616c6c656e676520706572696f642c2063616064820152766e6e6f74207365742073616d652068656967687420707360481b608482015260a401610743565b6120c2868660006121fc565b600085815260208790526040812061200a9161345c565b50505b50505050565b6040805160208082019590955280820193909352606080840192909252805180840390920182526080909201909152805191012090565b60008060008061212c89898989896131eb565b600082815260208c90526040902060010154919350915061219f5760405162461bcd60e51b815260206004820152602760248201527f426973656374696f6e2076657274657820646f6573206e6f7420616c726561646044820152661e48195e1a5cdd60ca1b6064820152608401610743565b600082815260208a9052604090206121b690613163565b1561191d5760405162461bcd60e51b815260206004820152601660248201527521b0b73737ba1036b2b933b2903a379030903632b0b360511b6044820152606401610743565b60008281526020849052604090206001015461222a5760405162461bcd60e51b81526004016107439061519b565b600082815260208490526040902061224190613163565b156122a35760405162461bcd60e51b815260206004820152602c60248201527f43616e6e6f7420666c757368206c6561662061732069742077696c6c206e657660448201526b65722068617665206120505360a01b6064820152608401610743565b60008281526020849052604090206007015415612331576000828152602084905260408120600801546122d690426154d4565b60008481526020869052604080822060070154825281206009015491925090612300908390615498565b90508281101561230d5750815b600084815260208690526040808220600701548252902061232e90826135ae565b50505b60008281526020849052604090206118469042613639565b6000818152602083905260409020600101546123b35760405162461bcd60e51b8152602060048201526024808201527f466f726b2063616e6469646174652076657274657820646f6573206e6f7420656044820152631e1a5cdd60e21b6064820152608401610743565b60008181526020839052604090206123ca90613163565b156124225760405162461bcd60e51b815260206004820152602260248201527f4c6561662063616e206e65766572206265206120666f726b2063616e64696461604482015261746560f01b6064820152608401610743565b600081815260208390526040808220600a015482529020600101546124795760405162461bcd60e51b815260206004820152600d60248201526c4e6f20737563636573736f727360981b6044820152606401610743565b600081815260208390526040808220600a81015483529082206002908101549284905201546124a890826154d4565b60011461250e5760405162461bcd60e51b815260206004820152602e60248201527f4c6f7765737420686569676874206e6f74206f6e652061626f7665207468652060448201526d18dd5c9c995b9d081a195a59da1d60921b6064820152608401610743565b600082815260208490526040902060070154156118465760405162461bcd60e51b81526020600482015260196024820152782430b990383932b9bab6b83a34bb329039bab1b1b2b9b9b7b960391b6044820152606401610743565b600080826001600160a01b0316639ca565d48560400151602001516040518263ffffffff1660e01b81526004016125a291815260200190565b602060405180830381865afa1580156125bf573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906125e39190615379565b6040858101515190516317f7d0c560e11b815260048101839052919250906001600160a01b03851690632fefa18a90602401602060405180830381865afa158015612632573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906126569190615379565b146126ba5760405162461bcd60e51b815260206004820152602e60248201527f436c61696d207072656465636573736f72206e6f74206c696e6b656420746f2060448201526d74686973206368616c6c656e676560901b6064820152608401610743565b6040808501516020015190516344b77df960e11b81526000916001600160a01b0386169163896efbf2916126f49160040190815260200190565b602060405180830381865afa158015612711573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906127359190615379565b6040516344b77df960e11b8152600481018490529091506000906001600160a01b0386169063896efbf290602401602060405180830381865afa158015612780573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906127a49190615379565b905060006127b282846154d4565b905086604001516040015181146128015760405162461bcd60e51b8152602060048201526013602482015272125b9d985b1a59081b195859881a195a59da1d606a1b6044820152606401610743565b6128148888604001518960000151613688565b50505050604083810151805160608201519282015160209092015160009361284893909133612843828a6138a0565b613a11565b604080860151516000908152602088815291902054908601519192506128719188918491611b02565b9695505050505050565b604080820151602090810151600090815290859052908120600101546128b35760405162461bcd60e51b815260040161074390615562565b60408281015160209081015160009081529086905281812060040154808252919020600101546128f55760405162461bcd60e51b815260040161074390615590565b6000818152602086815260408083206002908101548783015190930151845292209091015461292491906154d4565b6001146129435760405162461bcd60e51b8152600401610743906155c5565b60408084015151600083815260208890529190912060030154146129795760405162461bcd60e51b81526004016107439061560b565b60008061298f8686604001518760000151613688565b50505060408083015180516060820151928201516020909201516000936129bd939091336128438b8361192b565b60408085015151600090815260208781529190205490850151919250610f249187918491611b02565b60408082015160209081015160009081529085905290812060010154612a1e5760405162461bcd60e51b815260040161074390615562565b6040828101516020908101516000908152908690528181206004015480825291902060010154612a605760405162461bcd60e51b815260040161074390615590565b60008181526020868152604080832060029081015487830151909301518452922090910154612a8f91906154d4565b600114612aae5760405162461bcd60e51b8152600401610743906155c5565b6040808401515160008381526020889052919091206003015414612ae45760405162461bcd60e51b81526004016107439061560b565b6000612afc846040015160c001518560800151613b77565b60008381526020889052604081206002015491925090612b20906210000090615651565b905061298f8686604001518760000151613688565b600080612b428685612349565b60008481526020869052604090206001015415612b715760405162461bcd60e51b815260040161074390615668565b612b7c868585613537565b15612bd35760405162461bcd60e51b815260206004820152602160248201527f50726573756d707469766520737563636573736f7220636f6e6669726d61626c6044820152606560f81b6064820152608401610743565b60008481526020879052604090206003015415612c025760405162461bcd60e51b815260040161074390615699565b6000848152602087815260408083205480845291889052822060020154909190612c3f9060ff166003811115612c3a57612c3a614e6f565b613b82565b90506000612c4d878361172b565b600081815260208a9052604090205490915015612c7c5760405162461bcd60e51b815260040161074390615699565b9890975095505050505050565b612c91614bbb565b6000849003612cb25760405162461bcd60e51b8152600401610743906154e7565b6000839003612cd35760405162461bcd60e51b815260040161074390615512565b50604080516101808101825293845260208401929092526000918301829052606083018290526080830182905260a083015260c08201819052600160e083015261010082018190526101208201819052610140820181905261016082015290565b60006105e08260000151836020015184604001516120e2565b6001820154612d6e5760405162461bcd60e51b81526004016107439061519b565b612d7782613163565b15612d945760405162461bcd60e51b8152600401610743906156cb565b600390910155565b612da68382612f52565b60008181526020849052604080822060040154825281206003015490819003612de15760405162461bcd60e51b815260040161074390615392565b60008181526020849052604090206001015482146120dc5760405162461bcd60e51b815260206004820152603b60248201527f53756363657373696f6e206368616c6c656e676520646964206e6f742064656360448201527f6c617265207468697320766572746578207468652077696e6e657200000000006064820152608401610743565b6000336001600160a01b03831614612ed45760405162461bcd60e51b815260206004820152602a60248201527f4f6e6c7920617373657274696f6e20636861696e2063616e20637265617465206044820152696368616c6c656e67657360b01b6064820152608401610743565b6000612ee184600061172b565b600081815260208790526040902054909150156107d75760405162461bcd60e51b815260040161074390615699565b600080612f46838587604051602001612f2b91815260200190565b60405160208183030381529060405280519060200120613c51565b90951495945050505050565b600081815260208390526040902060010154612f805760405162461bcd60e51b81526004016107439061519b565b60008082815260208490526040902060060154600160a01b900460ff166001811115612fae57612fae614e6f565b14612ff35760405162461bcd60e51b8152602060048201526015602482015274566572746578206973206e6f742070656e64696e6760581b6044820152606401610743565b6000818152602083905260408082206004015480835291206001015461302b5760405162461bcd60e51b815260040161074390615338565b6001600082815260208590526040902060060154600160a01b900460ff16600181111561305a5761305a614e6f565b146118465760405162461bcd60e51b8152602060048201526019602482015278141c99591958d95cdcdbdc881b9bdd0818dbdb999a5c9b5959603a1b6044820152606401610743565b60018101546130c45760405162461bcd60e51b81526004016107439061519b565b60006006820154600160a01b900460ff1660018111156130e6576130e6614e6f565b1461314d5760405162461bcd60e51b815260206004820152603160248201527f566572746578206d7573742062652050656e64696e67206265666f72652062656044820152701a5b99c81cd95d0810dbdb999a5c9b5959607a1b6064820152608401610743565b600601805460ff60a01b1916600160a01b179055565b60006131728260010154151590565b6131ca5760405162461bcd60e51b8152602060048201526024808201527f506f74656e7469616c206c6561662076657274657820646f6573206e6f7420656044820152631e1a5cdd60e21b6064820152608401610743565b6001820154151580156105e0575050600601546001600160a01b0316151590565b600083815260208690526040812060010154819061321b5760405162461bcd60e51b81526004016107439061519b565b600085815260208881526040808320548084529189905290912060010154156132565760405162461bcd60e51b815260040161074390615668565b6000868152602089905260408082206004015480835291206001015461328e5760405162461bcd60e51b815260040161074390615338565b600081815260208a905260409020600701548790036132fb5760405162461bcd60e51b815260206004820152602360248201527f43616e6e6f74206269736563742070726573756d70746976652073756363657360448201526239b7b960e91b6064820152608401610743565b60006133078a89613cf3565b905060008087806020019051810190613320919061576a565b909250905061337389613334856001615498565b8e60008e8152602001908152602001600020600101548f60008f815260200190815260200160002060020154600161336c9190615498565b8686613d83565b61337e858a856120e2565b9c929b50919950505050505050505050565b60018201546133b15760405162461bcd60e51b81526004016107439061519b565b808260040154036133fe5760405162461bcd60e51b8152602060048201526017602482015276141c99591958d95cdcdbdc88185b1c9958591e481cd95d604a1b6044820152606401610743565b61340782614054565b156134545760405162461bcd60e51b815260206004820152601e60248201527f43616e6e6f7420736574207072656465636573736f72206f6e20726f6f7400006044820152606401610743565b600490910155565b600182015461347d5760405162461bcd60e51b81526004016107439061519b565b80158061348e575080826007015414155b6134cb5760405162461bcd60e51b815260206004820152600e60248201526d141cc8185b1c9958591e481cd95d60921b6044820152606401610743565b6134d482613163565b156135215760405162461bcd60e51b815260206004820152601a60248201527f43616e6e6f7420736574207073206964206f6e2061206c6561660000000000006044820152606401610743565b6007820181905580156118a257600a9190910155565b6000828152602084905260408120600101546135655760405162461bcd60e51b815260040161074390615338565b6000838152602085905260408120600701549003613585575060006106c6565b816135a5858660008781526020019081526020016000206007015461192b565b11949350505050565b60018201546135cf5760405162461bcd60e51b81526004016107439061519b565b6135d882614054565b156136315760405162461bcd60e51b8152602060048201526024808201527f43616e6e6f742073657420707320666c75736865642074696d65206f6e2061206044820152631c9bdbdd60e21b6064820152608401610743565b600990910155565b600182015461365a5760405162461bcd60e51b81526004016107439061519b565b61366382613163565b156136805760405162461bcd60e51b8152600401610743906156cb565b600890910155565b60208201516000036136cc5760405162461bcd60e51b815260206004820152600d60248201526c115b5c1d1e4818db185a5b5259609a1b6044820152606401610743565b60608201516000036137145760405162461bcd60e51b8152602060048201526011602482015270115b5c1d1e481a1a5cdd1bdc9e549bdbdd607a1b6044820152606401610743565b81604001516000036137575760405162461bcd60e51b815260206004820152600c60248201526b115b5c1d1e481a195a59da1d60a21b6044820152606401610743565b8034146137a65760405162461bcd60e51b815260206004820152601b60248201527f496e636f7272656374206d696e692d7374616b6520616d6f756e7400000000006044820152606401610743565b8151600090815260208490526040902060010154156137d75760405162461bcd60e51b815260040161074390615668565b6137f382606001518360c0015184604001518560e00151612f10565b61383b5760405162461bcd60e51b81526020600482015260196024820152784c617374207374617465206e6f7420696e20686973746f727960381b6044820152606401610743565b6138548260600151836080015160008560a00151612f10565b6118465760405162461bcd60e51b815260206004820152601a60248201527f4669727374207374617465206e6f7420696e20686973746f72790000000000006044820152606401610743565b6040516306106c4560e31b81526004810183905260009081906001600160a01b03841690633083622890602401602060405180830381865afa1580156138ea573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061390e91906157cd565b90508015613a0757604051632729597560e21b8152600481018590526000906001600160a01b03851690639ca565d490602401602060405180830381865afa15801561395e573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906139829190615379565b6040516343ed6ad960e01b8152600481018290529091506000906001600160a01b038616906343ed6ad990602401602060405180830381865afa1580156139cd573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906139f19190615379565b90506139fd81426154d4565b93505050506105e0565b60009150506105e0565b613a19614bbb565b6000879003613a3a5760405162461bcd60e51b8152600401610743906154e7565b6000869003613a5b5760405162461bcd60e51b815260040161074390615512565b84600003613a7b5760405162461bcd60e51b81526004016107439061553d565b6000849003613abc5760405162461bcd60e51b815260206004820152600d60248201526c16995c9bc818db185a5b481a59609a1b6044820152606401610743565b6001600160a01b038316613b085760405162461bcd60e51b81526020600482015260136024820152725a65726f207374616b6572206164647265737360681b6044820152606401610743565b5060408051610180810182529687526020870195909552938501929092526000606085018190526080850181905260a08501919091526001600160a01b0390911660c084015260e083018190526101008301819052610120830181905261014083019190915261016082015290565b60006105dd826157ef565b600080826003811115613b9757613b97614e6f565b03613ba457506001919050565b6001826003811115613bb857613bb8614e6f565b03613bc557506002919050565b6002826003811115613bd957613bd9614e6f565b03613be657506003919050565b60405162461bcd60e51b815260206004820152603560248201527f43616e6e6f7420676574206e657874206368616c6c656e6765207479706520666044820152746f72206f6e652073746570206368616c6c656e676560581b6064820152608401610743565b919050565b8251600090610100811115613c8457604051637ed6198f60e11b8152600481018290526101006024820152604401610743565b8260005b82811015613ce9576000878281518110613ca457613ca4615816565b60200260200101519050816001901b8716600003613cd057826000528060205260406000209250613ce0565b8060005282602052604060002092505b50600101613c88565b5095945050505050565b600081815260208390526040812060010154613d215760405162461bcd60e51b81526004016107439061519b565b60008281526020849052604080822060040154808352912060010154613d595760405162461bcd60e51b815260040161074390615338565b60008181526020859052604080822060029081015486845291909220909101546107d791906140db565b60008511613dca5760405162461bcd60e51b815260206004820152601460248201527305072652d73697a652063616e6e6f7420626520360641b6044820152606401610743565b85613dd483614190565b14613e215760405162461bcd60e51b815260206004820152601b60248201527f50726520657870616e73696f6e20726f6f74206d69736d6174636800000000006044820152606401610743565b84613e2b836142f9565b14613e825760405162461bcd60e51b815260206004820152602160248201527f5072652073697a6520646f6573206e6f74206d6174636820657870616e73696f6044820152603760f91b6064820152608401610743565b828510613ed15760405162461bcd60e51b815260206004820181905260248201527f5072652073697a65206e6f74206c657373207468616e20706f73742073697a656044820152606401610743565b6000859050600080613ee68560008751614354565b90505b85831015613f9e576000613efd8488614486565b905084518310613f445760405162461bcd60e51b8152602060048201526012602482015271496e646578206f7574206f662072616e676560701b6044820152606401610743565b613f688282878681518110613f5b57613f5b615816565b602002602001015161454b565b91506001811b613f788186615498565b945087851115613f8a57613f8a61582c565b83613f9481615842565b9450505050613ee9565b86613fa882614190565b146140005760405162461bcd60e51b815260206004820152602260248201527f506f737420657870616e73696f6e20726f6f74206e6f7420657175616c20706f6044820152611cdd60f21b6064820152608401610743565b835182146140495760405162461bcd60e51b8152602060048201526016602482015275496e636f6d706c6574652070726f6f6620757361676560501b6044820152606401610743565b505050505050505050565b60006140638260010154151590565b6140bb5760405162461bcd60e51b8152602060048201526024808201527f506f74656e7469616c20726f6f742076657274657820646f6573206e6f7420656044820152631e1a5cdd60e21b6064820152608401610743565b60068201546001600160a01b03161580156105e057505060050154151590565b600060026140e984846154d4565b10156141375760405162461bcd60e51b815260206004820181905260248201527f48656967687420646966666572656e74206e6f742074776f206f72206d6f72656044820152606401610743565b61414183836154d4565b60020361415a57614153836001615498565b90506105e0565b60006141718461416b6001866154d4565b18614a62565b9050600019811b60018161418582876154d4565b16610f2491906154d4565b6000808251116141db5760405162461bcd60e51b815260206004820152601660248201527522b6b83a3c9036b2b935b6329032bc3830b739b4b7b760511b6044820152606401610743565b6040825111156141fd5760405162461bcd60e51b81526004016107439061585b565b6000805b83518110156142f257600084828151811061421e5761421e615816565b60200260200101519050826000801b0361428a578015614285578092506001855161424991906154d4565b82146142855760405161426c908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b6142df565b80156142a957604080516020810183905290810184905260600161426c565b6040516142c6908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b50806142ea81615842565b915050614201565b5092915050565b600080805b83518110156142f25783818151811061431957614319615816565b60200260200101516000801b1461434257614335816002615976565b61433f9083615498565b91505b8061434c81615842565b9150506142fe565b60608183106143755760405162461bcd60e51b815260040161074390615982565b83518211156143d05760405162461bcd60e51b815260206004820152602160248201527f456e64206e6f74206c657373206f7220657175616c207468616e206c656e67746044820152600d60fb1b6064820152608401610743565b60006143dc84846154d4565b6001600160401b038111156143f3576143f3614d59565b60405190808252806020026020018201604052801561441c578160200160208202803683370190505b509050835b8381101561447d5785818151811061443b5761443b615816565b602002602001015182868361445091906154d4565b8151811061446057614460615816565b60209081029190910101528061447581615842565b915050614421565b50949350505050565b60008183106144a75760405162461bcd60e51b815260040161074390615982565b60006144b4838518614b41565b9050600060016144c48382615498565b6001901b6144d291906154d4565b905084811684821681156144f4576144e982614b7e565b9450505050506105e0565b8015614503576144e981614b41565b60405162461bcd60e51b815260206004820152601b60248201527f426f7468207920616e64207a2063616e6e6f74206265207a65726f00000000006044820152606401610743565b60606040831061458e5760405162461bcd60e51b815260206004820152600e60248201526d098caeccad840e8dede40d0d2ced60931b6044820152606401610743565b60008290036145df5760405162461bcd60e51b815260206004820152601b60248201527f43616e6e6f7420617070656e6420656d707479207375627472656500000000006044820152606401610743565b6040845111156146015760405162461bcd60e51b81526004016107439061585b565b835160000361467f576000614617846001615498565b6001600160401b0381111561462e5761462e614d59565b604051908082528060200260200182016040528015614657578160200160208202803683370190505b5090508281858151811061466d5761466d615816565b602090810291909101015290506106c6565b835183106146ed5760405162461bcd60e51b815260206004820152603560248201527f4c6576656c2067726561746572207468616e2068696768657374206c6576656c6044820152741037b31031bab93932b73a1032bc3830b739b4b7b760591b6064820152608401610743565b8160006146f9866142f9565b90506000614708866002615976565b6147129083615498565b9050600061471f83614b41565b61472883614b41565b116147755787516001600160401b0381111561474657614746614d59565b60405190808252806020026020018201604052801561476f578160200160208202803683370190505b506147c4565b8751614782906001615498565b6001600160401b0381111561479957614799614d59565b6040519080825280602002602001820160405280156147c2578160200160208202803683370190505b505b90506040815111156148185760405162461bcd60e51b815260206004820152601c60248201527f417070656e642063726561746573206f76657273697a652074726565000000006044820152606401610743565b60005b88518110156149b957878110156148a75788818151811061483e5761483e615816565b60200260200101516000801b146148a25760405162461bcd60e51b815260206004820152602260248201527f417070656e642061626f7665206c65617374207369676e69666963616e7420626044820152611a5d60f21b6064820152608401610743565b6149a7565b60008590036148ed578881815181106148c2576148c2615816565b60200260200101518282815181106148dc576148dc615816565b6020026020010181815250506149a7565b8881815181106148ff576148ff615816565b60200260200101516000801b03614937578482828151811061492357614923615816565b6020908102919091010152600094506149a7565b6000801b82828151811061494d5761494d615816565b60200260200101818152505088818151811061496b5761496b615816565b60200260200101518560405160200161498e929190918252602082015260400190565b6040516020818303038152906040528051906020012094505b806149b181615842565b91505061481b565b5083156149ed578381600183516149d091906154d4565b815181106149e0576149e0615816565b6020026020010181815250505b80600182516149fc91906154d4565b81518110614a0c57614a0c615816565b60200260200101516000801b03614a575760405162461bcd60e51b815260206004820152600f60248201526e4c61737420656e747279207a65726f60881b6044820152606401610743565b979650505050505050565b6000600160801b8210614a8257608091821c91614a7f9082615498565b90505b600160401b8210614aa057604091821c91614a9d9082615498565b90505b6401000000008210614abf57602091821c91614abc9082615498565b90505b620100008210614adc57601091821c91614ad99082615498565b90505b6101008210614af857600891821c91614af59082615498565b90505b60108210614b1357600491821c91614b109082615498565b90505b60048210614b2e57600291821c91614b2b9082615498565b90505b60028210613c4c576105e0600182615498565b600081600003614b635760405162461bcd60e51b8152600401610743906159b3565b600160801b8210614a8257608091821c91614a7f9082615498565b6000808211614b9f5760405162461bcd60e51b8152600401610743906159b3565b60008280614bae6001826154d4565b161890506106c681614b41565b6040805161018081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c081018290529060e08201908152600060208201819052604082018190526060820181905260809091015290565b60008083601f840112614c2e57600080fd5b5081356001600160401b03811115614c4557600080fd5b6020830191508360208260051b8501011115614c6057600080fd5b9250929050565b60008060008060008060808789031215614c8057600080fd5b8635955060208701356001600160401b0380821115614c9e57600080fd5b9088019060a0828b031215614cb257600080fd5b90955060408801359080821115614cc857600080fd5b614cd48a838b01614c1c565b90965094506060890135915080821115614ced57600080fd5b50614cfa89828a01614c1c565b979a9699509497509295939492505050565b60008060408385031215614d1f57600080fd5b82359150602083013560048110614d3557600080fd5b809150509250929050565b600060208284031215614d5257600080fd5b5035919050565b634e487b7160e01b600052604160045260246000fd5b60405161010081016001600160401b0381118282101715614d9257614d92614d59565b60405290565b604051601f8201601f191681016001600160401b0381118282101715614dc057614dc0614d59565b604052919050565b600080600060608486031215614ddd57600080fd5b83359250602080850135925060408501356001600160401b0380821115614e0357600080fd5b818701915087601f830112614e1757600080fd5b813581811115614e2957614e29614d59565b614e3b601f8201601f19168501614d98565b91508082528884828501011115614e5157600080fd5b80848401858401376000848284010152508093505050509250925092565b634e487b7160e01b600052602160045260246000fd5b60048110614e9557614e95614e6f565b9052565b600060808201905082518252602083015160208301526040830151614ec16040840182614e85565b506060928301516001600160a01b0316919092015290565b600080600060608486031215614eee57600080fd5b505081359360208301359350604090920135919050565b60028110614e9557614e95614e6f565b600061018082019050825182526020830151602083015260408301516040830152606083015160608301526080830151608083015260a083015160a083015260c0830151614f6e60c08401826001600160a01b03169052565b5060e0830151614f8160e0840182614f05565b5061010083810151908301526101208084015190830152610140808401519083015261016092830151929091019190915290565b6001600160a01b03811681146105fd57600080fd5b60008060008060808587031215614fe057600080fd5b8435614feb81614fb5565b93506020850135925060408501359150606085013561500981614fb5565b939692955090935050565b60008083601f84011261502657600080fd5b5081356001600160401b0381111561503d57600080fd5b602083019150836020828501011115614c6057600080fd5b60008060008060006060868803121561506d57600080fd5b85356001600160401b038082111561508457600080fd5b90870190610100828a03121561509957600080fd5b909550602087013590808211156150af57600080fd5b6150bb89838a01615014565b909650945060408801359150808211156150d457600080fd5b506150e188828901615014565b969995985093965092949392505050565b848152602081018490526080810161510d6040830185614e85565b6001600160a01b0392909216606091909101529392505050565b8c8152602081018c9052604081018b9052606081018a90526080810189905260a081018890526001600160a01b03871660c0820152610180810161516e60e0830188614f05565b856101008301528461012083015283610140830152826101608301529d9c50505050505050505050505050565b60208082526015908201527415995c9d195e08191bd95cc81b9bdd08195e1a5cdd605a1b604082015260600190565b6020808252601a908201527f5072656465636573736f7220646f6573206e6f74206578697374000000000000604082015260600190565b60006001600160401b0382111561521a5761521a614d59565b5060051b60200190565b600082601f83011261523557600080fd5b8135602061524a61524583615201565b614d98565b82815260059290921b8401810191818101908684111561526957600080fd5b8286015b84811015615284578035835291830191830161526d565b509695505050505050565b600061010082360312156152a257600080fd5b6152aa614d6f565b823581526020830135602082015260408301356040820152606083013560608201526080830135608082015260a08301356001600160401b03808211156152f057600080fd5b6152fc36838701615224565b60a084015260c085013560c084015260e085013591508082111561531f57600080fd5b5061532c36828601615224565b60e08301525092915050565b60208082526021908201527f5072656465636573736f722076657274657820646f6573206e6f7420657869736040820152601d60fa1b606082015260800190565b60006020828403121561538b57600080fd5b5051919050565b60208082526023908201527f53756363657373696f6e206368616c6c656e676520646f6573206e6f742065786040820152621a5cdd60ea1b606082015260800190565b6000808335601e198436030181126153ec57600080fd5b8301803591506001600160401b0382111561540657600080fd5b602001915036819003821315614c6057600080fd5b853581526000602087013561542f81614fb5565b6001600160a01b03166020830152604082018690526060820185905260a0608083018190528201839052828460c0840137600060c0848401015260c0601f19601f85011683010190509695505050505050565b634e487b7160e01b600052601160045260246000fd5b808201808211156105e0576105e0615482565b8281526000600483106154c0576154c0614e6f565b5060f89190911b6020820152602101919050565b818103818111156105e0576105e0615482565b60208082526011908201527016995c9bc818da185b1b195b99d9481a59607a1b604082015260600190565b60208082526011908201527016995c9bc81a1a5cdd1bdc9e481c9bdbdd607a1b604082015260600190565b6020808252600b908201526a16995c9bc81a195a59da1d60aa1b604082015260600190565b60208082526014908201527310db185a5b48191bd95cc81b9bdd08195e1a5cdd60621b604082015260600190565b6020808252818101527f436c61696d207072656465636573736f7220646f6573206e6f74206578697374604082015260600190565b60208082526026908201527f436c61696d206e6f7420686569676874206f6e652061626f766520707265646560408201526531b2b9b9b7b960d11b606082015260800190565b60208082526026908201527f436c61696d2068617320696e76616c69642073756363657373696f6e206368616040820152656c6c656e676560d01b606082015260800190565b80820281158282048414176105e0576105e0615482565b60208082526017908201527615da5b9b995c88185b1c9958591e48191958db185c9959604a1b604082015260600190565b6020808252601890820152774368616c6c656e676520616c72656164792065786973747360401b604082015260600190565b60208082526024908201527f43616e6e6f7420736574207073206c6173742075706461746564206f6e2061206040820152633632b0b360e11b606082015260800190565b600082601f83011261572057600080fd5b8151602061573061524583615201565b82815260059290921b8401810191818101908684111561574f57600080fd5b8286015b848110156152845780518352918301918301615753565b6000806040838503121561577d57600080fd5b82516001600160401b038082111561579457600080fd5b6157a08683870161570f565b935060208501519150808211156157b657600080fd5b506157c38582860161570f565b9150509250929050565b6000602082840312156157df57600080fd5b815180151581146106c657600080fd5b80516020808301519190811015615810576000198160200360031b1b821691505b50919050565b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052600160045260246000fd5b60006001820161585457615854615482565b5060010190565b6020808252601a908201527f4d65726b6c6520657870616e73696f6e20746f6f206c61726765000000000000604082015260600190565b600181815b808511156158cd5781600019048211156158b3576158b3615482565b808516156158c057918102915b93841c9390800290615897565b509250929050565b6000826158e4575060016105e0565b816158f1575060006105e0565b816001811461590757600281146159115761592d565b60019150506105e0565b60ff84111561592257615922615482565b50506001821b6105e0565b5060208310610133831016604e8410600b8410161715615950575081810a6105e0565b61595a8383615892565b806000190482111561596e5761596e615482565b029392505050565b60006105dd83836158d5565b60208082526017908201527614dd185c9d081b9bdd081b195cdcc81d1a185b88195b99604a1b604082015260600190565b6020808252601c908201527f5a65726f20686173206e6f207369676e69666963616e7420626974730000000060408201526060019056fea2646970667358221220b96c4279b3c7f4227e8a2650d0f4885de248da8dd9fb6074df686837145e135064736f6c63430008110033",
}

// ChallengeManagerImplABI is the input ABI used to generate the binding from.
// Deprecated: Use ChallengeManagerImplMetaData.ABI instead.
var ChallengeManagerImplABI = ChallengeManagerImplMetaData.ABI

// ChallengeManagerImplBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ChallengeManagerImplMetaData.Bin instead.
var ChallengeManagerImplBin = ChallengeManagerImplMetaData.Bin

// DeployChallengeManagerImpl deploys a new Ethereum contract, binding an instance of ChallengeManagerImpl to it.
func DeployChallengeManagerImpl(auth *bind.TransactOpts, backend bind.ContractBackend, _assertionChain common.Address, _miniStakeValue *big.Int, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (common.Address, *types.Transaction, *ChallengeManagerImpl, error) {
	parsed, err := ChallengeManagerImplMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ChallengeManagerImplBin), backend, _assertionChain, _miniStakeValue, _challengePeriodSec, _oneStepProofEntry)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ChallengeManagerImpl{ChallengeManagerImplCaller: ChallengeManagerImplCaller{contract: contract}, ChallengeManagerImplTransactor: ChallengeManagerImplTransactor{contract: contract}, ChallengeManagerImplFilterer: ChallengeManagerImplFilterer{contract: contract}}, nil
}

// ChallengeManagerImpl is an auto generated Go binding around an Ethereum contract.
type ChallengeManagerImpl struct {
	ChallengeManagerImplCaller     // Read-only binding to the contract
	ChallengeManagerImplTransactor // Write-only binding to the contract
	ChallengeManagerImplFilterer   // Log filterer for contract events
}

// ChallengeManagerImplCaller is an auto generated read-only Go binding around an Ethereum contract.
type ChallengeManagerImplCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChallengeManagerImplTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ChallengeManagerImplTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChallengeManagerImplFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ChallengeManagerImplFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChallengeManagerImplSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ChallengeManagerImplSession struct {
	Contract     *ChallengeManagerImpl // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// ChallengeManagerImplCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ChallengeManagerImplCallerSession struct {
	Contract *ChallengeManagerImplCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// ChallengeManagerImplTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ChallengeManagerImplTransactorSession struct {
	Contract     *ChallengeManagerImplTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// ChallengeManagerImplRaw is an auto generated low-level Go binding around an Ethereum contract.
type ChallengeManagerImplRaw struct {
	Contract *ChallengeManagerImpl // Generic contract binding to access the raw methods on
}

// ChallengeManagerImplCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ChallengeManagerImplCallerRaw struct {
	Contract *ChallengeManagerImplCaller // Generic read-only contract binding to access the raw methods on
}

// ChallengeManagerImplTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ChallengeManagerImplTransactorRaw struct {
	Contract *ChallengeManagerImplTransactor // Generic write-only contract binding to access the raw methods on
}

// NewChallengeManagerImpl creates a new instance of ChallengeManagerImpl, bound to a specific deployed contract.
func NewChallengeManagerImpl(address common.Address, backend bind.ContractBackend) (*ChallengeManagerImpl, error) {
	contract, err := bindChallengeManagerImpl(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerImpl{ChallengeManagerImplCaller: ChallengeManagerImplCaller{contract: contract}, ChallengeManagerImplTransactor: ChallengeManagerImplTransactor{contract: contract}, ChallengeManagerImplFilterer: ChallengeManagerImplFilterer{contract: contract}}, nil
}

// NewChallengeManagerImplCaller creates a new read-only instance of ChallengeManagerImpl, bound to a specific deployed contract.
func NewChallengeManagerImplCaller(address common.Address, caller bind.ContractCaller) (*ChallengeManagerImplCaller, error) {
	contract, err := bindChallengeManagerImpl(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerImplCaller{contract: contract}, nil
}

// NewChallengeManagerImplTransactor creates a new write-only instance of ChallengeManagerImpl, bound to a specific deployed contract.
func NewChallengeManagerImplTransactor(address common.Address, transactor bind.ContractTransactor) (*ChallengeManagerImplTransactor, error) {
	contract, err := bindChallengeManagerImpl(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerImplTransactor{contract: contract}, nil
}

// NewChallengeManagerImplFilterer creates a new log filterer instance of ChallengeManagerImpl, bound to a specific deployed contract.
func NewChallengeManagerImplFilterer(address common.Address, filterer bind.ContractFilterer) (*ChallengeManagerImplFilterer, error) {
	contract, err := bindChallengeManagerImpl(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerImplFilterer{contract: contract}, nil
}

// bindChallengeManagerImpl binds a generic wrapper to an already deployed contract.
func bindChallengeManagerImpl(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ChallengeManagerImplABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ChallengeManagerImpl *ChallengeManagerImplRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ChallengeManagerImpl.Contract.ChallengeManagerImplCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ChallengeManagerImpl *ChallengeManagerImplRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.ChallengeManagerImplTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ChallengeManagerImpl *ChallengeManagerImplRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.ChallengeManagerImplTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ChallengeManagerImpl *ChallengeManagerImplCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ChallengeManagerImpl.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.contract.Transact(opts, method, params...)
}

// AssertionChain is a free data retrieval call binding the contract method 0x48dd2924.
//
// Solidity: function assertionChain() view returns(address)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) AssertionChain(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "assertionChain")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AssertionChain is a free data retrieval call binding the contract method 0x48dd2924.
//
// Solidity: function assertionChain() view returns(address)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) AssertionChain() (common.Address, error) {
	return _ChallengeManagerImpl.Contract.AssertionChain(&_ChallengeManagerImpl.CallOpts)
}

// AssertionChain is a free data retrieval call binding the contract method 0x48dd2924.
//
// Solidity: function assertionChain() view returns(address)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) AssertionChain() (common.Address, error) {
	return _ChallengeManagerImpl.Contract.AssertionChain(&_ChallengeManagerImpl.CallOpts)
}

// CalculateChallengeId is a free data retrieval call binding the contract method 0x16ef5534.
//
// Solidity: function calculateChallengeId(bytes32 assertionId, uint8 typ) pure returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) CalculateChallengeId(opts *bind.CallOpts, assertionId [32]byte, typ uint8) ([32]byte, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "calculateChallengeId", assertionId, typ)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateChallengeId is a free data retrieval call binding the contract method 0x16ef5534.
//
// Solidity: function calculateChallengeId(bytes32 assertionId, uint8 typ) pure returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) CalculateChallengeId(assertionId [32]byte, typ uint8) ([32]byte, error) {
	return _ChallengeManagerImpl.Contract.CalculateChallengeId(&_ChallengeManagerImpl.CallOpts, assertionId, typ)
}

// CalculateChallengeId is a free data retrieval call binding the contract method 0x16ef5534.
//
// Solidity: function calculateChallengeId(bytes32 assertionId, uint8 typ) pure returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) CalculateChallengeId(assertionId [32]byte, typ uint8) ([32]byte, error) {
	return _ChallengeManagerImpl.Contract.CalculateChallengeId(&_ChallengeManagerImpl.CallOpts, assertionId, typ)
}

// CalculateChallengeVertexId is a free data retrieval call binding the contract method 0x4a658788.
//
// Solidity: function calculateChallengeVertexId(bytes32 challengeId, bytes32 commitmentMerkle, uint256 commitmentHeight) pure returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) CalculateChallengeVertexId(opts *bind.CallOpts, challengeId [32]byte, commitmentMerkle [32]byte, commitmentHeight *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "calculateChallengeVertexId", challengeId, commitmentMerkle, commitmentHeight)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateChallengeVertexId is a free data retrieval call binding the contract method 0x4a658788.
//
// Solidity: function calculateChallengeVertexId(bytes32 challengeId, bytes32 commitmentMerkle, uint256 commitmentHeight) pure returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) CalculateChallengeVertexId(challengeId [32]byte, commitmentMerkle [32]byte, commitmentHeight *big.Int) ([32]byte, error) {
	return _ChallengeManagerImpl.Contract.CalculateChallengeVertexId(&_ChallengeManagerImpl.CallOpts, challengeId, commitmentMerkle, commitmentHeight)
}

// CalculateChallengeVertexId is a free data retrieval call binding the contract method 0x4a658788.
//
// Solidity: function calculateChallengeVertexId(bytes32 challengeId, bytes32 commitmentMerkle, uint256 commitmentHeight) pure returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) CalculateChallengeVertexId(challengeId [32]byte, commitmentMerkle [32]byte, commitmentHeight *big.Int) ([32]byte, error) {
	return _ChallengeManagerImpl.Contract.CalculateChallengeVertexId(&_ChallengeManagerImpl.CallOpts, challengeId, commitmentMerkle, commitmentHeight)
}

// ChallengeExists is a free data retrieval call binding the contract method 0x1b7bbecb.
//
// Solidity: function challengeExists(bytes32 challengeId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) ChallengeExists(opts *bind.CallOpts, challengeId [32]byte) (bool, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "challengeExists", challengeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ChallengeExists is a free data retrieval call binding the contract method 0x1b7bbecb.
//
// Solidity: function challengeExists(bytes32 challengeId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) ChallengeExists(challengeId [32]byte) (bool, error) {
	return _ChallengeManagerImpl.Contract.ChallengeExists(&_ChallengeManagerImpl.CallOpts, challengeId)
}

// ChallengeExists is a free data retrieval call binding the contract method 0x1b7bbecb.
//
// Solidity: function challengeExists(bytes32 challengeId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) ChallengeExists(challengeId [32]byte) (bool, error) {
	return _ChallengeManagerImpl.Contract.ChallengeExists(&_ChallengeManagerImpl.CallOpts, challengeId)
}

// ChallengePeriodSec is a free data retrieval call binding the contract method 0x654f0dc2.
//
// Solidity: function challengePeriodSec() view returns(uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) ChallengePeriodSec(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "challengePeriodSec")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChallengePeriodSec is a free data retrieval call binding the contract method 0x654f0dc2.
//
// Solidity: function challengePeriodSec() view returns(uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) ChallengePeriodSec() (*big.Int, error) {
	return _ChallengeManagerImpl.Contract.ChallengePeriodSec(&_ChallengeManagerImpl.CallOpts)
}

// ChallengePeriodSec is a free data retrieval call binding the contract method 0x654f0dc2.
//
// Solidity: function challengePeriodSec() view returns(uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) ChallengePeriodSec() (*big.Int, error) {
	return _ChallengeManagerImpl.Contract.ChallengePeriodSec(&_ChallengeManagerImpl.CallOpts)
}

// Challenges is a free data retrieval call binding the contract method 0xc1e69b66.
//
// Solidity: function challenges(bytes32 ) view returns(bytes32 rootId, bytes32 winningClaim, uint8 challengeType, address challenger)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) Challenges(opts *bind.CallOpts, arg0 [32]byte) (struct {
	RootId        [32]byte
	WinningClaim  [32]byte
	ChallengeType uint8
	Challenger    common.Address
}, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "challenges", arg0)

	outstruct := new(struct {
		RootId        [32]byte
		WinningClaim  [32]byte
		ChallengeType uint8
		Challenger    common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.RootId = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.WinningClaim = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.ChallengeType = *abi.ConvertType(out[2], new(uint8)).(*uint8)
	outstruct.Challenger = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// Challenges is a free data retrieval call binding the contract method 0xc1e69b66.
//
// Solidity: function challenges(bytes32 ) view returns(bytes32 rootId, bytes32 winningClaim, uint8 challengeType, address challenger)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) Challenges(arg0 [32]byte) (struct {
	RootId        [32]byte
	WinningClaim  [32]byte
	ChallengeType uint8
	Challenger    common.Address
}, error) {
	return _ChallengeManagerImpl.Contract.Challenges(&_ChallengeManagerImpl.CallOpts, arg0)
}

// Challenges is a free data retrieval call binding the contract method 0xc1e69b66.
//
// Solidity: function challenges(bytes32 ) view returns(bytes32 rootId, bytes32 winningClaim, uint8 challengeType, address challenger)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) Challenges(arg0 [32]byte) (struct {
	RootId        [32]byte
	WinningClaim  [32]byte
	ChallengeType uint8
	Challenger    common.Address
}, error) {
	return _ChallengeManagerImpl.Contract.Challenges(&_ChallengeManagerImpl.CallOpts, arg0)
}

// ChildrenAreAtOneStepFork is a free data retrieval call binding the contract method 0x7a4d47dc.
//
// Solidity: function childrenAreAtOneStepFork(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) ChildrenAreAtOneStepFork(opts *bind.CallOpts, vId [32]byte) (bool, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "childrenAreAtOneStepFork", vId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ChildrenAreAtOneStepFork is a free data retrieval call binding the contract method 0x7a4d47dc.
//
// Solidity: function childrenAreAtOneStepFork(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) ChildrenAreAtOneStepFork(vId [32]byte) (bool, error) {
	return _ChallengeManagerImpl.Contract.ChildrenAreAtOneStepFork(&_ChallengeManagerImpl.CallOpts, vId)
}

// ChildrenAreAtOneStepFork is a free data retrieval call binding the contract method 0x7a4d47dc.
//
// Solidity: function childrenAreAtOneStepFork(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) ChildrenAreAtOneStepFork(vId [32]byte) (bool, error) {
	return _ChallengeManagerImpl.Contract.ChildrenAreAtOneStepFork(&_ChallengeManagerImpl.CallOpts, vId)
}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,bytes32,uint8,address))
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) GetChallenge(opts *bind.CallOpts, challengeId [32]byte) (Challenge, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "getChallenge", challengeId)

	if err != nil {
		return *new(Challenge), err
	}

	out0 := *abi.ConvertType(out[0], new(Challenge)).(*Challenge)

	return out0, err

}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,bytes32,uint8,address))
func (_ChallengeManagerImpl *ChallengeManagerImplSession) GetChallenge(challengeId [32]byte) (Challenge, error) {
	return _ChallengeManagerImpl.Contract.GetChallenge(&_ChallengeManagerImpl.CallOpts, challengeId)
}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,bytes32,uint8,address))
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) GetChallenge(challengeId [32]byte) (Challenge, error) {
	return _ChallengeManagerImpl.Contract.GetChallenge(&_ChallengeManagerImpl.CallOpts, challengeId)
}

// GetCurrentPsTimer is a free data retrieval call binding the contract method 0x8ac04349.
//
// Solidity: function getCurrentPsTimer(bytes32 vId) view returns(uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) GetCurrentPsTimer(opts *bind.CallOpts, vId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "getCurrentPsTimer", vId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentPsTimer is a free data retrieval call binding the contract method 0x8ac04349.
//
// Solidity: function getCurrentPsTimer(bytes32 vId) view returns(uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) GetCurrentPsTimer(vId [32]byte) (*big.Int, error) {
	return _ChallengeManagerImpl.Contract.GetCurrentPsTimer(&_ChallengeManagerImpl.CallOpts, vId)
}

// GetCurrentPsTimer is a free data retrieval call binding the contract method 0x8ac04349.
//
// Solidity: function getCurrentPsTimer(bytes32 vId) view returns(uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) GetCurrentPsTimer(vId [32]byte) (*big.Int, error) {
	return _ChallengeManagerImpl.Contract.GetCurrentPsTimer(&_ChallengeManagerImpl.CallOpts, vId)
}

// GetVertex is a free data retrieval call binding the contract method 0x86f048ed.
//
// Solidity: function getVertex(bytes32 vId) view returns((bytes32,bytes32,uint256,bytes32,bytes32,bytes32,address,uint8,bytes32,uint256,uint256,bytes32))
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) GetVertex(opts *bind.CallOpts, vId [32]byte) (ChallengeVertex, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "getVertex", vId)

	if err != nil {
		return *new(ChallengeVertex), err
	}

	out0 := *abi.ConvertType(out[0], new(ChallengeVertex)).(*ChallengeVertex)

	return out0, err

}

// GetVertex is a free data retrieval call binding the contract method 0x86f048ed.
//
// Solidity: function getVertex(bytes32 vId) view returns((bytes32,bytes32,uint256,bytes32,bytes32,bytes32,address,uint8,bytes32,uint256,uint256,bytes32))
func (_ChallengeManagerImpl *ChallengeManagerImplSession) GetVertex(vId [32]byte) (ChallengeVertex, error) {
	return _ChallengeManagerImpl.Contract.GetVertex(&_ChallengeManagerImpl.CallOpts, vId)
}

// GetVertex is a free data retrieval call binding the contract method 0x86f048ed.
//
// Solidity: function getVertex(bytes32 vId) view returns((bytes32,bytes32,uint256,bytes32,bytes32,bytes32,address,uint8,bytes32,uint256,uint256,bytes32))
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) GetVertex(vId [32]byte) (ChallengeVertex, error) {
	return _ChallengeManagerImpl.Contract.GetVertex(&_ChallengeManagerImpl.CallOpts, vId)
}

// HasConfirmedSibling is a free data retrieval call binding the contract method 0x98b67d59.
//
// Solidity: function hasConfirmedSibling(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) HasConfirmedSibling(opts *bind.CallOpts, vId [32]byte) (bool, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "hasConfirmedSibling", vId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasConfirmedSibling is a free data retrieval call binding the contract method 0x98b67d59.
//
// Solidity: function hasConfirmedSibling(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) HasConfirmedSibling(vId [32]byte) (bool, error) {
	return _ChallengeManagerImpl.Contract.HasConfirmedSibling(&_ChallengeManagerImpl.CallOpts, vId)
}

// HasConfirmedSibling is a free data retrieval call binding the contract method 0x98b67d59.
//
// Solidity: function hasConfirmedSibling(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) HasConfirmedSibling(vId [32]byte) (bool, error) {
	return _ChallengeManagerImpl.Contract.HasConfirmedSibling(&_ChallengeManagerImpl.CallOpts, vId)
}

// IsPresumptiveSuccessor is a free data retrieval call binding the contract method 0xe41b5058.
//
// Solidity: function isPresumptiveSuccessor(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) IsPresumptiveSuccessor(opts *bind.CallOpts, vId [32]byte) (bool, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "isPresumptiveSuccessor", vId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsPresumptiveSuccessor is a free data retrieval call binding the contract method 0xe41b5058.
//
// Solidity: function isPresumptiveSuccessor(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) IsPresumptiveSuccessor(vId [32]byte) (bool, error) {
	return _ChallengeManagerImpl.Contract.IsPresumptiveSuccessor(&_ChallengeManagerImpl.CallOpts, vId)
}

// IsPresumptiveSuccessor is a free data retrieval call binding the contract method 0xe41b5058.
//
// Solidity: function isPresumptiveSuccessor(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) IsPresumptiveSuccessor(vId [32]byte) (bool, error) {
	return _ChallengeManagerImpl.Contract.IsPresumptiveSuccessor(&_ChallengeManagerImpl.CallOpts, vId)
}

// MiniStakeValue is a free data retrieval call binding the contract method 0x59c69996.
//
// Solidity: function miniStakeValue() view returns(uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) MiniStakeValue(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "miniStakeValue")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MiniStakeValue is a free data retrieval call binding the contract method 0x59c69996.
//
// Solidity: function miniStakeValue() view returns(uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) MiniStakeValue() (*big.Int, error) {
	return _ChallengeManagerImpl.Contract.MiniStakeValue(&_ChallengeManagerImpl.CallOpts)
}

// MiniStakeValue is a free data retrieval call binding the contract method 0x59c69996.
//
// Solidity: function miniStakeValue() view returns(uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) MiniStakeValue() (*big.Int, error) {
	return _ChallengeManagerImpl.Contract.MiniStakeValue(&_ChallengeManagerImpl.CallOpts)
}

// VertexExists is a free data retrieval call binding the contract method 0x6b0b2592.
//
// Solidity: function vertexExists(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) VertexExists(opts *bind.CallOpts, vId [32]byte) (bool, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "vertexExists", vId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// VertexExists is a free data retrieval call binding the contract method 0x6b0b2592.
//
// Solidity: function vertexExists(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) VertexExists(vId [32]byte) (bool, error) {
	return _ChallengeManagerImpl.Contract.VertexExists(&_ChallengeManagerImpl.CallOpts, vId)
}

// VertexExists is a free data retrieval call binding the contract method 0x6b0b2592.
//
// Solidity: function vertexExists(bytes32 vId) view returns(bool)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) VertexExists(vId [32]byte) (bool, error) {
	return _ChallengeManagerImpl.Contract.VertexExists(&_ChallengeManagerImpl.CallOpts, vId)
}

// Vertices is a free data retrieval call binding the contract method 0xf4f81db2.
//
// Solidity: function vertices(bytes32 ) view returns(bytes32 challengeId, bytes32 historyRoot, uint256 height, bytes32 successionChallenge, bytes32 predecessorId, bytes32 claimId, address staker, uint8 status, bytes32 psId, uint256 psLastUpdatedTimestamp, uint256 flushedPsTimeSec, bytes32 lowestHeightSuccessorId)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) Vertices(opts *bind.CallOpts, arg0 [32]byte) (struct {
	ChallengeId             [32]byte
	HistoryRoot             [32]byte
	Height                  *big.Int
	SuccessionChallenge     [32]byte
	PredecessorId           [32]byte
	ClaimId                 [32]byte
	Staker                  common.Address
	Status                  uint8
	PsId                    [32]byte
	PsLastUpdatedTimestamp  *big.Int
	FlushedPsTimeSec        *big.Int
	LowestHeightSuccessorId [32]byte
}, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "vertices", arg0)

	outstruct := new(struct {
		ChallengeId             [32]byte
		HistoryRoot             [32]byte
		Height                  *big.Int
		SuccessionChallenge     [32]byte
		PredecessorId           [32]byte
		ClaimId                 [32]byte
		Staker                  common.Address
		Status                  uint8
		PsId                    [32]byte
		PsLastUpdatedTimestamp  *big.Int
		FlushedPsTimeSec        *big.Int
		LowestHeightSuccessorId [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ChallengeId = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.HistoryRoot = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.Height = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.SuccessionChallenge = *abi.ConvertType(out[3], new([32]byte)).(*[32]byte)
	outstruct.PredecessorId = *abi.ConvertType(out[4], new([32]byte)).(*[32]byte)
	outstruct.ClaimId = *abi.ConvertType(out[5], new([32]byte)).(*[32]byte)
	outstruct.Staker = *abi.ConvertType(out[6], new(common.Address)).(*common.Address)
	outstruct.Status = *abi.ConvertType(out[7], new(uint8)).(*uint8)
	outstruct.PsId = *abi.ConvertType(out[8], new([32]byte)).(*[32]byte)
	outstruct.PsLastUpdatedTimestamp = *abi.ConvertType(out[9], new(*big.Int)).(**big.Int)
	outstruct.FlushedPsTimeSec = *abi.ConvertType(out[10], new(*big.Int)).(**big.Int)
	outstruct.LowestHeightSuccessorId = *abi.ConvertType(out[11], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// Vertices is a free data retrieval call binding the contract method 0xf4f81db2.
//
// Solidity: function vertices(bytes32 ) view returns(bytes32 challengeId, bytes32 historyRoot, uint256 height, bytes32 successionChallenge, bytes32 predecessorId, bytes32 claimId, address staker, uint8 status, bytes32 psId, uint256 psLastUpdatedTimestamp, uint256 flushedPsTimeSec, bytes32 lowestHeightSuccessorId)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) Vertices(arg0 [32]byte) (struct {
	ChallengeId             [32]byte
	HistoryRoot             [32]byte
	Height                  *big.Int
	SuccessionChallenge     [32]byte
	PredecessorId           [32]byte
	ClaimId                 [32]byte
	Staker                  common.Address
	Status                  uint8
	PsId                    [32]byte
	PsLastUpdatedTimestamp  *big.Int
	FlushedPsTimeSec        *big.Int
	LowestHeightSuccessorId [32]byte
}, error) {
	return _ChallengeManagerImpl.Contract.Vertices(&_ChallengeManagerImpl.CallOpts, arg0)
}

// Vertices is a free data retrieval call binding the contract method 0xf4f81db2.
//
// Solidity: function vertices(bytes32 ) view returns(bytes32 challengeId, bytes32 historyRoot, uint256 height, bytes32 successionChallenge, bytes32 predecessorId, bytes32 claimId, address staker, uint8 status, bytes32 psId, uint256 psLastUpdatedTimestamp, uint256 flushedPsTimeSec, bytes32 lowestHeightSuccessorId)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) Vertices(arg0 [32]byte) (struct {
	ChallengeId             [32]byte
	HistoryRoot             [32]byte
	Height                  *big.Int
	SuccessionChallenge     [32]byte
	PredecessorId           [32]byte
	ClaimId                 [32]byte
	Staker                  common.Address
	Status                  uint8
	PsId                    [32]byte
	PsLastUpdatedTimestamp  *big.Int
	FlushedPsTimeSec        *big.Int
	LowestHeightSuccessorId [32]byte
}, error) {
	return _ChallengeManagerImpl.Contract.Vertices(&_ChallengeManagerImpl.CallOpts, arg0)
}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) WinningClaim(opts *bind.CallOpts, challengeId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "winningClaim", challengeId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) WinningClaim(challengeId [32]byte) ([32]byte, error) {
	return _ChallengeManagerImpl.Contract.WinningClaim(&_ChallengeManagerImpl.CallOpts, challengeId)
}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) WinningClaim(challengeId [32]byte) ([32]byte, error) {
	return _ChallengeManagerImpl.Contract.WinningClaim(&_ChallengeManagerImpl.CallOpts, challengeId)
}

// AddLeaf is a paid mutator transaction binding the contract method 0x9e7cee54.
//
// Solidity: function addLeaf((bytes32,bytes32,uint256,bytes32,bytes32,bytes32[],bytes32,bytes32[]) leafData, bytes proof1, bytes proof2) payable returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactor) AddLeaf(opts *bind.TransactOpts, leafData AddLeafArgs, proof1 []byte, proof2 []byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.contract.Transact(opts, "addLeaf", leafData, proof1, proof2)
}

// AddLeaf is a paid mutator transaction binding the contract method 0x9e7cee54.
//
// Solidity: function addLeaf((bytes32,bytes32,uint256,bytes32,bytes32,bytes32[],bytes32,bytes32[]) leafData, bytes proof1, bytes proof2) payable returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) AddLeaf(leafData AddLeafArgs, proof1 []byte, proof2 []byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.AddLeaf(&_ChallengeManagerImpl.TransactOpts, leafData, proof1, proof2)
}

// AddLeaf is a paid mutator transaction binding the contract method 0x9e7cee54.
//
// Solidity: function addLeaf((bytes32,bytes32,uint256,bytes32,bytes32,bytes32[],bytes32,bytes32[]) leafData, bytes proof1, bytes proof2) payable returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorSession) AddLeaf(leafData AddLeafArgs, proof1 []byte, proof2 []byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.AddLeaf(&_ChallengeManagerImpl.TransactOpts, leafData, proof1, proof2)
}

// Bisect is a paid mutator transaction binding the contract method 0x359076cf.
//
// Solidity: function bisect(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactor) Bisect(opts *bind.TransactOpts, vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.contract.Transact(opts, "bisect", vId, prefixHistoryRoot, prefixProof)
}

// Bisect is a paid mutator transaction binding the contract method 0x359076cf.
//
// Solidity: function bisect(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) Bisect(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.Bisect(&_ChallengeManagerImpl.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// Bisect is a paid mutator transaction binding the contract method 0x359076cf.
//
// Solidity: function bisect(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorSession) Bisect(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.Bisect(&_ChallengeManagerImpl.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// ConfirmForPsTimer is a paid mutator transaction binding the contract method 0x1d5618ac.
//
// Solidity: function confirmForPsTimer(bytes32 vId) returns()
func (_ChallengeManagerImpl *ChallengeManagerImplTransactor) ConfirmForPsTimer(opts *bind.TransactOpts, vId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.contract.Transact(opts, "confirmForPsTimer", vId)
}

// ConfirmForPsTimer is a paid mutator transaction binding the contract method 0x1d5618ac.
//
// Solidity: function confirmForPsTimer(bytes32 vId) returns()
func (_ChallengeManagerImpl *ChallengeManagerImplSession) ConfirmForPsTimer(vId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.ConfirmForPsTimer(&_ChallengeManagerImpl.TransactOpts, vId)
}

// ConfirmForPsTimer is a paid mutator transaction binding the contract method 0x1d5618ac.
//
// Solidity: function confirmForPsTimer(bytes32 vId) returns()
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorSession) ConfirmForPsTimer(vId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.ConfirmForPsTimer(&_ChallengeManagerImpl.TransactOpts, vId)
}

// ConfirmForSucessionChallengeWin is a paid mutator transaction binding the contract method 0xd1bac9a4.
//
// Solidity: function confirmForSucessionChallengeWin(bytes32 vId) returns()
func (_ChallengeManagerImpl *ChallengeManagerImplTransactor) ConfirmForSucessionChallengeWin(opts *bind.TransactOpts, vId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.contract.Transact(opts, "confirmForSucessionChallengeWin", vId)
}

// ConfirmForSucessionChallengeWin is a paid mutator transaction binding the contract method 0xd1bac9a4.
//
// Solidity: function confirmForSucessionChallengeWin(bytes32 vId) returns()
func (_ChallengeManagerImpl *ChallengeManagerImplSession) ConfirmForSucessionChallengeWin(vId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.ConfirmForSucessionChallengeWin(&_ChallengeManagerImpl.TransactOpts, vId)
}

// ConfirmForSucessionChallengeWin is a paid mutator transaction binding the contract method 0xd1bac9a4.
//
// Solidity: function confirmForSucessionChallengeWin(bytes32 vId) returns()
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorSession) ConfirmForSucessionChallengeWin(vId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.ConfirmForSucessionChallengeWin(&_ChallengeManagerImpl.TransactOpts, vId)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0xf696dc55.
//
// Solidity: function createChallenge(bytes32 assertionId) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactor) CreateChallenge(opts *bind.TransactOpts, assertionId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.contract.Transact(opts, "createChallenge", assertionId)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0xf696dc55.
//
// Solidity: function createChallenge(bytes32 assertionId) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) CreateChallenge(assertionId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.CreateChallenge(&_ChallengeManagerImpl.TransactOpts, assertionId)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0xf696dc55.
//
// Solidity: function createChallenge(bytes32 assertionId) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorSession) CreateChallenge(assertionId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.CreateChallenge(&_ChallengeManagerImpl.TransactOpts, assertionId)
}

// CreateSubChallenge is a paid mutator transaction binding the contract method 0xbd623251.
//
// Solidity: function createSubChallenge(bytes32 vId) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactor) CreateSubChallenge(opts *bind.TransactOpts, vId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.contract.Transact(opts, "createSubChallenge", vId)
}

// CreateSubChallenge is a paid mutator transaction binding the contract method 0xbd623251.
//
// Solidity: function createSubChallenge(bytes32 vId) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) CreateSubChallenge(vId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.CreateSubChallenge(&_ChallengeManagerImpl.TransactOpts, vId)
}

// CreateSubChallenge is a paid mutator transaction binding the contract method 0xbd623251.
//
// Solidity: function createSubChallenge(bytes32 vId) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorSession) CreateSubChallenge(vId [32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.CreateSubChallenge(&_ChallengeManagerImpl.TransactOpts, vId)
}

// ExecuteOneStep is a paid mutator transaction binding the contract method 0x1421ba3d.
//
// Solidity: function executeOneStep(bytes32 winnerVId, ((uint256,address),uint256,bytes32,bytes) oneStepData, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactor) ExecuteOneStep(opts *bind.TransactOpts, winnerVId [32]byte, oneStepData OneStepData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.contract.Transact(opts, "executeOneStep", winnerVId, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ExecuteOneStep is a paid mutator transaction binding the contract method 0x1421ba3d.
//
// Solidity: function executeOneStep(bytes32 winnerVId, ((uint256,address),uint256,bytes32,bytes) oneStepData, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) ExecuteOneStep(winnerVId [32]byte, oneStepData OneStepData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.ExecuteOneStep(&_ChallengeManagerImpl.TransactOpts, winnerVId, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ExecuteOneStep is a paid mutator transaction binding the contract method 0x1421ba3d.
//
// Solidity: function executeOneStep(bytes32 winnerVId, ((uint256,address),uint256,bytes32,bytes) oneStepData, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorSession) ExecuteOneStep(winnerVId [32]byte, oneStepData OneStepData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.ExecuteOneStep(&_ChallengeManagerImpl.TransactOpts, winnerVId, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// Initialize is a paid mutator transaction binding the contract method 0x9e3d87cd.
//
// Solidity: function initialize(address _assertionChain, uint256 _miniStakeValue, uint256 _challengePeriodSec, address _oneStepProofEntry) returns()
func (_ChallengeManagerImpl *ChallengeManagerImplTransactor) Initialize(opts *bind.TransactOpts, _assertionChain common.Address, _miniStakeValue *big.Int, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _ChallengeManagerImpl.contract.Transact(opts, "initialize", _assertionChain, _miniStakeValue, _challengePeriodSec, _oneStepProofEntry)
}

// Initialize is a paid mutator transaction binding the contract method 0x9e3d87cd.
//
// Solidity: function initialize(address _assertionChain, uint256 _miniStakeValue, uint256 _challengePeriodSec, address _oneStepProofEntry) returns()
func (_ChallengeManagerImpl *ChallengeManagerImplSession) Initialize(_assertionChain common.Address, _miniStakeValue *big.Int, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.Initialize(&_ChallengeManagerImpl.TransactOpts, _assertionChain, _miniStakeValue, _challengePeriodSec, _oneStepProofEntry)
}

// Initialize is a paid mutator transaction binding the contract method 0x9e3d87cd.
//
// Solidity: function initialize(address _assertionChain, uint256 _miniStakeValue, uint256 _challengePeriodSec, address _oneStepProofEntry) returns()
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorSession) Initialize(_assertionChain common.Address, _miniStakeValue *big.Int, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.Initialize(&_ChallengeManagerImpl.TransactOpts, _assertionChain, _miniStakeValue, _challengePeriodSec, _oneStepProofEntry)
}

// Merge is a paid mutator transaction binding the contract method 0x597e1e0b.
//
// Solidity: function merge(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactor) Merge(opts *bind.TransactOpts, vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.contract.Transact(opts, "merge", vId, prefixHistoryRoot, prefixProof)
}

// Merge is a paid mutator transaction binding the contract method 0x597e1e0b.
//
// Solidity: function merge(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) Merge(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.Merge(&_ChallengeManagerImpl.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// Merge is a paid mutator transaction binding the contract method 0x597e1e0b.
//
// Solidity: function merge(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_ChallengeManagerImpl *ChallengeManagerImplTransactorSession) Merge(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _ChallengeManagerImpl.Contract.Merge(&_ChallengeManagerImpl.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// ChallengeManagerImplBisectedIterator is returned from FilterBisected and is used to iterate over the raw logs and unpacked data for Bisected events raised by the ChallengeManagerImpl contract.
type ChallengeManagerImplBisectedIterator struct {
	Event *ChallengeManagerImplBisected // Event containing the contract specifics and raw log

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
func (it *ChallengeManagerImplBisectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChallengeManagerImplBisected)
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
		it.Event = new(ChallengeManagerImplBisected)
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
func (it *ChallengeManagerImplBisectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChallengeManagerImplBisectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChallengeManagerImplBisected represents a Bisected event raised by the ChallengeManagerImpl contract.
type ChallengeManagerImplBisected struct {
	FromId [32]byte
	ToId   [32]byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBisected is a free log retrieval operation binding the contract event 0x69d5465c81edf7aaaf2e5c6c8829500df87d84c87f8d5b1221b59eaeaca70d27.
//
// Solidity: event Bisected(bytes32 fromId, bytes32 toId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) FilterBisected(opts *bind.FilterOpts) (*ChallengeManagerImplBisectedIterator, error) {

	logs, sub, err := _ChallengeManagerImpl.contract.FilterLogs(opts, "Bisected")
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerImplBisectedIterator{contract: _ChallengeManagerImpl.contract, event: "Bisected", logs: logs, sub: sub}, nil
}

// WatchBisected is a free log subscription operation binding the contract event 0x69d5465c81edf7aaaf2e5c6c8829500df87d84c87f8d5b1221b59eaeaca70d27.
//
// Solidity: event Bisected(bytes32 fromId, bytes32 toId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) WatchBisected(opts *bind.WatchOpts, sink chan<- *ChallengeManagerImplBisected) (event.Subscription, error) {

	logs, sub, err := _ChallengeManagerImpl.contract.WatchLogs(opts, "Bisected")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChallengeManagerImplBisected)
				if err := _ChallengeManagerImpl.contract.UnpackLog(event, "Bisected", log); err != nil {
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

// ParseBisected is a log parse operation binding the contract event 0x69d5465c81edf7aaaf2e5c6c8829500df87d84c87f8d5b1221b59eaeaca70d27.
//
// Solidity: event Bisected(bytes32 fromId, bytes32 toId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) ParseBisected(log types.Log) (*ChallengeManagerImplBisected, error) {
	event := new(ChallengeManagerImplBisected)
	if err := _ChallengeManagerImpl.contract.UnpackLog(event, "Bisected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChallengeManagerImplChallengeCreatedIterator is returned from FilterChallengeCreated and is used to iterate over the raw logs and unpacked data for ChallengeCreated events raised by the ChallengeManagerImpl contract.
type ChallengeManagerImplChallengeCreatedIterator struct {
	Event *ChallengeManagerImplChallengeCreated // Event containing the contract specifics and raw log

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
func (it *ChallengeManagerImplChallengeCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChallengeManagerImplChallengeCreated)
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
		it.Event = new(ChallengeManagerImplChallengeCreated)
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
func (it *ChallengeManagerImplChallengeCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChallengeManagerImplChallengeCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChallengeManagerImplChallengeCreated represents a ChallengeCreated event raised by the ChallengeManagerImpl contract.
type ChallengeManagerImplChallengeCreated struct {
	ChallengeId [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChallengeCreated is a free log retrieval operation binding the contract event 0x867c977ac47adb20fcc4fb6b981269b44d23560057a29eed03cd5afb81750b34.
//
// Solidity: event ChallengeCreated(bytes32 challengeId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) FilterChallengeCreated(opts *bind.FilterOpts) (*ChallengeManagerImplChallengeCreatedIterator, error) {

	logs, sub, err := _ChallengeManagerImpl.contract.FilterLogs(opts, "ChallengeCreated")
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerImplChallengeCreatedIterator{contract: _ChallengeManagerImpl.contract, event: "ChallengeCreated", logs: logs, sub: sub}, nil
}

// WatchChallengeCreated is a free log subscription operation binding the contract event 0x867c977ac47adb20fcc4fb6b981269b44d23560057a29eed03cd5afb81750b34.
//
// Solidity: event ChallengeCreated(bytes32 challengeId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) WatchChallengeCreated(opts *bind.WatchOpts, sink chan<- *ChallengeManagerImplChallengeCreated) (event.Subscription, error) {

	logs, sub, err := _ChallengeManagerImpl.contract.WatchLogs(opts, "ChallengeCreated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChallengeManagerImplChallengeCreated)
				if err := _ChallengeManagerImpl.contract.UnpackLog(event, "ChallengeCreated", log); err != nil {
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

// ParseChallengeCreated is a log parse operation binding the contract event 0x867c977ac47adb20fcc4fb6b981269b44d23560057a29eed03cd5afb81750b34.
//
// Solidity: event ChallengeCreated(bytes32 challengeId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) ParseChallengeCreated(log types.Log) (*ChallengeManagerImplChallengeCreated, error) {
	event := new(ChallengeManagerImplChallengeCreated)
	if err := _ChallengeManagerImpl.contract.UnpackLog(event, "ChallengeCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChallengeManagerImplMergedIterator is returned from FilterMerged and is used to iterate over the raw logs and unpacked data for Merged events raised by the ChallengeManagerImpl contract.
type ChallengeManagerImplMergedIterator struct {
	Event *ChallengeManagerImplMerged // Event containing the contract specifics and raw log

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
func (it *ChallengeManagerImplMergedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChallengeManagerImplMerged)
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
		it.Event = new(ChallengeManagerImplMerged)
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
func (it *ChallengeManagerImplMergedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChallengeManagerImplMergedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChallengeManagerImplMerged represents a Merged event raised by the ChallengeManagerImpl contract.
type ChallengeManagerImplMerged struct {
	FromId [32]byte
	ToId   [32]byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterMerged is a free log retrieval operation binding the contract event 0x72b50597145599e4288d411331c925b40b33b0fa3cccadc1f57d2a1ab973553a.
//
// Solidity: event Merged(bytes32 fromId, bytes32 toId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) FilterMerged(opts *bind.FilterOpts) (*ChallengeManagerImplMergedIterator, error) {

	logs, sub, err := _ChallengeManagerImpl.contract.FilterLogs(opts, "Merged")
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerImplMergedIterator{contract: _ChallengeManagerImpl.contract, event: "Merged", logs: logs, sub: sub}, nil
}

// WatchMerged is a free log subscription operation binding the contract event 0x72b50597145599e4288d411331c925b40b33b0fa3cccadc1f57d2a1ab973553a.
//
// Solidity: event Merged(bytes32 fromId, bytes32 toId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) WatchMerged(opts *bind.WatchOpts, sink chan<- *ChallengeManagerImplMerged) (event.Subscription, error) {

	logs, sub, err := _ChallengeManagerImpl.contract.WatchLogs(opts, "Merged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChallengeManagerImplMerged)
				if err := _ChallengeManagerImpl.contract.UnpackLog(event, "Merged", log); err != nil {
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

// ParseMerged is a log parse operation binding the contract event 0x72b50597145599e4288d411331c925b40b33b0fa3cccadc1f57d2a1ab973553a.
//
// Solidity: event Merged(bytes32 fromId, bytes32 toId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) ParseMerged(log types.Log) (*ChallengeManagerImplMerged, error) {
	event := new(ChallengeManagerImplMerged)
	if err := _ChallengeManagerImpl.contract.UnpackLog(event, "Merged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChallengeManagerImplVertexAddedIterator is returned from FilterVertexAdded and is used to iterate over the raw logs and unpacked data for VertexAdded events raised by the ChallengeManagerImpl contract.
type ChallengeManagerImplVertexAddedIterator struct {
	Event *ChallengeManagerImplVertexAdded // Event containing the contract specifics and raw log

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
func (it *ChallengeManagerImplVertexAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChallengeManagerImplVertexAdded)
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
		it.Event = new(ChallengeManagerImplVertexAdded)
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
func (it *ChallengeManagerImplVertexAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChallengeManagerImplVertexAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChallengeManagerImplVertexAdded represents a VertexAdded event raised by the ChallengeManagerImpl contract.
type ChallengeManagerImplVertexAdded struct {
	VertexId [32]byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterVertexAdded is a free log retrieval operation binding the contract event 0x4383ba11a7cd16be5880c5f674b93be38b3b1fcafd7a7b06151998fa2a675349.
//
// Solidity: event VertexAdded(bytes32 vertexId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) FilterVertexAdded(opts *bind.FilterOpts) (*ChallengeManagerImplVertexAddedIterator, error) {

	logs, sub, err := _ChallengeManagerImpl.contract.FilterLogs(opts, "VertexAdded")
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerImplVertexAddedIterator{contract: _ChallengeManagerImpl.contract, event: "VertexAdded", logs: logs, sub: sub}, nil
}

// WatchVertexAdded is a free log subscription operation binding the contract event 0x4383ba11a7cd16be5880c5f674b93be38b3b1fcafd7a7b06151998fa2a675349.
//
// Solidity: event VertexAdded(bytes32 vertexId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) WatchVertexAdded(opts *bind.WatchOpts, sink chan<- *ChallengeManagerImplVertexAdded) (event.Subscription, error) {

	logs, sub, err := _ChallengeManagerImpl.contract.WatchLogs(opts, "VertexAdded")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChallengeManagerImplVertexAdded)
				if err := _ChallengeManagerImpl.contract.UnpackLog(event, "VertexAdded", log); err != nil {
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

// ParseVertexAdded is a log parse operation binding the contract event 0x4383ba11a7cd16be5880c5f674b93be38b3b1fcafd7a7b06151998fa2a675349.
//
// Solidity: event VertexAdded(bytes32 vertexId)
func (_ChallengeManagerImpl *ChallengeManagerImplFilterer) ParseVertexAdded(log types.Log) (*ChallengeManagerImplVertexAdded, error) {
	event := new(ChallengeManagerImplVertexAdded)
	if err := _ChallengeManagerImpl.contract.UnpackLog(event, "VertexAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ChallengeManagerLibMetaData contains all meta data concerning the ChallengeManagerLib contract.
var ChallengeManagerLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212205167bab34233fd30192af121952a554322b1f0ebf8c867f1d83e143e132a149664736f6c63430008110033",
}

// ChallengeManagerLibABI is the input ABI used to generate the binding from.
// Deprecated: Use ChallengeManagerLibMetaData.ABI instead.
var ChallengeManagerLibABI = ChallengeManagerLibMetaData.ABI

// ChallengeManagerLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ChallengeManagerLibMetaData.Bin instead.
var ChallengeManagerLibBin = ChallengeManagerLibMetaData.Bin

// DeployChallengeManagerLib deploys a new Ethereum contract, binding an instance of ChallengeManagerLib to it.
func DeployChallengeManagerLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ChallengeManagerLib, error) {
	parsed, err := ChallengeManagerLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ChallengeManagerLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ChallengeManagerLib{ChallengeManagerLibCaller: ChallengeManagerLibCaller{contract: contract}, ChallengeManagerLibTransactor: ChallengeManagerLibTransactor{contract: contract}, ChallengeManagerLibFilterer: ChallengeManagerLibFilterer{contract: contract}}, nil
}

// ChallengeManagerLib is an auto generated Go binding around an Ethereum contract.
type ChallengeManagerLib struct {
	ChallengeManagerLibCaller     // Read-only binding to the contract
	ChallengeManagerLibTransactor // Write-only binding to the contract
	ChallengeManagerLibFilterer   // Log filterer for contract events
}

// ChallengeManagerLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type ChallengeManagerLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChallengeManagerLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ChallengeManagerLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChallengeManagerLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ChallengeManagerLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChallengeManagerLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ChallengeManagerLibSession struct {
	Contract     *ChallengeManagerLib // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ChallengeManagerLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ChallengeManagerLibCallerSession struct {
	Contract *ChallengeManagerLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// ChallengeManagerLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ChallengeManagerLibTransactorSession struct {
	Contract     *ChallengeManagerLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// ChallengeManagerLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type ChallengeManagerLibRaw struct {
	Contract *ChallengeManagerLib // Generic contract binding to access the raw methods on
}

// ChallengeManagerLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ChallengeManagerLibCallerRaw struct {
	Contract *ChallengeManagerLibCaller // Generic read-only contract binding to access the raw methods on
}

// ChallengeManagerLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ChallengeManagerLibTransactorRaw struct {
	Contract *ChallengeManagerLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewChallengeManagerLib creates a new instance of ChallengeManagerLib, bound to a specific deployed contract.
func NewChallengeManagerLib(address common.Address, backend bind.ContractBackend) (*ChallengeManagerLib, error) {
	contract, err := bindChallengeManagerLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerLib{ChallengeManagerLibCaller: ChallengeManagerLibCaller{contract: contract}, ChallengeManagerLibTransactor: ChallengeManagerLibTransactor{contract: contract}, ChallengeManagerLibFilterer: ChallengeManagerLibFilterer{contract: contract}}, nil
}

// NewChallengeManagerLibCaller creates a new read-only instance of ChallengeManagerLib, bound to a specific deployed contract.
func NewChallengeManagerLibCaller(address common.Address, caller bind.ContractCaller) (*ChallengeManagerLibCaller, error) {
	contract, err := bindChallengeManagerLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerLibCaller{contract: contract}, nil
}

// NewChallengeManagerLibTransactor creates a new write-only instance of ChallengeManagerLib, bound to a specific deployed contract.
func NewChallengeManagerLibTransactor(address common.Address, transactor bind.ContractTransactor) (*ChallengeManagerLibTransactor, error) {
	contract, err := bindChallengeManagerLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerLibTransactor{contract: contract}, nil
}

// NewChallengeManagerLibFilterer creates a new log filterer instance of ChallengeManagerLib, bound to a specific deployed contract.
func NewChallengeManagerLibFilterer(address common.Address, filterer bind.ContractFilterer) (*ChallengeManagerLibFilterer, error) {
	contract, err := bindChallengeManagerLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ChallengeManagerLibFilterer{contract: contract}, nil
}

// bindChallengeManagerLib binds a generic wrapper to an already deployed contract.
func bindChallengeManagerLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ChallengeManagerLibABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ChallengeManagerLib *ChallengeManagerLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ChallengeManagerLib.Contract.ChallengeManagerLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ChallengeManagerLib *ChallengeManagerLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChallengeManagerLib.Contract.ChallengeManagerLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ChallengeManagerLib *ChallengeManagerLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ChallengeManagerLib.Contract.ChallengeManagerLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ChallengeManagerLib *ChallengeManagerLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ChallengeManagerLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ChallengeManagerLib *ChallengeManagerLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ChallengeManagerLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ChallengeManagerLib *ChallengeManagerLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ChallengeManagerLib.Contract.contract.Transact(opts, method, params...)
}

// EdgeChallengeManagerMetaData contains all meta data concerning the EdgeChallengeManager contract.
var EdgeChallengeManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriodSec\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"actualLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxProofLength\",\"type\":\"uint256\"}],\"name\":\"MerkleProofTooLong\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"bisectedEdgeId\",\"type\":\"bytes32\"}],\"name\":\"Bisected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"LevelZeroEdgeAdded\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"bisectionHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"bisectEdge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"name\":\"calculateEdgeId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"}],\"name\":\"calculateMutualId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"challengePeriodSec\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"confirmEdgeByChildren\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimingEdgeId\",\"type\":\"bytes32\"}],\"name\":\"confirmEdgeByClaim\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"components\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"machineStep\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"internalType\":\"structOneStepData\",\"name\":\"oneStepData\",\"type\":\"tuple\"},{\"internalType\":\"bytes32[]\",\"name\":\"beforeHistoryInclusionProof\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"afterHistoryInclusionProof\",\"type\":\"bytes32[]\"}],\"name\":\"confirmEdgeByOneStepProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"ancestorEdges\",\"type\":\"bytes32[]\"}],\"name\":\"confirmEdgeByTime\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"}],\"internalType\":\"structCreateEdgeArgs\",\"name\":\"args\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"createLayerZeroEdge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"edgeExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"edgeLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"firstRival\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"getEdge\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"lowerChildId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"upperChildId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"createdWhen\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"enumEdgeStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"enumEdgeType\",\"name\":\"eType\",\"type\":\"uint8\"}],\"internalType\":\"structChallengeEdge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"hasLengthOneRival\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"hasRival\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriodSec\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"}],\"name\":\"timeUnrivaled\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b5060405162003ff038038062003ff08339810160408190526200003491620000e4565b620000418383836200004a565b5050506200012c565b6003546001600160a01b031615620000975760405162461bcd60e51b815260206004820152600c60248201526b1053149150511657d253925560a21b604482015260640160405180910390fd5b600380546001600160a01b039485166001600160a01b03199182161790915560029290925560048054919093169116179055565b6001600160a01b0381168114620000e157600080fd5b50565b600080600060608486031215620000fa57600080fd5b83516200010781620000cb565b6020850151604086015191945092506200012181620000cb565b809150509250925092565b613eb4806200013c6000396000f3fe6080604052600436106100dd5760003560e01c8063908517e911610085578063908517e9146101f057806392fd750b14610210578063bce6f54f14610230578063c32d8c631461025d578063c350a1b51461027d578063c8bc4e431461029d578063e5b6c5b8146102d2578063eae0328b146102f2578063fda2892e1461031257600080fd5b80624d8efe146100e25780630f73bfad1461011557806316a0ca0b146101375780632eaa00431461014a5780633e35f5e81461016a57806354b641511461018a578063654f0dc2146101ba578063750e0c0f146101d0575b600080fd5b3480156100ee57600080fd5b506101026100fd366004613382565b61033f565b6040519081526020015b60405180910390f35b34801561012157600080fd5b506101356101303660046133cc565b61035a565b005b6101026101453660046134c6565b61036a565b34801561015657600080fd5b50610135610165366004613595565b6106a4565b34801561017657600080fd5b50610102610185366004613595565b6106b2565b34801561019657600080fd5b506101aa6101a5366004613595565b6106c4565b604051901515815260200161010c565b3480156101c657600080fd5b5061010260025481565b3480156101dc57600080fd5b506101aa6101eb366004613595565b6106d0565b3480156101fc57600080fd5b506101aa61020b366004613595565b6106e9565b34801561021c57600080fd5b5061013561022b3660046135d1565b6106f5565b34801561023c57600080fd5b5061010261024b366004613595565b60009081526001602052604090205490565b34801561026957600080fd5b50610102610278366004613672565b610708565b34801561028957600080fd5b506101356102983660046136c9565b610721565b3480156102a957600080fd5b506102bd6102b836600461377a565b61079d565b6040805192835260208301919091520161010c565b3480156102de57600080fd5b506101356102ed36600461380d565b6107b8565b3480156102fe57600080fd5b5061010261030d366004613595565b61084a565b34801561031e57600080fd5b5061033261032d366004613595565b61085e565b60405161010c91906138ec565b600061034f87878787878761095f565b979650505050505050565b610366600083836109a4565b5050565b6000808087516002811115610381576103816138b2565b036104055760035460a0880151604051632729597560e21b81526001600160a01b0390921691639ca565d4916103bd9160040190815260200190565b602060405180830381865afa1580156103da573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906103fe9190613990565b905061061d565b60018751600281111561041a5761041a6138b2565b0361053457600060a088015161043290600090610bcb565b60090154600160a81b900460ff166002811115610451576104516138b2565b146104ad5760405162461bcd60e51b815260206004820152602160248201527f436c61696d206368616c6c656e67652074797065206973206e6f7420426c6f636044820152606b60f81b60648201526084015b60405180910390fd5b6104cd6104c88860a001516000610bcb90919063ffffffff16565b610c0c565b90506104e78760a001516000610c3c90919063ffffffff16565b61052f5760405162461bcd60e51b815260206004820152601960248201527810db185a5b48191bd95cc81b9bdd081a185d99481c9a5d985b603a1b60448201526064016104a4565b61061d565b600287516002811115610549576105496138b2565b036105d957600160a088015161056190600090610bcb565b60090154600160a81b900460ff166002811115610580576105806138b2565b146104ad5760405162461bcd60e51b815260206004820152602360248201527f436c61696d206368616c6c656e67652074797065206973206e6f74204269675360448201526207465760ec1b60648201526084016104a4565b60405162461bcd60e51b8152602060048201526019602482015278556e6578706563746564206368616c6c656e6765207479706560381b60448201526064016104a4565b60006106478289602001518a604001518b606001518c608001518d60a00151338f60000151610cea565b9050610654600082610e16565b7f102ba5fcc71c9f7d7075d3f9cc9cb52fe4feb2cb843bef52f5f9fe9825b539e561067e826110a0565b60405190815260200160405180910390a1610698816110a0565b98975050505050505050565b6106af6000826110c9565b50565b60006106be8183611361565b92915050565b60006106be81836114fa565b60008181526020819052604081206007015415156106be565b60006106be8183610c3c565b600254610366906000908490849061152f565b600061071786868686866117f3565b9695505050505050565b6003546001600160a01b0316156107695760405162461bcd60e51b815260206004820152600c60248201526b1053149150511657d253925560a21b60448201526064016104a4565b600380546001600160a01b039485166001600160a01b03199182161790915560029290925560048054919093169116179055565b6000806107ac8186868661182f565b91509150935093915050565b6004546108429087906001600160a01b03166107d3886139a9565b87878080602002602001604051908101604052809392919081815260200183836020028082843760009201919091525050604080516020808b0282810182019093528a82529093508a92508991829185019084908082843760009201829052509796959493925050611c039050565b505050505050565b60006106be6108598284610bcb565b611f34565b610866613310565b610871600083610bcb565b60408051610180810182528254815260018084015460208301526002840154928201929092526003830154606082015260048301546080820152600583015460a0820152600683015460c0820152600783015460e0820152600883015461010082015260098301546001600160a01b038116610120830152909291610140840191600160a01b900460ff169081111561090c5761090c6138b2565b600181111561091d5761091d6138b2565b81526020016009820160159054906101000a900460ff166002811115610945576109456138b2565b6002811115610956576109566138b2565b90525092915050565b600061096e87878787876117f3565b60408051602081019290925281018390526060016040516020818303038152906040528051906020012090509695505050505050565b6000828152602084905260409020600701546109d25760405162461bcd60e51b81526004016104a490613a5a565b60008083815260208590526040902060090154600160a01b900460ff166001811115610a0057610a006138b2565b14610a1d5760405162461bcd60e51b81526004016104a490613a87565b600081815260208490526040902060070154610a7b5760405162461bcd60e51b815260206004820152601c60248201527f436c61696d696e67206564676520646f6573206e6f742065786973740000000060448201526064016104a4565b6001600082815260208590526040902060090154600160a01b900460ff166001811115610aaa57610aaa6138b2565b14610af75760405162461bcd60e51b815260206004820152601b60248201527f436c61696d696e672065646765206e6f7420636f6e6669726d6564000000000060448201526064016104a4565b610b02838383611f6d565b6000818152602084905260409020600801548214610b5e5760405162461bcd60e51b8152602060048201526019602482015278436c61696d20646f6573206e6f74206d61746368206564676560381b60448201526064016104a4565b6000828152602084905260409020610b75906120a1565b6000828152602084905260409020610b8c90610c0c565b827fb924f3aa473645c7cf5b10262f927ae4ccf869d7fc239c17144b0c67490d1c7383604051610bbe91815260200190565b60405180910390a3505050565b600081815260208390526040812060070154610bf95760405162461bcd60e51b81526004016104a490613a5a565b5060009081526020919091526040902090565b60006106be8260090160159054906101000a900460ff1683600001548460020154856001015486600401546117f3565b600081815260208390526040812060070154610c6a5760405162461bcd60e51b81526004016104a490613a5a565b6000828152602084905260408120610c8190610c0c565b6000818152600186016020526040812054919250819003610cd85760405162461bcd60e51b8152602060048201526011602482015270115b5c1d1e48199a5c9cdd081c9a5d985b607a1b60448201526064016104a4565b610ce181612132565b95945050505050565b610cf2613310565b6001600160a01b038316610d375760405162461bcd60e51b815260206004820152600c60248201526b22b6b83a3c9039ba30b5b2b960a11b60448201526064016104a4565b6000849003610d795760405162461bcd60e51b815260206004820152600e60248201526d115b5c1d1e4818db185a5b481a5960921b60448201526064016104a4565b610d868989898989612163565b6040518061018001604052808a81526020018981526020018881526020018781526020018681526020016000801b81526020016000801b8152602001428152602001858152602001846001600160a01b0316815260200160006001811115610df057610df06138b2565b8152602001836002811115610e0757610e076138b2565b90529998505050505050505050565b6000610e21826110a0565b60008181526020859052604090206007015490915015610e795760405162461bcd60e51b81526020600482015260136024820152724564676520616c72656164792065786973747360681b60448201526064016104a4565b600081815260208481526040918290208451815590840151600180830191909155918401516002820155606084015160038201556080840151600482015560a0840151600582015560c0840151600682015560e0840151600782015561010084015160088201556101208401516009820180546001600160a01b039092166001600160a01b0319831681178255610140870151879590936001600160a81b03191690911790600160a01b908490811115610f3557610f356138b2565b021790555061016082015160098201805460ff60a81b1916600160a81b836002811115610f6457610f646138b2565b02179055509050506000610f9083610160015184600001518560400151866020015187608001516117f3565b6000818152600186016020526040812054919250818103610fe657604051602001610fba90613ab1565b60408051601f198184030181529181528151602092830120600086815260018a01909352912055611028565b604051602001610ff590613ab1565b60405160208183030381529060405280519060200120820361102857506000828152600180870160205260409091208490555b84516000858152602088905260409020849086907fddd14992ee7cd971b2a5cc510ebc7a33a1a7bd11dd74c3c5a83000328a0d590690859061106990611f34565b6101608b01516101208c0151604051611090949392916001600160a01b0316151590613ac6565b60405180910390a4505050505050565b60006106be8261016001518360000151846040015185602001518660800151876060015161095f565b6000818152602083905260409020600701546110f75760405162461bcd60e51b81526004016104a490613a5a565b60008082815260208490526040902060090154600160a01b900460ff166001811115611125576111256138b2565b146111425760405162461bcd60e51b81526004016104a490613a87565b600081815260208390526040808220600501548083529120600701546111aa5760405162461bcd60e51b815260206004820152601a60248201527f4c6f776572206368696c6420646f6573206e6f7420657869737400000000000060448201526064016104a4565b6001600082815260208590526040902060090154600160a01b900460ff1660018111156111d9576111d96138b2565b146112225760405162461bcd60e51b8152602060048201526019602482015278131bddd95c8818da1a5b19081b9bdd0818dbdb999a5c9b5959603a1b60448201526064016104a4565b6000828152602084905260408082206006015480835291206007015461128a5760405162461bcd60e51b815260206004820152601a60248201527f5570706572206368696c6420646f6573206e6f7420657869737400000000000060448201526064016104a4565b6001600082815260208690526040902060090154600160a01b900460ff1660018111156112b9576112b96138b2565b146113025760405162461bcd60e51b8152602060048201526019602482015278155c1c195c8818da1a5b19081b9bdd0818dbdb999a5c9b5959603a1b60448201526064016104a4565b6000838152602085905260409020611319906120a1565b600083815260208590526040902061133090610c0c565b60405184907f0d27fcaf1adc41547a5cfc99d2364f6c0dc7e81c9fc3fe8cb38abb409b48358a90600090a350505050565b60008181526020839052604081206007015461138f5760405162461bcd60e51b81526004016104a490613a5a565b60008281526020849052604081206113a690610c0c565b60008181526001860160205260408120549192508190036113fe5760405162461bcd60e51b8152602060048201526012602482015271115b5c1d1e481c9a5d985b081c9958dbdc9960721b60448201526064016104a4565b60405160200161140d90613ab1565b60405160208183030381529060405280519060200120810361144e576000848152602086905260409020600701546114459042613b0a565b925050506106be565b6000818152602086905260409020600701546114a85760405162461bcd60e51b8152602060048201526019602482015278149a5d985b08195919d948191bd95cc81b9bdd08195e1a5cdd603a1b60448201526064016104a4565b6000818152602086905260408082206007908101548784529190922090910154808211156114e5576114da8183613b0a565b9450505050506106be565b60009450505050506106be565b505092915050565b60006115068383610c3c565b80156115285750600082815260208490526040902061152490611f34565b6001145b9392505050565b60008381526020859052604090206007015461155d5760405162461bcd60e51b81526004016104a490613a5a565b60008084815260208690526040902060090154600160a01b900460ff16600181111561158b5761158b6138b2565b146115a85760405162461bcd60e51b81526004016104a490613a87565b8260006115b58683611361565b905060005b845181101561170d5760006115e8888784815181106115db576115db613b1d565b6020026020010151610bcb565b905083816005015414806115ff5750838160060154145b1561164357611616886116118361228e565b611361565b6116209084613b33565b925085828151811061163457611634613b1d565b602002602001015193506116fa565b600084815260208990526040902060080154865187908490811061166957611669613b1d565b6020026020010151036116a75761169a8887848151811061168c5761168c613b1d565b602002602001015186611f6d565b611616886116118361228e565b60405162461bcd60e51b815260206004820152602260248201527f43757272656e74206973206e6f742061206368696c64206f6620616e6365737460448201526137b960f11b60648201526084016104a4565b508061170581613b46565b9150506115ba565b508281116117835760405162461bcd60e51b815260206004820152603c60248201527f546f74616c2074696d6520756e726976616c6564206e6f74206772656174657260448201527f207468616e20636f6e6669726d6174696f6e207468726573686f6c640000000060648201526084016104a4565b600085815260208790526040902061179a906120a1565b60008581526020879052604090206117b190610c0c565b857f2e0808830a22204cb3fb8f8d784b28bc97e9ce2e39d2f9cde2860de0957d68eb836040516117e391815260200190565b60405180910390a3505050505050565b6000858585858560405160200161180e959493929190613b5f565b60405160208183030381529060405280519060200120905095945050505050565b60008080600086815260208890526040902060090154600160a01b900460ff166001811115611860576118606138b2565b1461187d5760405162461bcd60e51b81526004016104a490613a87565b6118878686610c3c565b6118d35760405162461bcd60e51b815260206004820152601f60248201527f43616e6e6f742062697365637420616e20756e726976616c656420656467650060448201526064016104a4565b60006118df8787610bcb565b60408051610180810182528254815260018084015460208301526002840154928201929092526003830154606082015260048301546080820152600583015460a0820152600683015460c0820152600783015460e0820152600883015461010082015260098301546001600160a01b038116610120830152909291610140840191600160a01b900460ff169081111561197a5761197a6138b2565b600181111561198b5761198b6138b2565b81526020016009820160159054906101000a900460ff1660028111156119b3576119b36138b2565b60028111156119c4576119c46138b2565b9052506000878152602089905260409020600501549091501580156119f85750600086815260208890526040902060060154155b611a405760405162461bcd60e51b815260206004820152601960248201527822b233b29030b63932b0b23c903430b99031b434b6323932b760391b60448201526064016104a4565b6000611a54826040015183608001516122c3565b905060008086806020019051810190611a6d9190613c00565b9092509050611a9d88611a81856001613b33565b60608701516080880151611a96906001613b33565b8686612386565b50506000806000611ac38560000151866020015187604001518c888a6101600151612657565b9050611ace816110a0565b600081815260208d9052604090206007015490935015611af15760019150611b00565b611afb8b82610e16565b600091505b50600080611b2386600001518b8789606001518a608001518b6101600151612657565b9050611b2e816110a0565b600081815260208e9052604090206007015490925015611b905760405162461bcd60e51b815260206004820152601a60248201527f53746f726520636f6e7461696e73207570706572206368696c6400000000000060448201526064016104a4565b611b9a8c82610e16565b5060008a815260208c905260409020611bb49084836126e5565b80838b7f7340510d24b7ec9b5c100f5500d93429d80d00d46f0d18e4e85d0c4cc22b992485604051611bea911515815260200190565b60405180910390a4919a91995090975050505050505050565b600085815260208790526040902060070154611c315760405162461bcd60e51b81526004016104a490613a5a565b60008086815260208890526040902060090154600160a01b900460ff166001811115611c5f57611c5f6138b2565b14611c7c5760405162461bcd60e51b81526004016104a490613a87565b6002600086815260208890526040902060090154600160a81b900460ff166002811115611cab57611cab6138b2565b14611cf35760405162461bcd60e51b8152602060048201526018602482015277045646765206973206e6f74206120736d616c6c20737465760441b60448201526064016104a4565b6000858152602087905260409020611d0a90611f34565b600114611d595760405162461bcd60e51b815260206004820152601e60248201527f4564676520646f6573206e6f7420686176652073696e676c652073746570000060448201526064016104a4565b611d8586600001600087815260200190815260200160002060010154846040015185602001518561274c565b611dd15760405162461bcd60e51b815260206004820152601b60248201527f4265666f7265207374617465206e6f7420696e20686973746f7279000000000060448201526064016104a4565b8251602084015160408086015160608701519151635d3adcfb60e01b81526000946001600160a01b038a1694635d3adcfb94611e14949293919291600401613c63565b602060405180830381865afa158015611e31573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611e559190613990565b6000878152602089815260409091206003015490860151919250611e86918390611e80906001613b33565b8561274c565b611ed25760405162461bcd60e51b815260206004820152601a60248201527f4166746572207374617465206e6f7420696e20686973746f727900000000000060448201526064016104a4565b6000868152602088905260409020611ee9906120a1565b6000868152602088905260409020611f0090610c0c565b60405187907fe11db4b27bc8c6ea5943ecbb205ae1ca8d56c42c719717aaf8a53d43d0cee7c290600090a350505050505050565b60008082600201548360040154611f4b9190613b0a565b9050600081116106be5760405162461bcd60e51b81526004016104a490613a5a565b600081815260208490526040808220548483529120611f8b90610c0c565b14611fd85760405162461bcd60e51b815260206004820152601c60248201527f4f726967696e2069642d6d757475616c206964206d69736d617463680000000060448201526064016104a4565b600081815260208490526040902060090154600160a81b900460ff166002811115612005576120056138b2565b60008381526020859052604090206009015461202a90600160a81b900460ff1661278e565b600281111561203b5761203b6138b2565b1461209c5760405162461bcd60e51b815260206004820152602b60248201527f45646765207479706520646f6573206e6f74206d6174636820636c61696d696e60448201526a672065646765207479706560a81b60648201526084016104a4565b505050565b60006009820154600160a01b900460ff1660018111156120c3576120c36138b2565b1461211c5760405162461bcd60e51b815260206004820152602360248201527f4f6e6c792050656e64696e672065646765732063616e20626520436f6e6669726044820152621b595960ea1b60648201526084016104a4565b600901805460ff60a01b1916600160a01b179055565b600060405160200161214390613ab1565b604051602081830303815290604052805190602001208214159050919050565b60008590036121a65760405162461bcd60e51b815260206004820152600f60248201526e115b5c1d1e481bdc9a59da5b881a59608a1b60448201526064016104a4565b60006121b28483613b0a565b116121f15760405162461bcd60e51b815260206004820152600f60248201526e496e76616c6964206865696768747360881b60448201526064016104a4565b600084900361223d5760405162461bcd60e51b8152602060048201526018602482015277115b5c1d1e481cdd185c9d081a1a5cdd1bdc9e481c9bdbdd60421b60448201526064016104a4565b60008290036122875760405162461bcd60e51b8152602060048201526016602482015275115b5c1d1e48195b99081a1a5cdd1bdc9e481c9bdbdd60521b60448201526064016104a4565b5050505050565b60006106be8260090160159054906101000a900460ff168360000154846002015485600101548660040154876003015461095f565b600060026122d18484613b0a565b10156123295760405162461bcd60e51b815260206004820152602160248201527f48656967687420646966666572656e6365206e6f742074776f206f72206d6f726044820152606560f81b60648201526084016104a4565b6123338383613b0a565b60020361234c57612345836001613b33565b90506106be565b60008361235a600185613b0a565b189050600061236882612876565b9050600019811b8061237b600187613b0a565b169695505050505050565b600085116123cd5760405162461bcd60e51b815260206004820152601460248201527305072652d73697a652063616e6e6f7420626520360641b60448201526064016104a4565b856123d783612975565b146124245760405162461bcd60e51b815260206004820152601b60248201527f50726520657870616e73696f6e20726f6f74206d69736d61746368000000000060448201526064016104a4565b8461242e83612ade565b146124855760405162461bcd60e51b815260206004820152602160248201527f5072652073697a6520646f6573206e6f74206d6174636820657870616e73696f6044820152603760f91b60648201526084016104a4565b8285106124d45760405162461bcd60e51b815260206004820181905260248201527f5072652073697a65206e6f74206c657373207468616e20706f73742073697a6560448201526064016104a4565b60008590506000806124e98560008751612b39565b90505b858310156125a15760006125008488612c6b565b9050845183106125475760405162461bcd60e51b8152602060048201526012602482015271496e646578206f7574206f662072616e676560701b60448201526064016104a4565b61256b828287868151811061255e5761255e613b1d565b6020026020010151612d25565b91506001811b61257b8186613b33565b94508785111561258d5761258d613cd9565b8361259781613b46565b94505050506124ec565b866125ab82612975565b146126035760405162461bcd60e51b815260206004820152602260248201527f506f737420657870616e73696f6e20726f6f74206e6f7420657175616c20706f6044820152611cdd60f21b60648201526084016104a4565b8351821461264c5760405162461bcd60e51b8152602060048201526016602482015275496e636f6d706c6574652070726f6f6620757361676560501b60448201526064016104a4565b505050505050505050565b61265f613310565b61266c8787878787612163565b6040805161018081018252888152602081018890529081018690526060810185905260808101849052600060a0820181905260c082018190524260e08301526101008201819052610120820181905261014082015261016081018360028111156126d8576126d86138b2565b9052979650505050505050565b60058301541580156126f957506006830154155b61273c5760405162461bcd60e51b815260206004820152601460248201527310da1a5b191c995b88185b1c9958591e481cd95d60621b60448201526064016104a4565b6005830191909155600690910155565b60008061278283858760405160200161276791815260200190565b60405160208183030381529060405280519060200120613231565b90951495945050505050565b6000808260028111156127a3576127a36138b2565b036127b057506001919050565b60018260028111156127c4576127c46138b2565b036127d157506002919050565b60028260028111156127e5576127e56138b2565b036128325760405162461bcd60e51b815260206004820152601c60248201527f4e6f206e657874207479706520616674657220536d616c6c537465700000000060448201526064016104a4565b60405162461bcd60e51b8152602060048201526014602482015273556e65787065637465642065646765207479706560601b60448201526064016104a4565b919050565b6000816000036128985760405162461bcd60e51b81526004016104a490613cef565b600160801b82106128b657608091821c916128b39082613b33565b90505b600160401b82106128d457604091821c916128d19082613b33565b90505b64010000000082106128f357602091821c916128f09082613b33565b90505b62010000821061291057601091821c9161290d9082613b33565b90505b610100821061292c57600891821c916129299082613b33565b90505b6010821061294757600491821c916129449082613b33565b90505b6004821061296257600291821c9161295f9082613b33565b90505b60028210612871576106be600182613b33565b6000808251116129c05760405162461bcd60e51b815260206004820152601660248201527522b6b83a3c9036b2b935b6329032bc3830b739b4b7b760511b60448201526064016104a4565b6040825111156129e25760405162461bcd60e51b81526004016104a490613d26565b6000805b8351811015612ad7576000848281518110612a0357612a03613b1d565b60200260200101519050826000801b03612a6f578015612a6a5780925060018551612a2e9190613b0a565b8214612a6a57604051612a51908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b612ac4565b8015612a8e576040805160208101839052908101849052606001612a51565b604051612aab908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b5080612acf81613b46565b9150506129e6565b5092915050565b600080805b8351811015612ad757838181518110612afe57612afe613b1d565b60200260200101516000801b14612b2757612b1a816002613e41565b612b249083613b33565b91505b80612b3181613b46565b915050612ae3565b6060818310612b5a5760405162461bcd60e51b81526004016104a490613e4d565b8351821115612bb55760405162461bcd60e51b815260206004820152602160248201527f456e64206e6f74206c657373206f7220657175616c207468616e206c656e67746044820152600d60fb1b60648201526084016104a4565b6000612bc18484613b0a565b6001600160401b03811115612bd857612bd86133ee565b604051908082528060200260200182016040528015612c01578160200160208202803683370190505b509050835b83811015612c6257858181518110612c2057612c20613b1d565b6020026020010151828683612c359190613b0a565b81518110612c4557612c45613b1d565b602090810291909101015280612c5a81613b46565b915050612c06565b50949350505050565b6000818310612c8c5760405162461bcd60e51b81526004016104a490613e4d565b6000612c99838518612876565b905060006001612ca98382613b33565b6001901b612cb79190613b0a565b90508481168482168115612cce576114da826132d3565b8015612cdd576114da81612876565b60405162461bcd60e51b815260206004820152601b60248201527f426f7468207920616e64207a2063616e6e6f74206265207a65726f000000000060448201526064016104a4565b606060408310612d685760405162461bcd60e51b815260206004820152600e60248201526d098caeccad840e8dede40d0d2ced60931b60448201526064016104a4565b6000829003612db95760405162461bcd60e51b815260206004820152601b60248201527f43616e6e6f7420617070656e6420656d7074792073756274726565000000000060448201526064016104a4565b604084511115612ddb5760405162461bcd60e51b81526004016104a490613d26565b8351600003612e59576000612df1846001613b33565b6001600160401b03811115612e0857612e086133ee565b604051908082528060200260200182016040528015612e31578160200160208202803683370190505b50905082818581518110612e4757612e47613b1d565b60209081029190910101529050611528565b83518310612ec75760405162461bcd60e51b815260206004820152603560248201527f4c6576656c2067726561746572207468616e2068696768657374206c6576656c6044820152741037b31031bab93932b73a1032bc3830b739b4b7b760591b60648201526084016104a4565b816000612ed386612ade565b90506000612ee2866002613e41565b612eec9083613b33565b90506000612ef983612876565b612f0283612876565b11612f4f5787516001600160401b03811115612f2057612f206133ee565b604051908082528060200260200182016040528015612f49578160200160208202803683370190505b50612f9e565b8751612f5c906001613b33565b6001600160401b03811115612f7357612f736133ee565b604051908082528060200260200182016040528015612f9c578160200160208202803683370190505b505b9050604081511115612ff25760405162461bcd60e51b815260206004820152601c60248201527f417070656e642063726561746573206f76657273697a6520747265650000000060448201526064016104a4565b60005b885181101561319357878110156130815788818151811061301857613018613b1d565b60200260200101516000801b1461307c5760405162461bcd60e51b815260206004820152602260248201527f417070656e642061626f7665206c65617374207369676e69666963616e7420626044820152611a5d60f21b60648201526084016104a4565b613181565b60008590036130c75788818151811061309c5761309c613b1d565b60200260200101518282815181106130b6576130b6613b1d565b602002602001018181525050613181565b8881815181106130d9576130d9613b1d565b60200260200101516000801b0361311157848282815181106130fd576130fd613b1d565b602090810291909101015260009450613181565b6000801b82828151811061312757613127613b1d565b60200260200101818152505088818151811061314557613145613b1d565b602002602001015185604051602001613168929190918252602082015260400190565b6040516020818303038152906040528051906020012094505b8061318b81613b46565b915050612ff5565b5083156131c7578381600183516131aa9190613b0a565b815181106131ba576131ba613b1d565b6020026020010181815250505b80600182516131d69190613b0a565b815181106131e6576131e6613b1d565b60200260200101516000801b0361034f5760405162461bcd60e51b815260206004820152600f60248201526e4c61737420656e747279207a65726f60881b60448201526064016104a4565b825160009061010081111561326457604051637ed6198f60e11b81526004810182905261010060248201526044016104a4565b8260005b828110156132c957600087828151811061328457613284613b1d565b60200260200101519050816001901b87166000036132b0578260005280602052604060002092506132c0565b8060005282602052604060002092505b50600101613268565b5095945050505050565b60008082116132f45760405162461bcd60e51b81526004016104a490613cef565b60008280613303600182613b0a565b1618905061152881612876565b6040805161018081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290529061014082019081526020016000905290565b80356003811061287157600080fd5b60008060008060008060c0878903121561339b57600080fd5b6133a487613373565b9860208801359850604088013597606081013597506080810135965060a00135945092505050565b600080604083850312156133df57600080fd5b50508035926020909101359150565b634e487b7160e01b600052604160045260246000fd5b60405160c081016001600160401b0381118282101715613426576134266133ee565b60405290565b604080519081016001600160401b0381118282101715613426576134266133ee565b604051601f8201601f191681016001600160401b0381118282101715613476576134766133ee565b604052919050565b60008083601f84011261349057600080fd5b5081356001600160401b038111156134a757600080fd5b6020830191508360208285010111156134bf57600080fd5b9250929050565b60008060008060008587036101008112156134e057600080fd5b60c08112156134ee57600080fd5b506134f7613404565b61350087613373565b81526020870135602082015260408701356040820152606087013560608201526080870135608082015260a087013560a08201528095505060c08601356001600160401b038082111561355257600080fd5b61355e89838a0161347e565b909650945060e088013591508082111561357757600080fd5b506135848882890161347e565b969995985093965092949392505050565b6000602082840312156135a757600080fd5b5035919050565b60006001600160401b038211156135c7576135c76133ee565b5060051b60200190565b600080604083850312156135e457600080fd5b823591506020808401356001600160401b0381111561360257600080fd5b8401601f8101861361361357600080fd5b8035613626613621826135ae565b61344e565b81815260059190911b8201830190838101908883111561364557600080fd5b928401925b828410156136635783358252928401929084019061364a565b80955050505050509250929050565b600080600080600060a0868803121561368a57600080fd5b61369386613373565b97602087013597506040870135966060810135965060800135945092505050565b6001600160a01b03811681146106af57600080fd5b6000806000606084860312156136de57600080fd5b83356136e9816136b4565b9250602084013591506040840135613700816136b4565b809150509250925092565b600082601f83011261371c57600080fd5b81356001600160401b03811115613735576137356133ee565b613748601f8201601f191660200161344e565b81815284602083860101111561375d57600080fd5b816020850160208301376000918101602001919091529392505050565b60008060006060848603121561378f57600080fd5b833592506020840135915060408401356001600160401b038111156137b357600080fd5b6137bf8682870161370b565b9150509250925092565b60008083601f8401126137db57600080fd5b5081356001600160401b038111156137f257600080fd5b6020830191508360208260051b85010111156134bf57600080fd5b6000806000806000806080878903121561382657600080fd5b8635955060208701356001600160401b038082111561384457600080fd5b9088019060a0828b03121561385857600080fd5b9095506040880135908082111561386e57600080fd5b61387a8a838b016137c9565b9096509450606089013591508082111561389357600080fd5b506138a089828a016137c9565b979a9699509497509295939492505050565b634e487b7160e01b600052602160045260246000fd5b600281106138d8576138d86138b2565b9052565b600381106138d8576138d86138b2565b600061018082019050825182526020830151602083015260408301516040830152606083015160608301526080830151608083015260a083015160a083015260c083015160c083015260e083015160e083015261010080840151818401525061012080840151613966828501826001600160a01b03169052565b50506101408084015161397b828501826138c8565b5050610160808401516114f2828501826138dc565b6000602082840312156139a257600080fd5b5051919050565b600081360360a08112156139bc57600080fd5b604051608081016001600160401b0382821081831117156139df576139df6133ee565b8160405260408412156139f157600080fd5b6139f961342c565b93508535845260208601359150613a0f826136b4565b81602085015283835260408601356020840152606086013560408401526080860135935080841115613a4057600080fd5b5050613a4e3683860161370b565b60608201529392505050565b602080825260139082015272115919d948191bd95cc81b9bdd08195e1a5cdd606a1b604082015260600190565b60208082526010908201526f45646765206e6f742070656e64696e6760801b604082015260600190565b6815539492559053115160ba1b815260090190565b84151581526020810184905260808101613ae360408301856138dc565b821515606083015295945050505050565b634e487b7160e01b600052601160045260246000fd5b818103818111156106be576106be613af4565b634e487b7160e01b600052603260045260246000fd5b808201808211156106be576106be613af4565b600060018201613b5857613b58613af4565b5060010190565b600060038710613b7157613b716138b2565b5060f89590951b8552600185019390935260218401919091526041830152606182015260810190565b600082601f830112613bab57600080fd5b81516020613bbb613621836135ae565b82815260059290921b84018101918181019086841115613bda57600080fd5b8286015b84811015613bf55780518352918301918301613bde565b509695505050505050565b60008060408385031215613c1357600080fd5b82516001600160401b0380821115613c2a57600080fd5b613c3686838701613b9a565b93506020850151915080821115613c4c57600080fd5b50613c5985828601613b9a565b9150509250929050565b845181526000602060018060a01b0381880151168184015285604084015284606084015260a0608084015283518060a085015260005b81811015613cb55785810183015185820160c001528201613c99565b50600060c0828601015260c0601f19601f8301168501019250505095945050505050565b634e487b7160e01b600052600160045260246000fd5b6020808252601c908201527f5a65726f20686173206e6f207369676e69666963616e74206269747300000000604082015260600190565b6020808252601a908201527f4d65726b6c6520657870616e73696f6e20746f6f206c61726765000000000000604082015260600190565b600181815b80851115613d98578160001904821115613d7e57613d7e613af4565b80851615613d8b57918102915b93841c9390800290613d62565b509250929050565b600082613daf575060016106be565b81613dbc575060006106be565b8160018114613dd25760028114613ddc57613df8565b60019150506106be565b60ff841115613ded57613ded613af4565b50506001821b6106be565b5060208310610133831016604e8410600b8410161715613e1b575081810a6106be565b613e258383613d5d565b8060001904821115613e3957613e39613af4565b029392505050565b60006115288383613da0565b60208082526017908201527614dd185c9d081b9bdd081b195cdcc81d1a185b88195b99604a1b60408201526060019056fea264697066735822122024b75d44bde653da91790769ed3fb58dfe0b744a2196dbd806b0ae07c43a762564736f6c63430008110033",
}

// EdgeChallengeManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use EdgeChallengeManagerMetaData.ABI instead.
var EdgeChallengeManagerABI = EdgeChallengeManagerMetaData.ABI

// EdgeChallengeManagerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use EdgeChallengeManagerMetaData.Bin instead.
var EdgeChallengeManagerBin = EdgeChallengeManagerMetaData.Bin

// DeployEdgeChallengeManager deploys a new Ethereum contract, binding an instance of EdgeChallengeManager to it.
func DeployEdgeChallengeManager(auth *bind.TransactOpts, backend bind.ContractBackend, _assertionChain common.Address, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (common.Address, *types.Transaction, *EdgeChallengeManager, error) {
	parsed, err := EdgeChallengeManagerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(EdgeChallengeManagerBin), backend, _assertionChain, _challengePeriodSec, _oneStepProofEntry)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &EdgeChallengeManager{EdgeChallengeManagerCaller: EdgeChallengeManagerCaller{contract: contract}, EdgeChallengeManagerTransactor: EdgeChallengeManagerTransactor{contract: contract}, EdgeChallengeManagerFilterer: EdgeChallengeManagerFilterer{contract: contract}}, nil
}

// EdgeChallengeManager is an auto generated Go binding around an Ethereum contract.
type EdgeChallengeManager struct {
	EdgeChallengeManagerCaller     // Read-only binding to the contract
	EdgeChallengeManagerTransactor // Write-only binding to the contract
	EdgeChallengeManagerFilterer   // Log filterer for contract events
}

// EdgeChallengeManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type EdgeChallengeManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeChallengeManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type EdgeChallengeManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeChallengeManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type EdgeChallengeManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EdgeChallengeManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type EdgeChallengeManagerSession struct {
	Contract     *EdgeChallengeManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// EdgeChallengeManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type EdgeChallengeManagerCallerSession struct {
	Contract *EdgeChallengeManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// EdgeChallengeManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type EdgeChallengeManagerTransactorSession struct {
	Contract     *EdgeChallengeManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// EdgeChallengeManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type EdgeChallengeManagerRaw struct {
	Contract *EdgeChallengeManager // Generic contract binding to access the raw methods on
}

// EdgeChallengeManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type EdgeChallengeManagerCallerRaw struct {
	Contract *EdgeChallengeManagerCaller // Generic read-only contract binding to access the raw methods on
}

// EdgeChallengeManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type EdgeChallengeManagerTransactorRaw struct {
	Contract *EdgeChallengeManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewEdgeChallengeManager creates a new instance of EdgeChallengeManager, bound to a specific deployed contract.
func NewEdgeChallengeManager(address common.Address, backend bind.ContractBackend) (*EdgeChallengeManager, error) {
	contract, err := bindEdgeChallengeManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManager{EdgeChallengeManagerCaller: EdgeChallengeManagerCaller{contract: contract}, EdgeChallengeManagerTransactor: EdgeChallengeManagerTransactor{contract: contract}, EdgeChallengeManagerFilterer: EdgeChallengeManagerFilterer{contract: contract}}, nil
}

// NewEdgeChallengeManagerCaller creates a new read-only instance of EdgeChallengeManager, bound to a specific deployed contract.
func NewEdgeChallengeManagerCaller(address common.Address, caller bind.ContractCaller) (*EdgeChallengeManagerCaller, error) {
	contract, err := bindEdgeChallengeManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerCaller{contract: contract}, nil
}

// NewEdgeChallengeManagerTransactor creates a new write-only instance of EdgeChallengeManager, bound to a specific deployed contract.
func NewEdgeChallengeManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*EdgeChallengeManagerTransactor, error) {
	contract, err := bindEdgeChallengeManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerTransactor{contract: contract}, nil
}

// NewEdgeChallengeManagerFilterer creates a new log filterer instance of EdgeChallengeManager, bound to a specific deployed contract.
func NewEdgeChallengeManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*EdgeChallengeManagerFilterer, error) {
	contract, err := bindEdgeChallengeManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerFilterer{contract: contract}, nil
}

// bindEdgeChallengeManager binds a generic wrapper to an already deployed contract.
func bindEdgeChallengeManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(EdgeChallengeManagerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EdgeChallengeManager *EdgeChallengeManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EdgeChallengeManager.Contract.EdgeChallengeManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EdgeChallengeManager *EdgeChallengeManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.EdgeChallengeManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EdgeChallengeManager *EdgeChallengeManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.EdgeChallengeManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EdgeChallengeManager *EdgeChallengeManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EdgeChallengeManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.contract.Transact(opts, method, params...)
}

// CalculateEdgeId is a free data retrieval call binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) CalculateEdgeId(opts *bind.CallOpts, edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "calculateEdgeId", edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateEdgeId is a free data retrieval call binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) CalculateEdgeId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.CalculateEdgeId(&_EdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
}

// CalculateEdgeId is a free data retrieval call binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) CalculateEdgeId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.CalculateEdgeId(&_EdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
}

// CalculateMutualId is a free data retrieval call binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) CalculateMutualId(opts *bind.CallOpts, edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "calculateMutualId", edgeType, originId, startHeight, startHistoryRoot, endHeight)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CalculateMutualId is a free data retrieval call binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) CalculateMutualId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.CalculateMutualId(&_EdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight)
}

// CalculateMutualId is a free data retrieval call binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) pure returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) CalculateMutualId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.CalculateMutualId(&_EdgeChallengeManager.CallOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight)
}

// ChallengePeriodSec is a free data retrieval call binding the contract method 0x654f0dc2.
//
// Solidity: function challengePeriodSec() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) ChallengePeriodSec(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "challengePeriodSec")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChallengePeriodSec is a free data retrieval call binding the contract method 0x654f0dc2.
//
// Solidity: function challengePeriodSec() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ChallengePeriodSec() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.ChallengePeriodSec(&_EdgeChallengeManager.CallOpts)
}

// ChallengePeriodSec is a free data retrieval call binding the contract method 0x654f0dc2.
//
// Solidity: function challengePeriodSec() view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) ChallengePeriodSec() (*big.Int, error) {
	return _EdgeChallengeManager.Contract.ChallengePeriodSec(&_EdgeChallengeManager.CallOpts)
}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) EdgeExists(opts *bind.CallOpts, edgeId [32]byte) (bool, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "edgeExists", edgeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) EdgeExists(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.EdgeExists(&_EdgeChallengeManager.CallOpts, edgeId)
}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) EdgeExists(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.EdgeExists(&_EdgeChallengeManager.CallOpts, edgeId)
}

// EdgeLength is a free data retrieval call binding the contract method 0xeae0328b.
//
// Solidity: function edgeLength(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) EdgeLength(opts *bind.CallOpts, edgeId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "edgeLength", edgeId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EdgeLength is a free data retrieval call binding the contract method 0xeae0328b.
//
// Solidity: function edgeLength(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) EdgeLength(edgeId [32]byte) (*big.Int, error) {
	return _EdgeChallengeManager.Contract.EdgeLength(&_EdgeChallengeManager.CallOpts, edgeId)
}

// EdgeLength is a free data retrieval call binding the contract method 0xeae0328b.
//
// Solidity: function edgeLength(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) EdgeLength(edgeId [32]byte) (*big.Int, error) {
	return _EdgeChallengeManager.Contract.EdgeLength(&_EdgeChallengeManager.CallOpts, edgeId)
}

// FirstRival is a free data retrieval call binding the contract method 0xbce6f54f.
//
// Solidity: function firstRival(bytes32 edgeId) view returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) FirstRival(opts *bind.CallOpts, edgeId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "firstRival", edgeId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// FirstRival is a free data retrieval call binding the contract method 0xbce6f54f.
//
// Solidity: function firstRival(bytes32 edgeId) view returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) FirstRival(edgeId [32]byte) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.FirstRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// FirstRival is a free data retrieval call binding the contract method 0xbce6f54f.
//
// Solidity: function firstRival(bytes32 edgeId) view returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) FirstRival(edgeId [32]byte) ([32]byte, error) {
	return _EdgeChallengeManager.Contract.FirstRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 edgeId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8))
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) GetEdge(opts *bind.CallOpts, edgeId [32]byte) (ChallengeEdge, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "getEdge", edgeId)

	if err != nil {
		return *new(ChallengeEdge), err
	}

	out0 := *abi.ConvertType(out[0], new(ChallengeEdge)).(*ChallengeEdge)

	return out0, err

}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 edgeId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8))
func (_EdgeChallengeManager *EdgeChallengeManagerSession) GetEdge(edgeId [32]byte) (ChallengeEdge, error) {
	return _EdgeChallengeManager.Contract.GetEdge(&_EdgeChallengeManager.CallOpts, edgeId)
}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 edgeId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8))
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) GetEdge(edgeId [32]byte) (ChallengeEdge, error) {
	return _EdgeChallengeManager.Contract.GetEdge(&_EdgeChallengeManager.CallOpts, edgeId)
}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) HasLengthOneRival(opts *bind.CallOpts, edgeId [32]byte) (bool, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "hasLengthOneRival", edgeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) HasLengthOneRival(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.HasLengthOneRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) HasLengthOneRival(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.HasLengthOneRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) HasRival(opts *bind.CallOpts, edgeId [32]byte) (bool, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "hasRival", edgeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) HasRival(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.HasRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 edgeId) view returns(bool)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) HasRival(edgeId [32]byte) (bool, error) {
	return _EdgeChallengeManager.Contract.HasRival(&_EdgeChallengeManager.CallOpts, edgeId)
}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCaller) TimeUnrivaled(opts *bind.CallOpts, edgeId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _EdgeChallengeManager.contract.Call(opts, &out, "timeUnrivaled", edgeId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) TimeUnrivaled(edgeId [32]byte) (*big.Int, error) {
	return _EdgeChallengeManager.Contract.TimeUnrivaled(&_EdgeChallengeManager.CallOpts, edgeId)
}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 edgeId) view returns(uint256)
func (_EdgeChallengeManager *EdgeChallengeManagerCallerSession) TimeUnrivaled(edgeId [32]byte) (*big.Int, error) {
	return _EdgeChallengeManager.Contract.TimeUnrivaled(&_EdgeChallengeManager.CallOpts, edgeId)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) BisectEdge(opts *bind.TransactOpts, edgeId [32]byte, bisectionHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "bisectEdge", edgeId, bisectionHistoryRoot, prefixProof)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) BisectEdge(edgeId [32]byte, bisectionHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.BisectEdge(&_EdgeChallengeManager.TransactOpts, edgeId, bisectionHistoryRoot, prefixProof)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 edgeId, bytes32 bisectionHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) BisectEdge(edgeId [32]byte, bisectionHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.BisectEdge(&_EdgeChallengeManager.TransactOpts, edgeId, bisectionHistoryRoot, prefixProof)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 edgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) ConfirmEdgeByChildren(opts *bind.TransactOpts, edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "confirmEdgeByChildren", edgeId)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 edgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ConfirmEdgeByChildren(edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByChildren(&_EdgeChallengeManager.TransactOpts, edgeId)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 edgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) ConfirmEdgeByChildren(edgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByChildren(&_EdgeChallengeManager.TransactOpts, edgeId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) ConfirmEdgeByClaim(opts *bind.TransactOpts, edgeId [32]byte, claimingEdgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "confirmEdgeByClaim", edgeId, claimingEdgeId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ConfirmEdgeByClaim(edgeId [32]byte, claimingEdgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByClaim(&_EdgeChallengeManager.TransactOpts, edgeId, claimingEdgeId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 edgeId, bytes32 claimingEdgeId) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) ConfirmEdgeByClaim(edgeId [32]byte, claimingEdgeId [32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByClaim(&_EdgeChallengeManager.TransactOpts, edgeId, claimingEdgeId)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0xe5b6c5b8.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, ((uint256,address),uint256,bytes32,bytes) oneStepData, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) ConfirmEdgeByOneStepProof(opts *bind.TransactOpts, edgeId [32]byte, oneStepData OneStepData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "confirmEdgeByOneStepProof", edgeId, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0xe5b6c5b8.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, ((uint256,address),uint256,bytes32,bytes) oneStepData, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ConfirmEdgeByOneStepProof(edgeId [32]byte, oneStepData OneStepData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByOneStepProof(&_EdgeChallengeManager.TransactOpts, edgeId, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0xe5b6c5b8.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, ((uint256,address),uint256,bytes32,bytes) oneStepData, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) ConfirmEdgeByOneStepProof(edgeId [32]byte, oneStepData OneStepData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByOneStepProof(&_EdgeChallengeManager.TransactOpts, edgeId, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x92fd750b.
//
// Solidity: function confirmEdgeByTime(bytes32 edgeId, bytes32[] ancestorEdges) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) ConfirmEdgeByTime(opts *bind.TransactOpts, edgeId [32]byte, ancestorEdges [][32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "confirmEdgeByTime", edgeId, ancestorEdges)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x92fd750b.
//
// Solidity: function confirmEdgeByTime(bytes32 edgeId, bytes32[] ancestorEdges) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) ConfirmEdgeByTime(edgeId [32]byte, ancestorEdges [][32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByTime(&_EdgeChallengeManager.TransactOpts, edgeId, ancestorEdges)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x92fd750b.
//
// Solidity: function confirmEdgeByTime(bytes32 edgeId, bytes32[] ancestorEdges) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) ConfirmEdgeByTime(edgeId [32]byte, ancestorEdges [][32]byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.ConfirmEdgeByTime(&_EdgeChallengeManager.TransactOpts, edgeId, ancestorEdges)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x16a0ca0b.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,uint256,bytes32) args, bytes , bytes ) payable returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) CreateLayerZeroEdge(opts *bind.TransactOpts, args CreateEdgeArgs, arg1 []byte, arg2 []byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "createLayerZeroEdge", args, arg1, arg2)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x16a0ca0b.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,uint256,bytes32) args, bytes , bytes ) payable returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerSession) CreateLayerZeroEdge(args CreateEdgeArgs, arg1 []byte, arg2 []byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.CreateLayerZeroEdge(&_EdgeChallengeManager.TransactOpts, args, arg1, arg2)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x16a0ca0b.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,uint256,bytes32) args, bytes , bytes ) payable returns(bytes32)
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) CreateLayerZeroEdge(args CreateEdgeArgs, arg1 []byte, arg2 []byte) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.CreateLayerZeroEdge(&_EdgeChallengeManager.TransactOpts, args, arg1, arg2)
}

// Initialize is a paid mutator transaction binding the contract method 0xc350a1b5.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodSec, address _oneStepProofEntry) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactor) Initialize(opts *bind.TransactOpts, _assertionChain common.Address, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _EdgeChallengeManager.contract.Transact(opts, "initialize", _assertionChain, _challengePeriodSec, _oneStepProofEntry)
}

// Initialize is a paid mutator transaction binding the contract method 0xc350a1b5.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodSec, address _oneStepProofEntry) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerSession) Initialize(_assertionChain common.Address, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.Initialize(&_EdgeChallengeManager.TransactOpts, _assertionChain, _challengePeriodSec, _oneStepProofEntry)
}

// Initialize is a paid mutator transaction binding the contract method 0xc350a1b5.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodSec, address _oneStepProofEntry) returns()
func (_EdgeChallengeManager *EdgeChallengeManagerTransactorSession) Initialize(_assertionChain common.Address, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _EdgeChallengeManager.Contract.Initialize(&_EdgeChallengeManager.TransactOpts, _assertionChain, _challengePeriodSec, _oneStepProofEntry)
}

// EdgeChallengeManagerBisectedIterator is returned from FilterBisected and is used to iterate over the raw logs and unpacked data for Bisected events raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerBisectedIterator struct {
	Event *EdgeChallengeManagerBisected // Event containing the contract specifics and raw log

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
func (it *EdgeChallengeManagerBisectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeChallengeManagerBisected)
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
		it.Event = new(EdgeChallengeManagerBisected)
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
func (it *EdgeChallengeManagerBisectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeChallengeManagerBisectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeChallengeManagerBisected represents a Bisected event raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerBisected struct {
	BisectedEdgeId [32]byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterBisected is a free log retrieval operation binding the contract event 0xaab36db1c086a1a8a2a953ec2a3f131e133f7be8e6e1970f8fd79a2ab341c001.
//
// Solidity: event Bisected(bytes32 bisectedEdgeId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) FilterBisected(opts *bind.FilterOpts) (*EdgeChallengeManagerBisectedIterator, error) {

	logs, sub, err := _EdgeChallengeManager.contract.FilterLogs(opts, "Bisected")
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerBisectedIterator{contract: _EdgeChallengeManager.contract, event: "Bisected", logs: logs, sub: sub}, nil
}

// WatchBisected is a free log subscription operation binding the contract event 0xaab36db1c086a1a8a2a953ec2a3f131e133f7be8e6e1970f8fd79a2ab341c001.
//
// Solidity: event Bisected(bytes32 bisectedEdgeId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) WatchBisected(opts *bind.WatchOpts, sink chan<- *EdgeChallengeManagerBisected) (event.Subscription, error) {

	logs, sub, err := _EdgeChallengeManager.contract.WatchLogs(opts, "Bisected")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeChallengeManagerBisected)
				if err := _EdgeChallengeManager.contract.UnpackLog(event, "Bisected", log); err != nil {
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

// ParseBisected is a log parse operation binding the contract event 0xaab36db1c086a1a8a2a953ec2a3f131e133f7be8e6e1970f8fd79a2ab341c001.
//
// Solidity: event Bisected(bytes32 bisectedEdgeId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) ParseBisected(log types.Log) (*EdgeChallengeManagerBisected, error) {
	event := new(EdgeChallengeManagerBisected)
	if err := _EdgeChallengeManager.contract.UnpackLog(event, "Bisected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EdgeChallengeManagerLevelZeroEdgeAddedIterator is returned from FilterLevelZeroEdgeAdded and is used to iterate over the raw logs and unpacked data for LevelZeroEdgeAdded events raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerLevelZeroEdgeAddedIterator struct {
	Event *EdgeChallengeManagerLevelZeroEdgeAdded // Event containing the contract specifics and raw log

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
func (it *EdgeChallengeManagerLevelZeroEdgeAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EdgeChallengeManagerLevelZeroEdgeAdded)
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
		it.Event = new(EdgeChallengeManagerLevelZeroEdgeAdded)
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
func (it *EdgeChallengeManagerLevelZeroEdgeAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EdgeChallengeManagerLevelZeroEdgeAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EdgeChallengeManagerLevelZeroEdgeAdded represents a LevelZeroEdgeAdded event raised by the EdgeChallengeManager contract.
type EdgeChallengeManagerLevelZeroEdgeAdded struct {
	EdgeId [32]byte
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterLevelZeroEdgeAdded is a free log retrieval operation binding the contract event 0x102ba5fcc71c9f7d7075d3f9cc9cb52fe4feb2cb843bef52f5f9fe9825b539e5.
//
// Solidity: event LevelZeroEdgeAdded(bytes32 edgeId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) FilterLevelZeroEdgeAdded(opts *bind.FilterOpts) (*EdgeChallengeManagerLevelZeroEdgeAddedIterator, error) {

	logs, sub, err := _EdgeChallengeManager.contract.FilterLogs(opts, "LevelZeroEdgeAdded")
	if err != nil {
		return nil, err
	}
	return &EdgeChallengeManagerLevelZeroEdgeAddedIterator{contract: _EdgeChallengeManager.contract, event: "LevelZeroEdgeAdded", logs: logs, sub: sub}, nil
}

// WatchLevelZeroEdgeAdded is a free log subscription operation binding the contract event 0x102ba5fcc71c9f7d7075d3f9cc9cb52fe4feb2cb843bef52f5f9fe9825b539e5.
//
// Solidity: event LevelZeroEdgeAdded(bytes32 edgeId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) WatchLevelZeroEdgeAdded(opts *bind.WatchOpts, sink chan<- *EdgeChallengeManagerLevelZeroEdgeAdded) (event.Subscription, error) {

	logs, sub, err := _EdgeChallengeManager.contract.WatchLogs(opts, "LevelZeroEdgeAdded")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EdgeChallengeManagerLevelZeroEdgeAdded)
				if err := _EdgeChallengeManager.contract.UnpackLog(event, "LevelZeroEdgeAdded", log); err != nil {
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

// ParseLevelZeroEdgeAdded is a log parse operation binding the contract event 0x102ba5fcc71c9f7d7075d3f9cc9cb52fe4feb2cb843bef52f5f9fe9825b539e5.
//
// Solidity: event LevelZeroEdgeAdded(bytes32 edgeId)
func (_EdgeChallengeManager *EdgeChallengeManagerFilterer) ParseLevelZeroEdgeAdded(log types.Log) (*EdgeChallengeManagerLevelZeroEdgeAdded, error) {
	event := new(EdgeChallengeManagerLevelZeroEdgeAdded)
	if err := _EdgeChallengeManager.contract.UnpackLog(event, "LevelZeroEdgeAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IAssertionChainMetaData contains all meta data concerning the IAssertionChain contract.
var IAssertionChainMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getFirstChildCreationTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getHeight\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getInboxMsgCountSeen\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getPredecessorId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getStateHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"getSuccessionChallenge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"isFirstChild\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IAssertionChainABI is the input ABI used to generate the binding from.
// Deprecated: Use IAssertionChainMetaData.ABI instead.
var IAssertionChainABI = IAssertionChainMetaData.ABI

// IAssertionChain is an auto generated Go binding around an Ethereum contract.
type IAssertionChain struct {
	IAssertionChainCaller     // Read-only binding to the contract
	IAssertionChainTransactor // Write-only binding to the contract
	IAssertionChainFilterer   // Log filterer for contract events
}

// IAssertionChainCaller is an auto generated read-only Go binding around an Ethereum contract.
type IAssertionChainCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAssertionChainTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IAssertionChainTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAssertionChainFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IAssertionChainFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IAssertionChainSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IAssertionChainSession struct {
	Contract     *IAssertionChain  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IAssertionChainCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IAssertionChainCallerSession struct {
	Contract *IAssertionChainCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// IAssertionChainTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IAssertionChainTransactorSession struct {
	Contract     *IAssertionChainTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// IAssertionChainRaw is an auto generated low-level Go binding around an Ethereum contract.
type IAssertionChainRaw struct {
	Contract *IAssertionChain // Generic contract binding to access the raw methods on
}

// IAssertionChainCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IAssertionChainCallerRaw struct {
	Contract *IAssertionChainCaller // Generic read-only contract binding to access the raw methods on
}

// IAssertionChainTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IAssertionChainTransactorRaw struct {
	Contract *IAssertionChainTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIAssertionChain creates a new instance of IAssertionChain, bound to a specific deployed contract.
func NewIAssertionChain(address common.Address, backend bind.ContractBackend) (*IAssertionChain, error) {
	contract, err := bindIAssertionChain(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IAssertionChain{IAssertionChainCaller: IAssertionChainCaller{contract: contract}, IAssertionChainTransactor: IAssertionChainTransactor{contract: contract}, IAssertionChainFilterer: IAssertionChainFilterer{contract: contract}}, nil
}

// NewIAssertionChainCaller creates a new read-only instance of IAssertionChain, bound to a specific deployed contract.
func NewIAssertionChainCaller(address common.Address, caller bind.ContractCaller) (*IAssertionChainCaller, error) {
	contract, err := bindIAssertionChain(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IAssertionChainCaller{contract: contract}, nil
}

// NewIAssertionChainTransactor creates a new write-only instance of IAssertionChain, bound to a specific deployed contract.
func NewIAssertionChainTransactor(address common.Address, transactor bind.ContractTransactor) (*IAssertionChainTransactor, error) {
	contract, err := bindIAssertionChain(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IAssertionChainTransactor{contract: contract}, nil
}

// NewIAssertionChainFilterer creates a new log filterer instance of IAssertionChain, bound to a specific deployed contract.
func NewIAssertionChainFilterer(address common.Address, filterer bind.ContractFilterer) (*IAssertionChainFilterer, error) {
	contract, err := bindIAssertionChain(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IAssertionChainFilterer{contract: contract}, nil
}

// bindIAssertionChain binds a generic wrapper to an already deployed contract.
func bindIAssertionChain(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(IAssertionChainABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IAssertionChain *IAssertionChainRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IAssertionChain.Contract.IAssertionChainCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IAssertionChain *IAssertionChainRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IAssertionChain.Contract.IAssertionChainTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IAssertionChain *IAssertionChainRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IAssertionChain.Contract.IAssertionChainTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IAssertionChain *IAssertionChainCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IAssertionChain.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IAssertionChain *IAssertionChainTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IAssertionChain.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IAssertionChain *IAssertionChainTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IAssertionChain.Contract.contract.Transact(opts, method, params...)
}

// GetFirstChildCreationTime is a free data retrieval call binding the contract method 0x43ed6ad9.
//
// Solidity: function getFirstChildCreationTime(bytes32 assertionId) view returns(uint256)
func (_IAssertionChain *IAssertionChainCaller) GetFirstChildCreationTime(opts *bind.CallOpts, assertionId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "getFirstChildCreationTime", assertionId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetFirstChildCreationTime is a free data retrieval call binding the contract method 0x43ed6ad9.
//
// Solidity: function getFirstChildCreationTime(bytes32 assertionId) view returns(uint256)
func (_IAssertionChain *IAssertionChainSession) GetFirstChildCreationTime(assertionId [32]byte) (*big.Int, error) {
	return _IAssertionChain.Contract.GetFirstChildCreationTime(&_IAssertionChain.CallOpts, assertionId)
}

// GetFirstChildCreationTime is a free data retrieval call binding the contract method 0x43ed6ad9.
//
// Solidity: function getFirstChildCreationTime(bytes32 assertionId) view returns(uint256)
func (_IAssertionChain *IAssertionChainCallerSession) GetFirstChildCreationTime(assertionId [32]byte) (*big.Int, error) {
	return _IAssertionChain.Contract.GetFirstChildCreationTime(&_IAssertionChain.CallOpts, assertionId)
}

// GetHeight is a free data retrieval call binding the contract method 0x896efbf2.
//
// Solidity: function getHeight(bytes32 assertionId) view returns(uint256)
func (_IAssertionChain *IAssertionChainCaller) GetHeight(opts *bind.CallOpts, assertionId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "getHeight", assertionId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetHeight is a free data retrieval call binding the contract method 0x896efbf2.
//
// Solidity: function getHeight(bytes32 assertionId) view returns(uint256)
func (_IAssertionChain *IAssertionChainSession) GetHeight(assertionId [32]byte) (*big.Int, error) {
	return _IAssertionChain.Contract.GetHeight(&_IAssertionChain.CallOpts, assertionId)
}

// GetHeight is a free data retrieval call binding the contract method 0x896efbf2.
//
// Solidity: function getHeight(bytes32 assertionId) view returns(uint256)
func (_IAssertionChain *IAssertionChainCallerSession) GetHeight(assertionId [32]byte) (*big.Int, error) {
	return _IAssertionChain.Contract.GetHeight(&_IAssertionChain.CallOpts, assertionId)
}

// GetInboxMsgCountSeen is a free data retrieval call binding the contract method 0x7cfd5ab9.
//
// Solidity: function getInboxMsgCountSeen(bytes32 assertionId) view returns(uint256)
func (_IAssertionChain *IAssertionChainCaller) GetInboxMsgCountSeen(opts *bind.CallOpts, assertionId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "getInboxMsgCountSeen", assertionId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetInboxMsgCountSeen is a free data retrieval call binding the contract method 0x7cfd5ab9.
//
// Solidity: function getInboxMsgCountSeen(bytes32 assertionId) view returns(uint256)
func (_IAssertionChain *IAssertionChainSession) GetInboxMsgCountSeen(assertionId [32]byte) (*big.Int, error) {
	return _IAssertionChain.Contract.GetInboxMsgCountSeen(&_IAssertionChain.CallOpts, assertionId)
}

// GetInboxMsgCountSeen is a free data retrieval call binding the contract method 0x7cfd5ab9.
//
// Solidity: function getInboxMsgCountSeen(bytes32 assertionId) view returns(uint256)
func (_IAssertionChain *IAssertionChainCallerSession) GetInboxMsgCountSeen(assertionId [32]byte) (*big.Int, error) {
	return _IAssertionChain.Contract.GetInboxMsgCountSeen(&_IAssertionChain.CallOpts, assertionId)
}

// GetPredecessorId is a free data retrieval call binding the contract method 0x9ca565d4.
//
// Solidity: function getPredecessorId(bytes32 assertionId) view returns(bytes32)
func (_IAssertionChain *IAssertionChainCaller) GetPredecessorId(opts *bind.CallOpts, assertionId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "getPredecessorId", assertionId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetPredecessorId is a free data retrieval call binding the contract method 0x9ca565d4.
//
// Solidity: function getPredecessorId(bytes32 assertionId) view returns(bytes32)
func (_IAssertionChain *IAssertionChainSession) GetPredecessorId(assertionId [32]byte) ([32]byte, error) {
	return _IAssertionChain.Contract.GetPredecessorId(&_IAssertionChain.CallOpts, assertionId)
}

// GetPredecessorId is a free data retrieval call binding the contract method 0x9ca565d4.
//
// Solidity: function getPredecessorId(bytes32 assertionId) view returns(bytes32)
func (_IAssertionChain *IAssertionChainCallerSession) GetPredecessorId(assertionId [32]byte) ([32]byte, error) {
	return _IAssertionChain.Contract.GetPredecessorId(&_IAssertionChain.CallOpts, assertionId)
}

// GetStateHash is a free data retrieval call binding the contract method 0xf9bce634.
//
// Solidity: function getStateHash(bytes32 assertionId) view returns(bytes32)
func (_IAssertionChain *IAssertionChainCaller) GetStateHash(opts *bind.CallOpts, assertionId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "getStateHash", assertionId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetStateHash is a free data retrieval call binding the contract method 0xf9bce634.
//
// Solidity: function getStateHash(bytes32 assertionId) view returns(bytes32)
func (_IAssertionChain *IAssertionChainSession) GetStateHash(assertionId [32]byte) ([32]byte, error) {
	return _IAssertionChain.Contract.GetStateHash(&_IAssertionChain.CallOpts, assertionId)
}

// GetStateHash is a free data retrieval call binding the contract method 0xf9bce634.
//
// Solidity: function getStateHash(bytes32 assertionId) view returns(bytes32)
func (_IAssertionChain *IAssertionChainCallerSession) GetStateHash(assertionId [32]byte) ([32]byte, error) {
	return _IAssertionChain.Contract.GetStateHash(&_IAssertionChain.CallOpts, assertionId)
}

// GetSuccessionChallenge is a free data retrieval call binding the contract method 0x2fefa18a.
//
// Solidity: function getSuccessionChallenge(bytes32 assertionId) view returns(bytes32)
func (_IAssertionChain *IAssertionChainCaller) GetSuccessionChallenge(opts *bind.CallOpts, assertionId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "getSuccessionChallenge", assertionId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetSuccessionChallenge is a free data retrieval call binding the contract method 0x2fefa18a.
//
// Solidity: function getSuccessionChallenge(bytes32 assertionId) view returns(bytes32)
func (_IAssertionChain *IAssertionChainSession) GetSuccessionChallenge(assertionId [32]byte) ([32]byte, error) {
	return _IAssertionChain.Contract.GetSuccessionChallenge(&_IAssertionChain.CallOpts, assertionId)
}

// GetSuccessionChallenge is a free data retrieval call binding the contract method 0x2fefa18a.
//
// Solidity: function getSuccessionChallenge(bytes32 assertionId) view returns(bytes32)
func (_IAssertionChain *IAssertionChainCallerSession) GetSuccessionChallenge(assertionId [32]byte) ([32]byte, error) {
	return _IAssertionChain.Contract.GetSuccessionChallenge(&_IAssertionChain.CallOpts, assertionId)
}

// IsFirstChild is a free data retrieval call binding the contract method 0x30836228.
//
// Solidity: function isFirstChild(bytes32 assertionId) view returns(bool)
func (_IAssertionChain *IAssertionChainCaller) IsFirstChild(opts *bind.CallOpts, assertionId [32]byte) (bool, error) {
	var out []interface{}
	err := _IAssertionChain.contract.Call(opts, &out, "isFirstChild", assertionId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsFirstChild is a free data retrieval call binding the contract method 0x30836228.
//
// Solidity: function isFirstChild(bytes32 assertionId) view returns(bool)
func (_IAssertionChain *IAssertionChainSession) IsFirstChild(assertionId [32]byte) (bool, error) {
	return _IAssertionChain.Contract.IsFirstChild(&_IAssertionChain.CallOpts, assertionId)
}

// IsFirstChild is a free data retrieval call binding the contract method 0x30836228.
//
// Solidity: function isFirstChild(bytes32 assertionId) view returns(bool)
func (_IAssertionChain *IAssertionChainCallerSession) IsFirstChild(assertionId [32]byte) (bool, error) {
	return _IAssertionChain.Contract.IsFirstChild(&_IAssertionChain.CallOpts, assertionId)
}

// IChallengeManagerMetaData contains all meta data concerning the IChallengeManager contract.
var IChallengeManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"historyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"firstState\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"firstStatehistoryProof\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32\",\"name\":\"lastState\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"lastStatehistoryProof\",\"type\":\"bytes32[]\"}],\"internalType\":\"structAddLeafArgs\",\"name\":\"leafData\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof1\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof2\",\"type\":\"bytes\"}],\"name\":\"addLeaf\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"prefixHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"bisect\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"challengeExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"childrenAreAtOneStepFork\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"confirmForPsTimer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"confirmForSucessionChallengeWin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"createChallenge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"createSubChallenge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"getChallenge\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"rootId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"winningClaim\",\"type\":\"bytes32\"},{\"internalType\":\"enumChallengeType\",\"name\":\"challengeType\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"challenger\",\"type\":\"address\"}],\"internalType\":\"structChallenge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"getCurrentPsTimer\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"getVertex\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"historyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"successionChallenge\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"predecessorId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"enumVertexStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"psId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"psLastUpdatedTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"flushedPsTimeSec\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"lowestHeightSuccessorId\",\"type\":\"bytes32\"}],\"internalType\":\"structChallengeVertex\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"hasConfirmedSibling\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_miniStakeValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriod\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"prefixHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"merge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"vertexExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"winningClaim\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IChallengeManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use IChallengeManagerMetaData.ABI instead.
var IChallengeManagerABI = IChallengeManagerMetaData.ABI

// IChallengeManager is an auto generated Go binding around an Ethereum contract.
type IChallengeManager struct {
	IChallengeManagerCaller     // Read-only binding to the contract
	IChallengeManagerTransactor // Write-only binding to the contract
	IChallengeManagerFilterer   // Log filterer for contract events
}

// IChallengeManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type IChallengeManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IChallengeManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IChallengeManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IChallengeManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IChallengeManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IChallengeManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IChallengeManagerSession struct {
	Contract     *IChallengeManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// IChallengeManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IChallengeManagerCallerSession struct {
	Contract *IChallengeManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// IChallengeManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IChallengeManagerTransactorSession struct {
	Contract     *IChallengeManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// IChallengeManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type IChallengeManagerRaw struct {
	Contract *IChallengeManager // Generic contract binding to access the raw methods on
}

// IChallengeManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IChallengeManagerCallerRaw struct {
	Contract *IChallengeManagerCaller // Generic read-only contract binding to access the raw methods on
}

// IChallengeManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IChallengeManagerTransactorRaw struct {
	Contract *IChallengeManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIChallengeManager creates a new instance of IChallengeManager, bound to a specific deployed contract.
func NewIChallengeManager(address common.Address, backend bind.ContractBackend) (*IChallengeManager, error) {
	contract, err := bindIChallengeManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IChallengeManager{IChallengeManagerCaller: IChallengeManagerCaller{contract: contract}, IChallengeManagerTransactor: IChallengeManagerTransactor{contract: contract}, IChallengeManagerFilterer: IChallengeManagerFilterer{contract: contract}}, nil
}

// NewIChallengeManagerCaller creates a new read-only instance of IChallengeManager, bound to a specific deployed contract.
func NewIChallengeManagerCaller(address common.Address, caller bind.ContractCaller) (*IChallengeManagerCaller, error) {
	contract, err := bindIChallengeManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerCaller{contract: contract}, nil
}

// NewIChallengeManagerTransactor creates a new write-only instance of IChallengeManager, bound to a specific deployed contract.
func NewIChallengeManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*IChallengeManagerTransactor, error) {
	contract, err := bindIChallengeManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerTransactor{contract: contract}, nil
}

// NewIChallengeManagerFilterer creates a new log filterer instance of IChallengeManager, bound to a specific deployed contract.
func NewIChallengeManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*IChallengeManagerFilterer, error) {
	contract, err := bindIChallengeManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerFilterer{contract: contract}, nil
}

// bindIChallengeManager binds a generic wrapper to an already deployed contract.
func bindIChallengeManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(IChallengeManagerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IChallengeManager *IChallengeManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IChallengeManager.Contract.IChallengeManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IChallengeManager *IChallengeManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IChallengeManager.Contract.IChallengeManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IChallengeManager *IChallengeManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IChallengeManager.Contract.IChallengeManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IChallengeManager *IChallengeManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IChallengeManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IChallengeManager *IChallengeManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IChallengeManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IChallengeManager *IChallengeManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IChallengeManager.Contract.contract.Transact(opts, method, params...)
}

// ChallengeExists is a free data retrieval call binding the contract method 0x1b7bbecb.
//
// Solidity: function challengeExists(bytes32 challengeId) view returns(bool)
func (_IChallengeManager *IChallengeManagerCaller) ChallengeExists(opts *bind.CallOpts, challengeId [32]byte) (bool, error) {
	var out []interface{}
	err := _IChallengeManager.contract.Call(opts, &out, "challengeExists", challengeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ChallengeExists is a free data retrieval call binding the contract method 0x1b7bbecb.
//
// Solidity: function challengeExists(bytes32 challengeId) view returns(bool)
func (_IChallengeManager *IChallengeManagerSession) ChallengeExists(challengeId [32]byte) (bool, error) {
	return _IChallengeManager.Contract.ChallengeExists(&_IChallengeManager.CallOpts, challengeId)
}

// ChallengeExists is a free data retrieval call binding the contract method 0x1b7bbecb.
//
// Solidity: function challengeExists(bytes32 challengeId) view returns(bool)
func (_IChallengeManager *IChallengeManagerCallerSession) ChallengeExists(challengeId [32]byte) (bool, error) {
	return _IChallengeManager.Contract.ChallengeExists(&_IChallengeManager.CallOpts, challengeId)
}

// ChildrenAreAtOneStepFork is a free data retrieval call binding the contract method 0x7a4d47dc.
//
// Solidity: function childrenAreAtOneStepFork(bytes32 vId) view returns(bool)
func (_IChallengeManager *IChallengeManagerCaller) ChildrenAreAtOneStepFork(opts *bind.CallOpts, vId [32]byte) (bool, error) {
	var out []interface{}
	err := _IChallengeManager.contract.Call(opts, &out, "childrenAreAtOneStepFork", vId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ChildrenAreAtOneStepFork is a free data retrieval call binding the contract method 0x7a4d47dc.
//
// Solidity: function childrenAreAtOneStepFork(bytes32 vId) view returns(bool)
func (_IChallengeManager *IChallengeManagerSession) ChildrenAreAtOneStepFork(vId [32]byte) (bool, error) {
	return _IChallengeManager.Contract.ChildrenAreAtOneStepFork(&_IChallengeManager.CallOpts, vId)
}

// ChildrenAreAtOneStepFork is a free data retrieval call binding the contract method 0x7a4d47dc.
//
// Solidity: function childrenAreAtOneStepFork(bytes32 vId) view returns(bool)
func (_IChallengeManager *IChallengeManagerCallerSession) ChildrenAreAtOneStepFork(vId [32]byte) (bool, error) {
	return _IChallengeManager.Contract.ChildrenAreAtOneStepFork(&_IChallengeManager.CallOpts, vId)
}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,bytes32,uint8,address))
func (_IChallengeManager *IChallengeManagerCaller) GetChallenge(opts *bind.CallOpts, challengeId [32]byte) (Challenge, error) {
	var out []interface{}
	err := _IChallengeManager.contract.Call(opts, &out, "getChallenge", challengeId)

	if err != nil {
		return *new(Challenge), err
	}

	out0 := *abi.ConvertType(out[0], new(Challenge)).(*Challenge)

	return out0, err

}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,bytes32,uint8,address))
func (_IChallengeManager *IChallengeManagerSession) GetChallenge(challengeId [32]byte) (Challenge, error) {
	return _IChallengeManager.Contract.GetChallenge(&_IChallengeManager.CallOpts, challengeId)
}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,bytes32,uint8,address))
func (_IChallengeManager *IChallengeManagerCallerSession) GetChallenge(challengeId [32]byte) (Challenge, error) {
	return _IChallengeManager.Contract.GetChallenge(&_IChallengeManager.CallOpts, challengeId)
}

// GetCurrentPsTimer is a free data retrieval call binding the contract method 0x8ac04349.
//
// Solidity: function getCurrentPsTimer(bytes32 vId) view returns(uint256)
func (_IChallengeManager *IChallengeManagerCaller) GetCurrentPsTimer(opts *bind.CallOpts, vId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IChallengeManager.contract.Call(opts, &out, "getCurrentPsTimer", vId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentPsTimer is a free data retrieval call binding the contract method 0x8ac04349.
//
// Solidity: function getCurrentPsTimer(bytes32 vId) view returns(uint256)
func (_IChallengeManager *IChallengeManagerSession) GetCurrentPsTimer(vId [32]byte) (*big.Int, error) {
	return _IChallengeManager.Contract.GetCurrentPsTimer(&_IChallengeManager.CallOpts, vId)
}

// GetCurrentPsTimer is a free data retrieval call binding the contract method 0x8ac04349.
//
// Solidity: function getCurrentPsTimer(bytes32 vId) view returns(uint256)
func (_IChallengeManager *IChallengeManagerCallerSession) GetCurrentPsTimer(vId [32]byte) (*big.Int, error) {
	return _IChallengeManager.Contract.GetCurrentPsTimer(&_IChallengeManager.CallOpts, vId)
}

// GetVertex is a free data retrieval call binding the contract method 0x86f048ed.
//
// Solidity: function getVertex(bytes32 vId) view returns((bytes32,bytes32,uint256,bytes32,bytes32,bytes32,address,uint8,bytes32,uint256,uint256,bytes32))
func (_IChallengeManager *IChallengeManagerCaller) GetVertex(opts *bind.CallOpts, vId [32]byte) (ChallengeVertex, error) {
	var out []interface{}
	err := _IChallengeManager.contract.Call(opts, &out, "getVertex", vId)

	if err != nil {
		return *new(ChallengeVertex), err
	}

	out0 := *abi.ConvertType(out[0], new(ChallengeVertex)).(*ChallengeVertex)

	return out0, err

}

// GetVertex is a free data retrieval call binding the contract method 0x86f048ed.
//
// Solidity: function getVertex(bytes32 vId) view returns((bytes32,bytes32,uint256,bytes32,bytes32,bytes32,address,uint8,bytes32,uint256,uint256,bytes32))
func (_IChallengeManager *IChallengeManagerSession) GetVertex(vId [32]byte) (ChallengeVertex, error) {
	return _IChallengeManager.Contract.GetVertex(&_IChallengeManager.CallOpts, vId)
}

// GetVertex is a free data retrieval call binding the contract method 0x86f048ed.
//
// Solidity: function getVertex(bytes32 vId) view returns((bytes32,bytes32,uint256,bytes32,bytes32,bytes32,address,uint8,bytes32,uint256,uint256,bytes32))
func (_IChallengeManager *IChallengeManagerCallerSession) GetVertex(vId [32]byte) (ChallengeVertex, error) {
	return _IChallengeManager.Contract.GetVertex(&_IChallengeManager.CallOpts, vId)
}

// HasConfirmedSibling is a free data retrieval call binding the contract method 0x98b67d59.
//
// Solidity: function hasConfirmedSibling(bytes32 vId) view returns(bool)
func (_IChallengeManager *IChallengeManagerCaller) HasConfirmedSibling(opts *bind.CallOpts, vId [32]byte) (bool, error) {
	var out []interface{}
	err := _IChallengeManager.contract.Call(opts, &out, "hasConfirmedSibling", vId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasConfirmedSibling is a free data retrieval call binding the contract method 0x98b67d59.
//
// Solidity: function hasConfirmedSibling(bytes32 vId) view returns(bool)
func (_IChallengeManager *IChallengeManagerSession) HasConfirmedSibling(vId [32]byte) (bool, error) {
	return _IChallengeManager.Contract.HasConfirmedSibling(&_IChallengeManager.CallOpts, vId)
}

// HasConfirmedSibling is a free data retrieval call binding the contract method 0x98b67d59.
//
// Solidity: function hasConfirmedSibling(bytes32 vId) view returns(bool)
func (_IChallengeManager *IChallengeManagerCallerSession) HasConfirmedSibling(vId [32]byte) (bool, error) {
	return _IChallengeManager.Contract.HasConfirmedSibling(&_IChallengeManager.CallOpts, vId)
}

// VertexExists is a free data retrieval call binding the contract method 0x6b0b2592.
//
// Solidity: function vertexExists(bytes32 vId) view returns(bool)
func (_IChallengeManager *IChallengeManagerCaller) VertexExists(opts *bind.CallOpts, vId [32]byte) (bool, error) {
	var out []interface{}
	err := _IChallengeManager.contract.Call(opts, &out, "vertexExists", vId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// VertexExists is a free data retrieval call binding the contract method 0x6b0b2592.
//
// Solidity: function vertexExists(bytes32 vId) view returns(bool)
func (_IChallengeManager *IChallengeManagerSession) VertexExists(vId [32]byte) (bool, error) {
	return _IChallengeManager.Contract.VertexExists(&_IChallengeManager.CallOpts, vId)
}

// VertexExists is a free data retrieval call binding the contract method 0x6b0b2592.
//
// Solidity: function vertexExists(bytes32 vId) view returns(bool)
func (_IChallengeManager *IChallengeManagerCallerSession) VertexExists(vId [32]byte) (bool, error) {
	return _IChallengeManager.Contract.VertexExists(&_IChallengeManager.CallOpts, vId)
}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_IChallengeManager *IChallengeManagerCaller) WinningClaim(opts *bind.CallOpts, challengeId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IChallengeManager.contract.Call(opts, &out, "winningClaim", challengeId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_IChallengeManager *IChallengeManagerSession) WinningClaim(challengeId [32]byte) ([32]byte, error) {
	return _IChallengeManager.Contract.WinningClaim(&_IChallengeManager.CallOpts, challengeId)
}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_IChallengeManager *IChallengeManagerCallerSession) WinningClaim(challengeId [32]byte) ([32]byte, error) {
	return _IChallengeManager.Contract.WinningClaim(&_IChallengeManager.CallOpts, challengeId)
}

// AddLeaf is a paid mutator transaction binding the contract method 0x9e7cee54.
//
// Solidity: function addLeaf((bytes32,bytes32,uint256,bytes32,bytes32,bytes32[],bytes32,bytes32[]) leafData, bytes proof1, bytes proof2) payable returns(bytes32)
func (_IChallengeManager *IChallengeManagerTransactor) AddLeaf(opts *bind.TransactOpts, leafData AddLeafArgs, proof1 []byte, proof2 []byte) (*types.Transaction, error) {
	return _IChallengeManager.contract.Transact(opts, "addLeaf", leafData, proof1, proof2)
}

// AddLeaf is a paid mutator transaction binding the contract method 0x9e7cee54.
//
// Solidity: function addLeaf((bytes32,bytes32,uint256,bytes32,bytes32,bytes32[],bytes32,bytes32[]) leafData, bytes proof1, bytes proof2) payable returns(bytes32)
func (_IChallengeManager *IChallengeManagerSession) AddLeaf(leafData AddLeafArgs, proof1 []byte, proof2 []byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.AddLeaf(&_IChallengeManager.TransactOpts, leafData, proof1, proof2)
}

// AddLeaf is a paid mutator transaction binding the contract method 0x9e7cee54.
//
// Solidity: function addLeaf((bytes32,bytes32,uint256,bytes32,bytes32,bytes32[],bytes32,bytes32[]) leafData, bytes proof1, bytes proof2) payable returns(bytes32)
func (_IChallengeManager *IChallengeManagerTransactorSession) AddLeaf(leafData AddLeafArgs, proof1 []byte, proof2 []byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.AddLeaf(&_IChallengeManager.TransactOpts, leafData, proof1, proof2)
}

// Bisect is a paid mutator transaction binding the contract method 0x359076cf.
//
// Solidity: function bisect(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManager *IChallengeManagerTransactor) Bisect(opts *bind.TransactOpts, vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManager.contract.Transact(opts, "bisect", vId, prefixHistoryRoot, prefixProof)
}

// Bisect is a paid mutator transaction binding the contract method 0x359076cf.
//
// Solidity: function bisect(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManager *IChallengeManagerSession) Bisect(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.Bisect(&_IChallengeManager.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// Bisect is a paid mutator transaction binding the contract method 0x359076cf.
//
// Solidity: function bisect(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManager *IChallengeManagerTransactorSession) Bisect(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.Bisect(&_IChallengeManager.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// ConfirmForPsTimer is a paid mutator transaction binding the contract method 0x1d5618ac.
//
// Solidity: function confirmForPsTimer(bytes32 vId) returns()
func (_IChallengeManager *IChallengeManagerTransactor) ConfirmForPsTimer(opts *bind.TransactOpts, vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.contract.Transact(opts, "confirmForPsTimer", vId)
}

// ConfirmForPsTimer is a paid mutator transaction binding the contract method 0x1d5618ac.
//
// Solidity: function confirmForPsTimer(bytes32 vId) returns()
func (_IChallengeManager *IChallengeManagerSession) ConfirmForPsTimer(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.ConfirmForPsTimer(&_IChallengeManager.TransactOpts, vId)
}

// ConfirmForPsTimer is a paid mutator transaction binding the contract method 0x1d5618ac.
//
// Solidity: function confirmForPsTimer(bytes32 vId) returns()
func (_IChallengeManager *IChallengeManagerTransactorSession) ConfirmForPsTimer(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.ConfirmForPsTimer(&_IChallengeManager.TransactOpts, vId)
}

// ConfirmForSucessionChallengeWin is a paid mutator transaction binding the contract method 0xd1bac9a4.
//
// Solidity: function confirmForSucessionChallengeWin(bytes32 vId) returns()
func (_IChallengeManager *IChallengeManagerTransactor) ConfirmForSucessionChallengeWin(opts *bind.TransactOpts, vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.contract.Transact(opts, "confirmForSucessionChallengeWin", vId)
}

// ConfirmForSucessionChallengeWin is a paid mutator transaction binding the contract method 0xd1bac9a4.
//
// Solidity: function confirmForSucessionChallengeWin(bytes32 vId) returns()
func (_IChallengeManager *IChallengeManagerSession) ConfirmForSucessionChallengeWin(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.ConfirmForSucessionChallengeWin(&_IChallengeManager.TransactOpts, vId)
}

// ConfirmForSucessionChallengeWin is a paid mutator transaction binding the contract method 0xd1bac9a4.
//
// Solidity: function confirmForSucessionChallengeWin(bytes32 vId) returns()
func (_IChallengeManager *IChallengeManagerTransactorSession) ConfirmForSucessionChallengeWin(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.ConfirmForSucessionChallengeWin(&_IChallengeManager.TransactOpts, vId)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0xf696dc55.
//
// Solidity: function createChallenge(bytes32 assertionId) returns(bytes32)
func (_IChallengeManager *IChallengeManagerTransactor) CreateChallenge(opts *bind.TransactOpts, assertionId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.contract.Transact(opts, "createChallenge", assertionId)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0xf696dc55.
//
// Solidity: function createChallenge(bytes32 assertionId) returns(bytes32)
func (_IChallengeManager *IChallengeManagerSession) CreateChallenge(assertionId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.CreateChallenge(&_IChallengeManager.TransactOpts, assertionId)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0xf696dc55.
//
// Solidity: function createChallenge(bytes32 assertionId) returns(bytes32)
func (_IChallengeManager *IChallengeManagerTransactorSession) CreateChallenge(assertionId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.CreateChallenge(&_IChallengeManager.TransactOpts, assertionId)
}

// CreateSubChallenge is a paid mutator transaction binding the contract method 0xbd623251.
//
// Solidity: function createSubChallenge(bytes32 vId) returns(bytes32)
func (_IChallengeManager *IChallengeManagerTransactor) CreateSubChallenge(opts *bind.TransactOpts, vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.contract.Transact(opts, "createSubChallenge", vId)
}

// CreateSubChallenge is a paid mutator transaction binding the contract method 0xbd623251.
//
// Solidity: function createSubChallenge(bytes32 vId) returns(bytes32)
func (_IChallengeManager *IChallengeManagerSession) CreateSubChallenge(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.CreateSubChallenge(&_IChallengeManager.TransactOpts, vId)
}

// CreateSubChallenge is a paid mutator transaction binding the contract method 0xbd623251.
//
// Solidity: function createSubChallenge(bytes32 vId) returns(bytes32)
func (_IChallengeManager *IChallengeManagerTransactorSession) CreateSubChallenge(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.CreateSubChallenge(&_IChallengeManager.TransactOpts, vId)
}

// Initialize is a paid mutator transaction binding the contract method 0x9e3d87cd.
//
// Solidity: function initialize(address _assertionChain, uint256 _miniStakeValue, uint256 _challengePeriod, address _oneStepProofEntry) returns()
func (_IChallengeManager *IChallengeManagerTransactor) Initialize(opts *bind.TransactOpts, _assertionChain common.Address, _miniStakeValue *big.Int, _challengePeriod *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _IChallengeManager.contract.Transact(opts, "initialize", _assertionChain, _miniStakeValue, _challengePeriod, _oneStepProofEntry)
}

// Initialize is a paid mutator transaction binding the contract method 0x9e3d87cd.
//
// Solidity: function initialize(address _assertionChain, uint256 _miniStakeValue, uint256 _challengePeriod, address _oneStepProofEntry) returns()
func (_IChallengeManager *IChallengeManagerSession) Initialize(_assertionChain common.Address, _miniStakeValue *big.Int, _challengePeriod *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _IChallengeManager.Contract.Initialize(&_IChallengeManager.TransactOpts, _assertionChain, _miniStakeValue, _challengePeriod, _oneStepProofEntry)
}

// Initialize is a paid mutator transaction binding the contract method 0x9e3d87cd.
//
// Solidity: function initialize(address _assertionChain, uint256 _miniStakeValue, uint256 _challengePeriod, address _oneStepProofEntry) returns()
func (_IChallengeManager *IChallengeManagerTransactorSession) Initialize(_assertionChain common.Address, _miniStakeValue *big.Int, _challengePeriod *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _IChallengeManager.Contract.Initialize(&_IChallengeManager.TransactOpts, _assertionChain, _miniStakeValue, _challengePeriod, _oneStepProofEntry)
}

// Merge is a paid mutator transaction binding the contract method 0x597e1e0b.
//
// Solidity: function merge(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManager *IChallengeManagerTransactor) Merge(opts *bind.TransactOpts, vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManager.contract.Transact(opts, "merge", vId, prefixHistoryRoot, prefixProof)
}

// Merge is a paid mutator transaction binding the contract method 0x597e1e0b.
//
// Solidity: function merge(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManager *IChallengeManagerSession) Merge(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.Merge(&_IChallengeManager.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// Merge is a paid mutator transaction binding the contract method 0x597e1e0b.
//
// Solidity: function merge(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManager *IChallengeManagerTransactorSession) Merge(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManager.Contract.Merge(&_IChallengeManager.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// IChallengeManagerCoreMetaData contains all meta data concerning the IChallengeManagerCore contract.
var IChallengeManagerCoreMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"historyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"firstState\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"firstStatehistoryProof\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32\",\"name\":\"lastState\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"lastStatehistoryProof\",\"type\":\"bytes32[]\"}],\"internalType\":\"structAddLeafArgs\",\"name\":\"leafData\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof1\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof2\",\"type\":\"bytes\"}],\"name\":\"addLeaf\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"prefixHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"bisect\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"confirmForPsTimer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"confirmForSucessionChallengeWin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"createChallenge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"createSubChallenge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_miniStakeValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriod\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"prefixHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"merge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// IChallengeManagerCoreABI is the input ABI used to generate the binding from.
// Deprecated: Use IChallengeManagerCoreMetaData.ABI instead.
var IChallengeManagerCoreABI = IChallengeManagerCoreMetaData.ABI

// IChallengeManagerCore is an auto generated Go binding around an Ethereum contract.
type IChallengeManagerCore struct {
	IChallengeManagerCoreCaller     // Read-only binding to the contract
	IChallengeManagerCoreTransactor // Write-only binding to the contract
	IChallengeManagerCoreFilterer   // Log filterer for contract events
}

// IChallengeManagerCoreCaller is an auto generated read-only Go binding around an Ethereum contract.
type IChallengeManagerCoreCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IChallengeManagerCoreTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IChallengeManagerCoreTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IChallengeManagerCoreFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IChallengeManagerCoreFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IChallengeManagerCoreSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IChallengeManagerCoreSession struct {
	Contract     *IChallengeManagerCore // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// IChallengeManagerCoreCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IChallengeManagerCoreCallerSession struct {
	Contract *IChallengeManagerCoreCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// IChallengeManagerCoreTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IChallengeManagerCoreTransactorSession struct {
	Contract     *IChallengeManagerCoreTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// IChallengeManagerCoreRaw is an auto generated low-level Go binding around an Ethereum contract.
type IChallengeManagerCoreRaw struct {
	Contract *IChallengeManagerCore // Generic contract binding to access the raw methods on
}

// IChallengeManagerCoreCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IChallengeManagerCoreCallerRaw struct {
	Contract *IChallengeManagerCoreCaller // Generic read-only contract binding to access the raw methods on
}

// IChallengeManagerCoreTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IChallengeManagerCoreTransactorRaw struct {
	Contract *IChallengeManagerCoreTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIChallengeManagerCore creates a new instance of IChallengeManagerCore, bound to a specific deployed contract.
func NewIChallengeManagerCore(address common.Address, backend bind.ContractBackend) (*IChallengeManagerCore, error) {
	contract, err := bindIChallengeManagerCore(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerCore{IChallengeManagerCoreCaller: IChallengeManagerCoreCaller{contract: contract}, IChallengeManagerCoreTransactor: IChallengeManagerCoreTransactor{contract: contract}, IChallengeManagerCoreFilterer: IChallengeManagerCoreFilterer{contract: contract}}, nil
}

// NewIChallengeManagerCoreCaller creates a new read-only instance of IChallengeManagerCore, bound to a specific deployed contract.
func NewIChallengeManagerCoreCaller(address common.Address, caller bind.ContractCaller) (*IChallengeManagerCoreCaller, error) {
	contract, err := bindIChallengeManagerCore(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerCoreCaller{contract: contract}, nil
}

// NewIChallengeManagerCoreTransactor creates a new write-only instance of IChallengeManagerCore, bound to a specific deployed contract.
func NewIChallengeManagerCoreTransactor(address common.Address, transactor bind.ContractTransactor) (*IChallengeManagerCoreTransactor, error) {
	contract, err := bindIChallengeManagerCore(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerCoreTransactor{contract: contract}, nil
}

// NewIChallengeManagerCoreFilterer creates a new log filterer instance of IChallengeManagerCore, bound to a specific deployed contract.
func NewIChallengeManagerCoreFilterer(address common.Address, filterer bind.ContractFilterer) (*IChallengeManagerCoreFilterer, error) {
	contract, err := bindIChallengeManagerCore(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerCoreFilterer{contract: contract}, nil
}

// bindIChallengeManagerCore binds a generic wrapper to an already deployed contract.
func bindIChallengeManagerCore(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(IChallengeManagerCoreABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IChallengeManagerCore *IChallengeManagerCoreRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IChallengeManagerCore.Contract.IChallengeManagerCoreCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IChallengeManagerCore *IChallengeManagerCoreRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.IChallengeManagerCoreTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IChallengeManagerCore *IChallengeManagerCoreRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.IChallengeManagerCoreTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IChallengeManagerCore *IChallengeManagerCoreCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IChallengeManagerCore.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IChallengeManagerCore *IChallengeManagerCoreTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IChallengeManagerCore *IChallengeManagerCoreTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.contract.Transact(opts, method, params...)
}

// AddLeaf is a paid mutator transaction binding the contract method 0x9e7cee54.
//
// Solidity: function addLeaf((bytes32,bytes32,uint256,bytes32,bytes32,bytes32[],bytes32,bytes32[]) leafData, bytes proof1, bytes proof2) payable returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreTransactor) AddLeaf(opts *bind.TransactOpts, leafData AddLeafArgs, proof1 []byte, proof2 []byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.contract.Transact(opts, "addLeaf", leafData, proof1, proof2)
}

// AddLeaf is a paid mutator transaction binding the contract method 0x9e7cee54.
//
// Solidity: function addLeaf((bytes32,bytes32,uint256,bytes32,bytes32,bytes32[],bytes32,bytes32[]) leafData, bytes proof1, bytes proof2) payable returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreSession) AddLeaf(leafData AddLeafArgs, proof1 []byte, proof2 []byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.AddLeaf(&_IChallengeManagerCore.TransactOpts, leafData, proof1, proof2)
}

// AddLeaf is a paid mutator transaction binding the contract method 0x9e7cee54.
//
// Solidity: function addLeaf((bytes32,bytes32,uint256,bytes32,bytes32,bytes32[],bytes32,bytes32[]) leafData, bytes proof1, bytes proof2) payable returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreTransactorSession) AddLeaf(leafData AddLeafArgs, proof1 []byte, proof2 []byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.AddLeaf(&_IChallengeManagerCore.TransactOpts, leafData, proof1, proof2)
}

// Bisect is a paid mutator transaction binding the contract method 0x359076cf.
//
// Solidity: function bisect(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreTransactor) Bisect(opts *bind.TransactOpts, vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.contract.Transact(opts, "bisect", vId, prefixHistoryRoot, prefixProof)
}

// Bisect is a paid mutator transaction binding the contract method 0x359076cf.
//
// Solidity: function bisect(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreSession) Bisect(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.Bisect(&_IChallengeManagerCore.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// Bisect is a paid mutator transaction binding the contract method 0x359076cf.
//
// Solidity: function bisect(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreTransactorSession) Bisect(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.Bisect(&_IChallengeManagerCore.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// ConfirmForPsTimer is a paid mutator transaction binding the contract method 0x1d5618ac.
//
// Solidity: function confirmForPsTimer(bytes32 vId) returns()
func (_IChallengeManagerCore *IChallengeManagerCoreTransactor) ConfirmForPsTimer(opts *bind.TransactOpts, vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.contract.Transact(opts, "confirmForPsTimer", vId)
}

// ConfirmForPsTimer is a paid mutator transaction binding the contract method 0x1d5618ac.
//
// Solidity: function confirmForPsTimer(bytes32 vId) returns()
func (_IChallengeManagerCore *IChallengeManagerCoreSession) ConfirmForPsTimer(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.ConfirmForPsTimer(&_IChallengeManagerCore.TransactOpts, vId)
}

// ConfirmForPsTimer is a paid mutator transaction binding the contract method 0x1d5618ac.
//
// Solidity: function confirmForPsTimer(bytes32 vId) returns()
func (_IChallengeManagerCore *IChallengeManagerCoreTransactorSession) ConfirmForPsTimer(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.ConfirmForPsTimer(&_IChallengeManagerCore.TransactOpts, vId)
}

// ConfirmForSucessionChallengeWin is a paid mutator transaction binding the contract method 0xd1bac9a4.
//
// Solidity: function confirmForSucessionChallengeWin(bytes32 vId) returns()
func (_IChallengeManagerCore *IChallengeManagerCoreTransactor) ConfirmForSucessionChallengeWin(opts *bind.TransactOpts, vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.contract.Transact(opts, "confirmForSucessionChallengeWin", vId)
}

// ConfirmForSucessionChallengeWin is a paid mutator transaction binding the contract method 0xd1bac9a4.
//
// Solidity: function confirmForSucessionChallengeWin(bytes32 vId) returns()
func (_IChallengeManagerCore *IChallengeManagerCoreSession) ConfirmForSucessionChallengeWin(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.ConfirmForSucessionChallengeWin(&_IChallengeManagerCore.TransactOpts, vId)
}

// ConfirmForSucessionChallengeWin is a paid mutator transaction binding the contract method 0xd1bac9a4.
//
// Solidity: function confirmForSucessionChallengeWin(bytes32 vId) returns()
func (_IChallengeManagerCore *IChallengeManagerCoreTransactorSession) ConfirmForSucessionChallengeWin(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.ConfirmForSucessionChallengeWin(&_IChallengeManagerCore.TransactOpts, vId)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0xf696dc55.
//
// Solidity: function createChallenge(bytes32 assertionId) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreTransactor) CreateChallenge(opts *bind.TransactOpts, assertionId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.contract.Transact(opts, "createChallenge", assertionId)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0xf696dc55.
//
// Solidity: function createChallenge(bytes32 assertionId) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreSession) CreateChallenge(assertionId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.CreateChallenge(&_IChallengeManagerCore.TransactOpts, assertionId)
}

// CreateChallenge is a paid mutator transaction binding the contract method 0xf696dc55.
//
// Solidity: function createChallenge(bytes32 assertionId) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreTransactorSession) CreateChallenge(assertionId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.CreateChallenge(&_IChallengeManagerCore.TransactOpts, assertionId)
}

// CreateSubChallenge is a paid mutator transaction binding the contract method 0xbd623251.
//
// Solidity: function createSubChallenge(bytes32 vId) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreTransactor) CreateSubChallenge(opts *bind.TransactOpts, vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.contract.Transact(opts, "createSubChallenge", vId)
}

// CreateSubChallenge is a paid mutator transaction binding the contract method 0xbd623251.
//
// Solidity: function createSubChallenge(bytes32 vId) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreSession) CreateSubChallenge(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.CreateSubChallenge(&_IChallengeManagerCore.TransactOpts, vId)
}

// CreateSubChallenge is a paid mutator transaction binding the contract method 0xbd623251.
//
// Solidity: function createSubChallenge(bytes32 vId) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreTransactorSession) CreateSubChallenge(vId [32]byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.CreateSubChallenge(&_IChallengeManagerCore.TransactOpts, vId)
}

// Initialize is a paid mutator transaction binding the contract method 0x9e3d87cd.
//
// Solidity: function initialize(address _assertionChain, uint256 _miniStakeValue, uint256 _challengePeriod, address _oneStepProofEntry) returns()
func (_IChallengeManagerCore *IChallengeManagerCoreTransactor) Initialize(opts *bind.TransactOpts, _assertionChain common.Address, _miniStakeValue *big.Int, _challengePeriod *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _IChallengeManagerCore.contract.Transact(opts, "initialize", _assertionChain, _miniStakeValue, _challengePeriod, _oneStepProofEntry)
}

// Initialize is a paid mutator transaction binding the contract method 0x9e3d87cd.
//
// Solidity: function initialize(address _assertionChain, uint256 _miniStakeValue, uint256 _challengePeriod, address _oneStepProofEntry) returns()
func (_IChallengeManagerCore *IChallengeManagerCoreSession) Initialize(_assertionChain common.Address, _miniStakeValue *big.Int, _challengePeriod *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.Initialize(&_IChallengeManagerCore.TransactOpts, _assertionChain, _miniStakeValue, _challengePeriod, _oneStepProofEntry)
}

// Initialize is a paid mutator transaction binding the contract method 0x9e3d87cd.
//
// Solidity: function initialize(address _assertionChain, uint256 _miniStakeValue, uint256 _challengePeriod, address _oneStepProofEntry) returns()
func (_IChallengeManagerCore *IChallengeManagerCoreTransactorSession) Initialize(_assertionChain common.Address, _miniStakeValue *big.Int, _challengePeriod *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.Initialize(&_IChallengeManagerCore.TransactOpts, _assertionChain, _miniStakeValue, _challengePeriod, _oneStepProofEntry)
}

// Merge is a paid mutator transaction binding the contract method 0x597e1e0b.
//
// Solidity: function merge(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreTransactor) Merge(opts *bind.TransactOpts, vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.contract.Transact(opts, "merge", vId, prefixHistoryRoot, prefixProof)
}

// Merge is a paid mutator transaction binding the contract method 0x597e1e0b.
//
// Solidity: function merge(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreSession) Merge(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.Merge(&_IChallengeManagerCore.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// Merge is a paid mutator transaction binding the contract method 0x597e1e0b.
//
// Solidity: function merge(bytes32 vId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32)
func (_IChallengeManagerCore *IChallengeManagerCoreTransactorSession) Merge(vId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IChallengeManagerCore.Contract.Merge(&_IChallengeManagerCore.TransactOpts, vId, prefixHistoryRoot, prefixProof)
}

// IChallengeManagerExternalViewMetaData contains all meta data concerning the IChallengeManagerExternalView contract.
var IChallengeManagerExternalViewMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"challengeExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"childrenAreAtOneStepFork\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"getChallenge\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"rootId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"winningClaim\",\"type\":\"bytes32\"},{\"internalType\":\"enumChallengeType\",\"name\":\"challengeType\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"challenger\",\"type\":\"address\"}],\"internalType\":\"structChallenge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"getCurrentPsTimer\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"getVertex\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"historyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"successionChallenge\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"predecessorId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"enumVertexStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"psId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"psLastUpdatedTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"flushedPsTimeSec\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"lowestHeightSuccessorId\",\"type\":\"bytes32\"}],\"internalType\":\"structChallengeVertex\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"hasConfirmedSibling\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"vertexExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"winningClaim\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IChallengeManagerExternalViewABI is the input ABI used to generate the binding from.
// Deprecated: Use IChallengeManagerExternalViewMetaData.ABI instead.
var IChallengeManagerExternalViewABI = IChallengeManagerExternalViewMetaData.ABI

// IChallengeManagerExternalView is an auto generated Go binding around an Ethereum contract.
type IChallengeManagerExternalView struct {
	IChallengeManagerExternalViewCaller     // Read-only binding to the contract
	IChallengeManagerExternalViewTransactor // Write-only binding to the contract
	IChallengeManagerExternalViewFilterer   // Log filterer for contract events
}

// IChallengeManagerExternalViewCaller is an auto generated read-only Go binding around an Ethereum contract.
type IChallengeManagerExternalViewCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IChallengeManagerExternalViewTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IChallengeManagerExternalViewTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IChallengeManagerExternalViewFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IChallengeManagerExternalViewFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IChallengeManagerExternalViewSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IChallengeManagerExternalViewSession struct {
	Contract     *IChallengeManagerExternalView // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                  // Call options to use throughout this session
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// IChallengeManagerExternalViewCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IChallengeManagerExternalViewCallerSession struct {
	Contract *IChallengeManagerExternalViewCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                        // Call options to use throughout this session
}

// IChallengeManagerExternalViewTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IChallengeManagerExternalViewTransactorSession struct {
	Contract     *IChallengeManagerExternalViewTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                        // Transaction auth options to use throughout this session
}

// IChallengeManagerExternalViewRaw is an auto generated low-level Go binding around an Ethereum contract.
type IChallengeManagerExternalViewRaw struct {
	Contract *IChallengeManagerExternalView // Generic contract binding to access the raw methods on
}

// IChallengeManagerExternalViewCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IChallengeManagerExternalViewCallerRaw struct {
	Contract *IChallengeManagerExternalViewCaller // Generic read-only contract binding to access the raw methods on
}

// IChallengeManagerExternalViewTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IChallengeManagerExternalViewTransactorRaw struct {
	Contract *IChallengeManagerExternalViewTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIChallengeManagerExternalView creates a new instance of IChallengeManagerExternalView, bound to a specific deployed contract.
func NewIChallengeManagerExternalView(address common.Address, backend bind.ContractBackend) (*IChallengeManagerExternalView, error) {
	contract, err := bindIChallengeManagerExternalView(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerExternalView{IChallengeManagerExternalViewCaller: IChallengeManagerExternalViewCaller{contract: contract}, IChallengeManagerExternalViewTransactor: IChallengeManagerExternalViewTransactor{contract: contract}, IChallengeManagerExternalViewFilterer: IChallengeManagerExternalViewFilterer{contract: contract}}, nil
}

// NewIChallengeManagerExternalViewCaller creates a new read-only instance of IChallengeManagerExternalView, bound to a specific deployed contract.
func NewIChallengeManagerExternalViewCaller(address common.Address, caller bind.ContractCaller) (*IChallengeManagerExternalViewCaller, error) {
	contract, err := bindIChallengeManagerExternalView(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerExternalViewCaller{contract: contract}, nil
}

// NewIChallengeManagerExternalViewTransactor creates a new write-only instance of IChallengeManagerExternalView, bound to a specific deployed contract.
func NewIChallengeManagerExternalViewTransactor(address common.Address, transactor bind.ContractTransactor) (*IChallengeManagerExternalViewTransactor, error) {
	contract, err := bindIChallengeManagerExternalView(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerExternalViewTransactor{contract: contract}, nil
}

// NewIChallengeManagerExternalViewFilterer creates a new log filterer instance of IChallengeManagerExternalView, bound to a specific deployed contract.
func NewIChallengeManagerExternalViewFilterer(address common.Address, filterer bind.ContractFilterer) (*IChallengeManagerExternalViewFilterer, error) {
	contract, err := bindIChallengeManagerExternalView(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IChallengeManagerExternalViewFilterer{contract: contract}, nil
}

// bindIChallengeManagerExternalView binds a generic wrapper to an already deployed contract.
func bindIChallengeManagerExternalView(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(IChallengeManagerExternalViewABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IChallengeManagerExternalView.Contract.IChallengeManagerExternalViewCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IChallengeManagerExternalView.Contract.IChallengeManagerExternalViewTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IChallengeManagerExternalView.Contract.IChallengeManagerExternalViewTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IChallengeManagerExternalView.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IChallengeManagerExternalView.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IChallengeManagerExternalView.Contract.contract.Transact(opts, method, params...)
}

// ChallengeExists is a free data retrieval call binding the contract method 0x1b7bbecb.
//
// Solidity: function challengeExists(bytes32 challengeId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCaller) ChallengeExists(opts *bind.CallOpts, challengeId [32]byte) (bool, error) {
	var out []interface{}
	err := _IChallengeManagerExternalView.contract.Call(opts, &out, "challengeExists", challengeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ChallengeExists is a free data retrieval call binding the contract method 0x1b7bbecb.
//
// Solidity: function challengeExists(bytes32 challengeId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewSession) ChallengeExists(challengeId [32]byte) (bool, error) {
	return _IChallengeManagerExternalView.Contract.ChallengeExists(&_IChallengeManagerExternalView.CallOpts, challengeId)
}

// ChallengeExists is a free data retrieval call binding the contract method 0x1b7bbecb.
//
// Solidity: function challengeExists(bytes32 challengeId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCallerSession) ChallengeExists(challengeId [32]byte) (bool, error) {
	return _IChallengeManagerExternalView.Contract.ChallengeExists(&_IChallengeManagerExternalView.CallOpts, challengeId)
}

// ChildrenAreAtOneStepFork is a free data retrieval call binding the contract method 0x7a4d47dc.
//
// Solidity: function childrenAreAtOneStepFork(bytes32 vId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCaller) ChildrenAreAtOneStepFork(opts *bind.CallOpts, vId [32]byte) (bool, error) {
	var out []interface{}
	err := _IChallengeManagerExternalView.contract.Call(opts, &out, "childrenAreAtOneStepFork", vId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ChildrenAreAtOneStepFork is a free data retrieval call binding the contract method 0x7a4d47dc.
//
// Solidity: function childrenAreAtOneStepFork(bytes32 vId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewSession) ChildrenAreAtOneStepFork(vId [32]byte) (bool, error) {
	return _IChallengeManagerExternalView.Contract.ChildrenAreAtOneStepFork(&_IChallengeManagerExternalView.CallOpts, vId)
}

// ChildrenAreAtOneStepFork is a free data retrieval call binding the contract method 0x7a4d47dc.
//
// Solidity: function childrenAreAtOneStepFork(bytes32 vId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCallerSession) ChildrenAreAtOneStepFork(vId [32]byte) (bool, error) {
	return _IChallengeManagerExternalView.Contract.ChildrenAreAtOneStepFork(&_IChallengeManagerExternalView.CallOpts, vId)
}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,bytes32,uint8,address))
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCaller) GetChallenge(opts *bind.CallOpts, challengeId [32]byte) (Challenge, error) {
	var out []interface{}
	err := _IChallengeManagerExternalView.contract.Call(opts, &out, "getChallenge", challengeId)

	if err != nil {
		return *new(Challenge), err
	}

	out0 := *abi.ConvertType(out[0], new(Challenge)).(*Challenge)

	return out0, err

}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,bytes32,uint8,address))
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewSession) GetChallenge(challengeId [32]byte) (Challenge, error) {
	return _IChallengeManagerExternalView.Contract.GetChallenge(&_IChallengeManagerExternalView.CallOpts, challengeId)
}

// GetChallenge is a free data retrieval call binding the contract method 0x458d2bf1.
//
// Solidity: function getChallenge(bytes32 challengeId) view returns((bytes32,bytes32,uint8,address))
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCallerSession) GetChallenge(challengeId [32]byte) (Challenge, error) {
	return _IChallengeManagerExternalView.Contract.GetChallenge(&_IChallengeManagerExternalView.CallOpts, challengeId)
}

// GetCurrentPsTimer is a free data retrieval call binding the contract method 0x8ac04349.
//
// Solidity: function getCurrentPsTimer(bytes32 vId) view returns(uint256)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCaller) GetCurrentPsTimer(opts *bind.CallOpts, vId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IChallengeManagerExternalView.contract.Call(opts, &out, "getCurrentPsTimer", vId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentPsTimer is a free data retrieval call binding the contract method 0x8ac04349.
//
// Solidity: function getCurrentPsTimer(bytes32 vId) view returns(uint256)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewSession) GetCurrentPsTimer(vId [32]byte) (*big.Int, error) {
	return _IChallengeManagerExternalView.Contract.GetCurrentPsTimer(&_IChallengeManagerExternalView.CallOpts, vId)
}

// GetCurrentPsTimer is a free data retrieval call binding the contract method 0x8ac04349.
//
// Solidity: function getCurrentPsTimer(bytes32 vId) view returns(uint256)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCallerSession) GetCurrentPsTimer(vId [32]byte) (*big.Int, error) {
	return _IChallengeManagerExternalView.Contract.GetCurrentPsTimer(&_IChallengeManagerExternalView.CallOpts, vId)
}

// GetVertex is a free data retrieval call binding the contract method 0x86f048ed.
//
// Solidity: function getVertex(bytes32 vId) view returns((bytes32,bytes32,uint256,bytes32,bytes32,bytes32,address,uint8,bytes32,uint256,uint256,bytes32))
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCaller) GetVertex(opts *bind.CallOpts, vId [32]byte) (ChallengeVertex, error) {
	var out []interface{}
	err := _IChallengeManagerExternalView.contract.Call(opts, &out, "getVertex", vId)

	if err != nil {
		return *new(ChallengeVertex), err
	}

	out0 := *abi.ConvertType(out[0], new(ChallengeVertex)).(*ChallengeVertex)

	return out0, err

}

// GetVertex is a free data retrieval call binding the contract method 0x86f048ed.
//
// Solidity: function getVertex(bytes32 vId) view returns((bytes32,bytes32,uint256,bytes32,bytes32,bytes32,address,uint8,bytes32,uint256,uint256,bytes32))
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewSession) GetVertex(vId [32]byte) (ChallengeVertex, error) {
	return _IChallengeManagerExternalView.Contract.GetVertex(&_IChallengeManagerExternalView.CallOpts, vId)
}

// GetVertex is a free data retrieval call binding the contract method 0x86f048ed.
//
// Solidity: function getVertex(bytes32 vId) view returns((bytes32,bytes32,uint256,bytes32,bytes32,bytes32,address,uint8,bytes32,uint256,uint256,bytes32))
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCallerSession) GetVertex(vId [32]byte) (ChallengeVertex, error) {
	return _IChallengeManagerExternalView.Contract.GetVertex(&_IChallengeManagerExternalView.CallOpts, vId)
}

// HasConfirmedSibling is a free data retrieval call binding the contract method 0x98b67d59.
//
// Solidity: function hasConfirmedSibling(bytes32 vId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCaller) HasConfirmedSibling(opts *bind.CallOpts, vId [32]byte) (bool, error) {
	var out []interface{}
	err := _IChallengeManagerExternalView.contract.Call(opts, &out, "hasConfirmedSibling", vId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasConfirmedSibling is a free data retrieval call binding the contract method 0x98b67d59.
//
// Solidity: function hasConfirmedSibling(bytes32 vId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewSession) HasConfirmedSibling(vId [32]byte) (bool, error) {
	return _IChallengeManagerExternalView.Contract.HasConfirmedSibling(&_IChallengeManagerExternalView.CallOpts, vId)
}

// HasConfirmedSibling is a free data retrieval call binding the contract method 0x98b67d59.
//
// Solidity: function hasConfirmedSibling(bytes32 vId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCallerSession) HasConfirmedSibling(vId [32]byte) (bool, error) {
	return _IChallengeManagerExternalView.Contract.HasConfirmedSibling(&_IChallengeManagerExternalView.CallOpts, vId)
}

// VertexExists is a free data retrieval call binding the contract method 0x6b0b2592.
//
// Solidity: function vertexExists(bytes32 vId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCaller) VertexExists(opts *bind.CallOpts, vId [32]byte) (bool, error) {
	var out []interface{}
	err := _IChallengeManagerExternalView.contract.Call(opts, &out, "vertexExists", vId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// VertexExists is a free data retrieval call binding the contract method 0x6b0b2592.
//
// Solidity: function vertexExists(bytes32 vId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewSession) VertexExists(vId [32]byte) (bool, error) {
	return _IChallengeManagerExternalView.Contract.VertexExists(&_IChallengeManagerExternalView.CallOpts, vId)
}

// VertexExists is a free data retrieval call binding the contract method 0x6b0b2592.
//
// Solidity: function vertexExists(bytes32 vId) view returns(bool)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCallerSession) VertexExists(vId [32]byte) (bool, error) {
	return _IChallengeManagerExternalView.Contract.VertexExists(&_IChallengeManagerExternalView.CallOpts, vId)
}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCaller) WinningClaim(opts *bind.CallOpts, challengeId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IChallengeManagerExternalView.contract.Call(opts, &out, "winningClaim", challengeId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewSession) WinningClaim(challengeId [32]byte) ([32]byte, error) {
	return _IChallengeManagerExternalView.Contract.WinningClaim(&_IChallengeManagerExternalView.CallOpts, challengeId)
}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_IChallengeManagerExternalView *IChallengeManagerExternalViewCallerSession) WinningClaim(challengeId [32]byte) ([32]byte, error) {
	return _IChallengeManagerExternalView.Contract.WinningClaim(&_IChallengeManagerExternalView.CallOpts, challengeId)
}

// IEdgeChallengeManagerMetaData contains all meta data concerning the IEdgeChallengeManager contract.
var IEdgeChallengeManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"eId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"prefixHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"bisectEdge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"}],\"name\":\"calculateEdgeId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"}],\"name\":\"calculateMutualId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"eId\",\"type\":\"bytes32\"}],\"name\":\"confirmEdgeByChildren\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"eId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"}],\"name\":\"confirmEdgeByClaim\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"edgeId\",\"type\":\"bytes32\"},{\"components\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"machineStep\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"internalType\":\"structOneStepData\",\"name\":\"oneStepData\",\"type\":\"tuple\"},{\"internalType\":\"bytes32[]\",\"name\":\"beforeHistoryInclusionProof\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"afterHistoryInclusionProof\",\"type\":\"bytes32[]\"}],\"name\":\"confirmEdgeByOneStepProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"eId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"ancestorIds\",\"type\":\"bytes32[]\"}],\"name\":\"confirmEdgeByTime\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"enumEdgeType\",\"name\":\"edgeType\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"}],\"internalType\":\"structCreateEdgeArgs\",\"name\":\"args\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"createLayerZeroEdge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"eId\",\"type\":\"bytes32\"}],\"name\":\"edgeExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"eId\",\"type\":\"bytes32\"}],\"name\":\"getEdge\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"originId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"startHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"startHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"endHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"endHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"lowerChildId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"upperChildId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"createdWhen\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"enumEdgeStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"enumEdgeType\",\"name\":\"eType\",\"type\":\"uint8\"}],\"internalType\":\"structChallengeEdge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"eId\",\"type\":\"bytes32\"}],\"name\":\"hasLengthOneRival\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"eId\",\"type\":\"bytes32\"}],\"name\":\"hasRival\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriodSec\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"eId\",\"type\":\"bytes32\"}],\"name\":\"timeUnrivaled\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IEdgeChallengeManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use IEdgeChallengeManagerMetaData.ABI instead.
var IEdgeChallengeManagerABI = IEdgeChallengeManagerMetaData.ABI

// IEdgeChallengeManager is an auto generated Go binding around an Ethereum contract.
type IEdgeChallengeManager struct {
	IEdgeChallengeManagerCaller     // Read-only binding to the contract
	IEdgeChallengeManagerTransactor // Write-only binding to the contract
	IEdgeChallengeManagerFilterer   // Log filterer for contract events
}

// IEdgeChallengeManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type IEdgeChallengeManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IEdgeChallengeManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IEdgeChallengeManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IEdgeChallengeManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IEdgeChallengeManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IEdgeChallengeManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IEdgeChallengeManagerSession struct {
	Contract     *IEdgeChallengeManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// IEdgeChallengeManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IEdgeChallengeManagerCallerSession struct {
	Contract *IEdgeChallengeManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// IEdgeChallengeManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IEdgeChallengeManagerTransactorSession struct {
	Contract     *IEdgeChallengeManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// IEdgeChallengeManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type IEdgeChallengeManagerRaw struct {
	Contract *IEdgeChallengeManager // Generic contract binding to access the raw methods on
}

// IEdgeChallengeManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IEdgeChallengeManagerCallerRaw struct {
	Contract *IEdgeChallengeManagerCaller // Generic read-only contract binding to access the raw methods on
}

// IEdgeChallengeManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IEdgeChallengeManagerTransactorRaw struct {
	Contract *IEdgeChallengeManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIEdgeChallengeManager creates a new instance of IEdgeChallengeManager, bound to a specific deployed contract.
func NewIEdgeChallengeManager(address common.Address, backend bind.ContractBackend) (*IEdgeChallengeManager, error) {
	contract, err := bindIEdgeChallengeManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IEdgeChallengeManager{IEdgeChallengeManagerCaller: IEdgeChallengeManagerCaller{contract: contract}, IEdgeChallengeManagerTransactor: IEdgeChallengeManagerTransactor{contract: contract}, IEdgeChallengeManagerFilterer: IEdgeChallengeManagerFilterer{contract: contract}}, nil
}

// NewIEdgeChallengeManagerCaller creates a new read-only instance of IEdgeChallengeManager, bound to a specific deployed contract.
func NewIEdgeChallengeManagerCaller(address common.Address, caller bind.ContractCaller) (*IEdgeChallengeManagerCaller, error) {
	contract, err := bindIEdgeChallengeManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IEdgeChallengeManagerCaller{contract: contract}, nil
}

// NewIEdgeChallengeManagerTransactor creates a new write-only instance of IEdgeChallengeManager, bound to a specific deployed contract.
func NewIEdgeChallengeManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*IEdgeChallengeManagerTransactor, error) {
	contract, err := bindIEdgeChallengeManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IEdgeChallengeManagerTransactor{contract: contract}, nil
}

// NewIEdgeChallengeManagerFilterer creates a new log filterer instance of IEdgeChallengeManager, bound to a specific deployed contract.
func NewIEdgeChallengeManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*IEdgeChallengeManagerFilterer, error) {
	contract, err := bindIEdgeChallengeManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IEdgeChallengeManagerFilterer{contract: contract}, nil
}

// bindIEdgeChallengeManager binds a generic wrapper to an already deployed contract.
func bindIEdgeChallengeManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(IEdgeChallengeManagerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IEdgeChallengeManager *IEdgeChallengeManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IEdgeChallengeManager.Contract.IEdgeChallengeManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IEdgeChallengeManager *IEdgeChallengeManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.IEdgeChallengeManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IEdgeChallengeManager *IEdgeChallengeManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.IEdgeChallengeManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IEdgeChallengeManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.contract.Transact(opts, method, params...)
}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 eId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) EdgeExists(opts *bind.CallOpts, eId [32]byte) (bool, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "edgeExists", eId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 eId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) EdgeExists(eId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.EdgeExists(&_IEdgeChallengeManager.CallOpts, eId)
}

// EdgeExists is a free data retrieval call binding the contract method 0x750e0c0f.
//
// Solidity: function edgeExists(bytes32 eId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) EdgeExists(eId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.EdgeExists(&_IEdgeChallengeManager.CallOpts, eId)
}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 eId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8))
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) GetEdge(opts *bind.CallOpts, eId [32]byte) (ChallengeEdge, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "getEdge", eId)

	if err != nil {
		return *new(ChallengeEdge), err
	}

	out0 := *abi.ConvertType(out[0], new(ChallengeEdge)).(*ChallengeEdge)

	return out0, err

}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 eId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8))
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) GetEdge(eId [32]byte) (ChallengeEdge, error) {
	return _IEdgeChallengeManager.Contract.GetEdge(&_IEdgeChallengeManager.CallOpts, eId)
}

// GetEdge is a free data retrieval call binding the contract method 0xfda2892e.
//
// Solidity: function getEdge(bytes32 eId) view returns((bytes32,bytes32,uint256,bytes32,uint256,bytes32,bytes32,uint256,bytes32,address,uint8,uint8))
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) GetEdge(eId [32]byte) (ChallengeEdge, error) {
	return _IEdgeChallengeManager.Contract.GetEdge(&_IEdgeChallengeManager.CallOpts, eId)
}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 eId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) HasLengthOneRival(opts *bind.CallOpts, eId [32]byte) (bool, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "hasLengthOneRival", eId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 eId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) HasLengthOneRival(eId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.HasLengthOneRival(&_IEdgeChallengeManager.CallOpts, eId)
}

// HasLengthOneRival is a free data retrieval call binding the contract method 0x54b64151.
//
// Solidity: function hasLengthOneRival(bytes32 eId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) HasLengthOneRival(eId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.HasLengthOneRival(&_IEdgeChallengeManager.CallOpts, eId)
}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 eId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) HasRival(opts *bind.CallOpts, eId [32]byte) (bool, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "hasRival", eId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 eId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) HasRival(eId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.HasRival(&_IEdgeChallengeManager.CallOpts, eId)
}

// HasRival is a free data retrieval call binding the contract method 0x908517e9.
//
// Solidity: function hasRival(bytes32 eId) view returns(bool)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) HasRival(eId [32]byte) (bool, error) {
	return _IEdgeChallengeManager.Contract.HasRival(&_IEdgeChallengeManager.CallOpts, eId)
}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 eId) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCaller) TimeUnrivaled(opts *bind.CallOpts, eId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IEdgeChallengeManager.contract.Call(opts, &out, "timeUnrivaled", eId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 eId) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) TimeUnrivaled(eId [32]byte) (*big.Int, error) {
	return _IEdgeChallengeManager.Contract.TimeUnrivaled(&_IEdgeChallengeManager.CallOpts, eId)
}

// TimeUnrivaled is a free data retrieval call binding the contract method 0x3e35f5e8.
//
// Solidity: function timeUnrivaled(bytes32 eId) view returns(uint256)
func (_IEdgeChallengeManager *IEdgeChallengeManagerCallerSession) TimeUnrivaled(eId [32]byte) (*big.Int, error) {
	return _IEdgeChallengeManager.Contract.TimeUnrivaled(&_IEdgeChallengeManager.CallOpts, eId)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 eId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) BisectEdge(opts *bind.TransactOpts, eId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "bisectEdge", eId, prefixHistoryRoot, prefixProof)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 eId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) BisectEdge(eId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.BisectEdge(&_IEdgeChallengeManager.TransactOpts, eId, prefixHistoryRoot, prefixProof)
}

// BisectEdge is a paid mutator transaction binding the contract method 0xc8bc4e43.
//
// Solidity: function bisectEdge(bytes32 eId, bytes32 prefixHistoryRoot, bytes prefixProof) returns(bytes32, bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) BisectEdge(eId [32]byte, prefixHistoryRoot [32]byte, prefixProof []byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.BisectEdge(&_IEdgeChallengeManager.TransactOpts, eId, prefixHistoryRoot, prefixProof)
}

// CalculateEdgeId is a paid mutator transaction binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) CalculateEdgeId(opts *bind.TransactOpts, edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "calculateEdgeId", edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
}

// CalculateEdgeId is a paid mutator transaction binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) CalculateEdgeId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.CalculateEdgeId(&_IEdgeChallengeManager.TransactOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
}

// CalculateEdgeId is a paid mutator transaction binding the contract method 0x004d8efe.
//
// Solidity: function calculateEdgeId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight, bytes32 endHistoryRoot) returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) CalculateEdgeId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int, endHistoryRoot [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.CalculateEdgeId(&_IEdgeChallengeManager.TransactOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight, endHistoryRoot)
}

// CalculateMutualId is a paid mutator transaction binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) CalculateMutualId(opts *bind.TransactOpts, edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "calculateMutualId", edgeType, originId, startHeight, startHistoryRoot, endHeight)
}

// CalculateMutualId is a paid mutator transaction binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) CalculateMutualId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.CalculateMutualId(&_IEdgeChallengeManager.TransactOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight)
}

// CalculateMutualId is a paid mutator transaction binding the contract method 0xc32d8c63.
//
// Solidity: function calculateMutualId(uint8 edgeType, bytes32 originId, uint256 startHeight, bytes32 startHistoryRoot, uint256 endHeight) returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) CalculateMutualId(edgeType uint8, originId [32]byte, startHeight *big.Int, startHistoryRoot [32]byte, endHeight *big.Int) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.CalculateMutualId(&_IEdgeChallengeManager.TransactOpts, edgeType, originId, startHeight, startHistoryRoot, endHeight)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 eId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) ConfirmEdgeByChildren(opts *bind.TransactOpts, eId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "confirmEdgeByChildren", eId)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 eId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) ConfirmEdgeByChildren(eId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByChildren(&_IEdgeChallengeManager.TransactOpts, eId)
}

// ConfirmEdgeByChildren is a paid mutator transaction binding the contract method 0x2eaa0043.
//
// Solidity: function confirmEdgeByChildren(bytes32 eId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) ConfirmEdgeByChildren(eId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByChildren(&_IEdgeChallengeManager.TransactOpts, eId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 eId, bytes32 claimId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) ConfirmEdgeByClaim(opts *bind.TransactOpts, eId [32]byte, claimId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "confirmEdgeByClaim", eId, claimId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 eId, bytes32 claimId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) ConfirmEdgeByClaim(eId [32]byte, claimId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByClaim(&_IEdgeChallengeManager.TransactOpts, eId, claimId)
}

// ConfirmEdgeByClaim is a paid mutator transaction binding the contract method 0x0f73bfad.
//
// Solidity: function confirmEdgeByClaim(bytes32 eId, bytes32 claimId) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) ConfirmEdgeByClaim(eId [32]byte, claimId [32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByClaim(&_IEdgeChallengeManager.TransactOpts, eId, claimId)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0xe5b6c5b8.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, ((uint256,address),uint256,bytes32,bytes) oneStepData, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) ConfirmEdgeByOneStepProof(opts *bind.TransactOpts, edgeId [32]byte, oneStepData OneStepData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "confirmEdgeByOneStepProof", edgeId, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0xe5b6c5b8.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, ((uint256,address),uint256,bytes32,bytes) oneStepData, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) ConfirmEdgeByOneStepProof(edgeId [32]byte, oneStepData OneStepData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByOneStepProof(&_IEdgeChallengeManager.TransactOpts, edgeId, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByOneStepProof is a paid mutator transaction binding the contract method 0xe5b6c5b8.
//
// Solidity: function confirmEdgeByOneStepProof(bytes32 edgeId, ((uint256,address),uint256,bytes32,bytes) oneStepData, bytes32[] beforeHistoryInclusionProof, bytes32[] afterHistoryInclusionProof) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) ConfirmEdgeByOneStepProof(edgeId [32]byte, oneStepData OneStepData, beforeHistoryInclusionProof [][32]byte, afterHistoryInclusionProof [][32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByOneStepProof(&_IEdgeChallengeManager.TransactOpts, edgeId, oneStepData, beforeHistoryInclusionProof, afterHistoryInclusionProof)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x92fd750b.
//
// Solidity: function confirmEdgeByTime(bytes32 eId, bytes32[] ancestorIds) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) ConfirmEdgeByTime(opts *bind.TransactOpts, eId [32]byte, ancestorIds [][32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "confirmEdgeByTime", eId, ancestorIds)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x92fd750b.
//
// Solidity: function confirmEdgeByTime(bytes32 eId, bytes32[] ancestorIds) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) ConfirmEdgeByTime(eId [32]byte, ancestorIds [][32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByTime(&_IEdgeChallengeManager.TransactOpts, eId, ancestorIds)
}

// ConfirmEdgeByTime is a paid mutator transaction binding the contract method 0x92fd750b.
//
// Solidity: function confirmEdgeByTime(bytes32 eId, bytes32[] ancestorIds) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) ConfirmEdgeByTime(eId [32]byte, ancestorIds [][32]byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.ConfirmEdgeByTime(&_IEdgeChallengeManager.TransactOpts, eId, ancestorIds)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x16a0ca0b.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,uint256,bytes32) args, bytes , bytes ) payable returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) CreateLayerZeroEdge(opts *bind.TransactOpts, args CreateEdgeArgs, arg1 []byte, arg2 []byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "createLayerZeroEdge", args, arg1, arg2)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x16a0ca0b.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,uint256,bytes32) args, bytes , bytes ) payable returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) CreateLayerZeroEdge(args CreateEdgeArgs, arg1 []byte, arg2 []byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.CreateLayerZeroEdge(&_IEdgeChallengeManager.TransactOpts, args, arg1, arg2)
}

// CreateLayerZeroEdge is a paid mutator transaction binding the contract method 0x16a0ca0b.
//
// Solidity: function createLayerZeroEdge((uint8,bytes32,uint256,bytes32,uint256,bytes32) args, bytes , bytes ) payable returns(bytes32)
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) CreateLayerZeroEdge(args CreateEdgeArgs, arg1 []byte, arg2 []byte) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.CreateLayerZeroEdge(&_IEdgeChallengeManager.TransactOpts, args, arg1, arg2)
}

// Initialize is a paid mutator transaction binding the contract method 0xc350a1b5.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodSec, address _oneStepProofEntry) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactor) Initialize(opts *bind.TransactOpts, _assertionChain common.Address, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _IEdgeChallengeManager.contract.Transact(opts, "initialize", _assertionChain, _challengePeriodSec, _oneStepProofEntry)
}

// Initialize is a paid mutator transaction binding the contract method 0xc350a1b5.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodSec, address _oneStepProofEntry) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerSession) Initialize(_assertionChain common.Address, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.Initialize(&_IEdgeChallengeManager.TransactOpts, _assertionChain, _challengePeriodSec, _oneStepProofEntry)
}

// Initialize is a paid mutator transaction binding the contract method 0xc350a1b5.
//
// Solidity: function initialize(address _assertionChain, uint256 _challengePeriodSec, address _oneStepProofEntry) returns()
func (_IEdgeChallengeManager *IEdgeChallengeManagerTransactorSession) Initialize(_assertionChain common.Address, _challengePeriodSec *big.Int, _oneStepProofEntry common.Address) (*types.Transaction, error) {
	return _IEdgeChallengeManager.Contract.Initialize(&_IEdgeChallengeManager.TransactOpts, _assertionChain, _challengePeriodSec, _oneStepProofEntry)
}

// IInboxMetaData contains all meta data concerning the IInbox contract.
var IInboxMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"msgCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
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
	parsed, err := abi.JSON(strings.NewReader(IInboxABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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

// MsgCount is a paid mutator transaction binding the contract method 0x8f1a2810.
//
// Solidity: function msgCount() returns(uint256)
func (_IInbox *IInboxTransactor) MsgCount(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IInbox.contract.Transact(opts, "msgCount")
}

// MsgCount is a paid mutator transaction binding the contract method 0x8f1a2810.
//
// Solidity: function msgCount() returns(uint256)
func (_IInbox *IInboxSession) MsgCount() (*types.Transaction, error) {
	return _IInbox.Contract.MsgCount(&_IInbox.TransactOpts)
}

// MsgCount is a paid mutator transaction binding the contract method 0x8f1a2810.
//
// Solidity: function msgCount() returns(uint256)
func (_IInbox *IInboxTransactorSession) MsgCount() (*types.Transaction, error) {
	return _IInbox.Contract.MsgCount(&_IInbox.TransactOpts)
}

// OneStepProofManagerMetaData contains all meta data concerning the OneStepProofManager contract.
var OneStepProofManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"startState\",\"type\":\"bytes32\"}],\"name\":\"createOneStepProof\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"startState\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_winner\",\"type\":\"bytes32\"}],\"name\":\"setWinningClaim\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"winningClaim\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"winningClaims\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061018a806100206000396000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c80632f3069611461005157806335025bde1461007357806373d154e814610098578063a4714dbb146100b8575b600080fd5b61007161005f366004610119565b60009182526020829052604090912055565b005b61008661008136600461013b565b6100d8565b60405190815260200160405180910390f35b6100866100a636600461013b565b60009081526020819052604090205490565b6100866100c636600461013b565b60006020819052908152604090205481565b60405162461bcd60e51b815260206004820152600f60248201526e1393d517d253541311535153951151608a1b604482015260009060640160405180910390fd5b6000806040838503121561012c57600080fd5b50508035926020909101359150565b60006020828403121561014d57600080fd5b503591905056fea26469706673582212204fd0efc4e86d8d267b74c9e0202272abcceb8c8adce94ad9fe519f8893c5293d64736f6c63430008110033",
}

// OneStepProofManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use OneStepProofManagerMetaData.ABI instead.
var OneStepProofManagerABI = OneStepProofManagerMetaData.ABI

// OneStepProofManagerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OneStepProofManagerMetaData.Bin instead.
var OneStepProofManagerBin = OneStepProofManagerMetaData.Bin

// DeployOneStepProofManager deploys a new Ethereum contract, binding an instance of OneStepProofManager to it.
func DeployOneStepProofManager(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OneStepProofManager, error) {
	parsed, err := OneStepProofManagerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OneStepProofManagerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OneStepProofManager{OneStepProofManagerCaller: OneStepProofManagerCaller{contract: contract}, OneStepProofManagerTransactor: OneStepProofManagerTransactor{contract: contract}, OneStepProofManagerFilterer: OneStepProofManagerFilterer{contract: contract}}, nil
}

// OneStepProofManager is an auto generated Go binding around an Ethereum contract.
type OneStepProofManager struct {
	OneStepProofManagerCaller     // Read-only binding to the contract
	OneStepProofManagerTransactor // Write-only binding to the contract
	OneStepProofManagerFilterer   // Log filterer for contract events
}

// OneStepProofManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type OneStepProofManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProofManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OneStepProofManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProofManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OneStepProofManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OneStepProofManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OneStepProofManagerSession struct {
	Contract     *OneStepProofManager // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// OneStepProofManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OneStepProofManagerCallerSession struct {
	Contract *OneStepProofManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// OneStepProofManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OneStepProofManagerTransactorSession struct {
	Contract     *OneStepProofManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// OneStepProofManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type OneStepProofManagerRaw struct {
	Contract *OneStepProofManager // Generic contract binding to access the raw methods on
}

// OneStepProofManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OneStepProofManagerCallerRaw struct {
	Contract *OneStepProofManagerCaller // Generic read-only contract binding to access the raw methods on
}

// OneStepProofManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OneStepProofManagerTransactorRaw struct {
	Contract *OneStepProofManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOneStepProofManager creates a new instance of OneStepProofManager, bound to a specific deployed contract.
func NewOneStepProofManager(address common.Address, backend bind.ContractBackend) (*OneStepProofManager, error) {
	contract, err := bindOneStepProofManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OneStepProofManager{OneStepProofManagerCaller: OneStepProofManagerCaller{contract: contract}, OneStepProofManagerTransactor: OneStepProofManagerTransactor{contract: contract}, OneStepProofManagerFilterer: OneStepProofManagerFilterer{contract: contract}}, nil
}

// NewOneStepProofManagerCaller creates a new read-only instance of OneStepProofManager, bound to a specific deployed contract.
func NewOneStepProofManagerCaller(address common.Address, caller bind.ContractCaller) (*OneStepProofManagerCaller, error) {
	contract, err := bindOneStepProofManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProofManagerCaller{contract: contract}, nil
}

// NewOneStepProofManagerTransactor creates a new write-only instance of OneStepProofManager, bound to a specific deployed contract.
func NewOneStepProofManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*OneStepProofManagerTransactor, error) {
	contract, err := bindOneStepProofManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OneStepProofManagerTransactor{contract: contract}, nil
}

// NewOneStepProofManagerFilterer creates a new log filterer instance of OneStepProofManager, bound to a specific deployed contract.
func NewOneStepProofManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*OneStepProofManagerFilterer, error) {
	contract, err := bindOneStepProofManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OneStepProofManagerFilterer{contract: contract}, nil
}

// bindOneStepProofManager binds a generic wrapper to an already deployed contract.
func bindOneStepProofManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OneStepProofManagerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProofManager *OneStepProofManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProofManager.Contract.OneStepProofManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProofManager *OneStepProofManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProofManager.Contract.OneStepProofManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProofManager *OneStepProofManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProofManager.Contract.OneStepProofManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OneStepProofManager *OneStepProofManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OneStepProofManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OneStepProofManager *OneStepProofManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OneStepProofManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OneStepProofManager *OneStepProofManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OneStepProofManager.Contract.contract.Transact(opts, method, params...)
}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_OneStepProofManager *OneStepProofManagerCaller) WinningClaim(opts *bind.CallOpts, challengeId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _OneStepProofManager.contract.Call(opts, &out, "winningClaim", challengeId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_OneStepProofManager *OneStepProofManagerSession) WinningClaim(challengeId [32]byte) ([32]byte, error) {
	return _OneStepProofManager.Contract.WinningClaim(&_OneStepProofManager.CallOpts, challengeId)
}

// WinningClaim is a free data retrieval call binding the contract method 0x73d154e8.
//
// Solidity: function winningClaim(bytes32 challengeId) view returns(bytes32)
func (_OneStepProofManager *OneStepProofManagerCallerSession) WinningClaim(challengeId [32]byte) ([32]byte, error) {
	return _OneStepProofManager.Contract.WinningClaim(&_OneStepProofManager.CallOpts, challengeId)
}

// WinningClaims is a free data retrieval call binding the contract method 0xa4714dbb.
//
// Solidity: function winningClaims(bytes32 ) view returns(bytes32)
func (_OneStepProofManager *OneStepProofManagerCaller) WinningClaims(opts *bind.CallOpts, arg0 [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _OneStepProofManager.contract.Call(opts, &out, "winningClaims", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// WinningClaims is a free data retrieval call binding the contract method 0xa4714dbb.
//
// Solidity: function winningClaims(bytes32 ) view returns(bytes32)
func (_OneStepProofManager *OneStepProofManagerSession) WinningClaims(arg0 [32]byte) ([32]byte, error) {
	return _OneStepProofManager.Contract.WinningClaims(&_OneStepProofManager.CallOpts, arg0)
}

// WinningClaims is a free data retrieval call binding the contract method 0xa4714dbb.
//
// Solidity: function winningClaims(bytes32 ) view returns(bytes32)
func (_OneStepProofManager *OneStepProofManagerCallerSession) WinningClaims(arg0 [32]byte) ([32]byte, error) {
	return _OneStepProofManager.Contract.WinningClaims(&_OneStepProofManager.CallOpts, arg0)
}

// CreateOneStepProof is a paid mutator transaction binding the contract method 0x35025bde.
//
// Solidity: function createOneStepProof(bytes32 startState) returns(bytes32)
func (_OneStepProofManager *OneStepProofManagerTransactor) CreateOneStepProof(opts *bind.TransactOpts, startState [32]byte) (*types.Transaction, error) {
	return _OneStepProofManager.contract.Transact(opts, "createOneStepProof", startState)
}

// CreateOneStepProof is a paid mutator transaction binding the contract method 0x35025bde.
//
// Solidity: function createOneStepProof(bytes32 startState) returns(bytes32)
func (_OneStepProofManager *OneStepProofManagerSession) CreateOneStepProof(startState [32]byte) (*types.Transaction, error) {
	return _OneStepProofManager.Contract.CreateOneStepProof(&_OneStepProofManager.TransactOpts, startState)
}

// CreateOneStepProof is a paid mutator transaction binding the contract method 0x35025bde.
//
// Solidity: function createOneStepProof(bytes32 startState) returns(bytes32)
func (_OneStepProofManager *OneStepProofManagerTransactorSession) CreateOneStepProof(startState [32]byte) (*types.Transaction, error) {
	return _OneStepProofManager.Contract.CreateOneStepProof(&_OneStepProofManager.TransactOpts, startState)
}

// SetWinningClaim is a paid mutator transaction binding the contract method 0x2f306961.
//
// Solidity: function setWinningClaim(bytes32 startState, bytes32 _winner) returns()
func (_OneStepProofManager *OneStepProofManagerTransactor) SetWinningClaim(opts *bind.TransactOpts, startState [32]byte, _winner [32]byte) (*types.Transaction, error) {
	return _OneStepProofManager.contract.Transact(opts, "setWinningClaim", startState, _winner)
}

// SetWinningClaim is a paid mutator transaction binding the contract method 0x2f306961.
//
// Solidity: function setWinningClaim(bytes32 startState, bytes32 _winner) returns()
func (_OneStepProofManager *OneStepProofManagerSession) SetWinningClaim(startState [32]byte, _winner [32]byte) (*types.Transaction, error) {
	return _OneStepProofManager.Contract.SetWinningClaim(&_OneStepProofManager.TransactOpts, startState, _winner)
}

// SetWinningClaim is a paid mutator transaction binding the contract method 0x2f306961.
//
// Solidity: function setWinningClaim(bytes32 startState, bytes32 _winner) returns()
func (_OneStepProofManager *OneStepProofManagerTransactorSession) SetWinningClaim(startState [32]byte, _winner [32]byte) (*types.Transaction, error) {
	return _OneStepProofManager.Contract.SetWinningClaim(&_OneStepProofManager.TransactOpts, startState, _winner)
}
