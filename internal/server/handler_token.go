// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/render"

	"github.com/wneessen/js-mailer/internal/forms"
	"github.com/wneessen/js-mailer/internal/logger"
)

const (
	encodingMPFormData = "multipart/form-data"
)

var ErrDomainNotAllowed = fmt.Errorf("domain not allowed")

// TokenRequest is the JSON request struct for the token endpoint
type TokenRequest struct {
	FormID string `json:"form_id"`
}

// TokenResponse is the JSON response struct for the token endpoint
type TokenResponse struct {
	Token      string `json:"token"`
	FormID     string `json:"form_id"`
	CreateTime int64  `json:"create_time,omitempty"`
	ExpireTime int64  `json:"expire_time,omitempty"`
	URL        string `json:"url"`
	Encoding   string `json:"encoding"`
	ReqMethod  string `json:"request_method"`
}

func (r *TokenRequest) Bind(_ *http.Request) error {
	if r.FormID == "" {
		return fmt.Errorf("form_id is required")
	}
	return nil
}

func (s *Server) HandlerAPITokenPost(w http.ResponseWriter, r *http.Request) {
	data := new(TokenRequest)
	if err := render.Bind(r, data); err != nil {
		s.log.Error("failed to bind request data to TokenRequest", logger.Err(err))
		_ = render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Get the form configuration
	form, err := forms.New(s.config.Forms.Path, data.FormID)
	if err != nil {
		_ = render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Validate that the request is coming from the correct origin
	origin := r.Header.Get("origin")
	if origin == "" {
		_ = render.Render(w, r, ErrUnauthorized(ErrDomainNotAllowed))
		return
	}

	allowedDomain := false
	for _, domain := range form.Domains {
		if strings.EqualFold(origin, fmt.Sprintf("https://%s", domain)) {
			allowedDomain = true
			break
		}
	}
	if !allowedDomain {
		s.log.Error("domain not allowed", slog.String("origin", origin), slog.String("form", form.ID),
			slog.Any("allowed_domains", form.Domains))
		_ = render.Render(w, r, ErrUnauthorized(ErrDomainNotAllowed))
		return
	}

	schema := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		schema = "https"
	}
	now := time.Now()
	expire := now.Add(s.config.Forms.DefaultExpiration)
	value := fmt.Sprintf("%s_%d_%d_%s_%s", origin, now.UnixNano(), expire.UnixNano(), form.ID, form.Secret)
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
	token := &TokenResponse{
		Token:      checksum,
		FormID:     form.ID,
		CreateTime: now.Unix(),
		ExpireTime: expire.Unix(),
		URL: fmt.Sprintf("%s://%s/send/%s/%s", schema, r.Host, url.QueryEscape(form.ID),
			url.QueryEscape(checksum)),
		Encoding:  encodingMPFormData,
		ReqMethod: http.MethodPost,
	}
	s.cache.Set(checksum, token)

	resp := NewResponse(http.StatusCreated, "sender token successfully created", token)
	if renderErr := render.Render(w, r, resp); renderErr != nil {
		s.log.Error("failed to render TokenResposne", logger.Err(renderErr))
	}
}
