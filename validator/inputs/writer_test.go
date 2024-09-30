package inputs

import (
	"os"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/validator/server_api"
)

func TestDefaultBaseDir(t *testing.T) {
	// Simply testing that the default baseDir is set relative to the user's home directory.
	// This way, the other tests can all override the baseDir to a temporary directory.
	w, err := NewWriter()
	if err != nil {
		t.Fatal(err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if w.baseDir != homeDir+"/.arbitrum/validation-inputs" {
		t.Errorf("unexpected baseDir: %v", w.baseDir)
	}
}

type fakeClock struct {
	now time.Time
}

func (c fakeClock) Now() time.Time {
	return c.now
}

func TestWriting(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(
		withTestClock(fakeClock{now: time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC)}),
		WithBaseDir(dir),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Write(&server_api.InputJSON{Id: 24601})
	if err != nil {
		t.Fatal(err)
	}
	// The file should exist.
	if _, err := os.Stat(dir + "/20210102_030405/block_inputs_24601.json"); err != nil {
		t.Error(err)
	}
}

func TestWritingWithSlug(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(
		withTestClock(fakeClock{now: time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC)}),
		WithBaseDir(dir),
		WithSlug("foo"),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Write(&server_api.InputJSON{Id: 24601})
	if err != nil {
		t.Fatal(err)
	}
	// The file should exist.
	if _, err := os.Stat(dir + "/foo/20210102_030405/block_inputs_24601.json"); err != nil {
		t.Error(err)
	}
}

func TestWritingWithoutTimestampDir(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(
		withTestClock(fakeClock{now: time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC)}),
		WithBaseDir(dir),
		WithTimestampDirEnabled(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Write(&server_api.InputJSON{Id: 24601})
	if err != nil {
		t.Fatal(err)
	}
	// The file should exist.
	if _, err := os.Stat(dir + "/block_inputs_24601.json"); err != nil {
		t.Error(err)
	}
}
