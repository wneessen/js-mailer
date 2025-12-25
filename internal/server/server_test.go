// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wneessen/js-mailer/internal/cache"
	"github.com/wneessen/js-mailer/internal/config"
	"github.com/wneessen/js-mailer/internal/forms"
	"github.com/wneessen/js-mailer/internal/logger"
	"github.com/wneessen/js-mailer/internal/testhelper"
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
		server, err := testServer(t, slog.LevelDebug, io.Discard)
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
			Success    bool         `json:"success"`
			StatusCode int          `json:"statusCode"`
			Status     string       `json:"status"`
			Message    string       `json:"message,omitempty"`
			Timestamp  time.Time    `json:"timestamp"`
			RequestID  string       `json:"requestId,omitempty"`
			Data       PingResponse `json:"data,omitempty"`
			Errors     []string     `json:"errors,omitempty"`
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
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Get("/token/{formID}", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodGet, "/token/testform_toml", nil)
		req.TLS = &tls.ConnectionState{}
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
			Errors     []string      `json:"errors,omitempty"`
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
		wantFormID := "testform_toml"
		if body.Data.FormID != wantFormID {
			t.Errorf("expected form ID %s, got: %s", wantFormID, body.Data.FormID)
		}
		urlPrefix := "https://example.com/send/testform_toml/"
		if !strings.HasPrefix(body.Data.URL, urlPrefix) {
			t.Errorf("expected URL prefix %s, got: %s", urlPrefix, body.Data.URL)
		}
	})
	t.Run("a token is not provided to non-allowed domains", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Get("/token/{formID}", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodGet, "/token/testform_toml", nil)
		req.TLS = &tls.ConnectionState{}
		req.Header.Set("Origin", "https://non-allowed.example.com")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusForbidden {
			t.Errorf("expected status code %d, got: %d", http.StatusForbidden, recorder.Code)
		}
	})
	t.Run("a token is not provided when no origin is sent", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Get("/token/{formID}", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodGet, "/token/testform_toml", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusForbidden {
			t.Errorf("expected status code %d, got: %d", http.StatusForbidden, recorder.Code)
		}
	})
	t.Run("a token is not provided when no form ID is set", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Get("/token", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodGet, "/token", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		data := new(Response)
		if err = json.NewDecoder(recorder.Result().Body).Decode(data); err != nil {
			t.Fatalf("failed to decode JSON response: %s", err)
		}
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got: %d", http.StatusBadRequest, recorder.Code)
		}
		if data.Success {
			t.Error("expected request not to succeed")
		}
		if data.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status code %d, got: %d", http.StatusBadRequest, data.StatusCode)
		}
		if data.Status != http.StatusText(http.StatusBadRequest) {
			t.Errorf("expected status %s, got: %s", http.StatusText(http.StatusBadRequest), data.Status)
		}
		wantMsg := "request could not be processed"
		if data.Message != wantMsg {
			t.Errorf("expected message %s, got: %s", wantMsg, data.Message)
		}
		if data.Timestamp.IsZero() {
			t.Error("expected timestamp to be set")
		}
		if len(data.Errors) != 1 {
			t.Errorf("expected an error, got: %d", len(data.Errors))
		}
		if data.Errors[0] != ErrNoFormID.Error() {
			t.Errorf("expected error %s, got: %s", ErrNoFormID.Error(), data.Errors[0])
		}
	})
	t.Run("a token is not provided when no form ID is set", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Get("/token/{formID}", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodGet, "/token/non_existing_form", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		data := new(Response)
		if err = json.NewDecoder(recorder.Result().Body).Decode(data); err != nil {
			t.Fatalf("failed to decode JSON response: %s", err)
		}
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got: %d", http.StatusBadRequest, recorder.Code)
		}
		if len(data.Errors) != 1 {
			t.Errorf("expected an error, got: %d", len(data.Errors))
		}
		wantErr := "form not found"
		if data.Errors[0] != wantErr {
			t.Errorf("expected error %s, got: %s", wantErr, data.Errors[0])
		}
	})
	t.Run("token response contains a random anti-spam field", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Get("/token/{formID}", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodGet, "/token/testform_toml_random", nil)
		req.TLS = &tls.ConnectionState{}
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
			Errors     []string      `json:"errors,omitempty"`
		}
		body := new(response)
		if err = json.NewDecoder(recorder.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode JSON response: %s", err)
		}
		_, params, ok := server.cache.Get(body.Data.Token)
		if !ok {
			t.Error("expected to find form in cache")
		}
		if params.RandomFieldName == "" {
			t.Error("expected random field name to be set")
		}
		if params.RandomFieldValue == "" {
			t.Error("expected random field value to be set")
		}
		want := `<input type="hidden" name="_` + params.RandomFieldName + `" value="` +
			params.RandomFieldValue + `">`
		if body.Data.RandomField == "" {
			t.Error("expected random field to be set in token response")
		}
		if body.Data.RandomField != want {
			t.Errorf("expected random field %s, got: %s", want, body.Data.RandomField)
		}
	})
}

