/*
* Package challengecache stores validator state roots for L2 states within
challenges in text files using a directory hierarchy structure for efficient lookup. Each file
contains a list of state roots (32 byte hashes), concatenated together as bytes.
Using this structure, we can namespace state roots by assertion hash,
message number, big step challenge, and small step challenge ranges.

Once a validator computes the set of machine state roots for a given challenge move the first time,
it will write the roots to this filesystem hierarchy for fast access next time these roots are needed.
Each file can be written to ONCE, and then accessed at unpredictable times as a challenge is ongoing.

Use cases:
- State roots for a block challenge from message 0 to N
- State roots for a big step challenge from message N to N+1
- State roots 0 to M for a big step challenge from message N to N+1
- State roots for a small step challenge from message N to N+1, and big step M to M+1
- State roots 0 to P for a small step challenge from message N to N+1, and big step M to M+1

	  wavm-module-root-0xab/
		assertion-0x123/
			message-num-0-100/
				roots.txt
			message-num-70-71/
				big-step-0-2048/
					roots.txt
				big-step-100-101/
					small-step-0-256/
						roots.txt

We namespace top-level block challenges by wavm module root and assertion hash. Then, we can retrieve
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

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrNotFoundInCache   = errors.New("no found in challenge cache")
	ErrFileAlreadyExists = errors.New("file already exists")
	stateRootsFileName   = "roots.txt"
	wavmModuleRootPrefix = "wavm-module-root"
	assertionPrefix      = "assertion"
	messageNumberPrefix  = "message-num"
	bigStepPrefix        = "big-step"
	smallStepPrefix      = "small-step"
)

// HistoryCommitmentCacher can retrieve history commitment state roots given lookup keys.
type HistoryCommitmentCacher interface {
	Get(lookup *Key, readUpTo option.Option[protocol.Height]) ([]common.Hash, error)
	Put(lookup *Key, stateRoots []common.Hash) error
}

// Cache for history commitments on disk.
type Cache struct {
	baseDir string
}

// New cache from a base directory path.
func New(baseDir string) (*Cache, error) {
	return &Cache{
		baseDir: baseDir,
	}, nil
}

// Key for cache lookups includes the wavm module root and assertion of a challenge, as well
// as the height ranges for messages, big steps, and small steps as needed.
type Key struct {
	WavmModuleRoot common.Hash
	AssertionHash  common.Hash
	MessageRange   HeightRange
	BigStepRange   option.Option[HeightRange]
	ToSmallStep    option.Option[protocol.Height]
}

// HeightRange within a challenge.
type HeightRange struct {
	from protocol.Height
	to   protocol.Height
}

// Get a list of state roots from the cache up to a certain index if specified. If none, then all
// state roots for the lookup key will be retrieved.
func (c *Cache) Get(
	lookup *Key,
	readUpTo option.Option[protocol.Height],
) ([]common.Hash, error) {
	fName, err := determineFilePath(c.baseDir, lookup)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(fName); err != nil {
		return nil, ErrNotFoundInCache
	}
	f, err := os.Open(fName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readStateRoots(f, readUpTo)
}

// Put a list of state roots into the cache. If the file already exists, ErrFileAlreadyExists will be returned.
func (c *Cache) Put(lookup *Key, stateRoots []common.Hash) error {
	fName, err := determineFilePath(c.baseDir, lookup)
	if err != nil {
		return err
	}
	if _, err := os.Stat(fName); err == nil {
		return ErrFileAlreadyExists
	}
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
	defer f.Close()
	if err := writeStateRoots(f, stateRoots); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fName), os.ModePerm); err != nil {
		return fmt.Errorf("could not make file directory %s: %w", fName, err)
	}
	// if the file writing was successful, we rename the file from the tmp directory
	// into our cache directory.
	return os.Rename(tmpFName /* old */, fName /* new */)
}

// Reads 32 bytes at a time from a reader up to a specified height. If none, then read all.
func readStateRoots(r io.Reader, readUpTo option.Option[protocol.Height]) ([]common.Hash, error) {
	br := bufio.NewReader(r)
	stateRoots := make([]common.Hash, 0)
	buf := make([]byte, 0, 32)
	idx := uint64(0)
	for {
		n, err := br.Read(buf[:cap(buf)])
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			return nil, err
		}
		stateRoots = append(stateRoots, common.BytesToHash(buf))
		if !readUpTo.IsNone() {
			if idx >= uint64(readUpTo.Unwrap()) {
				return stateRoots, nil
			}
		}
		if err != nil && err != io.EOF {
			return nil, err
		}
		idx++
	}
	if !readUpTo.IsNone() {
		if readUpTo.Unwrap() > protocol.Height(len(stateRoots)) {
			return nil, fmt.Errorf(
				"wanted to read up to %d, but only read %d state roots",
				readUpTo.Unwrap(),
				len(stateRoots),
			)
		}
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
		assertion-0x123/
			message-num-0-100/
				roots.txt
			message-num-70-71/
				big-step-0-2048/
					roots.txt
				big-step-100-101/
					small-step-0-256/
						roots.txt

Invariants:
- Message number height from < to
- If big step range exists, message number height to == from + 1
- If small step exists, big step height to == from + 1
- Small step roots are always from 0 to N
*/
func determineFilePath(baseDir string, lookup *Key) (string, error) {
	key := make([]string, 0)
	key = append(key, fmt.Sprintf("%s-%s", wavmModuleRootPrefix, lookup.WavmModuleRoot.Hex()))
	key = append(key, fmt.Sprintf("%s-%s", assertionPrefix, lookup.AssertionHash.Hex()))
	if err := lookup.MessageRange.ValidateIncreasing(); err != nil {
		return "", fmt.Errorf("message number range invalid")
	}
	key = append(key, fmt.Sprintf("%s-%d-%d", messageNumberPrefix, lookup.MessageRange.from, lookup.MessageRange.to))
	if !lookup.BigStepRange.IsNone() {
		if err := lookup.MessageRange.ValidateOneStepFork(); err != nil {
			return "", fmt.Errorf("message number range invalid")
		}
		bigRange := lookup.BigStepRange.Unwrap()
		if err := bigRange.ValidateIncreasing(); err != nil {
			return "", fmt.Errorf("big step range invalid")
		}
		key = append(key, fmt.Sprintf("%s-%d-%d", bigStepPrefix, bigRange.from, bigRange.to))
		if !lookup.ToSmallStep.IsNone() {
			if err := bigRange.ValidateOneStepFork(); err != nil {
				return "", fmt.Errorf("big step range invalid")
			}
			key = append(key, fmt.Sprintf("%s-0-%d", smallStepPrefix, lookup.ToSmallStep.Unwrap()))
		}
	}
	key = append(key, stateRootsFileName)
	return filepath.Join(baseDir, filepath.Join(key...)), nil
}

// ValidateIncreasing checks if a height range has from < to.
func (h HeightRange) ValidateIncreasing() error {
	if h.from >= h.to {
		return fmt.Errorf("from %d was >= to %d", h.from, h.to)
	}
	return nil
}

// ValidateOneStepFork checks if a height range has a difference of 1.
func (h HeightRange) ValidateOneStepFork() error {
	if h.to != h.from+1 {
		return fmt.Errorf(
			"expected range difference of 1, got range from %d to %d",
			h.from,
			h.to,
		)
	}
	return nil
}
