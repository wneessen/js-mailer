// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Response struct {
	Success    bool      `json:"success"`
	StatusCode int       `json:"status_code"`
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	RequestID  string    `json:"request_id,omitempty"`
	Data       any       `json:"data,omitempty"`
	Errors     []string  `json:"errors,omitempty"`
}

// Render satisfies the go-chi render.Renderer interface.
func (re *Response) Render(_ http.ResponseWriter, r *http.Request) error {
	if re.StatusCode != 0 {
		render.Status(r, re.StatusCode)
	}
	if re.Timestamp.IsZero() {
		re.Timestamp = time.Now().UTC()
	}
	re.RequestID = middleware.GetReqID(r.Context())
	return nil
}

func NewResponse(code int, msg string, data any) *Response {
	return &Response{
		Success:    true,
		StatusCode: code,
		Status:     http.StatusText(code),
		Message:    msg,
		Timestamp:  time.Now().UTC(),
		Data:       data,
	}
}

func NewErrResponse(code int, err error) render.Renderer {
	errList := append([]string{}, strings.Split(err.Error(), "\n")...)
	return &Response{
		Success:    false,
		StatusCode: code,
		Status:     http.StatusText(code),
		Message:    "request could not be processed",
		Timestamp:  time.Now().UTC(),
		Errors:     errList,
	}
}

func ErrBadRequest(err error) render.Renderer {
	return NewErrResponse(http.StatusBadRequest, err)
}

func ErrForbidden(err error) render.Renderer {
	return NewErrResponse(http.StatusForbidden, err)
}

func ErrNotFound(err error) render.Renderer {
	return NewErrResponse(http.StatusNotFound, err)
}

func ErrUnexpected(err error) render.Renderer {
	return NewErrResponse(http.StatusInternalServerError, err)
}
