package api

import "net/http"

func (s *Server) listAssertionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ah, err := s.assertions.LatestCreatedAssertionHashes(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := make([]*Assertion, len(ah))

	for idx, h := range ah {
		aci, err := s.assertions.ReadAssertionCreationInfo(ctx, h)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		resp[idx] = AssertionCreatedInfoToAssertion(aci)
	}

	if err := writeJSONResponse(w, 200, resp); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
}

func (s *Server) getAssertionHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	if _, err := w.Write([]byte("not implemented")); err != nil {
		log.Error("Could not write response body", "err", err)
	}
}
