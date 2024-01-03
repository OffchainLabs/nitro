package api

import "net/http"

// healthzHandler returns OK if ready to serve requests.
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		log.Error("could not write response body", "err", err)
	}
}
