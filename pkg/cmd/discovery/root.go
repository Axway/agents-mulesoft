package discovery

import (
	"fmt"

	"github.com/Axway/agents-mulesoft/pkg/subscription/clientid"

	"github.com/Axway/agent-sdk/pkg/agent"

	"github.com/Axway/agent-sdk/pkg/apic"
	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/cmd/service"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-mulesoft/pkg/subscription"
	"github.com/sirupsen/logrus"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/discovery"
)

// RootCmd - Agent root command
var RootCmd corecmd.AgentRootCmd

func init() {
	// Create new root command with callbacks to initialize the agent config and command execution.
	// The first parameter identifies the name of the yaml file that agent will look for to load the config
	RootCmd = corecmd.NewRootCmd(
		"mulesoft_discovery_agent", // Name of the yaml file
		"Mulesoft Discovery Agent", // Agent description
		initConfig,                 // Callback for initializing the agent config
		run,                        // Callback for executing the agent
		corecfg.DiscoveryAgent,     // Agent Type (Discovery or Traceability)
	)
	config.AddConfigProperties(RootCmd.GetProperties())
	RootCmd.AddCommand(service.GenServiceCmd("pathConfig"))
}

// run Callback that agent will call to process the execution
func run() error {
	cfg := config.GetConfig()
	client := anypoint.NewClient(cfg.MulesoftConfig)
	sm, err := initSubscriptionManager(client, agent.GetCentralClient())
	if err != nil {
		return fmt.Errorf("Error while initing subscription manager %s", err)
	}

	discoveryAgent := discovery.NewAgent(cfg, client, sm)
	err = discoveryAgent.CheckHealth()
	if err != nil {
		return err
	}

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
	return conf, nil
}

func initSubscriptionManager(apc anypoint.Client, centralClient apic.Client) (*subscription.Manager, error) {
	subManager := centralClient.GetSubscriptionManager()
	sm := subscription.New(logrus.StandardLogger(), centralClient, clientid.NewClientIDContract(apc))

	// register schemas
	for _, schema := range sm.Schemas() {
		if err := centralClient.RegisterSubscriptionSchema(schema, true); err != nil {
			return nil, fmt.Errorf("failed to register subscription schema %s: %w", schema.GetSubscriptionName(), err)
		}
		log.Infof("Schema registered: %s", schema.GetSubscriptionName())
	}

	// register validator and handlers
	subManager.RegisterValidator(sm.ValidateSubscription)
	subManager.RegisterProcessor(apic.SubscriptionApproved, sm.ProcessSubscribe)
	subManager.RegisterProcessor(apic.SubscriptionUnsubscribeInitiated, sm.ProcessUnsubscribe)

	// start polling for subscriptions
	subManager.Start()

	return sm, nil
}
