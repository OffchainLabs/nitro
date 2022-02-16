package statetransfer

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/solgen/go/classicgen"
)

var ArbosTestAddress = common.HexToAddress("0x0000000000000000000000000000000000000069")

var tempAccountList map[common.Address]struct{} //TODO: this is used while we cannot get a list of accounts from arbos
var arbostestABI abi.ABI

func init() {
	tempAccountList = make(map[common.Address]struct{})

	var err error
	arbostestABI, err = abi.JSON(strings.NewReader(classicgen.ArbosTestABI))
	if err != nil {
		panic(err)
	}
}

func AddressSeen(address common.Address) {
	if address.Hash().Big().BitLen() > 16 {
		tempAccountList[address] = struct{}{}
	}
}

func openClassicArbosTest(client *ethclient.Client) (*classicgen.ArbosTestCaller, error) {
	return classicgen.NewArbosTestCaller(ArbosTestAddress, client)
}

func getAccountMap(arbosTest *classicgen.ArbosTestCaller, callopts *bind.CallOpts) (map[common.Address]struct{}, error) {
	result := make(map[common.Address]struct{})
	serializedAccountAddresses, err := arbosTest.GetAllAccountAddresses(callopts)
	if err != nil {
		return nil, err
	}
	for len(serializedAccountAddresses) > 0 {
		address := common.BytesToAddress(serializedAccountAddresses[:32])
		serializedAccountAddresses = serializedAccountAddresses[32:]
		result[address] = struct{}{}
	}
	return result, nil
}

