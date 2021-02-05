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
	GetAccessToken() (string, *User, time.Duration, error)

	GetEnvironmentByName(name string) (*Environment, error)

	ListAssets(page *Page) ([]Asset, error)
	GetPolicies(api *API) ([]Policy, error)
	GetExchangeAsset(api *API) (*ExchangeAsset, error)
	GetExchangeAssetIcon(asset *ExchangeAsset) (icon string, contentType string, err error)
	GetExchangeFileContent(file *ExchangeFile) (fileContent []byte, err error)
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
}

// NewClient creates a new client for interacting with Mulesoft.
func NewClient(mulesoftConfig *config.MulesoftConfig) Client {
	client := &anypointClient{}
	client.OnConfigChange(mulesoftConfig)

	// Register the healthcheck
	hc.RegisterHealthcheck("Mulesoft Anypoint Exchange", "mulesoft", client.healthcheck)

	return client
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

// ListAssets lists the API Assets.
func (c *anypointClient) ListAssets(page *Page) ([]Asset, error) {
	var assetResult AssetSearch
	url := c.baseURL + "/apimanager/api/v1/organizations/" + c.auth.GetOrgID() + "/environments/" + c.environment.ID + "/apis"
	err := c.invokeJSONGet(url, page, &assetResult)

	if err != nil {
		return nil, err
	}

	return assetResult.Assets, err
}

// GetPolicies lists the API policies.
func (c *anypointClient) GetPolicies(api *API) ([]Policy, error) {
	var policies []Policy
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d/policies", c.baseURL, c.auth.GetOrgID(), c.environment.ID, api.ID)
	err := c.invokeJSONGet(url, nil, &policies)

	if err != nil {
		return nil, err
	}

	return policies, err
}

// GetExchangeAsset creates the AssetDetail form the Asset API.
func (c *anypointClient) GetExchangeAsset(api *API) (*ExchangeAsset, error) {
	var exchangeAsset ExchangeAsset
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s", c.baseURL, api.GroupID, api.AssetID, api.AssetVersion)
	err := c.invokeJSONGet(url, nil, &exchangeAsset)
	if err != nil {
		return nil, err
	}
	return &exchangeAsset, nil
}

// GetExchangeAssetIcon get the icon as a base64 encoded string from the Exchange Asset files.
func (c *anypointClient) GetExchangeAssetIcon(asset *ExchangeAsset) (string, string, error) {
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

	return base64.StdEncoding.EncodeToString([]byte(iconBuffer)), contentType, nil
}

// GetExchangeFileContent download the file from the ExternalLink reference. If the file is a zip file
// and thre is a MainFile set then the content of the MainFile is returned.
func (c *anypointClient) GetExchangeFileContent(file *ExchangeFile) (fileContent []byte, err error) {
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

func (c *anypointClient) invokeJSONGet(url string, page *Page, resp interface{}) error {
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

func (c *anypointClient) invokeJSON(request coreapi.Request, resp interface{}) error {
	body, _, err := c.invoke(request)
	if err != nil {
		return agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}

	err = json.Unmarshal(body, resp)
	if err != nil {
		return agenterrors.Wrap(ErrMarshallingBody, err.Error())
	}
	return nil
}

func (c *anypointClient) invokeGet(url string) ([]byte, map[string][]string, error) {
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		Headers:     nil,
		QueryParams: nil,
	}

	return c.invoke(request)
}

func (c *anypointClient) invoke(request coreapi.Request) ([]byte, map[string][]string, error) {
	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, nil, agenterrors.Wrap(ErrCommunicatingWithGateway, err.Error())
	}
	if response.Code != http.StatusOK {
		return nil, nil, agenterrors.Wrap(ErrCommunicatingWithGateway, fmt.Sprint(response.Code))
	}

	return response.Body, response.Headers, nil
}
