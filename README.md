<!--
SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>

SPDX-License-Identifier: MIT
-->

# JS-Mailer - form mailer for JavaScript-based websites

[![Go Report Card](https://goreportcard.com/badge/github.com/wneessen/js-mailer)](https://goreportcard.com/report/github.com/wneessen/js-mailer) [![Build Status](https://api.cirrus-ci.com/github/wneessen/js-mailer.svg)](https://cirrus-ci.com/github/wneessen/js-mailer) <a href="https://ko-fi.com/D1D24V9IX"><img src="https://uploads-ssl.webflow.com/5c14e387dab576fe667689cf/5cbed8a4ae2b88347c06c923_BuyMeACoffee_blue.png" height="20" alt="buy ma a coffee"></a>

JS-Mailer is a simple webservice, that allows JavaScript-based websites to easily send form data, by providing a simple
API that can be accessed via JavaScript `Fetch()` or `XMLHttpRequest`.

## Features

* Single-binary webservice
* Multi-form support
* Multiple recipients per form
* Only display form-fields that are configured for the form in the resulting mail
* Check for required form fields
* Anti-SPAM functionality via built-in, auto-expiring and single-use security token feature
* Anti-SPAM functionality via honeypot fields
* Limit form access to specific domains
* Per-form mail server configuration
* hCaptcha support
* reCaptcha v2 (Checkbox) support
* Turnstile support
* Private Captcha support
* Form field type validation (text, email, number, boolean, matchvalue)
* Confirmation mail to poster
* Custom Reply-To header based on sending mail address

### Planed features

* [ ] Form body templates (possibly HTML)

## Installation

There is a ready-to-use Docker image hosted on Github.

* Download the image:
  ```shell
  $ docker pull ghcr.io/wneessen/js-mailer:main
  ```
* Get your config files in place
* Run the image:
  ```shell
  $ docker run -p 8765:8765 -v /etc/js-mailer:/etc/js-mailer ghcr.io/wneessen/js-mailer:main
  ```

## Configuration

### Server configuration

The service, by default, searches for its configuration file in `$HOME/.config/js-mailer/`. In there it
will look for a file named `js-mailer.ext` where `ext` is one of `json`, `toml` or `yaml`.
The config format is very simple and looks like this:
```toml
[log]
# Log level (slog): -4=DEBUG, 0=INFO, 4=WARN, 8=ERROR
level = 0
format = "json"

[forms]
# Directory where form definitions are stored
path = "/var/lib/app/forms"

# Default expiration for generated forms
default_expiration = "10m"

[server]
# Address and port the HTTP server binds to
address = "127.0.0.1"
port = "8765"

# Cache lifetime for captcha / form responses
cache_lifetime = "10m"

# Request timeout
timeout = "15s"
```

### Form configuration

Each form has its own configuration file. The configuration is searched for in the forms path that has been defined in
the configuration file. The form configuration file must be named `<formid>.ext` where `<formid>` is the form id and
`ext` is one of `json`, `toml` or `yaml`.
Equivalent to the server configuration, the form configuration file format is very simple and looks like this:
```toml
# Unique form identifier
id = "contact_form"

# Domains allowed to submit this form
domains = ["example.com", "www.example.com"]

# Email recipients for form submissions
recipients = ["support@example.com"]

# Sender address used for outgoing emails
sender = "no-reply@example.com"

# Shared secret used for form token generation
secret = "super-secret-value"

# Mail content configuration
[content]
subject = "New contact form submission"
fields = ["name", "email", "message"]

# Confirmation mail configuration
[confirmation]
enabled = true
rcpt_field = "email"
subject = "We received your message"
content = "Thank you for contacting us. We will get back to you shortly."

# Form Reply-To address configuration
[reply_to]
field = "email"

# Mail server configuration
[server]
host = "smtp.example.com"
port = 587
username = "smtp-user"
password = "smtp-password"
force_tls = true
dry_run = false

# Form validation configuration
[validation]
honeypot = "company"

# Form field validation configuration
[[validation.fields]]
name = "name"
required = true
type = "string"

[[validation.fields]]
name = "email"
required = true
type = "email"

[[validation.fields]]
name = "message"
required = true
type = "string"

# Form captcha providers configuration
[validation.hcaptcha]
enabled = false
secret_key = ""

[validation.recaptcha]
enabled = true
secret_key = "recaptcha-secret-key"

[validation.turnstile]
enabled = false
secret_key = ""

[validation.private_captcha]
enabled = false
host = "captcha.internal.example"
api_key = "private-captcha-api-key"
```

## Workflow

`JS-Mailer` follows a two-step workflow. First your JavaScript requests a token from the API using the `/token`
endpoint. If the request is valid and website is authorized to request a token, this endpoint returns all information 
required by your HTML form and JavaScript to submit form data to the API.

Use the values from the response as follows:

- Set the form’s `action` attribute to the value of `data.url`
- Set the form’s `method` attribute to `POST` (from `data.request_method`)
- Set the form’s `enctype` attribute to the value of `data.encoding`

Once the form is submitted, the API validates the sender token, checks all submitted fields against the configured
form validation rules, and—if validation succeeds—delivers the form data to the configured recipients using the
configured mail server.

The sender token is bound to the form (`data.form_id`) and is only valid within the time window defined by
`data.create_time` and `data.expire_time`. Submissions using expired or invalid tokens will be rejected.

### Example response

```json
{
  "success": true,
  "statusCode": 201,
  "status": "Created",
  "message": "sender token successfully created",
  "timestamp": "2025-12-21T17:56:22.867942901Z",
  "data": {
    "token": "cb8620734dd48c81d843be9c70d32b546643e0aff64c79ba195aa90db0b55059",
    "form_id": "test_form",
    "create_time": 1766339782,
    "expire_time": 1766340382,
    "url": "https://jsmailer.example.internal/send/test_form/cb8620734dd48c81d843be9c70d32b546643e0aff64c79ba195aa90db0b55059",
    "encoding": "multipart/form-data",
    "request_method": "POST"
  }
}
```

## API Response Format

All API endpoints return a JSON response that follows a consistent, envelope-based format. This ensures predictable
handling of both successful and failed requests across the entire API.

### Response Object

| Field        | Type                | Description                                                            |
|--------------|---------------------|------------------------------------------------------------------------|
| `success`    | `boolean`           | Indicates whether the request was processed successfully.              |
| `statusCode` | `number`            | HTTP status code associated with the response.                         |
| `status`     | `string`            | Human-readable HTTP status text (e.g. `OK`, `Created`, `Bad Request`). |
| `message`    | `string`            | Optional short description of the result.                              |
| `timestamp`  | `string` (RFC 3339) | Server-side timestamp indicating when the response was generated.      |
| `requestId`  | `string`            | Optional unique identifier for request tracing and debugging.          |
| `data`       | `object`            | Optional endpoint-specific response payload.                           |
| `errors`     | `string[]`          | Optional list of error messages describing why the request failed.     |

### Successful Response

A successful request has the following characteristics:

- `success` is `true`
- `statusCode` is a 2xx HTTP status code
- `data` contains the endpoint-specific payload
- `errors` is omitted

#### Example

```json
{
  "success": true,
  "statusCode": 200,
  "status": "OK",
  "message": "request processed successfully",
  "timestamp": "2025-12-21T18:10:00Z",
  "data": {
    "result": "example"
  }
}
```

### Error Response

A failed request returns structured error information:

- `success` is `false`
- `statusCode` is a 4xx or 5xx HTTP status code
- `errors` contains one or more descriptive error messages
- `data` is omitted

#### Example

```json
{
  "success": false,
  "statusCode": 400,
  "status": "Bad Request",
  "message": "validation failed",
  "timestamp": "2025-12-21T18:11:00Z",
  "errors": [
    "email is required",
    "captcha verification failed"
  ]
}
```

### Client Guidelines

- Always check `success` before processing `data`.
- Use `statusCode` for programmatic error handling.
- Treat `message` as informational and not machine-readable.
- Do not assume optional fields are present.

This unified response format enables consistent client-side handling and simplified API integrations.

## Example implementation

A very basic HTML/JS example implementation for the `JS-Mailer` system can be found in
the [code-example](code-examples/) directory
