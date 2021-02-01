package anypoint

import (
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Auth represents the authentication information.
type Auth interface {
	GetToken() string
}

// auth represents the authentication information.
type auth struct {
	token string
}

// NewAuth creates a new authentication token
func NewAuth() (Auth, error) {
	a := &auth{}
	token, err := a.authenticate()

	if err != nil {
		return nil, err
	}
	a.token = token
	return a, nil
}

// GetToken returns the access token
func (a *auth) GetToken() string {
	return a.token
}

func (a *auth) authenticate() (string, error) {
	log.Info("Logging into Mulesoft Anypoint Exchange")
	return "todo", nil
}
