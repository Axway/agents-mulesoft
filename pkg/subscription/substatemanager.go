package subscription

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Axway/agents-mulesoft/pkg/common"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/sirupsen/logrus"
)

// SubStateManager handles the updates to the state of a subscription for a given schema/policy type.
type SubStateManager struct {
	SubSchema
	client anypoint.Client
}

// NewSubStateManager creates a new
func NewSubStateManager(client anypoint.Client, policy SubSchema) *SubStateManager {
	return &SubStateManager{
		SubSchema: policy,
		client:    client,
	}
}

func (ssm *SubStateManager) Subscribe(log logrus.FieldLogger, sub apic.Subscription) error {
	clientID, clientSecret, err := ssm.doSubscribe(log, sub)

	if err != nil {
		log.WithError(err).Error("Failed to subscribe")
		return sub.UpdateState(apic.SubscriptionFailedToSubscribe, err.Error())
	}

	return sub.UpdateStateWithProperties(apic.SubscriptionActive, "", map[string]interface{}{
		anypoint.ClientIDProp: clientID, anypoint.ClientSecretProp: clientSecret})
}

func (ssm *SubStateManager) Unsubscribe(log logrus.FieldLogger, sub apic.Subscription) error {
	log.Infof("Delete SLA Tier subscription for %s", ssm.Name())

	appName := sub.GetPropertyValue(anypoint.AppName)

	appID, err := cache.GetCache().Get(appName)
	if err != nil {
		return err
	}

	muleAppID, ok := appID.(int64)
	if !ok {
		return fmt.Errorf("Error while performing type assertion on %#v", appID)
	}

	err = ssm.client.DeleteClientApplication(muleAppID)
	if err != nil {
		log.WithError(err).Error("Failed to delete client application")
		return sub.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to delete client application with ID %v", appID))

	}
	return sub.UpdateState(apic.SubscriptionUnsubscribed, "")
}

func (ssm *SubStateManager) doSubscribe(log logrus.FieldLogger, sub apic.Subscription) (string, string, error) {
	// Create a new application and create a new contract
	apiID := sub.GetRemoteAPIAttributes()[common.AttrAPIID]
	stage := sub.GetRemoteAPIAttributes()[common.AttrProductVersion]
	tier := sub.GetPropertyValue(anypoint.TierLabel)

	application, err := ssm.createApp(apiID, sub)
	if err != nil {
		return "", "", err
	}

	log.WithField("Client application", application.Name).Debug("Created a client application on Mulesoft")

	muleAPI, err := getMuleAPI(apiID, stage)
	if err != nil {
		return "", "", err
	}

	tId := parseTierID(tier, log)

	// Need to fetch the exchange asset to get the version group
	exchangeAsset, err := ssm.client.GetExchangeAsset(muleAPI.GroupID, muleAPI.AssetID, muleAPI.AssetVersion)
	if err != nil {
		return "", "", err
	}

	_, err = ssm.createContract(apiID, application.ID, tId, muleAPI, exchangeAsset)
	if err != nil {
		return "", "", fmt.Errorf("Error while creating a contract %s", err)
	}

	log.WithField("Client application", application.Name).Info("Created a new contract")

	return application.ClientID, application.ClientSecret, nil
}

func (ssm *SubStateManager) createContract(apiID string, appId, tId int64, muleApi *anypoint.API, exchangeAsset *anypoint.ExchangeAsset) (*anypoint.Contract, error) {
	contract := &anypoint.Contract{
		APIID:           apiID,
		EnvironmentID:   muleApi.EnvironmentID,
		AcceptedTerms:   true,
		OrganizationID:  muleApi.OrganizationID,
		GroupID:         muleApi.GroupID,
		AssetID:         muleApi.AssetID,
		Version:         muleApi.AssetVersion,
		VersionGroup:    exchangeAsset.VersionGroup,
		RequestedTierID: tId,
	}

	return ssm.client.CreateContract(appId, contract)
}

func (ssm *SubStateManager) createApp(apiID string, sub apic.Subscription) (*anypoint.Application, error) {
	appName := sub.GetPropertyValue(anypoint.AppName)
	description := sub.GetPropertyValue(anypoint.Description)

	appl := &anypoint.AppRequestBody{
		Name:        appName,
		Description: description,
	}

	application, err := ssm.client.CreateClientApplication(apiID, appl)
	if err != nil {
		return nil, fmt.Errorf("Error creating client app: %s", err)
	}

	err = cache.GetCache().Set(appName, application.ID)
	if err != nil {
		return nil, err
	}
	return application, nil
}

func parseTierID(tierValue string, logger logrus.FieldLogger) int64 {
	tierID := strings.Split(tierValue, "-")[0]
	i, err := strconv.ParseInt(tierID, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func getMuleAPI(apiID, stage string) (*anypoint.API, error) {
	api, err := cache.GetCache().GetBySecondaryKey(common.FormatAPICacheKey(apiID, stage))
	if err != nil {
		return nil, err
	}

	muleApi, ok := api.(anypoint.API)
	if !ok {
		return nil, fmt.Errorf("Unable to perform type assertion on %#v", api)
	}
	return &muleApi, nil
}
