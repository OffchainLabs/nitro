package dbutil

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/node"
)

func testIsNotExistError(t *testing.T, dbEngine string, isNotExist func(error) bool) {
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stackConf.DBEngine = dbEngine
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to created test stack: %v", err)
	}
	defer stack.Close()
	readonly := true
	_, err = stack.OpenDatabaseWithExtraOptions("test", 16, 16, "", readonly, nil)
	if err == nil {
		t.Fatal("Opening non-existent database did not fail")
	}
	if !isNotExist(err) {
		t.Fatalf("Failed to classify error as not exist error - internal implementation of OpenDatabaseWithExtraOptions might have changed, err: %v", err)
	}
	err = errors.New("some other error")
	if isNotExist(err) {
		t.Fatalf("Classified other error as not exist, err: %v", err)
	}
}

func TestIsNotExistError(t *testing.T) {
	t.Run("TestIsPebbleNotExistError", func(t *testing.T) {
		testIsNotExistError(t, "pebble", isPebbleNotExistError)
	})
	t.Run("TestIsLeveldbNotExistError", func(t *testing.T) {
		testIsNotExistError(t, "leveldb", isLeveldbNotExistError)
	})
	t.Run("TestIsNotExistErrorWithPebble", func(t *testing.T) {
		testIsNotExistError(t, "pebble", IsNotExistError)
	})
	t.Run("TestIsNotExistErrorWithLeveldb", func(t *testing.T) {
		testIsNotExistError(t, "leveldb", IsNotExistError)
	})
}
