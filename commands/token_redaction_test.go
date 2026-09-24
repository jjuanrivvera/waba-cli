package commands

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The token is a System User credential: one leak into a log, a screenshot or an agent's
// transcript is a live key in someone else's hands. `auth status` and `--dry-run` were already
// covered; these are the paths that were not — the config dump and the error paths, which are
// exactly what a user pastes into a bug report (issue #5).
const secretToken = "EAAtestSECRETtokenVALUE0123456789"

func TestTokenNeverPrinted_ConfigView(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_ACCESS_TOKEN", "")
	m.on("GET", "/v25.0/me", `{"id":"555","name":"Rivera System User"}`)

	_, _, err := runCmd(t, "auth", "login", "--token", secretToken, "--account", "prod")
	require.NoError(t, err)

	for _, format := range []string{"table", "json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			out, errOut, err := runCmd(t, "config", "view", "-o", format)
			require.NoError(t, err)
			assert.NotContains(t, out, secretToken, "the stored token must not be dumped by config view")
			assert.NotContains(t, errOut, secretToken)
		})
	}
}

// A failing request is the moment the output gets copied into an issue, so it is the worst
// possible place for the credential to appear — in the message, the hint or the trace id line.
func TestTokenNeverPrinted_ErrorPaths(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_ACCESS_TOKEN", secretToken)

	m.onFunc("GET", "/v25.0/222/phone_numbers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"(#100) Tried accessing nonexisting field (phone_numbers)",` +
			`"type":"OAuthException","code":100,"fbtrace_id":"AbC123"}}`))
	})

	out, errOut, err := runCmd(t, "phone", "list")
	require.Error(t, err)
	for name, text := range map[string]string{"stdout": out, "stderr": errOut, "error": err.Error()} {
		assert.NotContains(t, text, secretToken, "the token reached %s", name)
	}
	// The failure itself must still be readable, or redaction has cost the user the diagnosis.
	assert.True(t, strings.Contains(err.Error(), "nonexisting field") || strings.Contains(errOut, "nonexisting field"),
		"the Graph message should survive:\n%s\n%s", errOut, err)
}

// A transport failure prints Go's own error, which carries the URL. The token rides in a
// header here rather than the path, and this pins that: the day someone moves it into a query
// parameter "for convenience", this test is what says no.
func TestTokenNeverPrinted_TransportFailure(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_ACCESS_TOKEN", secretToken)

	m.onFunc("GET", "/v25.0/222/phone_numbers", func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler) // drop the connection, as a dead link would
	})

	out, errOut, err := runCmd(t, "phone", "list")
	require.Error(t, err)
	for name, text := range map[string]string{"stdout": out, "stderr": errOut, "error": err.Error()} {
		assert.NotContains(t, text, secretToken, "the token reached %s", name)
	}
}
