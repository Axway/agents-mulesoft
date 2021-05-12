package traceability

import (
	"fmt"
	"testing"
	"time"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"

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
	client := &mockAnalyticsClient{
		events: []anypoint.AnalyticsEvent{event},
		err:    nil,
	}
	emitter := NewMuleEventEmitter(ac, eventCh, client)

	assert.NotNil(t, emitter)

	go emitter.Start()

	e := <-eventCh
	assert.NotEmpty(t, e)

	// Should throw an error when the client returns an error
	client = &mockAnalyticsClient{
		events: []anypoint.AnalyticsEvent{},
		err:    fmt.Errorf("failed"),
	}
	emitter = NewMuleEventEmitter(ac, eventCh, client)
	err := emitter.Start()
	assert.Equal(t, client.err, err)

	ac.MulesoftConfig.PollInterval = 2 * time.Second
	emitter.OnConfigChange(ac)
	assert.Equal(t, ac.MulesoftConfig.PollInterval, emitter.pollInterval)
}

func TestMuleEventEmitterJob(t *testing.T) {
	ac := &config.AgentConfig{
		CentralConfig: corecfg.NewCentralConfig(corecfg.TraceabilityAgent),
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: 1 * time.Second,
		},
	}

	eventCh := make(chan string)
	client := &mockAnalyticsClient{
		events: []anypoint.AnalyticsEvent{event},
		err:    nil,
	}
	emitter := NewMuleEventEmitter(ac, eventCh, client)

	job, err := NewMuleEventEmitterJob(emitter, emitter.pollInterval, mockHealthCheck)
	assert.Nil(t, err)
	assert.Equal(t, emitter.pollInterval, job.pollInterval)

	// should set the id of the job
	err = job.Start()
	assert.Nil(t, err)
	assert.NotEmpty(t, job.jobID)

	// trigger the health check job so the Status func passes
	hc.RunChecks()

	err = job.Status()
	assert.Nil(t, err)
	assert.Equal(t, 0, job.consecutiveErrors)

	ok := job.Ready()
	assert.True(t, ok)

	// Should fail because there is no anypoint health check registered
	status := traceabilityHealthCheck("trace")
	assert.Equal(t, hc.FAIL, status.Result)

	// Register a mock health check for anypoint so the trace health check passes.
	hc.RegisterHealthcheck("anypoint", anypoint.HealthCheckEndpoint, mockHealthCheck)
	status = traceabilityHealthCheck("trace")
	assert.Equal(t, &hc.Status{Result: hc.OK}, status)
}

func mockHealthCheck(string) *hc.Status {
	return &hc.Status{
		Result: hc.OK,
	}
}

func mockRegisterHealthCheck(name, endpoint string, check hc.CheckStatus) (string, error) {
	return "", nil
}
