// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"github.com/offchainlabs/nitro/blsSignatures"
)

// Provides a registry of BLS public keys for accounts.
type ArbBLS struct {
	Address addr
}

// Deprecated -- equivalent to registerAltBN128
func (con ArbBLS) Register(c ctx, evm mech, x0, x1, y0, y1 huge) error {
	return con.RegisterAltBN128(c, evm, x0, x1, y0, y1)
}

// Deprecated -- equivalent to getAltBN128
func (con ArbBLS) GetPublicKey(c ctx, evm mech, address addr) (huge, huge, huge, huge, error) {
	return con.GetAltBN128(c, evm, address)
}

// Associate an AltBN128 public key with the caller's address
func (con ArbBLS) RegisterAltBN128(c ctx, evm mech, x0, x1, y0, y1 huge) error {
	return c.State.BLSTable().RegisterLegacyPublicKey(c.caller, x0, x1, y0, y1)
}

// Get the AltBN128 public key associated with an address (revert if there isn't one)
func (con ArbBLS) GetAltBN128(c ctx, evm mech, address addr) (huge, huge, huge, huge, error) {
	return c.State.BLSTable().GetLegacyPublicKey(address)
}

// Associate a BLS 12-381 public key with the caller's address
func (con ArbBLS) RegisterBLS12381(c ctx, evm mech, keyBuf []byte) error {
	key, err := blsSignatures.PublicKeyFromBytes(keyBuf, false)
	if err != nil {
		return err
	}
	return c.State.BLSTable().RegisterBLS12381PublicKey(c.caller, key)
}

// Get the BLS 12-381 public key associated with an address (revert if there isn't one)
func (con ArbBLS) GetBLS12381(c ctx, evm mech, address addr) ([]byte, error) {
	pubKey, err := c.State.BLSTable().GetBLS12381PublicKey(address)
	if err != nil {
		return nil, err
	}
	return blsSignatures.PublicKeyToBytes(pubKey), nil
}
