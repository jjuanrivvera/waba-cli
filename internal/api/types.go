package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Flexible JSON types. Atlassian is inconsistent about scalar encodings across its five API
// families and across versions of the same endpoint — ids arrive as both `"10001"` and
// `10001`, counts as both numbers and numeric strings, and single-valued fields as both a
// bare object and a one-element array. Decoding into plain Go types therefore fails on real
// payloads; these types absorb the variation at the edge so the rest of the CLI can assume
// one shape.

// ID is an Atlassian identifier. It decodes from a JSON string or number and always encodes
// as a string.
//
// Encoding as a string is deliberate: Jira issue and Confluence content ids exceed 2^53 on
// large instances, so round-tripping through float64 — which is what `any` decoding does —
// silently corrupts them.
type ID string

func (i *ID) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*i = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*i = ID(s)
		return nil
	}
	// Numbers are taken verbatim from the raw bytes rather than parsed and reprinted, which
	// keeps arbitrary precision without a big.Int round trip.
	if err := validJSONNumber(string(b)); err != nil {
		return fmt.Errorf("id: %w", err)
	}
	*i = ID(b)
	return nil
}

func (i ID) MarshalJSON() ([]byte, error) { return json.Marshal(string(i)) }
func (i ID) String() string               { return string(i) }
func (i ID) Empty() bool                  { return i == "" }

// Int accepts a JSON number or a numeric string.
type Int int64

func (n *Int) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*n = 0
		return nil
	}
	s := string(b)
	if b[0] == '"' {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		str = strings.TrimSpace(str)
		if str == "" {
			*n = 0
			return nil
		}
		s = str
	}
	if err := validJSONNumber(s); err != nil {
		return fmt.Errorf("int: %w", err)
	}
	// Int64 first: parsing as float64 first would lose precision above 2^53, which real
	// Atlassian ids exceed.
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		*n = Int(v)
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("int: %q is not a number", s)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("int: %q is not finite", s)
	}
	*n = Int(int64(f))
	return nil
}

func (n Int) MarshalJSON() ([]byte, error) { return json.Marshal(int64(n)) }
func (n Int) Int64() int64                 { return int64(n) }
func (n Int) String() string               { return strconv.FormatInt(int64(n), 10) }

// Bool accepts a real JSON bool or the string forms Atlassian sometimes emits in query-echo
// and property payloads ("true", "1", "yes", "on").
type Bool bool

func (b *Bool) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		*b = false
		return nil
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "1", "yes", "on":
			*b = true
		case "false", "0", "no", "off", "":
			*b = false
		default:
			return fmt.Errorf("bool: %q is not a boolean", s)
		}
		return nil
	}
	var v bool
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("bool: %w", err)
	}
	*b = Bool(v)
	return nil
}

func (b Bool) MarshalJSON() ([]byte, error) { return json.Marshal(bool(b)) }
func (b Bool) Bool() bool                   { return bool(b) }

// Number is an exact decimal held as text — used for time-tracking and story-point style
// fields. Keeping the original digits avoids the rounding a float64 would introduce on
// values like 0.1, which matters when the value is written back to the API.
type Number string

func (m *Number) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*m = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*m = ""
			return nil
		}
		if err := validJSONNumber(s); err != nil {
			return fmt.Errorf("number: %w", err)
		}
		*m = Number(s)
		return nil
	}
	if err := validJSONNumber(string(b)); err != nil {
		return fmt.Errorf("number: %w", err)
	}
	*m = Number(b)
	return nil
}

// MarshalJSON emits the value as a JSON number so it round-trips to the API unchanged.
func (m Number) MarshalJSON() ([]byte, error) {
	if m == "" {
		return []byte("null"), nil
	}
	return []byte(m), nil
}

func (m Number) String() string { return string(m) }

// Float returns the value as a float64 for arithmetic; the text form remains authoritative.
func (m Number) Float() (float64, error) {
	if m == "" {
		return 0, nil
	}
	return strconv.ParseFloat(string(m), 64)
}

// Ref is a nested `{id, name, key, ...}` reference — Jira returns these for project, status,
// priority, issue type, assignee and many more. Rendering a whole object into a table cell
// is useless, so Ref carries the pieces a table actually wants.
type Ref struct {
	ID          ID     `json:"id,omitempty"`
	Key         string `json:"key,omitempty"`
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	AccountID   string `json:"accountId,omitempty"`
	Email       string `json:"emailAddress,omitempty"`
	Self        string `json:"self,omitempty"`
}

// Label picks the most human-readable identifier present, so a table cell shows
// "In Progress" rather than an opaque id.
func (r Ref) Label() string {
	switch {
	case r.DisplayName != "":
		return r.DisplayName
	case r.Name != "":
		return r.Name
	case r.Key != "":
		return r.Key
	case r.Email != "":
		return r.Email
	case !r.ID.Empty():
		return r.ID.String()
	case r.AccountID != "":
		return r.AccountID
	}
	return ""
}

func (r Ref) String() string { return r.Label() }

// UnmarshalJSON also accepts a bare string, because several Atlassian endpoints degrade a
// reference to just its name or key depending on the `expand` parameter.
func (r *Ref) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*r = Ref{}
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*r = Ref{Name: s}
		return nil
	}
	type plain Ref // avoid recursing into this method
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*r = Ref(p)
	return nil
}

// Refs decodes either a single reference object or an array of them.
type Refs []Ref

func (r *Refs) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*r = nil
		return nil
	}
	if b[0] == '[' {
		var many []Ref
		if err := json.Unmarshal(b, &many); err != nil {
			return err
		}
		*r = many
		return nil
	}
	var one Ref
	if err := json.Unmarshal(b, &one); err != nil {
		return err
	}
	*r = Refs{one}
	return nil
}

// Labels returns each reference's human-readable label.
func (r Refs) Labels() []string {
	out := make([]string, 0, len(r))
	for _, x := range r {
		if l := x.Label(); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func (r Refs) String() string { return strings.Join(r.Labels(), ", ") }

// StringOrSlice accepts `"x"` or `["x","y"]`. Atlassian uses both for labels, fields and
// expand echoes.
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*s = nil
		return nil
	}
	if b[0] == '[' {
		var many []string
		if err := json.Unmarshal(b, &many); err != nil {
			return err
		}
		*s = many
		return nil
	}
	var one string
	if err := json.Unmarshal(b, &one); err != nil {
		return err
	}
	*s = StringOrSlice{one}
	return nil
}

func (s StringOrSlice) String() string { return strings.Join(s, ", ") }

// validJSONNumber rejects anything the JSON grammar does not admit as a number, including
// the NaN/Inf spellings Go's ParseFloat accepts but JSON does not.
func validJSONNumber(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("empty")
	}
	switch strings.ToLower(s) {
	case "nan", "inf", "+inf", "-inf", "infinity", "+infinity", "-infinity":
		return fmt.Errorf("%q is not finite", s)
	}
	var d json.Number
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		return fmt.Errorf("%q is not a JSON number", s)
	}
	return nil
}
