//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package statetransfer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/solgen/go/classicgen"
	"io"
	"math/big"
	"math/rand"
)

func GetDataFromClassicAsJson(maybeUrl *string, sampleRate *float64) ([]byte, error) {
	data, err := getDataFromClassic(maybeUrl, sampleRate)
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func getDataFromClassic(maybeUrl *string, sampleRate *float64) (*ArbosInitializationInfo, error) {
	ctx := context.Background()
	client, err := openClassicClient(maybeUrl)
	if err != nil {
		return nil, err
	}

	callopts := &bind.CallOpts{
		Pending:     false,
		From:        common.Address{},
		BlockNumber: nil,
		Context:     ctx,
	}

	classicArbAddressTable, err := openClassicArbAddressTable(client)
	if err != nil {
		return nil, err
	}
	addressTableData, err := getAddressTableContents(classicArbAddressTable, callopts)
	if err != nil {
		return nil, err
	}

	classicArbosTest, err := openClassicArbosTest(client)
	if err != nil {
		return nil, err
	}

	accountAddresses, err := getAccountAddresses(classicArbosTest, callopts)
	if err != nil {
		return nil, err
	}

	accounts := []AccountInitializationInfo{}
	for _, addr := range accountAddresses {
		if sampleRate == nil || rand.Float64() < *sampleRate {
			acctInfo, err := getAccountInfo(classicArbosTest, callopts, addr)
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, acctInfo)
		}
	}

	classicArbRetryableTx, err := openClassicArbRetryableTx(client)
	if err != nil {
		return nil, err
	}
	retryables, err := getRetryables(classicArbRetryableTx, callopts)
	if err != nil {
		return nil, err
	}

	return &ArbosInitializationInfo{
		AddressTableContents: addressTableData,
		SendPartials:         []common.Hash{},
		DefaultAggregator:    common.Address{},
		RetryableData:        retryables,
		Accounts:             accounts,
	}, nil
}

func openClassicClient(maybeUrl *string) (*ethclient.Client, error) {
	if maybeUrl == nil {
		url := "https://Arb1-graph.arbitrum.io/rpc"
		maybeUrl = &url
	}
	return ethclient.Dial(*maybeUrl)
}

const ArbAddressTableAsInt = 102
const ArbosTestAddressAsInt = 105
const ArbosArbRetryableTxAddressAsInt = 110

func openClassicArbosTest(client *ethclient.Client) (*classicgen.ArbosTestCaller, error) {
	return classicgen.NewArbosTestCaller(common.BigToAddress(big.NewInt(ArbosTestAddressAsInt)), client)
}

func getAccountAddresses(arbosTest *classicgen.ArbosTestCaller, callopts *bind.CallOpts) ([]common.Address, error) {
	serializedAccountAddresses, err := arbosTest.GetAllAccountAddresses(callopts)
	if err != nil {
		return nil, err
	}
	ret := []common.Address{}
	for len(serializedAccountAddresses) > 0 {
		ret = append(ret, common.BytesToAddress(serializedAccountAddresses[:32]))
		serializedAccountAddresses = serializedAccountAddresses[32:]
	}
	return ret, nil
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

	return AccountInitializationInfo{
		Addr:            addr,
		Nonce:           nonce.Big().Uint64(),
		EthBalance:      ethBalance.Big(),
		ContractInfo:    contractInfo,
		AggregatorInfo:  aggInfo,
		AggregatorToPay: aggregatorToPay,
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

func openClassicArbRetryableTx(client *ethclient.Client) (*classicgen.ArbRetryableTxCaller, error) {
	return classicgen.NewArbRetryableTxCaller(common.BigToAddress(big.NewInt(ArbosArbRetryableTxAddressAsInt)), client)
}

func getRetryables(caller *classicgen.ArbRetryableTxCaller, callopts *bind.CallOpts) ([]InitializationDataForRetryable, error) {
	serializedRetryables, err := caller.SerializeAllRetryables(callopts)
	if err != nil {
		return []InitializationDataForRetryable{}, err
	}
	rd := bytes.NewReader(serializedRetryables)
	retryables := []InitializationDataForRetryable{}
	for rd.Len() > 0 {
		retryable, err := getRetryable(rd)
		if err != nil {
			return []InitializationDataForRetryable{}, err
		}
		retryables = append(retryables, retryable)
	}
	return retryables, nil
}

func getRetryable(rd io.Reader) (InitializationDataForRetryable, error) {
	txId, err := util.HashFromReader(rd)
	if err != nil {
		return InitializationDataForRetryable{}, err
	}
	sender, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return InitializationDataForRetryable{}, err
	}
	destination, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return InitializationDataForRetryable{}, err
	}
	callvalue, err := util.HashFromReader(rd)
	if err != nil {
		return InitializationDataForRetryable{}, err
	}
	beneficiary, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return InitializationDataForRetryable{}, err
	}
	expiryTimeHash, err := util.HashFromReader(rd)
	if err != nil {
		return InitializationDataForRetryable{}, err
	}
	expiryTimeBig := expiryTimeHash.Big()
	if !expiryTimeBig.IsUint64() {
		return InitializationDataForRetryable{}, errors.New("invalid expiry time in getRetryable")
	}
	calldataSizeHash, err := util.HashFromReader(rd)
	if err != nil {
		return InitializationDataForRetryable{}, err
	}
	calldataSizeBig := calldataSizeHash.Big()
	if !calldataSizeBig.IsUint64() {
		return InitializationDataForRetryable{}, errors.New("invalid calldata size in getRetryable")
	}
	calldata := make([]byte, calldataSizeBig.Uint64())
	if _, err = rd.Read(calldata); err != nil {
		return InitializationDataForRetryable{}, err
	}

	return InitializationDataForRetryable{
		Id:          txId,
		Timeout:     expiryTimeBig.Uint64(),
		From:        sender,
		To:          destination,
		Callvalue:   callvalue.Big(),
		Beneficiary: beneficiary,
		Calldata:    calldata,
	}, nil
}
