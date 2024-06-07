// Copyright 2023, Offchain Labs, Inc.
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
it will write the roots to this filesystem hierarchy for fast access next time these roots are needed.

Example:
- Compute all the hashes for the execution of message num 70 with the required step size for the big step challenge level.
- Compute all the hashes for the execution of individual steps for a small step challenge level from big step 100 to 101

	  wavm-module-root-0xab/
		message-num-70/
			roots.txt
			subchallenge-level-1-big-step-100/
				roots.txt

We namespace top-level block challenges by wavm module root. Then, we can retrieve
the state roots for any data within a challenge or associated subchallenge based on the hierarchy above.
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

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNotFoundInCache   = errors.New("no found in challenge cache")
	ErrFileAlreadyExists = errors.New("file already exists")
	ErrNoStateRoots      = errors.New("no state roots being written")
	stateRootsFileName   = "state-roots"
	wavmModuleRootPrefix = "wavm-module-root"
	messageNumberPrefix  = "message-num"
	bigStepPrefix        = "big-step"
	challengeLevelPrefix = "subchallenge-level"
	srvlog               = log.New("service", "bold-history-commit-cache")
)

// HistoryCommitmentCacher can retrieve history commitment state roots given lookup keys.
type HistoryCommitmentCacher interface {
	Get(lookup *Key, numToRead uint64) ([]common.Hash, error)
	Put(lookup *Key, stateRoots []common.Hash) error
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
	WavmModuleRoot common.Hash
	MessageHeight  protocol.Height
	StepHeights    []l2stateprovider.Height
}

// Get a list of state roots from the cache up to a certain index. State roots are saved as files in the directory
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
		srvlog.Warn("Cache miss", "fileName", fName)
		return nil, ErrNotFoundInCache
	}
	srvlog.Debug("Cache hit", "fileName", fName)
	f, err := os.Open(fName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error("Could not close file after reading", "err", err, "file", fName)
		}
	}()
	return readStateRoots(f, numToRead)
}

// Put a list of state roots into the cache.
// State roots are saved as files in a directory hierarchy for the cache.
// This function first creates a temporary file, writes the state roots to it, and then renames the file
// to the final directory to ensure atomic writes.
func (c *Cache) Put(lookup *Key, stateRoots []common.Hash) error {
	// We should error if trying to put 0 state roots to disk.
	if len(stateRoots) == 0 {
		return ErrNoStateRoots
	}
	fName, err := determineFilePath(c.baseDir, lookup)
	if err != nil {
		return err
	}
	// We create a tmp file to write our state roots to first. If writing fails,
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
	if err := writeStateRoots(f, stateRoots); err != nil {
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
func readStateRoots(r io.Reader, numToRead uint64) ([]common.Hash, error) {
	br := bufio.NewReader(r)
	stateRoots := make([]common.Hash, 0)
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
		stateRoots = append(stateRoots, common.BytesToHash(buf))
	}
	if protocol.Height(numToRead) > protocol.Height(len(stateRoots)) {
		return nil, fmt.Errorf(
			"wanted to read %d roots, but only read %d state roots",
			numToRead,
			len(stateRoots),
		)
	}
	return stateRoots, nil
}

func writeStateRoots(w io.Writer, stateRoots []common.Hash) error {
	for i, rt := range stateRoots {
		n, err := w.Write(rt[:])
		if err != nil {
			return err
		}
		if n != len(rt) {
			return fmt.Errorf(
				"for state root %d, wrote %d bytes, expected to write %d bytes",
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
		message-num-70/
			roots.txt
			subchallenge-level-1-big-step-100/
				roots.txt
*/
func determineFilePath(baseDir string, lookup *Key) (string, error) {
	key := make([]string, 0)
	key = append(key, fmt.Sprintf("%s-%s", wavmModuleRootPrefix, lookup.WavmModuleRoot.Hex()))
	key = append(key, fmt.Sprintf("%s-%d", messageNumberPrefix, lookup.MessageHeight))
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
	key = append(key, stateRootsFileName)
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
