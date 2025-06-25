package espressotee

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/hf/nitrite"
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/solgen/go/espressogen"
)

type EspressoNitroTEEVerifierInterface interface {
	VerifyCert(
		dataPoster *dataposter.DataPoster,
		certificate []byte,
		parentCertHash [32]byte,
		isCA bool,
		registerSignerOpts EspressoRegisterSignerOpts,
	) (common.Hash, error)
	VerifyAttestationAndCertificates(
		attestationBytes []byte,
		dataPoster *dataposter.DataPoster,
		registerSignerOpts EspressoRegisterSignerOpts,
	) ([]byte, []byte, error)
	IsPCR0HashRegistered(pcr0Hash [32]byte) (bool, error)
}

type EspressoNitroTEEVerifier struct {
	contract *espressogen.IEspressoNitroTEEVerifier
	l1Client *ethclient.Client
	address  common.Address
}

func NewEspressoNitroTEEVerifier(contract *espressogen.IEspressoNitroTEEVerifier, l1Client *ethclient.Client, nitroAddr common.Address) *EspressoNitroTEEVerifier {
	return &EspressoNitroTEEVerifier{contract: contract, l1Client: l1Client, address: nitroAddr}
}

func (e *EspressoNitroTEEVerifier) IsPCR0HashRegistered(pcr0Hash [32]byte) (bool, error) {
	return e.contract.RegisteredEnclaveHash(&bind.CallOpts{}, pcr0Hash)
}

/**
 * This functions checks and verifies a certificate on-chain.
 * Always verify certificate on chain, if certificate is already verified it is very cheap to verify again on chain
 */
func (e *EspressoNitroTEEVerifier) VerifyCert(
	dataPoster *dataposter.DataPoster,
	certificate []byte, parentCertHash [32]byte,
	isCA bool,
	registerSignerOpts EspressoRegisterSignerOpts,
) (common.Hash, error) {
	// Get certificate hash
	certHash := crypto.Keccak256Hash(certificate)

	// Try and verify the certificate either CA or client
	contractABI, err := espressogen.IEspressoNitroTEEVerifierMetaData.GetAbi()
	if err != nil {
		return certHash, err
	}

	// Always reverify the certificate, this is cheap once verified
	// Pack the function arguments (cerificate, parentCertHash)
	var calldata []byte
	if isCA {
		calldata, err = contractABI.Pack("verifyCACert", certificate, parentCertHash)
	} else {
		calldata, err = contractABI.Pack("verifyClientCert", certificate, parentCertHash)
	}
	if err != nil {
		return certHash, err
	}
	msg := ethereum.CallMsg{
		From:  dataPoster.Sender(),
		To:    &e.address,
		Data:  calldata,
		Value: dataPoster.Auth().Value,
	}
	estimate, err := e.l1Client.EstimateGas(context.Background(), msg)
	if err != nil {
		return certHash, err
	}
	nonce, err := e.l1Client.NonceAt(context.Background(), dataPoster.Sender(), nil)
	if err == nil {
		log.Info("verify cert: on chain nonce", "nonce", nonce)
	}
	dataPosterNonce, _, err := dataPoster.GetNextNonceAndMeta(context.Background())
	if err == nil {
		log.Info("verify cert: dataposter next nonce", "nonce", dataPosterNonce)
	}
	// Add a buffer to the estimate for the gas limit
	gasLimit := estimate * (100 + registerSignerOpts.GasLimitBufferIncreasePercent) / 100
	log.Info("verify cert gas limit", "gas limit", gasLimit)

	// Since we use batch poster private key to register signer, we need to use dataposter to post transaction
	// So the dataposter can track the proper nonce once we start posting batches
	tx, err := dataPoster.PostSimpleTransaction(
		context.Background(),
		e.address,
		calldata,
		gasLimit,
		dataPoster.Auth().Value,
	)
	if err != nil {
		return certHash, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), registerSignerOpts.MaxTxnWaitTime)
	defer cancel()

	log.Info("Waiting for cert tx to be mined",
		"tx", tx.Hash().Hex(),
		"isCA", isCA,
		"timeout", registerSignerOpts.MaxTxnWaitTime,
	)

	receipt, err := bind.WaitMined(ctx, e.l1Client, tx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return certHash, fmt.Errorf("cert verification timed out after %v minutes waiting for tx %s to be mined", registerSignerOpts.MaxTxnWaitTime, tx.Hash().Hex())
		}
		return certHash, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return certHash, errors.New("cert transaction failed")
	}

	// Make sure certificate is verified, after tx succeeded this should always be the case
	// Add retries in case of delay on chain
	verified, err := ContractVerification(
		registerSignerOpts.MaxRetries,
		registerSignerOpts.RetryReadContractDelay,
		func() (bool, error) {
			return e.contract.CertVerified(&bind.CallOpts{}, certHash)
		},
		"attestation certificate is not yet verified",
	)
	if err != nil {
		return certHash, err
	}
	if verified {
		log.Info("cert verified", "cert hash", certHash, "isCA", isCA)
		return certHash, nil
	} else {
		return certHash, errors.New("attestation certificate is not registered in contract even after successful transaction")
	}
}

