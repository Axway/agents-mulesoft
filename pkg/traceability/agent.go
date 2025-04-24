package traceability

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Axway/agent-sdk/pkg/agent"
	coreagent "github.com/Axway/agent-sdk/pkg/agent"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	cache "github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	coreutil "github.com/Axway/agent-sdk/pkg/util"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	cmn "github.com/Axway/agents-mulesoft/pkg/common"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

// Agent - mulesoft Beater configuration. Implements the beat.Beater interface.
type Agent struct {
	logger          log.FieldLogger
	client          beat.Client
	doneCh          chan struct{}
	eventChannel    chan cmn.MetricEvent
	mule            Emitter
	credentialCache cache.Cache
	eventGenerator  transaction.EventGenerator
	eventReport     transaction.EventReport
}

// NewBeater creates an instance of mulesoft_traceability_agent.
func NewBeater(_ *beat.Beat, _ *common.Config) (beat.Beater, error) {
	eventChannel := make(chan cmn.MetricEvent)
	agentConfig := config.GetConfig()
	pollInterval := agentConfig.MulesoftConfig.PollInterval

	var err error
	client := anypoint.NewClient(agentConfig.MulesoftConfig)
	emitter := NewMuleEventEmitter(agentConfig.MulesoftConfig, eventChannel, client, agent.GetCacheManager())

	emitterJob, err := NewMuleEventEmitterJob(emitter, pollInterval, traceabilityHealthCheck, hc.GetStatus, hc.RegisterHealthcheck)
	if err != nil {
		return nil, err
	}

	credentialCache := cache.New()
	credentialHandler := NewCredentialHandler(credentialCache, agent.GetCacheManager())
	agent.RegisterResourceEventHandler(management.CredentialGVK().Kind, credentialHandler)

	return newAgent(emitterJob, eventChannel, credentialCache)
}

func newAgent(
	emitter Emitter,
	eventChannel chan cmn.MetricEvent,
	credentialCache cache.Cache,
) (*Agent, error) {
	a := &Agent{
		logger:          log.NewFieldLogger().WithPackage("traceability.agent").WithComponent("agent"),
		doneCh:          make(chan struct{}),
		eventChannel:    eventChannel,
		mule:            emitter,
		credentialCache: credentialCache,
		eventGenerator:  transaction.NewEventGenerator(),
	}

	eventReport, err := transaction.NewEventReportBuilder().SetOnlyTrackMetrics(true).Build()
	if err != nil {
		a.logger.WithError(err).Error("failed to create event report")
		return nil, err
	}
	a.eventReport = eventReport

	return a, nil
}

// Run starts the Mulesoft traceability agent.
func (a *Agent) Run(b *beat.Beat) error {
	coreagent.OnConfigChange(a.onConfigChange)

	var err error
	a.client, err = b.Publisher.Connect()
	if err != nil {
		coreagent.UpdateStatus(coreagent.AgentFailed, err.Error())
		return err
	}

	go a.mule.Start()

	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, os.Interrupt)

	for {
		select {
		case <-a.doneCh:
			return a.client.Close()
		case <-gracefulStop:
			return a.client.Close()
		case event := <-a.eventChannel:
			a.processEvent(event)
		}
	}
}

func (a *Agent) processEvent(me cmn.MetricEvent) {
	switch me.Type {
	case cmn.Initialize:
		a.logger.Debug("starting metrics processing")
	case cmn.Metric:
		a.processMetricEvent(me.Metric)
	case cmn.Completed:
		err := a.eventGenerator.AddMetricDetailsFromEventReport(a.eventReport)
		if err != nil {
			a.logger.WithError(err).Error("failed to add metrics to collector")
		}
	}
}

func (a *Agent) processMetricEvent(m cmn.Metrics) {
	if m.Instance == nil {
		return
	}

	a.eventReport.AddMetricDetail(metric.MetricDetail{
		APIDetails: a.getAPIDetails(m),
		AppDetails: a.getAppDetails(m),
		StatusCode: m.StatusCode,
		Count:      m.Count,
		Response: metric.ResponseMetrics{
			Max: m.Max,
			Min: m.Min,
		},
		Observation: models.ObservationDetails{
			Start: m.StartTime.UnixMilli(),
			End:   m.EndTime.UnixMilli(),
		},
	})
}

func (a *Agent) getAPIDetails(m cmn.Metrics) models.APIDetails {
	apisRef := m.Instance.GetReferenceByGVK(management.APIServiceGVK())
	externalAPIID, _ := coreutil.GetAgentDetailsValue(m.Instance, definitions.AttrExternalAPIID)
	stage, _ := coreutil.GetAgentDetailsValue(m.Instance, definitions.AttrExternalAPIStage)
	return models.APIDetails{
		ID:                 externalAPIID,
		Name:               apisRef.Name,
		Revision:           1,
		APIServiceInstance: m.Instance.Name,
		Stage:              stage,
	}
}

func (a *Agent) getAppDetails(me cmn.Metrics) models.AppDetails {
	appDetails := models.AppDetails{}
	if item, err := a.credentialCache.Get(me.ClientID); err == nil && item != nil {
		ri, ok := item.(*v1.ResourceInstance)
		if ok && ri != nil {
			appRef := ri.GetReferenceByGVK(management.ManagedApplicationGVK())
			app := agent.GetCacheManager().GetManagedApplicationByName(appRef.Name)
			if app != nil {
				managedApp := &management.ManagedApplication{}
				managedApp.FromInstance(app)
				appDetails = models.AppDetails{
					ID:            managedApp.Metadata.ID,
					Name:          managedApp.Name,
					ConsumerOrgID: managedApp.Marketplace.Resource.Owner.Organization.ID,
				}
			}
		}
	}
	return appDetails
}

// onConfigChange apply configuration changes
func (a *Agent) onConfigChange() {
	cfg := config.GetConfig()
	a.mule.OnConfigChange(cfg)
}

// Stop stops the agent.
func (a *Agent) Stop() {
	a.doneCh <- struct{}{}
}
