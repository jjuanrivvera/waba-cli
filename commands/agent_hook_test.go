package commands

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// The hook is the enforcement layer, so it is tested by actually running it under bash with
// real PreToolUse payloads. Every case below corresponds to a bypass that a naive
// implementation lets through; asserting on the generated text instead of executing it would
// not catch any of them.

// writeHook generates the guard hook for the real command tree and returns its path.
func writeHook(t *testing.T) string {
	t.Helper()
	root := NewRootCmd()
	commands := classifyCommands(root)

	files, err := renderHostConfig("claude-code", guardInput{
		Binary:     "waba",
		ToolPrefix: "waba",
		Blocked:    blockedPaths(commands),
		Approvals:  approvalPaths(commands),
	})
	require.NoError(t, err)

	var hook string
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".sh") {
			hook = f.Content
		}
	}
	require.NotEmpty(t, hook, "no hook script was generated")

	path := filepath.Join(t.TempDir(), "waba-guard.sh")
	require.NoError(t, os.WriteFile(path, []byte(hook), 0o700)) //nolint:gosec // test fixture must be executable
	return path
}

// runHook feeds a payload to the hook and reports whether it denied (exit 2).
func runHook(t *testing.T, hookPath string, payload map[string]any, extraPath string) bool {
	t.Helper()

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	cmd := exec.Command("bash", hookPath)
	cmd.Stdin = strings.NewReader(string(raw))
	if extraPath != "" {
		cmd.Env = append(os.Environ(), "PATH="+extraPath)
	}
	err = cmd.Run()

	if err == nil {
		return false // exit 0 == allow
	}
	var exitErr *exec.ExitError
	if ok := asExitError(err, &exitErr); ok {
		return exitErr.ExitCode() == 2
	}
	t.Fatalf("hook failed to run: %v", err)
	return false
}

func asExitError(err error, target **exec.ExitError) bool {
	if e, ok := err.(*exec.ExitError); ok {
		*target = e
		return true
	}
	return false
}

func bashPayload(command string) map[string]any {
	return map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": command}}
}

func toolPayload(tool string) map[string]any {
	return map[string]any{"tool_name": tool, "tool_input": map[string]any{}}
}

func TestGuardHook_BlocksAndAllows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the hook is a bash script; the host is expected to be POSIX")
	}
	hook := writeHook(t)

	cases := []struct {
		name     string
		payload  map[string]any
		wantDeny bool
	}{
		// --- must DENY ---
		{"plain blocked command", bashPayload("waba templates delete promo"), true},
		{"path-prefixed binary", bashPayload("./bin/waba templates delete promo"), true},
		{"absolute path binary", bashPayload("/usr/local/bin/waba templates delete promo"), true},
		// `delete promo;true` glues the separator to the last arg... but a no-arg destructive
		// command glues it to the VERB: a trailing class of space-or-EOL alone lets it through.
		{"glued separator", bashPayload("waba flows delete 123;true"), true},
		{"quote split", bashPayload(`waba templates de""lete promo`), true},
		{"backslash split", bashPayload(`waba templates de\lete promo`), true},
		{"newline obfuscation", bashPayload("waba templates\ndelete promo"), true},
		{"chained with semicolon", bashPayload("echo hi; waba templates delete promo"), true},
		{"chained with pipe", bashPayload("echo hi | waba templates delete promo"), true},
		{"chained with and", bashPayload("true && waba templates delete promo"), true},
		{"env prefixed", bashPayload("env FOO=1 waba templates delete promo"), true},
		{"group alias spelling", bashPayload("waba tpl delete promo"), true},
		{"phone deregister", bashPayload("waba phone deregister"), true},
		{"flow deprecate", bashPayload("waba flows deprecate 123"), true},
		{"apps unsubscribe", bashPayload("waba apps unsubscribe"), true},
		{"groups remove-participants", bashPayload("waba groups remove-participants 1 573001"), true},
		{"alias set self-weakening", bashPayload("waba alias set x 'templates delete'"), true},
		{"raw api delete", bashPayload("waba api DELETE 111/message_templates"), true},
		{"raw api lowercase delete", bashPayload("waba api delete 111/message_templates"), true},
		{"raw api post", bashPayload("waba api POST 111/messages"), true},
		{"mcp blocked tool exact", toolPayload("mcp__waba__templates_delete"), true},

		// --- must ALLOW ---
		{"read command", bashPayload("waba templates list --status APPROVED"), false},
		{"read command with limit", bashPayload("waba phone list --all"), false},
		{"send is write not blocked", bashPayload("waba send text --to 573001 'hola'"), false},
		// The verb appears only as an argument to another program, not in command position.
		{"blocked verb inside an argument", bashPayload("echo templates delete"), false},
		{"reading a source file named delete", bashPayload("cat templates_delete.go"), false},
		// A GET whose PATH contains "delete" must survive: the method position is what matters.
		{"api GET with delete in path", bashPayload("waba api GET 111/delete-preview"), false},
		// A different binary that merely ends in the guarded name.
		{"different binary suffix", bashPayload("mywaba templates delete promo"), false},
		{"mcp read tool", toolPayload("mcp__waba__templates_list"), false},
		// Near-miss on an MCP tool name: exact matching means this is NOT the blocked tool.
		{"mcp near-miss tool", toolPayload("mcp__waba__templates_delete2"), false},
		{"unrelated command", bashPayload("git status"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runHook(t, hook, tc.payload, "")
			if tc.wantDeny {
				require.True(t, got, "expected DENY for %v", tc.payload)
			} else {
				require.False(t, got, "expected ALLOW for %v", tc.payload)
			}
		})
	}
}

