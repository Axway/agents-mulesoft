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

// Agent - mulesoft Beater configuration.
type Agent struct {
	client         beat.Client
	done           chan struct{}
	eventChannel   chan string
	eventProcessor Processor
	mule           MuleEmitter
}

// New creates an instance of mulesoft_traceability_agent.
func New(_ *beat.Beat, _ *common.Config) (beat.Beater, error) {
	agentConfig := config.GetConfig()
	bt := &Agent{
		done:         make(chan struct{}),
		eventChannel: make(chan string),
	}

	var err error
	eventGen := transaction.NewEventGenerator()
	bt.eventProcessor = NewEventProcessor(agentConfig, eventGen, &EventMapper{})
	client := anypoint.NewClient(agentConfig.MulesoftConfig)
	bt.mule, err = NewMuleEventEmitter(agentConfig, bt.eventChannel, client)
	if err != nil {
		return nil, err
	}

	// Validate that all necessary services are up and running. If not, return error
	if hc.RunChecks() != hc.OK {
		return nil, agenterrors.ErrInitServicesNotReady
	}

	return bt, nil
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
