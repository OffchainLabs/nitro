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
	"path"
	"path/filepath"
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

const (
	archiveName = "random_data.tar.gz"
	numParts    = 3
	partSize    = 1024 * 1024
	dataSize    = numParts * partSize
	filePerm    = 0600
	dirPerm     = 0700
)

func TestDownloadInitWithoutChecksum(t *testing.T) {
	// Create archive with random data
	serverDir := t.TempDir()
	data := testhelpers.RandomSlice(dataSize)

	// Write archive file
	archiveFile := fmt.Sprintf("%s/%s", serverDir, archiveName)
	err := os.WriteFile(archiveFile, data, filePerm)
	Require(t, err, "failed to write archive")

	// Start HTTP server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr := startFileServer(t, ctx, serverDir)

	// Download file
	initConfig := conf.InitConfigDefault
	initConfig.Url = fmt.Sprintf("http://%s/%s", addr, archiveName)
	initConfig.DownloadPath = t.TempDir()
	initConfig.ValidateChecksum = false
	receivedArchive, err := downloadInit(ctx, &initConfig)
	Require(t, err, "failed to download")

	// Check archive contents
	receivedData, err := os.ReadFile(receivedArchive)
	Require(t, err, "failed to read received archive")
	if !bytes.Equal(receivedData, data) {
		t.Error("downloaded archive is different from generated one")
	}
}

