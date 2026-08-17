package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jjuanrivvera/waba-cli/internal/auth"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func TestAuth_LoginStatusLogout(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_ACCESS_TOKEN", "") // force the stored-credential path
	m.on("GET", "/v25.0/me", `{"id":"555","name":"Rivera System User"}`)

	// Scripted login verifies against /me and stores the token.
	_, errOut, err := runCmd(t, "auth", "login", "--token", "tok-abc", "--account", "prod")
	require.NoError(t, err)
	assert.Contains(t, errOut, "Rivera System User")

	cred, err := auth.NewStore().Get("prod")
	require.NoError(t, err)
	assert.Equal(t, "tok-abc", cred.Token)

	// The account was persisted and became current.
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "prod", cfg.CurrentAccount)

	// status reports identity and backend.
	out, _, err := runCmd(t, "auth", "status", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "Rivera System User")
	assert.Contains(t, out, "encrypted-file")
	assert.NotContains(t, out, "tok-abc", "the token itself must never be printed")

	// whoami is the alias.
	_, _, err = runCmd(t, "auth", "whoami")
	require.NoError(t, err)

	// logout removes the credential; a second logout reports absence without failing.
	_, _, err = runCmd(t, "auth", "logout")
	require.NoError(t, err)
	_, err = auth.NewStore().Get("prod")
	assert.ErrorIs(t, err, auth.ErrNotFound)
	_, _, err = runCmd(t, "auth", "logout")
	require.NoError(t, err)
}

func TestAuth_LoginRejectsBadToken(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_ACCESS_TOKEN", "")
	m.onStatus("GET", "/v25.0/me", 401, `{"error":{"message":"bad","code":190}}`)

	_, _, err := runCmd(t, "auth", "login", "--token", "bad-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verification failed")

	_, storeErr := auth.NewStore().Get("default")
	assert.ErrorIs(t, storeErr, auth.ErrNotFound, "a token that fails verification must not be stored")
}

func TestConfig_Commands(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_ACCOUNT", "work")

	out, _, err := runCmd(t, "config", "path")
	require.NoError(t, err)
	assert.Contains(t, out, filepath.Join("waba-cli", "config.yaml"))

	_, _, err = runCmd(t, "config", "set", "waba_id", "999888")
	require.NoError(t, err)
	_, _, err = runCmd(t, "config", "set", "graph_version", "v23.0")
	require.NoError(t, err)
	_, _, err = runCmd(t, "config", "set", "graph_version", "not-a-version")
	require.Error(t, err)
	_, _, err = runCmd(t, "config", "set", "output", "json")
	require.NoError(t, err)
	_, _, err = runCmd(t, "config", "set", "rate_limit", "5")
	require.NoError(t, err)
	_, _, err = runCmd(t, "config", "set", "rate_limit", "fast")
	require.Error(t, err)
	_, _, err = runCmd(t, "config", "set", "bogus_key", "x")
	require.Error(t, err)

	out, _, err = runCmd(t, "config", "view", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "999888")
	assert.Contains(t, out, "v23.0")

	out, _, err = runCmd(t, "config", "list-accounts", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "work")

	_, _, err = runCmd(t, "config", "use", "work")
	require.NoError(t, err)
	_, _, err = runCmd(t, "config", "use", "ghost")
	require.Error(t, err)

	_, _, err = runCmd(t, "config", "remove", "work")
	require.NoError(t, err)
	_, _, err = runCmd(t, "config", "remove", "work")
	require.Error(t, err)
}

func TestInit_NonInteractive(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_ACCESS_TOKEN", "")
	t.Setenv("WABA_WABA_ID", "")
	t.Setenv("WABA_PHONE_NUMBER_ID", "")
	m.on("GET", "/v25.0/me", `{"id":"555","name":"SU"}`)
	m.on("GET", "/v25.0/98765/phone_numbers", `{"data":[{"id":"12321","display_phone_number":"+57 300","verified_name":"Rivera"}]}`)

	_, errOut, err := runCmd(t, "init", "--name", "prod", "--token", "tok-1", "--waba-id", "98765")
	require.NoError(t, err)
	assert.Contains(t, errOut, "12321", "the lone phone number is discovered and adopted")

	cfg, err := config.Load()
	require.NoError(t, err)
	acct := cfg.Accounts["prod"]
	require.NotNil(t, acct)
	assert.Equal(t, "98765", acct.WABAID)
	assert.Equal(t, "12321", acct.PhoneNumberID)

	cred, err := auth.NewStore().Get("prod")
	require.NoError(t, err)
	assert.Equal(t, "tok-1", cred.Token)
}

