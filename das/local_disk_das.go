// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"

	flag "github.com/spf13/pflag"
)

var ErrDasKeysetNotFound = errors.New("no such keyset")

type StorageConfig struct {
	KeyDir                string               `koanf:"key-dir"`
	PrivKey               string               `koanf:"priv-key"`
	LocalConfig           LocalConfig          `koanf:"local"`
	S3Config              genericconf.S3Config `koanf:"s3"`
	RedisConfig           RedisConfig          `koanf:"redis"`
	BigCacheConfig        BigCacheConfig       `koanf:"big-cache"`
	AllowGenerateKeys     bool                 `koanf:"allow-generate-keys"`
	L1NodeURL             string               `koanf:"l1-node-url"`
	SequencerInboxAddress string               `koanf:"sequencer-inbox-address"`
	StorageType           string               `koanf:"storage-type"`
}

type LocalConfig struct {
	DataDir string `koanf:"data-dir"`
}

var DefaultLocalConfig = LocalConfig{
	DataDir: "",
}

func LocalConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".data-dir", DefaultLocalConfig.DataDir, "Local data directory")
}

func StorageConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".key-dir", "", fmt.Sprintf("The directory to read the bls keypair ('%s' and '%s') from", DefaultPubKeyFilename, DefaultPrivKeyFilename))
	f.String(prefix+".priv-key", "", "The base64 BLS private key to use for signing DAS certificates")
	LocalConfigAddOptions(prefix+".local", f)
	genericconf.S3ConfigAddOptions(prefix+".s3", f)
	RedisConfigAddOptions(prefix+".redis", f)
	BigCacheConfigAddOptions(prefix+".big-cache", f)
	f.Bool(prefix+".allow-generate-keys", false, "Allow the local disk DAS to generate its own keys in key-dir if they don't already exist")
	f.String(prefix+".l1-node-url", "", "URL of L1 Ethereum node")
	f.String(prefix+".sequencer-inbox-address", "", "L1 address of SequencerInbox contract")
}

type DAS struct {
	config         StorageConfig
	privKey        *blsSignatures.PrivateKey
	keysetHash     [32]byte
	keysetBytes    []byte
	storageService StorageService
	bpVerifier     *BatchPosterVerifier
}

func NewDAS(ctx context.Context, config StorageConfig) (*DAS, error) {
	storageService, err := NewStorageServiceFromStorageConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	if config.L1NodeURL == "none" {
		return NewDASWithSeqInboxCaller(ctx, config, nil, storageService)
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
		return NewDASWithSeqInboxCaller(ctx, config, nil, storageService)
	}
	return NewDASWithL1Info(ctx, config, l1client, *seqInboxAddress, storageService)
}

func NewDASWithL1Info(
	ctx context.Context,
	config StorageConfig,
	l1client arbutil.L1Interface,
	seqInboxAddress common.Address,
	storageService StorageService,
) (*DAS, error) {
	seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(seqInboxAddress, l1client)
	if err != nil {
		return nil, err
	}
	return NewDASWithSeqInboxCaller(ctx, config, seqInboxCaller, storageService)
}

func NewDASWithSeqInboxCaller(
	ctx context.Context,
	config StorageConfig,
	seqInboxCaller *bridgegen.SequencerInboxCaller,
	storageService StorageService,
) (*DAS, error) {
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
				if config.AllowGenerateKeys {
					_, privKey, err = GenerateAndStoreKeys(config.KeyDir)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, fmt.Errorf("Required BLS keypair did not exist at %s", config.KeyDir)
				}
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

	return &DAS{
		config:         config,
		privKey:        privKey,
		keysetHash:     ksHash,
		keysetBytes:    ksBuf.Bytes(),
		storageService: storageService,
		bpVerifier:     bpVerifier,
	}, nil
}

func NewStorageServiceFromStorageConfig(ctx context.Context, config StorageConfig) (StorageService, error) {
	var storageService StorageService
	var err error
	switch config.StorageType {
	case "", "files":
		storageService = NewLocalDiskStorageService(config.LocalConfig.DataDir)
	case "db":
		storageService, err = NewDBStorageService(ctx, config.LocalConfig.DataDir, false)
		if err != nil {
			return nil, err
		}
		go func() {
			<-ctx.Done()
			_ = storageService.Close(context.Background())
		}()
	case "s3":
		storageService, err = NewS3StorageService(config.S3Config)
		if err != nil {
			return nil, err
		}
	case "redis":
		s3StorageService, err := NewS3StorageService(config.S3Config)
		if err != nil {
			return nil, err
		}
		storageService, err = NewRedisStorageService(config.RedisConfig, s3StorageService)
		if err != nil {
			return nil, err
		}
	case "bigCache":
		s3StorageService, err := NewS3StorageService(config.S3Config)
		if err != nil {
			return nil, err
		}
		redisStorageService, err := NewRedisStorageService(config.RedisConfig, s3StorageService)
		if err != nil {
			return nil, err
		}
		storageService, err = NewBigCacheStorageService(config.BigCacheConfig, redisStorageService)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("Storage service type not recognized: " + config.StorageType)
	}
	return storageService, nil
}

func (d *DAS) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (c *arbstate.DataAvailabilityCertificate, err error) {
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

func (d *DAS) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	return d.storageService.GetByHash(ctx, hash)
}

func (d *DAS) KeysetFromHash(ctx context.Context, ksHash []byte) ([]byte, error) {
	if bytes.Equal(ksHash, d.keysetHash[:]) {
		return d.keysetBytes, nil
	}
	contents, err := d.GetByHash(ctx, ksHash)
	if err == nil {
		return contents, nil
	}
	return nil, ErrDasKeysetNotFound
}

func (d *DAS) CurrentKeysetBytes(ctx context.Context) ([]byte, error) {
	return d.keysetBytes, nil
}

func (d *DAS) String() string {
	return fmt.Sprintf("DAS{config:%v}", d.config)
}
