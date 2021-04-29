package traceability

import (
	coreagent "github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/transaction"
	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

// Agent - mulesoft Beater configuration. Implements the beat.Beater interface.
type Agent struct {
	client         beat.Client
	done           chan struct{}
	eventChannel   chan string
	eventProcessor Processor
	mule           Emitter
}

// NewBeater creates an instance of mulesoft_traceability_agent.
func NewBeater(_ *beat.Beat, _ *common.Config) (beat.Beater, error) {
	eventChannel := make(chan string)
	agentConfig := config.GetConfig()

	var err error
	generator := transaction.NewEventGenerator()
	processor := NewEventProcessor(agentConfig, generator, &EventMapper{})
	client := anypoint.NewClient(agentConfig.MulesoftConfig)
	emitter, err := NewMuleEventEmitter(agentConfig, eventChannel, client)
	if err != nil {
		return nil, err
	}

	return newAgent(processor, emitter, eventChannel)
}

func newAgent(
	processor Processor,
	emitter Emitter,
	eventChannel chan string,
) (*Agent, error) {
	a := &Agent{
		done:           make(chan struct{}),
		eventChannel:   eventChannel,
		eventProcessor: processor,
		mule:           emitter,
	}

	// Validate that all necessary services are up and running. If not, return error
	if hc.RunChecks() != hc.OK {
		return nil, agenterrors.ErrInitServicesNotReady
	}

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

	for {
		select {
		case <-a.done:
			a.mule.Stop()
			return nil
		case event := <-a.eventChannel:
			eventsToPublish := a.eventProcessor.ProcessRaw([]byte(event))
			a.client.PublishAll(eventsToPublish)
		}
	}
}

// onConfigChange apply configuration changes
func (a *Agent) onConfigChange() {
	cfg := config.GetConfig()
	a.mule.OnConfigChange(cfg)
}

// Stop stops customLogTraceabilityAgent.
func (a *Agent) Stop() {
	a.client.Close()
	a.mule.Stop()
	close(a.done)
}
