package commands

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sendOK = `{"messaging_product":"whatsapp","contacts":[{"input":"573001","wa_id":"573001"}],"messages":[{"id":"wamid.X1"}]}`

func TestSend_Text(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	out, _, err := runCmd(t, "send", "text", "--to", "573001", "hola", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "wamid.X1")

	body := jsonBody(t, m.last())
	assert.Equal(t, "whatsapp", body["messaging_product"])
	assert.Equal(t, "text", body["type"])
	assert.Equal(t, "573001", body["to"])
	assert.Equal(t, "hola", body["text"].(map[string]any)["body"])
}

func TestSend_TextReplyContext(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "text", "--to", "573001", "--reply-to", "wamid.PREV", "ok")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	assert.Equal(t, "wamid.PREV", body["context"].(map[string]any)["message_id"])
}

func TestSend_MediaVariants(t *testing.T) {
	cases := []struct {
		verb    string
		args    []string
		key     string
		wantErr bool
	}{
		{"image", []string{"--link", "https://x/cat.jpg", "--caption", "gato"}, "image", false},
		{"audio", []string{"--id", "12345"}, "audio", false},
		{"video", []string{"--link", "https://x/v.mp4"}, "video", false},
		{"document", []string{"--id", "9", "--filename", "factura.pdf"}, "document", false},
		{"sticker", []string{"--link", "https://x/s.webp"}, "sticker", false},
		{"image", []string{"--link", "https://x/a.jpg", "--id", "1"}, "", true}, // both refs
		{"image", []string{}, "", true},                                         // neither ref
	}
	for _, tc := range cases {
		t.Run(tc.verb+"_"+strings.Join(tc.args, "_"), func(t *testing.T) {
			m := newMockGraph(t)
			testEnv(t, m)
			m.on("POST", "/v25.0/111/messages", sendOK)

			args := append([]string{"send", tc.verb, "--to", "573001"}, tc.args...)
			_, _, err := runCmd(t, args...)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			body := jsonBody(t, m.last())
			assert.Equal(t, tc.key, body["type"])
			assert.NotNil(t, body[tc.key])
		})
	}
}

func TestSend_Location(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "location", "--to", "573001",
		"--lat", "4.7110", "--lng", "-74.0721", "--name", "Oficina")
	require.NoError(t, err)
	loc := jsonBody(t, m.last())["location"].(map[string]any)
	assert.InDelta(t, 4.7110, loc["latitude"], 1e-9)
	assert.InDelta(t, -74.0721, loc["longitude"], 1e-9)
	assert.Equal(t, "Oficina", loc["name"])
}

func TestSend_Contacts(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "contacts", "--to", "573001",
		"-d", `[{"name":{"formatted_name":"Ana"}}]`)
	require.NoError(t, err)
	contacts := jsonBody(t, m.last())["contacts"].([]any)
	require.Len(t, contacts, 1)
}

func TestSend_Reaction(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "reaction", "--to", "573001", "--message-id", "wamid.Y", "--emoji", "👍")
	require.NoError(t, err)
	r := jsonBody(t, m.last())["reaction"].(map[string]any)
	assert.Equal(t, "wamid.Y", r["message_id"])
	assert.Equal(t, "👍", r["emoji"])
}

func TestSend_TemplateWithParams(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "template", "--to", "573001",
		"--name", "order_update", "--lang", "es_MX", "--param", "Juan", "--param", "#42")
	require.NoError(t, err)
	tpl := jsonBody(t, m.last())["template"].(map[string]any)
	assert.Equal(t, "order_update", tpl["name"])
	assert.Equal(t, "es_MX", tpl["language"].(map[string]any)["code"])
	comps := tpl["components"].([]any)
	require.Len(t, comps, 1)
	params := comps[0].(map[string]any)["parameters"].([]any)
	require.Len(t, params, 2)
	assert.Equal(t, "Juan", params[0].(map[string]any)["text"])
}

func TestSend_TemplateComponentsOverrideParams(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "template", "--to", "573001", "--name", "n", "--lang", "en",
		"--param", "ignored", "--components", `[{"type":"header","parameters":[]}]`)
	require.NoError(t, err)
	tpl := jsonBody(t, m.last())["template"].(map[string]any)
	comps := tpl["components"].([]any)
	assert.Equal(t, "header", comps[0].(map[string]any)["type"])
}

