// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Implements SimpleDASReader
type RestfulDasClient struct {
	url string
}

func NewRestfulDasClient(protocol string, host string, port int) *RestfulDasClient {
	return &RestfulDasClient{
		url: fmt.Sprintf("%s://%s:%d/get-by-hash/", protocol, host, port),
	}
}

func (c *RestfulDasClient) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	base32EncodedHash := make([]byte, base32.StdEncoding.EncodedLen(len(hash)))
	base32.StdEncoding.Encode(base32EncodedHash, hash)

	res, err := http.Get(c.url + url.QueryEscape(string(base32EncodedHash)))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error with status %d returned by server: %s", res.StatusCode, http.StatusText(res.StatusCode))
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response RestfulDasServerResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader([]byte(response.Data)))
	decodedBytes, err := ioutil.ReadAll(decoder)
	if err != nil {
		return nil, err
	}

	return decodedBytes, nil
}
