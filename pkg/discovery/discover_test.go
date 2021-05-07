package discovery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sirupsen/logrus"

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
	ID:                   16810512,
	MasterOrganizationID: "d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4",
	Name:                 "groupId:d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4:assetId:petstore-3",
	OrganizationID:       "d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4",
	TotalAPIs:            1,
}
var assets = []anypoint.Asset{asset}

func TestDiscovery_Loop(t *testing.T) {
	apiChan := make(chan *ServiceDetail)
	stopCh := make(chan bool)

	client := &anypoint.MockAnypointClient{}
	client.On("ListAssets").Return(assets, nil)

	msh := &mockServiceHandler{}
	msh.On("ToServiceDetails").Return([]*ServiceDetail{sd})

	disc := &discovery{
		apiChan:           apiChan,
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
			client := &anypoint.MockAnypointClient{}
			client.On("ListAssets").Return(make([]anypoint.Asset, tc.listSize), tc.err)
			client.On("ListAssets").Return(make([]anypoint.Asset, 0), tc.err)
			msh := &mockServiceHandler{}
			msh.On("ToServiceDetails").Return([]*ServiceDetail{sd})
			disc := &discovery{
				apiChan:           apiChan,
				client:            client,
				discoveryPageSize: tc.pageSize,
				pollInterval:      0001 * time.Second,
				stopDiscovery:     stopCh,
				serviceHandler:    msh,
			}
			go disc.discoverAPIs()

			svc := <-disc.apiChan

			assert.Equal(t, sd, svc)
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