func TestInit_DetectsPhoneIDPastedAsWABA(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_ACCESS_TOKEN", "")
	t.Setenv("WABA_WABA_ID", "")
	t.Setenv("WABA_PHONE_NUMBER_ID", "")
	m.on("GET", "/v25.0/me", `{"id":"555","name":"SU"}`)
	// Pasting the phone number id at the WABA prompt: the phone_numbers edge 400s, but the
	// node itself is a phone number — init must self-correct instead of saving a broken WABA.
	m.onStatus("GET", "/v25.0/1214/phone_numbers", 400,
		`{"error":{"message":"(#100) Tried accessing nonexisting field (phone_numbers)","code":100}}`)
	m.on("GET", "/v25.0/1214", `{"display_phone_number":"+57 323 4379352","id":"1214"}`)

	_, errOut, err := runCmd(t, "init", "--name", "oops", "--token", "tok-1", "--waba-id", "1214")
	require.NoError(t, err)
	assert.Contains(t, errOut, "phone number id")
	assert.Contains(t, errOut, "config set waba_id")

	cfg, err := config.Load()
	require.NoError(t, err)
	acct := cfg.Accounts["oops"]
	require.NotNil(t, acct)
	assert.Empty(t, acct.WABAID, "the mistaken id must not be saved as the WABA")
	assert.Equal(t, "1214", acct.PhoneNumberID, "the id is adopted as the default phone number")
}

