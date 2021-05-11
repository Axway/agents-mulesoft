package subscription

import (
	"fmt"
	"sync"

	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/sirupsen/logrus"
)

type SchemaHandler interface {
	GetSubscriptionSchemaName(pd config.PolicyDetail) string
	RegisterNewSchema(schemaConstructor SchemaConstructor, apc anypoint.Client)
}

// SchemaConstructor -
type SchemaConstructor func(client anypoint.Client) Contract

var constructors []SchemaConstructor

func Register(constructor SchemaConstructor) {
	constructors = append(constructors, constructor)
}

// ConsumerInstanceGetter gets a consumer instance by id.
type ConsumerInstanceGetter interface {
	GetConsumerInstanceByID(id string) (*v1alpha1.ConsumerInstance, error)
}

// SubscriptionPolicy the policy displayed to a user during the subscription process.
type SubscriptionPolicy interface {
	Schema() apic.SubscriptionSchema
	Name() string
	IsApplicable(policyDetail config.PolicyDetail) bool
}

type Contract interface {
	SubscriptionPolicy
	Subscribe(log logrus.FieldLogger, subs apic.Subscription) error
	Unsubscribe(log logrus.FieldLogger, subs apic.Subscription) error
}

// Manager handles the subscription aspects
type Manager struct {
	log      logrus.FieldLogger
	handlers map[string]Contract
	cig      ConsumerInstanceGetter
	dg       *duplicateGuard
}

type duplicateGuard struct {
	cache map[string]interface{}
	lock  *sync.Mutex
}

func New(log logrus.FieldLogger,
	cig ConsumerInstanceGetter,
	apc anypoint.Client,
) *Manager {
	handlers := make(map[string]Contract, len(constructors))

	for _, c := range constructors {
		h := c(apc)
		handlers[h.Name()] = h
	}

	return &Manager{
		log:      log,
		handlers: handlers,
		cig:      cig,
		dg: &duplicateGuard{
			cache: map[string]interface{}{},
			lock:  &sync.Mutex{},
		},
	}
}

func (sm *Manager) RegisterNewSchema(
	schemaConstructor SchemaConstructor,
	apc anypoint.Client,
) {
	h := schemaConstructor(apc)
	sm.handlers[h.Name()] = h
}

func (sm *Manager) Schemas() []apic.SubscriptionSchema {
	res := make([]apic.SubscriptionSchema, 0, len(sm.handlers))
	for _, h := range sm.handlers {
		res = append(res, h.Schema())
	}

	return res
}

// markActive returns
func (dg *duplicateGuard) markActive(id string) bool {
	dg.lock.Lock()
	defer dg.lock.Unlock()
	if _, ok := dg.cache[id]; ok {
		return true
	}

	dg.cache[id] = true

	return false
}

// markInactive returns
func (dg *duplicateGuard) markInactive(id string) {
	dg.lock.Lock()
	defer dg.lock.Unlock()

	delete(dg.cache, id)
}

func (sm *Manager) ValidateSubscription(subscription apic.Subscription) bool {
	if sm.dg.markActive(subscription.GetID()) {
		sm.log.Info("duplicate subscription event; already handling subscription")
		return false
	}
	return true
}

// GetSubscriptionSchemaName returns the appropriate subscription schema name given a policy
func (sm *Manager) GetSubscriptionSchemaName(pd config.PolicyDetail) string {
	for _, h := range sm.handlers {
		if h.IsApplicable(pd) {
			return h.Name()
		}
	}
	return ""
}

// ProcessUnsubscribe moves a subscription from Approved to Active.
func (sm *Manager) ProcessSubscribe(subscription apic.Subscription) {
	log := sm.log.WithFields(logFields(subscription))
	if err := sm.processForState(subscription, log, apic.SubscriptionApproved); err != nil {
		log.WithError(err).Error("failed to update subscription state")
	}
}

// processForState processes a subscription state change based on the current state.
func (sm *Manager) processForState(subscription apic.Subscription, log logrus.FieldLogger, state apic.SubscriptionState) error {
	subID := subscription.GetID()
	consumerInstanceID := subscription.GetApicID()

	defer sm.dg.markInactive(subID)

	ci, err := sm.cig.GetConsumerInstanceByID(consumerInstanceID)
	if err != nil {
		return fmt.Errorf("failed to fetch consumer instance")
	}

	log.Info("processing subscription state change")

	def := ci.Spec.Subscription.SubscriptionDefinition
	if h, ok := sm.handlers[def]; ok {
		switch state {
		case apic.SubscriptionApproved:
			return h.Subscribe(log.WithField("handler", h.Name()), subscription)
		case apic.SubscriptionUnsubscribeInitiated:
			return h.Unsubscribe(log.WithField("handler", h.Name()), subscription)
		}
	}

	log.Infof("No known handler for type: %s", def)
	return nil
}

// ProcessUnsubscribe moves a subscription from Unsubscribe Initiated to Unsubscribed.
func (sm *Manager) ProcessUnsubscribe(subscription apic.Subscription) {
	log := sm.log.WithFields(logFields(subscription))
	if err := sm.processForState(subscription, log, apic.SubscriptionUnsubscribeInitiated); err != nil {
		log.WithError(err).Error("failed to update subscription state")
	}
}

func logFields(sub apic.Subscription) logrus.Fields {
	return logrus.Fields{
		"subscriptionName":   sub.GetName(),
		"subscriptionID":     sub.GetID(),
		"catalogItemID":      sub.GetCatalogItemID(),
		"remoteID":           sub.GetRemoteAPIID(),
		"consumerInstanceID": sub.GetApicID(),
		"currentState":       sub.GetState(),
	}
}
