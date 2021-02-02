package anypoint

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/config"
)

// Page describes the page query parameter
type Page struct {
	Offset   int
	PageSize int
}

// Client interface to gateway.
type Client interface {
	OnConfigChange(mulesoftConfig *config.MulesoftConfig)
	GetAccessToken() (string, time.Duration, error)

	ListAssets(page *Page) ([]Asset, error)
	GetAssetDetails(asset *Asset) error
	// GetAssetHomePage(orgID string, groupID string, assetID string, version string) error // TODO RETURN
	// GetAssetLinkedAPI(orgID string, environmentID string, assetID string) error          // TODO RETURN
	// API Details
	// API Details Icon
	// API Details Instances

}

// anypointClient is the client for interacting with Mulesoft Anypoint.
type anypointClient struct {
	baseURL        string
	organizationID string
	username       string
	password       string
	lifetime       time.Duration
	apiClient      coreapi.Client
	auth           Auth
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
	c.organizationID = mulesoftConfig.OrganizationID
	c.username = mulesoftConfig.Username
	c.password = mulesoftConfig.Password
	c.lifetime = mulesoftConfig.SessionLifetime
	c.apiClient = coreapi.NewClient(mulesoftConfig.TLS, mulesoftConfig.ProxyURL)

	if c.auth != nil {
		c.auth.Stop()
	}
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
func (c *anypointClient) GetAccessToken() (string, time.Duration, error) {
	body := map[string]string{
		"username": c.username,
		"password": c.password,
	}
	buffer, err := json.Marshal(body)
	if err != nil {
		return "", 0, agenterrors.Wrap(ErrMarshallingBody, err.Error())
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
		return "", 0, agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}
	if response.Code != http.StatusOK {
		return "", 0, ErrAuthentication
	}

	respMap := make(map[string]interface{})
	err = json.Unmarshal(response.Body, &respMap)
	if err != nil {
		return "", 0, agenterrors.Wrap(ErrAuthentication, err.Error())
	}

	// Would be better to look up the lifetime.
	return respMap["access_token"].(string), c.lifetime, nil
}

// ListAssets lists the managed assets in Mulesoft: https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/f1e97bc6-315a-4490-82a7-23abe036327a.anypoint-platform/exchange-experience-api/minor/2.0/console/method/%231431/
func (c *anypointClient) ListAssets(page *Page) ([]Asset, error) {
	assets := make([]Asset, 0, page.PageSize)
	err := c.invokeGet(c.baseURL+"/exchange/api/v2/assets", page, &assets)
	return assets, err
}

// GetAssetDetails creates the AssetDetail form the Asset.
func (c *anypointClient) GetAssetDetails(asset *Asset) error {
	return nil
}

// GetAssetIcon creates the AssetDetail form the Asset.
func (c *anypointClient) GetAssetIcon(asset *Asset) (string, error) {
	if asset.Icon == "" {
		return "", nil
	}

	var icon []byte
	err := c.invokeGet(asset.Icon, nil, &icon)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(icon)), nil
}

func (c *anypointClient) invokeGet(url string, page *Page, resp interface{}) error {
	headers := map[string]string{
		"Authorization": "Bearer " + c.auth.GetToken(),
	}

	query := map[string]string{
		"masterOrganizationId": c.organizationID,
	}

	if page != nil {
		query["offset"] = fmt.Sprint(page.Offset)
		query["limit"] = fmt.Sprint(page.PageSize)
	}

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		Headers:     headers,
		QueryParams: query,
	}

	return c.invoke(request, resp)
}

func (c *anypointClient) invoke(request coreapi.Request, resp interface{}) error {
	response, err := c.apiClient.Send(request)
	if err != nil {
		return agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}
	if response.Code != http.StatusOK {
		return agenterrors.Wrap(ErrCommunicatingWithGateway, fmt.Sprint(response.Code))
	}

	err = json.Unmarshal(response.Body, resp)
	if err != nil {
		return agenterrors.Wrap(ErrAuthentication, err.Error())
	}
	return nil
}
