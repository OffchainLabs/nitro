// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

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
	Bidder    common.Address
	ChainId   *big.Int
	Round     *big.Int
	Amount    *big.Int
	Signature []byte
}

// ExpressLaneAuctionMetaData contains all meta data concerning the ExpressLaneAuction contract.
var ExpressLaneAuctionMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_chainOwnerAddr\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_reservePriceSetterAddr\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_bidReceiverAddr\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_roundLengthSeconds\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"_initialTimestamp\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_stakeToken\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_currentReservePrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_minimalReservePrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"bidReceiver\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"bidSignatureDomainValue\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"bidderBalance\",\"inputs\":[{\"name\":\"bidder\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"cancelUpcomingRound\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"chainOwnerAddress\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"currentExpressLaneController\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"currentRound\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"delegateExpressLane\",\"inputs\":[{\"name\":\"delegate\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"depositBalance\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"expressLaneControllerByRound\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"finalizeWithdrawal\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"getCurrentReservePrice\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getminimalReservePrice\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialRoundTimestamp\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initiateWithdrawal\",\"inputs\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"pendingWithdrawalByBidder\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"submittedRound\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"reservePriceSetterAddress\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"resolveAuction\",\"inputs\":[{\"name\":\"bid1\",\"type\":\"tuple\",\"internalType\":\"structBid\",\"components\":[{\"name\":\"bidder\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"round\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signature\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"bid2\",\"type\":\"tuple\",\"internalType\":\"structBid\",\"components\":[{\"name\":\"bidder\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"round\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signature\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resolveSingleBidAuction\",\"inputs\":[{\"name\":\"bid\",\"type\":\"tuple\",\"internalType\":\"structBid\",\"components\":[{\"name\":\"bidder\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"round\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signature\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"roundDurationSeconds\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setCurrentReservePrice\",\"inputs\":[{\"name\":\"_currentReservePrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setMinimalReservePrice\",\"inputs\":[{\"name\":\"_minimalReservePrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setReservePriceAddresses\",\"inputs\":[{\"name\":\"_reservePriceSetterAddr\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"submitDeposit\",\"inputs\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"verifySignature\",\"inputs\":[{\"name\":\"signer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"message\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"signature\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"pure\"},{\"type\":\"event\",\"name\":\"AuctionResolved\",\"inputs\":[{\"name\":\"winningBidAmount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"secondPlaceBidAmount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"winningBidder\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"winnerRound\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DepositSubmitted\",\"inputs\":[{\"name\":\"bidder\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ExpressLaneControlDelegated\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"round\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"WithdrawalFinalized\",\"inputs\":[{\"name\":\"bidder\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"WithdrawalInitiated\",\"inputs\":[{\"name\":\"bidder\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"IncorrectBidAmount\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InsufficientBalance\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"LessThanCurrentReservePrice\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"LessThanMinReservePrice\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotChainOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotExpressLaneController\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotReservePriceSetter\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ZeroAmount\",\"inputs\":[]}]",
	Bin: "0x60806040526006805461ffff1916600f1790553480156200001f57600080fd5b5060405162001a8838038062001a888339810160408190526200004291620000ef565b600080546001600160a01b03998a166001600160a01b03199182161790915560018054988a169890911697909717909655600280546001600160401b03909516600160a01b026001600160e01b031990951695881695909517939093179093556003556004556005919091556006805491909216620100000262010000600160b01b03199091161790556200018a565b80516001600160a01b0381168114620000ea57600080fd5b919050565b600080600080600080600080610100898b0312156200010d57600080fd5b6200011889620000d2565b97506200012860208a01620000d2565b96506200013860408a01620000d2565b60608a01519096506001600160401b03811681146200015657600080fd5b60808a015190955093506200016e60a08a01620000d2565b60c08a015160e0909a0151989b979a5095989497939692505050565b6118ee806200019a6000396000f3fe608060405234801561001057600080fd5b50600436106101735760003560e01c80638296df03116100de578063cc963d1511610097578063d6e5fb7d11610071578063d6e5fb7d1461037c578063dbeb20121461038f578063f5f754d6146103a2578063f66fda64146103b557600080fd5b8063cc963d151461033e578063cd4abf7114610356578063d6ded1bc1461036957600080fd5b80638296df03146102bd5780638a19c8bc146102e6578063956501bb14610306578063b941ce6e14610326578063c03899791461032e578063c5b6aa2f1461033657600080fd5b80634d1846dc116101305780634d1846dc1461022b5780634f2a9bdb14610233578063574a9b5f1461023b5780635f70f9031461024e57806379a47e291461029b5780637c62b5cd146102ac57600080fd5b806303ba666214610178578063048fae731461018f57806312edde5e146101b857806324e359e7146101cd57806338265efd146101f05780634bc37ea614610206575b600080fd5b6005545b6040519081526020015b60405180910390f35b61017c61019d3660046115a4565b6001600160a01b031660009081526007602052604090205490565b6101cb6101c63660046115c6565b6103c8565b005b6101e06101db366004611681565b61055d565b6040519015158152602001610186565b60065460405161ffff9091168152602001610186565b6002546001600160a01b03165b6040516001600160a01b039091168152602001610186565b6101cb6105e9565b61021361066b565b6101cb6102493660046115c6565b6106a1565b61027e61025c3660046115a4565b600860205260009081526040902080546001909101546001600160401b031682565b604080519283526001600160401b03909116602083015201610186565b6000546001600160a01b0316610213565b6001546001600160a01b0316610213565b6102136102cb3660046115c6565b6009602052600090815260409020546001600160a01b031681565b6102ee6106f4565b6040516001600160401b039091168152602001610186565b61017c6103143660046115a4565b60076020526000908152604090205481565b60045461017c565b60035461017c565b6101cb61073d565b600254600160a01b90046001600160401b03166102ee565b6101cb61036436600461170c565b6108ef565b6101cb6103773660046115c6565b610f31565b6101cb61038a3660046115a4565b610f61565b6101cb61039d3660046115c6565b611024565b6101cb6103b036600461176f565b611122565b6101cb6103c33660046115a4565b611432565b806000036103e957604051631f2a200560e01b815260040160405180910390fd5b3360009081526007602052604090205481111561041957604051631e9acf1760e31b815260040160405180910390fd5b33600090815260086020908152604091829020825180840190935280548084526001909101546001600160401b031691830191909152156104a15760405162461bcd60e51b815260206004820152601c60248201527f7769746864726177616c20616c726561647920696e697469617465640000000060448201526064015b60405180910390fd5b33600090815260076020526040812080548492906104c09084906117c1565b9250508190555060405180604001604052808381526020016104e06106f4565b6001600160401b039081169091523360008181526008602090815260409182902085518155948101516001909501805467ffffffffffffffff19169590941694909417909255905184815290917f6d92f7d3303f995bf21956bb0c51b388bae348eaf45c23debd2cfa3fcd9ec64691015b60405180910390a25050565b8151602083012060009060006105c0826040517f19457468657265756d205369676e6564204d6573736167653a0a3332000000006020820152603c8101829052600090605c01604051602081830303815290604052805190602001209050919050565b905060006105ce828661147f565b6001600160a01b039081169088161493505050509392505050565b60006105f36106f4565b6105fe9060016117d4565b6001600160401b0381166000908152600960205260409020549091506001600160a01b0316338114610643576040516302e001e360e01b815260040160405180910390fd5b506001600160401b0316600090815260096020526040902080546001600160a01b0319169055565b6000600960006106796106f4565b6001600160401b031681526020810191909152604001600020546001600160a01b0316919050565b6001546001600160a01b031633146106cc576040516305fbc41160e01b815260040160405180910390fd5b6005548110156106ef57604051632e99443560e21b815260040160405180910390fd5b600455565b600042600354111561070c57506001600160401b0390565b600254600354600160a01b9091046001600160401b03169061072e90426117c1565b61073891906117fb565b905090565b336000908152600860209081526040808320815180830190925280548083526001909101546001600160401b03169282019290925291036107c05760405162461bcd60e51b815260206004820152601760248201527f6e6f207769746864726177616c20696e697469617465640000000000000000006044820152606401610498565b60006107ca6106f4565b9050816020015160026107dd91906117d4565b6001600160401b0316816001600160401b03161461083d5760405162461bcd60e51b815260206004820152601b60248201527f7769746864726177616c206973206e6f742066696e616c697a656400000000006044820152606401610498565b600654825160405163a9059cbb60e01b81523360048201526024810191909152620100009091046001600160a01b03169063a9059cbb906044016020604051808303816000875af1158015610896573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906108ba919061181d565b50815160405190815233907f9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda890602001610551565b806020013582602001351461093f5760405162461bcd60e51b81526020600482015260166024820152750c6d0c2d2dc40d2c8e640c8de40dcdee840dac2e8c6d60531b6044820152606401610498565b806040013582604001351461098c5760405162461bcd60e51b81526020600482015260136024820152720e4deeadcc8e640c8de40dcdee840dac2e8c6d606b1b6044820152606401610498565b60006109966106f4565b6109a19060016117d4565b6001600160401b03169050808360400135146109f45760405162461bcd60e51b81526020600482015260126024820152711b9bdd081d5c18dbdb5a5b99c81c9bdd5b9960721b6044820152606401610498565b606083013560076000610a0a60208701876115a4565b6001600160a01b03166001600160a01b03168152602001908152602001600020541015610a4a5760405163017e521960e71b815260040160405180910390fd5b606082013560076000610a6060208601866115a4565b6001600160a01b03166001600160a01b03168152602001908152602001600020541015610aa05760405163017e521960e71b815260040160405180910390fd5b610b44610ab060208501856115a4565b6006546040805160f09290921b6001600160f01b031916602080840191909152870135602283015286013560428201526060860135606282015260820160408051601f19818403018152919052610b0a608087018761183f565b8080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061055d92505050565b610b905760405162461bcd60e51b815260206004820152601f60248201527f696e76616c6964207369676e617475726520666f7220666972737420626964006044820152606401610498565b610bfa610ba060208401846115a4565b6006546040805160f09290921b6001600160f01b031916602080840191909152860135602283015285013560428201526060850135606282015260820160408051601f19818403018152919052610b0a608086018661183f565b610c465760405162461bcd60e51b815260206004820181905260248201527f696e76616c6964207369676e617475726520666f72207365636f6e64206269646044820152606401610498565b816060013583606001351115610dca57606082013560076000610c6c60208701876115a4565b6001600160a01b03166001600160a01b031681526020019081526020016000206000828254610c9b91906117c1565b909155505060065460025460405163a9059cbb60e01b81526001600160a01b0391821660048201526060850135602482015262010000909204169063a9059cbb906044016020604051808303816000875af1158015610cfe573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610d22919061181d565b50610d3060208401846115a4565b60408481013560009081526009602090815291902080546001600160a01b0319166001600160a01b0393909316929092179091558190610d72908501856115a4565b6001600160a01b03167febab47201515f7ff99c665889a24e3ea116be175b1504243f6711d4734655ef085606001358560600135604051610dbd929190918252602082015260400190565b60405180910390a3505050565b606083013560076000610de060208601866115a4565b6001600160a01b03166001600160a01b031681526020019081526020016000206000828254610e0f91906117c1565b909155505060065460025460405163a9059cbb60e01b81526001600160a01b0391821660048201526060860135602482015262010000909204169063a9059cbb906044016020604051808303816000875af1158015610e72573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610e96919061181d565b50610ea460208301836115a4565b60408381013560009081526009602090815291902080546001600160a01b0319166001600160a01b0393909316929092179091558190610ee6908401846115a4565b6001600160a01b03167febab47201515f7ff99c665889a24e3ea116be175b1504243f6711d4734655ef084606001358660600135604051610dbd929190918252602082015260400190565b6000546001600160a01b03163314610f5c576040516311c29acf60e31b815260040160405180910390fd5b600555565b6000610f6b6106f4565b610f769060016117d4565b6001600160401b0381166000908152600960205260409020549091506001600160a01b0316338114610fbb576040516302e001e360e01b815260040160405180910390fd5b6001600160401b03821660008181526009602090815260409182902080546001600160a01b0319166001600160a01b0388169081179091559151928352909133917fdf423ef3c0bf417d64c30754b79583ec212ba0b1bd0f6f9cc2a7819c0844bede9101610dbd565b8060000361104557604051631f2a200560e01b815260040160405180910390fd5b336000908152600760205260408120805483929061106490849061188c565b90915550506006546040516323b872dd60e01b815233600482015230602482015260448101839052620100009091046001600160a01b0316906323b872dd906064016020604051808303816000875af11580156110c5573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906110e9919061181d565b5060405181815233907feafda908ad84599c76a83ab100b99811f430e25afb46e42febfe5552aeafa7059060200160405180910390a250565b600061112c6106f4565b6111379060016117d4565b9050806001600160401b031682604001351461118a5760405162461bcd60e51b81526020600482015260126024820152711b9bdd081d5c18dbdb5a5b99c81c9bdd5b9960721b6044820152606401610498565b6060820135600760006111a060208601866115a4565b6001600160a01b03166001600160a01b031681526020019081526020016000205410156111e05760405163017e521960e71b815260040160405180910390fd5b600454826060013510156112075760405163e709032960e01b815260040160405180910390fd5b60608201356007600061121d60208601866115a4565b6001600160a01b03166001600160a01b0316815260200190815260200160002054101561125d5760405163017e521960e71b815260040160405180910390fd5b61126d610ba060208401846115a4565b6112b95760405162461bcd60e51b815260206004820152601f60248201527f696e76616c6964207369676e617475726520666f7220666972737420626964006044820152606401610498565b6060820135600760006112cf60208601866115a4565b6001600160a01b03166001600160a01b0316815260200190815260200160002060008282546112fe91906117c1565b909155505060065460025460405163a9059cbb60e01b81526001600160a01b0391821660048201526060850135602482015262010000909204169063a9059cbb906044016020604051808303816000875af1158015611361573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611385919061181d565b5061139360208301836115a4565b60408381013560009081526009602090815291902080546001600160a01b0319166001600160a01b0393909316929092179091556001600160401b038216906113de908401846115a4565b6001600160a01b03167febab47201515f7ff99c665889a24e3ea116be175b1504243f6711d4734655ef084606001356000604051611426929190918252602082015260400190565b60405180910390a35050565b6000546001600160a01b0316331461145d576040516311c29acf60e31b815260040160405180910390fd5b600180546001600160a01b0319166001600160a01b0392909216919091179055565b60008060008061148e856114ff565b6040805160008152602081018083528b905260ff8316918101919091526060810184905260808101839052929550909350915060019060a0016020604051602081039080840390855afa1580156114e9573d6000803e3d6000fd5b5050506020604051035193505050505b92915050565b600080600083516041146115555760405162461bcd60e51b815260206004820152601860248201527f696e76616c6964207369676e6174757265206c656e67746800000000000000006044820152606401610498565b50505060208101516040820151606083015160001a601b8110156115815761157e601b8261189f565b90505b9193909250565b80356001600160a01b038116811461159f57600080fd5b919050565b6000602082840312156115b657600080fd5b6115bf82611588565b9392505050565b6000602082840312156115d857600080fd5b5035919050565b634e487b7160e01b600052604160045260246000fd5b600082601f83011261160657600080fd5b81356001600160401b0380821115611620576116206115df565b604051601f8301601f19908116603f01168101908282118183101715611648576116486115df565b8160405283815286602085880101111561166157600080fd5b836020870160208301376000602085830101528094505050505092915050565b60008060006060848603121561169657600080fd5b61169f84611588565b925060208401356001600160401b03808211156116bb57600080fd5b6116c7878388016115f5565b935060408601359150808211156116dd57600080fd5b506116ea868287016115f5565b9150509250925092565b600060a0828403121561170657600080fd5b50919050565b6000806040838503121561171f57600080fd5b82356001600160401b038082111561173657600080fd5b611742868387016116f4565b9350602085013591508082111561175857600080fd5b50611765858286016116f4565b9150509250929050565b60006020828403121561178157600080fd5b81356001600160401b0381111561179757600080fd5b6117a3848285016116f4565b949350505050565b634e487b7160e01b600052601160045260246000fd5b818103818111156114f9576114f96117ab565b6001600160401b038181168382160190808211156117f4576117f46117ab565b5092915050565b60008261181857634e487b7160e01b600052601260045260246000fd5b500490565b60006020828403121561182f57600080fd5b815180151581146115bf57600080fd5b6000808335601e1984360301811261185657600080fd5b8301803591506001600160401b0382111561187057600080fd5b60200191503681900382131561188557600080fd5b9250929050565b808201808211156114f9576114f96117ab565b60ff81811683821601908111156114f9576114f96117ab56fea26469706673582212206429478454ed8215ff5008fd2094b9c08b2ac458e30cc85f0f5be4106743765f64736f6c63430008130033",
}

