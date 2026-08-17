package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticAuth struct{ token string }

func (s *staticAuth) Apply(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+s.token)
}

// newTestClient wires a client to an httptest server with pacing disabled, so tests run at
// full speed and never touch the network.
func newTestClient(t *testing.T, handler http.HandlerFunc, opts ...Option) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	base := []Option{
		WithAuthenticator(&staticAuth{token: "test-token"}),
		WithRateLimit(0),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}),
	}
	return NewClient(srv.URL, "v25.0", append(base, opts...)...)
}

func TestClient_VersionPrefixAndAuth(t *testing.T) {
	var gotPath, gotAuth string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"id":"123"}`))
	})

	var out struct{ ID ID }
	require.NoError(t, c.GetJSON(context.Background(), "123/messages", nil, &out))
	assert.Equal(t, "/v25.0/123/messages", gotPath, "every relative path is version-prefixed")
	assert.Equal(t, "Bearer test-token", gotAuth)
	assert.Equal(t, ID("123"), out.ID)
}

func TestClient_QueryParams(t *testing.T) {
	var got url.Values
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		_, _ = w.Write([]byte(`{}`))
	})

	q := url.Values{"fields": {"name,status"}}
	require.NoError(t, c.GetJSON(context.Background(), "123", q, nil))
	assert.Equal(t, "name,status", got.Get("fields"))
}

func TestClient_RetriesIdempotentOn500(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	require.NoError(t, c.GetJSON(context.Background(), "x", nil, nil))
	assert.Equal(t, int32(3), calls.Load())
}

func TestClient_NeverRetriesPOST(t *testing.T) {
	// A retried POST /messages would double-send a WhatsApp message to a human; the client
	// must surface the failure instead.
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	err := c.PostJSON(context.Background(), "123/messages", map[string]string{"a": "b"}, nil)
	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load(), "POST must not be auto-retried")
}

func TestClient_HonorsRetryAfter(t *testing.T) {
	var calls atomic.Int32
	start := time.Now()
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0.05")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})

	require.NoError(t, c.GetJSON(context.Background(), "x", nil, nil))
	assert.GreaterOrEqual(t, time.Since(start), 50*time.Millisecond,
		"the server's Retry-After must be honoured before the retry")
	assert.Equal(t, int32(2), calls.Load())
}

func TestClient_GraphErrorParsedWithHint(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Error validating access token","type":"OAuthException","code":190,"error_subcode":463,"fbtrace_id":"AbC"}}`))
	})

	err := c.GetJSON(context.Background(), "me", nil, nil)
	require.Error(t, err)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 190, apiErr.Code)
	assert.Equal(t, 463, apiErr.Subcode)
	assert.Contains(t, err.Error(), "waba auth login", "the 190 hint must be actionable")
	assert.Contains(t, err.Error(), "AbC", "fbtrace_id must surface for Meta support")
}

func TestClient_ErrorHintsByCode(t *testing.T) {
	cases := []struct {
		code int
		want string
	}{
		{131047, "24-hour"},
		{131030, "allowed list"},
		{132001, "templates list"},
		{133010, "phone register"},
		{100, "--dry-run"},
		{80007, "rate limit"},
	}
	for _, tc := range cases {
		e := &APIError{StatusCode: 400, Code: tc.code}
		assert.Contains(t, e.Hint(), tc.want, "code %d", tc.code)
	}
	// Unknown code falls through to HTTP status.
	e := &APIError{StatusCode: 500, Code: 999999}
	assert.Contains(t, e.Hint(), "transient")
}

func TestClient_NonJSONErrorBodyStillUseful(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>upstream sad</html>"))
	})

	err := c.PostJSON(context.Background(), "x", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upstream sad")
}

func TestClient_DryRunPrintsRedactedCurl(t *testing.T) {
	var buf bytes.Buffer
	called := false
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
	}, WithDryRun(true, &buf))

	err := c.PostJSON(context.Background(), "123/messages",
		map[string]any{"messaging_product": "whatsapp", "to": "573001112233"}, nil)
	require.NoError(t, err)
	assert.False(t, called, "dry-run must perform no request")

	out := buf.String()
	assert.Contains(t, out, "curl -X POST")
	assert.Contains(t, out, "/v25.0/123/messages")
	assert.Contains(t, out, "messaging_product")
	assert.Contains(t, out, "<redacted")
	assert.NotContains(t, out, "test-token", "the token must never leak into dry-run output")
}

func TestClient_DryRunShowToken(t *testing.T) {
	var buf bytes.Buffer
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {},
		WithDryRun(true, &buf), WithShowToken(true))

	require.NoError(t, c.GetJSON(context.Background(), "me", nil, nil))
	assert.Contains(t, buf.String(), "test-token")
}

func TestClient_AbsoluteURLHostRestriction(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})

	// A lookaside URL on an unexpected host must be refused — the bearer token travels with
	// the request, so this is the exfiltration guard.
	_, err := c.Do(context.Background(), Request{Path: "https://evil.example.com/media"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected host")
}

func TestClient_AbsoluteURLSameHostAllowed(t *testing.T) {
	var gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte("bytes"))
	})

	body, err := c.Do(context.Background(), Request{Path: c.BaseURL() + "/media/blob"})
	require.NoError(t, err)
	assert.Equal(t, "/media/blob", gotPath, "absolute URLs skip the version prefix")
	assert.Equal(t, "bytes", string(body))
}

