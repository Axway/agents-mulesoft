package anypoint

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Auth represents the authentication information.
type Auth interface {
	Stop()
	GetToken() string
	GetOrgID() string
}

// auth represents the authentication information.
type auth struct {
	token    string
	user     *User
	client   Client
	stopChan chan struct{}
}

// NewAuth creates a new authentication token
func NewAuth(client Client) (Auth, error) {
	a := &auth{
		stopChan: make(chan struct{}),
	}
	token, user, lifetime, err := client.GetAccessToken()
	if err != nil {
		return nil, err
	}

	a.token = token
	a.user = user
	a.client = client
	a.startRefreshToken(lifetime)

	return a, nil
}

// Stop terminates the background access token refresh.
func (a *auth) Stop() {
	a.stopChan <- struct{}{}
}

// startRefreshToken starts the background token refresh.
func (a *auth) startRefreshToken(lifetime time.Duration) {
	if lifetime <= 0 {
		return
	}

	// Refresh the token at 75% of lifetime and allow for changing interval
	threshold := 0.75
	interval := time.Duration(float64(lifetime.Nanoseconds()) * threshold)
	timer := time.NewTimer(interval)
	go func() {
		for {
			select {
			case <-timer.C:
				log.Debug("refreshing access token")
				token, user, lifetime, err := a.client.GetAccessToken()
				if err != nil {
					// In an error scenario retry every 10 seconds
					log.Error(err)
					timer = time.NewTimer(10 * time.Second)
					continue
				}
				a.token = token
				a.user = user

				if lifetime <= 0 {
					break
				} else {
					interval = time.Duration(float64(lifetime.Nanoseconds()) * threshold)
					timer = time.NewTimer(interval)
				}

			case <-a.stopChan:
				log.Debug("stopping access token refresh")
				timer.Stop()
				break
			}
		}
	}()
}

// GetToken returns the access token
func (a *auth) GetToken() string {
	return a.token
}

// GetOrgID returns the organization Id of the currently authenticated user.
func (a *auth) GetOrgID() string {
	if a.user != nil {
		return a.user.Organization.ID
	}
	return ""
}
