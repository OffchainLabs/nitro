package extractionfunction

import (
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

var seqInboxABI *abi.ABI

func init() {
	var err error
	seqInboxABI, err = bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
}
