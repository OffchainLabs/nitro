package key_manager_test

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/system_tests/espresso/key_manager"
)

// fakeRandReader is a simple implementation of io.Reader that returns a
// deterministic byte slice. This is used for testing purposes to ensure
// consistent behavior in tests.
type fakeRandReader struct{}

// Read implements the io.Reader
//
// This implementation simply fills the byte slice with an ascending sequence of
// values from 0 to 255, repeating as necessary. This is not secure and should
// only be used for testing purposes.
func (f *fakeRandReader) Read(p []byte) (n int, err error) {
	// Fill the byte slice with a simple pattern, for deterministic output.
	for i, l, b := uint64(0), uint64(len(p)), uint8(0); i < l; i, b = i+1, b+1 {
		p[i] = b
	}

	return len(p), nil
}

// ExampleNewMockEspressoKeyManager demonstrates how to create a new
// MockEspressoKeyManager instance with a private key, and how to use it to
// sign messages and retrieve the public key.
func ExampleNewMockEspressoKeyManager() {
	// NOTE: This example uses a deterministic random reader for the
	// private key generation for this example to work.  In practice, you
	// should always use a secure random number generator, such as
	// crypto/rand.Reader
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), new(fakeRandReader))
	if err != nil {
		panic(fmt.Errorf("failed to generate private key: %w", err))
	}

	km := key_manager.NewMockEspressoKeyManager(
		key_manager.WithPrivateKey(privateKey),
	)

	{
		// Public Key
		publicKey := km.GetCurrentKey()
		address := crypto.PubkeyToAddress(*publicKey)

		fmt.Printf("public key: %x\n", address)
		fmt.Printf("address from public key: %s\n", address)
	}

	{
		// Sign SignHotShotPayload example
		signature, err := km.SignHotShotPayload([]byte("hotshot payload"))
		if err != nil {
			panic(err)
		}

		fmt.Printf("signature for sign hotshot payload: %x\n", signature)
	}

	{
		// Sign Batch Example
		signature, err := km.SignBatch([]byte("test message"))
		if err != nil {
			panic(err)
		}

		fmt.Printf("signature for batch: %x\n", signature)
	}

	// Output:
	// public key: ede35562d3555e61120a151b3c8e8e91d83a378a
	// address from public key: 0xedE35562d3555e61120a151B3c8e8e91d83a378a
	// signature for sign hotshot payload: 68a9dc2c647ea7a46401da33410937c4eebd26fc3555ec27f49e1dedc1a365ec7db54dc6a420d2c5331beb1f62537ead5b8df3d477191ef6c6555400c995c69301
	// signature for batch: b709442f40834aeb37134db52e846737101cf403d337e463a8980546507fcb3f07da4926475c90f2349b7812977e3788f43b18f8bcf128e08cb3b685b3298cd001
}
