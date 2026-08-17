// Package output renders API results in the four supported formats.
//
// Everything goes through one renderer driven by JSON normalization, so a new resource gets
// table/json/yaml/csv output for free and all four stay consistent. Notes and warnings go to
// stderr so stdout stays pipe-clean.
package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// Supported formats.
const (
	FormatTable = "table"
	FormatJSON  = "json"
	FormatYAML  = "yaml"
	FormatCSV   = "csv"
	FormatID    = "id"
)

// Formats lists every valid -o value, for validation and completion.
var Formats = []string{FormatTable, FormatJSON, FormatYAML, FormatCSV, FormatID}

// maxAutoColumns caps how many columns are chosen automatically. Atlassian objects can carry
// 60+ fields; rendering all of them produces an unreadable wall, so the table is capped and a
// note points at -o json for the rest.
const maxAutoColumns = 10

// maxCellWidth truncates a table cell. Long ADF descriptions would otherwise destroy the
// column alignment for every other row.
const maxCellWidth = 60

// Renderer writes results.
type Renderer struct {
	Out    io.Writer
	Err    io.Writer
	Format string

	// Columns restricts and orders table/csv columns. Empty means automatic selection.
	Columns []string
	// Preferred lists the fields a resource wants first when columns are automatic.
	Preferred []string
	// NoColor disables ANSI styling regardless of TTY detection.
	NoColor bool
	// Quiet suppresses stderr notes.
	Quiet bool
	// IDField names the field `-o id` prints. Defaults to "id".
	IDField string
}

// New builds a renderer with sensible defaults.
func New(out, errW io.Writer, format string) *Renderer {
	return &Renderer{Out: out, Err: errW, Format: format, IDField: "id"}
}

// ValidateFormat reports whether f is a supported format.
func ValidateFormat(f string) error {
	for _, v := range Formats {
		if f == v {
			return nil
		}
	}
	return fmt.Errorf("unknown output format %q (want one of: %s)", f, strings.Join(Formats, ", "))
}

// Render writes v in the configured format. v may be any JSON-encodable value: a slice for
// lists, a map or struct for single items.
func (r *Renderer) Render(v any) error {
	switch r.Format {
	case FormatJSON, "":
		return r.renderJSON(v)
	case FormatYAML:
		return r.renderYAML(v)
	case FormatCSV:
		return r.renderCSV(v)
	case FormatID:
		return r.renderIDs(v)
	case FormatTable:
		return r.renderTable(v)
	default:
		return ValidateFormat(r.Format)
	}
}

// RenderRaw writes a pre-encoded JSON payload straight through for `-o json`, so the API's
// own response is never re-marshalled (which would reorder keys and lose number precision).
func (r *Renderer) RenderRaw(raw []byte) error {
	if r.Format == FormatJSON || r.Format == "" {
		// Indent the original bytes rather than decoding and re-encoding them. Decoding into
		// `any` turns every number into a float64, which silently rounds ids above 2^53 —
		// and Jira issue and Confluence content ids on large instances exceed that.
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, raw, "", "  "); err != nil {
			// Not valid JSON (a plain-text or HTML body); emit it verbatim.
			_, werr := r.Out.Write(append(raw, '\n'))
			return werr
		}
		pretty.WriteByte('\n')
		_, err := r.Out.Write(pretty.Bytes())
		return err
	}

	// The other formats need a decoded value. UseNumber keeps the digits intact here too.
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return r.Render(v)
}

