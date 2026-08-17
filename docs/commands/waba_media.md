## waba media

Upload, inspect, download and delete media

### Options

```
  -h, --help   help for media
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
* [waba media delete](waba_media_delete)	 - Delete an uploaded media item
* [waba media download](waba_media_download)	 - Download a media item to a file
* [waba media upload](waba_media_upload)	 - Upload a media file, returning its media id
* [waba media url](waba_media_url)	 - Get a media item's download URL and metadata

