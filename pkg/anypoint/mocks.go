package anypoint

import (
	"fmt"
	"time"

	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/mock"

	"github.com/Axway/agent-sdk/pkg/api"
)

type MockClient struct {
	Reqs map[string]*api.Response
}

func (mc *MockClient) Send(request api.Request) (*api.Response, error) {
	req, ok := mc.Reqs[request.URL]
	if ok {
		return req, nil
	} else {
		return nil, fmt.Errorf("no request found for %s", request.URL)
	}
}

type MockAnypointClient struct {
	mock.Mock
}

func (m *MockAnypointClient) OnConfigChange(*config.MulesoftConfig) {
}

func (m *MockAnypointClient) GetAccessToken() (string, *User, time.Duration, error) {
	args := m.Called()
	token := args.String(0)
	user := args.Get(1).(*User)
	duration := args.Get(2).(time.Duration)
	return token, user, duration, args.Error(1)
}

func (m *MockAnypointClient) GetEnvironmentByName(string) (*Environment, error) {
	args := m.Called()
	result := args.Get(0)
	return result.(*Environment), args.Error(1)
}

func (m *MockAnypointClient) ListAssets(*Page) ([]Asset, error) {
	args := m.Called()
	result := args.Get(0)
	return result.([]Asset), args.Error(1)
}

func (m *MockAnypointClient) GetPolicies(*API) (Policies, error) {
	args := m.Called()
	result := args.Get(0)
	return result.(Policies), args.Error(1)
}

func (m *MockAnypointClient) GetExchangeAsset(*API) (*ExchangeAsset, error) {
	args := m.Called()
	result := args.Get(0)
	return result.(*ExchangeAsset), args.Error(1)
}

func (m *MockAnypointClient) GetExchangeAssetIcon(*ExchangeAsset) (string, string, error) {
	args := m.Called()
	icon := args.String(0)
	contentType := args.String(1)
	return icon, contentType, args.Error(2)
}

func (m *MockAnypointClient) GetExchangeFileContent(*ExchangeFile) ([]byte, error) {
	args := m.Called()
	result := args.Get(0)
	return result.([]byte), args.Error(1)
}

func (m *MockAnypointClient) GetAnalyticsWindow() ([]AnalyticsEvent, error) {
	args := m.Called()
	result := args.Get(0)
	return result.([]AnalyticsEvent), args.Error(1)
}
