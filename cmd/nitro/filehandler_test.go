package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func readLogMessagesFromJSONFile(t *testing.T, path string) ([]string, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{}, err
	}
	messages := []string{}
	decoder := json.NewDecoder(bytes.NewBuffer(data))
	var record map[string]interface{}
	for {
		if err = decoder.Decode(&record); err != nil {
			break
		}
		msg, ok := record["msg"]
		if !ok {
			testhelpers.FailImpl(t, "Incorrect record, msg key is missing", "record", record)
		}
		messages = append(messages, msg.(string))
	}
	if errors.Is(err, io.EOF) {
		return messages, nil
	}
	return []string{}, err
}

func testFileHandler(t *testing.T, testCompressed bool) {
	t.Helper()
	testDir := t.TempDir()
	testFileName := "test-file"
	testFile := filepath.Join(testDir, testFileName)
	config := genericconf.DefaultFileLoggingConfig
	config.MaxSize = 1
	config.Compress = testCompressed
	config.File = testFile
	fileHandler := globalFileHandlerFactory.newHandler(log.JSONFormat(), &config)
	defer func() { testhelpers.RequireImpl(t, globalFileHandlerFactory.close()) }()
	log.Root().SetHandler(fileHandler)
	expected := []string{"dead", "beef", "ate", "bad", "beef"}
	for _, e := range expected {
		log.Warn(e)
	}
	time.Sleep(100 * time.Millisecond)
	msgs, err := readLogMessagesFromJSONFile(t, testFile)
	testhelpers.RequireImpl(t, err)
	if len(msgs) != len(expected) {
		testhelpers.FailImpl(t, "Unexpected number of messages logged to file")
	}
	for i, m := range msgs {
		if m != expected[i] {
			testhelpers.FailImpl(t, "Unexpected message logged to file, have: ", m, " want:", expected[i])
		}
	}
	bigData := make([]byte, 512*1024)
	for i := range bigData {
		bigData[i] = 'x'
	}
	bigString := string(bigData)
	// make sure logs size exceeds 1MB, while keeping log msg < 1MB
	log.Warn(bigString)
	log.Warn(bigString)
	time.Sleep(100 * time.Millisecond)
	msgs, err = readLogMessagesFromJSONFile(t, testFile)
	testhelpers.RequireImpl(t, err)
	if len(msgs) != 1 {
		testhelpers.FailImpl(t, "Unexpected number of messages in the logfile - possible file rotation failure, have: ", len(msgs), " wants: 1")
	}
	if msgs[0] != bigString {
		testhelpers.FailImpl(t, "Unexpected message logged to file, have: ", msgs[0], " want:", bigString)
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
	testFileHandler(t, false)
}

func TestFileLoggerWithCompression(t *testing.T) {
	testFileHandler(t, true)
}
