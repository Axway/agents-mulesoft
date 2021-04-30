package traceability

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"gopkg.in/yaml.v2"

	"github.com/Axway/agent-sdk/pkg/agent"

	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

var agentConfig *config.AgentConfig

const (
	TenantID       = "332211"
	APICDeployment = "prod"
	Environment    = "mule"
	EnvID          = "envid00"
	TeamID         = "678"
)

func init() {
	setupRedaction()
	setupConfig()
}

func TestEventProcessor_ProcessRaw(t *testing.T) {
	processor := NewEventProcessor(agentConfig, &eventGeneratorMock{}, &EventMapper{})

	bts, err := json.Marshal(&event)
	assert.Nil(t, err)
	evts := processor.ProcessRaw(bts)

	summaryRaw := evts[0]
	summaryEvent := &transaction.LogEvent{}
	summaryMsg := summaryRaw.Fields["message"].(string)
	err = json.Unmarshal([]byte(summaryMsg), summaryEvent)
	assert.Nil(t, err)
	// TransactionSummary assertions
	assertLegCommonFields(t, event, summaryEvent, transaction.TypeTransactionSummary)
	assert.Nil(t, summaryEvent.TransactionEvent)
	assert.Equal(t, "Success", summaryEvent.TransactionSummary.Status)
	assert.Equal(t, "200", summaryEvent.TransactionSummary.StatusDetail)
	assert.Equal(t, 60, summaryEvent.TransactionSummary.Duration)
	assert.Equal(t, TeamID, summaryEvent.TransactionSummary.Team.ID)
	assert.Equal(t, transaction.FormatProxyID(event.APIVersionName), summaryEvent.TransactionSummary.Proxy.ID)
	assert.Equal(t, 1, summaryEvent.TransactionSummary.Proxy.Revision)
	assert.Equal(t, event.APIName, summaryEvent.TransactionSummary.Proxy.Name)
	assert.Nil(t, summaryEvent.TransactionSummary.Runtime)
	assert.Equal(t, "http", summaryEvent.TransactionSummary.EntryPoint.Type)
	assert.Equal(t, event.Verb, summaryEvent.TransactionSummary.EntryPoint.Method)
	assert.Equal(t, event.ResourcePath, summaryEvent.TransactionSummary.EntryPoint.Path)
	assert.Equal(t, event.ClientIP, summaryEvent.TransactionSummary.EntryPoint.Host)

	leg0Raw := evts[1]
	leg0Event := &transaction.LogEvent{}
	leg0Msg := leg0Raw.Fields["message"].(string)
	err = json.Unmarshal([]byte(leg0Msg), leg0Event)
	assert.Nil(t, err)
	assertLegCommonFields(t, event, leg0Event, transaction.TypeTransactionEvent)
	assert.Equal(t, FormatLeg0(event.MessageID), leg0Event.TransactionEvent.ID)
	assertLegTransactionEvent(t, event, leg0Event, Inbound, "")

	leg1Raw := evts[2]
	leg1Event := &transaction.LogEvent{}
	leg1Msg := leg1Raw.Fields["message"].(string)
	err = json.Unmarshal([]byte(leg1Msg), leg1Event)
	assert.Nil(t, err)
	assertLegCommonFields(t, event, leg1Event, transaction.TypeTransactionEvent)
	assert.Equal(t, FormatLeg1(event.MessageID), leg1Event.TransactionEvent.ID)
	assertLegTransactionEvent(t, event, leg1Event, Outbound, FormatLeg0(event.MessageID))
}

func TestEventProcessor_ProcessRaw_Errors(t *testing.T) {
	// returns nil when the EventMapper throws an error
	processor := NewEventProcessor(agentConfig, &eventGeneratorMock{}, &eventMapperErr{})
	bts, err := json.Marshal(&event)
	assert.Nil(t, err)
	evts := processor.ProcessRaw(bts)
	assert.Nil(t, evts)

	// returns an empty array when the EventGenerator throws an error
	processor = NewEventProcessor(agentConfig, &eventGenMockErr{}, &EventMapper{})
	bts, err = json.Marshal(&event)
	assert.Nil(t, err)
	evts = processor.ProcessRaw(bts)
	assert.Equal(t, 0, len(evts))

	// return nil when given bad json
	processor = NewEventProcessor(agentConfig, &eventGeneratorMock{}, &EventMapper{})
	evts = processor.ProcessRaw([]byte("nope"))
	assert.Nil(t, evts)
}

func assertLegCommonFields(t *testing.T, muleEvent anypoint.AnalyticsEvent, logEvent *transaction.LogEvent, logType string) {
	assert.Equal(t, "1.0", logEvent.Version)
	assert.Equal(t, FormatTxnId(muleEvent.APIVersionID, muleEvent.MessageID), logEvent.TransactionID)
	assert.Equal(t, "", logEvent.Environment)
	assert.Equal(t, APICDeployment, logEvent.APICDeployment)
	assert.Equal(t, EnvID, logEvent.EnvironmentID)
	assert.Equal(t, TenantID, logEvent.TenantID)
	assert.Equal(t, TenantID, logEvent.TrcbltPartitionID)
	assert.Equal(t, logType, logEvent.Type)
	assert.Equal(t, "", logEvent.TargetPath)
	assert.Equal(t, "", logEvent.ResourcePath)
}

func assertLegTransactionEvent(t *testing.T, muleEvent anypoint.AnalyticsEvent, logEvent *transaction.LogEvent, direction, parent string) {
	assert.Nil(t, logEvent.TransactionSummary)
	assert.Equal(t, parent, logEvent.TransactionEvent.ParentID)
	assert.Equal(t, muleEvent.ClientIP+":0", logEvent.TransactionEvent.Source)
	assert.Equal(t, muleEvent.APIName, logEvent.TransactionEvent.Destination)
	assert.Equal(t, 0, logEvent.TransactionEvent.Duration)
	assert.Equal(t, direction, logEvent.TransactionEvent.Direction)
	assert.Equal(t, "Pass", logEvent.TransactionEvent.Status)
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

func setupConfig() {
	agentConfig = &config.AgentConfig{
		CentralConfig: &corecfg.CentralConfiguration{
			CentralConfig:    nil,
			IConfigValidator: nil,
			AgentType:        corecfg.TraceabilityAgent,
			TenantID:         TenantID,
			APICDeployment:   APICDeployment,
			Environment:      Environment,
		},
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: 1 * time.Second,
		},
	}
	agentConfig.CentralConfig.SetEnvironmentID(EnvID)
	agentConfig.CentralConfig.SetTeamID(TeamID)
	agent.Initialize(agentConfig.CentralConfig)
}

type eventGenMockErr struct{}

func (c *eventGenMockErr) CreateEvent(
	_ transaction.LogEvent,
	_ time.Time,
	_ common.MapStr,
	_ common.MapStr,
	_ interface{},
) (event beat.Event, err error) {
	return beat.Event{}, fmt.Errorf("create event error")
}

type eventMapperErr struct{}

func (em *eventMapperErr) ProcessMapping(_ anypoint.AnalyticsEvent) ([]*transaction.LogEvent, error) {
	return nil, fmt.Errorf("event mapping error")
}
