package traceability

import (
	"testing"
	"time"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_MuleEventEmitter(t *testing.T) {
	ac := &config.AgentConfig{
		CentralConfig: corecfg.NewCentralConfig(corecfg.TraceabilityAgent),
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: 1 * time.Second,
		},
	}

	eventCh := make(chan string)
	emitter := NewMuleEventEmitter(ac, eventCh, &mockAnalyticsClient{})

	assert.NotNil(t, emitter)

	go emitter.Start()

	e := <-eventCh
	assert.NotEmpty(t, e)
}

func mockHealthCheck(string) *hc.Status {
	return &hc.Status{
		Result: hc.OK,
	}
}

func mockRegisterHealthCheck(name, endpoint string, check hc.CheckStatus) (string, error) {
	return "", nil
}
