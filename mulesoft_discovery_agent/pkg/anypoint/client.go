package anypoint

import (
	coreapi "github.com/Axway/agent-sdk/pkg/api"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/config"
)

// Client interface to gateway.
type Client interface {
	OnConfigChange(mulesoftConfig *config.MulesoftConfig)
}

// anypointClient is the client for interacting with Mulesoft Anypoint.
type anypointClient struct {
	url       string
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
	c.url = mulesoftConfig.AnypointExchangeURL
	// c.authToken = base64.StdEncoding.EncodeToString([]byte(mgrConfig.User + ":" + mgrConfig.Password))
	c.apiClient = coreapi.NewClient(mulesoftConfig.TLS, mulesoftConfig.ProxyURL)
	// c.apiVersion = mgrConfig.APIVersion
	// c.pollInterval = mgrConfig.PollInterval

	c.auth, _ = NewAuth()

	// TODO HANDLE ERR
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
