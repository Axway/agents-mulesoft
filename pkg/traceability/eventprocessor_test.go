package traceability

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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

	processor := NewEventProcessor(ac, &eventGeneratorMock{}, &EventMapper{})
	events := []publisher.Event{
		{
			Content: beat.Event{
				Timestamp: time.Time{},
				Fields: map[string]interface{}{
					"message": `{
						"API ID": "211799904",
						"API Name": "petstore-3",
						"API Version ID": "16810512",
						"API Version Name": "v1",
						"Application Name": "",
						"Client IP": "1.2.3.4",
						"Continent": "North America",
						"Country": "United States",
						"Message ID": "e2029ea0-a873-11eb-875c-064449f4dd2c",
						"Request Outcome": "PROCESSED",
						"Request Size": 0,
						"Resource Path": "/pets",
						"Response Size": 18,
						"Response Time": 58,
						"Status Code": 200,
						"User Agent Name": "Mozilla",
						"User Agent Version": "5.0",
						"Verb": "GET"
					}`,
				},
				Private:    nil,
				TimeSeries: false,
			},
		},
	}
	processedEvents := processor.Process(events)
	assert.Equal(t, 3, len(processedEvents))
	summaryJSON := processedEvents[0].Content.Fields["message"]
	s := summaryJSON.(string)
	summaryLogEvent := &transaction.LogEvent{}
	json.Unmarshal([]byte(s), summaryLogEvent)
	// leg0 := processedEvents[1]
	// leg1 := processedEvents[2]

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

// eventGeneratorMock - mock event generator
type eventGeneratorMock struct{}

// CreateEvent - Creates a new mocked event for tests
func (c *eventGeneratorMock) CreateEvent(
	logEvent transaction.LogEvent,
	eventTime time.Time,
	metaData common.MapStr,
	_ common.MapStr,
	privateData interface{},
) (event beat.Event, err error) {
	serializedLogEvent, _ := json.Marshal(logEvent)
	eventData := make(map[string]interface{})
	eventData["message"] = string(serializedLogEvent)
	event = beat.Event{
		Timestamp: eventTime,
		Meta:      metaData,
		Private:   privateData,
		Fields:    eventData,
	}
	return
}
