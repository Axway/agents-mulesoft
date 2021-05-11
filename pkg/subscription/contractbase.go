package subscription

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/sirupsen/logrus"
)

type ContractBase struct {
	SubscriptionPolicy
	client anypoint.Client
}

func NewContractBase(client anypoint.Client, policy SubscriptionPolicy) *ContractBase {
	return &ContractBase{
		SubscriptionPolicy: policy,
		client:             client,
	}
}

func (cb *ContractBase) Subscribe(log logrus.FieldLogger, subs apic.Subscription) error {
	clientID, clientSecret, err := cb.doSubscribe(log, subs)

	if err != nil {
		log.WithError(err).Error("Failed to subscribe")
		return subs.UpdateState(apic.SubscriptionFailedToSubscribe, err.Error())
	}

	return subs.UpdateStateWithProperties(apic.SubscriptionActive, "", map[string]interface{}{
		anypoint.ClientIDProp: clientID, anypoint.ClientSecretProp: clientSecret})
}

func (cb *ContractBase) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) error {
	log.Info("Delete SLA Tier subscription for ", cb.Name())

	appName := subs.GetPropertyValue(anypoint.AppName)

	appID, err := cache.GetCache().Get(appName)
	if err != nil {
		return err
	}

	muleAppID, ok := appID.(int64)
	if !ok {
		return fmt.Errorf("Error while performing type assertion on %#v", appID)
	}

	err = cb.client.DeleteClientApplication(muleAppID)
	if err != nil {
		log.WithError(err).Error("Failed to delete client application")
		return subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to delete client application with ID %v", appID))

	}
	return subs.UpdateState(apic.SubscriptionUnsubscribed, "")
}

func (cb *ContractBase) doSubscribe(log logrus.FieldLogger, subs apic.Subscription) (string, string, error) {
	// Create a new application and create a new contract
	apiID := subs.GetRemoteAPIID()
	tier := subs.GetPropertyValue(anypoint.TierLabel)

	application, err := cb.createApp(apiID, subs)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Mulesoft application: %s", err)
	}

	log.WithField("Client application", application.Name).Debug("Created a client application on Mulesoft")

	muleAPI, err := getMuleAPI(apiID)
	if err != nil {
		return "", "", err
	}

	tId, err := parseTierID(tier)
	if err != nil {
		log.Debug("unable to parse tier ID")
		tId = 0
	}

	// Need to fetch the exchange asset to get the version group
	exchangeAsset, err := cb.client.GetExchangeAsset(muleAPI.GroupID, muleAPI.AssetID, muleAPI.AssetVersion)
	if err != nil {
		return "", "", err
	}

	_, err = cb.createContract(apiID, application.ID, tId, muleAPI, exchangeAsset)
	if err != nil {
		return "", "", fmt.Errorf("Error while creating a contract %s", err)
	}

	log.WithField("Client application", application.Name).Info("Created a new contract")

	return application.ClientID, application.ClientSecret, nil
}

func (cb *ContractBase) createContract(apiID string, appId, tId int64, muleApi *anypoint.API, exchangeAsset *anypoint.ExchangeAsset) (*anypoint.Contract, error) {
	cnt := &anypoint.Contract{
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

	return cb.client.CreateContract(appId, cnt)
}

func (cb *ContractBase) createApp(apiID string, subs apic.Subscription) (*anypoint.Application, error) {
	appName := subs.GetPropertyValue(anypoint.AppName)
	description := subs.GetPropertyValue(anypoint.Description)

	appl := &anypoint.AppRequestBody{
		Name:        appName,
		Description: description,
	}

	application, err := cb.client.CreateClientApplication(apiID, appl)
	if err != nil {
		return nil, fmt.Errorf("Error creating client app: %s", err)
	}

	err = cache.GetCache().Set(appName, application.ID)
	if err != nil {
		return nil, err
	}
	return application, nil
}

func parseTierID(tierValue string) (int64, error) {
	tierID := strings.Split(tierValue, "-")[0]
	return strconv.ParseInt(tierID, 10, 64)
}

func getMuleAPI(apiID string) (*anypoint.API, error) {
	api, err := cache.GetCache().GetBySecondaryKey(apiID)
	if err != nil {
		return nil, err
	}

	muleApi, ok := api.(anypoint.API)
	if !ok {
		return nil, fmt.Errorf("Unable to perform type assertion on %#v", api)
	}
	return &muleApi, nil
}
