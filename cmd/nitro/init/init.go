// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package nitroinit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/codeclysm/extract/v3"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/bold/protocol"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/nitro/config"
	"github.com/offchainlabs/nitro/cmd/pruning"
	"github.com/offchainlabs/nitro/cmd/staterecovery"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/dbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
)

var errNotFound = errors.New("file not found")

// taken from wasmer's lib/types/src/serialize.rs: MetadataHeader::CURRENT_VERSION
// 8 is a bug (should have been 6) but we're skipping the real version 8 so it does not matter
const WasmerSerializeVersion = 8
const InitialWasmerSerializeVersion = 8

func initializeAndDownloadInit(ctx context.Context, initConfig *conf.InitConfig, stack *node.Node) (string, func(), error) {
	cleanUpTmp := func() {}
	if initConfig.DownloadPath == "" {
		tmpPath := filepath.Join(stack.InstanceDir(), "tmp")
		_, err := os.Stat(tmpPath)
		if err == nil {
			return "", cleanUpTmp, fmt.Errorf("tmp directory for downloading init file already exists")
		}
		if !os.IsNotExist(err) {
			return "", cleanUpTmp, fmt.Errorf("error checking if tmp directory for downloading init file already exists: %w", err)
		}
		if err := os.MkdirAll(tmpPath, os.ModePerm); err != nil {
			return "", cleanUpTmp, fmt.Errorf("failed to create tmp directory for downloading init file: %w", err)
		}
		initConfig.DownloadPath = tmpPath
		cleanUpTmp = func() {
			if err := os.RemoveAll(tmpPath); err != nil {
				log.Error("Failed to clean up tmp directory after downloading init file", "err", err)
			}
		}
	}
	initFile, err := downloadInit(ctx, initConfig)
	if err != nil {
		return "", cleanUpTmp, err
	}
	return initFile, cleanUpTmp, nil
}

func downloadInit(ctx context.Context, initConfig *conf.InitConfig) (string, error) {
	if initConfig.Url == "" {
		return "", nil
	}
	if strings.HasPrefix(initConfig.Url, "file:") {
		return initConfig.Url[5:], nil
	}
	log.Info("Downloading initial database", "url", initConfig.Url)
	if !initConfig.ValidateChecksum {
		file, err := downloadFile(ctx, initConfig, initConfig.Url, nil)
		if err != nil && errors.Is(err, errNotFound) {
			return downloadInitInParts(ctx, initConfig)
		}
		return file, err
	}
	checksum, err := fetchChecksum(ctx, initConfig.Url+".sha256")
	if err != nil {
		if errors.Is(err, errNotFound) {
			return downloadInitInParts(ctx, initConfig)
		}
		return "", fmt.Errorf("error fetching checksum: %w", err)
	}
	file, err := downloadFile(ctx, initConfig, initConfig.Url, checksum)
	if err != nil && errors.Is(err, errNotFound) {
		return "", fmt.Errorf("file not found but checksum exists")
	}
	return file, err
}

func downloadFile(ctx context.Context, initConfig *conf.InitConfig, url string, checksum []byte) (string, error) {
	grabclient := grab.NewClient()
	printTicker := time.NewTicker(time.Second)
	defer printTicker.Stop()
	attempt := 0
	for {
		attempt++
		req, err := grab.NewRequest(initConfig.DownloadPath, url)
		if err != nil {
			panic(err)
		}
		if checksum != nil {
			const deleteOnError = true
			req.SetChecksum(sha256.New(), checksum, deleteOnError)
		}
		resp := grabclient.Do(req.WithContext(ctx))
		firstPrintTime := time.Now().Add(time.Second * 2)
	updateLoop:
		for {
			select {
			case <-printTicker.C:
				if time.Now().After(firstPrintTime) {
					bps := resp.BytesPerSecond()
					if bps == 0 {
						bps = 1 // avoid division by zero
					}
					done := resp.BytesComplete()
					total := resp.Size()
					timeRemaining := time.Second * (time.Duration(total-done) / time.Duration(bps))
					timeRemaining = timeRemaining.Truncate(time.Millisecond * 10)
					fmt.Printf("\033[2K\r  transferred %v / %v bytes (%.2f%%) [%.2fMbps, %s remaining]",
						done,
						total,
						resp.Progress()*100,
						bps*8/1000000,
						timeRemaining.String())
				}
			case <-resp.Done:
				if err := resp.Err(); err != nil {
					if resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode == http.StatusNotFound {
						return "", errNotFound
					}
					fmt.Printf("\n  attempt %d failed: %v\n", attempt, err)
					break updateLoop
				}
				fmt.Printf("\n")
				log.Info("Download done", "filename", resp.Filename, "duration", resp.Duration())
				fmt.Println()
				return resp.Filename, nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(initConfig.DownloadPoll):
		}
	}
}

// httpGet performs a GET request to the specified URL
func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making GET request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errNotFound
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %v", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	return body, nil
}

// fetchChecksum performs a GET request to the specified URL and returns the checksum
func fetchChecksum(ctx context.Context, url string) ([]byte, error) {
	body, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}
	checksumStr := strings.TrimSpace(string(body))
	checksum, err := hex.DecodeString(checksumStr)
	if err != nil {
		return nil, fmt.Errorf("error decoding checksum: %w", err)
	}
	if len(checksum) != sha256.Size {
		return nil, fmt.Errorf("invalid checksum length")
	}
	return checksum, nil
}

func downloadInitInParts(ctx context.Context, initConfig *conf.InitConfig) (string, error) {
	log.Info("File not found; trying to download database in parts")
	fileInfo, err := os.Stat(initConfig.DownloadPath)
	if err != nil || !fileInfo.IsDir() {
		return "", fmt.Errorf("download path must be a directory: %v", initConfig.DownloadPath)
	}
	archiveUrl, err := url.Parse(initConfig.Url)
	if err != nil {
		return "", fmt.Errorf("failed to parse init url \"%s\": %w", initConfig.Url, err)
	}

	// Get parts from manifest file
	manifest, err := httpGet(ctx, archiveUrl.String()+".manifest.txt")
	if err != nil {
		return "", fmt.Errorf("failed to get manifest file: %w", err)
	}
	partNames := []string{}
	checksums := [][]byte{}
	lines := strings.Split(strings.TrimSpace(string(manifest)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return "", fmt.Errorf("manifest file in wrong format")
		}
		checksum, err := hex.DecodeString(fields[0])
		if err != nil {
			return "", fmt.Errorf("failed decoding checksum in manifest file: %w", err)
		}
		checksums = append(checksums, checksum)
		partNames = append(partNames, fields[1])
	}

	partFiles := []string{}
	defer func() {
		// remove all temporary files.
		for _, part := range partFiles {
			err := os.Remove(part)
			if err != nil {
				log.Warn("Failed to remove temporary file", "file", part)
			}
		}
	}()

	// Download parts
	for i, partName := range partNames {
		log.Info("Downloading database part", "part", partName)
		partUrl := archiveUrl.JoinPath("..", partName).String()
		var checksum []byte
		if initConfig.ValidateChecksum {
			checksum = checksums[i]
		}
		partFile, err := downloadFile(ctx, initConfig, partUrl, checksum)
		if err != nil {
			return "", fmt.Errorf("error downloading part \"%s\": %w", partName, err)
		}
		partFiles = append(partFiles, partFile)
	}
	archivePath := path.Join(initConfig.DownloadPath, path.Base(archiveUrl.Path))
	return joinArchive(partFiles, archivePath)
}

