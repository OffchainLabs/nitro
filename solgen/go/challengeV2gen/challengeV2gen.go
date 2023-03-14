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
	Bin: "0x60a060405268056bc75e2d6310000060805234801561001d57600080fd5b5060405161162c38038061162c83398101604081905261003c916101f9565b6002818155604080516101208101825260008082526020808301828152938301828152606084018381526080850184815260a086018a815260c08701868152600160e089018181526101008a018990528880529681905288517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4990815599517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4a5594517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4b805491151560ff1992831617905593517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4c5591517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4d55517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4e55517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb4f5591517fa6eef7e35abe7026729641147f7915573c7e97b47efa546f5f6e3230263bcb50805494979596959194909316919084908111156101de576101de61021d565b02179055506101008201518160080155905050505050610233565b6000806040838503121561020c57600080fd5b505080516020909101519092909150565b634e487b7160e01b600052602160045260246000fd5b6080516113d76102556000396000818161025f01526108b801526113d76000f3fe6080604052600436106101045760003560e01c80636894bdd5116100a05780639ca565d4116100645780639ca565d41461032e578063d60715b51461034e578063f9bce634146103cf578063fb601294146103ef578063ff8aef871461040557600080fd5b80636894bdd51461028157806375dc6098146102a15780637cfd5ab9146102c157806388302884146102e1578063896efbf21461030e57600080fd5b8063295dfd32146101095780632fefa18a14610148578063308362281461017b57806343ed6ad9146101ab57806349635f9a146101cb5780635625c360146101eb5780635a4038f5146102135780635a627dbc1461024557806360c7dc471461024d575b600080fd5b34801561011557600080fd5b506101466101243660046111a2565b600080546001600160a01b0319166001600160a01b0392909216919091179055565b005b34801561015457600080fd5b506101686101633660046111d2565b610425565b6040519081526020015b60405180910390f35b34801561018757600080fd5b5061019b6101963660046111d2565b610473565b6040519015158152602001610172565b3480156101b757600080fd5b506101686101c63660046111d2565b6104ba565b3480156101d757600080fd5b506101466101e63660046111eb565b6104fe565b3480156101f757600080fd5b506000546040516001600160a01b039091168152602001610172565b34801561021f57600080fd5b5061019b61022e3660046111d2565b600090815260016020526040902060050154151590565b6101466108b6565b34801561025957600080fd5b506101687f000000000000000000000000000000000000000000000000000000000000000081565b34801561028d57600080fd5b5061014661029c3660046111d2565b610927565b3480156102ad57600080fd5b506101466102bc3660046111d2565b610b38565b3480156102cd57600080fd5b506101686102dc3660046111d2565b610d43565b3480156102ed57600080fd5b506103016102fc3660046111d2565b610d87565b604051610172919061124f565b34801561031a57600080fd5b506101686103293660046111d2565b610e7b565b34801561033a57600080fd5b506101686103493660046111d2565b610ebf565b34801561035a57600080fd5b506103ba6103693660046111d2565b6001602081905260009182526040909120805491810154600282015460038301546004840154600585015460068601546007870154600890970154959660ff95861696949593949293919291169089565b604051610172999897969594939291906112bc565b3480156103db57600080fd5b506101686103ea3660046111d2565b610f00565b3480156103fb57600080fd5b5061016860025481565b34801561041157600080fd5b506101466104203660046111d2565b610f44565b60008181526001602052604081206005015461045c5760405162461bcd60e51b815260040161045390611310565b60405180910390fd5b506000908152600160208190526040909120015490565b6000818152600160205260408120600501546104a15760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090206002015460ff1690565b6000818152600160205260408120600501546104e85760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090206004015490565b60408051602081018590529081018390526060810182905260009060800160405160208183030381529060405280519060200120905061054f81600090815260016020526040902060050154151590565b156105975760405162461bcd60e51b8152602060048201526018602482015277417373657274696f6e20616c72656164792065786973747360401b6044820152606401610453565b6000828152600160205260409020600501546105ff5760405162461bcd60e51b815260206004820152602160248201527f50726576696f757320617373657274696f6e20646f6573206e6f7420657869736044820152601d60fa1b6064820152608401610453565b600260008281526001602052604080822054825290206007015460ff16600281111561062d5761062d611217565b0361067a5760405162461bcd60e51b815260206004820152601b60248201527f50726576696f757320617373657274696f6e2072656a656374656400000000006044820152606401610453565b6000818152600160205260408082205482529020839060060154106106ed5760405162461bcd60e51b815260206004820152602360248201527f486569676874206e6f742067726561746572207468616e20707265646563657360448201526239b7b960e91b6064820152608401610453565b600082815260016020526040902060040154151580610720576000838152600160205260409020426004909101556107c1565b6002546000838152600160205260408082205482529020600401546107459190611358565b42106107935760405162461bcd60e51b815260206004820152601a60248201527f546f6f206c61746520746f20637265617465207369626c696e670000000000006044820152606401610453565b60008381526001602052604081206003015490036107c1576000838152600160205260409020426003909101555b6040518061012001604052808481526020016000801b815260200182151515815260200160008152602001600081526020018681526020018581526020016000600281111561081257610812611217565b8152600060209182018190528481526001808352604091829020845181559284015183820155908301516002808401805492151560ff19938416179055606085015160038501556080850151600485015560a0850151600585015560c0850151600685015560e0850151600785018054919490939190911691849081111561089c5761089c611217565b021790555061010082015181600801559050505050505050565b7f000000000000000000000000000000000000000000000000000000000000000034146109255760405162461bcd60e51b815260206004820152601a60248201527f436f7272656374207374616b65206e6f742070726f76696465640000000000006044820152606401610453565b565b6000818152600160205260409020600501546109555760405162461bcd60e51b815260040161045390611310565b600160008281526001602052604080822054825290206007015460ff16600281111561098357610983611217565b146109d05760405162461bcd60e51b815260206004820181905260248201527f50726576696f757320617373657274696f6e206e6f7420636f6e6669726d65646044820152606401610453565b600081815260016020526040808220548252902060030154158015610a185750600254600082815260016020526040808220548252902060040154610a159190611358565b42115b15610a42576000818152600160208190526040909120600701805460ff191682805b021790555050565b60008181526001602081905260408083205483528220015490819003610a7e57604051631895e8f560e21b815260048101839052602401610453565b60008054604051630e7a2a9d60e31b8152600481018490526001600160a01b03909116906373d154e890602401602060405180830381865afa158015610ac8573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610aec9190611371565b9050828114610b1157604051632158b7ff60e11b815260048101849052602401610453565b6000838152600160208190526040909120600701805460ff191682805b0217905550505050565b600081815260016020526040902060050154610b665760405162461bcd60e51b815260040161045390611310565b60008181526001602052604081206007015460ff166002811115610b8c57610b8c611217565b14610bd45760405162461bcd60e51b8152602060048201526018602482015277417373657274696f6e206973206e6f742070656e64696e6760401b6044820152606401610453565b600260008281526001602052604080822054825290206007015460ff166002811115610c0257610c02611217565b03610c2d576000818152600160208190526040909120600701805460029260ff199091169083610a3a565b60008181526001602081905260408083205483528220015490819003610c6957604051632158b7ff60e11b815260048101839052602401610453565b60008054604051630e7a2a9d60e31b8152600481018490526001600160a01b03909116906373d154e890602401602060405180830381865afa158015610cb3573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610cd79190611371565b905080610cfa57604051632158b7ff60e11b815260048101849052602401610453565b828103610d1d57604051632158b7ff60e11b815260048101849052602401610453565b6000838152600160208190526040909120600701805460029260ff199091169083610b2e565b600081815260016020526040812060050154610d715760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090206008015490565b6040805161012081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e0810182905261010081019190915260008281526001602081815260409283902083516101208101855281548152928101549183019190915260028082015460ff9081161515948401949094526003820154606084015260048201546080840152600582015460a0840152600682015460c084015260078201549293919260e08501921690811115610e5557610e55611217565b6002811115610e6657610e66611217565b81526020016008820154815250509050919050565b600081815260016020526040812060050154610ea95760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090206006015490565b600081815260016020526040812060050154610eed5760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090205490565b600081815260016020526040812060050154610f2e5760405162461bcd60e51b815260040161045390611310565b5060009081526001602052604090206005015490565b600081815260016020526040902060050154610f725760405162461bcd60e51b815260040161045390611310565b600260008281526001602052604090206007015460ff166002811115610f9a57610f9a611217565b03610fe75760405162461bcd60e51b815260206004820152601a60248201527f417373657274696f6e20616c72656164792072656a65637465640000000000006044820152606401610453565b60008181526001602081905260409091200154156110435760405162461bcd60e51b815260206004820152601960248201527810da185b1b195b99d948185b1c9958591e4818dc99585d1959603a1b6044820152606401610453565b60008181526001602052604081206003015490036110ad5760405162461bcd60e51b815260206004820152602160248201527f4174206c656173742074776f206368696c6472656e206e6f74206372656174656044820152601960fa1b6064820152608401610453565b600280546110ba9161138a565b6000828152600160205260409020600401546110d69190611358565b421061111c5760405162461bcd60e51b8152602060048201526015602482015274546f6f206c61746520746f206368616c6c656e676560581b6044820152606401610453565b60005460405163f696dc5560e01b8152600481018390526001600160a01b039091169063f696dc55906024016020604051808303816000875af1158015611167573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061118b9190611371565b600091825260016020819052604090922090910155565b6000602082840312156111b457600080fd5b81356001600160a01b03811681146111cb57600080fd5b9392505050565b6000602082840312156111e457600080fd5b5035919050565b60008060006060848603121561120057600080fd5b505081359360208301359350604090920135919050565b634e487b7160e01b600052602160045260246000fd5b6003811061124b57634e487b7160e01b600052602160045260246000fd5b9052565b6000610120820190508251825260208301516020830152604083015115156040830152606083015160608301526080830151608083015260a083015160a083015260c083015160c083015260e08301516112ac60e084018261122d565b5061010092830151919092015290565b6000610120820190508a825289602083015288151560408301528760608301528660808301528560a08301528460c08301526112fb60e083018561122d565b826101008301529a9950505050505050505050565b602080825260189082015277105cdcd95c9d1a5bdb88191bd95cc81b9bdd08195e1a5cdd60421b604082015260600190565b634e487b7160e01b600052601160045260246000fd5b8082018082111561136b5761136b611342565b92915050565b60006020828403121561138357600080fd5b5051919050565b808202811582820484141761136b5761136b61134256fea264697066735822122095c68ab8a6f7a1bf53e06ca3e26ac5f562bbc05d92e420c5fa629b76a6a3206864736f6c63430008110033",
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
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_miniStakeValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriodSec\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"actualLength\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxProofLength\",\"type\":\"uint256\"}],\"name\":\"MerkleProofTooLong\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"fromId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"toId\",\"type\":\"bytes32\"}],\"name\":\"Bisected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"ChallengeCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"fromId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"toId\",\"type\":\"bytes32\"}],\"name\":\"Merged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"vertexId\",\"type\":\"bytes32\"}],\"name\":\"VertexAdded\",\"type\":\"event\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"historyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"firstState\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"firstStatehistoryProof\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32\",\"name\":\"lastState\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"lastStatehistoryProof\",\"type\":\"bytes32[]\"}],\"internalType\":\"structAddLeafArgs\",\"name\":\"leafData\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"proof1\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof2\",\"type\":\"bytes\"}],\"name\":\"addLeaf\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"assertionChain\",\"outputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"prefixHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"bisect\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"},{\"internalType\":\"enumChallengeType\",\"name\":\"typ\",\"type\":\"uint8\"}],\"name\":\"calculateChallengeId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"commitmentMerkle\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"commitmentHeight\",\"type\":\"uint256\"}],\"name\":\"calculateChallengeVertexId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"challengeExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"challengePeriodSec\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"challenges\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"winningClaim\",\"type\":\"bytes32\"},{\"internalType\":\"enumChallengeType\",\"name\":\"challengeType\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"challenger\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"childrenAreAtOneStepFork\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"confirmForPsTimer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"confirmForSucessionChallengeWin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"crazyStuff\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"assertionId\",\"type\":\"bytes32\"}],\"name\":\"createChallenge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"createSubChallenge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"item\",\"type\":\"bytes\"}],\"name\":\"decodeLens\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"winnerVId\",\"type\":\"bytes32\"},{\"components\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxInboxMessagesRead\",\"type\":\"uint256\"},{\"internalType\":\"contractIBridge\",\"name\":\"bridge\",\"type\":\"address\"}],\"internalType\":\"structExecutionContext\",\"name\":\"execCtx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"machineStep\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"beforeHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"internalType\":\"structOneStepData\",\"name\":\"oneStepData\",\"type\":\"tuple\"},{\"internalType\":\"bytes32[]\",\"name\":\"beforeHistoryInclusionProof\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"afterHistoryInclusionProof\",\"type\":\"bytes32[]\"}],\"name\":\"executeOneStep\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"getChallenge\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"rootId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"winningClaim\",\"type\":\"bytes32\"},{\"internalType\":\"enumChallengeType\",\"name\":\"challengeType\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"challenger\",\"type\":\"address\"}],\"internalType\":\"structChallenge\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"getCurrentPsTimer\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"getVertex\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"historyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"successionChallenge\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"predecessorId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"enumVertexStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"psId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"psLastUpdatedTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"flushedPsTimeSec\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"lowestHeightSuccessorId\",\"type\":\"bytes32\"}],\"internalType\":\"structChallengeVertex\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"hasConfirmedSibling\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIAssertionChain\",\"name\":\"_assertionChain\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_miniStakeValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_challengePeriodSec\",\"type\":\"uint256\"},{\"internalType\":\"contractIOneStepProofEntry\",\"name\":\"_oneStepProofEntry\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"isPresumptiveSuccessor\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"prefixHistoryRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"prefixProof\",\"type\":\"bytes\"}],\"name\":\"merge\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"miniStakeValue\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vId\",\"type\":\"bytes32\"}],\"name\":\"vertexExists\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"vertices\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"historyRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"successionChallenge\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"predecessorId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"claimId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"staker\",\"type\":\"address\"},{\"internalType\":\"enumVertexStatus\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"psId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"psLastUpdatedTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"flushedPsTimeSec\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"lowestHeightSuccessorId\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"challengeId\",\"type\":\"bytes32\"}],\"name\":\"winningClaim\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b5060405162005acf38038062005acf8339810160408190526200003491620000ec565b62000042848484846200004c565b505050506200013d565b6002546001600160a01b031615620000995760405162461bcd60e51b815260206004820152600c60248201526b1053149150511657d253925560a21b604482015260640160405180910390fd5b600280546001600160a01b039586166001600160a01b03199182161790915560049390935560059190915560038054919093169116179055565b6001600160a01b0381168114620000e957600080fd5b50565b600080600080608085870312156200010357600080fd5b84516200011081620000d3565b80945050602085015192506040850151915060608501516200013281620000d3565b939692955090935050565b615982806200014d6000396000f3fe60806040526004361061016c5760003560e01c80637a4d47dc116100cc578063bd6232511161007a578063bd6232511461045c578063c1e69b661461047c578063d1bac9a4146104da578063e41b5058146104fa578063e95756d51461051a578063f4f81db21461053c578063f696dc55146105d957600080fd5b80637a4d47dc1461036757806386f048ed146103875780638ac04349146103b457806398b67d59146103d45780639e3d87cd146103f45780639e7cee5414610414578063b4f172011461042757600080fd5b806348dd29241161012957806348dd2924146102725780634a658788146102aa578063597e1e0b146102ca57806359c69996146102ea578063654f0dc2146103005780636b0b25921461031657806373d154e81461033657600080fd5b80631421ba3d1461017157806316ef5534146101a45780631b7bbecb146101c45780631d5618ac14610203578063359076cf14610225578063458d2bf114610245575b600080fd5b34801561017d57600080fd5b5061019161018c366004614bb1565b6105f9565b6040519081526020015b60405180910390f35b3480156101b057600080fd5b506101916101bf366004614c56565b61063e565b3480156101d057600080fd5b506101f36101df366004614c8a565b600090815260016020526040902054151590565b604051901515815260200161019b565b34801561020f57600080fd5b5061022361021e366004614c8a565b610653565b005b34801561023157600080fd5b50610191610240366004614d12565b61066d565b34801561025157600080fd5b50610265610260366004614c8a565b61073a565b60405161019b9190614de3565b34801561027e57600080fd5b50600254610292906001600160a01b031681565b6040516001600160a01b03909116815260200161019b565b3480156102b657600080fd5b506101916102c5366004614e23565b610837565b3480156102d657600080fd5b506101916102e5366004614d12565b61084c565b3480156102f657600080fd5b5061019160045481565b34801561030c57600080fd5b5061019160055481565b34801561032257600080fd5b506101f3610331366004614c8a565b6108e4565b34801561034257600080fd5b50610191610351366004614c8a565b6000908152600160208190526040909120015490565b34801561037357600080fd5b506101f3610382366004614c8a565b6108fd565b34801561039357600080fd5b506103a76103a2366004614c8a565b610911565b60405161019b9190614e5f565b3480156103c057600080fd5b506101916103cf366004614c8a565b610a13565b3480156103e057600080fd5b506101f36103ef366004614c8a565b610a1f565b34801561040057600080fd5b5061022361040f366004614f14565b610bee565b610191610422366004614f9f565b610c70565b34801561043357600080fd5b5061044761044236600461503c565b610f9a565b6040805192835260208301919091520161019b565b34801561046857600080fd5b50610191610477366004614c8a565b610fbe565b34801561048857600080fd5b506104ca610497366004614c8a565b600160208190526000918252604090912080549181015460029091015460ff81169061010090046001600160a01b031684565b60405161019b949392919061507d565b3480156104e657600080fd5b506102236104f5366004614c8a565b61119f565b34801561050657600080fd5b506101f3610515366004614c8a565b6111ac565b34801561052657600080fd5b5061052f61122c565b60405161019b91906150b2565b34801561054857600080fd5b506105c1610557366004614c8a565b600060208190529081526040902080546001820154600283015460038401546004850154600586015460068701546007880154600889015460098a0154600a909a01549899979896979596949593946001600160a01b03841694600160a01b90940460ff1693908c565b60405161019b9c9b9a99989796959493929190615100565b3480156105e557600080fd5b506101916105f4366004614c8a565b611371565b600354600090819061061e9082906001906001600160a01b03168b8b8b8b8b8b6115eb565b600090815260016020819052604090912001979097559695505050505050565b600061064a8383611901565b90505b92915050565b610661600082600554611934565b61066a81611a21565b50565b600080600061068160006001888888611a7c565b6000888152602081905260408120600401549294509092506106a38189611b01565b600089815260208190526040812054919250906106c290898685611c0c565b90506106de81846005546000611cd8909392919063ffffffff16565b506005546106f29060009087908c90611e14565b604080518a8152602081018790527f69d5465c81edf7aaaf2e5c6c8829500df87d84c87f8d5b1221b59eaeaca70d27910160405180910390a1509293505050505b9392505050565b6040805160808101825260008082526020820181905291810182905260608101919091526000828152600160205260409020546107b95760405162461bcd60e51b815260206004820152601860248201527710da185b1b195b99d948191bd95cc81b9bdd08195e1a5cdd60421b60448201526064015b60405180910390fd5b60008281526001602081815260409283902083516080810185528154815292810154918301919091526002810154919290919083019060ff16600381111561080357610803614db9565b600381111561081457610814614db9565b81526002919091015461010090046001600160a01b031660209091015292915050565b60006108448484846122b8565b949350505050565b60008061085e600060018787876122ef565b50905061087b81866005546000611e14909392919063ffffffff16565b60008181526020819052604080822060040154878352908220600901546108a39291906123d2565b60408051868152602081018390527f72b50597145599e4288d411331c925b40b33b0fa3cccadc1f57d2a1ab973553a910160405180910390a1949350505050565b600081815260208190526040812060010154151561064d565b6000610909818361251f565b506001919050565b610919614b0c565b6000828152602081905260409020600101546109475760405162461bcd60e51b81526004016107b090615174565b6000828152602081815260409182902082516101808101845281548152600180830154938201939093526002820154938101939093526003810154606084015260048101546080840152600581015460a084015260068101546001600160a01b03811660c0850152909160e0840191600160a01b900460ff16908111156109d0576109d0614db9565b60018111156109e1576109e1614db9565b8152600782015460208201526008820154604082015260098201546060820152600a9091015460809091015292915050565b600061064d8183611b01565b600081815260208190526040812060010154610a4d5760405162461bcd60e51b81526004016107b090615174565b60008281526020819052604080822060040154808352912060010154610a855760405162461bcd60e51b81526004016107b0906151a3565b6000818152602081905260409020600301548015610b6157600081815260016020819052604090912001548015610b5f57600081815260208190526040902060010154610b145760405162461bcd60e51b815260206004820152601c60248201527f57696e6e696e6720636c61696d20646f6573206e6f742065786973740000000060448201526064016107b0565b848103610b2657506000949350505050565b6001600082815260208190526040902060060154600160a01b900460ff166001811115610b5557610b55614db9565b1495945050505050565b505b6000828152602081905260409020600701548015610be357600081815260208190526040902060010154610b145760405162461bcd60e51b8152602060048201526024808201527f50726573756d707469766520737563636573736f7220646f6573206e6f7420656044820152631e1a5cdd60e21b60648201526084016107b0565b506000949350505050565b6002546001600160a01b031615610c365760405162461bcd60e51b815260206004820152600c60248201526b1053149150511657d253925560a21b60448201526064016107b0565b600280546001600160a01b039586166001600160a01b03199182161790915560049390935560059190915560038054919093169116179055565b600080863560009081526001602052604090206002015460ff166003811115610c9b57610c9b614db9565b03610d99576000610d57600060016040518060a00160405280600454815260200160055481526020018b610cce90615268565b81526020018a8a8080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250505090825250604080516020601f8b0181900481028201810190925289815291810191908a908a908190840183828082843760009201919091525050509152506002546001600160a01b031661273f565b90507f4383ba11a7cd16be5880c5f674b93be38b3b1fcafd7a7b06151998fa2a67534981604051610d8a91815260200190565b60405180910390a19050610f91565b6001863560009081526001602052604090206002015460ff166003811115610dc357610dc3614db9565b03610e73576000610d57600060016040518060a00160405280600454815260200160055481526020018b610df690615268565b81526020018a8a8080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250505090825250604080516020601f8b0181900481028201810190925289815291810191908a908a90819084018382808284376000920191909152505050915250612a51565b6002863560009081526001602052604090206002015460ff166003811115610e9d57610e9d614db9565b03610f4d576000610d57600060016040518060a00160405280600454815260200160055481526020018b610ed090615268565b81526020018a8a8080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250505090825250604080516020601f8b0181900481028201810190925289815291810191908a908a90819084018382808284376000920191909152505050915250612bbc565b60405162461bcd60e51b8152602060048201526019602482015278556e6578706563746564206368616c6c656e6765207479706560381b60448201526064016107b0565b95945050505050565b6000808080610fab85870187615311565b90519051909450925050505b9250929050565b6000806000610fd36000600186600554612d0b565b600086815260208190526040812060010154929450909250610ff6848388612e5f565b9050600061100382612f0a565b600081815260208181526040918290208551815590850151600180830191909155918501516002820155606085015160038201556080850151600482015560a0850151600582015560c08501516006820180546001600160a01b039092166001600160a01b031983168117825560e08801519596508795939491926001600160a81b0319161790600160a01b9084908111156110a1576110a1614db9565b021790555061010082015160078201556101208201516008820155610140820151600982015561016090910151600a9091015560408051608081018252828152600060208201529081018560038111156110fd576110fd614db9565b81523360209182015260008781526001808352604091829020845181559284015183820155908301516002830180549192909160ff19169083600381111561114757611147614db9565b021790555060609190910151600290910180546001600160a01b0390921661010002610100600160a81b031990921691909117905560008781526020819052604090206111949086612f23565b509295945050505050565b6106616000600183612f72565b6000818152602081905260408120600101546111da5760405162461bcd60e51b81526004016107b090615174565b600082815260208190526040808220600401548083529120600101546112125760405162461bcd60e51b81526004016107b090615374565b600090815260208190526040902060070154909114919050565b6040805160028082526060808301845292600092919060208301908036833750506040805160018082528183019092529293506000929150602080830190803683370190505060408051600160208201529192500160405160208183030381529060405280519060200120826000815181106112aa576112aa6153b5565b60200260200101818152505060026040516020016112ca91815260200190565b60405160208183030381529060405280519060200120826001815181106112f3576112f36153b5565b602002602001018181525050600160405160200161131391815260200190565b604051602081830303815290604052805190602001208160008151811061133c5761133c6153b5565b602002602001018181525050818160405160200161135b929190615406565b6040516020818303038152906040529250505090565b60025460009081906113909060019085906001600160a01b031661303d565b600254604051633e6f398d60e21b8152600481018690529192506000916001600160a01b039091169063f9bce63490602401602060405180830381865afa1580156113df573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611403919061542b565b9050600061143a838360405160200161141e91815260200190565b6040516020818303038152906040528051906020012087612e5f565b9050600061144782612f0a565b600081815260208181526040918290208551815590850151600180830191909155918501516002820155606085015160038201556080850151600482015560a0850151600582015560c08501516006820180546001600160a01b039092166001600160a01b031983168117825560e08801519596508795939491926001600160a81b0319161790600160a01b9084908111156114e5576114e5614db9565b021790555061010082015160078201556101208201516008820155610140820151600982015561016090910151600a90910155604080516080810182528281526000602080830182815283850183815233606086015289845260019283905294909220835181559151828201559251600282018054939492939192909160ff19169083600381111561157957611579614db9565b021790555060609190910151600290910180546001600160a01b0390921661010002610100600160a81b03199092169190911790556040518481527f867c977ac47adb20fcc4fb6b981269b44d23560057a29eed03cd5afb81750b349060200160405180910390a15091949350505050565b600086815260208a905260408120600101546116195760405162461bcd60e51b81526004016107b090615174565b600087815260208b90526040808220600401548083529120600101546116515760405162461bcd60e51b81526004016107b0906151a3565b600081815260208c90526040812060030154908190036116835760405162461bcd60e51b81526004016107b090615444565b6003600082815260208d9052604090206002015460ff1660038111156116ab576116ab614db9565b1461170d5760405162461bcd60e51b815260206004820152602c60248201527f4368616c6c656e6765206973206e6f74206174206f6e6520737465702065786560448201526b18dd5d1a5bdb881c1bda5b9d60a21b60648201526084016107b0565b6117698c60008481526020019081526020016000206001015489606001358a604001358a8a808060200260200160405190810160405280939291908181526020018383602002808284376000920191909152506130e692505050565b6117b55760405162461bcd60e51b815260206004820152601b60248201527f4265666f7265207374617465206e6f7420696e20686973746f7279000000000060448201526064016107b0565b60006001600160a01b038b16635d3adcfb8a604081013560608201356117de6080840184615487565b6040518663ffffffff1660e01b81526004016117fe9594939291906154cd565b602060405180830381865afa15801561181b573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061183f919061542b565b90506118a58d60008c815260200190815260200160002060010154828b60400135600161186c919061554a565b8989808060200260200160405190810160405280939291908181526020018383602002808284376000920191909152506130e692505050565b6118f15760405162461bcd60e51b815260206004820152601a60248201527f4166746572207374617465206e6f7420696e20686973746f727900000000000060448201526064016107b0565b509b9a5050505050505050505050565b6000828260405160200161191692919061555d565b60405160208183030381529060405280519060200120905092915050565b61193e8383613128565b600082815260208490526040808220600401548252902060030154156119b25760405162461bcd60e51b815260206004820152602360248201527f53756363657373696f6e206368616c6c656e676520616c7265616479206f70656044820152621b995960ea1b60648201526084016107b0565b806119bd8484611b01565b11611a1c5760405162461bcd60e51b815260206004820152602960248201527f507354696d6572206e6f742067726561746572207468616e206368616c6c656e60448201526819d9481c195c9a5bd960ba1b60648201526084016107b0565b505050565b6000818152602081905260409020611a3890613279565b6000818152602081905260409020805490611a5290613339565b15611a785760008281526020818152604080832060050154848452600192839052922001555b5050565b600080600080611a8f89898989896133c1565b600082815260208c90526040902060010154919350915015611af35760405162461bcd60e51b815260206004820152601f60248201527f426973656374696f6e2076657274657820616c7265616479206578697374730060448201526064016107b0565b909890975095505050505050565b600081815260208390526040812060010154611b6a5760405162461bcd60e51b815260206004820152602260248201527f56657274657820646f6573206e6f7420657869737420666f722070732074696d60448201526132b960f11b60648201526084016107b0565b60008281526020849052604080822060040154808352912060010154611ba25760405162461bcd60e51b81526004016107b090615374565b600081815260208590526040902060070154839003611bf35760008381526020859052604080822060090154838352912060080154611be19042615586565b611beb919061554a565b91505061064d565b505060008181526020839052604090206009015461064d565b611c14614b0c565b6000859003611c355760405162461bcd60e51b81526004016107b090615599565b6000849003611c565760405162461bcd60e51b81526004016107b0906155c4565b82600003611c765760405162461bcd60e51b81526004016107b0906155ef565b5060408051610180810182529485526020850193909352918301526000606083018190526080830181905260a0830181905260c0830181905260e083018190526101008301819052610120830181905261014083019190915261016082015290565b600080611ce485612f0a565b60008181526020889052604090206001015490915015611d3e5760405162461bcd60e51b815260206004820152601560248201527456657274657820616c72656164792065786973747360581b60448201526064016107b0565b600081815260208781526040918290208751815590870151600180830191909155918701516002820155606087015160038201556080870151600482015560a0870151600582015560c08701516006820180546001600160a01b039092166001600160a01b031983168117825560e08a01518a9590936001600160a81b03191690911790600160a01b908490811115611dd957611dd9614db9565b021790555061010082015160078201556101208201516008820155610140820151600982015561016090910151600a90910155610f91868583865b600083815260208590526040902060010154611e725760405162461bcd60e51b815260206004820152601b60248201527f53746172742076657274657820646f6573206e6f74206578697374000000000060448201526064016107b0565b6000838152602085905260409020611e8990613339565b15611ee25760405162461bcd60e51b8152602060048201526024808201527f43616e6e6f7420636f6e6e656374206120737563636573736f7220746f2061206044820152633632b0b360e11b60648201526084016107b0565b600082815260208590526040902060010154611f3c5760405162461bcd60e51b8152602060048201526019602482015278115b99081d995c9d195e08191bd95cc81b9bdd08195e1a5cdd603a1b60448201526064016107b0565b600082815260208590526040902060040154839003611f9d5760405162461bcd60e51b815260206004820152601a60248201527f566572746963657320616c726561647920636f6e6e656374656400000000000060448201526064016107b0565b6000828152602085905260408082206002908101548684529190922090910154106120195760405162461bcd60e51b815260206004820152602660248201527f537461727420686569676874206e6f74206c6f776572207468616e20656e64206044820152651a195a59da1d60d21b60648201526084016107b0565b600082815260208590526040808220548583529120541461209a5760405162461bcd60e51b815260206004820152603560248201527f5072656465636573736f7220616e6420737563636573736f722061726520696e60448201527420646966666572656e74206368616c6c656e67657360581b60648201526084016107b0565b60008281526020859052604090206120b290846135a7565b6000838152602085905260408120600a015490036120f3576120d6848460006123d2565b60008381526020859052604090206120ee9083613673565b6122b2565b600082815260208590526040808220600290810154868452828420600a015484529190922090910154808210156121e75761212f86868561374e565b156121bc5760405162461bcd60e51b815260206004820152605160248201527f5374617274207665727465782068617320707320776974682074696d6572206760448201527f726561746572207468616e206368616c6c656e676520706572696f642c2063616064820152706e6e6f7420736574206c6f77657220707360781b608482015260a4016107b0565b6121c8868660006123d2565b60008581526020879052604090206121e09085613673565b50506122b2565b8082036122af576121f986868561374e565b1561228c5760405162461bcd60e51b815260206004820152605760248201527f5374617274207665727465782068617320707320776974682074696d6572206760448201527f726561746572207468616e206368616c6c656e676520706572696f642c2063616064820152766e6e6f74207365742073616d652068656967687420707360481b608482015260a4016107b0565b612298868660006123d2565b60008581526020879052604081206121e091613673565b50505b50505050565b6040805160208082019590955280820193909352606080840192909252805180840390920182526080909201909152805191012090565b60008060008061230289898989896133c1565b600082815260208c9052604090206001015491935091506123755760405162461bcd60e51b815260206004820152602760248201527f426973656374696f6e2076657274657820646f6573206e6f7420616c726561646044820152661e48195e1a5cdd60ca1b60648201526084016107b0565b600082815260208a90526040902061238c90613339565b15611af35760405162461bcd60e51b815260206004820152601660248201527521b0b73737ba1036b2b933b2903a379030903632b0b360511b60448201526064016107b0565b6000828152602084905260409020600101546124005760405162461bcd60e51b81526004016107b090615174565b600082815260208490526040902061241790613339565b156124795760405162461bcd60e51b815260206004820152602c60248201527f43616e6e6f7420666c757368206c6561662061732069742077696c6c206e657660448201526b65722068617665206120505360a01b60648201526084016107b0565b60008281526020849052604090206007015415612507576000828152602084905260408120600801546124ac9042615586565b600084815260208690526040808220600701548252812060090154919250906124d690839061554a565b9050828110156124e35750815b600084815260208690526040808220600701548252902061250490826137c5565b50505b6000828152602084905260409020611a1c9042613850565b6000818152602083905260409020600101546125895760405162461bcd60e51b8152602060048201526024808201527f466f726b2063616e6469646174652076657274657820646f6573206e6f7420656044820152631e1a5cdd60e21b60648201526084016107b0565b60008181526020839052604090206125a090613339565b156125f85760405162461bcd60e51b815260206004820152602260248201527f4c6561662063616e206e65766572206265206120666f726b2063616e64696461604482015261746560f01b60648201526084016107b0565b600081815260208390526040808220600a0154825290206001015461264f5760405162461bcd60e51b815260206004820152600d60248201526c4e6f20737563636573736f727360981b60448201526064016107b0565b600081815260208390526040808220600a810154835290822060029081015492849052015461267e9082615586565b6001146126e45760405162461bcd60e51b815260206004820152602e60248201527f4c6f7765737420686569676874206e6f74206f6e652061626f7665207468652060448201526d18dd5c9c995b9d081a195a59da1d60921b60648201526084016107b0565b60008281526020849052604090206007015415611a1c5760405162461bcd60e51b81526020600482015260196024820152782430b990383932b9bab6b83a34bb329039bab1b1b2b9b9b7b960391b60448201526064016107b0565b600080826001600160a01b0316639ca565d48560400151602001516040518263ffffffff1660e01b815260040161277891815260200190565b602060405180830381865afa158015612795573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906127b9919061542b565b6040858101515190516317f7d0c560e11b815260048101839052919250906001600160a01b03851690632fefa18a90602401602060405180830381865afa158015612808573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061282c919061542b565b146128905760405162461bcd60e51b815260206004820152602e60248201527f436c61696d207072656465636573736f72206e6f74206c696e6b656420746f2060448201526d74686973206368616c6c656e676560901b60648201526084016107b0565b6040808501516020015190516344b77df960e11b81526000916001600160a01b0386169163896efbf2916128ca9160040190815260200190565b602060405180830381865afa1580156128e7573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061290b919061542b565b6040516344b77df960e11b8152600481018490529091506000906001600160a01b0386169063896efbf290602401602060405180830381865afa158015612956573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061297a919061542b565b905060006129888284615586565b905086604001516040015181146129d75760405162461bcd60e51b8152602060048201526013602482015272125b9d985b1a59081b195859881a195a59da1d606a1b60448201526064016107b0565b6129ea888860400151896000015161389f565b505050506040838101518051606082015192820151602090920151600093612a1e93909133612a19828a613ac3565b613c34565b60408086015151600090815260208881529190205490860151919250612a479188918491611cd8565b9695505050505050565b60408082015160209081015160009081529085905290812060010154612a895760405162461bcd60e51b81526004016107b090615614565b6040828101516020908101516000908152908690528181206004015480825291902060010154612acb5760405162461bcd60e51b81526004016107b090615642565b60008181526020868152604080832060029081015487830151909301518452922090910154612afa9190615586565b600114612b195760405162461bcd60e51b81526004016107b090615677565b6040808401515160008381526020889052919091206003015414612b4f5760405162461bcd60e51b81526004016107b0906156bd565b600080612b65868660400151876000015161389f565b5050506040808301518051606082015192820151602090920151600093612b9393909133612a198b83611b01565b60408085015151600090815260208781529190205490850151919250610f919187918491611cd8565b60408082015160209081015160009081529085905290812060010154612bf45760405162461bcd60e51b81526004016107b090615614565b6040828101516020908101516000908152908690528181206004015480825291902060010154612c365760405162461bcd60e51b81526004016107b090615642565b60008181526020868152604080832060029081015487830151909301518452922090910154612c659190615586565b600114612c845760405162461bcd60e51b81526004016107b090615677565b6040808401515160008381526020889052919091206003015414612cba5760405162461bcd60e51b81526004016107b0906156bd565b6000612cd2846040015160c001518560800151613d9a565b60008381526020889052604081206002015491925090612cf6906210000090615703565b9050612b65868660400151876000015161389f565b600080612d18868561251f565b60008481526020869052604090206001015415612d475760405162461bcd60e51b81526004016107b09061571a565b612d5286858561374e565b15612da95760405162461bcd60e51b815260206004820152602160248201527f50726573756d707469766520737563636573736f7220636f6e6669726d61626c6044820152606560f81b60648201526084016107b0565b60008481526020879052604090206003015415612dd85760405162461bcd60e51b81526004016107b09061574b565b6000848152602087815260408083205480845291889052822060020154909190612e159060ff166003811115612e1057612e10614db9565b613da5565b90506000612e238783611901565b600081815260208a9052604090205490915015612e525760405162461bcd60e51b81526004016107b09061574b565b9890975095505050505050565b612e67614b0c565b6000849003612e885760405162461bcd60e51b81526004016107b090615599565b6000839003612ea95760405162461bcd60e51b81526004016107b0906155c4565b50604080516101808101825293845260208401929092526000918301829052606083018290526080830182905260a083015260c08201819052600160e083015261010082018190526101208201819052610140820181905261016082015290565b600061064d8260000151836020015184604001516122b8565b6001820154612f445760405162461bcd60e51b81526004016107b090615174565b612f4d82613339565b15612f6a5760405162461bcd60e51b81526004016107b09061577d565b600390910155565b612f7c8382613128565b60008181526020849052604080822060040154825281206003015490819003612fb75760405162461bcd60e51b81526004016107b090615444565b60008181526020849052604090206001015482146122b25760405162461bcd60e51b815260206004820152603b60248201527f53756363657373696f6e206368616c6c656e676520646964206e6f742064656360448201527f6c617265207468697320766572746578207468652077696e6e6572000000000060648201526084016107b0565b6000336001600160a01b038316146130aa5760405162461bcd60e51b815260206004820152602a60248201527f4f6e6c7920617373657274696f6e20636861696e2063616e20637265617465206044820152696368616c6c656e67657360b01b60648201526084016107b0565b60006130b7846000611901565b600081815260208790526040902054909150156108445760405162461bcd60e51b81526004016107b09061574b565b60008061311c83858760405160200161310191815260200190565b60405160208183030381529060405280519060200120613e74565b90951495945050505050565b6000818152602083905260409020600101546131565760405162461bcd60e51b81526004016107b090615174565b60008082815260208490526040902060060154600160a01b900460ff16600181111561318457613184614db9565b146131c95760405162461bcd60e51b8152602060048201526015602482015274566572746578206973206e6f742070656e64696e6760581b60448201526064016107b0565b600081815260208390526040808220600401548083529120600101546132015760405162461bcd60e51b81526004016107b090615374565b6001600082815260208590526040902060060154600160a01b900460ff16600181111561323057613230614db9565b14611a1c5760405162461bcd60e51b8152602060048201526019602482015278141c99591958d95cdcdbdc881b9bdd0818dbdb999a5c9b5959603a1b60448201526064016107b0565b600181015461329a5760405162461bcd60e51b81526004016107b090615174565b60006006820154600160a01b900460ff1660018111156132bc576132bc614db9565b146133235760405162461bcd60e51b815260206004820152603160248201527f566572746578206d7573742062652050656e64696e67206265666f72652062656044820152701a5b99c81cd95d0810dbdb999a5c9b5959607a1b60648201526084016107b0565b600601805460ff60a01b1916600160a01b179055565b60006133488260010154151590565b6133a05760405162461bcd60e51b8152602060048201526024808201527f506f74656e7469616c206c6561662076657274657820646f6573206e6f7420656044820152631e1a5cdd60e21b60648201526084016107b0565b60018201541515801561064d575050600601546001600160a01b0316151590565b60008381526020869052604081206001015481906133f15760405162461bcd60e51b81526004016107b090615174565b6000858152602088815260408083205480845291899052909120600101541561342c5760405162461bcd60e51b81526004016107b09061571a565b600086815260208990526040808220600401548083529120600101546134645760405162461bcd60e51b81526004016107b090615374565b600081815260208a905260409020600701548790036134d15760405162461bcd60e51b815260206004820152602360248201527f43616e6e6f74206269736563742070726573756d70746976652073756363657360448201526239b7b960e91b60648201526084016107b0565b60006134dd8a89613f16565b9050600080878060200190518101906134f6919061581c565b91509150815160001461353c5760405162461bcd60e51b815260206004820152600e60248201526d06e6f7420657175616c20746f20360941b60448201526064016107b0565b61358a8961354b85600161554a565b8e60008e8152602001908152602001600020600101548f60008f8152602001908152602001600020600201546001613583919061554a565b8686613fa6565b613595858a856122b8565b9c929b50919950505050505050505050565b60018201546135c85760405162461bcd60e51b81526004016107b090615174565b808260040154036136155760405162461bcd60e51b8152602060048201526017602482015276141c99591958d95cdcdbdc88185b1c9958591e481cd95d604a1b60448201526064016107b0565b61361e826141bc565b1561366b5760405162461bcd60e51b815260206004820152601e60248201527f43616e6e6f7420736574207072656465636573736f72206f6e20726f6f74000060448201526064016107b0565b600490910155565b60018201546136945760405162461bcd60e51b81526004016107b090615174565b8015806136a5575080826007015414155b6136e25760405162461bcd60e51b815260206004820152600e60248201526d141cc8185b1c9958591e481cd95d60921b60448201526064016107b0565b6136eb82613339565b156137385760405162461bcd60e51b815260206004820152601a60248201527f43616e6e6f7420736574207073206964206f6e2061206c65616600000000000060448201526064016107b0565b600782018190558015611a7857600a9190910155565b60008281526020849052604081206001015461377c5760405162461bcd60e51b81526004016107b090615374565b600083815260208590526040812060070154900361379c57506000610733565b816137bc8586600087815260200190815260200160002060070154611b01565b11949350505050565b60018201546137e65760405162461bcd60e51b81526004016107b090615174565b6137ef826141bc565b156138485760405162461bcd60e51b8152602060048201526024808201527f43616e6e6f742073657420707320666c75736865642074696d65206f6e2061206044820152631c9bdbdd60e21b60648201526084016107b0565b600990910155565b60018201546138715760405162461bcd60e51b81526004016107b090615174565b61387a82613339565b156138975760405162461bcd60e51b81526004016107b09061577d565b600890910155565b60208201516000036138e35760405162461bcd60e51b815260206004820152600d60248201526c115b5c1d1e4818db185a5b5259609a1b60448201526064016107b0565b606082015160000361392b5760405162461bcd60e51b8152602060048201526011602482015270115b5c1d1e481a1a5cdd1bdc9e549bdbdd607a1b60448201526064016107b0565b816040015160000361396e5760405162461bcd60e51b815260206004820152600c60248201526b115b5c1d1e481a195a59da1d60a21b60448201526064016107b0565b8034146139bd5760405162461bcd60e51b815260206004820152601b60248201527f496e636f7272656374206d696e692d7374616b6520616d6f756e74000000000060448201526064016107b0565b8151600090815260208490526040902060010154156139ee5760405162461bcd60e51b81526004016107b09061571a565b613a1682606001518360c0015160018560400151613a0c9190615586565b8560e001516130e6565b613a5e5760405162461bcd60e51b81526020600482015260196024820152784c617374207374617465206e6f7420696e20686973746f727960381b60448201526064016107b0565b613a778260600151836080015160008560a001516130e6565b611a1c5760405162461bcd60e51b815260206004820152601a60248201527f4669727374207374617465206e6f7420696e20686973746f727900000000000060448201526064016107b0565b6040516306106c4560e31b81526004810183905260009081906001600160a01b03841690633083622890602401602060405180830381865afa158015613b0d573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190613b319190615875565b90508015613c2a57604051632729597560e21b8152600481018590526000906001600160a01b03851690639ca565d490602401602060405180830381865afa158015613b81573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190613ba5919061542b565b6040516343ed6ad960e01b8152600481018290529091506000906001600160a01b038616906343ed6ad990602401602060405180830381865afa158015613bf0573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190613c14919061542b565b9050613c208142615586565b935050505061064d565b600091505061064d565b613c3c614b0c565b6000879003613c5d5760405162461bcd60e51b81526004016107b090615599565b6000869003613c7e5760405162461bcd60e51b81526004016107b0906155c4565b84600003613c9e5760405162461bcd60e51b81526004016107b0906155ef565b6000849003613cdf5760405162461bcd60e51b815260206004820152600d60248201526c16995c9bc818db185a5b481a59609a1b60448201526064016107b0565b6001600160a01b038316613d2b5760405162461bcd60e51b81526020600482015260136024820152725a65726f207374616b6572206164647265737360681b60448201526064016107b0565b5060408051610180810182529687526020870195909552938501929092526000606085018190526080850181905260a08501919091526001600160a01b0390911660c084015260e083018190526101008301819052610120830181905261014083019190915261016082015290565b600061064a82615897565b600080826003811115613dba57613dba614db9565b03613dc757506001919050565b6001826003811115613ddb57613ddb614db9565b03613de857506002919050565b6002826003811115613dfc57613dfc614db9565b03613e0957506003919050565b60405162461bcd60e51b815260206004820152603560248201527f43616e6e6f7420676574206e657874206368616c6c656e6765207479706520666044820152746f72206f6e652073746570206368616c6c656e676560581b60648201526084016107b0565b919050565b8251600090610100811115613ea757604051637ed6198f60e11b81526004810182905261010060248201526044016107b0565b8260005b82811015613f0c576000878281518110613ec757613ec76153b5565b60200260200101519050816001901b8716600003613ef357826000528060205260406000209250613f03565b8060005282602052604060002092505b50600101613eab565b5095945050505050565b600081815260208390526040812060010154613f445760405162461bcd60e51b81526004016107b090615174565b60008281526020849052604080822060040154808352912060010154613f7c5760405162461bcd60e51b81526004016107b090615374565b60008181526020859052604080822060029081015486845291909220909101546108449190614243565b60008511613fed5760405162461bcd60e51b815260206004820152601460248201527305072652d73697a652063616e6e6f7420626520360641b60448201526064016107b0565b85613ff7836142f8565b146140445760405162461bcd60e51b815260206004820152601b60248201527f50726520657870616e73696f6e20726f6f74206d69736d61746368000000000060448201526064016107b0565b8285106140935760405162461bcd60e51b815260206004820181905260248201527f5072652073697a65206e6f74206c657373207468616e20706f73742073697a6560448201526064016107b0565b8460005b848210156141075760006140ab83876143f8565b90506140d185828685815181106140c4576140c46153b5565b60200260200101516144e7565b94506001811b6140e1818561554a565b9350868411156140f3576140f36158be565b826140fd816158d4565b9350505050614097565b85614111856142f8565b146141695760405162461bcd60e51b815260206004820152602260248201527f506f737420657870616e73696f6e20726f6f74206e6f7420657175616c20706f6044820152611cdd60f21b60648201526084016107b0565b825181146141b25760405162461bcd60e51b8152602060048201526016602482015275496e636f6d706c6574652070726f6f6620757361676560501b60448201526064016107b0565b5050505050505050565b60006141cb8260010154151590565b6142235760405162461bcd60e51b8152602060048201526024808201527f506f74656e7469616c20726f6f742076657274657820646f6573206e6f7420656044820152631e1a5cdd60e21b60648201526084016107b0565b60068201546001600160a01b031615801561064d57505060050154151590565b600060026142518484615586565b101561429f5760405162461bcd60e51b815260206004820181905260248201527f48656967687420646966666572656e74206e6f742074776f206f72206d6f726560448201526064016107b0565b6142a98383615586565b6002036142c2576142bb83600161554a565b905061064d565b60006142d9846142d3600186615586565b186148ba565b9050600019811b6001816142ed8287615586565b16610f919190615586565b6000600181805b84518110156143f057600085828151811061431c5761431c6153b5565b6020026020010151905083156143885780156143835760009350809250600186516143479190615586565b82146143835760405161436a908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b6143dd565b80156143a757604080516020810183905290810184905260600161436a565b6040516143c4908490600090602001918252602082015260400190565b6040516020818303038152906040528051906020012092505b50806143e8816158d4565b9150506142ff565b509392505050565b60008183106144435760405162461bcd60e51b815260206004820152601760248201527614dd185c9d081b9bdd081b195cdcc81d1a185b88195b99604a1b60448201526064016107b0565b6000614450838518614999565b905060006001614460838261554a565b6001901b61446e9190615586565b9050848116848216811561449057614485826149d6565b94505050505061064d565b801561449f5761448581614999565b60405162461bcd60e51b815260206004820152601b60248201527f426f7468207920616e64207a2063616e6e6f74206265207a65726f000000000060448201526064016107b0565b6060610100831061450a5760405162461bcd60e51b81526004016107b0906158ed565b600082900361455b5760405162461bcd60e51b815260206004820152601b60248201527f43616e6e6f7420617070656e6420656d7074792073756274726565000000000060448201526064016107b0565b600061456884600161554a565b6001600160401b0381111561457f5761457f614ca3565b6040519080825280602002602001820160405280156145a8578160200160208202803683370190505b50905084516000036146215760005b84811161461f578481036145eb57838282815181106145d8576145d86153b5565b6020908102919091010152509050610733565b6000801b828281518110614601576146016153b5565b60200260200101818152505080614617816158d4565b9150506145b7565b505b8451841061468f5760405162461bcd60e51b815260206004820152603560248201527f4c6576656c2067726561746572207468616e2068696768657374206c6576656c6044820152741037b31031bab93932b73a1032bc3830b739b4b7b760591b60648201526084016107b0565b845183906000906001600160401b038111156146ad576146ad614ca3565b6040519080825280602002602001820160405280156146d6578160200160208202803683370190505b50905060005b875181101561487a5786811015614768578781815181106146ff576146ff6153b5565b60200260200101516000801b146147635760405162461bcd60e51b815260206004820152602260248201527f417070656e642061626f7665206c65617374207369676e69666963616e7420626044820152611a5d60f21b60648201526084016107b0565b614868565b60008390036147ae57878181518110614783576147836153b5565b602002602001015182828151811061479d5761479d6153b5565b602002602001018181525050614868565b8781815181106147c0576147c06153b5565b60200260200101516000801b036147f857828282815181106147e4576147e46153b5565b602090810291909101015260009250614868565b6000801b82828151811061480e5761480e6153b5565b60200260200101818152505087818151811061482c5761482c6153b5565b60200260200101518360405160200161484f929190918252602082015260400190565b6040516020818303038152906040528051906020012092505b80614872816158d4565b9150506146dc565b50811561488e5761488b8183614a2f565b90505b61489b610100600161554a565b815110612a475760405162461bcd60e51b81526004016107b0906158ed565b6000600160801b82106148da57608091821c916148d7908261554a565b90505b600160401b82106148f857604091821c916148f5908261554a565b90505b640100000000821061491757602091821c91614914908261554a565b90505b62010000821061493457601091821c91614931908261554a565b90505b610100821061495057600891821c9161494d908261554a565b90505b6010821061496b57600491821c91614968908261554a565b90505b6004821061498657600291821c91614983908261554a565b90505b60028210613e6f5761064d60018261554a565b6000816000036149bb5760405162461bcd60e51b81526004016107b090615915565b600160801b82106148da57608091821c916148d7908261554a565b60008082116149f75760405162461bcd60e51b81526004016107b090615915565b60005b60019290921b918215614a1757614a10816158d4565b90506149fa565b6001614a2582610100615586565b6107339190615586565b6060600083516001614a41919061554a565b6001600160401b03811115614a5857614a58614ca3565b604051908082528060200260200182016040528015614a81578160200160208202803683370190505b50905060005b8451811015614ad957848181518110614aa257614aa26153b5565b6020026020010151828281518110614abc57614abc6153b5565b602090810291909101015280614ad1816158d4565b915050614a87565b50828160018351614aea9190615586565b81518110614afa57614afa6153b5565b60209081029190910101529392505050565b6040805161018081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c081018290529060e08201908152600060208201819052604082018190526060820181905260809091015290565b60008083601f840112614b7f57600080fd5b5081356001600160401b03811115614b9657600080fd5b6020830191508360208260051b8501011115610fb757600080fd5b60008060008060008060808789031215614bca57600080fd5b8635955060208701356001600160401b0380821115614be857600080fd5b9088019060a0828b031215614bfc57600080fd5b90955060408801359080821115614c1257600080fd5b614c1e8a838b01614b6d565b90965094506060890135915080821115614c3757600080fd5b50614c4489828a01614b6d565b979a9699509497509295939492505050565b60008060408385031215614c6957600080fd5b82359150602083013560048110614c7f57600080fd5b809150509250929050565b600060208284031215614c9c57600080fd5b5035919050565b634e487b7160e01b600052604160045260246000fd5b60405161010081016001600160401b0381118282101715614cdc57614cdc614ca3565b60405290565b604051601f8201601f191681016001600160401b0381118282101715614d0a57614d0a614ca3565b604052919050565b600080600060608486031215614d2757600080fd5b83359250602080850135925060408501356001600160401b0380821115614d4d57600080fd5b818701915087601f830112614d6157600080fd5b813581811115614d7357614d73614ca3565b614d85601f8201601f19168501614ce2565b91508082528884828501011115614d9b57600080fd5b80848401858401376000848284010152508093505050509250925092565b634e487b7160e01b600052602160045260246000fd5b60048110614ddf57614ddf614db9565b9052565b600060808201905082518252602083015160208301526040830151614e0b6040840182614dcf565b506060928301516001600160a01b0316919092015290565b600080600060608486031215614e3857600080fd5b505081359360208301359350604090920135919050565b60028110614ddf57614ddf614db9565b600061018082019050825182526020830151602083015260408301516040830152606083015160608301526080830151608083015260a083015160a083015260c0830151614eb860c08401826001600160a01b03169052565b5060e0830151614ecb60e0840182614e4f565b5061010083810151908301526101208084015190830152610140808401519083015261016092830151929091019190915290565b6001600160a01b038116811461066a57600080fd5b60008060008060808587031215614f2a57600080fd5b8435614f3581614eff565b935060208501359250604085013591506060850135614f5381614eff565b939692955090935050565b60008083601f840112614f7057600080fd5b5081356001600160401b03811115614f8757600080fd5b602083019150836020828501011115610fb757600080fd5b600080600080600060608688031215614fb757600080fd5b85356001600160401b0380821115614fce57600080fd5b90870190610100828a031215614fe357600080fd5b90955060208701359080821115614ff957600080fd5b61500589838a01614f5e565b9096509450604088013591508082111561501e57600080fd5b5061502b88828901614f5e565b969995985093965092949392505050565b6000806020838503121561504f57600080fd5b82356001600160401b0381111561506557600080fd5b61507185828601614f5e565b90969095509350505050565b84815260208101849052608081016150986040830185614dcf565b6001600160a01b0392909216606091909101529392505050565b600060208083528351808285015260005b818110156150df578581018301518582016040015282016150c3565b506000604082860101526040601f19601f8301168501019250505092915050565b8c8152602081018c9052604081018b9052606081018a90526080810189905260a081018890526001600160a01b03871660c0820152610180810161514760e0830188614e4f565b856101008301528461012083015283610140830152826101608301529d9c50505050505050505050505050565b60208082526015908201527415995c9d195e08191bd95cc81b9bdd08195e1a5cdd605a1b604082015260600190565b6020808252601a908201527f5072656465636573736f7220646f6573206e6f74206578697374000000000000604082015260600190565b60006001600160401b038211156151f3576151f3614ca3565b5060051b60200190565b600082601f83011261520e57600080fd5b8135602061522361521e836151da565b614ce2565b82815260059290921b8401810191818101908684111561524257600080fd5b8286015b8481101561525d5780358352918301918301615246565b509695505050505050565b6000610100823603121561527b57600080fd5b615283614cb9565b823581526020830135602082015260408301356040820152606083013560608201526080830135608082015260a08301356001600160401b03808211156152c957600080fd5b6152d5368387016151fd565b60a084015260c085013560c084015260e08501359150808211156152f857600080fd5b50615305368286016151fd565b60e08301525092915050565b6000806040838503121561532457600080fd5b82356001600160401b038082111561533b57600080fd5b615347868387016151fd565b9350602085013591508082111561535d57600080fd5b5061536a858286016151fd565b9150509250929050565b60208082526021908201527f5072656465636573736f722076657274657820646f6573206e6f7420657869736040820152601d60fa1b606082015260800190565b634e487b7160e01b600052603260045260246000fd5b600081518084526020808501945080840160005b838110156153fb578151875295820195908201906001016153df565b509495945050505050565b60408152600061541960408301856153cb565b8281036020840152610f9181856153cb565b60006020828403121561543d57600080fd5b5051919050565b60208082526023908201527f53756363657373696f6e206368616c6c656e676520646f6573206e6f742065786040820152621a5cdd60ea1b606082015260800190565b6000808335601e1984360301811261549e57600080fd5b8301803591506001600160401b038211156154b857600080fd5b602001915036819003821315610fb757600080fd5b85358152600060208701356154e181614eff565b6001600160a01b03166020830152604082018690526060820185905260a0608083018190528201839052828460c0840137600060c0848401015260c0601f19601f85011683010190509695505050505050565b634e487b7160e01b600052601160045260246000fd5b8082018082111561064d5761064d615534565b82815260006004831061557257615572614db9565b5060f89190911b6020820152602101919050565b8181038181111561064d5761064d615534565b60208082526011908201527016995c9bc818da185b1b195b99d9481a59607a1b604082015260600190565b60208082526011908201527016995c9bc81a1a5cdd1bdc9e481c9bdbdd607a1b604082015260600190565b6020808252600b908201526a16995c9bc81a195a59da1d60aa1b604082015260600190565b60208082526014908201527310db185a5b48191bd95cc81b9bdd08195e1a5cdd60621b604082015260600190565b6020808252818101527f436c61696d207072656465636573736f7220646f6573206e6f74206578697374604082015260600190565b60208082526026908201527f436c61696d206e6f7420686569676874206f6e652061626f766520707265646560408201526531b2b9b9b7b960d11b606082015260800190565b60208082526026908201527f436c61696d2068617320696e76616c69642073756363657373696f6e206368616040820152656c6c656e676560d01b606082015260800190565b808202811582820484141761064d5761064d615534565b60208082526017908201527615da5b9b995c88185b1c9958591e48191958db185c9959604a1b604082015260600190565b6020808252601890820152774368616c6c656e676520616c72656164792065786973747360401b604082015260600190565b60208082526024908201527f43616e6e6f7420736574207073206c6173742075706461746564206f6e2061206040820152633632b0b360e11b606082015260800190565b600082601f8301126157d257600080fd5b815160206157e261521e836151da565b82815260059290921b8401810191818101908684111561580157600080fd5b8286015b8481101561525d5780518352918301918301615805565b6000806040838503121561582f57600080fd5b82516001600160401b038082111561584657600080fd5b615852868387016157c1565b9350602085015191508082111561586857600080fd5b5061536a858286016157c1565b60006020828403121561588757600080fd5b8151801515811461073357600080fd5b805160208083015191908110156158b8576000198160200360031b1b821691505b50919050565b634e487b7160e01b600052600160045260246000fd5b6000600182016158e6576158e6615534565b5060010190565b6020808252600e908201526d098caeccad840e8dede40d0d2ced60931b604082015260600190565b6020808252601c908201527f5a65726f20686173206e6f207369676e69666963616e7420626974730000000060408201526060019056fea2646970667358221220cb83907adf3a2481fb60b72e372e14629f6a84df510cb5b81ad5a49edc04371064736f6c63430008110033",
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

// CrazyStuff is a free data retrieval call binding the contract method 0xe95756d5.
//
// Solidity: function crazyStuff() pure returns(bytes)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) CrazyStuff(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "crazyStuff")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// CrazyStuff is a free data retrieval call binding the contract method 0xe95756d5.
//
// Solidity: function crazyStuff() pure returns(bytes)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) CrazyStuff() ([]byte, error) {
	return _ChallengeManagerImpl.Contract.CrazyStuff(&_ChallengeManagerImpl.CallOpts)
}

