package nomadclient

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRoundTripper struct {
	resp        *http.Response
	err         error
	receivedReq *http.Request
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.receivedReq = req
	return f.resp, f.err
}

func newRequest(t *testing.T) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, "http://nomad.example.com/v1/jobs", nil)
	require.NoError(t, err, "building request")

	return req
}

// requestLogLine returns the line in output containing msg="nomad request",
// so assertions about the request line don't accidentally match content from
// the response line logged right after it.
func requestLogLine(t *testing.T, output string) string {
	t.Helper()

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, `msg="nomad request"`) {
			return line
		}
	}

	t.Fatalf("no \"nomad request\" log line found in output: %q", output)
	return ""
}

// responseLogLine returns the line in output containing msg="nomad response".
func responseLogLine(t *testing.T, output string) string {
	t.Helper()

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, `msg="nomad response"`) {
			return line
		}
	}

	t.Fatalf("no \"nomad response\" log line found in output: %q", output)
	return ""
}

// logFieldValue extracts the value of key=value from a single slog text line,
// where value may or may not be quoted.
func logFieldValue(t *testing.T, line, key string) string {
	t.Helper()

	prefix := key + "="
	idx := strings.Index(line, prefix)
	require.GreaterOrEqualf(t, idx, 0, "field %q not found in line %q", key, line)

	rest := line[idx+len(prefix):]
	if strings.HasPrefix(rest, `"`) {
		end := strings.Index(rest[1:], `"`)
		require.GreaterOrEqualf(t, end, 0, "unterminated quoted value for field %q in line %q", key, line)
		return rest[1 : end+1]
	}

	end := strings.IndexByte(rest, ' ')
	if end == -1 {
		end = len(rest)
	}
	return rest[:end]
}

// delayingRoundTripper adds an artificial delay before delegating, so tests
// can prove elapsed-time logging measures real wall-clock time rather than
// just checking that the field is present.
type delayingRoundTripper struct {
	base  http.RoundTripper
	delay time.Duration
}

func (d *delayingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	time.Sleep(d.delay)
	return d.base.RoundTrip(req)
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

	requestLine := requestLogLine(t, output)
	assert.Contains(t, requestLine, "method=GET")
	assert.Contains(t, requestLine, "url=http://nomad.example.com/v1/jobs")
	assert.NotContains(t, requestLine, "bytes=", "GET request has no body, so no byte count should be logged")

	assert.Contains(t, output, `msg="nomad response"`)
	assert.Contains(t, output, "status=200")
	assert.Contains(t, output, "bytes=5")
}

func TestLoggingTransportLogsRequestBodyByteCount(t *testing.T) {
	var logs bytes.Buffer

	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(originalLogger)

	base := &fakeRoundTripper{
		resp: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("")),
		},
	}
	transport := &loggingTransport{base: base}

	req, err := http.NewRequest(http.MethodPost, "http://nomad.example.com/v1/jobs", strings.NewReader("payload"))
	require.NoError(t, err, "building request")

	_, err = transport.RoundTrip(req)
	require.NoError(t, err)

	requestLine := requestLogLine(t, logs.String())
	assert.Contains(t, requestLine, "bytes=7")

	// The underlying transport must still see the full, unconsumed body.
	require.NotNil(t, base.receivedReq.Body)
	gotBody, err := io.ReadAll(base.receivedReq.Body)
	require.NoError(t, err, "reading request body forwarded to the underlying transport")
	assert.Equal(t, "payload", string(gotBody))
}

func TestLoggingTransportLogsElapsedTime(t *testing.T) {
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
	const injectedDelay = 50 * time.Millisecond
	delayed := &delayingRoundTripper{base: base, delay: injectedDelay}
	transport := &loggingTransport{base: delayed}

	_, err := transport.RoundTrip(newRequest(t))
	require.NoError(t, err)

	responseLine := responseLogLine(t, logs.String())
	elapsedStr := logFieldValue(t, responseLine, "elapsed")

	elapsed, err := time.ParseDuration(elapsedStr)
	require.NoErrorf(t, err, "parsing logged elapsed duration %q", elapsedStr)
	assert.GreaterOrEqual(t, elapsed, injectedDelay, "logged elapsed time should reflect the injected delay")
}

func TestLoggingTransportPropagatesTransportError(t *testing.T) {
	base := &fakeRoundTripper{err: io.ErrClosedPipe}
	transport := &loggingTransport{base: base}

	_, err := transport.RoundTrip(newRequest(t))
	assert.ErrorIs(t, err, io.ErrClosedPipe)
}