// joinArchive joins the archive parts into a single file and return its path.
func joinArchive(parts []string, archivePath string) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("no database parts found")
	}
	archive, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to create archive: %w", err)
	}
	defer archive.Close()
	for _, part := range parts {
		partFile, err := os.Open(part)
		if err != nil {
			return "", fmt.Errorf("failed to open part file %s: %w", part, err)
		}
		_, err = io.Copy(archive, partFile)
		if err != nil {
			partFile.Close()
			return "", fmt.Errorf("failed to copy part file %s: %w", part, err)
		}
		partFile.Close()
		log.Info("Joined database part into archive", "part", part)
	}
	log.Info("Successfully joined parts into archive", "archive", archivePath)
	return archivePath, nil
}

// setLatestSnapshotUrl sets the Url in initConfig to the latest one available on the mirror.
func setLatestSnapshotUrl(ctx context.Context, initConfig *conf.InitConfig, chain string) error {
	if initConfig.Latest == "" {
		return nil
	}
	if initConfig.Url != "" {
		return fmt.Errorf("cannot set latest url if url is already set")
	}
	baseUrl, err := url.Parse(initConfig.LatestBase)
	if err != nil {
		return fmt.Errorf("failed to parse latest mirror \"%s\": %w", initConfig.LatestBase, err)
	}
	latestFileUrl := baseUrl.JoinPath(chain, "latest-"+initConfig.Latest+".txt").String()
	latestFileUrl = strings.ToLower(latestFileUrl)
	latestFileBytes, err := httpGet(ctx, latestFileUrl)
	if err != nil {
		return fmt.Errorf("failed to get latest file at \"%s\": %w", latestFileUrl, err)
	}
	latestFile := strings.TrimSpace(string(latestFileBytes))
	containsScheme := regexp.MustCompile("https?://")
	if containsScheme.MatchString(latestFile) {
		initConfig.Url = latestFile
	} else {
		initConfig.Url = baseUrl.JoinPath(latestFile).String()
	}
	initConfig.Url = strings.ToLower(initConfig.Url)
	log.Info("Set latest snapshot url", "url", initConfig.Url)
	return nil
}

func ValidateBlockChain(blockChain *core.BlockChain, chainConfig *params.ChainConfig) error {
	statedb, err := blockChain.State()
	if err != nil {
		return err
	}
	currentArbosState, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	if err != nil {
		return err
	}
	chainId, err := currentArbosState.ChainId()
	if err != nil {
		return err
	}
	if chainId.Cmp(chainConfig.ChainID) != 0 {
		return fmt.Errorf("attempted to launch node with chain ID %v on ArbOS state with chain ID %v", chainConfig.ChainID, chainId)
	}
	oldSerializedConfig, err := currentArbosState.ChainConfig()
	if err != nil {
		return fmt.Errorf("failed to get old chain config from ArbOS state: %w", err)
	}
	if len(oldSerializedConfig) != 0 {
		var oldConfig params.ChainConfig
		err = json.Unmarshal(oldSerializedConfig, &oldConfig)
		if err != nil {
			return fmt.Errorf("failed to deserialize old chain config: %w", err)
		}
		currentBlock := blockChain.CurrentBlock()
		if currentBlock == nil {
			return errors.New("failed to get current block")
		}
		if err := oldConfig.CheckCompatible(chainConfig, currentBlock.Number.Uint64(), currentBlock.Time); err != nil {
			return fmt.Errorf("invalid chain config, not compatible with previous: %w", err)
		}
	}
	// Make sure we don't allow accidentally downgrading ArbOS
	if chainConfig.DebugMode() {
		if currentArbosState.ArbOSVersion() > params.MaxDebugArbosVersionSupported {
			return fmt.Errorf("attempted to launch node in debug mode with ArbOS version %v on ArbOS state with version %v", params.MaxDebugArbosVersionSupported, currentArbosState.ArbOSVersion())
		}
	} else {
		if currentArbosState.ArbOSVersion() > params.MaxArbosVersionSupported {
			return fmt.Errorf("attempted to launch node with ArbOS version %v on ArbOS state with version %v", params.MaxArbosVersionSupported, currentArbosState.ArbOSVersion())
		}

	}

	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func checkEmptyDatabaseDir(dir string, force bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to open database dir %s: %w", dir, err)
	}
	unexpectedFiles := []string{}
	allowedFiles := map[string]bool{
		"LOCK": true, "classic-msg": true, "l2chaindata": true,
	}
	for _, entry := range entries {
		if !allowedFiles[entry.Name()] {
			unexpectedFiles = append(unexpectedFiles, entry.Name())
		}
	}
	if len(unexpectedFiles) > 0 {
		if force {
			return fmt.Errorf("trying to overwrite old database directory '%s' (delete the database directory and try again)", dir)
		}
		firstThreeFilenames := strings.Join(unexpectedFiles[:min(len(unexpectedFiles), 3)], ", ")
		return fmt.Errorf("found %d unexpected files in database directory, including: %s", len(unexpectedFiles), firstThreeFilenames)
	}
	return nil
}

func databaseIsEmpty(db ethdb.Database) bool {
	it := db.NewIterator(nil, nil)
	defer it.Release()
	return !it.Next()
}

func isWasmDB(path string) bool {
	path = strings.ToLower(path) // lowers the path to handle case-insensitive file systems
	path = filepath.Clean(path)
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) >= 1 && parts[0] == "wasm" {
		return true
	}
	if len(parts) >= 2 && parts[0] == "" && parts[1] == "wasm" { // Cover "/wasm" case
		return true
	}
	return false
}

