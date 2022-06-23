package discovery

import (
	"testing"
	"time"

	"github.com/Axway/agents-mulesoft/pkg/common"
	subs "github.com/Axway/agents-mulesoft/pkg/subscription"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/stretchr/testify/assert"
)

func TestCleanTags(t *testing.T) {
	input := "discoverone,discovertwo,discoverthree "
	assert.Equal(t, []string{"discoverone", "discovertwo", "discoverthree"}, cleanTags(input))
}

func TestAgent(t *testing.T) {
	cfg := newConfig()
	mockClient := &anypoint.MockAnypointClient{}
	mss := &mockSchemaHandler{}
	a := NewAgent(cfg, mockClient, mss)
	assert.NotNil(t, a)
	assert.Equal(t, mockClient, a.client)
	assert.NotNil(t, a.discovery)
	assert.NotNil(t, a.publisher)
	assert.NotNil(t, a.stopAgent)
}

func TestAgent_Run(t *testing.T) {
	cfg := newConfig()
	config.SetConfig(cfg)
	discHit := make(chan bool)
	pubHit := make(chan bool)
	mockClient := &anypoint.MockAnypointClient{}
	stopDisc := make(chan bool)
	stopPub := make(chan bool)
	disc := &mockDiscovery{
		stopCh: stopDisc,
		hitCh:  discHit,
	}
	pub := &mockPublisher{
		stopCh: stopPub,
		hitCh:  pubHit,
	}
	a := newAgent(mockClient, disc, pub)
	go a.Run()

	go a.onConfigChange()
	discStopped := <-disc.stopCh
	assert.True(t, discStopped)
	pubStopped := <-pub.stopCh
	assert.True(t, pubStopped)

	discRunning := <-disc.hitCh
	assert.True(t, discRunning)
	pubRunning := <-pub.hitCh
	assert.True(t, pubRunning)
	go a.Stop()
	done := <-disc.stopCh
	assert.True(t, done)
	done = <-pub.stopCh
	assert.True(t, done)
}

func newConfig() *config.AgentConfig {
	return &config.AgentConfig{
		CentralConfig: corecfg.NewCentralConfig(corecfg.DiscoveryAgent),
		MulesoftConfig: &config.MulesoftConfig{
			AnypointExchangeURL: "abc.com",
			CachePath:           "/tmp",
			DiscoveryIgnoreTags: "",
			DiscoveryTags:       "",
			Environment:         "mule",
			Password:            "123",
			PollInterval:        1 * time.Second,
			ProxyURL:            "",
			SessionLifetime:     60 * time.Minute,
			TLS:                 nil,
			Username:            "abc",
		},
	}
}

type mockDiscovery struct {
	stopCh chan bool
	hitCh  chan bool
}

func (m mockDiscovery) Loop() {
	m.hitCh <- true
}

func (m mockDiscovery) OnConfigChange(*config.MulesoftConfig) {
}

func (m mockDiscovery) Stop() {
	m.stopCh <- true
}

type mockPublisher struct {
	stopCh chan bool
	hitCh  chan bool
}

func (m mockPublisher) Loop() {
	m.hitCh <- true
}

func (m mockPublisher) OnConfigChange(*config.MulesoftConfig) {
}

func (m mockPublisher) Stop() {
	m.stopCh <- true
}

type mockSchemaHandler struct{}

func (m *mockSchemaHandler) GetSubscriptionSchemaName(_ common.PolicyDetail) string {
	return ""
}
func (m *mockSchemaHandler) RegisterNewSchema(_ subs.SubSchema) {
}
