// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/conf"
)

type S3DAS struct {
	s3Config        conf.S3Config
	localDiskConfig LocalDiskDASConfig
	privKey         *blsSignatures.PrivateKey
	uploader        *manager.Uploader
	downloader      *manager.Downloader
}

func NewS3DAS(s3Config conf.S3Config, localDiskConfig LocalDiskDASConfig) (*S3DAS, error) {
	var privKey *blsSignatures.PrivateKey
	var err error
	if len(localDiskConfig.PrivKey) != 0 {
		privKey, err = DecodeBase64BLSPrivateKey([]byte(localDiskConfig.PrivKey))
		if err != nil {
			return nil, fmt.Errorf("'priv-key' was invalid: %w", err)
		}
	} else {
		_, privKey, err = ReadKeysFromFile(localDiskConfig.KeyDir)
		if err != nil {
			if os.IsNotExist(err) {
				if localDiskConfig.AllowGenerateKeys {
					_, privKey, err = GenerateAndStoreKeys(localDiskConfig.KeyDir)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, fmt.Errorf("Required BLS keypair did not exist at %s", localDiskConfig.KeyDir)
				}
			} else {
				return nil, err
			}
		}
	}
	client := s3.New(s3.Options{
		Region:      s3Config.Region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(s3Config.AccessKey, s3Config.SecretKey, "")),
	})
	uploader := manager.NewUploader(client)
	downloader := manager.NewDownloader(client)

	return &S3DAS{
		s3Config:        s3Config,
		privKey:         privKey,
		localDiskConfig: localDiskConfig,
		uploader:        uploader,
		downloader:      downloader,
	}, nil
}

func (das *S3DAS) Store(ctx context.Context, message []byte, timeout uint64) (c *arbstate.DataAvailabilityCertificate, err error) {
	c = &arbstate.DataAvailabilityCertificate{}
	copy(c.DataHash[:], crypto.Keccak256(message))

	c.Timeout = timeout
	c.SignersMask = 0 // The aggregator decides on the mask for each signer.

	fields := serializeSignableFields(*c)
	c.Sig, err = blsSignatures.SignMessage(*das.privKey, fields)
	if err != nil {
		return nil, err
	}

	path := base32.StdEncoding.EncodeToString(c.DataHash[:])
	log.Debug("Storing message at", "path", path)

	_, err = das.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(das.s3Config.Bucket),
		Key:    aws.String(path),
		Body:   bytes.NewReader(message),
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (das *S3DAS) Retrieve(ctx context.Context, certBytes []byte) ([]byte, error) {
	cert, err := arbstate.DeserializeDASCertFrom(bytes.NewReader(certBytes))
	if err != nil {
		return nil, err
	}

	path := base32.StdEncoding.EncodeToString(cert.DataHash[:])
	log.Debug("Retrieving message from", "path", path)

	originalMessageBuf := manager.NewWriteAtBuffer([]byte{})
	_, err = das.downloader.Download(ctx, originalMessageBuf, &s3.GetObjectInput{
		Bucket: aws.String(das.s3Config.Bucket),
		Key:    aws.String(path),
	})
	originalMessage := originalMessageBuf.Bytes()
	if err != nil {
		return nil, err
	}

	var originalMessageHash [32]byte
	copy(originalMessageHash[:], crypto.Keccak256(originalMessage))
	if originalMessageHash != cert.DataHash {
		return nil, errors.New("Retrieved message stored hash doesn't match calculated hash.")
	}

	// The cert passed in may have an aggregate signature, so we don't
	// check the signature against this DAS's public key here.

	return originalMessage, nil
}

func (das *S3DAS) String() string {
	return fmt.Sprintf("S3DAS{s3Config:%v, localDiskConfig:%v}", das.s3Config, das.localDiskConfig)
}
