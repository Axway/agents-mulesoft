package main

import (
	"fmt"
	"os"

	"github.com/Axway/agents-mulesoft/mulesoft_discovery_agent/pkg/cmd/discovery"
)

func main() {
	if err := discovery.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
