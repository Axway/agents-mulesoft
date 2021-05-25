package anypoint

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

// Client interface for Mulesoft
type Client interface {
	Authenticate() error
	CreateClientApplication(string, *AppRequestBody) (*Application, error)
	CreateContract(int64, *Contract) (*Contract, error)
	DeleteClientApplication(appId int64) error
	GetAccessToken() (string, *User, time.Duration, error)
	GetEnvironmentByName(name string) (*Environment, error)
	GetExchangeAsset(groupID, assetID, assetVersion string) (*ExchangeAsset, error)
	GetExchangeAssetIcon(icon string) (string, string, error)
	GetExchangeFileContent(link, packaging, mainFile string) ([]byte, error)
	GetPolicies(apiID int64) (Policies, error)
	GetSLATiers(int642 int64) (*Tiers, error)
	ListAssets(page *Page) ([]Asset, error)
	OnConfigChange(mulesoftConfig *config.MulesoftConfig)
}

type AnalyticsClient interface {
	GetAnalyticsWindow(string, string) ([]AnalyticsEvent, error)
	GetClientApplication(appId string) (*Application, error)
	OnConfigChange(mulesoftConfig *config.MulesoftConfig)
}

type TokenClient interface {
	GetAccessToken() (string, *User, time.Duration, error)
}

type ListAssetClient interface {
	ListAssets(page *Page) ([]Asset, error)
}

// AnypointClient is the client for interacting with Mulesoft Anypoint.
type AnypointClient struct {
	apiClient       coreapi.Client
	auth            Auth
	baseURL         string
	clientID        string
	clientSecret    string
	environment     *Environment
	environmentName string
	lifetime        time.Duration
	orgName         string
	password        string
	username        string
}

// ClientOptions callback for overriding the default config of the client
type ClientOptions func(*AnypointClient)

// NewClient creates a new client for interacting with Mulesoft.
func NewClient(mulesoftConfig *config.MulesoftConfig, registerHealth hc.RegisterHealth, options ...ClientOptions) (*AnypointClient, error) {
	client := &AnypointClient{
		apiClient:       nil,
		auth:            nil,
		baseURL:         mulesoftConfig.AnypointExchangeURL,
		clientID:        mulesoftConfig.ClientID,
		clientSecret:    mulesoftConfig.ClientSecret,
		environmentName: mulesoftConfig.Environment,
		environment:     nil,
		lifetime:        mulesoftConfig.SessionLifetime,
		orgName:         mulesoftConfig.OrgName,
		password:        mulesoftConfig.Password,
		username:        mulesoftConfig.Username,
	}

	// Create a new client before invoking additional options, which may want to override the client
	client.apiClient = coreapi.NewClient(mulesoftConfig.TLS, mulesoftConfig.ProxyURL)

	for _, o := range options {
		o(client)
	}

	_, err := registerHealth("Mulesoft Anypoint Exchange", HealthCheckEndpoint, client.healthcheck)

	return client, err
}

