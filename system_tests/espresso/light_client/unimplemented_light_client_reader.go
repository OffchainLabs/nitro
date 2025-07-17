package light_client

import (
	"fmt"

	espresso_light_client "github.com/EspressoSystems/espresso-network/sdks/go/light-client"
	"github.com/EspressoSystems/espresso-network/sdks/go/types/common"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// ErrorLightClientUnimplementedMethod is an error type that is returned when a
// method of the LightClientReaderInterface is called that has not been
// implemented.
type ErrorLightClientUnimplementedMethod struct {
	Method string
}

// Error implements error
func (e ErrorLightClientUnimplementedMethod) Error() string {
	return fmt.Sprintf("unimplemented light-client method: %s", e.Method)
}

// UnimplementedLightClientReader is a struct that implements the
// espresso_light_client.LightClientReaderInterface but panics on all method
// calls. This is useful for testing purposes when you want to ensure that a
// method is not called or to provide a default implementation that can be
// overridden in tests.
type UnimplementedLightClientReader struct{}

// Compile time check to ensure that UnimplementedLightClientReader implements
// the espresso_light_client.LightClientReaderInterface.
var _ espresso_light_client.LightClientReaderInterface = &UnimplementedLightClientReader{}

// FetchMerkleRoot implements lightclient.LightClientReaderInterface.
func (u *UnimplementedLightClientReader) FetchMerkleRoot(hotShotHeight uint64, opts *bind.CallOpts) (common.BlockMerkleSnapshot, error) {
	panic(ErrorLightClientUnimplementedMethod{Method: "FetchMerkleRoot"})
}

// IsHotShotLive implements lightclient.LightClientReaderInterface.
func (u *UnimplementedLightClientReader) IsHotShotLive(delayThreshold uint64) (bool, error) {
	panic(ErrorLightClientUnimplementedMethod{Method: "IsHotShotLive"})
}

// IsHotShotLiveAtHeight implements lightclient.LightClientReaderInterface.
func (u *UnimplementedLightClientReader) IsHotShotLiveAtHeight(height uint64, delayThreshold uint64) (bool, error) {
	panic(ErrorLightClientUnimplementedMethod{Method: "IsHotShotLiveAtHeight"})
}

// ValidatedHeight implements lightclient.LightClientReaderInterface.
func (u *UnimplementedLightClientReader) ValidatedHeight() (validatedHeight uint64, l1Height uint64, err error) {
	panic(ErrorLightClientUnimplementedMethod{Method: "ValidatedHeight"})
}
