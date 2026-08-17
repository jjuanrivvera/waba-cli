package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func render(t *testing.T, format string, v any, configure func(*Renderer)) (string, string) {
	t.Helper()
	var out, errBuf bytes.Buffer
	r := New(&out, &errBuf, format)
	if configure != nil {
		configure(r)
	}
	require.NoError(t, r.Render(v))
	return out.String(), errBuf.String()
}

func TestRender_AllFourFormats(t *testing.T) {
	rows := []map[string]any{
		{"id": "1", "name": "alpha", "active": true},
		{"id": "2", "name": "beta", "active": false},
	}

	t.Run("json", func(t *testing.T) {
		got, _ := render(t, FormatJSON, rows, nil)
		assert.Contains(t, got, `"name": "alpha"`)
	})

	t.Run("yaml", func(t *testing.T) {
		got, _ := render(t, FormatYAML, rows, nil)
		var back []map[string]any
		require.NoError(t, yaml.Unmarshal([]byte(got), &back))
		assert.Len(t, back, 2)
		assert.Equal(t, "alpha", back[0]["name"])
	})

	t.Run("csv", func(t *testing.T) {
		got, _ := render(t, FormatCSV, rows, func(r *Renderer) {
			r.Preferred = []string{"id", "name", "active"}
		})
		records, err := csv.NewReader(strings.NewReader(got)).ReadAll()
		require.NoError(t, err)
		assert.Equal(t, []string{"id", "name", "active"}, records[0])
		assert.Equal(t, []string{"1", "alpha", "true"}, records[1])
	})

	t.Run("table", func(t *testing.T) {
		got, _ := render(t, FormatTable, rows, func(r *Renderer) { r.Preferred = []string{"id", "name"} })
		assert.Contains(t, got, "ID")
		assert.Contains(t, got, "alpha")
		assert.Contains(t, got, "beta")
	})

	t.Run("id", func(t *testing.T) {
		got, _ := render(t, FormatID, rows, nil)
		assert.Equal(t, "1\n2\n", got, "-o id must be one bare id per line for xargs")
	})
}

func TestRender_IDFallsBackToKey(t *testing.T) {
	// Jira issues and projects are identified by key, not id.
	rows := []map[string]any{{"key": "PP-1"}, {"key": "PP-2"}}
	got, _ := render(t, FormatID, rows, nil)
	assert.Equal(t, "PP-1\nPP-2\n", got)
}

func TestRender_ColumnOrderIsDeterministic(t *testing.T) {
	// Go randomizes map iteration, so without an explicit order the same command would print
	// its columns in a different order on every run.
	row := map[string]any{"zebra": 1, "alpha": 2, "id": 3, "name": 4}

	t.Run("preferred columns limit and order the view", func(t *testing.T) {
		var first string
		for i := range 20 {
			got, _ := render(t, FormatCSV, []map[string]any{row}, func(r *Renderer) {
				r.Preferred = []string{"id", "name"}
			})
			header := strings.SplitN(got, "\n", 2)[0]
			if i == 0 {
				first = header
				// Exactly the declared columns: a curated view is not "these first, then
				// everything else too".
				assert.Equal(t, "id,name", header)
			}
			assert.Equal(t, first, header, "column order must be stable across runs")
		}
	})

	t.Run("automatic selection sorts alphabetically", func(t *testing.T) {
		var first string
		for i := range 20 {
			got, _ := render(t, FormatCSV, []map[string]any{row}, nil)
			header := strings.SplitN(got, "\n", 2)[0]
			if i == 0 {
				first = header
				assert.Equal(t, "alpha,id,name,zebra", header)
			}
			assert.Equal(t, first, header, "column order must be stable across runs")
		}
	})
}

func TestRender_ExplicitColumns(t *testing.T) {
	rows := []map[string]any{{"id": "1", "name": "alpha", "extra": "x"}}
	got, _ := render(t, FormatCSV, rows, func(r *Renderer) { r.Columns = []string{"name", "id"} })
	assert.True(t, strings.HasPrefix(got, "name,id\n"))
	assert.NotContains(t, got, "extra")
}

func TestCSV_FormulaInjectionIsNeutralized(t *testing.T) {
	// An issue summary is attacker-controllable text. A spreadsheet executes a cell that
	// starts with = + - or @, so exporting issues to CSV would otherwise run whatever
	// someone put in a ticket title (CWE-1236).
	cases := []struct {
		in   string
		want string
	}{
		{"=1+1", "'=1+1"},
		{"+SUM(A1)", "'+SUM(A1)"},
		{"@import", "'@import"},
		{"-cmd|'/c calc'", "'-cmd|'/c calc'"},
		{"=HYPERLINK(\"http://evil\")", "'=HYPERLINK(\"http://evil\")"},
		// A genuine negative number must stay a number.
		{"-5", "-5"},
		{"-1.5", "-1.5"},
		{"normal", "normal"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, sanitizeCSV(tc.in))
		})
	}
}

