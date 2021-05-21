package discovery

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Axway/agents-mulesoft/pkg/subscription"

	coreAgent "github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cache"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

type Repeater interface {
	Loop()
	OnConfigChange(cfg *config.MulesoftConfig)
	Stop()
}

// Agent links the mulesoft client and the gateway client.
type Agent struct {
	client    anypoint.Client
	stopAgent chan bool
	discovery Repeater
	publisher Repeater
}

// NewAgent creates a new agent
func NewAgent(cfg *config.AgentConfig, client anypoint.Client, sm subscription.SchemaHandler) (agent *Agent) {
	buffer := 5
	apiChan := make(chan *ServiceDetail, buffer)

	pub := &publisher{
		apiChan:     apiChan,
		stopPublish: make(chan bool),
		publishAPI:  coreAgent.PublishAPI,
	}

	assetCache := cache.GetCache()

	svcHandler := &serviceHandler{
		muleEnv:             cfg.MulesoftConfig.Environment,
		discoveryTags:       cleanTags(cfg.MulesoftConfig.DiscoveryTags),
		discoveryIgnoreTags: cleanTags(cfg.MulesoftConfig.DiscoveryIgnoreTags),
		client:              client,
		subscriptionManager: sm,
		cache:               assetCache,
	}

	disc := &discovery{
		apiChan:           apiChan,
		cache:             assetCache,
		client:            client,
		centralClient:     coreAgent.GetCentralClient(),
		discoveryPageSize: 50,
		pollInterval:      cfg.MulesoftConfig.PollInterval,
		stopDiscovery:     make(chan bool),
		serviceHandler:    svcHandler,
	}

	return newAgent(client, disc, pub)
}

func newAgent(
	client anypoint.Client,
	discovery Repeater,
	publisher Repeater,
) *Agent {
	return &Agent{
		client:    client,
		discovery: discovery,
		publisher: publisher,
		stopAgent: make(chan bool),
	}
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
	go a.discovery.Loop()
	go a.publisher.Loop()
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
	coreAgent.OnConfigChange(a.onConfigChange)

	go a.discovery.Loop()
	go a.publisher.Loop()

	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, os.Interrupt)

	select {
	case <-gracefulStop:
		a.Stop()
		return
	}
}

// Stop stops the discovery agent.
func (a *Agent) Stop() {
	a.discovery.Stop()
	a.publisher.Stop()
	close(a.stopAgent)
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
