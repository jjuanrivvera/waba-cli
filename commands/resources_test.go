package commands

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// One test per resource group, asserting the request each verb produces. Destructive verbs
// pass --yes; the prompt path is covered separately in TestConfirmPromptAborts.

func TestMedia_Lifecycle(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	// upload: multipart with messaging_product + type + file part
	file := filepath.Join(t.TempDir(), "promo.jpg")
	require.NoError(t, os.WriteFile(file, []byte("jpegbytes"), 0o600))
	m.on("POST", "/v25.0/111/media", `{"id":"MEDIA1"}`)
	out, _, err := runCmd(t, "media", "upload", file, "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "MEDIA1")
	up := m.last()
	assert.Contains(t, up.Header.Get("Content-Type"), "multipart/form-data")
	assert.Contains(t, string(up.Body), "messaging_product")
	assert.Contains(t, string(up.Body), "jpegbytes")

	// unknown extension without --type must fail before any request
	weird := filepath.Join(t.TempDir(), "blob.zzz")
	require.NoError(t, os.WriteFile(weird, []byte("x"), 0o600))
	_, _, err = runCmd(t, "media", "upload", weird)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--type")

	// url
	m.on("GET", "/v25.0/MEDIA1", `{"id":"MEDIA1","url":"`+m.server.URL+`/lookaside/blob","mime_type":"image/jpeg","file_size":9}`)
	out, _, err = runCmd(t, "media", "url", "MEDIA1", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "lookaside")

	// download follows the URL (same host as the mock → allowed) and writes the file
	m.on("GET", "/lookaside/blob", `binarybytes`)
	dest := filepath.Join(t.TempDir(), "out.jpg")
	_, _, err = runCmd(t, "media", "download", "MEDIA1", "-f", dest)
	require.NoError(t, err)
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "binarybytes", string(data))

	// delete
	_, _, err = runCmd(t, "media", "delete", "MEDIA1", "--yes")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, m.last().Method)
	assert.Equal(t, "/v25.0/MEDIA1", m.last().Path)
}

func TestPhone_Verbs(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("GET", "/v25.0/222/phone_numbers", `{"data":[{"id":"111","display_phone_number":"+57 300","verified_name":"Rivera"}]}`)
	out, _, err := runCmd(t, "phone", "list", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "Rivera")

	_, _, err = runCmd(t, "phone", "get", "--fields", "id,status")
	require.NoError(t, err)
	assert.Equal(t, "id,status", m.last().Query["fields"])
	assert.Equal(t, "/v25.0/111", m.last().Path, "the account's phone id is the default")

	_, _, err = runCmd(t, "phone", "name-status")
	require.NoError(t, err)
	assert.Equal(t, "name_status", m.last().Query["fields"])

	_, _, err = runCmd(t, "phone", "register", "--pin", "123456", "--region", "CO")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	assert.Equal(t, "123456", body["pin"])
	assert.Equal(t, "CO", body["data_localization_region"])

	_, _, err = runCmd(t, "phone", "deregister", "--yes")
	require.NoError(t, err)
	assert.Equal(t, "/v25.0/111/deregister", m.last().Path)

	_, _, err = runCmd(t, "phone", "request-code", "--method", "voice")
	require.NoError(t, err)
	assert.Equal(t, "VOICE", jsonBody(t, m.last())["code_method"])
	_, _, err = runCmd(t, "phone", "request-code", "--method", "carrier-pigeon")
	require.Error(t, err)

	_, _, err = runCmd(t, "phone", "verify-code", "042042")
	require.NoError(t, err)
	assert.Equal(t, "042042", jsonBody(t, m.last())["code"])

	_, _, err = runCmd(t, "phone", "set-pin", "654321")
	require.NoError(t, err)
	assert.Equal(t, "654321", jsonBody(t, m.last())["pin"])
	_, _, err = runCmd(t, "phone", "set-pin", "12")
	require.Error(t, err, "a non-6-digit PIN must be rejected")

	_, _, err = runCmd(t, "phone", "settings", "--sip-credentials")
	require.NoError(t, err)
	assert.Equal(t, "true", m.last().Query["include_sip_credentials"])

	_, _, err = runCmd(t, "phone", "update-settings", "-d", `{"calling":{"status":"ENABLED"}}`)
	require.NoError(t, err)
	assert.Equal(t, "/v25.0/111/settings", m.last().Path)
}