func TestServer_preflightCheck(t *testing.T) {
	t.Run("preflight request is allowed", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Options("/token/{formID}", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodOptions, "/token/testform_toml", nil)
		req.TLS = &tls.ConnectionState{}
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusNoContent {
			t.Errorf("expected status code %d, got: %d", http.StatusNoContent, recorder.Code)
		}
	})
	t.Run("preflight request with non-allowed domain", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Options("/token/{formID}", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodOptions, "/token/testform_toml", nil)
		req.TLS = &tls.ConnectionState{}
		req.Header.Set("Origin", "https://example-is-not-allowed.com")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusForbidden {
			t.Errorf("expected status code %d, got: %d", http.StatusForbidden, recorder.Code)
		}
	})
	t.Run("preflight request with existing config", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Options("/token/{formID}", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodOptions, "/token/testform_does_not_exist", nil)
		req.TLS = &tls.ConnectionState{}
		req.Header.Set("Origin", "https://example.com")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got: %d", http.StatusForbidden, recorder.Code)
		}
	})
	t.Run("preflight request without form id", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Options("/token", server.HandlerAPITokenGet)

		req := httptest.NewRequest(http.MethodOptions, "/token", nil)
		req.TLS = &tls.ConnectionState{}
		req.Header.Set("Origin", "https://example.com")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got: %d", http.StatusForbidden, recorder.Code)
		}
	})
}

