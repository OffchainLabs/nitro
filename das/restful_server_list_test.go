// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestRestfulServerList(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	urlsIn := []string{"https://supersecret.nowhere.com:9871", "http://www.google.com"}
	listContents := urlsIn[0] + " \t" + urlsIn[1]
	port, server := newListHttpServerForTest(t, listContents)

	listUrl := fmt.Sprintf("http://localhost:%d", port)
	urls, err := RestfulServerURLsFromList(ctx, listUrl)
	Require(t, err)
	if !stringListIsPermutation(urlsIn, urls) {
		t.Fatal()
	}

	err = server.Shutdown(ctx)
	Require(t, err)
}

func TestRestfulServerListDaemon(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	urlsIn := []string{"https://supersecret.nowhere.com:9871", "http://www.google.com"}
	listContents := urlsIn[0] + " \t" + urlsIn[1]
	port, server := newListHttpServerForTest(t, listContents)

	listUrl := fmt.Sprintf("http://localhost:%d", port)

	listChan := StartRestfulServerListFetchDaemon(ctx, listUrl, 200*time.Millisecond)
	for i := 0; i < 4; i++ {
		list := <-listChan
		if !stringListIsPermutation(list, urlsIn) {
			t.Fatal(i)
		}
	}

	err := server.Shutdown(ctx)
	Require(t, err)
}

func stringListIsPermutation(lis1, lis2 []string) bool {
	if len(lis1) != len(lis2) {
		return false
	}
	lookup := make(map[string]bool)
	for _, s := range lis1 {
		lookup[s] = true
	}
	for _, s := range lis2 {
		if !lookup[s] {
			return false
		}
	}
	return true
}

func newListHttpServerForTest(t *testing.T, contents string) (int, *http.Server) {
	server := &http.Server{
		Handler: &testHandler{contents},
	}
	listener, err := net.Listen("tcp", ":0")
	Require(t, err)
	go func() {
		_ = server.Serve(listener)
	}()
	tcpAddr, _ := listener.Addr().(*net.TCPAddr)
	return tcpAddr.Port, server
}

type testHandler struct {
	contents string
}

func (th *testHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, _ = w.Write([]byte(th.contents))
}
