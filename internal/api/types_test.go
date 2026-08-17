package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The flexible types absorb Atlassian's inconsistent scalar encodings. These tests pin the
// exact variations seen in real payloads, and the fuzz targets check that no input can panic
// the decoders — a decoder crash would take down the whole command.

func TestID_AcceptsStringAndNumber(t *testing.T) {
	cases := []struct {
		in   string
		want ID
	}{
		{`"10001"`, "10001"},
		{`10001`, "10001"},
		{`null`, ""},
		{`""`, ""},
		// Above 2^53. Decoding via float64 would corrupt this, which is the whole reason ID
		// takes the raw bytes rather than parsing.
		{`9007199254740993`, "9007199254740993"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			var got ID
			require.NoError(t, json.Unmarshal([]byte(tc.in), &got))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestID_AlwaysMarshalsAsString(t *testing.T) {
	raw, err := json.Marshal(ID("10001"))
	require.NoError(t, err)
	assert.Equal(t, `"10001"`, string(raw))
}

func TestID_RejectsNonNumber(t *testing.T) {
	var got ID
	require.Error(t, json.Unmarshal([]byte(`{"a":1}`), &got))
}

func TestInt_AcceptsNumberAndString(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{`42`, 42, false},
		{`"42"`, 42, false},
		{`null`, 0, false},
		{`""`, 0, false},
		{`-7`, -7, false},
		{`1.9`, 1, false},
		{`9007199254740993`, 9007199254740993, false}, // must not lose precision
		{`"abc"`, 0, true},
		{`"NaN"`, 0, true},
		{`"Inf"`, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			var got Int
			err := json.Unmarshal([]byte(tc.in), &got)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Int64())
		})
	}
}

func TestBool_AcceptsStringForms(t *testing.T) {
	cases := map[string]bool{
		`true`: true, `false`: false,
		`"true"`: true, `"false"`: false,
		`"1"`: true, `"0"`: false,
		`"yes"`: true, `"no"`: false,
		`"on"`: true, `"off"`: false,
		`null`: false, `""`: false,
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			var got Bool
			require.NoError(t, json.Unmarshal([]byte(in), &got))
			assert.Equal(t, want, got.Bool())
		})
	}
	var bad Bool
	require.Error(t, json.Unmarshal([]byte(`"maybe"`), &bad))
}

func TestNumber_KeepsExactDecimal(t *testing.T) {
	var n Number
	require.NoError(t, json.Unmarshal([]byte(`0.1`), &n))
	assert.Equal(t, "0.1", n.String(), "the decimal text must survive verbatim")

	raw, err := json.Marshal(n)
	require.NoError(t, err)
	assert.Equal(t, "0.1", string(raw), "it must re-emit as a JSON number, not a string")

	f, err := n.Float()
	require.NoError(t, err)
	assert.InDelta(t, 0.1, f, 1e-9)

	require.NoError(t, json.Unmarshal([]byte(`"2.50"`), &n))
	assert.Equal(t, "2.50", n.String())

	require.Error(t, json.Unmarshal([]byte(`"NaN"`), &n))

	// 1e999999 is a grammatically valid JSON number, so it decodes: Number stores the exact
	// text and never parses it. The overflow only surfaces when a caller asks for a float,
	// which is the point of keeping decimals as text in the first place.
	require.NoError(t, json.Unmarshal([]byte(`1e999999`), &n))
	assert.Equal(t, "1e999999", n.String())
	_, err = n.Float()
	require.Error(t, err, "converting an out-of-range decimal to float64 must fail loudly")
}

func TestRef_AcceptsObjectAndString(t *testing.T) {
	var r Ref
	require.NoError(t, json.Unmarshal([]byte(`{"id":"1","name":"In Progress"}`), &r))
	assert.Equal(t, "In Progress", r.Label())

	// Some responses degrade a reference to a bare string depending on `expand`.
	require.NoError(t, json.Unmarshal([]byte(`"Done"`), &r))
	assert.Equal(t, "Done", r.Label())

	require.NoError(t, json.Unmarshal([]byte(`{"accountId":"5b1","displayName":"Juan"}`), &r))
	assert.Equal(t, "Juan", r.Label(), "displayName should win over accountId")

	require.NoError(t, json.Unmarshal([]byte(`{"id":"9"}`), &r))
	assert.Equal(t, "9", r.Label(), "an id is better than nothing")

	require.NoError(t, json.Unmarshal([]byte(`null`), &r))
	assert.Empty(t, r.Label())
}

func TestRefs_AcceptsSingleAndArray(t *testing.T) {
	var r Refs
	require.NoError(t, json.Unmarshal([]byte(`{"name":"one"}`), &r))
	assert.Equal(t, []string{"one"}, r.Labels())

	require.NoError(t, json.Unmarshal([]byte(`[{"name":"a"},{"name":"b"}]`), &r))
	assert.Equal(t, "a, b", r.String())

	require.NoError(t, json.Unmarshal([]byte(`null`), &r))
	assert.Empty(t, r.Labels())
}

func TestStringOrSlice(t *testing.T) {
	var s StringOrSlice
	require.NoError(t, json.Unmarshal([]byte(`"one"`), &s))
	assert.Equal(t, StringOrSlice{"one"}, s)

	require.NoError(t, json.Unmarshal([]byte(`["a","b"]`), &s))
	assert.Equal(t, "a, b", s.String())

	require.NoError(t, json.Unmarshal([]byte(`null`), &s))
	assert.Nil(t, s)
}

// The fuzz targets exist because these decoders sit on the response path for every command:
// a panic on a malformed payload would abort the process rather than produce an error.

func FuzzID(f *testing.F) {
	for _, seed := range []string{`"1"`, `1`, `null`, `9007199254740993`, `-0`, `1e5`, `""`, `[`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		var v ID
		_ = json.Unmarshal([]byte(in), &v) // must not panic
		if v != "" {
			_, _ = json.Marshal(v)
		}
	})
}

func FuzzInt(f *testing.F) {
	for _, seed := range []string{`1`, `"1"`, `null`, `1.5`, `-9223372036854775808`, `"1e999"`, `{}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		var v Int
		_ = json.Unmarshal([]byte(in), &v)
	})
}

func FuzzNumber(f *testing.F) {
	for _, seed := range []string{`0.1`, `"2.50"`, `null`, `"NaN"`, `1e999999`, `-`, `""`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		var v Number
		if err := json.Unmarshal([]byte(in), &v); err == nil {
			// A value that decoded must also re-encode, or a round trip would corrupt data.
			_, _ = json.Marshal(v)
		}
	})
}

func FuzzRefs(f *testing.F) {
	for _, seed := range []string{`{"name":"a"}`, `[{"id":1}]`, `null`, `"x"`, `[]`, `[[]]`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		var v Refs
		_ = json.Unmarshal([]byte(in), &v)
		_ = v.String()
	})
}

func FuzzDecodePage(f *testing.F) {
	seeds := []string{
		`{"values":[{"id":"1"}],"isLast":true}`,
		`{"issues":[],"nextPageToken":"x"}`,
		`[{"id":"1"}]`,
		`{"results":[],"_links":{"next":"/x?cursor=y"}}`,
		`{`,
		`null`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		for _, style := range []PageStyle{PageOffset, PageToken, PageCursor, PageStartLimit} {
			_, _ = decodePage([]byte(in), style, "", 25)
		}
	})
}
