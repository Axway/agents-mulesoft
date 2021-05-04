package traceability

import (
	"testing"
	"time"

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
	emitter, err := NewMuleEventEmitter(ac, eventCh, &mockAnalyticsClient{})

	assert.NotNil(t, emitter)
	assert.Nil(t, err)

	emitter.Start()

	e := <-eventCh
	assert.NotEmpty(t, e)

	emitter.Stop()
}
