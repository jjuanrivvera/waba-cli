## waba groups

WhatsApp groups (requires an Official Business Account)

### Options

```
  -h, --help   help for groups
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
* [waba groups create](waba_groups_create)	 - Create a group
* [waba groups delete](waba_groups_delete)	 - Delete a group
* [waba groups get](waba_groups_get)	 - Show one group
* [waba groups invite-link](waba_groups_invite-link)	 - Get the group's invite link
* [waba groups list](waba_groups_list)	 - List the number's active groups
* [waba groups pin](waba_groups_pin)	 - Pin a message in a group (admin only, max 3)
* [waba groups remove-participants](waba_groups_remove-participants)	 - Remove participants from a group
* [waba groups reset-invite-link](waba_groups_reset-invite-link)	 - Revoke the invite link and issue a new one
* [waba groups send](waba_groups_send)	 - Send a text message to a group
* [waba groups unpin](waba_groups_unpin)	 - Unpin a message in a group
* [waba groups update](waba_groups_update)	 - Update a group's subject or description