func TestServer_HandlerAPISendFormPost(t *testing.T) {
	t.Run("a form is sent successfully", func(t *testing.T) {
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"

		router := chi.NewRouter()
		router.With(server.preflightCheck).Get("/token/{formID}", server.HandlerAPITokenGet)
		router.With(server.preflightCheck).Post("/send/{formID}/{hash}", server.HandlerAPISendFormPost)

		req := httptest.NewRequest(http.MethodGet, "/token/testform_toml", nil)
		req.TLS = &tls.ConnectionState{}
		req.Header.Set("Origin", "https://example.com")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusCreated {
			t.Errorf("expected status code %d, got: %d", http.StatusCreated, recorder.Code)
		}

		type tokenResponse struct {
			Data TokenResponse `json:"data,omitempty"`
		}
		body := new(tokenResponse)
		if err = json.NewDecoder(recorder.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode JSON response: %s", err)
		}

		buf := bytes.NewBuffer(nil)
		writer := multipart.NewWriter(buf)
		_ = writer.WriteField("email", "example@example.com")
		_ = writer.WriteField("message", "this is a test message")
		_ = writer.Close()
		req = httptest.NewRequest(http.MethodPost, body.Data.URL, buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.TLS = &tls.ConnectionState{}
		req.Header.Set("Origin", "https://example.com")
		router.ServeHTTP(recorder, req)

		type sendResponse struct {
			Success    bool         `json:"success"`
			StatusCode int          `json:"statusCode"`
			Status     string       `json:"status"`
			Message    string       `json:"message,omitempty"`
			Timestamp  time.Time    `json:"timestamp"`
			RequestID  string       `json:"requestId,omitempty"`
			Data       SendResponse `json:"data,omitempty"`
			Errors     []string     `json:"errors,omitempty"`
		}
		resp := new(sendResponse)
		if err = json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode JSON response: %s", err)
		}

		if !resp.Success {
			t.Error("expected request to succeed")
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code %d, got: %d", http.StatusOK, resp.StatusCode)
		}
		if resp.Status != http.StatusText(http.StatusOK) {
			t.Errorf("expected status %s, got: %s", http.StatusText(http.StatusOK), resp.Status)
		}
		wantMsg := "form mail successfully delivered"
		if resp.Message != wantMsg {
			t.Errorf("expected message %s, got: %s", wantMsg, resp.Message)
		}
		if resp.Timestamp.IsZero() {
			t.Errorf("expected timestamp to be set, got: %s", resp.Timestamp)
		}
		wantFormID := "contact-form"
		if resp.Data.FormID != wantFormID {
			t.Errorf("expected form ID %s, got: %s", wantFormID, resp.Data.FormID)
		}
		if time.Unix(resp.Data.SentAt, 0).IsZero() {
			t.Errorf("expected sent at to be set, got: %s", time.Unix(resp.Data.SentAt, 0))
		}
		wantStatus := "dry-run succeeded"
		if resp.Data.ConfirmationResponse != wantStatus {
			t.Errorf("expected confirmation response %s, got: %s", wantStatus, resp.Data.ConfirmationResponse)
		}
		if resp.Data.MessageResponse != wantStatus {
			t.Errorf("expected message response %s, got: %s", wantStatus, resp.Data.MessageResponse)
		}
	})
	t.Run("sending form fails on", func(t *testing.T) {
		origin := "https://example.com"
		tokenCreatedAt := time.Now()
		tokenExpiresAt := tokenCreatedAt.Add(time.Hour)

		form, err := forms.New("../../testdata", "testform_toml")
		if err != nil {
			t.Fatalf("failed to create form: %s", err)
		}
		hasher := sha256.New()
		value := fmt.Sprintf("%s_%d_%d_%s_%s", origin, tokenCreatedAt.UnixNano(),
			tokenExpiresAt.UnixNano(), form.ID, form.Secret)
		hasher.Write([]byte(value))
		computedHash := fmt.Sprintf("%x", hasher.Sum(nil))

		tests := []struct {
			name     string
			routerFn func(*Server, chi.Router)
			reqFn    func(string) *http.Request
			code     int
		}{
			{
				"short hash",
				func(server *Server, router chi.Router) {
					router.With(server.preflightCheck).Post("/send/{formID}/{hash}", server.HandlerAPISendFormPost)
				},
				func(string) *http.Request {
					buf := bytes.NewBuffer(nil)
					writer := multipart.NewWriter(buf)
					_ = writer.WriteField("email", "example@example.com")
					_ = writer.WriteField("message", "this is a test message")
					_ = writer.Close()
					req := httptest.NewRequest(http.MethodPost, "/send/testform_toml/9f86d081", buf)
					req.Header.Set("Content-Type", writer.FormDataContentType())
					req.TLS = &tls.ConnectionState{}
					req.Header.Set("Origin", origin)
					return req
				},
				http.StatusBadRequest,
			},
			{
				"invalid hash",
				func(server *Server, router chi.Router) {
					router.With(server.preflightCheck).Post("/send/{formID}/{hash}", server.HandlerAPISendFormPost)
				},
				func(string) *http.Request {
					buf := bytes.NewBuffer(nil)
					writer := multipart.NewWriter(buf)
					_ = writer.WriteField("email", "example@example.com")
					_ = writer.WriteField("message", "this is a test message")
					_ = writer.Close()
					req := httptest.NewRequest(http.MethodPost, "/send/testform_toml/invalidhash", buf)
					req.Header.Set("Content-Type", writer.FormDataContentType())
					req.TLS = &tls.ConnectionState{}
					req.Header.Set("Origin", origin)
					return req
				},
				http.StatusBadRequest,
			},
			{
				"no form id given",
				func(server *Server, router chi.Router) {
					router.With(server.preflightCheck).Post("/send/{hash}", server.HandlerAPISendFormPost)
				},
				func(string) *http.Request {
					buf := bytes.NewBuffer(nil)
					writer := multipart.NewWriter(buf)
					_ = writer.WriteField("email", "example@example.com")
					_ = writer.WriteField("message", "this is a test message")
					_ = writer.Close()
					req := httptest.NewRequest(http.MethodPost, "/send/invalidhash", buf)
					req.Header.Set("Content-Type", writer.FormDataContentType())
					req.TLS = &tls.ConnectionState{}
					req.Header.Set("Origin", origin)
					return req
				},
				http.StatusBadRequest,
			},
			{
				"form does not exist",
				func(server *Server, router chi.Router) {
					router.With(server.preflightCheck).Post("/send/{formID}/{hash}", server.HandlerAPISendFormPost)
				},
				func(string) *http.Request {
					buf := bytes.NewBuffer(nil)
					writer := multipart.NewWriter(buf)
					_ = writer.WriteField("message", "this is a test message")
					_ = writer.WriteField("email", "example@example.com")
					_ = writer.Close()
					req := httptest.NewRequest(http.MethodPost, "/send/form_not_existing/9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", buf)
					req.Header.Set("Content-Type", writer.FormDataContentType())
					req.TLS = &tls.ConnectionState{}
					req.Header.Set("Origin", origin)
					return req
				},
				http.StatusNotFound,
			},
			{
				"hash does not match",
				func(server *Server, router chi.Router) {
					router.With(server.preflightCheck).Post("/send/{formID}/{hash}", server.HandlerAPISendFormPost)
					server.cache.Set("9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", &forms.Form{}, cache.ItemParams{
						TokenCreatedAt: time.Now(),
						TokenExpiresAt: time.Now().Add(time.Minute),
					})
				},
				func(string) *http.Request {
					buf := bytes.NewBuffer(nil)
					writer := multipart.NewWriter(buf)
					_ = writer.WriteField("message", "this is a test message")
					_ = writer.WriteField("email", "example@example.com")
					_ = writer.Close()
					req := httptest.NewRequest(http.MethodPost, "/send/testform_toml/9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", buf)
					req.Header.Set("Content-Type", writer.FormDataContentType())
					req.TLS = &tls.ConnectionState{}
					req.Header.Set("Origin", origin)
					return req
				},
				http.StatusNotFound,
			},
			{
				"no submission data",
				func(server *Server, router chi.Router) {
					router.With(server.preflightCheck).Post("/send/{formID}/{hash}", server.HandlerAPISendFormPost)
				},
				func(hash string) *http.Request {
					req := httptest.NewRequest(http.MethodPost, "/send/testform_toml/"+hash, nil)
					req.TLS = &tls.ConnectionState{}
					req.Header.Set("Origin", origin)
					return req
				},
				http.StatusInternalServerError,
			},
			{
				"honeypot triggers",
				func(server *Server, router chi.Router) {
					router.With(server.preflightCheck).Post("/send/{formID}/{hash}", server.HandlerAPISendFormPost)
				},
				func(hash string) *http.Request {
					buf := bytes.NewBuffer(nil)
					writer := multipart.NewWriter(buf)
					_ = writer.WriteField("message", "this is a test message")
					_ = writer.WriteField("email", "example@example.com")
					_ = writer.WriteField("company", "Honeypot Inc.")
					_ = writer.Close()
					req := httptest.NewRequest(http.MethodPost, "/send/testform_toml/"+hash, buf)
					req.Header.Set("Content-Type", writer.FormDataContentType())
					req.TLS = &tls.ConnectionState{}
					req.Header.Set("Origin", origin)
					return req
				},
				http.StatusNotFound,
			},
			{
				"required fields missing",
				func(server *Server, router chi.Router) {
					router.With(server.preflightCheck).Post("/send/{formID}/{hash}", server.HandlerAPISendFormPost)
				},
				func(hash string) *http.Request {
					buf := bytes.NewBuffer(nil)
					writer := multipart.NewWriter(buf)
					_ = writer.Close()
					req := httptest.NewRequest(http.MethodPost, "/send/testform_toml/"+hash, buf)
					req.Header.Set("Content-Type", writer.FormDataContentType())
					req.TLS = &tls.ConnectionState{}
					req.Header.Set("Origin", origin)
					return req
				},
				http.StatusBadRequest,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server, err := testServer(t, slog.LevelDebug, io.Discard)
				if err != nil {
					t.Fatalf("failed to create test server: %s", err)
				}
				server.config.Forms.Path = "../../testdata"
				server.cache.Set(computedHash, form, cache.ItemParams{
					TokenCreatedAt: tokenCreatedAt,
					TokenExpiresAt: tokenExpiresAt,
				})

				router := chi.NewRouter()
				tt.routerFn(server, router)
				req := tt.reqFn(computedHash)
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, req)
				if recorder.Code != tt.code {
					t.Errorf("expected status code %d, got: %d", tt.code, recorder.Code)
				}
			})
		}
	})
	t.Run("too fast submission fails", func(t *testing.T) {
		origin := "https://example.com"
		tokenCreatedAt := time.Now()
		tokenExpiresAt := tokenCreatedAt.Add(time.Hour)

		form, err := forms.New("../../testdata", "testform_toml")
		if err != nil {
			t.Fatalf("failed to create form: %s", err)
		}
		form.Validation.DisableSubmissionSpeedCheck = false
		hasher := sha256.New()
		value := fmt.Sprintf("%s_%d_%d_%s_%s", origin, tokenCreatedAt.UnixNano(),
			tokenExpiresAt.UnixNano(), form.ID, form.Secret)
		hasher.Write([]byte(value))
		computedHash := fmt.Sprintf("%x", hasher.Sum(nil))
		server, err := testServer(t, slog.LevelDebug, io.Discard)
		if err != nil {
			t.Fatalf("failed to create test server: %s", err)
		}
		server.config.Forms.Path = "../../testdata"
		server.cache.Set(computedHash, form, cache.ItemParams{
			TokenCreatedAt: tokenCreatedAt,
			TokenExpiresAt: tokenExpiresAt,
		})

		router := chi.NewRouter()
		router.With(server.preflightCheck).Post("/send/{formID}/{hash}", server.HandlerAPISendFormPost)
		buf := bytes.NewBuffer(nil)
		writer := multipart.NewWriter(buf)
		_ = writer.Close()
		req := httptest.NewRequest(http.MethodPost, "/send/testform_toml/"+computedHash, buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.TLS = &tls.ConnectionState{}
		req.Header.Set("Origin", origin)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusTooEarly {
			t.Errorf("expected status code %d, got: %d", http.StatusTooEarly, recorder.Code)
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

func TestServer_failsRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		validations []forms.ValidationField
		submission  map[string][]string
		fails       bool
	}{
		{
			"text passes through",
			[]forms.ValidationField{{Name: "text", Type: "text", Required: true}},
			map[string][]string{"text": {"hello world"}},
			false,
		},
		{
			"no valid email format",
			[]forms.ValidationField{{Name: "email", Type: "email", Required: true}},
			map[string][]string{"email": {"not@val."}},
			true,
		},
		{
			"valid email format",
			[]forms.ValidationField{{Name: "email", Type: "email", Required: true}},
			map[string][]string{"email": {"valid@example.com"}},
			false,
		},
		{
			"not a number",
			[]forms.ValidationField{{Name: "number", Type: "number", Required: true}},
			map[string][]string{"number": {"text"}},
			true,
		},
		{
			"is a number",
			[]forms.ValidationField{
				{Name: "number", Type: "number", Required: true},
				{Name: "number2", Type: "number", Required: true},
				{Name: "number3", Type: "number", Required: true},
			},
			map[string][]string{
				"number":  {"1"},
				"number2": {"-42"},
				"number3": {"3.14"},
			},
			false,
		},
		{
			"not a boolean",
			[]forms.ValidationField{{Name: "bool", Type: "bool", Required: true}},
			map[string][]string{"bool": {"text"}},
			true,
		},
		{
			"boolean is valid",
			[]forms.ValidationField{
				{Name: "bool", Type: "bool", Required: true},
				{Name: "bool2", Type: "bool", Required: true},
				{Name: "bool3", Type: "bool", Required: true},
			},
			map[string][]string{
				"bool":  {"1"},
				"bool2": {"true"},
				"bool3": {"on"},
			},
			false,
		},
		{
			"value does not match",
			[]forms.ValidationField{{Name: "text", Type: "matchval", Required: true, Value: "expected"}},
			map[string][]string{"text": {"unexpected"}},
			true,
		},
		{
			"value matches",
			[]forms.ValidationField{{Name: "text", Type: "matchval", Required: true, Value: "expected"}},
			map[string][]string{"text": {"expected"}},
			false,
		},
	}

	t.Run("checking required fields", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server, err := testServer(t, slog.LevelDebug, io.Discard)
				if err != nil {
					t.Fatalf("failed to create test server: %s", err)
				}

				fails, _ := server.failsRequiredFields(tt.validations, tt.submission)
				if fails != tt.fails {
					t.Errorf("expected fails to be %t, got: %t", tt.fails, fails)
				}
			})
		}
	})
}

