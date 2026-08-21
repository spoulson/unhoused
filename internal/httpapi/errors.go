package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	nomadapi "github.com/hashicorp/nomad/api"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		log.Printf("httpapi: failed to write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorEnvelope{Error: errorDetail{Message: message}})
}

// classifyNomadErr maps an error returned from the Nomad SDK to the HTTP
// status and message unhoused should respond with, per specs/api.md:
// 404 for an unknown profile/job/allocation, 502 for any other Nomad API
// failure (unreachable, timeout, unexpected status).
func classifyNomadErr(err error, notFoundMessage string) (int, string) {
	var uerr nomadapi.UnexpectedResponseError

	ok := errors.As(err, &uerr)
	if ok && uerr.HasStatusCode() && uerr.StatusCode() == http.StatusNotFound {
		return http.StatusNotFound, notFoundMessage
	}

	return http.StatusBadGateway, "Nomad API error: " + err.Error()
}
