// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package nitroinit

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/nitro/config"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
)

const (
	archiveName = "random_data.tar.gz"
	numParts    = 3
	partSize    = 1024 * 1024
	dataSize    = numParts * partSize
	filePerm    = 0600
	dirPerm     = 0700
)

func TestInitializeAndDownloadInit(t *testing.T) {
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
	addr, server := startFileServer(t, serverDir)
	defer gracefulShutdown(t, ctx, server)

	stackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	stack, err := node.New(stackConfig)
	Require(t, err)
	defer stack.Close()

	// Download file
	initConfig := conf.InitConfigDefault
	initConfig.Url = fmt.Sprintf("http://%s/%s", addr, archiveName)
	initConfig.ValidateChecksum = false
	initConfig.DownloadPath = ""
	receivedArchive, cleanUpTmp, err := initializeAndDownloadInit(ctx, &initConfig, stack)
	Require(t, err, "failed to download")

	// Check archive contents
	receivedData, err := os.ReadFile(receivedArchive)
	Require(t, err, "failed to read received archive")
	if !bytes.Equal(receivedData, data) {
		t.Error("downloaded archive is different from generated one")
	}

	// Check if initConfig.DownloadPath is as expected and that the cleanup function deletes temporary directory (tmp) inside chain directory
	expectedDownloadPath := filepath.Join(stack.InstanceDir(), "tmp")
	if initConfig.DownloadPath != expectedDownloadPath {
		t.Errorf("unexpected default download path. Want: %s Got: %s", expectedDownloadPath, initConfig.DownloadPath)
	}
	_, err = os.Stat(initConfig.DownloadPath)
	Require(t, err)
	cleanUpTmp()
	_, err = os.Stat(initConfig.DownloadPath)
	if !os.IsNotExist(err) {
		t.Errorf("expecting os.stat to return os.ErrNotExist error after tmp directory cleanup, found: %s", err.Error())
	}
}

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
	addr, server := startFileServer(t, serverDir)
	defer gracefulShutdown(t, ctx, server)

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
	addr, server := startFileServer(t, serverDir)
	defer gracefulShutdown(t, ctx, server)

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
	addr, server := startFileServer(t, serverDir)
	defer gracefulShutdown(t, ctx, server)

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
	addr, server := startFileServer(t, serverDir)
	defer gracefulShutdown(t, ctx, server)

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
		chain          string
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
		{
			name:           "chain and contents with upper case",
			chain:          "ARB1",
			latestContents: "ARB1/2024/21/ARCHIVE.TAR.GZ",
			wantUrl:        func(serverAddr string) string { return serverAddr + "/arb1/2024/21/archive.tar.gz" },
		},
	}

	for _, testCase := range testCases {
		func() {
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
			addr, server := startFileServer(t, serverDir)
			defer gracefulShutdown(t, ctx, server)
			addr = "http://" + addr

			// Set latest snapshot URL
			initConfig := conf.InitConfigDefault
			initConfig.Latest = snapshotKind
			initConfig.LatestBase = addr
			configChain := testCase.chain
			if configChain == "" {
				configChain = chain
			}
			err = setLatestSnapshotUrl(ctx, &initConfig, configChain)
			Require(t, err)

			// Check url
			want := testCase.wantUrl(addr)
			if initConfig.Url != want {
				t.Fatalf("initConfig.Url = %s; want: %s", initConfig.Url, want)
			}
		}()
	}
}

