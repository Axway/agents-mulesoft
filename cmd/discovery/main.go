package main

import (
	"fmt"
	"os"

	"github.com/Axway/agents-mulesoft/pkg/cmd/discovery"
)

func main() {
	if err := discovery.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
