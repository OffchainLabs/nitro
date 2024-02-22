// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/pretty"
)

var (
	restGetByHashRequestGauge       = metrics.NewRegisteredGauge("arb/das/rest/getbyhash/requests", nil)
	restGetByHashSuccessGauge       = metrics.NewRegisteredGauge("arb/das/rest/getbyhash/success", nil)
	restGetByHashFailureGauge       = metrics.NewRegisteredGauge("arb/das/rest/getbyhash/failure", nil)
	restGetByHashReturnedBytesGauge = metrics.NewRegisteredGauge("arb/das/rest/getbyhash/bytes", nil)
	restGetByHashDurationHistogram  = metrics.NewRegisteredHistogram("arb/das/rest/getbyhash/duration", nil, metrics.NewBoundedHistogramSample())
)

type RestfulDasServer struct {
	server               *http.Server
	daReader             arbstate.DataAvailabilityReader
	daHealthChecker      DataAvailabilityServiceHealthChecker
	httpServerExitedChan chan interface{}
	httpServerError      error
}

func NewRestfulDasServer(address string, port uint64, restServerTimeouts genericconf.HTTPServerTimeoutConfig, daReader arbstate.DataAvailabilityReader, daHealthChecker DataAvailabilityServiceHealthChecker) (*RestfulDasServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return nil, err
	}
	return NewRestfulDasServerOnListener(listener, restServerTimeouts, daReader, daHealthChecker)
}

func NewRestfulDasServerOnListener(listener net.Listener, restServerTimeouts genericconf.HTTPServerTimeoutConfig, daReader arbstate.DataAvailabilityReader, daHealthChecker DataAvailabilityServiceHealthChecker) (*RestfulDasServer, error) {

	ret := &RestfulDasServer{
		daReader:             daReader,
		daHealthChecker:      daHealthChecker,
		httpServerExitedChan: make(chan interface{}),
	}

	ret.server = &http.Server{
		Handler:           ret,
		ReadTimeout:       restServerTimeouts.ReadTimeout,
		ReadHeaderTimeout: restServerTimeouts.ReadHeaderTimeout,
		WriteTimeout:      restServerTimeouts.WriteTimeout,
		IdleTimeout:       restServerTimeouts.IdleTimeout,
	}

	go func() {
		// #nosec G114
		err := ret.server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			ret.httpServerError = err
		}
		close(ret.httpServerExitedChan)
	}()

	return ret, nil
}

type RestfulDasServerResponse struct {
	Data             string `json:"data,omitempty"`
	ExpirationPolicy string `json:"expirationPolicy,omitempty"`
}

var cacheControlKey = http.CanonicalHeaderKey("cache-control")

const cacheControlValueDefault = "public, max-age=1"                                 // cache for up to 1 second (Used to reduce DOS possibility)
const cacheControlValueForSuccessfulGetByHash = "public, max-age=2419200, immutable" // cache for up to 28 days
const healthRequestPath = "/health"
const expirationPolicyRequestPath = "/expiration-policy/"
const getByHashRequestPath = "/get-by-hash/"

func (rds *RestfulDasServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header()[cacheControlKey] = []string{cacheControlValueDefault}
	requestPath := path.Clean(r.URL.Path)
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

// HealthHandler implements health requests for remote health-checks
func (rds *RestfulDasServer) HealthHandler(w http.ResponseWriter, r *http.Request, requestPath string) {
	err := rds.daHealthChecker.HealthCheck(r.Context())
	if err != nil {
		log.Warn("Unhealthy service", "path", requestPath, "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (rds *RestfulDasServer) ExpirationPolicyHandler(w http.ResponseWriter, r *http.Request, requestPath string) {
	expirationPolicy, err := rds.daReader.ExpirationPolicy(r.Context())
	if err != nil {
		log.Warn("Error retrieving expiration policy", "path", requestPath, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	expirationPolicyString, err := expirationPolicy.String()
	if err != nil {
		log.Warn("Got invalid expiration policy", "path", requestPath, "expirationPolicy", expirationPolicy)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(RestfulDasServerResponse{ExpirationPolicy: expirationPolicyString})
	if err != nil {
		log.Warn("Failed encoding and writing response", "path", requestPath, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (rds *RestfulDasServer) GetByHashHandler(w http.ResponseWriter, r *http.Request, requestPath string) {
	log.Debug("Got request", "requestPath", requestPath)
	restGetByHashRequestGauge.Inc(1)
	start := time.Now()
	success := false
	defer func() {
		if success {
			restGetByHashSuccessGauge.Inc(1)
		} else {
			restGetByHashFailureGauge.Inc(1)
		}
		restGetByHashDurationHistogram.Update(time.Since(start).Nanoseconds())
	}()

	hashBytes, err := DecodeStorageServiceKey(strings.TrimPrefix(requestPath, "/get-by-hash/"))
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

	responseData, err := rds.daReader.GetByHash(r.Context(), common.BytesToHash(hashBytes[:32]))
	if err != nil {
		log.Warn("Unable to find data", "path", requestPath, "err", err, "remoteAddr", r.RemoteAddr)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	log.Trace("RestfulDasServer.ServeHTTP returning", "message", pretty.FirstFewBytes(responseData), "message length", len(responseData))

	encodedResponseData := make([]byte, base64.StdEncoding.EncodedLen(len(responseData)))
	base64.StdEncoding.Encode(encodedResponseData, responseData)
	var response RestfulDasServerResponse
	response.Data = string(encodedResponseData)
	restGetByHashReturnedBytesGauge.Inc(int64(len(response.Data)))

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Warn("Failed encoding and writing response", "path", requestPath, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header()[cacheControlKey] = []string{cacheControlValueForSuccessfulGetByHash}
	success = true
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
