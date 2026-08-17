// Package auth stores WhatsApp Cloud API credentials and applies them to outgoing requests.
//
// Secrets go to the OS keyring (macOS Keychain, Linux Secret Service, Windows Credential
// Manager). Headless Linux frequently has no Secret Service at all, so there is an
// AES-256-GCM encrypted-file fallback keyed by WABA_KEYRING_PASSWORD — without it, the
// CLI would be unusable in exactly the CI and container environments an agent runs in.
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/zalando/go-keyring"
)

// KeyringService namespaces this CLI's entries in the OS keyring.
const KeyringService = "waba-cli"

// KeyringPasswordEnv unlocks the encrypted-file fallback.
const KeyringPasswordEnv = "WABA_KEYRING_PASSWORD"

// KeyringPasswordFileEnv names a file to read the password from, for the common case where
// exporting it into the shell does not reach the CLI: a non-interactive `ssh host 'waba
// …'`, cron, or any bash invocation, none of which source .bashrc/.zshrc. Absent both env
// vars, a `keyring-password` file in the config dir is read if present. A file is also
// simply safer — the secret is not in the environment of every child process.
const KeyringPasswordFileEnv = "WABA_KEYRING_PASSWORD_FILE"

// keyringPasswordFile is the default password-file name, read when neither env var is set.
const keyringPasswordFile = "keyring-password"

// keyringPassword resolves the encrypted-file password from, in order: the env var, a file
// named by the file env var, or the default file in the config dir. `fromFile` reports
// whether it came from a file rather than the env var, which is what lets a password file be
// a persistent opt-in to the file store (see NewStore). Returning "" means none was
// configured — the caller decides whether that is fatal.
//
// The file must not be readable by anyone but its owner: a password the whole machine can
// read is not protecting the credential it encrypts, so a loose-permissions file is refused
// rather than used, and the refusal is surfaced (not swallowed) so it cannot look like
// "no password configured".
func keyringPassword() (pw string, fromFile bool, err error) {
	if pw := os.Getenv(KeyringPasswordEnv); pw != "" {
		return pw, false, nil
	}
	path := os.Getenv(KeyringPasswordFileEnv)
	explicit := path != ""
	if path == "" {
		dir, err := configDir()
		if err != nil {
			return "", false, nil // no config dir → treat as "no password", not an error
		}
		path = filepath.Join(dir, keyringPasswordFile)
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return "", false, nil // default file simply absent
		}
		return "", false, fmt.Errorf("read %s: %w", KeyringPasswordFileEnv, err)
	}
	// Windows has no POSIX permission bits — Perm() reports 0666 for any writable file, so
	// this check would refuse every password file there. NTFS ACLs are the protection on
	// Windows; the strict-mode check is meaningful only on Unix.
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return "", false, fmt.Errorf("%s is readable by others (%#o); run: chmod 600 %s",
			path, info.Mode().Perm(), path)
	}
	raw, err := os.ReadFile(path) // #nosec G304 -- path is the user's own configured password file
	if err != nil {
		return "", false, fmt.Errorf("read keyring password file: %w", err)
	}
	return strings.TrimRight(string(raw), "\r\n"), true, nil
}

// KeyringBackendEnv forces a specific credential store: "file" for the encrypted file,
// "keyring" for the OS keyring. Unset means "OS keyring, falling back to the file".
//
// Forcing it matters in two real cases: a machine where the OS keyring exists but you want
// portable, copyable credentials, and a test suite, which must never write into the
// developer's actual Keychain or Secret Service.
const KeyringBackendEnv = "WABA_KEYRING_BACKEND"

// Credential is everything secret for one account.
type Credential struct {
	// Token is the Graph API access token — in practice a System User token generated in
	// Meta Business Manager (long-lived) or a temporary token from the App Dashboard.
	Token string `json:"token,omitempty"`
}

// Empty reports whether there is nothing worth storing.
func (c Credential) Empty() bool { return c.Token == "" }

// Store persists credentials per account.
type Store interface {
	Get(account string) (Credential, error)
	Set(account string, c Credential) error
	Delete(account string) error
	// Backend names the storage in use, for `auth status` and doctor output.
	Backend() string
}

// ErrNotFound means no credential is stored for that account.
var ErrNotFound = errors.New("no stored credential")

// NewStore selects the credential store. Precedence: an explicit WABA_KEYRING_BACKEND;
// then a keyring-password *file*, which forces the file store; then the OS keyring, with the
// encrypted file as a fallback only when the keyring is unavailable.
func NewStore() Store {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(KeyringBackendEnv))) {
	case "file":
		if fs, err := NewFileStore(""); err == nil {
			return fs
		}
		// Falling through to the OS keyring here would silently ignore an explicit request;
		// the file store only fails when the password is unset, which the error names.
	case "keyring":
		return &keyringStore{}
	}

	pw, fromFile, _ := keyringPassword()

	// A password *file* is a persistent, shell-independent opt-in to the file store, so honour
	// it even where the OS keyring is usable. This is the fix for the read/write split: a
	// credential written to the file store on a box that also has a keyring must be read back
	// from the file store, not from an empty keyring — and the choice cannot ride on a shell
	// variable, because a non-interactive `ssh host 'waba …'` sources no rc to set one.
	if fromFile {
		if fs, err := NewFileStore(""); err == nil {
			return fs
		}
	}

	// Env-var-only password: keep the narrower rule — fall back to the file only when the OS
	// keyring is genuinely unavailable (headless Linux, no Secret Service).
	if pw != "" && !keyringUsable() {
		if fs, err := NewFileStore(""); err == nil {
			return fs
		}
	}
	return &keyringStore{}
}

