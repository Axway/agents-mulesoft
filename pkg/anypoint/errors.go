package anypoint

import (
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/Axway/agent-sdk/pkg/util/errors"
)

var (
	ErrCommunicatingWithGateway = errors.New(3005, "could not make request to Mulesoft")
	ErrMarshallingBody          = errors.New(3006, "could not create the body of the request to Mulesoft")
	ErrAuthentication           = errors.New(3401, "authentication failed")
)

// ErrorResponse Error response from Mulesoft
type ErrorResponse struct {
	Code    int
	Message string
}

// NewErrorResponse returns an ErrorResponse struct
func NewErrorResponse(body string, code int) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: gjson.Get(body, "message").String(),
	}
}

// String format the error response
func (er *ErrorResponse) String() string {
	return fmt.Sprintf("%d - %s", er.Code, er.Message)
}