func extractSnapshot(archive string, location string, importWasm bool) error {
	reader, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("couldn't open init '%v' archive: %w", archive, err)
	}
	defer reader.Close()
	stat, err := reader.Stat()
	if err != nil {
		return err
	}
	log.Info("extracting downloaded init archive", "size", fmt.Sprintf("%dMB", stat.Size()/1024/1024))
	var rename extract.Renamer
	if !importWasm {
		rename = func(path string) string {
			if isWasmDB(path) {
				return "" // do not extract wasm files
			}
			return path
		}
	}
	err = extract.Archive(context.Background(), reader, location, rename)
	if err != nil {
		return fmt.Errorf("couldn't extract init archive '%v' err: %w", archive, err)
	}
	return nil
}

func deleteWasmEntries(db ethdb.Database, prefixes [][]byte, checkKeyLength bool, expectedKeyLength int) error {
	batch := db.NewBatch()
	notMatchingLengthKeyLogged := false
	for _, prefix := range prefixes {
		it := db.NewIterator(prefix, nil)
		defer it.Release()
		for it.Next() {
			key := it.Key()
			if checkKeyLength && len(key) != expectedKeyLength {
				if !notMatchingLengthKeyLogged {
					log.Warn("Found key with deprecated prefix but not matching length, skipping removal. (this warning is logged only once)", "key", key)
					notMatchingLengthKeyLogged = true
				}
				continue
			}
			if err := batch.Delete(key); err != nil {
				return fmt.Errorf("failed to remove key %v : %w", key, err)
			}

			// Recreate the iterator after every batch commit in order
			// to allow the underlying compactor to delete the entries.
			if batch.ValueSize() >= ethdb.IdealBatchSize {
				if err := batch.Write(); err != nil {
					return fmt.Errorf("failed to write batch: %w", err)
				}
				batch.Reset()
				it.Release()
				it = db.NewIterator(prefix, key)
			}
		}
	}
	if batch.ValueSize() > 0 {
		if err := batch.Write(); err != nil {
			return fmt.Errorf("failed to write batch: %w", err)
		}
		batch.Reset()
	}
	return nil
}

func validateOrUpgradeWasmerSerializeVersion(db ethdb.Database) error {
	if !databaseIsEmpty(db) {
		versionInDB, err := rawdb.ReadWasmerSerializeVersion(db)
		if err != nil {
			if rawdb.IsDbErrNotFound(err) {
				versionInDB = InitialWasmerSerializeVersion
			} else {
				return fmt.Errorf("failed to retrieve wasmer serialize version: %w", err)
			}
		}
		if versionInDB != WasmerSerializeVersion {
			log.Warn("Detected wasmer serialize version %v, expected version %v - removing old wasm entries", versionInDB, WasmerSerializeVersion)
			prefixes := rawdb.WasmPrefixesExceptWavm()
			if err := deleteWasmEntries(db, prefixes, false, 0); err != nil {
				return fmt.Errorf("failed to purge wasm entries: %w", err)
			}
			log.Info("Wasm entries successfully removed.")
			err = rawdb.WriteWasmerSerializeVersion(db, WasmerSerializeVersion)
			if err != nil {
				return fmt.Errorf("failed to write wasmer serialize version: %w", err)
			}
		}
	}
	return nil
}

// if db is not empty, validates if wasm database schema version matches current version
// otherwise persists current version
func validateOrUpgradeWasmStoreSchemaVersion(db ethdb.Database) error {
	if !databaseIsEmpty(db) {
		version, err := rawdb.ReadWasmSchemaVersion(db)
		if err != nil {
			if rawdb.IsDbErrNotFound(err) {
				version = []byte{0}
			} else {
				return fmt.Errorf("failed to retrieve wasm schema version: %w", err)
			}
		}
		if len(version) != 1 || version[0] > rawdb.WasmSchemaVersion {
			return fmt.Errorf("unsupported wasm database schema version, current version: %v, read from wasm database: %v", rawdb.WasmSchemaVersion, version)
		}
		// special step for upgrading from version 0 - remove all entries added in version 0
		if version[0] == 0 {
			log.Warn("Detected wasm store schema version 0 - removing all old wasm store entries")
			prefixes, keyLength := rawdb.DeprecatedPrefixesV0()
			if err := deleteWasmEntries(db, prefixes, true, keyLength); err != nil {
				return fmt.Errorf("failed to purge wasm store version 0 entries: %w", err)
			}
			log.Info("Wasm store schema version 0 entries successfully removed.")
		}
	}
	rawdb.WriteWasmSchemaVersion(db)
	return nil
}

func rebuildLocalWasm(ctx context.Context, config *gethexec.Config, l2BlockChain *core.BlockChain, executionDB, wasmDB ethdb.Database, rebuildMode string) (ethdb.Database, *core.BlockChain, error) {
	var err error
	latestBlock := l2BlockChain.CurrentBlock()
	if latestBlock == nil || latestBlock.Number.Uint64() <= l2BlockChain.Config().ArbitrumChainParams.GenesisBlockNum ||
		types.DeserializeHeaderExtraInformation(latestBlock).ArbOSFormatVersion < params.ArbosVersion_Stylus {
		// If there is only genesis block or no blocks in the blockchain, set Rebuilding of wasm store to Done
		// If Stylus upgrade hasn't yet happened, skipping rebuilding of wasm store
		log.Info("Setting rebuilding of wasm store to done")
		if err = gethexec.WriteToKeyValueStore(wasmDB, gethexec.RebuildingPositionKey, gethexec.RebuildingDone); err != nil {
			return nil, nil, fmt.Errorf("unable to set rebuilding status of wasm store to done: %w", err)
		}
	} else if rebuildMode != "false" {
		var position common.Hash
		if rebuildMode == "force" {
			log.Info("Commencing force rebuilding of wasm store by setting codehash position in rebuilding to beginning")
			if err := gethexec.WriteToKeyValueStore(wasmDB, gethexec.RebuildingPositionKey, common.Hash{}); err != nil {
				return nil, nil, fmt.Errorf("unable to initialize codehash position in rebuilding of wasm store to beginning: %w", err)
			}
		} else {
			position, err = gethexec.ReadFromKeyValueStore[common.Hash](wasmDB, gethexec.RebuildingPositionKey)
			if err != nil {
				log.Info("Unable to get codehash position in rebuilding of wasm store, its possible it isn't initialized yet, so initializing it and starting rebuilding", "err", err)
				if err := gethexec.WriteToKeyValueStore(wasmDB, gethexec.RebuildingPositionKey, common.Hash{}); err != nil {
					return nil, nil, fmt.Errorf("unable to initialize codehash position in rebuilding of wasm store to beginning: %w", err)
				}
			}
		}
		if position != gethexec.RebuildingDone {
			startBlockHash, err := gethexec.ReadFromKeyValueStore[common.Hash](wasmDB, gethexec.RebuildingStartBlockHashKey)
			if err != nil {
				log.Info("Unable to get start block hash in rebuilding of wasm store, its possible it isn't initialized yet, so initializing it to latest block hash", "err", err)
				if err := gethexec.WriteToKeyValueStore(wasmDB, gethexec.RebuildingStartBlockHashKey, latestBlock.Hash()); err != nil {
					return nil, nil, fmt.Errorf("unable to initialize start block hash in rebuilding of wasm store to latest block hash: %w", err)
				}
				startBlockHash = latestBlock.Hash()
			}
			log.Info("Starting or continuing rebuilding of wasm store", "codeHash", position, "startBlockHash", startBlockHash)
			if err := gethexec.RebuildWasmStore(ctx, wasmDB, executionDB, config.RPC.MaxRecreateStateDepth, &config.StylusTarget, l2BlockChain, position, startBlockHash); err != nil {
				return nil, nil, fmt.Errorf("error rebuilding of wasm store: %w", err)
			}
		}
	}
	return executionDB, l2BlockChain, nil
}

