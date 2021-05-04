package anypoint

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

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
			c := &authClient{}
			user := &User{
				Organization: Organization{
					ID: "1",
				},
			}

			c.On("GetAccessToken").Return(tc.token, user, time.Duration(10), tc.err)

			auth, err := NewAuth(c)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				assert.NotNil(t, auth)
				org := auth.GetOrgID()
				token := auth.GetToken()
				assert.Equal(t, "1", org)
				assert.Equal(t, tc.token, token)
				auth.Stop()

			} else {
				assert.Nil(t, auth)
			}

		})
	}

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

type authClientRefreshErr struct {
	stop chan bool
}

func (a authClientRefreshErr) GetAccessToken() (string, *User, time.Duration, error) {
	a.stop <- true
	return "", &User{}, 0, fmt.Errorf("auth error")
}

type authClient struct {
	mock.Mock
}

func (a *authClient) GetAccessToken() (string, *User, time.Duration, error) {
	args := a.Called()
	token := args.String(0)
	user := args.Get(1)
	duration := args.Get(2)
	return token, user.(*User), duration.(time.Duration), args.Error(3)
}
