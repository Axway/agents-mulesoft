package agent

import (
	coreagent "github.com/Axway/agent-sdk/pkg/agent"
	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/config"
)

// Agent - mulesoft Beater configuration.
type Agent struct {
	done           chan struct{}
	mule           *MuleEventEmitter
	eventProcessor *EventProcessor
	client         beat.Client
	eventChannel   chan string
}

// New creates an instance of mulesoft_traceability_agent.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	agentConfig := config.GetConfig()
	bt := &Agent{
		done:         make(chan struct{}),
		eventChannel: make(chan string),
	}

	var err error
	bt.eventProcessor = NewEventProcessor(agentConfig)
	bt.mule, err = NewMuleEventEmitter(agentConfig, bt.eventChannel)
	if err != nil {
		return nil, err
	}

	// Validate that all necessary services are up and running. If not, return error
	if hc.RunChecks() != hc.OK {
		return nil, agenterrors.ErrInitServicesNotReady
	}

	return bt, nil
}

// Run starts awsApigwTraceabilityAgent.
func (bt *Agent) Run(b *beat.Beat) error {
	logp.Info("mulesoft_traceability_agent is running! Hit CTRL-C to stop it.")
	coreagent.OnConfigChange(bt.onConfigChange)

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		coreagent.UpdateStatus(coreagent.AgentFailed, err.Error())
		return err
	}

	go bt.mule.Start()

	for {
		select {
		case <-bt.done:
			bt.mule.Stop()
			return nil
		case event := <-bt.eventChannel:
			eventsToPublish := bt.eventProcessor.ProcessRaw([]byte(event))
			bt.client.PublishAll(eventsToPublish)

		}
	}
}

// onConfigChange apply configuation changes
func (bt *Agent) onConfigChange() {
	cfg := config.GetConfig()
	bt.mule.OnConfigChange(cfg)
}

// Stop stops customLogTraceabilityAgent.
func (bt *Agent) Stop() {
	bt.client.Close()
	bt.mule.Stop()
	close(bt.done)
}
