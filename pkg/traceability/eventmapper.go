package traceability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/transaction"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const Inbound = "Inbound"
const Outbound = "Outbound"
const Client = "Client"
const MuleProxy = "Mule.APIProxy"
const Backend = "Backend"

type Mapper interface {
	ProcessMapping(event anypoint.AnalyticsEvent) ([]*transaction.LogEvent, error)
}

// EventMapper -
type EventMapper struct{}

func (em *EventMapper) ProcessMapping(event anypoint.AnalyticsEvent) ([]*transaction.LogEvent, error) {
	centralCfg := agent.GetCentralConfig()

	eventTime := event.Timestamp.UnixNano() / 1000000
	txID := FormatTxnId(event.APIVersionID, event.MessageID)
	txEventID := event.MessageID
	leg0ID := FormatLeg0(txEventID)
	leg1ID := FormatLeg1(txEventID)

	transSummaryLogEvent, err := em.createSummaryEvent(eventTime, txID, event, centralCfg.GetTeamID())
	if err != nil {
		return nil, err
	}

	transOutboundLogEventLeg, err := em.createTransactionEvent(eventTime, txID, event, leg0ID, "", Outbound)
	if err != nil {
		return nil, err
	}

	transInboundLogEventLeg, err := em.createTransactionEvent(eventTime, txID, event, leg1ID, leg0ID, Inbound)
	if err != nil {
		return nil, err
	}

	return []*transaction.LogEvent{
		transSummaryLogEvent,
		transOutboundLogEventLeg,
		transInboundLogEventLeg,
	}, nil
}

func (em *EventMapper) createTransactionEvent(
	eventTime int64,
	txID string,
	txDetails anypoint.AnalyticsEvent,
	eventID,
	parentEventID,
	direction string,
) (*transaction.LogEvent, error) {

	req := map[string]string{
		"User-AgentName":    txDetails.UserAgentName + txDetails.UserAgentVersion,
		"Request-ID":        txDetails.MessageID,
		"Forwarded-For":     txDetails.ClientIP,
		"Violated-Policies": txDetails.ViolatedPolicyName,
	}
	res := map[string]string{
		"Request-Outcome": txDetails.RequestOutcome,
		"Response-Time":   strconv.Itoa(txDetails.ResponseTime),
	}

	httpProtocolDetails, err := transaction.NewHTTPProtocolBuilder().
		SetByteLength(txDetails.RequestSize, txDetails.ResponseSize).
		SetHeaders(buildHeaders(req), buildHeaders(res)).
		SetHost(txDetails.ClientIP).
		SetMethod(txDetails.Verb).
		SetStatus(txDetails.StatusCode, http.StatusText(txDetails.StatusCode)).
		SetURI(txDetails.ResourcePath).
		Build()
	if err != nil {
		return nil, err
	}

	builder := transaction.NewTransactionEventBuilder().
		SetDirection(direction).
		SetID(eventID).
		SetParentID(parentEventID).
		SetProtocolDetail(httpProtocolDetails).
		SetStatus(getTransactionEventStatus(txDetails.StatusCode)).
		SetTimestamp(eventTime).
		SetTransactionID(txID)

	if direction == Outbound {
		builder.SetSource(Client).
			SetDestination(MuleProxy)
	} else {
		builder.SetSource(MuleProxy).
			SetDestination(Backend + txDetails.APIName)
	}

	return builder.Build()

}

func (em *EventMapper) createSummaryEvent(
	eventTime int64,
	txID string,
	event anypoint.AnalyticsEvent,
	teamID string,
) (*transaction.LogEvent, error) {
	host := event.ClientIP
	method := event.Verb
	name := event.APIName + "-" + event.APIVersionName
	statusCode := event.StatusCode
	uri := event.ResourcePath

	// must be the same as the the 'externalAPIID' attribute set on the APIService.
	apiVersionID := event.APIVersionID

	builder := transaction.NewTransactionSummaryBuilder().
		SetDuration(event.ResponseTime).
		SetEntryPoint("http", method, uri, host).
		SetProxy(transaction.FormatProxyID(apiVersionID), name, 1).
		SetStatus(getTransactionSummaryStatus(statusCode), strconv.Itoa(statusCode)).
		SetTeam(teamID).
		SetTransactionID(txID).
		SetTimestamp(eventTime)

	//Associating the Client ID as the Application ID here since that is what we get from the analytics API as event.Application, the client ID can be changed from Mulesoft side but not from Amplify
	//TODO an enhancement can be made to call another API and get the application ID associated with the Client ID application
	// https://anypoint.mulesoft.com/exchange/api/v2/organizations/{Org_ID}/applications/{Client_ID}
	if event.ApplicationName != "" && event.Application != "" {
		builder.SetApplication(transaction.FormatApplicationID(event.Application), event.ApplicationName)
	}

	return builder.Build()
}

func getTransactionSummaryStatus(statusCode int) transaction.TxSummaryStatus {
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

func buildHeaders(headers map[string]string) string {
	jsonHeader, err := json.Marshal(headers)
	if err != nil {
		log.Error(err.Error())
		return ""
	}
	return string(jsonHeader)
}

func getTransactionEventStatus(code int) transaction.TxEventStatus {
	if code >= 400 {
		return transaction.TxEventStatusFail
	}
	return transaction.TxEventStatusPass
}

func FormatTxnId(apiVersionID, messageID string) string {
	return fmt.Sprintf("%s-%s", apiVersionID, messageID)
}

func FormatLeg0(id string) string {
	return fmt.Sprintf("%s-leg0", id)
}
func FormatLeg1(id string) string {
	return fmt.Sprintf("%s-leg1", id)
}
