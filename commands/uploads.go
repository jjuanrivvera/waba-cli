package commands

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("uploads", "Resumable uploads (template header media, profile pictures)", nil, uploadsSpecs)
}

// The resumable upload API is the one Graph surface that rejects "Bearer" — its calls send
// "Authorization: OAuth <token>" (DECISIONS.md #14), which the client swaps in when a
// request sets AuthScheme.

func uploadsSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "start <file>", Short: "Create an upload session for a file",
			Long:    "Starts a resumable upload session sized to the local file. Accepted types:\napplication/pdf, image/jpeg, image/jpg, image/png, video/mp4.",
			Example: `  waba uploads start ./header.png`,
			Kind:    kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("type", "", "MIME type (inferred from the extension when omitted)")
			},
			Columns: []string{"id"},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				app, err := o.appID(acct)
				if err != nil {
					return nil, err
				}
				data, err := readFileForFlag(args[0])
				if err != nil {
					return nil, err
				}
				mimeType := mustString(cmd, "type")
				if mimeType == "" {
					mimeType = mime.TypeByExtension(filepath.Ext(args[0]))
				}
				if mimeType == "" {
					return nil, fmt.Errorf("cannot infer the MIME type of %s — pass --type", args[0])
				}
				var out struct {
					ID string `json:"id"`
				}
				if err := c.DoInto(cmd.Context(), api.Request{Method: http.MethodPost, Path: app + "/uploads",
					Query: urlValues(
						"file_name", filepath.Base(args[0]),
						"file_length", strconv.Itoa(len(data)),
						"file_type", mimeType,
					)}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "session created — next: waba uploads upload %q %s", out.ID, args[0])
				return map[string]any{"id": out.ID}, nil
			},
		},
		{
			Use: "upload <session-id> <file>", Short: "Upload the file bytes, returning the media handle",
			Long: "Uploads the file into the session and prints the handle (`h`), which is what\ntemplate creation takes as header_handle and `waba profile update` as --picture-handle.\nIf a previous attempt was interrupted, `waba uploads status` tells the resume offset.",
			Example: `  waba uploads upload "upload:MTphdHRh..." ./header.png
  waba uploads upload "upload:MTphdHRh..." ./header.png --offset 1048576`,
			Kind: kindWrite, Args: cobra.ExactArgs(2),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().Int("offset", 0, "byte offset to resume from (from `waba uploads status`)")
			},
			Columns: []string{"h"},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				data, err := readFileForFlag(args[1])
				if err != nil {
					return nil, err
				}
				offset, _ := cmd.Flags().GetInt("offset")
				if offset < 0 || offset > len(data) {
					return nil, fmt.Errorf("--offset %d is outside the file (%d bytes)", offset, len(data))
				}
				h := http.Header{}
				h.Set("file_offset", strconv.Itoa(offset))
				var out struct {
					Handle string `json:"h"`
				}
				if err := c.DoInto(cmd.Context(), api.Request{Method: http.MethodPost, Path: args[0],
					Headers: h, AuthScheme: "OAuth", Body: data[offset:]}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "uploaded %d bytes", len(data)-offset)
				return map[string]any{"h": out.Handle}, nil
			},
		},
		{
			Use: "status <session-id>", Short: "Show how many bytes a session has received",
			Example: `  waba uploads status "upload:MTphdHRh..."`,
			Kind:    kindRead, Args: cobra.ExactArgs(1),
			Columns: []string{"id", "file_offset"},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				raw, err := c.Do(cmd.Context(), api.Request{Path: args[0], AuthScheme: "OAuth"})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
	}
}
