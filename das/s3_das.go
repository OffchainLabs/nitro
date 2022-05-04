// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/conf"
)

type S3DataAvailabilityService struct {
	s3Config   conf.S3Config
	pubKey     *blsSignatures.PublicKey
	privKey    blsSignatures.PrivateKey
	uploader   *manager.Uploader
	downloader *manager.Downloader
	signerMask uint64
}

func readKeysFromS3(s3Config conf.S3Config, downloader *manager.Downloader) (*blsSignatures.PublicKey, blsSignatures.PrivateKey, error) {
	pubKeyBuf := manager.NewWriteAtBuffer([]byte{})
	_, err := downloader.Download(context.TODO(), pubKeyBuf, &s3.GetObjectInput{
		Bucket: aws.String(s3Config.Bucket),
		Key:    aws.String("pubkey"),
	})
	if err != nil {
		return nil, nil, err
	}

	privKeyBuf := manager.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(context.TODO(), privKeyBuf, &s3.GetObjectInput{
		Bucket: aws.String(s3Config.Bucket),
		Key:    aws.String("privkey"),
	})
	if err != nil {
		return nil, nil, err
	}

	pubKey, err := blsSignatures.PublicKeyFromBytes(pubKeyBuf.Bytes(), true)
	if err != nil {
		return nil, nil, err
	}
	privKey, err := blsSignatures.PrivateKeyFromBytes(privKeyBuf.Bytes())
	if err != nil {
		return nil, nil, err
	}
	return &pubKey, privKey, nil
}

func generateAndStoreKeysInS3(s3Config conf.S3Config, uploader *manager.Uploader) (*blsSignatures.PublicKey, blsSignatures.PrivateKey, error) {
	pubKey, privKey, err := blsSignatures.GenerateKeys()
	if err != nil {
		return nil, nil, err
	}

	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s3Config.Bucket),
		Key:    aws.String("pubkey"),
		Body:   bytes.NewReader(blsSignatures.PublicKeyToBytes(pubKey)),
	})
	if err != nil {
		return nil, nil, err
	}
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s3Config.Bucket),
		Key:    aws.String("privkey"),
		Body:   bytes.NewReader(blsSignatures.PrivateKeyToBytes(privKey)),
	})
	if err != nil {
		return nil, nil, err
	}
	return &pubKey, privKey, nil
}

func NewS3DataAvailabilityService(s3Config conf.S3Config) (*S3DataAvailabilityService, error) {
	client := s3.New(s3.Options{
		Region:      s3Config.Region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(s3Config.AccessKey, s3Config.SecretKey, "")),
	})
	uploader := manager.NewUploader(client)
	downloader := manager.NewDownloader(client)

	pubKey, privKey, err := readKeysFromS3(s3Config, downloader)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			pubKey, privKey, err = generateAndStoreKeysInS3(s3Config, uploader)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &S3DataAvailabilityService{
		s3Config:   s3Config,
		pubKey:     pubKey,
		privKey:    privKey,
		uploader:   uploader,
		downloader: downloader,
	}, nil
}

func (das *S3DataAvailabilityService) Store(ctx context.Context, message []byte, timeout uint64) (c *arbstate.DataAvailabilityCertificate, err error) {
	c = &arbstate.DataAvailabilityCertificate{}
	copy(c.DataHash[:], crypto.Keccak256(message))

	c.Timeout = timeout
	c.SignersMask = das.signerMask

	fields := serializeSignableFields(*c)
	c.Sig, err = blsSignatures.SignMessage(das.privKey, fields)
	if err != nil {
		return nil, err
	}

	path := base32.StdEncoding.EncodeToString(c.DataHash[:])
	log.Debug("Storing message at", "path", path)

	_, err = das.uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(das.s3Config.Bucket),
		Key:    aws.String(path),
		Body:   bytes.NewReader(message),
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (das *S3DataAvailabilityService) Retrieve(ctx context.Context, certBytes []byte) ([]byte, error) {
	cert, _, err := arbstate.DeserializeDASCertFrom(certBytes)
	if err != nil {
		return nil, err
	}

	path := base32.StdEncoding.EncodeToString(cert.DataHash[:])
	log.Debug("Retrieving message from", "path", path)

	originalMessageBuf := manager.NewWriteAtBuffer([]byte{})
	_, err = das.downloader.Download(context.TODO(), originalMessageBuf, &s3.GetObjectInput{
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

	signedBlob := serializeSignableFields(*cert)
	sigMatch, err := blsSignatures.VerifySignature(cert.Sig, signedBlob, *das.pubKey)
	if err != nil {
		return nil, err
	}
	if !sigMatch {
		return nil, errors.New("Signature of data in cert passed in doesn't match")
	}

	return originalMessage, nil
}
