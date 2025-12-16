// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/wneessen/js-mailer/internal/forms"
)

const (
	privateCaptchaSolutionField = "private-captcha-solution"
)

var (
	ErrPrivateCaptchaFailed = errors.New("private captcha validation failed")
)

func (s *Server) validateCaptcha(ctx context.Context, form *forms.Form, submission map[string][]string) error {
	// Private Captcha
	if form.Validation.PrivateCaptcha.Enabled {
		if err := s.privateCaptcha(ctx, form, submission); err != nil {
			return fmt.Errorf("private captcha validation failed: %w", err)
		}

	}

	return nil
}

// privateCaptcha verifies the captcha solution against the Private Captcha provider.
func (s *Server) privateCaptcha(ctx context.Context, form *forms.Form, submission map[string][]string) error {
	type response struct {
		Success            bool   `json:"success"`
		ChallengeTimestamp string `json:"timestamp"`
		Origin             string `json:"origin"`
		Code               int    `json:"code"`
	}

	solution, ok := submission[privateCaptchaSolutionField]
	if !ok || len(solution) == 0 {
		return fmt.Errorf("missing private captcha solution")
	}

	endpoint, err := url.Parse(fmt.Sprintf("https://%s/verify", form.Validation.PrivateCaptcha.Host))
	if err != nil {
		return fmt.Errorf("failed to parse private captcha endpoint: %w", err)
	}

	res := new(response)
	body := strings.NewReader(solution[0])
	header := map[string]string{"X-Api-Key": form.Validation.PrivateCaptcha.APIKey}
	code, err := s.httpClient.Post(ctx, endpoint.String(), res, body, header)
	if err != nil {
		return fmt.Errorf("failed to verify private captcha solution: %w", err)
	}
	if code != http.StatusOK {
		s.log.Error("private captcha solution verification failed", slog.Int("status_code", code))
		return ErrPrivateCaptchaFailed
	}

	if !res.Success {
		s.log.Error("private captcha solution verification failed", slog.Any("response", res))
		return ErrPrivateCaptchaFailed
	}

	return nil
}
