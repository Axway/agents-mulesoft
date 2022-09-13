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

func (m *MockMuleSubscriptionClient) CreateContract(apiID, tier string, appID int64) (*anypoint.Contract, error) {
	return m.contract, m.err
}

func (m *MockMuleSubscriptionClient) DeleteApp(appName int64) error {
	return m.err
}

func (m *MockMuleSubscriptionClient) DeleteContract(apiID, contractID string) error {
	return m.err
}

func (m *MockMuleSubscriptionClient) GetApp(id string) (*anypoint.Application, error) {
	return m.app, m.err
}

func (m *MockMuleSubscriptionClient) ResetAppSecret(appID int64) (*anypoint.Application, error) {
	return m.newApp, m.rotateErr
}
