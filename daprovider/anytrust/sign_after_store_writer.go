// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/daprovider/anytrust/tree"
	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
	"github.com/offchainlabs/nitro/util/pretty"
)

type KeyConfig struct {
	KeyDir  string `koanf:"key-dir"`
	PrivKey string `koanf:"priv-key"`
}

func (c *KeyConfig) BLSPrivKey() (blsSignatures.PrivateKey, error) {
	var privKeyBytes []byte
	if len(c.PrivKey) != 0 {
		privKeyBytes = []byte(c.PrivKey)
	} else if len(c.KeyDir) != 0 {
		var err error
		privKeyBytes, err = os.ReadFile(c.KeyDir + "/" + DefaultPrivKeyFilename)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("required BLS keypair did not exist at %s", c.KeyDir)
			}
			return nil, err
		}
	} else {
		return nil, errors.New("must specify PrivKey or KeyDir")
	}
	privKey, err := DecodeBase64BLSPrivateKey(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("'priv-key' was invalid: %w", err)
	}
	return privKey, nil
}

var DefaultKeyConfig = KeyConfig{}

func KeyConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".key-dir", DefaultKeyConfig.KeyDir, fmt.Sprintf("the directory to read the bls keypair ('%s' and '%s') from; if using any of the AnyTrust storage types exactly one of key-dir or priv-key must be specified", DefaultPubKeyFilename, DefaultPrivKeyFilename))
	f.String(prefix+".priv-key", DefaultKeyConfig.PrivKey, "the base64 BLS private key to use for signing AnyTrust certificates; if using any of the AnyTrust storage types exactly one of key-dir or priv-key must be specified")
}

// SignAfterStoreWriter provides AnyTrust signature functionality over a StorageService
// by adapting DataAvailabilityServiceWriter.Store(...) to StorageService.Put(...).
// There are two different signature functionalities it provides:
//
// 1) SignAfterStoreWriter.Store(...) assembles the returned hash into a
// DataAvailabilityCertificate and signs it with its BLS private key.
type SignAfterStoreWriter struct {
	privKey        blsSignatures.PrivateKey
	pubKey         *blsSignatures.PublicKey
	keysetHash     [32]byte
	keysetBytes    []byte
	storageService StorageService
}

func NewSignAfterStoreWriter(ctx context.Context, config Config, storageService StorageService) (*SignAfterStoreWriter, error) {
	privKey, err := config.Key.BLSPrivKey()
	if err != nil {
		return nil, err
	}

	publicKey, err := blsSignatures.PublicKeyFromPrivateKey(privKey)
	if err != nil {
		return nil, err
	}
	log.Info("AnyTrust public key used for signing", "key", hexutil.Encode(blsSignatures.PublicKeyToBytes(publicKey)))

	keyset := &anytrustutil.DataAvailabilityKeyset{
		AssumedHonest: 1,
		PubKeys:       []blsSignatures.PublicKey{publicKey},
	}
	ksBuf := bytes.NewBuffer([]byte{})
	if err := keyset.Serialize(ksBuf); err != nil {
		return nil, err
	}
	ksHash, err := keyset.Hash()
	if err != nil {
		return nil, err
	}

	return &SignAfterStoreWriter{
		privKey:        privKey,
		pubKey:         &publicKey,
		keysetHash:     ksHash,
		keysetBytes:    ksBuf.Bytes(),
		storageService: storageService,
	}, nil
}

func (d *SignAfterStoreWriter) Store(ctx context.Context, message []byte, timeout uint64) (c *anytrustutil.DataAvailabilityCertificate, err error) {
	// #nosec G115
	log.Trace("anytrust.SignAfterStoreWriter.Store", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0), "this", d)
	c = &anytrustutil.DataAvailabilityCertificate{
		Timeout:     timeout,
		DataHash:    tree.Hash(message),
		Version:     1,
		SignersMask: 1, // The aggregator will override this if we're part of a committee.
	}

	fields := c.SerializeSignableFields()
	c.Sig, err = blsSignatures.SignMessage(d.privKey, fields)
	if err != nil {
		return nil, err
	}

	err = d.storageService.Put(ctx, message, timeout)
	if err != nil {
		return nil, err
	}
	err = d.storageService.Sync(ctx)
	if err != nil {
		return nil, err
	}

	c.KeysetHash = d.keysetHash

	return c, nil
}

func (d *SignAfterStoreWriter) String() string {
	return fmt.Sprintf("SignAfterStoreWriter{%v}", hexutil.Encode(blsSignatures.PublicKeyToBytes(*d.pubKey)))
}
