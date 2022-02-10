package statetransfer

import (
	"bytes"
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/solgen/go/classicgen"
)

var ArbosArbRetryableTxAddress = common.HexToAddress("0x000000000000000000000000000000000000006E")

func openClassicArbRetryableTx(client *ethclient.Client) (*classicgen.ArbRetryableTxCaller, error) {
	return classicgen.NewArbRetryableTxCaller(ArbosArbRetryableTxAddress, client)
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

func skipRetriables(reader *JsonMultiListReader) error {
	for reader.More() {
		var retriableData InitializationDataForRetryable
		err := reader.GetNextElement(&retriableData)
		if err != nil {
			return err
		}
	}
	return nil
}
