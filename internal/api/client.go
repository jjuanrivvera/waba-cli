package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Authenticator applies credentials to an outgoing request. The Cloud API has exactly one
// scheme (a bearer access token), so this stays deliberately minimal.
type Authenticator interface {
	Apply(req *http.Request)
}

// Client talks to the Graph API for one WhatsApp Business account.
type Client struct {
	http     *http.Client
	auth     Authenticator
	baseURL  string // https://graph.facebook.com, or a mock in tests
	version  string // e.g. v25.0 — prefixed onto every relative path
	limiter  *limiter
	retry    RetryPolicy
	ua       string
	dryRun   bool
	showTok  bool
	verbose  io.Writer // non-nil enables request tracing
	dryRunTo io.Writer

	// retryAfterHint carries a server-supplied Retry-After from a failed attempt to the wait
	// before the next one. It is client state rather than a local because the wait happens at
	// the top of the following loop iteration, after the response that carried the header has
	// already been consumed and closed.
	retryAfterHint time.Duration
}

// Option configures a Client.
type Option func(*Client)

func WithHTTPClient(h *http.Client) Option     { return func(c *Client) { c.http = h } }
func WithAuthenticator(a Authenticator) Option { return func(c *Client) { c.auth = a } }
func WithRetryPolicy(p RetryPolicy) Option     { return func(c *Client) { c.retry = p } }
func WithUserAgent(ua string) Option           { return func(c *Client) { c.ua = ua } }
func WithVerbose(w io.Writer) Option           { return func(c *Client) { c.verbose = w } }
func WithGraphVersion(v string) Option         { return func(c *Client) { c.version = v } }

// WithRateLimit sets the steady-state request rate. Zero or less disables pacing.
func WithRateLimit(rps float64) Option { return func(c *Client) { c.limiter = newLimiter(rps) } }

// WithDryRun makes every request print an equivalent curl command to w and perform no I/O.
func WithDryRun(on bool, w io.Writer) Option {
	return func(c *Client) { c.dryRun = on; c.dryRunTo = w }
}

// WithTimeout sets the per-request timeout in seconds. Zero or less leaves the default,
// which matters because media uploads over slow links can legitimately take minutes.
func WithTimeout(seconds int) Option {
	return func(c *Client) {
		if seconds > 0 {
			c.http.Timeout = time.Duration(seconds) * time.Second
		}
	}
}

// WithShowToken un-redacts credentials in dry-run output. Off by default so a copied command
// never leaks a token into a terminal history or a bug report.
func WithShowToken(on bool) Option { return func(c *Client) { c.showTok = on } }

// DefaultRPS is the steady-state rate. Meta publishes percentage-based budgets rather than a
// fixed figure; this conservative floor is adjusted live from X-Business-Use-Case-Usage.
const DefaultRPS = 10

