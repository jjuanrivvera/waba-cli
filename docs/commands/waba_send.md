## waba send

Send WhatsApp messages

### Options

```
  -h, --help   help for send
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
* [waba send audio](waba_send_audio)	 - Send an audio file or voice note
* [waba send buttons](waba_send_buttons)	 - Send up to 3 reply buttons
* [waba send contacts](waba_send_contacts)	 - Send contact cards
* [waba send cta-url](waba_send_cta-url)	 - Send a call-to-action URL button
* [waba send document](waba_send_document)	 - Send a document (PDF, office files, …)
* [waba send flow](waba_send_flow)	 - Send a WhatsApp Flow
* [waba send image](waba_send_image)	 - Send an image (JPEG/PNG)
* [waba send interactive](waba_send_interactive)	 - Send a raw interactive message
* [waba send list](waba_send_list)	 - Send an interactive list menu
* [waba send location](waba_send_location)	 - Send a location pin
* [waba send reaction](waba_send_reaction)	 - React to a message with an emoji
* [waba send sticker](waba_send_sticker)	 - Send a sticker (WebP)
* [waba send template](waba_send_template)	 - Send an approved message template
* [waba send text](waba_send_text)	 - Send a text message
* [waba send video](waba_send_video)	 - Send a video (MP4/3GP)

