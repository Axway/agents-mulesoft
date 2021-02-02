package agent

import (
	"github.com/Axway/agent-sdk/pkg/agent"
	coreagent "github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/filter"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	log "github.com/Axway/agent-sdk/pkg/util/log"
	anypoint "github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/anypoint"
	config "github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/config"
)

// Agent links the mulesoft client and the gateway client.
type Agent struct {
	discoveryIgnoreTags string
	discoveryFilter     filter.Filter
	anypointClient      anypoint.Client
	apicClient          apic.Client
	stopChan            chan struct{}
}

// New creates a new agent
func New(anypointClient anypoint.Client) (agent *Agent, err error) {
	cfg := config.GetConfig()

	discoveryFilter, err := filter.NewFilter(cfg.MulesoftConfig.Filter)
	if err != nil {
		return nil, err
	}

	agent = &Agent{
		discoveryIgnoreTags: cfg.MulesoftConfig.DiscoveryIgnoreTags,
		apicClient:          coreagent.GetCentralClient(),
		anypointClient:      anypointClient,
		discoveryFilter:     discoveryFilter,
		stopChan:            make(chan struct{}),
	}

	if anypointClient == nil {
		agent.anypointClient = anypoint.NewClient(cfg.MulesoftConfig)
	}
	return agent, nil
}

// CheckHealth - check the health of all clients associated with the agent
func (a *Agent) CheckHealth() error {
	if hc.RunChecks() != hc.OK {
		return utilErrors.ErrInitServicesNotReady
	}
	return nil
}

// Run the agent loop
func (a *Agent) Run() {

	agent.RegisterAPIValidator(a.validateAPI)
	agent.OnConfigChange(a.onConfigChange)

	// TODO - listen to mulesoft
	assets, err := a.anypointClient.ListAssets(&anypoint.Page{Offset: 0, PageSize: 20})
	if err != nil {
		log.Error(err)
	}
	log.Infof("%+v", assets)

	select {
	case <-a.stopChan:
		log.Info("Received request to kill agent")
		return
	}
}

// onConfigChange apply configuation changes
func (a *Agent) onConfigChange() {
	cfg := config.GetConfig()

	discoveryFilter, err := filter.NewFilter(cfg.MulesoftConfig.Filter)
	if err != nil {
		log.Error(err)
	}

	a.discoveryFilter = discoveryFilter
	a.discoveryIgnoreTags = cfg.MulesoftConfig.DiscoveryIgnoreTags
	a.apicClient = coreagent.GetCentralClient()
	a.anypointClient.OnConfigChange(cfg.MulesoftConfig)
}

func (a *Agent) validateAPI(apiID, stageName string) bool {
	// TODO
	return true
}