// Authenticate authenticate with Mulesoft
func (c *AnypointClient) Authenticate() error {
	c.auth = NewAuth(c)
	err := c.auth.Start()
	if err != nil {
		return err
	}

	c.environment, err = c.GetEnvironmentByName(c.environmentName)
	if c.environment == nil {
		return fmt.Errorf("environment for '%s' not found", c.environmentName)
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *AnypointClient) OnConfigChange(mulesoftConfig *config.MulesoftConfig) {
	if c.auth != nil {
		c.auth.Stop()
	}

	c.baseURL = mulesoftConfig.AnypointExchangeURL
	c.clientID = mulesoftConfig.ClientID
	c.clientSecret = mulesoftConfig.ClientSecret
	c.lifetime = mulesoftConfig.SessionLifetime
	c.orgName = mulesoftConfig.OrgName
	c.password = mulesoftConfig.Password
	c.username = mulesoftConfig.Username

	err := c.Authenticate()
	if err != nil {
		logrus.Fatalf("failed to connect to mulesoft: %s", err)
	}
}

// healthcheck performs Mulesoft healthcheck.
func (c *AnypointClient) healthcheck(name string) (status *hc.Status) {
	// Create the default status
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

// getTokenByClientID gets a token by the environment client id and secret.
// This is not being used by anything at the moment since some api's do not accept this token.
func (c *AnypointClient) getTokenByClientID() (string, error) {
	creds := &ClientAuthCredentials{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		GrantType:    "client_credentials",
	}

	buffer, err := json.Marshal(creds)
	if err != nil {
		return "", agenterrors.Wrap(ErrMarshallingBody, err.Error())
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	request := coreapi.Request{
		Method:  coreapi.POST,
		URL:     c.baseURL + "/accounts/oauth2/token",
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
		return "", agenterrors.Wrap(ErrParsingResponse, err.Error())
	}
	token := respMap["access_token"].(string)

	return token, nil
}

// getTokenByUsername Gets a bearer token by the username and password
func (c *AnypointClient) getTokenByUsername() (string, error) {
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
		return "", agenterrors.Wrap(ErrParsingResponse, err.Error())
	}
	token := respMap["access_token"].(string)

	return token, nil
}

// GetAccessToken gets an access token
func (c *AnypointClient) GetAccessToken() (string, *User, time.Duration, error) {
	var token string
	var err error

	if c.username != "" && c.password != "" {
		token, err = c.getTokenByUsername()
	} else {
		return "", nil, 0, fmt.Errorf("unable to authenticate due to misconfigured auth credentials")
	}

	if err != nil {
		return "", nil, 0, err
	}

	user, err := c.getCurrentUser(token)
	if err != nil {
		return "", nil, 0, agenterrors.Wrap(ErrAuthentication, err.Error())
	}

	return token, user, c.lifetime, nil
}

// getUser returns the current user.
func (c *AnypointClient) getUser() (*User, error) {
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

	//this sets the User.Organization.ID as the Org ID of the Business Unit specified in Config
	for _, value := range user.User.MemberOfOrganizations {
		if value.Name == c.orgName {
			user.User.Organization.ID = value.ID
			user.User.Organization.Name = value.Name
		}

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
func (c *AnypointClient) GetPolicies(apiID int64) (Policies, error) {
	var policies Policies
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d/policies", c.baseURL, c.auth.GetOrgID(), c.environment.ID, apiID)
	err := c.invokeJSONGet(url, nil, &policies)

	if err != nil {
		return Policies{}, err
	}

	return policies, err
}

// GetExchangeAsset creates the AssetDetail form the Asset API.
func (c *AnypointClient) GetExchangeAsset(groupID, assetID, assetVersion string) (*ExchangeAsset, error) {
	var exchangeAsset ExchangeAsset
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s", c.baseURL, groupID, assetID, assetVersion)
	err := c.invokeJSONGet(url, nil, &exchangeAsset)
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
// and thre is a MainFile set then the content of the MainFile is returned.
func (c *AnypointClient) GetExchangeFileContent(link, packaging, mainFile string) (fileContent []byte, err error) {
	fileContent, _, err = c.invokeGet(link)

	if packaging == "zip" && mainFile != "" {
		zipReader, err := zip.NewReader(bytes.NewReader(fileContent), int64(len(fileContent)))
		if err != nil {
			return nil, err
		}

		for _, f := range zipReader.File {
			if f.Name == mainFile {
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
func (c *AnypointClient) GetAnalyticsWindow(startDate, endDate string) ([]AnalyticsEvent, error) {
	query := map[string]string{
		"format":    "json",
		"startDate": startDate,
		"endDate":   endDate,
		"fields":    "Application Name.Application.Browser.City.Client IP.Continent.Country.Hardware Platform.Message ID.OS Family.OS Major Version.OS Minor Version.OS Version.Postal Code.Request Outcome.Request Size.Resource Path.Response Size.Response Time.Status Code.Timezone.User Agent Name.User Agent Version.Verb.Violated Policy Name",
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

func (c *AnypointClient) GetSLATiers(apiId int64) (*Tiers, error) {
	var slatiers Tiers
	headers := map[string]string{
		"Authorization": "Bearer " + c.auth.GetToken(),
	}
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d/tiers",
		c.baseURL, c.auth.GetOrgID(), c.environment.ID, apiId)

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
		"apiInstanceID": apiInstanceID,
	}

	// TODO Mulesoft trial uses v1 for create app and v2 for delete app. Check this on enterprise account
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

func (c *AnypointClient) DeleteClientApplication(appId int64) error {
	url := fmt.Sprintf("%s/exchange/api/v2/organizations/%s/applications/%v", c.baseURL, c.auth.GetOrgID(), appId)

	headers := map[string]string{
		"Authorization": "Bearer " + c.auth.GetToken(),
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

func (c *AnypointClient) GetClientApplication(appId string) (*Application, error) {

	var application Application
	url := fmt.Sprintf("%s/exchange/api/v2/organizations/%s/applications/%s", c.baseURL, c.auth.GetOrgID(), appId)

	headers := map[string]string{
		"Authorization": "Bearer " + c.auth.GetToken(),
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

func (c *AnypointClient) invokeJSONPost(url string, query map[string]string, body []byte, resp interface{}) error {
	headers := map[string]string{
		"Authorization": "Bearer " + c.auth.GetToken(),
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
