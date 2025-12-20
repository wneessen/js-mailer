// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package httpclient

import (
	"errors"
	"io"
	"log/slog"
	stdhttp "net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/wneessen/js-mailer/internal/logger"
	"github.com/wneessen/js-mailer/internal/testhelper"
)

type testType struct {
	String string  `json:"string"`
	Int    int     `json:"int"`
	Float  float64 `json:"float"`
	Bool   bool    `json:"bool"`
}

const testFile = "../../testdata/testtype.json"

func TestNew(t *testing.T) {
	client := New(logger.New(slog.LevelInfo, "text"))
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
}

func TestClient_Get(t *testing.T) {
	t.Run("getting and serializing JSON should work", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(testFile)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}

		client := New(logger.New(slog.LevelInfo, "text"))
		client.Transport = testhelper.MockRoundTripper{Fn: rtFn}
		query := url.Values{}
		query.Add("key", "value")
		headers := make(map[string]string)
		headers["X-Custom-Header"] = "custom-value"

		target := new(testType)
		response, err := client.Get(t.Context(), "https://example.com", target, query, headers)
		if err != nil {
			t.Fatalf("failed to get JSON response: %s", err)
		}

		if response != 200 {
			t.Errorf("expected status code 200, got %d", response)
		}
		if target.String != "test" {
			t.Errorf("expected target string to be 'test', got %s", target.String)
		}
		if target.Int != 123 {
			t.Errorf("expected target int to be 123, got %d", target.Int)
		}
		if target.Float != 123.456 {
			t.Errorf("expected target float to be 123.456, got %f", target.Float)
		}
		if !target.Bool {
			t.Error("expected target bool to be true")
		}
	})
	t.Run("unmarshalling into non-pointer should fail", func(t *testing.T) {
		client := New(logger.New(slog.LevelInfo, "text"))
		var target testType
		_, err := client.Get(t.Context(), "https://example.com", target, nil, nil)
		if err == nil {
			t.Fatal("expected get to fail")
		}
		if !errors.Is(err, ErrNonPointerTarget) {
			t.Errorf("expected error to be %s, got %s", ErrNonPointerTarget, err)
		}
	})
	t.Run("parsing an invalid url should fail", func(t *testing.T) {
		client := New(logger.New(slog.LevelInfo, "text"))
		target := new(testType)
		_, err := client.Get(t.Context(), "https://example.com/xyz%", target, nil, nil)
		if err == nil {
			t.Fatal("expected get to fail")
		}
		if !strings.Contains(err.Error(), "failed to parse URL") {
			t.Errorf("expected error to contain 'failed to parse URL', got %s", err)
		}
	})
	t.Run("get request fails", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			return nil, errors.New("intentionally failing")
		}

		client := New(logger.New(slog.LevelInfo, "text"))
		client.Transport = testhelper.MockRoundTripper{Fn: rtFn}

		target := new(testType)
		_, err := client.Get(t.Context(), "https://example.com", target, nil, nil)
		if err == nil {
			t.Fatal("expected get request to fail")
		}
	})
	t.Run("getting a nil response", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			return &stdhttp.Response{
				StatusCode: 200,
				Body:       &failReadCloser{},
				Header:     make(stdhttp.Header),
			}, nil
		}

		client := New(logger.NewLogger(slog.LevelInfo, "text", io.Discard))
		client.Transport = testhelper.MockRoundTripper{Fn: rtFn}

		target := new(testType)
		_, err := client.Get(t.Context(), "https://example.com", target, nil, nil)
		if err == nil {
			t.Fatal("expected get request to fail")
		}
	})
}

func TestClient_Post(t *testing.T) {
	t.Run("post request succeeds", func(t *testing.T) {
		rtFn := func(req *stdhttp.Request) (*stdhttp.Response, error) {
			data, err := os.Open(testFile)
			if err != nil {
				t.Fatalf("failed to open JSON response file: %s", err)
			}

			return &stdhttp.Response{
				StatusCode: 200,
				Body:       data,
				Header:     make(stdhttp.Header),
			}, nil
		}

		client := New(logger.New(slog.LevelInfo, "text"))
		client.Transport = testhelper.MockRoundTripper{Fn: rtFn}

		target := new(testType)
		_, err := client.Post(t.Context(), testhelper.TestOnlineAPIURL, target, nil, nil)
		if err != nil {
			t.Fatalf("post request failed: %s", err)
		}
	})
}

type failReadCloser struct{}

func (failReadCloser) Read(p []byte) (int, error) { return len(p), nil }
func (failReadCloser) Close() error               { return errors.New("failed to close") }
