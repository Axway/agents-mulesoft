package cmd

import (
	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/cmd/service"
	corecfg "github.com/Axway/agent-sdk/pkg/config"

	libcmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"

	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/agent"
	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/config"
)

// RootCmd - Agent root command
var RootCmd corecmd.AgentRootCmd
var beatCmd *libcmd.BeatsRootCmd

func init() {
	name := "mulesoft_traceability_agent"
	settings := instance.Settings{
		Name:          name,
		HasDashboards: true,
	}

	// Initialize the beat command
	beatCmd = libcmd.GenRootCmdWithSettings(agent.New, settings)
	cmd := beatCmd.Command
	// Wrap the beat command with the agent command processor with callbacks to initialize the agent config and command execution.
	// The first parameter identifies the name of the yaml file that agent will look for to load the config
	RootCmd = corecmd.NewCmd(
		&cmd,
		name,                          // Name of the agent and yaml config file
		"Mulesoft Traceability Agent", // Agent description
		initConfig,                    // Callback for initializing the agent config
		run,                           // Callback for executing the agent
		corecfg.TraceabilityAgent,     // Agent Type (Discovery or Traceability)
	)
	RootCmd.AddCommand(service.GenServiceCmd("pathConfig"))

	config.AddMulesoftConfigProperties(RootCmd.GetProperties())

}

// Callback that agent will call to process the execution
func run() error {
	return beatCmd.Execute()
}

// Callback that agent will call to initialize the config. CentralConfig is parsed by Agent SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {

	agentConfig := &config.AgentConfig{
		CentralConfig:  centralConfig,
		MulesoftConfig: config.ParseMulesoftConfig(RootCmd.GetProperties()),
	}
	config.SetConfig(agentConfig)
	return agentConfig, nil
}
