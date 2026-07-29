// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/johannesboyne/gofakes3"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/s3client"
	"github.com/offchainlabs/nitro/util/s3syncer"
	"github.com/offchainlabs/nitro/util/s3syncer/s3syncertest"
)

var testFilterSalt = uuid.MustParse("3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7")

type fakeS3AddressFilter struct {
	backend   gofakes3.Backend
	bucket    string
	objectKey string
	scheme    addressfilter.HashingScheme
}

func setupFakeS3AddressFilter(t *testing.T, builder *NodeBuilder) *fakeS3AddressFilter {
	return setupFakeS3AddressFilterWithScheme(t, builder, addressfilter.HashingSchemeRawBytesInput)
}

func setupFakeS3AddressFilterWithScheme(t *testing.T, builder *NodeBuilder, scheme addressfilter.HashingScheme) *fakeS3AddressFilter {
	return setupFakeS3AddressFilterForConfig(t, builder.execConfig, scheme)
}

func setupFakeS3AddressFilterForConfig(t *testing.T, execConfig *gethexec.Config, scheme addressfilter.HashingScheme) *fakeS3AddressFilter {
	t.Helper()
	const bucket = "addressfilter-test"
	const objectKey = "filtered-addresses-hashed-list.json"
	endpoint, backend := s3syncertest.NewFakeS3(t, bucket, map[string][]byte{
		objectKey: hashListJSON(t, scheme, nil),
	})
	f := &fakeS3AddressFilter{
		backend:   backend,
		bucket:    bucket,
		objectKey: objectKey,
		scheme:    scheme,
	}

	filteringConfig := &execConfig.TransactionFiltering
	filteringConfig.Enable = true
	if filteringConfig.TransactionFiltererRPCClient.URL == "" {
		filteringConfig.TransactionFiltererRPCClient.URL = gethexec.TransactionFiltererURLNone
	}
	filteringConfig.AddressFilter.S3 = s3syncer.Config{
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
	}
	return f
}

func hashListJSON(t *testing.T, scheme addressfilter.HashingScheme, addrs []common.Address) []byte {
	t.Helper()
	hashes := make([]string, len(addrs))
	if scheme == addressfilter.HashingSchemeRawBytesInput {
		for i, addr := range addrs {
			hashes[i] = addressfilter.HashRawBytesInput(testFilterSalt, addr).Hex()
		}
	} else {
		hashPrefix := addressfilter.GetHashStringInputPrefix(testFilterSalt)
		for i, addr := range addrs {
			hashes[i] = addressfilter.HashStringInputWithPrefix(hashPrefix, addr).Hex()
		}
	}
	payload := struct {
		Id            string   `json:"id"`
		Salt          string   `json:"salt"`
		HashingScheme string   `json:"hashing_scheme"`
		Hashes        []string `json:"hashes"`
	}{
		Id:            uuid.New().String(),
		Salt:          testFilterSalt.String(),
		HashingScheme: string(scheme),
		Hashes:        hashes,
	}
	data, err := json.Marshal(payload)
	Require(t, err)
	return data
}

func (f *fakeS3AddressFilter) setFilteredAddresses(t *testing.T, ctx context.Context, execNode *gethexec.ExecutionNode, addrs []common.Address) {
	t.Helper()
	body := hashListJSON(t, f.scheme, addrs)
	_, err := f.backend.PutObject(f.bucket, f.objectKey, map[string]string{}, bytes.NewReader(body), int64(len(body)), &gofakes3.PutConditions{})
	Require(t, err)
	Require(t, execNode.AddressFilterService.TriggerSyncForTest(t, ctx))
}
