// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package cache

import (
	"time"

	"github.com/wneessen/js-mailer/internal/forms"
)

type Cache interface {
	Start()
	Set(string, *forms.Form, ItemParams)
	Get(string) (*forms.Form, ItemParams, error)
	Remove(string) error
	Stop()
}

type ItemParams struct {
	TokenCreatedAt   time.Time
	TokenExpiresAt   time.Time
	RandomFieldName  string
	RandomFieldValue string
}
