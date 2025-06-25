package espressotee

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/solgen/go/espressogen"
)

type EspressoTEEVerifierInterface interface {
	RegisterSigner(
		dataPoster *dataposter.DataPoster,
		attestation []byte,
		data []byte,
		teeType uint8,
		registerSignerOpts EspressoRegisterSignerOpts,
	) error
	RegisteredSigners(
		signer common.Address,
		teeType uint8,
		registerSignerOpts EspressoRegisterSignerOpts,
	) (bool, error)
}

type EspressoTEEVerifier struct {
	contract *espressogen.IEspressoTEEVerifier
	l1Client *ethclient.Client
	address  common.Address
}

func NewEspressoTEEVerifier(contract *espressogen.IEspressoTEEVerifier, l1Client *ethclient.Client, address common.Address) *EspressoTEEVerifier {
	return &EspressoTEEVerifier{contract: contract, l1Client: l1Client, address: address}
}

func (e *EspressoTEEVerifier) RegisterSigner(
	dataPoster *dataposter.DataPoster,
	attestation []byte,
	data []byte,
	teeType uint8,
	registerSignerOpts EspressoRegisterSignerOpts,
) error {
	// First check base fee is low enough
	err := BaseFeeCheck(
		registerSignerOpts.MaxBaseFee,
		registerSignerOpts.MaxRetries,
		registerSignerOpts.RetryBaseFeeDelay,
		func() (*big.Int, error) {
			return dataPoster.BaseFee()
		},
		"register signer: latest base fee is greater than max base fee",
	)
	if err != nil {
		return err
	}

	contractABI, err := espressogen.IEspressoTEEVerifierMetaData.GetAbi()
	if err != nil {
		return err
	}

	// Pack the function arguments (attestation, data, teeType)
	calldata, err := contractABI.Pack("registerSigner", attestation, data, teeType)
	if err != nil {
		return err
	}
	msg := ethereum.CallMsg{
		From:  dataPoster.Sender(),
		To:    &e.address,
		Data:  calldata,
		Value: dataPoster.Auth().Value,
	}

	estimate, err := e.l1Client.EstimateGas(context.Background(), msg)
	if err != nil {
		return err
	}
	nonce, err := e.l1Client.NonceAt(context.Background(), dataPoster.Sender(), nil)
	if err == nil {
		log.Info("registering signer: on chain nonce", "nonce", nonce)
	}
	dataPosterNonce, _, err := dataPoster.GetNextNonceAndMeta(context.Background())
	if err == nil {
		log.Info("registering signer: dataposter next nonce", "nonce", dataPosterNonce)
	}
	// Add a buffer to the estimate for the gas limit
	gasLimit := estimate * (100 + registerSignerOpts.GasLimitBufferIncreasePercent) / 100
	log.Info("register signer gas limit", "gas limit", gasLimit)

	// Since we use batch poster private key to register signer, we need to use dataposter to post transaction
	// So the dataposter can track the proper nonce once we start posting batches
	tx, err := dataPoster.PostSimpleTransaction(context.Background(), e.address, calldata, gasLimit, dataPoster.Auth().Value)
	if err != nil {
		log.Info("failed to post register signer transaction", "err", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), registerSignerOpts.MaxTxnWaitTime)
	defer cancel()
	log.Info("waiting for register signer tx to be mined", "tx", tx.Hash().Hex(), "timeout", registerSignerOpts.MaxTxnWaitTime)

	receipt, err := bind.WaitMined(ctx, e.l1Client, tx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf(
				"register signer timed out after %v minutes waiting for tx %s to be mined",
				registerSignerOpts.MaxTxnWaitTime,
				tx.Hash().Hex(),
			)
		}
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("transaction failed")
	}

	log.Info("register signer tx succeeded", "tx", tx.Hash().Hex())

	return nil
}

func (e *EspressoTEEVerifier) RegisteredSigners(address common.Address, teeType uint8, registerSignerOpts EspressoRegisterSignerOpts) (bool, error) {
	ok, err := ContractVerification(
		registerSignerOpts.MaxRetries,
		registerSignerOpts.RetryReadContractDelay,
		func() (bool, error) {
			return e.contract.RegisteredSigners(&bind.CallOpts{}, address, teeType)
		},
		"address not yet registered in contract",
	)
	if err != nil {
		return false, err
	}
	return ok, nil
}
