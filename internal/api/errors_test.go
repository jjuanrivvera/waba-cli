package api

import (
	"fmt"
	"strings"
	"testing"
)

// The report in issue #11, verbatim: the account is an SMB business type, the token holds both
// WhatsApp permissions, and Meta's own message already names the cause.
func TestHint_Code10_SMBBusinessType_NoPermissionHint(t *testing.T) {
	e := &APIError{
		StatusCode: 400,
		Code:       10,
		Message:    "(#10) This operation can not be performed on SMB business type",
	}

	if hint := e.Hint(); hint != "" {
		t.Errorf("code 10 with a non-permission message must not claim a permission problem, got %q", hint)
	}
	if strings.Contains(e.Error(), "permission missing") {
		t.Errorf("the rendered error still contradicts Meta's message:\n%s", e.Error())
	}
	// Meta's account of the failure must survive.
	if !strings.Contains(e.Error(), "SMB business type") {
		t.Errorf("Meta's own message should be shown:\n%s", e.Error())
	}
}

// Code 10 is still Graph's permission code most of the time, and that hint is worth keeping.
func TestHint_Code10_RealPermissionFailure(t *testing.T) {
	cases := []string{
		"(#10) Application does not have permission for this action",
		"Unauthorized use of this endpoint",
		"(#10) Access denied for this WABA",
	}
	for _, msg := range cases {
		t.Run(msg[:20], func(t *testing.T) {
			e := &APIError{StatusCode: 403, Code: 10, Message: msg}
			if hint := e.Hint(); hint == "" {
				t.Error("a permission refusal should still get the permission hint")
			}
		})
	}
}

// Code 10 also covers App Review: "your use of this endpoint must be reviewed" is a gate on
// the app, not on the token's scopes, so the WhatsApp-permission hint would send the reader to
// the wrong place there too — the same failure mode as the SMB case, in different words.
func TestHint_Code10_AppReviewIsNotAScopeProblem(t *testing.T) {
	e := &APIError{
		Code:    10,
		Message: "(#10) To use 'Page Public Content Access', your use of this endpoint must be reviewed and approved",
	}
	if hint := e.Hint(); strings.Contains(hint, "whatsapp_business_messaging") {
		t.Errorf("an App Review gate is not a missing WhatsApp scope, got %q", hint)
	}
}

// The wording can arrive in error_data.details or in error_user_msg instead of the message.
func TestHint_Code10_PermissionWordingInOtherFields(t *testing.T) {
	fromDetails := &APIError{Code: 10, Message: "(#10) Operation failed", Details: "missing permission whatsapp_business_management"}
	if fromDetails.Hint() == "" {
		t.Error("permission wording in error_data.details should still produce the hint")
	}
	fromUserMsg := &APIError{Code: 10, Message: "(#10) Operation failed", UserMsg: "You are not authorized to manage this account"}
	if fromUserMsg.Hint() == "" {
		t.Error("permission wording in error_user_msg should still produce the hint")
	}
}

// The codes Graph reserves exclusively for permissions keep the unconditional hint.
func TestHint_PermissionCodeRangeUnchanged(t *testing.T) {
	for _, code := range []int{200, 210, 220, 294, 299} {
		e := &APIError{Code: code, Message: fmt.Sprintf("(#%d) Some failure", code)}
		if !strings.Contains(e.Hint(), "permission missing") {
			t.Errorf("code %d should still hint at permissions, got %q", code, e.Hint())
		}
	}
}

// error_user_msg is often the only field naming the real cause, so it has to reach the reader.
func TestError_ShowsMetaUserMessage(t *testing.T) {
	e := &APIError{
		StatusCode: 400,
		Code:       131047,
		Message:    "(#131047) Re-engagement message",
		UserTitle:  "Message failed to send",
		UserMsg:    "More than 24 hours have passed since the recipient last replied.",
	}
	got := e.Error()
	if !strings.Contains(got, "Message failed to send: More than 24 hours") {
		t.Errorf("Meta's user-facing message should be rendered:\n%s", got)
	}
	// And the actionable hint is still there — this replaces nothing.
	if !strings.Contains(got, "approved template") {
		t.Errorf("the hint should survive alongside it:\n%s", got)
	}
}

