package discovery

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"

	corecfg "github.com/Axway/agent-sdk/pkg/config"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/cache"

	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/stretchr/testify/mock"
)

type mockAnypointClient struct {
	mock.Mock
}

func (m *mockAnypointClient) OnConfigChange(*config.MulesoftConfig) {
	//noop
}

func (m *mockAnypointClient) GetAccessToken() (string, *anypoint.User, time.Duration, error) {
	return "", nil, 0, nil
}

func (m *mockAnypointClient) GetEnvironmentByName(_ string) (*anypoint.Environment, error) {
	return nil, nil
}

func (m *mockAnypointClient) ListAssets(*anypoint.Page) ([]anypoint.Asset, error) {
	assets := []anypoint.Asset{{
		ID:                12345,
		Name:              "asset1",
		ExchangeAssetName: "dummyasset",
		APIs: []anypoint.API{{
			ID:          6789,
			EndpointURI: "google.com",
			AssetID:     "12345",
		}},
	}}

	return assets, nil
}

func (m *mockAnypointClient) GetPolicies(*anypoint.API) ([]anypoint.Policy, error) {
	return nil, nil
}

func (m *mockAnypointClient) GetExchangeAsset(*anypoint.API) (*anypoint.ExchangeAsset, error) {
	return &anypoint.ExchangeAsset{
		Files: []anypoint.ExchangeFile{{
			Classifier: "oas",
			Generated:  false,
		}},
	}, nil
}

func (m *mockAnypointClient) GetExchangeAssetIcon(_ *anypoint.ExchangeAsset) (icon string, contentType string, err error) {
	return "", "", nil
}

func (m *mockAnypointClient) GetExchangeFileContent(*anypoint.ExchangeFile) (fileContent []byte, err error) {
	var a = `{
"swagger": "2.0"
}`
	return []byte(a), nil
}

func (m *mockAnypointClient) GetAnalyticsWindow() ([]anypoint.AnalyticsEvent, error) {
	return nil, nil
}

func getAgent() Agent {
	mac := &mockAnypointClient{}
	assetCache := cache.New()
	buffer := 5

	return Agent{
		discoveryPageSize: 1,
		anypointClient:    mac,
		stage:             "Sandbox",
		assetCache:        assetCache,
		stopDiscovery:     make(chan bool),
		pollInterval:      time.Second,
		apiChan:           make(chan *ServiceDetail, buffer),
	}
}

func TestDiscoverAPIs(t *testing.T) {
	a := getAgent()

	go func() {
		a.discoverAPIs()
	}()

	select {
	case sd := <-a.apiChan:
		assert.Equal(t, "dummyasset", sd.Title)
		assert.Equal(t, a.stage, sd.Stage)
		assert.Equal(t, "12345", sd.APIName)
		assert.Equal(t, "oas2", sd.ResourceType)
		assert.Equal(t, "6789", sd.ID)
		assert.Equal(t, []byte("{\"basePath\":\"google.com\",\"host\":\"\",\"schemes\":[\"\"],\"swagger\":\"2.0\"}"), sd.APISpec)
		assert.Equal(t, map[string]string{"checksum": "c3aad4f6ae04e69c1ff4f1d81abf40e8cae91acd22e16a798ac9a88a932286a7"}, sd.ServiceAttributes)

	case <-time.After(time.Second * 1):
		t.Errorf("Timed out waiting for the discovery")
	}
}

type mockClient struct {
	reqs map[string]*api.Response
}

func (mc *mockClient) Send(request api.Request) (*api.Response, error) {
	req, ok := mc.reqs[request.URL]
	if ok {
		return req, nil
	} else {
		return nil, fmt.Errorf("no request found for %s", request.URL)
	}
}

var mapOfResponses = map[string]*api.Response{
	"/accounts/login": {
		Code:    200,
		Body:    []byte("{\"access_token\":\"abc123\"}"),
		Headers: nil,
	},
	"/accounts/api/me": {
		Code: 200,
		Body: []byte(`{
					"user":{
						"identityType": "idtype",
						"id": "123",
						"username": "name",
						"firstName": "first",
						"lastName": "last",
						"email": "email",
						"organization": {
							"id": "333",
							"name": "org1",
							"domain": "abc.com"
						}
					}
				}`),
	},
	"/accounts/api/organizations/333/environments": {
		Code: 200,
		Body: []byte(`{
					"data": [{
						"id": "111",
						"name": "Sandbox",
						"organizationId": "333",
						"type": "fake",
						"clientId": "abc123"
					}],
					"total": 1
				}`),
	},
}

func buildAgentWithCustomMockAnypointClient() (*Agent, *mockClient) {
	ac := &config.AgentConfig{
		CentralConfig: corecfg.NewCentralConfig(corecfg.DiscoveryAgent),
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: 1 * time.Second,
		},
	}

	mc := &mockClient{}
	mc.reqs = make(map[string]*api.Response)
	for k, v := range mapOfResponses {
		mc.reqs[k] = v
	}

	apClient := anypoint.NewClient(ac.MulesoftConfig, anypoint.SetClient(mc))
	agent := New(ac, apClient)

	return agent, mc
}

