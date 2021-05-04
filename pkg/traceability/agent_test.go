package traceability

import (
	"testing"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/elastic/beats/v7/libbeat/beat"
	publisher "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/stretchr/testify/assert"
)

func TestAgent_Run(t *testing.T) {
	processorChannel := make(chan bool)
	agent, err := newMockAgent(processorChannel)
	assert.Nil(t, err)
	assert.NotNil(t, agent)
	client := publisher.NewChanClientWith(make(chan beat.Event))
	pub := publisher.PublisherWithClient(client)
	b := &beat.Beat{
		Publisher: pub,
	}

	go agent.Run(b)

	done := <-processorChannel
	assert.True(t, done)
	agent.Stop()
}

type mockAnalyticsClient struct{}

func (m mockAnalyticsClient) GetAnalyticsWindow() ([]anypoint.AnalyticsEvent, error) {
	return []anypoint.AnalyticsEvent{event}, nil
}

func (m mockAnalyticsClient) OnConfigChange(_ *config.MulesoftConfig) {
}

type mockProcessor struct {
	channel chan bool
}

func (m mockProcessor) ProcessRaw(_ []byte) []beat.Event {
	m.channel <- true
	return []beat.Event{}
}

func newMockAgent(processorChannel chan bool) (*Agent, error) {
	eventChannel := make(chan string)
	client := &mockAnalyticsClient{}
	processor := &mockProcessor{
		channel: processorChannel,
	}
	emitter, err := NewMuleEventEmitter(agentConfig, eventChannel, client)
	if err != nil {
		return nil, err
	}
	return newAgent(processor, emitter, eventChannel)
}
