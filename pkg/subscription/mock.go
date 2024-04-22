package subscription

import (
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
)

type MockMuleSubscriptionClient struct {
	app       *anypoint.Application
	newApp    *anypoint.Application
	err       error
	rotateErr error
	contract  *anypoint.Contract
}

func (m *MockMuleSubscriptionClient) CreateApp(appName, apiID, description string) (*anypoint.Application, error) {
	return m.app, m.err
}

func (m *MockMuleSubscriptionClient) CreateContract(_, _, _ string) (*anypoint.Contract, error) {
	return m.contract, m.err
}

func (m *MockMuleSubscriptionClient) DeleteApp(appName string) error {
	return m.err
}

func (m *MockMuleSubscriptionClient) DeleteContract(apiID, contractID string) error {
	return m.err
}

func (m *MockMuleSubscriptionClient) GetApp(appID string) (*anypoint.Application, error) {
	return m.app, m.err
}

func (m *MockMuleSubscriptionClient) ResetAppSecret(appID string) (*anypoint.Application, error) {
	return m.newApp, m.rotateErr
}

func (m *MockMuleSubscriptionClient) CreateIfNotExistingSLATier(appID string) (string, error) {
	return "", nil
}