func OpenInitializeExecutionDB(ctx context.Context, stack *node.Node, config *config.NodeConfig, chainId *big.Int, cacheConfig *core.BlockChainConfig, tracer *tracing.Hooks, persistentConfig *conf.PersistentConfig, l1Client *ethclient.Client, rollupAddrs chaininfo.RollupAddresses) (ethdb.Database, statetransfer.InitDataReader, *core.BlockChain, error) {
	executionDB, wasmDB, l2BlockChain, chainConfig, err := OpenExistingExecutionDB(stack, config, chainId, cacheConfig, tracer, persistentConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	var initDataReader statetransfer.InitDataReader

	if executionDB == nil {
		err := checkAndDownloadDB(ctx, stack, config)
		if err != nil {
			return nil, nil, nil, err
		}

		executionDB, wasmDB, err = openDownloadedExecutionDB(stack, config, cacheConfig, persistentConfig)
		if err != nil {
			return nil, nil, nil, err
		}

		var genesisArbOSInit *params.ArbOSInit

		initDataReader, chainConfig, genesisArbOSInit, err = GetInit(config, executionDB)
		if err != nil {
			return executionDB, nil, nil, err
		}

		parsedInitMessage, err := GetConsensusParsedInitMsg(ctx, config.Node.ParentChainReader.Enable, chainId, l1Client, &rollupAddrs, chainConfig)
		if err != nil {
			return executionDB, nil, nil, err
		}

		l2BlockChain, err = getNewBlockchain(parsedInitMessage, config, initDataReader, chainConfig, genesisArbOSInit, executionDB, cacheConfig, tracer)
		if err != nil {
			return executionDB, nil, nil, err
		}
	}

	err = pruneExecutionDB(ctx, executionDB, stack, config, cacheConfig, persistentConfig, l1Client, rollupAddrs)
	if err != nil {
		return executionDB, nil, nil, fmt.Errorf("error pruning: %w", err)
	}

	err = ValidateBlockChain(l2BlockChain, chainConfig)
	if err != nil {
		return executionDB, nil, l2BlockChain, err
	}

	err = recreateMissingStates(config, executionDB, l2BlockChain, cacheConfig)
	if err != nil {
		return executionDB, nil, l2BlockChain, fmt.Errorf("failed to recreate missing states: %w", err)
	}

	executionDB, l2BlockChain, err = rebuildLocalWasm(ctx, &config.Execution, l2BlockChain, executionDB, wasmDB, config.Init.RebuildLocalWasm)

	return executionDB, initDataReader, l2BlockChain, err
}

func recreateMissingStates(config *config.NodeConfig, executionDB ethdb.Database, l2BlockChain *core.BlockChain, cacheConfig *core.BlockChainConfig) error {
	if config.Init.RecreateMissingStateFrom > 0 {
		err := staterecovery.RecreateMissingStates(executionDB, l2BlockChain, cacheConfig, config.Init.RecreateMissingStateFrom)
		if err != nil {
			return fmt.Errorf("failed to recreate missing states: %w", err)
		}
	}
	return nil
}

func pruneExecutionDB(ctx context.Context, executionDB ethdb.Database, stack *node.Node, config *config.NodeConfig, cacheConfig *core.BlockChainConfig, persistentConfig *conf.PersistentConfig, l1Client *ethclient.Client, rollupAddrs chaininfo.RollupAddresses) error {
	err := executionDB.SyncAncient()
	if err != nil {
		return err
	}

	return pruning.PruneExecutionDB(ctx, executionDB, stack, &config.Init, cacheConfig, persistentConfig, l1Client, rollupAddrs, config.Node.ValidatorRequired(), false)
}

func GetInit(config *config.NodeConfig, executionDB ethdb.Database) (statetransfer.InitDataReader, *params.ChainConfig, *params.ArbOSInit, error) {
	var (
		initDataReader   statetransfer.InitDataReader
		chainConfig      *params.ChainConfig
		genesisArbOSInit *params.ArbOSInit
		err              error
	)

	if config.Init.ImportFile != "" {
		initDataReader, err = statetransfer.NewJsonInitDataReader(config.Init.ImportFile)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error reading import file: %w", err)
		}
	}
	if config.Init.Empty {
		if initDataReader != nil {
			return nil, nil, nil, errors.New("multiple init methods supplied")
		}
		initData := statetransfer.ArbosInitializationInfo{
			NextBlockNumber: 0,
		}
		initDataReader = statetransfer.NewMemoryInitDataReader(&initData)
	}
	if config.Init.DevInit {
		if initDataReader != nil {
			return nil, nil, nil, errors.New("multiple init methods supplied")
		}
		initData := statetransfer.ArbosInitializationInfo{
			NextBlockNumber: config.Init.DevInitBlockNum,
			Accounts: []statetransfer.AccountInitializationInfo{
				{
					Addr:       common.HexToAddress(config.Init.DevInitAddress),
					EthBalance: new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(1000)),
					Nonce:      0,
				},
			},
			ChainOwner: common.HexToAddress(config.Init.DevInitAddress),
		}
		initDataReader = statetransfer.NewMemoryInitDataReader(&initData)
	}

	genesisJsonFile := config.Init.GenesisJsonFile
	if genesisJsonFile == "" {
		genesisJsonFile, err = GetGenesisFileNameFromDirectory(config.Init.GenesisJsonFileDirectory, config.Chain.ID)
		if err != nil {
			log.Error("error getting genesis json file from directory", "err", err)
		}
	}

	if genesisJsonFile != "" {
		if initDataReader != nil {
			return nil, nil, nil, errors.New("multiple init methods supplied")
		}
		genesisJson, err := os.ReadFile(genesisJsonFile)
		if err != nil {
			return nil, nil, nil, err
		}
		var gen core.Genesis
		if err := json.Unmarshal(genesisJson, &gen); err != nil {
			return nil, nil, nil, err
		}
		var accounts []statetransfer.AccountInitializationInfo
		for address, account := range gen.Alloc {
			accounts = append(accounts, statetransfer.AccountInitializationInfo{
				Addr:       address,
				EthBalance: account.Balance,
				Nonce:      account.Nonce,
				ContractInfo: &statetransfer.AccountInitContractInfo{
					Code:            account.Code,
					ContractStorage: account.Storage,
				},
			})
		}
		initDataReader = statetransfer.NewMemoryInitDataReader(&statetransfer.ArbosInitializationInfo{
			Accounts: accounts,
		})
		chainConfig, err = gen.GetConfig()
		if err != nil {
			return nil, nil, nil, err
		}
		genesisArbOSInit = gen.ArbOSInit
	} else {
		if initDataReader == nil {
			chainConfig = gethexec.TryReadStoredChainConfig(executionDB)
			if chainConfig == nil {
				return nil, nil, nil, errors.New("no --init.* mode supplied and chain data not in expected directory")
			}
		} else {
			genesisBlockNr, err := initDataReader.GetNextBlockNumber()
			if err != nil {
				return nil, nil, nil, err
			}
			chainConfig, err = chaininfo.GetChainConfig(new(big.Int).SetUint64(config.Chain.ID), config.Chain.Name, genesisBlockNr, config.Chain.InfoFiles, config.Chain.InfoJson)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}

	if initDataReader != nil && config.Init.DevInit && config.Init.DevMaxCodeSize != 0 {
		chainConfig.ArbitrumChainParams.MaxCodeSize = config.Init.DevMaxCodeSize
	}

	return initDataReader, chainConfig, genesisArbOSInit, nil
}

