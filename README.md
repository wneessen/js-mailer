# JS-Mailer - Form mailer for JavaScript-based websites

[![Go Report Card](https://goreportcard.com/badge/github.com/wneessen/js-mailer)](https://goreportcard.com/report/github.com/wneessen/js-mailer) [![Build Status](https://api.cirrus-ci.com/github/wneessen/js-mailer.svg)](https://cirrus-ci.com/github/wneessen/js-mailer) <a href="https://ko-fi.com/D1D24V9IX"><img src="https://uploads-ssl.webflow.com/5c14e387dab576fe667689cf/5cbed8a4ae2b88347c06c923_BuyMeACoffee_blue.png" height="20" alt="buy ma a coffee"></a>

JS-Mailer is a simple webservice, that allows JavaScript-based websites to easily send form data, by providing a simple
API that can be accessed via JavaScript `Fetch()` or `XMLHttpRequest`.

## Features

* Single-binary webservice
* Multi-form support
* Multiple recipients per form
* Only display form-fields that are configured in for the form in the resulting mail
* Check for required form fields
* Anti-SPAM functionality via built-in, auto-expiring and single-use security token feature
* Anti-SPAM functionality via honeypot fields
* Limit form access to specific domains
* Per-form mail server configuration
* hCaptcha support
* reCaptcha v2 support
* Turnstile support
* Form field type validation (text, email, number, bool)
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

The server configuration, by default, is searched for in `/etc/js-mailer/js-mailer.json`. The JSON syntax is very basic
and comes with sane defaults.

```json
{
  "forms": {
    "path": "/etc/js-mailer/forms",
    "maxlength": "10M"
  },
  "loglevel": "debug",
  "server": {
    "bind_addr": "0.0.0.0",
    "port": 8765,
    "timeout": "15s"
  }
}
```

* `server (type: struct)`: The struct for the web api configuration
    * `bind_addr (type: string)`: The IP address to bind the web service to
    * `port (type: uint)`: The port for the webservice to listen on
    * `timeout (type: time.Duration)`: The duration a request can take at max.
* `forms (type: struct)`: The struct for the forms configuration
    * `path (type: string)`: The path in which `js-mailer` will look for form configuration JSON files
    * `maxlength (type: string)`: Maximum size of the request body (default: "10M")
* `loglevel (type: string)`: The log level for the web service

### Form configuration

Each form has its own configuration file. The configuration is searched in the forms path and are named by its id. Again
the JSON syntax of the form configuration is very simple, yet flexible.

```json
{
  "id": "test_form",
  "secret": "SuperSecretsString",
  "recipients": [
    "who@cares.net"
  ],
  "sender": "website@example.com",
  "domains": [
    "www.example.com",
    "example.com"
  ],
  "content": {
    "subject": "New message through the www.example.com contact form",
    "fields": [
      "name",
      "email",
      "message"
    ]
  },
  "replyto": {
    "field": "email"
  },
  "confirmation": {
    "enabled": true,
    "rcpt_field": "email",
    "subject": "Thank you for your message",
    "content": "We have received your message via www.example.com and will tough base with you, shortly."
  },
  "validation": {
    "hcaptcha": {
      "enabled": true,
      "secret_key": "0x01234567890"
    },
    "recaptcha": {
      "enabled": true,
      "secret_key": "0x01234567890"
    },
    "turnstile": {
      "enabled": true,
      "secret_key": "0x01234567890"
    },
    "honeypot": "street",
    "fields": [
      {
        "name": "name",
        "type": "text",
        "required": true
      },
      {
        "name": "mail_addr",
        "type": "email",
        "required": true
      },
      {
        "name": "terms_checked",
        "type": "matchval",
        "value": "on",
        "required": true
      }
    ]
  },
  "server": {
    "host": "mail.example.com",
    "port": 25,
    "username": "website@example.com",
    "password": "verySecurePassword",
    "timeout": "5s",
    "force_tls": true
  }
}
```

* `id (type: string)`: The id of the form (will be looked for in the `formid` parameter of the token request)
* `secret (type: string)`: Secret for the form. This will be used for the token generation
* `recipients (type: []string)`: List of recipients, that should receive the mails with the submitted form data
* `domains (type: []string)`: List of origin domains, that are allowed to use this form
* `content (type: struct)`: The struct for the mail content configuration
    * `subject (type: string)`: Subject for the mail notification of the form submission
    * `fields (type: []string)`: List of field names that should show up in the mail notification
* `confirmation (type: struct)`: The struct for the mail confirmail mail configuration
    * `enabled (type: boolean)`: If true, the confirmation mail will be sent
    * `rcpt_field (type: string)`: Name of the form field holding the confirmation mail recipient
    * `subject (type: string)`: Subject for the confirmation mail
    * `content (type: string)`: Content for the confirmation mail
      * `fields (type: []string)`: List of field names that should show up in the mail notification
* `validation (type: struct)`: The struct for the form validation configuration
    * `hcaptcha (type: struct)`: The struct for the forms hCaptcha configuration
        * `enabled (type: bool)`: Enable hCaptcha challenge-response validation
        * `secret_key (type: string)`: Your hCaptcha secret key
    * `recaptcha (type: struct)`: The struct for the forms reCaptcha configuration
        * `enabled (type: bool)`: Enable reCaptcha challenge-response validation
        * `secret_key (type: string)`: Your reCaptcha secret key
    * `turnstile (type: struct)`: The struct for the forms Turnstile configuration
      * `enabled (type: bool)`: Enable Turnstile challenge-response validation
      * `secret_key (type: string)`: Your Turnstile secret key
    * `honeypot (type: string)`: Name of the honeypot field, that is expected to be empty (Anti-SPAM)
    * `fields (type: []struct)`: Array of single field validation configurations
        * `name (type: string)`: Field validation identifier
        * `type (type: string)`: Type of validation to run on field (text, email, nummber, bool)
        * `required (type: boolean)`: If set to true, the field is required
* `replyto (type: struct)`: The struct for the reply to configuration
    * `rcpt_field (type: string)`: Name of the form field holding the reply-to mail sender address
* `server (type: struct)`: The struct for the forms mail server configuration
    * `host (type: string)`: Hostname of the sending mail server
    * `port (type: uint32)`: Port to connect to on the sending mail server
    * `username (type: string)`: Username for the mail server authentication
    * `password (type: string)`: Password for the mail server authentication
    * `timeout (type: duration)`: Timeout duration for the mail server connection
    * `force_tls (type: boolean)`: If set to true, the mail server connection will require mandatory TLS

## Workflow

`JS-Mailer` follows a two-step workflow. First your JavaScript requests a token from the API using the `/api/v1/token`
endpoint. If the request is valid and website is authorized to request a token, the API will respond with a
TokenResponseJson. This holds some data, which needs to be included into your form as hidden inputs. It will also
provide a submission URL endpoint `/api/v1/send/<formid>/<token>` that can be used as action in your form. Once the form
is submitted, the API will then validate that all submitted data is correct and submit the form data to the configured
recipients.

## API responses

The API basically responds with two different types of JSON objects. A `success` response or an `error` response.

### Success response

The succss response JSON struct is very simple:

```json
{
  "status_code": 200,
  "status": "Ok",
  "data": {}
}
```

* `status_code (type: uint32)`: The HTTP status code of the success response
* `status (type: string)`: The HTTP status string of the success response
* `data (type: object)`: An object with abritrary data, based on the type of response

#### Successful token retrieval data object

The `data` object of the success response for a successful token retrieval looks like this:

```json
{
  "token": "5b19fca2b154a2681f8d6014c63b5f81bdfdd01036a64f8a835465ab5247feff",
  "form_id": "test_form",
  "create_time": 1628670201,
  "expire_time": 1628670801,
  "url": "https://jsmailer.example.com/api/v1/send/test_form/5b19fca2b154a2681f8d6014c63b5f81bdfdd01036a64f8a835465ab5247feff",
  "enc_type": "multipart/form-data",
  "method": "post"
}
```

* `token (type: string)`: The security token of this send request
* `form_id (type: string)`: The form id of the current form (for reference or automatic inclusion via JS)
* `create_time (type: int64)`: The epoch timestamp when the token was created
* `expire_time (type: int64)`: The epoch timestamp when the token will expire
* `url (type: string)`: API endpoint to set your form action to
* `enc_type (type: string)`: The enctype for your form
* `method (type: string)`: The method for your form

#### Sent successful data object

The `data` object of the success response for a successfully sent message looks like this:

The API response to a send request (`/api/v1/send/<formid>/<token>`) looks like this:

```json
{
  "form_id": "test_form",
  "send_time": 1628670331,
  "confirmation_sent": true,
  "confirmation_rcpt": "toni.tester@example.com"
}
```

* `form_id (type: string)`: The form id of the current form (for reference)
* `send_time (type: int64)`: The epoch timestamp when the message was sent
* `confirmation_sent (type: boolean)`: Is set to true, if a confirmation was sent successfully
* `confirmation_rcpt (type: string)`: The recipient mail address that the confirmation was sent to

### Error response

The error response JSON struct is also very simple:

```json
{
  "status_code": 404,
  "status": "Not Found",
  "error_message": "Validation failed",
  "error_data": "Not a valid send URL"
}
```

* `status_code (type: uint32)`: The HTTP status code of the success response
* `status (type: string)`: The HTTP status string of the success response
* `error_message (type: string)`: The general error message why this request failed
* `error_data (type: interface{})`: Optional details in addtion to the error message (i. e. missing fields)

## Example implementation

A very basic HTML/JS example implementation for the `JS-Mailer` system can be found in
the [code-example](code-examples/) directory
