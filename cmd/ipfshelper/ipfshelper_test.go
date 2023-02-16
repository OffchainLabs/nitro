package ipfshelper

import (
	"bytes"
	"context"
	"math/rand"
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
	testData := make([]byte, 1024*1024)
	_, err = rand.Read(testData)
	testhelpers.RequireImpl(t, err)
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
	testhelpers.RequireImpl(t, err)
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
	testTimeout := 300 * time.Millisecond
	timeoutCtx, cancel := context.WithTimeout(ctx, testTimeout)
	defer cancel()
	_, err = ipfsD.DownloadFile(timeoutCtx, testFileCid, t.TempDir())
	if err == nil {
		testhelpers.FailImpl(t, "Download attempt did not fail as expected")
	}
	err = ipfsD.Close()
	testhelpers.RequireImpl(t, err)
}

func TestNormalizeCidString(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected string
	}{
		{"ipfs://QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ", "/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"},
		{"ipns://k51qzi5uqu5dlvj2baxnqndepeb86cbk3ng7n3i46uzyxzyqj2xjonzllnv0v8", "/ipns/k51qzi5uqu5dlvj2baxnqndepeb86cbk3ng7n3i46uzyxzyqj2xjonzllnv0v8"},
		{"ipns://docs.ipfs.tech/introduction/", "/ipns/docs.ipfs.tech/introduction/"},
		{"/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ", "/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"},
		{"/ipns/k51qzi5uqu5dlvj2baxnqndepeb86cbk3ng7n3i46uzyxzyqj2xjonzllnv0v8", "/ipns/k51qzi5uqu5dlvj2baxnqndepeb86cbk3ng7n3i46uzyxzyqj2xjonzllnv0v8"},
		{"QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ", "QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"},
	} {
		if res := normalizeCidString(test.input); res != test.expected {
			testhelpers.FailImpl(t, "Failed to normalize cid string, input: ", test.input, " got: ", res, " expected: ", test.expected)
		}
	}
}

func TestCanBeIpfsPath(t *testing.T) {
	correctPaths := []string{
		"QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ipns/k51qzi5uqu5dlvj2baxnqndepeb86cbk3ng7n3i46uzyxzyqj2xjonzllnv0v8",
		"/ipns/docs.ipfs.tech/introduction/",
		"ipfs://QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"ipns://k51qzi5uqu5dlvj2baxnqndepeb86cbk3ng7n3i46uzyxzyqj2xjonzllnv0v8",
	}
	for _, path := range correctPaths {
		if !CanBeIpfsPath(path) {
			testhelpers.FailImpl(t, "false negative result for path:", path)
		}
	}
	incorrectPaths := []string{"www.ipfs.tech", "https://www.ipfs.tech", "QmIncorrect"}
	for _, path := range incorrectPaths {
		if CanBeIpfsPath(path) {
			testhelpers.FailImpl(t, "false positive result for path:", path)
		}
	}
}
