// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

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
	port, server := newListHttpServerForTest(t, &stringHandler{listContents})

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
	port, server := newListHttpServerForTest(t, &stringHandler{listContents})

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

func TestRestfulServerListDaemonWithErrors(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	urlsIn := []string{"https://supersecret.nowhere.com:9871", "http://www.google.com"}
	listContents := urlsIn[0] + " \t" + urlsIn[1]
	port, server := newListHttpServerForTest(
		t,
		Handlers(
			&connectionClosingHandler{},
			&connectionClosingHandler{},
			&stringHandler{listContents},
			&erroringHandler{},
			&erroringHandler{},
			&stringHandler{listContents},
			&erroringHandler{},
			&connectionClosingHandler{},
			&stringHandler{listContents},
		),
	)

	listUrl := fmt.Sprintf("http://localhost:%d", port)

	listChan := StartRestfulServerListFetchDaemon(ctx, listUrl, 200*time.Millisecond)
	for i := 0; i < 3; i++ {
		list := <-listChan
		if !stringListIsPermutation(list, urlsIn) {
			t.Fatal(i, "not a match")
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

func newListHttpServerForTest(t *testing.T, handler http.Handler) (int, *http.Server) {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	go func() {
		_ = server.Serve(listener)
	}()
	tcpAddr, _ := listener.Addr().(*net.TCPAddr)
	return tcpAddr.Port, server
}

type stringHandler struct {
	contents string
}

func (h *stringHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, _ = w.Write([]byte(h.contents))
}

type erroringHandler struct {
}

func (h *erroringHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(404)
}

type connectionClosingHandler struct {
}

func (h *connectionClosingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	panic("close connection")
}

type multiHandler struct {
	current  int
	handlers []http.Handler
}

func Handlers(hs ...http.Handler) *multiHandler {
	return &multiHandler{0, hs}
}

func (h *multiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	i := h.current % len(h.handlers)
	h.current++
	h.handlers[i].ServeHTTP(w, req)
}
