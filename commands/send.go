package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("send", "Send WhatsApp messages", nil, sendSpecs)
}

// flatSend reduces a SendResult to the row a human actually wants after sending.
func flatSend(res *api.SendResult) any {
	row := map[string]any{}
	if len(res.Messages) > 0 {
		row["message_id"] = res.Messages[0].ID.String()
		if res.Messages[0].MessageStatus != "" {
			row["status"] = res.Messages[0].MessageStatus
		}
	}
	if len(res.Contacts) > 0 {
		row["wa_id"] = res.Contacts[0].WaID
	}
	return row
}

var sendColumns = []string{"message_id", "wa_id", "status"}

// sendRun wraps the common send flow: resolve phone id, post the payload, flatten.
func sendRun(build func(cmd *cobra.Command, args []string) (msgType string, content any, err error)) func(*cobra.Command, *globalOptions, *api.Client, *config.Account, []string) (any, error) {
	return func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
		phone, err := o.phoneNumberID(acct)
		if err != nil {
			return nil, err
		}
		to, _ := cmd.Flags().GetString("to")
		msgType, content, err := build(cmd, args)
		if err != nil {
			return nil, err
		}
		payload := api.MessagePayload(to, msgType, content)
		if replyTo, _ := cmd.Flags().GetString("reply-to"); replyTo != "" {
			payload["context"] = map[string]string{"message_id": replyTo}
		}
		res, err := c.SendMessage(cmd.Context(), phone, payload)
		if err != nil {
			return nil, err
		}
		o.noteWrite(cmd.ErrOrStderr(), "sent %s message to %s", msgType, to)
		return flatSend(res), nil
	}
}

// sendFlags wires the flags every send verb shares.
func sendFlags(cmd *cobra.Command) {
	cmd.Flags().String("to", "", "recipient phone number in E.164 digits (e.g. 573001112233)")
	_ = cmd.MarkFlagRequired("to")
	cmd.Flags().String("reply-to", "", "wamid of the message this one replies to")
}

// mediaContent reads the shared media flags into a MediaRef.
func mediaContent(cmd *cobra.Command, allowCaption, allowFilename bool) (*api.MediaRef, error) {
	id, _ := cmd.Flags().GetString("id")
	link, _ := cmd.Flags().GetString("link")
	if (id == "") == (link == "") {
		return nil, fmt.Errorf("exactly one of --id (uploaded media) or --link (hosted URL) is required")
	}
	ref := &api.MediaRef{ID: id, Link: link}
	if allowCaption {
		ref.Caption, _ = cmd.Flags().GetString("caption")
	}
	if allowFilename {
		ref.Filename, _ = cmd.Flags().GetString("filename")
	}
	return ref, nil
}

func mediaFlags(cmd *cobra.Command, caption, filename bool) {
	cmd.Flags().String("id", "", "media id from `waba media upload`")
	cmd.Flags().String("link", "", "public https URL of the media")
	if caption {
		cmd.Flags().String("caption", "", "caption shown under the media")
	}
	if filename {
		cmd.Flags().String("filename", "", "filename shown to the recipient")
	}
}

func sendSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "text <body>", Short: "Send a text message", Kind: kindWrite,
			Example: `  waba send text --to 573001112233 "Your order shipped!"
  waba send text --to 573001112233 --preview-url "Check https://example.com"`,
			Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().Bool("preview-url", false, "render a link preview for the first URL")
			},
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, args []string) (string, any, error) {
				preview, _ := cmd.Flags().GetBool("preview-url")
				return "text", map[string]any{"body": args[0], "preview_url": preview}, nil
			}),
		},
		{
			Use: "image", Short: "Send an image (JPEG/PNG)", Kind: kindWrite,
			Example: `  waba send image --to 573001112233 --link https://example.com/cat.jpg --caption "gato"
  waba send image --to 573001112233 --id 1013859600285441`,
			Args:    cobra.NoArgs,
			Flags:   func(cmd *cobra.Command) { sendFlags(cmd); mediaFlags(cmd, true, false) },
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, _ []string) (string, any, error) {
				ref, err := mediaContent(cmd, true, false)
				return "image", ref, err
			}),
		},
		{
			Use: "audio", Short: "Send an audio file or voice note", Kind: kindWrite,
			Args:    cobra.NoArgs,
			Flags:   func(cmd *cobra.Command) { sendFlags(cmd); mediaFlags(cmd, false, false) },
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, _ []string) (string, any, error) {
				ref, err := mediaContent(cmd, false, false)
				return "audio", ref, err
			}),
		},
		{
			Use: "video", Short: "Send a video (MP4/3GP)", Kind: kindWrite,
			Args:    cobra.NoArgs,
			Flags:   func(cmd *cobra.Command) { sendFlags(cmd); mediaFlags(cmd, true, false) },
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, _ []string) (string, any, error) {
				ref, err := mediaContent(cmd, true, false)
				return "video", ref, err
			}),
		},
		{
			Use: "document", Short: "Send a document (PDF, office files, …)", Kind: kindWrite,
			Args:    cobra.NoArgs,
			Flags:   func(cmd *cobra.Command) { sendFlags(cmd); mediaFlags(cmd, true, true) },
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, _ []string) (string, any, error) {
				ref, err := mediaContent(cmd, true, true)
				return "document", ref, err
			}),
		},
		{
			Use: "sticker", Short: "Send a sticker (WebP)", Kind: kindWrite,
			Args:    cobra.NoArgs,
			Flags:   func(cmd *cobra.Command) { sendFlags(cmd); mediaFlags(cmd, false, false) },
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, _ []string) (string, any, error) {
				ref, err := mediaContent(cmd, false, false)
				return "sticker", ref, err
			}),
		},
		{
			Use: "location", Short: "Send a location pin", Kind: kindWrite,
			Example: `  waba send location --to 573001112233 --lat 4.7110 --lng -74.0721 --name "Oficina" --address "Bogotá"`,
			Args:    cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().Float64("lat", 0, "latitude")
				cmd.Flags().Float64("lng", 0, "longitude")
				_ = cmd.MarkFlagRequired("lat")
				_ = cmd.MarkFlagRequired("lng")
				cmd.Flags().String("name", "", "location name")
				cmd.Flags().String("address", "", "location address")
			},
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, _ []string) (string, any, error) {
				lat, _ := cmd.Flags().GetFloat64("lat")
				lng, _ := cmd.Flags().GetFloat64("lng")
				name, _ := cmd.Flags().GetString("name")
				addr, _ := cmd.Flags().GetString("address")
				return "location", map[string]any{"latitude": lat, "longitude": lng, "name": name, "address": addr}, nil
			}),
		},
		{
			Use: "contacts", Short: "Send contact cards", Kind: kindWrite,
			Long: "Send one or more vCard-style contacts. The payload is the documented contacts array,\npassed as JSON because its shape (names, phones, emails, orgs, addresses) is too rich for flags.",
			Example: `  waba send contacts --to 573001112233 --data '[{"name":{"formatted_name":"Ana","first_name":"Ana"},"phones":[{"phone":"+57301...","type":"CELL"}]}]'
  waba send contacts --to 573001112233 --data @contacts.json`,
			Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().StringP("data", "d", "", "contacts array as JSON, @file, or @- for stdin")
				_ = cmd.MarkFlagRequired("data")
			},
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, _ []string) (string, any, error) {
				data, _ := cmd.Flags().GetString("data")
				raw, err := readJSONBody(data)
				if err != nil {
					return "", nil, err
				}
				return "contacts", raw, nil
			}),
		},
		{
			Use: "reaction", Short: "React to a message with an emoji", Kind: kindWrite,
			Example: `  waba send reaction --to 573001112233 --message-id wamid.HBg... --emoji 👍
  # remove a reaction: send an empty --emoji ""`,
			Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().String("message-id", "", "wamid of the message to react to")
				_ = cmd.MarkFlagRequired("message-id")
				cmd.Flags().String("emoji", "", "emoji to react with (empty string removes the reaction)")
			},
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, _ []string) (string, any, error) {
				id, _ := cmd.Flags().GetString("message-id")
				emoji, _ := cmd.Flags().GetString("emoji")
				return "reaction", map[string]string{"message_id": id, "emoji": emoji}, nil
			}),
		},
		{
			Use: "template", Short: "Send an approved message template", Kind: kindWrite,
			Long: "Send a template by name and language. Simple body placeholders can be filled with\nrepeatable --param flags; anything richer (media headers, buttons) takes the documented\ncomponents array via --components.",
			Example: `  waba send template --to 573001112233 --name hello_world --lang en_US
  waba send template --to 573001112233 --name order_update --lang es_MX --param "Juan" --param "#1234"
  waba send template --to 573001112233 --name promo --lang es --components @components.json`,
			Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().String("name", "", "template name")
				_ = cmd.MarkFlagRequired("name")
				cmd.Flags().String("lang", "", "template language code (e.g. en_US, es_MX)")
				_ = cmd.MarkFlagRequired("lang")
				cmd.Flags().StringArray("param", nil, "positional body parameter (repeatable, in order)")
				cmd.Flags().String("components", "", "components array as JSON, @file, or @- (overrides --param)")
			},
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, _ []string) (string, any, error) {
				name, _ := cmd.Flags().GetString("name")
				lang, _ := cmd.Flags().GetString("lang")
				tpl := map[string]any{"name": name, "language": map[string]string{"code": lang}}

				if compJSON, _ := cmd.Flags().GetString("components"); compJSON != "" {
					raw, err := readJSONBody(compJSON)
					if err != nil {
						return "", nil, err
					}
					tpl["components"] = raw
				} else if params, _ := cmd.Flags().GetStringArray("param"); len(params) > 0 {
					ps := make([]map[string]string, len(params))
					for i, p := range params {
						ps[i] = map[string]string{"type": "text", "text": p}
					}
					tpl["components"] = []map[string]any{{"type": "body", "parameters": ps}}
				}
				return "template", tpl, nil
			}),
		},
		{
			Use: "interactive", Short: "Send a raw interactive message", Kind: kindWrite,
			Long:    "Send any interactive payload verbatim — the escape hatch for shapes without a dedicated\nverb (address messages, product lists, …). For the common shapes use `send buttons`,\n`send list`, `send cta-url` or `send flow`.",
			Example: `  waba send interactive --to 573001112233 --data @interactive.json`,
			Args:    cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().StringP("data", "d", "", "interactive object as JSON, @file, or @- for stdin")
				_ = cmd.MarkFlagRequired("data")
			},
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, _ []string) (string, any, error) {
				data, _ := cmd.Flags().GetString("data")
				raw, err := readJSONBody(data)
				if err != nil {
					return "", nil, err
				}
				return "interactive", raw, nil
			}),
		},
		{
			Use: "buttons <body>", Short: "Send up to 3 reply buttons", Kind: kindWrite,
			Example: `  waba send buttons --to 573001112233 "Confirm your visit" --button yes:Sí --button no:No`,
			Args:    cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().StringArray("button", nil, "reply button as id:title (repeatable, max 3)")
				cmd.Flags().String("header", "", "optional header text")
				cmd.Flags().String("footer", "", "optional footer text")
			},
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, args []string) (string, any, error) {
				defs, _ := cmd.Flags().GetStringArray("button")
				if len(defs) == 0 || len(defs) > 3 {
					return "", nil, fmt.Errorf("between 1 and 3 --button id:title flags are required")
				}
				buttons := make([]map[string]any, len(defs))
				for i, d := range defs {
					id, title, ok := strings.Cut(d, ":")
					if !ok {
						return "", nil, fmt.Errorf("--button %q is not id:title", d)
					}
					buttons[i] = map[string]any{"type": "reply", "reply": map[string]string{"id": id, "title": title}}
				}
				content := map[string]any{
					"type":   "button",
					"body":   map[string]string{"text": args[0]},
					"action": map[string]any{"buttons": buttons},
				}
				addHeaderFooter(cmd, content)
				return "interactive", content, nil
			}),
		},
		{
			Use: "list <body>", Short: "Send an interactive list menu", Kind: kindWrite,
			Example: `  waba send list --to 573001112233 "Pick a service" --button-text Menú \
    --section "Servicios" --row "rev:Revisión:Diagnóstico completo" --row "rep:Reparación"`,
			Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().String("button-text", "Menu", "text on the button that opens the list")
				cmd.Flags().String("section", "", "section title for the rows")
				cmd.Flags().StringArray("row", nil, "row as id:title[:description] (repeatable, max 10)")
				cmd.Flags().String("header", "", "optional header text")
				cmd.Flags().String("footer", "", "optional footer text")
			},
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, args []string) (string, any, error) {
				rowDefs, _ := cmd.Flags().GetStringArray("row")
				if len(rowDefs) == 0 || len(rowDefs) > 10 {
					return "", nil, fmt.Errorf("between 1 and 10 --row flags are required")
				}
				rows := make([]map[string]string, len(rowDefs))
				for i, d := range rowDefs {
					parts := strings.SplitN(d, ":", 3)
					if len(parts) < 2 {
						return "", nil, fmt.Errorf("--row %q is not id:title[:description]", d)
					}
					row := map[string]string{"id": parts[0], "title": parts[1]}
					if len(parts) == 3 {
						row["description"] = parts[2]
					}
					rows[i] = row
				}
				sectionTitle, _ := cmd.Flags().GetString("section")
				buttonText, _ := cmd.Flags().GetString("button-text")
				content := map[string]any{
					"type": "list",
					"body": map[string]string{"text": args[0]},
					"action": map[string]any{
						"button":   buttonText,
						"sections": []map[string]any{{"title": sectionTitle, "rows": rows}},
					},
				}
				addHeaderFooter(cmd, content)
				return "interactive", content, nil
			}),
		},
		{
			Use: "cta-url <body>", Short: "Send a call-to-action URL button", Kind: kindWrite,
			Example: `  waba send cta-url --to 573001112233 "See your invitation" --display-text "Abrir" --url https://invitas.co/e/abc`,
			Args:    cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().String("display-text", "", "button label")
				_ = cmd.MarkFlagRequired("display-text")
				cmd.Flags().String("url", "", "https URL the button opens")
				_ = cmd.MarkFlagRequired("url")
				cmd.Flags().String("header", "", "optional header text")
				cmd.Flags().String("footer", "", "optional footer text")
			},
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, args []string) (string, any, error) {
				display, _ := cmd.Flags().GetString("display-text")
				u, _ := cmd.Flags().GetString("url")
				content := map[string]any{
					"type": "cta_url",
					"body": map[string]string{"text": args[0]},
					"action": map[string]any{
						"name":       "cta_url",
						"parameters": map[string]string{"display_text": display, "url": u},
					},
				}
				addHeaderFooter(cmd, content)
				return "interactive", content, nil
			}),
		},
		{
			Use: "flow <body>", Short: "Send a WhatsApp Flow", Kind: kindWrite,
			Example: `  waba send flow --to 573001112233 "Book your appointment" --flow-id 123456 --flow-cta "Agendar" --flow-token tok-1`,
			Args:    cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().String("flow-id", "", "flow id (or use --flow-name)")
				cmd.Flags().String("flow-name", "", "flow name (or use --flow-id)")
				cmd.Flags().String("flow-cta", "", "button label that opens the flow")
				_ = cmd.MarkFlagRequired("flow-cta")
				cmd.Flags().String("flow-token", "", "opaque token echoed back in the flow webhook")
				cmd.Flags().String("flow-action", "navigate", "flow action: navigate|data_exchange")
				cmd.Flags().String("flow-action-payload", "", "action payload JSON (screen + data) for navigate")
				cmd.Flags().String("mode", "", "flow mode: draft to test an unpublished flow")
				cmd.Flags().String("header", "", "optional header text")
				cmd.Flags().String("footer", "", "optional footer text")
			},
			Columns: sendColumns,
			Run: sendRun(func(cmd *cobra.Command, args []string) (string, any, error) {
				flowID, _ := cmd.Flags().GetString("flow-id")
				flowName, _ := cmd.Flags().GetString("flow-name")
				if (flowID == "") == (flowName == "") {
					return "", nil, fmt.Errorf("exactly one of --flow-id or --flow-name is required")
				}
				params := map[string]any{
					"flow_message_version": "3",
					"flow_cta":             mustString(cmd, "flow-cta"),
					"flow_action":          mustString(cmd, "flow-action"),
				}
				if flowID != "" {
					params["flow_id"] = flowID
				} else {
					params["flow_name"] = flowName
				}
				if tok := mustString(cmd, "flow-token"); tok != "" {
					params["flow_token"] = tok
				}
				if mode := mustString(cmd, "mode"); mode != "" {
					params["mode"] = mode
				}
				if ap := mustString(cmd, "flow-action-payload"); ap != "" {
					raw, err := readJSONBody(ap)
					if err != nil {
						return "", nil, err
					}
					params["flow_action_payload"] = raw
				}
				content := map[string]any{
					"type":   "flow",
					"body":   map[string]string{"text": args[0]},
					"action": map[string]any{"name": "flow", "parameters": params},
				}
				addHeaderFooter(cmd, content)
				return "interactive", content, nil
			}),
		},
	}
}

// addHeaderFooter attaches the optional text header/footer shared by interactive shapes.
func addHeaderFooter(cmd *cobra.Command, content map[string]any) {
	if h := mustString(cmd, "header"); h != "" {
		content["header"] = map[string]string{"type": "text", "text": h}
	}
	if f := mustString(cmd, "footer"); f != "" {
		content["footer"] = map[string]string{"text": f}
	}
}

// mustString reads a string flag that is known to exist on the command.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
