// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/render"
)

type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string   `json:"status"`           // user-level status message
	AppCode    int64    `json:"code,omitempty"`   // application-specific error code
	ErrorText  []string `json:"errors,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func ErrBadRequest(err error) render.Renderer {
	return jsonError(http.StatusBadRequest, err)
}

func ErrForbidden(err error) render.Renderer {
	return jsonError(http.StatusForbidden, err)
}

func ErrNotFound(err error) render.Renderer {
	return jsonError(http.StatusNotFound, err)
}

func ErrUnexpected(err error) render.Renderer {
	return jsonError(http.StatusInternalServerError, err)
}

func jsonError(code int, err error) render.Renderer {
	errList := make([]string, 0)
	for _, line := range strings.Split(err.Error(), "\n") {
		errList = append(errList, line)
	}

	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: code,
		StatusText:     http.StatusText(code),
		ErrorText:      errList,
	}
}
