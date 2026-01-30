// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
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

func TestRestfulServerURLsFromListWithWait_Success(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	urlsIn := []string{"https://supersecret.nowhere.com:9871", "http://www.google.com"}
	listContents := urlsIn[0] + " \t" + urlsIn[1]
	port, server := newListHttpServerForTest(t, &stringHandler{listContents})

	listUrl := fmt.Sprintf("http://localhost:%d", port)
	urls, err := RestfulServerURLsFromListWithWait(ctx, listUrl, 2*time.Second)
	Require(t, err)
	if !stringListIsPermutation(urlsIn, urls) {
		t.Fatal("URLs don't match")
	}

	err = server.Shutdown(ctx)
	Require(t, err)
}

func TestRestfulServerURLsFromListWithWait_RetrySuccess(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	urlsIn := []string{"https://supersecret.nowhere.com:9871", "http://www.google.com"}
	listContents := urlsIn[0] + " \t" + urlsIn[1]
	// Force a few failures to then succeed
	port, server := newListHttpServerForTest(
		t,
		Handlers(
			&erroringHandler{},
			&connectionClosingHandler{},
			&connectionClosingHandler{},
			&erroringHandler{},
			&stringHandler{listContents},
		),
	)

	listUrl := fmt.Sprintf("http://localhost:%d", port)
	// Wait up to 6 seconds, should succeed after the few retries
	urls, err := RestfulServerURLsFromListWithWait(ctx, listUrl, 6*time.Second)
	Require(t, err)
	if !stringListIsPermutation(urlsIn, urls) {
		t.Fatal("URLs don't match")
	}

	err = server.Shutdown(ctx)
	Require(t, err)
}

func TestRestfulServerURLsFromListWithWait_Timeout(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Server always returns errors
	port, server := newListHttpServerForTest(t, &erroringHandler{})

	listUrl := fmt.Sprintf("http://localhost:%d", port)
	// Wait only 1 second, should timeout
	urls, err := RestfulServerURLsFromListWithWait(ctx, listUrl, 1*time.Second)
	if err == nil {
		t.Fatal("Expected timeout error")
	}
	if urls != nil {
		t.Fatal("Expected nil URLs on timeout")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("Expected timeout error, got: %v", err)
	}

	err = server.Shutdown(ctx)
	Require(t, err)
}

func TestRestfulServerURLsFromListWithWait_ContextCancellation(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithCancel(context.Background())

	// Server always returns errors
	port, server := newListHttpServerForTest(t, &erroringHandler{})

	listUrl := fmt.Sprintf("http://localhost:%d", port)

	// Cancel context after a short delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	// Wait up to 5 seconds, but context should cancel first
	urls, err := RestfulServerURLsFromListWithWait(ctx, listUrl, 5*time.Second)
	if err == nil {
		t.Fatal("Expected context cancellation error")
	}
	if urls != nil {
		t.Fatal("Expected nil URLs on context cancellation")
	}
	if err != context.Canceled {
		t.Fatalf("Expected context.Canceled, got: %v", err)
	}

	err = server.Shutdown(context.Background())
	Require(t, err)
}

func TestRestfulServerURLsFromListWithWait_MalformedURL(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a malformed URL that will cause a parse error
	malformedUrl := "://invalid-url"
	urls, err := RestfulServerURLsFromListWithWait(ctx, malformedUrl, 5*time.Second)
	if err == nil {
		t.Fatal("Expected error for malformed URL")
	}
	if urls != nil {
		t.Fatal("Expected nil URLs on parse error")
	}
	// Should return immediately without retrying for parse errors
	if !strings.Contains(err.Error(), "parse") && !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("Expected parse/malformed error, got: %v", err)
	}
}

func TestRestfulServerURLsFromListWithWait_ConnectionRefused(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a URL that will refuse connection
	listUrl := "http://localhost:99999"
	urls, err := RestfulServerURLsFromListWithWait(ctx, listUrl, 1*time.Second)
	if err == nil {
		t.Fatal("Expected error for connection refused")
	}
	if urls != nil {
		t.Fatal("Expected nil URLs on connection error")
	}
	// Should timeout after retrying
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("Expected timeout error, got: %v", err)
	}
}
