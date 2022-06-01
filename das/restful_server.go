// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
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
	Data             string `json:"data"`
	ExpirationPolicy string `json:"expirationPolicy"`
}

var cacheControlKey = http.CanonicalHeaderKey("cache-control")

const cacheControlValue = "public, max-age=2419200, immutable" // cache for up to 28 days
const healthRequestPath = "/health/"
const expirationPolicyRequestPath = "/expiration-policy/"
const getByHashRequestPath = "/get-by-hash/"

func (rds *RestfulDasServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestPath := r.URL.Path
	log.Debug("Got request", "requestPath", requestPath)
	switch {
	case strings.HasPrefix(requestPath, healthRequestPath):
		rds.HealthHandler(w, r, requestPath)
	case strings.HasPrefix(requestPath, expirationPolicyRequestPath):
		rds.ExpirationPolicyHandler(w, r, requestPath)
	case strings.HasPrefix(requestPath, getByHashRequestPath):
		rds.GetByHashHandler(w, r, requestPath)
	default:
		log.Warn("Unknown requestPath", "requestPath", requestPath)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// Health requests for remote health-checks
func (rds *RestfulDasServer) HealthHandler(w http.ResponseWriter, r *http.Request, requestPath string) {
	err := rds.storage.HealthCheck(r.Context())
	if err != nil {
		log.Warn("Unhealthy service", "path", requestPath, "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (rds *RestfulDasServer) ExpirationPolicyHandler(w http.ResponseWriter, r *http.Request, requestPath string) {
	expirationPolicy, err := rds.storage.ExpirationPolicy(r.Context()).String()
	if err != nil {
		log.Warn("Got invalid expiration policy", "path", requestPath, "expirationPolicy", expirationPolicy)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(RestfulDasServerResponse{ExpirationPolicy: expirationPolicy})
	if err != nil {
		log.Warn("Failed encoding and writing response", "path", requestPath, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (rds *RestfulDasServer) GetByHashHandler(w http.ResponseWriter, r *http.Request, requestPath string) {
	log.Debug("Got request", "requestPath", requestPath)

	hashBytes, err := hexutil.Decode(strings.TrimPrefix(requestPath, "/get-by-hash/"))
	if err != nil {
		log.Warn("Failed to decode hex-encoded hash", "path", requestPath, "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(hashBytes) < 32 {
		log.Warn("Decoded hash was too short", "path", requestPath, "len(hashBytes)", len(hashBytes))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	responseData, err := rds.storage.GetByHash(r.Context(), hashBytes[:32])
	if err != nil {
		log.Warn("Unable to find data", "path", requestPath, "err", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	encodedResponseData := make([]byte, base64.StdEncoding.EncodedLen(len(responseData)))
	base64.StdEncoding.Encode(encodedResponseData, responseData)
	var response RestfulDasServerResponse
	response.Data = string(encodedResponseData)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Warn("Failed encoding and writing response", "path", requestPath, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header()[cacheControlKey] = []string{cacheControlValue}
}

func (rds *RestfulDasServer) GetServerExitedChan() <-chan interface{} { // channel will close when server terminates
	return rds.httpServerExitedChan
}

func (rds *RestfulDasServer) WaitForShutdown() error {
	<-rds.httpServerExitedChan
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
