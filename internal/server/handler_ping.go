// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/wneessen/js-mailer/internal/logger"
)

type PingResponse struct {
	Ping string `json:"ping"`
}

func (s *Server) HandlerAPIPingGet(w http.ResponseWriter, r *http.Request) {
	resp := NewResponse(http.StatusOK,
		"ping request received",
		PingResponse{
			Ping: "pong",
		},
	)
	if err := render.Render(w, r, resp); err != nil {
		s.log.Error("failed to render PingResponse", logger.Err(err))
	}
}
