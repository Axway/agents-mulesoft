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
	// Create a new application and create a new contract
	apiInstanceId := subs.GetRemoteAPIID()

	app := subs.GetPropertyValue(appName)

	d := subs.GetPropertyValue(desc)

	// Use this API Instance ID to make a POST call to create a new client application
	appl := &anypoint.AppRequestBody{
		Name:        app,
		Description: d,
	}

	application, err := c.apc.CreateClientApplication(apiInstanceId, appl)
	if err != nil {
		return "", "", fmt.Errorf("Error creating client app", err)
	}

	api, err := cache.GetCache().GetBySecondaryKey(subs.GetRemoteAPIID())
	if err != nil {
		return "", "", err
	}

	muleApi := api.(anypoint.API)

	// And make another POST call to create a new contract
	cnt := &anypoint.Contract{
		APIID:          apiInstanceId,
		EnvironmentId:  muleApi.EnvironmentID,
		AcceptedTerms:  true,
		OrganizationId: muleApi.OrganizationID,
		GroupId:        muleApi.GroupID,
		AssetId:        muleApi.AssetID,
		Version:        muleApi.AssetVersion,
		VersionGroup:   muleApi.AssetVersion,
	}

	_, err = c.apc.CreateContract(application.Id, cnt)
	if err != nil {
		return "", "", fmt.Errorf("Error while creating a contract")
	}

	return application.ClientId, application.ClientSecret, nil
}

func (c *clientId) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	panic("implement me")
}