func (r *Renderer) renderJSON(v any) error {
	enc := json.NewEncoder(r.Out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func (r *Renderer) renderYAML(v any) error {
	// Round-trip through JSON first so struct json tags (not Go field names) drive the keys.
	normalized, err := normalize(v)
	if err != nil {
		return err
	}
	enc := yaml.NewEncoder(r.Out)
	enc.SetIndent(2)
	if err := enc.Encode(normalized); err != nil {
		return err
	}
	return enc.Close()
}

func (r *Renderer) renderIDs(v any) error {
	rows, _, err := rowsOf(v)
	if err != nil {
		return err
	}
	field := r.IDField
	if field == "" {
		field = "id"
	}
	for _, row := range rows {
		val := row[field]
		// Fall back to the OTHER identifier, never the same one again: a resource whose
		// IDField is "key" would otherwise retry "key", find nothing, and silently print an
		// empty list — which reads identically to "no results".
		if val == nil {
			for _, alt := range []string{"key", "id"} {
				if alt == field {
					continue
				}
				if v := row[alt]; v != nil {
					val = v
					break
				}
			}
		}
		if val == nil {
			continue
		}
		fmt.Fprintln(r.Out, scalar(val))
	}
	return nil
}

func (r *Renderer) renderCSV(v any) error {
	rows, single, err := rowsOf(v)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	cols := r.chooseColumns(rows, single)

	w := csv.NewWriter(r.Out)
	if err := w.Write(cols); err != nil {
		return err
	}
	for _, row := range rows {
		rec := make([]string, len(cols))
		for i, c := range cols {
			rec[i] = sanitizeCSV(scalar(row[c]))
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func (r *Renderer) renderTable(v any) error {
	// A scalar result (`--jq length`, `--jq .key`) is a value, not a table. Wrapping it in a
	// one-cell table headed VALUE makes it unusable in a shell substitution.
	if s, ok := scalarValue(v); ok {
		fmt.Fprintln(r.Out, s)
		return nil
	}

	rows, single, err := rowsOf(v)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		r.note("no results")
		return nil
	}

	// A single object reads far better as a key/value list than as a one-row wide table.
	if single && len(r.Columns) == 0 {
		return r.renderKeyValue(rows[0])
	}

	cols := r.chooseColumns(rows, single)
	cells := make([][]string, 0, len(rows)+1)
	header := make([]string, len(cols))
	for i, c := range cols {
		header[i] = headerLabel(c)
	}
	cells = append(cells, header)

	truncated := false
	for _, row := range rows {
		rec := make([]string, len(cols))
		for i, c := range cols {
			// Timestamps are shortened for the eye only. CSV and JSON keep the exact value,
			// because those are what a script parses.
			s := humanTime(sanitizeTerminal(scalar(row[c])))
			if t, cut := truncateRunes(s, maxCellWidth); cut {
				truncated = true
				s = t
			}
			rec[i] = s
		}
		cells = append(cells, rec)
	}

	widths := make([]int, len(cols))
	for _, rec := range cells {
		for i, s := range rec {
			if w := utf8.RuneCountInString(s); w > widths[i] {
				widths[i] = w
			}
		}
	}

	for _, rec := range cells {
		var b strings.Builder
		for i, s := range rec {
			b.WriteString(s)
			if i < len(rec)-1 {
				b.WriteString(strings.Repeat(" ", widths[i]-utf8.RuneCountInString(s)+2))
			}
		}
		fmt.Fprintln(r.Out, strings.TrimRight(b.String(), " "))
	}

	if truncated {
		r.note("some cells were truncated — use -o json for full values")
	}
	return nil
}

// scalarValue reports whether v is a bare scalar and renders it.
func scalarValue(v any) (string, bool) {
	switch t := v.(type) {
	case nil:
		return "", false
	case string:
		return t, true
	case bool:
		return strconv.FormatBool(t), true
	case json.Number:
		return t.String(), true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case int, int64:
		return fmt.Sprint(t), true
	}
	return "", false
}

// humanTime shortens a full timestamp for table display.
//
// Atlassian returns "2026-07-24T16:35:15.911-0400". The milliseconds and offset are never
// what someone scanning a table needs, and they cost 16 characters that then push a real
// column into truncation.
func humanTime(v string) string {
	if len(v) < 19 || v[4] != '-' || v[7] != '-' || v[10] != 'T' {
		return v
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05-0700",
		time.RFC3339Nano,
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.Local().Format("2006-01-02 15:04")
		}
	}
	return v
}

// headerLabel turns a lookup path into a column heading.
//
// The path is how the value is addressed (`fields.summary`), which is not how a person reads
// a table. The container prefix carries no information — every column in an issue table is
// under `fields` — so it is dropped, and the remainder is spaced out.
func headerLabel(col string) string {
	label := col
	for _, prefix := range []string{"fields.", "location.", "content.", "version.", "currentStatus."} {
		if strings.HasPrefix(label, prefix) {
			label = strings.TrimPrefix(label, prefix)
			break
		}
	}
	label = strings.ReplaceAll(label, ".", " ")
	// camelCase -> spaced words, so `projectTypeKey` reads as PROJECT TYPE KEY.
	var b strings.Builder
	for i, r := range label {
		if i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(rune(label[i-1])) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(b.String())
}

func (r *Renderer) renderKeyValue(row map[string]any) error {
	keys := r.orderKeys(collectKeys([]map[string]any{row}), false)
	width := 0
	for _, k := range keys {
		if len(k) > width {
			width = len(k)
		}
	}
	for _, k := range keys {
		v := scalar(row[k])
		if v == "" {
			continue
		}
		fmt.Fprintf(r.Out, "%-*s  %s\n", width, k, sanitizeTerminal(v))
	}
	return nil
}

// chooseColumns resolves --columns, else the resource's curated set, else picks automatically.
func (r *Renderer) chooseColumns(rows []map[string]any, single bool) []string {
	if len(r.Columns) > 0 {
		return r.Columns
	}

	present := map[string]bool{}
	for _, k := range collectKeys(rows) {
		present[k] = true
	}

	// A resource that declares Preferred columns has curated a view; show exactly that, not
	// that plus every other key in the payload. Treating Preferred as a mere sort order is
	// what produced ten-column walls padded with `self` URLs and internal ids, and a
	// truncation note on almost every command.
	if len(r.Preferred) > 0 {
		curated := make([]string, 0, len(r.Preferred))
		for _, p := range r.Preferred {
			if present[p] {
				curated = append(curated, p)
			}
		}
		if len(curated) > 0 {
			return curated
		}
		// The declared columns are absent from this payload (a partial `--fields`, say), so
		// fall through to automatic selection rather than printing an empty table.
	}

	keys := r.orderKeys(collectKeys(rows), true)
	if !single && len(keys) > maxAutoColumns {
		r.note(fmt.Sprintf("showing %d of %d columns — use --columns or -o json for the rest",
			maxAutoColumns, len(keys)))
		keys = keys[:maxAutoColumns]
	}
	return keys
}

// orderKeys produces a deterministic column order: the resource's preferred fields first, in
// their declared order, then everything else alphabetically.
//
// Map iteration order in Go is randomized, so without this the same command would print its
// columns in a different order on every run.
func (r *Renderer) orderKeys(keys []string, dropEmpty bool) []string {
	present := make(map[string]bool, len(keys))
	for _, k := range keys {
		present[k] = true
	}

	out := make([]string, 0, len(keys))
	used := make(map[string]bool, len(keys))
	for _, p := range r.Preferred {
		if present[p] && !used[p] {
			out = append(out, p)
			used[p] = true
		}
	}

	rest := make([]string, 0, len(keys))
	for _, k := range keys {
		if !used[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	_ = dropEmpty
	return append(out, rest...)
}

func collectKeys(rows []map[string]any) []string {
	seen := map[string]bool{}
	var out []string
	for _, row := range rows {
		for k := range row {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}

// rowsOf normalizes any value into a list of flat rows. The bool reports whether the input
// was a single object rather than a collection.
func rowsOf(v any) ([]map[string]any, bool, error) {
	n, err := normalize(v)
	if err != nil {
		return nil, false, err
	}
	switch t := n.(type) {
	case []any:
		rows := make([]map[string]any, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				rows = append(rows, flatten(m))
			} else {
				rows = append(rows, map[string]any{"value": item})
			}
		}
		return rows, false, nil
	case map[string]any:
		return []map[string]any{flatten(t)}, true, nil
	case nil:
		return nil, true, nil
	default:
		return []map[string]any{{"value": t}}, true, nil
	}
}

// flatten lifts one level of nesting so a table can show `status.name` rather than a Go map
// printed as `map[...]`. It stops at one level deliberately: deeper flattening produces
// dozens of columns nobody asked for.
func flatten(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		switch t := v.(type) {
		case map[string]any:
			if label, ok := refLabel(t); ok {
				out[k] = label
				continue
			}
			for sk, sv := range t {
				if nested, isMap := sv.(map[string]any); isMap {
					// A second-level reference object is where Jira keeps the values people
					// actually want in a table: issue.fields.status, .assignee, .priority.
					// Skipping every nested map would drop all of them and leave a table of
					// just summaries and timestamps.
					if label, ok := refLabel(nested); ok {
						out[k+"."+sk] = label
					}
					continue
				}
				if _, arr := sv.([]any); arr {
					continue
				}
				out[k+"."+sk] = sv
			}
		default:
			out[k] = v
		}
	}
	return out
}

// refLabel collapses Atlassian's ubiquitous `{id,name,...}` reference objects to their most
// readable field, so a status cell reads "In Progress".
func refLabel(m map[string]any) (string, bool) {
	for _, key := range []string{"displayName", "name", "key", "value"} {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s, true
			}
		}
	}
	return "", false
}

// normalize round-trips through JSON so struct tags, custom marshalers and flexible types
// all resolve to plain maps/slices before rendering.
func normalize(v any) (any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode result: %w", err)
	}
	var out any
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber() // preserve large ids and decimals instead of degrading them to float64
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("decode result: %w", err)
	}
	return out, nil
}

// scalar formats a value for a table or CSV cell.
func scalar(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				if label, ok := refLabel(m); ok {
					parts = append(parts, label)
					continue
				}
			}
			parts = append(parts, scalar(item))
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		if label, ok := refLabel(t); ok {
			return label
		}
		raw, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return string(raw)
	default:
		return fmt.Sprint(t)
	}
}

