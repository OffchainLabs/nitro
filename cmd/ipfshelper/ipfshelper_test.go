package ipfshelper

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func getTempFileWithData(t *testing.T, data []byte) string {
	path := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(path, []byte(data), 0600)
	testhelpers.RequireImpl(t, err)
	return path
}

func fileDataEqual(t *testing.T, path string, expected []byte) bool {
	data, err := os.ReadFile(path)
	testhelpers.RequireImpl(t, err)
	return bytes.Equal(data, expected)
}

func TestIpfsHelper(t *testing.T) {
	ctx := context.Background()
	ipfsA, err := createIpfsHelperImpl(ctx, t.TempDir(), false, []string{}, "test")
	testhelpers.RequireImpl(t, err)
	// add a test file to node A
	testData := []byte("here be dragons")
	testFile := getTempFileWithData(t, testData)
	ipfsTestFilePath, err := ipfsA.AddFile(ctx, testFile, false)
	testhelpers.RequireImpl(t, err)
	testFileCid := ipfsTestFilePath.Cid().String()
	addrsA, err := ipfsA.GetPeerHostAddresses()
	testhelpers.RequireImpl(t, err)
	// create node B connected to node A
	ipfsB, err := createIpfsHelperImpl(ctx, t.TempDir(), false, addrsA, "test")
	testhelpers.RequireImpl(t, err)
	// download the test file with node B
	downloadedFile, err := ipfsB.DownloadFile(ctx, testFileCid, t.TempDir())
	testhelpers.RequireImpl(t, err)
	if !fileDataEqual(t, downloadedFile, testData) {
		testhelpers.FailImpl(t, "Downloaded file does not contain expected data")
	}
	// clean up node A and test downloading the file from yet another node C
	err = ipfsA.Close()
	os.RemoveAll(ipfsA.repoPath)
	testhelpers.RequireImpl(t, err)
	addrsB, err := ipfsB.GetPeerHostAddresses()
	testhelpers.RequireImpl(t, err)
	ipfsC, err := createIpfsHelperImpl(ctx, t.TempDir(), false, addrsB, "test")
	testhelpers.RequireImpl(t, err)
	downloadedFile, err = ipfsC.DownloadFile(ctx, testFileCid, t.TempDir())
	if !fileDataEqual(t, downloadedFile, testData) {
		testhelpers.FailImpl(t, "Downloaded file does not contain expected data")
	}
	// make sure closing B and C nodes (A already closed) will make it impossible to download the test file from new node D
	ipfsD, err := createIpfsHelperImpl(ctx, t.TempDir(), false, addrsB, "test")
	testhelpers.RequireImpl(t, err)
	err = ipfsB.Close()
	testhelpers.RequireImpl(t, err)
	err = ipfsC.Close()
	testhelpers.RequireImpl(t, err)
	testTimeout := 500 * time.Millisecond
	finished := make(chan interface{})
	go func() {
		downloadedFile, err = ipfsD.downloadFileImpl(ctx, testFileCid, t.TempDir(), testTimeout)
		if err == nil {
			testhelpers.FailImpl(t, "Download attempt did not fail as expected")
		}
		close(finished)
	}()
	select {
	case <-time.After(testTimeout + 100*time.Millisecond):
		testhelpers.FailImpl(t, "Download attempt did not time out as expected")
	case <-finished:
	}
	err = ipfsD.Close()
	testhelpers.RequireImpl(t, err)
}
