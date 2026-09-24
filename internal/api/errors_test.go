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
