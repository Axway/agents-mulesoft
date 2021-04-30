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

func (m MockAnypointClient) OnConfigChange(*config.MulesoftConfig) {
}

func (m MockAnypointClient) GetAccessToken() (string, *User, time.Duration, error) {
	return "abc123", &User{}, 10, nil
}

func (m MockAnypointClient) GetEnvironmentByName(name string) (*Environment, error) {
	return nil, nil
}

func (m MockAnypointClient) ListAssets(page *Page) ([]Asset, error) {
	return nil, nil
}

func (m MockAnypointClient) GetPolicies(api *API) ([]Policy, error) {
	return nil, nil
}

func (m MockAnypointClient) GetExchangeAsset(api *API) (*ExchangeAsset, error) {
	return nil, nil
}

func (m MockAnypointClient) GetExchangeAssetIcon(asset *ExchangeAsset) (icon string, contentType string, err error) {
	return "", "", nil
}

func (m MockAnypointClient) GetExchangeFileContent(file *ExchangeFile) (fileContent []byte, err error) {
	return nil, nil
}

func (m MockAnypointClient) GetAnalyticsWindow() ([]AnalyticsEvent, error) {
	return nil, nil
}
