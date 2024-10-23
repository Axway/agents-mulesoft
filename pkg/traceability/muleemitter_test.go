package traceability

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/common"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/assert"
)

type mockInstaceCache struct {
	instances cache.Cache
}

func (c *mockInstaceCache) AddAPIServiceInstance(ri *v1.ResourceInstance) {
	if c.instances == nil {
		c.instances = cache.New()
	}
	c.instances.Set(ri.Metadata.ID, ri)
}

func (c *mockInstaceCache) GetAPIServiceInstanceKeys() []string {
	if c.instances == nil {
		c.instances = cache.New()
	}
	return c.instances.GetKeys()
}

func (c *mockInstaceCache) GetAPIServiceInstanceByID(id string) (*v1.ResourceInstance, error) {
	item, err := c.instances.Get(id)
	if err != nil {
		return nil, err
	}
	ri, ok := item.(*v1.ResourceInstance)
	if ok {
		return ri, nil
	}
	return nil, fmt.Errorf("error")
}

type mockEventReceiver struct {
	metricBatchInitialized bool
	metricBatchPublish     bool
	metricReceived         bool
	receivedMetric         common.Metrics
	wg                     sync.WaitGroup
}

func (r *mockEventReceiver) init() {
	r.wg.Add(1)
}

func (r *mockEventReceiver) wait() {
	r.wg.Wait()
}

func (r *mockEventReceiver) receiveEvents(eventChannel chan common.MetricEvent) {
	for {
		select {
		case event := <-eventChannel:
			switch event.Type {
			case common.Initialize:
				r.metricBatchInitialized = true
			case common.Metric:
				r.metricReceived = true
				r.receivedMetric = event.Metric
			case common.Completed:
				r.metricBatchPublish = true
				r.wg.Done()
				return
			}
		}
	}
}

func Test_MuleEventEmitter(t *testing.T) {
	eventCh := make(chan common.MetricEvent)
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

	emitter := NewMuleEventEmitter(&config.MulesoftConfig{CachePath: "/tmp", UseMonitoringAPI: true}, eventCh, client, instanceCache)

	assert.NotNil(t, emitter)

	eventReceiver := &mockEventReceiver{}
	go emitter.Start()
	eventReceiver.init()
	go eventReceiver.receiveEvents(eventCh)

	eventReceiver.wait()
	assert.True(t, eventReceiver.metricBatchInitialized)
	assert.True(t, eventReceiver.metricReceived)
	assert.True(t, eventReceiver.metricBatchPublish)

	// Should throw an error when the client returns an error
	eventCh = make(chan common.MetricEvent)
	client = &mockAnalyticsClient{
		events: []anypoint.APIMonitoringMetric{},
		err:    fmt.Errorf("failed"),
	}
	emitter = NewMuleEventEmitter(&config.MulesoftConfig{CachePath: "/tmp", UseMonitoringAPI: true}, eventCh, client, instanceCache)
	eventReceiver = &mockEventReceiver{}
	eventReceiver.init()
	go func() {
		err := emitter.Start()
		assert.Equal(t, client.err, err)
	}()
	go eventReceiver.receiveEvents(eventCh)

	eventReceiver.wait()
	assert.True(t, eventReceiver.metricBatchInitialized)
	assert.True(t, eventReceiver.metricBatchPublish)
}

func TestMuleEventEmitterJob(t *testing.T) {
	pollInterval := 1 * time.Second
	ac := &config.AgentConfig{
		CentralConfig: corecfg.NewCentralConfig(corecfg.TraceabilityAgent),
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: pollInterval,
		},
	}

	eventCh := make(chan common.MetricEvent)
	client := &mockAnalyticsClient{
		events: []anypoint.APIMonitoringMetric{},
		err:    nil,
	}
	emitter := NewMuleEventEmitter(&config.MulesoftConfig{CachePath: "/tmp", UseMonitoringAPI: true}, eventCh, client, &mockInstaceCache{})

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
