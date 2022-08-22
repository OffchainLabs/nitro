// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/cmd/genericconf"

	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/das/dastree"
	flag "github.com/spf13/pflag"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		panic("Usage: datool [client|keygen|generatehash] ...")
	}

	var err error
	switch strings.ToLower(args[1]) {
	case "client":
		err = startClient(args[2:])
	case "keygen":
		err = startKeyGen(args[2:])
	case "generatehash":
		err = generateHash(args[2])
	default:
		panic(fmt.Sprintf("Unknown tool '%s' specified, valid tools are 'client', 'keygen', 'generatehash'", args[1]))
	}
	if err != nil {
		panic(err)
	}
}

// datool client ...

func startClient(args []string) error {
	switch strings.ToLower(args[0]) {
	case "rpc":
		switch strings.ToLower(args[1]) {
		case "store":
			return startClientStore(args[2:])
		default:
			return fmt.Errorf("datool client rpc '%s' not supported, valid arguments are 'store'", args[1])

		}
	case "rest":
		switch strings.ToLower(args[1]) {
		case "getbyhash":
			return startRESTClientGetByHash(args[2:])
		default:
			return fmt.Errorf("datool client rest '%s' not supported, valid argument is 'getByHash'", args[1])
		}

	}
	return fmt.Errorf("datool client '%s' not supported, valid arguments are 'rpc' and 'rest'", args[0])

}

// datool client rpc store

type ClientStoreConfig struct {
	URL                   string                 `koanf:"url"`
	Message               string                 `koanf:"message"`
	RandomMessageSize     int                    `koanf:"random-message-size"`
	DASRetentionPeriod    time.Duration          `koanf:"das-retention-period"`
	SigningKey            string                 `koanf:"signing-key"`
	SigningWallet         string                 `koanf:"signing-wallet"`
	SigningWalletPassword string                 `koanf:"signing-wallet-password"`
	ConfConfig            genericconf.ConfConfig `koanf:"conf"`
}

