// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/pretty"

	flag "github.com/spf13/pflag"
)

var ErrDasKeysetNotFound = errors.New("no such keyset")

type KeyConfig struct {
	KeyDir  string `koanf:"key-dir"`
	PrivKey string `koanf:"priv-key"`
}

var DefaultKeyConfig = KeyConfig{}

func KeyConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".key-dir", DefaultKeyConfig.KeyDir, fmt.Sprintf("the directory to read the bls keypair ('%s' and '%s') from; if using any of the DAS storage types exactly one of key-dir or priv-key must be specified", DefaultPubKeyFilename, DefaultPrivKeyFilename))
	f.String(prefix+".priv-key", DefaultKeyConfig.PrivKey, "the base64 BLS private key to use for signing DAS certificates; if using any of the DAS storage types exactly one of key-dir or priv-key must be specified")
}

// Provides DAS signature functionality over a StorageService by adapting
// DataAvailabilityService.Store(...) to StorageService.Put(...).
// There are two different signature functionalities it provides:
//
// 1) SignAfterStoreDAS.Store(...) assembles the returned hash into a
// DataAvailabilityCertificate and signs it with its BLS private key.
//
// 2) If Sequencer Inbox contract details are provided when a SignAfterStoreDAS is
// constructed, calls to Store(...) will try to verify the passed-in data's signature
// is from the batch poster. If the contract details are not provided, then the
// signature is not checked, which is useful for testing.
type SignAfterStoreDAS struct {
	config         KeyConfig
	privKey        *blsSignatures.PrivateKey
	keysetHash     [32]byte
	keysetBytes    []byte
	storageService StorageService
	bpVerifier     *BatchPosterVerifier
}

func NewSignAfterStoreDAS(ctx context.Context, config DataAvailabilityConfig, storageService StorageService) (*SignAfterStoreDAS, error) {
	if config.L1NodeURL == "none" {
		return NewSignAfterStoreDASWithSeqInboxCaller(ctx, config.KeyConfig, nil, storageService)
	}
	l1client, err := ethclient.Dial(config.L1NodeURL)
	if err != nil {
		return nil, err
	}
	seqInboxAddress, err := OptionalAddressFromString(config.SequencerInboxAddress)
	if err != nil {
		return nil, err
	}
	if seqInboxAddress == nil {
		return NewSignAfterStoreDASWithSeqInboxCaller(ctx, config.KeyConfig, nil, storageService)
	}

	seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(*seqInboxAddress, l1client)
	if err != nil {
		return nil, err
	}
	return NewSignAfterStoreDASWithSeqInboxCaller(ctx, config.KeyConfig, seqInboxCaller, storageService)
}

func NewSignAfterStoreDASWithSeqInboxCaller(
	ctx context.Context,
	config KeyConfig,
	seqInboxCaller *bridgegen.SequencerInboxCaller,
	storageService StorageService,
) (*SignAfterStoreDAS, error) {
	var privKey *blsSignatures.PrivateKey
	var err error
	if len(config.PrivKey) != 0 {
		privKey, err = DecodeBase64BLSPrivateKey([]byte(config.PrivKey))
		if err != nil {
			return nil, fmt.Errorf("'priv-key' was invalid: %w", err)
		}
	} else {
		_, privKey, err = ReadKeysFromFile(config.KeyDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("Required BLS keypair did not exist at %s", config.KeyDir)
			} else {
				return nil, err
			}
		}
	}

	publicKey, err := blsSignatures.PublicKeyFromPrivateKey(*privKey)
	if err != nil {
		return nil, err
	}

	keyset := &arbstate.DataAvailabilityKeyset{
		AssumedHonest: 1,
		PubKeys:       []blsSignatures.PublicKey{publicKey},
	}
	ksBuf := bytes.NewBuffer([]byte{})
	if err := keyset.Serialize(ksBuf); err != nil {
		return nil, err
	}
	ksHashBuf, err := keyset.Hash()
	if err != nil {
		return nil, err
	}
	var ksHash [32]byte
	copy(ksHash[:], ksHashBuf)

	var bpVerifier *BatchPosterVerifier
	if seqInboxCaller != nil {
		bpVerifier = NewBatchPosterVerifier(seqInboxCaller)
	}

	return &SignAfterStoreDAS{
		config:         config,
		privKey:        privKey,
		keysetHash:     ksHash,
		keysetBytes:    ksBuf.Bytes(),
		storageService: storageService,
		bpVerifier:     bpVerifier,
	}, nil
}

func (d *SignAfterStoreDAS) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (c *arbstate.DataAvailabilityCertificate, err error) {
	log.Trace("das.SignAfterStoreDAS.Store", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0), "sig", pretty.FirstFewBytes(sig), "this", d)
	if d.bpVerifier != nil {
		actualSigner, err := DasRecoverSigner(message, timeout, sig)
		if err != nil {
			return nil, err
		}
		isBatchPoster, err := d.bpVerifier.IsBatchPoster(ctx, actualSigner)
		if err != nil {
			return nil, err
		}
		if !isBatchPoster {
			return nil, errors.New("store request not properly signed")
		}
	}

	c = &arbstate.DataAvailabilityCertificate{}
	copy(c.DataHash[:], crypto.Keccak256(message))

	c.Timeout = timeout
	c.SignersMask = 1 // The aggregator will override this if we're part of a committee.

	fields := c.SerializeSignableFields()
	c.Sig, err = blsSignatures.SignMessage(*d.privKey, fields)
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

func (d *SignAfterStoreDAS) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	return d.storageService.GetByHash(ctx, hash)
}

func (d *SignAfterStoreDAS) String() string {
	return fmt.Sprintf("SignAfterStoreDAS{config:%v}", d.config)
}
