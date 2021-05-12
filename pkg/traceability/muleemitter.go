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

// MuleEventEmitter - Represents the Gateway client
type MuleEventEmitter struct {
	client            anypoint.AnalyticsClient
	consecutiveErrors int
	cfg               *config.AgentConfig
	done              chan bool
	eventChannel      chan string
	pollInterval      time.Duration
	jobID             string
}

// MuleEventEmitterJob wraps an Emitter in the Job interface so that it can be executed by the sdk.
type MuleEventEmitterJob struct {
	Emitter
	consecutiveErrors int
	jobID             string
	pollInterval      time.Duration
}

// NewMuleEventEmitter - Creates a client to poll for events.
func NewMuleEventEmitter(
	gatewayCfg *config.AgentConfig,
	eventChannel chan string,
	client anypoint.AnalyticsClient,
) *MuleEventEmitter {
	return &MuleEventEmitter{
		cfg:          gatewayCfg,
		pollInterval: gatewayCfg.MulesoftConfig.PollInterval,
		done:         make(chan bool),
		eventChannel: eventChannel,
		client:       client,
	}
}

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

// OnConfigChange -
func (me *MuleEventEmitter) OnConfigChange(gatewayCfg *config.AgentConfig) {
	me.pollInterval = gatewayCfg.MulesoftConfig.PollInterval
	me.client.OnConfigChange(gatewayCfg.MulesoftConfig)
}

func NewMuleEventEmitterJob(
	emitter Emitter,
	pollInterval time.Duration,
	healthCheck hc.CheckStatus,
) (*MuleEventEmitterJob, error) {
	_, err := hc.RegisterHealthcheck("Data Ingestion Endpoint", healthCheckEndpoint, healthCheck)
	if err != nil {
		return nil, err
	}
	return &MuleEventEmitterJob{
		Emitter:      emitter,
		pollInterval: pollInterval,
	}, nil
}

// Start registers the job with the sdk.
func (m *MuleEventEmitterJob) Start() error {
	jobID, err := jobs.RegisterIntervalJob(m, m.pollInterval)
	m.jobID = jobID
	return err
}

// Execute called by the sdk on each interval.
func (m *MuleEventEmitterJob) Execute() error {
	return m.Emitter.Start()
}

// Status Performs a health check for this job before it is executed.
func (m *MuleEventEmitterJob) Status() error {
	max := 3
	status := hc.GetStatus(healthCheckEndpoint)
	if status == hc.OK {
		m.consecutiveErrors = 0
		return nil
	} else if m.consecutiveErrors >= max {
		// If the job fails 3 times return an error
		return fmt.Errorf("failed to start the Traceability agent %d times in a row", max)
	}
	m.consecutiveErrors++
	return nil
}

// Ready determines if the job is ready to run.
func (m *MuleEventEmitterJob) Ready() bool {
	status := hc.GetStatus(healthCheckEndpoint)
	if status == hc.OK {
		return true
	}
	return false
}

func traceabilityHealthCheck(name string) *hc.Status {
	status := &hc.Status{
		Result: hc.OK,
	}
	// Check the status of the connection to mulesoft
	health := hc.GetStatus(anypoint.HealthCheckEndpoint)
	if health == hc.FAIL {
		status.Result = hc.FAIL
		status.Details = fmt.Sprintf("%s Failed. Unable to connect to Mulesoft, check Mulesoft configuration.", name)
	}
	return status
}
