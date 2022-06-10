// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"errors"
)

// Provides a registry of BLS public keys for accounts.
type ArbBLS struct {
	Address addr
}

var ErrKeyRegNotSupported = errors.New("BLS key registration is not currently supported")

// Deprecated -- equivalent to registerAltBN128
func (con ArbBLS) Register(c ctx, evm mech, x0, x1, y0, y1 huge) error {
	return ErrKeyRegNotSupported
}

// Deprecated -- equivalent to getAltBN128
func (con ArbBLS) GetPublicKey(c ctx, evm mech, address addr) (huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, ErrKeyRegNotSupported
}

// Associate an AltBN128 public key with the caller's address
func (con ArbBLS) RegisterAltBN128(c ctx, evm mech, x0, x1, y0, y1 huge) error {
	return ErrKeyRegNotSupported
}

// Get the AltBN128 public key associated with an address (revert if there isn't one)
func (con ArbBLS) GetAltBN128(c ctx, evm mech, address addr) (huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, ErrKeyRegNotSupported
}

// Associate a BLS 12-381 public key with the caller's address
func (con ArbBLS) RegisterBLS12381(c ctx, evm mech, keyBuf []byte) error {
	return ErrKeyRegNotSupported
}

// Get the BLS 12-381 public key associated with an address (revert if there isn't one)
func (con ArbBLS) GetBLS12381(c ctx, evm mech, address addr) ([]byte, error) {
	return nil, ErrKeyRegNotSupported
}
