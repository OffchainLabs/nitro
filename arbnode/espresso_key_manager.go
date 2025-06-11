package arbnode

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/espressotee"
	"github.com/offchainlabs/nitro/util/signature"
)

type TEE = espressotee.TEE

const (
	SGX   = espressotee.SGX
	NITRO = espressotee.NITRO
)

type EspressoKeyManagerInterface interface {
	HasRegistered() (bool, error)
	Register(getAttestationFunc func([]byte) ([]byte, error)) error
	GetCurrentKey() *ecdsa.PublicKey
	SignHotShotPayload(message []byte) ([]byte, error)
	SignBatch(message []byte) ([]byte, error)
	TeeType() TEE
}

var _ EspressoKeyManagerInterface = &EspressoKeyManager{}

type EspressoKeyManager struct {
	espressoTEEVerifierCaller espressotee.EspressoTEEVerifierInterface
	espressoNitroTEEVerifier  espressotee.EspressoNitroTEEVerifierInterface
	pubKey                    *ecdsa.PublicKey
	privKey                   *ecdsa.PrivateKey

	batchPosterSigner  signature.DataSignerFunc
	dataPoster         *dataposter.DataPoster
	teeType            TEE
	registerSignerOpts espressotee.EspressoRegisterSignerOpts

	hasRegistered bool
}

func NewEspressoKeyManager(espressoTEEVerifierCaller espressotee.EspressoTEEVerifierInterface, espressoNitroTEEVerifier espressotee.EspressoNitroTEEVerifierInterface, dataPoster *dataposter.DataPoster, signerFunc signature.DataSignerFunc, teeType TEE, registerSignerConfig espressotee.EspressoRegisterSignerConfig) *EspressoKeyManager {
	// ephemeral key
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	pubKey, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		panic("failed to get public key")
	}

	if signerFunc == nil {
		panic("DataSigner is nil")
	}

	if registerSignerConfig.GasLimitBufferIncreasePercent > 20 {
		panic("Gas limit buffer increase should not be greater than 20 percent")
	}

	if registerSignerConfig.MaxRetries > 10 {
		panic("Max retries cannot be more than 10")
	}

	if registerSignerConfig.MaxTxnWaitTime > 5*time.Minute {
		panic("Max txn wait time cannot be more than 5 minutes")
	}

	if registerSignerConfig.RetryDelay > 20*time.Second {
		panic("Retry delay cannot be more than 20 seconds")
	}

	return &EspressoKeyManager{
		pubKey:                    pubKey,
		privKey:                   privKey,
		batchPosterSigner:         signerFunc,
		espressoTEEVerifierCaller: espressoTEEVerifierCaller,
		espressoNitroTEEVerifier:  espressoNitroTEEVerifier,
		dataPoster:                dataPoster,
		teeType:                   teeType,
		registerSignerOpts: espressotee.EspressoRegisterSignerOpts{
			MaxTxnWaitTime:                registerSignerConfig.MaxTxnWaitTime,
			MaxRetries:                    int(registerSignerConfig.MaxRetries),
			RetryDelay:                    registerSignerConfig.RetryDelay,
			GasLimitBufferIncreasePercent: registerSignerConfig.GasLimitBufferIncreasePercent,
		},
	}
}

func (k *EspressoKeyManager) HasRegistered() (bool, error) {
	if k.hasRegistered {
		return true, nil
	}
	pubKey, ok := k.privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		panic("failed to get public key")
	}
	signerAddr := crypto.PubkeyToAddress(*pubKey)
	for i := 0; i < k.registerSignerOpts.MaxRetries; i++ {
		ok, err := k.espressoTEEVerifierCaller.RegisteredSigners(signerAddr, uint8(k.teeType))
		if err != nil {
			return false, err
		}

		if ok {
			return ok, nil
		}

		if i < k.registerSignerOpts.MaxRetries-1 {
			log.Info("address not registered in contract again, retrying...")
			time.Sleep(k.registerSignerOpts.RetryDelay)
		}
	}
	return false, nil
}

/*
 * This function will get the attestation in order to properly register the signing address on chain for a given TEE type
 */
func (k *EspressoKeyManager) PrepareRegisterSigner(getAttestationFunc func([]byte) ([]byte, error)) ([]byte, []byte, error) {
	signerAddr := crypto.PubkeyToAddress(*k.pubKey)
	switch k.teeType {
	case SGX:
		addr := signerAddr.Bytes()
		log.Info("sgx signing address", "addr", signerAddr)

		attestationQuote, err := getAttestationFunc(addr)
		if err != nil {
			return nil, nil, fmt.Errorf("sgx signing failed: %w", err)
		}
		return attestationQuote, addr, nil

	case NITRO:
		pubKeyBytes := crypto.FromECDSAPub(k.pubKey)
		log.Info("nitro signing address", "addr", signerAddr)

		attestationBytes, err := getAttestationFunc(pubKeyBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("nitro signing failed: %w", err)
		}

		attestation, data, err := k.espressoNitroTEEVerifier.VerifyAttestationAndCertificates(
			attestationBytes,
			k.dataPoster,
			k.registerSignerOpts,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("attestation verification failed: %w", err)
		}
		return attestation, data, nil

	default:
		return nil, nil, fmt.Errorf("unsupported TEE type: %v", k.teeType)
	}
}

func (k *EspressoKeyManager) Register(getAttestationFunc func([]byte) ([]byte, error)) error {
	if k.hasRegistered {
		log.Info("EspressoKeyManager already registered")
		return nil
	}

	// Get the attestation and data needed to register the signer
	attestation, data, err := k.PrepareRegisterSigner(getAttestationFunc)
	if err != nil {
		return err
	}

	err = k.espressoTEEVerifierCaller.RegisterSigner(k.dataPoster, attestation, data, uint8(k.teeType), k.registerSignerOpts)
	if err != nil {
		return err
	}

	signerAddr := crypto.PubkeyToAddress(*k.pubKey)
	log.Info("Register signer succeeded", "signer address", signerAddr.Hex())

	// Verify our address is actually registered in contract
	hasRegistered, err := k.HasRegistered()
	if err != nil {
		return err
	}
	if !hasRegistered {
		return errors.New("address is not registered in contract")
	}
	k.hasRegistered = true
	return nil
}

func (k *EspressoKeyManager) GetCurrentKey() *ecdsa.PublicKey {
	return k.pubKey
}

func (k *EspressoKeyManager) TeeType() TEE {
	return k.teeType
}

func (k *EspressoKeyManager) SignHotShotPayload(message []byte) ([]byte, error) {
	hash := crypto.Keccak256Hash(message)
	return k.batchPosterSigner(hash.Bytes())
}

func (k *EspressoKeyManager) SignBatch(message []byte) ([]byte, error) {
	hash := crypto.Keccak256Hash(message)
	return crypto.Sign(hash.Bytes(), k.privKey)
}