func startFileServer(t *testing.T, dir string) (string, *http.Server) {
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
	return addr, server
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

func TestOpenInitializeExecutionDBIncompatibleStateScheme(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	stackConfig.DBEngine = rawdb.DBPebble
	stack, err := node.New(stackConfig)
	Require(t, err)
	defer stack.Close()

	nodeConfig := config.NodeConfigDefault
	nodeConfig.Execution.Caching.StateScheme = rawdb.PathScheme
	nodeConfig.Chain.ID = 42161
	nodeConfig.Node = *arbnode.ConfigDefaultL2Test()
	nodeConfig.Init.DevInit = true
	nodeConfig.Init.DevInitAddress = "0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E"
	nodeConfig.Init.ValidateGenesisAssertion = false

	l1Client := ethclient.NewClient(stack.Attach())

	// opening for the first time doesn't error
	executionDB, _, blockchain, err := OpenInitializeExecutionDB(
		ctx,
		stack,
		&nodeConfig,
		new(big.Int).SetUint64(nodeConfig.Chain.ID),
		gethexec.DefaultCacheConfigFor(&nodeConfig.Execution.Caching),
		nil,
		&nodeConfig.Persistent,
		l1Client,
		chaininfo.RollupAddresses{},
	)
	Require(t, err)
	blockchain.Stop()
	err = executionDB.Close()
	Require(t, err)

	// opening for the second time doesn't error
	executionDB, _, blockchain, err = OpenInitializeExecutionDB(
		ctx,
		stack,
		&nodeConfig,
		new(big.Int).SetUint64(nodeConfig.Chain.ID),
		gethexec.DefaultCacheConfigFor(&nodeConfig.Execution.Caching),
		nil,
		&nodeConfig.Persistent,
		l1Client,
		chaininfo.RollupAddresses{},
	)
	Require(t, err)
	blockchain.Stop()
	err = executionDB.Close()
	Require(t, err)

	// opening with a different state scheme errors
	nodeConfig.Execution.Caching.StateScheme = rawdb.HashScheme
	_, _, _, err = OpenInitializeExecutionDB(
		ctx,
		stack,
		&nodeConfig,
		new(big.Int).SetUint64(nodeConfig.Chain.ID),
		gethexec.DefaultCacheConfigFor(&nodeConfig.Execution.Caching),
		nil,
		&nodeConfig.Persistent,
		l1Client,
		chaininfo.RollupAddresses{},
	)
	if !strings.Contains(err.Error(), "incompatible state scheme, stored: path, provided: hash") {
		t.Fatalf("Failed to detect incompatible state scheme")
	}
}

func writeKeys(t *testing.T, db ethdb.Database, keys [][]byte) {
	t.Helper()
	batch := db.NewBatch()
	for _, key := range keys {
		err := batch.Put(key, []byte("some data"))
		if err != nil {
			t.Fatal("Internal test error - failed to insert key:", err)
		}
	}
	err := batch.Write()
	if err != nil {
		t.Fatal("Internal test error - failed to write batch:", err)
	}
	batch.Reset()
}

func checkKeys(t *testing.T, db ethdb.Database, keys [][]byte, shouldExist bool) {
	t.Helper()
	for _, key := range keys {
		has, err := db.Has(key)
		if err != nil {
			t.Fatal("Failed to check key existence, key: ", key)
		}
		if shouldExist && !has {
			t.Fatal("Key not found:", key)
		}
		if !shouldExist && has {
			t.Fatal("Key found:", key, "k3:", string(key[:3]), "len", len(key))
		}
	}
}

func generateKeys(prefix []byte, numKeys int) [][]byte {
	var keys [][]byte
	for i := 0; i < numKeys; i++ {
		keys = append(keys, append(prefix, testhelpers.RandomSlice(32)...))
	}
	return keys
}

func TestPurgeIncompatibleWasmerSerializeVersionEntries(t *testing.T) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create test stack: %v", err)
	}
	defer stack.Close()
	db, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", Cache: config.NodeConfigDefault.Execution.Caching.DatabaseCache, Handles: config.NodeConfigDefault.Persistent.Handles, NoFreezer: true})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	version0Keys := generateKeys([]byte{0x00, 'w', 'a'}, 20)
	version0Keys = append(version0Keys, generateKeys([]byte{0x00, 'w', 'm'}, 20)...)
	wavmKeys := generateKeys([]byte{0x00, 'w', 'w'}, 20)
	armKeys := generateKeys([]byte{0x00, 'w', 'r'}, 20)
	x86Keys := generateKeys([]byte{0x00, 'w', 'x'}, 20)
	hostKeys := generateKeys([]byte{0x00, 'w', 'h'}, 20)

	var otherKeys [][]byte
	for i := 0x00; i <= 0xff; i++ {
		if byte(i) == 'a' || byte(i) == 'm' || byte(i) == 'w' || byte(i) == 'r' || byte(i) == 'x' || byte(i) == 'h' {
			continue
		}
		for k := 0x00; k <= 0xff; k++ {
			otherKeys = append(otherKeys, generateKeys([]byte{0x00, byte(k), byte(i)}, 2)...)
		}
	}

	// write all keys and check they exist in the db
	writeKeys(t, db, version0Keys)
	writeKeys(t, db, wavmKeys)
	writeKeys(t, db, armKeys)
	writeKeys(t, db, x86Keys)
	writeKeys(t, db, hostKeys)
	writeKeys(t, db, otherKeys)
	checkKeys(t, db, version0Keys, true)
	checkKeys(t, db, wavmKeys, true)
	checkKeys(t, db, armKeys, true)
	checkKeys(t, db, x86Keys, true)
	checkKeys(t, db, hostKeys, true)
	checkKeys(t, db, otherKeys, true)

	// if Nitro's WasmerSerializeVersion is compatible with WasmerSerializeVersion
	// stored in the database then all keys should still exist
	err = rawdb.WriteWasmerSerializeVersion(db, WasmerSerializeVersion)
	Require(t, err)
	err = validateOrUpgradeWasmerSerializeVersion(db)
	Require(t, err)
	checkKeys(t, db, version0Keys, true)
	checkKeys(t, db, wavmKeys, true)
	checkKeys(t, db, armKeys, true)
	checkKeys(t, db, x86Keys, true)
	checkKeys(t, db, hostKeys, true)
	checkKeys(t, db, otherKeys, true)
	currWasmerSerializeVersion, err := rawdb.ReadWasmerSerializeVersion(db)
	Require(t, err)
	if currWasmerSerializeVersion != WasmerSerializeVersion {
		t.Fatalf("Expected current WasmerSerializeVersion to be %d, got %d", WasmerSerializeVersion, currWasmerSerializeVersion)
	}

	// if Nitro's WasmerSerializeVersion is not compatible with WasmerSerializeVersion
	// stored in the database then all keys, except wavm and other keys, should be removed
	err = rawdb.WriteWasmerSerializeVersion(db, WasmerSerializeVersion-1)
	Require(t, err)
	err = validateOrUpgradeWasmerSerializeVersion(db)
	Require(t, err)
	checkKeys(t, db, version0Keys, false)
	checkKeys(t, db, wavmKeys, true)
	checkKeys(t, db, armKeys, false)
	checkKeys(t, db, x86Keys, false)
	checkKeys(t, db, hostKeys, false)
	checkKeys(t, db, otherKeys, true)
	currWasmerSerializeVersion, err = rawdb.ReadWasmerSerializeVersion(db)
	Require(t, err)
	if currWasmerSerializeVersion != WasmerSerializeVersion {
		t.Fatalf("Expected current WasmerSerializeVersion to be %d, got %d", WasmerSerializeVersion, currWasmerSerializeVersion)
	}
}

func TestPurgeIncompatibleWavmSerializeVersionEntries(t *testing.T) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create test stack: %v", err)
	}
	defer stack.Close()
	db, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", Cache: config.NodeConfigDefault.Execution.Caching.DatabaseCache, Handles: config.NodeConfigDefault.Persistent.Handles, NoFreezer: true})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	// Seed all the wasmdb prefixes the production validators interact with.
	// The wavm validator must purge only `wavmKeys` and leave everything else.
	wavmKeys := generateKeys([]byte{0x00, 'w', 'w'}, 20)
	armKeys := generateKeys([]byte{0x00, 'w', 'r'}, 20)
	x86Keys := generateKeys([]byte{0x00, 'w', 'x'}, 20)
	hostKeys := generateKeys([]byte{0x00, 'w', 'h'}, 20)
	otherKeys := generateKeys([]byte{0x01, 'z', 'z'}, 20)

	writeKeys(t, db, wavmKeys)
	writeKeys(t, db, armKeys)
	writeKeys(t, db, x86Keys)
	writeKeys(t, db, hostKeys)
	writeKeys(t, db, otherKeys)

	// Case 1: no WavmSerializeVersion key in the DB. The validator must treat
	// this as incompatible (existing nodes upgrading from bincode have no key
	// and their stored bytes are in the old format), purge wavm entries, leave
	// the wasmer-managed prefixes untouched, and write the current version.
	purged, err := validateOrUpgradeWavmSerializeVersion(db)
	Require(t, err)
	if !purged {
		t.Fatal("Case 1: validator must report purged=true when wavm keys existed without a version key")
	}
	checkKeys(t, db, wavmKeys, false)
	checkKeys(t, db, armKeys, true)
	checkKeys(t, db, x86Keys, true)
	checkKeys(t, db, hostKeys, true)
	checkKeys(t, db, otherKeys, true)
	got, err := rawdb.ReadWavmSerializeVersion(db)
	Require(t, err)
	if got != WavmSerializeVersion {
		t.Fatalf("after missing-key purge, expected version %d, got %d", WavmSerializeVersion, got)
	}

	// Case 2: matching version. Re-seed wavm keys and confirm the validator
	// preserves them and reports purged=false.
	writeKeys(t, db, wavmKeys)
	purged, err = validateOrUpgradeWavmSerializeVersion(db)
	Require(t, err)
	if purged {
		t.Fatal("Case 2: validator must report purged=false on matching version")
	}
	checkKeys(t, db, wavmKeys, true)
	checkKeys(t, db, armKeys, true)
	checkKeys(t, db, x86Keys, true)
	checkKeys(t, db, hostKeys, true)
	checkKeys(t, db, otherKeys, true)

	// Case 3: mismatched version. Wavm entries are purged, the wasmer-managed
	// prefixes survive untouched, and the version key is rewritten to current.
	Require(t, rawdb.WriteWavmSerializeVersion(db, WavmSerializeVersion+1))
	purged, err = validateOrUpgradeWavmSerializeVersion(db)
	Require(t, err)
	if !purged {
		t.Fatal("Case 3: validator must report purged=true on mismatched version")
	}
	checkKeys(t, db, wavmKeys, false)
	checkKeys(t, db, armKeys, true)
	checkKeys(t, db, x86Keys, true)
	checkKeys(t, db, hostKeys, true)
	checkKeys(t, db, otherKeys, true)
	got, err = rawdb.ReadWavmSerializeVersion(db)
	Require(t, err)
	if got != WavmSerializeVersion {
		t.Fatalf("after mismatched-version purge, expected version %d, got %d", WavmSerializeVersion, got)
	}
}

