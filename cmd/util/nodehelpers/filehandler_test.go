package nodehelpers

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

func pollLogMessagesFromJSONFile(t *testing.T, path string, expected []string) ([]string, error) {
	t.Helper()
	var msgs []string
	var err error
Retry:
	for i := 0; i < 30; i++ {
		time.Sleep(20 * time.Millisecond)
		msgs, err = readLogMessagesFromJSONFile(t, path)
		if err != nil {
			continue
		}
		if len(msgs) == len(expected) {
			for i, m := range msgs {
				if m != expected[i] {
					continue Retry
				}
			}
			return msgs, nil
		}
	}
	return msgs, err
}

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
	fileHandler := globalFileHandlerFactory.newHandler(log.JSONFormat(), &config, func(path string) string { return path })
	defer func() { testhelpers.RequireImpl(t, globalFileHandlerFactory.close()) }()
	log.Root().SetHandler(fileHandler)
	expected := []string{"dead", "beef", "ate", "bad", "beef"}
	for _, e := range expected {
		log.Warn(e)
	}
	msgs, err := pollLogMessagesFromJSONFile(t, testFile, expected)
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
	msgs, err = pollLogMessagesFromJSONFile(t, testFile, []string{bigString})
	testhelpers.RequireImpl(t, err)
	if len(msgs) != 1 {
		testhelpers.FailImpl(t, "Unexpected number of messages in the logfile - possible file rotation failure, have: ", len(msgs), " wants: 1")
	}
	if msgs[0] != bigString {
		testhelpers.FailImpl(t, "Unexpected message logged to file, have: ", msgs[0], " want:", bigString)
	}
	var gzFiles int
	var entries []os.DirEntry
	for i := 0; i < 60; i++ {
		time.Sleep(20 * time.Millisecond)
		gzFiles = 0
		var err error
		entries, err = os.ReadDir(testDir)
		testhelpers.RequireImpl(t, err)
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), testFileName) {
				testhelpers.FailImpl(t, "Unexpected file in test dir:", entry.Name())
			}
			if strings.HasSuffix(entry.Name(), ".gz") {
				gzFiles++
			}
		}
		if len(entries) == 2 && (!testCompressed || gzFiles == 1) {
			break
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
