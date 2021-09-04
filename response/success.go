package response

// SuccessResponse reflects the default JSON structure of an error response
type SuccessResponse struct {
	StatusCode int         `json:"status_code"`
	Status     string      `json:"status"`
	Data       interface{} `json:"data,omitempty"`
}