func TestSend_Buttons(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "buttons", "--to", "573001", "Confirm?",
		"--button", "yes:Sí", "--button", "no:No", "--footer", "Rivera Refrigeración")
	require.NoError(t, err)
	inter := jsonBody(t, m.last())["interactive"].(map[string]any)
	assert.Equal(t, "button", inter["type"])
	buttons := inter["action"].(map[string]any)["buttons"].([]any)
	require.Len(t, buttons, 2)
	assert.Equal(t, "Sí", buttons[0].(map[string]any)["reply"].(map[string]any)["title"])
	assert.Equal(t, "Rivera Refrigeración", inter["footer"].(map[string]any)["text"])

	_, _, err = runCmd(t, "send", "buttons", "--to", "573001", "x", "--button", "not-id-title-format-missing-colon")
	require.Error(t, err, "malformed --button must be rejected")
}

func TestSend_List(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "list", "--to", "573001", "Pick one",
		"--button-text", "Menú", "--section", "Servicios",
		"--row", "rev:Revisión:Diagnóstico", "--row", "rep:Reparación")
	require.NoError(t, err)
	inter := jsonBody(t, m.last())["interactive"].(map[string]any)
	assert.Equal(t, "list", inter["type"])
	action := inter["action"].(map[string]any)
	assert.Equal(t, "Menú", action["button"])
	rows := action["sections"].([]any)[0].(map[string]any)["rows"].([]any)
	require.Len(t, rows, 2)
	assert.Equal(t, "Diagnóstico", rows[0].(map[string]any)["description"])
}

func TestSend_CtaURL(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "cta-url", "--to", "573001", "See it",
		"--display-text", "Abrir", "--url", "https://invitas.co/e/x")
	require.NoError(t, err)
	inter := jsonBody(t, m.last())["interactive"].(map[string]any)
	assert.Equal(t, "cta_url", inter["type"])
	params := inter["action"].(map[string]any)["parameters"].(map[string]any)
	assert.Equal(t, "https://invitas.co/e/x", params["url"])
}

func TestSend_Flow(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "flow", "--to", "573001", "Book now",
		"--flow-id", "9999", "--flow-cta", "Agendar", "--flow-token", "tok-1", "--mode", "draft")
	require.NoError(t, err)
	inter := jsonBody(t, m.last())["interactive"].(map[string]any)
	assert.Equal(t, "flow", inter["type"])
	params := inter["action"].(map[string]any)["parameters"].(map[string]any)
	assert.Equal(t, "9999", params["flow_id"])
	assert.Equal(t, "draft", params["mode"])
	assert.Equal(t, "3", params["flow_message_version"])

	_, _, err = runCmd(t, "send", "flow", "--to", "573001", "x", "--flow-cta", "Go")
	require.Error(t, err, "one of --flow-id/--flow-name is required")
}

func TestSend_Interactive_RawPassthrough(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.on("POST", "/v25.0/111/messages", sendOK)

	_, _, err := runCmd(t, "send", "interactive", "--to", "573001",
		"-d", `{"type":"address_message","action":{"name":"address_message"}}`)
	require.NoError(t, err)
	inter := jsonBody(t, m.last())["interactive"].(map[string]any)
	assert.Equal(t, "address_message", inter["type"])
}

func TestSend_MissingPhoneIDIsActionable(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	t.Setenv("WABA_PHONE_NUMBER_ID", "")

	_, _, err := runCmd(t, "send", "text", "--to", "573001", "hola")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--phone-id")
}

func TestSend_GraphErrorSurfacesHint(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)
	m.onStatus("POST", "/v25.0/111/messages", 400,
		`{"error":{"message":"Re-engagement message","type":"OAuthException","code":131047}}`)

	_, _, err := runCmd(t, "send", "text", "--to", "573001", "hola")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "24-hour", "the 131047 hint must reach the user")
}

func TestMessages_ReadAndTyping(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	_, _, err := runCmd(t, "messages", "read", "wamid.ABC")
	require.NoError(t, err)
	body := jsonBody(t, m.last())
	assert.Equal(t, "read", body["status"])
	assert.Equal(t, "wamid.ABC", body["message_id"])
	assert.Nil(t, body["typing_indicator"])

	_, _, err = runCmd(t, "messages", "typing", "wamid.ABC")
	require.NoError(t, err)
	body = jsonBody(t, m.last())
	assert.Equal(t, "text", body["typing_indicator"].(map[string]any)["type"])
}
