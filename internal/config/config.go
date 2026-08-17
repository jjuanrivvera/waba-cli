// Package config stores non-secret settings: which WhatsApp Business accounts are known,
// their default IDs (WABA, phone number, app), and the Graph API version each targets.
// Credentials never live here — they go to the OS keyring (see internal/auth).
//
// Precedence is resolved manually per field (flag > env > file > default) rather than through
// a config framework, so the rules are visible in one place and a wrong lookup is a readable
// bug rather than framework behaviour.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultBaseURL is the Graph API host every WhatsApp Cloud API call goes through. Unlike a
// self-hosted product there is exactly one production endpoint; the override exists for
// mock servers and tests, not for alternative deployments.
const DefaultBaseURL = "https://graph.facebook.com"

// DefaultGraphVersion tracks the Graph API version the Meta reference documents its examples
// with. Bump deliberately: message payload validation differs across versions.
const DefaultGraphVersion = "v25.0"

// Account is one WhatsApp Business Account profile: the IDs a command needs so the user
// doesn't have to repeat them per invocation. The access token itself is in the keyring.
type Account struct {
	Name string `yaml:"name"`

	// WABAID is the WhatsApp Business Account id — the node template, analytics and phone
	// number listing operations hang off.
	WABAID string `yaml:"waba_id,omitempty"`

	// PhoneNumberID is the default business phone number id used to send messages. It is an
	// id from GET /{waba-id}/phone_numbers, not the display number.
	PhoneNumberID string `yaml:"phone_number_id,omitempty"`

	// AppID is the Meta app id, needed only by the resumable upload API (template header
	// media) — POST /{app-id}/uploads.
	AppID string `yaml:"app_id,omitempty"`

	// BusinessID is the Meta business portfolio id, needed only to list the WABAs the
	// portfolio owns or has been shared.
	BusinessID string `yaml:"business_id,omitempty"`

	// GraphVersion pins the Graph API version for this account (e.g. "v25.0").
	GraphVersion string `yaml:"graph_version,omitempty"`

	// BaseURL overrides the Graph host — mock servers and tests only.
	BaseURL string `yaml:"base_url,omitempty"`
}

// Config is the whole file.
type Config struct {
	CurrentAccount string              `yaml:"current_account,omitempty"`
	Accounts       map[string]*Account `yaml:"accounts,omitempty"`

	// Defaults applied when a flag and env var are both absent.
	Output    string  `yaml:"output,omitempty"`
	RateLimit float64 `yaml:"rate_limit,omitempty"`

	path string `yaml:"-"`
}

// EnvPrefix namespaces every environment override.
const EnvPrefix = "WABA_"

// Dir returns the configuration directory, honouring XDG_CONFIG_HOME.
func Dir() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "waba-cli"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".waba-cli"), nil
}

