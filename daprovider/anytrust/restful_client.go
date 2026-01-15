// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

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

	"github.com/offchainlabs/nitro/daprovider/anytrust/tree"
	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
)

// RestfulClient implements anytrustutil.Reader
type RestfulClient struct {
	url string
}

func (c *RestfulClient) String() string {
	return fmt.Sprintf("Restful AnyTrust client for %s", c.url)
}

func NewRestfulClient(protocol string, host string, port int) *RestfulClient {
	return &RestfulClient{
		url: fmt.Sprintf("%s://%s:%d", protocol, host, port),
	}
}

func NewRestfulClientFromURL(url string) (*RestfulClient, error) {
	if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		return nil, fmt.Errorf("protocol prefix 'http://' or 'https://' must be specified for RestfulClient; got '%s'", url)

	}
	return &RestfulClient{
		url: url,
	}, nil
}

func (c *RestfulClient) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
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
	var response RestfulServerResponse
	if err := jsonDecoder.Decode(&response); err != nil {
		return nil, err
	}

	b64Decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader([]byte(response.Data)))
	decodedBytes, err := io.ReadAll(b64Decoder)
	if err != nil {
		return nil, err
	}
	if !tree.ValidHash(hash, decodedBytes) {
		return nil, anytrustutil.ErrHashMismatch
	}

	return decodedBytes, nil
}

func (c *RestfulClient) HealthCheck(ctx context.Context) error {
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

func (c *RestfulClient) ExpirationPolicy(ctx context.Context) (anytrustutil.ExpirationPolicy, error) {
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

	var response RestfulServerResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return -1, err
	}

	return anytrustutil.StringToExpirationPolicy(response.ExpirationPolicy)
}