// NewClient builds a client for one account.
func NewClient(baseURL, graphVersion string, opts ...Option) *Client {
	c := &Client{
		http:    &http.Client{Timeout: 60 * time.Second},
		baseURL: strings.TrimRight(baseURL, "/"),
		version: graphVersion,
		limiter: newLimiter(DefaultRPS),
		retry:   DefaultRetryPolicy(),
		ua:      "waba-cli",
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// DryRun reports whether the client is in dry-run mode.
func (c *Client) DryRun() bool { return c.dryRun }

// GraphVersion reports the version every relative path is prefixed with.
func (c *Client) GraphVersion() string { return c.version }

// BaseURL reports the Graph host in use (doctor output).
func (c *Client) BaseURL() string { return c.baseURL }

// Rate reports the limiter's current effective requests/second.
func (c *Client) Rate() float64 {
	if c.limiter == nil {
		return 0
	}
	return c.limiter.rate()
}

// Request describes one Graph API call. Commands build these; only the client turns them
// into URLs. Path is version-relative ("12345/messages"); an absolute http(s) Path is used
// verbatim — that is how media downloads follow the lookaside URL the API returned.
type Request struct {
	Method  string      // http.MethodGet, ...
	Path    string      // "{id}/edge", or an absolute URL for media downloads
	Query   url.Values  // query parameters
	Body    any         // JSON-encoded when non-nil; []byte and io.Reader pass through
	Headers http.Header // extra headers (multipart uploads set Content-Type here)
	// AuthScheme replaces the default "Bearer" prefix. The resumable upload API demands
	// the legacy "OAuth <token>" form and rejects Bearer.
	AuthScheme string
	// NoAuth skips credential application (doctor's unauthenticated reachability probe).
	NoAuth bool
}

// Do performs a request and returns the raw response body, so callers that need the
// untouched payload (-o json, `waba api`) never re-encode what the server sent.
func (c *Client) Do(ctx context.Context, r Request) ([]byte, error) {
	//nolint:bodyclose // doWithResponse reads and closes the body before returning; the
	// *http.Response it hands back is retained only for its status and headers.
	body, _, err := c.doWithResponse(ctx, r)
	return body, err
}

// DoInto performs a request and unmarshals the response into out (nil discards).
func (c *Client) DoInto(ctx context.Context, r Request, out any) error {
	body, err := c.Do(ctx, r)
	if err != nil {
		return err
	}
	if out == nil || len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s %s: %w", r.Method, r.Path, err)
	}
	return nil
}

// GetJSON is the common case: a GET whose JSON response is decoded into out.
func (c *Client) GetJSON(ctx context.Context, path string, q url.Values, out any) error {
	return c.DoInto(ctx, Request{Method: http.MethodGet, Path: path, Query: q}, out)
}

// PostJSON posts a JSON body and decodes the response into out.
func (c *Client) PostJSON(ctx context.Context, path string, body, out any) error {
	return c.DoInto(ctx, Request{Method: http.MethodPost, Path: path, Body: body}, out)
}

func (c *Client) doWithResponse(ctx context.Context, r Request) ([]byte, *http.Response, error) {
	if r.Method == "" {
		r.Method = http.MethodGet
	}

	target, err := c.buildURL(r)
	if err != nil {
		return nil, nil, err
	}

	payload, contentType, err := encodeBody(r.Body)
	if err != nil {
		return nil, nil, err
	}

	if c.dryRun {
		return nil, nil, c.printCurl(ctx, r, target, payload, contentType)
	}

	var lastErr error
	attempts := c.retry.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			// Ctrl-C must interrupt the wait, not be swallowed by it.
			if err := waitFor(ctx, c.retryDelay(attempt-1)); err != nil {
				return nil, nil, err
			}
		}
		if c.limiter != nil {
			if err := c.limiter.wait(ctx); err != nil {
				return nil, nil, err
			}
		}

		req, err := c.newHTTPRequest(ctx, r, target, payload, contentType)
		if err != nil {
			return nil, nil, err
		}

		start := time.Now()
		resp, err := c.http.Do(req)
		if c.verbose != nil {
			c.trace(r.Method, target, resp, err, time.Since(start))
		}
		if resp != nil && c.limiter != nil {
			c.limiter.observe(resp, time.Now())
			c.limiter.observeUsage(maxUsagePercent(resp.Header), time.Now())
		}

		if err != nil {
			lastErr = err
			if shouldRetry(r.Method, nil, err) && attempt < attempts-1 {
				c.retryAfterHint = 0
				continue
			}
			return nil, nil, fmt.Errorf("%s %s: %w", r.Method, target, err)
		}

		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if isIdempotent(r.Method) && attempt < attempts-1 {
				continue
			}
			return nil, resp, fmt.Errorf("%s %s: read body: %w", r.Method, target, readErr)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, resp, nil
		}

		apiErr := parseAPIError(resp.StatusCode, resp.Status, r.Method, target, respBody)
		lastErr = apiErr
		if shouldRetry(r.Method, resp, nil) && attempt < attempts-1 {
			// Retry-After is authoritative when present; remember it for the next wait.
			if d, ok := retryAfter(resp, time.Now()); ok {
				c.retryAfterHint = d
			} else {
				c.retryAfterHint = 0
			}
			continue
		}
		return respBody, resp, apiErr
	}
	return nil, nil, lastErr
}

// retryDelay is the wait before the given (0-based) retry attempt: the server's own
// Retry-After when it supplied one, otherwise full-jitter exponential backoff.
func (c *Client) retryDelay(attempt int) time.Duration {
	if c.retryAfterHint > 0 {
		d := c.retryAfterHint
		c.retryAfterHint = 0
		if d > c.retry.MaxDelay && c.retry.MaxDelay > 0 && c.verbose != nil {
			// Respect the server, but do not hang for minutes without saying why.
			fmt.Fprintf(c.verbose, "  retry-after %s exceeds max delay, waiting anyway\n", d)
		}
		return d
	}
	return backoff(c.retry, attempt)
}

// maxResponseBytes caps a single response. Media downloads are the largest legitimate
// payloads (Cloud API media caps at 100MB); this stays above that while stopping a
// pathological response from exhausting memory.
const maxResponseBytes = 512 << 20

