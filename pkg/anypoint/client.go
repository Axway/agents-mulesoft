package anypoint

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/Axway/agents-mulesoft/pkg/config"
)

const HealthCheckEndpoint = "mulesoft"

// Page describes the page query parameter
type Page struct {
	Offset   int
	PageSize int
}

// Client interface to gateway
type Client interface {
	CreateClientApplication(apiInstanceID string, app *AppRequestBody) (*Application, error)
	CreateContract(int64, *Contract) (*Contract, error)
	DeleteClientApplication(appID int64) error
	GetAccessToken() (string, *User, time.Duration, error)
	GetAPI(id string) (*API, error)
	GetClientApplication(appID string) (*Application, error)
	GetEnvironmentByName(name string) (*Environment, error)
	GetExchangeAsset(groupID, assetID, assetVersion string) (*ExchangeAsset, error)
	GetExchangeAssetIcon(icon string) (string, string, error)
	GetExchangeFileContent(link, packaging, mainFile string, useOriginalRaml bool) ([]byte, bool, error)
	GetPolicies(apiID int64) ([]Policy, error)
	GetSLATiers(int642 int64) (*Tiers, error)
	ListAssets(page *Page) ([]Asset, error)
	OnConfigChange(mulesoftConfig *config.MulesoftConfig)
	DeleteContract(apiID string, contractID string) error
	RevokeContract(apiID, contractID string) error
	ResetAppSecret(appID int64) (*Application, error)
}

type AnalyticsClient interface {
	GetAnalyticsWindow(string, string) ([]AnalyticsEvent, error)
	OnConfigChange(mulesoftConfig *config.MulesoftConfig)
	GetClientApplication(appId string) (*Application, error)
	GetAPI(id string) (*API, error)
}

type AuthClient interface {
	GetAccessToken() (string, *User, time.Duration, error)
}

type ListAssetClient interface {
	ListAssets(page *Page) ([]Asset, error)
}

// AnypointClient is the client for interacting with Mulesoft Anypoint.
type AnypointClient struct {
	baseURL      string
	username     string
	password     string
	clientID     string
	clientSecret string
	lifetime     time.Duration
	apiClient    coreapi.Client
	auth         Auth
	environment  *Environment
	orgName      string
}

type ClientOptions func(*AnypointClient)

// NewClient creates a new client for interacting with Mulesoft.
func NewClient(mulesoftConfig *config.MulesoftConfig, options ...ClientOptions) *AnypointClient {
	client := &AnypointClient{}
	// Create a new client before invoking additional options, which may want to override the client
	client.apiClient = coreapi.NewClient(mulesoftConfig.TLS, mulesoftConfig.ProxyURL)

	for _, o := range options {
		o(client)
	}
	client.OnConfigChange(mulesoftConfig)

	hc.RegisterHealthcheck("Mulesoft Anypoint Exchange", HealthCheckEndpoint, client.healthcheck)

	return client
}

func (c *AnypointClient) OnConfigChange(mulesoftConfig *config.MulesoftConfig) {
	if c.auth != nil {
		c.auth.Stop()
	}

	c.baseURL = mulesoftConfig.AnypointExchangeURL
	c.clientID = mulesoftConfig.ClientID
	c.clientSecret = mulesoftConfig.ClientSecret
	c.orgName = mulesoftConfig.OrgName
	c.lifetime = mulesoftConfig.SessionLifetime

	var err error
	c.auth, err = NewAuth(c)
	if err != nil {
		logrus.Fatalf("Failed to authenticate with Mulesoft: %s", err.Error())
	}

	c.environment, err = c.GetEnvironmentByName(mulesoftConfig.Environment)
	if c.environment == nil {
		logrus.Fatalf("Failed to connect to Mulesoft. Environment for '%s' not found", mulesoftConfig.Environment)
	}
	if err != nil {
		logrus.Fatalf("Failed to connect to Mulesoft environment %s: %s", mulesoftConfig.Environment, err.Error())
	}
}

