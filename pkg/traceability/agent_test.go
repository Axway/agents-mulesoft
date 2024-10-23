package traceability

import (
	"testing"
	"time"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	cache "github.com/Axway/agent-sdk/pkg/cache"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/util"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/common"
	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/elastic/beats/v7/libbeat/beat"
	publisher "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/stretchr/testify/assert"
)

type mockMetricCollector struct {
	channel chan bool
	details []metric.MetricDetail
}

func (m *mockMetricCollector) InitializeBatch() {
}

func (m *mockMetricCollector) AddAPIMetricDetail(detail metric.MetricDetail) {
	if m.details == nil {
		m.details = make([]metric.MetricDetail, 0)
	}
	m.details = append(m.details, detail)
}

func (m *mockMetricCollector) Publish() {
	m.channel <- true
}

func TestAgent_Run(t *testing.T) {
	processorChannel := make(chan bool)
	eventChannel := make(chan common.MetricEvent)

	event := anypoint.APIMonitoringMetric{
		Time: time.Now().Add(10 * time.Second),
		Events: []anypoint.APISummaryMetricEvent{
			{
				APIName:          "test",
				ClientID:         "test",
				StatusCode:       "200",
				RequestSizeCount: 1,
				ResponseTimeMax:  2,
				ResponseTimeMin:  1,
			},
		},
	}
	client := &mockAnalyticsClient{
		events: []anypoint.APIMonitoringMetric{event},
		err:    nil,
	}
	instanceCache := &mockInstaceCache{}
	svcInst := management.NewAPIServiceInstance("api", "env")
	util.SetAgentDetailsKey(svcInst, common.AttrAPIID, "1234")
	util.SetAgentDetailsKey(svcInst, common.AttrAssetID, "1234")
	svcInst.Metadata.ID = "1234"
	ri, _ := svcInst.AsInstance()
	instanceCache.AddAPIServiceInstance(ri)
	emitter := NewMuleEventEmitter(&config.MulesoftConfig{CachePath: "/tmp", UseMonitoringAPI: true}, eventChannel, client, instanceCache)
	collector := &mockMetricCollector{
		channel: processorChannel,
	}
	credCache := cache.New()
	traceAgent, err := newAgent(emitter, eventChannel, collector, credCache)

	assert.Nil(t, err)
	assert.NotNil(t, traceAgent)

	pubClient := publisher.NewChanClientWith(make(chan beat.Event))

	pub := publisher.PublisherWithClient(pubClient)
	b := &beat.Beat{
		Publisher: pub,
	}

	cfg := &config.AgentConfig{
		CentralConfig: corecfg.NewCentralConfig(corecfg.TraceabilityAgent),
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: 2 * time.Second,
		},
	}

	config.SetConfig(cfg)
	traceAgent.onConfigChange()

	go traceAgent.Run(b)

	done := <-processorChannel
	assert.True(t, done)
	traceAgent.Stop()
}

type mockAnalyticsClient struct {
	events []anypoint.APIMonitoringMetric
	app    *anypoint.Application
	err    error
}

func (m mockAnalyticsClient) GetMonitoringBootstrap() (*anypoint.MonitoringBootInfo, error) {
	return nil, m.err
}

func (m mockAnalyticsClient) GetMonitoringMetrics(dataSourceName string, dataSourceID int, apiID, apiVersionID string, startDate, endTime time.Time) ([]anypoint.APIMonitoringMetric, error) {
	return m.events, m.err
}

func (m mockAnalyticsClient) GetMonitoringArchive(apiID string, startDate time.Time) ([]anypoint.APIMonitoringMetric, error) {
	return m.events, m.err
}

func (m mockAnalyticsClient) GetClientApplication(string) (*anypoint.Application, error) {
	return m.app, m.err
}

func (m mockAnalyticsClient) OnConfigChange(_ *config.MulesoftConfig) {
}

func (m mockAnalyticsClient) GetAPI(_ string) (*anypoint.API, error) {
	return nil, nil
}