// ExpressLaneAuctionABI is the input ABI used to generate the binding from.
// Deprecated: Use ExpressLaneAuctionMetaData.ABI instead.
var ExpressLaneAuctionABI = ExpressLaneAuctionMetaData.ABI

// ExpressLaneAuctionBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ExpressLaneAuctionMetaData.Bin instead.
var ExpressLaneAuctionBin = ExpressLaneAuctionMetaData.Bin

// DeployExpressLaneAuction deploys a new Ethereum contract, binding an instance of ExpressLaneAuction to it.
func DeployExpressLaneAuction(auth *bind.TransactOpts, backend bind.ContractBackend, _chainOwnerAddr common.Address, _reservePriceSetterAddr common.Address, _bidReceiverAddr common.Address, _roundLengthSeconds uint64, _initialTimestamp *big.Int, _stakeToken common.Address, _currentReservePrice *big.Int, _minimalReservePrice *big.Int) (common.Address, *types.Transaction, *ExpressLaneAuction, error) {
	parsed, err := ExpressLaneAuctionMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ExpressLaneAuctionBin), backend, _chainOwnerAddr, _reservePriceSetterAddr, _bidReceiverAddr, _roundLengthSeconds, _initialTimestamp, _stakeToken, _currentReservePrice, _minimalReservePrice)
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

