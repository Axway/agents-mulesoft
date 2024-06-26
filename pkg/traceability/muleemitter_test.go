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
	eventCh := make(chan string)
	client := &mockAnalyticsClient{
		events: []anypoint.AnalyticsEvent{event},
		err:    nil,
	}
	emitter := NewMuleEventEmitter("/tmp", eventCh, client)

	assert.NotNil(t, emitter)

	go emitter.Start()

	e := <-eventCh
	assert.NotEmpty(t, e)

	// Should throw an error when the client returns an error
	client = &mockAnalyticsClient{
		events: []anypoint.AnalyticsEvent{},
		err:    fmt.Errorf("failed"),
	}
	emitter = NewMuleEventEmitter("/tmp", eventCh, client)
	err := emitter.Start()
	assert.Equal(t, client.err, err)
}

func TestMuleEventEmitterJob(t *testing.T) {
	pollInterval := 1 * time.Second
	ac := &config.AgentConfig{
		CentralConfig: corecfg.NewCentralConfig(corecfg.TraceabilityAgent),
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: pollInterval,
		},
	}

	eventCh := make(chan string)
	client := &mockAnalyticsClient{
		events: []anypoint.AnalyticsEvent{event},
		err:    nil,
	}
	emitter := NewMuleEventEmitter("/tmp", eventCh, client)

	job, err := NewMuleEventEmitterJob(emitter, pollInterval, mockHealthCheck, getStatusSuccess, mockRegisterHC)
	assert.Nil(t, err)
	assert.Equal(t, pollInterval, job.pollInterval)

	// expect the poll interval value to change when the config changes.
	ac.MulesoftConfig.PollInterval = 2 * time.Second
	job.OnConfigChange(ac)
	assert.Equal(t, ac.MulesoftConfig.PollInterval, job.pollInterval)

	// should set the id of the job
	err = job.Start()
	assert.Nil(t, err)
	assert.NotEmpty(t, job.jobID)

	ok := job.Ready()
	assert.True(t, ok)

	// Should fail when there is no anypoint health check registered
	status := traceabilityHealthCheck("trace")
	assert.Equal(t, hc.FAIL, status.Result)

	// Register the anypoint healthcheck, since the traceability health check depends on it.
	// Should pass when it returns ok
	hc.RegisterHealthcheck("mulesoft", anypoint.HealthCheckEndpoint, mockHealthCheck)
	status = traceabilityHealthCheck("mulesoft")
	assert.Equal(t, hc.OK, status.Result)
}

func mockHealthCheck(string) *hc.Status {
	return &hc.Status{
		Result: hc.OK,
	}
}

func getStatusSuccess(string) hc.StatusLevel {
	return hc.OK
}

func mockRegisterHC(name, endpoint string, check hc.CheckStatus) (string, error) {
	fmt.Println("Here")
	return string(hc.OK), nil
}
