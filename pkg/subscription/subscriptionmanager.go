package subscription

import (
	"sync"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/sirupsen/logrus"
)

type SchemaHandler interface {
	GetSubscriptionSchemaName(pd PolicyDetail) string
	RegisterNewSchema(schemaConstructor SchemaConstructor, apc anypoint.Client)
}

// SchemaConstructor -
type SchemaConstructor func(client anypoint.Client) Handler

var constructors []SchemaConstructor

func Register(constructor SchemaConstructor) {
	constructors = append(constructors, constructor)
}

// ConsumerInstanceGetter gets a consumer instance by id.
type ConsumerInstanceGetter interface {
	GetConsumerInstanceByID(id string) (*v1alpha1.ConsumerInstance, error)
}

// SubscriptionsGetter gets the all the subscriptions in any of the states for the catalog item with id
type SubscriptionsGetter interface {
	GetSubscriptionsForCatalogItem(states []string, id string) ([]apic.CentralSubscription, error)
}

type Handler interface {
	Schema() apic.SubscriptionSchema
	Name() string
	IsApplicable(policyDetail PolicyDetail) bool
	Subscribe(log logrus.FieldLogger, subs apic.Subscription) error
	Unsubscribe(log logrus.FieldLogger, subs apic.Subscription)
}

type PolicyDetail struct {
	Policy     string
	IsSlaBased bool
	APIId      string
}

// Manager handles the subscription aspects
type Manager struct {
	log      logrus.FieldLogger
	handlers map[string]Handler
	cig      ConsumerInstanceGetter
	sg       SubscriptionsGetter
	dg       *duplicateGuard
}

type duplicateGuard struct {
	cache map[string]interface{}
	lock  *sync.Mutex
}

func New(log logrus.FieldLogger,
	cig ConsumerInstanceGetter,
	sg SubscriptionsGetter,
	apc anypoint.Client,
) *Manager {
	handlers := make(map[string]Handler, len(constructors))

	for _, c := range constructors {
		h := c(apc)
		handlers[h.Name()] = h
	}

	return &Manager{
		log:      log,
		handlers: handlers,
		cig:      cig,
		sg:       sg,
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

func (sm *Manager) checkSubscriptionState(subscriptionID, catalogItemID, subscriptionState string) (bool, error) {

	subs, err := sm.sg.GetSubscriptionsForCatalogItem([]string{subscriptionState}, catalogItemID)
	if err != nil {
		return false, err
	}

	for _, sub := range subs {
		if sub.GetID() == subscriptionID {
			return true, nil
		}
	}

	return false, nil
}

// GetSubscriptionSchemaName returns the appropriate subscription schema name given a policy
func (sm *Manager) GetSubscriptionSchemaName(pd PolicyDetail) string {
	for _, h := range sm.handlers {
		if h.IsApplicable(pd) {
			return h.Name()
		}
	}
	return ""
}

func (sm *Manager) ProcessSubscribe(subscription apic.Subscription) {
	defer sm.dg.markInactive(subscription.GetID())

	log := sm.log.
		WithField("subscriptionID", subscription.GetID()).
		WithField("catalogItemID", subscription.GetCatalogItemID()).
		WithField("remoteID", subscription.GetRemoteAPIID()).
		WithField("consumerInstanceID", subscription.GetApicID())

	isApproved, err := sm.checkSubscriptionState(subscription.GetID(), subscription.GetCatalogItemID(), string(apic.SubscriptionApproved))
	if err != nil {
		log.WithError(err).Error("Failed to verify subscription state")
		return
	}

	if !isApproved {
		log.Info("Subscription not in approved state. Nothing to do")
		return
	}

	log.Info("Processing subscription")

	ci, err := sm.cig.GetConsumerInstanceByID(subscription.GetApicID())
	if err != nil {
		log.WithError(err).Error("Failed to fetch consumer instance")
		return
	}

	if h, ok := sm.handlers[ci.Spec.Subscription.SubscriptionDefinition]; ok {
		h.Subscribe(log.WithField("handler", h.Name()), subscription)
	} else {
		log.Info("No known handler for type: ", ci.Spec.Subscription.SubscriptionDefinition)
	}
}

func (sm *Manager) ProcessUnsubscribe(subscription apic.Subscription) {
	defer sm.dg.markInactive(subscription.GetID())

	log := sm.log.
		WithField("subscriptionID", subscription.GetID()).
		WithField("catalogItemID", subscription.GetCatalogItemID()).
		WithField("remoteID", subscription.GetRemoteAPIID()).
		WithField("consumerInstanceID", subscription.GetApicID())

	isUnsubscribeInitiated, err := sm.checkSubscriptionState(subscription.GetID(), subscription.GetCatalogItemID(), string(apic.SubscriptionUnsubscribeInitiated))
	if err != nil {
		log.WithError(err).Error("Failed to verify subscription state")
		return
	}

	if !isUnsubscribeInitiated {
		log.Info("Subscription not in unsubscribe initiated state. Nothing to do")
		return
	}

	log.Info("Removing subscription")

	ci, err := sm.cig.GetConsumerInstanceByID(subscription.GetApicID())
	if err != nil {
		log.WithError(err).Error("Failed to fetch consumer instance")
		return
	}

	if h, ok := sm.handlers[ci.Spec.Subscription.SubscriptionDefinition]; ok {
		h.Unsubscribe(log.WithField("handler", h.Name()), subscription)
	} else {
		log.Info("No known handler for type: ", ci.Spec.Subscription.SubscriptionDefinition)
	}
}
