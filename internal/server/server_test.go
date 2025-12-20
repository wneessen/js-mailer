// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wneessen/js-mailer/internal/config"
	"github.com/wneessen/js-mailer/internal/logger"
)

const testBasePort = 65000

var testPortInc atomic.Int32

func TestNew(t *testing.T) {
	t.Run("return a server", func(t *testing.T) {
		conf, err := config.New()
		if err != nil {
			t.Fatalf("failed to create config: %s", err)
		}
		log := logger.New(slog.LevelError, "text")
		server := New(conf, log)
		if server == nil {
			t.Fatal("server is nil")
		}
		if server.log != log {
			t.Errorf("expected log to be %p, got %p", log, server.log)
		}
		if server.config != conf {
			t.Errorf("expected config to be %p, got %p", conf, server.config)
		}
	})
}

func TestServer_Start(t *testing.T) {
	t.Run("start and shutdown the server", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			afterFuncCalled := false
			context.AfterFunc(ctx, func() {
				afterFuncCalled = true
			})

			server, err := testServer(t, slog.LevelDebug, io.Discard)
			if err != nil {
				t.Fatalf("failed to create test server: %s", err)
			}

			go func() {
				if err = server.Start(ctx); err != nil {
					t.Errorf("failed to start server: %s", err)
				}
			}()

			cancel()
			synctest.Wait()
			if !afterFuncCalled {
				t.Fatalf("before context is canceled: AfterFunc not called")
			}
		})
	})
	t.Run("starting server with invalid port fails", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.httpSrv.Addr = ":invalid"
		if err = server.Start(context.Background()); err == nil {
			t.Fatal("expected error when starting server with invalid port")
		}
	})
}

func TestServer_HandlerAPIPingGet(t *testing.T) {
	t.Run("ping returns pong", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, os.Stderr)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}

		router := chi.NewRouter()
		router.Get("/ping", server.HandlerAPIPingGet)

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("expected status code %d, got: %d", http.StatusOK, recorder.Code)
		}

		type response struct {
			Success    bool          `json:"success"`
			StatusCode int           `json:"statusCode"`
			Status     string        `json:"status"`
			Message    string        `json:"message,omitempty"`
			Timestamp  time.Time     `json:"timestamp"`
			RequestID  string        `json:"requestId,omitempty"`
			Data       PingResponse  `json:"data,omitempty"`
			Errors     []ErrorDetail `json:"errors,omitempty"`
		}
		body := new(response)
		if err = json.NewDecoder(recorder.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode JSON response: %s", err)
		}

		if !body.Success {
			t.Errorf("expected success, got: %t", body.Success)
		}
		if body.StatusCode != http.StatusOK {
			t.Errorf("expected status code %d, got: %d", http.StatusOK, body.StatusCode)
		}
		if body.Status != http.StatusText(http.StatusOK) {
			t.Errorf("expected status %s, got: %s", http.StatusText(http.StatusOK), body.Status)
		}
		wantMsg := "ping request received"
		if body.Message != wantMsg {
			t.Errorf("expected message %s, got: %s", wantMsg, body.Message)
		}
		if body.Timestamp.IsZero() {
			t.Errorf("expected timestamp to be set, got: %s", body.Timestamp)
		}
		if len(body.Errors) != 0 {
			t.Errorf("expected no errors, got: %v", body.Errors)
		}
		want := "pong"
		if body.Data.Ping != want {
			t.Errorf("expected ping %s, got: %s", want, body.Data.Ping)
		}
	})
}

func TestServer_HandlerAPITokenGet(t *testing.T) {
	t.Run("a token is returned for a valid config", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, os.Stderr)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Get("/token/{formID}", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodGet, "/token/testform_toml", nil)
		req.Header.Set("Origin", "https://example.com")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusCreated {
			t.Errorf("expected status code %d, got: %d", http.StatusCreated, recorder.Code)
		}

		type response struct {
			Success    bool          `json:"success"`
			StatusCode int           `json:"statusCode"`
			Status     string        `json:"status"`
			Message    string        `json:"message,omitempty"`
			Timestamp  time.Time     `json:"timestamp"`
			RequestID  string        `json:"requestId,omitempty"`
			Data       TokenResponse `json:"data,omitempty"`
			Errors     []ErrorDetail `json:"errors,omitempty"`
		}
		body := new(response)
		if err = json.NewDecoder(recorder.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode JSON response: %s", err)
		}
		if !body.Success {
			t.Errorf("expected success, got: %t", body.Success)
		}
		if body.StatusCode != http.StatusCreated {
			t.Errorf("expected status code %d, got: %d", http.StatusCreated, body.StatusCode)
		}
		if body.Status != http.StatusText(http.StatusCreated) {
			t.Errorf("expected status %s, got: %s", http.StatusText(http.StatusCreated), body.Status)
		}
		wantMsg := "sender token successfully created"
		if body.Message != wantMsg {
			t.Errorf("expected message %s, got: %s", wantMsg, body.Message)
		}
		if body.Timestamp.IsZero() {
			t.Errorf("expected timestamp to be set, got: %s", body.Timestamp)
		}
		if body.Data.Token == "" {
			t.Error("expected token to be set")
		}
		if time.Unix(body.Data.CreateTime, 0).IsZero() {
			t.Errorf("expected create time to be set, got: %s", time.Unix(body.Data.CreateTime, 0))
		}
		if time.Unix(body.Data.ExpireTime, 0).IsZero() {
			t.Errorf("expected expire time to be set, got: %s", time.Unix(body.Data.ExpireTime, 0))
		}
		if body.Data.ReqMethod != http.MethodPost {
			t.Errorf("expected request method %s, got: %s", http.MethodPost, body.Data.ReqMethod)
		}
		if body.Data.Encoding != encodingMPFormData {
			t.Errorf("expected encoding %s, got: %s", encodingMPFormData, body.Data.Encoding)
		}
		wantFormID := "contact-form"
		if body.Data.FormID != wantFormID {
			t.Errorf("expected form ID %s, got: %s", wantFormID, body.Data.FormID)
		}
		urlPrefix := "http://example.com/send/contact-form/"
		if !strings.HasPrefix(body.Data.URL, urlPrefix) {
			t.Errorf("expected URL prefix %s, got: %s", urlPrefix, body.Data.URL)
		}
	})
}

func TestResponse_Render(t *testing.T) {
	t.Run("render response without timestamp", func(t *testing.T) {
		req := new(http.Request)
		resp := NewResponse(http.StatusOK, "test", map[string]interface{}{"ping": "pong"})
		resp.Timestamp = time.Time{}
		if err := resp.Render(nil, req); err != nil {
			t.Errorf("failed to render response: %s", err)
		}
		if resp.Timestamp.IsZero() {
			t.Errorf("expected timestamp to be set, got: %s", resp.Timestamp)
		}
	})
}

func testServer(t *testing.T, level slog.Level, output io.Writer) (*Server, error) {
	t.Helper()

	// Create server
	log := logger.NewLogger(level, "json", output)
	conf, err := config.New()
	if err != nil {
		t.Fatalf("failed to create config: %s", err)
	}
	testPortInc.Add(1)
	conf.Server.BindPort = fmt.Sprintf("%d", testBasePort+testPortInc.Load())

	server := New(conf, log)
	if server == nil {
		t.Fatal("server is nil")
	}

	return server, nil
}
