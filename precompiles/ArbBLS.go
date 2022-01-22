//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import "github.com/offchainlabs/arbstate/arbos/blsSignatures"

type ArbBLS struct {
	Address addr
}

func (con ArbBLS) Register(c ctx, evm mech, x0, x1, y0, y1 huge) error {
	return con.RegisterAltBN128(c, evm, x0, x1, y0, y1)
}

func (con ArbBLS) GetPublicKey(c ctx, evm mech, address addr) (huge, huge, huge, huge, error) {
	return con.GetAltBN128(c, evm, address)
}

func (con ArbBLS) RegisterAltBN128(c ctx, evm mech, x0, x1, y0, y1 huge) error {
	return c.state.BLSTable().RegisterLegacyPublicKey(c.caller, x0, x1, y0, y1)
}

func (con ArbBLS) GetAltBN128(c ctx, evm mech, address addr) (huge, huge, huge, huge, error) {
	return c.state.BLSTable().GetLegacyPublicKey(address)
}

func (con ArbBLS) RegisterBLS12381(c ctx, evm mech, keyBuf []byte) error {
	key, err := blsSignatures.PublicKeyFromBytes(keyBuf, false)
	if err != nil {
		return err
	}
	return c.state.BLSTable().RegisterBLS12381PublicKey(c.caller, key)
}

func (con ArbBLS) GetBLS12381(c ctx, evm mech, address addr) ([]byte, error) {
	pubKey, err := c.state.BLSTable().GetBLS12381PublicKey(address)
	if err != nil {
		return nil, err
	}
	return blsSignatures.PublicKeyToBytes(pubKey), nil
}
