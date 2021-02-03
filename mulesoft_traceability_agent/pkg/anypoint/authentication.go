package anypoint

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Auth represents the authentication information.
type Auth interface {
	Stop()
	GetToken() string
}

// auth represents the authentication information.
type auth struct {
	token    string
	client   Client
	stopChan chan struct{}
}

// NewAuth creates a new authentication token
func NewAuth(client Client) (Auth, error) {
	a := &auth{
		stopChan: make(chan struct{}),
	}
	token, lifetime, err := client.GetAccessToken()
	if err != nil {
		return nil, err
	}
	a.token = token
	a.client = client
	a.startRefreshToken(lifetime)

	return a, nil
}

func (a *auth) Stop() {
	a.stopChan <- struct{}{}
}

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
				token, lifetime, err := a.client.GetAccessToken()
				if err != nil {
					// In an error scenario retry every 10 seconds
					log.Error(err)
					timer = time.NewTimer(10 * time.Second)
					continue
				}
				a.token = token

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
