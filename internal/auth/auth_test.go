package auth

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fileStoreAt builds a file store rooted in a temp dir with a known password, without
// touching the developer's real keyring or config.
func fileStoreAt(t *testing.T) Store {
	t.Helper()
	t.Setenv(KeyringPasswordEnv, "test-password")
	s, err := NewFileStore(filepath.Join(t.TempDir(), "credentials.enc"))
	require.NoError(t, err)
	return s
}

func TestBearer_Apply(t *testing.T) {
	req := httptest.NewRequest("GET", "https://graph.facebook.com/v25.0/me", nil)
	(&Bearer{Token: "tok-123"}).Apply(req)
	assert.Equal(t, "Bearer tok-123", req.Header.Get("Authorization"))
}

func TestResolveToken_EnvWins(t *testing.T) {
	s := fileStoreAt(t)
	require.NoError(t, s.Set("acc", Credential{Token: "stored"}))
	t.Setenv(TokenEnv, "from-env")

	got, err := ResolveToken(s, "acc")
	require.NoError(t, err)
	assert.Equal(t, "from-env", got)
}

func TestResolveToken_StoreFallback(t *testing.T) {
	s := fileStoreAt(t)
	require.NoError(t, s.Set("acc", Credential{Token: "stored"}))
	t.Setenv(TokenEnv, "")

	got, err := ResolveToken(s, "acc")
	require.NoError(t, err)
	assert.Equal(t, "stored", got)
}

func TestResolveToken_MissingIsActionable(t *testing.T) {
	s := fileStoreAt(t)
	t.Setenv(TokenEnv, "")

	_, err := ResolveToken(s, "ghost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "waba auth login")
	assert.Contains(t, err.Error(), TokenEnv)
}

func TestFileStore_RoundTrip(t *testing.T) {
	s := fileStoreAt(t)
	require.NoError(t, s.Set("prod", Credential{Token: "secret-1"}))
	require.NoError(t, s.Set("staging", Credential{Token: "secret-2"}))

	got, err := s.Get("prod")
	require.NoError(t, err)
	assert.Equal(t, "secret-1", got.Token)

	require.NoError(t, s.Delete("prod"))
	_, err = s.Get("prod")
	assert.ErrorIs(t, err, ErrNotFound)

	// The other credential must survive its sibling's deletion.
	got, err = s.Get("staging")
	require.NoError(t, err)
	assert.Equal(t, "secret-2", got.Token)
}

func TestFileStore_CiphertextOnDisk(t *testing.T) {
	t.Setenv(KeyringPasswordEnv, "test-password")
	path := filepath.Join(t.TempDir(), "credentials.enc")
	s, err := NewFileStore(path)
	require.NoError(t, err)
	require.NoError(t, s.Set("a", Credential{Token: "hunter2-very-secret"}))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "hunter2-very-secret", "token must never appear in plaintext on disk")
}

func TestFileStore_WrongPasswordFailsLoudly(t *testing.T) {
	t.Setenv(KeyringPasswordEnv, "right-password")
	path := filepath.Join(t.TempDir(), "credentials.enc")
	s, err := NewFileStore(path)
	require.NoError(t, err)
	require.NoError(t, s.Set("a", Credential{Token: "x"}))

	t.Setenv(KeyringPasswordEnv, "wrong-password")
	s2, err := NewFileStore(path)
	require.NoError(t, err)
	_, err = s2.Get("a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), KeyringPasswordEnv)
}

func TestFileStore_RequiresPassword(t *testing.T) {
	t.Setenv(KeyringPasswordEnv, "")
	t.Setenv(KeyringPasswordFileEnv, "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // no default password file either
	_, err := NewFileStore(filepath.Join(t.TempDir(), "c.enc"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), KeyringPasswordEnv)
}

func TestKeyringPassword_FileSource(t *testing.T) {
	dir := t.TempDir()
	pwFile := filepath.Join(dir, "pw")
	require.NoError(t, os.WriteFile(pwFile, []byte("file-password\n"), 0o600))
	t.Setenv(KeyringPasswordEnv, "")
	t.Setenv(KeyringPasswordFileEnv, pwFile)

	pw, fromFile, err := keyringPassword()
	require.NoError(t, err)
	assert.True(t, fromFile)
	assert.Equal(t, "file-password", pw, "trailing newline is trimmed")
}

func TestKeyringPassword_RefusesLoosePermissions(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("unix permission semantics")
	}
	dir := t.TempDir()
	pwFile := filepath.Join(dir, "pw")
	require.NoError(t, os.WriteFile(pwFile, []byte("p"), 0o644))
	t.Setenv(KeyringPasswordEnv, "")
	t.Setenv(KeyringPasswordFileEnv, pwFile)

	_, _, err := keyringPassword()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chmod 600")
}

func TestKeyringPassword_DefaultFileInConfigDir(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv(KeyringPasswordEnv, "")
	t.Setenv(KeyringPasswordFileEnv, "")
	require.NoError(t, os.MkdirAll(filepath.Join(cfg, "waba-cli"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(cfg, "waba-cli", "keyring-password"),
		[]byte("default-file-pw"), 0o600))

	pw, fromFile, err := keyringPassword()
	require.NoError(t, err)
	assert.True(t, fromFile)
	assert.Equal(t, "default-file-pw", pw)
}

func TestNewStore_PasswordFileForcesFileStore(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv(KeyringPasswordEnv, "")
	t.Setenv(KeyringPasswordFileEnv, "")
	t.Setenv(KeyringBackendEnv, "")
	require.NoError(t, os.MkdirAll(filepath.Join(cfg, "waba-cli"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(cfg, "waba-cli", "keyring-password"),
		[]byte("pw"), 0o600))

	s := NewStore()
	assert.Equal(t, "encrypted-file", s.Backend(),
		"a password FILE is a persistent opt-in to the file store even where a keyring exists")
}

func TestNewStore_ExplicitBackendFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(KeyringPasswordEnv, "pw")
	t.Setenv(KeyringBackendEnv, "file")
	assert.Equal(t, "encrypted-file", NewStore().Backend())
}

func TestNewStore_ExplicitBackendKeyring(t *testing.T) {
	t.Setenv(KeyringBackendEnv, "keyring")
	assert.Equal(t, "os-keyring", NewStore().Backend())
}

func TestDecodeCredential_LegacyBareToken(t *testing.T) {
	c, err := decodeCredential("bare-token-string")
	require.NoError(t, err)
	assert.Equal(t, "bare-token-string", c.Token)

	c, err = decodeCredential(`{"token":"json-token"}`)
	require.NoError(t, err)
	assert.Equal(t, "json-token", c.Token)

	_, err = decodeCredential("   ")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCredential_Empty(t *testing.T) {
	assert.True(t, Credential{}.Empty())
	assert.False(t, Credential{Token: "x"}.Empty())
}
