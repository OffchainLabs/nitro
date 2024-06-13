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

Once a validator receives a full list of computed machine hashes for the first time from a validatio node,
it will write the hashes to this filesystem hierarchy for fast access next time these hashes are needed.

Example uses:
- Obtain all the hashes for the execution of message num 70 to 71 for a given wavm module root.
- Obtain all the hashes from step 100 to 101 at subchallenge level 1 for the execution of message num 70.

	  wavm-module-root-0xab/
		rollup-block-hash-0x12...-message-num-70/
			hashes.bin
			subchallenge-level-1-big-step-100/
				hashes.bin

We namespace top-level block challenges by wavm module root. Then, we can retrieve
the hashes for any data within a challenge or associated subchallenge based on the hierarchy above.
*/

package challengecache

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNotFoundInCache    = errors.New("no found in challenge cache")
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

// Cache for history commitments on disk.
type Cache struct {
	baseDir string
}

// New cache from a base directory path.
func New(baseDir string) *Cache {
	return &Cache{
		baseDir: baseDir,
	}
}

// Key for cache lookups includes the wavm module root of a challenge, as well
// as the heights for messages and big steps as needed.
type Key struct {
	RollupBlockHash common.Hash
	WavmModuleRoot  common.Hash
	MessageHeight   uint64
	StepHeights     []uint64
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
	fName, err := determineFilePath(c.baseDir, lookup)
	if err != nil {
		return err
	}
	// We create a tmp file to write our hashes to first. If writing fails,
	// we don't want to leave a half-written file in our cache directory.
	// Once writing succeeds, we rename in an atomic operation to the correct file name
	// in the cache directory hierarchy.
	tmp := os.TempDir()
	tmpFName := filepath.Join(tmp, fName)
	dir := filepath.Dir(tmpFName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("could not make tmp directory %s: %w", dir, err)
	}
	f, err := os.Create(tmpFName)
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
	// If the file writing was successful, we rename the file from the tmp directory
	// into our cache directory. This is an atomic operation.
	// For more information on this atomic write pattern, see:
	// https://stackoverflow.com/questions/2333872/how-to-make-file-creation-an-atomic-operation
	return Move(tmpFName /* old */, fName /* new */)
}

// Reads 32 bytes at a time from a reader up to a specified height. If none, then read all.
func readHashes(r io.Reader, numToRead uint64) ([]common.Hash, error) {
	br := bufio.NewReader(r)
	hashes := make([]common.Hash, 0)
	buf := make([]byte, 0, 32)
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
		if n != 32 {
			return nil, fmt.Errorf("expected to read 32 bytes, got %d bytes", n)
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
	for i, rt := range hashes {
		n, err := w.Write(rt[:])
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
	return nil
}

/*
*
When provided with a cache lookup struct, this function determines the file path
for the data requested within the cache directory hierarchy. The folder structure
for a given filesystem challenge cache will look as follows:

	  wavm-module-root-0xab/
		rollup-block-hash-0x12...-message-num-70/
			hashes.bin
			subchallenge-level-1-big-step-100/
				hashes.bin
*/
func determineFilePath(baseDir string, lookup *Key) (string, error) {
	key := make([]string, 0)
	key = append(key, fmt.Sprintf("%s-%s", wavmModuleRootPrefix, lookup.WavmModuleRoot.Hex()))
	key = append(key, fmt.Sprintf("%s-%s-%s-%d", rollupBlockHashPrefix, lookup.RollupBlockHash.Hex(), messageNumberPrefix, lookup.MessageHeight))
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

// Move function that is robust against cross-device link errors. Credits to:
// https://gist.github.com/var23rav/23ae5d0d4d830aff886c3c970b8f6c6b
func Move(source, destination string) error {
	err := os.Rename(source, destination)
	if err != nil && strings.Contains(err.Error(), "cross-device link") {
		return moveCrossDevice(source, destination)
	}
	return err
}

func moveCrossDevice(source, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	dst, err := os.Create(destination)
	if err != nil {
		src.Close()
		return err
	}
	_, err = io.Copy(dst, src)
	src.Close()
	dst.Close()
	if err != nil {
		return err
	}
	fi, err := os.Stat(source)
	if err != nil {
		os.Remove(destination)
		return err
	}
	err = os.Chmod(destination, fi.Mode())
	if err != nil {
		os.Remove(destination)
		return err
	}
	os.Remove(source)
	return nil
}
