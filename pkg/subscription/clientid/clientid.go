package clientid

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/cache"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/subscription"
	"github.com/sirupsen/logrus"
)

type clientId struct {
	apc *anypoint.AnypointClient
}

const (
	name = "client-id-enforcement"
)

func init() {
	subscription.Register(func(apc *anypoint.AnypointClient) subscription.Handler {
		return &clientId{apc: apc}
	})
}

func (c *clientId) IsApplicable(pd subscription.PolicyDetail) bool {
	return pd.Policy == apic.Apikey && pd.IsSlaBased == false
}

func (c *clientId) Schema() apic.SubscriptionSchema {
	schema := apic.NewSubscriptionSchema(name)

	schema.AddProperty(anypoint.AppName,
		"string",
		"Name of the new app",
		"",
		true,
		nil)

	schema.AddProperty(anypoint.Description,
		"string",
		"Description",
		"",
		false,
		nil)

	return schema
}

func (c *clientId) Name() string {
	return name
}

func (c *clientId) Subscribe(log logrus.FieldLogger, subs apic.Subscription) error {
	clientID, clientSecret, err := c.doSubscribe(log, subs)

	if err != nil {
		log.WithError(err).Error("Failed to subscribe")
		return subs.UpdateState(apic.SubscriptionFailedToSubscribe, err.Error())
	}

	return subs.UpdateStateWithProperties(apic.SubscriptionActive, "", map[string]interface{}{
		anypoint.ClientIDProp: clientID, anypoint.ClientSecretProp: clientSecret})
}

func (c *clientId) doSubscribe(log logrus.FieldLogger, subs apic.Subscription) (string, string, error) {
	// Create a new client application on Mulesoft
	apiId := subs.GetRemoteAPIID()

	appName := subs.GetPropertyValue(anypoint.AppName)

	d := subs.GetPropertyValue(anypoint.Description)

	appl := &anypoint.AppRequestBody{
		Name:        appName,
		Description: d,
	}

	application, err := c.apc.CreateClientApplication(apiId, appl)
	if err != nil {
		return "", "", fmt.Errorf("Error creating client application %s", err)
	}

	// Add App name and ID to cache, need it later during unsubscribing
	err = cache.GetCache().Set(appName, application.Id)
	if err != nil {
		return "", "", err
	}

	log.WithField("Client application", application.Name).Debug("Created a client application on Mulesoft")

	api, err := cache.GetCache().GetBySecondaryKey(subs.GetRemoteAPIID())
	if err != nil {
		return "", "", err
	}

	muleApi, ok := api.(anypoint.API)
	if !ok {
		return "", "", fmt.Errorf("Unable to perform type assertion on %#v", api)
	}

	// Need to fetch the exchange asset to get the version group
	exchangeAsset, err := c.apc.GetExchangeAsset(muleApi.GroupID, muleApi.AssetID, muleApi.AssetVersion)
	if err != nil {
		return "", "", err
	}

	// Create a new contract for the created client application
	cnt := &anypoint.Contract{
		APIID:          apiId,
		EnvironmentId:  muleApi.EnvironmentID,
		AcceptedTerms:  true,
		OrganizationId: muleApi.OrganizationID,
		GroupId:        muleApi.GroupID,
		AssetId:        muleApi.AssetID,
		Version:        muleApi.AssetVersion,
		VersionGroup:   exchangeAsset.VersionGroup,
	}

	_, err = c.apc.CreateContract(application.Id, cnt)
	if err != nil {
		return "", "", fmt.Errorf("Error while creating a contract %s", err)
	}

	log.WithField("Client application", application.Name).Debug("Created a new contract")

	return application.ClientId, application.ClientSecret, nil
}

func (c *clientId) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	log.Info("Delete subscription for ", name)

	appName := subs.GetPropertyValue(anypoint.AppName)

	appID, err := cache.GetCache().Get(appName)
	if err != nil {
		log.WithError(err).Error("Error while retrieving item with key %s from cache", appName)
		return
	}

	muleAppID, ok := appID.(int64)
	if !ok {
		log.WithError(err).Error("Error while performing type assertion on %#v", appID)
	}

	err = c.apc.DeleteClientApplication(muleAppID)
	if err != nil {
		log.WithError(err).Error("Failed to delete client application")
		err1 := subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to delete client application with ID %v", appID))
		if err1 != nil {
			log.WithError(err1).Error("Error updating the subscription state")
		}
		return
	}
	err = subs.UpdateState(apic.SubscriptionUnsubscribed, "")
	if err != nil {
		log.WithError(err).Error("failed to update subscription state")
	}
}
