package commands

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// applyJQ filters a result through a gojq expression.
//
// This is the escape hatch for everything the built-in --columns/--filter flags do not
// cover: the full Atlassian payload is far richer than any fixed column set, and piping to
// an external jq would force the user to choose -o json and lose table rendering.
func applyJQ(expr string, v any) (any, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid --jq expression: %w", err)
	}

	// Round-trip through JSON so gojq sees plain maps/slices rather than Go structs, and so
	// custom marshalers (the flexible types) have already been applied.
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode result for --jq: %w", err)
	}
	var input any
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("decode result for --jq: %w", err)
	}

	var out []any
	iter := query.Run(input)
	for {
		got, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := got.(error); isErr {
			return nil, fmt.Errorf("--jq: %w", err)
		}
		out = append(out, got)
	}

	// A single result is returned unwrapped so `--jq '.key'` prints a scalar rather than a
	// one-element array, which is what the equivalent jq invocation would do.
	if len(out) == 1 {
		return out[0], nil
	}
	return out, nil
}