func TestProfile_GetUpdate(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("GET", "/v25.0/111/whatsapp_business_profile", `{"data":[{"about":"Reparación de neveras","vertical":"OTHER"}]}`)
	out, _, err := runCmd(t, "profile", "get", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "neveras")

	_, _, err = runCmd(t, "profile", "update", "--about", "Nuevo", "--website", "https://a.co", "--website", "https://b.co")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	assert.Equal(t, "whatsapp", body["messaging_product"])
	assert.Equal(t, "Nuevo", body["about"])
	assert.Len(t, body["websites"].([]any), 2)
}

func TestQR_Lifecycle(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("POST", "/v25.0/111/message_qrdls", `{"code":"QR1","prefilled_message":"hola","deep_link_url":"https://wa.me/x"}`)
	_, _, err := runCmd(t, "qr", "create", "hola", "--image", "SVG")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	assert.Equal(t, "hola", body["prefilled_message"])
	assert.Equal(t, "SVG", body["generate_qr_image"])

	m.on("GET", "/v25.0/111/message_qrdls", `{"data":[{"code":"QR1"}]}`)
	_, _, err = runCmd(t, "qr", "list")
	require.NoError(t, err)

	m.on("GET", "/v25.0/111/message_qrdls/QR1", `{"data":[{"code":"QR1"}]}`)
	_, _, err = runCmd(t, "qr", "get", "QR1")
	require.NoError(t, err)

	_, _, err = runCmd(t, "qr", "update", "QR1", "nuevo mensaje")
	require.NoError(t, err)
	body = jsonBody(t, m.last())
	assert.Equal(t, "QR1", body["code"], "update rides the collection path with a code param")

	_, _, err = runCmd(t, "qr", "delete", "QR1", "--yes")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, m.last().Method)
	assert.Equal(t, "/v25.0/111/message_qrdls/QR1", m.last().Path)
}

func TestBlock_Verbs(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	_, _, err := runCmd(t, "block", "add", "573001", "573002")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	users := body["block_users"].([]any)
	require.Len(t, users, 2)
	assert.Equal(t, "573001", users[0].(map[string]any)["user"])

	_, _, err = runCmd(t, "block", "remove", "573001")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, m.last().Method)

	m.on("GET", "/v25.0/111/block_users", `{"data":[{"wa_id":"573001"}]}`)
	_, _, err = runCmd(t, "block", "list")
	require.NoError(t, err)
}

func TestCommerce_GetUpdate(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("GET", "/v25.0/111/whatsapp_commerce_settings", `{"data":[{"is_cart_enabled":true,"is_catalog_visible":false,"id":"1"}]}`)
	_, _, err := runCmd(t, "commerce", "get")
	require.NoError(t, err)

	_, _, err = runCmd(t, "commerce", "update", "--cart=false")
	require.NoError(t, err)
	assert.Equal(t, "false", m.last().Query["is_cart_enabled"])
	_, hasCatalog := m.last().Query["is_catalog_visible"]
	assert.False(t, hasCatalog, "an unset flag must not be sent — it would reset the server value")

	_, _, err = runCmd(t, "commerce", "update")
	require.Error(t, err, "no flags set must be an error, not an empty request")
}

func TestAutomation_GetUpdate(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("GET", "/v25.0/111", `{"conversational_automation":{"prompts":["¿Horario?"]},"id":"111"}`)
	out, _, err := runCmd(t, "automation", "get", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "Horario")

	_, _, err = runCmd(t, "automation", "update",
		"--command", "cotizar:Recibe una cotización", "--prompt", "¿Atienden hoy?")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	cmds := body["commands"].([]any)
	assert.Equal(t, "cotizar", cmds[0].(map[string]any)["command_name"])
	assert.Equal(t, []any{"¿Atienden hoy?"}, body["prompts"])
	assert.Nil(t, body["enable_welcome_message"], "removed from the API — must never be sent")

	_, _, err = runCmd(t, "automation", "update", "--command", "sin-descripcion")
	require.Error(t, err)
}

