package discovery

import (
	"fmt"
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cache"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

// Agent links the mulesoft client and the gateway client.
type Agent struct {
	client     anypoint.Client
	assetCache cache.Cache
	stopAgent  chan bool
	discovery  APIDiscovery
	publisher  APIPublisher
}

// New creates a new agent
func New(cfg *config.AgentConfig, client anypoint.Client) (agent *Agent) {
	buffer := 5
	assetCache := cache.New()
	apiChan := make(chan *ServiceDetail, buffer)

	pub := &publisher{
		apiChan:     apiChan,
		stopPublish: make(chan bool),
	}

	disc := &discovery{
		apiChan:             apiChan,
		assetCache:          assetCache,
		client:              client,
		discoveryIgnoreTags: cleanTags(cfg.MulesoftConfig.DiscoveryIgnoreTags),
		discoveryPageSize:   50,
		discoveryTags:       cleanTags(cfg.MulesoftConfig.DiscoveryTags),
		pollInterval:        cfg.MulesoftConfig.PollInterval,
		stage:               cfg.MulesoftConfig.Environment,
		stopDiscovery:       make(chan bool),
	}

	agent = &Agent{
		assetCache: assetCache,
		discovery:  disc,
		publisher:  pub,
		stopAgent:  make(chan bool),
	}

	return agent
}

// onConfigChange apply configuration changes
func (a *Agent) onConfigChange() {
	cfg := config.GetConfig()

	// Stop Discovery & Publish
	a.discovery.Stop()
	a.publisher.Stop()

	a.client.OnConfigChange(cfg.MulesoftConfig)
	a.discovery.OnConfigChange(cfg.MulesoftConfig)
	// Restart Discovery & Publish
	go a.discovery.DiscoveryLoop()
	go a.publisher.PublishLoop()
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

	go a.discovery.DiscoveryLoop()
	go a.publisher.PublishLoop()

	select {
	case <-a.stopAgent:
		log.Info("Received request to kill agent")
		a.discovery.Stop()
		a.publisher.Stop()
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
