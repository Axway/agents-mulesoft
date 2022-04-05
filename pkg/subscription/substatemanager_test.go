package subscription

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agents-mulesoft/pkg/common"

	"github.com/Axway/agent-sdk/pkg/apic"

	"github.com/Axway/agents-mulesoft/pkg/subscription/mocks"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var muleAPI = anypoint.API{
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

func Test_doSubscribeErrors(t *testing.T) {
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

			cache.GetCache().SetWithSecondaryKey(fmt.Sprintf("%s-sandbox", mockSub.RemoteAPIID), tc.secondaryKey, muleAPI)

			base := NewSubStateManager(client, mockContract)
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

func Test_doSubscribeSuccess(t *testing.T) {
	// should subscribe with no error
	apiID := mockSub.RemoteAPIAttributes[common.AttrAPIID]

	// set an item in the cache so it can be found by the function.
	cache.GetCache().SetWithSecondaryKey("fake-checksum", common.FormatAPICacheKey(apiID, mockSub.RemoteAPIStage), muleAPI)

	client := &anypoint.MockAnypointClient{}
	mockContract := &mocks.MockContract{}
	appID := int64(987)
	app := &anypoint.Application{
		Name:         "app1",
		Description:  "app1",
		ClientID:     "1",
		ClientSecret: "2",
		ID:           appID,
		APIEndpoints: false,
	}
	asset := &anypoint.ExchangeAsset{
		Version: "1.0.2",
	}
	client.On("CreateClientApplication").Return(app, nil)
	client.On("GetExchangeAsset").Return(asset, nil)
	client.On("CreateContract").Return(&anypoint.Contract{}, nil)

	base := NewSubStateManager(client, mockContract)
	cID, secret, err := base.doSubscribe(logrus.StandardLogger(), mockSub)

	assert.Nil(t, err)
	assert.Equal(t, app.ClientID, cID)
	assert.Equal(t, app.ClientSecret, secret)

	// should subscribe with no tier id
	mockSub.PropertyVals[anypoint.TierLabel] = ""
	client = &anypoint.MockAnypointClient{}
	base = NewSubStateManager(client, mockContract)
	client.On("CreateClientApplication").Return(app, nil)
	client.On("GetExchangeAsset").Return(asset, nil)
	client.On("CreateContract").Return(&anypoint.Contract{}, nil)
	cID, secret, err = base.doSubscribe(logrus.StandardLogger(), mockSub)

	assert.Nil(t, err)
	assert.Equal(t, app.ClientID, cID)
	assert.Equal(t, app.ClientSecret, secret)

}

func Test_createContract(t *testing.T) {
	client := &anypoint.MockAnypointClient{
		CreateContractAssertArgs: true,
	}
	mockContract := &mocks.MockContract{}

	base := NewSubStateManager(client, mockContract)

	asset := &anypoint.ExchangeAsset{
		VersionGroup: "v1",
	}
	apiID := "fake-api-id"
	appID := int64(1234)
	tID := int64(566)

	contract := &anypoint.Contract{
		APIID:           apiID,
		EnvironmentID:   muleAPI.EnvironmentID,
		AcceptedTerms:   true,
		OrganizationID:  muleAPI.OrganizationID,
		GroupID:         muleAPI.GroupID,
		AssetID:         muleAPI.AssetID,
		Version:         muleAPI.AssetVersion,
		VersionGroup:    asset.VersionGroup,
		RequestedTierID: tID,
	}

	client.On("CreateContract", appID, contract).Return(contract, nil)

	var err error
	contract, err = base.createContract(apiID, appID, tID, &muleAPI, asset)
	assert.Nil(t, err)

	client.AssertExpectations(t)

	contractArg := client.Mock.ExpectedCalls[0].Arguments[1].(*anypoint.Contract)
	assert.Equal(t, appID, client.Mock.ExpectedCalls[0].Arguments[0])
	assert.Equal(t, apiID, contractArg.APIID)
	assert.Equal(t, muleAPI.EnvironmentID, contractArg.EnvironmentID)
	assert.True(t, contractArg.AcceptedTerms)
	assert.Equal(t, muleAPI.OrganizationID, contractArg.OrganizationID)
	assert.Equal(t, muleAPI.GroupID, contractArg.GroupID)
	assert.Equal(t, muleAPI.AssetID, contractArg.AssetID)
	assert.Equal(t, asset.VersionGroup, contractArg.VersionGroup)
	assert.Equal(t, tID, contractArg.RequestedTierID)
}

func Test_createApp(t *testing.T) {
	client := &anypoint.MockAnypointClient{
		CreateContractAssertArgs: true,
	}
	mockContract := &mocks.MockContract{}

	base := NewSubStateManager(client, mockContract)

	apiID := "fake-api-id"

	appBody := &anypoint.AppRequestBody{
		Name:        util.ToString(mockSub.PropertyVals[anypoint.AppName]),
		Description: util.ToString(mockSub.PropertyVals[anypoint.Description]),
	}
	client.On("CreateClientApplication", apiID, appBody).Return(&anypoint.Application{}, nil)

	_, err := base.createApp(apiID, mockSub)
	assert.Nil(t, err)

	client.AssertExpectations(t)

	assert.Equal(t, apiID, client.Mock.ExpectedCalls[0].Arguments[0])
	appArg := client.Mock.ExpectedCalls[0].Arguments[1].(*anypoint.AppRequestBody)
	assert.Equal(t, appBody.Name, appArg.Name)
	assert.Equal(t, appBody.Description, appArg.Description)
}

func TestSubscribe(t *testing.T) {
	cache.GetCache().SetWithSecondaryKey(fmt.Sprintf("%s-sandbox", mockSub.RemoteAPIID), mockRemoteAPIID, muleAPI)

	client := &anypoint.MockAnypointClient{}
	mockContract := &mocks.MockContract{}

	clientID := "1"
	clientSecret := "2"
	client.On("CreateClientApplication").Return(&anypoint.Application{
		Name:         "app1",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ID:           1,
	}, nil)
	client.On("GetExchangeAsset").Return(&anypoint.ExchangeAsset{}, nil)
	client.On("CreateContract").Return(&anypoint.Contract{}, nil)

	base := NewSubStateManager(client, mockContract)
	err := base.Subscribe(logrus.StandardLogger(), mockSub)

	assert.Equal(t, apic.SubscriptionActive, mockSub.State)

	assert.Nil(t, err)

	client = &anypoint.MockAnypointClient{}
	client.On("CreateClientApplication").Return(&anypoint.Application{}, fmt.Errorf("err"))
	base = NewSubStateManager(client, mockContract)

	err = base.Subscribe(logrus.StandardLogger(), mockSub)
	assert.Nil(t, err)
	assert.Equal(t, apic.SubscriptionFailedToSubscribe, mockSub.State)
	assert.Equal(t, mockSub.ReceivedValues[anypoint.ClientIDProp], clientID)
	assert.Equal(t, mockSub.ReceivedValues[anypoint.ClientSecretProp], clientSecret)
}

func TestUnsubscribe(t *testing.T) {
	cache.GetCache().Set(util.ToString(mockSub.PropertyVals[anypoint.AppName]), int64(64))

	client := &anypoint.MockAnypointClient{}
	mockContract := &mocks.MockContract{}
	mockContract.On("Name").Return("first")
	mockContract.On("Schema").Return("fake schema")

	client.On("DeleteClientApplication").Return(nil)

	base := NewSubStateManager(client, mockContract)
	err := base.Unsubscribe(logrus.StandardLogger(), mockSub)
	assert.Nil(t, err)
	assert.Equal(t, apic.SubscriptionUnsubscribed, mockSub.State)

	client = &anypoint.MockAnypointClient{}
	client.On("DeleteClientApplication").Return(fmt.Errorf("hi"))
	base = NewSubStateManager(client, mockContract)
	err = base.Unsubscribe(logrus.StandardLogger(), mockSub)
	assert.Nil(t, err)
	assert.Equal(t, apic.SubscriptionFailedToSubscribe, mockSub.State)

	cache.GetCache().Delete(util.ToString(mockSub.PropertyVals[anypoint.AppName]))
	err = base.Unsubscribe(logrus.StandardLogger(), mockSub)
	assert.NotNil(t, err)

	cache.GetCache().Set(util.ToString(mockSub.PropertyVals[anypoint.AppName]), "string")
	err = base.Unsubscribe(logrus.StandardLogger(), mockSub)
	assert.NotNil(t, err)
}