// BidReceiver is a free data retrieval call binding the contract method 0x4bc37ea6.
//
// Solidity: function bidReceiver() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) BidReceiver(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "bidReceiver")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BidReceiver is a free data retrieval call binding the contract method 0x4bc37ea6.
//
// Solidity: function bidReceiver() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) BidReceiver() (common.Address, error) {
	return _ExpressLaneAuction.Contract.BidReceiver(&_ExpressLaneAuction.CallOpts)
}

// BidReceiver is a free data retrieval call binding the contract method 0x4bc37ea6.
//
// Solidity: function bidReceiver() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) BidReceiver() (common.Address, error) {
	return _ExpressLaneAuction.Contract.BidReceiver(&_ExpressLaneAuction.CallOpts)
}

// BidSignatureDomainValue is a free data retrieval call binding the contract method 0x38265efd.
//
// Solidity: function bidSignatureDomainValue() view returns(uint16)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) BidSignatureDomainValue(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "bidSignatureDomainValue")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// BidSignatureDomainValue is a free data retrieval call binding the contract method 0x38265efd.
//
// Solidity: function bidSignatureDomainValue() view returns(uint16)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) BidSignatureDomainValue() (uint16, error) {
	return _ExpressLaneAuction.Contract.BidSignatureDomainValue(&_ExpressLaneAuction.CallOpts)
}