func TestServer_validateCaptcha(t *testing.T) {
	tests := []struct {
		name       string
		captchaFn  func(*http.Request) (*http.Response, error)
		formFn     func(*forms.Form)
		submission map[string][]string
		succeeds   bool
	}{
		{
			"private captcha successful response",
			testResponseFromFile(t, "../../testdata/private_captcha_success.json", http.StatusOK, false),
			func(form *forms.Form) {
				form.Validation.PrivateCaptcha.Enabled = true
			},
			map[string][]string{
				privateCaptchaSolutionField: {"private_captcha_token"},
			},
			true,
		},
		{
			"private captcha failure response",
			testResponseFromFile(t, "../../testdata/private_captcha_failure.json", http.StatusUnauthorized, false),
			func(form *forms.Form) {
				form.Validation.PrivateCaptcha.Enabled = true
			},
			map[string][]string{
				privateCaptchaSolutionField: {"private_captcha_token"},
			},
			false,
		},
		{
			"private captcha http request errors",
			testResponseFromFile(t, "../../testdata/private_captcha_failure.json", http.StatusUnauthorized, true),
			func(form *forms.Form) {
				form.Validation.PrivateCaptcha.Enabled = true
			},
			map[string][]string{
				privateCaptchaSolutionField: {"private_captcha_token"},
			},
			false,
		},
		{
			"private captcha succeeds but is negative",
			testResponseFromFile(t, "../../testdata/private_captcha_success_but_negative.json", http.StatusOK, false),
			func(form *forms.Form) {
				form.Validation.PrivateCaptcha.Enabled = true
			},
			map[string][]string{
				privateCaptchaSolutionField: {"private_captcha_token"},
			},
			false,
		},
		{
			"private captcha without challange field",
			testResponseFromFile(t, "../../testdata/private_captcha_success.json", http.StatusUnauthorized, false),
			func(form *forms.Form) {
				form.Validation.PrivateCaptcha.Enabled = true
			},
			map[string][]string{},
			false,
		},
		{
			"private captcha with invalid endpoint",
			testResponseFromFile(t, "../../testdata/private_captcha_success.json", http.StatusUnauthorized, false),
			func(form *forms.Form) {
				form.Validation.PrivateCaptcha.Enabled = true
				form.Validation.PrivateCaptcha.Host = "invalid%"
			},
			map[string][]string{
				privateCaptchaSolutionField: {"private_captcha_token"},
			},
			false,
		},
		{
			"hCaptcha successful response",
			testResponseFromFile(t, "../../testdata/hcaptcha_success.json", http.StatusOK, false),
			func(form *forms.Form) {
				form.Validation.Hcaptcha.Enabled = true
			},
			map[string][]string{
				hCaptchaSolutionField: {"hcaptcha_token"},
			},
			true,
		},
		{
			"hCaptcha successful request but negative solution",
			testResponseFromFile(t, "../../testdata/hcaptcha_success_but_negative.json", http.StatusOK, false),
			func(form *forms.Form) {
				form.Validation.Hcaptcha.Enabled = true
			},
			map[string][]string{
				hCaptchaSolutionField: {"hcaptcha_token"},
			},
			false,
		},
		{
			"hCaptcha failure response",
			testResponseFromFile(t, "../../testdata/hcaptcha_failure.json", http.StatusUnauthorized, false),
			func(form *forms.Form) {
				form.Validation.Hcaptcha.Enabled = true
			},
			map[string][]string{
				hCaptchaSolutionField: {"hcaptcha_token"},
			},
			false,
		},
		{
			"hCaptcha fails http request",
			testResponseFromFile(t, "../../testdata/hcaptcha_failure.json", http.StatusUnauthorized, true),
			func(form *forms.Form) {
				form.Validation.Hcaptcha.Enabled = true
			},
			map[string][]string{
				hCaptchaSolutionField: {"hcaptcha_token"},
			},
			false,
		},
		{
			"hCaptcha solution field is missing",
			testResponseFromFile(t, "../../testdata/hcaptcha_failure.json", http.StatusUnauthorized, false),
			func(form *forms.Form) {
				form.Validation.Hcaptcha.Enabled = true
			},
			map[string][]string{},
			false,
		},
		{
			"turnstile successful response",
			testResponseFromFile(t, "../../testdata/turnstile_success.json", http.StatusOK, false),
			func(form *forms.Form) {
				form.Validation.Turnstile.Enabled = true
			},
			map[string][]string{
				turnstileSolutionField: {"turnstile_token"},
			},
			true,
		},
		{
			"turnstile successful request but negative solution",
			testResponseFromFile(t, "../../testdata/turnstile_success_but_negative.json", http.StatusOK, false),
			func(form *forms.Form) {
				form.Validation.Turnstile.Enabled = true
			},
			map[string][]string{
				turnstileSolutionField: {"turnstile_token"},
			},
			false,
		},
		{
			"turnstile failure response",
			testResponseFromFile(t, "../../testdata/turnstile_failure.json", http.StatusUnauthorized, false),
			func(form *forms.Form) {
				form.Validation.Turnstile.Enabled = true
			},
			map[string][]string{
				turnstileSolutionField: {"turnstile_token"},
			},
			false,
		},
		{
			"turnstile fails http request",
			testResponseFromFile(t, "../../testdata/turnstile_failure.json", http.StatusUnauthorized, true),
			func(form *forms.Form) {
				form.Validation.Turnstile.Enabled = true
			},
			map[string][]string{
				turnstileSolutionField: {"turnstile_token"},
			},
			false,
		},
		{
			"turnstile solution field is missing",
			testResponseFromFile(t, "../../testdata/turnstile_failure.json", http.StatusUnauthorized, false),
			func(form *forms.Form) {
				form.Validation.Turnstile.Enabled = true
			},
			map[string][]string{},
			false,
		},
		{
			"reCaptcha successful response",
			testResponseFromFile(t, "../../testdata/recaptcha_success.json", http.StatusOK, false),
			func(form *forms.Form) {
				form.Validation.Recaptcha.Enabled = true
			},
			map[string][]string{
				reCaptchaSolutionField: {"reCaptcha_token"},
			},
			true,
		},
		{
			"reCaptcha successful request but negative solution",
			testResponseFromFile(t, "../../testdata/recaptcha_success_but_negative.json", http.StatusOK, false),
			func(form *forms.Form) {
				form.Validation.Recaptcha.Enabled = true
			},
			map[string][]string{
				reCaptchaSolutionField: {"reCaptcha_token"},
			},
			false,
		},
		{
			"reCaptcha failure response",
			testResponseFromFile(t, "../../testdata/recaptcha_failure.json", http.StatusUnauthorized, false),
			func(form *forms.Form) {
				form.Validation.Recaptcha.Enabled = true
			},
			map[string][]string{
				reCaptchaSolutionField: {"reCaptcha_token"},
			},
			false,
		},
		{
			"reCaptcha fails http request",
			testResponseFromFile(t, "../../testdata/recaptcha_failure.json", http.StatusUnauthorized, true),
			func(form *forms.Form) {
				form.Validation.Recaptcha.Enabled = true
			},
			map[string][]string{
				reCaptchaSolutionField: {"reCaptcha_token"},
			},
			false,
		},
		{
			"reCaptcha solution field is missing",
			testResponseFromFile(t, "../../testdata/recaptcha_failure.json", http.StatusUnauthorized, false),
			func(form *forms.Form) {
				form.Validation.Recaptcha.Enabled = true
			},
			map[string][]string{},
			false,
		},
	}

	remoteAddr := "127.0.0.1"
	t.Run("validate captcha against", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server, err := testServer(t, slog.LevelDebug, io.Discard)
				if err != nil {
					t.Fatalf("failed to create test server: %s", err)
				}
				form, err := forms.New("../../testdata", "testform_toml")
				if err != nil {
					t.Fatalf("failed to load form: %s", err)
				}
				tt.formFn(form)

				server.httpClient.Transport = testhelper.MockRoundTripper{Fn: tt.captchaFn}
				if err = server.validateCaptcha(t.Context(), form, tt.submission, remoteAddr); err != nil && tt.succeeds {
					t.Errorf("captcha validation failed: %s", err)
				}
			})
		}
	})
}

