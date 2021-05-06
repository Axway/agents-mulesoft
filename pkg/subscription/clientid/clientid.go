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
	name             = "client-id-enforcement"
	appName          = "appName"
	desc             = "description"
	ClientIDProp     = "client_id"
	ClientSecretProp = "client_secret"
)

func init() {
	fmt.Print("Initing clientid.go")
	subscription.Register(func(apc *anypoint.AnypointClient) subscription.Handler {
		return &clientId{apc: apc}
	})
}

func (c *clientId) IsApplicable(pd subscription.PolicyDetail) bool {
	return pd.Policy == apic.Apikey && pd.IsSlaBased == false
}

func (c *clientId) Schema() apic.SubscriptionSchema {
	schema := apic.NewSubscriptionSchema(name)

	schema.AddProperty(appName,
		"string",
		"Name of the new app",
		"",
		true,
		nil)

	schema.AddProperty(desc,
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

	return subs.UpdateStateWithProperties(apic.SubscriptionActive, "", map[string]interface{}{ClientIDProp: clientID, ClientSecretProp: clientSecret})
}

func (c *clientId) doSubscribe(log logrus.FieldLogger, subs apic.Subscription) (string, string, error) {
	// Create a new client application on Mulesoft
	apiId := subs.GetRemoteAPIID()

	app := subs.GetPropertyValue(appName)

	d := subs.GetPropertyValue(desc)

	appl := &anypoint.AppRequestBody{
		Name:        app,
		Description: d,
	}

	application, err := c.apc.CreateClientApplication(apiId, appl)
	if err != nil {
		return "", "", fmt.Errorf("Error creating client application", err)
	}

	log.WithField("Client application", application.Name).Debug("Created a client application on Mulesoft")

	api, err := cache.GetCache().GetBySecondaryKey(subs.GetRemoteAPIID())
	if err != nil {
		return "", "", err
	}

	muleApi := api.(anypoint.API)

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
		return "", "", fmt.Errorf("Error while creating a contract", err)
	}

	log.WithField("Client application", application.Name).Debug("Created a new contract")

	return application.ClientId, application.ClientSecret, nil
}

func (c *clientId) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	log.Info("Delete subscription for ", name)

	app := subs.GetPropertyValue(appName)

	err := c.apc.DeleteClientApplication(app)
	if err != nil {
		log.WithError(err).Error("Failed to delete client application")
		subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to delete client application %s", app))
		return
	}
	err = subs.UpdateState(apic.SubscriptionUnsubscribed, "")
	if err != nil {
		log.WithError(err).Error("failed to update subscription state")
	}
}
