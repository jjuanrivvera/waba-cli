package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError is a non-2xx response from any Atlassian product, decoded into one shape.
//
// The five API families report failures differently — Jira uses `errorMessages` plus a field
// keyed `errors` map, Confluence v2 uses an `errors` array of `{status,title,detail}`,
// Confluence v1 uses `{statusCode,message}`, and JSM uses `{errorMessage}`. Collapsing them
// here means every caller and every command renders errors the same way.
type APIError struct {
	StatusCode int
	Status     string
	Method     string
	URL        string
	Code       string
	Message    string
	Details    []string
	Body       string
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s: %d", e.Method, e.URL, e.StatusCode)
	if e.Status != "" && !strings.HasPrefix(e.Status, fmt.Sprint(e.StatusCode)) {
		fmt.Fprintf(&b, " %s", e.Status)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	for _, d := range e.Details {
		fmt.Fprintf(&b, "\n  - %s", d)
	}
	if h := e.Hint(); h != "" {
		fmt.Fprintf(&b, "\nhint: %s", h)
	}
	return b.String()
}

// Hint maps a status (and a few Atlassian-specific failure shapes) to the next thing to try.
// An error that only says "403" costs a support round trip; one that names the command to run
// does not.
func (e *APIError) Hint() string {
	switch e.StatusCode {
	case http.StatusUnauthorized:
		return "credentials rejected — run `atlassian auth login` (Cloud API tokens are created at https://id.atlassian.com/manage-profile/security/api-tokens)"
	case http.StatusForbidden:
		// Atlassian returns 403 both for "you lack the permission" and for "your token is
		// fine but the product isn't licensed for you", which need different fixes.
		return "authenticated but not permitted — check the account's project/space permissions and that it has a licence for this product; `atlassian auth status` shows who you are"
	case http.StatusNotFound:
		return "not found — verify the id/key with the matching `list` command, and confirm the right site is selected (`atlassian config list-sites`)"
	case http.StatusMethodNotAllowed:
		return "wrong method for this path — `atlassian op describe <operationId>` shows the documented method"
	case http.StatusConflict:
		return "conflict — the resource changed since you read it; re-read and retry"
	case http.StatusRequestEntityTooLarge:
		return "payload too large — Atlassian caps attachment and body size; split the request"
	case http.StatusUnprocessableEntity:
		return "the payload was understood but rejected — a required field is missing or a value is invalid; `atlassian op describe <operationId>` lists the documented parameters"
	case http.StatusTooManyRequests:
		return "rate limited — the client already backs off and honours Retry-After; lower --rps or narrow the query if this persists"
	case http.StatusGatewayTimeout, http.StatusBadGateway, http.StatusServiceUnavailable:
		return "server error, usually transient — idempotent requests were retried automatically"
	}
	if e.StatusCode >= 500 {
		return "server error, usually transient — retry, then check https://status.atlassian.com"
	}
	if e.StatusCode == http.StatusBadRequest {
		// A 400 mentioning ADF is nearly always a plain string sent where rich text is
		// required. Jira spells this several ways depending on the endpoint — "Operation
		// value must be an Atlassian Document", "atlassianDocument", "ADF" — so match all of
		// them rather than the one spelling that happened to be seen first.
		haystack := strings.ToLower(e.Body + " " + e.Message)
		if strings.Contains(haystack, "atlassiandocument") ||
			strings.Contains(haystack, "atlassian document") ||
			strings.Contains(haystack, "adf") {
			return "Jira v3 expects Atlassian Document Format for rich-text fields — pass text/Markdown and let the CLI convert it, or supply raw ADF with the `-adf` variant of the flag"
		}
		return "the request was malformed — check required parameters with `atlassian op describe <operationId>`"
	}
	return ""
}

// Is lets callers match on status with errors.Is(err, api.ErrNotFound).
func (e *APIError) Is(target error) bool {
	var sentinel statusError
	if errors.As(target, &sentinel) {
		return e.StatusCode == int(sentinel)
	}
	return false
}

// statusError is the concrete type behind the sentinel errors below.
type statusError int

func (s statusError) Error() string { return fmt.Sprintf("http %d", int(s)) }

// Sentinels for errors.Is checks.
var (
	ErrUnauthorized = statusError(http.StatusUnauthorized)
	ErrForbidden    = statusError(http.StatusForbidden)
	ErrNotFound     = statusError(http.StatusNotFound)
	ErrRateLimited  = statusError(http.StatusTooManyRequests)
	ErrConflict     = statusError(http.StatusConflict)
)

// maxErrorBody caps how much of a failed response is retained. Atlassian can return a full
// HTML error page from an edge proxy; keeping megabytes of it in an error serves nobody.
const maxErrorBody = 8 << 10

// parseAPIError builds an APIError from a failed response body, trying each product's error
// shape in turn and falling back to the raw text.
func parseAPIError(status int, statusText, method, url string, body []byte) *APIError {
	e := &APIError{
		StatusCode: status,
		Status:     statusText,
		Method:     method,
		URL:        url,
		Body:       truncate(string(body), maxErrorBody),
	}

	// Jira platform / Agile: {"errorMessages":[...],"errors":{"field":"reason"}}
	var jira struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
		Message       string            `json:"message"`
		ErrorMessage  string            `json:"errorMessage"` // JSM
		StatusCode    int               `json:"statusCode"`   // Confluence v1
	}
	if err := json.Unmarshal(body, &jira); err == nil {
		if len(jira.ErrorMessages) > 0 {
			e.Message = jira.ErrorMessages[0]
			e.Details = append(e.Details, jira.ErrorMessages[1:]...)
		}
		for field, reason := range jira.Errors {
			e.Details = append(e.Details, fmt.Sprintf("%s: %s", field, reason))
		}
		if e.Message == "" {
			e.Message = firstNonEmpty(jira.Message, jira.ErrorMessage)
		}
	}

	// Confluence v2: {"errors":[{"status":404,"code":"...","title":"...","detail":"..."}]}
	if e.Message == "" {
		var conf struct {
			Errors []struct {
				Code   string `json:"code"`
				Title  string `json:"title"`
				Detail string `json:"detail"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(body, &conf); err == nil && len(conf.Errors) > 0 {
			e.Code = conf.Errors[0].Code
			e.Message = firstNonEmpty(conf.Errors[0].Title, conf.Errors[0].Detail)
			for _, extra := range conf.Errors[1:] {
				e.Details = append(e.Details, firstNonEmpty(extra.Title, extra.Detail))
			}
		}
	}

	if e.Message == "" {
		// Not JSON, or an unrecognised shape (an HTML error page from a proxy, typically).
		if s := strings.TrimSpace(string(body)); s != "" && !strings.HasPrefix(s, "<") {
			e.Message = truncate(s, 400)
		}
	}
	// sortDetails keeps the rendering of a map-derived field-error list stable across runs.
	sortDetails(e.Details)
	return e
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
