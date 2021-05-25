package anypoint

import (
	"net/http"
	"testing"
	"time"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	cfg := &config.MulesoftConfig{
		AnypointExchangeURL: "",
		CachePath:           "/tmp",
		Environment:         "Sandbox",
		OrgName:             "BusinessOrg1",
		Password:            "abc",
		PollInterval:        10,
		ProxyURL:            "",
		SessionLifetime:     60,
		Username:            "123",
	}
	mcb := &MockClientBase{}
	mcb.Res = map[string]*api.Response{
		"/accounts/login": {
			Code:    200,
			Body:    []byte(`{"access_token":"abc123"}`),
			Headers: nil,
		},
		"/accounts/api/me": {
			Code: 200,
			Body: []byte(`{
							"user": {
								"identityType": "idtype",
								"id": "123",
								"username": "name",
								"firstName": "first",
								"lastName": "last",
								"email": "email",
								"organization": {
									"id": "333",
									"name": "org1",
									"domain": "abc.com"
								},
								"memberOfOrganizations": [{
										"id": "333",
										"name": "org1"
									},
									{
										"id": "444",
										"name": "BusinessOrg1"
									}
								]
						
							}
				}`),
		},
		"/accounts/api/organizations/444/environments": {
			Code: 200,
			Body: []byte(`{
					"data": [{
						"id": "111",
						"name": "Sandbox",
						"organizationId": "444",
						"type": "fake",
						"clientId": "abc123"
					}],
					"total": 1
				}`),
		},
		"/apimanager/api/v1/organizations/444/environments/111/apis": {
			Code: 200,
			Body: []byte(`{
				"assets": [
					{
						"apis": []
					}
				],
				"total": 1
			}`),
		},
		"/apimanager/api/v1/organizations/444/environments/111/apis/10/policies": {
			Code: 200,
			Body: []byte(`{
			"policies": [
					{
						"id": 0
					}
				]
			}`),
		},
		"/exchange/api/v2/assets/1/2/3": {
			Code: 200,
			Body: []byte(`{
				"assetId": "petstore"
			}`),
		},
		"/icon": {
			Code: 200,
			Body: []byte(`content`),
		},
		"/analytics/1.0/444/environments/111/events": {
			Code: 200,
			Body: []byte(`[{}]`),
		},
		"https://123.com": {
			Code: 500,
			Body: []byte(`{}`),
		},
		"emeptyslice.com": {
			Code: 200,
			Body: []byte(`[]`),
		},
		"/apimanager/api/v1/organizations/444/environments/111/apis/1/tiers": {
			Code: 200,
			Body: []byte(`{"total":1, "tiers":[{"description":"hi","limits":[],"status":"status","name":"tier1"}]}`),
		},
	}

	client, err := NewClient(cfg, mockRegisterHealth, SetClient(mcb))
	assert.Nil(t, err)

	ma := &MockAuth{
		ch: make(chan bool),
	}
	err = client.Authenticate()
	assert.Nil(t, err)

	client.auth = ma
	status := client.healthcheck("check")
	assert.Equal(t, hc.OK, status.Result)

	req := api.Request{
		Method:      "GET",
		URL:         "https://abc.com",
		QueryParams: nil,
		Headers:     nil,
		Body:        nil,
	}
	// test that invoke can throw an error when communication to the gateway cannot be established
	_, _, err = client.invoke(req)
	assert.NotNil(t, err)

	// test that invoke can throw an error when the endpoint returns a non success response
	req.URL = "https://123.com"
	_, _, err = client.invoke(req)
	assert.NotNil(t, err)
	req.URL = "fake.com"
	err = client.invokeJSON(req, map[string]interface{}{})
	assert.NotNil(t, err)

	token, user, duration, err := client.GetAccessToken()
	logrus.Info(token, user, duration, err)
	assert.Equal(t, "abc123", token)
	assert.Equal(t, "123", user.ID)
	assert.Equal(t, "444", user.Organization.ID)
	assert.Equal(t, time.Duration(60), duration)
	assert.Equal(t, nil, err)

	env, err := client.GetEnvironmentByName("/env1")
	assert.Nil(t, err)
	assert.Equal(t, "Sandbox", env.Name)

	assets, err := client.ListAssets(&Page{
		Offset:   0,
		PageSize: 50,
	})
	assert.Equal(t, 1, len(assets))
	assert.Nil(t, err)

	py, err := client.GetPolicies(10)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(py.Policies))

	a, err := client.GetExchangeAsset("1", "2", "3")
	assert.Nil(t, err)
	assert.Equal(t, "petstore", a.AssetID)

	i, contentType, err := client.GetExchangeAssetIcon("/icon")
	assert.Nil(t, err)
	logrus.Info(i, contentType)
	assert.NotEmpty(t, i)
	assert.Empty(t, contentType)

	events, err := client.GetAnalyticsWindow("2021-05-19T14:30:20-07:00", "2021-05-19T14:30:22-07:00")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(events))

	slaTiers, err := client.GetSLATiers(1)
	assert.Nil(t, err)
	assert.NotNil(t, slaTiers)
	assert.Equal(t, 1, slaTiers.Total)
	assert.Equal(t, 1, len(slaTiers.Tiers))

	go client.auth.Stop()
	done := <-ma.ch
	assert.True(t, done)
}

func TestGetTokenByClientID(t *testing.T) {
	cfg := &config.MulesoftConfig{
		AnypointExchangeURL: "",
		CachePath:           "/tmp",
		Environment:         "Sandbox",
		OrgName:             "master",
		Password:            "",
		PollInterval:        10,
		ProxyURL:            "",
		SessionLifetime:     60,
		Username:            "",
	}

	mcb := &MockClientBase{}

	mcb.Res = map[string]*api.Response{
		"/accounts/oauth2/token": {
			Code:    200,
			Body:    []byte(`{"access_token":"abc123"}`),
			Headers: nil,
		},
	}

	client, err := NewClient(cfg, mockRegisterHealth, SetClient(mcb))
	assert.Nil(t, err)

	ma := &MockAuth{
		ch: make(chan bool),
	}
	client.auth = ma
}

func Test_invokeDelete(t *testing.T) {
	cfg := &config.MulesoftConfig{
		AnypointExchangeURL: "",
		CachePath:           "/tmp",
		Environment:         "Sandbox",
		OrgName:             "BusinessOrg1",
		Password:            "abc",
		PollInterval:        10,
		ProxyURL:            "",
		SessionLifetime:     60,
		Username:            "123",
	}
	mcb := &MockClientBase{}
	mcb.Res = map[string]*api.Response{
		"/api": {
			Code: 204,
			Body: nil,
		},
		"/500": {
			Code: 500,
			Body: nil,
		},
	}
	client, err := NewClient(cfg, mockRegisterHealth, SetClient(mcb))

	req1 := api.Request{
		Method: http.MethodDelete,
		URL:    "/api",
	}
	err = client.invokeDelete(req1)
	assert.Nil(t, err)

	req2 := api.Request{
		Method: http.MethodDelete,
		URL:    "/500",
	}
	err = client.invokeDelete(req2)
	assert.NotNil(t, err)

	req3 := api.Request{
		Method: http.MethodDelete,
		URL:    "/notfound",
	}
	err = client.invokeDelete(req3)
	assert.NotNil(t, err)
}

func mockRegisterHealth(name, endpoint string, check hc.CheckStatus) (string, error) {
	return "", nil
}
