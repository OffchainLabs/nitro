package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func testFileLogger(t *testing.T, testCompressed bool) {
	t.Helper()
	testDir := t.TempDir()
	testFileName := "test-file"
	testFile := filepath.Join(testDir, testFileName)
	config := genericconf.DefaultFileLoggingConfig
	config.MaxSize = 1
	config.Compress = testCompressed
	config.File = testFile
	writer := fileLogger.NewWriter(&config)
	defer func() { testhelpers.RequireImpl(t, fileLogger.Close()) }()
	expected := []byte("dead beef ate bad beef")
	n, err := writer.Write(expected)
	testhelpers.RequireImpl(t, err)
	if n != len(expected) {
		testhelpers.FailImpl(t, "Failed to write to file logger, wrote wrong number of bytes, wrote:", n, "expected:", len(expected))
	}
	data, err := os.ReadFile(testFile)
	testhelpers.RequireImpl(t, err)
	if !bytes.Equal(data, expected) {
		testhelpers.FailImpl(t, "Data read from file is different then expected")
	}
	zeroes := make([]byte, 1024*1024)
	n, err = writer.Write(zeroes)
	testhelpers.RequireImpl(t, err)
	if n != len(zeroes) {
		testhelpers.FailImpl(t, "Failed to write to file logger, wrote wrong number of bytes, wrote:", n, "expected:", len(expected))
	}
	data, err = os.ReadFile(testFile)
	testhelpers.RequireImpl(t, err)
	if bytes.HasPrefix(data, expected) {
		testhelpers.FailImpl(t, "It seems that file wasn't rotated")
	}
	if testCompressed {
		time.Sleep(100 * time.Millisecond)
	}
	entries, err := os.ReadDir(testDir)
	testhelpers.RequireImpl(t, err)
	var gzFiles int
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), testFileName) {
			testhelpers.FailImpl(t, "Unexpected file in test dir:", entry.Name())
		}
		if strings.HasSuffix(entry.Name(), ".gz") {
			gzFiles++
		}
	}
	if testCompressed && gzFiles != 1 {
		testhelpers.FailImpl(t, "Unexpected number of gzip files in test dir:", gzFiles)
	}
	if len(entries) != 2 {
		testhelpers.FailImpl(t, "Unexpected number of files in test dir:", len(entries))
	}
}

func TestFileLoggerWithoutCompression(t *testing.T) {
	testFileLogger(t, false)
}

func TestFileLoggerWithCompression(t *testing.T) {
	testFileLogger(t, true)
}
