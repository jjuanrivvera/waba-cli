package commands

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("media", "Upload, inspect, download and delete media", nil, mediaSpecs)
}

var mediaColumns = []string{"id", "mime_type", "file_size", "url"}

func mediaSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "upload <file>", Short: "Upload a media file, returning its media id",
			Long: "Uploads a local file to the phone number's media store. The returned id is what\n`waba send image --id …` takes; uploaded media expires after 30 days.",
			Example: `  waba media upload ./catalogo.pdf
  waba media upload ./promo.jpg --type image/jpeg`,
			Kind: kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("type", "", "MIME type (inferred from the file extension when omitted)")
			},
			Columns: []string{"id"},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
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
				body, contentType, err := api.MultipartBody(
					map[string]string{"messaging_product": "whatsapp", "type": mimeType},
					"file", filepath.Base(args[0]), data, mimeType)
				if err != nil {
					return nil, err
				}
				var out struct {
					ID api.ID `json:"id"`
				}
				if err := c.PostMultipart(cmd.Context(), phone+"/media", body, contentType, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "uploaded %s (%d bytes, %s)", args[0], len(data), mimeType)
				return map[string]any{"id": out.ID.String()}, nil
			},
		},
		{
			Use: "url <media-id>", Short: "Get a media item's download URL and metadata",
			Long:    "The returned URL is valid for only 5 minutes — `waba media download` fetches it\nimmediately, which is almost always what you want.",
			Example: `  waba media url 1013859600285441`,
			Kind:    kindRead, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("phone", "", "restrict the lookup to this phone number id")
			},
			Columns: mediaColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				var info api.MediaInfo
				q := urlValues()
				if p := mustString(cmd, "phone"); p != "" {
					q.Set("phone_number_id", p)
				}
				if err := c.GetJSON(cmd.Context(), args[0], q, &info); err != nil {
					return nil, err
				}
				return info, nil
			},
		},
		{
			Use: "download <media-id>", Short: "Download a media item to a file",
			Example: `  waba media download 1013859600285441 -f voice-note.ogg`,
			Kind:    kindRead, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().StringP("file", "f", "", "output path (defaults to the media id)")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				var info api.MediaInfo
				if err := c.GetJSON(cmd.Context(), args[0], nil, &info); err != nil {
					return nil, err
				}
				if info.URL == "" {
					return nil, fmt.Errorf("no download URL returned for media %s", args[0])
				}
				// The lookaside URL still requires the bearer token; the client restricts
				// which hosts may receive it (DECISIONS.md #13).
				data, err := c.Do(cmd.Context(), api.Request{Path: info.URL})
				if err != nil {
					return nil, err
				}
				out := mustString(cmd, "file")
				if out == "" {
					out = args[0]
					if exts, _ := mime.ExtensionsByType(info.MimeType); len(exts) > 0 {
						out += exts[0]
					}
				}
				if err := os.WriteFile(out, data, 0o600); err != nil {
					return nil, err
				}
				o.note(cmd.ErrOrStderr(), "wrote %s (%d bytes, %s)", out, len(data), info.MimeType)
				return nil, nil
			},
		},
		{
			Use: "delete <media-id>", Short: "Delete an uploaded media item",
			Kind: kindDestructive, Args: cobra.ExactArgs(1),
			Confirm: "Delete media %s?",
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("phone", "", "restrict the deletion to this phone number id")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				q := urlValues()
				if p := mustString(cmd, "phone"); p != "" {
					q.Set("phone_number_id", p)
				}
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "DELETE", Path: args[0], Query: q}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "deleted media %s", args[0])
				return nil, nil
			},
		},
	}
}
