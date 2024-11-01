package traceability

import (
	"fmt"
	"strconv"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util"

	"github.com/Axway/agent-sdk/pkg/jobs"

	"github.com/sirupsen/logrus"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/common"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

const (
	healthCheckEndpoint = "ingestion"
	CacheKeyTimeStamp   = "LAST_RUN"
)

type instanceCache interface {
	GetAPIServiceInstanceKeys() []string
	GetAPIServiceInstanceByID(id string) (*v1.ResourceInstance, error)
}

type Emitter interface {
	Start() error
	OnConfigChange(gatewayCfg *config.AgentConfig)
}

type healthChecker func(name, endpoint string, check hc.CheckStatus) (string, error)

// MuleEventEmitter - Gathers analytics data for publishing to Central.
type MuleEventEmitter struct {
	client           anypoint.AnalyticsClient
	eventChannel     chan common.MetricEvent
	cache            cache.Cache
	cachePath        string
	instanceCache    instanceCache
	useMonitoringAPI bool
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
func NewMuleEventEmitter(config *config.MulesoftConfig, eventChannel chan common.MetricEvent, client anypoint.AnalyticsClient, instanceCache instanceCache) *MuleEventEmitter {
	me := &MuleEventEmitter{
		eventChannel:     eventChannel,
		client:           client,
		instanceCache:    instanceCache,
		useMonitoringAPI: config.UseMonitoringAPI,
	}
	me.cachePath = formatCachePath(config.CachePath)
	me.cache = cache.Load(me.cachePath)
	return me
}

// Start retrieves analytics data from anypoint and sends them on the event channel for processing.
func (me *MuleEventEmitter) Start() error {
	var bootInfo *anypoint.MonitoringBootInfo
	if !me.useMonitoringAPI {
		bi, err := me.client.GetMonitoringBootstrap()
		if err != nil {
			return err
		}
		bootInfo = bi
	}

	// Initialize Metric Batch
	me.eventChannel <- common.MetricEvent{Type: common.Initialize}

	// Publish metrics, event receiver takes care if no metrics needs to be published
	defer func() {
		me.eventChannel <- common.MetricEvent{Type: common.Completed}
	}()

	// change the cache to store startTime per API
	instanceKeys := me.instanceCache.GetAPIServiceInstanceKeys()
	reportEndTime := time.Now()
	for _, instanceID := range instanceKeys {
		instance, _ := me.instanceCache.GetAPIServiceInstanceByID(instanceID)
		apiID, _ := util.GetAgentDetailsValue(instance, common.AttrAssetID)
		apiVersionID, _ := util.GetAgentDetailsValue(instance, common.AttrAPIID)
		if apiID == "" {
			continue
		}
		lastAPIReportTime := me.getLastRun(apiID)
		metrics, err := me.getMetrics(bootInfo, apiID, apiVersionID, lastAPIReportTime, reportEndTime)
		endTime := lastAPIReportTime
		for _, metric := range metrics {
			// Report only latest entries, ignore old entries
			if metric.Time.UnixMilli() > lastAPIReportTime.UnixMilli() {
				for _, event := range metric.Events {
					m := common.MetricEvent{
						Type: common.Metric,
						Metric: common.Metrics{
							StartTime:  lastAPIReportTime,
							EndTime:    metric.Time,
							APIID:      apiID,
							Instance:   instance,
							StatusCode: event.StatusCode,
							Count:      int64(event.RequestSizeCount),
							Max:        int64(event.ResponseTimeMax),
							Min:        int64(event.ResponseTimeMin),
						},
					}
					me.eventChannel <- m
					logrus.WithField("apiID", apiID).
						WithField("apiVersionID", apiVersionID).
						WithField("statusCode", event.StatusCode).
						WithField("count", event.RequestSizeCount).
						WithField("metricTime", metric.Time).
						Info("storing API metrics")
				}
			}
			// Results are not sorted. We want the most recent time to bubble up for next run cycle
			if metric.Time.UnixMilli() > endTime.UnixMilli() {
				endTime = metric.Time
			}
		}
		logrus.WithField("apiID", apiID).
			WithField("apiVersionID", apiVersionID).
			WithField("lastReportTime", endTime).
			Info("updating next query time")
		me.saveLastRun(apiID, endTime)
		if err != nil {
			logrus.WithError(err).Error("failed to get analytics data")
			return err
		}
	}

	return nil

}

func (me *MuleEventEmitter) getMetrics(bootInfo *anypoint.MonitoringBootInfo, apiID, apiVersionID string, startTime, endTime time.Time) ([]anypoint.APIMonitoringMetric, error) {
	if me.useMonitoringAPI {
		return me.client.GetMonitoringArchive(apiID, startTime)
	}

	return me.client.GetMonitoringMetrics(bootInfo.Settings.DataSource.InfluxDB.Database, bootInfo.Settings.DataSource.InfluxDB.ID, apiID, apiVersionID, startTime, endTime)
}

func (me *MuleEventEmitter) getLastRun(apiID string) time.Time {
	tStamp, _ := me.cache.Get(CacheKeyTimeStamp + "-" + apiID)
	tStart := time.Now()
	if tStamp != nil {
		tm, err := strconv.ParseInt(tStamp.(string), 10, 64)
		if err == nil {
			tStart = time.UnixMilli(tm)
		}
	}
	return tStart
}

func (me *MuleEventEmitter) saveLastRun(apiID string, lastTime time.Time) {
	me.cache.Set(CacheKeyTimeStamp+"-"+apiID, fmt.Sprintf("%d", lastTime.UnixMilli()))
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
	registerHC healthChecker,
) (*MuleEventEmitterJob, error) {
	_, err := registerHC("Data Ingestion Endpoint", healthCheckEndpoint, checkStatus)
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
	return status == hc.OK
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
