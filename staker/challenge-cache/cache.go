/*
* Package challengecache stores validator state roots for L2 states within
challenges in text files using a directory hierarchy structure for efficient lookup. Each file
contains a list of state roots (32 byte hashes), concatenated together as bytes.
Using this structure, we can namespace state roots by message number and big step challenge.

Once a validator computes the set of machine state roots for a given challenge move the first time,
it will write the roots to this filesystem hierarchy for fast access next time these roots are needed.
Each file can be written to ONCE, and then accessed at unpredictable times as a challenge is ongoing.

Use cases:
- State roots for a big step challenge from message N to N+1
- State roots 0 to M for a big step challenge from message N to N+1
- State roots for a small step challenge from message N to N+1, and big step M to M+1
- State roots 0 to P for a small step challenge from message N to N+1, and big step M to M+1

	  wavm-module-root-0xab/
		message-num-70-71/
			roots.txt
			big-step-100-101/
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
func New(baseDir string) *Cache {
	return &Cache{
		baseDir: baseDir,
	}
}

// Key for cache lookups includes the wavm module root and assertion of a challenge, as well
// as the height ranges for messages, big steps, and small steps as needed.
type Key struct {
	WavmModuleRoot common.Hash
	MessageRange   HeightRange
	BigStepRange   option.Option[HeightRange]
}

// HeightRange within a challenge.
type HeightRange struct {
	From protocol.Height
	To   protocol.Height
}

// Get a list of state roots from the cache up to a certain index if specified. If none, then all
// state roots for the lookup key will be retrieved. State roots are saved as files in the directory
// hierarchy for the cache, and can only be written to once. If a file is not present, ErrNotFoundInCache
// is returned.
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
	defer func() {
		if err := f.Close(); err != nil {
			log.Error("Could not close file after reading", "err", err, "file", fName)
		}
	}()
	return readStateRoots(f, readUpTo)
}

// Put a list of state roots into the cache. If the file already exists, ErrFileAlreadyExists will be returned.
// State roots are saved as files in a directory hierarchy for the cache, and can only be written to once.
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
	if _, err := os.Stat(fName); err == nil {
		return ErrFileAlreadyExists
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
func readStateRoots(r io.Reader, readUpTo option.Option[protocol.Height]) ([]common.Hash, error) {
	br := bufio.NewReader(r)
	stateRoots := make([]common.Hash, 0)
	buf := make([]byte, 0, 32)
	totalRead := uint64(0)
	for {
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
		if !readUpTo.IsNone() {
			if totalRead >= uint64(readUpTo.Unwrap()) {
				return stateRoots, nil
			}
		}
		totalRead++
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
		message-num-70-71/
			roots.txt
			big-step-100-101/
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
	if err := lookup.MessageRange.ValidateOneStepFork(); err != nil {
		return "", fmt.Errorf("message number range invalid")
	}
	key = append(key, fmt.Sprintf("%s-%d-%d", messageNumberPrefix, lookup.MessageRange.From, lookup.MessageRange.To))
	if !lookup.BigStepRange.IsNone() {
		bigStepRange := lookup.BigStepRange.Unwrap()
		if err := bigStepRange.ValidateOneStepFork(); err != nil {
			return "", fmt.Errorf("big step range invalid")
		}
		key = append(key, fmt.Sprintf("%s-%d-%d", bigStepPrefix, bigStepRange.From, bigStepRange.To))
	}
	key = append(key, stateRootsFileName)
	return filepath.Join(baseDir, filepath.Join(key...)), nil
}

// ValidateOneStepFork checks if a height range has a difference of 1.
func (h HeightRange) ValidateOneStepFork() error {
	if h.To != h.From+1 {
		return fmt.Errorf(
			"expected range difference of 1, got range from %d to %d",
			h.From,
			h.To,
		)
	}
	return nil
}
