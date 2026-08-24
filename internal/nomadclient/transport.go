package nomadclient

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
)

// loggingTransport wraps an http.RoundTripper to log every HTTP request and
// response made to the Nomad API.
type loggingTransport struct {
	base http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	slog.Info("nomad request", "method", req.Method, "url", req.URL.String())

	resp, err := base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}

	err = resp.Body.Close()
	if err != nil {
		return resp, err
	}

	slog.Info("nomad response",
		"method", req.Method,
		"url", req.URL.String(),
		"status", resp.StatusCode,
		"bytes", len(body),
	)

	resp.Body = io.NopCloser(bytes.NewReader(body))

	return resp, nil
}
