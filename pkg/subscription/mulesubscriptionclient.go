package subscription

import (
	"fmt"
	"strconv"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/common"
)

// MuleSubscriptionClient interface for managing mulesoft subscriptions
type MuleSubscriptionClient interface {
	CreateApp(appName string, apiID string, description string) (*anypoint.Application, error)
	CreateContract(apiID, tierID, appID string) (*anypoint.Contract, error)
	DeleteApp(appID string) error
	DeleteContract(apiID, contractID string) error
	GetApp(appID string) (*anypoint.Application, error)
	ResetAppSecret(appID string) (*anypoint.Application, error)
	CreateIfNotExistingSLATier(apiID string) (string, error)
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

// ResetAppSecret resets the secret for an app
func (c muleSubscription) ResetAppSecret(appID string) (*anypoint.Application, error) {
	return c.client.ResetAppSecret(appID)
}

// GetApp gets a mulesoft app by id
func (c muleSubscription) GetApp(appID string) (*anypoint.Application, error) {
	return c.client.GetClientApplication(appID)
}

// CreateApp creates an app in Mulesoft
func (c muleSubscription) CreateApp(appName string, apiID string, description string) (*anypoint.Application, error) {

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
func (c muleSubscription) CreateContract(apiID, tierID, appID string) (*anypoint.Contract, error) {
	api, err := c.client.GetAPI(apiID)
	if err != nil {
		return nil, err
	}

	// Need to fetch the exchange asset to get the version group
	exchangeAsset, err := c.client.GetExchangeAsset(api.GroupID, api.AssetID, api.AssetVersion)
	if err != nil {
		return nil, err
	}

	contract := newContract(apiID, exchangeAsset.VersionGroup, tierID, api)
	return c.client.CreateContract(appID, contract)
}

// DeleteApp deletes the mulesoft app
func (c muleSubscription) DeleteApp(appID string) error {
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

func (c muleSubscription) CreateIfNotExistingSLATier(apiID string) (string, error) {
	existingTiers, err := c.client.GetSLATiers(apiID, common.AxwayAgentSLATierName)
	if err != nil {
		return "", fmt.Errorf("error getting SLA tiers: %s", err)
	}
	for _, tier := range existingTiers.Tiers {
		if tier.Name == common.AxwayAgentSLATierName {
			return strconv.Itoa(*tier.ID), nil
		}
	}

	tierID, err := c.client.CreateSLATier(apiID)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(tierID), nil
}

func newContract(apiID, versionGroup string, tierID string, api *anypoint.API) *anypoint.Contract {
	tID, _ := strconv.Atoi(tierID)
	return &anypoint.Contract{
		AcceptedTerms:   true,
		ApiID:           apiID,
		AssetID:         api.AssetID,
		EnvironmentID:   api.EnvironmentID,
		GroupID:         api.GroupID,
		OrganizationID:  api.OrganizationID,
		RequestedTierID: tID,
		Version:         api.AssetVersion,
		VersionGroup:    versionGroup,
	}
}