func TestTemplates_Verbs(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("GET", "/v25.0/222/message_templates", `{"data":[{"id":"T1","name":"hello","status":"APPROVED"}]}`)
	out, _, err := runCmd(t, "templates", "list", "--status", "APPROVED", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "hello")
	assert.Equal(t, "APPROVED", m.last().Query["status"])
	assert.NotEmpty(t, m.last().Query["fields"], "a default projection keeps the table useful")

	m.on("GET", "/v25.0/T1", `{"id":"T1","name":"hello","components":[]}`)
	_, _, err = runCmd(t, "templates", "get", "T1")
	require.NoError(t, err)

	m.on("POST", "/v25.0/222/message_templates", `{"id":"T2","status":"PENDING","category":"UTILITY"}`)
	_, _, err = runCmd(t, "templates", "create", "-d", `{"name":"n","language":"es","category":"UTILITY","components":[]}`)
	require.NoError(t, err)

	_, _, err = runCmd(t, "templates", "create", "-d", `not-json`)
	require.Error(t, err)

	_, _, err = runCmd(t, "templates", "edit", "T1", "-d", `{"components":[]}`)
	require.NoError(t, err)
	assert.Equal(t, "/v25.0/T1", m.last().Path)

	_, _, err = runCmd(t, "templates", "delete", "hello", "--yes")
	require.NoError(t, err)
	assert.Equal(t, "hello", m.last().Query["name"])

	_, _, err = runCmd(t, "templates", "delete-by-id", "T1", "hello", "--yes")
	require.NoError(t, err)
	assert.Equal(t, "T1", m.last().Query["hsm_id"])
	assert.Equal(t, "hello", m.last().Query["name"])

	_, _, err = runCmd(t, "templates", "bulk-delete", "T1", "T2", "--yes")
	require.NoError(t, err)
	assert.JSONEq(t, `["T1","T2"]`, m.last().Query["hsm_ids"])

	// compare validates the window
	restore := nowUnix
	nowUnix = func() int64 { return 1_755_000_000 }
	defer func() { nowUnix = restore }()
	m.on("GET", "/v25.0/T1/compare", `{"data":[{"metric":"BLOCK_RATE"}]}`)
	_, _, err = runCmd(t, "templates", "compare", "T1", "T2", "--days", "30")
	require.NoError(t, err)
	assert.Equal(t, "T2", m.last().Query["template_ids"])
	assert.Equal(t, "1755000000", m.last().Query["end"])
	_, _, err = runCmd(t, "templates", "compare", "T1", "T2", "--days", "15")
	require.Error(t, err, "only 7/30/60/90 day windows are documented")

	_, _, err = runCmd(t, "templates", "click-tracking", "T1", "--opt-out", "--category", "MARKETING")
	require.NoError(t, err)
	assert.Equal(t, "true", m.last().Query["cta_url_link_tracking_opted_out"])
	assert.Equal(t, "MARKETING", m.last().Query["category"])
}

func TestAccountAndApps_Verbs(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("GET", "/v25.0/222", `{"id":"222","name":"Rivera WABA","currency":"USD"}`)
	out, _, err := runCmd(t, "account", "get", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "Rivera WABA")
	assert.NotEmpty(t, m.last().Query["fields"])

	_, _, err = runCmd(t, "account", "update", "-d", `{"disable_marketing_messages_on_cloud_api":true}`)
	require.NoError(t, err)
	assert.Equal(t, true, jsonBody(t, m.last())["disable_marketing_messages_on_cloud_api"])

	_, _, err = runCmd(t, "account", "enable-insights", "--yes")
	require.NoError(t, err)
	assert.Equal(t, "true", m.last().Query["is_enabled_for_insights"])

	m.on("GET", "/v25.0/222/subscribed_apps", `{"data":[{"whatsapp_business_api_data":{"id":"A1","name":"MyApp","link":"https://x"},"override_callback_uri":"https://cb"}]}`)
	out, _, err = runCmd(t, "apps", "list", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "MyApp")
	assert.Contains(t, out, "https://cb")

	_, _, err = runCmd(t, "apps", "subscribe", "--callback-url", "https://bot.x/webhook", "--verify-token", "vt")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	assert.Equal(t, "https://bot.x/webhook", body["override_callback_uri"])

	_, _, err = runCmd(t, "apps", "subscribe")
	require.NoError(t, err)
	assert.Empty(t, m.last().Body, "no override → no body")

	_, _, err = runCmd(t, "apps", "unsubscribe", "--yes")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, m.last().Method)
}

