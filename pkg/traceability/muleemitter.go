package traceability

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

type Emitter interface {
	Start()
	Stop()
	OnConfigChange(gatewayCfg *config.AgentConfig)
}

// MuleEventEmitter - Represents the Gateway client
type MuleEventEmitter struct {
	client       anypoint.AnalyticsClient
	cfg          *config.AgentConfig
	done         chan bool
	eventChannel chan string
	pollInterval time.Duration
}

// NewMuleEventEmitter - Creates a new Gateway Client
func NewMuleEventEmitter(
	gatewayCfg *config.AgentConfig,
	eventChannel chan string,
	client anypoint.AnalyticsClient,
) (*MuleEventEmitter, error) {
	return &MuleEventEmitter{
		cfg:          gatewayCfg,
		pollInterval: gatewayCfg.MulesoftConfig.PollInterval,
		done:         make(chan bool),
		eventChannel: eventChannel,
		client:       client,
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
				oldTime := time.Now()
				events, err := me.client.GetAnalyticsWindow()
				currentTime := time.Now()
				duration := currentTime.Sub(oldTime)
				logrus.WithFields(logrus.Fields{
					"duration": duration * time.Millisecond,
					"count":    len(events),
				}).Debug("retrieved events from anypoint")
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
	me.client.OnConfigChange(gatewayCfg.MulesoftConfig)
}
