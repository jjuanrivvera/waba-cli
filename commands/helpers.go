package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// nowUnix is a seam so tests can pin "now" for window computations.
var nowUnix = func() int64 { return time.Now().Unix() }

// parseTimeArg accepts the two forms analytics windows are naturally written in: a Unix
// timestamp, or a calendar date (YYYY-MM-DD, interpreted as midnight UTC).
func parseTimeArg(s string) (int64, error) {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Unix(), nil
	}
	return 0, fmt.Errorf("time %q must be a Unix timestamp or YYYY-MM-DD", s)
}

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