// TestGuardHook_NoJQFallback exercises the branch taken when jq is absent.
//
// The PATH is rebuilt from scratch with symlinks to only the handful of tools the fallback
// needs. Merely prepending an empty directory leaves jq reachable further down PATH, so the
// fallback never runs and the test passes while the branch is broken — the exact flaw that
// hid a fail-open bug in two other CLIs in this fleet.
func TestGuardHook_NoJQFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the hook is a bash script; the host is expected to be POSIX")
	}
	hook := writeHook(t)

	binDir := filepath.Join(t.TempDir(), "strictbin")
	require.NoError(t, os.MkdirAll(binDir, 0o750))

	// Only these are reachable. Notably absent: jq.
	for _, tool := range []string{"cat", "tr", "grep", "sed", "head", "printf", "bash", "command"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			continue // builtins like printf/command need no binary
		}
		_ = os.Symlink(path, filepath.Join(binDir, tool))
	}

	// Prove jq really is unreachable, or the rest of this test means nothing.
	probe := exec.Command("bash", "-c", "command -v jq")
	probe.Env = append(os.Environ(), "PATH="+binDir)
	require.Error(t, probe.Run(), "jq must be unreachable for the fallback branch to be exercised")

	cases := []struct {
		name     string
		payload  map[string]any
		wantDeny bool
	}{
		{"blocked command without jq", bashPayload("waba templates delete promo"), true},
		{"path-prefixed without jq", bashPayload("./bin/waba templates delete promo"), true},
		{"glued separator without jq", bashPayload("waba templates delete promo;true"), true},
		{"raw api delete without jq", bashPayload("waba api DELETE 111/message_templates"), true},
		{"read command without jq", bashPayload("waba templates list"), false},
		{"unrelated command without jq", bashPayload("git status"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runHook(t, hook, tc.payload, binDir)
			if tc.wantDeny {
				require.True(t, got, "expected DENY (no-jq branch) for %v", tc.payload)
			} else {
				require.False(t, got, "expected ALLOW (no-jq branch) for %v", tc.payload)
			}
		})
	}
}

