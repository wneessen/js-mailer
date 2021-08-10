# JS-Mailer - Form mailer for JavaScript-based websites

[![Go Report Card](https://goreportcard.com/badge/github.com/wneessen/js-mailer)](https://goreportcard.com/report/github.com/wneessen/js-mailer) [![Build Status](https://api.cirrus-ci.com/github/wneessen/js-mailer.svg)](https://cirrus-ci.com/github/wneessen/js-mailer)

JS-Mailer is a simple webservice, that allows JavaScript-based websites to easily send form data, by providing a
simple API that can be accessed via JavaScript `Fetch()` or `XMLHttpRequest`.

## Features
* Single-binary webservice
* Multi-form support
* Multiple recipients per form
* Only display form-fields that are configured in for the form in the resulting mail
* Check for required form fields
* Anti-SPAM functionality via built-in, auto-expiring and single-use security token feature
* Limit form access to specific domains
* Per-form mail server configuration

### Planed features
* Anti-SPAM functionality via honeypot fields
* Form field-type validation
* Form body templates (possibly HTML)
* hCaptcha/gCaptcha support

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
    "api": {
        "bind_addr": "0.0.0.0",
        "port": 8765
    },
    "forms": {
        "path": "/etc/js-mailer/forms",
        "maxlength": 1024000
    },
    "loglevel": "debug"
}
```

* `api (type: struct)`: The struct for the web api configuration
  * `bind_addr (type: string)`: The IP address to bind the web service to
  * `port (type: uint)`: The port for the webservice to listen on
* `forms (type: struct)`: The struct for the forms configuration
    * `path (type: string)`: The path in which `js-mailer` will look for form configuration JSON files
    * `maxlength (type: int64)`: Maximum length in bytes of memory that will be read from the form data HTTP header
* `loglevel (type: string)`: The log level for the web service

### Form configuration
Each form has its own configuration file. The configuration is searched in the forms path and are named by its id.
Again the JSON syntax of the form configuration is very simple, yet flexible.

```json
{
    "id": 1,
    "secret": "SuperSecretsString",
    "recipients": ["who@cares.net"],
    "sender": "website@example.com",
    "domains": ["www.example.com", "example.com"],
    "content": {
        "subject": "New message through the www.example.com contact form",
        "fields": ["name", "email", "message"],
        "required_fields": ["name", "email"]
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
* `id (type: int)`: The id of the form (will be looked for in the `formid` parameter of the submission)
* `secret (type: string)`: Secret for the form. This will be used for the token generation
* `recipients (type: []string)`: List of recipients, that should receive the mails with the submitted form data
* `domains (type: []string)`: List of origin domains, that are allowed to use this form
* `content (type: struct)`: The struct for the mail content configuration
  * `subject (type: string)`: Subject for the mail notification of the form submission
  * `fields (type: []string)`: List of field names that should show up in the mail notification
  * `required_fields (type: []string)`: List of field names that are required to submitted
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
provide a submission URL endpoint `/api/v1/send` that can be used as action in your form. Once the form is submitted,
the API will then validate that all submitted data is correct and submit the form data to the configured recipients.

### API responses
#### Token request
The API response to a token request (`/api/v1/token`) looks like this:

```json
{
  "token": "0587ba3fff63ce2c54af0320b5a2d06612a0200aa139c1c150cbfae8a17084a8",
  "create_time": 1628633657,
  "expire_time": 1628634257,
  "form_id": 1,
  "url": "https://jsmailer.example.com/api/v1/send"
}
```
* `token (type: string)`: The security token that needs to be part of the actual form sending request
* `create_time (type: int64)`: The epoch timestamp when the token was created
* `expire_time (type: int64)`: The epoch timestamp when the token will expire
* `form_id (type: uint)`: The form id of the current form (for reference or automatic inclusion via JS)
* `url (type: string)`: API endpoint to set your form action to

#### Send response

The API response to a send request (`/api/v1/send`) looks like this:
```json
{
  "status_code": 200,
  "success_message": "Message successfully sent",
  "form_id": 1
}
```
* `status_code (type: uint32)`: The HTTP status code
* `success_message (type: string)`: The success message
* `form_id (type: uint)`: The form id of the current form (for reference)

## Example implementation

A very basic HTML/JS example implementation for the `JS-Mailer` system can be found in
the [code-example](code-examples/) directory