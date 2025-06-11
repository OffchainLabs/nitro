package arbtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func createDummyEspressoMetadata(t *testing.T) []byte {
	hotshotHeight := new(big.Int).SetUint64(1)
	signature := make([]byte, 32)
	teeType := uint8(0)

	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		t.Fatal("failed to create uint256 type")
	}

	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		t.Fatal("failed to create bytes type")
	}

	uint8Type, err := abi.NewType("uint8", "", nil)
	if err != nil {
		t.Fatal("failed to create uint8 type")
	}

	espressoMetadata, err := abi.Arguments{
		{Type: uint256Type},
		{Type: bytesType},
		{Type: uint8Type},
	}.Pack(hotshotHeight, signature, teeType)
	if err != nil {
		t.Fatal("failed to pack hotshot height and signature")
	}

	return espressoMetadata
}