func TestDownloadInitWithChecksum(t *testing.T) {
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

func TestDownloadInitInPartsWithoutChecksum(t *testing.T) {
	// Create parts with random data
	serverDir := t.TempDir()
	data := testhelpers.RandomSlice(dataSize)
	manifest := bytes.NewBuffer(nil)
	for i := 0; i < numParts; i++ {
		partData := data[partSize*i : partSize*(i+1)]
		partName := fmt.Sprintf("%s.part%d", archiveName, i)
		fmt.Fprintf(manifest, "%s  %s\n", strings.Repeat("0", 64), partName)
		err := os.WriteFile(path.Join(serverDir, partName), partData, filePerm)
		Require(t, err, "failed to write part")
	}
	manifestFile := fmt.Sprintf("%s/%s.manifest.txt", serverDir, archiveName)
	err := os.WriteFile(manifestFile, manifest.Bytes(), filePerm)
	Require(t, err, "failed to write manifest file")

	// Start HTTP server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr := startFileServer(t, ctx, serverDir)

	// Download file
	initConfig := conf.InitConfigDefault
	initConfig.Url = fmt.Sprintf("http://%s/%s", addr, archiveName)
	initConfig.DownloadPath = t.TempDir()
	initConfig.ValidateChecksum = false
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

func TestDownloadInitInPartsWithChecksum(t *testing.T) {
	// Create parts with random data
	serverDir := t.TempDir()
	data := testhelpers.RandomSlice(dataSize)
	manifest := bytes.NewBuffer(nil)
	for i := 0; i < numParts; i++ {
		// Create part and checksum
		partData := data[partSize*i : partSize*(i+1)]
		partName := fmt.Sprintf("%s.part%d", archiveName, i)
		checksumBytes := sha256.Sum256(partData)
		checksum := hex.EncodeToString(checksumBytes[:])
		fmt.Fprintf(manifest, "%s  %s\n", checksum, partName)
		// Write part file
		err := os.WriteFile(path.Join(serverDir, partName), partData, filePerm)
		Require(t, err, "failed to write part")
	}
	manifestFile := fmt.Sprintf("%s/%s.manifest.txt", serverDir, archiveName)
	err := os.WriteFile(manifestFile, manifest.Bytes(), filePerm)
	Require(t, err, "failed to write manifest file")

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

func TestSetLatestSnapshotUrl(t *testing.T) {
	const (
		chain        = "arb1"
		snapshotKind = "archive"
		latestFile   = "latest-" + snapshotKind + ".txt"
	)

	testCases := []struct {
		name           string
		latestContents string
		wantUrl        func(string) string
	}{
		{
			name:           "latest file with path",
			latestContents: "/arb1/2024/21/archive.tar.gz",
			wantUrl:        func(serverAddr string) string { return serverAddr + "/arb1/2024/21/archive.tar.gz" },
		},
		{
			name:           "latest file with rootless path",
			latestContents: "arb1/2024/21/archive.tar.gz",
			wantUrl:        func(serverAddr string) string { return serverAddr + "/arb1/2024/21/archive.tar.gz" },
		},
		{
			name:           "latest file with http url",
			latestContents: "http://some.domain.com/arb1/2024/21/archive.tar.gz",
			wantUrl:        func(serverAddr string) string { return "http://some.domain.com/arb1/2024/21/archive.tar.gz" },
		},
		{
			name:           "latest file with https url",
			latestContents: "https://some.domain.com/arb1/2024/21/archive.tar.gz",
			wantUrl:        func(serverAddr string) string { return "https://some.domain.com/arb1/2024/21/archive.tar.gz" },
		},
	}

	for _, testCase := range testCases {
		t.Log("running test case", testCase.name)

		// Create latest file
		serverDir := t.TempDir()
		err := os.Mkdir(filepath.Join(serverDir, chain), dirPerm)
		Require(t, err)
		err = os.WriteFile(filepath.Join(serverDir, chain, latestFile), []byte(testCase.latestContents), filePerm)
		Require(t, err)

		// Start HTTP server
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		addr := "http://" + startFileServer(t, ctx, serverDir)

		// Set latest snapshot URL
		initConfig := conf.InitConfigDefault
		initConfig.Latest = snapshotKind
		initConfig.LatestBase = addr
		err = setLatestSnapshotUrl(ctx, &initConfig, chain)
		Require(t, err)

		// Check url
		want := testCase.wantUrl(addr)
		if initConfig.Url != want {
			t.Fatalf("initConfig.Url = %s; want: %s", initConfig.Url, want)
		}
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

func testIsNotExistError(t *testing.T, dbEngine string, isNotExist func(error) bool) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stackConf.DBEngine = dbEngine
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to created test stack: %v", err)
	}
	defer stack.Close()
	readonly := true
	_, err = stack.OpenDatabaseWithExtraOptions("test", 16, 16, "", readonly, nil)
	if err == nil {
		t.Fatal("Opening non-existent database did not fail")
	}
	if !isNotExist(err) {
		t.Fatalf("Failed to classify error as not exist error - internal implementation of OpenDatabaseWithExtraOptions might have changed, err: %v", err)
	}
	err = errors.New("some other error")
	if isNotExist(err) {
		t.Fatalf("Classified other error as not exist, err: %v", err)
	}
}

func TestIsNotExistError(t *testing.T) {
	t.Run("TestIsPebbleNotExistError", func(t *testing.T) {
		testIsNotExistError(t, "pebble", isPebbleNotExistError)
	})
	t.Run("TestIsLeveldbNotExistError", func(t *testing.T) {
		testIsNotExistError(t, "leveldb", isLeveldbNotExistError)
	})
}

func TestEmptyDatabaseDir(t *testing.T) {
	testCases := []struct {
		name    string
		files   []string
		force   bool
		wantErr string
	}{
		{
			name: "succeed with empty dir",
		},
		{
			name:  "succeed with expected files",
			files: []string{"LOCK", "classic-msg", "l2chaindata"},
		},
		{
			name:    "fail with unexpected files",
			files:   []string{"LOCK", "a", "b", "c", "d"},
			wantErr: "found 4 unexpected files in database directory, including: a, b, c",
		},
		{
			name:    "fail with unexpected files when forcing",
			files:   []string{"LOCK", "a", "b", "c", "d"},
			force:   true,
			wantErr: "trying to overwrite old database directory",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range tc.files {
				const filePerm = 0600
				err := os.WriteFile(path.Join(dir, file), []byte{1, 2, 3}, filePerm)
				Require(t, err)
			}
			err := checkEmptyDatabaseDir(dir, tc.force)
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("expected nil error, got %q", err)
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("expected %q, got %q", tc.wantErr, err)
				}
			}
		})
	}
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