func parseClientStoreConfig(args []string) (*ClientStoreConfig, error) {
	f := flag.NewFlagSet("datool client store", flag.ContinueOnError)
	f.String("url", "", "URL of DAS server to connect to")
	f.String("message", "", "message to send")
	f.Int("random-message-size", 0, "send a message of a specified number of random bytes")
	f.String("signing-key", "", "ecdsa private key to sign the message with, treated as a hex string if prefixed with 0x otherise treated as a file; if not specified the message is not signed")
	f.String("signing-wallet", "", "wallet containing ecdsa key to sign the message with")
	f.String("signing-wallet-password", genericconf.PASSWORD_NOT_SET, "password to unlock the wallet, if not specified the user is prompted for the password")
	f.Duration("das-retention-period", 24*time.Hour, "The period which DASes are requested to retain the stored batches.")
	genericconf.ConfConfigAddOptions("conf", f)

	k, err := util.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config ClientStoreConfig
	if err := util.EndCommonParse(k, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func startClientStore(args []string) error {
	config, err := parseClientStoreConfig(args)
	if err != nil {
		return err
	}

	client, err := das.NewDASRPCClient(config.URL)
	if err != nil {
		return err
	}

	var dasClient das.DataAvailabilityServiceWriter = client
	if config.SigningKey != "" {
		var privateKey *ecdsa.PrivateKey
		if config.SigningKey[:2] == "0x" {
			privateKey, err = crypto.HexToECDSA(config.SigningKey[2:])
			if err != nil {
				return err
			}
		} else {
			privateKey, err = crypto.LoadECDSA(config.SigningKey)
			if err != nil {
				return err
			}
		}
		signer := das.DasSignerFromPrivateKey(privateKey)

		dasClient, err = das.NewStoreSigningDAS(dasClient, signer)
		if err != nil {
			return err
		}
	} else if config.SigningWallet != "" {
		walletConf := &genericconf.WalletConfig{
			Pathname:      config.SigningWallet,
			PasswordImpl:  config.SigningWalletPassword,
			PrivateKey:    "",
			Account:       "",
			OnlyCreateKey: false,
		}
		_, signer, err := util.OpenWallet("datool", walletConf, nil)
		if err != nil {
			return err
		}
		dasClient, err = das.NewStoreSigningDAS(dasClient, signer)
		if err != nil {
			return err
		}
	}

	ctx := context.Background()
	var cert *arbstate.DataAvailabilityCertificate

	if config.RandomMessageSize > 0 {
		message := make([]byte, config.RandomMessageSize)
		_, err = rand.Read(message)
		if err != nil {
			return err
		}
		cert, err = dasClient.Store(ctx, message, uint64(time.Now().Add(config.DASRetentionPeriod).Unix()), []byte{})
	} else if len(config.Message) > 0 {
		cert, err = dasClient.Store(ctx, []byte(config.Message), uint64(time.Now().Add(config.DASRetentionPeriod).Unix()), []byte{})
	} else {
		return errors.New("--message or --random-message-size must be specified")
	}

	if err != nil {
		return err
	}

	serializedCert := das.Serialize(cert)
	fmt.Printf("Hex Encoded Cert: %s\n", string(hexutil.Encode(serializedCert)))
	fmt.Printf("Hex Encoded Data Hash: %s\n", string(hexutil.Encode(cert.DataHash[:])))

	return nil
}

// datool client rest getbyhash

type RESTClientGetByHashConfig struct {
	URL        string                 `koanf:"url"`
	DataHash   string                 `koanf:"data-hash"`
	ConfConfig genericconf.ConfConfig `koanf:"conf"`
}

func parseRESTClientGetByHashConfig(args []string) (*RESTClientGetByHashConfig, error) {
	f := flag.NewFlagSet("datool client retrieve", flag.ContinueOnError)
	f.String("url", "http://localhost:9877", "URL of DAS server to connect to.")
	f.String("data-hash", "", "hash of the message to retrieve, if starts with '0x' it's treated as hex encoded, otherwise base64 encoded")

	genericconf.ConfConfigAddOptions("conf", f)

	k, err := util.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config RESTClientGetByHashConfig
	if err := util.EndCommonParse(k, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func startRESTClientGetByHash(args []string) error {
	config, err := parseRESTClientGetByHashConfig(args)
	if err != nil {
		return err
	}

	client, err := das.NewRestfulDasClientFromURL(config.URL)
	if err != nil {
		return err
	}

	var decodedHash []byte
	if strings.HasPrefix(config.DataHash, "0x") {
		decodedHash, err = hexutil.Decode(config.DataHash)
		if err != nil {
			return err
		}
	} else {
		hashDecoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader([]byte(config.DataHash)))
		decodedHash, err = io.ReadAll(hashDecoder)
		if err != nil {
			return err
		}
	}

	ctx := context.Background()
	message, err := client.GetByHash(ctx, common.BytesToHash(decodedHash))
	if err != nil {
		return err
	}
	fmt.Printf("Message: %s\n", message)
	return nil
}

// das keygen

type KeyGenConfig struct {
	Dir        string
	ConfConfig genericconf.ConfConfig `koanf:"conf"`
	ECDSAMode  bool                   `koanf:"ecdsa"`
	WalletMode bool                   `koanf:"wallet"`
}

func parseKeyGenConfig(args []string) (*KeyGenConfig, error) {
	f := flag.NewFlagSet("datool keygen", flag.ContinueOnError)
	f.String("dir", "", "the directory to generate the keys in")
	f.Bool("ecdsa", false, "generate an ECDSA keypair instead of BLS")
	f.Bool("wallet", false, "generate the ECDSA keypair in a wallet file")
	genericconf.ConfConfigAddOptions("conf", f)

	k, err := util.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config KeyGenConfig
	if err := util.EndCommonParse(k, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func startKeyGen(args []string) error {
	config, err := parseKeyGenConfig(args)
	if err != nil {
		return err
	}

	if !config.ECDSAMode {
		_, _, err = das.GenerateAndStoreKeys(config.Dir)
		if err != nil {
			return err
		}
		return nil
	} else if !config.WalletMode {
		return das.GenerateAndStoreECDSAKeys(config.Dir)
	} else {
		walletConf := &genericconf.WalletConfig{
			Pathname:      config.Dir,
			PasswordImpl:  genericconf.PASSWORD_NOT_SET, // This causes a prompt for the password
			PrivateKey:    "",
			Account:       "",
			OnlyCreateKey: true,
		}
		_, _, err = util.OpenWallet("datool", walletConf, nil)
		if err != nil && strings.Contains(fmt.Sprint(err), "wallet key created") {
			return nil
		}
		return err
	}
}

func generateHash(message string) error {
	fmt.Printf("Hex Encoded Data Hash: %s\n", hexutil.Encode(dastree.HashBytes([]byte(message))))
	return nil
}
