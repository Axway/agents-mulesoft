package discovery

import (
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

func (m *mockAnypointClient) GetEnvironmentByName(name string) (*anypoint.Environment, error) {
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

func (m *mockAnypointClient) GetPolicies(api *anypoint.API) ([]anypoint.Policy, error) {
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

func (m *mockAnypointClient) GetExchangeAssetIcon(asset *anypoint.ExchangeAsset) (icon string, contentType string, err error) {
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

func TestDiscoverAPIs(t *testing.T) {
	mac := &mockAnypointClient{}
	assetCache := cache.New()
	buffer := 5

	a := Agent{
		discoveryPageSize: 1,
		anypointClient:    mac,
		stage:             "Sandbox",
		assetCache:        assetCache,
		stopDiscovery:     make(chan bool),
		pollInterval:      time.Second,
		apiChan:           make(chan *ServiceDetail, buffer),
	}

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
