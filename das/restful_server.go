// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
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
		ret.httpServerError = err
		close(ret.httpServerExitedChan)
	}()

	return ret
}

func (rds *RestfulDasServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in ServeHttp")
	requestPath := r.URL.Path
	hexEncodedHash := strings.TrimPrefix(requestPath, "/get-by-hash/")
	hashBytes, err := hex.DecodeString(hexEncodedHash)
	if err != nil || len(hashBytes) != 32 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	responseBytes, err := rds.storage.GetByHash(r.Context(), hashBytes)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_, _ = w.Write(responseBytes) // w.Write will deal with errors itself
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
	return rds.httpServerError
}
