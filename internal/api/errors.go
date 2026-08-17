package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError is a Graph API failure with enough context to act on. Meta wraps every error in
// one envelope: {"error":{message,type,code,error_subcode,error_data,fbtrace_id}} — the
// numeric code, not the HTTP status, is what actually distinguishes "token expired" from
// "outside the 24-hour window", so hints key off both.
type APIError struct {
	StatusCode int
	Status     string
	Method     string
	URL        string

	Code      int    // Graph error code, e.g. 190 (bad token), 131047 (re-engagement)
	Subcode   int    // error_subcode, e.g. 463 (token expired)
	Type      string // e.g. OAuthException
	Message   string
	UserTitle string // error_user_title — already human-oriented when present
	UserMsg   string // error_user_msg
	Details   string // error_data.details — Cloud API puts the useful part here
	TraceID   string // fbtrace_id, what Meta support asks for
	Body      []byte // raw body for -o json and debugging
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s: %s", e.Method, e.URL, e.Status)
	if e.Code != 0 {
		fmt.Fprintf(&b, " (code %d", e.Code)
		if e.Subcode != 0 {
			fmt.Fprintf(&b, ".%d", e.Subcode)
		}
		b.WriteString(")")
	}
	msg := e.Message
	if e.Details != "" {
		msg = e.Details
	}
	if msg != "" {
		fmt.Fprintf(&b, ": %s", msg)
	}
	if hint := e.Hint(); hint != "" {
		fmt.Fprintf(&b, "\n  hint: %s", hint)
	}
	if e.TraceID != "" {
		fmt.Fprintf(&b, "\n  fbtrace_id: %s (quote this to Meta support)", e.TraceID)
	}
	return b.String()
}

// Hint maps the failure to the action that usually fixes it. Graph codes are checked before
// HTTP status because Meta reuses 400 for half its distinct failures.
func (e *APIError) Hint() string {
	switch e.Code {
	case 190:
		return "access token invalid or expired — run `waba auth login` with a fresh System User token"
	case 0:
		// No Graph code parsed; fall through to the HTTP status below.
	case 10, 200, 201, 202, 203, 204, 205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 294, 299:
		return "permission missing — the token needs whatsapp_business_messaging and whatsapp_business_management, granted to this WABA in Business Manager"
	case 3:
		return "app capability missing — check the app has the WhatsApp product added and API access enabled"
	case 4, 17, 32, 613:
		return "app-level rate limit hit — slow down and retry after the X-Business-Use-Case-Usage window resets"
	case 80007:
		return "WABA rate limit hit — slow down; check usage in WhatsApp Manager > Insights"
	case 130429:
		return "messaging throughput limit reached — queue and retry with backoff"
	case 131056:
		return "too many messages to this exact user pair — pause sends to this recipient briefly"
	case 100:
		if e.Subcode == 33 {
			return "object missing or no permission to it — verify the id (`waba phone list`, `waba templates list`) and that the token can access this WABA"
		}
		return "invalid parameter — compare the payload against the reference (`waba <cmd> --help` shows an example; --dry-run prints the exact request)"
	case 131030:
		return "recipient not in the allowed list — in development mode add the number under App Dashboard > WhatsApp > API Setup"
	case 131047, 131048:
		return "outside the 24-hour customer service window — send an approved template instead (`waba send template`)"
	case 131026:
		return "message undeliverable — the number may not be on WhatsApp, or its client is too old for this message type"
	case 131009:
		return "invalid message id — pass the wamid exactly as delivered in the webhook, within 30 days"
	case 132000:
		return "template parameter count mismatch — the components you sent don't match the approved template's placeholders"
	case 132001:
		return "template not found or not approved in this language — `waba templates list --status APPROVED` to check"
	case 133010:
		return "phone number not registered on Cloud API — run `waba phone register --pin <6-digit>`"
	case 368:
		return "temporarily blocked for policy violations — check WhatsApp Manager for the account status"
	case 131031:
		return "account locked — check WhatsApp Manager; messaging is disabled until resolved"
	case 131042:
		return "payment issue — verify the WABA's payment method in Business Manager"
	}

	switch e.StatusCode {
	case http.StatusUnauthorized:
		return "credentials rejected — run `waba auth login`"
	case http.StatusForbidden:
		return "authenticated but not permitted — check the token's scopes and WABA access"
	case http.StatusNotFound:
		return "not found — verify the id with the matching `list` command and confirm the Graph version supports this endpoint"
	case http.StatusTooManyRequests:
		return "rate limited — the CLI backs off automatically; consider lowering --rate"
	}
	if e.StatusCode >= 500 {
		return "Meta server error, usually transient — retry, then check https://metastatus.com/whatsapp-business-api"
	}
	return ""
}

// graphEnvelope mirrors Meta's error wrapper.
type graphEnvelope struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		Subcode   int    `json:"error_subcode"`
		UserTitle string `json:"error_user_title"`
		UserMsg   string `json:"error_user_msg"`
		TraceID   string `json:"fbtrace_id"`
		ErrorData struct {
			Details string `json:"details"`
		} `json:"error_data"`
	} `json:"error"`
}

// parseAPIError builds an APIError from a non-2xx response. A body that isn't the Graph
// envelope (HTML from a proxy, an empty 502) still yields a useful error.
func parseAPIError(statusCode int, status, method, url string, body []byte) *APIError {
	e := &APIError{StatusCode: statusCode, Status: status, Method: method, URL: url, Body: body}
	var env graphEnvelope
	if err := json.Unmarshal(body, &env); err == nil {
		e.Code = env.Error.Code
		e.Subcode = env.Error.Subcode
		e.Type = env.Error.Type
		e.Message = env.Error.Message
		e.UserTitle = env.Error.UserTitle
		e.UserMsg = env.Error.UserMsg
		e.Details = env.Error.ErrorData.Details
		e.TraceID = env.Error.TraceID
	}
	if e.Message == "" {
		e.Message = strings.TrimSpace(truncateBody(body))
	}
	return e
}

// truncateBody keeps error output readable when the body is a page of HTML.
func truncateBody(b []byte) string {
	const max = 300
	s := string(b)
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