// CrazyStuff is a free data retrieval call binding the contract method 0xe95756d5.
//
// Solidity: function crazyStuff() pure returns(bytes)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) CrazyStuff() ([]byte, error) {
	return _ChallengeManagerImpl.Contract.CrazyStuff(&_ChallengeManagerImpl.CallOpts)
}

// DecodeLens is a free data retrieval call binding the contract method 0xb4f17201.
//
// Solidity: function decodeLens(bytes item) pure returns(uint256, uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplCaller) DecodeLens(opts *bind.CallOpts, item []byte) (*big.Int, *big.Int, error) {
	var out []interface{}
	err := _ChallengeManagerImpl.contract.Call(opts, &out, "decodeLens", item)

	if err != nil {
		return *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// DecodeLens is a free data retrieval call binding the contract method 0xb4f17201.
//
// Solidity: function decodeLens(bytes item) pure returns(uint256, uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplSession) DecodeLens(item []byte) (*big.Int, *big.Int, error) {
	return _ChallengeManagerImpl.Contract.DecodeLens(&_ChallengeManagerImpl.CallOpts, item)
}

// DecodeLens is a free data retrieval call binding the contract method 0xb4f17201.
//
// Solidity: function decodeLens(bytes item) pure returns(uint256, uint256)
func (_ChallengeManagerImpl *ChallengeManagerImplCallerSession) DecodeLens(item []byte) (*big.Int, *big.Int, error) {
	return _ChallengeManagerImpl.Contract.DecodeLens(&_ChallengeManagerImpl.CallOpts, item)
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
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220092c99b45a0389518f55e263d7fe17f478b44d5adad66a1d847d7aea68c432b964736f6c63430008110033",
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
	Bin: "0x608060405234801561001057600080fd5b5061018a806100206000396000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c80632f3069611461005157806335025bde1461007357806373d154e814610098578063a4714dbb146100b8575b600080fd5b61007161005f366004610119565b60009182526020829052604090912055565b005b61008661008136600461013b565b6100d8565b60405190815260200160405180910390f35b6100866100a636600461013b565b60009081526020819052604090205490565b6100866100c636600461013b565b60006020819052908152604090205481565b60405162461bcd60e51b815260206004820152600f60248201526e1393d517d253541311535153951151608a1b604482015260009060640160405180910390fd5b6000806040838503121561012c57600080fd5b50508035926020909101359150565b60006020828403121561014d57600080fd5b503591905056fea264697066735822122004f0b2ab0487dd5b9f4d703852345ca047f29eba0cc99fa02365353ca83a553364736f6c63430008110033",
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
