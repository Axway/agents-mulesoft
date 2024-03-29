package discovery

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/migrate"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-mulesoft/pkg/common"

	"github.com/Axway/agent-sdk/pkg/agent"

	"github.com/Axway/agent-sdk/pkg/apic"
	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/cmd/service"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	subs "github.com/Axway/agents-mulesoft/pkg/subscription"
	"github.com/sirupsen/logrus"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/discovery"
)

// RootCmd - Agent root command
var (
	RootCmd        corecmd.AgentRootCmd
	client         *anypoint.AnypointClient
	discoveryAgent *discovery.Agent
)

func init() {
	// Create new root command with callbacks to initialize the agent config and command execution.
	// The first parameter identifies the name of the yaml file that agent will look for to load the config
	RootCmd = corecmd.NewRootCmd(
		"mulesoft_discovery_agent", // Name of the yaml file
		"MuleSoft Discovery Agent", // Agent description
		initConfig,                 // Callback for initializing the agent config
		run,                        // Callback for executing the agent
		corecfg.DiscoveryAgent,     // Agent Type (Discovery or Traceability)
	)
	config.AddConfigProperties(RootCmd.GetProperties())

	migrate.MatchAttr(
		common.AttrAPIID,
		common.AttrAssetID,
		common.AttrAssetVersion,
		common.AttrChecksum,
		common.AttrProductVersion,
	)

	RootCmd.AddCommand(service.GenServiceCmd("pathConfig"))
}

// run Callback that agent will call to process the execution
func run() error {
	discoveryAgent.Run()
	return nil
}

// initConfig Callback that agent will call to initialize the config. CentralConfig is parsed by Agent SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {
	conf := &config.AgentConfig{
		CentralConfig:  centralConfig,
		MulesoftConfig: config.NewMulesoftConfig(RootCmd.GetProperties()),
	}

	config.SetConfig(conf)

	if util.IsNotTest() {
		client = anypoint.NewClient(conf.MulesoftConfig)
		muleSubClient := subs.NewMuleSubscriptionClient(client)
		entry := logrus.NewEntry(log.Get())
		var sm *subs.Manager

		if centralConfig.IsMarketplaceSubsEnabled() {
			agent.RegisterProvisioner(subs.NewProvisioner(muleSubClient, entry))
			agent.NewAPIKeyAccessRequestBuilder().Register()
			agent.NewOAuthCredentialRequestBuilder(agent.WithCRDOAuthSecret()).IsRenewable().Register()
		} else {
			var err error
			sm, err = initUCSubscriptionManager(client, agent.GetCentralClient())
			if err != nil {
				return nil, fmt.Errorf("error while initializing the subscription manager %s", err)
			}
		}

		discoveryAgent = discovery.NewAgent(conf, client, sm)
	}
	return conf, nil
}

func initUCSubscriptionManager(apc anypoint.Client, central apic.Client) (*subs.Manager, error) {
	entry := logrus.NewEntry(log.Get())
	muleSubClient := subs.NewMuleSubscriptionClient(apc)
	clientID := subs.NewClientIDContract()
	sm := subs.NewManager(entry, muleSubClient, clientID)

	schema := clientID.Schema()
	err := central.RegisterSubscriptionSchema(schema, true)
	if err != nil {
		return nil, fmt.Errorf("failed to register subscription schema %s: %w", schema.GetSubscriptionName(), err)
	}

	subManager := central.GetSubscriptionManager()

	// register validator and handlers
	subManager.RegisterValidator(sm.ValidateSubscription)
	subManager.RegisterProcessor(apic.SubscriptionApproved, sm.ProcessSubscribe)
	subManager.RegisterProcessor(apic.SubscriptionUnsubscribeInitiated, sm.ProcessUnsubscribe)

	// start polling for subscriptions
	subManager.Start()

	return sm, nil
}
