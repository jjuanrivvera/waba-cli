package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The command tests drive the real cobra tree against a mock Graph API, so they cover the
// exact seam a user exercises: flag parsing, payload construction, the client, and the
// renderer. Environment variables configure everything — no config file, no keyring.

// capturedRequest records what the mock server received.
type capturedRequest struct {
	Method string
	Path   string
	Query  map[string]string
	Body   []byte
	Header http.Header
}

// mockGraph is a scripted Graph API: map "METHOD /v25.0/path" to a response.
type mockGraph struct {
	t        *testing.T
	server   *httptest.Server
	handlers map[string]func(w http.ResponseWriter, r *http.Request)
	requests []capturedRequest
}

func newMockGraph(t *testing.T) *mockGraph {
	t.Helper()
	m := &mockGraph{t: t, handlers: map[string]func(http.ResponseWriter, *http.Request){}}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := readAll(r.Body)
		q := map[string]string{}
		for k := range r.URL.Query() {
			q[k] = r.URL.Query().Get(k)
		}
		m.requests = append(m.requests, capturedRequest{
			Method: r.Method, Path: r.URL.Path, Query: q, Body: body, Header: r.Header.Clone(),
		})
		key := r.Method + " " + r.URL.Path
		if h, ok := m.handlers[key]; ok {
			h(w, r)
			return
		}
		// Default: succeed with the universal envelope so happy-path tests need no scripting.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	t.Cleanup(m.server.Close)
	return m
}

// on scripts a JSON response for "METHOD /v25.0/path".
func (m *mockGraph) on(method, path, response string) {
	m.handlers[method+" "+path] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}
}

// onStatus scripts a status-code response.
func (m *mockGraph) onStatus(method, path string, status int, response string) {
	m.handlers[method+" "+path] = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}
}

// last returns the most recent captured request.
func (m *mockGraph) last() capturedRequest {
	require.NotEmpty(m.t, m.requests, "the mock received no requests")
	return m.requests[len(m.requests)-1]
}

// testEnv points every id/credential at the mock. Individual tests override as needed.
func testEnv(t *testing.T, m *mockGraph) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("WABA_ACCESS_TOKEN", "test-token")
	t.Setenv("WABA_BASE_URL", m.server.URL)
	t.Setenv("WABA_PHONE_NUMBER_ID", "111")
	t.Setenv("WABA_WABA_ID", "222")
	t.Setenv("WABA_APP_ID", "333")
	t.Setenv("WABA_ACCOUNT", "")
	t.Setenv("WABA_KEYRING_BACKEND", "file")
	t.Setenv("WABA_KEYRING_PASSWORD", "test-pass")
}

// runCmd executes one CLI invocation on a fresh root, returning stdout, stderr and the error.
func runCmd(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := NewRootCmd()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetIn(strings.NewReader(""))
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return out.String(), errOut.String(), err
}

// jsonBody decodes a captured request body for assertions.
func jsonBody(t *testing.T, req capturedRequest) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(req.Body, &m), "request body is not JSON: %s", req.Body)
	return m
}
