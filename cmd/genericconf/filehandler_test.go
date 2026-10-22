// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package genericconf

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"

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
		msgString, ok := msg.(string)
		if !ok {
			testhelpers.FailImpl(t, "Incorrect record, msg is not a string", "record", record)
		}
		messages = append(messages, msgString)
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
	config := DefaultFileLoggingConfig
	config.MaxSize = 1
	config.Compress = testCompressed
	config.File = testFile
	handler, err := HandlerFromLogType("json", globalFileLoggerFactory.newFileWriter(&config, testFile))
	defer func() { testhelpers.RequireImpl(t, globalFileLoggerFactory.close()) }()
	testhelpers.RequireImpl(t, err)
	log.SetDefault(log.NewLogger(handler))
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

// A runtime crash can only be exercised by really crashing, so the child
// re-execs, rotates the log, then panics; the .crash file must survive rotation.
func TestCrashOutputWrittenToLogFile(t *testing.T) {
	const marker = "nitro-test-crash-marker"
	if logFile := os.Getenv("TEST_CRASH_LOG_FILE"); logFile != "" {
		config := DefaultFileLoggingConfig
		config.File = logFile
		config.MaxSize = 1
		config.Compress = false
		if err := InitLog("json", "info", &config, func(s string) string { return s }); err != nil {
			t.Fatalf("InitLog failed in child: %v", err)
		}
		// Force a rotation before crashing; the .crash file must survive it.
		big := strings.Repeat("x", 512*1024)
		log.Warn(big)
		log.Warn(big)
		dir := filepath.Dir(logFile)
		base := strings.TrimSuffix(filepath.Base(logFile), filepath.Ext(logFile))
		rotated := false
		for i := 0; i < 100 && !rotated; i++ {
			entries, _ := os.ReadDir(dir)
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), base+"-") {
					rotated = true
				}
			}
			if !rotated {
				time.Sleep(20 * time.Millisecond)
			}
		}
		if !rotated {
			// distinct message so the parent's marker check fails instead of passing without a rotation
			panic("rotation did not occur")
		}
		panic(marker)
	}

	logFile := filepath.Join(t.TempDir(), "node.log")
	testBinary := os.Args[0]
	cmd := exec.Command(testBinary, "-test.run=^TestCrashOutputWrittenToLogFile$")
	cmd.Env = append(os.Environ(), "TEST_CRASH_LOG_FILE="+logFile)
	output, err := cmd.CombinedOutput()
	if err == nil {
		testhelpers.FailImpl(t, "expected child process to crash, but it exited cleanly; output:", string(output))
	}
	data, err := os.ReadFile(logFile + ".crash")
	testhelpers.RequireImpl(t, err)
	if !strings.Contains(string(data), marker) {
		testhelpers.FailImpl(t, "panic message not written to crash file; contents:", string(data))
	}
	if !strings.Contains(string(data), "goroutine") {
		testhelpers.FailImpl(t, "panic traceback not written to crash file; contents:", string(data))
	}

	// crash output must live in the dedicated .crash file, never in a rotated log
	dir := filepath.Dir(logFile)
	base := strings.TrimSuffix(filepath.Base(logFile), filepath.Ext(logFile))
	entries, err := os.ReadDir(dir)
	testhelpers.RequireImpl(t, err)
	rotatedFound := false
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), base+"-") {
			continue
		}
		rotatedFound = true
		rotatedData, err := os.ReadFile(filepath.Join(dir, e.Name()))
		testhelpers.RequireImpl(t, err)
		if strings.Contains(string(rotatedData), marker) {
			testhelpers.FailImpl(t, "crash output leaked into rotated log file:", e.Name())
		}
	}
	if !rotatedFound {
		testhelpers.FailImpl(t, "expected a rotated backup log file; rotation did not occur")
	}
}

// InitLog must reject bad input rather than half-applying a (re)configuration.
func TestInitLogRejectsInvalidLogType(t *testing.T) {
	if err := InitLog("not-a-log-type", "info", &FileLoggingConfig{Enable: false}, nil); err == nil {
		testhelpers.FailImpl(t, "expected InitLog to reject an unknown log type")
	}
}

// TestCrashOutputReloadRearmsCrashFile checks that a second InitLog (a reload)
// re-points crash output at the new file and not the old one.
func TestCrashOutputReloadRearmsCrashFile(t *testing.T) {
	const marker = "nitro-test-reload-marker"
	if dir := os.Getenv("TEST_REARM_DIR"); dir != "" {
		id := func(s string) string { return s }
		cfgA := DefaultFileLoggingConfig
		cfgA.File = filepath.Join(dir, "a.log")
		if err := InitLog("json", "info", &cfgA, id); err != nil {
			t.Fatalf("InitLog A failed in child: %v", err)
		}
		cfgB := DefaultFileLoggingConfig
		cfgB.File = filepath.Join(dir, "b.log")
		if err := InitLog("json", "info", &cfgB, id); err != nil {
			t.Fatalf("InitLog B failed in child: %v", err)
		}
		panic(marker)
	}

	dir := t.TempDir()
	testBinary := os.Args[0]
	cmd := exec.Command(testBinary, "-test.run=^TestCrashOutputReloadRearmsCrashFile$")
	cmd.Env = append(os.Environ(), "TEST_REARM_DIR="+dir)
	output, err := cmd.CombinedOutput()
	if err == nil {
		testhelpers.FailImpl(t, "expected child process to crash; output:", string(output))
	}
	newData, err := os.ReadFile(filepath.Join(dir, "b.log.crash"))
	testhelpers.RequireImpl(t, err)
	if !strings.Contains(string(newData), marker) {
		testhelpers.FailImpl(t, "panic not written to the re-armed crash file; contents:", string(newData))
	}
	if oldData, _ := os.ReadFile(filepath.Join(dir, "a.log.crash")); strings.Contains(string(oldData), marker) {
		testhelpers.FailImpl(t, "panic written to the stale crash file after reload")
	}
}

func TestFileLoggerWithoutCompression(t *testing.T) {
	testFileHandler(t, false)
}

func TestFileLoggerWithCompression(t *testing.T) {
	testFileHandler(t, true)
}