// keyringUsable probes the OS keyring once. A headless Linux box without a Secret Service
// daemon returns an error here, which is the signal to use the file fallback.
func keyringUsable() bool {
	const probe = "__probe__"
	err := keyring.Set(KeyringService, probe, "1")
	if err != nil {
		return false
	}
	_ = keyring.Delete(KeyringService, probe)
	return true
}

type keyringStore struct{}

func (k *keyringStore) Backend() string { return "os-keyring" }

func keyFor(account string) string { return "account-" + account }

func (k *keyringStore) Get(account string) (Credential, error) {
	raw, err := keyring.Get(KeyringService, keyFor(account))
	if errors.Is(err, keyring.ErrNotFound) {
		return Credential{}, ErrNotFound
	}
	if err != nil {
		return Credential{}, fmt.Errorf("read keyring: %w", err)
	}
	return decodeCredential(raw)
}

func (k *keyringStore) Set(account string, c Credential) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := keyring.Set(KeyringService, keyFor(account), string(raw)); err != nil {
		return fmt.Errorf("write keyring: %w (on headless Linux set %s to use the encrypted-file fallback)", err, KeyringPasswordEnv)
	}
	return nil
}

func (k *keyringStore) Delete(account string) error {
	err := keyring.Delete(KeyringService, keyFor(account))
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// decodeCredential accepts both the JSON form and a bare token string, so credentials written
// by an older version (or by hand) still load.
func decodeCredential(raw string) (Credential, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Credential{}, ErrNotFound
	}
	if strings.HasPrefix(raw, "{") {
		var c Credential
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			return Credential{}, fmt.Errorf("decode credential: %w", err)
		}
		return c, nil
	}
	return Credential{Token: raw}, nil
}

// ---------- encrypted file fallback ----------

type fileStore struct {
	path     string
	password string
}

// NewFileStore creates the encrypted-file store. An empty path uses credentials.enc in the
// config directory.
func NewFileStore(path string) (Store, error) {
	pw, _, err := keyringPassword()
	if err != nil {
		return nil, err
	}
	if pw == "" {
		return nil, fmt.Errorf("set %s, or write the password to %s in the config dir, to use the encrypted-file credential store",
			KeyringPasswordEnv, keyringPasswordFile)
	}
	if path == "" {
		dir, err := configDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(dir, "credentials.enc")
	}
	return &fileStore{path: path, password: pw}, nil
}

func (f *fileStore) Backend() string { return "encrypted-file" }

func (f *fileStore) Get(account string) (Credential, error) {
	all, err := f.load()
	if err != nil {
		return Credential{}, err
	}
	c, ok := all[account]
	if !ok {
		return Credential{}, ErrNotFound
	}
	return c, nil
}

func (f *fileStore) Set(account string, c Credential) error {
	all, err := f.load()
	if err != nil {
		return err
	}
	all[account] = c
	return f.save(all)
}

func (f *fileStore) Delete(account string) error {
	all, err := f.load()
	if err != nil {
		return err
	}
	if _, ok := all[account]; !ok {
		return ErrNotFound
	}
	delete(all, account)
	return f.save(all)
}

// fileEnvelope is the on-disk format: a random per-file salt plus the sealed payload. The
// salt is stored in the clear (that is its purpose) so the same password yields a different
// key in every file.
type fileEnvelope struct {
	Version int    `json:"v"`
	Salt    string `json:"salt"`
	Nonce   string `json:"nonce"`
	Data    string `json:"data"`
}

const pbkdf2Iterations = 600_000 // OWASP guidance for PBKDF2-HMAC-SHA256

func (f *fileStore) load() (map[string]Credential, error) {
	raw, err := os.ReadFile(f.path) // #nosec G304 -- the path is this store's own file
	if errors.Is(err, os.ErrNotExist) {
		return map[string]Credential{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", f.path, err)
	}

	var env fileEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("parse %s: %w", f.path, err)
	}
	salt, err := base64.StdEncoding.DecodeString(env.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(env.Data)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	gcm, err := f.cipher(salt)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		// GCM authentication failing means the wrong password or a tampered file; both are
		// worth saying out loud rather than surfacing as a parse error.
		return nil, fmt.Errorf("decrypt %s: wrong %s, or the file was modified", f.path, KeyringPasswordEnv)
	}

	out := map[string]Credential{}
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, fmt.Errorf("decode credentials: %w", err)
	}
	return out, nil
}

func (f *fileStore) save(all map[string]Credential) error {
	plain, err := json.Marshal(all)
	if err != nil {
		return err
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}
	gcm, err := f.cipher(salt)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, plain, nil)

	env := fileEnvelope{
		Version: 1,
		Salt:    base64.StdEncoding.EncodeToString(salt),
		Nonce:   base64.StdEncoding.EncodeToString(nonce),
		Data:    base64.StdEncoding.EncodeToString(sealed),
	}
	out, err := json.Marshal(env)
	if err != nil {
		return err
	}

	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	// Same atomic temp-then-rename as the config, for the same reason.
	tmp, err := os.CreateTemp(dir, ".credentials-*.enc")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, f.path)
}

func (f *fileStore) cipher(salt []byte) (cipher.AEAD, error) {
	key, err := pbkdf2.Key(sha256.New, f.password, salt, pbkdf2Iterations, 32)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func configDir() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "waba-cli"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".waba-cli"), nil
}
