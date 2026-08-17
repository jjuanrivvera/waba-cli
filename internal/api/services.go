package api

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
)

// Shared response shapes and the helpers that keep command files thin. The Graph API is
// edge-shaped, so instead of per-resource CRUD services these are the building blocks every
// resource command composes: message sends, multipart uploads, success envelopes and the
// dot-notation analytics field expressions.

// SuccessResult is Meta's `{"success": true}` envelope, returned by most mutations that
// have nothing better to say.
type SuccessResult struct {
	Success Bool `json:"success"`
}

// SendResult is the response to POST /{phone-number-id}/messages.
type SendResult struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID ID `json:"id"`
		// MessageStatus appears only on some sends (e.g. "accepted").
		MessageStatus string `json:"message_status,omitempty"`
	} `json:"messages"`
}

// SendMessage posts a message payload to a phone number's /messages edge.
func (c *Client) SendMessage(ctx context.Context, phoneID string, payload any) (*SendResult, error) {
	var out SendResult
	if err := c.PostJSON(ctx, phoneID+"/messages", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MessagePayload builds the common envelope for POST /messages. Type-specific content is
// attached under its own key (e.g. "text", "image").
func MessagePayload(to, msgType string, content any) map[string]any {
	p := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              msgType,
	}
	if content != nil {
		p[msgType] = content
	}
	return p
}

// MediaRef is an uploaded-media id or a hosted link — exactly one is set. Every media
// message type (image, audio, video, document, sticker) takes this shape.
type MediaRef struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// MultipartBody assembles a multipart/form-data body. Returned content type carries the
// boundary, ready for the request's Content-Type header.
func MultipartBody(fields map[string]string, fileField, filename string, file []byte, fileMIME string) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, "", err
		}
	}
	if fileField != "" {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, fileField, filename))
		if fileMIME != "" {
			h.Set("Content-Type", fileMIME)
		}
		part, err := w.CreatePart(h)
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(file); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.FormDataContentType(), nil
}

// PostMultipart sends a multipart body to a version-relative path.
func (c *Client) PostMultipart(ctx context.Context, path string, body []byte, contentType string, out any) error {
	h := http.Header{}
	h.Set("Content-Type", contentType)
	return c.DoInto(ctx, Request{Method: http.MethodPost, Path: path, Body: body, Headers: h}, out)
}

// MediaInfo is the response to GET /{media-id}: a short-lived download URL plus metadata.
type MediaInfo struct {
	ID               ID     `json:"id"`
	URL              string `json:"url"`
	MimeType         string `json:"mime_type"`
	SHA256           string `json:"sha256"`
	FileSize         Int    `json:"file_size"`
	MessagingProduct string `json:"messaging_product"`
}

// AnalyticsExpr builds the dot-notation field expression Meta uses for WABA analytics:
// `analytics.start(123).end(456).granularity(DAY).phone_numbers([...])`.
//
// The expression syntax is Meta's, not URL syntax — values are embedded verbatim, lists as
// JSON arrays — so this builder is the single place that knows how to spell it.
type AnalyticsExpr struct {
	field string
	parts []string
}

// NewAnalyticsExpr starts an expression for a fields= analytics expansion.
func NewAnalyticsExpr(field string) *AnalyticsExpr {
	return &AnalyticsExpr{field: field}
}

// Arg appends `.name(value)` when value is non-empty.
func (a *AnalyticsExpr) Arg(name, value string) *AnalyticsExpr {
	if value != "" {
		a.parts = append(a.parts, fmt.Sprintf("%s(%s)", name, value))
	}
	return a
}

// List appends `.name(["a","b"])` when values exist.
func (a *AnalyticsExpr) List(name string, values []string) *AnalyticsExpr {
	if len(values) == 0 {
		return a
	}
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	a.parts = append(a.parts, fmt.Sprintf("%s([%s])", name, strings.Join(quoted, ",")))
	return a
}

// String renders the full fields= value.
func (a *AnalyticsExpr) String() string {
	if len(a.parts) == 0 {
		return a.field
	}
	return a.field + "." + strings.Join(a.parts, ".")
}

// FieldsQuery wraps an expression into `?fields=...`.
func (a *AnalyticsExpr) FieldsQuery() url.Values {
	return url.Values{"fields": {a.String()}}
}
