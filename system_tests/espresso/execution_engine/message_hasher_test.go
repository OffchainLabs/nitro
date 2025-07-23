package execution_engine_test

import (
	"crypto"
	"fmt"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	execution_engine "github.com/offchainlabs/nitro/system_tests/espresso/execution_engine"
)

// ExampleKeccakHasher demonstrates how to use the KeccakHasher
// to hash a message.
func ExampleKeccakHasher() {
	msg := &arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{},
		},
	}

	var hasher execution_engine.KeccakHasher
	hash := hasher.HashMessageWithMetadata(msg)

	fmt.Printf("Hash of the message: %s\n", hash)
	// Output: Hash of the message: 0xbc36789e7a1e281436464229828f817d6612f7b477d66591ff96a9e064bcc98a
}

// ExampleNewStdLibHasher demonstrates how to use the NewStdLibHasher
// to create a hasher that uses the standard library's crypto package.
//
// It hashes a message using the specified hashing algorithm (e.g., SHA256).
// This example uses SHA256, but you can replace it with any other supported
// algorithm.
//
// NOTE: The StdLibHasher is not expected to be used in production code, or
// really for testing, but is provided here for convenience and completeness.
func ExampleNewStdLibHasher() {
	msg := &arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{},
		},
	}

	hasher := execution_engine.NewStdLibHasher(crypto.SHA256)
	hash := hasher.HashMessageWithMetadata(msg)

	fmt.Printf("Hash of the message: %s\n", hash)
	// Output: Hash of the message: 0x6e340b9cffb37a989ca544e6bb780a2c78901d3fb33738768511a30617afa01d
}
