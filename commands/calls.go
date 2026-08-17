package commands

import (
	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("calls", "WhatsApp Calling: place, answer and manage calls", []string{"call"}, callSpecs)
}

// callActionRun posts one of the payload-discriminated actions on /calls. SDP flows in as
// JSON (@file works) because a WebRTC session description is not flag material.
func callActionRun(action string, withSession, withCallID bool) func(*cobra.Command, *globalOptions, *api.Client, *config.Account, []string) (any, error) {
	return func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
		phone, err := o.phoneNumberID(acct)
		if err != nil {
			return nil, err
		}
		body := map[string]any{"messaging_product": "whatsapp", "action": action}
		if withCallID {
			body["call_id"] = args[0]
		}
		if to := mustString(cmd, "to"); to != "" {
			body["to"] = to
		}
		if withSession {
			if sdp := mustString(cmd, "sdp"); sdp != "" {
				sdpType := mustString(cmd, "sdp-type")
				body["session"] = map[string]string{"sdp_type": sdpType, "sdp": sdp}
			}
		}
		if cb := mustString(cmd, "callback-data"); cb != "" {
			body["biz_opaque_callback_data"] = cb
		}
		raw, err := c.Do(cmd.Context(), api.Request{Method: "POST", Path: phone + "/calls", Body: body})
		if err != nil {
			return nil, err
		}
		o.noteWrite(cmd.ErrOrStderr(), "call %s sent", action)
		return jsonMap(raw), nil
	}
}

func sdpFlags(cmd *cobra.Command, defType string) {
	cmd.Flags().String("sdp", "", "RFC 8866 session description (inline or @file via shell)")
	cmd.Flags().String("sdp-type", defType, "sdp_type: offer or answer")
}

func callSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "connect", Short: "Start a business-initiated call",
			Long:    "Requires an existing call permission from the user (`waba calls permissions`),\ncalling enabled on the number, and a calls webhook. Limited by Meta to 10,000 call\ninitiations per number per 24h; unavailable for businesses in some countries.",
			Example: `  waba calls connect --to 573001112233 --sdp "$(cat offer.sdp)"`,
			Kind:    kindWrite, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("to", "", "callee phone number")
				_ = cmd.MarkFlagRequired("to")
				sdpFlags(cmd, "offer")
				cmd.Flags().String("callback-data", "", "opaque data echoed in call webhooks (≤512 chars)")
			},
			Run: callActionRun("connect", true, false),
		},
		{
			Use: "pre-accept <call-id>", Short: "Pre-accept an inbound call (early media setup)",
			Long: "Optional but recommended: establishing WebRTC before accept avoids clipping the\nfirst moments of audio. Respond within the 30-60s window after the Call Connect webhook.",
			Kind: kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) { sdpFlags(cmd, "answer") },
			Run:   callActionRun("pre_accept", true, true),
		},
		{
			Use: "accept <call-id>", Short: "Accept an inbound call",
			Kind: kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				sdpFlags(cmd, "answer")
				cmd.Flags().String("callback-data", "", "opaque data echoed in call webhooks (≤512 chars)")
			},
			Run: callActionRun("accept", true, true),
		},
		{
			Use: "reject <call-id>", Short: "Reject an inbound call",
			Kind: kindWrite, Args: cobra.ExactArgs(1),
			Run: callActionRun("reject", false, true),
		},
		{
			Use: "terminate <call-id>", Short: "End an active call",
			Long: "Required for accurate billing even when RTCP BYE was already sent (unless the user\nhung up first).",
			Kind: kindWrite, Args: cobra.ExactArgs(1),
			Run: callActionRun("terminate", false, true),
		},
		{
			Use: "permissions <user-number>", Short: "Check the call permission state for a user",
			Example: `  waba calls permissions 573001112233`,
			Kind:    kindRead, Args: cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				raw, err := c.Do(cmd.Context(), api.Request{Path: phone + "/call_permissions",
					Query: urlValues("user_wa_id", args[0])})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "request-permission <user-number>", Short: "Send an interactive call-permission request",
			Long:    "Free-form call permission requests only work inside an open customer-service window;\noutside it, send a template containing a call_permission_request component. Meta enforces\n1 request per user per 24h and 2 per 7 days.",
			Example: `  waba calls request-permission 573001112233 --body "¿Podemos llamarte para coordinar la visita?"`,
			Kind:    kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("body", "", "message text shown with the permission request")
			},
			Columns: sendColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				content := map[string]any{
					"type":   "call_permission_request",
					"action": map[string]any{"name": "call_permission_request"},
				}
				if b := mustString(cmd, "body"); b != "" {
					content["body"] = map[string]string{"text": b}
				}
				res, err := c.SendMessage(cmd.Context(), phone, api.MessagePayload(args[0], "interactive", content))
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "call permission requested from %s", args[0])
				return flatSend(res), nil
			},
		},
		{
			Use: "send-call-button <user-number>", Short: "Send a voice-call button message",
			Long:    "Sends an interactive voice_call button the user can tap to call the business.",
			Example: `  waba calls send-call-button 573001112233 --body "¿Prefieres hablar?" --display-text "Llámanos" --ttl-minutes 1440`,
			Kind:    kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("body", "", "message text above the button")
				cmd.Flags().String("display-text", "Call", "button label (≤20 chars)")
				cmd.Flags().Int("ttl-minutes", 0, "how long the button stays tappable (1–43200)")
				cmd.Flags().String("payload", "", "opaque payload echoed in call webhooks (≤512 chars)")
			},
			Columns: sendColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				params := map[string]any{"display_text": mustString(cmd, "display-text")}
				if ttl, _ := cmd.Flags().GetInt("ttl-minutes"); ttl > 0 {
					params["ttl_minutes"] = ttl
				}
				if p := mustString(cmd, "payload"); p != "" {
					params["payload"] = p
				}
				content := map[string]any{
					"type":   "voice_call",
					"action": map[string]any{"name": "voice_call", "parameters": params},
				}
				if b := mustString(cmd, "body"); b != "" {
					content["body"] = map[string]string{"text": b}
				}
				res, err := c.SendMessage(cmd.Context(), phone, api.MessagePayload(args[0], "interactive", content))
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "voice-call button sent to %s", args[0])
				return flatSend(res), nil
			},
		},
	}
}
