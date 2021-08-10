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
* Anti-SPAM functionality via built-in token feature
* Limit form access to specific domains
* Per-form mail server configuration

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

TBD