func TestCSV_InjectionGuardAppliesThroughTheRenderer(t *testing.T) {
	rows := []map[string]any{{"summary": "=cmd|'/c calc'!A1"}}
	got, _ := render(t, FormatCSV, rows, nil)
	records, err := csv.NewReader(strings.NewReader(got)).ReadAll()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(records[1][0], "'"), "the cell must be neutralized end to end")
}

func TestTable_SanitizesTerminalEscapes(t *testing.T) {
	// A page title or issue summary can contain an escape sequence; rendering it raw lets
	// remote content rewrite the terminal.
	rows := []map[string]any{{"title": "safe\x1b[2Jwiped\x07"}}
	got, _ := render(t, FormatTable, rows, nil)
	assert.NotContains(t, got, "\x1b")
	assert.NotContains(t, got, "\x07")
	assert.Contains(t, got, "safe")
}

func TestTable_TruncatesWideCellsAndSaysSo(t *testing.T) {
	rows := []map[string]any{{"id": "1", "body": strings.Repeat("x", 200)}}
	out, errOut := render(t, FormatTable, rows, nil)
	assert.Contains(t, out, "…")
	assert.Contains(t, errOut, "-o json", "the hint must go to stderr so stdout stays pipe-clean")
}

func TestTable_RuneAwareTruncation(t *testing.T) {
	// Truncating by bytes would split a multi-byte character and emit invalid UTF-8.
	rows := []map[string]any{{"title": strings.Repeat("日", 100)}}
	got, _ := render(t, FormatTable, rows, nil)
	assert.True(t, strings.Contains(got, "日"))
	assert.NotContains(t, got, "�", "no replacement characters from a split rune")
}

func TestTable_CapsAutoColumnsAndNotesIt(t *testing.T) {
	row := map[string]any{}
	for i := range 20 {
		row[string(rune('a'+i))+"col"] = i
	}
	_, errOut := render(t, FormatTable, []map[string]any{row, row}, nil)
	assert.Contains(t, errOut, "columns")
}

func TestTable_SingleObjectRendersAsKeyValue(t *testing.T) {
	// One wide row is unreadable; a key/value list is not.
	got, _ := render(t, FormatTable, map[string]any{"id": "1", "name": "alpha"}, nil)
	assert.Contains(t, got, "id")
	assert.Contains(t, got, "alpha")
	assert.NotContains(t, got, "ID  ", "a single object should not get a table header")
}

func TestTable_EmptyResultNotesOnStderr(t *testing.T) {
	out, errOut := render(t, FormatTable, []map[string]any{}, nil)
	assert.Empty(t, out)
	assert.Contains(t, errOut, "no results")
}

func TestRender_QuietSuppressesNotes(t *testing.T) {
	var out, errBuf bytes.Buffer
	r := New(&out, &errBuf, FormatTable)
	r.Quiet = true
	require.NoError(t, r.Render([]map[string]any{}))
	assert.Empty(t, errBuf.String())
}

func TestFlatten_CollapsesAtlassianReferences(t *testing.T) {
	// Jira nests {id,name} objects everywhere; a table cell should read "In Progress".
	rows := []map[string]any{{
		"key":    "PP-1",
		"status": map[string]any{"id": "3", "name": "In Progress"},
		"author": map[string]any{"accountId": "5b1", "displayName": "Juan"},
	}}
	got, _ := render(t, FormatCSV, rows, func(r *Renderer) {
		r.Columns = []string{"key", "status", "author"}
	})
	assert.Contains(t, got, "In Progress")
	assert.Contains(t, got, "Juan")
}

func TestFlatten_ExposesNestedScalars(t *testing.T) {
	rows := []map[string]any{{"version": map[string]any{"number": 3, "message": "edit"}}}
	got, _ := render(t, FormatCSV, rows, func(r *Renderer) { r.Columns = []string{"version.number"} })
	assert.Contains(t, got, "version.number")
	assert.Contains(t, got, "3")
}

func TestScalar_Formatting(t *testing.T) {
	assert.Equal(t, "", scalar(nil))
	assert.Equal(t, "text", scalar("text"))
	assert.Equal(t, "true", scalar(true))
	assert.Equal(t, "a, b", scalar([]any{"a", "b"}))
	assert.Equal(t, "In Progress", scalar(map[string]any{"name": "In Progress"}))
	assert.Equal(t, "one, two", scalar([]any{
		map[string]any{"name": "one"},
		map[string]any{"name": "two"},
	}))
}

