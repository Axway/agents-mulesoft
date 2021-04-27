package traceability

import (
	"fmt"
	"github.com/Axway/agent-sdk/pkg/api"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type mockClient struct{
	mock.Mock
}

func Test_MuleEventEmitter_Start(t *testing.T) {
	ac := &config.AgentConfig{
		CentralConfig:  corecfg.NewCentralConfig(corecfg.TraceabilityAgent),
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: 1*time.Second,
		},
	}
	eventCh := make(chan string)
	mc := &mockClient{}
	apClient := anypoint.NewClient(ac.MulesoftConfig, anypoint.SetClient(mc))
	emitter, err := NewMuleEventEmitter(ac, eventCh, apClient)
	assert.NotNil(t, emitter)
	assert.Nil(t, err)
	emitter.Start()
	e := <-eventCh
	assert.NotEmpty(t, e)
}

func (mc *mockClient) Send(request api.Request) (*api.Response, error) {
	reqs := map[string]*api.Response{
		"/accounts/login": {
			Code:    200,
			Body:    []byte("{\"access_token\":\"abc123\"}"),
			Headers: nil,
		},
		"/accounts/api/me": {
			Code: 200,
			Body: []byte(`{
					"user":{
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
						}
					}
				}`),
		},
		"/accounts/api/organizations/333/environments": {
			Code: 200,
			Body: []byte(`{
					"data": [{
						"id": "111",
						"name": "Sandbox",
						"organizationId": "333",
						"type": "fake",
						"clientId": "abc123"
					}],
					"total": 1
				}`),
		},
		"/analytics/1.0/333/environments/111/events": {
			Code: 200,
			Body: []byte(`[{}]`),
		},
	}
	req, ok := reqs[request.URL]
	if ok {
		return req, nil
	} else {
		return nil, fmt.Errorf("no request found for %s", request.URL)
	}
}