// TestClassifyAPICommands locks the read/write/destroy split so a future change cannot
// quietly reclassify a mutation as a read.
func TestClassifyAPICommands(t *testing.T) {
	commands := classifyCommands(NewRootCmd())
	byPath := map[string]string{}
	for _, c := range commands {
		byPath[c.Path] = c.Class
	}

	expect := map[string]string{
		"send text":                  classWrite,
		"send template":              classWrite,
		"send flow":                  classWrite,
		"messages read":              classWrite,
		"media upload":               classWrite,
		"media url":                  classRead,
		"media delete":               classDestroy,
		"phone list":                 classRead,
		"phone register":             classWrite,
		"phone deregister":           classDestroy,
		"templates list":             classRead,
		"templates create":           classWrite,
		"templates delete":           classDestroy,
		"templates delete-by-id":     classDestroy,
		"templates bulk-delete":      classDestroy,
		"qr delete":                  classDestroy,
		"flows publish":              classWrite,
		"flows deprecate":            classDestroy, // cannot be undone through the API
		"flows delete":               classDestroy,
		"apps unsubscribe":           classDestroy,
		"groups delete":              classDestroy,
		"groups remove-participants": classDestroy,
		"block add":                  classWrite,
		"block list":                 classRead,
		"analytics messaging":        classRead,
		"account get":                classRead,
		"account update":             classWrite,
		"api":                        classDestroy, // the raw escape hatch can issue any method
		"config set":                 classLocal,
		"auth login":                 classLocal,
		"agent guard":                classLocal,
	}
	for path, want := range expect {
		got, ok := byPath[path]
		require.True(t, ok, "command %q is missing from the tree", path)
		require.Equalf(t, want, got, "command %q classified as %q, expected %q", path, got, want)
	}
}

// TestEveryAPICommandIsAnnotated fails the build when a command that talks to the Graph API
// is added without an annotation.
//
// Without this, an unannotated command falls through the classifier to "write" — better than
// "read", but it also means nobody notices the omission, and the next such command might be a
// destructive one that only gets an approval prompt instead of a block.
func TestEveryAPICommandIsAnnotated(t *testing.T) {
	root := NewRootCmd()

	var missing []string
	var visit func(cmd *cobra.Command, path []string)
	visit = func(cmd *cobra.Command, path []string) {
		for _, child := range cmd.Commands() {
			if child.Hidden || child.Name() == "help" {
				continue
			}
			childPath := append(append([]string{}, path...), child.Name())
			if child.Runnable() {
				group := childPath[0]
				if !slices.Contains(localGroups, group) && AnnotationKind(child) == "" {
					missing = append(missing, strings.Join(childPath, " "))
				}
			}
			visit(child, childPath)
		}
	}
	visit(root, nil)

	require.Emptyf(t, missing,
		"these commands talk to the Graph API but carry no read/write/destructive annotation: %s",
		strings.Join(missing, ", "))
}

// TestMCPExcludesSetupCommands locks the MCP tool surface: setup, credential and
// self-management commands must never be reachable as tools, while the API surface must be.
func TestMCPExcludesSetupCommands(t *testing.T) {
	root := NewRootCmd()

	find := func(path ...string) *cobra.Command {
		node := root
		for _, name := range path {
			var next *cobra.Command
			for _, c := range node.Commands() {
				if c.Name() == name {
					next = c
					break
				}
			}
			require.NotNilf(t, next, "command %v not found", path)
			node = next
		}
		return node
	}

	for _, excluded := range [][]string{
		{"auth", "login"}, {"auth", "logout"}, {"config", "set"}, {"config", "use"},
		{"init"}, {"alias", "set"}, {"update"}, {"doctor"}, {"agent", "guard"}, {"completion"},
	} {
		require.Falsef(t, mcpCommandSelector(find(excluded...)),
			"%v must be excluded from the MCP surface", excluded)
	}

	for _, included := range [][]string{
		{"send", "text"}, {"templates", "list"}, {"templates", "delete"}, {"media", "upload"},
		{"phone", "list"}, {"flows", "publish"}, {"analytics", "messaging"}, {"api"},
	} {
		require.Truef(t, mcpCommandSelector(find(included...)),
			"%v must be included in the MCP surface", included)
	}
}

// TestMCPExcludedFlagsCoverProfileAliases locks the flag exclusions: the account selector
// under BOTH spellings, the id overrides, --base-url and --show-token must never appear in a
// tool schema — any of them lets an agent retarget or read credentials.
func TestMCPExcludedFlagsCoverProfileAliases(t *testing.T) {
	for _, f := range []string{"show-token", ProfileFlag, "profile", "base-url", "phone-id", "waba-id", "app-id"} {
		require.Containsf(t, mcpExcludedFlags, f, "flag %q must be excluded from MCP tool schemas", f)
	}
}
