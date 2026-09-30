package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The recording is only as trustworthy as the stand-in behind it: every path the
// tape drives has to answer, and nothing else may answer at all.
func TestDemoHandlerAnswersOnlyTheRecordedPaths(t *testing.T) {
	h := demoHandler()

	for _, path := range []string{"/v25.0/900001/message_templates", "/v25.0/900002/whatsapp_business_profile"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("GET %s content type = %q", path, ct)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Errorf("GET %s did not answer JSON: %v", path, err)
		}
		if _, ok := body["data"]; !ok {
			t.Errorf("GET %s has no data envelope: %s", path, rec.Body)
		}
	}

	// A path nobody recorded is a 404, not an empty success that would look
	// like a real answer in the recording.
	for _, r := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/v25.0/900001/phone_numbers", nil),
		httptest.NewRequest(http.MethodGet, "/v25.0/900002/messages", nil), // right path, wrong method
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404", r.Method, r.URL.Path, rec.Code)
		}
	}
}

// A send echoes what it was given, so the recording shows the number that was
// actually typed rather than a constant that happens to match.
func TestDemoHandlerEchoesTheSend(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v25.0/900002/messages",
		strings.NewReader(`{"messaging_product":"whatsapp","to":"12025550123","type":"text"}`))
	demoHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST messages = %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Contacts []struct {
			WaID string `json:"wa_id"`
		} `json:"contacts"`
		Messages []struct {
			ID     string `json:"id"`
			Status string `json:"message_status"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Contacts) != 1 || body.Contacts[0].WaID != "12025550123" {
		t.Errorf("the recipient should come back from the request: %s", rec.Body)
	}
	if len(body.Messages) != 1 || body.Messages[0].ID == "" || body.Messages[0].Status != "accepted" {
		t.Errorf("a send should come back accepted with an id: %s", rec.Body)
	}
}

func TestDemoHandlerRejectsBrokenJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	demoHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v25.0/900002/messages", strings.NewReader("not json")))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a broken body = %d, want 400", rec.Code)
	}
}
