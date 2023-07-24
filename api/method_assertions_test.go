package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAssertions(t *testing.T) {
	s, _ := NewTestServer(t)

	req, err := http.NewRequest("GET", "/assertions", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Serve the request with the http recorder.
	s.Router().ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNotImplemented {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotImplemented)
	}
}

func TestGetAssertion(t *testing.T) {
	s, _ := NewTestServer(t)

	req, err := http.NewRequest("GET", "/assertions/foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Serve the request with the http recorder.
	s.Router().ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNotImplemented {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotImplemented)
	}
}