func TestAnalytics_Expansions(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("GET", "/v25.0/222", `{"analytics":{"data_points":[]},"id":"222"}`)

	_, _, err := runCmd(t, "analytics", "messaging", "--start", "2026-08-01", "--end", "2026-08-17",
		"--granularity", "DAY", "--country-codes", "CO,MX")
	require.NoError(t, err)
	f := m.last().Query["fields"]
	assert.Contains(t, f, "analytics.start(")
	assert.Contains(t, f, "granularity(DAY)")
	assert.Contains(t, f, `country_codes(["CO","MX"])`)

	_, _, err = runCmd(t, "analytics", "conversations", "--start", "1754000000", "--end", "1755000000",
		"--dimensions", "CONVERSATION_CATEGORY,COUNTRY")
	require.NoError(t, err)
	f = m.last().Query["fields"]
	assert.Contains(t, f, "conversation_analytics.start(1754000000)")
	assert.Contains(t, f, "granularity(DAILY)", "conversation analytics uses DAILY, not DAY")
	assert.Contains(t, f, `dimensions(["CONVERSATION_CATEGORY","COUNTRY"])`)

	_, _, err = runCmd(t, "analytics", "pricing", "--start", "2026-08-01", "--end", "2026-08-17",
		"--pricing-categories", "MARKETING")
	require.NoError(t, err)
	assert.Contains(t, m.last().Query["fields"], "pricing_analytics.")

	_, _, err = runCmd(t, "analytics", "calls", "--start", "2026-08-01", "--end", "2026-08-17",
		"--metric-types", "COUNT")
	require.NoError(t, err)
	assert.Contains(t, m.last().Query["fields"], "call_analytics.")

	// the DAY/DAILY confusion is a real trap — reject the wrong one loudly
	_, _, err = runCmd(t, "analytics", "messaging", "--start", "2026-08-01", "--end", "2026-08-17", "--granularity", "DAILY")
	require.Error(t, err)
	_, _, err = runCmd(t, "analytics", "conversations", "--start", "2026-08-01", "--end", "2026-08-17", "--granularity", "DAY")
	require.Error(t, err)

	_, _, err = runCmd(t, "analytics", "messaging", "--start", "not-a-date", "--end", "2026-08-17")
	require.Error(t, err)
}

func TestAnalytics_Edges(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("GET", "/v25.0/222/template_analytics", `{"data":[]}`)
	_, _, err := runCmd(t, "analytics", "templates", "--start", "2026-08-01", "--end", "2026-08-17",
		"--template-ids", "T1,T2", "--metric-types", "SENT,READ", "--waba-timezone")
	require.NoError(t, err)
	q := m.last().Query
	assert.JSONEq(t, `["T1","T2"]`, q["template_ids"])
	assert.JSONEq(t, `["SENT","READ"]`, q["metric_types"])
	assert.Equal(t, "true", q["use_waba_timezone"])
	assert.Equal(t, "DAILY", q["granularity"])

	m.on("GET", "/v25.0/222/template_group_analytics", `{"data":[]}`)
	_, _, err = runCmd(t, "analytics", "template-groups", "--start", "2026-08-01", "--end", "2026-08-17", "--group-ids", "G1")
	require.NoError(t, err)
	assert.JSONEq(t, `["G1"]`, m.last().Query["template_group_ids"])

	m.on("GET", "/v25.0/222/group_analytics", `{"data":[]}`)
	_, _, err = runCmd(t, "analytics", "groups", "--start", "2026-08-01", "--end", "2026-08-17", "--group-ids", "G2")
	require.NoError(t, err)
	assert.JSONEq(t, `["G2"]`, m.last().Query["group_ids"])
}

