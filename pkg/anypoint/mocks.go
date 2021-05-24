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

func (m MockAuth) Start() error {
	return nil
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
	Res map[string]*api.Response
}

func (mc *MockClientBase) Send(request api.Request) (*api.Response, error) {
	res, ok := mc.Res[request.URL]
	if ok {
		return res, nil
	} else {
		return nil, fmt.Errorf("no request found for %s", request.URL)
	}
}

type MockAnypointClient struct {
	mock.Mock
	CreateContractAssertArgs          bool
	CreateClientApplicationAssertArgs bool
}

func (m *MockAnypointClient) Authenticate() error {
	return nil
}

func (m *MockAnypointClient) OnConfigChange(*config.MulesoftConfig) {
}

func (m *MockAnypointClient) GetAccessToken() (string, *User, time.Duration, error) {
	return "", nil, 0, nil
}

func (m *MockAnypointClient) GetEnvironmentByName(string) (*Environment, error) {
	return nil, nil
}

func (m *MockAnypointClient) ListAssets(*Page) ([]Asset, error) {
	args := m.Called()
	result := args.Get(0)
	return result.([]Asset), args.Error(1)
}

func (m *MockAnypointClient) GetPolicies(int64) (Policies, error) {
	args := m.Called()
	result := args.Get(0)
	return result.(Policies), args.Error(1)
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

func (m *MockAnypointClient) GetExchangeFileContent(_, _, _ string) ([]byte, error) {
	args := m.Called()
	result := args.Get(0)
	return result.([]byte), args.Error(1)
}

func (m *MockAnypointClient) GetAnalyticsWindow() ([]AnalyticsEvent, error) {
	return nil, nil
}

func (m *MockAnypointClient) CreateClientApplication(id string, body *AppRequestBody) (*Application, error) {
	var args mock.Arguments

	if m.CreateContractAssertArgs {
		args = m.Called(id, body)
	} else {
		args = m.Called()
	}
	result := args.Get(0)
	return result.(*Application), args.Error(1)
}

func (m *MockAnypointClient) CreateContract(id int64, contract *Contract) (*Contract, error) {
	var args mock.Arguments
	if m.CreateContractAssertArgs {
		args = m.Called(id, contract)
	} else {
		args = m.Called()
	}
	result := args.Get(0)
	return result.(*Contract), args.Error(1)
}

func (m *MockAnypointClient) GetSLATiers(int642 int64) (*Tiers, error) {
	return &Tiers{
		Total: 1,
		Tiers: []SLATier{
			{
				ID:   654272,
				Name: "Gold",
			},
		},
	}, nil
}

func (m *MockAnypointClient) DeleteClientApplication(appId int64) error {
	args := m.Called()
	return args.Error(0)
}
