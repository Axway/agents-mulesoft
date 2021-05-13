package traceability

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/jobs"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/sirupsen/logrus"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

const healthCheckEndpoint = "ingestion"

type Emitter interface {
	Start() error
	OnConfigChange(gatewayCfg *config.AgentConfig)
}

// MuleEventEmitter - Gathers analytics data for publishing to Central
type MuleEventEmitter struct {
	client            anypoint.AnalyticsClient
	consecutiveErrors int
	eventChannel      chan string
	jobID             string
}

// MuleEventEmitterJob wraps an Emitter and implements the Job interface so that it can be executed by the sdk.
type MuleEventEmitterJob struct {
	Emitter
	consecutiveErrors int
	jobID             string
	pollInterval      time.Duration
	getStatusLevel    hc.GetStatusLevel
}

// NewMuleEventEmitter - Creates a client to poll for events.
func NewMuleEventEmitter(eventChannel chan string, client anypoint.AnalyticsClient) *MuleEventEmitter {
	return &MuleEventEmitter{
		eventChannel: eventChannel,
		client:       client,
	}
}

// Start retrieves analytics data from anypoint and sends them on the event channel for processing.
func (me *MuleEventEmitter) Start() error {
	oldTime := time.Now()

	events, err := me.client.GetAnalyticsWindow()

	currentTime := time.Now()
	duration := currentTime.Sub(oldTime)
	logrus.WithFields(logrus.Fields{
		"duration": fmt.Sprintf("%d ms", duration.Milliseconds()), "count": len(events)}).Debug("retrieved events from anypoint")

	if err != nil {
		logrus.WithError(err).Error("failed to get analytics data")
		return err
	}

	for _, event := range events {
		j, err := json.Marshal(event)
		if err != nil {
			log.Warnf("failed to marshal event: %s", err.Error())
		}
		me.eventChannel <- string(j)
	}
	return nil
}

// OnConfigChange passes the new config to the client to handle config changes
// since the MuleEventEmitter does not have any config value references.
func (me *MuleEventEmitter) OnConfigChange(gatewayCfg *config.AgentConfig) {
	me.client.OnConfigChange(gatewayCfg.MulesoftConfig)
}

// NewMuleEventEmitterJob creates a struct that implements the Emitter and Job interfaces.
func NewMuleEventEmitterJob(
	emitter Emitter,
	pollInterval time.Duration,
	checkStatus hc.CheckStatus,
	getStatus func(endpoint string) hc.StatusLevel,
) (*MuleEventEmitterJob, error) {
	_, err := hc.RegisterHealthcheck("Data Ingestion Endpoint", healthCheckEndpoint, checkStatus)
	if err != nil {
		return nil, err
	}

	return &MuleEventEmitterJob{
		Emitter:        emitter,
		pollInterval:   pollInterval,
		getStatusLevel: getStatus,
	}, nil
}

// Start registers the job with the sdk.
func (m *MuleEventEmitterJob) Start() error {
	jobID, err := jobs.RegisterIntervalJob(m, m.pollInterval)
	m.jobID = jobID
	return err
}

// OnConfigChange updates the MuleEventEmitterJob with any config changes, and calls OnConfigChange on the Emitter
func (m *MuleEventEmitterJob) OnConfigChange(gatewayCfg *config.AgentConfig) {
	m.pollInterval = gatewayCfg.MulesoftConfig.PollInterval
	m.Emitter.OnConfigChange(gatewayCfg)
}

// Execute called by the sdk on each interval.
func (m *MuleEventEmitterJob) Execute() error {
	return m.Emitter.Start()
}

// Status Performs a health check for this job before it is executed.
func (m *MuleEventEmitterJob) Status() error {
	max := 3
	status := m.getStatusLevel(healthCheckEndpoint)

	if status == hc.OK {
		m.consecutiveErrors = 0
	} else {
		m.consecutiveErrors++
	}

	if m.consecutiveErrors >= max {
		// If the job fails 3 times return an error
		return fmt.Errorf("failed to start the Traceability agent %d times in a row", max)
	}

	return nil
}

// Ready determines if the job is ready to run.
func (m *MuleEventEmitterJob) Ready() bool {
	status := m.getStatusLevel(healthCheckEndpoint)
	if status == hc.OK {
		return true
	}
	return false
}

// Check the status of the connection to mulesoft
func traceabilityHealthCheck(name string) *hc.Status {
	health := hc.GetStatus(anypoint.HealthCheckEndpoint)
	if health == hc.FAIL {
		return &hc.Status{
			Result:  hc.FAIL,
			Details: fmt.Sprintf("%s Failed. Unable to connect to Mulesoft, check Mulesoft configuration.", name),
		}
	}
	return &hc.Status{
		Result: hc.OK,
	}
}