// Empty-wasmdb startup must write the version key proactively rather than
// running the "missing key = old format" purge branch. Without this, the next
// non-empty startup would mistake the absence of the key for legacy bincode
// data and purge legitimate new-format entries activated during this session.
func TestValidateOrUpgradeWavmOnEmptyDbWritesVersionKey(t *testing.T) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create test stack: %v", err)
	}
	defer stack.Close()
	db, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", Cache: config.NodeConfigDefault.Execution.Caching.DatabaseCache, Handles: config.NodeConfigDefault.Persistent.Handles, NoFreezer: true})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	// DB is empty: validator must write the version key and otherwise do nothing.
	purged, err := validateOrUpgradeWavmSerializeVersion(db)
	Require(t, err)
	if purged {
		t.Fatal("empty wasmdb must report purged=false; nothing was deleted")
	}

	got, err := rawdb.ReadWavmSerializeVersion(db)
	Require(t, err)
	if got != WavmSerializeVersion {
		t.Fatalf("expected version %d on empty-db startup, got %d", WavmSerializeVersion, got)
	}
}

// Wasmer-only wasmdb (arm/x86/host but no wavm) must take the
// no-wavm-entries branch: write version key, no purge, no misleading warn.
func TestValidateOrUpgradeWavmSkipsPurgeOnWasmerOnlyDb(t *testing.T) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create test stack: %v", err)
	}
	defer stack.Close()
	db, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", Cache: config.NodeConfigDefault.Execution.Caching.DatabaseCache, Handles: config.NodeConfigDefault.Persistent.Handles, NoFreezer: true})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	// Seed wasmer-managed prefixes but no wavm keys, no version key.
	armKeys := generateKeys([]byte{0x00, 'w', 'r'}, 5)
	x86Keys := generateKeys([]byte{0x00, 'w', 'x'}, 5)
	hostKeys := generateKeys([]byte{0x00, 'w', 'h'}, 5)
	writeKeys(t, db, armKeys)
	writeKeys(t, db, x86Keys)
	writeKeys(t, db, hostKeys)

	purged, err := validateOrUpgradeWavmSerializeVersion(db)
	Require(t, err)
	if purged {
		t.Fatal("wasmer-only wasmdb must take the no-wavm-entries branch (purged=false)")
	}
	checkKeys(t, db, armKeys, true)
	checkKeys(t, db, x86Keys, true)
	checkKeys(t, db, hostKeys, true)
	got, err := rawdb.ReadWavmSerializeVersion(db)
	Require(t, err)
	if got != WavmSerializeVersion {
		t.Fatalf("expected version %d to be written, got %d", WavmSerializeVersion, got)
	}
}

// iteratorFailingDB injects iterator failures into deleteWasmEntries.
// `failNextAfter` exercises the end-of-loop it.Error() check;
// `errorAfter` exercises the mid-loop (batch-boundary) check by making
// Error() return non-nil while Next() keeps working.
type iteratorFailingDB struct {
	ethdb.Database
	nextCalls     *int // shared across iterators to budget by global progress
	failNextAfter int
	errorAfter    int
	injected      error
}

func (d *iteratorFailingDB) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return &erroringIter{
		Iterator:      d.Database.NewIterator(prefix, start),
		nextCalls:     d.nextCalls,
		failNextAfter: d.failNextAfter,
		errorAfter:    d.errorAfter,
		injected:      d.injected,
	}
}

type erroringIter struct {
	ethdb.Iterator
	nextCalls     *int
	failNextAfter int
	errorAfter    int
	injected      error
	tripped       bool
}

func (it *erroringIter) Next() bool {
	if it.tripped {
		return false
	}
	if it.failNextAfter > 0 && *it.nextCalls >= it.failNextAfter {
		it.tripped = true
		return false
	}
	*it.nextCalls++
	return it.Iterator.Next()
}

func (it *erroringIter) Error() error {
	if it.tripped {
		return it.injected
	}
	if it.errorAfter > 0 && *it.nextCalls >= it.errorAfter {
		return it.injected
	}
	return it.Iterator.Error()
}

// End-of-loop it.Error() check: propagated, version stamp NOT advanced.
// FFI mismatch must short-circuit before touching the wasmdb: no purge, no
// version key written.
func TestValidateOrUpgradeWavmRefusesOnFFIMismatch(t *testing.T) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create test stack: %v", err)
	}
	defer stack.Close()
	db, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", Cache: config.NodeConfigDefault.Execution.Caching.DatabaseCache, Handles: config.NodeConfigDefault.Persistent.Handles, NoFreezer: true})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	// Seed wavm entries so we'd observe a purge if the guard failed.
	wavmKeys := generateKeys([]byte{0x00, 'w', 'w'}, 5)
	writeKeys(t, db, wavmKeys)

	// NOTE: do NOT add t.Parallel() to any test that touches
	// readRustWavmFormatVersion: the package-level override is restored via
	// t.Cleanup at the end of this test, which is only safe under sequential
	// execution within the package.
	original := readRustWavmFormatVersion
	t.Cleanup(func() { readRustWavmFormatVersion = original })
	readRustWavmFormatVersion = func() uint32 { return WavmSerializeVersion + 1 }

	purged, err := validateOrUpgradeWavmSerializeVersion(db)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("error must mention mismatch; got %v", err)
	}
	if purged {
		t.Fatal("purged must be false on mismatch")
	}

	// No side effects: keys still present, version key not written.
	checkKeys(t, db, wavmKeys, true)
	if _, readErr := rawdb.ReadWavmSerializeVersion(db); readErr == nil {
		t.Fatal("version key must not be written on mismatch")
	}
}

// Probe I/O failure must bail, not silently return false (a false return
// would skip the purge and leave stale entries for the decoder to reject
// later as "magic mismatch").
func TestValidateOrUpgradeWavmPropagatesProbeError(t *testing.T) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create test stack: %v", err)
	}
	defer stack.Close()
	db, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", Cache: config.NodeConfigDefault.Execution.Caching.DatabaseCache, Handles: config.NodeConfigDefault.Persistent.Handles, NoFreezer: true})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	calls := 0
	injected := errors.New("injected probe I/O error")
	// errorAfter=1 fires Error() on the first Next() the probe issues.
	wrapped := &iteratorFailingDB{
		Database:   db,
		nextCalls:  &calls,
		errorAfter: 1,
		injected:   injected,
	}

	purged, err := validateOrUpgradeWavmSerializeVersion(wrapped)
	if err == nil {
		t.Fatal("expected probe error to be propagated")
	}
	if !errors.Is(err, injected) {
		t.Fatalf("expected wrapped injected error, got %v", err)
	}
	if purged {
		t.Fatal("purged must be false when the probe errored out")
	}

	// Version key must NOT have been written: a future boot must retry.
	_, readErr := rawdb.ReadWavmSerializeVersion(db)
	if readErr == nil {
		t.Fatal("version key must not be written when the probe fails")
	}
}

