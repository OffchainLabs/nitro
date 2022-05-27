// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestRestfulServerList(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	urlsIn := []string{"https://supersecret.nowhere.com:9871", "http://www.google.com"}
	listContents := urlsIn[0] + " \t" + urlsIn[1]
	server := newListHttpServerForTest(LocalServerPortForTest, listContents)

	listUrl := "http://localhost:" + strconv.FormatInt(LocalServerPortForTest, 10) //nolint
	urls, err := RestfulServerURLsFromList(ctx, listUrl)
	Require(t, err)
	if len(urls) != 2 || (urls[0] != urlsIn[0] && urls[0] != urlsIn[1]) || (urls[1] != urlsIn[0] && urls[1] != urlsIn[1]) {
		t.Fatal()
	}

	err = server.Shutdown(ctx)
	Require(t, err)
}

func newListHttpServerForTest(port int64, contents string) *http.Server {
	server := &http.Server{
		Addr:    ":" + strconv.FormatInt(port, 10),
		Handler: &testHandler{contents},
	}
	go func() {
		_ = server.ListenAndServe()
	}()
	return server
}

type testHandler struct {
	contents string
}

func (th *testHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, _ = w.Write([]byte(th.contents))
}
