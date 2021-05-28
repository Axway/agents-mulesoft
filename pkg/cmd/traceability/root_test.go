package traceability

import (
	"testing"

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
	assert.NotNil(t, config.GetConfig())
}