// Path returns the config file path.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the config, returning an empty one when the file does not exist yet — a missing
// config is the first-run state, not an error.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads a config from an explicit path (used by tests).
func LoadFrom(path string) (*Config, error) {
	c := &Config{Accounts: map[string]*Account{}, path: path}

	raw, err := os.ReadFile(path) // #nosec G304 -- the path is the user's own config location
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(raw, c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Accounts == nil {
		c.Accounts = map[string]*Account{}
	}
	// The name is the map key; mirroring it into the value keeps Account self-describing
	// when one is passed around on its own.
	for name, a := range c.Accounts {
		if a != nil {
			a.Name = name
		}
	}
	c.path = path
	return c, nil
}

// Save writes the config atomically with restrictive permissions.
//
// Atomicity matters because a crash mid-write would otherwise leave a truncated YAML file
// and lock the user out of every configured account.
func (c *Config) Save() error {
	if c.path == "" {
		p, err := Path()
		if err != nil {
			return err
		}
		c.path = p
	}
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	out, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	// Temp file in the SAME directory, so the rename is atomic (a cross-device rename is not).
	tmp, err := os.CreateTemp(dir, ".config-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op once the rename succeeds

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpName, c.path); err != nil {
		return fmt.Errorf("install config: %w", err)
	}
	return nil
}

// SetPath overrides the file location (tests).
func (c *Config) SetPath(p string) { c.path = p }

// FilePath returns where this config reads and writes.
func (c *Config) FilePath() string { return c.path }

// AccountNames returns the configured account names, sorted.
func (c *Config) AccountNames() []string {
	out := make([]string, 0, len(c.Accounts))
	for n := range c.Accounts {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Resolve returns the account to use, applying precedence: explicit name > WABA_ACCOUNT >
// current_account > the only configured account.
//
// Falling back to a lone configured account means a single-WABA user never has to think
// about profiles at all.
func (c *Config) Resolve(explicit string) (*Account, error) {
	name := firstNonEmpty(explicit, os.Getenv(EnvPrefix+"ACCOUNT"), c.CurrentAccount)
	if name == "" && len(c.Accounts) == 1 {
		for _, a := range c.Accounts {
			return c.withEnvOverrides(a), nil
		}
	}
	if name == "" {
		// A fully env-configured run needs no file at all — support CI with no config.
		if a := c.accountFromEnv(); a != nil {
			return a, nil
		}
		return nil, errors.New("no account selected — run `waba init`, or set --account/WABA_ACCOUNT")
	}
	a, ok := c.Accounts[name]
	if !ok || a == nil {
		if env := c.accountFromEnv(); env != nil && explicit == "" {
			return env, nil
		}
		available := strings.Join(c.AccountNames(), ", ")
		if available == "" {
			available = "none configured"
		}
		return nil, fmt.Errorf("unknown account %q (known: %s) — add it with `waba init`", name, available)
	}
	return c.withEnvOverrides(a), nil
}

// accountFromEnv builds an ephemeral account purely from environment variables, so a
// container or CI job can run without ever writing a config file. Any one of the IDs is
// enough — a token-only run can still call `waba api` or id-flagged commands.
func (c *Config) accountFromEnv() *Account {
	a := &Account{
		Name:          "env",
		WABAID:        os.Getenv(EnvPrefix + "WABA_ID"),
		PhoneNumberID: os.Getenv(EnvPrefix + "PHONE_NUMBER_ID"),
		AppID:         os.Getenv(EnvPrefix + "APP_ID"),
		BusinessID:    os.Getenv(EnvPrefix + "BUSINESS_ID"),
		GraphVersion:  os.Getenv(EnvPrefix + "GRAPH_VERSION"),
		BaseURL:       os.Getenv(EnvPrefix + "BASE_URL"),
	}
	if a.WABAID == "" && a.PhoneNumberID == "" && a.BaseURL == "" && os.Getenv(EnvPrefix+"ACCESS_TOKEN") == "" {
		return nil
	}
	c.applyDefaults(a)
	return a
}

// withEnvOverrides layers environment variables over a stored account without mutating it.
func (c *Config) withEnvOverrides(a *Account) *Account {
	clone := *a
	if v := os.Getenv(EnvPrefix + "WABA_ID"); v != "" {
		clone.WABAID = v
	}
	if v := os.Getenv(EnvPrefix + "PHONE_NUMBER_ID"); v != "" {
		clone.PhoneNumberID = v
	}
	if v := os.Getenv(EnvPrefix + "APP_ID"); v != "" {
		clone.AppID = v
	}
	if v := os.Getenv(EnvPrefix + "BUSINESS_ID"); v != "" {
		clone.BusinessID = v
	}
	if v := os.Getenv(EnvPrefix + "GRAPH_VERSION"); v != "" {
		clone.GraphVersion = v
	}
	if v := os.Getenv(EnvPrefix + "BASE_URL"); v != "" {
		clone.BaseURL = v
	}
	c.applyDefaults(&clone)
	return &clone
}

func (c *Config) applyDefaults(a *Account) {
	if a.GraphVersion == "" {
		a.GraphVersion = DefaultGraphVersion
	}
	if a.BaseURL == "" {
		a.BaseURL = DefaultBaseURL
	}
}

// NewAccount builds an account with env-aware defaults, so a mock WABA_BASE_URL or a pinned
// WABA_GRAPH_VERSION reaches first-run flows (init, auth login) too — those construct an
// account before any config exists and would otherwise silently target production.
func NewAccount(name string) *Account {
	return &Account{
		Name:         name,
		BaseURL:      firstNonEmpty(os.Getenv(EnvPrefix+"BASE_URL"), DefaultBaseURL),
		GraphVersion: firstNonEmpty(os.Getenv(EnvPrefix+"GRAPH_VERSION"), DefaultGraphVersion),
	}
}

// Put adds or replaces an account.
func (c *Config) Put(a *Account) error {
	if err := ValidateAccountName(a.Name); err != nil {
		return err
	}
	if a.BaseURL != "" {
		if err := ValidateBaseURL(a.BaseURL); err != nil {
			return err
		}
	}
	if a.GraphVersion != "" {
		if err := ValidateGraphVersion(a.GraphVersion); err != nil {
			return err
		}
	}
	if c.Accounts == nil {
		c.Accounts = map[string]*Account{}
	}
	c.Accounts[a.Name] = a
	if c.CurrentAccount == "" {
		c.CurrentAccount = a.Name
	}
	return nil
}

// Remove deletes an account, clearing the current selection if it pointed there.
func (c *Config) Remove(name string) bool {
	if _, ok := c.Accounts[name]; !ok {
		return false
	}
	delete(c.Accounts, name)
	if c.CurrentAccount == name {
		c.CurrentAccount = ""
		if len(c.Accounts) == 1 {
			for n := range c.Accounts {
				c.CurrentAccount = n
			}
		}
	}
	return true
}

// ValidateAccountName rejects names that would escape the config or the keyring namespace.
// Account names become part of a keyring key, so a name containing a path separator could
// address another account's credential.
func ValidateAccountName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("account name cannot be empty")
	}
	if name != strings.TrimSpace(name) {
		return fmt.Errorf("account name %q has leading or trailing whitespace", name)
	}
	if strings.ContainsAny(name, `/\:*?"<>|`) {
		return fmt.Errorf(`account name %q contains a reserved character (/ \ : * ? " < > |)`, name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("account name %q is reserved", name)
	}
	return nil
}

// ValidateBaseURL requires an absolute http(s) URL with a host, and refuses cleartext HTTP
// to anything but loopback — an access token sent over plain HTTP is a credential disclosed
// to every hop in between.
func ValidateBaseURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return errors.New("base URL cannot be empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid base URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("base URL %q must use http or https", raw)
	}
	if u.Host == "" {
		return fmt.Errorf("base URL %q has no host", raw)
	}
	if u.Scheme == "http" && !isLoopback(u.Hostname()) {
		return fmt.Errorf("refusing cleartext http for %q — credentials would be sent unencrypted; use https", u.Host)
	}
	return nil
}

var graphVersionRe = regexp.MustCompile(`^v\d+\.\d+$`)

// ValidateGraphVersion accepts Meta's version literal form, e.g. "v25.0". A malformed
// version would otherwise surface as a baffling 400 on every single call.
func ValidateGraphVersion(v string) error {
	if !graphVersionRe.MatchString(v) {
		return fmt.Errorf("graph version %q must look like v25.0", v)
	}
	return nil
}

func isLoopback(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	}
	return strings.HasPrefix(host, "127.")
}

// FirstNonEmpty implements the flag > env > file > default precedence for string settings.
// Exported because commands resolve their own options the same way.
func FirstNonEmpty(vals ...string) string { return firstNonEmpty(vals...) }

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
