package discovery

import (
	"errors"
	"testing"

	subscriptionMocks "github.com/Axway/agents-mulesoft/pkg/subscription/mocks"

	"github.com/Axway/agents-mulesoft/pkg/subscription"

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
	assert.Equal(t, cfg.CentralConfig.GetDataPlaneName(), mulesoft)
}

func Test_initSubscriptionManager(t *testing.T) {
	mc := &anypoint.MockAnypointClient{}
	cc := &mocks.MockCentralClient{}

	sc := func(client anypoint.Client) subscription.Contract {
		mh := &subscriptionMocks.MockContract{}
		mh.On("Name").Return("sofake")
		mh.On("Schema").Return("sofake schema")
		return mh
	}

	subscription.Register(sc)

	cc.On("RegisterSubscriptionSchema").Return(errors.New("Cannot register subscription schema"))

	_, err := initSubscriptionManager(mc, cc)
	assert.NotNil(t, err)
}
