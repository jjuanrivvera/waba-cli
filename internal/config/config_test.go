package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempConfig(t *testing.T) *Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	c, err := LoadFrom(path)
	require.NoError(t, err)
	return c
}

func TestConfig_SaveLoadRoundTrip(t *testing.T) {
	c := tempConfig(t)
	require.NoError(t, c.Put(&Account{
		Name: "acme", WABAID: "1020304050", PhoneNumberID: "111222333",
		AppID: "555666", GraphVersion: "v25.0",
	}))
	require.NoError(t, c.Save())

	loaded, err := LoadFrom(c.FilePath())
	require.NoError(t, err)
	assert.Equal(t, "1020304050", loaded.Accounts["acme"].WABAID)
	assert.Equal(t, "111222333", loaded.Accounts["acme"].PhoneNumberID)
	assert.Equal(t, "acme", loaded.CurrentAccount)
}

func TestConfig_SaveIsAtomicAndPrivate(t *testing.T) {
	c := tempConfig(t)
	require.NoError(t, c.Put(&Account{Name: "a", WABAID: "1"}))
	require.NoError(t, c.Save())

	info, err := os.Stat(c.FilePath())
	require.NoError(t, err)
	if info.Mode().Perm() != 0o600 && os.PathSeparator == '/' {
		t.Errorf("config file mode = %v, want 0600", info.Mode().Perm())
	}
	// No temp file may survive the save.
	entries, err := os.ReadDir(filepath.Dir(c.FilePath()))
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".config-", "leftover temp file %s", e.Name())
	}
}

func TestValidateAccountName(t *testing.T) {
	require.NoError(t, ValidateAccountName("prod"))
	require.NoError(t, ValidateAccountName("invitas-mx"))
	for _, bad := range []string{"", "  ", "a/b", `a\b`, "a:b", "..", ".", " pad "} {
		assert.Error(t, ValidateAccountName(bad), "name %q should be rejected", bad)
	}
}

func TestValidateBaseURL(t *testing.T) {
	require.NoError(t, ValidateBaseURL("https://graph.facebook.com"))
	require.NoError(t, ValidateBaseURL("http://localhost:8080"))
	require.NoError(t, ValidateBaseURL("http://127.0.0.1:9999"))
	for _, bad := range []string{"", "ftp://x", "https://", "http://internal.corp"} {
		assert.Error(t, ValidateBaseURL(bad), "url %q should be rejected", bad)
	}
}

func TestValidateGraphVersion(t *testing.T) {
	require.NoError(t, ValidateGraphVersion("v25.0"))
	require.NoError(t, ValidateGraphVersion("v23.0"))
	for _, bad := range []string{"25.0", "v25", "latest", "v25.0.1", ""} {
		assert.Error(t, ValidateGraphVersion(bad), "version %q should be rejected", bad)
	}
}

func TestResolve_ExplicitWinsOverCurrent(t *testing.T) {
	c := tempConfig(t)
	require.NoError(t, c.Put(&Account{Name: "a", WABAID: "1"}))
	require.NoError(t, c.Put(&Account{Name: "b", WABAID: "2"}))
	c.CurrentAccount = "a"

	got, err := c.Resolve("b")
	require.NoError(t, err)
	assert.Equal(t, "2", got.WABAID)
}

func TestResolve_SingleAccountNeedsNoSelection(t *testing.T) {
	c := tempConfig(t)
	require.NoError(t, c.Put(&Account{Name: "only", WABAID: "42"}))
	c.CurrentAccount = ""

	got, err := c.Resolve("")
	require.NoError(t, err)
	assert.Equal(t, "42", got.WABAID)
}

func TestResolve_DefaultsApplied(t *testing.T) {
	c := tempConfig(t)
	require.NoError(t, c.Put(&Account{Name: "a", WABAID: "1"}))

	got, err := c.Resolve("a")
	require.NoError(t, err)
	assert.Equal(t, DefaultBaseURL, got.BaseURL)
	assert.Equal(t, DefaultGraphVersion, got.GraphVersion)
}

func TestResolve_EnvOverrides(t *testing.T) {
	c := tempConfig(t)
	require.NoError(t, c.Put(&Account{Name: "a", WABAID: "stored", PhoneNumberID: "stored-phone"}))
	t.Setenv(EnvPrefix+"WABA_ID", "env-waba")
	t.Setenv(EnvPrefix+"GRAPH_VERSION", "v23.0")

	got, err := c.Resolve("a")
	require.NoError(t, err)
	assert.Equal(t, "env-waba", got.WABAID, "env overrides the stored id")
	assert.Equal(t, "stored-phone", got.PhoneNumberID, "unset env leaves the stored value")
	assert.Equal(t, "v23.0", got.GraphVersion)
	// The stored account must not be mutated by the overlay.
	assert.Equal(t, "stored", c.Accounts["a"].WABAID)
}

func TestResolve_EnvOnlyRun(t *testing.T) {
	c := tempConfig(t)
	t.Setenv(EnvPrefix+"PHONE_NUMBER_ID", "12345")

	got, err := c.Resolve("")
	require.NoError(t, err)
	assert.Equal(t, "env", got.Name)
	assert.Equal(t, "12345", got.PhoneNumberID)
	assert.Equal(t, DefaultBaseURL, got.BaseURL)
}

func TestResolve_NoAccountIsActionableError(t *testing.T) {
	c := tempConfig(t)
	_, err := c.Resolve("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "waba init")
}

func TestResolve_UnknownAccountListsKnown(t *testing.T) {
	c := tempConfig(t)
	require.NoError(t, c.Put(&Account{Name: "a", WABAID: "1"}))
	require.NoError(t, c.Put(&Account{Name: "b", WABAID: "2"}))

	_, err := c.Resolve("nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "a, b")
}

func TestRemove_ClearsCurrentSelection(t *testing.T) {
	c := tempConfig(t)
	require.NoError(t, c.Put(&Account{Name: "a", WABAID: "1"}))
	require.NoError(t, c.Put(&Account{Name: "b", WABAID: "2"}))
	c.CurrentAccount = "a"

	require.True(t, c.Remove("a"))
	assert.Equal(t, "b", c.CurrentAccount, "lone survivor becomes current")
	assert.False(t, c.Remove("a"), "double remove reports absence")
}

func TestAccountNames_Sorted(t *testing.T) {
	c := tempConfig(t)
	for _, n := range []string{"zeta", "alpha", "mid"} {
		require.NoError(t, c.Put(&Account{Name: n, WABAID: "x"}))
	}
	assert.Equal(t, []string{"alpha", "mid", "zeta"}, c.AccountNames())
}

func TestDirAndPath_XDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	dir, err := Dir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(xdg, "waba-cli"), dir)

	path, err := Path()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(xdg, "waba-cli", "config.yaml"), path)
}

func TestFirstNonEmpty(t *testing.T) {
	assert.Equal(t, "flag", FirstNonEmpty("flag", "env", "file"))
	assert.Equal(t, "env", FirstNonEmpty("", "env", "file"))
	assert.Equal(t, "file", FirstNonEmpty("", "  ", "file"))
	assert.Equal(t, "", FirstNonEmpty("", ""))
}
