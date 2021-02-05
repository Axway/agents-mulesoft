package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Axway/agents-mulesoft/mulesoft_traceability_agent/pkg/anypoint"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// EventMapper -
type EventMapper struct {
}

func (m *EventMapper) processMapping(anypointAnalyticsEvent anypoint.AnalyticsEvent) ([]*transaction.LogEvent, error) {
	centralCfg := agent.GetCentralConfig()

	eventTime := time.Now().Unix()
	txID := fmt.Sprintf("%s-%s", anypointAnalyticsEvent.APIVersionID, anypointAnalyticsEvent.MessageID)
	txEventID := anypointAnalyticsEvent.MessageID
	transInboundLogEventLeg, err := m.createTransactionEvent(eventTime, txID, anypointAnalyticsEvent, txEventID+"-leg0", "", "Inbound")
	if err != nil {
		return nil, err
	}

	transOutboundLogEventLeg, err := m.createTransactionEvent(eventTime, txID, anypointAnalyticsEvent, txEventID+"-leg1", txEventID+"-leg0", "Outbound")
	if err != nil {
		return nil, err
	}

	transSummaryLogEvent, err := m.createSummaryEvent(eventTime, txID, anypointAnalyticsEvent, centralCfg.GetTeamID())
	if err != nil {
		return nil, err
	}

	return []*transaction.LogEvent{
		transSummaryLogEvent,
		transInboundLogEventLeg,
		transOutboundLogEventLeg,
	}, nil
}

func (m *EventMapper) getTransactionEventStatus(code int) transaction.TxEventStatus {
	if code >= 400 {
		return transaction.TxEventStatusFail
	}
	return transaction.TxEventStatusFail
}

func (m *EventMapper) getTransactionSummaryStatus(statusCode int) transaction.TxSummaryStatus {
	transSummaryStatus := transaction.TxSummaryStatusUnknown
	if statusCode >= http.StatusOK && statusCode < http.StatusBadRequest {
		transSummaryStatus = transaction.TxSummaryStatusSuccess
	} else if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		transSummaryStatus = transaction.TxSummaryStatusFailure
	} else if statusCode >= http.StatusInternalServerError && statusCode < http.StatusNetworkAuthenticationRequired {
		transSummaryStatus = transaction.TxSummaryStatusException
	}
	return transSummaryStatus
}

func (m *EventMapper) buildHeaders(headers map[string]string) string {
	jsonHeader, err := json.Marshal(headers)
	if err != nil {
		log.Error(err.Error())
	}
	return string(jsonHeader)
}

func (m *EventMapper) createTransactionEvent(eventTime int64, txID string, txDetails anypoint.AnalyticsEvent, eventID, parentEventID, direction string) (*transaction.LogEvent, error) {
	//TODO - Slim pickings on header data
	req:=map[string]string{"User-AgentName": txDetails.UserAgentName,}
	res:=map[string]string{"Request-Outcome": txDetails.RequestOutcome}
	httpProtocolDetails, err := transaction.NewHTTPProtocolBuilder().
		SetURI(fmt.Sprintf("https://mulepoop%s", txDetails.ResourcePath)).
		SetMethod(txDetails.Verb).
		SetStatus(txDetails.StatusCode, http.StatusText(txDetails.StatusCode)).
		SetHost(txDetails.ClientIP).
		SetHeaders(	m.buildHeaders(req),	m.buildHeaders(res)).
		SetByteLength(txDetails.RequestSize, txDetails.ResponseSize).
		//SetRemoteAddress("", txDetails.DesHost, txDetails.DestPort).
		//SetLocalAddress(txDetails.SourceHost, txDetails.SourcePort).
		Build()
	if err != nil {
		return nil, err
	}

	return transaction.NewTransactionEventBuilder().
		SetTimestamp(eventTime).
		SetTransactionID(txID).
		SetID(eventID).
		SetParentID(parentEventID).
		SetSource(txDetails.ClientIP + ":0").
		SetDestination("mulepoop:443").
		SetDirection(direction).
		SetStatus(m.getTransactionEventStatus(txDetails.StatusCode)).
		SetProtocolDetail(httpProtocolDetails).
		Build()
}

func (m *EventMapper) createSummaryEvent(eventTime int64, txID string, anypointAnalyticsEvent anypoint.AnalyticsEvent, teamID string) (*transaction.LogEvent, error) {
	statusCode := anypointAnalyticsEvent.StatusCode
	method := anypointAnalyticsEvent.Verb
	uri := fmt.Sprintf("https://mulepoop%s", anypointAnalyticsEvent.ResourcePath)
	host := anypointAnalyticsEvent.ClientIP

	return transaction.NewTransactionSummaryBuilder().
		SetTimestamp(eventTime).
		SetTransactionID(txID).
		SetStatus(m.getTransactionSummaryStatus(statusCode), strconv.Itoa(statusCode)).
		SetTeam(teamID).
		SetEntryPoint("http", method, uri, host).
		// If the API is published to Central as unified catalog item/API service, se the Proxy details with the API definition
		// The Proxy.Name represents the name of the API
		// The Proxy.ID should be of format "remoteApiId_<ID Of the API on remote gateway>". Use transaction.FormatProxyID(<ID Of the API on remote gateway>) to get the formatted value.
		SetProxy(transaction.FormatProxyID(anypointAnalyticsEvent.APIVersionID), anypointAnalyticsEvent.APIName, 1).
		Build()
}