func GetGenesisFileNameFromDirectory(genesisFileDirectory string, chainId uint64) (string, error) {
	files, err := os.ReadDir(genesisFileDirectory)
	if err != nil {
		return "", fmt.Errorf("error reading genesis json file directory %s: %w", genesisFileDirectory, err)
	}
	requiredFileName := fmt.Sprintf("%d.json", chainId)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if file.Name() != requiredFileName {
			continue
		}

		fullPath := path.Join(genesisFileDirectory, file.Name())
		genesisJson, err := os.ReadFile(fullPath)
		if err != nil {
			log.Error("error reading genesis json file", "file", fullPath, "err", err)
			continue
		}
		var gen core.Genesis
		if err := json.Unmarshal(genesisJson, &gen); err != nil {
			log.Error("error unmarshaling genesis json file", "file", fullPath, "err", err)
			continue
		}
		if gen.Config == nil || gen.Config.ChainID == nil || gen.Config.ChainID.Uint64() != chainId {
			log.Error("genesis json file chain id does not match configured chain id", "file", fullPath, "genesisChainId", gen.Config.ChainID, "configuredChainId", chainId)
			continue
		}
		log.Info("found genesis json file for chain id from genesis json file directory", "file", fullPath, "chainId", chainId)
		return fullPath, nil
	}
	return "", fmt.Errorf("no genesis json file found for chain id %d in directory %s", chainId, genesisFileDirectory)
}

func getNewBlockchain(parsedInitMessage *arbostypes.ParsedInitMessage, config *config.NodeConfig, initDataReader statetransfer.InitDataReader, chainConfig *params.ChainConfig, genesisArbOSInit *params.ArbOSInit, executionDB ethdb.Database, cacheConfig *core.BlockChainConfig, tracer *tracing.Hooks) (*core.BlockChain, error) {
	var l2BlockChain *core.BlockChain
	txIndexWg := sync.WaitGroup{}

	if initDataReader == nil {
		l2BlockChain, err := gethexec.GetBlockChain(executionDB, cacheConfig, chainConfig, tracer, &config.Execution.TxIndexer)
		if err != nil {
			return nil, err
		}
		genesisBlockNr := chainConfig.ArbitrumChainParams.GenesisBlockNum
		genesisBlock := l2BlockChain.GetBlockByNumber(genesisBlockNr)
		if genesisBlock != nil {
			log.Info("loaded genesis block from database", "number", genesisBlockNr, "hash", genesisBlock.Hash())
		} else {
			// The node will probably die later, but might as well not kill it here?
			log.Error("database missing genesis block", "number", genesisBlockNr)
		}
		testUpdateTxIndex(executionDB, chainConfig, &txIndexWg)
	} else {
		genesisBlockNr, err := initDataReader.GetNextBlockNumber()
		if err != nil {
			return nil, err
		}
		testUpdateTxIndex(executionDB, chainConfig, &txIndexWg)
		ancients, err := executionDB.Ancients()
		if err != nil {
			return nil, err
		}
		if ancients < genesisBlockNr {
			return nil, fmt.Errorf("%v pre-init blocks required, but only %v found", genesisBlockNr, ancients)
		}
		if ancients > genesisBlockNr {
			storedGenHash := rawdb.ReadCanonicalHash(executionDB, genesisBlockNr)
			storedGenBlock := rawdb.ReadBlock(executionDB, storedGenHash, genesisBlockNr)
			if storedGenBlock.Header().Root == (common.Hash{}) {
				return nil, fmt.Errorf("attempting to init genesis block %x, but this block is in database with no state root", genesisBlockNr)
			}
			log.Warn("Re-creating genesis though it seems to exist in database", "blockNr", genesisBlockNr)
		}
		log.Info("Initializing", "ancients", ancients, "genesisBlockNr", genesisBlockNr)
		if config.Init.ThenQuit {
			cacheConfig.SnapshotWait = true
		}

		emptyBlockChain := rawdb.ReadHeadHeader(executionDB) == nil
		if !emptyBlockChain && (cacheConfig.StateScheme == rawdb.PathScheme) && config.Init.Force {
			return nil, errors.New("it is not possible to force init with non-empty blockchain when using path scheme")
		}
		l2BlockChain, err = gethexec.WriteOrTestBlockChain(executionDB, cacheConfig, initDataReader, chainConfig, genesisArbOSInit, tracer, parsedInitMessage, &config.Execution.TxIndexer, config.Init.AccountsPerSync)
		if err != nil {
			return nil, err
		}
	}

	txIndexWg.Wait()

	return l2BlockChain, nil
}

