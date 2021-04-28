package discovery

import (
	"fmt"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	coreagent "github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cache"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

// Agent links the mulesoft client and the gateway client.
type Agent struct {
	anypointClient      anypoint.Client
	apicClient          apic.Client
	apiChan             chan *ServiceDetail
	assetCache          cache.Cache
	discoveryIgnoreTags []string
	discoveryPageSize   int
	discoveryTags       []string
	pollInterval        time.Duration
	publishBufferSize   int
	stage               string
	stopAgent           chan bool
	stopDiscovery       chan bool
	stopPublish         chan bool
}

// New creates a new agent
func New(cfg *config.AgentConfig, client anypoint.Client) (agent *Agent) {
	buffer := 5
	assetCache := cache.New()
	agent = &Agent{
		anypointClient:      client,
		apicClient:          coreagent.GetCentralClient(),
		apiChan:             make(chan *ServiceDetail, buffer),
		assetCache:          assetCache,
		discoveryIgnoreTags: cleanTags(cfg.MulesoftConfig.DiscoveryIgnoreTags),
		discoveryPageSize:   50,
		discoveryTags:       cleanTags(cfg.MulesoftConfig.DiscoveryTags),
		pollInterval:        cfg.MulesoftConfig.PollInterval,
		stage:               cfg.MulesoftConfig.Environment,
		stopAgent:           make(chan bool),
		stopDiscovery:       make(chan bool),
		stopPublish:         make(chan bool),
	}

	return agent
}

// onConfigChange apply configuration changes
func (a *Agent) onConfigChange() {
	cfg := config.GetConfig()

	// Stop Discovery & Publish
	a.stopDiscovery <- true
	a.stopPublish <- true

	a.stage = cfg.MulesoftConfig.Environment
	a.discoveryTags = cleanTags(cfg.MulesoftConfig.DiscoveryTags)
	a.discoveryIgnoreTags = cleanTags(cfg.MulesoftConfig.DiscoveryIgnoreTags)
	a.apicClient = coreagent.GetCentralClient()
	a.pollInterval = cfg.MulesoftConfig.PollInterval
	a.anypointClient.OnConfigChange(cfg.MulesoftConfig)

	// Restart Discovery & Publish
	go a.discoveryLoop()
	go a.publishLoop()
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

// validateAPI checks that the API still exists on the dataplane. If it doesn't the agent
// performs cleanup on the API Central environment. The asset cache is populated by the
// discovery loop.
func (a *Agent) validateAPI(apiID, stageName string) bool {
	asset, err := a.assetCache.Get(formatCacheKey(apiID, stageName))
	if err != nil {
		log.Warnf("Unable to validate API: %s", err.Error())
		// If we can't validate it exists then assume it does until known otherwise.
		return true
	}
	return asset != nil
}

// cleanTags splits the CSV and trims off whitespace
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

// formatCacheKey ensure consistent naming of asset cache key
func formatCacheKey(apiID, stageName string) string {
	return fmt.Sprintf("%s-%s", apiID, stageName)
}
