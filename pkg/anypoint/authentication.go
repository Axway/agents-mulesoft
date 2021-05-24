package anypoint

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Auth gets a token and starts a refresh interval.
type Auth interface {
	Start() error
	Stop()
	GetToken() string
	GetOrgID() string
}

type auth struct {
	token    string
	user     *User
	client   TokenClient
	stopChan chan struct{}
	done     bool
}

// NewAuth gets an access token
func NewAuth(client TokenClient) *auth {
	return &auth{
		stopChan: make(chan struct{}),
		client:   client,
	}
}

// Start gets a token and starts a loop to refresh the token
func (a *auth) Start() error {
	token, user, lifetime, err := a.client.GetAccessToken()
	if err != nil {
		return err
	}
	a.token = token
	a.user = user
	a.startRefreshToken(lifetime)
	return nil
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
				a.done = true
				break
			}
		}
	}()
}

// GetToken returns the access token
func (a *auth) GetToken() string {
	return a.token
}

// GetOrgID returns the organization ID of the currently authenticated user.
func (a *auth) GetOrgID() string {
	if a.user != nil {
		return a.user.Organization.ID
	}
	return ""
}
