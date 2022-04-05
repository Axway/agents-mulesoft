package subscription

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agents-mulesoft/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestClientID(t *testing.T) {
	clientIDPolicy := NewClientIDContract()
	assert.Equal(t, name, clientIDPolicy.Name())
	assert.NotEmpty(t, clientIDPolicy.Schema().GetProperty(common.AppName))
	assert.NotEmpty(t, clientIDPolicy.Schema().GetProperty(common.Description))
	pd := common.PolicyDetail{
		Policy:     apic.Apikey,
		IsSLABased: false,
		APIId:      name,
	}
	assert.True(t, clientIDPolicy.IsApplicable(pd))
}