func TestUploads_SessionFlow(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	file := filepath.Join(t.TempDir(), "header.png")
	require.NoError(t, os.WriteFile(file, []byte("png-bytes-here"), 0o600))

	m.on("POST", "/v25.0/333/uploads", `{"id":"upload:SESS1"}`)
	out, _, err := runCmd(t, "uploads", "start", file, "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "upload:SESS1")
	q := m.last().Query
	assert.Equal(t, "header.png", q["file_name"])
	assert.Equal(t, "14", q["file_length"])
	assert.Equal(t, "image/png", q["file_type"])

	m.on("POST", "/v25.0/upload:SESS1", `{"h":"HANDLE1"}`)
	out, _, err = runCmd(t, "uploads", "upload", "upload:SESS1", file, "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "HANDLE1")
	req := m.last()
	assert.Equal(t, "OAuth test-token", req.Header.Get("Authorization"),
		"the resumable upload API rejects Bearer")
	assert.Equal(t, "0", req.Header.Get("file_offset"))
	assert.Equal(t, "png-bytes-here", string(req.Body))

	// resume from an offset uploads only the tail
	_, _, err = runCmd(t, "uploads", "upload", "upload:SESS1", file, "--offset", "4")
	require.NoError(t, err)
	assert.Equal(t, "bytes-here", string(m.last().Body))
	assert.Equal(t, "4", m.last().Header.Get("file_offset"))

	_, _, err = runCmd(t, "uploads", "upload", "upload:SESS1", file, "--offset", "999")
	require.Error(t, err, "an offset beyond the file must be rejected")

	m.on("GET", "/v25.0/upload:SESS1", `{"id":"upload:SESS1","file_offset":4}`)
	out, _, err = runCmd(t, "uploads", "status", "upload:SESS1", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "file_offset")
	assert.Equal(t, "OAuth test-token", m.last().Header.Get("Authorization"))
}

func TestFlows_Lifecycle(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("GET", "/v25.0/222/flows", `{"data":[{"id":"F1","name":"booking","status":"DRAFT"}]}`)
	out, _, err := runCmd(t, "flows", "list", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "booking")

	flowJSON := filepath.Join(t.TempDir(), "flow.json")
	require.NoError(t, os.WriteFile(flowJSON, []byte(`{"version":"7.0","screens":[]}`), 0o600))
	m.on("POST", "/v25.0/222/flows", `{"id":"F2","success":true}`)
	_, _, err = runCmd(t, "flows", "create", "Agendar", "--categories", "APPOINTMENT_BOOKING", "--json", flowJSON)
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	assert.Equal(t, "Agendar", body["name"])
	assert.Contains(t, body["flow_json"], "screens")

	m.on("GET", "/v25.0/F1", `{"id":"F1","name":"booking","status":"DRAFT"}`)
	_, _, err = runCmd(t, "flows", "get", "F1", "--fields", "id,name,status")
	require.NoError(t, err)
	assert.Equal(t, "id,name,status", m.last().Query["fields"])

	_, _, err = runCmd(t, "flows", "preview", "F1", "--invalidate")
	require.NoError(t, err)
	assert.Equal(t, "preview.invalidate(true)", m.last().Query["fields"])

	_, _, err = runCmd(t, "flows", "metrics", "F1", "--since", "2026-08-01", "--until", "2026-08-17")
	require.NoError(t, err)
	assert.Contains(t, m.last().Query["fields"], "metric.name(ENDPOINT_REQUEST_COUNT)")

	_, _, err = runCmd(t, "flows", "update", "F1", "--name", "booking-v2")
	require.NoError(t, err)
	assert.Equal(t, "booking-v2", jsonBody(t, m.last())["name"])
	_, _, err = runCmd(t, "flows", "update", "F1")
	require.Error(t, err)

	m.on("POST", "/v25.0/F1/assets", `{"success":true,"validation_errors":[]}`)
	_, _, err = runCmd(t, "flows", "upload-json", "F1", flowJSON)
	require.NoError(t, err)
	assert.Contains(t, m.last().Header.Get("Content-Type"), "multipart/form-data")
	assert.Contains(t, string(m.last().Body), "FLOW_JSON")

	m.on("GET", "/v25.0/F1/assets", `{"data":[{"name":"flow.json","asset_type":"FLOW_JSON"}]}`)
	_, _, err = runCmd(t, "flows", "assets", "F1")
	require.NoError(t, err)

	_, _, err = runCmd(t, "flows", "publish", "F1")
	require.NoError(t, err)
	assert.Equal(t, "/v25.0/F1/publish", m.last().Path)

	_, _, err = runCmd(t, "flows", "deprecate", "F1", "--yes")
	require.NoError(t, err)
	assert.Equal(t, "/v25.0/F1/deprecate", m.last().Path)

	_, _, err = runCmd(t, "flows", "delete", "F1", "--yes")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, m.last().Method)

	_, _, err = runCmd(t, "flows", "migrate", "--from", "999", "--names", "booking,lead_gen")
	require.NoError(t, err)
	assert.Equal(t, "/v25.0/222/migrate_flows", m.last().Path)
	assert.Equal(t, "999", m.last().Query["source_waba_id"])
	assert.Equal(t, "booking,lead_gen", m.last().Query["source_flow_names"])
}

func TestCalls_Verbs(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/calls", `{"messages":[{"id":"wacid.1"}]}`)

	_, _, err := runCmd(t, "calls", "connect", "--to", "573001", "--sdp", "v=0...", "--callback-data", "ctx-1")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	assert.Equal(t, "connect", body["action"])
	assert.Equal(t, "offer", body["session"].(map[string]any)["sdp_type"])
	assert.Equal(t, "ctx-1", body["biz_opaque_callback_data"])

	_, _, err = runCmd(t, "calls", "pre-accept", "wacid.9", "--sdp", "v=0...")
	require.NoError(t, err)
	body = jsonBody(t, m.last())
	assert.Equal(t, "pre_accept", body["action"])
	assert.Equal(t, "wacid.9", body["call_id"])
	assert.Equal(t, "answer", body["session"].(map[string]any)["sdp_type"])

	_, _, err = runCmd(t, "calls", "accept", "wacid.9", "--sdp", "v=0...")
	require.NoError(t, err)
	assert.Equal(t, "accept", jsonBody(t, m.last())["action"])

	_, _, err = runCmd(t, "calls", "reject", "wacid.9")
	require.NoError(t, err)
	assert.Equal(t, "reject", jsonBody(t, m.last())["action"])

	_, _, err = runCmd(t, "calls", "terminate", "wacid.9")
	require.NoError(t, err)
	assert.Equal(t, "terminate", jsonBody(t, m.last())["action"])

	m.on("GET", "/v25.0/111/call_permissions", `{"permission":{"status":"temporary"},"actions":[]}`)
	out, _, err := runCmd(t, "calls", "permissions", "573001", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "temporary")
	assert.Equal(t, "573001", m.last().Query["user_wa_id"])

	m.on("POST", "/v25.0/111/messages", sendOK)
	_, _, err = runCmd(t, "calls", "request-permission", "573001", "--body", "¿Te llamamos?")
	require.NoError(t, err)
	inter := jsonBody(t, m.last())["interactive"].(map[string]any)
	assert.Equal(t, "call_permission_request", inter["type"])

	_, _, err = runCmd(t, "calls", "send-call-button", "573001", "--display-text", "Llámanos", "--ttl-minutes", "1440")
	require.NoError(t, err)
	inter = jsonBody(t, m.last())["interactive"].(map[string]any)
	assert.Equal(t, "voice_call", inter["type"])
	params := inter["action"].(map[string]any)["parameters"].(map[string]any)
	assert.Equal(t, float64(1440), params["ttl_minutes"])
}

func TestGroups_Verbs(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	m.on("POST", "/v25.0/111/groups", `{"id":"G1"}`)
	_, _, err := runCmd(t, "groups", "create", "Clientes VIP", "--description", "Ofertas")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	assert.Equal(t, "Clientes VIP", body["subject"])
	assert.Equal(t, "Ofertas", body["description"])

	m.on("GET", "/v25.0/111/groups", `{"data":[{"id":"G1","subject":"Clientes VIP"}]}`)
	_, _, err = runCmd(t, "groups", "list")
	require.NoError(t, err)

	m.on("GET", "/v25.0/G1", `{"id":"G1","subject":"Clientes VIP"}`)
	_, _, err = runCmd(t, "groups", "get", "G1")
	require.NoError(t, err)

	_, _, err = runCmd(t, "groups", "update", "G1", "--subject", "VIP 2")
	require.NoError(t, err)
	assert.Equal(t, "VIP 2", jsonBody(t, m.last())["subject"])

	_, _, err = runCmd(t, "groups", "remove-participants", "G1", "573001", "--yes")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, m.last().Method)
	parts := jsonBody(t, m.last())["participants"].([]any)
	assert.Equal(t, "573001", parts[0].(map[string]any)["user"])

	m.on("GET", "/v25.0/G1/invite_link", `{"invite_link":"https://chat.whatsapp.com/X"}`)
	out, _, err := runCmd(t, "groups", "invite-link", "G1", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "chat.whatsapp.com")

	_, _, err = runCmd(t, "groups", "reset-invite-link", "G1", "--yes")
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, m.last().Method)

	m.on("POST", "/v25.0/111/messages", sendOK)
	_, _, err = runCmd(t, "groups", "send", "G1@g.us", "Hola grupo")
	require.NoError(t, err)
	body = jsonBody(t, m.last())
	assert.Equal(t, "group", body["recipient_type"])
	assert.Equal(t, "G1@g.us", body["to"])

	_, _, err = runCmd(t, "groups", "pin", "G1@g.us", "wamid.P", "--days", "3")
	require.NoError(t, err)
	body = jsonBody(t, m.last())
	assert.Equal(t, "pin", body["type"])
	assert.Equal(t, float64(3), body["pin"].(map[string]any)["expiration_days"])

	_, _, err = runCmd(t, "groups", "unpin", "G1@g.us", "wamid.P")
	require.NoError(t, err)
	body = jsonBody(t, m.last())
	assert.Equal(t, "unpin", body["type"])
	assert.Nil(t, body["pin"].(map[string]any)["expiration_days"])

	_, _, err = runCmd(t, "groups", "delete", "G1", "--yes")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, m.last().Method)
}

