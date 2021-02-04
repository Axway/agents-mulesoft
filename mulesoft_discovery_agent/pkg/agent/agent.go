package agent

import (
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	coreagent "github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	log "github.com/Axway/agent-sdk/pkg/util/log"
	anypoint "github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/anypoint"
	config "github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/config"
)

// Agent links the mulesoft client and the gateway client.
type Agent struct {
	stage               string
	discoveryTags       []string
	discoveryIgnoreTags []string
	anypointClient      anypoint.Client
	apicClient          apic.Client
	pollInterval        time.Duration
	apiChan             chan *ServiceDetail
	stopAgent           chan bool
	stopDiscovery       chan bool
	stopPublish         chan bool
}

// New creates a new agent
func New(anypointClient anypoint.Client) (agent *Agent, err error) {
	cfg := config.GetConfig()

	buffer := 5
	agent = &Agent{
		discoveryTags:       cleanTags(cfg.MulesoftConfig.DiscoveryTags),
		discoveryIgnoreTags: cleanTags(cfg.MulesoftConfig.DiscoveryIgnoreTags),
		apicClient:          coreagent.GetCentralClient(),
		anypointClient:      anypointClient,
		pollInterval:        cfg.MulesoftConfig.PollInterval,
		apiChan:             make(chan *ServiceDetail, buffer),
		stopAgent:           make(chan bool),
		stopDiscovery:       make(chan bool),
		stopPublish:         make(chan bool),
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

	go a.discoveryLoop()
	go a.publishLoop()

	select {
	case <-a.stopAgent:
		log.Info("Received request to kill agent")
		a.stopDiscovery <- true
		a.stopPublish <- true
		return
	}
}

// onConfigChange apply configuation changes
func (a *Agent) onConfigChange() {
	cfg := config.GetConfig()

	a.stage = cfg.MulesoftConfig.Environment
	a.discoveryTags = cleanTags(cfg.MulesoftConfig.DiscoveryTags)
	a.discoveryIgnoreTags = cleanTags(cfg.MulesoftConfig.DiscoveryIgnoreTags)
	a.apicClient = coreagent.GetCentralClient()
	a.anypointClient.OnConfigChange(cfg.MulesoftConfig)
}

func (a *Agent) validateAPI(apiID, stageName string) bool {
	return true
}

func cleanTags(tagCSV string) []string {
	clean := []string{}
	tags := strings.Split(tagCSV, ",")
	for _, v := range tags {
		tag := strings.TrimSpace(strings.ToLower(v))
		if len(tag) > 0 {
			clean = append(clean, tag)
		}
	}
	return clean
}