// BidSignatureDomainValue is a free data retrieval call binding the contract method 0x38265efd.
//
// Solidity: function bidSignatureDomainValue() view returns(uint16)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) BidSignatureDomainValue() (uint16, error) {
	return _ExpressLaneAuction.Contract.BidSignatureDomainValue(&_ExpressLaneAuction.CallOpts)
}

// BidderBalance is a free data retrieval call binding the contract method 0x048fae73.
//
// Solidity: function bidderBalance(address bidder) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) BidderBalance(opts *bind.CallOpts, bidder common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "bidderBalance", bidder)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BidderBalance is a free data retrieval call binding the contract method 0x048fae73.
//
// Solidity: function bidderBalance(address bidder) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) BidderBalance(bidder common.Address) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.BidderBalance(&_ExpressLaneAuction.CallOpts, bidder)
}

// BidderBalance is a free data retrieval call binding the contract method 0x048fae73.
//
// Solidity: function bidderBalance(address bidder) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) BidderBalance(bidder common.Address) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.BidderBalance(&_ExpressLaneAuction.CallOpts, bidder)
}

// ChainOwnerAddress is a free data retrieval call binding the contract method 0x79a47e29.
//
// Solidity: function chainOwnerAddress() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) ChainOwnerAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "chainOwnerAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ChainOwnerAddress is a free data retrieval call binding the contract method 0x79a47e29.
//
// Solidity: function chainOwnerAddress() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) ChainOwnerAddress() (common.Address, error) {
	return _ExpressLaneAuction.Contract.ChainOwnerAddress(&_ExpressLaneAuction.CallOpts)
}

// ChainOwnerAddress is a free data retrieval call binding the contract method 0x79a47e29.
//
// Solidity: function chainOwnerAddress() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) ChainOwnerAddress() (common.Address, error) {
	return _ExpressLaneAuction.Contract.ChainOwnerAddress(&_ExpressLaneAuction.CallOpts)
}

// CurrentExpressLaneController is a free data retrieval call binding the contract method 0x4f2a9bdb.
//
// Solidity: function currentExpressLaneController() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) CurrentExpressLaneController(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "currentExpressLaneController")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CurrentExpressLaneController is a free data retrieval call binding the contract method 0x4f2a9bdb.
//
// Solidity: function currentExpressLaneController() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) CurrentExpressLaneController() (common.Address, error) {
	return _ExpressLaneAuction.Contract.CurrentExpressLaneController(&_ExpressLaneAuction.CallOpts)
}

// CurrentExpressLaneController is a free data retrieval call binding the contract method 0x4f2a9bdb.
//
// Solidity: function currentExpressLaneController() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) CurrentExpressLaneController() (common.Address, error) {
	return _ExpressLaneAuction.Contract.CurrentExpressLaneController(&_ExpressLaneAuction.CallOpts)
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

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) DepositBalance(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "depositBalance", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) DepositBalance(arg0 common.Address) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.DepositBalance(&_ExpressLaneAuction.CallOpts, arg0)
}

// DepositBalance is a free data retrieval call binding the contract method 0x956501bb.
//
// Solidity: function depositBalance(address ) view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) DepositBalance(arg0 common.Address) (*big.Int, error) {
	return _ExpressLaneAuction.Contract.DepositBalance(&_ExpressLaneAuction.CallOpts, arg0)
}

// ExpressLaneControllerByRound is a free data retrieval call binding the contract method 0x8296df03.
//
// Solidity: function expressLaneControllerByRound(uint256 ) view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) ExpressLaneControllerByRound(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "expressLaneControllerByRound", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ExpressLaneControllerByRound is a free data retrieval call binding the contract method 0x8296df03.
//
// Solidity: function expressLaneControllerByRound(uint256 ) view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) ExpressLaneControllerByRound(arg0 *big.Int) (common.Address, error) {
	return _ExpressLaneAuction.Contract.ExpressLaneControllerByRound(&_ExpressLaneAuction.CallOpts, arg0)
}

// ExpressLaneControllerByRound is a free data retrieval call binding the contract method 0x8296df03.
//
// Solidity: function expressLaneControllerByRound(uint256 ) view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) ExpressLaneControllerByRound(arg0 *big.Int) (common.Address, error) {
	return _ExpressLaneAuction.Contract.ExpressLaneControllerByRound(&_ExpressLaneAuction.CallOpts, arg0)
}

// GetCurrentReservePrice is a free data retrieval call binding the contract method 0xb941ce6e.
//
// Solidity: function getCurrentReservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) GetCurrentReservePrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "getCurrentReservePrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentReservePrice is a free data retrieval call binding the contract method 0xb941ce6e.
//
// Solidity: function getCurrentReservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) GetCurrentReservePrice() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.GetCurrentReservePrice(&_ExpressLaneAuction.CallOpts)
}

// GetCurrentReservePrice is a free data retrieval call binding the contract method 0xb941ce6e.
//
// Solidity: function getCurrentReservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) GetCurrentReservePrice() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.GetCurrentReservePrice(&_ExpressLaneAuction.CallOpts)
}

// GetminimalReservePrice is a free data retrieval call binding the contract method 0x03ba6662.
//
// Solidity: function getminimalReservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) GetminimalReservePrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "getminimalReservePrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetminimalReservePrice is a free data retrieval call binding the contract method 0x03ba6662.
//
// Solidity: function getminimalReservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) GetminimalReservePrice() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.GetminimalReservePrice(&_ExpressLaneAuction.CallOpts)
}

