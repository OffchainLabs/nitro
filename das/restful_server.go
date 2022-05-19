// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

type RestfulDasServer struct {
	server               *http.Server
	storage              StorageService
	httpServerExitedChan chan interface{}
	httpServerError      error
}

func NewRestfulDasServerHTTP(address string, port uint64, storageService StorageService) *RestfulDasServer {
	ret := &RestfulDasServer{
		storage:              storageService,
		httpServerExitedChan: make(chan interface{}),
	}

	ret.server = &http.Server{
		Addr:    fmt.Sprint(address, ":", port),
		Handler: ret,
	}

	go func() {
		err := ret.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			ret.httpServerError = err
		}
		close(ret.httpServerExitedChan)
	}()

	return ret
}

type RestfulDasServerResponse struct {
	Data string `json:"data"`
}

func (rds *RestfulDasServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestPath := r.URL.Path
	urlEncodedBase32Hash := strings.TrimPrefix(requestPath, "/get-by-hash/")
	log.Debug("Got request", "requestPath", requestPath)

	// The DataHash bytes are base32 encoded, then URL encoded.
	// Base64 is not used since the '+' character is confused for
	// the URL encoding of ' '.
	base32Hash, err := url.QueryUnescape(urlEncodedBase32Hash)
	if err != nil {
		log.Warn("Bad URL encoding", "path", requestPath, "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hashDecoder := base32.NewDecoder(base32.StdEncoding, bytes.NewReader([]byte(base32Hash)))
	hashBytes, err := ioutil.ReadAll(hashDecoder)
	if err != nil || len(hashBytes) < 32 {
		log.Warn("Base32 decoding of hash failed", "base32Hash", base32Hash, "len(hashBytes)", len(hashBytes), "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	responseData, err := rds.storage.GetByHash(r.Context(), hashBytes[:32])
	if err != nil {
		log.Warn("Unable to find data", "base32Hash", base32Hash, "err", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	encodedResponseData := make([]byte, base64.StdEncoding.EncodedLen(len(responseData)))
	base64.StdEncoding.Encode(encodedResponseData, responseData)
	var response RestfulDasServerResponse
	response.Data = string(encodedResponseData)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Warn("Failed encoding and writing response", "requestPath", requestPath, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (rds *RestfulDasServer) GetServerExitedChan() <-chan interface{} { // channel will close when server terminates
	return rds.httpServerExitedChan
}

func (rds *RestfulDasServer) GetServerError() error {
	return rds.httpServerError
}

func (rds *RestfulDasServer) Shutdown() error {
	err := rds.server.Close()
	if err != nil {
		return err
	}
	<-rds.httpServerExitedChan
	return rds.httpServerError
}
