package nomadclient

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRoundTripper struct {
	resp *http.Response
	err  error
}

func (f *fakeRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return f.resp, f.err
}

func newRequest(t *testing.T) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, "http://nomad.example.com/v1/jobs", nil)
	require.NoError(t, err, "building request")

	return req
}

func TestLoggingTransportPreservesResponseBody(t *testing.T) {
	base := &fakeRoundTripper{
		resp: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		},
	}
	transport := &loggingTransport{base: base}

	resp, err := transport.RoundTrip(newRequest(t))
	require.NoError(t, err)

	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "reading response body")

	assert.Equal(t, `{"ok":true}`, string(got))
}

func TestLoggingTransportLogsRequestAndResponse(t *testing.T) {
	var logs bytes.Buffer

	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(originalLogger)

	base := &fakeRoundTripper{
		resp: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("hello")),
		},
	}
	transport := &loggingTransport{base: base}

	_, err := transport.RoundTrip(newRequest(t))
	require.NoError(t, err)

	output := logs.String()

	assert.Contains(t, output, `msg="nomad request"`)
	assert.Contains(t, output, "method=GET")
	assert.Contains(t, output, "url=http://nomad.example.com/v1/jobs")

	assert.Contains(t, output, `msg="nomad response"`)
	assert.Contains(t, output, "status=200")
	assert.Contains(t, output, "bytes=5")
}

func TestLoggingTransportPropagatesTransportError(t *testing.T) {
	base := &fakeRoundTripper{err: io.ErrClosedPipe}
	transport := &loggingTransport{base: base}

	_, err := transport.RoundTrip(newRequest(t))
	assert.ErrorIs(t, err, io.ErrClosedPipe)
}
