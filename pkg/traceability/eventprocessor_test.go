package traceability

import (
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"gopkg.in/yaml.v2"

	"github.com/Axway/agent-sdk/pkg/agent"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

func Test_EventProcessor(t *testing.T) {
	setupRedaction()
	assetCache := agent.GetAPICache()
	assetCache.Set("111", "{}")
	ac := &config.AgentConfig{
		CentralConfig: &corecfg.CentralConfiguration{
			CentralConfig:    nil,
			IConfigValidator: nil,
			AgentType:        corecfg.TraceabilityAgent,
			TenantID:         "332211",
			APICDeployment:   "prod",
			Environment:      "mule",
		},
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: 1 * time.Second,
		},
	}
	ac.CentralConfig.SetEnvironmentID("envid00")
	agent.Initialize(ac.CentralConfig)

	processor := NewEventProcessor(ac, &eventGenerator{})
	events := []publisher.Event{
		{
			Content: beat.Event{
				Timestamp: time.Time{},
				Fields: map[string]interface{}{
					"message": `{
						"Client IP": "1.2.3.4",
						"API ID": "",
						"API Name": "fake",
						"API Version ID": "111",
						"API Version Name": "version",
						"Application Name": "app1",
						"Message ID": "222",
						"Request Outcome": "outcome",
						"Verb": "GET",
						"Resource Path": "/pets",
						"Status Code": 200,
						"User Agent Name": "mulesoft",
						"User Agent Version": "1.0",
						"Request Size": 1000
					}`,
				},
				Private:    nil,
				TimeSeries: false,
			},
		},
	}
	processedEvents := processor.Process(events)
	logrus.Infof("Events: %+v", processedEvents)
}

func setupRedaction() {
	redactionCfg := `
path:
  show:
    - keyMatch: ".*"
queryArgument:
  show: 
    - keyMatch: ".*"
requestHeader:
  show: 
    - keyMatch: ".*"
responseHeader:
  show: 
    - keyMatch: ".*"
`
	var allowAllRedaction redaction.Config
	yaml.Unmarshal([]byte(redactionCfg), &allowAllRedaction)
	redaction.SetupGlobalRedaction(allowAllRedaction)
}

type eventGenerator struct{}

func (eg *eventGenerator) CreateEvent(
	logEvent transaction.LogEvent,
	eventTime time.Time,
	metaData common.MapStr,
	fields common.MapStr,
	privateData interface{},
) (event beat.Event, err error) {
	logrus.Infof("Event: %+v", eg)
	return beat.Event{}, nil
}