func TestRenderRaw_PassesServerJSONThrough(t *testing.T) {
	// Large ids must not be degraded through float64 by a needless re-marshal.
	var out, errBuf bytes.Buffer
	r := New(&out, &errBuf, FormatJSON)
	require.NoError(t, r.RenderRaw([]byte(`{"id":9007199254740993,"name":"x"}`)))
	assert.Contains(t, out.String(), "9007199254740993")
}

func TestValidateFormat(t *testing.T) {
	for _, f := range Formats {
		require.NoError(t, ValidateFormat(f))
	}
	err := ValidateFormat("xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "table", "the error should list the valid formats")
}

func TestColorEnabled_HonorsNoColorAndTTY(t *testing.T) {
	var buf bytes.Buffer
	// A buffer is not a terminal, so colour is off regardless.
	assert.False(t, ColorEnabled(&buf, false))
	assert.False(t, ColorEnabled(&buf, true))

	t.Setenv("NO_COLOR", "1")
	assert.False(t, ColorEnabled(&buf, false), "NO_COLOR must always win")
}

func TestRender_UnknownFormatErrors(t *testing.T) {
	var out, errBuf bytes.Buffer
	r := New(&out, &errBuf, "toml")
	require.Error(t, r.Render([]map[string]any{{"a": 1}}))
}

func TestPreferredColumnsLimitTheView(t *testing.T) {
	// A resource that declares its columns has curated a view. Treating Preferred as a mere
	// sort order produced ten-column walls padded with `self` URLs and internal ids, plus a
	// truncation note on nearly every command.
	rows := []map[string]any{{
		"key": "PP-1", "fields.summary": "s", "fields.status": "Open",
		"self": "https://x/rest/api/3/issue/1", "id": "10001", "expand": "renderedFields",
	}}
	got, _ := render(t, FormatTable, rows, func(r *Renderer) {
		r.Preferred = []string{"key", "fields.summary", "fields.status"}
	})
	assert.Contains(t, got, "KEY")
	assert.Contains(t, got, "SUMMARY")
	assert.NotContains(t, got, "SELF", "a curated view must not be padded with internals")
	assert.NotContains(t, got, "EXPAND")
}

func TestPreferredFallsBackWhenAbsent(t *testing.T) {
	// A partial --fields can leave none of the declared columns present; printing an empty
	// table would be worse than falling back to what the payload actually has.
	rows := []map[string]any{{"id": "1", "other": "x"}}
	got, _ := render(t, FormatTable, rows, func(r *Renderer) {
		r.Preferred = []string{"key", "fields.summary"}
	})
	assert.Contains(t, got, "ID")
}

func TestHeaderLabel(t *testing.T) {
	cases := map[string]string{
		"key":                 "KEY",
		"fields.summary":      "SUMMARY",
		"fields.status":       "STATUS",
		"location.projectKey": "PROJECT KEY",
		"projectTypeKey":      "PROJECT TYPE KEY",
		"version.number":      "NUMBER",
	}
	for in, want := range cases {
		assert.Equalf(t, want, headerLabel(in), "headerLabel(%q)", in)
	}
}

func TestHumanTime(t *testing.T) {
	// Milliseconds and a numeric offset are never what someone scanning a table needs, and
	// they cost 16 characters that push a real column into truncation.
	got := humanTime("2026-07-24T16:35:15.911-0400")
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$`, got)

	// Anything that is not a timestamp passes through untouched.
	for _, s := range []string{"", "PP-1071", "In Progress", "2026", "not-a-date-at-all"} {
		assert.Equal(t, s, humanTime(s))
	}
}

func TestScalarJQResultPrintsBare(t *testing.T) {
	// `--jq length` in a shell substitution must yield 112, not a table headed VALUE.
	got, _ := render(t, FormatTable, json.Number("112"), nil)
	assert.Equal(t, "112\n", got)

	got, _ = render(t, FormatTable, "PP-1071", nil)
	assert.Equal(t, "PP-1071\n", got)
}

func TestTableTrimsCellWhitespace(t *testing.T) {
	// Real Jira summaries carry stray leading spaces that misalign every other row.
	got, _ := render(t, FormatTable, []map[string]any{
		{"key": "PP-1", "summary": "  leading space"},
		{"key": "PP-2", "summary": "normal"},
	}, func(r *Renderer) { r.Preferred = []string{"key", "summary"} })
	// Two spaces are the column separator; a surviving stray space would make three.
	assert.NotContains(t, got, "   leading space")
	assert.Contains(t, got, "  leading space")
	// Both rows must start their summary at the same column.
	lines := strings.Split(strings.TrimSpace(got), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, strings.Index(lines[1], "leading"), strings.Index(lines[2], "normal"))
}
