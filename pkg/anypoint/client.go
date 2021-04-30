package anypoint

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/Axway/agent-sdk/pkg/cache"

	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/Axway/agents-mulesoft/pkg/config"
)

const (
	CacheKeyTimeStamp = "LAST_RUN"
)

// Page describes the page query parameter
type Page struct {
	Offset   int
	PageSize int
}

// Client interface to gateway.
type Client interface {
	OnConfigChange(mulesoftConfig *config.MulesoftConfig)
	GetAccessToken() (string, *User, time.Duration, error)
	GetEnvironmentByName(name string) (*Environment, error)
	ListAssets(page *Page) ([]Asset, error)
	GetPolicies(api *API) (Policies, error)
	GetExchangeAsset(api *API) (*ExchangeAsset, error)
	GetExchangeAssetIcon(asset *ExchangeAsset) (icon string, contentType string, err error)
	GetExchangeFileContent(file *ExchangeFile) (fileContent []byte, err error)
	GetAnalyticsWindow() ([]AnalyticsEvent, error)
}

type AnalyticsClient interface {
	GetAnalyticsWindow() ([]AnalyticsEvent, error)
	OnConfigChange(mulesoftConfig *config.MulesoftConfig)
}

type AuthClient interface {
	GetAccessToken() (string, *User, time.Duration, error)
}

// AnypointClient is the client for interacting with Mulesoft Anypoint.
type AnypointClient struct {
	baseURL     string
	username    string
	password    string
	lifetime    time.Duration
	apiClient   coreapi.Client
	auth        Auth
	environment *Environment
	cache       cache.Cache
	cachePath   string
}

type ClientOptions func(*AnypointClient)

// NewClient creates a new client for interacting with Mulesoft.
func NewClient(mulesoftConfig *config.MulesoftConfig, options ...ClientOptions) *AnypointClient {
	client := &AnypointClient{}
	client.cachePath = formatCachePath(mulesoftConfig.CachePath)
	// Create a new client before invoking additional options, which may want to override the client
	client.apiClient = coreapi.NewClient(mulesoftConfig.TLS, mulesoftConfig.ProxyURL)

	for _, o := range options {
		o(client)
	}
	client.OnConfigChange(mulesoftConfig)

	hc.RegisterHealthcheck("Mulesoft Anypoint Exchange", "mulesoft", client.healthcheck)
	// TODO: handle error

	return client
}