func TestServer_csvFromFields(t *testing.T) {
	server, err := testServer(t, slog.LevelDebug, os.Stderr)
	if err != nil {
		t.Fatalf("failed to create test server: %s", err)
	}
	req := newMultipartRequest(t, map[string][]string{
		"name":  {"Toni Tester"},
		"email": {"toni.tester@example.com"},
		"tags":  {"csv"},
	})
	if err = req.ParseMultipartForm(20 << 32); err != nil {
		t.Fatalf("failed to parse multipart form: %s", err)
	}

	t.Run("CSV is generated from a form submission", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		if err = server.csvFromFields(buf, req); err != nil {
			t.Errorf("failed to generate CSV: %s", err)
		}

		reader := csv.NewReader(buf)
		reader.Comma = ';'
		header, err := reader.Read()
		if err != nil {
			t.Fatalf("failed to read CSV header: %s", err)
		}

		found := 0
		wantHeader := []string{"name", "email", "tags"}
		for i := range header {
			for j := range wantHeader {
				if header[i] == wantHeader[j] {
					found++
				}
			}
		}
		if found != len(wantHeader) {
			t.Errorf("expected header to contain %d fields, got: %d", len(wantHeader), found)
		}

		found = 0
		row, err := reader.Read()
		if err != nil {
			t.Fatalf("failed to read CSV row: %s", err)
		}
		wantRow := []string{"Toni Tester", "toni.tester@example.com", "csv"}
		for i := range row {
			for j := range wantRow {
				if row[i] == wantRow[j] {
					found++
				}
			}
		}
		if found != len(wantRow) {
			t.Errorf("expected row to contain %d fields, got: %d", len(wantRow), found)
		}
	})
	t.Run("CSV generation fails with broken writer on first write", func(t *testing.T) {
		writer := new(failWriter)
		writer.maxBytes = 50
		if err = server.csvFromFields(writer, req); err == nil {
			t.Error("expected CSV generation to fail")
		}
	})
	t.Run("CSV generation fails with broken writer on 2nd write", func(t *testing.T) {
		writer := new(failWriter)
		writer.maxBytes = 1
		if err = server.csvFromFields(writer, req); err == nil {
			t.Error("expected CSV generation to fail")
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

func testResponseFromFile(t *testing.T, filename string, code int, fails bool) func(req *http.Request) (*http.Response, error) {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file: %s", err)
	}
	buf := bytes.NewBuffer(data)
	return func(req *http.Request) (*http.Response, error) {
		var resErr error
		if fails {
			resErr = errors.New("intentionally failing")
		}
		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(buf),
			Header:     make(http.Header),
		}, resErr
	}
}

func newMultipartRequest(t *testing.T, fields map[string][]string) *http.Request {
	t.Helper()

	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	for name, values := range fields {
		for _, v := range values {
			if err := writer.WriteField(name, v); err != nil {
				t.Fatalf("failed to write field to multipart form: %q: %s", name, err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %s", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/submit", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

type failWriter struct {
	maxBytes int
	written  int
}

func (w *failWriter) Write(p []byte) (n int, err error) {
	if w.written+len(p) > w.maxBytes {
		return 0, io.ErrShortWrite
	}
	w.written += len(p)
	return len(p), nil
}
