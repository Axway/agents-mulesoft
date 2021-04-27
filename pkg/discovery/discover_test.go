package discovery

import (
	"fmt"
	"net/url"
	"testing"
	"time"

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
		ID:   12345,
		Name: "asset1",
		APIs: []anypoint.API{{
			ID:          6789,
			EndpointURI: "google.com",
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
			Classifier:   "oas",
			Packaging:    "",
			DownloadURL:  "",
			ExternalLink: "",
			MD5:          "",
			SHA1:         "",
			CreatedDate:  time.Time{},
			MainFile:     "",
			Generated:    false,
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
		assert.Equal(t, sd.Stage, a.stage)
		assert.Equal(t, sd.ResourceType, "oas2")
		assert.Equal(t, sd.ID, "6789")
		assert.Equal(t, sd.APISpec, []byte("{\"basePath\":\"google.com\",\"host\":\"\",\"schemes\":[\"\"],\"swagger\":\"2.0\"}"))

	case <-time.After(time.Second * 1):
		t.Errorf("Timed out waiting for the discovery")
	}
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
