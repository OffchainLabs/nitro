// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package express_lane_auctiongen

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

// Bid is an auto generated low-level Go binding around an user-defined struct.
type Bid struct {
	ExpressLaneController common.Address
	Amount                *big.Int
	Signature             []byte
}

// ELCRound is an auto generated low-level Go binding around an user-defined struct.
type ELCRound struct {
	ExpressLaneController common.Address
	Round                 uint64
}

// InitArgs is an auto generated low-level Go binding around an user-defined struct.
type InitArgs struct {
	Auctioneer              common.Address
	BiddingToken            common.Address
	Beneficiary             common.Address
	RoundTimingInfo         RoundTimingInfo
	MinReservePrice         *big.Int
	AuctioneerAdmin         common.Address
	MinReservePriceSetter   common.Address
	ReservePriceSetter      common.Address
	ReservePriceSetterAdmin common.Address
	BeneficiarySetter       common.Address
	RoundTimingSetter       common.Address
	MasterAdmin             common.Address
}

// RoundTimingInfo is an auto generated low-level Go binding around an user-defined struct.
type RoundTimingInfo struct {
	OffsetTimestamp          int64
	RoundDurationSeconds     uint64
	AuctionClosingSeconds    uint64
	ReserveSubmissionSeconds uint64
}

// Transferor is an auto generated low-level Go binding around an user-defined struct.
type Transferor struct {
	Addr            common.Address
	FixedUntilRound uint64
}

// BalanceLibMetaData contains all meta data concerning the BalanceLib contract.
var BalanceLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220301bbe798c874e41585214eb8f3ae9d8921c3c8b36a26d0a38d6d83db6b852be64736f6c63430008110033",
}

// BalanceLibABI is the input ABI used to generate the binding from.
// Deprecated: Use BalanceLibMetaData.ABI instead.
var BalanceLibABI = BalanceLibMetaData.ABI

// BalanceLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BalanceLibMetaData.Bin instead.
var BalanceLibBin = BalanceLibMetaData.Bin

// DeployBalanceLib deploys a new Ethereum contract, binding an instance of BalanceLib to it.
func DeployBalanceLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BalanceLib, error) {
	parsed, err := BalanceLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BalanceLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BalanceLib{BalanceLibCaller: BalanceLibCaller{contract: contract}, BalanceLibTransactor: BalanceLibTransactor{contract: contract}, BalanceLibFilterer: BalanceLibFilterer{contract: contract}}, nil
}

// BalanceLib is an auto generated Go binding around an Ethereum contract.
type BalanceLib struct {
	BalanceLibCaller     // Read-only binding to the contract
	BalanceLibTransactor // Write-only binding to the contract
	BalanceLibFilterer   // Log filterer for contract events
}

// BalanceLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type BalanceLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BalanceLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BalanceLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BalanceLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BalanceLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BalanceLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BalanceLibSession struct {
	Contract     *BalanceLib       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BalanceLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BalanceLibCallerSession struct {
	Contract *BalanceLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// BalanceLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BalanceLibTransactorSession struct {
	Contract     *BalanceLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// BalanceLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type BalanceLibRaw struct {
	Contract *BalanceLib // Generic contract binding to access the raw methods on
}

// BalanceLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BalanceLibCallerRaw struct {
	Contract *BalanceLibCaller // Generic read-only contract binding to access the raw methods on
}

// BalanceLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BalanceLibTransactorRaw struct {
	Contract *BalanceLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBalanceLib creates a new instance of BalanceLib, bound to a specific deployed contract.
func NewBalanceLib(address common.Address, backend bind.ContractBackend) (*BalanceLib, error) {
	contract, err := bindBalanceLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BalanceLib{BalanceLibCaller: BalanceLibCaller{contract: contract}, BalanceLibTransactor: BalanceLibTransactor{contract: contract}, BalanceLibFilterer: BalanceLibFilterer{contract: contract}}, nil
}

// NewBalanceLibCaller creates a new read-only instance of BalanceLib, bound to a specific deployed contract.
func NewBalanceLibCaller(address common.Address, caller bind.ContractCaller) (*BalanceLibCaller, error) {
	contract, err := bindBalanceLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BalanceLibCaller{contract: contract}, nil
}

// NewBalanceLibTransactor creates a new write-only instance of BalanceLib, bound to a specific deployed contract.
func NewBalanceLibTransactor(address common.Address, transactor bind.ContractTransactor) (*BalanceLibTransactor, error) {
	contract, err := bindBalanceLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BalanceLibTransactor{contract: contract}, nil
}

// NewBalanceLibFilterer creates a new log filterer instance of BalanceLib, bound to a specific deployed contract.
func NewBalanceLibFilterer(address common.Address, filterer bind.ContractFilterer) (*BalanceLibFilterer, error) {
	contract, err := bindBalanceLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BalanceLibFilterer{contract: contract}, nil
}

// bindBalanceLib binds a generic wrapper to an already deployed contract.
func bindBalanceLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BalanceLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BalanceLib *BalanceLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BalanceLib.Contract.BalanceLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BalanceLib *BalanceLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BalanceLib.Contract.BalanceLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BalanceLib *BalanceLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BalanceLib.Contract.BalanceLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BalanceLib *BalanceLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BalanceLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BalanceLib *BalanceLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BalanceLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BalanceLib *BalanceLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BalanceLib.Contract.contract.Transact(opts, method, params...)
}

// BurnerMetaData contains all meta data concerning the Burner contract.
var BurnerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"ZeroAddress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token\",\"outputs\":[{\"internalType\":\"contractERC20BurnableUpgradeable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b506040516102a33803806102a383398101604081905261002f91610067565b6001600160a01b0381166100565760405163d92e233d60e01b815260040160405180910390fd5b6001600160a01b0316608052610097565b60006020828403121561007957600080fd5b81516001600160a01b038116811461009057600080fd5b9392505050565b6080516101ec6100b760003960008181604a015260c301526101ec6000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c806344df8e701461003b578063fc0c546a14610045575b600080fd5b610043610095565b005b61006c7f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200160405180910390f35b6040517f70a082310000000000000000000000000000000000000000000000000000000081523060048201527f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16906342966c689082906370a0823190602401602060405180830381865afa158015610127573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061014b919061019d565b6040518263ffffffff1660e01b815260040161016991815260200190565b600060405180830381600087803b15801561018357600080fd5b505af1158015610197573d6000803e3d6000fd5b50505050565b6000602082840312156101af57600080fd5b505191905056fea2646970667358221220151b7bb5a8b8e838041af342c4b353932ddaf5692053ccf3784e39b4e1d561ee64736f6c63430008110033",
}

// BurnerABI is the input ABI used to generate the binding from.
// Deprecated: Use BurnerMetaData.ABI instead.
var BurnerABI = BurnerMetaData.ABI

// BurnerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BurnerMetaData.Bin instead.
var BurnerBin = BurnerMetaData.Bin

// DeployBurner deploys a new Ethereum contract, binding an instance of Burner to it.
func DeployBurner(auth *bind.TransactOpts, backend bind.ContractBackend, _token common.Address) (common.Address, *types.Transaction, *Burner, error) {
	parsed, err := BurnerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BurnerBin), backend, _token)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Burner{BurnerCaller: BurnerCaller{contract: contract}, BurnerTransactor: BurnerTransactor{contract: contract}, BurnerFilterer: BurnerFilterer{contract: contract}}, nil
}

// Burner is an auto generated Go binding around an Ethereum contract.
type Burner struct {
	BurnerCaller     // Read-only binding to the contract
	BurnerTransactor // Write-only binding to the contract
	BurnerFilterer   // Log filterer for contract events
}