func (c *AnypointClient) OnConfigChange(mulesoftConfig *config.MulesoftConfig) {
	if c.auth != nil {
		c.auth.Stop()
	}

	c.baseURL = mulesoftConfig.AnypointExchangeURL
	c.username = mulesoftConfig.Username
	c.password = mulesoftConfig.Password
	c.lifetime = mulesoftConfig.SessionLifetime
	c.cachePath = formatCachePath(mulesoftConfig.CachePath)
	c.cache = cache.Load(c.cachePath)

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
func (c *AnypointClient) healthcheck(name string) (status *hc.Status) {
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

// GetAccessToken gets an access token
func (c *AnypointClient) GetAccessToken() (string, *User, time.Duration, error) {
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
func (c *AnypointClient) GetCurrentUser() (*User, error) {
	return c.getCurrentUser(c.auth.GetToken())
}

// getCurrentUser returns the current user. Used internally during authentication
func (c *AnypointClient) getCurrentUser(token string) (*User, error) {
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
func (c *AnypointClient) GetEnvironmentByName(name string) (*Environment, error) {
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

// ListAssets lists the API Assets.
func (c *AnypointClient) ListAssets(page *Page) ([]Asset, error) {
	var assetResult AssetSearch
	url := c.baseURL + "/apimanager/api/v1/organizations/" + c.auth.GetOrgID() + "/environments/" + c.environment.ID + "/apis"
	err := c.invokeJSONGet(url, page, &assetResult)

	if err != nil {
		return nil, err
	}

	return assetResult.Assets, err
}

// GetPolicies lists the API policies.
func (c *AnypointClient) GetPolicies(api *API) (Policies, error) {
	var policies Policies
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d/policies", c.baseURL, c.auth.GetOrgID(), c.environment.ID, api.ID)
	err := c.invokeJSONGet(url, nil, &policies)

	if err != nil {
		return Policies{}, err
	}

	return policies, err
}

// GetExchangeAsset creates the AssetDetail form the Asset API.
func (c *AnypointClient) GetExchangeAsset(api *API) (*ExchangeAsset, error) {
	var exchangeAsset ExchangeAsset
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s", c.baseURL, api.GroupID, api.AssetID, api.AssetVersion)
	err := c.invokeJSONGet(url, nil, &exchangeAsset)
	if err != nil {
		return nil, err
	}
	return &exchangeAsset, nil
}

// GetExchangeAssetIcon get the icon as a base64 encoded string from the Exchange Asset files.
func (c *AnypointClient) GetExchangeAssetIcon(asset *ExchangeAsset) (string, string, error) {
	if asset.Icon == "" {
		return "", "", nil
	}
	iconBuffer, headers, err := c.invokeGet(asset.Icon)
	if err != nil {
		return "", "", err
	}

	contentType := ""
	if val, exists := headers["Content-Type"]; exists {
		contentType = val[0]
	}

	return base64.StdEncoding.EncodeToString(iconBuffer), contentType, nil
}

// GetExchangeFileContent download the file from the ExternalLink reference. If the file is a zip file
// and thre is a MainFile set then the content of the MainFile is returned.
func (c *AnypointClient) GetExchangeFileContent(file *ExchangeFile) (fileContent []byte, err error) {
	fileContent, _, err = c.invokeGet(file.ExternalLink)

	if file.Packaging == "zip" && file.MainFile != "" {
		zipReader, err := zip.NewReader(bytes.NewReader(fileContent), int64(len(fileContent)))
		if err != nil {
			return nil, err
		}

		for _, f := range zipReader.File {
			if f.Name == file.MainFile {
				content, err := f.Open()
				if err != nil {
					return nil, err
				}

				fileContent, err = ioutil.ReadAll(content)
				content.Close()
				if err != nil {
					return nil, err
				}
				break
			}
		}
	}
	return fileContent, err
}

// GetAnalyticsWindow lists the managed assets in Mulesoft: https://docs.qax.mulesoft.com/api-manager/2.x/analytics-event-api
func (c *AnypointClient) GetAnalyticsWindow() ([]AnalyticsEvent, error) {
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

func (c *AnypointClient) getLastRun() (string, string) {
	tStamp, _ := c.cache.Get(CacheKeyTimeStamp)
	now := time.Now()
	tNow := now.Format(time.RFC3339)
	if tStamp == nil {
		tStamp = tNow
	}
	c.cache.Set(CacheKeyTimeStamp, tNow)
	c.cache.Save(c.cachePath)
	return tStamp.(string), tNow

}

func (c *AnypointClient) invokeJSONGet(url string, page *Page, resp interface{}) error {
	headers := map[string]string{
		"Authorization": "Bearer " + c.auth.GetToken(),
	}

	query := map[string]string{
		"masterOrganizationId": c.auth.GetOrgID(),
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

	return c.invokeJSON(request, resp)
}

func (c *AnypointClient) invokeJSON(request coreapi.Request, resp interface{}) error {
	body, _, err := c.invoke(request)
	if err != nil {
		return agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}

	//TODO this is to prevent Mule 3 APIs with no policies crashing the discovery agent -> need to verify if still required
	if string(body) == "[]" {
		return nil
	}

	err = json.Unmarshal(body, resp)
	if err != nil {
		return agenterrors.Wrap(ErrMarshallingBody, err.Error())
	}
	return nil
}

func (c *AnypointClient) invokeGet(url string) ([]byte, map[string][]string, error) {
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		Headers:     nil,
		QueryParams: nil,
	}

	return c.invoke(request)
}

func (c *AnypointClient) invoke(request coreapi.Request) ([]byte, map[string][]string, error) {
	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, nil, agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}
	if response.Code != http.StatusOK {
		return nil, nil, agenterrors.Wrap(ErrCommunicatingWithGateway, fmt.Sprint(response.Code))
	}

	return response.Body, response.Headers, nil
}

func formatCachePath(path string) string {
	return fmt.Sprintf("%s/anypoint.cache", path)
}

// SetClient replaces the default apiClient with anything that implements the Client interface. Can be used for writing tests.
func SetClient(c coreapi.Client) ClientOptions {
	return func(ac *AnypointClient) {
		ac.apiClient = c
	}
}
