package api

import (
	"errors"
	"fmt"

	"github.com/jellydator/ttlcache/v2"
	"github.com/wneessen/js-mailer/form"
)

// GetForm gets a form.Form object either from the in-memory cache or if not cached
// yet, from the file system
func (r *Route) GetForm(i string) (form.Form, error) {
	var formObj form.Form
	cacheForm, err := r.Cache.Get(fmt.Sprintf("formObj_%s", i))
	if err != nil && !errors.Is(err, ttlcache.ErrNotFound) {
		return formObj, err
	}
	if cacheForm != nil {
		formObj = cacheForm.(form.Form)
	} else {
		formObj, err = form.NewForm(r.Config, i)
		if err != nil {
			return formObj, err
		}
		if err := r.Cache.Set(fmt.Sprintf("formObj_%s", formObj.ID), formObj); err != nil {
			return formObj, err
		}
	}

	return formObj, nil
}
