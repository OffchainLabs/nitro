// Package chain provides mocks and implementations to simulate a simple
// Espresso Chain environment for testing purposes.
//
// The point of this package is to provide a controllable environment
// representation of the Espresso Chain.  As a result, it not only provides
// the the Mock Espresso Chain, with the ability for the user to control when
// the next block is produced / advanced, but it also provides several utilities
// for simulating errant behavior.
//
// Additionally, it provides an UnimplementedEspressoClient, which can be used
// to create other mocks that implement the EspressoClient interface without
// needing to implement all methods.
package chain
