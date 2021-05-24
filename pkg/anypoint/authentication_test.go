package anypoint

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	tests := []struct {
		name  string
		token string
		err   error
	}{
		{
			name:  "should return no token and an error",
			token: "",
			err:   fmt.Errorf("no token here"),
		},
	}
	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			c := &authClient{
				token:    "",
				user:     nil,
				duration: 0,
				err:      fmt.Errorf("no token here"),
			}

			auth := NewAuth(c)
			err := auth.Start()
			assert.Equal(t, tc.err, err)
			assert.NotNil(t, auth)
			org := auth.GetOrgID()
			token := auth.GetToken()
			assert.Equal(t, "", org)
			assert.Equal(t, tc.token, token)
			go auth.Stop()
			<-auth.stopChan
		})
	}

}

func Test_Start(t *testing.T) {
	client := &tokenClient{}
	a := NewAuth(client)
	go a.Start()
	a.Stop()
	assert.True(t, a.done)
}

type tokenClient struct {
}

func (a tokenClient) GetAccessToken() (string, *User, time.Duration, error) {
	return "123", &User{}, 1000, nil
}

type authClient struct {
	token    string
	user     *User
	duration time.Duration
	err      error
}

func (a *authClient) GetAccessToken() (string, *User, time.Duration, error) {
	return a.token, a.user, a.duration, a.err
}
