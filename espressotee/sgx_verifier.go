package espressotee

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/solgen/go/espressogen"
)

type EspressoSGXVerifierInterface interface {
	Verify(opts *bind.CallOpts, rawQuote []byte, reportDataHash [32]byte) (espressogen.EnclaveReport, error)
}

type EspressoSGXVerifier struct {
	verifier *espressogen.IEspressoSGXTEEVerifier
}

func (v *EspressoSGXVerifier) Verify(opts *bind.CallOpts, rawQuote []byte, reportDataHash [32]byte) (espressogen.EnclaveReport, error) {
	return v.verifier.Verify(opts, rawQuote, reportDataHash)
}

func NewEspressoSGXVerifier(l1Client *ethclient.Client, addr common.Address) (*EspressoSGXVerifier, error) {
	verifier, err := espressogen.NewIEspressoSGXTEEVerifier(addr, l1Client)
	if err != nil {
		return nil, err
	}
	return &EspressoSGXVerifier{verifier: verifier}, nil
}
