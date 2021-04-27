package traceability

import (
	"testing"
	"time"

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
	t.Skip()
	assetCache := agent.GetAPICache()
	assetCache.Set("111", "{}")
	ac := &config.AgentConfig{
		CentralConfig: corecfg.NewCentralConfig(corecfg.TraceabilityAgent),
		MulesoftConfig: &config.MulesoftConfig{
			PollInterval: 1 * time.Second,
		},
	}
	processor := NewEventProcessor(ac, &eventGenerator{})
	events := []publisher.Event{
		{
			Content: beat.Event{
				Timestamp: time.Time{},
				Fields: map[string]interface{}{
					"message": `{"Client IP": "","API ID": "", "API Version ID": "111", "Message ID": "222"}`,
				},
				Private:    nil,
				TimeSeries: false,
			},
		},
	}
	processedEvents := processor.Process(events)
	logrus.Infof("Events: %+v", processedEvents)
}

type eventGenerator struct {
}

func (eg *eventGenerator) CreateEvent(
	logEvent transaction.LogEvent,
	eventTime time.Time,
	metaData common.MapStr,
	fields common.MapStr,
	privateData interface{},
) (event beat.Event, err error) {
	return beat.Event{}, nil
}
