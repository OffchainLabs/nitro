package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestDownloadInit(t *testing.T) {
	const (
		archiveName = "random_data.tar.gz"
		dataSize    = 1024 * 1024
		filePerm    = 0600
	)

	// Create archive with random data
	serverDir := t.TempDir()
	data := testhelpers.RandomSlice(dataSize)
	checksumBytes := sha256.Sum256(data)
	checksum := hex.EncodeToString(checksumBytes[:])

	// Write archive file
	archiveFile := fmt.Sprintf("%s/%s", serverDir, archiveName)
	err := os.WriteFile(archiveFile, data, filePerm)
	Require(t, err, "failed to write archive")

	// Write checksum file
	checksumFile := archiveFile + ".sha256"
	err = os.WriteFile(checksumFile, []byte(checksum), filePerm)
	Require(t, err, "failed to write checksum")

	// Start HTTP server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr := startFileServer(t, ctx, serverDir)

	// Download file
	initConfig := conf.InitConfigDefault
	initConfig.Url = fmt.Sprintf("http://%s/%s", addr, archiveName)
	initConfig.DownloadPath = t.TempDir()
	receivedArchive, err := downloadInit(ctx, &initConfig)
	Require(t, err, "failed to download")

	// Check archive contents
	receivedData, err := os.ReadFile(receivedArchive)
	Require(t, err, "failed to read received archive")
	if !bytes.Equal(receivedData, data) {
		t.Error("downloaded archive is different from generated one")
	}
}

func TestDownloadInitInParts(t *testing.T) {
	const (
		archiveName = "random_data.tar.gz"
		numParts    = 3
		partSize    = 1024 * 1024
		dataSize    = numParts * partSize
		filePerm    = 0600
	)

	// Create parts with random data
	serverDir := t.TempDir()
	data := testhelpers.RandomSlice(dataSize)
	for i := 0; i < numParts; i++ {
		// Create part and checksum
		partData := data[partSize*i : partSize*(i+1)]
		checksumBytes := sha256.Sum256(partData)
		checksum := hex.EncodeToString(checksumBytes[:])
		// Write part file
		partFile := fmt.Sprintf("%s/%s.part%d", serverDir, archiveName, i)
		err := os.WriteFile(partFile, partData, filePerm)
		Require(t, err, "failed to write part")
		// Write checksum file
		checksumFile := partFile + ".sha256"
		err = os.WriteFile(checksumFile, []byte(checksum), filePerm)
		Require(t, err, "failed to write checksum")
	}

	// Start HTTP server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr := startFileServer(t, ctx, serverDir)

	// Download file
	initConfig := conf.InitConfigDefault
	initConfig.Url = fmt.Sprintf("http://%s/%s", addr, archiveName)
	initConfig.DownloadPath = t.TempDir()
	receivedArchive, err := downloadInit(ctx, &initConfig)
	Require(t, err, "failed to download")

	// check database contents
	receivedData, err := os.ReadFile(receivedArchive)
	Require(t, err, "failed to read received archive")
	if !bytes.Equal(receivedData, data) {
		t.Error("downloaded archive is different from generated one")
	}

	// Check if the function deleted the temporary files
	entries, err := os.ReadDir(initConfig.DownloadPath)
	Require(t, err, "failed to read temp dir")
	if len(entries) != 1 {
		t.Error("download function did not delete temp files")
	}
}

func startFileServer(t *testing.T, ctx context.Context, dir string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	Require(t, err, "failed to listen")
	addr := ln.Addr().String()
	server := &http.Server{
		Addr:              addr,
		Handler:           http.FileServer(http.Dir(dir)),
		ReadHeaderTimeout: time.Second,
	}
	go func() {
		err := server.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Error("failed to shutdown server")
		}
	}()
	go func() {
		<-ctx.Done()
		err := server.Shutdown(ctx)
		Require(t, err, "failed to shutdown server")
	}()
	return addr
}

func TestOpenInitializeChainDbIncompatibleStateScheme(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	stack, err := node.New(stackConfig)
	defer stack.Close()
	Require(t, err)

	nodeConfig := NodeConfigDefault
	nodeConfig.Execution.Caching.StateScheme = rawdb.PathScheme
	nodeConfig.Chain.ID = 42161
	nodeConfig.Node = *arbnode.ConfigDefaultL2Test()
	nodeConfig.Init.DevInit = true
	nodeConfig.Init.DevInitAddress = "0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E"

	l1Client := ethclient.NewClient(stack.Attach())

	// opening for the first time doesn't error
	chainDb, blockchain, err := openInitializeChainDb(
		ctx,
		stack,
		&nodeConfig,
		new(big.Int).SetUint64(nodeConfig.Chain.ID),
		gethexec.DefaultCacheConfigFor(stack, &nodeConfig.Execution.Caching),
		&nodeConfig.Persistent,
		l1Client,
		chaininfo.RollupAddresses{},
	)
	Require(t, err)
	blockchain.Stop()
	err = chainDb.Close()
	Require(t, err)

	// opening for the second time doesn't error
	chainDb, blockchain, err = openInitializeChainDb(
		ctx,
		stack,
		&nodeConfig,
		new(big.Int).SetUint64(nodeConfig.Chain.ID),
		gethexec.DefaultCacheConfigFor(stack, &nodeConfig.Execution.Caching),
		&nodeConfig.Persistent,
		l1Client,
		chaininfo.RollupAddresses{},
	)
	Require(t, err)
	blockchain.Stop()
	err = chainDb.Close()
	Require(t, err)

	// opening with a different state scheme errors
	nodeConfig.Execution.Caching.StateScheme = rawdb.HashScheme
	_, _, err = openInitializeChainDb(
		ctx,
		stack,
		&nodeConfig,
		new(big.Int).SetUint64(nodeConfig.Chain.ID),
		gethexec.DefaultCacheConfigFor(stack, &nodeConfig.Execution.Caching),
		&nodeConfig.Persistent,
		l1Client,
		chaininfo.RollupAddresses{},
	)
	if !strings.Contains(err.Error(), "incompatible state scheme, stored: path, provided: hash") {
		t.Fatalf("Failed to detect incompatible state scheme")
	}
}
