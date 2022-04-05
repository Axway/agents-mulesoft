package subscription

import (
	"fmt"
	"testing"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/stretchr/testify/assert"
)

func TestCreateApp(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		hasErr bool
	}{
		{
			name:   "should create an app",
			hasErr: false,
		},
		{
			name:   "should return an error when creating an app",
			err:    fmt.Errorf("err"),
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app1 := &anypoint.Application{
				Name:        "app one",
				ID:          123,
				Description: "description",
			}
			client := &anypoint.MockAnypointClient{}
			client.On("CreateClientApplication").Return(app1, tc.err)
			subClient := NewMuleSubscriptionClient(client)

			_, err := subClient.CreateApp(app1.Name, "apiID-123", app1.Description)
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCreateContract(t *testing.T) {
	tests := []struct {
		name              string
		hasErr            bool
		getAPIErr         error
		getAssetErr       error
		createContractErr error
	}{
		{
			name:   "should create a contract",
			hasErr: false,
		},
		{
			name:      "should return an error when failing to retrieve the api",
			hasErr:    true,
			getAPIErr: fmt.Errorf("get api err"),
		},
		{
			name:        "should return an error when failing to retrieve the exchange asset",
			hasErr:      true,
			getAssetErr: fmt.Errorf("get exchange asset err"),
		},
		{
			name:              "should return an error when failing to create the contract",
			hasErr:            true,
			createContractErr: fmt.Errorf("create contract err"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := &anypoint.API{
				AssetID:        "petstore-3",
				AssetVersion:   "2.2.3",
				EnvironmentID:  "e9a405ae-2789-4889-a267-548a1f7aa6f4",
				GroupID:        "d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4",
				ID:             16810512,
				Order:          1,
				OrganizationID: "d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4",
				ProductVersion: "v2",
			}
			asset := &anypoint.ExchangeAsset{
				VersionGroup: "v1",
			}

			client := &anypoint.MockAnypointClient{}
			client.On("GetAPI").Return(api, tc.getAPIErr)
			client.On("GetExchangeAsset").Return(asset, tc.getAssetErr)
			client.On("CreateContract").Return(&anypoint.Contract{}, tc.createContractErr)
			subClient := NewMuleSubscriptionClient(client)

			apiIDStr := fmt.Sprintf("%d", api.ID)
			var tID int64 = 7654
			tIDStr := fmt.Sprintf("%d", tID)

			contract, err := subClient.CreateContract(apiIDStr, tIDStr, 123)
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.True(t, contract.AcceptedTerms)
				assert.Equal(t, apiIDStr, contract.APIID)
				assert.Equal(t, api.AssetID, contract.AssetID)
				assert.Equal(t, api.EnvironmentID, contract.EnvironmentID)
				assert.Equal(t, api.GroupID, contract.GroupID)
				assert.Equal(t, api.OrganizationID, contract.OrganizationID)
				assert.Equal(t, tID, contract.RequestedTierID)
				assert.Equal(t, api.AssetVersion, contract.Version)
				assert.Equal(t, asset.VersionGroup, contract.VersionGroup)
				assert.Nil(t, err)
			}
		})
	}
}

func TestDeleteApp(t *testing.T) {
	tests := []struct {
		name         string
		hasErr       bool
		deleteAppErr error
	}{
		{
			name:   "should delete an app",
			hasErr: false,
		},
		{
			name:         "should return an error when deleting an app",
			hasErr:       true,
			deleteAppErr: fmt.Errorf("err"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var appID int64 = 123
			app1 := &anypoint.Application{
				Name:        "app one",
				ID:          appID,
				Description: "description",
			}
			client := &anypoint.MockAnypointClient{}
			client.On("DeleteClientApplication").Return(tc.deleteAppErr)
			subClient := NewMuleSubscriptionClient(client)

			err := subClient.DeleteApp(app1.ID)
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestDeleteContract(t *testing.T) {
	tests := []struct {
		name           string
		hasErr         bool
		delContractErr error
		revokeErr      error
	}{
		{
			name:   "should delete a contract",
			hasErr: false,
		},
		{
			name:           "should return an error when deleting a contract",
			hasErr:         true,
			delContractErr: fmt.Errorf("err"),
		},
		{
			name:      "should return an error when revoking a contract",
			hasErr:    true,
			revokeErr: fmt.Errorf("err"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &anypoint.MockAnypointClient{}
			client.On("RevokeContract").Return(tc.revokeErr)
			client.On("DeleteContract").Return(tc.delContractErr)
			subClient := NewMuleSubscriptionClient(client)

			err := subClient.DeleteContract("123", "456")
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
