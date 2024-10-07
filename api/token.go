package api

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/wneessen/js-mailer/response"
)

// TokenRequest reflects the incoming gettoken request data that for the parameter binding
type TokenRequest struct {
	FormID string `query:"formid" form:"formid"`
}

// TokenResponse reflects the JSON response struct for token request
type TokenResponse struct {
	Token      string `json:"token"`
	FormID     string `json:"form_id"`
	CreateTime int64  `json:"create_time,omitempty"`
	ExpireTime int64  `json:"expire_time,omitempty"`
	URL        string `json:"url"`
	EncType    string `json:"enc_type"`
	Method     string `json:"method"`
}

// GetToken handles the HTTP token requests and return a TokenResponse on success or
// an error on failure
func (r *Route) GetToken(c echo.Context) error {
	fr := &TokenRequest{}
	if err := c.Bind(fr); err != nil {
		c.Logger().Errorf("failed to bind request to TokenRequest object: %s", err)
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// Let's try to read formobj from cache
	formObj, err := r.GetForm(fr.FormID)
	if err != nil {
		c.Logger().Errorf("failed to get form object: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError, &response.ErrorObj{
			Message: "failed to get form object",
			Data:    "invalid form id or form configuration broken",
		})
	}

	// Let's validate the Origin header
	isValid := false
	reqOrigin := c.Request().Header.Get("origin")
	if reqOrigin == "" {
		c.Logger().Errorf("no origin domain set in HTTP request header")
		return echo.NewHTTPError(http.StatusUnauthorized,
			"domain is not authorized to access the requested form")
	}
	for _, d := range formObj.Domains {
		if reqOrigin == d || reqOrigin == fmt.Sprintf("http://%s", d) ||
			reqOrigin == fmt.Sprintf("https://%s", d) {
			isValid = true
		}
	}
	if !isValid {
		c.Logger().Errorf("domain %q not in allowed domains list for form %s", reqOrigin, formObj.ID)
		return echo.NewHTTPError(http.StatusUnauthorized,
			"domain is not authorized to access the requested form")
	}
	c.Response().Header().Set("Access-Control-Allow-Origin", reqOrigin)

	// Generate the token
	reqScheme := "http"
	if c.Request().Header.Get("X-Forwarded-Proto") == "https" || c.Request().TLS != nil {
		reqScheme = "https"
	}
	nowTime := time.Now()
	expTime := time.Now().Add(time.Minute * 10)
	tokenText := fmt.Sprintf("%s_%d_%d_%s_%s", reqOrigin, nowTime.Unix(), expTime.Unix(),
		formObj.ID, formObj.Secret)
	tokenSha := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenText)))
	respToken := TokenResponse{
		Token:      tokenSha,
		FormID:     formObj.ID,
		CreateTime: nowTime.Unix(),
		ExpireTime: expTime.Unix(),
		URL: fmt.Sprintf("%s://%s/api/v1/send/%s/%s", reqScheme,
			c.Request().Host, url.QueryEscape(formObj.ID), url.QueryEscape(tokenSha)),
		EncType: "multipart/form-data",
		Method:  "post",
	}
	if err := r.Cache.Set(tokenSha, respToken); err != nil {
		c.Logger().Errorf("Failed to store response token in cache: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error")
	}

	return c.JSON(http.StatusOK, &response.SuccessResponse{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Data:       respToken,
	})
}
