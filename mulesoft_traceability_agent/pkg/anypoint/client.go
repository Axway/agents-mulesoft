package anypoint

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/config"

	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

// Page describes the page query parameter
type Page struct {
	Offset   int
	PageSize int
}

const (
	// CacheKeyTimeStamp -
	CacheKeyTimeStamp = "LAST_RUN"
)

// Client interface to gateway.
type Client interface {
	OnConfigChange(mulesoftConfig *config.MulesoftConfig)
	GetAccessToken() (string, *User, time.Duration, error)
	GetAnalyticsWindow() ([]AnalyticsEvent, error)
}

// anypointClient is the client for interacting with Mulesoft Anypoint.
type anypointClient struct {
	baseURL     string
	username    string
	password    string
	lifetime    time.Duration
	apiClient   coreapi.Client
	auth        Auth
	environment *Environment
	cache       cache.Cache
}

// NewClient creates a new client for interacting with Mulesoft.
func NewClient(mulesoftConfig *config.MulesoftConfig) Client {

	client := &anypointClient{}
	client.OnConfigChange(mulesoftConfig)

	// Register the healthcheck
	hc.RegisterHealthcheck("Mulesoft Anypoint Exchange", "mulesoft", client.healthcheck)

	return client
}

// GetAnalyticsWindow lists the managed assets in Mulesoft: https://docs.qax.mulesoft.com/api-manager/2.x/analytics-event-api
func (c *anypointClient) GetAnalyticsWindow() ([]AnalyticsEvent, error) {
	startDate, endDate := c.getLastRun()
	query := map[string]string{
		"format":    "json",
		"startDate": startDate,
		"endDate":   endDate,
		"fields":    "Application Name.Browser.City.Client IP.Continent.Country.Hardware Platform.Message ID.OS Family.OS Major Version.OS Minor Version.OS Version.Postal Code.Request Outcome.Request Size.Resource Path.Response Size.Response Time.Status Code.Timezone.User Agent Name.User Agent Version.Verb.Violated Policy Name",
	}
	headers := map[string]string{
		"Authorization": "Bearer " + c.auth.GetToken(),
	}

	url := c.baseURL + "/analytics/1.0/" + c.auth.GetOrgID() + "/environments/" + c.environment.ID + "/events"
	events := make([]AnalyticsEvent, 0)
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		Headers:     headers,
		QueryParams: query,
	}
	err := c.invokeJSON(request, &events)
	return events, err

}

// GetAccessToken gets an access token
func (c *anypointClient) GetAccessToken() (string, *User, time.Duration, error) {
	body := map[string]string{
		"username": c.username,
		"password": c.password,
	}
	buffer, err := json.Marshal(body)
	if err != nil {
		return "", nil, 0, agenterrors.Wrap(ErrMarshallingBody, err.Error())
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
		return "", nil, 0, agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}
	if response.Code != http.StatusOK {
		return "", nil, 0, ErrAuthentication
	}

	respMap := make(map[string]interface{})
	err = json.Unmarshal(response.Body, &respMap)
	if err != nil {
		return "", nil, 0, agenterrors.Wrap(ErrAuthentication, err.Error())
	}
	token := respMap["access_token"].(string)

	user, err := c.getCurrentUser(token)
	if err != nil {
		return "", nil, 0, agenterrors.Wrap(ErrAuthentication, err.Error())
	}

	// Would be better to look up the lifetime.
	return token, user, c.lifetime, nil
}

// GetCurrentUser returns the current user.
func (c *anypointClient) GetCurrentUser() (*User, error) {
	return c.getCurrentUser(c.auth.GetToken())
}

// getCurrentUser returns the current user. Used internally during authentication
func (c *anypointClient) getCurrentUser(token string) (*User, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     c.baseURL + "/accounts/api/me",
		Headers: headers,
	}

	var user CurrentUser
	err := c.invokeJSON(request, &user)
	if err != nil {
		return nil, err
	}

	return &user.User, nil
}

// GetEnvironmentByName gets the Mulesoft environment with the specified name.
func (c *anypointClient) GetEnvironmentByName(name string) (*Environment, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + c.auth.GetToken(),
	}

	query := map[string]string{
		"name": name,
	}

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.baseURL + "/accounts/api/organizations/" + c.auth.GetOrgID() + "/environments",
		Headers:     headers,
		QueryParams: query,
	}

	var envSearch EnvironmentSearch
	err := c.invokeJSON(request, &envSearch)
	if err != nil {
		return nil, err
	}

	if len(envSearch.Data) == 0 {
		return nil, nil
	}
	return &envSearch.Data[0], nil
}

// loadOrCreateCache  build the cache or load from prior
func loadOrCreateCache(path string) cache.Cache {
	return cache.Load(path)
}

// OnConfigChange updates the client when the configuration changes.
func (c *anypointClient) OnConfigChange(mulesoftConfig *config.MulesoftConfig) {
	if c.auth != nil {
		c.auth.Stop()
	}

	c.baseURL = mulesoftConfig.AnypointExchangeURL
	c.username = mulesoftConfig.Username
	c.password = mulesoftConfig.Password
	c.lifetime = mulesoftConfig.SessionLifetime
	c.apiClient = coreapi.NewClient(mulesoftConfig.TLS, mulesoftConfig.ProxyURL)
	// TODO add to config
	c.cache = loadOrCreateCache("/tmp/anypoint.cache")

	var err error
	c.auth, err = NewAuth(c)
	if err != nil {
		log.Fatalf("Failed to authenticate: %s", err.Error())
	}

	c.environment, err = c.GetEnvironmentByName(mulesoftConfig.Environment)
	if err != nil {
		log.Fatalf("Failed to connect to Mulesoft environment %s: %s", mulesoftConfig.Environment, err.Error())
	}
}

// healthcheck performs Mulesoft healthcheck.
func (c *anypointClient) healthcheck(name string) (status *hc.Status) {
	// Create the default status
	status = &hc.Status{
		Result: hc.OK,
	}

	user, err := c.GetCurrentUser()
	if err != nil {
		status = &hc.Status{
			Result:  hc.FAIL,
			Details: fmt.Sprintf("%s Failed. Unable to connect to Mulesoft, check Mulesoft configuration. %s", name, err.Error()),
		}
	}
	if user == nil {
		status = &hc.Status{
			Result:  hc.FAIL,
			Details: fmt.Sprintf("%s Failed. Unable to connect to Mulesoft, check Mulesoft configuration.", name),
		}
	}

	return status
}

func (c *anypointClient) getLastRun() (string, string) {
	tStamp, _ := c.cache.Get(CacheKeyTimeStamp)
	now := time.Now()
	tNow := now.Format(time.RFC3339)
	if tStamp == nil {
		tStamp = tNow
	}
	c.cache.Set(CacheKeyTimeStamp, tNow)
	c.cache.Save("/tmp/anypoint.cache")
	return tStamp.(string), tNow

}

func (c *anypointClient) invokeJSON(request coreapi.Request, resp interface{}) error {
	body, err := c.invoke(request)
	if err != nil {
		return agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}

	err = json.Unmarshal(body, resp)
	if err != nil {
		return agenterrors.Wrap(ErrMarshallingBody, err.Error())
	}
	return nil
}

func (c *anypointClient) invoke(request coreapi.Request) ([]byte, error) {
	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}
	if response.Code != http.StatusOK {
		return nil, agenterrors.Wrap(ErrCommunicatingWithGateway, fmt.Sprint(response.Code))
	}

	return response.Body, nil
}
