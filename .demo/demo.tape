# waba demo — rendered with VHS (https://github.com/charmbracelet/vhs).
# No account, signup, keyring or external network: only invented local API data.
# Regenerate from the repository root:
#   go build -o bin/waba ./cmd/waba && go build -o bin/demoserver ./tools/demoserver && vhs .demo/demo.tape
# The shell exits on a failed command rather than recording an error as a result.

Output assets/demo.gif

Set Shell bash
Set FontSize 15
Set Width 1200
Set Height 760
Set Padding 18
Set Theme "Dracula"
Set TypingSpeed 28ms

Hide
Type "set -e -o pipefail"
Enter
Type "export PATH=$PWD/bin:$PATH XDG_CONFIG_HOME=$(mktemp -d)"
Enter
Type "export WABA_ACCESS_TOKEN=demo-access-token"
Enter
Type `demoserver 127.0.0.1:8642 >"$XDG_CONFIG_HOME/demoserver.log" 2>&1 & demo_server_pid=$!`
Enter
Type `trap 'kill "$demo_server_pid" 2>/dev/null || true; rm -rf "$XDG_CONFIG_HOME"' EXIT`
Enter
Type `sleep 1; kill -0 "$demo_server_pid"`
Enter
Type `alias waba='env -i PATH="$PATH" XDG_CONFIG_HOME="$XDG_CONFIG_HOME" WABA_ACCESS_TOKEN="$WABA_ACCESS_TOKEN" WABA_GRAPH_VERSION=v25.0 waba --base-url http://127.0.0.1:8642 --waba-id 900001 --phone-id 900002'`
Enter
Type "clear"
Enter
Show

Sleep 500ms

Type "waba templates list"
Sleep 250ms
Enter
Sleep 2400ms

Type "waba templates list -o json --jq '.[].name'"
Sleep 250ms
Enter
Sleep 2400ms

Type "clear"
Enter
Sleep 300ms

Type "waba send text --to 12025550123 'Your demo parcel is ready!'"
Sleep 250ms
Enter
Sleep 2400ms

Type "waba profile get -o json"
Sleep 250ms
Enter
Sleep 2400ms
