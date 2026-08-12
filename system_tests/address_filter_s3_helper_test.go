// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/johannesboyne/gofakes3"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/s3client"
	"github.com/offchainlabs/nitro/util/s3syncer"
	"github.com/offchainlabs/nitro/util/s3syncer/s3syncertest"
)

var testFilterSalt = uuid.MustParse("3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7")

type fakeS3AddressFilter struct {
	backend    gofakes3.Backend
	bucket     string
	objectKeys []string
	scheme     addressfilter.HashingScheme
}

func setupFakeS3AddressFilter(t *testing.T, builder *NodeBuilder) *fakeS3AddressFilter {
	return setupFakeS3AddressFilterWithScheme(t, builder, addressfilter.HashingSchemeRawBytesInput)
}

func setupFakeS3AddressFilterWithScheme(t *testing.T, builder *NodeBuilder, scheme addressfilter.HashingScheme) *fakeS3AddressFilter {
	return setupFakeS3AddressFilterForConfig(t, builder.execConfig, scheme, 1)
}

func setupFakeS3AddressFilterMultiFile(t *testing.T, builder *NodeBuilder, numFiles int) *fakeS3AddressFilter {
	return setupFakeS3AddressFilterForConfig(t, builder.execConfig, addressfilter.HashingSchemeRawBytesInput, numFiles)
}

func setupFakeS3AddressFilterForConfig(t *testing.T, execConfig *gethexec.Config, scheme addressfilter.HashingScheme, numFiles int) *fakeS3AddressFilter {
	t.Helper()
	const bucket = "addressfilter-test"
	objects := make(map[string][]byte, numFiles)
	objectKeys := make([]string, numFiles)
	for i := range objectKeys {
		objectKeys[i] = fmt.Sprintf("filtered-addresses-hashed-list-%d.json", i)
		objects[objectKeys[i]] = hashListJSON(t, scheme, nil)
	}
	endpoint, backend := s3syncertest.NewFakeS3(t, bucket, objects)

	filteringConfig := &execConfig.TransactionFiltering
	filteringConfig.Enable = true
	if filteringConfig.TransactionFiltererRPCClient.URL == "" {
		filteringConfig.TransactionFiltererRPCClient.URL = gethexec.TransactionFiltererURLNone
	}
	files := make([]addressfilter.FileConfig, numFiles)
	for i, objectKey := range objectKeys {
		files[i] = testAddressFilterFileConfig(t, endpoint, bucket, objectKey)
	}
	filteringConfig.AddressFilter.Files = files

	return &fakeS3AddressFilter{
		backend:    backend,
		bucket:     bucket,
		objectKeys: objectKeys,
		scheme:     scheme,
	}
}

func testAddressFilterFileConfig(t *testing.T, endpoint, bucket, objectKey string) addressfilter.FileConfig {
	return addressfilter.FileConfig{
		Config: s3syncer.Config{
			Config: s3client.Config{
				Region:    "us-east-1",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
				Endpoint:  endpoint,
			},
			Bucket:      bucket,
			ObjectKey:   objectKey,
			ChunkSizeMB: s3syncer.DefaultS3Config.ChunkSizeMB,
			MaxRetries:  s3syncer.DefaultS3Config.MaxRetries,
			Concurrency: s3syncer.DefaultS3Config.Concurrency,
			DownloadDir: t.TempDir(),
		},
		PollInterval: addressfilter.DefaultFileConfig.PollInterval,
	}
}

func (f *fakeS3AddressFilter) setFilteredAddressesForFile(t *testing.T, ctx context.Context, execNode *gethexec.ExecutionNode, fileIdx int, addrs []common.Address) {
	t.Helper()
	body := hashListJSON(t, f.scheme, addrs)
	_, err := f.backend.PutObject(f.bucket, f.objectKeys[fileIdx], map[string]string{}, bytes.NewReader(body), int64(len(body)), &gofakes3.PutConditions{})
	Require(t, err)
	Require(t, execNode.AddressFilterService.TriggerSyncForTest(t, ctx))
	require.Equal(t, len(addrs), execNode.AddressFilterService.GetHashStore(t, fileIdx).Size(), "filter list did not reach the node")
}

func hashListJSON(t *testing.T, scheme addressfilter.HashingScheme, addrs []common.Address) []byte {
	t.Helper()
	hashes := make([]string, len(addrs))
	switch scheme {
	case addressfilter.HashingSchemePlaintext:
		for i, addr := range addrs {
			hashes[i] = addr.Hex()
		}
	case addressfilter.HashingSchemeRawBytesInput:
		for i, addr := range addrs {
			hashes[i] = addressfilter.HashRawBytesInput(testFilterSalt, addr).Hex()
		}
	default:
		hashPrefix := addressfilter.GetHashStringInputPrefix(testFilterSalt)
		for i, addr := range addrs {
			hashes[i] = addressfilter.HashStringInputWithPrefix(hashPrefix, addr).Hex()
		}
	}
	payload := struct {
		Id            string   `json:"id"`
		Salt          string   `json:"salt,omitempty"`
		HashingScheme string   `json:"hashing_scheme"`
		Hashes        []string `json:"hashes"`
	}{
		Id:            uuid.New().String(),
		HashingScheme: string(scheme),
		Hashes:        hashes,
	}
	// Omit the salt for plaintext so the tests exercise the salt-less payload path.
	if scheme != addressfilter.HashingSchemePlaintext {
		payload.Salt = testFilterSalt.String()
	}
	data, err := json.Marshal(payload)
	Require(t, err)
	return data
}

func (f *fakeS3AddressFilter) setFilteredAddresses(t *testing.T, ctx context.Context, execNode *gethexec.ExecutionNode, addrs []common.Address) {
	f.setFilteredAddressesForFile(t, ctx, execNode, 0, addrs)
}
