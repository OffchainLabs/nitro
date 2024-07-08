// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
/*
* Package challengecache stores hashes required for making history commitments in Arbitrum BOLD.
When a challenge begins, validators need to post Merkle commitments to a series of block hashes to
narrow down their disagreement to a single block. Once a disagreement is reached, another BOLD challenge begins
to narrow down within the execution of a block. This requires using the Arbitrator emulator to compute
the intermediate hashes of executing the block as WASM opcodes. These hashes are expensive to compute, so we
store them in a filesystem cache to avoid recomputing them and for hierarchical access.
Each file contains a list of 32 byte hashes, concatenated together as bytes.
Using this structure, we can namespace hashes by message number and by challenge level.

Once a validator receives a full list of computed machine hashes for the first time from a validation node,
it will write the hashes to this filesystem hierarchy for fast access next time these hashes are needed.

Example uses:
- Obtain all the hashes for the execution of message num 70 to 71 for a given wavm module root.
- Obtain all the hashes from step 100 to 101 at subchallenge level 1 for the execution of message num 70.

	  wavm-module-root-0xab/
		message-num-70-rollup-block-hash-0x12.../
			hashes.bin
			subchallenge-level-1-big-step-100/
				hashes.bin

We namespace top-level block challenges by wavm module root. Then, we can retrieve
the hashes for any data within a challenge or associated subchallenge based on the hierarchy above.
*/

package challengecache

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNotFoundInCache    = errors.New("not found in challenge cache")
	ErrFileAlreadyExists  = errors.New("file already exists")
	ErrNoHashes           = errors.New("no hashes being written")
	hashesFileName        = "hashes.bin"
	wavmModuleRootPrefix  = "wavm-module-root"
	rollupBlockHashPrefix = "rollup-block-hash"
	messageNumberPrefix   = "message-num"
	bigStepPrefix         = "big-step"
	challengeLevelPrefix  = "subchallenge-level"
)

// HistoryCommitmentCacher can retrieve history commitment hashes given lookup keys.
type HistoryCommitmentCacher interface {
	Get(lookup *Key, numToRead uint64) ([]common.Hash, error)
	Put(lookup *Key, hashes []common.Hash) error
}

// Key for cache lookups includes the wavm module root of a challenge, as well
// as the heights for messages and big steps as needed.
type Key struct {
	RollupBlockHash common.Hash
	WavmModuleRoot  common.Hash
	MessageHeight   uint64
	StepHeights     []uint64
}

// Cache for history commitments on disk.
type Cache struct {
	baseDir       string
	tempWritesDir string
}

// New cache from a base directory path.
func New(baseDir string) (*Cache, error) {
	return &Cache{
		baseDir:       baseDir,
		tempWritesDir: "",
	}, nil
}

// Init a cache by verifying its base directory exists.
func (c *Cache) Init(_ context.Context) error {
	if _, err := os.Stat(c.baseDir); err != nil {
		if err := os.MkdirAll(c.baseDir, os.ModePerm); err != nil {
			return fmt.Errorf("could not make initialize challenge cache directory %s: %w", c.baseDir, err)
		}
	}
	// We create a temp directory to write our hashes to first when putting to the cache.
	// Once writing succeeds, we rename in an atomic operation to the correct file name
	// in the cache directory hierarchy in the `Put` function. All of these temporary writes
	// will occur in a subdir of the base directory called temp.
	tempWritesDir, err := os.MkdirTemp(c.baseDir, "temp")
	if err != nil {
		return err
	}
	c.tempWritesDir = tempWritesDir
	return nil
}

// Get a list of hashes from the cache from index 0 up to a certain index. Hashes are saved as files in the directory
// hierarchy for the cache. If a file is not present, ErrNotFoundInCache
// is returned.
func (c *Cache) Get(
	lookup *Key,
	numToRead uint64,
) ([]common.Hash, error) {
	fName, err := determineFilePath(c.baseDir, lookup)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(fName); err != nil {
		log.Warn("Cache miss", "fileName", fName)
		return nil, ErrNotFoundInCache
	}
	log.Debug("Cache hit", "fileName", fName)
	f, err := os.Open(fName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error("Could not close file after reading", "err", err, "file", fName)
		}
	}()
	return readHashes(f, numToRead)
}

