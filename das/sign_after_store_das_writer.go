// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/contracts"
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
		return nil, errors.Wrap(err, "'priv-key' was invalid")
	}
	return privKey, nil
}

var DefaultKeyConfig = KeyConfig{}

func KeyConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".key-dir", DefaultKeyConfig.KeyDir, fmt.Sprintf("the directory to read the bls keypair ('%s' and '%s') from; if using any of the DAS storage types exactly one of key-dir or priv-key must be specified", DefaultPubKeyFilename, DefaultPrivKeyFilename))
	f.String(prefix+".priv-key", DefaultKeyConfig.PrivKey, "the base64 BLS private key to use for signing DAS certificates; if using any of the DAS storage types exactly one of key-dir or priv-key must be specified")
}

// SignAfterStoreDASWriter provides DAS signature functionality over a StorageService
// by adapting DataAvailabilityServiceWriter.Store(...) to StorageService.Put(...).
// There are two different signature functionalities it provides:
//
// 1) SignAfterStoreDASWriter.Store(...) assembles the returned hash into a
// DataAvailabilityCertificate and signs it with its BLS private key.
//
// 2) If Sequencer Inbox contract details are provided when a SignAfterStoreDASWriter is
// constructed, calls to Store(...) will try to verify the passed-in data's signature
// is from the batch poster. If the contract details are not provided, then the
// signature is not checked, which is useful for testing.
type SignAfterStoreDASWriter struct {
	privKey        blsSignatures.PrivateKey
	pubKey         *blsSignatures.PublicKey
	keysetHash     [32]byte
	keysetBytes    []byte
	storageService StorageService
	bpVerifier     *contracts.BatchPosterVerifier

	// Extra batch poster verifier, for local installations to have their
	// own way of testing Stores.
	extraBpVerifier func(message []byte, timeout uint64, sig []byte) bool
}

func NewSignAfterStoreDASWriter(ctx context.Context, config DataAvailabilityConfig, storageService StorageService) (*SignAfterStoreDASWriter, error) {
	privKey, err := config.KeyConfig.BLSPrivKey()
	if err != nil {
		return nil, err
	}
	if config.L1NodeURL == "none" {
		return NewSignAfterStoreDASWriterWithSeqInboxCaller(privKey, nil, storageService, config.ExtraSignatureCheckingPublicKey)
	}
	l1client, err := GetL1Client(ctx, config.L1ConnectionAttempts, config.L1NodeURL)
	if err != nil {
		return nil, err
	}
	seqInboxAddress, err := OptionalAddressFromString(config.SequencerInboxAddress)
	if err != nil {
		return nil, err
	}
	if seqInboxAddress == nil {
		return NewSignAfterStoreDASWriterWithSeqInboxCaller(privKey, nil, storageService, config.ExtraSignatureCheckingPublicKey)
	}

	seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(*seqInboxAddress, l1client)
	if err != nil {
		return nil, err
	}
	return NewSignAfterStoreDASWriterWithSeqInboxCaller(privKey, seqInboxCaller, storageService, config.ExtraSignatureCheckingPublicKey)
}

func NewSignAfterStoreDASWriterWithSeqInboxCaller(
	privKey blsSignatures.PrivateKey,
	seqInboxCaller *bridgegen.SequencerInboxCaller,
	storageService StorageService,
	extraSignatureCheckingPublicKey string,
) (*SignAfterStoreDASWriter, error) {
	publicKey, err := blsSignatures.PublicKeyFromPrivateKey(privKey)
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
	ksHash, err := keyset.Hash()
	if err != nil {
		return nil, err
	}

	var bpVerifier *contracts.BatchPosterVerifier
	if seqInboxCaller != nil {
		bpVerifier = contracts.NewBatchPosterVerifier(seqInboxCaller)
	}

	var extraBpVerifier func(message []byte, timeout uint64, sig []byte) bool
	if extraSignatureCheckingPublicKey != "" {
		var pubkey []byte
		if extraSignatureCheckingPublicKey[:2] == "0x" {
			pubkey, err = hex.DecodeString(extraSignatureCheckingPublicKey[2:])
			if err != nil {
				return nil, err
			}
		} else {
			pubkeyEncoded, err := os.ReadFile(extraSignatureCheckingPublicKey)
			if err != nil {
				return nil, err
			}
			pubkey, err = hex.DecodeString(string(pubkeyEncoded))
			if err != nil {
				return nil, err
			}
		}
		extraBpVerifier = func(message []byte, timeout uint64, sig []byte) bool {
			if len(sig) >= 64 {
				return crypto.VerifySignature(pubkey, dasStoreHash(message, timeout), sig[:64])
			}
			return false
		}
	}

	return &SignAfterStoreDASWriter{
		privKey:         privKey,
		pubKey:          &publicKey,
		keysetHash:      ksHash,
		keysetBytes:     ksBuf.Bytes(),
		storageService:  storageService,
		bpVerifier:      bpVerifier,
		extraBpVerifier: extraBpVerifier,
	}, nil
}

func (d *SignAfterStoreDASWriter) Store(
	ctx context.Context, message []byte, timeout uint64, sig []byte,
) (c *arbstate.DataAvailabilityCertificate, err error) {
	log.Trace("das.SignAfterStoreDASWriter.Store", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0), "sig", pretty.FirstFewBytes(sig), "this", d)
	var verified bool
	if d.extraBpVerifier != nil {
		verified = d.extraBpVerifier(message, timeout, sig)
	}

	if !verified && d.bpVerifier != nil {
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

	c = &arbstate.DataAvailabilityCertificate{
		Timeout:     timeout,
		DataHash:    dastree.Hash(message),
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

func (d *SignAfterStoreDASWriter) String() string {
	return fmt.Sprintf("SignAfterStoreDASWriter{%v}", hexutil.Encode(blsSignatures.PublicKeyToBytes(*d.pubKey)))
}