// BurnerCaller is an auto generated read-only Go binding around an Ethereum contract.
type BurnerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BurnerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BurnerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BurnerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BurnerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BurnerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BurnerSession struct {
	Contract     *Burner           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BurnerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BurnerCallerSession struct {
	Contract *BurnerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// BurnerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BurnerTransactorSession struct {
	Contract     *BurnerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BurnerRaw is an auto generated low-level Go binding around an Ethereum contract.
type BurnerRaw struct {
	Contract *Burner // Generic contract binding to access the raw methods on
}

// BurnerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BurnerCallerRaw struct {
	Contract *BurnerCaller // Generic read-only contract binding to access the raw methods on
}

// BurnerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BurnerTransactorRaw struct {
	Contract *BurnerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBurner creates a new instance of Burner, bound to a specific deployed contract.
func NewBurner(address common.Address, backend bind.ContractBackend) (*Burner, error) {
	contract, err := bindBurner(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Burner{BurnerCaller: BurnerCaller{contract: contract}, BurnerTransactor: BurnerTransactor{contract: contract}, BurnerFilterer: BurnerFilterer{contract: contract}}, nil
}

// NewBurnerCaller creates a new read-only instance of Burner, bound to a specific deployed contract.
func NewBurnerCaller(address common.Address, caller bind.ContractCaller) (*BurnerCaller, error) {
	contract, err := bindBurner(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BurnerCaller{contract: contract}, nil
}

// NewBurnerTransactor creates a new write-only instance of Burner, bound to a specific deployed contract.
func NewBurnerTransactor(address common.Address, transactor bind.ContractTransactor) (*BurnerTransactor, error) {
	contract, err := bindBurner(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BurnerTransactor{contract: contract}, nil
}

// NewBurnerFilterer creates a new log filterer instance of Burner, bound to a specific deployed contract.
func NewBurnerFilterer(address common.Address, filterer bind.ContractFilterer) (*BurnerFilterer, error) {
	contract, err := bindBurner(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BurnerFilterer{contract: contract}, nil
}

// bindBurner binds a generic wrapper to an already deployed contract.
func bindBurner(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BurnerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Burner *BurnerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Burner.Contract.BurnerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Burner *BurnerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Burner.Contract.BurnerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Burner *BurnerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Burner.Contract.BurnerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Burner *BurnerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Burner.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Burner *BurnerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Burner.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Burner *BurnerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Burner.Contract.contract.Transact(opts, method, params...)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_Burner *BurnerCaller) Token(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Burner.contract.Call(opts, &out, "token")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_Burner *BurnerSession) Token() (common.Address, error) {
	return _Burner.Contract.Token(&_Burner.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_Burner *BurnerCallerSession) Token() (common.Address, error) {
	return _Burner.Contract.Token(&_Burner.CallOpts)
}

// Burn is a paid mutator transaction binding the contract method 0x44df8e70.
//
// Solidity: function burn() returns()
func (_Burner *BurnerTransactor) Burn(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Burner.contract.Transact(opts, "burn")
}

// Burn is a paid mutator transaction binding the contract method 0x44df8e70.
//
// Solidity: function burn() returns()
func (_Burner *BurnerSession) Burn() (*types.Transaction, error) {
	return _Burner.Contract.Burn(&_Burner.TransactOpts)
}

// Burn is a paid mutator transaction binding the contract method 0x44df8e70.
//
// Solidity: function burn() returns()
func (_Burner *BurnerTransactorSession) Burn() (*types.Transaction, error) {
	return _Burner.Contract.Burn(&_Burner.TransactOpts)
}

// ExpressLaneAuctionMetaData contains all meta data concerning the ExpressLaneAuction contract.
var ExpressLaneAuctionMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"AuctionNotClosed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BidsWrongOrder\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"fixedUntilRound\",\"type\":\"uint64\"}],\"name\":\"FixedTransferor\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountRequested\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"name\":\"InsufficientBalance\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amountRequested\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"name\":\"InsufficientBalanceAcc\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"currentRound\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"newRound\",\"type\":\"uint64\"}],\"name\":\"InvalidNewRound\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"currentStart\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"newStart\",\"type\":\"uint64\"}],\"name\":\"InvalidNewStart\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NegativeOffset\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"int64\",\"name\":\"roundStart\",\"type\":\"int64\"}],\"name\":\"NegativeRoundStart\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"controller\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"NotExpressLaneController\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"expectedTransferor\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgSender\",\"type\":\"address\"}],\"name\":\"NotTransferor\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NothingToWithdraw\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReserveBlackout\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"bidAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reservePrice\",\"type\":\"uint256\"}],\"name\":\"ReservePriceNotMet\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"reservePrice\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"minReservePrice\",\"type\":\"uint256\"}],\"name\":\"ReservePriceTooLow\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"name\":\"RoundAlreadyResolved\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RoundDurationTooShort\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"name\":\"RoundNotResolved\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"roundDurationSeconds\",\"type\":\"uint64\"}],\"name\":\"RoundTooLong\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"currentRound\",\"type\":\"uint64\"}],\"name\":\"RoundTooOld\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SameBidder\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TieBidsWrongOrder\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"WithdrawalInProgress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"WithdrawalMaxRound\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAuctionClosingSeconds\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroBiddingToken\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"isMultiBidAuction\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"firstPriceBidder\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"firstPriceExpressLaneController\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"firstPriceAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"roundStartTimestamp\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"roundEndTimestamp\",\"type\":\"uint64\"}],\"name\":\"AuctionResolved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldBeneficiary\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newBeneficiary\",\"type\":\"address\"}],\"name\":\"SetBeneficiary\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousExpressLaneController\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newExpressLaneController\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"transferor\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"startTimestamp\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"endTimestamp\",\"type\":\"uint64\"}],\"name\":\"SetExpressLaneController\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldPrice\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newPrice\",\"type\":\"uint256\"}],\"name\":\"SetMinReservePrice\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldReservePrice\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newReservePrice\",\"type\":\"uint256\"}],\"name\":\"SetReservePrice\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"currentRound\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"int64\",\"name\":\"offsetTimestamp\",\"type\":\"int64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"roundDurationSeconds\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"auctionClosingSeconds\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"reserveSubmissionSeconds\",\"type\":\"uint64\"}],\"name\":\"SetRoundTimingInfo\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"transferor\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"fixedUntilRound\",\"type\":\"uint64\"}],\"name\":\"SetTransferor\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"withdrawalAmount\",\"type\":\"uint256\"}],\"name\":\"WithdrawalFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"withdrawalAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"roundWithdrawable\",\"type\":\"uint256\"}],\"name\":\"WithdrawalInitiated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"AUCTIONEER_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"AUCTIONEER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"BENEFICIARY_SETTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_RESERVE_SETTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RESERVE_SETTER_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RESERVE_SETTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ROUND_TIMING_SETTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"name\":\"balanceOfAtRound\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"beneficiary\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"beneficiaryBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"biddingToken\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentRound\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"deposit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"domainSeparator\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"finalizeWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"flushBeneficiaryBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"getBidHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getRoleMember\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleMemberCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"_auctioneer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_biddingToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_beneficiary\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"int64\",\"name\":\"offsetTimestamp\",\"type\":\"int64\"},{\"internalType\":\"uint64\",\"name\":\"roundDurationSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"auctionClosingSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"reserveSubmissionSeconds\",\"type\":\"uint64\"}],\"internalType\":\"structRoundTimingInfo\",\"name\":\"_roundTimingInfo\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"_minReservePrice\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_auctioneerAdmin\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_minReservePriceSetter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_reservePriceSetter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_reservePriceSetterAdmin\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_beneficiarySetter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_roundTimingSetter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_masterAdmin\",\"type\":\"address\"}],\"internalType\":\"structInitArgs\",\"name\":\"args\",\"type\":\"tuple\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initiateWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isAuctionRoundClosed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isReserveBlackout\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minReservePrice\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"reservePrice\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structBid\",\"name\":\"firstPriceBid\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structBid\",\"name\":\"secondPriceBid\",\"type\":\"tuple\"}],\"name\":\"resolveMultiBidAuction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structBid\",\"name\":\"firstPriceBid\",\"type\":\"tuple\"}],\"name\":\"resolveSingleBidAuction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"resolvedRounds\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"internalType\":\"structELCRound\",\"name\":\"\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"internalType\":\"structELCRound\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"name\":\"roundTimestamps\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"roundTimingInfo\",\"outputs\":[{\"internalType\":\"int64\",\"name\":\"offsetTimestamp\",\"type\":\"int64\"},{\"internalType\":\"uint64\",\"name\":\"roundDurationSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"auctionClosingSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"reserveSubmissionSeconds\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newBeneficiary\",\"type\":\"address\"}],\"name\":\"setBeneficiary\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newMinReservePrice\",\"type\":\"uint256\"}],\"name\":\"setMinReservePrice\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newReservePrice\",\"type\":\"uint256\"}],\"name\":\"setReservePrice\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"int64\",\"name\":\"offsetTimestamp\",\"type\":\"int64\"},{\"internalType\":\"uint64\",\"name\":\"roundDurationSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"auctionClosingSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"reserveSubmissionSeconds\",\"type\":\"uint64\"}],\"internalType\":\"structRoundTimingInfo\",\"name\":\"newRoundTimingInfo\",\"type\":\"tuple\"}],\"name\":\"setRoundTimingInfo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"fixedUntilRound\",\"type\":\"uint64\"}],\"internalType\":\"structTransferor\",\"name\":\"transferor\",\"type\":\"tuple\"}],\"name\":\"setTransferor\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"newExpressLaneController\",\"type\":\"address\"}],\"name\":\"transferExpressLaneController\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"transferorOf\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"fixedUntilRound\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"withdrawableBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"name\":\"withdrawableBalanceAtRound\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a06040523060805234801561001457600080fd5b506080516147d7610030600039600061173901526147d76000f3fe608060405234801561001057600080fd5b50600436106103145760003560e01c80637b617f94116101a7578063c5b6aa2f116100ee578063e2fc6f6811610097578063e4d20c1d11610071578063e4d20c1d146107de578063f698da25146107f1578063fed87be8146107f957600080fd5b8063e2fc6f68146107a5578063e3f7bb55146107af578063e460d2c5146107d657600080fd5b8063cfe9232b116100c8578063cfe9232b14610761578063d547741f14610788578063db2e1eed1461079b57600080fd5b8063c5b6aa2f14610733578063ca15c8731461073b578063ce9c7c0d1461074e57600080fd5b80639a1fadd311610150578063b51d1d4f1161012a578063b51d1d4f14610705578063b6b55f251461070d578063bef0ec741461072057600080fd5b80639a1fadd3146106c3578063a217fddf146106d6578063b3ee252f146106de57600080fd5b80638a19c8bc116101815780638a19c8bc146106565780639010d07c1461067757806391d148541461068a57600080fd5b80637b617f94146105f157806383af0a1f146106255780638948cc4e1461062f57600080fd5b80632f2ff15d1161026b578063639d7566116102145780636dc4fc4e116101ee5780636dc4fc4e146105b85780636e8cace5146105cb57806370a08231146105de57600080fd5b8063639d7566146105375780636a514beb1461054b5780636ad72517146105b057600080fd5b806338af3eed1161024557806338af3eed146104e5578063447a709e146105115780635633c3371461052457600080fd5b80632f2ff15d14610498578063336a5b5e146104ab57806336568abe146104d257600080fd5b80630d253fbe116102cd5780631c31f710116102a75780631c31f7101461045a578063248a9ca31461046d5780632d668ce71461049057600080fd5b80630d253fbe146103f657806314d963161461040c5780631682e50b1461043357600080fd5b806301ffc9a7116102fe57806301ffc9a71461039f57806302b62938146103c257806304c584ad146103e357600080fd5b80627be2fe146103195780630152682d1461032e575b600080fd5b61032c610327366004613e9b565b61080c565b005b6101045461036590600781900b9067ffffffffffffffff600160401b8204811691600160801b8104821691600160c01b9091041684565b6040805160079590950b855267ffffffffffffffff9384166020860152918316918401919091521660608201526080015b60405180910390f35b6103b26103ad366004613ed4565b610ab2565b6040519015158152602001610396565b6103d56103d0366004613efe565b610af6565b604051908152602001610396565b6103d56103f1366004613f1b565b610b6f565b6103fe610bf0565b604051610396929190613f5c565b6103d57f3fb9f0655b78e8eabe9e0f51d65db56c7690d4329012c3faf1fbd6d43f65826181565b6103d57f6d8dad7188c7ed005c55bf77fbf589583d8668b0dad30a9b9dd016321a5c256f81565b61032c610468366004613efe565b610cb1565b6103d561047b366004613faa565b60009081526065602052604090206001015490565b6103b2610d54565b61032c6104a6366004613fc3565b610db0565b6103d57fc1b97c934675624ef2089089ac12ae8922988c11dc8a578dfbac10d9eecf476181565b61032c6104e0366004613fc3565b610dda565b610100546104f9906001600160a01b031681565b6040516001600160a01b039091168152602001610396565b61032c61051f366004614000565b610e66565b6103d5610532366004614064565b611145565b610101546104f9906001600160a01b031681565b610588610559366004613efe565b610106602052600090815260409020546001600160a01b03811690600160a01b900467ffffffffffffffff1682565b604080516001600160a01b03909316835267ffffffffffffffff909116602083015201610396565b61032c611275565b61032c6105c6366004614092565b6112c3565b6103d56105d9366004614064565b61142b565b6103d56105ec366004613efe565b611515565b6106046105ff3660046140cf565b61158e565b6040805167ffffffffffffffff938416815292909116602083015201610396565b6103d56101035481565b6103d57fb07567e7223e21f7dce4c0a89131ce9c32d0d3484085f3f331dea8caef56d14181565b61065e6115f1565b60405167ffffffffffffffff9091168152602001610396565b6104f96106853660046140ec565b611648565b6103b2610698366004613fc3565b60009182526065602090815260408084206001600160a01b0393909316845291905290205460ff1690565b61032c6106d136600461410e565b611660565b6103d5600081565b6103d57f19e6f23df7275b48d1c33822c6ad041a743378552246ac819f578ae1d6709cf981565b61032c611c2b565b61032c61071b366004613faa565b611cf3565b61032c61072e366004614121565b611d5e565b61032c611ec4565b6103d5610749366004613faa565b611f81565b61032c61075c366004613faa565b611f98565b6103d57f1d693f62a755e2b3c6494da41af454605b9006057cb3c79b6adda1378f2a50a781565b61032c610796366004613fc3565b612070565b6103d56101025481565b6103d56101055481565b6103d57fa8131bb4589277d6866d942849029b416b39e61eb3969a32787130bbdd292a9681565b6103b2612095565b61032c6107ec366004613faa565b61210a565b6103d561218b565b61032c610807366004614133565b612195565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152600061086282612348565b90508067ffffffffffffffff168467ffffffffffffffff1610156108cb576040517f395f4fd600000000000000000000000000000000000000000000000000000000815267ffffffffffffffff8086166004830152821660248201526044015b60405180910390fd5b60006108d860fe8661237e565b80546001600160a01b039081166000818152610106602052604090205492935091168015610968576001600160a01b0381163314610963576040517f7621d94a00000000000000000000000000000000000000000000000000000000815267ffffffffffffffff881660048201526001600160a01b03821660248201523360448201526064016108c2565b6109cb565b6001600160a01b03821633146109cb576040517f660af6d200000000000000000000000000000000000000000000000000000000815267ffffffffffffffff881660048201526001600160a01b03831660248201523360448201526064016108c2565b825473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0387161783556000806109fe878a61241d565b90925090506001600160a01b038316610a175783610a19565b825b6001600160a01b0316886001600160a01b0316856001600160a01b03167fb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b8c4267ffffffffffffffff168767ffffffffffffffff1610610a795786610a7b565b425b6040805167ffffffffffffffff938416815291831660208301529187168183015290519081900360600190a4505050505050505050565b60006001600160e01b031982167f5a05180f000000000000000000000000000000000000000000000000000000001480610af05750610af0826124b5565b92915050565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152600090610af090610b5190612348565b6001600160a01b038416600090815260fd602052604090209061251c565b604080517f0358b2b705d5c5ef47651be44f418326852a390f3b4c933661a5f4f0d8fa1ee3602082015267ffffffffffffffff8516918101919091526001600160a01b038316606082015260808101829052600090610be69060a00160405160208183030381529060405280519060200120612546565b90505b9392505050565b6040805180820190915260008082526020820152604080518082019091526000808252602082015260fe60010154600160a01b900467ffffffffffffffff1660fe60000154600160a01b900467ffffffffffffffff1611610c545760ff60fe610c59565b60fe60ff5b60408051808201825292546001600160a01b03808216855267ffffffffffffffff600160a01b928390048116602080880191909152845180860190955294549182168452919004169181019190915290939092509050565b7fc1b97c934675624ef2089089ac12ae8922988c11dc8a578dfbac10d9eecf4761610cdb816125af565b61010054604080516001600160a01b03928316815291841660208301527f8a0149b2f3ddf2c9ee85738165131d82babbb938f749321d59f75750afa7f4e6910160405180910390a150610100805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0392909216919091179055565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152600090610dab906125b9565b905090565b600082815260656020526040902060010154610dcb816125af565b610dd58383612625565b505050565b6001600160a01b0381163314610e585760405162461bcd60e51b815260206004820152602f60248201527f416363657373436f6e74726f6c3a2063616e206f6e6c792072656e6f756e636560448201527f20726f6c657320666f722073656c66000000000000000000000000000000000060648201526084016108c2565b610e628282612647565b5050565b7f1d693f62a755e2b3c6494da41af454605b9006057cb3c79b6adda1378f2a50a7610e90816125af565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152610ee4816125b9565b610f1a576040517fb9adeefd00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b826020013584602001351015610f5c576040517fa234cb1900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6101025483602001351015610faf57610102546040517f56f9b75a0000000000000000000000000000000000000000000000000000000081526020850135600482015260248101919091526044016108c2565b6000610fba82612348565b90506000610fc9826001614171565b9050600080610fe0610fda89614202565b84612669565b91509150600080610ffa89610ff490614202565b86612669565b91509150816001600160a01b0316846001600160a01b031603611049576040517ff4a3e48500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b88602001358a602001351480156110db57506040516bffffffffffffffffffffffff19606084901b1660208201526034810182905260540160408051808303601f190181529082905280516020918201206bffffffffffffffffffffffff19606088901b169183019190915260348201859052906054016040516020818303038152906040528051906020012060001c105b15611112576040517f9185a0ae00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008061111f898861241d565b9150915061113760018d888e602001358c8787612760565b505050505050505050505050565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b9004909116606082015260009061119c90612348565b67ffffffffffffffff168267ffffffffffffffff161015611253576040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152829061120d90612348565b6040517f395f4fd600000000000000000000000000000000000000000000000000000000815267ffffffffffffffff9283166004820152911660248201526044016108c2565b6001600160a01b038316600090815260fd60205260409020610be990836128b8565b61010554600081900361129b57604051631f2a200560e01b815260040160405180910390fd5b60006101055561010054610101546112c0916001600160a01b039182169116836128d0565b50565b7f1d693f62a755e2b3c6494da41af454605b9006057cb3c79b6adda1378f2a50a76112ed816125af565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152611341816125b9565b611377576040517fb9adeefd00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b61010254836020013510156113ca57610102546040517f56f9b75a0000000000000000000000000000000000000000000000000000000081526020850135600482015260248101919091526044016108c2565b60006113d582612348565b905060006113e4826001614171565b905060006113fa6113f487614202565b83612669565b50905060008061140a868561241d565b915091506114216000898561010254898787612760565b5050505050505050565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b9004909116606082015260009061148290612348565b67ffffffffffffffff168267ffffffffffffffff1610156114f3576040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152829061120d90612348565b6001600160a01b038316600090815260fd60205260409020610be9908361251c565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152600090610af09061157090612348565b6001600160a01b038416600090815260fd60205260409020906128b8565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b9004909116606082015260009081906115e8908461241d565b91509150915091565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152600090610dab90612348565b6000828152609760205260408120610be99083612961565b600054610100900460ff16158080156116805750600054600160ff909116105b8061169a5750303b15801561169a575060005460ff166001145b61170c5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016108c2565b6000805460ff19166001179055801561172f576000805461ff0019166101001790555b6001600160a01b037f00000000000000000000000000000000000000000000000000000000000000001630036117cd5760405162461bcd60e51b815260206004820152602c60248201527f46756e6374696f6e206d7573742062652063616c6c6564207468726f7567682060448201527f64656c656761746563616c6c000000000000000000000000000000000000000060648201526084016108c2565b6117d561296d565b6118496040518060400160405280601281526020017f457870726573734c616e6541756374696f6e00000000000000000000000000008152506040518060400160405280600181526020017f31000000000000000000000000000000000000000000000000000000000000008152506129ec565b600061185b6040840160208501613efe565b6001600160a01b03160361189b576040517f3fb3c7af00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6118ab6040830160208401613efe565b610101805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03929092169190911790556118e96060830160408401613efe565b610100805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03929092169190911790557f8a0149b2f3ddf2c9ee85738165131d82babbb938f749321d59f75750afa7f4e6600061194a6060850160408601613efe565b604080516001600160a01b0393841681529290911660208301520160405180910390a160e0820135610103819055604080516000815260208101929092527f5848068f11aa3ba9fe3fc33c5f9f2a3cd1aed67986b85b5e0cedc67dbe96f0f0910160405180910390a160e0820135610102819055604080516000815260208101929092527f9725e37e079c5bda6009a8f54d86265849f30acf61c630f9e1ac91e67de98794910160405180910390a16000611a0b60808401606085016142ca565b60070b1215611a46576040517f16f46dfe00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b611a5282606001612a73565b611a6e6000611a696101e085016101c08601613efe565b612625565b611aa47fb07567e7223e21f7dce4c0a89131ce9c32d0d3484085f3f331dea8caef56d141611a6961014085016101208601613efe565b611ada7fc1b97c934675624ef2089089ac12ae8922988c11dc8a578dfbac10d9eecf4761611a696101a085016101808601613efe565b611b437f1d693f62a755e2b3c6494da41af454605b9006057cb3c79b6adda1378f2a50a7611b0b6020850185613efe565b7f3fb9f0655b78e8eabe9e0f51d65db56c7690d4329012c3faf1fbd6d43f658261611b3e61012087016101008801613efe565b612c82565b611bac7f19e6f23df7275b48d1c33822c6ad041a743378552246ac819f578ae1d6709cf9611b7961016085016101408601613efe565b7fa8131bb4589277d6866d942849029b416b39e61eb3969a32787130bbdd292a96611b3e61018087016101608801613efe565b611be27f6d8dad7188c7ed005c55bf77fbf589583d8668b0dad30a9b9dd016321a5c256f611a696101c085016101a08601613efe565b8015610e62576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152600090611c8290612348565b611c8d906002614171565b33600090815260fd602052604090208054919250611cab9083612ca6565b6040805182815267ffffffffffffffff8416602082015233917f31f69201fab7912e3ec9850e3ab705964bf46d9d4276bdcbb6d05e965e5f5401910160405180910390a25050565b33600090815260fd60205260409020611d0c9082612d83565b61010154611d25906001600160a01b0316333084612df5565b60405181815233907fe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c906020015b60405180910390a250565b3360009081526101066020526040902080546001600160a01b031615801590611df157506040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152611dd690612348565b815467ffffffffffffffff918216600160a01b909104909116115b15611e3d5780546040517f75d899f2000000000000000000000000000000000000000000000000000000008152600160a01b90910467ffffffffffffffff1660048201526024016108c2565b336000908152610106602052604090208290611e5982826142e7565b50611e6990506020830183613efe565b6001600160a01b0316337ff6d28df235d9fa45a42d45dbb7c4f4ac76edb51e528f09f25a0650d32b8b33c0611ea460408601602087016140cf565b60405167ffffffffffffffff909116815260200160405180910390a35050565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152600090611f3490611f1f90612348565b33600090815260fd6020526040902090612e46565b61010154909150611f4f906001600160a01b031633836128d0565b60405181815233907f9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda890602001611d53565b6000818152609760205260408120610af090612ee8565b7f19e6f23df7275b48d1c33822c6ad041a743378552246ac819f578ae1d6709cf9611fc2816125af565b6000611fce60fe612ef2565b5080546040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b9004821660608201529293506120309291600160a01b900416612f36565b15612067576040517f4f00697800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b610dd583612fef565b60008281526065602052604090206001015461208b816125af565b610dd58383612647565b6000806120a260fe612ef2565b5080546040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b9004821660608201529293506121049291600160a01b900416612f36565b91505090565b7fb07567e7223e21f7dce4c0a89131ce9c32d0d3484085f3f331dea8caef56d141612134816125af565b6101035460408051918252602082018490527f5848068f11aa3ba9fe3fc33c5f9f2a3cd1aed67986b85b5e0cedc67dbe96f0f0910160405180910390a161010382905561010254821115610e6257610e6282612fef565b6000610dab61307e565b7f6d8dad7188c7ed005c55bf77fbf589583d8668b0dad30a9b9dd016321a5c256f6121bf816125af565b6040805160808101825261010454600781900b825267ffffffffffffffff600160401b820481166020840152600160801b8204811693830193909352600160c01b90049091166060820152600061221582612348565b9050600061223061222b36879003870187614363565b612348565b90508067ffffffffffffffff168267ffffffffffffffff1614612293576040517f68c18ca900000000000000000000000000000000000000000000000000000000815267ffffffffffffffff8084166004830152821660248201526044016108c2565b60006122aa6122a3846001614171565b859061241d565b50905060006122d26122bd846001614171565b6122cc368a90038a018a614363565b9061241d565b5090508067ffffffffffffffff168267ffffffffffffffff1614612336576040517fa0e269d800000000000000000000000000000000000000000000000000000000815267ffffffffffffffff8084166004830152821660248201526044016108c2565b61233f87612a73565b50505050505050565b8051600090600790810b4290910b121561236457506000919050565b60208201518251612374906130f9565b610af091906143fe565b600067ffffffffffffffff821683820154600160a01b900467ffffffffffffffff16036123b1578260005b019050610af0565b67ffffffffffffffff82168360010154600160a01b900467ffffffffffffffff16036123df578260016123a9565b6040517ffbb052d800000000000000000000000000000000000000000000000000000000815267ffffffffffffffff831660048201526024016108c2565b60008060008385602001516124329190614425565b855161243e9190614451565b905060008160070b1215612484576040517ff160ad79000000000000000000000000000000000000000000000000000000008152600782900b60048201526024016108c2565b6020850151819060009060019061249b9084614171565b6124a59190614480565b91945090925050505b9250929050565b60006001600160e01b031982167f7965db0b000000000000000000000000000000000000000000000000000000001480610af057507f01ffc9a7000000000000000000000000000000000000000000000000000000006001600160e01b0319831614610af0565b600182015460009067ffffffffffffffff9081169083161015612540576000610be9565b50505490565b6000610af061255361307e565b836040517f19010000000000000000000000000000000000000000000000000000000000006020820152602281018390526042810182905260009060620160405160208183030381529060405280519060200120905092915050565b6112c08133613105565b8051600090600790810b4290910b12156125d557506000919050565b60006125e483600001516130f9565b905060008360200151826125f891906144a1565b90508360400151846020015161260e9190614480565b67ffffffffffffffff908116911610159392505050565b61262f8282613185565b6000828152609760205260409020610dd59082613227565b612651828261323c565b6000828152609760205260409020610dd590826132bf565b60008060006126818486600001518760200151610b6f565b9050600061269c8660400151836132d490919063ffffffff16565b905060006126ab600187614480565b6020808901516001600160a01b038516600090815260fd9092526040909120919250906126d890836128b8565b1015612755576020808801516001600160a01b038416600090815260fd909252604090912083919061270a90846128b8565b6040517f36b24c140000000000000000000000000000000000000000000000000000000081526001600160a01b039093166004840152602483019190915260448201526064016108c2565b509590945092505050565b600061276d846001614171565b90506127898161278060208a018a613efe565b60fe91906132f8565b6001600160a01b038616600090815260fd602052604090206127ac9086866133e8565b8461010560008282546127bf91906144c8565b90915550600090506127d46020890189613efe565b6040805167ffffffffffffffff8581168252878116602083015286168183015290516001600160a01b0392909216916000917fb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b919081900360600190a461283e6020880188613efe565b6040805167ffffffffffffffff848116825260208b8101359083015281830189905286811660608301528516608082015290516001600160a01b03928316928916918b1515917f7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d479181900360a00190a45050505050505050565b60006128c4838361251c565b8354610be991906144db565b6040516001600160a01b038316602482015260448101829052610dd59084907fa9059cbb00000000000000000000000000000000000000000000000000000000906064015b60408051601f198184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff166001600160e01b031990931692909217909152613462565b6000610be98383613547565b600054610100900460ff166129ea5760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016108c2565b565b600054610100900460ff16612a695760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016108c2565b610e628282613571565b612a8360608201604083016140cf565b67ffffffffffffffff16600003612ac6576040517f047bad5200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b62015180612ada60408301602084016140cf565b67ffffffffffffffff161115612b3957612afa60408201602083016140cf565b6040517fc34a76cf00000000000000000000000000000000000000000000000000000000815267ffffffffffffffff90911660048201526024016108c2565b612b4960408201602083016140cf565b67ffffffffffffffff16612b6360608301604084016140cf565b612b7360808401606085016140cf565b612b7d9190614171565b67ffffffffffffffff161115612bbf576040517f326de36000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b80610104612bcd82826144ee565b507f982cfb73783b8c64455c76cdeb1351467c4f1e6b3615fec07df232c1b46ffd479050612c0361222b36849003840184614363565b612c1060208401846142ca565b612c2060408501602086016140cf565b612c3060608601604087016140cf565b612c4060808701606088016140cf565b6040805167ffffffffffffffff968716815260079590950b6020860152928516848401529084166060840152909216608082015290519081900360a00190a150565b612c8c8484612625565b612c968282612625565b612ca08483613608565b50505050565b8154600003612cc857604051631f2a200560e01b815260040160405180910390fd5b67fffffffffffffffe1967ffffffffffffffff821601612d14576040517f3d89ddde00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600182015467ffffffffffffffff90811614612d5c576040517f04eb6b3f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600191909101805467ffffffffffffffff191667ffffffffffffffff909216919091179055565b80600003612da457604051631f2a200560e01b815260040160405180910390fd5b600182015467ffffffffffffffff90811614612dd85760018201805467ffffffffffffffff191667ffffffffffffffff1790555b80826000016000828254612dec91906144c8565b90915550505050565b6040516001600160a01b0380851660248301528316604482015260648101829052612ca09085907f23b872dd0000000000000000000000000000000000000000000000000000000090608401612915565b600067fffffffffffffffe1967ffffffffffffffff831601612e94576040517f3d89ddde00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000612ea0848461251c565b905080600003612edc576040517fd0d04f6000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008455905092915050565b6000610af0825490565b60008080838101905060008460010154825467ffffffffffffffff600160a01b92839004811692909104161015612f2c5750506001808401905b9094909350915050565b8151600090600790810b4290910b1215612f5257506000610af0565b6000612f5d84612348565b9050612f6a816001614171565b67ffffffffffffffff168367ffffffffffffffff1610612f8e576000915050610af0565b6000612f9d85600001516130f9565b90506000856020015182612fb191906144a1565b9050856060015186604001518760200151612fcc9190614480565b612fd69190614480565b67ffffffffffffffff9081169116101595945050505050565b6101035481101561303b57610103546040517fda4f272e0000000000000000000000000000000000000000000000000000000081526108c2918391600401918252602082015260400190565b6101025460408051918252602082018390527f9725e37e079c5bda6009a8f54d86265849f30acf61c630f9e1ac91e67de98794910160405180910390a161010255565b6000610dab7f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f6130ad60c95490565b60ca546040805160208101859052908101839052606081018290524660808201523060a082015260009060c0016040516020818303038152906040528051906020012090509392505050565b6000610af082426145e4565b60008281526065602090815260408083206001600160a01b038516845290915290205460ff16610e6257613143816001600160a01b03166014613653565b61314e836020613653565b60405160200161315f929190614637565b60408051601f198184030181529082905262461bcd60e51b82526108c2916004016146b8565b60008281526065602090815260408083206001600160a01b038516845290915290205460ff16610e625760008281526065602090815260408083206001600160a01b03851684529091529020805460ff191660011790556131e33390565b6001600160a01b0316816001600160a01b0316837f2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d60405160405180910390a45050565b6000610be9836001600160a01b03841661387c565b60008281526065602090815260408083206001600160a01b038516845290915290205460ff1615610e625760008281526065602090815260408083206001600160a01b0385168085529252808320805460ff1916905551339285917ff6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b9190a45050565b6000610be9836001600160a01b0384166138cb565b60008060006132e385856139c5565b915091506132f081613a07565b509392505050565b60008061330485612ef2565b8154919350915067ffffffffffffffff808616600160a01b9092041610613363576040517f451f873400000000000000000000000000000000000000000000000000000000815267ffffffffffffffff851660048201526024016108c2565b604080518082019091526001600160a01b038416815267ffffffffffffffff8516602082015260018218908660ff8316600281106133a3576133a3614145565b82519101805460209093015167ffffffffffffffff16600160a01b026001600160e01b03199093166001600160a01b0390921691909117919091179055505050505050565b60006133f484836128b8565b905080158061340257508281105b15613443576040517fcf47918100000000000000000000000000000000000000000000000000000000815260048101849052602481018290526044016108c2565b8284600001600082825461345791906144db565b909155505050505050565b60006134b7826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564815250856001600160a01b0316613bf39092919063ffffffff16565b805190915015610dd557808060200190518101906134d591906146eb565b610dd55760405162461bcd60e51b815260206004820152602a60248201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e60448201527f6f7420737563636565640000000000000000000000000000000000000000000060648201526084016108c2565b600082600001828154811061355e5761355e614145565b9060005260206000200154905092915050565b600054610100900460ff166135ee5760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016108c2565b81516020928301208151919092012060c99190915560ca55565b600082815260656020526040808220600101805490849055905190918391839186917fbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff9190a4505050565b6060600061366283600261470d565b61366d9060026144c8565b67ffffffffffffffff81111561368557613685614192565b6040519080825280601f01601f1916602001820160405280156136af576020820181803683370190505b5090507f3000000000000000000000000000000000000000000000000000000000000000816000815181106136e6576136e6614145565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053507f78000000000000000000000000000000000000000000000000000000000000008160018151811061374957613749614145565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350600061378584600261470d565b6137909060016144c8565b90505b600181111561382d577f303132333435363738396162636465660000000000000000000000000000000085600f16601081106137d1576137d1614145565b1a60f81b8282815181106137e7576137e7614145565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a90535060049490941c9361382681614724565b9050613793565b508315610be95760405162461bcd60e51b815260206004820181905260248201527f537472696e67733a20686578206c656e67746820696e73756666696369656e7460448201526064016108c2565b60008181526001830160205260408120546138c357508154600181810184556000848152602080822090930184905584548482528286019093526040902091909155610af0565b506000610af0565b600081815260018301602052604081205480156139b45760006138ef6001836144db565b8554909150600090613903906001906144db565b905081811461396857600086600001828154811061392357613923614145565b906000526020600020015490508087600001848154811061394657613946614145565b6000918252602080832090910192909255918252600188019052604090208390555b855486908061397957613979614759565b600190038181906000526020600020016000905590558560010160008681526020019081526020016000206000905560019350505050610af0565b6000915050610af0565b5092915050565b60008082516041036139fb5760208301516040840151606085015160001a6139ef87828585613c02565b945094505050506124ae565b506000905060026124ae565b6000816004811115613a1b57613a1b61476f565b03613a235750565b6001816004811115613a3757613a3761476f565b03613a845760405162461bcd60e51b815260206004820152601860248201527f45434453413a20696e76616c6964207369676e6174757265000000000000000060448201526064016108c2565b6002816004811115613a9857613a9861476f565b03613ae55760405162461bcd60e51b815260206004820152601f60248201527f45434453413a20696e76616c6964207369676e6174757265206c656e6774680060448201526064016108c2565b6003816004811115613af957613af961476f565b03613b6c5760405162461bcd60e51b815260206004820152602260248201527f45434453413a20696e76616c6964207369676e6174757265202773272076616c60448201527f756500000000000000000000000000000000000000000000000000000000000060648201526084016108c2565b6004816004811115613b8057613b8061476f565b036112c05760405162461bcd60e51b815260206004820152602260248201527f45434453413a20696e76616c6964207369676e6174757265202776272076616c60448201527f756500000000000000000000000000000000000000000000000000000000000060648201526084016108c2565b6060610be68484600085613cef565b6000807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0831115613c395750600090506003613ce6565b8460ff16601b14158015613c5157508460ff16601c14155b15613c625750600090506004613ce6565b6040805160008082526020820180845289905260ff881692820192909252606081018690526080810185905260019060a0016020604051602081039080840390855afa158015613cb6573d6000803e3d6000fd5b5050604051601f1901519150506001600160a01b038116613cdf57600060019250925050613ce6565b9150600090505b94509492505050565b606082471015613d675760405162461bcd60e51b815260206004820152602660248201527f416464726573733a20696e73756666696369656e742062616c616e636520666f60448201527f722063616c6c000000000000000000000000000000000000000000000000000060648201526084016108c2565b6001600160a01b0385163b613dbe5760405162461bcd60e51b815260206004820152601d60248201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e747261637400000060448201526064016108c2565b600080866001600160a01b03168587604051613dda9190614785565b60006040518083038185875af1925050503d8060008114613e17576040519150601f19603f3d011682016040523d82523d6000602084013e613e1c565b606091505b5091509150613e2c828286613e37565b979650505050505050565b60608315613e46575081610be9565b825115613e565782518084602001fd5b8160405162461bcd60e51b81526004016108c291906146b8565b67ffffffffffffffff811681146112c057600080fd5b6001600160a01b03811681146112c057600080fd5b60008060408385031215613eae57600080fd5b8235613eb981613e70565b91506020830135613ec981613e86565b809150509250929050565b600060208284031215613ee657600080fd5b81356001600160e01b031981168114610be957600080fd5b600060208284031215613f1057600080fd5b8135610be981613e86565b600080600060608486031215613f3057600080fd5b8335613f3b81613e70565b92506020840135613f4b81613e86565b929592945050506040919091013590565b82516001600160a01b0316815260208084015167ffffffffffffffff16908201526080810182516001600160a01b03166040830152602083015167ffffffffffffffff166060830152610be9565b600060208284031215613fbc57600080fd5b5035919050565b60008060408385031215613fd657600080fd5b823591506020830135613ec981613e86565b600060608284031215613ffa57600080fd5b50919050565b6000806040838503121561401357600080fd5b823567ffffffffffffffff8082111561402b57600080fd5b61403786838701613fe8565b9350602085013591508082111561404d57600080fd5b5061405a85828601613fe8565b9150509250929050565b6000806040838503121561407757600080fd5b823561408281613e86565b91506020830135613ec981613e70565b6000602082840312156140a457600080fd5b813567ffffffffffffffff8111156140bb57600080fd5b6140c784828501613fe8565b949350505050565b6000602082840312156140e157600080fd5b8135610be981613e70565b600080604083850312156140ff57600080fd5b50508035926020909101359150565b60006101e08284031215613ffa57600080fd5b600060408284031215613ffa57600080fd5b600060808284031215613ffa57600080fd5b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052601160045260246000fd5b67ffffffffffffffff8181168382160190808211156139be576139be61415b565b634e487b7160e01b600052604160045260246000fd5b6040516060810167ffffffffffffffff811182821017156141cb576141cb614192565b60405290565b604051601f8201601f1916810167ffffffffffffffff811182821017156141fa576141fa614192565b604052919050565b60006060823603121561421457600080fd5b61421c6141a8565b823561422781613e86565b815260208381013581830152604084013567ffffffffffffffff8082111561424e57600080fd5b9085019036601f83011261426157600080fd5b81358181111561427357614273614192565b61428584601f19601f840116016141d1565b9150808252368482850101111561429b57600080fd5b808484018584013760009082019093019290925250604082015292915050565b8060070b81146112c057600080fd5b6000602082840312156142dc57600080fd5b8135610be9816142bb565b81356142f281613e86565b6001600160a01b038116905081548173ffffffffffffffffffffffffffffffffffffffff198216178355602084013561432a81613e70565b7bffffffffffffffff00000000000000000000000000000000000000008160a01b16836001600160e01b03198416171784555050505050565b60006080828403121561437557600080fd5b6040516080810181811067ffffffffffffffff8211171561439857614398614192565b60405282356143a6816142bb565b815260208301356143b681613e70565b602082015260408301356143c981613e70565b604082015260608301356143dc81613e70565b60608201529392505050565b634e487b7160e01b600052601260045260246000fd5b600067ffffffffffffffff80841680614419576144196143e8565b92169190910492915050565b67ffffffffffffffff8181168382160280821691908281146144495761444961415b565b505092915050565b600781810b9083900b01677fffffffffffffff8113677fffffffffffffff1982121715610af057610af061415b565b67ffffffffffffffff8281168282160390808211156139be576139be61415b565b600067ffffffffffffffff808416806144bc576144bc6143e8565b92169190910692915050565b80820180821115610af057610af061415b565b81810381811115610af057610af061415b565b81356144f9816142bb565b815467ffffffffffffffff82811667ffffffffffffffff198316178455602085013561452481613e70565b6fffffffffffffffff0000000000000000604091821b16919093167fffffffffffffffffffffffffffffffff000000000000000000000000000000008316811782178555928501359061457682613e70565b77ffffffffffffffff000000000000000000000000000000008260801b1691507fffffffffffffffff0000000000000000000000000000000000000000000000008285828616178317178655606087013593506145d284613e70565b93171760c09190911b90911617905550565b600782810b9082900b03677fffffffffffffff198112677fffffffffffffff82131715610af057610af061415b565b60005b8381101561462e578181015183820152602001614616565b50506000910152565b7f416363657373436f6e74726f6c3a206163636f756e742000000000000000000081526000835161466f816017850160208801614613565b7f206973206d697373696e6720726f6c652000000000000000000000000000000060179184019182015283516146ac816028840160208801614613565b01602801949350505050565b60208152600082518060208401526146d7816040850160208701614613565b601f01601f19169190910160400192915050565b6000602082840312156146fd57600080fd5b81518015158114610be957600080fd5b8082028115828204841417610af057610af061415b565b6000816147335761473361415b565b507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0190565b634e487b7160e01b600052603160045260246000fd5b634e487b7160e01b600052602160045260246000fd5b60008251614797818460208701614613565b919091019291505056fea2646970667358221220d28e36b250079b8bbd5de3a12e920ff46340db4c2ca3fd06c26dc1cd2796c9f264736f6c63430008110033",
}

// ExpressLaneAuctionABI is the input ABI used to generate the binding from.
// Deprecated: Use ExpressLaneAuctionMetaData.ABI instead.
var ExpressLaneAuctionABI = ExpressLaneAuctionMetaData.ABI

// ExpressLaneAuctionBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ExpressLaneAuctionMetaData.Bin instead.
var ExpressLaneAuctionBin = ExpressLaneAuctionMetaData.Bin

// DeployExpressLaneAuction deploys a new Ethereum contract, binding an instance of ExpressLaneAuction to it.
func DeployExpressLaneAuction(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ExpressLaneAuction, error) {
	parsed, err := ExpressLaneAuctionMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ExpressLaneAuctionBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ExpressLaneAuction{ExpressLaneAuctionCaller: ExpressLaneAuctionCaller{contract: contract}, ExpressLaneAuctionTransactor: ExpressLaneAuctionTransactor{contract: contract}, ExpressLaneAuctionFilterer: ExpressLaneAuctionFilterer{contract: contract}}, nil
}

// ExpressLaneAuction is an auto generated Go binding around an Ethereum contract.
type ExpressLaneAuction struct {
	ExpressLaneAuctionCaller     // Read-only binding to the contract
	ExpressLaneAuctionTransactor // Write-only binding to the contract
	ExpressLaneAuctionFilterer   // Log filterer for contract events
}

// ExpressLaneAuctionCaller is an auto generated read-only Go binding around an Ethereum contract.
type ExpressLaneAuctionCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExpressLaneAuctionTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ExpressLaneAuctionTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExpressLaneAuctionFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ExpressLaneAuctionFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExpressLaneAuctionSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ExpressLaneAuctionSession struct {
	Contract     *ExpressLaneAuction // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ExpressLaneAuctionCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ExpressLaneAuctionCallerSession struct {
	Contract *ExpressLaneAuctionCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// ExpressLaneAuctionTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ExpressLaneAuctionTransactorSession struct {
	Contract     *ExpressLaneAuctionTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// ExpressLaneAuctionRaw is an auto generated low-level Go binding around an Ethereum contract.
type ExpressLaneAuctionRaw struct {
	Contract *ExpressLaneAuction // Generic contract binding to access the raw methods on
}

// ExpressLaneAuctionCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ExpressLaneAuctionCallerRaw struct {
	Contract *ExpressLaneAuctionCaller // Generic read-only contract binding to access the raw methods on
}

// ExpressLaneAuctionTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ExpressLaneAuctionTransactorRaw struct {
	Contract *ExpressLaneAuctionTransactor // Generic write-only contract binding to access the raw methods on
}

// NewExpressLaneAuction creates a new instance of ExpressLaneAuction, bound to a specific deployed contract.
func NewExpressLaneAuction(address common.Address, backend bind.ContractBackend) (*ExpressLaneAuction, error) {
	contract, err := bindExpressLaneAuction(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuction{ExpressLaneAuctionCaller: ExpressLaneAuctionCaller{contract: contract}, ExpressLaneAuctionTransactor: ExpressLaneAuctionTransactor{contract: contract}, ExpressLaneAuctionFilterer: ExpressLaneAuctionFilterer{contract: contract}}, nil
}

// NewExpressLaneAuctionCaller creates a new read-only instance of ExpressLaneAuction, bound to a specific deployed contract.
func NewExpressLaneAuctionCaller(address common.Address, caller bind.ContractCaller) (*ExpressLaneAuctionCaller, error) {
	contract, err := bindExpressLaneAuction(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionCaller{contract: contract}, nil
}

// NewExpressLaneAuctionTransactor creates a new write-only instance of ExpressLaneAuction, bound to a specific deployed contract.
func NewExpressLaneAuctionTransactor(address common.Address, transactor bind.ContractTransactor) (*ExpressLaneAuctionTransactor, error) {
	contract, err := bindExpressLaneAuction(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionTransactor{contract: contract}, nil
}

// NewExpressLaneAuctionFilterer creates a new log filterer instance of ExpressLaneAuction, bound to a specific deployed contract.
func NewExpressLaneAuctionFilterer(address common.Address, filterer bind.ContractFilterer) (*ExpressLaneAuctionFilterer, error) {
	contract, err := bindExpressLaneAuction(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionFilterer{contract: contract}, nil
}

// bindExpressLaneAuction binds a generic wrapper to an already deployed contract.
func bindExpressLaneAuction(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ExpressLaneAuctionMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ExpressLaneAuction *ExpressLaneAuctionRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ExpressLaneAuction.Contract.ExpressLaneAuctionCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ExpressLaneAuction *ExpressLaneAuctionRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.ExpressLaneAuctionTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ExpressLaneAuction *ExpressLaneAuctionRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.ExpressLaneAuctionTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ExpressLaneAuction *ExpressLaneAuctionCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ExpressLaneAuction.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.contract.Transact(opts, method, params...)
}

// AUCTIONEERADMINROLE is a free data retrieval call binding the contract method 0x14d96316.
//
// Solidity: function AUCTIONEER_ADMIN_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) AUCTIONEERADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "AUCTIONEER_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// AUCTIONEERADMINROLE is a free data retrieval call binding the contract method 0x14d96316.
//
// Solidity: function AUCTIONEER_ADMIN_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) AUCTIONEERADMINROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.AUCTIONEERADMINROLE(&_ExpressLaneAuction.CallOpts)
}

// AUCTIONEERADMINROLE is a free data retrieval call binding the contract method 0x14d96316.
//
// Solidity: function AUCTIONEER_ADMIN_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) AUCTIONEERADMINROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.AUCTIONEERADMINROLE(&_ExpressLaneAuction.CallOpts)
}

// AUCTIONEERROLE is a free data retrieval call binding the contract method 0xcfe9232b.
//
// Solidity: function AUCTIONEER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) AUCTIONEERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "AUCTIONEER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// AUCTIONEERROLE is a free data retrieval call binding the contract method 0xcfe9232b.
//
// Solidity: function AUCTIONEER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) AUCTIONEERROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.AUCTIONEERROLE(&_ExpressLaneAuction.CallOpts)
}

// AUCTIONEERROLE is a free data retrieval call binding the contract method 0xcfe9232b.
//
// Solidity: function AUCTIONEER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) AUCTIONEERROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.AUCTIONEERROLE(&_ExpressLaneAuction.CallOpts)
}

// BENEFICIARYSETTERROLE is a free data retrieval call binding the contract method 0x336a5b5e.
//
// Solidity: function BENEFICIARY_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) BENEFICIARYSETTERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "BENEFICIARY_SETTER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// BENEFICIARYSETTERROLE is a free data retrieval call binding the contract method 0x336a5b5e.
//
// Solidity: function BENEFICIARY_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) BENEFICIARYSETTERROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.BENEFICIARYSETTERROLE(&_ExpressLaneAuction.CallOpts)
}

// BENEFICIARYSETTERROLE is a free data retrieval call binding the contract method 0x336a5b5e.
//
// Solidity: function BENEFICIARY_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) BENEFICIARYSETTERROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.BENEFICIARYSETTERROLE(&_ExpressLaneAuction.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.DEFAULTADMINROLE(&_ExpressLaneAuction.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.DEFAULTADMINROLE(&_ExpressLaneAuction.CallOpts)
}

// MINRESERVESETTERROLE is a free data retrieval call binding the contract method 0x8948cc4e.
//
// Solidity: function MIN_RESERVE_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) MINRESERVESETTERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "MIN_RESERVE_SETTER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// MINRESERVESETTERROLE is a free data retrieval call binding the contract method 0x8948cc4e.
//
// Solidity: function MIN_RESERVE_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) MINRESERVESETTERROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.MINRESERVESETTERROLE(&_ExpressLaneAuction.CallOpts)
}

// MINRESERVESETTERROLE is a free data retrieval call binding the contract method 0x8948cc4e.
//
// Solidity: function MIN_RESERVE_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) MINRESERVESETTERROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.MINRESERVESETTERROLE(&_ExpressLaneAuction.CallOpts)
}

