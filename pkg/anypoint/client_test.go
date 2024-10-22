package anypoint

import (
	"testing"
	"time"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/assert"
)

var metricData = `
{"format":"v2","time":1585082947062,"type":"api_summary_metric","commons":{"deployment_type":"RTF","api_id":"204393","cluster_id":"rtf","env_id":"env","public_ip":"127.0.0.1","org_id":"org","worker_id":"worker-1"},"events":[{"response_size.max":2,"request_size.min":6,"status_code":"200","method":"POST","response_time.max":4,"api_version_id":"223337","response_size.count":1,"response_size.sum":2,"response_time.min":4,"request_size.count":1,"api_version":"v1:223337","request_size.sos":36,"client_id":"eb30101d7394407ea86f0643e1c63331","response_time.count":1,"response_time.sum":4,"request_size.max":6,"request_disposition":"processed","response_time.sos":16,"api_name":"groupId:6046b96d-c9aa-4cb2-9b30-90a54fc01a7b:assetId:policy_sla_rate_limit","response_size.min":2,"request_size.sum":6,"response_size.sos":4}],"metadata":{"batch_id":0,"aggregated":true,"limited":false,"producer_name":"analytics-metrics-collector-mule3","producer_version":"2.2.2-SNAPSHOT"}}
`

func TestClient(t *testing.T) {
	cfg := &config.MulesoftConfig{
		AnypointExchangeURL:   "",
		AnypointMonitoringURL: "",
		CachePath:             "/tmp",
		Environment:           "Sandbox",
		OrgName:               "BusinessOrg1",
		PollInterval:          10,
		ProxyURL:              "",
		SessionLifetime:       60,
		ClientID:              "1",
		ClientSecret:          "2",
	}
	mcb := &MockClientBase{}
	mcb.Reqs = map[string]*api.Response{
		"/accounts/api/v2/oauth2/token": {
			Code:    200,
			Body:    []byte(`{"access_token":"abc123","expires_in":3600}`),
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
		"/accounts/api/organizations/333/environments": {
			Code: 200,
			Body: []byte(`{
					"data": [{
						"id": "111",
						"name": "name",
						"organizationId": "333",
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
			Body: []byte(`[
				{
					"id": 0
				}
			]`),
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
		"https://123.com": {
			Code: 500,
			Body: []byte(`{}`),
		},
		"emeptyslice.com": {
			Code: 200,
			Body: []byte(`[]`),
		},
		"/monitoring/archive/api/v1/organizations/444/environments/111/apis/222/summary/2024/01/01": {
			Code: 200,
			Body: []byte(`{
			"resources": [
				{
					"id": "444-111-222.log"
				}
			]
			}`),
		},
		"/monitoring/archive/api/v1/organizations/444/environments/111/apis/222/summary/2024/01/01/444-111-222.log": {
			Code: 200,
			Body: []byte(metricData),
		},
	}

	client := NewClient(cfg, SetClient(mcb))
	ma := &MockAuth{
		ch: make(chan bool),
	}
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
	_, _, err := client.invoke(req)
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
	assert.Equal(t, time.Hour, duration)
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
	py, err := client.GetPolicies("10")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(py))
	a, err := client.GetExchangeAsset("1", "2", "3")
	assert.Nil(t, err)
	assert.Equal(t, "petstore", a.AssetID)
	i, contentType, err := client.GetExchangeAssetIcon("/icon")
	assert.Nil(t, err)
	logrus.Info(i, contentType)
	assert.NotEmpty(t, i)
	assert.Empty(t, contentType)

	startTime, _ := time.Parse(time.RFC3339, "2024-01-01T14:30:20-07:00")

	events, err := client.GetMonitoringArchive("222", startTime)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(events))

	go client.auth.Stop()
	done := <-ma.ch
	assert.True(t, done)
}
