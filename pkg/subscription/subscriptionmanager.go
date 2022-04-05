package subscription

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/Axway/agents-mulesoft/pkg/common"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/sirupsen/logrus"
)

// SchemaStore interface for saving and retrieving subscription schemas
type SchemaStore interface {
	// GetSubscriptionSchemaName returns the name of a schema for the provided policy if the schema is found
	GetSubscriptionSchemaName(pd common.PolicyDetail) string
	// RegisterNewSchema saves a schema to the store
	RegisterNewSchema(schema SubSchema)
}

// SubSchema the subscription schema to represent the policy in the Unified Catalog.
type SubSchema interface {
	// Schema is the Unified Catalog Subscription Schema
	Schema() apic.SubscriptionSchema
	// Name of the Subscription Schema based on the policy type, such as client-id
	Name() string
	// IsApplicable returns true if the provided policy matches the schema
	IsApplicable(policy common.PolicyDetail) bool
}

// Manager stores subscription schemas and handles subscription state changes
type Manager struct {
	log              logrus.FieldLogger
	schemas          map[string]SubSchema
	dg               *duplicateGuard
	muleSubscription MuleSubscriptionClient
}

type duplicateGuard struct {
	cache map[string]interface{}
	lock  *sync.Mutex
}

// NewManager creates a SubscriptionManager
func NewManager(
	log logrus.FieldLogger, muleSubscription MuleSubscriptionClient, schemas ...SubSchema,
) *Manager {
	handlers := make(map[string]SubSchema, len(schemas))

	manager := &Manager{
		log:     log,
		schemas: handlers,
		dg: &duplicateGuard{
			cache: map[string]interface{}{},
			lock:  &sync.Mutex{},
		},
		muleSubscription: muleSubscription,
	}

	for _, schema := range schemas {
		manager.RegisterNewSchema(schema)
	}

	return manager
}

// RegisterNewSchema registers a schema to represent a Mulesoft policy that can be subscribed to in the Unified Catalog.
func (sm *Manager) RegisterNewSchema(schema SubSchema) {
	sm.dg.lock.Lock()
	defer sm.dg.lock.Unlock()
	sm.schemas[schema.Name()] = schema
}

// markActive returns true when a subscription is being processed for a given id.
func (dg *duplicateGuard) markActive(id string) bool {
	dg.lock.Lock()
	defer dg.lock.Unlock()
	if _, ok := dg.cache[id]; ok {
		return true
	}

	dg.cache[id] = true

	return false
}

// markInactive is called after processing a subscription
func (dg *duplicateGuard) markInactive(id string) {
	dg.lock.Lock()
	defer dg.lock.Unlock()

	delete(dg.cache, id)
}

// ValidateSubscription checks if a subscription should  be processed or not. If a subscription is already marked as active, then false is returned
func (sm *Manager) ValidateSubscription(subscription apic.Subscription) bool {
	if sm.dg.markActive(subscription.GetID()) {
		sm.log.Info("duplicate subscription event; already handling subscription")
		return false
	}
	return true
}

// GetSubscriptionSchemaName returns the appropriate subscription schema name given a policy
func (sm *Manager) GetSubscriptionSchemaName(pd common.PolicyDetail) string {
	for _, h := range sm.schemas {
		if h.IsApplicable(pd) {
			return h.Name()
		}
	}
	return ""
}

// ProcessSubscribe moves a subscription from Approved to Active.
func (sm *Manager) ProcessSubscribe(subscription apic.Subscription) {
	sm.processForState(subscription, apic.SubscriptionApproved)
}

// ProcessUnsubscribe moves a subscription from Unsubscribe Initiated to Unsubscribed.
func (sm *Manager) ProcessUnsubscribe(subscription apic.Subscription) {
	sm.processForState(subscription, apic.SubscriptionUnsubscribeInitiated)
}

// processForState processes a subscription state change based on the current state.
func (sm *Manager) processForState(subscription apic.Subscription, state apic.SubscriptionState) error {
	subID := subscription.GetID()
	sm.dg.markActive(subID)
	defer sm.dg.markInactive(subID)

	switch state {
	case apic.SubscriptionApproved:
		return sm.Subscribe(subscription)
	case apic.SubscriptionUnsubscribeInitiated:
		return sm.Unsubscribe(subscription)
	default:
		return nil
	}
}

// Subscribe creates an application and a contract in Mulesoft, and updates the subscription state in central.
func (sm *Manager) Subscribe(sub apic.Subscription) error {
	log := sm.log.WithFields(logFields(sub))

	apiID := sub.GetRemoteAPIAttributes()[common.AttrAPIID]
	stage := sub.GetRemoteAPIAttributes()[common.AttrProductVersion]
	tier := sub.GetPropertyValue(common.TierLabel)
	appName := sub.GetPropertyValue(common.AppName)
	description := sub.GetPropertyValue(common.Description)

	app, err := sm.muleSubscription.CreateApp(appName, apiID, description)
	if err != nil {
		log.WithError(err).Error("failed to subscribe")
		return sub.UpdateState(apic.SubscriptionFailedToSubscribe, err.Error())
	}

	_, err = sm.muleSubscription.CreateContract(apiID, stage, tier, app.ID)
	if err != nil {
		log.WithError(err).Error("failed to subscribe")
		return sub.UpdateState(apic.SubscriptionFailedToSubscribe, err.Error())
	}

	props := map[string]interface{}{
		common.AppID:        app.ID,
		common.ClientID:     app.ClientID,
		common.ClientSecret: app.ClientSecret,
	}

	if err := sub.UpdateStateWithProperties(apic.SubscriptionActive, "", props); err != nil {
		log.WithError(err).Error("failed to subscribe")
		return err
	}

	return nil
}

// Unsubscribe deletes an application in Mulesoft
func (sm *Manager) Unsubscribe(sub apic.Subscription) error {
	log := sm.log.WithFields(logFields(sub))
	appName := sub.GetPropertyValue(common.AppName)
	appID := sub.GetPropertyValue(common.AppID)
	appID64, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return sub.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("failed to unsubscribe: %s", err))
	}

	if err := sm.muleSubscription.DeleteApp(appID64); err != nil {
		log.WithError(err).Error("failed to delete client application")
		return sub.UpdateState(apic.SubscriptionFailedToSubscribe, fmt.Sprintf("Failed to delete client application %s", appName))
	}

	if err := sub.UpdateState(apic.SubscriptionUnsubscribed, ""); err != nil {
		log.WithError(err).Error("failed to unsubscribe")
		return err
	}

	return nil
}

func logFields(sub apic.Subscription) logrus.Fields {
	return logrus.Fields{
		"subscriptionName": sub.GetName(),
		"catalogItemID":    sub.GetCatalogItemID(),
	}
}

func parseTierID(tierValue string) int64 {
	tierID := strings.Split(tierValue, "-")[0]
	i, err := strconv.ParseInt(tierID, 10, 64)
	if err != nil {
		return 0
	}
	return i
}
