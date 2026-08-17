package commands

import (
	"encoding/json"
	"net/url"
)

// urlValues is a terse constructor for query parameters: urlValues("limit", "1").
func urlValues(kv ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return v
}

// jsonMap decodes raw JSON into a map for rendering, tolerating failure by returning the
// raw string — an odd payload should display, not error.
func jsonMap(raw []byte) any {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}
