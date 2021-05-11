package subscription

import (
	"fmt"
	"testing"

	"github.com/Axway/agents-mulesoft/pkg/subscription/mocks"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewContractBase(t *testing.T) {
	type createAppReturns struct {
		app *anypoint.Application
		err error
	}

	type getExchangeAssetReturns struct {
		asset *anypoint.ExchangeAsset
		err   error
	}

	type createContractReturns struct {
		contract *anypoint.Contract
		err      error
	}

	tests := []struct {
		name         string
		car          createAppReturns
		gear         getExchangeAssetReturns
		ccr          createContractReturns
		secondaryKey string
		hasError     bool
	}{
		{
			name:         "should throw an error when the mule api is not in the cache",
			hasError:     true,
			secondaryKey: "nothere",
			car: createAppReturns{
				app: &anypoint.Application{
					Name:         "app1",
					Description:  "app1",
					ClientID:     "1",
					ClientSecret: "2",
					ID:           1,
					APIEndpoints: false,
				},
				err: nil,
			},
			gear: getExchangeAssetReturns{
				asset: &anypoint.ExchangeAsset{
					Version: "1.0.2",
				},
				err: nil,
			},
			ccr: createContractReturns{
				contract: &anypoint.Contract{},
				err:      nil,
			},
		},
		{
			name:         "should subscribe with no error",
			secondaryKey: mockRemoteAPIID,
			hasError:     false,
			car: createAppReturns{
				app: &anypoint.Application{
					Name:         "app1",
					Description:  "app1",
					ClientID:     "1",
					ClientSecret: "2",
					ID:           1,
					APIEndpoints: false,
				},
				err: nil,
			},
			gear: getExchangeAssetReturns{
				asset: &anypoint.ExchangeAsset{
					Version: "1.0.2",
				},
				err: nil,
			},
			ccr: createContractReturns{
				contract: &anypoint.Contract{},
				err:      nil,
			},
		},
		{
			name:         "should throw an error when creating an app",
			secondaryKey: mockRemoteAPIID,
			hasError:     true,
			car: createAppReturns{
				app: nil,
				err: fmt.Errorf("yep"),
			},
			gear: getExchangeAssetReturns{
				asset: &anypoint.ExchangeAsset{
					Version: "1.0.2",
				},
				err: nil,
			},
			ccr: createContractReturns{
				contract: &anypoint.Contract{},
				err:      nil,
			},
		},
		{
			name:         "should throw an error when getting the exchange asset",
			secondaryKey: mockRemoteAPIID,
			hasError:     true,
			car: createAppReturns{
				app: &anypoint.Application{
					Name:         "app1",
					Description:  "app1",
					ClientID:     "1",
					ClientSecret: "2",
					ID:           1,
					APIEndpoints: false,
				},
				err: nil,
			},
			gear: getExchangeAssetReturns{
				asset: nil,
				err:   fmt.Errorf("err"),
			},
			ccr: createContractReturns{
				contract: &anypoint.Contract{},
				err:      nil,
			},
		},
		{
			name:         "should throw an error when creating the contract",
			secondaryKey: mockRemoteAPIID,
			hasError:     true,
			car: createAppReturns{
				app: &anypoint.Application{
					Name:         "app1",
					Description:  "app1",
					ClientID:     "1",
					ClientSecret: "2",
					ID:           1,
					APIEndpoints: false,
				},
				err: nil,
			},
			gear: getExchangeAssetReturns{
				asset: &anypoint.ExchangeAsset{
					Version: "1.0.2",
				},
				err: nil,
			},
			ccr: createContractReturns{
				contract: nil,
				err:      fmt.Errorf("error"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &anypoint.MockAnypointClient{}
			mockContract := &mocks.MockContract{}

			client.On("CreateClientApplication").Return(tc.car.app, tc.car.err)
			client.On("GetExchangeAsset").Return(tc.gear.asset, tc.gear.err)
			client.On("CreateContract").Return(tc.ccr.contract, tc.ccr.err)

			muleAPI := anypoint.API{
				ActiveContractsCount:      0,
				AssetID:                   "petstore2",
				AssetVersion:              "1.0.2",
				Audit:                     anypoint.Audit{},
				AutodiscoveryInstanceName: "1.0.5:16806513",
				Deprecated:                false,
				Description:               "",
				EndpointURI:               "http://pestore2.us-e2.cloudhub.io",
				EnvironmentID:             "8918f696-ea4c-4876-970c-85dec0ce367c",
				GroupID:                   "6a574278-a8c0-4643-81fc-7baf3fc346d3",
				ID:                        16806513,
				InstanceLabel:             "",
				IsPublic:                  false,
				MasterOrganizationID:      "6a574278-a8c0-4643-81fc-7baf3fc346d3",
				Order:                     0,
				OrganizationID:            "6a574278-a8c0-4643-81fc-7baf3fc346d3",
				Pinned:                    false,
				ProductVersion:            "1.0.5",
				Tags:                      nil,
			}

			cache.GetCache().SetWithSecondaryKey(fmt.Sprintf("%s-sandbox", mockSub.RemoteAPIID), tc.secondaryKey, muleAPI)

			base := NewContractBase(client, mockContract)
			cID, scrt, err := base.doSubscribe(logrus.StandardLogger(), mockSub)
			if tc.hasError == true {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.car.app.ClientID, cID)
				assert.Equal(t, tc.car.app.ClientSecret, scrt)
			}
		})
	}
}
