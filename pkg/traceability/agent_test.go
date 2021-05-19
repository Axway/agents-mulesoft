package traceability

import (
	"testing"
	"time"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	corecfg "github.com/Axway/agent-sdk/pkg/config"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/elastic/beats/v7/libbeat/beat"
	publisher "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/stretchr/testify/assert"
)

func TestAgent_Run(t *testing.T) {
	processorChannel := make(chan bool)
	eventChannel := make(chan string)

	processor := &mockProcessor{
		channel: processorChannel,
	}

	client := &mockAnalyticsClient{
		events: []anypoint.AnalyticsEvent{event},
		err:    nil,
	}
	emitter := NewMuleEventEmitter(eventChannel, client)
	traceAgent, err := newAgent(processor, emitter, eventChannel)

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

func Test_newAgentError(t *testing.T) {
	// should return an error when the health check fails
	processorChannel := make(chan bool)
	eventChannel := make(chan string)

	processor := &mockProcessor{
		channel: processorChannel,
	}

	client := &mockAnalyticsClient{
		events: []anypoint.AnalyticsEvent{event},
		err:    nil,
	}
	emitter := NewMuleEventEmitter(eventChannel, client)
	hc.RegisterHealthcheck("fake", "fake", func(name string) *hc.Status {
		return &hc.Status{
			Result: hc.FAIL,
		}
	})
	_, err := newAgent(processor, emitter, eventChannel)
	assert.NotNil(t, err)

}

type mockAnalyticsClient struct {
	events []anypoint.AnalyticsEvent
	app    *anypoint.Application
	err    error
}

func (m mockAnalyticsClient) GetAnalyticsWindow() ([]anypoint.AnalyticsEvent, error) {
	return m.events, m.err
}

func (m mockAnalyticsClient) GetClientApplication(string) (*anypoint.Application, error) {
	return m.app, m.err
}

func (m mockAnalyticsClient) OnConfigChange(_ *config.MulesoftConfig) {
}
func (m mockAnalyticsClient) GetLastRun () (string,string){
	return "2021-05-18T23:23:40Z",""
}
func (m mockAnalyticsClient) SaveLastRun (string) (){
}
type mockProcessor struct {
	channel chan bool
}

func (m mockProcessor) ProcessRaw(_ []byte) []beat.Event {
	m.channel <- true
	return []beat.Event{}
}