func TestValidateOrUpgradeWavmPropagatesIteratorError(t *testing.T) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create test stack: %v", err)
	}
	defer stack.Close()
	db, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", Cache: config.NodeConfigDefault.Execution.Caching.DatabaseCache, Handles: config.NodeConfigDefault.Persistent.Handles, NoFreezer: true})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	wavmKeys := generateKeys([]byte{0x00, 'w', 'w'}, 20)
	writeKeys(t, db, wavmKeys)

	// 1 Next() for the wavmPrefixHasEntries probe; the next Next() (first
	// from the purge iterator) trips, exercising end-of-loop it.Error().
	calls := 0
	injected := errors.New("injected iterator I/O error")
	wrapped := &iteratorFailingDB{
		Database:      db,
		nextCalls:     &calls,
		failNextAfter: 1,
		injected:      injected,
	}

	purged, err := validateOrUpgradeWavmSerializeVersion(wrapped)
	if err == nil {
		t.Fatal("expected iterator error to be propagated")
	}
	if !errors.Is(err, injected) {
		t.Fatalf("expected wrapped injected error, got %v", err)
	}
	if purged {
		t.Fatal("purged must be false when the purge errored out")
	}

	// Version key must NOT have been written; next boot must retry.
	_, readErr := rawdb.ReadWavmSerializeVersion(db)
	if readErr == nil {
		t.Fatal("version key must not be written when the purge fails")
	}
}

// Mid-loop it.Error() check at the batch-write boundary. Requires enough
// keys to push batch.ValueSize() over ethdb.IdealBatchSize (100KB) so the
// mid-loop check is the first error surface (vs the end-of-loop one tested
// above).
func TestValidateOrUpgradeWavmPropagatesIteratorErrorAtBatchBoundary(t *testing.T) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create test stack: %v", err)
	}
	defer stack.Close()
	db, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", Cache: config.NodeConfigDefault.Execution.Caching.DatabaseCache, Handles: config.NodeConfigDefault.Persistent.Handles, NoFreezer: true})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	// 5000 35-byte-key deletes comfortably exceed IdealBatchSize (100KB).
	const numKeys = 5000
	wavmKeys := generateKeys([]byte{0x00, 'w', 'w'}, numKeys)
	writeKeys(t, db, wavmKeys)

	// Iteration runs normally; Error() starts firing after 100 Next() calls,
	// well before exhaustion. First batch fill then bails via mid-loop check.
	calls := 0
	injected := errors.New("injected mid-iteration I/O error")
	wrapped := &iteratorFailingDB{
		Database:   db,
		nextCalls:  &calls,
		errorAfter: 100,
		injected:   injected,
	}

	purged, err := validateOrUpgradeWavmSerializeVersion(wrapped)
	if err == nil {
		t.Fatal("expected mid-loop iterator error to be propagated")
	}
	if !errors.Is(err, injected) {
		t.Fatalf("expected wrapped injected error, got %v", err)
	}
	if purged {
		t.Fatal("purged must be false when the purge errored out")
	}
	// Distinguishes mid-loop from end-of-loop: end-of-loop bails without
	// running batch.Write (all keys survive); mid-loop runs after at least
	// one batch.Write succeeded (some keys gone).
	survivors := 0
	for _, k := range wavmKeys {
		has, hasErr := db.Has(k)
		if hasErr != nil {
			t.Fatalf("Has(%x): %v", k, hasErr)
		}
		if has {
			survivors++
		}
	}
	if survivors == len(wavmKeys) {
		t.Fatalf("no keys were deleted; mid-loop check did not fire (all %d keys survived)", survivors)
	}
	if survivors == 0 {
		t.Fatalf("all keys were deleted; the error must have surfaced before the trailing batch.Write")
	}

	// Version key must not have been written.
	_, readErr := rawdb.ReadWavmSerializeVersion(db)
	if readErr == nil {
		t.Fatal("version key must not be written when the purge fails")
	}
}

// Pins outer-caller behavior: when validateOrUpgradeWavmSerializeVersion
// reports purged=true, openDownloadedExecutionDB must skip the
// RebuildingDone stamp so the recovery rebuild runs.
func TestOpenDownloadedExecutionDBSkipsStampOnPurge(t *testing.T) {
	t.Parallel()

	stackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	stackConfig.DBEngine = rawdb.DBPebble
	stack, err := node.New(stackConfig)
	Require(t, err)
	defer stack.Close()

	nodeConfig := config.NodeConfigDefault
	nodeConfig.Execution.Caching.StateScheme = rawdb.PathScheme
	nodeConfig.Chain.ID = 42161

	// Stale wavm entries with no version key — simulates a snapshot from
	// the bincode era; the validator must treat this as incompatible.
	{
		seedDB, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{
			MetricsNamespace:   "wasm/",
			Cache:              nodeConfig.Execution.Caching.DatabaseCache,
			Handles:            nodeConfig.Persistent.Handles,
			NoFreezer:          true,
			PebbleExtraOptions: nodeConfig.Persistent.Pebble.ExtraOptions("wasm"),
		})
		Require(t, err)
		wavmKeys := generateKeys([]byte{0x00, 'w', 'w'}, 5)
		writeKeys(t, seedDB, wavmKeys)
		Require(t, seedDB.Close())
	}

	executionDB, wasmDB, err := openDownloadedExecutionDB(stack, &nodeConfig, gethexec.DefaultCacheConfigFor(&nodeConfig.Execution.Caching), &nodeConfig.Persistent)
	Require(t, err)
	defer func() { Require(t, executionDB.Close()) }()

	wavmIt := wasmDB.NewIterator([]byte{0x00, 'w', 'w'}, nil)
	if wavmIt.Next() {
		t.Fatal("wavm entries must have been purged")
	}
	wavmIt.Release()

	got, err := rawdb.ReadWavmSerializeVersion(wasmDB)
	Require(t, err)
	if got != WavmSerializeVersion {
		t.Fatalf("expected version %d after purge, got %d", WavmSerializeVersion, got)
	}

	// Centerpiece: RebuildingPositionKey must NOT be set to RebuildingDone.
	has, err := wasmDB.Has(gethexec.RebuildingPositionKey)
	Require(t, err)
	if has {
		stored, err := gethexec.ReadFromKeyValueStore[common.Hash](wasmDB, gethexec.RebuildingPositionKey)
		Require(t, err)
		t.Fatalf("RebuildingPositionKey must not be set after a purge; openDownloadedExecutionDB stamped %x", stored)
	}
}

