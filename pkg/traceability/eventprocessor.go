package traceability

import (
	"encoding/json"
	"time"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"

	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"

	"github.com/Axway/agents-mulesoft/pkg/config"
)

type Processor interface {
	Process(events []publisher.Event) []publisher.Event
	ProcessRaw(rawEventData []byte) []beat.Event
}

// EventProcessor - represents the processor for received event to generate event(s) for AmplifyCentral
// The event processing can be done either when the beat input receives the log entry or before the beat transport
// publishes the event to transport.
// When processing the received log entry on input, the log entry is mapped to structure expected for AmplifyCentral Observer
// and then beat.Event is published to beat output that produces the event over the configured transport.
// When processing the log entry on output, the log entry is published to output as beat.Event. The output transport invokes
// the Process(events []publisher.Event) method which is set as output event processor. The Process() method processes the received
// log entry and performs the mapping to structure expected for AmplifyCentral Observer. The method returns the converted Events to
// transport publisher which then produces the events over the transport.
type EventProcessor struct {
	cfg            *config.AgentConfig
	eventGenerator transaction.EventGenerator
	eventMapper    *EventMapper
}

// NewEventProcessor - return a new EventProcessor
func NewEventProcessor(gateway *config.AgentConfig, eventGenerator transaction.EventGenerator) *EventProcessor {
	ep := &EventProcessor{
		cfg:            gateway,
		eventGenerator: eventGenerator,
		eventMapper:    &EventMapper{},
	}
	return ep
}

// Process - callback set as output event processor that gets invoked by transport publisher to process the received events
func (p *EventProcessor) Process(events []publisher.Event) []publisher.Event {
	newEvents := make([]publisher.Event, 0)
	for _, event := range events {
		// Get the message from the log file
		eventMsgFieldVal, err := event.Content.Fields.GetValue("message")
		if err != nil {
			log.Error(err.Error())
			return newEvents
		}

		eventMsg, ok := eventMsgFieldVal.(string)
		if ok {
			// Unmarshal the message into the struct representing traffic log entry in gateway logs
			beatEvents := p.ProcessRaw([]byte(eventMsg))
			if beatEvents != nil {
				for _, beatEvent := range beatEvents {
					publisherEvent := publisher.Event{
						Content: beatEvent,
					}
					newEvents = append(newEvents, publisherEvent)
				}
			}
		}
	}
	return newEvents
}

// ProcessRaw - process the received log entry and returns the event to be published to Amplifyingestion service
func (p *EventProcessor) ProcessRaw(rawEventData []byte) []beat.Event {
	var gatewayTrafficLogEntry anypoint.AnalyticsEvent
	err := json.Unmarshal(rawEventData, &gatewayTrafficLogEntry)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	// Map the log entry to log event structure expected by AmplifyCentral Observer
	logEvents, err := p.eventMapper.processMapping(gatewayTrafficLogEntry)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	events := make([]beat.Event, 0)
	for _, logEvent := range logEvents {
		// Generates the beat.Event with attributes by Amplify ingestion service
		event, err := p.eventGenerator.CreateEvent(*logEvent, time.Now(), nil, nil, nil)
		if err != nil {
			log.Error(err.Error())
		} else {
			events = append(events, event)
		}
	}
	return events
}
