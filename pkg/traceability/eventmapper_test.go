package traceability

import (
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/transaction"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"

	"github.com/Axway/agent-sdk/pkg/agent"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/config"
)

var event = anypoint.AnalyticsEvent{
	APIID:              "211799904",
	APIName:            "petstore-3",
	APIVersionID:       "16810512",
	APIVersionName:     "v1",
	ApplicationName:    "",
	Browser:            "Other",
	City:               "Phoenix",
	ClientIP:           "1.2.3.4",
	Continent:          "North America",
	Country:            "USA",
	HardwarePlatform:   "Other",
	MessageID:          "e2029ea0-a873-11eb-875c-064449f4dd2c",
	OSFamily:           "Other",
	OSMajorVersion:     "1",
	OSMinorVersion:     "2",
	OSVersion:          "1.2",
	PostalCode:         "12345",
	RequestOutcome:     "PROCESSED",
	RequestSize:        0,
	ResourcePath:       "/pets",
	ResponseSize:       18,
	ResponseTime:       58,
	StatusCode:         200,
	Timestamp:          time.Now(),
	Timezone:           "Mountain",
	UserAgentName:      "Mozilla",
	UserAgentVersion:   "5.0",
	Verb:               "GET",
	ViolatedPolicyName: "",
}

func init() {
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
}

func TestEventMapper_processMapping(t *testing.T) {
	mapper := &EventMapper{}

	item, err := mapper.ProcessMapping(event)
	assert.Nil(t, err)
	logrus.Infof("Item: %+v", item)
}

func TestEventMapper_getTransactionEventStatus(t *testing.T) {
	mapper := &EventMapper{}

	status := mapper.getTransactionEventStatus(100)
	assert.Equal(t, transaction.TxEventStatusPass, status)

	status = mapper.getTransactionEventStatus(200)
	assert.Equal(t, transaction.TxEventStatusPass, status)

	status = mapper.getTransactionEventStatus(300)
	assert.Equal(t, transaction.TxEventStatusPass, status)

	status = mapper.getTransactionEventStatus(400)
	assert.Equal(t, transaction.TxEventStatusFail, status)

	status = mapper.getTransactionEventStatus(500)
	assert.Equal(t, transaction.TxEventStatusFail, status)

	status = mapper.getTransactionEventStatus(600)
	assert.Equal(t, transaction.TxEventStatusFail, status)
}

func TestEventMapper_getTransactionSummaryStatus(t *testing.T) {
	mapper := &EventMapper{}

	status := mapper.getTransactionSummaryStatus(200)
	assert.Equal(t, transaction.TxSummaryStatusSuccess, status)

	status = mapper.getTransactionSummaryStatus(300)
	assert.Equal(t, transaction.TxSummaryStatusSuccess, status)

	status = mapper.getTransactionSummaryStatus(400)
	assert.Equal(t, transaction.TxSummaryStatusFailure, status)

	status = mapper.getTransactionSummaryStatus(500)
	assert.Equal(t, transaction.TxSummaryStatusException, status)

	status = mapper.getTransactionSummaryStatus(600)
	assert.Equal(t, transaction.TxSummaryStatusUnknown, status)

	status = mapper.getTransactionSummaryStatus(100)
	assert.Equal(t, transaction.TxSummaryStatusUnknown, status)
}

func TestEventMapper_buildHeaders(t *testing.T) {
	mapper := &EventMapper{}

	h := map[string]string{
		"Authorization": "abc123",
		"User-Agent":    "MulesoftTraceability",
	}
	res := mapper.buildHeaders(h)
	assert.Equal(t, "{\"Authorization\":\"abc123\",\"User-Agent\":\"MulesoftTraceability\"}", res)
}

func TestEventMapper_createTransactionEvent(t *testing.T) {
	// mapper := &EventMapper{}
	// i := 1000
	// mapper.createTransactionEvent(int64(i), "1")
}

func TestEventMapper_createSummaryEvent(t *testing.T) {
	// mapper := &EventMapper{}
}