// Fresh snapshot (no purge): the RebuildingDone stamp still fires.
func TestOpenDownloadedExecutionDBStampsWhenNoPurge(t *testing.T) {
	t.Parallel()

	stackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	stackConfig.DBEngine = rawdb.DBPebble
	stack, err := node.New(stackConfig)
	Require(t, err)
	defer stack.Close()

	nodeConfig := config.NodeConfigDefault
	nodeConfig.Execution.Caching.StateScheme = rawdb.PathScheme
	nodeConfig.Chain.ID = 42161

	executionDB, wasmDB, err := openDownloadedExecutionDB(stack, &nodeConfig, gethexec.DefaultCacheConfigFor(&nodeConfig.Execution.Caching), &nodeConfig.Persistent)
	Require(t, err)
	defer func() { Require(t, executionDB.Close()) }()

	stored, err := gethexec.ReadFromKeyValueStore[common.Hash](wasmDB, gethexec.RebuildingPositionKey)
	Require(t, err)
	if stored != gethexec.RebuildingDone {
		t.Fatalf("expected RebuildingDone stamp on fresh open, got %x", stored)
	}
}

// After a wavm purge, RebuildingPositionKey must be cleared so
// rebuildLocalWasm restarts recovery (a leftover "done" would skip it).
func TestValidateOrUpgradeWavmResetsRebuildingPositionAfterPurge(t *testing.T) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create test stack: %v", err)
	}
	defer stack.Close()
	db, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", Cache: config.NodeConfigDefault.Execution.Caching.DatabaseCache, Handles: config.NodeConfigDefault.Persistent.Handles, NoFreezer: true})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	// Previously-completed rebuild + old-format wavm entries.
	wavmKeys := generateKeys([]byte{0x00, 'w', 'w'}, 5)
	writeKeys(t, db, wavmKeys)
	Require(t, gethexec.WriteToKeyValueStore(db, gethexec.RebuildingPositionKey, gethexec.RebuildingDone))

	purged, err := validateOrUpgradeWavmSerializeVersion(db)
	Require(t, err)
	if !purged {
		t.Fatal("missing version key + wavm entries must trigger a purge")
	}
	checkKeys(t, db, wavmKeys, false)

	has, err := db.Has(gethexec.RebuildingPositionKey)
	Require(t, err)
	if has {
		t.Fatal("RebuildingPositionKey must be deleted after a wavm purge so rebuild restarts")
	}
}

// CI drift guard: Go and Rust constants must agree. Reads via FFI so the
// check is robust to source-format changes.
func TestWavmSerializeVersionMatchesRustConstant(t *testing.T) {
	rustVersion := programs.RustWavmFormatVersion()
	if rustVersion != WavmSerializeVersion {
		t.Fatalf("WavmSerializeVersion drift: Go=%d, Rust=%d. Bump both together.", WavmSerializeVersion, rustVersion)
	}
}

func TestPurgeVersion0WasmStoreEntries(t *testing.T) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create test stack: %v", err)
	}
	defer stack.Close()
	db, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", Cache: config.NodeConfigDefault.Execution.Caching.DatabaseCache, Handles: config.NodeConfigDefault.Persistent.Handles, NoFreezer: true})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	var version0Keys [][]byte
	for i := 0; i < 20; i++ {
		version0Keys = append(version0Keys,
			append([]byte{0x00, 'w', 'a'}, testhelpers.RandomSlice(32)...))
		version0Keys = append(version0Keys,
			append([]byte{0x00, 'w', 'm'}, testhelpers.RandomSlice(32)...))
	}
	var collidedKeys [][]byte
	for i := 0; i < 5; i++ {
		collidedKeys = append(collidedKeys,
			append([]byte{0x00, 'w', 'a'}, testhelpers.RandomSlice(31)...))
		collidedKeys = append(collidedKeys,
			append([]byte{0x00, 'w', 'm'}, testhelpers.RandomSlice(31)...))
		collidedKeys = append(collidedKeys,
			append([]byte{0x00, 'w', 'a'}, testhelpers.RandomSlice(33)...))
		collidedKeys = append(collidedKeys,
			append([]byte{0x00, 'w', 'm'}, testhelpers.RandomSlice(33)...))
	}
	var otherKeys [][]byte
	for i := 0x00; i <= 0xff; i++ {
		if byte(i) == 'a' || byte(i) == 'm' {
			continue
		}
		otherKeys = append(otherKeys,
			append([]byte{0x00, 'w', byte(i)}, testhelpers.RandomSlice(32)...))
		otherKeys = append(otherKeys,
			append([]byte{0x00, 'w', byte(i)}, testhelpers.RandomSlice(32)...))
	}
	for i := 0; i < 10; i++ {
		var randomSlice []byte
		var j int
		for j = 0; j < 10; j++ {
			randomSlice = testhelpers.RandomSlice(testhelpers.RandomUint64(1, 40))
			if len(randomSlice) >= 3 && !bytes.Equal(randomSlice[:3], []byte{0x00, 'w', 'm'}) && !bytes.Equal(randomSlice[:3], []byte{0x00, 'w', 'a'}) {
				break
			}
		}
		if j == 10 {
			t.Fatal("Internal test error - failed to generate random key")
		}
		otherKeys = append(otherKeys, randomSlice)
	}
	writeKeys(t, db, version0Keys)
	writeKeys(t, db, collidedKeys)
	writeKeys(t, db, otherKeys)
	checkKeys(t, db, version0Keys, true)
	checkKeys(t, db, collidedKeys, true)
	checkKeys(t, db, otherKeys, true)
	prefixes, keyLength := rawdb.DeprecatedPrefixesV0()
	err = deleteWasmEntries(db, prefixes, true, keyLength)
	if err != nil {
		t.Fatal("Failed to purge version 0 keys, err:", err)
	}
	checkKeys(t, db, version0Keys, false)
	checkKeys(t, db, collidedKeys, true)
	checkKeys(t, db, otherKeys, true)
}

func TestOpenInitializeExecutionDbEmptyInit(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	stackConfig.DBEngine = rawdb.DBPebble
	stack, err := node.New(stackConfig)
	Require(t, err)
	defer stack.Close()

	nodeConfig := config.NodeConfigDefault
	nodeConfig.Execution.Caching.StateScheme = env.GetTestStateScheme()
	nodeConfig.Chain.ID = 42161
	nodeConfig.Node = *arbnode.ConfigDefaultL2Test()
	nodeConfig.Init.Empty = true
	nodeConfig.Init.ValidateGenesisAssertion = false

	l1Client := ethclient.NewClient(stack.Attach())

	executionDB, _, blockchain, err := OpenInitializeExecutionDB(
		ctx,
		stack,
		&nodeConfig,
		new(big.Int).SetUint64(nodeConfig.Chain.ID),
		gethexec.DefaultCacheConfigFor(&nodeConfig.Execution.Caching),
		nil,
		&nodeConfig.Persistent,
		l1Client,
		chaininfo.RollupAddresses{},
	)
	Require(t, err)
	blockchain.Stop()
	err = executionDB.Close()
	Require(t, err)
}

