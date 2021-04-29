package traceability

import (
	"encoding/json"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

type MuleEmitter interface {
	Start()
	Stop()
	OnConfigChange(gatewayCfg *config.AgentConfig)
}

// MuleEventEmitter - Represents the Gateway client
type MuleEventEmitter struct {
	anypointClient anypoint.Client
	cfg            *config.AgentConfig
	done           chan bool
	eventChannel   chan string
	pollInterval   time.Duration
}

// NewMuleEventEmitter - Creates a new Gateway Client
func NewMuleEventEmitter(
	gatewayCfg *config.AgentConfig,
	eventChannel chan string,
	client anypoint.Client,
) (*MuleEventEmitter, error) {
	return &MuleEventEmitter{
		cfg:            gatewayCfg,
		pollInterval:   gatewayCfg.MulesoftConfig.PollInterval,
		done:           make(chan bool),
		eventChannel:   eventChannel,
		anypointClient: client,
	}, nil
}

// Start - Starts reading log file
func (me *MuleEventEmitter) Start() {
	me.pollForEvents()
}

// pollForEvents - Polls for the events
func (me *MuleEventEmitter) pollForEvents() {
	ticker := time.NewTicker(me.pollInterval)
	go func() {
		for {
			select {
			case <-me.done:
				return
			case <-ticker.C:
				events, err := me.anypointClient.GetAnalyticsWindow()
				if err != nil {
					logp.Warn("Client Failure: %s", err.Error())
				}
				for _, event := range events {
					j, err := json.Marshal(event)
					if err != nil {
						logp.Warn("Marshal Failure: %s", err.Error())
					}
					me.eventChannel <- string(j)
				}
			}
		}
	}()
}

// Stop -
func (me *MuleEventEmitter) Stop() {
	me.done <- true
}

// OnConfigChange -
func (me *MuleEventEmitter) OnConfigChange(gatewayCfg *config.AgentConfig) {
	me.anypointClient.OnConfigChange(gatewayCfg.MulesoftConfig)
}
