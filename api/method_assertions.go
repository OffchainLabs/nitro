package api

import "net/http"

func listAssertionsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	if _, err := w.Write([]byte("not implemented")); err != nil {
		log.Error("failed to write response body", "err", err)
	}
}

func getAssertionHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	if _, err := w.Write([]byte("not implemented")); err != nil {
		log.Error("failed to write response body", "err", err)
	}
}