// RESERVESETTERADMINROLE is a free data retrieval call binding the contract method 0xe3f7bb55.
//
// Solidity: function RESERVE_SETTER_ADMIN_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) RESERVESETTERADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "RESERVE_SETTER_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RESERVESETTERADMINROLE is a free data retrieval call binding the contract method 0xe3f7bb55.
//
// Solidity: function RESERVE_SETTER_ADMIN_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) RESERVESETTERADMINROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.RESERVESETTERADMINROLE(&_ExpressLaneAuction.CallOpts)
}

// RESERVESETTERADMINROLE is a free data retrieval call binding the contract method 0xe3f7bb55.
//
// Solidity: function RESERVE_SETTER_ADMIN_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) RESERVESETTERADMINROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.RESERVESETTERADMINROLE(&_ExpressLaneAuction.CallOpts)
}

// RESERVESETTERROLE is a free data retrieval call binding the contract method 0xb3ee252f.
//
// Solidity: function RESERVE_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) RESERVESETTERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "RESERVE_SETTER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RESERVESETTERROLE is a free data retrieval call binding the contract method 0xb3ee252f.
//
// Solidity: function RESERVE_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) RESERVESETTERROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.RESERVESETTERROLE(&_ExpressLaneAuction.CallOpts)
}

// RESERVESETTERROLE is a free data retrieval call binding the contract method 0xb3ee252f.
//
// Solidity: function RESERVE_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) RESERVESETTERROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.RESERVESETTERROLE(&_ExpressLaneAuction.CallOpts)
}

// ROUNDTIMINGSETTERROLE is a free data retrieval call binding the contract method 0x1682e50b.
//
// Solidity: function ROUND_TIMING_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) ROUNDTIMINGSETTERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "ROUND_TIMING_SETTER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ROUNDTIMINGSETTERROLE is a free data retrieval call binding the contract method 0x1682e50b.
//
// Solidity: function ROUND_TIMING_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) ROUNDTIMINGSETTERROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.ROUNDTIMINGSETTERROLE(&_ExpressLaneAuction.CallOpts)
}

// ROUNDTIMINGSETTERROLE is a free data retrieval call binding the contract method 0x1682e50b.
//
// Solidity: function ROUND_TIMING_SETTER_ROLE() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) ROUNDTIMINGSETTERROLE() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.ROUNDTIMINGSETTERROLE(&_ExpressLaneAuction.CallOpts)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.BalanceOf(&_ExpressLaneAuction.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.BalanceOf(&_ExpressLaneAuction.CallOpts, account)
}

// BalanceOfAtRound is a free data retrieval call binding the contract method 0x5633c337.
//
// Solidity: function balanceOfAtRound(address account, uint64 round) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) BalanceOfAtRound(opts *bind.CallOpts, account common.Address, round uint64) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "balanceOfAtRound", account, round)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOfAtRound is a free data retrieval call binding the contract method 0x5633c337.
//
// Solidity: function balanceOfAtRound(address account, uint64 round) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) BalanceOfAtRound(account common.Address, round uint64) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.BalanceOfAtRound(&_ExpressLaneAuction.CallOpts, account, round)
}

// BalanceOfAtRound is a free data retrieval call binding the contract method 0x5633c337.
//
// Solidity: function balanceOfAtRound(address account, uint64 round) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) BalanceOfAtRound(account common.Address, round uint64) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.BalanceOfAtRound(&_ExpressLaneAuction.CallOpts, account, round)
}

// Beneficiary is a free data retrieval call binding the contract method 0x38af3eed.
//
// Solidity: function beneficiary() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) Beneficiary(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "beneficiary")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Beneficiary is a free data retrieval call binding the contract method 0x38af3eed.
//
// Solidity: function beneficiary() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) Beneficiary() (common.Address, error) {
	return _ExpressLaneAuction.Contract.Beneficiary(&_ExpressLaneAuction.CallOpts)
}

// Beneficiary is a free data retrieval call binding the contract method 0x38af3eed.
//
// Solidity: function beneficiary() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) Beneficiary() (common.Address, error) {
	return _ExpressLaneAuction.Contract.Beneficiary(&_ExpressLaneAuction.CallOpts)
}

// BeneficiaryBalance is a free data retrieval call binding the contract method 0xe2fc6f68.
//
// Solidity: function beneficiaryBalance() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) BeneficiaryBalance(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "beneficiaryBalance")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BeneficiaryBalance is a free data retrieval call binding the contract method 0xe2fc6f68.
//
// Solidity: function beneficiaryBalance() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) BeneficiaryBalance() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.BeneficiaryBalance(&_ExpressLaneAuction.CallOpts)
}

// BeneficiaryBalance is a free data retrieval call binding the contract method 0xe2fc6f68.
//
// Solidity: function beneficiaryBalance() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) BeneficiaryBalance() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.BeneficiaryBalance(&_ExpressLaneAuction.CallOpts)
}

// BiddingToken is a free data retrieval call binding the contract method 0x639d7566.
//
// Solidity: function biddingToken() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) BiddingToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "biddingToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BiddingToken is a free data retrieval call binding the contract method 0x639d7566.
//
// Solidity: function biddingToken() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) BiddingToken() (common.Address, error) {
	return _ExpressLaneAuction.Contract.BiddingToken(&_ExpressLaneAuction.CallOpts)
}

// BiddingToken is a free data retrieval call binding the contract method 0x639d7566.
//
// Solidity: function biddingToken() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) BiddingToken() (common.Address, error) {
	return _ExpressLaneAuction.Contract.BiddingToken(&_ExpressLaneAuction.CallOpts)
}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint64)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) CurrentRound(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "currentRound")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint64)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) CurrentRound() (uint64, error) {
	return _ExpressLaneAuction.Contract.CurrentRound(&_ExpressLaneAuction.CallOpts)
}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint64)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) CurrentRound() (uint64, error) {
	return _ExpressLaneAuction.Contract.CurrentRound(&_ExpressLaneAuction.CallOpts)
}

// DomainSeparator is a free data retrieval call binding the contract method 0xf698da25.
//
// Solidity: function domainSeparator() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) DomainSeparator(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "domainSeparator")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DomainSeparator is a free data retrieval call binding the contract method 0xf698da25.
//
// Solidity: function domainSeparator() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) DomainSeparator() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.DomainSeparator(&_ExpressLaneAuction.CallOpts)
}

// DomainSeparator is a free data retrieval call binding the contract method 0xf698da25.
//
// Solidity: function domainSeparator() view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) DomainSeparator() ([32]byte, error) {
	return _ExpressLaneAuction.Contract.DomainSeparator(&_ExpressLaneAuction.CallOpts)
}

// GetBidHash is a free data retrieval call binding the contract method 0x04c584ad.
//
// Solidity: function getBidHash(uint64 round, address expressLaneController, uint256 amount) view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) GetBidHash(opts *bind.CallOpts, round uint64, expressLaneController common.Address, amount *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "getBidHash", round, expressLaneController, amount)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBidHash is a free data retrieval call binding the contract method 0x04c584ad.
//
// Solidity: function getBidHash(uint64 round, address expressLaneController, uint256 amount) view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) GetBidHash(round uint64, expressLaneController common.Address, amount *big.Int) ([32]byte, error) {
	return _ExpressLaneAuction.Contract.GetBidHash(&_ExpressLaneAuction.CallOpts, round, expressLaneController, amount)
}

// GetBidHash is a free data retrieval call binding the contract method 0x04c584ad.
//
// Solidity: function getBidHash(uint64 round, address expressLaneController, uint256 amount) view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) GetBidHash(round uint64, expressLaneController common.Address, amount *big.Int) ([32]byte, error) {
	return _ExpressLaneAuction.Contract.GetBidHash(&_ExpressLaneAuction.CallOpts, round, expressLaneController, amount)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _ExpressLaneAuction.Contract.GetRoleAdmin(&_ExpressLaneAuction.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _ExpressLaneAuction.Contract.GetRoleAdmin(&_ExpressLaneAuction.CallOpts, role)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) GetRoleMember(opts *bind.CallOpts, role [32]byte, index *big.Int) (common.Address, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "getRoleMember", role, index)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _ExpressLaneAuction.Contract.GetRoleMember(&_ExpressLaneAuction.CallOpts, role, index)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _ExpressLaneAuction.Contract.GetRoleMember(&_ExpressLaneAuction.CallOpts, role, index)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) GetRoleMemberCount(opts *bind.CallOpts, role [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "getRoleMemberCount", role)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.GetRoleMemberCount(&_ExpressLaneAuction.CallOpts, role)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.GetRoleMemberCount(&_ExpressLaneAuction.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _ExpressLaneAuction.Contract.HasRole(&_ExpressLaneAuction.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _ExpressLaneAuction.Contract.HasRole(&_ExpressLaneAuction.CallOpts, role, account)
}

// IsAuctionRoundClosed is a free data retrieval call binding the contract method 0x2d668ce7.
//
// Solidity: function isAuctionRoundClosed() view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) IsAuctionRoundClosed(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "isAuctionRoundClosed")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAuctionRoundClosed is a free data retrieval call binding the contract method 0x2d668ce7.
//
// Solidity: function isAuctionRoundClosed() view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) IsAuctionRoundClosed() (bool, error) {
	return _ExpressLaneAuction.Contract.IsAuctionRoundClosed(&_ExpressLaneAuction.CallOpts)
}

// IsAuctionRoundClosed is a free data retrieval call binding the contract method 0x2d668ce7.
//
// Solidity: function isAuctionRoundClosed() view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) IsAuctionRoundClosed() (bool, error) {
	return _ExpressLaneAuction.Contract.IsAuctionRoundClosed(&_ExpressLaneAuction.CallOpts)
}

// IsReserveBlackout is a free data retrieval call binding the contract method 0xe460d2c5.
//
// Solidity: function isReserveBlackout() view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) IsReserveBlackout(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "isReserveBlackout")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsReserveBlackout is a free data retrieval call binding the contract method 0xe460d2c5.
//
// Solidity: function isReserveBlackout() view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) IsReserveBlackout() (bool, error) {
	return _ExpressLaneAuction.Contract.IsReserveBlackout(&_ExpressLaneAuction.CallOpts)
}

// IsReserveBlackout is a free data retrieval call binding the contract method 0xe460d2c5.
//
// Solidity: function isReserveBlackout() view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) IsReserveBlackout() (bool, error) {
	return _ExpressLaneAuction.Contract.IsReserveBlackout(&_ExpressLaneAuction.CallOpts)
}

// MinReservePrice is a free data retrieval call binding the contract method 0x83af0a1f.
//
// Solidity: function minReservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) MinReservePrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "minReservePrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinReservePrice is a free data retrieval call binding the contract method 0x83af0a1f.
//
// Solidity: function minReservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) MinReservePrice() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.MinReservePrice(&_ExpressLaneAuction.CallOpts)
}

// MinReservePrice is a free data retrieval call binding the contract method 0x83af0a1f.
//
// Solidity: function minReservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) MinReservePrice() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.MinReservePrice(&_ExpressLaneAuction.CallOpts)
}

// ReservePrice is a free data retrieval call binding the contract method 0xdb2e1eed.
//
// Solidity: function reservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) ReservePrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "reservePrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ReservePrice is a free data retrieval call binding the contract method 0xdb2e1eed.
//
// Solidity: function reservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) ReservePrice() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.ReservePrice(&_ExpressLaneAuction.CallOpts)
}

// ReservePrice is a free data retrieval call binding the contract method 0xdb2e1eed.
//
// Solidity: function reservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) ReservePrice() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.ReservePrice(&_ExpressLaneAuction.CallOpts)
}

// ResolvedRounds is a free data retrieval call binding the contract method 0x0d253fbe.
//
// Solidity: function resolvedRounds() view returns((address,uint64), (address,uint64))
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) ResolvedRounds(opts *bind.CallOpts) (ELCRound, ELCRound, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "resolvedRounds")

	if err != nil {
		return *new(ELCRound), *new(ELCRound), err
	}

	out0 := *abi.ConvertType(out[0], new(ELCRound)).(*ELCRound)
	out1 := *abi.ConvertType(out[1], new(ELCRound)).(*ELCRound)

	return out0, out1, err

}

// ResolvedRounds is a free data retrieval call binding the contract method 0x0d253fbe.
//
// Solidity: function resolvedRounds() view returns((address,uint64), (address,uint64))
func (_ExpressLaneAuction *ExpressLaneAuctionSession) ResolvedRounds() (ELCRound, ELCRound, error) {
	return _ExpressLaneAuction.Contract.ResolvedRounds(&_ExpressLaneAuction.CallOpts)
}

// ResolvedRounds is a free data retrieval call binding the contract method 0x0d253fbe.
//
// Solidity: function resolvedRounds() view returns((address,uint64), (address,uint64))
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) ResolvedRounds() (ELCRound, ELCRound, error) {
	return _ExpressLaneAuction.Contract.ResolvedRounds(&_ExpressLaneAuction.CallOpts)
}

// RoundTimestamps is a free data retrieval call binding the contract method 0x7b617f94.
//
// Solidity: function roundTimestamps(uint64 round) view returns(uint64, uint64)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) RoundTimestamps(opts *bind.CallOpts, round uint64) (uint64, uint64, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "roundTimestamps", round)

	if err != nil {
		return *new(uint64), *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)
	out1 := *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return out0, out1, err

}

// RoundTimestamps is a free data retrieval call binding the contract method 0x7b617f94.
//
// Solidity: function roundTimestamps(uint64 round) view returns(uint64, uint64)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) RoundTimestamps(round uint64) (uint64, uint64, error) {
	return _ExpressLaneAuction.Contract.RoundTimestamps(&_ExpressLaneAuction.CallOpts, round)
}

// RoundTimestamps is a free data retrieval call binding the contract method 0x7b617f94.
//
// Solidity: function roundTimestamps(uint64 round) view returns(uint64, uint64)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) RoundTimestamps(round uint64) (uint64, uint64, error) {
	return _ExpressLaneAuction.Contract.RoundTimestamps(&_ExpressLaneAuction.CallOpts, round)
}

// RoundTimingInfo is a free data retrieval call binding the contract method 0x0152682d.
//
// Solidity: function roundTimingInfo() view returns(int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) RoundTimingInfo(opts *bind.CallOpts) (struct {
	OffsetTimestamp          int64
	RoundDurationSeconds     uint64
	AuctionClosingSeconds    uint64
	ReserveSubmissionSeconds uint64
}, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "roundTimingInfo")

	outstruct := new(struct {
		OffsetTimestamp          int64
		RoundDurationSeconds     uint64
		AuctionClosingSeconds    uint64
		ReserveSubmissionSeconds uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.OffsetTimestamp = *abi.ConvertType(out[0], new(int64)).(*int64)
	outstruct.RoundDurationSeconds = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.AuctionClosingSeconds = *abi.ConvertType(out[2], new(uint64)).(*uint64)
	outstruct.ReserveSubmissionSeconds = *abi.ConvertType(out[3], new(uint64)).(*uint64)

	return *outstruct, err

}

// RoundTimingInfo is a free data retrieval call binding the contract method 0x0152682d.
//
// Solidity: function roundTimingInfo() view returns(int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) RoundTimingInfo() (struct {
	OffsetTimestamp          int64
	RoundDurationSeconds     uint64
	AuctionClosingSeconds    uint64
	ReserveSubmissionSeconds uint64
}, error) {
	return _ExpressLaneAuction.Contract.RoundTimingInfo(&_ExpressLaneAuction.CallOpts)
}

// RoundTimingInfo is a free data retrieval call binding the contract method 0x0152682d.
//
// Solidity: function roundTimingInfo() view returns(int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) RoundTimingInfo() (struct {
	OffsetTimestamp          int64
	RoundDurationSeconds     uint64
	AuctionClosingSeconds    uint64
	ReserveSubmissionSeconds uint64
}, error) {
	return _ExpressLaneAuction.Contract.RoundTimingInfo(&_ExpressLaneAuction.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ExpressLaneAuction.Contract.SupportsInterface(&_ExpressLaneAuction.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ExpressLaneAuction.Contract.SupportsInterface(&_ExpressLaneAuction.CallOpts, interfaceId)
}

// TransferorOf is a free data retrieval call binding the contract method 0x6a514beb.
//
// Solidity: function transferorOf(address ) view returns(address addr, uint64 fixedUntilRound)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) TransferorOf(opts *bind.CallOpts, arg0 common.Address) (struct {
	Addr            common.Address
	FixedUntilRound uint64
}, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "transferorOf", arg0)

	outstruct := new(struct {
		Addr            common.Address
		FixedUntilRound uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Addr = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.FixedUntilRound = *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return *outstruct, err

}

// TransferorOf is a free data retrieval call binding the contract method 0x6a514beb.
//
// Solidity: function transferorOf(address ) view returns(address addr, uint64 fixedUntilRound)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) TransferorOf(arg0 common.Address) (struct {
	Addr            common.Address
	FixedUntilRound uint64
}, error) {
	return _ExpressLaneAuction.Contract.TransferorOf(&_ExpressLaneAuction.CallOpts, arg0)
}

