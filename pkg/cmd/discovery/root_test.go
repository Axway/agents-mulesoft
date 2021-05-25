package discovery

import (
	"fmt"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"

	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/discovery/mocks"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agents-mulesoft/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_initConfig(t *testing.T) {
	conf, err := initConfig(corecfg.NewCentralConfig(corecfg.DiscoveryAgent))
	assert.Nil(t, err)
	assert.NotNil(t, conf)
	cfg, ok := conf.(*config.AgentConfig)
	assert.True(t, ok)
	assert.IsType(t, &config.AgentConfig{}, cfg)
	err = cfg.MulesoftConfig.ValidateCfg()
	assert.NotNil(t, err)

	cfg.MulesoftConfig.Username = "username"
	cfg.MulesoftConfig.Password = "123"
	cfg.MulesoftConfig.Environment = "env1"
	cfg.MulesoftConfig.PollInterval = 5 * time.Second
	cfg.MulesoftConfig.OrgName = "dev"
	err = cfg.MulesoftConfig.ValidateCfg()
	assert.Nil(t, err)
}

func Test_initSubscriptionManager(t *testing.T) {
	// should register with no error
	mc := &anypoint.MockAnypointClient{}
	cc := &mocks.MockCentralClient{}

	agent.InitializeForTest(cc)

	cc.On("RegisterSubscriptionSchema").Return(nil)
	manager, err := initSubscriptionManager(mc, cc)
	assert.NotNil(t, manager)
	assert.Equal(t, 1, len(manager.Schemas()))
	assert.Nil(t, err)

	cc = &mocks.MockCentralClient{}
	// should throw an error when registering
	cc.On("RegisterSubscriptionSchema").Return(fmt.Errorf("failed"))
	_, err = initSubscriptionManager(mc, cc)
	assert.NotNil(t, err)
}