func TestGetServiceDetailFailsWhenGetPoliciesFails(t *testing.T) {
	agent, mc := buildAgentWithCustomMockAnypointClient()

	mc.reqs["/apimanager/api/v1/organizations/333/environments/111/apis/456/policies"] = &api.Response{
		Code: 400,
		Body: nil,
	}

	asset := &anypoint.Asset{}

	a := &anypoint.API{
		ID:          456,
		EndpointURI: "google.com",
	}

	_, err := agent.getServiceDetail(asset, a)
	assert.NotNil(t, err)
}

func TestGetServiceDetailFailsWhenGetExchangeAssetFails(t *testing.T) {
	agent, mc := buildAgentWithCustomMockAnypointClient()

	mc.reqs["/apimanager/api/v1/organizations/333/environments/111/apis/456/policies"] = &api.Response{
		Code: 200,
		Body: []byte(`[{}]`),
	}

	mc.reqs["/exchange/api/v2/assets/123/456/89"] = &api.Response{
		Code: 400,
		Body: nil,
	}

	asset := &anypoint.Asset{}

	a := &anypoint.API{
		ID:           456,
		EndpointURI:  "google.com",
		GroupID:      "123",
		AssetID:      "456",
		AssetVersion: "89",
	}

	_, err := agent.getServiceDetail(asset, a)
	assert.NotNil(t, err)
}

func TestGetServiceDetailWhenExchangeAssetSpecFileIsEmpty(t *testing.T) {
	agent, mc := buildAgentWithCustomMockAnypointClient()

	mc.reqs["/apimanager/api/v1/organizations/333/environments/111/apis/456/policies"] = &api.Response{
		Code: 200,
		Body: []byte(`[{}]`),
	}

	mc.reqs["/exchange/api/v2/assets/123/456/89"] = &api.Response{
		Code: 200,
		Body: []byte("{\"Files\": [{}]}"),
	}

	asset := &anypoint.Asset{}

	a := &anypoint.API{
		ID:           456,
		EndpointURI:  "google.com",
		GroupID:      "123",
		AssetID:      "456",
		AssetVersion: "89",
	}

	sd, err := agent.getServiceDetail(asset, a)
	assert.Nil(t, sd)
	assert.Nil(t, err)
}

func TestGetServiceDetailFailsWhenGetSpecFromExchangeFileFails(t *testing.T) {
	agent, mc := buildAgentWithCustomMockAnypointClient()

	mc.reqs["/apimanager/api/v1/organizations/333/environments/111/apis/456/policies"] = &api.Response{
		Code: 200,
		Body: []byte(`[{}]`),
	}

	mc.reqs["/exchange/api/v2/assets/123/456/89"] = &api.Response{
		Code: 200,
		Body: []byte("{\"Files\": [{\"Classifier\":\"oas\",\"Generated\":false, \"ExternalLink\": \"https://localhost/swagger.json\"}]}"),
	}

	mc.reqs["https://localhost/swagger.json"] = &api.Response{
		Code: 400,
		Body: nil,
	}

	asset := &anypoint.Asset{}

	a := &anypoint.API{
		ID:           456,
		EndpointURI:  "google.com",
		GroupID:      "123",
		AssetID:      "456",
		AssetVersion: "89",
	}

	_, err := agent.getServiceDetail(asset, a)
	assert.NotNil(t, err)
}

func TestGetServiceDetailFailsWhenGetExchangeAssetIconFails(t *testing.T) {
	agent, mc := buildAgentWithCustomMockAnypointClient()

	mc.reqs["/apimanager/api/v1/organizations/333/environments/111/apis/456/policies"] = &api.Response{
		Code: 200,
		Body: []byte(`[{}]`),
	}

	mc.reqs["/exchange/api/v2/assets/123/456/89"] = &api.Response{
		Code: 200,
		Body: []byte("{\"Icon\": \"blahblah.com\", \"Files\": [{\"Classifier\":\"oas\",\"Generated\":false, \"ExternalLink\": \"https://localhost/swagger.json\"}]}"),
	}

	mc.reqs["https://localhost/swagger.json"] = &api.Response{
		Code: 200,
		Body: []byte("{\"basePath\":\"google.com\",\"host\":\"\",\"schemes\":[\"\"],\"swagger\":\"2.0\"}"),
	}

	mc.reqs["blahblah.com"] = &api.Response{
		Code: 400,
		Body: nil,
	}

	asset := &anypoint.Asset{}

	a := &anypoint.API{
		ID:           456,
		EndpointURI:  "google.com",
		GroupID:      "123",
		AssetID:      "456",
		AssetVersion: "89",
	}

	_, err := agent.getServiceDetail(asset, a)
	assert.NotNil(t, err)
}