func fillAccounts(writer *JsonMultiListWriter, arbosTest *classicgen.ArbosTestCaller, callopts *bind.CallOpts, addressList, excludeAddresse map[common.Address]struct{}) error {
	for address := range addressList {
		_, exclude := excludeAddresse[address]
		if exclude {
			delete(excludeAddresse, address) // not required, but might save some memory?
			continue
		}
		acctInfo, err := getAccountInfo(arbosTest, callopts, address)
		if err != nil {
			return err
		}
		err = writer.AddElement(acctInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func fillAccountsOld(writer *JsonMultiListWriter, client ethclient.Client, callopts *bind.CallOpts, addressList map[common.Address]struct{}) error {
	for address := range addressList {
		acctInfo, err := getAccountInfoOld(client, callopts, address)
		if err != nil {
			return err
		}
		err = writer.AddElement(acctInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func skipAccounts(reader *JsonMultiListReader) error {
	for reader.More() {
		var acctInfo AccountInitializationInfo
		err := reader.GetNextElement(&acctInfo)
		if err != nil {
			return err
		}
		AddressSeen(acctInfo.Addr)
	}
	return nil
}

func getAccountHashesAsMap(arbosTest *classicgen.ArbosTestCaller, callopts *bind.CallOpts) (map[common.Hash]struct{}, error) {
	serializedAccountHashes, err := arbosTest.GetAllAccountHashes(callopts)
	if err != nil {
		return nil, err
	}
	ret := make(map[common.Hash]struct{})
	for len(serializedAccountHashes) > 0 {
		ret[common.BytesToHash(serializedAccountHashes[:32])] = struct{}{}
		serializedAccountHashes = serializedAccountHashes[32:]
	}
	return ret, nil
}

func getAccountInfoOld(client ethclient.Client, callopts *bind.CallOpts, addr common.Address) (AccountInitializationInfo, error) {

	arbosGetAccountInfo := arbostestABI.Methods["getAccountInfo"]
	getInfoArgs, err := arbosGetAccountInfo.Inputs.Pack(addr)
	if err != nil {
		return AccountInitializationInfo{}, err
	}
	getInfoData := make([]byte, len(arbosGetAccountInfo.ID))
	copy(getInfoData, arbosGetAccountInfo.ID)
	getInfoData = append(getInfoData, getInfoArgs...)

	callmsg := ethereum.CallMsg{
		From: common.Address{},
		To:   &ArbosTestAddress,
		Data: getInfoData,
	}
	ctx := callopts.Context
	blockNum := callopts.BlockNumber
	// balance := client.BalanceAt(ctx, addr, blockNum)
	// nonce := client.NonceAt(ctx, addr, blockNum)
	code, err := client.CodeAt(ctx, addr, blockNum)
	if err != nil {
		return AccountInitializationInfo{}, err
	}
	serializedInfo, err := client.CallContract(ctx, callmsg, blockNum)
	if err != nil {
		return AccountInitializationInfo{}, err
	}
	infoReader := bytes.NewReader(serializedInfo)
	balanceHash, err := util.HashFromReader(infoReader)
	if err != nil {
		return AccountInitializationInfo{}, err
	}
	nonceHash, err := util.HashFromReader(infoReader)
	if err != nil {
		return AccountInitializationInfo{}, err
	}
	nonceBig := nonceHash.Big()
	if !nonceBig.IsUint64() {
		return AccountInitializationInfo{}, errors.New("nonce not uint64")
	}
	storage := make(map[common.Hash]common.Hash)
	for infoReader.Len() > 64 {
		key, err := util.HashFromReader(infoReader)
		if err != nil {
			return AccountInitializationInfo{}, err
		}
		val, err := util.HashFromReader(infoReader)
		if err != nil {
			return AccountInitializationInfo{}, err
		}
		storage[key] = val
	}
	var contractInfo *AccountInitContractInfo = nil
	if (len(code)) > 0 || len(storage) > 0 {
		contractInfo = &AccountInitContractInfo{
			Code:            code,
			ContractStorage: storage,
		}
	}
	return AccountInitializationInfo{
		Addr:            addr,
		Nonce:           nonceBig.Uint64(),
		EthBalance:      balanceHash.Big(),
		ContractInfo:    contractInfo,
		AggregatorInfo:  nil,
		AggregatorToPay: nil,
		ClassicHash:     common.Hash{},
	}, nil
}

func getAccountInfo(arbosTest *classicgen.ArbosTestCaller, callopts *bind.CallOpts, addr common.Address) (AccountInitializationInfo, error) {
	serializedInfo, err := arbosTest.GetSerializedEVMState(callopts, addr)
	if err != nil {
		return AccountInitializationInfo{}, err
	}
	rd := bytes.NewReader(serializedInfo)

	// verify that first 32 bytes is addr
	inAddr, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return AccountInitializationInfo{}, err
	}
	if inAddr != addr {
		panic("Address mismatch in serialized account state")
	}

	nonce, err := util.HashFromReader(rd)
	if err != nil {
		return AccountInitializationInfo{}, err
	}
	nonceBig := nonce.Big()
	if !nonceBig.IsUint64() {
		return AccountInitializationInfo{}, errors.New("invalid nonce in serialized account state")
	}

	ethBalance, err := util.HashFromReader(rd)
	if err != nil {
		return AccountInitializationInfo{}, err
	}

	var flag [1]byte
	if _, err := rd.Read(flag[:]); err != nil {
		return AccountInitializationInfo{}, err
	}
	var contractInfo *AccountInitContractInfo
	if flag[0] != 0 {
		contractInfo, err = getAccountContractInfo(rd)
		if err != nil {
			return AccountInitializationInfo{}, err
		}
	}

	if _, err := rd.Read(flag[:]); err != nil {
		return AccountInitializationInfo{}, err
	}
	var aggInfo *AccountInitAggregatorInfo
	if flag[0] != 0 {
		aggInfo, err = getAccountAggregatorInfo(rd)
		if err != nil {
			return AccountInitializationInfo{}, err
		}
	}

	if _, err := rd.Read(flag[:]); err != nil {
		return AccountInitializationInfo{}, err
	}
	var aggregatorToPay *common.Address
	if flag[0] != 0 {
		atp, err := util.AddressFrom256FromReader(rd)
		if err != nil {
			return AccountInitializationInfo{}, err
		}
		aggregatorToPay = &atp
	}

	classicHash, err := util.HashFromReader(rd)
	if err != nil {
		return AccountInitializationInfo{}, err
	}

	return AccountInitializationInfo{
		Addr:            addr,
		Nonce:           nonceBig.Uint64(),
		EthBalance:      ethBalance.Big(),
		ContractInfo:    contractInfo,
		AggregatorInfo:  aggInfo,
		AggregatorToPay: aggregatorToPay,
		ClassicHash:     classicHash,
	}, nil
}

func getAccountContractInfo(rd io.Reader) (*AccountInitContractInfo, error) {
	sizeBuf, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	sizeBig := sizeBuf.Big()
	if !sizeBig.IsUint64() {
		return nil, errors.New("invalid code size")
	}
	size := sizeBig.Uint64()
	code := make([]byte, size)
	if _, err := rd.Read(code); err != nil {
		return nil, err
	}

	sizeBuf, err = util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	sizeBig = sizeBuf.Big()
	if !sizeBig.IsUint64() {
		return nil, errors.New("invalid contract storage size")
	}
	size = sizeBig.Uint64()
	storage := make(map[common.Hash]common.Hash)
	for i := uint64(0); i < size; i++ {
		index, err := util.HashFromReader(rd)
		if err != nil {
			return nil, err
		}
		value, err := util.HashFromReader(rd)
		if err != nil {
			return nil, err
		}
		storage[index] = value
	}
	return &AccountInitContractInfo{
		Code:            code,
		ContractStorage: storage,
	}, nil
}

func getAccountAggregatorInfo(rd io.Reader) (*AccountInitAggregatorInfo, error) {
	feeCollector, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}
	baseTxFee, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	return &AccountInitAggregatorInfo{
		FeeCollector: feeCollector,
		BaseFeeL1Gas: baseTxFee.Big(),
	}, nil
}

func copyStillValidAccounts(reader *JsonMultiListReader, writer *JsonMultiListWriter, currentHashes map[common.Hash]struct{}) (map[common.Address]struct{}, error) {
	foundAddresses := make(map[common.Address]struct{})
	for reader.More() {
		var accountInfo AccountInitializationInfo
		err := reader.GetNextElement(&accountInfo)
		if err != nil {
			return nil, err
		}
		_, exists := currentHashes[accountInfo.ClassicHash]
		if exists {
			err := writer.AddElement(accountInfo)
			if err != nil {
				return nil, err
			}
			foundAddresses[accountInfo.Addr] = struct{}{}
			delete(currentHashes, accountInfo.ClassicHash)
		}
		AddressSeen(accountInfo.Addr)
	}
	return foundAddresses, nil
}
