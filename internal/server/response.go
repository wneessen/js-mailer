// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"net/http"
	"time"

	"github.com/go-chi/render"
)

type Response struct {
	Success    bool          `json:"success"`
	StatusCode int           `json:"statusCode"`
	Status     string        `json:"status"`
	Message    string        `json:"message,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`
	RequestID  string        `json:"requestId,omitempty"`
	Data       any           `json:"data,omitempty"`
	Errors     []ErrorDetail `json:"errors,omitempty"`
}

type ErrorDetail struct {
	Field   string `json:"field,omitempty"` // e.g. "email"
	Code    string `json:"code"`            // e.g. "invalid_format"
	Message string `json:"message"`         // user-facing error text
}

// Render satisfies the go-chi render.Renderer interface.
func (re *Response) Render(_ http.ResponseWriter, r *http.Request) error {
	if re.StatusCode != 0 {
		render.Status(r, re.StatusCode)
	}
	if re.Timestamp.IsZero() {
		re.Timestamp = time.Now().UTC()
	}
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