func checkAndDownloadDB(ctx context.Context, stack *node.Node, config *config.NodeConfig) error {
	err := checkDBDir(stack, config)
	if err != nil {
		return err
	}

	return downloadDB(ctx, stack, config)
}

func checkDBDir(stack *node.Node, config *config.NodeConfig) error {
	if err := checkEmptyDatabaseDir(stack.InstanceDir(), config.Init.Force); err != nil {
		return err
	}

	const errorFmt = "database was not found in %s, but it was found in %s (have you placed the database in the wrong directory?)"
	parentDir := filepath.Dir(stack.InstanceDir())
	if dirExists(path.Join(parentDir, "l2chaindata")) {
		return fmt.Errorf(errorFmt, stack.InstanceDir(), parentDir)
	}
	grandParentDir := filepath.Dir(parentDir)
	if dirExists(path.Join(grandParentDir, "l2chaindata")) {
		return fmt.Errorf(errorFmt, stack.InstanceDir(), grandParentDir)
	}

	return nil
}

func downloadDB(ctx context.Context, stack *node.Node, config *config.NodeConfig) error {
	if err := setLatestSnapshotUrl(ctx, &config.Init, config.Chain.Name); err != nil {
		return err
	}

	initFile, cleanUpTmp, err := initializeAndDownloadInit(ctx, &config.Init, stack)
	defer cleanUpTmp()
	if err != nil {
		return err
	}

	if initFile != "" {
		if err := extractSnapshot(initFile, stack.InstanceDir(), config.Init.ImportWasm); err != nil {
			return err
		}
	}

	return nil
}

func openDownloadedExecutionDB(stack *node.Node, config *config.NodeConfig, cacheConfig *core.BlockChainConfig, persistentConfig *conf.PersistentConfig) (ethdb.Database, ethdb.Database, error) {
	executionDB, wasmDB, err := openExecutionDB(stack, config, cacheConfig, persistentConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open executionDB: %w", err)
	}

	// Rebuilding wasm store is not required when just starting out
	err = gethexec.WriteToKeyValueStore(wasmDB, gethexec.RebuildingPositionKey, gethexec.RebuildingDone)
	log.Info("Setting codehash position in rebuilding of wasm store to done")
	if err != nil {
		return nil, nil, fmt.Errorf("unable to set codehash position in rebuilding of wasm store to done: %w", err)
	}

	return executionDB, wasmDB, nil
}

func OpenConsensusDB(stack *node.Node, config *config.NodeConfig) (ethdb.Database, error) {
	consensusDB, err := stack.OpenDatabaseWithOptions("arbitrumdata", node.DatabaseOptions{MetricsNamespace: "arbitrumdata/", PebbleExtraOptions: config.Persistent.Pebble.ExtraOptions("arbitrumdata"), NoFreezer: true})
	if err != nil {
		log.Error("failed to open database", "err", err)
		log.Error("database is corrupt; delete it and try again", "database-directory", stack.InstanceDir())
		return nil, err
	}
	if err := dbutil.UnfinishedConversionCheck(consensusDB); err != nil {
		log.Error("arbitrumdata unfinished conversion check error", "err", err)
		return nil, err
	}

	return consensusDB, nil
}

func openExecutionDB(stack *node.Node, config *config.NodeConfig, cacheConfig *core.BlockChainConfig, persistentConfig *conf.PersistentConfig) (ethdb.Database, ethdb.Database, error) {
	chainData, err := stack.OpenDatabaseWithOptions("l2chaindata", node.DatabaseOptions{AncientsDirectory: config.Persistent.Ancient, MetricsNamespace: "l2chaindata/", Cache: config.Execution.Caching.DatabaseCache, Handles: config.Persistent.Handles, PebbleExtraOptions: persistentConfig.Pebble.ExtraOptions("l2chaindata")})
	if err != nil {
		return nil, nil, err
	}
	if err := dbutil.UnfinishedConversionCheck(chainData); err != nil {
		return nil, nil, fmt.Errorf("l2chaindata unfinished database conversion check error: %w", err)
	}

	wasmDB, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{Cache: config.Execution.Caching.DatabaseCache, Handles: config.Persistent.Handles, MetricsNamespace: "wasm/", PebbleExtraOptions: persistentConfig.Pebble.ExtraOptions("wasm"), NoFreezer: true})
	if err != nil {
		return nil, nil, err
	}
	if err := validateOrUpgradeWasmStoreSchemaVersion(wasmDB); err != nil {
		return nil, nil, err
	}
	if err := validateOrUpgradeWasmerSerializeVersion(wasmDB); err != nil {
		return nil, nil, err
	}
	if err := dbutil.UnfinishedConversionCheck(wasmDB); err != nil {
		return nil, nil, fmt.Errorf("wasm unfinished database conversion check error: %w", err)
	}
	executionDB := rawdb.WrapDatabaseWithWasm(chainData, wasmDB)
	_, err = rawdb.ParseStateScheme(cacheConfig.StateScheme, executionDB)
	if err != nil {
		return nil, nil, err
	}

	return executionDB, wasmDB, nil
}

func OpenExistingExecutionDB(stack *node.Node, config *config.NodeConfig, chainId *big.Int, cacheConfig *core.BlockChainConfig, tracer *tracing.Hooks, persistentConfig *conf.PersistentConfig) (ethdb.Database, ethdb.Database, *core.BlockChain, *params.ChainConfig, error) {
	if !config.Init.Force {
		if readOnlyDb, err := stack.OpenDatabaseWithOptions("l2chaindata", node.DatabaseOptions{AncientsDirectory: config.Persistent.Ancient, MetricsNamespace: "l2chaindata/", ReadOnly: true, PebbleExtraOptions: persistentConfig.Pebble.ExtraOptions("l2chaindata")}); err == nil {
			if chainConfig := gethexec.TryReadStoredChainConfig(readOnlyDb); chainConfig != nil {
				readOnlyDb.Close()
				if !arbmath.BigEquals(chainConfig.ChainID, chainId) {
					return nil, nil, nil, chainConfig, fmt.Errorf("database has chain ID %v but config has chain ID %v (are you sure this database is for the right chain?)", chainConfig.ChainID, chainId)
				}

				executionDB, wasmDB, err := openExecutionDB(stack, config, cacheConfig, persistentConfig)
				if err != nil {
					return nil, nil, nil, chainConfig, err
				}

				l2BlockChain, err := gethexec.GetBlockChain(executionDB, cacheConfig, chainConfig, tracer, &config.Execution.TxIndexer)
				if err != nil {
					return nil, nil, nil, chainConfig, err
				}

				return executionDB, wasmDB, l2BlockChain, chainConfig, nil
			}
			readOnlyDb.Close()
		} else if !dbutil.IsNotExistError(err) {
			// we only want to continue if the database does not exist
			return nil, nil, nil, nil, fmt.Errorf("failed to open database: %w", err)
		} else {
			log.Debug("failed to open l2chaindata", "err", err)
		}
	}

	return nil, nil, nil, nil, nil
}

