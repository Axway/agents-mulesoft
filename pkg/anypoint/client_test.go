package anypoint

import (
	"io"
	"os"
	"testing"
	"time"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/assert"
)

func readTestDataFile(t *testing.T, fileName string) []byte {
	file, _ := os.Open(fileName)
	inputData, err := io.ReadAll(file)
	assert.Nil(t, err)

	return inputData
}

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
			Body: readTestDataFile(t, "./testdata/user.json"),
		},
		"/accounts/api/organizations/444/environments": {
			Code: 200,
			Body: readTestDataFile(t, "./testdata/org-444-envs.json"),
		},
		"/accounts/api/organizations/333/environments": {
			Code: 200,
			Body: readTestDataFile(t, "./testdata/org-333-envs.json"),
		},
		"/apimanager/api/v1/organizations/444/environments/111/apis": {
			Code: 200,
			Body: readTestDataFile(t, "./testdata/apis.json"),
		},
		"/apimanager/api/v1/organizations/444/environments/111/apis/10/policies": {
			Code: 200,
			Body: readTestDataFile(t, "./testdata/policies.json"),
		},
		"/exchange/api/v2/assets/1/2/3": {
			Code: 200,
			Body: readTestDataFile(t, "./testdata/assets.json"),
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
			Body: readTestDataFile(t, "./testdata/summary-datafiles.json"),
		},
		"/monitoring/archive/api/v1/organizations/444/environments/111/apis/222/summary/2024/01/01/444-111-222.log": {
			Code: 200,
			Body: readTestDataFile(t, "./testdata/monitoring-archive.txt"),
		},
		"/monitoring/api/visualizer/api/bootdata": {
			Code: 200,
			Body: readTestDataFile(t, "./testdata/boot-data.json"),
		},
		"/monitoring/api/visualizer/api/datasources/proxy/1234/query": {
			Code: 200,
			Body: readTestDataFile(t, "./testdata/query-response.json"),
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

	bootInfo, err := client.GetMonitoringBootstrap()
	assert.Nil(t, err)
	assert.NotNil(t, bootInfo)

	events, err = client.GetMonitoringMetrics(bootInfo.Settings.DataSource.InfluxDB.Database, bootInfo.Settings.DataSource.InfluxDB.ID, "222", "222", startTime, startTime)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(events))

	go client.auth.Stop()
	done := <-ma.ch
	assert.True(t, done)
}