// TransferorOf is a free data retrieval call binding the contract method 0x6a514beb.
//
// Solidity: function transferorOf(address ) view returns(address addr, uint64 fixedUntilRound)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) TransferorOf(arg0 common.Address) (struct {
	Addr            common.Address
	FixedUntilRound uint64
}, error) {
	return _ExpressLaneAuction.Contract.TransferorOf(&_ExpressLaneAuction.CallOpts, arg0)
}

// WithdrawableBalance is a free data retrieval call binding the contract method 0x02b62938.
//
// Solidity: function withdrawableBalance(address account) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) WithdrawableBalance(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "withdrawableBalance", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// WithdrawableBalance is a free data retrieval call binding the contract method 0x02b62938.
//
// Solidity: function withdrawableBalance(address account) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) WithdrawableBalance(account common.Address) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.WithdrawableBalance(&_ExpressLaneAuction.CallOpts, account)
}

// WithdrawableBalance is a free data retrieval call binding the contract method 0x02b62938.
//
// Solidity: function withdrawableBalance(address account) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) WithdrawableBalance(account common.Address) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.WithdrawableBalance(&_ExpressLaneAuction.CallOpts, account)
}

// WithdrawableBalanceAtRound is a free data retrieval call binding the contract method 0x6e8cace5.
//
// Solidity: function withdrawableBalanceAtRound(address account, uint64 round) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) WithdrawableBalanceAtRound(opts *bind.CallOpts, account common.Address, round uint64) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "withdrawableBalanceAtRound", account, round)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// WithdrawableBalanceAtRound is a free data retrieval call binding the contract method 0x6e8cace5.
//
// Solidity: function withdrawableBalanceAtRound(address account, uint64 round) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) WithdrawableBalanceAtRound(account common.Address, round uint64) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.WithdrawableBalanceAtRound(&_ExpressLaneAuction.CallOpts, account, round)
}

// WithdrawableBalanceAtRound is a free data retrieval call binding the contract method 0x6e8cace5.
//
// Solidity: function withdrawableBalanceAtRound(address account, uint64 round) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) WithdrawableBalanceAtRound(account common.Address, round uint64) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.WithdrawableBalanceAtRound(&_ExpressLaneAuction.CallOpts, account, round)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(uint256 amount) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) Deposit(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "deposit", amount)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(uint256 amount) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) Deposit(amount *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.Deposit(&_ExpressLaneAuction.TransactOpts, amount)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(uint256 amount) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) Deposit(amount *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.Deposit(&_ExpressLaneAuction.TransactOpts, amount)
}

// FinalizeWithdrawal is a paid mutator transaction binding the contract method 0xc5b6aa2f.
//
// Solidity: function finalizeWithdrawal() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) FinalizeWithdrawal(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "finalizeWithdrawal")
}

// FinalizeWithdrawal is a paid mutator transaction binding the contract method 0xc5b6aa2f.
//
// Solidity: function finalizeWithdrawal() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) FinalizeWithdrawal() (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.FinalizeWithdrawal(&_ExpressLaneAuction.TransactOpts)
}

// FinalizeWithdrawal is a paid mutator transaction binding the contract method 0xc5b6aa2f.
//
// Solidity: function finalizeWithdrawal() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) FinalizeWithdrawal() (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.FinalizeWithdrawal(&_ExpressLaneAuction.TransactOpts)
}

// FlushBeneficiaryBalance is a paid mutator transaction binding the contract method 0x6ad72517.
//
// Solidity: function flushBeneficiaryBalance() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) FlushBeneficiaryBalance(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "flushBeneficiaryBalance")
}

// FlushBeneficiaryBalance is a paid mutator transaction binding the contract method 0x6ad72517.
//
// Solidity: function flushBeneficiaryBalance() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) FlushBeneficiaryBalance() (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.FlushBeneficiaryBalance(&_ExpressLaneAuction.TransactOpts)
}

// FlushBeneficiaryBalance is a paid mutator transaction binding the contract method 0x6ad72517.
//
// Solidity: function flushBeneficiaryBalance() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) FlushBeneficiaryBalance() (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.FlushBeneficiaryBalance(&_ExpressLaneAuction.TransactOpts)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.GrantRole(&_ExpressLaneAuction.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.GrantRole(&_ExpressLaneAuction.TransactOpts, role, account)
}

// Initialize is a paid mutator transaction binding the contract method 0x9a1fadd3.
//
// Solidity: function initialize((address,address,address,(int64,uint64,uint64,uint64),uint256,address,address,address,address,address,address,address) args) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) Initialize(opts *bind.TransactOpts, args InitArgs) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "initialize", args)
}

// Initialize is a paid mutator transaction binding the contract method 0x9a1fadd3.
//
// Solidity: function initialize((address,address,address,(int64,uint64,uint64,uint64),uint256,address,address,address,address,address,address,address) args) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) Initialize(args InitArgs) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.Initialize(&_ExpressLaneAuction.TransactOpts, args)
}

// Initialize is a paid mutator transaction binding the contract method 0x9a1fadd3.
//
// Solidity: function initialize((address,address,address,(int64,uint64,uint64,uint64),uint256,address,address,address,address,address,address,address) args) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) Initialize(args InitArgs) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.Initialize(&_ExpressLaneAuction.TransactOpts, args)
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0xb51d1d4f.
//
// Solidity: function initiateWithdrawal() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) InitiateWithdrawal(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "initiateWithdrawal")
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0xb51d1d4f.
//
// Solidity: function initiateWithdrawal() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) InitiateWithdrawal() (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.InitiateWithdrawal(&_ExpressLaneAuction.TransactOpts)
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0xb51d1d4f.
//
// Solidity: function initiateWithdrawal() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) InitiateWithdrawal() (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.InitiateWithdrawal(&_ExpressLaneAuction.TransactOpts)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.RenounceRole(&_ExpressLaneAuction.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.RenounceRole(&_ExpressLaneAuction.TransactOpts, role, account)
}

// ResolveMultiBidAuction is a paid mutator transaction binding the contract method 0x447a709e.
//
// Solidity: function resolveMultiBidAuction((address,uint256,bytes) firstPriceBid, (address,uint256,bytes) secondPriceBid) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) ResolveMultiBidAuction(opts *bind.TransactOpts, firstPriceBid Bid, secondPriceBid Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "resolveMultiBidAuction", firstPriceBid, secondPriceBid)
}

// ResolveMultiBidAuction is a paid mutator transaction binding the contract method 0x447a709e.
//
// Solidity: function resolveMultiBidAuction((address,uint256,bytes) firstPriceBid, (address,uint256,bytes) secondPriceBid) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) ResolveMultiBidAuction(firstPriceBid Bid, secondPriceBid Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.ResolveMultiBidAuction(&_ExpressLaneAuction.TransactOpts, firstPriceBid, secondPriceBid)
}

// ResolveMultiBidAuction is a paid mutator transaction binding the contract method 0x447a709e.
//
// Solidity: function resolveMultiBidAuction((address,uint256,bytes) firstPriceBid, (address,uint256,bytes) secondPriceBid) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) ResolveMultiBidAuction(firstPriceBid Bid, secondPriceBid Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.ResolveMultiBidAuction(&_ExpressLaneAuction.TransactOpts, firstPriceBid, secondPriceBid)
}

// ResolveSingleBidAuction is a paid mutator transaction binding the contract method 0x6dc4fc4e.
//
// Solidity: function resolveSingleBidAuction((address,uint256,bytes) firstPriceBid) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) ResolveSingleBidAuction(opts *bind.TransactOpts, firstPriceBid Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "resolveSingleBidAuction", firstPriceBid)
}

// ResolveSingleBidAuction is a paid mutator transaction binding the contract method 0x6dc4fc4e.
//
// Solidity: function resolveSingleBidAuction((address,uint256,bytes) firstPriceBid) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) ResolveSingleBidAuction(firstPriceBid Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.ResolveSingleBidAuction(&_ExpressLaneAuction.TransactOpts, firstPriceBid)
}

// ResolveSingleBidAuction is a paid mutator transaction binding the contract method 0x6dc4fc4e.
//
// Solidity: function resolveSingleBidAuction((address,uint256,bytes) firstPriceBid) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) ResolveSingleBidAuction(firstPriceBid Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.ResolveSingleBidAuction(&_ExpressLaneAuction.TransactOpts, firstPriceBid)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.RevokeRole(&_ExpressLaneAuction.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.RevokeRole(&_ExpressLaneAuction.TransactOpts, role, account)
}

// SetBeneficiary is a paid mutator transaction binding the contract method 0x1c31f710.
//
// Solidity: function setBeneficiary(address newBeneficiary) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) SetBeneficiary(opts *bind.TransactOpts, newBeneficiary common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "setBeneficiary", newBeneficiary)
}

// SetBeneficiary is a paid mutator transaction binding the contract method 0x1c31f710.
//
// Solidity: function setBeneficiary(address newBeneficiary) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) SetBeneficiary(newBeneficiary common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetBeneficiary(&_ExpressLaneAuction.TransactOpts, newBeneficiary)
}

// SetBeneficiary is a paid mutator transaction binding the contract method 0x1c31f710.
//
// Solidity: function setBeneficiary(address newBeneficiary) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) SetBeneficiary(newBeneficiary common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetBeneficiary(&_ExpressLaneAuction.TransactOpts, newBeneficiary)
}

// SetMinReservePrice is a paid mutator transaction binding the contract method 0xe4d20c1d.
//
// Solidity: function setMinReservePrice(uint256 newMinReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) SetMinReservePrice(opts *bind.TransactOpts, newMinReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "setMinReservePrice", newMinReservePrice)
}

// SetMinReservePrice is a paid mutator transaction binding the contract method 0xe4d20c1d.
//
// Solidity: function setMinReservePrice(uint256 newMinReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) SetMinReservePrice(newMinReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetMinReservePrice(&_ExpressLaneAuction.TransactOpts, newMinReservePrice)
}

// SetMinReservePrice is a paid mutator transaction binding the contract method 0xe4d20c1d.
//
// Solidity: function setMinReservePrice(uint256 newMinReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) SetMinReservePrice(newMinReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetMinReservePrice(&_ExpressLaneAuction.TransactOpts, newMinReservePrice)
}

// SetReservePrice is a paid mutator transaction binding the contract method 0xce9c7c0d.
//
// Solidity: function setReservePrice(uint256 newReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) SetReservePrice(opts *bind.TransactOpts, newReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "setReservePrice", newReservePrice)
}

// SetReservePrice is a paid mutator transaction binding the contract method 0xce9c7c0d.
//
// Solidity: function setReservePrice(uint256 newReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) SetReservePrice(newReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetReservePrice(&_ExpressLaneAuction.TransactOpts, newReservePrice)
}

// SetReservePrice is a paid mutator transaction binding the contract method 0xce9c7c0d.
//
// Solidity: function setReservePrice(uint256 newReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) SetReservePrice(newReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetReservePrice(&_ExpressLaneAuction.TransactOpts, newReservePrice)
}

// SetRoundTimingInfo is a paid mutator transaction binding the contract method 0xfed87be8.
//
// Solidity: function setRoundTimingInfo((int64,uint64,uint64,uint64) newRoundTimingInfo) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) SetRoundTimingInfo(opts *bind.TransactOpts, newRoundTimingInfo RoundTimingInfo) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "setRoundTimingInfo", newRoundTimingInfo)
}

// SetRoundTimingInfo is a paid mutator transaction binding the contract method 0xfed87be8.
//
// Solidity: function setRoundTimingInfo((int64,uint64,uint64,uint64) newRoundTimingInfo) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) SetRoundTimingInfo(newRoundTimingInfo RoundTimingInfo) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetRoundTimingInfo(&_ExpressLaneAuction.TransactOpts, newRoundTimingInfo)
}

// SetRoundTimingInfo is a paid mutator transaction binding the contract method 0xfed87be8.
//
// Solidity: function setRoundTimingInfo((int64,uint64,uint64,uint64) newRoundTimingInfo) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) SetRoundTimingInfo(newRoundTimingInfo RoundTimingInfo) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetRoundTimingInfo(&_ExpressLaneAuction.TransactOpts, newRoundTimingInfo)
}

// SetTransferor is a paid mutator transaction binding the contract method 0xbef0ec74.
//
// Solidity: function setTransferor((address,uint64) transferor) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) SetTransferor(opts *bind.TransactOpts, transferor Transferor) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "setTransferor", transferor)
}

// SetTransferor is a paid mutator transaction binding the contract method 0xbef0ec74.
//
// Solidity: function setTransferor((address,uint64) transferor) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) SetTransferor(transferor Transferor) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetTransferor(&_ExpressLaneAuction.TransactOpts, transferor)
}

// SetTransferor is a paid mutator transaction binding the contract method 0xbef0ec74.
//
// Solidity: function setTransferor((address,uint64) transferor) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) SetTransferor(transferor Transferor) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetTransferor(&_ExpressLaneAuction.TransactOpts, transferor)
}

// TransferExpressLaneController is a paid mutator transaction binding the contract method 0x007be2fe.
//
// Solidity: function transferExpressLaneController(uint64 round, address newExpressLaneController) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) TransferExpressLaneController(opts *bind.TransactOpts, round uint64, newExpressLaneController common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "transferExpressLaneController", round, newExpressLaneController)
}

// TransferExpressLaneController is a paid mutator transaction binding the contract method 0x007be2fe.
//
// Solidity: function transferExpressLaneController(uint64 round, address newExpressLaneController) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) TransferExpressLaneController(round uint64, newExpressLaneController common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.TransferExpressLaneController(&_ExpressLaneAuction.TransactOpts, round, newExpressLaneController)
}

// TransferExpressLaneController is a paid mutator transaction binding the contract method 0x007be2fe.
//
// Solidity: function transferExpressLaneController(uint64 round, address newExpressLaneController) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) TransferExpressLaneController(round uint64, newExpressLaneController common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.TransferExpressLaneController(&_ExpressLaneAuction.TransactOpts, round, newExpressLaneController)
}

