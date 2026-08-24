package nomadclient

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"
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

	requestArgs := []any{"method", req.Method, "url", req.URL.String()}

	if req.Body != nil {
		reqBody, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		err = req.Body.Close()
		if err != nil {
			return nil, err
		}

		req.Body = io.NopCloser(bytes.NewReader(reqBody))
		requestArgs = append(requestArgs, "bytes", len(reqBody))
	}

	slog.Info("nomad request", requestArgs...)

	start := time.Now()

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
		slog.Duration("elapsed", time.Since(start)),
	)

	resp.Body = io.NopCloser(bytes.NewReader(body))

	return resp, nil
}