// Put a list of hashes into the cache.
// Hashes are saved as files in a directory hierarchy for the cache.
// This function first creates a temporary file, writes the hashes to it, and then renames the file
// to the final directory to ensure atomic writes.
func (c *Cache) Put(lookup *Key, hashes []common.Hash) error {
	// We should error if trying to put 0 hashes to disk.
	if len(hashes) == 0 {
		return ErrNoHashes
	}
	if c.tempWritesDir == "" {
		return fmt.Errorf("cache not initialized by calling .Init(ctx)")
	}
	fName, err := determineFilePath(c.baseDir, lookup)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(c.tempWritesDir, fmt.Sprintf("%s-*", hashesFileName))
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error("Could not close file after writing", "err", err, "file", fName)
		}
	}()
	if err := writeHashes(f, hashes); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fName), os.ModePerm); err != nil {
		return fmt.Errorf("could not make file directory %s: %w", fName, err)
	}
	// If the file writing was successful, we rename the file from the temp directory
	// into our cache directory. This is an atomic operation.
	// For more information on this atomic write pattern, see:
	// https://stackoverflow.com/questions/2333872/how-to-make-file-creation-an-atomic-operation
	return os.Rename(f.Name() /*old */, fName /* new */)
}

// Prune all entries in the cache with a message number <= a specified value.
func (c *Cache) Prune(ctx context.Context, messageNumber uint64) error {
	// Define a regex pattern to extract the message number
	numPruned := 0
	messageNumPattern := fmt.Sprintf(`%s-(\d+)-`, messageNumberPrefix)
	pattern := regexp.MustCompile(messageNumPattern)
	pathsToDelete := make([]string, 0)
	if err := filepath.Walk(c.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			matches := pattern.FindStringSubmatch(info.Name())
			if len(matches) > 1 {
				dirNameMessageNum, err := strconv.Atoi(matches[1])
				if err != nil {
					return err
				}
				// Collect the directory path if the message number is <= the specified value.
				if dirNameMessageNum <= int(messageNumber) {
					pathsToDelete = append(pathsToDelete, path)
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	// We delete separately from collecting the paths, as deleting while walking
	// a dir can cause issues with the filepath.Walk function.
	for _, path := range pathsToDelete {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("could not prune directory with path %s: %w", path, err)
		}
		numPruned += 1
	}
	log.Info("Pruned challenge cache", "numDirsPruned", numPruned, "messageNumber", messageNumPattern)
	return nil
}

// Reads 32 bytes at a time from a reader up to a specified height. If none, then read all.
func readHashes(r io.Reader, numToRead uint64) ([]common.Hash, error) {
	br := bufio.NewReader(r)
	hashes := make([]common.Hash, 0)
	buf := make([]byte, 0, common.HashLength)
	for totalRead := uint64(0); totalRead < numToRead; totalRead++ {
		n, err := br.Read(buf[:cap(buf)])
		if err != nil {
			// If we try to read but reach EOF, we break out of the loop.
			if err == io.EOF {
				break
			}
			return nil, err
		}
		buf = buf[:n]
		if n != common.HashLength {
			return nil, fmt.Errorf("expected to read %d bytes, got %d bytes", common.HashLength, n)
		}
		hashes = append(hashes, common.BytesToHash(buf))
	}
	if numToRead > uint64(len(hashes)) {
		return nil, fmt.Errorf(
			"wanted to read %d hashes, but only read %d hashes",
			numToRead,
			len(hashes),
		)
	}
	return hashes, nil
}

func writeHashes(w io.Writer, hashes []common.Hash) error {
	bw := bufio.NewWriter(w)
	for i, rt := range hashes {
		n, err := bw.Write(rt[:])
		if err != nil {
			return err
		}
		if n != len(rt) {
			return fmt.Errorf(
				"for hash %d, wrote %d bytes, expected to write %d bytes",
				i,
				n,
				len(rt),
			)
		}
	}
	return bw.Flush()
}

/*
*
When provided with a cache lookup struct, this function determines the file path
for the data requested within the cache directory hierarchy. The folder structure
for a given filesystem challenge cache will look as follows:

	  wavm-module-root-0xab/
		message-num-70-rollup-block-hash-0x12.../
			hashes.bin
			subchallenge-level-1-big-step-100/
				hashes.bin
*/
func determineFilePath(baseDir string, lookup *Key) (string, error) {
	key := make([]string, 0)
	key = append(key, fmt.Sprintf("%s-%s", wavmModuleRootPrefix, lookup.WavmModuleRoot.Hex()))
	key = append(key, fmt.Sprintf("%s-%d-%s-%s", messageNumberPrefix, lookup.MessageHeight, rollupBlockHashPrefix, lookup.RollupBlockHash.Hex()))
	for challengeLevel, height := range lookup.StepHeights {
		key = append(key, fmt.Sprintf(
			"%s-%d-%s-%d",
			challengeLevelPrefix,
			challengeLevel+1, // subchallenges start at 1, as level 0 is the block challenge level.
			bigStepPrefix,
			height,
		),
		)

	}
	key = append(key, hashesFileName)
	return filepath.Join(baseDir, filepath.Join(key...)), nil
}
