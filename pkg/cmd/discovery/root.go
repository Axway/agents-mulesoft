package discovery

import (
	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/cmd/service"
	corecfg "github.com/Axway/agent-sdk/pkg/config"

	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/discovery"
)

// RootCmd - Agent root command
var RootCmd corecmd.AgentRootCmd

const mulesoft = "Mulesoft"

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
	discoveryAgent, err := discovery.New(config.GetConfig())
	if err != nil {
		return err
	}
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
	// default the data plane name
	if centralConfig.GetDataPlaneName() == "" {
		centralConfig.SetDataPlaneName(mulesoft)
	}

	conf := &config.AgentConfig{
		CentralConfig:  centralConfig,
		MulesoftConfig: config.NewMulesoftConfig(RootCmd.GetProperties()),
	}

	config.SetConfig(conf)
	return conf, nil
}
