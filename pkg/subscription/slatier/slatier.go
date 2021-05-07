package slatier

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Axway/agents-mulesoft/pkg/subscription"

	"github.com/Axway/agents-mulesoft/pkg/subscription/clientid"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/sirupsen/logrus"
)

type slaTier struct {
	name   string
	schema apic.SubscriptionSchema
	apc    *anypoint.AnypointClient
}

const (
	// TODO extract the first two to a file that defines all constants
	AppName   = "appName"
	Desc      = "description"
	TierLabel = "SLA Tier"
)

func New(name string, apc *anypoint.AnypointClient, schema apic.SubscriptionSchema) *slaTier {
	return &slaTier{
		apc:    apc,
		name:   name,
		schema: schema,
	}
}

func (s *slaTier) Name() string {
	return s.name
}

func (s *slaTier) Schema() apic.SubscriptionSchema {
	return s.schema
}

func (s *slaTier) IsApplicable(pd subscription.PolicyDetail) bool {
	if pd.IsSlaBased {
		return pd.APIId == s.name && pd.Policy == apic.Apikey
	}
	return false
}

func (s *slaTier) Subscribe(log logrus.FieldLogger, subs apic.Subscription) error {
	clientID, clientSecret, err := s.doSubscribe(log, subs)

	if err != nil {
		log.WithError(err).Error("Failed to subscribe")
		return subs.UpdateState(apic.SubscriptionFailedToSubscribe, err.Error())
	}

	return subs.UpdateStateWithProperties(apic.SubscriptionActive, "", map[string]interface{}{clientid.ClientIDProp: clientID, clientid.ClientSecretProp: clientSecret})
}

func (s *slaTier) doSubscribe(log logrus.FieldLogger, subs apic.Subscription) (string, string, error) {
	// Create a new application and create a new contract
	apiID := subs.GetRemoteAPIID()
	app := subs.GetPropertyValue(AppName)
	d := subs.GetPropertyValue(Desc)

	appl := &anypoint.AppRequestBody{
		Name:        app,
		Description: d,
	}

	application, err := s.apc.CreateClientApplication(apiID, appl)
	if err != nil {
		return "", "", fmt.Errorf("Error creating client app: %s", err)
	}

	log.WithField("Client application", application.Name).Debug("Created a client application on Mulesoft")

	api, err := cache.GetCache().GetBySecondaryKey(subs.GetRemoteAPIID())
	if err != nil {
		return "", "", err
	}

	muleApi := api.(anypoint.API)

	tier := subs.GetPropertyValue(TierLabel)

	tierID := strings.Split(tier, "-")[0]
	tId, err := strconv.ParseInt(tierID, 10, 64)
	if err != nil {
		return "", "", err
	}

	// Need to fetch the exchange asset to get the version group
	exchangeAsset, err := s.apc.GetExchangeAsset(muleApi.GroupID, muleApi.AssetID, muleApi.AssetVersion)
	if err != nil {
		return "", "", err
	}

	cnt := &anypoint.Contract{
		APIID:           apiID,
		EnvironmentId:   muleApi.EnvironmentID,
		AcceptedTerms:   true,
		OrganizationId:  muleApi.OrganizationID,
		GroupId:         muleApi.GroupID,
		AssetId:         muleApi.AssetID,
		Version:         muleApi.AssetVersion,
		VersionGroup:    exchangeAsset.VersionGroup,
		RequestedTierID: tId,
	}

	_, err = s.apc.CreateContract(application.Id, cnt)
	if err != nil {
		return "", "", fmt.Errorf("Error while creating a contract")
	}
	log.WithField("Client application", application.Name).Debug("Created a new contract")

	return application.ClientId, application.ClientSecret, nil
}

func (s *slaTier) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	log.Info("Delete SLA Tier subscription for ", s.name)

	app := subs.GetPropertyValue(AppName)

	err := s.apc.DeleteClientApplication(app)
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