// ExpressLaneAuctionAuctionResolvedIterator is returned from FilterAuctionResolved and is used to iterate over the raw logs and unpacked data for AuctionResolved events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionAuctionResolvedIterator struct {
	Event *ExpressLaneAuctionAuctionResolved // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionAuctionResolvedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionAuctionResolved)
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
		it.Event = new(ExpressLaneAuctionAuctionResolved)
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
func (it *ExpressLaneAuctionAuctionResolvedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionAuctionResolvedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionAuctionResolved represents a AuctionResolved event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionAuctionResolved struct {
	IsMultiBidAuction               bool
	Round                           uint64
	FirstPriceBidder                common.Address
	FirstPriceExpressLaneController common.Address
	FirstPriceAmount                *big.Int
	Price                           *big.Int
	RoundStartTimestamp             uint64
	RoundEndTimestamp               uint64
	Raw                             types.Log // Blockchain specific contextual infos
}

// FilterAuctionResolved is a free log retrieval operation binding the contract event 0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47.
//
// Solidity: event AuctionResolved(bool indexed isMultiBidAuction, uint64 round, address indexed firstPriceBidder, address indexed firstPriceExpressLaneController, uint256 firstPriceAmount, uint256 price, uint64 roundStartTimestamp, uint64 roundEndTimestamp)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterAuctionResolved(opts *bind.FilterOpts, isMultiBidAuction []bool, firstPriceBidder []common.Address, firstPriceExpressLaneController []common.Address) (*ExpressLaneAuctionAuctionResolvedIterator, error) {

	var isMultiBidAuctionRule []interface{}
	for _, isMultiBidAuctionItem := range isMultiBidAuction {
		isMultiBidAuctionRule = append(isMultiBidAuctionRule, isMultiBidAuctionItem)
	}

	var firstPriceBidderRule []interface{}
	for _, firstPriceBidderItem := range firstPriceBidder {
		firstPriceBidderRule = append(firstPriceBidderRule, firstPriceBidderItem)
	}
	var firstPriceExpressLaneControllerRule []interface{}
	for _, firstPriceExpressLaneControllerItem := range firstPriceExpressLaneController {
		firstPriceExpressLaneControllerRule = append(firstPriceExpressLaneControllerRule, firstPriceExpressLaneControllerItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "AuctionResolved", isMultiBidAuctionRule, firstPriceBidderRule, firstPriceExpressLaneControllerRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionAuctionResolvedIterator{contract: _ExpressLaneAuction.contract, event: "AuctionResolved", logs: logs, sub: sub}, nil
}

// WatchAuctionResolved is a free log subscription operation binding the contract event 0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47.
//
// Solidity: event AuctionResolved(bool indexed isMultiBidAuction, uint64 round, address indexed firstPriceBidder, address indexed firstPriceExpressLaneController, uint256 firstPriceAmount, uint256 price, uint64 roundStartTimestamp, uint64 roundEndTimestamp)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchAuctionResolved(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionAuctionResolved, isMultiBidAuction []bool, firstPriceBidder []common.Address, firstPriceExpressLaneController []common.Address) (event.Subscription, error) {

	var isMultiBidAuctionRule []interface{}
	for _, isMultiBidAuctionItem := range isMultiBidAuction {
		isMultiBidAuctionRule = append(isMultiBidAuctionRule, isMultiBidAuctionItem)
	}

	var firstPriceBidderRule []interface{}
	for _, firstPriceBidderItem := range firstPriceBidder {
		firstPriceBidderRule = append(firstPriceBidderRule, firstPriceBidderItem)
	}
	var firstPriceExpressLaneControllerRule []interface{}
	for _, firstPriceExpressLaneControllerItem := range firstPriceExpressLaneController {
		firstPriceExpressLaneControllerRule = append(firstPriceExpressLaneControllerRule, firstPriceExpressLaneControllerItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "AuctionResolved", isMultiBidAuctionRule, firstPriceBidderRule, firstPriceExpressLaneControllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionAuctionResolved)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "AuctionResolved", log); err != nil {
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

// ParseAuctionResolved is a log parse operation binding the contract event 0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47.
//
// Solidity: event AuctionResolved(bool indexed isMultiBidAuction, uint64 round, address indexed firstPriceBidder, address indexed firstPriceExpressLaneController, uint256 firstPriceAmount, uint256 price, uint64 roundStartTimestamp, uint64 roundEndTimestamp)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseAuctionResolved(log types.Log) (*ExpressLaneAuctionAuctionResolved, error) {
	event := new(ExpressLaneAuctionAuctionResolved)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "AuctionResolved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionDepositIterator is returned from FilterDeposit and is used to iterate over the raw logs and unpacked data for Deposit events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionDepositIterator struct {
	Event *ExpressLaneAuctionDeposit // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionDeposit)
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
		it.Event = new(ExpressLaneAuctionDeposit)
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
func (it *ExpressLaneAuctionDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionDeposit represents a Deposit event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionDeposit struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterDeposit is a free log retrieval operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed account, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterDeposit(opts *bind.FilterOpts, account []common.Address) (*ExpressLaneAuctionDepositIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "Deposit", accountRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionDepositIterator{contract: _ExpressLaneAuction.contract, event: "Deposit", logs: logs, sub: sub}, nil
}

// WatchDeposit is a free log subscription operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed account, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchDeposit(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionDeposit, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "Deposit", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionDeposit)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "Deposit", log); err != nil {
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

// ParseDeposit is a log parse operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed account, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseDeposit(log types.Log) (*ExpressLaneAuctionDeposit, error) {
	event := new(ExpressLaneAuctionDeposit)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "Deposit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionInitializedIterator struct {
	Event *ExpressLaneAuctionInitialized // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionInitialized)
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
		it.Event = new(ExpressLaneAuctionInitialized)
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
func (it *ExpressLaneAuctionInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionInitialized represents a Initialized event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterInitialized(opts *bind.FilterOpts) (*ExpressLaneAuctionInitializedIterator, error) {

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionInitializedIterator{contract: _ExpressLaneAuction.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionInitialized) (event.Subscription, error) {

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionInitialized)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseInitialized(log types.Log) (*ExpressLaneAuctionInitialized, error) {
	event := new(ExpressLaneAuctionInitialized)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionRoleAdminChangedIterator struct {
	Event *ExpressLaneAuctionRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionRoleAdminChanged)
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
		it.Event = new(ExpressLaneAuctionRoleAdminChanged)
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
func (it *ExpressLaneAuctionRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionRoleAdminChanged represents a RoleAdminChanged event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*ExpressLaneAuctionRoleAdminChangedIterator, error) {

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

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionRoleAdminChangedIterator{contract: _ExpressLaneAuction.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionRoleAdminChanged)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseRoleAdminChanged(log types.Log) (*ExpressLaneAuctionRoleAdminChanged, error) {
	event := new(ExpressLaneAuctionRoleAdminChanged)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionRoleGrantedIterator struct {
	Event *ExpressLaneAuctionRoleGranted // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionRoleGranted)
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
		it.Event = new(ExpressLaneAuctionRoleGranted)
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
func (it *ExpressLaneAuctionRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionRoleGranted represents a RoleGranted event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*ExpressLaneAuctionRoleGrantedIterator, error) {

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

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionRoleGrantedIterator{contract: _ExpressLaneAuction.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionRoleGranted)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseRoleGranted(log types.Log) (*ExpressLaneAuctionRoleGranted, error) {
	event := new(ExpressLaneAuctionRoleGranted)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionRoleRevokedIterator struct {
	Event *ExpressLaneAuctionRoleRevoked // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionRoleRevoked)
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
		it.Event = new(ExpressLaneAuctionRoleRevoked)
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
func (it *ExpressLaneAuctionRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionRoleRevoked represents a RoleRevoked event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*ExpressLaneAuctionRoleRevokedIterator, error) {

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

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionRoleRevokedIterator{contract: _ExpressLaneAuction.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionRoleRevoked)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseRoleRevoked(log types.Log) (*ExpressLaneAuctionRoleRevoked, error) {
	event := new(ExpressLaneAuctionRoleRevoked)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionSetBeneficiaryIterator is returned from FilterSetBeneficiary and is used to iterate over the raw logs and unpacked data for SetBeneficiary events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetBeneficiaryIterator struct {
	Event *ExpressLaneAuctionSetBeneficiary // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionSetBeneficiaryIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionSetBeneficiary)
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
		it.Event = new(ExpressLaneAuctionSetBeneficiary)
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
func (it *ExpressLaneAuctionSetBeneficiaryIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionSetBeneficiaryIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionSetBeneficiary represents a SetBeneficiary event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetBeneficiary struct {
	OldBeneficiary common.Address
	NewBeneficiary common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterSetBeneficiary is a free log retrieval operation binding the contract event 0x8a0149b2f3ddf2c9ee85738165131d82babbb938f749321d59f75750afa7f4e6.
//
// Solidity: event SetBeneficiary(address oldBeneficiary, address newBeneficiary)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterSetBeneficiary(opts *bind.FilterOpts) (*ExpressLaneAuctionSetBeneficiaryIterator, error) {

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "SetBeneficiary")
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionSetBeneficiaryIterator{contract: _ExpressLaneAuction.contract, event: "SetBeneficiary", logs: logs, sub: sub}, nil
}

// WatchSetBeneficiary is a free log subscription operation binding the contract event 0x8a0149b2f3ddf2c9ee85738165131d82babbb938f749321d59f75750afa7f4e6.
//
// Solidity: event SetBeneficiary(address oldBeneficiary, address newBeneficiary)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchSetBeneficiary(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionSetBeneficiary) (event.Subscription, error) {

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "SetBeneficiary")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionSetBeneficiary)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetBeneficiary", log); err != nil {
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

// ParseSetBeneficiary is a log parse operation binding the contract event 0x8a0149b2f3ddf2c9ee85738165131d82babbb938f749321d59f75750afa7f4e6.
//
// Solidity: event SetBeneficiary(address oldBeneficiary, address newBeneficiary)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseSetBeneficiary(log types.Log) (*ExpressLaneAuctionSetBeneficiary, error) {
	event := new(ExpressLaneAuctionSetBeneficiary)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetBeneficiary", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionSetExpressLaneControllerIterator is returned from FilterSetExpressLaneController and is used to iterate over the raw logs and unpacked data for SetExpressLaneController events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetExpressLaneControllerIterator struct {
	Event *ExpressLaneAuctionSetExpressLaneController // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionSetExpressLaneControllerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionSetExpressLaneController)
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
		it.Event = new(ExpressLaneAuctionSetExpressLaneController)
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
func (it *ExpressLaneAuctionSetExpressLaneControllerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionSetExpressLaneControllerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionSetExpressLaneController represents a SetExpressLaneController event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetExpressLaneController struct {
	Round                         uint64
	PreviousExpressLaneController common.Address
	NewExpressLaneController      common.Address
	Transferor                    common.Address
	StartTimestamp                uint64
	EndTimestamp                  uint64
	Raw                           types.Log // Blockchain specific contextual infos
}

// FilterSetExpressLaneController is a free log retrieval operation binding the contract event 0xb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b.
//
// Solidity: event SetExpressLaneController(uint64 round, address indexed previousExpressLaneController, address indexed newExpressLaneController, address indexed transferor, uint64 startTimestamp, uint64 endTimestamp)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterSetExpressLaneController(opts *bind.FilterOpts, previousExpressLaneController []common.Address, newExpressLaneController []common.Address, transferor []common.Address) (*ExpressLaneAuctionSetExpressLaneControllerIterator, error) {

	var previousExpressLaneControllerRule []interface{}
	for _, previousExpressLaneControllerItem := range previousExpressLaneController {
		previousExpressLaneControllerRule = append(previousExpressLaneControllerRule, previousExpressLaneControllerItem)
	}
	var newExpressLaneControllerRule []interface{}
	for _, newExpressLaneControllerItem := range newExpressLaneController {
		newExpressLaneControllerRule = append(newExpressLaneControllerRule, newExpressLaneControllerItem)
	}
	var transferorRule []interface{}
	for _, transferorItem := range transferor {
		transferorRule = append(transferorRule, transferorItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "SetExpressLaneController", previousExpressLaneControllerRule, newExpressLaneControllerRule, transferorRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionSetExpressLaneControllerIterator{contract: _ExpressLaneAuction.contract, event: "SetExpressLaneController", logs: logs, sub: sub}, nil
}

// WatchSetExpressLaneController is a free log subscription operation binding the contract event 0xb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b.
//
// Solidity: event SetExpressLaneController(uint64 round, address indexed previousExpressLaneController, address indexed newExpressLaneController, address indexed transferor, uint64 startTimestamp, uint64 endTimestamp)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchSetExpressLaneController(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionSetExpressLaneController, previousExpressLaneController []common.Address, newExpressLaneController []common.Address, transferor []common.Address) (event.Subscription, error) {

	var previousExpressLaneControllerRule []interface{}
	for _, previousExpressLaneControllerItem := range previousExpressLaneController {
		previousExpressLaneControllerRule = append(previousExpressLaneControllerRule, previousExpressLaneControllerItem)
	}
	var newExpressLaneControllerRule []interface{}
	for _, newExpressLaneControllerItem := range newExpressLaneController {
		newExpressLaneControllerRule = append(newExpressLaneControllerRule, newExpressLaneControllerItem)
	}
	var transferorRule []interface{}
	for _, transferorItem := range transferor {
		transferorRule = append(transferorRule, transferorItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "SetExpressLaneController", previousExpressLaneControllerRule, newExpressLaneControllerRule, transferorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionSetExpressLaneController)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetExpressLaneController", log); err != nil {
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

// ParseSetExpressLaneController is a log parse operation binding the contract event 0xb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b.
//
// Solidity: event SetExpressLaneController(uint64 round, address indexed previousExpressLaneController, address indexed newExpressLaneController, address indexed transferor, uint64 startTimestamp, uint64 endTimestamp)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseSetExpressLaneController(log types.Log) (*ExpressLaneAuctionSetExpressLaneController, error) {
	event := new(ExpressLaneAuctionSetExpressLaneController)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetExpressLaneController", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionSetMinReservePriceIterator is returned from FilterSetMinReservePrice and is used to iterate over the raw logs and unpacked data for SetMinReservePrice events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetMinReservePriceIterator struct {
	Event *ExpressLaneAuctionSetMinReservePrice // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionSetMinReservePriceIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionSetMinReservePrice)
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
		it.Event = new(ExpressLaneAuctionSetMinReservePrice)
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
func (it *ExpressLaneAuctionSetMinReservePriceIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionSetMinReservePriceIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionSetMinReservePrice represents a SetMinReservePrice event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetMinReservePrice struct {
	OldPrice *big.Int
	NewPrice *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterSetMinReservePrice is a free log retrieval operation binding the contract event 0x5848068f11aa3ba9fe3fc33c5f9f2a3cd1aed67986b85b5e0cedc67dbe96f0f0.
//
// Solidity: event SetMinReservePrice(uint256 oldPrice, uint256 newPrice)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterSetMinReservePrice(opts *bind.FilterOpts) (*ExpressLaneAuctionSetMinReservePriceIterator, error) {

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "SetMinReservePrice")
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionSetMinReservePriceIterator{contract: _ExpressLaneAuction.contract, event: "SetMinReservePrice", logs: logs, sub: sub}, nil
}

// WatchSetMinReservePrice is a free log subscription operation binding the contract event 0x5848068f11aa3ba9fe3fc33c5f9f2a3cd1aed67986b85b5e0cedc67dbe96f0f0.
//
// Solidity: event SetMinReservePrice(uint256 oldPrice, uint256 newPrice)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchSetMinReservePrice(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionSetMinReservePrice) (event.Subscription, error) {

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "SetMinReservePrice")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionSetMinReservePrice)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetMinReservePrice", log); err != nil {
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

// ParseSetMinReservePrice is a log parse operation binding the contract event 0x5848068f11aa3ba9fe3fc33c5f9f2a3cd1aed67986b85b5e0cedc67dbe96f0f0.
//
// Solidity: event SetMinReservePrice(uint256 oldPrice, uint256 newPrice)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseSetMinReservePrice(log types.Log) (*ExpressLaneAuctionSetMinReservePrice, error) {
	event := new(ExpressLaneAuctionSetMinReservePrice)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetMinReservePrice", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionSetReservePriceIterator is returned from FilterSetReservePrice and is used to iterate over the raw logs and unpacked data for SetReservePrice events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetReservePriceIterator struct {
	Event *ExpressLaneAuctionSetReservePrice // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionSetReservePriceIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionSetReservePrice)
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
		it.Event = new(ExpressLaneAuctionSetReservePrice)
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
func (it *ExpressLaneAuctionSetReservePriceIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionSetReservePriceIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionSetReservePrice represents a SetReservePrice event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetReservePrice struct {
	OldReservePrice *big.Int
	NewReservePrice *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterSetReservePrice is a free log retrieval operation binding the contract event 0x9725e37e079c5bda6009a8f54d86265849f30acf61c630f9e1ac91e67de98794.
//
// Solidity: event SetReservePrice(uint256 oldReservePrice, uint256 newReservePrice)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterSetReservePrice(opts *bind.FilterOpts) (*ExpressLaneAuctionSetReservePriceIterator, error) {

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "SetReservePrice")
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionSetReservePriceIterator{contract: _ExpressLaneAuction.contract, event: "SetReservePrice", logs: logs, sub: sub}, nil
}

// WatchSetReservePrice is a free log subscription operation binding the contract event 0x9725e37e079c5bda6009a8f54d86265849f30acf61c630f9e1ac91e67de98794.
//
// Solidity: event SetReservePrice(uint256 oldReservePrice, uint256 newReservePrice)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchSetReservePrice(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionSetReservePrice) (event.Subscription, error) {

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "SetReservePrice")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionSetReservePrice)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetReservePrice", log); err != nil {
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

// ParseSetReservePrice is a log parse operation binding the contract event 0x9725e37e079c5bda6009a8f54d86265849f30acf61c630f9e1ac91e67de98794.
//
// Solidity: event SetReservePrice(uint256 oldReservePrice, uint256 newReservePrice)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseSetReservePrice(log types.Log) (*ExpressLaneAuctionSetReservePrice, error) {
	event := new(ExpressLaneAuctionSetReservePrice)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetReservePrice", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionSetRoundTimingInfoIterator is returned from FilterSetRoundTimingInfo and is used to iterate over the raw logs and unpacked data for SetRoundTimingInfo events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetRoundTimingInfoIterator struct {
	Event *ExpressLaneAuctionSetRoundTimingInfo // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionSetRoundTimingInfoIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionSetRoundTimingInfo)
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
		it.Event = new(ExpressLaneAuctionSetRoundTimingInfo)
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
func (it *ExpressLaneAuctionSetRoundTimingInfoIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionSetRoundTimingInfoIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionSetRoundTimingInfo represents a SetRoundTimingInfo event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetRoundTimingInfo struct {
	CurrentRound             uint64
	OffsetTimestamp          int64
	RoundDurationSeconds     uint64
	AuctionClosingSeconds    uint64
	ReserveSubmissionSeconds uint64
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterSetRoundTimingInfo is a free log retrieval operation binding the contract event 0x982cfb73783b8c64455c76cdeb1351467c4f1e6b3615fec07df232c1b46ffd47.
//
// Solidity: event SetRoundTimingInfo(uint64 currentRound, int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterSetRoundTimingInfo(opts *bind.FilterOpts) (*ExpressLaneAuctionSetRoundTimingInfoIterator, error) {

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "SetRoundTimingInfo")
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionSetRoundTimingInfoIterator{contract: _ExpressLaneAuction.contract, event: "SetRoundTimingInfo", logs: logs, sub: sub}, nil
}

// WatchSetRoundTimingInfo is a free log subscription operation binding the contract event 0x982cfb73783b8c64455c76cdeb1351467c4f1e6b3615fec07df232c1b46ffd47.
//
// Solidity: event SetRoundTimingInfo(uint64 currentRound, int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchSetRoundTimingInfo(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionSetRoundTimingInfo) (event.Subscription, error) {

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "SetRoundTimingInfo")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionSetRoundTimingInfo)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetRoundTimingInfo", log); err != nil {
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

// ParseSetRoundTimingInfo is a log parse operation binding the contract event 0x982cfb73783b8c64455c76cdeb1351467c4f1e6b3615fec07df232c1b46ffd47.
//
// Solidity: event SetRoundTimingInfo(uint64 currentRound, int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseSetRoundTimingInfo(log types.Log) (*ExpressLaneAuctionSetRoundTimingInfo, error) {
	event := new(ExpressLaneAuctionSetRoundTimingInfo)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetRoundTimingInfo", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionSetTransferorIterator is returned from FilterSetTransferor and is used to iterate over the raw logs and unpacked data for SetTransferor events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetTransferorIterator struct {
	Event *ExpressLaneAuctionSetTransferor // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionSetTransferorIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionSetTransferor)
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
		it.Event = new(ExpressLaneAuctionSetTransferor)
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
func (it *ExpressLaneAuctionSetTransferorIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionSetTransferorIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionSetTransferor represents a SetTransferor event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionSetTransferor struct {
	ExpressLaneController common.Address
	Transferor            common.Address
	FixedUntilRound       uint64
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterSetTransferor is a free log retrieval operation binding the contract event 0xf6d28df235d9fa45a42d45dbb7c4f4ac76edb51e528f09f25a0650d32b8b33c0.
//
// Solidity: event SetTransferor(address indexed expressLaneController, address indexed transferor, uint64 fixedUntilRound)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterSetTransferor(opts *bind.FilterOpts, expressLaneController []common.Address, transferor []common.Address) (*ExpressLaneAuctionSetTransferorIterator, error) {

	var expressLaneControllerRule []interface{}
	for _, expressLaneControllerItem := range expressLaneController {
		expressLaneControllerRule = append(expressLaneControllerRule, expressLaneControllerItem)
	}
	var transferorRule []interface{}
	for _, transferorItem := range transferor {
		transferorRule = append(transferorRule, transferorItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "SetTransferor", expressLaneControllerRule, transferorRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionSetTransferorIterator{contract: _ExpressLaneAuction.contract, event: "SetTransferor", logs: logs, sub: sub}, nil
}

// WatchSetTransferor is a free log subscription operation binding the contract event 0xf6d28df235d9fa45a42d45dbb7c4f4ac76edb51e528f09f25a0650d32b8b33c0.
//
// Solidity: event SetTransferor(address indexed expressLaneController, address indexed transferor, uint64 fixedUntilRound)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchSetTransferor(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionSetTransferor, expressLaneController []common.Address, transferor []common.Address) (event.Subscription, error) {

	var expressLaneControllerRule []interface{}
	for _, expressLaneControllerItem := range expressLaneController {
		expressLaneControllerRule = append(expressLaneControllerRule, expressLaneControllerItem)
	}
	var transferorRule []interface{}
	for _, transferorItem := range transferor {
		transferorRule = append(transferorRule, transferorItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "SetTransferor", expressLaneControllerRule, transferorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionSetTransferor)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetTransferor", log); err != nil {
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

// ParseSetTransferor is a log parse operation binding the contract event 0xf6d28df235d9fa45a42d45dbb7c4f4ac76edb51e528f09f25a0650d32b8b33c0.
//
// Solidity: event SetTransferor(address indexed expressLaneController, address indexed transferor, uint64 fixedUntilRound)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseSetTransferor(log types.Log) (*ExpressLaneAuctionSetTransferor, error) {
	event := new(ExpressLaneAuctionSetTransferor)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "SetTransferor", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionWithdrawalFinalizedIterator is returned from FilterWithdrawalFinalized and is used to iterate over the raw logs and unpacked data for WithdrawalFinalized events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionWithdrawalFinalizedIterator struct {
	Event *ExpressLaneAuctionWithdrawalFinalized // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionWithdrawalFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionWithdrawalFinalized)
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
		it.Event = new(ExpressLaneAuctionWithdrawalFinalized)
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
func (it *ExpressLaneAuctionWithdrawalFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionWithdrawalFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionWithdrawalFinalized represents a WithdrawalFinalized event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionWithdrawalFinalized struct {
	Account          common.Address
	WithdrawalAmount *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalFinalized is a free log retrieval operation binding the contract event 0x9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda8.
//
// Solidity: event WithdrawalFinalized(address indexed account, uint256 withdrawalAmount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterWithdrawalFinalized(opts *bind.FilterOpts, account []common.Address) (*ExpressLaneAuctionWithdrawalFinalizedIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "WithdrawalFinalized", accountRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionWithdrawalFinalizedIterator{contract: _ExpressLaneAuction.contract, event: "WithdrawalFinalized", logs: logs, sub: sub}, nil
}

// WatchWithdrawalFinalized is a free log subscription operation binding the contract event 0x9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda8.
//
// Solidity: event WithdrawalFinalized(address indexed account, uint256 withdrawalAmount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchWithdrawalFinalized(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionWithdrawalFinalized, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "WithdrawalFinalized", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionWithdrawalFinalized)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
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

// ParseWithdrawalFinalized is a log parse operation binding the contract event 0x9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda8.
//
// Solidity: event WithdrawalFinalized(address indexed account, uint256 withdrawalAmount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseWithdrawalFinalized(log types.Log) (*ExpressLaneAuctionWithdrawalFinalized, error) {
	event := new(ExpressLaneAuctionWithdrawalFinalized)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionWithdrawalInitiatedIterator is returned from FilterWithdrawalInitiated and is used to iterate over the raw logs and unpacked data for WithdrawalInitiated events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionWithdrawalInitiatedIterator struct {
	Event *ExpressLaneAuctionWithdrawalInitiated // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionWithdrawalInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionWithdrawalInitiated)
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
		it.Event = new(ExpressLaneAuctionWithdrawalInitiated)
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
func (it *ExpressLaneAuctionWithdrawalInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionWithdrawalInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionWithdrawalInitiated represents a WithdrawalInitiated event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionWithdrawalInitiated struct {
	Account           common.Address
	WithdrawalAmount  *big.Int
	RoundWithdrawable *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalInitiated is a free log retrieval operation binding the contract event 0x31f69201fab7912e3ec9850e3ab705964bf46d9d4276bdcbb6d05e965e5f5401.
//
// Solidity: event WithdrawalInitiated(address indexed account, uint256 withdrawalAmount, uint256 roundWithdrawable)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterWithdrawalInitiated(opts *bind.FilterOpts, account []common.Address) (*ExpressLaneAuctionWithdrawalInitiatedIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "WithdrawalInitiated", accountRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionWithdrawalInitiatedIterator{contract: _ExpressLaneAuction.contract, event: "WithdrawalInitiated", logs: logs, sub: sub}, nil
}

// WatchWithdrawalInitiated is a free log subscription operation binding the contract event 0x31f69201fab7912e3ec9850e3ab705964bf46d9d4276bdcbb6d05e965e5f5401.
//
// Solidity: event WithdrawalInitiated(address indexed account, uint256 withdrawalAmount, uint256 roundWithdrawable)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchWithdrawalInitiated(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionWithdrawalInitiated, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "WithdrawalInitiated", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionWithdrawalInitiated)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "WithdrawalInitiated", log); err != nil {
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

// ParseWithdrawalInitiated is a log parse operation binding the contract event 0x31f69201fab7912e3ec9850e3ab705964bf46d9d4276bdcbb6d05e965e5f5401.
//
// Solidity: event WithdrawalInitiated(address indexed account, uint256 withdrawalAmount, uint256 roundWithdrawable)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseWithdrawalInitiated(log types.Log) (*ExpressLaneAuctionWithdrawalInitiated, error) {
	event := new(ExpressLaneAuctionWithdrawalInitiated)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "WithdrawalInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionMetaData contains all meta data concerning the IExpressLaneAuction contract.
var IExpressLaneAuctionMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"isMultiBidAuction\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"firstPriceBidder\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"firstPriceExpressLaneController\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"firstPriceAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"roundStartTimestamp\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"roundEndTimestamp\",\"type\":\"uint64\"}],\"name\":\"AuctionResolved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldBeneficiary\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newBeneficiary\",\"type\":\"address\"}],\"name\":\"SetBeneficiary\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousExpressLaneController\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newExpressLaneController\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"transferor\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"startTimestamp\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"endTimestamp\",\"type\":\"uint64\"}],\"name\":\"SetExpressLaneController\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldPrice\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newPrice\",\"type\":\"uint256\"}],\"name\":\"SetMinReservePrice\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldReservePrice\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newReservePrice\",\"type\":\"uint256\"}],\"name\":\"SetReservePrice\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"currentRound\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"int64\",\"name\":\"offsetTimestamp\",\"type\":\"int64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"roundDurationSeconds\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"auctionClosingSeconds\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"reserveSubmissionSeconds\",\"type\":\"uint64\"}],\"name\":\"SetRoundTimingInfo\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"transferor\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"fixedUntilRound\",\"type\":\"uint64\"}],\"name\":\"SetTransferor\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"withdrawalAmount\",\"type\":\"uint256\"}],\"name\":\"WithdrawalFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"withdrawalAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"roundWithdrawable\",\"type\":\"uint256\"}],\"name\":\"WithdrawalInitiated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"AUCTIONEER_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"AUCTIONEER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"BENEFICIARY_SETTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_RESERVE_SETTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RESERVE_SETTER_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RESERVE_SETTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ROUND_TIMING_SETTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"name\":\"balanceOfAtRound\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"beneficiary\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"beneficiaryBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"biddingToken\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentRound\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"deposit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"domainSeparator\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"finalizeWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"flushBeneficiaryBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"getBidHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getRoleMember\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleMemberCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"_auctioneer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_biddingToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_beneficiary\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"int64\",\"name\":\"offsetTimestamp\",\"type\":\"int64\"},{\"internalType\":\"uint64\",\"name\":\"roundDurationSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"auctionClosingSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"reserveSubmissionSeconds\",\"type\":\"uint64\"}],\"internalType\":\"structRoundTimingInfo\",\"name\":\"_roundTimingInfo\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"_minReservePrice\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_auctioneerAdmin\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_minReservePriceSetter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_reservePriceSetter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_reservePriceSetterAdmin\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_beneficiarySetter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_roundTimingSetter\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_masterAdmin\",\"type\":\"address\"}],\"internalType\":\"structInitArgs\",\"name\":\"args\",\"type\":\"tuple\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initiateWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isAuctionRoundClosed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isReserveBlackout\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minReservePrice\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"reservePrice\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structBid\",\"name\":\"firstPriceBid\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structBid\",\"name\":\"secondPriceBid\",\"type\":\"tuple\"}],\"name\":\"resolveMultiBidAuction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structBid\",\"name\":\"firstPriceBid\",\"type\":\"tuple\"}],\"name\":\"resolveSingleBidAuction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"resolvedRounds\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"internalType\":\"structELCRound\",\"name\":\"\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"internalType\":\"structELCRound\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"name\":\"roundTimestamps\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"start\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"end\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"roundTimingInfo\",\"outputs\":[{\"internalType\":\"int64\",\"name\":\"offsetTimestamp\",\"type\":\"int64\"},{\"internalType\":\"uint64\",\"name\":\"roundDurationSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"auctionClosingSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"reserveSubmissionSeconds\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newBeneficiary\",\"type\":\"address\"}],\"name\":\"setBeneficiary\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newMinReservePrice\",\"type\":\"uint256\"}],\"name\":\"setMinReservePrice\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newReservePrice\",\"type\":\"uint256\"}],\"name\":\"setReservePrice\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"int64\",\"name\":\"offsetTimestamp\",\"type\":\"int64\"},{\"internalType\":\"uint64\",\"name\":\"roundDurationSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"auctionClosingSeconds\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"reserveSubmissionSeconds\",\"type\":\"uint64\"}],\"internalType\":\"structRoundTimingInfo\",\"name\":\"newRoundTimingInfo\",\"type\":\"tuple\"}],\"name\":\"setRoundTimingInfo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"fixedUntilRound\",\"type\":\"uint64\"}],\"internalType\":\"structTransferor\",\"name\":\"transferor\",\"type\":\"tuple\"}],\"name\":\"setTransferor\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"newExpressLaneController\",\"type\":\"address\"}],\"name\":\"transferExpressLaneController\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"expressLaneController\",\"type\":\"address\"}],\"name\":\"transferorOf\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"fixedUntil\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"withdrawableBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"round\",\"type\":\"uint64\"}],\"name\":\"withdrawableBalanceAtRound\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IExpressLaneAuctionABI is the input ABI used to generate the binding from.
// Deprecated: Use IExpressLaneAuctionMetaData.ABI instead.
var IExpressLaneAuctionABI = IExpressLaneAuctionMetaData.ABI

// IExpressLaneAuction is an auto generated Go binding around an Ethereum contract.
type IExpressLaneAuction struct {
	IExpressLaneAuctionCaller     // Read-only binding to the contract
	IExpressLaneAuctionTransactor // Write-only binding to the contract
	IExpressLaneAuctionFilterer   // Log filterer for contract events
}

// IExpressLaneAuctionCaller is an auto generated read-only Go binding around an Ethereum contract.
type IExpressLaneAuctionCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IExpressLaneAuctionTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IExpressLaneAuctionTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IExpressLaneAuctionFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IExpressLaneAuctionFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IExpressLaneAuctionSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IExpressLaneAuctionSession struct {
	Contract     *IExpressLaneAuction // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// IExpressLaneAuctionCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IExpressLaneAuctionCallerSession struct {
	Contract *IExpressLaneAuctionCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// IExpressLaneAuctionTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IExpressLaneAuctionTransactorSession struct {
	Contract     *IExpressLaneAuctionTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// IExpressLaneAuctionRaw is an auto generated low-level Go binding around an Ethereum contract.
type IExpressLaneAuctionRaw struct {
	Contract *IExpressLaneAuction // Generic contract binding to access the raw methods on
}

// IExpressLaneAuctionCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IExpressLaneAuctionCallerRaw struct {
	Contract *IExpressLaneAuctionCaller // Generic read-only contract binding to access the raw methods on
}

// IExpressLaneAuctionTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IExpressLaneAuctionTransactorRaw struct {
	Contract *IExpressLaneAuctionTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIExpressLaneAuction creates a new instance of IExpressLaneAuction, bound to a specific deployed contract.
func NewIExpressLaneAuction(address common.Address, backend bind.ContractBackend) (*IExpressLaneAuction, error) {
	contract, err := bindIExpressLaneAuction(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuction{IExpressLaneAuctionCaller: IExpressLaneAuctionCaller{contract: contract}, IExpressLaneAuctionTransactor: IExpressLaneAuctionTransactor{contract: contract}, IExpressLaneAuctionFilterer: IExpressLaneAuctionFilterer{contract: contract}}, nil
}

// NewIExpressLaneAuctionCaller creates a new read-only instance of IExpressLaneAuction, bound to a specific deployed contract.
func NewIExpressLaneAuctionCaller(address common.Address, caller bind.ContractCaller) (*IExpressLaneAuctionCaller, error) {
	contract, err := bindIExpressLaneAuction(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionCaller{contract: contract}, nil
}

// NewIExpressLaneAuctionTransactor creates a new write-only instance of IExpressLaneAuction, bound to a specific deployed contract.
func NewIExpressLaneAuctionTransactor(address common.Address, transactor bind.ContractTransactor) (*IExpressLaneAuctionTransactor, error) {
	contract, err := bindIExpressLaneAuction(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionTransactor{contract: contract}, nil
}

// NewIExpressLaneAuctionFilterer creates a new log filterer instance of IExpressLaneAuction, bound to a specific deployed contract.
func NewIExpressLaneAuctionFilterer(address common.Address, filterer bind.ContractFilterer) (*IExpressLaneAuctionFilterer, error) {
	contract, err := bindIExpressLaneAuction(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionFilterer{contract: contract}, nil
}

// bindIExpressLaneAuction binds a generic wrapper to an already deployed contract.
func bindIExpressLaneAuction(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IExpressLaneAuctionMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IExpressLaneAuction *IExpressLaneAuctionRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IExpressLaneAuction.Contract.IExpressLaneAuctionCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IExpressLaneAuction *IExpressLaneAuctionRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.IExpressLaneAuctionTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IExpressLaneAuction *IExpressLaneAuctionRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.IExpressLaneAuctionTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IExpressLaneAuction *IExpressLaneAuctionCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IExpressLaneAuction.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.contract.Transact(opts, method, params...)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _IExpressLaneAuction.Contract.BalanceOf(&_IExpressLaneAuction.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _IExpressLaneAuction.Contract.BalanceOf(&_IExpressLaneAuction.CallOpts, account)
}

// BalanceOfAtRound is a free data retrieval call binding the contract method 0x5633c337.
//
// Solidity: function balanceOfAtRound(address account, uint64 round) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) BalanceOfAtRound(opts *bind.CallOpts, account common.Address, round uint64) (*big.Int, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "balanceOfAtRound", account, round)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOfAtRound is a free data retrieval call binding the contract method 0x5633c337.
//
// Solidity: function balanceOfAtRound(address account, uint64 round) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) BalanceOfAtRound(account common.Address, round uint64) (*big.Int, error) {
	return _IExpressLaneAuction.Contract.BalanceOfAtRound(&_IExpressLaneAuction.CallOpts, account, round)
}

// BalanceOfAtRound is a free data retrieval call binding the contract method 0x5633c337.
//
// Solidity: function balanceOfAtRound(address account, uint64 round) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) BalanceOfAtRound(account common.Address, round uint64) (*big.Int, error) {
	return _IExpressLaneAuction.Contract.BalanceOfAtRound(&_IExpressLaneAuction.CallOpts, account, round)
}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint64)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) CurrentRound(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "currentRound")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint64)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) CurrentRound() (uint64, error) {
	return _IExpressLaneAuction.Contract.CurrentRound(&_IExpressLaneAuction.CallOpts)
}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint64)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) CurrentRound() (uint64, error) {
	return _IExpressLaneAuction.Contract.CurrentRound(&_IExpressLaneAuction.CallOpts)
}

// DomainSeparator is a free data retrieval call binding the contract method 0xf698da25.
//
// Solidity: function domainSeparator() view returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) DomainSeparator(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "domainSeparator")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DomainSeparator is a free data retrieval call binding the contract method 0xf698da25.
//
// Solidity: function domainSeparator() view returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) DomainSeparator() ([32]byte, error) {
	return _IExpressLaneAuction.Contract.DomainSeparator(&_IExpressLaneAuction.CallOpts)
}

// DomainSeparator is a free data retrieval call binding the contract method 0xf698da25.
//
// Solidity: function domainSeparator() view returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) DomainSeparator() ([32]byte, error) {
	return _IExpressLaneAuction.Contract.DomainSeparator(&_IExpressLaneAuction.CallOpts)
}

// GetBidHash is a free data retrieval call binding the contract method 0x04c584ad.
//
// Solidity: function getBidHash(uint64 round, address expressLaneController, uint256 amount) view returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) GetBidHash(opts *bind.CallOpts, round uint64, expressLaneController common.Address, amount *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "getBidHash", round, expressLaneController, amount)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBidHash is a free data retrieval call binding the contract method 0x04c584ad.
//
// Solidity: function getBidHash(uint64 round, address expressLaneController, uint256 amount) view returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) GetBidHash(round uint64, expressLaneController common.Address, amount *big.Int) ([32]byte, error) {
	return _IExpressLaneAuction.Contract.GetBidHash(&_IExpressLaneAuction.CallOpts, round, expressLaneController, amount)
}

// GetBidHash is a free data retrieval call binding the contract method 0x04c584ad.
//
// Solidity: function getBidHash(uint64 round, address expressLaneController, uint256 amount) view returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) GetBidHash(round uint64, expressLaneController common.Address, amount *big.Int) ([32]byte, error) {
	return _IExpressLaneAuction.Contract.GetBidHash(&_IExpressLaneAuction.CallOpts, round, expressLaneController, amount)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _IExpressLaneAuction.Contract.GetRoleAdmin(&_IExpressLaneAuction.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _IExpressLaneAuction.Contract.GetRoleAdmin(&_IExpressLaneAuction.CallOpts, role)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) GetRoleMember(opts *bind.CallOpts, role [32]byte, index *big.Int) (common.Address, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "getRoleMember", role, index)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _IExpressLaneAuction.Contract.GetRoleMember(&_IExpressLaneAuction.CallOpts, role, index)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _IExpressLaneAuction.Contract.GetRoleMember(&_IExpressLaneAuction.CallOpts, role, index)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) GetRoleMemberCount(opts *bind.CallOpts, role [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "getRoleMemberCount", role)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _IExpressLaneAuction.Contract.GetRoleMemberCount(&_IExpressLaneAuction.CallOpts, role)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _IExpressLaneAuction.Contract.GetRoleMemberCount(&_IExpressLaneAuction.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _IExpressLaneAuction.Contract.HasRole(&_IExpressLaneAuction.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _IExpressLaneAuction.Contract.HasRole(&_IExpressLaneAuction.CallOpts, role, account)
}

// IsAuctionRoundClosed is a free data retrieval call binding the contract method 0x2d668ce7.
//
// Solidity: function isAuctionRoundClosed() view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) IsAuctionRoundClosed(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "isAuctionRoundClosed")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAuctionRoundClosed is a free data retrieval call binding the contract method 0x2d668ce7.
//
// Solidity: function isAuctionRoundClosed() view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) IsAuctionRoundClosed() (bool, error) {
	return _IExpressLaneAuction.Contract.IsAuctionRoundClosed(&_IExpressLaneAuction.CallOpts)
}

// IsAuctionRoundClosed is a free data retrieval call binding the contract method 0x2d668ce7.
//
// Solidity: function isAuctionRoundClosed() view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) IsAuctionRoundClosed() (bool, error) {
	return _IExpressLaneAuction.Contract.IsAuctionRoundClosed(&_IExpressLaneAuction.CallOpts)
}

// IsReserveBlackout is a free data retrieval call binding the contract method 0xe460d2c5.
//
// Solidity: function isReserveBlackout() view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) IsReserveBlackout(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "isReserveBlackout")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsReserveBlackout is a free data retrieval call binding the contract method 0xe460d2c5.
//
// Solidity: function isReserveBlackout() view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) IsReserveBlackout() (bool, error) {
	return _IExpressLaneAuction.Contract.IsReserveBlackout(&_IExpressLaneAuction.CallOpts)
}

// IsReserveBlackout is a free data retrieval call binding the contract method 0xe460d2c5.
//
// Solidity: function isReserveBlackout() view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) IsReserveBlackout() (bool, error) {
	return _IExpressLaneAuction.Contract.IsReserveBlackout(&_IExpressLaneAuction.CallOpts)
}

// ResolvedRounds is a free data retrieval call binding the contract method 0x0d253fbe.
//
// Solidity: function resolvedRounds() view returns((address,uint64), (address,uint64))
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) ResolvedRounds(opts *bind.CallOpts) (ELCRound, ELCRound, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "resolvedRounds")

	if err != nil {
		return *new(ELCRound), *new(ELCRound), err
	}

	out0 := *abi.ConvertType(out[0], new(ELCRound)).(*ELCRound)
	out1 := *abi.ConvertType(out[1], new(ELCRound)).(*ELCRound)

	return out0, out1, err

}

// ResolvedRounds is a free data retrieval call binding the contract method 0x0d253fbe.
//
// Solidity: function resolvedRounds() view returns((address,uint64), (address,uint64))
func (_IExpressLaneAuction *IExpressLaneAuctionSession) ResolvedRounds() (ELCRound, ELCRound, error) {
	return _IExpressLaneAuction.Contract.ResolvedRounds(&_IExpressLaneAuction.CallOpts)
}

// ResolvedRounds is a free data retrieval call binding the contract method 0x0d253fbe.
//
// Solidity: function resolvedRounds() view returns((address,uint64), (address,uint64))
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) ResolvedRounds() (ELCRound, ELCRound, error) {
	return _IExpressLaneAuction.Contract.ResolvedRounds(&_IExpressLaneAuction.CallOpts)
}

// RoundTimestamps is a free data retrieval call binding the contract method 0x7b617f94.
//
// Solidity: function roundTimestamps(uint64 round) view returns(uint64 start, uint64 end)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) RoundTimestamps(opts *bind.CallOpts, round uint64) (struct {
	Start uint64
	End   uint64
}, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "roundTimestamps", round)

	outstruct := new(struct {
		Start uint64
		End   uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Start = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.End = *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return *outstruct, err

}

// RoundTimestamps is a free data retrieval call binding the contract method 0x7b617f94.
//
// Solidity: function roundTimestamps(uint64 round) view returns(uint64 start, uint64 end)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) RoundTimestamps(round uint64) (struct {
	Start uint64
	End   uint64
}, error) {
	return _IExpressLaneAuction.Contract.RoundTimestamps(&_IExpressLaneAuction.CallOpts, round)
}

// RoundTimestamps is a free data retrieval call binding the contract method 0x7b617f94.
//
// Solidity: function roundTimestamps(uint64 round) view returns(uint64 start, uint64 end)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) RoundTimestamps(round uint64) (struct {
	Start uint64
	End   uint64
}, error) {
	return _IExpressLaneAuction.Contract.RoundTimestamps(&_IExpressLaneAuction.CallOpts, round)
}

// RoundTimingInfo is a free data retrieval call binding the contract method 0x0152682d.
//
// Solidity: function roundTimingInfo() view returns(int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) RoundTimingInfo(opts *bind.CallOpts) (struct {
	OffsetTimestamp          int64
	RoundDurationSeconds     uint64
	AuctionClosingSeconds    uint64
	ReserveSubmissionSeconds uint64
}, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "roundTimingInfo")

	outstruct := new(struct {
		OffsetTimestamp          int64
		RoundDurationSeconds     uint64
		AuctionClosingSeconds    uint64
		ReserveSubmissionSeconds uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.OffsetTimestamp = *abi.ConvertType(out[0], new(int64)).(*int64)
	outstruct.RoundDurationSeconds = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.AuctionClosingSeconds = *abi.ConvertType(out[2], new(uint64)).(*uint64)
	outstruct.ReserveSubmissionSeconds = *abi.ConvertType(out[3], new(uint64)).(*uint64)

	return *outstruct, err

}

// RoundTimingInfo is a free data retrieval call binding the contract method 0x0152682d.
//
// Solidity: function roundTimingInfo() view returns(int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) RoundTimingInfo() (struct {
	OffsetTimestamp          int64
	RoundDurationSeconds     uint64
	AuctionClosingSeconds    uint64
	ReserveSubmissionSeconds uint64
}, error) {
	return _IExpressLaneAuction.Contract.RoundTimingInfo(&_IExpressLaneAuction.CallOpts)
}

