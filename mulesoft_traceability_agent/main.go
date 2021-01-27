package main

import (
	"fmt"
	"os"

	// Required Import to setup factory for traceability transport
	_ "github.com/Axway/agent-sdk/pkg/traceability"

	// CHANGE_HERE - Change the import path(s) below to reference packages correctly
	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