// sanitizeCSV neutralizes spreadsheet formula injection (CWE-1236).
//
// A Jira summary is user-supplied text, and a spreadsheet treats a cell starting with = + -
// or @ as a formula — so exporting issues to CSV and opening it in Excel would execute
// whatever an attacker put in a ticket title. Prefixing with a single quote makes the cell
// literal text. A genuine negative number is left alone.
func sanitizeCSV(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '@', '\t', '\r':
		return "'" + s
	case '-':
		// "-5" and "-1.5" are numbers, not formulas; "-cmd|..." is not.
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			return s
		}
		return "'" + s
	}
	return s
}

// sanitizeTerminal strips control characters and ANSI escapes from API-supplied text.
//
// Issue summaries and page titles are attacker-controllable; an embedded escape sequence
// could rewrite the terminal, hide output, or in some emulators inject input.
func sanitizeTerminal(s string) string {
	if !strings.ContainsFunc(s, isControl) {
		// Real summaries carry stray leading/trailing spaces, which misalign the column for
		// every other row.
		return strings.TrimSpace(s)
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(' ')
		case isControl(r):
			// dropped
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func isControl(r rune) bool {
	return r == 0x1b || (unicode.IsControl(r) && r != '\n' && r != '\t')
}

// truncateRunes shortens s to n runes, reporting whether it cut. Rune-aware so multi-byte
// text is never split mid-character.
func truncateRunes(s string, n int) (string, bool) {
	if utf8.RuneCountInString(s) <= n {
		return s, false
	}
	runes := []rune(s)
	return string(runes[:n-1]) + "…", true
}

// note writes an advisory to stderr. Never stdout: stdout must stay machine-parseable.
func (r *Renderer) note(msg string) {
	if r.Quiet || r.Err == nil {
		return
	}
	fmt.Fprintln(r.Err, "note: "+msg)
}

// IsTTY reports whether w is an interactive terminal, which gates colour.
func IsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// ColorEnabled resolves whether to emit ANSI colour: never when NO_COLOR is set (the de-facto
// standard), never when --no-color was passed, and only on a real terminal.
func ColorEnabled(w io.Writer, noColorFlag bool) bool {
	if noColorFlag {
		return false
	}
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	return IsTTY(w)
}