// RoundTimingInfo is a free data retrieval call binding the contract method 0x0152682d.
//
// Solidity: function roundTimingInfo() view returns(int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) RoundTimingInfo() (struct {
	OffsetTimestamp          int64
	RoundDurationSeconds     uint64
	AuctionClosingSeconds    uint64
	ReserveSubmissionSeconds uint64
}, error) {
	return _IExpressLaneAuction.Contract.RoundTimingInfo(&_IExpressLaneAuction.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _IExpressLaneAuction.Contract.SupportsInterface(&_IExpressLaneAuction.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _IExpressLaneAuction.Contract.SupportsInterface(&_IExpressLaneAuction.CallOpts, interfaceId)
}

// WithdrawableBalance is a free data retrieval call binding the contract method 0x02b62938.
//
// Solidity: function withdrawableBalance(address account) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) WithdrawableBalance(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "withdrawableBalance", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// WithdrawableBalance is a free data retrieval call binding the contract method 0x02b62938.
//
// Solidity: function withdrawableBalance(address account) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) WithdrawableBalance(account common.Address) (*big.Int, error) {
	return _IExpressLaneAuction.Contract.WithdrawableBalance(&_IExpressLaneAuction.CallOpts, account)
}

// WithdrawableBalance is a free data retrieval call binding the contract method 0x02b62938.
//
// Solidity: function withdrawableBalance(address account) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) WithdrawableBalance(account common.Address) (*big.Int, error) {
	return _IExpressLaneAuction.Contract.WithdrawableBalance(&_IExpressLaneAuction.CallOpts, account)
}

// WithdrawableBalanceAtRound is a free data retrieval call binding the contract method 0x6e8cace5.
//
// Solidity: function withdrawableBalanceAtRound(address account, uint64 round) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionCaller) WithdrawableBalanceAtRound(opts *bind.CallOpts, account common.Address, round uint64) (*big.Int, error) {
	var out []interface{}
	err := _IExpressLaneAuction.contract.Call(opts, &out, "withdrawableBalanceAtRound", account, round)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// WithdrawableBalanceAtRound is a free data retrieval call binding the contract method 0x6e8cace5.
//
// Solidity: function withdrawableBalanceAtRound(address account, uint64 round) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) WithdrawableBalanceAtRound(account common.Address, round uint64) (*big.Int, error) {
	return _IExpressLaneAuction.Contract.WithdrawableBalanceAtRound(&_IExpressLaneAuction.CallOpts, account, round)
}

// WithdrawableBalanceAtRound is a free data retrieval call binding the contract method 0x6e8cace5.
//
// Solidity: function withdrawableBalanceAtRound(address account, uint64 round) view returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionCallerSession) WithdrawableBalanceAtRound(account common.Address, round uint64) (*big.Int, error) {
	return _IExpressLaneAuction.Contract.WithdrawableBalanceAtRound(&_IExpressLaneAuction.CallOpts, account, round)
}

// AUCTIONEERADMINROLE is a paid mutator transaction binding the contract method 0x14d96316.
//
// Solidity: function AUCTIONEER_ADMIN_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) AUCTIONEERADMINROLE(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "AUCTIONEER_ADMIN_ROLE")
}

// AUCTIONEERADMINROLE is a paid mutator transaction binding the contract method 0x14d96316.
//
// Solidity: function AUCTIONEER_ADMIN_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) AUCTIONEERADMINROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.AUCTIONEERADMINROLE(&_IExpressLaneAuction.TransactOpts)
}

// AUCTIONEERADMINROLE is a paid mutator transaction binding the contract method 0x14d96316.
//
// Solidity: function AUCTIONEER_ADMIN_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) AUCTIONEERADMINROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.AUCTIONEERADMINROLE(&_IExpressLaneAuction.TransactOpts)
}

// AUCTIONEERROLE is a paid mutator transaction binding the contract method 0xcfe9232b.
//
// Solidity: function AUCTIONEER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) AUCTIONEERROLE(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "AUCTIONEER_ROLE")
}

// AUCTIONEERROLE is a paid mutator transaction binding the contract method 0xcfe9232b.
//
// Solidity: function AUCTIONEER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) AUCTIONEERROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.AUCTIONEERROLE(&_IExpressLaneAuction.TransactOpts)
}

// AUCTIONEERROLE is a paid mutator transaction binding the contract method 0xcfe9232b.
//
// Solidity: function AUCTIONEER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) AUCTIONEERROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.AUCTIONEERROLE(&_IExpressLaneAuction.TransactOpts)
}

// BENEFICIARYSETTERROLE is a paid mutator transaction binding the contract method 0x336a5b5e.
//
// Solidity: function BENEFICIARY_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) BENEFICIARYSETTERROLE(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "BENEFICIARY_SETTER_ROLE")
}

// BENEFICIARYSETTERROLE is a paid mutator transaction binding the contract method 0x336a5b5e.
//
// Solidity: function BENEFICIARY_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) BENEFICIARYSETTERROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.BENEFICIARYSETTERROLE(&_IExpressLaneAuction.TransactOpts)
}

// BENEFICIARYSETTERROLE is a paid mutator transaction binding the contract method 0x336a5b5e.
//
// Solidity: function BENEFICIARY_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) BENEFICIARYSETTERROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.BENEFICIARYSETTERROLE(&_IExpressLaneAuction.TransactOpts)
}

// MINRESERVESETTERROLE is a paid mutator transaction binding the contract method 0x8948cc4e.
//
// Solidity: function MIN_RESERVE_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) MINRESERVESETTERROLE(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "MIN_RESERVE_SETTER_ROLE")
}

// MINRESERVESETTERROLE is a paid mutator transaction binding the contract method 0x8948cc4e.
//
// Solidity: function MIN_RESERVE_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) MINRESERVESETTERROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.MINRESERVESETTERROLE(&_IExpressLaneAuction.TransactOpts)
}

// MINRESERVESETTERROLE is a paid mutator transaction binding the contract method 0x8948cc4e.
//
// Solidity: function MIN_RESERVE_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) MINRESERVESETTERROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.MINRESERVESETTERROLE(&_IExpressLaneAuction.TransactOpts)
}

// RESERVESETTERADMINROLE is a paid mutator transaction binding the contract method 0xe3f7bb55.
//
// Solidity: function RESERVE_SETTER_ADMIN_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) RESERVESETTERADMINROLE(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "RESERVE_SETTER_ADMIN_ROLE")
}

// RESERVESETTERADMINROLE is a paid mutator transaction binding the contract method 0xe3f7bb55.
//
// Solidity: function RESERVE_SETTER_ADMIN_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) RESERVESETTERADMINROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.RESERVESETTERADMINROLE(&_IExpressLaneAuction.TransactOpts)
}

// RESERVESETTERADMINROLE is a paid mutator transaction binding the contract method 0xe3f7bb55.
//
// Solidity: function RESERVE_SETTER_ADMIN_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) RESERVESETTERADMINROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.RESERVESETTERADMINROLE(&_IExpressLaneAuction.TransactOpts)
}

// RESERVESETTERROLE is a paid mutator transaction binding the contract method 0xb3ee252f.
//
// Solidity: function RESERVE_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) RESERVESETTERROLE(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "RESERVE_SETTER_ROLE")
}

// RESERVESETTERROLE is a paid mutator transaction binding the contract method 0xb3ee252f.
//
// Solidity: function RESERVE_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) RESERVESETTERROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.RESERVESETTERROLE(&_IExpressLaneAuction.TransactOpts)
}

// RESERVESETTERROLE is a paid mutator transaction binding the contract method 0xb3ee252f.
//
// Solidity: function RESERVE_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) RESERVESETTERROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.RESERVESETTERROLE(&_IExpressLaneAuction.TransactOpts)
}

// ROUNDTIMINGSETTERROLE is a paid mutator transaction binding the contract method 0x1682e50b.
//
// Solidity: function ROUND_TIMING_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) ROUNDTIMINGSETTERROLE(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "ROUND_TIMING_SETTER_ROLE")
}

// ROUNDTIMINGSETTERROLE is a paid mutator transaction binding the contract method 0x1682e50b.
//
// Solidity: function ROUND_TIMING_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) ROUNDTIMINGSETTERROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.ROUNDTIMINGSETTERROLE(&_IExpressLaneAuction.TransactOpts)
}

// ROUNDTIMINGSETTERROLE is a paid mutator transaction binding the contract method 0x1682e50b.
//
// Solidity: function ROUND_TIMING_SETTER_ROLE() returns(bytes32)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) ROUNDTIMINGSETTERROLE() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.ROUNDTIMINGSETTERROLE(&_IExpressLaneAuction.TransactOpts)
}

// Beneficiary is a paid mutator transaction binding the contract method 0x38af3eed.
//
// Solidity: function beneficiary() returns(address)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) Beneficiary(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "beneficiary")
}

// Beneficiary is a paid mutator transaction binding the contract method 0x38af3eed.
//
// Solidity: function beneficiary() returns(address)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) Beneficiary() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.Beneficiary(&_IExpressLaneAuction.TransactOpts)
}

// Beneficiary is a paid mutator transaction binding the contract method 0x38af3eed.
//
// Solidity: function beneficiary() returns(address)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) Beneficiary() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.Beneficiary(&_IExpressLaneAuction.TransactOpts)
}

// BeneficiaryBalance is a paid mutator transaction binding the contract method 0xe2fc6f68.
//
// Solidity: function beneficiaryBalance() returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) BeneficiaryBalance(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "beneficiaryBalance")
}

// BeneficiaryBalance is a paid mutator transaction binding the contract method 0xe2fc6f68.
//
// Solidity: function beneficiaryBalance() returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) BeneficiaryBalance() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.BeneficiaryBalance(&_IExpressLaneAuction.TransactOpts)
}

// BeneficiaryBalance is a paid mutator transaction binding the contract method 0xe2fc6f68.
//
// Solidity: function beneficiaryBalance() returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) BeneficiaryBalance() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.BeneficiaryBalance(&_IExpressLaneAuction.TransactOpts)
}

// BiddingToken is a paid mutator transaction binding the contract method 0x639d7566.
//
// Solidity: function biddingToken() returns(address)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) BiddingToken(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "biddingToken")
}

// BiddingToken is a paid mutator transaction binding the contract method 0x639d7566.
//
// Solidity: function biddingToken() returns(address)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) BiddingToken() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.BiddingToken(&_IExpressLaneAuction.TransactOpts)
}

// BiddingToken is a paid mutator transaction binding the contract method 0x639d7566.
//
// Solidity: function biddingToken() returns(address)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) BiddingToken() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.BiddingToken(&_IExpressLaneAuction.TransactOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(uint256 amount) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) Deposit(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "deposit", amount)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(uint256 amount) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) Deposit(amount *big.Int) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.Deposit(&_IExpressLaneAuction.TransactOpts, amount)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(uint256 amount) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) Deposit(amount *big.Int) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.Deposit(&_IExpressLaneAuction.TransactOpts, amount)
}

// FinalizeWithdrawal is a paid mutator transaction binding the contract method 0xc5b6aa2f.
//
// Solidity: function finalizeWithdrawal() returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) FinalizeWithdrawal(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "finalizeWithdrawal")
}

// FinalizeWithdrawal is a paid mutator transaction binding the contract method 0xc5b6aa2f.
//
// Solidity: function finalizeWithdrawal() returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) FinalizeWithdrawal() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.FinalizeWithdrawal(&_IExpressLaneAuction.TransactOpts)
}

// FinalizeWithdrawal is a paid mutator transaction binding the contract method 0xc5b6aa2f.
//
// Solidity: function finalizeWithdrawal() returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) FinalizeWithdrawal() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.FinalizeWithdrawal(&_IExpressLaneAuction.TransactOpts)
}

// FlushBeneficiaryBalance is a paid mutator transaction binding the contract method 0x6ad72517.
//
// Solidity: function flushBeneficiaryBalance() returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) FlushBeneficiaryBalance(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "flushBeneficiaryBalance")
}

// FlushBeneficiaryBalance is a paid mutator transaction binding the contract method 0x6ad72517.
//
// Solidity: function flushBeneficiaryBalance() returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) FlushBeneficiaryBalance() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.FlushBeneficiaryBalance(&_IExpressLaneAuction.TransactOpts)
}

// FlushBeneficiaryBalance is a paid mutator transaction binding the contract method 0x6ad72517.
//
// Solidity: function flushBeneficiaryBalance() returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) FlushBeneficiaryBalance() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.FlushBeneficiaryBalance(&_IExpressLaneAuction.TransactOpts)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.GrantRole(&_IExpressLaneAuction.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.GrantRole(&_IExpressLaneAuction.TransactOpts, role, account)
}

// Initialize is a paid mutator transaction binding the contract method 0x9a1fadd3.
//
// Solidity: function initialize((address,address,address,(int64,uint64,uint64,uint64),uint256,address,address,address,address,address,address,address) args) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) Initialize(opts *bind.TransactOpts, args InitArgs) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "initialize", args)
}

// Initialize is a paid mutator transaction binding the contract method 0x9a1fadd3.
//
// Solidity: function initialize((address,address,address,(int64,uint64,uint64,uint64),uint256,address,address,address,address,address,address,address) args) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) Initialize(args InitArgs) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.Initialize(&_IExpressLaneAuction.TransactOpts, args)
}

// Initialize is a paid mutator transaction binding the contract method 0x9a1fadd3.
//
// Solidity: function initialize((address,address,address,(int64,uint64,uint64,uint64),uint256,address,address,address,address,address,address,address) args) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) Initialize(args InitArgs) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.Initialize(&_IExpressLaneAuction.TransactOpts, args)
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0xb51d1d4f.
//
// Solidity: function initiateWithdrawal() returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) InitiateWithdrawal(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "initiateWithdrawal")
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0xb51d1d4f.
//
// Solidity: function initiateWithdrawal() returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) InitiateWithdrawal() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.InitiateWithdrawal(&_IExpressLaneAuction.TransactOpts)
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0xb51d1d4f.
//
// Solidity: function initiateWithdrawal() returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) InitiateWithdrawal() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.InitiateWithdrawal(&_IExpressLaneAuction.TransactOpts)
}