func TestExtractSnapshot(t *testing.T) {
	testCases := []struct {
		name         string
		archiveFiles []string
		importWasm   bool
		wantFiles    []string
	}{
		{
			name:       "extractAll",
			importWasm: true,
			archiveFiles: []string{
				"arbitrumdata/000001.ldb",
				"l2chaindata/000001.ldb",
				"l2chaindata/ancients/000001.ldb",
				"nodes/000001.ldb",
				"wasm/000001.ldb",
			},
			wantFiles: []string{
				"arbitrumdata/000001.ldb",
				"l2chaindata/000001.ldb",
				"l2chaindata/ancients/000001.ldb",
				"nodes/000001.ldb",
				"wasm/000001.ldb",
			},
		},
		{
			name:       "extractAllButWasm",
			importWasm: false,
			archiveFiles: []string{
				"arbitrumdata/000001.ldb",
				"l2chaindata/000001.ldb",
				"nodes/000001.ldb",
				"wasm/000001.ldb",
			},
			wantFiles: []string{
				"arbitrumdata/000001.ldb",
				"l2chaindata/000001.ldb",
				"nodes/000001.ldb",
			},
		},
		{
			name:       "extractAllButWasmWithPrefixDot",
			importWasm: false,
			archiveFiles: []string{
				"./arbitrumdata/000001.ldb",
				"./l2chaindata/000001.ldb",
				"./nodes/000001.ldb",
				"./wasm/000001.ldb",
			},
			wantFiles: []string{
				"arbitrumdata/000001.ldb",
				"l2chaindata/000001.ldb",
				"nodes/000001.ldb",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Create archive with dummy files
			archiveDir := t.TempDir()
			archivePath := path.Join(archiveDir, "archive.tar")
			{
				// Create context to close the file handlers
				archiveFile, err := os.Create(archivePath)
				Require(t, err)
				defer archiveFile.Close()
				tarWriter := tar.NewWriter(archiveFile)
				defer tarWriter.Close()
				for _, relativePath := range testCase.archiveFiles {
					filePath := path.Join(archiveDir, relativePath)
					dir := filepath.Dir(filePath)
					const dirPerm = 0700
					err := os.MkdirAll(dir, dirPerm)
					Require(t, err)
					const filePerm = 0600
					err = os.WriteFile(filePath, []byte{0xbe, 0xef}, filePerm)
					Require(t, err)
					file, err := os.Open(filePath)
					Require(t, err)
					info, err := file.Stat()
					Require(t, err)
					header, err := tar.FileInfoHeader(info, "")
					Require(t, err)
					header.Name = relativePath
					err = tarWriter.WriteHeader(header)
					Require(t, err)
					_, err = io.Copy(tarWriter, file)
					Require(t, err)
				}
			}

			// Extract archive and compare contents
			targetDir := t.TempDir()
			err := extractSnapshot(archivePath, targetDir, testCase.importWasm)
			Require(t, err, "failed to extract snapshot")
			gotFiles := []string{}
			err = filepath.WalkDir(targetDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					gotFiles = append(gotFiles, path)
				}
				return nil
			})
			Require(t, err)
			slices.Sort(gotFiles)
			for i, f := range testCase.wantFiles {
				testCase.wantFiles[i] = path.Join(targetDir, f)
			}
			if diff := cmp.Diff(gotFiles, testCase.wantFiles); diff != "" {
				t.Fatal("extracted files don't match", diff)
			}
		})
	}
}

func TestIsWasmDb(t *testing.T) {
	testCases := []struct {
		path string
		want bool
	}{
		{"wasm", true},
		{"wasm/", true},
		{"wasm/something", true},
		{"/wasm", true},
		{"./wasm", true},
		{"././wasm", true},
		{"/./wasm", true},
		{"WASM", true},
		{"wAsM", true},
		{"nitro/../wasm", true},
		{"/nitro/../wasm", true},
		{".//nitro/.//../wasm", true},
		{"not-wasm", false},
		{"l2chaindata/example@@", false},
		{"somedir/wasm", false},
	}
	for _, testCase := range testCases {
		name := fmt.Sprintf("%q", testCase.path)
		t.Run(name, func(t *testing.T) {
			got := isWasmDB(testCase.path)
			if testCase.want != got {
				t.Fatalf("want %v, but got %v", testCase.want, got)
			}
		})
	}
}

func TestInitConfigMustNotBeEmptyWhenGenesisJsonIsPresent(t *testing.T) {
	initConfig := conf.InitConfigDefault
	initConfig.GenesisJsonFile = "./genesis.json"
	initConfig.Empty = true
	err := initConfig.Validate()
	if err == nil {
		t.Fatal("expected error when both GenesisJsonFile and Empty are set")
	}
	if !strings.Contains(err.Error(), "init config cannot be both empty and have a genesis json file specified") {
		t.Fatal("expected conflict detection")
	}
}

func TestGetGenesisFileNameFromDirectoryWithCorrectFile(t *testing.T) {
	tempDir := t.TempDir()
	chainId := uint64(42161)
	genesisFileName := fmt.Sprintf("%d.json", chainId)
	genesisFilePath := tempDir + "/" + genesisFileName
	chainConfigSerialized, err := json.Marshal(params.ChainConfig{
		ChainID: big.NewInt(int64(chainId)),
	})
	Require(t, err)
	genesis := core.Genesis{
		SerializedChainConfig: string(chainConfigSerialized),
		GasLimit:              0,
		Difficulty:            big.NewInt(0),
		Alloc:                 core.GenesisAlloc{},
	}
	genesisBytes, err := genesis.MarshalJSON()
	Require(t, err)
	err = os.WriteFile(genesisFilePath, genesisBytes, 0600)
	Require(t, err)
	result, err := GetGenesisFileNameFromDirectory(tempDir, chainId)
	Require(t, err)
	if result != genesisFilePath {
		t.Fatalf("expected %s, got %s", genesisFilePath, result)
	}
}

