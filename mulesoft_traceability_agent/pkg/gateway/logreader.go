package gateway

import (
	"github.com/hpcloud/tail"

	// CHANGE_HERE - Change the import path(s) below to reference packages correctly
	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/config"
)

// LogReader - Represents the Gateway client
type LogReader struct {
	cfg          *config.GatewayConfig
	eventChannel chan string
}

// NewLogReader - Creates a new Gateway Client
func NewLogReader(gatewayCfg *config.GatewayConfig, eventChannel chan string) (*LogReader, error) {
	return &LogReader{
		cfg:          gatewayCfg,
		eventChannel: eventChannel,
	}, nil
}

// Start - Starts reading log file
func (r *LogReader) Start() {
	go r.tailFile()
}

func (r LogReader) tailFile() {
	t, _ := tail.TailFile(r.cfg.LogFile, tail.Config{Follow: true})
	for line := range t.Lines {
		r.eventChannel <- line.Text
	}
}
