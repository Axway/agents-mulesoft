package anypoint

import (
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/Axway/agent-sdk/pkg/util/errors"
)

// Errors hit communicating with and setting up resources on Mulesoft
var (
	// request and response errors 3001-3099
	ErrUnexpectedResponse       = errors.Newf(3001, "unexpected HTTP response code %v, when communicating with Mulesoft")
	ErrUnexpectedResponsePath   = errors.Newf(3002, "unexpected HTTP response code %v, when communicating with Mulesoft. URL: %s.")
	ErrAPICodeMessage           = errors.Newf(3003, "unexpected Mulesoft response code %s-%s")
	ErrParsingResponse          = errors.New(3004, "could not parse the body of the response from Mulesoft")
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
