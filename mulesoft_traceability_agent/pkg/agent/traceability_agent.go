package agent

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/anypoint"
	coreagent "github.com/Axway/agent-sdk/pkg/agent"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/config"
	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/gateway"
)

// customLogBeater configuration.
type Agent struct {
	done           chan struct{}
	logReader      *gateway.LogReader
	eventProcessor *gateway.EventProcessor
	client         beat.Client
	anypointClient anypoint.Client
	// Not sure if we need this
	apicClient     apic.Client

	eventChannel   chan string
}

var bt *Agent

// New creates an instance of mule_anypoint_traceability_agent.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	agentConfig:=config.GetConfig()
	bt := &Agent{
		done:         make(chan struct{}),
		eventChannel: make(chan string),
	}

	bt.eventProcessor = gateway.NewEventProcessor(agentConfig)
	bt.anypointClient = anypoint.NewClient(agentConfig.MulesoftConfig)
	var err error
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
	logp.Info("mule_anypoint_traceability_agent is running! Hit CTRL-C to stop it.")
	coreagent.OnConfigChange(bt.onConfigChange)

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		coreagent.UpdateStatus(coreagent.AgentFailed, err.Error())
		return err
	}

	for {
		select {
		case <-bt.done:
			return nil
		case event := <-bt.eventProcessor.GetEventChannel():
			bt.client.Publish(event)

		}
	}
}
// onConfigChange apply configuation changes
func (bt *Agent) onConfigChange() {
	cfg := config.GetConfig()

	bt.apicClient = coreagent.GetCentralClient()
	bt.anypointClient.OnConfigChange(cfg.MulesoftConfig)
}


// Stop stops customLogTraceabilityAgent.
func (bt *Agent) Stop() {
	bt.client.Close()
	close(bt.done)
}
