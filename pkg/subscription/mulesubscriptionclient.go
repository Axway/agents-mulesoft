package subscription

import (
	"fmt"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
)

// MuleSubscriptionClient interface for managing mulesoft subscriptions
type MuleSubscriptionClient interface {
	CreateApp(appName, apiID, description string) (*anypoint.Application, error)
	CreateContract(apiID, tier string, appID int64) (*anypoint.Contract, error)
	DeleteApp(appID int64) error
	DeleteContract(apiID, contractID string) error
	GetApp(id string) (*anypoint.Application, error)
}

type muleSubscription struct {
	client anypoint.Client
}

// NewMuleSubscriptionClient creates a MuleSubscriptionClient
func NewMuleSubscriptionClient(client anypoint.Client) MuleSubscriptionClient {
	return &muleSubscription{
		client: client,
	}
}

// GetApp gets a mulesoft app by id
func (c muleSubscription) GetApp(id string) (*anypoint.Application, error) {
	return c.client.GetClientApplication(id)
}

// CreateApp creates an app in Mulesoft
func (c muleSubscription) CreateApp(appName, apiID, description string) (*anypoint.Application, error) {

	body := &anypoint.AppRequestBody{
		Name:        appName,
		Description: description,
	}

	application, err := c.client.CreateClientApplication(apiID, body)
	if err != nil {
		return nil, fmt.Errorf("error creating client app: %s", err)
	}

	return application, nil
}

// CreateContract creates a contract between an API and an app
func (c muleSubscription) CreateContract(apiID, tier string, appID int64) (*anypoint.Contract, error) {
	api, err := c.client.GetAPI(apiID)
	if err != nil {
		return nil, err
	}

	tID := parseTierID(tier)

	// Need to fetch the exchange asset to get the version group
	exchangeAsset, err := c.client.GetExchangeAsset(api.GroupID, api.AssetID, api.AssetVersion)
	if err != nil {
		return nil, err
	}

	contract := newContract(apiID, exchangeAsset.VersionGroup, tID, api)

	return c.client.CreateContract(appID, contract)
}

// DeleteApp deletes the mulesoft app
func (c muleSubscription) DeleteApp(appID int64) error {
	return c.client.DeleteClientApplication(appID)
}

// DeleteContract removes the api from the app
func (c muleSubscription) DeleteContract(apiID, contractID string) error {
	err := c.client.RevokeContract(apiID, contractID)
	if err != nil {
		return err
	}
	return c.client.DeleteContract(apiID, contractID)
}

func newContract(apiID, versionGroup string, tID int64, api *anypoint.API) *anypoint.Contract {
	return &anypoint.Contract{
		AcceptedTerms:   true,
		APIID:           apiID,
		AssetID:         api.AssetID,
		EnvironmentID:   api.EnvironmentID,
		GroupID:         api.GroupID,
		OrganizationID:  api.OrganizationID,
		RequestedTierID: tID,
		Version:         api.AssetVersion,
		VersionGroup:    versionGroup,
	}
}