// A user message that merely repeats what is already printed adds noise, not information.
func TestError_SkipsRedundantUserMessage(t *testing.T) {
	msg := "This operation can not be performed on SMB business type"
	e := &APIError{StatusCode: 400, Code: 10, Message: msg, UserMsg: msg}
	if strings.Count(e.Error(), msg) != 1 {
		t.Errorf("the same sentence should appear once:\n%s", e.Error())
	}
}

// A refusal about a permission that isn't a WhatsApp scope must not prescribe WhatsApp scopes:
// that is the same wrong turn as the SMB case, one level in.
func TestHint_Code10_ForeignScopeGetsGenericAdvice(t *testing.T) {
	e := &APIError{Code: 10, Message: "(#10) requires pages_read_engagement permission"}
	hint := e.Hint()
	if hint == "" {
		t.Fatal("a genuine refusal should still be hinted")
	}
	if strings.Contains(hint, "whatsapp_business_") {
		t.Errorf("must not prescribe WABA scopes for someone else's permission: %q", hint)
	}
}

func TestHint_Code10_WhatsAppScopeNamed(t *testing.T) {
	e := &APIError{Code: 10, Message: "(#10) The token requires whatsapp_business_management permission"}
	if !strings.Contains(e.Hint(), "whatsapp_business_management") {
		t.Errorf("when Meta names the WABA scope, say so: %q", e.Hint())
	}
}

// Either user-facing field can arrive alone, and either can repeat what was already printed.
func TestError_MetaLine(t *testing.T) {
	cases := []struct {
		name             string
		msg, title, umsg string
		want, wantAbsent string
	}{{
		name:  "title without a message still reaches the reader",
		msg:   "(#100) Invalid parameter",
		title: "Number already registered",
		want:  "meta: Number already registered",
	}, {
		name:  "both halves already shown, so no meta line",
		msg:   "Cannot send. Rate limit hit",
		title: "Cannot send",
		umsg:  "Rate limit hit",
		want:  "",
		// The joined form would have looked new while both halves were already there.
		wantAbsent: "meta:",
	}, {
		name:  "only the new half is added",
		msg:   "Cannot send",
		title: "Cannot send",
		umsg:  "Wait 60 seconds and retry",
		want:  "meta: Wait 60 seconds and retry",
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &APIError{StatusCode: 400, Message: tc.msg, UserTitle: tc.title, UserMsg: tc.umsg}
			got := e.Error()
			if tc.want != "" && !strings.Contains(got, tc.want) {
				t.Errorf("want %q in:\n%s", tc.want, got)
			}
			if tc.wantAbsent != "" && strings.Contains(got, tc.wantAbsent) {
				t.Errorf("did not want %q in:\n%s", tc.wantAbsent, got)
			}
		})
	}
}

// error_user_title can be the only field that says "permission", and Error() prints it — so
// the hint has to read it too, or the reader sees the refusal with no guidance beside it.
func TestHint_Code10_PermissionWordingInUserTitle(t *testing.T) {
	e := &APIError{Code: 10, Message: "(#10) Operation failed", UserTitle: "Permission denied"}
	if e.Hint() == "" {
		t.Error("a refusal named only in error_user_title should still be hinted")
	}
}

// Meta's prose names the account, not a scope: "WhatsApp Business account" appears in
// refusals about entirely different permissions, and must not summon the WABA scopes.
func TestHint_Code10_AccountNameIsNotAScope(t *testing.T) {
	e := &APIError{
		Code:    10,
		Message: "(#10) Missing pages_read_engagement permission to access this WhatsApp Business account",
	}
	hint := e.Hint()
	if strings.Contains(hint, "whatsapp_business_messaging") {
		t.Errorf("an account name is not a scope name: %q", hint)
	}
	if hint == "" {
		t.Error("it is still a refusal, so it should still be hinted")
	}
}

// When Meta names no permission at all, the hint must not promise one.
func TestHint_Code10_NoPermissionNamed(t *testing.T) {
	e := &APIError{Code: 10, Message: "Unauthorized use of this endpoint"}
	hint := e.Hint()
	if strings.Contains(hint, "named above") {
		t.Errorf("nothing was named above: %q", hint)
	}
	if hint == "" {
		t.Error("a refusal should still be hinted")
	}
}
