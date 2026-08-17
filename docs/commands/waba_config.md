## waba config

Inspect and edit the configuration

### Synopsis

Configuration lives in a YAML file (see 'waba config path') and holds only non-secret
settings: accounts with their WABA / phone number / app ids, the Graph version, and output
defaults. Access tokens are never written here — they live in the OS keyring.

### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
      --account string    named account to use
      --app-id string     Meta app id (overrides the account default)
      --base-url string   override the Graph API base URL
      --columns strings   columns to show in table/csv output
      --dry-run           print the equivalent curl command and send nothing
      --jq string         filter the result through a gojq expression
      --no-color          disable colored output
  -o, --output string     output format: table|json|yaml|csv|id (default "table")
      --phone-id string   business phone number id (overrides the account default)
      --quiet             suppress notes and warnings
      --rps float         client-side request rate limit (requests/second) (default 10)
      --show-token        do not redact credentials in --dry-run output
      --timeout int       per-request timeout in seconds (default 60)
  -v, --verbose           trace requests to stderr
      --waba-id string    WhatsApp Business Account id (overrides the account default)
```

### SEE ALSO

* [waba](waba)	 - WhatsApp Cloud API from the command line
* [waba config list-accounts](waba_config_list-accounts)	 - List configured accounts
* [waba config path](waba_config_path)	 - Print the config file location
* [waba config remove](waba_config_remove)	 - Remove an account from the config (its token stays in the keyring until `auth logout`)
* [waba config set](waba_config_set)	 - Set an account field or a global default
* [waba config use](waba_config_use)	 - Switch the active account
* [waba config view](waba_config_view)	 - Show the resolved configuration

