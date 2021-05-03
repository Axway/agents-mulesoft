package discovery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"

	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/stretchr/testify/mock"
)

const assetID = "petstore-3"

var asset = anypoint.Asset{
	APIs: []anypoint.API{
		{
			AssetID:       assetID,
			AssetVersion:  "1.0.0",
			EndpointURI:   "https://petstore3.us-e2.cloudhub.io",
			EnvironmentID: "e9a405ae-2789-4889-a267-548a1f7aa6f4",
			ID:            16810512,
			Tags:          []string{"tag1"},
		},
	},
	AssetID:              assetID,
	Audit:                anypoint.Audit{},
	AutodiscoveryAPIName: "groupId:d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4:assetId:petstore-3",
	ExchangeAssetName:    "petstore-3",
	GroupID:              "d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4",
	ID:                   211799904,
	MasterOrganizationID: "d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4",
	Name:                 "groupId:d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4:assetId:petstore-3",
	OrganizationID:       "d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4",
	TotalAPIs:            1,
}
var assets = []anypoint.Asset{asset}

func TestDiscovery_Loop(t *testing.T) {
	apiChan := make(chan *ServiceDetail)
	stopCh := make(chan bool)

	client := &mockAnypointClient{}
	client.On("ListAssets").Return(assets, nil)

	msh := &mockServiceHandler{}
	msh.On("ToServiceDetails").Return([]*ServiceDetail{sd})

	disc := &discovery{
		apiChan:           apiChan,
		assetCache:        cache.New(),
		client:            client,
		discoveryPageSize: 50,
		pollInterval:      0001 * time.Second,
		stopDiscovery:     stopCh,
		serviceHandler:    msh,
	}

	go disc.Loop()

	// accounts for the immediate tick, and two ticks of the pollInterval
	count := 0
	for count < 3 {
		select {
		case <-disc.apiChan:
			// assert.Equal(t, )
			count++
		}
	}
	disc.Stop()
	disc.OnConfigChange(&config.MulesoftConfig{})
}

func Test_discoverAPIs(t *testing.T) {
	tests := []struct {
		name           string
		pageSize       int
		err            error
		expectedAssets int
		listSize       int
	}{
		{
			name:           "should fetch more assets when the returned length is equal to the page size",
			pageSize:       3,
			listSize:       3,
			expectedAssets: 6,
			err:            nil,
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			apiChan := make(chan *ServiceDetail)
			stopCh := make(chan bool)
			client := &mockAnypointClient{}
			client.On("ListAssets").Return(make([]anypoint.Asset, tc.listSize), tc.err)
			client.On("ListAssets").Return(make([]anypoint.Asset, 0), tc.err)
			msh := &mockServiceHandler{}
			msh.On("ToServiceDetails").Return([]*ServiceDetail{sd})
			disc := &discovery{
				apiChan:           apiChan,
				assetCache:        cache.New(),
				client:            client,
				discoveryPageSize: tc.pageSize,
				pollInterval:      0001 * time.Second,
				stopDiscovery:     stopCh,
				serviceHandler:    msh,
			}
			go disc.discoverAPIs()

			calls := 0
			for calls < tc.expectedAssets {
				select {
				case <-disc.apiChan:
					calls++
				}
			}

			assert.Equal(t, tc.expectedAssets, calls)
			logrus.Info(client)
		})
	}

}

type mockServiceHandler struct {
	mock.Mock
	count int
}

func (m *mockServiceHandler) ToServiceDetails(*anypoint.Asset) []*ServiceDetail {
	args := m.Called()
	result := args.Get(0)
	return result.([]*ServiceDetail)
}

func (m *mockServiceHandler) OnConfigChange(_ *config.MulesoftConfig) {
}

type mockAnypointClient struct {
	mock.Mock
}

func (m *mockAnypointClient) OnConfigChange(*config.MulesoftConfig) {
}

func (m *mockAnypointClient) GetAccessToken() (string, *anypoint.User, time.Duration, error) {
	return "", nil, 0, nil
}

func (m *mockAnypointClient) GetEnvironmentByName(_ string) (*anypoint.Environment, error) {
	return nil, nil
}

