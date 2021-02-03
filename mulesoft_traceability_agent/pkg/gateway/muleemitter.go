package gateway

import (
	"encoding/json"
	"github.com/elastic/beats/v7/libbeat/logp"
	"time"

	// CHANGE_HERE - Change the import path(s) below to reference packages correctly
	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/config"
)

// MuleEventEmitter - Represents the Gateway client
type MuleEventEmitter struct {
	cfg          *config.AgentConfig
	anypointClient anypoint.Client
	done           chan bool
	eventChannel chan string
}

// NewMuleEventEmitter - Creates a new Gateway Client
func NewMuleEventEmitter(gatewayCfg *config.AgentConfig, eventChannel chan string) (*MuleEventEmitter, error) {
	return &MuleEventEmitter{
		cfg:          gatewayCfg,
		done: make(chan bool),
		eventChannel: eventChannel,
		anypointClient: anypoint.NewClient(gatewayCfg.MulesoftConfig),
	}, nil
}

// Start - Starts reading log file
func (me *MuleEventEmitter) Start() {
		// Just get a simple payload to test that we can send to condor
/*		sample := GenerateSample()
		payload,_ := json.Marshal(sample)
		me.eventChannel <- string(payload)*/
	me.pollForEvents()
}
// pollForEvents - Polls for the events
func (me *MuleEventEmitter) pollForEvents() {

	ticker := time.NewTicker(60*time.Second)
	go func() {
		for {
			select {
			case <-me.done:
				return
			case <-ticker.C:
				logp.Info("Tick...")
				events, err := me.anypointClient.GetAnalyticsWindow()
				if err != nil {
					logp.Warn("Client Failure", err)
				}
				for _, event := range events {
					j, err := json.Marshal(event)
					if (err != nil) {
						logp.Warn("Marshal Failure", err)
					}
					me.eventChannel <- string(j)
				}

			}
		}
	}()
	/*	t, _ := tail.TailFile(r.cfg.LogFile, tail.Config{Follow: true})
	for line := range t.Lines {
		r.eventChannel <- line.Text
	}*/
}
func (me *MuleEventEmitter) Stop() {
	me.done <- true
}
func (me *MuleEventEmitter) OnConfigChange(gatewayCfg *config.AgentConfig){
	me.anypointClient.OnConfigChange(gatewayCfg.MulesoftConfig)
}