func (c *Client) buildURL(r Request) (string, error) {
	var raw string
	if strings.HasPrefix(r.Path, "http://") || strings.HasPrefix(r.Path, "https://") {
		// Absolute URLs come from API responses (media lookaside). The token travels with
		// the request, so an unexpected host would be an exfiltration primitive — refuse it.
		u, err := url.Parse(r.Path)
		if err != nil {
			return "", fmt.Errorf("parse media url: %w", err)
		}
		if !c.allowedHost(u.Hostname()) {
			return "", fmt.Errorf("refusing to send credentials to unexpected host %q", u.Hostname())
		}
		raw = r.Path
	} else {
		raw = c.baseURL + "/" + c.version + "/" + strings.TrimLeft(r.Path, "/")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("build url for %s %s: %w", r.Method, r.Path, err)
	}
	if len(r.Query) > 0 {
		q := u.Query()
		for k, vs := range r.Query {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

// allowedHost accepts the configured Graph host plus Meta's CDN domains, which is where
// media lookaside URLs live.
func (c *Client) allowedHost(host string) bool {
	host = strings.ToLower(host)
	if base, err := url.Parse(c.baseURL); err == nil && strings.EqualFold(base.Hostname(), host) {
		return true
	}
	for _, suffix := range []string{".facebook.com", ".fbcdn.net", ".fbsbx.com", ".whatsapp.net"} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

func (c *Client) newHTTPRequest(ctx context.Context, r Request, target string, payload []byte, contentType string) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, target, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.ua)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, vs := range r.Headers {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	if !r.NoAuth && c.auth != nil {
		c.auth.Apply(req)
		if r.AuthScheme != "" {
			// The resumable upload API rejects "Bearer"; swap the scheme, keep the token.
			if v, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer "); ok {
				req.Header.Set("Authorization", r.AuthScheme+" "+v)
			}
		}
	}
	return req, nil
}

// encodeBody turns a request body into bytes plus a Content-Type. []byte and io.Reader pass
// through untouched so callers can send pre-built multipart or raw payloads.
func encodeBody(body any) ([]byte, string, error) {
	switch v := body.(type) {
	case nil:
		return nil, "", nil
	case []byte:
		return v, "", nil
	case json.RawMessage:
		return v, "application/json", nil
	case io.Reader:
		b, err := io.ReadAll(v)
		if err != nil {
			return nil, "", err
		}
		return b, "", nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, "", fmt.Errorf("encode request body: %w", err)
		}
		return b, "application/json", nil
	}
}

// maxUsagePercent extracts the highest utilisation percentage from Meta's usage headers.
// X-Business-Use-Case-Usage maps a business id to entries with call_count/total_cputime/
// total_time percentages; X-App-Usage carries the same three keys flat.
func maxUsagePercent(h http.Header) float64 {
	highest := 0.0
	// Entries mix strings ("type":"whatsapp") with numbers, so decode loosely and pick out
	// the three percentage keys.
	scan := func(entry map[string]any) {
		for _, k := range []string{"call_count", "total_cputime", "total_time"} {
			if f, ok := usageFloat(entry[k]); ok {
				highest = max(highest, f)
			}
		}
	}
	if raw := h.Get("X-Business-Use-Case-Usage"); raw != "" {
		var m map[string][]map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err == nil {
			for _, entries := range m {
				for _, e := range entries {
					scan(e)
				}
			}
		}
	}
	if raw := h.Get("X-App-Usage"); raw != "" {
		var e map[string]any
		if err := json.Unmarshal([]byte(raw), &e); err == nil {
			scan(e)
		}
	}
	return highest
}

func usageFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	}
	return 0, false
}

func (c *Client) trace(method, target string, resp *http.Response, err error, took time.Duration) {
	switch {
	case err != nil:
		fmt.Fprintf(c.verbose, "→ %s %s\n← error after %s: %v\n", method, target, took.Round(time.Millisecond), err)
	default:
		fmt.Fprintf(c.verbose, "→ %s %s\n← %s in %s\n", method, target, resp.Status, took.Round(time.Millisecond))
		if u := resp.Header.Get("X-Business-Use-Case-Usage"); u != "" {
			fmt.Fprintf(c.verbose, "  usage: %s\n", u)
		}
	}
}

// printCurl writes the equivalent curl command for a request instead of sending it.
//
// The output is meant to be pasted into a shell and to reproduce the call exactly, which is
// why it is properly single-quote escaped rather than merely printed. Credentials are
// redacted unless --show-token was passed.
func (c *Client) printCurl(ctx context.Context, r Request, target string, payload []byte, contentType string) error {
	w := c.dryRunTo
	if w == nil {
		return nil
	}

	req, err := c.newHTTPRequest(ctx, r, target, payload, contentType)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("curl")
	if r.Method != http.MethodGet {
		fmt.Fprintf(&b, " -X %s", r.Method)
	}

	names := make([]string, 0, len(req.Header))
	for k := range req.Header {
		names = append(names, k)
	}
	sort.Strings(names) // deterministic output; map order would churn the dry-run diff
	for _, k := range names {
		v := req.Header.Get(k)
		if !c.showTok && isSecretHeader(k) {
			v = redactHeader(k, v)
		}
		fmt.Fprintf(&b, " \\\n  -H %s", shellQuote(k+": "+v))
	}

	if len(payload) > 0 {
		fmt.Fprintf(&b, " \\\n  -d %s", shellQuote(string(payload)))
	}
	fmt.Fprintf(&b, " \\\n  %s\n", shellQuote(target))

	_, err = io.WriteString(w, b.String())
	return err
}

func isSecretHeader(name string) bool {
	switch strings.ToLower(name) {
	case "authorization", "cookie", "proxy-authorization":
		return true
	}
	return false
}

// redactHeader keeps the scheme visible (so the reader can see which auth form was used)
// while hiding the credential itself.
func redactHeader(name, value string) string {
	if strings.EqualFold(name, "authorization") {
		if scheme, _, ok := strings.Cut(value, " "); ok {
			return scheme + " <redacted — re-run with --show-token to reveal>"
		}
	}
	return "<redacted>"
}

// shellQuote wraps s in single quotes, escaping any embedded single quote the POSIX way.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
