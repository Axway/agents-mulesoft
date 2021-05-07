package slatier

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Axway/agents-mulesoft/pkg/subscription"

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

	return subs.UpdateStateWithProperties(apic.SubscriptionActive, "", map[string]interface{}{
		anypoint.ClientIDProp: clientID, anypoint.ClientSecretProp: clientSecret})
}

func (s *slaTier) doSubscribe(log logrus.FieldLogger, subs apic.Subscription) (string, string, error) {
	// Create a new application and create a new contract
	apiID := subs.GetRemoteAPIID()
	appName := subs.GetPropertyValue(anypoint.AppName)
	d := subs.GetPropertyValue(anypoint.Description)

	appl := &anypoint.AppRequestBody{
		Name:        appName,
		Description: d,
	}

	application, err := s.apc.CreateClientApplication(apiID, appl)
	if err != nil {
		return "", "", fmt.Errorf("Error creating client app: %s", err.Error())
	}

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

	tier := subs.GetPropertyValue(anypoint.TierLabel)

	tId, err := parseTierID(tier)
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
		return "", "", fmt.Errorf("Error while creating a contract %s", err.Error())
	}
	log.WithField("Client application", application.Name).Debug("Created a new contract")

	return application.ClientId, application.ClientSecret, nil
}

func parseTierID(tierValue string) (int64, error) {
	tierID := strings.Split(tierValue, "-")[0]
	return strconv.ParseInt(tierID, 10, 64)
}

func (s *slaTier) Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) {
	log.Info("Delete SLA Tier subscription for ", s.name)

	appName := subs.GetPropertyValue(anypoint.AppName)

	appID, err := cache.GetCache().Get(appName)
	if err != nil {
		log.WithError(err).Error("Error while retrieving item with key %s from cache", appName)
		return
	}

	muleAppID, ok := appID.(int64)
	if !ok {
		log.WithError(err).Error("Error while performing type assertion on %#v", appID)
		return
	}

	err = s.apc.DeleteClientApplication(muleAppID)
	if err != nil {
		log.WithError(err).Error("Failed to delete client application")
		err1 := subs.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to delete client application with ID %v", appID))
		if err1 != nil {
			log.WithError(err1).Error("Failed to update the subscription state")
		}
		return
	}
	err = subs.UpdateState(apic.SubscriptionUnsubscribed, "")
	if err != nil {
		log.WithError(err).Error("failed to update subscription state")
	}
}