// GetminimalReservePrice is a free data retrieval call binding the contract method 0x03ba6662.
//
// Solidity: function getminimalReservePrice() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) GetminimalReservePrice() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.GetminimalReservePrice(&_ExpressLaneAuction.CallOpts)
}

// InitialRoundTimestamp is a free data retrieval call binding the contract method 0xc0389979.
//
// Solidity: function initialRoundTimestamp() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) InitialRoundTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "initialRoundTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// InitialRoundTimestamp is a free data retrieval call binding the contract method 0xc0389979.
//
// Solidity: function initialRoundTimestamp() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) InitialRoundTimestamp() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.InitialRoundTimestamp(&_ExpressLaneAuction.CallOpts)
}

// InitialRoundTimestamp is a free data retrieval call binding the contract method 0xc0389979.
//
// Solidity: function initialRoundTimestamp() view returns(uint256)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) InitialRoundTimestamp() (*big.Int, error) {
	return _ExpressLaneAuction.Contract.InitialRoundTimestamp(&_ExpressLaneAuction.CallOpts)
}

// PendingWithdrawalByBidder is a free data retrieval call binding the contract method 0x5f70f903.
//
// Solidity: function pendingWithdrawalByBidder(address ) view returns(uint256 amount, uint64 submittedRound)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) PendingWithdrawalByBidder(opts *bind.CallOpts, arg0 common.Address) (struct {
	Amount         *big.Int
	SubmittedRound uint64
}, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "pendingWithdrawalByBidder", arg0)

	outstruct := new(struct {
		Amount         *big.Int
		SubmittedRound uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Amount = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.SubmittedRound = *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return *outstruct, err

}

// PendingWithdrawalByBidder is a free data retrieval call binding the contract method 0x5f70f903.
//
// Solidity: function pendingWithdrawalByBidder(address ) view returns(uint256 amount, uint64 submittedRound)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) PendingWithdrawalByBidder(arg0 common.Address) (struct {
	Amount         *big.Int
	SubmittedRound uint64
}, error) {
	return _ExpressLaneAuction.Contract.PendingWithdrawalByBidder(&_ExpressLaneAuction.CallOpts, arg0)
}

// PendingWithdrawalByBidder is a free data retrieval call binding the contract method 0x5f70f903.
//
// Solidity: function pendingWithdrawalByBidder(address ) view returns(uint256 amount, uint64 submittedRound)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) PendingWithdrawalByBidder(arg0 common.Address) (struct {
	Amount         *big.Int
	SubmittedRound uint64
}, error) {
	return _ExpressLaneAuction.Contract.PendingWithdrawalByBidder(&_ExpressLaneAuction.CallOpts, arg0)
}

// ReservePriceSetterAddress is a free data retrieval call binding the contract method 0x7c62b5cd.
//
// Solidity: function reservePriceSetterAddress() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) ReservePriceSetterAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "reservePriceSetterAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ReservePriceSetterAddress is a free data retrieval call binding the contract method 0x7c62b5cd.
//
// Solidity: function reservePriceSetterAddress() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) ReservePriceSetterAddress() (common.Address, error) {
	return _ExpressLaneAuction.Contract.ReservePriceSetterAddress(&_ExpressLaneAuction.CallOpts)
}

// ReservePriceSetterAddress is a free data retrieval call binding the contract method 0x7c62b5cd.
//
// Solidity: function reservePriceSetterAddress() view returns(address)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) ReservePriceSetterAddress() (common.Address, error) {
	return _ExpressLaneAuction.Contract.ReservePriceSetterAddress(&_ExpressLaneAuction.CallOpts)
}

// RoundDurationSeconds is a free data retrieval call binding the contract method 0xcc963d15.
//
// Solidity: function roundDurationSeconds() view returns(uint64)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) RoundDurationSeconds(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "roundDurationSeconds")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RoundDurationSeconds is a free data retrieval call binding the contract method 0xcc963d15.
//
// Solidity: function roundDurationSeconds() view returns(uint64)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) RoundDurationSeconds() (uint64, error) {
	return _ExpressLaneAuction.Contract.RoundDurationSeconds(&_ExpressLaneAuction.CallOpts)
}

// RoundDurationSeconds is a free data retrieval call binding the contract method 0xcc963d15.
//
// Solidity: function roundDurationSeconds() view returns(uint64)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) RoundDurationSeconds() (uint64, error) {
	return _ExpressLaneAuction.Contract.RoundDurationSeconds(&_ExpressLaneAuction.CallOpts)
}

// VerifySignature is a free data retrieval call binding the contract method 0x24e359e7.
//
// Solidity: function verifySignature(address signer, bytes message, bytes signature) pure returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionCaller) VerifySignature(opts *bind.CallOpts, signer common.Address, message []byte, signature []byte) (bool, error) {
	var out []interface{}
	err := _ExpressLaneAuction.contract.Call(opts, &out, "verifySignature", signer, message, signature)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// VerifySignature is a free data retrieval call binding the contract method 0x24e359e7.
//
// Solidity: function verifySignature(address signer, bytes message, bytes signature) pure returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionSession) VerifySignature(signer common.Address, message []byte, signature []byte) (bool, error) {
	return _ExpressLaneAuction.Contract.VerifySignature(&_ExpressLaneAuction.CallOpts, signer, message, signature)
}

// VerifySignature is a free data retrieval call binding the contract method 0x24e359e7.
//
// Solidity: function verifySignature(address signer, bytes message, bytes signature) pure returns(bool)
func (_ExpressLaneAuction *ExpressLaneAuctionCallerSession) VerifySignature(signer common.Address, message []byte, signature []byte) (bool, error) {
	return _ExpressLaneAuction.Contract.VerifySignature(&_ExpressLaneAuction.CallOpts, signer, message, signature)
}

// CancelUpcomingRound is a paid mutator transaction binding the contract method 0x4d1846dc.
//
// Solidity: function cancelUpcomingRound() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) CancelUpcomingRound(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "cancelUpcomingRound")
}

// CancelUpcomingRound is a paid mutator transaction binding the contract method 0x4d1846dc.
//
// Solidity: function cancelUpcomingRound() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) CancelUpcomingRound() (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.CancelUpcomingRound(&_ExpressLaneAuction.TransactOpts)
}

// CancelUpcomingRound is a paid mutator transaction binding the contract method 0x4d1846dc.
//
// Solidity: function cancelUpcomingRound() returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) CancelUpcomingRound() (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.CancelUpcomingRound(&_ExpressLaneAuction.TransactOpts)
}

