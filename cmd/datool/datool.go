// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/offchainlabs/nitro/cmd/genericconf"

	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/das/dasrpc"
	flag "github.com/spf13/pflag"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		panic("Usage: datool [client|keygen] ...")
	}

	var err error
	switch strings.ToLower(args[1]) {
	case "client":
		err = startClient(args[2:])
	case "keygen":
		err = startKeyGen(args[2:])
	default:
		panic(fmt.Sprintf("Unknown tool '%s' specified, valid tools are 'client', 'keygen'", args[1]))
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
		case "getbyhash":
			return startRPCClientGetByHash(args[2:])
		default:
			return fmt.Errorf("datool client rpc '%s' not supported, valid arguments are 'store' and 'getByHash'", args[1])

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
	URL                string        `koanf:"url"`
	Message            string        `koanf:"message"`
	DASRetentionPeriod time.Duration `koanf:"das-retention-period"`
	// TODO ECDSA private key to sign message with
	ConfConfig genericconf.ConfConfig `koanf:"conf"`
}

func parseClientStoreConfig(args []string) (*ClientStoreConfig, error) {
	f := flag.NewFlagSet("datool client store", flag.ContinueOnError)
	f.String("url", "", "URL of DAS server to connect to.")
	f.String("message", "", "Message to send.")
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

	client, err := dasrpc.NewDASRPCClient(config.URL)
	if err != nil {
		return err
	}

	ctx := context.Background()
	cert, err := client.Store(ctx, []byte(config.Message), uint64(time.Now().Add(config.DASRetentionPeriod).Unix()), []byte{})
	if err != nil {
		return err
	}

	serializedCert := das.Serialize(cert)
	encodedCert := make([]byte, base64.StdEncoding.EncodedLen(len(serializedCert)))
	base64.StdEncoding.Encode(encodedCert, serializedCert)
	fmt.Printf("Base64 Encoded Cert: %s\n", string(encodedCert))

	encodedDataHash := make([]byte, base64.StdEncoding.EncodedLen(len(cert.DataHash)))
	base64.StdEncoding.Encode(encodedDataHash, cert.DataHash[:])
	fmt.Printf("Base64 Encoded Data Hash: %s\n", string(encodedDataHash))

	return nil
}

// datool client rpc getbyhash
type RPCClientGetByHashConfig struct {
	URL        string                 `koanf:"url"`
	DataHash   string                 `koanf:"data-hash"`
	ConfConfig genericconf.ConfConfig `koanf:"conf"`
}

func parseRPCClientGetByHashConfig(args []string) (*RPCClientGetByHashConfig, error) {
	f := flag.NewFlagSet("datool client retrieve", flag.ContinueOnError)
	f.String("url", "http://localhost:9876", "URL of DAS server to connect to.")
	f.String("data-hash", "", "hash of the message to retrieve, if starts with '0x' it's treated as hex encoded, otherwise base64 encoded")

	genericconf.ConfConfigAddOptions("conf", f)

	k, err := util.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config RPCClientGetByHashConfig
	if err := util.EndCommonParse(k, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func startRPCClientGetByHash(args []string) error {
	config, err := parseRPCClientGetByHashConfig(args)
	if err != nil {
		return err
	}

	client, err := dasrpc.NewDASRPCClient(config.URL)
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
		decodedHash, err = ioutil.ReadAll(hashDecoder)
		if err != nil {
			return err
		}
	}

	ctx := context.Background()
	message, err := client.GetByHash(ctx, decodedHash)
	if err != nil {
		return err
	}
	fmt.Printf("Message: %s\n", message)
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
		decodedHash, err = ioutil.ReadAll(hashDecoder)
		if err != nil {
			return err
		}
	}

	ctx := context.Background()
	message, err := client.GetByHash(ctx, decodedHash)
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
}

func parseKeyGenConfig(args []string) (*KeyGenConfig, error) {
	f := flag.NewFlagSet("datool keygen", flag.ContinueOnError)
	f.String("dir", "", "The directory to generate the keys in")
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

	_, _, err = das.GenerateAndStoreKeys(config.Dir)
	if err != nil {
		return err
	}
	return nil
}
