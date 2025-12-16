// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/wneessen/js-mailer/internal/forms"
	"github.com/wneessen/js-mailer/internal/logger"
)

const formMaxMemory = 32 << 20

var (
	ErrMissingFormIDOrHash            = errors.New("missing form ID or token hash")
	ErrInvalidFormIDOrToken           = errors.New("invalid form ID or token")
	ErrFailedToParseForm              = fmt.Errorf("failed to parse form submission")
	ErrRequiredFieldsValidationFailed = errors.New("required fields validation failed")
)

func (s *Server) HandlerAPISendFormPost(w http.ResponseWriter, r *http.Request) {
	formID := chi.URLParam(r, "formID")
	hash := chi.URLParam(r, "hash")
	if formID == "" || hash == "" {
		_ = render.Render(w, r, ErrBadRequest(ErrMissingFormIDOrHash))
		return
	}
	providedHash, err := hex.DecodeString(hash)
	if err != nil {
		s.log.Error("failed to decode provided form token hash", logger.Err(err))
		_ = render.Render(w, r, ErrBadRequest(ErrInvalidFormIDOrToken))
		return
	}
	if len(providedHash) != sha256.Size {
		s.log.Error("invalid form token hash length", slog.Int("length", len(providedHash)))
		_ = render.Render(w, r, ErrBadRequest(ErrInvalidFormIDOrToken))
		return
	}

	// Make sure the form exists and is valid
	defer s.cache.Remove(hash)
	form, tokenCreatedAt, tokenExpiresAt, err := s.formFromCache(formID, hash)
	if err != nil {
		s.log.Error("failed to validate requested form", logger.Err(err), slog.String("formID", formID),
			slog.String("hash", hash))
		_ = render.Render(w, r, ErrNotFound(ErrInvalidFormIDOrToken))
		return
	}

	// Validate the token
	hasher := sha256.New()
	value := fmt.Sprintf("%s_%d_%d_%s_%s", r.Header.Get("origin"), tokenCreatedAt.UnixNano(),
		tokenExpiresAt.UnixNano(), form.ID, form.Secret)
	hasher.Write([]byte(value))
	computedHash := hasher.Sum(nil)
	if subtle.ConstantTimeCompare(computedHash, providedHash) != 1 {
		s.log.Error("invalid form token", slog.String("formID", formID), slog.String("hash", hash))
		_ = render.Render(w, r, ErrNotFound(ErrInvalidFormIDOrToken))
		return
	}

	// Parse the form submission
	if err = r.ParseMultipartForm(formMaxMemory); err != nil {
		s.log.Error("failed to parse form submission", logger.Err(err))
		_ = render.Render(w, r, ErrUnexpected(ErrFailedToParseForm))
		return
	}

	// Check for honeypot fields
	if form.Validation.Honeypot != "" {
		fails := s.failsHoneypot(form.Validation.Honeypot, r.MultipartForm.Value)
		if fails {
			s.log.Warn("submitted values did not pass honeypot validation")
			_ = render.Render(w, r, ErrNotFound(ErrInvalidFormIDOrToken))
			return
		}
	}

	// Check if required fields are present
	if len(form.Validation.Fields) > 0 {
		fails, missingFields := s.failsRequiredFields(form.Validation.Fields, r.MultipartForm.Value)
		if fails {
			s.log.Warn("submitted values did not pass required field validation")
			errList := []error{ErrRequiredFieldsValidationFailed}
			for field, msg := range missingFields {
				errList = append(errList, fmt.Errorf("%s: %s", field, msg))
			}
			_ = render.Render(w, r, ErrBadRequest(errors.Join(errList...)))
			return
		}
	}

}

// formFromCache returns the form configuration from the cache.
func (s *Server) formFromCache(formID, hash string) (*forms.Form, time.Time, time.Time, error) {
	form, createdAt, expiresAt, ok := s.cache.Get(hash)
	if !ok || form == nil {
		return nil, createdAt, expiresAt, errors.New("form config not found in cache")
	}

	if !strings.EqualFold(formID, form.ID) {
		return nil, createdAt, expiresAt, errors.New("provided form id does not match the form config")
	}

	return form, createdAt, expiresAt, nil
}

// failsHoneypot checks if the submitted values fail the honeypot validation.
func (s *Server) failsHoneypot(honeyField string, values map[string][]string) bool {
	for key, val := range values {
		if strings.EqualFold(key, honeyField) && len(val) > 0 {
			for _, item := range val {
				if item != "" {
					return true
				}
			}
		}
	}
	return false
}

// failsRequiredFields checks if the submitted values fail the required field validation.
func (s *Server) failsRequiredFields(validations []forms.ValidationField, submission map[string][]string) (bool, map[string]string) {
	invalidFields := make(map[string]string)

	for _, field := range validations {
		var value string
		if values, ok := submission[field.Name]; ok {
			value = values[0]
		}
		if field.Required && value == "" {
			s.log.Warn("required field is missing", slog.String("field", field.Name))
			invalidFields[field.Name] = "required field is missing"
			continue
		}

		switch strings.ToLower(field.Type) {
		case "text":
			continue
		case "email":
			_, err := mail.ParseAddress(value)
			if err != nil {
				s.log.Warn("field is not of type email", logger.Err(err), slog.String("field", field.Name),
					slog.String("value", value))
				invalidFields[field.Name] = "field is not of type email"
			}
			continue
		case "number":
			_, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				s.log.Warn("field is not of type number", logger.Err(err), slog.String("field", field.Name),
					slog.String("value", value))
				invalidFields[field.Name] = "field is not of type number"
			}
			continue
		case "bool":
			_, err := strconv.ParseBool(value)
			if err != nil {
				s.log.Warn("field is not of type bool", logger.Err(err), slog.String("field", field.Name),
					slog.String("value", value))
				invalidFields[field.Name] = "field is not of type bool"
			}
			continue
		case "matchval":
			if !strings.EqualFold(field.Value, value) {
				s.log.Warn("field does not match configured value", slog.String("field", field.Name),
					slog.String("want_value", field.Value), slog.String("has_value", value))
				invalidFields[field.Name] = "field does not match configured value"
			}
		default:
			continue
		}
	}

	return len(invalidFields) > 0, invalidFields
}
