// Command demoserver supplies invented API responses for an account-free VHS recording.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := "127.0.0.1:8642"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	server := &http.Server{
		Addr: addr, Handler: demoHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	fmt.Fprintln(os.Stderr, "demo API listening on", addr)
	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func demoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /v25.0/900001/message_templates":
			writeJSON(w, map[string]any{"data": []any{
				map[string]any{"id": "900101", "name": "demo_order_ready", "language": "en_US", "category": "UTILITY", "status": "APPROVED", "quality_score": map[string]any{"score": "GREEN"}},
				map[string]any{"id": "900102", "name": "demo_delivery_update", "language": "en_US", "category": "UTILITY", "status": "APPROVED", "quality_score": map[string]any{"score": "GREEN"}},
			}})
		case "GET /v25.0/900002/whatsapp_business_profile":
			writeJSON(w, map[string]any{"data": []any{map[string]any{
				"about": "A fictional shop for this recording", "description": "Demo parcels and delivery updates",
				"email": "hello@example.invalid", "websites": []string{"https://shop.example.invalid"}, "vertical": "RETAIL",
			}}})
		case "POST /v25.0/900002/messages":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			// 202-555-0123 is a fictional-use number; the recipient is echoed from the request.
			writeJSON(w, map[string]any{
				"messaging_product": "whatsapp",
				"contacts":          []any{map[string]any{"input": body["to"], "wa_id": body["to"]}},
				"messages":          []any{map[string]any{"id": "wamid.DEMO_MESSAGE_001", "message_status": "accepted"}},
				"demo_request":      body,
			})
		default:
			http.NotFound(w, r)
		}
	})
}