/**
 * This function validates parses the attestation result we received from AWS Nitro Secure Module (NSM) then validates the following on-chain
 * 1. The PCR0 hash is registered in the espresso nitro tee verifier contract
 * 2. The CA certificate chain
 * 3. The client certificate
 */
func (e *EspressoNitroTEEVerifier) VerifyAttestationAndCertificates(
	attestationBytes []byte,
	dataPoster *dataposter.DataPoster,
	registerSignerOpts EspressoRegisterSignerOpts,
) ([]byte, []byte, error) {
	// First check base fee is low enough
	err := BaseFeeCheck(
		registerSignerOpts.MaxBaseFee,
		registerSignerOpts.MaxRetries,
		registerSignerOpts.RetryBaseFeeDelay,
		func() (*big.Int, error) {
			return dataPoster.BaseFee()
		},
		"verify certificate: latest base fee is greater than max base fee",
	)
	if err != nil {
		return nil, nil, err
	}

	// Unmarshal attestation document
	var res nitrite.Result
	err = json.Unmarshal(attestationBytes, &res)
	if err != nil {
		return nil, nil, err
	}

	pcr0Hash := crypto.Keccak256Hash(res.Document.PCRs[0])
	log.Info("successfully got attestation", "pcr0 hash", pcr0Hash)

	// Before verifying certificates on chain, check if the pcr0 hash is registered to save gas
	verified, err := e.IsPCR0HashRegistered(pcr0Hash)
	if err != nil {
		log.Error("failed to check if pcr0 hash is verified", "pcr0 hash", pcr0Hash)
		return nil, nil, err
	}

	if !verified {
		return nil, nil, fmt.Errorf("prc0 hash is not registered in espresso tee verifier contract")
	}

	// Verify CA certificate chain
	if len(res.Document.CABundle) == 0 {
		return nil, nil, errors.New("CA bundle is empty")
	}

	// Go over the CA certificate bundle in attestation and verify each
	parentCertHash := crypto.Keccak256Hash(res.Document.CABundle[0])
	for i := 0; i < len(res.Document.CABundle); i++ {
		cert := res.Document.CABundle[i]
		// Verify current certificate against parent hash in NitroEspressoTEEVerifier contracts
		certHash, err := e.VerifyCert(dataPoster, cert, parentCertHash, true, registerSignerOpts)
		if err != nil {
			log.Error("failed to get CA cert verified", "index", i, "err", err)
			return nil, nil, err
		}

		// For next certificate in bundle we need to compare it with this certificate hash
		parentCertHash = certHash
	}

	// Verify client certificate
	_, err = e.VerifyCert(dataPoster, res.Document.Certificate, parentCertHash, false, registerSignerOpts)
	if err != nil {
		log.Error("failed to get client cert verified", "err", err)
		return nil, nil, err
	}

	// Return attestation and signature
	return res.COSESign1, res.Signature, nil
}