func TestClient_AllowedHostSuffixes(t *testing.T) {
	c := NewClient("https://graph.facebook.com", "v25.0")
	assert.True(t, c.allowedHost("graph.facebook.com"))
	assert.True(t, c.allowedHost("lookaside.fbsbx.com"))
	assert.True(t, c.allowedHost("scontent.xx.fbcdn.net"))
	assert.True(t, c.allowedHost("mmg.whatsapp.net"))
	assert.False(t, c.allowedHost("evil.example.com"))
	assert.False(t, c.allowedHost("fbsbx.com.evil.example"))
}

func TestClient_AuthSchemeOverride(t *testing.T) {
	var gotAuth, gotOffset string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotOffset = r.Header.Get("file_offset")
		_, _ = w.Write([]byte(`{"h":"handle"}`))
	})

	h := http.Header{}
	h.Set("file_offset", "0")
	_, err := c.Do(context.Background(), Request{
		Method: http.MethodPost, Path: "upload:SESSION", Headers: h, AuthScheme: "OAuth",
		Body: []byte("raw-bytes"),
	})
	require.NoError(t, err)
	assert.Equal(t, "OAuth test-token", gotAuth,
		"the resumable upload API rejects Bearer; the scheme must be swapped")
	assert.Equal(t, "0", gotOffset)
}

func TestClient_UsageHeaderSlowsLimiter(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Business-Use-Case-Usage", `{"102290":[{"type":"whatsapp","call_count":96,"total_cputime":10,"total_time":12}]}`)
		_, _ = w.Write([]byte(`{}`))
	}, WithRateLimit(10))

	require.NoError(t, c.GetJSON(context.Background(), "x", nil, nil))
	assert.Less(t, c.Rate(), 10.0, "≥90% budget usage must slow the limiter down")
}

func TestMaxUsagePercent(t *testing.T) {
	h := http.Header{}
	assert.Zero(t, maxUsagePercent(h))

	h.Set("X-App-Usage", `{"call_count":10,"total_cputime":25,"total_time":5}`)
	assert.Equal(t, 25.0, maxUsagePercent(h))

	h.Set("X-Business-Use-Case-Usage", `{"1":[{"call_count":88}],"2":[{"total_time":91}]}`)
	assert.Equal(t, 91.0, maxUsagePercent(h))

	h.Set("X-Business-Use-Case-Usage", `not-json`)
	h.Set("X-App-Usage", `also-not-json`)
	assert.Zero(t, maxUsagePercent(h))
}

func TestClient_CtrlCCancelsRetryWait(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.GetJSON(ctx, "x", nil, nil) }()

	// Give the first attempt time to fail, then cancel mid-backoff.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("cancel did not interrupt the retry wait — Ctrl-C would hang for 30s")
	}
	assert.Equal(t, int32(1), calls.Load())
}

func TestListAll_WalksCursors(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			assert.Empty(t, r.URL.Query().Get("after"))
			_, _ = w.Write([]byte(`{"data":[{"id":"1"},{"id":"2"}],"paging":{"cursors":{"after":"CUR1"},"next":"https://graph.facebook.com/next"}}`))
		case 2:
			assert.Equal(t, "CUR1", r.URL.Query().Get("after"))
			_, _ = w.Write([]byte(`{"data":[{"id":"3"}],"paging":{"cursors":{"after":"CUR2"}}}`))
		default:
			t.Error("walk must stop when paging.next is absent")
		}
	})

	items, err := c.ListAll(context.Background(), "123/phone_numbers", nil, ListParams{All: true})
	require.NoError(t, err)
	require.Len(t, items, 3)
	var first struct{ ID ID }
	require.NoError(t, json.Unmarshal(items[0], &first))
	assert.Equal(t, ID("1"), first.ID)
}

func TestListAll_SinglePageByDefault(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"data":[{"id":"1"}],"paging":{"cursors":{"after":"MORE"},"next":"https://x"}}`))
	})

	items, err := c.ListAll(context.Background(), "e", nil, ListParams{Limit: 5})
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, int32(1), calls.Load(), "without --all only one page is fetched")
}

func TestListParams_Query(t *testing.T) {
	q := ListParams{Limit: 25, After: "a", Before: "b", Fields: "id,name"}.Query()
	assert.Equal(t, "25", q.Get("limit"))
	assert.Equal(t, "a", q.Get("after"))
	assert.Equal(t, "b", q.Get("before"))
	assert.Equal(t, "id,name", q.Get("fields"))
	assert.Empty(t, ListParams{}.Query())
}

func TestEncodeBody_Forms(t *testing.T) {
	b, ct, err := encodeBody(nil)
	require.NoError(t, err)
	assert.Nil(t, b)
	assert.Empty(t, ct)

	b, ct, err = encodeBody([]byte("raw"))
	require.NoError(t, err)
	assert.Equal(t, "raw", string(b))
	assert.Empty(t, ct, "raw bytes carry their own Content-Type via Headers")

	b, ct, err = encodeBody(json.RawMessage(`{"a":1}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1}`, string(b))
	assert.Equal(t, "application/json", ct)

	b, ct, err = encodeBody(bytes.NewReader([]byte("stream")))
	require.NoError(t, err)
	assert.Equal(t, "stream", string(b))
	assert.Empty(t, ct)

	b, ct, err = encodeBody(map[string]int{"n": 1})
	require.NoError(t, err)
	assert.JSONEq(t, `{"n":1}`, string(b))
	assert.Equal(t, "application/json", ct)

	_, _, err = encodeBody(func() {})
	require.Error(t, err, "unencodable bodies must fail loudly")
}

func TestShellQuote(t *testing.T) {
	assert.Equal(t, `'plain'`, shellQuote("plain"))
	assert.Equal(t, `'it'\''s'`, shellQuote("it's"))
}
