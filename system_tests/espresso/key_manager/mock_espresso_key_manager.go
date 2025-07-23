package key_manager

import (
	"crypto/ecdsa"
	crypto_rand "crypto/rand"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	key_manager "github.com/offchainlabs/nitro/espresso/key-manager"
	"github.com/offchainlabs/nitro/espressotee"
)

// MockEspressoKeyManager is a mock implementation of the
// key_manager.EspressoKeyManagerInterface.
//
// It is used for testing purposes and provides a simple implementation
// of the Espresso key management functionality without requiring a real
// Espresso environment.
type MockEspressoKeyManager struct {
	Key *ecdsa.PrivateKey
}

// Compile time check to ensure that MockEspressoKeyManager implements the
// key_manager.EspressoKeyManagerInterface.
var _ key_manager.EspressoKeyManagerInterface = &MockEspressoKeyManager{}

// MockEspressoKeyManagerConfig holds the configuration options for the
// MockEspressoKeyManager. It governs the customization of the
// MockEspressoKeyManager at the time of its creation using the
// NewMockEspressoKeyManager function.
type MockEspressoKeyManagerConfig struct {
	PrivateKey *ecdsa.PrivateKey
}

// MockEspressoKeyManagerOption is a function that configures the
// MockEspressoKeyManagerConfig.
type MockEspressoKeyManagerOption func(*MockEspressoKeyManagerConfig)

// WithPrivateKey is a MockEspressoKeyManagerOption that allows setting a
// custom private key for the MockEspressoKeyManager.
func WithPrivateKey(privKey *ecdsa.PrivateKey) MockEspressoKeyManagerOption {
	return func(config *MockEspressoKeyManagerConfig) {
		config.PrivateKey = privKey
	}
}

// mustGenerateNewPrivateKey generates a new ECDSA private key using the
// crypto/rand package.
//
// NOTE: This function panics if the key generation fails.
func mustGenerateNewPrivateKey() *ecdsa.PrivateKey {
	privKey, err := ecdsa.GenerateKey(crypto.S256(), crypto_rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("failed to generate mock private key: %v", err))
	}
	return privKey
}

// NewMockEspressoKeyManager creates a new instance of MockEspressoKeyManager
// with a randomly generated private key.
func NewMockEspressoKeyManager(options ...MockEspressoKeyManagerOption) *MockEspressoKeyManager {
	// Setup config with default options / values
	config := MockEspressoKeyManagerConfig{
		PrivateKey: mustGenerateNewPrivateKey(),
	}

	for _, option := range options {
		option(&config)
	}

	return &MockEspressoKeyManager{
		Key: config.PrivateKey,
	}
}

// GetCurrentKey implements key_manager.EspressoKeyManagerInterface.
func (m *MockEspressoKeyManager) GetCurrentKey() *ecdsa.PublicKey {
	return &m.Key.PublicKey
}

// HasRegistered implements key_manager.EspressoKeyManagerInterface.
func (m *MockEspressoKeyManager) HasRegistered() bool {
	return false
}

// Register implements key_manager.EspressoKeyManagerInterface.
func (m *MockEspressoKeyManager) Register(getAttestationFunc func([]byte) ([]byte, error)) error {
	return nil
}

// SignBatch implements key_manager.EspressoKeyManagerInterface.
func (m *MockEspressoKeyManager) SignBatch(message []byte) ([]byte, error) {
	hash := crypto.Keccak256Hash(message)
	return crypto.Sign(hash.Bytes(), m.Key)
}

// SignHotShotPayload implements key_manager.EspressoKeyManagerInterface.
func (m *MockEspressoKeyManager) SignHotShotPayload(message []byte) ([]byte, error) {
	hash := crypto.Keccak256Hash(message)
	return crypto.Sign(hash.Bytes(), m.Key)
}

// TeeType implements key_manager.EspressoKeyManagerInterface.
func (m *MockEspressoKeyManager) TeeType() espressotee.TEE {
	return espressotee.SGX
}
