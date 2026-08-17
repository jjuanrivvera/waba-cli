package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Host-config generation is pure text, so these tests pin the schema details each host
// silently ignores when wrong: Claude's literal-prefix permissions, Codex's top-level keys,
// OpenCode's singular "permission" map.

func guardFilesFor(t *testing.T, host string) map[string]string {
	t.Helper()
	commands := classifyCommands(NewRootCmd())
	files, err := renderHostConfig(host, guardInput{
		Binary: "waba", ToolPrefix: "waba",
		Blocked: blockedPaths(commands), Approvals: approvalPaths(commands),
	})
	require.NoError(t, err)
	out := map[string]string{}
	for _, f := range files {
		out[f.Path] = f.Content
	}
	return out
}

func TestClaudeCodeConfig_ExactToolNamesNotRegex(t *testing.T) {
	files := guardFilesFor(t, "claude-code")
	settings := files[".claude/settings.json"]
	require.NotEmpty(t, settings)

	var cfg struct {
		Permissions struct {
			Deny []string `json:"deny"`
			Ask  []string `json:"ask"`
		} `json:"permissions"`
		Hooks map[string]any `json:"hooks"`
	}
	require.NoError(t, json.Unmarshal([]byte(settings), &cfg))

	assert.Contains(t, cfg.Permissions.Deny, "Bash(waba templates delete:*)")
	assert.Contains(t, cfg.Permissions.Deny, "mcp__waba__templates_delete")
	assert.Contains(t, cfg.Permissions.Deny, "Bash(waba api DELETE:*)")
	assert.Contains(t, cfg.Permissions.Ask, "Bash(waba send text:*)")
	for _, rule := range append(cfg.Permissions.Deny, cfg.Permissions.Ask...) {
		assert.NotContains(t, rule, ".*", "permission rules are literal prefixes — a regex is dead config: %s", rule)
	}
	require.Contains(t, cfg.Hooks, "PreToolUse")
	assert.Contains(t, files[".claude/hooks/waba-guard.sh"], "exit 2")
}

func TestCodexConfig_TopLevelKeys(t *testing.T) {
	files := guardFilesFor(t, "codex")
	toml := files[".codex/config.toml"]
	require.NotEmpty(t, toml)
	assert.Contains(t, toml, `sandbox_mode = "read-only"`)
	assert.Contains(t, toml, `approval_policy = "on-request"`)
	assert.NotContains(t, toml, "[sandbox]", "an invented table is parsed and silently ignored")
}

func TestOpencodeConfig_SingularPermissionKey(t *testing.T) {
	files := guardFilesFor(t, "opencode")
	raw := files["opencode.json"]
	require.NotEmpty(t, raw)

	var cfg map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &cfg))
	perm, ok := cfg["permission"].(map[string]any)
	require.True(t, ok, `the key is "permission" (singular); the plural is ignored without error`)
	bash, ok := perm["bash"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "deny", bash["waba templates delete *"])
	assert.Equal(t, "deny", bash["waba api DELETE *"])
	assert.Equal(t, "ask", bash["waba send text *"])
	assert.Equal(t, "allow", bash["waba *"])
}

func TestRenderHostConfig_UnknownHost(t *testing.T) {
	_, err := renderHostConfig("emacs", guardInput{})
	require.Error(t, err)
}

func TestAgentGuard_PrintAndWrite(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	// Print mode emits everything to stdout and writes nothing.
	out, errOut, err := runCmd(t, "agent", "guard", "--host", "claude-code")
	require.NoError(t, err)
	assert.Contains(t, out, "waba-guard.sh")
	assert.Contains(t, errOut, "--write")

	// Write mode installs the files into --dir, backing up an existing one.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{}"), 0o600))
	out, errOut, err = runCmd(t, "agent", "guard", "--host", "claude-code", "--write", "--dir", dir)
	require.NoError(t, err)
	assert.Contains(t, out, "wrote")
	assert.Contains(t, errOut, "backed up")
	hook, err := os.ReadFile(filepath.Join(dir, "waba-guard.sh"))
	require.NoError(t, err)
	assert.Contains(t, string(hook), "waba")
	_, err = os.Stat(filepath.Join(dir, "settings.json.bak"))
	require.NoError(t, err, "an existing file must be backed up, not clobbered")
}

func TestAgentGuard_AllWrites(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	out, _, err := runCmd(t, "agent", "guard", "--host", "opencode", "--all-writes")
	require.NoError(t, err)
	// With --all-writes every write is denied too, so no "ask" entries remain.
	start := strings.Index(out, "{")
	require.Greater(t, start, -1)
	var cfg map[string]any
	require.NoError(t, json.Unmarshal([]byte(out[start:]), &cfg))
	bash := cfg["permission"].(map[string]any)["bash"].(map[string]any)
	assert.Equal(t, "deny", bash["waba send text *"])
	for _, v := range bash {
		assert.NotEqual(t, "ask", v)
	}
}

func TestAgentClassify_Renders(t *testing.T) {
	m := newMockGraph(t)
	testEnv(t, m)

	out, _, err := runCmd(t, "agent", "classify", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "templates delete")
	assert.Contains(t, out, "destroy")
}
