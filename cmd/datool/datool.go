// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/das/dasrpc"
	flag "github.com/spf13/pflag"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		panic(fmt.Sprintf("Usage: datool [client|keygen] ..."))
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
	case "store":
		return startClientStore(args[1:])
	case "retrieve":
		return startClientRetrieve(args[1:])
	}
	return fmt.Errorf("datool client '%s' not supported, valid arguments are 'store' and 'retrieve'", args[0])
}

// datool client store

type ClientStoreConfig struct {
	URL                string        `koanf:"url"`
	Message            string        `koanf:"message"`
	DASRetentionPeriod time.Duration `koanf:"das-retention-period"`
	// TODO ECDSA private key to sign message with
}

func parseClientStoreConfig(args []string) (*ClientStoreConfig, error) {
	f := flag.NewFlagSet("client", flag.ContinueOnError)
	f.String("url", "", "URL of DAS server to connect to.")
	f.String("message", "", "Message to send.")
	f.Duration("das-retention-period", 24*time.Hour, "The period which DASes are requested to retain the stored batches.")

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
	cert, err := client.Store(ctx, []byte(config.Message), uint64(time.Now().Add(config.DASRetentionPeriod).Unix()))
	if err != nil {
		return err
	}

	serializedCert := das.Serialize(*cert)
	encodedCert := make([]byte, base64.StdEncoding.EncodedLen(len(serializedCert)))
	base64.StdEncoding.Encode(encodedCert, serializedCert)

	fmt.Printf("Base64 Encoded Cert: %s\n", string(encodedCert))

	return nil
}

// datool client retrieve

type ClientRetrieveConfig struct {
	URL  string `koanf:"url"`
	Cert string `koanf:"cert"`
}

func parseClientRetrieveConfig(args []string) (*ClientRetrieveConfig, error) {
	f := flag.NewFlagSet("client", flag.ContinueOnError)
	f.String("url", "", "URL of DAS server to connect to.")
	f.String("cert", "", "Base64 encodeded DAS certificate of message to retrieve.")

	k, err := util.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config ClientRetrieveConfig
	if err := util.EndCommonParse(k, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func startClientRetrieve(args []string) error {
	config, err := parseClientRetrieveConfig(args)
	if err != nil {
		return err
	}

	client, err := dasrpc.NewDASRPCClient(config.URL)
	if err != nil {
		return err
	}

	decodedCert := make([]byte, base64.StdEncoding.DecodedLen(len(config.Cert)))
	base64.StdEncoding.Decode(decodedCert, []byte(config.Cert))
	ctx := context.Background()
	message, err := client.Retrieve(ctx, decodedCert)
	if err != nil {
		return err
	}
	fmt.Printf("Message: %s\n", message)
	return nil
}

//

type KeyGenConfig struct {
}

func startKeyGen(args []string) error {
	return errors.New("Not implemented")
}

func parseKeyGenConfig(args []string) error {
	return nil
}
