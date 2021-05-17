package traceability

import (
	"testing"
	"time"

	"github.com/Axway/agents-mulesoft/pkg/discovery"

	"github.com/Axway/agent-sdk/pkg/transaction"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
)

var event = anypoint.AnalyticsEvent{
	Application:        "4321",
	APIID:              "211799904",
	APIName:            "petstore-3",
	APIVersionID:       "16810512",
	APIVersionName:     "v1",
	ApplicationName:    "foo",
	Browser:            "Chrome",
	City:               "Phoenix",
	ClientIP:           "1.2.3.4",
	Continent:          "North America",
	Country:            "United States",
	HardwarePlatform:   "",
	MessageID:          "e2029ea0-a873-11eb-875c-064449f4dd2c",
	OSFamily:           "",
	OSMajorVersion:     "",
	OSMinorVersion:     "",
	OSVersion:          "",
	PostalCode:         "",
	RequestOutcome:     "PROCESSED",
	RequestSize:        0,
	ResourcePath:       "/pets",
	ResponseSize:       20,
	ResponseTime:       60,
	StatusCode:         200,
	Timestamp:          time.Now(),
	Timezone:           "",
	UserAgentName:      "Mozilla",
	UserAgentVersion:   "5.0",
	Verb:               "GET",
	ViolatedPolicyName: "",
}

var app = &anypoint.Application{
	APIEndpoints: false,
	ClientID:     "21",
	ClientSecret: "23",
	Description:  "app",
	ID:           1,
	Name:         "foo",
}

func init() {
	setupRedaction()
	setupConfig()
}

func TestEventMapper_processMapping(t *testing.T) {
	client := &mockAnalyticsClient{
		app: app,
	}
	mapper := &EventMapper{client: client}

	item, err := mapper.ProcessMapping(event)
	assert.Nil(t, err)
	assert.Equal(t, transaction.FormatApplicationID(event.Application), item[0].TransactionSummary.Application.ID)
	assert.Equal(t, event.ApplicationName, item[0].TransactionSummary.Application.Name)
	assert.Equal(t, 3, len(item))
	assert.NotNil(t, item[1].TransactionEvent.Protocol)
	for i := 0; i < 2; i++ {
		rqstHeader := item[i+1].TransactionEvent.Protocol.(*transaction.Protocol).RequestHeaders
		respHeader := item[i+1].TransactionEvent.Protocol.(*transaction.Protocol).ResponseHeaders
		assert.Contains(t, rqstHeader, "User-AgentName")
		assert.Contains(t, rqstHeader, "Request-ID")
		assert.Contains(t, rqstHeader, "Forwarded-For")
		assert.Contains(t, rqstHeader, "Violated-Policies")
		assert.Contains(t, respHeader, "Request-Outcome")
		assert.Contains(t, respHeader, "Response-Time")
	}

	// expect the application name and id to be empty when the event has no app.
	ev := event
	ev.Application = ""
	ev.ApplicationName = ""
	item, err = mapper.ProcessMapping(ev)
	assert.Nil(t, err)
	assert.Nil(t, item[0].TransactionSummary.Application)
}

func Test_getTransactionEventStatus(t *testing.T) {

	status := getTransactionEventStatus(100)
	assert.Equal(t, transaction.TxEventStatusPass, status)

	status = getTransactionEventStatus(200)
	assert.Equal(t, transaction.TxEventStatusPass, status)

	status = getTransactionEventStatus(300)
	assert.Equal(t, transaction.TxEventStatusPass, status)

	status = getTransactionEventStatus(400)
	assert.Equal(t, transaction.TxEventStatusFail, status)

	status = getTransactionEventStatus(500)
	assert.Equal(t, transaction.TxEventStatusFail, status)

	status = getTransactionEventStatus(600)
	assert.Equal(t, transaction.TxEventStatusFail, status)
}

func Test_getTransactionSummaryStatus(t *testing.T) {
	status := getTransactionSummaryStatus(200)
	assert.Equal(t, transaction.TxSummaryStatusSuccess, status)

	status = getTransactionSummaryStatus(300)
	assert.Equal(t, transaction.TxSummaryStatusSuccess, status)

	status = getTransactionSummaryStatus(400)
	assert.Equal(t, transaction.TxSummaryStatusFailure, status)

	status = getTransactionSummaryStatus(500)
	assert.Equal(t, transaction.TxSummaryStatusException, status)

	status = getTransactionSummaryStatus(600)
	assert.Equal(t, transaction.TxSummaryStatusUnknown, status)

	status = getTransactionSummaryStatus(100)
	assert.Equal(t, transaction.TxSummaryStatusUnknown, status)
}

func Test_buildHeaders(t *testing.T) {
	h := map[string]string{
		"Authorization": "abc123",
		"User-Agent":    "MulesoftTraceability",
	}
	res := buildHeaders(h)
	assert.Equal(t, "{\"Authorization\":\"abc123\",\"User-Agent\":\"MulesoftTraceability\"}", res)
}

func Test_APIServiceNameAndTransactionProxyNameAreEqual(t *testing.T) {
	sd := &discovery.ServiceDetail{
		APIName:           "petstore-3",
		APISpec:           []byte(`{"openapi":"3.0.1","servers":[{"url":"google.com"}],"paths":{},"info":{"title":"petstore3"}}`),
		APIUpdateSeverity: "",
		AuthPolicy:        "pass-through",
		Description:       "petstore api",
		Documentation:     nil,
		ID:                "16810512",
		Image:             "",
		ImageContentType:  "",
		ResourceType:      "oas3",
		ServiceAttributes: nil,
		Stage:             "Sandbox",
		State:             "",
		Status:            "",
		SubscriptionName:  "",
		Tags:              nil,
		Title:             "petstore-3",
		URL:               "",
		Version:           "1.0.0",
	}
	body, err := discovery.BuildServiceBody(sd)
	assert.Nil(t, err)
	apiServiceName := body.NameToPush

	client := &mockAnalyticsClient{
		app: app,
		err: nil,
	}
	em := &EventMapper{client: client}
	le, err := em.createSummaryEvent(100, FormatTxnId(event.APIVersionID, event.MessageID), event, "123")
	assert.Nil(t, err)
	transactionProxyName := le.TransactionSummary.Proxy.Name
	transactionProxyID := le.TransactionSummary.Proxy.ID
	assert.Contains(t, transactionProxyName, apiServiceName)
	assert.Equal(t, transaction.FormatProxyID(body.RestAPIID), transactionProxyID)
	assert.Equal(t, event.ApplicationName, le.TransactionSummary.Application.Name)
	assert.Equal(t, transaction.FormatApplicationID(event.Application), le.TransactionSummary.Application.ID)
}
