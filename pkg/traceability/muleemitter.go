package traceability

import (
	"encoding/json"
	"fmt"
	"github.com/Axway/agent-sdk/pkg/cache"
	"time"

	"github.com/Axway/agent-sdk/pkg/jobs"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/sirupsen/logrus"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

const (
	healthCheckEndpoint = "ingestion"
	CacheKeyTimeStamp   = "LAST_RUN"
)

type Emitter interface {
	Start() error
	OnConfigChange(gatewayCfg *config.AgentConfig)
}

// MuleEventEmitter - Gathers analytics data for publishing to Central.
type MuleEventEmitter struct {
	client       anypoint.AnalyticsClient
	eventChannel chan string
	jobID        string
	cache        cache.Cache
	cachePath    string
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
func NewMuleEventEmitter(cachePath string, eventChannel chan string, client anypoint.AnalyticsClient) *MuleEventEmitter {
	me :=&MuleEventEmitter{
		eventChannel: eventChannel,
		client:       client,
	}
    me.cachePath=formatCachePath(cachePath)
	me.cache= cache.Load(me.cachePath)
    return me
}

// Start retrieves analytics data from anypoint and sends them on the event channel for processing.
func (me *MuleEventEmitter) Start() error {
	oldTime := time.Now()
	strStartTime, strEndTime := me.getLastRun()

	events, err := me.client.GetAnalyticsWindow(strStartTime, strEndTime)

	currentTime := time.Now()
	duration := currentTime.Sub(oldTime)
	logrus.WithFields(logrus.Fields{
		"duration": fmt.Sprintf("%d ms", duration.Milliseconds()), "count": len(events)}).Debug("retrieved events from anypoint")

	if err != nil {
		logrus.WithError(err).Error("failed to get analytics data")
		return err
	}

	var lastTime time.Time
	lastTime, err = time.Parse(time.RFC3339, strStartTime)
	if err != nil {
		logrus.WithError(err).Error("Unable to Parse Last Time")
		return err
	}
	for _, event := range events {
		// Results are not sorted. We want the most recent time to bubble up
		if event.Timestamp.After(lastTime){
			lastTime = event.Timestamp
		}
		j, err := json.Marshal(event)
		if err != nil {
			log.Warnf("failed to marshal event: %s", err.Error())
		}
		me.eventChannel <- string(j)
	}
	// Add 1 second to the last time stamp if we found records from this pull.
	// This will prevent duplicate records from being retrieved
	if len(events) > 0 {
		me.saveLastRun(lastTime.Add(time.Second * 1).Format(time.RFC3339))
	}

	return nil

}
func (me *MuleEventEmitter) getLastRun() (string, string) {
	tStamp, _ := me.cache.Get(CacheKeyTimeStamp)
	now := time.Now()
	tNow := now.Format(time.RFC3339)
	if tStamp == nil {
		tStamp = tNow
		me.saveLastRun(tNow)
	}
	return tStamp.(string), tNow
}
func (me *MuleEventEmitter) saveLastRun(lastTime string)  {
	me.cache.Set(CacheKeyTimeStamp, lastTime)
	me.cache.Save(me.cachePath)
}

// OnConfigChange passes the new config to the client to handle config changes
// since the MuleEventEmitter only has cache config value references and should not be changed
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

func formatCachePath(path string) string {
	return fmt.Sprintf("%s/anypoint.cache", path)
}