// MinReservePrice is a paid mutator transaction binding the contract method 0x83af0a1f.
//
// Solidity: function minReservePrice() returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) MinReservePrice(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "minReservePrice")
}

// MinReservePrice is a paid mutator transaction binding the contract method 0x83af0a1f.
//
// Solidity: function minReservePrice() returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) MinReservePrice() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.MinReservePrice(&_IExpressLaneAuction.TransactOpts)
}

// MinReservePrice is a paid mutator transaction binding the contract method 0x83af0a1f.
//
// Solidity: function minReservePrice() returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) MinReservePrice() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.MinReservePrice(&_IExpressLaneAuction.TransactOpts)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.RenounceRole(&_IExpressLaneAuction.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.RenounceRole(&_IExpressLaneAuction.TransactOpts, role, account)
}

// ReservePrice is a paid mutator transaction binding the contract method 0xdb2e1eed.
//
// Solidity: function reservePrice() returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) ReservePrice(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "reservePrice")
}

// ReservePrice is a paid mutator transaction binding the contract method 0xdb2e1eed.
//
// Solidity: function reservePrice() returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) ReservePrice() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.ReservePrice(&_IExpressLaneAuction.TransactOpts)
}

// ReservePrice is a paid mutator transaction binding the contract method 0xdb2e1eed.
//
// Solidity: function reservePrice() returns(uint256)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) ReservePrice() (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.ReservePrice(&_IExpressLaneAuction.TransactOpts)
}

// ResolveMultiBidAuction is a paid mutator transaction binding the contract method 0x447a709e.
//
// Solidity: function resolveMultiBidAuction((address,uint256,bytes) firstPriceBid, (address,uint256,bytes) secondPriceBid) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) ResolveMultiBidAuction(opts *bind.TransactOpts, firstPriceBid Bid, secondPriceBid Bid) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "resolveMultiBidAuction", firstPriceBid, secondPriceBid)
}

// ResolveMultiBidAuction is a paid mutator transaction binding the contract method 0x447a709e.
//
// Solidity: function resolveMultiBidAuction((address,uint256,bytes) firstPriceBid, (address,uint256,bytes) secondPriceBid) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) ResolveMultiBidAuction(firstPriceBid Bid, secondPriceBid Bid) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.ResolveMultiBidAuction(&_IExpressLaneAuction.TransactOpts, firstPriceBid, secondPriceBid)
}

// ResolveMultiBidAuction is a paid mutator transaction binding the contract method 0x447a709e.
//
// Solidity: function resolveMultiBidAuction((address,uint256,bytes) firstPriceBid, (address,uint256,bytes) secondPriceBid) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) ResolveMultiBidAuction(firstPriceBid Bid, secondPriceBid Bid) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.ResolveMultiBidAuction(&_IExpressLaneAuction.TransactOpts, firstPriceBid, secondPriceBid)
}

// ResolveSingleBidAuction is a paid mutator transaction binding the contract method 0x6dc4fc4e.
//
// Solidity: function resolveSingleBidAuction((address,uint256,bytes) firstPriceBid) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) ResolveSingleBidAuction(opts *bind.TransactOpts, firstPriceBid Bid) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "resolveSingleBidAuction", firstPriceBid)
}

// ResolveSingleBidAuction is a paid mutator transaction binding the contract method 0x6dc4fc4e.
//
// Solidity: function resolveSingleBidAuction((address,uint256,bytes) firstPriceBid) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) ResolveSingleBidAuction(firstPriceBid Bid) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.ResolveSingleBidAuction(&_IExpressLaneAuction.TransactOpts, firstPriceBid)
}

// ResolveSingleBidAuction is a paid mutator transaction binding the contract method 0x6dc4fc4e.
//
// Solidity: function resolveSingleBidAuction((address,uint256,bytes) firstPriceBid) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) ResolveSingleBidAuction(firstPriceBid Bid) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.ResolveSingleBidAuction(&_IExpressLaneAuction.TransactOpts, firstPriceBid)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.RevokeRole(&_IExpressLaneAuction.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.RevokeRole(&_IExpressLaneAuction.TransactOpts, role, account)
}

// SetBeneficiary is a paid mutator transaction binding the contract method 0x1c31f710.
//
// Solidity: function setBeneficiary(address newBeneficiary) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) SetBeneficiary(opts *bind.TransactOpts, newBeneficiary common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "setBeneficiary", newBeneficiary)
}

// SetBeneficiary is a paid mutator transaction binding the contract method 0x1c31f710.
//
// Solidity: function setBeneficiary(address newBeneficiary) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) SetBeneficiary(newBeneficiary common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.SetBeneficiary(&_IExpressLaneAuction.TransactOpts, newBeneficiary)
}

// SetBeneficiary is a paid mutator transaction binding the contract method 0x1c31f710.
//
// Solidity: function setBeneficiary(address newBeneficiary) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) SetBeneficiary(newBeneficiary common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.SetBeneficiary(&_IExpressLaneAuction.TransactOpts, newBeneficiary)
}

// SetMinReservePrice is a paid mutator transaction binding the contract method 0xe4d20c1d.
//
// Solidity: function setMinReservePrice(uint256 newMinReservePrice) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) SetMinReservePrice(opts *bind.TransactOpts, newMinReservePrice *big.Int) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "setMinReservePrice", newMinReservePrice)
}

// SetMinReservePrice is a paid mutator transaction binding the contract method 0xe4d20c1d.
//
// Solidity: function setMinReservePrice(uint256 newMinReservePrice) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) SetMinReservePrice(newMinReservePrice *big.Int) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.SetMinReservePrice(&_IExpressLaneAuction.TransactOpts, newMinReservePrice)
}

// SetMinReservePrice is a paid mutator transaction binding the contract method 0xe4d20c1d.
//
// Solidity: function setMinReservePrice(uint256 newMinReservePrice) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) SetMinReservePrice(newMinReservePrice *big.Int) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.SetMinReservePrice(&_IExpressLaneAuction.TransactOpts, newMinReservePrice)
}

// SetReservePrice is a paid mutator transaction binding the contract method 0xce9c7c0d.
//
// Solidity: function setReservePrice(uint256 newReservePrice) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) SetReservePrice(opts *bind.TransactOpts, newReservePrice *big.Int) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "setReservePrice", newReservePrice)
}

// SetReservePrice is a paid mutator transaction binding the contract method 0xce9c7c0d.
//
// Solidity: function setReservePrice(uint256 newReservePrice) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) SetReservePrice(newReservePrice *big.Int) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.SetReservePrice(&_IExpressLaneAuction.TransactOpts, newReservePrice)
}

// SetReservePrice is a paid mutator transaction binding the contract method 0xce9c7c0d.
//
// Solidity: function setReservePrice(uint256 newReservePrice) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) SetReservePrice(newReservePrice *big.Int) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.SetReservePrice(&_IExpressLaneAuction.TransactOpts, newReservePrice)
}

// SetRoundTimingInfo is a paid mutator transaction binding the contract method 0xfed87be8.
//
// Solidity: function setRoundTimingInfo((int64,uint64,uint64,uint64) newRoundTimingInfo) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) SetRoundTimingInfo(opts *bind.TransactOpts, newRoundTimingInfo RoundTimingInfo) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "setRoundTimingInfo", newRoundTimingInfo)
}

// SetRoundTimingInfo is a paid mutator transaction binding the contract method 0xfed87be8.
//
// Solidity: function setRoundTimingInfo((int64,uint64,uint64,uint64) newRoundTimingInfo) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) SetRoundTimingInfo(newRoundTimingInfo RoundTimingInfo) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.SetRoundTimingInfo(&_IExpressLaneAuction.TransactOpts, newRoundTimingInfo)
}

// SetRoundTimingInfo is a paid mutator transaction binding the contract method 0xfed87be8.
//
// Solidity: function setRoundTimingInfo((int64,uint64,uint64,uint64) newRoundTimingInfo) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) SetRoundTimingInfo(newRoundTimingInfo RoundTimingInfo) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.SetRoundTimingInfo(&_IExpressLaneAuction.TransactOpts, newRoundTimingInfo)
}

// SetTransferor is a paid mutator transaction binding the contract method 0xbef0ec74.
//
// Solidity: function setTransferor((address,uint64) transferor) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) SetTransferor(opts *bind.TransactOpts, transferor Transferor) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "setTransferor", transferor)
}

// SetTransferor is a paid mutator transaction binding the contract method 0xbef0ec74.
//
// Solidity: function setTransferor((address,uint64) transferor) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) SetTransferor(transferor Transferor) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.SetTransferor(&_IExpressLaneAuction.TransactOpts, transferor)
}

// SetTransferor is a paid mutator transaction binding the contract method 0xbef0ec74.
//
// Solidity: function setTransferor((address,uint64) transferor) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) SetTransferor(transferor Transferor) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.SetTransferor(&_IExpressLaneAuction.TransactOpts, transferor)
}

// TransferExpressLaneController is a paid mutator transaction binding the contract method 0x007be2fe.
//
// Solidity: function transferExpressLaneController(uint64 round, address newExpressLaneController) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) TransferExpressLaneController(opts *bind.TransactOpts, round uint64, newExpressLaneController common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "transferExpressLaneController", round, newExpressLaneController)
}

// TransferExpressLaneController is a paid mutator transaction binding the contract method 0x007be2fe.
//
// Solidity: function transferExpressLaneController(uint64 round, address newExpressLaneController) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionSession) TransferExpressLaneController(round uint64, newExpressLaneController common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.TransferExpressLaneController(&_IExpressLaneAuction.TransactOpts, round, newExpressLaneController)
}

// TransferExpressLaneController is a paid mutator transaction binding the contract method 0x007be2fe.
//
// Solidity: function transferExpressLaneController(uint64 round, address newExpressLaneController) returns()
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) TransferExpressLaneController(round uint64, newExpressLaneController common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.TransferExpressLaneController(&_IExpressLaneAuction.TransactOpts, round, newExpressLaneController)
}

// TransferorOf is a paid mutator transaction binding the contract method 0x6a514beb.
//
// Solidity: function transferorOf(address expressLaneController) returns(address addr, uint64 fixedUntil)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactor) TransferorOf(opts *bind.TransactOpts, expressLaneController common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.contract.Transact(opts, "transferorOf", expressLaneController)
}

// TransferorOf is a paid mutator transaction binding the contract method 0x6a514beb.
//
// Solidity: function transferorOf(address expressLaneController) returns(address addr, uint64 fixedUntil)
func (_IExpressLaneAuction *IExpressLaneAuctionSession) TransferorOf(expressLaneController common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.TransferorOf(&_IExpressLaneAuction.TransactOpts, expressLaneController)
}

// TransferorOf is a paid mutator transaction binding the contract method 0x6a514beb.
//
// Solidity: function transferorOf(address expressLaneController) returns(address addr, uint64 fixedUntil)
func (_IExpressLaneAuction *IExpressLaneAuctionTransactorSession) TransferorOf(expressLaneController common.Address) (*types.Transaction, error) {
	return _IExpressLaneAuction.Contract.TransferorOf(&_IExpressLaneAuction.TransactOpts, expressLaneController)
}