func GetConsensusParsedInitMsg(ctx context.Context, parentChainReaderEnabled bool, chainId *big.Int, l1Client *ethclient.Client, rollupAddrs *chaininfo.RollupAddresses, chainConfig *params.ChainConfig) (*arbostypes.ParsedInitMessage, error) {
	var parsedInitMessage *arbostypes.ParsedInitMessage
	if parentChainReaderEnabled {
		delayedBridge, err := arbnode.NewDelayedBridge(l1Client, rollupAddrs.Bridge, rollupAddrs.DeployedAt)
		if err != nil {
			return nil, fmt.Errorf("failed creating delayed bridge while attempting to get serialized chain config from init message: %w", err)
		}
		deployedAt := new(big.Int).SetUint64(rollupAddrs.DeployedAt)
		delayedMessages, err := delayedBridge.LookupMessagesInRange(ctx, deployedAt, deployedAt, nil)
		if err != nil {
			return nil, fmt.Errorf("failed getting delayed messages while attempting to get serialized chain config from init message: %w", err)
		}
		var initMessage *arbostypes.L1IncomingMessage
		for _, msg := range delayedMessages {
			if msg.Message.Header.Kind == arbostypes.L1MessageType_Initialize {
				initMessage = msg.Message
				break
			}
		}
		if initMessage == nil {
			return nil, fmt.Errorf("failed to get init message while attempting to get serialized chain config")
		}
		parsedInitMessage, err = initMessage.ParseInitMessage()
		if err != nil {
			return nil, err
		}
		if parsedInitMessage.ChainId.Cmp(chainId) != 0 {
			return nil, fmt.Errorf("expected L2 chain ID %v but read L2 chain ID %v from init message in L1 inbox", chainId, parsedInitMessage.ChainId)
		}
		if parsedInitMessage.ChainConfig != nil {
			if err := parsedInitMessage.ChainConfig.CheckCompatible(chainConfig, chainConfig.ArbitrumChainParams.GenesisBlockNum, 0); err != nil {
				return nil, fmt.Errorf("incompatible chain config read from init message in L1 inbox: %w", err)
			}
		}
		log.Info("Read serialized chain config from init message", "json", string(parsedInitMessage.SerializedChainConfig))
	} else {
		serializedChainConfig, err := json.Marshal(chainConfig)
		if err != nil {
			return nil, err
		}
		parsedInitMessage = &arbostypes.ParsedInitMessage{
			ChainId:               chainConfig.ChainID,
			InitialL1BaseFee:      arbostypes.DefaultInitialL1BaseFee,
			ChainConfig:           chainConfig,
			SerializedChainConfig: serializedChainConfig,
		}
		log.Warn("Created fake init message as L1Reader is disabled and serialized chain config from init message is not available", "json", string(serializedChainConfig))
	}

	return parsedInitMessage, nil
}

func getGenesisAssertionCreationInfo(ctx context.Context, rollupAddress common.Address, l1Client *ethclient.Client, genesisHash common.Hash, sendRoot common.Hash) (*protocol.AssertionCreatedInfo, [32]byte, bool, error) {
	var assertionHash [32]byte

	if l1Client == nil {
		return nil, assertionHash, false, fmt.Errorf("no l1 client")
	}

	userLogic, err := rollupgen.NewRollupUserLogic(rollupAddress, l1Client)
	if err != nil {
		return nil, assertionHash, false, err
	}
	_, err = userLogic.ChallengeGracePeriodBlocks(&bind.CallOpts{Context: ctx})
	if err != nil {
		if !headerreader.IsExecutionReverted(err) {
			return nil, assertionHash, false, err
		}
		log.Warn("Genesis Assertion is not tested") // not a bold chain
		return nil, assertionHash, false, nil
	}

	assertionHash, err = userLogic.GenesisAssertionHash(&bind.CallOpts{Context: context.Background()})
	if err != nil {
		return nil, assertionHash, false, err
	}

	genesisAssertionCreationInfo, err := bold.ReadBoldAssertionCreationInfo(ctx, userLogic, l1Client, rollupAddress, assertionHash)

	if err != nil {
		// If we can't find the empty genesis assertion, try to compute the assertion for non-empty genesis
		genesisGlobalState := protocol.GoGlobalState{
			BlockHash:  genesisHash,
			SendRoot:   sendRoot,
			Batch:      1,
			PosInBatch: 0,
		}
		genesisAssertionState := rollupgen.AssertionState{
			GlobalState: rollupgen.GlobalState{
				Bytes32Vals: genesisGlobalState.AsSolidityStruct().Bytes32Vals,
				U64Vals:     genesisGlobalState.AsSolidityStruct().U64Vals,
			},
			MachineStatus:  1,
			EndHistoryRoot: [32]byte{},
		}

		assertionHash, err = userLogic.ComputeAssertionHash(&bind.CallOpts{Context: ctx}, common.Hash{}, genesisAssertionState, common.Hash{})
		if err != nil {
			return nil, assertionHash, false, err
		}

		genesisAssertionCreationInfo, err = bold.ReadBoldAssertionCreationInfo(ctx, userLogic, l1Client, rollupAddress, assertionHash)
	}

	return genesisAssertionCreationInfo, assertionHash, true, err
}

func GetAndValidateGenesisAssertion(ctx context.Context, l2BlockChain *core.BlockChain, initDataReader statetransfer.InitDataReader, rollupAddrs *chaininfo.RollupAddresses, l1Client *ethclient.Client) error {
	genesisBlock := l2BlockChain.Genesis()
	sendRoot := types.DeserializeHeaderExtraInformation(genesisBlock.Header()).SendRoot
	genesisAssertionCreationInfo, genesisAssertionHash, isBoldChain, err := getGenesisAssertionCreationInfo(ctx, rollupAddrs.Rollup, l1Client, genesisBlock.Hash(), sendRoot)
	if err != nil {
		return err
	}

	if isBoldChain {
		accountsReader, err := initDataReader.GetAccountDataReader()
		if err != nil {
			return err
		}

		return validateGenesisAssertion(genesisAssertionCreationInfo, genesisAssertionHash, genesisBlock.Hash(), sendRoot, accountsReader.More())
	}

	return nil
}

