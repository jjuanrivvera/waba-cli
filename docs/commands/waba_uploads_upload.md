## waba uploads upload

Upload the file bytes, returning the media handle

### Synopsis

Uploads the file into the session and prints the handle (`h`), which is what
template creation takes as header_handle and `waba profile update` as --picture-handle.
If a previous attempt was interrupted, `waba uploads status` tells the resume offset.

```
waba uploads upload <session-id> <file> [flags]
```

### Examples

```
  waba uploads upload "upload:MTphdHRh..." ./header.png
  waba uploads upload "upload:MTphdHRh..." ./header.png --offset 1048576
```

### Options

```
  -h, --help                         help for upload
      --offset waba uploads status   byte offset to resume from (from waba uploads status)
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

* [waba uploads](waba_uploads)	 - Resumable uploads (template header media, profile pictures)

