// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/daprovider/das/dastree"
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
)

// RestfulDasClient implements dasutil.DASReader
type RestfulDasClient struct {
	url string
}

func (c *RestfulDasClient) String() string {
	return fmt.Sprintf("Restful DAS client for %s", c.url)
}

func NewRestfulDasClient(protocol string, host string, port int) *RestfulDasClient {
	return &RestfulDasClient{
		url: fmt.Sprintf("%s://%s:%d", protocol, host, port),
	}
}

func NewRestfulDasClientFromURL(url string) (*RestfulDasClient, error) {
	if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		return nil, fmt.Errorf("protocol prefix 'http://' or 'https://' must be specified for RestfulDasClient; got '%s'", url)

	}
	return &RestfulDasClient{
		url: url,
	}, nil
}

func (c *RestfulDasClient) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.url+getByHashRequestPath+EncodeStorageServiceKey(hash), nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error with status %d returned by server: %s", res.StatusCode, http.StatusText(res.StatusCode))
	}

	jsonDecoder := json.NewDecoder(res.Body)
	var response RestfulDasServerResponse
	if err := jsonDecoder.Decode(&response); err != nil {
		return nil, err
	}

	b64Decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader([]byte(response.Data)))
	decodedBytes, err := io.ReadAll(b64Decoder)
	if err != nil {
		return nil, err
	}
	if !dastree.ValidHash(hash, decodedBytes) {
		return nil, dasutil.ErrHashMismatch
	}

	return decodedBytes, nil
}

func (c *RestfulDasClient) HealthCheck(ctx context.Context) error {
	res, err := http.Get(c.url + healthRequestPath)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error with status %d returned by server: %s", res.StatusCode, http.StatusText(res.StatusCode))
	}
	return nil
}

func (c *RestfulDasClient) ExpirationPolicy(ctx context.Context) (dasutil.ExpirationPolicy, error) {
	res, err := http.Get(c.url + expirationPolicyRequestPath)
	if err != nil {
		return -1, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("HTTP error with status %d returned by server: %s", res.StatusCode, http.StatusText(res.StatusCode))
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return -1, fmt.Errorf("failed to read response body: %w", err)
	}

	var response RestfulDasServerResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return -1, err
	}

	return dasutil.StringToExpirationPolicy(response.ExpirationPolicy)
}
