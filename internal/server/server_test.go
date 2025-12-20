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

		body := new(Response)
		if err = json.NewDecoder(recorder.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode JSON response: %s", err)
		}

		//&{25-12-20 12:40:05.672319275 +0000 UTC RequestID: Data:map[ping:pong] Errors:[]}
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
		if body.Data == nil {
			t.Fatal("expected data to be set")
		}
		resp, ok := body.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected data to be of type map[string]interface{}, got: %T", body.Data)
		}
		want := "pong"
		val, found := resp["ping"]
		if !found {
			t.Fatal("expected ping response to be set")
		}
		if val != want {
			t.Errorf("expected ping response to be %s, got: %s", want, val)
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
