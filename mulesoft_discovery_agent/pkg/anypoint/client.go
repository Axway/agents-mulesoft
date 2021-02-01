package anypoint

import (
	"encoding/json"
	"net/http"

	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/config"
)

// Client interface to gateway.
type Client interface {
	OnConfigChange(mulesoftConfig *config.MulesoftConfig)
	GetAccessToken() (string, error)
}

// anypointClient is the client for interacting with Mulesoft Anypoint.
type anypointClient struct {
	baseURL   string
	username  string
	password  string
	apiClient coreapi.Client
	auth      Auth
}

// NewClient creates a new client for interacting with Mulesoft.
func NewClient(mulesoftConfig *config.MulesoftConfig) Client {
	client := &anypointClient{}
	client.OnConfigChange(mulesoftConfig)

	// TODO

	// Register the healthcheck
	hc.RegisterHealthcheck("Mulesoft Anypoint Exchange", "mulesoft", client.healthcheck)

	return client
}

// OnConfigChange updates the client when the configuration changes.
func (c *anypointClient) OnConfigChange(mulesoftConfig *config.MulesoftConfig) {
	c.baseURL = mulesoftConfig.AnypointExchangeURL
	c.username = mulesoftConfig.Username
	c.password = mulesoftConfig.Password
	c.apiClient = coreapi.NewClient(mulesoftConfig.TLS, mulesoftConfig.ProxyURL)

	c.auth, _ = NewAuth(c)

	// TODO HANDLE ERR..WHAT'S THE EXPECATATION HERE?
}

// healthcheck performs healthcheck
func (c *anypointClient) healthcheck(name string) (status *hc.Status) {
	// Create the default status
	status = &hc.Status{
		Result: hc.OK,
	}

	// TODO: Implement gateway healthchecks

	return status
}

// GetAccessToken gets an access token
func (c *anypointClient) GetAccessToken() (string, error) {
	body := map[string]string{
		"username": c.username,
		"password": c.password,
	}
	buffer, err := json.Marshal(body)
	if err != nil {
		return "", agenterrors.Wrap(ErrMarshallingBody, err.Error())
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	request := coreapi.Request{
		Method:  coreapi.POST,
		URL:     c.baseURL + "/accounts/login",
		Headers: headers,
		Body:    buffer,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return "", agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}
	if response.Code != http.StatusOK {
		return "", ErrAuthentication
	}

	respMap := make(map[string]interface{})
	err = json.Unmarshal(response.Body, &respMap)
	if err != nil {
		return "", agenterrors.Wrap(ErrAuthentication, err.Error())
	}

	return respMap["access_token"].(string), nil

}