func TestShouldDiscoverAPIBasedOnTags(t *testing.T) {
	tests := []struct {
		name     string
		a        Agent
		api      *anypoint.API
		expected bool
	}{
		{
			name: "Should discover if matching discovery tag exists on API",
			a: Agent{
				discoveryTags: []string{"discover"},
			},
			api:      &anypoint.API{Tags: []string{"discover"}},
			expected: true,
		},
		{
			name: "Should not discover if API has a tag to be ignored",
			a: Agent{
				discoveryIgnoreTags: []string{"donotdiscover"},
			},
			api:      &anypoint.API{Tags: []string{"donotdiscover"}},
			expected: false,
		},
		{
			name: "Should not discover if API does not have any tags that the agent's config has",
			a: Agent{
				discoveryIgnoreTags: []string{"donotdiscover"},
				discoveryTags:       []string{"discover"},
			},
			api:      &anypoint.API{Tags: []string{}},
			expected: false,
		},
		{
			name:     "Should discover if API as well as agent's config have no discovery tags",
			a:        Agent{},
			api:      &anypoint.API{Tags: []string{}},
			expected: true,
		},
		{
			name: "Should not discover if API has both - a tag to be discovered and a tag to be ignored",
			a: Agent{
				discoveryIgnoreTags: []string{"donotdiscover"},
				discoveryTags:       []string{"discover"},
			},
			api:      &anypoint.API{Tags: []string{"discover", "donotdiscover"}},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.a.shouldDiscoverAPI(tc.api))
		})
	}
}

func TestGetExchangeAssetSpecFile(t *testing.T) {
	a := getAgent()

	tests := []struct {
		name      string
		asset     *anypoint.ExchangeAsset
		exchgfile *anypoint.ExchangeFile
		err       error
	}{
		{
			name: "Should return nil and no error if the Exchange asset has no files",
			asset: &anypoint.ExchangeAsset{
				Name:  "Sample exchange asset 1",
				Files: nil,
			},
			exchgfile: nil,
			err:       nil,
		},
		{
			name: "Should return nil and no error if the Exchange asset has a file that is not of expected classifier",
			asset: &anypoint.ExchangeAsset{
				Name: "Sample exchange asset 2",
				Files: []anypoint.ExchangeFile{{
					Classifier: "oas3",
				},
				},
			},
			exchgfile: nil,
			err:       nil,
		},
		{
			name: "Should return an exchange asset if it has a file that is of expected classifier",
			asset: &anypoint.ExchangeAsset{
				Name: "Sample exchange asset 3",
				Files: []anypoint.ExchangeFile{{
					Classifier: "oas",
				},
				},
			},
			exchgfile: &anypoint.ExchangeFile{
				Classifier: "oas",
			},
			err: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sd, err := a.getExchangeAssetSpecFile(tc.asset)
			assert.Equal(t, tc.exchgfile, sd)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestSetOAS2Endpoint(t *testing.T) {
	a := getAgent()

	tests := []struct {
		name        string
		endPointURL string
		specContent []byte
		result      []byte
		err         error
	}{
		{
			name:        "Should return error if Endpoint URL is not valid",
			endPointURL: "postgres://user:abc{def=ghi@sdf.com:5432",
			specContent: []byte("{\"basePath\":\"google.com\",\"host\":\"\",\"schemes\":[\"\"],\"swagger\":\"2.0\"}"),
			result:      []byte("{\"basePath\":\"google.com\",\"host\":\"\",\"schemes\":[\"\"],\"swagger\":\"2.0\"}"),
			err: &url.Error{
				Op:  "parse",
				URL: "postgres://user:abc{def=ghi@sdf.com:5432",
				Err: fmt.Errorf("net/url: invalid userinfo"),
			},
		},
		{
			name:        "Should return error if the spec content is not a valid JSON",
			endPointURL: "http://google.com",
			specContent: []byte("google.com"),
			result:      []byte("google.com"),
			err:         fmt.Errorf("invalid character 'g' looking for beginning of value"),
		},
		{
			name:        "Should return spec that has OAS2 endpoint set",
			endPointURL: "http://google.com",
			specContent: []byte("{\"basePath\":\"google.com\",\"host\":\"\",\"schemes\":[\"\"],\"swagger\":\"2.0\"}"),
			result:      []byte("{\"basePath\":\"\",\"host\":\"google.com\",\"schemes\":[\"http\"],\"swagger\":\"2.0\"}"),
			err:         nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := a.setOAS2Endpoint(tc.endPointURL, tc.specContent)

			if err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			}

			assert.Equal(t, tc.result, spec)
		})
	}
}

func TestSetOAS3Endpoint(t *testing.T) {
	a := getAgent()

	tests := []struct {
		name        string
		url         string
		specContent []byte
		result      []byte
		err         error
	}{
		{
			name:        "Should return error if the spec content is not a valid JSON",
			url:         "google.com",
			specContent: []byte("google.com"),
			result:      []byte("google.com"),
			err:         fmt.Errorf("invalid character 'g' looking for beginning of value"),
		},
		{
			name:        "Should return spec that has OAS3 endpoint set",
			url:         "google.com",
			specContent: []byte("{\"openapi\": \"3.0.1\"}"),
			result:      []byte("{\"openapi\":\"3.0.1\",\"servers\":[{\"url\":\"google.com\"}]}"),
			err:         nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := a.setOAS3Endpoint(tc.url, tc.specContent)
			if err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			}
			assert.Equal(t, tc.result, spec)
		})
	}
}
