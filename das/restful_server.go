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
	server  *http.Server
	storage StorageService
}

func NewRestfulDasServerHTTP(address string, storageService StorageService) *RestfulDasServer {
	ret := &RestfulDasServer{storage: storageService}

	ret.server = &http.Server{
		Addr:    address,
		Handler: ret,
	}

	go func() {
		_ = ret.server.ListenAndServe()
	}()

	return ret
}

func (rds *RestfulDasServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in ServeHttp")
	requestPath := r.URL.Path
	hexEncodedHash := strings.TrimPrefix(requestPath, "/get-by-hash/")
	hashBytes, err := hex.DecodeString(hexEncodedHash)
	if err != nil || len(hashBytes) != 32 {
		fmt.Println("A")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	responseBytes, err := rds.storage.GetByHash(r.Context(), hashBytes)
	if err != nil {
		fmt.Println("B")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_, _ = w.Write(responseBytes) // w.Write will deal with errors itself
}

func (rds *RestfulDasServer) Shutdown() error {
	return rds.server.Close()
}
