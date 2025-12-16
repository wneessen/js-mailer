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
	hCaptchaSolutionField       = "h-captcha-response"
	turnstileSolutionField      = "cf-turnstile-response"

	hCpatchaEndpoint  = "https://hcaptcha.com/siteverify"
	turnstileEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
)

var (
	ErrPrivateCaptchaFailed = errors.New("private captcha validation failed")
	ErrHCaptchaFailed       = errors.New("hCaptcha validation failed")
	ErrTurnstileFailed      = errors.New("turnstile validation failed")
)

func (s *Server) validateCaptcha(ctx context.Context, form *forms.Form, submission map[string][]string, remoteAddr string) error {
	// Private Captcha
	if form.Validation.PrivateCaptcha.Enabled {
		if err := s.privateCaptcha(ctx, form, submission); err != nil {
			return fmt.Errorf("private captcha validation failed: %w", err)
		}
		s.log.Debug("private captcha validation succeeded")
	}

	// HCaptcha
	if form.Validation.Hcaptcha.Enabled {
		if err := s.hCaptcha(ctx, form, submission, remoteAddr); err != nil {
			return fmt.Errorf("hCaptcha validation failed: %w", err)
		}
		s.log.Debug("hCaptcha validation succeeded")
	}

	// Cloudflare Turnstile
	if form.Validation.Turnstile.Enabled {
		if err := s.turnstile(ctx, form, submission, remoteAddr); err != nil {
			return fmt.Errorf("turnstile validation failed: %w", err)
		}
		s.log.Debug("turnstile validation succeeded")
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

func (s *Server) hCaptcha(ctx context.Context, form *forms.Form, submission map[string][]string, remoteAddr string) error {
	type response struct {
		Success     bool     `json:"success"`
		Timestamp   string   `json:"challenge_ts"`
		Hostname    string   `json:"hostname"`
		Credit      bool     `json:"credit"`
		ErrorCodes  []string `json:"error-codes"`
		Score       float64  `json:"score"`
		ScoreReason []string `json:"score_reason"`
	}

	solution, ok := submission[hCaptchaSolutionField]
	if !ok || len(solution) == 0 {
		return fmt.Errorf("missing hCaptcha solution")
	}

	endpoint := hCpatchaEndpoint
	data := url.Values{}
	data.Set("secret", form.Validation.Hcaptcha.SecretKey)
	data.Set("remoteip", remoteAddr)
	data.Set("response", solution[0])

	res := new(response)
	body := strings.NewReader(data.Encode())
	header := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	code, err := s.httpClient.Post(ctx, endpoint, res, body, header)
	if err != nil {
		return fmt.Errorf("failed to verify hCaptcha solution: %w", err)
	}
	if code != http.StatusOK {
		s.log.Error("hCaptcha solution verification failed", slog.Int("status_code", code))
		return ErrHCaptchaFailed
	}

	if !res.Success {
		s.log.Error("hCaptcha solution verification failed", slog.Any("response", res))
		return ErrHCaptchaFailed
	}

	return nil
}

func (s *Server) turnstile(ctx context.Context, form *forms.Form, submission map[string][]string, remoteAddr string) error {
	type response struct {
		Success    bool     `json:"success"`
		Timestamp  string   `json:"challenge_ts"`
		Hostname   string   `json:"hostname"`
		ErrorCodes []string `json:"error-codes"`
		Action     string   `json:"action"`
		CustomData string   `json:"cdata"` // Custom data payload from client-side
		Metadata   struct {
			EphemeralID string `json:"ephemeral_id"` // Device fingerprint ID (Enterprise only)
		}
	}

	solution, ok := submission[turnstileSolutionField]
	if !ok || len(solution) == 0 {
		return fmt.Errorf("missing turnstile solution")
	}

	endpoint := turnstileEndpoint
	data := url.Values{}
	data.Set("response", solution[0])
	data.Set("remoteip", remoteAddr)
	data.Set("secret", form.Validation.Turnstile.SecretKey)

	res := new(response)
	body := strings.NewReader(data.Encode())
	header := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	code, err := s.httpClient.Post(ctx, endpoint, res, body, header)
	if err != nil {
		return fmt.Errorf("failed to verify turnstile solution: %w", err)
	}
	if code != http.StatusOK {
		s.log.Error("turnstile solution verification failed", slog.Int("status_code", code))
		return ErrTurnstileFailed
	}

	if !res.Success {
		s.log.Error("turnstile solution verification failed", slog.Any("response", res))
		return ErrTurnstileFailed
	}

	return nil
}
