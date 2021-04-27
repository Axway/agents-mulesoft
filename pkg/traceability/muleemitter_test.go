package traceability

import (
	"github.com/Axway/agent-sdk/pkg/api"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMuleEventEmitter(t *testing.T) {
	ac := &config.AgentConfig{
		CentralConfig:  corecfg.NewCentralConfig(corecfg.TraceabilityAgent),
		MulesoftConfig: &config.MulesoftConfig{},
	}
	ch := make(chan string)
	mockClient := &api.MockHTTPClient{
		Client:        nil,
		Response:      &api.Response{
			Code:    200,
			Body:    []byte("{\"access_token\":\"abc123\"}"),
			Headers: nil,
		},
		ResponseCode:  200,
		ResponseError: nil,
		RespCount:     0,
		Responses:     nil,
		Requests:      nil,
	}
	apClient := anypoint.NewClient(ac.MulesoftConfig, anypoint.SetClient(mockClient))
	emitter, err := NewMuleEventEmitter(ac, ch, apClient)
	assert.NotNil(t, emitter)
	assert.Nil(t, err)
}