func TestDoctor_ReportsChecks(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("GET", "/v25.0/me", `{"id":"555","name":"SU"}`)
	m.on("GET", "/v25.0/222", `{"id":"222","name":"Rivera WABA"}`)
	m.on("GET", "/v25.0/111", `{"display_phone_number":"+57 300","status":"CONNECTED"}`)

	out, _, err := runCmd(t, "doctor", "--json")
	require.NoError(t, err)
	var report struct {
		Checks []doctorCheck `json:"checks"`
		OK     bool          `json:"ok"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &report))
	assert.True(t, report.OK)
	names := map[string]bool{}
	for _, c := range report.Checks {
		names[c.Name] = c.OK
	}
	for _, want := range []string{"config readable", "account resolvable", "token stored", "graph api reachable", "token valid", "waba accessible", "phone number accessible"} {
		ok, present := names[want]
		assert.Truef(t, present, "doctor must run the %q check", want)
		assert.Truef(t, ok, "check %q should pass against the healthy mock", want)
	}
}

func TestDoctor_FailsNonZeroOnBadToken(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.onStatus("GET", "/v25.0/me", 401, `{"error":{"code":190,"message":"bad"}}`)

	_, _, err := runCmd(t, "doctor")
	require.Error(t, err, "doctor must exit non-zero when a check fails")
}

func TestAPI_RawEscapeHatch(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("GET", "/v25.0/me", `{"id":"555"}`)

	out, _, err := runCmd(t, "api", "GET", "me", "-q", "fields=id", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "555")
	assert.Equal(t, "id", m.last().Query["fields"])

	_, _, err = runCmd(t, "api", "POST", "111/x", "-d", `{"a":1}`, "-H", "X-Extra: 1")
	require.NoError(t, err)
	assert.Equal(t, "1", m.last().Header.Get("X-Extra"))

	_, _, err = runCmd(t, "api", "BREW", "me")
	require.Error(t, err)
	_, _, err = runCmd(t, "api", "GET", "me", "-q", "not-key-value")
	require.Error(t, err)
	_, _, err = runCmd(t, "api", "GET", "me", "-H", "no-colon")
	require.Error(t, err)
}

func TestVersion_Command(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	out, _, err := runCmd(t, "version")
	require.NoError(t, err)
	assert.Contains(t, out, "waba")

	out, _, err = runCmd(t, "version", "--json")
	require.NoError(t, err)
	assert.Contains(t, out, "go_version")
}

func TestAlias_SetExpandList(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	_, _, err := runCmd(t, "alias", "set", "approved", "templates list --status APPROVED")
	require.NoError(t, err)

	out, _, err := runCmd(t, "alias", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "approved")

	// Expansion happens before cobra parses; built-ins always win.
	root := NewRootCmd()
	expanded := ExpandAlias([]string{"approved"}, BuiltinNames(root))
	assert.Equal(t, []string{"templates", "list", "--status", "APPROVED"}, expanded)

	_, _, err = runCmd(t, "alias", "set", "templates", "phone list")
	require.Error(t, err, "an alias must never shadow a built-in")

	_, _, err = runCmd(t, "alias", "remove", "approved")
	require.NoError(t, err)
}

func TestOutputFormats_AllFour(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("GET", "/v25.0/222/phone_numbers", `{"data":[{"id":"111","display_phone_number":"+57 300","verified_name":"Rivera"}]}`)

	out, _, err := runCmd(t, "phone", "list", "-o", "table")
	require.NoError(t, err)
	assert.Contains(t, out, "Rivera")

	out, _, err = runCmd(t, "phone", "list", "-o", "json")
	require.NoError(t, err)
	var rows []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &rows))

	out, _, err = runCmd(t, "phone", "list", "-o", "yaml")
	require.NoError(t, err)
	assert.Contains(t, out, "display_phone_number:")

	out, _, err = runCmd(t, "phone", "list", "-o", "csv")
	require.NoError(t, err)
	assert.Contains(t, out, ",")

	out, _, err = runCmd(t, "phone", "list", "-o", "id")
	require.NoError(t, err)
	assert.Equal(t, "111", strings.TrimSpace(out))

	_, _, err = runCmd(t, "phone", "list", "-o", "hologram")
	require.Error(t, err)
}

func TestJQFilter(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("GET", "/v25.0/222/phone_numbers", `{"data":[{"id":"111","verified_name":"Rivera"},{"id":"112","verified_name":"Otro"}]}`)

	out, _, err := runCmd(t, "phone", "list", "--jq", ".[].verified_name", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "Rivera")
	assert.Contains(t, out, "Otro")
	assert.NotContains(t, out, "111")

	_, _, err = runCmd(t, "phone", "list", "--jq", "((broken")
	require.Error(t, err)
}

func TestAccountFlagSelectsProfile(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_WABA_ID", "")

	// Two stored accounts with different WABA ids; --account picks which one list uses.
	cfg, err := config.Load()
	require.NoError(t, err)
	require.NoError(t, cfg.Put(&config.Account{Name: "a", WABAID: "1000", BaseURL: m.server.URL}))
	require.NoError(t, cfg.Put(&config.Account{Name: "b", WABAID: "2000", BaseURL: m.server.URL}))
	require.NoError(t, cfg.Save())

	m.on("GET", "/v25.0/2000/phone_numbers", `{"data":[]}`)
	_, _, err = runCmd(t, "phone", "list", "--account", "b")
	require.NoError(t, err)
	assert.Equal(t, "/v25.0/2000/phone_numbers", m.last().Path)

	// The hidden --profile alias reaches the same selector.
	m.on("GET", "/v25.0/1000/phone_numbers", `{"data":[]}`)
	_, _, err = runCmd(t, "phone", "list", "--profile", "a")
	require.NoError(t, err)
	assert.Equal(t, "/v25.0/1000/phone_numbers", m.last().Path)
}

func TestGlobalIDOverrideFlags(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("GET", "/v25.0/777/phone_numbers", `{"data":[]}`)
	_, _, err := runCmd(t, "phone", "list", "--waba-id", "777")
	require.NoError(t, err)
	assert.Equal(t, "/v25.0/777/phone_numbers", m.last().Path)

	m.on("POST", "/v25.0/888/messages", sendOK)
	_, _, err = runCmd(t, "send", "text", "--to", "1", "hola", "--phone-id", "888")
	require.NoError(t, err)
	assert.Equal(t, "/v25.0/888/messages", m.last().Path)
}

func TestQuietSuppressesNotes(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, errOut, err := runCmd(t, "send", "text", "--to", "1", "hola", "--quiet")
	require.NoError(t, err)
	assert.NotContains(t, errOut, "note:")

	_, errOut, err = runCmd(t, "send", "text", "--to", "1", "hola")
	require.NoError(t, err)
	assert.Contains(t, errOut, "note:")
}

func TestCompletionGeneratesScripts(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		root := NewRootCmd()
		var buf strings.Builder
		root.SetOut(&buf)
		root.SetErr(&strings.Builder{})
		root.SetArgs([]string{"completion", sh})
		done := make(chan error, 1)
		go func() { done <- root.Execute() }()
		require.NoError(t, <-done, "shell %s", sh)
		assert.NotEmpty(t, buf.String(), "shell %s", sh)
	}
}

func TestConfigDirPermissions(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("unix permission semantics")
	}
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_ACCOUNT", "p")

	_, _, err := runCmd(t, "config", "set", "waba_id", "1")
	require.NoError(t, err)
	dir, err := config.Dir()
	require.NoError(t, err)
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	finfo, err := os.Stat(filepath.Join(dir, "config.yaml"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), finfo.Mode().Perm())
}
