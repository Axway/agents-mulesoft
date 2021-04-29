package anypoint

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	c := &authClient{}
	auth, err := NewAuth(c)
	assert.Nil(t, err)
	assert.NotNil(t, auth)

	org := auth.GetOrgID()
	token := auth.GetToken()

	assert.Equal(t, "1", org)
	assert.Equal(t, "abc123", token)

	auth.Stop()
}

func TestAuthError(t *testing.T) {
	c := &authClientGetErr{}
	auth, err := NewAuth(c)
	assert.NotNil(t, err)
	assert.Nil(t, auth)
}

func Test_startRefreshToken(t *testing.T) {
	client := &authClientRefreshErr{
		stop: make(chan bool),
	}
	a := &auth{
		client: client,
	}
	a.startRefreshToken(1000)
	done := <-client.stop
	assert.True(t, done)
}

type authClientGetErr struct {
}

func (a authClientGetErr) GetAccessToken() (string, *User, time.Duration, error) {
	return "", &User{}, 0, fmt.Errorf("auth error")
}

type authClientRefreshErr struct {
	stop chan bool
}

func (a authClientRefreshErr) GetAccessToken() (string, *User, time.Duration, error) {
	a.stop <- true
	return "", &User{}, 0, fmt.Errorf("auth error")
}

type authClient struct{}

func (a authClient) GetAccessToken() (string, *User, time.Duration, error) {
	return "abc123", &User{
		Email:        "",
		FirstName:    "",
		ID:           "",
		IdentityType: "",
		LastName:     "",
		Organization: Organization{
			Domain: "",
			ID:     "1",
			Name:   "",
		},
		Username: "",
	}, 10, nil
}
