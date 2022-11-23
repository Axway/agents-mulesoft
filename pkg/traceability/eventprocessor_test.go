package traceability

import (
	"encoding/json"
	"fmt"
	"os"
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
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
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

func TestEventProcessor_ProcessRaw(t *testing.T) {
	client := &mockAnalyticsClient{
		app: app,
	}
	mapper := &EventMapper{client: client}
	processor := NewEventProcessor(agentConfig, &eventGeneratorMock{}, mapper)

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
	assert.Equal(t, transutil.FormatProxyID(event.APIID), summaryEvent.TransactionSummary.Proxy.ID)
	assert.Equal(t, 1, summaryEvent.TransactionSummary.Proxy.Revision)
	assert.Equal(t, FormatAPIName(event.APIName, event.APIVersionName), summaryEvent.TransactionSummary.Proxy.Name)
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
	assertLegTransactionEvent(t, event, leg0Event, Outbound, "")

	leg1Raw := evts[2]
	leg1Event := &transaction.LogEvent{}
	leg1Msg := leg1Raw.Fields["message"].(string)
	err = json.Unmarshal([]byte(leg1Msg), leg1Event)
	assert.Nil(t, err)
	assertLegCommonFields(t, event, leg1Event, transaction.TypeTransactionEvent)
	assert.Equal(t, FormatLeg1(event.MessageID), leg1Event.TransactionEvent.ID)
	assertLegTransactionEvent(t, event, leg1Event, Inbound, FormatLeg0(event.MessageID))
}

func TestEventProcessor_ProcessRaw_Errors(t *testing.T) {
	// returns nil when the EventMapper throws an error
	processor := NewEventProcessor(agentConfig, &eventGeneratorMock{}, &eventMapperErr{})
	bts, err := json.Marshal(&event)
	assert.Nil(t, err)
	evts := processor.ProcessRaw(bts)
	assert.Nil(t, evts)

	// returns an empty array when the EventGenerator throws an error
	client := &mockAnalyticsClient{
		app: app,
	}
	mapper := &EventMapper{client: client}
	processor = NewEventProcessor(agentConfig, &eventGenMockErr{}, mapper)
	bts, err = json.Marshal(&event)
	assert.Nil(t, err)
	evts = processor.ProcessRaw(bts)
	assert.Equal(t, 0, len(evts))

	// return nil when given bad json
	processor = NewEventProcessor(agentConfig, &eventGeneratorMock{}, mapper)
	evts = processor.ProcessRaw([]byte("nope"))
	assert.Nil(t, evts)
}

func assertLegCommonFields(t *testing.T, muleEvent anypoint.AnalyticsEvent, logEvent *transaction.LogEvent, logType string) {
	assert.Equal(t, "1.0", logEvent.Version)
	assert.Equal(t, FormatTxnID(muleEvent.APIVersionID, muleEvent.MessageID), logEvent.TransactionID)
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
	source := ""
	destination := ""
	if direction == Outbound {
		source = Client
		destination = MuleProxy
	} else {
		source = MuleProxy
		destination = Backend + muleEvent.APIName
	}
	assert.Nil(t, logEvent.TransactionSummary)
	assert.Equal(t, parent, logEvent.TransactionEvent.ParentID)
	assert.Equal(t, source, logEvent.TransactionEvent.Source)
	assert.Equal(t, destination, logEvent.TransactionEvent.Destination)
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
type eventGeneratorMock struct {
	shouldUseTrafficForAggregation bool
}

func (c *eventGeneratorMock) CreateEvents(transaction.LogEvent, []transaction.LogEvent, time.Time, common.MapStr, common.MapStr, interface{}) (events []beat.Event, err error) {
	return nil, nil
}

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

func (c *eventGeneratorMock) SetUseTrafficForAggregation(useTrafficForAggregation bool) {
	c.shouldUseTrafficForAggregation = useTrafficForAggregation
}

func setupConfig() {
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY_DATA", "12345")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY_DATA", "12345")

	agentConfig = &config.AgentConfig{
		CentralConfig: &corecfg.CentralConfiguration{
			AgentType:               corecfg.TraceabilityAgent,
			URL:                     "http://foo.bar",
			PlatformURL:             "http://foo.bar",
			TenantID:                TenantID,
			APICDeployment:          APICDeployment,
			Environment:             Environment,
			EnvironmentID:           EnvID,
			UsageReporting:          corecfg.NewUsageReporting(),
			ReportActivityFrequency: 1,
			ClientTimeout:           1,
			Auth: &corecfg.AuthConfiguration{
				URL:        "https://login.axway.com/auth",
				Realm:      "Broker",
				ClientID:   "DOSA_123456789123456789",
				PrivateKey: "/tmp/private_key.pem",
				PublicKey:  "/tmp/public_key",
				Timeout:    30 * time.Second,
			},
		},
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: 1 * time.Second,
		},
	}
	agentConfig.CentralConfig.SetEnvironmentID(EnvID)
	agentConfig.CentralConfig.SetTeamID(TeamID)
	config.SetConfig(agentConfig)
	agent.Initialize(agentConfig.CentralConfig)
}

type eventGenMockErr struct {
	shouldUseTrafficForAggregation bool
}

func (c *eventGenMockErr) CreateEvents(transaction.LogEvent, []transaction.LogEvent, time.Time, common.MapStr, common.MapStr, interface{}) (events []beat.Event, err error) {
	return nil, nil
}

func (c *eventGenMockErr) CreateEvent(
	_ transaction.LogEvent,
	_ time.Time,
	_ common.MapStr,
	_ common.MapStr,
	_ interface{},
) (event beat.Event, err error) {
	return beat.Event{}, fmt.Errorf("create event error")
}

func (c *eventGenMockErr) SetUseTrafficForAggregation(useTrafficForAggregation bool) {
	c.shouldUseTrafficForAggregation = useTrafficForAggregation
}

type eventMapperErr struct{}

func (em *eventMapperErr) ProcessMapping(_ anypoint.AnalyticsEvent) ([]*transaction.LogEvent, error) {
	return nil, fmt.Errorf("event mapping error")
}
