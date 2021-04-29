package traceability

import (
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/transaction"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
)

var event = anypoint.AnalyticsEvent{
	APIID:              "211799904",
	APIName:            "petstore-3",
	APIVersionID:       "16810512",
	APIVersionName:     "v1",
	ApplicationName:    "",
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

func init() {
	setupRedaction()
	setupConfig()
}

func TestEventMapper_processMapping(t *testing.T) {
	mapper := &EventMapper{}

	item, err := mapper.ProcessMapping(event)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(item))
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