func (m *mockAnypointClient) ListAssets(*anypoint.Page) ([]anypoint.Asset, error) {
	args := m.Called()
	result := args.Get(0)
	return result.([]anypoint.Asset), args.Error(1)

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

// var mapOfResponses = map[string]*api.Response{
// 	"/accounts/login": {
// 		Code:    200,
// 		Body:    []byte("{\"access_token\":\"abc123\"}"),
// 		Headers: nil,
// 	},
// 	"/accounts/api/me": {
// 		Code: 200,
// 		Body: []byte(`{
// 					"user":{
// 						"identityType": "idtype",
// 						"id": "123",
// 						"username": "name",
// 						"firstName": "first",
// 						"lastName": "last",
// 						"email": "email",
// 						"organization": {
// 							"id": "333",
// 							"name": "org1",
// 							"domain": "abc.com"
// 						}
// 					}
// 				}`),
// 	},
// 	"/accounts/api/organizations/333/environments": {
// 		Code: 200,
// 		Body: []byte(`{
// 					"data": [{
// 						"id": "111",
// 						"name": "Sandbox",
// 						"organizationId": "333",
// 						"type": "fake",
// 						"clientId": "abc123"
// 					}],
// 					"total": 1
// 				}`),
// 	},
// }
//
// func buildAgentWithCustomMockAnypointClient() (*Agent, *anypoint.MockClient) {
// 	ac := &config.AgentConfig{
// 		CentralConfig: corecfg.NewCentralConfig(corecfg.DiscoveryAgent),
// 		MulesoftConfig: &config.MulesoftConfig{
// 			PollInterval: 1 * time.Second,
// 		},
// 	}
//
// 	mc := &anypoint.MockClient{}
// 	mc.Reqs = make(map[string]*api.Response)
// 	for k, v := range mapOfResponses {
// 		mc.Reqs[k] = v
// 	}
//
// 	apClient := anypoint.NewClient(ac.MulesoftConfig, anypoint.SetClient(mc))
// 	agent := New(ac, apClient)
//
// 	return agent, mc
// }
//
// func TestGetServiceDetailFailsWhenGetPoliciesFails(t *testing.T) {
// 	agent, mc := buildAgentWithCustomMockAnypointClient()
//
// 	mc.Reqs["/apimanager/api/v1/organizations/333/environments/111/apis/456/policies"] = &api.Response{
// 		Code: 400,
// 		Body: nil,
// 	}
//
// 	asset := &anypoint.Asset{}
//
// 	a := &anypoint.API{
// 		ID:          456,
// 		EndpointURI: "google.com",
// 	}
//
// 	_, err := agent.getServiceDetail(asset, a)
// 	assert.NotNil(t, err)
// }
//
// func TestGetServiceDetailFailsWhenGetExchangeAssetFails(t *testing.T) {
// 	agent, mc := buildAgentWithCustomMockAnypointClient()
//
// 	mc.Reqs["/apimanager/api/v1/organizations/333/environments/111/apis/456/policies"] = &api.Response{
// 		Code: 200,
// 		Body: []byte(`[{}]`),
// 	}
//
// 	mc.Reqs["/exchange/api/v2/assets/123/456/89"] = &api.Response{
// 		Code: 400,
// 		Body: nil,
// 	}
//
// 	asset := &anypoint.Asset{}
//
// 	a := &anypoint.API{
// 		ID:           456,
// 		EndpointURI:  "google.com",
// 		GroupID:      "123",
// 		AssetID:      "456",
// 		AssetVersion: "89",
// 	}
//
// 	_, err := agent.getServiceDetail(asset, a)
// 	assert.NotNil(t, err)
// }
//
// func TestGetServiceDetailWhenExchangeAssetSpecFileIsEmpty(t *testing.T) {
// 	agent, mc := buildAgentWithCustomMockAnypointClient()
//
// 	mc.Reqs["/apimanager/api/v1/organizations/333/environments/111/apis/456/policies"] = &api.Response{
// 		Code: 200,
// 		Body: []byte(`[{}]`),
// 	}
//
// 	mc.Reqs["/exchange/api/v2/assets/123/456/89"] = &api.Response{
// 		Code: 200,
// 		Body: []byte("{\"Files\": [{}]}"),
// 	}
//
// 	asset := &anypoint.Asset{}
//
// 	a := &anypoint.API{
// 		ID:           456,
// 		EndpointURI:  "google.com",
// 		GroupID:      "123",
// 		AssetID:      "456",
// 		AssetVersion: "89",
// 	}
//
// 	sd, err := agent.getServiceDetail(asset, a)
// 	assert.Nil(t, sd)
// 	assert.Nil(t, err)
// }
//
// func TestGetServiceDetailFailsWhenGetSpecFromExchangeFileFails(t *testing.T) {
// 	agent, mc := buildAgentWithCustomMockAnypointClient()
//
// 	mc.Reqs["/apimanager/api/v1/organizations/333/environments/111/apis/456/policies"] = &api.Response{
// 		Code: 200,
// 		Body: []byte(`[{}]`),
// 	}
//
// 	mc.Reqs["/exchange/api/v2/assets/123/456/89"] = &api.Response{
// 		Code: 200,
// 		Body: []byte("{\"Files\": [{\"Classifier\":\"oas\",\"Generated\":false, \"ExternalLink\": \"https://localhost/swagger.json\"}]}"),
// 	}
//
// 	mc.Reqs["https://localhost/swagger.json"] = &api.Response{
// 		Code: 400,
// 		Body: nil,
// 	}
//
// 	asset := &anypoint.Asset{}
//
// 	a := &anypoint.API{
// 		ID:           456,
// 		EndpointURI:  "google.com",
// 		GroupID:      "123",
// 		AssetID:      "456",
// 		AssetVersion: "89",
// 	}
//
// 	_, err := agent.getServiceDetail(asset, a)
// 	assert.NotNil(t, err)
// }
//
