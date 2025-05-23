package traceability

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/cmd/service"
	corecfg "github.com/Axway/agent-sdk/pkg/config"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	libcmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"

	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/traceability"
)

// RootCmd - Agent root command
var RootCmd corecmd.AgentRootCmd
var beatCmd *libcmd.BeatsRootCmd

func init() {
	name := "mulesoft_traceability_agent"
	settings := instance.Settings{
		Name:            name,
		HasDashboards:   true,
		ConfigOverrides: corecfg.LogConfigOverrides(),
	}

	// Initialize the beat command
	beatCmd = libcmd.GenRootCmdWithSettings(traceability.NewBeater, settings)
	cmd := beatCmd.Command
	// Wrap the beat command with the agent command processor with callbacks to initialize the agent config and command execution.
	// The first parameter identifies the name of the yaml file that agent will look for to load the config
	RootCmd = corecmd.NewCmd(
		&cmd,
		name, // Name of the agent and yaml config file
		"Mulesoft Traceability Agent",
		initConfig,
		run,
		corecfg.TraceabilityAgent,
	)

	// set the dataplane type that will be added to the agent spec
	corecfg.AgentDataPlaneType = apic.Mulesoft.String()

	config.AddConfigProperties(RootCmd.GetProperties(), true)
	RootCmd.AddCommand(service.GenServiceCmd("pathConfig"))
}

// Callback that agent will call to process the execution
func run() error {
	return beatCmd.Execute()
}

// Callback that agent will call to initialize the config. CentralConfig is parsed by Agent SDK
// and passed to the callback allowing the agent code to access the central config
func initConfig(centralConfig corecfg.CentralConfig) (interface{}, error) {
	err := centralConfig.SetWatchResourceFilters([]corecfg.ResourceFilter{
		{
			Group:            management.CredentialGVK().Group,
			Kind:             management.CredentialGVK().Kind,
			Name:             "*",
			IsCachedResource: true,
		},
	})
	if err != nil {
		return nil, err
	}

	agentConfig := &config.AgentConfig{
		CentralConfig:  centralConfig,
		MulesoftConfig: config.NewMulesoftConfig(RootCmd.GetProperties()),
	}
	config.SetConfig(agentConfig)
	return agentConfig, nil
}
