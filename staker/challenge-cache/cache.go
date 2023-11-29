// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
/*
* Package challengecache stores validator state roots for L2 states within
challenges in text files using a directory hierarchy structure for efficient lookup. Each file
contains a list of state roots (32 byte hashes), concatenated together as bytes.
Using this structure, we can namespace state roots by message number and big step challenge.

Once a validator computes the set of machine state roots for a given challenge move the first time,
it will write the roots to this filesystem hierarchy for fast access next time these roots are needed.

Use cases:
- State roots for a big step challenge from message N to N+1
- State roots 0 to M for a big step challenge from message N to N+1
- State roots for a small step challenge from message N to N+1, and big step M to M+1
- State roots 0 to P for a small step challenge from message N to N+1, and big step M to M+1

	  wavm-module-root-0xab/
		message-num-70/
			roots.txt
			subchallenge-level-0-big-step-100/
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
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/mmap"

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
	Get(lookup *Key, numToRead uint64) (mmap.Mmap, error)
	Put(lookup *Key, stateRoots mmap.Mmap) error
}

// Cache for history commitments on disk.
type Cache struct {
	baseDir string
}

func isOlderThanFourteenDays(t time.Time) bool {
	return time.Since(t) > 14*24*time.Hour
}

func deleteFilesOlderThanFourteenDays(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	for _, file := range files {
		fileInfo, err := file.Info()
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			if err := deleteFilesOlderThanFourteenDays(filepath.Join(dir, fileInfo.Name())); err != nil {
				return err
			}
		} else {
			if isOlderThanFourteenDays(fileInfo.ModTime()) {
				if err := os.Remove(filepath.Join(dir, fileInfo.Name())); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// New cache from a base directory path.
func New(baseDir string) (*Cache, error) {
	err := deleteFilesOlderThanFourteenDays(baseDir)
	if err != nil {
		return nil, err
	}
	return &Cache{
		baseDir: baseDir,
	}, nil
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
) (mmap.Mmap, error) {
	fName, err := determineFilePath(c.baseDir, lookup)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(fName); err != nil {
		srvlog.Warn("Cache miss", log.Ctx{"fileName": fName})
		return nil, ErrNotFoundInCache
	}
	srvlog.Debug("Cache hit", log.Ctx{"fileName": fName})
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
func (c *Cache) Put(lookup *Key, stateRoots mmap.Mmap) error {
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
	return os.Rename(tmpFName /* old */, fName /* new */)
}

// Reads 32 bytes at a time from a reader up to a specified height. If none, then read all.
func readStateRoots(r io.Reader, numToRead uint64) (mmap.Mmap, error) {
	br := bufio.NewReader(r)
	stateRootsMmap, err := mmap.NewMmap(int(numToRead))
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 0, 32)
	var totalRead uint64
	for totalRead = uint64(0); totalRead < numToRead; totalRead++ {
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
		stateRootsMmap.Set(int(totalRead), common.BytesToHash(buf))
	}
	if protocol.Height(numToRead) > protocol.Height(totalRead) {
		return nil, fmt.Errorf(
			"wanted to read %d roots, but only read %d state roots",
			numToRead,
			totalRead,
		)
	}
	return stateRootsMmap, nil
}

func writeStateRoots(w io.Writer, stateRoots mmap.Mmap) error {
	for i := 0; i < stateRoots.Length(); i++ {
		n, err := w.Write(stateRoots.Get(i).Bytes())
		if err != nil {
			return err
		}
		if n != len(stateRoots.Get(i)) {
			return fmt.Errorf(
				"for state root %d, wrote %d bytes, expected to write %d bytes",
				i,
				n,
				len(stateRoots.Get(i)),
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
			subchallenge-level-0-big-step-100/
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