// DelegateExpressLane is a paid mutator transaction binding the contract method 0xd6e5fb7d.
//
// Solidity: function delegateExpressLane(address delegate) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) DelegateExpressLane(opts *bind.TransactOpts, delegate common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "delegateExpressLane", delegate)
}

// DelegateExpressLane is a paid mutator transaction binding the contract method 0xd6e5fb7d.
//
// Solidity: function delegateExpressLane(address delegate) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) DelegateExpressLane(delegate common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.DelegateExpressLane(&_ExpressLaneAuction.TransactOpts, delegate)
}

// DelegateExpressLane is a paid mutator transaction binding the contract method 0xd6e5fb7d.
//
// Solidity: function delegateExpressLane(address delegate) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) DelegateExpressLane(delegate common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.DelegateExpressLane(&_ExpressLaneAuction.TransactOpts, delegate)
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

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0x12edde5e.
//
// Solidity: function initiateWithdrawal(uint256 amount) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) InitiateWithdrawal(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "initiateWithdrawal", amount)
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0x12edde5e.
//
// Solidity: function initiateWithdrawal(uint256 amount) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) InitiateWithdrawal(amount *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.InitiateWithdrawal(&_ExpressLaneAuction.TransactOpts, amount)
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0x12edde5e.
//
// Solidity: function initiateWithdrawal(uint256 amount) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) InitiateWithdrawal(amount *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.InitiateWithdrawal(&_ExpressLaneAuction.TransactOpts, amount)
}

// ResolveAuction is a paid mutator transaction binding the contract method 0xcd4abf71.
//
// Solidity: function resolveAuction((address,uint256,uint256,uint256,bytes) bid1, (address,uint256,uint256,uint256,bytes) bid2) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) ResolveAuction(opts *bind.TransactOpts, bid1 Bid, bid2 Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "resolveAuction", bid1, bid2)
}

// ResolveAuction is a paid mutator transaction binding the contract method 0xcd4abf71.
//
// Solidity: function resolveAuction((address,uint256,uint256,uint256,bytes) bid1, (address,uint256,uint256,uint256,bytes) bid2) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) ResolveAuction(bid1 Bid, bid2 Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.ResolveAuction(&_ExpressLaneAuction.TransactOpts, bid1, bid2)
}

// ResolveAuction is a paid mutator transaction binding the contract method 0xcd4abf71.
//
// Solidity: function resolveAuction((address,uint256,uint256,uint256,bytes) bid1, (address,uint256,uint256,uint256,bytes) bid2) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) ResolveAuction(bid1 Bid, bid2 Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.ResolveAuction(&_ExpressLaneAuction.TransactOpts, bid1, bid2)
}

// ResolveSingleBidAuction is a paid mutator transaction binding the contract method 0xf5f754d6.
//
// Solidity: function resolveSingleBidAuction((address,uint256,uint256,uint256,bytes) bid) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) ResolveSingleBidAuction(opts *bind.TransactOpts, bid Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "resolveSingleBidAuction", bid)
}

// ResolveSingleBidAuction is a paid mutator transaction binding the contract method 0xf5f754d6.
//
// Solidity: function resolveSingleBidAuction((address,uint256,uint256,uint256,bytes) bid) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) ResolveSingleBidAuction(bid Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.ResolveSingleBidAuction(&_ExpressLaneAuction.TransactOpts, bid)
}

// ResolveSingleBidAuction is a paid mutator transaction binding the contract method 0xf5f754d6.
//
// Solidity: function resolveSingleBidAuction((address,uint256,uint256,uint256,bytes) bid) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) ResolveSingleBidAuction(bid Bid) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.ResolveSingleBidAuction(&_ExpressLaneAuction.TransactOpts, bid)
}

// SetCurrentReservePrice is a paid mutator transaction binding the contract method 0x574a9b5f.
//
// Solidity: function setCurrentReservePrice(uint256 _currentReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) SetCurrentReservePrice(opts *bind.TransactOpts, _currentReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "setCurrentReservePrice", _currentReservePrice)
}

// SetCurrentReservePrice is a paid mutator transaction binding the contract method 0x574a9b5f.
//
// Solidity: function setCurrentReservePrice(uint256 _currentReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) SetCurrentReservePrice(_currentReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetCurrentReservePrice(&_ExpressLaneAuction.TransactOpts, _currentReservePrice)
}

// SetCurrentReservePrice is a paid mutator transaction binding the contract method 0x574a9b5f.
//
// Solidity: function setCurrentReservePrice(uint256 _currentReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) SetCurrentReservePrice(_currentReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetCurrentReservePrice(&_ExpressLaneAuction.TransactOpts, _currentReservePrice)
}

// SetMinimalReservePrice is a paid mutator transaction binding the contract method 0xd6ded1bc.
//
// Solidity: function setMinimalReservePrice(uint256 _minimalReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) SetMinimalReservePrice(opts *bind.TransactOpts, _minimalReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "setMinimalReservePrice", _minimalReservePrice)
}

// SetMinimalReservePrice is a paid mutator transaction binding the contract method 0xd6ded1bc.
//
// Solidity: function setMinimalReservePrice(uint256 _minimalReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) SetMinimalReservePrice(_minimalReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetMinimalReservePrice(&_ExpressLaneAuction.TransactOpts, _minimalReservePrice)
}

// SetMinimalReservePrice is a paid mutator transaction binding the contract method 0xd6ded1bc.
//
// Solidity: function setMinimalReservePrice(uint256 _minimalReservePrice) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) SetMinimalReservePrice(_minimalReservePrice *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetMinimalReservePrice(&_ExpressLaneAuction.TransactOpts, _minimalReservePrice)
}

// SetReservePriceAddresses is a paid mutator transaction binding the contract method 0xf66fda64.
//
// Solidity: function setReservePriceAddresses(address _reservePriceSetterAddr) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) SetReservePriceAddresses(opts *bind.TransactOpts, _reservePriceSetterAddr common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "setReservePriceAddresses", _reservePriceSetterAddr)
}

// SetReservePriceAddresses is a paid mutator transaction binding the contract method 0xf66fda64.
//
// Solidity: function setReservePriceAddresses(address _reservePriceSetterAddr) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) SetReservePriceAddresses(_reservePriceSetterAddr common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetReservePriceAddresses(&_ExpressLaneAuction.TransactOpts, _reservePriceSetterAddr)
}

