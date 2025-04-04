package anypoint

import (
	"fmt"
	"time"

	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/mock"

	"github.com/Axway/agent-sdk/pkg/api"
)

type MockAuth struct {
	ch chan bool
}

func (m MockAuth) Stop() {
	m.ch <- true
}

func (m MockAuth) GetToken() string {
	return "abc123"
}

func (m MockAuth) GetOrgID() string {
	return "444"
}

type MockClientBase struct {
	Reqs map[string]*api.Response
}

func (mc *MockClientBase) Send(request api.Request) (*api.Response, error) {
	req, ok := mc.Reqs[request.URL]
	if ok {
		return req, nil
	}

	return nil, fmt.Errorf("no request found for %s", request.URL)
}

type MockAnypointClient struct {
	mock.Mock
}

func (m *MockAnypointClient) OnConfigChange(*config.MulesoftConfig) {
	// intentionally left empty for this mock
}

func (m *MockAnypointClient) GetAPI(apiID string) (*API, error) {
	args := m.Called()
	result := args.Get(0)
	return result.(*API), args.Error(1)
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

func (m *MockAnypointClient) GetPolicies(string) ([]Policy, error) {
	args := m.Called()
	result := args.Get(0)
	return result.([]Policy), args.Error(1)
}

func (m *MockAnypointClient) GetExchangeAsset(_, _, _ string) (*ExchangeAsset, error) {
	args := m.Called()
	result := args.Get(0)
	return result.(*ExchangeAsset), args.Error(1)
}

func (m *MockAnypointClient) GetExchangeAssetIcon(_ string) (string, string, error) {
	args := m.Called()
	icon := args.String(0)
	contentType := args.String(1)
	return icon, contentType, args.Error(2)
}

func (m *MockAnypointClient) GetExchangeFileContent(_, _, _ string, shouldConvert bool) ([]byte, bool, error) {
	args := m.Called()
	result := args.Get(0)
	return result.([]byte), shouldConvert, args.Error(2)
}

func (m *MockAnypointClient) GetMonitoringArchive(apiID string, startDate time.Time) ([]APIMonitoringMetric, error) {
	args := m.Called()
	result := args.Get(0)
	return result.([]APIMonitoringMetric), args.Error(1)
}

func (m *MockAnypointClient) CreateClientApplication(apiID string, body *AppRequestBody) (*Application, error) {
	args := m.Called()
	result := args.Get(0)
	return result.(*Application), args.Error(1)
}

func (m *MockAnypointClient) CreateContract(appID string, contract *Contract) (*Contract, error) {
	args := m.Called()
	return contract, args.Error(1)
}

func (m *MockAnypointClient) GetSLATiers(apiID, tierName string) (*Tiers, error) {
	return &Tiers{
		Total: 1,
		Tiers: []SLATier{
			{
				ID:   ToPointer[int](14214),
				Name: tierName,
			},
		},
	}, nil
}

func (m *MockAnypointClient) CreateSLATier(apiID string) (int, error) {
	return 1, nil
}

func (m *MockAnypointClient) DeleteClientApplication(appID string) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAnypointClient) GetClientApplication(appID string) (*Application, error) {
	return nil, nil
}

func (m *MockAnypointClient) DeleteContract(apiID, contractID string) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAnypointClient) RevokeContract(apiID, contractID string) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAnypointClient) ResetAppSecret(appID string) (*Application, error) {
	return nil, nil
}
