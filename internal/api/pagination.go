package api

import (
	"context"
	"encoding/json"
	"maps"
	"net/url"
)

// The Graph API paginates every list edge the same way: a data array plus a paging object
// with opaque cursors. The next page is requested by replaying the same query with
// after=<cursor>. The absolute paging.next URL is deliberately NOT followed verbatim — it
// would bypass the client's base-URL and version handling (and, in tests, the mock host).

// Page is one page of a Graph list response.
type Page struct {
	Data   json.RawMessage `json:"data"`
	Paging Paging          `json:"paging"`
}

// Paging carries the cursor block.
type Paging struct {
	Cursors struct {
		Before string `json:"before"`
		After  string `json:"after"`
	} `json:"cursors"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
}

// HasNext reports whether another page exists. Meta omits `next` on the last page even when
// an `after` cursor is still present, so `next` is the authoritative signal.
func (p Paging) HasNext() bool { return p.Next != "" }

// ListParams are the standard list controls every Graph edge accepts.
type ListParams struct {
	Limit  int    // per-page size; 0 leaves the server default
	After  string // resume cursor
	Before string // backward cursor
	Fields string // fields= projection
	All    bool   // walk every page
}

// Query renders the params for the first request.
func (p ListParams) Query() url.Values {
	q := url.Values{}
	if p.Limit > 0 {
		q.Set("limit", itoa(p.Limit))
	}
	if p.After != "" {
		q.Set("after", p.After)
	}
	if p.Before != "" {
		q.Set("before", p.Before)
	}
	if p.Fields != "" {
		q.Set("fields", p.Fields)
	}
	return q
}

// ListAll walks an edge, appending each page's raw items until the cursor runs out (or a
// single page when p.All is false). The items stay raw JSON so one walker serves every
// resource; callers decode into their own types.
func (c *Client) ListAll(ctx context.Context, path string, q url.Values, p ListParams) ([]json.RawMessage, error) {
	if q == nil {
		q = url.Values{}
	}
	maps.Copy(q, p.Query())

	var items []json.RawMessage
	for {
		var page struct {
			Data   []json.RawMessage `json:"data"`
			Paging Paging            `json:"paging"`
		}
		if err := c.GetJSON(ctx, path, q, &page); err != nil {
			return items, err
		}
		items = append(items, page.Data...)

		if !p.All || !page.Paging.HasNext() || page.Paging.Cursors.After == "" {
			return items, nil
		}
		// Replay the same query with the new cursor; drop `before` so the walk only moves
		// forward.
		q.Del("before")
		q.Set("after", page.Paging.Cursors.After)

		// A cancelled context must stop the walk between pages too, not only mid-request.
		if err := ctx.Err(); err != nil {
			return items, err
		}
	}
}

func itoa(n int) string {
	// strconv.Itoa behind a name the tests share; kept tiny and local.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
