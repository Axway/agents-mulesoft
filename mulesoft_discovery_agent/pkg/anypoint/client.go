package anypoint

import (
	"encoding/json"
	"fmt"
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

	// ListAssets(page *Page) ([]Asset, error)
	// GetAssetDetails(asset *Asset) (*AssetDetails, error)
	// GetAssetIcon(asset *Asset) (string, error)
	// GetAssetSpecification(asset *AssetDetails) (specContent []byte, packaging string, err error)
	// GetAssetHomePage(orgID string, groupID string, assetID string, version string) error // TODO RETURN
	// GetAssetLinkedAPI(orgID string, environmentID string, assetID string) error          // TODO RETURN
	// API Details
	// API Details Icon
	// API Details Instances

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
	c.lifetime = mulesoftConfig.SessionLifetime
	c.apiClient = coreapi.NewClient(mulesoftConfig.TLS, mulesoftConfig.ProxyURL)

	if c.auth != nil {
		c.auth.Stop()
	}

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

// GetEnvironmentByName -
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

// // ListAssets lists the managed assets in Mulesoft: https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/f1e97bc6-315a-4490-82a7-23abe036327a.anypoint-platform/exchange-experience-api/minor/2.0/console/method/%231431/
// func (c *anypointClient) ListAssets(page *Page) ([]Asset, error) {
// 	assets := make([]Asset, 0, page.PageSize)
// 	err := c.invokeJSONGet(c.baseURL+"/exchange/api/v2/assets", page, &assets)
// 	return assets, err
//}

// // GetAssetDetails creates the AssetDetail form the Asset.
// func (c *anypointClient) GetAssetDetails(asset *Asset) (*AssetDetails, error) {
// 	var assetDetails AssetDetails
// 	err := c.invokeJSONGet(c.baseURL+"/exchange/api/v2/assets/"+asset.ID, nil, &assetDetails)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &assetDetails, nil
// }

// // GetAssetIcon creates the AssetDetail form the Asset.
// func (c *anypointClient) GetAssetIcon(asset *Asset) (string, error) {
// 	if asset.Icon == "" {
// 		return "", nil
// 	}
// 	icon, err := c.invokeGet(asset.Icon)
// 	if err != nil {
// 		return "", err
// 	}
// 	return base64.StdEncoding.EncodeToString([]byte(icon)), nil
// }

// func (c *anypointClient) GetAssetSpecification(asset *AssetDetails) (specContent []byte, packaging string, err error) {
// 	packaging = "txt"
// 	specFileClassifier := asset.AssetType
// 	if val, ok := assetTypeMap[asset.AssetType]; ok {
// 		specFileClassifier = val
// 	}

// 	// Find the spec:
// 	// - known classifier
// 	// - mainfile from zip packages with spec type oas or wsdl,
// 	// - type oas with classifier fat-raml
// 	// - jar packages

// 	files := c.filterFiles(asset.Files, specFileClassifier)
// 	if len(files) > 0 {
// 		specContent, packaging, err = c.getFileSpec(files[0])
// 		if err != nil {
// 			return nil, "", err
// 		}

// 	} else if specFileClassifier == "oas" {
// 		// for rest-api, there might not be an oas spec generated if it's an old asset, download raml
// 		files = c.filterFiles(asset.Files, "fat-raml")
// 		if len(files) > 0 {
// 			specContent, packaging, err = c.getFileSpec(files[0])
// 			if err != nil {
// 				return nil, "", err
// 			}
// 		}

// 	} else {
// 		// majority have connector as a jar, so search for that
// 		files = c.filterFiles(asset.Files, "jar")
// 		if len(files) > 0 {
// 			specContent, packaging, err = c.getFileSpec(files[0])
// 			if err != nil {
// 				return nil, "", err
// 			}
// 		}
// 	}

// 	return specContent, packaging, err
// }

// // Filter the files by classifier
// func (c *anypointClient) filterFiles(files []File, classifier string) []File {
// 	filtered := []File{}

// 	for _, file := range files {
// 		if file.Classifier == classifier {
// 			filtered = append(filtered, file)
// 		}
// 	}
// 	return filtered
// }

// // getFileSpec loads the spec from the external link of the file.
// func (c *anypointClient) getFileSpec(file File) (specContent []byte, packaging string, err error) {
// 	packaging = file.Packaging
// 	specContent, err = c.invokeGet(file.ExternalLink)

// 	if file.Packaging == "zip" && file.MainFile != "" {
// 		zipReader, err := zip.NewReader(bytes.NewReader(specContent), int64(len(specContent)))
// 		if err != nil {
// 			return nil, "", err
// 		}

// 		for _, f := range zipReader.File {
// 			if f.Name == file.MainFile {
// 				content, err := f.Open()
// 				if err != nil {
// 					return nil, "", err
// 				}

// 				specContent, err = ioutil.ReadAll(content)
// 				content.Close()
// 				if err != nil {
// 					return nil, "", err
// 				}
// 				break
// 			}
// 		}
// 		packaging = "json"
// 	}
// 	return specContent, packaging, err
// }

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

func (c *anypointClient) invokeGet(url string) ([]byte, error) {
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		Headers:     nil,
		QueryParams: nil,
	}

	return c.invoke(request)
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
