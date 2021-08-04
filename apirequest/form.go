package apirequest

import (
	"fmt"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/wneessen/js-mailer/form"
)

func (a *ApiRequest) GetForm(i string) (form.Form, error) {
	var formObj form.Form
	cacheForm, err := a.Cache.Get(fmt.Sprintf("formObj_%s", i))
	if err != nil && err != ttlcache.ErrNotFound {
		return formObj, err
	}
	if cacheForm != nil {
		formObj = cacheForm.(form.Form)
	} else {
		formObj, err = form.NewForm(a.Config, i)
		if err != nil {
			return formObj, err
		}
		if err := a.Cache.Set(fmt.Sprintf("formObj_%d", formObj.Id), formObj); err != nil {
			return formObj, err
		}
	}

	return formObj, nil
}
