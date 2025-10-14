// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// This is a command line tool for testing beacon/blobs and blob_sidecars endpoints.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"

	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/headerreader"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: blobtool [fetch] ...")
		os.Exit(1)
	}

	var err error
	switch strings.ToLower(args[1]) {
	case "fetch":
		err = fetchBlobs(args[2:])
	default:
		err = fmt.Errorf("unknown command '%s', valid commands are: fetch", args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

type FetchConfig struct {
	BeaconURL         string   `koanf:"beacon-url"`
	Slot              uint64   `koanf:"slot"`
	VersionedHashes   []string `koanf:"versioned-hashes"`
	UseLegacyEndpoint bool     `koanf:"use-legacy-endpoint"`
	CompareEndpoints  bool     `koanf:"compare-endpoints"`
}

func parseFetchConfig(args []string) (*FetchConfig, error) {
	f := flag.NewFlagSet("blobtool fetch", flag.ContinueOnError)
	f.String("beacon-url", "", "Beacon Chain RPC URL. For example with --beacon-url=http://localhost, an RPC call will be made to http://localhost/eth/v1/beacon/blobs")
	f.Uint64("slot", 0, "Beacon chain slot number to fetch blobs from")
	f.StringSlice("versioned-hashes", []string{}, "Comma-separated list of versioned hashes to fetch (optional - fetches all if not provided)")
	f.Bool("use-legacy-endpoint", false, "Use the legacy blob_sidecars endpoint")
	f.Bool("compare-endpoints", false, "Fetch using both endpoints and compare results")

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config FetchConfig
	if err := confighelpers.EndCommonParse(k, &config); err != nil {
		return nil, err
	}

	if config.BeaconURL == "" {
		return nil, fmt.Errorf("--beacon-url is required")
	}
	if config.Slot == 0 {
		return nil, fmt.Errorf("--slot is required")
	}

	return &config, nil
}

func fetchBlobs(args []string) error {
	config, err := parseFetchConfig(args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	versionedHashes := make([]common.Hash, len(config.VersionedHashes))
	for i, hashStr := range config.VersionedHashes {
		if !common.IsHexAddress(hashStr) && len(hashStr) != 66 {
			return fmt.Errorf("invalid versioned hash at index %d: %s", i, hashStr)
		}
		versionedHashes[i] = common.HexToHash(hashStr)
	}

	if config.UseLegacyEndpoint && len(versionedHashes) == 0 {
		return fmt.Errorf("--versioned-hashes is required when using --use-legacy-endpoint")
	}

	if config.CompareEndpoints {
		if len(versionedHashes) == 0 {
			return fmt.Errorf("--versioned-hashes is required when using --compare-endpoints")
		}
		return compareEndpoints(ctx, config, versionedHashes)
	}

	blobClientConfig := headerreader.BlobClientConfig{
		BeaconUrl:         config.BeaconURL,
		UseLegacyEndpoint: config.UseLegacyEndpoint,
	}

	blobClient, err := headerreader.NewBlobClient(blobClientConfig, nil)
	if err != nil {
		return fmt.Errorf("failed to create blob client: %w", err)
	}

	if err := blobClient.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize blob client: %w", err)
	}

	endpointType := "new blobs"
	if config.UseLegacyEndpoint {
		endpointType = "legacy blob_sidecars"
	}

	if len(versionedHashes) > 0 {
		fmt.Printf("Fetching %d blobs for slot %d using %s endpoint...\n", len(versionedHashes), config.Slot, endpointType)
	} else {
		fmt.Printf("Fetching all blobs for slot %d using %s endpoint...\n", config.Slot, endpointType)
	}

	startTime := time.Now()
	fetchedBlobs, err := blobClient.GetBlobsBySlot(ctx, config.Slot, versionedHashes)
	if err != nil {
		return fmt.Errorf("failed to fetch blobs: %w", err)
	}
	duration := time.Since(startTime)

	fmt.Printf("Successfully fetched %d blobs in %v\n", len(fetchedBlobs), duration)

	for i, blob := range fetchedBlobs {
		_, hashes, err := blobs.ComputeCommitmentsAndHashes([]kzg4844.Blob{blob})
		if err != nil {
			return fmt.Errorf("failed to compute commitment for blob %d: %w", i, err)
		}
		if len(versionedHashes) > 0 {
			fmt.Printf("Blob %d: versioned_hash=%s (computed=%s), size=%d bytes\n", i, versionedHashes[i].Hex(), hashes[0].Hex(), len(blob))
		} else {
			fmt.Printf("Blob %d: versioned_hash=%s, size=%d bytes\n", i, hashes[0].Hex(), len(blob))
		}
	}

	return nil
}

func compareEndpoints(ctx context.Context, config *FetchConfig, versionedHashes []common.Hash) error {
	fmt.Println("Comparing legacy blob_sidecars and new blobs endpoints...")
	fmt.Println()

	legacyConfig := headerreader.BlobClientConfig{
		BeaconUrl:         config.BeaconURL,
		UseLegacyEndpoint: true,
	}
	legacyClient, err := headerreader.NewBlobClient(legacyConfig, nil)
	if err != nil {
		return fmt.Errorf("failed to create legacy blob client: %w", err)
	}
	if err := legacyClient.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize legacy blob client: %w", err)
	}

	fmt.Println("Fetching with legacy blob_sidecars endpoint...")
	legacyStart := time.Now()
	legacyBlobs, err := legacyClient.GetBlobsBySlot(ctx, config.Slot, versionedHashes)
	legacyDuration := time.Since(legacyStart)
	if err != nil {
		return fmt.Errorf("failed to fetch blobs with legacy endpoint: %w", err)
	}
	fmt.Printf("✓ Legacy endpoint: fetched %d blobs in %v\n", len(legacyBlobs), legacyDuration)
	fmt.Println()

	newConfig := headerreader.BlobClientConfig{
		BeaconUrl:         config.BeaconURL,
		UseLegacyEndpoint: false,
	}
	newClient, err := headerreader.NewBlobClient(newConfig, nil)
	if err != nil {
		return fmt.Errorf("failed to create new blob client: %w", err)
	}
	if err := newClient.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize new blob client: %w", err)
	}

	fmt.Println("Fetching with new blobs endpoint...")
	newStart := time.Now()
	newBlobs, err := newClient.GetBlobsBySlot(ctx, config.Slot, versionedHashes)
	newDuration := time.Since(newStart)
	if err != nil {
		return fmt.Errorf("failed to fetch blobs with new endpoint: %w", err)
	}
	fmt.Printf("✓ New endpoint: fetched %d blobs in %v\n", len(newBlobs), newDuration)
	fmt.Println()

	if len(legacyBlobs) != len(newBlobs) {
		return fmt.Errorf("blob count mismatch: legacy=%d, new=%d", len(legacyBlobs), len(newBlobs))
	}

	fmt.Println("Comparing blob data...")
	for i := range legacyBlobs {
		if legacyBlobs[i] != newBlobs[i] {
			return fmt.Errorf("blob %d data mismatch", i)
		}
		_, hashes, err := blobs.ComputeCommitmentsAndHashes([]kzg4844.Blob{legacyBlobs[i]})
		if err != nil {
			return fmt.Errorf("failed to compute hash for blob %d: %w", i, err)
		}
		fmt.Printf("  Blob %d: ✓ identical (%s)\n", i, hashes[0].Hex())
	}

	fmt.Println()
	fmt.Printf("Performance comparison:\n")
	fmt.Printf("  Legacy endpoint: %v\n", legacyDuration)
	fmt.Printf("  New endpoint:    %v\n", newDuration)
	if newDuration < legacyDuration {
		improvement := float64(legacyDuration-newDuration) / float64(legacyDuration) * 100
		fmt.Printf("  New endpoint is %.1f%% faster\n", improvement)
	} else {
		slower := float64(newDuration-legacyDuration) / float64(legacyDuration) * 100
		fmt.Printf("  New endpoint is %.1f%% slower\n", slower)
	}

	return nil
}