func (c *AnypointClient) healthcheck(name string) (status *hc.Status) {
	status = &hc.Status{
		Result: hc.OK,
	}

	user, err := c.getUser()
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

// GetAccessToken retrieves a token
func (c *AnypointClient) GetAccessToken() (string, *User, time.Duration, error) {
	if c.clientID == "" || c.clientSecret == "" {
		return "", nil, 0, fmt.Errorf("authentication only available through clientID and clientSecret")
	}
	url := c.baseURL + "/accounts/api/v2/oauth2/token"
	body := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
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
		URL:     url,
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
	token, ok := respMap["access_token"].(string)
	if !ok {
		return "", nil, 0, ErrMarshallingBody
	}
	lifetime, ok := respMap["expires_in"].(float64)
	if !ok {
		return "", nil, 0, ErrMarshallingBody
	}

	c.lifetime = time.Second * time.Duration(lifetime)
	user, err := c.getCurrentUser(token)
	if err != nil {
		return "", nil, 0, agenterrors.Wrap(ErrAuthentication, err.Error())
	}

	// Would be better to look up the lifetime.
	return token, user, c.lifetime, nil
}

// getUser returns the current user.
func (c *AnypointClient) getUser() (*User, error) {
	return c.getCurrentUser(c.auth.GetToken())
}

// getCurrentUser returns the current user. Used internally during authentication
func (c *AnypointClient) getCurrentUser(token string) (*User, error) {
	headers := map[string]string{
		"Authorization": c.getAuthString(token),
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

	// this sets the User.Organization.ID as the Org ID of the Business Unit specified in Config
	for _, value := range user.User.MemberOfOrganizations {
		if value.ID == c.orgName {
			user.User.Organization.ID = value.ID
			user.User.Organization.Name = value.Name
		}

	}

	return &user.User, nil
}

// GetEnvironmentByName gets the Mulesoft environment with the specified name.
func (c *AnypointClient) GetEnvironmentByName(name string) (*Environment, error) {
	headers := map[string]string{
		"Authorization": c.getAuthString(c.auth.GetToken()),
	}

	query := map[string]string{
		"name": name,
	}

	url := fmt.Sprintf("%s/accounts/api/organizations/%s/environments", c.baseURL, c.auth.GetOrgID())
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
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
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis", c.baseURL, c.auth.GetOrgID(), c.environment.ID)
	query := map[string]string{
		"filters": "active",
	}
	err := c.invokeJSONGet(url, page, &assetResult, query)

	if err != nil {
		return nil, err
	}

	return assetResult.Assets, err
}

// GetAPI gets a single api by id
func (c *AnypointClient) GetAPI(id string) (*API, error) {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%s", c.baseURL, c.auth.GetOrgID(), c.environment.ID, id)
	res := &API{}
	err := c.invokeJSONGet(url, nil, res, nil)

	if err != nil {
		return nil, err
	}

	return res, err
}

// GetPolicies lists the API policies.
func (c *AnypointClient) GetPolicies(apiID int64) ([]Policy, error) {
	policies := Policies{}
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d/policies", c.baseURL, c.auth.GetOrgID(), c.environment.ID, apiID)
	err := c.invokeJSONGet(url, nil, &policies, nil)
	// Older versions of mulesoft may return []Policy JSON format instead.
	if err != nil && strings.HasPrefix(err.Error(), "json: cannot unmarshal") {
		err = c.invokeJSONGet(url, nil, &(policies.Policies), nil)
	}
	// Same issue, but with ConfigurationData and Configuration
	for i, pCfg := range policies.Policies {
		if pCfg.ConfigurationData != nil {
			policies.Policies[i].Configuration = pCfg.Configuration
		}
	}
	return policies.Policies, err
}

// GetExchangeAsset creates the AssetDetail form the Asset API.
func (c *AnypointClient) GetExchangeAsset(groupID, assetID, assetVersion string) (*ExchangeAsset, error) {
	var exchangeAsset ExchangeAsset
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s", c.baseURL, groupID, assetID, assetVersion)
	err := c.invokeJSONGet(url, nil, &exchangeAsset, nil)
	if err != nil {
		return nil, err
	}
	return &exchangeAsset, nil
}

// GetExchangeAssetIcon get the icon as a base64 encoded string from the Exchange Asset files.
func (c *AnypointClient) GetExchangeAssetIcon(icon string) (string, string, error) {
	if icon == "" {
		return "", "", nil
	}
	iconBuffer, headers, err := c.invokeGet(icon)
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
// and there is a MainFile set then the content of the MainFile is returned.
func (c *AnypointClient) GetExchangeFileContent(link, packaging, mainFile string, useOriginalRaml bool) ([]byte, bool, error) {
	wasConverted := false
	fileContent, _, err := c.invokeGet(link)
	if packaging != "zip" {
		return fileContent, wasConverted, err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(fileContent), int64(len(fileContent)))
	if err != nil {
		return nil, wasConverted, err
	}

	for _, f := range zipReader.File {
		// In case of RAML spec, this gets automatically converted and is renamed to api.json
		if f.Name != mainFile && f.Name != "api.json" {
			continue
		}
		content, err := f.Open()
		if err != nil {
			return nil, wasConverted, err
		}

		fileContent, err = io.ReadAll(content)
		content.Close()
		if err != nil {
			return nil, wasConverted, err
		}
		if !useOriginalRaml && f.Name == "api.json" {
			return fileContent, true, err
		}
		break
	}
	return fileContent, wasConverted, err
}

// GetAnalyticsWindow lists the managed assets in Mulesoft: https://docs.qax.mulesoft.com/api-manager/2.x/analytics-event-api
func (c *AnypointClient) GetAnalyticsWindow(startDate, endDate string) ([]AnalyticsEvent, error) {
	query := map[string]string{
		"format":    "json",
		"startDate": startDate,
		"endDate":   endDate,
		"fields":    "Application Name.Application.Browser.City.Client IP.Continent.Country.Hardware Platform.Message ID.OS Family.OS Major Version.OS Minor Version.OS Version.Postal Code.Request Outcome.Request Size.Resource Path.Response Size.Response Time.Status Code.Timezone.User Agent Name.User Agent Version.Verb.Violated Policy Name",
	}
	headers := map[string]string{
		"Authorization": c.getAuthString(c.auth.GetToken()),
	}

	url := fmt.Sprintf("%s/analytics/1.0/%s/environments/%s/events", c.baseURL, c.auth.GetOrgID(), c.environment.ID)
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

func (c *AnypointClient) GetSLATiers(apiID int64) (*Tiers, error) {
	var slatiers Tiers
	headers := map[string]string{
		"Authorization": c.getAuthString(c.auth.GetToken()),
	}
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d/tiers",
		c.baseURL, c.auth.GetOrgID(), c.environment.ID, apiID)

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		QueryParams: nil,
		Headers:     headers,
	}
	err := c.invokeJSON(request, &slatiers)
	return &slatiers, err
}

func (c *AnypointClient) CreateClientApplication(apiInstanceID string, app *AppRequestBody) (*Application, error) {
	var application Application
	query := map[string]string{
		"apiInstanceId": apiInstanceID,
	}

	url := fmt.Sprintf("%s/exchange/api/v1/organizations/%s/applications", c.baseURL, c.auth.GetOrgID())

	buffer, err := json.Marshal(app)
	if err != nil {
		return nil, agenterrors.Wrap(ErrMarshallingBody, err.Error())
	}

	err = c.invokeJSONPost(url, query, buffer, &application)
	if err != nil {
		return nil, err
	}
	return &application, nil
}

func (c *AnypointClient) ResetAppSecret(appID int64) (*Application, error) {
	url := fmt.Sprintf("%s/exchange/api/v2/organizations/%s/applications/%v/secret/reset", c.baseURL, c.auth.GetOrgID(), appID)
	application := &Application{}
	err := c.invokeJSONPost(url, nil, []byte{}, application)
	return application, err
}

func (c *AnypointClient) DeleteClientApplication(appID int64) error {
	url := fmt.Sprintf("%s/exchange/api/v2/organizations/%s/applications/%v", c.baseURL, c.auth.GetOrgID(), appID)

	headers := map[string]string{
		"Authorization": c.getAuthString(c.auth.GetToken()),
	}

	request := coreapi.Request{
		Method:      coreapi.DELETE,
		URL:         url,
		QueryParams: nil,
		Headers:     headers,
		Body:        nil,
	}

	return c.invokeDelete(request)
}

func (c *AnypointClient) GetClientApplication(appID string) (*Application, error) {
	var application Application
	url := fmt.Sprintf("%s/exchange/api/v2/organizations/%s/applications/%s", c.baseURL, c.auth.GetOrgID(), appID)

	headers := map[string]string{
		"Authorization": c.getAuthString(c.auth.GetToken()),
	}

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		QueryParams: nil,
		Headers:     headers,
	}
	err := c.invokeJSON(request, &application)
	return &application, err
}

func (c *AnypointClient) DeleteContract(apiID string, contractID string) error {
	url := fmt.Sprintf(
		"%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%s/contracts/%s",
		c.baseURL, c.auth.GetOrgID(), c.environment.ID, apiID, contractID,
	)

	headers := map[string]string{
		"Authorization": c.getAuthString(c.auth.GetToken()),
	}

	request := coreapi.Request{
		Method:      coreapi.DELETE,
		URL:         url,
		QueryParams: nil,
		Headers:     headers,
		Body:        nil,
	}

	return c.invokeDelete(request)
}

func (c *AnypointClient) RevokeContract(apiID, contractID string) error {
	res := map[string]interface{}{}

	url := fmt.Sprintf(
		"%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%s/contracts/%s/revoke",
		c.baseURL, c.auth.GetOrgID(), c.environment.ID, apiID, contractID,
	)

	err := c.invokeJSONPost(url, nil, nil, &res)
	if err != nil {
		return err
	}

	return nil
}

func (c *AnypointClient) GetContract(apiID string, contractID string) (*Contract, error) {
	var cnt Contract

	url := fmt.Sprintf(
		"%s/exchange/api/v1/organizations/%s/environments/%s/apis/%s/contracts/%s",
		c.baseURL, c.auth.GetOrgID(), c.environment.ID, apiID, contractID,
	)

	err := c.invokeJSONGet(url, nil, &cnt, nil)
	if err != nil {
		return nil, err
	}

	return &cnt, nil
}

func (c *AnypointClient) CreateContract(appID int64, contract *Contract) (*Contract, error) {
	var cnt Contract

	url := fmt.Sprintf("%s/exchange/api/v1/organizations/%s/applications/%d/contracts", c.baseURL, c.auth.GetOrgID(), appID)

	buffer, err := json.Marshal(contract)
	if err != nil {
		return nil, agenterrors.Wrap(ErrMarshallingBody, err.Error())
	}

	err = c.invokeJSONPost(url, nil, buffer, &cnt)
	if err != nil {
		return nil, err
	}

	return &cnt, nil
}

func (c *AnypointClient) invokeJSONGet(url string, page *Page, resp interface{}, query map[string]string) error {
	headers := map[string]string{
		"Authorization": c.getAuthString(c.auth.GetToken()),
	}

	if query == nil {
		query = map[string]string{}
	}

	query["masterOrganizationId"] = c.auth.GetOrgID()
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

func (c *AnypointClient) invokeJSONPost(url string, query map[string]string, body []byte, resp interface{}) error {
	headers := map[string]string{
		"Authorization": c.getAuthString(c.auth.GetToken()),
		"Content-Type":  "application/json",
		"Accept":        "application/json",
	}

	request := coreapi.Request{
		Method:      coreapi.POST,
		URL:         url,
		QueryParams: query,
		Headers:     headers,
		Body:        body,
	}

	return c.invokeJSON(request, resp)
}

func (c *AnypointClient) invokeDelete(request coreapi.Request) error {
	response, err := c.apiClient.Send(request)
	if err != nil {
		return agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}

	if response.Code != http.StatusNoContent {
		return agenterrors.Wrap(ErrCommunicatingWithGateway, fmt.Sprint(response.Code))
	}
	return nil
}

func (c *AnypointClient) invokeJSON(request coreapi.Request, resp interface{}) error {
	body, _, err := c.invoke(request)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, resp)
	if err != nil {
		return err
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
	if !(response.Code == http.StatusOK || response.Code == http.StatusCreated) {
		res := NewErrorResponse(string(response.Body), response.Code)
		return nil, nil, agenterrors.Wrap(ErrCommunicatingWithGateway, res.String())
	}

	return response.Body, response.Headers, nil
}

// SetClient replaces the default apiClient with anything that implements the Client interface. Can be used for writing tests.
func SetClient(c coreapi.Client) ClientOptions {
	return func(ac *AnypointClient) {
		ac.apiClient = c
	}
}

func (c *AnypointClient) getAuthString(token string) string {
	return "Bearer " + token
}
