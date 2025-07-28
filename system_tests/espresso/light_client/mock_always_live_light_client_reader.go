package light_client

import (
	espresso_light_client "github.com/EspressoSystems/espresso-network/sdks/go/light-client"
)

// MockAlwaysLiveLightClientReader is a mock implementation of the
// espresso_light_client.LightClientReaderInterface  that always returns
// true for IsHotShotLive, simulating a scenario where the light client is
// always live.
type MockAlwaysLiveLightClientReader struct {
	UnimplementedLightClientReader
}

// This is compile time check to ensure that MockAlwaysLiveLightClientReader
// implements LightClientReaderInterface.
var _ espresso_light_client.LightClientReaderInterface = &UnimplementedLightClientReader{}

// IsHotShotLive is a mock implementation that always returns true,
func (m *MockAlwaysLiveLightClientReader) IsHotShotLive(delayThreshold uint64) (bool, error) {
	return true, nil
}

func NewMockAlwaysLiveLightClientReader() *MockAlwaysLiveLightClientReader {
	return new(MockAlwaysLiveLightClientReader)
}