// SetReservePriceAddresses is a paid mutator transaction binding the contract method 0xf66fda64.
//
// Solidity: function setReservePriceAddresses(address _reservePriceSetterAddr) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) SetReservePriceAddresses(_reservePriceSetterAddr common.Address) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SetReservePriceAddresses(&_ExpressLaneAuction.TransactOpts, _reservePriceSetterAddr)
}

// SubmitDeposit is a paid mutator transaction binding the contract method 0xdbeb2012.
//
// Solidity: function submitDeposit(uint256 amount) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactor) SubmitDeposit(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.contract.Transact(opts, "submitDeposit", amount)
}

// SubmitDeposit is a paid mutator transaction binding the contract method 0xdbeb2012.
//
// Solidity: function submitDeposit(uint256 amount) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionSession) SubmitDeposit(amount *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SubmitDeposit(&_ExpressLaneAuction.TransactOpts, amount)
}

// SubmitDeposit is a paid mutator transaction binding the contract method 0xdbeb2012.
//
// Solidity: function submitDeposit(uint256 amount) returns()
func (_ExpressLaneAuction *ExpressLaneAuctionTransactorSession) SubmitDeposit(amount *big.Int) (*types.Transaction, error) {
	return _ExpressLaneAuction.Contract.SubmitDeposit(&_ExpressLaneAuction.TransactOpts, amount)
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
	WinningBidAmount     *big.Int
	SecondPlaceBidAmount *big.Int
	WinningBidder        common.Address
	WinnerRound          *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterAuctionResolved is a free log retrieval operation binding the contract event 0xebab47201515f7ff99c665889a24e3ea116be175b1504243f6711d4734655ef0.
//
// Solidity: event AuctionResolved(uint256 winningBidAmount, uint256 secondPlaceBidAmount, address indexed winningBidder, uint256 indexed winnerRound)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterAuctionResolved(opts *bind.FilterOpts, winningBidder []common.Address, winnerRound []*big.Int) (*ExpressLaneAuctionAuctionResolvedIterator, error) {

	var winningBidderRule []interface{}
	for _, winningBidderItem := range winningBidder {
		winningBidderRule = append(winningBidderRule, winningBidderItem)
	}
	var winnerRoundRule []interface{}
	for _, winnerRoundItem := range winnerRound {
		winnerRoundRule = append(winnerRoundRule, winnerRoundItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "AuctionResolved", winningBidderRule, winnerRoundRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionAuctionResolvedIterator{contract: _ExpressLaneAuction.contract, event: "AuctionResolved", logs: logs, sub: sub}, nil
}

// WatchAuctionResolved is a free log subscription operation binding the contract event 0xebab47201515f7ff99c665889a24e3ea116be175b1504243f6711d4734655ef0.
//
// Solidity: event AuctionResolved(uint256 winningBidAmount, uint256 secondPlaceBidAmount, address indexed winningBidder, uint256 indexed winnerRound)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchAuctionResolved(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionAuctionResolved, winningBidder []common.Address, winnerRound []*big.Int) (event.Subscription, error) {

	var winningBidderRule []interface{}
	for _, winningBidderItem := range winningBidder {
		winningBidderRule = append(winningBidderRule, winningBidderItem)
	}
	var winnerRoundRule []interface{}
	for _, winnerRoundItem := range winnerRound {
		winnerRoundRule = append(winnerRoundRule, winnerRoundItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "AuctionResolved", winningBidderRule, winnerRoundRule)
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

// ParseAuctionResolved is a log parse operation binding the contract event 0xebab47201515f7ff99c665889a24e3ea116be175b1504243f6711d4734655ef0.
//
// Solidity: event AuctionResolved(uint256 winningBidAmount, uint256 secondPlaceBidAmount, address indexed winningBidder, uint256 indexed winnerRound)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseAuctionResolved(log types.Log) (*ExpressLaneAuctionAuctionResolved, error) {
	event := new(ExpressLaneAuctionAuctionResolved)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "AuctionResolved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionDepositSubmittedIterator is returned from FilterDepositSubmitted and is used to iterate over the raw logs and unpacked data for DepositSubmitted events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionDepositSubmittedIterator struct {
	Event *ExpressLaneAuctionDepositSubmitted // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionDepositSubmittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionDepositSubmitted)
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
		it.Event = new(ExpressLaneAuctionDepositSubmitted)
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
func (it *ExpressLaneAuctionDepositSubmittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionDepositSubmittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionDepositSubmitted represents a DepositSubmitted event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionDepositSubmitted struct {
	Bidder common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterDepositSubmitted is a free log retrieval operation binding the contract event 0xeafda908ad84599c76a83ab100b99811f430e25afb46e42febfe5552aeafa705.
//
// Solidity: event DepositSubmitted(address indexed bidder, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterDepositSubmitted(opts *bind.FilterOpts, bidder []common.Address) (*ExpressLaneAuctionDepositSubmittedIterator, error) {

	var bidderRule []interface{}
	for _, bidderItem := range bidder {
		bidderRule = append(bidderRule, bidderItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "DepositSubmitted", bidderRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionDepositSubmittedIterator{contract: _ExpressLaneAuction.contract, event: "DepositSubmitted", logs: logs, sub: sub}, nil
}

// WatchDepositSubmitted is a free log subscription operation binding the contract event 0xeafda908ad84599c76a83ab100b99811f430e25afb46e42febfe5552aeafa705.
//
// Solidity: event DepositSubmitted(address indexed bidder, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchDepositSubmitted(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionDepositSubmitted, bidder []common.Address) (event.Subscription, error) {

	var bidderRule []interface{}
	for _, bidderItem := range bidder {
		bidderRule = append(bidderRule, bidderItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "DepositSubmitted", bidderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionDepositSubmitted)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "DepositSubmitted", log); err != nil {
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

// ParseDepositSubmitted is a log parse operation binding the contract event 0xeafda908ad84599c76a83ab100b99811f430e25afb46e42febfe5552aeafa705.
//
// Solidity: event DepositSubmitted(address indexed bidder, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseDepositSubmitted(log types.Log) (*ExpressLaneAuctionDepositSubmitted, error) {
	event := new(ExpressLaneAuctionDepositSubmitted)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "DepositSubmitted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExpressLaneAuctionExpressLaneControlDelegatedIterator is returned from FilterExpressLaneControlDelegated and is used to iterate over the raw logs and unpacked data for ExpressLaneControlDelegated events raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionExpressLaneControlDelegatedIterator struct {
	Event *ExpressLaneAuctionExpressLaneControlDelegated // Event containing the contract specifics and raw log

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
func (it *ExpressLaneAuctionExpressLaneControlDelegatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExpressLaneAuctionExpressLaneControlDelegated)
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
		it.Event = new(ExpressLaneAuctionExpressLaneControlDelegated)
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
func (it *ExpressLaneAuctionExpressLaneControlDelegatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExpressLaneAuctionExpressLaneControlDelegatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExpressLaneAuctionExpressLaneControlDelegated represents a ExpressLaneControlDelegated event raised by the ExpressLaneAuction contract.
type ExpressLaneAuctionExpressLaneControlDelegated struct {
	From  common.Address
	To    common.Address
	Round uint64
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterExpressLaneControlDelegated is a free log retrieval operation binding the contract event 0xdf423ef3c0bf417d64c30754b79583ec212ba0b1bd0f6f9cc2a7819c0844bede.
//
// Solidity: event ExpressLaneControlDelegated(address indexed from, address indexed to, uint64 round)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterExpressLaneControlDelegated(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*ExpressLaneAuctionExpressLaneControlDelegatedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "ExpressLaneControlDelegated", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionExpressLaneControlDelegatedIterator{contract: _ExpressLaneAuction.contract, event: "ExpressLaneControlDelegated", logs: logs, sub: sub}, nil
}

// WatchExpressLaneControlDelegated is a free log subscription operation binding the contract event 0xdf423ef3c0bf417d64c30754b79583ec212ba0b1bd0f6f9cc2a7819c0844bede.
//
// Solidity: event ExpressLaneControlDelegated(address indexed from, address indexed to, uint64 round)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchExpressLaneControlDelegated(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionExpressLaneControlDelegated, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "ExpressLaneControlDelegated", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExpressLaneAuctionExpressLaneControlDelegated)
				if err := _ExpressLaneAuction.contract.UnpackLog(event, "ExpressLaneControlDelegated", log); err != nil {
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

// ParseExpressLaneControlDelegated is a log parse operation binding the contract event 0xdf423ef3c0bf417d64c30754b79583ec212ba0b1bd0f6f9cc2a7819c0844bede.
//
// Solidity: event ExpressLaneControlDelegated(address indexed from, address indexed to, uint64 round)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseExpressLaneControlDelegated(log types.Log) (*ExpressLaneAuctionExpressLaneControlDelegated, error) {
	event := new(ExpressLaneAuctionExpressLaneControlDelegated)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "ExpressLaneControlDelegated", log); err != nil {
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
	Bidder common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalFinalized is a free log retrieval operation binding the contract event 0x9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda8.
//
// Solidity: event WithdrawalFinalized(address indexed bidder, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterWithdrawalFinalized(opts *bind.FilterOpts, bidder []common.Address) (*ExpressLaneAuctionWithdrawalFinalizedIterator, error) {

	var bidderRule []interface{}
	for _, bidderItem := range bidder {
		bidderRule = append(bidderRule, bidderItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "WithdrawalFinalized", bidderRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionWithdrawalFinalizedIterator{contract: _ExpressLaneAuction.contract, event: "WithdrawalFinalized", logs: logs, sub: sub}, nil
}

// WatchWithdrawalFinalized is a free log subscription operation binding the contract event 0x9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda8.
//
// Solidity: event WithdrawalFinalized(address indexed bidder, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchWithdrawalFinalized(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionWithdrawalFinalized, bidder []common.Address) (event.Subscription, error) {

	var bidderRule []interface{}
	for _, bidderItem := range bidder {
		bidderRule = append(bidderRule, bidderItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "WithdrawalFinalized", bidderRule)
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
// Solidity: event WithdrawalFinalized(address indexed bidder, uint256 amount)
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
	Bidder common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalInitiated is a free log retrieval operation binding the contract event 0x6d92f7d3303f995bf21956bb0c51b388bae348eaf45c23debd2cfa3fcd9ec646.
//
// Solidity: event WithdrawalInitiated(address indexed bidder, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) FilterWithdrawalInitiated(opts *bind.FilterOpts, bidder []common.Address) (*ExpressLaneAuctionWithdrawalInitiatedIterator, error) {

	var bidderRule []interface{}
	for _, bidderItem := range bidder {
		bidderRule = append(bidderRule, bidderItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.FilterLogs(opts, "WithdrawalInitiated", bidderRule)
	if err != nil {
		return nil, err
	}
	return &ExpressLaneAuctionWithdrawalInitiatedIterator{contract: _ExpressLaneAuction.contract, event: "WithdrawalInitiated", logs: logs, sub: sub}, nil
}

// WatchWithdrawalInitiated is a free log subscription operation binding the contract event 0x6d92f7d3303f995bf21956bb0c51b388bae348eaf45c23debd2cfa3fcd9ec646.
//
// Solidity: event WithdrawalInitiated(address indexed bidder, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) WatchWithdrawalInitiated(opts *bind.WatchOpts, sink chan<- *ExpressLaneAuctionWithdrawalInitiated, bidder []common.Address) (event.Subscription, error) {

	var bidderRule []interface{}
	for _, bidderItem := range bidder {
		bidderRule = append(bidderRule, bidderItem)
	}

	logs, sub, err := _ExpressLaneAuction.contract.WatchLogs(opts, "WithdrawalInitiated", bidderRule)
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

// ParseWithdrawalInitiated is a log parse operation binding the contract event 0x6d92f7d3303f995bf21956bb0c51b388bae348eaf45c23debd2cfa3fcd9ec646.
//
// Solidity: event WithdrawalInitiated(address indexed bidder, uint256 amount)
func (_ExpressLaneAuction *ExpressLaneAuctionFilterer) ParseWithdrawalInitiated(log types.Log) (*ExpressLaneAuctionWithdrawalInitiated, error) {
	event := new(ExpressLaneAuctionWithdrawalInitiated)
	if err := _ExpressLaneAuction.contract.UnpackLog(event, "WithdrawalInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
