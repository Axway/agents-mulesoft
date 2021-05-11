package mocks

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/stretchr/testify/mock"
)

type MockCentralClient struct {
	mock.Mock
}

func (m *MockCentralClient) SetTokenGetter(auth.PlatformTokenGetter) {}
func (m *MockCentralClient) PublishService(apic.ServiceBody) (*v1alpha1.APIService, error) {
	return nil, nil
}
func (m *MockCentralClient) RegisterSubscriptionWebhook() error { return nil }
func (m *MockCentralClient) RegisterSubscriptionSchema(apic.SubscriptionSchema, bool) error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockCentralClient) UpdateSubscriptionSchema(apic.SubscriptionSchema) error {
	return nil
}
func (m *MockCentralClient) GetSubscriptionManager() apic.SubscriptionManager {
	return &apic.MockSubscriptionManager{}
}
func (m *MockCentralClient) GetCatalogItemIDForConsumerInstance(string) (string, error) {
	return "", nil
}
func (m *MockCentralClient) DeleteConsumerInstance(string) error { return nil }
func (m *MockCentralClient) UpdateConsumerInstanceSubscriptionDefinition(string, string) error {
	return nil
}
func (m *MockCentralClient) GetConsumerInstanceByID(string) (*v1alpha1.ConsumerInstance, error) {
	return nil, nil
}
func (m *MockCentralClient) GetUserEmailAddress(string) (string, error) { return "", nil }
func (m *MockCentralClient) GetSubscriptionsForCatalogItem([]string, string) ([]apic.CentralSubscription, error) {
	return nil, nil
}
func (m *MockCentralClient) GetSubscriptionDefinitionPropertiesForCatalogItem(string, string) (apic.SubscriptionSchema, error) {
	return nil, nil
}
func (m *MockCentralClient) Healthcheck(string) *hc.Status {
	return &hc.Status{Result: hc.OK}
}

// UpdateSubscriptionDefinitionPropertiesForCatalogItem -
func (m *MockCentralClient) UpdateSubscriptionDefinitionPropertiesForCatalogItem(string, string, apic.SubscriptionSchema) error {
	return nil
}

func (m *MockCentralClient) GetCatalogItemName(string) (string, error) { return "", nil }
func (m *MockCentralClient) ExecuteAPI(string, string, map[string]string, []byte) ([]byte, error) {
	return nil, nil
}
func (m *MockCentralClient) OnConfigChange(config.CentralConfig) {}

func (m *MockCentralClient) DeleteServiceByAPIID(string) error {
	return nil
}

func (m *MockCentralClient) GetConsumerInstancesByExternalAPIID(string) ([]*v1alpha1.ConsumerInstance, error) {
	return nil, nil
}