func TestGetGenesisFileNameFromDirectoryWithWrongFileName(t *testing.T) {
	tempDir := t.TempDir()
	chainId := uint64(42161)
	genesisFileName := fmt.Sprintf("%d_wrong.json", chainId)
	genesisFilePath := tempDir + "/" + genesisFileName
	chainConfigSerialized, err := json.Marshal(params.ChainConfig{
		ChainID: big.NewInt(int64(chainId)),
	})
	Require(t, err)
	genesis := core.Genesis{
		SerializedChainConfig: string(chainConfigSerialized),
		GasLimit:              0,
		Difficulty:            big.NewInt(0),
		Alloc:                 core.GenesisAlloc{},
	}
	genesisBytes, err := genesis.MarshalJSON()
	Require(t, err)
	err = os.WriteFile(genesisFilePath, genesisBytes, 0600)
	Require(t, err)
	_, err = GetGenesisFileNameFromDirectory(tempDir, chainId)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetGenesisFileNameFromDirectoryWithWrongChainId(t *testing.T) {
	tempDir := t.TempDir()
	chainId := uint64(42161)
	wrongChainId := uint64(42162)
	genesisFileName := fmt.Sprintf("%d.json", chainId)
	genesisFilePath := tempDir + "/" + genesisFileName
	chainConfigSerialized, err := json.Marshal(params.ChainConfig{
		ChainID: big.NewInt(int64(wrongChainId)),
	})
	Require(t, err)
	genesis := core.Genesis{
		SerializedChainConfig: string(chainConfigSerialized),
		GasLimit:              0,
		Difficulty:            big.NewInt(0),
		Alloc:                 core.GenesisAlloc{},
	}
	genesisBytes, err := genesis.MarshalJSON()
	Require(t, err)
	err = os.WriteFile(genesisFilePath, genesisBytes, 0600)
	Require(t, err)
	_, err = GetGenesisFileNameFromDirectory(tempDir, chainId)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSimpleCheckDBDir(t *testing.T) {
	t.Parallel()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	stackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	stackConfig.DBEngine = rawdb.DBPebble
	stack, err := node.New(stackConfig)
	Require(t, err)
	defer stack.Close()

	nodeConfig := config.NodeConfigDefault

	err = checkDBDir(stack, &nodeConfig)
	Require(t, err)

}

func TestCheckDBDirReturnsErrorOnl2chaindataWrongDir(t *testing.T) {
	t.Parallel()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	rootTargetDir := t.TempDir()
	targetDir := filepath.Join(rootTargetDir, "do_not_exist")

	stackConfig := testhelpers.CreateStackConfigForTest(targetDir)
	stack, err := node.New(stackConfig)
	Require(t, err)
	defer stack.Close()

	// We create a l2chaindata on the data directory to simulate putting it in the wrong place
	instdir := filepath.Join(targetDir, "l2chaindata")
	err = os.MkdirAll(instdir, 0700)
	Require(t, err)

	nodeConfig := config.NodeConfigDefault

	err = checkDBDir(stack, &nodeConfig)
	require.Error(t, err)
	require.ErrorContains(t, err, "have you placed the database in the wrong directory?")
}

func TestCheckAndDownloadDBNoSnapshot(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	stack, err := node.New(stackConfig)
	Require(t, err)
	defer stack.Close()

	nodeConfig := config.NodeConfigDefault

	err = checkAndDownloadDB(ctx, stack, &nodeConfig)
	Require(t, err)
}

func getInitHelper(t *testing.T, ownerAdress string, chainID uint64, emptyState bool, importFile, genesisJsonFile string, useDevInit, skipInitDataReader bool) (statetransfer.InitDataReader, *params.ChainConfig, *params.ArbOSInit, ethdb.Database, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	stackConfig.DBEngine = rawdb.DBPebble
	stack, err := node.New(stackConfig)
	Require(t, err)
	cleanup := func() {
		stack.Close()
	}

	nodeConfig := config.NodeConfigDefault
	nodeConfig.Execution.Caching.StateScheme = rawdb.PathScheme
	nodeConfig.Chain.ID = chainID
	nodeConfig.Node = *arbnode.ConfigDefaultL2Test()
	if emptyState {
		nodeConfig.Init.Empty = emptyState
	}

	if importFile != "" {
		nodeConfig.Init.ImportFile = importFile
	}
	if genesisJsonFile != "" {
		nodeConfig.Init.GenesisJsonFile = genesisJsonFile
	}

	if useDevInit {
		nodeConfig.Init.DevInit = true
	}

	nodeConfig.Init.DevInitAddress = ownerAdress
	nodeConfig.Init.ValidateGenesisAssertion = false

	l1Client := ethclient.NewClient(stack.Attach())

	executionDB, _, _, err := OpenInitializeExecutionDB(
		ctx,
		stack,
		&nodeConfig,
		new(big.Int).SetUint64(nodeConfig.Chain.ID),
		gethexec.DefaultCacheConfigFor(&nodeConfig.Execution.Caching),
		nil,
		&nodeConfig.Persistent,
		l1Client,
		chaininfo.RollupAddresses{},
	)
	Require(t, err)

	// This means no init method is supplied to GetInit
	if skipInitDataReader {
		nodeConfig.Init.Empty = false
		nodeConfig.Init.ImportFile = ""
		nodeConfig.Init.GenesisJsonFile = ""
		nodeConfig.Init.DevInit = false
	}

	// We already call getInit once inside openInitializeExecutionDB but calling a
	// second time is okay since we're just loading configs
	initDataReader, chainConfig, arbOsInit, err := GetInit(&nodeConfig, executionDB)
	return initDataReader, chainConfig, arbOsInit, executionDB, cleanup, err

}

func TestSimpleGetInit(t *testing.T) {
	t.Parallel()

	ownerAdress := "0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E"
	expectedChainConfig := chaininfo.ArbitrumDevTestChainConfig()
	initDataReader, chainConfig, arbOsInit, _, cleanup, err := getInitHelper(t, ownerAdress, expectedChainConfig.ChainID.Uint64(), false, "", "", true, false)
	Require(t, err)
	defer cleanup()

	if chainConfig == nil {
		t.Fatalf("Expected chainConfig to be non nil")
	}

	require.Equal(t, expectedChainConfig, chainConfig)

	if arbOsInit != nil {
		t.Fatalf("Expected nil arbOsInit but got  = %v", arbOsInit)
	}

	if initDataReader == nil {
		t.Fatalf("initDataReader shouldn't be nil")
	}

	chainOwner, err := initDataReader.GetChainOwner()
	Require(t, err)

	expectedOwnerAddress := common.HexToAddress(ownerAdress)
	if chainOwner != expectedOwnerAddress {
		t.Fatalf("chainOwner address %s does not match expected address: %s", chainOwner.Hex(), expectedOwnerAddress.Hex())
	}

	blockNumber, err := initDataReader.GetNextBlockNumber()
	Require(t, err)

	if blockNumber != 0 {
		t.Fatalf("GetNextBlockNumber expected to return 0 but returned: %d", blockNumber)
	}

	err = initDataReader.Close()
	Require(t, err)
}

// Tests GetInit by not setting any init method. In which case GetInit would
// return a nil initDataReader with a chainConfig read using TryReadStoredChainConfig
func TestGetInitSkipInitDataReader(t *testing.T) {
	t.Parallel()

	ownerAdress := "0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E"
	expectedChainConfig := chaininfo.ArbitrumRollupGoerliTestnetChainConfig()
	initDataReader, chainConfig, arbOsInit, _, cleanup, err := getInitHelper(t, ownerAdress, expectedChainConfig.ChainID.Uint64(), false, "", "", true, true)
	Require(t, err)
	defer cleanup()

	if chainConfig == nil {
		t.Fatalf("Expected chainConfig to be non nil")
	}

	require.Equal(t, expectedChainConfig, chainConfig)

	if arbOsInit != nil {
		t.Fatalf("Expected nil arbOsInit but got  = %v", arbOsInit)
	}

	if initDataReader != nil {
		t.Fatalf("initDataReader expected to be nil")
	}
}

func TestGetInitWithEmpty(t *testing.T) {
	t.Parallel()

	ownerAdress := "0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E"
	expectedChainConfig := chaininfo.ArbitrumOneChainConfig()
	initDataReader, chainConfig, arbOsInit, _, cleanup, err := getInitHelper(t, ownerAdress, expectedChainConfig.ChainID.Uint64(), true, "", "", false, false)
	Require(t, err)
	defer cleanup()

	if chainConfig == nil {
		t.Fatalf("Expected chainConfig to be non nil")
	}

	// needed since ArbOne chain genesis is non zero
	expectedChainConfig.ArbitrumChainParams.GenesisBlockNum = config.NodeConfigDefault.Init.DevInitBlockNum
	require.Equal(t, expectedChainConfig, chainConfig)

	if arbOsInit != nil {
		t.Fatalf("Expected nil arbOsInit but got  = %v", arbOsInit)
	}

	if initDataReader == nil {
		t.Fatalf("initDataReader shouldn't be nil")
	}

	chainOwner, err := initDataReader.GetChainOwner()
	Require(t, err)

	// initData is mostly empty when Init.Empty is set to true therefore we never set owner
	// address so we should expect it to be the zero address
	emptyAdress := "0x0000000000000000000000000000000000000000"
	expectedOwnerAddress := common.HexToAddress(emptyAdress)
	if chainOwner != expectedOwnerAddress {
		t.Fatalf("chainOwner address %s does not match expected empty address: %s", chainOwner.Hex(), expectedOwnerAddress.Hex())
	}

	blockNumber, err := initDataReader.GetNextBlockNumber()
	Require(t, err)

	if blockNumber != 0 {
		t.Fatalf("GetNextBlockNumber expected to return 0 but returned: %d", blockNumber)
	}

	err = initDataReader.Close()
	Require(t, err)
}

func TestGetInitWithImportFile(t *testing.T) {
	t.Parallel()

	ownerAdress := "0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E"
	importFile := "testdata/initFileContent.json"
	expectedChainConfig := chaininfo.ArbitrumDevTestAnyTrustChainConfig()
	initDataReader, chainConfig, arbOsInit, _, cleanup, err := getInitHelper(t, ownerAdress, expectedChainConfig.ChainID.Uint64(), false, importFile, "", false, false)
	Require(t, err)
	defer cleanup()

	if chainConfig == nil {
		t.Fatalf("Expected chainConfig to be non nil")
	}

	require.Equal(t, expectedChainConfig, chainConfig)

	if arbOsInit != nil {
		t.Fatalf("Expected nil arbOsInit but got  = %v", chainConfig)
	}

	if initDataReader == nil {
		t.Fatalf("initDataReader shouldn't be nil")
	}

	chainOwner, err := initDataReader.GetChainOwner()
	Require(t, err)

	// JsonInitDataReader always returns empty owner address
	emptyAdress := "0x0000000000000000000000000000000000000000"
	expectedOwnerAddress := common.HexToAddress(emptyAdress)
	if chainOwner != expectedOwnerAddress {
		t.Fatalf("chainOwner address %s does not match expected empty address: %s", chainOwner.Hex(), expectedOwnerAddress.Hex())
	}

	blockNumber, err := initDataReader.GetNextBlockNumber()
	Require(t, err)

	if blockNumber != 0 {
		t.Fatalf("GetNextBlockNumber expected to return 100 but returned: %d", blockNumber)
	}

	err = initDataReader.Close()
	Require(t, err)
}

func TestGetInitWithGenesis(t *testing.T) {
	t.Parallel()

	ownerAdress := "0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E"
	genesisJsonFile := "testdata/testGenesis.json"
	expectedChainIdNum := uint64(3503995874084926)
	initDataReader, chainConfig, arbOsInit, _, cleanup, err := getInitHelper(t, ownerAdress, expectedChainIdNum, false, "", genesisJsonFile, false, false)
	Require(t, err)
	defer cleanup()

	if chainConfig == nil {
		t.Fatalf("Expected non nil chainConfig")
	}

	// First make sure some key fields have the expected value
	expectedChainId := new(big.Int).SetUint64(expectedChainIdNum)
	if chainConfig.ChainID.Uint64() != expectedChainId.Uint64() {
		t.Fatalf("chainConfig chainID %d does not match expected chain ID: %d", chainConfig.ChainID, expectedChainId)
	}
	if *chainConfig.CancunTime != 60 {
		t.Fatalf("expected chainConfig.CancunTime to be 60 but got: %d", *chainConfig.CancunTime)
	}

	if *chainConfig.PragueTime != 120 {
		t.Fatalf("expected chainConfig.PragueTime to be 120 but got: %d", *chainConfig.PragueTime)
	}

	// Make sure getInitHelper read the correct genesis file with all its fields
	genesisJson, err := os.ReadFile(genesisJsonFile)
	Require(t, err)
	var gen core.Genesis
	err = json.Unmarshal(genesisJson, &gen)
	Require(t, err)
	expectedChainConfig, err := gen.GetConfig()
	Require(t, err)

	require.Equal(t, expectedChainConfig, chainConfig)

	if arbOsInit != nil {
		t.Fatalf("arbOsInit expected to be nil")
	}

	if initDataReader == nil {
		t.Fatalf("initDataReader shouldn't be nil")
	}

	chainOwner, err := initDataReader.GetChainOwner()
	Require(t, err)

	// We never init owner address when GenesisJsonFile != "", therefore we should expect the zero address
	emptyAdress := "0x0000000000000000000000000000000000000000"
	expectedOwnerAddress := common.HexToAddress(emptyAdress)
	if chainOwner != expectedOwnerAddress {
		t.Fatalf("chainOwner address %s does not match expected empty address: %s", chainOwner.Hex(), expectedOwnerAddress.Hex())
	}

	blockNumber, err := initDataReader.GetNextBlockNumber()
	Require(t, err)

	if blockNumber != 0 {
		t.Fatalf("GetNextBlockNumber expected to return 0 but returned: %d", blockNumber)
	}

	err = initDataReader.Close()
	Require(t, err)
}

func TestGetInitWithChainconfigInDB(t *testing.T) {
	t.Parallel()

	// Force getInitHelper to store chainConfig to DB (similar to TestGetInitSkipInitDataReader)
	ownerAdress := "0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E"
	expectedChainConfig := chaininfo.ArbitrumRollupGoerliTestnetChainConfig()
	initDataReader, chainConfig, arbOsInit, executionDB, cleanup, err := getInitHelper(t, ownerAdress, expectedChainConfig.ChainID.Uint64(), false, "", "", true, true)
	Require(t, err)
	defer cleanup()

	if chainConfig == nil {
		t.Fatalf("Expected chainConfig to be non nil")
	}

	require.Equal(t, expectedChainConfig, chainConfig)

	if arbOsInit != nil {
		t.Fatalf("Expected nil arbOsInit but got  = %v", arbOsInit)
	}

	if initDataReader != nil {
		t.Fatalf("initDataReader expected to be nil")
	}

	// Call GetInit with a different chainID and make sure we still read chainConfig from DB
	nodeConfig := config.NodeConfigDefault
	nodeConfig.Execution.Caching.StateScheme = rawdb.PathScheme
	nodeConfig.Chain.ID = 4444
	nodeConfig.Node = *arbnode.ConfigDefaultL2Test()
	_, chainConfig, _, err = GetInit(&nodeConfig, executionDB)
	Require(t, err)

	// Make sure chainConfig that was read from DB still matches expected chainConfig
	require.Equal(t, expectedChainConfig, chainConfig)
}

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func gracefulShutdown(t *testing.T, ctx context.Context, server *http.Server) {
	if err := server.Shutdown(ctx); err != nil {
		t.Logf("HTTP server shutdown error: %v", err)
	}
}