func validateGenesisAssertion(genesisAssertionCreationInfo *protocol.AssertionCreatedInfo, genesisAssertionHash [32]byte, genesisHash common.Hash, sendRoot common.Hash, initDataReaderHasAccounts bool) error {
	beforeGlobalState := protocol.GoGlobalStateFromSolidity(genesisAssertionCreationInfo.BeforeState.GlobalState)
	afterGlobalState := protocol.GoGlobalStateFromSolidity(genesisAssertionCreationInfo.AfterState.GlobalState)
	isNullAssertion := beforeGlobalState.Batch == afterGlobalState.Batch && beforeGlobalState.PosInBatch == afterGlobalState.PosInBatch
	if isNullAssertion && initDataReaderHasAccounts {
		return errors.New("genesis assertion is null but there are accounts in the init data")
	}
	if !isNullAssertion && afterGlobalState.BlockHash != genesisHash {
		return errors.New("genesis assertion is non null and its afterGlobalState.BlockHash doesn't match the genesis blockHash")
	}
	log.Info("Genesis assertion validated", "genesisAssertionHash", genesisAssertionHash, "genesisBlockHash", genesisHash, "genesisSendRoot", sendRoot)
	return nil
}

func testTxIndexUpdated(executionDB ethdb.Database, lastBlock uint64) bool {
	var transactions types.Transactions
	blockHash := rawdb.ReadCanonicalHash(executionDB, lastBlock)
	_, reReadNumberFound := rawdb.ReadHeaderNumber(executionDB, blockHash)
	if !reReadNumberFound {
		return false
	}
	for ; ; lastBlock-- {
		blockHash := rawdb.ReadCanonicalHash(executionDB, lastBlock)
		block := rawdb.ReadBlock(executionDB, blockHash, lastBlock)
		transactions = block.Transactions()
		if len(transactions) == 0 {
			if lastBlock == 0 {
				return true
			}
			continue
		}
		entry := rawdb.ReadTxLookupEntry(executionDB, transactions[len(transactions)-1].Hash())
		return entry != nil
	}
}

func testUpdateTxIndex(executionDB ethdb.Database, chainConfig *params.ChainConfig, globalWg *sync.WaitGroup) {
	lastBlock := chainConfig.ArbitrumChainParams.GenesisBlockNum
	if lastBlock == 0 {
		// no Tx, no need to update index
		return
	}

	lastBlock -= 1
	if testTxIndexUpdated(executionDB, lastBlock) {
		return
	}

	var localWg sync.WaitGroup
	threads := util.GoMaxProcs()
	var failedTxIndiciesMutex sync.Mutex
	failedTxIndicies := make(map[common.Hash]uint64)
	for thread := 0; thread < threads; thread++ {
		thread := thread
		localWg.Add(1)
		go func() {
			batch := executionDB.NewBatch()
			// #nosec G115
			for blockNum := uint64(thread); blockNum <= lastBlock; blockNum += uint64(threads) {
				blockHash := rawdb.ReadCanonicalHash(executionDB, blockNum)
				block := rawdb.ReadBlock(executionDB, blockHash, blockNum)
				receipts := rawdb.ReadRawReceipts(executionDB, blockHash, blockNum)
				for i, receipt := range receipts {
					// receipt.TxHash isn't populated as we used ReadRawReceipts
					txHash := block.Transactions()[i].Hash()
					if receipt.Status != 0 || receipt.GasUsed != 0 {
						rawdb.WriteTxLookupEntries(batch, blockNum, []common.Hash{txHash})
					} else {
						failedTxIndiciesMutex.Lock()
						prev, exists := failedTxIndicies[txHash]
						if !exists || prev < blockNum {
							failedTxIndicies[txHash] = blockNum
						}
						failedTxIndiciesMutex.Unlock()
					}
				}
				rawdb.WriteHeaderNumber(batch, block.Header().Hash(), blockNum)
				if blockNum%1_000_000 == 0 {
					log.Info("writing tx lookup entries", "block", blockNum)
				}
				if batch.ValueSize() >= ethdb.IdealBatchSize {
					err := batch.Write()
					if err != nil {
						panic(err)
					}
					batch.Reset()
				}
			}
			err := batch.Write()
			if err != nil {
				panic(err)
			}
			localWg.Done()
		}()
	}

	globalWg.Add(1)
	go func() {
		localWg.Wait()
		batch := executionDB.NewBatch()
		for txHash, blockNum := range failedTxIndicies {
			if rawdb.ReadTxLookupEntry(executionDB, txHash) == nil {
				rawdb.WriteTxLookupEntries(batch, blockNum, []common.Hash{txHash})
			}
			if batch.ValueSize() >= ethdb.IdealBatchSize {
				err := batch.Write()
				if err != nil {
					panic(err)
				}
				batch.Reset()
			}
		}
		err := batch.Write()
		if err != nil {
			panic(err)
		}
		log.Info("Tx lookup entries written")
		globalWg.Done()
	}()
}

func InitReorg(initConfig conf.InitConfig, chainConfig *params.ChainConfig, inboxTracker *arbnode.InboxTracker) error {
	var batchCount uint64
	if initConfig.ReorgToBatch >= 0 {
		// #nosec G115
		batchCount = uint64(initConfig.ReorgToBatch) + 1
	} else {
		var messageIndex arbutil.MessageIndex
		if initConfig.ReorgToMessageBatch >= 0 {
			// #nosec G115
			messageIndex = arbutil.MessageIndex(initConfig.ReorgToMessageBatch)
		} else if initConfig.ReorgToBlockBatch > 0 {
			genesis := chainConfig.ArbitrumChainParams.GenesisBlockNum
			// #nosec G115
			blockNum := uint64(initConfig.ReorgToBlockBatch)
			if blockNum < genesis {
				return fmt.Errorf("ReorgToBlockBatch %d before genesis %d", blockNum, genesis)
			}
			messageIndex = arbutil.MessageIndex(blockNum - genesis)
		} else {
			log.Warn("Tried to do init reorg, but no init reorg options specified")
			return nil
		}
		// Reorg out the batch containing the next message
		var found bool
		var err error
		batchCount, found, err = inboxTracker.FindInboxBatchContainingMessage(messageIndex + 1)
		if err != nil {
			return err
		}
		if !found {
			log.Warn("init-reorg: no need to reorg, because message ahead of chain", "messageIndex", messageIndex)
			return nil
		}
	}
	return inboxTracker.ReorgBatchesTo(batchCount)
}
