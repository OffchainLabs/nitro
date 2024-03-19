package arbnode

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type MockLightClientReader struct {
}

func NewMockLightClientReader(lightClientAddr common.Address, l1client bind.ContractBackend) (*MockLightClientReader, error) {
	return &MockLightClientReader{}, nil
}

// Returns the L1 block number where the light client has validated a particular
// hotshot block number
func (l *MockLightClientReader) ValidatedHeight() (validatedHeight uint64, l1Height uint64, err error) {
	return 18446744073709551615, 18446744073709551615, nil
}
