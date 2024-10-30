package discovery

import (
	"testing"
	"time"

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

	cfg.MulesoftConfig.ClientID = "1"
	cfg.MulesoftConfig.ClientSecret = "2"
	cfg.MulesoftConfig.Environment = "env1"
	cfg.MulesoftConfig.PollInterval = 5 * time.Second
	cfg.MulesoftConfig.OrgName = "dev"
	cfg.MulesoftConfig.CachePath = "/tmp"
	err = cfg.MulesoftConfig.ValidateCfg()
	assert.Nil(t, err)
}
