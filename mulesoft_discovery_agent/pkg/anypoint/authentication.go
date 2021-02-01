package anypoint

import "github.com/Axway/agent-sdk/pkg/util/log"

// Auth represents the authentication information.
type Auth interface {
	GetToken() string
}

// auth represents the authentication information.
type auth struct {
	token string
	c     Client
}

// NewAuth creates a new authentication token
func NewAuth(c Client) (Auth, error) {
	a := &auth{}
	token, err := c.GetAccessToken()
	if err != nil {
		return nil, err
	}
	a.token = token

	log.Info("Logged into mulesoft with " + token)
	// TODO: START GO ROUTINE TO KEEP TOKEN FRESH

	return a, nil
}

// GetToken returns the access token
func (a *auth) GetToken() string {
	return a.token
}