func TestMarketing_Send(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/marketing_messages", sendOK)

	_, _, err := runCmd(t, "marketing", "send", "--to", "573001", "--name", "promo", "--lang", "es_MX",
		"--param", "Juan", "--fallback")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	assert.Equal(t, "template", body["type"])
	assert.Equal(t, "CLOUD_API_FALLBACK", body["product_policy"])
	tpl := body["template"].(map[string]any)
	assert.Equal(t, "promo", tpl["name"])
}

func TestListPagination_AllWalksCursors(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	page := 0
	m.handlers["GET /v25.0/222/message_templates"] = func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			assert.Empty(t, r.URL.Query().Get("after"))
			_, _ = w.Write([]byte(`{"data":[{"id":"T1","name":"a"}],"paging":{"cursors":{"after":"C1"},"next":"https://next"}}`))
			return
		}
		assert.Equal(t, "C1", r.URL.Query().Get("after"))
		_, _ = w.Write([]byte(`{"data":[{"id":"T2","name":"b"}],"paging":{}}`))
	}

	out, _, err := runCmd(t, "templates", "list", "--all", "-o", "json")
	require.NoError(t, err)
	assert.Equal(t, 2, page)
	var rows []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &rows))
	assert.Len(t, rows, 2)
}

func TestConfirmPromptRefusesNonTTY(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	// Piped stdin cannot answer a destructive confirmation: the command must refuse with an
	// actionable message and send nothing. A "yes" typed into a pipe is exactly the kind of
	// accidental approval the prompt exists to prevent.
	_, _, err := runCmd(t, "templates", "delete", "promo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
	assert.Empty(t, m.requests, "a refused confirm must not send anything")

	// --yes is the scripted path.
	_, _, err = runCmd(t, "templates", "delete", "promo", "--yes")
	require.NoError(t, err)
	require.NotEmpty(t, m.requests)
}

func TestDryRun_SuppressesRequestAcrossGroups(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	for _, args := range [][]string{
		{"send", "text", "--to", "1", "hola", "--dry-run"},
		{"templates", "delete", "x", "--dry-run"},
		{"phone", "register", "--pin", "123456", "--dry-run"},
	} {
		out, _, err := runCmd(t, args...)
		require.NoError(t, err, "args %v", args)
		assert.Contains(t, out, "curl", "args %v", args)
	}
	assert.Empty(t, m.requests, "--dry-run must never reach the API")
}