// IExpressLaneAuctionAuctionResolvedIterator is returned from FilterAuctionResolved and is used to iterate over the raw logs and unpacked data for AuctionResolved events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionAuctionResolvedIterator struct {
	Event *IExpressLaneAuctionAuctionResolved // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionAuctionResolvedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionAuctionResolved)
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
		it.Event = new(IExpressLaneAuctionAuctionResolved)
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
func (it *IExpressLaneAuctionAuctionResolvedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionAuctionResolvedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionAuctionResolved represents a AuctionResolved event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionAuctionResolved struct {
	IsMultiBidAuction               bool
	Round                           uint64
	FirstPriceBidder                common.Address
	FirstPriceExpressLaneController common.Address
	FirstPriceAmount                *big.Int
	Price                           *big.Int
	RoundStartTimestamp             uint64
	RoundEndTimestamp               uint64
	Raw                             types.Log // Blockchain specific contextual infos
}

// FilterAuctionResolved is a free log retrieval operation binding the contract event 0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47.
//
// Solidity: event AuctionResolved(bool indexed isMultiBidAuction, uint64 round, address indexed firstPriceBidder, address indexed firstPriceExpressLaneController, uint256 firstPriceAmount, uint256 price, uint64 roundStartTimestamp, uint64 roundEndTimestamp)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterAuctionResolved(opts *bind.FilterOpts, isMultiBidAuction []bool, firstPriceBidder []common.Address, firstPriceExpressLaneController []common.Address) (*IExpressLaneAuctionAuctionResolvedIterator, error) {

	var isMultiBidAuctionRule []interface{}
	for _, isMultiBidAuctionItem := range isMultiBidAuction {
		isMultiBidAuctionRule = append(isMultiBidAuctionRule, isMultiBidAuctionItem)
	}

	var firstPriceBidderRule []interface{}
	for _, firstPriceBidderItem := range firstPriceBidder {
		firstPriceBidderRule = append(firstPriceBidderRule, firstPriceBidderItem)
	}
	var firstPriceExpressLaneControllerRule []interface{}
	for _, firstPriceExpressLaneControllerItem := range firstPriceExpressLaneController {
		firstPriceExpressLaneControllerRule = append(firstPriceExpressLaneControllerRule, firstPriceExpressLaneControllerItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "AuctionResolved", isMultiBidAuctionRule, firstPriceBidderRule, firstPriceExpressLaneControllerRule)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionAuctionResolvedIterator{contract: _IExpressLaneAuction.contract, event: "AuctionResolved", logs: logs, sub: sub}, nil
}

// WatchAuctionResolved is a free log subscription operation binding the contract event 0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47.
//
// Solidity: event AuctionResolved(bool indexed isMultiBidAuction, uint64 round, address indexed firstPriceBidder, address indexed firstPriceExpressLaneController, uint256 firstPriceAmount, uint256 price, uint64 roundStartTimestamp, uint64 roundEndTimestamp)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchAuctionResolved(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionAuctionResolved, isMultiBidAuction []bool, firstPriceBidder []common.Address, firstPriceExpressLaneController []common.Address) (event.Subscription, error) {

	var isMultiBidAuctionRule []interface{}
	for _, isMultiBidAuctionItem := range isMultiBidAuction {
		isMultiBidAuctionRule = append(isMultiBidAuctionRule, isMultiBidAuctionItem)
	}

	var firstPriceBidderRule []interface{}
	for _, firstPriceBidderItem := range firstPriceBidder {
		firstPriceBidderRule = append(firstPriceBidderRule, firstPriceBidderItem)
	}
	var firstPriceExpressLaneControllerRule []interface{}
	for _, firstPriceExpressLaneControllerItem := range firstPriceExpressLaneController {
		firstPriceExpressLaneControllerRule = append(firstPriceExpressLaneControllerRule, firstPriceExpressLaneControllerItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "AuctionResolved", isMultiBidAuctionRule, firstPriceBidderRule, firstPriceExpressLaneControllerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionAuctionResolved)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "AuctionResolved", log); err != nil {
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

// ParseAuctionResolved is a log parse operation binding the contract event 0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47.
//
// Solidity: event AuctionResolved(bool indexed isMultiBidAuction, uint64 round, address indexed firstPriceBidder, address indexed firstPriceExpressLaneController, uint256 firstPriceAmount, uint256 price, uint64 roundStartTimestamp, uint64 roundEndTimestamp)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseAuctionResolved(log types.Log) (*IExpressLaneAuctionAuctionResolved, error) {
	event := new(IExpressLaneAuctionAuctionResolved)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "AuctionResolved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionDepositIterator is returned from FilterDeposit and is used to iterate over the raw logs and unpacked data for Deposit events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionDepositIterator struct {
	Event *IExpressLaneAuctionDeposit // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionDeposit)
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
		it.Event = new(IExpressLaneAuctionDeposit)
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
func (it *IExpressLaneAuctionDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionDeposit represents a Deposit event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionDeposit struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterDeposit is a free log retrieval operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed account, uint256 amount)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterDeposit(opts *bind.FilterOpts, account []common.Address) (*IExpressLaneAuctionDepositIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "Deposit", accountRule)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionDepositIterator{contract: _IExpressLaneAuction.contract, event: "Deposit", logs: logs, sub: sub}, nil
}

// WatchDeposit is a free log subscription operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed account, uint256 amount)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchDeposit(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionDeposit, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "Deposit", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionDeposit)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "Deposit", log); err != nil {
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

// ParseDeposit is a log parse operation binding the contract event 0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c.
//
// Solidity: event Deposit(address indexed account, uint256 amount)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseDeposit(log types.Log) (*IExpressLaneAuctionDeposit, error) {
	event := new(IExpressLaneAuctionDeposit)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "Deposit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionRoleAdminChangedIterator struct {
	Event *IExpressLaneAuctionRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionRoleAdminChanged)
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
		it.Event = new(IExpressLaneAuctionRoleAdminChanged)
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
func (it *IExpressLaneAuctionRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionRoleAdminChanged represents a RoleAdminChanged event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*IExpressLaneAuctionRoleAdminChangedIterator, error) {

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

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionRoleAdminChangedIterator{contract: _IExpressLaneAuction.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionRoleAdminChanged)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseRoleAdminChanged(log types.Log) (*IExpressLaneAuctionRoleAdminChanged, error) {
	event := new(IExpressLaneAuctionRoleAdminChanged)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionRoleGrantedIterator struct {
	Event *IExpressLaneAuctionRoleGranted // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionRoleGranted)
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
		it.Event = new(IExpressLaneAuctionRoleGranted)
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
func (it *IExpressLaneAuctionRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionRoleGranted represents a RoleGranted event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*IExpressLaneAuctionRoleGrantedIterator, error) {

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

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionRoleGrantedIterator{contract: _IExpressLaneAuction.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionRoleGranted)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseRoleGranted(log types.Log) (*IExpressLaneAuctionRoleGranted, error) {
	event := new(IExpressLaneAuctionRoleGranted)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionRoleRevokedIterator struct {
	Event *IExpressLaneAuctionRoleRevoked // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionRoleRevoked)
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
		it.Event = new(IExpressLaneAuctionRoleRevoked)
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
func (it *IExpressLaneAuctionRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionRoleRevoked represents a RoleRevoked event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*IExpressLaneAuctionRoleRevokedIterator, error) {

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

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionRoleRevokedIterator{contract: _IExpressLaneAuction.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionRoleRevoked)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseRoleRevoked(log types.Log) (*IExpressLaneAuctionRoleRevoked, error) {
	event := new(IExpressLaneAuctionRoleRevoked)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionSetBeneficiaryIterator is returned from FilterSetBeneficiary and is used to iterate over the raw logs and unpacked data for SetBeneficiary events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetBeneficiaryIterator struct {
	Event *IExpressLaneAuctionSetBeneficiary // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionSetBeneficiaryIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionSetBeneficiary)
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
		it.Event = new(IExpressLaneAuctionSetBeneficiary)
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
func (it *IExpressLaneAuctionSetBeneficiaryIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionSetBeneficiaryIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionSetBeneficiary represents a SetBeneficiary event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetBeneficiary struct {
	OldBeneficiary common.Address
	NewBeneficiary common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterSetBeneficiary is a free log retrieval operation binding the contract event 0x8a0149b2f3ddf2c9ee85738165131d82babbb938f749321d59f75750afa7f4e6.
//
// Solidity: event SetBeneficiary(address oldBeneficiary, address newBeneficiary)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterSetBeneficiary(opts *bind.FilterOpts) (*IExpressLaneAuctionSetBeneficiaryIterator, error) {

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "SetBeneficiary")
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionSetBeneficiaryIterator{contract: _IExpressLaneAuction.contract, event: "SetBeneficiary", logs: logs, sub: sub}, nil
}

// WatchSetBeneficiary is a free log subscription operation binding the contract event 0x8a0149b2f3ddf2c9ee85738165131d82babbb938f749321d59f75750afa7f4e6.
//
// Solidity: event SetBeneficiary(address oldBeneficiary, address newBeneficiary)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchSetBeneficiary(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionSetBeneficiary) (event.Subscription, error) {

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "SetBeneficiary")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionSetBeneficiary)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetBeneficiary", log); err != nil {
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

// ParseSetBeneficiary is a log parse operation binding the contract event 0x8a0149b2f3ddf2c9ee85738165131d82babbb938f749321d59f75750afa7f4e6.
//
// Solidity: event SetBeneficiary(address oldBeneficiary, address newBeneficiary)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseSetBeneficiary(log types.Log) (*IExpressLaneAuctionSetBeneficiary, error) {
	event := new(IExpressLaneAuctionSetBeneficiary)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetBeneficiary", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionSetExpressLaneControllerIterator is returned from FilterSetExpressLaneController and is used to iterate over the raw logs and unpacked data for SetExpressLaneController events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetExpressLaneControllerIterator struct {
	Event *IExpressLaneAuctionSetExpressLaneController // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionSetExpressLaneControllerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionSetExpressLaneController)
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
		it.Event = new(IExpressLaneAuctionSetExpressLaneController)
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
func (it *IExpressLaneAuctionSetExpressLaneControllerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionSetExpressLaneControllerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionSetExpressLaneController represents a SetExpressLaneController event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetExpressLaneController struct {
	Round                         uint64
	PreviousExpressLaneController common.Address
	NewExpressLaneController      common.Address
	Transferor                    common.Address
	StartTimestamp                uint64
	EndTimestamp                  uint64
	Raw                           types.Log // Blockchain specific contextual infos
}

// FilterSetExpressLaneController is a free log retrieval operation binding the contract event 0xb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b.
//
// Solidity: event SetExpressLaneController(uint64 round, address indexed previousExpressLaneController, address indexed newExpressLaneController, address indexed transferor, uint64 startTimestamp, uint64 endTimestamp)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterSetExpressLaneController(opts *bind.FilterOpts, previousExpressLaneController []common.Address, newExpressLaneController []common.Address, transferor []common.Address) (*IExpressLaneAuctionSetExpressLaneControllerIterator, error) {

	var previousExpressLaneControllerRule []interface{}
	for _, previousExpressLaneControllerItem := range previousExpressLaneController {
		previousExpressLaneControllerRule = append(previousExpressLaneControllerRule, previousExpressLaneControllerItem)
	}
	var newExpressLaneControllerRule []interface{}
	for _, newExpressLaneControllerItem := range newExpressLaneController {
		newExpressLaneControllerRule = append(newExpressLaneControllerRule, newExpressLaneControllerItem)
	}
	var transferorRule []interface{}
	for _, transferorItem := range transferor {
		transferorRule = append(transferorRule, transferorItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "SetExpressLaneController", previousExpressLaneControllerRule, newExpressLaneControllerRule, transferorRule)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionSetExpressLaneControllerIterator{contract: _IExpressLaneAuction.contract, event: "SetExpressLaneController", logs: logs, sub: sub}, nil
}

// WatchSetExpressLaneController is a free log subscription operation binding the contract event 0xb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b.
//
// Solidity: event SetExpressLaneController(uint64 round, address indexed previousExpressLaneController, address indexed newExpressLaneController, address indexed transferor, uint64 startTimestamp, uint64 endTimestamp)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchSetExpressLaneController(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionSetExpressLaneController, previousExpressLaneController []common.Address, newExpressLaneController []common.Address, transferor []common.Address) (event.Subscription, error) {

	var previousExpressLaneControllerRule []interface{}
	for _, previousExpressLaneControllerItem := range previousExpressLaneController {
		previousExpressLaneControllerRule = append(previousExpressLaneControllerRule, previousExpressLaneControllerItem)
	}
	var newExpressLaneControllerRule []interface{}
	for _, newExpressLaneControllerItem := range newExpressLaneController {
		newExpressLaneControllerRule = append(newExpressLaneControllerRule, newExpressLaneControllerItem)
	}
	var transferorRule []interface{}
	for _, transferorItem := range transferor {
		transferorRule = append(transferorRule, transferorItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "SetExpressLaneController", previousExpressLaneControllerRule, newExpressLaneControllerRule, transferorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionSetExpressLaneController)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetExpressLaneController", log); err != nil {
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

// ParseSetExpressLaneController is a log parse operation binding the contract event 0xb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b.
//
// Solidity: event SetExpressLaneController(uint64 round, address indexed previousExpressLaneController, address indexed newExpressLaneController, address indexed transferor, uint64 startTimestamp, uint64 endTimestamp)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseSetExpressLaneController(log types.Log) (*IExpressLaneAuctionSetExpressLaneController, error) {
	event := new(IExpressLaneAuctionSetExpressLaneController)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetExpressLaneController", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionSetMinReservePriceIterator is returned from FilterSetMinReservePrice and is used to iterate over the raw logs and unpacked data for SetMinReservePrice events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetMinReservePriceIterator struct {
	Event *IExpressLaneAuctionSetMinReservePrice // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionSetMinReservePriceIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionSetMinReservePrice)
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
		it.Event = new(IExpressLaneAuctionSetMinReservePrice)
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
func (it *IExpressLaneAuctionSetMinReservePriceIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionSetMinReservePriceIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionSetMinReservePrice represents a SetMinReservePrice event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetMinReservePrice struct {
	OldPrice *big.Int
	NewPrice *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterSetMinReservePrice is a free log retrieval operation binding the contract event 0x5848068f11aa3ba9fe3fc33c5f9f2a3cd1aed67986b85b5e0cedc67dbe96f0f0.
//
// Solidity: event SetMinReservePrice(uint256 oldPrice, uint256 newPrice)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterSetMinReservePrice(opts *bind.FilterOpts) (*IExpressLaneAuctionSetMinReservePriceIterator, error) {

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "SetMinReservePrice")
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionSetMinReservePriceIterator{contract: _IExpressLaneAuction.contract, event: "SetMinReservePrice", logs: logs, sub: sub}, nil
}

// WatchSetMinReservePrice is a free log subscription operation binding the contract event 0x5848068f11aa3ba9fe3fc33c5f9f2a3cd1aed67986b85b5e0cedc67dbe96f0f0.
//
// Solidity: event SetMinReservePrice(uint256 oldPrice, uint256 newPrice)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchSetMinReservePrice(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionSetMinReservePrice) (event.Subscription, error) {

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "SetMinReservePrice")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionSetMinReservePrice)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetMinReservePrice", log); err != nil {
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

// ParseSetMinReservePrice is a log parse operation binding the contract event 0x5848068f11aa3ba9fe3fc33c5f9f2a3cd1aed67986b85b5e0cedc67dbe96f0f0.
//
// Solidity: event SetMinReservePrice(uint256 oldPrice, uint256 newPrice)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseSetMinReservePrice(log types.Log) (*IExpressLaneAuctionSetMinReservePrice, error) {
	event := new(IExpressLaneAuctionSetMinReservePrice)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetMinReservePrice", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionSetReservePriceIterator is returned from FilterSetReservePrice and is used to iterate over the raw logs and unpacked data for SetReservePrice events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetReservePriceIterator struct {
	Event *IExpressLaneAuctionSetReservePrice // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionSetReservePriceIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionSetReservePrice)
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
		it.Event = new(IExpressLaneAuctionSetReservePrice)
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
func (it *IExpressLaneAuctionSetReservePriceIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionSetReservePriceIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionSetReservePrice represents a SetReservePrice event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetReservePrice struct {
	OldReservePrice *big.Int
	NewReservePrice *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterSetReservePrice is a free log retrieval operation binding the contract event 0x9725e37e079c5bda6009a8f54d86265849f30acf61c630f9e1ac91e67de98794.
//
// Solidity: event SetReservePrice(uint256 oldReservePrice, uint256 newReservePrice)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterSetReservePrice(opts *bind.FilterOpts) (*IExpressLaneAuctionSetReservePriceIterator, error) {

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "SetReservePrice")
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionSetReservePriceIterator{contract: _IExpressLaneAuction.contract, event: "SetReservePrice", logs: logs, sub: sub}, nil
}

// WatchSetReservePrice is a free log subscription operation binding the contract event 0x9725e37e079c5bda6009a8f54d86265849f30acf61c630f9e1ac91e67de98794.
//
// Solidity: event SetReservePrice(uint256 oldReservePrice, uint256 newReservePrice)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchSetReservePrice(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionSetReservePrice) (event.Subscription, error) {

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "SetReservePrice")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionSetReservePrice)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetReservePrice", log); err != nil {
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

// ParseSetReservePrice is a log parse operation binding the contract event 0x9725e37e079c5bda6009a8f54d86265849f30acf61c630f9e1ac91e67de98794.
//
// Solidity: event SetReservePrice(uint256 oldReservePrice, uint256 newReservePrice)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseSetReservePrice(log types.Log) (*IExpressLaneAuctionSetReservePrice, error) {
	event := new(IExpressLaneAuctionSetReservePrice)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetReservePrice", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionSetRoundTimingInfoIterator is returned from FilterSetRoundTimingInfo and is used to iterate over the raw logs and unpacked data for SetRoundTimingInfo events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetRoundTimingInfoIterator struct {
	Event *IExpressLaneAuctionSetRoundTimingInfo // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionSetRoundTimingInfoIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionSetRoundTimingInfo)
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
		it.Event = new(IExpressLaneAuctionSetRoundTimingInfo)
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
func (it *IExpressLaneAuctionSetRoundTimingInfoIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionSetRoundTimingInfoIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionSetRoundTimingInfo represents a SetRoundTimingInfo event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetRoundTimingInfo struct {
	CurrentRound             uint64
	OffsetTimestamp          int64
	RoundDurationSeconds     uint64
	AuctionClosingSeconds    uint64
	ReserveSubmissionSeconds uint64
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterSetRoundTimingInfo is a free log retrieval operation binding the contract event 0x982cfb73783b8c64455c76cdeb1351467c4f1e6b3615fec07df232c1b46ffd47.
//
// Solidity: event SetRoundTimingInfo(uint64 currentRound, int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterSetRoundTimingInfo(opts *bind.FilterOpts) (*IExpressLaneAuctionSetRoundTimingInfoIterator, error) {

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "SetRoundTimingInfo")
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionSetRoundTimingInfoIterator{contract: _IExpressLaneAuction.contract, event: "SetRoundTimingInfo", logs: logs, sub: sub}, nil
}

// WatchSetRoundTimingInfo is a free log subscription operation binding the contract event 0x982cfb73783b8c64455c76cdeb1351467c4f1e6b3615fec07df232c1b46ffd47.
//
// Solidity: event SetRoundTimingInfo(uint64 currentRound, int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchSetRoundTimingInfo(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionSetRoundTimingInfo) (event.Subscription, error) {

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "SetRoundTimingInfo")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionSetRoundTimingInfo)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetRoundTimingInfo", log); err != nil {
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

// ParseSetRoundTimingInfo is a log parse operation binding the contract event 0x982cfb73783b8c64455c76cdeb1351467c4f1e6b3615fec07df232c1b46ffd47.
//
// Solidity: event SetRoundTimingInfo(uint64 currentRound, int64 offsetTimestamp, uint64 roundDurationSeconds, uint64 auctionClosingSeconds, uint64 reserveSubmissionSeconds)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseSetRoundTimingInfo(log types.Log) (*IExpressLaneAuctionSetRoundTimingInfo, error) {
	event := new(IExpressLaneAuctionSetRoundTimingInfo)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetRoundTimingInfo", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionSetTransferorIterator is returned from FilterSetTransferor and is used to iterate over the raw logs and unpacked data for SetTransferor events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetTransferorIterator struct {
	Event *IExpressLaneAuctionSetTransferor // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionSetTransferorIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionSetTransferor)
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
		it.Event = new(IExpressLaneAuctionSetTransferor)
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
func (it *IExpressLaneAuctionSetTransferorIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionSetTransferorIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionSetTransferor represents a SetTransferor event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionSetTransferor struct {
	ExpressLaneController common.Address
	Transferor            common.Address
	FixedUntilRound       uint64
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterSetTransferor is a free log retrieval operation binding the contract event 0xf6d28df235d9fa45a42d45dbb7c4f4ac76edb51e528f09f25a0650d32b8b33c0.
//
// Solidity: event SetTransferor(address indexed expressLaneController, address indexed transferor, uint64 fixedUntilRound)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterSetTransferor(opts *bind.FilterOpts, expressLaneController []common.Address, transferor []common.Address) (*IExpressLaneAuctionSetTransferorIterator, error) {

	var expressLaneControllerRule []interface{}
	for _, expressLaneControllerItem := range expressLaneController {
		expressLaneControllerRule = append(expressLaneControllerRule, expressLaneControllerItem)
	}
	var transferorRule []interface{}
	for _, transferorItem := range transferor {
		transferorRule = append(transferorRule, transferorItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "SetTransferor", expressLaneControllerRule, transferorRule)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionSetTransferorIterator{contract: _IExpressLaneAuction.contract, event: "SetTransferor", logs: logs, sub: sub}, nil
}

// WatchSetTransferor is a free log subscription operation binding the contract event 0xf6d28df235d9fa45a42d45dbb7c4f4ac76edb51e528f09f25a0650d32b8b33c0.
//
// Solidity: event SetTransferor(address indexed expressLaneController, address indexed transferor, uint64 fixedUntilRound)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchSetTransferor(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionSetTransferor, expressLaneController []common.Address, transferor []common.Address) (event.Subscription, error) {

	var expressLaneControllerRule []interface{}
	for _, expressLaneControllerItem := range expressLaneController {
		expressLaneControllerRule = append(expressLaneControllerRule, expressLaneControllerItem)
	}
	var transferorRule []interface{}
	for _, transferorItem := range transferor {
		transferorRule = append(transferorRule, transferorItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "SetTransferor", expressLaneControllerRule, transferorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionSetTransferor)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetTransferor", log); err != nil {
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

// ParseSetTransferor is a log parse operation binding the contract event 0xf6d28df235d9fa45a42d45dbb7c4f4ac76edb51e528f09f25a0650d32b8b33c0.
//
// Solidity: event SetTransferor(address indexed expressLaneController, address indexed transferor, uint64 fixedUntilRound)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseSetTransferor(log types.Log) (*IExpressLaneAuctionSetTransferor, error) {
	event := new(IExpressLaneAuctionSetTransferor)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "SetTransferor", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionWithdrawalFinalizedIterator is returned from FilterWithdrawalFinalized and is used to iterate over the raw logs and unpacked data for WithdrawalFinalized events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionWithdrawalFinalizedIterator struct {
	Event *IExpressLaneAuctionWithdrawalFinalized // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionWithdrawalFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionWithdrawalFinalized)
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
		it.Event = new(IExpressLaneAuctionWithdrawalFinalized)
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
func (it *IExpressLaneAuctionWithdrawalFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionWithdrawalFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionWithdrawalFinalized represents a WithdrawalFinalized event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionWithdrawalFinalized struct {
	Account          common.Address
	WithdrawalAmount *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalFinalized is a free log retrieval operation binding the contract event 0x9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda8.
//
// Solidity: event WithdrawalFinalized(address indexed account, uint256 withdrawalAmount)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterWithdrawalFinalized(opts *bind.FilterOpts, account []common.Address) (*IExpressLaneAuctionWithdrawalFinalizedIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "WithdrawalFinalized", accountRule)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionWithdrawalFinalizedIterator{contract: _IExpressLaneAuction.contract, event: "WithdrawalFinalized", logs: logs, sub: sub}, nil
}

// WatchWithdrawalFinalized is a free log subscription operation binding the contract event 0x9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda8.
//
// Solidity: event WithdrawalFinalized(address indexed account, uint256 withdrawalAmount)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchWithdrawalFinalized(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionWithdrawalFinalized, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "WithdrawalFinalized", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionWithdrawalFinalized)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
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

// ParseWithdrawalFinalized is a log parse operation binding the contract event 0x9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda8.
//
// Solidity: event WithdrawalFinalized(address indexed account, uint256 withdrawalAmount)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseWithdrawalFinalized(log types.Log) (*IExpressLaneAuctionWithdrawalFinalized, error) {
	event := new(IExpressLaneAuctionWithdrawalFinalized)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IExpressLaneAuctionWithdrawalInitiatedIterator is returned from FilterWithdrawalInitiated and is used to iterate over the raw logs and unpacked data for WithdrawalInitiated events raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionWithdrawalInitiatedIterator struct {
	Event *IExpressLaneAuctionWithdrawalInitiated // Event containing the contract specifics and raw log

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
func (it *IExpressLaneAuctionWithdrawalInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IExpressLaneAuctionWithdrawalInitiated)
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
		it.Event = new(IExpressLaneAuctionWithdrawalInitiated)
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
func (it *IExpressLaneAuctionWithdrawalInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IExpressLaneAuctionWithdrawalInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IExpressLaneAuctionWithdrawalInitiated represents a WithdrawalInitiated event raised by the IExpressLaneAuction contract.
type IExpressLaneAuctionWithdrawalInitiated struct {
	Account           common.Address
	WithdrawalAmount  *big.Int
	RoundWithdrawable *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalInitiated is a free log retrieval operation binding the contract event 0x31f69201fab7912e3ec9850e3ab705964bf46d9d4276bdcbb6d05e965e5f5401.
//
// Solidity: event WithdrawalInitiated(address indexed account, uint256 withdrawalAmount, uint256 roundWithdrawable)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) FilterWithdrawalInitiated(opts *bind.FilterOpts, account []common.Address) (*IExpressLaneAuctionWithdrawalInitiatedIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.FilterLogs(opts, "WithdrawalInitiated", accountRule)
	if err != nil {
		return nil, err
	}
	return &IExpressLaneAuctionWithdrawalInitiatedIterator{contract: _IExpressLaneAuction.contract, event: "WithdrawalInitiated", logs: logs, sub: sub}, nil
}

// WatchWithdrawalInitiated is a free log subscription operation binding the contract event 0x31f69201fab7912e3ec9850e3ab705964bf46d9d4276bdcbb6d05e965e5f5401.
//
// Solidity: event WithdrawalInitiated(address indexed account, uint256 withdrawalAmount, uint256 roundWithdrawable)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) WatchWithdrawalInitiated(opts *bind.WatchOpts, sink chan<- *IExpressLaneAuctionWithdrawalInitiated, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _IExpressLaneAuction.contract.WatchLogs(opts, "WithdrawalInitiated", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IExpressLaneAuctionWithdrawalInitiated)
				if err := _IExpressLaneAuction.contract.UnpackLog(event, "WithdrawalInitiated", log); err != nil {
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

// ParseWithdrawalInitiated is a log parse operation binding the contract event 0x31f69201fab7912e3ec9850e3ab705964bf46d9d4276bdcbb6d05e965e5f5401.
//
// Solidity: event WithdrawalInitiated(address indexed account, uint256 withdrawalAmount, uint256 roundWithdrawable)
func (_IExpressLaneAuction *IExpressLaneAuctionFilterer) ParseWithdrawalInitiated(log types.Log) (*IExpressLaneAuctionWithdrawalInitiated, error) {
	event := new(IExpressLaneAuctionWithdrawalInitiated)
	if err := _IExpressLaneAuction.contract.UnpackLog(event, "WithdrawalInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LatestELCRoundsLibMetaData contains all meta data concerning the LatestELCRoundsLib contract.
var LatestELCRoundsLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212205149eb104e8bfd1c6cd79377a8e5710161f5ae9f2d3b715e9cd1f757cd14569064736f6c63430008110033",
}

// LatestELCRoundsLibABI is the input ABI used to generate the binding from.
// Deprecated: Use LatestELCRoundsLibMetaData.ABI instead.
var LatestELCRoundsLibABI = LatestELCRoundsLibMetaData.ABI

// LatestELCRoundsLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use LatestELCRoundsLibMetaData.Bin instead.
var LatestELCRoundsLibBin = LatestELCRoundsLibMetaData.Bin

// DeployLatestELCRoundsLib deploys a new Ethereum contract, binding an instance of LatestELCRoundsLib to it.
func DeployLatestELCRoundsLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LatestELCRoundsLib, error) {
	parsed, err := LatestELCRoundsLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(LatestELCRoundsLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LatestELCRoundsLib{LatestELCRoundsLibCaller: LatestELCRoundsLibCaller{contract: contract}, LatestELCRoundsLibTransactor: LatestELCRoundsLibTransactor{contract: contract}, LatestELCRoundsLibFilterer: LatestELCRoundsLibFilterer{contract: contract}}, nil
}

// LatestELCRoundsLib is an auto generated Go binding around an Ethereum contract.
type LatestELCRoundsLib struct {
	LatestELCRoundsLibCaller     // Read-only binding to the contract
	LatestELCRoundsLibTransactor // Write-only binding to the contract
	LatestELCRoundsLibFilterer   // Log filterer for contract events
}

// LatestELCRoundsLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type LatestELCRoundsLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LatestELCRoundsLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LatestELCRoundsLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LatestELCRoundsLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LatestELCRoundsLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LatestELCRoundsLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LatestELCRoundsLibSession struct {
	Contract     *LatestELCRoundsLib // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// LatestELCRoundsLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LatestELCRoundsLibCallerSession struct {
	Contract *LatestELCRoundsLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// LatestELCRoundsLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LatestELCRoundsLibTransactorSession struct {
	Contract     *LatestELCRoundsLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// LatestELCRoundsLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type LatestELCRoundsLibRaw struct {
	Contract *LatestELCRoundsLib // Generic contract binding to access the raw methods on
}

// LatestELCRoundsLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LatestELCRoundsLibCallerRaw struct {
	Contract *LatestELCRoundsLibCaller // Generic read-only contract binding to access the raw methods on
}

// LatestELCRoundsLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LatestELCRoundsLibTransactorRaw struct {
	Contract *LatestELCRoundsLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLatestELCRoundsLib creates a new instance of LatestELCRoundsLib, bound to a specific deployed contract.
func NewLatestELCRoundsLib(address common.Address, backend bind.ContractBackend) (*LatestELCRoundsLib, error) {
	contract, err := bindLatestELCRoundsLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LatestELCRoundsLib{LatestELCRoundsLibCaller: LatestELCRoundsLibCaller{contract: contract}, LatestELCRoundsLibTransactor: LatestELCRoundsLibTransactor{contract: contract}, LatestELCRoundsLibFilterer: LatestELCRoundsLibFilterer{contract: contract}}, nil
}

// NewLatestELCRoundsLibCaller creates a new read-only instance of LatestELCRoundsLib, bound to a specific deployed contract.
func NewLatestELCRoundsLibCaller(address common.Address, caller bind.ContractCaller) (*LatestELCRoundsLibCaller, error) {
	contract, err := bindLatestELCRoundsLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LatestELCRoundsLibCaller{contract: contract}, nil
}

// NewLatestELCRoundsLibTransactor creates a new write-only instance of LatestELCRoundsLib, bound to a specific deployed contract.
func NewLatestELCRoundsLibTransactor(address common.Address, transactor bind.ContractTransactor) (*LatestELCRoundsLibTransactor, error) {
	contract, err := bindLatestELCRoundsLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LatestELCRoundsLibTransactor{contract: contract}, nil
}

// NewLatestELCRoundsLibFilterer creates a new log filterer instance of LatestELCRoundsLib, bound to a specific deployed contract.
func NewLatestELCRoundsLibFilterer(address common.Address, filterer bind.ContractFilterer) (*LatestELCRoundsLibFilterer, error) {
	contract, err := bindLatestELCRoundsLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LatestELCRoundsLibFilterer{contract: contract}, nil
}

// bindLatestELCRoundsLib binds a generic wrapper to an already deployed contract.
func bindLatestELCRoundsLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := LatestELCRoundsLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LatestELCRoundsLib *LatestELCRoundsLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LatestELCRoundsLib.Contract.LatestELCRoundsLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LatestELCRoundsLib *LatestELCRoundsLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LatestELCRoundsLib.Contract.LatestELCRoundsLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LatestELCRoundsLib *LatestELCRoundsLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LatestELCRoundsLib.Contract.LatestELCRoundsLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LatestELCRoundsLib *LatestELCRoundsLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LatestELCRoundsLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LatestELCRoundsLib *LatestELCRoundsLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LatestELCRoundsLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LatestELCRoundsLib *LatestELCRoundsLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LatestELCRoundsLib.Contract.contract.Transact(opts, method, params...)
}

// RoundTimingInfoLibMetaData contains all meta data concerning the RoundTimingInfoLib contract.
var RoundTimingInfoLibMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566037600b82828239805160001a607314602a57634e487b7160e01b600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212208045a4c4c59960fbfd1cb02dad8291d7400ec4475eab6e3d573374409f19f7ab64736f6c63430008110033",
}

// RoundTimingInfoLibABI is the input ABI used to generate the binding from.
// Deprecated: Use RoundTimingInfoLibMetaData.ABI instead.
var RoundTimingInfoLibABI = RoundTimingInfoLibMetaData.ABI

// RoundTimingInfoLibBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use RoundTimingInfoLibMetaData.Bin instead.
var RoundTimingInfoLibBin = RoundTimingInfoLibMetaData.Bin

// DeployRoundTimingInfoLib deploys a new Ethereum contract, binding an instance of RoundTimingInfoLib to it.
func DeployRoundTimingInfoLib(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *RoundTimingInfoLib, error) {
	parsed, err := RoundTimingInfoLibMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(RoundTimingInfoLibBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &RoundTimingInfoLib{RoundTimingInfoLibCaller: RoundTimingInfoLibCaller{contract: contract}, RoundTimingInfoLibTransactor: RoundTimingInfoLibTransactor{contract: contract}, RoundTimingInfoLibFilterer: RoundTimingInfoLibFilterer{contract: contract}}, nil
}

// RoundTimingInfoLib is an auto generated Go binding around an Ethereum contract.
type RoundTimingInfoLib struct {
	RoundTimingInfoLibCaller     // Read-only binding to the contract
	RoundTimingInfoLibTransactor // Write-only binding to the contract
	RoundTimingInfoLibFilterer   // Log filterer for contract events
}

// RoundTimingInfoLibCaller is an auto generated read-only Go binding around an Ethereum contract.
type RoundTimingInfoLibCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoundTimingInfoLibTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RoundTimingInfoLibTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoundTimingInfoLibFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RoundTimingInfoLibFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoundTimingInfoLibSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RoundTimingInfoLibSession struct {
	Contract     *RoundTimingInfoLib // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// RoundTimingInfoLibCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RoundTimingInfoLibCallerSession struct {
	Contract *RoundTimingInfoLibCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// RoundTimingInfoLibTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RoundTimingInfoLibTransactorSession struct {
	Contract     *RoundTimingInfoLibTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// RoundTimingInfoLibRaw is an auto generated low-level Go binding around an Ethereum contract.
type RoundTimingInfoLibRaw struct {
	Contract *RoundTimingInfoLib // Generic contract binding to access the raw methods on
}

// RoundTimingInfoLibCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RoundTimingInfoLibCallerRaw struct {
	Contract *RoundTimingInfoLibCaller // Generic read-only contract binding to access the raw methods on
}

// RoundTimingInfoLibTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RoundTimingInfoLibTransactorRaw struct {
	Contract *RoundTimingInfoLibTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRoundTimingInfoLib creates a new instance of RoundTimingInfoLib, bound to a specific deployed contract.
func NewRoundTimingInfoLib(address common.Address, backend bind.ContractBackend) (*RoundTimingInfoLib, error) {
	contract, err := bindRoundTimingInfoLib(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RoundTimingInfoLib{RoundTimingInfoLibCaller: RoundTimingInfoLibCaller{contract: contract}, RoundTimingInfoLibTransactor: RoundTimingInfoLibTransactor{contract: contract}, RoundTimingInfoLibFilterer: RoundTimingInfoLibFilterer{contract: contract}}, nil
}

// NewRoundTimingInfoLibCaller creates a new read-only instance of RoundTimingInfoLib, bound to a specific deployed contract.
func NewRoundTimingInfoLibCaller(address common.Address, caller bind.ContractCaller) (*RoundTimingInfoLibCaller, error) {
	contract, err := bindRoundTimingInfoLib(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RoundTimingInfoLibCaller{contract: contract}, nil
}

// NewRoundTimingInfoLibTransactor creates a new write-only instance of RoundTimingInfoLib, bound to a specific deployed contract.
func NewRoundTimingInfoLibTransactor(address common.Address, transactor bind.ContractTransactor) (*RoundTimingInfoLibTransactor, error) {
	contract, err := bindRoundTimingInfoLib(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RoundTimingInfoLibTransactor{contract: contract}, nil
}

// NewRoundTimingInfoLibFilterer creates a new log filterer instance of RoundTimingInfoLib, bound to a specific deployed contract.
func NewRoundTimingInfoLibFilterer(address common.Address, filterer bind.ContractFilterer) (*RoundTimingInfoLibFilterer, error) {
	contract, err := bindRoundTimingInfoLib(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RoundTimingInfoLibFilterer{contract: contract}, nil
}

// bindRoundTimingInfoLib binds a generic wrapper to an already deployed contract.
func bindRoundTimingInfoLib(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RoundTimingInfoLibMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RoundTimingInfoLib *RoundTimingInfoLibRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RoundTimingInfoLib.Contract.RoundTimingInfoLibCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RoundTimingInfoLib *RoundTimingInfoLibRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoundTimingInfoLib.Contract.RoundTimingInfoLibTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RoundTimingInfoLib *RoundTimingInfoLibRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RoundTimingInfoLib.Contract.RoundTimingInfoLibTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RoundTimingInfoLib *RoundTimingInfoLibCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RoundTimingInfoLib.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RoundTimingInfoLib *RoundTimingInfoLibTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoundTimingInfoLib.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RoundTimingInfoLib *RoundTimingInfoLibTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RoundTimingInfoLib.Contract.contract.Transact(opts, method, params...)
}
